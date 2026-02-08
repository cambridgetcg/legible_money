package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/ontology/types"
)

// BankKeeper defines the expected bank keeper interface for the ontology module.
type BankKeeper interface {
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}

// Keeper manages the ontology module's state.
type Keeper struct {
	cdc      codec.Codec
	storeKey storetypes.StoreKey

	bankKeeper BankKeeper

	// Module authority (typically governance module address)
	authority string
}

// NewKeeper creates a new ontology module Keeper.
func NewKeeper(
	cdc codec.Codec,
	storeKey storetypes.StoreKey,
	bankKeeper BankKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:        cdc,
		storeKey:   storeKey,
		bankKeeper: bankKeeper,
		authority:  authority,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// GetAuthority returns the module authority address.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// InitGenesis initializes the module's state from genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) {
	k.SetParams(ctx, genState.Params)

	// Initialize strata
	for _, stratum := range genState.Strata {
		k.SetStratum(ctx, stratum)
	}

	// Initialize domains
	for _, domain := range genState.Domains {
		k.SetDomain(ctx, domain)
	}

	// Initialize proposals (if any from export)
	for _, proposal := range genState.Proposals {
		k.SetProposal(ctx, proposal)
	}

	// Initialize cross-stratum links
	for _, link := range genState.Links {
		k.SetLink(ctx, link)
	}
}

// ExportGenesis exports the module's state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{
		Params:    k.GetParams(ctx),
		Strata:    k.GetAllStrata(ctx),
		Domains:   k.GetAllDomains(ctx),
		Proposals: k.getAllProposals(ctx),
		Links:     k.GetAllLinks(ctx),
	}
}

// getAllProposals returns all proposals (used in ExportGenesis).
func (k Keeper) getAllProposals(ctx sdk.Context) []types.DomainProposal {
	var proposals []types.DomainProposal
	k.IterateProposals(ctx, func(p types.DomainProposal) bool {
		proposals = append(proposals, p)
		return false
	})
	return proposals
}
