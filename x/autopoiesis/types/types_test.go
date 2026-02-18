package types_test

import (
	"testing"

	"github.com/legible-money/legible/x/autopoiesis/types"
)

// ============ Cross-Stack Golden Vectors ============
//
// These test vectors ensure the Go and TS implementations produce identical
// results for identical inputs. The TS equivalents are in:
//   legible-prototype/tests/cross_stack/autopoiesis_golden_vectors.test.ts
//
// If any golden vector test fails, the autopoiesis system has diverged
// between the two stacks and must be reconciled before merge.

func TestGoldenVector_SSI_Healthy(t *testing.T) {
	// Healthy chain: 95% accuracy, 20/22 validators, 1/10 expiration
	obs := types.EpochObservation{
		ActiveValidators:     20,
		TotalValidators:      22,
		ValidatorAccuracyBps: 950_000, // 95%
		FactsCreated:         10,
		FactsExpired:         1,
		TotalActiveFacts:     100,
	}
	ssi := types.ComputeSSI(obs)

	// Expected: 0.30*(1-0.95) + 0.20*0 + 0.20*(1/10) + 0.15*(1-20/22) + 0.15*0
	//         = 0.30*0.05 + 0.20*0.10 + 0.15*0.0909
	//         = 0.015 + 0.020 + 0.01363
	//         = 0.04863
	// In BPS: ~48,630
	// Go integer math: inaccuracy=50000, expRatio=100000, nonPart=90909
	// (300000*50000 + 200000*0 + 200000*100000 + 150000*90909 + 150000*0) / 1000000
	// = (15000000000 + 0 + 20000000000 + 13636350000 + 0) / 1000000
	// = 48636350000 / 1000000 = 48636
	if ssi != 48636 {
		t.Errorf("SSI golden vector (healthy): expected 48636, got %d", ssi)
	}
}

func TestGoldenVector_SSI_Critical(t *testing.T) {
	// Critical chain: 30% accuracy, 3/22 validators, 10x expiration, 80% challenges, 70% decline
	obs := types.EpochObservation{
		ActiveValidators:        3,
		TotalValidators:         22,
		ValidatorAccuracyBps:    300_000,
		ChallengeSuccessRateBps: 800_000,
		FactsCreated:            1,
		FactsExpired:            10,
		TotalActiveFacts:        100,
		AvgConfidenceDeclineBps: 700_000,
	}
	ssi := types.ComputeSSI(obs)

	// inaccuracy = 1M - 300K = 700K
	// challengeRate = 800K
	// expRatio = (10 * 1M) / 1 = 10M → clamped to 1M
	// nonPart = 1M - (3 * 1M / 22) = 1M - 136363 = 863637
	// confDecline = 700K
	// SSI = (300K*700K + 200K*800K + 200K*1M + 150K*863637 + 150K*700K) / 1M
	//     = (210B + 160B + 200B + 129.54555B + 105B) / 1M
	//     = 804545 / 1M? Let me compute more carefully...
	// 300000*700000 = 210,000,000,000
	// 200000*800000 = 160,000,000,000
	// 200000*1000000 = 200,000,000,000
	// 150000*863637 = 129,545,550,000
	// 150000*700000 = 105,000,000,000
	// sum = 804,545,550,000
	// / 1,000,000 = 804,545
	if ssi != 804545 {
		t.Errorf("SSI golden vector (critical): expected 804545, got %d", ssi)
	}
}

func TestGoldenVector_SSI_Adaptation_Moderate(t *testing.T) {
	// Starting at 1.0x, moderate stress
	params := types.DefaultParams()
	current := uint64(1_000_000)
	ssi := uint64(600_000) // moderate

	newMult, delta := types.ComputeSSIAdaptation(ssi, current, &params)

	// Expected: moderate → +50000 (5%) target
	// Rate limit: max change = 1M * 50K / 1M = 50K
	// Target: 1050000, diff=50000, maxDelta=50000 → passes
	// Bounds: 500K < 1050K < 3M → passes
	if newMult != 1_050_000 {
		t.Errorf("SSI adaptation golden (moderate): expected 1050000, got %d", newMult)
	}
	if delta != 50_000 {
		t.Errorf("SSI adaptation delta golden (moderate): expected 50000, got %d", delta)
	}
}

func TestGoldenVector_SSI_Adaptation_Critical_RateLimited(t *testing.T) {
	// Starting at 1.0x, critical stress → wants +150000 but rate limited to +50000
	params := types.DefaultParams()
	current := uint64(1_000_000)
	ssi := uint64(900_000) // critical

	newMult, delta := types.ComputeSSIAdaptation(ssi, current, &params)

	// Expected: critical → +150000 target, rate limit 50000 → capped at 1050000
	if newMult != 1_050_000 {
		t.Errorf("SSI adaptation golden (critical rate-limited): expected 1050000, got %d", newMult)
	}
	if delta != 50_000 {
		t.Errorf("SSI adaptation delta golden (critical rate-limited): expected 50000, got %d", delta)
	}
}

func TestGoldenVector_GRM_HighUtilization(t *testing.T) {
	obs := types.EpochObservation{
		VerifiedFactsCreated:     800,
		MaxPossibleVerifications: 1000,
	}
	grm := types.ComputeGRM(obs)
	// 800 * 1M / 1000 = 800,000
	if grm != 800_000 {
		t.Errorf("GRM golden (high): expected 800000, got %d", grm)
	}
}

func TestGoldenVector_GRM_LowUtilization(t *testing.T) {
	obs := types.EpochObservation{
		VerifiedFactsCreated:     100,
		MaxPossibleVerifications: 1000,
	}
	grm := types.ComputeGRM(obs)
	// 100 * 1M / 1000 = 100,000
	if grm != 100_000 {
		t.Errorf("GRM golden (low): expected 100000, got %d", grm)
	}
}

func TestGoldenVector_GRM_Zero(t *testing.T) {
	obs := types.EpochObservation{
		VerifiedFactsCreated:     0,
		MaxPossibleVerifications: 0,
	}
	grm := types.ComputeGRM(obs)
	if grm != 0 {
		t.Errorf("GRM golden (zero): expected 0, got %d", grm)
	}
}

// ============ CRI Tests ============

func TestComputeCRI_Basic(t *testing.T) {
	retentions := []types.CategoryRetention{
		{Domain: "physics", Category: "empirical", WeightedRetBps: 850_000, FactCount: 10},
		{Domain: "physics", Category: "analytic", WeightedRetBps: 999_000, FactCount: 5}, // eternal — excluded
		{Domain: "math", Category: "formal", WeightedRetBps: 990_000, FactCount: 20},     // eternal — excluded
		{Domain: "general", Category: "historical", WeightedRetBps: 600_000, FactCount: 3},
	}
	cri := types.ComputeCRI(retentions)
	if len(cri) != 2 {
		t.Fatalf("expected 2 CRI values (eternal excluded), got %d", len(cri))
	}
	if cri[0].Domain != "physics" || cri[0].Category != "empirical" || cri[0].Value != 850_000 {
		t.Errorf("CRI[0] unexpected: %+v", cri[0])
	}
	if cri[1].Domain != "general" || cri[1].Category != "historical" || cri[1].Value != 600_000 {
		t.Errorf("CRI[1] unexpected: %+v", cri[1])
	}
}

func TestComputeCRI_ClampToMax(t *testing.T) {
	retentions := []types.CategoryRetention{
		{Domain: "physics", Category: "empirical", WeightedRetBps: 2_000_000}, // over max
	}
	cri := types.ComputeCRI(retentions)
	if len(cri) != 1 || cri[0].Value != types.BasisPoints {
		t.Errorf("expected clamped to 1M, got %d", cri[0].Value)
	}
}

func TestIsEternalCategory(t *testing.T) {
	if !types.IsEternalCategory("analytic") {
		t.Error("analytic should be eternal")
	}
	if !types.IsEternalCategory("formal") {
		t.Error("formal should be eternal")
	}
	if !types.IsEternalCategory("protocol") {
		t.Error("protocol should be eternal")
	}
	if types.IsEternalCategory("empirical") {
		t.Error("empirical should not be eternal")
	}
	if types.IsEternalCategory("historical") {
		t.Error("historical should not be eternal")
	}
}

// ============ DVI Tests ============

func TestComputeDVI_Growing(t *testing.T) {
	domains := []types.DomainObservation{
		{Domain: "physics", FactsCreated: 10, FactsExpired: 2, TotalActiveFacts: 100,
			AvgInitialConfBps: 800_000, AvgFinalConfBps: 200_000},
	}
	dvi := types.ComputeDVI(domains, nil)
	if len(dvi) != 1 {
		t.Fatalf("expected 1 DVI value, got %d", len(dvi))
	}
	if dvi[0].Value <= 0 {
		t.Errorf("expected positive DVI for growing domain, got %d", dvi[0].Value)
	}
	if dvi[0].ConsecutiveNegativeEpochs != 0 {
		t.Errorf("expected 0 consecutive negative, got %d", dvi[0].ConsecutiveNegativeEpochs)
	}
}

func TestComputeDVI_Shrinking(t *testing.T) {
	domains := []types.DomainObservation{
		{Domain: "physics", FactsCreated: 1, FactsExpired: 10, TotalActiveFacts: 100,
			AvgInitialConfBps: 800_000, AvgFinalConfBps: 800_000},
	}
	dvi := types.ComputeDVI(domains, nil)
	if len(dvi) != 1 || dvi[0].Value >= 0 {
		t.Errorf("expected negative DVI for shrinking domain, got %d", dvi[0].Value)
	}
	if dvi[0].ConsecutiveNegativeEpochs != 1 {
		t.Errorf("expected 1 consecutive negative epoch, got %d", dvi[0].ConsecutiveNegativeEpochs)
	}
}

func TestComputeDVI_ConsecutiveNegativeTracking(t *testing.T) {
	domains := []types.DomainObservation{
		{Domain: "physics", FactsCreated: 1, FactsExpired: 10, TotalActiveFacts: 100,
			AvgInitialConfBps: 800_000, AvgFinalConfBps: 800_000},
	}
	prev := []types.DVIValue{
		{Domain: "physics", Value: -50_000, ConsecutiveNegativeEpochs: 2},
	}
	dvi := types.ComputeDVI(domains, prev)
	if dvi[0].ConsecutiveNegativeEpochs != 3 {
		t.Errorf("expected 3 consecutive negative epochs, got %d", dvi[0].ConsecutiveNegativeEpochs)
	}
}

func TestComputeDVI_ResetOnPositive(t *testing.T) {
	domains := []types.DomainObservation{
		{Domain: "physics", FactsCreated: 20, FactsExpired: 1, TotalActiveFacts: 100,
			AvgInitialConfBps: 900_000, AvgFinalConfBps: 100_000},
	}
	prev := []types.DVIValue{
		{Domain: "physics", Value: -50_000, ConsecutiveNegativeEpochs: 5},
	}
	dvi := types.ComputeDVI(domains, prev)
	if dvi[0].ConsecutiveNegativeEpochs != 0 {
		t.Errorf("expected 0 consecutive after positive, got %d", dvi[0].ConsecutiveNegativeEpochs)
	}
}

// ============ VCF Tests ============

func TestComputeVCF_HighCascade(t *testing.T) {
	domains := []types.DomainObservation{
		{Domain: "physics", TotalActiveFacts: 100, NewCitations: 60},
	}
	vcf := types.ComputeVCF(domains)
	if len(vcf) != 1 {
		t.Fatalf("expected 1 VCF value, got %d", len(vcf))
	}
	// 60 * 1M / 100 = 600,000
	if vcf[0].Value != 600_000 {
		t.Errorf("expected 600000, got %d", vcf[0].Value)
	}
}

func TestComputeVCF_NoCitations(t *testing.T) {
	domains := []types.DomainObservation{
		{Domain: "physics", TotalActiveFacts: 100, NewCitations: 0},
	}
	vcf := types.ComputeVCF(domains)
	if vcf[0].Value != 0 {
		t.Errorf("expected 0, got %d", vcf[0].Value)
	}
}

func TestComputeVCF_ClampToMax(t *testing.T) {
	domains := []types.DomainObservation{
		{Domain: "physics", TotalActiveFacts: 1, NewCitations: 100},
	}
	vcf := types.ComputeVCF(domains)
	if vcf[0].Value != types.BasisPoints {
		t.Errorf("expected clamped to 1M, got %d", vcf[0].Value)
	}
}

// ============ GRM Adaptation Tests ============

func TestComputeGRMAdaptation_HighUtilization(t *testing.T) {
	params := types.DefaultParams()
	grm := uint64(800_000) // 80% utilization
	currentMaxTx := uint64(types.DefaultMultiplier)
	currentGas := uint64(types.DefaultMultiplier)

	adaptations := types.ComputeGRMAdaptation(grm, currentMaxTx, currentGas, &params)
	if len(adaptations) != 2 {
		t.Fatalf("expected 2 adaptations for high GRM, got %d", len(adaptations))
	}

	// MaxTx should increase
	if adaptations[0].Path != types.PathMaxTxSystem || adaptations[0].Delta <= 0 {
		t.Errorf("expected maxtx increase, got path=%s delta=%d", adaptations[0].Path, adaptations[0].Delta)
	}
	// Gas should decrease
	if adaptations[1].Path != types.PathGasSystem || adaptations[1].Delta >= 0 {
		t.Errorf("expected gas decrease, got path=%s delta=%d", adaptations[1].Path, adaptations[1].Delta)
	}
}

func TestComputeGRMAdaptation_LowUtilization(t *testing.T) {
	params := types.DefaultParams()
	grm := uint64(100_000) // 10% utilization
	currentMaxTx := uint64(types.DefaultMultiplier)
	currentGas := uint64(types.DefaultMultiplier)

	adaptations := types.ComputeGRMAdaptation(grm, currentMaxTx, currentGas, &params)
	if len(adaptations) != 2 {
		t.Fatalf("expected 2 adaptations for low GRM, got %d", len(adaptations))
	}

	// MaxTx should decrease
	if adaptations[0].Delta >= 0 {
		t.Errorf("expected maxtx decrease, got delta=%d", adaptations[0].Delta)
	}
	// Gas should increase
	if adaptations[1].Delta <= 0 {
		t.Errorf("expected gas increase, got delta=%d", adaptations[1].Delta)
	}
}

func TestComputeGRMAdaptation_Normal_NoChange(t *testing.T) {
	params := types.DefaultParams()
	grm := uint64(500_000) // 50% — normal, in [0.3, 0.7]

	adaptations := types.ComputeGRMAdaptation(grm, types.DefaultMultiplier, types.DefaultMultiplier, &params)
	if len(adaptations) != 0 {
		t.Errorf("expected no adaptations for normal GRM, got %d", len(adaptations))
	}
}

// ============ Life Force Tests ============

func TestComputeLifeForce_Healthy(t *testing.T) {
	obs := types.EpochObservation{
		FactsCreated:     10,
		FactsExpired:     2,
		TotalActiveFacts: 100,
	}
	cri := []types.CRIValue{
		{Value: 800_000},
		{Value: 900_000},
	}
	ssi := uint64(100_000) // low stress

	lf := types.ComputeLifeForce(obs, cri, ssi)

	if lf.Status != "healthy" && lf.Status != "thriving" {
		t.Errorf("expected healthy/thriving for good metrics, got %s (value=%d)", lf.Status, lf.Value)
	}
	if lf.Growth <= 0 {
		t.Errorf("expected positive growth, got %d", lf.Growth)
	}
	if lf.Resilience == 0 {
		t.Error("expected non-zero resilience with low SSI")
	}
}

func TestComputeLifeForce_Dying(t *testing.T) {
	obs := types.EpochObservation{
		FactsCreated:     0,
		FactsExpired:     100,
		TotalActiveFacts: 100,
	}
	cri := []types.CRIValue{
		{Value: 0}, // zero retention
	}
	ssi := uint64(1_000_000) // max stress

	lf := types.ComputeLifeForce(obs, cri, ssi)

	// growth=-1M, retention=0, connectivity=0, resilience=0
	// value = 0.25*(-1M) + 0.25*0 + 0.25*0 + 0.25*0 = -250,000 → "stressed"
	if lf.Status != "stressed" && lf.Status != "dying" {
		t.Errorf("expected stressed/dying for terrible metrics, got %s (value=%d)", lf.Status, lf.Value)
	}
}

func TestComputeLifeForceStatus(t *testing.T) {
	tests := []struct {
		value  int64
		expect string
	}{
		{600_000, "thriving"},
		{300_000, "healthy"},
		{0, "homeostatic"},
		{-300_000, "stressed"},
		{-600_000, "dying"},
	}
	for _, tt := range tests {
		got := types.ComputeLifeForceStatus(tt.value)
		if got != tt.expect {
			t.Errorf("LifeForceStatus(%d): expected %s, got %s", tt.value, tt.expect, got)
		}
	}
}

// ============ Damping Tests ============

func TestComputeDamping_AtGenesis(t *testing.T) {
	// At 1.0x (genesis), damping should be 1.0 (no resistance)
	damping := types.ComputeDamping(types.DefaultMultiplier, types.BoundsSlashMultiplier)
	if damping != types.BasisPoints {
		t.Errorf("expected full damping (1M) at genesis, got %d", damping)
	}
}

func TestComputeDamping_AtBounds(t *testing.T) {
	// At max bound (3.0x for slash), damping should be ~0.5 (high resistance)
	damping := types.ComputeDamping(types.BoundsSlashMultiplier.Max, types.BoundsSlashMultiplier)
	// d = (3M - 1M) / (3M - 1M) = 1.0, damping = 1/(1+1) = 0.5 = 500K BPS
	if damping != 500_000 {
		t.Errorf("expected ~500000 at max bound, got %d", damping)
	}
}

func TestComputeDamping_Midpoint(t *testing.T) {
	// At 2.0x (halfway to 3.0x bound for slash)
	damping := types.ComputeDamping(2_000_000, types.BoundsSlashMultiplier)
	// d = (2M - 1M) / (3M - 1M) = 0.5, d² = 0.25, damping = 1/1.25 = 0.8 = 800K BPS
	if damping != 800_000 {
		t.Errorf("expected 800000 at midpoint, got %d", damping)
	}
}

func TestApplyDampedDelta_SmallDelta(t *testing.T) {
	// At genesis, full damping → delta passes through
	effective := types.ApplyDampedDelta(50_000, types.DefaultMultiplier, types.BoundsSlashMultiplier, 100_000)
	if effective != 50_000 {
		t.Errorf("expected 50000 at genesis (full pass-through), got %d", effective)
	}
}

func TestApplyDampedDelta_RateLimited(t *testing.T) {
	// Large delta should be rate-limited
	effective := types.ApplyDampedDelta(200_000, types.DefaultMultiplier, types.BoundsSlashMultiplier, 50_000)
	if effective != 50_000 {
		t.Errorf("expected 50000 (rate limited), got %d", effective)
	}
}

func TestApplyDampedDelta_BoundEnforced(t *testing.T) {
	// Near max bound, delta should be clamped
	effective := types.ApplyDampedDelta(500_000, 2_900_000, types.BoundsSlashMultiplier, 500_000)
	// Max is 3M, current is 2.9M, max delta to reach bound = 100K
	// But damping at 2.9M: d = (2.9M - 1M) / (3M - 1M) = 0.95, d² = 0.9025
	// damping = 1 / 1.9025 ≈ 0.5256 ≈ 525641 BPS
	// damped = 500K * 525641 / 1M = 262820
	// Rate limit: 262820 < 500K → passes
	// Bound: 2.9M + 262820 = 3162820 > 3M → clamped to 3M - 2.9M = 100K
	if effective != 100_000 {
		t.Errorf("expected 100000 (bound clamped), got %d", effective)
	}
}

// ============ Multiplier Path Tests ============

func TestMultiplierPaths(t *testing.T) {
	if types.PathDecay("empirical", "physics") != "decay.empirical.physics" {
		t.Error("PathDecay mismatch")
	}
	if types.PathRevBonus("empirical", "physics") != "revbonus.empirical.physics" {
		t.Error("PathRevBonus mismatch")
	}
	if types.PathReward("physics") != "reward.physics" {
		t.Error("PathReward mismatch")
	}
	if types.PathClaimStake("physics") != "claimstake.physics" {
		t.Error("PathClaimStake mismatch")
	}
	if types.PathCitation("physics") != "citation.physics" {
		t.Error("PathCitation mismatch")
	}
	if types.PathDomainPrior("physics") != "domainprior.physics" {
		t.Error("PathDomainPrior mismatch")
	}
}

// ============ Hormone Priority Tests ============

func TestHormonePriority(t *testing.T) {
	if types.HormonePriority[types.HormoneSSI] <= types.HormonePriority[types.HormoneCRI] {
		t.Error("SSI should have higher priority than CRI")
	}
	if types.HormonePriority[types.HormoneCRI] <= types.HormonePriority[types.HormoneDVI] {
		t.Error("CRI should have higher priority than DVI")
	}
	if types.HormonePriority[types.HormoneDVI] != types.HormonePriority[types.HormoneVCF] {
		t.Error("DVI and VCF should have equal priority")
	}
	if types.HormonePriority[types.HormoneDVI] <= types.HormonePriority[types.HormoneGRM] {
		t.Error("DVI should have higher priority than GRM")
	}
}
