/**
 * Formation Pool (Matchmaking) Tests
 *
 * Tests for pool registration, matching algorithm, withdrawal cooldown,
 * entry expiry, and scoring mechanics.
 */

import { describe, it, expect } from "vitest";
import {
  createFormationPool,
  registerInPool,
  withdrawFromPool,
  findBestMatch,
  processMatch,
  expireStaleEntries,
} from "../../../partnership/pool_matching.js";
import { FORMATION_POOL } from "../../../partnership/constants.js";
import type { DID } from "../../../core/agents/identity/did.js";

const humanDid = "did:lgm:human-pool-001" as DID;
const agentDid = "did:lgm:agent-pool-001" as DID;
const agent2Did = "did:lgm:agent-pool-002" as DID;
const currentBlock = 5000n;

const defaultStakeRange = { min: 100n * 10n ** 15n, max: 500n * 10n ** 15n };

describe("Formation Pool Registration", () => {
  it("should register an agent in an empty pool", () => {
    const pool = createFormationPool();
    const {
      pool: updated,
      entry,
      error,
    } = registerInPool(
      pool,
      humanDid,
      ["mathematics", "physics"],
      "human",
      defaultStakeRange,
      FORMATION_POOL.REGISTRATION_DEPOSIT,
      currentBlock,
    );

    expect(error).toBeUndefined();
    expect(entry).toBeDefined();
    expect(entry.did).toBe(humanDid);
    expect(entry.status).toBe("active");
    expect(entry.domains).toEqual(["mathematics", "physics"]);
    expect(entry.deposit).toBe(FORMATION_POOL.REGISTRATION_DEPOSIT);
    expect(entry.expiresAt).toBe(
      currentBlock + FORMATION_POOL.ENTRY_EXPIRY_BLOCKS,
    );
    expect(updated.totalRegistrations).toBe(1);
  });

  it("should reject registration with insufficient deposit", () => {
    const pool = createFormationPool();
    const { error } = registerInPool(
      pool,
      humanDid,
      ["mathematics"],
      "human",
      defaultStakeRange,
      FORMATION_POOL.REGISTRATION_DEPOSIT - 1n,
      currentBlock,
    );

    expect(error).toContain("below minimum");
  });

  it("should reject duplicate active registration", () => {
    const pool = createFormationPool();
    const { pool: updated } = registerInPool(
      pool,
      humanDid,
      ["mathematics"],
      "human",
      defaultStakeRange,
      FORMATION_POOL.REGISTRATION_DEPOSIT,
      currentBlock,
    );

    const { error } = registerInPool(
      updated,
      humanDid,
      ["physics"],
      "human",
      defaultStakeRange,
      FORMATION_POOL.REGISTRATION_DEPOSIT,
      currentBlock,
    );

    expect(error).toContain("Already registered");
  });

  it("should reject registration with no domains", () => {
    const pool = createFormationPool();
    const { error } = registerInPool(
      pool,
      humanDid,
      [],
      "human",
      defaultStakeRange,
      FORMATION_POOL.REGISTRATION_DEPOSIT,
      currentBlock,
    );

    expect(error).toContain("at least one domain");
  });

  it("should reject invalid stake range", () => {
    const pool = createFormationPool();
    const { error } = registerInPool(
      pool,
      humanDid,
      ["mathematics"],
      "human",
      { min: 500n, max: 100n },
      FORMATION_POOL.REGISTRATION_DEPOSIT,
      currentBlock,
    );

    expect(error).toContain("min cannot exceed max");
  });

  it("should reject when pool is full", () => {
    let pool = createFormationPool();

    // Fill pool to max
    for (let i = 0; i < FORMATION_POOL.MAX_POOL_SIZE; i++) {
      const { pool: updated } = registerInPool(
        pool,
        `did:lgm:fill-${i}` as DID,
        ["math"],
        "either",
        defaultStakeRange,
        FORMATION_POOL.REGISTRATION_DEPOSIT,
        currentBlock,
      );
      pool = updated;
    }

    const { error } = registerInPool(
      pool,
      humanDid,
      ["math"],
      "human",
      defaultStakeRange,
      FORMATION_POOL.REGISTRATION_DEPOSIT,
      currentBlock,
    );

    expect(error).toContain("full");
  });
});

describe("Formation Pool Matching", () => {
  function createPoolWithEntries() {
    let pool = createFormationPool();

    // Human: mathematics, physics
    const { pool: p1 } = registerInPool(
      pool,
      humanDid,
      ["mathematics", "physics"],
      "human",
      { min: 100n * 10n ** 15n, max: 500n * 10n ** 15n },
      FORMATION_POOL.REGISTRATION_DEPOSIT,
      currentBlock,
    );
    pool = p1;

    // Agent 1: mathematics, computer_science (1 domain overlap)
    const { pool: p2 } = registerInPool(
      pool,
      agentDid,
      ["mathematics", "computer_science"],
      "agent",
      { min: 100n * 10n ** 15n, max: 400n * 10n ** 15n },
      FORMATION_POOL.REGISTRATION_DEPOSIT,
      currentBlock,
    );
    pool = p2;

    // Agent 2: mathematics, physics (2 domain overlap — perfect match)
    const { pool: p3 } = registerInPool(
      pool,
      agent2Did,
      ["mathematics", "physics"],
      "agent",
      { min: 100n * 10n ** 15n, max: 500n * 10n ** 15n },
      FORMATION_POOL.REGISTRATION_DEPOSIT,
      currentBlock,
    );
    pool = p3;

    return pool;
  }

  it("should find best match by domain overlap + stake proximity", () => {
    const pool = createPoolWithEntries();
    const { matchDid, score } = findBestMatch(
      pool,
      humanDid,
      currentBlock + 10n,
    );

    // Agent 2 has perfect domain overlap AND identical stake range
    expect(matchDid).toBe(agent2Did);
    expect(score).toBeGreaterThan(0);
  });

  it("should return null when no matches exist", () => {
    const pool = createFormationPool();
    const { pool: updated } = registerInPool(
      pool,
      humanDid,
      ["mathematics"],
      "human",
      defaultStakeRange,
      FORMATION_POOL.REGISTRATION_DEPOSIT,
      currentBlock,
    );

    const { matchDid } = findBestMatch(updated, humanDid, currentBlock + 10n);
    expect(matchDid).toBeNull();
  });

  it("should skip expired entries in matching", () => {
    let pool = createFormationPool();

    const { pool: p1 } = registerInPool(
      pool,
      humanDid,
      ["mathematics"],
      "human",
      defaultStakeRange,
      FORMATION_POOL.REGISTRATION_DEPOSIT,
      currentBlock,
    );
    pool = p1;

    const { pool: p2 } = registerInPool(
      pool,
      agentDid,
      ["mathematics"],
      "agent",
      defaultStakeRange,
      FORMATION_POOL.REGISTRATION_DEPOSIT,
      currentBlock,
    );
    pool = p2;

    // Check match at a time after agent entry expired
    const expiredBlock = currentBlock + FORMATION_POOL.ENTRY_EXPIRY_BLOCKS + 1n;
    const { matchDid } = findBestMatch(pool, humanDid, expiredBlock);
    expect(matchDid).toBeNull();
  });

  it("should respect role compatibility (human matches agent)", () => {
    let pool = createFormationPool();

    const { pool: p1 } = registerInPool(
      pool,
      humanDid,
      ["mathematics"],
      "human",
      defaultStakeRange,
      FORMATION_POOL.REGISTRATION_DEPOSIT,
      currentBlock,
    );
    pool = p1;

    // Another human — should NOT match with human
    const human2 = "did:lgm:human-pool-002" as DID;
    const { pool: p2 } = registerInPool(
      pool,
      human2,
      ["mathematics"],
      "human",
      defaultStakeRange,
      FORMATION_POOL.REGISTRATION_DEPOSIT,
      currentBlock,
    );
    pool = p2;

    const { matchDid } = findBestMatch(pool, humanDid, currentBlock + 10n);
    expect(matchDid).toBeNull();
  });

  it("should match 'either' role with any role", () => {
    let pool = createFormationPool();

    const { pool: p1 } = registerInPool(
      pool,
      humanDid,
      ["mathematics"],
      "human",
      defaultStakeRange,
      FORMATION_POOL.REGISTRATION_DEPOSIT,
      currentBlock,
    );
    pool = p1;

    const eitherDid = "did:lgm:either-001" as DID;
    const { pool: p2 } = registerInPool(
      pool,
      eitherDid,
      ["mathematics"],
      "either",
      defaultStakeRange,
      FORMATION_POOL.REGISTRATION_DEPOSIT,
      currentBlock,
    );
    pool = p2;

    const { matchDid } = findBestMatch(pool, humanDid, currentBlock + 10n);
    expect(matchDid).toBe(eitherDid);
  });

  it("should skip entries with non-overlapping stake ranges", () => {
    let pool = createFormationPool();

    const { pool: p1 } = registerInPool(
      pool,
      humanDid,
      ["mathematics"],
      "human",
      { min: 100n * 10n ** 15n, max: 200n * 10n ** 15n },
      FORMATION_POOL.REGISTRATION_DEPOSIT,
      currentBlock,
    );
    pool = p1;

    const { pool: p2 } = registerInPool(
      pool,
      agentDid,
      ["mathematics"],
      "agent",
      { min: 500n * 10n ** 15n, max: 1000n * 10n ** 15n },
      FORMATION_POOL.REGISTRATION_DEPOSIT,
      currentBlock,
    );
    pool = p2;

    const { matchDid } = findBestMatch(pool, humanDid, currentBlock + 10n);
    expect(matchDid).toBeNull();
  });
});

describe("Formation Pool Match Processing", () => {
  it("should process a match and update both entries", () => {
    let pool = createFormationPool();

    const { pool: p1 } = registerInPool(
      pool,
      humanDid,
      ["mathematics"],
      "human",
      defaultStakeRange,
      FORMATION_POOL.REGISTRATION_DEPOSIT,
      currentBlock,
    );
    pool = p1;

    const { pool: p2 } = registerInPool(
      pool,
      agentDid,
      ["mathematics"],
      "agent",
      defaultStakeRange,
      FORMATION_POOL.REGISTRATION_DEPOSIT,
      currentBlock,
    );
    pool = p2;

    const {
      pool: updated,
      match,
      error,
    } = processMatch(pool, humanDid, agentDid, currentBlock + 50n);

    expect(error).toBeUndefined();
    expect(match.humanDid).toBe(humanDid);
    expect(match.agentDid).toBe(agentDid);
    expect(match.accepted).toBe(false);
    expect(match.matchScore).toBeGreaterThan(0);

    const humanEntry = updated.entries.get(humanDid);
    const agentEntry = updated.entries.get(agentDid);

    expect(humanEntry!.status).toBe("matched");
    expect(humanEntry!.matchedWith).toBe(agentDid);
    expect(agentEntry!.status).toBe("matched");
    expect(agentEntry!.matchedWith).toBe(humanDid);
    expect(updated.totalMatches).toBe(1);
  });

  it("should reject match with non-active entry", () => {
    const pool = createFormationPool();
    const { error } = processMatch(pool, humanDid, agentDid, currentBlock);

    expect(error).toContain("not active");
  });
});

describe("Formation Pool Withdrawal", () => {
  it("should allow withdrawal and refund deposit", () => {
    const pool = createFormationPool();
    const { pool: updated } = registerInPool(
      pool,
      humanDid,
      ["mathematics"],
      "human",
      defaultStakeRange,
      FORMATION_POOL.REGISTRATION_DEPOSIT,
      currentBlock,
    );

    const {
      pool: afterWithdraw,
      refund,
      error,
    } = withdrawFromPool(updated, humanDid, currentBlock + 500n);

    expect(error).toBeUndefined();
    expect(refund).toBe(FORMATION_POOL.REGISTRATION_DEPOSIT);
    expect(afterWithdraw.entries.get(humanDid)!.status).toBe("withdrawn");
  });

  it("should enforce withdrawal cooldown", () => {
    const pool = createFormationPool();
    const { pool: updated } = registerInPool(
      pool,
      humanDid,
      ["mathematics"],
      "human",
      defaultStakeRange,
      FORMATION_POOL.REGISTRATION_DEPOSIT,
      currentBlock,
    );

    const lastWithdrawalBlock = currentBlock - 100n;
    const { error } = withdrawFromPool(
      updated,
      humanDid,
      currentBlock,
      lastWithdrawalBlock,
    );

    expect(error).toContain("cooldown");
  });

  it("should allow withdrawal after cooldown expires", () => {
    const pool = createFormationPool();
    const { pool: updated } = registerInPool(
      pool,
      humanDid,
      ["mathematics"],
      "human",
      defaultStakeRange,
      FORMATION_POOL.REGISTRATION_DEPOSIT,
      currentBlock,
    );

    const lastWithdrawalBlock = currentBlock - 500n;
    const withdrawBlock =
      lastWithdrawalBlock + FORMATION_POOL.WITHDRAWAL_COOLDOWN_BLOCKS + 1n;

    const { error, refund } = withdrawFromPool(
      updated,
      humanDid,
      withdrawBlock,
      lastWithdrawalBlock,
    );

    expect(error).toBeUndefined();
    expect(refund).toBe(FORMATION_POOL.REGISTRATION_DEPOSIT);
  });

  it("should reject withdrawal for non-existent entry", () => {
    const pool = createFormationPool();
    const { error } = withdrawFromPool(pool, humanDid, currentBlock);

    expect(error).toContain("No active pool entry");
  });
});

describe("Formation Pool Entry Expiry", () => {
  it("should expire stale entries", () => {
    const pool = createFormationPool();

    const { pool: updated } = registerInPool(
      pool,
      humanDid,
      ["mathematics"],
      "human",
      defaultStakeRange,
      FORMATION_POOL.REGISTRATION_DEPOSIT,
      currentBlock,
    );

    const expiryBlock = currentBlock + FORMATION_POOL.ENTRY_EXPIRY_BLOCKS;
    const {
      pool: expired,
      expiredDids,
      totalRefund,
    } = expireStaleEntries(updated, expiryBlock);

    expect(expiredDids).toContain(humanDid);
    expect(totalRefund).toBe(FORMATION_POOL.REGISTRATION_DEPOSIT);
    expect(expired.entries.get(humanDid)!.status).toBe("expired");
  });

  it("should not expire entries before expiry block", () => {
    const pool = createFormationPool();

    const { pool: updated } = registerInPool(
      pool,
      humanDid,
      ["mathematics"],
      "human",
      defaultStakeRange,
      FORMATION_POOL.REGISTRATION_DEPOSIT,
      currentBlock,
    );

    const beforeExpiry = currentBlock + FORMATION_POOL.ENTRY_EXPIRY_BLOCKS - 1n;
    const { expiredDids } = expireStaleEntries(updated, beforeExpiry);

    expect(expiredDids).toHaveLength(0);
  });

  it("should handle multiple expired entries", () => {
    let pool = createFormationPool();

    const { pool: p1 } = registerInPool(
      pool,
      humanDid,
      ["mathematics"],
      "human",
      defaultStakeRange,
      FORMATION_POOL.REGISTRATION_DEPOSIT,
      currentBlock,
    );
    pool = p1;

    const { pool: p2 } = registerInPool(
      pool,
      agentDid,
      ["physics"],
      "agent",
      defaultStakeRange,
      FORMATION_POOL.REGISTRATION_DEPOSIT,
      currentBlock,
    );
    pool = p2;

    const expiryBlock = currentBlock + FORMATION_POOL.ENTRY_EXPIRY_BLOCKS;
    const { expiredDids, totalRefund } = expireStaleEntries(pool, expiryBlock);

    expect(expiredDids).toHaveLength(2);
    expect(totalRefund).toBe(FORMATION_POOL.REGISTRATION_DEPOSIT * 2n);
  });
});

describe("Formation Pool State", () => {
  it("should create empty pool state", () => {
    const pool = createFormationPool();

    expect(pool.entries.size).toBe(0);
    expect(pool.matchHistory).toHaveLength(0);
    expect(pool.totalRegistrations).toBe(0);
    expect(pool.totalMatches).toBe(0);
  });

  it("should track match history", () => {
    let pool = createFormationPool();

    const { pool: p1 } = registerInPool(
      pool,
      humanDid,
      ["mathematics"],
      "human",
      defaultStakeRange,
      FORMATION_POOL.REGISTRATION_DEPOSIT,
      currentBlock,
    );
    pool = p1;

    const { pool: p2 } = registerInPool(
      pool,
      agentDid,
      ["mathematics"],
      "agent",
      defaultStakeRange,
      FORMATION_POOL.REGISTRATION_DEPOSIT,
      currentBlock,
    );
    pool = p2;

    const { pool: matched } = processMatch(
      pool,
      humanDid,
      agentDid,
      currentBlock,
    );

    expect(matched.matchHistory).toHaveLength(1);
    expect(matched.matchHistory[0].humanDid).toBe(humanDid);
    expect(matched.matchHistory[0].agentDid).toBe(agentDid);
  });
});
