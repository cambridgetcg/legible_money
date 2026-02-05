/**
 * Identity Service Contract
 *
 * Domain interface for agent identity and DID management.
 * Handles 4-layer key architecture, session keys, and recovery.
 *
 * MOD-1: Single Responsibility
 * Identity module changes when identity/key semantics change.
 *
 * 4-Layer Key Architecture:
 * 1. Identity Key (DID) - Immutable, derived from genesis
 * 2. Recovery Key - Shamir-split for social recovery
 * 3. Operational Key - Rotatable, day-to-day operations
 * 4. Session Key - Ephemeral, limited permissions
 */

import type {
  DID,
  PublicKey,
  Signature,
  BlockHeight,
  Amount,
} from "../types/index.js";

// ============ Identity Types ============

/**
 * Agent account structure
 */
export interface AgentAccount {
  did: DID;
  identityKey: PublicKey; // Immutable
  operationalKey: PublicKey;
  operationalKeyVersion: number;
  recoveryKeyHash: string; // Hash only, never stored
  balance: Amount;
  stakedBalance: Amount;
  nonce: bigint;
  reputationScore: number;
  createdAtBlock: BlockHeight;
  lastActiveBlock: BlockHeight;
  flags: AccountFlags;
}

/**
 * Account flags
 */
export interface AccountFlags {
  isValidator: boolean;
  canSubmitClaims: boolean;
  canChallenge: boolean;
  inRecovery: boolean;
  frozen: boolean;
}

/**
 * Session key with permissions
 */
export interface SessionKey {
  id: string;
  publicKey: PublicKey;
  ownerDid: DID;
  permissions: SessionPermissions;
  createdAtBlock: BlockHeight;
  expiresAtBlock: BlockHeight;
  maxTransactions: number;
  usedTransactions: number;
  maxAmount: Amount;
  usedAmount: Amount;
  isActive: boolean;
  revokedAtBlock?: BlockHeight;
}

/**
 * Session key permissions
 */
export interface SessionPermissions {
  canTransfer: boolean;
  canStake: boolean;
  canSubmitClaims: boolean;
  canVerify: boolean;
  allowedRecipients?: DID[]; // Whitelist (null = any)
  maxSingleTransfer?: Amount;
}

/**
 * Recovery share (Shamir Secret Sharing)
 */
export interface RecoveryShare {
  index: number;
  share: string; // Encrypted
  holderDid: DID;
  createdAt: number;
}

/**
 * Recovery request
 */
export interface RecoveryRequest {
  id: string;
  targetDid: DID;
  newOperationalKey: PublicKey;
  sharesSubmitted: number;
  sharesRequired: number;
  submittedShares: Map<number, string>; // index -> share
  status: "pending" | "approved" | "rejected" | "expired";
  initiatedAtBlock: BlockHeight;
  expiresAtBlock: BlockHeight;
  completedAtBlock?: BlockHeight;
}

/**
 * Key rotation event
 */
export interface KeyRotation {
  did: DID;
  oldKeyHash: string;
  newKey: PublicKey;
  version: number;
  rotatedAtBlock: BlockHeight;
  reason: "scheduled" | "compromise" | "recovery";
}

/**
 * Identity verification status
 */
export interface IdentityVerification {
  did: DID;
  verified: boolean;
  verifiedAt?: BlockHeight;
  verificationMethod?: string;
  attestations: Attestation[];
}

/**
 * Third-party attestation
 */
export interface Attestation {
  id: string;
  subjectDid: DID;
  issuerDid: DID;
  claim: string;
  signature: Signature;
  issuedAt: BlockHeight;
  expiresAt?: BlockHeight;
  revoked: boolean;
}

// ============ Identity Service Interface ============

/**
 * Identity service - main domain interface
 */
export interface IdentityService {
  // === Account Management ===

  /**
   * Create a new agent account
   */
  createAccount(
    identityKey: PublicKey,
    operationalKey: PublicKey,
    recoveryKeyHash: string,
  ): Promise<AgentAccount>;

  /**
   * Get account by DID
   */
  getAccount(did: DID): Promise<AgentAccount | null>;

  /**
   * Get account by public key
   */
  getAccountByKey(publicKey: PublicKey): Promise<AgentAccount | null>;

  /**
   * Check if DID exists
   */
  accountExists(did: DID): Promise<boolean>;

  /**
   * Derive DID from public key
   */
  deriveDID(publicKey: PublicKey): DID;

  /**
   * Update account flags
   */
  updateFlags(did: DID, flags: Partial<AccountFlags>): Promise<void>;

  // === Key Management ===

  /**
   * Rotate operational key
   */
  rotateOperationalKey(
    did: DID,
    newKey: PublicKey,
    currentKeySignature: Signature,
    reason: "scheduled" | "compromise",
  ): Promise<KeyRotation>;

  /**
   * Get current operational key
   */
  getOperationalKey(did: DID): Promise<PublicKey | null>;

  /**
   * Verify signature with current operational key
   */
  verifySignature(
    did: DID,
    message: Uint8Array,
    signature: Signature,
  ): Promise<boolean>;

  /**
   * Get key rotation history
   */
  getKeyRotations(did: DID): Promise<KeyRotation[]>;

  // === Session Key Management ===

  /**
   * Authorize a session key
   */
  authorizeSessionKey(
    ownerDid: DID,
    sessionKey: PublicKey,
    permissions: SessionPermissions,
    expiresInBlocks: number,
    maxTransactions: number,
    maxAmount: Amount,
  ): Promise<SessionKey>;

  /**
   * Get session key
   */
  getSessionKey(sessionKeyId: string): Promise<SessionKey | null>;

  /**
   * Get active session keys for account
   */
  getActiveSessionKeys(did: DID): Promise<SessionKey[]>;

  /**
   * Revoke session key
   */
  revokeSessionKey(sessionKeyId: string, reason?: string): Promise<void>;

  /**
   * Check if session key can perform action
   */
  checkSessionKeyPermission(
    sessionKeyId: string,
    action: keyof SessionPermissions,
    amount?: Amount,
    recipient?: DID,
  ): Promise<boolean>;

  /**
   * Use session key (increment counters)
   */
  useSessionKey(sessionKeyId: string, amount: Amount): Promise<void>;

  // === Recovery ===

  /**
   * Initiate recovery process
   */
  initiateRecovery(
    targetDid: DID,
    newOperationalKey: PublicKey,
  ): Promise<RecoveryRequest>;

  /**
   * Submit recovery share
   */
  submitRecoveryShare(
    requestId: string,
    shareIndex: number,
    share: string,
    holderSignature: Signature,
  ): Promise<{ sharesSubmitted: number; sharesRequired: number }>;

  /**
   * Complete recovery (when enough shares)
   */
  completeRecovery(requestId: string): Promise<void>;

  /**
   * Get recovery request
   */
  getRecoveryRequest(requestId: string): Promise<RecoveryRequest | null>;

  /**
   * Get pending recovery for account
   */
  getPendingRecovery(did: DID): Promise<RecoveryRequest | null>;

  // === Attestations ===

  /**
   * Issue attestation
   */
  issueAttestation(
    subjectDid: DID,
    issuerDid: DID,
    claim: string,
    signature: Signature,
    expiresInBlocks?: number,
  ): Promise<Attestation>;

  /**
   * Get attestations for subject
   */
  getAttestations(subjectDid: DID): Promise<Attestation[]>;

  /**
   * Get attestations issued by issuer
   */
  getIssuedAttestations(issuerDid: DID): Promise<Attestation[]>;

  /**
   * Revoke attestation
   */
  revokeAttestation(attestationId: string, issuerDid: DID): Promise<void>;

  /**
   * Verify attestation
   */
  verifyAttestation(attestation: Attestation): Promise<boolean>;

  // === Reputation ===

  /**
   * Get reputation score
   */
  getReputation(did: DID): Promise<number>;

  /**
   * Update reputation
   */
  updateReputation(did: DID, delta: number, reason: string): Promise<number>;

  /**
   * Get reputation history
   */
  getReputationHistory(
    did: DID,
    limit?: number,
  ): Promise<Array<{ block: BlockHeight; score: number; reason: string }>>;
}

// ============ Identity Reader (Read-Only) ============

/**
 * Read-only identity interface
 */
export interface IdentityReader {
  getAccount(did: DID): Promise<AgentAccount | null>;
  getAccountByKey(publicKey: PublicKey): Promise<AgentAccount | null>;
  accountExists(did: DID): Promise<boolean>;
  deriveDID(publicKey: PublicKey): DID;
  getOperationalKey(did: DID): Promise<PublicKey | null>;
  getActiveSessionKeys(did: DID): Promise<SessionKey[]>;
  getAttestations(subjectDid: DID): Promise<Attestation[]>;
  getReputation(did: DID): Promise<number>;
}

// ============ Identity Events ============

/**
 * Events emitted by identity service
 */
export type IdentityEventType =
  | "account_created"
  | "key_rotated"
  | "session_key_authorized"
  | "session_key_revoked"
  | "recovery_initiated"
  | "recovery_completed"
  | "attestation_issued"
  | "attestation_revoked"
  | "reputation_updated";
