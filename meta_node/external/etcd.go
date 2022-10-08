package external

import (
	"context"
	"encoding/json"
	"path/filepath"

	"github.com/DataDog/zstd"
	"github.com/pkg/errors"

	"github.com/GDVFox/gostreaming/meta_node/planner"
	"github.com/GDVFox/gostreaming/util/storage"
)

const (
	plansPath   = "/plans"
	actionsPath = "/actions"

	actionsCompressLevel = 11
)

// ETCDClient клиент для работы с etcd.
type ETCDClient struct {
	cli *storage.ETCDClient
}

// NewETCDClient создает новый etcd клиент.
func NewETCDClient(cfg *storage.ETCDConfig) (*ETCDClient, error) {
	cli, err := storage.NewETCDClient(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "can not create etcd client")
	}
	return &ETCDClient{cli: cli}, nil
}

// LoadPlanNames получает список названий планов, доступных в etcd.
func (c *ETCDClient) LoadPlanNames(ctx context.Context) ([]string, error) {
	planNames, err := c.cli.List(ctx, plansPath)
	if err != nil {
		return nil, errors.Wrap(err, "can not list of plan from etcd")
	}
	return planNames, nil
}

// LoadPlan получает план выполнения из etcd.
func (c *ETCDClient) LoadPlan(ctx context.Context, name string) (*planner.Plan, error) {
	resp, err := c.cli.Get(ctx, buildPlansKey(name))
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

	if err := c.cli.Put(ctx, buildPlansKey(plan.Name), string(planData)); err != nil {
		return errors.Wrap(err, "can not register plan in etcd")
	}
	return nil
}

// DeletePlan удаления плана из etcd.
func (c *ETCDClient) DeletePlan(ctx context.Context, name string) error {
	if err := c.cli.Delete(ctx, buildPlansKey(name)); err != nil {
		return errors.Wrap(err, "can not delete plan from etcd")
	}
	return nil
}

// LoadActionNames получает список названий планов, доступных в etcd.
func (c *ETCDClient) LoadActionNames(ctx context.Context) ([]string, error) {
	actionNames, err := c.cli.List(ctx, actionsPath)
	if err != nil {
		return nil, errors.Wrap(err, "can not list of actions from etcd")
	}
	return actionNames, nil
}

// LoadAction получает действие из etcd.
func (c *ETCDClient) LoadAction(ctx context.Context, name string) ([]byte, error) {
	resp, err := c.cli.Get(ctx, buildActionKey(name))
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

	if err := c.cli.Put(ctx, buildActionKey(name), string(compressedAction)); err != nil {
		return errors.Wrap(err, "can not register action in etcd")
	}
	return nil
}

// DeleteAction удаления действия из etcd.
func (c *ETCDClient) DeleteAction(ctx context.Context, name string) error {
	if err := c.cli.Delete(ctx, buildActionKey(name)); err != nil {
		return errors.Wrap(err, "can not delete action from etcd")
	}
	return nil
}

func buildActionKey(actionName string) string {
	return filepath.Join(actionsPath, actionName)
}

func buildPlansKey(planName string) string {
	return filepath.Join(plansPath, planName)
}
