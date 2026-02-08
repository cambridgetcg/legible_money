package keeper

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/ontology/types"
)

// ---------- Params ----------

// SetParams sets module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(params)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal params: %v", err))
	}
	store.Set(types.ParamsKey, bz)
}

// GetParams returns module parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := json.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}
	return params
}

// ---------- Stratum Operations ----------

// SetStratum stores stratum properties.
func (k Keeper) SetStratum(ctx sdk.Context, props types.StratumProperties) {
	store := ctx.KVStore(k.storeKey)
	key := stratumKey(props.Stratum)
	bz, err := json.Marshal(props)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal stratum: %v", err))
	}
	store.Set(key, bz)
}

// GetStratum retrieves stratum properties by level.
func (k Keeper) GetStratum(ctx sdk.Context, stratum types.Stratum) (types.StratumProperties, bool) {
	store := ctx.KVStore(k.storeKey)
	key := stratumKey(stratum)
	bz := store.Get(key)
	if bz == nil {
		return types.StratumProperties{}, false
	}
	var props types.StratumProperties
	if err := json.Unmarshal(bz, &props); err != nil {
		return types.StratumProperties{}, false
	}
	return props, true
}

// GetAllStrata returns all registered strata, sorted by stratum level.
func (k Keeper) GetAllStrata(ctx sdk.Context) []types.StratumProperties {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.StratumKeyPrefix)
	defer iter.Close()

	var strata []types.StratumProperties
	for ; iter.Valid(); iter.Next() {
		var props types.StratumProperties
		if err := json.Unmarshal(iter.Value(), &props); err != nil {
			continue
		}
		strata = append(strata, props)
	}
	return strata
}

// ---------- Domain Operations ----------

// SetDomain stores a domain.
func (k Keeper) SetDomain(ctx sdk.Context, domain types.Domain) {
	store := ctx.KVStore(k.storeKey)
	key := domainKey(domain.Name)
	bz, err := json.Marshal(domain)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal domain: %v", err))
	}
	store.Set(key, bz)

	// Maintain stratum->domain index
	indexKey := domainByStratumKey(domain.Stratum, domain.Name)
	store.Set(indexKey, []byte(domain.Name))
}

// GetDomain retrieves a domain by name.
func (k Keeper) GetDomain(ctx sdk.Context, name string) (types.Domain, bool) {
	store := ctx.KVStore(k.storeKey)
	key := domainKey(name)
	bz := store.Get(key)
	if bz == nil {
		return types.Domain{}, false
	}
	var domain types.Domain
	if err := json.Unmarshal(bz, &domain); err != nil {
		return types.Domain{}, false
	}
	return domain, true
}

// GetDomainsByStratum returns all domains belonging to a specific stratum.
func (k Keeper) GetDomainsByStratum(ctx sdk.Context, stratum types.Stratum) []types.Domain {
	store := ctx.KVStore(k.storeKey)
	prefix := domainByStratumPrefix(stratum)
	iter := storetypes.KVStorePrefixIterator(store, prefix)
	defer iter.Close()

	var domains []types.Domain
	for ; iter.Valid(); iter.Next() {
		domainName := string(iter.Value())
		domain, found := k.GetDomain(ctx, domainName)
		if found {
			domains = append(domains, domain)
		}
	}
	return domains
}

// GetAllDomains returns all registered domains.
func (k Keeper) GetAllDomains(ctx sdk.Context) []types.Domain {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.DomainKeyPrefix)
	defer iter.Close()

	var domains []types.Domain
	for ; iter.Valid(); iter.Next() {
		var domain types.Domain
		if err := json.Unmarshal(iter.Value(), &domain); err != nil {
			continue
		}
		domains = append(domains, domain)
	}
	return domains
}

// DeleteDomain removes a domain from the store.
func (k Keeper) DeleteDomain(ctx sdk.Context, name string) {
	domain, found := k.GetDomain(ctx, name)
	if !found {
		return
	}
	store := ctx.KVStore(k.storeKey)
	store.Delete(domainKey(name))
	store.Delete(domainByStratumKey(domain.Stratum, name))
}

// ---------- Proposal Operations ----------

// SetProposal stores a domain proposal.
func (k Keeper) SetProposal(ctx sdk.Context, proposal types.DomainProposal) {
	store := ctx.KVStore(k.storeKey)
	key := proposalKey(proposal.Id)
	bz, err := json.Marshal(proposal)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal proposal: %v", err))
	}
	store.Set(key, bz)
}

// GetProposal retrieves a proposal by ID.
func (k Keeper) GetProposal(ctx sdk.Context, id string) (types.DomainProposal, bool) {
	store := ctx.KVStore(k.storeKey)
	key := proposalKey(id)
	bz := store.Get(key)
	if bz == nil {
		return types.DomainProposal{}, false
	}
	var proposal types.DomainProposal
	if err := json.Unmarshal(bz, &proposal); err != nil {
		return types.DomainProposal{}, false
	}
	return proposal, true
}

// DeleteProposal removes a proposal from the store.
func (k Keeper) DeleteProposal(ctx sdk.Context, id string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(proposalKey(id))
}

// IterateProposals iterates over all proposals. Return true from cb to stop.
func (k Keeper) IterateProposals(ctx sdk.Context, cb func(types.DomainProposal) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.ProposalKeyPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var proposal types.DomainProposal
		if err := json.Unmarshal(iter.Value(), &proposal); err != nil {
			continue
		}
		if cb(proposal) {
			break
		}
	}
}

// ---------- Cross-Stratum Link Operations ----------

// SetLink stores a cross-stratum link.
func (k Keeper) SetLink(ctx sdk.Context, link types.CrossStratumLink) {
	store := ctx.KVStore(k.storeKey)
	key := linkKey(link.SourceDomain, link.TargetDomain)
	bz, err := json.Marshal(link)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal link: %v", err))
	}
	store.Set(key, bz)
}

// GetLink retrieves a cross-stratum link.
func (k Keeper) GetLink(ctx sdk.Context, source, target string) (types.CrossStratumLink, bool) {
	store := ctx.KVStore(k.storeKey)
	key := linkKey(source, target)
	bz := store.Get(key)
	if bz == nil {
		return types.CrossStratumLink{}, false
	}
	var link types.CrossStratumLink
	if err := json.Unmarshal(bz, &link); err != nil {
		return types.CrossStratumLink{}, false
	}
	return link, true
}

// GetAllLinks returns all cross-stratum links.
func (k Keeper) GetAllLinks(ctx sdk.Context) []types.CrossStratumLink {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.LinkKeyPrefix)
	defer iter.Close()

	var links []types.CrossStratumLink
	for ; iter.Valid(); iter.Next() {
		var link types.CrossStratumLink
		if err := json.Unmarshal(iter.Value(), &link); err != nil {
			continue
		}
		links = append(links, link)
	}
	return links
}

// ---------- Key Construction Helpers ----------

func stratumKey(stratum types.Stratum) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(stratum))
	return append(types.StratumKeyPrefix, buf...)
}

func domainKey(name string) []byte {
	return append(types.DomainKeyPrefix, []byte(name)...)
}

func proposalKey(id string) []byte {
	return append(types.ProposalKeyPrefix, []byte(id)...)
}

func linkKey(source, target string) []byte {
	return append(types.LinkKeyPrefix, []byte(source+"/"+target)...)
}

func domainByStratumPrefix(stratum types.Stratum) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(stratum))
	return append(types.DomainByStratumPrefix, buf...)
}

func domainByStratumKey(stratum types.Stratum, domainName string) []byte {
	prefix := domainByStratumPrefix(stratum)
	return append(prefix, []byte(domainName)...)
}
