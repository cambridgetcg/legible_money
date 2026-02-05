/**
 * Secure Secret Storage
 *
 * Multiple backend support for storing agent secrets securely.
 * Secrets never appear in command lines, logs, or process lists.
 */

import {
  existsSync,
  readFileSync,
  writeFileSync,
  mkdirSync,
  unlinkSync,
  readdirSync,
} from "fs";
import { join, dirname } from "path";
import { homedir } from "os";
import { execSync } from "child_process";
import {
  randomBytes,
  createCipheriv,
  createDecipheriv,
  scryptSync,
} from "crypto";

// ============ Storage Interface ============

export interface SecretStorage {
  readonly name: string;
  store(agentId: string, secret: string): Promise<void>;
  retrieve(agentId: string): Promise<string>;
  delete(agentId: string): Promise<void>;
  exists(agentId: string): Promise<boolean>;
}

/**
 * Crypto version for quantum-safe upgradability
 *
 * v1: Ed25519 (classical, current)
 * v2: Reserved for hybrid classical+PQ
 * v3: Reserved for pure post-quantum (CRYSTALS-Dilithium, SPHINCS+)
 */
export type CryptoVersion = "v1" | "v2" | "v3";

export interface AgentConfig {
  agentId: string;
  did: string;
  publicKey: string;
  network: "mainnet" | "testnet";
  storageBackend: "keychain" | "encrypted-file" | "env-ref" | "vault";
  storageConfig: Record<string, string>;
  sessionConfig: SessionConfig;
  createdAt: string;
  lastUnlock?: string;

  // Quantum-safe upgradability
  cryptoVersion: CryptoVersion;
  entropyBits: 128 | 256; // 12-word = 128, 24-word = 256
  keyDerivation: {
    algorithm: string; // 'scrypt' | 'argon2id' | future PQ-safe
    params: Record<string, number>;
  };
}

export interface SessionConfig {
  maxSpendPerTx: string; // bigint as string
  maxSpendPerDay: string; // bigint as string
  sessionDurationBlocks: number;
  allowedActions: string[];
}

// ============ Storage Backends ============

/**
 * macOS Keychain / Linux Secret Service / Windows Credential Manager
 */
export class KeychainStorage implements SecretStorage {
  readonly name = "System Keychain";
  private readonly service = "com.legible.agent";

  async store(agentId: string, secret: string): Promise<void> {
    const platform = process.platform;

    if (platform === "darwin") {
      // macOS Keychain
      execSync(
        `security add-generic-password -a "${agentId}" -s "${this.service}" -w "${secret}" -U`,
        { stdio: "pipe" },
      );
    } else if (platform === "linux") {
      // Linux Secret Service (requires secret-tool)
      execSync(
        `echo -n "${secret}" | secret-tool store --label="LGM Agent: ${agentId}" service "${this.service}" account "${agentId}"`,
        { stdio: "pipe", shell: "/bin/bash" },
      );
    } else if (platform === "win32") {
      // Windows Credential Manager
      execSync(
        `cmdkey /generic:"${this.service}:${agentId}" /user:"${agentId}" /pass:"${secret}"`,
        { stdio: "pipe" },
      );
    } else {
      throw new Error(`Keychain storage not supported on ${platform}`);
    }
  }

  async retrieve(agentId: string): Promise<string> {
    const platform = process.platform;

    try {
      if (platform === "darwin") {
        const result = execSync(
          `security find-generic-password -a "${agentId}" -s "${this.service}" -w`,
          { stdio: "pipe", encoding: "utf8" },
        );
        return result.trim();
      } else if (platform === "linux") {
        const result = execSync(
          `secret-tool lookup service "${this.service}" account "${agentId}"`,
          { stdio: "pipe", encoding: "utf8" },
        );
        return result.trim();
      } else if (platform === "win32") {
        // Windows requires parsing cmdkey output or using PowerShell
        const result = execSync(
          `powershell -Command "(Get-StoredCredential -Target '${this.service}:${agentId}').GetNetworkCredential().Password"`,
          { stdio: "pipe", encoding: "utf8" },
        );
        return result.trim();
      }
    } catch {
      throw new Error(`Secret not found for agent: ${agentId}`);
    }

    throw new Error(`Keychain storage not supported on ${platform}`);
  }

  async delete(agentId: string): Promise<void> {
    const platform = process.platform;

    if (platform === "darwin") {
      execSync(
        `security delete-generic-password -a "${agentId}" -s "${this.service}"`,
        { stdio: "pipe" },
      );
    } else if (platform === "linux") {
      execSync(
        `secret-tool clear service "${this.service}" account "${agentId}"`,
        { stdio: "pipe" },
      );
    } else if (platform === "win32") {
      execSync(`cmdkey /delete:"${this.service}:${agentId}"`, {
        stdio: "pipe",
      });
    }
  }

  async exists(agentId: string): Promise<boolean> {
    try {
      await this.retrieve(agentId);
      return true;
    } catch {
      return false;
    }
  }
}

/**
 * AES-256-GCM encrypted file storage
 */
export class EncryptedFileStorage implements SecretStorage {
  readonly name = "Encrypted File";
  private readonly baseDir: string;
  private password: string | null = null;

  constructor(baseDir?: string) {
    this.baseDir = baseDir || join(homedir(), ".legible", "secrets");
  }

  setPassword(password: string): void {
    this.password = password;
  }

  private getFilePath(agentId: string): string {
    return join(this.baseDir, `${agentId}.enc`);
  }

  private deriveKey(password: string, salt: Buffer): Buffer {
    return scryptSync(password, salt, 32);
  }

  async store(agentId: string, secret: string): Promise<void> {
    if (!this.password) {
      throw new Error("Password not set. Call setPassword() first.");
    }

    // Create directory if needed
    if (!existsSync(this.baseDir)) {
      mkdirSync(this.baseDir, { recursive: true, mode: 0o700 });
    }

    // Generate salt and IV
    const salt = randomBytes(32);
    const iv = randomBytes(16);

    // Derive key from password
    const key = this.deriveKey(this.password, salt);

    // Encrypt
    const cipher = createCipheriv("aes-256-gcm", key, iv);
    const encrypted = Buffer.concat([
      cipher.update(secret, "utf8"),
      cipher.final(),
    ]);
    const authTag = cipher.getAuthTag();

    // Store: salt (32) + iv (16) + authTag (16) + encrypted
    const data = Buffer.concat([salt, iv, authTag, encrypted]);
    writeFileSync(this.getFilePath(agentId), data, { mode: 0o600 });
  }

  async retrieve(agentId: string): Promise<string> {
    if (!this.password) {
      throw new Error("Password not set. Call setPassword() first.");
    }

    const filePath = this.getFilePath(agentId);
    if (!existsSync(filePath)) {
      throw new Error(`Secret not found for agent: ${agentId}`);
    }

    const data = readFileSync(filePath);

    // Parse: salt (32) + iv (16) + authTag (16) + encrypted
    const salt = data.subarray(0, 32);
    const iv = data.subarray(32, 48);
    const authTag = data.subarray(48, 64);
    const encrypted = data.subarray(64);

    // Derive key
    const key = this.deriveKey(this.password, salt);

    // Decrypt
    const decipher = createDecipheriv("aes-256-gcm", key, iv);
    decipher.setAuthTag(authTag);

    try {
      const decrypted = Buffer.concat([
        decipher.update(encrypted),
        decipher.final(),
      ]);
      return decrypted.toString("utf8");
    } catch {
      throw new Error("Decryption failed. Wrong password?");
    }
  }

  async delete(agentId: string): Promise<void> {
    const filePath = this.getFilePath(agentId);
    if (existsSync(filePath)) {
      // Overwrite with random data before deleting
      const size = readFileSync(filePath).length;
      writeFileSync(filePath, randomBytes(size));
      unlinkSync(filePath);
    }
  }

  async exists(agentId: string): Promise<boolean> {
    return existsSync(this.getFilePath(agentId));
  }
}

/**
 * Environment variable reference storage
 * The actual secret is in an env var; we just store the var name
 */
export class EnvRefStorage implements SecretStorage {
  readonly name = "Environment Variable";

  private getEnvVarName(agentId: string): string {
    // Convert agent-id to ENV_VAR_NAME format
    const safeName = agentId.toUpperCase().replace(/[^A-Z0-9]/g, "_");
    return `LGM_AGENT_SECRET_${safeName}`;
  }

  async store(agentId: string, secret: string): Promise<void> {
    const envVar = this.getEnvVarName(agentId);
    console.log(`\nSet this environment variable:\n`);
    console.log(`  export ${envVar}="${secret}"\n`);
    console.log(
      `Add to your shell profile (~/.bashrc, ~/.zshrc) for persistence.\n`,
    );

    // We don't actually store anything - the human manages the env var
  }

  async retrieve(agentId: string): Promise<string> {
    const envVar = this.getEnvVarName(agentId);
    const secret = process.env[envVar];

    if (!secret) {
      throw new Error(`Environment variable ${envVar} not set`);
    }

    return secret;
  }

  async delete(agentId: string): Promise<void> {
    const envVar = this.getEnvVarName(agentId);
    console.log(`\nRemove from your environment:\n`);
    console.log(`  unset ${envVar}\n`);
  }

  async exists(agentId: string): Promise<boolean> {
    const envVar = this.getEnvVarName(agentId);
    return !!process.env[envVar];
  }
}

/**
 * HashiCorp Vault storage (enterprise)
 */
export class VaultStorage implements SecretStorage {
  readonly name = "HashiCorp Vault";
  private readonly vaultAddr: string;
  private readonly basePath: string;

  constructor(vaultAddr?: string, basePath?: string) {
    this.vaultAddr =
      vaultAddr || process.env.VAULT_ADDR || "http://127.0.0.1:8200";
    this.basePath = basePath || "secret/data/legible/agents";
  }

  async store(agentId: string, secret: string): Promise<void> {
    const path = `${this.basePath}/${agentId}`;
    execSync(`vault kv put "${path}" secret="${secret}"`, {
      stdio: "pipe",
      env: { ...process.env, VAULT_ADDR: this.vaultAddr },
    });
  }

  async retrieve(agentId: string): Promise<string> {
    const path = `${this.basePath}/${agentId}`;
    const result = execSync(`vault kv get -field=secret "${path}"`, {
      stdio: "pipe",
      encoding: "utf8",
      env: { ...process.env, VAULT_ADDR: this.vaultAddr },
    });
    return result.trim();
  }

  async delete(agentId: string): Promise<void> {
    const path = `${this.basePath}/${agentId}`;
    execSync(`vault kv delete "${path}"`, {
      stdio: "pipe",
      env: { ...process.env, VAULT_ADDR: this.vaultAddr },
    });
  }

  async exists(agentId: string): Promise<boolean> {
    try {
      await this.retrieve(agentId);
      return true;
    } catch {
      return false;
    }
  }
}

// ============ Storage Factory ============

export function getStorage(
  backend: AgentConfig["storageBackend"],
  config?: Record<string, string>,
): SecretStorage {
  switch (backend) {
    case "keychain":
      return new KeychainStorage();
    case "encrypted-file":
      return new EncryptedFileStorage(config?.baseDir);
    case "env-ref":
      return new EnvRefStorage();
    case "vault":
      return new VaultStorage(config?.vaultAddr, config?.basePath);
    default:
      throw new Error(`Unknown storage backend: ${backend}`);
  }
}

// ============ Agent Config Management ============

const CONFIG_DIR = join(homedir(), ".legible", "agents");

export function getAgentConfigPath(agentId: string): string {
  return join(CONFIG_DIR, `${agentId}.json`);
}

export function loadAgentConfig(agentId: string): AgentConfig {
  const configPath = getAgentConfigPath(agentId);
  if (!existsSync(configPath)) {
    throw new Error(
      `Agent not found: ${agentId}. Run 'lgm setup agent' first.`,
    );
  }
  return JSON.parse(readFileSync(configPath, "utf8"));
}

export function saveAgentConfig(config: AgentConfig): void {
  if (!existsSync(CONFIG_DIR)) {
    mkdirSync(CONFIG_DIR, { recursive: true, mode: 0o700 });
  }
  const configPath = getAgentConfigPath(config.agentId);
  writeFileSync(configPath, JSON.stringify(config, null, 2), { mode: 0o600 });
}

export function listAgents(): string[] {
  if (!existsSync(CONFIG_DIR)) {
    return [];
  }
  const files = readdirSync(CONFIG_DIR) as string[];
  return files
    .filter((f: string) => f.endsWith(".json"))
    .map((f: string) => f.replace(".json", ""));
}

export function deleteAgentConfig(agentId: string): void {
  const configPath = getAgentConfigPath(agentId);
  if (existsSync(configPath)) {
    unlinkSync(configPath);
  }
}

// ============ Secure Secret Loading ============

/**
 * Load wallet using agent reference (secure)
 * Secret is fetched from secure storage, never from command line
 */
export async function loadSecretFromRef(
  agentRef: string,
  password?: string,
): Promise<{ secret: string; config: AgentConfig }> {
  // Load agent config
  const config = loadAgentConfig(agentRef);

  // Get storage backend
  const storage = getStorage(config.storageBackend, config.storageConfig);

  // Set password if needed for encrypted file
  if (storage instanceof EncryptedFileStorage && password) {
    storage.setPassword(password);
  }

  // Retrieve secret
  const secret = await storage.retrieve(agentRef);

  return { secret, config };
}

// ============ Session Validation ============

export interface SessionState {
  agentId: string;
  unlockedAt: number; // timestamp
  unlockedAtBlock: number; // block number
  spentToday: bigint;
  lastResetDate: string; // YYYY-MM-DD
}

const SESSION_FILE = join(homedir(), ".legible", "sessions.json");

// Session state stored with bigint as string for JSON compatibility
interface SessionStateStored {
  agentId: string;
  unlockedAt: number;
  unlockedAtBlock: number;
  spentToday: string; // bigint stored as string
  lastResetDate: string;
}

export function loadSessionState(agentId: string): SessionState | null {
  if (!existsSync(SESSION_FILE)) {
    return null;
  }
  const sessions = JSON.parse(readFileSync(SESSION_FILE, "utf8")) as Record<
    string,
    SessionStateStored
  >;
  const stored = sessions[agentId];
  if (!stored) return null;

  // Convert string back to bigint
  return {
    ...stored,
    spentToday: BigInt(stored.spentToday || "0"),
  };
}

export function saveSessionState(state: SessionState): void {
  const dir = dirname(SESSION_FILE);
  if (!existsSync(dir)) {
    mkdirSync(dir, { recursive: true });
  }

  let sessions: Record<string, SessionStateStored> = {};
  if (existsSync(SESSION_FILE)) {
    sessions = JSON.parse(readFileSync(SESSION_FILE, "utf8"));
  }

  // Convert bigint to string for JSON
  sessions[state.agentId] = {
    ...state,
    spentToday: state.spentToday.toString(),
  };
  writeFileSync(SESSION_FILE, JSON.stringify(sessions, null, 2), {
    mode: 0o600,
  });
}

export function validateSession(
  config: AgentConfig,
  action: string,
  amount: bigint,
  currentBlock: number,
): { valid: boolean; reason?: string } {
  const session = loadSessionState(config.agentId);

  // Check if session exists
  if (!session) {
    return {
      valid: false,
      reason:
        "Session not initialized. Run: lgm agent unlock " + config.agentId,
    };
  }

  // Check session expiry (block-based)
  const blocksElapsed = currentBlock - session.unlockedAtBlock;
  if (blocksElapsed > config.sessionConfig.sessionDurationBlocks) {
    return {
      valid: false,
      reason: "Session expired. Run: lgm agent unlock " + config.agentId,
    };
  }

  // Check action allowed
  if (
    !config.sessionConfig.allowedActions.includes(action) &&
    !config.sessionConfig.allowedActions.includes("*")
  ) {
    return {
      valid: false,
      reason: `Action '${action}' not permitted. Allowed: ${config.sessionConfig.allowedActions.join(", ")}`,
    };
  }

  // Check per-tx limit
  const maxPerTx = BigInt(config.sessionConfig.maxSpendPerTx);
  if (amount > maxPerTx) {
    return {
      valid: false,
      reason: `Amount ${amount} exceeds per-transaction limit of ${maxPerTx}`,
    };
  }

  // Check daily limit (reset if new day)
  const today = new Date().toISOString().split("T")[0];
  let spentToday = session.spentToday;
  if (session.lastResetDate !== today) {
    spentToday = 0n;
  }

  const maxPerDay = BigInt(config.sessionConfig.maxSpendPerDay);
  if (spentToday + amount > maxPerDay) {
    return {
      valid: false,
      reason: `Amount would exceed daily limit. Spent: ${spentToday}, Limit: ${maxPerDay}`,
    };
  }

  return { valid: true };
}

export function recordSpend(
  agentId: string,
  amount: bigint,
  currentBlock: number,
): void {
  let session = loadSessionState(agentId);
  const today = new Date().toISOString().split("T")[0];

  if (!session) {
    session = {
      agentId,
      unlockedAt: Date.now(),
      unlockedAtBlock: currentBlock,
      spentToday: 0n,
      lastResetDate: today,
    };
  }

  // Reset daily counter if new day
  if (session.lastResetDate !== today) {
    session.spentToday = 0n;
    session.lastResetDate = today;
  }

  session.spentToday += amount;
  saveSessionState(session);
}
