/**
 * Foundation Utilities
 *
 * Pure utility functions with no side effects.
 * These are the building blocks used throughout Legible Money.
 */

// ============ Byte Operations ============

/**
 * Convert bytes to hex string
 */
export function bytesToHex(bytes: Uint8Array): string {
  return Array.from(bytes)
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
}

/**
 * Convert hex string to bytes
 */
export function hexToBytes(hex: string): Uint8Array {
  const bytes = new Uint8Array(hex.length / 2);
  for (let i = 0; i < hex.length; i += 2) {
    bytes[i / 2] = parseInt(hex.slice(i, i + 2), 16);
  }
  return bytes;
}

/**
 * Concatenate multiple Uint8Arrays
 */
export function concatBytes(...arrays: Uint8Array[]): Uint8Array {
  const totalLength = arrays.reduce((sum, arr) => sum + arr.length, 0);
  const result = new Uint8Array(totalLength);
  let offset = 0;
  for (const arr of arrays) {
    result.set(arr, offset);
    offset += arr.length;
  }
  return result;
}

/**
 * Compare two Uint8Arrays for equality
 */
export function bytesEqual(a: Uint8Array, b: Uint8Array): boolean {
  if (a.length !== b.length) return false;
  for (let i = 0; i < a.length; i++) {
    if (a[i] !== b[i]) return false;
  }
  return true;
}

/**
 * Constant-time comparison to prevent timing attacks
 */
export function constantTimeEqual(a: Uint8Array, b: Uint8Array): boolean {
  if (a.length !== b.length) return false;
  let result = 0;
  for (let i = 0; i < a.length; i++) {
    result |= a[i] ^ b[i];
  }
  return result === 0;
}

// ============ BigInt Operations ============

/**
 * Minimum of two bigints
 */
export function bigintMin(a: bigint, b: bigint): bigint {
  return a < b ? a : b;
}

/**
 * Maximum of two bigints
 */
export function bigintMax(a: bigint, b: bigint): bigint {
  return a > b ? a : b;
}

/**
 * Clamp bigint to range
 */
export function bigintClamp(value: bigint, min: bigint, max: bigint): bigint {
  return bigintMax(min, bigintMin(value, max));
}

/**
 * Safe division with bigint (returns floor)
 */
export function bigintDiv(a: bigint, b: bigint): bigint {
  if (b === 0n) throw new Error("Division by zero");
  return a / b;
}

/**
 * Percentage of bigint (with precision)
 */
export function bigintPercent(
  value: bigint,
  percent: number,
  precision = 10000,
): bigint {
  const scaledPercent = BigInt(Math.round(percent * precision));
  return (value * scaledPercent) / BigInt(precision);
}

// ============ String Operations ============

/**
 * Normalize whitespace (trim and collapse)
 */
export function normalizeWhitespace(str: string): string {
  return str.trim().replace(/\s+/g, " ");
}

/**
 * Truncate string with ellipsis
 */
export function truncate(
  str: string,
  maxLength: number,
  suffix = "...",
): string {
  if (str.length <= maxLength) return str;
  return str.slice(0, maxLength - suffix.length) + suffix;
}

/**
 * Truncate hex string showing start and end
 */
export function truncateHex(hex: string, chars = 8): string {
  if (hex.length <= chars * 2 + 2) return hex;
  return `${hex.slice(0, chars)}...${hex.slice(-chars)}`;
}

// ============ Array Operations ============

/**
 * Chunk array into smaller arrays
 */
export function chunk<T>(arr: T[], size: number): T[][] {
  const chunks: T[][] = [];
  for (let i = 0; i < arr.length; i += size) {
    chunks.push(arr.slice(i, i + size));
  }
  return chunks;
}

/**
 * Remove duplicates from array
 */
export function unique<T>(arr: T[]): T[] {
  return [...new Set(arr)];
}

/**
 * Group array items by key
 */
export function groupBy<T, K extends string | number>(
  arr: T[],
  keyFn: (item: T) => K,
): Record<K, T[]> {
  return arr.reduce(
    (acc, item) => {
      const key = keyFn(item);
      if (!acc[key]) acc[key] = [];
      acc[key].push(item);
      return acc;
    },
    {} as Record<K, T[]>,
  );
}

// ============ Object Operations ============

/**
 * Deep freeze an object (make immutable)
 */
export function deepFreeze<T extends object>(obj: T): Readonly<T> {
  Object.keys(obj).forEach((key) => {
    const value = (obj as Record<string, unknown>)[key];
    if (value && typeof value === "object") {
      deepFreeze(value as object);
    }
  });
  return Object.freeze(obj);
}

/**
 * Pick specific keys from object
 */
export function pick<T extends object, K extends keyof T>(
  obj: T,
  keys: K[],
): Pick<T, K> {
  const result = {} as Pick<T, K>;
  for (const key of keys) {
    if (key in obj) {
      result[key] = obj[key];
    }
  }
  return result;
}

/**
 * Omit specific keys from object
 */
export function omit<T extends object, K extends keyof T>(
  obj: T,
  keys: K[],
): Omit<T, K> {
  const result = { ...obj };
  for (const key of keys) {
    delete (result as Record<string, unknown>)[key as string];
  }
  return result as Omit<T, K>;
}

// ============ Time Operations ============

/**
 * Sleep for specified milliseconds
 */
export function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * Get current timestamp in milliseconds
 */
export function now(): number {
  return Date.now();
}

/**
 * Calculate blocks from milliseconds
 */
export function msToBlocks(ms: number, blockTimeMs: number): bigint {
  return BigInt(Math.floor(ms / blockTimeMs));
}

/**
 * Calculate milliseconds from blocks
 */
export function blocksToMs(blocks: bigint, blockTimeMs: number): number {
  return Number(blocks) * blockTimeMs;
}

// ============ Validation ============

/**
 * Assert condition or throw
 */
export function assert(condition: boolean, message: string): asserts condition {
  if (!condition) {
    throw new Error(message);
  }
}

/**
 * Assert value is not null/undefined
 */
export function assertDefined<T>(
  value: T | null | undefined,
  name: string,
): asserts value is T {
  if (value === null || value === undefined) {
    throw new Error(`${name} is required but was ${value}`);
  }
}

// ============ Serialization ============

/**
 * Deterministic JSON stringify (sorted keys, BigInt safe)
 */
export function deterministicStringify(obj: unknown): string {
  return JSON.stringify(obj, (_, value) => {
    // Handle BigInt first (cannot be serialized by JSON.stringify)
    if (typeof value === "bigint") {
      return value.toString();
    }
    // Sort object keys for determinism
    if (value && typeof value === "object" && !Array.isArray(value)) {
      return Object.keys(value)
        .sort()
        .reduce(
          (sorted, key) => {
            sorted[key] = (value as Record<string, unknown>)[key];
            return sorted;
          },
          {} as Record<string, unknown>,
        );
    }
    return value;
  });
}

/**
 * Convert bigint to string for JSON
 */
export function bigintReplacer(_: string, value: unknown): unknown {
  if (typeof value === "bigint") {
    return value.toString();
  }
  return value;
}

/**
 * Parse JSON with bigint support
 */
export function bigintReviver(_: string, value: unknown): unknown {
  // Heuristic: if it's a string of all digits, try to parse as bigint
  if (typeof value === "string" && /^\d+$/.test(value) && value.length > 15) {
    return BigInt(value);
  }
  return value;
}
