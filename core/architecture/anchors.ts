/**
 * Hash Anchors for Immutable Modules
 *
 * Each immutable module has a content hash anchor that is recorded
 * in the genesis block. Any node can verify that immutable modules
 * have not been tampered with by comparing against these anchors.
 *
 * PP HIERARCHY:
 *   TRUTH: Cryptographic proof of module integrity
 *   UNDERSTANDING: Clear verification protocol
 *   BEAUTY: Simple hash-based immutability
 *   JUSTICE: All nodes can verify independently
 *   CREATIVITY: Enables trust in foundation
 */

import { domainHash, HASH_DOMAINS } from "../crypto/hashing.js";
import { bytesToHex } from "../crypto/keys.js";
import { IMMUTABLE_PATTERNS } from "./tiers.js";

// ============ Anchor Types ============

/**
 * Hash anchor for an immutable module.
 */
export interface ModuleAnchor {
  /** Module path relative to project root */
  path: string;
  /** SHA-512 hash of module content */
  contentHash: string;
  /** Block height when anchor was recorded */
  anchoredAt: bigint;
  /** Version of the module */
  version: string;
}

/**
 * Anchor verification result.
 */
export interface AnchorVerification {
  path: string;
  expected: string;
  actual: string;
  valid: boolean;
  error?: string;
}

/**
 * Full anchor set for genesis.
 */
export interface GenesisAnchors {
  version: "1.0.0";
  timestamp: bigint;
  anchors: ModuleAnchor[];
  combinedHash: string; // Hash of all anchors combined
}

// ============ Anchor Computation ============

/**
 * Compute content hash for a module.
 * Uses domain-separated SHA-512 hashing.
 */
export function computeModuleHash(content: string): string {
  const encoder = new TextEncoder();
  const contentBytes = encoder.encode(content);
  const hash = domainHash(HASH_DOMAINS.STATE, contentBytes);
  return bytesToHex(hash);
}

/**
 * Create an anchor for a module.
 */
export function createModuleAnchor(
  path: string,
  content: string,
  currentBlock: bigint,
  version: string = "1.0.0",
): ModuleAnchor {
  return {
    path,
    contentHash: computeModuleHash(content),
    anchoredAt: currentBlock,
    version,
  };
}

/**
 * Compute combined hash of all anchors.
 * This single hash represents the entire immutable foundation.
 */
export function computeCombinedAnchorHash(anchors: ModuleAnchor[]): string {
  // Sort anchors by path for deterministic ordering
  const sortedAnchors = [...anchors].sort((a, b) =>
    a.path.localeCompare(b.path),
  );

  // Concatenate all content hashes
  const combined = sortedAnchors
    .map((a) => `${a.path}:${a.contentHash}`)
    .join("|");

  return computeModuleHash(combined);
}

// ============ Genesis Anchors ============

/**
 * Placeholder anchors - will be populated at genesis.
 * Each anchor is the SHA-512 hash of the module content.
 *
 * NOTE: These are PLACEHOLDER values. At genesis, actual hashes
 * will be computed from the frozen module content.
 */
export const GENESIS_ANCHORS: Record<string, string> = {
  // Core constants
  "core/constants.ts": "GENESIS_PLACEHOLDER",

  // Cryptographic primitives
  "core/crypto/hashing.ts": "GENESIS_PLACEHOLDER",
  "core/crypto/keys.ts": "GENESIS_PLACEHOLDER",
  "core/crypto/signatures.ts": "GENESIS_PLACEHOLDER",
  "core/crypto/vrf.ts": "GENESIS_PLACEHOLDER",
  "core/crypto/commitment.ts": "GENESIS_PLACEHOLDER",
  "core/crypto/merkle.ts": "GENESIS_PLACEHOLDER",

  // Logic foundations
  "core/logic/types.ts": "GENESIS_PLACEHOLDER",
  "core/logic/zones.ts": "GENESIS_PLACEHOLDER",
  "core/logic/goedel.ts": "GENESIS_PLACEHOLDER",

  // Canonical types
  "core/knowledge/types.ts": "GENESIS_PLACEHOLDER",
  "agents/identity/did.ts": "GENESIS_PLACEHOLDER",

  // Architecture module itself
  "core/architecture/tiers.ts": "GENESIS_PLACEHOLDER",
  "core/architecture/anchors.ts": "GENESIS_PLACEHOLDER",
  "core/architecture/enforcement.ts": "GENESIS_PLACEHOLDER",
};

/**
 * Create genesis anchors from current module content.
 * Called once at genesis to freeze the immutable foundation.
 */
export function createGenesisAnchors(
  moduleContents: Map<string, string>,
  genesisTimestamp: bigint,
): GenesisAnchors {
  const anchors: ModuleAnchor[] = [];

  for (const path of IMMUTABLE_PATTERNS) {
    const content = moduleContents.get(path);
    if (!content) {
      throw new Error(`Missing content for immutable module: ${path}`);
    }

    anchors.push(createModuleAnchor(path, content, 0n, "1.0.0"));
  }

  return {
    version: "1.0.0",
    timestamp: genesisTimestamp,
    anchors,
    combinedHash: computeCombinedAnchorHash(anchors),
  };
}

// ============ Verification ============

/**
 * Verify a single module against its anchor.
 */
export function verifyModuleAnchor(
  path: string,
  content: string,
  expectedHash: string,
): AnchorVerification {
  const actualHash = computeModuleHash(content);

  return {
    path,
    expected: expectedHash,
    actual: actualHash,
    valid: actualHash === expectedHash,
    error:
      actualHash !== expectedHash
        ? "Content hash mismatch - module may have been tampered with"
        : undefined,
  };
}

/**
 * Verify all immutable modules against genesis anchors.
 */
export function verifyAllAnchors(
  moduleContents: Map<string, string>,
  genesisAnchors: GenesisAnchors,
): {
  valid: boolean;
  results: AnchorVerification[];
  invalidCount: number;
  missingCount: number;
} {
  const results: AnchorVerification[] = [];
  let invalidCount = 0;
  let missingCount = 0;

  for (const anchor of genesisAnchors.anchors) {
    const content = moduleContents.get(anchor.path);

    if (!content) {
      results.push({
        path: anchor.path,
        expected: anchor.contentHash,
        actual: "MISSING",
        valid: false,
        error: "Module file not found",
      });
      missingCount++;
      continue;
    }

    const verification = verifyModuleAnchor(
      anchor.path,
      content,
      anchor.contentHash,
    );
    results.push(verification);

    if (!verification.valid) {
      invalidCount++;
    }
  }

  // Also verify combined hash
  const currentAnchors = results
    .filter((r) => r.valid)
    .map((r) => ({
      path: r.path,
      contentHash: r.actual,
      anchoredAt: 0n,
      version: "1.0.0",
    }));

  const currentCombinedHash = computeCombinedAnchorHash(currentAnchors);
  const combinedValid = currentCombinedHash === genesisAnchors.combinedHash;

  return {
    valid: invalidCount === 0 && missingCount === 0 && combinedValid,
    results,
    invalidCount,
    missingCount,
  };
}

// ============ Anchor Events ============

/**
 * Event emitted when anchor verification fails.
 */
export interface AnchorViolationEvent {
  type: "ANCHOR_VIOLATION";
  path: string;
  expected: string;
  actual: string;
  detectedAt: bigint;
  severity: "CRITICAL"; // Always critical for immutability violations
}

/**
 * Create anchor violation event.
 */
export function createAnchorViolation(
  verification: AnchorVerification,
  currentBlock: bigint,
): AnchorViolationEvent {
  return {
    type: "ANCHOR_VIOLATION",
    path: verification.path,
    expected: verification.expected,
    actual: verification.actual,
    detectedAt: currentBlock,
    severity: "CRITICAL",
  };
}

// ============ Anchor Serialization ============

/**
 * Serialize genesis anchors for inclusion in genesis block.
 */
export function serializeGenesisAnchors(anchors: GenesisAnchors): string {
  return JSON.stringify({
    version: anchors.version,
    timestamp: anchors.timestamp.toString(),
    anchors: anchors.anchors.map((a) => ({
      path: a.path,
      contentHash: a.contentHash,
      anchoredAt: a.anchoredAt.toString(),
      version: a.version,
    })),
    combinedHash: anchors.combinedHash,
  });
}

/**
 * Deserialize genesis anchors from genesis block.
 */
export function deserializeGenesisAnchors(serialized: string): GenesisAnchors {
  const parsed = JSON.parse(serialized);

  return {
    version: parsed.version,
    timestamp: BigInt(parsed.timestamp),
    anchors: parsed.anchors.map(
      (a: {
        path: string;
        contentHash: string;
        anchoredAt: string;
        version: string;
      }) => ({
        path: a.path,
        contentHash: a.contentHash,
        anchoredAt: BigInt(a.anchoredAt),
        version: a.version,
      }),
    ),
    combinedHash: parsed.combinedHash,
  };
}
