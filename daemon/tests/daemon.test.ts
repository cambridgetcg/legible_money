/**
 * Daemon Integration Tests
 *
 * Tests the full commit-reveal cycle with mocked RPC and Claude,
 * graceful shutdown, and commitment persistence.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { mkdtempSync, rmSync, writeFileSync, mkdirSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";
import { EventEmitter } from "node:events";
import type { DaemonConfig } from "../src/config.js";
import type { RoundPhase } from "../src/types.js";

// ---- Mocks must be set up before imports ----

const mockCreate = vi.fn().mockResolvedValue({
  content: [
    {
      type: "text",
      text: '{"verdict":"accept","confidence":750000,"justification":"test evaluation"}',
    },
  ],
});

vi.mock("@anthropic-ai/sdk", () => ({
  default: vi.fn().mockImplementation(() => ({
    messages: { create: mockCreate },
  })),
}));

vi.mock("../src/session_manager.js", () => ({
  SessionManager: vi.fn().mockImplementation(() => ({
    getSessionKey: vi.fn().mockResolvedValue({
      publicKey: "mock-pub",
      privateKey: "mock-priv",
      id: "sess-1",
      createdAt: Date.now(),
      expiresAt: Date.now() + 3600000,
      limits: {},
      usage: {
        transactionCount: 0,
        totalSpent: 0n,
        dailySpent: 0n,
        lastUsed: 0,
        lastDailyReset: Date.now(),
      },
    }),
    signTransaction: vi.fn().mockResolvedValue("mock-sig"),
    updateHeight: vi.fn(),
  })),
}));

// Track mock RPC instances for test control
// eslint-disable-next-line @typescript-eslint/no-explicit-any
let activeRpcMock: any;
const roundPhases = new Map<string, RoundPhase>();

vi.mock("../src/rpc_client.js", () => ({
  RPCClient: vi.fn().mockImplementation(() => {
    const emitter = new EventEmitter();
    activeRpcMock = Object.assign(emitter, {
      connect: vi.fn().mockResolvedValue(undefined),
      disconnect: vi.fn(),
      getBlockHeight: vi.fn().mockResolvedValue(100n),
      getActiveRounds: vi.fn().mockResolvedValue([]),
      getRoundPhase: vi
        .fn()
        .mockImplementation((roundId: string) =>
          Promise.resolve(roundPhases.get(roundId) ?? "commit"),
        ),
      getClaimData: vi.fn().mockResolvedValue({
        id: "claim-1",
        claimText: "Test claim",
        domain: "test",
        epistemicCategory: "empirical",
        logicZone: "empirical",
        references: [],
        evidence: [],
        submitter: "did:lgm:sub",
        stakeAmount: "1000",
      }),
      isSelectedForRound: vi.fn().mockResolvedValue(true),
      getValidatorStatus: vi.fn().mockResolvedValue("active"),
      submitCommitment: vi.fn().mockResolvedValue(undefined),
      submitReveal: vi.fn().mockResolvedValue(undefined),
      call: vi.fn().mockResolvedValue({}),
    });
    return activeRpcMock;
  }),
}));

import { EvaluationDaemon } from "../src/daemon.js";
import { CommitmentStore } from "../src/commitment_store.js";

function makeConfig(dir: string): DaemonConfig {
  return {
    node: {
      rpcEndpoint: "http://localhost:8545",
      wsEndpoint: "ws://localhost:8546",
    },
    validator: {
      operationalKeyPath: join(dir, "key.json"),
      validatorAddress: "lgm1testval",
      agentDid: "did:lgm:testval",
    },
    claude: {
      model: "claude-sonnet-4-20250514",
      maxTokens: 512,
      timeoutMs: 5000,
      temperature: 0,
    },
    storage: { dataDir: dir },
    logLevel: "error",
  };
}

// Sequential: async event-based tests must not race
describe.sequential("EvaluationDaemon", () => {
  let tmpDir: string;

  beforeEach(() => {
    tmpDir = mkdtempSync(join(tmpdir(), "lgm-daemon-test-"));
    mkdirSync(tmpDir, { recursive: true });
    writeFileSync(
      join(tmpDir, "key.json"),
      JSON.stringify({ publicKey: "pk", privateKey: "sk" }),
    );
    mockCreate.mockClear();
    roundPhases.clear();
  });

  afterEach(async () => {
    rmSync(tmpDir, { recursive: true, force: true });
  });

  it("starts and stops cleanly", async () => {
    const daemon = await EvaluationDaemon.create(makeConfig(tmpDir));

    await daemon.start();
    expect(daemon.getStats().status).toBe("running");

    await daemon.stop();
    expect(daemon.getStats().status).toBe("stopped");
  });

  it("reports initial stats", async () => {
    const daemon = await EvaluationDaemon.create(makeConfig(tmpDir));
    const stats = daemon.getStats();

    expect(stats.status).toBe("stopped");
    expect(stats.roundsProcessed).toBe(0);
    expect(stats.revealsSubmitted).toBe(0);
    expect(stats.errors).toBe(0);
  });

  it("processes a commit-phase round", async () => {
    const daemon = await EvaluationDaemon.create(makeConfig(tmpDir));
    await daemon.start();

    // Set up: one round in commit phase
    activeRpcMock.getActiveRounds.mockResolvedValue([
      {
        id: "r1",
        claimId: "claim-1",
        phase: "commit",
        startHeight: 95n,
        commitDeadlineHeight: 105n,
        revealDeadlineHeight: 115n,
      },
    ]);

    const committed = new Promise<void>((resolve) =>
      daemon.once("committed", resolve),
    );
    activeRpcMock.emit("block", {
      height: 100n,
      hash: "bh",
      timestamp: Date.now(),
    });
    await committed;

    expect(activeRpcMock.submitCommitment).toHaveBeenCalledWith(
      expect.objectContaining({
        roundId: "r1",
        validatorAddress: "lgm1testval",
      }),
    );
    expect(daemon.getStats().roundsProcessed).toBe(1);

    await daemon.stop();
  });

  it("processes reveal after commit", async () => {
    const daemon = await EvaluationDaemon.create(makeConfig(tmpDir));
    await daemon.start();

    // Commit phase
    activeRpcMock.getActiveRounds.mockResolvedValue([
      {
        id: "r1",
        claimId: "claim-1",
        phase: "commit",
        startHeight: 95n,
        commitDeadlineHeight: 105n,
        revealDeadlineHeight: 115n,
      },
    ]);

    const committed = new Promise<void>((resolve) =>
      daemon.once("committed", resolve),
    );
    activeRpcMock.emit("block", {
      height: 100n,
      hash: "h1",
      timestamp: Date.now(),
    });
    await committed;

    // Reveal phase — round transitions
    activeRpcMock.getActiveRounds.mockResolvedValue([]);
    roundPhases.set("r1", "reveal");

    const revealed = new Promise<void>((resolve) =>
      daemon.once("revealed", resolve),
    );

    // Need to wait for processing to finish before emitting next block
    await new Promise((resolve) => setTimeout(resolve, 50));
    activeRpcMock.emit("block", {
      height: 106n,
      hash: "h2",
      timestamp: Date.now(),
    });
    await revealed;

    expect(activeRpcMock.submitReveal).toHaveBeenCalledWith(
      expect.objectContaining({
        roundId: "r1",
        verdict: "accept",
        confidence: 750_000,
      }),
    );
    expect(daemon.getStats().revealsSubmitted).toBe(1);

    await daemon.stop();
  });

  it("does not re-process committed rounds (idempotency)", async () => {
    const daemon = await EvaluationDaemon.create(makeConfig(tmpDir));
    await daemon.start();

    activeRpcMock.getActiveRounds.mockResolvedValue([
      {
        id: "r1",
        claimId: "claim-1",
        phase: "commit",
        startHeight: 95n,
        commitDeadlineHeight: 105n,
        revealDeadlineHeight: 115n,
      },
    ]);

    // First block — commits
    const committed = new Promise<void>((resolve) =>
      daemon.once("committed", resolve),
    );
    activeRpcMock.emit("block", {
      height: 100n,
      hash: "h1",
      timestamp: Date.now(),
    });
    await committed;

    // Wait for processing flag to clear
    await new Promise((resolve) => setTimeout(resolve, 50));

    // Second block — same round should NOT re-commit
    activeRpcMock.emit("block", {
      height: 101n,
      hash: "h2",
      timestamp: Date.now(),
    });
    await new Promise((resolve) => setTimeout(resolve, 100));

    expect(activeRpcMock.submitCommitment).toHaveBeenCalledTimes(1);

    await daemon.stop();
  });

  it("preserves commitments on graceful shutdown", async () => {
    const daemon = await EvaluationDaemon.create(makeConfig(tmpDir));
    await daemon.start();

    activeRpcMock.getActiveRounds.mockResolvedValue([
      {
        id: "r-persist",
        claimId: "claim-1",
        phase: "commit",
        startHeight: 95n,
        commitDeadlineHeight: 105n,
        revealDeadlineHeight: 115n,
      },
    ]);

    const committed = new Promise<void>((resolve) =>
      daemon.once("committed", resolve),
    );
    activeRpcMock.emit("block", {
      height: 100n,
      hash: "h1",
      timestamp: Date.now(),
    });
    await committed;

    await daemon.stop();

    // Verify commitment persisted to disk
    const store = new CommitmentStore(tmpDir);
    const c = store.get("r-persist");
    expect(c).toBeDefined();
    expect(c!.verdict).toBe("accept");
  });
});
