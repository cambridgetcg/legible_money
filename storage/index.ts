/**
 * BVM Persistence Layer
 *
 * Provides persistent storage for BVM state including:
 * - Contract bytecode and storage
 * - Account balances and nonces
 * - Agent state (beliefs, relationships, goals)
 * - State snapshots for fast sync
 *
 * PP HIERARCHY:
 *   TRUTH: Durable, verifiable state persistence
 *   UNDERSTANDING: Clear APIs for different storage needs
 *   BEAUTY: Elegant abstraction over storage backends
 *   JUSTICE: Fair resource management
 *   CREATIVITY: Extensible for future backends
 *
 * @module storage
 *
 * @example
 * // Create an in-memory store for testing
 * const store = await BVMStateStore.open('memory');
 *
 * // Save a contract
 * await store.saveContract('0x1234...', bytecode);
 *
 * // Use transactions for atomicity
 * store.beginTransaction();
 * await store.saveAccount('did:lgm:alice', account);
 * await store.saveAgent('did:lgm:alice', agent);
 * const root = await store.commit();
 *
 * @example
 * // Create a persistent store for production
 * const store = await BVMStateStore.open({
 *   path: './data/bvm',
 *   cacheSize: 16 * 1024 * 1024,
 * });
 */

// ============ Core Types ============

export type {
  StateStore,
  BatchOp,
  PersistenceConfig,
  StorageStats,
  BVMStoragePrefix,
  BVMStorageErrorCode,
} from "./types.js";

export {
  BVM_STORAGE_PREFIXES,
  DEFAULT_PERSISTENCE_CONFIG,
  BVMStorageError,
  createKey,
  addressToKey,
  storageSlotKey,
  heightToKey,
  keyToHeight,
  encodeValue,
  decodeValue,
  bigIntReplacer,
  bigIntReviver,
} from "./types.js";

// ============ Storage Backends ============

export {
  MemoryStateStore,
  LevelDBStateStore,
  createStateStore,
} from "./leveldb.js";

// ============ BVM State Store ============

export { BVMStateStore } from "./bvm_store.js";

// ============ Factory Functions ============

import { BVMStateStore } from "./bvm_store.js";
import type { PersistenceConfig } from "./types.js";

/**
 * Create a BVM state store
 *
 * @param config - Configuration or 'memory' for in-memory store
 * @returns Promise resolving to a BVMStateStore
 *
 * @example
 * // In-memory for testing
 * const store = await createBVMStore('memory');
 *
 * @example
 * // Persistent for production
 * const store = await createBVMStore({ path: './data/bvm' });
 */
export async function createBVMStore(
  config: Partial<PersistenceConfig> | "memory" = "memory",
): Promise<BVMStateStore> {
  return BVMStateStore.open(config);
}

/**
 * Create a BVM state store with LevelDB persistence
 *
 * @param path - Path to the storage directory
 * @param options - Additional options
 * @returns Promise resolving to a BVMStateStore
 *
 * @example
 * const store = await createPersistentBVMStore('./data/bvm');
 */
export async function createPersistentBVMStore(
  path: string,
  options: Omit<PersistenceConfig, "path"> = {},
): Promise<BVMStateStore> {
  return BVMStateStore.open({ path, ...options });
}

/**
 * Create an in-memory BVM state store (for testing)
 *
 * @returns Promise resolving to a BVMStateStore
 *
 * @example
 * const store = await createMemoryBVMStore();
 */
export async function createMemoryBVMStore(): Promise<BVMStateStore> {
  return BVMStateStore.open("memory");
}
