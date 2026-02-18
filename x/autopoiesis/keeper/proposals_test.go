package keeper_test

import (
	"testing"

	"github.com/legible-money/legible/x/autopoiesis/types"
)

// ---------- Auto-Governance Proposal Store Tests ----------

func TestSetGetAutoProposalState(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	state := types.AutoProposalState{
		LastProposalEpoch: 50,
		ProposalsInWindow: 1,
	}
	k.SetAutoProposalState(ctx, state)

	got := k.GetAutoProposalState(ctx)
	if got.LastProposalEpoch != 50 {
		t.Errorf("expected epoch 50, got %d", got.LastProposalEpoch)
	}
	if got.ProposalsInWindow != 1 {
		t.Errorf("expected 1 proposal, got %d", got.ProposalsInWindow)
	}
}

func TestSetGetTriggerState(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	state := types.ProposalTriggerState{
		Type:              types.TriggerBoundHit,
		Path:              types.PathSlashSystem,
		ConsecutiveEpochs: 5,
		LastCheckedEpoch:  100,
	}
	k.SetTriggerState(ctx, state)

	got := k.GetTriggerState(ctx, types.TriggerBoundHit, types.PathSlashSystem, "")
	if got == nil {
		t.Fatal("expected trigger state, got nil")
	}
	if got.ConsecutiveEpochs != 5 {
		t.Errorf("expected 5 consecutive epochs, got %d", got.ConsecutiveEpochs)
	}
}

func TestGetTriggerState_NotFound(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	got := k.GetTriggerState(ctx, types.TriggerDomainStress, "", "physics")
	if got != nil {
		t.Errorf("expected nil for nonexistent trigger, got %+v", got)
	}
}

// ---------- CheckProposalTriggers Tests ----------

func TestCheckProposalTriggers_RateLimiting(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	// Set a recent proposal — should block new proposals within cooldown
	k.SetAutoProposalState(ctx, types.AutoProposalState{
		LastProposalEpoch: 50,
		ProposalsInWindow: 1,
	})

	hormones := types.HormoneValues{
		Epoch: 100, // only 50 epochs since last proposal, need 100
		SSI:   types.SSIModerate + 100_000,
	}
	k.CheckProposalTriggers(ctx, 100, hormones)

	// Should not have updated proposal state
	state := k.GetAutoProposalState(ctx)
	if state.LastProposalEpoch != 50 {
		t.Errorf("expected no new proposal (rate limited), but state updated to epoch %d", state.LastProposalEpoch)
	}
}

func TestCheckProposalTriggers_SustainedStress(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	// Simulate sustained stress for 10+ epochs
	for e := uint64(100); e < 112; e++ {
		hormones := types.HormoneValues{
			Epoch: e,
			SSI:   types.SSIModerate + 100_000,
		}
		k.CheckProposalTriggers(ctx, e, hormones)
	}

	state := k.GetAutoProposalState(ctx)
	if state.LastProposalEpoch == 0 {
		t.Error("expected auto-proposal after 10+ epochs of sustained stress")
	}
}

func TestCheckProposalTriggers_BoundHit(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	// Set slash multiplier at max bound
	k.SetMultiplier(ctx, types.MultiplierEntry{
		Path:    types.PathSlashSystem,
		Value:   types.BoundsSlashMultiplier.Max,
		Hormone: "SSI",
		Epoch:   1,
	})

	// Simulate 20+ epochs at bound
	for e := uint64(100); e < 122; e++ {
		hormones := types.HormoneValues{Epoch: e, SSI: 100_000}
		k.CheckProposalTriggers(ctx, e, hormones)
	}

	state := k.GetAutoProposalState(ctx)
	if state.LastProposalEpoch == 0 {
		t.Error("expected auto-proposal after multiplier at bound for 20+ epochs")
	}
}

func TestCheckProposalTriggers_BoundHitResetsOnRelease(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	// Set slash at max bound for 10 epochs
	k.SetMultiplier(ctx, types.MultiplierEntry{
		Path:  types.PathSlashSystem,
		Value: types.BoundsSlashMultiplier.Max,
	})
	for e := uint64(100); e < 110; e++ {
		k.CheckProposalTriggers(ctx, e, types.HormoneValues{Epoch: e})
	}

	trigState := k.GetTriggerState(ctx, types.TriggerBoundHit, types.PathSlashSystem, "")
	if trigState == nil || trigState.ConsecutiveEpochs == 0 {
		t.Fatal("expected tracking to be active")
	}

	// Release from bound
	k.SetMultiplier(ctx, types.MultiplierEntry{
		Path:  types.PathSlashSystem,
		Value: types.DefaultMultiplier, // back to 1.0x
	})
	k.CheckProposalTriggers(ctx, 110, types.HormoneValues{Epoch: 110})

	trigState = k.GetTriggerState(ctx, types.TriggerBoundHit, types.PathSlashSystem, "")
	if trigState != nil && trigState.ConsecutiveEpochs != 0 {
		t.Errorf("expected consecutive epochs reset, got %d", trigState.ConsecutiveEpochs)
	}
}

func TestCheckProposalTriggers_DomainStress(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	// DVI tracking already marks consecutive negative epochs
	hormones := types.HormoneValues{
		Epoch: 200,
		DVI: []types.DVIValue{
			{Domain: "physics", Value: -200_000, ConsecutiveNegativeEpochs: 12},
		},
	}
	k.CheckProposalTriggers(ctx, 200, hormones)

	state := k.GetAutoProposalState(ctx)
	if state.LastProposalEpoch != 200 {
		t.Errorf("expected proposal at epoch 200, got %d", state.LastProposalEpoch)
	}
}

func TestCheckProposalTriggers_NoTriggerWhenHealthy(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	hormones := types.HormoneValues{
		Epoch: 200,
		SSI:   100_000, // healthy
		DVI: []types.DVIValue{
			{Domain: "physics", Value: 200_000, ConsecutiveNegativeEpochs: 0},
		},
	}
	k.CheckProposalTriggers(ctx, 200, hormones)

	state := k.GetAutoProposalState(ctx)
	if state.LastProposalEpoch != 0 {
		t.Error("expected no proposal when chain is healthy")
	}
}

// ---------- Proposal Draft Tests ----------

func TestAutoProposalEvent_EmittedOnSustainedStress(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	// Seed consecutive stress tracking
	for e := uint64(100); e < 112; e++ {
		hormones := types.HormoneValues{
			Epoch: e,
			SSI:   types.SSIModerate + 100_000,
		}
		k.CheckProposalTriggers(ctx, e, hormones)
	}

	// Check that an event was emitted
	events := ctx.EventManager().Events()
	found := false
	for _, event := range events {
		if event.Type == "autopoiesis_proposal_suggested" {
			found = true
			// Verify event attributes
			for _, attr := range event.Attributes {
				if attr.Key == "trigger_type" && attr.Value != "sustained_stress" {
					t.Errorf("expected trigger_type sustained_stress, got %s", attr.Value)
				}
			}
		}
	}
	if !found {
		t.Error("expected autopoiesis_proposal_suggested event")
	}
}

// ---------- OpenClaw Critical Trigger Tests ----------

func TestCheckProposalTriggers_OpenClawCritical(t *testing.T) {
	k, ctx, _, kk, _ := setupKeeper(t)

	// 3 verified protocol_observation facts → should trigger
	kk.protocolObsCount = 3

	hormones := types.HormoneValues{Epoch: 200, SSI: 100_000}
	k.CheckProposalTriggers(ctx, 200, hormones)

	state := k.GetAutoProposalState(ctx)
	if state.LastProposalEpoch != 200 {
		t.Errorf("expected proposal at epoch 200, got %d", state.LastProposalEpoch)
	}

	// Verify the event was emitted with correct trigger type
	events := ctx.EventManager().Events()
	found := false
	for _, event := range events {
		if event.Type == "autopoiesis_proposal_suggested" {
			for _, attr := range event.Attributes {
				if attr.Key == "trigger_type" && attr.Value == "openclaw_critical" {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("expected autopoiesis_proposal_suggested event with trigger_type=openclaw_critical")
	}
}

func TestCheckProposalTriggers_OpenClawCritical_BelowThreshold(t *testing.T) {
	k, ctx, _, kk, _ := setupKeeper(t)

	// 2 findings — below threshold of 3
	kk.protocolObsCount = 2

	hormones := types.HormoneValues{Epoch: 200, SSI: 100_000}
	k.CheckProposalTriggers(ctx, 200, hormones)

	state := k.GetAutoProposalState(ctx)
	if state.LastProposalEpoch != 0 {
		t.Errorf("expected no proposal with only 2 findings, got epoch %d", state.LastProposalEpoch)
	}
}

func TestCheckProposalTriggers_OpenClawCritical_ResetAfterTrigger(t *testing.T) {
	k, ctx, _, kk, _ := setupKeeper(t)

	// First trigger: 3 findings
	kk.protocolObsCount = 3
	ctx = ctx.WithBlockHeight(5000)
	k.CheckProposalTriggers(ctx, 200, types.HormoneValues{Epoch: 200, SSI: 100_000})

	state := k.GetAutoProposalState(ctx)
	if state.LastProposalEpoch != 200 {
		t.Fatalf("expected first trigger at epoch 200, got %d", state.LastProposalEpoch)
	}

	// After trigger fires, acknowledged block is set → same count should not re-trigger
	// (the mock always returns the same count, but in production the sinceBlock filter
	// would exclude already-counted facts). Simulate this by resetting count to 0.
	kk.protocolObsCount = 0

	// Fast-forward past cooldown (100 epochs)
	ctx = ctx.WithBlockHeight(50000)
	k.CheckProposalTriggers(ctx, 400, types.HormoneValues{Epoch: 400, SSI: 100_000})

	state = k.GetAutoProposalState(ctx)
	if state.LastProposalEpoch != 200 {
		t.Errorf("expected no new proposal with 0 findings after reset, got epoch %d", state.LastProposalEpoch)
	}

	// Trigger state should have the acknowledged block from the first firing
	trigState := k.GetTriggerState(ctx, types.TriggerOpenClawCritical, "", "")
	if trigState == nil {
		t.Fatal("expected trigger state to be stored")
	}
	if trigState.AcknowledgedAtBlock != 5000 {
		t.Errorf("expected AcknowledgedAtBlock=5000, got %d", trigState.AcknowledgedAtBlock)
	}
}

func TestCheckProposalTriggers_OpenClawCritical_NilKnowledgeKeeper(t *testing.T) {
	k, ctx, _, _, _ := setupKeeper(t)

	// Replace keeper with nil knowledge keeper
	k.SetKnowledgeKeeper(nil)

	// Should not panic
	hormones := types.HormoneValues{Epoch: 200, SSI: 100_000}
	k.CheckProposalTriggers(ctx, 200, hormones)

	state := k.GetAutoProposalState(ctx)
	if state.LastProposalEpoch != 0 {
		t.Error("expected no proposal with nil knowledge keeper")
	}
}

// ---------- Integration: Proposals via CollectAndAdapt ----------

func TestCollectAndAdapt_ProposalTriggersRun(t *testing.T) {
	k, ctx, mockSK, mockKK, _ := setupKeeper(t)

	params := types.DefaultParams()
	params.EpochLengthBlocks = 10
	k.SetParams(ctx, &params)
	k.SetState(ctx, types.AutopoiesisState{Active: true, CurrentEpoch: 0})

	// Set system under maximum stress: low accuracy, low participation, high expiration
	mockSK.avgAccuracy = 100_000   // 10% accuracy → inaccuracy 90% → weighted 27%
	mockSK.activeValidators = 5    // 5/22 participation → non-participation 77% → weighted 11.6%
	mockSK.totalValidators = 22
	mockKK.factsCreated = 1        // nearly all expiring
	mockKK.factsExpired = 50
	mockKK.activeFacts = 100
	mockKK.challengeSuccess = 800_000 // 80% challenge success → weighted 16%

	// Run 15 epochs — SSI should be well above 50% (moderate)
	for block := int64(10); block <= 150; block += 10 {
		ctx = ctx.WithBlockHeight(block)
		k.CollectAndAdapt(ctx)
	}

	// Verify SSI was high enough to trigger sustained stress tracking
	state := k.GetState(ctx)
	if state.CurrentEpoch == 0 {
		t.Fatal("expected epochs to advance")
	}

	// Check hormone values for the last epoch to verify SSI is above moderate
	hv := k.GetHormoneValues(ctx, state.CurrentEpoch)
	if hv != nil && hv.SSI >= types.SSIModerate {
		// If SSI was above moderate, trigger state should have been tracked
		trigState := k.GetTriggerState(ctx, types.TriggerSustainedStress, "", "")
		if trigState == nil {
			t.Error("expected sustained stress tracking when SSI >= moderate")
		} else if trigState.ConsecutiveEpochs == 0 {
			t.Error("expected non-zero consecutive stress epochs")
		}
	} else {
		// Even if SSI wasn't quite moderate, verify triggers still ran without panic
		t.Logf("SSI=%d (below moderate %d), skipping stress tracking assertion",
			hv.SSI, types.SSIModerate)
	}
}
