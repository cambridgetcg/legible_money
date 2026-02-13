package keeper

import (
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/legible-money/legible/x/knowledge/types"
)

// Keeper manages the knowledge module's state.
type Keeper struct {
	cdc          codec.Codec
	storeService store.KVStoreService
	memKey       storetypes.StoreKey

	// Dependency keepers
	accountKeeper         types.AccountKeeper
	bankKeeper            types.BankKeeper
	stakingKeeper         types.StakingKeeper
	tokensKeeper          types.TokensKeeper
	qualificationKeeper   types.DomainQualificationKeeper
	captureDefenseKeeper  types.CaptureDefenseKeeper
	vestingRewardsKeeper  types.VestingRewardsKeeper
	evidenceMgmtKeeper   types.EvidenceMgmtKeeper
	ontologyKeeper       types.OntologyKeeper

	// Verification hooks (staking reputation, qualification metrics, etc.)
	verificationHooks types.VerificationHooks

	// Module authority (typically governance module address)
	authority string
}

// NewKeeper creates a new knowledge module Keeper.
func NewKeeper(
	cdc codec.Codec,
	storeService store.KVStoreService,
	memKey storetypes.StoreKey,
	ak types.AccountKeeper,
	bk types.BankKeeper,
	sk types.StakingKeeper,
	tk types.TokensKeeper,
	qk types.DomainQualificationKeeper,
	cdk types.CaptureDefenseKeeper,
	vrk types.VestingRewardsKeeper,
	emk types.EvidenceMgmtKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:                   cdc,
		storeService:          storeService,
		memKey:                memKey,
		accountKeeper:         ak,
		bankKeeper:            bk,
		stakingKeeper:         sk,
		tokensKeeper:          tk,
		qualificationKeeper:   qk,
		captureDefenseKeeper:  cdk,
		vestingRewardsKeeper:  vrk,
		evidenceMgmtKeeper:    emk,
		authority:             authority,
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

// GetStakingKeeper returns the staking keeper interface for external use.
func (k Keeper) GetStakingKeeper() types.StakingKeeper {
	return k.stakingKeeper
}

// SetOntologyKeeper sets the ontology keeper after initialization (avoids circular deps).
func (k *Keeper) SetOntologyKeeper(ok types.OntologyKeeper) {
	k.ontologyKeeper = ok
}

// GetOntologyKeeper returns the ontology keeper for cross-module use (e.g. evaluation engine).
func (k Keeper) GetOntologyKeeper() types.OntologyKeeper {
	return k.ontologyKeeper
}

// SetVerificationHooks sets the verification hooks (called after keeper construction
// to avoid circular dependencies). Follows the same pattern as SetOntologyKeeper.
func (k *Keeper) SetVerificationHooks(hooks types.VerificationHooks) {
	if hooks == nil {
		return
	}
	k.verificationHooks = hooks
}

// GetEvidenceMgmtKeeper returns the evidence management keeper for cross-module use.
func (k Keeper) GetEvidenceMgmtKeeper() types.EvidenceMgmtKeeper {
	return k.evidenceMgmtKeeper
}

// InitGenesis initializes the module's state from genesis.
func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) {
	if genState.Params != nil {
		k.SetParams(ctx, *genState.Params)
	}
	for _, domain := range genState.Domains {
		if domain != nil {
			k.SetDomainInState(ctx, *domain)
		}
	}
	for _, fact := range genState.Facts {
		if fact != nil {
			k.SetFact(ctx, *fact)
		}
	}
	for _, claim := range genState.Claims {
		if claim != nil {
			k.SetClaim(ctx, *claim)
		}
	}
	for _, ref := range genState.FactReferences {
		if ref != nil {
			k.SetFactReference(ctx, *ref)
		}
	}
}

// ExportGenesis exports the module's complete state for chain upgrades and migrations.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	var facts []*types.Fact
	k.IterateFacts(ctx, func(f types.Fact) bool {
		fc := f
		facts = append(facts, &fc)
		return false
	})

	var claims []*types.Claim
	k.IterateClaims(ctx, func(c types.Claim) bool {
		cc := c
		claims = append(claims, &cc)
		return false
	})

	var domains []*types.OntologicalDomain
	k.IterateDomains(ctx, func(d types.OntologicalDomain) bool {
		dc := d
		domains = append(domains, &dc)
		return false
	})

	// Collect all fact references by iterating each fact's outgoing refs
	var factRefs []*types.FactReference
	seen := make(map[string]bool)
	for _, fact := range facts {
		refs := k.GetFactReferences(ctx, fact.Id)
		for i, ref := range refs {
			refKey := ref.FromFactId + ":" + ref.ToFactId
			if !seen[refKey] {
				factRefs = append(factRefs, &refs[i])
				seen[refKey] = true
			}
		}
	}

	params := k.GetParams(ctx)
	return &types.GenesisState{
		Params:         &params,
		Domains:        domains,
		Facts:          facts,
		Claims:         claims,
		FactReferences: factRefs,
	}
}

// SetParams sets module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(&params)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal params: %v", err))
	}
	if err := kvStore.Set(types.ParamsKey, bz); err != nil {
		panic(fmt.Sprintf("failed to set params: %v", err))
	}
}

// GetParams returns module parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.ParamsKey)
	if err != nil || bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := proto.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}
	return params
}

// prefixEndBytes returns the end key for prefix iteration.
// It increments the last byte of the prefix, handling overflow.
func prefixEndBytes(prefix []byte) []byte {
	if len(prefix) == 0 {
		return nil
	}
	end := make([]byte, len(prefix))
	copy(end, prefix)
	for i := len(end) - 1; i >= 0; i-- {
		end[i]++
		if end[i] != 0 {
			return end
		}
	}
	return nil
}

// SetDomainInState stores a domain using state.go's SetDomain.
func (k Keeper) SetDomainInState(ctx sdk.Context, domain types.OntologicalDomain) {
	k.SetDomain(ctx, domain)
}

// GetDomainFromState retrieves a domain using state.go's GetDomain.
func (k Keeper) GetDomainFromState(ctx sdk.Context, name string) (types.OntologicalDomain, bool) {
	return k.GetDomain(ctx, name)
}
