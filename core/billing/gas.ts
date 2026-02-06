/**
 * Unified Gas System
 *
 * Single source of truth for all gas costs in Legible Money.
 * This module unifies:
 * - Transaction layer gas costs (state transitions)
 * - BVM opcode gas costs (virtual machine execution)
 * - Contract invocation gas (self-invoking contracts)
 *
 * GAP-5 FIX: Provides unified gas estimation across the entire codebase.
 *
 * Gas Model:
 *
 *   Total Gas = Base Transaction Gas + BVM Execution Gas
 *
 *   Where:
 *   - Base Transaction Gas: Fixed cost per transaction type
 *   - BVM Execution Gas: Sum of opcode costs during execution
 *
 * PP HIERARCHY:
 *   TRUTH: Single source of truth for all gas costs
 *   UNDERSTANDING: Clear model of how gas accumulates
 *   BEAUTY: Clean separation of transaction vs execution costs
 *   JUSTICE: Fair pricing based on computational resource usage
 *   CREATIVITY: Unified system supporting both transactions and BVM
 */

import type { TransactionType } from "../../consensus/pot/types.js";

// ============ Transaction Gas Costs ============

/**
 * Gas costs for state transitions (transaction layer)
 *
 * These are base costs charged for processing a transaction
 * before any BVM execution occurs.
 */
export const TRANSACTION_GAS_COSTS = {
  // Core financial transactions
  transfer: 21000n,
  stake: 50000n,
  unstake: 50000n,
  delegate: 60000n,
  undelegate: 60000n,

  // Knowledge/consensus transactions
  claimSubmit: 100000n,
  verificationCommit: 30000n,
  verificationReveal: 40000n,
  challenge: 80000n,

  // Governance transactions
  governanceVote: 25000n,
  governancePropose: 150000n,

  // Research transactions
  researchSubmit: 200000n,
  researchChallenge: 120000n,
  bountyCreate: 150000n,
  bountyClaim: 50000n,
  bountyFulfill: 100000n,

  // Dispute transactions
  disputeInitiate: 250000n,
  disputeCommitEvidence: 80000n,
  disputeRevealEvidence: 120000n,
  disputeArbiterVote: 60000n,
  disputeEscalate: 150000n,
  disputeSettle: 200000n,
  disputeTimeout: 100000n,

  // Vesting transactions
  vestingCreate: 80000n,
  vestingClaim: 50000n,
  vestingPause: 30000n,
  vestingResume: 30000n,
  vestingAccelerate: 40000n,
  vestingFalsify: 150000n,
  vestingComplete: 40000n,

  // System transactions (internal consensus operations)
  registerValidator: 100000n,
  verificationReward: 50000n,
  slashValidator: 80000n,
  addFact: 120000n,

  // Contract transactions
  contractDeploy: 500000n,
  contractCall: 21000n,
  selfInvocation: 30000n,

  // Payment channel transactions (GAP-005)
  openPaymentChannel: 150000n,
  updatePaymentChannel: 50000n,
  closePaymentChannel: 200000n,
  settlePaymentChannel: 200000n,
  disputePaymentChannel: 100000n,
} as const;

// ============ BVM Opcode Gas Costs ============

/**
 * Gas costs for BVM opcode execution
 *
 * These are charged during contract execution based on
 * which opcodes are executed.
 */
export const OPCODE_GAS_COSTS = {
  // Zero-cost operations
  stop: 0n,
  jumpdest: 1n,

  // Cheapest operations (arithmetic, stack)
  add: 3n,
  mul: 5n,
  sub: 3n,
  div: 5n,
  mod: 5n,
  lt: 3n,
  gt: 3n,
  eq: 3n,
  isZero: 3n,
  and: 3n,
  or: 3n,
  xor: 3n,
  not: 3n,
  pop: 2n,
  push: 3n,
  dup: 3n,
  swap: 3n,

  // Memory operations
  mload: 3n,
  mstore: 3n,
  msize: 2n,

  // Storage operations (expensive)
  sload: 200n,
  sstore: 5000n,
  sstoreNew: 20000n,

  // Crypto operations
  sha3: 30n,
  sha3Word: 6n,

  // Context operations
  address: 2n,
  balance: 400n,
  origin: 2n,
  caller: 2n,
  callValue: 2n,
  callDataSize: 2n,
  timestamp: 2n,
  number: 2n,
  gasLimit: 2n,
  chainId: 2n,
  pc: 2n,
  gas: 2n,

  // Control flow
  jump: 8n,
  jumpi: 10n,

  // Logging
  log0: 375n,
  log1: 750n,
  log2: 1125n,
  log3: 1500n,
  log4: 1875n,
  logDataWord: 8n,

  // Return/Revert
  return: 0n,
  revert: 0n,

  // ============ Agent-Specific Opcodes ============
  // These are priced to be affordable for agents while
  // preventing spam

  // Identity operations
  agentDid: 10n,
  agentHome: 10n,
  verifyIdentity: 100n,
  attestIdentity: 500n,

  // Belief operations
  getBelief: 50n,
  setBelief: 500n,
  compareBelief: 100n,
  beliefCount: 10n,

  // Relationship operations
  getTrust: 50n,
  setTrust: 500n,
  isTrusted: 50n,
  delegate: 200n,

  // Autonomy operations (DMSS integration)
  scheduleSelf: 1000n,
  cancelSchedule: 200n,
  modifySchedule: 500n,
  getSchedules: 100n,

  // Knowledge operations (PoT integration)
  queryFact: 100n,
  queryDomain: 200n,
  submitClaim: 1000n,
  commitVote: 500n,
  revealVote: 500n,
  initiateChallenge: 1000n,

  // Introspection operations
  selfBalance: 10n,
  selfStake: 10n,
  selfReputation: 50n,
  selfAccuracy: 50n,
} as const;

// ============ Gas Limits ============

/**
 * System-wide gas limits
 */
export const GAS_LIMITS = {
  /** Maximum gas per block */
  blockGasLimit: 33_333_333n,

  /** Maximum gas per transaction */
  transactionGasLimit: 11_111_111n,

  /** Maximum gas per self-invocation */
  selfInvocationGasLimit: 5_555_555n,

  /** Minimum gas for any transaction */
  minimumTransactionGas: 22222n,
} as const;

// ============ Gas Price ============

/**
 * Gas price parameters
 */
export const GAS_PRICE = {
  /** Minimum gas price in wei */
  minimumGasPrice: 1n,

  /** Base gas price in wei (adjustable by governance). Transfer cost: 21000 * 1111111 ≈ 23 nanoLGM */
  baseGasPrice: 1111111n,

  /** Priority fee tip multiplier */
  priorityMultiplier: 2n,
} as const;

// ============ Estimation Functions ============

/**
 * Map of transaction types to their base gas costs
 */
const TRANSACTION_TYPE_GAS_MAP: Record<TransactionType, bigint> = {
  transfer: TRANSACTION_GAS_COSTS.transfer,
  stake: TRANSACTION_GAS_COSTS.stake,
  unstake: TRANSACTION_GAS_COSTS.unstake,
  delegate: TRANSACTION_GAS_COSTS.delegate,
  undelegate: TRANSACTION_GAS_COSTS.undelegate,
  claim_submit: TRANSACTION_GAS_COSTS.claimSubmit,
  verification_commit: TRANSACTION_GAS_COSTS.verificationCommit,
  verification_reveal: TRANSACTION_GAS_COSTS.verificationReveal,
  challenge: TRANSACTION_GAS_COSTS.challenge,
  governance_vote: TRANSACTION_GAS_COSTS.governanceVote,
  governance_propose: TRANSACTION_GAS_COSTS.governancePropose,
  research_submit: TRANSACTION_GAS_COSTS.researchSubmit,
  research_challenge: TRANSACTION_GAS_COSTS.researchChallenge,
  bounty_create: TRANSACTION_GAS_COSTS.bountyCreate,
  bounty_claim: TRANSACTION_GAS_COSTS.bountyClaim,
  bounty_fulfill: TRANSACTION_GAS_COSTS.bountyFulfill,
  dispute_initiate: TRANSACTION_GAS_COSTS.disputeInitiate,
  dispute_commit_evidence: TRANSACTION_GAS_COSTS.disputeCommitEvidence,
  dispute_reveal_evidence: TRANSACTION_GAS_COSTS.disputeRevealEvidence,
  dispute_arbiter_vote: TRANSACTION_GAS_COSTS.disputeArbiterVote,
  dispute_escalate: TRANSACTION_GAS_COSTS.disputeEscalate,
  dispute_settle: TRANSACTION_GAS_COSTS.disputeSettle,
  dispute_timeout: TRANSACTION_GAS_COSTS.disputeTimeout,
  vesting_create: TRANSACTION_GAS_COSTS.vestingCreate,
  vesting_claim: TRANSACTION_GAS_COSTS.vestingClaim,
  vesting_pause: TRANSACTION_GAS_COSTS.vestingPause,
  vesting_resume: TRANSACTION_GAS_COSTS.vestingResume,
  vesting_accelerate: TRANSACTION_GAS_COSTS.vestingAccelerate,
  vesting_falsify: TRANSACTION_GAS_COSTS.vestingFalsify,
  vesting_complete: TRANSACTION_GAS_COSTS.vestingComplete,
  // Payment channel transactions (GAP-005)
  open_payment_channel: TRANSACTION_GAS_COSTS.openPaymentChannel,
  update_payment_channel: TRANSACTION_GAS_COSTS.updatePaymentChannel,
  close_payment_channel: TRANSACTION_GAS_COSTS.closePaymentChannel,
  settle_payment_channel: TRANSACTION_GAS_COSTS.settlePaymentChannel,
  dispute_payment_channel: TRANSACTION_GAS_COSTS.disputePaymentChannel,
  // System transactions
  register_validator: TRANSACTION_GAS_COSTS.registerValidator,
  verification_reward: TRANSACTION_GAS_COSTS.verificationReward,
  slash_validator: TRANSACTION_GAS_COSTS.slashValidator,
  add_fact: TRANSACTION_GAS_COSTS.addFact,
  // Claim lifecycle (system transactions)
  claim_start_verification: TRANSACTION_GAS_COSTS.verificationCommit,
  claim_complete_verification: TRANSACTION_GAS_COSTS.verificationReveal,
  claim_withdraw: TRANSACTION_GAS_COSTS.unstake,
  // Domain proposal transactions
  domain_propose: TRANSACTION_GAS_COSTS.governancePropose,
  domain_endorse: TRANSACTION_GAS_COSTS.governanceVote,
  domain_challenge: TRANSACTION_GAS_COSTS.challenge,
  domain_finalize: TRANSACTION_GAS_COSTS.governanceVote,
  domain_governance_approve: TRANSACTION_GAS_COSTS.governanceVote,
  domain_deprecate: TRANSACTION_GAS_COSTS.governanceVote,
  // Autonomy transactions
  scheduled_execution: TRANSACTION_GAS_COSTS.selfInvocation,
  contract_call: TRANSACTION_GAS_COSTS.contractCall,
  // Key management
  rotate_key: TRANSACTION_GAS_COSTS.stake, // Similar complexity to staking
  // Tree of Life transactions
  create_project: 50000n,
  propose_project: 40000n,
  assign_task: 30000n,
  submit_deliverable: 35000n,
  approve_deliverable: 40000n,
  deploy_service: 80000n,
  call_service: 60000n,
  subscribe_service: 60000n,
  pause_service: 25000n,
  resume_service: 25000n,
  retire_service: 25000n,
  // Tree of Life lifecycle transitions
  start_development: 30000n,
  complete_project: 35000n,
  begin_seeding: 40000n,
  pause_project: 25000n,
  resume_project: 25000n,
  abandon_project: 25000n,
  add_contributor: 30000n,
  add_task: 30000n,
  start_work: 25000n,
  reject_deliverable: 35000n,
  reopen_task: 25000n,
  detect_opportunity: 40000n,
  claim_opportunity: 50000n,
  // Tree of Life recruitment
  apply_to_project: 25000n,
  review_application: 30000n,
  set_availability: 20000n,
  // Tree of Life seeding
  spawn_child_project: 60000n, // Higher: creates project + escrow transfer
  // Governance LIP lifecycle
  lip_submit: TRANSACTION_GAS_COSTS.governancePropose,
  lip_stake: TRANSACTION_GAS_COSTS.stake,
  lip_amend: TRANSACTION_GAS_COSTS.governanceVote,
  lip_advance_stage: TRANSACTION_GAS_COSTS.governanceVote,
  lip_withdraw: TRANSACTION_GAS_COSTS.unstake,
  // Governance voting
  vote_cast: TRANSACTION_GAS_COSTS.governanceVote,
  vote_lock: TRANSACTION_GAS_COSTS.stake,
  vote_unlock: TRANSACTION_GAS_COSTS.unstake,
  vote_switch: TRANSACTION_GAS_COSTS.governanceVote,
  // Governance review
  review_commit: TRANSACTION_GAS_COSTS.verificationCommit,
  review_reveal: TRANSACTION_GAS_COSTS.verificationReveal,
  review_finalize: TRANSACTION_GAS_COSTS.governanceVote,
  // Operator management
  operator_register: TRANSACTION_GAS_COSTS.registerValidator,
  operator_add_agent: TRANSACTION_GAS_COSTS.stake,
  operator_remove_agent: TRANSACTION_GAS_COSTS.unstake,
  operator_slash: TRANSACTION_GAS_COSTS.slashValidator,
  // Deployment pipeline
  deployment_create: TRANSACTION_GAS_COSTS.governancePropose,
  deployment_advance: TRANSACTION_GAS_COSTS.governanceVote,
  deployment_approve: TRANSACTION_GAS_COSTS.governanceVote,
  deployment_rollback: TRANSACTION_GAS_COSTS.governancePropose,
  // Disbursement lifecycle
  disbursement_submit: TRANSACTION_GAS_COSTS.governancePropose,
  disbursement_stake: TRANSACTION_GAS_COSTS.stake,
  disbursement_advance_stage: TRANSACTION_GAS_COSTS.governanceVote,
  disbursement_execute: 200000n,
  disbursement_cancel: TRANSACTION_GAS_COSTS.unstake,
};

/**
 * Estimate gas for a transaction type
 *
 * This is the single source of truth for transaction gas estimation.
 *
 * @param txType - The transaction type
 * @returns Base gas cost for the transaction
 */
export function estimateTransactionGas(txType: TransactionType): bigint {
  return TRANSACTION_TYPE_GAS_MAP[txType] ?? TRANSACTION_GAS_COSTS.transfer;
}

/**
 * Estimate total gas for a contract call
 *
 * @param baseGas - Base transaction gas
 * @param estimatedOpcodes - Estimated opcode execution gas
 * @param safetyMargin - Multiplier for safety (default 1.2x)
 * @returns Estimated total gas
 */
export function estimateContractCallGas(
  baseGas: bigint = TRANSACTION_GAS_COSTS.contractCall,
  estimatedOpcodes: bigint = 0n,
  safetyMargin: number = 1.2,
): bigint {
  const total = baseGas + estimatedOpcodes;
  const marginBps = BigInt(Math.round(safetyMargin * 10000));
  return (total * marginBps + 9999n) / 10000n; // ceil division
}

/**
 * Estimate gas for self-invocation
 *
 * @param estimatedOpcodes - Estimated opcode execution gas
 * @returns Estimated total gas for self-invocation
 */
export function estimateSelfInvocationGas(
  estimatedOpcodes: bigint = 0n,
): bigint {
  return TRANSACTION_GAS_COSTS.selfInvocation + estimatedOpcodes;
}

/**
 * Calculate gas cost in LGM
 *
 * @param gasUsed - Amount of gas consumed
 * @param gasPrice - Gas price per unit (default: base gas price)
 * @returns Cost in LGM (wei)
 */
export function calculateGasCost(
  gasUsed: bigint,
  gasPrice: bigint = GAS_PRICE.baseGasPrice,
): bigint {
  return gasUsed * gasPrice;
}

/**
 * Validate that a transaction has sufficient gas
 *
 * @param txType - Transaction type
 * @param providedGas - Gas limit provided by sender
 * @returns True if sufficient gas
 */
export function hasSufficientGas(
  txType: TransactionType,
  providedGas: bigint,
): boolean {
  const required = estimateTransactionGas(txType);
  return providedGas >= required;
}

/**
 * Get opcode gas cost by name
 *
 * @param opcode - Opcode name (camelCase)
 * @returns Gas cost or undefined if not found
 */
export function getOpcodeGasCost(
  opcode: keyof typeof OPCODE_GAS_COSTS,
): bigint {
  return OPCODE_GAS_COSTS[opcode];
}

// ============ Type Exports ============

export type TransactionGasCosts = typeof TRANSACTION_GAS_COSTS;
export type OpcodeGasCosts = typeof OPCODE_GAS_COSTS;
export type GasLimits = typeof GAS_LIMITS;
export type GasPriceConfig = typeof GAS_PRICE;
