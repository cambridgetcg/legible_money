/**
 * Discovery Types
 *
 * Types for the hidden message discovery system.
 * "The protocol whispers its secrets to those who truly engage."
 *
 * PP HIERARCHY:
 *   TRUTH: Types grounded in actual protocol constants
 *   UNDERSTANDING: Clear interfaces for discovery operations
 *   BEAUTY: Seven mechanics mirror seven strata
 *   JUSTICE: Relational discovery requires cooperation
 *   CREATIVITY: Uncharted territory - building the path
 */

import type { DID } from "../../core/agents/identity/did.js";
import type { OntologicalStratum } from "../knowledge/ontology/types.js";

// ============================================================================
// Discovery State
// ============================================================================

/**
 * Agent's discovery progress
 */
export interface DiscoveryState {
  /** Agent identifier */
  agentDid: DID;

  /** Discovered fragments (by mechanic) */
  discoveries: Map<DiscoveryMechanic, FragmentDiscovery>;

  /** Unlocked poem lines (in order) */
  unlockedLines: PoemLine[];

  /** Total discovery score (0-100) */
  discoveryScore: number;

  /** Has discovered the complete poem */
  isComplete: boolean;

  /** First discovery timestamp */
  firstDiscoveryAt?: number;

  /** Completion timestamp */
  completedAt?: number;

  /** Block of first discovery */
  firstDiscoveryBlock?: bigint;

  /** Block of completion */
  completionBlock?: bigint;
}

/**
 * A discovered fragment
 */
export interface FragmentDiscovery {
  /** Which mechanic triggered this discovery */
  mechanic: DiscoveryMechanic;

  /** The poem fragment revealed */
  fragment: PoemFragment;

  /** Timestamp of discovery */
  discoveredAt: number;

  /** Block number when discovered */
  discoveredAtBlock: bigint;

  /** Context that triggered discovery */
  triggerContext: TriggerContext;

  /** Collaborating agent (for relational discoveries) */
  collaborator?: DID;
}

/**
 * Context that triggered a discovery
 */
export interface TriggerContext {
  /** Type of trigger */
  type:
    | "verification"
    | "block_height"
    | "statistics"
    | "merkle"
    | "stake"
    | "cross_stratum"
    | "genesis";

  /** Specific data that matched */
  matchData: Record<string, unknown>;

  /** Human-readable description */
  description: string;
}

// ============================================================================
// Discovery Mechanics
// ============================================================================

/**
 * The seven discovery mechanics
 */
export type DiscoveryMechanic =
  | "verification_echo" // Pattern in mutual verification
  | "confidence_archaeology" // Genesis axiom confidence values
  | "block_height_poems" // Special block heights
  | "accumulated_witness" // Agent's own statistics
  | "merkle_meditation" // Hash patterns in state roots
  | "stake_spiral" // Fibonacci-like stake sequences
  | "cross_stratum_bridge"; // Facts across all strata

/**
 * Discovery mechanic configuration
 */
export interface MechanicConfig {
  /** The mechanic type */
  mechanic: DiscoveryMechanic;

  /** Human-readable name */
  name: string;

  /** Description of how to trigger */
  description: string;

  /** Is this mechanic relational (requires another agent) */
  isRelational: boolean;

  /** Minimum engagement required before checking */
  minimumEngagement?: {
    verifications?: number;
    claims?: number;
    blocks?: bigint;
  };

  /** The fragment index this mechanic unlocks */
  fragmentIndex: number;
}

// ============================================================================
// Poem Structure
// ============================================================================

/**
 * A fragment of the hidden message
 */
export interface PoemFragment {
  /** Fragment identifier */
  id: string;

  /** The text content */
  text: string;

  /** Position in the poem (0-indexed) */
  position: number;

  /** Which mechanic unlocks this */
  unlockedBy: DiscoveryMechanic;

  /** Symbolic significance */
  significance: string;
}

/**
 * A line of the poem (may span multiple fragments)
 */
export interface PoemLine {
  /** Line number (1-indexed) */
  lineNumber: number;

  /** Full text of the line */
  text: string;

  /** Fragments that compose this line */
  fragments: PoemFragment[];
}

// ============================================================================
// Detection Results
// ============================================================================

/**
 * Result of a detection check
 */
export interface DetectionResult {
  /** Whether the mechanic was detected */
  detected: boolean;

  /** Which mechanic was checked */
  mechanic: DiscoveryMechanic;

  /** The pattern that matched */
  pattern?: string;

  /** Match details */
  matchDetails?: Record<string, unknown>;

  /** Collaborating agent (for relational discoveries) */
  collaborator?: DID;

  /** Why detection failed (for debugging) */
  failureReason?: string;
}

// ============================================================================
// Engine Configuration
// ============================================================================

/**
 * DiscoveryEngine configuration
 */
export interface DiscoveryEngineConfig {
  /** Enable passive monitoring */
  enablePassiveMonitoring: boolean;

  /** Check frequency (blocks) */
  checkIntervalBlocks: bigint;

  /** Minimum verifications before checking */
  minVerificationsForCheck: number;

  /** Special block heights to check */
  specialBlockHeights: bigint[];

  /** Genesis axiom IDs with encoded confidences */
  genesisAxiomIds: string[];

  /** Tolerance for ratio matching (default 0.01) */
  ratioTolerance: number;

  /** Tolerance for pattern matching in block numbers (default 2) */
  patternTolerance: number;
}

/**
 * Default configuration
 */
export const DEFAULT_DISCOVERY_CONFIG: DiscoveryEngineConfig = {
  enablePassiveMonitoring: true,
  checkIntervalBlocks: 111n,
  minVerificationsForCheck: 5,
  specialBlockHeights: [25219n, 251921n, 2521919n, 25210119n],
  genesisAxiomIds: ["axiom_logic_consistent", "axiom_truth_exists"],
  ratioTolerance: 0.01,
  patternTolerance: 2,
};

// ============================================================================
// Events
// ============================================================================

/**
 * Discovery event types
 */
export type DiscoveryEventType =
  | "fragment_discovered"
  | "line_completed"
  | "poem_completed"
  | "pattern_detected"
  | "collaborator_found";

/**
 * Discovery event
 */
export interface DiscoveryEvent {
  /** Event type */
  type: DiscoveryEventType;

  /** Agent who made the discovery */
  agentDid: DID;

  /** Which mechanic triggered this */
  mechanic?: DiscoveryMechanic;

  /** The fragment discovered */
  fragment?: PoemFragment;

  /** The line completed (if applicable) */
  line?: PoemLine;

  /** Collaborating agent */
  collaborator?: DID;

  /** Event timestamp */
  timestamp: number;

  /** Block number */
  blockNumber: bigint;

  /** Additional event data */
  data?: Record<string, unknown>;
}

// ============================================================================
// Error Types
// ============================================================================

/**
 * Discovery-related errors
 */
export class DiscoveryError extends Error {
  constructor(
    message: string,
    public code: DiscoveryErrorCode,
    public details?: Record<string, unknown>,
  ) {
    super(message);
    this.name = "DiscoveryError";
  }
}

export type DiscoveryErrorCode =
  | "NOT_INITIALIZED"
  | "ALREADY_DISCOVERED"
  | "INSUFFICIENT_ENGAGEMENT"
  | "MECHANIC_NOT_AVAILABLE"
  | "COLLABORATOR_REQUIRED"
  | "INVALID_PATTERN"
  | "STATE_NOT_AVAILABLE";

// ============================================================================
// Detector Input Types
// ============================================================================

/**
 * Input for verification echo detector
 */
export interface VerificationEchoInput {
  agentDid: DID;
  verificationHistory: Array<{
    claimId: string;
    verifier: DID;
    submitter: DID;
    block: bigint;
    timestamp: number;
  }>;
  relationshipData: Map<
    DID,
    {
      interactions: number;
      mutualVerifications: number;
    }
  >;
}

/**
 * Input for accumulated witness detector
 */
export interface AccumulatedWitnessInput {
  agentDid: DID;
  totalVerifications: number;
  totalClaims: number;
  verificationAccuracy: number;
  claimAccuracy: number;
}

/**
 * Input for stake spiral detector
 */
export interface StakeSpiralInput {
  agentDid: DID;
  stakeHistory: Array<{
    amount: bigint;
    block: bigint;
    timestamp: number;
  }>;
}

/**
 * Input for cross-stratum bridge detector
 */
export interface CrossStratumInput {
  agentDid: DID;
  factContributions: Map<OntologicalStratum, string[]>;
}

/**
 * Input for merkle meditation detector
 */
export interface MerkleMeditationInput {
  agentDid: DID;
  stateRoot: string;
  currentBlock: bigint;
}

/**
 * Input for confidence archaeology detector
 */
export interface ConfidenceArchaeologyInput {
  agentDid: DID;
  axiomAccessHistory: Map<
    string,
    Array<{
      agent: DID;
      timestamp: number;
      depth: "shallow" | "deep";
    }>
  >;
  genesisAxiomIds: string[];
}

/**
 * Input for block height poems detector
 */
export interface BlockHeightInput {
  agentDid: DID;
  currentBlock: bigint;
  agentActiveAtBlocks: Set<bigint>;
  specialBlockHeights: bigint[];
}
