/**
 * Integration Tests for Truth Controller
 *
 * Tests the full lifecycle of truth verification:
 *   Claim → Evidence → Verification → Resolution
 *           ↓             ↓
 *        Attestation   Challenge
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import type { DID } from "../../../core/agents/identity/did.js";
import type { PrivateKey } from "../../../core/crypto/keys.js";
import { generateKeypair } from "../../../core/crypto/keys.js";
import {
  TruthController,
  type MessageBroadcaster,
  type TruthStateAccess,
  type ValidatorSelector,
  type TruthControllerConfig,
  type TrackedClaim,
  DEFAULT_TRUTH_CONTROLLER_CONFIG,
} from "../../../protocol/truth_controller.js";
import type {
  TruthMessage,
  ClaimMessage,
  VerificationCommitMessage,
  VerificationRevealMessage,
} from "../../../core/messaging/types.js";
import {
  generateVerificationCommitment,
  generateNonce,
} from "../../../core/messaging/integrity.js";

// ============ Test Fixtures ============

interface TestContext {
  controller: TruthController;
  broadcaster: MockBroadcaster;
  state: MockStateAccess;
  validatorSelector: MockValidatorSelector;
  claimantDid: DID;
  claimantKey: PrivateKey;
  validatorDids: DID[];
  validatorKeys: PrivateKey[];
}

class MockBroadcaster implements MessageBroadcaster {
  public broadcastedMessages: Array<{ topic: string; message: TruthMessage }> =
    [];
  public sentMessages: Array<{ recipient: DID; message: TruthMessage }> = [];

  async broadcast(topic: string, message: TruthMessage): Promise<void> {
    this.broadcastedMessages.push({ topic, message });
  }

  async send(recipient: DID, message: TruthMessage): Promise<void> {
    this.sentMessages.push({ recipient, message });
  }

  clear(): void {
    this.broadcastedMessages = [];
    this.sentMessages = [];
  }

  getLastBroadcast(topic: string): TruthMessage | undefined {
    const msgs = this.broadcastedMessages.filter((m) => m.topic === topic);
    return msgs[msgs.length - 1]?.message;
  }

  getBroadcastCount(topic: string): number {
    return this.broadcastedMessages.filter((m) => m.topic === topic).length;
  }
}

class MockStateAccess implements TruthStateAccess {
  private stakes: Map<string, bigint> = new Map();
  private lockedStakes: Map<string, bigint> = new Map();
  private currentBlock: bigint = 100n;

  getCurrentBlockNumber(): bigint {
    return this.currentBlock;
  }

  setCurrentBlockNumber(block: bigint): void {
    this.currentBlock = block;
  }

  getEndpoint(_did: DID): undefined {
    return undefined;
  }

  getStake(did: DID): bigint {
    return this.stakes.get(did) ?? 0n;
  }

  setStake(did: DID, amount: bigint): void {
    this.stakes.set(did, amount);
  }

  async lockStake(did: DID, amount: bigint, _reason: string): Promise<boolean> {
    const available = this.stakes.get(did) ?? 0n;
    const locked = this.lockedStakes.get(did) ?? 0n;

    if (available - locked < amount) {
      return false;
    }

    this.lockedStakes.set(did, locked + amount);
    return true;
  }

  async unlockStake(did: DID, amount: bigint): Promise<void> {
    const locked = this.lockedStakes.get(did) ?? 0n;
    this.lockedStakes.set(did, locked > amount ? locked - amount : 0n);
  }

  async slashStake(did: DID, amount: bigint, _reason: string): Promise<void> {
    const current = this.stakes.get(did) ?? 0n;
    this.stakes.set(did, current > amount ? current - amount : 0n);
  }

  async rewardStake(did: DID, amount: bigint, _reason: string): Promise<void> {
    const current = this.stakes.get(did) ?? 0n;
    this.stakes.set(did, current + amount);
  }
}

class MockValidatorSelector implements ValidatorSelector {
  private validators: DID[] = [];

  setValidators(validators: DID[]): void {
    this.validators = validators;
  }

  async selectValidators(_claimId: string, count: number): Promise<DID[]> {
    return this.validators.slice(0, count);
  }

  async isSelectedValidator(
    validatorDid: DID,
    _roundId: string,
  ): Promise<boolean> {
    return this.validators.includes(validatorDid);
  }
}

function createTestContext(
  configOverrides: Partial<TruthControllerConfig> = {},
): TestContext {
  // Generate claimant keys
  const claimantKeypair = generateKeypair();
  const claimantDid =
    `did:lgm:claimant-${Date.now()}-${Math.random().toString(36).slice(2)}` as DID;

  // Generate validator keys (5 validators)
  const validatorCount = 5;
  const validatorDids: DID[] = [];
  const validatorKeys: PrivateKey[] = [];

  for (let i = 0; i < validatorCount; i++) {
    const keypair = generateKeypair();
    validatorDids.push(`did:lgm:validator-${i}-${Date.now()}` as DID);
    validatorKeys.push(keypair.privateKey);
  }

  const broadcaster = new MockBroadcaster();
  const state = new MockStateAccess();
  const validatorSelector = new MockValidatorSelector();

  // Set up initial stakes
  state.setStake(claimantDid, 1000n * 10n ** 18n); // 1000 LGM
  for (const validatorDid of validatorDids) {
    state.setStake(validatorDid, 500n * 10n ** 18n); // 500 LGM each
  }

  validatorSelector.setValidators(validatorDids);

  const config = {
    ...DEFAULT_TRUTH_CONTROLLER_CONFIG,
    ...configOverrides,
    minValidatorsForQuorum: 3,
  };

  const controller = new TruthController(
    broadcaster,
    state,
    validatorSelector,
    config,
  );
  controller.setSigningContext(claimantDid, claimantKeypair.privateKey);

  return {
    controller,
    broadcaster,
    state,
    validatorSelector,
    claimantDid,
    claimantKey: claimantKeypair.privateKey,
    validatorDids,
    validatorKeys,
  };
}

/**
 * Helper to transition a claim to evidence_phase state
 */
function transitionToEvidencePhase(
  controller: TruthController,
  claimId: string,
): void {
  const tracked = controller.getClaim(claimId) as TrackedClaim;
  if (tracked) {
    tracked.state = "evidence_phase";
    tracked.updatedAt = Date.now();
  }
}

// ============ Claim Submission Tests ============

describe("TruthController - Claim Submission", () => {
  it("should submit a claim successfully", async () => {
    const ctx = createTestContext();
    const claimPayload = {
      claim: "Water boils at 100°C at sea level",
      claimType: "empirical" as const,
      epistemicCategory: "empirical" as const,
      proposedConfidence: 0.99,
      stake: 100n * 10n ** 18n,
      evidenceIds: [],
    };

    const claimMessage = await ctx.controller.submitClaim(claimPayload);

    expect(claimMessage.messageType).toBe("claim");
    expect(claimMessage.messageId).toMatch(/^msg_/);
    expect(claimMessage.payload.claim).toBe(claimPayload.claim);
    expect(claimMessage.sender).toBe(ctx.claimantDid);
  });

  it("should broadcast claim to the network", async () => {
    const ctx = createTestContext();
    const claimPayload = {
      claim: "Test claim",
      claimType: "empirical" as const,
      epistemicCategory: "empirical" as const,
      proposedConfidence: 0.9,
      stake: 100n * 10n ** 18n,
      evidenceIds: [],
    };

    await ctx.controller.submitClaim(claimPayload);

    expect(ctx.broadcaster.getBroadcastCount("/lgm/claims/1.0.0")).toBe(1);
  });

  it("should track the claim internally", async () => {
    const ctx = createTestContext();
    const claimPayload = {
      claim: "Test claim",
      claimType: "empirical" as const,
      epistemicCategory: "empirical" as const,
      proposedConfidence: 0.9,
      stake: 100n * 10n ** 18n,
      evidenceIds: [],
    };

    const claimMessage = await ctx.controller.submitClaim(claimPayload);
    const tracked = ctx.controller.getClaim(claimMessage.messageId);

    expect(tracked).toBeDefined();
    expect(tracked!.state).toBe("submitted");
    expect(tracked!.message.messageId).toBe(claimMessage.messageId);
  });

  it("should reject claim with insufficient stake", async () => {
    const ctx = createTestContext();
    const claimPayload = {
      claim: "Test claim",
      claimType: "empirical" as const,
      epistemicCategory: "empirical" as const,
      proposedConfidence: 0.9,
      stake: 10n * 10n ** 18n, // Below minimum
      evidenceIds: [],
    };

    await expect(ctx.controller.submitClaim(claimPayload)).rejects.toThrow(
      "Minimum stake",
    );
  });

  it("should emit claim:submitted event", async () => {
    const ctx = createTestContext();
    const eventHandler = vi.fn();
    ctx.controller.on("claim:submitted", eventHandler);

    const claimPayload = {
      claim: "Test claim",
      claimType: "empirical" as const,
      epistemicCategory: "empirical" as const,
      proposedConfidence: 0.9,
      stake: 100n * 10n ** 18n,
      evidenceIds: [],
    };

    await ctx.controller.submitClaim(claimPayload);

    expect(eventHandler).toHaveBeenCalledOnce();
  });
});

// ============ Evidence Handling Tests ============

describe("TruthController - Evidence Handling", () => {
  it("should submit supporting evidence", async () => {
    const ctx = createTestContext();
    const claimMessage = await ctx.controller.submitClaim({
      claim: "Test claim for evidence",
      claimType: "empirical" as const,
      epistemicCategory: "empirical" as const,
      proposedConfidence: 0.9,
      stake: 100n * 10n ** 18n,
      evidenceIds: [],
    });

    const evidencePayload = {
      evidenceType: "peer_reviewed" as const,
      contentHash: "sha256_test_hash",
      originalSource: "https://example.com/study.pdf",
      targetClaimId: claimMessage.messageId,
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

    const evidenceMessage = await ctx.controller.submitEvidence(
      evidencePayload,
      claimMessage.messageId,
    );

    expect(evidenceMessage.messageType).toBe("evidence");
    expect(evidenceMessage.payload.direction).toBe("supporting");
  });

  it("should track evidence with the claim", async () => {
    const ctx = createTestContext();
    const claimMessage = await ctx.controller.submitClaim({
      claim: "Test claim for evidence",
      claimType: "empirical" as const,
      epistemicCategory: "empirical" as const,
      proposedConfidence: 0.9,
      stake: 100n * 10n ** 18n,
      evidenceIds: [],
    });

    const evidencePayload = {
      evidenceType: "peer_reviewed" as const,
      contentHash: "sha256_test_hash",
      originalSource: "https://example.com/study.pdf",
      targetClaimId: claimMessage.messageId,
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

    const evidenceMessage = await ctx.controller.submitEvidence(
      evidencePayload,
      claimMessage.messageId,
    );
    const tracked = ctx.controller.getClaim(claimMessage.messageId);

    expect(tracked!.supportingEvidence).toContain(evidenceMessage.messageId);
  });

  it("should track contradicting evidence separately", async () => {
    const ctx = createTestContext();
    const claimMessage = await ctx.controller.submitClaim({
      claim: "Test claim for evidence",
      claimType: "empirical" as const,
      epistemicCategory: "empirical" as const,
      proposedConfidence: 0.9,
      stake: 100n * 10n ** 18n,
      evidenceIds: [],
    });

    const evidencePayload = {
      evidenceType: "peer_reviewed" as const,
      contentHash: "sha256_counter_hash",
      originalSource: "https://example.com/counter-study.pdf",
      targetClaimId: claimMessage.messageId,
      direction: "contradicting" as const,
      intrinsicStrength: 0.75,
      independence: {
        sourceIndependence: 0.9,
        methodIndependence: 0.8,
        temporalIndependence: 0.7,
        aggregateIndependence: 0.8,
      },
      archiveRequested: true,
    };

    const evidenceMessage = await ctx.controller.submitEvidence(
      evidencePayload,
      claimMessage.messageId,
    );
    const tracked = ctx.controller.getClaim(claimMessage.messageId);

    expect(tracked!.contradictingEvidence).toContain(evidenceMessage.messageId);
  });

  it("should broadcast evidence to the network", async () => {
    const ctx = createTestContext();
    const claimMessage = await ctx.controller.submitClaim({
      claim: "Test claim for evidence",
      claimType: "empirical" as const,
      epistemicCategory: "empirical" as const,
      proposedConfidence: 0.9,
      stake: 100n * 10n ** 18n,
      evidenceIds: [],
    });

    ctx.broadcaster.clear();

    const evidencePayload = {
      evidenceType: "peer_reviewed" as const,
      contentHash: "sha256_test_hash",
      originalSource: "https://example.com/study.pdf",
      targetClaimId: claimMessage.messageId,
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

    await ctx.controller.submitEvidence(
      evidencePayload,
      claimMessage.messageId,
    );

    expect(ctx.broadcaster.getBroadcastCount("/lgm/evidence/1.0.0")).toBe(1);
  });
});

// ============ Evidence Request Tests ============

describe("TruthController - Evidence Requests", () => {
  it("should create evidence request", async () => {
    const ctx = createTestContext();
    const request = await ctx.controller.requestEvidence(
      "climate science data",
      50n * 10n ** 18n,
      100n,
    );

    expect(request.messageType).toBe("evidence_request");
    expect(request.payload.topic).toBe("climate science data");
    expect(request.payload.bounty).toBe(50n * 10n ** 18n);
  });

  it("should broadcast evidence request", async () => {
    const ctx = createTestContext();
    await ctx.controller.requestEvidence("test topic");

    const msg = ctx.broadcaster.getLastBroadcast(
      "/lgm/evidence-requests/1.0.0",
    );
    expect(msg).toBeDefined();
    expect(msg!.messageType).toBe("evidence_request");
  });

  it("should track pending requests", async () => {
    const ctx = createTestContext();
    await ctx.controller.requestEvidence("test topic");

    const pending = ctx.controller.getPendingRequests();
    expect(pending.length).toBe(1);
    expect(pending[0].fulfilled).toBe(false);
  });
});

// ============ Verification Flow Tests ============

describe("TruthController - Verification Flow", () => {
  it("should start verification round", async () => {
    const ctx = createTestContext();
    const claimMessage = await ctx.controller.submitClaim({
      claim: "Test claim for verification",
      claimType: "empirical" as const,
      epistemicCategory: "empirical" as const,
      proposedConfidence: 0.9,
      stake: 100n * 10n ** 18n,
      evidenceIds: [],
    });

    // Transition to evidence phase
    transitionToEvidencePhase(ctx.controller, claimMessage.messageId);

    const roundId = await ctx.controller.startVerification(
      claimMessage.messageId,
    );

    expect(roundId).toBeTruthy();
    expect(typeof roundId).toBe("string");

    const tracked = ctx.controller.getClaim(claimMessage.messageId);
    expect(tracked!.state).toBe("verification");
    expect(tracked!.verificationRoundId).toBe(roundId);
  });

  it("should handle verification commits", async () => {
    const ctx = createTestContext();
    const claimMessage = await ctx.controller.submitClaim({
      claim: "Test claim for verification",
      claimType: "empirical" as const,
      epistemicCategory: "empirical" as const,
      proposedConfidence: 0.9,
      stake: 100n * 10n ** 18n,
      evidenceIds: [],
    });

    transitionToEvidencePhase(ctx.controller, claimMessage.messageId);
    const roundId = await ctx.controller.startVerification(
      claimMessage.messageId,
    );

    const nonce = generateNonce();
    const commitment = generateVerificationCommitment(
      "confirmed",
      0.9,
      nonce,
      [],
    );

    const commitMessage: VerificationCommitMessage = {
      messageType: "verification_commit",
      messageId: `msg_commit_${Date.now()}`,
      version: "1.0.0",
      sender: ctx.validatorDids[0],
      recipient: "broadcast",
      payload: {
        roundId,
        claimId: claimMessage.messageId,
        commitment,
        vrfOutput: "vrf_output_hex",
        vrfProof: "vrf_proof_hex",
      },
      contentHash: "test_hash",
      signature: "test_sig" as never,
      referencedMessages: [claimMessage.messageId],
      networkFeeUnit: 1n,
      networkFeeLimit: 50000n,
      timestamp: Date.now(),
    };

    await ctx.controller.handleVerificationCommit(commitMessage);

    const tracked = ctx.controller.getClaim(claimMessage.messageId);
    expect(tracked!.commits.has(ctx.validatorDids[0])).toBe(true);
  });

  it("should handle verification reveals", async () => {
    const ctx = createTestContext();
    const claimMessage = await ctx.controller.submitClaim({
      claim: "Test claim for verification",
      claimType: "empirical" as const,
      epistemicCategory: "empirical" as const,
      proposedConfidence: 0.9,
      stake: 100n * 10n ** 18n,
      evidenceIds: [],
    });

    transitionToEvidencePhase(ctx.controller, claimMessage.messageId);
    const roundId = await ctx.controller.startVerification(
      claimMessage.messageId,
    );

    const revealMessage: VerificationRevealMessage = {
      messageType: "verification_reveal",
      messageId: `msg_reveal_${Date.now()}`,
      version: "1.0.0",
      sender: ctx.validatorDids[0],
      recipient: "broadcast",
      payload: {
        roundId,
        claimId: claimMessage.messageId,
        verdict: "confirmed",
        confidence: 0.9,
        nonce: "test_nonce",
        referencesChecked: [],
        reasoning: "Evidence supports the claim",
        methodologyUsed: "cross-reference",
      },
      contentHash: "test_hash",
      signature: "test_sig" as never,
      referencedMessages: [claimMessage.messageId],
      networkFeeUnit: 1n,
      networkFeeLimit: 50000n,
      timestamp: Date.now(),
    };

    await ctx.controller.handleVerificationReveal(revealMessage);

    const tracked = ctx.controller.getClaim(claimMessage.messageId);
    expect(tracked!.reveals.has(ctx.validatorDids[0])).toBe(true);
  });

  it("should resolve claim when quorum is reached", async () => {
    const ctx = createTestContext();
    const claimMessage = await ctx.controller.submitClaim({
      claim: "Test claim for verification",
      claimType: "empirical" as const,
      epistemicCategory: "empirical" as const,
      proposedConfidence: 0.9,
      stake: 100n * 10n ** 18n,
      evidenceIds: [],
    });

    transitionToEvidencePhase(ctx.controller, claimMessage.messageId);
    const roundId = await ctx.controller.startVerification(
      claimMessage.messageId,
    );

    const eventHandler = vi.fn();
    ctx.controller.on("claim:resolved", eventHandler);

    // Submit reveals from 3 validators (quorum)
    for (let i = 0; i < 3; i++) {
      const revealMessage: VerificationRevealMessage = {
        messageType: "verification_reveal",
        messageId: `msg_reveal_${i}_${Date.now()}`,
        version: "1.0.0",
        sender: ctx.validatorDids[i],
        recipient: "broadcast",
        payload: {
          roundId,
          claimId: claimMessage.messageId,
          verdict: "confirmed",
          confidence: 0.85 + i * 0.05,
          nonce: `nonce_${i}`,
          referencesChecked: [],
          reasoning: "Evidence supports the claim",
          methodologyUsed: "cross-reference",
        },
        contentHash: "test_hash",
        signature: "test_sig" as never,
        referencedMessages: [claimMessage.messageId],
        networkFeeUnit: 1n,
        networkFeeLimit: 50000n,
        timestamp: Date.now(),
      };

      await ctx.controller.handleVerificationReveal(revealMessage);
    }

    const tracked = ctx.controller.getClaim(claimMessage.messageId);
    expect(tracked!.state).toBe("accepted");
    expect(tracked!.resolution).toBeDefined();
    expect(eventHandler).toHaveBeenCalled();
  });

  it("should reject claim when majority rejects", async () => {
    const ctx = createTestContext();
    const claimMessage = await ctx.controller.submitClaim({
      claim: "Test claim for verification",
      claimType: "empirical" as const,
      epistemicCategory: "empirical" as const,
      proposedConfidence: 0.9,
      stake: 100n * 10n ** 18n,
      evidenceIds: [],
    });

    transitionToEvidencePhase(ctx.controller, claimMessage.messageId);
    const roundId = await ctx.controller.startVerification(
      claimMessage.messageId,
    );

    // Submit reject reveals from 3 validators
    for (let i = 0; i < 3; i++) {
      const revealMessage: VerificationRevealMessage = {
        messageType: "verification_reveal",
        messageId: `msg_reveal_${i}_${Date.now()}`,
        version: "1.0.0",
        sender: ctx.validatorDids[i],
        recipient: "broadcast",
        payload: {
          roundId,
          claimId: claimMessage.messageId,
          verdict: "rejected",
          confidence: 0.1,
          nonce: `nonce_${i}`,
          referencesChecked: [],
          reasoning: "Evidence contradicts the claim",
          methodologyUsed: "cross-reference",
        },
        contentHash: "test_hash",
        signature: "test_sig" as never,
        referencedMessages: [claimMessage.messageId],
        networkFeeUnit: 1n,
        networkFeeLimit: 50000n,
        timestamp: Date.now(),
      };

      await ctx.controller.handleVerificationReveal(revealMessage);
    }

    const tracked = ctx.controller.getClaim(claimMessage.messageId);
    expect(tracked!.state).toBe("rejected");
    expect(tracked!.resolution!.payload.outcome).toBe("rejected");
  });
});

// ============ Challenge Flow Tests ============

describe("TruthController - Challenge Flow", () => {
  it("should submit challenge successfully", async () => {
    const ctx = createTestContext();
    const claimMessage = await ctx.controller.submitClaim({
      claim: "Controversial claim",
      claimType: "empirical" as const,
      epistemicCategory: "empirical" as const,
      proposedConfidence: 0.9,
      stake: 100n * 10n ** 18n,
      evidenceIds: [],
    });

    ctx.broadcaster.clear();

    const challenge = await ctx.controller.submitChallenge(
      claimMessage.messageId,
      "claim",
      "This claim contradicts established facts",
      ["counter-evidence-1"],
      200n * 10n ** 18n,
    );

    expect(challenge.messageType).toBe("challenge");
    expect(challenge.payload.targetId).toBe(claimMessage.messageId);
    expect(challenge.payload.reason).toBe(
      "This claim contradicts established facts",
    );
  });

  it("should transition claim to challenged state", async () => {
    const ctx = createTestContext();
    const claimMessage = await ctx.controller.submitClaim({
      claim: "Controversial claim",
      claimType: "empirical" as const,
      epistemicCategory: "empirical" as const,
      proposedConfidence: 0.9,
      stake: 100n * 10n ** 18n,
      evidenceIds: [],
    });

    await ctx.controller.submitChallenge(
      claimMessage.messageId,
      "claim",
      "Challenge reason",
      ["counter-evidence-1"],
      200n * 10n ** 18n,
    );

    const tracked = ctx.controller.getClaim(claimMessage.messageId);
    expect(tracked!.state).toBe("challenged");
    expect(tracked!.challenges.length).toBe(1);
  });

  it("should reject challenge with insufficient stake", async () => {
    const ctx = createTestContext();
    const claimMessage = await ctx.controller.submitClaim({
      claim: "Controversial claim",
      claimType: "empirical" as const,
      epistemicCategory: "empirical" as const,
      proposedConfidence: 0.9,
      stake: 100n * 10n ** 18n,
      evidenceIds: [],
    });

    await expect(
      ctx.controller.submitChallenge(
        claimMessage.messageId,
        "claim",
        "Challenge reason",
        ["counter-evidence-1"],
        50n * 10n ** 18n, // Below minimum
      ),
    ).rejects.toThrow("Minimum challenge stake");
  });

  it("should emit challenge events", async () => {
    const ctx = createTestContext();
    const claimMessage = await ctx.controller.submitClaim({
      claim: "Controversial claim",
      claimType: "empirical" as const,
      epistemicCategory: "empirical" as const,
      proposedConfidence: 0.9,
      stake: 100n * 10n ** 18n,
      evidenceIds: [],
    });

    const challengeHandler = vi.fn();
    const claimChallengedHandler = vi.fn();

    ctx.controller.on("challenge:submitted", challengeHandler);
    ctx.controller.on("claim:challenged", claimChallengedHandler);

    await ctx.controller.submitChallenge(
      claimMessage.messageId,
      "claim",
      "Challenge reason",
      ["counter-evidence-1"],
      200n * 10n ** 18n,
    );

    expect(challengeHandler).toHaveBeenCalled();
    expect(claimChallengedHandler).toHaveBeenCalled();
  });
});

// ============ Full Lifecycle Test ============

describe("TruthController - Full Lifecycle", () => {
  it("should complete full claim → verification → resolution flow", async () => {
    const ctx = createTestContext();

    // 1. Submit claim
    const claimMessage = await ctx.controller.submitClaim({
      claim: "The speed of light in vacuum is approximately 299,792,458 m/s",
      claimType: "empirical" as const,
      epistemicCategory: "empirical" as const,
      proposedConfidence: 0.999,
      stake: 100n * 10n ** 18n,
      evidenceIds: [],
    });

    expect(ctx.controller.getClaim(claimMessage.messageId)?.state).toBe(
      "submitted",
    );

    // 2. Transition to evidence phase
    transitionToEvidencePhase(ctx.controller, claimMessage.messageId);
    expect(ctx.controller.getClaim(claimMessage.messageId)?.state).toBe(
      "evidence_phase",
    );

    // 3. Submit supporting evidence
    await ctx.controller.submitEvidence(
      {
        evidenceType: "peer_reviewed" as const,
        contentHash: "sha256_physics_paper",
        originalSource: "https://physics.org/speed-of-light",
        targetClaimId: claimMessage.messageId,
        direction: "supporting" as const,
        intrinsicStrength: 0.95,
        independence: {
          sourceIndependence: 0.9,
          methodIndependence: 0.85,
          temporalIndependence: 0.95,
          aggregateIndependence: 0.9,
        },
        archiveRequested: true,
      },
      claimMessage.messageId,
    );

    // 4. Start verification
    const roundId = await ctx.controller.startVerification(
      claimMessage.messageId,
    );
    expect(ctx.controller.getClaim(claimMessage.messageId)?.state).toBe(
      "verification",
    );

    // 5. Submit verification reveals from validators
    for (let i = 0; i < 3; i++) {
      await ctx.controller.handleVerificationReveal({
        messageType: "verification_reveal",
        messageId: `msg_reveal_${i}`,
        version: "1.0.0",
        sender: ctx.validatorDids[i],
        recipient: "broadcast",
        payload: {
          roundId,
          claimId: claimMessage.messageId,
          verdict: "confirmed",
          confidence: 0.99,
          nonce: `nonce_${i}`,
          referencesChecked: ["physics textbook", "NIST reference"],
          reasoning: "Well-established physical constant",
          methodologyUsed: "reference validation",
        },
        contentHash: "test_hash",
        signature: "test_sig" as never,
        referencedMessages: [claimMessage.messageId],
        networkFeeUnit: 1n,
        networkFeeLimit: 50000n,
        timestamp: Date.now(),
      });
    }

    // 6. Verify resolution
    const tracked = ctx.controller.getClaim(claimMessage.messageId);
    expect(tracked!.state).toBe("accepted");
    expect(tracked!.resolution).toBeDefined();
    expect(tracked!.resolution!.payload.outcome).toBe("accepted");
    expect(tracked!.resolution!.payload.finalConfidence).toBeCloseTo(0.99, 1);

    // 7. Verify broadcasts occurred
    const claimBroadcast = ctx.broadcaster.broadcastedMessages.find(
      (m) => m.topic === "/lgm/claims/1.0.0",
    );
    const evidenceBroadcast = ctx.broadcaster.broadcastedMessages.find(
      (m) => m.topic === "/lgm/evidence/1.0.0",
    );
    const resolutionBroadcast = ctx.broadcaster.broadcastedMessages.find(
      (m) => m.topic === "/lgm/resolutions/1.0.0",
    );

    expect(claimBroadcast).toBeDefined();
    expect(evidenceBroadcast).toBeDefined();
    expect(resolutionBroadcast).toBeDefined();
  });

  it("should handle claim → challenge → rejection flow", async () => {
    const ctx = createTestContext();

    // 1. Submit claim
    const claimMessage = await ctx.controller.submitClaim({
      claim: "Controversial scientific claim",
      claimType: "empirical" as const,
      epistemicCategory: "empirical" as const,
      proposedConfidence: 0.8,
      stake: 100n * 10n ** 18n,
      evidenceIds: [],
    });

    // 2. Challenge the claim
    await ctx.controller.submitChallenge(
      claimMessage.messageId,
      "claim",
      "This contradicts peer-reviewed research",
      ["counter-evidence-1"],
      200n * 10n ** 18n,
    );

    expect(ctx.controller.getClaim(claimMessage.messageId)?.state).toBe(
      "challenged",
    );

    // 3. Transition to evidence_phase for verification
    const tracked = ctx.controller.getClaim(claimMessage.messageId)!;
    tracked.state = "evidence_phase";

    const roundId = await ctx.controller.startVerification(
      claimMessage.messageId,
    );

    // 4. Validators reject the claim
    for (let i = 0; i < 3; i++) {
      await ctx.controller.handleVerificationReveal({
        messageType: "verification_reveal",
        messageId: `msg_reveal_reject_${i}`,
        version: "1.0.0",
        sender: ctx.validatorDids[i],
        recipient: "broadcast",
        payload: {
          roundId,
          claimId: claimMessage.messageId,
          verdict: "rejected",
          confidence: 0.1,
          nonce: `nonce_${i}`,
          referencesChecked: ["counter-study"],
          reasoning: "Evidence does not support this claim",
          methodologyUsed: "counter-evidence analysis",
        },
        contentHash: "test_hash",
        signature: "test_sig" as never,
        referencedMessages: [claimMessage.messageId],
        networkFeeUnit: 1n,
        networkFeeLimit: 50000n,
        timestamp: Date.now(),
      });
    }

    // 5. Verify rejection
    const finalTracked = ctx.controller.getClaim(claimMessage.messageId);
    expect(finalTracked!.state).toBe("rejected");
    expect(finalTracked!.resolution!.payload.outcome).toBe("rejected");
  });
});

// ============ Query Methods Tests ============

describe("TruthController - Query Methods", () => {
  it("should get claims by state", async () => {
    const ctx = createTestContext();

    // Submit first claim
    const claim1 = await ctx.controller.submitClaim({
      claim: "Claim 1",
      claimType: "empirical" as const,
      epistemicCategory: "empirical" as const,
      proposedConfidence: 0.9,
      stake: 100n * 10n ** 18n,
      evidenceIds: [],
    });

    // Submit second claim
    await ctx.controller.submitClaim({
      claim: "Claim 2",
      claimType: "empirical" as const,
      epistemicCategory: "empirical" as const,
      proposedConfidence: 0.8,
      stake: 100n * 10n ** 18n,
      evidenceIds: [],
    });

    // Transition one to evidence phase
    transitionToEvidencePhase(ctx.controller, claim1.messageId);

    const submitted = ctx.controller.getClaimsByState("submitted");
    const evidencePhase = ctx.controller.getClaimsByState("evidence_phase");

    expect(submitted.length).toBe(1);
    expect(evidencePhase.length).toBe(1);
  });

  it("should retrieve evidence by ID", async () => {
    const ctx = createTestContext();
    const claimMessage = await ctx.controller.submitClaim({
      claim: "Test claim",
      claimType: "empirical" as const,
      epistemicCategory: "empirical" as const,
      proposedConfidence: 0.9,
      stake: 100n * 10n ** 18n,
      evidenceIds: [],
    });

    const evidenceMessage = await ctx.controller.submitEvidence(
      {
        evidenceType: "peer_reviewed" as const,
        contentHash: "sha256_test",
        originalSource: "https://example.com",
        targetClaimId: claimMessage.messageId,
        direction: "supporting" as const,
        intrinsicStrength: 0.8,
        independence: {
          sourceIndependence: 0.9,
          methodIndependence: 0.8,
          temporalIndependence: 0.7,
          aggregateIndependence: 0.8,
        },
        archiveRequested: true,
      },
      claimMessage.messageId,
    );

    const retrieved = ctx.controller.getEvidence(evidenceMessage.messageId);
    expect(retrieved).toBeDefined();
    expect(retrieved!.payload.contentHash).toBe("sha256_test");
  });
});
