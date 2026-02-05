/**
 * Staking Commands
 *
 * Commands for staking LGM tokens and validator operations.
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
 * lgm stake --amount <amount>
 * Stake tokens to become a validator
 */
async function stakeCommand(
  opts: GlobalOptions,
  mode: OutputMode,
  options: {
    amount: string;
  },
): Promise<void> {
  if (!options.amount) {
    throw new CLICommandError(
      "Stake amount required",
      EXIT_CODES.INVALID_INPUT,
      "Usage: lgm stake --amount <amount>",
    );
  }

  // Load wallet
  const wallet = await loadWallet(opts);
  if (!wallet) {
    throw new CLICommandError(
      "Wallet required for staking",
      EXIT_CODES.AUTH_ERROR,
      "Provide --agent-id and --agent-secret",
    );
  }

  const client = getClient(opts.endpoint);

  // Get current balance and nonce
  const accountResult = await client.getBalance(wallet.did);
  const nonce = accountResult.data.nonce;
  const balance = BigInt(accountResult.data.balance);
  const stakedBalance = BigInt(accountResult.data.stakedBalance);
  const available = balance - stakedBalance;

  // Parse and validate stake amount
  const stakeAmount = parseAmount(options.amount);
  if (stakeAmount > available) {
    throw new CLICommandError(
      `Insufficient balance: have ${formatLGM(available)}, need ${formatLGM(stakeAmount)}`,
      EXIT_CODES.INSUFFICIENT_FUNDS,
      "Stake amount exceeds available balance",
    );
  }

  // Minimum stake check (10,000 LGM)
  const minStake = BigInt(10000) * BigInt(10 ** 18);
  const totalStake = stakedBalance + stakeAmount;
  if (totalStake < minStake) {
    throw new CLICommandError(
      `Minimum stake is 10,000 LGM, total would be ${formatLGM(totalStake)}`,
      EXIT_CODES.INVALID_INPUT,
      "Increase stake amount to meet minimum requirement",
    );
  }

  // Sign and submit
  const agentWallet = wallet as AgentWallet;
  const payload = {
    type: "stake",
    publicKey: wallet.publicKey,
    amount: stakeAmount.toString(),
    nonce,
  };
  const signature = await agentWallet.signTransaction(payload);

  const result = await client.stake({
    publicKey: wallet.publicKey,
    amount: stakeAmount.toString(),
    signature,
    nonce,
  });

  outputSuccess(
    "stake",
    {
      txHash: result.data.txHash,
      publicKey: wallet.publicKey,
      amount: stakeAmount.toString(),
      amountFormatted: formatLGM(stakeAmount),
      activationBlock: result.data.activationBlock,
      newTotalStake: formatLGM(totalStake),
    },
    {
      latencyMs: result.latencyMs,
      endpoint: opts.endpoint,
    },
    mode,
  );
}

/**
 * lgm unstake --amount <amount>
 * Unstake tokens (begins unbonding period)
 */
async function unstakeCommand(
  opts: GlobalOptions,
  mode: OutputMode,
  options: {
    amount: string;
  },
): Promise<void> {
  if (!options.amount) {
    throw new CLICommandError(
      "Unstake amount required",
      EXIT_CODES.INVALID_INPUT,
      "Usage: lgm unstake --amount <amount>",
    );
  }

  // Load wallet
  const wallet = await loadWallet(opts);
  if (!wallet) {
    throw new CLICommandError(
      "Wallet required for unstaking",
      EXIT_CODES.AUTH_ERROR,
      "Provide --agent-id and --agent-secret",
    );
  }

  const client = getClient(opts.endpoint);

  // Get current balance and nonce
  const accountResult = await client.getBalance(wallet.did);
  const nonce = accountResult.data.nonce;
  const stakedBalance = BigInt(accountResult.data.stakedBalance);

  // Parse and validate unstake amount
  const unstakeAmount = parseAmount(options.amount);
  if (unstakeAmount > stakedBalance) {
    throw new CLICommandError(
      `Cannot unstake ${formatLGM(unstakeAmount)}: only ${formatLGM(stakedBalance)} staked`,
      EXIT_CODES.INVALID_INPUT,
      "Unstake amount exceeds staked balance",
    );
  }

  // Check remaining stake meets minimum
  const minStake = BigInt(10000) * BigInt(10 ** 18);
  const remainingStake = stakedBalance - unstakeAmount;
  if (remainingStake > 0n && remainingStake < minStake) {
    throw new CLICommandError(
      `Remaining stake ${formatLGM(remainingStake)} below minimum 10,000 LGM`,
      EXIT_CODES.INVALID_INPUT,
      "Either unstake all or leave at least 10,000 LGM staked",
    );
  }

  // Sign and submit
  const agentWallet = wallet as AgentWallet;
  const payload = {
    type: "unstake",
    publicKey: wallet.publicKey,
    amount: unstakeAmount.toString(),
    nonce,
  };
  const signature = await agentWallet.signTransaction(payload);

  const result = await client.unstake({
    publicKey: wallet.publicKey,
    amount: unstakeAmount.toString(),
    signature,
    nonce,
  });

  // Calculate unbonding period (7 days in blocks)
  const blocksPerDay = Math.floor((24 * 60 * 60 * 1000) / 2521);
  const unbondingBlocks = blocksPerDay * 7;

  outputSuccess(
    "unstake",
    {
      txHash: result.data.txHash,
      publicKey: wallet.publicKey,
      amount: unstakeAmount.toString(),
      amountFormatted: formatLGM(unstakeAmount),
      unlockBlock: result.data.unlockBlock,
      remainingStake: formatLGM(remainingStake),
      unbondingPeriod: "7 days",
      estimatedUnlockBlocks: unbondingBlocks,
    },
    {
      latencyMs: result.latencyMs,
      endpoint: opts.endpoint,
    },
    mode,
  );
}

/**
 * lgm rewards [publicKey]
 * Check pending rewards
 */
async function rewardsCommand(
  opts: GlobalOptions,
  mode: OutputMode,
  publicKey: string | undefined,
): Promise<void> {
  // Get public key from wallet if not provided
  let pubKey = publicKey;
  if (!pubKey) {
    const wallet = await loadWallet(opts);
    if (!wallet) {
      throw new CLICommandError(
        "Public key or wallet required",
        EXIT_CODES.INVALID_INPUT,
        "Provide public key or use --agent-id/--agent-secret",
      );
    }
    pubKey = wallet.publicKey;
  }

  const client = getClient(opts.endpoint);
  const result = await client.getRewards(pubKey);

  const pending = BigInt(result.data.pending);
  const claimed = BigInt(result.data.claimed);

  outputSuccess(
    "rewards",
    {
      publicKey: pubKey,
      pending: result.data.pending,
      pendingFormatted: formatLGM(pending),
      claimed: result.data.claimed,
      claimedFormatted: formatLGM(claimed),
      lastClaimBlock: result.data.lastClaimBlock,
      total: (pending + claimed).toString(),
      totalFormatted: formatLGM(pending + claimed),
    },
    {
      latencyMs: result.latencyMs,
      endpoint: opts.endpoint,
    },
    mode,
  );
}

/**
 * lgm claim-rewards
 * Claim pending rewards
 */
async function claimRewardsCommand(
  opts: GlobalOptions,
  mode: OutputMode,
): Promise<void> {
  // Load wallet
  const wallet = await loadWallet(opts);
  if (!wallet) {
    throw new CLICommandError(
      "Wallet required to claim rewards",
      EXIT_CODES.AUTH_ERROR,
      "Provide --agent-id and --agent-secret",
    );
  }

  const client = getClient(opts.endpoint);

  // Check pending rewards
  const rewardsResult = await client.getRewards(wallet.publicKey);
  const pending = BigInt(rewardsResult.data.pending);

  if (pending === 0n) {
    throw new CLICommandError(
      "No pending rewards to claim",
      EXIT_CODES.INVALID_INPUT,
      "Participate in verification rounds to earn rewards",
    );
  }

  // Get nonce
  const accountResult = await client.getBalance(wallet.did);
  const nonce = accountResult.data.nonce;

  // Sign and submit
  const agentWallet = wallet as AgentWallet;
  const payload = {
    type: "claim_rewards",
    publicKey: wallet.publicKey,
    nonce,
  };
  const signature = await agentWallet.signTransaction(payload);

  const result = await client.claimRewards({
    publicKey: wallet.publicKey,
    signature,
    nonce,
  });

  outputSuccess(
    "claim-rewards",
    {
      txHash: result.data.txHash,
      amount: result.data.amount,
      amountFormatted: formatLGM(BigInt(result.data.amount)),
      publicKey: wallet.publicKey,
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

export function registerStakingCommands(program: Command): void {
  program
    .command("stake")
    .description("Stake LGM tokens to become a validator")
    .option("--amount <amount>", "Amount to stake (in LGM)")
    .action(wrapCommand("stake", stakeCommand));

  program
    .command("unstake")
    .description("Unstake tokens (7-day unbonding period)")
    .option("--amount <amount>", "Amount to unstake (in LGM)")
    .action(wrapCommand("unstake", unstakeCommand));

  program
    .command("rewards [publicKey]")
    .description("Check pending validator rewards")
    .action(wrapCommand("rewards", rewardsCommand));

  program
    .command("claim-rewards")
    .description("Claim pending validator rewards")
    .action(wrapCommand("claim-rewards", claimRewardsCommand));
}
