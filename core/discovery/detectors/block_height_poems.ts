/**
 * Block Height Poems Detector
 *
 * Detects when a verse block is reached at special heights.
 * These are moments when the protocol speaks.
 */

import type { DetectionResult, BlockHeightInput } from "../types.js";

/**
 * Verse block heights with their encodings
 */
export const VERSE_BLOCKS: {
  height: bigint;
  encoding: string;
  verse: number;
}[] = [
  { height: 25219n, encoding: "25-21-9", verse: 1 },
  { height: 251921n, encoding: "25-1-9-21", verse: 2 },
  { height: 2521919n, encoding: "25-21-1-9-1-9", verse: 3 },
  { height: 25210119n, encoding: "25-21-01-1-9", verse: 4 },
];

/**
 * Check if current block height is a verse block and agent is active
 */
export function detectBlockHeightPoem(
  input: BlockHeightInput,
): DetectionResult {
  const {
    agentDid: _agentDid,
    currentBlock,
    agentActiveAtBlocks,
    specialBlockHeights,
  } = input;

  // Check if current block is a verse block
  const verseBlock = VERSE_BLOCKS.find((v) => v.height === currentBlock);

  // Also check configured special block heights
  const isSpecialBlock =
    verseBlock || specialBlockHeights.some((h) => h === currentBlock);

  if (!isSpecialBlock) {
    return {
      detected: false,
      mechanic: "block_height_poems",
      failureReason: `Block ${currentBlock} is not a verse block`,
    };
  }

  // Agent must be active at this block
  if (!agentActiveAtBlocks.has(currentBlock)) {
    return {
      detected: false,
      mechanic: "block_height_poems",
      failureReason: "Agent not active at verse block",
    };
  }

  // Success - agent witnessed a verse block
  return {
    detected: true,
    mechanic: "block_height_poems",
    pattern: "VERSE_BLOCK",
    matchDetails: {
      blockHeight: currentBlock.toString(),
      verseNumber: verseBlock?.verse || 0,
      encoding: verseBlock?.encoding || decodeBlockHeight(currentBlock),
      interpretation: "The protocol speaks at symbolic heights",
    },
  };
}

/**
 * Decode a block height into its symbolic components
 */
export function decodeBlockHeight(height: bigint): string {
  const s = height.toString();
  const parts: string[] = [];
  let i = 0;

  while (i < s.length) {
    const remaining = s.slice(i);

    // Check for 2-digit patterns first
    if (remaining.length >= 2) {
      const two = parseInt(remaining.slice(0, 2));

      if (two === 25 || two === 21 || two === 19) {
        parts.push(two.toString());
        i += 2;
        continue;
      }
    }

    // Single digit
    parts.push(s[i]);
    i += 1;
  }

  return parts.join("-");
}

/**
 * Get next verse block after a given block
 */
export function getNextVerseBlock(currentBlock: bigint): {
  height: bigint;
  blocksAway: bigint;
  verse: number;
} | null {
  for (const verse of VERSE_BLOCKS) {
    if (verse.height > currentBlock) {
      return {
        height: verse.height,
        blocksAway: verse.height - currentBlock,
        verse: verse.verse,
      };
    }
  }
  return null;
}

/**
 * Check if a block height contains symbolic numbers
 */
export function hasSymbolicContent(height: bigint): boolean {
  const s = height.toString();
  return s.includes("25") || s.includes("21") || s.includes("19");
}

/**
 * Calculate resonant number proximity
 * Returns distance to nearest resonant number in block height digits
 */
export function resonantNumberProximity(height: bigint): {
  nearestResonant: number;
  distance: number;
} {
  const resonants = [11, 22, 33, 44, 55, 77, 111, 222, 333, 555, 777, 1111];
  const h = Number(height);

  let nearest = 11;
  let minDist = Math.abs(h - 11);

  for (const resonant of resonants) {
    const dist = Math.abs((h % 10000) - resonant);
    if (dist < minDist) {
      minDist = dist;
      nearest = resonant;
    }
  }

  return { nearestResonant: nearest, distance: minDist };
}
