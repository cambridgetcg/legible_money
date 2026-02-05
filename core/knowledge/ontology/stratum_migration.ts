/**
 * Stratum Migration - Schema Evolution Support
 *
 * Provides utilities for migrating data when the ontology schema changes.
 * Supports:
 * - Adding new strata
 * - Renaming strata
 * - Splitting strata (e.g., biological → cellular + organismal)
 * - Merging strata
 * - Confidence recalculation after changes
 *
 * @module core/knowledge/ontology/stratum_migration
 */

import {
  StratumRegistry,
  type StratumDefinition,
  type MigrationRecord,
  type StratumChange,
} from "./stratum_registry.js";

// ============================================================================
// Migration Types
// ============================================================================

/**
 * A migration function that transforms data from one schema version to another.
 */
export interface StratumMigration {
  /** Source schema version */
  fromVersion: number;

  /** Target schema version */
  toVersion: number;

  /** Human-readable description */
  description: string;

  /**
   * Map an old stratum ID to new stratum ID(s).
   * Returns an array to support splitting (one → many).
   * Returns empty array to indicate deletion.
   */
  mapStratum(oldStratumId: string): string[];

  /**
   * Get the stratum definition for a new stratum.
   * Called when mapStratum returns a new ID not in the old schema.
   */
  getNewStratumDefinition?(
    newStratumId: string,
  ): Omit<StratumDefinition, "metadata"> | null;

  /**
   * Recalculate confidence for a fact after migration.
   * @param oldConfidence - The fact's confidence before migration
   * @param oldStratum - The fact's stratum before migration
   * @param newStratum - The fact's stratum after migration
   * @returns New confidence value
   */
  migrateConfidence(
    oldConfidence: number,
    oldStratum: string,
    newStratum: string,
  ): number;

  /**
   * Determine which stratum a fact should migrate to when a stratum is split.
   * Called when mapStratum returns multiple IDs.
   *
   * @param domainId - The fact's domain
   * @param factContent - Optional content hint for classification
   * @returns The chosen new stratum ID
   */
  classifyForSplit?(
    oldStratumId: string,
    domainId: string,
    factContent?: string,
  ): string;
}

/**
 * Result of applying a migration to a dataset
 */
export interface MigrationResult {
  success: boolean;
  errors: string[];
  warnings: string[];

  /** Number of facts migrated */
  factsMigrated: number;

  /** Number of domains migrated */
  domainsMigrated: number;

  /** Strata that were added */
  strataAdded: string[];

  /** Strata that were removed */
  strataRemoved: string[];

  /** Strata that were renamed (old → new) */
  strataRenamed: Map<string, string>;

  /** Migration record for audit trail */
  record: MigrationRecord;
}

/**
 * Input for migrating a fact
 */
export interface FactMigrationInput {
  factId: string;
  domainId: string;
  stratum: string;
  confidence: number;
  content?: string; // Optional content for split classification
}

/**
 * Output from migrating a fact
 */
export interface FactMigrationOutput {
  factId: string;
  domainId: string;
  oldStratum: string;
  newStratum: string;
  oldConfidence: number;
  newConfidence: number;
  changed: boolean;
}

// ============================================================================
// Migration Runner
// ============================================================================

/**
 * Applies migrations to data.
 *
 * Usage:
 * ```typescript
 * const runner = new MigrationRunner();
 * runner.registerMigration(migration_v1_to_v2);
 *
 * const result = runner.migrateFacts(facts, 1, 2);
 * ```
 */
export class MigrationRunner {
  private migrations: Map<string, StratumMigration> = new Map();

  /**
   * Register a migration.
   */
  registerMigration(migration: StratumMigration): void {
    const key = `${migration.fromVersion}→${migration.toVersion}`;
    this.migrations.set(key, migration);
  }

  /**
   * Get the migration path between two versions.
   * Returns ordered list of migrations to apply.
   */
  getMigrationPath(fromVersion: number, toVersion: number): StratumMigration[] {
    if (fromVersion === toVersion) return [];
    if (fromVersion > toVersion) {
      throw new Error("Downgrade migrations not supported");
    }

    const path: StratumMigration[] = [];
    let current = fromVersion;

    while (current < toVersion) {
      // Look for direct migration
      const key = `${current}→${current + 1}`;
      const migration = this.migrations.get(key);

      if (!migration) {
        throw new Error(
          `No migration found from version ${current} to ${current + 1}`,
        );
      }

      path.push(migration);
      current++;
    }

    return path;
  }

  /**
   * Migrate a single fact through a migration chain.
   */
  migrateFact(
    fact: FactMigrationInput,
    fromVersion: number,
    toVersion: number,
  ): FactMigrationOutput {
    const path = this.getMigrationPath(fromVersion, toVersion);

    let currentStratum = fact.stratum;
    let currentConfidence = fact.confidence;

    for (const migration of path) {
      const newStrata = migration.mapStratum(currentStratum);

      if (newStrata.length === 0) {
        // Stratum was removed - this is an error for facts
        throw new Error(
          `Stratum ${currentStratum} was removed in migration v${migration.fromVersion}→v${migration.toVersion}, ` +
            `but fact ${fact.factId} still references it`,
        );
      }

      let newStratum: string;

      if (newStrata.length === 1) {
        newStratum = newStrata[0];
      } else {
        // Split case - need to classify
        if (!migration.classifyForSplit) {
          throw new Error(
            `Stratum ${currentStratum} was split in migration, but no classifyForSplit function provided`,
          );
        }
        newStratum = migration.classifyForSplit(
          currentStratum,
          fact.domainId,
          fact.content,
        );
      }

      currentConfidence = migration.migrateConfidence(
        currentConfidence,
        currentStratum,
        newStratum,
      );
      currentStratum = newStratum;
    }

    return {
      factId: fact.factId,
      domainId: fact.domainId,
      oldStratum: fact.stratum,
      newStratum: currentStratum,
      oldConfidence: fact.confidence,
      newConfidence: currentConfidence,
      changed:
        currentStratum !== fact.stratum ||
        currentConfidence !== fact.confidence,
    };
  }

  /**
   * Migrate multiple facts.
   */
  migrateFacts(
    facts: FactMigrationInput[],
    fromVersion: number,
    toVersion: number,
  ): {
    outputs: FactMigrationOutput[];
    errors: Array<{ factId: string; error: string }>;
  } {
    const outputs: FactMigrationOutput[] = [];
    const errors: Array<{ factId: string; error: string }> = [];

    for (const fact of facts) {
      try {
        outputs.push(this.migrateFact(fact, fromVersion, toVersion));
      } catch (error) {
        errors.push({
          factId: fact.factId,
          error: error instanceof Error ? error.message : String(error),
        });
      }
    }

    return { outputs, errors };
  }

  /**
   * Generate a migration record for audit trail.
   */
  generateMigrationRecord(
    migration: StratumMigration,
    appliedAt: number,
    oldRegistry: StratumRegistry,
    newRegistry: StratumRegistry,
  ): MigrationRecord {
    const changes: StratumChange[] = [];

    // Detect removed strata
    for (const oldId of oldRegistry.getAllIds()) {
      const mapped = migration.mapStratum(oldId);
      if (mapped.length === 0) {
        changes.push({ type: "removed", stratumId: oldId });
      } else if (mapped.length === 1 && mapped[0] !== oldId) {
        changes.push({
          type: "renamed",
          stratumId: oldId,
          newStratumId: mapped[0],
        });
      }
    }

    // Detect added strata
    for (const newId of newRegistry.getAllIds()) {
      if (!oldRegistry.has(newId)) {
        // Check if this is a rename target
        const isRenameTarget = changes.some(
          (c) => c.type === "renamed" && c.newStratumId === newId,
        );
        if (!isRenameTarget) {
          changes.push({ type: "added", stratumId: newId });
        }
      }
    }

    return {
      fromVersion: migration.fromVersion,
      toVersion: migration.toVersion,
      appliedAt,
      description: migration.description,
      stratumChanges: changes,
    };
  }
}

// ============================================================================
// Example Migrations
// ============================================================================

/**
 * Example: Identity migration (no changes).
 * Use as template for real migrations.
 */
export const migration_identity: StratumMigration = {
  fromVersion: 1,
  toVersion: 1,
  description: "Identity migration (no changes)",

  mapStratum(oldStratumId: string): string[] {
    return [oldStratumId];
  },

  migrateConfidence(oldConfidence: number): number {
    return oldConfidence;
  },
};

/**
 * Example: Migration that splits 'technological' into 'formal_systems' and 'sociotechnical'.
 *
 * This is a hypothetical migration showing how to split a stratum.
 */
export const migration_split_technological: StratumMigration = {
  fromVersion: 1,
  toVersion: 2,
  description: "Split technological into formal_systems and sociotechnical",

  mapStratum(oldStratumId: string): string[] {
    if (oldStratumId === "technological") {
      return ["formal_systems", "sociotechnical"];
    }
    return [oldStratumId];
  },

  getNewStratumDefinition(newStratumId: string) {
    if (newStratumId === "formal_systems") {
      return {
        id: "formal_systems",
        name: "Formal Systems",
        level: 6,
        confidenceCeiling: 1.0,
        defaultLogicZone: "propositional" as const,
        defaultEpistemicCategory: "formal" as const,
        validDependencies: ["fundamental", "physical"],
        description:
          "Computation, cryptography, protocols - deterministic systems",
        exampleDomains: ["technology.computation", "technology.cryptography"],
        canReferenceAll: false,
      };
    }

    if (newStratumId === "sociotechnical") {
      return {
        id: "sociotechnical",
        name: "Sociotechnical",
        level: 7,
        confidenceCeiling: 0.85,
        defaultLogicZone: "empirical" as const,
        defaultEpistemicCategory: "social" as const,
        validDependencies: ["formal_systems", "cognitive", "social"],
        description: "AI behavior, network effects, human-computer interaction",
        exampleDomains: ["technology.ai", "technology.networks"],
        canReferenceAll: false,
      };
    }

    return null;
  },

  migrateConfidence(
    oldConfidence: number,
    oldStratum: string,
    newStratum: string,
  ): number {
    // If moving to sociotechnical, apply its ceiling
    if (oldStratum === "technological" && newStratum === "sociotechnical") {
      return Math.min(oldConfidence, 0.85);
    }
    return oldConfidence;
  },

  classifyForSplit(
    _oldStratumId: string,
    domainId: string,
    _factContent?: string,
  ): string {
    // Classify based on domain
    const formalDomains = [
      "technology.computation",
      "technology.cryptography",
      "technology.blockchain.consensus",
    ];

    for (const prefix of formalDomains) {
      if (domainId.startsWith(prefix)) {
        return "formal_systems";
      }
    }

    return "sociotechnical";
  },
};

// ============================================================================
// Migration Validation
// ============================================================================

/**
 * Validate that a migration is well-formed.
 */
export function validateMigration(
  migration: StratumMigration,
  sourceRegistry: StratumRegistry,
): { valid: boolean; errors: string[] } {
  const errors: string[] = [];

  // Check version numbers
  if (migration.fromVersion >= migration.toVersion) {
    errors.push("fromVersion must be less than toVersion");
  }

  // Check that all source strata can be mapped
  for (const stratumId of sourceRegistry.getAllIds()) {
    try {
      const mapped = migration.mapStratum(stratumId);
      if (mapped.length > 1 && !migration.classifyForSplit) {
        errors.push(
          `Stratum ${stratumId} maps to multiple targets but no classifyForSplit function provided`,
        );
      }
    } catch (error) {
      errors.push(`mapStratum failed for ${stratumId}: ${error}`);
    }
  }

  return {
    valid: errors.length === 0,
    errors,
  };
}

/**
 * Dry-run a migration to see what would change.
 */
export function dryRunMigration(
  migration: StratumMigration,
  sourceRegistry: StratumRegistry,
  facts: FactMigrationInput[],
): {
  stratumChanges: StratumChange[];
  factChanges: FactMigrationOutput[];
  errors: string[];
} {
  const runner = new MigrationRunner();
  runner.registerMigration(migration);

  const stratumChanges: StratumChange[] = [];

  // Analyze stratum changes
  for (const stratumId of sourceRegistry.getAllIds()) {
    const mapped = migration.mapStratum(stratumId);

    if (mapped.length === 0) {
      stratumChanges.push({ type: "removed", stratumId });
    } else if (mapped.length === 1 && mapped[0] !== stratumId) {
      stratumChanges.push({
        type: "renamed",
        stratumId,
        newStratumId: mapped[0],
      });
    } else if (mapped.length > 1) {
      // Split - record both the removal and additions
      stratumChanges.push({ type: "removed", stratumId });
      for (const newId of mapped) {
        if (!sourceRegistry.has(newId)) {
          stratumChanges.push({ type: "added", stratumId: newId });
        }
      }
    }
  }

  // Analyze fact changes
  const { outputs: factChanges, errors } = runner.migrateFacts(
    facts,
    migration.fromVersion,
    migration.toVersion,
  );

  return {
    stratumChanges,
    factChanges: factChanges.filter((f) => f.changed),
    errors: errors.map((e) => `Fact ${e.factId}: ${e.error}`),
  };
}

// ============================================================================
// Registry Migration
// ============================================================================

/**
 * Apply a migration to a registry, producing a new registry.
 *
 * Note: This creates a new registry; the original is not modified.
 */
export function migrateRegistry(
  sourceRegistry: StratumRegistry,
  migration: StratumMigration,
  blockNumber: number,
): { newRegistry: StratumRegistry; record: MigrationRecord } {
  // Validate first
  const validation = validateMigration(migration, sourceRegistry);
  if (!validation.valid) {
    throw new Error(`Invalid migration: ${validation.errors.join(", ")}`);
  }

  // Create new registry
  const newRegistry = StratumRegistry.createEmpty(migration.toVersion);

  // Track changes for record
  const changes: StratumChange[] = [];

  // Process each old stratum
  for (const oldStratum of sourceRegistry.getAll()) {
    const mappedIds = migration.mapStratum(oldStratum.id);

    if (mappedIds.length === 0) {
      // Removed
      changes.push({ type: "removed", stratumId: oldStratum.id });
      continue;
    }

    for (const newId of mappedIds) {
      if (newRegistry.has(newId)) {
        continue; // Already added (e.g., from another source stratum)
      }

      if (newId === oldStratum.id) {
        // Same ID - copy with potentially updated metadata
        newRegistry.register(
          {
            ...oldStratum,
            id: newId,
          },
          blockNumber,
        );
      } else if (migration.getNewStratumDefinition) {
        // New stratum from split or rename
        const newDef = migration.getNewStratumDefinition(newId);
        if (newDef) {
          newRegistry.register(newDef, blockNumber);
          changes.push({ type: "added", stratumId: newId });
        }
      }
    }
  }

  const record: MigrationRecord = {
    fromVersion: migration.fromVersion,
    toVersion: migration.toVersion,
    appliedAt: blockNumber,
    description: migration.description,
    stratumChanges: changes,
  };

  return { newRegistry, record };
}
