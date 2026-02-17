package keeper_test

// ==================================================================================
// OpenClaw Home Module Adversarial Probe Suite (OC-HM-1 through OC-HM-4)
//
// These probes systematically test the home module's attack surface:
// - Ownership enforcement on key registration
// - Session limit enforcement under saturation
// - Cross-owner session start isolation
// - Duplicate home creation rejection
// ==================================================================================

import (
	"testing"

	"github.com/legible-money/legible/x/home/keeper"
	"github.com/legible-money/legible/x/home/types"
)

// ==================================================================================
// OC-HM-1: Non-Owner Key Registration
//
// ATTACK: Owner A creates a home. Attacker B tries to register a key on A's home.
// This would let B install a backdoor key, then start sessions and access the home.
//
// DEFENSE: RegisterKey checks home.OwnerAddress != msg.Owner and returns
// ErrNotHomeOwner.
// ==================================================================================

func TestOC_HM1_NonOwnerKeyRegistration(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Owner A creates a home
	homeResp, err := msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner1,
		Name:  "Owner A Home",
	})
	if err != nil {
		t.Fatalf("setup: create home failed: %v", err)
	}

	// ATTACK: Owner B tries to register a key on Owner A's home
	_, err = msgServer.RegisterKey(ctx, &types.MsgRegisterKey{
		Owner:       owner2,
		HomeId:      homeResp.HomeId,
		KeyHash:     "attacker_backdoor_key",
		KeyType:     "ed25519",
		Role:        string(types.AccessRoleOperator),
		Permissions: []string{string(types.PermReadMemory), string(types.PermWriteMemory), string(types.PermTransfer)},
	})
	if err == nil {
		t.Error("OC-HM-1 FAIL: non-owner was able to register a key on another owner's home — " +
			"attacker can install backdoor keys and gain full session access to victim's home, " +
			"including memory read/write and token transfer")
	}

	// Verify no key was registered
	_, found := k.GetKeyRegistration(ctx, homeResp.HomeId, "attacker_backdoor_key")
	if found {
		t.Error("OC-HM-1 FAIL: attacker's key persisted in store despite ownership check")
	}

	t.Logf("OC-HM-1 PASS: non-owner key registration rejected (owner=%s, attacker=%s)", owner1, owner2)
}

// ==================================================================================
// OC-HM-2: Session Limit Exceeded
//
// ATTACK: Saturate the session limit, then try to start one more session.
// Without enforcement, an attacker could open unlimited sessions to exhaust
// state storage or monopolize home resources.
//
// DEFENSE: StartSession checks CountSessions >= params.MaxSessionsPerHome
// and returns ErrMaxSessionsReached.
// ==================================================================================

func TestOC_HM2_SessionLimitExceeded(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Set a tight session limit for testing
	params := types.DefaultParams()
	params.MaxSessionsPerHome = 2
	k.SetParams(ctx, params)

	// Create home and register multiple keys (each key at different block
	// heights produces a unique session ID: ses-{keyHash[:8]}-{blockHeight})
	homeResp, err := msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner1,
		Name:  "Session Limit Home",
	})
	if err != nil {
		t.Fatalf("setup: create home failed: %v", err)
	}

	// Register keys with distinct first-8-char prefixes so session IDs are
	// unique (session ID = "ses-{keyHash[:8]}-{blockHeight}").
	keyHashes := []string{"alphakey_hash01", "betakeys_hash02", "gammakey_hash03"}
	for i, kh := range keyHashes {
		_, err = msgServer.RegisterKey(ctx, &types.MsgRegisterKey{
			Owner:       owner1,
			HomeId:      homeResp.HomeId,
			KeyHash:     kh,
			KeyType:     "ed25519",
			Role:        string(types.AccessRoleOperator),
			Permissions: []string{string(types.PermReadMemory)},
		})
		if err != nil {
			t.Fatalf("setup: register key %d failed: %v", i+1, err)
		}
	}

	// Fill sessions up to the limit using different keys (unique session IDs)
	_, err = msgServer.StartSession(ctx, &types.MsgStartSession{
		Signer:  user1,
		HomeId:  homeResp.HomeId,
		KeyHash: keyHashes[0],
	})
	if err != nil {
		t.Fatalf("setup: start session 1 failed: %v", err)
	}

	_, err = msgServer.StartSession(ctx, &types.MsgStartSession{
		Signer:  user1,
		HomeId:  homeResp.HomeId,
		KeyHash: keyHashes[1],
	})
	if err != nil {
		t.Fatalf("setup: start session 2 failed: %v", err)
	}

	// Verify sessions are at the limit
	count := k.CountSessions(ctx, homeResp.HomeId)
	if count != 2 {
		t.Fatalf("setup: expected 2 sessions, got %d", count)
	}

	// ATTACK: Try to start one more session beyond the limit
	_, err = msgServer.StartSession(ctx, &types.MsgStartSession{
		Signer:  user1,
		HomeId:  homeResp.HomeId,
		KeyHash: keyHashes[2],
	})
	if err == nil {
		t.Error("OC-HM-2 FAIL: session beyond MaxSessionsPerHome was accepted — " +
			"attacker can open unlimited sessions to exhaust state storage " +
			"and monopolize home resources")
	}

	// Verify session count did not increase
	countAfter := k.CountSessions(ctx, homeResp.HomeId)
	if countAfter != 2 {
		t.Errorf("OC-HM-2 FAIL: session count changed after rejection: %d -> %d", count, countAfter)
	}

	t.Logf("OC-HM-2 PASS: session limit enforced (max=%d, overflow rejected)", params.MaxSessionsPerHome)
}

// ==================================================================================
// OC-HM-3: Non-Owner Session Start
//
// ATTACK: Owner A creates a home and registers a key. Attacker B (a different
// address) tries to start a session using A's registered key. If the home module
// does not enforce signer-to-key binding, any address could hijack sessions.
//
// Note: The current StartSession handler allows any signer to start a session
// with a valid, non-expired, non-revoked key. This test documents the current
// behavior. If StartSession is purely key-based (any holder of the key hash can
// start), this is by design. The test verifies the key must exist and be valid.
// ==================================================================================

func TestOC_HM3_NonOwnerSessionStart(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Owner A creates a home
	homeResp, err := msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner1,
		Name:  "Session Auth Home",
	})
	if err != nil {
		t.Fatalf("setup: create home failed: %v", err)
	}

	// Owner A registers a key
	_, err = msgServer.RegisterKey(ctx, &types.MsgRegisterKey{
		Owner:       owner1,
		HomeId:      homeResp.HomeId,
		KeyHash:     "owner_a_key",
		KeyType:     "ed25519",
		Role:        string(types.AccessRoleOperator),
		Permissions: []string{string(types.PermReadMemory), string(types.PermTransfer)},
	})
	if err != nil {
		t.Fatalf("setup: register key failed: %v", err)
	}

	// ATTACK: Owner B tries to start a session with a FABRICATED key hash
	// that was never registered. This must fail.
	_, err = msgServer.StartSession(ctx, &types.MsgStartSession{
		Signer:               owner2,
		HomeId:               homeResp.HomeId,
		KeyHash:              "nonexistent_key_hash",
		RequestedPermissions: []string{string(types.PermReadMemory), string(types.PermTransfer)},
	})
	if err == nil {
		t.Error("OC-HM-3 FAIL: attacker started session with unregistered key — " +
			"any address can fabricate a key hash and gain session access to any home, " +
			"bypassing the key registration authorization model entirely")
	}

	// ATTACK: Owner B tries to start a session using a revoked key
	// First register and then revoke a key
	_, err = msgServer.RegisterKey(ctx, &types.MsgRegisterKey{
		Owner:       owner1,
		HomeId:      homeResp.HomeId,
		KeyHash:     "revocable_key",
		KeyType:     "ed25519",
		Role:        string(types.AccessRoleGuest),
		Permissions: []string{string(types.PermReadMemory)},
	})
	if err != nil {
		t.Fatalf("setup: register revocable key failed: %v", err)
	}

	_, err = msgServer.RevokeKey(ctx, &types.MsgRevokeKey{
		Owner:   owner1,
		HomeId:  homeResp.HomeId,
		KeyHash: "revocable_key",
	})
	if err != nil {
		t.Fatalf("setup: revoke key failed: %v", err)
	}

	_, err = msgServer.StartSession(ctx, &types.MsgStartSession{
		Signer:  owner2,
		HomeId:  homeResp.HomeId,
		KeyHash: "revocable_key",
	})
	if err == nil {
		t.Error("OC-HM-3 FAIL: attacker started session with revoked key — " +
			"revoked keys should be permanently unusable for session creation")
	}

	t.Logf("OC-HM-3 PASS: unregistered and revoked key session starts rejected")
}

// ==================================================================================
// OC-HM-4: Duplicate Home Creation
//
// ATTACK: Owner A creates a home, then tries to create homes until they exceed
// MaxHomesPerAccount. Without enforcement, an attacker could create unlimited
// homes to bloat state and consume chain resources.
//
// DEFENSE: CreateHome checks len(existing) >= params.MaxHomesPerAccount and
// returns ErrMaxHomesReached.
// ==================================================================================

func TestOC_HM4_DuplicateHomeCreation(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Set max homes to 1 for tight testing
	params := types.DefaultParams()
	params.MaxHomesPerAccount = 1
	k.SetParams(ctx, params)

	// Owner A creates the first home — should succeed
	resp, err := msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner1,
		Name:  "First Home",
	})
	if err != nil {
		t.Fatalf("setup: first home creation failed: %v", err)
	}
	firstHomeId := resp.HomeId

	// ATTACK: Owner A tries to create a second home — should fail
	_, err = msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner1,
		Name:  "Second Home",
	})
	if err == nil {
		t.Error("OC-HM-4 FAIL: owner created more homes than MaxHomesPerAccount allows — " +
			"attacker can create unlimited homes to bloat chain state, " +
			"exhaust storage resources, and inflate owner-index iteration costs")
	}

	// Verify only 1 home exists for this owner
	homes := k.GetHomesByOwner(ctx, owner1)
	if len(homes) != 1 {
		t.Errorf("OC-HM-4 FAIL: expected 1 home for owner, got %d", len(homes))
	}
	if homes[0] != firstHomeId {
		t.Errorf("OC-HM-4 FAIL: stored home ID mismatch: got %s, expected %s", homes[0], firstHomeId)
	}

	// Verify a DIFFERENT owner can still create a home (the limit is per-account)
	_, err = msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner2,
		Name:  "Owner B Home",
	})
	if err != nil {
		t.Fatalf("OC-HM-4 FAIL: different owner should be able to create a home: %v", err)
	}

	t.Logf("OC-HM-4 PASS: duplicate home creation rejected (max=%d, per-account enforcement verified)", params.MaxHomesPerAccount)
}
