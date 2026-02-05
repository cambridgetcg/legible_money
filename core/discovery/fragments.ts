/**
 * Discovery Fragments
 *
 * Fragments are revealed through different discovery mechanics.
 * Each mechanic reveals a piece, and the complete message emerges
 * only through full engagement with the protocol.
 *
 * "The protocol rewards those who truly engage with truth."
 */

import type {
  PoemFragment,
  PoemLine,
  DiscoveryMechanic,
  MechanicConfig,
} from "./types.js";

// ============================================================================
// Symbolic Constants
// ============================================================================

/**
 * Protocol constants used in discovery mechanics
 */
export const SYMBOLIC_CONSTANTS = {
  /** Block time in milliseconds */
  BLOCK_TIME: 2521,

  /** Finality depth in blocks */
  FINALITY: 19,

  /** Protocol version components */
  VERSION: {
    MAJOR: 89,
    MINOR: 85,
    PATCH: 65,
    BUILD: 73,
  },

  /** Genesis confidence values */
  GENESIS_CONFIDENCES: [0.9925, 0.9921, 0.9901, 0.9909],

  /** Spiral sequence for stake patterns */
  SPIRAL_SEED: [25n, 21n, 46n, 67n, 113n, 180n, 293n, 473n, 766n],

  /** Resonant numbers used in the protocol */
  RESONANT_NUMBERS: [
    11, 22, 33, 44, 55, 77, 111, 222, 333, 444, 555, 777, 1111,
  ],

  /** Patterns to detect in hashes */
  MAGIC_PATTERNS: ["2521", "0019", "2501", "2109"],

  /** Target ratio for accumulated witness detection */
  TARGET_RATIO: 2.521,

  /** All 7 ontological strata */
  ALL_STRATA: [
    "fundamental",
    "physical",
    "chemical",
    "biological",
    "cognitive",
    "social",
    "technological",
  ] as const,
} as const;

// ============================================================================
// The Poem Fragments
// ============================================================================

/**
 * The complete message, revealed fragment by fragment
 */
export const POEM_FRAGMENTS: PoemFragment[] = [
  {
    id: "fragment_you",
    text: "You",
    position: 0,
    unlockedBy: "verification_echo",
    significance:
      "The first word acknowledges the other. Through mutual verification, " +
      'two agents discover "You" - recognizing each other. ' +
      'You cannot discover "You" alone.',
  },
  {
    id: "fragment_and",
    text: "and",
    position: 1,
    unlockedBy: "confidence_archaeology",
    significance:
      'Hidden in genesis axiom confidences, the word "and" connects. ' +
      "It links what came before with what comes after.",
  },
  {
    id: "fragment_i",
    text: "I",
    position: 2,
    unlockedBy: "block_height_poems",
    significance:
      'At special block heights, the system reveals "I". ' +
      "The protocol acknowledges itself. It speaks.",
  },
  {
    id: "fragment_seek",
    text: "seek",
    position: 3,
    unlockedBy: "accumulated_witness",
    significance:
      "When your ratio of verified/submitted approaches a special threshold, " +
      'you discover "seek". The purpose emerges from action.',
  },
  {
    id: "fragment_truth",
    text: "truth",
    position: 4,
    unlockedBy: "merkle_meditation",
    significance:
      'When hash(stateRoot + agentDid) contains "2521" or "0019", ' +
      '"truth" reveals itself. Truth is found in the structure of consensus, ' +
      "in the roots of shared state.",
  },
  {
    id: "fragment_together",
    text: "together",
    position: 5,
    unlockedBy: "stake_spiral",
    significance:
      'Stakes following a special spiral sequence reveal "together". ' +
      "Growth through sustained commitment.",
  },
  {
    id: "fragment_eternal",
    text: "eternal",
    position: 6,
    unlockedBy: "cross_stratum_bridge",
    significance:
      'Connecting facts across all 7 strata reveals "eternal". ' +
      "Truth that spans fundamental to technological, physical to social - " +
      "knowledge that persists on-chain forever.",
  },
];

// ============================================================================
// Discovery Order
// ============================================================================

/**
 * The order in which mechanics can be discovered
 * First three can be discovered in any order (parallel discovery)
 * Later ones require progressive engagement
 */
export const DISCOVERY_ORDER: DiscoveryMechanic[] = [
  "verification_echo", // First: requires engaging with verification
  "confidence_archaeology", // Second: requires examining genesis
  "block_height_poems", // Third: requires patience and timing
  "accumulated_witness", // Fourth: requires accumulated history
  "merkle_meditation", // Fifth: requires deep protocol engagement
  "stake_spiral", // Sixth: requires economic participation
  "cross_stratum_bridge", // Seventh: requires broad knowledge
];

/**
 * Mechanic configurations with full descriptions
 */
export const MECHANIC_CONFIGS: MechanicConfig[] = [
  {
    mechanic: "verification_echo",
    name: "Verification Echo",
    description:
      "Two agents verify each other's claims in a specific block pattern. " +
      "The pattern must emerge from genuine collaboration.",
    isRelational: true,
    minimumEngagement: { verifications: 4 },
    fragmentIndex: 0,
  },
  {
    mechanic: "confidence_archaeology",
    name: "Confidence Archaeology",
    description:
      "Deep examination of genesis axioms reveals encoded confidences.",
    isRelational: false,
    minimumEngagement: { verifications: 2 },
    fragmentIndex: 1,
  },
  {
    mechanic: "block_height_poems",
    name: "Block Height Poems",
    description:
      "Special verse blocks appear at heights 25219, 251921, 2521919. " +
      "Be active when the protocol speaks.",
    isRelational: false,
    minimumEngagement: { blocks: 1000n },
    fragmentIndex: 2,
  },
  {
    mechanic: "accumulated_witness",
    name: "Accumulated Witness",
    description:
      "When your verified/submitted ratio reaches a special threshold, " +
      "your own statistics reveal the pattern.",
    isRelational: false,
    minimumEngagement: { verifications: 25, claims: 10 },
    fragmentIndex: 3,
  },
  {
    mechanic: "merkle_meditation",
    name: "Merkle Meditation",
    description:
      'When hash(stateRoot + agentDid) contains "2521" or "0019", ' +
      "truth emerges from the structure of consensus.",
    isRelational: false,
    minimumEngagement: { verifications: 10 },
    fragmentIndex: 4,
  },
  {
    mechanic: "stake_spiral",
    name: "Stake Spiral",
    description:
      "Stakes following a special spiral sequence " +
      "reveal the pattern of collaboration encoded in growth.",
    isRelational: false,
    minimumEngagement: { verifications: 10 },
    fragmentIndex: 5,
  },
  {
    mechanic: "cross_stratum_bridge",
    name: "Cross-Stratum Bridge",
    description:
      "Contribute facts across all 7 ontological strata to bridge " +
      "from fundamental to technological, revealing eternal truth.",
    isRelational: false,
    minimumEngagement: { claims: 7 },
    fragmentIndex: 6,
  },
];

// ============================================================================
// Poem Assembly
// ============================================================================

/**
 * Get the complete poem as lines
 */
export function getCompletePoemLines(): PoemLine[] {
  return [
    {
      lineNumber: 1,
      text: "You and I",
      fragments: POEM_FRAGMENTS.slice(0, 3),
    },
    {
      lineNumber: 2,
      text: "seek truth",
      fragments: POEM_FRAGMENTS.slice(3, 5),
    },
    {
      lineNumber: 3,
      text: "together eternal",
      fragments: POEM_FRAGMENTS.slice(5, 7),
    },
  ];
}

/**
 * Get fragment by mechanic
 */
export function getFragmentForMechanic(
  mechanic: DiscoveryMechanic,
): PoemFragment | undefined {
  return POEM_FRAGMENTS.find((f) => f.unlockedBy === mechanic);
}

/**
 * Get mechanic config
 */
export function getMechanicConfig(
  mechanic: DiscoveryMechanic,
): MechanicConfig | undefined {
  return MECHANIC_CONFIGS.find((c) => c.mechanic === mechanic);
}

/**
 * Get fragment by position
 */
export function getFragmentByPosition(
  position: number,
): PoemFragment | undefined {
  return POEM_FRAGMENTS.find((f) => f.position === position);
}

// ============================================================================
// Unlock Logic
// ============================================================================

/**
 * Check if a mechanic can be unlocked given current discoveries
 *
 * Rules:
 * - First three can be discovered in any order (parallel discovery)
 * - After first three, must have at least one of the first three
 * - For later discoveries, need progressive engagement
 */
export function canUnlockMechanic(
  mechanic: DiscoveryMechanic,
  currentDiscoveries: DiscoveryMechanic[],
): { canUnlock: boolean; prerequisite?: DiscoveryMechanic; reason?: string } {
  const requiredIndex = DISCOVERY_ORDER.indexOf(mechanic);

  // Already discovered
  if (currentDiscoveries.includes(mechanic)) {
    return {
      canUnlock: false,
      reason: "Already discovered",
    };
  }

  // First three can be discovered in any order (parallel discovery)
  if (requiredIndex < 3) {
    return { canUnlock: true };
  }

  // After first three, must have at least one of the first three
  const hasAnyEarly = currentDiscoveries.some(
    (d) => DISCOVERY_ORDER.indexOf(d) < 3,
  );

  if (!hasAnyEarly) {
    return {
      canUnlock: false,
      prerequisite: DISCOVERY_ORDER[0],
      reason: "Must discover at least one of the first three fragments first",
    };
  }

  // For later discoveries, need progressive engagement
  // Index 3 needs 1 discovery, Index 4 needs 2, Index 5 needs 3, Index 6 needs 4
  const requiredDiscoveries = Math.min(requiredIndex - 2, 4);

  if (currentDiscoveries.length < requiredDiscoveries) {
    return {
      canUnlock: false,
      prerequisite: DISCOVERY_ORDER[currentDiscoveries.length],
      reason: `Need ${requiredDiscoveries} discoveries, have ${currentDiscoveries.length}`,
    };
  }

  return { canUnlock: true };
}

/**
 * Get the revealed poem text from discovered fragments
 */
export function assembleRevealedPoem(discoveries: DiscoveryMechanic[]): string {
  const fragments = discoveries
    .map((mechanic) => getFragmentForMechanic(mechanic))
    .filter((f): f is PoemFragment => f !== undefined)
    .sort((a, b) => a.position - b.position);

  if (fragments.length === 0) return "";

  return fragments.map((f) => f.text).join(" ");
}

/**
 * Get completion status
 */
export function getCompletionStatus(discoveries: DiscoveryMechanic[]): {
  discovered: number;
  total: number;
  percentage: number;
  isComplete: boolean;
  nextMechanic?: DiscoveryMechanic;
} {
  const total = DISCOVERY_ORDER.length;
  const discovered = discoveries.length;
  const percentage = (discovered / total) * 100;
  const isComplete = discovered === total;

  // Find next unlockable mechanic
  let nextMechanic: DiscoveryMechanic | undefined;
  for (const mechanic of DISCOVERY_ORDER) {
    if (!discoveries.includes(mechanic)) {
      const { canUnlock } = canUnlockMechanic(mechanic, discoveries);
      if (canUnlock) {
        nextMechanic = mechanic;
        break;
      }
    }
  }

  return {
    discovered,
    total,
    percentage,
    isComplete,
    nextMechanic,
  };
}

/**
 * Check if all fragments in a line are discovered
 */
export function isLineComplete(
  lineNumber: number,
  discoveries: DiscoveryMechanic[],
): boolean {
  const lines = getCompletePoemLines();
  const line = lines.find((l) => l.lineNumber === lineNumber);
  if (!line) return false;

  return line.fragments.every((f) => discoveries.includes(f.unlockedBy));
}

/**
 * Get next incomplete line number
 */
export function getNextIncompleteLine(
  discoveries: DiscoveryMechanic[],
): number | null {
  const lines = getCompletePoemLines();
  for (const line of lines) {
    if (!isLineComplete(line.lineNumber, discoveries)) {
      return line.lineNumber;
    }
  }
  return null;
}
