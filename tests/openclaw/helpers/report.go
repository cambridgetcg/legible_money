//go:build openclaw

package helpers

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// TestReport is the structured JSON output for CI dashboards.
type TestReport struct {
	RunID       string       `json:"run_id"`
	Timestamp   time.Time    `json:"timestamp"`
	Environment string       `json:"environment"`
	ChainID     string       `json:"chain_id"`
	Validators  int          `json:"validators"`
	Duration    string       `json:"duration"`
	Results     []TestResult `json:"results"`
	Summary     Summary      `json:"summary"`
}

// TestResult captures the outcome of a single test probe.
type TestResult struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Phase    string   `json:"phase"`
	Status   string   `json:"status"` // PASS, FAIL, SKIP
	Duration string   `json:"duration"`
	Error    string   `json:"error,omitempty"`
	Insights []string `json:"insights,omitempty"`
}

// Summary aggregates test results.
type Summary struct {
	Total   int `json:"total"`
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}

// Reporter collects test results and writes a JSON report.
type Reporter struct {
	mu         sync.Mutex
	envKind    string
	chainID    string
	validators int
	startTime  time.Time
	results    []TestResult
}

// NewReporter creates a reporter with environment metadata.
func NewReporter(envKind, chainID string, validators int) *Reporter {
	return &Reporter{
		envKind:    envKind,
		chainID:    chainID,
		validators: validators,
		startTime:  time.Now(),
	}
}

// Record adds a test result to the report.
func (r *Reporter) Record(result TestResult) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.results = append(r.results, result)
}

// Finalize writes the JSON report to the given path.
func (r *Reporter) Finalize(path string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	summary := Summary{Total: len(r.results)}
	for _, res := range r.results {
		switch res.Status {
		case "PASS":
			summary.Passed++
		case "FAIL":
			summary.Failed++
		case "SKIP":
			summary.Skipped++
		}
	}

	report := TestReport{
		RunID:       fmt.Sprintf("openclaw-%d", r.startTime.UnixNano()),
		Timestamp:   r.startTime,
		Environment: r.envKind,
		ChainID:     r.chainID,
		Validators:  r.validators,
		Duration:    time.Since(r.startTime).Round(time.Millisecond).String(),
		Results:     r.results,
		Summary:     summary,
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling report: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing report to %s: %w", path, err)
	}
	return nil
}
