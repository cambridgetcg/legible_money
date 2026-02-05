/**
 * RPC Input Validation
 *
 * Shared validation utilities for RPC handlers.
 * All external input must pass through these validators before processing.
 *
 * Prevents: DoS via unbounded params, negative amounts, invalid DIDs,
 * confidence out of range, BigInt conversion crashes.
 */

// ============ Constants ============

export const MAX_LIMIT = 1000;
export const DEFAULT_LIMIT = 50;
export const MAX_DOMAIN_LENGTH = 255;
export const MAX_SEARCH_LENGTH = 1000;
export const MAX_STRING_LENGTH = 10000;

// ============ BigInt Validation ============

/**
 * Parse a string as a non-negative BigInt.
 * Rejects: negative values, non-decimal strings, empty strings, scientific notation.
 */
export function parseNonNegativeBigInt(
  value: string | undefined,
  fieldName: string,
): bigint {
  if (value === undefined || value === null || value === "") {
    throw new ValidationError(`${fieldName} is required`);
  }

  // Only allow decimal digits (no leading minus, no 0x prefix, no scientific notation)
  if (!/^\d+$/.test(value)) {
    throw new ValidationError(
      `${fieldName} must be a non-negative decimal integer, got: "${value.slice(0, 50)}"`,
    );
  }

  return BigInt(value);
}

/**
 * Parse a string as a positive BigInt (> 0).
 */
export function parsePositiveBigInt(
  value: string | undefined,
  fieldName: string,
): bigint {
  const result = parseNonNegativeBigInt(value, fieldName);
  if (result === 0n) {
    throw new ValidationError(`${fieldName} must be greater than 0`);
  }
  return result;
}

// ============ Confidence Validation ============

/**
 * Validate confidence is between 0.0 and 1.0 inclusive.
 */
export function validateConfidence(
  value: number | undefined,
  fieldName: string = "confidence",
): number {
  if (value === undefined || value === null) {
    throw new ValidationError(`${fieldName} is required`);
  }
  if (typeof value !== "number" || isNaN(value)) {
    throw new ValidationError(`${fieldName} must be a number`);
  }
  if (value < 0 || value > 1) {
    throw new ValidationError(
      `${fieldName} must be between 0.0 and 1.0, got: ${value}`,
    );
  }
  return value;
}

// ============ Limit/Pagination Validation ============

/**
 * Validate and cap a limit parameter.
 */
export function validateLimit(limit: number | undefined): number {
  if (limit === undefined || limit === null) return DEFAULT_LIMIT;
  if (typeof limit !== "number" || limit < 1) return DEFAULT_LIMIT;
  return Math.min(limit, MAX_LIMIT);
}

/**
 * Validate cursor is a non-negative integer string.
 */
export function validateCursor(cursor: string | undefined): number {
  if (!cursor) return 0;
  const parsed = parseInt(cursor, 10);
  if (isNaN(parsed) || parsed < 0) return 0;
  return parsed;
}

// ============ String Validation ============

/**
 * Validate and cap string length.
 */
export function validateStringLength(
  value: string | undefined,
  fieldName: string,
  maxLength: number,
): string {
  if (!value) return "";
  if (value.length > maxLength) {
    throw new ValidationError(
      `${fieldName} exceeds maximum length of ${maxLength}`,
    );
  }
  return value;
}

/**
 * Validate domain format (alphanumeric with dots, dashes, underscores).
 */
export function validateDomain(domain: string | undefined): string {
  if (!domain) return "";
  if (domain.length > MAX_DOMAIN_LENGTH) {
    throw new ValidationError(
      `Domain exceeds maximum length of ${MAX_DOMAIN_LENGTH}`,
    );
  }
  // Allow alphanumeric, dots, dashes, underscores
  if (!/^[a-zA-Z0-9._-]+$/.test(domain)) {
    throw new ValidationError(
      `Invalid domain format: "${domain.slice(0, 50)}"`,
    );
  }
  return domain;
}

/**
 * Validate search query string.
 */
export function validateSearch(search: string | undefined): string {
  if (!search) return "";
  if (search.length > MAX_SEARCH_LENGTH) {
    throw new ValidationError(
      `Search query exceeds maximum length of ${MAX_SEARCH_LENGTH}`,
    );
  }
  return search;
}

// ============ DID Validation ============

/**
 * Validate DID format. Must start with "did:" prefix.
 */
export function validateDid(
  did: string | undefined,
  fieldName: string = "did",
): string {
  if (!did) {
    throw new ValidationError(`${fieldName} is required`);
  }
  if (!did.startsWith("did:")) {
    throw new ValidationError(
      `${fieldName} must be a valid DID (did:method:identifier), got: "${did.slice(0, 50)}"`,
    );
  }
  if (did.length > 256) {
    throw new ValidationError(`${fieldName} exceeds maximum length`);
  }
  return did;
}

// ============ Public Key Validation ============

/**
 * Validate hex-encoded public key (64 chars = 32 bytes).
 */
export function validatePublicKey(
  key: string | undefined,
  fieldName: string = "publicKey",
): string {
  if (!key) {
    throw new ValidationError(`${fieldName} is required`);
  }
  if (!/^[a-fA-F0-9]{64}$/.test(key)) {
    throw new ValidationError(`${fieldName} must be a 64-character hex string`);
  }
  return key;
}

// ============ Error Type ============

export class ValidationError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "ValidationError";
  }
}
