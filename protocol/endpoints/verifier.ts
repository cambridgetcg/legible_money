/**
 * Verifier Endpoint Implementation
 *
 * Verifiers (validators) participate in claim verification:
 * - Commit to verdicts (commit-reveal mechanism)
 * - Reveal verdicts after commit phase
 * - Validate claims based on evidence
 * - Earn rewards for correct verification
 *
 * Economic model: Verification accuracy affects reputation and selection.
 */

import type {
  VerificationCommitMessage,
  VerificationCommitPayload,
  VerificationRevealMessage,
  VerificationRevealPayload,
  ClaimMessage,
} from "../../core/messaging/types.js";
import type {
  VerifierEndpoint,
  EndpointRole,
  EndpointCapability,
} from "./types.js";
import type { Confidence } from "../../core/knowledge/types.js";
import { BaseEndpoint, type EndpointConfig, type SendOptions } from "./base.js";
import {
  generateVerificationCommitment,
  generateNonce,
} from "../../core/messaging/integrity.js";

// ============ Types ============

/**
 * Verdict options
 */
export type Verdict = "confirmed" | "rejected" | "uncertain";

/**
 * Verification parameters
 */
export interface VerificationParams {
  /** Round ID */
  roundId: string;
  /** Claim ID being verified */
  claimId: string;
  /** The verdict */
  verdict: Verdict;
  /** Confidence in the verdict (0-1) */
  confidence: Confidence;
  /** Evidence IDs that were examined */
  referencesChecked: string[];
  /** VRF output proving selection */
  vrfOutput: string;
  /** VRF proof */
  vrfProof: string;
  /** Reasoning (optional) */
  reasoning?: string;
  /** Methodology used */
  methodologyUsed?: string;
}

/**
 * Pending commitment data
 */
interface PendingCommitment {
  roundId: string;
  claimId: string;
  verdict: Verdict;
  confidence: Confidence;
  nonce: string;
  referencesChecked: string[];
  reasoning?: string;
  methodologyUsed?: string;
  committedAt: number;
}

// ============ Verifier Implementation ============

/**
 * Verifier endpoint for claim verification
 */
export class Verifier extends BaseEndpoint implements VerifierEndpoint {
  // VerifierEndpoint fields
  verificationsParticipated = 0;
  correctVerdicts = 0;
  verificationDomains?: string[];

  // Pending commitments (round ID -> commitment data)
  private readonly pendingCommitments = new Map<string, PendingCommitment>();

  // Verification history
  private readonly verificationHistory = new Map<
    string,
    {
      roundId: string;
      verdict: Verdict;
      confidence: Confidence;
      consensusVerdict?: Verdict;
      correct?: boolean;
    }
  >();

  constructor(config: EndpointConfig & { verificationDomains?: string[] }) {
    super({
      ...config,
      roles: [
        "verifier",
        ...config.roles.filter((r: EndpointRole) => r !== "verifier"),
      ],
      capabilities: [
        "verification_vote",
        "verification_commit",
        "verification_reveal",
        ...config.capabilities.filter(
          (c: EndpointCapability) =>
            ![
              "verification_vote",
              "verification_commit",
              "verification_reveal",
            ].includes(c),
        ),
      ],
    });

    this.verificationDomains = config.verificationDomains;

    // Subscribe to round completions to track accuracy
    this.eventBus.on("round_completed" as never, (event) => {
      this.handleRoundCompleted(
        event as unknown as { payload: { roundId: string; outcome: string } },
      );
    });
  }

  /**
   * Verification accuracy (correct / participated)
   */
  get verificationAccuracy(): number {
    if (this.verificationsParticipated === 0) return 0;
    return this.correctVerdicts / this.verificationsParticipated;
  }

  // ============ Commit Phase ============

  /**
   * Commit to a verification verdict
   *
   * This creates a cryptographic commitment that hides the actual verdict
   * until the reveal phase. Prevents voting cascades.
   */
  async commitVerification(
    params: VerificationParams,
    options: SendOptions = {},
  ): Promise<VerificationCommitMessage> {
    // Validate
    if (params.confidence < 0 || params.confidence > 1) {
      throw new Error("Confidence must be between 0 and 1");
    }

    // Check for existing commitment
    if (this.pendingCommitments.has(params.roundId)) {
      throw new Error(`Already committed to round ${params.roundId}`);
    }

    // Generate nonce
    const nonce = generateNonce();

    // Generate commitment hash
    const commitment = generateVerificationCommitment(
      params.verdict,
      params.confidence,
      nonce,
      params.referencesChecked,
    );

    // Create payload
    const payload: VerificationCommitPayload = {
      roundId: params.roundId,
      claimId: params.claimId,
      commitment,
      vrfOutput: params.vrfOutput,
      vrfProof: params.vrfProof,
    };

    // Create and finalize message
    const messageBase = this.createMessage("verification_commit", payload, {
      ...options,
      recipient: "broadcast",
      referencedMessages: [params.claimId],
    });

    const message = (await this.finalizeMessage(
      messageBase,
    )) as VerificationCommitMessage;

    // Store pending commitment for reveal
    this.pendingCommitments.set(params.roundId, {
      roundId: params.roundId,
      claimId: params.claimId,
      verdict: params.verdict,
      confidence: params.confidence,
      nonce,
      referencesChecked: params.referencesChecked,
      reasoning: params.reasoning,
      methodologyUsed: params.methodologyUsed,
      committedAt: Date.now(),
    });

    // Track participation
    this.verificationsParticipated++;

    // Send
    await this.send(message);

    return message;
  }

  // ============ Reveal Phase ============

  /**
   * Reveal a previous commitment
   *
   * Must be called after commit phase ends. Reveals the actual verdict
   * and allows verification that commitment was genuine.
   */
  async revealVerification(
    roundId: string,
    options: SendOptions = {},
  ): Promise<VerificationRevealMessage> {
    // Get pending commitment
    const pending = this.pendingCommitments.get(roundId);
    if (!pending) {
      throw new Error(`No pending commitment for round ${roundId}`);
    }

    // Create payload
    const payload: VerificationRevealPayload = {
      roundId: pending.roundId,
      claimId: pending.claimId,
      verdict: pending.verdict,
      confidence: pending.confidence,
      nonce: pending.nonce,
      referencesChecked: pending.referencesChecked,
      reasoning: pending.reasoning,
      methodologyUsed: pending.methodologyUsed,
    };

    // Create and finalize message
    const messageBase = this.createMessage("verification_reveal", payload, {
      ...options,
      recipient: "broadcast",
      referencedMessages: [pending.claimId],
    });

    const message = (await this.finalizeMessage(
      messageBase,
    )) as VerificationRevealMessage;

    // Store in history
    this.verificationHistory.set(roundId, {
      roundId,
      verdict: pending.verdict,
      confidence: pending.confidence,
    });

    // Remove from pending
    this.pendingCommitments.delete(roundId);

    // Send
    await this.send(message);

    return message;
  }

  /**
   * Commit and reveal in sequence (for testing/simple cases)
   */
  async verify(
    params: VerificationParams,
    commitOptions: SendOptions = {},
    revealOptions: SendOptions = {},
  ): Promise<{
    commit: VerificationCommitMessage;
    reveal: VerificationRevealMessage;
  }> {
    const commit = await this.commitVerification(params, commitOptions);
    const reveal = await this.revealVerification(params.roundId, revealOptions);
    return { commit, reveal };
  }

  // ============ Verification Helpers ============

  /**
   * Analyze a claim and determine verdict
   *
   * This is a helper that can be overridden by specialized verifiers.
   * Default implementation returns 'uncertain' with low confidence.
   */
  async analyzeClaim(
    _claim: ClaimMessage,
    _evidenceIds: string[],
  ): Promise<{
    verdict: Verdict;
    confidence: Confidence;
    reasoning: string;
    methodology: string;
  }> {
    // Default implementation - subclasses should override
    return {
      verdict: "uncertain",
      confidence: 0.5,
      reasoning: "No automated analysis available",
      methodology: "manual",
    };
  }

  /**
   * Check if this verifier should participate in a round
   */
  shouldParticipate(domain?: string): boolean {
    // If no domain restriction, always participate
    if (!domain) return true;

    // If we have domain expertise, check if it matches
    if (this.verificationDomains) {
      return this.verificationDomains.includes(domain);
    }

    // Default: participate in all domains
    return true;
  }

  // ============ Event Handlers ============

  /**
   * Handle round completion to track accuracy
   */
  private handleRoundCompleted(event: {
    payload: { roundId: string; outcome: string };
  }): void {
    const { roundId, outcome } = event.payload;

    const history = this.verificationHistory.get(roundId);
    if (!history) return;

    // Determine if our verdict matched consensus
    const consensusVerdict = outcome === "accepted" ? "confirmed" : "rejected";
    const correct = history.verdict === consensusVerdict;

    // Update history
    history.consensusVerdict = consensusVerdict as Verdict;
    history.correct = correct;

    // Update accuracy
    if (correct) {
      this.correctVerdicts++;
      this.updateReputation(0.01);
    } else {
      this.updateReputation(-0.02);
    }
  }

  // ============ Queries ============

  /**
   * Get pending commitments
   */
  getPendingCommitments(): PendingCommitment[] {
    return Array.from(this.pendingCommitments.values());
  }

  /**
   * Get verification history
   */
  getVerificationHistory(): Array<{
    roundId: string;
    verdict: Verdict;
    confidence: Confidence;
    consensusVerdict?: Verdict;
    correct?: boolean;
  }> {
    return Array.from(this.verificationHistory.values());
  }

  /**
   * Check if has pending commitment for round
   */
  hasPendingCommitment(roundId: string): boolean {
    return this.pendingCommitments.has(roundId);
  }
}

// ============ Factory ============

/**
 * Create a verifier endpoint
 */
export function createVerifier(
  config: EndpointConfig & { verificationDomains?: string[] },
): Verifier {
  return new Verifier(config);
}
