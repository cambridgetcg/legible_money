/**
 * Debug Commands
 *
 * Utility commands for debugging and development.
 * These commands help agents and developers troubleshoot issues.
 */

import type { Command } from "commander";
import { getClient } from "../lib/rpc.js";
import { outputSuccess } from "../lib/output.js";
import { wrapCommand } from "../lgm.js";
import { CLICommandError } from "../lib/errors.js";
import {
  loadWallet,
  getWalletSource,
  hasWallet,
} from "../lib/wallet-loader.js";
import { EXIT_CODES, type GlobalOptions, type OutputMode } from "../types.js";

// ============ Commands ============

/**
 * lgm debug config
 * Show current CLI configuration
 */
async function configCommand(
  opts: GlobalOptions,
  mode: OutputMode,
): Promise<void> {
  const walletSource = getWalletSource(opts);
  const walletLoaded = hasWallet(opts);

  let walletInfo = null;
  if (walletLoaded) {
    try {
      const wallet = await loadWallet(opts);
      if (wallet) {
        walletInfo = {
          did: wallet.did,
          publicKey: wallet.publicKey.slice(0, 20) + "...",
        };
      }
    } catch {
      walletInfo = { error: "Failed to load wallet" };
    }
  }

  outputSuccess(
    "debug config",
    {
      endpoint: opts.endpoint,
      network: opts.network,
      outputMode: mode,
      wallet: {
        available: walletLoaded,
        source: walletSource,
        info: walletInfo,
      },
      environment: {
        LGM_PRIVATE_KEY: process.env.LGM_PRIVATE_KEY ? "[SET]" : "[NOT SET]",
        NODE_ENV: process.env.NODE_ENV || "development",
      },
    },
    {
      latencyMs: 0,
      endpoint: opts.endpoint,
    },
    mode,
  );
}

/**
 * lgm debug ping
 * Test RPC connection
 */
async function pingCommand(
  opts: GlobalOptions,
  mode: OutputMode,
): Promise<void> {
  const client = getClient(opts.endpoint);
  const start = Date.now();

  try {
    const result = await client.getHealth();
    const latency = Date.now() - start;

    outputSuccess(
      "debug ping",
      {
        status: "connected",
        endpoint: opts.endpoint,
        latencyMs: latency,
        nodeStatus: result.data.status,
        nodeVersion: result.data.version,
        latestBlock: result.data.latestBlock,
        syncStatus: result.data.syncStatus,
        peerCount: result.data.peerCount,
      },
      {
        latencyMs: latency,
        endpoint: opts.endpoint,
      },
      mode,
    );
  } catch (error) {
    throw new CLICommandError(
      `Connection failed: ${error instanceof Error ? error.message : "Unknown error"}`,
      EXIT_CODES.NETWORK_ERROR,
      "Check endpoint URL and ensure node is running",
    );
  }
}

/**
 * lgm debug block [number]
 * Get block information
 */
async function blockCommand(
  opts: GlobalOptions,
  mode: OutputMode,
  blockNumber: string | undefined,
): Promise<void> {
  const client = getClient(opts.endpoint);

  let result;
  if (blockNumber) {
    result = await client.getBlock({ number: blockNumber });
  } else {
    result = await client.getLatestBlock();
  }

  if (!result.data.block) {
    throw new CLICommandError(
      "Block not found",
      EXIT_CODES.INVALID_INPUT,
      "Check the block number",
    );
  }

  outputSuccess(
    "debug block",
    {
      block: result.data.block,
      number: "number" in result.data ? result.data.number : blockNumber,
      hash: "hash" in result.data ? result.data.hash : undefined,
    },
    {
      latencyMs: result.latencyMs,
      endpoint: opts.endpoint,
    },
    mode,
  );
}

/**
 * lgm debug nonce <did>
 * Get account nonce (useful for transaction sequencing)
 */
async function nonceCommand(
  opts: GlobalOptions,
  mode: OutputMode,
  did: string | undefined,
): Promise<void> {
  // Get DID from wallet if not provided
  let targetDid = did;
  if (!targetDid) {
    const wallet = await loadWallet(opts);
    if (!wallet) {
      throw new CLICommandError(
        "DID or wallet required",
        EXIT_CODES.INVALID_INPUT,
        "Provide DID or use --agent-id/--agent-secret",
      );
    }
    targetDid = wallet.did;
  }

  const client = getClient(opts.endpoint);
  const result = await client.getBalance(targetDid);

  outputSuccess(
    "debug nonce",
    {
      did: targetDid,
      nonce: result.data.nonce,
      nextNonce: (BigInt(result.data.nonce) + 1n).toString(),
    },
    {
      latencyMs: result.latencyMs,
      endpoint: opts.endpoint,
    },
    mode,
  );
}

/**
 * lgm debug derive
 * Show key derivation results (for debugging agent key issues)
 */
async function deriveCommand(
  opts: GlobalOptions,
  mode: OutputMode,
): Promise<void> {
  if (!opts.agentId || !opts.agentSecret) {
    throw new CLICommandError(
      "Agent ID and secret required",
      EXIT_CODES.INVALID_INPUT,
      "Provide --agent-id and --agent-secret",
    );
  }

  const wallet = await loadWallet(opts);
  if (!wallet) {
    throw new CLICommandError(
      "Failed to derive wallet",
      EXIT_CODES.GENERIC_ERROR,
      "Check agent ID and secret",
    );
  }

  outputSuccess(
    "debug derive",
    {
      input: {
        agentId: opts.agentId,
        secretLength: opts.agentSecret.length,
        network: opts.network,
      },
      output: {
        did: wallet.did,
        publicKey: wallet.publicKey,
      },
      verification: {
        didPrefix: wallet.did.split(":").slice(0, 2).join(":"),
        keyLength: wallet.publicKey.length,
      },
    },
    {
      latencyMs: 0,
      endpoint: opts.endpoint,
    },
    mode,
  );
}

/**
 * lgm debug rpc <method> [params]
 * Make a raw RPC call (for advanced debugging)
 */
async function rpcCommand(
  opts: GlobalOptions,
  mode: OutputMode,
  method: string | undefined,
  paramsJson: string | undefined,
): Promise<void> {
  if (!method) {
    throw new CLICommandError(
      "RPC method required",
      EXIT_CODES.INVALID_INPUT,
      "Usage: lgm debug rpc <method> [paramsJson]",
    );
  }

  let params;
  if (paramsJson) {
    try {
      params = JSON.parse(paramsJson);
    } catch {
      throw new CLICommandError(
        "Invalid JSON params",
        EXIT_CODES.INVALID_INPUT,
        "Params must be valid JSON",
      );
    }
  }

  const client = getClient(opts.endpoint);
  const result = await client.call(method, params);

  outputSuccess(
    "debug rpc",
    {
      method,
      params,
      result: result.data,
      latencyMs: result.latencyMs,
    },
    {
      latencyMs: result.latencyMs,
      endpoint: opts.endpoint,
    },
    mode,
  );
}

/**
 * lgm debug sign <message>
 * Sign a message with the loaded wallet (for testing signatures)
 */
async function signCommand(
  opts: GlobalOptions,
  mode: OutputMode,
  message: string | undefined,
): Promise<void> {
  if (!message) {
    throw new CLICommandError(
      "Message required",
      EXIT_CODES.INVALID_INPUT,
      "Usage: lgm debug sign <message>",
    );
  }

  const wallet = await loadWallet(opts);
  if (!wallet) {
    throw new CLICommandError(
      "Wallet required",
      EXIT_CODES.AUTH_ERROR,
      "Provide --agent-id and --agent-secret",
    );
  }

  const signature = await wallet.sign(new TextEncoder().encode(message));

  outputSuccess(
    "debug sign",
    {
      message,
      messageHex: Buffer.from(message).toString("hex"),
      signature,
      publicKey: wallet.publicKey,
    },
    {
      latencyMs: 0,
      endpoint: opts.endpoint,
    },
    mode,
  );
}

// ============ Registration ============

export function registerDebugCommands(program: Command): void {
  const debug = program
    .command("debug")
    .description("Debugging and development utilities");

  debug
    .command("config")
    .description("Show current CLI configuration")
    .action(wrapCommand("debug config", configCommand));

  debug
    .command("ping")
    .description("Test RPC connection")
    .action(wrapCommand("debug ping", pingCommand));

  debug
    .command("block [number]")
    .description("Get block information (latest if no number)")
    .action(wrapCommand("debug block", blockCommand));

  debug
    .command("nonce [did]")
    .description("Get account nonce")
    .action(wrapCommand("debug nonce", nonceCommand));

  debug
    .command("derive")
    .description("Show key derivation results")
    .action(wrapCommand("debug derive", deriveCommand));

  debug
    .command("rpc <method> [params]")
    .description("Make raw RPC call (params as JSON)")
    .action(wrapCommand("debug rpc", rpcCommand));

  debug
    .command("sign <message>")
    .description("Sign a message with wallet")
    .action(wrapCommand("debug sign", signCommand));
}
