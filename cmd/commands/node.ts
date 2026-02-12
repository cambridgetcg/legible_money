/**
 * Node Commands
 *
 * CLI commands for running Legible Money nodes.
 *
 * Usage:
 *   lgm node start --config ./config.json --genesis ./genesis.json
 *   lgm node start --devnet
 */

import type { Command } from "commander";
import { wrapCommand } from "../lgm.js";
import type { GlobalOptions, OutputMode } from "../types.js";

// ============ Start Command ============

async function handleStart(
  _opts: GlobalOptions,
  _mode: OutputMode,
  _options: unknown,
): Promise<void> {
  // infrastructure/node module removed — use devnet/cmd/node.ts instead
  throw new Error(
    "The `lgm node start` command is not yet available. Use `npm run devnet` to start a devnet node.",
  );
}

// ============ Registration ============

export function registerNodeCommands(program: Command): void {
  const nodeCmd = program
    .command("node")
    .description("Run a Legible Money node");

  nodeCmd
    .command("start")
    .description("Start a node")
    .option("--config <path>", "Path to node configuration file")
    .option("--genesis <path>", "Path to genesis configuration file")
    .option("--devnet", "Run in devnet mode with defaults")
    .option("--name <name>", "Node name", "lgm-node")
    .option("--data-dir <path>", "Data directory for persistent storage")
    .option("--rpc-port <port>", "RPC port", (v) => parseInt(v, 10))
    .option("--p2p-port <port>", "P2P port", (v) => parseInt(v, 10))
    .option("--validator", "Run as validator")
    .option("--validator-key <key>", "Validator private key (hex)")
    .action(
      wrapCommand(
        "node:start",
        (
          globalOpts: GlobalOptions,
          outputMode: OutputMode,
          cmdOptions: unknown,
        ) => handleStart(globalOpts, outputMode, cmdOptions),
      ),
    );
}
