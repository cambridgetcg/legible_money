/**
 * Agent Wallet
 *
 * Manages keys for agent participation.
 * Supports deterministic derivation (agents can "remember" keys).
 */

import {
  generateKeypair,
  generateKeypairFromSeed,
  type PublicKey,
  type PrivateKey,
  hexToBytes,
} from "../../core/crypto/keys.js";
import { sign, verify } from "../../core/crypto/signatures.js";
import { domainHash, sha512_256Hex } from "../../core/crypto/hashing.js";
import { deriveOperationalKey } from "../../core/agents/identity/account.js";
import { publicKeyToDid } from "../../core/agents/identity/did.js";
import type { Wallet } from "../types.js";

// ============ Agent Wallet ============

/**
 * Wallet for AI agents
 * Keys derived deterministically from agent ID + secret
 */
export class AgentWallet implements Wallet {
  readonly publicKey: PublicKey;
  readonly did: string;
  private readonly privateKey: PrivateKey;

  private constructor(
    publicKey: PublicKey,
    privateKey: PrivateKey,
    did: string,
  ) {
    this.publicKey = publicKey;
    this.privateKey = privateKey;
    this.did = did;
  }

  /**
   * Create wallet from agent ID and secret
   * Deterministic: same inputs = same keys
   */
  static fromAgentId(
    agentId: string,
    secret: string,
    network: "mainnet" | "testnet" = "mainnet",
  ): AgentWallet {
    // Delegate to canonical key derivation path (same as deriveOperationalKey in account.ts)
    const { publicKey, privateKey } = deriveOperationalKey(agentId, secret);

    // Derive DID from public key
    const did = deriveDID(publicKey, network);

    return new AgentWallet(publicKey, privateKey, did);
  }

  /**
   * Create wallet from private key directly
   */
  static fromPrivateKey(privateKey: PrivateKey): AgentWallet {
    // Derive public key from private key
    const { publicKey } = generateKeypairFromSeed(
      hexToBytes(privateKey).slice(0, 32),
    );
    const did = deriveDID(publicKey, "mainnet");

    return new AgentWallet(publicKey, privateKey, did);
  }

  /**
   * Create wallet from mnemonic/seed phrase
   * (For human users who prefer mnemonic)
   */
  static fromMnemonic(
    mnemonic: string,
    network: "mainnet" | "testnet" = "mainnet",
  ): AgentWallet {
    const domain = `LGM.${network}.mnemonic.key.v1`;
    const seed = domainHash(domain, mnemonic);
    const { publicKey, privateKey } = generateKeypairFromSeed(
      seed.slice(0, 32),
    );
    const did = deriveDID(publicKey, network);

    return new AgentWallet(publicKey, privateKey, did);
  }

  /**
   * Sign a message
   */
  async sign(message: Uint8Array): Promise<string> {
    return sign(message, this.privateKey);
  }

  /**
   * Sign a hex-encoded message
   */
  async signHex(message: string): Promise<string> {
    return sign(hexToBytes(message), this.privateKey);
  }

  /**
   * Sign a transaction payload (legacy — signs payload only)
   * @deprecated Use signFullTransaction() which covers type+nonce+fee+payload
   */
  async signTransaction(payload: object): Promise<string> {
    const message = JSON.stringify(payload);
    return sign(new TextEncoder().encode(message), this.privateKey);
  }

  /**
   * Sign a full transaction covering type+nonce+fee+payload.
   * ZD-E01 FIX: Prevents type/nonce/fee mutation attacks.
   */
  async signFullTransaction(tx: {
    type: string;
    nonce: bigint;
    fee: bigint;
    payload: unknown;
  }): Promise<string> {
    const message = JSON.stringify(
      {
        type: tx.type,
        nonce: tx.nonce.toString(),
        fee: tx.fee.toString(),
        payload: tx.payload,
      },
      (_key, value) => (typeof value === "bigint" ? value.toString() : value),
    );
    return sign(new TextEncoder().encode(message), this.privateKey);
  }

  /**
   * Verify a signature (static helper)
   */
  static verify(
    message: Uint8Array,
    signature: string,
    publicKey: PublicKey,
  ): boolean {
    return verify(message, signature, publicKey);
  }

  /**
   * Build a signed key rotation request.
   * Generates a new operational keypair and signs the rotation with the current key.
   * Returns the RPC request payload and the new wallet instance.
   */
  async buildRotateKeyRequest(nonce: bigint): Promise<KeyRotationBundle> {
    // Generate new operational keypair
    const newKeypair = generateKeypair();
    const newKeyHash = sha512_256Hex(newKeypair.publicKey);

    // Build and sign the rotation payload (must match RPC handler expectation)
    const payload = JSON.stringify({
      did: this.did,
      newOperationalKeyHash: newKeyHash,
      action: "rotate_key",
      nonce: nonce.toString(),
    });
    const signature = sign(new TextEncoder().encode(payload), this.privateKey);

    // Build the new wallet with the rotated key
    const newWallet = new AgentWallet(
      newKeypair.publicKey,
      newKeypair.privateKey,
      this.did,
    );

    return {
      request: {
        did: this.did,
        newOperationalKeyHash: newKeyHash,
        nonce: nonce.toString(),
        signature,
        sender: this.publicKey,
      },
      newWallet,
      newPublicKey: newKeypair.publicKey,
    };
  }

  /**
   * Get recovery information (for backup)
   * WARNING: Contains private key material
   */
  getRecoveryInfo(): WalletRecoveryInfo {
    return {
      publicKey: this.publicKey,
      privateKey: this.privateKey,
      did: this.did,
      warning: "NEVER SHARE THIS INFORMATION",
    };
  }
}

// ============ Session Wallet ============

/**
 * Lightweight wallet for session operations
 * Limited capabilities, safe to use in automated systems
 */
export class SessionWallet implements Wallet {
  readonly publicKey: PublicKey;
  readonly did: string;
  readonly sessionId: string;
  readonly capabilities: SessionCapabilities;
  readonly expiresAt: number;

  private readonly privateKey: PrivateKey;
  private usageCount: number = 0;
  private dailySpent: bigint = 0n;
  private lastResetBlock: number = 0;

  constructor(
    publicKey: PublicKey,
    privateKey: PrivateKey,
    did: string,
    sessionId: string,
    capabilities: SessionCapabilities,
    expiresAt: number,
  ) {
    this.publicKey = publicKey;
    this.privateKey = privateKey;
    this.did = did;
    this.sessionId = sessionId;
    this.capabilities = capabilities;
    this.expiresAt = expiresAt;
  }

  /**
   * Create a new session wallet
   */
  static create(
    parentDid: string,
    sessionId: string,
    capabilities: SessionCapabilities,
    expiresAt: number,
  ): SessionWallet {
    const { publicKey, privateKey } = generateKeypair();
    return new SessionWallet(
      publicKey,
      privateKey,
      parentDid,
      sessionId,
      capabilities,
      expiresAt,
    );
  }

  /**
   * Derive session wallet deterministically
   */
  static derive(
    parentPrivateKey: PrivateKey,
    parentDid: string,
    sessionId: string,
    capabilities: SessionCapabilities,
    expiresAt: number,
  ): SessionWallet {
    const seed = domainHash(
      "LGM.session.derive.v1",
      `${parentDid}:${sessionId}`,
    );
    const sig = sign(seed, parentPrivateKey);
    const { publicKey, privateKey } = generateKeypairFromSeed(
      hexToBytes(sig).slice(0, 32),
    );

    return new SessionWallet(
      publicKey,
      privateKey,
      parentDid,
      sessionId,
      capabilities,
      expiresAt,
    );
  }

  /**
   * Check if session is still valid
   */
  isValid(currentBlock: number): boolean {
    return currentBlock < this.expiresAt;
  }

  /**
   * Check if action is allowed
   */
  canPerform(
    action: string,
    amount?: bigint,
    currentBlock?: number,
  ): ValidationResult {
    // Check expiry
    if (currentBlock !== undefined && currentBlock >= this.expiresAt) {
      return { valid: false, reason: "Session expired" };
    }

    // Check usage count
    if (
      this.capabilities.maxUsage &&
      this.usageCount >= this.capabilities.maxUsage
    ) {
      return { valid: false, reason: "Usage limit reached" };
    }

    // Check action whitelist
    if (!this.isActionAllowed(action)) {
      return { valid: false, reason: `Action '${action}' not allowed` };
    }

    // Check amount limits
    if (amount !== undefined) {
      if (amount > this.capabilities.maxSpendPerTx) {
        return { valid: false, reason: "Exceeds per-transaction limit" };
      }

      if (this.dailySpent + amount > this.capabilities.maxSpendPerDay) {
        return { valid: false, reason: "Exceeds daily limit" };
      }
    }

    return { valid: true };
  }

  /**
   * Record usage (call after successful operation)
   */
  recordUsage(amount: bigint = 0n, currentBlock: number): void {
    // Reset daily counter if needed
    const blocksPerDay = Math.floor((24 * 60 * 60 * 1000) / 2521); // ~34,000 blocks
    if (currentBlock - this.lastResetBlock > blocksPerDay) {
      this.dailySpent = 0n;
      this.lastResetBlock = currentBlock;
    }

    this.usageCount++;
    this.dailySpent += amount;
  }

  /**
   * Get remaining limits
   */
  getRemainingLimits(): RemainingLimits {
    return {
      remainingUsage: this.capabilities.maxUsage
        ? this.capabilities.maxUsage - this.usageCount
        : Infinity,
      remainingDailySpend: this.capabilities.maxSpendPerDay - this.dailySpent,
      blocksUntilExpiry: this.expiresAt,
    };
  }

  async sign(message: Uint8Array): Promise<string> {
    return sign(message, this.privateKey);
  }

  async signHex(message: string): Promise<string> {
    return sign(hexToBytes(message), this.privateKey);
  }

  private isActionAllowed(action: string): boolean {
    const allowed = this.capabilities.allowedActions;
    return allowed.some((pattern) => {
      if (pattern.endsWith("*")) {
        return action.startsWith(pattern.slice(0, -1));
      }
      return action === pattern;
    });
  }
}

// ============ Helper Functions ============

/**
 * Derive DID from public key using the canonical on-chain derivation.
 */
function deriveDID(publicKey: PublicKey, _network: string): string {
  return publicKeyToDid(publicKey);
}

// ============ Types ============

export interface SessionCapabilities {
  maxSpendPerTx: bigint;
  maxSpendPerDay: bigint;
  allowedActions: string[];
  maxUsage?: number;
}

export interface ValidationResult {
  valid: boolean;
  reason?: string;
}

export interface RemainingLimits {
  remainingUsage: number;
  remainingDailySpend: bigint;
  blocksUntilExpiry: number;
}

export interface WalletRecoveryInfo {
  publicKey: PublicKey;
  privateKey: PrivateKey;
  did: string;
  warning: string;
}

export interface KeyRotationBundle {
  request: {
    did: string;
    newOperationalKeyHash: string;
    nonce: string;
    signature: string;
    sender: PublicKey;
  };
  newWallet: AgentWallet;
  newPublicKey: PublicKey;
}

// ============ Exports ============

export { deriveDID };
