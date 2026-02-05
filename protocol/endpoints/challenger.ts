/**
 * Challenger Endpoint Implementation
 *
 * Challengers dispute claims, evidence, and facts:
 * - Initiate challenges with counter-evidence
 * - Submit additional evidence during dispute
 * - Escalate disputes if needed
 * - Earn rewards for successful challenges
 *
 * Economic model: "Intention does not matter, result does"
 * - Win: 60% of defender's stake + bonuses
 * - Lose: 30% of own stake slashed
 */

import type {
  ChallengeMessage,
  ChallengePayload,
  ChallengeType,
  ResolutionMessage,
} from "../../core/messaging/types.js";
import type {
  ChallengerEndpoint,
  EndpointRole,
  EndpointCapability,
} from "./types.js";
import { BaseEndpoint, type EndpointConfig, type SendOptions } from "./base.js";

// ============ Types ============

/**
 * Challenge parameters
 */
export interface ChallengeParams {
  /** What is being challenged */
  targetType: "claim" | "evidence" | "fact" | "verification";
  /** ID of the target */
  targetId: string;
  /** Reason for challenge */
  reason: string;
  /** Type of challenge */
  challengeType: ChallengeType;
  /** Counter-evidence IDs */
  counterEvidenceIds: string[];
  /** Challenge stake */
  stake: bigint;
  /** Requested outcome */
  requestedOutcome: "retraction" | "correction" | "confidence_reduction";
  /** Proposed correction (if requestedOutcome is 'correction') */
  proposedCorrection?: string;
  /** Knowledge domain */
  domain?: string;
}

/**
 * Escalation parameters
 */
export interface EscalationParams {
  /** Challenge/dispute ID */
  disputeId: string;
  /** Reason for escalation */
  reason: string;
  /** Additional stake for escalation */
  additionalStake: bigint;
  /** Additional evidence IDs */
  additionalEvidenceIds?: string[];
}

/**
 * Challenge status
 */
export interface ChallengeStatus {
  challengeId: string;
  targetId: string;
  targetType: string;
  status: "pending" | "active" | "resolved";
  stake: bigint;
  outcome?: "won" | "lost" | "draw" | "withdrawn";
  reward?: bigint;
  penalty?: bigint;
}

// ============ Challenger Implementation ============

/**
 * Challenger endpoint for disputing claims/evidence
 */
export class Challenger extends BaseEndpoint implements ChallengerEndpoint {
  // ChallengerEndpoint fields
  challengesInitiated = 0;
  challengesWon = 0;
  challengeRewardsEarned = 0n;

  // Active challenges
  private readonly activeChallenges = new Map<string, ChallengeStatus>();

  constructor(config: EndpointConfig) {
    super({
      ...config,
      roles: [
        "challenger",
        ...config.roles.filter((r: EndpointRole) => r !== "challenger"),
      ],
      capabilities: [
        "challenge_initiate",
        "challenge_respond",
        "evidence_submit",
        ...config.capabilities.filter(
          (c: EndpointCapability) =>
            ![
              "challenge_initiate",
              "challenge_respond",
              "evidence_submit",
            ].includes(c),
        ),
      ],
    });

    // Subscribe to dispute resolutions
    this.eventBus.on("dispute_resolved" as never, (event) => {
      this.handleDisputeResolved(
        event as unknown as { payload: ResolutionMessage },
      );
    });
  }

  /**
   * Challenge success rate (won / initiated)
   */
  get challengeSuccessRate(): number {
    if (this.challengesInitiated === 0) return 0;
    return this.challengesWon / this.challengesInitiated;
  }

  // ============ Challenge Initiation ============

  /**
   * Initiate a challenge
   */
  async challenge(
    params: ChallengeParams,
    options: SendOptions = {},
  ): Promise<ChallengeMessage> {
    // Validate stake
    if (params.stake <= 0n) {
      throw new Error("Stake must be positive");
    }
    if (params.stake > this.stakedAmount) {
      throw new Error("Insufficient staked balance");
    }

    // Validate counter-evidence
    if (params.counterEvidenceIds.length === 0) {
      throw new Error("Must provide at least one counter-evidence");
    }

    // Create payload
    const payload: ChallengePayload = {
      targetType: params.targetType,
      targetId: params.targetId,
      reason: params.reason,
      challengeType: params.challengeType,
      counterEvidenceIds: params.counterEvidenceIds,
      stake: params.stake,
      requestedOutcome: params.requestedOutcome,
      proposedCorrection: params.proposedCorrection,
    };

    // Create and finalize message
    const messageBase = this.createMessage("challenge", payload, {
      ...options,
      domain: params.domain,
      recipient: "broadcast",
      referencedMessages: [params.targetId, ...params.counterEvidenceIds],
    });

    const message = (await this.finalizeMessage(
      messageBase,
    )) as ChallengeMessage;

    // Track challenge
    this.activeChallenges.set(message.messageId, {
      challengeId: message.messageId,
      targetId: params.targetId,
      targetType: params.targetType,
      status: "pending",
      stake: params.stake,
    });
    this.challengesInitiated++;

    // Send
    await this.send(message);

    return message;
  }

  /**
   * Challenge a claim as factually incorrect
   */
  async challengeClaim(
    claimId: string,
    reason: string,
    counterEvidenceIds: string[],
    stake: bigint,
    options: SendOptions = {},
  ): Promise<ChallengeMessage> {
    return this.challenge(
      {
        targetType: "claim",
        targetId: claimId,
        reason,
        challengeType: "factual_error",
        counterEvidenceIds,
        stake,
        requestedOutcome: "retraction",
      },
      options,
    );
  }

  /**
   * Challenge evidence as unreliable
   */
  async challengeEvidence(
    evidenceId: string,
    reason: string,
    challengeType: ChallengeType,
    counterEvidenceIds: string[],
    stake: bigint,
    options: SendOptions = {},
  ): Promise<ChallengeMessage> {
    return this.challenge(
      {
        targetType: "evidence",
        targetId: evidenceId,
        reason,
        challengeType,
        counterEvidenceIds,
        stake,
        requestedOutcome: "retraction",
      },
      options,
    );
  }

  /**
   * Challenge a verified fact
   */
  async challengeFact(
    factId: string,
    reason: string,
    challengeType: ChallengeType,
    counterEvidenceIds: string[],
    stake: bigint,
    requestedOutcome: "retraction" | "correction" | "confidence_reduction",
    proposedCorrection?: string,
    options: SendOptions = {},
  ): Promise<ChallengeMessage> {
    return this.challenge(
      {
        targetType: "fact",
        targetId: factId,
        reason,
        challengeType,
        counterEvidenceIds,
        stake,
        requestedOutcome,
        proposedCorrection,
      },
      options,
    );
  }

  // ============ Challenge Management ============

  /**
   * Add evidence to an active challenge
   */
  async addChallengeEvidence(
    challengeId: string,
    evidenceIds: string[],
    options: SendOptions = {},
  ): Promise<void> {
    const challenge = this.activeChallenges.get(challengeId);
    if (!challenge) {
      throw new Error("Challenge not found");
    }
    if (challenge.status !== "active") {
      throw new Error("Challenge is not active");
    }

    // Submit additional evidence referencing the challenge
    for (const evidenceId of evidenceIds) {
      const payload = {
        evidenceType: "attestation" as const,
        contentHash: evidenceId,
        originalSource: `challenge:${challengeId}`,
        targetClaimId:
          challenge.targetType === "claim" ? challenge.targetId : undefined,
        targetFactId:
          challenge.targetType === "fact" ? challenge.targetId : undefined,
        direction: "contradicting" as const,
        intrinsicStrength: 0.7,
        independence: {
          methodIndependence: 0.7,
          sourceIndependence: 0.7,
          temporalIndependence: 0.8,
          aggregateIndependence: 0.7,
        },
        archiveRequested: false,
      };

      const messageBase = this.createMessage("evidence", payload, {
        ...options,
        causedBy: challengeId,
        referencedMessages: [challengeId],
      });

      const message = await this.finalizeMessage(messageBase);
      await this.send(message);
    }
  }

  /**
   * Escalate a challenge to higher tier
   */
  async escalateChallenge(
    params: EscalationParams,
    options: SendOptions = {},
  ): Promise<void> {
    const challenge = this.activeChallenges.get(params.disputeId);
    if (!challenge) {
      throw new Error("Challenge not found");
    }

    // Validate additional stake
    if (params.additionalStake <= 0n) {
      throw new Error("Additional stake must be positive");
    }

    // Create escalation message
    const payload = {
      targetType: "challenge" as const,
      targetId: params.disputeId,
      attestationType: "qualify" as const,
      qualification: `ESCALATION: ${params.reason}`,
      confidence: 1.0,
      stake: params.additionalStake,
      reasoning: params.reason,
    };

    const messageBase = this.createMessage("attestation", payload as never, {
      ...options,
      causedBy: params.disputeId,
      referencedMessages: [
        params.disputeId,
        ...(params.additionalEvidenceIds ?? []),
      ],
    });

    const message = await this.finalizeMessage(messageBase);

    // Update stake
    challenge.stake += params.additionalStake;

    await this.send(message);
  }

  /**
   * Withdraw a challenge (with stake penalty)
   */
  async withdrawChallenge(
    challengeId: string,
    reason: string,
    options: SendOptions = {},
  ): Promise<void> {
    const challenge = this.activeChallenges.get(challengeId);
    if (!challenge) {
      throw new Error("Challenge not found");
    }

    // Create withdrawal message
    const payload = {
      targetType: "challenge" as const,
      targetId: challengeId,
      outcome: "withdrawn" as const,
      finalConfidence: 0,
      resolvedAtBlock: 0n,
      rewards: [],
      slashes: [
        {
          target: this.did,
          amount: challenge.stake / 10n, // 10% withdrawal penalty
          reason: "Challenge withdrawal",
        },
      ],
      withdrawalReason: reason,
    };

    const messageBase = this.createMessage("resolution", payload as never, {
      ...options,
      causedBy: challengeId,
    });

    const message = await this.finalizeMessage(messageBase);
    await this.send(message);

    // Update status
    challenge.status = "resolved";
    challenge.outcome = "withdrawn";
    challenge.penalty = challenge.stake / 10n;
  }

  // ============ Event Handlers ============

  /**
   * Handle dispute resolution
   */
  private handleDisputeResolved(event: { payload: ResolutionMessage }): void {
    const resolution = event.payload;

    // Find matching challenge
    const challenge = this.activeChallenges.get(resolution.payload.targetId);
    if (!challenge) return;

    // Update status
    challenge.status = "resolved";

    // Determine outcome
    const ourReward = resolution.payload.rewards.find(
      (r) => r.recipient === this.did,
    );
    const ourSlash = resolution.payload.slashes.find(
      (s) => s.target === this.did,
    );

    if (ourReward) {
      challenge.outcome = "won";
      challenge.reward = ourReward.amount;
      this.challengesWon++;
      this.challengeRewardsEarned += ourReward.amount;
      this.updateReputation(0.03); // Significant reputation boost for successful challenge
    } else if (ourSlash) {
      challenge.outcome = "lost";
      challenge.penalty = ourSlash.amount;
      this.updateReputation(-0.02);
    } else {
      challenge.outcome = "draw";
    }
  }

  // ============ Queries ============

  /**
   * Get active challenges
   */
  getActiveChallenges(): ChallengeStatus[] {
    return Array.from(this.activeChallenges.values()).filter(
      (c) => c.status === "active" || c.status === "pending",
    );
  }

  /**
   * Get challenge history
   */
  getChallengeHistory(): ChallengeStatus[] {
    return Array.from(this.activeChallenges.values());
  }

  /**
   * Get challenge by ID
   */
  getChallenge(challengeId: string): ChallengeStatus | undefined {
    return this.activeChallenges.get(challengeId);
  }
}

// ============ Factory ============

/**
 * Create a challenger endpoint
 */
export function createChallenger(config: EndpointConfig): Challenger {
  return new Challenger(config);
}
