/**
 * Claimant Endpoint Implementation
 *
 * Claimants submit truth claims to the network:
 * - Create and submit claims with stake
 * - Attach supporting evidence
 * - Respond to challenges
 * - Withdraw claims (before verification)
 *
 * Economic model: Stake is slashed if claim is proven false.
 */

import type {
  ClaimMessage,
  ClaimPayload,
  ChallengeMessage,
  ResolutionMessage,
} from "../../core/messaging/types.js";
import type {
  ClaimantEndpoint,
  EndpointRole,
  EndpointCapability,
} from "./types.js";
import type {
  ClaimType,
  EpistemicCategory,
  InferenceType,
  Confidence,
} from "../../core/knowledge/types.js";
import { BaseEndpoint, type EndpointConfig, type SendOptions } from "./base.js";

// ============ Types ============

/**
 * Claim submission parameters
 */
export interface ClaimSubmissionParams {
  /** The claim text */
  claim: string;
  /** Type of claim */
  claimType: ClaimType;
  /** Epistemic category */
  epistemicCategory: EpistemicCategory;
  /** Proposed confidence level */
  proposedConfidence: Confidence;
  /** Stake amount (in LGM base units) */
  stake: bigint;
  /** Knowledge domain */
  domain: string;
  /** Pre-attached evidence IDs */
  evidenceIds?: string[];
  /** External references */
  externalReferences?: string[];
  /** Derivation info (if derived from other facts) */
  derivation?: {
    factIds: string[];
    inferenceType: InferenceType;
    logicChain: string;
  };
}

/**
 * Challenge response parameters
 */
export interface ChallengeResponseParams {
  /** Challenge message ID */
  challengeId: string;
  /** Response type */
  responseType: "defend" | "withdraw" | "modify";
  /** Defense reasoning */
  reasoning?: string;
  /** Additional evidence IDs */
  additionalEvidenceIds?: string[];
  /** Modified claim (if responseType is 'modify') */
  modifiedClaim?: string;
  /** Modified confidence (if responseType is 'modify') */
  modifiedConfidence?: Confidence;
}

// ============ Claimant Implementation ============

/**
 * Claimant endpoint for submitting truth claims
 */
export class Claimant extends BaseEndpoint implements ClaimantEndpoint {
  // ClaimantEndpoint fields
  claimsSubmitted = 0;
  claimsAccepted = 0;
  claimsRejected = 0;

  // Track active claims
  private readonly activeClaims = new Map<string, ClaimMessage>();

  constructor(config: EndpointConfig) {
    super({
      ...config,
      roles: [
        "claimant",
        ...config.roles.filter((r: EndpointRole) => r !== "claimant"),
      ],
      capabilities: [
        "claim_submit",
        "claim_withdraw",
        "evidence_submit",
        ...config.capabilities.filter(
          (c: EndpointCapability) =>
            !["claim_submit", "claim_withdraw", "evidence_submit"].includes(c),
        ),
      ],
    });

    // Subscribe to resolutions for our claims
    this.eventBus.on("resolution" as never, (event) => {
      this.handleResolution(event as unknown as { payload: ResolutionMessage });
    });

    // Subscribe to challenges against our claims
    this.eventBus.on("challenge" as never, (event) => {
      this.handleChallenge(event as unknown as { payload: ChallengeMessage });
    });
  }

  /**
   * Success rate (accepted / submitted)
   */
  get claimSuccessRate(): number {
    if (this.claimsSubmitted === 0) return 0;
    return this.claimsAccepted / this.claimsSubmitted;
  }

  // ============ Claim Submission ============

  /**
   * Submit a new claim
   */
  async submitClaim(
    params: ClaimSubmissionParams,
    options: SendOptions = {},
  ): Promise<ClaimMessage> {
    // Validate stake
    if (params.stake <= 0n) {
      throw new Error("Stake must be positive");
    }
    if (params.stake > this.stakedAmount) {
      throw new Error("Insufficient staked balance");
    }

    // Validate confidence
    if (params.proposedConfidence < 0 || params.proposedConfidence > 1) {
      throw new Error("Confidence must be between 0 and 1");
    }

    // Create payload
    const payload: ClaimPayload = {
      claim: params.claim,
      claimType: params.claimType,
      epistemicCategory: params.epistemicCategory,
      proposedConfidence: params.proposedConfidence,
      stake: params.stake,
      derivation: params.derivation,
      evidenceIds: params.evidenceIds,
      externalReferences: params.externalReferences,
    };

    // Create and finalize message
    const messageBase = this.createMessage("claim", payload, {
      ...options,
      domain: params.domain,
      recipient: "broadcast",
    });

    const message = (await this.finalizeMessage(messageBase)) as ClaimMessage;

    // Track the claim
    this.activeClaims.set(message.messageId, message);
    this.claimsSubmitted++;

    // Send
    await this.send(message);

    return message;
  }

  /**
   * Withdraw a claim (before verification completes)
   */
  async withdrawClaim(
    claimId: string,
    reason: string,
    options: SendOptions = {},
  ): Promise<void> {
    const claim = this.activeClaims.get(claimId);
    if (!claim) {
      throw new Error("Claim not found or not owned by this endpoint");
    }

    // Create withdrawal message (using resolution type)
    const payload = {
      targetType: "claim" as const,
      targetId: claimId,
      outcome: "withdrawn" as const,
      finalConfidence: 0 as Confidence,
      resolvedAtBlock: 0n, // Will be set by network
      rewards: [],
      slashes: [],
      withdrawalReason: reason,
    };

    const messageBase = this.createMessage("resolution", payload, {
      ...options,
      causedBy: claimId,
    });

    const message = await this.finalizeMessage(messageBase);
    await this.send(message);

    // Remove from active claims
    this.activeClaims.delete(claimId);
  }

  /**
   * Respond to a challenge against one of our claims
   */
  async respondToChallenge(
    params: ChallengeResponseParams,
    options: SendOptions = {},
  ): Promise<void> {
    // Create response based on type
    if (params.responseType === "withdraw") {
      await this.withdrawClaim(
        params.challengeId,
        params.reasoning ?? "Withdrawn in response to challenge",
      );
      return;
    }

    // For defend/modify, submit additional evidence or modified claim
    const payload = {
      targetType: "challenge" as const,
      targetId: params.challengeId,
      responseType: params.responseType,
      reasoning: params.reasoning,
      additionalEvidenceIds: params.additionalEvidenceIds,
      modifiedClaim: params.modifiedClaim,
      modifiedConfidence: params.modifiedConfidence,
    };

    const messageBase = this.createMessage("attestation", payload as never, {
      ...options,
      causedBy: params.challengeId,
    });

    const message = await this.finalizeMessage(messageBase);
    await this.send(message);
  }

  // ============ Evidence Attachment ============

  /**
   * Attach additional evidence to an existing claim
   */
  async attachEvidence(
    claimId: string,
    evidenceIds: string[],
    options: SendOptions = {},
  ): Promise<void> {
    const claim = this.activeClaims.get(claimId);
    if (!claim) {
      throw new Error("Claim not found or not owned by this endpoint");
    }

    // Create evidence attachment message
    for (const evidenceId of evidenceIds) {
      const payload = {
        evidenceType: "attestation" as const,
        contentHash: evidenceId,
        originalSource: `claim:${claimId}`,
        targetClaimId: claimId,
        direction: "supporting" as const,
        intrinsicStrength: 0.5,
        independence: {
          methodIndependence: 0.5,
          sourceIndependence: 0.5,
          temporalIndependence: 0.5,
          aggregateIndependence: 0.5,
        },
        archiveRequested: false,
      };

      const messageBase = this.createMessage("evidence", payload, {
        ...options,
        causedBy: claimId,
        referencedMessages: [claimId],
      });

      const message = await this.finalizeMessage(messageBase);
      await this.send(message);
    }
  }

  // ============ Event Handlers ============

  /**
   * Handle resolution of our claims
   */
  private handleResolution(event: { payload: ResolutionMessage }): void {
    const resolution = event.payload;

    if (resolution.payload.targetType !== "claim") return;

    const claim = this.activeClaims.get(resolution.payload.targetId);
    if (!claim) return;

    // Update statistics
    if (resolution.payload.outcome === "accepted") {
      this.claimsAccepted++;
      this.updateReputation(0.01); // Small reputation boost
    } else if (resolution.payload.outcome === "rejected") {
      this.claimsRejected++;
      this.updateReputation(-0.02); // Reputation penalty
    }

    // Remove from active claims
    this.activeClaims.delete(resolution.payload.targetId);
  }

  /**
   * Handle challenges against our claims
   */
  private handleChallenge(event: { payload: ChallengeMessage }): void {
    const challenge = event.payload;

    // Check if this challenge targets one of our claims
    if (challenge.payload.targetType !== "claim") return;

    const claim = this.activeClaims.get(challenge.payload.targetId);
    if (!claim) return;

    // Emit event for application layer to handle
    this.eventBus.emit({
      type: "claim_challenged" as never,
      timestamp: Date.now(),
      source: "claimant",
      payload: {
        claimId: challenge.payload.targetId,
        challengeId: challenge.messageId,
        challenger: challenge.sender,
        reason: challenge.payload.reason,
        challengeType: challenge.payload.challengeType,
      } as never,
    });
  }

  // ============ Queries ============

  /**
   * Get active claims
   */
  getActiveClaims(): ClaimMessage[] {
    return Array.from(this.activeClaims.values());
  }

  /**
   * Get claim by ID
   */
  getClaim(claimId: string): ClaimMessage | undefined {
    return this.activeClaims.get(claimId);
  }
}

// ============ Factory ============

/**
 * Create a claimant endpoint
 */
export function createClaimant(config: EndpointConfig): Claimant {
  return new Claimant(config);
}
