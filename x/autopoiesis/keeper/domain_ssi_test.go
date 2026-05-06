package keeper_test

import (
	"testing"

	"github.com/legible-money/legible/x/autopoiesis/types"
)

// TestDomainSSI_HealthyDomain verifies that a healthy domain produces a low SSI.
func TestDomainSSI_HealthyDomain(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	dom := types.DomainObservation{
		Domain:            "physics",
		FactsCreated:      50,
		FactsExpired:      2,
		TotalActiveFacts:  500,
		AvgInitialConfBps: 800_000, // 0.80
		AvgFinalConfBps:   780_000, // 0.78 — minor decline
		NewCitations:      10,
	}

	score := k.ComputeDomainSSI(ctx, dom)

	// Healthy domain: small confidence decline, low expiration, good participation.
	// Should be well below the moderate threshold (200_000).
	if score >= types.SSILowModerate {
		t.Errorf("healthy domain SSI too high: got %d, want < %d", score, types.SSILowModerate)
	}

	// Verify it was persisted
	stored := k.GetDomainSSI(ctx, "physics")
	if stored != score {
		t.Errorf("stored domain SSI mismatch: got %d, want %d", stored, score)
	}
}

// TestDomainSSI_TotalFailure verifies that a domain with 100% verification failure
// (confidence drops to zero) produces a high SSI.
func TestDomainSSI_TotalFailure(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	dom := types.DomainObservation{
		Domain:            "astrology",
		FactsCreated:      5,
		FactsExpired:      50, // massive expiration
		TotalActiveFacts:  100,
		AvgInitialConfBps: 900_000, // 0.90
		AvgFinalConfBps:   0,       // total collapse
		NewCitations:      0,
	}

	score := k.ComputeDomainSSI(ctx, dom)

	// Verification failure: 100% (0.40 weight → 400_000)
	// Expiration ratio: 50/5 = 10x, clamped to 1.0 (0.35 weight → 350_000)
	// Participation: 5 created, expected 1 (100/100), 5 > 1 so 0
	// Expected ~750_000 (high stress)
	if score < types.SSIModerate {
		t.Errorf("total failure domain SSI too low: got %d, want >= %d", score, types.SSIModerate)
	}
}

// TestDomainSSI_LowParticipation verifies that a domain with low participation
// (no new facts despite many active) produces elevated SSI.
func TestDomainSSI_LowParticipation(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	dom := types.DomainObservation{
		Domain:            "alchemy",
		FactsCreated:      0, // no new facts
		FactsExpired:      0, // nothing expiring either
		TotalActiveFacts:  1000,
		AvgInitialConfBps: 800_000,
		AvgFinalConfBps:   800_000, // no decline
		NewCitations:      0,
	}

	score := k.ComputeDomainSSI(ctx, dom)

	// Verification failure: 0 (no decline)
	// Expiration ratio: 0/0 = 0 (no facts created, no expired)
	// Non-participation: expected 10 (1000/100), got 0 → 100% → 250_000
	// Expected ~250_000 (elevated but not critical)
	if score < 200_000 {
		t.Errorf("low participation domain SSI too low: got %d, want >= 200000", score)
	}
	if score >= types.SSIModerate {
		t.Errorf("low participation domain SSI too high: got %d, want < %d (only participation signal)", score, types.SSIModerate)
	}
}

// TestDomainSSI_CaptureDetection verifies that CheckDomainCapture flags domains
// where domain SSI exceeds 2x the global SSI.
func TestDomainSSI_CaptureDetection(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	globalSSI := uint64(150_000) // 0.15 — healthy global state

	domainSSIs := []types.DomainSSIValue{
		{Domain: "physics", Value: 100_000},  // healthy — below threshold
		{Domain: "biology", Value: 180_000},  // slightly elevated but below 2x
		{Domain: "astrology", Value: 700_000}, // captured — way above 2x global
		{Domain: "alchemy", Value: 350_000},   // above 2x (300_000) and above min (200_000)
	}

	captured := k.CheckDomainCapture(ctx, globalSSI, domainSSIs)

	if len(captured) != 2 {
		t.Fatalf("expected 2 captured domains, got %d: %+v", len(captured), captured)
	}

	// Verify the right domains were flagged
	capturedNames := map[string]bool{}
	for _, c := range captured {
		capturedNames[c.Domain] = true
		if c.GlobalSSI != globalSSI {
			t.Errorf("captured domain %s has wrong global SSI: got %d, want %d", c.Domain, c.GlobalSSI, globalSSI)
		}
		if c.Ratio == 0 {
			t.Errorf("captured domain %s has zero ratio", c.Domain)
		}
	}
	if !capturedNames["astrology"] {
		t.Error("expected astrology to be flagged as captured")
	}
	if !capturedNames["alchemy"] {
		t.Error("expected alchemy to be flagged as captured")
	}
}

// TestDomainSSI_CaptureDetection_LowGlobalSSI verifies that when global SSI is very low,
// the minimum threshold prevents false positives.
func TestDomainSSI_CaptureDetection_LowGlobalSSI(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	globalSSI := uint64(10_000) // 0.01 — very healthy

	domainSSIs := []types.DomainSSIValue{
		{Domain: "physics", Value: 100_000}, // Above 2x (20_000) but below min threshold (200_000)
		{Domain: "biology", Value: 250_000}, // Above min threshold — should be flagged
	}

	captured := k.CheckDomainCapture(ctx, globalSSI, domainSSIs)

	if len(captured) != 1 {
		t.Fatalf("expected 1 captured domain (min threshold guards against false positives), got %d: %+v", len(captured), captured)
	}
	if captured[0].Domain != "biology" {
		t.Errorf("expected biology to be flagged, got %s", captured[0].Domain)
	}
}

// TestDomainSSI_Integration verifies that ComputeAllDomainSSI processes multiple
// domains and stores all results.
func TestDomainSSI_Integration(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	obs := types.EpochObservation{
		Epoch:       5,
		BlockHeight: 5555,
		Domains: []types.DomainObservation{
			{
				Domain:            "physics",
				FactsCreated:      20,
				FactsExpired:      1,
				TotalActiveFacts:  200,
				AvgInitialConfBps: 900_000,
				AvgFinalConfBps:   880_000,
			},
			{
				Domain:            "astrology",
				FactsCreated:      1,
				FactsExpired:      40,
				TotalActiveFacts:  50,
				AvgInitialConfBps: 800_000,
				AvgFinalConfBps:   100_000,
			},
		},
	}

	results := k.ComputeAllDomainSSI(ctx, obs)

	if len(results) != 2 {
		t.Fatalf("expected 2 domain SSI results, got %d", len(results))
	}

	// Physics should be healthy
	physicsSSI := k.GetDomainSSI(ctx, "physics")
	astrologySSI := k.GetDomainSSI(ctx, "astrology")

	if physicsSSI >= astrologySSI {
		t.Errorf("physics SSI (%d) should be lower than astrology SSI (%d)", physicsSSI, astrologySSI)
	}

	if astrologySSI < types.SSIModerate {
		t.Errorf("astrology SSI should be at least moderate (%d), got %d", types.SSIModerate, astrologySSI)
	}
}
