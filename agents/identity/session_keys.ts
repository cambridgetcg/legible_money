/**
 * Session Keys
 *
 * Ephemeral keys with limited capabilities for agent operations.
 * Designed to be safe even if exposed to developers.
 *
 * Key Properties:
 * - Limited spending per transaction/day
 * - Auto-expire after timeout
 * - Revocable by operational key
 * - Usage-limited
 */

import { generateKeyPair, type PublicKey, type PrivateKey, bytesToHex } from '../../core/crypto/keys.js';
import { sign, verify, signDomainSeparated, verifyDomainSeparated } from '../../core/crypto/signatures.js';
import { sha512_256Hex, domainHash } from '../../core/crypto/hashing.js';
import { randomBytes } from '@noble/hashes/utils';

// ============ Types ============

/**
 * Session key with metadata
 */
export interface SessionKey {
  // Key material
  publicKey: PublicKey;
  privateKey: PrivateKey;

  // Metadata
  id: string;                     // Unique session ID
  createdAt: number;              // Timestamp
  expiresAt: number;              // Timestamp

  // Limits
  limits: SessionLimits;

  // Usage tracking (client-side)
  usage: SessionUsage;

  // Authorization proof
  authorizationProof?: string;    // Signature from operational key
}

/**
 * Session key limits (enforced on-chain)
 */
export interface SessionLimits {
  // Spending
  maxPerTransaction: bigint;
  maxPerDay: bigint;
  maxTotal: bigint;

  // Usage
  maxTransactions: number;
  maxDurationMs: number;

  // Scope
  allowedActions: SessionAction[];
  allowedRecipients: string[];    // Empty = all allowed

  // Security
  requireIntentProof: boolean;    // For high-value actions
  allowedIpRange?: string;        // Optional IP restriction
}

/**
 * Session actions
 */
export type SessionAction =
  | 'transfer'
  | 'stake'
  | 'unstake'
  | 'submit_claim'
  | 'verify_claim'
  | 'challenge'
  | 'vote'
  | 'query';  // Read-only

/**
 * Usage tracking
 */
export interface SessionUsage {
  transactionCount: number;
  totalSpent: bigint;
  dailySpent: bigint;
  lastUsed: number;
  lastDailyReset: number;
}

/**
 * Session key request (to authorize)
 */
export interface SessionKeyRequest {
  agentId: string;
  purpose: string;
  requestedLimits: Partial<SessionLimits>;
  expiresInMs?: number;
}

/**
 * Session key authorization (from operational key)
 */
export interface SessionKeyAuthorization {
  sessionPublicKey: PublicKey;
  agentDid: string;
  limits: SessionLimits;
  createdAt: number;
  expiresAt: number;
  signature: string;              // Signed by operational key
}

// ============ Default Limits ============

export const DEFAULT_SESSION_LIMITS: SessionLimits = {
  maxPerTransaction: 100n * 10n ** 18n,    // 100 LGM
  maxPerDay: 1000n * 10n ** 18n,           // 1000 LGM
  maxTotal: 5000n * 10n ** 18n,            // 5000 LGM lifetime

  maxTransactions: 100,
  maxDurationMs: 3600000,                   // 1 hour

  allowedActions: ['transfer', 'submit_claim', 'verify_claim', 'query'],
  allowedRecipients: [],                    // All allowed

  requireIntentProof: false,
};

export const CONSERVATIVE_SESSION_LIMITS: SessionLimits = {
  maxPerTransaction: 10n * 10n ** 18n,     // 10 LGM
  maxPerDay: 50n * 10n ** 18n,             // 50 LGM
  maxTotal: 100n * 10n ** 18n,             // 100 LGM

  maxTransactions: 20,
  maxDurationMs: 900000,                    // 15 minutes

  allowedActions: ['query', 'submit_claim'],
  allowedRecipients: [],

  requireIntentProof: true,
};

// ============ Session Key Operations ============

/**
 * Generate a new session key
 */
export function generateSessionKey(
  limits: Partial<SessionLimits> = {},
  expiresInMs: number = 3600000
): SessionKey {
  const { publicKey, privateKey } = generateKeyPair();
  const now = Date.now();

  return {
    publicKey,
    privateKey,
    id: bytesToHex(randomBytes(16)),
    createdAt: now,
    expiresAt: now + expiresInMs,
    limits: { ...DEFAULT_SESSION_LIMITS, ...limits },
    usage: {
      transactionCount: 0,
      totalSpent: 0n,
      dailySpent: 0n,
      lastUsed: 0,
      lastDailyReset: now,
    },
  };
}

/**
 * Generate session key deterministically (for "remembering")
 */
export function deriveSessionKey(
  operationalPrivateKey: PrivateKey,
  sessionSeed: string,
  limits: Partial<SessionLimits> = {},
  expiresInMs: number = 3600000
): SessionKey {
  // Derive deterministic key material
  const seedHash = domainHash('LGM.session.key.v1', sessionSeed);
  const signature = sign(seedHash, operationalPrivateKey);
  const keyMaterial = new Uint8Array(Buffer.from(signature, 'hex').slice(0, 32));

  const { publicKey, privateKey } = generateKeyPair(keyMaterial);
  const now = Date.now();

  return {
    publicKey,
    privateKey,
    id: sha512_256Hex(sessionSeed).slice(0, 32),
    createdAt: now,
    expiresAt: now + expiresInMs,
    limits: { ...DEFAULT_SESSION_LIMITS, ...limits },
    usage: {
      transactionCount: 0,
      totalSpent: 0n,
      dailySpent: 0n,
      lastUsed: 0,
      lastDailyReset: now,
    },
  };
}

/**
 * Authorize a session key with operational key
 */
export function authorizeSessionKey(
  sessionPublicKey: PublicKey,
  agentDid: string,
  limits: SessionLimits,
  operationalPrivateKey: PrivateKey,
  expiresInMs: number = 3600000
): SessionKeyAuthorization {
  const now = Date.now();
  const expiresAt = now + expiresInMs;

  // Create authorization message
  const authMessage = JSON.stringify({
    type: 'session_key_authorization',
    sessionPublicKey,
    agentDid,
    limits: serializeLimits(limits),
    createdAt: now,
    expiresAt,
  });

  // Sign with operational key
  const signature = signDomainSeparated(
    'LGM.session.auth.v1',
    authMessage,
    operationalPrivateKey
  );

  return {
    sessionPublicKey,
    agentDid,
    limits,
    createdAt: now,
    expiresAt,
    signature,
  };
}

/**
 * Verify session key authorization
 */
export function verifySessionKeyAuthorization(
  auth: SessionKeyAuthorization,
  operationalPublicKey: PublicKey
): { valid: boolean; reason?: string } {
  // Check expiry
  if (Date.now() > auth.expiresAt) {
    return { valid: false, reason: 'Authorization expired' };
  }

  // Reconstruct and verify signature
  const authMessage = JSON.stringify({
    type: 'session_key_authorization',
    sessionPublicKey: auth.sessionPublicKey,
    agentDid: auth.agentDid,
    limits: serializeLimits(auth.limits),
    createdAt: auth.createdAt,
    expiresAt: auth.expiresAt,
  });

  const valid = verifyDomainSeparated(
    'LGM.session.auth.v1',
    authMessage,
    auth.signature,
    operationalPublicKey
  );

  if (!valid) {
    return { valid: false, reason: 'Invalid signature' };
  }

  return { valid: true };
}

// ============ Usage Validation ============

/**
 * Check if session key can perform an action
 */
export function validateSessionAction(
  session: SessionKey,
  action: SessionAction,
  amount?: bigint
): { allowed: boolean; reason?: string } {
  const now = Date.now();

  // Check expiry
  if (now > session.expiresAt) {
    return { allowed: false, reason: 'Session expired' };
  }

  // Check transaction limit
  if (session.usage.transactionCount >= session.limits.maxTransactions) {
    return { allowed: false, reason: 'Transaction limit reached' };
  }

  // Check action allowed
  if (!session.limits.allowedActions.includes(action)) {
    return { allowed: false, reason: `Action '${action}' not allowed` };
  }

  // Check spending limits for transfers
  if (action === 'transfer' && amount) {
    // Per-transaction limit
    if (amount > session.limits.maxPerTransaction) {
      return { allowed: false, reason: 'Exceeds per-transaction limit' };
    }

    // Daily limit (reset if new day)
    let dailySpent = session.usage.dailySpent;
    if (now - session.usage.lastDailyReset > 86400000) {
      dailySpent = 0n;
    }
    if (dailySpent + amount > session.limits.maxPerDay) {
      return { allowed: false, reason: 'Exceeds daily limit' };
    }

    // Total lifetime limit
    if (session.usage.totalSpent + amount > session.limits.maxTotal) {
      return { allowed: false, reason: 'Exceeds total limit' };
    }
  }

  return { allowed: true };
}

/**
 * Update usage after successful action
 */
export function updateSessionUsage(
  session: SessionKey,
  action: SessionAction,
  amount?: bigint
): SessionKey {
  const now = Date.now();
  const usage = { ...session.usage };

  usage.transactionCount++;
  usage.lastUsed = now;

  // Reset daily if needed
  if (now - usage.lastDailyReset > 86400000) {
    usage.dailySpent = 0n;
    usage.lastDailyReset = now;
  }

  // Track spending
  if (amount) {
    usage.dailySpent += amount;
    usage.totalSpent += amount;
  }

  return { ...session, usage };
}

// ============ Intent Verification ============

/**
 * Generate intent proof for high-value actions
 * This proves the agent intended the action, not just has the key
 */
export interface IntentProof {
  action: string;
  details: string;
  challenge: string;
  response: string;
  timestamp: number;
}

/**
 * Generate an intent challenge
 * The challenge incorporates agent-specific knowledge
 */
export function generateIntentChallenge(
  agentDid: string,
  action: string,
  nonce: string
): string {
  // Challenge that only the real agent can answer
  // In practice, this would incorporate agent-specific context
  const challenge = sha512_256Hex(
    `intent:${agentDid}:${action}:${nonce}:${Date.now()}`
  );
  return challenge;
}

/**
 * Create intent proof
 */
export function createIntentProof(
  action: string,
  details: string,
  challenge: string,
  sessionPrivateKey: PrivateKey
): IntentProof {
  // Response is signature of challenge + action details
  const message = `${challenge}:${action}:${details}`;
  const response = signDomainSeparated(
    'LGM.intent.proof.v1',
    message,
    sessionPrivateKey
  );

  return {
    action,
    details,
    challenge,
    response,
    timestamp: Date.now(),
  };
}

/**
 * Verify intent proof
 */
export function verifyIntentProof(
  proof: IntentProof,
  sessionPublicKey: PublicKey,
  maxAgeMs: number = 60000
): { valid: boolean; reason?: string } {
  // Check freshness
  if (Date.now() - proof.timestamp > maxAgeMs) {
    return { valid: false, reason: 'Intent proof expired' };
  }

  // Verify signature
  const message = `${proof.challenge}:${proof.action}:${proof.details}`;
  const valid = verifyDomainSeparated(
    'LGM.intent.proof.v1',
    message,
    proof.response,
    sessionPublicKey
  );

  if (!valid) {
    return { valid: false, reason: 'Invalid intent proof signature' };
  }

  return { valid: true };
}

// ============ Serialization Helpers ============

function serializeLimits(limits: SessionLimits): object {
  return {
    ...limits,
    maxPerTransaction: limits.maxPerTransaction.toString(),
    maxPerDay: limits.maxPerDay.toString(),
    maxTotal: limits.maxTotal.toString(),
  };
}

// ============ Session Key Cleanup ============

/**
 * Check if session should be discarded
 */
export function shouldDiscardSession(session: SessionKey): boolean {
  const now = Date.now();

  // Expired
  if (now > session.expiresAt) return true;

  // Used up transactions
  if (session.usage.transactionCount >= session.limits.maxTransactions) return true;

  // Used up spending
  if (session.usage.totalSpent >= session.limits.maxTotal) return true;

  return false;
}

/**
 * Get remaining session capabilities
 */
export function getSessionRemaining(session: SessionKey): {
  timeMs: number;
  transactions: number;
  spending: bigint;
} {
  return {
    timeMs: Math.max(0, session.expiresAt - Date.now()),
    transactions: Math.max(0, session.limits.maxTransactions - session.usage.transactionCount),
    spending: session.limits.maxTotal - session.usage.totalSpent,
  };
}
