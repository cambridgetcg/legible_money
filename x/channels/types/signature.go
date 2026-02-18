package types

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// PackedSignatureSize is 32-byte Ed25519 pubkey + 64-byte signature.
	PackedSignatureSize = 96
	// Ed25519PubKeySize is the size of an Ed25519 public key.
	Ed25519PubKeySize = 32
	// Ed25519SigSize is the size of an Ed25519 signature.
	Ed25519SigSize = 64
)

// ChannelSigningPayload builds a canonical signing payload for channel operations.
// Format: SHA-256("legible.channels.v1:{op}" | "|" | chainId | "|" | channelId | "|" | BE(sequence) | "|" | spent)
func ChannelSigningPayload(operation, chainId, channelId string, sequence uint64, spent string) []byte {
	h := sha256.New()
	h.Write([]byte("legible.channels.v1:" + operation))
	h.Write([]byte("|"))
	h.Write([]byte(chainId))
	h.Write([]byte("|"))
	h.Write([]byte(channelId))
	h.Write([]byte("|"))
	seqBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(seqBytes, sequence)
	h.Write(seqBytes)
	h.Write([]byte("|"))
	h.Write([]byte(spent))
	return h.Sum(nil)
}

// PackSignature packs an Ed25519 public key and signature into a single byte slice.
// Layout: [32-byte pubkey][64-byte signature]
func PackSignature(pubkey ed25519.PublicKey, sig []byte) []byte {
	packed := make([]byte, PackedSignatureSize)
	copy(packed[:Ed25519PubKeySize], pubkey)
	copy(packed[Ed25519PubKeySize:], sig)
	return packed
}

// VerifyPackedSignature verifies a packed [pubkey||sig] against a canonical payload
// and checks the pubkey derives to the expected bech32 address.
func VerifyPackedSignature(payload, packed []byte, expectedAddr string) error {
	if len(packed) != PackedSignatureSize {
		return fmt.Errorf("packed signature must be %d bytes (32 pubkey + 64 sig), got %d", PackedSignatureSize, len(packed))
	}

	pubkey := ed25519.PublicKey(packed[:Ed25519PubKeySize])
	sig := packed[Ed25519PubKeySize:]

	// Derive address: SHA-256(pubkey)[:20] → bech32
	addrHash := sha256.Sum256(pubkey)
	derivedAddr := sdk.AccAddress(addrHash[:20])
	if derivedAddr.String() != expectedAddr {
		return fmt.Errorf("pubkey does not match expected address: derived %s, expected %s", derivedAddr.String(), expectedAddr)
	}

	if !ed25519.Verify(pubkey, payload, sig) {
		return ErrInvalidSignature
	}
	return nil
}

// VerifyHexPackedSignature verifies a hex-encoded packed signature.
func VerifyHexPackedSignature(payload []byte, hexSig string, expectedAddr string) error {
	if hexSig == "" {
		return fmt.Errorf("signature is empty")
	}
	packed, err := hex.DecodeString(hexSig)
	if err != nil {
		return fmt.Errorf("invalid hex signature: %w", err)
	}
	return VerifyPackedSignature(payload, packed, expectedAddr)
}
