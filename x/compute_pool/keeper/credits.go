package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/compute_pool/types"
)

// TierMultiplierBps returns the credit multiplier (bps) for a validator tier.
//
//	Apprentice: 500   (0.5x)
//	Verified:   800   (0.8x)
//	Bonded:     1000  (1.0x)
//	Guardian:   1500  (1.5x)
func TierMultiplierBps(tier string) uint64 {
	switch tier {
	case "apprentice":
		return 500
	case "verified":
		return 800
	case "bonded":
		return 1000
	case "guardian":
		return 1500
	default:
		return 500 // default to apprentice
	}
}

// CalculateCredits computes the CU to grant for a verification.
//
//	credits = base_cu × (1 + 0.25 × evidence + 0.5 × crossDomain) × tier_multiplier
//
// All arithmetic is integer; multipliers are in bps (1000 = 1.0x).
func CalculateCredits(params types.Params, tier string, evidenceCount, crossDomainRefs uint64) uint64 {
	base := params.BaseCuPerVerification

	// Complexity factor in bps: 1000000 + complexityBonusBps × evidence + crossDomainBonusBps × crossDomain
	complexityBps := uint64(1000000)
	complexityBps += params.ComplexityBonusBps * evidenceCount
	complexityBps += params.CrossDomainBonusBps * crossDomainRefs

	// Apply complexity: base × complexityBps / 1000000
	credits := base * complexityBps / 1000000

	// Apply tier multiplier: credits × tierBps / 1000
	tierBps := TierMultiplierBps(tier)
	credits = credits * tierBps / 1000

	if credits == 0 {
		credits = 1 // minimum 1 CU
	}
	return credits
}

// GrantCredits allocates compute credits to a validator for completing a verification.
func (k Keeper) GrantCredits(ctx sdk.Context, validatorAddr string, claimId string, evidenceCount, crossDomainRefs uint64) (uint64, error) {
	params := k.GetParams(ctx)
	height := uint64(ctx.BlockHeight())
	epoch := height / types.BlocksPerEpoch

	// Look up validator tier
	tier := "apprentice"
	if k.stakingKeeper != nil {
		t, _, found := k.stakingKeeper.GetValidatorByAddress(ctx, validatorAddr)
		if found {
			tier = t
		}
	}

	// Calculate base credits, then apply utilization adjustment factor
	cuAmount := CalculateCredits(params, tier, evidenceCount, crossDomainRefs)
	adjustmentFactor := k.GetAdjustmentFactor(ctx)
	cuAmount = cuAmount * adjustmentFactor / 1000000
	if cuAmount == 0 {
		cuAmount = 1 // minimum 1 CU even after adjustment
	}

	// Get or create credit record
	credit, _ := k.GetCredit(ctx, validatorAddr)
	if credit.ValidatorAddr == "" {
		credit.ValidatorAddr = validatorAddr
	}

	// Check if existing credits have expired
	if credit.Balance > 0 && height > credit.ExpiresAt {
		// Expire old credits
		credit.ExpiredTotal += credit.Balance
		k.recordExpiredCredits(ctx, epoch, credit.Balance)
		credit.Balance = 0
	}

	// Check max balance cap
	maxBalance := params.BaseCuPerVerification * params.MaxCreditBalanceMultiplier * TierMultiplierBps(tier) / 1000
	if credit.Balance+cuAmount > maxBalance {
		// Cap at maximum
		cuAmount = maxBalance - credit.Balance
		if cuAmount == 0 {
			return 0, nil // already at cap
		}
	}

	// Grant credits
	credit.Balance += cuAmount
	credit.EarnedTotal += cuAmount
	credit.LastEarnedAt = height
	credit.ExpiresAt = height + params.CreditExpiryBlocks

	k.SetCredit(ctx, credit)

	// Update epoch stats
	stats := k.GetOrCreateEpochStats(ctx, epoch)
	stats.CreditsIssued += cuAmount
	k.SetEpochStats(ctx, stats)

	k.Logger(ctx).Debug("granted compute credits",
		"validator", validatorAddr,
		"amount", cuAmount,
		"balance", credit.Balance,
		"claim", claimId,
	)

	return cuAmount, nil
}

// ExpireCredits expires all credits past their ExpiresAt block.
// Called in BeginBlock.
func (k Keeper) ExpireCredits(ctx sdk.Context) {
	height := uint64(ctx.BlockHeight())
	epoch := height / types.BlocksPerEpoch

	var toExpire []types.ComputeCredit

	k.IterateCredits(ctx, func(credit types.ComputeCredit) bool {
		if credit.Balance > 0 && height > credit.ExpiresAt {
			toExpire = append(toExpire, credit)
		}
		return false
	})

	for _, credit := range toExpire {
		expired := credit.Balance
		credit.ExpiredTotal += expired
		credit.Balance = 0
		k.SetCredit(ctx, credit)
		k.recordExpiredCredits(ctx, epoch, expired)

		k.Logger(ctx).Debug("expired compute credits",
			"validator", credit.ValidatorAddr,
			"amount", expired,
		)
	}
}

// DeductCredits removes credits from a validator's balance.
func (k Keeper) DeductCredits(ctx sdk.Context, validatorAddr string, amount uint64) error {
	credit, found := k.GetCredit(ctx, validatorAddr)
	if !found {
		return types.ErrInsufficientCredits
	}

	height := uint64(ctx.BlockHeight())

	// Check expiration
	if credit.Balance > 0 && height > credit.ExpiresAt {
		credit.ExpiredTotal += credit.Balance
		credit.Balance = 0
		k.SetCredit(ctx, credit)
		return types.ErrCreditExpired
	}

	if credit.Balance < amount {
		return types.ErrInsufficientCredits
	}

	credit.Balance -= amount
	credit.RedeemedTotal += amount
	k.SetCredit(ctx, credit)

	return nil
}

// recordExpiredCredits updates the epoch stats with expired credit count.
func (k Keeper) recordExpiredCredits(ctx sdk.Context, epoch uint64, amount uint64) {
	stats := k.GetOrCreateEpochStats(ctx, epoch)
	stats.CreditsExpired += amount
	k.SetEpochStats(ctx, stats)
}
