import * as ed from "@noble/ed25519";
import { sha512 } from "@noble/hashes/sha512";

// Configure ed25519 to use sha512 (required for sync operations in v3.x)
// Must set hashes.sha512 for synchronous operations
ed.hashes.sha512 = (message: Uint8Array) => sha512(message);

// ============ Type Definitions ============
// All types are hex-encoded strings for JSON serialization compatibility
// Internal operations convert to/from Uint8Array as needed

/** Hex-encoded private key (64 chars = 32 bytes) */
export type PrivateKey = string;

/** Hex-encoded public key (64 chars = 32 bytes) */
export type PublicKey = string;

/** Hex-encoded signature (128 chars = 64 bytes) */
export type Signature = string;

/** Raw byte representations for internal use */
export type PrivateKeyBytes = Uint8Array;
export type PublicKeyBytes = Uint8Array;
export type SignatureBytes = Uint8Array;

export interface Keypair {
  privateKey: PrivateKey;
  publicKey: PublicKey;
}

export interface KeypairBytes {
  privateKey: PrivateKeyBytes;
  publicKey: PublicKeyBytes;
}

// ============ Key Generation ============

/**
 * Generate a new Ed25519 keypair for an agent
 * Returns hex-encoded keys for easy storage/serialization
 */
export function generateKeypair(): Keypair {
  const privateKeyBytes = ed.utils.randomSecretKey();
  const publicKeyBytes = ed.getPublicKey(privateKeyBytes);
  return {
    privateKey: bytesToHex(privateKeyBytes),
    publicKey: bytesToHex(publicKeyBytes),
  };
}

/**
 * Generate a new keypair returning raw bytes (for internal use)
 */
export function generateKeypairBytes(): KeypairBytes {
  const privateKey = ed.utils.randomSecretKey();
  const publicKey = ed.getPublicKey(privateKey);
  return { privateKey, publicKey };
}

/**
 * Generate a keypair from a seed (deterministic)
 * @param seed - 32-byte seed for deterministic generation
 * @returns Keypair derived from seed
 */
export function generateKeypairFromSeed(seed: Uint8Array): Keypair {
  if (seed.length !== 32) {
    throw new Error("Seed must be exactly 32 bytes");
  }
  // Use the seed as the private key
  const privateKeyBytes = seed;
  const publicKeyBytes = ed.getPublicKey(privateKeyBytes);
  return {
    privateKey: bytesToHex(privateKeyBytes),
    publicKey: bytesToHex(publicKeyBytes),
  };
}

/**
 * Generate a keypair from a seed (raw bytes)
 */
export function generateKeypairBytesFromSeed(seed: Uint8Array): KeypairBytes {
  if (seed.length !== 32) {
    throw new Error("Seed must be exactly 32 bytes");
  }
  const privateKey = seed;
  const publicKey = ed.getPublicKey(privateKey);
  return { privateKey, publicKey };
}

/**
 * Derive public key from private key
 */
export function getPublicKey(privateKey: PrivateKey): PublicKey {
  const privateKeyBytes = hexToBytes(privateKey);
  const publicKeyBytes = ed.getPublicKey(privateKeyBytes);
  return bytesToHex(publicKeyBytes);
}

/**
 * Derive public key from private key bytes
 */
export function getPublicKeyFromBytes(
  privateKey: PrivateKeyBytes,
): PublicKeyBytes {
  return ed.getPublicKey(privateKey);
}

// ============ Signing ============

/**
 * Sign a message with a private key
 * @param message - Raw bytes to sign
 * @param privateKey - Hex-encoded private key
 * @returns Hex-encoded signature
 */
export function sign(message: Uint8Array, privateKey: PrivateKey): Signature {
  const privateKeyBytes = hexToBytes(privateKey);
  const signatureBytes = ed.sign(message, privateKeyBytes);
  return bytesToHex(signatureBytes);
}

/**
 * Sign a message using raw byte keys (for internal use)
 */
export function signBytes(
  message: Uint8Array,
  privateKey: PrivateKeyBytes,
): SignatureBytes {
  return ed.sign(message, privateKey);
}

/**
 * Sign a string message
 */
export function signString(message: string, privateKey: PrivateKey): Signature {
  const messageBytes = new TextEncoder().encode(message);
  return sign(messageBytes, privateKey);
}

// ============ Verification ============

/**
 * Verify a signature against a message and public key
 * @param signature - Hex-encoded signature
 * @param message - Raw bytes that were signed
 * @param publicKey - Hex-encoded public key
 */
export function verify(
  signature: Signature,
  message: Uint8Array,
  publicKey: PublicKey,
): boolean {
  const signatureBytes = hexToBytes(signature);
  const publicKeyBytes = hexToBytes(publicKey);
  return ed.verify(signatureBytes, message, publicKeyBytes);
}

/**
 * Verify using raw bytes (for internal use)
 */
export function verifyBytes(
  signature: SignatureBytes,
  message: Uint8Array,
  publicKey: PublicKeyBytes,
): boolean {
  return ed.verify(signature, message, publicKey);
}

/**
 * Convert public key to agent address (first 20 bytes of hash)
 */
export function publicKeyToAddress(publicKey: PublicKey): string {
  const hash = sha512(publicKey);
  const addressBytes = hash.slice(0, 20);
  return "lgm1" + bytesToHex(addressBytes);
}

/**
 * Helper: bytes to hex string
 */
export function bytesToHex(bytes: Uint8Array): string {
  return Array.from(bytes)
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
}

/**
 * Helper: hex string to bytes
 */
export function hexToBytes(hex: string): Uint8Array {
  const bytes = new Uint8Array(hex.length / 2);
  for (let i = 0; i < hex.length; i += 2) {
    bytes[i / 2] = parseInt(hex.slice(i, i + 2), 16);
  }
  return bytes;
}
