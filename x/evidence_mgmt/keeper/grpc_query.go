package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/evidence_mgmt/types"
)

type queryServer struct {
	Keeper
	types.UnimplementedQueryServer
}

// NewQueryServerImpl returns an implementation of the QueryServer interface.
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

var _ types.QueryServer = queryServer{}

func (k queryServer) Evidence(goCtx context.Context, req *types.QueryEvidenceRequest) (*types.QueryEvidenceResponse, error) {
	if req == nil || req.EvidenceId == "" {
		return nil, fmt.Errorf("evidence_id is required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	evidence, found := k.GetEvidence(ctx, req.EvidenceId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrEvidenceNotFound, req.EvidenceId)
	}
	return &types.QueryEvidenceResponse{Evidence: &evidence}, nil
}

func (k queryServer) EvidenceByClaimId(goCtx context.Context, req *types.QueryEvidenceByClaimIdRequest) (*types.QueryEvidenceByClaimIdResponse, error) {
	if req == nil || req.ClaimId == "" {
		return nil, fmt.Errorf("claim_id is required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	evidences := k.GetEvidenceByClaimId(ctx, req.ClaimId)
	ptrs := make([]*types.Evidence, len(evidences))
	for i := range evidences {
		ptrs[i] = &evidences[i]
	}
	return &types.QueryEvidenceByClaimIdResponse{Evidences: ptrs}, nil
}

func (k queryServer) Audit(goCtx context.Context, req *types.QueryAuditRequest) (*types.QueryAuditResponse, error) {
	if req == nil || req.AuditId == "" {
		return nil, fmt.Errorf("audit_id is required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	audit, found := k.GetAudit(ctx, req.AuditId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrAuditNotFound, req.AuditId)
	}
	return &types.QueryAuditResponse{Audit: &audit}, nil
}

func (k queryServer) AuditsForEvidence(goCtx context.Context, req *types.QueryAuditsForEvidenceRequest) (*types.QueryAuditsForEvidenceResponse, error) {
	if req == nil || req.EvidenceId == "" {
		return nil, fmt.Errorf("evidence_id is required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	audits := k.GetAuditsForEvidence(ctx, req.EvidenceId)
	ptrs := make([]*types.EvidenceAudit, len(audits))
	for i := range audits {
		ptrs[i] = &audits[i]
	}
	return &types.QueryAuditsForEvidenceResponse{Audits: ptrs}, nil
}

func (k queryServer) Oracle(goCtx context.Context, req *types.QueryOracleRequest) (*types.QueryOracleResponse, error) {
	if req == nil || req.OracleId == "" {
		return nil, fmt.Errorf("oracle_id is required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	oracle, found := k.GetOracle(ctx, req.OracleId)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrOracleNotFound, req.OracleId)
	}
	return &types.QueryOracleResponse{Oracle: &oracle}, nil
}

func (k queryServer) ActiveOracles(goCtx context.Context, req *types.QueryActiveOraclesRequest) (*types.QueryActiveOraclesResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	oracles := k.GetActiveOracles(ctx)
	ptrs := make([]*types.Oracle, len(oracles))
	for i := range oracles {
		ptrs[i] = &oracles[i]
	}
	return &types.QueryActiveOraclesResponse{Oracles: ptrs}, nil
}

func (k queryServer) Consensus(goCtx context.Context, req *types.QueryConsensusRequest) (*types.QueryConsensusResponse, error) {
	if req == nil || req.EvidenceId == "" {
		return nil, fmt.Errorf("evidence_id is required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	consensus, found := k.GetConsensus(ctx, req.EvidenceId)
	if !found {
		return nil, fmt.Errorf("consensus not found for evidence %s", req.EvidenceId)
	}
	return &types.QueryConsensusResponse{Consensus: &consensus}, nil
}

func (k queryServer) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)
	return &types.QueryParamsResponse{Params: &params}, nil
}
