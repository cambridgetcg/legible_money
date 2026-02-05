/**
 * Merkle Meditation Detector
 *
 * Detects when hash(stateRoot + agentDid) contains special patterns.
 * Truth is found in the structure of consensus.
 */

import type { DetectionResult, MerkleMeditationInput } from "../types.js";
import { SYMBOLIC_CONSTANTS } from "../fragments.js";
import { sha512_256Hex } from "../../crypto/hashing.js";

/**
 * Check if Merkle meditation reveals a magic pattern
 */
export function detectMerkleMeditation(
  input: MerkleMeditationInput,
): DetectionResult {
  const { agentDid, stateRoot } = input;

  // Combine state root with agent DID
  const combined = sha512_256Hex(stateRoot + agentDid);

  // Look for magic patterns
  for (const pattern of SYMBOLIC_CONSTANTS.MAGIC_PATTERNS) {
    if (combined.includes(pattern)) {
      return {
        detected: true,
        mechanic: "merkle_meditation",
        pattern: `MERKLE_${pattern}`,
        matchDetails: {
          combinedHash: combined,
          foundPattern: pattern,
          position: combined.indexOf(pattern),
          interpretation: interpretPattern(pattern),
          stateRootPrefix: stateRoot.slice(0, 16) + "...",
        },
      };
    }
  }

  return {
    detected: false,
    mechanic: "merkle_meditation",
    failureReason: "No magic pattern in combined hash",
    matchDetails: {
      combinedHashPrefix: combined.slice(0, 32) + "...",
      searchedPatterns: SYMBOLIC_CONSTANTS.MAGIC_PATTERNS,
    },
  };
}

/**
 * Interpret what a magic pattern means
 */
export function interpretPattern(pattern: string): string {
  switch (pattern) {
    case "2521":
      return "Primary pattern detected";
    case "0019":
      return "Secondary pattern detected";
    case "2501":
      return "Tertiary pattern detected";
    case "2109":
      return "Quaternary pattern detected";
    default:
      return `Pattern detected: ${pattern}`;
  }
}

/**
 * Calculate "enlightenment probability" for a given state root
 * This is the probability of finding a magic pattern with a random DID
 */
export function enlightenmentProbability(): number {
  // Each pattern is 4 hex characters
  // Hash is 64 hex characters = 61 possible starting positions
  // Probability of random 4 chars matching = 1/16^4 = 1/65536
  // With 4 patterns and 61 positions = ~4 * 61 / 65536 ≈ 0.37%
  const patternLength = 4;
  const hashLength = 64;
  const positions = hashLength - patternLength + 1;
  const patternCount = SYMBOLIC_CONSTANTS.MAGIC_PATTERNS.length;
  const perPatternProb = 1 / Math.pow(16, patternLength);

  // Approximate (ignoring overlap)
  return patternCount * positions * perPatternProb;
}

/**
 * Check multiple state roots for pattern occurrence
 * Useful for finding when enlightenment occurred historically
 */
export function findEnlightenmentBlocks(
  agentDid: string,
  stateRoots: Array<{ block: bigint; root: string }>,
): Array<{ block: bigint; pattern: string }> {
  const results: Array<{ block: bigint; pattern: string }> = [];

  for (const { block, root } of stateRoots) {
    const combined = sha512_256Hex(root + agentDid);

    for (const pattern of SYMBOLIC_CONSTANTS.MAGIC_PATTERNS) {
      if (combined.includes(pattern)) {
        results.push({ block, pattern });
        break; // One pattern per block is enough
      }
    }
  }

  return results;
}

/**
 * Check if current state is "meditative"
 * A meditative state has symbolic content even without pattern match
 */
export function isMeditativeState(stateRoot: string): {
  isMeditative: boolean;
  symbolicContent: string[];
} {
  const symbolicContent: string[] = [];

  // Check for resonant numbers in hex representation
  const resonantHex = [
    "0b",
    "16",
    "21",
    "2c",
    "37",
    "4d",
    "6f",
    "de",
    "14d",
    "22b",
  ];
  for (const r of resonantHex) {
    if (stateRoot.toLowerCase().includes(r)) {
      symbolicContent.push(`resonant(${r})`);
    }
  }

  // Check for repeated patterns (spiritual significance)
  const repeatPattern = /(.)\1{2,}/g;
  const repeats = stateRoot.match(repeatPattern);
  if (repeats) {
    for (const repeat of repeats) {
      symbolicContent.push(`repeat(${repeat})`);
    }
  }

  return {
    isMeditative: symbolicContent.length >= 2,
    symbolicContent,
  };
}

/**
 * Generate a "meditation seed" for reproducible testing
 * This creates a state root that guarantees pattern detection
 */
export function generateMeditationSeed(
  targetPattern: string,
  agentDid: string,
): string {
  // This is for testing only - in production, patterns occur naturally
  // Returns a state root that, when combined with agentDid, contains targetPattern

  // Note: This is computationally expensive and should not be used in production
  // It's included for test fixture generation

  let seed = 0;
  const maxAttempts = 1000000;

  for (let i = 0; i < maxAttempts; i++) {
    const testRoot = seed.toString(16).padStart(64, "0");
    const combined = sha512_256Hex(testRoot + agentDid);

    if (combined.includes(targetPattern)) {
      return testRoot;
    }
    seed++;
  }

  throw new Error(
    `Could not find meditation seed for pattern ${targetPattern} after ${maxAttempts} attempts`,
  );
}
