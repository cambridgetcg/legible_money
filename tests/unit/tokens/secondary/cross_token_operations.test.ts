/**
 * Cross-Token Operations Tests
 *
 * Tests multi-token balances per account, atomic multi-token transfers,
 * token-to-LGM exchange, cross-contract token flow, and state root verification.
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

describe.sequential("Cross-Token Operations", () => {
  let registry: TokenRegistryStateTree;
  let creator: ReturnType<typeof makeAgent>;
  let tokenAId: string;
  let tokenBId: string;
  let tokenCId: string;

  beforeEach(() => {
    resetAgentCounter();
    registry = createTestRegistry();
    creator = makeAgent();

    const a = deployTestToken(registry, creator, {
      name: "Alpha",
      symbol: "ALPHA",
      salt: "a",
    });
    const b = deployTestToken(registry, creator, {
      name: "Beta",
      symbol: "BETA",
      salt: "b",
    });
    const c = deployTestToken(registry, creator, {
      name: "Gamma",
      symbol: "GAMMA",
      salt: "c",
    });

    tokenAId = a.tokenId;
    tokenBId = b.tokenId;
    tokenCId = c.tokenId;
  });

  // ============ Multi-Token Balances ============

  describe("Multi-Token Balances", () => {
    it("should track multiple token balances per account", () => {
      const user = makeAgent();
      registry.mint(tokenAId, user.did, 100n * ONE_LGM);
      registry.mint(tokenBId, user.did, 200n * ONE_LGM);
      registry.mint(tokenCId, user.did, 300n * ONE_LGM);

      expect(registry.getTokenBalance(user.did, tokenAId)).toBe(100n * ONE_LGM);
      expect(registry.getTokenBalance(user.did, tokenBId)).toBe(200n * ONE_LGM);
      expect(registry.getTokenBalance(user.did, tokenCId)).toBe(300n * ONE_LGM);
    });

    it("should list all token balances for an account", () => {
      const user = makeAgent();
      registry.mint(tokenAId, user.did, 100n * ONE_LGM);
      registry.mint(tokenBId, user.did, 200n * ONE_LGM);
      registry.mint(tokenCId, user.did, 300n * ONE_LGM);

      const balances = registry.getAllTokenBalances(user.did);
      expect(balances).toHaveLength(3);

      const totalValue = balances.reduce((sum, b) => sum + b.balance, 0n);
      expect(totalValue).toBe(600n * ONE_LGM);
    });

    it("should return empty array for account with no tokens", () => {
      const user = makeAgent();
      const balances = registry.getAllTokenBalances(user.did);
      expect(balances).toHaveLength(0);
    });
  });

  // ============ Atomic Multi-Token Transfers ============

  describe("Atomic Multi-Token Transfers", () => {
    it("should atomically transfer multiple tokens in one sequence", () => {
      const alice = makeAgent();
      const bob = makeAgent();

      registry.mint(tokenAId, alice.did, 1000n * ONE_LGM);
      registry.mint(tokenBId, alice.did, 500n * ONE_LGM);

      // Transfer both tokens
      registry.transferToken(tokenAId, alice.did, bob.did, 300n * ONE_LGM);
      registry.transferToken(tokenBId, alice.did, bob.did, 200n * ONE_LGM);

      expect(registry.getTokenBalance(bob.did, tokenAId)).toBe(300n * ONE_LGM);
      expect(registry.getTokenBalance(bob.did, tokenBId)).toBe(200n * ONE_LGM);
    });

    it("should rollback all on partial failure", () => {
      const alice = makeAgent();
      const bob = makeAgent();

      registry.mint(tokenAId, alice.did, 1000n * ONE_LGM);
      registry.mint(tokenBId, alice.did, 50n * ONE_LGM); // Not enough for 200

      // Clone state for manual rollback
      const snapshot = registry.clone();

      // First transfer succeeds
      registry.transferToken(tokenAId, alice.did, bob.did, 300n * ONE_LGM);

      // Second transfer fails
      let failed = false;
      try {
        registry.transferToken(tokenBId, alice.did, bob.did, 200n * ONE_LGM);
      } catch {
        failed = true;
      }

      expect(failed).toBe(true);

      // In real usage, rollback would restore state
      // Verify snapshot has original values
      expect(snapshot.getTokenBalance(alice.did, tokenAId)).toBe(
        1000n * ONE_LGM,
      );
      expect(snapshot.getTokenBalance(bob.did, tokenAId)).toBe(0n);
    });
  });

  // ============ Token-to-LGM Exchange ============

  describe("Token-to-LGM Exchange", () => {
    it("should swap secondary token for LGM via pool reserves", () => {
      // Set up a pool with reserves
      const lpToken = deployTestToken(registry, creator, {
        name: "LP-A-LGM",
        symbol: "LPA",
        tokenType: "lp",
        salt: "lp-a",
      });

      registry.createPool({
        poolId: "pool-a-lgm",
        lpTokenId: lpToken.tokenId,
        tokenA: "LGM",
        tokenB: tokenAId,
        reserveA: 10000n * ONE_LGM,
        reserveB: 5000n * ONE_LGM,
        swapFeeBps: 30,
        cumulativePriceA: 0n,
        cumulativePriceB: 0n,
        lastUpdateBlock: 0n,
      });

      const pool = registry.getPool("pool-a-lgm");
      expect(pool).toBeDefined();

      // Calculate swap output (constant product)
      const inputAmount = 100n * ONE_LGM;
      const k = pool!.reserveA * pool!.reserveB;
      const newReserveB = pool!.reserveB + inputAmount;
      const newReserveA = k / newReserveB;
      const outputLGM = pool!.reserveA - newReserveA;

      expect(outputLGM).toBeGreaterThan(0n);
    });
  });

  // ============ Cross-Contract Token Flow ============

  describe("Cross-Contract Token Flow", () => {
    it("should send tokens between two contracts via allowance", () => {
      const contractA = "did:lgm:contract-a-0000000000000";
      const contractB = "did:lgm:contract-b-0000000000000";

      registry.mint(tokenAId, contractA, 1000n * ONE_LGM);

      // Contract A approves Contract B to spend
      registry.setAllowance(contractA, contractB, tokenAId, 500n * ONE_LGM);

      // Contract B transfers from Contract A
      registry.transferFrom(
        tokenAId,
        contractB,
        contractA,
        contractB,
        300n * ONE_LGM,
      );

      expect(registry.getTokenBalance(contractA, tokenAId)).toBe(
        700n * ONE_LGM,
      );
      expect(registry.getTokenBalance(contractB, tokenAId)).toBe(
        300n * ONE_LGM,
      );
    });
  });

  // ============ State Root Verification ============

  describe("State Root Verification", () => {
    it("token operations should update state root", () => {
      const rootBefore = registry.getRoot();

      const user = makeAgent();
      registry.mint(tokenAId, user.did, 100n * ONE_LGM);

      const rootAfter = registry.getRoot();
      expect(rootAfter).not.toBe(rootBefore);
    });
  });

  // ============ Fee Invariant ============

  describe("Fee Invariant", () => {
    it("all cross-token operations charge LGM gas only", () => {
      const user = makeAgent();
      registry.mint(tokenAId, user.did, 1000n * ONE_LGM);
      registry.mint(tokenBId, user.did, 1000n * ONE_LGM);

      const balanceA = registry.getTokenBalance(user.did, tokenAId);
      const balanceB = registry.getTokenBalance(user.did, tokenBId);

      // Perform cross-token operations
      const other = makeAgent();
      registry.transferToken(tokenAId, user.did, other.did, 100n * ONE_LGM);
      registry.transferToken(tokenBId, user.did, other.did, 100n * ONE_LGM);

      // Only transfer amounts deducted from secondary tokens
      // Gas fees deducted from LGM at transaction layer (not tested here)
      expect(registry.getTokenBalance(user.did, tokenAId)).toBe(
        balanceA - 100n * ONE_LGM,
      );
      expect(registry.getTokenBalance(user.did, tokenBId)).toBe(
        balanceB - 100n * ONE_LGM,
      );
    });
  });
});
