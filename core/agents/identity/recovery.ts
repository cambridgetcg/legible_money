/**
 * Recovery System
 *
 * Multi-path recovery for agent accounts:
 * 1. Self-recovery (agent has recovery key shard)
 * 2. Operator recovery (operator has recovery key shard)
 * 3. Social recovery (trusted guardians hold shards)
 * 4. Time-locked recovery (wait period to prevent theft)
 *
 * Uses Shamir Secret Sharing for key splitting.
 */

import { type PublicKey, bytesToHex } from "../../crypto/keys.js";
import { sha512_256Hex, domainHash } from "../../crypto/hashing.js";
import { verify } from "../../crypto/signatures.js";
import { randomBytes } from "@noble/hashes/utils";
import { publicKeyToDid } from "./did.js";

// ============ Types ============

/**
 * Recovery configuration for an account
 */
export interface RecoveryConfig {
  // Threshold settings
  threshold: number; // Minimum shards needed (e.g., 2)
  totalShards: number; // Total shards created (e.g., 3)

  // Shard holders
  shardHolders: ShardHolder[];

  // Time lock
  recoveryDelayBlocks: number; // Blocks to wait before recovery completes

  // Challenge period
  challengePeriodBlocks: number; // Blocks during which recovery can be challenged

  // Social recovery
  socialRecovery?: SocialRecoveryConfig;
}

/**
 * Who holds a shard
 */
export interface ShardHolder {
  type: "agent" | "operator" | "protocol" | "guardian";
  identifier: string; // DID or address
  shardIndex: number;
  publicKeyHash: string; // To verify the holder
  addedAt: number;
  canInitiateRecovery: boolean;
}

/**
 * Social recovery configuration
 */
export interface SocialRecoveryConfig {
  guardians: Guardian[];
  requiredGuardians: number; // How many must approve
  guardianDelay: number; // Extra delay for guardian recovery
}

/**
 * A recovery guardian
 */
export interface Guardian {
  did: string;
  publicKey: PublicKey;
  addedAt: number;
  lastActive: number;
  trustScore: number; // Based on guardian history
}

/**
 * A recovery shard (Shamir share)
 */
export interface RecoveryShard {
  index: number;
  value: Uint8Array;
  holderType: ShardHolder["type"];
  encryptedFor?: PublicKey; // If encrypted for specific holder
}

/**
 * Recovery request
 */
export interface RecoveryRequest {
  id: string;
  agentDid: string;
  initiatedBy: string; // DID of initiator
  initiatedAt: number; // Block number

  // New key
  newOperationalKey: PublicKey;

  // Progress
  shardsProvided: number[]; // Indices of provided shards
  shardsRequired: number;

  // Timing
  delayExpires: number; // Block when delay ends
  challengeExpires: number; // Block when challenge period ends

  // Status
  status:
    | "pending"
    | "delayed"
    | "challengeable"
    | "executable"
    | "completed"
    | "cancelled";

  // If challenged
  challenge?: RecoveryChallenge;
}

/**
 * Challenge to a recovery request
 */
export interface RecoveryChallenge {
  challenger: string; // DID
  reason: string;
  evidence?: string;
  signature: string;
  submittedAt: number;
  resolved: boolean;
  resolution?: "upheld" | "rejected";
}

// ============ Default Configuration ============

export const DEFAULT_RECOVERY_CONFIG: Omit<RecoveryConfig, "shardHolders"> = {
  threshold: 2,
  totalShards: 3,
  recoveryDelayBlocks: 1000, // ~40 minutes at 2.5s blocks
  challengePeriodBlocks: 500, // ~20 minutes
};

export const HIGH_SECURITY_RECOVERY_CONFIG: Omit<
  RecoveryConfig,
  "shardHolders"
> = {
  threshold: 3,
  totalShards: 5,
  recoveryDelayBlocks: 10000, // ~7 hours
  challengePeriodBlocks: 5000, // ~3.5 hours
};

// ============ Shamir Secret Sharing ============

/**
 * Prime for finite field operations
 * Using a 256-bit prime for security
 */
const PRIME = BigInt(
  "0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141",
);

/**
 * Split a secret into shares using Shamir's Secret Sharing
 */
export function splitSecret(
  secret: Uint8Array,
  threshold: number,
  totalShards: number,
): RecoveryShard[] {
  if (threshold > totalShards) {
    throw new Error("Threshold cannot exceed total shards");
  }
  if (threshold < 2) {
    throw new Error("Threshold must be at least 2");
  }

  // Convert secret to bigint
  const secretNum = bytesToBigInt(secret);

  // Generate random coefficients for polynomial
  // f(x) = secret + a1*x + a2*x^2 + ... + a(t-1)*x^(t-1)
  const coefficients: bigint[] = [secretNum];
  for (let i = 1; i < threshold; i++) {
    coefficients.push(bytesToBigInt(randomBytes(32)) % PRIME);
  }

  // Evaluate polynomial at points 1, 2, 3, ..., totalShards
  const shards: RecoveryShard[] = [];
  for (let x = 1; x <= totalShards; x++) {
    const y = evaluatePolynomial(coefficients, BigInt(x), PRIME);
    shards.push({
      index: x,
      value: bigIntToBytes(y, 32),
      holderType: "agent", // Set by caller
    });
  }

  return shards;
}

/**
 * Reconstruct secret from shares using Lagrange interpolation
 */
export function reconstructSecret(
  shards: RecoveryShard[],
  threshold: number,
): Uint8Array {
  if (shards.length < threshold) {
    throw new Error(`Need at least ${threshold} shards, got ${shards.length}`);
  }

  // Take only threshold shards
  const usedShards = shards.slice(0, threshold);

  // Lagrange interpolation at x=0 to recover secret
  let secret = 0n;

  for (let i = 0; i < usedShards.length; i++) {
    const xi = BigInt(usedShards[i].index);
    const yi = bytesToBigInt(usedShards[i].value);

    // Calculate Lagrange basis polynomial
    let numerator = 1n;
    let denominator = 1n;

    for (let j = 0; j < usedShards.length; j++) {
      if (i !== j) {
        const xj = BigInt(usedShards[j].index);
        numerator = (numerator * (0n - xj)) % PRIME;
        denominator = (denominator * (xi - xj)) % PRIME;
      }
    }

    // Modular division: multiply by modular inverse
    const lagrangeCoeff = (numerator * modInverse(denominator, PRIME)) % PRIME;

    // Add contribution
    secret = (secret + yi * lagrangeCoeff) % PRIME;
  }

  // Handle negative modulo
  secret = ((secret % PRIME) + PRIME) % PRIME;

  return bigIntToBytes(secret, 32);
}

// ============ Recovery Operations ============

/**
 * Create recovery configuration for an account
 */
export function createRecoveryConfig(
  recoveryPublicKey: PublicKey,
  shardHolders: Array<{
    type: ShardHolder["type"];
    identifier: string;
    canInitiate: boolean;
  }>,
  options: Partial<RecoveryConfig> = {},
  currentBlock: number = 0,
): { config: RecoveryConfig; shards: Map<number, RecoveryShard> } {
  const config: RecoveryConfig = {
    ...DEFAULT_RECOVERY_CONFIG,
    ...options,
    shardHolders: [],
  };

  // Validate
  if (shardHolders.length < config.threshold) {
    throw new Error("Not enough shard holders for threshold");
  }
  config.totalShards = shardHolders.length;

  // Generate recovery key and split it
  const recoveryKeyHash = sha512_256Hex(recoveryPublicKey);

  // For actual implementation, we'd split the private key
  // Here we'll split a derived secret
  const secret = domainHash("LGM.recovery.secret.v1", recoveryKeyHash);
  const shards = splitSecret(secret, config.threshold, config.totalShards);

  // Assign shards to holders
  const shardMap = new Map<number, RecoveryShard>();
  config.shardHolders = shardHolders.map((holder, i) => {
    const shard = shards[i];
    shard.holderType = holder.type;
    shardMap.set(shard.index, shard);

    return {
      type: holder.type,
      identifier: holder.identifier,
      shardIndex: shard.index,
      publicKeyHash: sha512_256Hex(holder.identifier),
      addedAt: currentBlock, // P1-A5 FIX: Use block number, not wall clock
      canInitiateRecovery: holder.canInitiate,
    };
  });

  return { config, shards: shardMap };
}

/**
 * Initiate a recovery request
 */
export function createRecoveryRequest(
  agentDid: string,
  initiatorDid: string,
  newOperationalKey: PublicKey,
  config: RecoveryConfig,
  currentBlock: number,
): RecoveryRequest {
  return {
    id: sha512_256Hex(
      `recovery:${agentDid}:${currentBlock}:${bytesToHex(randomBytes(8))}`,
    ),
    agentDid,
    initiatedBy: initiatorDid,
    initiatedAt: currentBlock,
    newOperationalKey,
    shardsProvided: [],
    shardsRequired: config.threshold,
    delayExpires: currentBlock + config.recoveryDelayBlocks,
    challengeExpires:
      currentBlock + config.recoveryDelayBlocks + config.challengePeriodBlocks,
    status: "pending",
  };
}

/**
 * Add a shard to a recovery request
 */
export function addShardToRecovery(
  request: RecoveryRequest,
  shard: RecoveryShard,
  holderSignature: string,
  holderPublicKey: PublicKey,
  config?: RecoveryConfig,
): RecoveryRequest {
  // Verify shard hasn't been added
  if (request.shardsProvided.includes(shard.index)) {
    throw new Error("Shard already provided");
  }

  // Verify holder identity against config (if provided)
  if (config) {
    const holderEntry = config.shardHolders.find(
      (h) => h.shardIndex === shard.index,
    );
    if (!holderEntry) {
      throw new Error("Shard index not assigned to any holder");
    }
    // Verify signer is the authorized holder by DID derivation
    const signerDid = publicKeyToDid(holderPublicKey);
    if (
      signerDid !== holderEntry.identifier &&
      sha512_256Hex(signerDid) !== holderEntry.publicKeyHash
    ) {
      throw new Error(
        "Shard holder identity mismatch - signer not authorized for this shard",
      );
    }
  }

  // Verify holder signature
  const message = `recovery:${request.id}:shard:${shard.index}`;
  if (
    !verify(new TextEncoder().encode(message), holderSignature, holderPublicKey)
  ) {
    throw new Error("Invalid shard holder signature");
  }

  // Add shard
  const updated = { ...request };
  updated.shardsProvided = [...updated.shardsProvided, shard.index];

  // Check if threshold met
  if (updated.shardsProvided.length >= updated.shardsRequired) {
    updated.status = "delayed";
  }

  return updated;
}

/**
 * Update recovery status based on current block
 */
export function updateRecoveryStatus(
  request: RecoveryRequest,
  currentBlock: number,
): RecoveryRequest {
  const updated = { ...request };

  if (updated.status === "delayed" && currentBlock >= updated.delayExpires) {
    updated.status = "challengeable";
  }

  if (
    updated.status === "challengeable" &&
    currentBlock >= updated.challengeExpires
  ) {
    updated.status = "executable";
  }

  return updated;
}

/**
 * Challenge a recovery request
 */
export function challengeRecovery(
  request: RecoveryRequest,
  challengerDid: string,
  reason: string,
  signature: string,
  challengerPublicKey: PublicKey,
  currentBlock: number,
): RecoveryRequest {
  if (request.status !== "challengeable") {
    throw new Error("Recovery not in challengeable state");
  }

  if (currentBlock >= request.challengeExpires) {
    throw new Error("Challenge period expired");
  }

  // P0-A2 FIX: Verify the challenger's signature on the challenge message.
  // Without verification, anyone could submit fake challenges to stall recovery.
  const challengeMessage = `challenge:${request.id}:${challengerDid}:${reason}`;
  if (
    !verify(
      new TextEncoder().encode(challengeMessage),
      signature,
      challengerPublicKey,
    )
  ) {
    throw new Error("Invalid challenger signature");
  }

  return {
    ...request,
    challenge: {
      challenger: challengerDid,
      reason,
      signature,
      submittedAt: currentBlock,
      resolved: false,
    },
  };
}

// ============ Utility Functions ============

function evaluatePolynomial(
  coefficients: bigint[],
  x: bigint,
  prime: bigint,
): bigint {
  let result = 0n;
  let xPow = 1n;

  for (const coef of coefficients) {
    result = (result + coef * xPow) % prime;
    xPow = (xPow * x) % prime;
  }

  return result;
}

function modInverse(a: bigint, m: bigint): bigint {
  // Extended Euclidean algorithm
  let [old_r, r] = [a, m];
  let [old_s, s] = [1n, 0n];

  while (r !== 0n) {
    const quotient = old_r / r;
    [old_r, r] = [r, old_r - quotient * r];
    [old_s, s] = [s, old_s - quotient * s];
  }

  return ((old_s % m) + m) % m;
}

function bytesToBigInt(bytes: Uint8Array): bigint {
  let result = 0n;
  for (const byte of bytes) {
    result = (result << 8n) + BigInt(byte);
  }
  return result;
}

function bigIntToBytes(num: bigint, length: number): Uint8Array {
  const bytes = new Uint8Array(length);
  let temp = num;
  for (let i = length - 1; i >= 0; i--) {
    bytes[i] = Number(temp & 0xffn);
    temp = temp >> 8n;
  }
  return bytes;
}

// ============ Guardian Management ============

/**
 * Add a guardian to social recovery
 */
export function addGuardian(
  config: RecoveryConfig,
  guardian: Guardian,
): RecoveryConfig {
  // P0-A3 FIX: Return new object instead of mutating input
  const socialRecovery = config.socialRecovery
    ? {
        ...config.socialRecovery,
        guardians: [...config.socialRecovery.guardians],
      }
    : {
        guardians: [],
        requiredGuardians: 2,
        guardianDelay: 5000, // Extra delay blocks
      };

  // P2-A6 FIX: Prevent duplicate guardian DIDs
  if (socialRecovery.guardians.some((g) => g.did === guardian.did)) {
    throw new Error("Guardian already exists");
  }

  socialRecovery.guardians.push(guardian);
  return { ...config, socialRecovery };
}

/**
 * Remove a guardian
 */
export function removeGuardian(
  config: RecoveryConfig,
  guardianDid: string,
): RecoveryConfig {
  if (!config.socialRecovery) return config;

  // P0-A3 FIX: Return new object instead of mutating input
  return {
    ...config,
    socialRecovery: {
      ...config.socialRecovery,
      guardians: config.socialRecovery.guardians.filter(
        (g) => g.did !== guardianDid,
      ),
    },
  };
}

/**
 * Block-based constants for guardian trust scoring.
 * At 2521ms/block:
 *   1 year  ≈ 12,537,089 blocks (use 12_500_000)
 *   30 days ≈  1,029,750 blocks (use 1_000_000)
 */
const GUARDIAN_AGE_MAX_BLOCKS = 12_500_000;
const GUARDIAN_ACTIVITY_DECAY_BLOCKS = 1_000_000;

/**
 * Calculate guardian trust score based on history
 *
 * @param guardian - Guardian data (addedAt and lastActive are block numbers)
 * @param currentBlock - Current block number
 */
export function calculateGuardianTrustScore(
  guardian: Guardian,
  currentBlock: number,
): number {
  // Factors:
  // - How long they've been a guardian (blocks)
  // - How recently they were active (blocks)
  // - History of correct recoveries

  const ageScore = Math.min(
    1,
    (currentBlock - guardian.addedAt) / GUARDIAN_AGE_MAX_BLOCKS,
  );
  const activityScore = Math.max(
    0,
    1 - (currentBlock - guardian.lastActive) / GUARDIAN_ACTIVITY_DECAY_BLOCKS,
  );

  return ageScore * 0.4 + activityScore * 0.3 + guardian.trustScore * 0.3;
}
