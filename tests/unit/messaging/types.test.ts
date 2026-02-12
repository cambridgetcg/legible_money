/**
 * Unit Tests for Truth Message Types
 *
 * Tests message creation, type guards, and validation.
 */

import { describe, it, expect } from "vitest";
import type { DID } from "../../../core/agents/identity/did.js";
import type { Signature } from "../../../core/crypto/keys.js";
import {
  isClaimMessage,
  isEvidenceMessage,
  isAttestationMessage,
  isChallengeMessage,
  isVerificationCommitMessage,
  isVerificationRevealMessage,
  isResolutionMessage,
  isEvidenceRequestMessage,
  isEvidenceResponseMessage,
  isAcknowledgmentMessage,
  validateTruthMessage,
  type ClaimMessage,
  type EvidenceMessage,
  type AttestationMessage,
  type ChallengeMessage,
  type VerificationCommitMessage,
  type VerificationRevealMessage,
  type ResolutionMessage,
  type EvidenceRequestMessage,
  type TruthMessage,
} from "../../../core/messaging/types.js";

// ============ Test Fixtures ============

const TEST_DID = "did:lgm:test-sender" as DID;
const TEST_RECIPIENT = "did:lgm:test-recipient" as DID;
const TEST_SIGNATURE = "test-signature-hex" as Signature;

function createBaseMessage(
  messageType: string,
  payload: unknown,
): TruthMessage {
  return {
    messageType: messageType as TruthMessage["messageType"],
    messageId: `msg_${messageType}_test123`,
    version: "1.0.0",
    sender: TEST_DID,
    recipient: "broadcast",
    payload: payload as TruthMessage["payload"],
    contentHash: "hash_abc123",
    signature: TEST_SIGNATURE,
    referencedMessages: [],
    networkFeeUnit: 1n,
    networkFeeLimit: 100000n,
    timestamp: Date.now(),
  };
}

// ============ ClaimMessage Tests ============

describe("ClaimMessage", () => {
  const validClaimPayload = {
    claim: "The Earth orbits the Sun",
    claimType: "empirical" as const,
    epistemicCategory: "empirical" as const,
    proposedConfidence: 0.99,
    stake: 100n * 10n ** 15n,
    evidenceIds: ["evidence-1", "evidence-2"],
  };

  it("should validate a valid claim message", () => {
    const message = createBaseMessage(
      "claim",
      validClaimPayload,
    ) as ClaimMessage;

    expect(isClaimMessage(message)).toBe(true);
  });

  it("should reject non-claim message types", () => {
    const message = createBaseMessage("evidence", validClaimPayload);

    expect(isClaimMessage(message)).toBe(false);
  });

  it("should handle claim with derivation", () => {
    const payloadWithDerivation = {
      ...validClaimPayload,
      derivation: {
        factIds: ["fact-1", "fact-2"],
        inferenceType: "deductive" as const,
        logicChain: "If A then B; A; therefore B",
      },
    };

    const message = createBaseMessage(
      "claim",
      payloadWithDerivation,
    ) as ClaimMessage;

    expect(isClaimMessage(message)).toBe(true);
    expect(message.payload.derivation?.factIds).toHaveLength(2);
  });
});

// ============ EvidenceMessage Tests ============

describe("EvidenceMessage", () => {
  const validEvidencePayload = {
    evidenceType: "peer_reviewed" as const,
    contentHash: "sha256_evidence_hash",
    originalSource: "https://example.com/paper.pdf",
    targetClaimId: "claim-123",
    direction: "supporting" as const,
    intrinsicStrength: 0.85,
    independence: {
      sourceIndependence: 0.9,
      methodIndependence: 0.8,
      temporalIndependence: 0.7,
      aggregateIndependence: 0.8,
    },
    archiveRequested: true,
  };

  it("should validate a valid evidence message", () => {
    const message = createBaseMessage(
      "evidence",
      validEvidencePayload,
    ) as EvidenceMessage;

    expect(isEvidenceMessage(message)).toBe(true);
  });

  it("should reject non-evidence message types", () => {
    const message = createBaseMessage("claim", validEvidencePayload);

    expect(isEvidenceMessage(message)).toBe(false);
  });

  it("should handle contradicting evidence", () => {
    const contradictingPayload = {
      ...validEvidencePayload,
      direction: "contradicting" as const,
    };

    const message = createBaseMessage(
      "evidence",
      contradictingPayload,
    ) as EvidenceMessage;

    expect(isEvidenceMessage(message)).toBe(true);
    expect(message.payload.direction).toBe("contradicting");
  });

  it("should handle oracle evidence with metadata", () => {
    const oraclePayload = {
      ...validEvidencePayload,
      evidenceType: "oracle" as const,
      oracleMetadata: {
        oracleId: "oracle-chainlink",
        dataFeedId: "ETH/USD",
        timestamp: Date.now(),
        signature: "oracle-sig-hex",
      },
    };

    const message = createBaseMessage(
      "evidence",
      oraclePayload,
    ) as EvidenceMessage;

    expect(isEvidenceMessage(message)).toBe(true);
    expect(message.payload.oracleMetadata?.oracleId).toBe("oracle-chainlink");
  });
});

// ============ AttestationMessage Tests ============

describe("AttestationMessage", () => {
  const validAttestationPayload = {
    targetType: "claim" as const,
    targetId: "claim-123",
    attestationType: "confirm" as const,
    confidence: 0.9,
    stake: 50n * 10n ** 15n,
    expertise: ["physics", "astronomy"],
    reasoning: "Verified through independent observation",
  };

  it("should validate a valid attestation message", () => {
    const message = createBaseMessage(
      "attestation",
      validAttestationPayload,
    ) as AttestationMessage;

    expect(isAttestationMessage(message)).toBe(true);
  });

  it("should handle qualification attestation", () => {
    const qualificationPayload = {
      ...validAttestationPayload,
      attestationType: "qualify" as const,
      qualification: "True only under standard atmospheric conditions",
    };

    const message = createBaseMessage(
      "attestation",
      qualificationPayload,
    ) as AttestationMessage;

    expect(isAttestationMessage(message)).toBe(true);
    expect(message.payload.attestationType).toBe("qualify");
  });

  it("should handle deny attestation", () => {
    const denyPayload = {
      ...validAttestationPayload,
      attestationType: "deny" as const,
      reasoning: "Contradicts established physics",
    };

    const message = createBaseMessage(
      "attestation",
      denyPayload,
    ) as AttestationMessage;

    expect(isAttestationMessage(message)).toBe(true);
    expect(message.payload.attestationType).toBe("deny");
  });
});

// ============ ChallengeMessage Tests ============

describe("ChallengeMessage", () => {
  const validChallengePayload = {
    targetType: "claim" as const,
    targetId: "claim-123",
    reason: "Contradicts established scientific consensus",
    challengeType: "factual_error" as const,
    counterEvidenceIds: ["counter-evidence-1", "counter-evidence-2"],
    stake: 200n * 10n ** 15n,
    requestedOutcome: "retraction" as const,
  };

  it("should validate a valid challenge message", () => {
    const message = createBaseMessage(
      "challenge",
      validChallengePayload,
    ) as ChallengeMessage;

    expect(isChallengeMessage(message)).toBe(true);
  });

  it("should handle correction request", () => {
    const correctionPayload = {
      ...validChallengePayload,
      requestedOutcome: "correction" as const,
      proposedCorrection: "The corrected statement should be...",
    };

    const message = createBaseMessage(
      "challenge",
      correctionPayload,
    ) as ChallengeMessage;

    expect(isChallengeMessage(message)).toBe(true);
    expect(message.payload.requestedOutcome).toBe("correction");
  });

  it("should handle confidence reduction request", () => {
    const confidencePayload = {
      ...validChallengePayload,
      requestedOutcome: "confidence_reduction" as const,
    };

    const message = createBaseMessage(
      "challenge",
      confidencePayload,
    ) as ChallengeMessage;

    expect(isChallengeMessage(message)).toBe(true);
    expect(message.payload.requestedOutcome).toBe("confidence_reduction");
  });
});

// ============ VerificationCommitMessage Tests ============

describe("VerificationCommitMessage", () => {
  const validCommitPayload = {
    roundId: "round-abc123",
    claimId: "claim-123",
    commitment: "commitment_hash_xyz",
    vrfOutput: "vrf_output_hex",
    vrfProof: "vrf_proof_hex",
  };

  it("should validate a valid verification commit message", () => {
    const message = createBaseMessage(
      "verification_commit",
      validCommitPayload,
    ) as VerificationCommitMessage;

    expect(isVerificationCommitMessage(message)).toBe(true);
  });

  it("should reject non-commit message types", () => {
    const message = createBaseMessage(
      "verification_reveal",
      validCommitPayload,
    );

    expect(isVerificationCommitMessage(message)).toBe(false);
  });
});

// ============ VerificationRevealMessage Tests ============

describe("VerificationRevealMessage", () => {
  const validRevealPayload = {
    roundId: "round-abc123",
    claimId: "claim-123",
    verdict: "confirmed" as const,
    confidence: 0.95,
    nonce: "nonce_xyz",
    referencesChecked: ["ref-1", "ref-2"],
    reasoning: "Evidence strongly supports the claim",
    methodologyUsed: "cross-reference validation",
  };

  it("should validate a valid verification reveal message", () => {
    const message = createBaseMessage(
      "verification_reveal",
      validRevealPayload,
    ) as VerificationRevealMessage;

    expect(isVerificationRevealMessage(message)).toBe(true);
  });

  it("should handle rejected verdict", () => {
    const rejectPayload = {
      ...validRevealPayload,
      verdict: "rejected" as const,
      reasoning: "Counter-evidence is stronger",
    };

    const message = createBaseMessage(
      "verification_reveal",
      rejectPayload,
    ) as VerificationRevealMessage;

    expect(isVerificationRevealMessage(message)).toBe(true);
    expect(message.payload.verdict).toBe("rejected");
  });

  it("should handle uncertain verdict", () => {
    const uncertainPayload = {
      ...validRevealPayload,
      verdict: "uncertain" as const,
      confidence: 0.5,
      reasoning: "Insufficient evidence to determine",
    };

    const message = createBaseMessage(
      "verification_reveal",
      uncertainPayload,
    ) as VerificationRevealMessage;

    expect(isVerificationRevealMessage(message)).toBe(true);
    expect(message.payload.verdict).toBe("uncertain");
  });
});

// ============ ResolutionMessage Tests ============

describe("ResolutionMessage", () => {
  const validResolutionPayload = {
    targetType: "claim" as const,
    targetId: "claim-123",
    outcome: "accepted" as const,
    finalConfidence: 0.92,
    resolvedAtBlock: 12345n,
    rewards: [
      {
        recipient: TEST_DID,
        amount: 10n * 10n ** 15n,
        reason: "Correct verification",
      },
    ],
    slashes: [],
    participationSummary: {
      totalValidators: 5,
      confirmVotes: 4,
      rejectVotes: 1,
      uncertainVotes: 0,
      abstentions: 0,
    },
  };

  it("should validate a valid resolution message", () => {
    const message = createBaseMessage(
      "resolution",
      validResolutionPayload,
    ) as ResolutionMessage;

    expect(isResolutionMessage(message)).toBe(true);
  });

  it("should handle rejection resolution", () => {
    const rejectPayload = {
      ...validResolutionPayload,
      outcome: "rejected" as const,
      finalConfidence: 0,
      slashes: [
        { target: TEST_DID, amount: 50n * 10n ** 15n, reason: "False claim" },
      ],
    };

    const message = createBaseMessage(
      "resolution",
      rejectPayload,
    ) as ResolutionMessage;

    expect(isResolutionMessage(message)).toBe(true);
    expect(message.payload.outcome).toBe("rejected");
  });
});

// ============ EvidenceRequestMessage Tests ============

describe("EvidenceRequestMessage", () => {
  const validRequestPayload = {
    requestType: "by_topic" as const,
    topic: "climate change evidence",
    domain: "environmental_science",
    minStrength: 0.7,
    bounty: 50n * 10n ** 15n,
    deadline: 1000n,
    limit: 10,
  };

  it("should validate a valid evidence request message", () => {
    const message = createBaseMessage(
      "evidence_request",
      validRequestPayload,
    ) as EvidenceRequestMessage;

    expect(isEvidenceRequestMessage(message)).toBe(true);
  });

  it("should handle request by claim", () => {
    const byClaimPayload = {
      ...validRequestPayload,
      requestType: "by_claim" as const,
      claimId: "claim-456",
    };

    const message = createBaseMessage(
      "evidence_request",
      byClaimPayload,
    ) as EvidenceRequestMessage;

    expect(isEvidenceRequestMessage(message)).toBe(true);
    expect(message.payload.requestType).toBe("by_claim");
  });

  it("should handle corroboration request", () => {
    const corroborationPayload = {
      ...validRequestPayload,
      requestType: "corroboration" as const,
      factId: "fact-789",
    };

    const message = createBaseMessage(
      "evidence_request",
      corroborationPayload,
    ) as EvidenceRequestMessage;

    expect(isEvidenceRequestMessage(message)).toBe(true);
    expect(message.payload.requestType).toBe("corroboration");
  });
});

// ============ validateTruthMessage Tests ============

describe("validateTruthMessage", () => {
  it("should accept a valid claim message", () => {
    const message = createBaseMessage("claim", {
      claim: "Test claim",
      claimType: "empirical",
      epistemicCategory: "empirical",
      proposedConfidence: 0.9,
      stake: 100n * 10n ** 15n,
    });

    const result = validateTruthMessage(message);

    expect(result.valid).toBe(true);
    expect(result.errors).toHaveLength(0);
  });

  it("should reject message without messageId", () => {
    const message = createBaseMessage("claim", {
      claim: "Test claim",
      claimType: "empirical",
      epistemicCategory: "empirical",
      proposedConfidence: 0.9,
      stake: 100n * 10n ** 15n,
    });
    (message as { messageId: string }).messageId = "";

    const result = validateTruthMessage(message);

    expect(result.valid).toBe(false);
    expect(result.errors.some((e) => e.includes("messageId"))).toBe(true);
  });

  it("should reject message without sender", () => {
    const message = createBaseMessage("claim", {
      claim: "Test claim",
      claimType: "empirical",
      epistemicCategory: "empirical",
      proposedConfidence: 0.9,
      stake: 100n * 10n ** 15n,
    });
    (message as { sender: string }).sender = "";

    const result = validateTruthMessage(message);

    expect(result.valid).toBe(false);
    expect(result.errors.some((e) => e.includes("sender"))).toBe(true);
  });

  it("should reject message with invalid version", () => {
    const message = createBaseMessage("claim", {
      claim: "Test claim",
      claimType: "empirical",
      epistemicCategory: "empirical",
      proposedConfidence: 0.9,
      stake: 100n * 10n ** 15n,
    });
    (message as { version: string }).version = "2.0.0" as "1.0.0";

    const result = validateTruthMessage(message);

    expect(result.valid).toBe(false);
    expect(result.errors.some((e) => e.includes("version"))).toBe(true);
  });

  it("should reject message with invalid messageType", () => {
    const message = createBaseMessage("invalid_type", {
      claim: "Test claim",
    });

    const result = validateTruthMessage(message);

    expect(result.valid).toBe(false);
    expect(result.errors.some((e) => e.includes("messageType"))).toBe(true);
  });
});
