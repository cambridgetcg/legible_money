package keeper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

// CommitEvidence handles MsgCommitEvidence — commits a hash of evidence (phase 1 of commit-reveal).
func (k msgServer) CommitEvidence(goCtx context.Context, msg *types.MsgCommitEvidence) (*types.MsgCommitEvidenceResponse, error) {
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

	// Check for existing commitment
	if _, exists := k.GetEvidenceCommitment(ctx, msg.DisputeId, msg.Submitter); exists {
		return nil, fmt.Errorf("evidence already committed for dispute %s by %s", msg.DisputeId, msg.Submitter)
	}

	commitment := types.EvidenceCommitment{
		DisputeId:   msg.DisputeId,
		Submitter:   msg.Submitter,
		Side:        msg.Side,
		ContentHash: msg.ContentHash,
		CommittedAt: currentHeight,
		Revealed:    false,
	}
	k.SetEvidenceCommitment(ctx, commitment)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"evidence_committed",
			sdk.NewAttribute("dispute_id", msg.DisputeId),
			sdk.NewAttribute("submitter", msg.Submitter),
			sdk.NewAttribute("side", msg.Side),
		),
	)

	return &types.MsgCommitEvidenceResponse{}, nil
}

// SubmitEvidence handles MsgSubmitEvidence — reveals evidence for a dispute (phase 2 of commit-reveal).
// If a commitment exists, verifies the content hash matches. If no commitment exists,
// accepts the evidence directly (backward-compatible with non-commit-reveal flow).
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

	// Check if a commitment exists — if so, verify hash match (commit-reveal enforcement)
	if commitment, exists := k.GetEvidenceCommitment(ctx, msg.DisputeId, msg.Submitter); exists {
		if commitment.Revealed {
			return nil, fmt.Errorf("evidence already revealed for dispute %s by %s", msg.DisputeId, msg.Submitter)
		}
		// Verify content matches committed hash
		contentHash := sha256.Sum256([]byte(msg.Content))
		computedHex := hex.EncodeToString(contentHash[:])
		if computedHex != commitment.ContentHash {
			return nil, fmt.Errorf("%w: committed hash %s does not match content hash %s",
				types.ErrEvidenceHashMismatch, commitment.ContentHash, computedHex)
		}
		// Mark commitment as revealed
		commitment.Revealed = true
		k.SetEvidenceCommitment(ctx, commitment)
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
			"evidence_revealed",
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

// SettleDispute handles MsgSettleDispute — permissionless trigger for tally + resolution
// once the voting deadline has passed. Anyone can submit this to avoid waiting for EndBlock.
func (k msgServer) SettleDispute(goCtx context.Context, msg *types.MsgSettleDispute) (*types.MsgSettleDisputeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	dispute, found := k.GetDispute(ctx, msg.DisputeId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrDisputeNotFound, msg.DisputeId)
	}

	// Must be in voting phase
	if dispute.Status != types.DisputeStatusVoting {
		return nil, fmt.Errorf("%w: dispute is in %s phase", types.ErrNotInVotingPhase, dispute.Status)
	}

	// Voting deadline must have passed
	currentHeight := uint64(ctx.BlockHeight())
	if currentHeight <= dispute.VotingDeadline {
		return nil, fmt.Errorf("voting deadline has not passed: current %d, deadline %d",
			currentHeight, dispute.VotingDeadline)
	}

	// Tally votes
	outcome, err := k.Keeper.TallyVotes(ctx, dispute)
	if err != nil {
		return nil, fmt.Errorf("failed to tally votes: %w", err)
	}

	// Resolve
	if err := k.Keeper.ResolveDispute(ctx, dispute, outcome); err != nil {
		return nil, fmt.Errorf("failed to resolve dispute: %w", err)
	}

	return &types.MsgSettleDisputeResponse{Outcome: string(outcome)}, nil
}

// TimeoutDispute handles MsgTimeoutDispute — permissionless trigger for timeout processing.
// Transitions evidence→voting if evidence deadline passed, or expires open disputes.
func (k msgServer) TimeoutDispute(goCtx context.Context, msg *types.MsgTimeoutDispute) (*types.MsgTimeoutDisputeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	dispute, found := k.GetDispute(ctx, msg.DisputeId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrDisputeNotFound, msg.DisputeId)
	}

	currentHeight := uint64(ctx.BlockHeight())

	switch dispute.Status {
	case types.DisputeStatusEvidence:
		// Evidence deadline passed — transition to voting
		if currentHeight <= dispute.EvidenceDeadline {
			return nil, fmt.Errorf("evidence deadline has not passed: current %d, deadline %d",
				currentHeight, dispute.EvidenceDeadline)
		}
		if err := k.Keeper.TransitionToVoting(ctx, dispute); err != nil {
			return nil, fmt.Errorf("failed to transition to voting: %w", err)
		}
		return &types.MsgTimeoutDisputeResponse{NewStatus: string(types.DisputeStatusVoting)}, nil

	case types.DisputeStatusOpen:
		// Open disputes past evidence deadline — expire
		if currentHeight <= dispute.EvidenceDeadline {
			return nil, fmt.Errorf("evidence deadline has not passed: current %d, deadline %d",
				currentHeight, dispute.EvidenceDeadline)
		}
		dispute.Status = types.DisputeStatusExpired
		k.SetDispute(ctx, dispute)
		return &types.MsgTimeoutDisputeResponse{NewStatus: string(types.DisputeStatusExpired)}, nil

	case types.DisputeStatusResolved, types.DisputeStatusExpired:
		return nil, fmt.Errorf("%w: %s", types.ErrDisputeAlreadyResolved, msg.DisputeId)

	default:
		return nil, fmt.Errorf("dispute in %s phase cannot be timed out via this message", dispute.Status)
	}
}
