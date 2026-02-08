package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/disputes/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// OpenDispute handles MsgOpenDispute — opens a new dispute against a verified fact.
func (k msgServer) OpenDispute(goCtx context.Context, msg *types.MsgOpenDispute) (*types.MsgOpenDisputeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Lock the challenger's bond in the module account
	bondAmount := new(big.Int)
	if _, ok := bondAmount.SetString(msg.Bond, 10); !ok || bondAmount.Sign() <= 0 {
		return nil, fmt.Errorf("%w: invalid bond amount", types.ErrInvalidBond)
	}

	challengerAddr, err := sdk.AccAddressFromBech32(msg.Challenger)
	if err != nil {
		return nil, fmt.Errorf("invalid challenger address: %w", err)
	}

	bondCoin := sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(bondAmount))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(
		ctx, challengerAddr, types.ModuleName, sdk.NewCoins(bondCoin),
	); err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrInsufficientBond, err)
	}

	// Open the dispute (validates fact, selects arbiters, etc.)
	dispute, err := k.Keeper.OpenDispute(ctx, msg.FactId, msg.Challenger, msg.Reason, msg.Bond)
	if err != nil {
		// Refund bond on failure
		_ = k.bankKeeper.SendCoinsFromModuleToAccount(
			ctx, types.ModuleName, challengerAddr, sdk.NewCoins(bondCoin),
		)
		return nil, err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"dispute_opened",
			sdk.NewAttribute("dispute_id", dispute.Id),
			sdk.NewAttribute("fact_id", msg.FactId),
			sdk.NewAttribute("challenger", msg.Challenger),
			sdk.NewAttribute("bond", msg.Bond),
			sdk.NewAttribute("tier", fmt.Sprintf("%d", dispute.Tier)),
			sdk.NewAttribute("height", fmt.Sprintf("%d", ctx.BlockHeight())),
		),
	)

	return &types.MsgOpenDisputeResponse{DisputeId: dispute.Id}, nil
}

// SubmitEvidence handles MsgSubmitEvidence — submits evidence for a dispute.
func (k msgServer) SubmitEvidence(goCtx context.Context, msg *types.MsgSubmitEvidence) (*types.MsgSubmitEvidenceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	dispute, found := k.GetDispute(ctx, msg.DisputeId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrDisputeNotFound, msg.DisputeId)
	}

	// Check dispute is in evidence phase
	if dispute.Status != types.DisputeStatusEvidence {
		return nil, fmt.Errorf("%w: dispute is in %s phase", types.ErrNotInEvidencePhase, dispute.Status)
	}

	// Check evidence deadline
	currentHeight := uint64(ctx.BlockHeight())
	if currentHeight > dispute.EvidenceDeadline {
		return nil, fmt.Errorf("%w: deadline was at height %d",
			types.ErrEvidenceDeadlinePassed, dispute.EvidenceDeadline)
	}

	// Validate submitter is challenger or defender
	if msg.Side == "challenger" && msg.Submitter != dispute.Challenger {
		return nil, fmt.Errorf("submitter %s is not the challenger %s", msg.Submitter, dispute.Challenger)
	}
	if msg.Side == "defender" && msg.Submitter != dispute.Defender {
		return nil, fmt.Errorf("submitter %s is not the defender %s", msg.Submitter, dispute.Defender)
	}

	// Generate evidence ID and store
	evidenceId := GenerateEvidenceID(msg.DisputeId, msg.Submitter, currentHeight)
	evidence := types.DisputeEvidence{
		Id:          evidenceId,
		DisputeId:   msg.DisputeId,
		Submitter:   msg.Submitter,
		Side:        msg.Side,
		Content:     msg.Content,
		SubmittedAt: currentHeight,
	}
	k.SetEvidence(ctx, evidence)

	// Update evidence count on dispute
	dispute.EvidenceCount++
	k.SetDispute(ctx, dispute)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"evidence_submitted",
			sdk.NewAttribute("dispute_id", msg.DisputeId),
			sdk.NewAttribute("evidence_id", evidenceId),
			sdk.NewAttribute("submitter", msg.Submitter),
			sdk.NewAttribute("side", msg.Side),
		),
	)

	return &types.MsgSubmitEvidenceResponse{EvidenceId: evidenceId}, nil
}

// VoteDispute handles MsgVoteDispute — an arbiter casts a vote.
func (k msgServer) VoteDispute(goCtx context.Context, msg *types.MsgVoteDispute) (*types.MsgVoteDisputeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := k.Keeper.CastVote(ctx, msg.DisputeId, msg.Arbiter, msg.Vote, msg.Rationale); err != nil {
		return nil, err
	}

	return &types.MsgVoteDisputeResponse{}, nil
}

// EscalateDispute handles MsgEscalateDispute — escalates to the next tier.
func (k msgServer) EscalateDispute(goCtx context.Context, msg *types.MsgEscalateDispute) (*types.MsgEscalateDisputeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := k.Keeper.EscalateDispute(ctx, msg.DisputeId, msg.Requester, msg.AdditionalBond); err != nil {
		return nil, err
	}

	return &types.MsgEscalateDisputeResponse{}, nil
}
