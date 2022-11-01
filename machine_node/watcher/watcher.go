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
	ErrRuntimeAlreadyRegistered = errors.New("action already registered")
	ErrUnknownRuntime           = errors.New("unknown action")
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
	runtimesMutex sync.RWMutex
	runtimes      map[string]*Runtime

	cfg    *Config
	logger *util.Logger
}

// NewWatcher создает новый объект watcher
func newWatcher(l *util.Logger, cfg *Config) *Watcher {
	return &Watcher{
		runtimes: make(map[string]*Runtime, 0),
		cfg:      cfg,
		logger:   l,
	}
}

// RegisterRuntime регистрирует действие для наблюдения
func (w *Watcher) RegisterRuntime(a *Runtime) error {
	w.runtimesMutex.Lock()
	defer w.runtimesMutex.Unlock()

	runtimeName := a.Name()
	if _, ok := w.runtimes[runtimeName]; ok {
		return ErrRuntimeAlreadyRegistered
	}
	w.runtimes[runtimeName] = a
	return nil
}

// StopRuntime остановка действия.
func (w *Watcher) StopRuntime(schemeName, actionName string) error {
	w.runtimesMutex.Lock()
	defer w.runtimesMutex.Unlock()

	runtimeName := buildRuntimeName(schemeName, actionName)
	runtime, ok := w.runtimes[runtimeName]
	if !ok {
		return ErrUnknownRuntime
	}
	delete(w.runtimes, runtimeName)

	return runtime.Stop()
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
				if err := w.pingRuntimes(); err != nil {
					w.logger.Errorf("ping got error: %v", err)
				}
			}
		}
	}()

	return nil
}

func (w *Watcher) pingRuntimes() error {
	w.runtimesMutex.RLock()
	defer w.runtimesMutex.RUnlock()

	for _, runtime := range w.runtimes {
		if err := runtime.Ping(); err != nil {
			w.logger.Errorf("ping for runtime '%s' failed: %v", runtime.Name(), err)
		}
	}
	return nil
}
