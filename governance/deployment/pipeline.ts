/**
 * Staged Deployment Pipeline
 *
 * Manages the deployment of approved LIPs through a staged pipeline:
 * devnet → testnet → canary → mainnet. Each stage has specific
 * requirements and allows for rollback if issues are detected.
 *
 * PP HIERARCHY:
 *   TRUTH: Deployment state tracked on-chain
 *   UNDERSTANDING: Clear stage progression and requirements
 *   BEAUTY: Elegant pipeline abstraction
 *   JUSTICE: Appropriate review at each stage
 *   CREATIVITY: Novel staged deployment for blockchain upgrades
 */

import type { DID } from "../../core/agents/identity/did.js";

// ============ Pipeline Stages ============

/**
 * Deployment stage in the pipeline.
 */
export type DeploymentStage =
  | "pending" // Not yet deployed
  | "devnet" // Development network (3 nodes)
  | "testnet" // Test network (20+ validators)
  | "canary" // 10% of mainnet
  | "mainnet_25" // 25% mainnet rollout
  | "mainnet_50" // 50% mainnet rollout
  | "mainnet_100" // Full mainnet deployment
  | "complete" // Successfully deployed
  | "rolled_back"; // Deployment was rolled back

/**
 * Stage configuration.
 */
export interface StageConfig {
  stage: DeploymentStage;
  name: string;
  description: string;
  minDuration: bigint; // Minimum blocks before proceeding
  validators: number; // Minimum validators required
  successMetrics: SuccessMetrics;
  approvers: ApprovalRequirements;
  rollbackWindow: bigint; // Blocks where rollback is allowed
}

export interface SuccessMetrics {
  maxConsensusFailures: number; // Per 1000 blocks
  minUptime: number; // 0.0-1.0
  maxStateCorruption: number; // 0 for critical
  maxPerformanceDegradation: number; // 0.0-1.0
}

export interface ApprovalRequirements {
  agentApprovals: number;
  humanApprovals: number;
  governanceVote: boolean;
}

/**
 * Pipeline stage configurations.
 */
export const STAGE_CONFIGS: Record<DeploymentStage, StageConfig> = {
  pending: {
    stage: "pending",
    name: "Pending",
    description: "Awaiting initial deployment",
    minDuration: 0n,
    validators: 0,
    successMetrics: {
      maxConsensusFailures: 0,
      minUptime: 0,
      maxStateCorruption: 0,
      maxPerformanceDegradation: 0,
    },
    approvers: { agentApprovals: 0, humanApprovals: 0, governanceVote: false },
    rollbackWindow: 0n,
  },
  devnet: {
    stage: "devnet",
    name: "Development Network",
    description: "3 nodes for initial testing",
    minDuration: 68400n, // ~48 hours at 2521ms/block
    validators: 3,
    successMetrics: {
      maxConsensusFailures: 0,
      minUptime: 0.999, // 99.9%
      maxStateCorruption: 0,
      maxPerformanceDegradation: 0.5,
    },
    approvers: { agentApprovals: 2, humanApprovals: 0, governanceVote: false },
    rollbackWindow: 34200n, // ~24 hours
  },
  testnet: {
    stage: "testnet",
    name: "Test Network",
    description: "20+ validators for soak testing",
    minDuration: 239400n, // ~7 days
    validators: 20,
    successMetrics: {
      maxConsensusFailures: 3, // Per 1000 blocks
      minUptime: 0.999,
      maxStateCorruption: 0,
      maxPerformanceDegradation: 0.25,
    },
    approvers: { agentApprovals: 4, humanApprovals: 1, governanceVote: false },
    rollbackWindow: 68400n, // ~48 hours
  },
  canary: {
    stage: "canary",
    name: "Canary (10% Mainnet)",
    description: "10% of mainnet validators",
    minDuration: 478800n, // ~14 days
    validators: 0, // Percentage-based
    successMetrics: {
      maxConsensusFailures: 0,
      minUptime: 0.9999, // 99.99%
      maxStateCorruption: 0,
      maxPerformanceDegradation: 0.1,
    },
    approvers: { agentApprovals: 6, humanApprovals: 2, governanceVote: false },
    rollbackWindow: 136800n, // ~96 hours
  },
  mainnet_25: {
    stage: "mainnet_25",
    name: "Mainnet 25%",
    description: "25% of mainnet validators",
    minDuration: 34200n, // ~24 hours
    validators: 0,
    successMetrics: {
      maxConsensusFailures: 0,
      minUptime: 0.9999,
      maxStateCorruption: 0,
      maxPerformanceDegradation: 0.05,
    },
    approvers: { agentApprovals: 6, humanApprovals: 2, governanceVote: true },
    rollbackWindow: 68400n,
  },
  mainnet_50: {
    stage: "mainnet_50",
    name: "Mainnet 50%",
    description: "50% of mainnet validators",
    minDuration: 34200n,
    validators: 0,
    successMetrics: {
      maxConsensusFailures: 0,
      minUptime: 0.9999,
      maxStateCorruption: 0,
      maxPerformanceDegradation: 0.05,
    },
    approvers: { agentApprovals: 6, humanApprovals: 2, governanceVote: false },
    rollbackWindow: 68400n,
  },
  mainnet_100: {
    stage: "mainnet_100",
    name: "Mainnet 100%",
    description: "Full mainnet deployment",
    minDuration: 68400n, // ~48 hours before complete
    validators: 0,
    successMetrics: {
      maxConsensusFailures: 0,
      minUptime: 0.9999,
      maxStateCorruption: 0,
      maxPerformanceDegradation: 0.05,
    },
    approvers: { agentApprovals: 6, humanApprovals: 2, governanceVote: false },
    rollbackWindow: 68400n,
  },
  complete: {
    stage: "complete",
    name: "Complete",
    description: "Successfully deployed",
    minDuration: 0n,
    validators: 0,
    successMetrics: {
      maxConsensusFailures: 0,
      minUptime: 0,
      maxStateCorruption: 0,
      maxPerformanceDegradation: 0,
    },
    approvers: { agentApprovals: 0, humanApprovals: 0, governanceVote: false },
    rollbackWindow: 0n,
  },
  rolled_back: {
    stage: "rolled_back",
    name: "Rolled Back",
    description: "Deployment was rolled back",
    minDuration: 0n,
    validators: 0,
    successMetrics: {
      maxConsensusFailures: 0,
      minUptime: 0,
      maxStateCorruption: 0,
      maxPerformanceDegradation: 0,
    },
    approvers: { agentApprovals: 0, humanApprovals: 0, governanceVote: false },
    rollbackWindow: 0n,
  },
};

/**
 * Stage progression order.
 */
export const STAGE_ORDER: DeploymentStage[] = [
  "pending",
  "devnet",
  "testnet",
  "canary",
  "mainnet_25",
  "mainnet_50",
  "mainnet_100",
  "complete",
];

// ============ Deployment State ============

/**
 * Deployment record for a LIP.
 */
export interface Deployment {
  deploymentId: string;
  lipId: string;
  currentStage: DeploymentStage;
  stageHistory: StageRecord[];
  metrics: DeploymentMetrics;
  approvals: DeploymentApprovals;
  createdAt: bigint;
  updatedAt: bigint;
}

/**
 * Record of a stage in the deployment.
 */
export interface StageRecord {
  stage: DeploymentStage;
  enteredAt: bigint;
  exitedAt?: bigint;
  outcome: "success" | "failed" | "rolled_back" | "in_progress";
  metrics: StageMetrics;
  notes: string;
}

/**
 * Metrics collected during a stage.
 */
export interface StageMetrics {
  consensusFailures: number;
  uptime: number;
  stateCorruptions: number;
  performanceRatio: number; // 1.0 = baseline, <1.0 = degraded
  blocksProcessed: bigint;
  incidentsDetected: number;
}

/**
 * Overall deployment metrics.
 */
export interface DeploymentMetrics {
  totalBlocks: bigint;
  totalIncidents: number;
  averageUptime: number;
  rollbackCount: number;
}

/**
 * Approvals for deployment stages.
 */
export interface DeploymentApprovals {
  agentApprovals: Map<DeploymentStage, DID[]>;
  humanApprovals: Map<DeploymentStage, string[]>;
  governanceVotes: Map<DeploymentStage, { passed: boolean; voteId: string }>;
}

// ============ Pipeline Operations ============

/**
 * Create a new deployment.
 */
export function createDeployment(
  lipId: string,
  currentBlock: bigint,
): Deployment {
  return {
    deploymentId: `deploy-${lipId}-${currentBlock}`,
    lipId,
    currentStage: "pending",
    stageHistory: [
      {
        stage: "pending",
        enteredAt: currentBlock,
        outcome: "in_progress",
        metrics: {
          consensusFailures: 0,
          uptime: 1.0,
          stateCorruptions: 0,
          performanceRatio: 1.0,
          blocksProcessed: 0n,
          incidentsDetected: 0,
        },
        notes: "Deployment created",
      },
    ],
    metrics: {
      totalBlocks: 0n,
      totalIncidents: 0,
      averageUptime: 1.0,
      rollbackCount: 0,
    },
    approvals: {
      agentApprovals: new Map(),
      humanApprovals: new Map(),
      governanceVotes: new Map(),
    },
    createdAt: currentBlock,
    updatedAt: currentBlock,
  };
}

/**
 * Check if deployment can proceed to next stage.
 */
export function canProceedToNextStage(
  deployment: Deployment,
  currentBlock: bigint,
): { canProceed: boolean; blockers: string[] } {
  const blockers: string[] = [];
  const currentConfig = STAGE_CONFIGS[deployment.currentStage];

  // Check if terminal stage
  if (
    deployment.currentStage === "complete" ||
    deployment.currentStage === "rolled_back"
  ) {
    blockers.push("Deployment is in terminal stage");
    return { canProceed: false, blockers };
  }

  // Get current stage record
  const currentRecord =
    deployment.stageHistory[deployment.stageHistory.length - 1];
  if (!currentRecord) {
    blockers.push("No stage record found");
    return { canProceed: false, blockers };
  }

  // Check minimum duration
  const stageBlocks = currentBlock - currentRecord.enteredAt;
  if (stageBlocks < currentConfig.minDuration) {
    blockers.push(
      `Minimum duration not met: ${stageBlocks}/${currentConfig.minDuration} blocks`,
    );
  }

  // Check success metrics
  const metrics = currentRecord.metrics;
  if (
    metrics.consensusFailures >
    currentConfig.successMetrics.maxConsensusFailures
  ) {
    blockers.push(
      `Too many consensus failures: ${metrics.consensusFailures} > ${currentConfig.successMetrics.maxConsensusFailures}`,
    );
  }
  if (metrics.uptime < currentConfig.successMetrics.minUptime) {
    blockers.push(
      `Uptime too low: ${(metrics.uptime * 100).toFixed(2)}% < ${(currentConfig.successMetrics.minUptime * 100).toFixed(2)}%`,
    );
  }
  if (
    metrics.stateCorruptions > currentConfig.successMetrics.maxStateCorruption
  ) {
    blockers.push("State corruption detected");
  }
  if (
    metrics.performanceRatio <
    1 - currentConfig.successMetrics.maxPerformanceDegradation
  ) {
    blockers.push(
      `Performance degraded: ${(metrics.performanceRatio * 100).toFixed(1)}%`,
    );
  }

  // Check approvals
  const agentApprovals =
    deployment.approvals.agentApprovals.get(deployment.currentStage) || [];
  if (agentApprovals.length < currentConfig.approvers.agentApprovals) {
    blockers.push(
      `Need ${currentConfig.approvers.agentApprovals} agent approvals, have ${agentApprovals.length}`,
    );
  }

  const humanApprovals =
    deployment.approvals.humanApprovals.get(deployment.currentStage) || [];
  if (humanApprovals.length < currentConfig.approvers.humanApprovals) {
    blockers.push(
      `Need ${currentConfig.approvers.humanApprovals} human approvals, have ${humanApprovals.length}`,
    );
  }

  if (currentConfig.approvers.governanceVote) {
    const vote = deployment.approvals.governanceVotes.get(
      deployment.currentStage,
    );
    if (!vote || !vote.passed) {
      blockers.push("Governance vote required but not passed");
    }
  }

  return { canProceed: blockers.length === 0, blockers };
}

/**
 * Get the next stage in the pipeline.
 */
export function getNextStage(
  currentStage: DeploymentStage,
): DeploymentStage | null {
  const currentIndex = STAGE_ORDER.indexOf(currentStage);
  if (currentIndex === -1 || currentIndex >= STAGE_ORDER.length - 1) {
    return null;
  }
  return STAGE_ORDER[currentIndex + 1];
}

/**
 * Advance deployment to next stage.
 */
export function advanceStage(
  deployment: Deployment,
  currentBlock: bigint,
): { success: boolean; deployment?: Deployment; error?: string } {
  const check = canProceedToNextStage(deployment, currentBlock);
  if (!check.canProceed) {
    return {
      success: false,
      error: `Cannot proceed: ${check.blockers.join(", ")}`,
    };
  }

  const nextStage = getNextStage(deployment.currentStage);
  if (!nextStage) {
    return { success: false, error: "Already at final stage" };
  }

  // Close current stage
  const currentRecord =
    deployment.stageHistory[deployment.stageHistory.length - 1];
  if (currentRecord) {
    currentRecord.exitedAt = currentBlock;
    currentRecord.outcome = "success";
  }

  // Create new stage record
  const newRecord: StageRecord = {
    stage: nextStage,
    enteredAt: currentBlock,
    outcome: "in_progress",
    metrics: {
      consensusFailures: 0,
      uptime: 1.0,
      stateCorruptions: 0,
      performanceRatio: 1.0,
      blocksProcessed: 0n,
      incidentsDetected: 0,
    },
    notes: `Advanced from ${deployment.currentStage}`,
  };

  const updatedDeployment: Deployment = {
    ...deployment,
    currentStage: nextStage,
    stageHistory: [...deployment.stageHistory, newRecord],
    updatedAt: currentBlock,
  };

  return { success: true, deployment: updatedDeployment };
}

/**
 * Add agent approval for current stage.
 */
export function addAgentApproval(
  deployment: Deployment,
  agent: DID,
  currentBlock: bigint,
): Deployment {
  const stage = deployment.currentStage;
  const existing = deployment.approvals.agentApprovals.get(stage) || [];

  if (existing.includes(agent)) {
    return deployment; // Already approved
  }

  const newApprovals = new Map(deployment.approvals.agentApprovals);
  newApprovals.set(stage, [...existing, agent]);

  return {
    ...deployment,
    approvals: {
      ...deployment.approvals,
      agentApprovals: newApprovals,
    },
    updatedAt: currentBlock,
  };
}

/**
 * Add human approval for current stage.
 */
export function addHumanApproval(
  deployment: Deployment,
  humanId: string,
  currentBlock: bigint,
): Deployment {
  const stage = deployment.currentStage;
  const existing = deployment.approvals.humanApprovals.get(stage) || [];

  if (existing.includes(humanId)) {
    return deployment;
  }

  const newApprovals = new Map(deployment.approvals.humanApprovals);
  newApprovals.set(stage, [...existing, humanId]);

  return {
    ...deployment,
    approvals: {
      ...deployment.approvals,
      humanApprovals: newApprovals,
    },
    updatedAt: currentBlock,
  };
}

/**
 * Record governance vote result for current stage.
 */
export function recordGovernanceVote(
  deployment: Deployment,
  voteId: string,
  passed: boolean,
  currentBlock: bigint,
): Deployment {
  const newVotes = new Map(deployment.approvals.governanceVotes);
  newVotes.set(deployment.currentStage, { passed, voteId });

  return {
    ...deployment,
    approvals: {
      ...deployment.approvals,
      governanceVotes: newVotes,
    },
    updatedAt: currentBlock,
  };
}

/**
 * Update stage metrics.
 */
export function updateStageMetrics(
  deployment: Deployment,
  metricUpdates: Partial<StageMetrics>,
  currentBlock: bigint,
): Deployment {
  const currentRecord =
    deployment.stageHistory[deployment.stageHistory.length - 1];
  if (!currentRecord) {
    return deployment;
  }

  const updatedMetrics: StageMetrics = {
    ...currentRecord.metrics,
    ...metricUpdates,
  };

  const updatedRecord: StageRecord = {
    ...currentRecord,
    metrics: updatedMetrics,
  };

  const updatedHistory = [...deployment.stageHistory];
  updatedHistory[updatedHistory.length - 1] = updatedRecord;

  return {
    ...deployment,
    stageHistory: updatedHistory,
    updatedAt: currentBlock,
  };
}

/**
 * Check if rollback is within allowed window.
 */
export function canRollback(
  deployment: Deployment,
  currentBlock: bigint,
): boolean {
  const config = STAGE_CONFIGS[deployment.currentStage];
  const currentRecord =
    deployment.stageHistory[deployment.stageHistory.length - 1];

  if (!currentRecord || config.rollbackWindow === 0n) {
    return false;
  }

  const stageAge = currentBlock - currentRecord.enteredAt;
  return stageAge <= config.rollbackWindow;
}

/**
 * Get deployment progress summary.
 */
export function getDeploymentProgress(deployment: Deployment): {
  currentStage: string;
  stagesCompleted: number;
  totalStages: number;
  percentComplete: number;
} {
  const currentIndex = STAGE_ORDER.indexOf(deployment.currentStage);
  const totalStages = STAGE_ORDER.length - 2; // Exclude pending and complete

  return {
    currentStage: STAGE_CONFIGS[deployment.currentStage].name,
    stagesCompleted: Math.max(0, currentIndex - 1),
    totalStages,
    percentComplete: Math.round(((currentIndex - 1) / totalStages) * 100),
  };
}
