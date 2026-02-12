/**
 * Commitment Store
 *
 * Persistent commitment storage with atomic disk writes and Go-compatible
 * commitment hashing. Mirrors app/local_state.go but adds disk persistence
 * so commitments survive daemon restarts between commit and reveal phases.
 *
 * Safety invariant: ALWAYS persist to disk BEFORE submitting commitment
 * on-chain. If crash after submit, data is on disk for reveal.
 */

import { createHash, randomBytes } from "node:crypto";
import {
  readFileSync,
  writeFileSync,
  renameSync,
  mkdirSync,
  existsSync,
} from "node:fs";
import { join, dirname } from "node:path";
import type { StoredCommitment, StoredCommitmentJSON } from "./types.js";

/**
 * Compute commitment hash matching Go's ComputeCommitmentHash at app/abci.go:457-479.
 *
 * hash = SHA-256("LGM.commitment.v1:" || len(roundID) || roundID ||
 *                len(verdict) || verdict || confidence || len(salt) || salt)
 *
 * Length-prefixed with 4-byte big-endian uint32, confidence as 8-byte big-endian uint64.
 */
export function computeCommitmentHash(
  roundId: string,
  verdict: string,
  confidence: number,
  salt: string,
): string {
  const h = createHash("sha256");

  // Domain separation (matches Go: h.Write([]byte("LGM.commitment.v1:")))
  h.update("LGM.commitment.v1:");

  // Length-prefixed roundID
  const len4 = Buffer.alloc(4);
  len4.writeUInt32BE(Buffer.byteLength(roundId, "utf-8"));
  h.update(len4);
  h.update(roundId, "utf-8");

  // Length-prefixed verdict
  len4.writeUInt32BE(Buffer.byteLength(verdict, "utf-8"));
  h.update(len4);
  h.update(verdict, "utf-8");

  // Fixed-size confidence (8 bytes, big-endian uint64)
  const conf8 = Buffer.alloc(8);
  conf8.writeBigUInt64BE(BigInt(confidence));
  h.update(conf8);

  // Length-prefixed salt
  len4.writeUInt32BE(Buffer.byteLength(salt, "utf-8"));
  h.update(len4);
  h.update(salt, "utf-8");

  return h.digest("hex");
}

/** Generate a 32-byte random salt as hex string */
export function generateSalt(): string {
  return randomBytes(32).toString("hex");
}

export class CommitmentStore {
  private commitments: Map<string, StoredCommitment> = new Map();
  private readonly filePath: string;

  constructor(dataDir: string) {
    this.filePath = join(dataDir, "commitments.json");
    mkdirSync(dirname(this.filePath), { recursive: true });
    this.load();
  }

  /**
   * Store a commitment and immediately flush to disk.
   * MUST be called BEFORE submitting the commitment on-chain.
   */
  store(commitment: StoredCommitment): void {
    this.commitments.set(commitment.roundId, commitment);
    this.save();
  }

  /** Retrieve a stored commitment by round ID */
  get(roundId: string): StoredCommitment | undefined {
    return this.commitments.get(roundId);
  }

  /** Delete a commitment (after successful reveal or round completion) and flush */
  delete(roundId: string): void {
    this.commitments.delete(roundId);
    this.save();
  }

  /** Number of stored commitments */
  count(): number {
    return this.commitments.size;
  }

  /** All stored round IDs (for reveal phase iteration) */
  roundIds(): string[] {
    return Array.from(this.commitments.keys());
  }

  /**
   * Remove commitments for rounds older than maxAge blocks.
   * Mirrors app/local_state.go:CleanupExpired.
   */
  cleanupExpired(currentHeight: bigint, maxAge: bigint): void {
    if (currentHeight < maxAge) return;
    const cutoff = currentHeight - maxAge;
    let changed = false;

    for (const [id, c] of this.commitments) {
      if (c.height < cutoff) {
        this.commitments.delete(id);
        changed = true;
      }
    }

    if (changed) this.save();
  }

  /**
   * Verify commitment hash before revealing — recompute from stored data
   * and ensure it matches. Prevents revealing with corrupted data.
   */
  verifyBeforeReveal(roundId: string): boolean {
    const c = this.commitments.get(roundId);
    if (!c) return false;

    const recomputed = computeCommitmentHash(
      roundId,
      c.verdict,
      c.confidence,
      c.salt,
    );
    return recomputed === c.commitmentHash;
  }

  /** Load commitments from disk (called on construction) */
  private load(): void {
    if (!existsSync(this.filePath)) return;

    try {
      const raw = readFileSync(this.filePath, "utf-8");
      const entries: StoredCommitmentJSON[] = JSON.parse(raw);

      for (const entry of entries) {
        this.commitments.set(entry.roundId, {
          ...entry,
          height: BigInt(entry.height),
        });
      }
    } catch {
      // Corrupted file — start fresh (commitments are lost but safe)
    }
  }

  /**
   * Persist to disk using atomic write (tmp + rename).
   * Prevents corruption from crashes mid-write.
   */
  private save(): void {
    const entries: StoredCommitmentJSON[] = [];
    for (const c of this.commitments.values()) {
      entries.push({
        ...c,
        height: c.height.toString(),
      });
    }

    const json = JSON.stringify(entries, null, 2);
    const tmpPath = this.filePath + ".tmp";

    writeFileSync(tmpPath, json, "utf-8");
    renameSync(tmpPath, this.filePath);
  }
}
