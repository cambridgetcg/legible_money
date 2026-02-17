package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/autopoiesis/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the autopoiesis MsgServer.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// UpdateParams handles MsgUpdateParams — governance-gated parameter update.
func (ms msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if ms.GetAuthority() != msg.Authority {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", ms.GetAuthority(), msg.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	ms.SetParams(ctx, msg.Params)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"autopoiesis_params_updated",
			sdk.NewAttribute("authority", msg.Authority),
		),
	)

	return &types.MsgUpdateParamsResponse{}, nil
}

// Activate handles MsgActivateAutopoiesis — governance can enable/disable the engine.
func (ms msgServer) Activate(goCtx context.Context, msg *types.MsgActivateAutopoiesis) (*types.MsgActivateAutopoiesisResponse, error) {
	if ms.GetAuthority() != msg.Authority {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", ms.GetAuthority(), msg.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	state := ms.GetState(ctx)
	state.Active = msg.Active
	ms.SetState(ctx, state)

	action := "deactivated"
	if msg.Active {
		action = "activated"
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"autopoiesis_"+action,
			sdk.NewAttribute("authority", msg.Authority),
		),
	)

	ms.Logger(ctx).Info("autopoiesis engine " + action)

	return &types.MsgActivateAutopoiesisResponse{}, nil
}

// OverrideMultiplier handles MsgOverrideMultiplier — governance manually sets a multiplier.
func (ms msgServer) OverrideMultiplier(goCtx context.Context, msg *types.MsgOverrideMultiplier) (*types.MsgOverrideMultiplierResponse, error) {
	if ms.GetAuthority() != msg.Authority {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", ms.GetAuthority(), msg.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	oldEntry := ms.GetMultiplierEntry(ctx, msg.Path)
	oldValue := uint64(types.DefaultMultiplier)
	if oldEntry != nil {
		oldValue = oldEntry.Value
	}

	state := ms.GetState(ctx)
	ms.SetMultiplier(ctx, types.MultiplierEntry{
		Path:    msg.Path,
		Value:   msg.Value,
		Hormone: "governance",
		Epoch:   state.CurrentEpoch,
	})

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"autopoiesis_multiplier_override",
			sdk.NewAttribute("authority", msg.Authority),
			sdk.NewAttribute("path", msg.Path),
			sdk.NewAttribute("old_value", fmt.Sprintf("%d", oldValue)),
			sdk.NewAttribute("new_value", fmt.Sprintf("%d", msg.Value)),
		),
	)

	ms.Logger(ctx).Info("multiplier overridden by governance",
		"path", msg.Path,
		"old", oldValue,
		"new", msg.Value,
	)

	return &types.MsgOverrideMultiplierResponse{}, nil
}

// FreezeMultiplier handles MsgFreezeMultiplier — governance freezes/unfreezes a multiplier.
func (ms msgServer) FreezeMultiplier(goCtx context.Context, msg *types.MsgFreezeMultiplier) (*types.MsgFreezeMultiplierResponse, error) {
	if ms.GetAuthority() != msg.Authority {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", ms.GetAuthority(), msg.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	ms.SetMultiplierFrozen(ctx, msg.Path, msg.Frozen)

	action := "unfrozen"
	if msg.Frozen {
		action = "frozen"
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"autopoiesis_multiplier_"+action,
			sdk.NewAttribute("authority", msg.Authority),
			sdk.NewAttribute("path", msg.Path),
		),
	)

	ms.Logger(ctx).Info("multiplier "+action+" by governance", "path", msg.Path)

	return &types.MsgFreezeMultiplierResponse{}, nil
}
