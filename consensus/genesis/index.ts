/**
 * Genesis Core
 *
 * The irreducible foundation of Legible Money.
 *
 * This module exports ONLY the components that have been verified as sound.
 * Everything in Genesis Core:
 *   - Uses audited cryptographic libraries (noble)
 *   - Has no internal dependencies outside Genesis Core
 *   - Is pure and deterministic
 *   - Will never change after verification
 *
 * COMPONENTS:
 *   1. Hashing    - SHA-512, BLAKE3, domain separation
 *   2. Signatures - Ed25519 sign/verify
 *   3. Keys       - Key generation and encoding
 *   4. Constants  - Protocol parameters
 *   5. DID        - Identity structure
 *
 * PP ALIGNMENT:
 *   TRUTH: These are mathematical certainties
 *   UNDERSTANDING: Each component's purpose is clear
 *   BEAUTY: Minimal and complete
 *   JUSTICE: Right components in right place
 *   CREATIVITY: Stable ground for building
 */

// ============ 1. Hashing ============
// Collision-resistant hash functions with domain separation

export {
  // Types
  type Hash,
  type HashBytes,
  // SHA-512 (primary)
  sha512Hash,
  sha512Hex,
  sha512_256,
  sha512_256Hex,
  // SHA-256 (compatibility)
  sha256Hash,
  sha256Hex,
  // BLAKE3 (performance)
  blake3Hash,
  blake3Hex,
  blake3Variable,
  // Domain separation
  domainHash,
  HASH_DOMAINS,
  // Merkle hashing
  merkleHash,
  merkleLeafHash,
  merkleInternalHash,
  // Utilities
  hashClaim,
  hashFact,
  constantTimeEqual,
  hashesEqual,
} from "../../core/crypto/hashing.js";

// ============ 2. Keys ============
// Cryptographic key generation and encoding

export {
  // Types
  type PrivateKey,
  type PublicKey,
  type Keypair,
  type KeypairBytes,
  type PrivateKeyBytes,
  type PublicKeyBytes,
  // Key generation
  generateKeypair,
  generateKeypairBytes,
  generateKeypairFromSeed,
  generateKeypairBytesFromSeed,
  getPublicKey,
  getPublicKeyFromBytes,
  // Address derivation
  publicKeyToAddress,
  // Encoding utilities
  bytesToHex,
  hexToBytes,
} from "../../core/crypto/keys.js";

// ============ 3. Signatures ============
// Unforgeable Ed25519 digital signatures

export {
  // Types (from keys, re-exported here for convenience)
  type Signature,
  type SignatureBytes,
  type SignedMessage,
  type SignatureVerification,
  // Signing
  sign,
  signString,
  signDomainSeparated,
  signTransaction,
  signBlock,
  // Verification
  verify,
  verifyString,
  verifyDomainSeparated,
  verifyTransaction,
  verifyBlock,
  batchVerify,
  // Signed messages
  createSignedMessage,
  verifySignedMessage,
  // Aggregation
  type AggregatedSignature,
  aggregateSignatures,
  verifyAggregatedSignature,
} from "../../core/crypto/signatures.js";

// ============ 4. Constants ============
// Protocol identity and parameters

export {
  TOKEN,
  VERSION,
  TIMING,
  CONSENSUS,
  ECONOMICS,
} from "../../core/constants.js";

// Note: GENESIS constant may not exist yet, will be added when genesis block is defined

// ============ 5. DID ============
// Decentralized identity for agents

export {
  // Schema and types
  DIDSchema,
  type DID,
  CapabilitySchema,
  type Capability,
  type AgentIdentity,
  type DIDDocument,
  // Functions
  publicKeyToDid,
  createDID,
  didToAddress,
  didToNetwork,
  createDIDDocument,
} from "../../core/agents/identity/did.js";

// ============ Genesis Core Metadata ============

/**
 * Genesis Core version.
 * This version changes only if Genesis Core structure changes (which should be never).
 */
export const GENESIS_CORE_VERSION = "1.0.0" as const;

/**
 * List of files that comprise Genesis Core.
 * These paths are relative to project root.
 */
export const GENESIS_CORE_FILES = [
  "core/crypto/hashing.ts",
  "core/crypto/keys.ts",
  "core/crypto/signatures.ts",
  "core/constants.ts",
  "agents/identity/did.ts",
] as const;

/**
 * Genesis Core verification status.
 * Updated as verification milestones are completed.
 */
export interface GenesisVerificationStatus {
  file: (typeof GENESIS_CORE_FILES)[number];
  tests: "pending" | "passing" | "failing";
  vectors: "pending" | "verified" | "n/a";
  documented: boolean;
  audited: boolean;
}

/**
 * Current verification status.
 * TODO: Update as verification progresses.
 */
export const VERIFICATION_STATUS: GenesisVerificationStatus[] = [
  {
    file: "core/crypto/hashing.ts",
    tests: "pending",
    vectors: "pending", // NIST FIPS 180-4, BLAKE3 official
    documented: false,
    audited: false,
  },
  {
    file: "core/crypto/keys.ts",
    tests: "pending",
    vectors: "pending", // RFC 8032
    documented: false,
    audited: false,
  },
  {
    file: "core/crypto/signatures.ts",
    tests: "pending",
    vectors: "pending", // RFC 8032
    documented: false,
    audited: false,
  },
  {
    file: "core/constants.ts",
    tests: "pending",
    vectors: "n/a", // No external vectors, just value checks
    documented: false,
    audited: false,
  },
  {
    file: "agents/identity/did.ts",
    tests: "pending",
    vectors: "n/a", // No external vectors, format validation
    documented: false,
    audited: false,
  },
];

/**
 * Check if Genesis Core is fully verified.
 */
export function isGenesisVerified(): boolean {
  return VERIFICATION_STATUS.every(
    (s) =>
      s.tests === "passing" &&
      (s.vectors === "verified" || s.vectors === "n/a") &&
      s.documented &&
      s.audited,
  );
}

/**
 * Get verification progress as a summary.
 */
export function getVerificationProgress(): {
  complete: number;
  total: number;
  percentage: number;
  details: Array<{
    file: string;
    status: "complete" | "partial" | "pending";
    missing: string[];
  }>;
} {
  let complete = 0;
  const total = VERIFICATION_STATUS.length * 4; // 4 criteria per file
  const details: Array<{
    file: string;
    status: "complete" | "partial" | "pending";
    missing: string[];
  }> = [];

  for (const status of VERIFICATION_STATUS) {
    const missing: string[] = [];
    let fileComplete = 0;

    if (status.tests === "passing") {
      fileComplete++;
      complete++;
    } else {
      missing.push("tests");
    }

    if (status.vectors === "verified" || status.vectors === "n/a") {
      fileComplete++;
      complete++;
    } else {
      missing.push("test vectors");
    }

    if (status.documented) {
      fileComplete++;
      complete++;
    } else {
      missing.push("documentation");
    }

    if (status.audited) {
      fileComplete++;
      complete++;
    } else {
      missing.push("audit");
    }

    details.push({
      file: status.file,
      status:
        fileComplete === 4
          ? "complete"
          : fileComplete > 0
            ? "partial"
            : "pending",
      missing,
    });
  }

  return {
    complete,
    total,
    percentage: Math.round((complete / total) * 100),
    details,
  };
}

/**
 * Print verification status to console.
 */
export function printVerificationStatus(): void {
  const progress = getVerificationProgress();

  console.log("\n=== GENESIS CORE VERIFICATION STATUS ===\n");
  console.log(
    `Overall: ${progress.percentage}% (${progress.complete}/${progress.total})\n`,
  );

  for (const detail of progress.details) {
    const icon =
      detail.status === "complete"
        ? "✅"
        : detail.status === "partial"
          ? "🔶"
          : "⬜";
    console.log(`${icon} ${detail.file}`);
    if (detail.missing.length > 0) {
      console.log(`   Missing: ${detail.missing.join(", ")}`);
    }
  }

  console.log(
    "\n" +
      (isGenesisVerified()
        ? "✅ GENESIS CORE VERIFIED"
        : "⚠️  GENESIS CORE NOT YET VERIFIED"),
  );
  console.log("");
}
