package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/vesting_rewards/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns a msg server implementation.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// ClaimVestedRewards claims available vested rewards for the sender.
func (m msgServer) ClaimVestedRewards(
	goCtx context.Context,
	msg *types.MsgClaimVestedRewards,
) (*types.MsgClaimVestedRewardsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	claimed, err := m.Keeper.ClaimRewards(ctx, msg.Sender, msg.VestingId)
	if err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"claim_vested_rewards",
			sdk.NewAttribute("sender", msg.Sender),
			sdk.NewAttribute("claimed_amount", claimed),
		),
	)

	scheduleCount := uint32(1)
	if msg.VestingId == "" {
		schedules := m.Keeper.GetVestingSchedulesByRecipient(ctx, msg.Sender)
		scheduleCount = uint32(len(schedules))
	}

	return &types.MsgClaimVestedRewardsResponse{
		ClaimedAmount: claimed,
		ScheduleCount: scheduleCount,
	}, nil
}

// FalsifyClaim triggers clawback for a falsified claim.
// Only callable by module authority (governance).
func (m msgServer) FalsifyClaim(
	goCtx context.Context,
	msg *types.MsgFalsifyClaim,
) (*types.MsgFalsifyClaimResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Only authority can falsify claims
	if msg.Authority != m.Keeper.GetAuthority() {
		return nil, types.ErrUnauthorized
	}

	record, err := m.Keeper.FalsifyClaim(ctx, msg.ClaimId, msg.Challenger)
	if err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"falsify_claim",
			sdk.NewAttribute("claim_id", msg.ClaimId),
			sdk.NewAttribute("challenger", msg.Challenger),
			sdk.NewAttribute("clawback_amount", record.ReleasedClawback),
			sdk.NewAttribute("challenger_reward", record.ChallengerReward),
			sdk.NewAttribute("reason", msg.Reason),
		),
	)

	return &types.MsgFalsifyClaimResponse{
		ClawbackAmount:   record.ReleasedClawback,
		ChallengerReward: record.ChallengerReward,
	}, nil
}

// CreateVesting creates a new truth-linked vesting schedule.
// Only callable by module authority (governance/knowledge module).
func (m msgServer) CreateVesting(
	goCtx context.Context,
	msg *types.MsgCreateVesting,
) (*types.MsgCreateVestingResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if msg.Authority != m.Keeper.GetAuthority() {
		return nil, types.ErrUnauthorized
	}

	schedule, err := m.Keeper.CreateVestingSchedule(
		ctx, msg.ClaimId, msg.FactId, msg.Recipient,
		msg.TotalAmount, msg.Category, msg.Source,
	)
	if err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"vesting_created",
			sdk.NewAttribute("vesting_id", schedule.Id),
			sdk.NewAttribute("claim_id", msg.ClaimId),
			sdk.NewAttribute("recipient", msg.Recipient),
			sdk.NewAttribute("amount", msg.TotalAmount),
			sdk.NewAttribute("category", string(msg.Category)),
		),
	)

	return &types.MsgCreateVestingResponse{VestingId: schedule.Id}, nil
}

// PauseVesting pauses a vesting schedule due to active challenge.
// Only callable by module authority (governance/dispute module).
func (m msgServer) PauseVesting(
	goCtx context.Context,
	msg *types.MsgPauseVesting,
) (*types.MsgPauseVestingResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if msg.Authority != m.Keeper.GetAuthority() {
		return nil, types.ErrUnauthorized
	}

	if err := m.Keeper.PauseVesting(ctx, msg.VestingId); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"vesting_paused",
			sdk.NewAttribute("vesting_id", msg.VestingId),
			sdk.NewAttribute("reason", msg.Reason),
		),
	)

	return &types.MsgPauseVestingResponse{}, nil
}

// ResumeVesting resumes a paused vesting schedule.
// Only callable by module authority (governance/dispute module).
func (m msgServer) ResumeVesting(
	goCtx context.Context,
	msg *types.MsgResumeVesting,
) (*types.MsgResumeVestingResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if msg.Authority != m.Keeper.GetAuthority() {
		return nil, types.ErrUnauthorized
	}

	if err := m.Keeper.ResumeVesting(ctx, msg.VestingId); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"vesting_resumed",
			sdk.NewAttribute("vesting_id", msg.VestingId),
		),
	)

	return &types.MsgResumeVestingResponse{}, nil
}

// AccelerateVesting records evidence that accelerates a vesting schedule.
// Only callable by module authority (knowledge/dispute module).
func (m msgServer) AccelerateVesting(
	goCtx context.Context,
	msg *types.MsgAccelerateVesting,
) (*types.MsgAccelerateVestingResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if msg.Authority != m.Keeper.GetAuthority() {
		return nil, types.ErrUnauthorized
	}

	switch msg.AccelerationType {
	case "defense":
		if err := m.Keeper.RecordDefense(ctx, msg.VestingId); err != nil {
			return nil, err
		}
	case "replication":
		if err := m.Keeper.RecordReplication(ctx, msg.VestingId); err != nil {
			return nil, err
		}
	default:
		return nil, types.ErrInvalidAccelerationType
	}

	// Re-read schedule to return updated counts
	schedule, found := m.Keeper.GetVestingSchedule(ctx, msg.VestingId)
	if !found {
		return nil, types.ErrScheduleNotFound
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"vesting_accelerated",
			sdk.NewAttribute("vesting_id", msg.VestingId),
			sdk.NewAttribute("acceleration_type", msg.AccelerationType),
			sdk.NewAttribute("evidence", msg.EvidenceReference),
		),
	)

	return &types.MsgAccelerateVestingResponse{
		DefenseCount:     schedule.DefenseCount,
		ReplicationCount: schedule.ReplicationCount,
	}, nil
}

// CompleteVesting marks a vesting schedule as completed.
// Can be called by recipient (when max release reached) or authority.
func (m msgServer) CompleteVesting(
	goCtx context.Context,
	msg *types.MsgCompleteVesting,
) (*types.MsgCompleteVestingResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	schedule, found := m.Keeper.GetVestingSchedule(ctx, msg.VestingId)
	if !found {
		return nil, types.ErrScheduleNotFound
	}

	// Check authorization: must be recipient or module authority
	if msg.Sender != schedule.Recipient && msg.Sender != m.Keeper.GetAuthority() {
		return nil, types.ErrNotRecipientOrAuthority
	}

	if schedule.Status == types.VestingStatusCompleted {
		return nil, types.ErrScheduleAlreadyCompleted
	}

	if schedule.Status == types.VestingStatusFalsified || schedule.Status == types.VestingStatusAbandoned {
		return nil, types.ErrScheduleNotActive
	}

	// Update claimable before completing
	if err := m.Keeper.UpdateClaimableAmount(ctx, msg.VestingId); err != nil {
		return nil, err
	}

	// Re-read after update
	schedule, _ = m.Keeper.GetVestingSchedule(ctx, msg.VestingId)

	// Mark as completed
	schedule.Status = types.VestingStatusCompleted
	schedule.UpdatedAt = uint64(ctx.BlockHeight())
	m.Keeper.SetVestingSchedule(ctx, schedule)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"vesting_completed",
			sdk.NewAttribute("vesting_id", msg.VestingId),
			sdk.NewAttribute("released_amount", schedule.ReleasedAmount),
		),
	)

	return &types.MsgCompleteVestingResponse{
		FinalReleasedAmount: schedule.ReleasedAmount,
	}, nil
}
