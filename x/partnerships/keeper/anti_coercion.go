package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/partnerships/types"
)

// HandleSafetyFreeze applies a unilateral safety freeze to a partnership.
//
// Rules:
// - 222 blocks duration (SafetyFreezeDurationBlocks)
// - Max 3 per epoch (MaxFreezesPerEpoch)
// - Self-punishing if abused (cooperation score penalty)
func (k Keeper) HandleSafetyFreeze(ctx sdk.Context, partnershipId string, freezer string) (*types.SafetyFreeze, error) {
	params := k.GetParams(ctx)
	currentBlock := uint64(ctx.BlockHeight())

	// Check if there is already an active freeze
	if existing, found := k.GetSafetyFreeze(ctx, partnershipId); found {
		if existing.ExpiresAt > currentBlock {
			return nil, fmt.Errorf("%w: existing freeze expires at block %d",
				types.ErrFreezeActive, existing.ExpiresAt)
		}
	}

	// Load the partnership
	partnership, found := k.GetPartnership(ctx, partnershipId)
	if !found {
		return nil, types.ErrPartnershipNotFound
	}

	// Check freeze count this epoch
	var freezeCount uint32
	if existing, found := k.GetSafetyFreeze(ctx, partnershipId); found {
		freezeCount = existing.FreezeCountThisEpoch
	}

	if freezeCount >= params.MaxFreezesPerEpoch {
		return nil, fmt.Errorf("%w: %d of %d used",
			types.ErrMaxFreezesReached, freezeCount, params.MaxFreezesPerEpoch)
	}

	// Create the freeze
	sf := &types.SafetyFreeze{
		PartnershipId:        partnershipId,
		FrozenBy:             freezer,
		FrozenAt:             currentBlock,
		ExpiresAt:            currentBlock + params.SafetyFreezeDurationBlocks,
		FreezeCountThisEpoch: freezeCount + 1,
	}
	k.SetSafetyFreeze(ctx, sf)

	// Suspend the partnership
	partnership.Status = types.StatusSuspended
	k.SetPartnership(ctx, partnership)

	// Sync linked home to guarded status
	if k.homeKeeper != nil {
		if homeId, found := k.homeKeeper.GetHomeByPartnership(ctx, partnershipId); found {
			k.homeKeeper.SetHomeStatus(ctx, homeId, "guarded")
		}
	}

	// Self-punishing: reduce cooperation score by 100 bps per freeze
	// (abuse prevention: freezing too often hurts the freezer's partnership)
	penalty := uint64(100) * uint64(sf.FreezeCountThisEpoch)
	if partnership.CooperationScore > penalty {
		partnership.CooperationScore -= penalty
	} else {
		partnership.CooperationScore = 0
	}
	k.SetPartnership(ctx, partnership)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"safety_freeze_applied",
			sdk.NewAttribute("partnership_id", partnershipId),
			sdk.NewAttribute("frozen_by", freezer),
			sdk.NewAttribute("expires_at", fmt.Sprintf("%d", sf.ExpiresAt)),
			sdk.NewAttribute("freeze_count", fmt.Sprintf("%d", sf.FreezeCountThisEpoch)),
		),
	)

	k.Logger(ctx).Info("safety freeze applied",
		"partnership_id", partnershipId,
		"frozen_by", freezer,
		"expires_at", sf.ExpiresAt,
	)

	return sf, nil
}

// HandleCoercionSignal raises a coercion/duress flag on a partnership.
//
// Rules:
// - 1111 block review period (CoercionReviewBlocks)
// - Partnership is suspended during review
// - If no resolution, partnership may be dissolved
func (k Keeper) HandleCoercionSignal(ctx sdk.Context, partnershipId string, raiser string) (*types.CoercionSignal, error) {
	params := k.GetParams(ctx)
	currentBlock := uint64(ctx.BlockHeight())

	// Check for existing active signal
	signals := k.GetAllCoercionSignals(ctx)
	for _, s := range signals {
		if s.PartnershipId == partnershipId && !s.Resolved && s.ExpiresAt > currentBlock {
			return nil, fmt.Errorf("%w: existing signal %s expires at block %d",
				types.ErrCoercionActive, s.SignalId, s.ExpiresAt)
		}
	}

	// Load the partnership
	partnership, found := k.GetPartnership(ctx, partnershipId)
	if !found {
		return nil, types.ErrPartnershipNotFound
	}

	// Create the signal
	seq := k.NextSequence(ctx)
	signalId := fmt.Sprintf("coercion-%d", seq)

	cs := &types.CoercionSignal{
		SignalId:      signalId,
		PartnershipId: partnershipId,
		RaisedBy:      raiser,
		RaisedAt:      currentBlock,
		ExpiresAt:     currentBlock + params.CoercionReviewBlocks,
		Resolved:      false,
	}
	k.SetCoercionSignal(ctx, cs)

	// Suspend the partnership during review
	partnership.Status = types.StatusSuspended
	k.SetPartnership(ctx, partnership)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"coercion_signal_raised",
			sdk.NewAttribute("signal_id", signalId),
			sdk.NewAttribute("partnership_id", partnershipId),
			sdk.NewAttribute("raised_by", raiser),
			sdk.NewAttribute("expires_at", fmt.Sprintf("%d", cs.ExpiresAt)),
		),
	)

	k.Logger(ctx).Info("coercion signal raised",
		"signal_id", signalId,
		"partnership_id", partnershipId,
		"raised_by", raiser,
	)

	return cs, nil
}

// LiftExpiredFreezes clears freezes that have expired and restores partnership status.
//
// Fix: Instead of deleting the freeze record (which loses FreezeCountThisEpoch),
// we zero out ExpiresAt to mark it as lifted while preserving the epoch counter.
// Also checks for active coercion signals before restoring partnership to active.
func (k Keeper) LiftExpiredFreezes(ctx sdk.Context) {
	currentBlock := uint64(ctx.BlockHeight())
	freezes := k.GetAllSafetyFreezes(ctx)

	for _, sf := range freezes {
		// Only process active (non-lifted) freezes that have expired.
		if sf.ExpiresAt > 0 && sf.ExpiresAt <= currentBlock {
			// Restore partnership to active ONLY if no other active suspension cause.
			if partnership, found := k.GetPartnership(ctx, sf.PartnershipId); found {
				if partnership.Status == types.StatusSuspended {
					// Check for active coercion signals before restoring.
					if !k.hasActiveCoercionSignal(ctx, sf.PartnershipId, currentBlock) {
						partnership.Status = types.StatusActive
						k.SetPartnership(ctx, partnership)

						// Restore linked home to active status
						if k.homeKeeper != nil {
							if homeId, found := k.homeKeeper.GetHomeByPartnership(ctx, sf.PartnershipId); found {
								k.homeKeeper.SetHomeStatus(ctx, homeId, "active")
							}
						}
					}
				}
			}
			// Preserve the record (keeps FreezeCountThisEpoch) but mark as lifted.
			sf.ExpiresAt = 0
			sf.FrozenAt = 0
			k.SetSafetyFreeze(ctx, sf)
		}
	}
}

// hasActiveCoercionSignal returns true if any unresolved, non-expired coercion
// signal exists for the given partnership.
func (k Keeper) hasActiveCoercionSignal(ctx sdk.Context, partnershipId string, currentBlock uint64) bool {
	signals := k.GetAllCoercionSignals(ctx)
	for _, s := range signals {
		if s.PartnershipId == partnershipId && !s.Resolved && s.ExpiresAt > currentBlock {
			return true
		}
	}
	return false
}

// ExpireCoercionSignals marks expired coercion signals as resolved.
func (k Keeper) ExpireCoercionSignals(ctx sdk.Context) {
	currentBlock := uint64(ctx.BlockHeight())
	signals := k.GetAllCoercionSignals(ctx)

	for _, cs := range signals {
		if !cs.Resolved && cs.ExpiresAt <= currentBlock {
			cs.Resolved = true
			k.SetCoercionSignal(ctx, cs)

			// If no other active freeze/signal, restore partnership
			if partnership, found := k.GetPartnership(ctx, cs.PartnershipId); found {
				if partnership.Status == types.StatusSuspended {
					// Only restore if no active freeze
					if _, hasFreezeStill := k.GetSafetyFreeze(ctx, cs.PartnershipId); !hasFreezeStill {
						partnership.Status = types.StatusActive
						k.SetPartnership(ctx, partnership)
					}
				}
			}
		}
	}
}
