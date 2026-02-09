package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/qualification/types"
)

var _ types.MsgServer = msgServer{}

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns a MsgServer implementation for the qualification module.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

// RequestQualification handles qualification requests via any of the 4 pathways.
func (ms msgServer) RequestQualification(goCtx context.Context, msg *types.MsgRequestQualification) (*types.MsgRequestQualificationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	var q *types.DomainQualification
	var err error

	switch msg.Pathway {
	case types.PathwayStakeCommitment:
		if msg.Stake == "" {
			return nil, fmt.Errorf("stake amount is required for stake_commitment pathway")
		}
		q, err = ms.Keeper.QualifyViaStakeCommitment(ctx, msg.ValidatorAddr, msg.Domain, msg.Stake)

	case types.PathwayTrackRecord:
		q, err = ms.Keeper.QualifyViaTrackRecord(ctx, msg.ValidatorAddr, msg.Domain)

	case types.PathwayCrossReference:
		q, err = ms.Keeper.QualifyViaCrossReference(ctx, msg.ValidatorAddr, msg.Domain)

	case types.PathwayStratumInheritance:
		if msg.InheritFrom == "" {
			return nil, fmt.Errorf("inherit_from is required for stratum_inheritance pathway")
		}
		q, err = ms.Keeper.QualifyViaStratumInheritance(ctx, msg.ValidatorAddr, msg.Domain, msg.InheritFrom)

	default:
		return nil, fmt.Errorf("unsupported pathway: %s", msg.Pathway)
	}

	if err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"qualification_granted",
		sdk.NewAttribute("validator", msg.ValidatorAddr),
		sdk.NewAttribute("domain", msg.Domain),
		sdk.NewAttribute("pathway", string(msg.Pathway)),
		sdk.NewAttribute("status", string(q.Status)),
	))

	return &types.MsgRequestQualificationResponse{
		Status: q.Status,
	}, nil
}

// EndorseValidator endorses another validator for a domain.
func (ms msgServer) EndorseValidator(goCtx context.Context, msg *types.MsgEndorseValidator) (*types.MsgEndorseValidatorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Verify endorser is qualified in the domain
	if !ms.Keeper.IsQualified(ctx, msg.EndorserAddr, msg.Domain) {
		return nil, fmt.Errorf("endorser is not qualified in domain %s", msg.Domain)
	}

	// Verify target validator exists
	if _, found := ms.Keeper.stakingKeeper.GetValidator(ctx, msg.TargetValidator); !found {
		return nil, types.ErrValidatorNotFound
	}

	params := ms.Keeper.GetParams(ctx)

	// Verify endorser stake meets minimum
	if msg.Stake != "" {
		required, ok := new(big.Int).SetString(params.EndorserStake, 10)
		if ok {
			provided, pOk := new(big.Int).SetString(msg.Stake, 10)
			if pOk && provided.Cmp(required) < 0 {
				return nil, fmt.Errorf("%w: need %s, provided %s", types.ErrInsufficientStake, params.EndorserStake, msg.Stake)
			}
		}
	}

	// Lock endorser's stake
	if k := ms.Keeper.bankKeeper; k != nil && msg.Stake != "" {
		addr, addrErr := sdk.AccAddressFromBech32(msg.EndorserAddr)
		if addrErr == nil {
			amount, ok := new(big.Int).SetString(msg.Stake, 10)
			if ok && amount.Sign() > 0 {
				coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(amount)))
				if sendErr := k.SendCoinsFromAccountToModule(ctx, addr, types.ModuleName, coins); sendErr != nil {
					return nil, fmt.Errorf("failed to lock endorser stake: %v", sendErr)
				}
			}
		}
	}

	currentBlock := uint64(ctx.BlockHeight())
	endorsementId := ms.Keeper.GetNextEndorsementId(ctx)

	endorsement := types.QualificationEndorsement{
		Id:              endorsementId,
		EndorserAddr:    msg.EndorserAddr,
		TargetValidator: msg.TargetValidator,
		Domain:          msg.Domain,
		Stake:           msg.Stake,
		EndorsedAtBlock: currentBlock,
		ExpiresAtBlock:  currentBlock + params.EndorsementDecayBlocks,
	}

	ms.Keeper.SetEndorsement(ctx, endorsement)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"validator_endorsed",
		sdk.NewAttribute("endorser", msg.EndorserAddr),
		sdk.NewAttribute("target", msg.TargetValidator),
		sdk.NewAttribute("domain", msg.Domain),
		sdk.NewAttribute("endorsement_id", endorsementId),
	))

	return &types.MsgEndorseValidatorResponse{
		EndorsementId: endorsementId,
	}, nil
}

// RenewQualification renews an expiring qualification.
func (ms msgServer) RenewQualification(goCtx context.Context, msg *types.MsgRenewQualification) (*types.MsgRenewQualificationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := ms.Keeper.RenewQualification(ctx, msg.ValidatorAddr, msg.Domain); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"qualification_renewed",
		sdk.NewAttribute("validator", msg.ValidatorAddr),
		sdk.NewAttribute("domain", msg.Domain),
	))

	return &types.MsgRenewQualificationResponse{}, nil
}

// WithdrawQualification voluntarily withdraws a qualification.
func (ms msgServer) WithdrawQualification(goCtx context.Context, msg *types.MsgWithdrawQualification) (*types.MsgWithdrawQualificationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := ms.Keeper.WithdrawQualification(ctx, msg.ValidatorAddr, msg.Domain); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"qualification_withdrawn",
		sdk.NewAttribute("validator", msg.ValidatorAddr),
		sdk.NewAttribute("domain", msg.Domain),
	))

	return &types.MsgWithdrawQualificationResponse{}, nil
}

// UpdateParams updates module parameters.
func (ms msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if msg.Authority != ms.Keeper.GetAuthority() {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", ms.Keeper.GetAuthority(), msg.Authority)
	}

	if err := msg.Params.Validate(); err != nil {
		return nil, err
	}

	ms.Keeper.SetParams(ctx, msg.Params)
	return &types.MsgUpdateParamsResponse{}, nil
}
