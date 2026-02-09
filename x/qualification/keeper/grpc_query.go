package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/qualification/types"
)

var _ types.QueryServer = queryServer{}

type queryServer struct {
	Keeper
}

// NewQueryServerImpl returns a QueryServer implementation for the qualification module.
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

func (qs queryServer) Qualification(goCtx context.Context, req *types.QueryQualificationRequest) (*types.QueryQualificationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	q, found := qs.Keeper.GetQualification(ctx, req.ValidatorAddr, req.Domain)
	return &types.QueryQualificationResponse{
		Qualification: q,
		Found:         found,
	}, nil
}

func (qs queryServer) QualificationsByValidator(goCtx context.Context, req *types.QueryQualificationsByValidatorRequest) (*types.QueryQualificationsByValidatorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	qualifications := qs.Keeper.GetQualificationsByValidator(ctx, req.ValidatorAddr)
	if qualifications == nil {
		qualifications = []types.DomainQualification{}
	}
	return &types.QueryQualificationsByValidatorResponse{
		Qualifications: qualifications,
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
	e, found := qs.Keeper.GetEndorsement(ctx, req.EndorsementId)
	return &types.QueryEndorsementResponse{
		Endorsement: e,
		Found:       found,
	}, nil
}

func (qs queryServer) EndorsementsForValidator(goCtx context.Context, req *types.QueryEndorsementsForValidatorRequest) (*types.QueryEndorsementsForValidatorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	endorsements := qs.Keeper.GetEndorsementsForValidator(ctx, req.TargetValidator)
	if endorsements == nil {
		endorsements = []types.QualificationEndorsement{}
	}
	return &types.QueryEndorsementsForValidatorResponse{
		Endorsements: endorsements,
	}, nil
}

func (qs queryServer) Params(goCtx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := qs.Keeper.GetParams(ctx)
	return &types.QueryParamsResponse{
		Params: params,
	}, nil
}
