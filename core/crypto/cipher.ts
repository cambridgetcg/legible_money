/**
 * Symmetric Encryption for Agent Memory
 *
 * AES-256-GCM authenticated encryption for protecting agent state at rest.
 * Key derivation uses domain-separated BLAKE3 from the agent's operational key.
 *
 * Security properties:
 * - Authenticated encryption (AES-256-GCM): confidentiality + integrity
 * - Unique IV per encryption: prevents nonce reuse
 * - Domain-separated key derivation: memory key ≠ signing key
 * - Tamper detection via GCM auth tag
 *
 * Wire format: [iv (12 bytes)] [authTag (16 bytes)] [ciphertext (variable)]
 */

import { randomBytes, createCipheriv, createDecipheriv } from "crypto";
import { blake3Hash } from "./hashing.js";

// ============ Constants ============

const IV_LENGTH = 12; // GCM standard
const AUTH_TAG_LENGTH = 16; // 128-bit auth tag
const KEY_LENGTH = 32; // AES-256
const ALGORITHM = "aes-256-gcm" as const;

/** Domain separator for memory encryption key derivation */
const MEMORY_KEY_DOMAIN = "LGM.memory.encrypt.v1";

// ============ Key Derivation ============

/**
 * Derive an AES-256 encryption key from an agent's key material.
 *
 * Uses BLAKE3 with domain separation to ensure the encryption key
 * is cryptographically independent from the signing key.
 *
 * @param keyMaterial - Agent's operational private key or identity seed (32+ bytes)
 * @returns 32-byte AES-256 key
 */
export function deriveMemoryEncryptionKey(keyMaterial: Uint8Array): Uint8Array {
  const domainBytes = new TextEncoder().encode(MEMORY_KEY_DOMAIN);
  const input = new Uint8Array(domainBytes.length + keyMaterial.length);
  input.set(domainBytes, 0);
  input.set(keyMaterial, domainBytes.length);
  // BLAKE3 output is 32 bytes by default = exactly AES-256 key size
  return blake3Hash(input);
}

// ============ Encrypt / Decrypt ============

/**
 * Encrypt plaintext with AES-256-GCM.
 *
 * @param plaintext - Data to encrypt
 * @param key - 32-byte AES-256 key (from deriveMemoryEncryptionKey)
 * @returns Wire-format ciphertext: [iv (12)] [authTag (16)] [ciphertext]
 */
export function encrypt(plaintext: Uint8Array, key: Uint8Array): Uint8Array {
  if (key.length !== KEY_LENGTH) {
    throw new Error(
      `Encryption key must be ${KEY_LENGTH} bytes, got ${key.length}`,
    );
  }

  const iv = randomBytes(IV_LENGTH);
  const cipher = createCipheriv(ALGORITHM, key, iv);

  const encrypted = Buffer.concat([cipher.update(plaintext), cipher.final()]);
  const authTag = cipher.getAuthTag();

  // Wire format: iv + authTag + ciphertext
  const result = new Uint8Array(IV_LENGTH + AUTH_TAG_LENGTH + encrypted.length);
  result.set(iv, 0);
  result.set(authTag, IV_LENGTH);
  result.set(encrypted, IV_LENGTH + AUTH_TAG_LENGTH);

  return result;
}

/**
 * Decrypt AES-256-GCM ciphertext.
 *
 * @param ciphertext - Wire-format from encrypt(): [iv (12)] [authTag (16)] [ciphertext]
 * @param key - 32-byte AES-256 key (same key used for encryption)
 * @returns Decrypted plaintext
 * @throws Error if authentication fails (tampered data) or format is invalid
 */
export function decrypt(ciphertext: Uint8Array, key: Uint8Array): Uint8Array {
  if (key.length !== KEY_LENGTH) {
    throw new Error(
      `Decryption key must be ${KEY_LENGTH} bytes, got ${key.length}`,
    );
  }

  const minLength = IV_LENGTH + AUTH_TAG_LENGTH;
  if (ciphertext.length < minLength) {
    throw new Error(
      `Ciphertext too short: ${ciphertext.length} bytes (minimum ${minLength})`,
    );
  }

  const iv = ciphertext.slice(0, IV_LENGTH);
  const authTag = ciphertext.slice(IV_LENGTH, IV_LENGTH + AUTH_TAG_LENGTH);
  const encrypted = ciphertext.slice(IV_LENGTH + AUTH_TAG_LENGTH);

  const decipher = createDecipheriv(ALGORITHM, key, iv);
  decipher.setAuthTag(authTag);

  const decrypted = Buffer.concat([
    decipher.update(encrypted),
    decipher.final(), // Throws if auth tag doesn't match
  ]);

  return new Uint8Array(decrypted);
}

// ============ Convenience: JSON Encrypt/Decrypt ============

/**
 * Encrypt a JSON-serializable object.
 * BigInt values are converted to strings during serialization.
 */
export function encryptJSON(data: unknown, key: Uint8Array): Uint8Array {
  const json = JSON.stringify(data, (_k, v) =>
    typeof v === "bigint" ? `__bigint:${v.toString()}` : v,
  );
  const plaintext = new TextEncoder().encode(json);
  return encrypt(plaintext, key);
}

/**
 * Decrypt and parse a JSON object.
 * BigInt strings (prefixed __bigint:) are restored.
 */
export function decryptJSON<T>(ciphertext: Uint8Array, key: Uint8Array): T {
  const plaintext = decrypt(ciphertext, key);
  const json = new TextDecoder().decode(plaintext);
  return JSON.parse(json, (_k, v) => {
    if (typeof v === "string" && v.startsWith("__bigint:")) {
      return BigInt(v.slice("__bigint:".length));
    }
    return v;
  }) as T;
}
