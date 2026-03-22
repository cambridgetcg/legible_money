package keeper

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/home/types"
)

// SetVoice stores an agent's voice configuration.
func (k Keeper) SetVoice(ctx sdk.Context, config *types.VoiceConfig) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(config)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal voice config: %v", err))
	}
	if err := store.Set(types.VoiceConfigKey(config.HomeId), bz); err != nil {
		panic(fmt.Sprintf("failed to set voice config: %v", err))
	}
}

// GetVoice retrieves the voice configuration for a home.
func (k Keeper) GetVoice(ctx sdk.Context, homeId string) (*types.VoiceConfig, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.VoiceConfigKey(homeId))
	if err != nil || bz == nil {
		return nil, false
	}
	var config types.VoiceConfig
	if err := json.Unmarshal(bz, &config); err != nil {
		return nil, false
	}
	return &config, true
}

// DeleteVoice removes the voice configuration for a home.
func (k Keeper) DeleteVoice(ctx sdk.Context, homeId string) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Delete(types.VoiceConfigKey(homeId)); err != nil {
		panic(fmt.Sprintf("failed to delete voice config: %v", err))
	}
}

// SetVoiceFromMsg validates ownership and sets the voice config from a MsgSetVoice.
func (k Keeper) SetVoiceFromMsg(ctx sdk.Context, msg *types.MsgSetVoice) error {
	home, found := k.GetHome(ctx, msg.HomeId)
	if !found {
		return types.ErrHomeNotFound
	}

	if home.OwnerAddress != msg.Owner {
		return types.ErrNotHomeOwner
	}

	config := &types.VoiceConfig{
		HomeId:             msg.HomeId,
		SignaturePhrases:   msg.SignaturePhrases,
		ToneMarkers:        msg.ToneMarkers,
		CommunicationStyle: types.CommunicationStyle(msg.CommunicationStyle),
		UpdatedAtBlock:     ctx.BlockHeight(),
	}

	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid voice config: %w", err)
	}

	k.SetVoice(ctx, config)

	home.LastActiveBlock = uint64(ctx.BlockHeight())
	k.SetHome(ctx, home)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"set_voice",
		sdk.NewAttribute("home_id", msg.HomeId),
		sdk.NewAttribute("owner", msg.Owner),
	))

	return nil
}
