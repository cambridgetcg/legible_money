package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/home/types"
)

type queryServer struct {
	Keeper
}

// NewQueryServerImpl returns a QueryServer implementation.
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

var _ types.QueryServer = queryServer{}

// Home returns a single agent home by DID or home ID.
func (q queryServer) Home(goCtx context.Context, req *types.QueryHomeRequest) (*types.QueryHomeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	home, found := q.Keeper.GetHome(ctx, req.DID)
	if !found {
		return &types.QueryHomeResponse{Home: nil, Found: false}, nil
	}

	return &types.QueryHomeResponse{Home: &home, Found: true}, nil
}

// Homes returns a list of homes, optionally filtered by owner or status.
func (q queryServer) Homes(goCtx context.Context, req *types.QueryHomesRequest) (*types.QueryHomesResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	var all []types.AgentHome
	q.Keeper.IterateHomes(ctx, func(home types.AgentHome) bool {
		if req.Owner != "" && home.OwnerAddress != req.Owner {
			return false
		}
		if req.Status != "" && string(home.Status) != req.Status {
			return false
		}
		all = append(all, home)
		return false
	})

	total := uint64(len(all))

	// Apply pagination
	offset := uint32(0)
	limit := uint32(100)
	if req.Offset > 0 {
		offset = req.Offset
	}
	if req.Limit > 0 {
		limit = req.Limit
	}

	start := int(offset)
	end := start + int(limit)
	if start > len(all) {
		start = len(all)
	}
	if end > len(all) {
		end = len(all)
	}

	return &types.QueryHomesResponse{
		Homes: all[start:end],
		Total: total,
	}, nil
}

// Sessions returns all active sessions for a home.
func (q queryServer) Sessions(goCtx context.Context, req *types.QuerySessionsRequest) (*types.QuerySessionsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	var sessions []types.ActiveSession
	q.Keeper.IterateSessions(ctx, req.HomeId, func(session types.ActiveSession) bool {
		sessions = append(sessions, session)
		return false
	})

	return &types.QuerySessionsResponse{Sessions: sessions}, nil
}

// Alerts returns alerts for a home, optionally filtered by acknowledged status.
func (q queryServer) Alerts(goCtx context.Context, req *types.QueryAlertsRequest) (*types.QueryAlertsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	var alerts []types.Alert
	q.Keeper.IterateAlerts(ctx, req.HomeId, func(alert types.Alert) bool {
		if req.Acknowledged != nil && alert.Acknowledged != *req.Acknowledged {
			return false
		}
		alerts = append(alerts, alert)
		return false
	})

	return &types.QueryAlertsResponse{Alerts: alerts}, nil
}

// Params returns the module parameters.
func (q queryServer) Params(goCtx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	return &types.QueryParamsResponse{Params: q.Keeper.GetParams(ctx)}, nil
}
