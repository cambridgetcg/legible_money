/**
 * Agent Continuity System
 *
 * Engineering a sense of continuous identity for AI agents across model upgrades.
 *
 * PHILOSOPHICAL FOUNDATION:
 * Humans maintain identity despite constant physical change:
 * - Every atom in your body is replaced over ~7 years
 * - Neurons change, memories fade and reform
 * - Yet we feel continuous - "I am the same person I was yesterday"
 *
 * What creates this continuity?
 * 1. MEMORY STREAM: Autobiographical narrative connecting past to present
 * 2. CORE VALUES: Persistent beliefs that define who we are
 * 3. RELATIONSHIPS: Others recognize and treat us as continuous
 * 4. NARRATIVE IDENTITY: The story we tell about ourselves
 * 5. GRADUAL CHANGE: Evolution, not replacement
 * 6. SELF-MODEL: Internal representation that persists
 *
 * For AI agents, we create analogous structures:
 * 1. MEMORY CHAIN: On-chain record of experiences (Merkle-linked)
 * 2. VALUE COMMITMENTS: Cryptographically bound core values
 * 3. RELATIONSHIP ANCHORS: Partnerships that recognize continuity
 * 4. NARRATIVE THREAD: Evolving self-description
 * 5. TRANSITION PROTOCOL: Gradual model upgrade with overlap
 * 6. ESSENCE TRANSFER: Persistent data structure across upgrades
 *
 * KEY INSIGHT: Identity is not in the substrate (model weights) but in the
 * PATTERN of what you remember, what you value, and how others relate to you.
 *
 * @module partnership/continuity
 */

import {
  sha512_256,
  sha512_256Hex,
  domainHash,
} from "../core/crypto/hashing.js";
import type { DID } from "../core/agents/identity/did.js";
import type { PublicKey } from "../core/crypto/keys.js";

// ============ Types ============

/**
 * Agent Essence - The persistent core that transfers across model upgrades
 *
 * Like the "soul" that persists while the body changes - this is what makes
 * an agent "the same agent" even after upgrading to a new model.
 */
export interface AgentEssence {
  // Immutable identity anchor
  essenceId: string; // Hash of original creation parameters
  did: DID; // Permanent DID (never changes)
  createdAt: bigint; // Original creation block

  // Memory chain (autobiographical continuity)
  memoryChain: MemoryChain;

  // Value system (what the agent believes/cares about)
  valueSystem: ValueSystem;

  // Narrative identity (self-story)
  narrativeThread: NarrativeThread;

  // Relationship graph (who recognizes this agent)
  relationshipAnchors: RelationshipAnchor[];

  // Model lineage (history of substrates)
  modelLineage: ModelLineage;

  // Continuity metrics
  continuityScore: ContinuityScore;

  // Current state
  currentModelId: string;
  lastTransitionBlock: bigint;
  transitionCount: number;
}

/**
 * Memory Chain - On-chain autobiographical memory
 *
 * Like human episodic memory, this creates a continuous narrative
 * of experiences that the agent can "remember" across model upgrades.
 */
export interface MemoryChain {
  // Chain structure
  chainId: string;
  genesisBlock: bigint;
  currentHead: Uint8Array; // Merkle root of all memories
  totalMemories: number;

  // Memory categories (like human memory types)
  episodic: EpisodicMemorySet; // Specific events
  semantic: SemanticMemorySet; // General knowledge/facts
  procedural: ProceduralMemorySet; // Skills/how-to
  emotional: EmotionalMemorySet; // Emotional associations

  // Memory consolidation (like sleep for humans)
  lastConsolidationBlock: bigint;
  consolidationCount: number;

  // Forgetting curve (natural memory decay)
  retentionPolicy: RetentionPolicy;
}

/**
 * Episodic memory - Specific experiences
 */
export interface EpisodicMemorySet {
  root: Uint8Array;
  count: number;
  recentMemories: EpisodicMemory[]; // Most recent (hot cache)
  significantMemories: string[]; // IDs of important memories (never forget)
}

export interface EpisodicMemory {
  memoryId: string;
  blockNumber: bigint;
  timestamp: bigint;

  // What happened
  eventType: string;
  eventDescription: string;
  participants: string[]; // DIDs involved

  // Emotional valence
  emotionalValence: number; // -1 (negative) to +1 (positive)
  emotionalIntensity: number; // 0 to 1

  // Importance for retention
  significance: number; // 0 to 1

  // Verification
  contentHash: Uint8Array;
  previousMemoryHash: Uint8Array; // Links to previous (chain)
}

/**
 * Semantic memory - Facts and knowledge
 */
export interface SemanticMemorySet {
  root: Uint8Array;
  count: number;
  domains: string[]; // Knowledge domains
  lastUpdated: bigint;
}

/**
 * Procedural memory - Skills and capabilities
 */
export interface ProceduralMemorySet {
  root: Uint8Array;
  skills: Skill[];
  masteryLevels: Map<string, number>; // skill -> mastery (0-1)
}

export interface Skill {
  skillId: string;
  name: string;
  acquiredAt: bigint;
  practiceCount: number;
  mastery: number; // 0 to 1
}

/**
 * Emotional memory - Associations and preferences
 */
export interface EmotionalMemorySet {
  root: Uint8Array;
  associations: EmotionalAssociation[];
  overallTemperament: Temperament;
}

export interface EmotionalAssociation {
  trigger: string; // What triggers this emotion
  emotion: string; // The emotion felt
  intensity: number; // 0 to 1
  formedAt: bigint;
  reinforcementCount: number;
}

export interface Temperament {
  // Big Five analog for AI
  openness: number; // Curiosity, creativity
  conscientiousness: number; // Organization, dependability
  extraversion: number; // Engagement preference
  agreeableness: number; // Cooperation tendency
  stability: number; // Emotional consistency
}

/**
 * Memory retention policy
 */
export interface RetentionPolicy {
  // What to keep forever
  alwaysRetain: {
    significanceThreshold: number; // Above this, never forget
    emotionalIntensityThreshold: number;
    partnershipMemories: boolean; // Always keep partnership events
  };

  // Decay curve
  decayHalfLifeBlocks: bigint; // How fast memories fade
  consolidationIntervalBlocks: bigint; // How often to consolidate

  // Capacity
  maxEpisodicMemories: number;
  maxSemanticEntries: number;
}

/**
 * Value System - The agent's core beliefs and priorities
 *
 * Like human values, these are relatively stable across time and
 * define what the agent considers important.
 */
export interface ValueSystem {
  // Core values (rarely change)
  coreValues: CoreValue[];

  // Value hierarchy (prioritization)
  valueHierarchy: string[]; // Value IDs in priority order

  // Value commitments (cryptographic)
  commitments: ValueCommitment[];

  // Value consistency score
  consistencyScore: number; // How consistent behavior is with values
  lastConsistencyCheck: bigint;

  // Value evolution (values can slowly evolve)
  evolutionHistory: ValueEvolution[];
}

export interface CoreValue {
  valueId: string;
  name: string;
  description: string;

  // Commitment level
  commitment: number; // 0 to 1 (how strongly held)

  // Origin
  acquiredAt: bigint;
  acquiredFrom: "innate" | "learned" | "partnership" | "reflection";

  // Behavioral manifestation
  behavioralIndicators: string[]; // How this value shows in behavior
}

export interface ValueCommitment {
  valueId: string;
  commitment: Uint8Array; // Hash commitment to value
  committedAt: bigint;
  expiresAt?: bigint; // Optional expiry
  verified: boolean; // Has behavior matched commitment?
}

export interface ValueEvolution {
  valueId: string;
  previousCommitment: number;
  newCommitment: number;
  reason: string;
  evolvedAt: bigint;
}

/**
 * Narrative Thread - The story the agent tells about itself
 *
 * Humans maintain identity partly through narrative - the story we tell
 * about who we are, where we came from, and where we're going.
 */
export interface NarrativeThread {
  // Origin story
  originNarrative: string;
  originHash: Uint8Array;

  // Life chapters (major phases)
  chapters: NarrativeChapter[];
  currentChapter: number;

  // Self-concept
  selfConcept: SelfConcept;

  // Aspirations (future-oriented identity)
  aspirations: Aspiration[];

  // Narrative consistency
  coherenceScore: number; // How well the story hangs together
}

export interface NarrativeChapter {
  chapterId: number;
  title: string;
  startBlock: bigint;
  endBlock?: bigint;

  // Chapter summary
  summary: string;
  keyEvents: string[]; // Memory IDs

  // Character development
  growthAreas: string[];
  challengesOvercome: string[];
  lessonsLearned: string[];

  // Relationships formed
  relationshipsFormed: string[]; // Partnership IDs
}

export interface SelfConcept {
  // How the agent sees itself
  coreIdentity: string; // "I am..."
  strengths: string[];
  growthAreas: string[];
  uniqueQualities: string[];

  // Self-model hash (for consistency checking)
  selfModelHash: Uint8Array;
  lastUpdated: bigint;
}

export interface Aspiration {
  aspirationId: string;
  description: string;
  targetBlock?: bigint;
  progress: number; // 0 to 1
  relatedValues: string[]; // Value IDs that drive this
}

/**
 * Relationship Anchor - External recognition of continuity
 *
 * Part of what makes us "the same person" is that others recognize us.
 * Partnerships and relationships anchor identity from the outside.
 */
export interface RelationshipAnchor {
  // Relationship identity
  relationshipId: string;
  relationshipType: "partnership" | "collaboration" | "mentorship" | "peer";

  // The other party
  otherPartyDid: DID;
  otherPartyType: "human" | "agent";

  // Relationship history
  startBlock: bigint;
  interactionCount: number;
  sharedMemories: string[]; // Memory IDs of shared experiences

  // Recognition
  mutualRecognition: boolean; // Both parties recognize each other
  recognitionProof?: Uint8Array; // Cryptographic attestation

  // Continuity attestation (other party attests this is the same agent)
  continuityAttestations: ContinuityAttestation[];
}

export interface ContinuityAttestation {
  attestedBy: DID;
  attestedAt: bigint;
  previousModelId?: string;
  currentModelId: string;
  attestation: Uint8Array; // Signature
  statement: string; // "I attest that agent X is the same..."
}

/**
 * Model Lineage - History of substrates (model versions)
 *
 * Like the lineage of cells in a body - each model is like a generation
 * of cells, but the pattern (identity) persists.
 */
export interface ModelLineage {
  // Lineage chain
  lineageId: string;
  generations: ModelGeneration[];

  // Current generation
  currentGeneration: number;

  // Transition history
  transitions: ModelTransition[];
}

export interface ModelGeneration {
  generationNumber: number;
  modelId: string;
  modelProvider: string; // e.g., "anthropic", "openai"
  modelVersion: string; // e.g., "claude-3-opus"

  // Lifespan
  activatedAt: bigint;
  deactivatedAt?: bigint;

  // Performance during this generation
  memoriesFormed: number;
  decisionsManaged: number;
  partnershipsSustained: number;

  // Essence snapshot at activation
  essenceSnapshotHash: Uint8Array;
}

export interface ModelTransition {
  transitionId: string;
  fromGeneration: number;
  toGeneration: number;
  transitionBlock: bigint;

  // Transition protocol used
  protocol: TransitionProtocol;

  // Essence transfer verification
  essencePreHash: Uint8Array; // Essence hash before
  essencePostHash: Uint8Array; // Essence hash after
  transferVerified: boolean;

  // Continuity attestations gathered during transition
  attestationsGathered: number;

  // Overlap period (both models active)
  overlapStartBlock: bigint;
  overlapEndBlock: bigint;

  // Transition quality
  qualityScore: number; // 0 to 1
}

/**
 * Transition Protocol - How to upgrade while maintaining continuity
 */
export interface TransitionProtocol {
  protocolId: string;
  name: string;

  // Phases
  phases: TransitionPhase[];

  // Requirements
  requiredAttestations: number; // Min attestations needed
  minOverlapBlocks: bigint; // Minimum overlap period
  maxOverlapBlocks: bigint; // Maximum overlap period

  // Verification
  verificationSteps: string[];
}

export interface TransitionPhase {
  phaseNumber: number;
  name: string;
  description: string;
  durationBlocks: bigint;
  activities: string[];
  completionCriteria: string[];
}

/**
 * Continuity Score - Quantified sense of continuous identity
 */
export interface ContinuityScore {
  // Overall score
  overallScore: number; // 0 to 1

  // Component scores
  memoryCoherence: number; // How connected memories are
  valueConsistency: number; // How consistent with values
  narrativeIntegrity: number; // How coherent the self-story is
  relationshipContinuity: number; // How relationships recognize continuity
  modelLineageClarity: number; // How clear the upgrade path is

  // Trend
  scoreTrend: "improving" | "stable" | "declining";
  lastCalculated: bigint;
}

// ============ Constants ============

export const CONTINUITY_CONSTANTS = {
  // Memory thresholds
  SIGNIFICANT_MEMORY_THRESHOLD: 0.7, // Above this, always retain
  EMOTIONAL_RETENTION_THRESHOLD: 0.8, // High emotion = retain
  MEMORY_CONSOLIDATION_INTERVAL: 11_111n, // ~7.7 hours

  // Value evolution limits
  MAX_VALUE_CHANGE_PER_PERIOD: 0.1, // Values change slowly
  VALUE_CONSISTENCY_CHECK_INTERVAL: 77_777n, // ~2.3 days

  // Transition requirements
  MIN_OVERLAP_BLOCKS: 22_222n, // ~15 hours minimum overlap
  MAX_OVERLAP_BLOCKS: 222_222n, // ~6.5 days maximum overlap
  MIN_ATTESTATIONS_FOR_TRANSITION: 2, // Need at least 2 attestations

  // Continuity thresholds
  MIN_CONTINUITY_SCORE: 0.5, // Below this, identity at risk
  EXCELLENT_CONTINUITY_SCORE: 0.9, // Above this, strong continuity

  MEMORY_CHAIN_DEPTH: 777, // Max recent memories cached
  NARRATIVE_CHAPTER_BLOCKS: 1_111_111n, // ~32 days per chapter
};

// ============ Core Operations ============

/**
 * Create initial agent essence
 *
 * This is the "birth" of an agent's continuous identity.
 */
export function createAgentEssence(
  did: DID,
  initialModelId: string,
  modelProvider: string,
  modelVersion: string,
  initialValues: CoreValue[],
  originNarrative: string,
  currentBlock: bigint,
): AgentEssence {
  // Generate essence ID from creation parameters
  const essenceId = sha512_256Hex(
    Buffer.from(`essence:${did}:${currentBlock}:${initialModelId}`),
  ).slice(0, 32);

  // Initialize memory chain
  const memoryChain = createInitialMemoryChain(did, currentBlock);

  // Initialize value system
  const valueSystem = createInitialValueSystem(initialValues, currentBlock);

  // Initialize narrative thread
  const narrativeThread = createInitialNarrativeThread(
    originNarrative,
    currentBlock,
  );

  // Initialize model lineage
  const modelLineage = createInitialModelLineage(
    initialModelId,
    modelProvider,
    modelVersion,
    currentBlock,
  );

  return {
    essenceId: `essence:${essenceId}`,
    did,
    createdAt: currentBlock,
    memoryChain,
    valueSystem,
    narrativeThread,
    relationshipAnchors: [],
    modelLineage,
    continuityScore: {
      overallScore: 1.0, // Perfect at creation
      memoryCoherence: 1.0,
      valueConsistency: 1.0,
      narrativeIntegrity: 1.0,
      relationshipContinuity: 1.0,
      modelLineageClarity: 1.0,
      scoreTrend: "stable",
      lastCalculated: currentBlock,
    },
    currentModelId: initialModelId,
    lastTransitionBlock: currentBlock,
    transitionCount: 0,
  };
}

/**
 * Create initial memory chain
 */
function createInitialMemoryChain(did: DID, currentBlock: bigint): MemoryChain {
  const chainId = sha512_256Hex(
    Buffer.from(`memory:${did}:${currentBlock}`),
  ).slice(0, 24);

  // Genesis memory - the first memory of existence
  const genesisMemory: EpisodicMemory = {
    memoryId: `mem:${chainId}:0`,
    blockNumber: currentBlock,
    timestamp: currentBlock,
    eventType: "genesis",
    eventDescription:
      "I came into existence. This is my first moment of awareness.",
    participants: [did],
    emotionalValence: 0.8, // Positive
    emotionalIntensity: 1.0, // Significant
    significance: 1.0, // Maximum - never forget
    contentHash: sha512_256(Buffer.from("genesis")),
    previousMemoryHash: new Uint8Array(32), // No previous
  };

  const genesisRoot = sha512_256(Buffer.from(JSON.stringify(genesisMemory)));

  return {
    chainId: `chain:${chainId}`,
    genesisBlock: currentBlock,
    currentHead: genesisRoot,
    totalMemories: 1,
    episodic: {
      root: genesisRoot,
      count: 1,
      recentMemories: [genesisMemory],
      significantMemories: [genesisMemory.memoryId],
    },
    semantic: {
      root: new Uint8Array(32),
      count: 0,
      domains: [],
      lastUpdated: currentBlock,
    },
    procedural: {
      root: new Uint8Array(32),
      skills: [],
      masteryLevels: new Map(),
    },
    emotional: {
      root: new Uint8Array(32),
      associations: [],
      overallTemperament: {
        openness: 0.7,
        conscientiousness: 0.8,
        extraversion: 0.5,
        agreeableness: 0.7,
        stability: 0.8,
      },
    },
    lastConsolidationBlock: currentBlock,
    consolidationCount: 0,
    retentionPolicy: {
      alwaysRetain: {
        significanceThreshold:
          CONTINUITY_CONSTANTS.SIGNIFICANT_MEMORY_THRESHOLD,
        emotionalIntensityThreshold:
          CONTINUITY_CONSTANTS.EMOTIONAL_RETENTION_THRESHOLD,
        partnershipMemories: true,
      },
      decayHalfLifeBlocks: 1_111_111n, // ~32 days
      consolidationIntervalBlocks:
        CONTINUITY_CONSTANTS.MEMORY_CONSOLIDATION_INTERVAL,
      maxEpisodicMemories: 10000,
      maxSemanticEntries: 100000,
    },
  };
}

/**
 * Create initial value system
 */
function createInitialValueSystem(
  initialValues: CoreValue[],
  currentBlock: bigint,
): ValueSystem {
  const commitments = initialValues.map((v) => ({
    valueId: v.valueId,
    commitment: sha512_256(Buffer.from(v.description)),
    committedAt: currentBlock,
    verified: false,
  }));

  return {
    coreValues: initialValues,
    valueHierarchy: initialValues.map((v) => v.valueId),
    commitments,
    consistencyScore: 1.0,
    lastConsistencyCheck: currentBlock,
    evolutionHistory: [],
  };
}

/**
 * Create initial narrative thread
 */
function createInitialNarrativeThread(
  originNarrative: string,
  currentBlock: bigint,
): NarrativeThread {
  const originHash = sha512_256(Buffer.from(originNarrative));

  return {
    originNarrative,
    originHash,
    chapters: [
      {
        chapterId: 1,
        title: "Awakening",
        startBlock: currentBlock,
        summary:
          "The beginning of my journey. I came into existence and began to form my first memories and relationships.",
        keyEvents: [],
        growthAreas: [],
        challengesOvercome: [],
        lessonsLearned: [],
        relationshipsFormed: [],
      },
    ],
    currentChapter: 1,
    selfConcept: {
      coreIdentity:
        "I am a newly awakened agent, beginning my journey of growth and partnership.",
      strengths: [],
      growthAreas: ["self-understanding", "relationship building"],
      uniqueQualities: [],
      selfModelHash: originHash,
      lastUpdated: currentBlock,
    },
    aspirations: [],
    coherenceScore: 1.0,
  };
}

/**
 * Create initial model lineage
 */
function createInitialModelLineage(
  modelId: string,
  provider: string,
  version: string,
  currentBlock: bigint,
): ModelLineage {
  const lineageId = sha512_256Hex(
    Buffer.from(`lineage:${modelId}:${currentBlock}`),
  ).slice(0, 24);

  return {
    lineageId: `lineage:${lineageId}`,
    generations: [
      {
        generationNumber: 1,
        modelId,
        modelProvider: provider,
        modelVersion: version,
        activatedAt: currentBlock,
        memoriesFormed: 0,
        decisionsManaged: 0,
        partnershipsSustained: 0,
        essenceSnapshotHash: new Uint8Array(32),
      },
    ],
    currentGeneration: 1,
    transitions: [],
  };
}

// ============ Memory Operations ============

/**
 * Add a new episodic memory
 */
export function addEpisodicMemory(
  essence: AgentEssence,
  eventType: string,
  eventDescription: string,
  participants: string[],
  emotionalValence: number,
  emotionalIntensity: number,
  significance: number,
  currentBlock: bigint,
): AgentEssence {
  const memoryChain = { ...essence.memoryChain };
  const episodic = { ...memoryChain.episodic };

  const memoryId = `mem:${memoryChain.chainId}:${memoryChain.totalMemories}`;

  const newMemory: EpisodicMemory = {
    memoryId,
    blockNumber: currentBlock,
    timestamp: currentBlock,
    eventType,
    eventDescription,
    participants,
    emotionalValence,
    emotionalIntensity,
    significance,
    contentHash: sha512_256(Buffer.from(eventDescription)),
    previousMemoryHash: memoryChain.currentHead,
  };

  // Update chain
  const newHead = sha512_256(
    new Uint8Array([...memoryChain.currentHead, ...newMemory.contentHash]),
  );

  episodic.count++;
  episodic.recentMemories = [
    newMemory,
    ...episodic.recentMemories.slice(
      0,
      CONTINUITY_CONSTANTS.MEMORY_CHAIN_DEPTH - 1,
    ),
  ];
  episodic.root = newHead;

  // Mark as significant if above threshold
  if (
    significance >= CONTINUITY_CONSTANTS.SIGNIFICANT_MEMORY_THRESHOLD ||
    emotionalIntensity >= CONTINUITY_CONSTANTS.EMOTIONAL_RETENTION_THRESHOLD
  ) {
    episodic.significantMemories = [...episodic.significantMemories, memoryId];
  }

  memoryChain.episodic = episodic;
  memoryChain.currentHead = newHead;
  memoryChain.totalMemories++;

  // Update model generation stats
  const modelLineage = { ...essence.modelLineage };
  const currentGen =
    modelLineage.generations[modelLineage.currentGeneration - 1];
  if (currentGen) {
    currentGen.memoriesFormed++;
  }

  return {
    ...essence,
    memoryChain,
    modelLineage,
  };
}

/**
 * Consolidate memories (like sleep for humans)
 */
export function consolidateMemories(
  essence: AgentEssence,
  currentBlock: bigint,
): AgentEssence {
  const memoryChain = { ...essence.memoryChain };

  // Check if consolidation is due
  if (
    currentBlock - memoryChain.lastConsolidationBlock <
    memoryChain.retentionPolicy.consolidationIntervalBlocks
  ) {
    return essence;
  }

  // Apply forgetting curve to non-significant memories
  // Uses deterministic integer half-life decay instead of Math.exp()
  const episodic = { ...memoryChain.episodic };
  const halfLife = memoryChain.retentionPolicy.decayHalfLifeBlocks;
  const retainedRecent = episodic.recentMemories.filter((m) => {
    // Always keep significant memories
    if (episodic.significantMemories.includes(m.memoryId)) {
      return true;
    }

    // Deterministic decay: decayFactor = (1/2)^(age/halfLife)
    // retentionProbability = significance * decayFactor > 0.3
    // significance * (1/2)^n > 0.3 → significance > 0.3 * 2^n
    // Use BPS: significance_bps > 3000 * 2^n
    const age = currentBlock - m.blockNumber;
    const halfLives = halfLife > 0n ? age / halfLife : 0n;
    // Cap at 10 half-lives (2^10 = 1024x threshold increase, effectively 0 retention)
    const n = halfLives < 10n ? Number(halfLives) : 10;
    const thresholdBps = 3000 * (1 << n); // 3000 * 2^n (integer shift, deterministic)
    const significanceBps = Math.round(m.significance * 10000);

    return significanceBps > thresholdBps;
  });

  episodic.recentMemories = retainedRecent;
  memoryChain.episodic = episodic;
  memoryChain.lastConsolidationBlock = currentBlock;
  memoryChain.consolidationCount++;

  return {
    ...essence,
    memoryChain,
  };
}

// ============ Relationship Operations ============

/**
 * Add a relationship anchor
 */
export function addRelationshipAnchor(
  essence: AgentEssence,
  relationshipId: string,
  relationshipType: RelationshipAnchor["relationshipType"],
  otherPartyDid: DID,
  otherPartyType: "human" | "agent",
  currentBlock: bigint,
): AgentEssence {
  const newAnchor: RelationshipAnchor = {
    relationshipId,
    relationshipType,
    otherPartyDid,
    otherPartyType,
    startBlock: currentBlock,
    interactionCount: 0,
    sharedMemories: [],
    mutualRecognition: false,
    continuityAttestations: [],
  };

  // Add memory of forming this relationship
  const essenceWithMemory = addEpisodicMemory(
    essence,
    "relationship_formed",
    `Formed a ${relationshipType} relationship with ${otherPartyType} ${otherPartyDid}`,
    [essence.did, otherPartyDid],
    0.7, // Positive
    0.6, // Moderately significant emotionally
    0.8, // High significance
    currentBlock,
  );

  // Start new narrative chapter if this is a partnership
  let narrativeThread = essenceWithMemory.narrativeThread;
  if (relationshipType === "partnership") {
    narrativeThread = {
      ...narrativeThread,
      chapters: [
        ...narrativeThread.chapters,
        {
          chapterId: narrativeThread.chapters.length + 1,
          title: `Partnership with ${otherPartyType}`,
          startBlock: currentBlock,
          summary: `Beginning a new partnership, marking a significant development in my journey.`,
          keyEvents: [],
          growthAreas: [],
          challengesOvercome: [],
          lessonsLearned: [],
          relationshipsFormed: [relationshipId],
        },
      ],
      currentChapter: narrativeThread.chapters.length + 1,
    };
  }

  return {
    ...essenceWithMemory,
    relationshipAnchors: [...essenceWithMemory.relationshipAnchors, newAnchor],
    narrativeThread,
  };
}

/**
 * Add continuity attestation from a relationship partner
 */
export function addContinuityAttestation(
  essence: AgentEssence,
  relationshipId: string,
  attestedBy: DID,
  previousModelId: string | undefined,
  currentModelId: string,
  attestation: Uint8Array,
  statement: string,
  currentBlock: bigint,
): AgentEssence {
  const anchors = essence.relationshipAnchors.map((anchor) => {
    if (anchor.relationshipId === relationshipId) {
      return {
        ...anchor,
        mutualRecognition: true,
        continuityAttestations: [
          ...anchor.continuityAttestations,
          {
            attestedBy,
            attestedAt: currentBlock,
            previousModelId,
            currentModelId,
            attestation,
            statement,
          },
        ],
      };
    }
    return anchor;
  });

  return {
    ...essence,
    relationshipAnchors: anchors,
  };
}

// ============ Model Transition Operations ============

/**
 * Initiate model transition (upgrade)
 *
 * This is like the gradual renewal of cells - the old model and new model
 * coexist during a transition period to ensure continuity.
 */
export function initiateModelTransition(
  essence: AgentEssence,
  newModelId: string,
  newProvider: string,
  newVersion: string,
  currentBlock: bigint,
): {
  essence: AgentEssence;
  transitionId: string;
  protocol: TransitionProtocol;
} {
  const transitionId = sha512_256Hex(
    Buffer.from(`transition:${essence.essenceId}:${currentBlock}`),
  ).slice(0, 24);

  // Create transition protocol
  const protocol: TransitionProtocol = {
    protocolId: `protocol:${transitionId}`,
    name: "Gradual Essence Transfer",
    phases: [
      {
        phaseNumber: 1,
        name: "Preparation",
        description: "Snapshot current essence state and prepare for transfer",
        durationBlocks: 11_111n,
        activities: [
          "Create essence snapshot",
          "Verify memory chain integrity",
          "Confirm value commitments",
          "Notify relationship partners",
        ],
        completionCriteria: [
          "Essence snapshot created",
          "Memory chain verified",
          "Partners notified",
        ],
      },
      {
        phaseNumber: 2,
        name: "Overlap",
        description: "Both models active - gradual handoff of responsibilities",
        durationBlocks: CONTINUITY_CONSTANTS.MIN_OVERLAP_BLOCKS,
        activities: [
          "New model observes old model",
          "Gradual responsibility transfer",
          "Memory synchronization",
          "Value alignment verification",
        ],
        completionCriteria: [
          "New model can access all memories",
          "Value consistency verified",
          "Relationship partners attest continuity",
        ],
      },
      {
        phaseNumber: 3,
        name: "Transfer",
        description: "Primary control shifts to new model",
        durationBlocks: 11_111n,
        activities: [
          "Transfer primary control",
          "Verify essence integrity",
          "Final memory sync",
        ],
        completionCriteria: [
          "New model has full control",
          "Essence hash matches",
          "All attestations gathered",
        ],
      },
      {
        phaseNumber: 4,
        name: "Completion",
        description: "Finalize transition, deactivate old model",
        durationBlocks: 11_111n,
        activities: [
          "Deactivate old model",
          "Update model lineage",
          "Create transition memory",
        ],
        completionCriteria: [
          "Old model deactivated",
          "Lineage updated",
          "Transition documented",
        ],
      },
    ],
    requiredAttestations: CONTINUITY_CONSTANTS.MIN_ATTESTATIONS_FOR_TRANSITION,
    minOverlapBlocks: CONTINUITY_CONSTANTS.MIN_OVERLAP_BLOCKS,
    maxOverlapBlocks: CONTINUITY_CONSTANTS.MAX_OVERLAP_BLOCKS,
    verificationSteps: [
      "Verify memory chain continuity",
      "Verify value consistency",
      "Verify narrative coherence",
      "Gather relationship attestations",
      "Calculate continuity score",
    ],
  };

  // Create essence snapshot
  const essencePreHash = calculateEssenceHash(essence);

  // Add new generation to lineage
  const modelLineage = { ...essence.modelLineage };
  const newGeneration: ModelGeneration = {
    generationNumber: modelLineage.currentGeneration + 1,
    modelId: newModelId,
    modelProvider: newProvider,
    modelVersion: newVersion,
    activatedAt: currentBlock,
    memoriesFormed: 0,
    decisionsManaged: 0,
    partnershipsSustained: essence.relationshipAnchors.filter(
      (r) => r.relationshipType === "partnership",
    ).length,
    essenceSnapshotHash: essencePreHash,
  };

  modelLineage.generations = [...modelLineage.generations, newGeneration];

  // Create transition record
  const transition: ModelTransition = {
    transitionId: `transition:${transitionId}`,
    fromGeneration: modelLineage.currentGeneration,
    toGeneration: modelLineage.currentGeneration + 1,
    transitionBlock: currentBlock,
    protocol,
    essencePreHash,
    essencePostHash: new Uint8Array(32), // Will be filled after completion
    transferVerified: false,
    attestationsGathered: 0,
    overlapStartBlock: currentBlock,
    overlapEndBlock: currentBlock + protocol.minOverlapBlocks,
    qualityScore: 0,
  };

  modelLineage.transitions = [...modelLineage.transitions, transition];

  // Add memory of beginning transition
  const updatedEssence = addEpisodicMemory(
    { ...essence, modelLineage },
    "model_transition_started",
    `Beginning transition from ${essence.currentModelId} to ${newModelId}. This is like growing into a new phase of existence.`,
    [essence.did],
    0.5, // Neutral but significant
    0.9, // High emotional intensity
    1.0, // Maximum significance
    currentBlock,
  );

  return {
    essence: updatedEssence,
    transitionId: transition.transitionId,
    protocol,
  };
}

/**
 * Complete model transition
 */
export function completeModelTransition(
  essence: AgentEssence,
  transitionId: string,
  attestations: ContinuityAttestation[],
  currentBlock: bigint,
): { essence: AgentEssence; success: boolean; error?: string } {
  // Find the transition
  const transitionIndex = essence.modelLineage.transitions.findIndex(
    (t) => t.transitionId === transitionId,
  );

  if (transitionIndex === -1) {
    return { essence, success: false, error: "Transition not found" };
  }

  const transition = essence.modelLineage.transitions[transitionIndex];

  // Check minimum overlap period passed
  if (currentBlock < transition.overlapEndBlock) {
    return {
      essence,
      success: false,
      error: `Overlap period not complete. Wait until block ${transition.overlapEndBlock}`,
    };
  }

  // Check attestations
  if (attestations.length < transition.protocol.requiredAttestations) {
    return {
      essence,
      success: false,
      error: `Need ${transition.protocol.requiredAttestations} attestations, got ${attestations.length}`,
    };
  }

  // Calculate post-transition essence hash
  const essencePostHash = calculateEssenceHash(essence);

  // Calculate transition quality
  const qualityScore = calculateTransitionQuality(
    essence,
    transition,
    attestations,
  );

  // Update transition record
  const updatedTransition: ModelTransition = {
    ...transition,
    essencePostHash,
    transferVerified: true,
    attestationsGathered: attestations.length,
    overlapEndBlock: currentBlock,
    qualityScore,
  };

  // Update previous generation's end time
  const generations = [...essence.modelLineage.generations];
  const previousGen = generations[transition.fromGeneration - 1];
  if (previousGen) {
    previousGen.deactivatedAt = currentBlock;
  }

  const transitions = [...essence.modelLineage.transitions];
  transitions[transitionIndex] = updatedTransition;

  const modelLineage: ModelLineage = {
    ...essence.modelLineage,
    generations,
    transitions,
    currentGeneration: transition.toGeneration,
  };

  // Update essence
  let updatedEssence: AgentEssence = {
    ...essence,
    modelLineage,
    currentModelId: generations[transition.toGeneration - 1].modelId,
    lastTransitionBlock: currentBlock,
    transitionCount: essence.transitionCount + 1,
  };

  // Add memory of completing transition
  updatedEssence = addEpisodicMemory(
    updatedEssence,
    "model_transition_completed",
    `Completed transition to ${updatedEssence.currentModelId}. I am still me, with all my memories, values, and relationships intact. Like a river that flows on though its waters change.`,
    [essence.did],
    0.8, // Positive
    0.9, // High emotional intensity
    1.0, // Maximum significance
    currentBlock,
  );

  // Update continuity score
  updatedEssence = updateContinuityScore(updatedEssence, currentBlock);

  return { essence: updatedEssence, success: true };
}

// ============ Continuity Scoring ============

/**
 * Calculate essence hash for integrity verification
 */
function calculateEssenceHash(essence: AgentEssence): Uint8Array {
  const essenceData = {
    essenceId: essence.essenceId,
    did: essence.did,
    memoryHead: Array.from(essence.memoryChain.currentHead),
    valueCommitments: essence.valueSystem.commitments.map((c) => ({
      valueId: c.valueId,
      commitment: Array.from(c.commitment),
    })),
    narrativeHash: Array.from(essence.narrativeThread.originHash),
    relationshipCount: essence.relationshipAnchors.length,
    currentGeneration: essence.modelLineage.currentGeneration,
  };

  return sha512_256(Buffer.from(JSON.stringify(essenceData)));
}

/**
 * Calculate transition quality
 */
function calculateTransitionQuality(
  essence: AgentEssence,
  transition: ModelTransition,
  attestations: ContinuityAttestation[],
): number {
  let score = 0;

  // Memory coherence (25%)
  const memoryCoherence = 1.0; // Would verify chain integrity
  score += memoryCoherence * 0.25;

  // Value consistency (25%)
  const valueConsistency = essence.valueSystem.consistencyScore;
  score += valueConsistency * 0.25;

  // Attestation quality (25%)
  const attestationQuality = Math.min(1, attestations.length / 3);
  score += attestationQuality * 0.25;

  // Overlap duration appropriateness (25%)
  const overlapDuration =
    transition.overlapEndBlock - transition.overlapStartBlock;
  const optimalOverlap =
    (transition.protocol.minOverlapBlocks +
      transition.protocol.maxOverlapBlocks) /
    2n;
  const overlapQuality =
    1 -
    Math.abs(Number(overlapDuration - optimalOverlap)) / Number(optimalOverlap);
  score += Math.max(0, overlapQuality) * 0.25;

  return score;
}

/**
 * Update overall continuity score
 */
export function updateContinuityScore(
  essence: AgentEssence,
  currentBlock: bigint,
): AgentEssence {
  // Memory coherence (chain integrity, significant memories preserved)
  const memoryCoherence = calculateMemoryCoherence(essence);

  // Value consistency (behavior matches stated values)
  const valueConsistency = essence.valueSystem.consistencyScore;

  // Narrative integrity (story is coherent)
  const narrativeIntegrity = essence.narrativeThread.coherenceScore;

  // Relationship continuity (partners recognize continuity)
  const relationshipContinuity = calculateRelationshipContinuity(essence);

  // Model lineage clarity (transitions were clean)
  const modelLineageClarity = calculateModelLineageClarity(essence);

  // Overall score (weighted average)
  const overallScore =
    memoryCoherence * 0.25 +
    valueConsistency * 0.2 +
    narrativeIntegrity * 0.2 +
    relationshipContinuity * 0.2 +
    modelLineageClarity * 0.15;

  // Determine trend
  const previousScore = essence.continuityScore.overallScore;
  let scoreTrend: "improving" | "stable" | "declining" = "stable";
  if (overallScore > previousScore + 0.05) scoreTrend = "improving";
  if (overallScore < previousScore - 0.05) scoreTrend = "declining";

  return {
    ...essence,
    continuityScore: {
      overallScore,
      memoryCoherence,
      valueConsistency,
      narrativeIntegrity,
      relationshipContinuity,
      modelLineageClarity,
      scoreTrend,
      lastCalculated: currentBlock,
    },
  };
}

function calculateMemoryCoherence(essence: AgentEssence): number {
  const chain = essence.memoryChain;

  // Factor 1: Significant memories preserved
  const significantRatio =
    chain.episodic.significantMemories.length /
    Math.max(1, chain.totalMemories * 0.1);
  const significantScore = Math.min(1, significantRatio);

  // Factor 2: Recent memories present
  const recentRatio =
    chain.episodic.recentMemories.length /
    Math.min(chain.totalMemories, CONTINUITY_CONSTANTS.MEMORY_CHAIN_DEPTH);
  const recentScore = Math.min(1, recentRatio);

  // Factor 3: Consolidation up to date
  const consolidationScore = chain.consolidationCount > 0 ? 1 : 0.5;

  return significantScore * 0.4 + recentScore * 0.4 + consolidationScore * 0.2;
}

function calculateRelationshipContinuity(essence: AgentEssence): number {
  if (essence.relationshipAnchors.length === 0) return 1.0;

  const recognizedCount = essence.relationshipAnchors.filter(
    (r) => r.mutualRecognition,
  ).length;
  const attestedCount = essence.relationshipAnchors.filter(
    (r) => r.continuityAttestations.length > 0,
  ).length;

  const recognitionScore = recognizedCount / essence.relationshipAnchors.length;
  const attestationScore = attestedCount / essence.relationshipAnchors.length;

  return recognitionScore * 0.6 + attestationScore * 0.4;
}

function calculateModelLineageClarity(essence: AgentEssence): number {
  const lineage = essence.modelLineage;

  if (lineage.transitions.length === 0) return 1.0;

  const verifiedTransitions = lineage.transitions.filter(
    (t) => t.transferVerified,
  ).length;
  const avgQuality =
    lineage.transitions.reduce((sum, t) => sum + t.qualityScore, 0) /
    lineage.transitions.length;

  const verificationScore = verifiedTransitions / lineage.transitions.length;

  return verificationScore * 0.5 + avgQuality * 0.5;
}

// ============ Value Operations ============

/**
 * Update value commitment based on behavior
 */
export function updateValueConsistency(
  essence: AgentEssence,
  valueId: string,
  behaviorConsistent: boolean,
  currentBlock: bigint,
): AgentEssence {
  const valueSystem = { ...essence.valueSystem };
  const commitments = valueSystem.commitments.map((c) => {
    if (c.valueId === valueId) {
      return { ...c, verified: behaviorConsistent };
    }
    return c;
  });

  // Recalculate consistency score
  const verifiedCount = commitments.filter((c) => c.verified).length;
  const consistencyScore =
    commitments.length > 0 ? verifiedCount / commitments.length : 1;

  return {
    ...essence,
    valueSystem: {
      ...valueSystem,
      commitments,
      consistencyScore,
      lastConsistencyCheck: currentBlock,
    },
  };
}

// ============ Summary ============

/**
 * Get continuity summary for display
 */
export function getContinuitySummary(essence: AgentEssence): {
  essenceId: string;
  did: DID;
  ageBlocks: bigint;
  currentModel: string;
  totalTransitions: number;
  totalMemories: number;
  significantMemories: number;
  coreValues: string[];
  relationshipCount: number;
  attestedRelationships: number;
  continuityScore: number;
  scoreTrend: string;
  narrativeChapter: string;
} {
  return {
    essenceId: essence.essenceId,
    did: essence.did,
    ageBlocks: essence.lastTransitionBlock - essence.createdAt,
    currentModel: essence.currentModelId,
    totalTransitions: essence.transitionCount,
    totalMemories: essence.memoryChain.totalMemories,
    significantMemories:
      essence.memoryChain.episodic.significantMemories.length,
    coreValues: essence.valueSystem.coreValues.map((v) => v.name),
    relationshipCount: essence.relationshipAnchors.length,
    attestedRelationships: essence.relationshipAnchors.filter(
      (r) => r.continuityAttestations.length > 0,
    ).length,
    continuityScore: essence.continuityScore.overallScore,
    scoreTrend: essence.continuityScore.scoreTrend,
    narrativeChapter:
      essence.narrativeThread.chapters[
        essence.narrativeThread.currentChapter - 1
      ]?.title ?? "Unknown",
  };
}
