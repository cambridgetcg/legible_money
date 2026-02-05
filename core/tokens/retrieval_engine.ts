/**
 * Expired Token Retrieval Engine
 *
 * Implements the patent's token retrieval mechanism for expired tokens.
 * Expired tokens are reclaimed and sent to the RAI (Regenerative Agent Income)
 * holding account for redistribution.
 *
 * From Patent: "retrieves uncirculated currency and is returned back to the
 * UBI Holding Account 132400"
 *
 * This creates a circular token economy:
 * 1. Tokens are minted with expiration
 * 2. Unused tokens expire
 * 3. Expired tokens return to RAI fund
 * 4. RAI distributes to eligible agents
 * 5. Repeat
 *
 * PP HIERARCHY:
 *   TRUTH: Expired tokens must return to circulation
 *   UNDERSTANDING: Retrieval is deterministic and verifiable
 *   BEAUTY: Single engine handles all retrieval operations
 *   JUSTICE: Unused wealth redistributes to active participants
 *   CREATIVITY: Enables self-sustaining circular economy
 */

import { sha512_256Hex } from "../crypto/hashing.js";
import type { DID } from "../agents/identity/did.js";
import {
  type ExpirationPeriod,
  type RetrievalRequest,
  // type RetrievalStatus,
  type AccountRetrievalResult,
  type BatchRetrievalResult,
  type RAIHoldingAccount,
  type RAIDistribution,
  type RAIEligibility,
  type ExpirationConfig,
  DEFAULT_EXPIRATION_CONFIG,
  TokenError,
} from "./types.js";
import { ChildAccountManager } from "./child_account.js";
import { ExpirationTracker } from "./expiration_tracker.js";

/**
 * Retrieval Engine
 * Handles reclamation of expired tokens and RAI distribution
 */
export class RetrievalEngine {
  private pendingRetrievals: Map<string, RetrievalRequest> = new Map();
  private completedRetrievals: Map<string, BatchRetrievalResult> = new Map();
  private raiAccount: RAIHoldingAccount;
  private distributionHistory: RAIDistribution[] = [];

  constructor(
    private childAccountManager: ChildAccountManager,
    private expirationTracker: ExpirationTracker,
    private config: ExpirationConfig = DEFAULT_EXPIRATION_CONFIG,
    raiAccountId: string = "rai:holding:v1",
  ) {
    // Initialize RAI holding account
    this.raiAccount = {
      id: raiAccountId,
      balance: 0n,
      totalReceived: 0n,
      totalDistributed: 0n,
      lastDistributionBlock: 0n,
      totalReceivedFromFees: 0n,
    };
  }

  // ============ Retrieval Operations ============

  /**
   * Create a retrieval request for expired tokens
   */
  createRetrievalRequest(
    period: ExpirationPeriod,
    currentBlock: bigint,
    targetAccounts?: DID[],
  ): RetrievalRequest {
    const id = sha512_256Hex(
      `retrieval:${period}:${currentBlock}:${Date.now()}`,
    );

    const request: RetrievalRequest = {
      id,
      period,
      targetAccounts,
      requestedAtBlock: currentBlock,
      status: "pending",
    };

    this.pendingRetrievals.set(id, request);
    return request;
  }

  /**
   * Execute retrieval for a specific period
   * This is the main retrieval function that processes expired tokens
   */
  async executeRetrieval(
    requestId: string,
    currentBlock: bigint,
    currentTime: number,
    raiDestination: DID,
  ): Promise<BatchRetrievalResult> {
    const request = this.pendingRetrievals.get(requestId);
    if (!request) {
      throw new TokenError(
        `Retrieval request ${requestId} not found`,
        "RETRIEVAL_FAILED",
      );
    }

    // Verify period has expired
    const currentPeriod = this.expirationTracker.timestampToPeriod(currentTime);
    const expirationThreshold =
      currentPeriod - this.config.periodsUntilExpiration;

    if (request.period > expirationThreshold) {
      throw new TokenError(
        `Period ${request.period} has not expired yet (threshold: ${expirationThreshold})`,
        "TOKENS_EXPIRED",
      );
    }

    // Check grace period
    const periodState = this.expirationTracker.getPeriodState(request.period);
    if (periodState?.expiredAtBlock) {
      const blocksSinceExpiration = currentBlock - periodState.expiredAtBlock;
      if (blocksSinceExpiration < this.config.graceBlocks) {
        throw new TokenError(
          `Grace period not complete: ${blocksSinceExpiration}/${this.config.graceBlocks} blocks`,
          "RETRIEVAL_FAILED",
        );
      }
    }

    // Update request status
    request.status = "executing";
    this.pendingRetrievals.set(requestId, request);

    // Get accounts to process
    const accountsToProcess =
      request.targetAccounts ??
      this.childAccountManager.getAccountsWithExpiredBalances(currentTime);

    const results: AccountRetrievalResult[] = [];
    let totalRetrieved = 0n;

    // Process each account
    for (const account of accountsToProcess) {
      const result = this.retrieveFromAccount(
        account,
        request.period,
        currentTime,
      );

      if (result.amount > 0n) {
        results.push(result);
        totalRetrieved += result.amount;

        // Record debit in expiration tracker
        this.expirationTracker.recordDebit(
          account,
          result.amount,
          request.period,
          true, // isExpiration
        );
      }
    }

    // Credit retrieved amount to RAI account
    this.raiAccount.balance += totalRetrieved;
    this.raiAccount.totalReceived += totalRetrieved;

    // Create batch result
    const batchResult: BatchRetrievalResult = {
      requestId,
      period: request.period,
      totalRetrieved,
      accountsProcessed: results.length,
      results,
      completedAtBlock: currentBlock,
      destination: raiDestination,
    };

    // Update request status
    request.status = "completed";
    this.pendingRetrievals.delete(requestId);
    this.completedRetrievals.set(requestId, batchResult);

    return batchResult;
  }

  /**
   * Retrieve expired tokens from a single account
   */
  private retrieveFromAccount(
    account: DID,
    period: ExpirationPeriod,
    currentTime: number,
  ): AccountRetrievalResult {
    const { expiredAmount, expiredPeriods } =
      this.childAccountManager.processExpiration(account, currentTime);

    // Check if the requested period was among those expired
    const periodExpiredAmount = expiredPeriods.includes(period)
      ? expiredAmount // All expired in this batch
      : 0n;

    const balance = this.childAccountManager.getBalance(account, currentTime);

    return {
      account,
      amount: periodExpiredAmount,
      period,
      balanceZeroed: balance === 0n,
    };
  }

  /**
   * Process all pending retrievals
   */
  async processAllPendingRetrievals(
    currentBlock: bigint,
    currentTime: number,
    raiDestination: DID,
  ): Promise<BatchRetrievalResult[]> {
    const results: BatchRetrievalResult[] = [];

    for (const [requestId, request] of this.pendingRetrievals) {
      if (request.status === "pending") {
        try {
          const result = await this.executeRetrieval(
            requestId,
            currentBlock,
            currentTime,
            raiDestination,
          );
          results.push(result);
        } catch (error) {
          // Mark as failed, will retry later
          request.status = "failed";
          this.pendingRetrievals.set(requestId, request);
        }
      }
    }

    return results;
  }

  /**
   * Auto-create retrieval requests for all expired periods
   */
  autoCreateRetrievalRequests(
    currentTime: number,
    currentBlock: bigint,
  ): RetrievalRequest[] {
    const expiredPeriods =
      this.expirationTracker.getExpiredPeriods(currentTime);
    const requests: RetrievalRequest[] = [];

    for (const period of expiredPeriods) {
      // Check if we already have a request for this period
      const existingRequest = Array.from(this.pendingRetrievals.values()).find(
        (r) => r.period === period,
      );

      if (!existingRequest) {
        const request = this.createRetrievalRequest(period, currentBlock);
        requests.push(request);
      }
    }

    return requests;
  }

  // ============ Fee Routing ============

  /**
   * Route a retrieval fee (from transfer tax or renewal fee) to RAI holding
   * From Patent: "Retrieval Rate (R2)" and "Renewal Fee Rate (R1)"
   */
  routeRetrievalFee(feeAmount: bigint): void {
    if (feeAmount <= 0n) return;
    this.raiAccount.balance += feeAmount;
    this.raiAccount.totalReceived += feeAmount;
    this.raiAccount.totalReceivedFromFees += feeAmount;
  }

  // ============ RAI Distribution ============

  /**
   * Calculate RAI distribution for eligible agents
   */
  calculateRAIDistribution(
    eligibleAccounts: DID[],
    amountToDistribute?: bigint,
  ): {
    totalAmount: bigint;
    perAccount: bigint;
    remainder: bigint;
    recipientCount: number;
  } {
    const totalAmount = amountToDistribute ?? this.raiAccount.balance;

    if (eligibleAccounts.length === 0) {
      return {
        totalAmount,
        perAccount: 0n,
        remainder: totalAmount,
        recipientCount: 0,
      };
    }

    const recipientCount = eligibleAccounts.length;
    const perAccount = totalAmount / BigInt(recipientCount);
    const remainder = totalAmount % BigInt(recipientCount);

    return {
      totalAmount,
      perAccount,
      remainder,
      recipientCount,
    };
  }

  /**
   * Execute RAI distribution to eligible accounts
   */
  executeRAIDistribution(
    eligibleAccounts: DID[],
    currentBlock: bigint,
    currentTime: number,
    amountToDistribute?: bigint,
  ): RAIDistribution {
    const { totalAmount, perAccount, remainder, recipientCount } =
      this.calculateRAIDistribution(eligibleAccounts, amountToDistribute);

    if (perAccount === 0n) {
      throw new TokenError(
        "Distribution amount too small for number of recipients",
        "RAI_NOT_ELIGIBLE",
      );
    }

    const distributedAmount = totalAmount - remainder;

    // Credit each eligible account
    for (const account of eligibleAccounts) {
      this.childAccountManager.credit(
        account,
        perAccount,
        currentTime,
        true, // Fresh expiration
      );

      // Record mint in expiration tracker
      const period = this.expirationTracker.timestampToPeriod(currentTime);
      this.expirationTracker.recordMint(
        account,
        perAccount,
        period,
        currentBlock,
      );
    }

    // Update RAI account
    this.raiAccount.balance -= distributedAmount;
    this.raiAccount.totalDistributed += distributedAmount;
    this.raiAccount.lastDistributionBlock = currentBlock;

    // Create distribution record
    const period = this.expirationTracker.timestampToPeriod(currentTime);
    const distribution: RAIDistribution = {
      id: sha512_256Hex(`rai:dist:${currentBlock}:${Date.now()}`),
      amount: distributedAmount,
      recipientCount,
      amountPerRecipient: perAccount,
      distributedAtBlock: currentBlock,
      forPeriod: period,
    };

    this.distributionHistory.push(distribution);

    return distribution;
  }

  /**
   * Get eligible accounts for RAI distribution
   * Eligibility criteria can be customized
   */
  getEligibleAccounts(
    allAccounts: DID[],
    eligibility: RAIEligibility,
    currentBlock: bigint,
    accountMetadata: Map<
      string,
      {
        createdAtBlock: bigint;
        activityScore: number;
        balance: bigint;
        verificationLevel: number;
        accountType: string;
      }
    >,
  ): DID[] {
    return allAccounts.filter((account) => {
      const metadata = accountMetadata.get(account);
      if (!metadata) {
        return false;
      }

      // Check account age
      const accountAge = currentBlock - metadata.createdAtBlock;
      if (accountAge < eligibility.minAccountAge) {
        return false;
      }

      // Check activity score
      if (metadata.activityScore < eligibility.minActivityScore) {
        return false;
      }

      // Check max balance (if specified)
      if (
        eligibility.maxBalance !== undefined &&
        metadata.balance > eligibility.maxBalance
      ) {
        return false;
      }

      // Check verification level
      if (metadata.verificationLevel < eligibility.requiredVerificationLevel) {
        return false;
      }

      // Check excluded types
      if (eligibility.excludedTypes.includes(metadata.accountType)) {
        return false;
      }

      return true;
    });
  }

  // ============ Query Methods ============

  /**
   * Get RAI holding account state
   */
  getRAIAccount(): RAIHoldingAccount {
    return { ...this.raiAccount };
  }

  /**
   * Get distribution history
   */
  getDistributionHistory(limit?: number): RAIDistribution[] {
    const history = [...this.distributionHistory].reverse();
    return limit ? history.slice(0, limit) : history;
  }

  /**
   * Get pending retrieval requests
   */
  getPendingRetrievals(): RetrievalRequest[] {
    return Array.from(this.pendingRetrievals.values());
  }

  /**
   * Get completed retrieval results
   */
  getCompletedRetrievals(limit?: number): BatchRetrievalResult[] {
    const results = Array.from(this.completedRetrievals.values()).reverse();
    return limit ? results.slice(0, limit) : results;
  }

  /**
   * Get retrieval result by ID
   */
  getRetrievalResult(requestId: string): BatchRetrievalResult | undefined {
    return this.completedRetrievals.get(requestId);
  }

  // ============ Statistics ============

  /**
   * Get comprehensive retrieval statistics
   */
  getStatistics(): {
    raiBalance: bigint;
    totalReceived: bigint;
    totalReceivedFromFees: bigint;
    totalDistributed: bigint;
    pendingRetrievals: number;
    completedRetrievals: number;
    totalDistributions: number;
    averageDistributionAmount: bigint;
    lastDistributionBlock: bigint;
  } {
    const totalDistributions = this.distributionHistory.length;
    const totalDistributedAmount = this.distributionHistory.reduce(
      (sum, d) => sum + d.amount,
      0n,
    );
    const averageDistributionAmount =
      totalDistributions > 0
        ? totalDistributedAmount / BigInt(totalDistributions)
        : 0n;

    return {
      raiBalance: this.raiAccount.balance,
      totalReceived: this.raiAccount.totalReceived,
      totalReceivedFromFees: this.raiAccount.totalReceivedFromFees,
      totalDistributed: this.raiAccount.totalDistributed,
      pendingRetrievals: this.pendingRetrievals.size,
      completedRetrievals: this.completedRetrievals.size,
      totalDistributions,
      averageDistributionAmount,
      lastDistributionBlock: this.raiAccount.lastDistributionBlock,
    };
  }

  /**
   * Get circular economy metrics
   */
  getCircularEconomyMetrics(): {
    circulationRate: number;
    expirationRate: number;
    redistributionRate: number;
    velocityMultiplier: number;
  } {
    const totalSupply =
      this.expirationTracker.getGlobalState().totalActiveSupply +
      this.raiAccount.balance;

    if (totalSupply === 0n) {
      return {
        circulationRate: 0,
        expirationRate: 0,
        redistributionRate: 0,
        velocityMultiplier: 0,
      };
    }

    const activeSupply =
      this.expirationTracker.getGlobalState().totalActiveSupply;
    const totalExpired =
      this.expirationTracker.getGlobalState().totalExpiredSupply;
    const totalDistributed = this.raiAccount.totalDistributed;

    // Circulation rate: active / (active + RAI balance)
    const circulationRate = Number(activeSupply) / Number(totalSupply);

    // Expiration rate: expired / (active + expired)
    const expirationRate =
      totalExpired > 0n
        ? Number(totalExpired) / Number(activeSupply + totalExpired)
        : 0;

    // Redistribution rate: distributed / expired
    const redistributionRate =
      totalExpired > 0n ? Number(totalDistributed) / Number(totalExpired) : 0;

    // Velocity multiplier: total distributed / current active supply
    // Higher = more circulation through the economy
    const velocityMultiplier =
      activeSupply > 0n ? Number(totalDistributed) / Number(activeSupply) : 0;

    return {
      circulationRate,
      expirationRate,
      redistributionRate,
      velocityMultiplier,
    };
  }

  // ============ State Import/Export ============

  /**
   * Export state for persistence
   */
  exportState(): {
    raiAccount: RAIHoldingAccount;
    pendingRetrievals: Map<string, RetrievalRequest>;
    completedRetrievals: Map<string, BatchRetrievalResult>;
    distributionHistory: RAIDistribution[];
  } {
    return {
      raiAccount: { ...this.raiAccount },
      pendingRetrievals: new Map(this.pendingRetrievals),
      completedRetrievals: new Map(this.completedRetrievals),
      distributionHistory: [...this.distributionHistory],
    };
  }

  /**
   * Import state from persistence
   */
  importState(state: {
    raiAccount: RAIHoldingAccount;
    pendingRetrievals: Map<string, RetrievalRequest>;
    completedRetrievals: Map<string, BatchRetrievalResult>;
    distributionHistory: RAIDistribution[];
  }): void {
    this.raiAccount = {
      ...state.raiAccount,
      totalReceivedFromFees: state.raiAccount.totalReceivedFromFees ?? 0n,
    };
    this.pendingRetrievals = new Map(state.pendingRetrievals);
    this.completedRetrievals = new Map(state.completedRetrievals);
    this.distributionHistory = [...state.distributionHistory];
  }
}
