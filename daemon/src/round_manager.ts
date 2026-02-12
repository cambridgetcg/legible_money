/**
 * Round Manager
 *
 * Tracks active verification rounds and prevents double-processing.
 * The processedRounds set is the primary defense against equivocation
 * (submitting conflicting commitments), which carries a 100% slash penalty.
 */

import type { RPCClient } from "./rpc_client.js";
import type { ActiveRound, ClaimData } from "./types.js";

/** Max age in blocks before pruning processed round IDs */
const DEFAULT_MAX_ROUND_AGE = 200n;

export class RoundManager {
  private readonly rpc: RPCClient;
  private readonly validatorAddress: string;

  /**
   * Set of round IDs that have been committed to.
   * Once a round is in this set, it MUST NOT be processed again.
   * This is the primary equivocation defense.
   */
  private processedRounds: Map<string, bigint> = new Map(); // roundId → height when processed

  constructor(rpc: RPCClient, validatorAddress: string) {
    this.rpc = rpc;
    this.validatorAddress = validatorAddress;
  }

  /**
   * Fetch active rounds in commit phase that this validator is selected for.
   * Filters to only rounds the validator is VRF-selected for.
   */
  async getCommitRounds(height: bigint): Promise<ActiveRound[]> {
    const rounds = await this.rpc.getActiveRounds(height);
    const commitRounds = rounds.filter((r) => r.phase === "commit");

    // Filter to rounds this validator is selected for
    const selected: ActiveRound[] = [];
    for (const round of commitRounds) {
      const isSelected = await this.rpc.isSelectedForRound(
        round.id,
        this.validatorAddress,
      );
      if (isSelected) {
        selected.push(round);
      }
    }

    return selected;
  }

  /**
   * Fetch active rounds in reveal phase (for submitting reveals).
   */
  async getRevealRounds(height: bigint): Promise<ActiveRound[]> {
    const rounds = await this.rpc.getActiveRounds(height);
    return rounds.filter((r) => r.phase === "reveal");
  }

  /**
   * Fetch full claim data for evaluation, enriched with counter-fact data
   * for any referenced facts. This gives Claude (and the deterministic
   * fallback) awareness of negative knowledge in the fact graph.
   */
  async fetchClaimData(claimId: string): Promise<ClaimData> {
    const claim = await this.rpc.getClaimData(claimId);

    // Enrich references with counter-fact data
    for (const ref of claim.references) {
      try {
        const [counterFacts, bidir] = await Promise.all([
          this.rpc.getCounterFactsByFact(ref.factId),
          this.rpc.getBidirectionalConfidence(ref.factId),
        ]);
        ref.counterFacts = counterFacts.filter(
          (cf) => cf.status === "verified",
        );
        ref.bidirectional = bidir;
      } catch {
        // Non-critical: if counter-fact queries fail, proceed without them
      }
    }

    return claim;
  }

  /**
   * Check if a round has already been processed (committed to).
   * Returns true if the round should be SKIPPED.
   */
  isProcessed(roundId: string): boolean {
    return this.processedRounds.has(roundId);
  }

  /**
   * Mark a round as processed after successful commitment.
   * Once marked, this round will NEVER be processed again.
   */
  markProcessed(roundId: string, height: bigint): void {
    this.processedRounds.set(roundId, height);
  }

  /**
   * Prune old processed round IDs to prevent unbounded memory growth.
   * Only prunes entries older than maxAge blocks.
   */
  cleanupOld(
    currentHeight: bigint,
    maxAge: bigint = DEFAULT_MAX_ROUND_AGE,
  ): void {
    if (currentHeight < maxAge) return;
    const cutoff = currentHeight - maxAge;

    for (const [id, height] of this.processedRounds) {
      if (height < cutoff) {
        this.processedRounds.delete(id);
      }
    }
  }

  /** Number of currently tracked processed rounds */
  processedCount(): number {
    return this.processedRounds.size;
  }
}
