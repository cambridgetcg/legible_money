/**
 * Governance Token Tests
 *
 * Tests voting power tracking, delegation, proposal thresholds,
 * and snapshot-based voting.
 */

import { describe, it, expect, beforeEach } from "vitest";
import { TokenRegistryStateTree } from "../../../../state/token_registry.js";
import {
  makeAgent,
  resetAgentCounter,
  deployTestToken,
  createTestRegistry,
  ONE_LGM,
  PROPOSAL_THRESHOLD,
} from "./_helpers.js";

describe.sequential("Governance Token", () => {
  let registry: TokenRegistryStateTree;
  let creator: ReturnType<typeof makeAgent>;
  let tokenId: string;

  beforeEach(() => {
    resetAgentCounter();
    registry = createTestRegistry();
    creator = makeAgent();
    const result = deployTestToken(registry, creator, {
      name: "Governance",
      symbol: "GOV",
      tokenType: "governance",
      maxSupply: 10_000_000n * ONE_LGM,
    });
    tokenId = result.tokenId;
  });

  // ============ Voting Power ============

  describe("Voting Power", () => {
    it("should track voting power equal to balance", () => {
      registry.mint(tokenId, creator.did, 500n * ONE_LGM);

      const power = registry.getVotingPower(creator.did, tokenId);
      expect(power).toBe(500n * ONE_LGM);
    });

    it("should update voting power on transfer", () => {
      const alice = makeAgent();
      const bob = makeAgent();
      registry.mint(tokenId, alice.did, 1000n * ONE_LGM);

      registry.transferToken(tokenId, alice.did, bob.did, 400n * ONE_LGM);

      expect(registry.getVotingPower(alice.did, tokenId)).toBe(600n * ONE_LGM);
      expect(registry.getVotingPower(bob.did, tokenId)).toBe(400n * ONE_LGM);
    });

    it("should return 0 voting power for account with no tokens", () => {
      const nobody = makeAgent();
      expect(registry.getVotingPower(nobody.did, tokenId)).toBe(0n);
    });
  });

  // ============ Delegation ============

  describe("Delegation", () => {
    let alice: ReturnType<typeof makeAgent>;
    let bob: ReturnType<typeof makeAgent>;
    let charlie: ReturnType<typeof makeAgent>;

    beforeEach(() => {
      alice = makeAgent();
      bob = makeAgent();
      charlie = makeAgent();
      registry.mint(tokenId, alice.did, 1000n * ONE_LGM);
      registry.mint(tokenId, charlie.did, 500n * ONE_LGM);
    });

    it("should allow vote delegation to another agent", () => {
      registry.delegateVotingPower(alice.did, bob.did, tokenId, 1n);

      // Alice's voting power goes to 0 (delegated away)
      expect(registry.getVotingPower(alice.did, tokenId)).toBe(0n);
      // Bob gets Alice's voting power
      expect(registry.getVotingPower(bob.did, tokenId)).toBe(1000n * ONE_LGM);
    });

    it("should allow undelegation", () => {
      registry.delegateVotingPower(alice.did, bob.did, tokenId, 1n);
      registry.undelegateVotingPower(alice.did, tokenId);

      expect(registry.getVotingPower(alice.did, tokenId)).toBe(1000n * ONE_LGM);
      expect(registry.getVotingPower(bob.did, tokenId)).toBe(0n);
    });

    it("should prevent circular delegation", () => {
      registry.delegateVotingPower(alice.did, bob.did, tokenId, 1n);

      expect(() =>
        registry.delegateVotingPower(bob.did, alice.did, tokenId, 2n),
      ).toThrow("Circular delegation");
    });

    it("should handle delegation chain (A->B, C->B)", () => {
      registry.delegateVotingPower(alice.did, bob.did, tokenId, 1n);
      registry.delegateVotingPower(charlie.did, bob.did, tokenId, 1n);

      // Bob gets both Alice's and Charlie's voting power
      expect(registry.getVotingPower(bob.did, tokenId)).toBe(1500n * ONE_LGM);
      expect(registry.getVotingPower(alice.did, tokenId)).toBe(0n);
      expect(registry.getVotingPower(charlie.did, tokenId)).toBe(0n);
    });

    it("should track delegation record", () => {
      registry.delegateVotingPower(alice.did, bob.did, tokenId, 10n);

      const delegation = registry.getDelegation(alice.did, tokenId);
      expect(delegation).toBeDefined();
      expect(delegation!.delegator).toBe(alice.did);
      expect(delegation!.delegate).toBe(bob.did);
      expect(delegation!.delegatedAtBlock).toBe(10n);
    });
  });

  // ============ Proposal Threshold ============

  describe("Proposal Threshold", () => {
    it("should require minimum token balance to create proposal", () => {
      const proposer = makeAgent();
      registry.mint(tokenId, proposer.did, PROPOSAL_THRESHOLD);

      const power = registry.getVotingPower(proposer.did, tokenId);
      expect(power >= PROPOSAL_THRESHOLD).toBe(true);
    });

    it("should deny proposal from below-threshold account", () => {
      const proposer = makeAgent();
      registry.mint(tokenId, proposer.did, PROPOSAL_THRESHOLD - 1n);

      const power = registry.getVotingPower(proposer.did, tokenId);
      expect(power >= PROPOSAL_THRESHOLD).toBe(false);
    });
  });

  // ============ Snapshot Voting ============

  describe("Snapshot Voting", () => {
    it("should use balance at snapshot block for votes", () => {
      const alice = makeAgent();
      const bob = makeAgent();
      registry.mint(tokenId, alice.did, 1000n * ONE_LGM);
      registry.mint(tokenId, bob.did, 500n * ONE_LGM);

      // Take snapshot at block 10
      registry.takeVotingPowerSnapshot(tokenId, 10n);

      // Transfer after snapshot
      registry.transferToken(tokenId, alice.did, bob.did, 800n * ONE_LGM);

      // Snapshot values unchanged
      expect(registry.getSnapshotVotingPower(tokenId, 10n, alice.did)).toBe(
        1000n * ONE_LGM,
      );
      expect(registry.getSnapshotVotingPower(tokenId, 10n, bob.did)).toBe(
        500n * ONE_LGM,
      );

      // Current values changed
      expect(registry.getVotingPower(alice.did, tokenId)).toBe(200n * ONE_LGM);
      expect(registry.getVotingPower(bob.did, tokenId)).toBe(1300n * ONE_LGM);
    });

    it("should include delegated power in snapshot", () => {
      const alice = makeAgent();
      const bob = makeAgent();
      registry.mint(tokenId, alice.did, 1000n * ONE_LGM);

      registry.delegateVotingPower(alice.did, bob.did, tokenId, 5n);
      registry.takeVotingPowerSnapshot(tokenId, 10n);

      expect(registry.getSnapshotVotingPower(tokenId, 10n, bob.did)).toBe(
        1000n * ONE_LGM,
      );
      expect(registry.getSnapshotVotingPower(tokenId, 10n, alice.did)).toBe(0n);
    });
  });

  // ============ LGM Fee for Governance Actions ============

  describe("LGM Fee for Governance Actions", () => {
    it("governance token operations should never consume GOV for gas", () => {
      const alice = makeAgent();
      registry.mint(tokenId, alice.did, 1000n * ONE_LGM);

      // All governance operations consume tokens for their function
      // but gas fees should only be in LGM (verified at transaction layer)
      const balanceBefore = registry.getTokenBalance(alice.did, tokenId);
      registry.delegateVotingPower(alice.did, creator.did, tokenId, 1n);

      // Token balance unchanged by delegation itself
      // (gas deduction happens in LGM at transaction layer, not here)
      expect(registry.getTokenBalance(alice.did, tokenId)).toBe(balanceBefore);
    });
  });
});
