package watcher

import (
	"context"

	"github.com/GDVFox/gostreaming/util"
)

// Watcher объект синглтон для слежения за работоспособностью планов.
var Watcher *PlanWatcher

// StartWatcher инициализирует синглтон RuntimeWatcher и запускает его.
func StartWatcher(ctx context.Context, l *util.Logger, cfg *PlanWatcherConfig) error {
	var err error
	Watcher, err = newPlanWatcher(l, cfg)
	if err != nil {
		return err
	}
	go Watcher.run(ctx)
	return nil
}
