/**
 * Child Account Manager
 *
 * Implements the patent's child account structure for tracking token expiration.
 * Each account has virtual "child accounts" for each time period when tokens were minted.
 *
 * From Patent: "child accounts are created for each time period" to enable
 * efficient tracking of token age without per-token metadata.
 *
 * PP HIERARCHY:
 *   TRUTH: Each period's balance is independently tracked and verifiable
 *   UNDERSTANDING: Child accounts map directly to expiration periods
 *   BEAUTY: O(1) operations for most common actions
 *   JUSTICE: LIFO transfers preserve value for sender
 */

import { sha512_256Hex } from "../crypto/hashing.js";
import type { DID } from "../agents/identity/did.js";
import {
  type ChildAccount,
  type ExpirationPeriod,
  type ExpirableBalance,
  type ExpirationConfig,
  type TransferPolicy,
  type TransferWithFeeResult,
  type UnlockResult,
  type AccountType,
  type AccountTypeConfig,
  type ParticipationResolver,
  DEFAULT_EXPIRATION_CONFIG,
  DEFAULT_TRANSFER_POLICY,
  TokenError,
} from "./types.js";

/**
 * Child Account Manager
 * Manages the virtual child accounts for token expiration tracking
 */
export class ChildAccountManager {
  private accounts: Map<string, ExpirableBalance> = new Map();
  private childAccounts: Map<string, ChildAccount> = new Map();
  private accountTypes: Map<DID, AccountType> = new Map();
  private typeConfigs: Map<AccountType, AccountTypeConfig> = new Map();

  constructor(
    private config: ExpirationConfig = DEFAULT_EXPIRATION_CONFIG,
    private transferPolicy: TransferPolicy = DEFAULT_TRANSFER_POLICY,
    typeConfigs?: Map<AccountType, AccountTypeConfig>,
    private participationResolver?: ParticipationResolver,
  ) {
    if (typeConfigs) {
      this.typeConfigs = new Map(typeConfigs);
    }
  }

  // ============ Account Type Management ============

  /**
   * Set account type for per-type config resolution
   */
  setAccountType(account: DID, type: AccountType): void {
    this.accountTypes.set(account, type);
  }

  /**
   * Get account type (undefined if not set)
   */
  getAccountType(account: DID): AccountType | undefined {
    return this.accountTypes.get(account);
  }

  /**
   * Set type-specific config override
   */
  setTypeConfig(type: AccountType, typeConfig: AccountTypeConfig): void {
    if ("periodDurationMs" in typeConfig || "tokenLifespanMs" in typeConfig) {
      throw new TokenError(
        "Cannot override periodDurationMs or tokenLifespanMs per type",
        "INVALID_TYPE_CONFIG",
      );
    }
    this.typeConfigs.set(type, typeConfig);
  }

  /**
   * Get type-specific config (undefined if not set)
   */
  getTypeConfig(type: AccountType): AccountTypeConfig | undefined {
    return this.typeConfigs.get(type);
  }

  /**
   * Set the participation resolver (for dependency injection after construction)
   */
  setParticipationResolver(resolver: ParticipationResolver | undefined): void {
    this.participationResolver = resolver;
  }

  /**
   * Resolve effective config for an account.
   * Priority: participationResolver > typeConfig > globalConfig
   *
   * When a participationResolver is set, it dynamically computes
   * periodsUntilExpiration based on the account's protocol participation.
   * If the resolver returns null, falls back to static config.
   */
  private resolveConfig(account: DID): ExpirationConfig {
    // Start with global config
    let resolved = this.config;

    // Apply per-type override if applicable
    const accountType = this.accountTypes.get(account);
    if (accountType) {
      const override = this.typeConfigs.get(accountType);
      if (override) {
        resolved = {
          ...resolved,
          ...override,
          periodDurationMs: this.config.periodDurationMs,
          tokenLifespanMs: this.config.tokenLifespanMs,
        };
      }
    }

    // Apply participation-based override (highest priority)
    if (this.participationResolver) {
      const score = this.participationResolver(account);
      if (score !== null) {
        resolved = {
          ...resolved,
          periodsUntilExpiration: score.periodsGranted,
          tokenLifespanMs: score.periodsGranted * this.config.periodDurationMs,
        };
      }
    }

    return resolved;
  }

  // ============ Account Operations ============

  /**
   * Get or create an expirable balance for an account
   */
  getOrCreateBalance(account: DID): ExpirableBalance {
    const existing = this.accounts.get(account);
    if (existing) {
      return existing;
    }

    const newBalance: ExpirableBalance = {
      account,
      totalBalance: 0n,
      periodBalances: new Map(),
      earliestPeriod: Number.MAX_SAFE_INTEGER,
      latestPeriod: 0,
      totalExpired: 0n,
      updatedAt: Date.now(),
    };

    this.accounts.set(account, newBalance);
    return newBalance;
  }

  /**
   * Get current balance (non-expired only)
   */
  getBalance(account: DID, currentTime: number): bigint {
    const balance = this.accounts.get(account);
    if (!balance) {
      return 0n;
    }

    // Calculate non-expired balance
    const resolved = this.resolveConfig(account);
    const currentPeriod = this.timestampToPeriod(currentTime);
    const expirationPeriod = currentPeriod - resolved.periodsUntilExpiration;

    let activeBalance = 0n;
    for (const [period, amount] of balance.periodBalances) {
      if (period > expirationPeriod) {
        activeBalance += amount;
      }
    }

    return activeBalance;
  }

  /**
   * Get detailed balance breakdown by period
   */
  getBalanceBreakdown(
    account: DID,
    currentTime: number,
  ): {
    active: Map<ExpirationPeriod, bigint>;
    expired: Map<ExpirationPeriod, bigint>;
    expiringSoon: Map<ExpirationPeriod, bigint>;
    locked: Map<ExpirationPeriod, bigint>;
    totalActive: bigint;
    totalExpired: bigint;
    totalLocked: bigint;
  } {
    const balance = this.accounts.get(account);
    const active = new Map<ExpirationPeriod, bigint>();
    const expired = new Map<ExpirationPeriod, bigint>();
    const expiringSoon = new Map<ExpirationPeriod, bigint>();
    const locked = new Map<ExpirationPeriod, bigint>();
    let totalActive = 0n;
    let totalExpired = 0n;
    let totalLocked = 0n;

    if (!balance) {
      return {
        active,
        expired,
        expiringSoon,
        locked,
        totalActive,
        totalExpired,
        totalLocked,
      };
    }

    const resolved = this.resolveConfig(account);
    const currentPeriod = this.timestampToPeriod(currentTime);
    const expirationPeriod = currentPeriod - resolved.periodsUntilExpiration;
    const expiringSoonPeriod = expirationPeriod + 4; // Next 4 periods

    for (const [period, amount] of balance.periodBalances) {
      if (period <= expirationPeriod) {
        // Check if this period is locked (lock_and_unlock mode)
        const childKey = this.getChildAccountKey(account, period);
        const childAccount = this.childAccounts.get(childKey);
        if (childAccount?.locked && !childAccount.swept) {
          locked.set(period, amount);
          totalLocked += amount;
        } else {
          expired.set(period, amount);
          totalExpired += amount;
        }
      } else if (period <= expiringSoonPeriod) {
        expiringSoon.set(period, amount);
        active.set(period, amount);
        totalActive += amount;
      } else {
        active.set(period, amount);
        totalActive += amount;
      }
    }

    return {
      active,
      expired,
      expiringSoon,
      locked,
      totalActive,
      totalExpired,
      totalLocked,
    };
  }

  // ============ Credit Operations ============

  /**
   * Credit tokens to an account (mint or receive)
   */
  credit(
    account: DID,
    amount: bigint,
    currentTime: number,
    freshExpiration: boolean = true,
  ): {
    balance: ExpirableBalance;
    childAccount: ChildAccount;
    period: ExpirationPeriod;
  } {
    const resolved = this.resolveConfig(account);
    if (amount < resolved.dustThreshold) {
      throw new TokenError(
        `Amount ${amount} below dust threshold ${resolved.dustThreshold}`,
        "DUST_AMOUNT",
      );
    }

    const balance = this.getOrCreateBalance(account);
    const period = freshExpiration
      ? this.timestampToPeriod(currentTime)
      : this.timestampToPeriod(currentTime); // Could inherit period from sender

    // Update period balance
    const currentPeriodBalance = balance.periodBalances.get(period) ?? 0n;
    balance.periodBalances.set(period, currentPeriodBalance + amount);

    // Update totals
    balance.totalBalance += amount;
    balance.earliestPeriod = Math.min(balance.earliestPeriod, period);
    balance.latestPeriod = Math.max(balance.latestPeriod, period);
    balance.updatedAt = Date.now();

    // Create/update child account
    const childKey = this.getChildAccountKey(account, period);
    const childAccount = this.getOrCreateChildAccount(
      account,
      period,
      currentTime,
    );
    childAccount.balance += amount;

    this.childAccounts.set(childKey, childAccount);
    this.accounts.set(account, balance);

    return { balance, childAccount, period };
  }

  /**
   * Debit tokens from an account (transfer or burn)
   * Uses LIFO by default to preserve value for sender
   */
  debit(
    account: DID,
    amount: bigint,
    currentTime: number,
  ): {
    balance: ExpirableBalance;
    periodBreakdown: Map<ExpirationPeriod, bigint>;
  } {
    const balance = this.accounts.get(account);
    if (!balance) {
      throw new TokenError(`Account ${account} not found`, "ACCOUNT_NOT_FOUND");
    }

    const activeBalance = this.getBalance(account, currentTime);
    if (activeBalance < amount) {
      throw new TokenError(
        `Insufficient balance: ${activeBalance} < ${amount}`,
        "INSUFFICIENT_BALANCE",
      );
    }

    const periodBreakdown = new Map<ExpirationPeriod, bigint>();
    let remaining = amount;

    // Get sorted periods based on transfer policy
    const resolved = this.resolveConfig(account);
    const currentPeriod = this.timestampToPeriod(currentTime);
    const expirationPeriod = currentPeriod - resolved.periodsUntilExpiration;
    const activePeriods = Array.from(balance.periodBalances.keys()).filter(
      (p) => p > expirationPeriod,
    );

    // Sort: LIFO = newest first, FIFO = oldest first
    if (this.transferPolicy.order === "lifo") {
      activePeriods.sort((a, b) => b - a);
    } else {
      activePeriods.sort((a, b) => a - b);
    }

    // Debit from periods
    for (const period of activePeriods) {
      if (remaining === 0n) break;

      const periodBalance = balance.periodBalances.get(period) ?? 0n;
      if (periodBalance === 0n) continue;

      const debitAmount = periodBalance < remaining ? periodBalance : remaining;
      balance.periodBalances.set(period, periodBalance - debitAmount);
      remaining -= debitAmount;
      periodBreakdown.set(period, debitAmount);

      // Update child account
      const childKey = this.getChildAccountKey(account, period);
      const childAccount = this.childAccounts.get(childKey);
      if (childAccount) {
        childAccount.balance -= debitAmount;
        if (childAccount.balance === 0n) {
          // Could delete, but keeping for history
        }
      }

      // Clean up zero balances
      if (balance.periodBalances.get(period) === 0n) {
        balance.periodBalances.delete(period);
      }
    }

    // Update balance totals
    balance.totalBalance -= amount;
    balance.updatedAt = Date.now();

    // Update earliest/latest pointers
    this.updatePeriodPointers(balance);

    this.accounts.set(account, balance);

    return { balance, periodBreakdown };
  }

  // ============ Expiration Operations ============

  /**
   * Process expiration for an account
   * Returns amount expired and updates balance
   */
  processExpiration(
    account: DID,
    currentTime: number,
  ): {
    expiredAmount: bigint;
    expiredPeriods: ExpirationPeriod[];
    lockedAmount: bigint;
    lockedPeriods: ExpirationPeriod[];
  } {
    const balance = this.accounts.get(account);
    if (!balance) {
      return {
        expiredAmount: 0n,
        expiredPeriods: [],
        lockedAmount: 0n,
        lockedPeriods: [],
      };
    }

    const resolved = this.resolveConfig(account);
    const currentPeriod = this.timestampToPeriod(currentTime);
    const expirationPeriod = currentPeriod - resolved.periodsUntilExpiration;

    let expiredAmount = 0n;
    const expiredPeriods: ExpirationPeriod[] = [];
    let lockedAmount = 0n;
    const lockedPeriods: ExpirationPeriod[] = [];

    for (const [period, amount] of balance.periodBalances) {
      if (period <= expirationPeriod && amount > 0n) {
        const childKey = this.getChildAccountKey(account, period);
        const childAccount = this.childAccounts.get(childKey);

        if (resolved.expirationMode === "lock_and_unlock") {
          // Lock mode: freeze tokens in-place instead of sweeping
          if (childAccount && !childAccount.locked && !childAccount.swept) {
            childAccount.locked = true;
            childAccount.lockedAtBlock = BigInt(currentPeriod); // approximate
            lockedAmount += amount;
            lockedPeriods.push(period);
          }
          // Do NOT delete from periodBalances or subtract from totalBalance
        } else {
          // Direct retrieval mode: sweep expired tokens (existing behavior)
          expiredAmount += amount;
          expiredPeriods.push(period);
          balance.periodBalances.delete(period);

          if (childAccount) {
            childAccount.swept = true;
          }
        }
      }
    }

    if (expiredAmount > 0n) {
      balance.totalBalance -= expiredAmount;
      balance.totalExpired += expiredAmount;
      balance.updatedAt = Date.now();
      this.updatePeriodPointers(balance);
      this.accounts.set(account, balance);
    } else if (lockedAmount > 0n) {
      balance.updatedAt = Date.now();
      this.accounts.set(account, balance);
    }

    return { expiredAmount, expiredPeriods, lockedAmount, lockedPeriods };
  }

  /**
   * Get all accounts with expired balances
   */
  getAccountsWithExpiredBalances(currentTime: number): DID[] {
    const currentPeriod = this.timestampToPeriod(currentTime);
    const accounts: DID[] = [];

    for (const [account, balance] of this.accounts) {
      const resolved = this.resolveConfig(account);
      const expirationPeriod = currentPeriod - resolved.periodsUntilExpiration;
      if (balance.earliestPeriod <= expirationPeriod) {
        accounts.push(account);
      }
    }

    return accounts;
  }

  /**
   * Get total expired amount across all accounts for a period
   */
  getTotalExpiredForPeriod(
    period: ExpirationPeriod,
    currentTime: number,
  ): bigint {
    const currentPeriod = this.timestampToPeriod(currentTime);

    let total = 0n;
    for (const [account, balance] of this.accounts) {
      const resolved = this.resolveConfig(account);
      const expirationPeriod = currentPeriod - resolved.periodsUntilExpiration;

      if (period > expirationPeriod) {
        continue; // Period hasn't expired for this account yet
      }

      const periodBalance = balance.periodBalances.get(period);
      if (periodBalance) {
        total += periodBalance;
      }
    }

    return total;
  }

  // ============ Extension Operations ============

  /**
   * Extend expiration for tokens (e.g., when staked)
   */
  extendExpiration(
    account: DID,
    amount: bigint,
    extensionPeriods: number,
    currentTime: number,
  ): {
    extended: bigint;
    newPeriod: ExpirationPeriod;
    oldPeriods: ExpirationPeriod[];
  } {
    const resolved = this.resolveConfig(account);
    if (!resolved.allowExtension) {
      throw new TokenError(
        "Token expiration extension is not allowed",
        "EXTENSION_NOT_ALLOWED",
      );
    }

    if (extensionPeriods > resolved.maxExtensionPeriods) {
      throw new TokenError(
        `Extension ${extensionPeriods} exceeds max ${this.config.maxExtensionPeriods}`,
        "MAX_EXTENSION_EXCEEDED",
      );
    }

    // Debit from oldest periods
    const { periodBreakdown } = this.debit(account, amount, currentTime);

    // Credit to extended period (current + extension)
    const newPeriod = this.timestampToPeriod(currentTime) + extensionPeriods;
    const balance = this.getOrCreateBalance(account);
    const currentPeriodBalance = balance.periodBalances.get(newPeriod) ?? 0n;
    balance.periodBalances.set(newPeriod, currentPeriodBalance + amount);

    balance.totalBalance += amount;
    balance.latestPeriod = Math.max(balance.latestPeriod, newPeriod);
    balance.updatedAt = Date.now();

    // Update child account
    const childAccount = this.getOrCreateChildAccount(
      account,
      newPeriod,
      currentTime,
    );
    childAccount.balance += amount;
    childAccount.expiresAt = this.periodToExpiration(newPeriod, account);
    this.childAccounts.set(
      this.getChildAccountKey(account, newPeriod),
      childAccount,
    );

    this.accounts.set(account, balance);

    return {
      extended: amount,
      newPeriod,
      oldPeriods: Array.from(periodBreakdown.keys()),
    };
  }

  // ============ Retrieval Rate Operations ============

  /**
   * Calculate retrieval fee for a given amount
   * Uses basis points (rate * 10000) for integer precision
   * From Patent: "Retrieval Rate (R2)" applied on transfers
   */
  calculateRetrievalFee(amount: bigint, account?: DID): bigint {
    const resolved = account ? this.resolveConfig(account) : this.config;
    const rate = resolved.retrievalRate;
    if (rate <= 0 || amount <= 0n) return 0n;
    const rateBasisPoints = BigInt(Math.floor(rate * 10000));
    return (amount * rateBasisPoints) / 10000n;
  }

  /**
   * Transfer with retrieval rate applied
   * Debits gross amount from sender, credits net (gross - fee) to recipient
   * Fee is returned for caller to route to RAI holding
   */
  transferWithFee(
    from: DID,
    to: DID,
    amount: bigint,
    currentTime: number,
    applyRate: boolean = this.transferPolicy.applyRetrievalRate,
  ): TransferWithFeeResult {
    // Debit full amount from sender
    const { periodBreakdown } = this.debit(from, amount, currentTime);

    // Calculate fee
    const fee = applyRate ? this.calculateRetrievalFee(amount, from) : 0n;
    const netAmount = amount - fee;

    // Credit net amount to recipient
    const { period: recipientPeriod } = this.credit(
      to,
      netAmount,
      currentTime,
      this.transferPolicy.freshExpirationOnReceive,
    );

    return {
      grossAmount: amount,
      fee,
      netAmount,
      periodBreakdown,
      recipientPeriod,
    };
  }

  // ============ Lock and Unlock Operations ============

  /**
   * Unlock expired (locked) tokens and re-activate them with a renewal fee
   * Only available in 'lock_and_unlock' expiration mode
   * From Patent: "Lock and Unlock" method with "Renewal Fee Rate (R1)"
   */
  unlockExpired(
    account: DID,
    currentTime: number,
    periods?: ExpirationPeriod[],
  ): UnlockResult {
    const resolved = this.resolveConfig(account);
    if (resolved.expirationMode !== "lock_and_unlock") {
      throw new TokenError(
        "Unlock is only available in lock_and_unlock expiration mode",
        "PERIOD_NOT_LOCKED",
      );
    }

    const balance = this.accounts.get(account);
    if (!balance) {
      throw new TokenError(`Account ${account} not found`, "ACCOUNT_NOT_FOUND");
    }

    const currentPeriod = this.timestampToPeriod(currentTime);
    const expirationPeriod = currentPeriod - resolved.periodsUntilExpiration;

    // Find locked periods
    let lockedTotal = 0n;
    const unlockedPeriods: ExpirationPeriod[] = [];

    for (const [period, amount] of balance.periodBalances) {
      if (period > expirationPeriod) continue; // Not expired
      if (amount <= 0n) continue;

      const childKey = this.getChildAccountKey(account, period);
      const childAccount = this.childAccounts.get(childKey);
      if (!childAccount || !childAccount.locked || childAccount.swept) continue;

      // Filter to specific periods if requested
      if (periods && !periods.includes(period)) continue;

      lockedTotal += amount;
      unlockedPeriods.push(period);
    }

    if (lockedTotal === 0n) {
      throw new TokenError(
        `No locked balance found for account ${account}`,
        "NO_LOCKED_BALANCE",
      );
    }

    // Calculate renewal fee (R1 defaults to R2 if not set)
    const renewalRate = resolved.renewalFeeRate ?? resolved.retrievalRate;
    const renewalFeeBasisPoints = BigInt(Math.floor(renewalRate * 10000));
    const renewalFee = (lockedTotal * renewalFeeBasisPoints) / 10000n;
    const netUnlocked = lockedTotal - renewalFee;

    // Remove old locked periods
    for (const period of unlockedPeriods) {
      balance.periodBalances.delete(period);
      const childKey = this.getChildAccountKey(account, period);
      const childAccount = this.childAccounts.get(childKey);
      if (childAccount) {
        childAccount.swept = true;
        childAccount.locked = false;
      }
    }

    // Subtract fee from totalBalance (fee leaves the account)
    balance.totalBalance -= renewalFee;

    // Credit net amount to current period (fresh expiration)
    const newPeriod = currentPeriod;
    const currentPeriodBalance = balance.periodBalances.get(newPeriod) ?? 0n;
    balance.periodBalances.set(newPeriod, currentPeriodBalance + netUnlocked);
    balance.latestPeriod = Math.max(balance.latestPeriod, newPeriod);

    // Create/update child account for new period
    const newChildKey = this.getChildAccountKey(account, newPeriod);
    const newChildAccount = this.getOrCreateChildAccount(
      account,
      newPeriod,
      currentTime,
    );
    newChildAccount.balance += netUnlocked;
    this.childAccounts.set(newChildKey, newChildAccount);

    balance.updatedAt = Date.now();
    this.updatePeriodPointers(balance);
    this.accounts.set(account, balance);

    return {
      unlocked: lockedTotal,
      renewalFee,
      netUnlocked,
      newPeriod,
      unlockedPeriods,
    };
  }

  /**
   * Get total locked balance for an account
   */
  getLockedBalance(account: DID, currentTime: number): bigint {
    const breakdown = this.getBalanceBreakdown(account, currentTime);
    return breakdown.totalLocked;
  }

  // ============ Period Utilities ============

  /**
   * Convert timestamp to period number
   */
  timestampToPeriod(timestamp: number): ExpirationPeriod {
    return Math.floor(timestamp / this.config.periodDurationMs);
  }

  /**
   * Convert period to expiration timestamp
   */
  periodToExpiration(period: ExpirationPeriod, account?: DID): number {
    const resolved = account ? this.resolveConfig(account) : this.config;
    return (
      (period + resolved.periodsUntilExpiration) * this.config.periodDurationMs
    );
  }

  /**
   * Get current period
   */
  getCurrentPeriod(): ExpirationPeriod {
    return this.timestampToPeriod(Date.now());
  }

  /**
   * Check if a period has expired
   */
  isPeriodExpired(
    period: ExpirationPeriod,
    currentTime: number,
    account?: DID,
  ): boolean {
    const resolved = account ? this.resolveConfig(account) : this.config;
    const currentPeriod = this.timestampToPeriod(currentTime);
    return period <= currentPeriod - resolved.periodsUntilExpiration;
  }

  // ============ Child Account Utilities ============

  /**
   * Generate child account key
   */
  private getChildAccountKey(account: DID, period: ExpirationPeriod): string {
    return sha512_256Hex(`child:${account}:${period}`);
  }

  /**
   * Get or create a child account
   */
  private getOrCreateChildAccount(
    account: DID,
    period: ExpirationPeriod,
    _currentTime: number,
  ): ChildAccount {
    const key = this.getChildAccountKey(account, period);
    const existing = this.childAccounts.get(key);

    if (existing) {
      return existing;
    }

    const childAccount: ChildAccount = {
      parentAccount: account,
      period,
      balance: 0n,
      expiresAt: this.periodToExpiration(period, account),
      swept: false,
      locked: false,
    };

    return childAccount;
  }

  /**
   * Update earliest/latest period pointers after balance change
   */
  private updatePeriodPointers(balance: ExpirableBalance): void {
    if (balance.periodBalances.size === 0) {
      balance.earliestPeriod = Number.MAX_SAFE_INTEGER;
      balance.latestPeriod = 0;
      return;
    }

    const periods = Array.from(balance.periodBalances.keys());
    balance.earliestPeriod = Math.min(...periods);
    balance.latestPeriod = Math.max(...periods);
  }

  // ============ Query Methods ============

  /**
   * Get all balances (for state export)
   */
  getAllBalances(): Map<string, ExpirableBalance> {
    return new Map(this.accounts);
  }

  /**
   * Get all child accounts (for state export)
   */
  getAllChildAccounts(): Map<string, ChildAccount> {
    return new Map(this.childAccounts);
  }

  /**
   * Get child accounts for a specific account
   */
  getChildAccountsFor(account: DID): ChildAccount[] {
    const children: ChildAccount[] = [];
    for (const [, child] of this.childAccounts) {
      if (child.parentAccount === account) {
        children.push(child);
      }
    }
    return children.sort((a, b) => a.period - b.period);
  }

  /**
   * Import state (for initialization from storage)
   */
  importState(
    balances: Map<string, ExpirableBalance>,
    childAccounts: Map<string, ChildAccount>,
  ): void {
    this.accounts = new Map(balances);
    this.childAccounts = new Map(childAccounts);
  }

  /**
   * Get statistics
   */
  getStats(currentTime: number): {
    totalAccounts: number;
    totalChildAccounts: number;
    totalActiveBalance: bigint;
    totalExpiredBalance: bigint;
    periodDistribution: Map<ExpirationPeriod, bigint>;
  } {
    let totalActiveBalance = 0n;
    let totalExpiredBalance = 0n;
    const periodDistribution = new Map<ExpirationPeriod, bigint>();

    for (const [, balance] of this.accounts) {
      const breakdown = this.getBalanceBreakdown(balance.account, currentTime);
      totalActiveBalance += breakdown.totalActive;
      totalExpiredBalance += breakdown.totalExpired;

      for (const [period, amount] of breakdown.active) {
        const current = periodDistribution.get(period) ?? 0n;
        periodDistribution.set(period, current + amount);
      }
    }

    return {
      totalAccounts: this.accounts.size,
      totalChildAccounts: this.childAccounts.size,
      totalActiveBalance,
      totalExpiredBalance,
      periodDistribution,
    };
  }
}
