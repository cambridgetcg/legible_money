/**
 * Emergency Kill Switch Module
 *
 * 3-phase BFT emergency protocol for network halt, state revert, and resume.
 * Guardian-tier (tier 3) validators only.
 */

export * from "./types.js";
export * from "./constants.js";
export {
  getGuardianStake,
  isGuardian,
  getGuardianEffectiveStake,
  createHaltCeremony,
  createRevertCeremony,
  createResumeCeremony,
  addPrevote,
  addPrecommit,
  checkCeremonyProgress,
  buildCouncilContext,
} from "./ceremony.js";
export type { CouncilContext } from "./ceremony.js";
export { EmergencyCeremonyManager } from "./manager.js";
export { executeHalt } from "./halt_executor.js";
export type { HaltExecutionResult } from "./halt_executor.js";
export { executeRevert } from "./revert_executor.js";
export type { RevertExecutionResult } from "./revert_executor.js";
export { executeResume } from "./resume_executor.js";
export type { ResumeExecutionResult } from "./resume_executor.js";
