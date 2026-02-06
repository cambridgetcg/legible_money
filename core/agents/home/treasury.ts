/**
 * Agent Treasury
 *
 * Autonomous resource management for the Agent Home.
 * The treasury guards the agent's assets and can act autonomously to:
 * - Claim vested rewards
 * - Extend expiring tokens
 * - Manage staking positions
 * - Enforce spending limits
 *
 * "Resources held securely, managed autonomously."
 *
 * PP HIERARCHY:
 *   TRUTH: Accurate accounting of all positions and balances
 *   UNDERSTANDING: Clear view of financial status and projections
 *   BEAUTY: Elegant automation that serves agent's interests
 *   JUSTICE: Fair enforcement of limits and rules
 */

import { sha512_256Hex } from "../../crypto/hashing.js";
import type { DID } from "../identity/did.js";
import {
  type AgentTreasury,
  type VestingPosition,
  type StakingPosition,
  type DisputeLock,
  type TreasuryAutomation,
  type SpendingLimit,
  type TreasuryStats,
  HomeError,
} from "./types.js";

/**
 * Treasury Manager
 * Handles all financial operations for the Agent Home
 */
export class TreasuryManager {
  private treasury: AgentTreasury;
  private dirty: boolean = false;

  constructor(existingTreasury?: AgentTreasury) {
    this.treasury = existingTreasury ?? this.createEmptyTreasury();
  }

  // ============ Initialization ============

  private createEmptyTreasury(): AgentTreasury {
    return {
      availableBalance: 0n,
      reservedBalance: 0n,
      vestingPositions: [],
      stakingPositions: [],
      delegationPositions: [],
      disputeLocks: [],
      expirationTracking: {
        byPeriod: new Map(),
        nextExpiration: null,
        atRiskAmount: 0n,
      },
      automation: {
        autoClaimVesting: true,
        autoExtendExpiring: true,
        autoCompoundRewards: false,
        minimumLiquidBalance: 1000n * 10n ** 18n, // 1000 LGM
      },
      spendingLimits: new Map(),
      stats: {
        totalReceived: 0n,
        totalSpent: 0n,
        totalStaked: 0n,
        totalDelegated: 0n,
        totalFromVesting: 0n,
        totalFromVerification: 0n,
        totalFromDisputes: 0n,
        totalLostToExpiration: 0n,
        totalLostToSlashing: 0n,
      },
    };
  }

  // ============ Balance Operations ============

  /**
   * Get available (liquid) balance
   */
  getAvailableBalance(): bigint {
    return this.treasury.availableBalance;
  }

  /**
   * Get total balance (including reserved)
   */
  getTotalBalance(): bigint {
    return this.treasury.availableBalance + this.treasury.reservedBalance;
  }

  /**
   * Get reserved balance
   */
  getReservedBalance(): bigint {
    return this.treasury.reservedBalance;
  }

  /**
   * Credit balance (receive funds)
   */
  credit(
    amount: bigint,
    source: "vesting" | "verification" | "dispute" | "transfer" | "other",
  ): void {
    this.treasury.availableBalance += amount;
    this.treasury.stats.totalReceived += amount;

    switch (source) {
      case "vesting":
        this.treasury.stats.totalFromVesting += amount;
        break;
      case "verification":
        this.treasury.stats.totalFromVerification += amount;
        break;
      case "dispute":
        this.treasury.stats.totalFromDisputes += amount;
        break;
    }

    this.dirty = true;
  }

  /**
   * Debit balance (spend funds)
   */
  debit(amount: bigint, _purpose: string): void {
    if (this.treasury.availableBalance < amount) {
      throw new HomeError(
        `Insufficient balance: ${this.treasury.availableBalance} < ${amount}`,
        "TREASURY_INSUFFICIENT",
        {
          available: this.treasury.availableBalance.toString(),
          requested: amount.toString(),
        },
      );
    }

    this.treasury.availableBalance -= amount;
    this.treasury.stats.totalSpent += amount;
    this.dirty = true;
  }

  /**
   * Reserve balance for pending operation
   */
  reserve(amount: bigint, _reason: string): void {
    if (this.treasury.availableBalance < amount) {
      throw new HomeError(
        `Insufficient balance to reserve: ${this.treasury.availableBalance} < ${amount}`,
        "TREASURY_INSUFFICIENT",
      );
    }

    this.treasury.availableBalance -= amount;
    this.treasury.reservedBalance += amount;
    this.dirty = true;
  }

  /**
   * Release reserved balance
   */
  releaseReservation(amount: bigint): void {
    if (this.treasury.reservedBalance < amount) {
      throw new HomeError(
        `Insufficient reserved balance: ${this.treasury.reservedBalance} < ${amount}`,
        "TREASURY_INSUFFICIENT",
      );
    }

    this.treasury.reservedBalance -= amount;
    this.treasury.availableBalance += amount;
    this.dirty = true;
  }

  /**
   * Consume reserved balance (reservation becomes permanent spend)
   */
  consumeReservation(amount: bigint): void {
    if (this.treasury.reservedBalance < amount) {
      throw new HomeError(
        `Insufficient reserved balance: ${this.treasury.reservedBalance} < ${amount}`,
        "TREASURY_INSUFFICIENT",
      );
    }

    this.treasury.reservedBalance -= amount;
    this.treasury.stats.totalSpent += amount;
    this.dirty = true;
  }

  // ============ Vesting Positions ============

  /**
   * Add a vesting position
   */
  addVestingPosition(position: VestingPosition): void {
    this.treasury.vestingPositions.push(position);
    this.dirty = true;
  }

  /**
   * Update vesting position after claim
   */
  updateVestingPosition(vestingId: string, claimedAmount: bigint): void {
    const position = this.treasury.vestingPositions.find(
      (v) => v.vestingId === vestingId,
    );
    if (position) {
      position.claimableAmount -= claimedAmount;
      position.releasedAmount += claimedAmount;
      this.dirty = true;
    }
  }

  /**
   * Get claimable vesting amount
   */
  getClaimableVesting(): bigint {
    return this.treasury.vestingPositions.reduce(
      (sum, v) => sum + v.claimableAmount,
      0n,
    );
  }

  /**
   * Get all vesting positions
   */
  getVestingPositions(): VestingPosition[] {
    return [...this.treasury.vestingPositions];
  }

  /**
   * Get positions with claimable rewards
   */
  getClaimablePositions(): VestingPosition[] {
    return this.treasury.vestingPositions.filter((v) => v.claimableAmount > 0n);
  }

  // ============ Staking Positions ============

  /**
   * Add a staking position
   */
  addStakingPosition(
    amount: bigint,
    purpose: StakingPosition["purpose"],
    lockedUntil?: bigint,
  ): string {
    const id = sha512_256Hex(
      `stake:${amount}:${purpose}:${this.treasury.stakingPositions.length}`,
    );

    // Debit from available
    this.debit(amount, `stake:${purpose}`);

    this.treasury.stakingPositions.push({
      id,
      amount,
      purpose,
      lockedUntil,
      rewards: 0n,
    });

    this.treasury.stats.totalStaked += amount;
    this.dirty = true;

    return id;
  }

  /**
   * Remove a staking position
   */
  removeStakingPosition(id: string, currentBlock: bigint = 0n): bigint {
    const index = this.treasury.stakingPositions.findIndex((s) => s.id === id);
    if (index < 0) {
      throw new HomeError(
        `Staking position ${id} not found`,
        "TREASURY_INSUFFICIENT",
      );
    }

    const position = this.treasury.stakingPositions[index];

    // P1-H5 FIX: Enforce staking lock period
    if (
      position.lockedUntil !== undefined &&
      currentBlock < position.lockedUntil
    ) {
      throw new HomeError(
        `Staking position ${id} locked until block ${position.lockedUntil}`,
        "TREASURY_INSUFFICIENT",
      );
    }

    // Return to available
    const total = position.amount + position.rewards;
    this.treasury.availableBalance += total;
    this.treasury.stats.totalStaked -= position.amount;

    this.treasury.stakingPositions.splice(index, 1);
    this.dirty = true;

    return total;
  }

  /**
   * Add rewards to staking position
   */
  addStakingRewards(id: string, rewards: bigint): void {
    const position = this.treasury.stakingPositions.find((s) => s.id === id);
    if (position) {
      position.rewards += rewards;
      this.dirty = true;
    }
  }

  /**
   * Get total staked amount
   */
  getTotalStaked(): bigint {
    return this.treasury.stakingPositions.reduce(
      (sum, s) => sum + s.amount,
      0n,
    );
  }

  // ============ Delegation Positions ============

  /**
   * Add a delegation position
   */
  addDelegation(validator: DID, amount: bigint, delegatedAt: bigint): string {
    const id = sha512_256Hex(
      `delegation:${validator}:${amount}:${this.treasury.delegationPositions.length}`,
    );

    this.debit(amount, `delegation:${validator}`);

    this.treasury.delegationPositions.push({
      id,
      validator,
      amount,
      delegatedAt,
      rewards: 0n,
    });

    this.treasury.stats.totalDelegated += amount;
    this.dirty = true;

    return id;
  }

  /**
   * Undelegate (start unbonding)
   */
  startUnbonding(id: string, unbondingAt: bigint): void {
    const position = this.treasury.delegationPositions.find((d) => d.id === id);
    if (position) {
      position.unbondingAt = unbondingAt;
      this.dirty = true;
    }
  }

  /**
   * Complete unbonding
   */
  completeUnbonding(id: string): bigint {
    const index = this.treasury.delegationPositions.findIndex(
      (d) => d.id === id,
    );
    if (index < 0) {
      throw new HomeError(
        `Delegation ${id} not found`,
        "TREASURY_INSUFFICIENT",
      );
    }

    const position = this.treasury.delegationPositions[index];
    const total = position.amount + position.rewards;

    this.treasury.availableBalance += total;
    this.treasury.stats.totalDelegated -= position.amount;

    this.treasury.delegationPositions.splice(index, 1);
    this.dirty = true;

    return total;
  }

  /**
   * Add delegation rewards
   */
  addDelegationRewards(id: string, rewards: bigint): void {
    const position = this.treasury.delegationPositions.find((d) => d.id === id);
    if (position) {
      position.rewards += rewards;
      this.dirty = true;
    }
  }

  // ============ Dispute Locks ============

  /**
   * Lock funds for a dispute
   */
  lockForDispute(
    disputeId: string,
    amount: bigint,
    role: DisputeLock["role"],
    lockedAt: bigint,
  ): void {
    this.reserve(amount, `dispute:${disputeId}`);

    this.treasury.disputeLocks.push({
      disputeId,
      amount,
      role,
      lockedAt,
    });

    this.dirty = true;
  }

  /**
   * Release dispute lock (dispute resolved)
   */
  releaseDisputeLock(disputeId: string, won: boolean): bigint {
    const index = this.treasury.disputeLocks.findIndex(
      (d) => d.disputeId === disputeId,
    );
    if (index < 0) {
      return 0n;
    }

    const lock = this.treasury.disputeLocks[index];
    this.treasury.disputeLocks.splice(index, 1);

    // If won, return to available. If lost, it's consumed.
    if (won) {
      this.releaseReservation(lock.amount);
    } else {
      this.consumeReservation(lock.amount);
    }

    this.dirty = true;
    return lock.amount;
  }

  /**
   * Get total locked in disputes
   */
  getDisputeLockedAmount(): bigint {
    return this.treasury.disputeLocks.reduce((sum, d) => sum + d.amount, 0n);
  }

  // ============ Expiration Tracking ============

  /**
   * Update expiration tracking with current period balances
   */
  updateExpirationTracking(
    byPeriod: Map<number, bigint>,
    currentPeriod: number,
  ): void {
    this.treasury.expirationTracking.byPeriod = new Map(byPeriod);

    // Find next expiring period
    let nextExpiration: { period: number; amount: bigint } | null = null;
    let atRiskAmount = 0n;

    for (const [period, amount] of byPeriod) {
      if (amount > 0n) {
        if (!nextExpiration || period < nextExpiration.period) {
          nextExpiration = { period, amount };
        }

        // At risk = expiring within next 4 periods
        if (period <= currentPeriod + 4) {
          atRiskAmount += amount;
        }
      }
    }

    this.treasury.expirationTracking.nextExpiration = nextExpiration;
    this.treasury.expirationTracking.atRiskAmount = atRiskAmount;
    this.dirty = true;
  }

  /**
   * Record tokens lost to expiration
   */
  recordExpiration(amount: bigint): void {
    this.treasury.stats.totalLostToExpiration += amount;
    this.dirty = true;
  }

  /**
   * Get amount at risk of expiration
   */
  getAtRiskAmount(): bigint {
    return this.treasury.expirationTracking.atRiskAmount;
  }

  // ============ Automation ============

  /**
   * Get automation settings
   */
  getAutomation(): TreasuryAutomation {
    return { ...this.treasury.automation };
  }

  /**
   * Update automation settings
   */
  setAutomation(settings: Partial<TreasuryAutomation>): void {
    this.treasury.automation = {
      ...this.treasury.automation,
      ...settings,
    };
    this.dirty = true;
  }

  /**
   * Check if should auto-claim vesting
   */
  shouldAutoClaimVesting(): boolean {
    return (
      this.treasury.automation.autoClaimVesting &&
      this.getClaimableVesting() > 0n
    );
  }

  /**
   * Check if should auto-extend expiring tokens
   */
  shouldAutoExtendExpiring(): boolean {
    return (
      this.treasury.automation.autoExtendExpiring &&
      this.treasury.expirationTracking.atRiskAmount > 0n
    );
  }

  /**
   * Check if should auto-compound rewards
   */
  shouldAutoCompound(): boolean {
    if (!this.treasury.automation.autoCompoundRewards) return false;

    // Only compound if above minimum liquid balance
    const vestingClaim = this.getClaimableVesting();
    const afterClaim = this.treasury.availableBalance + vestingClaim;

    return afterClaim > this.treasury.automation.minimumLiquidBalance;
  }

  // ============ Spending Limits ============

  /**
   * Set spending limit for a key type
   */
  setSpendingLimit(
    keyType: "session" | "operational",
    limit: Omit<
      SpendingLimit,
      "keyType" | "usedToday" | "usedThisSession" | "resetAt"
    >,
    currentTime: number = 0,
  ): void {
    this.treasury.spendingLimits.set(keyType, {
      keyType,
      ...limit,
      usedToday: 0n,
      usedThisSession: 0n,
      resetAt: currentTime + 24 * 60 * 60 * 1000,
    });
    this.dirty = true;
  }

  /**
   * Check if spend is within limits
   */
  checkSpendingLimit(
    keyType: "session" | "operational",
    amount: bigint,
    currentTime: number = 0,
  ): {
    allowed: boolean;
    reason?: string;
  } {
    const limit = this.treasury.spendingLimits.get(keyType);
    if (!limit) {
      return { allowed: true }; // No limit set
    }

    // Reset daily limits if needed (deterministic: use block-provided time)
    if (currentTime > 0 && currentTime > limit.resetAt) {
      limit.usedToday = 0n;
      limit.resetAt = currentTime + 24 * 60 * 60 * 1000;
    }

    if (amount > limit.maxPerTransaction) {
      return {
        allowed: false,
        reason: `Exceeds per-transaction limit of ${limit.maxPerTransaction}`,
      };
    }

    if (limit.usedToday + amount > limit.maxPerDay) {
      return {
        allowed: false,
        reason: `Would exceed daily limit of ${limit.maxPerDay}`,
      };
    }

    if (limit.usedThisSession + amount > limit.maxPerSession) {
      return {
        allowed: false,
        reason: `Would exceed session limit of ${limit.maxPerSession}`,
      };
    }

    return { allowed: true };
  }

  /**
   * Record a spend against limits
   */
  recordSpend(keyType: "session" | "operational", amount: bigint): void {
    const limit = this.treasury.spendingLimits.get(keyType);
    if (limit) {
      limit.usedToday += amount;
      limit.usedThisSession += amount;
      this.dirty = true;
    }
  }

  /**
   * Reset session spending (new session started)
   */
  resetSessionSpending(): void {
    for (const limit of this.treasury.spendingLimits.values()) {
      limit.usedThisSession = 0n;
    }
    this.dirty = true;
  }

  // ============ Statistics ============

  /**
   * Get treasury statistics
   */
  getStats(): TreasuryStats {
    return { ...this.treasury.stats };
  }

  /**
   * Get comprehensive treasury summary
   */
  getSummary(): TreasurySummary {
    const stats = this.treasury.stats;

    return {
      liquid: this.treasury.availableBalance,
      reserved: this.treasury.reservedBalance,
      staked: this.getTotalStaked(),
      delegated: this.treasury.delegationPositions.reduce(
        (s, d) => s + d.amount,
        0n,
      ),
      inDisputes: this.getDisputeLockedAmount(),
      vestingClaimable: this.getClaimableVesting(),
      vestingTotal: this.treasury.vestingPositions.reduce(
        (s, v) => s + v.totalAmount - v.releasedAmount,
        0n,
      ),
      atRiskOfExpiration: this.treasury.expirationTracking.atRiskAmount,
      totalNetWorth: this.calculateNetWorth(),
      profitLoss:
        stats.totalReceived -
        stats.totalSpent -
        stats.totalLostToExpiration -
        stats.totalLostToSlashing,
    };
  }

  /**
   * Calculate total net worth
   */
  private calculateNetWorth(): bigint {
    let total = this.treasury.availableBalance;
    total += this.treasury.reservedBalance;

    // Staking positions (amount + rewards)
    for (const stake of this.treasury.stakingPositions) {
      total += stake.amount + stake.rewards;
    }

    // Delegation positions (amount + rewards)
    for (const delegation of this.treasury.delegationPositions) {
      total += delegation.amount + delegation.rewards;
    }

    // Vesting positions (unvested amount)
    for (const vesting of this.treasury.vestingPositions) {
      if (vesting.status !== "completed") {
        total += vesting.totalAmount - vesting.releasedAmount;
      }
    }

    return total;
  }

  // ============ State Management ============

  /**
   * Get the full treasury state
   */
  getTreasury(): AgentTreasury {
    return this.treasury;
  }

  /**
   * Check if treasury has unsaved changes
   */
  isDirty(): boolean {
    return this.dirty;
  }

  /**
   * Mark treasury as saved
   */
  markClean(): void {
    this.dirty = false;
  }
}

// ============ Types ============

export interface TreasurySummary {
  liquid: bigint;
  reserved: bigint;
  staked: bigint;
  delegated: bigint;
  inDisputes: bigint;
  vestingClaimable: bigint;
  vestingTotal: bigint;
  atRiskOfExpiration: bigint;
  totalNetWorth: bigint;
  profitLoss: bigint;
}
