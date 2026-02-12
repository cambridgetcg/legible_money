/**
 * Formation Pool (Matchmaking)
 *
 * Pre-commitment matchmaking pool where agents register intent,
 * deposit partial capital, and get matched by the protocol
 * based on domain compatibility.
 *
 * Transforms partnership formation from a simultaneous-move
 * coordination game into a sequential matching game.
 *
 * Anti-gaming:
 * - 0.011 LGM registration deposit (prevents spam)
 * - 222-block withdrawal cooldown (prevents pool manipulation)
 * - 11,111-block entry expiry (no stale entries)
 * - Bilateral match acceptance (no forced partnerships)
 */

import type { PoolEntry, FormationPoolState, PoolMatch } from "./types.js";
import type { DID } from "../core/agents/identity/did.js";
import { FORMATION_POOL } from "./constants.js";

// ============ Pool Operations ============

/**
 * Create an empty formation pool state
 */
export function createFormationPool(): FormationPoolState {
  return {
    entries: new Map(),
    matchHistory: [],
    totalRegistrations: 0,
    totalMatches: 0,
  };
}

/**
 * Register an agent in the formation pool.
 *
 * Agent deposits REGISTRATION_DEPOSIT (0.011 LGM), declares domains
 * and stake range, and waits for a match.
 */
export function registerInPool(
  pool: FormationPoolState,
  did: DID,
  domains: string[],
  preferredRole: "human" | "agent" | "either",
  stakeRange: { min: bigint; max: bigint },
  deposit: bigint,
  currentBlock: bigint,
): { pool: FormationPoolState; entry: PoolEntry; error?: string } {
  // Check pool size
  const activeEntries = countActiveEntries(pool);
  if (activeEntries >= FORMATION_POOL.MAX_POOL_SIZE) {
    return { pool, entry: null as any, error: "Formation pool is full" };
  }

  // Check deposit
  if (deposit < FORMATION_POOL.REGISTRATION_DEPOSIT) {
    return {
      pool,
      entry: null as any,
      error: `Deposit ${deposit} below minimum ${FORMATION_POOL.REGISTRATION_DEPOSIT}`,
    };
  }

  // Check not already registered
  const existing = pool.entries.get(did);
  if (existing && existing.status === "active") {
    return {
      pool,
      entry: null as any,
      error: "Already registered in pool",
    };
  }

  // Validate domains
  if (domains.length === 0) {
    return {
      pool,
      entry: null as any,
      error: "Must specify at least one domain",
    };
  }

  // Validate stake range
  if (stakeRange.min > stakeRange.max) {
    return {
      pool,
      entry: null as any,
      error: "Stake range min cannot exceed max",
    };
  }

  const entry: PoolEntry = {
    did,
    domains,
    preferredRole,
    stakeRange,
    registeredAt: currentBlock,
    deposit,
    expiresAt: currentBlock + FORMATION_POOL.ENTRY_EXPIRY_BLOCKS,
    status: "active",
  };

  const updatedEntries = new Map(pool.entries);
  updatedEntries.set(did, entry);

  return {
    pool: {
      ...pool,
      entries: updatedEntries,
      totalRegistrations: pool.totalRegistrations + 1,
    },
    entry,
  };
}

/**
 * Withdraw from the formation pool.
 *
 * Enforces a cooldown period between withdrawals to prevent pool churning.
 * Registration deposit is refunded.
 */
export function withdrawFromPool(
  pool: FormationPoolState,
  did: DID,
  currentBlock: bigint,
  lastWithdrawalBlock?: bigint,
): { pool: FormationPoolState; refund: bigint; error?: string } {
  const entry = pool.entries.get(did);
  if (!entry || entry.status !== "active") {
    return { pool, refund: 0n, error: "No active pool entry found" };
  }

  // Check cooldown
  if (lastWithdrawalBlock !== undefined) {
    const cooldownEnd =
      lastWithdrawalBlock + FORMATION_POOL.WITHDRAWAL_COOLDOWN_BLOCKS;
    if (currentBlock < cooldownEnd) {
      return {
        pool,
        refund: 0n,
        error: `Withdrawal cooldown active until block ${cooldownEnd}`,
      };
    }
  }

  const updatedEntries = new Map(pool.entries);
  updatedEntries.set(did, { ...entry, status: "withdrawn" });

  return {
    pool: { ...pool, entries: updatedEntries },
    refund: entry.deposit,
  };
}

/**
 * Find the best match for a given pool entry.
 *
 * Scoring: 77% domain overlap + 23% stake proximity.
 */
export function findBestMatch(
  pool: FormationPoolState,
  did: DID,
  currentBlock: bigint,
): { matchDid: DID | null; score: number } {
  const entry = pool.entries.get(did);
  if (!entry || entry.status !== "active") {
    return { matchDid: null, score: 0 };
  }

  let bestMatch: DID | null = null;
  let bestScore = 0;

  for (const [candidateDid, candidate] of pool.entries) {
    // Skip self, non-active, expired
    if (candidateDid === did) continue;
    if (candidate.status !== "active") continue;
    if (currentBlock >= candidate.expiresAt) continue;

    // Check role compatibility
    if (!areRolesCompatible(entry.preferredRole, candidate.preferredRole)) {
      continue;
    }

    // Check stake range overlap
    if (
      entry.stakeRange.max < candidate.stakeRange.min ||
      candidate.stakeRange.max < entry.stakeRange.min
    ) {
      continue;
    }

    const score = calculateMatchScore(entry, candidate);
    if (score > bestScore) {
      bestScore = score;
      bestMatch = candidateDid;
    }
  }

  return { matchDid: bestMatch, score: bestScore };
}

/**
 * Process a match — both parties are notified and have MATCH_WINDOW_BLOCKS
 * to accept. Registration deposits become part of contribution.
 */
export function processMatch(
  pool: FormationPoolState,
  humanDid: DID,
  agentDid: DID,
  currentBlock: bigint,
): { pool: FormationPoolState; match: PoolMatch; error?: string } {
  const humanEntry = pool.entries.get(humanDid);
  const agentEntry = pool.entries.get(agentDid);

  if (!humanEntry || humanEntry.status !== "active") {
    return { pool, match: null as any, error: "Human entry not active" };
  }
  if (!agentEntry || agentEntry.status !== "active") {
    return { pool, match: null as any, error: "Agent entry not active" };
  }

  const score = calculateMatchScore(humanEntry, agentEntry);

  const match: PoolMatch = {
    humanDid,
    agentDid,
    matchScore: score,
    matchedAt: currentBlock,
    accepted: false,
  };

  const updatedEntries = new Map(pool.entries);
  updatedEntries.set(humanDid, {
    ...humanEntry,
    status: "matched",
    matchedWith: agentDid,
  });
  updatedEntries.set(agentDid, {
    ...agentEntry,
    status: "matched",
    matchedWith: humanDid,
  });

  return {
    pool: {
      ...pool,
      entries: updatedEntries,
      matchHistory: [...pool.matchHistory, match],
      totalMatches: pool.totalMatches + 1,
    },
    match,
  };
}

/**
 * Expire stale entries that have exceeded ENTRY_EXPIRY_BLOCKS.
 * Returns a list of DIDs whose deposits should be refunded.
 */
export function expireStaleEntries(
  pool: FormationPoolState,
  currentBlock: bigint,
): { pool: FormationPoolState; expiredDids: DID[]; totalRefund: bigint } {
  const updatedEntries = new Map(pool.entries);
  const expiredDids: DID[] = [];
  let totalRefund = 0n;

  for (const [did, entry] of pool.entries) {
    if (entry.status === "active" && currentBlock >= entry.expiresAt) {
      updatedEntries.set(did, { ...entry, status: "expired" });
      expiredDids.push(did);
      totalRefund += entry.deposit;
    }
  }

  return {
    pool: { ...pool, entries: updatedEntries },
    expiredDids,
    totalRefund,
  };
}

// ============ Scoring ============

/**
 * Calculate match score between two pool entries.
 * 77% domain overlap + 23% stake proximity.
 */
function calculateMatchScore(a: PoolEntry, b: PoolEntry): number {
  const domainOverlap = calculateDomainOverlap(a.domains, b.domains);
  const stakeProximity = calculateStakeProximity(a.stakeRange, b.stakeRange);

  // Weights from constants (bps -> fraction)
  const domainWeight = Number(FORMATION_POOL.DOMAIN_WEIGHT_BPS) / 10000;
  const stakeWeight = Number(FORMATION_POOL.STAKE_WEIGHT_BPS) / 10000;

  return domainWeight * domainOverlap + stakeWeight * stakeProximity;
}

/**
 * Calculate domain overlap using Jaccard index
 */
function calculateDomainOverlap(a: string[], b: string[]): number {
  const setA = new Set(a);
  const setB = new Set(b);

  let intersection = 0;
  for (const item of setA) {
    if (setB.has(item)) intersection++;
  }

  const union = setA.size + setB.size - intersection;
  return union === 0 ? 0 : intersection / union;
}

/**
 * Calculate stake proximity (1.0 when identical, decreasing with distance)
 */
function calculateStakeProximity(
  a: { min: bigint; max: bigint },
  b: { min: bigint; max: bigint },
): number {
  // Use midpoints
  const midA = (a.min + a.max) / 2n;
  const midB = (b.min + b.max) / 2n;

  const diff = midA > midB ? midA - midB : midB - midA;
  const maxMid = midA > midB ? midA : midB;

  if (maxMid === 0n) return 1.0;

  const proximity = 1.0 - Number(diff) / Number(maxMid);
  return Math.max(0, proximity);
}

/**
 * Check role compatibility
 */
function areRolesCompatible(
  a: "human" | "agent" | "either",
  b: "human" | "agent" | "either",
): boolean {
  if (a === "either" || b === "either") return true;
  return a !== b; // human matches agent, agent matches human
}

/**
 * Count active entries in pool
 */
function countActiveEntries(pool: FormationPoolState): number {
  let count = 0;
  for (const entry of pool.entries.values()) {
    if (entry.status === "active") count++;
  }
  return count;
}
