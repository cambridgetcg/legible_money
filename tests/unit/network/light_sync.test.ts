/**
 * LightSyncManager Tests
 *
 * Tests lightweight synchronization for chatnode instances:
 * header sync, knowledge tree sync, on-demand proof retrieval,
 * live updates, cache management, and LightSyncProtocolHandler.
 *
 * NOTE: syncHeaders() has an async while loop that can starve the
 * macrotask queue. All requestFn mocks must yield via setTimeout(0).
 * Tests that need knowledge sync set peer headBlock > available headers
 * and abort from within the requestFn when headers are exhausted.
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import {
  LightSyncManager,
  LightSyncProtocolHandler,
  DEFAULT_LIGHT_SYNC_CONFIG,
  LIGHT_SYNC_TOPICS,
  type BlockHeader,
  type KnowledgeTreeNode,
  type KnowledgeProof,
  type KnowledgeUpdate,
  type LightSyncConfig,
} from "../../../network/protocols/light_sync.js";
import type { SyncPeer } from "../../../network/protocols/sync.js";
import {
  sha512_256Hex,
  domainHash,
  HASH_DOMAINS,
} from "../../../core/crypto/hashing.js";
import { bytesToHex, hexToBytes } from "../../../core/crypto/keys.js";

// ============ Helpers ============

/** Yield to macrotask queue so setTimeout callbacks can fire */
function yieldToMacrotasks(): Promise<void> {
  return new Promise((r) => setTimeout(r, 0));
}

function createMockPeer(
  id: string,
  headBlock: bigint,
  score: number = 50,
): SyncPeer {
  return { peerId: id, headBlock, latency: 10, score };
}

function createMockHeader(
  number: bigint,
  parentHash: string = `hash-${number - 1n}`,
  knowledgeRoot?: string,
): BlockHeader {
  return {
    number,
    hash: `hash-${number}`,
    parentHash,
    stateRoot: `state-root-${number}`,
    knowledgeRoot: knowledgeRoot ?? `knowledge-root-${number}`,
    timestamp: Date.now(),
    producer: "validator-1",
  };
}

function createMockKnowledgeNode(
  hash: string,
  left?: string,
  right?: string,
): KnowledgeTreeNode {
  return { hash, left, right };
}

/**
 * Build a valid KnowledgeProof that passes verifyKnowledgeProof().
 * ZD-C03: Uses domain-separated binary hashing with direction tracking.
 */
function buildValidProof(factId: string): {
  proof: KnowledgeProof;
  root: string;
} {
  const leafHash = sha512_256Hex(Buffer.from(`leaf-${factId}`));
  const sibling = sha512_256Hex(Buffer.from("sibling-0"));

  // Compute leaf hash: domainHash(STATE, 0x00 || leafBytes)
  const leafBytes = hexToBytes(leafHash);
  const leafPrefixed = new Uint8Array(1 + leafBytes.length);
  leafPrefixed[0] = 0x00;
  leafPrefixed.set(leafBytes, 1);
  const hashedLeaf = bytesToHex(domainHash(HASH_DOMAINS.STATE, leafPrefixed));

  // Compute internal node: domainHash(STATE, 0x01 || current || sibling) (direction=0: sibling on right)
  const currentBytes = hexToBytes(hashedLeaf);
  const siblingBytes = hexToBytes(sibling);
  const combined = new Uint8Array(
    1 + currentBytes.length + siblingBytes.length,
  );
  combined[0] = 0x01;
  combined.set(currentBytes, 1);
  combined.set(siblingBytes, 1 + currentBytes.length);
  const root = bytesToHex(domainHash(HASH_DOMAINS.STATE, combined));

  return {
    proof: {
      factId,
      proof: [sibling],
      directions: [0], // 0 = sibling on right
      leaf: { hash: leafHash },
      root,
      blockNumber: 10n,
    },
    root,
  };
}

/**
 * Create a requestFn for header-only sync (syncKnowledgeTree: false).
 * Peer headBlock should equal maxBlock so header loop exits cleanly.
 */
function createHeaderOnlyRequestFn(maxBlock: bigint) {
  const headers: BlockHeader[] = [];
  for (let i = 1n; i <= maxBlock; i++) {
    headers.push(createMockHeader(i, i === 1n ? "genesis" : `hash-${i - 1n}`));
  }

  return vi
    .fn()
    .mockImplementation(
      async (_peerId: string, _protocol: string, request: any) => {
        await yieldToMacrotasks();
        if (request.type === "headers") {
          const start = Number(request.startBlock ?? 0n);
          const count = request.count ?? 100;
          const slice = headers.filter(
            (h) =>
              h.number >= BigInt(start) && h.number < BigInt(start + count),
          );
          return { type: "headers", data: slice };
        }
        return { type: "headers", data: [] };
      },
    );
}

/**
 * Create a requestFn for full sync (headers + knowledge tree).
 * Peer headBlock should be > maxBlock so selectBestPeer() works for knowledge.
 * When headers are exhausted, calls manager.stopSync() to break the header loop.
 */
function createFullSyncRequestFn(
  maxBlock: bigint,
  knowledgeRootHash: string,
  mgr: LightSyncManager,
  extraHandlers?: (request: any) => any,
) {
  const headers: BlockHeader[] = [];
  for (let i = 1n; i <= maxBlock; i++) {
    headers.push(
      createMockHeader(
        i,
        i === 1n ? "genesis" : `hash-${i - 1n}`,
        i === maxBlock ? knowledgeRootHash : undefined,
      ),
    );
  }

  return vi
    .fn()
    .mockImplementation(
      async (_peerId: string, _protocol: string, request: any) => {
        await yieldToMacrotasks();

        if (request.type === "headers") {
          const start = Number(request.startBlock ?? 0n);
          const count = request.count ?? 100;
          const slice = headers.filter(
            (h) =>
              h.number >= BigInt(start) && h.number < BigInt(start + count),
          );
          if (slice.length === 0) {
            // No more headers — abort so knowledge sync can proceed
            mgr.stopSync();
            return { type: "headers", data: [] };
          }
          return { type: "headers", data: slice };
        }
        if (request.type === "knowledge_root") {
          return {
            type: "knowledge_root",
            data: createMockKnowledgeNode(knowledgeRootHash),
          };
        }
        if (request.type === "knowledge_nodes") {
          return { type: "knowledge_nodes", data: [] };
        }

        // Allow extra handlers (e.g., knowledge_proof)
        if (extraHandlers) {
          const result = extraHandlers(request);
          if (result) return result;
        }

        return { type: "headers", data: [] };
      },
    );
}

// ============ Tests ============

describe.sequential("LightSyncManager", () => {
  let saveHeader: ReturnType<typeof vi.fn>;
  let getHeaderFn: ReturnType<typeof vi.fn>;
  let validateHeader: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    saveHeader = vi.fn().mockResolvedValue(undefined);
    getHeaderFn = vi.fn().mockResolvedValue(undefined);
    validateHeader = vi.fn().mockResolvedValue(true);
  });

  /** Create a manager configured for header-only sync */
  function createHeaderOnlyManager(): LightSyncManager {
    return new LightSyncManager(0n, saveHeader, getHeaderFn, validateHeader, {
      ...DEFAULT_LIGHT_SYNC_CONFIG,
      syncKnowledgeTree: false,
      subscribeLiveUpdates: false,
    });
  }

  /** Create a manager configured for full sync */
  function createFullManager(
    config?: Partial<LightSyncConfig>,
  ): LightSyncManager {
    return new LightSyncManager(0n, saveHeader, getHeaderFn, validateHeader, {
      ...DEFAULT_LIGHT_SYNC_CONFIG,
      ...config,
    });
  }

  // ============ Status and Config ============

  describe("status and config", () => {
    it("should initialize with correct default status", () => {
      const manager = createHeaderOnlyManager();
      const status = manager.getStatus();
      expect(status.currentBlock).toBe(0n);
      expect(status.highestBlock).toBe(0n);
      expect(status.syncing).toBe(false);
      expect(status.progress).toBe(1.0);
      expect(status.headersSynced).toBe(false);
      expect(status.knowledgeRootSynced).toBe(false);
      expect(status.subscribedToUpdates).toBe(false);
      expect(status.pendingProofs).toBe(0);
    });

    it("should not be synced initially", () => {
      const manager = createHeaderOnlyManager();
      expect(manager.isSynced()).toBe(false);
    });
  });

  // ============ Peer Management ============

  describe("peer management", () => {
    it("should add peer and emit peer:added", () => {
      const manager = createHeaderOnlyManager();
      const handler = vi.fn();
      manager.on("peer:added", handler);
      const peer = createMockPeer("peer-1", 0n);
      manager.addPeer(peer);
      expect(handler).toHaveBeenCalledWith(peer);
    });

    it("should remove peer and emit peer:removed", () => {
      const manager = createHeaderOnlyManager();
      const handler = vi.fn();
      manager.on("peer:removed", handler);
      manager.addPeer(createMockPeer("peer-1", 5n));
      manager.removePeer("peer-1");
      expect(handler).toHaveBeenCalledWith("peer-1");
    });

    it("should update highest block on addPeer", () => {
      const manager = createHeaderOnlyManager();
      manager.addPeer(createMockPeer("peer-1", 50n));
      expect(manager.getStatus().highestBlock).toBe(50n);
    });

    it("should update peer head via updatePeerHead", () => {
      const manager = createHeaderOnlyManager();
      manager.addPeer(createMockPeer("peer-1", 0n));
      manager.updatePeerHead("peer-1", 200n);
      expect(manager.getStatus().highestBlock).toBe(200n);
    });
  });

  // ============ Header Sync ============

  describe("header sync", () => {
    it("should request headers from peer and save valid ones", async () => {
      const manager = createHeaderOnlyManager();
      const requestFn = createHeaderOnlyRequestFn(3n);
      manager.setRequestFunction(requestFn);
      manager.addPeer(createMockPeer("peer-1", 3n));

      await manager.startSync();

      expect(saveHeader).toHaveBeenCalledTimes(3);
      expect(validateHeader).toHaveBeenCalledTimes(3);
      expect(manager.getStatus().currentBlock).toBe(3n);
      expect(manager.getStatus().headersSynced).toBe(true);
    });

    it("should reject invalid header and emit peer error", async () => {
      const manager = createHeaderOnlyManager();

      let callCount = 0;
      validateHeader.mockImplementation(async (header: BlockHeader) => {
        // Fail header 2 on first two attempts, then accept
        if (header.number === 2n && callCount < 2) return false;
        return true;
      });

      const requestFn = vi
        .fn()
        .mockImplementation(
          async (_peerId: string, _protocol: string, request: any) => {
            await yieldToMacrotasks();
            if (request.type === "headers") {
              callCount++;
              if (callCount <= 2) {
                // First two calls: return both headers (2nd will fail)
                return {
                  type: "headers",
                  data: [createMockHeader(1n, "genesis"), createMockHeader(2n)],
                };
              }
              // After retries: return appropriate headers based on startBlock
              const start = Number(request.startBlock ?? 1n);
              if (start <= 1) {
                return {
                  type: "headers",
                  data: [createMockHeader(1n, "genesis")],
                };
              }
              return { type: "headers", data: [createMockHeader(2n)] };
            }
            return { type: "headers", data: [] };
          },
        );
      manager.setRequestFunction(requestFn);
      manager.addPeer(createMockPeer("peer-1", 2n));

      const errorHandler = vi.fn();
      manager.on("sync:peer_error", errorHandler);

      await manager.startSync();

      expect(errorHandler).toHaveBeenCalled();
      expect(manager.getStatus().currentBlock).toBe(2n);
    });

    it("should cache headers after saving", async () => {
      const manager = createHeaderOnlyManager();
      const requestFn = createHeaderOnlyRequestFn(1n);
      manager.setRequestFunction(requestFn);
      manager.addPeer(createMockPeer("peer-1", 1n));

      await manager.startSync();

      const cached = manager.getCachedHeader(1n);
      expect(cached).toBeDefined();
      expect(cached!.number).toBe(1n);
    });

    it("should emit sync:headers_start and sync:headers_complete", async () => {
      const manager = createHeaderOnlyManager();
      const requestFn = createHeaderOnlyRequestFn(1n);
      manager.setRequestFunction(requestFn);

      const headersStart = vi.fn();
      const headersComplete = vi.fn();
      manager.on("sync:headers_start", headersStart);
      manager.on("sync:headers_complete", headersComplete);

      manager.addPeer(createMockPeer("peer-1", 1n));
      await manager.startSync();

      expect(headersStart).toHaveBeenCalled();
      expect(headersComplete).toHaveBeenCalled();
    });
  });

  // ============ Knowledge Tree Sync ============

  describe("knowledge tree sync", () => {
    it("should fetch knowledge root from latest header", async () => {
      const manager = createFullManager();

      const requestFn = createFullSyncRequestFn(1n, "kr-custom", manager);
      manager.setRequestFunction(requestFn);
      // Peer must be ahead of available headers so knowledge sync has a peer
      manager.addPeer(createMockPeer("peer-1", 10n));

      await manager.startSync();

      expect(manager.getStatus().knowledgeRootSynced).toBe(true);
      expect(manager.getStatus().latestKnowledgeRoot).toBe("kr-custom");
      expect(manager.getCachedKnowledgeNode("kr-custom")).toBeDefined();
    });

    it("should recursively sync child nodes to configured depth", async () => {
      const childLeft = createMockKnowledgeNode("child-left");
      const childRight = createMockKnowledgeNode("child-right");
      const rootNode = createMockKnowledgeNode(
        "kr-root",
        "child-left",
        "child-right",
      );

      // Strategy: peer at 1n, serve header for block 1. Header sync completes
      // naturally (currentBlock=1=highestBlock). Then on sync:headers_complete,
      // bump peer headBlock so selectBestPeer works for knowledge sync.
      // This avoids abort (syncKnowledgeNodes checks this.aborted).
      const mgr = createFullManager({ knowledgeTreeDepth: 1 });

      // After headers complete, make peer appear "ahead" for knowledge sync
      mgr.on("sync:headers_complete", () => {
        mgr.updatePeerHead("peer-1", 100n);
      });

      const requestFn = vi
        .fn()
        .mockImplementation(
          async (_peerId: string, _protocol: string, request: any) => {
            await yieldToMacrotasks();
            if (request.type === "headers") {
              return {
                type: "headers",
                data: [createMockHeader(1n, "genesis", "kr-root")],
              };
            }
            if (request.type === "knowledge_root") {
              return { type: "knowledge_root", data: rootNode };
            }
            if (request.type === "knowledge_nodes") {
              const nodes: KnowledgeTreeNode[] = [];
              for (const hash of request.nodeHashes || []) {
                if (hash === "child-left") nodes.push(childLeft);
                if (hash === "child-right") nodes.push(childRight);
              }
              return { type: "knowledge_nodes", data: nodes };
            }
            return { type: "headers", data: [] };
          },
        );

      mgr.setRequestFunction(requestFn);
      mgr.addPeer(createMockPeer("peer-1", 1n));

      await mgr.startSync();

      expect(mgr.getCachedKnowledgeNode("child-left")).toEqual(childLeft);
      expect(mgr.getCachedKnowledgeNode("child-right")).toEqual(childRight);
    });
  });

  // ============ Knowledge Proof Retrieval ============

  describe("knowledge proof retrieval", () => {
    it("should fetch and verify knowledge proof", async () => {
      const { proof, root } = buildValidProof("fact-1");

      const manager = createFullManager();
      const requestFn = createFullSyncRequestFn(1n, root, manager, (req) => {
        if (req.type === "knowledge_proof") {
          return { type: "knowledge_proof", data: proof };
        }
        return null;
      });
      manager.setRequestFunction(requestFn);
      manager.addPeer(createMockPeer("peer-1", 10n));

      await manager.startSync();
      expect(manager.getStatus().latestKnowledgeRoot).toBe(root);

      const result = await manager.getKnowledgeProof("fact-1");
      expect(result.factId).toBe("fact-1");
    });

    it("should deduplicate concurrent proof requests for same factId", async () => {
      const { proof, root } = buildValidProof("fact-2");

      const manager = createFullManager();
      let proofCallCount = 0;
      const requestFn = createFullSyncRequestFn(1n, root, manager, (req) => {
        if (req.type === "knowledge_proof") {
          proofCallCount++;
          return { type: "knowledge_proof", data: proof };
        }
        return null;
      });
      manager.setRequestFunction(requestFn);
      manager.addPeer(createMockPeer("peer-1", 10n));

      await manager.startSync();

      const [p1, p2] = await Promise.all([
        manager.getKnowledgeProof("fact-2"),
        manager.getKnowledgeProof("fact-2"),
      ]);

      expect(p1.factId).toBe("fact-2");
      expect(p2.factId).toBe("fact-2");
      expect(proofCallCount).toBe(1);
    });

    it("should reject proof with invalid root", async () => {
      const { proof } = buildValidProof("fact-bad");
      const tamperedProof = { ...proof, root: "wrong-root" };

      const manager = createFullManager();
      const requestFn = createFullSyncRequestFn(
        1n,
        "correct-root",
        manager,
        (req) => {
          if (req.type === "knowledge_proof") {
            return { type: "knowledge_proof", data: tamperedProof };
          }
          return null;
        },
      );
      manager.setRequestFunction(requestFn);
      manager.addPeer(createMockPeer("peer-1", 10n));

      await manager.startSync();

      await expect(manager.getKnowledgeProof("fact-bad")).rejects.toThrow(
        "Invalid proof",
      );
    });

    it("should verify proof correctly with verifyKnowledgeProof", () => {
      const { proof } = buildValidProof("fact-v");
      const manager = createHeaderOnlyManager();
      // Without latestKnowledgeRoot set, root won't match → false
      expect(manager.verifyKnowledgeProof(proof)).toBe(false);
    });
  });

  // ============ Live Updates ============

  describe("live updates", () => {
    it("should handle knowledge update and invalidate affected cache", () => {
      const manager = createHeaderOnlyManager();
      const updateHandler = vi.fn();
      manager.on("knowledge:update", updateHandler);

      const update: KnowledgeUpdate = {
        type: "fact_added",
        factId: "fact-1",
        domain: "science",
        blockNumber: 10n,
        affectedRoot: "affected-root",
      };

      manager.handleKnowledgeUpdate(update);

      expect(updateHandler).toHaveBeenCalledWith(update);
      expect(manager.getCachedKnowledgeNode("affected-root")).toBeUndefined();
    });

    it("should call registered knowledge update callback", () => {
      const manager = createHeaderOnlyManager();
      const callback = vi.fn();
      manager.onKnowledgeUpdate(callback);

      const update: KnowledgeUpdate = {
        type: "confidence_changed",
        factId: "fact-1",
        domain: "science",
        newConfidence: 0.85,
        blockNumber: 10n,
        affectedRoot: "some-root",
      };

      manager.handleKnowledgeUpdate(update);
      expect(callback).toHaveBeenCalledWith(update);
    });

    it("should update highestBlock on header announcement", () => {
      const manager = createHeaderOnlyManager();
      const announcedHandler = vi.fn();
      manager.on("header:announced", announcedHandler);

      const header = createMockHeader(100n);
      manager.handleHeaderAnnouncement(header);

      expect(manager.getStatus().highestBlock).toBe(100n);
      expect(announcedHandler).toHaveBeenCalledWith(header);
    });

    it("should subscribe to live updates during full sync", async () => {
      const manager = createFullManager();
      const subscribedHandler = vi.fn();
      manager.on("sync:subscribed", subscribedHandler);

      const requestFn = createFullSyncRequestFn(1n, "kr-1", manager);
      manager.setRequestFunction(requestFn);
      manager.addPeer(createMockPeer("peer-1", 10n));

      await manager.startSync();

      expect(subscribedHandler).toHaveBeenCalledWith(
        LIGHT_SYNC_TOPICS.KNOWLEDGE_UPDATES,
      );
      expect(manager.getStatus().subscribedToUpdates).toBe(true);
    });
  });

  // ============ Cache Management ============

  describe("cache management", () => {
    it("should return cached header and knowledge node after sync", async () => {
      const manager = createFullManager();
      const requestFn = createFullSyncRequestFn(1n, "kr-cached", manager);
      manager.setRequestFunction(requestFn);
      manager.addPeer(createMockPeer("peer-1", 10n));

      await manager.startSync();

      expect(manager.getCachedHeader(1n)).toBeDefined();
      expect(manager.getCachedKnowledgeNode("kr-cached")).toBeDefined();
    });

    it("should clear all caches", async () => {
      const manager = createFullManager();
      const requestFn = createFullSyncRequestFn(1n, "kr-clear", manager);
      manager.setRequestFunction(requestFn);
      manager.addPeer(createMockPeer("peer-1", 10n));

      await manager.startSync();

      manager.clearCaches();
      expect(manager.getCachedHeader(1n)).toBeUndefined();
      expect(manager.getCachedKnowledgeNode("kr-clear")).toBeUndefined();
    });
  });

  // ============ Lifecycle ============

  describe("lifecycle", () => {
    it("should emit sync:started and sync:completed on header-only sync", async () => {
      const manager = createHeaderOnlyManager();
      const startHandler = vi.fn();
      const completeHandler = vi.fn();
      manager.on("sync:started", startHandler);
      manager.on("sync:completed", completeHandler);

      const requestFn = createHeaderOnlyRequestFn(1n);
      manager.setRequestFunction(requestFn);
      manager.addPeer(createMockPeer("peer-1", 1n));

      await manager.startSync();

      expect(startHandler).toHaveBeenCalledTimes(1);
      expect(completeHandler).toHaveBeenCalledTimes(1);
    });

    it("should not start sync if already syncing", async () => {
      const manager = createHeaderOnlyManager();
      const startHandler = vi.fn();
      manager.on("sync:started", startHandler);

      const requestFn = createHeaderOnlyRequestFn(1n);
      manager.setRequestFunction(requestFn);
      manager.addPeer(createMockPeer("peer-1", 1n));

      const p1 = manager.startSync();
      const p2 = manager.startSync();
      await Promise.all([p1, p2]);

      expect(startHandler).toHaveBeenCalledTimes(1);
    });

    it("should stop sync when stopSync is called", () => {
      const manager = createHeaderOnlyManager();
      const stoppedHandler = vi.fn();
      manager.on("sync:stopped", stoppedHandler);

      manager.stopSync();
      expect(stoppedHandler).toHaveBeenCalledTimes(1);
    });

    it("should report isSynced true after header-only sync", async () => {
      const manager = createHeaderOnlyManager();
      const requestFn = createHeaderOnlyRequestFn(1n);
      manager.setRequestFunction(requestFn);
      manager.addPeer(createMockPeer("peer-1", 1n));

      await manager.startSync();

      expect(manager.isSynced()).toBe(true);
    });
  });
});

// ============ LightSyncProtocolHandler ============

describe.sequential("LightSyncProtocolHandler", () => {
  let getHeader: ReturnType<typeof vi.fn>;
  let getKnowledgeNode: ReturnType<typeof vi.fn>;
  let getKnowledgeProofFn: ReturnType<typeof vi.fn>;
  let handler: LightSyncProtocolHandler;

  beforeEach(() => {
    getHeader = vi.fn();
    getKnowledgeNode = vi.fn();
    getKnowledgeProofFn = vi.fn();

    const dummyManager = {} as any;
    handler = new LightSyncProtocolHandler(
      dummyManager,
      getHeader,
      getKnowledgeNode,
      getKnowledgeProofFn,
    );
  });

  describe("handleRequest", () => {
    it("should return headers for headers request", async () => {
      const h1 = createMockHeader(1n, "genesis");
      const h2 = createMockHeader(2n);
      getHeader
        .mockResolvedValueOnce(h1)
        .mockResolvedValueOnce(h2)
        .mockResolvedValue(undefined);

      const response = await handler.handleRequest({
        type: "headers",
        startBlock: 1n,
        count: 5,
      });

      expect(response.type).toBe("headers");
      expect(response.data).toEqual([h1, h2]);
    });

    it("should stop at first missing header", async () => {
      getHeader
        .mockResolvedValueOnce(createMockHeader(0n, "genesis"))
        .mockResolvedValueOnce(undefined);

      const response = await handler.handleRequest({
        type: "headers",
        startBlock: 0n,
        count: 10,
      });

      const headers = response.data as BlockHeader[];
      expect(headers.length).toBe(1);
    });

    it("should return knowledge root node", async () => {
      const rootNode = createMockKnowledgeNode("root-hash", "left", "right");
      getKnowledgeNode.mockResolvedValue(rootNode);

      const response = await handler.handleRequest({
        type: "knowledge_root",
        nodeHashes: ["root-hash"],
      });

      expect(response.type).toBe("knowledge_root");
      expect(response.data).toEqual(rootNode);
    });

    it("should return knowledge nodes", async () => {
      const node1 = createMockKnowledgeNode("hash-1");
      const node2 = createMockKnowledgeNode("hash-2");
      getKnowledgeNode
        .mockResolvedValueOnce(node1)
        .mockResolvedValueOnce(node2);

      const response = await handler.handleRequest({
        type: "knowledge_nodes",
        nodeHashes: ["hash-1", "hash-2"],
      });

      expect(response.type).toBe("knowledge_nodes");
      expect(response.data).toEqual([node1, node2]);
    });

    it("should return knowledge proof", async () => {
      const { proof } = buildValidProof("fact-1");
      getKnowledgeProofFn.mockResolvedValue(proof);

      const response = await handler.handleRequest({
        type: "knowledge_proof",
        factId: "fact-1",
      });

      expect(response.type).toBe("knowledge_proof");
      expect(response.data).toEqual(proof);
    });

    it("should throw for missing factId on knowledge_proof request", async () => {
      await expect(
        handler.handleRequest({ type: "knowledge_proof" }),
      ).rejects.toThrow("factId required");
    });

    it("should throw for unknown request type", async () => {
      await expect(
        handler.handleRequest({ type: "unknown" as any }),
      ).rejects.toThrow("Unknown request type");
    });
  });
});
