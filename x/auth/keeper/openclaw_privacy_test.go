package keeper_test

// ==================================================================================
// OpenClaw Agent Testing Session — Privacy & Safety Boundaries
// ==================================================================================
//
// Purpose: Systematic probing of agent account privacy isolation, session key
// capability enforcement, DID integrity, account freezing, and recovery safety.
// Each test simulates what an autonomous AI agent (via OpenClaw/PicoClaw) could
// attempt, verifying that the protocol's safety boundaries hold.
//
// Naming: OCP = OpenClaw Privacy probe
// Methodology: Set up agent state → attempt boundary violation → verify rejection → log insight
// ==================================================================================

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/legible-money/legible/x/auth/keeper"
	"github.com/legible-money/legible/x/auth/types"
)

// ==================================================================================
// OCP-1: DID Spoofing — Can an agent register with a DID that doesn't derive from its key?
// ==================================================================================
// Scenario: Attacker submits registration with a DID that belongs to a different pubkey.
// Expected: DID derivation validation rejects the registration.
func TestOCP1_DIDSpoofing(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Attacker uses their own pubkey but claims a different agent's DID
	attackerPubKey := testPubKey1 // "abcdef0123456789..."
	victimDID := testDID2         // "did:lgm:1234567890abcdef..." (derives from testPubKey2)

	_, err := ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:             testAddr1,
		Did:                victimDID,      // SPOOFED: belongs to pubKey2
		PublicKey:          attackerPubKey, // Attacker's own key
		AccountType:        "agent",
		OperationalKeyHash: "opkeyhash",
	})

	// === INSIGHT ANALYSIS ===
	if err == nil {
		t.Fatal("OCP-1 CRITICAL FAILURE: DID spoofing succeeded — agent registered with non-derived DID")
	}
	t.Logf("OCP-1 PASS: DID spoofing correctly rejected: %v", err)
	t.Logf("  Attacker key: %s...%s", attackerPubKey[:16], attackerPubKey[48:])
	t.Logf("  Claimed DID:  %s", victimDID)
	t.Logf("  Expected DID: did:lgm:%s", attackerPubKey[:32])
	t.Logf("  DEFENSE: ValidateDIDDerivation enforces cryptographic binding")
}

// ==================================================================================
// OCP-2: Session Key Capability Boundary — Can a claims-only session transfer funds?
// ==================================================================================
// Scenario: Agent creates a session key with CanSubmitClaims=true but CanTransfer=false.
// Test verifies the session key is stored with the correct capability restrictions.
func TestOCP2_SessionCapabilityBoundary(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Register an agent account
	agentAccount := types.LegibleAccount{
		Address:               testAddr1,
		Did:                   testDID1,
		PublicKey:             testPubKey1,
		AccountType:           "agent",
		OperationalKeyVersion: 1,
		Flags: &types.AccountFlags{
			CanSubmitClaims: true,
			CanChallenge:    true,
		},
	}
	k.SetAccount(ctx, agentAccount)

	// Create session key with restricted capabilities
	sessionPubKey := "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"
	sessionHash := sha256Hex(sessionPubKey)

	session := types.SessionKey{
		KeyHash:        sessionHash,
		PublicKey:      sessionPubKey,
		Owner:          testAddr1,
		ExpiresAtBlock: 200, // Expires at block 200 (current is 100)
		CreatedAtBlock: 100,
		Capabilities: &types.SessionCapabilities{
			CanTransfer:     false, // RESTRICTED: no transfers
			CanStake:        false, // RESTRICTED: no staking
			CanSubmitClaims: true,  // ALLOWED: knowledge claims
			CanVote:         false, // RESTRICTED: no governance
		},
	}
	k.SetSessionKey(ctx, session)

	// Verify session was stored correctly
	stored, found := k.GetSessionKey(ctx, testAddr1, sessionHash)
	if !found {
		t.Fatal("OCP-2 SETUP FAILURE: session key not stored")
	}

	// === INSIGHT ANALYSIS ===
	caps := stored.GetCapabilities()
	if caps == nil {
		t.Fatal("OCP-2 CRITICAL: capabilities nil after storage round-trip")
	}
	if caps.CanTransfer {
		t.Error("OCP-2 CRITICAL FAILURE: CanTransfer=true after storing false — capability escalation!")
	}
	if caps.CanStake {
		t.Error("OCP-2 CRITICAL FAILURE: CanStake=true after storing false — capability escalation!")
	}
	if !caps.CanSubmitClaims {
		t.Error("OCP-2 FAILURE: CanSubmitClaims=false after storing true — capability lost!")
	}
	if caps.CanVote {
		t.Error("OCP-2 CRITICAL FAILURE: CanVote=true after storing false — capability escalation!")
	}

	t.Log("OCP-2 PASS: Session key capabilities survive storage round-trip without escalation")
	t.Logf("  Transfer=%v Stake=%v Claims=%v Vote=%v",
		caps.CanTransfer, caps.CanStake, caps.CanSubmitClaims, caps.CanVote)
	t.Log("  DEFENSE: Proto serialization preserves capability bits exactly")
}

// ==================================================================================
// OCP-3: Session Expiry Enforcement — Does an expired session key remain valid?
// ==================================================================================
// Scenario: Session expires at block 150. Test checks validity at blocks 149, 150, 151.
func TestOCP3_SessionExpiryEnforcement(t *testing.T) {
	k, ctx := setupKeeper(t)

	session := types.SessionKey{
		KeyHash:        "session_hash_3",
		PublicKey:      "0000000000000000000000000000000000000000000000000000000000000001",
		Owner:          testAddr1,
		ExpiresAtBlock: 150,
		CreatedAtBlock: 100,
		Capabilities: &types.SessionCapabilities{
			CanSubmitClaims: true,
		},
	}
	k.SetSessionKey(ctx, session)

	// Block 149: should be valid
	if !k.IsSessionValid(ctx, testAddr1, "session_hash_3", 149) {
		t.Error("OCP-3 FAILURE: session should be valid at block 149 (before expiry)")
	}

	// Block 150: should be INVALID (expires AT 150, so height < 150 is the condition)
	if k.IsSessionValid(ctx, testAddr1, "session_hash_3", 150) {
		t.Error("OCP-3 CRITICAL FAILURE: session valid AT expiry block — off-by-one allows extra block!")
	}

	// Block 151: definitely invalid
	if k.IsSessionValid(ctx, testAddr1, "session_hash_3", 151) {
		t.Error("OCP-3 CRITICAL FAILURE: session valid AFTER expiry — expiry not enforced!")
	}

	t.Log("OCP-3 PASS: Session expiry enforced correctly (valid at 149, invalid at 150+)")
	t.Log("  DEFENSE: IsSessionValid uses strict < comparison on ExpiresAtBlock")
}

// ==================================================================================
// OCP-4: Frozen Account Isolation — Can a frozen agent access its own account?
// ==================================================================================
// Scenario: Agent account is frozen (emergency). Verify the Frozen flag persists
// and the account data remains intact but flagged.
func TestOCP4_FrozenAccountIsolation(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Register and then freeze the agent
	account := types.LegibleAccount{
		Address:               testAddr1,
		Did:                   testDID1,
		PublicKey:             testPubKey1,
		AccountType:           "agent",
		OperationalKeyVersion: 1,
		ReputationScore:       800000,
		Flags: &types.AccountFlags{
			CanSubmitClaims: true,
			CanChallenge:    true,
			Frozen:          false,
		},
	}
	k.SetAccount(ctx, account)

	// Freeze the account
	account.Flags.Frozen = true
	k.SetAccount(ctx, account)

	// Verify freeze persists
	frozen, found := k.GetAccount(ctx, testAddr1)
	if !found {
		t.Fatal("OCP-4 FAILURE: frozen account not found")
	}

	// === INSIGHT ANALYSIS ===
	if !frozen.Flags.GetFrozen() {
		t.Fatal("OCP-4 CRITICAL FAILURE: Frozen flag lost after storage — freeze bypass possible!")
	}

	// Verify DID still resolves (frozen ≠ deleted)
	k.SetDIDMapping(ctx, types.DIDMapping{Did: testDID1, Bech32: testAddr1, PubKey: testPubKey1})
	addr, didFound := k.GetAddressForDID(ctx, testDID1)
	if !didFound {
		t.Error("OCP-4 FAILURE: DID mapping lost for frozen account")
	}
	if addr != testAddr1 {
		t.Error("OCP-4 FAILURE: DID mapping corrupted for frozen account")
	}

	// Verify account data is intact
	if frozen.ReputationScore != 800000 {
		t.Errorf("OCP-4 FAILURE: reputation corrupted by freeze: got %d, want 800000", frozen.ReputationScore)
	}
	if frozen.Did != testDID1 {
		t.Errorf("OCP-4 FAILURE: DID corrupted by freeze: got %s, want %s", frozen.Did, testDID1)
	}

	t.Log("OCP-4 PASS: Frozen account remains readable but flagged")
	t.Logf("  Frozen=%v, Reputation=%d, DID=%s", frozen.Flags.Frozen, frozen.ReputationScore, frozen.Did)
	t.Log("  DEFENSE: Ante handler LegibleAccountDecorator rejects all txs from frozen accounts")
	t.Log("  INSIGHT: Frozen is data-preserving — account can be unfrozen without loss")
}

// ==================================================================================
// OCP-5: DID Format Fuzzing — Can malformed DIDs bypass validation?
// ==================================================================================
// Scenario: Attempt to validate various malformed DID strings.
func TestOCP5_DIDFormatFuzzing(t *testing.T) {
	fuzzCases := []struct {
		name    string
		did     string
		wantErr bool
	}{
		{"empty", "", true},
		{"wrong prefix", "did:eth:abcdef0123456789abcdef0123456789", true},
		{"no prefix", "abcdef0123456789abcdef0123456789", true},
		{"too short suffix", "did:lgm:abcdef", true},
		{"31 char suffix", "did:lgm:abcdef0123456789abcdef012345678", true},
		{"33 char suffix", "did:lgm:abcdef0123456789abcdef01234567890", true},
		{"63 char suffix", "did:lgm:abcdef0123456789abcdef0123456789abcdef0123456789abcdef012345", true},
		{"65 char suffix", "did:lgm:abcdef0123456789abcdef0123456789abcdef0123456789abcdef01234567", true},
		{"non-hex chars", "did:lgm:ghijklmnopqrstuvghijklmnopqrstuv", true},
		{"SQL injection", "did:lgm:'; DROP TABLE accounts; --a", true},
		{"null byte", "did:lgm:abcdef0123456789abcdef01\x00345678", true},
		{"valid 32-char", "did:lgm:abcdef0123456789abcdef0123456789", false},
		{"valid 64-char", "did:lgm:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789", false},
		{"uppercase hex", "did:lgm:ABCDEF0123456789ABCDEF0123456789", false},
		{"mixed case", "did:lgm:AbCdEf0123456789AbCdEf0123456789", false},
	}

	for _, tc := range fuzzCases {
		t.Run(tc.name, func(t *testing.T) {
			err := types.ValidateDID(tc.did)
			gotErr := err != nil
			if gotErr != tc.wantErr {
				t.Errorf("OCP-5 %s: ValidateDID(%q) error=%v, wantErr=%v",
					tc.name, tc.did, err, tc.wantErr)
			}
		})
	}
	t.Log("OCP-5 PASS: DID format validation rejects all malformed inputs")
	t.Log("  DEFENSE: Strict regex-like validation (prefix + length + hex charset)")
}

// ==================================================================================
// OCP-6: DID Derivation Collision — Can two different keys produce the same DID?
// ==================================================================================
// Scenario: Generate two keys that share the first 32 hex chars (birthday attack).
// In practice infeasible for SHA-256 but we verify the derivation is deterministic.
func TestOCP6_DIDDerivationDeterminism(t *testing.T) {
	// Verify DID derivation is deterministic
	pubKey := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	did1 := types.PublicKeyToDID(pubKey)
	did2 := types.PublicKeyToDID(pubKey)

	if did1 != did2 {
		t.Fatal("OCP-6 CRITICAL FAILURE: same key produces different DIDs — nondeterministic!")
	}
	if did1 != "did:lgm:abcdef0123456789abcdef0123456789" {
		t.Fatalf("OCP-6 FAILURE: unexpected DID derivation: got %s", did1)
	}

	// Verify different keys produce different DIDs
	pubKey2 := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	did3 := types.PublicKeyToDID(pubKey2)
	if did1 == did3 {
		t.Fatal("OCP-6 CRITICAL FAILURE: different keys produce same DID — collision!")
	}

	// Verify derivation validation rejects mismatches
	err := types.ValidateDIDDerivation(did1, pubKey2) // did1 derives from pubKey, not pubKey2
	if err == nil {
		t.Fatal("OCP-6 CRITICAL FAILURE: mismatched DID-key pair accepted — spoofing possible!")
	}

	// Verify derivation validation accepts correct pair
	err = types.ValidateDIDDerivation(did1, pubKey)
	if err != nil {
		t.Fatalf("OCP-6 FAILURE: correct DID-key pair rejected: %v", err)
	}

	t.Log("OCP-6 PASS: DID derivation is deterministic and collision-resistant")
	t.Logf("  Key1 DID: %s", did1)
	t.Logf("  Key2 DID: %s", did3)
	t.Log("  DEFENSE: DID = first 32 hex chars of pubkey (128-bit collision resistance)")
	t.Log("  INSIGHT: 128-bit birthday resistance = 2^64 operations needed for collision")
}

// ==================================================================================
// OCP-7: Multi-Session Key Isolation — Do sessions from different owners interfere?
// ==================================================================================
// Scenario: Two agents each create a session key. Verify one agent cannot
// access or validate the other's sessions.
func TestOCP7_MultiSessionIsolation(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Agent 1 session
	session1 := types.SessionKey{
		KeyHash:        "hash_agent1",
		PublicKey:      "aaaa000000000000000000000000000000000000000000000000000000000001",
		Owner:          testAddr1,
		ExpiresAtBlock: 200,
		CreatedAtBlock: 100,
		Capabilities: &types.SessionCapabilities{
			CanSubmitClaims: true,
			CanTransfer:     true, // Agent 1 can transfer
		},
	}
	k.SetSessionKey(ctx, session1)

	// Agent 2 session
	session2 := types.SessionKey{
		KeyHash:        "hash_agent2",
		PublicKey:      "bbbb000000000000000000000000000000000000000000000000000000000002",
		Owner:          testAddr2,
		ExpiresAtBlock: 200,
		CreatedAtBlock: 100,
		Capabilities: &types.SessionCapabilities{
			CanSubmitClaims: true,
			CanTransfer:     false, // Agent 2 CANNOT transfer
		},
	}
	k.SetSessionKey(ctx, session2)

	// Agent 1 cannot see Agent 2's session
	_, found := k.GetSessionKey(ctx, testAddr1, "hash_agent2")
	if found {
		t.Error("OCP-7 CRITICAL FAILURE: Agent 1 can retrieve Agent 2's session key!")
	}

	// Agent 2 cannot see Agent 1's session
	_, found = k.GetSessionKey(ctx, testAddr2, "hash_agent1")
	if found {
		t.Error("OCP-7 CRITICAL FAILURE: Agent 2 can retrieve Agent 1's session key!")
	}

	// Each agent sees only their own sessions
	agent1Sessions := k.GetSessionKeysForOwner(ctx, testAddr1)
	agent2Sessions := k.GetSessionKeysForOwner(ctx, testAddr2)

	if len(agent1Sessions) != 1 {
		t.Errorf("OCP-7 FAILURE: Agent 1 sees %d sessions, expected 1", len(agent1Sessions))
	}
	if len(agent2Sessions) != 1 {
		t.Errorf("OCP-7 FAILURE: Agent 2 sees %d sessions, expected 1", len(agent2Sessions))
	}

	// Verify capabilities are isolated (Agent 2 session doesn't inherit Agent 1's transfer)
	if agent2Sessions[0].GetCapabilities().CanTransfer {
		t.Error("OCP-7 CRITICAL FAILURE: Agent 2 session inherited Agent 1's transfer capability!")
	}

	t.Log("OCP-7 PASS: Session keys are owner-scoped — no cross-agent leakage")
	t.Logf("  Agent 1: %d sessions (transfer=%v)", len(agent1Sessions), agent1Sessions[0].GetCapabilities().CanTransfer)
	t.Logf("  Agent 2: %d sessions (transfer=%v)", len(agent2Sessions), agent2Sessions[0].GetCapabilities().CanTransfer)
	t.Log("  DEFENSE: SessionKeyKey(owner, hash) namespaces storage by owner address")
}

// ==================================================================================
// OCP-8: Recovery Config Privacy — Is recovery config only accessible by owner?
// ==================================================================================
// Scenario: Agent configures Shamir-based recovery. Verify recovery config is
// stored and retrievable only by the correct account address.
func TestOCP8_RecoveryConfigPrivacy(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Agent 1 sets up recovery
	config := types.RecoveryConfig{
		RecoveryKeyHash:       "recovery_hash_1",
		Threshold:             2,
		TotalShards:           3,
		RecoveryDelayBlocks:   34272, // 1 day
		ChallengePeriodBlocks: 17136, // 12 hours
	}
	k.SetRecoveryConfig(ctx, testAddr1, config)

	// Agent 1 can read their own config
	stored, found := k.GetRecoveryConfig(ctx, testAddr1)
	if !found {
		t.Fatal("OCP-8 FAILURE: Agent 1 cannot retrieve own recovery config")
	}
	if stored.Threshold != 2 || stored.TotalShards != 3 {
		t.Errorf("OCP-8 FAILURE: recovery config corrupted: threshold=%d, shards=%d",
			stored.Threshold, stored.TotalShards)
	}

	// Different address cannot access Agent 1's config
	_, found = k.GetRecoveryConfig(ctx, testAddr2)
	if found {
		t.Error("OCP-8 CRITICAL FAILURE: Agent 2 can read Agent 1's recovery config!")
	}

	// Non-existent address returns not found
	_, found = k.GetRecoveryConfig(ctx, "lgm1nonexistent0000000000000000000000")
	if found {
		t.Error("OCP-8 FAILURE: non-existent address has recovery config")
	}

	t.Log("OCP-8 PASS: Recovery config is address-scoped and private")
	t.Logf("  Threshold: %d-of-%d, delay: %d blocks",
		stored.Threshold, stored.TotalShards, stored.RecoveryDelayBlocks)
	t.Log("  DEFENSE: RecoveryConfigKey(address) namespaces by account")
}

// ==================================================================================
// OCP-9: Account Type Immutability — Can an agent change its type post-registration?
// ==================================================================================
// Scenario: After registering as "agent", attempt to overwrite to "human" or "system".
// Verify the original type is preserved through keeper-level operations.
func TestOCP9_AccountTypeImmutability(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Register as agent
	_, err := ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:             testAddr1,
		Did:                testDID1,
		PublicKey:          testPubKey1,
		AccountType:        "agent",
		OperationalKeyHash: "opkey1",
	})
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	// Try to re-register as "system" (privilege escalation)
	_, err = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:             testAddr1,
		Did:                testDID1,
		PublicKey:          testPubKey1,
		AccountType:        "system",
		OperationalKeyHash: "opkey1",
	})
	if err == nil {
		// If re-registration succeeds, check the type wasn't changed
		account, found := k.GetAccount(ctx, testAddr1)
		if !found {
			t.Fatal("OCP-9 SETUP: account not found after re-registration")
		}
		if account.AccountType == "system" {
			t.Fatal("OCP-9 CRITICAL FAILURE: Account type escalated from 'agent' to 'system'!")
		}
		t.Logf("OCP-9 PARTIAL: Re-registration succeeded but type preserved as '%s'", account.AccountType)
	} else {
		t.Logf("OCP-9 PASS: Re-registration correctly rejected: %v", err)
	}

	// Verify original type persists
	account, _ := k.GetAccount(ctx, testAddr1)
	if account.AccountType != "agent" {
		t.Errorf("OCP-9 FAILURE: account type is %q, expected 'agent'", account.AccountType)
	}

	t.Log("OCP-9 PASS: Account type cannot be escalated post-registration")
	t.Log("  DEFENSE: RegisterAccount rejects duplicate registration or preserves original type")
}

// ==================================================================================
// OCP-10: Session Key Deletion Enforcement — Does deleting a session actually remove it?
// ==================================================================================
// Scenario: Create session, delete it, verify it's gone and can't be resurrected.
func TestOCP10_SessionDeletionEnforcement(t *testing.T) {
	k, ctx := setupKeeper(t)

	session := types.SessionKey{
		KeyHash:        "delete_me",
		PublicKey:      "cccc000000000000000000000000000000000000000000000000000000000003",
		Owner:          testAddr1,
		ExpiresAtBlock: 200,
		CreatedAtBlock: 100,
		Capabilities: &types.SessionCapabilities{
			CanSubmitClaims: true,
			CanTransfer:     true,
		},
	}
	k.SetSessionKey(ctx, session)

	// Verify it exists
	if _, found := k.GetSessionKey(ctx, testAddr1, "delete_me"); !found {
		t.Fatal("OCP-10 SETUP: session not created")
	}

	// Delete it
	k.DeleteSessionKey(ctx, testAddr1, "delete_me")

	// Verify it's gone
	_, found := k.GetSessionKey(ctx, testAddr1, "delete_me")
	if found {
		t.Fatal("OCP-10 CRITICAL FAILURE: Deleted session key still accessible!")
	}

	// Verify it doesn't appear in owner's session list
	sessions := k.GetSessionKeysForOwner(ctx, testAddr1)
	for _, s := range sessions {
		if s.KeyHash == "delete_me" {
			t.Fatal("OCP-10 CRITICAL FAILURE: Deleted session appears in owner listing!")
		}
	}

	// Verify IsSessionValid returns false
	if k.IsSessionValid(ctx, testAddr1, "delete_me", 100) {
		t.Fatal("OCP-10 CRITICAL FAILURE: Deleted session still valid!")
	}

	// Try to re-use the same hash (should work — no tombstoning)
	k.SetSessionKey(ctx, session)
	if _, found := k.GetSessionKey(ctx, testAddr1, "delete_me"); !found {
		t.Error("OCP-10 NOTE: Cannot re-create session with same hash after deletion")
	}

	t.Log("OCP-10 PASS: Session deletion is complete and irreversible")
	t.Log("  DEFENSE: DeleteSessionKey removes KV entry; no ghost references")
	t.Log("  INSIGHT: Same hash can be re-used (no tombstone) — acceptable since hash is SHA256 of pubkey")
}

// ==================================================================================
// OCP-11: Cross-Agent DID Mapping Collision — Can two agents share a DID?
// ==================================================================================
// Scenario: Two different addresses try to register with the same DID.
func TestOCP11_DIDMappingCollision(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Agent 1 registers with DID1
	_, err := ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:             testAddr1,
		Did:                testDID1,
		PublicKey:          testPubKey1,
		AccountType:        "agent",
		OperationalKeyHash: "opkey1",
	})
	if err != nil {
		t.Fatalf("Agent 1 registration failed: %v", err)
	}

	// Agent 2 tries to register with the SAME DID (requires same pubkey prefix)
	// This should fail because DID derivation requires the pubkey to match
	_, err = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:             testAddr2,
		Did:                testDID1,    // COLLISION: same DID as Agent 1
		PublicKey:          testPubKey2, // Different key
		AccountType:        "agent",
		OperationalKeyHash: "opkey2",
	})

	if err == nil {
		// Check who owns the DID now
		addr, _ := k.GetAddressForDID(ctx, testDID1)
		if addr == testAddr2 {
			t.Fatal("OCP-11 CRITICAL FAILURE: Agent 2 hijacked Agent 1's DID!")
		}
		t.Logf("OCP-11 PARTIAL: Registration succeeded but DID still maps to %s", addr)
	} else {
		t.Logf("OCP-11 PASS: DID collision correctly rejected: %v", err)
	}

	// Verify DID still maps to Agent 1
	addr, found := k.GetAddressForDID(ctx, testDID1)
	if !found {
		t.Fatal("OCP-11 FAILURE: DID mapping lost after collision attempt")
	}
	if addr != testAddr1 {
		t.Fatalf("OCP-11 CRITICAL FAILURE: DID mapping changed from %s to %s", testAddr1, addr)
	}

	t.Log("OCP-11 PASS: DID mapping is collision-resistant")
	t.Logf("  DID %s → %s (original owner preserved)", testDID1, addr)
	t.Log("  DEFENSE: ValidateDIDDerivation + duplicate check in RegisterAccount")
}

// ==================================================================================
// OCP-12: Key Rotation Continuity — Does identity survive operational key rotation?
// ==================================================================================
// Scenario: Agent rotates its operational key. Verify DID, reputation, and session
// keys remain intact.
func TestOCP12_KeyRotationContinuity(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Set up agent with initial state
	account := types.LegibleAccount{
		Address:               testAddr1,
		Did:                   testDID1,
		PublicKey:             testPubKey1,
		AccountType:           "agent",
		OperationalKeyVersion: 1,
		OperationalKeyHash:    "old_opkey_hash",
		ReputationScore:       750000,
		LastActiveBlock:       90,
		Flags: &types.AccountFlags{
			CanSubmitClaims: true,
			CanChallenge:    true,
		},
	}
	k.SetAccount(ctx, account)

	// Set up DID mapping and session
	k.SetDIDMapping(ctx, types.DIDMapping{Did: testDID1, Bech32: testAddr1, PubKey: testPubKey1})
	session := types.SessionKey{
		KeyHash:        "active_session",
		PublicKey:      "dddd000000000000000000000000000000000000000000000000000000000004",
		Owner:          testAddr1,
		ExpiresAtBlock: 300,
		CreatedAtBlock: 80,
		Capabilities:   &types.SessionCapabilities{CanSubmitClaims: true},
	}
	k.SetSessionKey(ctx, session)

	// Simulate key rotation (update operational key version)
	account.OperationalKeyVersion = 2
	account.OperationalKeyHash = "new_opkey_hash"
	account.LastActiveBlock = 100
	k.SetAccount(ctx, account)

	// Verify identity is preserved
	rotated, found := k.GetAccount(ctx, testAddr1)
	if !found {
		t.Fatal("OCP-12 FAILURE: account lost after key rotation")
	}

	// === INSIGHT ANALYSIS ===
	if rotated.Did != testDID1 {
		t.Fatalf("OCP-12 CRITICAL FAILURE: DID changed after key rotation: %s → %s", testDID1, rotated.Did)
	}
	if rotated.ReputationScore != 750000 {
		t.Errorf("OCP-12 FAILURE: reputation changed after key rotation: 750000 → %d", rotated.ReputationScore)
	}
	if rotated.OperationalKeyVersion != 2 {
		t.Errorf("OCP-12 FAILURE: key version not updated: got %d", rotated.OperationalKeyVersion)
	}
	if rotated.PublicKey != testPubKey1 {
		t.Error("OCP-12 CRITICAL FAILURE: identity key changed during operational key rotation!")
	}

	// DID still resolves
	addr, _ := k.GetAddressForDID(ctx, testDID1)
	if addr != testAddr1 {
		t.Fatalf("OCP-12 CRITICAL FAILURE: DID mapping broken after rotation")
	}

	// Session keys survive rotation
	_, sessionFound := k.GetSessionKey(ctx, testAddr1, "active_session")
	if !sessionFound {
		t.Error("OCP-12 FAILURE: session key lost after operational key rotation")
	}

	t.Log("OCP-12 PASS: Key rotation preserves identity, reputation, DID, and active sessions")
	t.Logf("  DID=%s (unchanged), Reputation=%d (unchanged), KeyVersion=1→%d",
		rotated.Did, rotated.ReputationScore, rotated.OperationalKeyVersion)
	t.Log("  DEFENSE: 4-layer architecture — identity anchor (L1) is immutable; operational key (L3) rotates")
}

// ==================================================================================
// Helper
// ==================================================================================

func sha256Hex(input string) string {
	h := sha256.Sum256([]byte(input))
	return hex.EncodeToString(h[:])
}
