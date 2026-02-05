/**
 * Genesis Treasury Derivation
 *
 * Deterministically derives treasury addresses from genesis block.
 * Anyone can verify these addresses are correctly derived.
 *
 * GOVERNANCE MODEL:
 *   Phase 1 (Launch): Centralized - founder holds keys, published spending
 *   Phase 2 (Future): Anonymous election via ZK proofs
 *
 * WHY CENTRALIZED INITIALLY:
 *   - Doxxing multisig signers creates social engineering attack surface
 *   - Anonymous election requires ZK infrastructure (not ready at launch)
 *   - Founder accountability with timelock transparency is safer than
 *     doxxed multisig with known attack targets
 */

import { sha512Hash } from "../../core/crypto/hashing.js";
import {
  generateKeypairFromSeed,
  publicKeyToAddress,
  hexToBytes,
  bytesToHex,
} from "../../core/crypto/keys.js";
import { GENESIS, TREASURY } from "../../core/constants.js";

// ============ Derivation Constants ============

/**
 * Domain separation for treasury key derivation
 */
const TREASURY_DOMAINS = {
  RESEARCH: "LGM.treasury.research.v1",
  FOUNDATION: "LGM.treasury.foundation.v1",
  COMMUNITY: "LGM.treasury.community.v1",
  DEVELOPERS: "LGM.treasury.developers.v1",
  RESERVE: "LGM.treasury.reserve.v1",
} as const;

// ============ Derivation Functions ============

/**
 * Derive a deterministic seed for a treasury fund
 *
 * seed = SHA-512(genesis_block_hash || domain)[:32]
 */
export function deriveTreasurySeed(domain: string): Uint8Array {
  const genesisBytes = hexToBytes(GENESIS.BLOCK_HASH.slice(2)); // Remove 0x prefix
  const domainBytes = new TextEncoder().encode(domain);

  // Combine genesis hash with domain
  const combined = new Uint8Array(genesisBytes.length + domainBytes.length);
  combined.set(genesisBytes);
  combined.set(domainBytes, genesisBytes.length);

  // Hash and take first 32 bytes as seed
  const hash = sha512Hash(combined);
  return hash.slice(0, 32);
}

/**
 * Derive treasury address from genesis
 *
 * This is deterministic - anyone can verify the address is correct
 * by running this derivation themselves.
 */
export function deriveTreasuryAddress(
  fund: keyof typeof TREASURY_DOMAINS,
): string {
  const domain = TREASURY_DOMAINS[fund];
  const seed = deriveTreasurySeed(domain);
  const keypair = generateKeypairFromSeed(seed);
  return publicKeyToAddress(keypair.publicKey);
}

/**
 * Derive all treasury addresses
 */
export function deriveAllTreasuryAddresses(): Record<string, string> {
  return {
    RESEARCH: deriveTreasuryAddress("RESEARCH"),
    FOUNDATION: deriveTreasuryAddress("FOUNDATION"),
    COMMUNITY: deriveTreasuryAddress("COMMUNITY"),
    DEVELOPERS: deriveTreasuryAddress("DEVELOPERS"),
    RESERVE: deriveTreasuryAddress("RESERVE"),
  };
}

/**
 * Verify that treasury addresses match expected derivation
 */
export function verifyTreasuryAddresses(addresses: Record<string, string>): {
  valid: boolean;
  mismatches: string[];
} {
  const derived = deriveAllTreasuryAddresses();
  const mismatches: string[] = [];

  for (const [fund, expected] of Object.entries(addresses)) {
    const actual = derived[fund];
    if (actual !== expected) {
      mismatches.push(`${fund}: expected ${expected}, got ${actual}`);
    }
  }

  return {
    valid: mismatches.length === 0,
    mismatches,
  };
}

// ============ Governance Transition ============

/**
 * Future: Anonymous Election System
 *
 * When ZK infrastructure is ready, transition to anonymous governance:
 *
 * 1. ELIGIBILITY (ZK proven, not revealed):
 *    - Holds >= X LGM for >= Y blocks
 *    - Not slashed in past Z blocks
 *    - Participated in >= N verifications
 *
 * 2. ELECTION:
 *    - Candidates submit ZK proof of eligibility
 *    - Token holders vote (encrypted until tally)
 *    - Winners revealed, identities stay private
 *
 * 3. SIGNING:
 *    - Threshold signatures (t-of-n)
 *    - Signers never meet or know each other
 *    - All coordination via encrypted channels
 *
 * This eliminates social engineering by eliminating knowledge of who to target.
 */
export interface AnonymousGovernanceConfig {
  // Eligibility requirements (proven via ZK)
  minStake: bigint;
  minStakeDurationBlocks: number;
  minVerifications: number;
  noSlashingPeriodBlocks: number;

  // Election parameters
  electionDurationBlocks: number;
  candidateCount: number;
  winnerCount: number;

  // Threshold signature
  threshold: number;
  totalSigners: number;
}

/**
 * Placeholder config for future anonymous governance
 */
export const FUTURE_ANONYMOUS_GOVERNANCE: AnonymousGovernanceConfig = {
  minStake: 1000n * 10n ** 18n, // 1000 LGM
  minStakeDurationBlocks: 100_000, // ~3 days
  minVerifications: 10, // Participated in protocol
  noSlashingPeriodBlocks: 500_000, // ~15 days clean record

  electionDurationBlocks: 50_000, // ~1.5 days voting
  candidateCount: 20, // Max candidates
  winnerCount: 7, // Elected signers

  threshold: 4, // 4-of-7
  totalSigners: 7,
};

// ============ Utility ============

/**
 * Print treasury derivation for verification
 */
export function printTreasuryDerivation(): void {
  console.log("=== TREASURY ADDRESS DERIVATION ===\n");
  console.log(`Genesis Block Hash: ${GENESIS.BLOCK_HASH}\n`);
  console.log(`Governance Phase: ${TREASURY.GOVERNANCE.phase}\n`);

  for (const fund of [
    "RESEARCH",
    "FOUNDATION",
    "COMMUNITY",
    "DEVELOPERS",
    "RESERVE",
  ] as const) {
    const domain = TREASURY_DOMAINS[fund];
    const seed = deriveTreasurySeed(domain);
    const address = deriveTreasuryAddress(fund);

    console.log(`${fund}:`);
    console.log(`  Domain: ${domain}`);
    console.log(`  Seed:   ${bytesToHex(seed)}`);
    console.log(`  Address: ${address}`);
    console.log(`  Allocation: ${TREASURY.ALLOCATION[fund] * 100}%`);
    console.log();
  }

  console.log(
    `Operation Timelock: ${TREASURY.OPERATION_TIMELOCK_BLOCKS} blocks (~42 min)`,
  );
  console.log("\nAll operations are public and delayed for transparency.");
}
