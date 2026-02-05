/**
 * State Persistence Layer
 *
 * Provides durable persistence for all state trees to LevelDB.
 * Implements Phase 2 of the persistence backend (GAP-001).
 *
 * Architecture:
 * - Each state tree's entries are stored under a namespaced prefix
 * - Modified entries are tracked for incremental persistence
 * - Full state can be loaded on startup for recovery
 *
 * Namespaces:
 * - 0x01: Accounts (DID -> AccountState)
 * - 0x02: Validators (PublicKey -> ValidatorState)
 * - 0x03: Knowledge/Facts (FactID -> FactState)
 * - 0x04: Evidence
 * - 0x05: Disputes
 * - 0x06: Vesting
 * - 0x07: Research
 * - 0x08: Claims
 *
 * @module storage/state_persistence
 */

import type { KVStore } from "./types.js";
import {
  STORAGE_NAMESPACES,
  namespacedKey,
  type StorageNamespace,
} from "./types.js";
import type {
  AccountState,
  ValidatorState,
  FactState,
  ClaimState,
} from "../types.js";

// ============ Serialization Helpers ============

/**
 * BigInt-safe JSON serialization
 */
function serialize<T>(value: T): Uint8Array {
  const json = JSON.stringify(value, (_, v) =>
    typeof v === "bigint" ? { __bigint__: v.toString() } : v,
  );
  return new TextEncoder().encode(json);
}

/**
 * BigInt-safe JSON deserialization
 */
function deserialize<T>(data: Uint8Array): T {
  const json = new TextDecoder().decode(data);
  return JSON.parse(json, (_, v) => {
    if (v && typeof v === "object" && "__bigint__" in v) {
      return BigInt(v.__bigint__);
    }
    return v;
  });
}

/**
 * Convert string key to bytes
 */
function stringToKey(str: string): Uint8Array {
  return new TextEncoder().encode(str);
}

/**
 * Convert bytes key to string
 */
function keyToString(key: Uint8Array): string {
  return new TextDecoder().decode(key);
}

// ============ Namespace Keys ============

// Extended namespaces for all state trees
const STATE_NAMESPACES = {
  ACCOUNTS: STORAGE_NAMESPACES.ACCOUNTS,
  VALIDATORS: STORAGE_NAMESPACES.VALIDATORS,
  KNOWLEDGE: STORAGE_NAMESPACES.KNOWLEDGE,
  EVIDENCE: STORAGE_NAMESPACES.EVIDENCE,
  DISPUTES: STORAGE_NAMESPACES.DISPUTES,
  VESTING: STORAGE_NAMESPACES.VESTING,
  RESEARCH: STORAGE_NAMESPACES.RESEARCH,
  CLAIMS: new Uint8Array([0x08]), // Claims namespace
  DELEGATIONS: new Uint8Array([0x09]), // Delegations namespace
} as const;

// ============ Modified Entry Tracker ============

/**
 * Tracks modified entries across all state trees for incremental persistence
 */
export class ModifiedEntryTracker {
  private modified: Map<string, Set<string>> = new Map();

  constructor() {
    // Initialize sets for each namespace
    for (const namespace of Object.keys(STATE_NAMESPACES)) {
      this.modified.set(namespace, new Set());
    }
  }

  /**
   * Mark an entry as modified
   */
  markModified(namespace: keyof typeof STATE_NAMESPACES, key: string): void {
    this.modified.get(namespace)?.add(key);
  }

  /**
   * Get all modified keys for a namespace
   */
  getModified(namespace: keyof typeof STATE_NAMESPACES): Set<string> {
    return this.modified.get(namespace) || new Set();
  }

  /**
   * Clear modified entries after persistence
   */
  clear(): void {
    for (const set of this.modified.values()) {
      set.clear();
    }
  }

  /**
   * Check if there are any modified entries
   */
  hasModifications(): boolean {
    for (const set of this.modified.values()) {
      if (set.size > 0) return true;
    }
    return false;
  }

  /**
   * Get total count of modified entries
   */
  getModifiedCount(): number {
    let count = 0;
    for (const set of this.modified.values()) {
      count += set.size;
    }
    return count;
  }
}

// ============ State Persistence Manager ============

/**
 * Manages persistence of all state trees to LevelDB
 */
export class StatePersistenceManager {
  private store: KVStore;
  private tracker: ModifiedEntryTracker;

  constructor(store: KVStore) {
    this.store = store;
    this.tracker = new ModifiedEntryTracker();
  }

  /**
   * Get the modified entry tracker
   */
  getTracker(): ModifiedEntryTracker {
    return this.tracker;
  }

  // ============ Account Persistence ============

  /**
   * Persist a single account
   */
  async persistAccount(did: string, account: AccountState): Promise<void> {
    const key = namespacedKey(STATE_NAMESPACES.ACCOUNTS, stringToKey(did));
    await this.store.put(key, serialize(account));
    this.tracker.markModified("ACCOUNTS", did);
  }

  /**
   * Load a single account
   */
  async loadAccount(did: string): Promise<AccountState | null> {
    const key = namespacedKey(STATE_NAMESPACES.ACCOUNTS, stringToKey(did));
    const data = await this.store.get(key);
    if (!data) return null;
    return deserialize<AccountState>(data);
  }

  /**
   * Delete an account
   */
  async deleteAccount(did: string): Promise<void> {
    const key = namespacedKey(STATE_NAMESPACES.ACCOUNTS, stringToKey(did));
    await this.store.delete(key);
    this.tracker.markModified("ACCOUNTS", did);
  }

  /**
   * Load all accounts
   */
  async loadAllAccounts(): Promise<Map<string, AccountState>> {
    const accounts = new Map<string, AccountState>();
    const prefix = STATE_NAMESPACES.ACCOUNTS;
    const prefixEnd = new Uint8Array([prefix[0] + 1]);

    for await (const { key, value } of this.store.iterator({
      gte: prefix,
      lt: prefixEnd,
    })) {
      const did = keyToString(key.slice(prefix.length));
      const account = deserialize<AccountState>(value);
      accounts.set(did, account);
    }

    return accounts;
  }

  // ============ Validator Persistence ============

  /**
   * Persist a single validator
   */
  async persistValidator(
    publicKeyHex: string,
    validator: ValidatorState,
  ): Promise<void> {
    const key = namespacedKey(
      STATE_NAMESPACES.VALIDATORS,
      stringToKey(publicKeyHex),
    );
    await this.store.put(key, serialize(validator));
    this.tracker.markModified("VALIDATORS", publicKeyHex);
  }

  /**
   * Load a single validator
   */
  async loadValidator(publicKeyHex: string): Promise<ValidatorState | null> {
    const key = namespacedKey(
      STATE_NAMESPACES.VALIDATORS,
      stringToKey(publicKeyHex),
    );
    const data = await this.store.get(key);
    if (!data) return null;
    return deserialize<ValidatorState>(data);
  }

  /**
   * Load all validators
   */
  async loadAllValidators(): Promise<Map<string, ValidatorState>> {
    const validators = new Map<string, ValidatorState>();
    const prefix = STATE_NAMESPACES.VALIDATORS;
    const prefixEnd = new Uint8Array([prefix[0] + 1]);

    for await (const { key, value } of this.store.iterator({
      gte: prefix,
      lt: prefixEnd,
    })) {
      const publicKeyHex = keyToString(key.slice(prefix.length));
      const validator = deserialize<ValidatorState>(value);
      validators.set(publicKeyHex, validator);
    }

    return validators;
  }

  // ============ Knowledge/Fact Persistence ============

  /**
   * Persist a single fact
   */
  async persistFact(factId: string, fact: FactState): Promise<void> {
    const key = namespacedKey(STATE_NAMESPACES.KNOWLEDGE, stringToKey(factId));
    await this.store.put(key, serialize(fact));
    this.tracker.markModified("KNOWLEDGE", factId);
  }

  /**
   * Load a single fact
   */
  async loadFact(factId: string): Promise<FactState | null> {
    const key = namespacedKey(STATE_NAMESPACES.KNOWLEDGE, stringToKey(factId));
    const data = await this.store.get(key);
    if (!data) return null;
    return deserialize<FactState>(data);
  }

  /**
   * Load all facts
   */
  async loadAllFacts(): Promise<Map<string, FactState>> {
    const facts = new Map<string, FactState>();
    const prefix = STATE_NAMESPACES.KNOWLEDGE;
    const prefixEnd = new Uint8Array([prefix[0] + 1]);

    for await (const { key, value } of this.store.iterator({
      gte: prefix,
      lt: prefixEnd,
    })) {
      const factId = keyToString(key.slice(prefix.length));
      const fact = deserialize<FactState>(value);
      facts.set(factId, fact);
    }

    return facts;
  }

  // ============ Claim Persistence ============

  /**
   * Persist a single claim
   */
  async persistClaim(claimId: string, claim: ClaimState): Promise<void> {
    const key = namespacedKey(STATE_NAMESPACES.CLAIMS, stringToKey(claimId));
    await this.store.put(key, serialize(claim));
    this.tracker.markModified("CLAIMS", claimId);
  }

  /**
   * Load a single claim
   */
  async loadClaim(claimId: string): Promise<ClaimState | null> {
    const key = namespacedKey(STATE_NAMESPACES.CLAIMS, stringToKey(claimId));
    const data = await this.store.get(key);
    if (!data) return null;
    return deserialize<ClaimState>(data);
  }

  /**
   * Load all claims
   */
  async loadAllClaims(): Promise<Map<string, ClaimState>> {
    const claims = new Map<string, ClaimState>();
    const prefix = STATE_NAMESPACES.CLAIMS;
    const prefixEnd = new Uint8Array([prefix[0] + 1]);

    for await (const { key, value } of this.store.iterator({
      gte: prefix,
      lt: prefixEnd,
    })) {
      const claimId = keyToString(key.slice(prefix.length));
      const claim = deserialize<ClaimState>(value);
      claims.set(claimId, claim);
    }

    return claims;
  }

  // ============ Batch Persistence ============

  /**
   * Persist multiple entries in a single atomic batch
   */
  async persistBatch(entries: {
    accounts?: Map<string, AccountState>;
    validators?: Map<string, ValidatorState>;
    facts?: Map<string, FactState>;
    claims?: Map<string, ClaimState>;
  }): Promise<number> {
    const batch = this.store.batch();
    let count = 0;

    if (entries.accounts) {
      for (const [did, account] of entries.accounts) {
        const key = namespacedKey(STATE_NAMESPACES.ACCOUNTS, stringToKey(did));
        batch.put(key, serialize(account));
        this.tracker.markModified("ACCOUNTS", did);
        count++;
      }
    }

    if (entries.validators) {
      for (const [publicKeyHex, validator] of entries.validators) {
        const key = namespacedKey(
          STATE_NAMESPACES.VALIDATORS,
          stringToKey(publicKeyHex),
        );
        batch.put(key, serialize(validator));
        this.tracker.markModified("VALIDATORS", publicKeyHex);
        count++;
      }
    }

    if (entries.facts) {
      for (const [factId, fact] of entries.facts) {
        const key = namespacedKey(
          STATE_NAMESPACES.KNOWLEDGE,
          stringToKey(factId),
        );
        batch.put(key, serialize(fact));
        this.tracker.markModified("KNOWLEDGE", factId);
        count++;
      }
    }

    if (entries.claims) {
      for (const [claimId, claim] of entries.claims) {
        const key = namespacedKey(
          STATE_NAMESPACES.CLAIMS,
          stringToKey(claimId),
        );
        batch.put(key, serialize(claim));
        this.tracker.markModified("CLAIMS", claimId);
        count++;
      }
    }

    await batch.write();
    return count;
  }

  // ============ Generic Persistence ============

  /**
   * Persist arbitrary data under a namespace
   */
  async persistGeneric<T>(
    namespace: StorageNamespace,
    key: string,
    value: T,
  ): Promise<void> {
    const fullKey = namespacedKey(namespace, stringToKey(key));
    await this.store.put(fullKey, serialize(value));
  }

  /**
   * Load arbitrary data from a namespace
   */
  async loadGeneric<T>(
    namespace: StorageNamespace,
    key: string,
  ): Promise<T | null> {
    const fullKey = namespacedKey(namespace, stringToKey(key));
    const data = await this.store.get(fullKey);
    if (!data) return null;
    return deserialize<T>(data);
  }

  /**
   * Load all entries from a namespace
   */
  async loadAllGeneric<T>(
    namespace: StorageNamespace,
  ): Promise<Map<string, T>> {
    const entries = new Map<string, T>();
    const prefixEnd = new Uint8Array([namespace[0] + 1]);

    for await (const { key, value } of this.store.iterator({
      gte: namespace,
      lt: prefixEnd,
    })) {
      const id = keyToString(key.slice(namespace.length));
      const entry = deserialize<T>(value);
      entries.set(id, entry);
    }

    return entries;
  }

  // ============ Statistics ============

  /**
   * Get count of entries in a namespace
   */
  async getCount(namespace: StorageNamespace): Promise<number> {
    let count = 0;
    const prefixEnd = new Uint8Array([namespace[0] + 1]);

    for await (const _ of this.store.iterator({
      gte: namespace,
      lt: prefixEnd,
    })) {
      count++;
    }

    return count;
  }

  /**
   * Get persistence statistics
   */
  async getStatistics(): Promise<PersistenceStatistics> {
    const [accountCount, validatorCount, factCount, claimCount] =
      await Promise.all([
        this.getCount(STATE_NAMESPACES.ACCOUNTS),
        this.getCount(STATE_NAMESPACES.VALIDATORS),
        this.getCount(STATE_NAMESPACES.KNOWLEDGE),
        this.getCount(STATE_NAMESPACES.CLAIMS),
      ]);

    return {
      accountCount,
      validatorCount,
      factCount,
      claimCount,
      modifiedCount: this.tracker.getModifiedCount(),
    };
  }

  /**
   * Clear the modified entry tracker
   */
  clearModified(): void {
    this.tracker.clear();
  }
}

// ============ Types ============

export interface PersistenceStatistics {
  accountCount: number;
  validatorCount: number;
  factCount: number;
  claimCount: number;
  modifiedCount: number;
}

// ============ Exports ============

export { STATE_NAMESPACES };
