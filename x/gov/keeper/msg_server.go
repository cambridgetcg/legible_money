package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/gov/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// SubmitLIP creates a new Legible Improvement Proposal.
func (ms msgServer) SubmitLIP(goCtx context.Context, msg *types.MsgSubmitLIP) (*types.MsgSubmitLIPResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	currentHeight := uint64(ctx.BlockHeight())

	// Validate category
	if !types.ValidLIPCategories[msg.Category] {
		return nil, types.ErrInvalidLIPCategory
	}

	// Generate LIP ID
	number := ms.GetNextLIPNumber(ctx)
	lipId := fmt.Sprintf("LIP-%d", number)
	ms.SetNextLIPNumber(ctx, number+1)

	lip := types.LIP{
		Id:             lipId,
		Title:          msg.Title,
		Abstract:       msg.Abstract,
		Category:       msg.Category,
		Proposer:       msg.Proposer,
		Stage:          types.StageDraft,
		StakedAmount:   "0",
		StakeHolder:    "",
		CreatedAtBlock: currentHeight,
		UpdatedAtBlock: currentHeight,
		VotingEndsAt:   0,
		SupportWeight:  "0",
		OpposeWeight:   "0",
		AbstainWeight:  "0",
		UniqueVoters:   0,
		QuorumMet:      false,
	}

	ms.SetLIP(ctx, lip)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"lip_submitted",
			sdk.NewAttribute("lip_id", lipId),
			sdk.NewAttribute("proposer", msg.Proposer),
			sdk.NewAttribute("category", string(msg.Category)),
		),
	)

	return &types.MsgSubmitLIPResponse{LipId: lipId}, nil
}

// StakeLIP stakes tokens on a LIP to advance it from draft to review.
func (ms msgServer) StakeLIP(goCtx context.Context, msg *types.MsgStakeLIP) (*types.MsgStakeLIPResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	lip, found := ms.GetLIP(ctx, msg.LipId)
	if !found {
		return nil, types.ErrLIPNotFound
	}

	if lip.Stage != types.StageDraft {
		return nil, types.ErrInvalidLIPStage
	}

	if types.IsTerminalLIPStage(lip.Stage) {
		return nil, types.ErrTerminalStage
	}

	// Check required stake for this category
	params := ms.GetParams(ctx)
	requiredStake := params.GetLIPStake(lip.Category)

	// Add staked amount
	currentStake := new(big.Int)
	currentStake.SetString(lip.StakedAmount, 10)
	addedStake := new(big.Int)
	addedStake.SetString(msg.Amount, 10)
	newStake := new(big.Int).Add(currentStake, addedStake)

	lip.StakedAmount = newStake.String()
	lip.StakeHolder = msg.Sender
	lip.UpdatedAtBlock = uint64(ctx.BlockHeight())

	// Check if stake threshold met — advance to review
	required := new(big.Int)
	required.SetString(requiredStake, 10)
	if newStake.Cmp(required) >= 0 {
		lip.Stage = types.StageReview
	}

	ms.SetLIP(ctx, lip)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"lip_staked",
			sdk.NewAttribute("lip_id", msg.LipId),
			sdk.NewAttribute("sender", msg.Sender),
			sdk.NewAttribute("amount", msg.Amount),
			sdk.NewAttribute("stage", string(lip.Stage)),
		),
	)

	return &types.MsgStakeLIPResponse{}, nil
}

// CastVote casts a quadratic vote on a LIP or disbursement proposal.
func (ms msgServer) CastVote(goCtx context.Context, msg *types.MsgCastVote) (*types.MsgCastVoteResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	currentHeight := uint64(ctx.BlockHeight())
	params := ms.GetParams(ctx)

	// Validate direction
	if !types.ValidVoteDirections[msg.Direction] {
		return nil, types.ErrInvalidVoteDirection
	}

	// Check max votes per voter
	if msg.Votes > params.MaxVotesPerVoter {
		return nil, types.ErrMaxVotesExceeded
	}

	// Check if voter already voted
	if ms.HasVoted(ctx, msg.ProposalId, msg.Voter) {
		return nil, types.ErrAlreadyVoted
	}

	// Determine if this is a LIP or disbursement vote
	isLIP := false
	if lip, found := ms.GetLIP(ctx, msg.ProposalId); found {
		if lip.Stage != types.StageVoting {
			return nil, types.ErrVotingNotActive
		}
		if lip.VotingEndsAt > 0 && currentHeight >= lip.VotingEndsAt {
			return nil, types.ErrVotingPeriodEnded
		}
		isLIP = true
	} else if disb, found := ms.GetDisbursement(ctx, msg.ProposalId); found {
		if disb.Stage != types.DisbStageVoting {
			return nil, types.ErrVotingNotActive
		}
		if disb.VotingEndsAt > 0 && currentHeight >= disb.VotingEndsAt {
			return nil, types.ErrVotingPeriodEnded
		}
	} else {
		return nil, types.ErrProposalNotFound
	}

	// Calculate quadratic vote cost: cost = votes^2 * baseCostPerVote
	baseCost := new(big.Int)
	baseCost.SetString(params.BaseCostPerVote, 10)
	votesBI := big.NewInt(int64(msg.Votes))
	cost := new(big.Int).Mul(votesBI, votesBI) // votes^2
	cost.Mul(cost, baseCost)                    // * baseCost

	// Calculate vote weight (sqrt-like: just the number of votes, since cost is quadratic)
	voteWeight := fmt.Sprintf("%d", msg.Votes)

	// Create vote record
	voteId := fmt.Sprintf("%s/%s", msg.ProposalId, msg.Voter)
	vote := types.QuadraticVote{
		VoteId:         voteId,
		Voter:          msg.Voter,
		ProposalId:     msg.ProposalId,
		Direction:      msg.Direction,
		Votes:          msg.Votes,
		Cost:           cost.String(),
		CreatedAtBlock: currentHeight,
	}
	ms.SetVote(ctx, vote)
	ms.SetVoteDedupe(ctx, msg.ProposalId, msg.Voter)

	// Update proposal tallies
	if isLIP {
		lip, _ := ms.GetLIP(ctx, msg.ProposalId)
		lip = addVoteWeight(lip, msg.Direction, voteWeight)
		lip.UniqueVoters++
		lip.UpdatedAtBlock = currentHeight
		ms.SetLIP(ctx, lip)
	} else {
		disb, _ := ms.GetDisbursement(ctx, msg.ProposalId)
		disb = addDisbVoteWeight(disb, msg.Direction, voteWeight)
		disb.UniqueVoters++
		ms.SetDisbursement(ctx, disb)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"vote_cast",
			sdk.NewAttribute("proposal_id", msg.ProposalId),
			sdk.NewAttribute("voter", msg.Voter),
			sdk.NewAttribute("direction", string(msg.Direction)),
			sdk.NewAttribute("votes", fmt.Sprintf("%d", msg.Votes)),
			sdk.NewAttribute("cost", cost.String()),
		),
	)

	return &types.MsgCastVoteResponse{Cost: cost.String()}, nil
}

// AdvanceLIPStage advances a LIP to its next stage.
func (ms msgServer) AdvanceLIPStage(goCtx context.Context, msg *types.MsgAdvanceLIPStage) (*types.MsgAdvanceLIPStageResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := ms.GetParams(ctx)
	currentHeight := uint64(ctx.BlockHeight())

	lip, found := ms.GetLIP(ctx, msg.LipId)
	if !found {
		return nil, types.ErrLIPNotFound
	}

	if types.IsTerminalLIPStage(lip.Stage) {
		return nil, types.ErrTerminalStage
	}

	var newStage types.LIPStage
	switch lip.Stage {
	case types.StageReview:
		newStage = types.StageLastCall
	case types.StageLastCall:
		newStage = types.StageVoting
	default:
		return nil, fmt.Errorf("%w: cannot advance from stage %s", types.ErrInvalidLIPStage, lip.Stage)
	}

	lip.Stage = newStage
	lip.UpdatedAtBlock = currentHeight

	// Set voting deadline when entering voting stage
	if newStage == types.StageVoting {
		lip.VotingEndsAt = currentHeight + params.VotingPeriodBlocks
	}

	ms.SetLIP(ctx, lip)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"lip_advanced",
			sdk.NewAttribute("lip_id", msg.LipId),
			sdk.NewAttribute("new_stage", string(newStage)),
		),
	)

	return &types.MsgAdvanceLIPStageResponse{NewStage: newStage}, nil
}

// WithdrawLIP withdraws a LIP (only by proposer, only if not terminal).
func (ms msgServer) WithdrawLIP(goCtx context.Context, msg *types.MsgWithdrawLIP) (*types.MsgWithdrawLIPResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	lip, found := ms.GetLIP(ctx, msg.LipId)
	if !found {
		return nil, types.ErrLIPNotFound
	}

	if lip.Proposer != msg.Proposer {
		return nil, types.ErrNotProposer
	}

	if types.IsTerminalLIPStage(lip.Stage) {
		return nil, types.ErrTerminalStage
	}

	lip.Stage = types.StageWithdrawn
	lip.UpdatedAtBlock = uint64(ctx.BlockHeight())
	ms.SetLIP(ctx, lip)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"lip_withdrawn",
			sdk.NewAttribute("lip_id", msg.LipId),
			sdk.NewAttribute("proposer", msg.Proposer),
		),
	)

	return &types.MsgWithdrawLIPResponse{}, nil
}

// SubmitDisbursement creates a new treasury disbursement proposal.
func (ms msgServer) SubmitDisbursement(goCtx context.Context, msg *types.MsgSubmitDisbursement) (*types.MsgSubmitDisbursementResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	currentHeight := uint64(ctx.BlockHeight())
	params := ms.GetParams(ctx)

	if !types.ValidDisbursementCategories[msg.Category] {
		return nil, fmt.Errorf("invalid disbursement category: %s", msg.Category)
	}

	// Generate disbursement ID
	number := ms.GetNextDisbNumber(ctx)
	disbId := fmt.Sprintf("DISB-%d", number)
	ms.SetNextDisbNumber(ctx, number+1)

	disb := types.DisbursementProposal{
		Id:             disbId,
		Title:          msg.Title,
		Description:    msg.Description,
		Category:       msg.Category,
		Proposer:       msg.Proposer,
		RecipientAddr:  msg.RecipientAddr,
		Amount:         msg.Amount,
		Stage:          types.DisbStageVoting,
		StakedAmount:   "0",
		StakeHolder:    "",
		CreatedAtBlock: currentHeight,
		VotingEndsAt:   currentHeight + params.VotingPeriodBlocks,
		SupportWeight:  "0",
		OpposeWeight:   "0",
		AbstainWeight:  "0",
		UniqueVoters:   0,
		QuorumMet:      false,
	}

	ms.SetDisbursement(ctx, disb)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"disbursement_submitted",
			sdk.NewAttribute("disbursement_id", disbId),
			sdk.NewAttribute("proposer", msg.Proposer),
			sdk.NewAttribute("amount", msg.Amount),
			sdk.NewAttribute("category", string(msg.Category)),
		),
	)

	return &types.MsgSubmitDisbursementResponse{DisbursementId: disbId}, nil
}

// ExecuteDisbursement executes an approved disbursement after the timelock.
func (ms msgServer) ExecuteDisbursement(goCtx context.Context, msg *types.MsgExecuteDisbursement) (*types.MsgExecuteDisbursementResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	currentHeight := uint64(ctx.BlockHeight())

	disb, found := ms.GetDisbursement(ctx, msg.DisbursementId)
	if !found {
		return nil, types.ErrDisbursementNotFound
	}

	if disb.Stage != types.DisbStageApproved {
		return nil, types.ErrInvalidDisbStage
	}

	// Check timelock
	if currentHeight < disb.TimelockExpiry {
		return nil, types.ErrTimelockNotExpired
	}

	// Mark as executed
	disb.Stage = types.DisbStageExecuted
	ms.SetDisbursement(ctx, disb)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"disbursement_executed",
			sdk.NewAttribute("disbursement_id", msg.DisbursementId),
			sdk.NewAttribute("recipient", disb.RecipientAddr),
			sdk.NewAttribute("amount", disb.Amount),
		),
	)

	return &types.MsgExecuteDisbursementResponse{}, nil
}

// CancelDisbursement cancels a disbursement (only by proposer, only if not terminal).
func (ms msgServer) CancelDisbursement(goCtx context.Context, msg *types.MsgCancelDisbursement) (*types.MsgCancelDisbursementResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	disb, found := ms.GetDisbursement(ctx, msg.DisbursementId)
	if !found {
		return nil, types.ErrDisbursementNotFound
	}

	if disb.Proposer != msg.Proposer {
		return nil, types.ErrNotProposer
	}

	if types.IsTerminalDisbStage(disb.Stage) {
		return nil, types.ErrTerminalStage
	}

	disb.Stage = types.DisbStageCancelled
	ms.SetDisbursement(ctx, disb)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"disbursement_cancelled",
			sdk.NewAttribute("disbursement_id", msg.DisbursementId),
			sdk.NewAttribute("proposer", msg.Proposer),
		),
	)

	return &types.MsgCancelDisbursementResponse{}, nil
}

// UpdateParams updates governance module parameters (authority only).
func (ms msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if ms.GetAuthority() != msg.Authority {
		return nil, types.ErrUnauthorized
	}

	if err := msg.Params.Validate(); err != nil {
		return nil, err
	}

	ms.SetParams(ctx, msg.Params)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"params_updated",
			sdk.NewAttribute("authority", msg.Authority),
		),
	)

	return &types.MsgUpdateParamsResponse{}, nil
}

// --- Helpers ---

// addVoteWeight adds vote weight to a LIP's tally.
func addVoteWeight(lip types.LIP, direction types.VoteDirection, weight string) types.LIP {
	switch direction {
	case types.VoteSupport:
		lip.SupportWeight = addBigInts(lip.SupportWeight, weight)
	case types.VoteOppose:
		lip.OpposeWeight = addBigInts(lip.OpposeWeight, weight)
	case types.VoteAbstain:
		lip.AbstainWeight = addBigInts(lip.AbstainWeight, weight)
	}
	return lip
}

// addDisbVoteWeight adds vote weight to a disbursement's tally.
func addDisbVoteWeight(disb types.DisbursementProposal, direction types.VoteDirection, weight string) types.DisbursementProposal {
	switch direction {
	case types.VoteSupport:
		disb.SupportWeight = addBigInts(disb.SupportWeight, weight)
	case types.VoteOppose:
		disb.OpposeWeight = addBigInts(disb.OpposeWeight, weight)
	case types.VoteAbstain:
		disb.AbstainWeight = addBigInts(disb.AbstainWeight, weight)
	}
	return disb
}

// addBigInts adds two big integer strings.
func addBigInts(a, b string) string {
	ai := new(big.Int)
	bi := new(big.Int)
	ai.SetString(a, 10)
	bi.SetString(b, 10)
	return new(big.Int).Add(ai, bi).String()
}
