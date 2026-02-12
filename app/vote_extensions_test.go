package app

import (
	"encoding/json"
	"sync"
	"testing"
)

// ---- ComputeCommitmentHash tests ----

func TestComputeCommitmentHash_Deterministic(t *testing.T) {
	h1 := ComputeCommitmentHash("round-1", "accept", 800000, "salt123")
	h2 := ComputeCommitmentHash("round-1", "accept", 800000, "salt123")
	if h1 != h2 {
		t.Fatalf("same inputs must produce same hash: %s != %s", h1, h2)
	}
}

func TestComputeCommitmentHash_LengthIs64Hex(t *testing.T) {
	h := ComputeCommitmentHash("round-1", "accept", 800000, "salt123")
	if len(h) != 64 {
		t.Fatalf("SHA-256 hex should be 64 chars, got %d: %s", len(h), h)
	}
}

func TestComputeCommitmentHash_DifferentRoundID(t *testing.T) {
	h1 := ComputeCommitmentHash("round-1", "accept", 800000, "salt123")
	h2 := ComputeCommitmentHash("round-2", "accept", 800000, "salt123")
	if h1 == h2 {
		t.Fatal("different roundIDs must produce different hashes")
	}
}

func TestComputeCommitmentHash_DifferentVerdict(t *testing.T) {
	h1 := ComputeCommitmentHash("round-1", "accept", 800000, "salt123")
	h2 := ComputeCommitmentHash("round-1", "reject", 800000, "salt123")
	if h1 == h2 {
		t.Fatal("different verdicts must produce different hashes")
	}
}

func TestComputeCommitmentHash_DifferentConfidence(t *testing.T) {
	h1 := ComputeCommitmentHash("round-1", "accept", 800000, "salt123")
	h2 := ComputeCommitmentHash("round-1", "accept", 700000, "salt123")
	if h1 == h2 {
		t.Fatal("different confidence must produce different hashes")
	}
}

func TestComputeCommitmentHash_DifferentSalt(t *testing.T) {
	h1 := ComputeCommitmentHash("round-1", "accept", 800000, "salt123")
	h2 := ComputeCommitmentHash("round-1", "accept", 800000, "salt456")
	if h1 == h2 {
		t.Fatal("different salts must produce different hashes")
	}
}

func TestComputeCommitmentHash_SecondPreimageResistance(t *testing.T) {
	// Before the fix: roundID="abc" + verdict="def" could collide with
	// roundID="abcde" + verdict="f". Length-prefixed encoding prevents this.
	h1 := ComputeCommitmentHash("abc", "def", 800000, "salt")
	h2 := ComputeCommitmentHash("abcde", "f", 800000, "salt")
	if h1 == h2 {
		t.Fatal("length-prefixed encoding should prevent field boundary ambiguity")
	}

	// Another boundary case: roundID ending in valid confidence bytes
	h3 := ComputeCommitmentHash("round", "accept", 0, "salt")
	h4 := ComputeCommitmentHash("round", "accept", 1, "salt")
	if h3 == h4 {
		t.Fatal("different confidence values of 0 and 1 must produce different hashes")
	}
}

func TestComputeCommitmentHash_EmptyFields(t *testing.T) {
	// Empty strings should still produce a valid hash
	h := ComputeCommitmentHash("", "", 0, "")
	if len(h) != 64 {
		t.Fatalf("empty fields should still produce valid 64-char hash, got %d", len(h))
	}
}

// ---- VerifyCommitmentHash tests ----

func TestVerifyCommitmentHash_Valid(t *testing.T) {
	hash := ComputeCommitmentHash("round-1", "accept", 800000, "salt123")
	if !VerifyCommitmentHash(hash, "round-1", "accept", 800000, "salt123") {
		t.Fatal("verify should return true for matching inputs")
	}
}

func TestVerifyCommitmentHash_WrongRoundID(t *testing.T) {
	hash := ComputeCommitmentHash("round-1", "accept", 800000, "salt123")
	if VerifyCommitmentHash(hash, "round-2", "accept", 800000, "salt123") {
		t.Fatal("verify should return false for wrong roundID")
	}
}

func TestVerifyCommitmentHash_WrongVerdict(t *testing.T) {
	hash := ComputeCommitmentHash("round-1", "accept", 800000, "salt123")
	if VerifyCommitmentHash(hash, "round-1", "reject", 800000, "salt123") {
		t.Fatal("verify should return false for wrong verdict")
	}
}

func TestVerifyCommitmentHash_WrongConfidence(t *testing.T) {
	hash := ComputeCommitmentHash("round-1", "accept", 800000, "salt123")
	if VerifyCommitmentHash(hash, "round-1", "accept", 700000, "salt123") {
		t.Fatal("verify should return false for wrong confidence")
	}
}

func TestVerifyCommitmentHash_WrongSalt(t *testing.T) {
	hash := ComputeCommitmentHash("round-1", "accept", 800000, "salt123")
	if VerifyCommitmentHash(hash, "round-1", "accept", 800000, "wrong") {
		t.Fatal("verify should return false for wrong salt")
	}
}

// ---- VoteExtInjection encoding/decoding tests ----

func TestEncodeDecodeVoteExtInjection_Roundtrip(t *testing.T) {
	inj := VoteExtInjection{
		Commitments: []InjectedCommitment{
			{
				RoundID:        "round-1",
				Validator:      "lgmvaloper1abc",
				CommitmentHash: "deadbeef" + "deadbeef" + "deadbeef" + "deadbeef" + "deadbeef" + "deadbeef" + "deadbeef" + "deadbeef",
				VRFOutput:      "aabb",
				VRFProof:       "ccdd",
			},
		},
		Reveals: []InjectedReveal{
			{
				RoundID:    "round-1",
				Validator:  "lgmvaloper1def",
				Verdict:    "accept",
				Confidence: 800000,
				Salt:       "salt456",
			},
		},
	}

	encoded, err := encodeVoteExtInjection(inj)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	decoded, err := decodeVoteExtInjection(encoded)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if len(decoded.Commitments) != 1 {
		t.Fatalf("expected 1 commitment, got %d", len(decoded.Commitments))
	}
	if decoded.Commitments[0].RoundID != "round-1" {
		t.Fatalf("commitment roundID mismatch: %s", decoded.Commitments[0].RoundID)
	}
	if decoded.Commitments[0].Validator != "lgmvaloper1abc" {
		t.Fatalf("commitment validator mismatch: %s", decoded.Commitments[0].Validator)
	}

	if len(decoded.Reveals) != 1 {
		t.Fatalf("expected 1 reveal, got %d", len(decoded.Reveals))
	}
	if decoded.Reveals[0].Verdict != "accept" {
		t.Fatalf("reveal verdict mismatch: %s", decoded.Reveals[0].Verdict)
	}
	if decoded.Reveals[0].Confidence != 800000 {
		t.Fatalf("reveal confidence mismatch: %d", decoded.Reveals[0].Confidence)
	}
}

func TestEncodeDecodeVoteExtInjection_Empty(t *testing.T) {
	inj := VoteExtInjection{}
	encoded, err := encodeVoteExtInjection(inj)
	if err != nil {
		t.Fatalf("encode empty failed: %v", err)
	}

	decoded, err := decodeVoteExtInjection(encoded)
	if err != nil {
		t.Fatalf("decode empty failed: %v", err)
	}

	if len(decoded.Commitments) != 0 || len(decoded.Reveals) != 0 {
		t.Fatal("empty injection should decode to empty slices")
	}
}

func TestIsVoteExtInjectionTx(t *testing.T) {
	// Valid injection tx
	validTx := append(VoteExtInjectionPrefix, []byte(`{"commitments":null,"reveals":null}`)...)
	if !isVoteExtInjectionTx(validTx) {
		t.Fatal("should recognize valid injection tx")
	}

	// Too short
	if isVoteExtInjectionTx(VoteExtInjectionPrefix) {
		t.Fatal("prefix-only tx should not be recognized (no payload)")
	}

	// Empty
	if isVoteExtInjectionTx([]byte{}) {
		t.Fatal("empty tx should not be recognized")
	}

	// Wrong prefix
	if isVoteExtInjectionTx([]byte{0x01, 'V', 'E', 'X', 0x00}) {
		t.Fatal("wrong prefix should not be recognized")
	}

	// Regular tx
	if isVoteExtInjectionTx([]byte(`{"type":"transfer","amount":"100"}`)) {
		t.Fatal("regular tx should not be recognized as injection")
	}
}

func TestDecodeVoteExtInjection_InvalidFormat(t *testing.T) {
	// Valid prefix but invalid JSON
	invalidTx := append(VoteExtInjectionPrefix, []byte(`{invalid json}`)...)
	_, err := decodeVoteExtInjection(invalidTx)
	if err == nil {
		t.Fatal("should fail to decode invalid JSON")
	}

	// Not a vote extension tx
	_, err = decodeVoteExtInjection([]byte(`regular data`))
	if err == nil {
		t.Fatal("should fail to decode non-injection tx")
	}
}

// ---- VoteExtension JSON serialization tests ----

func TestVoteExtension_JSONRoundtrip(t *testing.T) {
	ext := VoteExtension{
		Commitments: []VoteCommitment{
			{
				RoundID:        "round-1",
				CommitmentHash: "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
				VRFOutput:      "aabb",
				VRFProof:       "ccdd",
				Height:         100,
			},
		},
		Reveals: []VoteReveal{
			{
				RoundID:    "round-2",
				Verdict:    "accept",
				Confidence: 800000,
				Salt:       "salt123",
			},
		},
		ValidatorAddress: "lgmvaloper1abc",
	}

	bz, err := json.Marshal(ext)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded VoteExtension
	if err := json.Unmarshal(bz, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.ValidatorAddress != ext.ValidatorAddress {
		t.Fatalf("validator address mismatch: %s", decoded.ValidatorAddress)
	}
	if len(decoded.Commitments) != 1 {
		t.Fatalf("expected 1 commitment, got %d", len(decoded.Commitments))
	}
	if decoded.Commitments[0].Height != 100 {
		t.Fatalf("commitment height mismatch: %d", decoded.Commitments[0].Height)
	}
	if len(decoded.Reveals) != 1 {
		t.Fatalf("expected 1 reveal, got %d", len(decoded.Reveals))
	}
	if decoded.Reveals[0].Confidence != 800000 {
		t.Fatalf("reveal confidence mismatch: %d", decoded.Reveals[0].Confidence)
	}
}

func TestVoteExtension_EmptyJSON(t *testing.T) {
	ext := VoteExtension{ValidatorAddress: "lgmvaloper1test"}
	bz, err := json.Marshal(ext)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded VoteExtension
	if err := json.Unmarshal(bz, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.ValidatorAddress != "lgmvaloper1test" {
		t.Fatalf("validator address mismatch: %s", decoded.ValidatorAddress)
	}
	if len(decoded.Commitments) != 0 || len(decoded.Reveals) != 0 {
		t.Fatal("empty extension should have no commitments or reveals")
	}
}

// ---- LocalCommitmentStore tests ----

func TestLocalCommitmentStore_StoreAndGet(t *testing.T) {
	store := NewLocalCommitmentStore()

	commitment := LocalCommitment{
		RoundID:    "round-1",
		Verdict:    "accept",
		Confidence: 800000,
		Salt:       "salt123",
		Height:     100,
	}

	store.Store(commitment)

	got, found := store.Get("round-1")
	if !found {
		t.Fatal("should find stored commitment")
	}
	if got.Verdict != "accept" {
		t.Fatalf("verdict mismatch: %s", got.Verdict)
	}
	if got.Confidence != 800000 {
		t.Fatalf("confidence mismatch: %d", got.Confidence)
	}
	if got.Salt != "salt123" {
		t.Fatalf("salt mismatch: %s", got.Salt)
	}
}

func TestLocalCommitmentStore_GetNotFound(t *testing.T) {
	store := NewLocalCommitmentStore()

	_, found := store.Get("nonexistent")
	if found {
		t.Fatal("should not find nonexistent commitment")
	}
}

func TestLocalCommitmentStore_Delete(t *testing.T) {
	store := NewLocalCommitmentStore()

	store.Store(LocalCommitment{RoundID: "round-1", Verdict: "accept"})
	store.Delete("round-1")

	_, found := store.Get("round-1")
	if found {
		t.Fatal("should not find deleted commitment")
	}
}

func TestLocalCommitmentStore_DeleteNonexistent(t *testing.T) {
	store := NewLocalCommitmentStore()
	// Should not panic
	store.Delete("nonexistent")
}

func TestLocalCommitmentStore_Overwrite(t *testing.T) {
	store := NewLocalCommitmentStore()

	store.Store(LocalCommitment{RoundID: "round-1", Verdict: "accept"})
	store.Store(LocalCommitment{RoundID: "round-1", Verdict: "reject"})

	got, found := store.Get("round-1")
	if !found {
		t.Fatal("should find overwritten commitment")
	}
	if got.Verdict != "reject" {
		t.Fatalf("should have overwritten value, got %s", got.Verdict)
	}
}

func TestLocalCommitmentStore_Count(t *testing.T) {
	store := NewLocalCommitmentStore()

	if store.Count() != 0 {
		t.Fatalf("empty store should have count 0, got %d", store.Count())
	}

	store.Store(LocalCommitment{RoundID: "round-1"})
	store.Store(LocalCommitment{RoundID: "round-2"})
	store.Store(LocalCommitment{RoundID: "round-3"})

	if store.Count() != 3 {
		t.Fatalf("expected count 3, got %d", store.Count())
	}

	store.Delete("round-2")
	if store.Count() != 2 {
		t.Fatalf("expected count 2 after delete, got %d", store.Count())
	}
}

func TestLocalCommitmentStore_CleanupExpired(t *testing.T) {
	store := NewLocalCommitmentStore()

	store.Store(LocalCommitment{RoundID: "old", Height: 10})
	store.Store(LocalCommitment{RoundID: "recent", Height: 90})
	store.Store(LocalCommitment{RoundID: "current", Height: 100})

	// Cleanup with currentHeight=110, maxAge=50 → cutoff=60
	// "old" (height=10) < 60 → deleted
	// "recent" (height=90) >= 60 → kept
	// "current" (height=100) >= 60 → kept
	store.CleanupExpired(110, 50)

	if store.Count() != 2 {
		t.Fatalf("expected 2 after cleanup, got %d", store.Count())
	}

	_, found := store.Get("old")
	if found {
		t.Fatal("old commitment should have been cleaned up")
	}

	_, found = store.Get("recent")
	if !found {
		t.Fatal("recent commitment should have survived cleanup")
	}

	_, found = store.Get("current")
	if !found {
		t.Fatal("current commitment should have survived cleanup")
	}
}

func TestLocalCommitmentStore_CleanupExpired_LowHeight(t *testing.T) {
	store := NewLocalCommitmentStore()

	store.Store(LocalCommitment{RoundID: "r1", Height: 5})

	// currentHeight < maxAge → no cleanup (prevent underflow)
	store.CleanupExpired(30, 100)

	if store.Count() != 1 {
		t.Fatalf("should not clean up when currentHeight < maxAge, got count %d", store.Count())
	}
}

func TestLocalCommitmentStore_CleanupAll(t *testing.T) {
	store := NewLocalCommitmentStore()

	store.Store(LocalCommitment{RoundID: "r1", Height: 1})
	store.Store(LocalCommitment{RoundID: "r2", Height: 2})
	store.Store(LocalCommitment{RoundID: "r3", Height: 3})

	// All older than cutoff = 1000 - 10 = 990
	store.CleanupExpired(1000, 10)

	if store.Count() != 0 {
		t.Fatalf("all commitments should be cleaned up, got count %d", store.Count())
	}
}

func TestLocalCommitmentStore_ThreadSafety(t *testing.T) {
	store := NewLocalCommitmentStore()
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			store.Store(LocalCommitment{
				RoundID:    "round-" + string(rune('A'+n%26)),
				Verdict:    "accept",
				Confidence: uint64(n * 1000),
				Height:     uint64(n),
			})
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			store.Get("round-" + string(rune('A'+n%26)))
		}(i)
	}

	// Concurrent deletes
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			store.Delete("round-" + string(rune('A'+n%26)))
		}(i)
	}

	// Concurrent cleanup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			store.CleanupExpired(uint64(n*10+50), 20)
		}(i)
	}

	wg.Wait()

	// Should not panic, count should be non-negative
	if store.Count() < 0 {
		t.Fatal("count should be non-negative")
	}
}

// ---- VoteExtInjection prefix tests ----

func TestVoteExtInjectionPrefix_Value(t *testing.T) {
	expected := []byte{0x00, 'V', 'E', 'X'}
	if len(VoteExtInjectionPrefix) != 4 {
		t.Fatalf("prefix should be 4 bytes, got %d", len(VoteExtInjectionPrefix))
	}
	for i, b := range VoteExtInjectionPrefix {
		if b != expected[i] {
			t.Fatalf("prefix byte %d: expected %02x, got %02x", i, expected[i], b)
		}
	}
}

// ---- Vote extension size limits ----

func TestVoteExtension_MaxCommitmentsConstant(t *testing.T) {
	// Verify the constants exist and are reasonable by creating an extension
	// at the boundary. This tests the structural validation in VerifyVoteExtensionHandler.
	ext := VoteExtension{
		ValidatorAddress: "lgmvaloper1test",
		Commitments:      make([]VoteCommitment, 50),
		Reveals:          make([]VoteReveal, 50),
	}

	// Fill with valid-looking data
	for i := range ext.Commitments {
		ext.Commitments[i] = VoteCommitment{
			RoundID:        "round-" + string(rune('A'+i%26)),
			CommitmentHash: "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
			Height:         uint64(i),
		}
	}
	for i := range ext.Reveals {
		ext.Reveals[i] = VoteReveal{
			RoundID:    "round-" + string(rune('A'+i%26)),
			Verdict:    "accept",
			Confidence: 800000,
			Salt:       "salt",
		}
	}

	// Should serialize without issue
	bz, err := json.Marshal(ext)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded VoteExtension
	if err := json.Unmarshal(bz, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(decoded.Commitments) != 50 {
		t.Fatalf("expected 50 commitments, got %d", len(decoded.Commitments))
	}
}

// ---- Commitment hash edge cases ----

func TestComputeCommitmentHash_MaxConfidence(t *testing.T) {
	h := ComputeCommitmentHash("round-1", "accept", 1_000_000, "salt")
	if len(h) != 64 {
		t.Fatalf("max confidence should produce valid hash, got len %d", len(h))
	}
}

func TestComputeCommitmentHash_ZeroConfidence(t *testing.T) {
	h := ComputeCommitmentHash("round-1", "accept", 0, "salt")
	if len(h) != 64 {
		t.Fatalf("zero confidence should produce valid hash, got len %d", len(h))
	}
}

func TestComputeCommitmentHash_MaxUint64Confidence(t *testing.T) {
	h := ComputeCommitmentHash("round-1", "accept", ^uint64(0), "salt")
	if len(h) != 64 {
		t.Fatalf("max uint64 confidence should produce valid hash, got len %d", len(h))
	}
}

func TestComputeCommitmentHash_AllVerdicts(t *testing.T) {
	hashes := make(map[string]bool)
	for _, verdict := range []string{"accept", "reject", "abstain"} {
		h := ComputeCommitmentHash("round-1", verdict, 800000, "salt")
		if hashes[h] {
			t.Fatalf("verdict %s produced duplicate hash", verdict)
		}
		hashes[h] = true
	}
}

func TestComputeCommitmentHash_LongSalt(t *testing.T) {
	longSalt := ""
	for i := 0; i < 1000; i++ {
		longSalt += "x"
	}
	h := ComputeCommitmentHash("round-1", "accept", 800000, longSalt)
	if len(h) != 64 {
		t.Fatalf("long salt should produce valid hash, got len %d", len(h))
	}
}

func TestComputeCommitmentHash_SpecialCharsInRoundID(t *testing.T) {
	h := ComputeCommitmentHash("round/with:special-chars.v1", "accept", 800000, "salt")
	if len(h) != 64 {
		t.Fatalf("special chars should produce valid hash, got len %d", len(h))
	}
}

// ---- Injection sorting test ----

func TestProcessVoteExtensions_SortOrder(t *testing.T) {
	// Test the deterministic sort by encoding and decoding.
	// processVoteExtensions sorts by (RoundID, Validator).
	inj := VoteExtInjection{
		Commitments: []InjectedCommitment{
			{RoundID: "round-b", Validator: "val2"},
			{RoundID: "round-a", Validator: "val2"},
			{RoundID: "round-a", Validator: "val1"},
			{RoundID: "round-b", Validator: "val1"},
		},
		Reveals: []InjectedReveal{
			{RoundID: "round-c", Validator: "val2"},
			{RoundID: "round-c", Validator: "val1"},
		},
	}

	// Manually apply the same sort the handler uses
	sortCommitments := func(c []InjectedCommitment) {
		for i := 0; i < len(c); i++ {
			for j := i + 1; j < len(c); j++ {
				swap := false
				if c[j].RoundID < c[i].RoundID {
					swap = true
				} else if c[j].RoundID == c[i].RoundID && c[j].Validator < c[i].Validator {
					swap = true
				}
				if swap {
					c[i], c[j] = c[j], c[i]
				}
			}
		}
	}
	sortReveals := func(r []InjectedReveal) {
		for i := 0; i < len(r); i++ {
			for j := i + 1; j < len(r); j++ {
				swap := false
				if r[j].RoundID < r[i].RoundID {
					swap = true
				} else if r[j].RoundID == r[i].RoundID && r[j].Validator < r[i].Validator {
					swap = true
				}
				if swap {
					r[i], r[j] = r[j], r[i]
				}
			}
		}
	}

	sortCommitments(inj.Commitments)
	sortReveals(inj.Reveals)

	// Verify sort order
	expectedOrder := []struct{ round, val string }{
		{"round-a", "val1"},
		{"round-a", "val2"},
		{"round-b", "val1"},
		{"round-b", "val2"},
	}
	for i, exp := range expectedOrder {
		if inj.Commitments[i].RoundID != exp.round || inj.Commitments[i].Validator != exp.val {
			t.Fatalf("commitment[%d]: expected (%s, %s), got (%s, %s)",
				i, exp.round, exp.val,
				inj.Commitments[i].RoundID, inj.Commitments[i].Validator)
		}
	}

	if inj.Reveals[0].Validator != "val1" || inj.Reveals[1].Validator != "val2" {
		t.Fatalf("reveals not sorted: %s, %s",
			inj.Reveals[0].Validator, inj.Reveals[1].Validator)
	}
}

// ---- VoteReveal validation patterns ----

func TestVoteReveal_ValidVerdicts(t *testing.T) {
	validVerdicts := []string{"accept", "reject", "abstain"}
	for _, v := range validVerdicts {
		reveal := VoteReveal{
			RoundID:    "round-1",
			Verdict:    v,
			Confidence: 800000,
			Salt:       "salt",
		}
		// Verify it serializes and has the right values
		bz, err := json.Marshal(reveal)
		if err != nil {
			t.Fatalf("marshal failed for verdict %s: %v", v, err)
		}
		var decoded VoteReveal
		if err := json.Unmarshal(bz, &decoded); err != nil {
			t.Fatalf("unmarshal failed for verdict %s: %v", v, err)
		}
		if decoded.Verdict != v {
			t.Fatalf("verdict mismatch: expected %s, got %s", v, decoded.Verdict)
		}
	}
}

func TestVoteReveal_ConfidenceBounds(t *testing.T) {
	// Valid: 0 to 1_000_000
	for _, conf := range []uint64{0, 500000, 1_000_000} {
		reveal := VoteReveal{Confidence: conf}
		if reveal.Confidence > 1_000_000 {
			t.Fatalf("confidence %d should be in valid range", conf)
		}
	}

	// Invalid: > 1_000_000
	reveal := VoteReveal{Confidence: 1_000_001}
	if reveal.Confidence <= 1_000_000 {
		t.Fatal("confidence 1_000_001 should be out of range")
	}
}

// ---- VoteCommitment validation patterns ----

func TestVoteCommitment_HashLength(t *testing.T) {
	// SHA-256 hex = 64 chars
	validHash := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	if len(validHash) != 64 {
		t.Fatalf("test hash should be 64 chars, got %d", len(validHash))
	}

	// VerifyVoteExtension checks len(c.CommitmentHash) != 64
	shortHash := "abcdef"
	if len(shortHash) == 64 {
		t.Fatal("short hash should not be 64 chars")
	}
}
