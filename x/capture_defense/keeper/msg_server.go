package keeper

import (
	"fmt"
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/capture_defense/types"
)

var _ types.MsgServer = Keeper{}

// RequestQualification handles a qualification request for a domain.
func (k Keeper) RequestQualification(goCtx context.Context, msg *types.MsgRequestQualification) (*types.MsgRequestQualificationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	qual, err := k.ProcessQualificationRequest(
		ctx,
		msg.ValidatorAddr,
		msg.Domain,
		types.QualificationPathway(msg.Pathway),
		msg.Stake,
		msg.Endorsers,
		msg.InheritFrom,
	)
	if err != nil {
		return &types.MsgRequestQualificationResponse{
			Granted: false,
			Error:   err.Error(),
		}, nil
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"qualification_granted",
			sdk.NewAttribute("validator", msg.ValidatorAddr),
			sdk.NewAttribute("domain", msg.Domain),
			sdk.NewAttribute("pathway", msg.Pathway),
			sdk.NewAttribute("status", string(qual.Status)),
		),
	)

	return &types.MsgRequestQualificationResponse{Granted: true}, nil
}

// EndorseQualification handles an endorsement of another validator's qualification.
func (k Keeper) EndorseQualification(goCtx context.Context, msg *types.MsgEndorseQualification) (*types.MsgEndorseQualificationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Verify endorser is qualified for the domain
	currentBlock := uint64(ctx.BlockHeight())
	if !k.IsQualifiedForDomain(ctx, msg.Endorser, msg.Domain, currentBlock) {
		return nil, types.ErrEndorserNotQualified
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"qualification_endorsed",
			sdk.NewAttribute("endorser", msg.Endorser),
			sdk.NewAttribute("target", msg.TargetValidator),
			sdk.NewAttribute("domain", msg.Domain),
		),
	)

	return &types.MsgEndorseQualificationResponse{Success: true}, nil
}


// UpdateParams handles MsgUpdateParams -- governance-gated parameter update.
func (k Keeper) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", k.GetAuthority(), msg.Authority)
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	k.SetParams(ctx, msg.Params)
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"capture_defense_params_updated",
			sdk.NewAttribute("authority", msg.Authority),
		),
	)
	return &types.MsgUpdateParamsResponse{}, nil
}
