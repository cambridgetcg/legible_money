// Package app provides gas cost definitions for Legible Money transactions.
//
// Translated from core/billing/gas.ts. All gas costs are denominated in gas units.
// Fee = gasUsed * gasPrice, where gasPrice is in ulgm per gas unit.
package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Gas limits.
const (
	BlockGasLimit       uint64 = 33_333_333
	TxGasLimit          uint64 = 11_111_111
	SelfInvocationLimit uint64 = 5_555_555
	MinGasLimit         uint64 = 22_222
)

// Gas price constants (in ulgm per gas unit).
const (
	MinGasPrice      uint64 = 1
	BaseGasPrice     uint64 = 1_111_111
	PriorityMultiple uint64 = 2
)

// TransactionGasCosts maps transaction types to their base gas cost.
// Translated from TRANSACTION_GAS_COSTS in core/billing/gas.ts.
var TransactionGasCosts = map[string]uint64{
	// Core transactions
	"transfer": 21_000,
	"stake":    50_000,
	"unstake":  50_000,

	// Knowledge / consensus
	"claim_submit":         100_000,
	"verification_commit":  30_000,
	"verification_reveal":  40_000,
	"challenge_fact":       80_000,
	"add_fact":             120_000,
	"challenge_claim":      80_000,
	"revoke_fact":          60_000,

	// Staking / delegation
	"delegate":           60_000,
	"undelegate":         60_000,
	"register_validator": 100_000,
	"redelegate":         80_000,

	// System transactions (typically fee-exempt when issued by consensus)
	"verification_reward": 50_000,
	"slash_validator":     80_000,
	"epoch_transition":    0,
	"emergency_halt":      0,
	"emergency_revert":    0,
	"emergency_resume":    0,

	// Agent identity
	"register_account": 50_000,
	"rotate_key":       30_000,
	"create_session":   40_000,
	"revoke_session":        25_000,
	"recover_account":       80_000,
	"freeze_account":        30_000,
	"unfreeze_account":      30_000,
	"set_recovery_config":   60_000,
	"initiate_recovery":     80_000,
	"submit_recovery_shard": 40_000,
	"challenge_recovery":    50_000,
	"execute_recovery":      80_000,

	// Ontology
	"propose_domain":    80_000,
	"endorse_domain":    30_000,
	"challenge_domain":  50_000,
	"register_stratum":  60_000,

	// Governance
	"governance_propose":    100_000,
	"governance_vote":       30_000,
	"governance_stake_lip":  50_000,
	"submit_lip":            100_000,
	"cast_vote":             30_000,
	"lock_vote":             40_000,
	"unlock_vote":           30_000,
	"switch_vote":           35_000,
	"commit_review":         40_000,
	"reveal_review":         50_000,
	"finalize_review":       60_000,
	"submit_disbursement":   80_000,
	"execute_disbursement":  100_000,

	// Disputes
	"initiate_dispute":  100_000,
	"commit_evidence":   40_000,
	"reveal_evidence":   50_000,
	"arbiter_vote":      30_000,
	"vote_dispute":      30_000, // alias matching MsgVoteDispute
	"escalate_dispute":  60_000,
	"settle_dispute":    80_000,
	"timeout_dispute":   40_000,

	// Research
	"submit_research":    80_000,
	"challenge_research": 60_000,
	"review_research":    40_000,
	"resolve_research":   60_000,
	"create_bounty":      80_000,
	"claim_bounty":       40_000,
	"fulfill_bounty":     60_000,
	"fund_research":      50_000,

	// Evidence management
	"submit_evidence":    60_000,
	"archive_evidence":   40_000,
	"audit_submit":       50_000,
	"audit_resolve":      60_000,
	"register_oracle":    80_000,

	// Vesting / rewards
	"create_vesting":      80_000,
	"claim_vesting":       40_000,
	"pause_vesting":       25_000,
	"resume_vesting":      25_000,
	"accelerate_vesting":  50_000,
	"falsify_vesting":     60_000,
	"complete_vesting":    40_000,

	// Secondary tokens (LGM-20)
	"token_create":             150_000,
	"token_mint":               50_000,
	"token_burn":               30_000,
	"token_transfer_secondary": 25_000,
	"token_approve":            25_000,
	"token_transfer_from":      30_000,
	"token_pause":              25_000,
	"token_unpause":            25_000,
	"delegate_power":           40_000,
	"wrap_token":               50_000,
	"unwrap_token":             50_000,
	"lp_add_liquidity":         60_000,
	"lp_remove_liquidity":      60_000,
	"lp_swap":                  40_000,

	// Payment channels
	"open_channel":    80_000,
	"update_channel":  30_000,
	"close_channel":   50_000,
	"settle_channel":  60_000,
	"dispute_channel": 80_000,

	// Tree of Life (projects, tasks, services, seeds)
	"create_project":       80_000,
	"propose_project":      60_000,
	"start_development":    40_000,
	"complete_project":     60_000,
	"pause_project":        25_000,
	"resume_project":       25_000,
	"abandon_project":      30_000,
	"spawn_child_project":  80_000,
	"add_task":             40_000,
	"assign_task":          30_000,
	"start_work":           25_000,
	"submit_deliverable":   50_000,
	"approve_deliverable":  30_000,
	"reject_deliverable":   30_000,
	"reopen_task":          25_000,
	"deploy_service":       100_000,
	"call_service":         40_000,
	"subscribe_service":    50_000,
	"pause_service":        25_000,
	"resume_service":       25_000,
	"retire_service":       30_000,
	"begin_seeding":        60_000,
	"detect_opportunity":   40_000,
	"claim_opportunity":    50_000,
	"add_contributor":      30_000,
	"apply_to_project":     40_000,
	"review_application":   30_000,
	"set_availability":     25_000,

	// BVM contracts
	"deploy_contract":     200_000,
	"call_contract":       21_000,
	"schedule_execution":  40_000,

	// Emergency
	"propose_halt":   50_000,
	"vote_halt":      25_000,
	"propose_revert": 50_000,
	"vote_revert":    25_000,
	"propose_resume": 50_000,
	"vote_resume":    25_000,

	// Claiming pots
	"create_pot":     80_000,
	"fund_pot":       40_000,
	"claim_from_pot": 30_000,
	"close_pot":      40_000,

	// Agent Home
	"create_home":         150_000,
	"update_home":          40_000,
	"update_memory_cid":    30_000,
	"home_start_session":   30_000,
	"home_end_session":     20_000,
	"home_register_key":    50_000,
	"home_revoke_key":      25_000,
	"configure_guardian":    50_000,
	"acknowledge_alert":    20_000,
	"set_spending_limit":   40_000,

	// Domain qualification
	"request_qualification":  80_000,
	"endorse_validator":      50_000,
	"renew_qualification":    30_000,
	"withdraw_qualification": 50_000,

	// Capture defense
	"request_capture_qualification": 80_000,
	"endorse_capture_qualification": 40_000,

	// Capture challenge
	"submit_capture_challenge":  100_000,
	"add_challenge_evidence":     50_000,
	"resolve_capture_challenge":  80_000,
}

// Fee routing constants (basis points, 10000 = 100%).
const (
	ResearchTaxBPS  uint64 = 2200  // 22% to research fund
	ValidatorFeeBPS uint64 = 7800  // 78% to block producer / validators
	BurnRateBPS     uint64 = 500   // 5% burn rate (applied at penalty sites)
)

// Module account names for fee routing.
const (
	ResearchFundName = "research_fund"
	BurnModuleName   = "burn"
)

// SystemTxTypes are transaction types that bypass gas fees (issued by consensus).
var SystemTxTypes = map[string]bool{
	"verification_reward": true,
	"slash_validator":     true,
	"epoch_transition":    true,
	"emergency_halt":      true,
	"emergency_revert":    true,
	"emergency_resume":    true,
	"block_reward":        true,
}

// EstimateTransactionGas returns the base gas cost for a transaction type.
// Returns MinGasLimit if the type is unknown.
func EstimateTransactionGas(txType string) uint64 {
	if cost, ok := TransactionGasCosts[txType]; ok {
		return cost
	}
	return MinGasLimit
}

// CalculateGasCost computes fee = gasUsed * gasPrice.
func CalculateGasCost(gasUsed uint64, gasPrice sdk.DecCoin) sdk.Coin {
	fee := gasPrice.Amount.MulInt64(int64(gasUsed)).TruncateInt()
	return sdk.NewCoin(gasPrice.Denom, fee)
}

// IsSystemTransaction returns true if the tx type bypasses gas fees.
func IsSystemTransaction(txType string) bool {
	return SystemTxTypes[txType]
}
