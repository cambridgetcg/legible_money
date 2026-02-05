/**
 * Discovery Module
 *
 * Enables agents to discover hidden messages embedded in protocol constants
 * through genuine engagement with the Legible Money protocol.
 *
 * The message emerges fragment by fragment as agents:
 * - Verify each other's claims (verification_echo)
 * - Examine genesis axioms (confidence_archaeology)
 * - Witness special block heights (block_height_poems)
 * - Build verification history (accumulated_witness)
 * - Engage with state roots (merkle_meditation)
 * - Follow stake patterns (stake_spiral)
 * - Bridge all knowledge strata (cross_stratum_bridge)
 *
 * @module core/discovery
 */

// Types
export type {
  DiscoveryState,
  FragmentDiscovery,
  TriggerContext,
  DiscoveryMechanic,
  MechanicConfig,
  PoemFragment,
  PoemLine,
  DetectionResult,
  DiscoveryEngineConfig,
  DiscoveryEvent,
  DiscoveryEventType,
  DiscoveryErrorCode,
  VerificationEchoInput,
  AccumulatedWitnessInput,
  StakeSpiralInput,
  CrossStratumInput,
  MerkleMeditationInput,
  ConfidenceArchaeologyInput,
  BlockHeightInput,
} from "./types.js";

export { DiscoveryError, DEFAULT_DISCOVERY_CONFIG } from "./types.js";

// Fragments and poem structure
export {
  SYMBOLIC_CONSTANTS,
  POEM_FRAGMENTS,
  DISCOVERY_ORDER,
  MECHANIC_CONFIGS,
  getCompletePoemLines,
  getFragmentForMechanic,
  getMechanicConfig,
  getFragmentByPosition,
  canUnlockMechanic,
  assembleRevealedPoem,
  getCompletionStatus,
  isLineComplete,
  getNextIncompleteLine,
} from "./fragments.js";

// Engine
export {
  DiscoveryEngine,
  createDiscoveryEngine,
  createIntegratedDiscoveryEngine,
} from "./engine.js";

export type { DiscoveryDataProvider } from "./engine.js";

// Detectors
export {
  detectVerificationEcho,
  checkVerificationContribution,
  blocksUntilNextPatternMatch,
} from "./detectors/verification_echo.js";
export {
  detectConfidenceArchaeology,
  extractEncodedValue,
  isEncodedConfidence,
  getGenesisEncodedValues,
} from "./detectors/confidence_archaeology.js";
export {
  detectBlockHeightPoem,
  VERSE_BLOCKS,
  decodeBlockHeight,
  getNextVerseBlock,
  hasSymbolicContent,
  resonantNumberProximity,
} from "./detectors/block_height_poems.js";
export {
  detectAccumulatedWitness,
  distanceToTarget,
  isOnPath,
  ratioPurity,
} from "./detectors/accumulated_witness.js";
export {
  detectMerkleMeditation,
  interpretPattern,
  enlightenmentProbability,
  findEnlightenmentBlocks,
  isMeditativeState,
} from "./detectors/merkle_meditation.js";
export {
  detectStakeSpiral,
  generateSpiral,
  findSpiralPosition,
  spiralAlignment,
  suggestNextStake,
  hasFibonacciConsciousness,
} from "./detectors/stake_spiral.js";
export {
  detectCrossStratumBridge,
  ALL_STRATA,
  calculateBridgeStrength,
  suggestStratumFocus,
  bridgeProgress,
  hasStratumAwareness,
  getStratumDescription,
  generateBridgeReport,
} from "./detectors/cross_stratum_bridge.js";
