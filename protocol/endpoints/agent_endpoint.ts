/**
 * AI Agent Endpoint Implementation
 *
 * Adapts AI agents to participate in the truth exchange system:
 * - Autonomous verification with configurable thresholds
 * - Automatic evidence gathering and submission
 * - Challenge detection and response
 * - Rate limiting and human approval gates
 *
 * Economic model: Agents stake on behalf of their operators.
 */

import type { Capability } from "../../core/agents/identity/did.js";
import type {
  ClaimMessage,
  EvidenceMessage,
  ChallengeMessage,
  EvidenceRequestMessage,
} from "../../core/messaging/types.js";
import type {
  AgentTruthEndpoint,
  AgentType,
  AgentAutomationConfig,
} from "./types.js";
import { BaseEndpoint, type EndpointConfig, type SendOptions } from "./base.js";
import { Claimant, type ClaimSubmissionParams } from "./claimant.js";
import {
  Witness,
  type EvidenceSubmissionParams,
  type AttestationParams,
} from "./witness.js";
import { Verifier, type VerificationParams } from "./verifier.js";
import { Challenger, type ChallengeParams } from "./challenger.js";
import {
  verifyAgentOutput,
  type FactStateForVerification,
} from "../../core/api/hallucination_detector.js";
import type {
  AgentOutputClaim,
  OutputVerificationResult,
  AntiHallucinationConfig,
} from "../../core/api/output_verification.js";
import { DEFAULT_ANTI_HALLUCINATION_CONFIG } from "../../core/api/output_verification.js";
import type { OntologicalDomain } from "../../core/knowledge/ontology/types.js";

// ============ Types ============

/**
 * Agent endpoint configuration
 */
export interface AgentEndpointConfig extends EndpointConfig {
  /** Type of AI agent */
  agentType: AgentType;
  /** Model identifier */
  modelId?: string;
  /** Automation configuration */
  automation: AgentAutomationConfig;
  /** Agent capabilities from DID system */
  agentCapabilities?: Capability[];
  /** Orchestration endpoint */
  orchestrationEndpoint?: string;
}

/**
 * Automated action result
 */
export interface AutomatedActionResult {
  action: string;
  success: boolean;
  messageId?: string;
  error?: string;
  requiresApproval?: boolean;
}

/**
 * Human approval request
 */
export interface ApprovalRequest {
  id: string;
  action: string;
  params: unknown;
  stake: bigint;
  reason: string;
  createdAt: number;
  expiresAt: number;
}

// ============ Agent Endpoint Implementation ============

/**
 * AI Agent endpoint adapter
 *
 * Combines capabilities from multiple endpoint types and adds
 * automation features for autonomous participation.
 */
export class AgentEndpoint extends BaseEndpoint implements AgentTruthEndpoint {
  // AgentTruthEndpoint fields
  readonly agentType: AgentType;
  readonly modelId?: string;
  readonly automation: AgentAutomationConfig;
  readonly agentCapabilities: Capability[];
  orchestrationEndpoint?: string;

  // Delegate endpoints
  private readonly claimant: Claimant;
  private readonly witness: Witness;
  private readonly verifier: Verifier;
  private readonly challenger: Challenger;

  // Automation state
  private actionsThisHour = 0;
  private hourStart = Date.now();
  private readonly pendingApprovals = new Map<string, ApprovalRequest>();

  // Event handlers for automation
  private readonly automationHandlers: Map<
    string,
    (event: unknown) => Promise<void>
  >;

  constructor(config: AgentEndpointConfig) {
    super({
      ...config,
      roles: ["claimant", "witness", "verifier", "challenger"],
      capabilities: [
        "claim_submit",
        "evidence_submit",
        "evidence_provide",
        "verification_vote",
        "challenge_initiate",
      ],
    });

    this.agentType = config.agentType;
    this.modelId = config.modelId;
    this.automation = config.automation;
    this.agentCapabilities = config.agentCapabilities ?? [];
    this.orchestrationEndpoint = config.orchestrationEndpoint;

    // Create delegate endpoints
    this.claimant = new Claimant(config);
    this.witness = new Witness(config);
    this.verifier = new Verifier(config);
    this.challenger = new Challenger(config);

    // Setup automation handlers
    this.automationHandlers = new Map();
    this.setupAutomation();
  }

  // ============ Automation Setup ============

  /**
   * Setup automated event handlers
   */
  private setupAutomation(): void {
    // Auto-verify claims
    if (this.automation.autoVerify) {
      this.automationHandlers.set("claim", async (event) => {
        await this.handleAutoVerify(event as { payload: ClaimMessage });
      });
      this.eventBus.on("claim" as never, (event) => {
        this.automationHandlers.get("claim")?.(event);
      });
    }

    // Auto-challenge suspicious facts
    if (this.automation.autoChallenge) {
      this.automationHandlers.set("fact_verified", async (event) => {
        await this.handleAutoChallenge(
          event as { payload: { factId: string; confidence: number } },
        );
      });
      this.eventBus.on("fact_verified" as never, (event) => {
        this.automationHandlers.get("fact_verified")?.(event);
      });
    }

    // Auto-respond to evidence requests
    if (this.automation.autoEvidence) {
      this.automationHandlers.set("evidence_request", async (event) => {
        await this.handleAutoEvidence(
          event as { payload: EvidenceRequestMessage },
        );
      });
      this.eventBus.on("evidence_request" as never, (event) => {
        this.automationHandlers.get("evidence_request")?.(event);
      });
    }
  }

  // ============ Automated Actions ============

  /**
   * Handle automatic verification
   */
  private async handleAutoVerify(event: {
    payload: ClaimMessage;
  }): Promise<void> {
    const claim = event.payload;

    // Check domain authorization
    if (!this.isAuthorizedDomain(claim.domain)) {
      return;
    }

    // Analyze claim
    const analysis = await this.verifier.analyzeClaim(
      claim,
      claim.payload.evidenceIds ?? [],
    );

    // === ANTI-HALLUCINATION VERIFICATION ===
    // Verify output before committing to prevent hallucinated verifications
    const outputClaim: AgentOutputClaim = {
      outputId: `output:${claim.messageId}:${Date.now()}`,
      claim: claim.payload.claim,
      selfConfidence: analysis.confidence,
      domain: claim.domain ?? "general",
      reasoning: analysis.reasoning,
      citedFactIds: claim.payload.evidenceIds ?? [],
      outputType: "assertion",
      createdAt: Date.now(),
    };

    const verificationResult = await verifyAgentOutput(
      outputClaim,
      this.factLookup.bind(this),
      this.domainLookup.bind(this),
      this.antiHallucinationConfig,
    );

    // If verification fails, emit failure event and skip
    if (!verificationResult.passed) {
      this.emitVerificationFailure(claim, verificationResult);
      return;
    }

    // Use effective confidence from verification (may be lower than self-assessed)
    const effectiveConfidence = verificationResult.effectiveConfidence;
    // === END ANTI-HALLUCINATION VERIFICATION ===

    // Check confidence threshold with effective confidence
    if (effectiveConfidence < this.automation.confidenceThreshold) {
      return;
    }

    // Check rate limit
    if (!this.checkAutomationRateLimit()) {
      return;
    }

    // Determine stake (use small stake for auto-verification)
    const stake = this.calculateAutoStake(claim.payload.stake);

    // Check human approval if needed
    if (this.needsHumanApproval(stake)) {
      await this.requestApproval("verify", { claim, analysis }, stake);
      return;
    }

    // Auto-verify with effective confidence
    try {
      await this.verifier.commitVerification({
        roundId: `round-${claim.messageId}`,
        claimId: claim.messageId,
        verdict: analysis.verdict,
        confidence: effectiveConfidence, // Use verified confidence
        referencesChecked: claim.payload.evidenceIds ?? [],
        vrfOutput: "auto-generated", // Would need real VRF
        vrfProof: "auto-generated",
        reasoning: analysis.reasoning,
        methodologyUsed: analysis.methodology,
      });
      this.actionsThisHour++;
    } catch (error) {
      console.error("Auto-verify failed:", error);
    }
  }

  // Anti-hallucination configuration
  private antiHallucinationConfig: AntiHallucinationConfig =
    DEFAULT_ANTI_HALLUCINATION_CONFIG;

  /**
   * Set anti-hallucination configuration
   */
  setAntiHallucinationConfig(config: Partial<AntiHallucinationConfig>): void {
    this.antiHallucinationConfig = {
      ...this.antiHallucinationConfig,
      ...config,
    };
  }

  /**
   * Lookup fact by ID for verification (to be implemented by subclass or injected)
   */
  private async factLookup(
    _factId: string,
  ): Promise<FactStateForVerification | undefined> {
    // This would be connected to the actual state manager
    // For now, return undefined (no fact found = weak grounding)
    return undefined;
  }

  /**
   * Lookup domain by ID for verification (to be implemented by subclass or injected)
   */
  private async domainLookup(
    _domainId: string,
  ): Promise<OntologicalDomain | undefined> {
    // This would be connected to the actual domain registry
    // For now, return undefined
    return undefined;
  }

  /**
   * Emit verification failure event
   */
  private emitVerificationFailure(
    claim: ClaimMessage,
    result: OutputVerificationResult,
  ): void {
    this.eventBus.emit({
      type: "agent_verification_failed" as never,
      timestamp: Date.now(),
      source: "agent",
      payload: {
        agentDid: this.did,
        claimId: claim.messageId,
        groundednessScore: result.groundednessScore,
        effectiveConfidence: result.effectiveConfidence,
        failureReasons: result.failureReasons,
        suggestions: result.suggestions,
        ppScores: result.ppScores,
      } as never,
    });
  }

  /**
   * Handle automatic challenging
   */
  private async handleAutoChallenge(event: {
    payload: { factId: string; confidence: number };
  }): Promise<void> {
    const { factId, confidence } = event.payload;

    // Only challenge low-confidence facts
    if (confidence > 0.5) {
      return;
    }

    // Check rate limit
    if (!this.checkAutomationRateLimit()) {
      return;
    }

    // Calculate stake
    const stake = this.calculateAutoStake(100n * 10n ** 15n); // 0.1 LGM base

    // Check human approval
    if (this.needsHumanApproval(stake)) {
      await this.requestApproval("challenge", { factId, confidence }, stake);
      return;
    }

    // Auto-challenge would need counter-evidence
    // For now, just emit event for manual review
    this.eventBus.emit({
      type: "agent_challenge_candidate" as never,
      timestamp: Date.now(),
      source: "agent",
      payload: {
        agentDid: this.did,
        factId,
        confidence,
        reason: "Low confidence fact detected",
      } as never,
    });
  }

  /**
   * Handle automatic evidence provision
   */
  private async handleAutoEvidence(event: {
    payload: EvidenceRequestMessage;
  }): Promise<void> {
    const request = event.payload;

    // Check domain authorization
    if (
      request.payload.domain &&
      !this.isAuthorizedDomain(request.payload.domain)
    ) {
      return;
    }

    // Check rate limit
    if (!this.checkAutomationRateLimit()) {
      return;
    }

    // Delegate to witness
    // Note: This assumes witness has relevant evidence
    // Real implementation would search agent's knowledge base
    this.actionsThisHour++;
  }

  // ============ Automation Helpers ============

  /**
   * Check if action is within rate limit
   */
  private checkAutomationRateLimit(): boolean {
    const now = Date.now();

    // Reset hourly counter
    if (now - this.hourStart >= 3600000) {
      this.hourStart = now;
      this.actionsThisHour = 0;
    }

    return this.actionsThisHour < this.automation.actionsPerHour;
  }

  /**
   * Check if domain is authorized for automation
   */
  private isAuthorizedDomain(domain?: string): boolean {
    if (!domain) return true;
    if (!this.automation.authorizedDomains) return true;
    return this.automation.authorizedDomains.includes(domain);
  }

  /**
   * Check if action needs human approval
   */
  private needsHumanApproval(stake: bigint): boolean {
    if (!this.automation.humanApprovalThreshold) return false;
    return stake > this.automation.humanApprovalThreshold;
  }

  /**
   * Calculate automatic stake (conservative)
   */
  private calculateAutoStake(baseStake: bigint): bigint {
    const maxStake = this.automation.stakeLimit;
    return baseStake < maxStake ? baseStake : maxStake;
  }

  /**
   * Request human approval for an action
   */
  private async requestApproval(
    action: string,
    params: unknown,
    stake: bigint,
  ): Promise<void> {
    const request: ApprovalRequest = {
      id: `approval-${Date.now()}-${Math.random().toString(36).slice(2)}`,
      action,
      params,
      stake,
      reason: `Stake ${stake} exceeds approval threshold`,
      createdAt: Date.now(),
      expiresAt: Date.now() + 3600000, // 1 hour
    };

    this.pendingApprovals.set(request.id, request);

    // Emit event for external handling
    this.eventBus.emit({
      type: "agent_approval_required" as never,
      timestamp: Date.now(),
      source: "agent",
      payload: {
        agentDid: this.did,
        request,
      } as never,
    });
  }

  // ============ Manual Actions (Delegated) ============

  /**
   * Submit a claim (delegated to claimant)
   */
  async submitClaim(
    params: ClaimSubmissionParams,
    options?: SendOptions,
  ): Promise<ClaimMessage> {
    return this.claimant.submitClaim(params, options);
  }

  /**
   * Submit evidence (delegated to witness)
   */
  async submitEvidence(
    params: EvidenceSubmissionParams,
    options?: SendOptions,
  ): Promise<EvidenceMessage> {
    return this.witness.submitEvidence(params, options);
  }

  /**
   * Attest to something (delegated to witness)
   */
  async attest(
    params: AttestationParams,
    options?: SendOptions,
  ): Promise<ReturnType<Witness["attest"]>> {
    return this.witness.attest(params, options);
  }

  /**
   * Verify a claim (delegated to verifier)
   */
  async verify(
    params: VerificationParams,
    options?: SendOptions,
  ): Promise<ReturnType<Verifier["verify"]>> {
    return this.verifier.verify(params, options, options);
  }

  /**
   * Challenge something (delegated to challenger)
   */
  async challenge(
    params: ChallengeParams,
    options?: SendOptions,
  ): Promise<ChallengeMessage> {
    return this.challenger.challenge(params, options);
  }

  // ============ Approval Management ============

  /**
   * Approve a pending action
   */
  async approveAction(approvalId: string): Promise<AutomatedActionResult> {
    const request = this.pendingApprovals.get(approvalId);
    if (!request) {
      return { action: "unknown", success: false, error: "Approval not found" };
    }

    // Check expiry
    if (Date.now() > request.expiresAt) {
      this.pendingApprovals.delete(approvalId);
      return {
        action: request.action,
        success: false,
        error: "Approval expired",
      };
    }

    // Execute approved action
    try {
      let messageId: string | undefined;

      switch (request.action) {
        case "verify":
          // Re-run verification with approval
          break;
        case "challenge":
          // Execute challenge
          break;
        default:
          return {
            action: request.action,
            success: false,
            error: "Unknown action",
          };
      }

      this.pendingApprovals.delete(approvalId);
      return { action: request.action, success: true, messageId };
    } catch (error) {
      return {
        action: request.action,
        success: false,
        error: error instanceof Error ? error.message : "Unknown error",
      };
    }
  }

  /**
   * Reject a pending action
   */
  rejectAction(approvalId: string): void {
    this.pendingApprovals.delete(approvalId);
  }

  /**
   * Get pending approvals
   */
  getPendingApprovals(): ApprovalRequest[] {
    return Array.from(this.pendingApprovals.values());
  }

  // ============ Statistics ============

  /**
   * Get automation statistics
   */
  getAutomationStats(): {
    actionsThisHour: number;
    pendingApprovals: number;
    autoVerifyEnabled: boolean;
    autoChallengeEnabled: boolean;
    autoEvidenceEnabled: boolean;
  } {
    return {
      actionsThisHour: this.actionsThisHour,
      pendingApprovals: this.pendingApprovals.size,
      autoVerifyEnabled: this.automation.autoVerify,
      autoChallengeEnabled: this.automation.autoChallenge,
      autoEvidenceEnabled: this.automation.autoEvidence,
    };
  }

  /**
   * Get delegate statistics
   */
  getDelegateStats(): {
    claimant: { submitted: number; accepted: number; rejected: number };
    witness: { submitted: number; accepted: number; quality: number };
    verifier: { participated: number; correct: number; accuracy: number };
    challenger: { initiated: number; won: number; successRate: number };
  } {
    return {
      claimant: {
        submitted: this.claimant.claimsSubmitted,
        accepted: this.claimant.claimsAccepted,
        rejected: this.claimant.claimsRejected,
      },
      witness: {
        submitted: this.witness.evidenceSubmitted,
        accepted: this.witness.evidenceAccepted,
        quality: this.witness.evidenceQualityScore,
      },
      verifier: {
        participated: this.verifier.verificationsParticipated,
        correct: this.verifier.correctVerdicts,
        accuracy: this.verifier.verificationAccuracy,
      },
      challenger: {
        initiated: this.challenger.challengesInitiated,
        won: this.challenger.challengesWon,
        successRate: this.challenger.challengeSuccessRate,
      },
    };
  }

  // ============ Shutdown ============

  /**
   * Shutdown agent endpoint
   */
  shutdown(): void {
    super.shutdown();

    // Clear automation handlers
    this.automationHandlers.clear();

    // Shutdown delegates
    this.claimant.shutdown();
    this.witness.shutdown();
    this.verifier.shutdown();
    this.challenger.shutdown();
  }
}

// ============ Factory ============

/**
 * Create an AI agent endpoint
 */
export function createAgentEndpoint(
  config: AgentEndpointConfig,
): AgentEndpoint {
  return new AgentEndpoint(config);
}

/**
 * Create an inference agent (general reasoning)
 */
export function createInferenceAgent(
  config: Omit<AgentEndpointConfig, "agentType">,
): AgentEndpoint {
  return new AgentEndpoint({ ...config, agentType: "inference" });
}

/**
 * Create a verification agent
 */
export function createVerificationAgent(
  config: Omit<AgentEndpointConfig, "agentType">,
): AgentEndpoint {
  return new AgentEndpoint({
    ...config,
    agentType: "verification",
    automation: {
      ...config.automation,
      autoVerify: true,
    },
  });
}

/**
 * Create a research agent
 */
export function createResearchAgent(
  config: Omit<AgentEndpointConfig, "agentType">,
): AgentEndpoint {
  return new AgentEndpoint({
    ...config,
    agentType: "research",
    automation: {
      ...config.automation,
      autoEvidence: true,
    },
  });
}

/**
 * Create an oracle agent
 */
export function createOracleAgent(
  config: Omit<AgentEndpointConfig, "agentType">,
): AgentEndpoint {
  return new AgentEndpoint({ ...config, agentType: "oracle" });
}
