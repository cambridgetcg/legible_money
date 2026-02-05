/**
 * CLI Types
 *
 * Type definitions for the agent-first CLI tool.
 * Designed for machine-readable output (JSON by default).
 */

// ============ CLI Response Types ============

/**
 * Standard CLI success response
 * All successful commands return this structure
 */
export interface CLISuccess<T = unknown> {
  success: true;
  command: string;
  data: T;
  meta: CLIMeta;
}

/**
 * Standard CLI error response
 * All errors return this structure
 */
export interface CLIError {
  success: false;
  command: string;
  error: {
    code: number;
    name: string;
    message: string;
    retryable: boolean;
    suggestion?: string;
  };
}

/**
 * Union type for all CLI responses
 */
export type CLIResponse<T = unknown> = CLISuccess<T> | CLIError;

/**
 * Metadata included with every response
 */
export interface CLIMeta {
  timestamp: string;
  latencyMs: number;
  blockNumber?: string;
  endpoint?: string;
}

// ============ CLI Options ============

/**
 * Global CLI options available to all commands
 */
export interface GlobalOptions {
  endpoint: string;
  network: "mainnet" | "testnet" | "devnet";
  human: boolean;
  quiet: boolean;
  agentId?: string;
  agentSecret?: string;
  sessionKey?: string;
  agentRef?: string; // Secure: uses stored secret, not CLI arg
  yes: boolean;
}

// ============ Exit Codes ============

/**
 * Exit codes for machine-readable status
 */
export const EXIT_CODES = {
  SUCCESS: 0,
  GENERIC_ERROR: 1,
  INVALID_INPUT: 2,
  NETWORK_ERROR: 3,
  AUTH_ERROR: 4,
  INSUFFICIENT_FUNDS: 5,
  RATE_LIMITED: 6,
} as const;

export type ExitCode = (typeof EXIT_CODES)[keyof typeof EXIT_CODES];

// ============ Output Modes ============

export type OutputMode = "json" | "human" | "quiet";

// ============ Command Context ============

/**
 * Context passed to all command handlers
 */
export interface CommandContext {
  options: GlobalOptions;
  outputMode: OutputMode;
  startTime: number;
}

// ============ Wallet Types ============

export interface WalletInfo {
  did: string;
  publicKey: string;
  network: string;
}

export interface SessionInfo {
  sessionId: string;
  sessionPublicKey: string;
  expiresAt: number;
  maxSpendPerTx: string;
  maxSpendPerDay: string;
  allowedActions: string[];
}
