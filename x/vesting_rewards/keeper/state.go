package keeper

import (
	"encoding/json"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/vesting_rewards/types"
)

// SetVestingSchedule stores a vesting schedule.
func (k Keeper) SetVestingSchedule(ctx sdk.Context, schedule types.VestingSchedule) {
	store := ctx.KVStore(k.storeKey)

	bz, err := json.Marshal(schedule)
	if err != nil {
		panic("failed to marshal vesting schedule: " + err.Error())
	}

	// Primary key: vestingId -> schedule
	key := append(types.VestingScheduleKeyPrefix, []byte(schedule.Id)...)
	store.Set(key, bz)

	// Index: claimId -> vestingId
	if schedule.ClaimId != "" {
		claimKey := append(types.ClaimRecordKeyPrefix, []byte(schedule.ClaimId)...)
		store.Set(claimKey, []byte(schedule.Id))
	}

	// Index: recipient/vestingId -> nil (for queries by recipient)
	recipientKey := append(types.VestingByRecipientPrefix, []byte(schedule.Recipient+"/"+schedule.Id)...)
	store.Set(recipientKey, []byte{1})

	// Index: active schedules
	if schedule.Status == types.VestingStatusActive || schedule.Status == types.VestingStatusPaused {
		activeKey := append(types.ActiveVestingPrefix, []byte(schedule.Id)...)
		store.Set(activeKey, []byte{1})
	} else {
		// Remove from active index if no longer active
		activeKey := append(types.ActiveVestingPrefix, []byte(schedule.Id)...)
		store.Delete(activeKey)
	}
}

// GetVestingSchedule retrieves a vesting schedule by ID.
func (k Keeper) GetVestingSchedule(ctx sdk.Context, vestingId string) (types.VestingSchedule, bool) {
	store := ctx.KVStore(k.storeKey)
	key := append(types.VestingScheduleKeyPrefix, []byte(vestingId)...)
	bz := store.Get(key)
	if bz == nil {
		return types.VestingSchedule{}, false
	}
	var schedule types.VestingSchedule
	if err := json.Unmarshal(bz, &schedule); err != nil {
		return types.VestingSchedule{}, false
	}
	return schedule, true
}

// GetVestingByClaimId retrieves the vesting schedule for a claim.
func (k Keeper) GetVestingByClaimId(ctx sdk.Context, claimId string) (types.VestingSchedule, bool) {
	store := ctx.KVStore(k.storeKey)
	claimKey := append(types.ClaimRecordKeyPrefix, []byte(claimId)...)
	vestingIdBz := store.Get(claimKey)
	if vestingIdBz == nil {
		return types.VestingSchedule{}, false
	}
	return k.GetVestingSchedule(ctx, string(vestingIdBz))
}

// GetVestingSchedulesByRecipient returns all vesting schedules for a recipient.
func (k Keeper) GetVestingSchedulesByRecipient(ctx sdk.Context, recipient string) []types.VestingSchedule {
	store := ctx.KVStore(k.storeKey)
	prefix := append(types.VestingByRecipientPrefix, []byte(recipient+"/")...)

	var schedules []types.VestingSchedule
	iter := storetypes.KVStorePrefixIterator(store, prefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		// Key format: prefix + recipient + "/" + vestingId
		keyStr := string(iter.Key())
		vestingId := keyStr[len(prefix):]
		schedule, found := k.GetVestingSchedule(ctx, vestingId)
		if found {
			schedules = append(schedules, schedule)
		}
	}
	return schedules
}

// GetAllActiveVestingSchedules returns all active/paused vesting schedules.
func (k Keeper) GetAllActiveVestingSchedules(ctx sdk.Context) []types.VestingSchedule {
	store := ctx.KVStore(k.storeKey)

	var schedules []types.VestingSchedule
	iter := storetypes.KVStorePrefixIterator(store, types.ActiveVestingPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		vestingId := string(iter.Key()[len(types.ActiveVestingPrefix):])
		schedule, found := k.GetVestingSchedule(ctx, vestingId)
		if found {
			schedules = append(schedules, schedule)
		}
	}
	return schedules
}

// SetClawbackRecord stores a clawback record.
func (k Keeper) SetClawbackRecord(ctx sdk.Context, record types.ClawbackRecord) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(record)
	if err != nil {
		panic("failed to marshal clawback record: " + err.Error())
	}
	key := append(types.FalsificationKeyPrefix, []byte(record.Id)...)
	store.Set(key, bz)
}

// GetClawbackRecord retrieves a clawback record by ID.
func (k Keeper) GetClawbackRecord(ctx sdk.Context, id string) (types.ClawbackRecord, bool) {
	store := ctx.KVStore(k.storeKey)
	key := append(types.FalsificationKeyPrefix, []byte(id)...)
	bz := store.Get(key)
	if bz == nil {
		return types.ClawbackRecord{}, false
	}
	var record types.ClawbackRecord
	if err := json.Unmarshal(bz, &record); err != nil {
		return types.ClawbackRecord{}, false
	}
	return record, true
}

// SetBlockRewardDistribution stores a block reward distribution record.
func (k Keeper) SetBlockRewardDistribution(ctx sdk.Context, dist types.BlockRewardDistribution) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(dist)
	if err != nil {
		return
	}
	// Use big-endian block height as key for ordering
	key := append(types.BlockRewardKeyPrefix, sdk.Uint64ToBigEndian(dist.BlockHeight)...)
	store.Set(key, bz)
}
