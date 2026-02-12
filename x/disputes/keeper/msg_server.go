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
	types.UnimplementedMsgServer
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface (proto).
func NewMsgServerImpl(keeper Keeper) types.LegacyMsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.LegacyMsgServer = &msgServer{}

// ---------- Proto MsgServer methods ----------

// InitiateDispute implements the proto MsgServer.InitiateDispute method.
// Wraps the existing OpenDispute logic, mapping from proto MsgInitiateDispute fields.
func (k msgServer) InitiateDispute(goCtx context.Context, msg *types.MsgInitiateDispute) (*types.MsgInitiateDisputeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Map proto fields to internal OpenDispute call
	stake := msg.Stake
	bondAmount := new(big.Int)
	if _, ok := bondAmount.SetString(stake, 10); !ok || bondAmount.Sign() <= 0 {
		return nil, fmt.Errorf("%w: invalid stake amount", types.ErrInvalidBond)
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

	dispute, err := k.Keeper.OpenDispute(ctx, msg.TargetId, msg.Challenger, msg.Reason, stake)
	if err != nil {
		_ = k.bankKeeper.SendCoinsFromModuleToAccount(
			ctx, types.ModuleName, challengerAddr, sdk.NewCoins(bondCoin),
		)
		return nil, err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"dispute_opened",
			sdk.NewAttribute("dispute_id", dispute.Id),
			sdk.NewAttribute("target_id", msg.TargetId),
			sdk.NewAttribute("challenger", msg.Challenger),
			sdk.NewAttribute("stake", stake),
			sdk.NewAttribute("tier", fmt.Sprintf("%d", dispute.Tier)),
			sdk.NewAttribute("height", fmt.Sprintf("%d", ctx.BlockHeight())),
		),
	)

	return &types.MsgInitiateDisputeResponse{DisputeId: dispute.Id}, nil
}

// CommitEvidence implements the proto MsgServer.CommitEvidence method.
// Proto MsgCommitEvidence has: Submitter, DisputeId, CommitmentHash (no Side field).
func (k msgServer) CommitEvidence(goCtx context.Context, msg *types.MsgCommitEvidence) (*types.MsgCommitEvidenceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	dispute, found := k.GetDispute(ctx, msg.DisputeId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrDisputeNotFound, msg.DisputeId)
	}

	// Check dispute is in evidence phase
	if dispute.Status != string(types.DisputeStatusEvidence) {
		return nil, fmt.Errorf("%w: dispute is in %s phase", types.ErrNotInEvidencePhase, dispute.Status)
	}

	// Check evidence deadline
	currentHeight := uint64(ctx.BlockHeight())
	if currentHeight > dispute.EvidenceDeadline {
		return nil, fmt.Errorf("%w: deadline was at height %d",
			types.ErrEvidenceDeadlinePassed, dispute.EvidenceDeadline)
	}

	// Determine side from submitter address
	side := ""
	if msg.Submitter == dispute.Challenger {
		side = "challenger"
	} else if msg.Submitter == dispute.Defender {
		side = "defender"
	} else {
		return nil, fmt.Errorf("submitter %s is neither challenger nor defender", msg.Submitter)
	}

	// Check for existing commitment
	if _, exists := k.GetEvidenceCommitment(ctx, msg.DisputeId, msg.Submitter); exists {
		return nil, fmt.Errorf("evidence already committed for dispute %s by %s", msg.DisputeId, msg.Submitter)
	}

	commitment := types.EvidenceCommitment{
		DisputeId:   msg.DisputeId,
		Submitter:   msg.Submitter,
		Side:        side,
		ContentHash: msg.CommitmentHash,
		CommittedAt: currentHeight,
		Revealed:    false,
	}
	k.SetEvidenceCommitment(ctx, commitment)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"evidence_committed",
			sdk.NewAttribute("dispute_id", msg.DisputeId),
			sdk.NewAttribute("submitter", msg.Submitter),
			sdk.NewAttribute("side", side),
		),
	)

	return &types.MsgCommitEvidenceResponse{}, nil
}

// RevealEvidence implements the proto MsgServer.RevealEvidence method.
// Wraps the existing SubmitEvidence logic.
func (k msgServer) RevealEvidence(goCtx context.Context, msg *types.MsgRevealEvidence) (*types.MsgRevealEvidenceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	dispute, found := k.GetDispute(ctx, msg.DisputeId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrDisputeNotFound, msg.DisputeId)
	}

	// Check dispute is in evidence phase
	if dispute.Status != string(types.DisputeStatusEvidence) {
		return nil, fmt.Errorf("%w: dispute is in %s phase", types.ErrNotInEvidencePhase, dispute.Status)
	}

	// Check evidence deadline
	currentHeight := uint64(ctx.BlockHeight())
	if currentHeight > dispute.EvidenceDeadline {
		return nil, fmt.Errorf("%w: deadline was at height %d",
			types.ErrEvidenceDeadlinePassed, dispute.EvidenceDeadline)
	}

	// Determine side from submitter
	side := ""
	if msg.Submitter == dispute.Challenger {
		side = "challenger"
	} else if msg.Submitter == dispute.Defender {
		side = "defender"
	} else {
		return nil, fmt.Errorf("submitter %s is neither challenger nor defender", msg.Submitter)
	}

	// Check if a commitment exists — if so, verify hash match
	if commitment, exists := k.GetEvidenceCommitment(ctx, msg.DisputeId, msg.Submitter); exists {
		if commitment.Revealed {
			return nil, fmt.Errorf("evidence already revealed for dispute %s by %s", msg.DisputeId, msg.Submitter)
		}
		// Build content blob from evidence items for hash verification
		contentBlob := ""
		for _, ev := range msg.Evidence {
			contentBlob += ev.Content
		}
		contentHash := sha256.Sum256([]byte(contentBlob + msg.Nonce))
		computedHex := hex.EncodeToString(contentHash[:])
		if computedHex != commitment.ContentHash {
			return nil, fmt.Errorf("%w: committed hash %s does not match content hash %s",
				types.ErrEvidenceHashMismatch, commitment.ContentHash, computedHex)
		}
		commitment.Revealed = true
		k.SetEvidenceCommitment(ctx, commitment)
	}

	// Store each evidence item
	for _, ev := range msg.Evidence {
		evidenceId := GenerateEvidenceID(msg.DisputeId, msg.Submitter, currentHeight)
		evidence := types.DisputeEvidence{
			Id:          evidenceId,
			DisputeId:   msg.DisputeId,
			Submitter:   msg.Submitter,
			Side:        side,
			Content:     ev.Content,
			SubmittedAt: currentHeight,
		}
		k.SetEvidence(ctx, evidence)
		dispute.EvidenceCount++
	}

	k.SetDispute(ctx, dispute)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"evidence_revealed",
			sdk.NewAttribute("dispute_id", msg.DisputeId),
			sdk.NewAttribute("submitter", msg.Submitter),
			sdk.NewAttribute("side", side),
		),
	)

	return &types.MsgRevealEvidenceResponse{}, nil
}

// ArbiterVote implements the proto MsgServer.ArbiterVote method.
func (k msgServer) ArbiterVote(goCtx context.Context, msg *types.MsgArbiterVote) (*types.MsgArbiterVoteResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Map proto ArbiterDecision enum to string vote
	voteStr := ""
	switch msg.Vote {
	case types.ArbiterDecision_ARBITER_DECISION_CHALLENGER:
		voteStr = "overturn"
	case types.ArbiterDecision_ARBITER_DECISION_DEFENDER:
		voteStr = "uphold"
	case types.ArbiterDecision_ARBITER_DECISION_ABSTAIN:
		voteStr = "abstain"
	default:
		return nil, fmt.Errorf("invalid vote decision")
	}

	if err := k.Keeper.CastVote(ctx, msg.DisputeId, msg.Arbiter, voteStr, msg.Reasoning); err != nil {
		return nil, err
	}

	return &types.MsgArbiterVoteResponse{}, nil
}

// EscalateDispute implements the proto MsgServer.EscalateDispute method.
// Proto field: AdditionalStake (was AdditionalBond in hand-written type).
func (k msgServer) EscalateDispute(goCtx context.Context, msg *types.MsgEscalateDispute) (*types.MsgEscalateDisputeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := k.Keeper.EscalateDispute(ctx, msg.DisputeId, msg.Requester, msg.AdditionalStake); err != nil {
		return nil, err
	}

	return &types.MsgEscalateDisputeResponse{}, nil
}

// SettleDispute implements the proto MsgServer.SettleDispute method.
// Proto field: Authority (was Sender in hand-written type).
func (k msgServer) SettleDispute(goCtx context.Context, msg *types.MsgSettleDispute) (*types.MsgSettleDisputeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	dispute, found := k.GetDispute(ctx, msg.DisputeId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrDisputeNotFound, msg.DisputeId)
	}

	// Must be in voting phase
	if dispute.Status != string(types.DisputeStatusVoting) {
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

	return &types.MsgSettleDisputeResponse{}, nil
}

// TimeoutDispute implements the proto MsgServer.TimeoutDispute method.
// Proto field: Authority (was Sender in hand-written type).
func (k msgServer) TimeoutDispute(goCtx context.Context, msg *types.MsgTimeoutDispute) (*types.MsgTimeoutDisputeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	dispute, found := k.GetDispute(ctx, msg.DisputeId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrDisputeNotFound, msg.DisputeId)
	}

	currentHeight := uint64(ctx.BlockHeight())

	switch dispute.Status {
	case string(types.DisputeStatusEvidence):
		// Evidence deadline passed — transition to voting
		if currentHeight <= dispute.EvidenceDeadline {
			return nil, fmt.Errorf("evidence deadline has not passed: current %d, deadline %d",
				currentHeight, dispute.EvidenceDeadline)
		}
		if err := k.Keeper.TransitionToVoting(ctx, dispute); err != nil {
			return nil, fmt.Errorf("failed to transition to voting: %w", err)
		}
		return &types.MsgTimeoutDisputeResponse{}, nil

	case string(types.DisputeStatusOpen):
		// Open disputes past evidence deadline — expire
		if currentHeight <= dispute.EvidenceDeadline {
			return nil, fmt.Errorf("evidence deadline has not passed: current %d, deadline %d",
				currentHeight, dispute.EvidenceDeadline)
		}
		dispute.Status = string(types.DisputeStatusExpired)
		k.SetDispute(ctx, dispute)
		return &types.MsgTimeoutDisputeResponse{}, nil

	case string(types.DisputeStatusResolved), string(types.DisputeStatusExpired):
		return nil, fmt.Errorf("%w: %s", types.ErrDisputeAlreadyResolved, msg.DisputeId)

	default:
		return nil, fmt.Errorf("dispute in %s phase cannot be timed out via this message", dispute.Status)
	}
}

// ---------- Legacy methods (used by tests and internal keeper logic) ----------
// These methods use the hand-written Msg types that have no proto name collision.

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

// SubmitEvidence handles MsgSubmitEvidence — reveals evidence for a dispute (phase 2 of commit-reveal).
func (k msgServer) SubmitEvidence(goCtx context.Context, msg *types.MsgSubmitEvidence) (*types.MsgSubmitEvidenceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	dispute, found := k.GetDispute(ctx, msg.DisputeId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrDisputeNotFound, msg.DisputeId)
	}

	// Check dispute is in evidence phase
	if dispute.Status != string(types.DisputeStatusEvidence) {
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
