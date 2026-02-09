package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/capture_challenge/types"
)

var _ types.QueryServer = Keeper{}

// Challenge returns a challenge by ID.
func (k Keeper) Challenge(goCtx context.Context, req *types.QueryChallengeRequest) (*types.QueryChallengeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	ch, found := k.GetChallenge(ctx, req.ChallengeId)
	if !found {
		return &types.QueryChallengeResponse{Found: false}, nil
	}
	return &types.QueryChallengeResponse{
		Challenge: &ch,
		Found:     true,
	}, nil
}

// ActiveChallenges returns all active challenges, optionally filtered by domain.
func (k Keeper) ActiveChallenges(goCtx context.Context, req *types.QueryActiveChallengesRequest) (*types.QueryActiveChallengesResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	var active []types.CaptureChallenge
	k.IterateChallenges(ctx, func(ch types.CaptureChallenge) bool {
		// Filter by domain if specified
		if req.Domain != "" && ch.Domain != req.Domain {
			return false
		}
		// Only active statuses
		switch ch.Status {
		case types.ChallengeSubmitted, types.ChallengeEvidencePeriod, types.ChallengeUnderReview:
			active = append(active, ch)
		}
		return false
	})

	return &types.QueryActiveChallengesResponse{Challenges: active}, nil
}

// BountyPool returns the bounty pool for a domain.
func (k Keeper) BountyPool(goCtx context.Context, req *types.QueryBountyPoolRequest) (*types.QueryBountyPoolResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	pool, found := k.GetBountyPool(ctx, req.Domain)
	if !found {
		return &types.QueryBountyPoolResponse{Found: false}, nil
	}
	return &types.QueryBountyPoolResponse{
		Pool:  &pool,
		Found: true,
	}, nil
}
