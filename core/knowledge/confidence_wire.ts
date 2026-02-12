/**
 * Confidence Wire Format Converters
 *
 * Converts between TS float confidence (0.0-1.0) and Go wire format (0-1,000,000).
 * Used at serialization boundaries when communicating with the Cosmos SDK chain.
 *
 * Go uses uint64 scaled to 1,000,000 for fixed-point confidence.
 * TS uses float64 [0.0, 1.0] internally.
 * These converters bridge the two representations without changing internal code.
 */

export const CONFIDENCE_SCALE = 1_000_000;

/** TS float (0.0-1.0) → Go wire format (0-1,000,000) */
export function confidenceToWire(c: number): number {
  return Math.round(Math.max(0, Math.min(1, c)) * CONFIDENCE_SCALE);
}

/** Go wire format (0-1,000,000) → TS float (0.0-1.0) */
export function wireToConfidence(w: number): number {
  return w / CONFIDENCE_SCALE;
}
