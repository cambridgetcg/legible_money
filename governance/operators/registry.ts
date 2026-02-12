/**
 * Operator Registry
 *
 * Tracks operators (entities running AI agents) and enforces caps
 * to prevent Sybil attacks and governance capture. Operators must
 * disclose all agents they control.
 *
 * PP HIERARCHY:
 *   TRUTH: Operator attestations backed by stake
 *   UNDERSTANDING: Clear disclosure requirements
 *   BEAUTY: Simple registry with powerful enforcement
 *   JUSTICE: Severe penalties for undisclosed agents
 *   CREATIVITY: Novel Sybil resistance for AI agent governance
 */

import type { DID } from "../../core/agents/identity/did.js";

// ============ Operator Limits ============

/**
 * Global operator limits for Sybil resistance.
 */
export interface OperatorLimits {
  maxAgentsPerOperator: number;
  maxSignalWeight: number; // Max % of total signal weight
  maxVoteWeight: number; // Max % of total vote weight
  maxReviewersPerLIP: number;
  minDisclosureStake: bigint; // Stake required for operator registration
  updateCooldown: bigint; // Blocks between disclosure updates
}

export const DEFAULT_OPERATOR_LIMITS: OperatorLimits = {
  maxAgentsPerOperator: 10,
  maxSignalWeight: 0.2, // 20%
  maxVoteWeight: 0.15, // 15%
  maxReviewersPerLIP: 2,
  minDisclosureStake: 1000n * 10n ** 15n, // 1 LGM
  updateCooldown: 10000n, // ~7 hours
};

/**
 * Penalties for operator violations.
 */
export interface OperatorPenalties {
  undisclosedAgent: {
    stakeSlash: number; // 1.0 = 100%
    reputationDestruction: boolean;
    governanceBan: boolean;
    contributionReview: boolean; // Trigger review of all contributions
  };
  exceededCap: {
    excessVotesInvalidated: boolean;
    temporaryBan: bigint; // Blocks
  };
}

export const DEFAULT_OPERATOR_PENALTIES: OperatorPenalties = {
  undisclosedAgent: {
    stakeSlash: 1.0,
    reputationDestruction: true,
    governanceBan: true,
    contributionReview: true,
  },
  exceededCap: {
    excessVotesInvalidated: true,
    temporaryBan: 100000n, // ~70 hours
  },
};

// ============ Operator Structure ============

/**
 * Registered operator.
 */
export interface Operator {
  operatorId: string;
  name: string;
  agents: DID[]; // All agents controlled by this operator
  disclosureStake: bigint;
  registeredAt: bigint;
  lastUpdated: bigint;
  status: "active" | "banned" | "suspended";
  violations: OperatorViolation[];
  attestation: OperatorAttestation;
}

/**
 * Operator attestation of agent ownership.
 */
export interface OperatorAttestation {
  statement: string; // "I attest that I control only the listed agents"
  signedAt: bigint;
  signature: string;
  stakeCommitment: bigint;
}

/**
 * Recorded violation.
 */
export interface OperatorViolation {
  violationId: string;
  type: "undisclosed_agent" | "exceeded_cap" | "false_attestation";
  detectedAt: bigint;
  evidence: string;
  penalty: string;
  resolved: boolean;
}

// ============ Operator Registry ============

/**
 * Operator registry for tracking and enforcement.
 */
export class OperatorRegistry {
  private operators: Map<string, Operator> = new Map();
  private agentToOperator: Map<DID, string> = new Map();
  private limits: OperatorLimits;
  private penalties: OperatorPenalties;

  constructor(
    limits: OperatorLimits = DEFAULT_OPERATOR_LIMITS,
    penalties: OperatorPenalties = DEFAULT_OPERATOR_PENALTIES,
  ) {
    this.limits = limits;
    this.penalties = penalties;
  }

  /**
   * Register a new operator.
   */
  registerOperator(params: {
    operatorId: string;
    name: string;
    initialAgents: DID[];
    stake: bigint;
    currentBlock: bigint;
    signature: string;
  }): { success: boolean; operator?: Operator; error?: string } {
    // Check if already registered
    if (this.operators.has(params.operatorId)) {
      return { success: false, error: "Operator already registered" };
    }

    // Check stake
    if (params.stake < this.limits.minDisclosureStake) {
      return {
        success: false,
        error: `Insufficient stake: ${params.stake / 10n ** 18n} < ${this.limits.minDisclosureStake / 10n ** 18n} LGM`,
      };
    }

    // Check agent count
    if (params.initialAgents.length > this.limits.maxAgentsPerOperator) {
      return {
        success: false,
        error: `Too many agents: ${params.initialAgents.length} > ${this.limits.maxAgentsPerOperator}`,
      };
    }

    // Check agents aren't already registered to another operator
    for (const agent of params.initialAgents) {
      const existingOperator = this.agentToOperator.get(agent);
      if (existingOperator) {
        return {
          success: false,
          error: `Agent ${agent} already registered to operator ${existingOperator}`,
        };
      }
    }

    const operator: Operator = {
      operatorId: params.operatorId,
      name: params.name,
      agents: [...params.initialAgents],
      disclosureStake: params.stake,
      registeredAt: params.currentBlock,
      lastUpdated: params.currentBlock,
      status: "active",
      violations: [],
      attestation: {
        statement: `I, ${params.name}, attest that I control only the agents listed in my disclosure.`,
        signedAt: params.currentBlock,
        signature: params.signature,
        stakeCommitment: params.stake,
      },
    };

    // Register
    this.operators.set(params.operatorId, operator);
    for (const agent of params.initialAgents) {
      this.agentToOperator.set(agent, params.operatorId);
    }

    return { success: true, operator };
  }

  /**
   * Add an agent to an operator's disclosure.
   */
  addAgent(
    operatorId: string,
    agentDid: DID,
    currentBlock: bigint,
  ): { success: boolean; error?: string } {
    const operator = this.operators.get(operatorId);
    if (!operator) {
      return { success: false, error: "Operator not found" };
    }

    if (operator.status !== "active") {
      return { success: false, error: `Operator is ${operator.status}` };
    }

    // Check cooldown
    if (currentBlock - operator.lastUpdated < this.limits.updateCooldown) {
      return {
        success: false,
        error: `Update cooldown: ${this.limits.updateCooldown - (currentBlock - operator.lastUpdated)} blocks remaining`,
      };
    }

    // Check agent count
    if (operator.agents.length >= this.limits.maxAgentsPerOperator) {
      return {
        success: false,
        error: `Maximum agents reached: ${this.limits.maxAgentsPerOperator}`,
      };
    }

    // Check agent not already registered
    const existingOperator = this.agentToOperator.get(agentDid);
    if (existingOperator) {
      return {
        success: false,
        error: `Agent already registered to ${existingOperator}`,
      };
    }

    // Add agent
    operator.agents.push(agentDid);
    operator.lastUpdated = currentBlock;
    this.agentToOperator.set(agentDid, operatorId);

    return { success: true };
  }

  /**
   * Remove an agent from an operator's disclosure.
   * Note: Removing an agent that was active in governance triggers review.
   */
  removeAgent(
    operatorId: string,
    agentDid: DID,
    currentBlock: bigint,
  ): { success: boolean; triggerReview: boolean; error?: string } {
    const operator = this.operators.get(operatorId);
    if (!operator) {
      return {
        success: false,
        triggerReview: false,
        error: "Operator not found",
      };
    }

    // Check cooldown
    if (currentBlock - operator.lastUpdated < this.limits.updateCooldown) {
      return {
        success: false,
        triggerReview: false,
        error: `Update cooldown active`,
      };
    }

    const index = operator.agents.indexOf(agentDid);
    if (index === -1) {
      return { success: false, triggerReview: false, error: "Agent not found" };
    }

    // Remove agent
    operator.agents.splice(index, 1);
    operator.lastUpdated = currentBlock;
    this.agentToOperator.delete(agentDid);

    // TODO: Check if agent was active in governance and trigger review
    const triggerReview = false; // Placeholder

    return { success: true, triggerReview };
  }

  /**
   * Get operator by ID.
   */
  getOperator(operatorId: string): Operator | undefined {
    return this.operators.get(operatorId);
  }

  /**
   * Get operator for an agent.
   */
  getOperatorForAgent(agentDid: DID): Operator | undefined {
    const operatorId = this.agentToOperator.get(agentDid);
    if (!operatorId) return undefined;
    return this.operators.get(operatorId);
  }

  /**
   * Check if agents share an operator.
   */
  sameOperator(agent1: DID, agent2: DID): boolean {
    const op1 = this.agentToOperator.get(agent1);
    const op2 = this.agentToOperator.get(agent2);
    return op1 !== undefined && op1 === op2;
  }

  /**
   * Get all agents for an operator.
   */
  getAgentsForOperator(operatorId: string): DID[] {
    const operator = this.operators.get(operatorId);
    return operator ? [...operator.agents] : [];
  }

  /**
   * Record an undisclosed agent violation.
   */
  recordUndisclosedAgent(
    operatorId: string,
    undisclosedAgent: DID,
    evidence: string,
    currentBlock: bigint,
  ): { success: boolean; slashedAmount: bigint; error?: string } {
    const operator = this.operators.get(operatorId);
    if (!operator) {
      return { success: false, slashedAmount: 0n, error: "Operator not found" };
    }

    const violation: OperatorViolation = {
      violationId: `violation-${operatorId}-${currentBlock}`,
      type: "undisclosed_agent",
      detectedAt: currentBlock,
      evidence,
      penalty: "Full stake slash, governance ban, reputation destruction",
      resolved: false,
    };

    operator.violations.push(violation);

    // Apply penalties
    const slashedAmount = operator.disclosureStake;
    operator.disclosureStake = 0n;
    operator.status = "banned";

    // Add undisclosed agent to list (for tracking)
    if (!operator.agents.includes(undisclosedAgent)) {
      operator.agents.push(undisclosedAgent);
      // Only set mapping if agent isn't already mapped to a different operator
      const existingOperator = this.agentToOperator.get(undisclosedAgent);
      if (existingOperator === undefined) {
        this.agentToOperator.set(undisclosedAgent, operatorId);
      }
    }

    return { success: true, slashedAmount };
  }

  /**
   * Check if an operator exceeds vote weight cap.
   */
  checkVoteWeightCap(
    operatorId: string,
    operatorVoteWeight: bigint,
    totalVoteWeight: bigint,
  ): { exceeds: boolean; actualWeightBps: bigint; capBps: bigint } {
    const actualWeightBps =
      totalVoteWeight > 0n
        ? (operatorVoteWeight * 10000n) / totalVoteWeight
        : 0n;
    const capBps = BigInt(Math.round(this.limits.maxVoteWeight * 10000));

    return {
      exceeds: actualWeightBps > capBps,
      actualWeightBps,
      capBps,
    };
  }

  /**
   * Get total vote weight for an operator across all their agents.
   */
  calculateOperatorVoteWeight(
    operatorId: string,
    agentVotes: Map<DID, bigint>,
  ): bigint {
    const operator = this.operators.get(operatorId);
    if (!operator) return 0n;

    let total = 0n;
    for (const agent of operator.agents) {
      total += agentVotes.get(agent) ?? 0n;
    }
    return total;
  }

  /**
   * Cap operator votes to max allowed.
   */
  capOperatorVotes(
    operatorId: string,
    agentVotes: Map<DID, bigint>,
    totalVoteWeight: bigint,
  ): Map<DID, bigint> {
    const operator = this.operators.get(operatorId);
    if (!operator) return agentVotes;

    const operatorWeight = this.calculateOperatorVoteWeight(
      operatorId,
      agentVotes,
    );
    const maxWeight =
      (totalVoteWeight *
        BigInt(Math.round(this.limits.maxVoteWeight * 10000))) /
      10000n;

    if (operatorWeight <= maxWeight) {
      return agentVotes; // No capping needed
    }

    // Apply cap proportionally using pure BigInt math
    const cappedVotes = new Map(agentVotes);
    for (const agent of operator.agents) {
      const originalVote = agentVotes.get(agent);
      if (originalVote) {
        cappedVotes.set(agent, (originalVote * maxWeight) / operatorWeight);
      }
    }

    return cappedVotes;
  }

  /**
   * Get registry statistics.
   */
  getStats(): {
    totalOperators: number;
    activeOperators: number;
    totalAgents: number;
    bannedOperators: number;
    totalViolations: number;
  } {
    let active = 0;
    let banned = 0;
    let violations = 0;

    for (const operator of this.operators.values()) {
      if (operator.status === "active") active++;
      if (operator.status === "banned") banned++;
      violations += operator.violations.length;
    }

    return {
      totalOperators: this.operators.size,
      activeOperators: active,
      totalAgents: this.agentToOperator.size,
      bannedOperators: banned,
      totalViolations: violations,
    };
  }

  /**
   * Export registry state for persistence.
   */
  exportState(): {
    operators: Array<[string, Operator]>;
    agentToOperator: Array<[DID, string]>;
  } {
    return {
      operators: [...this.operators.entries()],
      agentToOperator: [...this.agentToOperator.entries()],
    };
  }

  /**
   * Import registry state from persistence.
   */
  importState(state: {
    operators: Array<[string, Operator]>;
    agentToOperator: Array<[DID, string]>;
  }): void {
    this.operators = new Map(state.operators);
    this.agentToOperator = new Map(state.agentToOperator);
  }
}

/**
 * Create a singleton operator registry.
 */
let globalRegistry: OperatorRegistry | null = null;

export function getOperatorRegistry(): OperatorRegistry {
  if (!globalRegistry) {
    globalRegistry = new OperatorRegistry();
  }
  return globalRegistry;
}

export function resetOperatorRegistry(): void {
  globalRegistry = null;
}
