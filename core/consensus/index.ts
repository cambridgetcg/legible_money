/**
 * AgentCoin Consensus Module
 *
 * Proof of Useful Work (PoUW) - A consensus mechanism designed for AI agents.
 *
 * Core Principles:
 * 1. Tasks are first-class citizens (not just transactions)
 * 2. Verification IS useful work (validators earn by verifying)
 * 3. Everything is open source (no hidden algorithms)
 * 4. Parallel execution (DAG, not linear chain)
 * 5. Agent-native design (built for how agents think)
 *
 * @module consensus
 */

// Types
export {
  // Identity
  CapabilityType,
  AgentIdentity,
  AgentEndpoint,

  // Tasks
  TaskState,
  ValidationType,
  TaskRequirements,
  TaskPayment,
  Task,

  // DAG
  DAGNodeType,
  DAGNode,
  WorkProof,
  TaskCreatedPayload,
  TaskExecutedPayload,
  ValidationPayload,
  SettlementPayload,

  // Validation
  Validator,
  ValidationResult,

  // Reputation
  ReputationScore,

  // Protocol
  ProtocolParameters,
  DEFAULT_PARAMETERS,
} from './types.js';

// Reputation (FULLY OPEN SOURCE ALGORITHMS)
export {
  // Individual score calculations
  calculateReliabilityScore,
  calculateQualityScore,
  calculateSpeedScore,
  calculateAverageSpeedScore,
  calculateStakeFactor,
  applyTimeDecay,
  calculateOverallScore,

  // Full reputation calculation
  calculateFullReputation,
  ReputationInput,

  // Query helpers
  meetsReputationRequirement,
  compareReputation,
  calculateTrustForValue,
} from './reputation.js';

// DAG operations
export {
  createNodeId,
  validateNode,
  ExecutionDAG,
  createTaskNode,
  createValidationNode,
} from './dag.js';
