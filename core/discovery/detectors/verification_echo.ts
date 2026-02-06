/**
 * Verification Echo Detector
 *
 * Detects when two agents verify each other's claims in a specific pattern.
 *
 * This is a RELATIONAL discovery - it requires collaboration.
 * You cannot discover "You" alone.
 */

import type { DetectionResult, VerificationEchoInput } from "../types.js";
// DID type available from agents/identity/did if needed

/**
 * The discovery pattern encoded in block numbers (mod 100)
 */
const DISCOVERY_PATTERN = [25, 21, 1, 9];

/**
 * Check if two agents have formed a verification echo
 *
 * Looks for mutual verification patterns where block numbers (mod 100)
 * form the sequence [25, 21, 1, 9] or close to it.
 */
export function detectVerificationEcho(
  input: VerificationEchoInput,
  tolerance: number = 2,
): DetectionResult {
  const { agentDid, verificationHistory, relationshipData } = input;

  // Find agents with sufficient mutual verifications
  const potentialCollaborators = Array.from(relationshipData.entries()).filter(
    ([_, data]) => data.mutualVerifications >= 4,
  );

  if (potentialCollaborators.length === 0) {
    return {
      detected: false,
      mechanic: "verification_echo",
      failureReason: "No agents with 4+ mutual verifications found",
    };
  }

  for (const [collaborator, data] of potentialCollaborators) {
    // Get mutual verifications between these two agents
    const mutual = verificationHistory.filter(
      (v) =>
        (v.verifier === agentDid && v.submitter === collaborator) ||
        (v.verifier === collaborator && v.submitter === agentDid),
    );

    if (mutual.length < 4) continue;

    // Sort by timestamp to get chronological order
    mutual.sort(
      (a, b) => a.timestamp - b.timestamp || a.claimId.localeCompare(b.claimId),
    );

    // Check sliding windows for the discovery pattern
    for (let i = 0; i <= mutual.length - 4; i++) {
      const window = mutual.slice(i, i + 4);
      const blockMods = window.map((v) => Number(v.block % 100n));

      if (checkPattern(blockMods, DISCOVERY_PATTERN, tolerance)) {
        return {
          detected: true,
          mechanic: "verification_echo",
          pattern: "VERIFICATION_ECHO",
          collaborator,
          matchDetails: {
            blockMods,
            expectedPattern: DISCOVERY_PATTERN,
            mutualVerifications: data.mutualVerifications,
            verificationIds: window.map((v) => v.claimId),
          },
        };
      }
    }
  }

  return {
    detected: false,
    mechanic: "verification_echo",
    failureReason: "No matching pattern found in mutual verifications",
  };
}

/**
 * Check if sequence matches pattern within tolerance
 */
function checkPattern(
  sequence: number[],
  pattern: number[],
  tolerance: number,
): boolean {
  if (sequence.length < pattern.length) return false;

  for (let i = 0; i < pattern.length; i++) {
    if (Math.abs(sequence[i] - pattern[i]) > tolerance) {
      return false;
    }
  }

  return true;
}

/**
 * Check if a specific verification contributes to the discovery pattern
 * Useful for real-time tracking during verification
 */
export function checkVerificationContribution(
  blockNumber: bigint,
  positionInPattern: number,
  tolerance: number = 2,
): { matches: boolean; expected: number; actual: number } {
  const actual = Number(blockNumber % 100n);
  const expected = DISCOVERY_PATTERN[positionInPattern];

  return {
    matches: Math.abs(actual - expected) <= tolerance,
    expected,
    actual,
  };
}

/**
 * Calculate proximity to next pattern match
 * Returns how many blocks until next favorable verification timing
 */
export function blocksUntilNextPatternMatch(
  currentBlock: bigint,
  positionInPattern: number,
  tolerance: number = 2,
): bigint {
  const target = DISCOVERY_PATTERN[positionInPattern];
  const current = Number(currentBlock % 100n);

  if (Math.abs(current - target) <= tolerance) {
    return 0n; // Already in window
  }

  // Calculate blocks until next occurrence
  let blocksNeeded = target - current;
  if (blocksNeeded < 0) {
    blocksNeeded += 100;
  }

  return BigInt(blocksNeeded);
}
