/**
 * Witness Endpoint Implementation
 *
 * Witnesses provide evidence to support or challenge claims:
 * - Submit evidence with provenance
 * - Attest to facts and claims
 * - Respond to evidence requests
 * - Maintain evidence archives
 *
 * Economic model: Quality evidence earns citation rewards.
 */

import type {
  EvidenceMessage,
  EvidencePayload,
  AttestationMessage,
  AttestationPayload,
  EvidenceRequestMessage,
  EvidenceResponseMessage,
  EvidenceResponsePayload,
  EvidenceSummary,
} from "../../core/messaging/types.js";
import type {
  WitnessEndpoint,
  EndpointRole,
  EndpointCapability,
} from "./types.js";
import type { EvidenceType } from "../../core/evidence/types.js";
import type {
  Confidence,
  EvidenceIndependence,
} from "../../core/knowledge/types.js";
import { BaseEndpoint, type EndpointConfig, type SendOptions } from "./base.js";

// ============ Types ============

/**
 * Evidence submission parameters
 */
export interface EvidenceSubmissionParams {
  /** Evidence type */
  evidenceType: EvidenceType;
  /** Content hash of the evidence */
  contentHash: string;
  /** Original source URL or identifier */
  originalSource: string;
  /** Target claim ID (optional) */
  targetClaimId?: string;
  /** Target fact ID (optional) */
  targetFactId?: string;
  /** Direction relative to target */
  direction: "supporting" | "contradicting" | "neutral";
  /** Intrinsic strength (0-1) */
  intrinsicStrength: number;
  /** Independence metrics */
  independence: EvidenceIndependence;
  /** Request archival */
  archiveRequested?: boolean;
  /** Archive locations (if already archived) */
  archiveLocations?: string[];
  /** Oracle metadata (for oracle evidence) */
  oracleMetadata?: {
    oracleId: string;
    dataFeedId?: string;
    timestamp: number;
    signature?: string;
  };
  /** Additional metadata */
  metadata?: Record<string, unknown>;
}

/**
 * Attestation parameters
 */
export interface AttestationParams {
  /** What is being attested */
  targetType: "claim" | "evidence" | "fact";
  /** ID of the target */
  targetId: string;
  /** Type of attestation */
  attestationType: "confirm" | "deny" | "qualify";
  /** Confidence in attestation */
  confidence: Confidence;
  /** Stake for attestation */
  stake: bigint;
  /** Qualification (if attestationType is 'qualify') */
  qualification?: string;
  /** Expertise areas */
  expertise?: string[];
  /** Reasoning */
  reasoning?: string;
}

// ============ Witness Implementation ============

/**
 * Witness endpoint for providing evidence
 */
export class Witness extends BaseEndpoint implements WitnessEndpoint {
  // WitnessEndpoint fields
  evidenceSubmitted = 0;
  evidenceAccepted = 0;
  expertiseDomains?: string[];
  evidenceQualityScore = 0.5;

  // Track submitted evidence
  private readonly submittedEvidence = new Map<string, EvidenceMessage>();

  // Track attestations
  private readonly attestations = new Map<string, AttestationMessage>();

  constructor(config: EndpointConfig & { expertiseDomains?: string[] }) {
    super({
      ...config,
      roles: [
        "witness",
        ...config.roles.filter((r: EndpointRole) => r !== "witness"),
      ],
      capabilities: [
        "evidence_submit",
        "evidence_provide",
        "evidence_attest",
        ...config.capabilities.filter(
          (c: EndpointCapability) =>
            ![
              "evidence_submit",
              "evidence_provide",
              "evidence_attest",
            ].includes(c),
        ),
      ],
    });

    this.expertiseDomains = config.expertiseDomains;

    // Subscribe to evidence requests
    this.eventBus.on("evidence_request" as never, (event) => {
      this.handleEvidenceRequest(
        event as unknown as { payload: EvidenceRequestMessage },
      );
    });
  }

  // ============ Evidence Submission ============

  /**
   * Submit evidence
   */
  async submitEvidence(
    params: EvidenceSubmissionParams,
    options: SendOptions = {},
  ): Promise<EvidenceMessage> {
    // Validate
    if (params.intrinsicStrength < 0 || params.intrinsicStrength > 1) {
      throw new Error("Intrinsic strength must be between 0 and 1");
    }

    // Create payload
    const payload: EvidencePayload = {
      evidenceType: params.evidenceType,
      contentHash: params.contentHash,
      originalSource: params.originalSource,
      targetClaimId: params.targetClaimId,
      targetFactId: params.targetFactId,
      direction: params.direction,
      intrinsicStrength: params.intrinsicStrength,
      independence: params.independence,
      archiveRequested: params.archiveRequested ?? false,
      archiveLocations: params.archiveLocations,
      oracleMetadata: params.oracleMetadata,
      metadata: params.metadata,
    };

    // Build referenced messages
    const referencedMessages = options.referencedMessages ?? [];
    if (params.targetClaimId) {
      referencedMessages.push(params.targetClaimId);
    }
    if (params.targetFactId) {
      referencedMessages.push(params.targetFactId);
    }

    // Create and finalize message
    const messageBase = this.createMessage("evidence", payload, {
      ...options,
      recipient: "broadcast",
      referencedMessages,
    });

    const message = (await this.finalizeMessage(
      messageBase,
    )) as EvidenceMessage;

    // Track
    this.submittedEvidence.set(message.messageId, message);
    this.evidenceSubmitted++;

    // Send
    await this.send(message);

    return message;
  }

  /**
   * Submit supporting evidence for a claim
   */
  async supportClaim(
    claimId: string,
    evidenceType: EvidenceType,
    contentHash: string,
    source: string,
    strength: number,
    options: SendOptions = {},
  ): Promise<EvidenceMessage> {
    return this.submitEvidence(
      {
        evidenceType,
        contentHash,
        originalSource: source,
        targetClaimId: claimId,
        direction: "supporting",
        intrinsicStrength: strength,
        independence: {
          methodIndependence: 0.7,
          sourceIndependence: 0.7,
          temporalIndependence: 0.8,
          aggregateIndependence: 0.7,
        },
      },
      options,
    );
  }

  /**
   * Submit contradicting evidence against a claim
   */
  async contradictClaim(
    claimId: string,
    evidenceType: EvidenceType,
    contentHash: string,
    source: string,
    strength: number,
    options: SendOptions = {},
  ): Promise<EvidenceMessage> {
    return this.submitEvidence(
      {
        evidenceType,
        contentHash,
        originalSource: source,
        targetClaimId: claimId,
        direction: "contradicting",
        intrinsicStrength: strength,
        independence: {
          methodIndependence: 0.7,
          sourceIndependence: 0.7,
          temporalIndependence: 0.8,
          aggregateIndependence: 0.7,
        },
      },
      options,
    );
  }

  // ============ Attestation ============

  /**
   * Attest to a claim, evidence, or fact
   */
  async attest(
    params: AttestationParams,
    options: SendOptions = {},
  ): Promise<AttestationMessage> {
    // Validate
    if (params.confidence < 0 || params.confidence > 1) {
      throw new Error("Confidence must be between 0 and 1");
    }
    if (params.stake <= 0n) {
      throw new Error("Stake must be positive");
    }

    // Create payload
    const payload: AttestationPayload = {
      targetType: params.targetType,
      targetId: params.targetId,
      attestationType: params.attestationType,
      confidence: params.confidence,
      stake: params.stake,
      qualification: params.qualification,
      expertise: params.expertise ?? this.expertiseDomains,
      reasoning: params.reasoning,
    };

    // Create and finalize message
    const messageBase = this.createMessage("attestation", payload, {
      ...options,
      recipient: "broadcast",
      referencedMessages: [
        params.targetId,
        ...(options.referencedMessages ?? []),
      ],
    });

    const message = (await this.finalizeMessage(
      messageBase,
    )) as AttestationMessage;

    // Track
    this.attestations.set(message.messageId, message);

    // Send
    await this.send(message);

    return message;
  }

  /**
   * Confirm a claim or fact
   */
  async confirm(
    targetType: "claim" | "fact",
    targetId: string,
    confidence: Confidence,
    stake: bigint,
    reasoning?: string,
    options: SendOptions = {},
  ): Promise<AttestationMessage> {
    return this.attest(
      {
        targetType,
        targetId,
        attestationType: "confirm",
        confidence,
        stake,
        reasoning,
      },
      options,
    );
  }

  /**
   * Deny a claim or fact
   */
  async deny(
    targetType: "claim" | "fact",
    targetId: string,
    confidence: Confidence,
    stake: bigint,
    reasoning: string,
    options: SendOptions = {},
  ): Promise<AttestationMessage> {
    return this.attest(
      {
        targetType,
        targetId,
        attestationType: "deny",
        confidence,
        stake,
        reasoning,
      },
      options,
    );
  }

  // ============ Evidence Request Handling ============

  /**
   * Handle evidence request
   */
  private handleEvidenceRequest(event: {
    payload: EvidenceRequestMessage;
  }): void {
    const request = event.payload;

    // Check if we should respond based on domain expertise
    if (request.payload.domain && this.expertiseDomains) {
      if (!this.expertiseDomains.includes(request.payload.domain)) {
        return; // Not our domain
      }
    }

    // Find matching evidence
    const matching = this.findMatchingEvidence(request);

    if (matching.length > 0) {
      // Auto-respond with matching evidence
      this.respondToEvidenceRequest(request.messageId, matching).catch(
        (error) => {
          console.error("Failed to respond to evidence request:", error);
        },
      );
    }
  }

  /**
   * Find evidence matching a request
   */
  private findMatchingEvidence(
    request: EvidenceRequestMessage,
  ): EvidenceSummary[] {
    const results: EvidenceSummary[] = [];
    const payload = request.payload;

    for (const evidence of this.submittedEvidence.values()) {
      // Check filters
      if (
        payload.evidenceTypes &&
        !payload.evidenceTypes.includes(evidence.payload.evidenceType)
      ) {
        continue;
      }
      if (
        payload.minStrength &&
        evidence.payload.intrinsicStrength < payload.minStrength
      ) {
        continue;
      }
      if (
        payload.claimId &&
        evidence.payload.targetClaimId !== payload.claimId
      ) {
        continue;
      }
      if (payload.factId && evidence.payload.targetFactId !== payload.factId) {
        continue;
      }

      results.push({
        id: evidence.messageId,
        type: evidence.payload.evidenceType,
        contentHash: evidence.payload.contentHash,
        direction: evidence.payload.direction,
        strength: evidence.payload.intrinsicStrength,
        submittedAt: evidence.timestamp,
        source: evidence.payload.originalSource,
      });

      // Apply limit
      if (payload.limit && results.length >= payload.limit) {
        break;
      }
    }

    return results;
  }

  /**
   * Respond to an evidence request
   */
  async respondToEvidenceRequest(
    requestId: string,
    evidence: EvidenceSummary[],
    options: SendOptions = {},
  ): Promise<EvidenceResponseMessage> {
    // Calculate relevance scores (simple version)
    const relevanceScores = evidence.map((e) => e.strength);

    // Create payload
    const payload: EvidenceResponsePayload = {
      requestId,
      evidence,
      total: evidence.length,
      relevanceScores,
      coverageEstimate: Math.min(1, evidence.length / 10), // Rough estimate
    };

    // Create and finalize message
    const messageBase = this.createMessage("evidence_response", payload, {
      ...options,
      causedBy: requestId,
      referencedMessages: [requestId],
    });

    const message = (await this.finalizeMessage(
      messageBase,
    )) as EvidenceResponseMessage;

    // Send
    await this.send(message);

    return message;
  }

  // ============ Quality Tracking ============

  /**
   * Update evidence quality score based on acceptance
   */
  updateQualityScore(accepted: boolean): void {
    const delta = accepted ? 0.02 : -0.01;
    this.evidenceQualityScore = Math.max(
      0,
      Math.min(1, this.evidenceQualityScore + delta),
    );

    if (accepted) {
      this.evidenceAccepted++;
    }
  }

  // ============ Queries ============

  /**
   * Get submitted evidence
   */
  getSubmittedEvidence(): EvidenceMessage[] {
    return Array.from(this.submittedEvidence.values());
  }

  /**
   * Get attestations
   */
  getAttestations(): AttestationMessage[] {
    return Array.from(this.attestations.values());
  }

  /**
   * Get evidence by ID
   */
  getEvidence(evidenceId: string): EvidenceMessage | undefined {
    return this.submittedEvidence.get(evidenceId);
  }
}

// ============ Factory ============

/**
 * Create a witness endpoint
 */
export function createWitness(
  config: EndpointConfig & { expertiseDomains?: string[] },
): Witness {
  return new Witness(config);
}
