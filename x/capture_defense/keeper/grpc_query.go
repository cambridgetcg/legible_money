package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/capture_defense/types"
)

var _ types.QueryServer = Keeper{}

// Qualification returns a validator's qualification for a domain.
func (k Keeper) Qualification(goCtx context.Context, req *types.QueryQualificationRequest) (*types.QueryQualificationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	qual, found := k.GetQualification(ctx, req.ValidatorAddr, req.Domain)
	if !found {
		return &types.QueryQualificationResponse{Found: false}, nil
	}
	return &types.QueryQualificationResponse{
		Qualification: &qual,
		Found:         true,
	}, nil
}

// ValidatorQualifications returns all qualifications for a validator.
func (k Keeper) ValidatorQualifications(goCtx context.Context, req *types.QueryValidatorQualificationsRequest) (*types.QueryValidatorQualificationsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	quals := k.GetValidatorQualifications(ctx, req.ValidatorAddr)
	qualPtrs := make([]*types.DomainQualification, len(quals))
	for i := range quals {
		qualPtrs[i] = &quals[i]
	}
	return &types.QueryValidatorQualificationsResponse{
		Qualifications: qualPtrs,
	}, nil
}

// DomainReputation returns the effective reputation for a validator in a domain.
func (k Keeper) DomainReputation(goCtx context.Context, req *types.QueryDomainReputationRequest) (*types.QueryDomainReputationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	result := k.GetEffectiveReputation(ctx, req.ValidatorAddr, req.Domain)
	return &types.QueryDomainReputationResponse{
		Reputation: &result,
	}, nil
}

// CaptureRisk returns the capture risk metrics for a domain.
func (k Keeper) CaptureRisk(goCtx context.Context, req *types.QueryCaptureRiskRequest) (*types.QueryCaptureRiskResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	metrics, found := k.GetCaptureMetrics(ctx, req.Domain)
	if !found {
		return &types.QueryCaptureRiskResponse{Found: false}, nil
	}
	return &types.QueryCaptureRiskResponse{
		Metrics: &metrics,
		Found:   true,
	}, nil
}
