package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/ontology/keeper"
	"github.com/legible-money/legible/x/ontology/types"
)

// ============ OpenClaw Adversarial Tests: Ontology Module ============
//
// These tests probe the domain proposal system, logic zone registration,
// and stratum management for authorization bypasses, economic manipulation,
// and state integrity violations.

// OC-ON-1: Zero Stake Domain Proposal — Try to propose domain with zero stake.
func TestOC_ON1_ZeroStakeDomainProposal(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Fund proposer
	proposer := testAddr("propzero")
	bk.fundAccount(proposer, largeBalance)

	// Ensure stratum 1 exists
	k.SetStratum(ctx, types.StratumProperties{Stratum: 1, Name: "Foundation", Description: "test"})

	// Try zero stake — should fail
	_, err := ms.ProposeDomain(sdk.WrapSDKContext(ctx), &types.MsgProposeDomain{
		Proposer:    proposer,
		Name:        "zero-domain",
		DisplayName: "Zero Domain",
		Description: "Testing zero stake",
		Stratum:     1,
		Stake:       "0",
	})
	if err == nil {
		t.Error("OC-ON-1: zero-stake domain proposal should fail — attacker can spam proposals for free")
	}

	// Try negative stake
	_, err = ms.ProposeDomain(sdk.WrapSDKContext(ctx), &types.MsgProposeDomain{
		Proposer:    proposer,
		Name:        "neg-domain",
		DisplayName: "Negative Domain",
		Description: "Testing negative stake",
		Stratum:     1,
		Stake:       "-100",
	})
	if err == nil {
		t.Error("OC-ON-1: negative-stake domain proposal should fail — input validation bypass")
	}
}

// OC-ON-2: Duplicate Domain Proposal — Try to create a domain that already exists.
func TestOC_ON2_DuplicateDomainProposal(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	proposer := testAddr("propdup_")
	bk.fundAccount(proposer, largeBalance)

	k.SetStratum(ctx, types.Stratum{Level: 1, Name: "Foundation", Description: "test"})

	// First proposal
	_, err := ms.ProposeDomain(sdk.WrapSDKContext(ctx), &types.MsgProposeDomain{
		Proposer:    proposer,
		Name:        "mathematics",
		DisplayName: "Mathematics",
		Description: "Math domain",
		Stratum:     1,
		Stake:       "1000000",
	})
	if err != nil {
		t.Fatalf("OC-ON-2: first proposal failed: %v", err)
	}

	// Manually accept the domain to make it exist in the domain registry
	k.SetDomain(ctx, types.Domain{
		Name:        "mathematics",
		DisplayName: "Mathematics",
		Description: "Math domain",
		Stratum:     1,
		Status:      "active",
		ProposedBy:  proposer,
	})

	// Second proposal for same domain — should fail
	proposer2 := testAddr("propdup2")
	bk.fundAccount(proposer2, largeBalance)

	_, err = ms.ProposeDomain(sdk.WrapSDKContext(ctx), &types.MsgProposeDomain{
		Proposer:    proposer2,
		Name:        "mathematics",
		DisplayName: "Mathematics 2",
		Description: "Duplicate",
		Stratum:     1,
		Stake:       "1000000",
	})
	if err == nil {
		t.Error("OC-ON-2: duplicate domain proposal should fail — attacker can overwrite existing domains")
	}
}

// OC-ON-3: Invalid Stratum Reference — Propose domain for non-existent stratum.
func TestOC_ON3_InvalidStratumReference(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	proposer := testAddr("propbad_")
	bk.fundAccount(proposer, largeBalance)

	// Try with stratum 99 (does not exist)
	_, err := ms.ProposeDomain(sdk.WrapSDKContext(ctx), &types.MsgProposeDomain{
		Proposer:    proposer,
		Name:        "phantom-domain",
		DisplayName: "Phantom",
		Description: "Non-existent stratum",
		Stratum:     99,
		Stake:       "1000000",
	})
	if err == nil {
		t.Error("OC-ON-3: domain proposal with invalid stratum should fail — attacker can create unanchored domains")
	}
}
