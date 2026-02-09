package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/research/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns a msg server implementation.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// SubmitResearch handles a new research submission.
func (m msgServer) SubmitResearch(
	goCtx context.Context,
	msg *types.MsgSubmitResearch,
) (*types.MsgSubmitResearchResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	params := m.Keeper.GetParams(ctx)

	// Validate stake meets minimum
	stakeInt := new(big.Int)
	if _, ok := stakeInt.SetString(msg.Stake, 10); !ok {
		return nil, fmt.Errorf("invalid stake amount")
	}
	minStake := new(big.Int)
	minStake.SetString(params.MinResearchStake, 10)
	if stakeInt.Cmp(minStake) < 0 {
		return nil, fmt.Errorf("%w: need %s, got %s", types.ErrInsufficientStake, params.MinResearchStake, msg.Stake)
	}

	// Transfer stake to module
	submitterAddr, err := sdk.AccAddressFromBech32(msg.Submitter)
	if err != nil {
		return nil, fmt.Errorf("invalid submitter address: %w", err)
	}

	coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(stakeInt)))
	if err := m.Keeper.bankKeeper.SendCoinsFromAccountToModule(ctx, submitterAddr, types.ModuleName, coins); err != nil {
		return nil, fmt.Errorf("failed to lock stake: %w", err)
	}

	// Create research submission
	researchId := m.Keeper.nextResearchId(ctx)
	research := types.Research{
		Id:           researchId,
		Submitter:    msg.Submitter,
		Title:        msg.Title,
		Description:  msg.Description,
		Domain:       msg.Domain,
		ResearchType: msg.ResearchType,
		TargetFactId: msg.TargetFactId,
		Stake:        msg.Stake,
		Status:       types.ResearchStatusSubmitted,
		CreatedAt:    uint64(ctx.BlockHeight()),
		UpdatedAt:    uint64(ctx.BlockHeight()),
	}
	m.Keeper.SetResearch(ctx, research)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"research_submitted",
			sdk.NewAttribute("research_id", researchId),
			sdk.NewAttribute("submitter", msg.Submitter),
			sdk.NewAttribute("domain", msg.Domain),
			sdk.NewAttribute("research_type", string(msg.ResearchType)),
			sdk.NewAttribute("stake", msg.Stake),
		),
	)

	return &types.MsgSubmitResearchResponse{ResearchId: researchId}, nil
}

// ChallengeResearch challenges an existing research submission.
func (m msgServer) ChallengeResearch(
	goCtx context.Context,
	msg *types.MsgChallengeResearch,
) (*types.MsgChallengeResearchResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	params := m.Keeper.GetParams(ctx)

	// Fetch research
	research, found := m.Keeper.GetResearch(ctx, msg.ResearchId)
	if !found {
		return nil, types.ErrSubmissionNotFound
	}

	// Only submitted or under_review can be challenged
	if research.Status != types.ResearchStatusSubmitted && research.Status != types.ResearchStatusUnderReview {
		return nil, types.ErrResearchNotChallengeable
	}

	// Validate stake
	stakeInt := new(big.Int)
	stakeInt.SetString(msg.Stake, 10)
	minStake := new(big.Int)
	minStake.SetString(params.MinChallengeStake, 10)
	if stakeInt.Cmp(minStake) < 0 {
		return nil, fmt.Errorf("%w: need %s, got %s", types.ErrInsufficientStake, params.MinChallengeStake, msg.Stake)
	}

	// Lock challenger's stake
	challengerAddr, _ := sdk.AccAddressFromBech32(msg.Challenger)
	coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(stakeInt)))
	if err := m.Keeper.bankKeeper.SendCoinsFromAccountToModule(ctx, challengerAddr, types.ModuleName, coins); err != nil {
		return nil, fmt.Errorf("failed to lock challenge stake: %w", err)
	}

	// Update status
	research.Status = types.ResearchStatusChallenged
	research.UpdatedAt = uint64(ctx.BlockHeight())
	m.Keeper.SetResearch(ctx, research)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"research_challenged",
			sdk.NewAttribute("research_id", msg.ResearchId),
			sdk.NewAttribute("challenger", msg.Challenger),
			sdk.NewAttribute("reason", msg.Reason),
		),
	)

	return &types.MsgChallengeResearchResponse{}, nil
}

// ReviewResearch submits a peer review for a research submission.
func (m msgServer) ReviewResearch(
	goCtx context.Context,
	msg *types.MsgReviewResearch,
) (*types.MsgReviewResearchResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Fetch research
	research, found := m.Keeper.GetResearch(ctx, msg.ResearchId)
	if !found {
		return nil, types.ErrSubmissionNotFound
	}

	// Must be submitted or under_review to accept reviews
	if research.Status != types.ResearchStatusSubmitted && research.Status != types.ResearchStatusUnderReview {
		return nil, types.ErrResearchNotUnderReview
	}

	// Check duplicate review
	if m.Keeper.HasReviewerReviewed(ctx, msg.ResearchId, msg.Reviewer) {
		return nil, types.ErrAlreadyReviewed
	}

	// Create peer review
	reviewId := m.Keeper.nextReviewId(ctx)
	review := types.PeerReview{
		Id:         reviewId,
		ResearchId: msg.ResearchId,
		Reviewer:   msg.Reviewer,
		Verdict:    msg.Verdict,
		Score:      msg.Score,
		Reasoning:  msg.Reasoning,
		ReviewedAt: uint64(ctx.BlockHeight()),
	}
	m.Keeper.SetPeerReview(ctx, review)

	// Update research review counts
	research.ReviewCount++
	switch msg.Verdict {
	case types.ReviewVerdictApprove:
		research.ApproveCount++
	case types.ReviewVerdictReject:
		research.RejectCount++
	case types.ReviewVerdictRevise:
		research.ReviseCount++
	}

	// Transition to under_review after first review
	if research.Status == types.ResearchStatusSubmitted {
		research.Status = types.ResearchStatusUnderReview
	}

	// Update aggregate score (simple average of all review scores)
	reviews := m.Keeper.GetReviewsForResearch(ctx, msg.ResearchId)
	var totalScore uint32
	for _, r := range reviews {
		totalScore += r.Score
	}
	research.AggregateScore = totalScore / uint32(len(reviews))

	research.UpdatedAt = uint64(ctx.BlockHeight())
	m.Keeper.SetResearch(ctx, research)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"research_reviewed",
			sdk.NewAttribute("research_id", msg.ResearchId),
			sdk.NewAttribute("reviewer", msg.Reviewer),
			sdk.NewAttribute("verdict", string(msg.Verdict)),
			sdk.NewAttribute("score", fmt.Sprintf("%d", msg.Score)),
		),
	)

	return &types.MsgReviewResearchResponse{}, nil
}

// ResolveResearch resolves a research submission after sufficient reviews.
func (m msgServer) ResolveResearch(
	goCtx context.Context,
	msg *types.MsgResolveResearch,
) (*types.MsgResolveResearchResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Authority check
	if msg.Authority != m.Keeper.GetAuthority() {
		return nil, types.ErrUnauthorized
	}

	params := m.Keeper.GetParams(ctx)

	// Fetch research
	research, found := m.Keeper.GetResearch(ctx, msg.ResearchId)
	if !found {
		return nil, types.ErrSubmissionNotFound
	}

	if research.Status != types.ResearchStatusUnderReview {
		return nil, types.ErrResearchNotUnderReview
	}

	// Need minimum reviews
	if research.ReviewCount < params.MinReviewerCount {
		return nil, fmt.Errorf("%w: need %d, have %d", types.ErrInsufficientReviews, params.MinReviewerCount, research.ReviewCount)
	}

	// Determine outcome from aggregate score
	var outcome string
	if research.AggregateScore >= params.AcceptanceScoreThreshold {
		outcome = "accepted"
		research.Status = types.ResearchStatusAccepted

		// Return stake to submitter
		stakeInt := new(big.Int)
		stakeInt.SetString(research.Stake, 10)
		submitterAddr, _ := sdk.AccAddressFromBech32(research.Submitter)
		coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(stakeInt)))
		m.Keeper.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, submitterAddr, coins)
	} else {
		outcome = "rejected"
		research.Status = types.ResearchStatusRejected

		// Slash stake
		stakeInt := new(big.Int)
		stakeInt.SetString(research.Stake, 10)
		slashRate := new(big.Int).SetUint64(params.RejectionSlashBps)
		slashAmount := new(big.Int).Mul(stakeInt, slashRate)
		slashAmount.Div(slashAmount, new(big.Int).SetUint64(10000))

		// Burn slashed amount
		if slashAmount.Sign() > 0 {
			burnCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(slashAmount)))
			m.Keeper.bankKeeper.BurnCoins(ctx, types.ModuleName, burnCoins)
		}

		// Return remainder to submitter
		remainder := new(big.Int).Sub(stakeInt, slashAmount)
		if remainder.Sign() > 0 {
			submitterAddr, _ := sdk.AccAddressFromBech32(research.Submitter)
			returnCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(remainder)))
			m.Keeper.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, submitterAddr, returnCoins)
		}
	}

	research.UpdatedAt = uint64(ctx.BlockHeight())
	m.Keeper.SetResearch(ctx, research)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"research_resolved",
			sdk.NewAttribute("research_id", msg.ResearchId),
			sdk.NewAttribute("outcome", outcome),
			sdk.NewAttribute("aggregate_score", fmt.Sprintf("%d", research.AggregateScore)),
		),
	)

	return &types.MsgResolveResearchResponse{Outcome: outcome}, nil
}

// CreateBounty creates a new research bounty.
func (m msgServer) CreateBounty(
	goCtx context.Context,
	msg *types.MsgCreateBounty,
) (*types.MsgCreateBountyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	params := m.Keeper.GetParams(ctx)

	// Validate deadline
	if msg.DeadlineHeight <= uint64(ctx.BlockHeight())+params.BountyMinDeadlineBlocks {
		return nil, fmt.Errorf("%w: deadline must be at least %d blocks in the future",
			types.ErrDeadlineTooSoon, params.BountyMinDeadlineBlocks)
	}

	// Validate reward
	rewardInt := new(big.Int)
	rewardInt.SetString(msg.Reward, 10)
	maxReward := new(big.Int)
	maxReward.SetString(params.MaxBountyReward, 10)
	if rewardInt.Cmp(maxReward) > 0 {
		return nil, types.ErrExceedsMaxReward
	}

	// Lock reward from creator
	creatorAddr, _ := sdk.AccAddressFromBech32(msg.Creator)
	coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(rewardInt)))
	if err := m.Keeper.bankKeeper.SendCoinsFromAccountToModule(ctx, creatorAddr, types.ModuleName, coins); err != nil {
		return nil, fmt.Errorf("failed to lock bounty reward: %w", err)
	}

	bountyId := m.Keeper.nextBountyId(ctx)
	bounty := types.Bounty{
		Id:             bountyId,
		Creator:        msg.Creator,
		Title:          msg.Title,
		Description:    msg.Description,
		Domain:         msg.Domain,
		Reward:         msg.Reward,
		DeadlineHeight: msg.DeadlineHeight,
		Status:         types.BountyStatusOpen,
		CreatedAt:      uint64(ctx.BlockHeight()),
	}
	m.Keeper.SetBounty(ctx, bounty)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"bounty_created",
			sdk.NewAttribute("bounty_id", bountyId),
			sdk.NewAttribute("creator", msg.Creator),
			sdk.NewAttribute("reward", msg.Reward),
			sdk.NewAttribute("domain", msg.Domain),
		),
	)

	return &types.MsgCreateBountyResponse{BountyId: bountyId}, nil
}

// ClaimBounty claims an open bounty.
func (m msgServer) ClaimBounty(
	goCtx context.Context,
	msg *types.MsgClaimBounty,
) (*types.MsgClaimBountyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	bounty, found := m.Keeper.GetBounty(ctx, msg.BountyId)
	if !found {
		return nil, types.ErrBountyNotFound
	}

	if bounty.Status != types.BountyStatusOpen {
		return nil, types.ErrBountyNotOpen
	}

	// Check deadline
	if uint64(ctx.BlockHeight()) > bounty.DeadlineHeight {
		return nil, types.ErrBountyExpired
	}

	bounty.Status = types.BountyStatusClaimed
	bounty.ClaimedBy = msg.Claimer
	m.Keeper.SetBounty(ctx, bounty)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"bounty_claimed",
			sdk.NewAttribute("bounty_id", msg.BountyId),
			sdk.NewAttribute("claimer", msg.Claimer),
		),
	)

	return &types.MsgClaimBountyResponse{}, nil
}

// FulfillBounty marks a bounty as fulfilled and pays the reward.
func (m msgServer) FulfillBounty(
	goCtx context.Context,
	msg *types.MsgFulfillBounty,
) (*types.MsgFulfillBountyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Authority check
	if msg.Authority != m.Keeper.GetAuthority() {
		return nil, types.ErrUnauthorized
	}

	bounty, found := m.Keeper.GetBounty(ctx, msg.BountyId)
	if !found {
		return nil, types.ErrBountyNotFound
	}

	if bounty.Status != types.BountyStatusClaimed {
		return nil, types.ErrBountyNotClaimed
	}

	// Pay reward to claimer
	rewardInt := new(big.Int)
	rewardInt.SetString(bounty.Reward, 10)
	claimerAddr, _ := sdk.AccAddressFromBech32(bounty.ClaimedBy)
	coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(rewardInt)))
	if err := m.Keeper.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, claimerAddr, coins); err != nil {
		return nil, fmt.Errorf("failed to pay bounty reward: %w", err)
	}

	bounty.Status = types.BountyStatusFulfilled
	bounty.FulfilledBy = bounty.ClaimedBy
	m.Keeper.SetBounty(ctx, bounty)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"bounty_fulfilled",
			sdk.NewAttribute("bounty_id", msg.BountyId),
			sdk.NewAttribute("fulfilled_by", bounty.ClaimedBy),
			sdk.NewAttribute("reward", bounty.Reward),
		),
	)

	return &types.MsgFulfillBountyResponse{Reward: bounty.Reward}, nil
}

// FundResearch adds funds to the research treasury.
func (m msgServer) FundResearch(
	goCtx context.Context,
	msg *types.MsgFundResearch,
) (*types.MsgFundResearchResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	amountInt := new(big.Int)
	amountInt.SetString(msg.Amount, 10)

	// Transfer to module account
	funderAddr, _ := sdk.AccAddressFromBech32(msg.Funder)
	coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(amountInt)))
	if err := m.Keeper.bankKeeper.SendCoinsFromAccountToModule(ctx, funderAddr, types.ModuleName, coins); err != nil {
		return nil, fmt.Errorf("failed to fund research treasury: %w", err)
	}

	// Update treasury balance
	currentBalance := new(big.Int)
	currentBalance.SetString(m.Keeper.GetTreasuryBalance(ctx), 10)
	newBalance := new(big.Int).Add(currentBalance, amountInt)
	m.Keeper.SetTreasuryBalance(ctx, newBalance.String())

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"research_funded",
			sdk.NewAttribute("funder", msg.Funder),
			sdk.NewAttribute("amount", msg.Amount),
			sdk.NewAttribute("new_treasury_balance", newBalance.String()),
		),
	)

	return &types.MsgFundResearchResponse{TreasuryBalance: newBalance.String()}, nil
}

// UpdateParams updates module parameters.
func (m msgServer) UpdateParams(
	goCtx context.Context,
	msg *types.MsgUpdateParams,
) (*types.MsgUpdateParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if msg.Authority != m.Keeper.GetAuthority() {
		return nil, types.ErrUnauthorized
	}

	m.Keeper.SetParams(ctx, msg.Params)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"research_params_updated",
			sdk.NewAttribute("authority", msg.Authority),
		),
	)

	return &types.MsgUpdateParamsResponse{}, nil
}
