/**
 * Halt Executor
 *
 * Executes the halt action after a halt ceremony is finalized:
 * 1. Sets emergency status to "halted"
 * 2. Records halt block for revert depth validation
 * 3. Appends audit entry
 */

import type { StateManager } from "../../state/state_manager.js";
import type { EmergencyCeremony, EmergencyHaltProposal } from "./types.js";

export interface HaltExecutionResult {
  success: boolean;
  error?: string;
  haltBlock: bigint;
}

/**
 * Execute a network halt after ceremony finalization.
 *
 * Note: Engine stop and snapshot creation are handled at the engine integration
 * level since executors operate at the state layer without engine access.
 */
export function executeHalt(
  state: StateManager,
  ceremony: EmergencyCeremony<EmergencyHaltProposal>,
  blockNumber: bigint,
): HaltExecutionResult {
  // Set halted status
  state.setEmergencyStatus("halted");
  state.setLastHaltBlock(blockNumber);

  // Audit log
  state.appendEmergencyAuditEntry({
    timestamp: Date.now(),
    blockNumber,
    action: "halt_executed",
    actor: ceremony.proposal.proposer,
    ceremonyId: ceremony.id,
    details: `Network halted: ${ceremony.proposal.reason}`,
  });

  return {
    success: true,
    haltBlock: blockNumber,
  };
}
