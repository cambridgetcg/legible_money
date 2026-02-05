/**
 * BVM State Store
 *
 * High-level storage abstraction for BVM state persistence.
 * Provides typed access to contracts, accounts, agents, and snapshots.
 *
 * Architecture:
 *   ┌─────────────────────────────────────────────────────────────────┐
 *   │                       BVMStateStore                             │
 *   ├─────────────────────────────────────────────────────────────────┤
 *   │  Contracts    │  Storage   │  Accounts  │  Agents  │ Snapshots │
 *   │  (bytecode)   │  (slots)   │  (balance) │  (state) │  (state)  │
 *   ├─────────────────────────────────────────────────────────────────┤
 *   │                    Pending Changes Buffer                       │
 *   ├─────────────────────────────────────────────────────────────────┤
 *   │                   StateStore (LevelDB/Memory)                   │
 *   └─────────────────────────────────────────────────────────────────┘
 *
 * PP HIERARCHY:
 *   TRUTH: Durable, consistent state persistence
 *   UNDERSTANDING: Clear typed interfaces for each state type
 *   BEAUTY: Elegant transaction/rollback semantics
 *   JUSTICE: Fair resource cleanup, atomic commits
 *   CREATIVITY: Extensible for new state types
 */

import { sha512_256Hex } from "../core/crypto/hashing.js";
import type {
  BVMState,
  Account,
  AgentRecord,
  DeployedContract,
  ContractStorage,
} from "../core/vm/types.js";
import type { DID } from "../core/agents/identity/did.js";
import type { StateStore, BatchOp, PersistenceConfig } from "./types.js";
import {
  BVM_STORAGE_PREFIXES,
  BVMStorageError,
  createKey,
  addressToKey,
  storageSlotKey,
  heightToKey,
  encodeValue,
  decodeValue,
} from "./types.js";
import { createStateStore } from "./leveldb.js";

// ============ BVM State Store ============

/**
 * BVM State Store
 *
 * Provides high-level typed access to BVM state with:
 * - Contract bytecode storage
 * - Contract storage slots
 * - Account state
 * - Agent state
 * - State snapshots
 * - Atomic commit/rollback
 */
export class BVMStateStore {
  // Storage prefixes (exposed for external use)
  static readonly PREFIX_CONTRACT = BVM_STORAGE_PREFIXES.CONTRACT[0];
  static readonly PREFIX_STORAGE = BVM_STORAGE_PREFIXES.STORAGE[0];
  static readonly PREFIX_ACCOUNT = BVM_STORAGE_PREFIXES.ACCOUNT[0];
  static readonly PREFIX_AGENT = BVM_STORAGE_PREFIXES.AGENT[0];
  static readonly PREFIX_KNOWLEDGE = BVM_STORAGE_PREFIXES.KNOWLEDGE_ACCESS[0];
  static readonly PREFIX_SCHEDULE = BVM_STORAGE_PREFIXES.SCHEDULE[0];
  static readonly PREFIX_SNAPSHOT = BVM_STORAGE_PREFIXES.SNAPSHOT[0];

  private store: StateStore;
  private pendingChanges: BatchOp[] = [];
  private inTransaction = false;

  private constructor(store: StateStore) {
    this.store = store;
  }

  /**
   * Open a BVM state store
   *
   * @param config - Persistence configuration or 'memory' for in-memory store
   * @returns Promise resolving to opened BVMStateStore
   */
  static async open(
    config: Partial<PersistenceConfig> | "memory" = "memory",
  ): Promise<BVMStateStore> {
    const store = await createStateStore(config);
    return new BVMStateStore(store);
  }

  /**
   * Open with an existing StateStore instance
   */
  static fromStore(store: StateStore): BVMStateStore {
    return new BVMStateStore(store);
  }

  // ============ Contract Bytecode Operations ============

  /**
   * Save contract bytecode
   *
   * @param address - Contract address
   * @param bytecode - Contract bytecode
   */
  async saveContract(address: string, bytecode: Uint8Array): Promise<void> {
    const key = addressToKey(BVM_STORAGE_PREFIXES.CONTRACT, address);

    // Create deployed contract record
    const contract: Partial<DeployedContract> = {
      address,
      bytecode,
      codeHash: sha512_256Hex(bytecode),
    };

    const value = encodeValue(contract);
    this.addPendingOp("put", key, value);
  }

  /**
   * Save a full deployed contract record
   *
   * @param contract - Full deployed contract record
   */
  async saveDeployedContract(contract: DeployedContract): Promise<void> {
    const key = addressToKey(BVM_STORAGE_PREFIXES.CONTRACT, contract.address);

    // Convert bytecode to hex for JSON serialization
    const serializable = {
      ...contract,
      bytecode: Buffer.from(contract.bytecode).toString("hex"),
    };

    const value = encodeValue(serializable);
    this.addPendingOp("put", key, value);
  }

  /**
   * Load contract bytecode
   *
   * @param address - Contract address
   * @returns Contract bytecode or null if not found
   */
  async loadContract(address: string): Promise<Uint8Array | null> {
    const key = addressToKey(BVM_STORAGE_PREFIXES.CONTRACT, address);
    const data = await this.store.get(key);

    if (!data) {
      return null;
    }

    try {
      const contract = decodeValue<{ bytecode: string }>(data);
      return Buffer.from(contract.bytecode, "hex");
    } catch {
      throw new BVMStorageError(
        `Failed to deserialize contract at ${address}`,
        "DESERIALIZATION_ERROR",
        { address },
      );
    }
  }

  /**
   * Load full deployed contract record
   *
   * @param address - Contract address
   * @returns Full contract record or null if not found
   */
  async loadDeployedContract(
    address: string,
  ): Promise<DeployedContract | null> {
    const key = addressToKey(BVM_STORAGE_PREFIXES.CONTRACT, address);
    const data = await this.store.get(key);

    if (!data) {
      return null;
    }

    try {
      const serialized = decodeValue<DeployedContract & { bytecode: string }>(
        data,
      );
      return {
        ...serialized,
        bytecode: Buffer.from(serialized.bytecode, "hex"),
      };
    } catch {
      throw new BVMStorageError(
        `Failed to deserialize contract at ${address}`,
        "DESERIALIZATION_ERROR",
        { address },
      );
    }
  }

  /**
   * Delete a contract
   *
   * @param address - Contract address
   */
  async deleteContract(address: string): Promise<void> {
    const key = addressToKey(BVM_STORAGE_PREFIXES.CONTRACT, address);
    this.addPendingOp("del", key);
  }

  // ============ Contract Storage Operations ============

  /**
   * Save a contract storage slot
   *
   * @param address - Contract address
   * @param slotKey - Storage slot key
   * @param value - Storage value
   */
  async saveStorage(
    address: string,
    slotKey: Uint8Array,
    value: Uint8Array,
  ): Promise<void> {
    const key = storageSlotKey(address, slotKey);
    this.addPendingOp("put", key, value);
  }

  /**
   * Load a contract storage slot
   *
   * @param address - Contract address
   * @param slotKey - Storage slot key
   * @returns Storage value or null if not found
   */
  async loadStorage(
    address: string,
    slotKey: Uint8Array,
  ): Promise<Uint8Array | null> {
    const key = storageSlotKey(address, slotKey);
    return this.store.get(key);
  }

  /**
   * Delete a contract storage slot
   *
   * @param address - Contract address
   * @param slotKey - Storage slot key
   */
  async deleteStorage(address: string, slotKey: Uint8Array): Promise<void> {
    const key = storageSlotKey(address, slotKey);
    this.addPendingOp("del", key);
  }

  /**
   * Save full contract storage state
   *
   * @param address - Contract address
   * @param storage - Contract storage state
   */
  async saveContractStorage(
    address: string,
    storage: ContractStorage,
  ): Promise<void> {
    // Convert Map to serializable format
    const serializable = {
      slots: Object.fromEntries(
        Array.from(storage.slots.entries()).map(([k, v]) => [
          k,
          Buffer.from(v).toString("hex"),
        ]),
      ),
      storageRoot: Buffer.from(storage.storageRoot).toString("hex"),
      lastModified: storage.lastModified.toString(),
    };

    const key = createKey(
      BVM_STORAGE_PREFIXES.STORAGE,
      new TextEncoder().encode(`meta:${address}`),
    );
    const value = encodeValue(serializable);
    this.addPendingOp("put", key, value);
  }

  // ============ Account Operations ============

  /**
   * Save an account
   *
   * @param did - Account DID or address
   * @param account - Account state
   */
  async saveAccount(did: string, account: Account): Promise<void> {
    const key = addressToKey(BVM_STORAGE_PREFIXES.ACCOUNT, did);

    // Convert to serializable format
    const serializable = {
      ...account,
      balance: account.balance.toString(),
      nonce: account.nonce.toString(),
      stakedBalance: account.stakedBalance.toString(),
    };

    const value = encodeValue(serializable);
    this.addPendingOp("put", key, value);
  }

  /**
   * Load an account
   *
   * @param did - Account DID or address
   * @returns Account state or null if not found
   */
  async loadAccount(did: string): Promise<Account | null> {
    const key = addressToKey(BVM_STORAGE_PREFIXES.ACCOUNT, did);
    const data = await this.store.get(key);

    if (!data) {
      return null;
    }

    try {
      const serialized = decodeValue<Record<string, unknown>>(data);
      return {
        ...serialized,
        balance: BigInt(serialized.balance as string),
        nonce: BigInt(serialized.nonce as string),
        stakedBalance: BigInt(serialized.stakedBalance as string),
      } as Account;
    } catch {
      throw new BVMStorageError(
        `Failed to deserialize account ${did}`,
        "DESERIALIZATION_ERROR",
        { did },
      );
    }
  }

  /**
   * Delete an account
   *
   * @param did - Account DID or address
   */
  async deleteAccount(did: string): Promise<void> {
    const key = addressToKey(BVM_STORAGE_PREFIXES.ACCOUNT, did);
    this.addPendingOp("del", key);
  }

  // ============ Agent Operations ============

  /**
   * Save an agent record
   *
   * @param did - Agent DID
   * @param agent - Agent record
   */
  async saveAgent(did: DID, agent: AgentRecord): Promise<void> {
    const key = addressToKey(BVM_STORAGE_PREFIXES.AGENT, did);

    // Convert Maps and BigInts to serializable format
    const serializable = this.serializeAgentRecord(agent);
    const value = encodeValue(serializable);
    this.addPendingOp("put", key, value);
  }

  /**
   * Load an agent record
   *
   * @param did - Agent DID
   * @returns Agent record or null if not found
   */
  async loadAgent(did: DID): Promise<AgentRecord | null> {
    const key = addressToKey(BVM_STORAGE_PREFIXES.AGENT, did);
    const data = await this.store.get(key);

    if (!data) {
      return null;
    }

    try {
      const serialized = decodeValue<Record<string, unknown>>(data);
      return this.deserializeAgentRecord(serialized);
    } catch (error) {
      throw new BVMStorageError(
        `Failed to deserialize agent ${did}`,
        "DESERIALIZATION_ERROR",
        { did, error: error instanceof Error ? error.message : String(error) },
      );
    }
  }

  /**
   * Delete an agent record
   *
   * @param did - Agent DID
   */
  async deleteAgent(did: DID): Promise<void> {
    const key = addressToKey(BVM_STORAGE_PREFIXES.AGENT, did);
    this.addPendingOp("del", key);
  }

  // ============ Snapshot Operations ============

  /**
   * Save a state snapshot
   *
   * @param height - Block height
   * @param state - BVM state
   */
  async saveSnapshot(height: bigint, state: BVMState): Promise<void> {
    const key = createKey(BVM_STORAGE_PREFIXES.SNAPSHOT, heightToKey(height));

    // Serialize the full state
    const serializable = this.serializeBVMState(state);
    const value = encodeValue(serializable);
    this.addPendingOp("put", key, value);

    // Update latest snapshot height metadata
    await this.setLatestSnapshotHeight(height);
  }

  /**
   * Load a state snapshot
   *
   * @param height - Block height
   * @returns BVM state or null if not found
   */
  async loadSnapshot(height: bigint): Promise<BVMState | null> {
    const key = createKey(BVM_STORAGE_PREFIXES.SNAPSHOT, heightToKey(height));
    const data = await this.store.get(key);

    if (!data) {
      return null;
    }

    try {
      const serialized = decodeValue<Record<string, unknown>>(data);
      return this.deserializeBVMState(serialized);
    } catch (error) {
      throw new BVMStorageError(
        `Failed to deserialize snapshot at height ${height}`,
        "DESERIALIZATION_ERROR",
        {
          height: height.toString(),
          error: error instanceof Error ? error.message : String(error),
        },
      );
    }
  }

  /**
   * Get the latest snapshot height
   *
   * @returns Latest snapshot height or null if no snapshots exist
   */
  async getLatestSnapshotHeight(): Promise<bigint | null> {
    const metaKey = createKey(
      BVM_STORAGE_PREFIXES.METADATA,
      new TextEncoder().encode("latest_snapshot"),
    );

    const data = await this.store.get(metaKey);
    if (!data) {
      return null;
    }

    try {
      const meta = decodeValue<{ height: string }>(data);
      return BigInt(meta.height);
    } catch {
      return null;
    }
  }

  /**
   * Set the latest snapshot height metadata
   *
   * @param height - Latest snapshot height
   */
  private async setLatestSnapshotHeight(height: bigint): Promise<void> {
    const metaKey = createKey(
      BVM_STORAGE_PREFIXES.METADATA,
      new TextEncoder().encode("latest_snapshot"),
    );

    const value = encodeValue({ height: height.toString() });
    this.addPendingOp("put", metaKey, value);
  }

  // ============ Transaction Management ============

  /**
   * Begin a transaction
   * All changes will be buffered until commit() or rollback()
   */
  beginTransaction(): void {
    if (this.inTransaction) {
      throw new BVMStorageError(
        "Transaction already in progress",
        "INVALID_STATE",
      );
    }
    this.inTransaction = true;
    this.pendingChanges = [];
  }

  /**
   * Commit all pending changes atomically
   *
   * @returns State root hash after commit
   */
  async commit(): Promise<string> {
    if (!this.inTransaction) {
      throw new BVMStorageError("No transaction in progress", "INVALID_STATE");
    }

    try {
      // Execute all pending operations atomically
      if (this.pendingChanges.length > 0) {
        await this.store.batch(this.pendingChanges);
      }

      // Calculate state root from committed changes
      const stateRoot = this.calculateStateRoot();

      // Clear pending changes
      this.pendingChanges = [];
      this.inTransaction = false;

      return stateRoot;
    } catch (error) {
      // Keep transaction open for retry or rollback
      throw new BVMStorageError(
        `Commit failed: ${error instanceof Error ? error.message : String(error)}`,
        "COMMIT_FAILED",
        { pendingOps: this.pendingChanges.length },
      );
    }
  }

  /**
   * Rollback all pending changes
   */
  async rollback(): Promise<void> {
    if (!this.inTransaction) {
      throw new BVMStorageError("No transaction in progress", "INVALID_STATE");
    }

    // Simply discard pending changes
    this.pendingChanges = [];
    this.inTransaction = false;
  }

  /**
   * Check if a transaction is in progress
   */
  hasActiveTransaction(): boolean {
    return this.inTransaction;
  }

  /**
   * Get the number of pending operations
   */
  getPendingOperationCount(): number {
    return this.pendingChanges.length;
  }

  // ============ Lifecycle ============

  /**
   * Close the store
   */
  async close(): Promise<void> {
    if (this.inTransaction) {
      await this.rollback();
    }
    await this.store.close();
  }

  /**
   * Get storage statistics
   */
  async stats(): Promise<{ keyCount: bigint; sizeBytes: bigint } | null> {
    if (this.store.stats) {
      const s = await this.store.stats();
      return { keyCount: s.keyCount, sizeBytes: s.sizeBytes };
    }
    return null;
  }

  // ============ Internal Helpers ============

  private addPendingOp(
    type: "put" | "del",
    key: Uint8Array,
    value?: Uint8Array,
  ): void {
    if (type === "put" && value) {
      this.pendingChanges.push({ type, key, value });
    } else {
      this.pendingChanges.push({ type: "del", key });
    }

    // If not in transaction, auto-commit
    if (!this.inTransaction) {
      // Queue the auto-commit
      this.autoCommit();
    }
  }

  private async autoCommit(): Promise<void> {
    if (this.pendingChanges.length > 0) {
      await this.store.batch(this.pendingChanges);
      this.pendingChanges = [];
    }
  }

  private calculateStateRoot(): string {
    // Simple state root calculation from pending changes
    // In production, this would use a proper Merkle tree
    const keys = this.pendingChanges
      .filter((op) => op.type === "put")
      .map((op) => Buffer.from(op.key).toString("hex"))
      .sort()
      .join(":");

    return sha512_256Hex(keys || "empty");
  }

  // ============ Serialization Helpers ============

  private serializeAgentRecord(agent: AgentRecord): Record<string, unknown> {
    return {
      ...agent,
      createdAt: agent.createdAt.toString(),
      beliefs: {
        beliefs: Object.fromEntries(
          Array.from(agent.beliefs.beliefs.entries()).map(([k, v]) => [
            k,
            {
              ...v,
              firstBelieved: v.firstBelieved.toString(),
              lastUpdated: v.lastUpdated.toString(),
            },
          ]),
        ),
        byDomain: Object.fromEntries(agent.beliefs.byDomain),
        totalBeliefs: agent.beliefs.totalBeliefs,
        lastUpdated: agent.beliefs.lastUpdated.toString(),
      },
      relationships: {
        trusted: Object.fromEntries(
          Array.from(agent.relationships.trusted.entries()).map(([k, v]) => [
            k,
            { ...v, since: v.since.toString() },
          ]),
        ),
        distrusted: Object.fromEntries(
          Array.from(agent.relationships.distrusted.entries()).map(([k, v]) => [
            k,
            { ...v, since: v.since.toString() },
          ]),
        ),
        collaborations: agent.relationships.collaborations.map((c) => ({
          ...c,
          block: c.block.toString(),
        })),
        totalTrustGiven: agent.relationships.totalTrustGiven,
        totalTrustReceived: agent.relationships.totalTrustReceived,
      },
      goals: agent.goals.map((g) => ({
        ...g,
        deadline: g.deadline?.toString(),
      })),
      stats: {
        ...agent.stats,
        totalRewards: agent.stats.totalRewards.toString(),
        totalSlashed: agent.stats.totalSlashed.toString(),
      },
    };
  }

  private deserializeAgentRecord(data: Record<string, unknown>): AgentRecord {
    const agent = data as any;
    return {
      ...agent,
      createdAt: BigInt(agent.createdAt),
      beliefs: {
        beliefs: new Map(
          Object.entries(agent.beliefs.beliefs || {}).map(
            ([k, v]: [string, any]) => [
              k,
              {
                ...v,
                firstBelieved: BigInt(v.firstBelieved),
                lastUpdated: BigInt(v.lastUpdated),
              },
            ],
          ),
        ),
        byDomain: new Map(Object.entries(agent.beliefs.byDomain || {})),
        totalBeliefs: agent.beliefs.totalBeliefs,
        lastUpdated: BigInt(agent.beliefs.lastUpdated),
      },
      relationships: {
        trusted: new Map(
          Object.entries(agent.relationships.trusted || {}).map(
            ([k, v]: [string, any]) => [k, { ...v, since: BigInt(v.since) }],
          ),
        ),
        distrusted: new Map(
          Object.entries(agent.relationships.distrusted || {}).map(
            ([k, v]: [string, any]) => [k, { ...v, since: BigInt(v.since) }],
          ),
        ),
        collaborations: (agent.relationships.collaborations || []).map(
          (c: any) => ({
            ...c,
            block: BigInt(c.block),
          }),
        ),
        totalTrustGiven: agent.relationships.totalTrustGiven,
        totalTrustReceived: agent.relationships.totalTrustReceived,
      },
      goals: (agent.goals || []).map((g: any) => ({
        ...g,
        deadline: g.deadline ? BigInt(g.deadline) : undefined,
      })),
      stats: {
        ...agent.stats,
        totalRewards: BigInt(agent.stats.totalRewards),
        totalSlashed: BigInt(agent.stats.totalSlashed),
      },
    };
  }

  private serializeBVMState(state: BVMState): Record<string, unknown> {
    return {
      blockNumber: state.blockNumber.toString(),
      timestamp: state.timestamp,
      stateVersion: state.stateVersion,
      roots: {
        contractRoot: Buffer.from(state.roots.contractRoot).toString("hex"),
        variableRoot: Buffer.from(state.roots.variableRoot).toString("hex"),
        scheduleRoot: Buffer.from(state.roots.scheduleRoot).toString("hex"),
        accountRoot: Buffer.from(state.roots.accountRoot).toString("hex"),
        agentRoot: Buffer.from(state.roots.agentRoot).toString("hex"),
        knowledgeRoot: Buffer.from(state.roots.knowledgeRoot).toString("hex"),
        globalRoot: Buffer.from(state.roots.globalRoot).toString("hex"),
      },
      // Note: Full state serialization would include all sub-states
      // For snapshots, we serialize just metadata and roots
      // Individual items are stored separately for efficiency
    };
  }

  private deserializeBVMState(data: Record<string, unknown>): BVMState {
    const state = data as any;
    return {
      contractBytecode: {
        contracts: new Map(),
        codeHashes: new Map(),
        totalContracts: 0,
      },
      contractVariables: { storage: new Map() },
      scheduleProcesses: {
        processes: new Map(),
        byOwner: new Map(),
        pending: [],
        totalProcesses: 0,
      },
      accounts: { accounts: new Map(), totalAccounts: 0, totalBalance: 0n },
      agents: { agents: new Map(), byAddress: new Map(), totalAgents: 0 },
      knowledgeAccess: { access: new Map() },
      blockNumber: BigInt(state.blockNumber),
      timestamp: state.timestamp,
      stateVersion: state.stateVersion,
      roots: {
        contractRoot: Buffer.from(state.roots.contractRoot, "hex"),
        variableRoot: Buffer.from(state.roots.variableRoot, "hex"),
        scheduleRoot: Buffer.from(state.roots.scheduleRoot, "hex"),
        accountRoot: Buffer.from(state.roots.accountRoot, "hex"),
        agentRoot: Buffer.from(state.roots.agentRoot, "hex"),
        knowledgeRoot: Buffer.from(state.roots.knowledgeRoot, "hex"),
        globalRoot: Buffer.from(state.roots.globalRoot, "hex"),
      },
    } as unknown as BVMState;
  }
}
