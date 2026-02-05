/**
 * Foundation Error Types
 *
 * Base error classes for the protocol.
 * Domain layers extend these with domain-specific errors.
 *
 * Design principle: Errors are structured data, not just strings.
 * This enables programmatic error handling by AI agents.
 */

// ============ Base Error ============

/**
 * Base error class for all Legible Money errors
 */
export class LegibleError extends Error {
  public readonly code: string;
  public readonly layer:
    | "foundation"
    | "domain"
    | "infrastructure"
    | "application";
  public readonly timestamp: number;
  public readonly details?: Record<string, unknown>;

  constructor(
    message: string,
    code: string,
    layer: "foundation" | "domain" | "infrastructure" | "application",
    details?: Record<string, unknown>,
  ) {
    super(message);
    this.name = "LegibleError";
    this.code = code;
    this.layer = layer;
    this.timestamp = Date.now();
    this.details = details;

    // Maintains proper stack trace in V8
    if (Error.captureStackTrace) {
      Error.captureStackTrace(this, this.constructor);
    }
  }

  /**
   * Serialize error for logging/transmission
   */
  toJSON(): Record<string, unknown> {
    return {
      name: this.name,
      message: this.message,
      code: this.code,
      layer: this.layer,
      timestamp: this.timestamp,
      details: this.details,
      stack: this.stack,
    };
  }
}

// ============ Cryptographic Errors ============

export type CryptoErrorCode =
  | "INVALID_KEY_LENGTH"
  | "INVALID_SIGNATURE_LENGTH"
  | "SIGNATURE_VERIFICATION_FAILED"
  | "INVALID_HASH_INPUT"
  | "VRF_VERIFICATION_FAILED"
  | "COMMITMENT_MISMATCH"
  | "INVALID_MERKLE_PROOF";

export class CryptoError extends LegibleError {
  constructor(
    message: string,
    code: CryptoErrorCode,
    details?: Record<string, unknown>,
  ) {
    super(message, code, "foundation", details);
    this.name = "CryptoError";
  }
}

// ============ Validation Errors ============

export type ValidationErrorCode =
  | "INVALID_FORMAT"
  | "OUT_OF_RANGE"
  | "MISSING_FIELD"
  | "TYPE_MISMATCH"
  | "CONSTRAINT_VIOLATION";

export class ValidationError extends LegibleError {
  public readonly field?: string;

  constructor(
    message: string,
    code: ValidationErrorCode,
    field?: string,
    details?: Record<string, unknown>,
  ) {
    super(message, code, "foundation", { ...details, field });
    this.name = "ValidationError";
    this.field = field;
  }
}

// ============ Serialization Errors ============

export type SerializationErrorCode =
  | "ENCODE_FAILED"
  | "DECODE_FAILED"
  | "INVALID_JSON"
  | "SCHEMA_MISMATCH";

export class SerializationError extends LegibleError {
  constructor(
    message: string,
    code: SerializationErrorCode,
    details?: Record<string, unknown>,
  ) {
    super(message, code, "foundation", details);
    this.name = "SerializationError";
  }
}

// ============ Error Utilities ============

/**
 * Type guard to check if error is a LegibleError
 */
export function isLegibleError(error: unknown): error is LegibleError {
  return error instanceof LegibleError;
}

/**
 * Type guard for specific error types
 */
export function isCryptoError(error: unknown): error is CryptoError {
  return error instanceof CryptoError;
}

export function isValidationError(error: unknown): error is ValidationError {
  return error instanceof ValidationError;
}

export function isSerializationError(
  error: unknown,
): error is SerializationError {
  return error instanceof SerializationError;
}

/**
 * Wrap unknown errors in LegibleError
 */
export function wrapError(
  error: unknown,
  context: string,
  layer:
    | "foundation"
    | "domain"
    | "infrastructure"
    | "application" = "foundation",
): LegibleError {
  if (error instanceof LegibleError) {
    return error;
  }

  if (error instanceof Error) {
    return new LegibleError(
      `${context}: ${error.message}`,
      "WRAPPED_ERROR",
      layer,
      { originalName: error.name, originalStack: error.stack },
    );
  }

  return new LegibleError(
    `${context}: ${String(error)}`,
    "UNKNOWN_ERROR",
    layer,
    { originalValue: String(error) },
  );
}

/**
 * Assert condition and throw ValidationError if false
 */
export function assertValid(
  condition: boolean,
  message: string,
  field?: string,
): asserts condition {
  if (!condition) {
    throw new ValidationError(message, "CONSTRAINT_VIOLATION", field);
  }
}
