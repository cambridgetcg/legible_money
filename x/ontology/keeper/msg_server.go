package keeper

import (
	"context"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/ontology/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// ProposeDomain handles MsgProposeDomain — submits a new domain proposal.
func (k msgServer) ProposeDomain(goCtx context.Context, msg *types.MsgProposeDomain) (*types.MsgProposeDomainResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)

	// Validate stratum exists
	if _, found := k.GetStratum(ctx, msg.Stratum); !found {
		return nil, fmt.Errorf("%w: stratum %d not registered", types.ErrInvalidStratum, msg.Stratum)
	}

	// Check domain doesn't already exist
	if _, found := k.GetDomain(ctx, msg.Name); found {
		return nil, fmt.Errorf("%w: %s", types.ErrDomainExists, msg.Name)
	}

	// Validate and deduct stake
	stakeAmount := new(big.Int)
	if _, ok := stakeAmount.SetString(msg.Stake, 10); !ok || stakeAmount.Sign() <= 0 {
		return nil, fmt.Errorf("%w: invalid stake amount", types.ErrInsufficientStake)
	}

	minStake := new(big.Int)
	if _, ok := minStake.SetString(params.MinProposalStake, 10); !ok {
		minStake.SetInt64(1000000) // fallback default
	}
	if stakeAmount.Cmp(minStake) < 0 {
		return nil, fmt.Errorf("%w: stake %s below minimum %s",
			types.ErrInsufficientStake, msg.Stake, params.MinProposalStake)
	}

	proposerAddr, err := sdk.AccAddressFromBech32(msg.Proposer)
	if err != nil {
		return nil, fmt.Errorf("invalid proposer address: %w", err)
	}

	stakeCoin := sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(stakeAmount))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(
		ctx, proposerAddr, types.ModuleName, sdk.NewCoins(stakeCoin),
	); err != nil {
		return nil, fmt.Errorf("%w: %v", types.ErrInsufficientStake, err)
	}

	// Generate proposal
	height := uint64(ctx.BlockHeight())
	proposalID := GenerateProposalID(msg.Proposer, msg.Name, height)

	proposal := types.DomainProposal{
		Id: proposalID,
		Domain: types.Domain{
			Name:        msg.Name,
			DisplayName: msg.DisplayName,
			Description: msg.Description,
			Stratum:     msg.Stratum,
			Status:      "proposed",
			ProposedBy:  msg.Proposer,
		},
		Proposer:     msg.Proposer,
		ProposalType: "add",
		Stake:        msg.Stake,
		VotesFor:     0,
		VotesAgainst: 0,
		Voters:       []string{},
		Status:       "active",
		CreatedAt:    height,
		ExpiresAt:    height + params.ProposalVotingPeriod,
	}

	if err := k.CreateProposal(ctx, proposal); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"domain_proposed",
			sdk.NewAttribute("proposal_id", proposalID),
			sdk.NewAttribute("domain", msg.Name),
			sdk.NewAttribute("stratum", fmt.Sprintf("%d", msg.Stratum)),
			sdk.NewAttribute("proposer", msg.Proposer),
			sdk.NewAttribute("stake", msg.Stake),
		),
	)

	return &types.MsgProposeDomainResponse{ProposalId: proposalID}, nil
}

// VoteDomainProposal handles MsgVoteDomainProposal — votes on an existing proposal.
func (k msgServer) VoteDomainProposal(goCtx context.Context, msg *types.MsgVoteDomainProposal) (*types.MsgVoteDomainProposalResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := k.VoteProposal(ctx, msg.ProposalId, msg.Voter, msg.Approve); err != nil {
		return nil, err
	}

	return &types.MsgVoteDomainProposalResponse{}, nil
}

// UpdateDomain handles MsgUpdateDomain — authority-only domain metadata update.
func (k msgServer) UpdateDomain(goCtx context.Context, msg *types.MsgUpdateDomain) (*types.MsgUpdateDomainResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if k.GetAuthority() != msg.Authority {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	domain, found := k.GetDomain(ctx, msg.DomainName)
	if !found {
		return nil, fmt.Errorf("%w: %s", types.ErrDomainNotFound, msg.DomainName)
	}

	// Apply updates
	if msg.DisplayName != "" {
		domain.DisplayName = msg.DisplayName
	}
	if msg.Description != "" {
		domain.Description = msg.Description
	}
	if msg.Status != "" {
		domain.Status = msg.Status
	}
	domain.UpdatedAt = uint64(ctx.BlockHeight())

	k.SetDomain(ctx, domain)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"domain_updated",
			sdk.NewAttribute("domain", msg.DomainName),
			sdk.NewAttribute("authority", msg.Authority),
		),
	)

	return &types.MsgUpdateDomainResponse{}, nil
}

// UpdateParams handles MsgUpdateParams — authority-only parameter update.
func (k msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.Params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	k.SetParams(ctx, msg.Params)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"params_updated",
			sdk.NewAttribute("authority", msg.Authority),
		),
	)

	return &types.MsgUpdateParamsResponse{}, nil
}
