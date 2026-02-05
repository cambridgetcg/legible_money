/**
 * LevelDB Storage Tests
 *
 * Tests for the LevelDB storage adapter implementation.
 * Verifies:
 * - Basic CRUD operations
 * - Batch writes
 * - Iteration with range queries
 * - Namespaced storage
 * - Error handling
 *
 * NOTE: LevelDB tests run sequentially to avoid file locking issues
 */

import {
  describe,
  it,
  expect,
  beforeEach,
  afterEach,
  beforeAll,
  afterAll,
} from "vitest";
import { mkdtemp, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import {
  LevelDBStore,
  MemoryStore,
  NamespacedStore,
  createStorage,
  createNamespacedStorage,
  STORAGE_NAMESPACES,
} from "../../../state/storage/index.js";

// ============ Test Helpers ============

function textToBytes(text: string): Uint8Array {
  return new TextEncoder().encode(text);
}

function bytesToText(bytes: Uint8Array): string {
  return new TextDecoder().decode(bytes);
}

// ============ MemoryStore Tests ============

describe("MemoryStore", () => {
  let store: MemoryStore;

  beforeEach(() => {
    store = new MemoryStore();
  });

  afterEach(async () => {
    await store.close();
  });

  describe("basic operations", () => {
    it("should store and retrieve a value", async () => {
      const key = textToBytes("test-key");
      const value = textToBytes("test-value");

      await store.put(key, value);
      const result = await store.get(key);

      expect(result).not.toBeNull();
      expect(bytesToText(result!)).toBe("test-value");
    });

    it("should return null for non-existent key", async () => {
      const result = await store.get(textToBytes("missing"));
      expect(result).toBeNull();
    });

    it("should check key existence", async () => {
      const key = textToBytes("exists");
      await store.put(key, textToBytes("value"));

      expect(await store.has(key)).toBe(true);
      expect(await store.has(textToBytes("missing"))).toBe(false);
    });

    it("should delete a key", async () => {
      const key = textToBytes("to-delete");
      await store.put(key, textToBytes("value"));
      expect(await store.has(key)).toBe(true);

      await store.delete(key);
      expect(await store.has(key)).toBe(false);
    });
  });

  describe("batch operations", () => {
    it("should execute batch writes atomically", async () => {
      const batch = store.batch();

      batch
        .put(textToBytes("key1"), textToBytes("value1"))
        .put(textToBytes("key2"), textToBytes("value2"))
        .put(textToBytes("key3"), textToBytes("value3"));

      await batch.write();

      expect(await store.has(textToBytes("key1"))).toBe(true);
      expect(await store.has(textToBytes("key2"))).toBe(true);
      expect(await store.has(textToBytes("key3"))).toBe(true);
    });

    it("should support delete in batch", async () => {
      await store.put(textToBytes("to-delete"), textToBytes("value"));

      const batch = store.batch();
      batch.delete(textToBytes("to-delete"));
      await batch.write();

      expect(await store.has(textToBytes("to-delete"))).toBe(false);
    });
  });

  describe("iteration", () => {
    beforeEach(async () => {
      // Insert some ordered keys
      for (let i = 0; i < 10; i++) {
        const key = textToBytes(`key-${i.toString().padStart(2, "0")}`);
        await store.put(key, textToBytes(`value-${i}`));
      }
    });

    it("should iterate all keys", async () => {
      const keys: string[] = [];
      for await (const { key } of store.iterator()) {
        keys.push(bytesToText(key));
      }
      expect(keys).toHaveLength(10);
    });

    it("should support range queries with gte", async () => {
      const keys: string[] = [];
      for await (const { key } of store.iterator({
        gte: textToBytes("key-05"),
      })) {
        keys.push(bytesToText(key));
      }
      // Should get keys 05-09
      expect(keys).toHaveLength(5);
      expect(keys[0]).toBe("key-05");
    });

    it("should support range queries with lt", async () => {
      const keys: string[] = [];
      for await (const { key } of store.iterator({
        lt: textToBytes("key-05"),
      })) {
        keys.push(bytesToText(key));
      }
      // Should get keys 00-04
      expect(keys).toHaveLength(5);
      expect(keys[4]).toBe("key-04");
    });

    it("should support limit", async () => {
      const keys: string[] = [];
      for await (const { key } of store.iterator({ limit: 3 })) {
        keys.push(bytesToText(key));
      }
      expect(keys).toHaveLength(3);
    });
  });

  describe("stats", () => {
    it("should return storage statistics", async () => {
      await store.put(textToBytes("key1"), textToBytes("value1"));
      await store.put(textToBytes("key2"), textToBytes("value2"));

      const stats = await store.stats();

      expect(stats.keyCount).toBe(2n);
      expect(stats.sizeBytes).toBeGreaterThan(0n);
    });
  });
});

// ============ LevelDBStore Tests ============

// Helper to create an isolated test store
async function withLevelDBStore<T>(
  testFn: (store: LevelDBStore, tempDir: string) => Promise<T>,
): Promise<T> {
  const tempDir = await mkdtemp(join(tmpdir(), "leveldb-test-"));
  const store = await LevelDBStore.open({
    type: "leveldb",
    path: tempDir,
  });
  try {
    return await testFn(store, tempDir);
  } finally {
    await store.close();
    await rm(tempDir, { recursive: true, force: true });
  }
}

describe("LevelDBStore", () => {
  describe("basic operations", () => {
    it("should store and retrieve a value", async () => {
      await withLevelDBStore(async (store) => {
        const key = textToBytes("test-key");
        const value = textToBytes("test-value");

        await store.put(key, value);
        const result = await store.get(key);

        expect(result).not.toBeNull();
        expect(bytesToText(result!)).toBe("test-value");
      });
    });

    it("should return null for non-existent key", async () => {
      await withLevelDBStore(async (store) => {
        const result = await store.get(textToBytes("missing"));
        expect(result).toBeNull();
      });
    });

    it("should check key existence", async () => {
      await withLevelDBStore(async (store) => {
        const key = textToBytes("exists");
        await store.put(key, textToBytes("value"));

        expect(await store.has(key)).toBe(true);
        expect(await store.has(textToBytes("missing"))).toBe(false);
      });
    });

    it("should delete a key", async () => {
      await withLevelDBStore(async (store) => {
        const key = textToBytes("to-delete");
        await store.put(key, textToBytes("value"));
        expect(await store.has(key)).toBe(true);

        await store.delete(key);
        expect(await store.has(key)).toBe(false);
      });
    });

    it("should handle binary data correctly", async () => {
      await withLevelDBStore(async (store) => {
        const key = new Uint8Array([0x00, 0xff, 0x80, 0x7f]);
        const value = new Uint8Array([0xde, 0xad, 0xbe, 0xef]);

        await store.put(key, value);
        const result = await store.get(key);

        expect(result).not.toBeNull();
        expect(Array.from(result!)).toEqual([0xde, 0xad, 0xbe, 0xef]);
      });
    });
  });

  describe("batch operations", () => {
    it("should execute batch writes atomically", async () => {
      await withLevelDBStore(async (store) => {
        const batch = store.batch();

        batch
          .put(textToBytes("key1"), textToBytes("value1"))
          .put(textToBytes("key2"), textToBytes("value2"))
          .put(textToBytes("key3"), textToBytes("value3"));

        await batch.write();

        expect(await store.has(textToBytes("key1"))).toBe(true);
        expect(await store.has(textToBytes("key2"))).toBe(true);
        expect(await store.has(textToBytes("key3"))).toBe(true);
      });
    });

    it("should support mixed put and delete in batch", async () => {
      await withLevelDBStore(async (store) => {
        await store.put(textToBytes("existing"), textToBytes("old-value"));

        const batch = store.batch();
        batch.put(textToBytes("new-key"), textToBytes("new-value"));
        batch.delete(textToBytes("existing"));
        await batch.write();

        expect(await store.has(textToBytes("new-key"))).toBe(true);
        expect(await store.has(textToBytes("existing"))).toBe(false);
      });
    });

    it("should clear batch operations", async () => {
      await withLevelDBStore(async (store) => {
        const batch = store.batch();
        batch.put(textToBytes("key1"), textToBytes("value1"));
        batch.clear();
        await batch.write();

        expect(await store.has(textToBytes("key1"))).toBe(false);
      });
    });
  });

  describe("iteration", () => {
    it("should iterate all keys", async () => {
      await withLevelDBStore(async (store) => {
        // Insert some ordered keys
        for (let i = 0; i < 10; i++) {
          const key = textToBytes(`key-${i.toString().padStart(2, "0")}`);
          await store.put(key, textToBytes(`value-${i}`));
        }

        const keys: string[] = [];
        for await (const { key } of store.iterator()) {
          keys.push(bytesToText(key));
        }
        expect(keys).toHaveLength(10);
      });
    });

    it("should support range queries with gte", async () => {
      await withLevelDBStore(async (store) => {
        for (let i = 0; i < 10; i++) {
          const key = textToBytes(`key-${i.toString().padStart(2, "0")}`);
          await store.put(key, textToBytes(`value-${i}`));
        }

        const keys: string[] = [];
        for await (const { key } of store.iterator({
          gte: textToBytes("key-05"),
        })) {
          keys.push(bytesToText(key));
        }
        expect(keys).toHaveLength(5);
        expect(keys[0]).toBe("key-05");
      });
    });

    it("should support range queries with lt", async () => {
      await withLevelDBStore(async (store) => {
        for (let i = 0; i < 10; i++) {
          const key = textToBytes(`key-${i.toString().padStart(2, "0")}`);
          await store.put(key, textToBytes(`value-${i}`));
        }

        const keys: string[] = [];
        for await (const { key } of store.iterator({
          lt: textToBytes("key-05"),
        })) {
          keys.push(bytesToText(key));
        }
        expect(keys).toHaveLength(5);
      });
    });

    it("should support reverse iteration", async () => {
      await withLevelDBStore(async (store) => {
        for (let i = 0; i < 10; i++) {
          const key = textToBytes(`key-${i.toString().padStart(2, "0")}`);
          await store.put(key, textToBytes(`value-${i}`));
        }

        const keys: string[] = [];
        for await (const { key } of store.iterator({
          reverse: true,
          limit: 3,
        })) {
          keys.push(bytesToText(key));
        }
        expect(keys).toHaveLength(3);
        expect(keys[0]).toBe("key-09");
        expect(keys[1]).toBe("key-08");
        expect(keys[2]).toBe("key-07");
      });
    });
  });

  describe("persistence", () => {
    it("should persist data across close and reopen", async () => {
      const tempDir = await mkdtemp(join(tmpdir(), "leveldb-persist-"));
      try {
        // Open, write, close
        let store = await LevelDBStore.open({
          type: "leveldb",
          path: tempDir,
        });
        await store.put(textToBytes("persistent"), textToBytes("data"));
        await store.close();

        // Reopen and verify
        store = await LevelDBStore.open({
          type: "leveldb",
          path: tempDir,
        });
        const result = await store.get(textToBytes("persistent"));
        expect(result).not.toBeNull();
        expect(bytesToText(result!)).toBe("data");
        await store.close();
      } finally {
        await rm(tempDir, { recursive: true, force: true });
      }
    });
  });

  describe("stats", () => {
    it("should return storage statistics", async () => {
      await withLevelDBStore(async (store) => {
        await store.put(textToBytes("key1"), textToBytes("value1"));
        await store.put(textToBytes("key2"), textToBytes("value2"));

        const stats = await store.stats();

        expect(stats.keyCount).toBe(2n);
        expect(stats.sizeBytes).toBeGreaterThan(0n);
      });
    });
  });

  describe("properties", () => {
    it("should report path", async () => {
      await withLevelDBStore(async (store, tempDir) => {
        expect(store.path).toBe(tempDir);
      });
    });

    it("should report open status", async () => {
      await withLevelDBStore(async (store) => {
        expect(store.isOpen).toBe(true);
      });
    });
  });
});

// ============ NamespacedStore Tests ============

describe("NamespacedStore", () => {
  it("should isolate keys between namespaces", async () => {
    const baseStore = new MemoryStore();
    const accountStore = createNamespacedStorage(
      baseStore,
      STORAGE_NAMESPACES.ACCOUNTS,
    );
    const knowledgeStore = createNamespacedStorage(
      baseStore,
      STORAGE_NAMESPACES.KNOWLEDGE,
    );

    try {
      const key = textToBytes("shared-key");

      await accountStore.put(key, textToBytes("account-value"));
      await knowledgeStore.put(key, textToBytes("knowledge-value"));

      const accountResult = await accountStore.get(key);
      const knowledgeResult = await knowledgeStore.get(key);

      expect(bytesToText(accountResult!)).toBe("account-value");
      expect(bytesToText(knowledgeResult!)).toBe("knowledge-value");
    } finally {
      await baseStore.close();
    }
  });

  it("should iterate only within namespace", async () => {
    const baseStore = new MemoryStore();
    const accountStore = createNamespacedStorage(
      baseStore,
      STORAGE_NAMESPACES.ACCOUNTS,
    );
    const knowledgeStore = createNamespacedStorage(
      baseStore,
      STORAGE_NAMESPACES.KNOWLEDGE,
    );

    try {
      await accountStore.put(textToBytes("acc-1"), textToBytes("v1"));
      await accountStore.put(textToBytes("acc-2"), textToBytes("v2"));
      await knowledgeStore.put(textToBytes("know-1"), textToBytes("v3"));

      const accountKeys: string[] = [];
      for await (const { key } of accountStore.iterator()) {
        accountKeys.push(bytesToText(key));
      }

      expect(accountKeys).toHaveLength(2);
      expect(accountKeys).toContain("acc-1");
      expect(accountKeys).toContain("acc-2");
      expect(accountKeys).not.toContain("know-1");
    } finally {
      await baseStore.close();
    }
  });
});

// ============ createStorage Factory Tests ============

describe("createStorage", () => {
  it("should create a MemoryStore for memory type", async () => {
    const store = await createStorage({ type: "memory" });
    expect(store).toBeInstanceOf(MemoryStore);
    await store.close();
  });

  it("should create a LevelDBStore for leveldb type", async () => {
    const tempDir = await mkdtemp(join(tmpdir(), "leveldb-factory-"));
    try {
      const store = await createStorage({
        type: "leveldb",
        path: tempDir,
      });
      expect(store).toBeInstanceOf(LevelDBStore);
      await store.close();
    } finally {
      await rm(tempDir, { recursive: true, force: true });
    }
  });

  it("should throw for unknown storage type", async () => {
    await expect(createStorage({ type: "unknown" } as any)).rejects.toThrow(
      "Unknown storage type",
    );
  });
});
