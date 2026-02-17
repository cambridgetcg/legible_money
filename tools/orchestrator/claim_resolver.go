package orchestrator

import (
	"context"
	"fmt"
	"sync"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
)

// QueryClientClaimResolver resolves claims using an agentsdk.QueryClient.
type QueryClientClaimResolver struct {
	client agentsdk.QueryClient
}

// NewQueryClientClaimResolver creates a resolver backed by a query client.
func NewQueryClientClaimResolver(qc agentsdk.QueryClient) *QueryClientClaimResolver {
	return &QueryClientClaimResolver{client: qc}
}

// ResolveClaim fetches claim info by looking up the round and extracting the claim.
func (r *QueryClientClaimResolver) ResolveClaim(ctx context.Context, claimID string) (*agentsdk.ClaimInfo, error) {
	if r.client == nil {
		return nil, fmt.Errorf("query client not set")
	}

	// Try to find the claim via active rounds first
	rounds, err := r.client.ActiveRounds(ctx)
	if err != nil {
		return nil, fmt.Errorf("querying active rounds: %w", err)
	}

	for _, round := range rounds {
		if round.ClaimID == claimID {
			return &agentsdk.ClaimInfo{
				ID:     claimID,
				Status: round.Phase,
			}, nil
		}
	}

	return nil, fmt.Errorf("claim not found: %s", claimID)
}

// QueryClientFactScanner scans facts using an agentsdk.QueryClient.
type QueryClientFactScanner struct {
	client agentsdk.QueryClient
}

// NewQueryClientFactScanner creates a scanner backed by a query client.
func NewQueryClientFactScanner(qc agentsdk.QueryClient) *QueryClientFactScanner {
	return &QueryClientFactScanner{client: qc}
}

// ScanFacts returns all facts in a domain.
func (s *QueryClientFactScanner) ScanFacts(ctx context.Context, domain string) ([]agentsdk.FactInfo, error) {
	if s.client == nil {
		return nil, fmt.Errorf("query client not set")
	}
	return s.client.FactsByDomain(ctx, domain)
}

// MockClaimResolver is a test-only claim resolver with configurable responses.
type MockClaimResolver struct {
	Claims map[string]*agentsdk.ClaimInfo
	mu     sync.RWMutex
}

// NewMockClaimResolver creates a mock resolver.
func NewMockClaimResolver() *MockClaimResolver {
	return &MockClaimResolver{
		Claims: make(map[string]*agentsdk.ClaimInfo),
	}
}

// AddClaim adds a claim to the mock.
func (m *MockClaimResolver) AddClaim(info agentsdk.ClaimInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Claims[info.ID] = &info
}

// ResolveClaim returns the mock claim or an error.
func (m *MockClaimResolver) ResolveClaim(_ context.Context, claimID string) (*agentsdk.ClaimInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if ci, ok := m.Claims[claimID]; ok {
		return ci, nil
	}
	return nil, fmt.Errorf("mock claim not found: %s", claimID)
}

// MockFactScanner is a test-only fact scanner with configurable responses.
type MockFactScanner struct {
	Facts map[string][]agentsdk.FactInfo // domain → facts
	mu    sync.RWMutex
}

// NewMockFactScanner creates a mock scanner.
func NewMockFactScanner() *MockFactScanner {
	return &MockFactScanner{
		Facts: make(map[string][]agentsdk.FactInfo),
	}
}

// AddFact adds a fact to the mock.
func (m *MockFactScanner) AddFact(domain string, info agentsdk.FactInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Facts[domain] = append(m.Facts[domain], info)
}

// ScanFacts returns the mock facts for a domain.
func (m *MockFactScanner) ScanFacts(_ context.Context, domain string) ([]agentsdk.FactInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Facts[domain], nil
}

// Compile-time checks
var (
	_ ClaimResolver = (*QueryClientClaimResolver)(nil)
	_ ClaimResolver = (*MockClaimResolver)(nil)
	_ FactScanner   = (*QueryClientFactScanner)(nil)
	_ FactScanner   = (*MockFactScanner)(nil)
)
