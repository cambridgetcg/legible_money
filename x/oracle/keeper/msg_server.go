package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/oracle/types"
)

// msgServer implements the oracle MsgServer interface.
type msgServer struct {
	types.UnimplementedMsgServer
	keeper Keeper
}

// NewMsgServerImpl returns an implementation of the oracle MsgServer interface.
func NewMsgServerImpl(k Keeper) types.MsgServer {
	return &msgServer{keeper: k}
}

var _ types.MsgServer = &msgServer{}

// SubmitPrediction handles MsgSubmitPrediction.
func (ms *msgServer) SubmitPrediction(goCtx context.Context, msg *types.MsgSubmitPrediction) (*types.MsgSubmitPredictionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	p, err := ms.keeper.SubmitPrediction(ctx, *msg)
	if err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"oracle_prediction_submitted",
		sdk.NewAttribute("prediction_id", p.PredictionId),
		sdk.NewAttribute("submitter", msg.Submitter),
		sdk.NewAttribute("domain", msg.Domain),
	))

	return &types.MsgSubmitPredictionResponse{
		PredictionId: p.PredictionId,
	}, nil
}

// ResolvePrediction handles MsgResolvePrediction.
func (ms *msgServer) ResolvePrediction(goCtx context.Context, msg *types.MsgResolvePrediction) (*types.MsgResolvePredictionResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := ms.keeper.ResolvePrediction(ctx, msg.PredictionId, msg.Outcome)
	if err != nil {
		return nil, err
	}

	p, _ := ms.keeper.GetPrediction(ctx, msg.PredictionId)

	return &types.MsgResolvePredictionResponse{
		BrierScore: p.BrierScore,
	}, nil
}
