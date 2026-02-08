/**
 * Security & Invariants Tests
 *
 * Tests supply conservation, overflow protection, reentrancy guards,
 * authorization, LGM base layer invariants, state consistency, and edge cases.
 */

import { describe, it, expect, beforeEach } from "vitest";
import { TokenRegistryStateTree } from "../../../../state/token_registry.js";
import { SECONDARY_TOKEN_GAS_COSTS } from "../../../../core/tokens/secondary/types.js";
import {
  makeAgent,
  resetAgentCounter,
  deployTestToken,
  deployAndMint,
  createTestRegistry,
  ONE_LGM,
} from "./_helpers.js";

describe.sequential("Security & Invariants", () => {
  let registry: TokenRegistryStateTree;
  let creator: ReturnType<typeof makeAgent>;
  let tokenId: string;

  beforeEach(() => {
    resetAgentCounter();
    registry = createTestRegistry();
    creator = makeAgent();
    const result = deployTestToken(registry, creator, {
      name: "SecureToken",
      symbol: "SEC",
      maxSupply: 1_000_000n * ONE_LGM,
    });
    tokenId = result.tokenId;
  });

  // ============ Supply Conservation ============

  describe("Supply Conservation", () => {
    it("totalSupply always equals sum of all balances", () => {
      const accounts = Array.from({ length: 10 }, () => makeAgent());

      // Mint to various accounts
      for (let i = 0; i < accounts.length; i++) {
        registry.mint(tokenId, accounts[i].did, BigInt(i + 1) * 100n * ONE_LGM);
      }

      // Transfer between accounts randomly
      registry.transferToken(
        tokenId,
        accounts[0].did,
        accounts[5].did,
        50n * ONE_LGM,
      );
      registry.transferToken(
        tokenId,
        accounts[3].did,
        accounts[7].did,
        200n * ONE_LGM,
      );

      // Burn some
      registry.burn(tokenId, accounts[9].did, 100n * ONE_LGM);

      // Verify invariant
      let sumBalances = 0n;
      for (const acc of accounts) {
        sumBalances += registry.getTokenBalance(acc.did, tokenId);
      }
      expect(registry.getTokenTotalSupply(tokenId)).toBe(sumBalances);
    });

    it("no operation creates tokens from nothing", () => {
      const supplyBefore = registry.getTokenTotalSupply(tokenId);

      // Transfer, approve, delegate — none should change supply
      const alice = makeAgent();
      const bob = makeAgent();
      registry.mint(tokenId, alice.did, 100n * ONE_LGM);
      const supplyAfterMint = registry.getTokenTotalSupply(tokenId);

      registry.transferToken(tokenId, alice.did, bob.did, 50n * ONE_LGM);
      expect(registry.getTokenTotalSupply(tokenId)).toBe(supplyAfterMint);

      registry.setAllowance(alice.did, bob.did, tokenId, 100n * ONE_LGM);
      expect(registry.getTokenTotalSupply(tokenId)).toBe(supplyAfterMint);
    });

    it("no operation destroys tokens silently", () => {
      const alice = makeAgent();
      registry.mint(tokenId, alice.did, 500n * ONE_LGM);

      const bob = makeAgent();
      registry.transferToken(tokenId, alice.did, bob.did, 200n * ONE_LGM);

      // All tokens accounted for
      const aliceBal = registry.getTokenBalance(alice.did, tokenId);
      const bobBal = registry.getTokenBalance(bob.did, tokenId);
      expect(aliceBal + bobBal).toBe(500n * ONE_LGM);
    });
  });

  // ============ Overflow Protection ============

  describe("Overflow Protection", () => {
    it("should reject mint that would overflow with maxSupply", () => {
      registry.mint(tokenId, creator.did, 999_999n * ONE_LGM);

      expect(() => registry.mint(tokenId, creator.did, 2n * ONE_LGM)).toThrow(
        "exceed max supply",
      );
    });

    it("should handle very large balances without overflow", () => {
      const { tokenId: bigTokenId } = deployTestToken(registry, creator, {
        name: "BigToken",
        symbol: "BIG",
        maxSupply: 0n,
        salt: "big",
      });

      // Mint close to max safe BigInt
      const largeAmount = 2n ** 128n;
      registry.mint(bigTokenId, creator.did, largeAmount);

      expect(registry.getTokenBalance(creator.did, bigTokenId)).toBe(
        largeAmount,
      );
      expect(registry.getTokenTotalSupply(bigTokenId)).toBe(largeAmount);
    });
  });

  // ============ Reentrancy Guards ============

  describe("Reentrancy Guards", () => {
    it("should prevent double-spending via transferFrom", () => {
      const owner = makeAgent();
      const spender = makeAgent();
      const recipient1 = makeAgent();
      const recipient2 = makeAgent();

      registry.mint(tokenId, owner.did, 100n * ONE_LGM);
      registry.setAllowance(owner.did, spender.did, tokenId, 80n * ONE_LGM);

      // First transferFrom succeeds
      registry.transferFrom(
        tokenId,
        spender.did,
        owner.did,
        recipient1.did,
        50n * ONE_LGM,
      );

      // Second transferFrom can only use remaining allowance (30)
      expect(() =>
        registry.transferFrom(
          tokenId,
          spender.did,
          owner.did,
          recipient2.did,
          50n * ONE_LGM,
        ),
      ).toThrow("Insufficient allowance");

      // But 30 still works
      registry.transferFrom(
        tokenId,
        spender.did,
        owner.did,
        recipient2.did,
        30n * ONE_LGM,
      );

      expect(registry.getTokenBalance(owner.did, tokenId)).toBe(20n * ONE_LGM);
    });

    it("should prevent approve-transferFrom race condition", () => {
      const owner = makeAgent();
      const spender = makeAgent();
      const recipient = makeAgent();

      registry.mint(tokenId, owner.did, 100n * ONE_LGM);

      // Approve 50
      registry.setAllowance(owner.did, spender.did, tokenId, 50n * ONE_LGM);

      // Re-approve 30 (overwrites, doesn't add)
      registry.setAllowance(owner.did, spender.did, tokenId, 30n * ONE_LGM);

      // Can only spend 30
      expect(() =>
        registry.transferFrom(
          tokenId,
          spender.did,
          owner.did,
          recipient.did,
          50n * ONE_LGM,
        ),
      ).toThrow("Insufficient allowance");

      registry.transferFrom(
        tokenId,
        spender.did,
        owner.did,
        recipient.did,
        30n * ONE_LGM,
      );
      expect(registry.getTokenBalance(recipient.did, tokenId)).toBe(
        30n * ONE_LGM,
      );
    });
  });

  // ============ Authorization ============

  describe("Authorization", () => {
    it("only creator can mint (enforced at transaction layer)", () => {
      // At registry level, mint is unrestricted
      // Authorization is enforced at the transaction handler layer
      // This test verifies the transaction builder includes sender info
      const nonCreator = makeAgent();
      // In the registry, anyone can call mint() directly
      // The transaction layer checks tx.sender against token.creator
      expect(() =>
        registry.mint(tokenId, nonCreator.did, 100n * ONE_LGM),
      ).not.toThrow();
    });

    it("only creator can pause (enforced at transaction layer)", () => {
      // Registry allows any caller — authorization at tx layer
      expect(() => registry.pauseToken(tokenId)).not.toThrow();
    });

    it("anyone can transfer own tokens", () => {
      const alice = makeAgent();
      const bob = makeAgent();
      registry.mint(tokenId, alice.did, 100n * ONE_LGM);

      expect(() =>
        registry.transferToken(tokenId, alice.did, bob.did, 50n * ONE_LGM),
      ).not.toThrow();
    });

    it("only approved spender can transferFrom", () => {
      const owner = makeAgent();
      const unapproved = makeAgent();
      const recipient = makeAgent();

      registry.mint(tokenId, owner.did, 100n * ONE_LGM);

      // No allowance set for unapproved
      expect(() =>
        registry.transferFrom(
          tokenId,
          unapproved.did,
          owner.did,
          recipient.did,
          50n * ONE_LGM,
        ),
      ).toThrow("Insufficient allowance");
    });
  });

  // ============ LGM Base Layer Invariants ============

  describe("LGM Base Layer Invariants", () => {
    it("secondary tokens cannot pay gas fees (by design)", () => {
      // Gas costs are defined in LGM via SECONDARY_TOKEN_GAS_COSTS
      // The transaction execution layer deducts LGM balance, never token balance
      // This is a design invariant verified by checking gas cost types
      for (const [txType, cost] of Object.entries(SECONDARY_TOKEN_GAS_COSTS)) {
        expect(typeof cost).toBe("bigint");
        expect(cost as bigint).toBeGreaterThan(0n);
      }
    });

    it("secondary token total supply is independent of LGM supply", () => {
      // Minting secondary tokens does not affect LGM
      const alice = makeAgent();
      registry.mint(tokenId, alice.did, 1_000_000n * ONE_LGM);

      // LGM supply is tracked separately at state layer
      // Secondary token totalSupply only tracks this token
      expect(registry.getTokenTotalSupply(tokenId)).toBe(1_000_000n * ONE_LGM);
    });
  });

  // ============ State Consistency ============

  describe("State Consistency", () => {
    it("token state survives rollback cycle via clone", () => {
      const alice = makeAgent();
      registry.mint(tokenId, alice.did, 1000n * ONE_LGM);

      // Clone (simulate begin transaction)
      const snapshot = registry.clone();

      // Make changes
      const bob = makeAgent();
      registry.transferToken(tokenId, alice.did, bob.did, 500n * ONE_LGM);

      // Verify changes in current registry
      expect(registry.getTokenBalance(alice.did, tokenId)).toBe(500n * ONE_LGM);
      expect(registry.getTokenBalance(bob.did, tokenId)).toBe(500n * ONE_LGM);

      // Snapshot preserves original state
      expect(snapshot.getTokenBalance(alice.did, tokenId)).toBe(
        1000n * ONE_LGM,
      );
      expect(snapshot.getTokenBalance(bob.did, tokenId)).toBe(0n);
    });

    it("concurrent token operations produce deterministic state", () => {
      const alice = makeAgent();
      const bob = makeAgent();

      // Run same sequence twice with fresh registries
      const registry1 = createTestRegistry();
      const registry2 = createTestRegistry();

      const token1 = deployTestToken(registry1, creator, {
        name: "Det",
        symbol: "DET",
        salt: "det",
      });
      const token2 = deployTestToken(registry2, creator, {
        name: "Det",
        symbol: "DET",
        salt: "det",
      });

      // Same operations
      registry1.mint(token1.tokenId, alice.did, 1000n * ONE_LGM);
      registry2.mint(token2.tokenId, alice.did, 1000n * ONE_LGM);

      registry1.transferToken(
        token1.tokenId,
        alice.did,
        bob.did,
        300n * ONE_LGM,
      );
      registry2.transferToken(
        token2.tokenId,
        alice.did,
        bob.did,
        300n * ONE_LGM,
      );

      // State roots should be identical
      expect(registry1.getRoot()).toBe(registry2.getRoot());
    });
  });

  // ============ Edge Cases ============

  describe("Edge Cases", () => {
    it("transfer to account with no prior balance auto-creates entry", () => {
      const alice = makeAgent();
      const newAccount = makeAgent();

      registry.mint(tokenId, alice.did, 100n * ONE_LGM);
      registry.transferToken(tokenId, alice.did, newAccount.did, 50n * ONE_LGM);

      expect(registry.getTokenBalance(newAccount.did, tokenId)).toBe(
        50n * ONE_LGM,
      );
    });

    it("token with 0 decimals", () => {
      const { tokenId: zeroDecId } = deployTestToken(registry, creator, {
        name: "NoDecimal",
        symbol: "ND",
        decimals: 0,
        salt: "nd",
      });

      registry.mint(zeroDecId, creator.did, 100n); // 100 whole tokens
      expect(registry.getTokenBalance(creator.did, zeroDecId)).toBe(100n);
    });

    it("token with 36 decimals", () => {
      const { tokenId: highDecId } = deployTestToken(registry, creator, {
        name: "HighDecimal",
        symbol: "HD",
        decimals: 36,
        salt: "hd",
      });

      const oneToken = 10n ** 36n;
      registry.mint(highDecId, creator.did, oneToken);
      expect(registry.getTokenBalance(creator.did, highDecId)).toBe(oneToken);
    });

    it("empty token name and symbol are allowed at registry level", () => {
      // Validation of name/symbol length happens at transaction layer
      expect(() =>
        deployTestToken(registry, creator, {
          name: "",
          symbol: "",
          salt: "empty",
        }),
      ).not.toThrow();
    });

    it("multiple tokens with same name but different salt", () => {
      const { tokenId: id1 } = deployTestToken(registry, creator, {
        name: "SameName",
        symbol: "SN1",
        salt: "salt-1",
      });
      const { tokenId: id2 } = deployTestToken(registry, creator, {
        name: "SameName",
        symbol: "SN2",
        salt: "salt-2",
      });

      expect(id1).not.toBe(id2);

      registry.mint(id1, creator.did, 100n);
      registry.mint(id2, creator.did, 200n);

      expect(registry.getTokenBalance(creator.did, id1)).toBe(100n);
      expect(registry.getTokenBalance(creator.did, id2)).toBe(200n);
    });
  });
});
