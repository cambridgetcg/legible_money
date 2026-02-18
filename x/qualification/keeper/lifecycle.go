package keeper

import (
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/qualification/types"
)

// PromoteProbationary promotes a probationary qualification to active if the
// probationary period has elapsed.
func (k Keeper) PromoteProbationary(ctx sdk.Context, q *types.DomainQualification) bool {
	if q.Status != string(types.StatusProbationary) {
		return false
	}
	params := k.GetParams(ctx)
	currentBlock := uint64(ctx.BlockHeight())

	if currentBlock >= q.EarnedAtBlock+params.ProbationaryPeriod {
		q.Status = string(types.StatusActive)
		if q.Metrics == nil {
			q.Metrics = &types.QualificationMetrics{}
		}
		q.Metrics.EffectiveWeightBps = k.calculateWeight(q)
		k.SetQualification(ctx, q)
		return true
	}
	return false
}

// RenewQualification extends the expiration of an active qualification.
func (k Keeper) RenewQualification(ctx sdk.Context, validatorAddr, domain string) error {
	q, found := k.GetQualification(ctx, validatorAddr, domain)
	if !found {
		return types.ErrNotQualified
	}

	if q.Status == string(types.StatusRevoked) {
		return fmt.Errorf("%w: cannot renew revoked qualification", types.ErrQualificationSuspended)
	}
	if q.Status == string(types.StatusSuspended) {
		return fmt.Errorf("%w: must resolve suspension first", types.ErrQualificationSuspended)
	}

	params := k.GetParams(ctx)
	currentBlock := uint64(ctx.BlockHeight())

	// Extend expiration (if it has one)
	if q.ExpiresAtBlock > 0 {
		q.ExpiresAtBlock = currentBlock + params.InactivityExpiration
	}

	// If expired, reactivate as probationary
	if q.Status == string(types.StatusExpired) {
		q.Status = string(types.StatusProbationary)
		q.EarnedAtBlock = currentBlock
	}

	k.SetQualification(ctx, q)
	return nil
}

// WithdrawQualification voluntarily removes a qualification and unlocks stake.
func (k Keeper) WithdrawQualification(ctx sdk.Context, validatorAddr, domain string) error {
	q, found := k.GetQualification(ctx, validatorAddr, domain)
	if !found {
		return types.ErrNotQualified
	}

	// Unlock stake if applicable
	if q.LockedStake != "" && q.LockedStake != "0" && k.bankKeeper != nil {
		currentBlock := uint64(ctx.BlockHeight())
		if currentBlock >= q.StakeUnlocksAt {
			addr, err := sdk.AccAddressFromBech32(validatorAddr)
			if err == nil {
				amount, ok := new(big.Int).SetString(q.LockedStake, 10)
				if ok {
					coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(amount)))
					_ = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, coins)
				}
			}
		}
		// If lock period hasn't expired, stake is forfeit (slashed)
	}

	k.DeleteQualification(ctx, validatorAddr, domain)
	return nil
}

// RecordChallenge records a challenge loss against a qualification.
func (k Keeper) RecordChallenge(ctx sdk.Context, validatorAddr, domain string) error {
	q, found := k.GetQualification(ctx, validatorAddr, domain)
	if !found {
		return types.ErrNotQualified
	}

	if q.Metrics == nil {
		q.Metrics = &types.QualificationMetrics{}
	}
	q.Metrics.ChallengesReceived++
	q.Metrics.ChallengesLost++

	params := k.GetParams(ctx)
	if q.Metrics.ChallengesLost >= params.RevocationChallenges {
		q.Status = string(types.StatusRevoked)
	}

	k.SetQualification(ctx, q)
	return nil
}

// CheckAndExpireQualifications iterates qualifications and expires any that have
// passed their expiration block. Called from BeginBlocker.
func (k Keeper) CheckAndExpireQualifications(ctx sdk.Context) {
	currentBlock := uint64(ctx.BlockHeight())
	qualifications := k.GetAllQualifications(ctx)

	for _, q := range qualifications {
		changed := false

		// Expire if past expiration
		if q.ExpiresAtBlock > 0 && currentBlock > q.ExpiresAtBlock &&
			(q.Status == string(types.StatusActive) || q.Status == string(types.StatusProbationary)) {
			q.Status = string(types.StatusExpired)
			changed = true
		}

		// Promote probationary if period elapsed
		if q.Status == string(types.StatusProbationary) {
			params := k.GetParams(ctx)
			if currentBlock >= q.EarnedAtBlock+params.ProbationaryPeriod {
				q.Status = string(types.StatusActive)
				if q.Metrics == nil {
					q.Metrics = &types.QualificationMetrics{}
				}
				q.Metrics.EffectiveWeightBps = k.calculateWeight(q)
				changed = true
			}
		}

		if changed {
			k.SetQualification(ctx, q)
		}
	}
}
