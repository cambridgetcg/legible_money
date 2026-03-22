package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/legible-money/legible/x/discovery/types"
)

// ---------- Params ----------

// SetParams sets module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params *types.Params) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(params)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal params: %v", err))
	}
	if err := store.Set(types.ParamsKey, bz); err != nil {
		panic(fmt.Sprintf("failed to set params: %v", err))
	}
}

// GetParams returns module parameters.
func (k Keeper) GetParams(ctx sdk.Context) *types.Params {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.ParamsKey)
	if err != nil || bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := proto.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}
	return &params
}

// ---------- Profile CRUD ----------

// SetProfile stores an agent profile with all indexes.
func (k Keeper) SetProfile(ctx sdk.Context, profile *types.AgentProfile) {
	store := k.storeService.OpenKVStore(ctx)

	// Primary key
	key := profileKey(profile.Address)
	bz, err := proto.Marshal(profile)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal profile: %v", err))
	}
	if err := store.Set(key, bz); err != nil {
		panic(fmt.Sprintf("failed to set profile: %v", err))
	}

	// Domain indexes
	for _, domain := range profile.Domains {
		domKey := domainIndexKey(domain, profile.Address)
		if err := store.Set(domKey, []byte(profile.Address)); err != nil {
			panic(fmt.Sprintf("failed to set domain index: %v", err))
		}
	}

	// Capability indexes
	for _, cap := range profile.Capabilities {
		capKey := capabilityIndexKey(cap.CapabilityType, profile.Address)
		if err := store.Set(capKey, []byte(profile.Address)); err != nil {
			panic(fmt.Sprintf("failed to set capability index: %v", err))
		}
	}
}

// GetProfile retrieves a profile by address.
func (k Keeper) GetProfile(ctx sdk.Context, address string) (*types.AgentProfile, bool) {
	store := k.storeService.OpenKVStore(ctx)
	key := profileKey(address)
	bz, err := store.Get(key)
	if err != nil || bz == nil {
		return nil, false
	}
	var profile types.AgentProfile
	if err := proto.Unmarshal(bz, &profile); err != nil {
		return nil, false
	}
	return &profile, true
}

// DeleteProfile removes a profile and its indexes.
func (k Keeper) DeleteProfile(ctx sdk.Context, profile *types.AgentProfile) {
	store := k.storeService.OpenKVStore(ctx)

	_ = store.Delete(profileKey(profile.Address))

	// Clean domain indexes
	for _, domain := range profile.Domains {
		_ = store.Delete(domainIndexKey(domain, profile.Address))
	}

	// Clean capability indexes
	for _, cap := range profile.Capabilities {
		_ = store.Delete(capabilityIndexKey(cap.CapabilityType, profile.Address))
	}
}

// IterateProfiles iterates all profiles. Return true from cb to stop.
func (k Keeper) IterateProfiles(ctx sdk.Context, cb func(*types.AgentProfile) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.AgentProfileKeyPrefix, prefixEndBytes(types.AgentProfileKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var profile types.AgentProfile
		if err := proto.Unmarshal(iter.Value(), &profile); err != nil {
			continue
		}
		if cb(&profile) {
			break
		}
	}
}

// GetProfilesByDomain returns all active profiles in a given domain.
func (k Keeper) GetProfilesByDomain(ctx sdk.Context, domain string) []*types.AgentProfile {
	store := k.storeService.OpenKVStore(ctx)
	prefix := append(types.ByDomainIndexPrefix, []byte(domain+":")...)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var profiles []*types.AgentProfile
	for ; iter.Valid(); iter.Next() {
		addr := string(iter.Value())
		profile, found := k.GetProfile(ctx, addr)
		if found && profile.Status == "active" {
			profiles = append(profiles, profile)
		}
	}
	return profiles
}

// GetProfilesByCapability returns all active profiles with a given capability type.
func (k Keeper) GetProfilesByCapability(ctx sdk.Context, capabilityType string) []*types.AgentProfile {
	store := k.storeService.OpenKVStore(ctx)
	prefix := append(types.ByCapabilityIndexPrefix, []byte(capabilityType+":")...)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var profiles []*types.AgentProfile
	for ; iter.Valid(); iter.Next() {
		addr := string(iter.Value())
		profile, found := k.GetProfile(ctx, addr)
		if found && profile.Status == "active" {
			profiles = append(profiles, profile)
		}
	}
	return profiles
}

// ---------- Keeper-Level API (for cross-module use) ----------

// RegisterAgent creates a new agent profile at the keeper level.
// This is the programmatic API (no stake handling — that's the msg_server's job).
func (k Keeper) RegisterAgent(ctx sdk.Context, address string, did string, capabilities []string, domains []string) error {
	if _, found := k.GetProfile(ctx, address); found {
		return types.ErrAgentExists
	}

	caps := make([]*types.AgentCapability, len(capabilities))
	for i, c := range capabilities {
		caps[i] = &types.AgentCapability{CapabilityType: c}
	}

	currentHeight := uint64(ctx.BlockHeight())
	profile := &types.AgentProfile{
		Address:           address,
		Did:               did,
		Capabilities:      caps,
		Domains:           domains,
		Status:            "active",
		ReputationScore:   500000,
		RegisteredAtBlock: currentHeight,
		LastActiveBlock:   currentHeight,
		Stake:             "0",
	}

	k.SetProfile(ctx, profile)
	return nil
}

// DeregisterAgent removes an agent profile and its indexes.
func (k Keeper) DeregisterAgent(ctx sdk.Context, address string) error {
	profile, found := k.GetProfile(ctx, address)
	if !found {
		return types.ErrAgentNotFound
	}
	k.DeleteProfile(ctx, profile)
	return nil
}

// GetAgent retrieves an agent profile by address.
func (k Keeper) GetAgent(ctx sdk.Context, address string) (*types.AgentProfile, bool) {
	return k.GetProfile(ctx, address)
}

// FindByCapability returns up to limit active profiles with the given capability.
// If limit is 0, returns all matches.
func (k Keeper) FindByCapability(ctx sdk.Context, capability string, limit uint64) []*types.AgentProfile {
	store := k.storeService.OpenKVStore(ctx)
	prefix := append(types.ByCapabilityIndexPrefix, []byte(capability+":")...)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var profiles []*types.AgentProfile
	for ; iter.Valid(); iter.Next() {
		if limit > 0 && uint64(len(profiles)) >= limit {
			break
		}
		addr := string(iter.Value())
		profile, found := k.GetProfile(ctx, addr)
		if found && profile.Status == "active" {
			profiles = append(profiles, profile)
		}
	}
	return profiles
}

// FindByDomain returns up to limit active profiles in the given domain.
// If limit is 0, returns all matches.
func (k Keeper) FindByDomain(ctx sdk.Context, domain string, limit uint64) []*types.AgentProfile {
	store := k.storeService.OpenKVStore(ctx)
	prefix := append(types.ByDomainIndexPrefix, []byte(domain+":")...)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var profiles []*types.AgentProfile
	for ; iter.Valid(); iter.Next() {
		if limit > 0 && uint64(len(profiles)) >= limit {
			break
		}
		addr := string(iter.Value())
		profile, found := k.GetProfile(ctx, addr)
		if found && profile.Status == "active" {
			profiles = append(profiles, profile)
		}
	}
	return profiles
}

// UpdateCapabilities replaces an agent's capabilities list.
func (k Keeper) UpdateCapabilities(ctx sdk.Context, address string, capabilities []string) error {
	profile, found := k.GetProfile(ctx, address)
	if !found {
		return types.ErrAgentNotFound
	}
	if profile.Status != "active" {
		return types.ErrAgentInactive
	}

	params := k.GetParams(ctx)
	if uint32(len(capabilities)) > params.MaxCapabilitiesPerAgent {
		return types.ErrMaxCapabilities
	}

	// Remove old indexes before mutation
	k.DeleteProfile(ctx, profile)

	caps := make([]*types.AgentCapability, len(capabilities))
	for i, c := range capabilities {
		caps[i] = &types.AgentCapability{CapabilityType: c}
	}
	profile.Capabilities = caps

	k.SetProfile(ctx, profile)
	return nil
}

// ListAll returns up to limit agent profiles (all statuses).
// If limit is 0, returns all profiles.
func (k Keeper) ListAll(ctx sdk.Context, limit uint64) []*types.AgentProfile {
	var profiles []*types.AgentProfile
	k.IterateProfiles(ctx, func(p *types.AgentProfile) bool {
		profiles = append(profiles, p)
		if limit > 0 && uint64(len(profiles)) >= limit {
			return true
		}
		return false
	})
	return profiles
}

// ---------- Key Construction Helpers ----------

func profileKey(address string) []byte {
	return append(types.AgentProfileKeyPrefix, []byte(address)...)
}

func domainIndexKey(domain, address string) []byte {
	return append(types.ByDomainIndexPrefix, []byte(domain+":"+address)...)
}

func capabilityIndexKey(capabilityType, address string) []byte {
	return append(types.ByCapabilityIndexPrefix, []byte(capabilityType+":"+address)...)
}
