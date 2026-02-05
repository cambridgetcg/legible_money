/**
 * Wallet Loader
 *
 * Load wallet from various sources for CLI commands.
 * Priority: agent-ref (secure) > session-key > agent-id+secret > env var > wallet file
 *
 * SECURITY NOTE:
 * - agent-ref is the RECOMMENDED method for agents
 * - agent-id+secret on command line is DEPRECATED (visible in ps, history)
 */

import { existsSync, readFileSync } from "fs";
import { join } from "path";
import { homedir } from "os";
import { AgentWallet } from "../../protocol/sdk/wallet.js";
import type { Wallet } from "../../protocol/types.js";
import type { GlobalOptions } from "../types.js";

// ============ Wallet Loading ============

/**
 * Load wallet based on CLI options
 * Returns null if no wallet is available
 */
export async function loadWallet(opts: GlobalOptions): Promise<Wallet | null> {
  // 0. Agent reference (RECOMMENDED - uses secure storage)
  if (opts.agentRef) {
    return loadWalletFromRef(opts.agentRef, opts.network);
  }

  // 1. Session key (safest for agents - limited permissions)
  // Note: Session key loading would require additional infrastructure
  // For now, we support agent derivation and private key

  // 2. Agent derivation (deterministic, good for agents)
  // ⚠️ DEPRECATED: Secret visible in process list and history
  if (opts.agentId && opts.agentSecret) {
    console.error(
      "[SECURITY WARNING] --agent-secret on command line is deprecated.",
    );
    console.error(
      "Use --agent-ref with secure storage instead. Run: lgm setup agent",
    );
    return AgentWallet.fromAgentId(
      opts.agentId,
      opts.agentSecret,
      opts.network === "mainnet" ? "mainnet" : "testnet",
    );
  }

  // 3. Private key from environment
  const envKey = process.env.LGM_PRIVATE_KEY;
  if (envKey) {
    return AgentWallet.fromPrivateKey(envKey);
  }

  // 4. Wallet file (for humans)
  const walletPath = getWalletPath();
  if (existsSync(walletPath)) {
    return loadWalletFile(walletPath);
  }

  return null;
}

/**
 * Load wallet using agent reference (secure method)
 * Secret is fetched from secure storage, never from command line
 */
async function loadWalletFromRef(
  agentRef: string,
  network: GlobalOptions["network"],
): Promise<Wallet> {
  try {
    // Dynamic import to avoid circular dependency
    const { loadSecretFromRef } = await import("./secret-storage.js");

    const { secret } = await loadSecretFromRef(agentRef);

    // TODO: Validate session if action requires it
    // For now, just load the wallet

    const wallet = AgentWallet.fromAgentId(
      agentRef,
      secret,
      network === "mainnet" ? "mainnet" : "testnet",
    );

    return wallet;
  } catch (error) {
    throw new Error(
      `Failed to load agent "${agentRef}". ` +
        (error instanceof Error ? error.message : "Unknown error") +
        "\nRun: lgm setup agent",
    );
  }
}

/**
 * Get the default wallet path
 */
export function getWalletPath(): string {
  return join(homedir(), ".legible", "wallet.json");
}

/**
 * Load wallet from a JSON file
 */
export function loadWalletFile(path: string): Wallet {
  const content = readFileSync(path, "utf8");
  const data = JSON.parse(content);

  if (data.privateKey) {
    return AgentWallet.fromPrivateKey(data.privateKey);
  }

  if (data.mnemonic) {
    return AgentWallet.fromMnemonic(data.mnemonic);
  }

  if (data.agentId && data.agentSecret) {
    return AgentWallet.fromAgentId(data.agentId, data.agentSecret);
  }

  throw new Error(
    "Invalid wallet file: missing privateKey, mnemonic, or agentId+agentSecret",
  );
}

/**
 * Check if wallet is available
 */
export function hasWallet(opts: GlobalOptions): boolean {
  if (opts.agentRef) return true; // Secure method
  if (opts.agentId && opts.agentSecret) return true; // Deprecated
  if (opts.sessionKey) return true;
  if (process.env.LGM_PRIVATE_KEY) return true;
  if (existsSync(getWalletPath())) return true;
  return false;
}

/**
 * Get wallet info without loading the full wallet
 */
export function getWalletSource(opts: GlobalOptions): string | null {
  if (opts.agentRef) return `agent-ref:${opts.agentRef}`; // Secure
  if (opts.sessionKey) return "session-key";
  if (opts.agentId && opts.agentSecret) return "agent-derivation (DEPRECATED)";
  if (process.env.LGM_PRIVATE_KEY) return "env:LGM_PRIVATE_KEY";
  if (existsSync(getWalletPath())) return `file:${getWalletPath()}`;
  return null;
}
