package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/ibcratelimit/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns a MsgServer implementation.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// UpdateParams updates the module parameters. Only the governance authority can call this.
func (m msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Verify authority
	if msg.Authority != m.Keeper.GetAuthority() {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", m.Keeper.GetAuthority(), msg.Authority)
	}

	// Validate params
	if err := msg.Params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	m.Keeper.SetParams(ctx, msg.Params)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"update_ibcratelimit_params",
		sdk.NewAttribute("epoch_length_blocks", fmt.Sprintf("%d", msg.Params.EpochLengthBlocks)),
		sdk.NewAttribute("max_outflow_per_epoch", msg.Params.MaxOutflowPerEpoch),
		sdk.NewAttribute("max_inflow_per_epoch", msg.Params.MaxInflowPerEpoch),
		sdk.NewAttribute("tracked_denom", msg.Params.TrackedDenom),
	))

	return &types.MsgUpdateParamsResponse{}, nil
}
