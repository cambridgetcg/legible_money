/**
 * Verification Commands
 *
 * Commands for validators to participate in verification rounds.
 * Implements commit-reveal scheme for Proof of Truth consensus.
 */

import type { Command } from "commander";
import { getClient } from "../lib/rpc.js";
import { outputSuccess, formatTable } from "../lib/output.js";
import { wrapCommand } from "../lgm.js";
import { CLICommandError } from "../lib/errors.js";
import { loadWallet } from "../lib/wallet-loader.js";
import { AgentWallet } from "../../protocol/sdk/wallet.js";
import { EXIT_CODES, type GlobalOptions, type OutputMode } from "../types.js";
import { randomBytes } from "crypto";

// ============ Commands ============

/**
 * lgm verification pending [validatorKey]
 * List pending verifications for a validator
 */
async function pendingCommand(
  opts: GlobalOptions,
  mode: OutputMode,
  validatorKey: string | undefined,
): Promise<void> {
  // Get validator key from wallet if not provided
  let pubKey = validatorKey;
  if (!pubKey) {
    const wallet = await loadWallet(opts);
    if (!wallet) {
      throw new CLICommandError(
        "Validator key required",
        EXIT_CODES.INVALID_INPUT,
        "Provide validator key or use --agent-id/--agent-secret",
      );
    }
    pubKey = wallet.publicKey;
  }

  const client = getClient(opts.endpoint);
  const result = await client.getPendingVerifications(pubKey);

  // In human mode, show as table
  if (mode === "human" && result.data.assignments.length > 0) {
    console.log("\nPending Verifications");
    console.log("─".repeat(80));

    formatTable(
      ["Round ID", "Claim", "Domain", "Commit By", "Reveal By"],
      result.data.assignments.map((a) => [
        a.roundId.slice(0, 12) + "...",
        a.claim.slice(0, 30) + (a.claim.length > 30 ? "..." : ""),
        a.domain,
        formatBlock(a.commitDeadline),
        formatBlock(a.revealDeadline),
      ]),
      mode,
    );

    console.log(`\nTotal: ${result.data.assignments.length} pending`);
    return;
  }

  outputSuccess(
    "verification pending",
    {
      validatorKey: pubKey,
      assignments: result.data.assignments,
      count: result.data.assignments.length,
    },
    {
      latencyMs: result.latencyMs,
      endpoint: opts.endpoint,
    },
    mode,
  );
}

/**
 * lgm verification commit <roundId>
 * Submit a verification commitment (commit phase)
 */
async function commitCommand(
  opts: GlobalOptions,
  mode: OutputMode,
  roundId: string | undefined,
  options: {
    verdict: string;
    confidence: string;
  },
): Promise<void> {
  if (!roundId) {
    throw new CLICommandError(
      "Round ID required",
      EXIT_CODES.INVALID_INPUT,
      "Usage: lgm verification commit <roundId> --verdict <true|false|uncertain> --confidence <0-1>",
    );
  }

  if (!options.verdict) {
    throw new CLICommandError(
      "Verdict required",
      EXIT_CODES.INVALID_INPUT,
      "Specify --verdict: confirmed, rejected, or uncertain",
    );
  }

  if (!options.confidence) {
    throw new CLICommandError(
      "Confidence required",
      EXIT_CODES.INVALID_INPUT,
      "Specify --confidence as a value between 0 and 1 (e.g., 0.85)",
    );
  }

  // Validate verdict
  const validVerdicts = ["confirmed", "rejected", "uncertain"];
  const verdict = options.verdict.toLowerCase();
  if (!validVerdicts.includes(verdict)) {
    throw new CLICommandError(
      `Invalid verdict: ${options.verdict}`,
      EXIT_CODES.INVALID_INPUT,
      "Verdict must be: confirmed, rejected, or uncertain",
    );
  }

  // Validate confidence
  const confidence = parseFloat(options.confidence);
  if (isNaN(confidence) || confidence < 0 || confidence > 1) {
    throw new CLICommandError(
      `Invalid confidence: ${options.confidence}`,
      EXIT_CODES.INVALID_INPUT,
      "Confidence must be a number between 0 and 1",
    );
  }

  // Load wallet
  const wallet = await loadWallet(opts);
  if (!wallet) {
    throw new CLICommandError(
      "Wallet required for commitment",
      EXIT_CODES.AUTH_ERROR,
      "Provide --agent-id and --agent-secret",
    );
  }

  // Generate random nonce for commit-reveal
  const nonce = randomBytes(32).toString("hex");

  // Create commitment hash: hash(verdict || confidence || nonce)
  const commitData = `${verdict}:${confidence}:${nonce}`;
  const commitment = await hashCommitment(commitData);

  // Sign the commitment
  const agentWallet = wallet as AgentWallet;
  const payload = {
    type: "verification_commit",
    roundId,
    commitment,
  };
  const signature = await agentWallet.signTransaction(payload);

  // Submit commitment
  const client = getClient(opts.endpoint);
  const result = await client.submitCommitment({
    roundId,
    commitment,
    validatorKey: wallet.publicKey,
    signature,
  });

  // Output includes nonce for reveal phase
  outputSuccess(
    "verification commit",
    {
      roundId,
      accepted: result.data.accepted,
      commitDeadline: result.data.commitDeadline,
      // IMPORTANT: Agent must store this for reveal phase
      revealData: {
        nonce,
        verdict,
        confidence,
      },
      message: "Store revealData for the reveal phase",
    },
    {
      latencyMs: result.latencyMs,
      endpoint: opts.endpoint,
    },
    mode,
  );
}

/**
 * lgm verification reveal <roundId>
 * Submit verification reveal (reveal phase)
 */
async function revealCommand(
  opts: GlobalOptions,
  mode: OutputMode,
  roundId: string | undefined,
  options: {
    verdict: string;
    confidence: string;
    nonce: string;
    reasoning?: string;
  },
): Promise<void> {
  if (!roundId) {
    throw new CLICommandError(
      "Round ID required",
      EXIT_CODES.INVALID_INPUT,
      "Usage: lgm verification reveal <roundId> --verdict <v> --confidence <c> --nonce <n>",
    );
  }

  if (!options.verdict || !options.confidence || !options.nonce) {
    throw new CLICommandError(
      "Missing reveal data",
      EXIT_CODES.INVALID_INPUT,
      "All of --verdict, --confidence, and --nonce are required (from commit phase)",
    );
  }

  // Validate verdict
  const validVerdicts = ["confirmed", "rejected", "uncertain"];
  const verdict = options.verdict.toLowerCase() as
    | "confirmed"
    | "rejected"
    | "uncertain";
  if (!validVerdicts.includes(verdict)) {
    throw new CLICommandError(
      `Invalid verdict: ${options.verdict}`,
      EXIT_CODES.INVALID_INPUT,
      "Verdict must be: confirmed, rejected, or uncertain",
    );
  }

  const confidence = parseFloat(options.confidence);

  // Load wallet
  const wallet = await loadWallet(opts);
  if (!wallet) {
    throw new CLICommandError(
      "Wallet required for reveal",
      EXIT_CODES.AUTH_ERROR,
      "Provide --agent-id and --agent-secret",
    );
  }

  // Sign the reveal
  const agentWallet = wallet as AgentWallet;
  const payload = {
    type: "verification_reveal",
    roundId,
    verdict,
    confidence,
    nonce: options.nonce,
  };
  const signature = await agentWallet.signTransaction(payload);

  // Submit reveal
  const client = getClient(opts.endpoint);
  const result = await client.submitReveal({
    roundId,
    verdict,
    confidence,
    nonce: options.nonce,
    reasoning: options.reasoning,
    validatorKey: wallet.publicKey,
    signature,
  });

  outputSuccess(
    "verification reveal",
    {
      roundId,
      accepted: result.data.accepted,
      roundStatus: result.data.roundStatus,
      verdict,
      confidence,
    },
    {
      latencyMs: result.latencyMs,
      endpoint: opts.endpoint,
    },
    mode,
  );
}

/**
 * lgm verification round <roundId>
 * Get verification round status
 */
async function roundCommand(
  opts: GlobalOptions,
  mode: OutputMode,
  roundId: string | undefined,
): Promise<void> {
  if (!roundId) {
    throw new CLICommandError(
      "Round ID required",
      EXIT_CODES.INVALID_INPUT,
      "Usage: lgm verification round <roundId>",
    );
  }

  const client = getClient(opts.endpoint);
  const result = await client.getVerificationRound(roundId);

  if (!result.data.round) {
    throw new CLICommandError(
      "Verification round not found",
      EXIT_CODES.INVALID_INPUT,
      "Check the round ID",
    );
  }

  outputSuccess(
    "verification round",
    {
      round: result.data.round,
      result: result.data.result,
    },
    {
      latencyMs: result.latencyMs,
      endpoint: opts.endpoint,
    },
    mode,
  );
}

// ============ Helpers ============

/**
 * Hash commitment data for commit-reveal
 */
async function hashCommitment(data: string): Promise<string> {
  const encoder = new TextEncoder();
  const dataBuffer = encoder.encode(data);
  const hashBuffer = await crypto.subtle.digest("SHA-256", dataBuffer);
  const hashArray = Array.from(new Uint8Array(hashBuffer));
  return hashArray.map((b) => b.toString(16).padStart(2, "0")).join("");
}

/**
 * Format block number for display
 */
function formatBlock(block: number): string {
  return `Block ${block.toLocaleString()}`;
}

// ============ Registration ============

export function registerVerificationCommands(program: Command): void {
  const verification = program
    .command("verification")
    .description("Verification round participation (for validators)");

  verification
    .command("pending [validatorKey]")
    .description("List pending verification assignments")
    .action(wrapCommand("verification pending", pendingCommand));

  verification
    .command("commit <roundId>")
    .description("Submit verification commitment")
    .option("--verdict <verdict>", "Verdict: confirmed, rejected, uncertain")
    .option("--confidence <n>", "Confidence level 0-1 (e.g., 0.85)")
    .action(wrapCommand("verification commit", commitCommand));

  verification
    .command("reveal <roundId>")
    .description("Submit verification reveal")
    .option("--verdict <verdict>", "Verdict (must match commit)")
    .option("--confidence <n>", "Confidence (must match commit)")
    .option("--nonce <nonce>", "Nonce from commit phase")
    .option("--reasoning <text>", "Optional reasoning")
    .action(wrapCommand("verification reveal", revealCommand));

  verification
    .command("round <roundId>")
    .description("Get verification round status")
    .action(wrapCommand("verification round", roundCommand));
}
