package orchestrator

import (
	"context"
	"fmt"
	"sort"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
)

// ChallengeCandidate represents a fact that may be worth challenging.
type ChallengeCandidate struct {
	FactID string `json:"fact_id"`
	Score  uint64 `json:"score"`  // Higher = more challengeable (integer, no floats)
	Reason string `json:"reason"`
}

// ChallengeDetector scans facts and identifies challenge candidates.
type ChallengeDetector interface {
	DetectChallenges(ctx context.Context, facts []agentsdk.FactInfo) ([]ChallengeCandidate, error)
}

// DefaultChallengeDetector scores facts based on confidence floor and age.
type DefaultChallengeDetector struct {
	// ConfidenceFloor — facts below this confidence are candidates.
	ConfidenceFloor uint32

	// CategoryDecay maps category to half-life in blocks. Categories with
	// shorter half-lives have facts that age faster (higher challenge score).
	CategoryDecay map[string]uint64

	// MinFactAge is the minimum age in blocks before a fact can be challenged.
	MinFactAge uint64

	// CurrentBlock is the current chain height (for age calculation).
	CurrentBlock uint64
}

// NewDefaultChallengeDetector creates a detector with standard parameters.
func NewDefaultChallengeDetector(currentBlock uint64) *DefaultChallengeDetector {
	return &DefaultChallengeDetector{
		ConfidenceFloor: 600_000, // 60% — facts below this are candidates
		CategoryDecay: map[string]uint64{
			"predictive": 20_000,
			"social":     150_000,
			"empirical":  1_000_000,
			"historical": 500_000,
		},
		MinFactAge:   1_000, // ~42 minutes
		CurrentBlock: currentBlock,
	}
}

// DetectChallenges scores facts and returns sorted candidates.
func (d *DefaultChallengeDetector) DetectChallenges(_ context.Context, facts []agentsdk.FactInfo) ([]ChallengeCandidate, error) {
	var candidates []ChallengeCandidate

	for _, fact := range facts {
		// Skip facts above confidence floor
		if fact.Confidence >= d.ConfidenceFloor {
			continue
		}

		// Calculate score: (floor - confidence) + age penalty
		baseScore := uint64(d.ConfidenceFloor - fact.Confidence)

		// Age penalty: older facts in decaying categories score higher
		agePenalty := uint64(0)
		if halfLife, ok := d.CategoryDecay[fact.Category]; ok && halfLife > 0 {
			// FactInfo doesn't carry a start block, so we use a flat age penalty
			// scaled by the category's decay rate. Categories with shorter half-lives
			// get higher age penalties (more urgently challengeable).
			agePenalty = (10_000 * 1000) / halfLife
		}

		score := baseScore + agePenalty
		reason := fmt.Sprintf("confidence %d below floor %d", fact.Confidence, d.ConfidenceFloor)

		candidates = append(candidates, ChallengeCandidate{
			FactID: fact.ID,
			Score:  score,
			Reason: reason,
		})
	}

	// Sort: highest score first, FactID lexicographic tiebreaker
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Score != candidates[j].Score {
			return candidates[i].Score > candidates[j].Score
		}
		return candidates[i].FactID < candidates[j].FactID
	})

	return candidates, nil
}

// PercussionInstrument implements challenge detection (the "percussion" section).
type PercussionInstrument struct {
	detector  ChallengeDetector
	scanner   FactScanner
	domains   []string
	navigator KnowledgeNavigator // optional: enables reference chain analysis
}

// NewPercussionInstrument creates a challenger instrument.
func NewPercussionInstrument(detector ChallengeDetector, scanner FactScanner, domains []string) *PercussionInstrument {
	return &PercussionInstrument{
		detector: detector,
		scanner:  scanner,
		domains:  domains,
	}
}

// SetNavigator sets the optional knowledge navigator for enhanced challenge detection.
func (p *PercussionInstrument) SetNavigator(nav KnowledgeNavigator) {
	p.navigator = nav
}

// Name returns the instrument name.
func (p *PercussionInstrument) Name() string { return "percussion" }

// Section returns the percussion section.
func (p *PercussionInstrument) Section() SectionName { return SectionPercussion }

// PrepareAction scans for challenge candidates and injects parameters.
func (p *PercussionInstrument) PrepareAction(ctx context.Context, agent *agentsdk.Agent, action ActionType, params map[string]string) (map[string]string, error) {
	if action != ActionChallenge {
		return params, nil
	}

	// If fact_id is already specified, pass through
	if params["fact_id"] != "" {
		return params, nil
	}

	// Scan domains for challengeable facts
	domain := params["domain"]
	domains := p.domains
	if domain != "" {
		domains = []string{domain}
	}

	var bestCandidate *ChallengeCandidate
	for _, d := range domains {
		if p.scanner == nil {
			continue
		}
		facts, err := p.scanner.ScanFacts(ctx, d)
		if err != nil {
			continue
		}
		if p.detector == nil {
			continue
		}
		candidates, err := p.detector.DetectChallenges(ctx, facts)
		if err != nil {
			continue
		}
		if len(candidates) > 0 {
			if bestCandidate == nil || candidates[0].Score > bestCandidate.Score {
				c := candidates[0]
				bestCandidate = &c
			}
		}
	}

	if bestCandidate == nil {
		return nil, fmt.Errorf("no challenge candidates found")
	}

	result := copyParams(params)
	result["fact_id"] = bestCandidate.FactID
	result["reason"] = bestCandidate.Reason

	// Set minimum challenge stake if not specified
	if params["stake"] == "" {
		result["stake"] = fmt.Sprintf("%d", MinChallengeStakeUlgm)
	}

	return result, nil
}

// Compile-time checks
var (
	_ Instrument        = (*PercussionInstrument)(nil)
	_ ChallengeDetector = (*DefaultChallengeDetector)(nil)
)
