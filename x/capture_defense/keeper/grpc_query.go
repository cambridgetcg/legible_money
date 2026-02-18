package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/capture_defense/types"
)

type queryServer struct {
	Keeper
	types.UnimplementedQueryServer
}

// NewQueryServerImpl returns a QueryServer implementation.
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

var _ types.QueryServer = queryServer{}

// Qualification returns a validator's qualification for a domain.
func (k queryServer) Qualification(goCtx context.Context, req *types.QueryQualificationRequest) (*types.QueryQualificationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	qual, found := k.GetQualification(ctx, req.ValidatorAddr, req.Domain)
	if !found {
		return &types.QueryQualificationResponse{Found: false}, nil
	}
	return &types.QueryQualificationResponse{
		Qualification: qual,
		Found:         true,
	}, nil
}

// ValidatorQualifications returns all qualifications for a validator.
func (k queryServer) ValidatorQualifications(goCtx context.Context, req *types.QueryValidatorQualificationsRequest) (*types.QueryValidatorQualificationsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	quals := k.GetValidatorQualifications(ctx, req.ValidatorAddr)
	return &types.QueryValidatorQualificationsResponse{
		Qualifications: quals,
	}, nil
}

// DomainReputation returns the effective reputation for a validator in a domain.
func (k queryServer) DomainReputation(goCtx context.Context, req *types.QueryDomainReputationRequest) (*types.QueryDomainReputationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	result := k.GetEffectiveReputation(ctx, req.ValidatorAddr, req.Domain)
	return &types.QueryDomainReputationResponse{
		Reputation: &result,
	}, nil
}

// CaptureRisk returns the capture risk metrics for a domain.
func (k queryServer) CaptureRisk(goCtx context.Context, req *types.QueryCaptureRiskRequest) (*types.QueryCaptureRiskResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	metrics, found := k.GetCaptureMetrics(ctx, req.Domain)
	if !found {
		return &types.QueryCaptureRiskResponse{Found: false}, nil
	}
	return &types.QueryCaptureRiskResponse{
		Metrics: metrics,
		Found:   true,
	}, nil
}
