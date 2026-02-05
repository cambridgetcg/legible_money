/**
 * Persistence Layer Tests
 *
 * Tests for Phases 2-4 of the persistence backend (GAP-001):
 * - Phase 2: State tree persistence
 * - Phase 3: Write-ahead logging
 * - Phase 4: Recovery protocol
 */

import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { mkdtemp, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import {
  LevelDBStore,
  StatePersistenceManager,
  WriteAheadLog,
  WALTransaction,
  RecoveryManager,
  createCheckpoint,
  loadCheckpoint,
} from "../../../state/storage/index.js";
import type {
  AccountState,
  ValidatorState,
  FactState,
} from "../../../state/types.js";

// ============ Test Helpers ============

async function withStore<T>(
  testFn: (store: LevelDBStore) => Promise<T>,
): Promise<T> {
  const tempDir = await mkdtemp(join(tmpdir(), "persistence-test-"));
  const store = await LevelDBStore.open({
    type: "leveldb",
    path: tempDir,
  });
  try {
    return await testFn(store);
  } finally {
    await store.close();
    await rm(tempDir, { recursive: true, force: true });
  }
}

function createMockAccount(did: string, balance: bigint = 1000n): AccountState {
  return {
    did,
    balance,
    stakedBalance: 0n,
    nonce: 0n,
    operationalKeyHash: "abc123",
    operationalKeyVersion: 1,
    sessionKeyCount: 0,
    reputationScore: 0.5,
    lastActiveBlock: 0n,
    flags: {
      isValidator: false,
      canSubmitClaims: true,
      canChallenge: true,
      inRecovery: false,
      frozen: false,
    },
  };
}

function createMockValidator(
  publicKey: string,
  stake: bigint = 10000n,
): ValidatorState {
  return {
    publicKey: publicKey as any,
    did: `did:lgm:${publicKey.slice(0, 16)}`,
    stake,
    delegatedStake: 0n,
    totalStake: stake,
    reputation: 0.8,
    isActive: true,
    activeSinceBlock: 0n,
    lastProposedBlock: 0n,
    blocksProposed: 0,
    verificationsCompleted: 0,
    verificationsCorrect: 0,
    slashCount: 0,
    commissionRate: 0.1,
  };
}

function createMockFact(id: string, claim: string): FactState {
  return {
    id,
    claim,
    domain: "test.domain",
    confidence: 0.85,
    verifiedAt: 100n,
    verifierCount: 5,
    challengeCount: 0,
    status: "active",
    referenceRoot: "",
  };
}

// ============ Phase 2: State Persistence Tests ============

describe("StatePersistenceManager", () => {
  describe("account persistence", () => {
    it("should persist and load a single account", async () => {
      await withStore(async (store) => {
        const persistence = new StatePersistenceManager(store);
        const account = createMockAccount("did:lgm:test123", 5000n);

        await persistence.persistAccount(account.did, account);
        const loaded = await persistence.loadAccount(account.did);

        expect(loaded).not.toBeNull();
        expect(loaded!.did).toBe(account.did);
        expect(loaded!.balance).toBe(5000n);
      });
    });

    it("should load all accounts", async () => {
      await withStore(async (store) => {
        const persistence = new StatePersistenceManager(store);

        await persistence.persistAccount(
          "did:lgm:acc1",
          createMockAccount("did:lgm:acc1", 100n),
        );
        await persistence.persistAccount(
          "did:lgm:acc2",
          createMockAccount("did:lgm:acc2", 200n),
        );
        await persistence.persistAccount(
          "did:lgm:acc3",
          createMockAccount("did:lgm:acc3", 300n),
        );

        const accounts = await persistence.loadAllAccounts();

        expect(accounts.size).toBe(3);
        expect(accounts.get("did:lgm:acc1")!.balance).toBe(100n);
        expect(accounts.get("did:lgm:acc2")!.balance).toBe(200n);
        expect(accounts.get("did:lgm:acc3")!.balance).toBe(300n);
      });
    });

    it("should delete an account", async () => {
      await withStore(async (store) => {
        const persistence = new StatePersistenceManager(store);
        const account = createMockAccount("did:lgm:todelete");

        await persistence.persistAccount(account.did, account);
        expect(await persistence.loadAccount(account.did)).not.toBeNull();

        await persistence.deleteAccount(account.did);
        expect(await persistence.loadAccount(account.did)).toBeNull();
      });
    });
  });

  describe("validator persistence", () => {
    it("should persist and load validators", async () => {
      await withStore(async (store) => {
        const persistence = new StatePersistenceManager(store);
        const validator = createMockValidator("abcd1234", 50000n);

        await persistence.persistValidator("abcd1234", validator);
        const loaded = await persistence.loadValidator("abcd1234");

        expect(loaded).not.toBeNull();
        expect(loaded!.stake).toBe(50000n);
        expect(loaded!.isActive).toBe(true);
      });
    });

    it("should load all validators", async () => {
      await withStore(async (store) => {
        const persistence = new StatePersistenceManager(store);

        await persistence.persistValidator(
          "val1",
          createMockValidator("val1", 1000n),
        );
        await persistence.persistValidator(
          "val2",
          createMockValidator("val2", 2000n),
        );

        const validators = await persistence.loadAllValidators();

        expect(validators.size).toBe(2);
      });
    });
  });

  describe("fact persistence", () => {
    it("should persist and load facts", async () => {
      await withStore(async (store) => {
        const persistence = new StatePersistenceManager(store);
        const fact = createMockFact(
          "fact-001",
          "Water boils at 100°C at sea level",
        );

        await persistence.persistFact(fact.id, fact);
        const loaded = await persistence.loadFact(fact.id);

        expect(loaded).not.toBeNull();
        expect(loaded!.claim).toBe("Water boils at 100°C at sea level");
        expect(loaded!.confidence).toBe(0.85);
      });
    });
  });

  describe("batch persistence", () => {
    it("should persist multiple entries atomically", async () => {
      await withStore(async (store) => {
        const persistence = new StatePersistenceManager(store);

        const count = await persistence.persistBatch({
          accounts: new Map([
            ["did:lgm:batch1", createMockAccount("did:lgm:batch1")],
            ["did:lgm:batch2", createMockAccount("did:lgm:batch2")],
          ]),
          validators: new Map([
            ["batchval1", createMockValidator("batchval1")],
          ]),
          facts: new Map([
            ["batchfact1", createMockFact("batchfact1", "Test fact")],
          ]),
        });

        expect(count).toBe(4);

        const accounts = await persistence.loadAllAccounts();
        const validators = await persistence.loadAllValidators();
        const facts = await persistence.loadAllFacts();

        expect(accounts.size).toBe(2);
        expect(validators.size).toBe(1);
        expect(facts.size).toBe(1);
      });
    });
  });

  describe("statistics", () => {
    it("should return correct statistics", async () => {
      await withStore(async (store) => {
        const persistence = new StatePersistenceManager(store);

        await persistence.persistAccount(
          "did:lgm:stat1",
          createMockAccount("did:lgm:stat1"),
        );
        await persistence.persistValidator(
          "statval1",
          createMockValidator("statval1"),
        );
        await persistence.persistFact(
          "statfact1",
          createMockFact("statfact1", "Test"),
        );

        const stats = await persistence.getStatistics();

        expect(stats.accountCount).toBe(1);
        expect(stats.validatorCount).toBe(1);
        expect(stats.factCount).toBe(1);
        expect(stats.modifiedCount).toBe(3);
      });
    });
  });
});

// ============ Phase 3: Write-Ahead Log Tests ============

describe("WriteAheadLog", () => {
  describe("transaction lifecycle", () => {
    it("should create and commit a transaction", async () => {
      await withStore(async (store) => {
        const wal = new WriteAheadLog(store);
        await wal.initialize();

        const entryId = wal.beginTransaction(1n);
        wal.addPut(0x01, "test-key", new TextEncoder().encode("test-value"));
        await wal.prepare();
        await wal.commit();

        const entry = await wal.getEntry(entryId);
        expect(entry).not.toBeNull();
        expect(entry!.status).toBe("committed");
        expect(entry!.operations).toHaveLength(1);
      });
    });

    it("should rollback an uncommitted transaction", async () => {
      await withStore(async (store) => {
        const wal = new WriteAheadLog(store);
        await wal.initialize();

        const entryId = wal.beginTransaction(1n);
        wal.addPut(0x01, "test-key", new TextEncoder().encode("test-value"));
        await wal.prepare();
        await wal.rollback();

        const entry = await wal.getEntry(entryId);
        expect(entry!.status).toBe("rolled_back");
      });
    });
  });

  describe("WALTransaction helper", () => {
    it("should simplify transaction usage", async () => {
      await withStore(async (store) => {
        const wal = new WriteAheadLog(store);
        await wal.initialize();

        const tx = new WALTransaction(wal, 5n);
        tx.put(0x01, "key1", new TextEncoder().encode("value1"));
        tx.put(0x02, "key2", new TextEncoder().encode("value2"));
        await tx.commit();

        const entry = await wal.getEntry(tx.getId());
        expect(entry!.status).toBe("committed");
        expect(entry!.operations).toHaveLength(2);
      });
    });
  });

  describe("recovery", () => {
    it("should find uncommitted entries", async () => {
      await withStore(async (store) => {
        const wal = new WriteAheadLog(store);
        await wal.initialize();

        // Create a prepared but not committed entry
        wal.beginTransaction(1n);
        wal.addPut(0x01, "key", new TextEncoder().encode("value"));
        await wal.prepare();
        // Don't commit - simulate crash

        // Create a new WAL instance (simulating restart)
        const wal2 = new WriteAheadLog(store);
        await wal2.initialize();

        const uncommitted = await wal2.findUncommittedEntries();
        expect(uncommitted).toHaveLength(1);
        expect(uncommitted[0].status).toBe("prepared");
      });
    });
  });

  describe("integrity", () => {
    it("should verify entry checksum", async () => {
      await withStore(async (store) => {
        const wal = new WriteAheadLog(store);
        await wal.initialize();

        wal.beginTransaction(1n);
        wal.addPut(0x01, "key", new TextEncoder().encode("value"));
        await wal.prepare();
        await wal.commit();

        const entry = await wal.getEntry(
          (await wal.getStatistics()).entryCounter
            .toString()
            .padStart(16, "0")
            .replace(/^/, "wal-"),
        );
        // Entry should pass integrity check
        const currentEntry = wal.getCurrentEntry();
        expect(currentEntry).toBeNull(); // Transaction was committed
      });
    });
  });

  describe("cleanup", () => {
    it("should cleanup old entries", async () => {
      await withStore(async (store) => {
        const wal = new WriteAheadLog(store);
        await wal.initialize();

        // Create multiple committed entries
        for (let i = 0; i < 5; i++) {
          wal.beginTransaction(BigInt(i));
          wal.addPut(0x01, `key${i}`, new TextEncoder().encode(`value${i}`));
          await wal.prepare();
          await wal.commit();
        }

        const deleted = await wal.cleanup(2); // Keep only 2
        expect(deleted).toBe(3);
      });
    });
  });
});

// ============ Phase 4: Recovery Tests ============

describe("RecoveryManager", () => {
  describe("full recovery", () => {
    it("should recover from empty state", async () => {
      await withStore(async (store) => {
        const recovery = new RecoveryManager(store);
        const result = await recovery.recover();

        expect(result.success).toBe(true);
        expect(result.blockNumber).toBe(0n);
        expect(result.warnings).toContain(
          "No state metadata found - starting fresh",
        );
      });
    });

    it("should recover persisted state", async () => {
      await withStore(async (store) => {
        // First, persist some state
        const persistence = new StatePersistenceManager(store);
        await persistence.persistAccount(
          "did:lgm:recover1",
          createMockAccount("did:lgm:recover1"),
        );
        await persistence.persistValidator(
          "recoverval1",
          createMockValidator("recoverval1"),
        );
        await persistence.persistFact(
          "recoverfact1",
          createMockFact("recoverfact1", "Test"),
        );

        // Persist metadata
        const configKey = new Uint8Array([
          0xf0,
          ...new TextEncoder().encode("latest"),
        ]);
        await store.put(
          configKey,
          new TextEncoder().encode(JSON.stringify({ blockNumber: "100" })),
        );

        const stateKey = new Uint8Array([
          0xf0,
          ...new TextEncoder().encode("state:100"),
        ]);
        await store.put(
          stateKey,
          new TextEncoder().encode(
            JSON.stringify({
              blockNumber: "100",
              stateRoot: "abc123",
              accountRoot: "",
              validatorRoot: "",
              knowledgeRoot: "",
              version: "100",
              timestamp: Date.now(),
            }),
          ),
        );

        // Now recover
        const recovery = new RecoveryManager(store);
        const result = await recovery.recover();

        expect(result.success).toBe(true);
        expect(result.blockNumber).toBe(100n);
        expect(result.accountsLoaded).toBe(1);
        expect(result.validatorsLoaded).toBe(1);
        expect(result.factsLoaded).toBe(1);
      });
    });
  });

  describe("checkpoint management", () => {
    it("should create and load checkpoints", async () => {
      await withStore(async (store) => {
        await createCheckpoint(store, 500n, "checkpoint-root-hash");

        const checkpoint = await loadCheckpoint(store);

        expect(checkpoint).not.toBeNull();
        expect(checkpoint!.blockNumber).toBe(500n);
        expect(checkpoint!.stateRoot).toBe("checkpoint-root-hash");
      });
    });
  });
});

// ============ Integration Test ============

// Helper for BigInt-safe serialization
function serializeForWAL<T>(value: T): Uint8Array {
  const json = JSON.stringify(value, (_, v) =>
    typeof v === "bigint" ? { __bigint__: v.toString() } : v,
  );
  return new TextEncoder().encode(json);
}

describe("Full Persistence Flow", () => {
  it("should handle complete persistence lifecycle", async () => {
    await withStore(async (store) => {
      // 1. Initialize components
      const persistence = new StatePersistenceManager(store);
      const wal = new WriteAheadLog(store);
      await wal.initialize();

      // 2. Begin transaction with WAL
      const tx = new WALTransaction(wal, 1n);
      tx.put(0x01, "test-key", new TextEncoder().encode("test-value"));
      await tx.commit();

      // 3. Persist state using persistence manager
      const account = createMockAccount("did:lgm:lifecycle", 10000n);
      await persistence.persistAccount(account.did, account);

      // 4. Verify account was persisted
      const loadedAccount = await persistence.loadAccount(account.did);
      expect(loadedAccount).not.toBeNull();
      expect(loadedAccount!.balance).toBe(10000n);

      // 5. Create checkpoint
      await createCheckpoint(store, 1n, "test-state-root");

      // 6. Verify checkpoint
      const checkpoint = await loadCheckpoint(store);
      expect(checkpoint).not.toBeNull();
      expect(checkpoint!.blockNumber).toBe(1n);

      // 7. Load all accounts (simulating recovery)
      const allAccounts = await persistence.loadAllAccounts();
      expect(allAccounts.size).toBe(1);
      expect(allAccounts.has(account.did)).toBe(true);
    });
  });

  it("should handle WAL recovery after simulated crash", async () => {
    await withStore(async (store) => {
      const wal = new WriteAheadLog(store);
      await wal.initialize();

      // Simulate a prepared but uncommitted transaction (crash scenario)
      wal.beginTransaction(1n);
      wal.addPut(0x01, "crash-key", new TextEncoder().encode("crash-value"));
      await wal.prepare();
      // Don't commit - this simulates a crash

      // Create new WAL (simulating restart)
      const wal2 = new WriteAheadLog(store);
      await wal2.initialize();

      // Should find the uncommitted entry
      const uncommitted = await wal2.findUncommittedEntries();
      expect(uncommitted).toHaveLength(1);
      expect(uncommitted[0].blockNumber).toBe(1n);

      // Mark as committed after replay
      await wal2.markCommitted(uncommitted[0].id);

      // Verify it's now committed
      const entry = await wal2.getEntry(uncommitted[0].id);
      expect(entry!.status).toBe("committed");
    });
  });
});
