package keeper

import (
	"fmt"
	"math/big"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/capture_challenge/types"
)

// Keeper manages the capture challenge module's state.
type Keeper struct {
	types.UnimplementedMsgServer

	cdc          codec.Codec
	storeService store.KVStoreService

	captureDefenseKeeper types.CaptureDefenseKeeper
	stakingKeeper        types.StakingKeeper
	bankKeeper           types.BankKeeper

	authority string
}

// NewKeeper creates a new capture challenge Keeper.
func NewKeeper(
	cdc codec.Codec,
	storeService store.KVStoreService,
	cdk types.CaptureDefenseKeeper,
	sk types.StakingKeeper,
	bk types.BankKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:                  cdc,
		storeService:         storeService,
		captureDefenseKeeper: cdk,
		stakingKeeper:        sk,
		bankKeeper:           bk,
		authority:            authority,
	}
}

// prefixEndBytes returns the end key for a prefix range scan.
// It increments the last byte of the prefix; if overflow occurs on all bytes, returns nil (no upper bound).
func prefixEndBytes(prefix []byte) []byte {
	if len(prefix) == 0 {
		return nil
	}
	end := make([]byte, len(prefix))
	copy(end, prefix)
	for i := len(end) - 1; i >= 0; i-- {
		end[i]++
		if end[i] != 0 {
			return end
		}
	}
	return nil
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// GetAuthority returns the module authority address.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// InitGenesis initializes the module from genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) {
	if genState.Params != nil {
		k.SetParams(ctx, *genState.Params)
	}
	for _, ch := range genState.Challenges {
		if ch != nil {
			k.SetChallenge(ctx, *ch)
		}
	}
	for _, pool := range genState.BountyPools {
		if pool != nil {
			k.SetBountyPool(ctx, *pool)
		}
	}
}

// ExportGenesis exports the module's genesis state.
func (k Keeper) ExportGenesis(ctx sdk.Context) types.GenesisState {
	var challenges []*types.CaptureChallenge
	k.IterateChallenges(ctx, func(ch types.CaptureChallenge) bool {
		c := ch // copy for pointer
		challenges = append(challenges, &c)
		return false
	})

	var pools []*types.DomainBountyPool
	k.IterateBountyPools(ctx, func(p types.DomainBountyPool) bool {
		pool := p // copy for pointer
		pools = append(pools, &pool)
		return false
	})

	params := k.GetParams(ctx)
	return types.GenesisState{
		Params:      &params,
		Challenges:  challenges,
		BountyPools: pools,
	}
}

// AdvanceChallengePhases checks active challenges and advances their phases.
func (k Keeper) AdvanceChallengePhases(ctx sdk.Context, currentBlock uint64) {
	k.IterateChallenges(ctx, func(ch types.CaptureChallenge) bool {
		changed := false

		switch types.ChallengeStatus(ch.Status) {
		case types.ChallengeSubmitted:
			// Move to evidence period
			ch.Status = string(types.ChallengeEvidencePeriod)
			changed = true

		case types.ChallengeEvidencePeriod:
			if currentBlock > ch.EvidenceDeadline {
				ch.Status = string(types.ChallengeUnderReview)
				changed = true
			}

		case types.ChallengeUnderReview:
			if currentBlock > ch.ReviewDeadline {
				// Auto-expire: governance failed to resolve within the review period.
				// Return full stake to challenger — they should not be penalized for
				// governance inaction.
				ch.Status = string(types.ChallengeExpired)
				ch.Resolution = &types.CaptureResolution{
					Outcome:          "expired",
					ChallengerReward: ch.ChallengeStake, // Full stake returned
					ResolvedAtBlock:  currentBlock,
					ResolvedBy:       "auto_expiry",
					Justification:    "Review deadline exceeded without governance resolution",
				}
				changed = true

				// Return locked stake to challenger
				if challengerAddr, err := sdk.AccAddressFromBech32(ch.Challenger); err == nil {
					stakeInt := new(big.Int)
					if _, ok := stakeInt.SetString(ch.ChallengeStake, 10); ok && stakeInt.Sign() > 0 {
						returnCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(stakeInt)))
						if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, challengerAddr, returnCoins); err != nil {
							k.Logger(ctx).Error("failed to return expired challenge stake",
								"challenge_id", ch.Id,
								"error", err,
							)
						}
					}
				}

				// Update bounty pool stats
				pool := k.GetOrCreateBountyPool(ctx, ch.Domain, currentBlock)
				pool.LastUpdatedBlock = currentBlock
				k.SetBountyPool(ctx, pool)

				ctx.EventManager().EmitEvent(
					sdk.NewEvent(
						"capture_challenge_expired",
						sdk.NewAttribute("challenge_id", ch.Id),
						sdk.NewAttribute("domain", ch.Domain),
						sdk.NewAttribute("challenger", ch.Challenger),
						sdk.NewAttribute("stake_returned", ch.ChallengeStake),
					),
				)
			}
		}

		if changed {
			k.SetChallenge(ctx, ch)
		}
		return false
	})
}

// AutoFundBountyPools funds bounty pools for domains with high capture risk.
func (k Keeper) AutoFundBountyPools(ctx sdk.Context, currentBlock uint64) {
	if k.captureDefenseKeeper == nil {
		return
	}

	// Iterate existing pools and check risk scores
	k.IterateBountyPools(ctx, func(pool types.DomainBountyPool) bool {
		metrics, found := k.captureDefenseKeeper.GetCaptureMetrics(ctx, pool.Domain)
		if !found {
			return false
		}

		// Auto-fund if risk > 0.4 (400,000 basis points)
		if metrics.RiskScore >= 400_000 {
			pool.CaptureRiskScore = metrics.RiskScore
			pool.AutoGenerated = true
			pool.LastAnalyzedBlock = currentBlock
			pool.LastUpdatedBlock = currentBlock
			k.SetBountyPool(ctx, pool)

			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					"bounty_pool_auto_funded",
					sdk.NewAttribute("domain", pool.Domain),
					sdk.NewAttribute("risk_score", fmt.Sprintf("%d", metrics.RiskScore)),
				),
			)
		}
		return false
	})
}
