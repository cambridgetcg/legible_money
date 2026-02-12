/**
 * Session Manager
 *
 * Manages session keys and their lifecycle.
 * Handles capability enforcement and usage tracking.
 */

import { SessionWallet, type SessionCapabilities } from "./wallet.js";
import type { LegibleClient } from "./client.js";

// ============ Session Manager ============

/**
 * Manages multiple sessions for an agent
 */
export class SessionManager {
  private sessions: Map<string, ManagedSession> = new Map();
  private client: LegibleClient;

  constructor(client: LegibleClient) {
    this.client = client;
  }

  /**
   * Create a new session with specific capabilities
   */
  async createSession(
    name: string,
    capabilities: SessionCapabilities,
  ): Promise<ManagedSession> {
    const wallet = await this.client.createSession(capabilities);

    const session: ManagedSession = {
      name,
      wallet,
      capabilities,
      createdAt: Date.now(),
      lastUsed: Date.now(),
      usageCount: 0,
    };

    this.sessions.set(name, session);
    return session;
  }

  /**
   * Get session by name
   */
  getSession(name: string): ManagedSession | undefined {
    return this.sessions.get(name);
  }

  /**
   * Get the best session for a specific action
   */
  getSessionForAction(action: string, amount?: bigint): ManagedSession | null {
    for (const session of this.sessions.values()) {
      const validation = session.wallet.canPerform(action, amount);
      if (validation.valid) {
        return session;
      }
    }
    return null;
  }

  /**
   * Use a session for an operation
   */
  async useSession<T>(
    sessionName: string,
    action: string,
    operation: (wallet: SessionWallet) => Promise<T>,
    amount?: bigint,
  ): Promise<T> {
    const session = this.sessions.get(sessionName);
    if (!session) {
      throw new Error(`Session '${sessionName}' not found`);
    }

    // Validate capability
    const currentBlock = Number(await this.client.getBlockNumber());
    const validation = session.wallet.canPerform(action, amount, currentBlock);
    if (!validation.valid) {
      throw new Error(`Session cannot perform action: ${validation.reason}`);
    }

    // Execute operation
    const result = await operation(session.wallet);

    // Record usage
    session.wallet.recordUsage(amount || 0n, currentBlock);
    session.lastUsed = Date.now();
    session.usageCount++;

    return result;
  }

  /**
   * List all sessions
   */
  listSessions(): SessionInfo[] {
    return Array.from(this.sessions.values()).map((session) => ({
      name: session.name,
      publicKey: session.wallet.publicKey,
      capabilities: session.capabilities,
      createdAt: session.createdAt,
      lastUsed: session.lastUsed,
      usageCount: session.usageCount,
      remainingLimits: session.wallet.getRemainingLimits(),
    }));
  }

  /**
   * Remove a session
   */
  removeSession(name: string): boolean {
    return this.sessions.delete(name);
  }

  /**
   * Clean up expired sessions
   */
  async cleanupExpired(): Promise<number> {
    const currentBlock = Number(await this.client.getBlockNumber());
    let removed = 0;

    for (const [name, session] of this.sessions) {
      if (!session.wallet.isValid(currentBlock)) {
        this.sessions.delete(name);
        removed++;
      }
    }

    return removed;
  }
}

// ============ Preset Session Configurations ============

/**
 * Common session configurations for different use cases
 */
export const SESSION_PRESETS = {
  /**
   * Minimal session for read-only operations
   */
  readonly: {
    maxSpendPerTx: 0n,
    maxSpendPerDay: 0n,
    allowedActions: ["account_*", "knowledge_get*", "block_*", "network_*"],
    maxUsage: 10000,
  } as SessionCapabilities,

  /**
   * Session for submitting claims
   */
  claimSubmitter: {
    maxSpendPerTx: 100n * 10n ** 15n, // 0.1 LGM per claim
    maxSpendPerDay: 1000n * 10n ** 15n, // 1 LGM per day
    allowedActions: ["knowledge_submitClaim", "account_*", "knowledge_get*"],
    maxUsage: 100,
  } as SessionCapabilities,

  /**
   * Session for small transfers
   */
  micropayments: {
    maxSpendPerTx: 10n * 10n ** 15n, // 0.01 LGM per tx
    maxSpendPerDay: 100n * 10n ** 15n, // 0.1 LGM per day
    allowedActions: ["transfer", "account_*"],
    maxUsage: 1000,
  } as SessionCapabilities,

  /**
   * Session for validators
   */
  validator: {
    maxSpendPerTx: 0n, // Validators don't spend in verification
    maxSpendPerDay: 0n,
    allowedActions: [
      "verification_*",
      "account_*",
      "knowledge_*",
      "validator_*",
    ],
    maxUsage: 100000,
  } as SessionCapabilities,

  /**
   * General purpose session
   */
  general: {
    maxSpendPerTx: 100n * 10n ** 15n,
    maxSpendPerDay: 1000n * 10n ** 15n,
    allowedActions: ["transfer", "knowledge_*", "account_*"],
    maxUsage: 1000,
  } as SessionCapabilities,
};

// ============ Types ============

export interface ManagedSession {
  name: string;
  wallet: SessionWallet;
  capabilities: SessionCapabilities;
  createdAt: number;
  lastUsed: number;
  usageCount: number;
}

export interface SessionInfo {
  name: string;
  publicKey: string;
  capabilities: SessionCapabilities;
  createdAt: number;
  lastUsed: number;
  usageCount: number;
  remainingLimits: {
    remainingUsage: number;
    remainingDailySpend: bigint;
    blocksUntilExpiry: number;
  };
}

// ============ Session Factory ============

/**
 * Create a session manager for a client
 */
export function createSessionManager(client: LegibleClient): SessionManager {
  return new SessionManager(client);
}

/**
 * Quick session creation with preset
 */
export async function quickSession(
  client: LegibleClient,
  preset: keyof typeof SESSION_PRESETS,
): Promise<SessionWallet> {
  return client.createSession(SESSION_PRESETS[preset]);
}
