package keeper

import (
	"encoding/json"
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
	accountKeeper        types.AccountKeeper
	bankKeeper           types.BankKeeper
	stakingKeeper        types.StakingKeeper
	qualificationKeeper  types.DomainQualificationKeeper
	captureDefenseKeeper types.CaptureDefenseKeeper
	vestingRewardsKeeper types.VestingRewardsKeeper
	evidenceMgmtKeeper   types.EvidenceMgmtKeeper
	ontologyKeeper       types.OntologyKeeper

	// Home keeper for epistemic activity notifications
	homeKeeper types.HomeKeeper

	// Autopoiesis keeper for adaptive multipliers
	autopoiesisKeeper types.AutopoiesisKeeper

	// Partnerships keeper for routing rewards through partnerships
	partnershipsKeeper types.PartnershipsKeeper

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
	qk types.DomainQualificationKeeper,
	cdk types.CaptureDefenseKeeper,
	vrk types.VestingRewardsKeeper,
	emk types.EvidenceMgmtKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:                  cdc,
		storeService:         storeService,
		memKey:               memKey,
		accountKeeper:        ak,
		bankKeeper:           bk,
		stakingKeeper:        sk,
		qualificationKeeper:  qk,
		captureDefenseKeeper: cdk,
		vestingRewardsKeeper: vrk,
		evidenceMgmtKeeper:   emk,
		authority:            authority,
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

// SetAutopoiesisKeeper sets the autopoiesis keeper after initialization (avoids circular deps).
func (k *Keeper) SetAutopoiesisKeeper(ak types.AutopoiesisKeeper) {
	k.autopoiesisKeeper = ak
}

// GetSlashMultiplier returns the current slash multiplier from autopoiesis (BPS, 1M = 1.0x).
// Returns 1M (no multiplier) if autopoiesis is not wired.
func (k Keeper) GetSlashMultiplier(ctx sdk.Context) uint64 {
	if k.autopoiesisKeeper != nil {
		return k.autopoiesisKeeper.GetMultiplier(ctx, "slash.system")
	}
	return 1_000_000
}

// SetHomeKeeper sets the home keeper after initialization (avoids circular deps).
// Enables epistemic activity notifications: verification results, claim submissions,
// and challenges flow to agent homes for patina score calculation.
func (k *Keeper) SetHomeKeeper(hk types.HomeKeeper) {
	k.homeKeeper = hk
}

// SetPartnershipsKeeper sets the partnerships keeper after initialization (avoids circular deps).
// Enables reward routing: verification, citation, and falsification rewards flow through
// active partnerships (22% common pot, 78% split per partnership terms).
func (k *Keeper) SetPartnershipsKeeper(pk types.PartnershipsKeeper) {
	k.partnershipsKeeper = pk
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
func (k Keeper) InitGenesis(ctx sdk.Context, genState *types.GenesisState) {
	if genState.Params != nil {
		k.SetParams(ctx, genState.Params)
	}
	// Initialize extended params with defaults if not already set
	k.SetExtendedParams(ctx, types.DefaultExtendedParams())
	for _, domain := range genState.Domains {
		if domain != nil {
			k.SetDomainInState(ctx, domain)
			// FARM-12: Bootstrap domain→stratum mapping from canonical strata
			if stratum, ok := types.BootstrapDomainStrata[domain.Name]; ok {
				k.SetDomainStratum(ctx, domain.Name, stratum)
			}
		}
	}
	for _, fact := range genState.Facts {
		if fact != nil {
			k.SetFact(ctx, fact)
		}
	}
	for _, claim := range genState.Claims {
		if claim != nil {
			k.SetClaim(ctx, claim)
		}
	}
	for _, ref := range genState.FactReferences {
		if ref != nil {
			k.SetFactReference(ctx, ref)
		}
	}
}

// ExportGenesis exports the module's complete state for chain upgrades and migrations.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	var facts []*types.Fact
	k.IterateFacts(ctx, func(f *types.Fact) bool {
		facts = append(facts, f)
		return false
	})

	var claims []*types.Claim
	k.IterateClaims(ctx, func(c *types.Claim) bool {
		claims = append(claims, c)
		return false
	})

	var domains []*types.OntologicalDomain
	k.IterateDomains(ctx, func(d *types.OntologicalDomain) bool {
		domains = append(domains, d)
		return false
	})

	// Collect all fact references by iterating each fact's outgoing refs
	var factRefs []*types.FactReference
	seen := make(map[string]bool)
	for _, fact := range facts {
		refs := k.GetFactReferences(ctx, fact.Id)
		for _, ref := range refs {
			refKey := ref.FromFactId + ":" + ref.ToFactId
			if !seen[refKey] {
				factRefs = append(factRefs, ref)
				seen[refKey] = true
			}
		}
	}

	params := k.GetParams(ctx)
	return &types.GenesisState{
		Params:         params,
		Domains:        domains,
		Facts:          facts,
		Claims:         claims,
		FactReferences: factRefs,
	}
}

// SetParams sets module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params *types.Params) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(params)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal params: %v", err))
	}
	if err := kvStore.Set(types.ParamsKey, bz); err != nil {
		panic(fmt.Sprintf("failed to set params: %v", err))
	}
}

// GetParams returns module parameters.
func (k Keeper) GetParams(ctx sdk.Context) *types.Params {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.ParamsKey)
	if err != nil || bz == nil {
		dp := types.DefaultParams()
		return &dp
	}
	var params types.Params
	if err := proto.Unmarshal(bz, &params); err != nil {
		dp := types.DefaultParams()
		return &dp
	}
	return &params
}

// SetExtendedParams sets extended module parameters (JSON-encoded).
func (k Keeper) SetExtendedParams(ctx sdk.Context, ep types.ExtendedParams) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(ep)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal extended params: %v", err))
	}
	if err := kvStore.Set(types.ExtendedParamsKey, bz); err != nil {
		panic(fmt.Sprintf("failed to set extended params: %v", err))
	}
}

// GetExtendedParams returns extended module parameters.
func (k Keeper) GetExtendedParams(ctx sdk.Context) types.ExtendedParams {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.ExtendedParamsKey)
	if err != nil || bz == nil {
		return types.DefaultExtendedParams()
	}
	var ep types.ExtendedParams
	if err := json.Unmarshal(bz, &ep); err != nil {
		return types.DefaultExtendedParams()
	}
	return ep
}

// IsAdversarialForDomain checks whether a domain uses adversarial verification mode.
// Per-domain overrides in ExtendedParams take precedence over the global toggle.
func (k Keeper) IsAdversarialForDomain(ctx sdk.Context, domain string) bool {
	params := k.GetParams(ctx)
	globalEnabled := params.AdversarialVerificationEnabled

	ep := k.GetExtendedParams(ctx)
	if ep.AdversarialDomainOverrides != nil {
		if override, ok := ep.AdversarialDomainOverrides[domain]; ok {
			return override
		}
	}

	return globalEnabled
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
func (k Keeper) SetDomainInState(ctx sdk.Context, domain *types.OntologicalDomain) {
	k.SetDomain(ctx, domain)
}

// GetDomainFromState retrieves a domain using state.go's GetDomain.
func (k Keeper) GetDomainFromState(ctx sdk.Context, name string) (*types.OntologicalDomain, bool) {
	return k.GetDomain(ctx, name)
}
