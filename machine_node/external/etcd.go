package external

import (
	"context"
	"path/filepath"

	"github.com/DataDog/zstd"
	"github.com/pkg/errors"

	"github.com/GDVFox/gostreaming/util/storage"
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

func buildActionKey(actionName string) string {
	return filepath.Join(actionsPath, actionName)
}
