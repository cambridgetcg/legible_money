/**
 * Expiration Tracker (Dynamic Array Implementation)
 *
 * Implements the patent's Dynamic Array method for efficient token expiration tracking.
 * Maintains global state of periods and coordinates expiration across all accounts.
 *
 * From Patent: "Token Expiration is calculated by the Dynamic Array method, in which
 * each balance position in array is reset at regular intervals"
 *
 * The Dynamic Array approach enables:
 * - O(1) balance queries (sum of non-expired periods)
 * - Batch expiration processing (all accounts in a period at once)
 * - Efficient state proofs (only active periods need inclusion)
 *
 * PP HIERARCHY:
 *   TRUTH: Deterministic expiration based on immutable mint time
 *   UNDERSTANDING: Period-based tracking is intuitive and verifiable
 *   BEAUTY: Single data structure tracks all expiration state
 *   JUSTICE: Automatic redistribution to RAI fund
 */

import { sha512_256Hex } from "../crypto/hashing.js";
import {
  type ExpirationPeriod,
  type PeriodTransition,
  type ExpirationConfig,
  type DynamicArrayState,
  DEFAULT_EXPIRATION_CONFIG,
  TokenError,
} from "./types.js";
import type { DID } from "../agents/identity/did.js";

/**
 * Period state tracking
 */
interface PeriodState {
  /** Period number */
  period: ExpirationPeriod;

  /** Total tokens minted in this period */
  totalMinted: bigint;

  /** Currently active balance in this period */
  activeBalance: bigint;

  /** Already expired from this period */
  expiredBalance: bigint;

  /** Number of accounts with balance in this period */
  accountCount: number;

  /** Block when this period started */
  startedAtBlock: bigint;

  /** Whether this period has fully expired */
  fullyExpired: boolean;

  /** Block when fully expired (if applicable) */
  expiredAtBlock?: bigint;
}

/**
 * Global expiration state
 */
interface GlobalExpirationState {
  /** Current period */
  currentPeriod: ExpirationPeriod;

  /** Oldest period with active balances */
  oldestActivePeriod: ExpirationPeriod;

  /** Total supply (active only) */
  totalActiveSupply: bigint;

  /** Total expired (all time) */
  totalExpiredSupply: bigint;

  /** Last period transition block */
  lastTransitionBlock: bigint;
}

/**
 * Expiration Tracker
 * Coordinates token expiration across the entire system
 */
export class ExpirationTracker {
  private periodStates: Map<ExpirationPeriod, PeriodState> = new Map();
  private accountPeriods: Map<string, Set<ExpirationPeriod>> = new Map();
  private globalState: GlobalExpirationState;

  constructor(
    private config: ExpirationConfig = DEFAULT_EXPIRATION_CONFIG,
    initialPeriod?: ExpirationPeriod,
  ) {
    const currentPeriod = initialPeriod ?? 0;

    this.globalState = {
      currentPeriod,
      oldestActivePeriod: currentPeriod,
      totalActiveSupply: 0n,
      totalExpiredSupply: 0n,
      lastTransitionBlock: 0n,
    };
  }

  // ============ Period Management ============

  /**
   * Get or create period state
   */
  getOrCreatePeriod(
    period: ExpirationPeriod,
    currentBlock: bigint,
  ): PeriodState {
    const existing = this.periodStates.get(period);
    if (existing) {
      return existing;
    }

    const periodState: PeriodState = {
      period,
      totalMinted: 0n,
      activeBalance: 0n,
      expiredBalance: 0n,
      accountCount: 0,
      startedAtBlock: currentBlock,
      fullyExpired: false,
    };

    this.periodStates.set(period, periodState);
    return periodState;
  }

  /**
   * Record a mint in a period
   */
  recordMint(
    account: DID,
    amount: bigint,
    period: ExpirationPeriod,
    currentBlock: bigint,
  ): void {
    const periodState = this.getOrCreatePeriod(period, currentBlock);

    // Check if this is a new account for this period
    const accountPeriodsSet = this.accountPeriods.get(account) ?? new Set();
    const isNewAccountForPeriod = !accountPeriodsSet.has(period);

    // Update period state
    periodState.totalMinted += amount;
    periodState.activeBalance += amount;
    if (isNewAccountForPeriod) {
      periodState.accountCount++;
      accountPeriodsSet.add(period);
      this.accountPeriods.set(account, accountPeriodsSet);
    }

    this.periodStates.set(period, periodState);

    // Update global state
    this.globalState.totalActiveSupply += amount;
    if (period > this.globalState.currentPeriod) {
      this.globalState.currentPeriod = period;
    }
  }

  /**
   * Record a debit (transfer or expiration) from a period
   */
  recordDebit(
    _account: DID,
    amount: bigint,
    period: ExpirationPeriod,
    isExpiration: boolean,
  ): void {
    const periodState = this.periodStates.get(period);
    if (!periodState) {
      throw new TokenError(`Period ${period} not found`, "PERIOD_NOT_FOUND");
    }

    periodState.activeBalance -= amount;
    if (isExpiration) {
      periodState.expiredBalance += amount;
      this.globalState.totalExpiredSupply += amount;
    }
    this.globalState.totalActiveSupply -= amount;

    // Note: We don't decrement accountCount on debit
    // Account may still have balance from other transactions
    // accountCount is "accounts that have ever had balance in period"

    this.periodStates.set(period, periodState);
  }

  /**
   * Process period transition (when time advances to new period)
   */
  processPeriodTransition(
    newPeriod: ExpirationPeriod,
    currentBlock: bigint,
  ): PeriodTransition | null {
    if (newPeriod <= this.globalState.currentPeriod) {
      return null; // No transition needed
    }

    // Find the period that just expired
    const expiredPeriod = newPeriod - this.config.periodsUntilExpiration;
    const expiredState = this.periodStates.get(expiredPeriod);

    let expiredAmount = 0n;
    let accountsAffected = 0;

    if (expiredState && !expiredState.fullyExpired) {
      expiredAmount = expiredState.activeBalance;
      accountsAffected = expiredState.accountCount;

      // Mark period as fully expired
      expiredState.fullyExpired = true;
      expiredState.expiredAtBlock = currentBlock;
      expiredState.expiredBalance += expiredState.activeBalance;
      expiredState.activeBalance = 0n;

      this.periodStates.set(expiredPeriod, expiredState);

      // Update global state
      this.globalState.totalExpiredSupply += expiredAmount;
      this.globalState.totalActiveSupply -= expiredAmount;
    }

    // Update global state
    // Previous period stored in globalState.currentPeriod before transition
    this.globalState.currentPeriod = newPeriod;
    this.globalState.lastTransitionBlock = currentBlock;

    // Update oldest active period
    while (
      this.globalState.oldestActivePeriod <
      newPeriod - this.config.periodsUntilExpiration
    ) {
      this.globalState.oldestActivePeriod++;
    }

    if (expiredAmount > 0n) {
      return {
        expiredPeriod,
        newPeriod,
        expiredAmount,
        accountsAffected,
        transitionBlock: currentBlock,
      };
    }

    return null;
  }

  // ============ Query Methods ============

  /**
   * Get periods that have expired but not been fully processed
   */
  getExpiredPeriods(currentTime: number): ExpirationPeriod[] {
    const currentPeriod = this.timestampToPeriod(currentTime);
    const expirationThreshold =
      currentPeriod - this.config.periodsUntilExpiration;
    const expired: ExpirationPeriod[] = [];

    for (const [period, state] of this.periodStates) {
      if (period <= expirationThreshold && !state.fullyExpired) {
        expired.push(period);
      }
    }

    return expired.sort((a, b) => a - b);
  }

  /**
   * Get periods expiring soon (for warnings)
   */
  getExpiringSoonPeriods(
    currentTime: number,
    withinPeriods: number = 4,
  ): Array<{
    period: ExpirationPeriod;
    balance: bigint;
    periodsUntilExpiration: number;
  }> {
    const currentPeriod = this.timestampToPeriod(currentTime);
    const expirationThreshold =
      currentPeriod - this.config.periodsUntilExpiration;
    const soonThreshold = expirationThreshold + withinPeriods;
    const expiringSoon: Array<{
      period: ExpirationPeriod;
      balance: bigint;
      periodsUntilExpiration: number;
    }> = [];

    for (const [period, state] of this.periodStates) {
      if (
        period > expirationThreshold &&
        period <= soonThreshold &&
        state.activeBalance > 0n
      ) {
        expiringSoon.push({
          period,
          balance: state.activeBalance,
          periodsUntilExpiration:
            period + this.config.periodsUntilExpiration - currentPeriod,
        });
      }
    }

    return expiringSoon.sort((a, b) => a.period - b.period);
  }

  /**
   * Get total balance that will expire in next N periods
   */
  getExpiringBalance(currentTime: number, withinPeriods: number): bigint {
    const expiring = this.getExpiringSoonPeriods(currentTime, withinPeriods);
    return expiring.reduce((sum, p) => sum + p.balance, 0n);
  }

  /**
   * Get period state
   */
  getPeriodState(period: ExpirationPeriod): PeriodState | undefined {
    return this.periodStates.get(period);
  }

  /**
   * Get global expiration state
   */
  getGlobalState(): GlobalExpirationState {
    return { ...this.globalState };
  }

  /**
   * Get all active periods (non-expired with balance)
   */
  getActivePeriods(): ExpirationPeriod[] {
    const active: ExpirationPeriod[] = [];
    for (const [period, state] of this.periodStates) {
      if (!state.fullyExpired && state.activeBalance > 0n) {
        active.push(period);
      }
    }
    return active.sort((a, b) => a - b);
  }

  /**
   * Get periods for an account
   */
  getAccountPeriods(account: DID): ExpirationPeriod[] {
    const periods = this.accountPeriods.get(account);
    if (!periods) {
      return [];
    }
    return Array.from(periods).sort((a, b) => a - b);
  }

  // ============ Dynamic Array State ============

  /**
   * Build Dynamic Array state for an account
   */
  buildDynamicArrayState(
    account: DID,
    periodBalances: Map<ExpirationPeriod, bigint>,
  ): DynamicArrayState {
    const periods = this.getAccountPeriods(account);
    const activePeriods = periods.filter((p) => {
      const state = this.periodStates.get(p);
      return state && !state.fullyExpired;
    });

    let activeBalance = 0n;
    let pendingExpiration = 0n;
    const expiringSoon = this.getExpiringSoonPeriods(
      this.globalState.currentPeriod * this.config.periodDurationMs,
      1,
    );
    const expiringSoonPeriods = new Set(expiringSoon.map((e) => e.period));

    for (const period of activePeriods) {
      const balance = periodBalances.get(period) ?? 0n;
      activeBalance += balance;
      if (expiringSoonPeriods.has(period)) {
        pendingExpiration += balance;
      }
    }

    return {
      account,
      periods: periodBalances,
      currentPeriod: this.globalState.currentPeriod,
      headPeriod: activePeriods[0] ?? this.globalState.currentPeriod,
      tailPeriod:
        activePeriods[activePeriods.length - 1] ??
        this.globalState.currentPeriod,
      activeBalance,
      pendingExpiration,
    };
  }

  // ============ Merkle Proof Support ============

  /**
   * Generate period state hash for Merkle inclusion
   */
  getPeriodStateHash(period: ExpirationPeriod): string {
    const state = this.periodStates.get(period);
    if (!state) {
      return sha512_256Hex(`period:${period}:empty`);
    }

    return sha512_256Hex(
      `period:${period}:${state.activeBalance}:${state.expiredBalance}:${state.fullyExpired}`,
    );
  }

  /**
   * Generate global state hash
   */
  getGlobalStateHash(): string {
    return sha512_256Hex(
      `global:${this.globalState.currentPeriod}:` +
        `${this.globalState.totalActiveSupply}:` +
        `${this.globalState.totalExpiredSupply}`,
    );
  }

  // ============ Utilities ============

  /**
   * Convert timestamp to period
   */
  timestampToPeriod(timestamp: number): ExpirationPeriod {
    return Math.floor(timestamp / this.config.periodDurationMs);
  }

  /**
   * Convert period to timestamp (start of period)
   */
  periodToTimestamp(period: ExpirationPeriod): number {
    return period * this.config.periodDurationMs;
  }

  /**
   * Get time until period expires
   */
  getTimeUntilExpiration(
    period: ExpirationPeriod,
    currentTime: number,
  ): number {
    const currentPeriod = this.timestampToPeriod(currentTime);
    const expiresAtPeriod = period + this.config.periodsUntilExpiration;
    const periodsRemaining = expiresAtPeriod - currentPeriod;

    if (periodsRemaining <= 0) {
      return 0;
    }

    return periodsRemaining * this.config.periodDurationMs;
  }

  // ============ Statistics ============

  /**
   * Get comprehensive statistics
   */
  getStatistics(): {
    totalPeriods: number;
    activePeriods: number;
    expiredPeriods: number;
    totalActiveSupply: bigint;
    totalExpiredSupply: bigint;
    totalMinted: bigint;
    accountsWithBalances: number;
    averageBalancePerPeriod: bigint;
    oldestActivePeriod: ExpirationPeriod;
    newestActivePeriod: ExpirationPeriod;
  } {
    let activePeriods = 0;
    let expiredPeriods = 0;
    let totalMinted = 0n;
    let newestActivePeriod = 0;

    for (const [period, state] of this.periodStates) {
      totalMinted += state.totalMinted;
      if (state.fullyExpired) {
        expiredPeriods++;
      } else if (state.activeBalance > 0n) {
        activePeriods++;
        newestActivePeriod = Math.max(newestActivePeriod, period);
      }
    }

    return {
      totalPeriods: this.periodStates.size,
      activePeriods,
      expiredPeriods,
      totalActiveSupply: this.globalState.totalActiveSupply,
      totalExpiredSupply: this.globalState.totalExpiredSupply,
      totalMinted,
      accountsWithBalances: this.accountPeriods.size,
      averageBalancePerPeriod:
        activePeriods > 0
          ? this.globalState.totalActiveSupply / BigInt(activePeriods)
          : 0n,
      oldestActivePeriod: this.globalState.oldestActivePeriod,
      newestActivePeriod,
    };
  }

  // ============ State Import/Export ============

  /**
   * Export state for persistence
   */
  exportState(): {
    periodStates: Map<ExpirationPeriod, PeriodState>;
    accountPeriods: Map<string, Set<ExpirationPeriod>>;
    globalState: GlobalExpirationState;
  } {
    return {
      periodStates: new Map(this.periodStates),
      accountPeriods: new Map(
        Array.from(this.accountPeriods.entries()).map(([k, v]) => [
          k,
          new Set(v),
        ]),
      ),
      globalState: { ...this.globalState },
    };
  }

  /**
   * Import state from persistence
   */
  importState(state: {
    periodStates: Map<ExpirationPeriod, PeriodState>;
    accountPeriods: Map<string, Set<ExpirationPeriod>>;
    globalState: GlobalExpirationState;
  }): void {
    this.periodStates = new Map(state.periodStates);
    this.accountPeriods = new Map(
      Array.from(state.accountPeriods.entries()).map(([k, v]) => [
        k,
        new Set(v),
      ]),
    );
    this.globalState = { ...state.globalState };
  }
}
