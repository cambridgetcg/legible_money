/**
 * Revert Executor
 *
 * Executes state revert after a revert ceremony is finalized:
 * 1. Validates revert target within MAX_EMERGENCY_REVERT_BLOCKS of halt
 * 2. Sets status to "reverting" during operation
 * 3. Returns to "halted" after revert (resume ceremony required)
 * 4. Appends audit entry
 *
 * Actual snapshot restoration is handled at the engine integration level
 * since the executor operates at the state layer.
 */

import type { StateManager } from "../../state/state_manager.js";
import type { EmergencyCeremony, EmergencyRevertProposal } from "./types.js";
import { EMERGENCY_PARAMS } from "./constants.js";

export interface RevertExecutionResult {
  success: boolean;
  error?: string;
  targetBlock: bigint;
}

/**
 * Execute a state revert after ceremony finalization.
 *
 * This validates the revert parameters and updates state status.
 * The actual snapshot restore is handled by the engine layer.
 */
export function executeRevert(
  state: StateManager,
  ceremony: EmergencyCeremony<EmergencyRevertProposal>,
  currentBlock: bigint,
): RevertExecutionResult {
  const proposal = ceremony.proposal;

  // Validate: must be halted
  if (state.getEmergencyStatus() !== "halted") {
    return {
      success: false,
      error: `Cannot revert: network is in '${state.getEmergencyStatus()}' state`,
      targetBlock: proposal.targetBlockNumber,
    };
  }

  // Validate: revert depth
  const haltBlock = state.getLastHaltBlock();
  if (haltBlock === null) {
    return {
      success: false,
      error: "Cannot revert: no halt block recorded",
      targetBlock: proposal.targetBlockNumber,
    };
  }
  if (
    haltBlock - proposal.targetBlockNumber >
    EMERGENCY_PARAMS.MAX_EMERGENCY_REVERT_BLOCKS
  ) {
    return {
      success: false,
      error: `Revert depth ${haltBlock - proposal.targetBlockNumber} exceeds maximum ${EMERGENCY_PARAMS.MAX_EMERGENCY_REVERT_BLOCKS}`,
      targetBlock: proposal.targetBlockNumber,
    };
  }

  // Validate: target must be in the past
  if (proposal.targetBlockNumber >= currentBlock) {
    return {
      success: false,
      error: "Revert target must be before current block",
      targetBlock: proposal.targetBlockNumber,
    };
  }

  // Set reverting status
  state.setEmergencyStatus("reverting");

  // Audit log
  state.appendEmergencyAuditEntry({
    timestamp: Date.now(),
    blockNumber: currentBlock,
    action: "revert_executed",
    actor: proposal.proposer,
    ceremonyId: ceremony.id,
    details: `State reverted to block ${proposal.targetBlockNumber} (depth: ${currentBlock - proposal.targetBlockNumber})`,
  });

  // After revert completes, status returns to "halted" (resume needed)
  state.setEmergencyStatus("halted");

  return {
    success: true,
    targetBlock: proposal.targetBlockNumber,
  };
}
