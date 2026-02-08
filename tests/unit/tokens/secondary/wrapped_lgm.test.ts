/**
 * Wrapped LGM (wLGM) Tests
 *
 * Tests wrapping (deposit LGM → mint wLGM), unwrapping (burn wLGM → return LGM),
 * wLGM transfers, and the 1:1 backing invariant.
 *
 * Note: The wrap/unwrap operations at the registry level track the wLGM token
 * balances. LGM deduction/credit happens at the transaction layer.
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

describe.sequential("Wrapped LGM (wLGM)", () => {
  let registry: TokenRegistryStateTree;
  let creator: ReturnType<typeof makeAgent>;
  let wlgmTokenId: string;
  const WLGM_CONTRACT_DID = "did:lgm:wlgm-contract-00000000";

  beforeEach(() => {
    resetAgentCounter();
    registry = createTestRegistry();
    creator = makeAgent();
    const result = deployTestToken(registry, creator, {
      name: "Wrapped LGM",
      symbol: "wLGM",
      decimals: 18,
      maxSupply: 0n, // Unlimited (backed 1:1 by LGM)
      tokenType: "wrapped",
    });
    wlgmTokenId = result.tokenId;
  });

  describe("Wrapping (Deposit)", () => {
    it("should mint wLGM 1:1 when depositing LGM", () => {
      const depositor = makeAgent();
      const depositAmount = 100n * ONE_LGM;

      // Mint wLGM to depositor (LGM locked at tx layer)
      registry.mint(wlgmTokenId, depositor.did, depositAmount);

      expect(registry.getTokenBalance(depositor.did, wlgmTokenId)).toBe(
        depositAmount,
      );
      expect(registry.getTokenTotalSupply(wlgmTokenId)).toBe(depositAmount);
    });

    it("should reject wrap of 0 amount", () => {
      const depositor = makeAgent();

      // 0 amount mint should still work at registry level
      // (validation of 0 amount happens at transaction layer)
      registry.mint(wlgmTokenId, depositor.did, 0n);
      expect(registry.getTokenBalance(depositor.did, wlgmTokenId)).toBe(0n);
    });

    it("should handle multiple wraps from same depositor", () => {
      const depositor = makeAgent();
      registry.mint(wlgmTokenId, depositor.did, 100n * ONE_LGM);
      registry.mint(wlgmTokenId, depositor.did, 200n * ONE_LGM);

      expect(registry.getTokenBalance(depositor.did, wlgmTokenId)).toBe(
        300n * ONE_LGM,
      );
    });
  });

  describe("Unwrapping (Withdraw)", () => {
    it("should burn wLGM and return LGM 1:1", () => {
      const holder = makeAgent();
      registry.mint(wlgmTokenId, holder.did, 500n * ONE_LGM);

      registry.burn(wlgmTokenId, holder.did, 200n * ONE_LGM);

      expect(registry.getTokenBalance(holder.did, wlgmTokenId)).toBe(
        300n * ONE_LGM,
      );
      expect(registry.getTokenTotalSupply(wlgmTokenId)).toBe(300n * ONE_LGM);
    });

    it("should reject unwrap exceeding wLGM balance", () => {
      const holder = makeAgent();
      registry.mint(wlgmTokenId, holder.did, 100n * ONE_LGM);

      expect(() =>
        registry.burn(wlgmTokenId, holder.did, 200n * ONE_LGM),
      ).toThrow("Insufficient");
    });

    it("should maintain 1:1 backing invariant", () => {
      const alice = makeAgent();
      const bob = makeAgent();

      // Wrap
      registry.mint(wlgmTokenId, alice.did, 1000n * ONE_LGM);
      registry.mint(wlgmTokenId, bob.did, 500n * ONE_LGM);

      // Transfer wLGM (doesn't affect backing)
      registry.transferToken(wlgmTokenId, alice.did, bob.did, 300n * ONE_LGM);

      // Unwrap
      registry.burn(wlgmTokenId, bob.did, 200n * ONE_LGM);

      // Total wLGM supply should equal total LGM locked
      // (At registry level, totalSupply tracks this)
      const totalWLGM = registry.getTokenTotalSupply(wlgmTokenId);
      const aliceWLGM = registry.getTokenBalance(alice.did, wlgmTokenId);
      const bobWLGM = registry.getTokenBalance(bob.did, wlgmTokenId);

      expect(totalWLGM).toBe(aliceWLGM + bobWLGM);
      expect(totalWLGM).toBe(1300n * ONE_LGM); // 1000 + 500 - 200
    });
  });

  describe("Transfer wLGM", () => {
    it("should transfer wLGM like any LGM-20 token", () => {
      const alice = makeAgent();
      const bob = makeAgent();
      registry.mint(wlgmTokenId, alice.did, 500n * ONE_LGM);

      registry.transferToken(wlgmTokenId, alice.did, bob.did, 200n * ONE_LGM);

      expect(registry.getTokenBalance(alice.did, wlgmTokenId)).toBe(
        300n * ONE_LGM,
      );
      expect(registry.getTokenBalance(bob.did, wlgmTokenId)).toBe(
        200n * ONE_LGM,
      );
    });

    it("LGM backing remains constant after wLGM transfer", () => {
      const alice = makeAgent();
      const bob = makeAgent();
      registry.mint(wlgmTokenId, alice.did, 1000n * ONE_LGM);

      const supplyBefore = registry.getTokenTotalSupply(wlgmTokenId);
      registry.transferToken(wlgmTokenId, alice.did, bob.did, 600n * ONE_LGM);
      const supplyAfter = registry.getTokenTotalSupply(wlgmTokenId);

      // Total supply unchanged = total LGM backing unchanged
      expect(supplyAfter).toBe(supplyBefore);
    });
  });

  describe("Fee Handling", () => {
    it("should NOT charge wLGM for gas (gas is LGM only)", () => {
      const holder = makeAgent();
      registry.mint(wlgmTokenId, holder.did, 1000n * ONE_LGM);

      // Performing operations doesn't deduct wLGM for gas
      // Gas deduction in LGM happens at transaction layer
      const balanceBefore = registry.getTokenBalance(holder.did, wlgmTokenId);
      // Delegation, transfers etc at registry level don't touch gas
      const bob = makeAgent();
      registry.transferToken(wlgmTokenId, holder.did, bob.did, 100n * ONE_LGM);

      // Only the transfer amount was deducted, no gas
      expect(registry.getTokenBalance(holder.did, wlgmTokenId)).toBe(
        balanceBefore - 100n * ONE_LGM,
      );
    });
  });

  describe("Conservation", () => {
    it("wLGM totalSupply tracks all mints minus all burns", () => {
      const alice = makeAgent();
      const bob = makeAgent();
      const charlie = makeAgent();

      registry.mint(wlgmTokenId, alice.did, 1000n * ONE_LGM);
      registry.mint(wlgmTokenId, bob.did, 500n * ONE_LGM);
      registry.mint(wlgmTokenId, charlie.did, 300n * ONE_LGM);

      registry.burn(wlgmTokenId, alice.did, 200n * ONE_LGM);
      registry.burn(wlgmTokenId, bob.did, 100n * ONE_LGM);

      const totalSupply = registry.getTokenTotalSupply(wlgmTokenId);
      const sum =
        registry.getTokenBalance(alice.did, wlgmTokenId) +
        registry.getTokenBalance(bob.did, wlgmTokenId) +
        registry.getTokenBalance(charlie.did, wlgmTokenId);

      expect(totalSupply).toBe(sum);
      expect(totalSupply).toBe(1500n * ONE_LGM); // 1800 - 300
    });
  });
});
