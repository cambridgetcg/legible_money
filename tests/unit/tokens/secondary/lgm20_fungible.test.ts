/**
 * LGM-20 Fungible Token Tests
 *
 * Tests the core ERC-20-equivalent functionality:
 * mint, burn, transfer, approve, transferFrom, and supply conservation.
 */

import { describe, it, expect, beforeEach } from "vitest";
import { TokenRegistryStateTree } from "../../../../state/token_registry.js";
import {
  makeAgent,
  resetAgentCounter,
  deployTestToken,
  deployAndMint,
  createTestRegistry,
  ONE_LGM,
} from "./_helpers.js";

describe.sequential("LGM-20 Fungible Token", () => {
  let registry: TokenRegistryStateTree;
  let creator: ReturnType<typeof makeAgent>;
  let tokenId: string;

  beforeEach(() => {
    resetAgentCounter();
    registry = createTestRegistry();
    creator = makeAgent();
    const result = deployTestToken(registry, creator, {
      name: "LGM20Token",
      symbol: "L20",
      maxSupply: 1_000_000n * ONE_LGM,
    });
    tokenId = result.tokenId;
  });

  // ============ Minting ============

  describe("Minting", () => {
    it("should mint tokens to recipient", () => {
      const recipient = makeAgent();
      registry.mint(tokenId, recipient.did, 500n * ONE_LGM);

      expect(registry.getTokenBalance(recipient.did, tokenId)).toBe(
        500n * ONE_LGM,
      );
      expect(registry.getTokenTotalSupply(tokenId)).toBe(500n * ONE_LGM);
    });

    it("should reject mint exceeding max supply", () => {
      registry.mint(tokenId, creator.did, 999_999n * ONE_LGM);

      expect(() => registry.mint(tokenId, creator.did, 2n * ONE_LGM)).toThrow(
        "exceed max supply",
      );
    });

    it("should mint exactly up to max supply", () => {
      registry.mint(tokenId, creator.did, 1_000_000n * ONE_LGM);
      expect(registry.getTokenTotalSupply(tokenId)).toBe(1_000_000n * ONE_LGM);
    });

    it("should allow unlimited minting when maxSupply is 0", () => {
      const { tokenId: unlimitedId } = deployTestToken(registry, creator, {
        name: "Unlimited",
        symbol: "UNL",
        maxSupply: 0n,
        salt: "unlimited",
      });

      registry.mint(unlimitedId, creator.did, 10n ** 30n);
      expect(registry.getTokenBalance(creator.did, unlimitedId)).toBe(
        10n ** 30n,
      );
    });

    it("should reject mint for non-existent token", () => {
      expect(() => registry.mint("nonexistent", creator.did, 100n)).toThrow(
        "not found",
      );
    });
  });

  // ============ Burning ============

  describe("Burning", () => {
    beforeEach(() => {
      registry.mint(tokenId, creator.did, 1000n * ONE_LGM);
    });

    it("should burn tokens from holder", () => {
      registry.burn(tokenId, creator.did, 300n * ONE_LGM);

      expect(registry.getTokenBalance(creator.did, tokenId)).toBe(
        700n * ONE_LGM,
      );
    });

    it("should reject burn of more than balance", () => {
      expect(() =>
        registry.burn(tokenId, creator.did, 1001n * ONE_LGM),
      ).toThrow("Insufficient");
    });

    it("should reduce totalSupply on burn", () => {
      registry.burn(tokenId, creator.did, 400n * ONE_LGM);

      expect(registry.getTokenTotalSupply(tokenId)).toBe(600n * ONE_LGM);
    });

    it("should allow burning entire balance", () => {
      registry.burn(tokenId, creator.did, 1000n * ONE_LGM);

      expect(registry.getTokenBalance(creator.did, tokenId)).toBe(0n);
      expect(registry.getTokenTotalSupply(tokenId)).toBe(0n);
    });
  });

  // ============ Transfer ============

  describe("Transfer", () => {
    let alice: ReturnType<typeof makeAgent>;
    let bob: ReturnType<typeof makeAgent>;

    beforeEach(() => {
      alice = makeAgent();
      bob = makeAgent();
      registry.mint(tokenId, alice.did, 1000n * ONE_LGM);
    });

    it("should transfer tokens between accounts", () => {
      registry.transferToken(tokenId, alice.did, bob.did, 250n * ONE_LGM);

      expect(registry.getTokenBalance(alice.did, tokenId)).toBe(750n * ONE_LGM);
      expect(registry.getTokenBalance(bob.did, tokenId)).toBe(250n * ONE_LGM);
    });

    it("should reject transfer with insufficient token balance", () => {
      expect(() =>
        registry.transferToken(tokenId, alice.did, bob.did, 1001n * ONE_LGM),
      ).toThrow("Insufficient");
    });

    it("should reject transfer to self", () => {
      expect(() =>
        registry.transferToken(tokenId, alice.did, alice.did, 100n * ONE_LGM),
      ).toThrow("self");
    });

    it("should reject transfer of 0 amount", () => {
      expect(() =>
        registry.transferToken(tokenId, alice.did, bob.did, 0n),
      ).toThrow("0 amount");
    });

    it("should reject transfer when token is paused", () => {
      registry.pauseToken(tokenId);

      expect(() =>
        registry.transferToken(tokenId, alice.did, bob.did, 100n * ONE_LGM),
      ).toThrow("paused");
    });

    it("should preserve total supply across transfers", () => {
      const supplyBefore = registry.getTokenTotalSupply(tokenId);

      registry.transferToken(tokenId, alice.did, bob.did, 500n * ONE_LGM);

      expect(registry.getTokenTotalSupply(tokenId)).toBe(supplyBefore);
    });

    it("should transfer to account with no prior balance", () => {
      const charlie = makeAgent();
      registry.transferToken(tokenId, alice.did, charlie.did, 100n * ONE_LGM);

      expect(registry.getTokenBalance(charlie.did, tokenId)).toBe(
        100n * ONE_LGM,
      );
    });
  });

  // ============ Approve & TransferFrom ============

  describe("Approve & TransferFrom", () => {
    let owner: ReturnType<typeof makeAgent>;
    let spender: ReturnType<typeof makeAgent>;
    let recipient: ReturnType<typeof makeAgent>;

    beforeEach(() => {
      owner = makeAgent();
      spender = makeAgent();
      recipient = makeAgent();
      registry.mint(tokenId, owner.did, 1000n * ONE_LGM);
    });

    it("should set allowance for spender", () => {
      registry.setAllowance(owner.did, spender.did, tokenId, 500n * ONE_LGM);

      expect(registry.getAllowance(owner.did, spender.did, tokenId)).toBe(
        500n * ONE_LGM,
      );
    });

    it("should allow approved spender to transferFrom", () => {
      registry.setAllowance(owner.did, spender.did, tokenId, 500n * ONE_LGM);

      registry.transferFrom(
        tokenId,
        spender.did,
        owner.did,
        recipient.did,
        200n * ONE_LGM,
      );

      expect(registry.getTokenBalance(owner.did, tokenId)).toBe(800n * ONE_LGM);
      expect(registry.getTokenBalance(recipient.did, tokenId)).toBe(
        200n * ONE_LGM,
      );
      // Allowance reduced
      expect(registry.getAllowance(owner.did, spender.did, tokenId)).toBe(
        300n * ONE_LGM,
      );
    });

    it("should reject transferFrom exceeding allowance", () => {
      registry.setAllowance(owner.did, spender.did, tokenId, 100n * ONE_LGM);

      expect(() =>
        registry.transferFrom(
          tokenId,
          spender.did,
          owner.did,
          recipient.did,
          200n * ONE_LGM,
        ),
      ).toThrow("Insufficient allowance");
    });

    it("should reject transferFrom with zero allowance", () => {
      expect(() =>
        registry.transferFrom(
          tokenId,
          spender.did,
          owner.did,
          recipient.did,
          100n * ONE_LGM,
        ),
      ).toThrow("Insufficient allowance");
    });

    it("should reset allowance to new value on re-approve", () => {
      registry.setAllowance(owner.did, spender.did, tokenId, 500n * ONE_LGM);
      registry.setAllowance(owner.did, spender.did, tokenId, 200n * ONE_LGM);

      expect(registry.getAllowance(owner.did, spender.did, tokenId)).toBe(
        200n * ONE_LGM,
      );
    });

    it("should remove allowance when set to 0", () => {
      registry.setAllowance(owner.did, spender.did, tokenId, 500n * ONE_LGM);
      registry.setAllowance(owner.did, spender.did, tokenId, 0n);

      expect(registry.getAllowance(owner.did, spender.did, tokenId)).toBe(0n);
    });
  });

  // ============ Supply Conservation Invariant ============

  describe("Supply Conservation Invariant", () => {
    it("totalSupply should equal sum of all balances after complex operations", () => {
      // Mint to 5 accounts
      const accounts = Array.from({ length: 5 }, () => makeAgent());
      for (const acc of accounts) {
        registry.mint(tokenId, acc.did, 200n * ONE_LGM);
      }

      // Transfer between them
      registry.transferToken(
        tokenId,
        accounts[0].did,
        accounts[1].did,
        50n * ONE_LGM,
      );
      registry.transferToken(
        tokenId,
        accounts[1].did,
        accounts[2].did,
        100n * ONE_LGM,
      );
      registry.transferToken(
        tokenId,
        accounts[3].did,
        accounts[4].did,
        200n * ONE_LGM,
      );

      // Burn some
      registry.burn(tokenId, accounts[2].did, 30n * ONE_LGM);

      // Verify invariant
      const totalSupply = registry.getTokenTotalSupply(tokenId);
      let sumBalances = 0n;
      for (const acc of accounts) {
        sumBalances += registry.getTokenBalance(acc.did, tokenId);
      }

      expect(totalSupply).toBe(sumBalances);
    });

    it("totalSupply should be 0 after burning all minted tokens", () => {
      const acc = makeAgent();
      registry.mint(tokenId, acc.did, 500n * ONE_LGM);
      registry.burn(tokenId, acc.did, 500n * ONE_LGM);

      expect(registry.getTokenTotalSupply(tokenId)).toBe(0n);
      expect(registry.getTokenBalance(acc.did, tokenId)).toBe(0n);
    });

    it("transfers should never change totalSupply", () => {
      const a = makeAgent();
      const b = makeAgent();
      const c = makeAgent();

      registry.mint(tokenId, a.did, 1000n * ONE_LGM);
      const supplyAfterMint = registry.getTokenTotalSupply(tokenId);

      registry.transferToken(tokenId, a.did, b.did, 300n * ONE_LGM);
      registry.transferToken(tokenId, b.did, c.did, 100n * ONE_LGM);
      registry.transferToken(tokenId, c.did, a.did, 50n * ONE_LGM);

      expect(registry.getTokenTotalSupply(tokenId)).toBe(supplyAfterMint);
    });
  });
});
