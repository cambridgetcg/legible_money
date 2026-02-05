/**
 * Setup Commands
 *
 * Interactive wizard for secure agent setup.
 * Guides humans through creating secure secret storage.
 */

import type { Command } from "commander";
import * as readline from "readline";
import { randomBytes } from "crypto";
import { AgentWallet } from "../../protocol/sdk/wallet.js";
import { outputSuccess } from "../lib/output.js";
import { wrapCommand } from "../lgm.js";
import { CLICommandError } from "../lib/errors.js";
import {
  getStorage,
  saveAgentConfig,
  loadAgentConfig,
  listAgents,
  deleteAgentConfig,
  loadSecretFromRef,
  saveSessionState,
  loadSessionState,
  EncryptedFileStorage,
  KeychainStorage,
  type AgentConfig,
  type SessionConfig,
} from "../lib/secret-storage.js";
import { EXIT_CODES, type GlobalOptions, type OutputMode } from "../types.js";

// ============ Terminal Utilities ============

function createPrompt(): readline.Interface {
  return readline.createInterface({
    input: process.stdin,
    output: process.stdout,
  });
}

async function ask(rl: readline.Interface, question: string): Promise<string> {
  return new Promise((resolve) => {
    rl.question(question, (answer) => {
      resolve(answer.trim());
    });
  });
}

async function askHidden(question: string): Promise<string> {
  return new Promise((resolve) => {
    const rl = readline.createInterface({
      input: process.stdin,
      output: process.stdout,
    });

    // Disable echo
    process.stdout.write(question);
    const stdin = process.stdin;
    const wasRaw = stdin.isRaw;

    if (stdin.isTTY) {
      stdin.setRawMode(true);
    }

    let input = "";
    const onData = (char: Buffer) => {
      const c = char.toString();
      if (c === "\n" || c === "\r") {
        stdin.removeListener("data", onData);
        if (stdin.isTTY && wasRaw !== undefined) {
          stdin.setRawMode(wasRaw);
        }
        process.stdout.write("\n");
        rl.close();
        resolve(input);
      } else if (c === "\u0003") {
        // Ctrl+C
        process.exit(1);
      } else if (c === "\u007F") {
        // Backspace
        if (input.length > 0) {
          input = input.slice(0, -1);
          process.stdout.write("\b \b");
        }
      } else {
        input += c;
        process.stdout.write("•");
      }
    };

    stdin.on("data", onData);
    stdin.resume();
  });
}

function box(title: string, content: string[]): void {
  const width = 68;
  const border = "─".repeat(width);

  console.log(`╭─ ${title} ${"─".repeat(width - title.length - 4)}╮`);
  for (const line of content) {
    const padding = " ".repeat(Math.max(0, width - line.length - 2));
    console.log(`│  ${line}${padding}│`);
  }
  console.log(`╰${border}╯`);
}

/**
 * Generate cryptographically secure entropy
 *
 * Returns 256-bit (32 bytes) entropy as hex string.
 * This is quantum-resistant entropy - the key derivation
 * algorithm can be upgraded without changing the entropy.
 *
 * 256 bits = 64 hex characters = equivalent to 24-word BIP39
 */
function generateSecureEntropy(): string {
  // 256 bits of entropy (equivalent to 24-word mnemonic)
  const entropy = randomBytes(32);
  return entropy.toString("hex");
}

/**
 * Format entropy for human display
 * Groups into 4-char segments for readability
 */
function formatEntropyForDisplay(entropy: string): string[] {
  const segments: string[] = [];
  for (let i = 0; i < entropy.length; i += 4) {
    segments.push(entropy.slice(i, i + 4));
  }
  // Return as 4 rows of 4 segments each
  return [
    segments.slice(0, 4).join(" "),
    segments.slice(4, 8).join(" "),
    segments.slice(8, 12).join(" "),
    segments.slice(12, 16).join(" "),
  ];
}

// ============ Setup Wizard ============

async function setupAgentCommand(
  opts: GlobalOptions,
  mode: OutputMode,
): Promise<void> {
  console.log("\n");
  box("LEGIBLE MONEY AGENT SETUP", [
    "",
    "This wizard will help you securely configure an agent wallet.",
    "The secret will be stored safely and never exposed to the",
    "agent or command line.",
    "",
  ]);

  const rl = createPrompt();

  try {
    // Step 1: Agent Identity
    console.log("\n");
    box("Step 1 of 5: Agent Identity", [
      "",
      "What should this agent be called?",
      "",
      "This ID is used to derive the wallet deterministically.",
      "Use something memorable and unique to this agent.",
      "",
      "Examples:",
      "  • claude-research-assistant",
      "  • trading-bot-prod-1",
      "  • knowledge-curator-v2",
      "",
    ]);

    let agentId = "";
    while (!agentId) {
      agentId = await ask(rl, "  Agent ID: ");
      if (!/^[a-zA-Z0-9_-]{3,64}$/.test(agentId)) {
        console.log(
          "  ⚠️  Must be 3-64 characters, alphanumeric/hyphen/underscore only",
        );
        agentId = "";
      }
      // Check if exists
      const existing = listAgents();
      if (existing.includes(agentId)) {
        console.log(
          `  ⚠️  Agent "${agentId}" already exists. Choose a different name.`,
        );
        agentId = "";
      }
    }

    // Step 2: Generate or Import Secret
    console.log("\n");
    box("Step 2 of 5: Agent Secret (256-bit entropy)", [
      "",
      "How would you like to create the agent secret?",
      "",
      "[1] Generate 256-bit entropy (recommended, quantum-resistant)",
      "[2] Enter my own secret (must be 64 hex chars for full security)",
      "",
    ]);

    const secretChoice = (await ask(rl, "  Choice [1]: ")) || "1";
    let secret: string;
    let entropyDisplay: string[] | null = null;

    if (secretChoice === "2") {
      // Enter own secret
      console.log("\n");
      secret = await askHidden("  Enter secret (hidden): ");
      const confirm = await askHidden("  Confirm secret: ");

      if (secret !== confirm) {
        throw new CLICommandError(
          "Secrets do not match",
          EXIT_CODES.INVALID_INPUT,
          "Try again",
        );
      }

      if (secret.length < 64) {
        console.log(
          "  ⚠️  Warning: For quantum-resistant security, use 64 hex characters (256 bits).",
        );
      }
    } else {
      // Generate 256-bit entropy
      secret = generateSecureEntropy();
      entropyDisplay = formatEntropyForDisplay(secret);

      console.log("\n");
      box("Recovery Key (WRITE THIS DOWN)", [
        "",
        "  256-bit entropy (64 hex characters):",
        "",
        `    ${entropyDisplay[0]}`,
        `    ${entropyDisplay[1]}`,
        `    ${entropyDisplay[2]}`,
        `    ${entropyDisplay[3]}`,
        "",
        "⚠️  This is the ONLY time you will see this key.",
        "⚠️  Store it in a secure location (password manager, safe).",
        "⚠️  Anyone with this key can access the wallet.",
        "",
        "  This entropy is quantum-resistant and future-proof.",
        "",
      ]);

      let confirmed = "";
      while (
        confirmed.toLowerCase() !== "y" &&
        confirmed.toLowerCase() !== "yes"
      ) {
        confirmed = await ask(
          rl,
          "\n  Have you written down the recovery key? [y/N]: ",
        );
        if (
          confirmed.toLowerCase() === "n" ||
          confirmed.toLowerCase() === "no"
        ) {
          console.log("\n  Please write it down before continuing.\n");
        }
      }
    }

    // Step 3: Storage Backend
    console.log("\n");
    box("Step 3 of 5: Secure Storage", [
      "",
      "Where should the secret be stored?",
      "",
      "[1] System Keychain (recommended)",
      "    macOS Keychain / Linux Secret Service / Windows Credential",
      "    ✓ Encrypted at rest",
      "    ✓ Requires user authentication",
      "",
      "[2] Encrypted file",
      "    ~/.legible/secrets/agent-name.enc",
      "    ✓ Portable across systems",
      "    ✓ Works in containers",
      "",
      "[3] Environment variable reference",
      "    Store in: LGM_AGENT_SECRET_<NAME>",
      "    ⚠️  You manage the env var securely",
      "",
    ]);

    const storageChoice = (await ask(rl, "  Choice [1]: ")) || "1";
    let storageBackend: AgentConfig["storageBackend"];
    const storageConfig: Record<string, string> = {};
    let encryptionPassword: string | null = null;

    switch (storageChoice) {
      case "2": {
        storageBackend = "encrypted-file";
        console.log("\n");
        encryptionPassword = await askHidden("  Encryption password: ");
        const confirmPw = await askHidden("  Confirm password: ");
        if (encryptionPassword !== confirmPw) {
          throw new CLICommandError(
            "Passwords do not match",
            EXIT_CODES.INVALID_INPUT,
            "Try again",
          );
        }
        break;
      }
      case "3":
        storageBackend = "env-ref";
        break;
      default:
        storageBackend = "keychain";
    }

    // Step 4: Session Configuration
    console.log("\n");
    box("Step 4 of 5: Agent Permissions", [
      "",
      "Session keys limit what the agent can do, protecting you from",
      "bugs, prompt injection, or compromised agents.",
      "",
    ]);

    const maxPerTx =
      (await ask(rl, "  Maximum spend per transaction [100] LGM: ")) || "100";
    const maxPerDay =
      (await ask(rl, "  Maximum spend per day [1000] LGM: ")) || "1000";
    const sessionHours =
      (await ask(rl, "  Session duration [24] hours: ")) || "24";

    console.log("\n  Allowed actions:");
    const actions: string[] = [];

    const allowQuery = await ask(rl, "  [y/N] Query knowledge (read-only): ");
    if (allowQuery.toLowerCase() === "y") actions.push("query");

    const allowSubmit = await ask(rl, "  [y/N] Submit claims: ");
    if (allowSubmit.toLowerCase() === "y") actions.push("submit");

    const allowTransfer = await ask(rl, "  [y/N] Transfer tokens: ");
    if (allowTransfer.toLowerCase() === "y") actions.push("transfer");

    const allowStake = await ask(rl, "  [y/N] Stake/unstake: ");
    if (allowStake.toLowerCase() === "y") actions.push("stake", "unstake");

    const allowChallenge = await ask(rl, "  [y/N] Challenge facts: ");
    if (allowChallenge.toLowerCase() === "y") actions.push("challenge");

    // Default to query if nothing selected
    if (actions.length === 0) {
      actions.push("query");
    }

    // Calculate blocks from hours (block time = 2521ms)
    const blocksPerHour = Math.floor((60 * 60 * 1000) / 2521);
    const sessionBlocks = parseInt(sessionHours) * blocksPerHour;

    const sessionConfig: SessionConfig = {
      maxSpendPerTx: (BigInt(maxPerTx) * BigInt(10 ** 18)).toString(),
      maxSpendPerDay: (BigInt(maxPerDay) * BigInt(10 ** 18)).toString(),
      sessionDurationBlocks: sessionBlocks,
      allowedActions: actions,
    };

    // Create wallet to get DID and public key
    const network = opts.network === "mainnet" ? "mainnet" : "testnet";
    const wallet = AgentWallet.fromAgentId(agentId, secret, network);

    // Step 5: Summary and Activation
    console.log("\n");
    box("Step 5 of 5: Review & Activate", [
      "",
      "Agent Configuration Summary:",
      "",
      `  Agent ID:     ${agentId}`,
      `  DID:          ${wallet.did}`,
      `  Public Key:   ${wallet.publicKey.slice(0, 24)}...`,
      `  Network:      ${network}`,
      "",
      `  Storage:      ${storageBackend}`,
      "",
      "  Permissions:",
      `    Max per tx:   ${maxPerTx} LGM`,
      `    Max per day:  ${maxPerDay} LGM`,
      `    Session:      ${sessionHours} hours`,
      `    Actions:      ${actions.join(", ")}`,
      "",
    ]);

    const confirm = await ask(rl, "  Activate this agent? [Y/n]: ");
    if (confirm.toLowerCase() === "n") {
      console.log("\n  Setup cancelled.\n");
      rl.close();
      return;
    }

    // Store secret
    console.log("\n  Storing secret...");
    const storage = getStorage(storageBackend, storageConfig);
    if (storage instanceof EncryptedFileStorage && encryptionPassword) {
      storage.setPassword(encryptionPassword);
    }
    await storage.store(agentId, secret);
    console.log("  ✓ Secret stored securely");

    // Save config with quantum-upgradeable fields
    const config: AgentConfig = {
      agentId,
      did: wallet.did,
      publicKey: wallet.publicKey,
      network,
      storageBackend,
      storageConfig,
      sessionConfig,
      createdAt: new Date().toISOString(),
      // Quantum-safe upgradability
      cryptoVersion: "v1", // Ed25519 (upgradeable to post-quantum)
      entropyBits: 256, // 256-bit = quantum-resistant entropy
      keyDerivation: {
        algorithm: "domain-hash-sha512",
        params: { iterations: 1 },
      },
    };
    saveAgentConfig(config);
    console.log("  ✓ Agent configuration saved");

    // Clear secret from memory
    secret = "";

    // Success
    console.log("\n");
    box("Setup Complete", [
      "",
      `✓ Agent "${agentId}" is ready`,
      "",
      "HOW THE AGENT USES THIS:",
      "",
      "  The agent runs commands like:",
      "",
      `    lgm transfer did:lgm:recipient --amount 50 \\`,
      `        --agent-ref "${agentId}"`,
      "",
      "  The CLI fetches the secret from secure storage.",
      "  The secret NEVER appears in commands or logs.",
      "",
      "TO UNLOCK (if session expires):",
      "",
      `    lgm agent unlock ${agentId}`,
      "",
      "  You'll be prompted for authentication.",
      "",
    ]);

    rl.close();

    if (mode === "json") {
      outputSuccess(
        "setup agent",
        {
          agentId,
          did: wallet.did,
          publicKey: wallet.publicKey,
          network,
          storageBackend,
          sessionConfig: {
            maxSpendPerTx: maxPerTx,
            maxSpendPerDay: maxPerDay,
            sessionDurationHours: sessionHours,
            allowedActions: actions,
          },
        },
        { latencyMs: 0 },
        mode,
      );
    }
  } catch (error) {
    rl.close();
    throw error;
  }
}

// ============ Agent Management Commands ============

/**
 * lgm agent list
 * List configured agents
 */
async function listAgentsCommand(
  _opts: GlobalOptions,
  mode: OutputMode,
): Promise<void> {
  const agents = listAgents();

  if (agents.length === 0) {
    console.log("\nNo agents configured. Run: lgm setup agent\n");
    return;
  }

  const agentData = agents.map((id) => {
    const config = loadAgentConfig(id);
    const session = loadSessionState(id);
    return {
      agentId: id,
      did: config.did,
      network: config.network,
      storage: config.storageBackend,
      hasActiveSession: !!session,
    };
  });

  if (mode === "human") {
    console.log("\nConfigured Agents:");
    console.log("─".repeat(60));
    for (const agent of agentData) {
      const sessionStatus = agent.hasActiveSession ? "🟢 Active" : "🔴 Locked";
      console.log(`  ${agent.agentId}`);
      console.log(`    DID:     ${agent.did}`);
      console.log(`    Network: ${agent.network}`);
      console.log(`    Storage: ${agent.storage}`);
      console.log(`    Session: ${sessionStatus}`);
      console.log();
    }
    return;
  }

  outputSuccess("agent list", { agents: agentData }, { latencyMs: 0 }, mode);
}

/**
 * lgm agent unlock <agentId>
 * Unlock agent session
 */
async function unlockAgentCommand(
  _opts: GlobalOptions,
  mode: OutputMode,
  agentId: string | undefined,
): Promise<void> {
  if (!agentId) {
    throw new CLICommandError(
      "Agent ID required",
      EXIT_CODES.INVALID_INPUT,
      "Usage: lgm agent unlock <agentId>",
    );
  }

  const config = loadAgentConfig(agentId);

  // For encrypted file, prompt for password
  let password: string | undefined;
  if (config.storageBackend === "encrypted-file") {
    password = await askHidden("Encryption password: ");
  }

  // Verify we can retrieve the secret
  try {
    const { secret } = await loadSecretFromRef(agentId, password);
    // Immediately clear from memory
    void secret;
  } catch (error) {
    throw new CLICommandError(
      "Failed to unlock agent",
      EXIT_CODES.AUTH_ERROR,
      error instanceof Error ? error.message : "Unknown error",
    );
  }

  // Update session state
  // TODO: Get current block from network
  const currentBlock = 12345; // Placeholder
  saveSessionState({
    agentId,
    unlockedAt: Date.now(),
    unlockedAtBlock: currentBlock,
    spentToday: 0n,
    lastResetDate: new Date().toISOString().split("T")[0],
  });

  outputSuccess(
    "agent unlock",
    {
      agentId,
      status: "unlocked",
      expiresInBlocks: config.sessionConfig.sessionDurationBlocks,
    },
    { latencyMs: 0 },
    mode,
  );
}

/**
 * lgm agent lock <agentId>
 * Lock agent session
 */
async function lockAgentCommand(
  _opts: GlobalOptions,
  mode: OutputMode,
  agentId: string | undefined,
): Promise<void> {
  if (!agentId) {
    throw new CLICommandError(
      "Agent ID required",
      EXIT_CODES.INVALID_INPUT,
      "Usage: lgm agent lock <agentId>",
    );
  }

  // Clear session
  // We can't truly delete from the JSON, but we can set it to expired
  saveSessionState({
    agentId,
    unlockedAt: 0,
    unlockedAtBlock: 0,
    spentToday: 0n,
    lastResetDate: "",
  });

  outputSuccess(
    "agent lock",
    {
      agentId,
      status: "locked",
    },
    { latencyMs: 0 },
    mode,
  );
}

/**
 * lgm agent revoke <agentId>
 * Permanently delete agent
 */
async function revokeAgentCommand(
  _opts: GlobalOptions,
  mode: OutputMode,
  agentId: string | undefined,
  options: { force?: boolean },
): Promise<void> {
  if (!agentId) {
    throw new CLICommandError(
      "Agent ID required",
      EXIT_CODES.INVALID_INPUT,
      "Usage: lgm agent revoke <agentId>",
    );
  }

  if (!options.force) {
    const rl = createPrompt();
    const confirm = await ask(
      rl,
      `\nAre you sure you want to permanently revoke agent "${agentId}"? [y/N]: `,
    );
    rl.close();

    if (confirm.toLowerCase() !== "y" && confirm.toLowerCase() !== "yes") {
      console.log("Cancelled.\n");
      return;
    }
  }

  const config = loadAgentConfig(agentId);

  // Delete secret from storage
  const storage = getStorage(config.storageBackend, config.storageConfig);
  await storage.delete(agentId);

  // Delete config
  deleteAgentConfig(agentId);

  outputSuccess(
    "agent revoke",
    {
      agentId,
      status: "revoked",
      warning:
        "Transfer any remaining funds before the agent is fully decommissioned",
    },
    { latencyMs: 0 },
    mode,
  );
}

// ============ Single-Command Init ============

/**
 * lgm init <agent-name>
 *
 * Simplified single-command setup with secure defaults:
 * - Generates 256-bit entropy (quantum-resistant strength)
 * - Stores in system keychain (most secure)
 * - Uses safe permission defaults
 * - Upgradeable crypto version for post-quantum future
 */
async function initAgentCommand(
  opts: GlobalOptions,
  mode: OutputMode,
  agentName: string | undefined,
): Promise<void> {
  if (!agentName) {
    throw new CLICommandError(
      "Agent name required",
      EXIT_CODES.INVALID_INPUT,
      "Usage: lgm init <agent-name>",
    );
  }

  // Validate agent name
  if (!/^[a-zA-Z0-9_-]{3,64}$/.test(agentName)) {
    throw new CLICommandError(
      "Invalid agent name",
      EXIT_CODES.INVALID_INPUT,
      "Must be 3-64 characters, alphanumeric/hyphen/underscore only",
    );
  }

  // Check if exists
  const existing = listAgents();
  if (existing.includes(agentName)) {
    throw new CLICommandError(
      `Agent "${agentName}" already exists`,
      EXIT_CODES.INVALID_INPUT,
      `Choose a different name or run: lgm agent revoke ${agentName}`,
    );
  }

  // Generate 256-bit entropy (equivalent to 24-word mnemonic)
  // This entropy is quantum-resistant in strength
  const entropy = generateSecureEntropy();
  const entropyDisplay = formatEntropyForDisplay(entropy);

  // Create wallet to get DID
  const network = opts.network === "mainnet" ? "mainnet" : "testnet";
  const wallet = AgentWallet.fromAgentId(agentName, entropy, network);

  // Safe defaults for permissions
  const sessionConfig: SessionConfig = {
    maxSpendPerTx: (BigInt(100) * BigInt(10 ** 18)).toString(), // 100 LGM
    maxSpendPerDay: (BigInt(1000) * BigInt(10 ** 18)).toString(), // 1000 LGM
    sessionDurationBlocks: Math.floor((24 * 60 * 60 * 1000) / 2521), // 24 hours
    allowedActions: ["query", "submit", "transfer"], // Safe defaults
  };

  // Store in keychain (most secure default)
  const storage = new KeychainStorage();
  try {
    await storage.store(agentName, entropy);
  } catch (error) {
    throw new CLICommandError(
      "Failed to store secret in keychain",
      EXIT_CODES.GENERIC_ERROR,
      error instanceof Error
        ? error.message
        : "Is your system keychain available?",
    );
  }

  // Save config with quantum-upgradeable fields
  const config: AgentConfig = {
    agentId: agentName,
    did: wallet.did,
    publicKey: wallet.publicKey,
    network,
    storageBackend: "keychain",
    storageConfig: {},
    sessionConfig,
    createdAt: new Date().toISOString(),
    // Quantum-safe upgradability
    cryptoVersion: "v1", // Ed25519 (upgradeable to v2/v3 post-quantum)
    entropyBits: 256, // 256-bit = quantum-resistant entropy
    keyDerivation: {
      algorithm: "domain-hash-sha512",
      params: { iterations: 1 }, // Can upgrade to Argon2id
    },
  };
  saveAgentConfig(config);

  // Output based on mode
  if (mode === "json") {
    outputSuccess(
      "init",
      {
        agentId: agentName,
        did: wallet.did,
        publicKey: wallet.publicKey,
        network,
        entropy, // 256-bit hex for programmatic use
        entropyBits: 256,
        cryptoVersion: "v1",
        warning: "SAVE THIS ENTROPY - shown only once",
      },
      { latencyMs: 0 },
      mode,
    );
    return;
  }

  // Human-readable output
  console.log("\n");
  box("AGENT CREATED", [
    "",
    `  Agent:       ${agentName}`,
    `  DID:         ${wallet.did}`,
    `  Network:     ${network}`,
    `  Crypto:      v1 (Ed25519, upgradeable to post-quantum)`,
    `  Entropy:     256-bit (quantum-resistant strength)`,
    "",
  ]);

  console.log("\n");
  box("RECOVERY KEY - WRITE THIS DOWN", [
    "",
    "  256-bit entropy (64 hex characters):",
    "",
    `    ${entropyDisplay[0]}`,
    `    ${entropyDisplay[1]}`,
    `    ${entropyDisplay[2]}`,
    `    ${entropyDisplay[3]}`,
    "",
    "  ⚠️  This is shown ONLY ONCE",
    "  ⚠️  Store securely (password manager, safe, paper backup)",
    "  ⚠️  Anyone with this key controls the wallet",
    "",
    "  Future-proof: When post-quantum crypto is available,",
    "  your entropy upgrades automatically - no re-keying needed.",
    "",
  ]);

  console.log("\n");
  box("USAGE", [
    "",
    "  The agent uses this reference to sign transactions:",
    "",
    `    lgm transfer did:lgm:recipient --amount 50 \\`,
    `        --agent-ref "${agentName}"`,
    "",
    "  The secret is retrieved from your system keychain.",
    "  It NEVER appears in commands or logs.",
    "",
    "  Defaults (change with: lgm setup agent):",
    "    • Max per transaction: 100 LGM",
    "    • Max per day: 1,000 LGM",
    "    • Actions: query, submit, transfer",
    "",
  ]);
}

// ============ Registration ============

export function registerSetupCommands(program: Command): void {
  // Single-command init (recommended)
  program
    .command("init <agent-name>")
    .description("Create agent with secure defaults (single command)")
    .action(wrapCommand("init", initAgentCommand));

  // Full setup wizard (advanced)
  program
    .command("setup")
    .description("Setup wizards")
    .command("agent")
    .description("Interactive wizard to securely configure an agent (advanced)")
    .action(wrapCommand("setup agent", setupAgentCommand));

  // Agent management
  const agent = program
    .command("agent")
    .description("Agent management commands");

  agent
    .command("list")
    .description("List configured agents")
    .action(wrapCommand("agent list", listAgentsCommand));

  agent
    .command("unlock <agentId>")
    .description("Unlock agent session")
    .action(wrapCommand("agent unlock", unlockAgentCommand));

  agent
    .command("lock <agentId>")
    .description("Lock agent session")
    .action(wrapCommand("agent lock", lockAgentCommand));

  agent
    .command("revoke <agentId>")
    .description("Permanently revoke and delete agent")
    .option("--force", "Skip confirmation")
    .action(wrapCommand("agent revoke", revokeAgentCommand));
}
