/**
 * Session Manager
 *
 * Manages validator session key lifecycle: derivation from operational key,
 * on-chain registration, expiry monitoring, and rotation.
 *
 * Uses deriveSessionKey() from core/agents/identity/session_keys.ts for
 * deterministic key derivation and the validator preset from
 * protocol/sdk/session.ts for capabilities.
 */

import { readFileSync } from "node:fs";
import { sign } from "../../core/crypto/signatures.js";
import { deriveSessionKey } from "../../core/agents/identity/session_keys.js";
import type { SessionKey } from "../../core/agents/identity/session_keys.js";
import type { PrivateKey, PublicKey } from "../../core/crypto/keys.js";
import type { RPCClient } from "./rpc_client.js";

/** Rotate session key when within this many blocks of expiry */
const ROTATION_THRESHOLD_BLOCKS = 200n;

/** Session duration: ~6 hours in ms */
const SESSION_DURATION_MS = 6 * 60 * 60 * 1000;

/** Approximate block time in ms */
const BLOCK_TIME_MS = 2521;

export class SessionManager {
  private readonly rpc: RPCClient;
  private readonly agentDid: string;
  private operationalPrivateKey: PrivateKey;
  private operationalPublicKey: PublicKey;
  private sessionKey: SessionKey | null = null;
  private sessionCounter = 0;
  private currentHeight = 0n;

  constructor(rpc: RPCClient, agentDid: string, operationalKeyPath: string) {
    this.rpc = rpc;
    this.agentDid = agentDid;

    // Load operational key (pattern from devnet/cmd/node.ts:545-559)
    const keyData = JSON.parse(readFileSync(operationalKeyPath, "utf-8"));
    this.operationalPrivateKey = keyData.privateKey as PrivateKey;
    this.operationalPublicKey = keyData.publicKey as PublicKey;
  }

  /** Get the current session key, creating one if needed */
  async getSessionKey(height: bigint): Promise<SessionKey> {
    this.currentHeight = height;

    if (!this.sessionKey || this.shouldRotate(height)) {
      await this.rotate();
    }

    return this.sessionKey!;
  }

  /** Sign data with the current session key */
  async signWithSession(data: Uint8Array): Promise<string> {
    const key = await this.getSessionKey(this.currentHeight);
    return sign(data, key.privateKey);
  }

  /** Sign a transaction payload with the session key */
  async signTransaction(payload: Record<string, unknown>): Promise<string> {
    const encoded = new TextEncoder().encode(JSON.stringify(payload));
    return this.signWithSession(encoded);
  }

  /** Get the operational public key */
  getOperationalPublicKey(): PublicKey {
    return this.operationalPublicKey;
  }

  /** Get the current session public key */
  getSessionPublicKey(): PublicKey | null {
    return this.sessionKey?.publicKey ?? null;
  }

  /** Update current height (call each block) */
  updateHeight(height: bigint): void {
    this.currentHeight = height;
  }

  /** Check if session key should be rotated */
  private shouldRotate(height: bigint): boolean {
    if (!this.sessionKey) return true;

    // Time-based expiry check
    if (Date.now() >= this.sessionKey.expiresAt) return true;

    // Block-based expiry check (if set)
    if (this.sessionKey.expiresAtBlock !== undefined) {
      const remaining = this.sessionKey.expiresAtBlock - height;
      if (remaining <= ROTATION_THRESHOLD_BLOCKS) return true;
    }

    // Approaching time-based expiry (200 blocks * 2521ms ≈ 8.4 minutes)
    const timeRemaining = this.sessionKey.expiresAt - Date.now();
    if (timeRemaining <= Number(ROTATION_THRESHOLD_BLOCKS) * BLOCK_TIME_MS)
      return true;

    return false;
  }

  /** Derive and register a new session key */
  private async rotate(): Promise<void> {
    this.sessionCounter++;
    const sessionSeed = `validator-daemon:${this.agentDid}:${this.sessionCounter}`;

    // Deterministic derivation (core/agents/identity/session_keys.ts:228-260)
    this.sessionKey = deriveSessionKey(
      this.operationalPrivateKey,
      sessionSeed,
      {
        maxPerTransaction: 0n, // Validators don't spend
        maxPerDay: 0n,
        maxTotal: 0n,
        maxTransactions: 100000,
        maxDurationMs: SESSION_DURATION_MS,
        allowedActions: ["verify_claim", "query"],
        allowedRecipients: [],
        requireIntentProof: false,
      },
      SESSION_DURATION_MS,
    );

    // Set block-based expiry
    const expiryBlocks = BigInt(
      Math.floor(SESSION_DURATION_MS / BLOCK_TIME_MS),
    );
    this.sessionKey.createdAtBlock = this.currentHeight;
    this.sessionKey.expiresAtBlock = this.currentHeight + expiryBlocks;

    // Register on-chain
    const registrationPayload = {
      did: this.agentDid,
      sessionPublicKey: this.sessionKey.publicKey,
      capabilities: {
        allowedActions: [
          "verification_*",
          "account_*",
          "knowledge_*",
          "validator_*",
        ],
        maxUsage: 100000,
        expiresInBlocks: Number(expiryBlocks),
      },
    };

    const signature = sign(
      new TextEncoder().encode(JSON.stringify(registrationPayload)),
      this.operationalPrivateKey,
    );

    await this.rpc.call("session_create", {
      ...registrationPayload,
      operationalSignature: signature,
    });
  }
}
