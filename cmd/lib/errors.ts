/**
 * CLI Error Handling
 *
 * Structured error handling for agent-first CLI.
 * Maps RPC error codes to CLI exit codes and provides suggestions.
 */

import { RPC_ERROR_CODES } from "../../protocol/types.js";
import { EXIT_CODES, type ExitCode, type CLIError } from "../types.js";

// ============ RPC Error Class ============

/**
 * RPC error with code and metadata
 */
export class RPCError extends Error {
  readonly code: number;
  readonly data?: unknown;
  readonly retryable: boolean;

  constructor(code: number, message: string, data?: unknown) {
    super(message);
    this.name = "RPCError";
    this.code = code;
    this.data = data;
    this.retryable = code === RPC_ERROR_CODES.RATE_LIMITED;
  }
}

/**
 * CLI-specific error with exit code
 */
export class CLICommandError extends Error {
  readonly exitCode: ExitCode;
  readonly suggestion?: string;

  constructor(message: string, exitCode: ExitCode, suggestion?: string) {
    super(message);
    this.name = "CLICommandError";
    this.exitCode = exitCode;
    this.suggestion = suggestion;
  }
}

// ============ Error Name Mapping ============

const ERROR_NAMES: Record<number, string> = {
  [RPC_ERROR_CODES.PARSE_ERROR]: "PARSE_ERROR",
  [RPC_ERROR_CODES.INVALID_REQUEST]: "INVALID_REQUEST",
  [RPC_ERROR_CODES.METHOD_NOT_FOUND]: "METHOD_NOT_FOUND",
  [RPC_ERROR_CODES.INVALID_PARAMS]: "INVALID_PARAMS",
  [RPC_ERROR_CODES.INTERNAL_ERROR]: "INTERNAL_ERROR",
  [RPC_ERROR_CODES.ACCOUNT_NOT_FOUND]: "ACCOUNT_NOT_FOUND",
  [RPC_ERROR_CODES.INSUFFICIENT_BALANCE]: "INSUFFICIENT_BALANCE",
  [RPC_ERROR_CODES.INVALID_SIGNATURE]: "INVALID_SIGNATURE",
  [RPC_ERROR_CODES.INVALID_NONCE]: "INVALID_NONCE",
  [RPC_ERROR_CODES.SESSION_EXPIRED]: "SESSION_EXPIRED",
  [RPC_ERROR_CODES.SESSION_LIMIT_EXCEEDED]: "SESSION_LIMIT_EXCEEDED",
  [RPC_ERROR_CODES.RATE_LIMITED]: "RATE_LIMITED",
  [RPC_ERROR_CODES.NOT_AUTHORIZED]: "NOT_AUTHORIZED",
  [RPC_ERROR_CODES.FACT_NOT_FOUND]: "FACT_NOT_FOUND",
  [RPC_ERROR_CODES.VALIDATOR_NOT_FOUND]: "VALIDATOR_NOT_FOUND",
  [RPC_ERROR_CODES.CLAIM_DUPLICATE]: "CLAIM_DUPLICATE",
  [RPC_ERROR_CODES.ROUND_DEADLINE_PASSED]: "ROUND_DEADLINE_PASSED",
};

// ============ Error Suggestions ============

const ERROR_SUGGESTIONS: Record<number, string> = {
  [RPC_ERROR_CODES.ACCOUNT_NOT_FOUND]:
    "Check the DID format or create the account first",
  [RPC_ERROR_CODES.INSUFFICIENT_BALANCE]:
    "Request funds from faucet (devnet) or add more LGM",
  [RPC_ERROR_CODES.INVALID_SIGNATURE]: "Check your private key or session key",
  [RPC_ERROR_CODES.INVALID_NONCE]:
    "Retry with the correct nonce from account info",
  [RPC_ERROR_CODES.SESSION_EXPIRED]: "Create a new session key",
  [RPC_ERROR_CODES.SESSION_LIMIT_EXCEEDED]:
    "Session spending limit reached, create new session",
  [RPC_ERROR_CODES.RATE_LIMITED]:
    "Wait and retry, or stake LGM for higher limits",
  [RPC_ERROR_CODES.NOT_AUTHORIZED]: "Check your credentials or permissions",
  [RPC_ERROR_CODES.FACT_NOT_FOUND]:
    "Verify the fact ID exists in the knowledge tree",
  [RPC_ERROR_CODES.VALIDATOR_NOT_FOUND]: "Check the validator public key",
  [RPC_ERROR_CODES.CLAIM_DUPLICATE]:
    "This claim already exists, query the existing fact",
  [RPC_ERROR_CODES.ROUND_DEADLINE_PASSED]: "The verification round has closed",
};

// ============ Error Code Mapping ============

/**
 * Map RPC error code to CLI exit code
 */
export function getExitCode(rpcCode: number): ExitCode {
  switch (rpcCode) {
    case RPC_ERROR_CODES.INVALID_PARAMS:
    case RPC_ERROR_CODES.INVALID_REQUEST:
    case RPC_ERROR_CODES.PARSE_ERROR:
      return EXIT_CODES.INVALID_INPUT;

    case RPC_ERROR_CODES.INVALID_SIGNATURE:
    case RPC_ERROR_CODES.NOT_AUTHORIZED:
    case RPC_ERROR_CODES.SESSION_EXPIRED:
    case RPC_ERROR_CODES.SESSION_LIMIT_EXCEEDED:
      return EXIT_CODES.AUTH_ERROR;

    case RPC_ERROR_CODES.INSUFFICIENT_BALANCE:
      return EXIT_CODES.INSUFFICIENT_FUNDS;

    case RPC_ERROR_CODES.RATE_LIMITED:
      return EXIT_CODES.RATE_LIMITED;

    default:
      return EXIT_CODES.GENERIC_ERROR;
  }
}

/**
 * Get human-readable error name
 */
export function getErrorName(code: number): string {
  return ERROR_NAMES[code] || "UNKNOWN_ERROR";
}

/**
 * Get suggestion for resolving error
 */
export function getErrorSuggestion(code: number): string | undefined {
  return ERROR_SUGGESTIONS[code];
}

// ============ Error Formatting ============

/**
 * Format error for CLI output
 */
export function formatError(error: unknown, command: string): CLIError {
  if (error instanceof RPCError) {
    return {
      success: false,
      command,
      error: {
        code: error.code,
        name: getErrorName(error.code),
        message: error.message,
        retryable: error.retryable,
        suggestion: getErrorSuggestion(error.code),
      },
    };
  }

  if (error instanceof CLICommandError) {
    return {
      success: false,
      command,
      error: {
        code: error.exitCode,
        name: "CLI_ERROR",
        message: error.message,
        retryable: false,
        suggestion: error.suggestion,
      },
    };
  }

  // Network errors
  if (
    error instanceof TypeError &&
    (error as Error).message.includes("fetch")
  ) {
    return {
      success: false,
      command,
      error: {
        code: EXIT_CODES.NETWORK_ERROR,
        name: "NETWORK_ERROR",
        message: "Cannot connect to RPC endpoint",
        retryable: true,
        suggestion:
          "Check that the node is running and the endpoint is correct",
      },
    };
  }

  // Generic error
  const message = error instanceof Error ? error.message : String(error);
  return {
    success: false,
    command,
    error: {
      code: EXIT_CODES.GENERIC_ERROR,
      name: "UNKNOWN_ERROR",
      message,
      retryable: false,
    },
  };
}
