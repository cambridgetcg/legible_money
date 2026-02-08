/**
 * Staking/Yield Token Tests
 *
 * Tests staking secondary tokens to earn LGM rewards,
 * reward claiming, unstaking with unbonding, and LGM reward funding.
 */

import { describe, it, expect, beforeEach } from "vitest";
import { TokenRegistryStateTree } from "../../../../state/token_registry.js";
import {
  makeAgent,
  resetAgentCounter,
  deployTestToken,
  createTestRegistry,
  ONE_LGM,
} from "./_helpers.js";

describe.sequential("Staking/Yield Token", () => {
  let registry: TokenRegistryStateTree;
  let creator: ReturnType<typeof makeAgent>;
  let tokenId: string;

  beforeEach(() => {
    resetAgentCounter();
    registry = createTestRegistry();
    creator = makeAgent();
    const result = deployTestToken(registry, creator, {
      name: "StakeToken",
      symbol: "STK",
      tokenType: "fungible",
    });
    tokenId = result.tokenId;
  });

  describe("Staking Secondary Token", () => {
    it("should lock secondary tokens in staking contract", () => {
      const staker = makeAgent();
      registry.mint(tokenId, staker.did, 1000n * ONE_LGM);

      registry.stakeSecondaryToken(staker.did, tokenId, 500n * ONE_LGM, 1n);

      // Balance reduced
      expect(registry.getTokenBalance(staker.did, tokenId)).toBe(
        500n * ONE_LGM,
      );
      // Stake position exists
      const position = registry.getStakePosition(staker.did, tokenId);
      expect(position).toBeDefined();
      expect(position!.stakedAmount).toBe(500n * ONE_LGM);
    });

    it("should reject staking more than balance", () => {
      const staker = makeAgent();
      registry.mint(tokenId, staker.did, 100n * ONE_LGM);

      expect(() =>
        registry.stakeSecondaryToken(staker.did, tokenId, 200n * ONE_LGM, 1n),
      ).toThrow("Insufficient");
    });

    it("should accumulate LGM rewards proportional to stake", () => {
      const staker = makeAgent();
      registry.mint(tokenId, staker.did, 1000n * ONE_LGM);
      registry.stakeSecondaryToken(staker.did, tokenId, 1000n * ONE_LGM, 1n);

      // Simulate reward accumulation over blocks
      registry.accumulateRewards(staker.did, tokenId, 10n * ONE_LGM, 100n);

      const position = registry.getStakePosition(staker.did, tokenId);
      expect(position!.accumulatedRewards).toBe(10n * ONE_LGM);
      expect(position!.lastRewardBlock).toBe(100n);
    });
  });

  describe("Reward Claiming", () => {
    it("should claim accumulated LGM rewards", () => {
      const staker = makeAgent();
      registry.mint(tokenId, staker.did, 1000n * ONE_LGM);
      registry.stakeSecondaryToken(staker.did, tokenId, 500n * ONE_LGM, 1n);
      registry.accumulateRewards(staker.did, tokenId, 25n * ONE_LGM, 50n);

      const claimed = registry.claimRewards(staker.did, tokenId);
      expect(claimed).toBe(25n * ONE_LGM);
    });

    it("should reset reward accumulator after claim", () => {
      const staker = makeAgent();
      registry.mint(tokenId, staker.did, 1000n * ONE_LGM);
      registry.stakeSecondaryToken(staker.did, tokenId, 500n * ONE_LGM, 1n);
      registry.accumulateRewards(staker.did, tokenId, 25n * ONE_LGM, 50n);

      registry.claimRewards(staker.did, tokenId);

      const position = registry.getStakePosition(staker.did, tokenId);
      expect(position!.accumulatedRewards).toBe(0n);
    });

    it("should not allow claiming with zero stake", () => {
      const staker = makeAgent();

      expect(() => registry.claimRewards(staker.did, tokenId)).toThrow(
        "No active staking position",
      );
    });
  });

  describe("Unstaking", () => {
    it("should return secondary tokens after unstaking", () => {
      const staker = makeAgent();
      registry.mint(tokenId, staker.did, 1000n * ONE_LGM);
      registry.stakeSecondaryToken(staker.did, tokenId, 800n * ONE_LGM, 1n);

      const returned = registry.unstakeSecondaryToken(staker.did, tokenId);

      expect(returned).toBe(800n * ONE_LGM);
      expect(registry.getTokenBalance(staker.did, tokenId)).toBe(
        1000n * ONE_LGM,
      );
    });

    it("should stop LGM reward accrual after unstake", () => {
      const staker = makeAgent();
      registry.mint(tokenId, staker.did, 1000n * ONE_LGM);
      registry.stakeSecondaryToken(staker.did, tokenId, 500n * ONE_LGM, 1n);
      registry.accumulateRewards(staker.did, tokenId, 10n * ONE_LGM, 50n);

      registry.unstakeSecondaryToken(staker.did, tokenId);

      // No position exists anymore
      const position = registry.getStakePosition(staker.did, tokenId);
      expect(position).toBeUndefined();
    });

    it("should reject unstaking with no position", () => {
      const staker = makeAgent();

      expect(() =>
        registry.unstakeSecondaryToken(staker.did, tokenId),
      ).toThrow();
    });
  });

  describe("LGM Reward Funding", () => {
    it("should track reward accumulation correctly across multiple blocks", () => {
      const staker = makeAgent();
      registry.mint(tokenId, staker.did, 1000n * ONE_LGM);
      registry.stakeSecondaryToken(staker.did, tokenId, 1000n * ONE_LGM, 1n);

      // Accumulate rewards over several blocks
      registry.accumulateRewards(staker.did, tokenId, 5n * ONE_LGM, 10n);
      registry.accumulateRewards(staker.did, tokenId, 5n * ONE_LGM, 20n);
      registry.accumulateRewards(staker.did, tokenId, 5n * ONE_LGM, 30n);

      const position = registry.getStakePosition(staker.did, tokenId);
      expect(position!.accumulatedRewards).toBe(15n * ONE_LGM);
      expect(position!.lastRewardBlock).toBe(30n);
    });

    it("should allow adding to existing stake", () => {
      const staker = makeAgent();
      registry.mint(tokenId, staker.did, 1000n * ONE_LGM);
      registry.stakeSecondaryToken(staker.did, tokenId, 300n * ONE_LGM, 1n);
      registry.stakeSecondaryToken(staker.did, tokenId, 200n * ONE_LGM, 10n);

      const position = registry.getStakePosition(staker.did, tokenId);
      expect(position!.stakedAmount).toBe(500n * ONE_LGM);
      expect(registry.getTokenBalance(staker.did, tokenId)).toBe(
        500n * ONE_LGM,
      );
    });
  });
});
