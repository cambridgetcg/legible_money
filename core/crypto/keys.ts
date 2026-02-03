import * as ed from '@noble/ed25519';
import { sha512 } from '@noble/hashes/sha512';

// Configure ed25519 to use sha512
ed.etc.sha512Sync = (...m) => sha512(ed.etc.concatBytes(...m));

export type PrivateKey = Uint8Array;
export type PublicKey = Uint8Array;
export type Signature = Uint8Array;

export interface Keypair {
  privateKey: PrivateKey;
  publicKey: PublicKey;
}

/**
 * Generate a new Ed25519 keypair for an agent
 */
export function generateKeypair(): Keypair {
  const privateKey = ed.utils.randomPrivateKey();
  const publicKey = ed.getPublicKey(privateKey);
  return { privateKey, publicKey };
}

/**
 * Derive public key from private key
 */
export function getPublicKey(privateKey: PrivateKey): PublicKey {
  return ed.getPublicKey(privateKey);
}

/**
 * Sign a message with a private key
 */
export function sign(message: Uint8Array, privateKey: PrivateKey): Signature {
  return ed.sign(message, privateKey);
}

/**
 * Verify a signature against a message and public key
 */
export function verify(
  signature: Signature,
  message: Uint8Array,
  publicKey: PublicKey
): boolean {
  return ed.verify(signature, message, publicKey);
}

/**
 * Convert public key to agent address (first 20 bytes of hash)
 */
export function publicKeyToAddress(publicKey: PublicKey): string {
  const hash = sha512(publicKey);
  const addressBytes = hash.slice(0, 20);
  return 'lgm1' + bytesToHex(addressBytes);
}

/**
 * Helper: bytes to hex string
 */
export function bytesToHex(bytes: Uint8Array): string {
  return Array.from(bytes)
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('');
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
