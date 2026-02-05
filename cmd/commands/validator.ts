/**
 * Validator Commands
 *
 * Commands for viewing validator information and rewards.
 */

import type { Command } from "commander";
import { getClient } from "../lib/rpc.js";
import { outputSuccess, formatTable } from "../lib/output.js";
import { wrapCommand } from "../lgm.js";
import type { GlobalOptions, OutputMode } from "../types.js";

// ============ Commands ============

/**
 * lgm validator list
 * List all validators
 */
async function listCommand(
  opts: GlobalOptions,
  mode: OutputMode,
  options: { activeOnly?: boolean; epoch?: string },
): Promise<void> {
  const client = getClient(opts.endpoint);
  const result = await client.getValidatorSet(options.epoch);

  let validators = result.data.validators;

  // Filter active only if requested
  if (options.activeOnly) {
    validators = validators.filter((v) => v.isActive);
  }

  // Calculate accuracy helper
  const getAccuracy = (v: (typeof validators)[0]) => {
    if (v.verificationsCompleted === 0) return 0;
    return v.verificationsCorrect / v.verificationsCompleted;
  };

  // In human mode, show as table
  if (mode === "human" && validators.length > 0) {
    console.log("\nValidator Set");
    console.log("─".repeat(70));

    formatTable(
      ["Rank", "Public Key", "Stake", "Accuracy", "Status"],
      validators.map((v, i) => [
        String(i + 1),
        v.publicKey.slice(0, 16) + "...",
        formatStake(v.stake),
        (getAccuracy(v) * 100).toFixed(1) + "%",
        v.isActive ? "Active" : "Inactive",
      ]),
      mode,
    );

    console.log(`\nTotal Stake: ${formatStake(result.data.totalStake)}`);
    console.log(`Active: ${result.data.activeCount} validators`);
    console.log(`Epoch: ${result.data.epoch}`);
    return;
  }

  outputSuccess(
    "validator list",
    {
      validators: validators.map((v) => ({
        publicKey: v.publicKey,
        stake: v.stake.toString(),
        verificationAccuracy: getAccuracy(v),
        isActive: v.isActive,
        activeSinceBlock: v.activeSinceBlock?.toString(),
      })),
      totalStake: result.data.totalStake,
      activeCount: result.data.activeCount,
      epoch: result.data.epoch,
    },
    {
      latencyMs: result.latencyMs,
      endpoint: opts.endpoint,
    },
    mode,
  );
}

/**
 * lgm validator pending <validatorKey>
 * Get pending verification assignments
 */
async function pendingCommand(
  opts: GlobalOptions,
  mode: OutputMode,
  validatorKey: string,
): Promise<void> {
  const client = getClient(opts.endpoint);
  const result = await client.getPendingVerifications(validatorKey);

  // In human mode, show as table
  if (mode === "human" && result.data.assignments.length > 0) {
    console.log("\nPending Verifications");
    console.log("─".repeat(60));

    formatTable(
      ["Round", "Claim", "Domain", "Commit Deadline"],
      result.data.assignments.map((a) => [
        a.roundId.slice(0, 12) + "...",
        a.claim.slice(0, 30) + (a.claim.length > 30 ? "..." : ""),
        a.domain,
        String(a.commitDeadline),
      ]),
      mode,
    );
    return;
  }

  outputSuccess(
    "validator pending",
    {
      validatorKey,
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

// ============ Helpers ============

/**
 * Format stake amount for display
 */
function formatStake(stake: bigint | string): string {
  const value = typeof stake === "bigint" ? stake : BigInt(stake);
  const lgm = value / BigInt(10 ** 18);
  return `${lgm.toLocaleString()} LGM`;
}

// ============ Registration ============

export function registerValidatorCommands(program: Command): void {
  const validator = program
    .command("validator")
    .description("Validator information and operations");

  validator
    .command("list")
    .description("List all validators")
    .option("--active-only", "Show only active validators")
    .option("--epoch <epoch>", "Specify epoch")
    .action(wrapCommand("validator list", listCommand));

  validator
    .command("pending <validatorKey>")
    .description("Get pending verification assignments")
    .action(wrapCommand("validator pending", pendingCommand));
}
