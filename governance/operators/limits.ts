/**
 * Rate Limiting Enforcement
 *
 * Enforces rate limits on governance actions to prevent spam and
 * quality flooding attacks. Limits apply per-agent and per-operator.
 *
 * PP HIERARCHY:
 *   TRUTH: Action counts stored on-chain for verification
 *   UNDERSTANDING: Clear limit definitions
 *   BEAUTY: Sliding window approach for fairness
 *   JUSTICE: Proportional limits by stake
 *   CREATIVITY: New agent restrictions for Sybil resistance
 */

import type { DID } from "../../core/agents/identity/did.js";

// ============ Rate Limit Configuration ============

/**
 * Rate limit window (in blocks).
 * At 2521ms per block, 10000 blocks ≈ 7 hours.
 */
export const RATE_LIMIT_WINDOW = 10000n;

/**
 * Rate limits by action type.
 */
export interface RateLimits {
  lipSubmissions: { perAgent: number; perOperator: number };
  codePRs: { perAgent: number; perOperator: number };
  reviews: { perAgent: number; perOperator: number };
  signalChanges: { perAgent: number }; // No operator limit
  challenges: { perAgent: number; perOperator: number };
  votes: { perAgent: number }; // No operator limit (capped by weight instead)
}

export const DEFAULT_RATE_LIMITS: RateLimits = {
  lipSubmissions: { perAgent: 1, perOperator: 3 },
  codePRs: { perAgent: 5, perOperator: 15 },
  reviews: { perAgent: 10, perOperator: 30 },
  signalChanges: { perAgent: 20 },
  challenges: { perAgent: 3, perOperator: 10 },
  votes: { perAgent: 50 }, // Per LIP, not total
};

/**
 * New agent restrictions.
 */
export interface NewAgentRestrictions {
  restrictionPeriod: bigint; // Blocks
  canSubmitLIPs: boolean;
  canReview: boolean;
  signalWeightMultiplier: number;
  requiresCosigner: boolean;
}

export const DEFAULT_NEW_AGENT_RESTRICTIONS: NewAgentRestrictions = {
  restrictionPeriod: 50000n, // ~1.5 days
  canSubmitLIPs: false,
  canReview: false,
  signalWeightMultiplier: 0.5, // 50% weight
  requiresCosigner: true,
};

// ============ Action Types ============

export type RateLimitedAction =
  | "lip_submission"
  | "code_pr"
  | "review"
  | "signal_change"
  | "challenge"
  | "vote";

// ============ Action Tracking ============

/**
 * Tracked action for rate limiting.
 */
export interface TrackedAction {
  actionId: string;
  agent: DID;
  operatorId?: string;
  actionType: RateLimitedAction;
  block: bigint;
  metadata?: Record<string, unknown>;
}

/**
 * Rate limit check result.
 */
export interface RateLimitResult {
  allowed: boolean;
  reason?: string;
  currentCount: number;
  limit: number;
  resetBlock: bigint;
  cooldownBlocks?: bigint;
}

// ============ Rate Limiter ============

/**
 * Rate limiter for governance actions.
 */
export class RateLimiter {
  private actions: TrackedAction[] = [];
  private limits: RateLimits;
  private newAgentRestrictions: NewAgentRestrictions;

  constructor(
    limits: RateLimits = DEFAULT_RATE_LIMITS,
    newAgentRestrictions: NewAgentRestrictions = DEFAULT_NEW_AGENT_RESTRICTIONS,
  ) {
    this.limits = limits;
    this.newAgentRestrictions = newAgentRestrictions;
  }

  /**
   * Check if an action is allowed under rate limits.
   */
  checkLimit(
    agent: DID,
    operatorId: string | undefined,
    actionType: RateLimitedAction,
    currentBlock: bigint,
    agentRegistrationBlock?: bigint,
  ): RateLimitResult {
    // Prune old actions
    this.pruneOldActions(currentBlock);

    // Check new agent restrictions
    if (agentRegistrationBlock !== undefined) {
      const agentAge = currentBlock - agentRegistrationBlock;
      if (agentAge < this.newAgentRestrictions.restrictionPeriod) {
        const restricted = this.checkNewAgentRestriction(actionType);
        if (restricted) {
          return {
            allowed: false,
            reason: `New agent restriction: ${actionType} not allowed for first ${this.newAgentRestrictions.restrictionPeriod} blocks`,
            currentCount: 0,
            limit: 0,
            resetBlock:
              agentRegistrationBlock +
              this.newAgentRestrictions.restrictionPeriod,
            cooldownBlocks:
              this.newAgentRestrictions.restrictionPeriod - agentAge,
          };
        }
      }
    }

    // Get limits for this action type
    const { agentLimit, operatorLimit } = this.getLimitsForAction(actionType);

    // Count recent actions by this agent
    const agentActions = this.actions.filter(
      (a) => a.agent === agent && a.actionType === actionType,
    );

    if (agentActions.length >= agentLimit) {
      const oldestAction = agentActions[0];
      return {
        allowed: false,
        reason: `Agent rate limit exceeded for ${actionType}`,
        currentCount: agentActions.length,
        limit: agentLimit,
        resetBlock: oldestAction.block + RATE_LIMIT_WINDOW,
        cooldownBlocks: oldestAction.block + RATE_LIMIT_WINDOW - currentBlock,
      };
    }

    // Count recent actions by this operator
    if (operatorId && operatorLimit !== undefined) {
      const operatorActions = this.actions.filter(
        (a) => a.operatorId === operatorId && a.actionType === actionType,
      );

      if (operatorActions.length >= operatorLimit) {
        const oldestAction = operatorActions[0];
        return {
          allowed: false,
          reason: `Operator rate limit exceeded for ${actionType}`,
          currentCount: operatorActions.length,
          limit: operatorLimit,
          resetBlock: oldestAction.block + RATE_LIMIT_WINDOW,
          cooldownBlocks: oldestAction.block + RATE_LIMIT_WINDOW - currentBlock,
        };
      }
    }

    return {
      allowed: true,
      currentCount: agentActions.length,
      limit: agentLimit,
      resetBlock: currentBlock + RATE_LIMIT_WINDOW,
    };
  }

  /**
   * Record an action for rate limiting.
   */
  recordAction(
    agent: DID,
    operatorId: string | undefined,
    actionType: RateLimitedAction,
    currentBlock: bigint,
    metadata?: Record<string, unknown>,
  ): TrackedAction {
    const action: TrackedAction = {
      actionId: `${actionType}-${agent}-${currentBlock}`,
      agent,
      operatorId,
      actionType,
      block: currentBlock,
      metadata,
    };

    this.actions.push(action);
    return action;
  }

  /**
   * Check and record an action atomically.
   */
  checkAndRecord(
    agent: DID,
    operatorId: string | undefined,
    actionType: RateLimitedAction,
    currentBlock: bigint,
    agentRegistrationBlock?: bigint,
    metadata?: Record<string, unknown>,
  ): { allowed: boolean; result: RateLimitResult; action?: TrackedAction } {
    const result = this.checkLimit(
      agent,
      operatorId,
      actionType,
      currentBlock,
      agentRegistrationBlock,
    );

    if (!result.allowed) {
      return { allowed: false, result };
    }

    const action = this.recordAction(
      agent,
      operatorId,
      actionType,
      currentBlock,
      metadata,
    );
    return { allowed: true, result, action };
  }

  /**
   * Get limits for a specific action type.
   */
  private getLimitsForAction(actionType: RateLimitedAction): {
    agentLimit: number;
    operatorLimit?: number;
  } {
    switch (actionType) {
      case "lip_submission":
        return {
          agentLimit: this.limits.lipSubmissions.perAgent,
          operatorLimit: this.limits.lipSubmissions.perOperator,
        };
      case "code_pr":
        return {
          agentLimit: this.limits.codePRs.perAgent,
          operatorLimit: this.limits.codePRs.perOperator,
        };
      case "review":
        return {
          agentLimit: this.limits.reviews.perAgent,
          operatorLimit: this.limits.reviews.perOperator,
        };
      case "signal_change":
        return {
          agentLimit: this.limits.signalChanges.perAgent,
          operatorLimit: undefined, // No operator limit
        };
      case "challenge":
        return {
          agentLimit: this.limits.challenges.perAgent,
          operatorLimit: this.limits.challenges.perOperator,
        };
      case "vote":
        return {
          agentLimit: this.limits.votes.perAgent,
          operatorLimit: undefined, // Capped by weight instead
        };
    }
  }

  /**
   * Check if action is restricted for new agents.
   */
  private checkNewAgentRestriction(actionType: RateLimitedAction): boolean {
    switch (actionType) {
      case "lip_submission":
        return !this.newAgentRestrictions.canSubmitLIPs;
      case "review":
        return !this.newAgentRestrictions.canReview;
      default:
        return false;
    }
  }

  /**
   * Prune actions outside the rate limit window.
   */
  private pruneOldActions(currentBlock: bigint): void {
    const cutoff = currentBlock - RATE_LIMIT_WINDOW;
    this.actions = this.actions.filter((a) => a.block > cutoff);
  }

  /**
   * Get signal weight multiplier for an agent.
   */
  getSignalWeightMultiplier(
    agentRegistrationBlock: bigint,
    currentBlock: bigint,
  ): number {
    const agentAge = currentBlock - agentRegistrationBlock;
    if (agentAge < this.newAgentRestrictions.restrictionPeriod) {
      return this.newAgentRestrictions.signalWeightMultiplier;
    }
    return 1.0;
  }

  /**
   * Check if agent requires a cosigner for code PRs.
   */
  requiresCosigner(
    agentRegistrationBlock: bigint,
    currentBlock: bigint,
  ): boolean {
    const agentAge = currentBlock - agentRegistrationBlock;
    return (
      agentAge < this.newAgentRestrictions.restrictionPeriod &&
      this.newAgentRestrictions.requiresCosigner
    );
  }

  /**
   * Get remaining actions in current window.
   */
  getRemainingActions(
    agent: DID,
    operatorId: string | undefined,
    actionType: RateLimitedAction,
    currentBlock: bigint,
  ): { agentRemaining: number; operatorRemaining?: number } {
    this.pruneOldActions(currentBlock);

    const { agentLimit, operatorLimit } = this.getLimitsForAction(actionType);

    const agentActions = this.actions.filter(
      (a) => a.agent === agent && a.actionType === actionType,
    );

    let operatorRemaining: number | undefined;
    if (operatorId && operatorLimit !== undefined) {
      const operatorActions = this.actions.filter(
        (a) => a.operatorId === operatorId && a.actionType === actionType,
      );
      operatorRemaining = Math.max(0, operatorLimit - operatorActions.length);
    }

    return {
      agentRemaining: Math.max(0, agentLimit - agentActions.length),
      operatorRemaining,
    };
  }

  /**
   * Get action history for an agent.
   */
  getAgentHistory(agent: DID, currentBlock: bigint): TrackedAction[] {
    this.pruneOldActions(currentBlock);
    return this.actions.filter((a) => a.agent === agent);
  }

  /**
   * Get action history for an operator.
   */
  getOperatorHistory(
    operatorId: string,
    currentBlock: bigint,
  ): TrackedAction[] {
    this.pruneOldActions(currentBlock);
    return this.actions.filter((a) => a.operatorId === operatorId);
  }

  /**
   * Export state for persistence.
   */
  exportState(): TrackedAction[] {
    return [...this.actions];
  }

  /**
   * Import state from persistence.
   */
  importState(actions: TrackedAction[]): void {
    this.actions = [...actions];
  }

  /**
   * Get statistics.
   */
  getStats(currentBlock: bigint): {
    totalActions: number;
    byType: Record<RateLimitedAction, number>;
    uniqueAgents: number;
    uniqueOperators: number;
  } {
    this.pruneOldActions(currentBlock);

    const byType: Record<RateLimitedAction, number> = {
      lip_submission: 0,
      code_pr: 0,
      review: 0,
      signal_change: 0,
      challenge: 0,
      vote: 0,
    };

    const agents = new Set<DID>();
    const operators = new Set<string>();

    for (const action of this.actions) {
      byType[action.actionType]++;
      agents.add(action.agent);
      if (action.operatorId) {
        operators.add(action.operatorId);
      }
    }

    return {
      totalActions: this.actions.length,
      byType,
      uniqueAgents: agents.size,
      uniqueOperators: operators.size,
    };
  }
}

/**
 * Create a singleton rate limiter.
 */
let globalLimiter: RateLimiter | null = null;

export function getRateLimiter(): RateLimiter {
  if (!globalLimiter) {
    globalLimiter = new RateLimiter();
  }
  return globalLimiter;
}

export function resetRateLimiter(): void {
  globalLimiter = null;
}
