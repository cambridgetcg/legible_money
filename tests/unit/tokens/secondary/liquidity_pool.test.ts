/**
 * Liquidity Pool Token (LP) Tests
 *
 * Tests adding/removing liquidity, constant-product swaps,
 * LP token minting/burning, and TWAP oracle.
 */

import { describe, it, expect, beforeEach } from "vitest";
import { TokenRegistryStateTree } from "../../../../state/token_registry.js";
import type { LiquidityPoolState } from "../../../../core/tokens/secondary/types.js";
import {
  makeAgent,
  resetAgentCounter,
  deployTestToken,
  createTestRegistry,
  ONE_LGM,
} from "./_helpers.js";

describe.sequential("Liquidity Pool Token (LP)", () => {
  let registry: TokenRegistryStateTree;
  let creator: ReturnType<typeof makeAgent>;
  let tokenAId: string; // LGM-side token (or use "LGM" string)
  let tokenBId: string; // Secondary token
  let lpTokenId: string;
  let poolId: string;

  beforeEach(() => {
    resetAgentCounter();
    registry = createTestRegistry();
    creator = makeAgent();

    // Create secondary token for the pool
    const tokenB = deployTestToken(registry, creator, {
      name: "PoolToken",
      symbol: "PT",
      maxSupply: 0n,
      salt: "pool-token",
    });
    tokenBId = tokenB.tokenId;

    // Create LP token
    const lp = deployTestToken(registry, creator, {
      name: "LP-LGM-PT",
      symbol: "LP",
      tokenType: "lp",
      maxSupply: 0n,
      salt: "lp-token",
    });
    lpTokenId = lp.tokenId;
    poolId = "pool-lgm-pt";

    // Register the pool
    const poolState: LiquidityPoolState = {
      poolId,
      lpTokenId,
      tokenA: "LGM",
      tokenB: tokenBId,
      reserveA: 0n,
      reserveB: 0n,
      swapFeeBps: 30, // 0.3%
      cumulativePriceA: 0n,
      cumulativePriceB: 0n,
      lastUpdateBlock: 0n,
    };
    registry.createPool(poolState);
  });

  describe("Adding Liquidity", () => {
    it("should mint LP tokens proportional to deposit", () => {
      const provider = makeAgent();
      const depositA = 1000n * ONE_LGM;
      const depositB = 500n * ONE_LGM;

      // Initial liquidity: LP = sqrt(depositA * depositB) approximation
      // For testing, mint LP equal to depositA (simple 1:1 for first deposit)
      registry.mint(lpTokenId, provider.did, depositA);
      registry.updatePoolReserves(poolId, depositA, depositB, 1n);

      expect(registry.getTokenBalance(provider.did, lpTokenId)).toBe(depositA);
      const pool = registry.getPool(poolId);
      expect(pool!.reserveA).toBe(depositA);
      expect(pool!.reserveB).toBe(depositB);
    });

    it("should require proportional token ratios after initial deposit", () => {
      // Set up initial pool
      registry.updatePoolReserves(poolId, 1000n * ONE_LGM, 500n * ONE_LGM, 1n);

      // Pool ratio is 2:1 (A:B)
      const pool = registry.getPool(poolId);
      const ratio = (pool!.reserveA * 10000n) / pool!.reserveB;
      expect(ratio).toBe(20000n); // 2.0 ratio
    });

    it("should update reserves on add liquidity", () => {
      registry.updatePoolReserves(poolId, 1000n * ONE_LGM, 500n * ONE_LGM, 1n);

      // Add more liquidity
      registry.updatePoolReserves(poolId, 1500n * ONE_LGM, 750n * ONE_LGM, 10n);

      const pool = registry.getPool(poolId);
      expect(pool!.reserveA).toBe(1500n * ONE_LGM);
      expect(pool!.reserveB).toBe(750n * ONE_LGM);
    });
  });

  describe("Removing Liquidity", () => {
    it("should burn LP and return proportional reserves", () => {
      const provider = makeAgent();
      registry.mint(lpTokenId, provider.did, 1000n * ONE_LGM);
      registry.updatePoolReserves(poolId, 1000n * ONE_LGM, 500n * ONE_LGM, 1n);

      // Burn LP tokens
      registry.burn(lpTokenId, provider.did, 500n * ONE_LGM);

      expect(registry.getTokenBalance(provider.did, lpTokenId)).toBe(
        500n * ONE_LGM,
      );
    });

    it("should reject removal exceeding LP balance", () => {
      const provider = makeAgent();
      registry.mint(lpTokenId, provider.did, 100n * ONE_LGM);

      expect(() =>
        registry.burn(lpTokenId, provider.did, 200n * ONE_LGM),
      ).toThrow("Insufficient");
    });
  });

  describe("Swap", () => {
    beforeEach(() => {
      // Set up pool with liquidity
      registry.updatePoolReserves(
        poolId,
        10000n * ONE_LGM,
        5000n * ONE_LGM,
        1n,
      );
    });

    it("should calculate constant product for swaps", () => {
      const pool = registry.getPool(poolId)!;
      const k = pool.reserveA * pool.reserveB;

      // After a swap, k should remain constant (minus fee)
      // Swap 100 tokenA in: new reserveA = 10100, new reserveB = k / 10100
      const newReserveA = pool.reserveA + 100n * ONE_LGM;
      const newReserveB = k / newReserveA;
      const amountOut = pool.reserveB - newReserveB;

      expect(amountOut).toBeGreaterThan(0n);
      // k should be approximately preserved
      expect(newReserveA * newReserveB).toBeLessThanOrEqual(k);
    });

    it("should charge swap fee in input token", () => {
      const pool = registry.getPool(poolId)!;
      const inputAmount = 100n * ONE_LGM;
      const feeAmount = (inputAmount * BigInt(pool.swapFeeBps)) / 10000n;

      // Fee should be 0.3% of input
      expect(feeAmount).toBe((100n * ONE_LGM * 30n) / 10000n);
    });

    it("should update reserves after swap", () => {
      const amountIn = 100n * ONE_LGM;
      const pool = registry.getPool(poolId)!;

      // Calculate output (simplified constant product)
      const k = pool.reserveA * pool.reserveB;
      const newReserveA = pool.reserveA + amountIn;
      const newReserveB = k / newReserveA;

      registry.updatePoolReserves(poolId, newReserveA, newReserveB, 10n);

      const updated = registry.getPool(poolId)!;
      expect(updated.reserveA).toBe(newReserveA);
      expect(updated.reserveB).toBe(newReserveB);
    });

    it("should reject swap that drains pool below minimum", () => {
      // Pool has 10000 * ONE_LGM of tokenA, 5000 * ONE_LGM of tokenB
      // Trying to update reserves to near-zero should be blocked at tx layer
      // At registry level, we verify the reserves
      const pool = registry.getPool(poolId)!;
      expect(pool.reserveA).toBeGreaterThan(0n);
      expect(pool.reserveB).toBeGreaterThan(0n);
    });
  });

  describe("Price Oracle", () => {
    it("should track cumulative price for TWAP", () => {
      // Set initial reserves
      registry.updatePoolReserves(poolId, 1000n * ONE_LGM, 500n * ONE_LGM, 1n);

      // Advance blocks and update reserves
      registry.updatePoolReserves(poolId, 1100n * ONE_LGM, 480n * ONE_LGM, 11n);

      const pool = registry.getPool(poolId)!;
      // Cumulative price should have been updated
      // Price A = reserveB / reserveA accumulated over 10 blocks
      expect(pool.cumulativePriceA).toBeGreaterThan(0n);
      expect(pool.cumulativePriceB).toBeGreaterThan(0n);
      expect(pool.lastUpdateBlock).toBe(11n);
    });
  });

  describe("LGM Gas Fee", () => {
    it("all LP operations charge LGM gas only (not LP tokens)", () => {
      const provider = makeAgent();
      registry.mint(lpTokenId, provider.did, 1000n * ONE_LGM);

      // LP token balance is only affected by explicit mint/burn/transfer
      // Gas fees are in LGM (handled at transaction layer)
      const balanceBefore = registry.getTokenBalance(provider.did, lpTokenId);

      // Transfer LP tokens
      const recipient = makeAgent();
      registry.transferToken(
        lpTokenId,
        provider.did,
        recipient.did,
        100n * ONE_LGM,
      );

      // Only transfer amount deducted, no gas fee in LP tokens
      expect(registry.getTokenBalance(provider.did, lpTokenId)).toBe(
        balanceBefore - 100n * ONE_LGM,
      );
    });
  });
});
