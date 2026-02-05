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
import { outputSuccess } from "../lib/output.js";
import type { GlobalOptions, OutputMode } from "../types.js";

// ============ Types ============

interface StartOptions {
  config?: string;
  genesis?: string;
  devnet?: boolean;
  name?: string;
  dataDir?: string;
  rpcPort?: number;
  p2pPort?: number;
  validator?: boolean;
  validatorKey?: string;
}

// ============ Start Command ============

async function handleStart(
  _opts: GlobalOptions,
  mode: OutputMode,
  options: StartOptions,
): Promise<void> {
  const nodeModule = await import("../../infrastructure/node/index.js");
  const { createNode, createDevnetNode, createProductionNode } = nodeModule;
  const { readFileSync } = await import("fs");

  let node: Awaited<ReturnType<typeof createNode>>;

  if (options.devnet) {
    // Devnet mode - quick start with defaults
    const genesisPath = options.genesis || "./devnet/config/genesis.json";
    let genesis: Parameters<typeof createDevnetNode>[1];

    try {
      genesis = JSON.parse(readFileSync(genesisPath, "utf-8"));
    } catch {
      // Minimal genesis for testing
      genesis = {
        network: { chainId: "lgm-devnet-1", name: "Devnet", version: "0.1.0" },
        consensus: {},
        initialValidators: [],
        initialAccounts: [],
        genesisAxioms: [],
        genesisTime: new Date().toISOString(),
      };
      console.warn("[WARN] Genesis file not found, using minimal genesis");
    }

    node = createDevnetNode(options.name || "devnet-node", genesis, {
      rpcPort: options.rpcPort || 8545,
      p2pPort: options.p2pPort || 26656,
      isValidator: options.validator,
      validatorKey: options.validatorKey,
    });
  } else if (options.config && options.genesis) {
    // Production mode from config files
    node = await createNode(options.config, options.genesis);
  } else if (options.dataDir && options.genesis) {
    // Production mode with data directory
    const genesis = JSON.parse(readFileSync(options.genesis, "utf-8"));

    node = createProductionNode(
      options.name || "lgm-node",
      options.dataDir,
      genesis,
      {
        rpcPort: options.rpcPort,
        p2pPort: options.p2pPort,
        isValidator: options.validator,
        validatorKey: options.validatorKey,
      },
    );
  } else {
    throw new Error(
      "Must specify either --devnet, --config+--genesis, or --data-dir+--genesis",
    );
  }

  // Start the node
  const startTime = Date.now();
  await node.start();

  outputSuccess(
    "node:start",
    {
      name: options.name || "lgm-node",
      rpcEndpoint: `http://localhost:${options.rpcPort || 8545}`,
      status: "running",
    },
    { latencyMs: Date.now() - startTime },
    mode,
  );

  // Keep running and handle shutdown
  const shutdown = async () => {
    console.log("[Node] Shutting down...");
    await node.stop();
    process.exit(0);
  };

  process.on("SIGINT", shutdown);
  process.on("SIGTERM", shutdown);

  // Keep alive
  await new Promise(() => {});
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
        ) => handleStart(globalOpts, outputMode, cmdOptions as StartOptions),
      ),
    );
}
