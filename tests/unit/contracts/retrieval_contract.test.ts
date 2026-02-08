/**
 * Genesis Retrieval Contract Tests
 *
 * Tests the self-invoking RAI retrieval/distribution system:
 * 1. Genesis schedule registration with SPU
 * 2. executeSweepExpired dispatch + real sweep behavior
 * 3. executeDistributeRAI distribution logic
 * 4. Full pipeline: sweep expired → fund RAI → distribute → verify
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import { StateManager } from "../../../state/state_manager.js";
import {
  executeTransaction,
  FEE_POOL_DID,
  REWARD_POOL_DID,
  PROTOCOL_TREASURY_DID,
} from "../../../state/transitions.js";
import type {
  Transaction,
  ConsensusParams,
} from "../../../consensus/pot/types.js";
import { SYSTEM_CONTRACTS, TREASURY } from "../../../core/constants.js";
import { DEFAULT_EXPIRATION_CONFIG } from "../../../core/tokens/types.js";
import { registerGenesisContracts } from "../../../consensus/genesis/system_contracts.js";
import { sha512_256Hex } from "../../../core/crypto/hashing.js";

// Minimum account age for RAI eligibility (matches transitions.ts MIN_RAI_AGE_BLOCKS)
const MIN_RAI_AGE = 7n * 34272n; // ~1 week in blocks

// ============ Test Helpers ============

function createTestState(): StateManager {
  return new StateManager();
}

function createTestParams(): ConsensusParams {
  return {
    targetBlockTimeMs: 2521,
    minValidatorStake: 1000n * 10n ** 18n,
    minClaimStake: 10n * 10n ** 18n,
    validatorQuorum: 0.66,
    verificationRoundBlocks: 20,
    commitPhaseBlocks: 6,
    revealPhaseBlocks: 6,
    epochBlocks: 100,
    maxValidators: 100,
    slashFractionDowntime: 0.01,
    slashFractionDoubleSign: 0.05,
    unbondingPeriod: 1000n,
  } as unknown as ConsensusParams;
}

/**
 * Create a scheduled_execution transaction targeting a system contract.
 * Uses sender="system" + fee=0n so validateTransaction bypasses sender checks.
 */
function createScheduledExecutionTx(
  targetContract: string,
  functionSelector: Uint8Array,
  timestamp: number = Date.now(),
): Transaction {
  return {
    type: "scheduled_execution",
    sender: "system",
    fee: 0n,
    nonce: 0n,
    timestamp,
    signature: "system",
    payload: {
      type: "scheduled_execution",
      processId: "test-process-1",
      executionIndex: 0,
      targetContract,
      functionSelector,
      proof: {
        processStateHash: sha512_256Hex("test-state"),
        conditionProof: {
          timestamp,
          stateProofs: [],
        },
        executionNonce: 0n,
        signature: "system",
      },
    },
  } as unknown as Transaction;
}

/**
 * Set up a StateManager with system accounts needed for retrieval tests.
 */
function setupSystemAccounts(state: StateManager): void {
  // RAI holding account (receives swept tokens, distributes to agents)
  state.createAccount(TREASURY.ADDRESSES.RAI_HOLDING, "rai-holding-hash", 0n);

  // System contract account (target of scheduled executions)
  state.createAccount(
    SYSTEM_CONTRACTS.RAI_RETRIEVAL.DID,
    "system-contract-hash",
    0n,
  );

  // Fee pool (required by transaction execution)
  state.createAccount(FEE_POOL_DID, "fee-pool-hash", 0n);

  // Reward pool
  state.createAccount(
    REWARD_POOL_DID,
    "reward-pool-hash",
    1_000_000n * 10n ** 18n,
  );

  // Protocol treasury
  state.createAccount(PROTOCOL_TREASURY_DID, "treasury-hash", 0n);

  // Treasury addresses (excluded from distribution)
  state.createAccount(
    TREASURY.ADDRESSES.RESEARCH,
    "research-hash",
    100n * 10n ** 18n,
  );
  state.createAccount(
    TREASURY.ADDRESSES.RESEARCH_FUND,
    "research-fund-hash",
    0n,
  );
  state.createAccount(TREASURY.ADDRESSES.FOUNDATION, "foundation-hash", 0n);
  state.createAccount(TREASURY.ADDRESSES.COMMUNITY, "community-hash", 0n);
  state.createAccount(TREASURY.ADDRESSES.DEVELOPERS, "developers-hash", 0n);
  state.createAccount(TREASURY.ADDRESSES.RESERVE, "reserve-hash", 0n);
}

// ============ Tests ============

describe.sequential("Genesis Retrieval Contract", () => {
  // ============ registerGenesisContracts ============

  describe.sequential("registerGenesisContracts", () => {
    it("should register two schedules with SPU", async () => {
      const mockProcessScheduleMessage = vi.fn().mockResolvedValue({
        id: "mock-process-id",
        sourceMessageId: "mock-source",
        creator: "system",
        condition: {},
        watchedStateKeys: [],
        status: "active",
        executionCount: 0,
        perpetual: true,
      });

      const mockAutonomy = {
        getSPU: () => ({
          processScheduleMessage: mockProcessScheduleMessage,
        }),
      } as any;

      const genesisTime = Date.now();
      const result = await registerGenesisContracts(mockAutonomy, genesisTime);

      expect(mockProcessScheduleMessage).toHaveBeenCalledTimes(2);
      expect(result.retrievalProcessId).toBe("mock-process-id");
      expect(result.distributionProcessId).toBe("mock-process-id");
    });

    it("should use periodic time condition with correct interval", async () => {
      const calls: any[] = [];
      const mockProcessScheduleMessage = vi.fn().mockImplementation((msg) => {
        calls.push(msg);
        return Promise.resolve({ id: `process-${calls.length}` });
      });

      const mockAutonomy = {
        getSPU: () => ({
          processScheduleMessage: mockProcessScheduleMessage,
        }),
      } as any;

      const genesisTime = 1700000000000;
      await registerGenesisContracts(mockAutonomy, genesisTime);

      const periodMs = DEFAULT_EXPIRATION_CONFIG.periodDurationMs;

      // First call: retrieval schedule
      const retrievalMsg = calls[0];
      expect(retrievalMsg.scheduleCondition.time.type).toBe("periodic");
      expect(retrievalMsg.scheduleCondition.time.interval).toBe(periodMs);
      expect(retrievalMsg.scheduleCondition.time.startTime).toBe(
        genesisTime + periodMs,
      );
      expect(retrievalMsg.scheduleType).toBe("expiration_clawback");

      // Second call: distribution schedule (1hr offset)
      const distributionMsg = calls[1];
      expect(distributionMsg.scheduleCondition.time.type).toBe("periodic");
      expect(distributionMsg.scheduleCondition.time.interval).toBe(periodMs);
      expect(distributionMsg.scheduleCondition.time.startTime).toBe(
        genesisTime + periodMs + 3_600_000,
      );
      expect(distributionMsg.scheduleType).toBe("rai_distribution");
    });

    it("should use 'system' sender to bypass signature verification", async () => {
      const calls: any[] = [];
      const mockProcessScheduleMessage = vi.fn().mockImplementation((msg) => {
        calls.push(msg);
        return Promise.resolve({ id: `process-${calls.length}` });
      });

      const mockAutonomy = {
        getSPU: () => ({
          processScheduleMessage: mockProcessScheduleMessage,
        }),
      } as any;

      await registerGenesisContracts(mockAutonomy, Date.now());

      for (const msg of calls) {
        expect(msg.senderAddress).toBe("system");
      }
    });

    it("should set perpetual=true with no maxExecutions", async () => {
      const calls: any[] = [];
      const mockProcessScheduleMessage = vi.fn().mockImplementation((msg) => {
        calls.push(msg);
        return Promise.resolve({ id: `process-${calls.length}` });
      });

      const mockAutonomy = {
        getSPU: () => ({
          processScheduleMessage: mockProcessScheduleMessage,
        }),
      } as any;

      await registerGenesisContracts(mockAutonomy, Date.now());

      for (const msg of calls) {
        expect(msg.perpetual).toBe(true);
        expect(msg.maxExecutions).toBeUndefined();
        expect(msg.prepaidFee).toBe(2n ** 63n);
      }
    });

    it("should target the RAI retrieval system contract DID", async () => {
      const calls: any[] = [];
      const mockProcessScheduleMessage = vi.fn().mockImplementation((msg) => {
        calls.push(msg);
        return Promise.resolve({ id: `process-${calls.length}` });
      });

      const mockAutonomy = {
        getSPU: () => ({
          processScheduleMessage: mockProcessScheduleMessage,
        }),
      } as any;

      await registerGenesisContracts(mockAutonomy, Date.now());

      for (const msg of calls) {
        expect(msg.recipientAddress).toBe(SYSTEM_CONTRACTS.RAI_RETRIEVAL.DID);
      }
    });

    it("should use correct function selectors in data field", async () => {
      const calls: any[] = [];
      const mockProcessScheduleMessage = vi.fn().mockImplementation((msg) => {
        calls.push(msg);
        return Promise.resolve({ id: `process-${calls.length}` });
      });

      const mockAutonomy = {
        getSPU: () => ({
          processScheduleMessage: mockProcessScheduleMessage,
        }),
      } as any;

      await registerGenesisContracts(mockAutonomy, Date.now());

      // Retrieval: SWEEP_EXPIRED selector [0x00, 0x00, 0x00, 0x01]
      expect(calls[0].data).toEqual(
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.SELECTORS.SWEEP_EXPIRED,
      );

      // Distribution: DISTRIBUTE_RAI selector [0x00, 0x00, 0x00, 0x02]
      expect(calls[1].data).toEqual(
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.SELECTORS.DISTRIBUTE_RAI,
      );
    });
  });

  // ============ executeSweepExpired (via executeTransaction) ============

  describe.sequential("executeSweepExpired", () => {
    let state: StateManager;
    let params: ConsensusParams;

    beforeEach(() => {
      state = createTestState();
      params = createTestParams();
      setupSystemAccounts(state);
    });

    it("should succeed as a no-op when chain is young (no expired periods)", () => {
      // Timestamp is recent — currentPeriod - periodsUntilExpiration < 0
      const ts = Date.now();
      state.setBlockTimestamp(ts);
      const tx = createScheduledExecutionTx(
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.DID,
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.SELECTORS.SWEEP_EXPIRED,
        ts,
      );

      const result = executeTransaction(state, tx, 1n, params);
      expect(result.success).toBe(true);
      expect(result.gasUsed).toBe(50000n); // Base gas for no-op
    });

    it("should return base gas when no periods have expired", () => {
      // Use a timestamp early enough that no period can be expired
      const earlyTimestamp = 1000; // Very early, period 0
      state.setBlockTimestamp(earlyTimestamp);
      const tx = createScheduledExecutionTx(
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.DID,
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.SELECTORS.SWEEP_EXPIRED,
        earlyTimestamp,
      );

      const result = executeTransaction(state, tx, 1n, params);
      expect(result.success).toBe(true);
      expect(result.gasUsed).toBe(50000n);
    });

    it("should return base gas when expiration threshold reached but no expired accounts", () => {
      const periodMs = DEFAULT_EXPIRATION_CONFIG.periodDurationMs;
      const periods = DEFAULT_EXPIRATION_CONFIG.periodsUntilExpiration;
      const farTimestamp = periodMs * (periods + 10);
      state.setBlockTimestamp(farTimestamp);

      const tx = createScheduledExecutionTx(
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.DID,
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.SELECTORS.SWEEP_EXPIRED,
        farTimestamp,
      );

      const result = executeTransaction(state, tx, 100n, params);
      expect(result.success).toBe(true);
      expect(result.gasUsed).toBe(50000n); // No accounts have periodBalances
    });

    it("should sweep expired period balances to RAI holding", () => {
      const periodMs = DEFAULT_EXPIRATION_CONFIG.periodDurationMs;
      const periods = DEFAULT_EXPIRATION_CONFIG.periodsUntilExpiration;

      // Create agent with balance in an old period
      const agent = "did:lgm:expiring-agent";
      state.createAccount(agent, "agent-hash", 0n);

      // Activate period tracking at period 5 and credit tokens
      const oldPeriod = 5;
      state.setBlockTimestamp(oldPeriod * periodMs);
      state.creditBalance(agent, 500n * 10n ** 18n);

      // Verify period tracking is active
      const account = state.getAccount(agent)!;
      expect(account.periodBalances).toBeDefined();
      expect(account.periodBalances!.get(oldPeriod)).toBe(500n * 10n ** 18n);

      // Now advance time so period 5 is expired
      // currentPeriod must be >= oldPeriod + periodsUntilExpiration + 1
      const sweepPeriod = oldPeriod + periods + 1;
      const sweepTimestamp = sweepPeriod * periodMs;

      // Set block timestamp to sweep time (sweep uses getBlockTimestamp())
      state.setBlockTimestamp(sweepTimestamp);

      const tx = createScheduledExecutionTx(
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.DID,
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.SELECTORS.SWEEP_EXPIRED,
        sweepTimestamp,
      );

      const result = executeTransaction(state, tx, 200n, params);
      expect(result.success).toBe(true);

      // Agent's balance should be swept
      expect(state.getBalance(agent)).toBe(0n);

      // RAI holding should receive the swept tokens
      expect(state.getBalance(TREASURY.ADDRESSES.RAI_HOLDING)).toBe(
        500n * 10n ** 18n,
      );

      // Gas: base 50000 + 1 account * 5000
      expect(result.gasUsed).toBe(50000n + 1n * 5000n);
    });

    it("should skip dust amounts and RAI holding account during sweep", () => {
      const periodMs = DEFAULT_EXPIRATION_CONFIG.periodDurationMs;
      const periods = DEFAULT_EXPIRATION_CONFIG.periodsUntilExpiration;
      const dustThreshold = DEFAULT_EXPIRATION_CONFIG.dustThreshold;

      // Create agent with dust amount in old period
      const dustAgent = "did:lgm:dust-agent";
      state.createAccount(dustAgent, "dust-hash", 0n);

      const oldPeriod = 2;
      state.setBlockTimestamp(oldPeriod * periodMs);
      state.creditBalance(dustAgent, dustThreshold - 1n); // Below dust threshold

      // Create agent with real balance
      const realAgent = "did:lgm:real-agent";
      state.createAccount(realAgent, "real-hash", 0n);
      state.creditBalance(realAgent, 1000n * 10n ** 18n);

      // Advance past expiration
      const sweepPeriod = oldPeriod + periods + 1;
      state.setBlockTimestamp(sweepPeriod * periodMs);

      const tx = createScheduledExecutionTx(
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.DID,
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.SELECTORS.SWEEP_EXPIRED,
        sweepPeriod * periodMs,
      );

      const result = executeTransaction(state, tx, 300n, params);
      expect(result.success).toBe(true);

      // Dust agent keeps their balance (below threshold)
      expect(state.getBalance(dustAgent)).toBe(dustThreshold - 1n);

      // Real agent gets swept
      expect(state.getBalance(realAgent)).toBe(0n);
      expect(state.getBalance(TREASURY.ADDRESSES.RAI_HOLDING)).toBe(
        1000n * 10n ** 18n,
      );
    });

    it("should only sweep expired periods, leaving active ones intact", () => {
      const periodMs = DEFAULT_EXPIRATION_CONFIG.periodDurationMs;
      const periods = DEFAULT_EXPIRATION_CONFIG.periodsUntilExpiration;

      const agent = "did:lgm:mixed-periods";
      state.createAccount(agent, "mix-hash", 0n);

      // Credit in old period (will expire)
      const oldPeriod = 3;
      state.setBlockTimestamp(oldPeriod * periodMs);
      state.creditBalance(agent, 200n * 10n ** 18n);

      // Credit in recent period (won't expire)
      const recentPeriod = oldPeriod + periods; // Just at the boundary, not expired
      state.setBlockTimestamp(recentPeriod * periodMs);
      state.creditBalance(agent, 300n * 10n ** 18n);

      // Sweep at a time where oldPeriod is expired but recentPeriod is not
      // expirationThreshold = currentPeriod - periodsUntilExpiration
      // For oldPeriod to be expired: oldPeriod <= threshold
      // threshold = sweepPeriod - periods, so sweepPeriod >= oldPeriod + periods
      const sweepPeriod = oldPeriod + periods + 1;
      state.setBlockTimestamp(sweepPeriod * periodMs);

      const tx = createScheduledExecutionTx(
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.DID,
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.SELECTORS.SWEEP_EXPIRED,
        sweepPeriod * periodMs,
      );

      const result = executeTransaction(state, tx, 400n, params);
      expect(result.success).toBe(true);

      // Only 200 LGM from oldPeriod swept, 300 LGM from recentPeriod remains
      expect(state.getBalance(agent)).toBe(300n * 10n ** 18n);
      expect(state.getBalance(TREASURY.ADDRESSES.RAI_HOLDING)).toBe(
        200n * 10n ** 18n,
      );
    });
  });

  // ============ executeDistributeRAI (via executeTransaction) ============

  describe.sequential("executeDistributeRAI", () => {
    let state: StateManager;
    let params: ConsensusParams;

    beforeEach(() => {
      state = createTestState();
      params = createTestParams();
      setupSystemAccounts(state);
    });

    it("should no-op when RAI balance is zero", () => {
      const tx = createScheduledExecutionTx(
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.DID,
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.SELECTORS.DISTRIBUTE_RAI,
      );

      const raiBalanceBefore = state.getBalance(TREASURY.ADDRESSES.RAI_HOLDING);
      expect(raiBalanceBefore).toBe(0n);

      const result = executeTransaction(state, tx, 1n, params);
      expect(result.success).toBe(true);
      expect(result.gasUsed).toBe(50000n); // Base gas, no-op
    });

    it("should distribute RAI evenly to eligible accounts", () => {
      // Create 3 regular agent accounts
      const agent1 = "did:lgm:agent-001";
      const agent2 = "did:lgm:agent-002";
      const agent3 = "did:lgm:agent-003";
      state.createAccount(agent1, "agent1-hash", 100n * 10n ** 18n);
      state.createAccount(agent2, "agent2-hash", 200n * 10n ** 18n);
      state.createAccount(agent3, "agent3-hash", 300n * 10n ** 18n);

      // Fund RAI holding with 900 LGM
      const raiAmount = 900n * 10n ** 18n;
      state.createAccount("did:lgm:temp-funder", "funder-hash", raiAmount);
      state.transfer(
        "did:lgm:temp-funder",
        TREASURY.ADDRESSES.RAI_HOLDING,
        raiAmount,
      );

      const tx = createScheduledExecutionTx(
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.DID,
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.SELECTORS.DISTRIBUTE_RAI,
      );

      const result = executeTransaction(state, tx, MIN_RAI_AGE + 10n, params);
      expect(result.success).toBe(true);

      // Each agent gets 300 LGM (900 / 3 eligible agents)
      const perAgent = 300n * 10n ** 18n;
      expect(state.getBalance(agent1)).toBe(100n * 10n ** 18n + perAgent);
      expect(state.getBalance(agent2)).toBe(200n * 10n ** 18n + perAgent);
      expect(state.getBalance(agent3)).toBe(300n * 10n ** 18n + perAgent);

      // RAI holding should be empty (900 / 3 = 300 exact, no remainder)
      expect(state.getBalance(TREASURY.ADDRESSES.RAI_HOLDING)).toBe(0n);
    });

    it("should exclude system/treasury accounts from distribution", () => {
      // Only create one regular agent
      const agent1 = "did:lgm:agent-solo";
      state.createAccount(agent1, "solo-hash", 50n * 10n ** 18n);

      // Fund RAI holding
      const raiAmount = 1000n * 10n ** 18n;
      state.createAccount("did:lgm:temp-funder", "funder-hash", raiAmount);
      state.transfer(
        "did:lgm:temp-funder",
        TREASURY.ADDRESSES.RAI_HOLDING,
        raiAmount,
      );

      // Treasury accounts exist with balance (from setupSystemAccounts: RESEARCH has 100 LGM)
      // but they should be excluded from distribution

      const tx = createScheduledExecutionTx(
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.DID,
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.SELECTORS.DISTRIBUTE_RAI,
      );

      const result = executeTransaction(state, tx, MIN_RAI_AGE + 10n, params);
      expect(result.success).toBe(true);

      // Solo agent should get everything (only eligible account)
      expect(state.getBalance(agent1)).toBe(50n * 10n ** 18n + raiAmount);

      // Treasury accounts should be unchanged
      expect(state.getBalance(TREASURY.ADDRESSES.RESEARCH)).toBe(
        100n * 10n ** 18n,
      );
    });

    it("should handle remainder (stays in RAI holding)", () => {
      // 2 agents, 100 tokens → 50 each, 0 remainder. Use odd amount.
      const agent1 = "did:lgm:agent-a";
      const agent2 = "did:lgm:agent-b";
      state.createAccount(agent1, "a-hash", 10n * 10n ** 18n);
      state.createAccount(agent2, "b-hash", 10n * 10n ** 18n);

      // Fund RAI with 7 LGM (indivisible by 2 at wei level won't leave remainder,
      // but 7 wei will: 7n / 2n = 3n each, 1 remainder)
      state.createAccount("did:lgm:temp-funder", "funder-hash", 7n);
      state.transfer("did:lgm:temp-funder", TREASURY.ADDRESSES.RAI_HOLDING, 7n);

      const tx = createScheduledExecutionTx(
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.DID,
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.SELECTORS.DISTRIBUTE_RAI,
      );

      const result = executeTransaction(state, tx, MIN_RAI_AGE + 10n, params);
      expect(result.success).toBe(true);

      // Each gets 3 (7 / 2 = 3), remainder 1 stays in RAI holding
      expect(state.getBalance(agent1)).toBe(10n * 10n ** 18n + 3n);
      expect(state.getBalance(agent2)).toBe(10n * 10n ** 18n + 3n);
      expect(state.getBalance(TREASURY.ADDRESSES.RAI_HOLDING)).toBe(1n);
    });

    it("should no-op when no eligible accounts exist", () => {
      // Fund RAI holding, but all accounts are system accounts
      const raiAmount = 500n * 10n ** 18n;
      state.createAccount("did:lgm:temp-funder", "funder-hash", raiAmount);
      state.transfer(
        "did:lgm:temp-funder",
        TREASURY.ADDRESSES.RAI_HOLDING,
        raiAmount,
      );

      // temp-funder now has 0 balance, so it's not eligible.
      // All other accounts are system/treasury accounts.

      const tx = createScheduledExecutionTx(
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.DID,
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.SELECTORS.DISTRIBUTE_RAI,
      );

      const result = executeTransaction(state, tx, MIN_RAI_AGE + 10n, params);
      expect(result.success).toBe(true);
      expect(result.gasUsed).toBe(50000n); // No eligible accounts, no distribution

      // RAI balance unchanged
      expect(state.getBalance(TREASURY.ADDRESSES.RAI_HOLDING)).toBe(raiAmount);
    });

    it("should return gas proportional to eligible account count", () => {
      // Create 5 regular agents
      for (let i = 0; i < 5; i++) {
        state.createAccount(
          `did:lgm:agent-${i}`,
          `hash-${i}`,
          10n * 10n ** 18n,
        );
      }

      // Fund RAI holding
      const raiAmount = 500n * 10n ** 18n;
      state.createAccount("did:lgm:temp-funder", "funder-hash", raiAmount);
      state.transfer(
        "did:lgm:temp-funder",
        TREASURY.ADDRESSES.RAI_HOLDING,
        raiAmount,
      );

      const tx = createScheduledExecutionTx(
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.DID,
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.SELECTORS.DISTRIBUTE_RAI,
      );

      const result = executeTransaction(state, tx, MIN_RAI_AGE + 10n, params);
      expect(result.success).toBe(true);

      // Gas: 50000 base + 5 accounts * 5000 = 75000
      expect(result.gasUsed).toBe(50000n + 5n * 5000n);
    });

    it("should handle single eligible account", () => {
      const agent = "did:lgm:lone-wolf";
      state.createAccount(agent, "wolf-hash", 1n * 10n ** 18n);

      const raiAmount = 42n * 10n ** 18n;
      state.createAccount("did:lgm:temp-funder", "funder-hash", raiAmount);
      state.transfer(
        "did:lgm:temp-funder",
        TREASURY.ADDRESSES.RAI_HOLDING,
        raiAmount,
      );

      const tx = createScheduledExecutionTx(
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.DID,
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.SELECTORS.DISTRIBUTE_RAI,
      );

      const result = executeTransaction(state, tx, MIN_RAI_AGE + 10n, params);
      expect(result.success).toBe(true);

      // Single agent gets entire RAI balance
      expect(state.getBalance(agent)).toBe(1n * 10n ** 18n + raiAmount);
      expect(state.getBalance(TREASURY.ADDRESSES.RAI_HOLDING)).toBe(0n);
    });
  });

  // ============ System Contract Dispatch ============

  describe.sequential("System Contract Dispatch", () => {
    let state: StateManager;
    let params: ConsensusParams;

    beforeEach(() => {
      state = createTestState();
      params = createTestParams();
      setupSystemAccounts(state);
    });

    it("should fail when target contract account does not exist", () => {
      const tx = createScheduledExecutionTx(
        "did:lgm:nonexistent-contract",
        new Uint8Array([0x00, 0x00, 0x00, 0x01]),
      );

      const result = executeTransaction(state, tx, 1n, params);
      expect(result.success).toBe(false);
      expect(result.error).toContain("not found");
    });

    it("should fall through to default handler for unknown selectors", () => {
      // Target exists (RAI_RETRIEVAL DID), but unknown selector
      const unknownSelector = new Uint8Array([0xff, 0xff, 0xff, 0xff]);
      const tx = createScheduledExecutionTx(
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.DID,
        unknownSelector,
      );

      const result = executeTransaction(state, tx, 1n, params);
      expect(result.success).toBe(true);
      // Falls through to the default stub handler (50000n)
      expect(result.gasUsed).toBe(50000n);
    });

    it("should fall through for non-system contracts", () => {
      // Create a user contract account
      const userContract = "did:lgm:user-contract-123";
      state.createAccount(userContract, "user-contract-hash", 10n * 10n ** 18n);

      const tx = createScheduledExecutionTx(
        userContract,
        new Uint8Array([0x00, 0x00, 0x00, 0x01]),
      );

      const result = executeTransaction(state, tx, 1n, params);
      expect(result.success).toBe(true);
      // Default stub
      expect(result.gasUsed).toBe(50000n);
    });
  });

  // ============ Full Cycle ============

  describe.sequential("Full Cycle", () => {
    let state: StateManager;
    let params: ConsensusParams;

    beforeEach(() => {
      state = createTestState();
      params = createTestParams();
      setupSystemAccounts(state);
    });

    it("should sweep expired tokens → fund RAI → distribute → verify balances", () => {
      const periodMs = DEFAULT_EXPIRATION_CONFIG.periodDurationMs;
      const periods = DEFAULT_EXPIRATION_CONFIG.periodsUntilExpiration;

      // Step 1: Create 3 agent accounts with tokens in an old period
      const agents = [
        "did:lgm:agent-alpha",
        "did:lgm:agent-beta",
        "did:lgm:agent-gamma",
      ];
      const initialBalance = 50n * 10n ** 18n;
      const oldPeriod = 2;
      state.setBlockTimestamp(oldPeriod * periodMs);
      for (const agent of agents) {
        state.createAccount(agent, `${agent}-hash`, 0n);
        state.creditBalance(agent, initialBalance);
      }

      // Step 2: Give agents fresh tokens in a recent period (should survive sweep)
      const recentPeriod = oldPeriod + periods; // Not yet expired
      state.setBlockTimestamp(recentPeriod * periodMs);
      const freshBalance = 100n * 10n ** 18n;
      for (const agent of agents) {
        state.creditBalance(agent, freshBalance);
      }

      // Each agent now has: 50 LGM (old) + 100 LGM (recent) = 150 LGM
      for (const agent of agents) {
        expect(state.getBalance(agent)).toBe(initialBalance + freshBalance);
      }

      // Step 3: Execute sweep at time when oldPeriod is expired
      const sweepPeriod = oldPeriod + periods + 1;
      const sweepTimestamp = sweepPeriod * periodMs;
      state.setBlockTimestamp(sweepTimestamp);

      const sweepTx = createScheduledExecutionTx(
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.DID,
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.SELECTORS.SWEEP_EXPIRED,
        sweepTimestamp,
      );
      const sweepResult = executeTransaction(
        state,
        sweepTx,
        MIN_RAI_AGE + 100n,
        params,
      );
      expect(sweepResult.success).toBe(true);

      // Each agent lost 50 LGM (old period swept), retains 100 LGM (recent)
      for (const agent of agents) {
        expect(state.getBalance(agent)).toBe(freshBalance);
      }

      // RAI holding received 3 * 50 = 150 LGM
      const totalSwept = initialBalance * 3n;
      expect(state.getBalance(TREASURY.ADDRESSES.RAI_HOLDING)).toBe(totalSwept);

      // Step 4: Execute distribution
      const distTx = createScheduledExecutionTx(
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.DID,
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.SELECTORS.DISTRIBUTE_RAI,
      );
      const distResult = executeTransaction(
        state,
        distTx,
        MIN_RAI_AGE + 101n,
        params,
      );
      expect(distResult.success).toBe(true);

      // Each agent gets 50 LGM back (150 / 3 = 50 exact)
      const perAgent = totalSwept / 3n;
      for (const agent of agents) {
        expect(state.getBalance(agent)).toBe(freshBalance + perAgent);
      }

      // RAI holding should be empty
      expect(state.getBalance(TREASURY.ADDRESSES.RAI_HOLDING)).toBe(0n);

      // Step 5: Second distribution should no-op
      const dist2Tx = createScheduledExecutionTx(
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.DID,
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.SELECTORS.DISTRIBUTE_RAI,
      );
      const dist2Result = executeTransaction(
        state,
        dist2Tx,
        MIN_RAI_AGE + 102n,
        params,
      );
      expect(dist2Result.success).toBe(true);
      expect(dist2Result.gasUsed).toBe(50000n);

      // Balances unchanged
      for (const agent of agents) {
        expect(state.getBalance(agent)).toBe(freshBalance + perAgent);
      }
    });

    it("should handle multiple distribution rounds", () => {
      // Create agents
      const agent1 = "did:lgm:persistent-agent-1";
      const agent2 = "did:lgm:persistent-agent-2";
      state.createAccount(agent1, "p1-hash", 10n * 10n ** 18n);
      state.createAccount(agent2, "p2-hash", 10n * 10n ** 18n);

      // Round 1: Fund 200, distribute 100 each
      state.createAccount("did:lgm:funder-1", "f1-hash", 200n * 10n ** 18n);
      state.transfer(
        "did:lgm:funder-1",
        TREASURY.ADDRESSES.RAI_HOLDING,
        200n * 10n ** 18n,
      );

      let tx = createScheduledExecutionTx(
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.DID,
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.SELECTORS.DISTRIBUTE_RAI,
      );
      let result = executeTransaction(state, tx, MIN_RAI_AGE + 200n, params);
      expect(result.success).toBe(true);
      expect(state.getBalance(agent1)).toBe(110n * 10n ** 18n);
      expect(state.getBalance(agent2)).toBe(110n * 10n ** 18n);

      // Round 2: Fund 600, distribute 300 each
      state.createAccount("did:lgm:funder-2", "f2-hash", 600n * 10n ** 18n);
      state.transfer(
        "did:lgm:funder-2",
        TREASURY.ADDRESSES.RAI_HOLDING,
        600n * 10n ** 18n,
      );

      tx = createScheduledExecutionTx(
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.DID,
        SYSTEM_CONTRACTS.RAI_RETRIEVAL.SELECTORS.DISTRIBUTE_RAI,
      );
      result = executeTransaction(state, tx, MIN_RAI_AGE + 300n, params);
      expect(result.success).toBe(true);

      // Agent1: 10 + 100 + 300 = 410 LGM
      // Agent2: 10 + 100 + 300 = 410 LGM
      expect(state.getBalance(agent1)).toBe(410n * 10n ** 18n);
      expect(state.getBalance(agent2)).toBe(410n * 10n ** 18n);
      expect(state.getBalance(TREASURY.ADDRESSES.RAI_HOLDING)).toBe(0n);
    });
  });
});
