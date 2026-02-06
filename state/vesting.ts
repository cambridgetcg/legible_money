/**
 * Vesting State Tree
 *
 * Sparse Merkle Tree storage for reward vesting schedules.
 * Provides O(log n) proofs for vesting state queries.
 *
 * Indexes:
 * - By vesting ID (primary)
 * - By claim ID (for dispute integration)
 * - By recipient (for user queries)
 * - By status (for batch processing)
 *
 * PP HIERARCHY:
 *   TRUTH: SMT provides cryptographic proofs of vesting state
 *   UNDERSTANDING: Clear indexing for different query patterns
 *   BEAUTY: Follows existing state tree patterns in codebase
 *   JUSTICE: Deterministic state transitions for all operations
 */

import { InMemorySMT, type MerkleRoot } from "../core/crypto/merkle.js";
import { sha512_256Hex } from "../core/crypto/hashing.js";
import {
  type RewardVesting,
  type VestingCategory,
  type VestingStatus,
  type CreateVestingPayload,
  VestingError,
} from "../core/rewards/types.js";
import {
  createVesting,
  updateRelease,
  claimRewards,
  handleChallengeStart,
  handleChallengeResolution,
  handleReplication,
  getVestingSummary,
  canClaim,
} from "../core/rewards/vesting_engine.js";

// ============ Key Generation ============

/**
 * Generate SMT key from vesting ID
 */
function vestingIdToKey(vestingId: string): Uint8Array {
  const hash = sha512_256Hex(`vesting:id:${vestingId}`);
  return new Uint8Array(Buffer.from(hash, "hex").slice(0, 32));
}

// ============ Serialization ============

/**
 * Serialize vesting for SMT storage
 */
function serializeVesting(vesting: RewardVesting): Uint8Array {
  const json = JSON.stringify(vesting, (_, value) =>
    typeof value === "bigint" ? value.toString() + "n" : value,
  );
  return new TextEncoder().encode(json);
}

/**
 * Deserialize vesting from SMT storage
 */
function deserializeVesting(data: Uint8Array): RewardVesting {
  const json = new TextDecoder().decode(data);
  return JSON.parse(json, (_, value) => {
    if (typeof value === "string" && value.endsWith("n")) {
      return BigInt(value.slice(0, -1));
    }
    return value;
  });
}

// ============ Vesting State Tree ============

export class VestingStateTree {
  private vestingSmt: InMemorySMT;
  private vestingCache: Map<string, RewardVesting>;

  // Indexes for efficient queries
  private claimToVestingIndex: Map<string, string>; // claimId -> vestingId
  private recipientToVestingsIndex: Map<string, Set<string>>; // recipient -> vestingIds
  private statusIndex: Map<VestingStatus, Set<string>>; // status -> vestingIds
  private categoryIndex: Map<VestingCategory, Set<string>>; // category -> vestingIds

  // Statistics
  private stats: {
    totalVestings: number;
    totalAmount: bigint;
    totalReleased: bigint;
    totalClaimed: bigint;
    activeVestings: number;
    pausedVestings: number;
    completedVestings: number;
    falsifiedVestings: number;
  };

  constructor() {
    this.vestingSmt = new InMemorySMT();
    this.vestingCache = new Map();
    this.claimToVestingIndex = new Map();
    this.recipientToVestingsIndex = new Map();
    this.statusIndex = new Map();
    this.categoryIndex = new Map();

    // Initialize status index
    for (const status of [
      "vesting",
      "paused",
      "completed",
      "falsified",
      "abandoned",
    ] as VestingStatus[]) {
      this.statusIndex.set(status, new Set());
    }

    this.stats = {
      totalVestings: 0,
      totalAmount: 0n,
      totalReleased: 0n,
      totalClaimed: 0n,
      activeVestings: 0,
      pausedVestings: 0,
      completedVestings: 0,
      falsifiedVestings: 0,
    };
  }

  // ============ Core Operations ============

  /**
   * Create a new vesting schedule
   */
  createVesting(
    payload: CreateVestingPayload,
    currentBlock: bigint,
  ): RewardVesting {
    // Check if vesting already exists for this claim
    if (this.claimToVestingIndex.has(payload.claimId)) {
      throw new VestingError(
        `Vesting already exists for claim ${payload.claimId}`,
        "INVALID_AMOUNT",
      );
    }

    // Create vesting
    const vesting = createVesting(payload, currentBlock);

    // Store in SMT
    const key = vestingIdToKey(vesting.id);
    this.vestingSmt.set(key, serializeVesting(vesting));

    // Update cache and indexes
    this.vestingCache.set(vesting.id, vesting);
    this.claimToVestingIndex.set(payload.claimId, vesting.id);

    // Recipient index
    if (!this.recipientToVestingsIndex.has(payload.recipient)) {
      this.recipientToVestingsIndex.set(payload.recipient, new Set());
    }
    this.recipientToVestingsIndex.get(payload.recipient)!.add(vesting.id);

    // Status index
    this.statusIndex.get("vesting")!.add(vesting.id);

    // Category index
    if (!this.categoryIndex.has(vesting.category)) {
      this.categoryIndex.set(vesting.category, new Set());
    }
    this.categoryIndex.get(vesting.category)!.add(vesting.id);

    // Update stats
    this.stats.totalVestings++;
    this.stats.totalAmount += payload.amount;
    this.stats.activeVestings++;

    return vesting;
  }

  /**
   * Get vesting by ID
   */
  getVesting(vestingId: string): RewardVesting | null {
    // Check cache first
    if (this.vestingCache.has(vestingId)) {
      return this.vestingCache.get(vestingId)!;
    }

    // Load from SMT
    const key = vestingIdToKey(vestingId);
    const data = this.vestingSmt.get(key);
    if (!data) {
      return null;
    }

    const vesting = deserializeVesting(data);
    this.vestingCache.set(vestingId, vesting);
    return vesting;
  }

  /**
   * Get vesting by claim ID
   */
  getVestingForClaim(claimId: string): RewardVesting | null {
    const vestingId = this.claimToVestingIndex.get(claimId);
    if (!vestingId) {
      return null;
    }
    return this.getVesting(vestingId);
  }

  /**
   * Update vesting
   */
  updateVesting(vesting: RewardVesting): void {
    const oldVesting = this.getVesting(vesting.id);

    // Update SMT
    const key = vestingIdToKey(vesting.id);
    this.vestingSmt.set(key, serializeVesting(vesting));

    // Update cache
    this.vestingCache.set(vesting.id, vesting);

    // Update status index if status changed
    if (oldVesting && oldVesting.status !== vesting.status) {
      this.statusIndex.get(oldVesting.status)?.delete(vesting.id);
      this.statusIndex.get(vesting.status)?.add(vesting.id);

      // Update stats
      this.updateStatusStats(oldVesting.status, vesting.status);
    }
  }

  /**
   * Update status statistics
   */
  private updateStatusStats(
    oldStatus: VestingStatus,
    newStatus: VestingStatus,
  ): void {
    // Decrement old
    switch (oldStatus) {
      case "vesting":
        this.stats.activeVestings--;
        break;
      case "paused":
        this.stats.pausedVestings--;
        break;
      case "completed":
        this.stats.completedVestings--;
        break;
      case "falsified":
        this.stats.falsifiedVestings--;
        break;
    }

    // Increment new
    switch (newStatus) {
      case "vesting":
        this.stats.activeVestings++;
        break;
      case "paused":
        this.stats.pausedVestings++;
        break;
      case "completed":
        this.stats.completedVestings++;
        break;
      case "falsified":
        this.stats.falsifiedVestings++;
        break;
    }
  }

  // ============ Vesting Operations ============

  /**
   * Update release and claim rewards
   */
  claimRewards(
    vestingId: string,
    claimer: string,
    amount: bigint | undefined,
    currentBlock: bigint,
  ): { vesting: RewardVesting; claimedAmount: bigint } {
    const vesting = this.getVesting(vestingId);
    if (!vesting) {
      throw new VestingError("Vesting not found", "VESTING_NOT_FOUND");
    }

    const { updatedVesting, claimedAmount } = claimRewards(
      vesting,
      { vestingId, claimer, amount },
      currentBlock,
    );

    this.updateVesting(updatedVesting);

    // Update stats
    this.stats.totalClaimed += claimedAmount;

    return { vesting: updatedVesting, claimedAmount };
  }

  /**
   * Pause vesting for challenge
   */
  pauseForChallenge(
    vestingId: string,
    challengeId: string,
    currentBlock: bigint,
  ): RewardVesting {
    const vesting = this.getVesting(vestingId);
    if (!vesting) {
      throw new VestingError("Vesting not found", "VESTING_NOT_FOUND");
    }

    const updated = handleChallengeStart(vesting, challengeId, currentBlock);
    this.updateVesting(updated);

    return updated;
  }

  /**
   * Resume vesting after challenge resolution
   */
  resumeAfterChallenge(
    vestingId: string,
    challengerWon: boolean,
    challengerDid: string,
    disputeId: string,
    currentBlock: bigint,
  ): { vesting: RewardVesting; clawback?: any } {
    const vesting = this.getVesting(vestingId);
    if (!vesting) {
      throw new VestingError("Vesting not found", "VESTING_NOT_FOUND");
    }

    const result = handleChallengeResolution(
      vesting,
      challengerWon,
      challengerDid,
      disputeId,
      currentBlock,
    );

    this.updateVesting(result.updatedVesting);

    return {
      vesting: result.updatedVesting,
      clawback: result.clawback,
    };
  }

  /**
   * Apply acceleration for replication
   */
  applyReplication(
    vestingId: string,
    researchId: string,
    currentBlock: bigint,
  ): RewardVesting {
    const vesting = this.getVesting(vestingId);
    if (!vesting) {
      throw new VestingError("Vesting not found", "VESTING_NOT_FOUND");
    }

    const updated = handleReplication(vesting, researchId, currentBlock);
    this.updateVesting(updated);

    return updated;
  }

  // ============ Query Operations ============

  /**
   * Get all vestings for a recipient
   */
  getVestingsForRecipient(recipient: string): RewardVesting[] {
    const vestingIds = this.recipientToVestingsIndex.get(recipient);
    if (!vestingIds) {
      return [];
    }

    return Array.from(vestingIds)
      .map((id) => this.getVesting(id))
      .filter((v): v is RewardVesting => v !== null);
  }

  /**
   * Get vestings by status
   */
  getVestingsByStatus(status: VestingStatus): RewardVesting[] {
    const vestingIds = this.statusIndex.get(status);
    if (!vestingIds) {
      return [];
    }

    return Array.from(vestingIds)
      .map((id) => this.getVesting(id))
      .filter((v): v is RewardVesting => v !== null);
  }

  /**
   * Get vestings by category
   */
  getVestingsByCategory(category: VestingCategory): RewardVesting[] {
    const vestingIds = this.categoryIndex.get(category);
    if (!vestingIds) {
      return [];
    }

    return Array.from(vestingIds)
      .map((id) => this.getVesting(id))
      .filter((v): v is RewardVesting => v !== null);
  }

  /**
   * Get active vestings that can be claimed
   */
  getClaimableVestings(currentBlock: bigint): RewardVesting[] {
    const activeVestings = this.getVestingsByStatus("vesting");

    return activeVestings.filter((v) => {
      const check = canClaim(v, currentBlock);
      return check.canClaim;
    });
  }

  /**
   * Get vesting summary
   */
  getVestingSummary(vestingId: string, currentBlock: bigint) {
    const vesting = this.getVesting(vestingId);
    if (!vesting) {
      throw new VestingError("Vesting not found", "VESTING_NOT_FOUND");
    }

    return getVestingSummary(vesting, currentBlock);
  }

  // ============ Batch Operations ============

  /**
   * Update releases for all active vestings
   */
  batchUpdateReleases(currentBlock: bigint): number {
    const activeVestings = this.getVestingsByStatus("vesting");
    let updatedCount = 0;

    for (const vesting of activeVestings) {
      if (vesting.isPaused) continue;

      const updated = updateRelease(vesting, currentBlock);
      if (
        updated.releasedAmount !== vesting.releasedAmount ||
        updated.status !== vesting.status
      ) {
        this.updateVesting(updated);
        this.stats.totalReleased =
          this.stats.totalReleased +
          (updated.releasedAmount - vesting.releasedAmount);
        updatedCount++;
      }
    }

    return updatedCount;
  }

  // ============ State Management ============

  /**
   * Get Merkle root
   */
  getRoot(): MerkleRoot {
    return this.vestingSmt.getRoot();
  }

  /**
   * Get statistics
   */
  getStats() {
    return { ...this.stats };
  }

  // ============ Persistence Export/Restore ============

  /**
   * Export all vestings for persistence
   */
  exportVestings(): RewardVesting[] {
    return Array.from(this.vestingCache.values());
  }

  /**
   * Restore a vesting from persistence (bypasses creation logic,
   * takes a pre-built RewardVesting)
   */
  restoreVesting(vesting: RewardVesting): void {
    // Store in SMT
    const key = vestingIdToKey(vesting.id);
    this.vestingSmt.set(key, serializeVesting(vesting));

    // Update cache
    this.vestingCache.set(vesting.id, vesting);

    // Claim-to-vesting index
    if (vesting.claimId) {
      this.claimToVestingIndex.set(vesting.claimId, vesting.id);
    }

    // Recipient index
    if (!this.recipientToVestingsIndex.has(vesting.recipient)) {
      this.recipientToVestingsIndex.set(vesting.recipient, new Set());
    }
    this.recipientToVestingsIndex.get(vesting.recipient)!.add(vesting.id);

    // Status index
    const status = vesting.status ?? "vesting";
    if (!this.statusIndex.has(status)) {
      this.statusIndex.set(status, new Set());
    }
    this.statusIndex.get(status)!.add(vesting.id);

    // Category index
    if (!this.categoryIndex.has(vesting.category)) {
      this.categoryIndex.set(vesting.category, new Set());
    }
    this.categoryIndex.get(vesting.category)!.add(vesting.id);

    // Update stats
    this.stats.totalVestings++;
    this.stats.totalAmount += vesting.totalAmount;
    this.stats.totalReleased += vesting.releasedAmount ?? 0n;
    this.stats.totalClaimed += vesting.claimableAmount ?? 0n;

    if (status === "vesting") this.stats.activeVestings++;
    else if (status === "paused") this.stats.pausedVestings++;
    else if (status === "completed") this.stats.completedVestings++;
    else if (status === "falsified") this.stats.falsifiedVestings++;
  }

  /**
   * Clone the state tree
   */
  clone(): VestingStateTree {
    const cloned = new VestingStateTree();

    // Clone cache and rebuild SMT
    for (const [id, vesting] of this.vestingCache) {
      cloned.vestingCache.set(id, { ...vesting });
      const key = vestingIdToKey(id);
      cloned.vestingSmt.set(key, serializeVesting(vesting));
    }

    // Clone indexes
    for (const [claimId, vestingId] of this.claimToVestingIndex) {
      cloned.claimToVestingIndex.set(claimId, vestingId);
    }

    for (const [recipient, vestingIds] of this.recipientToVestingsIndex) {
      cloned.recipientToVestingsIndex.set(recipient, new Set(vestingIds));
    }

    for (const [status, vestingIds] of this.statusIndex) {
      cloned.statusIndex.set(status, new Set(vestingIds));
    }

    for (const [category, vestingIds] of this.categoryIndex) {
      cloned.categoryIndex.set(category, new Set(vestingIds));
    }

    // Clone stats
    cloned.stats = { ...this.stats };

    return cloned;
  }

  /**
   * Clear all state (for testing)
   */
  clear(): void {
    this.vestingSmt = new InMemorySMT();
    this.vestingCache.clear();
    this.claimToVestingIndex.clear();
    this.recipientToVestingsIndex.clear();

    for (const status of this.statusIndex.keys()) {
      this.statusIndex.set(status, new Set());
    }
    this.categoryIndex.clear();

    this.stats = {
      totalVestings: 0,
      totalAmount: 0n,
      totalReleased: 0n,
      totalClaimed: 0n,
      activeVestings: 0,
      pausedVestings: 0,
      completedVestings: 0,
      falsifiedVestings: 0,
    };
  }
}
