package vm

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"
)

// === Audit: Confidence Scaling ===

func TestKnowledgeAudit_ConfidenceScaling_Zero(t *testing.T) {
	factKey := WordToBytes32(big.NewInt(1))
	factIdHex := fmt.Sprintf("%x", factKey[:])
	mk := &mockKnowledgeKeeper{
		facts:  map[string]interface{}{factIdHex: &mockFact{confidence: 0, domain: "test"}},
		domain: map[string][]interface{}{},
	}
	kh := NewKnowledgeHost(mk, context.Background())
	exists, conf, _ := kh.QueryFact(factKey[:])
	if !exists {
		t.Fatal("expected exists")
	}
	if conf != 0 {
		t.Fatalf("expected 0 confidence, got %d", conf)
	}
}

func TestKnowledgeAudit_ConfidenceScaling_Max(t *testing.T) {
	factKey := WordToBytes32(big.NewInt(2))
	factIdHex := fmt.Sprintf("%x", factKey[:])
	mk := &mockKnowledgeKeeper{
		facts:  map[string]interface{}{factIdHex: &mockFact{confidence: 1_000_000, domain: "test"}},
		domain: map[string][]interface{}{},
	}
	kh := NewKnowledgeHost(mk, context.Background())
	_, conf, _ := kh.QueryFact(factKey[:])
	// 1M * 1e12 = 1e18
	expected := uint64(1_000_000_000_000_000_000)
	if conf != expected {
		t.Fatalf("expected %d, got %d", expected, conf)
	}
}

func TestKnowledgeAudit_ConfidenceScaling_Mid(t *testing.T) {
	factKey := WordToBytes32(big.NewInt(3))
	factIdHex := fmt.Sprintf("%x", factKey[:])
	mk := &mockKnowledgeKeeper{
		facts:  map[string]interface{}{factIdHex: &mockFact{confidence: 500_000, domain: "test"}},
		domain: map[string][]interface{}{},
	}
	kh := NewKnowledgeHost(mk, context.Background())
	_, conf, _ := kh.QueryFact(factKey[:])
	expected := uint64(500_000_000_000_000_000)
	if conf != expected {
		t.Fatalf("expected %d, got %d", expected, conf)
	}
}

// === Audit: Nil Keeper Safety ===

func TestKnowledgeAudit_NilKeeper_AllMethods(t *testing.T) {
	kh := NewKnowledgeHost(nil, context.Background())

	exists, conf, domain := kh.QueryFact([]byte("test"))
	if exists || conf != 0 || domain != nil {
		t.Fatal("QueryFact should return zero values with nil keeper")
	}

	count, ids := kh.QueryDomain([]byte("test"), 10)
	if count != 0 || ids != nil {
		t.Fatal("QueryDomain should return zero values with nil keeper")
	}

	claimId := kh.SubmitClaim("did:lgm:x", []byte("d"), []byte("h"), []byte("e"), 1)
	if claimId != nil {
		t.Fatal("SubmitClaim should return nil with nil keeper")
	}

	ok := kh.CommitVote("did:lgm:x", []byte("c"), []byte("v"), 1)
	if ok {
		t.Fatal("CommitVote should return false with nil keeper")
	}

	ok = kh.RevealVote("did:lgm:x", []byte("c"), big.NewInt(1), big.NewInt(42), 1)
	if ok {
		t.Fatal("RevealVote should return false with nil keeper")
	}

	challengeId := kh.InitiateChallenge("did:lgm:x", []byte("f"), []byte("e"), big.NewInt(100), 1)
	if challengeId != nil {
		t.Fatal("InitiateChallenge should return nil with nil keeper")
	}
}

// === Audit: Empty/missing callerDID ===

func TestKnowledgeAudit_EmptyCallerDID(t *testing.T) {
	mk := newMockKnowledgeKeeper()
	kh := NewKnowledgeHost(mk, context.Background())

	// SubmitClaim with empty DID should fail
	claimId := kh.SubmitClaim("", []byte("domain"), []byte("hash"), []byte("ev"), 1)
	if claimId != nil {
		t.Fatal("SubmitClaim should reject empty callerDID")
	}

	// CommitVote with empty DID should fail
	ok := kh.CommitVote("", []byte("claim"), []byte("vote"), 1)
	if ok {
		t.Fatal("CommitVote should reject empty callerDID")
	}

	// RevealVote with empty DID should fail
	ok = kh.RevealVote("", []byte("claim"), big.NewInt(1), big.NewInt(42), 1)
	if ok {
		t.Fatal("RevealVote should reject empty callerDID")
	}

	// InitiateChallenge with empty DID should fail
	challengeId := kh.InitiateChallenge("", []byte("fact"), []byte("ev"), big.NewInt(100), 1)
	if challengeId != nil {
		t.Fatal("InitiateChallenge should reject empty callerDID")
	}
}

// === Audit: INITIATE_CHALLENGE with zero stake ===

func TestKnowledgeAudit_ChallengeZeroStake(t *testing.T) {
	mk := newMockKnowledgeKeeper()
	kh := NewKnowledgeHost(mk, context.Background())

	// Zero stake should be rejected
	result := kh.InitiateChallenge("did:lgm:x", []byte("fact"), []byte("ev"), big.NewInt(0), 1)
	if result != nil {
		t.Fatal("InitiateChallenge should reject zero stake")
	}

	// Nil stake should be rejected
	result = kh.InitiateChallenge("did:lgm:x", []byte("fact"), []byte("ev"), nil, 1)
	if result != nil {
		t.Fatal("InitiateChallenge should reject nil stake")
	}
}

// === Audit: INITIATE_CHALLENGE with empty factId ===

func TestKnowledgeAudit_ChallengeEmptyFactId(t *testing.T) {
	mk := newMockKnowledgeKeeper()
	kh := NewKnowledgeHost(mk, context.Background())

	result := kh.InitiateChallenge("did:lgm:x", []byte{}, []byte("ev"), big.NewInt(100), 1)
	if result != nil {
		t.Fatal("InitiateChallenge should reject empty factId")
	}

	result = kh.InitiateChallenge("did:lgm:x", nil, []byte("ev"), big.NewInt(100), 1)
	if result != nil {
		t.Fatal("InitiateChallenge should reject nil factId")
	}
}

// === Audit: CommitVote edge cases ===

func TestKnowledgeAudit_CommitVoteEmptyInputs(t *testing.T) {
	mk := newMockKnowledgeKeeper()
	kh := NewKnowledgeHost(mk, context.Background())

	// Empty claimId
	ok := kh.CommitVote("did:lgm:x", []byte{}, []byte("vote"), 1)
	if ok {
		t.Fatal("CommitVote should reject empty claimId")
	}

	// Empty voteHash
	ok = kh.CommitVote("did:lgm:x", []byte("claim"), []byte{}, 1)
	if ok {
		t.Fatal("CommitVote should reject empty voteHash")
	}
}

// === Audit: RevealVote nil vote/salt ===

func TestKnowledgeAudit_RevealVoteNilInputs(t *testing.T) {
	mk := newMockKnowledgeKeeper()
	kh := NewKnowledgeHost(mk, context.Background())

	ok := kh.RevealVote("did:lgm:x", []byte("claim"), nil, big.NewInt(42), 1)
	if ok {
		t.Fatal("RevealVote should reject nil vote")
	}

	ok = kh.RevealVote("did:lgm:x", []byte("claim"), big.NewInt(1), nil, 1)
	if ok {
		t.Fatal("RevealVote should reject nil salt")
	}
}

// === Audit: RevealVote verdict encoding ===

func TestKnowledgeAudit_RevealVoteVerdictEncoding(t *testing.T) {
	// Track what gets passed to the keeper
	var capturedVerdict string
	var capturedConfidence uint64

	mk := &trackingKnowledgeKeeper{
		mockKnowledgeKeeper: newMockKnowledgeKeeper(),
		onReveal: func(verdict string, confidence uint64) {
			capturedVerdict = verdict
			capturedConfidence = confidence
		},
	}
	kh := NewKnowledgeHost(mk, context.Background())

	// vote=0 → reject, default confidence 800k
	kh.RevealVote("did:lgm:x", []byte("claim"), big.NewInt(0), big.NewInt(1), 1)
	if capturedVerdict != "reject" {
		t.Fatalf("vote=0 should be reject, got %s", capturedVerdict)
	}
	if capturedConfidence != 800_000 {
		t.Fatalf("vote=0 confidence should be 800000, got %d", capturedConfidence)
	}

	// vote=1 → accept, default confidence 800k
	kh.RevealVote("did:lgm:x", []byte("claim"), big.NewInt(1), big.NewInt(1), 1)
	if capturedVerdict != "accept" {
		t.Fatalf("vote=1 should be accept, got %s", capturedVerdict)
	}
	if capturedConfidence != 800_000 {
		t.Fatalf("vote=1 confidence should be 800000, got %d", capturedConfidence)
	}

	// vote=500000 → accept, confidence=500000
	kh.RevealVote("did:lgm:x", []byte("claim"), big.NewInt(500_000), big.NewInt(1), 1)
	if capturedVerdict != "accept" {
		t.Fatalf("vote=500000 should be accept, got %s", capturedVerdict)
	}
	if capturedConfidence != 500_000 {
		t.Fatalf("vote=500000 confidence should be 500000, got %d", capturedConfidence)
	}

	// vote=2000000 → accept, confidence capped at 1M
	kh.RevealVote("did:lgm:x", []byte("claim"), big.NewInt(2_000_000), big.NewInt(1), 1)
	if capturedConfidence != 1_000_000 {
		t.Fatalf("vote=2M confidence should be capped at 1000000, got %d", capturedConfidence)
	}
}

// === Audit: trimNullBytes ===

func TestKnowledgeAudit_TrimNullBytes(t *testing.T) {
	// Normal case
	result := trimNullBytes([]byte("hello\x00\x00"))
	if string(result) != "hello" {
		t.Fatalf("expected 'hello', got %q", string(result))
	}

	// All nulls → nil
	result = trimNullBytes([]byte{0, 0, 0})
	if result != nil {
		t.Fatal("all-null bytes should return nil")
	}

	// Empty → nil
	result = trimNullBytes([]byte{})
	if result != nil {
		t.Fatal("empty bytes should return nil")
	}

	// No nulls
	result = trimNullBytes([]byte("abc"))
	if string(result) != "abc" {
		t.Fatalf("expected 'abc', got %q", string(result))
	}

	// Embedded nulls (only trailing removed)
	result = trimNullBytes([]byte("a\x00b\x00\x00"))
	if string(result) != "a\x00b" {
		t.Fatalf("expected 'a\\x00b', got %q", string(result))
	}
}

// === Audit: FARM-4 COMPARE_BELIEF anti-herding ===

func TestKnowledgeAudit_FARM4_CompareBelief_CommitPhase(t *testing.T) {
	mk := newMockKnowledgeKeeper()
	host := newMockFullHost(mk)

	// COMPARE_BELIEF pops 1, pushes 1
	bytecode := []byte{
		PUSH1, 0x42,
		COMPARE_BELIEF,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}

	// During commit phase: should return 0
	ctx := makeCtx(100000)
	ctx.CallerDID = "did:lgm:test"
	ctx.IsCommitPhase = true
	ctx.Capabilities = &SessionCapabilities{CanTransfer: true, CanStake: true, CanSubmitClaims: true, CanVote: true, CanManageTrust: true, CanManageIdentity: true, CanManageSchedules: true, CanManageDefenses: true, CanManageBeliefs: true}
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Sign() != 0 {
		t.Fatalf("COMPARE_BELIEF during commit phase should return 0, got %s", val)
	}

	// Outside commit phase: should work normally (mock returns 0 anyway, but code path differs)
	ctx2 := makeCtx(100000)
	ctx2.CallerDID = "did:lgm:test"
	ctx2.IsCommitPhase = false
	ctx2.Capabilities = &SessionCapabilities{CanTransfer: true, CanStake: true, CanSubmitClaims: true, CanVote: true}
	ctx2.Bytecode = bytecode
	interp2 := NewInterpreter()
	result2 := interp2.Execute(bytecode, ctx2, state, host)
	if !result2.Success {
		t.Fatalf("expected success outside commit phase, got error: %v", result2.Error)
	}
}

// === Audit: Capability checks ===

func TestKnowledgeAudit_SubmitClaimDeniedWithoutCapability(t *testing.T) {
	mk := newMockKnowledgeKeeper()
	host := newMockFullHost(mk)

	bytecode := []byte{
		PUSH1, 0xEE, PUSH1, 0xDD, PUSH1, 0xAA,
		SUBMIT_CLAIM,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}

	// No capabilities at all (nil = deny all)
	ctx := makeCtx(100000)
	ctx.CallerDID = "did:lgm:test"
	ctx.Capabilities = nil
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if !result.Success {
		t.Fatalf("expected success (denied gracefully), got error: %v", result.Error)
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Sign() != 0 {
		t.Fatal("expected 0 (denied) for SUBMIT_CLAIM without capabilities")
	}

	// With capabilities but CanSubmitClaims=false
	ctx2 := makeCtx(100000)
	ctx2.CallerDID = "did:lgm:test"
	ctx2.Capabilities = &SessionCapabilities{CanTransfer: true, CanStake: true, CanSubmitClaims: false, CanVote: true}
	ctx2.Bytecode = bytecode
	interp2 := NewInterpreter()
	result2 := interp2.Execute(bytecode, ctx2, state, host)
	if !result2.Success {
		t.Fatalf("expected success (denied gracefully)")
	}
	val2 := new(big.Int).SetBytes(result2.ReturnData)
	if val2.Sign() != 0 {
		t.Fatal("expected 0 (denied) for SUBMIT_CLAIM with CanSubmitClaims=false")
	}
}

func TestKnowledgeAudit_VoteDeniedWithoutCapability(t *testing.T) {
	mk := newMockKnowledgeKeeper()
	host := newMockFullHost(mk)

	// COMMIT_VOTE without vote capability
	bytecode := []byte{
		PUSH1, 0xBB, PUSH1, 0xAA,
		COMMIT_VOTE,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.CallerDID = "did:lgm:test"
	ctx.Capabilities = &SessionCapabilities{CanVote: false}
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if !result.Success {
		t.Fatalf("expected graceful denial")
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Sign() != 0 {
		t.Fatal("expected 0 for COMMIT_VOTE without CanVote")
	}
}

// === Audit: Host=nil graceful degradation ===

func TestKnowledgeAudit_NilHostGracefulDegradation(t *testing.T) {
	// All knowledge opcodes should push zeros when host is nil
	bytecode := []byte{
		PUSH1, 0x42,
		QUERY_FACT,
		// Stack: exists=0, confidence=0, domain=0
		POP, POP,
		PUSH1, 0, MSTORE,
		PUSH1, 32, PUSH1, 0, RETURN,
	}
	ctx := makeCtx(100000)
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, nil) // nil host
	if !result.Success {
		t.Fatalf("expected success with nil host, got error: %v", result.Error)
	}
	val := new(big.Int).SetBytes(result.ReturnData)
	if val.Sign() != 0 {
		t.Fatal("expected 0 from QUERY_FACT with nil host")
	}
}

// === Audit: QueryDomain null byte handling (potential bug) ===

func TestKnowledgeAudit_QueryDomainNullBytes(t *testing.T) {
	// QueryDomain does NOT call trimNullBytes on domain, unlike SubmitClaim.
	// A 32-byte padded domain "math\x00\x00..." won't match "math" in keeper.
	mk := &mockKnowledgeKeeper{
		facts: map[string]interface{}{},
		domain: map[string][]interface{}{
			"math": {&mockFact{confidence: 500000, domain: "math"}},
		},
	}
	kh := NewKnowledgeHost(mk, context.Background())

	// Padded domain (as would come from 32-byte stack word)
	padded := make([]byte, 32)
	copy(padded, "math")
	count, _ := kh.QueryDomain(padded, 10)

	// This will be 0 because "math\x00\x00..." != "math"
	// This is a known inconsistency: SubmitClaim trims nulls, QueryDomain does not
	if count != 0 {
		t.Log("QueryDomain matched with padded domain — trimming is applied")
	} else {
		t.Log("AUDIT NOTE: QueryDomain does NOT trim null bytes from domain — inconsistent with SubmitClaim which does trim")
	}
}

// === Audit: factIdHex encoding consistency ===

func TestKnowledgeAudit_FactIdHexEncoding(t *testing.T) {
	// Verify that fmt.Sprintf("%x", factId) matches hex.EncodeToString
	factId := WordToBytes32(big.NewInt(42))
	sprintfHex := fmt.Sprintf("%x", factId[:])
	encodeHex := hex.EncodeToString(factId[:])

	if sprintfHex != encodeHex {
		t.Fatalf("hex encoding mismatch: Sprintf=%q vs EncodeToString=%q", sprintfHex, encodeHex)
	}
}

// === Audit: SubmitClaim domain trimming ===

func TestKnowledgeAudit_SubmitClaimDomainTrimming(t *testing.T) {
	var capturedDomain string
	mk := &trackingKnowledgeKeeper{
		mockKnowledgeKeeper: newMockKnowledgeKeeper(),
		onSubmit: func(domain string) {
			capturedDomain = domain
		},
	}
	kh := NewKnowledgeHost(mk, context.Background())

	// Domain with trailing nulls (as from 32-byte word)
	domain := make([]byte, 32)
	copy(domain, "economics")
	kh.SubmitClaim("did:lgm:x", domain, []byte("hash"), []byte("ev"), 1)

	if capturedDomain != "economics" {
		t.Fatalf("expected trimmed domain 'economics', got %q", capturedDomain)
	}
}

// === Audit: SubmitClaim all-null domain ===

func TestKnowledgeAudit_SubmitClaimAllNullDomain(t *testing.T) {
	var capturedDomain string
	mk := &trackingKnowledgeKeeper{
		mockKnowledgeKeeper: newMockKnowledgeKeeper(),
		onSubmit: func(domain string) {
			capturedDomain = domain
		},
	}
	kh := NewKnowledgeHost(mk, context.Background())

	// All-null domain → trimNullBytes returns nil → string(nil) = ""
	domain := make([]byte, 32)
	kh.SubmitClaim("did:lgm:x", domain, []byte("hash"), []byte("ev"), 1)

	if capturedDomain != "" {
		t.Fatalf("expected empty domain from all-null bytes, got %q", capturedDomain)
	}
	t.Log("AUDIT NOTE: All-null domain produces empty string — keeper should validate")
}

// === Audit: Reentrancy guard (H-1) ===

func TestKnowledgeAudit_ReentrancyGuard(t *testing.T) {
	mk := newMockKnowledgeKeeper()
	host := newMockFullHost(mk)

	bytecode := []byte{
		PUSH1, 0x42,
		QUERY_FACT,
		POP, POP, POP,
		STOP,
	}
	ctx := makeCtx(100000)
	ctx.InHostCall = true // Simulate already inside a host call
	ctx.Bytecode = bytecode
	state := NewMemoryStateDB()
	interp := NewInterpreter()
	result := interp.Execute(bytecode, ctx, state, host)
	if result.Success {
		t.Fatal("expected failure due to reentrancy guard")
	}
}

// === Audit: INITIATE_CHALLENGE negative stake ===

func TestKnowledgeAudit_ChallengeNegativeStake(t *testing.T) {
	mk := newMockKnowledgeKeeper()
	kh := NewKnowledgeHost(mk, context.Background())

	// Negative big.Int has Sign() < 0, which is != 0, so it passes the Sign()==0 check
	// But Uint64() on negative big.Int returns 0 or wraps — potential issue
	negStake := big.NewInt(-100)
	result := kh.InitiateChallenge("did:lgm:x", []byte("fact"), []byte("ev"), negStake, 1)
	// Sign() is -1, not 0, so it passes the guard. Uint64() on -100 returns a large number.
	// This is a potential issue: negative stake could become a huge uint64.
	if result != nil {
		t.Log("AUDIT WARNING: Negative stake accepted by InitiateChallenge — stake.Uint64() wraps around")
	}
}

// === Tracking keeper for detailed assertion ===

type trackingKnowledgeKeeper struct {
	*mockKnowledgeKeeper
	onReveal func(verdict string, confidence uint64)
	onSubmit func(domain string)
}

func (m *trackingKnowledgeKeeper) RevealVoteFromBVM(_ context.Context, validatorDID, roundId, verdict string, confidence uint64, salt string) error {
	if m.onReveal != nil {
		m.onReveal(verdict, confidence)
	}
	return nil
}

func (m *trackingKnowledgeKeeper) SubmitClaimFromBVM(_ context.Context, submitterDID, domain, contentHash, evidenceHash, logicZone string, stakeAmount uint64) (string, error) {
	if m.onSubmit != nil {
		m.onSubmit(domain)
	}
	return fmt.Sprintf("claim-%s-%s", submitterDID, domain), nil
}
