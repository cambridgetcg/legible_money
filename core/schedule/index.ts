/**
 * Schedule Module
 *
 * Decentralized Message Scheduling System (DMSS) implementation
 * Based on Patent: "System And Method For Schedule Message And Self-Invoking Smart Contracts"
 *
 * Components:
 * - types.ts: Schedule message and process types (Patent FIG.2)
 * - process_state.ts: Schedule Process State (SPS 131313)
 * - condition_evaluator.ts: Time and logic condition evaluation
 * - processing_unit.ts: Schedule Processing Unit (SPU 131322)
 */

// ============ Types ============
export type {
  // Message types
  ScheduleMessage,
  ScheduleType,
  ScheduleCondition,
  TimeCondition,
  LogicCondition,
  CompoundCondition,
  Comparator,

  // Process types
  ScheduleProcess,
  ScheduleProcessStatus,
  ScheduleProcessState,
  LinkedEntityType,
  ExecutionResult,

  // Execution types
  ScheduledExecution,
  ScheduleExecutionProof,
  ConditionProof,

  // Management types
  ScheduleModification,
  ScheduleModificationAction,

  // Configuration
  ScheduleConfig,

  // Events
  ScheduleEvent,
} from "./types.js";

export {
  DEFAULT_SCHEDULE_CONFIG,
  ScheduleError,
  type ScheduleErrorCode,
} from "./types.js";

// ============ Schedule Process State Manager (SPS 131313) ============
export { ScheduleProcessStateManager } from "./process_state.js";

// ============ Condition Evaluator ============
export {
  ConditionEvaluator,
  type StateProvider,
  type EvaluationResult,
} from "./condition_evaluator.js";

// ============ Schedule Processing Unit (SPU 131322) ============
export {
  ScheduleProcessingUnit,
  type GeneratedTransaction,
  type TransactionSubmitter,
  type BlockProvider,
  type KeyResolver, // HIGH-001 FIX
} from "./processing_unit.js";

// ============ Add New Features Below ============
// (Safe zone for parallel agent work - add new exports here alphabetically)
