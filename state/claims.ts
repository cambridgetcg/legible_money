/**
 * Claim State Tree
 *
 * Sparse Merkle Tree storage for claim submissions.
 * Tracks claims through their lifecycle: pending → verifying → accepted/rejected.
 *
 * Indexes:
 * - By claim ID (primary)
 * - By submitter (for user queries)
 * - By status (for batch processing)
 * - By domain (for domain-specific queries)
 *
 * PP HIERARCHY:
 *   TRUTH: SMT provides cryptographic proofs of claim state
 *   UNDERSTANDING: Clear indexing for different query patterns
 *   BEAUTY: Follows existing state tree patterns in codebase
 *   JUSTICE: Deterministic state transitions for all operations
 */

import { InMemorySMT, type MerkleRoot } from "../core/crypto/merkle.js";
import { sha512_256Hex, domainHash } from "../core/crypto/hashing.js";
import { type ClaimState, type ClaimStatus, StateError } from "./types.js";

// ============ Key Generation ============

/**
 * Generate SMT key from claim ID
 */
function claimIdToKey(claimId: string): Uint8Array {
  const hash = sha512_256Hex(`claim:id:${claimId}`);
  return new Uint8Array(Buffer.from(hash, "hex").slice(0, 32));
}

/**
 * Generate deterministic claim ID from claim content + submitter + block
 */
export function generateClaimId(
  claim: string,
  submitter: string,
  block: bigint,
): string {
  const input = `${claim}:${submitter}:${block}`;
  const hash = domainHash("LGM.claim.id.v1", Buffer.from(input));
  return `claim:${Buffer.from(hash).toString("hex").slice(0, 32)}`;
}

// ============ Serialization ============

/**
 * Serialize claim for SMT storage
 */
function serializeClaim(claim: ClaimState): Uint8Array {
  const json = JSON.stringify(claim, (_, value) =>
    typeof value === "bigint" ? value.toString() + "n" : value,
  );
  return new TextEncoder().encode(json);
}

/**
 * Deserialize claim from SMT storage
 */
function deserializeClaim(data: Uint8Array): ClaimState {
  const json = new TextDecoder().decode(data);
  return JSON.parse(json, (_, value) => {
    if (typeof value === "string" && value.endsWith("n")) {
      return BigInt(value.slice(0, -1));
    }
    return value;
  });
}

// ============ Claim State Tree ============

export class ClaimStateTree {
  private claimSmt: InMemorySMT;
  private claimCache: Map<string, ClaimState>;

  // Indexes for efficient queries
  private submitterIndex: Map<string, Set<string>>; // submitter -> claimIds
  private statusIndex: Map<ClaimStatus, Set<string>>; // status -> claimIds
  private domainIndex: Map<string, Set<string>>; // domain -> claimIds
  private roundIndex: Map<string, string>; // roundId -> claimId

  // Statistics
  private stats: {
    totalClaims: number;
    pendingClaims: number;
    verifyingClaims: number;
    acceptedClaims: number;
    rejectedClaims: number;
    expiredClaims: number;
    withdrawnClaims: number;
    totalStakeLocked: bigint;
  };

  constructor() {
    this.claimSmt = new InMemorySMT();
    this.claimCache = new Map();
    this.submitterIndex = new Map();
    this.statusIndex = new Map();
    this.domainIndex = new Map();
    this.roundIndex = new Map();

    // Initialize status indexes
    const statuses: ClaimStatus[] = [
      "pending",
      "verifying",
      "accepted",
      "rejected",
      "expired",
      "withdrawn",
    ];
    for (const status of statuses) {
      this.statusIndex.set(status, new Set());
    }

    this.stats = {
      totalClaims: 0,
      pendingClaims: 0,
      verifyingClaims: 0,
      acceptedClaims: 0,
      rejectedClaims: 0,
      expiredClaims: 0,
      withdrawnClaims: 0,
      totalStakeLocked: 0n,
    };
  }

  // ============ Core Operations ============

  /**
   * Get claim root hash
   */
  getRoot(): MerkleRoot {
    return this.claimSmt.getRoot();
  }

  /**
   * Add a new claim to the tree
   */
  addClaim(claim: ClaimState): void {
    // Check for duplicates
    if (this.claimCache.has(claim.id)) {
      throw new StateError(
        `Claim ${claim.id} already exists`,
        "INVALID_TRANSITION",
      );
    }

    // Store in SMT
    const key = claimIdToKey(claim.id);
    this.claimSmt.set(key, serializeClaim(claim));

    // Update cache
    this.claimCache.set(claim.id, claim);

    // Update indexes
    this.addToSubmitterIndex(claim.submitter, claim.id);
    this.addToStatusIndex(claim.status, claim.id);
    this.addToDomainIndex(claim.domain, claim.id);

    // Update stats
    this.stats.totalClaims++;
    this.updateStatusStats(claim.status, 1);
    if (claim.status === "pending" || claim.status === "verifying") {
      this.stats.totalStakeLocked += claim.stake;
    }
  }

  /**
   * Get claim by ID
   */
  getClaim(claimId: string): ClaimState | undefined {
    // Check cache first
    const cached = this.claimCache.get(claimId);
    if (cached) return cached;

    // Try SMT
    const key = claimIdToKey(claimId);
    const data = this.claimSmt.get(key);
    if (!data) return undefined;

    const claim = deserializeClaim(data);
    this.claimCache.set(claimId, claim);
    return claim;
  }

  /**
   * Update claim status
   */
  updateClaimStatus(
    claimId: string,
    newStatus: ClaimStatus,
    blockNumber: bigint,
    updates?: Partial<ClaimState>,
  ): ClaimState {
    const claim = this.getClaim(claimId);
    if (!claim) {
      throw new StateError(`Claim ${claimId} not found`, "CLAIM_NOT_FOUND");
    }

    const oldStatus = claim.status;

    // Create updated claim
    const updatedClaim: ClaimState = {
      ...claim,
      ...updates,
      status: newStatus,
      statusChangedAt: blockNumber,
    };

    // Store in SMT
    const key = claimIdToKey(claimId);
    this.claimSmt.set(key, serializeClaim(updatedClaim));

    // Update cache
    this.claimCache.set(claimId, updatedClaim);

    // Update status indexes
    this.removeFromStatusIndex(oldStatus, claimId);
    this.addToStatusIndex(newStatus, claimId);

    // Update round index if verification started
    if (updatedClaim.verificationRoundId && !claim.verificationRoundId) {
      this.roundIndex.set(updatedClaim.verificationRoundId, claimId);
    }

    // Update stats
    this.updateStatusStats(oldStatus, -1);
    this.updateStatusStats(newStatus, 1);

    // Handle stake tracking
    const wasLocked = oldStatus === "pending" || oldStatus === "verifying";
    const isLocked = newStatus === "pending" || newStatus === "verifying";
    if (wasLocked && !isLocked) {
      this.stats.totalStakeLocked -= claim.stake;
    } else if (!wasLocked && isLocked) {
      this.stats.totalStakeLocked += updatedClaim.stake;
    }

    return updatedClaim;
  }

  // ============ Query Methods ============

  /**
   * Get claims by submitter
   */
  getClaimsBySubmitter(submitter: string): ClaimState[] {
    const claimIds = this.submitterIndex.get(submitter);
    if (!claimIds) return [];

    return Array.from(claimIds)
      .map((id) => this.getClaim(id))
      .filter((c): c is ClaimState => c !== undefined);
  }

  /**
   * Get claims by status
   */
  getClaimsByStatus(status: ClaimStatus): ClaimState[] {
    const claimIds = this.statusIndex.get(status);
    if (!claimIds) return [];

    return Array.from(claimIds)
      .map((id) => this.getClaim(id))
      .filter((c): c is ClaimState => c !== undefined);
  }

  /**
   * Get pending claims (for verification selection)
   */
  getPendingClaims(): ClaimState[] {
    return this.getClaimsByStatus("pending");
  }

  /**
   * Get claims by domain
   */
  getClaimsByDomain(domain: string): ClaimState[] {
    const claimIds = this.domainIndex.get(domain);
    if (!claimIds) return [];

    return Array.from(claimIds)
      .map((id) => this.getClaim(id))
      .filter((c): c is ClaimState => c !== undefined);
  }

  /**
   * Get claim by verification round ID
   */
  getClaimByRoundId(roundId: string): ClaimState | undefined {
    const claimId = this.roundIndex.get(roundId);
    if (!claimId) return undefined;
    return this.getClaim(claimId);
  }

  /**
   * Check if claim exists
   */
  hasClaim(claimId: string): boolean {
    return this.claimCache.has(claimId) || this.getClaim(claimId) !== undefined;
  }

  // ============ Statistics ============

  /**
   * Get claim statistics
   */
  getStats(): typeof this.stats {
    return { ...this.stats };
  }

  /**
   * Get pending claims count
   */
  getPendingCount(): number {
    return this.stats.pendingClaims;
  }

  /**
   * Get total stake locked in pending/verifying claims
   */
  getTotalStakeLocked(): bigint {
    return this.stats.totalStakeLocked;
  }

  // ============ Index Management ============

  private addToSubmitterIndex(submitter: string, claimId: string): void {
    let claims = this.submitterIndex.get(submitter);
    if (!claims) {
      claims = new Set();
      this.submitterIndex.set(submitter, claims);
    }
    claims.add(claimId);
  }

  private addToStatusIndex(status: ClaimStatus, claimId: string): void {
    const claims = this.statusIndex.get(status);
    if (claims) {
      claims.add(claimId);
    }
  }

  private removeFromStatusIndex(status: ClaimStatus, claimId: string): void {
    const claims = this.statusIndex.get(status);
    if (claims) {
      claims.delete(claimId);
    }
  }

  private addToDomainIndex(domain: string, claimId: string): void {
    let claims = this.domainIndex.get(domain);
    if (!claims) {
      claims = new Set();
      this.domainIndex.set(domain, claims);
    }
    claims.add(claimId);
  }

  private updateStatusStats(status: ClaimStatus, delta: number): void {
    switch (status) {
      case "pending":
        this.stats.pendingClaims += delta;
        break;
      case "verifying":
        this.stats.verifyingClaims += delta;
        break;
      case "accepted":
        this.stats.acceptedClaims += delta;
        break;
      case "rejected":
        this.stats.rejectedClaims += delta;
        break;
      case "expired":
        this.stats.expiredClaims += delta;
        break;
      case "withdrawn":
        this.stats.withdrawnClaims += delta;
        break;
    }
  }

  // ============ State Export/Import ============

  /**
   * Export all claims (for snapshots)
   */
  exportClaims(): ClaimState[] {
    return Array.from(this.claimCache.values());
  }

  /**
   * Import claims (for recovery)
   */
  importClaims(claims: ClaimState[]): void {
    for (const claim of claims) {
      // Store in SMT
      const key = claimIdToKey(claim.id);
      this.claimSmt.set(key, serializeClaim(claim));

      // Update cache
      this.claimCache.set(claim.id, claim);

      // Update indexes
      this.addToSubmitterIndex(claim.submitter, claim.id);
      this.addToStatusIndex(claim.status, claim.id);
      this.addToDomainIndex(claim.domain, claim.id);

      if (claim.verificationRoundId) {
        this.roundIndex.set(claim.verificationRoundId, claim.id);
      }

      // Update stats
      this.stats.totalClaims++;
      this.updateStatusStats(claim.status, 1);
      if (claim.status === "pending" || claim.status === "verifying") {
        this.stats.totalStakeLocked += claim.stake;
      }
    }
  }

  /**
   * Clear all claims (for testing)
   */
  clear(): void {
    this.claimSmt = new InMemorySMT();
    this.claimCache.clear();
    this.submitterIndex.clear();
    this.domainIndex.clear();
    this.roundIndex.clear();

    // Reinitialize status indexes
    const statuses: ClaimStatus[] = [
      "pending",
      "verifying",
      "accepted",
      "rejected",
      "expired",
      "withdrawn",
    ];
    for (const status of statuses) {
      this.statusIndex.set(status, new Set());
    }

    this.stats = {
      totalClaims: 0,
      pendingClaims: 0,
      verifyingClaims: 0,
      acceptedClaims: 0,
      rejectedClaims: 0,
      expiredClaims: 0,
      withdrawnClaims: 0,
      totalStakeLocked: 0n,
    };
  }

  /**
   * Clone the tree (for speculative execution / transaction rollback)
   * GAP-006 FIX: Required for StateManager transaction support
   */
  clone(): ClaimStateTree {
    const cloned = new ClaimStateTree();

    // Clone cache and rebuild SMT
    for (const [id, claim] of this.claimCache) {
      cloned.claimCache.set(id, { ...claim });
      const key = claimIdToKey(id);
      cloned.claimSmt.set(key, serializeClaim(claim));
    }

    // Clone indexes
    for (const [submitter, claimIds] of this.submitterIndex) {
      cloned.submitterIndex.set(submitter, new Set(claimIds));
    }

    for (const [status, claimIds] of this.statusIndex) {
      cloned.statusIndex.set(status, new Set(claimIds));
    }

    for (const [domain, claimIds] of this.domainIndex) {
      cloned.domainIndex.set(domain, new Set(claimIds));
    }

    for (const [roundId, claimId] of this.roundIndex) {
      cloned.roundIndex.set(roundId, claimId);
    }

    // Clone stats
    cloned.stats = { ...this.stats };

    return cloned;
  }
}
