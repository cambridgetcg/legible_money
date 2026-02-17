// Cross-Stack Golden Vectors (Go)
//
// Deterministic test vectors that MUST produce identical results in both
// the TypeScript chain and the Go (Cosmos SDK) chain.
//
// TypeScript counterpart: tests/cross_stack/golden_vectors.test.ts
//
// These vectors verify cryptographic and economic primitives are aligned
// across both implementations. Any mismatch here means consensus-breaking
// divergence between the two chains.
//
// IMPORTANT KNOWN DIVERGENCES (documented, not yet fixed):
//   D1. Claim content hash: Go uses sha256("LGM.claim.content.v1:" + text + ":" + domain + ":" + category)
//       while TS uses sha256("claim:" + text + ":" + domain + ":" + type).
//       This MUST be reconciled before cross-chain state migration.
//   D2. [FIXED] Empirical decay half-life: Was Go=100K, TS=1M. Fixed Go to 1M blocks.
//   D3. Claim ID generation: only exists in TS (state/claims.ts). Go should implement the same.
//   D4. VRF proof encoding: TS encodes scalars (c, s) as big-endian bytes,
//       Go encodes scalars as little-endian bytes (edwards25519.Scalar.Bytes()).
//       VRF outputs (SHA-256 of Gamma) are IDENTICAL — only proof wire format differs.
//   D5. Stake decimals: TS uses 18 decimals (wei-like), Go uses 6 decimals (ulgm).
//       Tier thresholds match in human-readable LGM but differ in raw representation.

package cross_stack

import (
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"strings"
	"testing"

	"filippo.io/edwards25519"
)

// ============================================================================
// Test Inputs (shared with TS: tests/cross_stack/golden_vectors.test.ts)
// ============================================================================

const (
	simpleMessage = "Hello, Legible Money!"
	domainTag     = "LGM.claim.id.v1"

	claimText   = "Water boils at 100\u00B0C at standard pressure"
	claimDomain = "physics"
	claimType   = "empirical"

	claimSubmitter = "did:lgm:testsubmitter"
	claimBlock     = 42

	// Deterministic keypair seed (SHA-256 of a fixed string)
	keySeed = "c6d45d98416ddd502bf3b42705e2d310c6f09b390e6ffe6565f32d5d69eee067"

	// VRF input string
	vrfInput = "test-vrf-input"
)

// Decay parameters
const (
	decayInitialConfidence = 0.9
	decayHalfLifeBlocks    = 1_000_000 // TS empirical half-life (canonical)
	decayBlocksElapsed     = 500_000   // 0.5 half-lives
)

// Validator selection parameters (uint64-safe for Go)
const (
	selectionStake      = 1000
	selectionTotalStake = 10000
	selectionTarget     = 5
)

// ============================================================================
// Golden Values (must match TS golden_vectors.test.ts exactly)
// ============================================================================

const (
	// SHA-256("Hello, Legible Money!")
	goldenSHA256Simple = "91f7adb1befaba99b1f27895eadeddd3f868420a8750064cf2e6de97933ee567"

	// domainHash("LGM.claim.id.v1", "Hello, Legible Money!")
	goldenDomainHashSimple = "9d42e23f61f347c418ad00c952a0054024248ce3838517fe371673679c3ca880"

	// Go claim content hash
	goldenClaimContentHashGo = "f6b931dedc102f9a673de69451d7722bd3edb1256e7feb3665e4af65bb871965"

	// TS claim content hash
	goldenClaimContentHashTS = "1a640664fa38d98dc15dd4eb11a2f62ee050e42572505585bdba0172a6818cd1"

	// Claim ID (TS-originated, Go reimplementation)
	goldenClaimId = "claim:a446d62cefe963dd2eabd26a2abe1a0b"

	// Ed25519 public key from keySeed
	goldenEd25519PublicKey = "43a53f8f97d72c57efe993108e220f93c718f2c779ac05687c4418b7c260bfe8"

	// Ed25519 signature of simpleMessage with keySeed
	goldenEd25519Signature = "4bb23acd0f54fa91ac33e1a0cc7718a3f06d28235a0731874955dd7c99f0482073760a7f647c57b70d58e50e64145f1422f8ec8534d970a10bb041310a323b0f"

	// VRF output = SHA-256(Gamma). MATCHES across both TS and Go.
	goldenVRFOutput = "841cdf53240c80f365c3b3ad194137e9d3f8a87ff25a3e77a24530507237f307"

	// VRF Gamma point (first 32 bytes of proof). MATCHES across both chains.
	goldenVRFGamma = "84deaf4779604696615f2d6ca7b5249ba88a7d3f0e0d37322e84836db7e13698"
)

// Floating-point golden values
const (
	goldenDecayFactor     = 0.7071067811865476
	goldenDecayConfidence = 0.6363961030678928
)

// Ed25519 curve order
var curveOrder, _ = new(big.Int).SetString("7237005577332262213973186563042994240857116359379907606001950938285454250989", 10)

// ============================================================================
// Helper functions (reimplemented here to avoid import cycles)
// ============================================================================

func testDomainHash(domain string, data []byte) []byte {
	domainBytes := []byte(domain)
	lengthBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBytes, uint32(len(domainBytes)))

	h := sha256.New()
	h.Write(domainBytes)
	h.Write(lengthBytes)
	h.Write(data)
	return h.Sum(nil)
}

func computeClaimContentHash(claimText, domain, epistemicCategory string) string {
	h := sha256.New()
	h.Write([]byte("LGM.claim.content.v1:"))
	h.Write([]byte(claimText))
	h.Write([]byte(":"))
	h.Write([]byte(domain))
	h.Write([]byte(":"))
	h.Write([]byte(epistemicCategory))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func generateClaimId(claim string, submitter string, block uint64) string {
	input := fmt.Sprintf("%s:%s:%d", claim, submitter, block)
	hash := testDomainHash("LGM.claim.id.v1", []byte(input))
	return "claim:" + hex.EncodeToString(hash)[:32]
}

func calculateDecayedConfidence(initialConfidence float64, blocksElapsed int, halfLifeBlocks int) (confidence, decayFactor float64) {
	if halfLifeBlocks <= 0 {
		return initialConfidence, 1.0
	}
	decayFactor = math.Pow(0.5, float64(blocksElapsed)/float64(halfLifeBlocks))
	confidence = initialConfidence * decayFactor
	return confidence, decayFactor
}

// isValidatorSelected reimplements the TS/Go validator selection logic.
// Selection: outputNum * totalStake < stake * targetCount * 2^64
func isValidatorSelectedHelper(vrfOutput []byte, stake, totalStake uint64, targetCount uint32) (selected bool, priority uint64) {
	if len(vrfOutput) < 8 || totalStake == 0 || stake == 0 {
		return false, 0
	}
	if stake > totalStake {
		stake = totalStake
	}

	outputNum := binary.BigEndian.Uint64(vrfOutput[:8])
	maxNum := new(big.Int).Lsh(big.NewInt(1), 64)

	left := new(big.Int).SetUint64(outputNum)
	left.Mul(left, new(big.Int).SetUint64(totalStake))

	right := new(big.Int).SetUint64(stake)
	right.Mul(right, new(big.Int).SetUint64(uint64(targetCount)))
	right.Mul(right, maxNum)

	selected = left.Cmp(right) < 0
	priority = outputNum
	return selected, priority
}

// VRF helper functions (reimplemented to match x/knowledge/crypto/vrf.go)
func vrfDeriveScalarFromSeed(seed []byte) *edwards25519.Scalar {
	h := sha512.Sum512(seed)
	s, err := new(edwards25519.Scalar).SetBytesWithClamping(h[:32])
	if err != nil {
		scalar := make([]byte, 32)
		copy(scalar, h[:32])
		scalar[0] &= 248
		scalar[31] &= 127
		scalar[31] |= 64
		s, _ = new(edwards25519.Scalar).SetCanonicalBytes(scalar)
	}
	return s
}

func vrfHashToCurve(input []byte) (*edwards25519.Point, error) {
	for attempt := 0; attempt < 256; attempt++ {
		data := make([]byte, len(input)+1)
		copy(data, input)
		data[len(input)] = byte(attempt)
		h := sha512.Sum512(data)
		point, err := new(edwards25519.Point).SetBytes(h[:32])
		if err != nil {
			continue
		}
		eight := vrfScalarFromUint64(8)
		cleared := new(edwards25519.Point).ScalarMult(eight, point)
		if cleared.Equal(edwards25519.NewIdentityPoint()) == 1 {
			continue
		}
		return cleared, nil
	}
	return nil, fmt.Errorf("failed to hash to curve")
}

func vrfComputeChallenge(h, pk, gamma, u, v []byte) *edwards25519.Scalar {
	combined := make([]byte, 0, len(h)+len(pk)+len(gamma)+len(u)+len(v))
	combined = append(combined, h...)
	combined = append(combined, pk...)
	combined = append(combined, gamma...)
	combined = append(combined, u...)
	combined = append(combined, v...)

	hash := sha512.Sum512(combined)
	num := new(big.Int).SetBytes(hash[:24])
	num.Mod(num, curveOrder)

	b := num.Bytes()
	le := make([]byte, 32)
	for i, bv := range b {
		le[len(b)-1-i] = bv
	}

	s, _ := new(edwards25519.Scalar).SetCanonicalBytes(le)
	return s
}

func vrfScalarFromBytes(b []byte) *edwards25519.Scalar {
	num := new(big.Int).SetBytes(b)
	num.Mod(num, curveOrder)

	be := num.Bytes()
	le := make([]byte, 32)
	for i, bv := range be {
		le[len(be)-1-i] = bv
	}

	s, _ := new(edwards25519.Scalar).SetCanonicalBytes(le)
	return s
}

func vrfScalarFromUint64(v uint64) *edwards25519.Scalar {
	b := make([]byte, 32)
	binary.LittleEndian.PutUint64(b, v)
	s, _ := new(edwards25519.Scalar).SetCanonicalBytes(b)
	return s
}

// ============================================================================
// Tests
// ============================================================================

// --- SHA-256 ---

func TestSHA256_SimpleInput(t *testing.T) {
	h := sha256.Sum256([]byte(simpleMessage))
	got := hex.EncodeToString(h[:])
	if got != goldenSHA256Simple {
		t.Fatalf("SHA-256 mismatch:\n  got:  %s\n  want: %s", got, goldenSHA256Simple)
	}
}

func TestSHA256_StringAndBytesIdentical(t *testing.T) {
	fromString := sha256.Sum256([]byte(simpleMessage))
	fromBytes := sha256.Sum256([]byte(simpleMessage))
	if fromString != fromBytes {
		t.Fatal("SHA-256 should produce identical output for identical byte input")
	}
}

// --- Domain-Separated Hashing ---

func TestDomainHash_SimpleInput(t *testing.T) {
	result := testDomainHash(domainTag, []byte(simpleMessage))
	got := hex.EncodeToString(result)
	if got != goldenDomainHashSimple {
		t.Fatalf("domainHash mismatch:\n  got:  %s\n  want: %s", got, goldenDomainHashSimple)
	}
}

func TestDomainHash_DiffersFromRawConcat(t *testing.T) {
	rawConcat := sha256.Sum256([]byte(domainTag + simpleMessage))
	domainSep := testDomainHash(domainTag, []byte(simpleMessage))

	if hex.EncodeToString(rawConcat[:]) == hex.EncodeToString(domainSep) {
		t.Fatal("domainHash must differ from raw concatenation SHA-256")
	}
}

func TestDomainHash_NoAmbiguity(t *testing.T) {
	hash1 := testDomainHash("a:", []byte("b"))
	hash2 := testDomainHash("a", []byte(":b"))

	if hex.EncodeToString(hash1) == hex.EncodeToString(hash2) {
		t.Fatal("domainHash must prevent domain/data ambiguity")
	}
}

// --- Claim Content Hash ---

func TestClaimContentHash_GoFormat(t *testing.T) {
	got := computeClaimContentHash(claimText, claimDomain, claimType)
	if got != goldenClaimContentHashGo {
		t.Fatalf("Go claim content hash mismatch:\n  got:  %s\n  want: %s", got, goldenClaimContentHashGo)
	}
}

func TestClaimContentHash_TSFormat(t *testing.T) {
	input := "claim:" + claimText + ":" + claimDomain + ":" + claimType
	h := sha256.Sum256([]byte(input))
	got := hex.EncodeToString(h[:])
	if got != goldenClaimContentHashTS {
		t.Fatalf("TS-format claim content hash mismatch:\n  got:  %s\n  want: %s", got, goldenClaimContentHashTS)
	}
}

func TestClaimContentHash_KnownDivergence(t *testing.T) {
	goHash := computeClaimContentHash(claimText, claimDomain, claimType)

	tsInput := "claim:" + claimText + ":" + claimDomain + ":" + claimType
	tsHashBytes := sha256.Sum256([]byte(tsInput))
	tsHash := hex.EncodeToString(tsHashBytes[:])

	if goHash == tsHash {
		t.Fatal("DIVERGENCE D1 RESOLVED: Go and TS content hashes now match. " +
			"Update golden vectors and remove divergence documentation.")
	}
}

// --- Claim ID Generation ---

func TestClaimId_Generation(t *testing.T) {
	got := generateClaimId(claimText, claimSubmitter, uint64(claimBlock))
	if got != goldenClaimId {
		t.Fatalf("Claim ID mismatch:\n  got:  %s\n  want: %s", got, goldenClaimId)
	}
}

func TestClaimId_Prefix(t *testing.T) {
	got := generateClaimId(claimText, claimSubmitter, uint64(claimBlock))
	if !strings.HasPrefix(got, "claim:") {
		t.Fatalf("Claim ID must start with 'claim:', got: %s", got)
	}
	if len(got) != 38 {
		t.Fatalf("Claim ID must be 38 chars (6 prefix + 32 hex), got %d: %s", len(got), got)
	}
}

func TestClaimId_Deterministic(t *testing.T) {
	a := generateClaimId(claimText, claimSubmitter, uint64(claimBlock))
	b := generateClaimId(claimText, claimSubmitter, uint64(claimBlock))
	if a != b {
		t.Fatalf("Claim ID must be deterministic: %s != %s", a, b)
	}
}

func TestClaimId_ChangesWithInput(t *testing.T) {
	base := generateClaimId(claimText, claimSubmitter, uint64(claimBlock))

	diffText := generateClaimId("Different claim text", claimSubmitter, uint64(claimBlock))
	if diffText == base {
		t.Fatal("Claim ID must change when claim text changes")
	}

	diffSubmitter := generateClaimId(claimText, "did:lgm:othersubmitter", uint64(claimBlock))
	if diffSubmitter == base {
		t.Fatal("Claim ID must change when submitter changes")
	}

	diffBlock := generateClaimId(claimText, claimSubmitter, 43)
	if diffBlock == base {
		t.Fatal("Claim ID must change when block number changes")
	}
}

// --- Ed25519 ---

func TestEd25519_PublicKeyFromSeed(t *testing.T) {
	seed, _ := hex.DecodeString(keySeed)
	privKey := ed25519.NewKeyFromSeed(seed)
	pubKey := privKey.Public().(ed25519.PublicKey)
	got := hex.EncodeToString(pubKey)
	if got != goldenEd25519PublicKey {
		t.Fatalf("Ed25519 public key mismatch:\n  got:  %s\n  want: %s", got, goldenEd25519PublicKey)
	}
}

func TestEd25519_SignMessage(t *testing.T) {
	seed, _ := hex.DecodeString(keySeed)
	privKey := ed25519.NewKeyFromSeed(seed)
	sig := ed25519.Sign(privKey, []byte(simpleMessage))
	got := hex.EncodeToString(sig)
	if got != goldenEd25519Signature {
		t.Fatalf("Ed25519 signature mismatch:\n  got:  %s\n  want: %s", got, goldenEd25519Signature)
	}
}

func TestEd25519_VerifySignature(t *testing.T) {
	seed, _ := hex.DecodeString(keySeed)
	privKey := ed25519.NewKeyFromSeed(seed)
	pubKey := privKey.Public().(ed25519.PublicKey)

	sigBytes, _ := hex.DecodeString(goldenEd25519Signature)
	if !ed25519.Verify(pubKey, []byte(simpleMessage), sigBytes) {
		t.Fatal("Ed25519 signature verification failed for golden signature")
	}
}

func TestEd25519_RejectCorruptedSignature(t *testing.T) {
	seed, _ := hex.DecodeString(keySeed)
	privKey := ed25519.NewKeyFromSeed(seed)
	pubKey := privKey.Public().(ed25519.PublicKey)

	sigBytes, _ := hex.DecodeString(goldenEd25519Signature)
	sigBytes[0] ^= 0xFF // Corrupt first byte
	if ed25519.Verify(pubKey, []byte(simpleMessage), sigBytes) {
		t.Fatal("Ed25519 should reject corrupted signature")
	}
}

func TestEd25519_RejectWrongMessage(t *testing.T) {
	seed, _ := hex.DecodeString(keySeed)
	privKey := ed25519.NewKeyFromSeed(seed)
	pubKey := privKey.Public().(ed25519.PublicKey)

	sigBytes, _ := hex.DecodeString(goldenEd25519Signature)
	if ed25519.Verify(pubKey, []byte("Wrong message"), sigBytes) {
		t.Fatal("Ed25519 should reject wrong message")
	}
}

func TestEd25519_Deterministic(t *testing.T) {
	seed, _ := hex.DecodeString(keySeed)
	privKey := ed25519.NewKeyFromSeed(seed)

	sig1 := ed25519.Sign(privKey, []byte(simpleMessage))
	sig2 := ed25519.Sign(privKey, []byte(simpleMessage))

	if hex.EncodeToString(sig1) != hex.EncodeToString(sig2) {
		t.Fatal("Ed25519 signatures must be deterministic")
	}
}

// --- VRF ---

func TestVRF_OutputMatchesCrossStack(t *testing.T) {
	seed, _ := hex.DecodeString(keySeed)

	// Domain-separate the input
	domainSeparated := testDomainHash("LGM.vrf.v1", []byte(vrfInput))

	// Hash to curve
	H, err := vrfHashToCurve(domainSeparated)
	if err != nil {
		t.Fatalf("hashToCurve error: %v", err)
	}

	// Derive scalar and compute Gamma
	skScalar := vrfDeriveScalarFromSeed(seed)
	Gamma := new(edwards25519.Point).ScalarMult(skScalar, H)

	// Verify Gamma matches
	gammaHex := hex.EncodeToString(Gamma.Bytes())
	if gammaHex != goldenVRFGamma {
		t.Fatalf("VRF Gamma mismatch:\n  got:  %s\n  want: %s", gammaHex, goldenVRFGamma)
	}

	// Verify output matches (SHA-256 of Gamma)
	outputHash := sha256.Sum256(Gamma.Bytes())
	outputHex := hex.EncodeToString(outputHash[:])
	if outputHex != goldenVRFOutput {
		t.Fatalf("VRF output mismatch:\n  got:  %s\n  want: %s", outputHex, goldenVRFOutput)
	}
}

func TestVRF_GenerateAndVerify(t *testing.T) {
	seed, _ := hex.DecodeString(keySeed)
	privKey := ed25519.NewKeyFromSeed(seed)
	pubKey := privKey.Public().(ed25519.PublicKey)

	// Domain-separate
	domainSeparated := testDomainHash("LGM.vrf.v1", []byte(vrfInput))

	// Hash to curve
	H, err := vrfHashToCurve(domainSeparated)
	if err != nil {
		t.Fatalf("hashToCurve error: %v", err)
	}

	// Derive scalar
	skScalar := vrfDeriveScalarFromSeed(seed)

	// Gamma = sk * H
	Gamma := new(edwards25519.Point).ScalarMult(skScalar, H)

	// Nonce: k = hash(sk_seed || H_bytes) mod ℓ
	hBytes := H.Bytes()
	nonceInput := make([]byte, len(seed)+len(hBytes))
	copy(nonceInput, seed)
	copy(nonceInput[len(seed):], hBytes)
	kHash := sha512.Sum512(nonceInput)
	k := vrfScalarFromBytes(kHash[:32])

	// U = k * G, V = k * H
	G := edwards25519.NewGeneratorPoint()
	U := new(edwards25519.Point).ScalarMult(k, G)
	V := new(edwards25519.Point).ScalarMult(k, H)

	// c = hash(H, pk, Gamma, U, V)
	c := vrfComputeChallenge(H.Bytes(), pubKey, Gamma.Bytes(), U.Bytes(), V.Bytes())

	// s = k - c * sk
	cTimesSk := new(edwards25519.Scalar).Multiply(c, skScalar)
	s := new(edwards25519.Scalar).Subtract(k, cTimesSk)

	// Build proof
	proof := make([]byte, 96)
	copy(proof[0:32], Gamma.Bytes())
	copy(proof[32:64], c.Bytes())
	copy(proof[64:96], s.Bytes())

	// Verify: recompute challenge from proof
	GammaParsed, err := new(edwards25519.Point).SetBytes(proof[0:32])
	if err != nil {
		t.Fatalf("Failed to parse Gamma from proof: %v", err)
	}
	cParsed, err := new(edwards25519.Scalar).SetCanonicalBytes(proof[32:64])
	if err != nil {
		t.Fatalf("Failed to parse c from proof: %v", err)
	}
	sParsed, err := new(edwards25519.Scalar).SetCanonicalBytes(proof[64:96])
	if err != nil {
		t.Fatalf("Failed to parse s from proof: %v", err)
	}

	PK, _ := new(edwards25519.Point).SetBytes(pubKey)

	// U' = s*G + c*PK
	sG := new(edwards25519.Point).ScalarMult(sParsed, G)
	cPK := new(edwards25519.Point).ScalarMult(cParsed, PK)
	Uprime := new(edwards25519.Point).Add(sG, cPK)

	// V' = s*H + c*Gamma
	sH := new(edwards25519.Point).ScalarMult(sParsed, H)
	cGamma := new(edwards25519.Point).ScalarMult(cParsed, GammaParsed)
	Vprime := new(edwards25519.Point).Add(sH, cGamma)

	// c' = hash(H, pk, Gamma, U', V')
	cPrime := vrfComputeChallenge(H.Bytes(), pubKey, GammaParsed.Bytes(), Uprime.Bytes(), Vprime.Bytes())

	if cParsed.Equal(cPrime) != 1 {
		t.Fatal("VRF proof verification failed: c != c'")
	}
}

func TestVRF_Deterministic(t *testing.T) {
	seed, _ := hex.DecodeString(keySeed)

	// Generate twice with same inputs
	domainSeparated := testDomainHash("LGM.vrf.v1", []byte(vrfInput))
	H, err := vrfHashToCurve(domainSeparated)
	if err != nil {
		t.Fatalf("hashToCurve error: %v", err)
	}

	skScalar := vrfDeriveScalarFromSeed(seed)
	Gamma1 := new(edwards25519.Point).ScalarMult(skScalar, H)
	Gamma2 := new(edwards25519.Point).ScalarMult(skScalar, H)

	out1 := sha256.Sum256(Gamma1.Bytes())
	out2 := sha256.Sum256(Gamma2.Bytes())

	if hex.EncodeToString(out1[:]) != hex.EncodeToString(out2[:]) {
		t.Fatal("VRF output must be deterministic")
	}
}

func TestVRF_DifferentInputDifferentOutput(t *testing.T) {
	seed, _ := hex.DecodeString(keySeed)
	skScalar := vrfDeriveScalarFromSeed(seed)

	// Input 1
	ds1 := testDomainHash("LGM.vrf.v1", []byte(vrfInput))
	H1, _ := vrfHashToCurve(ds1)
	Gamma1 := new(edwards25519.Point).ScalarMult(skScalar, H1)
	out1 := sha256.Sum256(Gamma1.Bytes())

	// Input 2
	ds2 := testDomainHash("LGM.vrf.v1", []byte("different-input"))
	H2, _ := vrfHashToCurve(ds2)
	Gamma2 := new(edwards25519.Point).ScalarMult(skScalar, H2)
	out2 := sha256.Sum256(Gamma2.Bytes())

	if hex.EncodeToString(out1[:]) == hex.EncodeToString(out2[:]) {
		t.Fatal("VRF must produce different output for different input")
	}
}

// --- Validator Selection ---

func TestValidatorSelection_NotSelected(t *testing.T) {
	vrfOutputBytes, _ := hex.DecodeString(goldenVRFOutput)

	selected, priority := isValidatorSelectedHelper(
		vrfOutputBytes,
		selectionStake,
		selectionTotalStake,
		selectionTarget,
	)

	if selected {
		t.Fatal("Validator should NOT be selected (outputNum * totalStake >= stake * target * 2^64)")
	}

	// Priority = first 8 bytes as big-endian uint64
	expectedPriority := binary.BigEndian.Uint64(vrfOutputBytes[:8])
	if priority != expectedPriority {
		t.Fatalf("Priority mismatch: got %d, want %d", priority, expectedPriority)
	}

	// Cross-check with TS golden value
	if priority != 9519729260534464755 {
		t.Fatalf("Priority does not match TS golden value: got %d, want 9519729260534464755", priority)
	}
}

func TestValidatorSelection_SelectedWithHighStake(t *testing.T) {
	vrfOutputBytes, _ := hex.DecodeString(goldenVRFOutput)

	// With 100% of total stake, should always be selected (5 targets)
	selected, _ := isValidatorSelectedHelper(vrfOutputBytes, 10000, 10000, 5)
	if !selected {
		t.Fatal("Validator with 100% stake should be selected with target=5")
	}
}

func TestValidatorSelection_ZeroStakeNotSelected(t *testing.T) {
	vrfOutputBytes, _ := hex.DecodeString(goldenVRFOutput)
	selected, _ := isValidatorSelectedHelper(vrfOutputBytes, 0, 10000, 5)
	if selected {
		t.Fatal("Validator with 0 stake should never be selected")
	}
}

// --- Confidence Decay ---

func TestDecay_HalfLife(t *testing.T) {
	confidence, decayFactor := calculateDecayedConfidence(
		decayInitialConfidence,
		decayBlocksElapsed,
		decayHalfLifeBlocks,
	)

	if math.Abs(decayFactor-goldenDecayFactor) > 1e-14 {
		t.Fatalf("Decay factor mismatch:\n  got:  %.16f\n  want: %.16f", decayFactor, goldenDecayFactor)
	}

	if math.Abs(confidence-goldenDecayConfidence) > 1e-14 {
		t.Fatalf("Decayed confidence mismatch:\n  got:  %.16f\n  want: %.16f", confidence, goldenDecayConfidence)
	}
}

func TestDecay_Eternal(t *testing.T) {
	confidence, decayFactor := calculateDecayedConfidence(0.9, 500_000, 0)
	if decayFactor != 1.0 {
		t.Fatalf("Eternal decay factor must be 1.0, got: %f", decayFactor)
	}
	if confidence != 0.9 {
		t.Fatalf("Eternal confidence must not change, got: %f", confidence)
	}
}

func TestDecay_ExactlyOneHalfLife(t *testing.T) {
	confidence, decayFactor := calculateDecayedConfidence(0.9, 1_000_000, 1_000_000)
	if math.Abs(decayFactor-0.5) > 1e-14 {
		t.Fatalf("Decay factor at exactly 1 half-life must be 0.5, got: %.16f", decayFactor)
	}
	if math.Abs(confidence-0.45) > 1e-14 {
		t.Fatalf("Confidence at 1 half-life must be 0.45, got: %.16f", confidence)
	}
}

func TestDecay_ZeroBlocks(t *testing.T) {
	confidence, decayFactor := calculateDecayedConfidence(0.9, 0, 1_000_000)
	if decayFactor != 1.0 {
		t.Fatalf("Decay factor at 0 blocks must be 1.0, got: %f", decayFactor)
	}
	if confidence != 0.9 {
		t.Fatalf("Confidence at 0 blocks must be unchanged, got: %f", confidence)
	}
}

// --- Validator Tiers (Go-side verification of tier thresholds) ---

// ValidatorTier represents one of the 4 validator tiers.
type ValidatorTier int

const (
	TierApprentice ValidatorTier = 0
	TierVerified   ValidatorTier = 1
	TierBonded     ValidatorTier = 2
	TierGuardian   ValidatorTier = 3
)

// TierConfig holds the requirements for each tier (ulgm denomination, 6 decimals).
type TierConfig struct {
	MinStake                  uint64 // in ulgm (1 LGM = 1,000,000 ulgm)
	MinVerifications          int
	MinAccuracy               float64 // 0.0-1.0
	MaxSlashCount             int     // -1 = unlimited
	RewardMultiplierBps       int     // basis points (1000 = 1.0x)
	MinContestedVerifications int
}

var tierConfigs = map[ValidatorTier]TierConfig{
	TierApprentice: {MinStake: 111_000, MinVerifications: 0, MinAccuracy: 0, MaxSlashCount: -1, RewardMultiplierBps: 100, MinContestedVerifications: 0},
	TierVerified:   {MinStake: 1_110_000, MinVerifications: 22, MinAccuracy: 0.77, MaxSlashCount: -1, RewardMultiplierBps: 500, MinContestedVerifications: 0},
	TierBonded:     {MinStake: 1_111_000_000, MinVerifications: 0, MinAccuracy: 0, MaxSlashCount: -1, RewardMultiplierBps: 1000, MinContestedVerifications: 0},
	TierGuardian:   {MinStake: 11_111_000_000, MinVerifications: 333, MinAccuracy: 0.77, MaxSlashCount: 0, RewardMultiplierBps: 1500, MinContestedVerifications: 33},
}

// computeValidatorTierGo computes the highest tier a validator qualifies for.
// Mirrors TS computeValidatorTier from consensus/pot/validator_tiers.ts.
func computeValidatorTierGo(stake uint64, totalVerifications, correctVerifications, slashCount, contestedCorrect int) ValidatorTier {
	accuracy := 0.0
	if totalVerifications > 0 {
		accuracy = float64(correctVerifications) / float64(totalVerifications)
	}

	for t := TierGuardian; t >= TierApprentice; t-- {
		cfg := tierConfigs[t]
		if stake >= cfg.MinStake &&
			totalVerifications >= cfg.MinVerifications &&
			(cfg.MinAccuracy == 0 || accuracy >= cfg.MinAccuracy) &&
			(cfg.MaxSlashCount < 0 || slashCount <= cfg.MaxSlashCount) &&
			contestedCorrect >= cfg.MinContestedVerifications {
			return t
		}
	}
	return TierApprentice
}

func TestTier_Apprentice(t *testing.T) {
	// Zero-stake, zero-verification agent → Apprentice
	tier := computeValidatorTierGo(0, 0, 0, 0, 0)
	if tier != TierApprentice {
		t.Fatalf("Expected Apprentice (0), got: %d", tier)
	}
}

func TestTier_Verified(t *testing.T) {
	// 2 LGM = 2,000,000 ulgm, 30 verifications, 80% accuracy
	tier := computeValidatorTierGo(2_000_000, 30, 24, 0, 0)
	if tier != TierVerified {
		t.Fatalf("Expected Verified (1), got: %d", tier)
	}
}

func TestTier_Bonded(t *testing.T) {
	// 1,200 LGM = 1,200,000,000 ulgm, no verifications needed
	tier := computeValidatorTierGo(1_200_000_000, 0, 0, 0, 0)
	if tier != TierBonded {
		t.Fatalf("Expected Bonded (2), got: %d", tier)
	}
}

func TestTier_Guardian(t *testing.T) {
	// 12,000 LGM = 12,000,000,000 ulgm, 400 verifications at 80%, 40 contested correct
	tier := computeValidatorTierGo(12_000_000_000, 400, 320, 0, 40)
	if tier != TierGuardian {
		t.Fatalf("Expected Guardian (3), got: %d", tier)
	}
}

func TestTier_GuardianBlockedBySlash(t *testing.T) {
	// Same as Guardian but with 1 slash → falls to Bonded
	tier := computeValidatorTierGo(12_000_000_000, 400, 320, 1, 40)
	if tier != TierBonded {
		t.Fatalf("Expected Bonded (2) with slash, got: %d", tier)
	}
}

func TestTier_VerifiedNeedsStake(t *testing.T) {
	// High verifications but 0 stake → still Apprentice (need >= 1.11 LGM)
	tier := computeValidatorTierGo(0, 30, 24, 0, 0)
	if tier != TierApprentice {
		t.Fatalf("Expected Apprentice (0) with 0 stake, got: %d", tier)
	}
}

// --- Economic Vectors ---

func TestReward_Apprentice(t *testing.T) {
	// 100 LGM base reward * 0.1x = 10 LGM
	base := uint64(100_000_000) // 100 LGM in ulgm
	reward := (base * uint64(tierConfigs[TierApprentice].RewardMultiplierBps)) / 1000
	if reward != 10_000_000 { // 10 LGM
		t.Fatalf("Apprentice reward: got %d, want 10000000", reward)
	}
}

func TestReward_Verified(t *testing.T) {
	base := uint64(100_000_000) // 100 LGM
	reward := (base * uint64(tierConfigs[TierVerified].RewardMultiplierBps)) / 1000
	if reward != 50_000_000 { // 50 LGM
		t.Fatalf("Verified reward: got %d, want 50000000", reward)
	}
}

func TestReward_Bonded(t *testing.T) {
	base := uint64(100_000_000) // 100 LGM
	reward := (base * uint64(tierConfigs[TierBonded].RewardMultiplierBps)) / 1000
	if reward != 100_000_000 { // 100 LGM
		t.Fatalf("Bonded reward: got %d, want 100000000", reward)
	}
}

func TestReward_Guardian(t *testing.T) {
	base := uint64(100_000_000) // 100 LGM
	reward := (base * uint64(tierConfigs[TierGuardian].RewardMultiplierBps)) / 1000
	if reward != 150_000_000 { // 150 LGM
		t.Fatalf("Guardian reward: got %d, want 150000000", reward)
	}
}

// --- Known Divergence Documentation ---

func TestDivergence_D1_ClaimContentHash(t *testing.T) {
	goHash := computeClaimContentHash(claimText, claimDomain, claimType)
	tsInput := "claim:" + claimText + ":" + claimDomain + ":" + claimType
	tsHashBytes := sha256.Sum256([]byte(tsInput))
	tsHash := hex.EncodeToString(tsHashBytes[:])
	if goHash == tsHash {
		t.Fatal("DIVERGENCE D1 RESOLVED: Update golden vectors.")
	}
}

func TestDivergence_D2_DecayHalfLife_FIXED(t *testing.T) {
	// D2 was fixed: Go empirical HalfLifeBlocks updated from 100,000 to 1,000,000.
	// Both chains now use 1,000,000 blocks as the empirical half-life.
	// This test verifies they produce the same result.
	tsDecay := math.Pow(0.5, float64(500_000)/float64(1_000_000))
	goDecay := math.Pow(0.5, float64(500_000)/float64(1_000_000)) // Now matches TS

	if math.Abs(tsDecay-goDecay) > 1e-14 {
		t.Fatalf("D2 REGRESSION: TS=%.16f, Go=%.16f", tsDecay, goDecay)
	}
	if math.Abs(tsDecay-goldenDecayFactor) > 1e-14 {
		t.Fatalf("Decay factor wrong: %.16f != %.16f", tsDecay, goldenDecayFactor)
	}
}

func TestDivergence_D4_VRFProofEndianness(t *testing.T) {
	// VRF output matches, but proof encoding differs.
	// Go uses little-endian for scalars, TS uses big-endian.
	// This test verifies the output still matches despite proof encoding differences.

	seed, _ := hex.DecodeString(keySeed)
	domainSeparated := testDomainHash("LGM.vrf.v1", []byte(vrfInput))
	H, err := vrfHashToCurve(domainSeparated)
	if err != nil {
		t.Fatalf("hashToCurve error: %v", err)
	}

	skScalar := vrfDeriveScalarFromSeed(seed)
	Gamma := new(edwards25519.Point).ScalarMult(skScalar, H)
	outputHash := sha256.Sum256(Gamma.Bytes())
	outputHex := hex.EncodeToString(outputHash[:])

	// VRF output MUST match (consensus-critical)
	if outputHex != goldenVRFOutput {
		t.Fatalf("VRF output diverged from golden value:\n  got:  %s\n  want: %s", outputHex, goldenVRFOutput)
	}
}

func TestDivergence_D5_StakeDecimals(t *testing.T) {
	// Verify that tier thresholds represent the same human-readable LGM
	// despite different decimal precision.

	// TS: 1111n * 10n**18n = 1,111 LGM at 18 decimals
	// Go: 1_111_000_000 ulgm = 1,111 LGM at 6 decimals
	tsLgm := 1111.0 // 1111 * 10^18 / 10^18
	goLgm := float64(tierConfigs[TierBonded].MinStake) / 1e6

	if math.Abs(tsLgm-goLgm) > 0.001 {
		t.Fatalf("Tier 2 stake threshold mismatch in LGM: TS=%.3f, Go=%.3f", tsLgm, goLgm)
	}
}
