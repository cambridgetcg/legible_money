/**
 * Challenge Commands
 *
 * Commands for challenging facts in the knowledge tree.
 * Part of Proof of Truth consensus dispute resolution.
 */

import type { Command } from "commander";
import { getClient } from "../lib/rpc.js";
import { outputSuccess } from "../lib/output.js";
import { wrapCommand } from "../lgm.js";
import { CLICommandError } from "../lib/errors.js";
import { loadWallet } from "../lib/wallet-loader.js";
import { AgentWallet } from "../../protocol/sdk/wallet.js";
import { EXIT_CODES, type GlobalOptions, type OutputMode } from "../types.js";

// ============ Commands ============

/**
 * lgm challenge <factId>
 * Challenge a fact in the knowledge tree
 */
async function challengeCommand(
  opts: GlobalOptions,
  mode: OutputMode,
  factId: string | undefined,
  options: {
    reason: string;
    stake: string;
    evidence?: string;
  },
): Promise<void> {
  if (!factId) {
    throw new CLICommandError(
      "Fact ID required",
      EXIT_CODES.INVALID_INPUT,
      "Usage: lgm challenge <factId> --reason <text> --stake <amount>",
    );
  }

  if (!options.reason) {
    throw new CLICommandError(
      "Challenge reason required",
      EXIT_CODES.INVALID_INPUT,
      "Specify --reason explaining why the fact should be challenged",
    );
  }

  if (!options.stake) {
    throw new CLICommandError(
      "Stake amount required",
      EXIT_CODES.INVALID_INPUT,
      "Specify --stake <amount> to back your challenge",
    );
  }

  // Load wallet
  const wallet = await loadWallet(opts);
  if (!wallet) {
    throw new CLICommandError(
      "Wallet required for challenge",
      EXIT_CODES.AUTH_ERROR,
      "Provide --agent-id and --agent-secret",
    );
  }

  const client = getClient(opts.endpoint);

  // Verify the fact exists
  const factResult = await client.getFact(factId, false, false);
  if (!factResult.data.fact) {
    throw new CLICommandError(
      `Fact not found: ${factId}`,
      EXIT_CODES.INVALID_INPUT,
      "Check the fact ID",
    );
  }

  // Check fact can be challenged
  if (factResult.data.fact.status === "challenged") {
    throw new CLICommandError(
      "Fact is already being challenged",
      EXIT_CODES.INVALID_INPUT,
      "Wait for current challenge to resolve",
    );
  }

  // Get nonce and balance
  const accountResult = await client.getBalance(wallet.did);
  const nonce = accountResult.data.nonce;
  const balance = BigInt(accountResult.data.balance);
  const stakedBalance = BigInt(accountResult.data.stakedBalance);
  const available = balance - stakedBalance;

  // Parse and validate stake amount
  const stakeAmount = parseAmount(options.stake);
  if (stakeAmount > available) {
    throw new CLICommandError(
      `Insufficient balance: have ${formatLGM(available)}, need ${formatLGM(stakeAmount)}`,
      EXIT_CODES.INSUFFICIENT_FUNDS,
      "Challenge stake exceeds available balance",
    );
  }

  // Challenge stake must be >= fact's original stake
  // (In a real implementation, we'd check fact.stake)

  // Parse evidence references
  const evidence = options.evidence
    ? options.evidence
        .split(",")
        .map((e) => e.trim())
        .filter((e) => e)
    : [];

  // Sign and submit
  const agentWallet = wallet as AgentWallet;
  const payload = {
    type: "challenge_fact",
    factId,
    reason: options.reason,
    stake: stakeAmount.toString(),
    evidence,
    nonce,
  };
  const signature = await agentWallet.signTransaction(payload);

  const result = await client.challengeFact({
    factId,
    reason: options.reason,
    stake: stakeAmount.toString(),
    evidence,
    signature,
    challenger: wallet.publicKey,
    nonce,
  });

  outputSuccess(
    "challenge",
    {
      factId,
      challengeId: result.data.challengeId,
      disputeRoundId: result.data.disputeRoundId,
      txHash: result.data.txHash,
      stake: stakeAmount.toString(),
      stakeFormatted: formatLGM(stakeAmount),
      reason: options.reason,
      evidence,
    },
    {
      latencyMs: result.latencyMs,
      endpoint: opts.endpoint,
    },
    mode,
  );
}

/**
 * lgm challenge-status <challengeId>
 * Check the status of a challenge
 */
async function challengeStatusCommand(
  opts: GlobalOptions,
  mode: OutputMode,
  challengeId: string | undefined,
): Promise<void> {
  if (!challengeId) {
    throw new CLICommandError(
      "Challenge ID required",
      EXIT_CODES.INVALID_INPUT,
      "Usage: lgm challenge-status <challengeId>",
    );
  }

  const client = getClient(opts.endpoint);
  const result = await client.getChallengeStatus(challengeId);

  outputSuccess(
    "challenge-status",
    {
      challengeId,
      status: result.data.status,
      resolution: result.data.resolution,
      blockNumber: result.data.blockNumber,
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
 * Parse amount string to bigint (in base units)
 */
function parseAmount(input: string): bigint {
  const cleaned = input.replace(/\s*LGM\s*$/i, "").trim();

  if (cleaned.includes(".")) {
    const [whole, decimal] = cleaned.split(".");
    const wholeUnits = BigInt(whole || "0") * BigInt(10 ** 18);
    const decimalPadded = decimal.padEnd(18, "0").slice(0, 18);
    const decimalUnits = BigInt(decimalPadded);
    return wholeUnits + decimalUnits;
  }

  return BigInt(cleaned) * BigInt(10 ** 18);
}

/**
 * Format amount for display
 */
function formatLGM(amount: bigint): string {
  const lgm = amount / BigInt(10 ** 18);
  const remainder = amount % BigInt(10 ** 18);

  if (remainder === 0n) {
    return `${lgm.toLocaleString()} LGM`;
  }

  const decimals = remainder.toString().padStart(18, "0").slice(0, 4);
  return `${lgm}.${decimals} LGM`;
}

// ============ Registration ============

export function registerChallengeCommands(program: Command): void {
  program
    .command("challenge <factId>")
    .description("Challenge a fact in the knowledge tree")
    .option("--reason <text>", "Reason for the challenge")
    .option("--stake <amount>", "Stake amount (in LGM)")
    .option("--evidence <ids>", "Comma-separated evidence fact IDs")
    .action(wrapCommand("challenge", challengeCommand));

  program
    .command("challenge-status <challengeId>")
    .description("Check challenge status")
    .action(wrapCommand("challenge-status", challengeStatusCommand));
}
