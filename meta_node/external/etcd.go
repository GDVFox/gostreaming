package external

import (
	"context"
	"encoding/json"
	"path/filepath"
	"time"

	"github.com/DataDog/zstd"
	"github.com/pkg/errors"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/GDVFox/gostreaming/meta_node/config"
	"github.com/GDVFox/gostreaming/meta_node/planner"
	"github.com/GDVFox/gostreaming/util"
)

var (
	// ErrNotFound значение не найдено в etcd.
	ErrNotFound = errors.New("key not found")
	// ErrAlreadyExists значение уже записано etcd.
	ErrAlreadyExists = errors.New("key already exists")
)

const (
	plansPath   = "/plans"
	actionsPath = "/actions"

	actionsCompressLevel = 11
)

// ETCDClient клиент для работы с etcd.
type ETCDClient struct {
	cli *clientv3.Client
	kv  clientv3.KV

	cfg *config.ETCDConfig
}

// NewETCDClient создает новый etcd клиент.
func NewETCDClient(cfg *config.ETCDConfig) (*ETCDClient, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:          cfg.Endpoints,
		DialTimeout:        time.Duration(cfg.Timeout),
		MaxCallSendMsgSize: 256 * 1024 * 1024, // 256MB
	})
	if err != nil {
		return nil, errors.Wrap(err, "can not create etcd client")
	}

	kv := clientv3.NewKV(cli)
	return &ETCDClient{
		cli: cli,
		kv:  kv,
		cfg: cfg,
	}, nil
}

// LoadPlanNames получает список названий планов, доступных в etcd.
func (c *ETCDClient) LoadPlanNames(ctx context.Context) ([]string, error) {
	planNames, err := c.list(ctx, plansPath)
	if err != nil {
		return nil, errors.Wrap(err, "can not list of plan from etcd")
	}
	return planNames, nil
}

// LoadPlan получает план выполнения из etcd.
func (c *ETCDClient) LoadPlan(ctx context.Context, name string) (*planner.Plan, error) {
	resp, err := c.get(ctx, buildPlansKey(name))
	if err != nil {
		return nil, errors.Wrap(err, "can not load plan from etcd")
	}

	plan := &planner.Plan{}
	if err := json.Unmarshal(resp, &plan); err != nil {
		return nil, errors.Wrap(err, "can not unmarshal plan")
	}
	return plan, nil
}

// RegisterPlan загружает план выполнения в etcd.
func (c *ETCDClient) RegisterPlan(ctx context.Context, plan *planner.Plan) error {
	planData, err := json.Marshal(&plan)
	if err != nil {
		return errors.Wrap(err, "can not marshal plan")
	}

	if err := c.put(ctx, buildPlansKey(plan.Name), string(planData)); err != nil {
		return errors.Wrap(err, "can not register plan in etcd")
	}
	return nil
}

// DeletePlan удаления плана из etcd.
func (c *ETCDClient) DeletePlan(ctx context.Context, name string) error {
	if err := c.delete(ctx, buildPlansKey(name)); err != nil {
		return errors.Wrap(err, "can not delete plan from etcd")
	}
	return nil
}

// LoadActionNames получает список названий планов, доступных в etcd.
func (c *ETCDClient) LoadActionNames(ctx context.Context) ([]string, error) {
	actionNames, err := c.list(ctx, actionsPath)
	if err != nil {
		return nil, errors.Wrap(err, "can not list of actions from etcd")
	}
	return actionNames, nil
}

// LoadAction получает действие из etcd.
func (c *ETCDClient) LoadAction(ctx context.Context, name string) ([]byte, error) {
	resp, err := c.get(ctx, buildActionKey(name))
	if err != nil {
		return nil, errors.Wrap(err, "can not load action from etcd")
	}

	action, err := zstd.Decompress(nil, resp)
	if err != nil {
		return nil, errors.Wrap(err, "can not decompress action")
	}

	return action, nil
}

// RegisterAction загружает действие в etcd.
func (c *ETCDClient) RegisterAction(ctx context.Context, name string, action []byte) error {
	compressedAction, err := zstd.CompressLevel(nil, action, zstd.BestSpeed)
	if err != nil {
		return errors.Wrap(err, "can not compess action in zstd")
	}

	if err := c.put(ctx, buildActionKey(name), string(compressedAction)); err != nil {
		return errors.Wrap(err, "can not register action in etcd")
	}
	return nil
}

// DeleteAction удаления действия из etcd.
func (c *ETCDClient) DeleteAction(ctx context.Context, name string) error {
	if err := c.delete(ctx, buildActionKey(name)); err != nil {
		return errors.Wrap(err, "can not delete action from etcd")
	}
	return nil
}

func (c *ETCDClient) list(ctx context.Context, prefix string) ([]string, error) {
	var resp *clientv3.GetResponse
	err := util.Retry(ctx, c.cfg.Retry, func() error {
		var err error
		requestCtx, requestCancel := context.WithTimeout(ctx, time.Duration(c.cfg.Timeout))
		resp, err = c.kv.Get(requestCtx, prefix, clientv3.WithPrefix(), clientv3.WithKeysOnly())
		requestCancel() // запрос выполнен, нужно очистить таймер.
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		_, name := filepath.Split(string(kv.Key))
		names = append(names, name)
	}
	return names, nil
}

func (c *ETCDClient) get(ctx context.Context, key string) ([]byte, error) {
	var resp *clientv3.GetResponse
	err := util.Retry(ctx, c.cfg.Retry, func() error {
		var err error
		requestCtx, requestCancel := context.WithTimeout(ctx, time.Duration(c.cfg.Timeout))
		resp, err = c.kv.Get(requestCtx, key)
		requestCancel() // запрос выполнен, нужно очистить таймер.
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(resp.Kvs) == 0 {
		return nil, ErrNotFound
	}

	return resp.Kvs[0].Value, nil
}

func (c *ETCDClient) put(ctx context.Context, key string, value string) error {
	var resp *clientv3.TxnResponse
	err := util.Retry(ctx, c.cfg.Retry, func() error {
		var err error
		requestCtx, requestCancel := context.WithTimeout(ctx, time.Duration(c.cfg.Timeout))
		resp, err = c.kv.Txn(requestCtx).If(
			clientv3.Compare(clientv3.CreateRevision(key), "=", 0),
		).Then(
			clientv3.OpPut(key, value),
		).Commit()
		requestCancel() // запрос выполнен, нужно очистить таймер.
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	if !resp.Succeeded {
		return ErrAlreadyExists
	}
	return nil
}

func (c *ETCDClient) delete(ctx context.Context, key string) error {
	var resp *clientv3.TxnResponse
	err := util.Retry(ctx, c.cfg.Retry, func() error {
		var err error
		requestCtx, requestCancel := context.WithTimeout(ctx, time.Duration(c.cfg.Timeout))
		resp, err = c.kv.Txn(requestCtx).If(
			clientv3.Compare(clientv3.CreateRevision(key), "!=", 0),
		).Then(
			clientv3.OpDelete(key),
		).Commit()
		requestCancel() // запрос выполнен, нужно очистить таймер.
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	if !resp.Succeeded {
		return ErrNotFound
	}
	return nil
}

func buildActionKey(actionName string) string {
	return filepath.Join(actionsPath, actionName)
}

func buildPlansKey(planName string) string {
	return filepath.Join(plansPath, planName)
}
