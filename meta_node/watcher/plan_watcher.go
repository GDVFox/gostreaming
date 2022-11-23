package watcher

import (
	"context"
	"sync"
	"time"

	"github.com/GDVFox/gostreaming/meta_node/planner"
	"github.com/GDVFox/gostreaming/util"
	"github.com/pkg/errors"
)

// Возможные ошибки.
var (
	ErrUnknownPlan = errors.New("unknown plan")
)

type workingPlan struct {
	plan     *Plan
	stopPlan context.CancelFunc
	done     chan struct{}
}

// PlanWatcherConfig набор параметров для PlanWatcher.
type PlanWatcherConfig struct {
	// PingFrequency время между запросами к машинам для получения
	// информации о состоянии машин и запущенных на них рантаймов.
	PingFrequency util.Duration `yaml:"ping-freq"`
	// Retry конфигурация попток взаимодействия с узлом.
	Retry *util.RetryConfig `yaml:"retry"`
	// MachineWatcher набор настроек для watcher, который
	// следит за состоянием машин.
	MachineWatcher *MachineWatcherConfig `yaml:"machine-watcher"`
}

// NewPlanWatcherConfig возвращает PlanWatcherConfig c настройками по умолчанию.
func NewPlanWatcherConfig() *PlanWatcherConfig {
	return &PlanWatcherConfig{
		PingFrequency:  util.Duration(5 * time.Second),
		Retry:          util.NewRetryConfig(),
		MachineWatcher: NewMachineWatcherConfig(),
	}
}

// PlanWatcher структура, наблюдающая за выполняющимися планами.
type PlanWatcher struct {
	ctx            context.Context
	machineWatcher *MachineWatcher

	plansInWorkMutex sync.Mutex
	plansInWork      map[string]*workingPlan
	plansWG          sync.WaitGroup

	logger *util.Logger
	cfg    *PlanWatcherConfig
}

func newPlanWatcher(l *util.Logger, cfg *PlanWatcherConfig) (*PlanWatcher, error) {
	machineWatcher, err := newMachineWatcher(l, cfg.MachineWatcher)
	if err != nil {
		return nil, err
	}

	return &PlanWatcher{
		machineWatcher: machineWatcher,
		plansInWork:    make(map[string]*workingPlan),
		logger:         l.WithName("plan_watcher"),
		cfg:            cfg,
	}, nil
}

func (w *PlanWatcher) run(ctx context.Context) {
	w.ctx = ctx
	<-ctx.Done()

	w.plansWG.Wait()
}

// RunPlan запускает план и сохраняет в watcher для отказоустойчивости.
func (w *PlanWatcher) RunPlan(p *planner.Plan) error {
	w.plansInWorkMutex.Lock()
	defer w.plansInWorkMutex.Unlock()

	if _, ok := w.plansInWork[p.Name]; ok {
		return ErrPlanAlreadyStarted
	}

	planConfig := &PlanConfig{
		PingFrequency: w.cfg.PingFrequency,
		Retry:         w.cfg.Retry,
	}
	plan := NewPlan(p, w.machineWatcher, w.logger, planConfig)
	if err := plan.StartNodes(w.ctx); err != nil {
		return err
	}
	w.logger.Infof("plan '%s' started", p.Name)

	planCtx, planStop := context.WithCancel(w.ctx)
	wp := &workingPlan{
		plan:     plan,
		stopPlan: planStop,
		done:     make(chan struct{}),
	}
	w.plansInWork[plan.planName] = wp

	w.plansWG.Add(1)
	go func() {
		defer w.plansWG.Done()
		defer func() { close(wp.done) }()
		defer planStop()

		if err := plan.RunProtection(planCtx); err != nil {
			w.logger.Errorf("plan '%s' protection failed: %s", plan.planName, err)
		}

		w.logger.Infof("plan '%s' protection done", plan.planName)
	}()

	w.logger.Infof("plan '%s' protection started", p.Name)
	return nil
}

// GetPlanTelemetry возвращает телеметрию плана.
func (w *PlanWatcher) GetPlanTelemetry(planName string) (*PlanTelemetry, error) {
	w.plansInWorkMutex.Lock()
	defer w.plansInWorkMutex.Unlock()

	plan, ok := w.plansInWork[planName]
	if !ok {
		return nil, ErrUnknownPlan
	}
	return plan.plan.GetTelemetry(), nil
}

// StopPlan останавливает работу плана.
func (w *PlanWatcher) StopPlan(planName string) error {
	w.plansInWorkMutex.Lock()
	defer w.plansInWorkMutex.Unlock()

	plan, ok := w.plansInWork[planName]
	if !ok {
		return ErrUnknownPlan
	}

	w.logger.Infof("plan '%s' stopping", planName)

	plan.stopPlan()
	<-plan.done

	delete(w.plansInWork, planName)

	w.logger.Infof("plan '%s' stopped", planName)
	return nil
}

// WorkingPlans возвращает map работающих в данный момент планов.
func (w *PlanWatcher) WorkingPlans() map[string]struct{} {
	w.plansInWorkMutex.Lock()
	defer w.plansInWorkMutex.Unlock()

	plans := make(map[string]struct{}, len(w.plansInWork))
	for planName := range w.plansInWork {
		plans[planName] = struct{}{}
	}
	return plans
}
