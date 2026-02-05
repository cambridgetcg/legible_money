/**
 * Reasoning Chain Tracker
 *
 * Session-level tracking of reasoning chains and drift detection.
 * Monitors cumulative drift from grounded knowledge and triggers
 * warnings or forced resets when thresholds are exceeded.
 *
 * Key behaviors:
 * - Tracks cumulative drift across all outputs in a session
 * - Warns when drift approaches threshold
 * - Forces grounding reset when threshold exceeded
 * - Maintains last grounding anchor for re-anchoring
 *
 * PP HIERARCHY:
 *   TRUTH: Ensures agents stay grounded in verified facts over time
 *   UNDERSTANDING: Tracks comprehension quality across session
 *   BEAUTY: Clean session state management
 *   JUSTICE: Placed in core/api alongside anti-hallucination system
 *   CREATIVITY: Novel session-level drift tracking
 */

import type { DID } from "../../core/agents/identity/did.js";
import type {
  ReasoningSessionState,
  GroundingCitation,
  OutputVerificationResult,
  ReasoningNode,
  AntiHallucinationConfig,
  DriftWarningEvent,
  GroundingResetEvent,
} from "./output_verification.js";
import { DEFAULT_ANTI_HALLUCINATION_CONFIG } from "./output_verification.js";

// ============================================================================
// Session Manager
// ============================================================================

/**
 * Manages reasoning sessions for multiple agents.
 */
export class ReasoningSessionManager {
  private sessions: Map<string, ReasoningSessionState> = new Map();
  private config: AntiHallucinationConfig;

  /** Event handlers */
  private onDriftWarning?: (event: DriftWarningEvent) => void;
  private onGroundingReset?: (event: GroundingResetEvent) => void;

  constructor(
    config: AntiHallucinationConfig = DEFAULT_ANTI_HALLUCINATION_CONFIG,
  ) {
    this.config = config;
  }

  /**
   * Set drift warning handler.
   */
  setDriftWarningHandler(handler: (event: DriftWarningEvent) => void): void {
    this.onDriftWarning = handler;
  }

  /**
   * Set grounding reset handler.
   */
  setGroundingResetHandler(
    handler: (event: GroundingResetEvent) => void,
  ): void {
    this.onGroundingReset = handler;
  }

  /**
   * Get or create a session for an agent.
   */
  getOrCreateSession(agentDid: DID, sessionId?: string): ReasoningSessionState {
    const id = sessionId ?? `session:${agentDid}:${Date.now()}`;

    let session = this.sessions.get(id);
    if (!session) {
      session = this.createNewSession(id, agentDid);
      this.sessions.set(id, session);
    }

    return session;
  }

  /**
   * Get existing session by ID.
   */
  getSession(sessionId: string): ReasoningSessionState | undefined {
    return this.sessions.get(sessionId);
  }

  /**
   * Create a new session state.
   */
  private createNewSession(
    sessionId: string,
    agentDid: DID,
  ): ReasoningSessionState {
    const now = Date.now();
    return {
      sessionId,
      agentDid,
      currentChain: [],
      accessedFacts: new Map(),
      cumulativeDrift: 0,
      ungroundedAssertionCount: 0,
      groundingResetCount: 0,
      lastGroundingAnchor: undefined,
      driftWarningThreshold: this.config.driftWarningThreshold,
      driftResetThreshold: this.config.driftResetThreshold,
      startedAt: now,
      lastActivityAt: now,
    };
  }

  /**
   * End a session.
   */
  endSession(sessionId: string): void {
    this.sessions.delete(sessionId);
  }

  /**
   * Get all active sessions.
   */
  getActiveSessions(): ReasoningSessionState[] {
    return Array.from(this.sessions.values());
  }

  /**
   * Get session count.
   */
  getSessionCount(): number {
    return this.sessions.size;
  }

  // ============================================================================
  // Drift Tracking
  // ============================================================================

  /**
   * Update session state after a verification result.
   *
   * This is the main entry point for tracking drift over a session.
   */
  updateSessionWithVerification(
    sessionId: string,
    verificationResult: OutputVerificationResult,
  ): SessionUpdateResult {
    const session = this.sessions.get(sessionId);
    if (!session) {
      return {
        success: false,
        error: "Session not found",
      };
    }

    // Update last activity
    session.lastActivityAt = Date.now();

    // Add drift from this verification
    const driftContribution =
      verificationResult.reasoningChainAnalysis.driftScore;
    session.cumulativeDrift += driftContribution;

    // Track ungrounded assertions
    if (
      verificationResult.groundednessScore < this.config.minGroundednessScore &&
      !verificationResult.passed
    ) {
      session.ungroundedAssertionCount++;
    }

    // Add accessed facts
    for (const citation of verificationResult.citations) {
      if (citation.relationship !== "contradicts") {
        session.accessedFacts.set(citation.factId, citation);
        // Update grounding anchor if this is a strong citation
        if (citation.groundingStrength > 0.5) {
          session.lastGroundingAnchor = citation;
        }
      }
    }

    // Update reasoning chain
    session.currentChain.push(
      ...verificationResult.reasoningChainAnalysis.chain,
    );

    // Check drift thresholds
    let warningIssued = false;
    let resetPerformed = false;

    if (session.cumulativeDrift >= session.driftResetThreshold) {
      // Force grounding reset
      this.forceGroundingReset(session);
      resetPerformed = true;
    } else if (session.cumulativeDrift >= session.driftWarningThreshold) {
      // Issue warning
      this.issueDriftWarning(session);
      warningIssued = true;
    }

    return {
      success: true,
      cumulativeDrift: session.cumulativeDrift,
      driftContribution,
      warningIssued,
      resetPerformed,
    };
  }

  /**
   * Record a fact access without full verification.
   */
  recordFactAccess(sessionId: string, citation: GroundingCitation): void {
    const session = this.sessions.get(sessionId);
    if (!session) return;

    session.lastActivityAt = Date.now();
    session.accessedFacts.set(citation.factId, citation);

    // Update grounding anchor if strong
    if (citation.groundingStrength > 0.5) {
      session.lastGroundingAnchor = citation;
      // Accessing grounded facts reduces drift slightly
      session.cumulativeDrift = Math.max(0, session.cumulativeDrift - 0.1);
    }
  }

  /**
   * Manually add a node to the reasoning chain.
   */
  addReasoningNode(sessionId: string, node: ReasoningNode): void {
    const session = this.sessions.get(sessionId);
    if (!session) return;

    session.lastActivityAt = Date.now();
    session.currentChain.push(node);

    // Add drift based on grounding distance
    if (node.groundingDistance > 0) {
      session.cumulativeDrift += node.groundingDistance * 0.2;
    }
  }

  // ============================================================================
  // Drift Warning and Reset
  // ============================================================================

  /**
   * Issue a drift warning.
   */
  private issueDriftWarning(session: ReasoningSessionState): void {
    const event: DriftWarningEvent = {
      sessionId: session.sessionId,
      agentDid: session.agentDid,
      driftScore: session.cumulativeDrift,
      threshold: session.driftWarningThreshold,
      suggestion: "Query knowledge layer for grounding facts to reduce drift",
      timestamp: Date.now(),
    };

    this.onDriftWarning?.(event);
  }

  /**
   * Force a grounding reset.
   */
  private forceGroundingReset(session: ReasoningSessionState): void {
    const event: GroundingResetEvent = {
      sessionId: session.sessionId,
      agentDid: session.agentDid,
      driftScore: session.cumulativeDrift,
      resetCount: session.groundingResetCount + 1,
      timestamp: Date.now(),
    };

    // Reset session state
    session.currentChain = [];
    session.cumulativeDrift = 0;
    session.groundingResetCount++;
    // Keep accessed facts and last anchor for re-grounding

    this.onGroundingReset?.(event);
  }

  /**
   * Manually trigger a grounding reset.
   */
  manualGroundingReset(sessionId: string): boolean {
    const session = this.sessions.get(sessionId);
    if (!session) return false;

    this.forceGroundingReset(session);
    return true;
  }

  // ============================================================================
  // Session Queries
  // ============================================================================

  /**
   * Get current drift status for a session.
   */
  getDriftStatus(sessionId: string): DriftStatus | undefined {
    const session = this.sessions.get(sessionId);
    if (!session) return undefined;

    const driftRatio = session.cumulativeDrift / session.driftResetThreshold;

    let status: "healthy" | "warning" | "critical";
    if (driftRatio < 0.6) {
      status = "healthy";
    } else if (driftRatio < 1.0) {
      status = "warning";
    } else {
      status = "critical";
    }

    return {
      sessionId,
      cumulativeDrift: session.cumulativeDrift,
      driftRatio,
      status,
      warningThreshold: session.driftWarningThreshold,
      resetThreshold: session.driftResetThreshold,
      resetCount: session.groundingResetCount,
      lastGroundingAnchor: session.lastGroundingAnchor,
      suggestion: this.getDriftSuggestion(status, session),
    };
  }

  /**
   * Get suggestion based on drift status.
   */
  private getDriftSuggestion(
    status: "healthy" | "warning" | "critical",
    session: ReasoningSessionState,
  ): string {
    switch (status) {
      case "healthy":
        return "Reasoning is well-grounded. Continue normal operation.";
      case "warning":
        return `Drift at ${session.cumulativeDrift.toFixed(1)} - query knowledge layer for grounding facts.`;
      case "critical":
        return "Drift threshold exceeded - grounding reset required.";
    }
  }

  /**
   * Get reasoning chain summary for a session.
   */
  getReasoningChainSummary(
    sessionId: string,
  ): ReasoningChainSummary | undefined {
    const session = this.sessions.get(sessionId);
    if (!session) return undefined;

    const chain = session.currentChain;
    const groundedCount = chain.filter(
      (n) => n.stepType === "grounded_fact",
    ).length;
    const inferenceCount = chain.filter(
      (n) => n.stepType === "inference",
    ).length;
    const speculationCount = chain.filter(
      (n) => n.stepType === "speculation",
    ).length;

    return {
      sessionId,
      chainLength: chain.length,
      groundedCount,
      inferenceCount,
      speculationCount,
      groundingRatio: chain.length > 0 ? groundedCount / chain.length : 0,
      accessedFactCount: session.accessedFacts.size,
      lastActivity: session.lastActivityAt,
      sessionDuration: Date.now() - session.startedAt,
    };
  }

  /**
   * Get all accessed facts for a session.
   */
  getAccessedFacts(sessionId: string): GroundingCitation[] {
    const session = this.sessions.get(sessionId);
    if (!session) return [];
    return Array.from(session.accessedFacts.values());
  }

  /**
   * Check if session needs grounding.
   */
  needsGrounding(sessionId: string): boolean {
    const status = this.getDriftStatus(sessionId);
    if (!status) return true; // No session = needs grounding
    return status.status !== "healthy";
  }

  /**
   * Get suggested grounding facts based on session context.
   */
  getSuggestedGroundingFacts(sessionId: string): string[] {
    const session = this.sessions.get(sessionId);
    if (!session) return [];

    // Return most recently accessed high-quality facts
    const facts = Array.from(session.accessedFacts.values())
      .filter((f) => f.groundingStrength > 0.5)
      .sort((a, b) => b.groundingStrength - a.groundingStrength)
      .slice(0, 5)
      .map((f) => f.factId);

    return facts;
  }

  // ============================================================================
  // Cleanup
  // ============================================================================

  /**
   * Clean up stale sessions (no activity for given duration).
   */
  cleanupStaleSessions(maxIdleMs: number = 30 * 60 * 1000): number {
    const now = Date.now();
    let cleaned = 0;

    for (const [sessionId, session] of this.sessions) {
      if (now - session.lastActivityAt > maxIdleMs) {
        this.sessions.delete(sessionId);
        cleaned++;
      }
    }

    return cleaned;
  }
}

// ============================================================================
// Result Types
// ============================================================================

/**
 * Result of updating session with verification.
 */
export interface SessionUpdateResult {
  success: boolean;
  error?: string;
  cumulativeDrift?: number;
  driftContribution?: number;
  warningIssued?: boolean;
  resetPerformed?: boolean;
}

/**
 * Drift status for a session.
 */
export interface DriftStatus {
  sessionId: string;
  cumulativeDrift: number;
  driftRatio: number;
  status: "healthy" | "warning" | "critical";
  warningThreshold: number;
  resetThreshold: number;
  resetCount: number;
  lastGroundingAnchor?: GroundingCitation;
  suggestion: string;
}

/**
 * Summary of reasoning chain for a session.
 */
export interface ReasoningChainSummary {
  sessionId: string;
  chainLength: number;
  groundedCount: number;
  inferenceCount: number;
  speculationCount: number;
  groundingRatio: number;
  accessedFactCount: number;
  lastActivity: number;
  sessionDuration: number;
}

// ============================================================================
// Utility Functions
// ============================================================================

/**
 * Create a singleton session manager with default config.
 */
let defaultManager: ReasoningSessionManager | undefined;

export function getDefaultSessionManager(): ReasoningSessionManager {
  if (!defaultManager) {
    defaultManager = new ReasoningSessionManager();
  }
  return defaultManager;
}

/**
 * Reset the default session manager (for testing).
 */
export function resetDefaultSessionManager(): void {
  defaultManager = undefined;
}

/**
 * Calculate session health score.
 */
export function calculateSessionHealth(status: DriftStatus): number {
  // Health score from 0 (critical) to 1 (healthy)
  const driftPenalty = Math.min(1, status.driftRatio);
  const resetPenalty = Math.min(0.3, status.resetCount * 0.1);
  return Math.max(0, 1 - driftPenalty - resetPenalty);
}
