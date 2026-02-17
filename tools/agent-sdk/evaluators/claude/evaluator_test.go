package claude

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
)

// --- Mock API Client ---

type mockMessageSender struct {
	response *MessagesResponse
	err      error
	calls    int
}

func (m *mockMessageSender) CreateMessage(_ context.Context, req MessagesRequest) (*MessagesResponse, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

// --- Mock Claim Fetcher ---

type mockClaimFetcher struct {
	claims map[string]*agentsdk.ClaimInfo
}

func (m *mockClaimFetcher) GetClaim(_ context.Context, claimID string) (*agentsdk.ClaimInfo, error) {
	claim, ok := m.claims[claimID]
	if !ok {
		return nil, fmt.Errorf("claim %s not found", claimID)
	}
	return claim, nil
}

// --- Helper to build a standard mock response ---

func mockEvalResponse(verdict string, confidence int, reasoning string) *MessagesResponse {
	text := fmt.Sprintf(`{"verdict": %q, "confidence": %d, "reasoning": %q}`,
		verdict, confidence, reasoning)
	return &MessagesResponse{
		ID:   "msg_test",
		Role: "assistant",
		Content: []ContentBlock{
			{Type: "text", Text: text},
		},
		Usage: Usage{InputTokens: 500, OutputTokens: 50},
	}
}

// --- Tests ---

func TestClaudeEvaluator_CacheHit(t *testing.T) {
	mock := &mockMessageSender{
		response: mockEvalResponse("true", 90, "well supported"),
	}
	fetcher := &mockClaimFetcher{
		claims: map[string]*agentsdk.ClaimInfo{
			"claim-1": {ID: "claim-1", Text: "Water boils at 100C at sea level", Domain: "physics", Category: "empirical"},
		},
	}

	eval := NewClaudeEvaluatorWithClient(mock, fetcher, "")

	// First call: should hit the API.
	verdict, conf, err := eval.Evaluate(context.Background(), "claim-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if verdict != agentsdk.VerdictTrue {
		t.Fatalf("expected VerdictTrue, got %s", verdict)
	}
	if conf != 90 {
		t.Fatalf("expected confidence 90, got %d", conf)
	}
	if mock.calls != 1 {
		t.Fatalf("expected 1 API call, got %d", mock.calls)
	}

	// Second call: should return cached verdict without API call.
	verdict2, conf2, err := eval.Evaluate(context.Background(), "claim-1")
	if err != nil {
		t.Fatalf("unexpected error on cache hit: %v", err)
	}
	if verdict2 != verdict || conf2 != conf {
		t.Fatalf("cached verdict mismatch: %s/%d vs %s/%d", verdict2, conf2, verdict, conf)
	}
	if mock.calls != 1 {
		t.Fatalf("expected still 1 API call after cache hit, got %d", mock.calls)
	}
}

func TestClaudeEvaluator_APICall(t *testing.T) {
	mock := &mockMessageSender{
		response: mockEvalResponse("false", 75, "contradicts known physics"),
	}
	fetcher := &mockClaimFetcher{
		claims: map[string]*agentsdk.ClaimInfo{
			"claim-2": {ID: "claim-2", Text: "Water boils at 50C at sea level", Domain: "physics", Category: "empirical"},
		},
	}

	eval := NewClaudeEvaluatorWithClient(mock, fetcher, "")

	verdict, conf, err := eval.Evaluate(context.Background(), "claim-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if verdict != agentsdk.VerdictFalse {
		t.Fatalf("expected VerdictFalse, got %s", verdict)
	}
	if conf != 75 {
		t.Fatalf("expected confidence 75, got %d", conf)
	}
	if mock.calls != 1 {
		t.Fatalf("expected 1 API call, got %d", mock.calls)
	}
}

func TestClaudeEvaluator_APIError(t *testing.T) {
	mock := &mockMessageSender{
		err: &APIError{StatusCode: 500, ErrorInfo: struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		}{"server_error", "internal error"}},
	}
	fetcher := &mockClaimFetcher{
		claims: map[string]*agentsdk.ClaimInfo{
			"claim-3": {ID: "claim-3", Text: "Some claim text for testing purposes", Domain: "general", Category: "analytic"},
		},
	}

	eval := NewClaudeEvaluatorWithClient(mock, fetcher, "")

	// API error should degrade to uncertain, not return error.
	verdict, conf, err := eval.Evaluate(context.Background(), "claim-3")
	if err != nil {
		t.Fatalf("expected no error on API failure, got: %v", err)
	}
	if verdict != agentsdk.VerdictUncertain {
		t.Fatalf("expected VerdictUncertain on API error, got %s", verdict)
	}
	if conf != 5 {
		t.Fatalf("expected low confidence on API error, got %d", conf)
	}
}

func TestClaudeEvaluator_RateLimited(t *testing.T) {
	mock := &mockMessageSender{
		response: mockEvalResponse("true", 80, "confirmed"),
	}
	fetcher := &mockClaimFetcher{
		claims: map[string]*agentsdk.ClaimInfo{
			"claim-4": {ID: "claim-4", Text: "Test claim about rate limiting behavior", Domain: "general", Category: "analytic"},
		},
	}

	eval := NewClaudeEvaluatorWithClient(mock, fetcher, "")

	// Set very tight rate limit.
	eval.subscription.MaxRequestsPerMinute = 1

	// First call should succeed.
	v1, _, err := eval.Evaluate(context.Background(), "claim-4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v1 != agentsdk.VerdictTrue {
		t.Fatalf("expected VerdictTrue, got %s", v1)
	}

	// Second call (different claim) should be rate limited → uncertain.
	v2, conf2, err := eval.Evaluate(context.Background(), "claim-5-nonexist")
	if err != nil {
		t.Fatalf("expected no error when rate limited, got: %v", err)
	}
	if v2 != agentsdk.VerdictUncertain {
		t.Fatalf("expected VerdictUncertain when rate limited, got %s", v2)
	}
	if conf2 != 10 {
		t.Fatalf("expected confidence 10 when rate limited, got %d", conf2)
	}
}

func TestClaudeEvaluator_Determinism(t *testing.T) {
	callCount := 0
	mock := &mockMessageSender{
		response: mockEvalResponse("true", 88, "verified"),
	}
	// Wrap to track calls precisely.
	wrapper := &mockMessageSender{}
	wrapper.response = mock.response

	fetcher := &mockClaimFetcher{
		claims: map[string]*agentsdk.ClaimInfo{
			"claim-det": {ID: "claim-det", Text: "Deterministic claim for commit-reveal test", Domain: "math", Category: "formal"},
		},
	}

	eval := NewClaudeEvaluatorWithClient(wrapper, fetcher, "")

	// Simulate commit phase.
	v1, c1, err := eval.Evaluate(context.Background(), "claim-det")
	if err != nil {
		t.Fatalf("commit phase error: %v", err)
	}
	callCount = wrapper.calls

	// Simulate reveal phase — MUST return same verdict.
	v2, c2, err := eval.Evaluate(context.Background(), "claim-det")
	if err != nil {
		t.Fatalf("reveal phase error: %v", err)
	}

	if v1 != v2 || c1 != c2 {
		t.Fatalf("verdict not deterministic: commit=%s/%d, reveal=%s/%d", v1, c1, v2, c2)
	}
	if wrapper.calls != callCount {
		t.Fatalf("reveal phase should use cache, not API: calls went from %d to %d", callCount, wrapper.calls)
	}
}

func TestClaudeEvaluator_ClaimNotFound(t *testing.T) {
	mock := &mockMessageSender{
		response: mockEvalResponse("true", 90, "ok"),
	}
	fetcher := &mockClaimFetcher{
		claims: map[string]*agentsdk.ClaimInfo{}, // empty
	}

	eval := NewClaudeEvaluatorWithClient(mock, fetcher, "")

	// Missing claim should degrade gracefully.
	verdict, conf, err := eval.Evaluate(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("expected no error for missing claim, got: %v", err)
	}
	if verdict != agentsdk.VerdictUncertain {
		t.Fatalf("expected VerdictUncertain for missing claim, got %s", verdict)
	}
	if conf != 5 {
		t.Fatalf("expected confidence 5 for missing claim, got %d", conf)
	}
	if mock.calls != 0 {
		t.Fatalf("should not call API when claim is missing, got %d calls", mock.calls)
	}
}

// --- Subscription Manager Tests ---

func TestSubscriptionManager_Limits(t *testing.T) {
	sm := NewSubscriptionManager(2, 1000)

	// First two requests should be allowed.
	if err := sm.CanRequest(100); err != nil {
		t.Fatalf("first request should be allowed: %v", err)
	}
	sm.RecordUsage(100, 50)

	if err := sm.CanRequest(100); err != nil {
		t.Fatalf("second request should be allowed: %v", err)
	}
	sm.RecordUsage(100, 50)

	// Third request should be rate limited.
	if err := sm.CanRequest(100); err != ErrRateLimited {
		t.Fatalf("expected ErrRateLimited, got: %v", err)
	}
}

func TestSubscriptionManager_BudgetExceeded(t *testing.T) {
	sm := NewSubscriptionManager(0, 500) // No rate limit, tight budget.

	sm.RecordUsage(300, 150) // 450 total

	// Next request would push over budget.
	if err := sm.CanRequest(100); err != ErrBudgetExceeded {
		t.Fatalf("expected ErrBudgetExceeded, got: %v", err)
	}
}

func TestSubscriptionManager_WindowReset(t *testing.T) {
	sm := NewSubscriptionManager(0, 100) // No rate limit, tight budget.

	// Override clock.
	baseTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	sm.now = func() time.Time { return baseTime }
	sm.windowStart = baseTime

	sm.RecordUsage(50, 50) // 100 total, at budget

	// Request should be denied.
	if err := sm.CanRequest(10); err != ErrBudgetExceeded {
		t.Fatalf("expected ErrBudgetExceeded, got: %v", err)
	}

	// Advance clock past 24h — window resets.
	sm.now = func() time.Time { return baseTime.Add(25 * time.Hour) }
	if err := sm.CanRequest(10); err != nil {
		t.Fatalf("expected success after window reset, got: %v", err)
	}
}

func TestSubscriptionManager_Usage(t *testing.T) {
	sm := NewSubscriptionManager(100, 0)
	sm.RecordUsage(500, 50)
	sm.RecordUsage(300, 100)

	stats := sm.Usage()
	if stats.InputTokensUsed != 800 {
		t.Fatalf("expected 800 input tokens, got %d", stats.InputTokensUsed)
	}
	if stats.OutputTokensUsed != 150 {
		t.Fatalf("expected 150 output tokens, got %d", stats.OutputTokensUsed)
	}
	if stats.RequestCount != 2 {
		t.Fatalf("expected 2 requests, got %d", stats.RequestCount)
	}
}

func TestSubscriptionManager_ResetWindow(t *testing.T) {
	sm := NewSubscriptionManager(100, 0)
	sm.RecordUsage(500, 50)
	sm.ResetWindow()

	stats := sm.Usage()
	if stats.InputTokensUsed != 0 {
		t.Fatalf("expected 0 after reset, got %d", stats.InputTokensUsed)
	}
	if stats.RequestCount != 0 {
		t.Fatalf("expected 0 requests after reset, got %d", stats.RequestCount)
	}
}

// --- Verdict Cache Tests ---

func TestVerdictCache_GetSet(t *testing.T) {
	vc := NewVerdictCache("")

	vc.Set("claim-a", agentsdk.VerdictTrue, 95)

	cached, ok := vc.Get("claim-a")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if cached.Verdict != agentsdk.VerdictTrue {
		t.Fatalf("expected VerdictTrue, got %s", cached.Verdict)
	}
	if cached.Confidence != 95 {
		t.Fatalf("expected confidence 95, got %d", cached.Confidence)
	}
}

func TestVerdictCache_Expiry(t *testing.T) {
	vc := NewVerdictCache("")
	vc.SetTTL(1 * time.Second)

	baseTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	vc.now = func() time.Time { return baseTime }

	vc.Set("claim-exp", agentsdk.VerdictFalse, 70)

	// Still valid.
	if _, ok := vc.Get("claim-exp"); !ok {
		t.Fatal("expected cache hit before expiry")
	}

	// Advance past TTL.
	vc.now = func() time.Time { return baseTime.Add(2 * time.Second) }
	if _, ok := vc.Get("claim-exp"); ok {
		t.Fatal("expected cache miss after expiry")
	}
}

func TestVerdictCache_Persistence(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "verdict-cache")

	// Create cache and add entry.
	vc1 := NewVerdictCache(dir)
	vc1.Set("claim-persist", agentsdk.VerdictTrue, 88)

	// Verify file was created.
	path := filepath.Join(dir, "claim-persist.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("cache file not created: %v", err)
	}

	// New cache instance should load from disk.
	vc2 := NewVerdictCache(dir)
	cached, ok := vc2.Get("claim-persist")
	if !ok {
		t.Fatal("expected to load cached verdict from disk")
	}
	if cached.Verdict != agentsdk.VerdictTrue || cached.Confidence != 88 {
		t.Fatalf("loaded verdict mismatch: %s/%d", cached.Verdict, cached.Confidence)
	}
}

func TestVerdictCache_Cleanup(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "verdict-cache")
	vc := NewVerdictCache(dir)
	vc.SetTTL(1 * time.Second)

	baseTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	vc.now = func() time.Time { return baseTime }

	vc.Set("claim-old", agentsdk.VerdictTrue, 80)
	vc.Set("claim-new", agentsdk.VerdictFalse, 60)

	if vc.Size() != 2 {
		t.Fatalf("expected 2 entries, got %d", vc.Size())
	}

	// Advance time past TTL of the entries.
	vc.now = func() time.Time { return baseTime.Add(2 * time.Second) }
	vc.Cleanup()

	if vc.Size() != 0 {
		t.Fatalf("expected 0 entries after cleanup, got %d", vc.Size())
	}

	// Verify disk files are cleaned.
	if _, err := os.Stat(filepath.Join(dir, "claim-old.json")); !os.IsNotExist(err) {
		t.Fatal("expected claim-old.json to be removed")
	}
}

func TestVerdictCache_Clear(t *testing.T) {
	vc := NewVerdictCache("")
	vc.Set("a", agentsdk.VerdictTrue, 90)
	vc.Set("b", agentsdk.VerdictFalse, 50)
	vc.Clear()

	if vc.Size() != 0 {
		t.Fatalf("expected empty cache after clear, got %d", vc.Size())
	}
}

// --- Prompt Building Tests ---

func TestBuildEvaluationPrompt_Basic(t *testing.T) {
	prompt := BuildEvaluationPrompt(ClaimContext{
		ClaimID:  "claim-123",
		Text:     "The speed of light is approximately 3x10^8 m/s",
		Domain:   "physics",
		Category: "empirical",
	})

	if !contains(prompt, "claim-123") {
		t.Fatal("prompt should contain claim ID")
	}
	if !contains(prompt, "speed of light") {
		t.Fatal("prompt should contain claim text")
	}
	if !contains(prompt, "DOMAIN: physics") {
		t.Fatal("prompt should contain domain")
	}
	if !contains(prompt, "empirical") {
		t.Fatal("prompt should contain category guidance")
	}
}

func TestBuildEvaluationPrompt_Categories(t *testing.T) {
	categories := []struct {
		cat      string
		expected string
	}{
		{"analytic", "logical validity"},
		{"empirical", "quality of evidence"},
		{"formal", "proof structure"},
		{"predictive", "methodology"},
		{"computational", "correctness"},
	}

	for _, tc := range categories {
		prompt := BuildEvaluationPrompt(ClaimContext{
			ClaimID:  "c-1",
			Text:     "test claim",
			Category: tc.cat,
		})
		if !contains(prompt, tc.expected) {
			t.Fatalf("category %q prompt should mention %q", tc.cat, tc.expected)
		}
	}
}

func TestBuildEvaluationPrompt_RelatedFacts(t *testing.T) {
	prompt := BuildEvaluationPrompt(ClaimContext{
		ClaimID:      "c-2",
		Text:         "follow-up claim",
		RelatedFacts: []string{"Fact 1: base claim", "Fact 2: supporting evidence"},
	})

	if !contains(prompt, "RELATED ACCEPTED FACTS") {
		t.Fatal("prompt should include related facts section")
	}
	if !contains(prompt, "Fact 1: base claim") {
		t.Fatal("prompt should include first related fact")
	}
}

// --- Response Parsing Tests ---

func TestParseEvaluationResponse_Valid(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		verdict    agentsdk.Verdict
		confidence uint32
	}{
		{
			"true verdict",
			`{"verdict": "true", "confidence": 85, "reasoning": "well supported"}`,
			agentsdk.VerdictTrue, 85,
		},
		{
			"false verdict",
			`{"verdict": "false", "confidence": 92, "reasoning": "contradicts evidence"}`,
			agentsdk.VerdictFalse, 92,
		},
		{
			"uncertain verdict",
			`{"verdict": "uncertain", "confidence": 30, "reasoning": "insufficient data"}`,
			agentsdk.VerdictUncertain, 30,
		},
		{
			"with code fences",
			"```json\n{\"verdict\": \"true\", \"confidence\": 70, \"reasoning\": \"ok\"}\n```",
			agentsdk.VerdictTrue, 70,
		},
		{
			"confidence clamped to 100",
			`{"verdict": "true", "confidence": 150, "reasoning": "overcounted"}`,
			agentsdk.VerdictTrue, 100,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v, c, err := parseEvaluationResponse(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if v != tc.verdict {
				t.Fatalf("expected %s, got %s", tc.verdict, v)
			}
			if c != tc.confidence {
				t.Fatalf("expected %d, got %d", tc.confidence, c)
			}
		})
	}
}

func TestParseEvaluationResponse_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"not json", "I think this claim is true"},
		{"unknown verdict", `{"verdict": "maybe", "confidence": 50, "reasoning": "idk"}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := parseEvaluationResponse(tc.input)
			if err == nil {
				t.Fatal("expected error for invalid input")
			}
		})
	}
}

// --- API Error Tests ---

func TestAPIError_Types(t *testing.T) {
	rateLimited := &APIError{StatusCode: 429}
	if !rateLimited.IsRateLimited() {
		t.Fatal("expected IsRateLimited() = true for 429")
	}

	overloaded := &APIError{StatusCode: 529}
	if !overloaded.IsOverloaded() {
		t.Fatal("expected IsOverloaded() = true for 529")
	}

	serverErr := &APIError{StatusCode: 500}
	if serverErr.IsRateLimited() || serverErr.IsOverloaded() {
		t.Fatal("500 should not be rate limited or overloaded")
	}
}

func TestMessagesResponse_TextContent(t *testing.T) {
	resp := &MessagesResponse{
		Content: []ContentBlock{
			{Type: "text", Text: "hello "},
			{Type: "text", Text: "world"},
		},
	}
	if resp.TextContent() != "hello world" {
		t.Fatalf("expected 'hello world', got %q", resp.TextContent())
	}

	empty := &MessagesResponse{}
	if empty.TextContent() != "" {
		t.Fatalf("expected empty string, got %q", empty.TextContent())
	}
}

// --- Helpers ---

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
