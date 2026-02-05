/**
 * Rollback Mechanisms
 *
 * Handles rollback of deployments when issues are detected.
 * Supports automatic rollback triggers and manual rollback requests.
 *
 * PP HIERARCHY:
 *   TRUTH: Rollback decisions based on objective metrics
 *   UNDERSTANDING: Clear trigger conditions
 *   BEAUTY: Clean rollback state machine
 *   JUSTICE: Fair rollback process with accountability
 *   CREATIVITY: Novel automated rollback for blockchain upgrades
 */

import type { DID } from "../../core/agents/identity/did.js";
import {
  type Deployment,
  type DeploymentStage,
  type StageMetrics,
  STAGE_CONFIGS,
  STAGE_ORDER,
  canRollback,
} from "./pipeline.js";

// ============ Rollback Triggers ============

/**
 * Automatic rollback trigger conditions.
 */
export interface RollbackTrigger {
  triggerId: string;
  name: string;
  description: string;
  severity: "critical" | "high" | "medium";
  autoRollback: boolean; // Trigger automatic rollback
  checkFn: (
    metrics: StageMetrics,
    stageConfig: typeof STAGE_CONFIGS.devnet,
  ) => boolean;
}

/**
 * Automatic rollback triggers.
 */
export const ROLLBACK_TRIGGERS: RollbackTrigger[] = [
  {
    triggerId: "TRIG-001",
    name: "Consensus Failure Threshold",
    description: "Too many consensus failures detected",
    severity: "critical",
    autoRollback: true,
    checkFn: (metrics, config) =>
      metrics.consensusFailures >
      config.successMetrics.maxConsensusFailures * 3,
  },
  {
    triggerId: "TRIG-002",
    name: "State Corruption",
    description: "Any state corruption detected",
    severity: "critical",
    autoRollback: true,
    checkFn: (metrics, _config) => metrics.stateCorruptions > 0,
  },
  {
    triggerId: "TRIG-003",
    name: "Severe Performance Degradation",
    description: "Performance degraded by more than 50%",
    severity: "critical",
    autoRollback: true,
    checkFn: (metrics, _config) => metrics.performanceRatio < 0.5,
  },
  {
    triggerId: "TRIG-004",
    name: "Uptime Critical",
    description: "Uptime dropped below 95%",
    severity: "critical",
    autoRollback: true,
    checkFn: (metrics, _config) => metrics.uptime < 0.95,
  },
  {
    triggerId: "TRIG-005",
    name: "Multiple Incidents",
    description: "More than 5 incidents in current stage",
    severity: "high",
    autoRollback: false, // Requires manual decision
    checkFn: (metrics, _config) => metrics.incidentsDetected > 5,
  },
];

// ============ Rollback State ============

/**
 * Rollback request.
 */
export interface RollbackRequest {
  requestId: string;
  deploymentId: string;
  requestedBy: DID | "system";
  reason: string;
  triggerId?: string; // If triggered automatically
  severity: "critical" | "high" | "medium";
  requestedAt: bigint;
  status: "pending" | "approved" | "rejected" | "executed";
  targetStage: DeploymentStage;
  evidence: RollbackEvidence[];
}

/**
 * Evidence supporting a rollback.
 */
export interface RollbackEvidence {
  type: "metric" | "incident" | "manual";
  description: string;
  value?: string;
  timestamp: bigint;
}

/**
 * Rollback execution result.
 */
export interface RollbackResult {
  success: boolean;
  deploymentId: string;
  fromStage: DeploymentStage;
  toStage: DeploymentStage;
  executedAt: bigint;
  error?: string;
  postRollbackActions: string[];
}

// ============ Rollback Operations ============

/**
 * Check if any automatic triggers are met.
 */
export function checkRollbackTriggers(
  deployment: Deployment,
  currentBlock: bigint,
): { triggered: boolean; triggers: RollbackTrigger[] } {
  const currentRecord =
    deployment.stageHistory[deployment.stageHistory.length - 1];
  if (!currentRecord) {
    return { triggered: false, triggers: [] };
  }

  const stageConfig = STAGE_CONFIGS[deployment.currentStage];
  const triggeredList: RollbackTrigger[] = [];

  for (const trigger of ROLLBACK_TRIGGERS) {
    if (trigger.checkFn(currentRecord.metrics, stageConfig)) {
      triggeredList.push(trigger);
    }
  }

  const hasAutoTrigger = triggeredList.some((t) => t.autoRollback);

  return { triggered: hasAutoTrigger, triggers: triggeredList };
}

/**
 * Create a rollback request.
 */
export function createRollbackRequest(
  deployment: Deployment,
  requestedBy: DID | "system",
  reason: string,
  evidence: RollbackEvidence[],
  currentBlock: bigint,
  triggerId?: string,
): RollbackRequest {
  const trigger = triggerId
    ? ROLLBACK_TRIGGERS.find((t) => t.triggerId === triggerId)
    : undefined;

  // Determine target stage (previous stage)
  const currentIndex = STAGE_ORDER.indexOf(deployment.currentStage);
  const targetStage =
    currentIndex > 1 ? STAGE_ORDER[currentIndex - 1] : "pending";

  return {
    requestId: `rollback-${deployment.deploymentId}-${currentBlock}`,
    deploymentId: deployment.deploymentId,
    requestedBy,
    reason,
    triggerId,
    severity: trigger?.severity || "medium",
    requestedAt: currentBlock,
    status: "pending",
    targetStage: targetStage as DeploymentStage,
    evidence,
  };
}

/**
 * Approve a rollback request.
 */
export function approveRollbackRequest(
  request: RollbackRequest,
  approver: DID,
  currentBlock: bigint,
): RollbackRequest {
  return {
    ...request,
    status: "approved",
  };
}

/**
 * Execute a rollback.
 */
export function executeRollback(
  deployment: Deployment,
  request: RollbackRequest,
  currentBlock: bigint,
): { deployment: Deployment; result: RollbackResult } {
  const fromStage = deployment.currentStage;
  const toStage = request.targetStage;

  // Validate rollback is allowed
  if (
    !canRollback(deployment, currentBlock) &&
    request.requestedBy !== "system"
  ) {
    return {
      deployment,
      result: {
        success: false,
        deploymentId: deployment.deploymentId,
        fromStage,
        toStage,
        executedAt: currentBlock,
        error: "Rollback window has expired",
        postRollbackActions: [],
      },
    };
  }

  // Close current stage
  const currentRecord =
    deployment.stageHistory[deployment.stageHistory.length - 1];
  if (currentRecord) {
    currentRecord.exitedAt = currentBlock;
    currentRecord.outcome = "rolled_back";
    currentRecord.notes = `Rolled back: ${request.reason}`;
  }

  // Determine post-rollback actions
  const postRollbackActions = generatePostRollbackActions(
    fromStage,
    toStage,
    request.severity,
  );

  // Update deployment
  const updatedDeployment: Deployment = {
    ...deployment,
    currentStage: toStage,
    stageHistory: [
      ...deployment.stageHistory,
      {
        stage: toStage,
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
        notes: `Rolled back from ${fromStage}: ${request.reason}`,
      },
    ],
    metrics: {
      ...deployment.metrics,
      rollbackCount: deployment.metrics.rollbackCount + 1,
    },
    updatedAt: currentBlock,
  };

  return {
    deployment: updatedDeployment,
    result: {
      success: true,
      deploymentId: deployment.deploymentId,
      fromStage,
      toStage,
      executedAt: currentBlock,
      postRollbackActions,
    },
  };
}

/**
 * Execute emergency rollback (bypasses approval).
 */
export function executeEmergencyRollback(
  deployment: Deployment,
  reason: string,
  evidence: RollbackEvidence[],
  currentBlock: bigint,
): { deployment: Deployment; result: RollbackResult } {
  const request = createRollbackRequest(
    deployment,
    "system",
    `EMERGENCY: ${reason}`,
    evidence,
    currentBlock,
  );

  request.status = "approved";
  request.severity = "critical";

  return executeRollback(deployment, request, currentBlock);
}

/**
 * Generate post-rollback actions based on severity.
 */
function generatePostRollbackActions(
  fromStage: DeploymentStage,
  toStage: DeploymentStage,
  severity: "critical" | "high" | "medium",
): string[] {
  const actions: string[] = [];

  // Always
  actions.push("Notify all stakeholders of rollback");
  actions.push("Generate incident report");

  if (severity === "critical") {
    actions.push("URGENT: Investigate root cause immediately");
    actions.push("Pause all other deployments");
    actions.push("Page on-call human reviewers");
    actions.push("Begin post-mortem process");
  }

  if (severity === "high") {
    actions.push("Investigate root cause within 24 hours");
    actions.push("Review deployment approval process");
  }

  // Stage-specific
  if (
    fromStage === "mainnet_100" ||
    fromStage === "mainnet_50" ||
    fromStage === "mainnet_25"
  ) {
    actions.push("Verify mainnet state consistency");
    actions.push("Monitor for lingering effects");
    actions.push("Communicate to node operators");
  }

  if (fromStage === "canary") {
    actions.push("Review canary metrics collection");
    actions.push("Improve monitoring thresholds");
  }

  return actions;
}

/**
 * Rollback to a specific previous stage.
 */
export function rollbackToStage(
  deployment: Deployment,
  targetStage: DeploymentStage,
  reason: string,
  requestedBy: DID | "system",
  currentBlock: bigint,
): { deployment: Deployment; result: RollbackResult } | { error: string } {
  const currentIndex = STAGE_ORDER.indexOf(deployment.currentStage);
  const targetIndex = STAGE_ORDER.indexOf(targetStage);

  if (targetIndex >= currentIndex) {
    return { error: "Can only rollback to previous stages" };
  }

  if (targetIndex < 0) {
    return { error: "Invalid target stage" };
  }

  const request = createRollbackRequest(
    deployment,
    requestedBy,
    reason,
    [
      {
        type: "manual",
        description: reason,
        timestamp: currentBlock,
      },
    ],
    currentBlock,
  );

  request.targetStage = targetStage;
  request.status = "approved";

  return executeRollback(deployment, request, currentBlock);
}

// ============ Incident Management ============

/**
 * Incident during deployment.
 */
export interface DeploymentIncident {
  incidentId: string;
  deploymentId: string;
  stage: DeploymentStage;
  severity: "critical" | "high" | "medium" | "low";
  type: "consensus" | "state" | "performance" | "security" | "other";
  description: string;
  detectedAt: bigint;
  resolvedAt?: bigint;
  resolution?: string;
  triggerRollback: boolean;
}

/**
 * Record an incident.
 */
export function recordIncident(
  deployment: Deployment,
  incident: Omit<DeploymentIncident, "incidentId" | "deploymentId" | "stage">,
  currentBlock: bigint,
): {
  deployment: Deployment;
  incident: DeploymentIncident;
  shouldRollback: boolean;
} {
  const fullIncident: DeploymentIncident = {
    ...incident,
    incidentId: `incident-${deployment.deploymentId}-${currentBlock}`,
    deploymentId: deployment.deploymentId,
    stage: deployment.currentStage,
  };

  // Update metrics
  const updatedDeployment = {
    ...deployment,
    metrics: {
      ...deployment.metrics,
      totalIncidents: deployment.metrics.totalIncidents + 1,
    },
  };

  // Check if this should trigger rollback
  const shouldRollback =
    incident.triggerRollback ||
    incident.severity === "critical" ||
    (incident.severity === "high" && incident.type === "state");

  return {
    deployment: updatedDeployment,
    incident: fullIncident,
    shouldRollback,
  };
}

// ============ Rollback History ============

/**
 * Rollback history for analysis.
 */
export interface RollbackHistory {
  deploymentId: string;
  rollbacks: Array<{
    requestId: string;
    fromStage: DeploymentStage;
    toStage: DeploymentStage;
    reason: string;
    severity: "critical" | "high" | "medium";
    executedAt: bigint;
  }>;
  totalRollbacks: number;
  mostCommonTrigger: string | null;
}

/**
 * Get rollback history for a deployment.
 */
export function getRollbackHistory(deployment: Deployment): RollbackHistory {
  const rollbacks: RollbackHistory["rollbacks"] = [];

  for (let i = 0; i < deployment.stageHistory.length; i++) {
    const record = deployment.stageHistory[i];
    if (record.outcome === "rolled_back") {
      const nextRecord = deployment.stageHistory[i + 1];
      if (nextRecord) {
        rollbacks.push({
          requestId: `rollback-${i}`,
          fromStage: record.stage,
          toStage: nextRecord.stage,
          reason: record.notes,
          severity: "medium", // Would need to track this
          executedAt: record.exitedAt!,
        });
      }
    }
  }

  return {
    deploymentId: deployment.deploymentId,
    rollbacks,
    totalRollbacks: rollbacks.length,
    mostCommonTrigger: null, // Would need trigger tracking
  };
}

/**
 * Get rollback risk score for current stage.
 */
export function getRollbackRisk(deployment: Deployment): {
  risk: "low" | "medium" | "high";
  factors: string[];
} {
  const factors: string[] = [];
  let riskScore = 0;

  // Previous rollbacks increase risk
  if (deployment.metrics.rollbackCount > 0) {
    riskScore += deployment.metrics.rollbackCount * 2;
    factors.push(`${deployment.metrics.rollbackCount} previous rollback(s)`);
  }

  // Incidents increase risk
  if (deployment.metrics.totalIncidents > 3) {
    riskScore += 2;
    factors.push(`${deployment.metrics.totalIncidents} incidents recorded`);
  }

  // Low uptime increases risk
  if (deployment.metrics.averageUptime < 0.999) {
    riskScore += 1;
    factors.push(
      `Average uptime ${(deployment.metrics.averageUptime * 100).toFixed(2)}%`,
    );
  }

  // Later stages have higher baseline risk
  const stageIndex = STAGE_ORDER.indexOf(deployment.currentStage);
  if (stageIndex >= 4) {
    riskScore += 1;
    factors.push("Mainnet stage");
  }

  let risk: "low" | "medium" | "high";
  if (riskScore >= 5) {
    risk = "high";
  } else if (riskScore >= 2) {
    risk = "medium";
  } else {
    risk = "low";
  }

  return { risk, factors };
}
