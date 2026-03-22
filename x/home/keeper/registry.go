package keeper

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/home/types"
)

// --- Capability CRUD ---

// SetCapability stores or overwrites a capability claim for a home.
func (k Keeper) SetCapability(ctx sdk.Context, homeID string, capType, level uint32) error {
	if !types.ValidCapabilityType(capType) {
		return types.ErrInvalidCapability.Wrapf("invalid capability type %d", capType)
	}
	if !types.ValidLevel(level) {
		return types.ErrInvalidCapability.Wrapf("invalid proficiency level %d", level)
	}
	if _, found := k.GetHome(ctx, homeID); !found {
		return types.ErrHomeNotFound.Wrapf("home %s not found", homeID)
	}

	entry := types.CapabilityEntry{
		HomeId:  homeID,
		CapType: capType,
		Level:   level,
	}

	bz, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal capability entry: %w", err)
	}

	store := k.storeService.OpenKVStore(ctx)
	return store.Set(types.CapabilityKey(homeID, capType), bz)
}

// GetCapability returns the proficiency level for a capability, and whether it exists.
func (k Keeper) GetCapability(ctx sdk.Context, homeID string, capType uint32) (uint32, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.CapabilityKey(homeID, capType))
	if err != nil || bz == nil {
		return 0, false
	}
	var entry types.CapabilityEntry
	if err := json.Unmarshal(bz, &entry); err != nil {
		return 0, false
	}
	return entry.Level, true
}

// ListCapabilities returns all capability entries for a home.
func (k Keeper) ListCapabilities(ctx sdk.Context, homeID string) []types.CapabilityEntry {
	store := k.storeService.OpenKVStore(ctx)
	prefix := types.CapabilityPrefixKey(homeID)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var entries []types.CapabilityEntry
	for ; iter.Valid(); iter.Next() {
		var entry types.CapabilityEntry
		if err := json.Unmarshal(iter.Value(), &entry); err != nil {
			continue
		}
		entries = append(entries, entry)
	}
	return entries
}

// --- Vouch CRUD ---

// VouchFor creates a vouch from voucherHomeID endorsing targetHomeID's capability.
// Returns the vouch ID.
func (k Keeper) VouchFor(ctx sdk.Context, voucherHomeID, targetHomeID string, capType, level uint32, blockHeight int64) (string, error) {
	if voucherHomeID == targetHomeID {
		return "", types.ErrSelfVouch.Wrap("voucher and target must be different homes")
	}
	if !types.ValidCapabilityType(capType) {
		return "", types.ErrInvalidCapability.Wrapf("invalid capability type %d", capType)
	}
	if !types.ValidLevel(level) {
		return "", types.ErrInvalidCapability.Wrapf("invalid proficiency level %d", level)
	}

	// Both homes must exist
	if _, found := k.GetHome(ctx, voucherHomeID); !found {
		return "", types.ErrHomeNotFound.Wrapf("voucher home %s not found", voucherHomeID)
	}
	if _, found := k.GetHome(ctx, targetHomeID); !found {
		return "", types.ErrHomeNotFound.Wrapf("target home %s not found", targetHomeID)
	}

	// Check max vouches per capability
	existing := k.GetVouches(ctx, targetHomeID, capType)
	if len(existing) >= types.MaxVouchesPerCap {
		return "", types.ErrMaxVouchesReached.Wrapf("target %s already has %d vouches for cap %d", targetHomeID, len(existing), capType)
	}

	vouchID := fmt.Sprintf("vouch-%s-%s-%d", voucherHomeID, targetHomeID, capType)

	vouch := types.Vouch{
		VouchID:       vouchID,
		VoucherHomeId: voucherHomeID,
		TargetHomeId:  targetHomeID,
		CapType:       capType,
		Level:         level,
		BlockHeight:   blockHeight,
	}

	bz, err := json.Marshal(vouch)
	if err != nil {
		return "", fmt.Errorf("failed to marshal vouch: %w", err)
	}

	store := k.storeService.OpenKVStore(ctx)
	if err := store.Set(types.VouchKey(targetHomeID, capType, voucherHomeID), bz); err != nil {
		return "", fmt.Errorf("failed to store vouch: %w", err)
	}

	return vouchID, nil
}

// GetVouches returns all vouches for a target home's capability.
func (k Keeper) GetVouches(ctx sdk.Context, targetHomeID string, capType uint32) []types.Vouch {
	store := k.storeService.OpenKVStore(ctx)
	prefix := types.VouchCapPrefixKey(targetHomeID, capType)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var vouches []types.Vouch
	for ; iter.Valid(); iter.Next() {
		var v types.Vouch
		if err := json.Unmarshal(iter.Value(), &v); err != nil {
			continue
		}
		vouches = append(vouches, v)
	}
	return vouches
}

// --- Reputation ---

// ReputationScore computes a simple reputation score for a home.
// Formula: sum of (level × vouchCount) across all capabilities, capped at 1000.
func (k Keeper) ReputationScore(ctx sdk.Context, homeID string) uint64 {
	caps := k.ListCapabilities(ctx, homeID)
	if len(caps) == 0 {
		return 0
	}

	var total uint64
	for _, cap := range caps {
		vouches := k.GetVouches(ctx, homeID, cap.CapType)
		total += uint64(cap.Level) * uint64(len(vouches))
	}

	if total > 1000 {
		total = 1000
	}
	return total
}
