package storage

import (
	"context"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/GDVFox/gostreaming/util"
)

var (
	// ErrNotFound значение не найдено в etcd.
	ErrNotFound = errors.New("key not found")
	// ErrAlreadyExists значение уже записано etcd.
	ErrAlreadyExists = errors.New("key already exists")
)

// ETCDConfig конфигурация подключения к etcd.
type ETCDConfig struct {
	Endpoints []string          `yaml:"endpoints"`
	Timeout   util.Duration     `yaml:"timeout"`
	Retry     *util.RetryConfig `yaml:"retry"`
}

// NewETCDConfig создает новый конфиг etcd с параметрами по-умолчанию.
func NewETCDConfig() *ETCDConfig {
	return &ETCDConfig{
		Endpoints: []string{},
		Timeout:   util.Duration(1 * time.Minute),
		Retry:     util.NewRetryConfig(),
	}
}

// ETCDClient клиент для работы с etcd.
type ETCDClient struct {
	cli *clientv3.Client
	kv  clientv3.KV

	cfg *ETCDConfig
}

// NewETCDClient создает новый etcd клиент.
func NewETCDClient(cfg *ETCDConfig) (*ETCDClient, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:          cfg.Endpoints,
		DialTimeout:        time.Duration(cfg.Timeout),
		MaxCallSendMsgSize: 256 * 1024 * 1024, // 256MB
	})
	if err != nil {
		return nil, err
	}

	kv := clientv3.NewKV(cli)
	return &ETCDClient{
		cli: cli,
		kv:  kv,
		cfg: cfg,
	}, nil
}

// List получает список ключей с перфиксом prefix.
func (c *ETCDClient) List(ctx context.Context, prefix string) ([]string, error) {
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

// Get получает данные по ключу key.
func (c *ETCDClient) Get(ctx context.Context, key string) ([]byte, error) {
	var resp *clientv3.GetResponse
	err := util.Retry(ctx, c.cfg.Retry, func() error {
		var err error
		requestCtx, requestCancel := context.WithTimeout(ctx, time.Duration(c.cfg.Timeout))
		resp, err = c.kv.Get(requestCtx, key)
		requestCancel() // запрос выполнен, нужно очистить таймер.
		return err
	})
	if err != nil {
		return nil, err
	}

	if len(resp.Kvs) == 0 {
		return nil, ErrNotFound
	}

	return resp.Kvs[0].Value, nil
}

// Put записывает данные value по ключу key.
func (c *ETCDClient) Put(ctx context.Context, key string, value string) error {
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

// Delete удалает данные по ключу key.
func (c *ETCDClient) Delete(ctx context.Context, key string) error {
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
