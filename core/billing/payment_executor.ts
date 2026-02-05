/**
 * Payment Distribution Executor
 *
 * Executes calculated payment distributions by transferring funds between accounts.
 * This is the critical bridge between billing calculation and actual state mutation.
 *
 * Flow: calculateDistribution() → PaymentDistribution → executePaymentDistribution() → state transfers
 *
 * All transfers execute atomically within a state transaction.
 * If any transfer fails, the entire distribution is rolled back.
 */

import type { PaymentDistribution } from "./types.js";
import type { StateManager } from "../../state/state_manager.js";
import type { DID } from "../agents/identity/did.js";
import { splitEvidenceTax } from "./reward_router.js";

// System accounts (must exist in genesis state)
const BURN_ADDRESS = "did:lgm:burn";
const PROTOCOL_TREASURY = "did:lgm:protocol-treasury";

// ============ Types ============

export interface PaymentTransfer {
  from: string;
  to: string;
  amount: bigint;
  reason: string;
}

export interface PaymentExecutionResult {
  success: boolean;
  transfers: PaymentTransfer[];
  totalTransferred: bigint;
  researchFunded: bigint;
  errors: string[];
}

// ============ Executor ============

/**
 * Execute a payment distribution by applying state transfers.
 *
 * Deducts total cost from caller and credits:
 * - API provider (for serving the query)
 * - Knowledge contributors (for facts used)
 * - Research treasury (22% for truth discovery)
 * - Protocol treasury + burn (protocol share)
 *
 * Uses atomic transaction - all or nothing.
 */
export function executePaymentDistribution(
  distribution: PaymentDistribution,
  callerDid: DID,
  state: StateManager,
  _blockNumber: bigint,
): PaymentExecutionResult {
  const transfers: PaymentTransfer[] = [];
  const errors: string[] = [];
  let researchFunded = 0n;

  // Calculate total cost by summing all distribution amounts
  const totalCost = calculateTotalCost(distribution);

  if (totalCost === 0n) {
    return {
      success: true,
      transfers: [],
      totalTransferred: 0n,
      researchFunded: 0n,
      errors: [],
    };
  }

  // Check caller balance before starting transaction
  const callerBalance = state.getBalance(callerDid);
  if (callerBalance < totalCost) {
    return {
      success: false,
      transfers: [],
      totalTransferred: 0n,
      researchFunded: 0n,
      errors: [
        `Insufficient balance: has ${callerBalance}, needs ${totalCost}`,
      ],
    };
  }

  state.beginTransaction();

  try {
    // Uses deductBalance/creditBalance to avoid nonce inflation.
    // Each state.transfer() increments nonce — with N distribution legs
    // that would mean N nonce increments per payment.

    // 1. Credit API provider
    if (distribution.apiProvider.amount > 0n) {
      state.deductBalance(callerDid, distribution.apiProvider.amount);
      state.creditBalance(
        distribution.apiProvider.did,
        distribution.apiProvider.amount,
      );
      transfers.push({
        from: callerDid,
        to: distribution.apiProvider.did,
        amount: distribution.apiProvider.amount,
        reason: "api_service",
      });
    }

    // 2. Credit knowledge contributors (citation rewards)
    for (const reward of distribution.knowledgeRewards) {
      if (reward.amount > 0n) {
        state.deductBalance(callerDid, reward.amount);
        state.creditBalance(reward.submitter, reward.amount);
        transfers.push({
          from: callerDid,
          to: reward.submitter,
          amount: reward.amount,
          reason: reward.reason,
        });
      }
    }

    // 3. Research fund (22% split across 5 treasury funds)
    if (distribution.research && distribution.research.amount > 0n) {
      const treasurySplit = splitEvidenceTax(distribution.research.amount);
      for (const entry of treasurySplit) {
        if (entry.amount > 0n) {
          state.deductBalance(callerDid, entry.amount);
          state.creditBalance(entry.did, entry.amount);
          transfers.push({
            from: callerDid,
            to: entry.did,
            amount: entry.amount,
            reason: `treasury_${entry.fund}`,
          });
        }
      }
      researchFunded = distribution.research.amount;
    }

    // 4. Protocol burn (deflationary)
    if (distribution.protocol.burned > 0n) {
      state.deductBalance(callerDid, distribution.protocol.burned);
      state.creditBalance(BURN_ADDRESS, distribution.protocol.burned);
      transfers.push({
        from: callerDid,
        to: BURN_ADDRESS,
        amount: distribution.protocol.burned,
        reason: "protocol_burn",
      });
    }

    // 5. Protocol treasury
    if (distribution.protocol.treasury > 0n) {
      state.deductBalance(callerDid, distribution.protocol.treasury);
      state.creditBalance(PROTOCOL_TREASURY, distribution.protocol.treasury);
      transfers.push({
        from: callerDid,
        to: PROTOCOL_TREASURY,
        amount: distribution.protocol.treasury,
        reason: "protocol_treasury",
      });
    }

    state.commitTransaction();

    return {
      success: true,
      transfers,
      totalTransferred: totalCost,
      researchFunded,
      errors: [],
    };
  } catch (error) {
    state.rollbackTransaction();
    errors.push(error instanceof Error ? error.message : String(error));
    return {
      success: false,
      transfers: [],
      totalTransferred: 0n,
      researchFunded: 0n,
      errors,
    };
  }
}

/**
 * Check if a caller can afford a query given a distribution.
 */
export function canAffordDistribution(
  callerBalance: bigint,
  distribution: PaymentDistribution,
  safetyMargin: number = 1.0,
): boolean {
  const totalCost = calculateTotalCost(distribution);
  const requiredBalance =
    safetyMargin > 1.0
      ? (totalCost * BigInt(Math.floor(safetyMargin * 100))) / 100n
      : totalCost;
  return callerBalance >= requiredBalance;
}

/**
 * Calculate total cost from a distribution (sum of all outflows).
 */
export function calculateTotalCost(distribution: PaymentDistribution): bigint {
  let total = distribution.apiProvider.amount;

  for (const reward of distribution.knowledgeRewards) {
    total += reward.amount;
  }

  total += distribution.protocol.burned;
  total += distribution.protocol.treasury;

  if (distribution.research) {
    total += distribution.research.amount;
  }

  return total;
}
