// Package bugreport provides a specialized ClaimEvaluator for protocol bug reports.
// It wraps the Claude evaluator with bug-specific prompt engineering and severity
// classification, delegating non-bug claims to the underlying evaluator.
package bugreport

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
	"github.com/legible-money/legible/tools/agent-sdk/evaluators/claude"
)

// CodeContextProvider fetches relevant source code for bug analysis.
// Implementations may read from local files, git, or a code search index.
type CodeContextProvider interface {
	// CodeForClaim returns source code snippets relevant to a claim's text.
	// Returns empty string if no relevant context is available.
	CodeForClaim(ctx context.Context, claimText string) (string, error)
}

// BugReportEvaluator evaluates protocol_observation claims as bug reports.
// For claims in other categories, it delegates to the wrapped Claude evaluator.
type BugReportEvaluator struct {
	delegate agentsdk.ClaimEvaluator
	fetcher  claude.ClaimFetcher
	sender   claude.MessageSender
	codeCtx  CodeContextProvider
	cache    *claude.VerdictCache
	model    string
	maxTok   int

	// queryClient fetches known bugs for dedup checks.
	queryClient agentsdk.QueryClient
}

// Compile-time check: BugReportEvaluator implements ClaimEvaluator.
var _ agentsdk.ClaimEvaluator = (*BugReportEvaluator)(nil)

// Config holds configuration for a BugReportEvaluator.
type Config struct {
	// Delegate is the base evaluator for non-bug claims. Required.
	Delegate agentsdk.ClaimEvaluator

	// Fetcher retrieves claim details from the chain. Required.
	Fetcher claude.ClaimFetcher

	// Sender sends messages to the Claude API. Required.
	Sender claude.MessageSender

	// CodeCtx provides source code context for bug analysis. Optional.
	CodeCtx CodeContextProvider

	// QueryClient for fetching known bugs. Optional.
	QueryClient agentsdk.QueryClient

	// CacheDir for verdict persistence. Empty disables disk caching.
	CacheDir string

	// Model to use. Defaults to claude-sonnet-4-5-20250929.
	Model string

	// MaxTokens per evaluation response. Defaults to 1024.
	MaxTokens int
}

// New creates a BugReportEvaluator from the given config.
func New(cfg Config) *BugReportEvaluator {
	model := cfg.Model
	if model == "" {
		model = "claude-sonnet-4-5-20250929"
	}
	maxTok := cfg.MaxTokens
	if maxTok == 0 {
		maxTok = 1024
	}

	return &BugReportEvaluator{
		delegate:    cfg.Delegate,
		fetcher:     cfg.Fetcher,
		sender:      cfg.Sender,
		codeCtx:     cfg.CodeCtx,
		cache:       claude.NewVerdictCache(cfg.CacheDir),
		model:       model,
		maxTok:      maxTok,
		queryClient: cfg.QueryClient,
	}
}

// Evaluate determines the verdict and confidence for a claim.
// For protocol_observation claims, applies bug-specific evaluation.
// All other categories are delegated to the wrapped evaluator.
func (e *BugReportEvaluator) Evaluate(ctx context.Context, claimID string) (agentsdk.Verdict, uint32, error) {
	// Cache hit = deterministic verdict for commit-reveal.
	if cached, ok := e.cache.Get(claimID); ok {
		return cached.Verdict, cached.Confidence, nil
	}

	// Fetch the claim to check its category.
	claim, err := e.fetcher.GetClaim(ctx, claimID)
	if err != nil {
		return agentsdk.VerdictUncertain, 5, nil
	}

	// Delegate non-bug claims to the wrapped evaluator.
	if claim.Category != "protocol_observation" {
		return e.delegate.Evaluate(ctx, claimID)
	}

	// Build bug-specific evaluation context.
	reportCtx := BugReportContext{
		ClaimID:  claimID,
		Text:     claim.Text,
		Domain:   claim.Domain,
		Category: claim.Category,
	}

	// Fetch code context if provider is available.
	if e.codeCtx != nil {
		code, codeErr := e.codeCtx.CodeForClaim(ctx, claim.Text)
		if codeErr == nil && code != "" {
			reportCtx.CodeContext = code
		}
	}

	// Fetch known bugs for dedup checking.
	if e.queryClient != nil {
		facts, qErr := e.queryClient.FactsByDomain(ctx, claim.Domain)
		if qErr == nil {
			for _, f := range facts {
				if f.Category == "protocol_observation" {
					reportCtx.KnownBugs = append(reportCtx.KnownBugs, f.ClaimText)
				}
			}
		}
	}

	// Build prompt and call Claude.
	userPrompt := BuildBugReportPrompt(reportCtx)
	resp, err := e.sender.CreateMessage(ctx, claude.MessagesRequest{
		Model:     e.model,
		MaxTokens: e.maxTok,
		System:    BugReportSystemPrompt(),
		Messages: []claude.Message{
			{Role: "user", Content: userPrompt},
		},
	})
	if err != nil {
		// API errors degrade to uncertain.
		return agentsdk.VerdictUncertain, 5, nil
	}

	// Parse bug-specific response (includes severity).
	verdict, confidence, _, err := parseBugReportResponse(resp.TextContent())
	if err != nil {
		return agentsdk.VerdictUncertain, 5, nil
	}

	// Cache for commit-reveal determinism.
	e.cache.Set(claimID, verdict, confidence)

	return verdict, confidence, nil
}

// Cache returns the verdict cache for inspection.
func (e *BugReportEvaluator) Cache() *claude.VerdictCache {
	return e.cache
}

// bugReportResult is the expected JSON structure from Claude for bug reports.
type bugReportResult struct {
	Verdict    string `json:"verdict"`
	Confidence int    `json:"confidence"`
	Severity   string `json:"severity"`
	Reasoning  string `json:"reasoning"`
}

// parseBugReportResponse extracts verdict, confidence, and severity from Claude's response.
func parseBugReportResponse(text string) (agentsdk.Verdict, uint32, Severity, error) {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	var result bugReportResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return "", 0, "", fmt.Errorf("parsing bug report response: %w", err)
	}

	verdict, err := mapVerdict(result.Verdict)
	if err != nil {
		return "", 0, "", err
	}

	// Clamp confidence to [0, 100].
	confidence := result.Confidence
	if confidence < 0 {
		confidence = 0
	}
	if confidence > 100 {
		confidence = 100
	}

	severity := Severity(strings.ToLower(result.Severity))
	if !severity.Valid() {
		severity = SeverityMedium // default fallback
	}

	return verdict, uint32(confidence), severity, nil
}

// mapVerdict converts the string verdict to an agentsdk.Verdict.
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
