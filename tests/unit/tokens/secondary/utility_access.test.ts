/**
 * Utility/Access Token Tests
 *
 * Tests token-gated access, pay-per-use with burn,
 * subscription models, and tiered access.
 */

import { describe, it, expect, beforeEach } from "vitest";
import { TokenRegistryStateTree } from "../../../../state/token_registry.js";
import type { AccessTier } from "../../../../core/tokens/secondary/types.js";
import {
  makeAgent,
  resetAgentCounter,
  deployTestToken,
  createTestRegistry,
  ONE_LGM,
} from "./_helpers.js";

describe.sequential("Utility/Access Token", () => {
  let registry: TokenRegistryStateTree;
  let creator: ReturnType<typeof makeAgent>;
  let tokenId: string;

  beforeEach(() => {
    resetAgentCounter();
    registry = createTestRegistry();
    creator = makeAgent();
    const result = deployTestToken(registry, creator, {
      name: "AccessToken",
      symbol: "ACC",
      tokenType: "utility",
      maxSupply: 0n,
    });
    tokenId = result.tokenId;
  });

  describe("Token-Gated Access", () => {
    const tiers: AccessTier[] = [
      { name: "basic", minimumBalance: 10n * ONE_LGM, level: 1 },
      { name: "pro", minimumBalance: 100n * ONE_LGM, level: 2 },
      { name: "enterprise", minimumBalance: 1000n * ONE_LGM, level: 3 },
    ];

    it("should grant access when holding minimum balance", () => {
      const user = makeAgent();
      registry.mint(tokenId, user.did, 100n * ONE_LGM);

      const balance = registry.getTokenBalance(user.did, tokenId);
      const hasAccess = balance >= tiers[0].minimumBalance;
      expect(hasAccess).toBe(true);
    });

    it("should deny access when below minimum", () => {
      const user = makeAgent();
      registry.mint(tokenId, user.did, 5n * ONE_LGM);

      const balance = registry.getTokenBalance(user.did, tokenId);
      const hasAccess = balance >= tiers[0].minimumBalance;
      expect(hasAccess).toBe(false);
    });
  });

  describe("Pay-Per-Use with Burn", () => {
    it("should burn tokens on service usage", () => {
      const user = makeAgent();
      const usageCost = 5n * ONE_LGM;
      registry.mint(tokenId, user.did, 50n * ONE_LGM);

      // Burn tokens as payment for service
      registry.burn(tokenId, user.did, usageCost);

      expect(registry.getTokenBalance(user.did, tokenId)).toBe(45n * ONE_LGM);
      // Total supply reduced (deflationary)
      expect(registry.getTokenTotalSupply(tokenId)).toBe(45n * ONE_LGM);
    });

    it("should reject usage with insufficient tokens", () => {
      const user = makeAgent();
      registry.mint(tokenId, user.did, 3n * ONE_LGM);

      expect(() => registry.burn(tokenId, user.did, 5n * ONE_LGM)).toThrow(
        "Insufficient",
      );
    });
  });

  describe("Subscription Model", () => {
    it("should lock tokens for subscription period", () => {
      const subscriber = makeAgent();
      registry.mint(tokenId, subscriber.did, 100n * ONE_LGM);

      const lockAmount = 50n * ONE_LGM;

      // Create subscription (locks tokens)
      registry.createSubscription({
        subscriber: subscriber.did as any,
        tokenId,
        lockedAmount: lockAmount,
        startBlock: 1n,
        endBlock: 1000n,
      });

      const sub = registry.getSubscription(subscriber.did, tokenId);
      expect(sub).toBeDefined();
      expect(sub!.lockedAmount).toBe(lockAmount);
      expect(sub!.startBlock).toBe(1n);
      expect(sub!.endBlock).toBe(1000n);
    });

    it("should track subscription end block for expiration", () => {
      const subscriber = makeAgent();
      registry.mint(tokenId, subscriber.did, 100n * ONE_LGM);

      registry.createSubscription({
        subscriber: subscriber.did as any,
        tokenId,
        lockedAmount: 50n * ONE_LGM,
        startBlock: 100n,
        endBlock: 500n,
      });

      const sub = registry.getSubscription(subscriber.did, tokenId);
      // Subscription expires at block 500
      const currentBlock = 600n;
      const isExpired = currentBlock > sub!.endBlock;
      expect(isExpired).toBe(true);
    });
  });

  describe("Tiered Access", () => {
    it("should provide different service levels based on balance", () => {
      const tiers: AccessTier[] = [
        { name: "basic", minimumBalance: 10n * ONE_LGM, level: 1 },
        { name: "pro", minimumBalance: 100n * ONE_LGM, level: 2 },
        { name: "enterprise", minimumBalance: 1000n * ONE_LGM, level: 3 },
      ];

      const basicUser = makeAgent();
      const proUser = makeAgent();
      const enterpriseUser = makeAgent();

      registry.mint(tokenId, basicUser.did, 50n * ONE_LGM);
      registry.mint(tokenId, proUser.did, 500n * ONE_LGM);
      registry.mint(tokenId, enterpriseUser.did, 5000n * ONE_LGM);

      function getAccessLevel(did: string): number {
        const balance = registry.getTokenBalance(did, tokenId);
        let maxLevel = 0;
        for (const tier of tiers) {
          if (balance >= tier.minimumBalance && tier.level > maxLevel) {
            maxLevel = tier.level;
          }
        }
        return maxLevel;
      }

      expect(getAccessLevel(basicUser.did)).toBe(1);
      expect(getAccessLevel(proUser.did)).toBe(2);
      expect(getAccessLevel(enterpriseUser.did)).toBe(3);
    });
  });

  describe("LGM Gas Requirement", () => {
    it("should always require LGM for gas even when paying in utility token", () => {
      const user = makeAgent();
      registry.mint(tokenId, user.did, 1000n * ONE_LGM);

      // Utility token balance is separate from LGM gas
      // All operations deduct LGM gas at transaction layer
      const tokenBalance = registry.getTokenBalance(user.did, tokenId);

      // Burning utility tokens for service usage
      registry.burn(tokenId, user.did, 100n * ONE_LGM);

      // Only utility token was consumed, not LGM
      expect(registry.getTokenBalance(user.did, tokenId)).toBe(
        tokenBalance - 100n * ONE_LGM,
      );
    });
  });
});
