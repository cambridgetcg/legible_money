package bugreport

import (
	"context"
	"fmt"
	"testing"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
	"github.com/legible-money/legible/tools/agent-sdk/evaluators/claude"
)

// --- Mocks ---

type mockMessageSender struct {
	response *claude.MessagesResponse
	err      error
	calls    int
}

func (m *mockMessageSender) CreateMessage(_ context.Context, _ claude.MessagesRequest) (*claude.MessagesResponse, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

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

type mockDelegate struct {
	verdict    agentsdk.Verdict
	confidence uint32
	err        error
	calls      int
}

func (m *mockDelegate) Evaluate(_ context.Context, _ string) (agentsdk.Verdict, uint32, error) {
	m.calls++
	return m.verdict, m.confidence, m.err
}

type mockCodeCtx struct {
	code string
	err  error
}

func (m *mockCodeCtx) CodeForClaim(_ context.Context, _ string) (string, error) {
	return m.code, m.err
}

func mockBugResponse(verdict string, confidence int, severity, reasoning string) *claude.MessagesResponse {
	text := fmt.Sprintf(`{"verdict": %q, "confidence": %d, "severity": %q, "reasoning": %q}`,
		verdict, confidence, severity, reasoning)
	return &claude.MessagesResponse{
		ID:   "msg_test",
		Role: "assistant",
		Content: []claude.ContentBlock{
			{Type: "text", Text: text},
		},
		Usage: claude.Usage{InputTokens: 500, OutputTokens: 50},
	}
}

// --- Tests ---

func TestBugReportEvaluator_RealBug(t *testing.T) {
	sender := &mockMessageSender{
		response: mockBugResponse("true", 85, "high", "Validator slashing applies penalty twice"),
	}
	fetcher := &mockClaimFetcher{
		claims: map[string]*agentsdk.ClaimInfo{
			"bug-1": {
				ID:       "bug-1",
				Text:     "BUG: validator slashing applies penalty twice when slash is called during unbonding",
				Domain:   "protocol",
				Category: "protocol_observation",
			},
		},
	}
	delegate := &mockDelegate{verdict: agentsdk.VerdictTrue, confidence: 50}

	eval := New(Config{
		Delegate: delegate,
		Fetcher:  fetcher,
		Sender:   sender,
	})

	verdict, conf, err := eval.Evaluate(context.Background(), "bug-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if verdict != agentsdk.VerdictTrue {
		t.Fatalf("expected VerdictTrue, got %s", verdict)
	}
	if conf != 85 {
		t.Fatalf("expected confidence 85, got %d", conf)
	}
	// Should NOT have delegated — this is a protocol_observation.
	if delegate.calls != 0 {
		t.Fatalf("expected 0 delegate calls for protocol_observation, got %d", delegate.calls)
	}
	if sender.calls != 1 {
		t.Fatalf("expected 1 sender call, got %d", sender.calls)
	}
}

func TestBugReportEvaluator_DelegatesNonBug(t *testing.T) {
	sender := &mockMessageSender{
		response: mockBugResponse("true", 90, "medium", "should not be called"),
	}
	fetcher := &mockClaimFetcher{
		claims: map[string]*agentsdk.ClaimInfo{
			"claim-normal": {
				ID:       "claim-normal",
				Text:     "Water boils at 100C at sea level",
				Domain:   "physics",
				Category: "empirical",
			},
		},
	}
	delegate := &mockDelegate{verdict: agentsdk.VerdictTrue, confidence: 92}

	eval := New(Config{
		Delegate: delegate,
		Fetcher:  fetcher,
		Sender:   sender,
	})

	verdict, conf, err := eval.Evaluate(context.Background(), "claim-normal")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if verdict != agentsdk.VerdictTrue {
		t.Fatalf("expected VerdictTrue from delegate, got %s", verdict)
	}
	if conf != 92 {
		t.Fatalf("expected confidence 92 from delegate, got %d", conf)
	}
	// Should have delegated, not called sender directly.
	if delegate.calls != 1 {
		t.Fatalf("expected 1 delegate call, got %d", delegate.calls)
	}
	if sender.calls != 0 {
		t.Fatalf("expected 0 sender calls for non-bug claim, got %d", sender.calls)
	}
}

func TestBugReportEvaluator_CacheHit(t *testing.T) {
	sender := &mockMessageSender{
		response: mockBugResponse("true", 80, "critical", "double slash confirmed"),
	}
	fetcher := &mockClaimFetcher{
		claims: map[string]*agentsdk.ClaimInfo{
			"bug-cache": {
				ID:       "bug-cache",
				Text:     "BUG: double slash in keeper.go",
				Domain:   "protocol",
				Category: "protocol_observation",
			},
		},
	}
	delegate := &mockDelegate{}

	eval := New(Config{
		Delegate: delegate,
		Fetcher:  fetcher,
		Sender:   sender,
	})

	// First call: should hit API.
	v1, c1, err := eval.Evaluate(context.Background(), "bug-cache")
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}
	if sender.calls != 1 {
		t.Fatalf("expected 1 API call, got %d", sender.calls)
	}

	// Second call: should return cached verdict.
	v2, c2, err := eval.Evaluate(context.Background(), "bug-cache")
	if err != nil {
		t.Fatalf("cache hit error: %v", err)
	}
	if v1 != v2 || c1 != c2 {
		t.Fatalf("cache mismatch: %s/%d vs %s/%d", v1, c1, v2, c2)
	}
	if sender.calls != 1 {
		t.Fatalf("expected still 1 API call after cache hit, got %d", sender.calls)
	}
}

func TestBugReportEvaluator_ClaimNotFound(t *testing.T) {
	sender := &mockMessageSender{}
	fetcher := &mockClaimFetcher{claims: map[string]*agentsdk.ClaimInfo{}}
	delegate := &mockDelegate{}

	eval := New(Config{
		Delegate: delegate,
		Fetcher:  fetcher,
		Sender:   sender,
	})

	verdict, conf, err := eval.Evaluate(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("expected no error for missing claim, got: %v", err)
	}
	if verdict != agentsdk.VerdictUncertain {
		t.Fatalf("expected VerdictUncertain, got %s", verdict)
	}
	if conf != 5 {
		t.Fatalf("expected confidence 5, got %d", conf)
	}
}

func TestBugReportEvaluator_APIError(t *testing.T) {
	sender := &mockMessageSender{
		err: fmt.Errorf("API unavailable"),
	}
	fetcher := &mockClaimFetcher{
		claims: map[string]*agentsdk.ClaimInfo{
			"bug-err": {
				ID:       "bug-err",
				Text:     "BUG: some protocol issue worth investigating",
				Domain:   "protocol",
				Category: "protocol_observation",
			},
		},
	}
	delegate := &mockDelegate{}

	eval := New(Config{
		Delegate: delegate,
		Fetcher:  fetcher,
		Sender:   sender,
	})

	verdict, conf, err := eval.Evaluate(context.Background(), "bug-err")
	if err != nil {
		t.Fatalf("expected no error on API failure, got: %v", err)
	}
	if verdict != agentsdk.VerdictUncertain {
		t.Fatalf("expected VerdictUncertain, got %s", verdict)
	}
	if conf != 5 {
		t.Fatalf("expected confidence 5, got %d", conf)
	}
}

func TestBugReportEvaluator_WithCodeContext(t *testing.T) {
	sender := &mockMessageSender{
		response: mockBugResponse("true", 90, "critical", "confirmed via code context"),
	}
	fetcher := &mockClaimFetcher{
		claims: map[string]*agentsdk.ClaimInfo{
			"bug-code": {
				ID:       "bug-code",
				Text:     "BUG: overflow in calculateReward at staking/keeper.go:340",
				Domain:   "protocol",
				Category: "protocol_observation",
			},
		},
	}
	codeProvider := &mockCodeCtx{
		code: "func calculateReward(stake uint64) uint64 { return stake * 2 }",
	}
	delegate := &mockDelegate{}

	eval := New(Config{
		Delegate: delegate,
		Fetcher:  fetcher,
		Sender:   sender,
		CodeCtx:  codeProvider,
	})

	verdict, conf, err := eval.Evaluate(context.Background(), "bug-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if verdict != agentsdk.VerdictTrue {
		t.Fatalf("expected VerdictTrue, got %s", verdict)
	}
	if conf != 90 {
		t.Fatalf("expected confidence 90, got %d", conf)
	}
}

func TestBugReportEvaluator_FalseBug(t *testing.T) {
	sender := &mockMessageSender{
		response: mockBugResponse("false", 75, "low", "describes expected behavior"),
	}
	fetcher := &mockClaimFetcher{
		claims: map[string]*agentsdk.ClaimInfo{
			"bug-false": {
				ID:       "bug-false",
				Text:     "BUG: validator loses stake when misbehaving",
				Domain:   "protocol",
				Category: "protocol_observation",
			},
		},
	}
	delegate := &mockDelegate{}

	eval := New(Config{
		Delegate: delegate,
		Fetcher:  fetcher,
		Sender:   sender,
	})

	verdict, _, err := eval.Evaluate(context.Background(), "bug-false")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if verdict != agentsdk.VerdictFalse {
		t.Fatalf("expected VerdictFalse, got %s", verdict)
	}
}

// --- Severity Tests ---

func TestSeverity_BountyMultiplier(t *testing.T) {
	tests := []struct {
		severity   Severity
		multiplier uint32
	}{
		{SeverityCritical, 400},
		{SeverityHigh, 200},
		{SeverityMedium, 100},
		{SeverityLow, 50},
		{Severity("unknown"), 100},
	}
	for _, tc := range tests {
		if got := tc.severity.BountyMultiplier(); got != tc.multiplier {
			t.Fatalf("severity %q: expected multiplier %d, got %d", tc.severity, tc.multiplier, got)
		}
	}
}

func TestSeverity_Valid(t *testing.T) {
	for _, s := range []Severity{SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow} {
		if !s.Valid() {
			t.Fatalf("expected %q to be valid", s)
		}
	}
	if Severity("invalid").Valid() {
		t.Fatal("expected 'invalid' to be invalid")
	}
}

// --- Prompt Tests ---

func TestBuildBugReportPrompt_Basic(t *testing.T) {
	prompt := BuildBugReportPrompt(BugReportContext{
		ClaimID: "bug-123",
		Text:    "BUG: double slash in validator staking",
		Domain:  "protocol",
	})

	if !containsStr(prompt, "bug-123") {
		t.Fatal("prompt should contain claim ID")
	}
	if !containsStr(prompt, "double slash") {
		t.Fatal("prompt should contain claim text")
	}
	if !containsStr(prompt, "DOMAIN: protocol") {
		t.Fatal("prompt should contain domain")
	}
}

func TestBuildBugReportPrompt_WithCodeContext(t *testing.T) {
	prompt := BuildBugReportPrompt(BugReportContext{
		ClaimID:     "bug-456",
		Text:        "BUG: overflow in reward calc",
		CodeContext: "func calcReward() { ... }",
	})

	if !containsStr(prompt, "RELEVANT CODE CONTEXT") {
		t.Fatal("prompt should include code context section")
	}
	if !containsStr(prompt, "func calcReward") {
		t.Fatal("prompt should include code snippet")
	}
}

func TestBuildBugReportPrompt_KnownBugs(t *testing.T) {
	prompt := BuildBugReportPrompt(BugReportContext{
		ClaimID:    "bug-789",
		Text:       "BUG: new issue",
		KnownBugs:  []string{"Known: slash applied twice", "Known: nonce overflow"},
	})

	if !containsStr(prompt, "PREVIOUSLY VERIFIED BUGS") {
		t.Fatal("prompt should include known bugs section")
	}
	if !containsStr(prompt, "slash applied twice") {
		t.Fatal("prompt should include known bug text")
	}
	if !containsStr(prompt, "duplicates a known bug") {
		t.Fatal("prompt should warn about duplicates")
	}
}

// --- Response Parsing Tests ---

func TestParseBugReportResponse_Valid(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		verdict    agentsdk.Verdict
		confidence uint32
		severity   Severity
	}{
		{
			"critical bug",
			`{"verdict": "true", "confidence": 95, "severity": "critical", "reasoning": "chain halt"}`,
			agentsdk.VerdictTrue, 95, SeverityCritical,
		},
		{
			"low severity false",
			`{"verdict": "false", "confidence": 80, "severity": "low", "reasoning": "not a bug"}`,
			agentsdk.VerdictFalse, 80, SeverityLow,
		},
		{
			"uncertain",
			`{"verdict": "uncertain", "confidence": 30, "severity": "medium", "reasoning": "needs more info"}`,
			agentsdk.VerdictUncertain, 30, SeverityMedium,
		},
		{
			"with code fences",
			"```json\n{\"verdict\": \"true\", \"confidence\": 70, \"severity\": \"high\", \"reasoning\": \"ok\"}\n```",
			agentsdk.VerdictTrue, 70, SeverityHigh,
		},
		{
			"invalid severity defaults to medium",
			`{"verdict": "true", "confidence": 60, "severity": "extreme", "reasoning": "bad sev"}`,
			agentsdk.VerdictTrue, 60, SeverityMedium,
		},
		{
			"confidence clamped to 100",
			`{"verdict": "true", "confidence": 150, "severity": "high", "reasoning": "overcounted"}`,
			agentsdk.VerdictTrue, 100, SeverityHigh,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v, c, s, err := parseBugReportResponse(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if v != tc.verdict {
				t.Fatalf("expected verdict %s, got %s", tc.verdict, v)
			}
			if c != tc.confidence {
				t.Fatalf("expected confidence %d, got %d", tc.confidence, c)
			}
			if s != tc.severity {
				t.Fatalf("expected severity %s, got %s", tc.severity, s)
			}
		})
	}
}

func TestParseBugReportResponse_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"not json", "I think this is a real bug"},
		{"unknown verdict", `{"verdict": "maybe", "confidence": 50, "severity": "high", "reasoning": "idk"}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, _, err := parseBugReportResponse(tc.input)
			if err == nil {
				t.Fatal("expected error for invalid input")
			}
		})
	}
}

// --- Helpers ---

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
