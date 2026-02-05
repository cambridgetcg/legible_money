/**
 * Wallet Commands
 *
 * Commands for wallet management and key derivation.
 * Agent-first: supports deterministic derivation from agent ID.
 */

import { writeFileSync, mkdirSync, existsSync } from "fs";
import { dirname } from "path";
import type { Command } from "commander";
import { AgentWallet } from "../../protocol/sdk/wallet.js";
import { outputSuccess } from "../lib/output.js";
import { wrapCommand } from "../lgm.js";
import { CLICommandError } from "../lib/errors.js";
import {
  getWalletPath,
  loadWallet,
  getWalletSource,
} from "../lib/wallet-loader.js";
import { getClient } from "../lib/rpc.js";
import { EXIT_CODES, type GlobalOptions, type OutputMode } from "../types.js";

// ============ Commands ============

/**
 * lgm wallet derive
 * Derive wallet from agent ID and secret
 */
async function deriveCommand(
  opts: GlobalOptions,
  mode: OutputMode,
  options: { exposePrivate?: boolean },
): Promise<void> {
  if (!opts.agentId || !opts.agentSecret) {
    throw new CLICommandError(
      "Agent ID and secret required for derivation",
      EXIT_CODES.INVALID_INPUT,
      "Usage: lgm wallet derive --agent-id <id> --agent-secret <secret>",
    );
  }

  const network = opts.network === "mainnet" ? "mainnet" : "testnet";
  const wallet = AgentWallet.fromAgentId(
    opts.agentId,
    opts.agentSecret,
    network,
  );

  const data: Record<string, unknown> = {
    did: wallet.did,
    publicKey: wallet.publicKey,
    network,
    derivedFrom: "agent-id",
  };

  // Only expose private key if explicitly requested
  if (options.exposePrivate) {
    const recovery = wallet.getRecoveryInfo();
    data.privateKey = recovery.privateKey;
    data.warning = "NEVER SHARE YOUR PRIVATE KEY";
  }

  outputSuccess("wallet derive", data, { latencyMs: 0 }, mode);
}

/**
 * lgm wallet info
 * Show current wallet information
 */
async function infoCommand(
  opts: GlobalOptions,
  mode: OutputMode,
): Promise<void> {
  const source = getWalletSource(opts);

  if (!source) {
    throw new CLICommandError(
      "No wallet configured",
      EXIT_CODES.AUTH_ERROR,
      "Provide --agent-id and --agent-secret, set LGM_PRIVATE_KEY, or create ~/.legible/wallet.json",
    );
  }

  const wallet = await loadWallet(opts);
  if (!wallet) {
    throw new CLICommandError(
      "Failed to load wallet",
      EXIT_CODES.AUTH_ERROR,
      "Check wallet configuration",
    );
  }

  outputSuccess(
    "wallet info",
    {
      did: wallet.did,
      publicKey: wallet.publicKey,
      source,
    },
    { latencyMs: 0 },
    mode,
  );
}

/**
 * lgm wallet create
 * Create a new random wallet
 */
async function createCommand(
  opts: GlobalOptions,
  mode: OutputMode,
  options: { output?: string; force?: boolean },
): Promise<void> {
  // Generate random agent ID and secret
  const agentId = `agent-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
  const agentSecret = Array.from({ length: 32 }, () =>
    Math.random().toString(36).slice(2, 3),
  ).join("");

  const network = opts.network === "mainnet" ? "mainnet" : "testnet";
  const wallet = AgentWallet.fromAgentId(agentId, agentSecret, network);

  // Determine output path
  const outputPath = options.output || getWalletPath();

  // Check if file exists
  if (existsSync(outputPath) && !options.force) {
    throw new CLICommandError(
      `Wallet file already exists: ${outputPath}`,
      EXIT_CODES.INVALID_INPUT,
      "Use --force to overwrite or specify --output <path>",
    );
  }

  // Create directory if needed
  const dir = dirname(outputPath);
  if (!existsSync(dir)) {
    mkdirSync(dir, { recursive: true });
  }

  // Save wallet
  const walletData = {
    agentId,
    agentSecret,
    did: wallet.did,
    publicKey: wallet.publicKey,
    network,
    createdAt: new Date().toISOString(),
    warning: "KEEP THIS FILE SECURE - it contains your wallet credentials",
  };

  writeFileSync(outputPath, JSON.stringify(walletData, null, 2), {
    mode: 0o600,
  });

  outputSuccess(
    "wallet create",
    {
      did: wallet.did,
      publicKey: wallet.publicKey,
      savedTo: outputPath,
      network,
    },
    { latencyMs: 0 },
    mode,
  );
}

/**
 * lgm wallet import
 * Import wallet from private key or mnemonic
 */
async function importCommand(
  _opts: GlobalOptions,
  mode: OutputMode,
  key: string | undefined,
  options: { format?: string; output?: string; force?: boolean },
): Promise<void> {
  // Read from stdin if no key provided
  let importKey = key;
  if (!importKey) {
    const chunks: Buffer[] = [];
    for await (const chunk of process.stdin) {
      chunks.push(Buffer.from(chunk));
    }
    importKey = Buffer.concat(chunks).toString("utf8").trim();
  }

  if (!importKey) {
    throw new CLICommandError(
      "Private key or mnemonic required",
      EXIT_CODES.INVALID_INPUT,
      "Usage: lgm wallet import <key> or pipe via stdin",
    );
  }

  // Detect format
  const format = options.format || detectFormat(importKey);

  let wallet: AgentWallet;
  if (format === "mnemonic") {
    wallet = AgentWallet.fromMnemonic(importKey);
  } else {
    // hex or raw private key
    wallet = AgentWallet.fromPrivateKey(importKey);
  }

  // Determine output path
  const outputPath = options.output || getWalletPath();

  // Check if file exists
  if (existsSync(outputPath) && !options.force) {
    throw new CLICommandError(
      `Wallet file already exists: ${outputPath}`,
      EXIT_CODES.INVALID_INPUT,
      "Use --force to overwrite or specify --output <path>",
    );
  }

  // Create directory if needed
  const dir = dirname(outputPath);
  if (!existsSync(dir)) {
    mkdirSync(dir, { recursive: true });
  }

  // Save wallet
  const recovery = wallet.getRecoveryInfo();
  const walletData = {
    privateKey: recovery.privateKey,
    did: wallet.did,
    publicKey: wallet.publicKey,
    importedAt: new Date().toISOString(),
    warning: "KEEP THIS FILE SECURE - it contains your private key",
  };

  writeFileSync(outputPath, JSON.stringify(walletData, null, 2), {
    mode: 0o600,
  });

  outputSuccess(
    "wallet import",
    {
      did: wallet.did,
      publicKey: wallet.publicKey,
      savedTo: outputPath,
      format,
    },
    { latencyMs: 0 },
    mode,
  );
}

/**
 * lgm wallet export
 * Export wallet private key
 */
async function exportCommand(
  opts: GlobalOptions,
  mode: OutputMode,
  options: { format?: string },
): Promise<void> {
  const wallet = await loadWallet(opts);
  if (!wallet) {
    throw new CLICommandError(
      "No wallet available to export",
      EXIT_CODES.AUTH_ERROR,
      "Configure wallet first",
    );
  }

  // AgentWallet has getRecoveryInfo
  if (!("getRecoveryInfo" in wallet)) {
    throw new CLICommandError(
      "Cannot export this wallet type",
      EXIT_CODES.INVALID_INPUT,
      "Only agent wallets can be exported",
    );
  }

  const recovery = (wallet as AgentWallet).getRecoveryInfo();
  const format = options.format || "json";

  if (format === "hex") {
    // Just output the private key in quiet mode
    if (mode === "quiet") {
      console.log(recovery.privateKey);
      return;
    }
  }

  outputSuccess(
    "wallet export",
    {
      did: recovery.did,
      publicKey: recovery.publicKey,
      privateKey: recovery.privateKey,
      format,
      warning: recovery.warning,
    },
    { latencyMs: 0 },
    mode,
  );
}

/**
 * lgm wallet rotate-key
 * Rotate the operational key for the current wallet
 */
async function rotateKeyCommand(
  opts: GlobalOptions,
  mode: OutputMode,
): Promise<void> {
  const wallet = await loadWallet(opts);
  if (!wallet) {
    throw new CLICommandError(
      "Wallet required for key rotation",
      EXIT_CODES.AUTH_ERROR,
      "Provide --agent-id and --agent-secret",
    );
  }

  if (!("buildRotateKeyRequest" in wallet)) {
    throw new CLICommandError(
      "Key rotation requires an AgentWallet",
      EXIT_CODES.INVALID_INPUT,
      "Session wallets cannot rotate keys",
    );
  }

  const agentWallet = wallet as AgentWallet;
  const client = getClient(opts.endpoint);

  // Get current nonce
  const accountResult = await client.getBalance(wallet.did);
  const nonce = BigInt(accountResult.data.nonce);

  // Build signed rotation request
  const bundle = await agentWallet.buildRotateKeyRequest(nonce);

  // Submit to network
  const result = await client.rotateKey(bundle.request);

  outputSuccess(
    "wallet rotate-key",
    {
      txHash: result.data.txHash,
      did: wallet.did,
      newKeyVersion: result.data.newKeyVersion,
      newPublicKey: bundle.newPublicKey,
      warning:
        "Your operational key has changed. Update any stored key material.",
    },
    {
      latencyMs: result.latencyMs,
      endpoint: opts.endpoint,
    },
    mode,
  );
}

/**
 * lgm wallet key-info
 * Query current key metadata for an account
 */
async function keyInfoCommand(
  opts: GlobalOptions,
  mode: OutputMode,
  options: { did?: string },
): Promise<void> {
  // Use provided DID or derive from wallet
  let did = options.did;
  if (!did) {
    const wallet = await loadWallet(opts);
    if (!wallet) {
      throw new CLICommandError(
        "DID or wallet required",
        EXIT_CODES.INVALID_INPUT,
        "Provide --did <did> or configure a wallet",
      );
    }
    did = wallet.did;
  }

  const client = getClient(opts.endpoint);
  const result = await client.getKeyInfo(did);

  outputSuccess(
    "wallet key-info",
    {
      did: result.data.did,
      operationalKeyHash: result.data.operationalKeyHash,
      operationalKeyVersion: result.data.operationalKeyVersion,
      isFrozen: result.data.isFrozen,
      isInRecovery: result.data.isInRecovery,
    },
    {
      latencyMs: result.latencyMs,
      endpoint: opts.endpoint,
    },
    mode,
  );
}

// ============ Helpers ============

function detectFormat(key: string): "hex" | "mnemonic" {
  // Hex keys are 64 chars (32 bytes)
  if (/^(0x)?[a-fA-F0-9]{64}$/.test(key)) {
    return "hex";
  }
  // Mnemonics have spaces
  if (key.includes(" ")) {
    return "mnemonic";
  }
  // Default to hex
  return "hex";
}

// ============ Registration ============

export function registerWalletCommands(program: Command): void {
  const wallet = program
    .command("wallet")
    .description("Wallet management and key operations");

  wallet
    .command("derive")
    .description("Derive wallet from agent ID and secret")
    .option("--expose-private", "Include private key in output (dangerous)")
    .action(wrapCommand("wallet derive", deriveCommand));

  wallet
    .command("info")
    .description("Show current wallet information")
    .action(wrapCommand("wallet info", infoCommand));

  wallet
    .command("create")
    .description("Create a new random wallet")
    .option("--output <path>", "Output file path")
    .option("--force", "Overwrite existing wallet")
    .action(wrapCommand("wallet create", createCommand));

  wallet
    .command("import [key]")
    .description("Import wallet from private key or mnemonic")
    .option("--format <format>", "Key format: hex, mnemonic")
    .option("--output <path>", "Output file path")
    .option("--force", "Overwrite existing wallet")
    .action(wrapCommand("wallet import", importCommand));

  wallet
    .command("export")
    .description("Export wallet private key (DANGEROUS)")
    .option("--format <format>", "Output format: json, hex")
    .action(wrapCommand("wallet export", exportCommand));

  wallet
    .command("rotate-key")
    .description("Rotate the operational key for the current wallet")
    .action(wrapCommand("wallet rotate-key", rotateKeyCommand));

  wallet
    .command("key-info")
    .description("Query key metadata for an account")
    .option("--did <did>", "Account DID (defaults to current wallet)")
    .action(wrapCommand("wallet key-info", keyInfoCommand));
}
