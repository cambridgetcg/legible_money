package crypto

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
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
