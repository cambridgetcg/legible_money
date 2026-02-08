/**
 * StateSyncManager Tests
 *
 * Tests snapshot-based state synchronization: peer snapshot tracking,
 * best snapshot selection, 5-phase sync flow, chunk verification,
 * and the StateSyncProtocolHandler.
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import {
  StateSyncManager,
  StateSyncProtocolHandler,
  STATE_SYNC_CONFIG,
  type SnapshotInfo,
  type StateSyncResponse,
} from "../../../network/protocols/state_sync.js";
import { SnapshotManager } from "../../../state/storage/snapshots.js";
import { MemoryStore } from "../../../state/storage/leveldb.js";
import { sha512_256Hex } from "../../../core/crypto/hashing.js";
import { STORAGE_NAMESPACES } from "../../../state/storage/types.js";
import type {
  StateSnapshot,
  SnapshotChunk,
} from "../../../state/storage/types.js";

// ============ Helpers ============

function textToBytes(text: string): Uint8Array {
  return new TextEncoder().encode(text);
}

/**
 * Write snapshot metadata directly into a MemoryStore so that
 * SnapshotManager.restoreSnapshot() can find it.
 * (applySnapshot imports chunks but never stores metadata — this bridges the gap)
 */
async function storeSnapshotMetadata(
  store: MemoryStore,
  snapshot: StateSnapshot,
): Promise<void> {
  const key = textToBytes(`snapshot:meta:${snapshot.id}`);
  const replacer = (_k: string, v: unknown) =>
    typeof v === "bigint" ? { __bigint__: v.toString() } : v;
  await store.put(key, textToBytes(JSON.stringify(snapshot, replacer)));
}

function createMockSnapshotInfo(
  id: string,
  blockNumber: bigint,
  chunkCount: number = 2,
): SnapshotInfo {
  return {
    id,
    blockNumber,
    blockHash: `hash-${blockNumber}`,
    stateRoot: `root-${blockNumber}`,
    chunkCount,
    compressedSize: BigInt(chunkCount * 1024),
  };
}

/**
 * Create a real snapshot in a store via SnapshotManager,
 * returning the manager, snapshot, and a function to make valid chunks.
 */
async function createRealSnapshot(): Promise<{
  store: MemoryStore;
  manager: SnapshotManager;
  snapshot: StateSnapshot;
}> {
  const store = new MemoryStore();
  const manager = new SnapshotManager(store, {
    compression: "none",
    maxChunkSize: 512,
  });

  // Seed some data
  for (let i = 0; i < 10; i++) {
    const key = new Uint8Array([...STORAGE_NAMESPACES.ACCOUNTS, i]);
    await store.put(key, textToBytes(`value-${i}`));
  }

  const snapshot = await manager.createSnapshot(
    100n,
    "block-hash-100",
    "state-root-100",
    [STORAGE_NAMESPACES.ACCOUNTS],
  );

  return { store, manager, snapshot };
}

// ============ Tests ============

describe.sequential("StateSyncManager", () => {
  let store: MemoryStore;
  let snapshotManager: SnapshotManager;

  beforeEach(() => {
    store = new MemoryStore();
    snapshotManager = new SnapshotManager(store, {
      compression: "none",
      maxChunkSize: 512,
    });
  });

  // ============ shouldUseStateSync ============

  describe("shouldUseStateSync", () => {
    it("should return true when gap exceeds SNAPSHOT_THRESHOLD", () => {
      const manager = new StateSyncManager(snapshotManager);
      expect(manager.shouldUseStateSync(0n, 2000n)).toBe(true);
    });

    it("should return false below threshold", () => {
      const manager = new StateSyncManager(snapshotManager);
      expect(manager.shouldUseStateSync(0n, 500n)).toBe(false);
    });

    it("should return true at exactly SNAPSHOT_THRESHOLD", () => {
      const manager = new StateSyncManager(snapshotManager);
      const threshold = STATE_SYNC_CONFIG.SNAPSHOT_THRESHOLD;
      expect(manager.shouldUseStateSync(0n, threshold)).toBe(true);
    });
  });

  // ============ Peer Snapshot Management ============

  describe("peer snapshot management", () => {
    it("should store peer snapshots and emit event", () => {
      const manager = new StateSyncManager(snapshotManager);
      const handler = vi.fn();
      manager.on("peer:snapshots", handler);

      const snapshots = [createMockSnapshotInfo("snap-1", 100n)];
      manager.addPeerSnapshots("peer-1", snapshots);

      expect(handler).toHaveBeenCalledWith(
        expect.objectContaining({ peerId: "peer-1", snapshots }),
      );
    });

    it("should remove peer snapshots", () => {
      const manager = new StateSyncManager(snapshotManager);
      manager.addPeerSnapshots("peer-1", [
        createMockSnapshotInfo("snap-1", 100n),
      ]);

      manager.removePeer("peer-1");

      // After removing, selectBestSnapshot should return null
      expect(manager.selectBestSnapshot()).toBeNull();
    });
  });

  // ============ selectBestSnapshot ============

  describe("selectBestSnapshot", () => {
    it("should return null when no peers have snapshots", () => {
      const manager = new StateSyncManager(snapshotManager);
      expect(manager.selectBestSnapshot()).toBeNull();
    });

    it("should select highest-block snapshot with >= MIN_SNAPSHOT_PEERS", () => {
      const manager = new StateSyncManager(snapshotManager);

      // Same snapshot from 2 peers (meets MIN_SNAPSHOT_PEERS=2)
      const snap100 = createMockSnapshotInfo("snap-100", 100n);
      const snap200 = createMockSnapshotInfo("snap-200", 200n);

      manager.addPeerSnapshots("peer-1", [snap100, snap200]);
      manager.addPeerSnapshots("peer-2", [snap100, snap200]);

      const result = manager.selectBestSnapshot();
      expect(result).not.toBeNull();
      expect(result!.snapshot.id).toBe("snap-200");
      expect(result!.peers).toHaveLength(2);
    });

    it("should fallback to any snapshot when none meet MIN_SNAPSHOT_PEERS", () => {
      const manager = new StateSyncManager(snapshotManager);

      // Only 1 peer with a snapshot
      manager.addPeerSnapshots("peer-1", [
        createMockSnapshotInfo("snap-1", 500n),
      ]);

      const result = manager.selectBestSnapshot();
      expect(result).not.toBeNull();
      expect(result!.snapshot.id).toBe("snap-1");
      expect(result!.peers).toHaveLength(1);
    });
  });

  // ============ startStateSync — 5 phases ============

  describe("startStateSync - 5 phases", () => {
    it("should proceed through all phases with real snapshot", async () => {
      // Create a real snapshot to serve
      const { manager: srcManager, snapshot: srcSnapshot } =
        await createRealSnapshot();

      // Collect chunks from source
      const chunks: SnapshotChunk[] = [];
      for await (const chunk of srcManager.streamChunks(srcSnapshot.id)) {
        chunks.push(chunk);
      }

      // Set up StateSyncManager with mocked requestFn
      const destStore = new MemoryStore();
      const destSnapshotManager = new SnapshotManager(destStore, {
        compression: "none",
        maxChunkSize: 512,
      });

      const requestFn = vi
        .fn()
        .mockImplementation(
          async (_peerId: string, _protocol: string, request: any) => {
            if (request.type === "snapshot_metadata") {
              return { type: "snapshot_metadata", snapshot: srcSnapshot };
            }
            if (request.type === "chunk") {
              const chunk = chunks[request.chunkIndex];
              return { type: "chunk", chunk };
            }
            return { type: "error", error: "unknown" };
          },
        );

      const syncManager = new StateSyncManager(destSnapshotManager, requestFn);

      // Store snapshot metadata in dest so restoreSnapshot can find it
      await storeSnapshotMetadata(destStore, srcSnapshot);

      // Add peer snapshots
      syncManager.addPeerSnapshots("peer-1", [
        createMockSnapshotInfo(
          srcSnapshot.id,
          srcSnapshot.blockNumber,
          srcSnapshot.chunkCount,
        ),
      ]);
      syncManager.addPeerSnapshots("peer-2", [
        createMockSnapshotInfo(
          srcSnapshot.id,
          srcSnapshot.blockNumber,
          srcSnapshot.chunkCount,
        ),
      ]);

      // Track events
      const startedHandler = vi.fn();
      const selectedHandler = vi.fn();
      const completedHandler = vi.fn();
      syncManager.on("state_sync:started", startedHandler);
      syncManager.on("state_sync:snapshot_selected", selectedHandler);
      syncManager.on("state_sync:completed", completedHandler);

      await syncManager.startStateSync();

      expect(startedHandler).toHaveBeenCalledTimes(1);
      expect(selectedHandler).toHaveBeenCalledTimes(1);
      expect(completedHandler).toHaveBeenCalledTimes(1);

      const status = syncManager.getStatus();
      expect(status.currentPhase).toBe("complete");
      expect(status.downloadedChunks).toBe(srcSnapshot.chunkCount);
      expect(status.verifiedChunks).toBe(srcSnapshot.chunkCount);
      expect(status.appliedChunks).toBe(srcSnapshot.chunkCount);
    });

    it("should throw when no snapshots available", async () => {
      const syncManager = new StateSyncManager(snapshotManager);

      await expect(syncManager.startStateSync()).rejects.toThrow(
        "No snapshots available from peers",
      );
    });

    it("should throw when request function is not set", async () => {
      const syncManager = new StateSyncManager(snapshotManager);

      syncManager.addPeerSnapshots("peer-1", [
        createMockSnapshotInfo("snap-1", 100n),
      ]);

      await expect(syncManager.startStateSync()).rejects.toThrow(
        "Request function not set",
      );
    });
  });

  // ============ Chunk download + verification ============

  describe("chunk download and verification", () => {
    it("should reject chunk with hash mismatch and retry", async () => {
      const { manager: srcManager, snapshot: srcSnapshot } =
        await createRealSnapshot();

      const realChunks: SnapshotChunk[] = [];
      for await (const c of srcManager.streamChunks(srcSnapshot.id)) {
        realChunks.push(c);
      }

      // Create bad chunk (wrong data but we return it for first call)
      const badChunk: SnapshotChunk = {
        ...realChunks[0],
        data: textToBytes("corrupted-data"),
      };

      let callCount = 0;
      const requestFn = vi
        .fn()
        .mockImplementation(
          async (_peerId: string, _protocol: string, request: any) => {
            if (request.type === "snapshot_metadata") {
              return { type: "snapshot_metadata", snapshot: srcSnapshot };
            }
            if (request.type === "chunk") {
              callCount++;
              if (request.chunkIndex === 0 && callCount <= 1) {
                // First attempt: return bad chunk
                return { type: "chunk", chunk: badChunk };
              }
              // Subsequent attempts or other chunks: return real
              return { type: "chunk", chunk: realChunks[request.chunkIndex] };
            }
            return { type: "error", error: "unknown" };
          },
        );

      const destStore = new MemoryStore();
      const destManager = new SnapshotManager(destStore, {
        compression: "none",
        maxChunkSize: 512,
      });

      const syncManager = new StateSyncManager(destManager, requestFn);

      // Store snapshot metadata in dest so restoreSnapshot can find it
      await storeSnapshotMetadata(destStore, srcSnapshot);

      syncManager.addPeerSnapshots("peer-1", [
        createMockSnapshotInfo(
          srcSnapshot.id,
          srcSnapshot.blockNumber,
          srcSnapshot.chunkCount,
        ),
      ]);
      syncManager.addPeerSnapshots("peer-2", [
        createMockSnapshotInfo(
          srcSnapshot.id,
          srcSnapshot.blockNumber,
          srcSnapshot.chunkCount,
        ),
      ]);

      // Override retry delay to 0 for test speed
      const origDelay = STATE_SYNC_CONFIG.CHUNK_RETRY_DELAY;
      STATE_SYNC_CONFIG.CHUNK_RETRY_DELAY = 0;

      try {
        await syncManager.startStateSync();
        // Should succeed after retry
        expect(syncManager.getStatus().currentPhase).toBe("complete");
      } finally {
        STATE_SYNC_CONFIG.CHUNK_RETRY_DELAY = origDelay;
      }
    });
  });

  // ============ Error handling ============

  describe("error handling", () => {
    it("should handle abort during download", async () => {
      const { snapshot: srcSnapshot } = await createRealSnapshot();

      // Make requestFn that hangs to simulate slow download
      const requestFn = vi
        .fn()
        .mockImplementation(
          async (_peerId: string, _protocol: string, request: any) => {
            if (request.type === "snapshot_metadata") {
              return { type: "snapshot_metadata", snapshot: srcSnapshot };
            }
            // Simulate slow chunk download
            await new Promise((r) => setTimeout(r, 500));
            return { type: "error", error: "timeout" };
          },
        );

      const destStore = new MemoryStore();
      const destManager = new SnapshotManager(destStore, {
        compression: "none",
        maxChunkSize: 512,
      });

      const syncManager = new StateSyncManager(destManager, requestFn);
      syncManager.addPeerSnapshots("peer-1", [
        createMockSnapshotInfo(
          srcSnapshot.id,
          srcSnapshot.blockNumber,
          srcSnapshot.chunkCount,
        ),
      ]);
      syncManager.addPeerSnapshots("peer-2", [
        createMockSnapshotInfo(
          srcSnapshot.id,
          srcSnapshot.blockNumber,
          srcSnapshot.chunkCount,
        ),
      ]);

      const origRetry = STATE_SYNC_CONFIG.CHUNK_RETRY_DELAY;
      STATE_SYNC_CONFIG.CHUNK_RETRY_DELAY = 0;

      try {
        // Stop sync quickly
        setTimeout(() => syncManager.stopStateSync(), 10);

        await expect(syncManager.startStateSync()).rejects.toThrow();
      } finally {
        STATE_SYNC_CONFIG.CHUNK_RETRY_DELAY = origRetry;
      }
    });
  });
});

// ============ StateSyncProtocolHandler ============

describe.sequential("StateSyncProtocolHandler", () => {
  let store: MemoryStore;
  let snapshotManager: SnapshotManager;
  let handler: StateSyncProtocolHandler;

  beforeEach(async () => {
    store = new MemoryStore();
    snapshotManager = new SnapshotManager(store, {
      compression: "none",
      maxChunkSize: 512,
    });
    handler = new StateSyncProtocolHandler(snapshotManager);

    // Seed data and create a snapshot
    for (let i = 0; i < 5; i++) {
      const key = new Uint8Array([...STORAGE_NAMESPACES.ACCOUNTS, i]);
      await store.put(key, textToBytes(`value-${i}`));
    }
  });

  describe("handleRequest", () => {
    it("should list snapshots for list_snapshots request", async () => {
      await snapshotManager.createSnapshot(50n, "h50", "r50", [
        STORAGE_NAMESPACES.ACCOUNTS,
      ]);

      const response = await handler.handleRequest({ type: "list_snapshots" });

      expect(response.type).toBe("snapshot_list");
      expect(response.snapshots).toHaveLength(1);
      expect(response.snapshots![0].blockNumber).toBe(50n);
    });

    it("should return metadata for snapshot_metadata request", async () => {
      const snap = await snapshotManager.createSnapshot(
        100n,
        "hash-100",
        "root-100",
        [STORAGE_NAMESPACES.ACCOUNTS],
      );

      const response = await handler.handleRequest({
        type: "snapshot_metadata",
        snapshotId: snap.id,
      });

      expect(response.type).toBe("snapshot_metadata");
      expect(response.snapshot).toBeDefined();
      expect(response.snapshot!.blockNumber).toBe(100n);
    });

    it("should return error for missing snapshotId on metadata request", async () => {
      const response = await handler.handleRequest({
        type: "snapshot_metadata",
      });

      expect(response.type).toBe("error");
      expect(response.error).toContain("snapshotId required");
    });

    it("should return chunk for chunk request", async () => {
      const snap = await snapshotManager.createSnapshot(10n, "h10", "r10", [
        STORAGE_NAMESPACES.ACCOUNTS,
      ]);

      if (snap.chunkCount > 0) {
        const response = await handler.handleRequest({
          type: "chunk",
          snapshotId: snap.id,
          chunkIndex: 0,
        });

        expect(response.type).toBe("chunk");
        expect(response.chunk).toBeDefined();
        expect(response.chunk!.chunkIndex).toBe(0);
      }
    });

    it("should return error for unknown request type", async () => {
      const response = await handler.handleRequest({
        type: "unknown" as any,
      });

      expect(response.type).toBe("error");
      expect(response.error).toContain("Unknown request type");
    });
  });
});
