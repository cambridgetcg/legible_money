import type { DID } from '../../agents/identity/did.js';

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
const DECAY_RATE = 0.01; // 1% decay per epoch
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
    score.reliability = Math.max(0, Math.min(1000, score.reliability + reliabilityDelta));

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
      .sort((a, b) => b.score.overall - a.score.overall)
      .slice(0, limit);
  }

  /**
   * Check if agent meets minimum reputation threshold
   */
  meetsThreshold(agent: DID, threshold: number): boolean {
    const score = this.getReputation(agent);
    return score.overall >= threshold && score.totalTransactions >= MIN_TRANSACTIONS_FOR_SCORE;
  }

  /**
   * Apply reputation decay (called at epoch boundaries)
   */
  applyDecay(currentBlock: number): void {
    const epochsPassed = Math.floor((currentBlock - this.currentBlock) / EPOCH_BLOCKS);

    if (epochsPassed === 0) return;

    for (const [did, score] of this.scores) {
      // Decay towards neutral (500)
      const decayFactor = Math.pow(1 - DECAY_RATE, epochsPassed);

      score.overall = 500 + Math.round((score.overall - 500) * decayFactor);
      score.reliability = 500 + Math.round((score.reliability - 500) * decayFactor);
      score.responsiveness = 500 + Math.round((score.responsiveness - 500) * decayFactor);
      score.quality = 500 + Math.round((score.quality - 500) * decayFactor);
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
   * Calculate reputation weight for voting/staking
   */
  getVotingWeight(agent: DID, stakeAmount: bigint): bigint {
    const score = this.getReputation(agent);

    // Reputation multiplier: 0.5x to 1.5x based on score
    const multiplier = 0.5 + (score.overall / 1000);

    return BigInt(Math.floor(Number(stakeAmount) * multiplier));
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

    // Quality = (positives - negatives) / total, normalized to 0-1000
    const qualityRatio = (positives - negatives) / total;
    score.quality = Math.round(500 + qualityRatio * 500);

    this.recalculateOverall(agent);
  }

  private recalculateOverall(agent: DID): void {
    const score = this.getReputation(agent);

    // Weighted average of components
    score.overall = Math.round(
      score.reliability * 0.4 +
      score.quality * 0.4 +
      score.responsiveness * 0.2
    );

    score.lastUpdated = this.currentBlock;
  }
}

/**
 * Singleton reputation system instance
 */
export const reputationSystem = new ReputationSystem();
