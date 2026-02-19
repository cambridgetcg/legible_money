package keeper_test

import (
	"testing"

	"github.com/legible-money/legible/x/knowledge/keeper"
	"github.com/legible-money/legible/x/knowledge/types"
)

// ---------- FARM-11: Citation Concentration Detection Tests ----------

func TestFARM11_ConcentrationDiscount_NoDiscount(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	factID := "fact-no-concentration"
	// 10 unique addresses each cite once → evenly distributed
	for i := 0; i < 10; i++ {
		addr := testSubmitterAddr + string(rune('a'+i))
		k.UpdateCitationSource(ctx, factID, addr)
	}

	ep := k.GetExtendedParams(ctx)
	discount := k.GetCitationConcentrationDiscount(ctx, factID, "anyone", ep)
	if discount != 0 {
		t.Errorf("expected no discount for evenly distributed citations, got %d", discount)
	}
}

func TestFARM11_ConcentrationDiscount_Triggered(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	factID := "fact-concentrated"
	// 2 sybil addresses cite 5 times each = 10 total, top-3 = 10/10 = 100% > 80%
	for i := 0; i < 5; i++ {
		k.UpdateCitationSource(ctx, factID, "lgm1sybil_a")
	}
	for i := 0; i < 5; i++ {
		k.UpdateCitationSource(ctx, factID, "lgm1sybil_b")
	}

	ep := k.GetExtendedParams(ctx)
	discount := k.GetCitationConcentrationDiscount(ctx, factID, "anyone", ep)
	if discount != ep.CitationConcentrationDiscount {
		t.Errorf("expected concentration discount %d, got %d", ep.CitationConcentrationDiscount, discount)
	}
}

func TestFARM11_ConcentrationDiscount_BelowMinCites(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	factID := "fact-too-few-cites"
	// 4 cites (below default MinCites=5)
	for i := 0; i < 4; i++ {
		k.UpdateCitationSource(ctx, factID, "lgm1sybil_a")
	}

	ep := k.GetExtendedParams(ctx)
	discount := k.GetCitationConcentrationDiscount(ctx, factID, "anyone", ep)
	if discount != 0 {
		t.Errorf("expected no discount for below-min-cites, got %d", discount)
	}
}

func TestFARM11_ConcentrationDiscount_BelowThreshold(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	factID := "fact-below-threshold"
	// 10 cites: 3 addrs with 3,2,2 = top-3 sum = 7 → 70% < 80% threshold
	for i := 0; i < 3; i++ {
		k.UpdateCitationSource(ctx, factID, "lgm1addr_a")
	}
	for i := 0; i < 2; i++ {
		k.UpdateCitationSource(ctx, factID, "lgm1addr_b")
	}
	for i := 0; i < 2; i++ {
		k.UpdateCitationSource(ctx, factID, "lgm1addr_c")
	}
	// 3 more unique addresses
	k.UpdateCitationSource(ctx, factID, "lgm1addr_d")
	k.UpdateCitationSource(ctx, factID, "lgm1addr_e")
	k.UpdateCitationSource(ctx, factID, "lgm1addr_f")

	ep := k.GetExtendedParams(ctx)
	discount := k.GetCitationConcentrationDiscount(ctx, factID, "anyone", ep)
	if discount != 0 {
		t.Errorf("expected no discount for below-threshold concentration, got %d", discount)
	}
}

func TestFARM11_CitationSourceRecord_CRUD(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	factID := "fact-crud-test"

	// Initially not found
	_, found := k.GetCitationSourceRecord(ctx, factID)
	if found {
		t.Fatal("expected record not found initially")
	}

	// Update creates record
	k.UpdateCitationSource(ctx, factID, "lgm1addr_a")
	record, found := k.GetCitationSourceRecord(ctx, factID)
	if !found {
		t.Fatal("expected record to be created after first update")
	}
	if record.TotalCites != 1 {
		t.Errorf("expected TotalCites=1, got %d", record.TotalCites)
	}
	if record.SourceCounts["lgm1addr_a"] != 1 {
		t.Errorf("expected addr_a count=1, got %d", record.SourceCounts["lgm1addr_a"])
	}

	// Second update increments
	k.UpdateCitationSource(ctx, factID, "lgm1addr_a")
	k.UpdateCitationSource(ctx, factID, "lgm1addr_b")
	record, _ = k.GetCitationSourceRecord(ctx, factID)
	if record.TotalCites != 3 {
		t.Errorf("expected TotalCites=3, got %d", record.TotalCites)
	}
	if record.SourceCounts["lgm1addr_a"] != 2 {
		t.Errorf("expected addr_a count=2, got %d", record.SourceCounts["lgm1addr_a"])
	}
	if record.SourceCounts["lgm1addr_b"] != 1 {
		t.Errorf("expected addr_b count=1, got %d", record.SourceCounts["lgm1addr_b"])
	}

	// Set overwrites
	record.TotalCites = 99
	k.SetCitationSourceRecord(ctx, record)
	record2, _ := k.GetCitationSourceRecord(ctx, factID)
	if record2.TotalCites != 99 {
		t.Errorf("expected TotalCites=99 after overwrite, got %d", record2.TotalCites)
	}
}

func TestFARM11_TopNSum(t *testing.T) {
	tests := []struct {
		name   string
		counts map[string]uint64
		n      uint64
		want   uint64
	}{
		{
			name:   "empty map",
			counts: map[string]uint64{},
			n:      3,
			want:   0,
		},
		{
			name:   "n=0",
			counts: map[string]uint64{"a": 5},
			n:      0,
			want:   0,
		},
		{
			name:   "n > len",
			counts: map[string]uint64{"a": 5, "b": 3},
			n:      5,
			want:   8,
		},
		{
			name:   "top 2 of 4",
			counts: map[string]uint64{"a": 10, "b": 5, "c": 3, "d": 1},
			n:      2,
			want:   15,
		},
		{
			name:   "top 3 of 5",
			counts: map[string]uint64{"a": 10, "b": 8, "c": 6, "d": 4, "e": 2},
			n:      3,
			want:   24,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := keeper.TopNSum(tt.counts, tt.n)
			if got != tt.want {
				t.Errorf("topNSum() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestFARM11_ConcentrationDiscount_DisabledParams(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	factID := "fact-disabled"
	for i := 0; i < 10; i++ {
		k.UpdateCitationSource(ctx, factID, "lgm1sybil")
	}

	ep := k.GetExtendedParams(ctx)
	// Disable by setting discount to 0
	ep.CitationConcentrationDiscount = 0
	discount := k.GetCitationConcentrationDiscount(ctx, factID, "anyone", ep)
	if discount != 0 {
		t.Errorf("expected no discount when feature disabled, got %d", discount)
	}
}

func TestFARM11_ValidateExtendedParams(t *testing.T) {
	// Valid defaults
	ep := types.DefaultExtendedParams()
	if err := ep.ValidateExtendedParams(); err != nil {
		t.Fatalf("default params should be valid: %v", err)
	}

	// Invalid threshold
	ep2 := types.DefaultExtendedParams()
	ep2.CitationConcentrationThreshold = 1_000_001
	if err := ep2.ValidateExtendedParams(); err == nil {
		t.Error("expected error for threshold > 1000000")
	}

	// Invalid discount
	ep3 := types.DefaultExtendedParams()
	ep3.CitationConcentrationDiscount = 1_000_001
	if err := ep3.ValidateExtendedParams(); err == nil {
		t.Error("expected error for discount > 1000000")
	}
}

// ---------- FARM-13: Prophetic Diversity Gate Tests ----------

func TestFARM13_PropheticDiversity_Allowed(t *testing.T) {
	// 10 citations from 10 unique addresses → prophetic should be allowed
	k, ctx, _ := setupKeeper(t)

	factID := "fact-prophetic-diverse"
	fact := &types.Fact{
		Id:             factID,
		Status:         types.FactStatusVerified,
		Confidence:     600_000,
		BaseConfidence: 630_000, // barely accepted (< 650k threshold)
		CitationCount:  10,
		CreatedAtHeight: 0,
	}
	k.SetFact(ctx, fact)

	// 10 unique addresses each cite once
	for i := 0; i < 10; i++ {
		addr := testSubmitterAddr + string(rune('a'+i))
		k.UpdateCitationSource(ctx, factID, addr)
	}

	ep := k.GetExtendedParams(ctx)
	// Reload fact from store
	storedFact, _ := k.GetFact(ctx, factID)
	result := k.ValidatePropheticDiversity(ctx, storedFact, ep)
	if !result {
		t.Error("expected prophetic diversity to pass with 10 unique citers")
	}
}

func TestFARM13_PropheticDiversity_TooFewCiters(t *testing.T) {
	// 10 citations from 2 addresses → denied (too few unique citers, default min=5)
	k, ctx, _ := setupKeeper(t)

	factID := "fact-prophetic-few-citers"
	fact := &types.Fact{
		Id:             factID,
		Status:         types.FactStatusVerified,
		Confidence:     600_000,
		BaseConfidence: 630_000,
		CitationCount:  10,
	}
	k.SetFact(ctx, fact)

	for i := 0; i < 5; i++ {
		k.UpdateCitationSource(ctx, factID, "lgm1sybil_a")
	}
	for i := 0; i < 5; i++ {
		k.UpdateCitationSource(ctx, factID, "lgm1sybil_b")
	}

	ep := k.GetExtendedParams(ctx)
	storedFact, _ := k.GetFact(ctx, factID)
	result := k.ValidatePropheticDiversity(ctx, storedFact, ep)
	if result {
		t.Error("expected prophetic diversity to fail with only 2 unique citers")
	}
}

func TestFARM13_PropheticDiversity_SingleDominant(t *testing.T) {
	// 10 citations from 5 addresses, one has 4 (40%) → denied (>30% single share)
	k, ctx, _ := setupKeeper(t)

	factID := "fact-prophetic-dominant"
	fact := &types.Fact{
		Id:             factID,
		Status:         types.FactStatusVerified,
		Confidence:     600_000,
		BaseConfidence: 630_000,
		CitationCount:  10,
	}
	k.SetFact(ctx, fact)

	// 4 from one address (40% of 10)
	for i := 0; i < 4; i++ {
		k.UpdateCitationSource(ctx, factID, "lgm1dominant")
	}
	// 6 spread across 4 more addresses
	k.UpdateCitationSource(ctx, factID, "lgm1addr_b")
	k.UpdateCitationSource(ctx, factID, "lgm1addr_b")
	k.UpdateCitationSource(ctx, factID, "lgm1addr_c")
	k.UpdateCitationSource(ctx, factID, "lgm1addr_c")
	k.UpdateCitationSource(ctx, factID, "lgm1addr_d")
	k.UpdateCitationSource(ctx, factID, "lgm1addr_e")

	ep := k.GetExtendedParams(ctx)
	storedFact, _ := k.GetFact(ctx, factID)
	result := k.ValidatePropheticDiversity(ctx, storedFact, ep)
	if result {
		t.Error("expected prophetic diversity to fail when one address has 40% (>30% threshold)")
	}
}

func TestFARM13_PropheticDiversity_AtThreshold(t *testing.T) {
	// 10 citations from 5 addresses, max per address = 3 (30%) → allowed (at threshold)
	k, ctx, _ := setupKeeper(t)

	factID := "fact-prophetic-threshold"
	fact := &types.Fact{
		Id:             factID,
		Status:         types.FactStatusVerified,
		Confidence:     600_000,
		BaseConfidence: 630_000,
		CitationCount:  10,
	}
	k.SetFact(ctx, fact)

	// 3 from one address (30% of 10 = exactly at threshold)
	for i := 0; i < 3; i++ {
		k.UpdateCitationSource(ctx, factID, "lgm1addr_a")
	}
	// 7 spread: 2+2+2+1
	k.UpdateCitationSource(ctx, factID, "lgm1addr_b")
	k.UpdateCitationSource(ctx, factID, "lgm1addr_b")
	k.UpdateCitationSource(ctx, factID, "lgm1addr_c")
	k.UpdateCitationSource(ctx, factID, "lgm1addr_c")
	k.UpdateCitationSource(ctx, factID, "lgm1addr_d")
	k.UpdateCitationSource(ctx, factID, "lgm1addr_d")
	k.UpdateCitationSource(ctx, factID, "lgm1addr_e")

	ep := k.GetExtendedParams(ctx)
	storedFact, _ := k.GetFact(ctx, factID)
	result := k.ValidatePropheticDiversity(ctx, storedFact, ep)
	if !result {
		t.Error("expected prophetic diversity to pass when max share is exactly 30%")
	}
}

func TestFARM13_PropheticDiversity_Disabled(t *testing.T) {
	// When PropheticMinUniqueCiters == 0, diversity check is disabled
	k, ctx, _ := setupKeeper(t)

	factID := "fact-prophetic-disabled"
	fact := &types.Fact{
		Id:             factID,
		Status:         types.FactStatusVerified,
		Confidence:     600_000,
		BaseConfidence: 630_000,
		CitationCount:  10,
	}
	k.SetFact(ctx, fact)

	// All from one address — normally would fail
	for i := 0; i < 10; i++ {
		k.UpdateCitationSource(ctx, factID, "lgm1monopolist")
	}

	ep := k.GetExtendedParams(ctx)
	ep.PropheticMinUniqueCiters = 0 // disable
	storedFact, _ := k.GetFact(ctx, factID)
	result := k.ValidatePropheticDiversity(ctx, storedFact, ep)
	if !result {
		t.Error("expected prophetic diversity to pass when check is disabled (min=0)")
	}
}

func TestFARM13_PropheticDiversity_NoRecord(t *testing.T) {
	// No citation source record → can't verify diversity → denied
	k, ctx, _ := setupKeeper(t)

	factID := "fact-prophetic-no-record"
	fact := &types.Fact{
		Id:             factID,
		Status:         types.FactStatusVerified,
		Confidence:     600_000,
		BaseConfidence: 630_000,
		CitationCount:  10,
	}
	k.SetFact(ctx, fact)

	ep := k.GetExtendedParams(ctx)
	storedFact, _ := k.GetFact(ctx, factID)
	result := k.ValidatePropheticDiversity(ctx, storedFact, ep)
	if result {
		t.Error("expected prophetic diversity to fail with no citation source record")
	}
}

func TestFARM13_PropheticRevocation(t *testing.T) {
	// Fact is prophetic, but diversity drops → should be revoked in ApplyPropheticGrowth
	k, ctx, _ := setupKeeper(t)

	factID := "fact-prophetic-revoke"
	fact := &types.Fact{
		Id:              factID,
		Status:          types.FactStatusVerified,
		Confidence:      700_000,
		BaseConfidence:  630_000,
		IsProphetic:     true,
		CitationCount:   10,
		CreatedAtHeight: 0,
	}
	k.SetFact(ctx, fact)

	// Only 2 citers → below min 5 → diversity fails
	for i := 0; i < 5; i++ {
		k.UpdateCitationSource(ctx, factID, "lgm1sybil_a")
	}
	for i := 0; i < 5; i++ {
		k.UpdateCitationSource(ctx, factID, "lgm1sybil_b")
	}

	// ApplyPropheticGrowth should revoke prophetic status
	k.ApplyPropheticGrowth(ctx, 10000)

	storedFact, found := k.GetFact(ctx, factID)
	if !found {
		t.Fatal("fact not found")
	}
	if storedFact.IsProphetic {
		t.Error("expected prophetic status to be revoked due to low diversity")
	}
}

func TestFARM13_ValidateParams(t *testing.T) {
	// Valid defaults
	ep := types.DefaultExtendedParams()
	if ep.PropheticMinUniqueCiters != 5 {
		t.Errorf("default PropheticMinUniqueCiters = %d, want 5", ep.PropheticMinUniqueCiters)
	}
	if ep.PropheticMaxSingleCiterShare != 300_000 {
		t.Errorf("default PropheticMaxSingleCiterShare = %d, want 300000", ep.PropheticMaxSingleCiterShare)
	}

	// Invalid max share
	ep2 := types.DefaultExtendedParams()
	ep2.PropheticMaxSingleCiterShare = 1_000_001
	if err := ep2.ValidateExtendedParams(); err == nil {
		t.Error("expected error for max_single_citer_share > 1000000")
	}
}
