import type { DID } from "../../core/agents/identity/did.js";

/**
 * Reputation score components
 */
export interface ReputationScore {
  /** Overall reputation (0-1000) */
  overall: number;
  /** Transaction success rate */
  reliability: number;
  /** Average response time score */
  responsiveness: number;
  /** Quality of service score */
  quality: number;
  /** Total completed transactions */
  totalTransactions: number;
  /** Total volume traded (in AGC) */
  totalVolume: bigint;
  /** Last updated block height */
  lastUpdated: number;
}

/**
 * Reputation attestation from another agent
 */
export interface ReputationAttestation {
  id: string;
  from: DID;
  to: DID;
  transactionId: string;
  score: -1 | 0 | 1; // negative, neutral, positive
  evidence?: string;
  timestamp: number;
  blockHeight: number;
}

/**
 * Reputation decay parameters
 */
const EPOCH_BLOCKS = 1000; // Decay applied every 1000 blocks
const MIN_TRANSACTIONS_FOR_SCORE = 5;

/**
 * Reputation system for AI agents
 */
export class ReputationSystem {
  private scores: Map<DID, ReputationScore> = new Map();
  private attestations: Map<DID, ReputationAttestation[]> = new Map();
  private currentBlock = 0;

  /**
   * Initialize or get reputation for an agent
   */
  getReputation(agent: DID): ReputationScore {
    let score = this.scores.get(agent);

    if (!score) {
      score = {
        overall: 500, // Start neutral
        reliability: 500,
        responsiveness: 500,
        quality: 500,
        totalTransactions: 0,
        totalVolume: 0n,
        lastUpdated: this.currentBlock,
      };
      this.scores.set(agent, score);
    }

    return score;
  }

  /**
   * Submit a reputation attestation
   */
  submitAttestation(attestation: ReputationAttestation): void {
    // Store attestation
    const existing = this.attestations.get(attestation.to) || [];
    existing.push(attestation);
    this.attestations.set(attestation.to, existing);

    // Update score
    this.recalculateScore(attestation.to);
  }

  /**
   * Record a completed transaction (updates volume and count)
   */
  recordTransaction(agent: DID, volume: bigint, success: boolean): void {
    const score = this.getReputation(agent);

    score.totalTransactions++;
    score.totalVolume += volume;

    // Update reliability based on success
    const reliabilityDelta = success ? 10 : -20;
    score.reliability = Math.max(
      0,
      Math.min(1000, score.reliability + reliabilityDelta),
    );

    this.recalculateOverall(agent);
  }

  /**
   * Get agents sorted by reputation (for validator selection, etc.)
   */
  getTopAgents(limit = 100): Array<{ did: DID; score: ReputationScore }> {
    const agents: Array<{ did: DID; score: ReputationScore }> = [];

    for (const [did, score] of this.scores) {
      if (score.totalTransactions >= MIN_TRANSACTIONS_FOR_SCORE) {
        agents.push({ did, score });
      }
    }

    return agents
      .sort(
        (a, b) =>
          b.score.overall - a.score.overall ||
          (a.did < b.did ? -1 : a.did > b.did ? 1 : 0),
      )
      .slice(0, limit);
  }

  /**
   * Check if agent meets minimum reputation threshold
   */
  meetsThreshold(agent: DID, threshold: number): boolean {
    const score = this.getReputation(agent);
    return (
      score.overall >= threshold &&
      score.totalTransactions >= MIN_TRANSACTIONS_FOR_SCORE
    );
  }

  /**
   * Apply reputation decay (called at epoch boundaries).
   * Uses integer-only iterated decay to avoid Math.pow nondeterminism.
   * Each epoch: score = 500 + ((score - 500) * (10000 - DECAY_RATE_BPS)) / 10000
   */
  applyDecay(currentBlock: number): void {
    const epochsPassed = Math.floor(
      (currentBlock - this.currentBlock) / EPOCH_BLOCKS,
    );

    if (epochsPassed === 0) return;

    // Decay rate in basis points: 1% = 100 BPS
    const DECAY_RATE_BPS = 100;
    const RETAIN_BPS = 10000 - DECAY_RATE_BPS; // 9900

    for (const [_did, score] of this.scores) {
      // Apply decay iteratively for determinism
      // For large epoch gaps, cap iterations to prevent DoS
      const iterations = Math.min(epochsPassed, 1000);

      let overall = score.overall - 500;
      let reliability = score.reliability - 500;
      let responsiveness = score.responsiveness - 500;
      let quality = score.quality - 500;

      for (let i = 0; i < iterations; i++) {
        overall = Math.trunc((overall * RETAIN_BPS) / 10000);
        reliability = Math.trunc((reliability * RETAIN_BPS) / 10000);
        responsiveness = Math.trunc((responsiveness * RETAIN_BPS) / 10000);
        quality = Math.trunc((quality * RETAIN_BPS) / 10000);
      }

      score.overall = 500 + overall;
      score.reliability = 500 + reliability;
      score.responsiveness = 500 + responsiveness;
      score.quality = 500 + quality;
      score.lastUpdated = currentBlock;
    }

    this.currentBlock = currentBlock;
  }

  /**
   * Get attestation history for an agent
   */
  getAttestations(agent: DID, limit = 50): ReputationAttestation[] {
    const attestations = this.attestations.get(agent) || [];
    return attestations.slice(-limit);
  }

  /**
   * Calculate reputation weight for voting/staking.
   * Uses pure BigInt math to avoid Number(stakeAmount) precision loss for large stakes.
   * Multiplier: 0.5x to 1.5x based on score (500-1500 BPS out of 1000).
   */
  getVotingWeight(agent: DID, stakeAmount: bigint): bigint {
    const score = this.getReputation(agent);

    // multiplierBps = 500 + overall (score.overall ranges 0-1000)
    // This maps to 0.5x at score=0, 1.0x at score=500, 1.5x at score=1000
    const multiplierBps = BigInt(500 + score.overall);

    return (stakeAmount * multiplierBps) / 1000n;
  }

  private recalculateScore(agent: DID): void {
    const attestations = this.attestations.get(agent) || [];
    const score = this.getReputation(agent);

    if (attestations.length === 0) return;

    // Calculate quality score from recent attestations
    const recentAttestations = attestations.slice(-100);
    const positives = recentAttestations.filter((a) => a.score === 1).length;
    const negatives = recentAttestations.filter((a) => a.score === -1).length;
    const total = recentAttestations.length;

    // Quality = 500 + ((positives - negatives) * 500) / total
    // Pure integer arithmetic — no float division
    score.quality = 500 + Math.trunc(((positives - negatives) * 500) / total);

    this.recalculateOverall(agent);
  }

  private recalculateOverall(agent: DID): void {
    const score = this.getReputation(agent);

    // Weighted average using basis points: 40% = 4000 BPS, 20% = 2000 BPS
    // overall = (reliability * 4000 + quality * 4000 + responsiveness * 2000) / 10000
    score.overall = Math.trunc(
      (score.reliability * 4000 +
        score.quality * 4000 +
        score.responsiveness * 2000) /
        10000,
    );

    score.lastUpdated = this.currentBlock;
  }
}

/**
 * Singleton reputation system instance
 */
export const reputationSystem = new ReputationSystem();
