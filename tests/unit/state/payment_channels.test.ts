/**
 * Payment Channel Tests
 *
 * Tests the complete payment channel lifecycle from opening to closing.
 * Validates GAP-005 resolution: open_payment_channel, update_payment_channel,
 * close_payment_channel, settle_payment_channel, and dispute_payment_channel.
 */

import { describe, it, expect, beforeEach } from "vitest";
import { StateManager } from "../../../state/state_manager.js";
import { executeTransaction } from "../../../state/transitions.js";
import {
  PAYMENT_CHANNEL_CONSTANTS,
  generateChannelId,
  computeStateHash,
  sumDistributions,
} from "../../../state/payment_channels.js";
import type {
  Transaction,
  ConsensusParams,
} from "../../../consensus/pot/types.js";
import type {
  OpenPaymentChannelPayload,
  UpdatePaymentChannelPayload,
  ClosePaymentChannelPayload,
  SettlePaymentChannelPayload,
  DisputePaymentChannelPayload,
  SerializedPaymentDistribution,
  SignedChannelState,
} from "../../../state/types.js";
import { publicKeyToDid } from "../../../core/agents/identity/did.js";

// Test constants
const ONE_LGM = 10n ** 18n;

const TEST_CONSENSUS_PARAMS: ConsensusParams = {
  targetBlockTimeMs: 2521,
  verificationRoundBlocks: 11,
  commitPhaseBlocks: 4,
  revealPhaseBlocks: 4,
  epochLength: 1111n,
  minValidators: 3,
  maxValidators: 22,
  validatorQuorum: 0.66,
  acceptanceThreshold: 0.77,
  rejectionThreshold: 0.77,
  contestedThreshold: 0.55,
  minValidatorStake: 111n * 10n ** 15n, // 0.111 LGM
  minClaimStake: 11n * 10n ** 15n,
  blockReward: 11n * ONE_LGM,
  verificationReward: 1n * ONE_LGM,
  citationRewardPerFact: 1n * ONE_LGM,
  wrongVerificationSlash: 0.06,
  missedVerificationSlash: 0.0001,
  invalidClaimSlash: 0.22,
  researchShareRate: 0.22,
  maxTransactionsPerBlock: 111,
  unbondingPeriod: 222222n,
};

const PAYER_KEY = "payer-public-key";
const PROVIDER_KEY = "provider-public-key";
const PAYER_DID = publicKeyToDid(PAYER_KEY);
const PROVIDER_DID = publicKeyToDid(PROVIDER_KEY);
const KNOWLEDGE_CONTRIBUTOR_DID = "did:lgm:knowledge-contributor";

// Use describe.sequential to prevent parallel test execution issues
describe.sequential("Payment Channels (GAP-005)", () => {
  let state: StateManager;
  let blockNumber: bigint;
  let nonceTracker: Map<string, bigint>;

  function createTransaction(
    type: string,
    payload: Record<string, unknown>,
    sender = PAYER_KEY,
  ): Transaction {
    const did = publicKeyToDid(sender);
    const nonce = nonceTracker.get(did) ?? 0n;
    nonceTracker.set(did, nonce + 1n);

    return {
      type: type as any,
      sender,
      nonce,
      fee: 0n,
      payload,
      signature: "" as any,
    };
  }

  function resetNonces(): void {
    nonceTracker = new Map<string, bigint>();
  }

  function createDistribution(
    providerAmount: bigint,
    knowledgeAmount: bigint = 0n,
    treasuryAmount: bigint = 0n,
    burnedAmount: bigint = 0n,
  ): SerializedPaymentDistribution {
    return {
      apiProvider: { did: PROVIDER_DID, amount: providerAmount.toString() },
      knowledgeRewards:
        knowledgeAmount > 0n
          ? [
              {
                did: KNOWLEDGE_CONTRIBUTOR_DID,
                factId: "fact:test-001",
                amount: knowledgeAmount.toString(),
              },
            ]
          : [],
      protocol: {
        burned: burnedAmount.toString(),
        treasury: treasuryAmount.toString(),
      },
    };
  }

  function createSignedState(
    channelId: string,
    nonce: bigint,
    spent: bigint,
    available: bigint,
    distributions: SerializedPaymentDistribution[] = [],
  ): SignedChannelState {
    const stateHash = computeStateHash(channelId, nonce, spent, distributions);
    return {
      channelId,
      nonce,
      spent,
      available,
      stateHash,
      distributions,
      payerSignature: "mock-payer-sig",
      providerSignature: "mock-provider-sig",
    };
  }

  beforeEach(() => {
    resetNonces();
    state = new StateManager();
    blockNumber = 100n;

    // Create test accounts with sufficient balance
    state.createAccount(PAYER_DID, "payer-op-hash", 10000n * ONE_LGM);
    state.createAccount(PROVIDER_DID, "provider-op-hash", 1000n * ONE_LGM);
    state.createAccount(
      KNOWLEDGE_CONTRIBUTOR_DID,
      "contrib-op-hash",
      100n * ONE_LGM,
    );

    // Create escrow account for channel funds
    state.createAccount(
      PAYMENT_CHANNEL_CONSTANTS.ESCROW_DID,
      "escrow-hash",
      0n,
    );

    // Create treasury account
    state.createAccount(
      PAYMENT_CHANNEL_CONSTANTS.TREASURY_DID,
      "treasury-hash",
      0n,
    );
  });

  // ============================================================================
  // open_payment_channel Tests
  // ============================================================================

  describe("executeOpenPaymentChannel", () => {
    it("should create a payment channel with valid deposit", () => {
      const deposit = 500n * ONE_LGM;
      const payload: OpenPaymentChannelPayload = {
        type: "open_payment_channel",
        provider: PROVIDER_DID,
        deposit,
        durationBlocks: 10000n,
      };

      const tx = createTransaction("open_payment_channel", payload);
      const result = executeTransaction(
        state,
        tx,
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(true);

      // Verify channel was created
      const channels = state.getPaymentChannelsByPayer(PAYER_DID);
      expect(channels.length).toBe(1);

      const channel = channels[0];
      expect(channel.payer).toBe(PAYER_DID);
      expect(channel.provider).toBe(PROVIDER_DID);
      expect(channel.deposited).toBe(deposit);
      expect(channel.available).toBe(deposit);
      expect(channel.spent).toBe(0n);
      expect(channel.status).toBe("open");
      expect(channel.expiresAt).toBe(blockNumber + 10000n);
    });

    it("should transfer deposit to escrow", () => {
      const deposit = 500n * ONE_LGM;
      const initialBalance = state.getAccount(PAYER_DID)!.balance;

      const payload: OpenPaymentChannelPayload = {
        type: "open_payment_channel",
        provider: PROVIDER_DID,
        deposit,
        durationBlocks: 10000n,
      };

      const tx = createTransaction("open_payment_channel", payload);
      executeTransaction(state, tx, blockNumber, TEST_CONSENSUS_PARAMS);

      // Payer balance reduced by deposit
      expect(state.getAccount(PAYER_DID)!.balance).toBe(
        initialBalance - deposit,
      );

      // Escrow balance increased
      expect(
        state.getAccount(PAYMENT_CHANNEL_CONSTANTS.ESCROW_DID)!.balance,
      ).toBe(deposit);
    });

    it("should reject deposit below minimum", () => {
      const payload: OpenPaymentChannelPayload = {
        type: "open_payment_channel",
        provider: PROVIDER_DID,
        deposit: 50n * 10n ** 15n, // Below 100 mLGM minimum
        durationBlocks: 10000n,
      };

      const tx = createTransaction("open_payment_channel", payload);
      const result = executeTransaction(
        state,
        tx,
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("Minimum deposit");
    });

    it("should reject duration above maximum", () => {
      const payload: OpenPaymentChannelPayload = {
        type: "open_payment_channel",
        provider: PROVIDER_DID,
        deposit: 500n * ONE_LGM,
        durationBlocks: PAYMENT_CHANNEL_CONSTANTS.MAX_DURATION_BLOCKS + 1n,
      };

      const tx = createTransaction("open_payment_channel", payload);
      const result = executeTransaction(
        state,
        tx,
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("Maximum duration");
    });

    it("should reject insufficient balance", () => {
      const payload: OpenPaymentChannelPayload = {
        type: "open_payment_channel",
        provider: PROVIDER_DID,
        deposit: 50000n * ONE_LGM, // More than payer has
        durationBlocks: 10000n,
      };

      const tx = createTransaction("open_payment_channel", payload);
      const result = executeTransaction(
        state,
        tx,
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("Insufficient balance");
    });

    it("should reject non-existent provider", () => {
      const payload: OpenPaymentChannelPayload = {
        type: "open_payment_channel",
        provider: "did:lgm:non-existent",
        deposit: 500n * ONE_LGM,
        durationBlocks: 10000n,
      };

      const tx = createTransaction("open_payment_channel", payload);
      const result = executeTransaction(
        state,
        tx,
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("Provider not found");
    });

    it("should reject duplicate channel between same parties", () => {
      const payload: OpenPaymentChannelPayload = {
        type: "open_payment_channel",
        provider: PROVIDER_DID,
        deposit: 500n * ONE_LGM,
        durationBlocks: 10000n,
      };

      // First channel opens successfully
      const tx1 = createTransaction("open_payment_channel", payload);
      const result1 = executeTransaction(
        state,
        tx1,
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );
      expect(result1.success).toBe(true);

      // Second channel should fail
      const tx2 = createTransaction("open_payment_channel", payload);
      const result2 = executeTransaction(
        state,
        tx2,
        blockNumber + 1n,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result2.success).toBe(false);
      expect(result2.error).toContain("already exists");
    });

    it("should use custom settlement frequency when provided", () => {
      const payload: OpenPaymentChannelPayload = {
        type: "open_payment_channel",
        provider: PROVIDER_DID,
        deposit: 500n * ONE_LGM,
        durationBlocks: 10000n,
        settlementFrequency: 50n,
      };

      const tx = createTransaction("open_payment_channel", payload);
      executeTransaction(state, tx, blockNumber, TEST_CONSENSUS_PARAMS);

      const channel = state.getPaymentChannelsByPayer(PAYER_DID)[0];
      expect(channel.settlementFrequency).toBe(50n);
    });
  });

  // ============================================================================
  // update_payment_channel Tests
  // ============================================================================

  describe("executeUpdatePaymentChannel", () => {
    let channelId: string;

    beforeEach(() => {
      // Open a channel first
      const payload: OpenPaymentChannelPayload = {
        type: "open_payment_channel",
        provider: PROVIDER_DID,
        deposit: 1000n * ONE_LGM,
        durationBlocks: 10000n,
      };
      const tx = createTransaction("open_payment_channel", payload);
      executeTransaction(state, tx, blockNumber, TEST_CONSENSUS_PARAMS);

      const channel = state.getPaymentChannelsByPayer(PAYER_DID)[0];
      channelId = channel.channelId;
    });

    it("should update channel state with valid signed state", () => {
      const spent = 100n * ONE_LGM;
      const available = 900n * ONE_LGM;
      const signedState = createSignedState(channelId, 1n, spent, available);

      const payload: UpdatePaymentChannelPayload = {
        type: "update_payment_channel",
        channelId,
        signedState,
      };

      const tx = createTransaction("update_payment_channel", payload);
      const result = executeTransaction(
        state,
        tx,
        blockNumber + 10n,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(true);

      const channel = state.getPaymentChannel(channelId)!;
      expect(channel.spent).toBe(spent);
      expect(channel.available).toBe(available);
      expect(channel.lastNonce).toBe(1n);
    });

    it("should reject update from non-participant", () => {
      const signedState = createSignedState(
        channelId,
        1n,
        100n * ONE_LGM,
        900n * ONE_LGM,
      );

      const payload: UpdatePaymentChannelPayload = {
        type: "update_payment_channel",
        channelId,
        signedState,
      };

      // Create account for unauthorized sender
      state.createAccount(
        "did:lgm:unauthorized",
        "unauth-hash",
        100n * ONE_LGM,
      );

      const tx = createTransaction(
        "update_payment_channel",
        payload,
        "unauthorized",
      );
      const result = executeTransaction(
        state,
        tx,
        blockNumber + 10n,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("Not channel participant");
    });

    it("should reject update with non-incrementing nonce", () => {
      // First update
      const signedState1 = createSignedState(
        channelId,
        1n,
        100n * ONE_LGM,
        900n * ONE_LGM,
      );
      const tx1 = createTransaction("update_payment_channel", {
        type: "update_payment_channel",
        channelId,
        signedState: signedState1,
      });
      executeTransaction(state, tx1, blockNumber + 10n, TEST_CONSENSUS_PARAMS);

      // Second update with same nonce should fail
      const signedState2 = createSignedState(
        channelId,
        1n,
        200n * ONE_LGM,
        800n * ONE_LGM,
      );
      const tx2 = createTransaction("update_payment_channel", {
        type: "update_payment_channel",
        channelId,
        signedState: signedState2,
      });
      const result = executeTransaction(
        state,
        tx2,
        blockNumber + 20n,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("Nonce");
    });

    it("should reject update that violates balance invariant", () => {
      const signedState = createSignedState(
        channelId,
        1n,
        100n * ONE_LGM,
        100n * ONE_LGM,
      ); // Wrong: 200 != 1000

      const payload: UpdatePaymentChannelPayload = {
        type: "update_payment_channel",
        channelId,
        signedState,
      };

      const tx = createTransaction("update_payment_channel", payload);
      const result = executeTransaction(
        state,
        tx,
        blockNumber + 10n,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("Balance invariant");
    });

    it("should allow provider to submit update", () => {
      const signedState = createSignedState(
        channelId,
        1n,
        100n * ONE_LGM,
        900n * ONE_LGM,
      );

      const payload: UpdatePaymentChannelPayload = {
        type: "update_payment_channel",
        channelId,
        signedState,
      };

      const tx = createTransaction(
        "update_payment_channel",
        payload,
        PROVIDER_KEY,
      );
      const result = executeTransaction(
        state,
        tx,
        blockNumber + 10n,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(true);
    });
  });

  // ============================================================================
  // close_payment_channel Tests
  // ============================================================================

  describe("executeClosePaymentChannel", () => {
    let channelId: string;
    const deposit = 1000n * ONE_LGM;

    beforeEach(() => {
      const payload: OpenPaymentChannelPayload = {
        type: "open_payment_channel",
        provider: PROVIDER_DID,
        deposit,
        durationBlocks: 10000n,
      };
      const tx = createTransaction("open_payment_channel", payload);
      executeTransaction(state, tx, blockNumber, TEST_CONSENSUS_PARAMS);

      const channel = state.getPaymentChannelsByPayer(PAYER_DID)[0];
      channelId = channel.channelId;
    });

    describe("Cooperative Close", () => {
      it("should close immediately with final state", () => {
        const spent = 200n * ONE_LGM;
        const available = 800n * ONE_LGM;
        const finalState = createSignedState(channelId, 5n, spent, available);

        const payload: ClosePaymentChannelPayload = {
          type: "close_payment_channel",
          channelId,
          finalState,
        };

        const initialPayerBalance = state.getAccount(PAYER_DID)!.balance;

        const tx = createTransaction("close_payment_channel", payload);
        const result = executeTransaction(
          state,
          tx,
          blockNumber + 100n,
          TEST_CONSENSUS_PARAMS,
        );

        expect(result.success).toBe(true);

        const channel = state.getPaymentChannel(channelId)!;
        expect(channel.status).toBe("closed");
        expect(channel.available).toBe(0n);

        // Remaining funds returned to payer
        expect(state.getAccount(PAYER_DID)!.balance).toBe(
          initialPayerBalance + available,
        );
      });

      it("should process distributions on cooperative close", () => {
        const providerAmount = 150n * ONE_LGM;
        const treasuryAmount = 50n * ONE_LGM;
        const spent = providerAmount + treasuryAmount;
        const available = deposit - spent;

        const distribution = createDistribution(
          providerAmount,
          0n,
          treasuryAmount,
        );
        const finalState = createSignedState(channelId, 5n, spent, available, [
          distribution,
        ]);

        const payload: ClosePaymentChannelPayload = {
          type: "close_payment_channel",
          channelId,
          finalState,
        };

        const initialProviderBalance = state.getAccount(PROVIDER_DID)!.balance;

        const tx = createTransaction("close_payment_channel", payload);
        executeTransaction(
          state,
          tx,
          blockNumber + 100n,
          TEST_CONSENSUS_PARAMS,
        );

        // Provider receives their share
        expect(state.getAccount(PROVIDER_DID)!.balance).toBe(
          initialProviderBalance + providerAmount,
        );

        // Treasury receives protocol share
        expect(
          state.getAccount(PAYMENT_CHANNEL_CONSTANTS.TREASURY_DID)!.balance,
        ).toBe(treasuryAmount);
      });
    });

    describe("Unilateral Close", () => {
      it("should start dispute period on unilateral close", () => {
        const payload: ClosePaymentChannelPayload = {
          type: "close_payment_channel",
          channelId,
          unilateral: true,
        };

        const tx = createTransaction("close_payment_channel", payload);
        const result = executeTransaction(
          state,
          tx,
          blockNumber + 100n,
          TEST_CONSENSUS_PARAMS,
        );

        expect(result.success).toBe(true);

        const channel = state.getPaymentChannel(channelId)!;
        expect(channel.status).toBe("closing");
        expect(channel.disputeStartBlock).toBe(blockNumber + 100n);
        expect(channel.disputeDeadline).toBe(
          blockNumber + 100n + PAYMENT_CHANNEL_CONSTANTS.DISPUTE_PERIOD_BLOCKS,
        );
      });

      it("should allow unilateral close from provider", () => {
        const payload: ClosePaymentChannelPayload = {
          type: "close_payment_channel",
          channelId,
          unilateral: true,
        };

        const tx = createTransaction(
          "close_payment_channel",
          payload,
          PROVIDER_KEY,
        );
        const result = executeTransaction(
          state,
          tx,
          blockNumber + 100n,
          TEST_CONSENSUS_PARAMS,
        );

        expect(result.success).toBe(true);
        expect(state.getPaymentChannel(channelId)!.status).toBe("closing");
      });
    });

    describe("Expiry Close", () => {
      it("should allow close after expiration", () => {
        const payload: ClosePaymentChannelPayload = {
          type: "close_payment_channel",
          channelId,
          // No finalState, no unilateral = expiry close
        };

        // Fast forward past expiration
        const expiredBlock = blockNumber + 10001n;

        const tx = createTransaction("close_payment_channel", payload);
        const result = executeTransaction(
          state,
          tx,
          expiredBlock,
          TEST_CONSENSUS_PARAMS,
        );

        expect(result.success).toBe(true);

        const channel = state.getPaymentChannel(channelId)!;
        expect(channel.status).toBe("closed");
      });

      it("should reject expiry close before expiration", () => {
        const payload: ClosePaymentChannelPayload = {
          type: "close_payment_channel",
          channelId,
        };

        const tx = createTransaction("close_payment_channel", payload);
        const result = executeTransaction(
          state,
          tx,
          blockNumber + 100n,
          TEST_CONSENSUS_PARAMS,
        );

        expect(result.success).toBe(false);
        expect(result.error).toContain("not expired");
      });

      it("should return remaining funds on expiry close", () => {
        const initialPayerBalance = state.getAccount(PAYER_DID)!.balance;

        const payload: ClosePaymentChannelPayload = {
          type: "close_payment_channel",
          channelId,
        };

        const tx = createTransaction("close_payment_channel", payload);
        executeTransaction(
          state,
          tx,
          blockNumber + 10001n,
          TEST_CONSENSUS_PARAMS,
        );

        // All funds returned (nothing spent)
        expect(state.getAccount(PAYER_DID)!.balance).toBe(
          initialPayerBalance + deposit,
        );
      });
    });
  });

  // ============================================================================
  // settle_payment_channel Tests
  // ============================================================================

  describe("executeSettlePaymentChannel", () => {
    let channelId: string;
    const deposit = 1000n * ONE_LGM;

    beforeEach(() => {
      const payload: OpenPaymentChannelPayload = {
        type: "open_payment_channel",
        provider: PROVIDER_DID,
        deposit,
        durationBlocks: 10000n,
        settlementFrequency: 100n,
      };
      const tx = createTransaction("open_payment_channel", payload);
      executeTransaction(state, tx, blockNumber, TEST_CONSENSUS_PARAMS);

      channelId = state.getPaymentChannelsByPayer(PAYER_DID)[0].channelId;
    });

    it("should settle distributions and update channel state", () => {
      const providerAmount = 100n * ONE_LGM;
      const spent = providerAmount;
      const available = deposit - spent;

      const distribution = createDistribution(providerAmount);
      const signedState = createSignedState(channelId, 1n, spent, available, [
        distribution,
      ]);

      const payload: SettlePaymentChannelPayload = {
        type: "settle_payment_channel",
        channelId,
        signedState,
      };

      const initialProviderBalance = state.getAccount(PROVIDER_DID)!.balance;

      // Wait for settlement frequency
      const tx = createTransaction(
        "settle_payment_channel",
        payload,
        PROVIDER_KEY,
      );
      const result = executeTransaction(
        state,
        tx,
        blockNumber + 100n,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(true);

      // Provider received payment
      expect(state.getAccount(PROVIDER_DID)!.balance).toBe(
        initialProviderBalance + providerAmount,
      );

      // Channel state updated
      const channel = state.getPaymentChannel(channelId)!;
      expect(channel.totalDistributed).toBe(providerAmount);
      expect(channel.lastSettlementBlock).toBe(blockNumber + 100n);
      expect(channel.pendingDistributions.length).toBe(0);
    });

    it("should reject settlement from non-provider", () => {
      const signedState = createSignedState(
        channelId,
        1n,
        100n * ONE_LGM,
        900n * ONE_LGM,
      );

      const payload: SettlePaymentChannelPayload = {
        type: "settle_payment_channel",
        channelId,
        signedState,
      };

      // Payer tries to settle
      const tx = createTransaction(
        "settle_payment_channel",
        payload,
        PAYER_KEY,
      );
      const result = executeTransaction(
        state,
        tx,
        blockNumber + 100n,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("Only provider can settle");
    });

    it("should reject settlement before frequency period", () => {
      const signedState = createSignedState(
        channelId,
        1n,
        100n * ONE_LGM,
        900n * ONE_LGM,
      );

      const payload: SettlePaymentChannelPayload = {
        type: "settle_payment_channel",
        channelId,
        signedState,
      };

      // Try to settle too soon (frequency is 100 blocks)
      const tx = createTransaction(
        "settle_payment_channel",
        payload,
        PROVIDER_KEY,
      );
      const result = executeTransaction(
        state,
        tx,
        blockNumber + 50n,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("Settlement too soon");
    });

    it("should distribute to knowledge contributors", () => {
      const providerAmount = 70n * ONE_LGM;
      const knowledgeAmount = 20n * ONE_LGM;
      const treasuryAmount = 10n * ONE_LGM;
      const spent = providerAmount + knowledgeAmount + treasuryAmount;
      const available = deposit - spent;

      const distribution = createDistribution(
        providerAmount,
        knowledgeAmount,
        treasuryAmount,
      );
      const signedState = createSignedState(channelId, 1n, spent, available, [
        distribution,
      ]);

      const payload: SettlePaymentChannelPayload = {
        type: "settle_payment_channel",
        channelId,
        signedState,
      };

      const initialContribBalance = state.getAccount(
        KNOWLEDGE_CONTRIBUTOR_DID,
      )!.balance;

      const tx = createTransaction(
        "settle_payment_channel",
        payload,
        PROVIDER_KEY,
      );
      executeTransaction(state, tx, blockNumber + 100n, TEST_CONSENSUS_PARAMS);

      // Knowledge contributor received reward
      expect(state.getAccount(KNOWLEDGE_CONTRIBUTOR_DID)!.balance).toBe(
        initialContribBalance + knowledgeAmount,
      );
    });
  });

  // ============================================================================
  // dispute_payment_channel Tests
  // ============================================================================

  describe("executeDisputePaymentChannel", () => {
    let channelId: string;
    const deposit = 1000n * ONE_LGM;

    beforeEach(() => {
      // Open channel
      const openPayload: OpenPaymentChannelPayload = {
        type: "open_payment_channel",
        provider: PROVIDER_DID,
        deposit,
        durationBlocks: 10000n,
      };
      const openTx = createTransaction("open_payment_channel", openPayload);
      const openResult = executeTransaction(
        state,
        openTx,
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );
      if (!openResult.success) {
        throw new Error(`Failed to open channel: ${openResult.error}`);
      }

      const channels = state.getPaymentChannelsByPayer(PAYER_DID);
      if (channels.length === 0) {
        throw new Error("No channels found after opening");
      }
      channelId = channels[0].channelId;

      // Submit some updates
      const updateState = createSignedState(
        channelId,
        1n,
        200n * ONE_LGM,
        800n * ONE_LGM,
      );
      const updateTx = createTransaction("update_payment_channel", {
        type: "update_payment_channel",
        channelId,
        signedState: updateState,
      });
      const updateResult = executeTransaction(
        state,
        updateTx,
        blockNumber + 10n,
        TEST_CONSENSUS_PARAMS,
      );
      if (!updateResult.success) {
        throw new Error(`Failed to update channel: ${updateResult.error}`);
      }

      // Start unilateral close
      const closeTx = createTransaction("close_payment_channel", {
        type: "close_payment_channel",
        channelId,
        unilateral: true,
      });
      const closeResult = executeTransaction(
        state,
        closeTx,
        blockNumber + 20n,
        TEST_CONSENSUS_PARAMS,
      );
      if (!closeResult.success) {
        throw new Error(`Failed to close channel: ${closeResult.error}`);
      }
    });

    it("should accept dispute with higher nonce state", () => {
      const channel = state.getPaymentChannel(channelId)!;
      expect(channel.status).toBe("closing");

      // Provider has a newer state
      const challengeState = createSignedState(
        channelId,
        5n,
        400n * ONE_LGM,
        600n * ONE_LGM,
      );

      const payload: DisputePaymentChannelPayload = {
        type: "dispute_payment_channel",
        channelId,
        challengeState,
      };

      const tx = createTransaction(
        "dispute_payment_channel",
        payload,
        PROVIDER_KEY,
      );
      const result = executeTransaction(
        state,
        tx,
        blockNumber + 30n,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(true);

      const updatedChannel = state.getPaymentChannel(channelId)!;
      expect(updatedChannel.status).toBe("disputed");
      expect(updatedChannel.lastNonce).toBe(5n);
      expect(updatedChannel.spent).toBe(400n * ONE_LGM);
    });

    it("should reject dispute with lower or equal nonce", () => {
      const challengeState = createSignedState(
        channelId,
        1n,
        300n * ONE_LGM,
        700n * ONE_LGM,
      );

      const payload: DisputePaymentChannelPayload = {
        type: "dispute_payment_channel",
        channelId,
        challengeState,
      };

      const tx = createTransaction(
        "dispute_payment_channel",
        payload,
        PROVIDER_KEY,
      );
      const result = executeTransaction(
        state,
        tx,
        blockNumber + 30n,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("Challenge nonce");
    });

    it("should reject dispute from non-participant", () => {
      const challengeState = createSignedState(
        channelId,
        5n,
        400n * ONE_LGM,
        600n * ONE_LGM,
      );

      state.createAccount("did:lgm:attacker", "attacker-hash", 100n * ONE_LGM);

      const payload: DisputePaymentChannelPayload = {
        type: "dispute_payment_channel",
        channelId,
        challengeState,
      };

      const tx = createTransaction(
        "dispute_payment_channel",
        payload,
        "attacker",
      );
      const result = executeTransaction(
        state,
        tx,
        blockNumber + 30n,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("Not channel participant");
    });

    it("should reject dispute after deadline", () => {
      const channel = state.getPaymentChannel(channelId)!;
      const challengeState = createSignedState(
        channelId,
        5n,
        400n * ONE_LGM,
        600n * ONE_LGM,
      );

      const payload: DisputePaymentChannelPayload = {
        type: "dispute_payment_channel",
        channelId,
        challengeState,
      };

      // Past the dispute deadline
      const tx = createTransaction(
        "dispute_payment_channel",
        payload,
        PROVIDER_KEY,
      );
      const result = executeTransaction(
        state,
        tx,
        channel.disputeDeadline! + 1n,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("Dispute period ended");
    });

    it("should reject dispute on non-closing channel", () => {
      // Reset - open new channel without starting close
      state = new StateManager();
      resetNonces();
      blockNumber = 100n;

      state.createAccount(PAYER_DID, "payer-op-hash", 10000n * ONE_LGM);
      state.createAccount(PROVIDER_DID, "provider-op-hash", 1000n * ONE_LGM);
      state.createAccount(
        PAYMENT_CHANNEL_CONSTANTS.ESCROW_DID,
        "escrow-hash",
        0n,
      );

      const openPayload: OpenPaymentChannelPayload = {
        type: "open_payment_channel",
        provider: PROVIDER_DID,
        deposit,
        durationBlocks: 10000n,
      };
      const openTx = createTransaction("open_payment_channel", openPayload);
      executeTransaction(state, openTx, blockNumber, TEST_CONSENSUS_PARAMS);

      const newChannelId =
        state.getPaymentChannelsByPayer(PAYER_DID)[0].channelId;
      const challengeState = createSignedState(
        newChannelId,
        5n,
        400n * ONE_LGM,
        600n * ONE_LGM,
      );

      const payload: DisputePaymentChannelPayload = {
        type: "dispute_payment_channel",
        channelId: newChannelId,
        challengeState,
      };

      const tx = createTransaction(
        "dispute_payment_channel",
        payload,
        PROVIDER_KEY,
      );
      const result = executeTransaction(
        state,
        tx,
        blockNumber + 30n,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("not in dispute period");
    });
  });

  // ============================================================================
  // PaymentChannelStateTree Tests
  // ============================================================================

  describe("PaymentChannelStateTree", () => {
    it("should generate deterministic channel IDs", () => {
      const id1 = generateChannelId(PAYER_DID, PROVIDER_DID, 100n);
      const id2 = generateChannelId(PAYER_DID, PROVIDER_DID, 100n);
      const id3 = generateChannelId(PAYER_DID, PROVIDER_DID, 101n);

      expect(id1).toBe(id2);
      expect(id1).not.toBe(id3);
      expect(id1.startsWith("pc:")).toBe(true);
    });

    it("should compute consistent state hashes", () => {
      const distributions: SerializedPaymentDistribution[] = [
        createDistribution(100n * ONE_LGM, 10n * ONE_LGM, 5n * ONE_LGM),
      ];

      const hash1 = computeStateHash("ch1", 1n, 115n * ONE_LGM, distributions);
      const hash2 = computeStateHash("ch1", 1n, 115n * ONE_LGM, distributions);
      const hash3 = computeStateHash("ch1", 2n, 115n * ONE_LGM, distributions);

      expect(hash1).toBe(hash2);
      expect(hash1).not.toBe(hash3);
    });

    it("should sum distributions correctly", () => {
      const dist1 = createDistribution(
        100n * ONE_LGM,
        20n * ONE_LGM,
        5n * ONE_LGM,
      );
      const dist2 = createDistribution(50n * ONE_LGM, 0n, 10n * ONE_LGM);

      const total = sumDistributions([dist1, dist2]);

      // dist1: 100 + 20 + 5 = 125
      // dist2: 50 + 10 = 60
      // total: 185
      expect(total).toBe(185n * ONE_LGM);
    });

    it("should query channels by status", () => {
      // Open two channels
      const payload1: OpenPaymentChannelPayload = {
        type: "open_payment_channel",
        provider: PROVIDER_DID,
        deposit: 500n * ONE_LGM,
        durationBlocks: 10000n,
      };

      // Create another provider
      state.createAccount("did:lgm:provider-2", "p2-hash", 100n * ONE_LGM);

      const payload2: OpenPaymentChannelPayload = {
        type: "open_payment_channel",
        provider: "did:lgm:provider-2",
        deposit: 500n * ONE_LGM,
        durationBlocks: 10000n,
      };

      const tx1 = createTransaction("open_payment_channel", payload1);
      const tx2 = createTransaction("open_payment_channel", payload2);

      executeTransaction(state, tx1, blockNumber, TEST_CONSENSUS_PARAMS);
      executeTransaction(state, tx2, blockNumber + 1n, TEST_CONSENSUS_PARAMS);

      const openChannels = state.getOpenPaymentChannels();
      expect(openChannels.length).toBe(2);

      // Close one unilaterally
      const channelToClose = openChannels[0].channelId;
      const closeTx = createTransaction("close_payment_channel", {
        type: "close_payment_channel",
        channelId: channelToClose,
        unilateral: true,
      });
      executeTransaction(
        state,
        closeTx,
        blockNumber + 10n,
        TEST_CONSENSUS_PARAMS,
      );

      // Now should have 1 open, 1 closing
      expect(state.getOpenPaymentChannels().length).toBe(1);
      expect(state.getPaymentChannelsByStatus("closing").length).toBe(1);
    });
  });

  // ============================================================================
  // Edge Cases and Integration
  // ============================================================================

  describe("Edge Cases", () => {
    it("should handle zero-value distributions", () => {
      // Open channel
      const payload: OpenPaymentChannelPayload = {
        type: "open_payment_channel",
        provider: PROVIDER_DID,
        deposit: 500n * ONE_LGM,
        durationBlocks: 10000n,
      };
      const tx = createTransaction("open_payment_channel", payload);
      executeTransaction(state, tx, blockNumber, TEST_CONSENSUS_PARAMS);

      const channelId = state.getPaymentChannelsByPayer(PAYER_DID)[0].channelId;

      // Distribution with zero provider amount (only treasury)
      const dist = createDistribution(0n, 0n, 10n * ONE_LGM);
      const finalState = createSignedState(
        channelId,
        1n,
        10n * ONE_LGM,
        490n * ONE_LGM,
        [dist],
      );

      const closeTx = createTransaction("close_payment_channel", {
        type: "close_payment_channel",
        channelId,
        finalState,
      });

      const result = executeTransaction(
        state,
        closeTx,
        blockNumber + 10n,
        TEST_CONSENSUS_PARAMS,
      );
      expect(result.success).toBe(true);
    });

    it("should handle maximum deposit amount", () => {
      // Give payer more balance
      state.createAccount(
        "did:lgm:rich-payer",
        "rich-hash",
        100000000n * ONE_LGM,
      );

      const payload: OpenPaymentChannelPayload = {
        type: "open_payment_channel",
        provider: PROVIDER_DID,
        deposit: 1000000n * ONE_LGM, // 1 million LGM
        durationBlocks: 10000n,
      };

      const tx = createTransaction(
        "open_payment_channel",
        payload,
        "rich-payer",
      );
      const result = executeTransaction(
        state,
        tx,
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(true);
    });

    it("should track channel statistics correctly", () => {
      // Open a channel
      const payload: OpenPaymentChannelPayload = {
        type: "open_payment_channel",
        provider: PROVIDER_DID,
        deposit: 500n * ONE_LGM,
        durationBlocks: 10000n,
      };
      const tx = createTransaction("open_payment_channel", payload);
      executeTransaction(state, tx, blockNumber, TEST_CONSENSUS_PARAMS);

      const stats = state.getPaymentChannelStats();
      expect(stats.totalChannels).toBe(1);
      expect(stats.openChannels).toBe(1);
      expect(stats.totalDeposited).toBe(500n * ONE_LGM);
      expect(stats.totalSpent).toBe(0n);
      expect(stats.totalAvailable).toBe(500n * ONE_LGM);
    });
  });
});
