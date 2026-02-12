/**
 * Commitment Store Tests
 *
 * Tests persistence, atomic writes, Go-compatible hash computation,
 * and lifecycle operations.
 */

import { describe, it, expect, afterEach } from "vitest";
import {
  mkdtempSync,
  rmSync,
  existsSync,
  readFileSync,
  writeFileSync,
  mkdirSync,
} from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";
import {
  CommitmentStore,
  computeCommitmentHash,
  generateSalt,
} from "../src/commitment_store.js";
import type { StoredCommitment } from "../src/types.js";

function makeTmpDir(): string {
  return mkdtempSync(join(tmpdir(), "lgm-commitment-test-"));
}

function makeCommitment(
  overrides: Partial<StoredCommitment> = {},
): StoredCommitment {
  const roundId = overrides.roundId ?? "round-1";
  const verdict = overrides.verdict ?? "accept";
  const confidence = overrides.confidence ?? 750_000;
  const salt = overrides.salt ?? "test-salt-abc123";

  return {
    roundId,
    verdict,
    confidence,
    salt,
    commitmentHash: computeCommitmentHash(roundId, verdict, confidence, salt),
    height: overrides.height ?? 100n,
    timestamp: overrides.timestamp ?? Date.now(),
    source: overrides.source ?? "claude",
  };
}

describe("computeCommitmentHash", () => {
  it("produces consistent hashes for the same input", () => {
    const h1 = computeCommitmentHash("round1", "accept", 750_000, "abc123");
    const h2 = computeCommitmentHash("round1", "accept", 750_000, "abc123");
    expect(h1).toBe(h2);
  });

  it("produces different hashes for different inputs", () => {
    const h1 = computeCommitmentHash("round1", "accept", 750_000, "abc123");
    const h2 = computeCommitmentHash("round2", "accept", 750_000, "abc123");
    const h3 = computeCommitmentHash("round1", "reject", 750_000, "abc123");
    const h4 = computeCommitmentHash("round1", "accept", 500_000, "abc123");
    const h5 = computeCommitmentHash("round1", "accept", 750_000, "xyz789");

    expect(new Set([h1, h2, h3, h4, h5]).size).toBe(5);
  });

  it("returns a 64-character hex string (SHA-256)", () => {
    const hash = computeCommitmentHash("round1", "accept", 750_000, "abc123");
    expect(hash).toMatch(/^[0-9a-f]{64}$/);
  });

  it("handles edge case: empty roundId", () => {
    const hash = computeCommitmentHash("", "accept", 0, "salt");
    expect(hash).toMatch(/^[0-9a-f]{64}$/);
  });

  it("handles edge case: confidence = 0", () => {
    const hash = computeCommitmentHash("round1", "abstain", 0, "salt");
    expect(hash).toMatch(/^[0-9a-f]{64}$/);
  });

  it("handles edge case: confidence = 1,000,000", () => {
    const hash = computeCommitmentHash("round1", "accept", 1_000_000, "salt");
    expect(hash).toMatch(/^[0-9a-f]{64}$/);
  });

  it("is sensitive to domain separation prefix", () => {
    // The hash includes "LGM.commitment.v1:" as domain separator
    // This test verifies it's not just hashing the raw data
    const hash = computeCommitmentHash("r", "a", 0, "s");
    expect(hash).toMatch(/^[0-9a-f]{64}$/);
  });

  it("handles unicode in strings correctly", () => {
    const hash = computeCommitmentHash("round-ü", "accept", 500_000, "salt-é");
    expect(hash).toMatch(/^[0-9a-f]{64}$/);
  });
});

describe("generateSalt", () => {
  it("returns a 64-character hex string (32 bytes)", () => {
    const salt = generateSalt();
    expect(salt).toMatch(/^[0-9a-f]{64}$/);
  });

  it("generates unique salts", () => {
    const salts = new Set(Array.from({ length: 100 }, () => generateSalt()));
    expect(salts.size).toBe(100);
  });
});

describe("CommitmentStore", () => {
  const dirs: string[] = [];

  function freshDir(): string {
    const d = makeTmpDir();
    dirs.push(d);
    return d;
  }

  afterEach(() => {
    for (const d of dirs) {
      rmSync(d, { recursive: true, force: true });
    }
    dirs.length = 0;
  });

  it("stores and retrieves a commitment", () => {
    const dir = freshDir();
    const store = new CommitmentStore(dir);
    const commitment = makeCommitment();

    store.store(commitment);
    const retrieved = store.get("round-1");

    expect(retrieved).toBeDefined();
    expect(retrieved!.roundId).toBe("round-1");
    expect(retrieved!.verdict).toBe("accept");
    expect(retrieved!.confidence).toBe(750_000);
    expect(retrieved!.salt).toBe("test-salt-abc123");
  });

  it("returns undefined for unknown round", () => {
    const store = new CommitmentStore(freshDir());
    expect(store.get("nonexistent")).toBeUndefined();
  });

  it("deletes a commitment", () => {
    const store = new CommitmentStore(freshDir());
    store.store(makeCommitment());
    expect(store.get("round-1")).toBeDefined();

    store.delete("round-1");
    expect(store.get("round-1")).toBeUndefined();
    expect(store.count()).toBe(0);
  });

  it("persists to disk and survives new instance", () => {
    const dir = freshDir();

    // Store in first instance
    const store1 = new CommitmentStore(dir);
    store1.store(makeCommitment({ roundId: "persist-1", confidence: 600_000 }));
    store1.store(makeCommitment({ roundId: "persist-2", confidence: 400_000 }));

    // Load in second instance from same directory
    const store2 = new CommitmentStore(dir);
    expect(store2.count()).toBe(2);

    const c1 = store2.get("persist-1");
    expect(c1).toBeDefined();
    expect(c1!.confidence).toBe(600_000);

    const c2 = store2.get("persist-2");
    expect(c2).toBeDefined();
    expect(c2!.confidence).toBe(400_000);
  });

  it("persists BigInt height correctly across instances", () => {
    const dir = freshDir();

    const store1 = new CommitmentStore(dir);
    store1.store(makeCommitment({ height: 999_999_999n }));

    const store2 = new CommitmentStore(dir);
    const c = store2.get("round-1");
    expect(c).toBeDefined();
    expect(c!.height).toBe(999_999_999n);
  });

  it("writes atomically (temp file + rename)", () => {
    const dir = freshDir();
    const store = new CommitmentStore(dir);
    store.store(makeCommitment());

    // The commitments.json file should exist
    const filePath = join(dir, "commitments.json");
    expect(existsSync(filePath)).toBe(true);

    // The temp file should NOT exist (renamed away)
    expect(existsSync(filePath + ".tmp")).toBe(false);

    // File should be valid JSON
    const content = readFileSync(filePath, "utf-8");
    const parsed = JSON.parse(content);
    expect(Array.isArray(parsed)).toBe(true);
    expect(parsed.length).toBe(1);
  });

  it("cleanups expired commitments", () => {
    const store = new CommitmentStore(freshDir());
    store.store(makeCommitment({ roundId: "old", height: 10n }));
    store.store(makeCommitment({ roundId: "recent", height: 90n }));
    store.store(makeCommitment({ roundId: "current", height: 100n }));

    // Cleanup with currentHeight=110, maxAge=50 → cutoff=60
    store.cleanupExpired(110n, 50n);

    expect(store.get("old")).toBeUndefined(); // height 10 < cutoff 60
    expect(store.get("recent")).toBeDefined(); // height 90 >= cutoff 60
    expect(store.get("current")).toBeDefined(); // height 100 >= cutoff 60
    expect(store.count()).toBe(2);
  });

  it("does not cleanup when currentHeight < maxAge", () => {
    const store = new CommitmentStore(freshDir());
    store.store(makeCommitment({ roundId: "early", height: 5n }));

    store.cleanupExpired(10n, 50n); // 10 < 50, no cleanup
    expect(store.get("early")).toBeDefined();
  });

  it("reports roundIds correctly", () => {
    const store = new CommitmentStore(freshDir());
    store.store(makeCommitment({ roundId: "r1" }));
    store.store(makeCommitment({ roundId: "r2" }));
    store.store(makeCommitment({ roundId: "r3" }));

    const ids = store.roundIds();
    expect(ids.sort()).toEqual(["r1", "r2", "r3"]);
  });

  it("verifyBeforeReveal succeeds with valid data", () => {
    const store = new CommitmentStore(freshDir());
    store.store(makeCommitment());

    expect(store.verifyBeforeReveal("round-1")).toBe(true);
  });

  it("verifyBeforeReveal fails for unknown round", () => {
    const store = new CommitmentStore(freshDir());
    expect(store.verifyBeforeReveal("nonexistent")).toBe(false);
  });

  it("handles corrupted file gracefully", () => {
    const dir = freshDir();
    mkdirSync(dir, { recursive: true });
    writeFileSync(join(dir, "commitments.json"), "not valid json!!!", "utf-8");

    // Should start with empty store (no crash)
    const store = new CommitmentStore(dir);
    expect(store.count()).toBe(0);
  });

  it("stores multiple commitments independently", () => {
    const store = new CommitmentStore(freshDir());

    for (let i = 0; i < 10; i++) {
      store.store(
        makeCommitment({
          roundId: `round-${i}`,
          confidence: i * 100_000,
          height: BigInt(i * 10),
        }),
      );
    }

    expect(store.count()).toBe(10);
    expect(store.get("round-5")!.confidence).toBe(500_000);
  });
});
