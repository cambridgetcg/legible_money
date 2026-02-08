/**
 * Token Registry & Lifecycle Tests
 *
 * Tests token creation, metadata management, pausing, and querying
 * against the TokenRegistryStateTree.
 */

import { describe, it, expect, beforeEach } from "vitest";
import { TokenRegistryStateTree } from "../../../../state/token_registry.js";
import type { TokenDefinition } from "../../../../core/tokens/secondary/types.js";
import {
  makeAgent,
  resetAgentCounter,
  deployTestToken,
  createTestRegistry,
  ONE_LGM,
} from "./_helpers.js";

describe.sequential("Token Registry & Lifecycle", () => {
  let registry: TokenRegistryStateTree;
  let creator: ReturnType<typeof makeAgent>;

  beforeEach(() => {
    resetAgentCounter();
    registry = createTestRegistry();
    creator = makeAgent();
  });

  // ============ Token Creation ============

  describe("Token Creation", () => {
    it("should create a fungible token with valid parameters", () => {
      const { tokenId, definition } = deployTestToken(registry, creator, {
        name: "AlphaToken",
        symbol: "ALPHA",
        decimals: 18,
        maxSupply: 1_000_000n * ONE_LGM,
        tokenType: "fungible",
      });

      const stored = registry.getTokenDefinition(tokenId);
      expect(stored).toBeDefined();
      expect(stored!.name).toBe("AlphaToken");
      expect(stored!.symbol).toBe("ALPHA");
      expect(stored!.decimals).toBe(18);
      expect(stored!.maxSupply).toBe(1_000_000n * ONE_LGM);
      expect(stored!.totalSupply).toBe(0n);
      expect(stored!.tokenType).toBe("fungible");
      expect(stored!.creator).toBe(creator.did);
      expect(stored!.paused).toBe(false);
    });

    it("should generate deterministic token ID from creator + name + salt", () => {
      const salt = "fixed-salt";
      const id1 = registry.generateTokenId(creator.did, "MyToken", salt);
      const id2 = registry.generateTokenId(creator.did, "MyToken", salt);
      expect(id1).toBe(id2);

      // Different name → different ID
      const id3 = registry.generateTokenId(creator.did, "OtherToken", salt);
      expect(id3).not.toBe(id1);

      // Different creator → different ID
      const other = makeAgent();
      const id4 = registry.generateTokenId(other.did, "MyToken", salt);
      expect(id4).not.toBe(id1);

      // Different salt → different ID
      const id5 = registry.generateTokenId(
        creator.did,
        "MyToken",
        "other-salt",
      );
      expect(id5).not.toBe(id1);
    });

    it("should reject duplicate token ID", () => {
      const salt = "fixed-salt";
      deployTestToken(registry, creator, {
        name: "Token",
        symbol: "TK1",
        salt,
      });

      expect(() =>
        deployTestToken(registry, creator, {
          name: "Token",
          symbol: "TK2",
          salt,
        }),
      ).toThrow("already exists");
    });

    it("should reject duplicate token symbol per creator", () => {
      deployTestToken(registry, creator, {
        name: "First",
        symbol: "DUP",
        salt: "salt-1",
      });

      expect(() =>
        deployTestToken(registry, creator, {
          name: "Second",
          symbol: "DUP",
          salt: "salt-2",
        }),
      ).toThrow("already used by creator");
    });

    it("should allow same symbol from different creators", () => {
      const creator2 = makeAgent();
      deployTestToken(registry, creator, {
        name: "First",
        symbol: "SAME",
        salt: "salt-1",
      });

      expect(() =>
        deployTestToken(registry, creator2, {
          name: "Second",
          symbol: "SAME",
          salt: "salt-2",
        }),
      ).not.toThrow();
    });

    it("should enforce maxSupply of 0 as unlimited", () => {
      const { tokenId } = deployTestToken(registry, creator, {
        maxSupply: 0n,
      });

      // Should be able to mint arbitrarily large amounts
      registry.mint(tokenId, creator.did, 10n ** 30n);
      expect(registry.getTokenBalance(creator.did, tokenId)).toBe(10n ** 30n);
    });

    it("should track token in creator registry", () => {
      deployTestToken(registry, creator, {
        name: "A",
        symbol: "A",
        salt: "s1",
      });
      deployTestToken(registry, creator, {
        name: "B",
        symbol: "B",
        salt: "s2",
      });

      const tokens = registry.listTokens();
      const creatorTokens = tokens.filter((t) => t.creator === creator.did);
      expect(creatorTokens).toHaveLength(2);
    });
  });

  // ============ Token Metadata ============

  describe("Token Metadata", () => {
    let tokenId: string;

    beforeEach(() => {
      const result = deployTestToken(registry, creator, {
        metadata: { description: "Original", icon: "token.png" },
      });
      tokenId = result.tokenId;
    });

    it("should store and retrieve token metadata", () => {
      const def = registry.getTokenDefinition(tokenId);
      expect(def!.metadata.description).toBe("Original");
      expect(def!.metadata.icon).toBe("token.png");
    });

    it("should allow creator to update metadata", () => {
      registry.updateTokenMetadata(tokenId, {
        description: "Updated",
        website: "https://example.com",
      });

      const def = registry.getTokenDefinition(tokenId);
      expect(def!.metadata.description).toBe("Updated");
      expect(def!.metadata.website).toBe("https://example.com");
      expect(def!.metadata.icon).toBe("token.png"); // Preserved
    });

    it("should reject metadata update for non-existent token", () => {
      expect(() =>
        registry.updateTokenMetadata("nonexistent", { foo: "bar" }),
      ).toThrow("not found");
    });
  });

  // ============ Token Pausing ============

  describe("Token Pausing", () => {
    let tokenId: string;

    beforeEach(() => {
      const result = deployTestToken(registry, creator);
      tokenId = result.tokenId;
      registry.mint(tokenId, creator.did, 1000n * ONE_LGM);
    });

    it("should allow pausing a token", () => {
      registry.pauseToken(tokenId);
      const def = registry.getTokenDefinition(tokenId);
      expect(def!.paused).toBe(true);
    });

    it("should reject transfers of paused token", () => {
      const recipient = makeAgent();
      registry.pauseToken(tokenId);

      expect(() =>
        registry.transferToken(tokenId, creator.did, recipient.did, 100n),
      ).toThrow("paused");
    });

    it("should allow unpausing a token", () => {
      registry.pauseToken(tokenId);
      registry.unpauseToken(tokenId);

      const def = registry.getTokenDefinition(tokenId);
      expect(def!.paused).toBe(false);

      // Transfers work again
      const recipient = makeAgent();
      expect(() =>
        registry.transferToken(tokenId, creator.did, recipient.did, 100n),
      ).not.toThrow();
    });

    it("should reject pause for non-existent token", () => {
      expect(() => registry.pauseToken("nonexistent")).toThrow("not found");
    });
  });

  // ============ Token Querying ============

  describe("Token Querying", () => {
    beforeEach(() => {
      deployTestToken(registry, creator, {
        name: "Fungible1",
        symbol: "F1",
        tokenType: "fungible",
        salt: "s1",
      });
      deployTestToken(registry, creator, {
        name: "Gov1",
        symbol: "GOV1",
        tokenType: "governance",
        salt: "s2",
      });
      deployTestToken(registry, creator, {
        name: "Utility1",
        symbol: "UT1",
        tokenType: "utility",
        salt: "s3",
      });

      const creator2 = makeAgent();
      deployTestToken(registry, creator2, {
        name: "Fungible2",
        symbol: "F2",
        tokenType: "fungible",
        salt: "s4",
      });
    });

    it("should list all tokens", () => {
      const tokens = registry.listTokens();
      expect(tokens).toHaveLength(4);
    });

    it("should query token by ID", () => {
      const tokens = registry.listTokens();
      const firstId = tokens[0].tokenId;

      const token = registry.getTokenDefinition(firstId);
      expect(token).toBeDefined();
      expect(token!.tokenId).toBe(firstId);
    });

    it("should query tokens by type", () => {
      const fungibles = registry.getTokensByType("fungible");
      expect(fungibles).toHaveLength(2);

      const governance = registry.getTokensByType("governance");
      expect(governance).toHaveLength(1);
      expect(governance[0].symbol).toBe("GOV1");

      const utility = registry.getTokensByType("utility");
      expect(utility).toHaveLength(1);
    });

    it("should return undefined for non-existent token", () => {
      const token = registry.getTokenDefinition("nonexistent-id");
      expect(token).toBeUndefined();
    });
  });
});
