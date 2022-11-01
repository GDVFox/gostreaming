package watcher

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/GDVFox/gostreaming/util"
)

// Возможные ошибки.
var (
	ErrActionAlreadyRegistered = errors.New("action already registered")
	ErrUnknownAction           = errors.New("unknown action")
)

// Config набор настроек для Watcher
type Config struct {
	PingFrequency util.Duration `yaml:"ping-freq"`
}

// NewConfig создает новый Config с настройками по-умолчанию.
func NewConfig() *Config {
	return &Config{
		PingFrequency: util.Duration(5 * time.Second),
	}
}

// Watcher структура для контроля запущенных действий.
type Watcher struct {
	actionsMutex sync.RWMutex
	actions      map[string]*Action

	cfg    *Config
	logger *util.Logger
}

// NewWatcher создает новый объект watcher
func newWatcher(l *util.Logger, cfg *Config) *Watcher {
	return &Watcher{
		actions: make(map[string]*Action, 0),
		cfg:     cfg,
		logger:  l,
	}
}

// RegisterAction регистрирует действие для наблюдения
func (w *Watcher) RegisterAction(a *Action) error {
	w.actionsMutex.Lock()
	defer w.actionsMutex.Unlock()

	actionName := a.Name()
	if _, ok := w.actions[actionName]; ok {
		return ErrActionAlreadyRegistered
	}
	w.actions[actionName] = a
	return nil
}

// StopAction остановка действия.
func (w *Watcher) StopAction(scheme, act string) error {
	w.actionsMutex.Lock()
	defer w.actionsMutex.Unlock()

	actionName := buildActionName(scheme, act)
	action, ok := w.actions[actionName]
	if !ok {
		return ErrUnknownAction
	}
	delete(w.actions, actionName)

	return action.Stop()
}

// Start запускает Watcher в работу и выходит
func (w *Watcher) start(ctx context.Context) error {
	ticker := time.NewTicker(time.Duration(w.cfg.PingFrequency))
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := w.pingActions(); err != nil {
					w.logger.Errorf("ping got error: %v", err)
				}
			}
		}
	}()

	return nil
}

func (w *Watcher) pingActions() error {
	w.actionsMutex.RLock()
	defer w.actionsMutex.RUnlock()

	for _, action := range w.actions {
		if err := action.Ping(); err != nil {
			w.logger.Errorf("ping for action '%s' failed: %v", action.name, err)
		}
	}
	return nil
}
