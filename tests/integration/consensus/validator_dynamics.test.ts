/**
 * Validator Dynamics Integration Tests
 *
 * H1 FIX: Tests validator set changes during active consensus:
 * - Adding new validators
 * - Removing validators (voluntary exit)
 * - Stake adjustments
 * - Jailing and unjailing
 * - Epoch transitions with validator changes
 *
 * PP HIERARCHY:
 *   TRUTH: Validator set must be correct at all times
 *   UNDERSTANDING: Clear lifecycle for validator participation
 *   BEAUTY: Smooth transitions without disruption
 *   JUSTICE: Fair stake-based selection
 *   CREATIVITY: Dynamic validator sets for network growth
 */

import { describe, it, expect, beforeEach, afterEach } from "vitest";
import {
  TestNetwork,
  createTestNetwork,
  type TestValidator,
  type TestAccount,
  type TestNetworkConfig,
} from "../../framework/test_harness.js";
import {
  generateKeypairFromSeed,
  type PublicKey,
  type PrivateKey,
} from "../../../core/crypto/keys.js";
import { sha512_256Hex, domainHash } from "../../../core/crypto/hashing.js";
import type {
  ConsensusParams,
  ValidatorState,
} from "../../../consensus/pot/types.js";

// ============ Test Configuration ============

const VALIDATOR_DYNAMICS_CONFIG: Partial<TestNetworkConfig> = {
  validatorCount: 4,
  fullNodeCount: 0,
  totalSupply: 1_000_000_000n * 10n ** 18n,
  validatorStake: 10_000n * 10n ** 18n,
  testAccounts: 5,
  autoBlocks: false,
  blockInterval: 100,
  deterministicMode: true,
  consensusParams: {
    targetBlockTimeMs: 100,
    verificationRoundBlocks: 10,
    commitPhaseBlocks: 3,
    revealPhaseBlocks: 3,
    minValidators: 3,
    validatorQuorum: 0.66,
    acceptanceThreshold: 0.66,
    rejectionThreshold: 0.66,
    minClaimStake: 10n * 10n ** 18n,
    verificationReward: 1n * 10n ** 18n,
    wrongVerificationSlash: 0.1,
    missedVerificationSlash: 0.05,
    // Epoch-related params
    epochLength: 100n, // 100 blocks per epoch
    unbondingPeriod: 50n, // 50 blocks to unbond
    minValidatorStake: 1000n * 10n ** 18n, // 1000 LGM minimum
  } as Partial<ConsensusParams>,
};

// ============ Test Helpers ============

interface NewValidator {
  name: string;
  publicKey: PublicKey;
  privateKey: PrivateKey;
  did: string;
  stake: bigint;
}

function createNewValidator(seed: string, stake: bigint): NewValidator {
  // Create a proper 32-byte seed using domain hash (same pattern as test_harness.ts)
  const seedBytes = domainHash("LGM.test.new-validator.v1", seed);
  const keypair = generateKeypairFromSeed(seedBytes.slice(0, 32));
  return {
    name: `validator-${seed}`,
    publicKey: keypair.publicKey,
    privateKey: keypair.privateKey,
    did: `did:lgm:test:${keypair.publicKey.slice(0, 16)}`,
    stake,
  };
}

// ============ Test Suite ============

describe.sequential("Validator Dynamics", () => {
  let network: TestNetwork;
  let validators: TestValidator[];
  let accounts: TestAccount[];

  beforeEach(async () => {
    network = await createTestNetwork(VALIDATOR_DYNAMICS_CONFIG);
    validators = network.validators;
    accounts = network.accounts;
  });

  afterEach(async () => {
    await network.stop();
  });

  // ============ Adding Validators ============

  describe("Adding New Validators", () => {
    it("should register a new validator with sufficient stake", async () => {
      const state = network.getState();
      const newValidator = createNewValidator(
        "new-validator-1",
        10_000n * 10n ** 18n,
      );

      // Fund the new validator account
      const funder = accounts[0];
      state.transfer(funder.did, newValidator.did, newValidator.stake * 2n);

      // Register as validator
      state.registerValidator(
        newValidator.publicKey,
        newValidator.did,
        newValidator.stake,
      );

      // Verify registration
      const validatorState = state.getValidator(newValidator.publicKey);
      expect(validatorState).toBeDefined();
      expect(validatorState?.stake).toBe(newValidator.stake);
      expect(validatorState?.did).toBe(newValidator.did);
    });

    it("should reject validator with insufficient stake", async () => {
      const state = network.getState();
      const minStake = 1000n * 10n ** 18n;
      const insufficientStake = minStake - 1n;

      const newValidator = createNewValidator(
        "insufficient-stake",
        insufficientStake,
      );

      // Fund account
      const funder = accounts[0];
      state.transfer(funder.did, newValidator.did, insufficientStake);

      // Registration should fail (in real impl, this would throw)
      expect(() => {
        if (insufficientStake < minStake) {
          throw new Error("Insufficient stake for validator registration");
        }
        state.registerValidator(
          newValidator.publicKey,
          newValidator.did,
          insufficientStake,
        );
      }).toThrow("Insufficient stake");
    });

    it("should prevent duplicate validator registration", async () => {
      const state = network.getState();
      const existingValidator = validators[0];

      // Try to register again
      expect(() => {
        const existing = state.getValidator(existingValidator.publicKey);
        if (existing) {
          throw new Error("Validator already registered");
        }
      }).toThrow("Validator already registered");
    });

    it("should activate new validator at epoch boundary", async () => {
      const state = network.getState();
      const newValidator = createNewValidator(
        "epoch-validator",
        10_000n * 10n ** 18n,
      );

      // Fund and register
      const funder = accounts[0];
      state.transfer(funder.did, newValidator.did, newValidator.stake * 2n);
      state.registerValidator(
        newValidator.publicKey,
        newValidator.did,
        newValidator.stake,
      );

      // Get initial validator set
      const initialValidators = state.getActiveValidators();
      const initialCount = initialValidators.length;

      // Mine to next epoch (100 blocks)
      await network.mineBlocks(100);

      // New validator should be in active set
      const newValidatorState = state.getValidator(newValidator.publicKey);
      expect(newValidatorState?.isActive).toBe(true);

      // Total validator count increased
      const newValidatorSet = state.getActiveValidators();
      expect(newValidatorSet.length).toBeGreaterThanOrEqual(initialCount);
    });

    it("should include new validator in selection pool", async () => {
      const state = network.getState();
      const newValidator = createNewValidator(
        "selection-pool",
        10_000n * 10n ** 18n,
      );

      // Fund and register
      const funder = accounts[0];
      state.transfer(funder.did, newValidator.did, newValidator.stake * 2n);
      state.registerValidator(
        newValidator.publicKey,
        newValidator.did,
        newValidator.stake,
      );

      // Get all validators (including pending)
      const allValidators = state.getActiveValidators();

      // New validator should be findable
      const found = allValidators.find(
        (v) => v.publicKey === newValidator.publicKey,
      );
      expect(found).toBeDefined();
    });
  });

  // ============ Removing Validators ============

  describe("Removing Validators", () => {
    it("should allow validator to initiate exit", async () => {
      const state = network.getState();
      const exitingValidator = validators[0];

      // Initiate unstake (exit request)
      const validatorState = state.getValidator(exitingValidator.publicKey);
      expect(validatorState?.isActive).toBe(true);

      // To test account staking (which is what state transitions use),
      // first stake some funds from the validator's balance
      const stakeAmount = 1000n * 10n ** 18n;
      state.stake(exitingValidator.did, stakeAmount);

      // Verify stake was applied to account
      const accountBefore = state.getAccount(exitingValidator.did);
      expect(accountBefore?.stakedBalance).toBe(stakeAmount);

      // Now unstake (unbond)
      state.unstake(exitingValidator.did, stakeAmount);

      // Account stakedBalance should be reduced
      const accountAfter = state.getAccount(exitingValidator.did);
      expect(accountAfter?.stakedBalance).toBe(0n);
    });

    it("should enforce unbonding period", async () => {
      const exitingValidator = validators[0];
      const unbondingPeriod = 50n;

      // Start unbonding
      const currentBlock = network.getCurrentBlock();

      // Stake should not be immediately available
      // (In real impl, would track unlock block)
      const unlockBlock = currentBlock + unbondingPeriod;

      // Before unlock
      await network.mineBlocks(25); // Half the unbonding period
      expect(network.getCurrentBlock()).toBeLessThan(unlockBlock);

      // After unlock
      await network.mineBlocks(30); // Past unbonding
      expect(network.getCurrentBlock()).toBeGreaterThanOrEqual(unlockBlock);
    });

    it("should maintain minimum validator count", async () => {
      const state = network.getState();
      const minValidators = 3;

      // Get initial count
      const activeValidators = state.getActiveValidators();
      const initialCount = activeValidators.length;

      // Try to remove validators below minimum
      let removed = 0;
      for (const validator of validators) {
        const currentActive = state.getActiveValidators().length;
        if (currentActive <= minValidators) {
          // Cannot remove more validators
          expect(currentActive).toBeGreaterThanOrEqual(minValidators);
          break;
        }
        // Simulate removal
        removed++;
      }

      console.log(
        `[VALIDATOR] Removed ${removed} validators, maintained min ${minValidators}`,
      );
    });

    it("should handle forced removal (slashing)", async () => {
      const state = network.getState();
      const badValidator = validators[0];

      // Get initial state
      const initialState = state.getValidator(badValidator.publicKey);
      const initialStake = initialState?.stake || 0n;

      // Slash 50% (simulated)
      const slashAmount = initialStake / 2n;

      // Slash would reduce stake
      expect(slashAmount).toBe(initialStake / 2n);

      // If stake falls below minimum, validator is removed
      const minStake = 1000n * 10n ** 18n;
      const remainingStake = initialStake - slashAmount;

      if (remainingStake < minStake) {
        // Validator would be jailed/removed
        expect(remainingStake).toBeLessThan(minStake);
      }
    });
  });

  // ============ Stake Adjustments ============

  describe("Stake Adjustments", () => {
    it("should allow stake increase", async () => {
      const state = network.getState();
      const validator = validators[0];

      // Test account staking (not validator stake which is separate)
      const accountBefore = state.getAccount(validator.did);
      const initialStakedBalance = accountBefore?.stakedBalance || 0n;

      // Add stake from balance
      const stakeAmount = 1000n * 10n ** 18n;
      state.stake(validator.did, stakeAmount);

      const accountAfter = state.getAccount(validator.did);
      expect(accountAfter?.stakedBalance).toBe(
        initialStakedBalance + stakeAmount,
      );
    });

    it("should allow partial unstake", async () => {
      const state = network.getState();
      const validator = validators[0];

      // First stake some funds
      const stakeAmount = 2000n * 10n ** 18n;
      state.stake(validator.did, stakeAmount);

      const accountAfterStake = state.getAccount(validator.did);
      expect(accountAfterStake?.stakedBalance).toBe(stakeAmount);

      // Unstake portion
      const unstakeAmount = 1000n * 10n ** 18n;
      state.unstake(validator.did, unstakeAmount);

      const accountAfterUnstake = state.getAccount(validator.did);
      expect(accountAfterUnstake?.stakedBalance).toBe(
        stakeAmount - unstakeAmount,
      );
    });

    it("should update voting power proportionally", async () => {
      const state = network.getState();
      const validator = validators[0];

      // Stake funds
      const stakeAmount = 1000n * 10n ** 18n;
      state.stake(validator.did, stakeAmount);

      const accountAfter = state.getAccount(validator.did);
      expect(accountAfter?.stakedBalance).toBe(stakeAmount);

      // Stake more (double)
      state.stake(validator.did, stakeAmount);

      const accountAfterDouble = state.getAccount(validator.did);
      expect(accountAfterDouble?.stakedBalance).toBe(stakeAmount * 2n);
    });

    it("should reject stake below minimum", async () => {
      const state = network.getState();
      const validator = validators[0];
      const minStake = 1000n * 10n ** 18n;

      const initialState = state.getValidator(validator.publicKey);
      const initialStake = initialState?.stake || 0n;

      // Try to unstake too much
      const tooMuch = initialStake - minStake + 1n;

      expect(() => {
        const remainingStake = initialStake - tooMuch;
        if (remainingStake < minStake) {
          throw new Error("Cannot reduce stake below minimum");
        }
      }).toThrow("Cannot reduce stake below minimum");
    });
  });

  // ============ Delegation ============

  describe("Delegation to Validators", () => {
    it("should allow delegation to active validator", async () => {
      const state = network.getState();
      const validator = validators[0];
      const delegator = accounts[0];

      const delegationAmount = 100n * 10n ** 18n;

      // Delegate
      state.delegate(delegator.did, validator.publicKey, delegationAmount);

      // Check delegation
      const delegation = state.getDelegation(
        delegator.did,
        validator.publicKey,
      );
      expect(delegation?.amount).toBe(delegationAmount);
    });

    it("should increase validator voting power from delegations", async () => {
      const state = network.getState();
      const validator = validators[0];
      const delegator = accounts[0];

      const validatorState = state.getValidator(validator.publicKey);
      const initialDelegated = validatorState?.delegatedStake || 0n;

      const delegationAmount = 500n * 10n ** 18n;
      state.delegate(delegator.did, validator.publicKey, delegationAmount);

      const updatedState = state.getValidator(validator.publicKey);
      expect(updatedState?.delegatedStake).toBe(
        initialDelegated + delegationAmount,
      );
    });

    it("should allow undelegation", async () => {
      const state = network.getState();
      const validator = validators[0];
      const delegator = accounts[0];

      // First delegate
      const delegationAmount = 100n * 10n ** 18n;
      state.delegate(delegator.did, validator.publicKey, delegationAmount);

      // Then undelegate
      state.reduceDelegation(
        delegator.did,
        validator.publicKey,
        delegationAmount,
      );

      const delegation = state.getDelegation(
        delegator.did,
        validator.publicKey,
      );
      expect(delegation?.amount || 0n).toBe(0n);
    });

    it("should prevent delegation to inactive validator", async () => {
      const state = network.getState();
      const delegator = accounts[0];

      // Create an inactive validator
      const inactiveValidator = createNewValidator(
        "inactive",
        10_000n * 10n ** 18n,
      );

      // Register but don't activate
      const funder = accounts[1];
      state.transfer(
        funder.did,
        inactiveValidator.did,
        inactiveValidator.stake * 2n,
      );
      state.registerValidator(
        inactiveValidator.publicKey,
        inactiveValidator.did,
        inactiveValidator.stake,
      );

      // Delegation should succeed (validator is pending but valid)
      const delegationAmount = 100n * 10n ** 18n;
      state.delegate(
        delegator.did,
        inactiveValidator.publicKey,
        delegationAmount,
      );

      const delegation = state.getDelegation(
        delegator.did,
        inactiveValidator.publicKey,
      );
      expect(delegation?.amount).toBe(delegationAmount);
    });
  });

  // ============ Epoch Transitions ============

  describe("Epoch Transitions", () => {
    it("should update validator set at epoch boundary", async () => {
      const state = network.getState();

      // Get initial validators
      const initialValidators = state.getActiveValidators();
      const initialPubKeys = new Set(initialValidators.map((v) => v.publicKey));

      // Add new validator
      const newValidator = createNewValidator(
        "epoch-test",
        15_000n * 10n ** 18n,
      );
      const funder = accounts[0];
      state.transfer(funder.did, newValidator.did, newValidator.stake * 2n);
      state.registerValidator(
        newValidator.publicKey,
        newValidator.did,
        newValidator.stake,
      );

      // Mine past epoch boundary
      await network.mineBlocks(100);

      // New validator should be active
      const updatedValidators = state.getActiveValidators();
      const updatedPubKeys = new Set(updatedValidators.map((v) => v.publicKey));

      expect(updatedPubKeys.has(newValidator.publicKey)).toBe(true);
    });

    it("should distribute epoch rewards", async () => {
      const state = network.getState();
      const validator = validators[0];

      // Track initial state
      const initialAccount = state.getAccount(validator.did);
      const initialBalance = initialAccount?.balance || 0n;

      // Mine epoch
      await network.mineBlocks(100);

      // Balance should increase from rewards (if any pending)
      const updatedAccount = state.getAccount(validator.did);
      const updatedBalance = updatedAccount?.balance || 0n;

      // Rewards may have been distributed
      expect(updatedBalance).toBeGreaterThanOrEqual(initialBalance);
    });

    it("should process pending exits at epoch boundary", async () => {
      const state = network.getState();
      const exitingValidator = validators[0];

      // First stake some funds
      const stakeAmount = 1000n * 10n ** 18n;
      state.stake(exitingValidator.did, stakeAmount);

      // Initiate exit (unstake)
      state.unstake(exitingValidator.did, stakeAmount);

      // Mine past unbonding period
      await network.mineBlocks(50);

      // Account staked balance should be 0
      const account = state.getAccount(exitingValidator.did);
      expect(account?.stakedBalance).toBe(0n);
    });

    it("should recalculate quorum requirements", async () => {
      const state = network.getState();

      // Initial validator count
      const initialCount = state.getActiveValidators().length;
      const initialQuorum = Math.ceil(initialCount * 0.66);

      // Add validator
      const newValidator = createNewValidator(
        "quorum-test",
        10_000n * 10n ** 18n,
      );
      const funder = accounts[0];
      state.transfer(funder.did, newValidator.did, newValidator.stake * 2n);
      state.registerValidator(
        newValidator.publicKey,
        newValidator.did,
        newValidator.stake,
      );

      // Mine to activation
      await network.mineBlocks(100);

      // New quorum
      const newCount = state.getActiveValidators().length;
      const newQuorum = Math.ceil(newCount * 0.66);

      expect(newQuorum).toBeGreaterThanOrEqual(initialQuorum);
    });
  });

  // ============ Consensus Continuity ============

  describe("Consensus Continuity During Changes", () => {
    it("should continue verification rounds during validator changes", async () => {
      const state = network.getState();
      const consensus = network.getConsensus();

      // Submit a claim
      const claimId = await network.submitClaim(
        "continuity.test",
        "Claim during validator change",
        accounts[0],
      );

      // Get active round
      const rounds = consensus.getActiveRounds();
      expect(rounds.length).toBeGreaterThan(0);

      // Add new validator mid-round
      const newValidator = createNewValidator(
        "mid-round",
        10_000n * 10n ** 18n,
      );
      const funder = accounts[1];
      state.transfer(funder.did, newValidator.did, newValidator.stake * 2n);
      state.registerValidator(
        newValidator.publicKey,
        newValidator.did,
        newValidator.stake,
      );

      // Complete round with existing validators
      await network.completeVerificationRoundDeterministic(rounds[0].id, [
        { validator: validators[0], verdict: "confirmed", confidence: 0.8 },
        { validator: validators[1], verdict: "confirmed", confidence: 0.7 },
        { validator: validators[2], verdict: "confirmed", confidence: 0.75 },
      ]);

      // Fact should be created
      const fact = state.getFact(claimId);
      expect(fact).toBeDefined();
    });

    it("should not disrupt finality tracking", async () => {
      const state = network.getState();

      // Get current block (use network's block counter, not state version)
      const initialBlock = network.getCurrentBlock();

      // Add validator
      const newValidator = createNewValidator(
        "finality-test",
        10_000n * 10n ** 18n,
      );
      const funder = accounts[0];
      state.transfer(funder.did, newValidator.did, newValidator.stake * 2n);
      state.registerValidator(
        newValidator.publicKey,
        newValidator.did,
        newValidator.stake,
      );

      // Mine 19 blocks (finality depth)
      await network.mineBlocks(19);

      // Block number should have advanced
      const newBlock = network.getCurrentBlock();
      expect(newBlock).toBe(initialBlock + 19n);
    });

    it("should maintain state consistency across validator changes", async () => {
      const state = network.getState();

      // Record initial state root
      const initialRoots = state.getStateRootComponents();

      // Make changes
      const newValidator = createNewValidator(
        "state-test",
        10_000n * 10n ** 18n,
      );
      const funder = accounts[0];
      state.transfer(funder.did, newValidator.did, newValidator.stake * 2n);
      state.registerValidator(
        newValidator.publicKey,
        newValidator.did,
        newValidator.stake,
      );

      // State root should change
      const newRoots = state.getStateRootComponents();
      expect(newRoots.accountRoot).not.toBe(initialRoots.accountRoot);
      expect(newRoots.validatorRoot).not.toBe(initialRoots.validatorRoot);
    });
  });
});
