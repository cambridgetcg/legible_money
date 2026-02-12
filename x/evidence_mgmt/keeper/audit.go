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
	if evidence.Status != string(types.EvidenceStatusSubmitted) {
		return fmt.Errorf("%w: evidence %s is %s, not submitted",
			types.ErrInvalidEvidence, audit.EvidenceId, evidence.Status)
	}

	// Prevent self-auditing: submitter cannot audit their own evidence
	if audit.Auditor == evidence.Submitter {
		return fmt.Errorf("%w: auditor %s is the evidence submitter",
			types.ErrSelfAudit, audit.Auditor)
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
	audit.Status = string(types.AuditStatusPending)
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
		if audit.Status == string(types.AuditStatusExpired) {
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
		if audit.Status == string(types.AuditStatusExpired) {
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
//
// Economics: Rewards are funded from slashed stakes within the same resolution batch.
// The module account holds escrowed stakes. Inaccurate auditors lose a portion (slash),
// which funds rewards for accurate auditors. If total rewards exceed total slashed,
// rewards are proportionally scaled down to available funds. Any remaining slashed
// funds after paying rewards are burned.
func (k Keeper) ResolveAudits(ctx sdk.Context, evidenceId string, groundTruth uint64) error {
	params := k.GetParams(ctx)
	audits := k.GetAuditsForEvidence(ctx, evidenceId)

	// Two-pass resolution:
	// Pass 1: Classify audits, compute slashes and theoretical rewards
	// Pass 2: Scale rewards to available funds, execute transfers

	type auditResolution struct {
		audit       types.EvidenceAudit
		accurate    bool
		stakeAmount *big.Int
		delta       uint64
		auditorAddr sdk.AccAddress
		// For accurate: theoretical reward (before scaling)
		reward *big.Int
		// For inaccurate: slash amount
		slashAmount *big.Int
	}

	var resolutions []auditResolution
	totalSlashed := new(big.Int)
	totalRewards := new(big.Int)

	// Pass 1: Classify and compute
	for _, audit := range audits {
		if audit.Status != string(types.AuditStatusPending) {
			continue
		}

		stakeAmount := new(big.Int)
		if _, ok := stakeAmount.SetString(audit.Stake, 10); !ok {
			continue
		}

		auditorAddr, err := sdk.AccAddressFromBech32(audit.Auditor)
		if err != nil {
			k.Logger(ctx).Error("invalid auditor address during resolution",
				"auditor", audit.Auditor, "error", err)
			continue
		}

		var delta uint64
		if audit.Score > groundTruth {
			delta = audit.Score - groundTruth
		} else {
			delta = groundTruth - audit.Score
		}

		accurate := IsAccurate(audit.Score, groundTruth, params.AccuracyThreshold)

		res := auditResolution{
			audit:       audit,
			accurate:    accurate,
			stakeAmount: stakeAmount,
			delta:       delta,
			auditorAddr: auditorAddr,
		}

		if accurate {
			// Theoretical reward: RewardRate basis points of stake
			rewardBps := new(big.Int).SetUint64(params.RewardRate)
			reward := new(big.Int).Mul(stakeAmount, rewardBps)
			reward.Div(reward, new(big.Int).SetUint64(1000000))
			res.reward = reward
			totalRewards.Add(totalRewards, reward)
		} else {
			// Slash: SlashRate basis points of stake
			slashBps := new(big.Int).SetUint64(params.SlashRate)
			slashAmt := new(big.Int).Mul(stakeAmount, slashBps)
			slashAmt.Div(slashAmt, new(big.Int).SetUint64(1000000))
			res.slashAmount = slashAmt
			totalSlashed.Add(totalSlashed, slashAmt)
		}

		resolutions = append(resolutions, res)
	}

	// Compute reward scaling factor: if total rewards > total slashed,
	// scale rewards down proportionally so they don't exceed available funds.
	// rewardScale = min(1, totalSlashed / totalRewards) as basis points (0-1000000)
	rewardScaleBps := uint64(1000000) // 100% by default
	if totalRewards.Sign() > 0 && totalSlashed.Cmp(totalRewards) < 0 {
		// Scale down: rewardScaleBps = totalSlashed * 1000000 / totalRewards
		scaled := new(big.Int).Mul(totalSlashed, new(big.Int).SetUint64(1000000))
		scaled.Div(scaled, totalRewards)
		rewardScaleBps = scaled.Uint64()
	}

	// Track actual rewards paid for burn calculation
	actualRewardsPaid := new(big.Int)

	// Pass 2: Execute transfers
	for _, res := range resolutions {
		res.audit.AccuracyDelta = res.delta

		if res.accurate {
			// Scale reward by available slash funds
			scaledReward := new(big.Int).Mul(res.reward, new(big.Int).SetUint64(rewardScaleBps))
			scaledReward.Div(scaledReward, new(big.Int).SetUint64(1000000))

			// Return stake + scaled reward to auditor
			totalReturn := new(big.Int).Add(res.stakeAmount, scaledReward)
			returnCoin := sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(totalReturn))
			if err := k.bankKeeper.SendCoinsFromModuleToAccount(
				ctx, types.ModuleName, res.auditorAddr, sdk.NewCoins(returnCoin),
			); err != nil {
				k.Logger(ctx).Error("failed to reward auditor",
					"auditor", res.audit.Auditor, "error", err)
				continue
			}

			actualRewardsPaid.Add(actualRewardsPaid, scaledReward)

			res.audit.Status = string(types.AuditStatusRewarded)
			res.audit.RewardAmount = scaledReward.String()
			res.audit.ResolvedAt = uint64(ctx.BlockHeight())
		} else {
			// Return remainder (stake - slash) to auditor
			remainder := new(big.Int).Sub(res.stakeAmount, res.slashAmount)
			if remainder.Sign() > 0 {
				returnCoin := sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(remainder))
				if err := k.bankKeeper.SendCoinsFromModuleToAccount(
					ctx, types.ModuleName, res.auditorAddr, sdk.NewCoins(returnCoin),
				); err != nil {
					k.Logger(ctx).Error("failed to return remainder to auditor",
						"auditor", res.audit.Auditor, "error", err)
					continue
				}
			}

			res.audit.Status = string(types.AuditStatusSlashed)
			res.audit.SlashAmount = res.slashAmount.String()
			res.audit.ResolvedAt = uint64(ctx.BlockHeight())
		}

		k.SetAudit(ctx, res.audit)

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				"audit_resolved",
				sdk.NewAttribute("audit_id", res.audit.Id),
				sdk.NewAttribute("evidence_id", res.audit.EvidenceId),
				sdk.NewAttribute("auditor", res.audit.Auditor),
				sdk.NewAttribute("status", string(res.audit.Status)),
				sdk.NewAttribute("accuracy_delta", fmt.Sprintf("%d", res.audit.AccuracyDelta)),
			),
		)
	}

	// Burn excess slashed funds that weren't used for rewards
	excessSlash := new(big.Int).Sub(totalSlashed, actualRewardsPaid)
	if excessSlash.Sign() > 0 {
		burnCoin := sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(excessSlash))
		if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(burnCoin)); err != nil {
			k.Logger(ctx).Error("failed to burn excess slashed funds",
				"amount", excessSlash.String(), "error", err)
		} else {
			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					"audit_slash_burned",
					sdk.NewAttribute("evidence_id", evidenceId),
					sdk.NewAttribute("burned_amount", excessSlash.String()),
				),
			)
		}
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
