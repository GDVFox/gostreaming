package watcher

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GDVFox/gostreaming/meta_node/planner"
	"github.com/GDVFox/gostreaming/util"
	"github.com/mohae/deepcopy"
	"github.com/pkg/errors"
)

type planDescription struct {
	nodes           []*planner.NodePlan
	planNames       map[string]*planner.NodePlan
	planAddrIndexes map[int]int
}

// PlanConfig набор настроек плана
type PlanConfig struct {
	PingFrequency util.Duration
	RetryDelay    util.Duration
}

// Plan отслеживает состояние машин в плане.
type Plan struct {
	planName       string
	machineWatcher *MachineWatcher

	planNodesMutex sync.RWMutex
	plan           *planDescription

	logger *util.Logger
	cfg    *PlanConfig
}

// NewPlan создает новый объект Plan
func NewPlan(plan *planner.Plan, w *MachineWatcher, l *util.Logger, cfg *PlanConfig) *Plan {
	planNames := make(map[string]*planner.NodePlan, len(plan.Nodes))
	planAddrIndexes := make(map[int]int, len(plan.Nodes))
	for i, node := range plan.Nodes {
		planNames[node.Name] = node
		planAddrIndexes[i] = 0
	}

	return &Plan{
		planName:       plan.Name,
		machineWatcher: w,
		plan: &planDescription{
			nodes:           plan.Nodes,
			planNames:       planNames,
			planAddrIndexes: planAddrIndexes,
		},
		logger: l,
		cfg:    cfg,
	}
}

// StartNodes запускает ноды плана в работу.
func (p *Plan) StartNodes(ctx context.Context) error {
	p.planNodesMutex.RLock()
	defer p.planNodesMutex.RUnlock()

	for _, node := range p.plan.nodes {
		if err := p.machineWatcher.sendRunAction(ctx, p.planName, node); err != nil {
			if errors.Cause(err) == ErrNoAction {
				return errors.Wrapf(err, "scheme contains unknown action: %s", node.Action)
			} else if errors.Cause(err) == ErrNoHost {
				return errors.Wrapf(ErrNoHost, "scheme contains unknown host: %s", node.Host)
			}
			return err
		}
	}
	p.logger.Debugf("plan %s: nodes started", p.planName)

	return nil
}

// RunProtection запускает проверку работоспобности.
func (p *Plan) RunProtection(ctx context.Context) error {
	ticker := time.NewTicker(time.Duration(p.cfg.PingFrequency))

ProctionLoop:
	for {
		select {
		case <-ctx.Done():
			break ProctionLoop
		case <-ticker.C:
			p.protectPlan(ctx)
		}
	}

	return p.stopNodes(ctx)
}

func (p *Plan) protectPlan(ctx context.Context) {
	p.logger.Debugf("plan %s: plan check started", p.planName)

	telemetry := p.machineWatcher.getRuntimesTelemetry()

	// останавливаем в обратном порядке.
	for i := len(p.plan.nodes) - 1; i >= 0; i-- {
		node := p.plan.nodes[i]
		runtimeName := buildRuntimeName(p.planName, node.Name)

		_, isRunning := telemetry[runtimeName]
		if isRunning {
			continue
		}

		p.logger.Warnf("plan %s: action %s not found, starting fix", p.planName, node.Name)

		// Пытаемся поднять потерянные действия, пока не получится.
		retryConfig := &util.RetryConfig{Count: 0, Delay: p.cfg.RetryDelay}
		util.Retry(ctx, retryConfig, func() error {
			if err := p.fixAction(ctx, i); err != nil {
				p.logger.Errorf("plan %s: can not fix action %s: %s", p.planName, node.Name, err)
				return err
			}
			return nil
		})

	}

	p.logger.Debugf("plan %s: plan check done", p.planName)
}

func (p *Plan) fixAction(ctx context.Context, actionIndex int) error {
	p.planNodesMutex.RLock()
	planNode := deepcopy.Copy(p.plan.nodes[actionIndex]).(*planner.NodePlan)
	addrIndex := p.plan.planAddrIndexes[actionIndex]
	p.planNodesMutex.RUnlock()

	oldNodeAddr := planNode.Host + ":" + strconv.Itoa(planNode.Port)

	newAddrIndex := (addrIndex + 1) % len(planNode.Addresses)
	newAddr := planNode.Addresses[newAddrIndex]
	newNodeAddr := newAddr.Host + ":" + strconv.Itoa(newAddr.Port)

	planNode.Host = newAddr.Host
	planNode.Port = newAddr.Port

	if err := p.machineWatcher.sendRunAction(ctx, p.planName, planNode); err != nil {
		return fmt.Errorf("plan %s: can not send run action %s: %w", p.planName, planNode.Action, err)
	}

	for _, in := range planNode.In {
		inParts := strings.Split(in, "_")
		inNode, ok := p.plan.planNames[inParts[1]]
		if !ok {
			return fmt.Errorf("plan %s: unknown in: %s", p.planName, in)
		}

		// Пытаемся обновить выходы в вечном retry, так как транзакций нет,
		// а частичный запуск мы не хотим.
		retryConfig := &util.RetryConfig{Count: 0, Delay: p.cfg.RetryDelay}
		util.Retry(ctx, retryConfig, func() error {
			if err := p.machineWatcher.sendChangeOut(ctx, p.planName, oldNodeAddr, newNodeAddr, inNode); err != nil {
				p.logger.Errorf("plan %s: can not change out in action %s: %s", p.planName, inNode.Action, err)
				return err
			}
			return nil
		})

		oldAddrIndex := util.FindStringIndex(inNode.Out, oldNodeAddr)
		inNode.Out[oldAddrIndex] = newNodeAddr
	}

	p.planNodesMutex.Lock()
	p.plan.nodes[actionIndex] = planNode
	p.plan.planAddrIndexes[actionIndex] = newAddrIndex
	p.planNodesMutex.Unlock()
	return nil
}

func (p *Plan) stopNodes(ctx context.Context) error {
	p.planNodesMutex.RLock()
	defer p.planNodesMutex.RUnlock()

	// останавливаем в обратном порядке.
	for i := len(p.plan.nodes) - 1; i >= 0; i-- {
		node := p.plan.nodes[i]
		if err := p.machineWatcher.sendStopAction(ctx, p.planName, node); err != nil {
			if errors.Cause(err) == ErrNoAction {
				return errors.Wrapf(err, "scheme contains unknown action: %s", node.Action)
			} else if errors.Cause(err) == ErrNoHost {
				return errors.Wrapf(ErrNoHost, "scheme contains unknown host: %s", node.Host)
			}
			return err
		}
	}

	p.logger.Debugf("plan %s: nodes stopped", p.planName)
	return nil
}
