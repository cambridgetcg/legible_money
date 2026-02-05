/**
 * Transfer Command
 *
 * Send LGM tokens between accounts.
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
 * lgm transfer <to> --amount <amount>
 * Send tokens to another account
 */
async function transferCommand(
  opts: GlobalOptions,
  mode: OutputMode,
  to: string | undefined,
  options: {
    amount: string;
    memo?: string;
  },
): Promise<void> {
  if (!to) {
    throw new CLICommandError(
      "Recipient DID required",
      EXIT_CODES.INVALID_INPUT,
      "Usage: lgm transfer <to> --amount <amount>",
    );
  }

  if (!options.amount) {
    throw new CLICommandError(
      "Amount required",
      EXIT_CODES.INVALID_INPUT,
      "Specify --amount <amount> in LGM",
    );
  }

  // Load wallet for signing
  const wallet = await loadWallet(opts);
  if (!wallet) {
    throw new CLICommandError(
      "Wallet required for transfer",
      EXIT_CODES.AUTH_ERROR,
      "Provide --agent-id and --agent-secret, or set LGM_PRIVATE_KEY",
    );
  }

  const client = getClient(opts.endpoint);

  // Get current nonce
  const accountResult = await client.getBalance(wallet.did);
  const nonce = accountResult.data.nonce;

  // Check sufficient balance
  const amount = parseAmount(options.amount);
  const balance = BigInt(accountResult.data.balance);
  const stakedBalance = BigInt(accountResult.data.stakedBalance);
  const available = balance - stakedBalance;

  if (amount > available) {
    throw new CLICommandError(
      `Insufficient balance: have ${formatLGM(available)}, need ${formatLGM(amount)}`,
      EXIT_CODES.INSUFFICIENT_FUNDS,
      "Transfer amount exceeds available balance (balance minus staked)",
    );
  }

  // Build and sign the transfer
  const payload = {
    type: "transfer",
    from: wallet.did,
    to,
    amount: amount.toString(),
    nonce,
  };

  const agentWallet = wallet as AgentWallet;
  const signature = await agentWallet.signTransaction(payload);

  // Submit transfer
  const result = await client.transfer({
    from: wallet.did,
    to,
    amount: amount.toString(),
    signature,
    nonce,
  });

  outputSuccess(
    "transfer",
    {
      txHash: result.data.txHash,
      from: wallet.did,
      to,
      amount: amount.toString(),
      amountFormatted: formatLGM(amount),
    },
    {
      latencyMs: result.latencyMs,
      endpoint: opts.endpoint,
    },
    mode,
  );
}

/**
 * lgm transfer status <txHash>
 * Check transfer status
 */
async function statusCommand(
  opts: GlobalOptions,
  mode: OutputMode,
  txHash: string | undefined,
): Promise<void> {
  if (!txHash) {
    throw new CLICommandError(
      "Transaction hash required",
      EXIT_CODES.INVALID_INPUT,
      "Usage: lgm transfer status <txHash>",
    );
  }

  const client = getClient(opts.endpoint);
  const result = await client.getTransactionStatus(txHash);

  outputSuccess(
    "transfer status",
    {
      txHash,
      status: result.data.status,
      blockNumber: result.data.blockNumber,
      error: result.data.error,
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
 * Supports: "100", "100 LGM", "100.5"
 */
function parseAmount(input: string): bigint {
  // Remove "LGM" suffix if present
  const cleaned = input.replace(/\s*LGM\s*$/i, "").trim();

  // Handle decimal amounts
  if (cleaned.includes(".")) {
    const [whole, decimal] = cleaned.split(".");
    const wholeUnits = BigInt(whole || "0") * BigInt(10 ** 18);
    // Pad or truncate decimal to 18 places
    const decimalPadded = decimal.padEnd(18, "0").slice(0, 18);
    const decimalUnits = BigInt(decimalPadded);
    return wholeUnits + decimalUnits;
  }

  // Whole number - assume it's already in LGM
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

  // Show up to 4 decimal places
  const decimals = remainder.toString().padStart(18, "0").slice(0, 4);
  return `${lgm}.${decimals} LGM`;
}

// ============ Registration ============

export function registerTransferCommands(program: Command): void {
  program
    .command("transfer [to]")
    .description("Send LGM tokens to another account")
    .option("--amount <amount>", "Amount to send (in LGM)")
    .option("--memo <text>", "Optional memo (not stored on-chain)")
    .action(wrapCommand("transfer", transferCommand));

  // Add status subcommand
  program
    .command("tx-status <txHash>")
    .description("Check transaction status")
    .action(wrapCommand("tx-status", statusCommand));
}
