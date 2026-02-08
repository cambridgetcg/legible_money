package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

// AggregateVerificationResult computes the final outcome of a verification round
// using stake-weighted confidence aggregation.
//
// Algorithm:
//  1. Weight each reveal's confidence by the validator's effective stake
//  2. Compute agreement ratio = majority_weight / total_weight
//  3. Determine outcome: accepted, rejected, contested, insufficient, inconclusive
//  4. Calculate rewards/slashes for each participant
func (k Keeper) AggregateVerificationResult(
	ctx sdk.Context,
	round types.VerificationRound,
) (types.VerificationResult, error) {
	params := k.GetParams(ctx)

	// Gather stake weights for all revealing validators
	stakeMap := make(map[string]uint64, len(round.Reveals))
	for validatorAddr := range round.Reveals {
		stakeMap[validatorAddr] = k.GetEffectiveStakeForValidator(ctx, validatorAddr)
	}

	// Compute stake-weighted verdicts
	var acceptWeight, rejectWeight, abstainWeight, totalWeight uint64
	for validatorAddr, reveal := range round.Reveals {
		stake := stakeMap[validatorAddr]
		if stake == 0 {
			stake = 1 // Prevent zero-weight for Tier 0 validators
		}
		weightedConf := reveal.Confidence * stake / 1_000_000 // normalize

		switch reveal.Verdict {
		case "accept":
			acceptWeight += weightedConf
		case "reject":
			rejectWeight += weightedConf
		case "abstain":
			abstainWeight += weightedConf
		}
		totalWeight += weightedConf
	}

	result := types.VerificationResult{
		ParticipantCount: uint32(len(round.Reveals)),
		Rewards:          make(map[string]uint64),
		Slashes:          make(map[string]uint64),
	}

	if totalWeight == 0 {
		result.Verdict = "insufficient"
		return result, nil
	}

	// Participation ratio: reveals / selected validators
	participationRatio := uint64(1_000_000) // Default 100% if no selection tracking
	if len(round.SelectedValidators) > 0 {
		participationRatio = (uint64(len(round.Reveals)) * 1_000_000) / uint64(len(round.SelectedValidators))
	}

	// Check quorum
	if participationRatio < params.QuorumThreshold {
		result.Verdict = "insufficient"
		result.Confidence = 0
		return result, nil
	}

	// Determine majority verdict and agreement ratio
	var majorWeight uint64
	var majorVerdict string
	if acceptWeight >= rejectWeight && acceptWeight >= abstainWeight {
		majorWeight = acceptWeight
		majorVerdict = "accepted"
	} else if rejectWeight > acceptWeight && rejectWeight >= abstainWeight {
		majorWeight = rejectWeight
		majorVerdict = "rejected"
	} else {
		majorWeight = abstainWeight
		majorVerdict = "inconclusive"
	}

	agreementRatio := (majorWeight * 1_000_000) / totalWeight
	result.AgreementRatio = agreementRatio
	result.TotalWeight = totalWeight

	// Apply thresholds to determine outcome
	if agreementRatio >= params.AcceptanceThreshold {
		result.Verdict = majorVerdict
		result.Confidence = agreementRatio
	} else if agreementRatio >= 550000 { // 55% contested threshold
		result.Verdict = "contested"
		result.Confidence = agreementRatio
	} else {
		result.Verdict = "inconclusive"
		result.Confidence = agreementRatio
	}

	// Calculate rewards and slashes
	k.calculateRewardsAndSlashes(ctx, round, &result, stakeMap)

	return result, nil
}

// GetEffectiveStakeForValidator returns the effective stake for a validator,
// including virtual stake for Tier 0/1 validators.
func (k Keeper) GetEffectiveStakeForValidator(ctx sdk.Context, validatorAddr string) uint64 {
	return k.stakingKeeper.GetEffectiveStake(ctx, validatorAddr)
}

// calculateRewardsAndSlashes computes reward and slash amounts for each
// participant based on their verdict alignment with the final outcome.
//
// Reward tiers (from TS chain consensus/pot/state_integration.ts):
//   - Correct verdict: base_reward * tier_multiplier
//   - Wrong verdict: slashed proportionally
//   - Missed reveal (committed but didn't reveal): missed_verification slash
func (k Keeper) calculateRewardsAndSlashes(
	ctx sdk.Context,
	round types.VerificationRound,
	result *types.VerificationResult,
	stakeMap map[string]uint64,
) {
	if result.Verdict == "insufficient" || result.Verdict == "inconclusive" {
		return
	}

	// Base reward per correct verification (in ulgm)
	// Phase 2: Fixed 100000 ulgm (0.1 LGM). Phase 3: configurable via params.
	baseReward := uint64(100000)

	for validatorAddr, reveal := range round.Reveals {
		// Check if validator's verdict aligns with final outcome
		correct := (result.Verdict == "accepted" && reveal.Verdict == "accept") ||
			(result.Verdict == "rejected" && reveal.Verdict == "reject")

		if correct {
			// Reward correct verdict. Multiplier based on confidence accuracy.
			reward := baseReward
			result.Rewards[validatorAddr] = reward
		} else if reveal.Verdict != "abstain" {
			// Slash wrong verdict (abstain is not slashed)
			stake := stakeMap[validatorAddr]
			slashAmount := stake / 100 // 1% slash
			if slashAmount > 0 {
				result.Slashes[validatorAddr] = slashAmount
			}
		}
	}

	// Slash validators who committed but failed to reveal
	for validatorAddr := range round.Commitments {
		if _, revealed := round.Reveals[validatorAddr]; !revealed {
			// Lookup stake for non-revealing validator (not in stakeMap)
			stake := stakeMap[validatorAddr]
			if stake == 0 {
				stake = k.GetEffectiveStakeForValidator(ctx, validatorAddr)
			}
			// Missed reveal penalty (harsher than wrong verdict)
			slashAmount := stake / 50 // 2% slash for missed reveal
			if slashAmount > 0 {
				result.Slashes[validatorAddr] = slashAmount
			}
		}
	}
}

// ApplyRewardsAndSlashes distributes rewards and applies slashes from a
// verification result to validator accounts.
func (k Keeper) ApplyRewardsAndSlashes(
	ctx sdk.Context,
	result types.VerificationResult,
) error {
	// Apply rewards
	for validatorAddr, amount := range result.Rewards {
		if err := k.RecordVerificationOutcome(ctx, validatorAddr, true); err != nil {
			k.Logger(ctx).Error("failed to record correct verification",
				"validator", validatorAddr, "error", err)
		}
		_ = amount // Phase 2: BankKeeper coin transfers in Phase 3
	}

	// Apply slashes
	for validatorAddr, amount := range result.Slashes {
		if err := k.SlashMissedVerification(ctx, validatorAddr, fmt.Sprintf("%d", amount)); err != nil {
			k.Logger(ctx).Error("failed to apply slash",
				"validator", validatorAddr, "amount", amount, "error", err)
		}
		if err := k.RecordVerificationOutcome(ctx, validatorAddr, false); err != nil {
			k.Logger(ctx).Error("failed to record incorrect verification",
				"validator", validatorAddr, "error", err)
		}
	}

	return nil
}
