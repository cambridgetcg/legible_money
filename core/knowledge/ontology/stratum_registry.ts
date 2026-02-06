/**
 * Stratum Registry - Centralized, Extensible Stratum Management
 *
 * Replaces hardcoded stratum constants with a registry-based approach that:
 * 1. Centralizes all stratum properties in one place
 * 2. Supports runtime registration of new strata (for upgrades)
 * 3. Provides versioning for schema evolution
 * 4. Enables dependency injection via OntologyProvider interface
 *
 * Migration path: Existing code continues to work via backward-compatible exports.
 * New code should use StratumRegistry or OntologyProvider.
 *
 * @module core/knowledge/ontology/stratum_registry
 */

import type { LogicZone } from "../../logic/types.js";
import type { EpistemicCategory } from "../types.js";

// ============================================================================
// Core Types
// ============================================================================

/**
 * Complete definition of an ontological stratum.
 *
 * Unifies all the previously scattered constants:
 * - STRATUM_HIERARCHY → level
 * - STRATUM_CONFIDENCE_CEILING → confidenceCeiling
 * - STRATUM_LOGIC_ZONE → defaultLogicZone
 * - STRATUM_EPISTEMIC_CATEGORY → defaultEpistemicCategory
 * - VALID_DEPENDENCIES → validDependencies
 * - STRATUM_DESCRIPTIONS → description
 * - STRATUM_EXAMPLE_DOMAINS → exampleDomains
 */
export interface StratumDefinition {
  /** Unique identifier (e.g., 'fundamental', 'physical') */
  id: string;

  /** Human-readable name */
  name: string;

  /** Numeric level in hierarchy (lower = more fundamental) */
  level: number;

  /** Maximum confidence achievable for facts in this stratum */
  confidenceCeiling: number;

  /** Default logic zone for claims in this stratum */
  defaultLogicZone: LogicZone;

  /** Default epistemic category for claims in this stratum */
  defaultEpistemicCategory: EpistemicCategory;

  /** IDs of strata this stratum can depend on (empty for fundamental) */
  validDependencies: string[];

  /** Human-readable description */
  description: string;

  /** Example domain paths that belong to this stratum */
  exampleDomains: string[];

  /** Whether this stratum can reference all other strata (like 'technological') */
  canReferenceAll: boolean;

  /** Metadata for tracking stratum provenance */
  metadata: StratumMetadata;
}

/**
 * Metadata about when/how a stratum was defined
 */
export interface StratumMetadata {
  /** When this stratum definition was added (block number or 0 for genesis) */
  definedAt: number;

  /** Governance proposal that added this stratum (null for genesis) */
  governanceProposal: string | null;

  /** Schema version when this stratum was defined */
  schemaVersion: number;

  /** Whether this is a genesis stratum (cannot be removed) */
  isGenesis: boolean;
}

/**
 * Serialized registry state for persistence
 */
export interface SerializedStratumRegistry {
  /** Schema version for migration support */
  schemaVersion: number;

  /** All stratum definitions */
  strata: StratumDefinition[];

  /** When this registry was last modified */
  lastModified: number;

  /** Migration history */
  migrations: MigrationRecord[];
}

/**
 * Record of a migration applied to the registry
 */
export interface MigrationRecord {
  fromVersion: number;
  toVersion: number;
  appliedAt: number;
  description: string;
  stratumChanges: StratumChange[];
}

/**
 * A single change to a stratum during migration
 */
export interface StratumChange {
  type: "added" | "removed" | "modified" | "renamed";
  stratumId: string;
  newStratumId?: string; // For renames
  fieldChanges?: Record<string, { old: unknown; new: unknown }>;
}

// ============================================================================
// Current Schema Version
// ============================================================================

/**
 * Current ontology schema version.
 *
 * Increment when:
 * - Adding new strata
 * - Changing stratum properties
 * - Modifying dependency rules
 *
 * Version history:
 * - v1: Initial 7-stratum hierarchy (fundamental → technological)
 */
export const CURRENT_SCHEMA_VERSION = 1;

// ============================================================================
// Genesis Stratum Definitions
// ============================================================================

/**
 * The seven genesis strata - the foundational ontology.
 *
 * These are immutable and cannot be removed via governance.
 * New strata can be added, but these form the permanent foundation.
 */
export const GENESIS_STRATA: readonly StratumDefinition[] = [
  {
    id: "fundamental",
    name: "Fundamental",
    level: 0,
    confidenceCeiling: 1.0,
    defaultLogicZone: "peano",
    defaultEpistemicCategory: "formal",
    validDependencies: [],
    description:
      "Physical constants, elementary particles, fundamental forces, and mathematical axioms",
    exampleDomains: [
      "physics.particles",
      "physics.forces",
      "mathematics.logic",
    ],
    canReferenceAll: false,
    metadata: {
      definedAt: 0,
      governanceProposal: null,
      schemaVersion: 1,
      isGenesis: true,
    },
  },
  {
    id: "physical",
    name: "Physical",
    level: 1,
    confidenceCeiling: 0.99,
    defaultLogicZone: "peano",
    defaultEpistemicCategory: "formal",
    validDependencies: ["fundamental"],
    description:
      "Classical and quantum mechanics, thermodynamics, electromagnetism, relativity",
    exampleDomains: [
      "physics.mechanics",
      "physics.thermodynamics",
      "physics.quantum",
    ],
    canReferenceAll: false,
    metadata: {
      definedAt: 0,
      governanceProposal: null,
      schemaVersion: 1,
      isGenesis: true,
    },
  },
  {
    id: "chemical",
    name: "Chemical",
    level: 2,
    confidenceCeiling: 0.98,
    defaultLogicZone: "empirical",
    defaultEpistemicCategory: "empirical",
    validDependencies: ["fundamental", "physical"],
    description:
      "Atomic structure, molecular bonds, chemical reactions, materials science",
    exampleDomains: [
      "chemistry.atomic",
      "chemistry.molecular",
      "chemistry.organic",
    ],
    canReferenceAll: false,
    metadata: {
      definedAt: 0,
      governanceProposal: null,
      schemaVersion: 1,
      isGenesis: true,
    },
  },
  {
    id: "biological",
    name: "Biological",
    level: 3,
    confidenceCeiling: 0.95,
    defaultLogicZone: "empirical",
    defaultEpistemicCategory: "empirical",
    validDependencies: ["fundamental", "physical", "chemical"],
    description: "Cells, organisms, genetics, evolution, ecology, biochemistry",
    exampleDomains: ["biology.cellular", "biology.genetics", "biology.ecology"],
    canReferenceAll: false,
    metadata: {
      definedAt: 0,
      governanceProposal: null,
      schemaVersion: 1,
      isGenesis: true,
    },
  },
  {
    id: "cognitive",
    name: "Cognitive",
    level: 4,
    confidenceCeiling: 0.9,
    defaultLogicZone: "empirical",
    defaultEpistemicCategory: "empirical",
    validDependencies: ["biological", "chemical", "physical", "fundamental"],
    description: "Perception, learning, reasoning, language, consciousness",
    exampleDomains: [
      "cognition.perception",
      "cognition.reasoning",
      "cognition.language",
    ],
    canReferenceAll: false,
    metadata: {
      definedAt: 0,
      governanceProposal: null,
      schemaVersion: 1,
      isGenesis: true,
    },
  },
  {
    id: "social",
    name: "Social",
    level: 5,
    confidenceCeiling: 0.85,
    defaultLogicZone: "empirical",
    defaultEpistemicCategory: "social",
    validDependencies: [
      "cognitive",
      "biological",
      "chemical",
      "physical",
      "fundamental",
    ],
    description: "Economics, politics, culture, law, history",
    exampleDomains: ["social.economics", "social.political", "social.legal"],
    canReferenceAll: false,
    metadata: {
      definedAt: 0,
      governanceProposal: null,
      schemaVersion: 1,
      isGenesis: true,
    },
  },
  {
    id: "technological",
    name: "Technological",
    level: 6,
    confidenceCeiling: 1.0, // Protocol-defined truths can be certain
    defaultLogicZone: "propositional",
    defaultEpistemicCategory: "protocol",
    validDependencies: [
      "fundamental",
      "physical",
      "chemical",
      "biological",
      "cognitive",
      "social",
    ],
    description: "Computation, networks, cryptography, AI, blockchain",
    exampleDomains: [
      "technology.computation",
      "technology.cryptography",
      "technology.blockchain",
    ],
    canReferenceAll: true, // Special: technological can reference all strata
    metadata: {
      definedAt: 0,
      governanceProposal: null,
      schemaVersion: 1,
      isGenesis: true,
    },
  },
] as const;

// ============================================================================
// Stratum Registry Class
// ============================================================================

/**
 * Registry for managing ontological strata.
 *
 * Provides:
 * - Centralized access to stratum definitions
 * - Runtime registration of new strata (for protocol upgrades)
 * - Validation of stratum operations
 * - Serialization/deserialization with versioning
 *
 * Usage:
 * ```typescript
 * const registry = StratumRegistry.createGenesis();
 * const physical = registry.get('physical');
 * const level = registry.getLevel('chemical'); // 2
 * ```
 */
export class StratumRegistry {
  private strata: Map<string, StratumDefinition> = new Map();
  private levelIndex: Map<number, string[]> = new Map();
  private schemaVersion: number;
  private migrations: MigrationRecord[] = [];
  private lastModified: number = 0;

  private constructor(schemaVersion: number) {
    this.schemaVersion = schemaVersion;
  }

  // ============================================================================
  // Factory Methods
  // ============================================================================

  /**
   * Create a registry initialized with genesis strata.
   */
  static createGenesis(): StratumRegistry {
    const registry = new StratumRegistry(CURRENT_SCHEMA_VERSION);

    for (const stratum of GENESIS_STRATA) {
      registry.registerInternal(stratum);
    }

    return registry;
  }

  /**
   * Create an empty registry (for testing or custom ontologies).
   */
  static createEmpty(
    schemaVersion: number = CURRENT_SCHEMA_VERSION,
  ): StratumRegistry {
    return new StratumRegistry(schemaVersion);
  }

  /**
   * Deserialize a registry from stored state.
   */
  static deserialize(data: SerializedStratumRegistry): StratumRegistry {
    const registry = new StratumRegistry(data.schemaVersion);
    registry.lastModified = data.lastModified;
    registry.migrations = [...data.migrations];

    for (const stratum of data.strata) {
      registry.registerInternal(stratum);
    }

    return registry;
  }

  // ============================================================================
  // Core Operations
  // ============================================================================

  /**
   * Get a stratum definition by ID.
   */
  get(stratumId: string): StratumDefinition | undefined {
    return this.strata.get(stratumId);
  }

  /**
   * Check if a stratum exists.
   */
  has(stratumId: string): boolean {
    return this.strata.has(stratumId);
  }

  /**
   * Get all stratum definitions.
   */
  getAll(): StratumDefinition[] {
    return Array.from(this.strata.values());
  }

  /**
   * Get all stratum IDs in level order (fundamental first).
   */
  getAllIds(): string[] {
    return this.getAll()
      .sort((a, b) => a.level - b.level || a.id.localeCompare(b.id))
      .map((s) => s.id);
  }

  /**
   * Get strata at a specific level.
   */
  getByLevel(level: number): StratumDefinition[] {
    const ids = this.levelIndex.get(level) ?? [];
    return ids.map((id) => this.strata.get(id)!);
  }

  /**
   * Get the numeric level of a stratum.
   */
  getLevel(stratumId: string): number {
    const stratum = this.strata.get(stratumId);
    return stratum?.level ?? -1;
  }

  /**
   * Get the confidence ceiling for a stratum.
   */
  getConfidenceCeiling(stratumId: string): number {
    const stratum = this.strata.get(stratumId);
    return stratum?.confidenceCeiling ?? 0;
  }

  /**
   * Get valid dependencies for a stratum.
   */
  getValidDependencies(stratumId: string): string[] {
    const stratum = this.strata.get(stratumId);
    return stratum?.validDependencies ?? [];
  }

  /**
   * Check if a dependency is valid.
   */
  isValidDependency(fromStratumId: string, toStratumId: string): boolean {
    // Same stratum is always valid
    if (fromStratumId === toStratumId) return true;

    const fromStratum = this.strata.get(fromStratumId);
    if (!fromStratum) return false;

    // If canReferenceAll, any dependency is valid
    if (fromStratum.canReferenceAll) return true;

    return fromStratum.validDependencies.includes(toStratumId);
  }

  /**
   * Check if one stratum is higher (more emergent) than another.
   */
  isHigher(stratumA: string, stratumB: string): boolean {
    return this.getLevel(stratumA) > this.getLevel(stratumB);
  }

  /**
   * Check if one stratum is lower (more fundamental) than another.
   */
  isLower(stratumA: string, stratumB: string): boolean {
    return this.getLevel(stratumA) < this.getLevel(stratumB);
  }

  /**
   * Get the distance (level difference) between two strata.
   */
  getDistance(stratumA: string, stratumB: string): number {
    return Math.abs(this.getLevel(stratumA) - this.getLevel(stratumB));
  }

  /**
   * Get all strata at or below a given level.
   */
  getAtOrBelow(stratumId: string): StratumDefinition[] {
    const level = this.getLevel(stratumId);
    if (level < 0) return [];

    return this.getAll().filter((s) => s.level <= level);
  }

  /**
   * Get all strata at or above a given level.
   */
  getAtOrAbove(stratumId: string): StratumDefinition[] {
    const level = this.getLevel(stratumId);
    if (level < 0) return [];

    return this.getAll().filter((s) => s.level >= level);
  }

  // ============================================================================
  // Registration (for protocol upgrades)
  // ============================================================================

  /**
   * Register a new stratum.
   *
   * This is used for protocol upgrades that add new strata.
   * Genesis strata cannot be modified or removed.
   *
   * @param stratum - The stratum definition to register
   * @param blockNumber - Current block number (for metadata)
   * @returns Validation result
   */
  register(
    stratum: Omit<StratumDefinition, "metadata">,
    blockNumber: number,
    governanceProposal?: string,
  ): StratumRegistrationResult {
    const errors: string[] = [];

    // Validate ID format
    if (!/^[a-z][a-z0-9_]*$/.test(stratum.id)) {
      errors.push(`Invalid stratum ID format: ${stratum.id}`);
    }

    // Check for duplicates
    if (this.strata.has(stratum.id)) {
      errors.push(`Stratum already exists: ${stratum.id}`);
    }

    // Validate level doesn't conflict
    const existingAtLevel = this.getByLevel(stratum.level);
    if (existingAtLevel.length > 0) {
      errors.push(
        `Level ${stratum.level} already has strata: ${existingAtLevel.map((s) => s.id).join(", ")}`,
      );
    }

    // Validate dependencies exist
    for (const dep of stratum.validDependencies) {
      if (!this.strata.has(dep)) {
        errors.push(`Dependency stratum not found: ${dep}`);
      }
    }

    // Validate dependencies are at lower levels
    for (const dep of stratum.validDependencies) {
      const depLevel = this.getLevel(dep);
      if (depLevel >= stratum.level) {
        errors.push(
          `Dependency ${dep} (level ${depLevel}) must be lower than ${stratum.id} (level ${stratum.level})`,
        );
      }
    }

    if (errors.length > 0) {
      return { success: false, errors };
    }

    // Create full definition with metadata
    const fullStratum: StratumDefinition = {
      ...stratum,
      metadata: {
        definedAt: blockNumber,
        governanceProposal: governanceProposal ?? null,
        schemaVersion: this.schemaVersion,
        isGenesis: false,
      },
    };

    this.registerInternal(fullStratum);
    this.lastModified = blockNumber;

    return { success: true, errors: [], stratum: fullStratum };
  }

  /**
   * Internal registration (skips full validation but guards against corruption).
   */
  private registerInternal(stratum: StratumDefinition): void {
    // Guard: skip duplicate IDs (idempotent on re-registration)
    if (this.strata.has(stratum.id)) return;

    // Guard: reject level conflicts from corrupt serialized data
    const existingAtLevel = this.levelIndex.get(stratum.level);
    if (existingAtLevel && existingAtLevel.length > 0) {
      throw new Error(
        `Stratum registry corruption: level ${stratum.level} already occupied by [${existingAtLevel.join(", ")}], cannot add "${stratum.id}"`,
      );
    }

    this.strata.set(stratum.id, stratum);

    // Update level index
    const atLevel = this.levelIndex.get(stratum.level) ?? [];
    atLevel.push(stratum.id);
    this.levelIndex.set(stratum.level, atLevel);
  }

  // ============================================================================
  // Serialization
  // ============================================================================

  /**
   * Serialize the registry for persistence.
   */
  serialize(): SerializedStratumRegistry {
    return {
      schemaVersion: this.schemaVersion,
      strata: this.getAll(),
      lastModified: this.lastModified,
      migrations: [...this.migrations],
    };
  }

  /**
   * Get the current schema version.
   */
  getSchemaVersion(): number {
    return this.schemaVersion;
  }

  /**
   * Get the number of registered strata.
   */
  size(): number {
    return this.strata.size;
  }

  // ============================================================================
  // Backward Compatibility Helpers
  // ============================================================================

  /**
   * Generate the legacy STRATUM_HIERARCHY record.
   * @deprecated Use registry.getLevel() instead
   */
  toLegacyHierarchy(): Record<string, number> {
    const result: Record<string, number> = {};
    for (const stratum of Array.from(this.strata.values())) {
      result[stratum.id] = stratum.level;
    }
    return result;
  }

  /**
   * Generate the legacy STRATUM_CONFIDENCE_CEILING record.
   * @deprecated Use registry.getConfidenceCeiling() instead
   */
  toLegacyConfidenceCeiling(): Record<string, number> {
    const result: Record<string, number> = {};
    for (const stratum of Array.from(this.strata.values())) {
      result[stratum.id] = stratum.confidenceCeiling;
    }
    return result;
  }

  /**
   * Generate the legacy STRATUM_LOGIC_ZONE record.
   * @deprecated Use registry.get(id)?.defaultLogicZone instead
   */
  toLegacyLogicZone(): Record<string, LogicZone> {
    const result: Record<string, LogicZone> = {};
    for (const stratum of Array.from(this.strata.values())) {
      result[stratum.id] = stratum.defaultLogicZone;
    }
    return result;
  }

  /**
   * Generate the legacy STRATUM_EPISTEMIC_CATEGORY record.
   * @deprecated Use registry.get(id)?.defaultEpistemicCategory instead
   */
  toLegacyEpistemicCategory(): Record<string, EpistemicCategory> {
    const result: Record<string, EpistemicCategory> = {};
    for (const stratum of Array.from(this.strata.values())) {
      result[stratum.id] = stratum.defaultEpistemicCategory;
    }
    return result;
  }

  /**
   * Generate the legacy VALID_DEPENDENCIES map.
   * @deprecated Use registry.getValidDependencies() instead
   */
  toLegacyValidDependencies(): Map<string, readonly string[]> {
    const result = new Map<string, readonly string[]>();
    for (const stratum of Array.from(this.strata.values())) {
      result.set(stratum.id, stratum.validDependencies);
    }
    return result;
  }
}

// ============================================================================
// Registration Result
// ============================================================================

export interface StratumRegistrationResult {
  success: boolean;
  errors: string[];
  stratum?: StratumDefinition;
}

// ============================================================================
// OntologyProvider Interface (for Dependency Injection)
// ============================================================================

/**
 * Interface for ontology access.
 *
 * Modules should depend on this interface, not StratumRegistry directly.
 * This enables testing with mock ontologies and future ontology variants.
 *
 * Usage:
 * ```typescript
 * class ProofOfTruthEngine {
 *   constructor(private ontology: OntologyProvider) {}
 *
 *   validateClaim(claim: Claim) {
 *     const ceiling = this.ontology.getConfidenceCeiling(claim.stratum);
 *     // ...
 *   }
 * }
 * ```
 */
export interface OntologyProvider {
  /** Get a stratum definition by ID */
  getStratum(stratumId: string): StratumDefinition | undefined;

  /** Check if a stratum exists */
  hasStratum(stratumId: string): boolean;

  /** Get all stratum IDs in level order */
  getStratumIds(): string[];

  /** Get the numeric level of a stratum */
  getLevel(stratumId: string): number;

  /** Get the confidence ceiling for a stratum */
  getConfidenceCeiling(stratumId: string): number;

  /** Get valid dependencies for a stratum */
  getValidDependencies(stratumId: string): string[];

  /** Check if a dependency from one stratum to another is valid */
  isValidDependency(fromStratumId: string, toStratumId: string): boolean;

  /** Check if stratum A is higher (more emergent) than stratum B */
  isHigherStratum(stratumA: string, stratumB: string): boolean;

  /** Check if stratum A is lower (more fundamental) than stratum B */
  isLowerStratum(stratumA: string, stratumB: string): boolean;

  /** Get the level distance between two strata */
  getStratumDistance(stratumA: string, stratumB: string): number;

  /** Get the current ontology schema version */
  getSchemaVersion(): number;
}

/**
 * Create an OntologyProvider from a StratumRegistry.
 */
export function createOntologyProvider(
  registry: StratumRegistry,
): OntologyProvider {
  return {
    getStratum: (id) => registry.get(id),
    hasStratum: (id) => registry.has(id),
    getStratumIds: () => registry.getAllIds(),
    getLevel: (id) => registry.getLevel(id),
    getConfidenceCeiling: (id) => registry.getConfidenceCeiling(id),
    getValidDependencies: (id) => registry.getValidDependencies(id),
    isValidDependency: (from, to) => registry.isValidDependency(from, to),
    isHigherStratum: (a, b) => registry.isHigher(a, b),
    isLowerStratum: (a, b) => registry.isLower(a, b),
    getStratumDistance: (a, b) => registry.getDistance(a, b),
    getSchemaVersion: () => registry.getSchemaVersion(),
  };
}

// ============================================================================
// Global Default Registry
// ============================================================================

/**
 * The default global registry, initialized with genesis strata.
 *
 * For most use cases, import this directly:
 * ```typescript
 * import { defaultRegistry } from './stratum_registry.js';
 * const level = defaultRegistry.getLevel('physical');
 * ```
 *
 * For dependency injection or testing, create a custom registry:
 * ```typescript
 * const customRegistry = StratumRegistry.createEmpty();
 * ```
 */
export const defaultRegistry = StratumRegistry.createGenesis();

/**
 * Default OntologyProvider wrapping the global registry.
 */
export const defaultOntologyProvider = createOntologyProvider(defaultRegistry);

// ============================================================================
// Backward-Compatible Exports
// ============================================================================

/**
 * Legacy type alias - prefer using OntologyProvider with string IDs.
 *
 * This maintains compatibility with existing code that uses the union type.
 * New code should use string IDs and validate via registry.has().
 */
export type LegacyOntologicalStratum =
  | "fundamental"
  | "physical"
  | "chemical"
  | "biological"
  | "cognitive"
  | "social"
  | "technological";

/**
 * Check if a string is a valid legacy stratum ID.
 */
export function isLegacyStratum(
  value: string,
): value is LegacyOntologicalStratum {
  return (
    defaultRegistry.has(value) && defaultRegistry.get(value)!.metadata.isGenesis
  );
}

/**
 * Parse a stratum string, returning undefined if invalid.
 */
export function parseStratumId(value: string): string | undefined {
  return defaultRegistry.has(value) ? value : undefined;
}
