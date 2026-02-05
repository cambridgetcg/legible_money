/**
 * Unit Tests for State Persistence Layer
 *
 * Tests for StateManager persistence methods and StateIntegration
 * persistence wiring.
 */

import { describe, it, expect, beforeEach, afterEach } from "vitest";
import {
  StateManager,
  type PersistedBlockHeader,
} from "../../../state/state_manager.js";
import { MemoryStore } from "../../../state/storage/leveldb.js";
import type { KVStore } from "../../../state/storage/types.js";

// ============ Test Fixtures ============

function createTestStore(): KVStore {
  return new MemoryStore();
}

// ============ StateManager Persistence Tests ============

describe("StateManager Persistence", () => {
  describe("hasPersistence", () => {
    it("should return true when store is provided", () => {
      const store = createTestStore();
      const stateManager = new StateManager({}, store);
      expect(stateManager.hasPersistence()).toBe(true);
    });

    it("should return false when no store is provided", () => {
      const memoryOnlyManager = new StateManager();
      expect(memoryOnlyManager.hasPersistence()).toBe(false);
    });
  });

  describe("getStore", () => {
    it("should return the underlying store", () => {
      const store = createTestStore();
      const stateManager = new StateManager({}, store);
      expect(stateManager.getStore()).toBe(store);
    });

    it("should return undefined when no store is provided", () => {
      const memoryOnlyManager = new StateManager();
      expect(memoryOnlyManager.getStore()).toBeUndefined();
    });
  });

  describe("persistStateRoots", () => {
    it("should persist state roots for a block", async () => {
      const store = createTestStore();
      const stateManager = new StateManager({}, store);

      // Create some state changes
      stateManager.createAccount("did:lgm:test1", "keyhash1", 1000n);
      stateManager.createAccount("did:lgm:test2", "keyhash2", 2000n);

      // Persist
      await stateManager.persistStateRoots(1n);

      // Verify by loading
      const loaded = await stateManager.loadStateRoots(1n);

      expect(loaded).not.toBeNull();
      expect(loaded!.version).toBe(0n); // Current version before any block application
    });

    it("should do nothing when no store is available", async () => {
      const memoryOnlyManager = new StateManager();

      // Should not throw
      await memoryOnlyManager.persistStateRoots(1n);
    });
  });

  describe("loadStateRoots", () => {
    it("should load previously persisted state roots", async () => {
      const store = createTestStore();
      const stateManager = new StateManager({}, store);

      // Create state and persist
      stateManager.createAccount("did:lgm:test", "keyhash", 1000n);
      const originalComponents = stateManager.getStateRootComponents();

      await stateManager.persistStateRoots(5n);

      // Load and verify
      const loaded = await stateManager.loadStateRoots(5n);

      expect(loaded).not.toBeNull();
      expect(loaded!.accountRoot).toBe(originalComponents.accountRoot);
      expect(loaded!.validatorRoot).toBe(originalComponents.validatorRoot);
      expect(loaded!.knowledgeRoot).toBe(originalComponents.knowledgeRoot);
    });

    it("should return null for non-existent block", async () => {
      const store = createTestStore();
      const stateManager = new StateManager({}, store);

      const loaded = await stateManager.loadStateRoots(999n);
      expect(loaded).toBeNull();
    });

    it("should return null when no store is available", async () => {
      const memoryOnlyManager = new StateManager();
      const loaded = await memoryOnlyManager.loadStateRoots(1n);
      expect(loaded).toBeNull();
    });
  });

  describe("getLatestBlockNumber", () => {
    it("should return the latest persisted block number", async () => {
      const store = createTestStore();
      const stateManager = new StateManager({}, store);

      // Persist multiple blocks
      await stateManager.persistStateRoots(1n);
      await stateManager.persistStateRoots(2n);
      await stateManager.persistStateRoots(5n);

      const latest = await stateManager.getLatestBlockNumber();
      expect(latest).toBe(5n);
    });

    it("should return null when no blocks are persisted", async () => {
      const store = createTestStore();
      const stateManager = new StateManager({}, store);

      const latest = await stateManager.getLatestBlockNumber();
      expect(latest).toBeNull();
    });

    it("should return null when no store is available", async () => {
      const memoryOnlyManager = new StateManager();
      const latest = await memoryOnlyManager.getLatestBlockNumber();
      expect(latest).toBeNull();
    });
  });

  describe("initializeFromStorage", () => {
    it("should initialize state manager from persisted storage", async () => {
      const store = createTestStore();
      const stateManager = new StateManager({}, store);

      // Setup: persist some state
      stateManager.createAccount("did:lgm:init-test", "keyhash", 5000n);

      // Apply a block to advance version
      stateManager.beginTransaction();
      stateManager.commitTransaction();

      // Persist
      await stateManager.persistStateRoots(10n);

      // Create new state manager with same store
      const newManager = new StateManager({}, store);
      const initialized = await newManager.initializeFromStorage();

      expect(initialized).toBe(true);
      // Note: Full state restoration requires per-tree persistence (future enhancement)
    });

    it("should return false when no state is persisted", async () => {
      const freshStore = createTestStore();
      const freshManager = new StateManager({}, freshStore);

      const initialized = await freshManager.initializeFromStorage();
      expect(initialized).toBe(false);
    });

    it("should return false when no store is available", async () => {
      const memoryOnlyManager = new StateManager();
      const initialized = await memoryOnlyManager.initializeFromStorage();
      expect(initialized).toBe(false);
    });
  });

  describe("persistBlockHeader", () => {
    it("should persist a block header", async () => {
      const store = createTestStore();
      const stateManager = new StateManager({}, store);

      const header: PersistedBlockHeader = {
        number: "100",
        hash: "blockhash123",
        parentHash: "parenthash456",
        stateRoot: "stateroot789",
        timestamp: Date.now(),
        proposer: "validator-pubkey",
      };

      await stateManager.persistBlockHeader(header);

      // Load and verify
      const loaded = await stateManager.loadBlockHeader(100n);

      expect(loaded).not.toBeNull();
      expect(loaded!.number).toBe("100");
      expect(loaded!.hash).toBe("blockhash123");
      expect(loaded!.parentHash).toBe("parenthash456");
      expect(loaded!.stateRoot).toBe("stateroot789");
      expect(loaded!.proposer).toBe("validator-pubkey");
    });

    it("should do nothing when no store is available", async () => {
      const memoryOnlyManager = new StateManager();

      const header: PersistedBlockHeader = {
        number: "1",
        hash: "hash",
        parentHash: "parent",
        stateRoot: "state",
        timestamp: Date.now(),
      };

      // Should not throw
      await memoryOnlyManager.persistBlockHeader(header);
    });
  });

  describe("loadBlockHeader", () => {
    it("should load previously persisted block header", async () => {
      const store = createTestStore();
      const stateManager = new StateManager({}, store);

      const header: PersistedBlockHeader = {
        number: "42",
        hash: "hash42",
        parentHash: "hash41",
        stateRoot: "stateroot42",
        timestamp: 1700000000000,
      };

      await stateManager.persistBlockHeader(header);

      const loaded = await stateManager.loadBlockHeader(42n);

      expect(loaded).not.toBeNull();
      expect(loaded!.number).toBe("42");
      expect(loaded!.timestamp).toBe(1700000000000);
    });

    it("should return null for non-existent block", async () => {
      const store = createTestStore();
      const stateManager = new StateManager({}, store);

      const loaded = await stateManager.loadBlockHeader(999n);
      expect(loaded).toBeNull();
    });

    it("should return null when no store is available", async () => {
      const memoryOnlyManager = new StateManager();
      const loaded = await memoryOnlyManager.loadBlockHeader(1n);
      expect(loaded).toBeNull();
    });
  });
});

// ============ Persist and Load Round-Trip Tests ============

describe("Persistence Round-Trip", () => {
  it("should preserve state root components through persist/load cycle", async () => {
    const store = createTestStore();
    const stateManager = new StateManager({}, store);

    // Setup state (create both accounts first to avoid state change from transfer creating account)
    stateManager.createAccount("did:lgm:alice", "keyhash-alice", 10000n);
    stateManager.createAccount("did:lgm:bob", "keyhash-bob", 5000n);

    // Get components AFTER all state changes
    const originalComponents = stateManager.getStateRootComponents();

    // Persist
    await stateManager.persistStateRoots(1n);

    // Load
    const loadedComponents = await stateManager.loadStateRoots(1n);

    expect(loadedComponents).not.toBeNull();
    expect(loadedComponents!.accountRoot).toBe(originalComponents.accountRoot);
    expect(loadedComponents!.validatorRoot).toBe(
      originalComponents.validatorRoot,
    );
    expect(loadedComponents!.knowledgeRoot).toBe(
      originalComponents.knowledgeRoot,
    );
  });

  it("should handle multiple block persistence", async () => {
    const store = createTestStore();
    const stateManager = new StateManager({}, store);

    // Block 1
    stateManager.createAccount("did:lgm:user1", "key1", 1000n);
    const block1Root = stateManager.getStateRootComponents().accountRoot;
    await stateManager.persistStateRoots(1n);

    // Block 2
    stateManager.createAccount("did:lgm:user2", "key2", 2000n);
    const block2Root = stateManager.getStateRootComponents().accountRoot;
    await stateManager.persistStateRoots(2n);

    // Block 3
    stateManager.createAccount("did:lgm:user3", "key3", 3000n);
    const block3Root = stateManager.getStateRootComponents().accountRoot;
    await stateManager.persistStateRoots(3n);

    // Verify all blocks are retrievable
    const loaded1 = await stateManager.loadStateRoots(1n);
    const loaded2 = await stateManager.loadStateRoots(2n);
    const loaded3 = await stateManager.loadStateRoots(3n);

    expect(loaded1!.accountRoot).toBe(block1Root);
    expect(loaded2!.accountRoot).toBe(block2Root);
    expect(loaded3!.accountRoot).toBe(block3Root);

    // Verify latest points to block 3
    const latest = await stateManager.getLatestBlockNumber();
    expect(latest).toBe(3n);
  });

  it("should handle block header persistence with state", async () => {
    const store = createTestStore();
    const stateManager = new StateManager({}, store);
    const blockNumber = 100n;

    // Create state
    stateManager.createAccount("did:lgm:test", "keyhash", 1000n);
    const stateRoot = stateManager.getStateRoot();

    // Persist both
    const header: PersistedBlockHeader = {
      number: blockNumber.toString(),
      hash: "blockhash",
      parentHash: "parenthash",
      stateRoot,
      timestamp: Date.now(),
    };

    await stateManager.persistBlockHeader(header);
    await stateManager.persistStateRoots(blockNumber);

    // Load both
    const loadedHeader = await stateManager.loadBlockHeader(blockNumber);
    const loadedRoots = await stateManager.loadStateRoots(blockNumber);

    expect(loadedHeader).not.toBeNull();
    expect(loadedRoots).not.toBeNull();
    expect(loadedHeader!.stateRoot).toBe(stateRoot);
  });
});

// ============ Backward Compatibility Tests ============

describe("Backward Compatibility", () => {
  it("should work in-memory without persistence", () => {
    const manager = new StateManager();

    // All operations should work without a store
    manager.createAccount("did:lgm:test", "keyhash", 1000n);
    expect(manager.getBalance("did:lgm:test")).toBe(1000n);

    manager.transfer("did:lgm:test", "did:lgm:test2", 500n);
    expect(manager.getBalance("did:lgm:test")).toBe(500n);

    manager.beginTransaction();
    manager.transfer("did:lgm:test", "did:lgm:test2", 100n);
    manager.rollbackTransaction();
    expect(manager.getBalance("did:lgm:test")).toBe(500n);

    const stateRoot = manager.getStateRoot();
    expect(typeof stateRoot).toBe("string");
    expect(stateRoot.length).toBeGreaterThan(0);
  });

  it("should not change behavior when store is not provided", () => {
    const withStore = new StateManager({}, createTestStore());
    const withoutStore = new StateManager();

    // Create same state in both
    withStore.createAccount("did:lgm:test", "keyhash", 1000n);
    withoutStore.createAccount("did:lgm:test", "keyhash", 1000n);

    // State roots should be identical
    expect(withStore.getStateRoot()).toBe(withoutStore.getStateRoot());
  });
});
