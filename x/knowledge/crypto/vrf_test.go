package crypto

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"math"
	"testing"
)

func TestDomainHash_LengthPrefixFormat(t *testing.T) {
	// Verify domainHash uses the length-prefix format: SHA-256(domain || uint32BE(len(domain)) || data)
	domain := "LGM.vrf.v1"
	data := []byte("test-data")

	result := domainHash(domain, data)

	// Manually compute expected hash
	domainBytes := []byte(domain)
	lengthBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBytes, uint32(len(domainBytes)))

	h := sha256.New()
	h.Write(domainBytes)
	h.Write(lengthBytes)
	h.Write(data)
	expected := h.Sum(nil)

	if hex.EncodeToString(result) != hex.EncodeToString(expected) {
		t.Fatalf("domainHash does not match length-prefix format: got %x, want %x", result, expected)
	}

	// Verify it does NOT match the old colon format
	hOld := sha256.New()
	hOld.Write(domainBytes)
	hOld.Write([]byte(":"))
	hOld.Write(data)
	oldFormat := hOld.Sum(nil)

	if hex.EncodeToString(result) == hex.EncodeToString(oldFormat) {
		t.Fatal("domainHash should NOT match old colon-separator format")
	}
}

func TestDomainHash_NoAmbiguity(t *testing.T) {
	// The old colon format was ambiguous: domain="a:", data="b" == domain="a", data=":b"
	// Length-prefix format should produce distinct hashes.
	hash1 := domainHash("a:", []byte("b"))
	hash2 := domainHash("a", []byte(":b"))

	if hex.EncodeToString(hash1) == hex.EncodeToString(hash2) {
		t.Fatal("domainHash should not produce collisions for different domain/data splits")
	}
}

func TestGenerateVerifyVRF_Roundtrip(t *testing.T) {
	// Generate a keypair
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	seed := []byte("test-claim-verification-seed")

	// Use priv directly (NOT a copy) — GenerateVRF must not mutate the key
	output, proof, err := GenerateVRF(seed, priv)
	if err != nil {
		t.Fatalf("GenerateVRF failed: %v", err)
	}

	if len(output) != 32 {
		t.Fatalf("VRF output should be 32 bytes, got %d", len(output))
	}
	if len(proof) != 96 {
		t.Fatalf("VRF proof should be 96 bytes, got %d", len(proof))
	}

	// priv.Public() should still return the correct key (not corrupted by GenerateVRF)
	pub := priv.Public().(ed25519.PublicKey)
	verifiedOutput, valid := VerifyVRF(seed, pub, proof)
	if !valid {
		t.Fatal("VRF verification failed for valid proof")
	}
	if hex.EncodeToString(verifiedOutput) != hex.EncodeToString(output) {
		t.Fatal("VRF verified output does not match generated output")
	}

	// Verify with wrong seed fails
	_, valid = VerifyVRF([]byte("wrong-seed"), pub, proof)
	if valid {
		t.Fatal("VRF verification should fail with wrong seed")
	}

	// Verify with wrong key fails
	_, priv2, _ := ed25519.GenerateKey(nil)
	_, valid = VerifyVRF(seed, priv2.Public().(ed25519.PublicKey), proof)
	if valid {
		t.Fatal("VRF verification should fail with wrong public key")
	}
}

func TestGenerateVRF_Deterministic(t *testing.T) {
	// VRF uniqueness: same key + same seed must always produce identical output and proof
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	seed := []byte("determinism-test-seed")

	output1, proof1, err := GenerateVRF(seed, priv)
	if err != nil {
		t.Fatal(err)
	}

	output2, proof2, err := GenerateVRF(seed, priv)
	if err != nil {
		t.Fatal(err)
	}

	if hex.EncodeToString(output1) != hex.EncodeToString(output2) {
		t.Fatal("VRF output should be deterministic for same key+seed")
	}
	if hex.EncodeToString(proof1) != hex.EncodeToString(proof2) {
		t.Fatal("VRF proof should be deterministic for same key+seed")
	}
}

func TestGenerateVRF_DifferentSeedsDifferentOutputs(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	output1, _, err := GenerateVRF([]byte("seed-alpha"), priv)
	if err != nil {
		t.Fatal(err)
	}

	output2, _, err := GenerateVRF([]byte("seed-beta"), priv)
	if err != nil {
		t.Fatal(err)
	}

	if hex.EncodeToString(output1) == hex.EncodeToString(output2) {
		t.Fatal("different seeds should produce different VRF outputs")
	}
}

func TestGenerateVRF_32ByteKeyPath(t *testing.T) {
	// 32-byte seed should produce the same output as the 64-byte key derived from it
	_, priv64, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	seed := []byte("32-byte-key-test")

	output64, proof64, err := GenerateVRF(seed, priv64)
	if err != nil {
		t.Fatal(err)
	}

	// Use just the 32-byte seed portion
	output32, proof32, err := GenerateVRF(seed, priv64[:32])
	if err != nil {
		t.Fatal(err)
	}

	if hex.EncodeToString(output64) != hex.EncodeToString(output32) {
		t.Fatal("32-byte and 64-byte key should produce identical VRF output")
	}
	if hex.EncodeToString(proof64) != hex.EncodeToString(proof32) {
		t.Fatal("32-byte and 64-byte key should produce identical VRF proof")
	}
}

func TestGenerateVRF_InvalidKeyLength(t *testing.T) {
	seed := []byte("test")

	_, _, err := GenerateVRF(seed, make([]byte, 16))
	if err == nil {
		t.Fatal("should reject 16-byte key")
	}

	_, _, err = GenerateVRF(seed, make([]byte, 48))
	if err == nil {
		t.Fatal("should reject 48-byte key")
	}
}

func TestVerifyVRF_TamperedProof_Gamma(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	seed := []byte("tamper-test")

	_, proof, err := GenerateVRF(seed, priv)
	if err != nil {
		t.Fatal(err)
	}
	pub := priv.Public().(ed25519.PublicKey)

	// Flip a bit in Gamma (bytes 0-31)
	tampered := make([]byte, len(proof))
	copy(tampered, proof)
	tampered[0] ^= 0x01

	_, valid := VerifyVRF(seed, pub, tampered)
	if valid {
		t.Fatal("VRF should reject proof with tampered Gamma")
	}
}

func TestVerifyVRF_TamperedProof_Challenge(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	seed := []byte("tamper-test-c")

	_, proof, err := GenerateVRF(seed, priv)
	if err != nil {
		t.Fatal(err)
	}
	pub := priv.Public().(ed25519.PublicKey)

	// Flip a bit in c (bytes 32-63)
	tampered := make([]byte, len(proof))
	copy(tampered, proof)
	tampered[40] ^= 0x01

	_, valid := VerifyVRF(seed, pub, tampered)
	if valid {
		t.Fatal("VRF should reject proof with tampered challenge scalar")
	}
}

func TestVerifyVRF_TamperedProof_S(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	seed := []byte("tamper-test-s")

	_, proof, err := GenerateVRF(seed, priv)
	if err != nil {
		t.Fatal(err)
	}
	pub := priv.Public().(ed25519.PublicKey)

	// Flip a bit in s (bytes 64-95)
	tampered := make([]byte, len(proof))
	copy(tampered, proof)
	tampered[70] ^= 0x01

	_, valid := VerifyVRF(seed, pub, tampered)
	if valid {
		t.Fatal("VRF should reject proof with tampered s scalar")
	}
}

func TestVerifyVRF_InvalidProofLength(t *testing.T) {
	_, valid := VerifyVRF([]byte("seed"), make([]byte, 32), make([]byte, 64))
	if valid {
		t.Fatal("should reject 64-byte proof (need 96)")
	}

	_, valid = VerifyVRF([]byte("seed"), make([]byte, 32), make([]byte, 128))
	if valid {
		t.Fatal("should reject 128-byte proof (need 96)")
	}
}

func TestVerifyVRF_InvalidPublicKeyLength(t *testing.T) {
	_, valid := VerifyVRF([]byte("seed"), make([]byte, 16), make([]byte, 96))
	if valid {
		t.Fatal("should reject 16-byte public key (need 32)")
	}
}

func TestGenerateVRFSeed_Deterministic(t *testing.T) {
	prevHash := make([]byte, 32)
	for i := range prevHash {
		prevHash[i] = byte(i)
	}

	seed1 := GenerateVRFSeed("claim-123", 100, prevHash)
	seed2 := GenerateVRFSeed("claim-123", 100, prevHash)

	if hex.EncodeToString(seed1) != hex.EncodeToString(seed2) {
		t.Fatal("GenerateVRFSeed should be deterministic")
	}

	// Different claim ID → different seed
	seed3 := GenerateVRFSeed("claim-456", 100, prevHash)
	if hex.EncodeToString(seed1) == hex.EncodeToString(seed3) {
		t.Fatal("Different claim IDs should produce different seeds")
	}

	// Different block number → different seed
	seed4 := GenerateVRFSeed("claim-123", 101, prevHash)
	if hex.EncodeToString(seed1) == hex.EncodeToString(seed4) {
		t.Fatal("Different block numbers should produce different seeds")
	}
}

func TestGenerateBlockSeed_Deterministic(t *testing.T) {
	prevHash := make([]byte, 32)
	for i := range prevHash {
		prevHash[i] = byte(i)
	}

	seed1 := GenerateBlockSeed(prevHash, 100, 5)
	seed2 := GenerateBlockSeed(prevHash, 100, 5)

	if hex.EncodeToString(seed1) != hex.EncodeToString(seed2) {
		t.Fatal("GenerateBlockSeed should be deterministic")
	}

	// Different epoch → different seed
	seed3 := GenerateBlockSeed(prevHash, 100, 6)
	if hex.EncodeToString(seed1) == hex.EncodeToString(seed3) {
		t.Fatal("Different epochs should produce different seeds")
	}
}

func TestIsValidatorSelected_StakeWeighted(t *testing.T) {
	// With 100% of stake, should always be selected
	output := make([]byte, 32)
	output[0] = 0x7F // Mid-range
	selected, _ := IsValidatorSelected(output, 1000, 1000, 1)
	if !selected {
		t.Fatal("Validator with 100% stake should be selected")
	}

	// With 0 stake, never selected
	selected, _ = IsValidatorSelected(output, 0, 1000, 1)
	if selected {
		t.Fatal("Validator with 0 stake should not be selected")
	}
}

func TestIsValidatorSelected_Boundaries(t *testing.T) {
	// Edge: stake == totalStake, targetCount > 1
	output := make([]byte, 32)
	output[0] = 0xFF // Max range
	output[1] = 0xFF
	output[2] = 0xFF
	output[3] = 0xFF
	output[4] = 0xFF
	output[5] = 0xFF
	output[6] = 0xFF
	output[7] = 0xFF
	selected, _ := IsValidatorSelected(output, 1000, 1000, 3)
	if !selected {
		t.Fatal("100% stake with targetCount=3 should always be selected")
	}

	// Edge: zero totalStake
	selected, _ = IsValidatorSelected(output, 100, 0, 1)
	if selected {
		t.Fatal("zero totalStake should not select anyone")
	}

	// Edge: short output (< 8 bytes)
	selected, _ = IsValidatorSelected(make([]byte, 4), 100, 1000, 1)
	if selected {
		t.Fatal("output shorter than 8 bytes should not select")
	}
}

func TestIsValidatorSelected_MaxUint64(t *testing.T) {
	// Exercise the big.Int overflow protection path with max uint64 values
	output := make([]byte, 32)
	binary.BigEndian.PutUint64(output[:8], math.MaxUint64/2)

	// This should not panic despite large values
	selected, priority := IsValidatorSelected(output, math.MaxUint64, math.MaxUint64, 1)
	if !selected {
		t.Fatal("100% stake should be selected even with max uint64 values")
	}
	if priority != math.MaxUint64/2 {
		t.Fatalf("priority should match output value, got %d", priority)
	}
}

func TestIsValidatorSelected_StakeClamp(t *testing.T) {
	// If stake > totalStake (shouldn't happen but defensive), clamp it
	output := make([]byte, 32)
	output[0] = 0x7F
	selected, _ := IsValidatorSelected(output, 2000, 1000, 1)
	if !selected {
		t.Fatal("clamped stake (2000→1000 = 100%) should be selected")
	}
}

func TestHashToCurve_DoesNotAliasInput(t *testing.T) {
	// Verify M1 fix: hashToCurve must not mutate caller's buffer via append aliasing
	input := make([]byte, 32, 64) // cap > len — would trigger aliasing with append
	for i := range input {
		input[i] = byte(i)
	}
	// Write a sentinel in the spare capacity
	spare := input[32:64]
	for i := range spare {
		spare[i] = 0xAA
	}

	_, err := hashToCurve(input)
	if err != nil {
		t.Fatal(err)
	}

	// Check spare capacity was NOT overwritten
	for i, b := range spare {
		if b != 0xAA {
			t.Fatalf("hashToCurve aliased caller memory at spare[%d]: got 0x%02x, want 0xAA", i, b)
		}
	}
}

func TestGenerateVRF_DoesNotMutatePrivateKey(t *testing.T) {
	// Verify that GenerateVRF does not overwrite the public key portion
	// of a 64-byte Ed25519 private key via append aliasing
	_, priv, _ := ed25519.GenerateKey(nil)

	// Save a copy of the key before calling GenerateVRF
	origKey := make([]byte, len(priv))
	copy(origKey, priv)

	_, _, err := GenerateVRF([]byte("mutation-test"), priv)
	if err != nil {
		t.Fatal(err)
	}

	if hex.EncodeToString(priv) != hex.EncodeToString(origKey) {
		t.Fatal("GenerateVRF mutated the caller's private key")
	}
}
