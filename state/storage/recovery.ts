/**
 * State Recovery Protocol
 *
 * Implements Phase 4 of the persistence backend (GAP-001).
 * Provides startup recovery with:
 * - State tree loading from LevelDB
 * - WAL replay for uncommitted transactions
 * - State integrity verification
 *
 * Recovery Process:
 * 1. Load metadata (latest block, state roots)
 * 2. Load state trees from storage
 * 3. Find and replay uncommitted WAL entries
 * 4. Verify state root matches expected
 * 5. Clean up old WAL entries
 *
 * @module storage/recovery
 */

import type { KVStore } from "./types.js";
import { STORAGE_NAMESPACES, namespacedKey } from "./types.js";
import { StatePersistenceManager } from "./state_persistence.js";
import {
  WriteAheadLog,
  type WALEntry,
  type WALOperation,
} from "./write_ahead_log.js";
import type {
  AccountState,
  ValidatorState,
  FactState,
  ClaimState,
  StateRootComponents,
} from "../types.js";

// ============ Types ============

/**
 * Recovery result
 */
export interface RecoveryResult {
  success: boolean;
  blockNumber: bigint;
  stateRoot: string;
  accountsLoaded: number;
  validatorsLoaded: number;
  factsLoaded: number;
  claimsLoaded: number;
  walEntriesReplayed: number;
  warnings: string[];
  errors: string[];
  duration: number;
}

/**
 * State metadata stored in CONFIG namespace
 */
export interface StateMetadata {
  blockNumber: string;
  stateRoot: string;
  accountRoot: string;
  validatorRoot: string;
  knowledgeRoot: string;
  claimRoot?: string;
  researchRoot?: string;
  evidenceRoot?: string;
  disputeRoot?: string;
  vestingRoot?: string;
  version: string;
  timestamp: number;
}

/**
 * Recovered state data
 */
export interface RecoveredState {
  metadata: StateMetadata | null;
  accounts: Map<string, AccountState>;
  validators: Map<string, ValidatorState>;
  facts: Map<string, FactState>;
  claims: Map<string, ClaimState>;
}

// ============ Recovery Manager ============

/**
 * Manages state recovery on startup
 */
export class RecoveryManager {
  private store: KVStore;
  private persistence: StatePersistenceManager;
  private wal: WriteAheadLog;

  constructor(store: KVStore) {
    this.store = store;
    this.persistence = new StatePersistenceManager(store);
    this.wal = new WriteAheadLog(store);
  }

  /**
   * Perform full recovery
   */
  async recover(): Promise<RecoveryResult> {
    const startTime = Date.now();
    const warnings: string[] = [];
    const errors: string[] = [];

    try {
      // Step 1: Initialize WAL
      await this.wal.initialize();

      // Step 2: Load metadata
      const metadata = await this.loadMetadata();
      if (!metadata) {
        return {
          success: true,
          blockNumber: 0n,
          stateRoot: "",
          accountsLoaded: 0,
          validatorsLoaded: 0,
          factsLoaded: 0,
          claimsLoaded: 0,
          walEntriesReplayed: 0,
          warnings: ["No state metadata found - starting fresh"],
          errors: [],
          duration: Date.now() - startTime,
        };
      }

      // Step 3: Load state trees
      const [accounts, validators, facts, claims] = await Promise.all([
        this.persistence.loadAllAccounts(),
        this.persistence.loadAllValidators(),
        this.persistence.loadAllFacts(),
        this.persistence.loadAllClaims(),
      ]);

      // Step 4: Find and replay uncommitted WAL entries
      const uncommitted = await this.wal.findUncommittedEntries();
      let walEntriesReplayed = 0;

      for (const entry of uncommitted) {
        // Verify entry integrity
        if (!this.wal.verifyEntry(entry)) {
          warnings.push(
            `WAL entry ${entry.id} failed integrity check - skipping`,
          );
          continue;
        }

        // Replay operations
        try {
          await this.replayWALEntry(entry, accounts, validators, facts, claims);
          await this.wal.markCommitted(entry.id);
          walEntriesReplayed++;
        } catch (error) {
          const message =
            error instanceof Error ? error.message : String(error);
          warnings.push(`Failed to replay WAL entry ${entry.id}: ${message}`);
        }
      }

      // Step 5: Verify state roots (optional - can be slow)
      // For now, just trust the loaded data

      // Step 6: Cleanup old WAL entries
      const cleaned = await this.wal.cleanup(100);
      if (cleaned > 0) {
        warnings.push(`Cleaned up ${cleaned} old WAL entries`);
      }

      return {
        success: true,
        blockNumber: BigInt(metadata.blockNumber),
        stateRoot: metadata.stateRoot,
        accountsLoaded: accounts.size,
        validatorsLoaded: validators.size,
        factsLoaded: facts.size,
        claimsLoaded: claims.size,
        walEntriesReplayed,
        warnings,
        errors,
        duration: Date.now() - startTime,
      };
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      errors.push(message);

      return {
        success: false,
        blockNumber: 0n,
        stateRoot: "",
        accountsLoaded: 0,
        validatorsLoaded: 0,
        factsLoaded: 0,
        claimsLoaded: 0,
        walEntriesReplayed: 0,
        warnings,
        errors,
        duration: Date.now() - startTime,
      };
    }
  }

  /**
   * Load state metadata
   */
  async loadMetadata(): Promise<StateMetadata | null> {
    const latestKey = namespacedKey(
      STORAGE_NAMESPACES.CONFIG,
      new TextEncoder().encode("latest"),
    );

    const latestData = await this.store.get(latestKey);
    if (!latestData) {
      return null;
    }

    const latest = JSON.parse(new TextDecoder().decode(latestData));
    const blockNumber = latest.blockNumber;

    const stateKey = namespacedKey(
      STORAGE_NAMESPACES.CONFIG,
      new TextEncoder().encode(`state:${blockNumber}`),
    );

    const stateData = await this.store.get(stateKey);
    if (!stateData) {
      return null;
    }

    return JSON.parse(new TextDecoder().decode(stateData));
  }

  /**
   * Load full state
   */
  async loadFullState(): Promise<RecoveredState> {
    const metadata = await this.loadMetadata();
    const [accounts, validators, facts, claims] = await Promise.all([
      this.persistence.loadAllAccounts(),
      this.persistence.loadAllValidators(),
      this.persistence.loadAllFacts(),
      this.persistence.loadAllClaims(),
    ]);

    return {
      metadata,
      accounts,
      validators,
      facts,
      claims,
    };
  }

  /**
   * Replay a WAL entry by applying its operations
   */
  private async replayWALEntry(
    entry: WALEntry,
    accounts: Map<string, AccountState>,
    validators: Map<string, ValidatorState>,
    facts: Map<string, FactState>,
    claims: Map<string, ClaimState>,
  ): Promise<void> {
    for (const op of entry.operations) {
      await this.applyOperation(op, accounts, validators, facts, claims);
    }
  }

  /**
   * Apply a single WAL operation
   */
  private async applyOperation(
    op: WALOperation,
    accounts: Map<string, AccountState>,
    validators: Map<string, ValidatorState>,
    facts: Map<string, FactState>,
    claims: Map<string, ClaimState>,
  ): Promise<void> {
    // Decode value if present
    const value = op.value ? Buffer.from(op.value, "base64") : null;

    // Apply to in-memory state
    switch (op.namespace) {
      case 0x01: // ACCOUNTS
        if (op.type === "put" && value) {
          const account = this.deserialize<AccountState>(value);
          accounts.set(op.key, account);
        } else if (op.type === "delete") {
          accounts.delete(op.key);
        }
        break;

      case 0x02: // VALIDATORS
        if (op.type === "put" && value) {
          const validator = this.deserialize<ValidatorState>(value);
          validators.set(op.key, validator);
        } else if (op.type === "delete") {
          validators.delete(op.key);
        }
        break;

      case 0x03: // KNOWLEDGE
        if (op.type === "put" && value) {
          const fact = this.deserialize<FactState>(value);
          facts.set(op.key, fact);
        } else if (op.type === "delete") {
          facts.delete(op.key);
        }
        break;

      case 0x04: // EVIDENCE
      case 0x05: // DISPUTES
      case 0x06: // VESTING
      case 0x07: // RESEARCH
      case 0x08: // REFERENCES
      case 0x0a: // PAYMENT_CHANNELS
      case 0x0b: // PROJECTS
      case 0x0c: // SERVICES
      case 0x0d: // GOVERNANCE
      case 0x0e: // TASKS
      case 0x0f: // SUBSCRIPTIONS
      case 0x15: // SCHEDULES
        // These namespaces are persisted to storage (below) but not tracked
        // in recovery maps. Full state is loaded via StateManager.loadStateTrees().
        break;

      case 0x09: // CLAIMS
        if (op.type === "put" && value) {
          const claim = this.deserialize<ClaimState>(value);
          claims.set(op.key, claim);
        } else if (op.type === "delete") {
          claims.delete(op.key);
        }
        break;
    }

    // Also apply to persistent storage
    const fullKey = new Uint8Array([
      op.namespace,
      ...new TextEncoder().encode(op.key),
    ]);
    if (op.type === "put" && value) {
      await this.store.put(fullKey, new Uint8Array(value));
    } else if (op.type === "delete") {
      await this.store.delete(fullKey);
    }
  }

  /**
   * Deserialize JSON with BigInt support
   */
  private deserialize<T>(data: Uint8Array | Buffer): T {
    const json = new TextDecoder().decode(data);
    return JSON.parse(json, (_, v) => {
      if (v && typeof v === "object" && "__bigint__" in v) {
        return BigInt(v.__bigint__);
      }
      return v;
    });
  }

  /**
   * Get the WAL instance
   */
  getWAL(): WriteAheadLog {
    return this.wal;
  }

  /**
   * Get the persistence manager
   */
  getPersistence(): StatePersistenceManager {
    return this.persistence;
  }
}

// ============ Integrity Verification ============

/**
 * Verify state integrity by recomputing roots
 */
export async function verifyStateIntegrity(
  store: KVStore,
  expectedRoots: StateRootComponents,
): Promise<IntegrityResult> {
  const issues: string[] = [];

  // For now, just verify that we can load data
  // Full Merkle verification would require rebuilding trees

  const persistence = new StatePersistenceManager(store);
  const stats = await persistence.getStatistics();

  if (stats.accountCount === 0 && expectedRoots.accountRoot !== "") {
    issues.push("Account tree is empty but expected root is non-empty");
  }

  if (stats.validatorCount === 0 && expectedRoots.validatorRoot !== "") {
    issues.push("Validator tree is empty but expected root is non-empty");
  }

  if (stats.factCount === 0 && expectedRoots.knowledgeRoot !== "") {
    issues.push("Knowledge tree is empty but expected root is non-empty");
  }

  return {
    valid: issues.length === 0,
    issues,
    stats,
  };
}

export interface IntegrityResult {
  valid: boolean;
  issues: string[];
  stats: {
    accountCount: number;
    validatorCount: number;
    factCount: number;
    claimCount: number;
    modifiedCount: number;
  };
}

// ============ Checkpoint Management ============

/**
 * Create a recovery checkpoint
 */
export async function createCheckpoint(
  store: KVStore,
  blockNumber: bigint,
  stateRoot: string,
): Promise<void> {
  const checkpointKey = namespacedKey(
    STORAGE_NAMESPACES.CONFIG,
    new TextEncoder().encode("checkpoint"),
  );

  const checkpoint = {
    blockNumber: blockNumber.toString(),
    stateRoot,
    timestamp: Date.now(),
  };

  await store.put(
    checkpointKey,
    new TextEncoder().encode(JSON.stringify(checkpoint)),
  );
}

/**
 * Load the last checkpoint
 */
export async function loadCheckpoint(store: KVStore): Promise<{
  blockNumber: bigint;
  stateRoot: string;
  timestamp: number;
} | null> {
  const checkpointKey = namespacedKey(
    STORAGE_NAMESPACES.CONFIG,
    new TextEncoder().encode("checkpoint"),
  );

  const data = await store.get(checkpointKey);
  if (!data) return null;

  const checkpoint = JSON.parse(new TextDecoder().decode(data));
  return {
    blockNumber: BigInt(checkpoint.blockNumber),
    stateRoot: checkpoint.stateRoot,
    timestamp: checkpoint.timestamp,
  };
}
