/**
 * Digital Signatures
 *
 * Ed25519 signature operations for Legible Money.
 * All protocol messages and transactions are signed.
 */

import { ed25519 } from '@noble/curves/ed25519';
import { sha512 } from '@noble/hashes/sha512';
import { bytesToHex, hexToBytes, type PublicKey, type PrivateKey } from './keys.js';
import { domainHash, HASH_DOMAINS } from './hashing.js';

// ============ Types ============

export type Signature = string;  // Hex-encoded signature (128 chars = 64 bytes)
export type SignatureBytes = Uint8Array;

export interface SignedMessage<T> {
  message: T;
  signature: Signature;
  signer: PublicKey;
}

export interface SignatureVerification {
  valid: boolean;
  signer: PublicKey;
  timestamp?: number;
}

// ============ Signing ============

/**
 * Sign a message with Ed25519
 */
export function sign(message: Uint8Array, privateKey: PrivateKey): Signature {
  const keyBytes = hexToBytes(privateKey);
  const signature = ed25519.sign(message, keyBytes);
  return bytesToHex(signature);
}

/**
 * Sign a string message
 */
export function signString(message: string, privateKey: PrivateKey): Signature {
  const messageBytes = new TextEncoder().encode(message);
  return sign(messageBytes, privateKey);
}

/**
 * Sign with domain separation (recommended for protocol messages)
 */
export function signDomainSeparated(
  domain: string,
  message: Uint8Array | string,
  privateKey: PrivateKey
): Signature {
  const hash = domainHash(domain, message);
  return sign(hash, privateKey);
}

/**
 * Sign a transaction
 */
export function signTransaction(
  transaction: object,
  privateKey: PrivateKey
): Signature {
  const serialized = JSON.stringify(transaction, Object.keys(transaction).sort());
  return signDomainSeparated(HASH_DOMAINS.TRANSACTION, serialized, privateKey);
}

/**
 * Sign a block
 */
export function signBlock(
  blockHeader: object,
  privateKey: PrivateKey
): Signature {
  const serialized = JSON.stringify(blockHeader, Object.keys(blockHeader).sort());
  return signDomainSeparated(HASH_DOMAINS.BLOCK, serialized, privateKey);
}

// ============ Verification ============

/**
 * Verify an Ed25519 signature
 */
export function verify(
  message: Uint8Array,
  signature: Signature,
  publicKey: PublicKey
): boolean {
  try {
    const sigBytes = hexToBytes(signature);
    const keyBytes = hexToBytes(publicKey);
    return ed25519.verify(sigBytes, message, keyBytes);
  } catch {
    return false;
  }
}

/**
 * Verify a string message signature
 */
export function verifyString(
  message: string,
  signature: Signature,
  publicKey: PublicKey
): boolean {
  const messageBytes = new TextEncoder().encode(message);
  return verify(messageBytes, signature, publicKey);
}

/**
 * Verify with domain separation
 */
export function verifyDomainSeparated(
  domain: string,
  message: Uint8Array | string,
  signature: Signature,
  publicKey: PublicKey
): boolean {
  const hash = domainHash(domain, message);
  return verify(hash, signature, publicKey);
}

/**
 * Verify a transaction signature
 */
export function verifyTransaction(
  transaction: object,
  signature: Signature,
  publicKey: PublicKey
): boolean {
  const serialized = JSON.stringify(transaction, Object.keys(transaction).sort());
  return verifyDomainSeparated(HASH_DOMAINS.TRANSACTION, serialized, publicKey);
}

/**
 * Verify a block signature
 */
export function verifyBlock(
  blockHeader: object,
  signature: Signature,
  publicKey: PublicKey
): boolean {
  const serialized = JSON.stringify(blockHeader, Object.keys(blockHeader).sort());
  return verifyDomainSeparated(HASH_DOMAINS.BLOCK, serialized, publicKey);
}

// ============ Batch Verification ============

/**
 * Batch verify multiple signatures (more efficient)
 * Returns true only if ALL signatures are valid
 */
export function batchVerify(
  items: Array<{
    message: Uint8Array;
    signature: Signature;
    publicKey: PublicKey;
  }>
): boolean {
  // Ed25519 batch verification
  // Note: noble/curves supports this natively
  try {
    const messages = items.map(i => i.message);
    const signatures = items.map(i => hexToBytes(i.signature));
    const publicKeys = items.map(i => hexToBytes(i.publicKey));

    // Use batch verification when available
    // Falls back to individual verification
    for (let i = 0; i < items.length; i++) {
      if (!ed25519.verify(signatures[i], messages[i], publicKeys[i])) {
        return false;
      }
    }
    return true;
  } catch {
    return false;
  }
}

// ============ Signed Message Helpers ============

/**
 * Create a signed message
 */
export function createSignedMessage<T>(
  message: T,
  privateKey: PrivateKey,
  publicKey: PublicKey
): SignedMessage<T> {
  const serialized = JSON.stringify(message);
  const signature = signString(serialized, privateKey);

  return {
    message,
    signature,
    signer: publicKey,
  };
}

/**
 * Verify a signed message
 */
export function verifySignedMessage<T>(
  signedMessage: SignedMessage<T>
): SignatureVerification {
  const serialized = JSON.stringify(signedMessage.message);
  const valid = verifyString(serialized, signedMessage.signature, signedMessage.signer);

  return {
    valid,
    signer: signedMessage.signer,
  };
}

// ============ Aggregated Signatures (Future) ============

/**
 * Placeholder for aggregated signature support
 * Will be used for validator consensus
 */
export interface AggregatedSignature {
  signatures: Signature[];
  signers: PublicKey[];
  bitmap: Uint8Array;  // Which validators signed
}

/**
 * Create aggregated signature from multiple signatures
 * Note: Ed25519 doesn't natively support aggregation,
 * but we can concatenate for now and upgrade later
 */
export function aggregateSignatures(
  signatures: Array<{ signature: Signature; signer: PublicKey }>
): AggregatedSignature {
  const bitmap = new Uint8Array(Math.ceil(signatures.length / 8));

  signatures.forEach((_, i) => {
    bitmap[Math.floor(i / 8)] |= 1 << (i % 8);
  });

  return {
    signatures: signatures.map(s => s.signature),
    signers: signatures.map(s => s.signer),
    bitmap,
  };
}

/**
 * Verify aggregated signature
 */
export function verifyAggregatedSignature(
  message: Uint8Array,
  aggregated: AggregatedSignature
): boolean {
  // Verify each signature individually
  for (let i = 0; i < aggregated.signatures.length; i++) {
    if (!verify(message, aggregated.signatures[i], aggregated.signers[i])) {
      return false;
    }
  }
  return true;
}
