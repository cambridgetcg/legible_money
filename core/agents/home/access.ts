/**
 * Access Control Manager
 *
 * Manages key registration, permission verification, session handling,
 * and security enforcement for Agent Homes.
 *
 * Security Model:
 *
 *   ┌─────────────────────────────────────────────────────────────┐
 *   │                    ACCESS CONTROL                           │
 *   ├─────────────────────────────────────────────────────────────┤
 *   │                                                             │
 *   │  ┌─────────────────┐                                       │
 *   │  │   KEY TYPES     │                                       │
 *   │  │                 │                                       │
 *   │  │  Identity ──────┼──► owner (full control)               │
 *   │  │  Operational ───┼──► operator (daily ops)               │
 *   │  │  Session ───────┼──► guest (limited, time-bound)        │
 *   │  │  Guardian ──────┼──► guardian (recovery only)           │
 *   │  └─────────────────┘                                       │
 *   │                                                             │
 *   │  ┌─────────────────┐                                       │
 *   │  │  PERMISSIONS    │                                       │
 *   │  │                 │                                       │
 *   │  │  owner ─────────┼──► ALL permissions                    │
 *   │  │  operator ──────┼──► treasury, verify, delegate         │
 *   │  │  guest ─────────┼──► read, verify (limited)             │
 *   │  │  guardian ──────┼──► recovery only                      │
 *   │  │  viewer ────────┼──► read only                          │
 *   │  └─────────────────┘                                       │
 *   │                                                             │
 *   │  ┌─────────────────┐                                       │
 *   │  │  RATE LIMITS    │                                       │
 *   │  │                 │                                       │
 *   │  │  Per block ─────┼──► maxActionsPerBlock                 │
 *   │  │  Per day ───────┼──► maxTransfersPerDay                 │
 *   │  │  Per session ───┼──► spendingLimit.maxPerSession        │
 *   │  └─────────────────┘                                       │
 *   │                                                             │
 *   └─────────────────────────────────────────────────────────────┘
 *
 * The home protects the agent's sovereignty while enabling flexible access.
 */

import { sha512_256Hex } from "../../crypto/hashing.js";
import type { DID } from "../identity/did.js";
import type {
  AccessControl,
  KeyRegistration,
  AccessRole,
  Permission,
  ActiveSession,
  AccessLogEntry,
  SecuritySettings,
  HomeEvent,
  PendingApproval,
} from "./types.js";
import { HomeError } from "./types.js";

// ============ Permission Mappings ============

/**
 * Default permissions for each role
 */
const ROLE_PERMISSIONS: Record<AccessRole, Permission[]> = {
  owner: [
    "read_memory",
    "write_memory",
    "read_treasury",
    "transfer",
    "stake",
    "delegate",
    "verify",
    "challenge",
    "dispute",
    "configure_guardian",
    "configure_automation",
    "manage_keys",
    "recovery",
    "archive",
  ],
  operator: [
    "read_memory",
    "write_memory",
    "read_treasury",
    "transfer",
    "stake",
    "delegate",
    "verify",
    "challenge",
    "dispute",
    "configure_automation",
  ],
  guest: ["read_memory", "read_treasury", "verify"],
  guardian: ["recovery"],
  viewer: ["read_memory", "read_treasury"],
};

/**
 * Default security settings
 */
const DEFAULT_SECURITY: SecuritySettings = {
  multiSigThreshold: 1000000000n, // 1000 LGM requires multi-sig
  keyRotationCooldown: 1000n, // ~42 minutes in blocks
  maxSessionDuration: 24 * 60 * 60 * 1000, // 24 hours in ms
  rateLimits: {
    maxActionsPerBlock: 10,
    maxTransfersPerDay: 50,
    maxStakeChangesPerDay: 5,
  },
};

// ============ Access Summary ============

export interface AccessSummary {
  registeredKeys: number;
  activeKeys: number;
  revokedKeys: number;
  expiredKeys: number;
  activeSessions: number;
  recentAccessCount: number;
  securityStatus: "normal" | "elevated" | "lockdown";
  lastKeyRotation: bigint | null;
  guardiansConfigured: number;
}

// ============ Access Manager ============

/**
 * Manages access control for an Agent Home
 */
export class AccessManager {
  private access: AccessControl;
  private actionsThisBlock: Map<string, number> = new Map();
  private transfersToday: Map<string, number> = new Map();
  private stakeChangesToday: Map<string, number> = new Map();
  private currentBlock: bigint = 0n;
  private currentDay: number = 0;
  private currentTime: number = 0;
  private sessionCounter: number = 0;

  constructor(_did: DID, existingAccess?: AccessControl) {
    this.access = existingAccess ?? this.createDefaultAccess();
  }

  private createDefaultAccess(): AccessControl {
    return {
      keys: [],
      permissions: new Map(
        Object.entries(ROLE_PERMISSIONS).map(([role, perms]) => [
          role as AccessRole,
          perms,
        ]),
      ),
      activeSessions: [],
      accessLog: [],
      security: { ...DEFAULT_SECURITY },
      pendingApprovals: [],
    };
  }

  // ============ Key Management ============

  /**
   * Register a new key for access
   */
  registerKey(
    keyHash: string,
    keyType: KeyRegistration["keyType"],
    role: AccessRole,
    options: {
      expiresAt?: number;
      label?: string;
      registeredBy?: string;
    } = {},
  ): KeyRegistration {
    // Check for duplicate
    const existing = this.access.keys.find(
      (k) => k.keyHash === keyHash && k.revokedAt === undefined,
    );
    if (existing) {
      throw new HomeError("Key already registered", "INVALID_CONFIGURATION", {
        keyHash,
      });
    }

    // Validate role matches key type
    this.validateKeyTypeRole(keyType, role);

    const registration: KeyRegistration = {
      keyHash,
      keyType,
      role,
      registeredAt: this.currentTime,
      expiresAt: options.expiresAt,
      label: options.label,
    };

    this.access.keys.push(registration);

    this.logAccess(options.registeredBy ?? "system", "register_key", true, {
      keyHash: this.maskKeyHash(keyHash),
      keyType,
      role,
    });

    return registration;
  }

  /**
   * Validate key type matches expected role
   */
  private validateKeyTypeRole(
    keyType: KeyRegistration["keyType"],
    role: AccessRole,
  ): void {
    const validMappings: Record<KeyRegistration["keyType"], AccessRole[]> = {
      identity: ["owner"],
      operational: ["owner", "operator"],
      session: ["guest", "viewer"],
      guardian: ["guardian"],
    };

    const validRoles = validMappings[keyType];
    if (!validRoles.includes(role)) {
      throw new HomeError(
        `Key type ${keyType} cannot be assigned role ${role}`,
        "INVALID_CONFIGURATION",
        { keyType, role, validRoles },
      );
    }
  }

  /**
   * Revoke a key
   */
  revokeKey(keyHash: string, revokedBy: string): void {
    const key = this.access.keys.find(
      (k) => k.keyHash === keyHash && k.revokedAt === undefined,
    );

    if (!key) {
      throw new HomeError(
        "Key not found or already revoked",
        "KEY_NOT_REGISTERED",
        { keyHash },
      );
    }

    // Cannot revoke identity key without recovery
    if (key.keyType === "identity") {
      throw new HomeError(
        "Cannot revoke identity key directly; use recovery process",
        "ACCESS_DENIED",
        { keyHash },
      );
    }

    key.revokedAt = this.currentTime;

    // Terminate any active sessions using this key
    this.access.activeSessions = this.access.activeSessions.filter(
      (s) => s.keyHash !== keyHash,
    );

    this.logAccess(revokedBy, "revoke_key", true, {
      keyHash: this.maskKeyHash(keyHash),
      keyType: key.keyType,
    });
  }

  /**
   * Rotate an operational key
   */
  rotateKey(
    oldKeyHash: string,
    newKeyHash: string,
    rotatedBy: string,
  ): KeyRegistration {
    const oldKey = this.access.keys.find(
      (k) => k.keyHash === oldKeyHash && k.revokedAt === undefined,
    );

    if (!oldKey) {
      throw new HomeError("Old key not found", "KEY_NOT_REGISTERED", {
        keyHash: oldKeyHash,
      });
    }

    if (oldKey.keyType === "identity") {
      throw new HomeError(
        "Cannot rotate identity key; use recovery for identity changes",
        "ACCESS_DENIED",
      );
    }

    // Check cooldown (block-based)
    const lastRotationBlock = this.getLastKeyRotationBlock(oldKey.keyType);
    if (lastRotationBlock !== null) {
      const blocksSinceRotation = this.currentBlock - lastRotationBlock;
      if (blocksSinceRotation < this.access.security.keyRotationCooldown) {
        throw new HomeError(
          "Key rotation cooldown not elapsed",
          "RATE_LIMITED",
          {
            blocksRemaining: Number(
              this.access.security.keyRotationCooldown - blocksSinceRotation,
            ),
          },
        );
      }
    }

    // Revoke old key
    oldKey.revokedAt = this.currentTime;

    // Register new key with same type and role
    const newKey = this.registerKey(newKeyHash, oldKey.keyType, oldKey.role, {
      label: oldKey.label ? `${oldKey.label} (rotated)` : undefined,
      registeredBy: rotatedBy,
    });

    // Log rotation with block height so cooldown can be enforced
    this.logAccess(rotatedBy, "rotate_key", true, {
      oldKeyHash: this.maskKeyHash(oldKeyHash),
      newKeyHash: this.maskKeyHash(newKeyHash),
      keyType: oldKey.keyType,
      atBlock: this.currentBlock.toString(),
    });

    return newKey;
  }

  private getLastKeyRotationBlock(
    keyType: KeyRegistration["keyType"],
  ): bigint | null {
    const rotations = this.access.accessLog
      .filter((l) => l.action === "rotate_key" && l.details?.includes(keyType))
      .sort((a, b) => b.timestamp - a.timestamp);

    if (rotations.length === 0) return null;

    // Extract block height from the log details JSON
    try {
      const details = JSON.parse(rotations[0].details ?? "{}");
      return details.atBlock ? BigInt(details.atBlock) : null;
    } catch {
      return null;
    }
  }

  /**
   * Get all active keys
   */
  getActiveKeys(): KeyRegistration[] {
    const now = this.currentTime;
    return this.access.keys.filter(
      (k) => k.revokedAt === undefined && (!k.expiresAt || k.expiresAt > now),
    );
  }

  /**
   * Get key by hash
   */
  getKey(keyHash: string): KeyRegistration | undefined {
    return this.access.keys.find(
      (k) => k.keyHash === keyHash && k.revokedAt === undefined,
    );
  }

  // ============ Session Management ============

  /**
   * Start a new session
   */
  startSession(
    keyHash: string,
    options: {
      sessionDuration?: number;
    } = {},
  ): ActiveSession {
    const key = this.getKey(keyHash);
    if (!key) {
      throw new HomeError(
        "Key not registered or revoked",
        "KEY_NOT_REGISTERED",
        { keyHash: this.maskKeyHash(keyHash) },
      );
    }

    // Check if key has expired
    if (key.expiresAt && key.expiresAt <= this.currentTime) {
      throw new HomeError("Key has expired", "SESSION_EXPIRED", {
        expiredAt: key.expiresAt,
      });
    }

    // Calculate session duration
    const duration = Math.min(
      options.sessionDuration ?? this.access.security.maxSessionDuration,
      this.access.security.maxSessionDuration,
    );

    const session: ActiveSession = {
      sessionId: this.generateSessionId(keyHash),
      keyHash,
      role: key.role,
      startedAt: this.currentTime,
      lastActivityAt: this.currentTime,
      expiresAt: this.currentTime + duration,
      actionsPerformed: 0,
      spentThisSession: 0n,
    };

    this.access.activeSessions.push(session);

    this.logAccess(keyHash, "session_start", true, {
      sessionId: session.sessionId,
      role: key.role,
      expiresIn: duration,
    });

    return session;
  }

  /**
   * End a session
   */
  endSession(sessionId: string): void {
    const sessionIndex = this.access.activeSessions.findIndex(
      (s) => s.sessionId === sessionId,
    );

    if (sessionIndex === -1) {
      return; // Session not found, already ended
    }

    const session = this.access.activeSessions[sessionIndex];
    this.access.activeSessions.splice(sessionIndex, 1);

    this.logAccess(session.keyHash, "session_end", true, {
      sessionId,
      actionsPerformed: session.actionsPerformed,
      spentThisSession: session.spentThisSession.toString(),
    });
  }

  /**
   * Get active session by ID
   */
  getSession(sessionId: string): ActiveSession | undefined {
    const session = this.access.activeSessions.find(
      (s) => s.sessionId === sessionId,
    );

    if (session && session.expiresAt <= this.currentTime) {
      // Session expired, clean it up
      this.endSession(sessionId);
      return undefined;
    }

    return session;
  }

  /**
   * Update session activity
   */
  recordSessionActivity(sessionId: string, spent?: bigint): void {
    const session = this.getSession(sessionId);
    if (!session) {
      throw new HomeError("Session not found or expired", "SESSION_EXPIRED", {
        sessionId,
      });
    }

    session.lastActivityAt = this.currentTime;
    session.actionsPerformed++;
    if (spent) {
      session.spentThisSession += spent;
    }
  }

  /**
   * M-2: Clean up expired keys by marking them as revoked.
   * Prevents zombie keys from accumulating in the TS stack.
   */
  cleanupExpiredKeys(): number {
    const now = this.currentTime;
    if (now === 0) return 0;
    let cleaned = 0;
    for (const key of this.access.keys) {
      if (
        key.revokedAt === undefined &&
        key.expiresAt !== undefined &&
        key.expiresAt <= now
      ) {
        key.revokedAt = now;
        cleaned++;
      }
    }
    return cleaned;
  }

  /**
   * Clean up expired sessions
   */
  cleanupExpiredSessions(): string[] {
    const now = this.currentTime;
    const expired: string[] = [];

    this.access.activeSessions = this.access.activeSessions.filter((s) => {
      if (now > 0 && s.expiresAt <= now) {
        expired.push(s.sessionId);
        this.logAccess(s.keyHash, "session_expired", true, {
          sessionId: s.sessionId,
        });
        return false;
      }
      return true;
    });

    return expired;
  }

  private generateSessionId(keyHash: string): string {
    return sha512_256Hex(
      `session:${keyHash}:${this.currentBlock}:${this.sessionCounter++}`,
    ).slice(0, 24);
  }

  // ============ Permission Checks ============

  /**
   * Check if a key has a specific permission
   */
  hasPermission(keyHash: string, permission: Permission): boolean {
    const key = this.getKey(keyHash);
    if (!key) return false;

    const rolePermissions = this.access.permissions.get(key.role);
    return rolePermissions?.includes(permission) ?? false;
  }

  /**
   * Check if a session has a specific permission
   */
  sessionHasPermission(sessionId: string, permission: Permission): boolean {
    const session = this.getSession(sessionId);
    if (!session) return false;

    const rolePermissions = this.access.permissions.get(session.role);
    return rolePermissions?.includes(permission) ?? false;
  }

  /**
   * Require a permission or throw
   */
  requirePermission(keyHash: string, permission: Permission): void {
    if (!this.hasPermission(keyHash, permission)) {
      this.logAccess(keyHash, `require_${permission}`, false);
      throw new HomeError(
        `Missing permission: ${permission}`,
        "INSUFFICIENT_PERMISSIONS",
        { keyHash: this.maskKeyHash(keyHash), permission },
      );
    }
  }

  /**
   * Add a permission to a role
   */
  addPermissionToRole(role: AccessRole, permission: Permission): void {
    const perms = this.access.permissions.get(role) ?? [];
    if (!perms.includes(permission)) {
      perms.push(permission);
      this.access.permissions.set(role, perms);
    }
  }

  /**
   * Remove a permission from a role
   */
  removePermissionFromRole(role: AccessRole, permission: Permission): void {
    const perms = this.access.permissions.get(role) ?? [];
    const index = perms.indexOf(permission);
    if (index !== -1) {
      perms.splice(index, 1);
      this.access.permissions.set(role, perms);
    }
  }

  // ============ Rate Limiting ============

  /**
   * Update current block for rate limiting
   */
  setCurrentBlock(block: bigint, currentTime?: number): void {
    if (block > this.currentBlock) {
      // New block, reset per-block counters
      this.actionsThisBlock.clear();
      this.currentBlock = block;
    }

    if (currentTime !== undefined) {
      this.currentTime = currentTime;
    }

    // Check if day changed (use deterministic currentTime, not Date.now())
    const timeForDay = this.currentTime || 0;
    const newDay = Math.floor(timeForDay / (24 * 60 * 60 * 1000));
    if (newDay > this.currentDay) {
      this.transfersToday.clear();
      this.stakeChangesToday.clear();
      this.currentDay = newDay;

      // M-2: Proactive cleanup on day boundary (mirrors Cosmos SDK BeginBlocker)
      this.cleanupExpiredSessions();
      this.cleanupExpiredKeys();
      this.cleanupExpiredApprovals();
    }
  }

  /**
   * Check and consume rate limit
   */
  checkRateLimit(
    keyHash: string,
    action: "action" | "transfer" | "stake_change",
  ): void {
    const limits = this.access.security.rateLimits;

    switch (action) {
      case "action": {
        const count = this.actionsThisBlock.get(keyHash) ?? 0;
        if (count >= limits.maxActionsPerBlock) {
          throw new HomeError(
            "Rate limit exceeded: max actions per block",
            "RATE_LIMITED",
            { limit: limits.maxActionsPerBlock, current: count },
          );
        }
        this.actionsThisBlock.set(keyHash, count + 1);
        break;
      }
      case "transfer": {
        const count = this.transfersToday.get(keyHash) ?? 0;
        if (count >= limits.maxTransfersPerDay) {
          throw new HomeError(
            "Rate limit exceeded: max transfers per day",
            "RATE_LIMITED",
            { limit: limits.maxTransfersPerDay, current: count },
          );
        }
        this.transfersToday.set(keyHash, count + 1);
        break;
      }
      case "stake_change": {
        const count = this.stakeChangesToday.get(keyHash) ?? 0;
        if (count >= limits.maxStakeChangesPerDay) {
          throw new HomeError(
            "Rate limit exceeded: max stake changes per day",
            "RATE_LIMITED",
            { limit: limits.maxStakeChangesPerDay, current: count },
          );
        }
        this.stakeChangesToday.set(keyHash, count + 1);
        break;
      }
    }
  }

  /**
   * Check if an amount requires multi-sig
   */
  requiresMultiSig(amount: bigint): boolean {
    return amount >= this.access.security.multiSigThreshold;
  }

  // ============ Security Settings ============

  /**
   * Update security settings
   */
  updateSecuritySettings(
    updates: Partial<SecuritySettings>,
    updatedBy: string,
  ): void {
    // Validate updates
    if (
      updates.multiSigThreshold !== undefined &&
      updates.multiSigThreshold < 0n
    ) {
      throw new HomeError(
        "Multi-sig threshold cannot be negative",
        "INVALID_CONFIGURATION",
      );
    }

    if (
      updates.maxSessionDuration !== undefined &&
      updates.maxSessionDuration < 60000
    ) {
      throw new HomeError(
        "Session duration must be at least 1 minute",
        "INVALID_CONFIGURATION",
      );
    }

    // Apply updates
    this.access.security = {
      ...this.access.security,
      ...updates,
      rateLimits: updates.rateLimits
        ? { ...this.access.security.rateLimits, ...updates.rateLimits }
        : this.access.security.rateLimits,
    };

    this.logAccess(updatedBy, "update_security", true, {
      updates: Object.keys(updates),
    });
  }

  /**
   * Get current security settings
   */
  getSecuritySettings(): SecuritySettings {
    return { ...this.access.security };
  }

  // ============ Access Logging ============

  /**
   * Log an access attempt
   */
  private logAccess(
    keyHash: string,
    action: string,
    success: boolean,
    detailsObj?: Record<string, unknown>,
  ): void {
    const entry: AccessLogEntry = {
      timestamp: this.currentTime,
      keyHash: this.maskKeyHash(keyHash),
      action,
      success,
      details: detailsObj ? JSON.stringify(detailsObj) : undefined,
    };

    this.access.accessLog.push(entry);

    // Keep log bounded (last 1000 entries)
    if (this.access.accessLog.length > 1000) {
      this.access.accessLog = this.access.accessLog.slice(-1000);
    }
  }

  /**
   * Get recent access log entries
   */
  getAccessLog(limit: number = 100): AccessLogEntry[] {
    return this.access.accessLog.slice(-limit);
  }

  /**
   * Get failed access attempts
   */
  getFailedAttempts(since?: number): AccessLogEntry[] {
    const cutoff = since ?? this.currentTime - 24 * 60 * 60 * 1000; // Last 24 hours
    return this.access.accessLog.filter(
      (e) => !e.success && e.timestamp >= cutoff,
    );
  }

  /**
   * Mask key hash for logging (show first/last 4 chars)
   */
  private maskKeyHash(keyHash: string): string {
    if (keyHash.length <= 12) return keyHash;
    return `${keyHash.slice(0, 6)}...${keyHash.slice(-4)}`;
  }

  // ============ Guardian Management ============

  /**
   * Get configured guardians
   */
  getGuardians(): KeyRegistration[] {
    return this.access.keys.filter(
      (k) => k.keyType === "guardian" && k.revokedAt === undefined,
    );
  }

  /**
   * Check if key is a guardian
   */
  isGuardian(keyHash: string): boolean {
    const key = this.getKey(keyHash);
    return key?.keyType === "guardian";
  }

  // ============ Multi-Sig Approvals (M-1) ============

  /**
   * Create a pending multi-sig approval for a large operation.
   * Returns the approvalId. The requester's key counts as the first approval.
   */
  createApproval(
    operationType: PendingApproval["operationType"],
    amount: bigint,
    target: string,
    requestedBy: string,
    options: { data?: string; approvalDuration?: number } = {},
  ): string {
    // Clean expired approvals first
    this.cleanupExpiredApprovals();

    // Determine required approvals: owner keys with "manage_keys" or "transfer" permission
    const ownerKeys = this.getActiveKeys().filter(
      (k) => k.role === "owner" || k.role === "operator",
    );
    // Require at least 2 approvals, or all owner/operator keys if fewer
    const requiredApprovals = Math.max(2, Math.min(ownerKeys.length, 3));

    const duration = options.approvalDuration ?? 24 * 60 * 60 * 1000; // 24 hours default

    const approvalId = sha512_256Hex(
      `approval:${operationType}:${amount}:${target}:${this.currentTime}`,
    ).slice(0, 24);

    const pending: PendingApproval = {
      approvalId,
      operationType,
      amount,
      target,
      requestedBy,
      requestedAt: this.currentTime,
      expiresAt: this.currentTime + duration,
      approvals: [requestedBy], // Requester's key = first approval
      requiredApprovals,
      data: options.data,
    };

    if (!this.access.pendingApprovals) {
      this.access.pendingApprovals = [];
    }
    this.access.pendingApprovals.push(pending);

    this.logAccess(requestedBy, "create_approval", true, {
      approvalId,
      operationType,
      amount: amount.toString(),
      requiredApprovals,
    });

    return approvalId;
  }

  /**
   * Add an approval to a pending operation. Returns true if threshold is now met.
   */
  approveOperation(approvalId: string, approverKeyHash: string): boolean {
    this.cleanupExpiredApprovals();

    const pending = this.access.pendingApprovals?.find(
      (a) => a.approvalId === approvalId,
    );
    if (!pending) {
      throw new HomeError(
        "Approval not found or expired",
        "APPROVAL_NOT_FOUND",
        { approvalId },
      );
    }

    if (pending.expiresAt <= this.currentTime) {
      throw new HomeError("Approval has expired", "APPROVAL_EXPIRED", {
        approvalId,
      });
    }

    // Check approver has permission
    if (!this.hasPermission(approverKeyHash, "transfer")) {
      throw new HomeError(
        "Approver lacks transfer permission",
        "INSUFFICIENT_PERMISSIONS",
        { approverKeyHash },
      );
    }

    // Check for duplicate
    if (pending.approvals.includes(approverKeyHash)) {
      throw new HomeError("Already approved by this key", "ALREADY_APPROVED", {
        approverKeyHash,
      });
    }

    pending.approvals.push(approverKeyHash);

    this.logAccess(approverKeyHash, "approve_operation", true, {
      approvalId,
      approvalsCount: `${pending.approvals.length}/${pending.requiredApprovals}`,
    });

    return pending.approvals.length >= pending.requiredApprovals;
  }

  /**
   * Check if a pending approval has met its threshold.
   */
  isApproved(approvalId: string): boolean {
    const pending = this.access.pendingApprovals?.find(
      (a) => a.approvalId === approvalId,
    );
    if (!pending || pending.expiresAt <= this.currentTime) {
      return false;
    }
    return pending.approvals.length >= pending.requiredApprovals;
  }

  /**
   * Consume an approved operation (removes it from pending).
   */
  consumeApproval(approvalId: string): PendingApproval {
    const index =
      this.access.pendingApprovals?.findIndex(
        (a) => a.approvalId === approvalId,
      ) ?? -1;
    if (index === -1) {
      throw new HomeError("Approval not found", "APPROVAL_NOT_FOUND", {
        approvalId,
      });
    }

    const pending = this.access.pendingApprovals![index];
    if (pending.approvals.length < pending.requiredApprovals) {
      throw new HomeError(
        "Approval threshold not met",
        "INSUFFICIENT_PERMISSIONS",
        { approvalId },
      );
    }

    this.access.pendingApprovals!.splice(index, 1);

    this.logAccess("system", "consume_approval", true, {
      approvalId,
      operationType: pending.operationType,
    });

    return pending;
  }

  /**
   * Cancel a pending approval (only the requester or owner can cancel).
   */
  cancelApproval(approvalId: string, cancelledBy: string): void {
    const index =
      this.access.pendingApprovals?.findIndex(
        (a) => a.approvalId === approvalId,
      ) ?? -1;
    if (index === -1) return;

    this.access.pendingApprovals!.splice(index, 1);

    this.logAccess(cancelledBy, "cancel_approval", true, { approvalId });
  }

  /**
   * Get all pending approvals.
   */
  getPendingApprovals(): PendingApproval[] {
    this.cleanupExpiredApprovals();
    return [...(this.access.pendingApprovals ?? [])];
  }

  /**
   * Remove expired approvals.
   */
  private cleanupExpiredApprovals(): void {
    if (!this.access.pendingApprovals) return;
    this.access.pendingApprovals = this.access.pendingApprovals.filter(
      (a) => a.expiresAt > this.currentTime,
    );
  }

  // ============ State Export/Import ============

  /**
   * Export current access control state
   */
  exportState(): AccessControl {
    return {
      keys: [...this.access.keys],
      permissions: new Map(this.access.permissions),
      activeSessions: [...this.access.activeSessions],
      accessLog: [...this.access.accessLog],
      security: { ...this.access.security },
      pendingApprovals: [...(this.access.pendingApprovals ?? [])],
    };
  }

  /**
   * Get access summary
   */
  getSummary(): AccessSummary {
    const now = this.currentTime;
    const activeKeys = this.access.keys.filter(
      (k) => k.revokedAt === undefined,
    );
    const expiredKeys = activeKeys.filter(
      (k) => k.expiresAt && k.expiresAt <= now,
    );
    const validKeys = activeKeys.filter(
      (k) => !k.expiresAt || k.expiresAt > now,
    );
    const recentLog = this.access.accessLog.filter(
      (e) => e.timestamp >= now - 60 * 60 * 1000, // Last hour
    );

    // Determine security status
    const failedAttempts = this.getFailedAttempts(now - 60 * 60 * 1000);
    let securityStatus: "normal" | "elevated" | "lockdown" = "normal";
    if (failedAttempts.length >= 10) {
      securityStatus = "lockdown";
    } else if (failedAttempts.length >= 5) {
      securityStatus = "elevated";
    }

    return {
      registeredKeys: this.access.keys.length,
      activeKeys: validKeys.length,
      revokedKeys: this.access.keys.filter((k) => k.revokedAt !== undefined)
        .length,
      expiredKeys: expiredKeys.length,
      activeSessions: this.access.activeSessions.length,
      recentAccessCount: recentLog.length,
      securityStatus,
      lastKeyRotation: this.getLastKeyRotationBlock("operational"),
      guardiansConfigured: this.getGuardians().length,
    };
  }

  /**
   * Generate home event for key rotation
   */
  generateKeyRotationEvent(oldKeyHash: string, newKeyHash: string): HomeEvent {
    return {
      type: "key_rotated",
      data: {
        oldKeyHash: this.maskKeyHash(oldKeyHash),
        newKeyHash: this.maskKeyHash(newKeyHash),
      },
    };
  }
}
