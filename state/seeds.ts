/**
 * Seed State Tree
 *
 * Sparse Merkle Tree for Tree of Life opportunity seeds.
 * Key: hash(seedId) - 32 bytes
 * Value: serialized OpportunitySeed
 *
 * Provides:
 * - Cryptographic proof of seed existence
 * - Seed lookup by domain and status
 * - On-chain opportunity lifecycle tracking
 */

import {
  InMemorySMT,
  type MerkleRoot,
  type MerkleProofData,
} from "../core/crypto/merkle.js";
import { sha512_256 } from "../core/crypto/hashing.js";
import type { OpportunitySeed, SeedStatus } from "../core/tree/types.js";
import { StateError } from "./types.js";

// ============ Serialization ============

/**
 * Serialize OpportunitySeed to bytes for SMT storage.
 * BigInt fields are converted to strings for JSON compatibility.
 */
function serializeSeed(seed: OpportunitySeed): Uint8Array {
  const json = JSON.stringify({
    id: seed.id,
    detectedAtBlock: seed.detectedAtBlock.toString(),
    knowledgeDomain: seed.knowledgeDomain,
    signal: seed.signal,
    confidence: seed.confidence,
    estimatedDemand: seed.estimatedDemand.toString(),
    estimatedEffort: seed.estimatedEffort,
    estimatedROI: seed.estimatedROI,
    status: seed.status,
    claimedBy: seed.claimedBy,
    projectId: seed.projectId,
    expiresAtBlock: seed.expiresAtBlock.toString(),
  });
  return new TextEncoder().encode(json);
}

/**
 * Deserialize OpportunitySeed from bytes.
 */
function deserializeSeed(bytes: Uint8Array): OpportunitySeed {
  const json = new TextDecoder().decode(bytes);
  const obj = JSON.parse(json);
  return {
    id: obj.id,
    detectedAtBlock: BigInt(obj.detectedAtBlock),
    knowledgeDomain: obj.knowledgeDomain,
    signal: obj.signal,
    confidence: obj.confidence,
    estimatedDemand: BigInt(obj.estimatedDemand),
    estimatedEffort: obj.estimatedEffort,
    estimatedROI: obj.estimatedROI,
    status: obj.status as SeedStatus,
    claimedBy: obj.claimedBy,
    projectId: obj.projectId,
    expiresAtBlock: BigInt(obj.expiresAtBlock),
  };
}

/**
 * Convert seed ID to SMT key
 */
function seedIdToKey(seedId: string): Uint8Array {
  return sha512_256(seedId);
}

// ============ Seed State Tree ============

export class SeedStateTree {
  private smt: InMemorySMT;
  private cache: Map<string, OpportunitySeed> = new Map();

  constructor() {
    this.smt = new InMemorySMT();
  }

  /**
   * Get the current root hash
   */
  getRoot(): MerkleRoot {
    return this.smt.getRoot();
  }

  // ============ CRUD Operations ============

  /**
   * Get seed by ID.
   * WARNING: Returns a cached reference. Do NOT mutate the returned object directly.
   */
  getSeed(seedId: string): OpportunitySeed | null {
    const cached = this.cache.get(seedId);
    if (cached) return cached;

    const key = seedIdToKey(seedId);
    const value = this.smt.get(key);
    if (!value) return null;

    const seed = deserializeSeed(value);
    this.cache.set(seedId, seed);
    return seed;
  }

  /**
   * Add a new seed
   */
  addSeed(seed: OpportunitySeed): MerkleRoot {
    if (this.getSeed(seed.id) !== null) {
      throw new StateError("Seed already exists", "INVALID_TRANSITION", {
        seedId: seed.id,
      });
    }

    const key = seedIdToKey(seed.id);
    const value = serializeSeed(seed);
    const newRoot = this.smt.set(key, value);

    this.cache.set(seed.id, seed);
    return newRoot;
  }

  /**
   * Update an existing seed
   */
  updateSeed(seedId: string, updates: Partial<OpportunitySeed>): MerkleRoot {
    const existing = this.getSeed(seedId);
    if (!existing) {
      throw new StateError("Seed not found", "SEED_NOT_FOUND", { seedId });
    }

    const updated: OpportunitySeed = {
      ...existing,
      ...updates,
    };

    const key = seedIdToKey(seedId);
    const value = serializeSeed(updated);
    const newRoot = this.smt.set(key, value);

    this.cache.set(seedId, updated);
    return newRoot;
  }

  // ============ Query Operations ============

  /**
   * Get all seeds in a domain
   */
  getSeedsByDomain(domain: string): OpportunitySeed[] {
    const results: OpportunitySeed[] = [];
    for (const seed of this.cache.values()) {
      if (seed.knowledgeDomain === domain) {
        results.push(seed);
      }
    }
    return results;
  }

  /**
   * Get all seeds with a given status
   */
  getSeedsByStatus(status: SeedStatus): OpportunitySeed[] {
    const results: OpportunitySeed[] = [];
    for (const seed of this.cache.values()) {
      if (seed.status === status) {
        results.push(seed);
      }
    }
    return results;
  }

  /**
   * Get seeds that are available to claim (detected + not expired)
   */
  getAvailableSeeds(currentBlock: bigint): OpportunitySeed[] {
    const results: OpportunitySeed[] = [];
    for (const seed of this.cache.values()) {
      if (seed.status === "detected" && seed.expiresAtBlock > currentBlock) {
        results.push(seed);
      }
    }
    return results;
  }

  /**
   * Get total seed count
   */
  getSeedCount(): number {
    return this.cache.size;
  }

  // ============ Proofs ============

  /**
   * Generate Merkle proof for a seed.
   */
  proveSeed(seedId: string): MerkleProofData {
    const key = seedIdToKey(seedId);
    return this.smt.prove(key);
  }

  // ============ Persistence Export/Restore ============

  /**
   * Export all seeds for persistence
   */
  exportSeeds(): OpportunitySeed[] {
    return Array.from(this.cache.values());
  }

  /**
   * Restore seeds from persistence
   */
  restoreSeeds(seeds: OpportunitySeed[]): void {
    for (const seed of seeds) {
      const key = seedIdToKey(seed.id);
      const value = serializeSeed(seed);
      this.smt.set(key, value);
      this.cache.set(seed.id, seed);
    }
  }

  // ============ Utilities ============

  /**
   * Clone for speculative execution
   */
  clone(): SeedStateTree {
    const cloned = new SeedStateTree();

    for (const [seedId, seed] of this.cache) {
      const clonedSeed: OpportunitySeed = {
        ...seed,
        signal: { ...seed.signal },
      };

      cloned.cache.set(seedId, clonedSeed);
      const key = seedIdToKey(seedId);
      cloned.smt.set(key, serializeSeed(clonedSeed));
    }

    return cloned;
  }
}
