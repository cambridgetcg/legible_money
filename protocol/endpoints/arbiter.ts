/**
 * Arbiter Endpoint Implementation
 *
 * Arbiters vote on disputes to resolve them:
 * - VRF-selected from eligible validators
 * - Review evidence from both parties
 * - Cast votes (challenger/defender/abstain)
 * - Request additional evidence if needed
 * - Earn rewards for correct arbitration
 *
 * Economic model: Voting accuracy affects reputation and future selection.
 */

import type {
  EvidenceRequestMessage,
  EvidenceRequestPayload,
  AttestationMessage,
  AttestationPayload,
  ResolutionMessage,
} from "../../core/messaging/types.js";
import type {
  ArbiterEndpoint,
  EndpointRole,
  EndpointCapability,
} from "./types.js";
import type { Confidence } from "../../core/knowledge/types.js";
import { BaseEndpoint, type EndpointConfig, type SendOptions } from "./base.js";

// ============ Types ============

/**
 * Arbitration vote options
 */
export type ArbiterVote = "challenger" | "defender" | "abstain";

/**
 * Vote parameters
 */
export interface VoteParams {
  /** Dispute ID */
  disputeId: string;
  /** The vote */
  vote: ArbiterVote;
  /** Confidence in the vote (0-1) */
  confidence: Confidence;
  /** Justification hash (optional, for accountability) */
  justificationHash?: string;
  /** Reasoning (optional) */
  reasoning?: string;
  /** Evidence IDs reviewed */
  evidenceReviewed: string[];
}

/**
 * Evidence request parameters
 */
export interface EvidenceRequestParams {
  /** Dispute ID this request is for */
  disputeId: string;
  /** Type of request */
  requestType:
    | "by_topic"
    | "by_claim"
    | "by_fact"
    | "contradiction"
    | "corroboration";
  /** Topic to search for */
  topic?: string;
  /** Claim ID (if relevant) */
  claimId?: string;
  /** Fact ID (if relevant) */
  factId?: string;
  /** Knowledge domain */
  domain?: string;
  /** Minimum evidence strength */
  minStrength?: number;
  /** Optional bounty for quality responses */
  bounty?: bigint;
  /** Deadline in blocks */
  deadlineBlocks?: bigint;
}

/**
 * Arbitration record
 */
interface ArbitrationRecord {
  disputeId: string;
  vote: ArbiterVote;
  confidence: Confidence;
  votedAt: number;
  finalOutcome?: "challenger_wins" | "defender_wins" | "draw";
  correct?: boolean;
  reward?: bigint;
  penalty?: bigint;
}

// ============ Arbiter Implementation ============

/**
 * Arbiter endpoint for dispute resolution
 */
export class Arbiter extends BaseEndpoint implements ArbiterEndpoint {
  // ArbiterEndpoint fields
  disputesArbitrated = 0;
  correctArbitrations = 0;
  minArbiterStake: bigint;

  // Arbitration history
  private readonly arbitrationHistory = new Map<string, ArbitrationRecord>();

  // Active disputes we're arbitrating
  private readonly activeDisputes = new Set<string>();

  constructor(config: EndpointConfig & { minArbiterStake?: bigint }) {
    super({
      ...config,
      roles: [
        "arbiter",
        ...config.roles.filter((r: EndpointRole) => r !== "arbiter"),
      ],
      capabilities: [
        "dispute_arbitrate",
        "dispute_vote",
        "evidence_submit",
        ...config.capabilities.filter(
          (c: EndpointCapability) =>
            !["dispute_arbitrate", "dispute_vote", "evidence_submit"].includes(
              c,
            ),
        ),
      ],
    });

    this.minArbiterStake = config.minArbiterStake ?? 1000n * 10n ** 18n; // 1000 LGM default

    // Subscribe to dispute resolutions
    this.eventBus.on("dispute_resolved" as never, (event) => {
      this.handleDisputeResolved(
        event as unknown as { payload: ResolutionMessage },
      );
    });
  }

  /**
   * Arbitration accuracy (correct / arbitrated)
   */
  get arbitrationAccuracy(): number {
    if (this.disputesArbitrated === 0) return 0;
    return this.correctArbitrations / this.disputesArbitrated;
  }

  // ============ Arbitration ============

  /**
   * Accept assignment to arbitrate a dispute
   */
  async acceptArbitration(
    disputeId: string,
    vrfOutput: string,
    _vrfProof: string,
    options: SendOptions = {},
  ): Promise<void> {
    // Validate stake
    if (this.stakedAmount < this.minArbiterStake) {
      throw new Error(
        `Insufficient stake for arbitration. Required: ${this.minArbiterStake}, Have: ${this.stakedAmount}`,
      );
    }

    // Check not already arbitrating
    if (this.activeDisputes.has(disputeId)) {
      throw new Error("Already arbitrating this dispute");
    }

    // Create acceptance attestation
    const payload: AttestationPayload = {
      targetType: "claim", // Disputes are about claims/facts
      targetId: disputeId,
      attestationType: "confirm",
      confidence: 1.0,
      stake: this.minArbiterStake / 10n, // Lock 10% of min stake
      reasoning: `Accepted arbitration assignment. VRF: ${vrfOutput.slice(0, 16)}...`,
    };

    const messageBase = this.createMessage("attestation", payload, {
      ...options,
      recipient: "broadcast",
      referencedMessages: [disputeId],
    });

    const message = await this.finalizeMessage(messageBase);

    // Track active dispute
    this.activeDisputes.add(disputeId);

    await this.send(message);
  }

  /**
   * Cast a vote on a dispute
   */
  async castVote(
    params: VoteParams,
    options: SendOptions = {},
  ): Promise<AttestationMessage> {
    // Validate
    if (!this.activeDisputes.has(params.disputeId)) {
      throw new Error("Not assigned to arbitrate this dispute");
    }
    if (params.confidence < 0 || params.confidence > 1) {
      throw new Error("Confidence must be between 0 and 1");
    }

    // Create vote attestation
    const payload: AttestationPayload = {
      targetType: "claim",
      targetId: params.disputeId,
      attestationType: params.vote === "abstain" ? "qualify" : "confirm",
      qualification:
        params.vote === "abstain"
          ? "ABSTAIN: Insufficient evidence"
          : undefined,
      confidence: params.confidence,
      stake: this.minArbiterStake / 10n,
      reasoning: params.reasoning,
      expertise: [`arbitration:${params.vote}`],
    };

    // Add vote metadata
    const voteMetadata = {
      ...payload,
      arbiterVote: params.vote,
      justificationHash: params.justificationHash,
      evidenceReviewed: params.evidenceReviewed,
    };

    const messageBase = this.createMessage(
      "attestation",
      voteMetadata as never,
      {
        ...options,
        recipient: "broadcast",
        referencedMessages: [params.disputeId, ...params.evidenceReviewed],
      },
    );

    const message = (await this.finalizeMessage(
      messageBase,
    )) as AttestationMessage;

    // Record arbitration
    this.arbitrationHistory.set(params.disputeId, {
      disputeId: params.disputeId,
      vote: params.vote,
      confidence: params.confidence,
      votedAt: Date.now(),
    });
    this.disputesArbitrated++;

    // Remove from active (vote cast)
    this.activeDisputes.delete(params.disputeId);

    await this.send(message);

    return message;
  }

  /**
   * Vote for challenger
   */
  async voteChallenger(
    disputeId: string,
    confidence: Confidence,
    evidenceReviewed: string[],
    reasoning?: string,
    options: SendOptions = {},
  ): Promise<AttestationMessage> {
    return this.castVote(
      {
        disputeId,
        vote: "challenger",
        confidence,
        evidenceReviewed,
        reasoning,
      },
      options,
    );
  }

  /**
   * Vote for defender
   */
  async voteDefender(
    disputeId: string,
    confidence: Confidence,
    evidenceReviewed: string[],
    reasoning?: string,
    options: SendOptions = {},
  ): Promise<AttestationMessage> {
    return this.castVote(
      {
        disputeId,
        vote: "defender",
        confidence,
        evidenceReviewed,
        reasoning,
      },
      options,
    );
  }

  /**
   * Abstain from voting
   */
  async abstain(
    disputeId: string,
    evidenceReviewed: string[],
    reasoning: string,
    options: SendOptions = {},
  ): Promise<AttestationMessage> {
    return this.castVote(
      {
        disputeId,
        vote: "abstain",
        confidence: 0.5,
        evidenceReviewed,
        reasoning,
      },
      options,
    );
  }

  // ============ Evidence Requests ============

  /**
   * Request additional evidence for a dispute
   */
  async requestEvidence(
    params: EvidenceRequestParams,
    options: SendOptions = {},
  ): Promise<EvidenceRequestMessage> {
    // Create payload
    const payload: EvidenceRequestPayload = {
      requestType: params.requestType,
      topic: params.topic,
      domain: params.domain,
      claimId: params.claimId,
      factId: params.factId,
      minStrength: params.minStrength ?? 0.5,
      bounty: params.bounty,
      deadline: params.deadlineBlocks,
    };

    const messageBase = this.createMessage("evidence_request", payload, {
      ...options,
      recipient: "broadcast",
      causedBy: params.disputeId,
      referencedMessages: [params.disputeId],
    });

    const message = (await this.finalizeMessage(
      messageBase,
    )) as EvidenceRequestMessage;

    await this.send(message);

    return message;
  }

  /**
   * Request corroborating evidence
   */
  async requestCorroboration(
    disputeId: string,
    topic: string,
    domain: string,
    bounty?: bigint,
    options: SendOptions = {},
  ): Promise<EvidenceRequestMessage> {
    return this.requestEvidence(
      {
        disputeId,
        requestType: "corroboration",
        topic,
        domain,
        minStrength: 0.6,
        bounty,
      },
      options,
    );
  }

  /**
   * Request contradicting evidence
   */
  async requestContradiction(
    disputeId: string,
    topic: string,
    domain: string,
    bounty?: bigint,
    options: SendOptions = {},
  ): Promise<EvidenceRequestMessage> {
    return this.requestEvidence(
      {
        disputeId,
        requestType: "contradiction",
        topic,
        domain,
        minStrength: 0.6,
        bounty,
      },
      options,
    );
  }

  // ============ Event Handlers ============

  /**
   * Handle dispute resolution to track accuracy
   */
  private handleDisputeResolved(event: { payload: ResolutionMessage }): void {
    const resolution = event.payload;

    // Find our arbitration record
    const record = this.arbitrationHistory.get(resolution.payload.targetId);
    if (!record) return;

    // Determine final outcome
    let finalOutcome: "challenger_wins" | "defender_wins" | "draw";
    if (resolution.payload.outcome === "accepted") {
      // Claim/fact was upheld = defender wins
      finalOutcome = "defender_wins";
    } else if (resolution.payload.outcome === "rejected") {
      // Claim/fact was rejected = challenger wins
      finalOutcome = "challenger_wins";
    } else {
      finalOutcome = "draw";
    }

    record.finalOutcome = finalOutcome;

    // Check if our vote was correct
    const correct =
      (record.vote === "challenger" && finalOutcome === "challenger_wins") ||
      (record.vote === "defender" && finalOutcome === "defender_wins") ||
      record.vote === "abstain"; // Abstain is neutral

    record.correct = correct;

    // Check for reward/penalty
    const ourReward = resolution.payload.rewards.find(
      (r) => r.recipient === this.did,
    );
    const ourSlash = resolution.payload.slashes.find(
      (s) => s.target === this.did,
    );

    if (ourReward) {
      record.reward = ourReward.amount;
    }
    if (ourSlash) {
      record.penalty = ourSlash.amount;
    }

    // Update accuracy
    if (correct && record.vote !== "abstain") {
      this.correctArbitrations++;
      this.updateReputation(0.02);
    } else if (!correct && record.vote !== "abstain") {
      this.updateReputation(-0.03);
    }
  }

  // ============ Queries ============

  /**
   * Get active disputes
   */
  getActiveDisputes(): string[] {
    return Array.from(this.activeDisputes);
  }

  /**
   * Get arbitration history
   */
  getArbitrationHistory(): ArbitrationRecord[] {
    return Array.from(this.arbitrationHistory.values());
  }

  /**
   * Get arbitration record for a dispute
   */
  getArbitrationRecord(disputeId: string): ArbitrationRecord | undefined {
    return this.arbitrationHistory.get(disputeId);
  }

  /**
   * Check if eligible to arbitrate
   */
  isEligible(): boolean {
    return this.stakedAmount >= this.minArbiterStake;
  }
}

// ============ Factory ============

/**
 * Create an arbiter endpoint
 */
export function createArbiter(
  config: EndpointConfig & { minArbiterStake?: bigint },
): Arbiter {
  return new Arbiter(config);
}
