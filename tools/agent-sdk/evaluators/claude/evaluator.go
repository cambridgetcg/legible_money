package claude

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
)

// ClaimFetcher fetches claim details from the chain. The agent-sdk QueryClient
// does not have a GetClaim method, so this interface allows flexible integration.
// CLIClient-based implementations can shell out to `legbled query knowledge claim <id>`.
type ClaimFetcher interface {
	GetClaim(ctx context.Context, claimID string) (*agentsdk.ClaimInfo, error)
}

// MessageSender is the subset of AnthropicClient used by the evaluator.
// Extracted as an interface for testability.
type MessageSender interface {
	CreateMessage(ctx context.Context, req MessagesRequest) (*MessagesResponse, error)
}

// EvaluatorConfig configures the Claude-backed claim evaluator.
type EvaluatorConfig struct {
	// APIKey is the Anthropic API key. Required.
	APIKey string

	// Model to use. Defaults to claude-sonnet-4-5-20250929.
	Model string

	// MaxTokens per evaluation response. Defaults to 1024.
	MaxTokens int

	// CacheDir for verdict persistence. Empty disables disk caching.
	CacheDir string

	// MaxRequestsPerMin is the per-minute rate limit. 0 = unlimited.
	MaxRequestsPerMin int

	// MaxTokensPerDay is the daily token budget. 0 = unlimited.
	MaxTokensPerDay int64
}

// ClaudeEvaluator implements agentsdk.ClaimEvaluator using the Claude API.
// It caches verdicts for commit-reveal determinism and manages subscription limits.
type ClaudeEvaluator struct {
	client       MessageSender
	subscription *SubscriptionManager
	cache        *VerdictCache
	fetcher      ClaimFetcher

	model     string
	maxTokens int
}

// Compile-time check: ClaudeEvaluator implements ClaimEvaluator.
var _ agentsdk.ClaimEvaluator = (*ClaudeEvaluator)(nil)

// NewClaudeEvaluator creates an evaluator from config and a claim fetcher.
func NewClaudeEvaluator(cfg EvaluatorConfig, fetcher ClaimFetcher) *ClaudeEvaluator {
	model := cfg.Model
	if model == "" {
		model = defaultModel
	}
	maxTokens := cfg.MaxTokens
	if maxTokens == 0 {
		maxTokens = defaultMaxTokens
	}

	client := NewAnthropicClient(cfg.APIKey)
	client.Model = model

	return &ClaudeEvaluator{
		client:       client,
		subscription: NewSubscriptionManager(cfg.MaxRequestsPerMin, cfg.MaxTokensPerDay),
		cache:        NewVerdictCache(cfg.CacheDir),
		fetcher:      fetcher,
		model:        model,
		maxTokens:    maxTokens,
	}
}

// NewClaudeEvaluatorWithClient creates an evaluator with a custom MessageSender.
// Used for testing with mock API clients.
func NewClaudeEvaluatorWithClient(client MessageSender, fetcher ClaimFetcher, cacheDir string) *ClaudeEvaluator {
	return &ClaudeEvaluator{
		client:       client,
		subscription: NewSubscriptionManager(0, 0),
		cache:        NewVerdictCache(cacheDir),
		fetcher:      fetcher,
		model:        defaultModel,
		maxTokens:    defaultMaxTokens,
	}
}

// Evaluate determines the verdict and confidence for a claim.
// This is the core method satisfying agentsdk.ClaimEvaluator.
//
// Flow:
//  1. Check cache (determinism for reveal phase)
//  2. Check subscription limits
//  3. Fetch claim from chain via fetcher
//  4. Build evaluation prompt
//  5. Call Claude API
//  6. Parse structured response
//  7. Cache verdict
//  8. Record usage
func (ce *ClaudeEvaluator) Evaluate(ctx context.Context, claimID string) (agentsdk.Verdict, uint32, error) {
	// 1. Cache hit = deterministic verdict for reveal phase.
	if cached, ok := ce.cache.Get(claimID); ok {
		return cached.Verdict, cached.Confidence, nil
	}

	// 2. Check subscription limits before making an API call.
	// Estimate ~2000 tokens per evaluation (prompt + response).
	if err := ce.subscription.CanRequest(2000); err != nil {
		if errors.Is(err, ErrRateLimited) || errors.Is(err, ErrBudgetExceeded) {
			// Degrade gracefully: return uncertain with low confidence.
			return agentsdk.VerdictUncertain, 10, nil
		}
		return "", 0, fmt.Errorf("subscription check: %w", err)
	}

	// 3. Fetch claim details from chain.
	claim, err := ce.fetcher.GetClaim(ctx, claimID)
	if err != nil {
		return agentsdk.VerdictUncertain, 5, nil
	}

	// 4. Build evaluation prompt.
	claimCtx := ClaimContext{
		ClaimID:  claimID,
		Text:     claim.Text,
		Domain:   claim.Domain,
		Category: claim.Category,
	}
	userPrompt := BuildEvaluationPrompt(claimCtx)

	// 5. Call Claude API.
	resp, err := ce.client.CreateMessage(ctx, MessagesRequest{
		Model:     ce.model,
		MaxTokens: ce.maxTokens,
		System:    SystemPrompt(),
		Messages: []Message{
			{Role: "user", Content: userPrompt},
		},
	})
	if err != nil {
		// API errors degrade to uncertain — never crash the verification loop.
		return agentsdk.VerdictUncertain, 5, nil
	}

	// 6. Parse structured response.
	verdict, confidence, err := parseEvaluationResponse(resp.TextContent())
	if err != nil {
		return agentsdk.VerdictUncertain, 5, nil
	}

	// 7. Cache the verdict for commit-reveal determinism.
	ce.cache.Set(claimID, verdict, confidence)

	// 8. Record usage.
	ce.subscription.RecordUsage(resp.Usage.InputTokens, resp.Usage.OutputTokens)

	return verdict, confidence, nil
}

// Cache returns the verdict cache for inspection or manual management.
func (ce *ClaudeEvaluator) Cache() *VerdictCache {
	return ce.cache
}

// Subscription returns the subscription manager for usage stats.
func (ce *ClaudeEvaluator) Subscription() *SubscriptionManager {
	return ce.subscription
}

// evaluationResult is the expected JSON structure from Claude.
type evaluationResult struct {
	Verdict    string `json:"verdict"`
	Confidence int    `json:"confidence"`
	Reasoning  string `json:"reasoning"`
}

// parseEvaluationResponse extracts verdict and confidence from Claude's JSON response.
func parseEvaluationResponse(text string) (agentsdk.Verdict, uint32, error) {
	// Claude may wrap JSON in markdown code fences.
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	var result evaluationResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return "", 0, fmt.Errorf("parsing evaluation response: %w", err)
	}

	verdict, err := mapVerdict(result.Verdict)
	if err != nil {
		return "", 0, err
	}

	// Clamp confidence to [0, 100].
	confidence := result.Confidence
	if confidence < 0 {
		confidence = 0
	}
	if confidence > 100 {
		confidence = 100
	}

	return verdict, uint32(confidence), nil
}

// mapVerdict converts the string verdict from Claude to an agentsdk.Verdict.
func mapVerdict(v string) (agentsdk.Verdict, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true":
		return agentsdk.VerdictTrue, nil
	case "false":
		return agentsdk.VerdictFalse, nil
	case "uncertain":
		return agentsdk.VerdictUncertain, nil
	default:
		return "", fmt.Errorf("unknown verdict: %q", v)
	}
}
