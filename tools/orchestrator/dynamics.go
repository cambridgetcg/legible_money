package orchestrator

import (
	"context"
	"log"
	"time"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
	swarm "github.com/legible-money/legible/tools/agent-swarm"
)

// DynamicsController monitors economic health and adjusts sections dynamically.
type DynamicsController struct {
	conductor *Conductor
	logger    *log.Logger

	cancel context.CancelFunc
	done   chan struct{}
}

// NewDynamicsController creates a controller that monitors and adjusts the orchestra.
func NewDynamicsController(conductor *Conductor, logger *log.Logger) *DynamicsController {
	if logger == nil {
		logger = log.Default()
	}
	return &DynamicsController{
		conductor: conductor,
		logger:    logger,
		done:      make(chan struct{}),
	}
}

// Start begins the background monitoring loop.
func (d *DynamicsController) Start(ctx context.Context) {
	monCtx, cancel := context.WithCancel(ctx)
	d.cancel = cancel

	go d.loop(monCtx)
}

// Stop halts the monitoring loop and waits for it to finish.
func (d *DynamicsController) Stop() {
	if d.cancel != nil {
		d.cancel()
		<-d.done
	}
}

func (d *DynamicsController) loop(ctx context.Context) {
	defer close(d.done)

	score := d.conductor.Score()
	pollInterval := score.Tempo.PollInterval
	if pollInterval <= 0 {
		pollInterval = 5 * time.Second
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			state := d.conductor.State()
			if state != StatePerforming && state != StatePaused {
				continue
			}
			d.checkBalances(ctx)
			d.checkSectionLoad()
		}
	}
}

// checkBalances queries agent balances and pauses/unpauses members accordingly.
func (d *DynamicsController) checkBalances(ctx context.Context) {
	minBalance := d.conductor.score.Economics.MinOperatingBalance
	if minBalance == 0 {
		return
	}

	for _, sec := range d.conductor.Sections() {
		for _, member := range sec.Members {
			qc := member.Managed.Agent.QueryClient()
			if qc == nil {
				continue
			}

			bal, err := qc.Balance(ctx, member.Managed.Agent.Address())
			if err != nil {
				continue
			}

			if bal < minBalance && !member.Paused {
				member.Paused = true
				member.Managed.Status = swarm.StatusDegraded
				d.conductor.emit(Event{
					Type:    EventBalanceLow,
					Section: sec.Name,
					Agent:   member.Managed.Spec.Name,
					Message: "balance below minimum operating threshold",
					Data:    map[string]string{"balance": uintToStr(bal), "min": uintToStr(minBalance)},
				})
			} else if bal >= minBalance && member.Paused {
				member.Paused = false
				member.Managed.Status = swarm.StatusActive
				d.conductor.emit(Event{
					Type:    EventBalanceOK,
					Section: sec.Name,
					Agent:   member.Managed.Spec.Name,
					Message: "balance recovered above minimum",
				})
			}
		}
	}
}

// checkSectionLoad examines active part counts and scales sections if needed.
func (d *DynamicsController) checkSectionLoad() {
	for _, sec := range d.conductor.Sections() {
		active := sec.ActiveCount()
		totalMembers := len(sec.Members)

		// If all active members are heavily loaded, log a warning.
		if active > 0 {
			totalActive := 0
			for _, m := range sec.Members {
				if !m.Paused && m.Managed.Status != swarm.StatusFailed {
					totalActive += m.ActiveParts
				}
			}
			avgLoad := totalActive / active
			if avgLoad > 3 && totalMembers < sec.Spec.MaxCount {
				d.logger.Printf("[dynamics] section %s: avg load %d, could scale up (current: %d, max: %d)",
					sec.Name, avgLoad, totalMembers, sec.Spec.MaxCount)
				d.conductor.emit(Event{
					Type:    EventSectionScaled,
					Section: sec.Name,
					Message: "section overloaded, scale-up recommended",
				})
			}
		}
	}
}

// EconomicSnapshot computes the current economic health from agent balances.
func EconomicSnapshot(ctx context.Context, sections map[SectionName]*Section, economics EconomicsSpec) EconomicHealth {
	h := EconomicHealth{}

	for _, sec := range sections {
		for _, member := range sec.Members {
			qc := member.Managed.Agent.QueryClient()
			if qc == nil {
				continue
			}
			bal, err := qc.Balance(ctx, member.Managed.Agent.Address())
			if err != nil {
				continue
			}
			h.TotalBalance += bal
			h.TotalRewards += member.Managed.Metrics.RewardsEarned
			h.TotalGasSpent += member.Managed.Metrics.GasSpent

			if bal < economics.MinOperatingBalance {
				h.LowBalanceAgents++
			}
		}
	}

	h.NetPosition = int64(h.TotalRewards) - int64(h.TotalGasSpent)
	h.Sustainable = h.TotalRewards >= h.TotalGasSpent
	if economics.MaxTotalGasSpend > h.TotalGasSpent {
		h.BudgetRemaining = economics.MaxTotalGasSpend - h.TotalGasSpent
	}

	return h
}

// Compile-time interface checks.
var (
	_ agentsdk.QueryClient = (*RehearsalQueryClient)(nil)
	_ agentsdk.TxClient    = (*RehearsalTxClient)(nil)
)
