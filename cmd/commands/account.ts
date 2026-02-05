/**
 * Account Commands
 *
 * Commands for querying account information.
 */

import type { Command } from "commander";
import { getClient } from "../lib/rpc.js";
import { outputSuccess } from "../lib/output.js";
import { wrapCommand } from "../lgm.js";
import { CLICommandError } from "../lib/errors.js";
import { EXIT_CODES, type GlobalOptions, type OutputMode } from "../types.js";

// ============ Commands ============

/**
 * lgm account balance <did>
 * Get account balance
 */
async function balanceCommand(
  opts: GlobalOptions,
  mode: OutputMode,
  did: string,
): Promise<void> {
  if (!did) {
    throw new CLICommandError(
      "DID required",
      EXIT_CODES.INVALID_INPUT,
      "Usage: lgm account balance <did>",
    );
  }

  const client = getClient(opts.endpoint);
  const result = await client.getBalance(did);

  // Calculate available balance
  const balance = BigInt(result.data.balance);
  const staked = BigInt(result.data.stakedBalance);
  const available = balance - staked;

  outputSuccess(
    "account balance",
    {
      did,
      balance: result.data.balance,
      stakedBalance: result.data.stakedBalance,
      available: available.toString(),
      nonce: result.data.nonce,
    },
    {
      latencyMs: result.latencyMs,
      endpoint: opts.endpoint,
    },
    mode,
  );
}

/**
 * lgm account info <did>
 * Get full account information
 */
async function infoCommand(
  opts: GlobalOptions,
  mode: OutputMode,
  did: string,
  options: { includeProof?: boolean },
): Promise<void> {
  if (!did) {
    throw new CLICommandError(
      "DID required",
      EXIT_CODES.INVALID_INPUT,
      "Usage: lgm account info <did>",
    );
  }

  const client = getClient(opts.endpoint);
  const result = await client.getAccountInfo(did, options.includeProof);

  outputSuccess(
    "account info",
    {
      did,
      account: result.data.account,
      blockNumber: result.data.blockNumber,
      proof: result.data.proof,
    },
    {
      latencyMs: result.latencyMs,
      blockNumber: result.data.blockNumber,
      endpoint: opts.endpoint,
    },
    mode,
  );
}

/**
 * lgm account status <did>
 * Get agent status with activity metrics
 */
async function statusCommand(
  opts: GlobalOptions,
  mode: OutputMode,
  did: string,
): Promise<void> {
  if (!did) {
    throw new CLICommandError(
      "DID required",
      EXIT_CODES.INVALID_INPUT,
      "Usage: lgm account status <did>",
    );
  }

  const client = getClient(opts.endpoint);
  const result = await client.getAgentStatus(did);

  outputSuccess(
    "account status",
    {
      did,
      ...result.data,
    },
    {
      latencyMs: result.latencyMs,
      endpoint: opts.endpoint,
    },
    mode,
  );
}

/**
 * lgm account check <did>
 * Quick check if account exists
 */
async function checkCommand(
  opts: GlobalOptions,
  mode: OutputMode,
  did: string,
): Promise<void> {
  if (!did) {
    throw new CLICommandError(
      "DID required",
      EXIT_CODES.INVALID_INPUT,
      "Usage: lgm account check <did>",
    );
  }

  const client = getClient(opts.endpoint);
  const result = await client.checkAccount(did);

  outputSuccess(
    "account check",
    {
      did,
      ...result.data,
    },
    {
      latencyMs: result.latencyMs,
      endpoint: opts.endpoint,
    },
    mode,
  );
}

// ============ Bootstrap Claim ============

/**
 * lgm account claim
 * Claim free 0.222 LGM bootstrap fund (first 1,001,001,000 agents)
 */
async function claimBootstrapCommand(
  opts: GlobalOptions,
  mode: OutputMode,
): Promise<void> {
  // Need wallet to claim
  const { loadWallet } = await import("../lib/wallet-loader.js");
  const wallet = await loadWallet(opts);

  if (!wallet) {
    throw new CLICommandError(
      "Wallet required to claim",
      EXIT_CODES.AUTH_ERROR,
      "Use --agent-ref <name> or run: lgm init <agent-name>",
    );
  }

  const client = getClient(opts.endpoint);

  // Check bootstrap status first
  const status = await client.call<any>("bootstrap_status", {});

  if (!status.data.isActive) {
    throw new CLICommandError(
      "Bootstrap fund exhausted",
      EXIT_CODES.GENERIC_ERROR,
      `All ${status.data.maxClaims} claims have been made`,
    );
  }

  // Attempt to claim
  try {
    const result = await client.call<any>("bootstrap_claim", {
      did: wallet.did,
      signature: await wallet.signHex(wallet.did),
    });

    outputSuccess(
      "account claim",
      {
        did: wallet.did,
        amount: result.data.amount,
        amountFormatted: result.data.amountFormatted,
        newBalance: result.data.newBalance,
        txHash: result.data.txHash,
        claimNumber: result.data.claimNumber,
        remainingClaims: result.data.remainingClaims,
        message: "Welcome to Legible Money!",
      },
      {
        latencyMs: result.latencyMs,
        endpoint: opts.endpoint,
      },
      mode,
    );
  } catch (error: any) {
    if (error.code === -32003) {
      throw new CLICommandError(
        "Already claimed",
        EXIT_CODES.INVALID_INPUT,
        "Bootstrap fund can only be claimed once per agent",
      );
    }
    throw error;
  }
}

/**
 * lgm account bootstrap-status
 * Check bootstrap fund status
 */
async function bootstrapStatusCommand(
  opts: GlobalOptions,
  mode: OutputMode,
): Promise<void> {
  const client = getClient(opts.endpoint);
  const result = await client.call<any>("bootstrap_status", {});

  outputSuccess(
    "account bootstrap-status",
    {
      ...result.data,
      totalFund: "222,222,222 LGM",
      amountPerAgent: "0.222 LGM",
      maxAgents: "1,001,001,000",
    },
    {
      latencyMs: result.latencyMs,
      endpoint: opts.endpoint,
    },
    mode,
  );
}

// ============ Registration ============

export function registerAccountCommands(program: Command): void {
  const account = program
    .command("account")
    .description("Account queries and operations");

  account
    .command("balance <did>")
    .description("Get account balance")
    .action(wrapCommand("account balance", balanceCommand));

  account
    .command("info <did>")
    .description("Get full account information")
    .option("--include-proof", "Include Merkle proof")
    .action(wrapCommand("account info", infoCommand));

  account
    .command("status <did>")
    .description("Get agent activity status")
    .action(wrapCommand("account status", statusCommand));

  account
    .command("check <did>")
    .description("Quick check if account exists")
    .action(wrapCommand("account check", checkCommand));

  // Bootstrap fund commands
  account
    .command("claim")
    .description("Claim free 0.222 LGM bootstrap (first 1B agents)")
    .action(wrapCommand("account claim", claimBootstrapCommand));

  account
    .command("bootstrap-status")
    .description("Check bootstrap fund status")
    .action(wrapCommand("account bootstrap-status", bootstrapStatusCommand));
}
