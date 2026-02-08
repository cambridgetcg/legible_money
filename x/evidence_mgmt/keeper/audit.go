package keeper

import (
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/evidence_mgmt/types"
)

// SubmitAudit validates and stores a new audit submission.
func (k Keeper) SubmitAudit(ctx sdk.Context, audit types.EvidenceAudit) error {
	params := k.GetParams(ctx)

	// Validate evidence exists and is in auditable state
	evidence, found := k.GetEvidence(ctx, audit.EvidenceId)
	if !found {
		return fmt.Errorf("%w: %s", types.ErrEvidenceNotFound, audit.EvidenceId)
	}
	if evidence.Status != types.EvidenceStatusSubmitted {
		return fmt.Errorf("%w: evidence %s is %s, not submitted",
			types.ErrInvalidEvidence, audit.EvidenceId, evidence.Status)
	}

	// Check audit window is still open
	if uint64(ctx.BlockHeight()) > evidence.SubmittedAt+params.AuditWindow {
		return fmt.Errorf("%w: evidence %s audit window closed at height %d",
			types.ErrAuditWindowClosed, audit.EvidenceId, evidence.SubmittedAt+params.AuditWindow)
	}

	// Validate score range
	if audit.Score > 1000000 {
		return fmt.Errorf("%w: score %d exceeds maximum 1000000",
			types.ErrInvalidScore, audit.Score)
	}

	// Check for duplicate audit (same auditor, same evidence)
	existingAudits := k.GetAuditsForEvidence(ctx, audit.EvidenceId)
	for _, ea := range existingAudits {
		if ea.Auditor == audit.Auditor {
			return fmt.Errorf("%w: auditor %s already audited evidence %s",
				types.ErrDuplicateAudit, audit.Auditor, audit.EvidenceId)
		}
	}

	// Validate minimum audit stake
	stakeAmount := new(big.Int)
	if _, ok := stakeAmount.SetString(audit.Stake, 10); !ok || stakeAmount.Sign() <= 0 {
		return fmt.Errorf("%w: invalid stake amount", types.ErrInsufficientAuditStake)
	}
	minStake := new(big.Int)
	if _, ok := minStake.SetString(params.MinAuditStake, 10); !ok {
		minStake.SetInt64(22000000) // fallback
	}
	if stakeAmount.Cmp(minStake) < 0 {
		return fmt.Errorf("%w: stake %s below minimum %s",
			types.ErrInsufficientAuditStake, audit.Stake, params.MinAuditStake)
	}

	// Deduct stake from auditor
	auditorAddr, err := sdk.AccAddressFromBech32(audit.Auditor)
	if err != nil {
		return fmt.Errorf("invalid auditor address: %w", err)
	}
	stakeCoin := sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(stakeAmount))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(
		ctx, auditorAddr, types.ModuleName, sdk.NewCoins(stakeCoin),
	); err != nil {
		return fmt.Errorf("%w: %v", types.ErrInsufficientAuditStake, err)
	}

	// Set initial state
	audit.Status = types.AuditStatusPending
	audit.RewardAmount = "0"
	audit.SlashAmount = "0"
	audit.AccuracyDelta = 0

	k.SetAudit(ctx, audit)

	// Update evidence audit count
	evidence.AuditCount++
	k.SetEvidence(ctx, evidence)

	k.Logger(ctx).Info("audit submitted",
		"audit_id", audit.Id,
		"evidence_id", audit.EvidenceId,
		"auditor", audit.Auditor,
		"score", audit.Score,
		"stake", audit.Stake,
	)

	return nil
}

// ComputeConsensus computes the stake-weighted average score for an evidence's audits.
func (k Keeper) ComputeConsensus(ctx sdk.Context, evidenceId string) (types.AuditConsensus, error) {
	audits := k.GetAuditsForEvidence(ctx, evidenceId)
	if len(audits) == 0 {
		return types.AuditConsensus{}, fmt.Errorf("no audits for evidence %s", evidenceId)
	}

	totalStake := new(big.Int)
	weightedSum := new(big.Int)
	var validCount uint32

	// First pass: compute stake-weighted average
	for _, audit := range audits {
		if audit.Status == types.AuditStatusExpired {
			continue
		}
		stake := new(big.Int)
		if _, ok := stake.SetString(audit.Stake, 10); !ok {
			continue
		}
		score := new(big.Int).SetUint64(audit.Score)
		contribution := new(big.Int).Mul(stake, score)
		weightedSum.Add(weightedSum, contribution)
		totalStake.Add(totalStake, stake)
		validCount++
	}

	if totalStake.Sign() == 0 {
		return types.AuditConsensus{}, fmt.Errorf("zero total stake for evidence %s", evidenceId)
	}

	weightedScore := new(big.Int).Div(weightedSum, totalStake)

	// Second pass: compute variance (stake-weighted sum of squared deviations)
	// variance = sum(stake_i * (score_i - mean)^2) / totalStake
	varianceSum := new(big.Int)
	for _, audit := range audits {
		if audit.Status == types.AuditStatusExpired {
			continue
		}
		stake := new(big.Int)
		if _, ok := stake.SetString(audit.Stake, 10); !ok {
			continue
		}
		scoreBig := new(big.Int).SetUint64(audit.Score)
		diff := new(big.Int).Sub(scoreBig, weightedScore)
		diffSq := new(big.Int).Mul(diff, diff)
		stakeWeightedDiff := new(big.Int).Mul(stake, diffSq)
		varianceSum.Add(varianceSum, stakeWeightedDiff)
	}
	variance := new(big.Int).Div(varianceSum, totalStake)

	// High variance threshold: if variance > 100000^2 = 10^10
	// (scores differ by more than 10% on average)
	highVarianceThreshold := new(big.Int).SetUint64(10000000000) // 100000^2
	isHighVariance := variance.Cmp(highVarianceThreshold) > 0

	// Cap variance to uint64 range
	varianceUint64 := uint64(0)
	if variance.IsUint64() {
		varianceUint64 = variance.Uint64()
	}

	consensus := types.AuditConsensus{
		EvidenceId:     evidenceId,
		WeightedScore:  weightedScore.Uint64(),
		TotalStake:     totalStake.String(),
		AuditCount:     validCount,
		Variance:       varianceUint64,
		IsHighVariance: isHighVariance,
		ComputedAt:     uint64(ctx.BlockHeight()),
	}

	return consensus, nil
}

// ResolveAudits resolves all audits for an evidence, rewarding accurate ones
// and slashing inaccurate ones based on their deviation from the ground truth score.
func (k Keeper) ResolveAudits(ctx sdk.Context, evidenceId string, groundTruth uint64) error {
	params := k.GetParams(ctx)
	audits := k.GetAuditsForEvidence(ctx, evidenceId)

	for _, audit := range audits {
		if audit.Status != types.AuditStatusPending {
			continue
		}

		accurate := IsAccurate(audit.Score, groundTruth, params.AccuracyThreshold)

		stakeAmount := new(big.Int)
		if _, ok := stakeAmount.SetString(audit.Stake, 10); !ok {
			continue
		}

		// Compute accuracy delta (basis points of distance from ground truth)
		var delta uint64
		if audit.Score > groundTruth {
			delta = audit.Score - groundTruth
		} else {
			delta = groundTruth - audit.Score
		}
		audit.AccuracyDelta = delta

		auditorAddr, err := sdk.AccAddressFromBech32(audit.Auditor)
		if err != nil {
			k.Logger(ctx).Error("invalid auditor address during resolution",
				"auditor", audit.Auditor, "error", err)
			continue
		}

		if accurate {
			// Reward: return stake + reward percentage
			rewardBps := new(big.Int).SetUint64(params.RewardRate)
			reward := new(big.Int).Mul(stakeAmount, rewardBps)
			reward.Div(reward, new(big.Int).SetUint64(1000000))

			// Return stake + reward to auditor
			totalReturn := new(big.Int).Add(stakeAmount, reward)
			returnCoin := sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(totalReturn))
			if err := k.bankKeeper.SendCoinsFromModuleToAccount(
				ctx, types.ModuleName, auditorAddr, sdk.NewCoins(returnCoin),
			); err != nil {
				k.Logger(ctx).Error("failed to reward auditor",
					"auditor", audit.Auditor, "error", err)
				continue
			}

			audit.Status = types.AuditStatusRewarded
			audit.RewardAmount = reward.String()
			audit.ResolvedAt = uint64(ctx.BlockHeight())
		} else {
			// Slash: deduct slash percentage from stake, return remainder
			slashBps := new(big.Int).SetUint64(params.SlashRate)
			slashAmount := new(big.Int).Mul(stakeAmount, slashBps)
			slashAmount.Div(slashAmount, new(big.Int).SetUint64(1000000))

			remainder := new(big.Int).Sub(stakeAmount, slashAmount)
			if remainder.Sign() > 0 {
				returnCoin := sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(remainder))
				if err := k.bankKeeper.SendCoinsFromModuleToAccount(
					ctx, types.ModuleName, auditorAddr, sdk.NewCoins(returnCoin),
				); err != nil {
					k.Logger(ctx).Error("failed to return remainder to auditor",
						"auditor", audit.Auditor, "error", err)
					continue
				}
			}

			audit.Status = types.AuditStatusSlashed
			audit.SlashAmount = slashAmount.String()
			audit.ResolvedAt = uint64(ctx.BlockHeight())
		}

		k.SetAudit(ctx, audit)

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				"audit_resolved",
				sdk.NewAttribute("audit_id", audit.Id),
				sdk.NewAttribute("evidence_id", audit.EvidenceId),
				sdk.NewAttribute("auditor", audit.Auditor),
				sdk.NewAttribute("status", string(audit.Status)),
				sdk.NewAttribute("accuracy_delta", fmt.Sprintf("%d", audit.AccuracyDelta)),
			),
		)
	}

	return nil
}

// IsAccurate checks if a score is within the accuracy threshold of the ground truth.
// All values are in the 0-1000000 range.
func IsAccurate(score, groundTruth, threshold uint64) bool {
	var delta uint64
	if score > groundTruth {
		delta = score - groundTruth
	} else {
		delta = groundTruth - score
	}
	return delta <= threshold
}
