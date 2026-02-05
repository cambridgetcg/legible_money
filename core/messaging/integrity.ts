/**
 * Message Integrity
 *
 * Provides cryptographic integrity for TruthMessages:
 * - Content-addressed message IDs (SHA-512/256)
 * - Ed25519 signatures with domain separation
 * - Signature verification
 * - Canonical serialization for deterministic hashing
 *
 * All TruthMessages must be signed by the sender and can be
 * verified by any recipient using the sender's public key.
 */

import { domainHash } from "../crypto/hashing.js";
import { bytesToHex } from "../crypto/keys.js";
import type { Signature, PublicKey, PrivateKey } from "../crypto/keys.js";
import type { TruthMessage, TruthPayload } from "./types.js";

// ============ Constants ============

/**
 * Domain separation tags for truth messages
 */
export const TRUTH_MESSAGE_DOMAINS = {
  /** Domain for message ID generation */
  MESSAGE_ID: "LGM.truth.message.id.v1",

  /** Domain for message signing */
  SIGNATURE: "LGM.truth.message.sig.v1",

  /** Domain for content hashing */
  CONTENT: "LGM.truth.message.content.v1",

  /** Domain for commitment generation */
  COMMITMENT: "LGM.truth.commitment.v1",
} as const;

// ============ Canonical Serialization ============

/**
 * Canonicalize an object for deterministic hashing
 *
 * Ensures the same logical object always produces the same hash
 * regardless of property insertion order.
 *
 * @param obj - Object to canonicalize
 * @returns Deterministic JSON string
 */
export function canonicalize(obj: unknown): string {
  return JSON.stringify(obj, (_, value) => {
    // Handle bigint serialization
    if (typeof value === "bigint") {
      return value.toString() + "n";
    }
    // Handle Uint8Array serialization
    if (value instanceof Uint8Array) {
      return Array.from(value);
    }
    // Handle Map serialization
    if (value instanceof Map) {
      return Object.fromEntries(value);
    }
    // Handle Set serialization
    if (value instanceof Set) {
      return Array.from(value);
    }
    // Sort object keys for determinism
    if (value && typeof value === "object" && !Array.isArray(value)) {
      return Object.keys(value)
        .sort()
        .reduce((sorted: Record<string, unknown>, key) => {
          sorted[key] = (value as Record<string, unknown>)[key];
          return sorted;
        }, {});
    }
    return value;
  });
}

/**
 * Parse a canonical string back to an object
 *
 * Handles bigint and Uint8Array deserialization.
 *
 * @param str - Canonical JSON string
 * @returns Parsed object
 */
export function parseCanonical<T>(str: string): T {
  return JSON.parse(str, (_, value) => {
    // Handle bigint deserialization
    if (typeof value === "string" && value.endsWith("n")) {
      const numStr = value.slice(0, -1);
      if (/^-?\d+$/.test(numStr)) {
        return BigInt(numStr);
      }
    }
    return value;
  });
}

// ============ Message ID Generation ============

/**
 * Generate a content-addressed message ID
 *
 * The message ID is a hash of the message content (excluding signature
 * and messageId itself), ensuring the same content always produces
 * the same ID.
 *
 * @param message - Message without messageId and signature
 * @returns Content-addressed message ID
 */
export function generateMessageId(
  message: Omit<TruthMessage, "messageId" | "signature" | "contentHash">,
): string {
  const content = canonicalize({
    messageType: message.messageType,
    version: message.version,
    sender: message.sender,
    recipient: message.recipient,
    payload: message.payload,
    referencedMessages: message.referencedMessages,
    causedBy: message.causedBy,
    networkFeeUnit: message.networkFeeUnit,
    networkFeeLimit: message.networkFeeLimit,
    timestamp: message.timestamp,
    blockHeight: message.blockHeight,
    domain: message.domain,
    tags: message.tags,
    ttlBlocks: message.ttlBlocks,
  });

  const hashBytes = domainHash(TRUTH_MESSAGE_DOMAINS.MESSAGE_ID, content);
  const hash = bytesToHex(hashBytes);
  return `msg_${hash.slice(0, 32)}`;
}

/**
 * Generate content hash for a payload
 *
 * @param payload - Message payload
 * @returns SHA-512/256 hash of canonicalized payload
 */
export function generateContentHash(payload: TruthPayload): string {
  const content = canonicalize(payload);
  return bytesToHex(domainHash(TRUTH_MESSAGE_DOMAINS.CONTENT, content));
}

// ============ Signing ============

/**
 * Sign function type (matches core/crypto/signatures.ts)
 */
type SignFunction = (message: string, privateKey: PrivateKey) => Signature;

/**
 * Verify function type (matches core/crypto/signatures.ts)
 */
type VerifyFunction = (
  message: string,
  signature: Signature,
  publicKey: PublicKey,
) => boolean;

/**
 * Prepare a message for signing
 *
 * Creates the canonical representation that will be signed.
 *
 * @param message - Message to prepare (without signature)
 * @returns Canonical string to sign
 */
export function prepareForSigning(
  message: Omit<TruthMessage, "signature">,
): string {
  const toSign = {
    messageId: message.messageId,
    messageType: message.messageType,
    version: message.version,
    sender: message.sender,
    recipient: message.recipient,
    contentHash: message.contentHash,
    referencedMessages: message.referencedMessages,
    causedBy: message.causedBy,
    networkFeeUnit: message.networkFeeUnit,
    networkFeeLimit: message.networkFeeLimit,
    timestamp: message.timestamp,
    blockHeight: message.blockHeight,
    domain: message.domain,
    tags: message.tags,
    ttlBlocks: message.ttlBlocks,
  };

  return bytesToHex(
    domainHash(TRUTH_MESSAGE_DOMAINS.SIGNATURE, canonicalize(toSign)),
  );
}

/**
 * Sign a TruthMessage
 *
 * Adds messageId, contentHash, and signature to the message.
 *
 * @param message - Message without signature fields
 * @param privateKey - Sender's private key
 * @param signFn - Signing function (injected for flexibility)
 * @returns Signed message
 */
export function signTruthMessage<T extends TruthMessage>(
  message: Omit<T, "messageId" | "signature" | "contentHash">,
  privateKey: PrivateKey,
  signFn: SignFunction,
): T {
  // Generate content hash
  const contentHash = generateContentHash(message.payload as TruthPayload);

  // Generate message ID
  const messageId = generateMessageId(message);

  // Prepare message with ID and content hash
  const messageToSign = {
    ...message,
    messageId,
    contentHash,
  };

  // Generate signature
  const signData = prepareForSigning(messageToSign);
  const signature = signFn(signData, privateKey);

  return {
    ...messageToSign,
    signature,
  } as T;
}

/**
 * Verify a TruthMessage signature
 *
 * @param message - Signed message to verify
 * @param publicKey - Sender's public key
 * @param verifyFn - Verification function (injected for flexibility)
 * @returns True if signature is valid
 */
export function verifyTruthMessageSignature(
  message: TruthMessage,
  publicKey: PublicKey,
  verifyFn: VerifyFunction,
): boolean {
  // Reconstruct what was signed
  const signData = prepareForSigning(message);

  // Verify signature
  return verifyFn(signData, message.signature, publicKey);
}

/**
 * Verify message ID matches content
 *
 * Ensures the messageId is correctly derived from the content.
 *
 * @param message - Message to verify
 * @returns True if messageId matches content
 */
export function verifyMessageId(message: TruthMessage): boolean {
  const { messageId, signature, contentHash, ...rest } = message;
  const expectedId = generateMessageId(
    rest as Omit<TruthMessage, "messageId" | "signature" | "contentHash">,
  );
  return messageId === expectedId;
}

/**
 * Verify content hash matches payload
 *
 * @param message - Message to verify
 * @returns True if contentHash matches payload
 */
export function verifyContentHash(message: TruthMessage): boolean {
  const expectedHash = generateContentHash(message.payload);
  return message.contentHash === expectedHash;
}

// ============ Full Integrity Verification ============

/**
 * Result of integrity verification
 */
export interface IntegrityVerificationResult {
  /** Overall validity */
  valid: boolean;

  /** Individual check results */
  checks: {
    messageIdValid: boolean;
    contentHashValid: boolean;
    signatureValid: boolean;
  };

  /** Errors if any checks failed */
  errors: string[];
}

/**
 * Verify full message integrity
 *
 * Checks:
 * 1. Message ID matches content
 * 2. Content hash matches payload
 * 3. Signature is valid
 *
 * @param message - Message to verify
 * @param publicKey - Sender's public key
 * @param verifyFn - Verification function
 * @returns Verification result
 */
export function verifyMessageIntegrity(
  message: TruthMessage,
  publicKey: PublicKey,
  verifyFn: VerifyFunction,
): IntegrityVerificationResult {
  const errors: string[] = [];

  // Check message ID
  const messageIdValid = verifyMessageId(message);
  if (!messageIdValid) {
    errors.push("Message ID does not match content");
  }

  // Check content hash
  const contentHashValid = verifyContentHash(message);
  if (!contentHashValid) {
    errors.push("Content hash does not match payload");
  }

  // Check signature
  const signatureValid = verifyTruthMessageSignature(
    message,
    publicKey,
    verifyFn,
  );
  if (!signatureValid) {
    errors.push("Signature verification failed");
  }

  return {
    valid: messageIdValid && contentHashValid && signatureValid,
    checks: {
      messageIdValid,
      contentHashValid,
      signatureValid,
    },
    errors,
  };
}

// ============ Commitment Generation ============

/**
 * Generate a commitment for verification (commit-reveal)
 *
 * Used in VerificationCommitMessage to hide the verdict until reveal.
 *
 * @param verdict - The verdict (confirmed, rejected, uncertain)
 * @param confidence - Confidence level
 * @param nonce - Random nonce
 * @param referencesChecked - Evidence IDs examined
 * @returns Commitment hash
 */
export function generateVerificationCommitment(
  verdict: string,
  confidence: number,
  nonce: string,
  referencesChecked: string[],
): string {
  const data = canonicalize({
    verdict,
    confidence,
    nonce,
    referencesChecked: referencesChecked.sort(),
  });

  return bytesToHex(domainHash(TRUTH_MESSAGE_DOMAINS.COMMITMENT, data));
}

/**
 * Verify a verification commitment matches the reveal
 *
 * @param commitment - Original commitment hash
 * @param verdict - Revealed verdict
 * @param confidence - Revealed confidence
 * @param nonce - Revealed nonce
 * @param referencesChecked - Revealed references
 * @returns True if reveal matches commitment
 */
export function verifyVerificationCommitment(
  commitment: string,
  verdict: string,
  confidence: number,
  nonce: string,
  referencesChecked: string[],
): boolean {
  const expectedCommitment = generateVerificationCommitment(
    verdict,
    confidence,
    nonce,
    referencesChecked,
  );
  return commitment === expectedCommitment;
}

// ============ Nonce Generation ============

/**
 * Generate a random nonce for commitments
 *
 * @returns Random 32-character hex string
 */
export function generateNonce(): string {
  const bytes = new Uint8Array(16);
  crypto.getRandomValues(bytes);
  return Array.from(bytes)
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
}
