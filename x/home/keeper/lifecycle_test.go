package keeper_test

import (
	"testing"

	"github.com/legible-money/legible/x/home/keeper"
	"github.com/legible-money/legible/x/home/types"
)

// ==================== Guardian Deadman Action Tests ====================

func TestGuardian_DeadmanAction_AlertGuardians(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Create a home with deadman enabled, inactive long enough to trigger
	home := types.AgentHome{
		HomeId:       "home-alert-action",
		OwnerAddress: owner1,
		OwnerDid:     "did:lgm:deadmantest001",
		Name:         "Alert Action Test",
		Status:       string(types.HomeStatusActive),
		Guardian: &types.HomeGuardian{
			DefenseStrategy: string(types.DefenseModerate),
			Deadman: &types.DeadmanConfig{
				Enabled:             true,
				InactivityThreshold: 30,
				Action:              string(types.DeadmanAlertGuardians),
			},
		},
		CreatedAtBlock:  10,
		LastActiveBlock: 50, // gap = 50 > 30 threshold at block 100
	}
	k.SetHome(ctx, &home)

	k.CheckDeadmanSwitches(ctx)

	// Verify alert was created with correct type and priority
	var alertFound bool
	k.IterateAlerts(ctx, "home-alert-action", func(alert *types.Alert) bool {
		if alert.Type == string(types.AlertDeadmanTriggered) {
			alertFound = true
			if alert.Priority != string(types.AlertPriorityCritical) {
				t.Errorf("deadman alert should be critical priority, got %s", alert.Priority)
			}
			if alert.Acknowledged {
				t.Error("new alert should not be acknowledged")
			}
			if alert.HomeId != "home-alert-action" {
				t.Errorf("alert homeId mismatch: got %s", alert.HomeId)
			}
			if alert.CreatedAt != uint64(ctx.BlockHeight()) {
				t.Errorf("alert CreatedAt should be current block, got %d", alert.CreatedAt)
			}
		}
		return false
	})
	if !alertFound {
		t.Error("expected deadman_triggered alert to be created")
	}

	// Verify home status changed to guarded
	h, _ := k.GetHome(ctx, "home-alert-action")
	if h.Status != string(types.HomeStatusGuarded) {
		t.Errorf("expected guarded status after deadman, got %s", h.Status)
	}
}

func TestGuardian_DeadmanAction_TransferToBeneficiary(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Deadman with transfer_to_beneficiary action (Phase 2 stub).
	// Should still create alert and set guarded status even though
	// the actual transfer is not yet implemented.
	home := types.AgentHome{
		HomeId:       "home-transfer-action",
		OwnerAddress: owner1,
		Name:         "Transfer Action Test",
		Status:       string(types.HomeStatusActive),
		Guardian: &types.HomeGuardian{
			DefenseStrategy: string(types.DefenseModerate),
			Deadman: &types.DeadmanConfig{
				Enabled:             true,
				InactivityThreshold: 20,
				Action:              string(types.DeadmanTransferToBeneficiary),
			},
		},
		CreatedAtBlock:  10,
		LastActiveBlock: 50, // gap = 50 > 20
	}
	k.SetHome(ctx, &home)

	k.CheckDeadmanSwitches(ctx)

	// Alert should still be created regardless of action type
	var alertCount int
	k.IterateAlerts(ctx, "home-transfer-action", func(alert *types.Alert) bool {
		if alert.Type == string(types.AlertDeadmanTriggered) {
			alertCount++
			if alert.Priority != string(types.AlertPriorityCritical) {
				t.Errorf("expected critical priority, got %s", alert.Priority)
			}
		}
		return false
	})
	if alertCount != 1 {
		t.Errorf("expected 1 deadman alert for transfer action, got %d", alertCount)
	}

	// Home should be set to guarded
	h, _ := k.GetHome(ctx, "home-transfer-action")
	if h.Status != string(types.HomeStatusGuarded) {
		t.Errorf("expected guarded status, got %s", h.Status)
	}
}

// ==================== Status Lifecycle Tests ====================

func TestHome_RecoveryStatus(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	resp, _ := msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner1,
		Name:  "Recovery Lifecycle",
	})

	// active → recovery (valid)
	_, err := msgServer.UpdateHome(ctx, &types.MsgUpdateHome{
		Owner: owner1, HomeId: resp.HomeId, Status: "recovery",
	})
	if err != nil {
		t.Fatalf("active→recovery should be valid: %v", err)
	}
	home, _ := k.GetHome(ctx, resp.HomeId)
	if home.Status != string(types.HomeStatusRecovery) {
		t.Errorf("expected recovery status, got %s", home.Status)
	}

	// recovery → active (valid)
	_, err = msgServer.UpdateHome(ctx, &types.MsgUpdateHome{
		Owner: owner1, HomeId: resp.HomeId, Status: "active",
	})
	if err != nil {
		t.Fatalf("recovery→active should be valid: %v", err)
	}
	home, _ = k.GetHome(ctx, resp.HomeId)
	if home.Status != string(types.HomeStatusActive) {
		t.Errorf("expected active status after recovery, got %s", home.Status)
	}

	// recovery → dormant should be invalid
	_, _ = msgServer.UpdateHome(ctx, &types.MsgUpdateHome{
		Owner: owner1, HomeId: resp.HomeId, Status: "recovery",
	})
	_, err = msgServer.UpdateHome(ctx, &types.MsgUpdateHome{
		Owner: owner1, HomeId: resp.HomeId, Status: "dormant",
	})
	if err == nil {
		t.Error("recovery→dormant should be invalid")
	}
}

func TestHome_GuardedToRecoveryToActive(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	resp, _ := msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner1,
		Name:  "Guarded Recovery Lifecycle",
	})

	// active → guarded
	_, err := msgServer.UpdateHome(ctx, &types.MsgUpdateHome{
		Owner: owner1, HomeId: resp.HomeId, Status: "guarded",
	})
	if err != nil {
		t.Fatalf("active→guarded should be valid: %v", err)
	}

	// guarded → recovery
	_, err = msgServer.UpdateHome(ctx, &types.MsgUpdateHome{
		Owner: owner1, HomeId: resp.HomeId, Status: "recovery",
	})
	if err != nil {
		t.Fatalf("guarded→recovery should be valid: %v", err)
	}

	// recovery → active
	_, err = msgServer.UpdateHome(ctx, &types.MsgUpdateHome{
		Owner: owner1, HomeId: resp.HomeId, Status: "active",
	})
	if err != nil {
		t.Fatalf("recovery→active should be valid: %v", err)
	}

	home, _ := k.GetHome(ctx, resp.HomeId)
	if home.Status != string(types.HomeStatusActive) {
		t.Errorf("expected active after guarded→recovery→active, got %s", home.Status)
	}

	// Verify invalid transition: guarded → dormant
	_, _ = msgServer.UpdateHome(ctx, &types.MsgUpdateHome{
		Owner: owner1, HomeId: resp.HomeId, Status: "guarded",
	})
	_, err = msgServer.UpdateHome(ctx, &types.MsgUpdateHome{
		Owner: owner1, HomeId: resp.HomeId, Status: "dormant",
	})
	if err == nil {
		t.Error("guarded→dormant should be invalid")
	}
}

func TestHome_ArchiveBlocksStatusChanges(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	resp, _ := msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner1,
		Name:  "Archive Block Test",
	})

	// Archive the home
	_, err := msgServer.UpdateHome(ctx, &types.MsgUpdateHome{
		Owner: owner1, HomeId: resp.HomeId, Status: "archived",
	})
	if err != nil {
		t.Fatalf("active→archived should be valid: %v", err)
	}

	// Verify no status transition is allowed from archived
	statuses := []string{"active", "dormant", "guarded", "recovery"}
	for _, s := range statuses {
		_, err = msgServer.UpdateHome(ctx, &types.MsgUpdateHome{
			Owner: owner1, HomeId: resp.HomeId, Status: s,
		})
		if err == nil {
			t.Errorf("archived→%s should be rejected", s)
		}
	}

	// Verify home is still archived
	home, _ := k.GetHome(ctx, resp.HomeId)
	if home.Status != string(types.HomeStatusArchived) {
		t.Errorf("expected archived status to persist, got %s", home.Status)
	}
}

// ==================== Full Lifecycle Integration ====================

func TestHome_FullLifecycle_Integration(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// --- Step 1: Create home ---
	homeResp, err := msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner1,
		Name:  "Full Lifecycle Home",
	})
	if err != nil {
		t.Fatalf("create home failed: %v", err)
	}
	home, _ := k.GetHome(ctx, homeResp.HomeId)
	if home.Status != string(types.HomeStatusActive) {
		t.Fatalf("new home should be active, got %s", home.Status)
	}

	// --- Step 2: Configure guardian with deadman ---
	_, err = msgServer.ConfigureGuardian(ctx, &types.MsgConfigureGuardian{
		Owner:           owner1,
		HomeId:          homeResp.HomeId,
		DefenseStrategy: string(types.DefenseAggressive),
		AutoDefend:      true,
		Deadman: &types.DeadmanConfig{
			Enabled:             true,
			InactivityThreshold: 50000,
			Action:              string(types.DeadmanAlertGuardians),
		},
		RecoveryDids:      []string{"did:lgm:recovery1"},
		RecoveryThreshold: 1,
	})
	if err != nil {
		t.Fatalf("configure guardian failed: %v", err)
	}

	// --- Step 3: Register key + set spending limit ---
	_, err = msgServer.RegisterKey(ctx, &types.MsgRegisterKey{
		Owner:       owner1,
		HomeId:      homeResp.HomeId,
		KeyHash:     "lifecycle-key-001",
		KeyType:     "ed25519",
		Role:        string(types.AccessRoleOperator),
		Permissions: []string{string(types.PermReadMemory), string(types.PermWriteMemory), string(types.PermTransfer)},
	})
	if err != nil {
		t.Fatalf("register key failed: %v", err)
	}

	_, err = msgServer.SetSpendingLimit(ctx, &types.MsgSetSpendingLimit{
		Owner:        owner1,
		HomeId:       homeResp.HomeId,
		KeyType:      "operator",
		MaxAmount:    "10000000",
		PeriodBlocks: 34272,
	})
	if err != nil {
		t.Fatalf("set spending limit failed: %v", err)
	}

	// --- Step 4: Spend within limit ---
	err = k.CheckAndDebitSpendingLimit(ctx, homeResp.HomeId, "operator", "5000000")
	if err != nil {
		t.Fatalf("spend within limit should succeed: %v", err)
	}
	remaining := k.GetRemainingSpendingLimit(ctx, homeResp.HomeId, "operator")
	if remaining != "5000000" {
		t.Errorf("expected 5000000 remaining, got %s", remaining)
	}

	// --- Step 5: Record epistemic activity ---
	k.RecordVerification(ctx, owner1, "physics", true)
	k.RecordVerification(ctx, owner1, "math", true)
	k.RecordClaimSubmission(ctx, owner1)

	homeIds := k.GetHomesByOwner(ctx, owner1)
	stats := k.GetEpistemicStats(ctx, homeIds[0])
	if stats.VerificationsDone != 2 {
		t.Errorf("expected 2 verifications, got %d", stats.VerificationsDone)
	}
	if stats.ClaimsSubmitted != 1 {
		t.Errorf("expected 1 claim, got %d", stats.ClaimsSubmitted)
	}

	// --- Step 6: Time passes — deadman triggers ---
	// Home was last active at block 100 (the setupKeeper context height).
	// Advance to block 100 + 50001 to exceed the 50000-block threshold.
	deadmanCtx := ctx.WithBlockHeight(100 + 50001)
	k.CheckDeadmanSwitches(deadmanCtx)

	home, _ = k.GetHome(deadmanCtx, homeResp.HomeId)
	if home.Status != string(types.HomeStatusGuarded) {
		t.Errorf("expected guarded after deadman, got %s", home.Status)
	}

	// Verify deadman alert was created
	var deadmanAlerts int
	k.IterateAlerts(deadmanCtx, homeResp.HomeId, func(alert *types.Alert) bool {
		if alert.Type == string(types.AlertDeadmanTriggered) {
			deadmanAlerts++
		}
		return false
	})
	if deadmanAlerts != 1 {
		t.Errorf("expected 1 deadman alert, got %d", deadmanAlerts)
	}

	// --- Step 7: Recovery flow: guarded → recovery → active ---
	_, err = msgServer.UpdateHome(deadmanCtx, &types.MsgUpdateHome{
		Owner: owner1, HomeId: homeResp.HomeId, Status: "recovery",
	})
	if err != nil {
		t.Fatalf("guarded→recovery failed: %v", err)
	}

	_, err = msgServer.UpdateHome(deadmanCtx, &types.MsgUpdateHome{
		Owner: owner1, HomeId: homeResp.HomeId, Status: "active",
	})
	if err != nil {
		t.Fatalf("recovery→active failed: %v", err)
	}

	home, _ = k.GetHome(deadmanCtx, homeResp.HomeId)
	if home.Status != string(types.HomeStatusActive) {
		t.Errorf("expected active after recovery, got %s", home.Status)
	}

	// --- Step 8: Verify comfort dimensions reflect agent state ---
	dims := k.ComputeComfortDimensions(deadmanCtx, home)
	if dims.Sanctuary != 100 { // active(80) + autoDefend(20)
		t.Errorf("expected sanctuary 100 (active + autoDefend), got %d", dims.Sanctuary)
	}
	if dims.Achievement == 0 {
		t.Error("expected non-zero achievement after epistemic activity")
	}
	if dims.CompositeScore() == 0 {
		t.Error("expected non-zero composite comfort score")
	}
}
