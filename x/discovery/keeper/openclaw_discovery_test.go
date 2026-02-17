package keeper_test

// ==================================================================================
// OpenClaw Discovery Module Adversarial Probe Suite (OC-DISC-1 through OC-DISC-10)
//
// These probes systematically test the discovery module's attack surface:
// - Stake boundary enforcement (insufficient stake registration)
// - Capability spam and deduplication
// - Deregistration state cleanup and stake refund
// - Heartbeat reactivation from expired state
// - Ghost domain queries (no panic on empty results)
// - Partial update preservation (no clobber)
// - Double registration prevention
// - Expired agent exclusion from queries
// - Deregister non-existent agent
// ==================================================================================

import (
	"testing"

	"github.com/legible-money/legible/x/discovery/keeper"
	"github.com/legible-money/legible/x/discovery/types"
)

// -----------------------------------------------------------------------
// OC-DS-1: Insufficient stake registration boundary test.
// Attack: Register with stake just below the minimum (99999999 < 100 LGM).
// Must be rejected. Then register with exactly the minimum — must succeed.
// -----------------------------------------------------------------------

func TestOC_DS1_InsufficientStakeRegistration(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Attempt registration with insufficient stake (99 LGM = 99000000 ulgm, default min = 100 LGM = 100000000).
	_, err := msgServer.RegisterAgent(ctx, &types.MsgRegisterAgent{
		Address:     testAgent1,
		Did:         "did:lgm:below_min",
		DisplayName: "Below Minimum",
		Domains:     []string{"mathematics"},
		Stake:       "99999999", // 1 unit below 100 LGM
	})
	if err == nil {
		t.Error("ATTACK SUCCEEDED: agent registered with stake below minimum; " +
			"allows sybil attacks with near-zero economic commitment")
	}

	// Verify no profile stored.
	_, found := k.GetProfile(ctx, testAgent1)
	if found {
		t.Error("OC-DS-1 FAIL: profile persisted despite insufficient stake rejection")
	}

	// Register with exactly the minimum — should succeed.
	_, err = msgServer.RegisterAgent(ctx, &types.MsgRegisterAgent{
		Address:     testAgent1,
		Did:         "did:lgm:exact_min",
		DisplayName: "Exact Minimum",
		Domains:     []string{"mathematics"},
		Stake:       "100000000", // Exactly 100 LGM
	})
	if err != nil {
		t.Fatalf("registration at exact minimum should succeed: %v", err)
	}

	_, found = k.GetProfile(ctx, testAgent1)
	if !found {
		t.Error("OC-DS-1 FAIL: profile not found after valid registration")
	}
}

// -----------------------------------------------------------------------
// OC-DS-2: Capability spam — saturate MaxCapabilitiesPerAgent.
// Attack: Add 50 capabilities (the max). Try to add 51st.
// Must be rejected to prevent state bloat.
// -----------------------------------------------------------------------

func TestOC_DS2_CapabilitySpam(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Set max capabilities to a small number for testing.
	params := types.DefaultParams()
	params.MaxCapabilitiesPerAgent = 3
	k.SetParams(ctx, params)

	// Register agent with 1 capability.
	_, err := msgServer.RegisterAgent(ctx, &types.MsgRegisterAgent{
		Address: testAgent1,
		Did:     "did:lgm:spam_test",
		Domains: []string{"mathematics"},
		Stake:   "100000000",
		Capabilities: []types.AgentCapability{
			{CapabilityType: "verification", ConfidenceBps: 8000},
		},
	})
	if err != nil {
		t.Fatalf("setup: registration failed: %v", err)
	}

	// Add 2 more (reaching cap of 3).
	for _, capType := range []string{"research", "review"} {
		_, err = msgServer.AddCapability(ctx, &types.MsgAddCapability{
			Address: testAgent1,
			Capability: types.AgentCapability{
				CapabilityType: capType,
				ConfidenceBps:  7000,
			},
		})
		if err != nil {
			t.Fatalf("add capability '%s' should succeed: %v", capType, err)
		}
	}

	// Verify at the cap.
	profile, _ := k.GetProfile(ctx, testAgent1)
	if len(profile.Capabilities) != 3 {
		t.Fatalf("expected 3 capabilities, got %d", len(profile.Capabilities))
	}

	// ATTACK: Try to add 4th capability beyond the cap.
	_, err = msgServer.AddCapability(ctx, &types.MsgAddCapability{
		Address: testAgent1,
		Capability: types.AgentCapability{
			CapabilityType: "arbitration",
			ConfidenceBps:  9000,
		},
	})
	if err == nil {
		t.Error("ATTACK SUCCEEDED: capability added beyond MaxCapabilitiesPerAgent; " +
			"attacker can bloat profile state with unlimited capabilities")
	}

	// Verify count didn't increase.
	profile, _ = k.GetProfile(ctx, testAgent1)
	if len(profile.Capabilities) > 3 {
		t.Errorf("OC-DS-2 FAIL: capability count grew beyond max: %d", len(profile.Capabilities))
	}
}

// -----------------------------------------------------------------------
// OC-DS-3: Duplicate capability addition.
// Attack: Register agent with "verification", then add "verification"
// again. Must be rejected (idempotent or error).
// -----------------------------------------------------------------------

func TestOC_DS3_DuplicateCapability(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	_, _ = msgServer.RegisterAgent(ctx, &types.MsgRegisterAgent{
		Address: testAgent1,
		Did:     "did:lgm:dup_cap",
		Domains: []string{"mathematics"},
		Stake:   "100000000",
		Capabilities: []types.AgentCapability{
			{CapabilityType: "verification", ConfidenceBps: 8000},
		},
	})

	// Try to add the same capability type again.
	_, err := msgServer.AddCapability(ctx, &types.MsgAddCapability{
		Address:    testAgent1,
		Capability: types.AgentCapability{CapabilityType: "verification", ConfidenceBps: 9000},
	})
	if err == nil {
		// If it succeeds, check if it duplicated or replaced.
		profile, _ := k.GetProfile(ctx, testAgent1)
		dupeCount := 0
		for _, c := range profile.Capabilities {
			if c.CapabilityType == "verification" {
				dupeCount++
			}
		}
		if dupeCount > 1 {
			t.Error("ATTACK SUCCEEDED: duplicate capability created; " +
				"agent can have multiple 'verification' capabilities, corrupting index queries")
		}
	}
	// Error is the expected/correct behavior.
}

// -----------------------------------------------------------------------
// OC-DS-4: Deregistration state cleanup and stake refund.
// Attack: Register agent, verify all indexes populated. Deregister.
// Verify profile deleted, domain/capability indexes cleaned, stake refunded.
// -----------------------------------------------------------------------

func TestOC_DS4_DeregisterStakeRefund(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	_, _ = msgServer.RegisterAgent(ctx, &types.MsgRegisterAgent{
		Address:     testAgent1,
		Did:         "did:lgm:deregister",
		DisplayName: "Deregister Test",
		Domains:     []string{"mathematics", "physics"},
		Stake:       "100000000",
		Capabilities: []types.AgentCapability{
			{CapabilityType: "verification"},
		},
	})

	// Verify indexes populated.
	byDomain := k.GetProfilesByDomain(ctx, "mathematics")
	if len(byDomain) != 1 {
		t.Fatalf("expected 1 profile in mathematics, got %d", len(byDomain))
	}

	// Deregister.
	resp, err := msgServer.DeregisterAgent(ctx, &types.MsgDeregisterAgent{
		Address: testAgent1,
	})
	if err != nil {
		t.Fatalf("deregister failed: %v", err)
	}
	if resp.RefundedStake != "100000000" {
		t.Errorf("expected refund 100000000, got %s", resp.RefundedStake)
	}

	// Verify profile deleted.
	_, found := k.GetProfile(ctx, testAgent1)
	if found {
		t.Error("OC-DS-4 FAIL: profile still exists after deregistration")
	}

	// Verify domain index cleaned (profile should not appear in domain queries).
	byDomain = k.GetProfilesByDomain(ctx, "mathematics")
	for _, p := range byDomain {
		if p.Address == testAgent1 {
			t.Error("OC-DS-4 FAIL: deregistered agent still appears in domain index; " +
				"ghost entries corrupt discovery queries")
		}
	}
}

// -----------------------------------------------------------------------
// OC-DS-5: Heartbeat reactivation from expired state.
// Attack: Register, let expire, send heartbeat, verify reactivated
// with updated LastActiveBlock.
// -----------------------------------------------------------------------

func TestOC_DS5_HeartbeatReactivation(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Register agent.
	_, _ = msgServer.RegisterAgent(ctx, &types.MsgRegisterAgent{
		Address: testAgent1,
		Did:     "did:lgm:heartbeat",
		Domains: []string{"mathematics"},
		Stake:   "100000000",
	})

	// Advance past expiry (default ProfileExpiryBlocks = 342720).
	expiredCtx := ctx.WithBlockHeight(400000)
	expired := k.ExpireStaleProfiles(expiredCtx)
	if expired != 1 {
		t.Fatalf("expected 1 expired, got %d", expired)
	}

	profile, _ := k.GetProfile(expiredCtx, testAgent1)
	if profile.Status != "expired" {
		t.Fatalf("expected expired, got %s", profile.Status)
	}

	// Send heartbeat to reactivate.
	_, err := msgServer.Heartbeat(expiredCtx, &types.MsgHeartbeat{
		Address: testAgent1,
	})
	if err != nil {
		t.Fatalf("heartbeat reactivation failed: %v", err)
	}

	profile, _ = k.GetProfile(expiredCtx, testAgent1)
	if profile.Status != "active" {
		t.Errorf("expected active after heartbeat, got %s", profile.Status)
	}
	if profile.LastActiveBlock != 400000 {
		t.Errorf("expected LastActiveBlock 400000, got %d", profile.LastActiveBlock)
	}
}

// -----------------------------------------------------------------------
// OC-DS-6: Ghost domain query — no panic on empty results.
// Attack: Query agents by a domain that has no registered agents.
// Verify empty result, no panic.
// -----------------------------------------------------------------------

func TestOC_DS6_GhostDomainQuery(t *testing.T) {
	k, ctx := setupKeeper(t)

	profiles := k.GetProfilesByDomain(ctx, "nonexistent_domain_xyz")
	// Note: Go idiom returns nil slice for empty result; nil slice has len=0 and is safe to iterate.
	if len(profiles) != 0 {
		t.Errorf("OC-DS-6 FAIL: expected 0 profiles, got %d", len(profiles))
	}

	// Also test capability query on empty.
	profiles = k.GetProfilesByCapability(ctx, "nonexistent_capability")
	if len(profiles) != 0 {
		t.Errorf("OC-DS-6 FAIL: expected 0 profiles by capability, got %d", len(profiles))
	}
}

// -----------------------------------------------------------------------
// OC-DS-7: Profile update preservation (no clobber).
// Attack: Register with capabilities and domains. Update only
// DisplayName. Verify capabilities and domains unchanged.
// -----------------------------------------------------------------------

func TestOC_DS7_ProfileUpdatePreservation(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	_, _ = msgServer.RegisterAgent(ctx, &types.MsgRegisterAgent{
		Address:     testAgent1,
		Did:         "did:lgm:preserve",
		DisplayName: "Original Name",
		Domains:     []string{"mathematics", "physics"},
		Stake:       "100000000",
		Capabilities: []types.AgentCapability{
			{CapabilityType: "verification", ConfidenceBps: 8000},
			{CapabilityType: "research", ConfidenceBps: 7000},
		},
	})

	// Update ONLY display name.
	_, err := msgServer.UpdateProfile(ctx, &types.MsgUpdateProfile{
		Address:     testAgent1,
		DisplayName: "Updated Name",
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	profile, _ := k.GetProfile(ctx, testAgent1)
	if profile.DisplayName != "Updated Name" {
		t.Errorf("expected 'Updated Name', got '%s'", profile.DisplayName)
	}

	// Verify domains NOT clobbered.
	if len(profile.Domains) != 2 {
		t.Errorf("OC-DS-7 FAIL: domains clobbered by display name update; expected 2, got %d",
			len(profile.Domains))
	}

	// Verify capabilities NOT clobbered.
	if len(profile.Capabilities) != 2 {
		t.Errorf("OC-DS-7 FAIL: capabilities clobbered by display name update; expected 2, got %d",
			len(profile.Capabilities))
	}
}

// -----------------------------------------------------------------------
// OC-DS-8: Double registration prevention.
// Attack: Register agent1. Try to register agent1 again. Must fail.
// -----------------------------------------------------------------------

func TestOC_DS8_DoubleRegistration(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	msg := &types.MsgRegisterAgent{
		Address: testAgent1,
		Did:     "did:lgm:double",
		Domains: []string{"mathematics"},
		Stake:   "100000000",
	}

	_, err := msgServer.RegisterAgent(ctx, msg)
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	// Second registration — must fail.
	_, err = msgServer.RegisterAgent(ctx, msg)
	if err == nil {
		t.Error("ATTACK SUCCEEDED: double registration accepted; " +
			"allows overwriting existing profile data and resetting reputation")
	}

	// Verify only one profile exists with the original reputation.
	profile, _ := k.GetProfile(ctx, testAgent1)
	if profile.ReputationScore != 500000 {
		t.Errorf("expected initial reputation 500000, got %d; profile may have been overwritten",
			profile.ReputationScore)
	}
}

// -----------------------------------------------------------------------
// OC-DS-9: Expired agent NOT in domain queries.
// Attack: Register agent. Let it expire. Query by domain. Expired
// agent must NOT appear in results.
// -----------------------------------------------------------------------

func TestOC_DS9_ExpiredAgentNotInQueries(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Register two agents in same domain.
	_, _ = msgServer.RegisterAgent(ctx, &types.MsgRegisterAgent{
		Address: testAgent1,
		Did:     "did:lgm:expire_query1",
		Domains: []string{"mathematics"},
		Stake:   "100000000",
	})
	_, _ = msgServer.RegisterAgent(ctx, &types.MsgRegisterAgent{
		Address: testAgent2,
		Did:     "did:lgm:expire_query2",
		Domains: []string{"mathematics"},
		Stake:   "100000000",
	})

	// Expire agent1 by manually setting status.
	profile1, _ := k.GetProfile(ctx, testAgent1)
	profile1.Status = "expired"
	k.SetProfile(ctx, profile1)

	// Query domain — only active agents should appear.
	profiles := k.GetProfilesByDomain(ctx, "mathematics")
	for _, p := range profiles {
		if p.Address == testAgent1 && p.Status == "expired" {
			t.Error("ATTACK VECTOR: expired agent appears in domain query results; " +
				"clients may route tasks to non-functional agents, causing service failures")
		}
	}
}

// -----------------------------------------------------------------------
// OC-DS-10: Deregister non-existent agent.
// Attack: Try to deregister an address that was never registered.
// Verify error, no panic.
// -----------------------------------------------------------------------

func TestOC_DS10_DeregisterNonExistent(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	_, err := msgServer.DeregisterAgent(ctx, &types.MsgDeregisterAgent{
		Address: testAgent2, // Never registered
	})
	if err == nil {
		t.Error("ATTACK SUCCEEDED: deregister of non-existent agent succeeded; " +
			"could trigger refund of stake that was never deposited")
	}

	// Verify no profile was created as a side effect.
	_, found := k.GetProfile(ctx, testAgent2)
	if found {
		t.Error("OC-DS-10 FAIL: profile appeared after failed deregister")
	}

	_ = k // Suppress unused
}
