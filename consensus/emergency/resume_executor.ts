/**
 * Resume Executor
 *
 * Executes network resume after a resume ceremony is finalized:
 * 1. Verifies state root matches expected
 * 2. Sets emergency status to "normal"
 * 3. Appends audit entry
 *
 * Engine restart is handled at the engine integration level.
 */

import type { StateManager } from "../../state/state_manager.js";
import type { EmergencyCeremony, EmergencyResumeProposal } from "./types.js";

export interface ResumeExecutionResult {
  success: boolean;
  error?: string;
  resumeBlock: bigint;
}

/**
 * Execute a network resume after ceremony finalization.
 */
export function executeResume(
  state: StateManager,
  ceremony: EmergencyCeremony<EmergencyResumeProposal>,
  blockNumber: bigint,
): ResumeExecutionResult {
  const proposal = ceremony.proposal;

  // Validate: must be halted
  if (state.getEmergencyStatus() !== "halted") {
    return {
      success: false,
      error: `Cannot resume: network is in '${state.getEmergencyStatus()}' state`,
      resumeBlock: proposal.resumeFromBlock,
    };
  }

  // Verify state root matches expected
  const currentRoot = state.getStateRoot();
  if (proposal.resumeStateRoot !== currentRoot) {
    return {
      success: false,
      error: `State root mismatch: expected ${proposal.resumeStateRoot}, got ${currentRoot}`,
      resumeBlock: proposal.resumeFromBlock,
    };
  }

  // Clear emergency state
  state.setEmergencyStatus("normal");
  state.setHaltSnapshotId(null);

  // Audit log
  state.appendEmergencyAuditEntry({
    timestamp: Date.now(),
    blockNumber,
    action: "resume_executed",
    actor: proposal.proposer,
    ceremonyId: ceremony.id,
    details: `Network resumed from block ${proposal.resumeFromBlock}`,
  });

  return {
    success: true,
    resumeBlock: proposal.resumeFromBlock,
  };
}
