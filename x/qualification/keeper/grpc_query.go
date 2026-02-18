package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/qualification/types"
)

var _ types.QueryServer = &queryServer{}

type queryServer struct {
	types.UnimplementedQueryServer
	Keeper
}

// NewQueryServerImpl returns a QueryServer implementation for the qualification module.
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

func (qs queryServer) Qualification(goCtx context.Context, req *types.QueryQualificationRequest) (*types.QueryQualificationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	q, found := qs.Keeper.GetQualification(ctx, req.ValidatorAddr, req.Domain)
	if !found {
		return nil, fmt.Errorf("qualification not found for %s in %s", req.ValidatorAddr, req.Domain)
	}
	return &types.QueryQualificationResponse{
		Qualification: &q,
	}, nil
}

func (qs queryServer) QualificationsByValidator(goCtx context.Context, req *types.QueryQualificationsByValidatorRequest) (*types.QueryQualificationsByValidatorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	qualifications := qs.Keeper.GetQualificationsByValidator(ctx, req.ValidatorAddr)
	ptrs := make([]*types.DomainQualification, len(qualifications))
	for i := range qualifications {
		ptrs[i] = &qualifications[i]
	}
	return &types.QueryQualificationsByValidatorResponse{
		Qualifications: ptrs,
	}, nil
}

func (qs queryServer) QualifiedValidators(goCtx context.Context, req *types.QueryQualifiedValidatorsRequest) (*types.QueryQualifiedValidatorsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	validators := qs.Keeper.GetQualifiedValidators(ctx, req.Domain)
	if validators == nil {
		validators = []string{}
	}
	return &types.QueryQualifiedValidatorsResponse{
		Validators: validators,
	}, nil
}

func (qs queryServer) Endorsement(goCtx context.Context, req *types.QueryEndorsementRequest) (*types.QueryEndorsementResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	// Compose endorsement key from endorser + target + domain
	id := fmt.Sprintf("%s/%s/%s", req.Endorser, req.TargetValidator, req.Domain)
	e, found := qs.Keeper.GetEndorsement(ctx, id)
	if !found {
		return nil, fmt.Errorf("endorsement not found: %s", id)
	}
	return &types.QueryEndorsementResponse{
		Endorsement: &e,
	}, nil
}

func (qs queryServer) EndorsementsForValidator(goCtx context.Context, req *types.QueryEndorsementsForValidatorRequest) (*types.QueryEndorsementsForValidatorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	endorsements := qs.Keeper.GetEndorsementsForValidator(ctx, req.ValidatorAddr)
	ptrs := make([]*types.QualificationEndorsement, len(endorsements))
	for i := range endorsements {
		ptrs[i] = &endorsements[i]
	}
	return &types.QueryEndorsementsForValidatorResponse{
		Endorsements: ptrs,
	}, nil
}

func (qs queryServer) Params(goCtx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := qs.Keeper.GetParams(ctx)
	return &types.QueryParamsResponse{
		Params: &params,
	}, nil
}
