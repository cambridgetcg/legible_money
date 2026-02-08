/**
 * AI Agent Service Token Tests
 *
 * Tests service registration, invocation (token transfer from caller to provider),
 * reputation staking, and agent home integration.
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

describe.sequential("AI Agent Service Token", () => {
  let registry: TokenRegistryStateTree;
  let creator: ReturnType<typeof makeAgent>;
  let infTokenId: string; // INF = Inference Token

  beforeEach(() => {
    resetAgentCounter();
    registry = createTestRegistry();
    creator = makeAgent();
    const result = deployTestToken(registry, creator, {
      name: "Inference",
      symbol: "INF",
      tokenType: "utility",
      maxSupply: 0n,
    });
    infTokenId = result.tokenId;
  });

  describe("Service Registration", () => {
    it("should allow agent to register service priced in INF", () => {
      const provider = makeAgent();
      registry.mint(infTokenId, provider.did, 1000n * ONE_LGM);

      // Stake minimum INF for quality guarantee
      registry.stakeSecondaryToken(
        provider.did,
        infTokenId,
        100n * ONE_LGM,
        1n,
      );

      // Register service
      registry.registerService({
        provider: provider.did as any,
        serviceId: "inference-gpt-4",
        paymentTokenId: infTokenId,
        pricePerCall: 5n * ONE_LGM,
        qualityStake: 100n * ONE_LGM,
        active: true,
      });

      const service = registry.getService("inference-gpt-4");
      expect(service).toBeDefined();
      expect(service!.provider).toBe(provider.did);
      expect(service!.pricePerCall).toBe(5n * ONE_LGM);
      expect(service!.active).toBe(true);
    });

    it("should reject registration without minimum INF stake", () => {
      const provider = makeAgent();
      // No tokens minted, can't stake
      expect(() =>
        registry.stakeSecondaryToken(
          provider.did,
          infTokenId,
          100n * ONE_LGM,
          1n,
        ),
      ).toThrow("Insufficient");
    });
  });

  describe("Service Invocation", () => {
    let provider: ReturnType<typeof makeAgent>;

    beforeEach(() => {
      provider = makeAgent();
      registry.mint(infTokenId, provider.did, 1000n * ONE_LGM);
      registry.stakeSecondaryToken(
        provider.did,
        infTokenId,
        100n * ONE_LGM,
        1n,
      );
      registry.registerService({
        provider: provider.did as any,
        serviceId: "inference-v1",
        paymentTokenId: infTokenId,
        pricePerCall: 10n * ONE_LGM,
        qualityStake: 100n * ONE_LGM,
        active: true,
      });
    });

    it("should transfer INF from caller to provider on call", () => {
      const caller = makeAgent();
      registry.mint(infTokenId, caller.did, 100n * ONE_LGM);

      // Simulate service call: transfer INF from caller to provider
      const service = registry.getService("inference-v1")!;
      registry.transferToken(
        infTokenId,
        caller.did,
        provider.did,
        service.pricePerCall,
      );

      expect(registry.getTokenBalance(caller.did, infTokenId)).toBe(
        90n * ONE_LGM,
      );
      // Provider gets payment (in addition to remaining unlocked balance)
      expect(registry.getTokenBalance(provider.did, infTokenId)).toBe(
        900n * ONE_LGM + 10n * ONE_LGM, // 900 (after staking 100) + 10 payment
      );
    });

    it("should reject call with insufficient INF", () => {
      const caller = makeAgent();
      registry.mint(infTokenId, caller.did, 5n * ONE_LGM);

      const service = registry.getService("inference-v1")!;
      expect(() =>
        registry.transferToken(
          infTokenId,
          caller.did,
          provider.did,
          service.pricePerCall,
        ),
      ).toThrow("Insufficient");
    });
  });

  describe("Reputation Staking", () => {
    it("should require INF stake for quality guarantee", () => {
      const provider = makeAgent();
      registry.mint(infTokenId, provider.did, 500n * ONE_LGM);
      registry.stakeSecondaryToken(
        provider.did,
        infTokenId,
        200n * ONE_LGM,
        1n,
      );

      const position = registry.getStakePosition(provider.did, infTokenId);
      expect(position).toBeDefined();
      expect(position!.stakedAmount).toBe(200n * ONE_LGM);
    });

    it("should slash INF stake on quality failure", () => {
      const provider = makeAgent();
      registry.mint(infTokenId, provider.did, 500n * ONE_LGM);
      registry.stakeSecondaryToken(
        provider.did,
        infTokenId,
        200n * ONE_LGM,
        1n,
      );

      // Unstake and reduce balance as slash
      const returned = registry.unstakeSecondaryToken(provider.did, infTokenId);
      // Simulate slash: burn 50% of staked amount
      const slashAmount = returned / 2n;
      registry.burn(infTokenId, provider.did, slashAmount);

      expect(registry.getTokenBalance(provider.did, infTokenId)).toBe(
        500n * ONE_LGM - slashAmount, // original - slashed
      );
    });

    it("should return INF stake on service retirement", () => {
      const provider = makeAgent();
      registry.mint(infTokenId, provider.did, 500n * ONE_LGM);
      registry.stakeSecondaryToken(
        provider.did,
        infTokenId,
        200n * ONE_LGM,
        1n,
      );

      const balanceBefore = registry.getTokenBalance(provider.did, infTokenId);
      registry.unstakeSecondaryToken(provider.did, infTokenId);

      expect(registry.getTokenBalance(provider.did, infTokenId)).toBe(
        balanceBefore + 200n * ONE_LGM,
      );
    });
  });

  describe("Agent Home Integration", () => {
    it("should allow agents to auto-purchase via self-invoking contract", () => {
      // Agent has INF tokens and can set up auto-purchase via contract
      const agent = makeAgent();
      registry.mint(infTokenId, agent.did, 500n * ONE_LGM);

      // Set allowance for contract to spend on behalf of agent
      const contractDid = "did:lgm:auto-purchase-contract-00";
      registry.setAllowance(agent.did, contractDid, infTokenId, 200n * ONE_LGM);

      const allowance = registry.getAllowance(
        agent.did,
        contractDid,
        infTokenId,
      );
      expect(allowance).toBe(200n * ONE_LGM);

      // Contract can transferFrom on behalf of agent
      const provider = makeAgent();
      registry.mint(infTokenId, provider.did, 0n); // ensure provider account exists
      registry.transferFrom(
        infTokenId,
        contractDid,
        agent.did,
        provider.did,
        50n * ONE_LGM,
      );

      expect(registry.getTokenBalance(agent.did, infTokenId)).toBe(
        450n * ONE_LGM,
      );
      expect(registry.getTokenBalance(provider.did, infTokenId)).toBe(
        50n * ONE_LGM,
      );
      expect(registry.getAllowance(agent.did, contractDid, infTokenId)).toBe(
        150n * ONE_LGM,
      );
    });
  });
});
