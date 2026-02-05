/**
 * Network Commands
 *
 * Commands for checking node and network status.
 */

import type { Command } from "commander";
import { getClient } from "../lib/rpc.js";
import { outputSuccess } from "../lib/output.js";
import { wrapCommand } from "../lgm.js";
import type { GlobalOptions, OutputMode } from "../types.js";

// ============ Commands ============

/**
 * lgm network health
 * Check node health status
 */
async function healthCommand(
  opts: GlobalOptions,
  mode: OutputMode,
): Promise<void> {
  const client = getClient(opts.endpoint);
  const result = await client.getHealth();

  outputSuccess(
    "network health",
    result.data,
    {
      latencyMs: result.latencyMs,
      blockNumber: result.data.latestBlock,
      endpoint: opts.endpoint,
    },
    mode,
  );
}

// ============ Registration ============

export function registerNetworkCommands(program: Command): void {
  const network = program
    .command("network")
    .description("Network and node status commands");

  network
    .command("health")
    .description("Check node health and sync status")
    .action(wrapCommand("network health", healthCommand));
}
