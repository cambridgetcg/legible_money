package orchestrator

import (
	"context"
	"fmt"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
)

// ClaimStrategy generates claims for a domain.
type ClaimStrategy interface {
	GenerateClaim(ctx context.Context, domain string) (text string, category string, stake uint64, err error)
}

// ClaimTemplate is a pre-defined claim template.
type ClaimTemplate struct {
	Text     string `json:"text"`
	Category string `json:"category"`
	Stake    uint64 `json:"stake"`
}

// TemplateClaimStrategy generates claims from pre-defined templates.
type TemplateClaimStrategy struct {
	Templates map[string][]ClaimTemplate // domain → templates
	cursors   map[string]int             // domain → next index
}

// NewTemplateClaimStrategy creates a template-based strategy.
func NewTemplateClaimStrategy(templates map[string][]ClaimTemplate) *TemplateClaimStrategy {
	return &TemplateClaimStrategy{
		Templates: templates,
		cursors:   make(map[string]int),
	}
}

// GenerateClaim returns the next template for a domain, cycling through them.
func (ts *TemplateClaimStrategy) GenerateClaim(_ context.Context, domain string) (string, string, uint64, error) {
	templates, ok := ts.Templates[domain]
	if !ok || len(templates) == 0 {
		return "", "", 0, fmt.Errorf("no templates for domain %q", domain)
	}

	idx := ts.cursors[domain] % len(templates)
	ts.cursors[domain] = idx + 1

	t := templates[idx]
	return t.Text, t.Category, t.Stake, nil
}

// BrassInstrument implements claim generation (the "brass" section).
type BrassInstrument struct {
	strategy ClaimStrategy
	domains  []string
	minStake uint64
}

// NewBrassInstrument creates a claimer instrument.
func NewBrassInstrument(strategy ClaimStrategy, domains []string, minStake uint64) *BrassInstrument {
	return &BrassInstrument{
		strategy: strategy,
		domains:  domains,
		minStake: minStake,
	}
}

// Name returns the instrument name.
func (b *BrassInstrument) Name() string { return "brass" }

// Section returns the brass section.
func (b *BrassInstrument) Section() SectionName { return SectionBrass }

// PrepareAction generates claim parameters before execution.
func (b *BrassInstrument) PrepareAction(ctx context.Context, agent *agentsdk.Agent, action ActionType, params map[string]string) (map[string]string, error) {
	if action != ActionSubmitClaim {
		return params, nil
	}

	// If params already has text, pass through
	if params["text"] != "" {
		return params, nil
	}

	// Determine target domain
	domain := params["domain"]
	if domain == "" && len(b.domains) > 0 {
		domain = b.domains[0]
	}
	if domain == "" {
		domain = "general"
	}

	// Generate claim
	if b.strategy == nil {
		return nil, fmt.Errorf("no claim strategy configured")
	}

	text, category, stake, err := b.strategy.GenerateClaim(ctx, domain)
	if err != nil {
		return nil, fmt.Errorf("generating claim for domain %q: %w", domain, err)
	}

	// Enforce minimum stake
	if stake < b.minStake {
		stake = b.minStake
	}

	result := copyParams(params)
	result["text"] = text
	result["domain"] = domain
	result["category"] = category
	result["stake"] = fmt.Sprintf("%d", stake)
	return result, nil
}

// Compile-time check
var _ Instrument = (*BrassInstrument)(nil)
