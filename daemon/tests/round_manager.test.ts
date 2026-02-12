/**
 * Round Manager Tests
 *
 * Tests idempotency guard, phase detection, and old round cleanup.
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { RoundManager } from "../src/round_manager.js";
import type { RPCClient } from "../src/rpc_client.js";

function mockRPC(): RPCClient {
  return {
    getActiveRounds: vi.fn().mockResolvedValue([]),
    isSelectedForRound: vi.fn().mockResolvedValue(true),
    getClaimData: vi.fn().mockResolvedValue({
      id: "claim-1",
      claimText: "Test claim",
      domain: "test",
      epistemicCategory: "empirical",
      logicZone: "empirical",
      references: [],
      evidence: [],
      submitter: "did:lgm:test",
      stakeAmount: "1000",
    }),
  } as unknown as RPCClient;
}

// Sequential to avoid mock interference from concurrent execution
describe.sequential("RoundManager", () => {
  let rpc: RPCClient;
  let manager: RoundManager;

  beforeEach(() => {
    rpc = mockRPC();
    manager = new RoundManager(rpc, "lgm1validator");
  });

  describe("idempotency", () => {
    it("markProcessed prevents re-processing", () => {
      expect(manager.isProcessed("round-1")).toBe(false);

      manager.markProcessed("round-1", 100n);

      expect(manager.isProcessed("round-1")).toBe(true);
    });

    it("different rounds are independent", () => {
      manager.markProcessed("round-1", 100n);

      expect(manager.isProcessed("round-1")).toBe(true);
      expect(manager.isProcessed("round-2")).toBe(false);
    });
  });

  describe("getCommitRounds", () => {
    it("filters to commit-phase rounds only", async () => {
      (rpc.getActiveRounds as ReturnType<typeof vi.fn>).mockResolvedValue([
        {
          id: "r1",
          claimId: "c1",
          phase: "commit",
          startHeight: 90n,
          commitDeadlineHeight: 100n,
          revealDeadlineHeight: 110n,
        },
        {
          id: "r2",
          claimId: "c2",
          phase: "reveal",
          startHeight: 80n,
          commitDeadlineHeight: 90n,
          revealDeadlineHeight: 100n,
        },
        {
          id: "r3",
          claimId: "c3",
          phase: "commit",
          startHeight: 95n,
          commitDeadlineHeight: 105n,
          revealDeadlineHeight: 115n,
        },
      ]);

      const rounds = await manager.getCommitRounds(95n);

      expect(rounds).toHaveLength(2);
      expect(rounds[0].id).toBe("r1");
      expect(rounds[1].id).toBe("r3");
    });

    it("filters to VRF-selected rounds", async () => {
      (rpc.getActiveRounds as ReturnType<typeof vi.fn>).mockResolvedValue([
        {
          id: "r1",
          claimId: "c1",
          phase: "commit",
          startHeight: 90n,
          commitDeadlineHeight: 100n,
          revealDeadlineHeight: 110n,
        },
        {
          id: "r2",
          claimId: "c2",
          phase: "commit",
          startHeight: 90n,
          commitDeadlineHeight: 100n,
          revealDeadlineHeight: 110n,
        },
      ]);

      // Only selected for r1, not r2 (match by roundId argument)
      (rpc.isSelectedForRound as ReturnType<typeof vi.fn>).mockImplementation(
        (roundId: string) => Promise.resolve(roundId === "r1"),
      );

      const rounds = await manager.getCommitRounds(95n);

      expect(rounds).toHaveLength(1);
      expect(rounds[0].id).toBe("r1");
    });

    it("returns empty when no commit rounds exist", async () => {
      (rpc.getActiveRounds as ReturnType<typeof vi.fn>).mockResolvedValue([
        {
          id: "r1",
          claimId: "c1",
          phase: "reveal",
          startHeight: 80n,
          commitDeadlineHeight: 90n,
          revealDeadlineHeight: 100n,
        },
      ]);

      const rounds = await manager.getCommitRounds(95n);
      expect(rounds).toHaveLength(0);
    });
  });

  describe("getRevealRounds", () => {
    it("filters to reveal-phase rounds", async () => {
      (rpc.getActiveRounds as ReturnType<typeof vi.fn>).mockResolvedValue([
        {
          id: "r1",
          claimId: "c1",
          phase: "commit",
          startHeight: 90n,
          commitDeadlineHeight: 100n,
          revealDeadlineHeight: 110n,
        },
        {
          id: "r2",
          claimId: "c2",
          phase: "reveal",
          startHeight: 80n,
          commitDeadlineHeight: 90n,
          revealDeadlineHeight: 100n,
        },
      ]);

      const rounds = await manager.getRevealRounds(95n);

      expect(rounds).toHaveLength(1);
      expect(rounds[0].id).toBe("r2");
    });
  });

  describe("cleanup", () => {
    it("prunes old processed rounds", () => {
      manager.markProcessed("old-1", 10n);
      manager.markProcessed("old-2", 20n);
      manager.markProcessed("recent", 90n);
      manager.markProcessed("current", 100n);

      expect(manager.processedCount()).toBe(4);

      // Cleanup: currentHeight=110, maxAge=50 → cutoff=60
      manager.cleanupOld(110n, 50n);

      expect(manager.processedCount()).toBe(2);
      expect(manager.isProcessed("old-1")).toBe(false);
      expect(manager.isProcessed("old-2")).toBe(false);
      expect(manager.isProcessed("recent")).toBe(true);
      expect(manager.isProcessed("current")).toBe(true);
    });

    it("does not prune when currentHeight < maxAge", () => {
      manager.markProcessed("early", 5n);

      manager.cleanupOld(10n, 50n); // 10 < 50

      expect(manager.isProcessed("early")).toBe(true);
    });

    it("uses default maxAge of 200 blocks", () => {
      manager.markProcessed("old", 10n);
      manager.markProcessed("recent", 250n);

      manager.cleanupOld(300n); // default maxAge = 200, cutoff = 100

      expect(manager.isProcessed("old")).toBe(false);
      expect(manager.isProcessed("recent")).toBe(true);
    });
  });

  describe("fetchClaimData", () => {
    it("delegates to RPC client", async () => {
      const claim = await manager.fetchClaimData("claim-1");

      expect(claim.id).toBe("claim-1");
      expect(claim.claimText).toBe("Test claim");
      expect(rpc.getClaimData).toHaveBeenCalledWith("claim-1");
    });
  });
});
