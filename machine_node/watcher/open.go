package watcher

import (
	"context"

	"github.com/GDVFox/gostreaming/util"
)

// RuntimeWatcher объект синглтон для слежения за работоспособностью действий.
var RuntimeWatcher *Watcher

// StartWatcher инициализирует синглтон RuntimeWatcher и запускает его.
func StartWatcher(ctx context.Context, l *util.Logger, cfg *Config) error {
	RuntimeWatcher = newWatcher(l, cfg)
	return RuntimeWatcher.start(ctx)
}
