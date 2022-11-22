package watcher

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/GDVFox/gostreaming/util"
	"github.com/GDVFox/gostreaming/util/message"
)

// Возможные ошибки.
var (
	ErrRuntimeAlreadyRegistered = errors.New("action already registered")
	ErrUnknownRuntime           = errors.New("unknown action")
)

type workingRuntime struct {
	runtime      *Runtime
	pingsFailed  int
	oldestOutput uint32
}

// Config набор настроек для Watcher
type Config struct {
	// PingsToStop количество неудачных ping запросов,
	// после которых рантайм признается отказавшим.
	PingsToStop int `yaml:"pings-to-stop"`
	// PingFrequency время между запросами к рантаймам для получения состояния.
	PingFrequency util.Duration `yaml:"ping-freq"`
}

// NewConfig создает новый Config с настройками по-умолчанию.
func NewConfig() *Config {
	return &Config{
		PingsToStop:   3,
		PingFrequency: util.Duration(5 * time.Second),
	}
}

// Watcher структура для контроля запущенных действий.
type Watcher struct {
	runtimesMutex sync.RWMutex
	runtimes      map[string]*workingRuntime

	cfg    *Config
	logger *util.Logger
}

// NewWatcher создает новый объект watcher
func newWatcher(l *util.Logger, cfg *Config) *Watcher {
	return &Watcher{
		runtimes: make(map[string]*workingRuntime),
		cfg:      cfg,
		logger:   l.WithName("watcher"),
	}
}

// StartRuntime запускает регистрирует действие для наблюдения
func (w *Watcher) StartRuntime(ctx context.Context, r *Runtime) error {
	w.runtimesMutex.Lock()
	defer w.runtimesMutex.Unlock()

	if err := r.Start(ctx); err != nil {
		return err
	}

	runtimeName := r.Name()
	if _, ok := w.runtimes[runtimeName]; ok {
		return ErrRuntimeAlreadyRegistered
	}

	w.runtimes[runtimeName] = &workingRuntime{
		runtime:     r,
		pingsFailed: 0,
	}

	w.logger.Infof("runtime '%s' started", runtimeName)
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

	if err := runtime.runtime.Stop(); err != nil {
		return err
	}

	w.logger.Infof("runtime '%s' stopped", runtimeName)
	return nil
}

// ChangeOutRuntime изменяет один из выходных потоков рантайма.
func (w *Watcher) ChangeOutRuntime(schemeName, actionName, oldOut, newOut string) error {
	w.runtimesMutex.Lock()
	defer w.runtimesMutex.Unlock()

	runtimeName := buildRuntimeName(schemeName, actionName)
	runtime, ok := w.runtimes[runtimeName]
	if !ok {
		return ErrUnknownRuntime
	}

	if err := runtime.runtime.ChangeOut(oldOut, newOut); err != nil {
		return err
	}

	w.logger.Infof("runtime '%s' changed out %s -> %s", runtimeName, oldOut, newOut)
	return nil
}

// GetRuntimesTelemetry возвращает информацию о состояниях действий.
func (w *Watcher) GetRuntimesTelemetry() []*message.RuntimeTelemetry {
	w.runtimesMutex.Lock()
	defer w.runtimesMutex.Unlock()

	runtimes := make([]*message.RuntimeTelemetry, 0, len(w.runtimes))
	for _, runtime := range w.runtimes {
		status := message.RuntimeStatusOK
		if runtime.pingsFailed > 0 {
			status = message.RuntimeStatusPending
		}

		telemetry := &message.RuntimeTelemetry{
			SchemeName:   runtime.runtime.SchemeName(),
			ActionName:   runtime.runtime.ActionName(),
			Status:       status,
			OldestOutput: runtime.oldestOutput,
		}

		runtimes = append(runtimes, telemetry)
	}

	// Здесь Debugf, так как этот метод вызывается на каждый ping от meta_node.
	w.logger.Debugf("telemetry for %d runtimes loaded", len(runtimes))
	return runtimes
}

// Start запускает Watcher в работу и выходит
func (w *Watcher) run(ctx context.Context) {
	w.logger.Info("watcher started")
	defer w.logger.Info("watcher stopped")

	ticker := time.NewTicker(time.Duration(w.cfg.PingFrequency))
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.pingRuntimes()
		}
	}
}

func (w *Watcher) pingRuntimes() {
	defer w.logger.Debugf("ping runtimes done")

	w.runtimesMutex.RLock()
	defer w.runtimesMutex.RUnlock()

	for runtimeName, runtime := range w.runtimes {
		telemetry, err := runtime.runtime.Ping()
		if err != nil {
			runtime.pingsFailed++
			w.logger.Warnf("ping for runtime '%s' failed: %v", runtimeName, err)

			if runtime.pingsFailed >= w.cfg.PingsToStop {
				w.logger.Warnf("runtime '%s' %d pings failed: stopping runtime", runtimeName, runtime.pingsFailed)

				delete(w.runtimes, runtimeName)
				if err := runtime.runtime.Stop(); err != nil {
					w.logger.Errorf("runtime '%s' stop failed: skipping runtime: %v", runtimeName, err)
				}
				w.logger.Warnf("runtime '%s' stopped", runtimeName)
			}

			continue
		}
		runtime.oldestOutput = telemetry.OldestOutput
		runtime.pingsFailed = 0
	}
}
