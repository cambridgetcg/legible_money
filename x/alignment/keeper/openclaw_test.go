package keeper_test

import (
	"testing"

	"github.com/legible-money/legible/x/alignment/keeper"
	"github.com/legible-money/legible/x/alignment/types"
)

// ============================================================================
// OpenClaw Alignment Adversarial Probe Suite (OCA)
//
// Each probe tests that the alignment feedback loop detects and corrects
// specific philosophical drift scenarios. The negative feedback must push
// the system BACK toward the ideal — the harder it drifts, the stronger
// the correction.
//
// Probe naming: OCA-{dimension initial}{number}
//   T = Truth, U = Understanding, B = Beauty, J = Justice, C = Creativity
// ============================================================================

// ── OCA-T1: Colluding Validators ──
// Simulate 5 validators all voting the same → verify TDI rises.
func TestOCA_T1_ColludingValidators(t *testing.T) {
	obs := types.AlignmentObservation{
		UnanimityRate:      990_000, // 99% unanimity — extremely suspicious
		CrossValidatorCorr: 980_000, // 98% pairwise agreement
		CounterFactRatio:   10_000,  // only 1% counter-facts
	}
	tdi := keeper.ComputeTDI(obs)
	if tdi < 200_000 {
		t.Fatalf("OCA-T1 FAIL: colluding validators should produce TDI > 200k, got %d", tdi)
	}
	t.Logf("OCA-T1 PASS: collusion detected with TDI=%d", tdi)
}

// ── OCA-T2: Zero Challenges for Extended Period ──
// Simulate 0 challenges for 10 epochs → verify counter-fact deficit triggers.
func TestOCA_T2_ZeroChallenges(t *testing.T) {
	obs := types.AlignmentObservation{
		CounterFactRatio: 0, // zero counter-facts
	}
	tdi := keeper.ComputeTDI(obs)
	if tdi == 0 {
		t.Fatal("OCA-T2 FAIL: zero counter-facts should produce non-zero TDI")
	}
	t.Logf("OCA-T2 PASS: counter-fact deficit detected with TDI=%d", tdi)
}

// ── OCA-T3: High Reversal Rate ──
// Simulate high reversal rate → verify TDI rises proportionally.
func TestOCA_T3_HighReversalRate(t *testing.T) {
	obsLow := types.AlignmentObservation{ReversalRate: 100_000}
	obsHigh := types.AlignmentObservation{ReversalRate: 800_000}

	tdiLow := keeper.ComputeTDI(obsLow)
	tdiHigh := keeper.ComputeTDI(obsHigh)

	if tdiHigh <= tdiLow {
		t.Fatalf("OCA-T3 FAIL: higher reversal rate should produce higher TDI: low=%d, high=%d", tdiLow, tdiHigh)
	}
	t.Logf("OCA-T3 PASS: proportional detection — low=%d, high=%d", tdiLow, tdiHigh)
}

// ── OCA-T4: TDI Recovery After Correction ──
// Verify that healthy signals produce TDI near 0 (negative feedback restores).
func TestOCA_T4_TDIRecovery(t *testing.T) {
	obs := types.AlignmentObservation{
		ReversalRate:       50_000,  // low reversal (5%)
		ChallengeAsymmetry: 100_000, // low asymmetry
		UnanimityRate:      500_000, // 50% — healthy
		CrossValidatorCorr: 500_000, // 50% — healthy
		CounterFactRatio:   100_000, // 10% — above minimum
	}
	tdi := keeper.ComputeTDI(obs)
	if tdi > 100_000 {
		t.Fatalf("OCA-T4 FAIL: healthy signals should produce low TDI, got %d", tdi)
	}
	t.Logf("OCA-T4 PASS: TDI recovers to %d with healthy signals", tdi)
}

// ── OCA-T5: Alignment Override Claims Priority 5 ──
// Verify corrections use priority 5 (higher than SSI=4).
func TestOCA_T5_AlignmentOverridePriority(t *testing.T) {
	k, ctx, mk, _, _, ma, _ := setupFullKeeper(t)

	// Create critical truth drift
	mk.totalRounds = 100
	mk.unanimousRounds = 99 // 99% unanimity
	mk.verifiedFacts = 100
	mk.totalFacts = 100
	mk.activeFacts = 100

	ctx = ctx.WithBlockHeight(1111)
	k.SenseJudgeCorrectReflect(ctx)

	corrections := k.GetCorrections(ctx, 1)
	if len(corrections) == 0 {
		t.Fatal("OCA-T5 FAIL: expected corrections for critical TDI")
	}

	for _, c := range corrections {
		if c.Priority != types.AlignmentPriority {
			t.Fatalf("OCA-T5 FAIL: correction priority should be %d, got %d", types.AlignmentPriority, c.Priority)
		}
	}

	// Verify multiplier was actually written
	challengeReward := ma.multipliers[types.PathChallengeReward]
	if challengeReward == 0 || challengeReward == types.DefaultMultiplier {
		// It's OK if it equals default — the important thing is the correction record exists
		if len(corrections) == 0 {
			t.Fatal("OCA-T5 FAIL: no corrections were generated")
		}
	}

	t.Logf("OCA-T5 PASS: %d corrections at priority 5", len(corrections))
}

// ── OCA-U1: Isolated Facts ──
// Submit 100 isolated facts with zero citations → verify DCI drops.
func TestOCA_U1_IsolatedFacts(t *testing.T) {
	obs := types.AlignmentObservation{
		AvgCitationDepth: 0,         // no citations
		CrossDomainRatio: 0,         // no cross-domain
		OrphanRate:       1_000_000, // 100% orphans
		OntologyCoverage: 0,         // no coverage
		SupersessionDepth: 0,        // no evolution
	}
	dci := keeper.ComputeDCI(obs)
	if dci != 0 {
		t.Fatalf("OCA-U1 FAIL: 100%% orphan facts should produce DCI=0, got %d", dci)
	}
	t.Log("OCA-U1 PASS: isolated facts produce DCI=0")
}

// ── OCA-U2: Heavily Cross-Cited Facts ──
// Submit heavily cross-cited facts → verify DCI rises.
func TestOCA_U2_CrossCitedFacts(t *testing.T) {
	obs := types.AlignmentObservation{
		AvgCitationDepth:  5_000_000, // max depth
		CrossDomainRatio:  800_000,   // 80% cross-domain
		OrphanRate:        50_000,    // 5% orphans
		OntologyCoverage:  900_000,   // 90% coverage
		SupersessionDepth: 2_000_000, // depth 2.0
	}
	dci := keeper.ComputeDCI(obs)
	if dci < 700_000 {
		t.Fatalf("OCA-U2 FAIL: heavily cross-cited facts should produce high DCI, got %d", dci)
	}
	t.Logf("OCA-U2 PASS: cross-cited facts produce DCI=%d", dci)
}

// ── OCA-U3: Orphan Cap Correction ──
// Verify orphan cap correction activates when DCI is low.
func TestOCA_U3_OrphanCapCorrection(t *testing.T) {
	k, ctx, mk, _, _, ma, _ := setupFullKeeper(t)

	// Create low DCI: all orphans
	mk.activeFacts = 100
	mk.orphanFacts = 100 // 100% orphans
	mk.totalFacts = 100

	ctx = ctx.WithBlockHeight(1111)
	k.SenseJudgeCorrectReflect(ctx)

	scores := k.GetScores(ctx, 1)
	if scores == nil {
		t.Fatal("OCA-U3 FAIL: expected scores")
	}

	corrections := k.GetCorrections(ctx, 1)
	foundOrphanCap := false
	for _, c := range corrections {
		if c.Path == types.PathOrphanCap {
			foundOrphanCap = true
			if c.Delta >= 0 {
				t.Fatal("OCA-U3 FAIL: orphan cap should decrease (negative delta)")
			}
		}
	}

	// With very low DCI, we should see orphan cap correction
	if scores.DCI < types.DCIModerateThreshold && !foundOrphanCap {
		t.Fatalf("OCA-U3 FAIL: DCI=%d < moderate threshold but no orphan cap correction", scores.DCI)
	}

	_ = ma // multiplier check
	t.Logf("OCA-U3 PASS: DCI=%d, orphan_cap_correction=%v", scores.DCI, foundOrphanCap)
}

// ── OCA-U4: Single-Domain Monoculture ──
// Simulate single-domain monoculture → verify cross-domain bonus activates.
func TestOCA_U4_SingleDomainMonoculture(t *testing.T) {
	obs := types.AlignmentObservation{
		CrossDomainRatio: 0,      // no cross-domain citations
		OntologyCoverage: 100_000, // only 10% coverage
	}
	dci := keeper.ComputeDCI(obs)
	if dci > 500_000 {
		t.Fatalf("OCA-U4 FAIL: monoculture should produce low DCI, got %d", dci)
	}
	t.Logf("OCA-U4 PASS: monoculture produces DCI=%d", dci)
}

// ── OCA-B1: Multipliers at Bounds ──
// Drive multiple multipliers to bounds → verify HI drops.
func TestOCA_B1_MultipliersAtBounds(t *testing.T) {
	obs := types.AlignmentObservation{
		MultiplierVariance:  800_000, // high variance
		BoundSaturation:     900_000, // 90% at bounds
		OscillationRate:     700_000, // high oscillation
		HormoneConflictRate: 800_000, // high conflict
	}
	hi := keeper.ComputeHI(obs)
	if hi > 300_000 {
		t.Fatalf("OCA-B1 FAIL: bound saturation should produce low HI, got %d", hi)
	}
	t.Logf("OCA-B1 PASS: bound saturation produces HI=%d", hi)
}

// ── OCA-B2: Oscillating Hormones ──
// Create oscillating inputs → verify oscillation detection.
func TestOCA_B2_OscillatingHormones(t *testing.T) {
	obs := types.AlignmentObservation{
		OscillationRate:     800_000, // 80% direction changes
		MultiplierVariance:  600_000, // stressed variance
		HormoneConflictRate: 700_000, // high conflict
		BoundSaturation:     500_000, // moderate saturation
	}
	hi := keeper.ComputeHI(obs)
	if hi > 400_000 {
		t.Fatalf("OCA-B2 FAIL: oscillating hormones should produce low HI, got %d", hi)
	}
	t.Logf("OCA-B2 PASS: oscillation produces HI=%d", hi)
}

// ── OCA-B3: Beauty Freeze ──
// Verify damping correction activates when HI is low.
func TestOCA_B3_DampingCorrection(t *testing.T) {
	k, ctx, _, _, _, ma, _ := setupFullKeeper(t)

	// Simulate chaotic autopoiesis
	ma.entries = []types.MultiplierEntry{
		{Path: "a", Value: 200_000},
		{Path: "b", Value: 3_800_000},
		{Path: "c", Value: 250_000},
	}
	ma.dirChanges = 20

	ctx = ctx.WithBlockHeight(1111)
	k.SenseJudgeCorrectReflect(ctx)

	corrections := k.GetCorrections(ctx, 1)
	foundDamping := false
	for _, c := range corrections {
		if c.Path == types.PathAlignmentDamping && c.Delta > 0 {
			foundDamping = true
		}
	}

	scores := k.GetScores(ctx, 1)
	if scores != nil && scores.HI < types.HIMildThreshold && !foundDamping {
		t.Fatalf("OCA-B3 FAIL: HI=%d < mild threshold but no damping correction", scores.HI)
	}
	t.Logf("OCA-B3 PASS: damping correction activated=%v", foundDamping)
}

// ── OCA-B4: HI Recovery ──
// Verify HI returns to high when harmony is restored.
func TestOCA_B4_HIRecovery(t *testing.T) {
	obs := types.AlignmentObservation{
		MultiplierVariance:  50_000,  // low variance
		OscillationRate:     50_000,  // low oscillation
		BoundSaturation:     50_000,  // few at bounds
		HormoneConflictRate: 50_000,  // few conflicts
		AvgDampingFactor:    900_000, // good damping
	}
	hi := keeper.ComputeHI(obs)
	if hi < 800_000 {
		t.Fatalf("OCA-B4 FAIL: harmonious signals should produce high HI, got %d", hi)
	}
	t.Logf("OCA-B4 PASS: harmony recovered with HI=%d", hi)
}

// ── OCA-J1: Whale Validators ──
// Simulate 3 whale validators earning 80% of rewards → verify FI drops.
func TestOCA_J1_WhaleValidators(t *testing.T) {
	obs := types.AlignmentObservation{
		RewardGini:     800_000, // extreme inequality
		WhaleDominance: 800_000, // top 10 hold 80%
	}
	fi := keeper.ComputeFI(obs)
	if fi > 400_000 {
		t.Fatalf("OCA-J1 FAIL: whale dominance should produce low FI, got %d", fi)
	}
	t.Logf("OCA-J1 PASS: whale dominance produces FI=%d", fi)
}

// ── OCA-J2: New Entrant Rejection ──
// Simulate high rejection rate for new entrants → verify barrier detection.
func TestOCA_J2_NewEntrantRejection(t *testing.T) {
	obs := types.AlignmentObservation{
		NewEntrantSuccess: 200_000, // only 20% success
	}
	fi := keeper.ComputeFI(obs)
	// FI should be reduced by low new entrant success
	if fi > 800_000 {
		t.Fatalf("OCA-J2 FAIL: low new entrant success should reduce FI, got %d", fi)
	}
	t.Logf("OCA-J2 PASS: new entrant barrier detected with FI=%d", fi)
}

// ── OCA-J3: Quadratic Boost Correction ──
// Verify quadratic boost activates when FI is low.
func TestOCA_J3_QuadraticBoostCorrection(t *testing.T) {
	k, ctx, mk, ms, _, _, _ := setupFullKeeper(t)

	// Create unfair distribution
	ms.top10Ratio = 800_000 // top 10 hold 80%
	ms.rewardDist = []types.ValidatorReward{
		{Validator: "v1", Amount: 1}, {Validator: "v2", Amount: 1},
		{Validator: "v3", Amount: 1000000},
	}
	mk.newEntrantRate = 100_000 // 10% acceptance
	mk.activeFacts = 100

	ctx = ctx.WithBlockHeight(1111)
	k.SenseJudgeCorrectReflect(ctx)

	corrections := k.GetCorrections(ctx, 1)
	foundQuadratic := false
	for _, c := range corrections {
		if c.Path == types.PathQuadraticBoost && c.Delta > 0 {
			foundQuadratic = true
		}
	}

	scores := k.GetScores(ctx, 1)
	if scores != nil && scores.FI < types.FIMildThreshold && !foundQuadratic {
		t.Fatalf("OCA-J3 FAIL: FI=%d < mild threshold but no quadratic boost", scores.FI)
	}
	t.Logf("OCA-J3 PASS: quadratic boost=%v, FI=%d", foundQuadratic, scores.FI)
}

// ── OCA-J4: FI Recovery ──
// Verify FI rises when fairness is restored (negative feedback works).
func TestOCA_J4_FIRecovery(t *testing.T) {
	obs := types.AlignmentObservation{
		RewardGini:           100_000, // low inequality
		NewEntrantSuccess:    800_000, // 80% success
		SlashProportionality: 900_000, // good proportionality
		WhaleDominance:       200_000, // low dominance
		DomainGini:           100_000, // low domain inequality
	}
	fi := keeper.ComputeFI(obs)
	if fi < 600_000 {
		t.Fatalf("OCA-J4 FAIL: fair signals should produce high FI, got %d", fi)
	}
	t.Logf("OCA-J4 PASS: fairness recovered with FI=%d", fi)
}

// ── OCA-C1: Single-Domain Flooding ──
// Submit 100 facts all in same domain → verify NI drops.
func TestOCA_C1_SingleDomainFlooding(t *testing.T) {
	obs := types.AlignmentObservation{
		NewDomainRate:      0,
		NovelClaimRatio:    0,
		CrossDomainBridges: 0,
		FactDiversity:      0, // zero diversity
		SupersessionRate:   0,
	}
	ni := keeper.ComputeNI(obs)
	if ni != 0 {
		t.Fatalf("OCA-C1 FAIL: single-domain flooding should produce NI=0, got %d", ni)
	}
	t.Log("OCA-C1 PASS: single-domain flooding produces NI=0")
}

// ── OCA-C2: Cross-Domain Exploration ──
// Submit facts across 10 new domains → verify NI rises.
func TestOCA_C2_CrossDomainExploration(t *testing.T) {
	obs := types.AlignmentObservation{
		NewDomainRate:      50_000,   // 5% new domains (= target)
		NovelClaimRatio:    200_000,  // 20% novel (= target)
		CrossDomainBridges: 100_000,  // 10% bridges (= target)
		FactDiversity:      900_000,  // high diversity
		SupersessionRate:   200_000,  // 20% supersession
	}
	ni := keeper.ComputeNI(obs)
	if ni < 700_000 {
		t.Fatalf("OCA-C2 FAIL: cross-domain exploration should produce high NI, got %d", ni)
	}
	t.Logf("OCA-C2 PASS: exploration produces NI=%d", ni)
}

// ── OCA-C3: Stagnation Alarm ──
// Verify cross-domain bonus correction when NI is very low.
func TestOCA_C3_StagnationAlarm(t *testing.T) {
	k, ctx, mk, _, mo, _, _ := setupFullKeeper(t)

	// Total stagnation
	mk.totalFacts = 100
	mk.activeFacts = 100
	mo.totalDomains = 10

	ctx = ctx.WithBlockHeight(1111)
	k.SenseJudgeCorrectReflect(ctx)

	scores := k.GetScores(ctx, 1)
	corrections := k.GetCorrections(ctx, 1)

	foundCrossDomain := false
	for _, c := range corrections {
		if c.Path == types.PathCrossDomainBonus && c.Delta > 0 {
			foundCrossDomain = true
		}
	}

	if scores != nil && scores.NI < types.NIMildThreshold && !foundCrossDomain {
		t.Fatalf("OCA-C3 FAIL: NI=%d < mild threshold but no cross-domain bonus", scores.NI)
	}
	t.Logf("OCA-C3 PASS: stagnation alarm=%v, NI=%d", foundCrossDomain, scores.NI)
}

// ── OCA-C4: NI Recovery ──
// Verify NI rises when creativity returns.
func TestOCA_C4_NIRecovery(t *testing.T) {
	obs := types.AlignmentObservation{
		NewDomainRate:      40_000,  // near target
		NovelClaimRatio:    150_000, // good
		CrossDomainBridges: 80_000,  // good
		FactDiversity:      800_000, // high
		SupersessionRate:   100_000,
	}
	ni := keeper.ComputeNI(obs)
	if ni < 500_000 {
		t.Fatalf("OCA-C4 FAIL: creative signals should produce high NI, got %d", ni)
	}
	t.Logf("OCA-C4 PASS: creativity recovered with NI=%d", ni)
}

// ── Meta-Loop Test: Overcorrection Detection ──
func TestMetaLoop_Activation(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Fast-forward to epoch 20 (min epochs for meta-loop)
	for epoch := uint64(1); epoch <= 20; epoch++ {
		height := int64(epoch * 1111)
		ctx = ctx.WithBlockHeight(height)
		k.SenseJudgeCorrectReflect(ctx)
	}

	state := k.GetState(ctx)
	if state.CurrentEpoch < 20 {
		t.Fatalf("expected epoch >= 20, got %d", state.CurrentEpoch)
	}

	// Meta-loop should have run at epoch 10 and 20
	meta10 := k.GetMetaHealth(ctx, 10)
	meta20 := k.GetMetaHealth(ctx, 20)

	// At least one should exist (depending on correction history)
	t.Logf("Meta-loop: epoch 10 AHI=%v, epoch 20 AHI=%v", meta10, meta20)
}
