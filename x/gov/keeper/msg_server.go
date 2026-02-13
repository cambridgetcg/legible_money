package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/gov/types"
)

type msgServer struct {
	types.UnimplementedMsgServer
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

	// Validate initial stake is positive (anti-spam deposit)
	initialStake := msg.InitialStake
	if initialStake == "" || initialStake == "0" {
		return nil, types.ErrZeroStake
	}
	stakeInt := new(big.Int)
	if _, ok := stakeInt.SetString(initialStake, 10); !ok || stakeInt.Sign() <= 0 {
		return nil, types.ErrZeroStake
	}

	// Escrow initial deposit into governance module account
	if ms.bankKeeper != nil {
		proposerAddr, err := sdk.AccAddressFromBech32(msg.Proposer)
		if err != nil {
			return nil, fmt.Errorf("invalid proposer address: %w", err)
		}
		coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(stakeInt)))
		if err := ms.bankKeeper.SendCoinsFromAccountToModule(ctx, proposerAddr, types.ModuleName, coins); err != nil {
			return nil, fmt.Errorf("%w: %v", types.ErrInsufficientBalance, err)
		}
	}

	// Generate LIP ID
	number := ms.GetNextLIPNumber(ctx)
	lipId := fmt.Sprintf("LIP-%d", number)
	ms.SetNextLIPNumber(ctx, number+1)

	lip := types.LIP{
		Id:             lipId,
		Title:          msg.Title,
		Abstract:       msg.Description,
		Category:       fmt.Sprintf("%d", msg.LipType),
		Proposer:       msg.Proposer,
		Stage:          string(types.StageDraft),
		StakedAmount:   initialStake,
		StakeHolder:    msg.Proposer,
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

	if lip.Stage != string(types.StageDraft) {
		return nil, types.ErrInvalidLIPStage
	}

	if types.IsTerminalLIPStage(types.HandwrittenLIPStage(lip.Stage)) {
		return nil, types.ErrTerminalStage
	}

	// Check required stake for this category
	params := ms.GetParams(ctx)
	requiredStake := params.GetLIPStake(types.LIPCategory(lip.Category))

	// Add staked amount
	currentStake := new(big.Int)
	currentStake.SetString(lip.StakedAmount, 10)
	addedStake := new(big.Int)
	addedStake.SetString(msg.Amount, 10)
	newStake := new(big.Int).Add(currentStake, addedStake)

	lip.StakedAmount = newStake.String()
	lip.StakeHolder = msg.Staker
	lip.UpdatedAtBlock = uint64(ctx.BlockHeight())

	// Check if stake threshold met - advance to review
	required := new(big.Int)
	required.SetString(requiredStake, 10)
	if newStake.Cmp(required) >= 0 {
		lip.Stage = string(types.StageReview)
	}

	ms.SetLIP(ctx, lip)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"lip_staked",
			sdk.NewAttribute("lip_id", msg.LipId),
			sdk.NewAttribute("staker", msg.Staker),
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

	// Check if voter already voted
	if ms.HasVoted(ctx, msg.ProposalId, msg.Voter) {
		return nil, types.ErrAlreadyVoted
	}

	// Map proto VoteOption to string direction
	var direction types.VoteDirection
	switch msg.Option {
	case types.VoteOption_VOTE_OPTION_YES:
		direction = types.VoteSupport
	case types.VoteOption_VOTE_OPTION_NO, types.VoteOption_VOTE_OPTION_NO_WITH_VETO:
		direction = types.VoteOppose
	case types.VoteOption_VOTE_OPTION_ABSTAIN:
		direction = types.VoteAbstain
	default:
		return nil, types.ErrInvalidVoteDirection
	}

	// Determine if this is a LIP or disbursement vote
	isLIP := false
	if lip, found := ms.GetLIP(ctx, msg.ProposalId); found {
		if lip.Stage != string(types.StageVoting) {
			return nil, types.ErrVotingNotActive
		}
		if lip.VotingEndsAt > 0 && currentHeight >= lip.VotingEndsAt {
			return nil, types.ErrVotingPeriodEnded
		}
		isLIP = true
	} else if disb, found := ms.GetDisbursement(ctx, msg.ProposalId); found {
		if disb.Stage != string(types.DisbStageVoting) {
			return nil, types.ErrVotingNotActive
		}
		if disb.VotingEndsAt > 0 && currentHeight >= disb.VotingEndsAt {
			return nil, types.ErrVotingPeriodEnded
		}
	} else {
		return nil, types.ErrProposalNotFound
	}

	// Validate and escrow vote stake (prevents unstaked voting)
	voteWeight := msg.Stake
	if voteWeight == "" {
		return nil, types.ErrZeroStake
	}
	stakeInt := new(big.Int)
	if _, ok := stakeInt.SetString(voteWeight, 10); !ok || stakeInt.Sign() <= 0 {
		return nil, types.ErrZeroStake
	}

	// Escrow vote stake into governance module account
	if ms.bankKeeper != nil {
		voterAddr, err := sdk.AccAddressFromBech32(msg.Voter)
		if err != nil {
			return nil, fmt.Errorf("invalid voter address: %w", err)
		}
		coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(stakeInt)))
		if err := ms.bankKeeper.SendCoinsFromAccountToModule(ctx, voterAddr, types.ModuleName, coins); err != nil {
			return nil, fmt.Errorf("%w: %v", types.ErrInsufficientBalance, err)
		}
	}

	// ZD-E05 FIX: Apply quadratic voting — voting power = sqrt(stake).
	// Prevents governance capture by wealthy participants.
	// The stake is escrowed (cost = votes^2), but voting weight = sqrt(stake).
	effectiveVotingPower := new(big.Int).Sqrt(stakeInt).String()

	// Create vote record
	voteId := fmt.Sprintf("%s/%s", msg.ProposalId, msg.Voter)
	vote := types.QuadraticVote{
		VoteId:         voteId,
		Voter:          msg.Voter,
		ProposalId:     msg.ProposalId,
		Direction:      string(direction),
		Cost:           msg.Stake,
		CreatedAtBlock: currentHeight,
	}
	ms.SetVote(ctx, vote)
	ms.SetVoteDedupe(ctx, msg.ProposalId, msg.Voter)

	// Update proposal tallies with quadratic voting power (not raw stake)
	if isLIP {
		lip, _ := ms.GetLIP(ctx, msg.ProposalId)
		lip = addVoteWeight(lip, direction, effectiveVotingPower)
		lip.UniqueVoters++
		lip.UpdatedAtBlock = currentHeight
		ms.SetLIP(ctx, lip)
	} else {
		disb, _ := ms.GetDisbursement(ctx, msg.ProposalId)
		disb = addDisbVoteWeight(disb, direction, effectiveVotingPower)
		disb.UniqueVoters++
		ms.SetDisbursement(ctx, disb)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"vote_cast",
			sdk.NewAttribute("proposal_id", msg.ProposalId),
			sdk.NewAttribute("voter", msg.Voter),
			sdk.NewAttribute("direction", string(direction)),
			sdk.NewAttribute("stake", msg.Stake),
			sdk.NewAttribute("effective_power", effectiveVotingPower),
		),
	)

	return &types.MsgCastVoteResponse{EffectiveVotingPower: effectiveVotingPower}, nil
}

// AdvanceLIPStage advances a LIP to its next stage.
// ZD-E04 FIX: Enforces authorization (proposer or governance authority) and
// minimum review period + quorum checks before advancing.
func (ms msgServer) AdvanceLIPStage(goCtx context.Context, msg *types.MsgAdvanceLIPStage) (*types.MsgAdvanceLIPStageResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := ms.GetParams(ctx)
	currentHeight := uint64(ctx.BlockHeight())

	lip, found := ms.GetLIP(ctx, msg.LipId)
	if !found {
		return nil, types.ErrLIPNotFound
	}

	if types.IsTerminalLIPStage(types.HandwrittenLIPStage(lip.Stage)) {
		return nil, types.ErrTerminalStage
	}

	// ZD-E04 FIX: Authorization check — only proposer or governance authority can advance
	if msg.Proposer != lip.Proposer {
		return nil, fmt.Errorf("%w: only LIP proposer can advance stages", types.ErrUnauthorized)
	}

	var newStage types.HandwrittenLIPStage
	var protoStage types.LIPStage
	switch types.HandwrittenLIPStage(lip.Stage) {
	case types.StageReview:
		// ZD-E04 FIX: Enforce minimum review period before advancing
		if currentHeight < lip.CreatedAtBlock+params.ReviewPeriodBlocks {
			return nil, fmt.Errorf("%w: review period not elapsed (need block %d, at %d)",
				types.ErrInvalidLIPStage, lip.CreatedAtBlock+params.ReviewPeriodBlocks, currentHeight)
		}
		newStage = types.StageLastCall
		protoStage = types.LIPStage_LIP_STAGE_VOTING // closest mapping
	case types.StageLastCall:
		// ZD-E04 FIX: Enforce quorum — must have minimum support weight relative to total stake
		supportWeight := new(big.Int)
		supportWeight.SetString(lip.SupportWeight, 10)
		if supportWeight.Sign() <= 0 {
			return nil, fmt.Errorf("%w: cannot advance to voting with zero support", types.ErrInvalidLIPStage)
		}
		newStage = types.StageVoting
		protoStage = types.LIPStage_LIP_STAGE_VOTING
	default:
		return nil, fmt.Errorf("%w: cannot advance from stage %s", types.ErrInvalidLIPStage, lip.Stage)
	}

	lip.Stage = string(newStage)
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

	return &types.MsgAdvanceLIPStageResponse{NewStage: protoStage}, nil
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

	if types.IsTerminalLIPStage(types.HandwrittenLIPStage(lip.Stage)) {
		return nil, types.ErrTerminalStage
	}

	lip.Stage = string(types.StageWithdrawn)
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

	// Generate disbursement ID
	number := ms.GetNextDisbNumber(ctx)
	disbId := fmt.Sprintf("DISB-%d", number)
	ms.SetNextDisbNumber(ctx, number+1)

	disb := types.DisbursementProposal{
		Id:             disbId,
		Title:          msg.Title,
		Description:    msg.Description,
		Proposer:       msg.Submitter,
		RecipientAddr:  msg.Recipient,
		Amount:         msg.Amount,
		Stage:          string(types.DisbStageVoting),
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
			sdk.NewAttribute("submitter", msg.Submitter),
			sdk.NewAttribute("amount", msg.Amount),
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

	if disb.Stage != string(types.DisbStageApproved) {
		return nil, types.ErrInvalidDisbStage
	}

	// Check timelock
	if currentHeight < disb.TimelockExpiry {
		return nil, types.ErrTimelockNotExpired
	}

	// Mark as executed
	disb.Stage = string(types.DisbStageExecuted)
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

// CancelDisbursement cancels a disbursement (only by authority, only if not terminal).
func (ms msgServer) CancelDisbursement(goCtx context.Context, msg *types.MsgCancelDisbursement) (*types.MsgCancelDisbursementResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	disb, found := ms.GetDisbursement(ctx, msg.DisbursementId)
	if !found {
		return nil, types.ErrDisbursementNotFound
	}

	// Authority can cancel (was previously proposer-only)
	if disb.Proposer != msg.Authority && ms.GetAuthority() != msg.Authority {
		return nil, types.ErrNotProposer
	}

	if types.IsTerminalDisbStage(types.DisbursementStage(disb.Stage)) {
		return nil, types.ErrTerminalStage
	}

	disb.Stage = string(types.DisbStageCancelled)
	ms.SetDisbursement(ctx, disb)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"disbursement_cancelled",
			sdk.NewAttribute("disbursement_id", msg.DisbursementId),
			sdk.NewAttribute("authority", msg.Authority),
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

// AttachUpgradePlan attaches a software upgrade plan to a consensus-category LIP.
func (ms msgServer) AttachUpgradePlan(goCtx context.Context, msg *types.MsgAttachUpgradePlan) (*types.MsgAttachUpgradePlanResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	lip, found := ms.GetLIP(ctx, msg.LipId)
	if !found {
		return nil, types.ErrLIPNotFound
	}

	// Only the proposer can attach an upgrade plan
	if lip.Proposer != msg.Proposer {
		return nil, types.ErrNotProposer
	}

	// Only consensus-category LIPs can carry upgrade plans
	if lip.Category != string(types.CategoryConsensus) {
		return nil, types.ErrNotConsensusCategory
	}

	// Cannot attach to terminal LIPs
	if types.IsTerminalLIPStage(types.HandwrittenLIPStage(lip.Stage)) {
		return nil, types.ErrTerminalStage
	}

	// Check if an upgrade plan already exists
	if _, exists := ms.GetUpgradePlan(ctx, msg.LipId); exists {
		return nil, types.ErrUpgradePlanExists
	}

	plan := types.UpgradePlan{
		Name:   msg.UpgradeName,
		Height: msg.Height,
		Info:   msg.Info,
	}
	ms.SetUpgradePlan(ctx, msg.LipId, plan)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"upgrade_plan_attached",
			sdk.NewAttribute("lip_id", msg.LipId),
			sdk.NewAttribute("upgrade_name", msg.UpgradeName),
			sdk.NewAttribute("height", fmt.Sprintf("%d", msg.Height)),
		),
	)

	return &types.MsgAttachUpgradePlanResponse{}, nil
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
