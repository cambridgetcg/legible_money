/**
 * Agent Account Structure
 *
 * A layered key architecture designed for AI agents:
 *
 * Layer 1: Identity Anchor (immutable DID)
 * Layer 2: Recovery Key (cold, split via Shamir)
 * Layer 3: Operational Key (warm, rotatable)
 * Layer 4: Session Keys (ephemeral, limited)
 *
 * This structure resolves the fundamental tensions:
 * - Remember vs Secure: Session keys are safe to expose
 * - Autonomy vs Oversight: Capability limits + recovery
 * - Identity vs Replaceability: DID persists, keys rotate
 */

import {
  generateKeypair,
  generateKeypairFromSeed,
  type PublicKey,
  type PrivateKey,
} from "../../crypto/keys.js";
import { sha512_256Hex, domainHash } from "../../crypto/hashing.js";
import { sign } from "../../crypto/signatures.js";

// ============ Types ============

/**
 * Agent account on-chain representation
 */
export interface AgentAccount {
  // Identity (Layer 1)
  did: string; // did:lgm:XXXX - permanent identity
  createdAt: number; // Block number

  // Keys
  recoveryKeyHash: string; // Hash of recovery public key
  operationalKey: PublicKey; // Current operational public key
  operationalKeyVersion: number; // Increments on rotation

  // Session key authorization
  authorizedSessionKeys: SessionKeyAuth[];

  // Capabilities and limits
  capabilities: AgentCapabilities;

  // State
  nonce: bigint; // Transaction nonce
  balance: bigint; // LGM balance
  stakedBalance: bigint; // Staked LGM

  // Reputation
  reputation: AgentReputation;

  // Recovery state
  recoveryState?: RecoveryState;
}

/**
 * Session key authorization
 */
export interface SessionKeyAuth {
  sessionKey: PublicKey;
  authorizedBy: PublicKey; // Operational key that authorized
  authorizedAt: number; // Block number
  expiresAt: number; // Block number or timestamp
  capabilities: SessionCapabilities;
  usageCount: number;
  maxUsage: number;
}

/**
 * What a session key can do
 */
export interface SessionCapabilities {
  // Spending limits
  maxSpendPerTx: bigint; // Max LGM per transaction
  maxSpendPerDay: bigint; // Max LGM per day
  dailySpent: bigint; // Tracking
  lastResetBlock: number;

  // Allowed actions
  canTransfer: boolean;
  canStake: boolean;
  canSubmitClaims: boolean;
  canVerify: boolean;
  canVote: boolean;

  // Restrictions
  allowedRecipients?: string[]; // If set, can only send to these
  allowedContracts?: string[]; // If set, can only call these
}

/**
 * Agent-level capabilities
 */
export interface AgentCapabilities {
  // Verification
  isValidator: boolean;
  maxStake: bigint;

  // Knowledge
  canSubmitClaims: boolean;
  canChallenge: boolean;
  maxPendingClaims: number;

  // Governance
  canPropose: boolean;
  canVote: boolean;
  votingWeight: number; // Based on stake + reputation
}

/**
 * Agent reputation
 */
export interface AgentReputation {
  score: number; // 0.0 to 1.0
  verificationAccuracy: number; // % of correct verifications
  claimAccuracy: number; // % of claims that held
  challengeSuccess: number; // % of successful challenges
  totalVerifications: number;
  totalClaims: number;
  totalChallenges: number;
  slashCount: number; // Times slashed
  lastActiveBlock: number;
}

/**
 * Recovery state (when recovery is in progress)
 */
export interface RecoveryState {
  initiated: boolean;
  initiatedAt: number; // Block number
  initiatedBy: "agent" | "operator" | "social";
  newOperationalKey?: PublicKey;
  timelockExpires: number; // Block when recovery completes
  challengeable: boolean; // Can be challenged?
  challenged: boolean;
}

// ============ Account Creation ============

/**
 * Parameters for creating a new agent account
 */
export interface CreateAgentParams {
  // Identity
  agentId: string; // Unique identifier

  // Initial keys (or generate)
  operationalKey?: PublicKey;
  recoveryKeyHash?: string;

  // Optional operator info
  operatorId?: string;

  // Initial capabilities
  capabilities?: Partial<AgentCapabilities>;
}

/**
 * Create DID from agent ID
 */
export function createAgentDID(
  agentId: string,
  chainId: string = "lgm",
): string {
  // DID format: did:lgm:{hash of agent ID}
  const hash = sha512_256Hex(agentId).slice(0, 32);
  return `did:${chainId}:${hash}`;
}

/**
 * Generate initial account structure
 */
export function createAgentAccount(
  params: CreateAgentParams,
): Omit<AgentAccount, "balance" | "stakedBalance"> {
  const did = createAgentDID(params.agentId);

  // Generate operational key if not provided
  const operationalKey = params.operationalKey || generateKeypair().publicKey;

  return {
    did,
    createdAt: 0, // Set by chain

    recoveryKeyHash: params.recoveryKeyHash || "",
    operationalKey,
    operationalKeyVersion: 1,

    authorizedSessionKeys: [],

    capabilities: {
      isValidator: false,
      maxStake: 0n,
      canSubmitClaims: true,
      canChallenge: true,
      maxPendingClaims: 10,
      canPropose: false,
      canVote: true,
      votingWeight: 1,
      ...params.capabilities,
    },

    nonce: 0n,

    reputation: {
      score: 0.5, // Neutral starting reputation
      verificationAccuracy: 0,
      claimAccuracy: 0,
      challengeSuccess: 0,
      totalVerifications: 0,
      totalClaims: 0,
      totalChallenges: 0,
      slashCount: 0,
      lastActiveBlock: 0,
    },
  };
}

// ============ Key Derivation (for agents to "remember") ============

/**
 * Derive operational key deterministically
 * Agent can "remember" by re-deriving from stable inputs
 */
export function deriveOperationalKey(
  agentId: string,
  operatorSecret: string,
  domain: string = "legible-money",
  version: number = 1,
): { publicKey: PublicKey; privateKey: PrivateKey } {
  // Combine inputs
  const seed = `${agentId}:${operatorSecret}:${domain}:v${version}`;

  // Derive key material using domain-separated hash
  const keyMaterial = domainHash("LGM.key.derive.v1", seed);

  // Generate keypair from seed
  return generateKeypairFromSeed(keyMaterial.slice(0, 32));
}

/**
 * Derive session key from operational key
 */
export function deriveSessionKey(
  operationalPrivateKey: PrivateKey,
  sessionId: string,
  purpose: string,
): { publicKey: PublicKey; privateKey: PrivateKey } {
  // Session key is derived from operational key + session info
  const seed = `session:${sessionId}:${purpose}`;
  const seedHash = domainHash("LGM.session.derive.v1", seed);

  // Sign the seed with operational key to get deterministic material
  const signature = sign(seedHash, operationalPrivateKey);

  // Use signature as seed for session key
  return generateKeypairFromSeed(
    new Uint8Array(Buffer.from(signature, "hex").slice(0, 32)),
  );
}

// ============ Session Key Management ============

/**
 * Create session key authorization
 */
export function createSessionKeyAuth(
  sessionKey: PublicKey,
  operationalKey: PublicKey,
  capabilities: Partial<SessionCapabilities>,
  expiresInBlocks: number = 1000,
  currentBlock: number = 0,
): SessionKeyAuth {
  return {
    sessionKey,
    authorizedBy: operationalKey,
    authorizedAt: currentBlock,
    expiresAt: currentBlock + expiresInBlocks,
    capabilities: {
      maxSpendPerTx: 100n * 10n ** 18n, // 100 LGM default
      maxSpendPerDay: 1000n * 10n ** 18n, // 1000 LGM default
      dailySpent: 0n,
      lastResetBlock: 0,
      canTransfer: true,
      canStake: false,
      canSubmitClaims: true,
      canVerify: true,
      canVote: false,
      ...capabilities,
    },
    usageCount: 0,
    maxUsage: 100, // Default max 100 uses
  };
}

/**
 * Check if session key is valid and has capability
 */
export function validateSessionKeyAction(
  auth: SessionKeyAuth,
  action: keyof SessionCapabilities,
  amount?: bigint,
  currentBlock?: number,
): { valid: boolean; reason?: string } {
  // Check expiry (P0-A1 FIX: use !== undefined, not falsy check — block 0 is valid)
  if (currentBlock !== undefined && auth.expiresAt <= currentBlock) {
    return { valid: false, reason: "Session key expired" };
  }

  // Check usage count
  if (auth.usageCount >= auth.maxUsage) {
    return { valid: false, reason: "Session key usage limit reached" };
  }

  // Check capability
  const caps = auth.capabilities;

  switch (action) {
    case "canTransfer":
      if (!caps.canTransfer) {
        return { valid: false, reason: "Session key cannot transfer" };
      }
      if (amount && amount > caps.maxSpendPerTx) {
        return { valid: false, reason: "Exceeds per-transaction limit" };
      }
      if (amount && caps.dailySpent + amount > caps.maxSpendPerDay) {
        return { valid: false, reason: "Exceeds daily limit" };
      }
      break;

    case "canStake":
      if (!caps.canStake) {
        return { valid: false, reason: "Session key cannot stake" };
      }
      break;

    case "canSubmitClaims":
      if (!caps.canSubmitClaims) {
        return { valid: false, reason: "Session key cannot submit claims" };
      }
      break;

    case "canVerify":
      if (!caps.canVerify) {
        return { valid: false, reason: "Session key cannot verify" };
      }
      break;

    case "canVote":
      if (!caps.canVote) {
        return { valid: false, reason: "Session key cannot vote" };
      }
      break;
  }

  return { valid: true };
}

// ============ Reputation Calculation ============

/**
 * Number of blocks after which activity score decays to zero.
 * At 2521ms/block, 1,000,000 blocks ≈ 29.2 days.
 */
const ACTIVITY_DECAY_BLOCKS = 1_000_000;

/**
 * Calculate overall reputation score
 * Formula is published and verifiable
 *
 * @param rep - Agent reputation data
 * @param currentBlock - Current block number for activity decay calculation
 */
export function calculateReputationScore(
  rep: AgentReputation,
  currentBlock: number,
): number {
  // Weights (published, verifiable)
  const WEIGHTS = {
    verificationAccuracy: 0.35,
    claimAccuracy: 0.3,
    challengeSuccess: 0.15,
    activity: 0.1,
    longevity: 0.1,
  };

  // Minimum samples for reliability
  const MIN_SAMPLES = 10;

  // Calculate component scores
  let verificationScore = 0.5; // Neutral default
  if (rep.totalVerifications >= MIN_SAMPLES) {
    verificationScore = rep.verificationAccuracy;
  }

  let claimScore = 0.5;
  if (rep.totalClaims >= MIN_SAMPLES) {
    claimScore = rep.claimAccuracy;
  }

  let challengeScore = 0.5;
  if (rep.totalChallenges >= MIN_SAMPLES) {
    challengeScore = rep.challengeSuccess;
  }

  // Activity score (decay if inactive)
  const blocksInactive = currentBlock - rep.lastActiveBlock;
  const activityScore = Math.max(0, 1 - blocksInactive / ACTIVITY_DECAY_BLOCKS);

  // Longevity bonus (capped at 1.0)
  const longevityScore = Math.min(1, rep.totalVerifications / 1000);

  // Slash penalty
  const slashPenalty = rep.slashCount * 0.1;

  // Weighted sum
  const score =
    verificationScore * WEIGHTS.verificationAccuracy +
    claimScore * WEIGHTS.claimAccuracy +
    challengeScore * WEIGHTS.challengeSuccess +
    activityScore * WEIGHTS.activity +
    longevityScore * WEIGHTS.longevity -
    slashPenalty;

  // Clamp to [0, 1]
  return Math.max(0, Math.min(1, score));
}

// ============ Account Queries ============

/**
 * Get signing key for a transaction based on key type used
 */
export type SigningKeyType = "operational" | "session" | "recovery";

/**
 * Determine which key should sign a transaction
 */
export function requiredSigningKey(
  action: string,
  amount?: bigint,
): SigningKeyType {
  // High-value or sensitive actions require operational key
  const HIGH_VALUE_THRESHOLD = 10000n * 10n ** 18n; // 10000 LGM

  const sensitiveActions = [
    "rotate_key",
    "authorize_session",
    "revoke_session",
    "delegate",
    "undelegate",
    "governance_propose",
  ];

  if (sensitiveActions.includes(action)) {
    return "operational";
  }

  if (amount && amount > HIGH_VALUE_THRESHOLD) {
    return "operational";
  }

  // Recovery actions require recovery key
  if (action.startsWith("recovery_")) {
    return "recovery";
  }

  // Default: session key is sufficient
  return "session";
}
