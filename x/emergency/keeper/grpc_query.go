package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/emergency/types"
)

type queryServer struct {
	Keeper
}

// NewQueryServerImpl returns an implementation of the QueryServer interface.
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

var _ types.QueryServer = queryServer{}

// Status returns the current emergency status.
func (q queryServer) Status(goCtx context.Context, _ *types.QueryStatusRequest) (*types.QueryStatusResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	return &types.QueryStatusResponse{
		Status:               q.GetEmergencyStatus(ctx),
		IsHalted:             q.IsHalted(ctx),
		ActiveHaltCeremonyId: q.GetActiveHaltCeremonyId(ctx),
	}, nil
}

// ActiveCeremony returns the currently active ceremony, if any.
func (q queryServer) ActiveCeremony(goCtx context.Context, _ *types.QueryActiveCeremonyRequest) (*types.QueryActiveCeremonyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	ceremony, found := q.GetActiveCeremony(ctx)
	if !found {
		return &types.QueryActiveCeremonyResponse{Found: false}, nil
	}
	return &types.QueryActiveCeremonyResponse{
		Ceremony: &ceremony,
		Found:    true,
	}, nil
}

// CompletedCeremonies returns completed (finalized/failed) ceremonies.
func (q queryServer) CompletedCeremonies(goCtx context.Context, req *types.QueryCompletedCeremoniesRequest) (*types.QueryCompletedCeremoniesResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	limit := req.Limit
	if limit == 0 || limit > 100 {
		limit = 100
	}

	var completed []types.EmergencyCeremony
	var total uint64

	q.IterateCeremonies(ctx, func(c types.EmergencyCeremony) bool {
		if c.Phase == types.PhaseFinalized || c.Phase == types.PhaseFailed {
			total++
			if total > uint64(req.Offset) && uint32(len(completed)) < limit {
				completed = append(completed, c)
			}
		}
		return false
	})

	return &types.QueryCompletedCeremoniesResponse{
		Ceremonies: completed,
		Total:      total,
	}, nil
}

// AuditLog returns the emergency audit log.
func (q queryServer) AuditLog(goCtx context.Context, req *types.QueryAuditLogRequest) (*types.QueryAuditLogResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	limit := req.Limit
	if limit == 0 || limit > 100 {
		limit = 100
	}

	entries := q.GetAuditLog(ctx)
	total := uint64(len(entries))

	// Apply offset and limit
	start := uint32(0)
	if req.Offset < uint32(len(entries)) {
		start = req.Offset
	} else {
		start = uint32(len(entries))
	}
	end := start + limit
	if end > uint32(len(entries)) {
		end = uint32(len(entries))
	}

	return &types.QueryAuditLogResponse{
		Entries: entries[start:end],
		Total:   total,
	}, nil
}
