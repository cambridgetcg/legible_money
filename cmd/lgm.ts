#!/usr/bin/env node
/**
 * Legible Money CLI
 *
 * Agent-first command-line interface for Legible Money protocol.
 * JSON output by default, human-readable with --human flag.
 *
 * @example
 * lgm account balance did:lgm:abc123
 * lgm account balance did:lgm:abc123 --human
 * lgm knowledge query physics.mechanics --limit 10
 */

import { Command } from "commander";
import { EXIT_CODES, type GlobalOptions, type OutputMode } from "./types.js";
import { formatError } from "./lib/errors.js";
import { output } from "./lib/output.js";

// Import commands
import { registerAccountCommands } from "./commands/account.js";
import { registerNetworkCommands } from "./commands/network.js";
import { registerKnowledgeCommands } from "./commands/knowledge.js";
import { registerWalletCommands } from "./commands/wallet.js";
import { registerValidatorCommands } from "./commands/validator.js";
import { registerTransferCommands } from "./commands/transfer.js";
import { registerVerificationCommands } from "./commands/verification.js";
import { registerStakingCommands } from "./commands/staking.js";
import { registerChallengeCommands } from "./commands/challenge.js";
import { registerDebugCommands } from "./commands/debug.js";
import { registerSetupCommands } from "./commands/setup.js";
import { registerNodeCommands } from "./commands/node.js";

// ============ CLI Setup ============

const program = new Command();

program
  .name("lgm")
  .description("Legible Money CLI - Agent-first interface for the LGM protocol")
  .version("0.1.0")

  // Global options
  .option("--endpoint <url>", "RPC endpoint URL", "http://localhost:8545")
  .option("--network <net>", "Network: mainnet, testnet, devnet", "devnet")
  .option("--human", "Human-readable output (default: JSON)")
  .option("--quiet", "Minimal output for scripting")
  .option("--agent-id <id>", "Agent ID for deterministic key derivation")
  .option("--agent-secret <secret>", "Agent secret for key derivation")
  .option("--session-key <key>", "Session key (hex) for limited operations")
  .option("--agent-ref <name>", "Agent reference (uses secure storage)")
  .option("--yes", "Skip confirmation prompts");

// ============ Helper Functions ============

/**
 * Get global options from commander
 */
export function getGlobalOptions(cmd: Command): GlobalOptions {
  const opts = cmd.optsWithGlobals();
  return {
    endpoint: opts.endpoint ?? "http://localhost:8545",
    network: opts.network ?? "devnet",
    human: opts.human ?? false,
    quiet: opts.quiet ?? false,
    agentId: opts.agentId,
    agentSecret: opts.agentSecret,
    sessionKey: opts.sessionKey,
    agentRef: opts.agentRef,
    yes: opts.yes ?? false,
  };
}

/**
 * Determine output mode from options
 */
export function getOutputMode(opts: GlobalOptions): OutputMode {
  if (opts.quiet) return "quiet";
  if (opts.human) return "human";
  return "json";
}

/**
 * Wrap command handler with error handling
 *
 * Note: Uses 'any' for handler args because Commander's dynamic argument passing
 * (positional args + options + Command) doesn't map cleanly to TypeScript types.
 * Type safety is maintained by the actual command handler implementations.
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function wrapCommand(
  commandName: string,
  handler: (
    opts: GlobalOptions,
    mode: OutputMode,
    ...args: any[]
  ) => Promise<void>,
): (...args: any[]) => Promise<void> {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return async (...args: any[]) => {
    // Last arg is the Command object
    const cmd = args[args.length - 1] as Command;
    const opts = getGlobalOptions(cmd);
    const mode = getOutputMode(opts);

    try {
      await handler(opts, mode, ...args.slice(0, -1));
      process.exit(EXIT_CODES.SUCCESS);
    } catch (error) {
      const errorResponse = formatError(error, commandName);
      output(errorResponse, mode);

      // Set exit code based on error
      const exitCode =
        errorResponse.error.code >= 0
          ? errorResponse.error.code
          : EXIT_CODES.GENERIC_ERROR;
      process.exit(exitCode);
    }
  };
}

// ============ Register Commands ============

registerAccountCommands(program);
registerNetworkCommands(program);
registerKnowledgeCommands(program);
registerWalletCommands(program);
registerValidatorCommands(program);
registerTransferCommands(program);
registerVerificationCommands(program);
registerStakingCommands(program);
registerChallengeCommands(program);
registerDebugCommands(program);
registerSetupCommands(program);
registerNodeCommands(program);

// ============ Run CLI ============

program.parse();
