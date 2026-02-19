package keeper

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

// RecordPartnershipFactCreation updates partnership citation stats when a fact is created
// with a partnership attribution. Called from all fact creation paths.
func (k Keeper) RecordPartnershipFactCreation(ctx sdk.Context, fact *types.Fact) {
	if fact.PartnershipId == "" {
		return
	}

	stats := k.GetPartnershipCitationStats(ctx, fact.PartnershipId)
	stats.TotalFactsCreated++

	// Track unique domains
	domainFound := false
	for _, d := range stats.DomainsCovered {
		if d == fact.Domain {
			domainFound = true
			break
		}
	}
	if !domainFound {
		stats.DomainsCovered = append(stats.DomainsCovered, fact.Domain)
	}

	k.SetPartnershipCitationStats(ctx, stats)
}

// RecordPartnershipCitation increments the citation count for a fact's partnership.
// Called when any fact with a partnership_id gets cited.
func (k Keeper) RecordPartnershipCitation(ctx sdk.Context, factID string) {
	fact, found := k.GetFact(ctx, factID)
	if !found || fact.PartnershipId == "" {
		return
	}

	stats := k.GetPartnershipCitationStats(ctx, fact.PartnershipId)
	stats.TotalCitations++
	k.SetPartnershipCitationStats(ctx, stats)
}

// CreditPartnershipReputation credits reputation to both partners when a partnered
// fact earns reputation (via citation, verification, etc.). Both partners get equal
// reputation credit regardless of reward split.
func (k Keeper) CreditPartnershipReputation(ctx sdk.Context, fact *types.Fact, accepted bool) {
	if fact.PartnershipId == "" || k.partnershipsKeeper == nil {
		return
	}

	partnership, found := k.partnershipsKeeper.GetActivePartnershipForAddress(ctx, fact.Submitter)
	if !found || partnership.Id != fact.PartnershipId {
		return
	}

	// Credit calibration record to both partners (50/50 reputation)
	// The submitter already gets credit through normal UpdateSubmitterCalibration;
	// we credit the OTHER partner here.
	otherAddr := partnership.HumanAddr
	if fact.Submitter == partnership.HumanAddr {
		otherAddr = partnership.AgentAddr
	}
	k.UpdateSubmitterCalibration(ctx, otherAddr, accepted)

	// Notify home module for patina scoring of both partners
	if k.homeKeeper != nil {
		k.homeKeeper.RecordClaimSubmission(ctx, otherAddr)
	}
}

// ---------- Partnership Citation Stats Storage ----------

// GetPartnershipCitationStats retrieves stats for a partnership, returning empty stats if not found.
func (k Keeper) GetPartnershipCitationStats(ctx sdk.Context, partnershipID string) *types.PartnershipCitationStats {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := partnershipCitationStatsKey(partnershipID)
	bz, err := kvStore.Get(key)
	if err != nil || bz == nil {
		return &types.PartnershipCitationStats{
			PartnershipId: partnershipID,
		}
	}
	var stats types.PartnershipCitationStats
	if err := json.Unmarshal(bz, &stats); err != nil {
		return &types.PartnershipCitationStats{
			PartnershipId: partnershipID,
		}
	}
	return &stats
}

// SetPartnershipCitationStats stores partnership citation stats.
func (k Keeper) SetPartnershipCitationStats(ctx sdk.Context, stats *types.PartnershipCitationStats) {
	kvStore := k.storeService.OpenKVStore(ctx)
	key := partnershipCitationStatsKey(stats.PartnershipId)
	bz, err := json.Marshal(stats)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal partnership citation stats: %v", err))
	}
	if err := kvStore.Set(key, bz); err != nil {
		panic(fmt.Sprintf("failed to set partnership citation stats: %v", err))
	}
}

// GetPartnershipProfile returns a summary of a partnership's knowledge contributions.
func (k Keeper) GetPartnershipProfile(ctx sdk.Context, partnershipID string) *types.PartnershipCitationStats {
	return k.GetPartnershipCitationStats(ctx, partnershipID)
}

// ---------- Key Construction ----------

func partnershipCitationStatsKey(partnershipID string) []byte {
	return append(types.PartnershipCitationStatsPrefix, []byte(partnershipID)...)
}
