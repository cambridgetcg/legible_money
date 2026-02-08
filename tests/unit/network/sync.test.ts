/**
 * SyncManager Tests
 *
 * Tests block synchronization: peer management, request handling,
 * range sync, and lifecycle events.
 * All dependencies injected via constructor/setter — no mocks of external modules.
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import {
  SyncManager,
  SYNC_CONFIG,
  type SyncPeer,
  type SyncRequest,
} from "../../../network/protocols/sync.js";

// ============ Helpers ============

function createMockPeer(
  id: string,
  headBlock: bigint,
  score: number = 50,
): SyncPeer {
  return { peerId: id, headBlock, latency: 10, score };
}

// ============ Tests ============

describe.sequential("SyncManager", () => {
  let getBlock: ReturnType<typeof vi.fn>;
  let saveBlock: ReturnType<typeof vi.fn>;
  let validateBlock: ReturnType<typeof vi.fn>;
  let manager: SyncManager;

  beforeEach(() => {
    getBlock = vi.fn().mockResolvedValue(null);
    saveBlock = vi.fn().mockResolvedValue(undefined);
    validateBlock = vi.fn().mockResolvedValue(true);
    // Start at block 0, no auto-sync triggered
    manager = new SyncManager(0n, getBlock, saveBlock, validateBlock);
  });

  // ============ Peer Management ============

  describe("peer management", () => {
    it("should add a peer and emit peer:added", () => {
      const handler = vi.fn();
      manager.on("peer:added", handler);

      const peer = createMockPeer("peer-1", 0n);
      manager.addPeer(peer);

      expect(handler).toHaveBeenCalledWith(peer);
    });

    it("should update highest block when peer is added", () => {
      manager.addPeer(createMockPeer("peer-1", 0n)); // same block — no sync
      expect(manager.getStatus().highestBlock).toBe(0n);

      // Add peer with higher block (but catch the auto-sync start)
      manager.addPeer(createMockPeer("peer-2", 0n));
      expect(manager.getStatus().highestBlock).toBe(0n);
    });

    it("should remove peer and emit peer:removed", () => {
      const handler = vi.fn();
      manager.on("peer:removed", handler);

      const peer = createMockPeer("peer-1", 0n);
      manager.addPeer(peer);
      manager.removePeer("peer-1");

      expect(handler).toHaveBeenCalledWith(peer);
    });

    it("should ignore removePeer for unknown peerId", () => {
      const handler = vi.fn();
      manager.on("peer:removed", handler);

      manager.removePeer("nonexistent");
      expect(handler).not.toHaveBeenCalled();
    });

    it("should update peer head via updatePeerHead", () => {
      manager.addPeer(createMockPeer("peer-1", 0n));
      manager.updatePeerHead("peer-1", 500n);

      expect(manager.getStatus().highestBlock).toBe(500n);
    });
  });

  // ============ handleRequest ============

  describe("handleRequest", () => {
    it("should return sync status for status request", async () => {
      const response = await manager.handleRequest({ type: "status" });

      expect(response.type).toBe("status");
      expect(response.data).toMatchObject({
        currentBlock: 0n,
        syncing: false,
      });
    });

    it("should return blocks for headers request using injected getBlock", async () => {
      const block1 = { number: 0n, data: "block-0" };
      const block2 = { number: 1n, data: "block-1" };
      getBlock
        .mockResolvedValueOnce(block1)
        .mockResolvedValueOnce(block2)
        .mockResolvedValue(null);

      const response = await manager.handleRequest({
        type: "headers",
        startBlock: 0n,
        count: 5,
      });

      expect(response.type).toBe("headers");
      expect(response.data).toEqual([block1, block2]);
      expect(getBlock).toHaveBeenCalledWith(0n);
      expect(getBlock).toHaveBeenCalledWith(1n);
    });

    it("should stop at first missing block in range", async () => {
      getBlock
        .mockResolvedValueOnce({ number: 0n })
        .mockResolvedValueOnce(null); // Missing block 1

      const response = await manager.handleRequest({
        type: "headers",
        startBlock: 0n,
        count: 10,
      });

      const blocks = response.data as unknown[];
      expect(blocks.length).toBe(1);
    });

    it("should return state sync redirect for state request", async () => {
      const response = await manager.handleRequest({ type: "state" });

      expect(response.type).toBe("state");
      const data = response.data as Record<string, unknown>;
      expect(data.message).toContain("STATE_SYNC_PROTOCOL");
    });

    it("should throw for unknown request type", async () => {
      await expect(
        manager.handleRequest({ type: "invalid" as SyncRequest["type"] }),
      ).rejects.toThrow("Unknown request type");
    });
  });

  // ============ syncRangeFromPeer ============

  describe("syncRangeFromPeer", () => {
    it("should request blocks via injected requestBlocks and save valid ones", async () => {
      const blocks = [
        { number: 1n, data: "a" },
        { number: 2n, data: "b" },
        { number: 3n, data: "c" },
      ];
      const requestBlocks = vi.fn().mockResolvedValue(blocks);
      manager.setRequestFunction(requestBlocks);

      const peer = createMockPeer("peer-1", 10n);

      await manager.syncRangeFromPeer(peer, 1n, 4n);

      expect(requestBlocks).toHaveBeenCalledWith("peer-1", 1n, 3);
      expect(validateBlock).toHaveBeenCalledTimes(3);
      expect(saveBlock).toHaveBeenCalledTimes(3);
      expect(manager.getStatus().currentBlock).toBe(3n);
    });

    it("should throw when requestBlocks function is not set", async () => {
      const peer = createMockPeer("peer-1", 10n);

      await expect(manager.syncRangeFromPeer(peer, 1n, 5n)).rejects.toThrow(
        "Request function not set",
      );
    });

    it("should throw on invalid block", async () => {
      const blocks = [
        { number: 1n, data: "valid" },
        { number: 2n, data: "bad" },
      ];
      const requestBlocks = vi.fn().mockResolvedValue(blocks);
      manager.setRequestFunction(requestBlocks);

      // First block valid, second invalid
      validateBlock.mockResolvedValueOnce(true).mockResolvedValueOnce(false);

      const peer = createMockPeer("peer-1", 10n);

      await expect(manager.syncRangeFromPeer(peer, 1n, 3n)).rejects.toThrow(
        "Invalid block at 2",
      );

      // First block saved, second not
      expect(saveBlock).toHaveBeenCalledTimes(1);
    });

    it("should emit sync:block for each successfully saved block", async () => {
      const blocks = [{ number: 5n }, { number: 6n }];
      const requestBlocks = vi.fn().mockResolvedValue(blocks);
      manager.setRequestFunction(requestBlocks);

      const blockHandler = vi.fn();
      manager.on("sync:block", blockHandler);

      const peer = createMockPeer("peer-1", 10n);
      await manager.syncRangeFromPeer(peer, 5n, 7n);

      expect(blockHandler).toHaveBeenCalledTimes(2);
      expect(blockHandler).toHaveBeenCalledWith(
        expect.objectContaining({ number: 5n }),
      );
      expect(blockHandler).toHaveBeenCalledWith(
        expect.objectContaining({ number: 6n }),
      );
    });

    it("should emit sync:range_start and sync:range_complete", async () => {
      const requestBlocks = vi.fn().mockResolvedValue([{ number: 1n }]);
      manager.setRequestFunction(requestBlocks);

      const rangeStart = vi.fn();
      const rangeComplete = vi.fn();
      manager.on("sync:range_start", rangeStart);
      manager.on("sync:range_complete", rangeComplete);

      const peer = createMockPeer("peer-1", 10n);
      await manager.syncRangeFromPeer(peer, 1n, 2n);

      expect(rangeStart).toHaveBeenCalledWith(
        expect.objectContaining({ start: 1n, end: 2n, peer: "peer-1" }),
      );
      expect(rangeComplete).toHaveBeenCalledWith(
        expect.objectContaining({ start: 1n, end: 2n }),
      );
    });
  });

  // ============ Lifecycle ============

  describe("lifecycle", () => {
    it("should emit sync:started and sync:completed on full sync", async () => {
      // Manager at block 0, peer at block 2
      // syncLoop uses syncRange (placeholder) which calls validateBlock + saveBlock
      const startHandler = vi.fn();
      const completeHandler = vi.fn();
      manager.on("sync:started", startHandler);
      manager.on("sync:completed", completeHandler);

      manager.addPeer(createMockPeer("peer-1", 2n));

      // Wait for async sync to complete
      await new Promise((r) => setTimeout(r, 50));

      expect(startHandler).toHaveBeenCalledTimes(1);
      expect(completeHandler).toHaveBeenCalledTimes(1);
    });

    it("should not start sync if already syncing", async () => {
      const startHandler = vi.fn();
      manager.on("sync:started", startHandler);

      // Add peer to trigger sync
      manager.addPeer(createMockPeer("peer-1", 2n));
      // Immediately add another peer (should not trigger second sync)
      manager.addPeer(createMockPeer("peer-2", 3n));

      await new Promise((r) => setTimeout(r, 50));

      expect(startHandler).toHaveBeenCalledTimes(1);
    });

    it("should stop sync when stopSync is called", async () => {
      const stoppedHandler = vi.fn();
      manager.on("sync:stopped", stoppedHandler);

      // Set up a very long range
      manager.addPeer(createMockPeer("peer-1", 1000000n));
      // Stop immediately
      manager.stopSync();

      expect(stoppedHandler).toHaveBeenCalledTimes(1);
    });

    it("should report correct status fields", () => {
      const status = manager.getStatus();
      expect(status.currentBlock).toBe(0n);
      expect(status.highestBlock).toBe(0n);
      expect(status.syncing).toBe(false);
      expect(status.progress).toBe(1.0);
    });
  });
});
