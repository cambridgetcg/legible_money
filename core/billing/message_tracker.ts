/**
 * Message Billing Tracker
 *
 * GAP-007: Tracks costs across queries within a message, applies complexity
 * multipliers, and provides streaming checkpoints for real-time billing.
 *
 * Lifecycle:
 *   1. startMessage(message) → creates ActiveMessage
 *   2. recordQuery(queryCost) → accumulates query costs
 *   3. updateTokens(count) → triggers streaming checkpoints
 *   4. completeMessage() → calculates final MessageCost
 *
 * PP HIERARCHY:
 *   TRUTH: Actual costs tracked and aggregated
 *   UNDERSTANDING: Clear message lifecycle and cost breakdown
 *   BEAUTY: Clean separation from query-level billing
 *   JUSTICE: Fair aggregation with complexity adjustment
 *   CREATIVITY: Streaming checkpoints enable real-time budget control
 */

import type { QueryCostEstimate } from "../knowledge/query/types.js";
import {
  type ChatMessage,
  type MessageCost,
  type ActiveMessage,
  type StreamingCheckpoint,
  type MessageBillingConfig,
  type SessionMessageStats,
  type MessageCostEstimate,
  DEFAULT_MESSAGE_BILLING_CONFIG,
} from "./message_billing.js";
import {
  estimateReasoningDepth,
  calculateComplexityMultiplier,
} from "./token_counter.js";

// ============================================================================
// Session Budget Interface (minimal interface to avoid circular deps)
// ============================================================================

/**
 * Minimal session interface for budget checking
 */
export interface BudgetSession {
  sessionId: string;
  totalCost: bigint;
  budgetLimit?: bigint;
}

// ============================================================================
// Message Billing Tracker
// ============================================================================

/**
 * Tracks message costs and provides billing aggregation.
 */
export class MessageBillingTracker {
  private config: MessageBillingConfig;

  /** Active messages being tracked */
  private activeMessages: Map<string, ActiveMessage> = new Map();

  /** Completed message history by session */
  private messageHistory: Map<string, MessageCost[]> = new Map();

  /** Session message counts for free tier */
  private sessionMessageCounts: Map<string, number> = new Map();

  constructor(config: MessageBillingConfig = DEFAULT_MESSAGE_BILLING_CONFIG) {
    this.config = { ...config };
  }

  // ============================================================================
  // Message Lifecycle
  // ============================================================================

  /**
   * Start tracking a new message.
   */
  startMessage(message: ChatMessage): ActiveMessage {
    const active: ActiveMessage = {
      messageId: message.messageId,
      sessionId: message.sessionId,
      agentDid: message.agentDid,
      type: message.type,
      startedAt: Date.now(),
      queries: [],
      checkpoints: [],
      currentTokens: 0,
      provisionalCost: this.config.baseMessageCost,
      factsAccessed: [],
      crossStratumQueries: 0,
    };

    this.activeMessages.set(message.messageId, active);
    return active;
  }

  /**
   * Record a query within the current message.
   */
  recordQuery(
    messageId: string,
    queryCost: QueryCostEstimate,
    factsAccessed?: string[],
    isCrossStratum?: boolean,
  ): void {
    const active = this.activeMessages.get(messageId);
    if (!active) {
      return; // Silently ignore if message not found
    }

    active.queries.push(queryCost);
    active.provisionalCost += queryCost.cost;

    if (factsAccessed) {
      for (const factId of factsAccessed) {
        if (!active.factsAccessed.includes(factId)) {
          active.factsAccessed.push(factId);
        }
      }
    }

    if (isCrossStratum) {
      active.crossStratumQueries++;
    }
  }

  /**
   * Update token count (for streaming).
   * Returns a checkpoint if one was triggered.
   */
  updateTokens(
    messageId: string,
    tokenCount: number,
    session: BudgetSession,
  ): StreamingCheckpoint | null {
    const active = this.activeMessages.get(messageId);
    if (!active) {
      return null;
    }

    active.currentTokens = tokenCount;

    // Check if we need a checkpoint
    const checkpointIndex = Math.floor(
      tokenCount / this.config.streamingCheckpointInterval,
    );
    if (checkpointIndex > active.checkpoints.length) {
      return this.createCheckpoint(active, session);
    }

    return null;
  }

  /**
   * Complete message and calculate final cost.
   */
  completeMessage(
    messageId: string,
    finalTokenCount: number,
    reasoningDepth: number | undefined,
    session: BudgetSession,
  ): MessageCost {
    const active = this.activeMessages.get(messageId);
    if (!active) {
      throw new Error(`Message ${messageId} not found`);
    }

    // Auto-estimate reasoning depth if not provided
    const depth =
      reasoningDepth ??
      estimateReasoningDepth(
        active.factsAccessed.length,
        active.queries.length,
        active.crossStratumQueries,
      );

    // Calculate costs
    const messageCost = this.calculateMessageCost(
      active,
      finalTokenCount,
      depth,
      session,
    );

    // Store in history
    const history = this.messageHistory.get(active.sessionId) || [];
    history.push(messageCost);
    this.messageHistory.set(active.sessionId, history);

    // Update session message count
    const count = this.sessionMessageCounts.get(active.sessionId) || 0;
    this.sessionMessageCounts.set(active.sessionId, count + 1);

    // Clean up active message
    this.activeMessages.delete(messageId);

    return messageCost;
  }

  /**
   * Abort message (e.g., budget exhausted, error).
   * Returns partial cost incurred.
   */
  abortMessage(messageId: string): bigint {
    const active = this.activeMessages.get(messageId);
    if (!active) {
      return 0n;
    }

    const partialCost = active.provisionalCost;
    this.activeMessages.delete(messageId);
    return partialCost;
  }

  // ============================================================================
  // Cost Calculation
  // ============================================================================

  /**
   * Calculate final message cost.
   */
  private calculateMessageCost(
    active: ActiveMessage,
    tokenCount: number,
    reasoningDepth: number,
    session: BudgetSession,
  ): MessageCost {
    // Sum query costs
    const queryCost = active.queries.reduce((sum, q) => sum + q.cost, 0n);

    // Token-based cost
    const tokenCost =
      (BigInt(tokenCount) * this.config.costPer1kTokens) / 1000n;

    // Base cost
    const baseCost = this.config.baseMessageCost + queryCost + tokenCost;

    // Complexity multiplier
    const complexityMultiplier = calculateComplexityMultiplier(
      tokenCount,
      reasoningDepth,
      this.config.complexityThresholds,
      this.config.depthMultipliers,
    );

    // Complexity premium (additional cost from multiplier)
    const multiplierFactor = Math.max(0, complexityMultiplier - 1);
    const complexityPremium = BigInt(
      Math.floor(Number(baseCost) * multiplierFactor),
    );

    // Streaming fees
    const streamingFees =
      BigInt(active.checkpoints.length) * this.config.streamingCheckpointFee;

    // Check free tier
    const freeTier = this.isWithinFreeTier(session.sessionId, tokenCount);

    // Total cost (0 if free tier)
    const totalCost = freeTier
      ? 0n
      : baseCost + complexityPremium + streamingFees;

    // Research and knowledge shares
    const researchContribution = BigInt(
      Math.floor(Number(totalCost) * this.config.researchShareRate),
    );
    const knowledgeRewards = BigInt(
      Math.floor(Number(totalCost) * this.config.knowledgeRewardsRate),
    );

    return {
      messageId: active.messageId,
      sessionId: active.sessionId,
      baseCost,
      complexityMultiplier,
      totalCost,
      breakdown: {
        queryCost,
        queryCount: active.queries.length,
        tokenCost,
        complexityPremium,
        streamingFees,
      },
      queryCosts: active.queries,
      researchContribution,
      knowledgeRewards,
      freeTier,
      tokenCount,
      reasoningDepth,
      completedAt: Date.now(),
    };
  }

  /**
   * Check if message is within free tier.
   */
  private isWithinFreeTier(sessionId: string, tokenCount: number): boolean {
    const messageCount = this.sessionMessageCounts.get(sessionId) || 0;

    // Check message count
    if (messageCount >= this.config.freeTierMessagesPerSession) {
      return false;
    }

    // Check token count
    if (tokenCount > this.config.freeTierMaxTokens) {
      return false;
    }

    return true;
  }

  // ============================================================================
  // Streaming Checkpoints
  // ============================================================================

  /**
   * Create a streaming checkpoint.
   */
  private createCheckpoint(
    active: ActiveMessage,
    session: BudgetSession,
  ): StreamingCheckpoint {
    const provisionalCost = this.estimateProvisionalCost(active);

    const budgetAvailable =
      session.budgetLimit === undefined ||
      session.totalCost + provisionalCost <= session.budgetLimit;

    const checkpoint: StreamingCheckpoint = {
      index: active.checkpoints.length,
      tokensProcessed: active.currentTokens,
      provisionalCost,
      timestamp: Date.now(),
      budgetAvailable,
      remainingBudget:
        session.budgetLimit !== undefined
          ? session.budgetLimit - session.totalCost - provisionalCost
          : undefined,
    };

    active.checkpoints.push(checkpoint);
    active.provisionalCost = provisionalCost;

    return checkpoint;
  }

  /**
   * Estimate provisional cost at current state.
   */
  private estimateProvisionalCost(active: ActiveMessage): bigint {
    const queryCost = active.queries.reduce((sum, q) => sum + q.cost, 0n);
    const tokenCost =
      (BigInt(active.currentTokens) * this.config.costPer1kTokens) / 1000n;
    return this.config.baseMessageCost + queryCost + tokenCost;
  }

  // ============================================================================
  // Pre-Execution Estimation
  // ============================================================================

  /**
   * Estimate cost before message execution.
   */
  estimateMessageCost(
    sessionId: string,
    estimatedTokens: number,
    estimatedQueries: number,
    session: BudgetSession,
  ): MessageCostEstimate {
    const messageCount = this.sessionMessageCounts.get(sessionId) || 0;

    // Check free tier
    const wouldBeFreeTier =
      messageCount < this.config.freeTierMessagesPerSession &&
      estimatedTokens <= this.config.freeTierMaxTokens;

    // Estimate base cost
    const tokenCost =
      (BigInt(estimatedTokens) * this.config.costPer1kTokens) / 1000n;
    const queryCost = BigInt(estimatedQueries) * 100n; // Rough query estimate
    const estimatedBaseCost =
      this.config.baseMessageCost + tokenCost + queryCost;

    // Estimate multiplier (assume moderate complexity)
    const estimatedDepth = Math.min(
      5,
      Math.max(1, Math.ceil(estimatedQueries / 2)),
    );
    const estimatedMultiplier = calculateComplexityMultiplier(
      estimatedTokens,
      estimatedDepth,
      this.config.complexityThresholds,
      this.config.depthMultipliers,
    );

    // Estimate total
    const multiplierFactor = Math.max(0, estimatedMultiplier - 1);
    const complexityPremium = BigInt(
      Math.floor(Number(estimatedBaseCost) * multiplierFactor),
    );
    const estimatedTotalCost = wouldBeFreeTier
      ? 0n
      : estimatedBaseCost + complexityPremium;

    // Check budget
    const hasBudget =
      session.budgetLimit === undefined ||
      session.totalCost + estimatedTotalCost <= session.budgetLimit;

    const remainingBudget =
      session.budgetLimit !== undefined
        ? session.budgetLimit - session.totalCost
        : undefined;

    return {
      estimatedBaseCost,
      estimatedTotalCost,
      wouldBeFreeTier,
      estimatedMultiplier,
      hasBudget,
      remainingBudget,
    };
  }

  // ============================================================================
  // Queries
  // ============================================================================

  /**
   * Get active message by ID.
   */
  getActiveMessage(messageId: string): ActiveMessage | undefined {
    return this.activeMessages.get(messageId);
  }

  /**
   * Get message history for session.
   */
  getSessionHistory(sessionId: string): MessageCost[] {
    return this.messageHistory.get(sessionId) || [];
  }

  /**
   * Get total session message cost.
   */
  getSessionMessageCost(sessionId: string): bigint {
    const history = this.messageHistory.get(sessionId) || [];
    return history.reduce((sum, m) => sum + m.totalCost, 0n);
  }

  /**
   * Get session message statistics.
   */
  getSessionStats(sessionId: string): SessionMessageStats {
    const history = this.messageHistory.get(sessionId) || [];

    if (history.length === 0) {
      return {
        messageCount: 0,
        totalCost: 0n,
        totalTokens: 0,
        totalQueries: 0,
        freeTierMessages: 0,
        avgComplexityMultiplier: 1.0,
        totalResearchContribution: 0n,
        totalKnowledgeRewards: 0n,
      };
    }

    let totalTokens = 0;
    let totalQueries = 0;
    let freeTierMessages = 0;
    let totalMultiplier = 0;
    let totalResearch = 0n;
    let totalKnowledge = 0n;

    for (const msg of history) {
      totalTokens += msg.tokenCount;
      totalQueries += msg.breakdown.queryCount;
      if (msg.freeTier) freeTierMessages++;
      totalMultiplier += msg.complexityMultiplier;
      totalResearch += msg.researchContribution;
      totalKnowledge += msg.knowledgeRewards;
    }

    return {
      messageCount: history.length,
      totalCost: this.getSessionMessageCost(sessionId),
      totalTokens,
      totalQueries,
      freeTierMessages,
      avgComplexityMultiplier: totalMultiplier / history.length,
      totalResearchContribution: totalResearch,
      totalKnowledgeRewards: totalKnowledge,
    };
  }

  /**
   * Get remaining free tier messages for session.
   */
  getRemainingFreeTierMessages(sessionId: string): number {
    const count = this.sessionMessageCounts.get(sessionId) || 0;
    return Math.max(0, this.config.freeTierMessagesPerSession - count);
  }

  /**
   * Check if session can afford a message (quick check).
   */
  canAffordMessage(
    session: BudgetSession,
    estimatedTokens: number,
    estimatedQueries: number,
  ): boolean {
    const estimate = this.estimateMessageCost(
      session.sessionId,
      estimatedTokens,
      estimatedQueries,
      session,
    );
    return estimate.hasBudget;
  }

  // ============================================================================
  // Session Management
  // ============================================================================

  /**
   * Clear session data (when session ends).
   */
  clearSession(sessionId: string): void {
    this.messageHistory.delete(sessionId);
    this.sessionMessageCounts.delete(sessionId);

    // Clear any active messages for this session
    for (const [messageId, active] of this.activeMessages) {
      if (active.sessionId === sessionId) {
        this.activeMessages.delete(messageId);
      }
    }
  }

  /**
   * Get all active message IDs.
   */
  getActiveMessageIds(): string[] {
    return Array.from(this.activeMessages.keys());
  }

  /**
   * Get configuration.
   */
  getConfig(): MessageBillingConfig {
    return { ...this.config };
  }

  /**
   * Update configuration.
   */
  updateConfig(config: Partial<MessageBillingConfig>): void {
    this.config = { ...this.config, ...config };
  }
}
