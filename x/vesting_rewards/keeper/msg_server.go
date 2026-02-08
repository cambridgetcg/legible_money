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
