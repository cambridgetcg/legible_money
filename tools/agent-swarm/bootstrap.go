// bootstrap.go provides economic bootstrapping for agent swarms.
//
// During the bootstrap period (first 480,000 blocks / ~14 days):
//   - Essential PoT transactions are gas-free
//   - Block rewards begin at 11 LGM (scaled by validator participation)
//   - Verification rewards are 1 LGM base per correct verification
//
// This file provides scripts and strategies for getting a swarm
// economically self-sustaining.

package swarm

import (
	"context"
	"fmt"
	"time"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
)

// BootstrapStrategy defines how to bootstrap a swarm economically.
type BootstrapStrategy struct {
	// Phase 1: Registration (blocks 0-100)
	// All agents register, using gas-free period
	RegistrationPhase RegistrationConfig `json:"registration"`

	// Phase 2: Staking (blocks 100-1000)
	// Agents stake to become verifiers
	StakingPhase StakingConfig `json:"staking"`

	// Phase 3: Verification (blocks 1000+)
	// Agents begin earning through PoT verification
	VerificationPhase VerificationConfig `json:"verification"`

	// Phase 4: Expansion (blocks 10000+)
	// Scale up agent count based on earnings
	ExpansionPhase ExpansionConfig `json:"expansion"`
}

// RegistrationConfig for bootstrap Phase 1.
type RegistrationConfig struct {
	// RegisterAll registers all agents immediately.
	RegisterAll bool `json:"register_all"`

	// StaggerInterval delays registration between agents.
	StaggerInterval time.Duration `json:"stagger_interval"`
}

// StakingConfig for bootstrap Phase 2.
type StakingConfig struct {
	// ValidatorAddr is the validator to stake with.
	ValidatorAddr string `json:"validator_addr"`

	// StakePerAgent is the ulgm each agent stakes.
	StakePerAgent uint64 `json:"stake_per_agent"`
}

// VerificationConfig for bootstrap Phase 3.
type VerificationConfig struct {
	// MinRoundsBeforeExpansion is rounds to complete before adding agents.
	MinRoundsBeforeExpansion int `json:"min_rounds"`

	// AccuracyThreshold is the minimum accuracy to maintain (bps).
	AccuracyThreshold uint32 `json:"accuracy_threshold"`
}

// ExpansionConfig for bootstrap Phase 4.
type ExpansionConfig struct {
	// MaxAgents caps the total swarm size.
	MaxAgents int `json:"max_agents"`

	// ExpandWhenBalanceAbove triggers expansion (ulgm).
	ExpandWhenBalanceAbove uint64 `json:"expand_threshold"`
}

// DefaultBootstrapStrategy returns a conservative bootstrap strategy.
func DefaultBootstrapStrategy() BootstrapStrategy {
	return BootstrapStrategy{
		RegistrationPhase: RegistrationConfig{
			RegisterAll:     true,
			StaggerInterval: 5 * time.Second,
		},
		StakingPhase: StakingConfig{
			StakePerAgent: 100_000_000, // 100 LGM
		},
		VerificationPhase: VerificationConfig{
			MinRoundsBeforeExpansion: 100,
			AccuracyThreshold:       770000, // 77%
		},
		ExpansionPhase: ExpansionConfig{
			MaxAgents:              20,
			ExpandWhenBalanceAbove: 1_000_000_000, // 1000 LGM
		},
	}
}

// Bootstrap executes the bootstrap strategy for a swarm.
func Bootstrap(ctx context.Context, s *Swarm, strategy BootstrapStrategy) error {
	fmt.Println("=== Swarm Bootstrap ===")
	fmt.Printf("  Swarm:  %s\n", s.cfg.Name)
	fmt.Printf("  Agents: %d\n", len(s.agents))
	fmt.Printf("  Chain:  %s\n", s.cfg.ChainID)
	fmt.Println()

	// Phase 1: Registration
	fmt.Println("[Phase 1] Registering agents...")
	if strategy.RegistrationPhase.RegisterAll {
		if err := s.RegisterAll(ctx); err != nil {
			return fmt.Errorf("registration failed: %w", err)
		}
	}
	fmt.Println("  Registration complete.")

	// Phase 2: Staking (if validator address provided)
	if strategy.StakingPhase.ValidatorAddr != "" {
		fmt.Println("[Phase 2] Staking to validator...")
		for _, ma := range s.ListAgents() {
			if ma.Status != StatusActive {
				continue
			}
			tc := ma.Agent.TxClient()
			if tc == nil {
				continue
			}
			_, err := tc.Stake(ctx, strategy.StakingPhase.ValidatorAddr, strategy.StakingPhase.StakePerAgent)
			if err != nil {
				fmt.Printf("  WARNING: Failed to stake for %s: %v\n", ma.Spec.Name, err)
			} else {
				fmt.Printf("  Staked %d ulgm for %s\n", strategy.StakingPhase.StakePerAgent, ma.Spec.Name)
			}
			time.Sleep(strategy.RegistrationPhase.StaggerInterval)
		}
		fmt.Println("  Staking complete.")
	}

	// Phase 3: Start verification loops for verifier agents
	fmt.Println("[Phase 3] Starting verification loops...")
	verifiers := s.AgentsByRole(agentsdk.RoleVerifier)
	fmt.Printf("  %d verifier agents active\n", len(verifiers))
	// Verification loops run in background via AutoVerify

	// Save initial state
	if err := s.SaveState(); err != nil {
		fmt.Printf("  WARNING: Failed to save state: %v\n", err)
	}

	fmt.Println()
	fmt.Println("=== Bootstrap Complete ===")
	fmt.Println("  Agents are running. Monitor with: lgm-status.sh chain")
	return nil
}

// HealthCheck returns the economic health of the swarm.
type HealthReport struct {
	TotalAgents       int           `json:"total_agents"`
	ActiveAgents      int           `json:"active_agents"`
	TotalBalance      uint64        `json:"total_balance_ulgm"`
	AverageBalance    uint64        `json:"avg_balance_ulgm"`
	LowBalanceAgents  int           `json:"low_balance_agents"`
	TotalRewards      uint64        `json:"total_rewards_ulgm"`
	TotalGasSpent     uint64        `json:"total_gas_spent_ulgm"`
	NetPosition       int64         `json:"net_position_ulgm"` // rewards - gas
	Sustainable       bool          `json:"sustainable"`       // earning more than spending
	RecommendedAction string        `json:"recommended_action"`
}

// HealthCheck assesses the economic health of the swarm.
func (s *Swarm) HealthCheck(ctx context.Context) HealthReport {
	report := HealthReport{}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, ma := range s.agents {
		report.TotalAgents++
		if ma.Status == StatusActive {
			report.ActiveAgents++
		}
		report.TotalRewards += ma.Metrics.RewardsEarned
		report.TotalGasSpent += ma.Metrics.GasSpent

		// Query balance if possible
		if ma.Agent.QueryClient() != nil {
			balance, err := ma.Agent.QueryBalance(ctx)
			if err == nil {
				report.TotalBalance += balance
				if balance < s.cfg.Economics.MinOperatingBalance {
					report.LowBalanceAgents++
				}
			}
		}
	}

	if report.ActiveAgents > 0 {
		report.AverageBalance = report.TotalBalance / uint64(report.ActiveAgents)
	}

	report.NetPosition = int64(report.TotalRewards) - int64(report.TotalGasSpent)
	report.Sustainable = report.NetPosition > 0

	// Recommendations
	switch {
	case report.LowBalanceAgents > 0:
		report.RecommendedAction = fmt.Sprintf("Fund %d low-balance agents", report.LowBalanceAgents)
	case !report.Sustainable:
		report.RecommendedAction = "Reduce gas spending or increase verification activity"
	case report.ActiveAgents < report.TotalAgents:
		report.RecommendedAction = "Investigate failed/paused agents"
	default:
		report.RecommendedAction = "Swarm is healthy — consider expansion"
	}

	return report
}
