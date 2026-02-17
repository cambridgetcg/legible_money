package orchestrator

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
	swarm "github.com/legible-money/legible/tools/agent-swarm"
)

// RehearsalQueryClient is a mock QueryClient for dry-run execution.
// All operations succeed with synthetic data.
type RehearsalQueryClient struct {
	height     atomic.Int64
	balances   map[string]uint64
	balancesMu sync.RWMutex
}

// NewRehearsalQueryClient creates a query client for rehearsal mode.
func NewRehearsalQueryClient() *RehearsalQueryClient {
	r := &RehearsalQueryClient{
		balances: make(map[string]uint64),
	}
	r.height.Store(100) // Start at a reasonable height.
	return r
}

func (r *RehearsalQueryClient) ChainStatus(_ context.Context) (*agentsdk.ChainStatus, error) {
	return &agentsdk.ChainStatus{
		ChainID:        "legible-rehearsal-1",
		LatestHeight:   r.height.Load(),
		LatestTime:     time.Now().Format(time.RFC3339),
		CatchingUp:     false,
		ValidatorCount: 4,
	}, nil
}

func (r *RehearsalQueryClient) Balance(_ context.Context, address string) (uint64, error) {
	r.balancesMu.RLock()
	defer r.balancesMu.RUnlock()
	bal, ok := r.balances[address]
	if !ok {
		return 10_000_000_000, nil // 10k LGM default
	}
	return bal, nil
}

func (r *RehearsalQueryClient) AgentProfile(_ context.Context, address string) (*agentsdk.AgentProfileInfo, error) {
	return &agentsdk.AgentProfileInfo{
		Address:      address,
		DID:          "did:lgm:rehearsal-" + address,
		DisplayName:  "rehearsal-agent",
		Status:       "active",
		Reputation:   500,
		Domains:      []string{"general"},
		Stake:        "1000000000",
		RegisteredAt: 1,
	}, nil
}

func (r *RehearsalQueryClient) ActiveRounds(_ context.Context) ([]agentsdk.RoundInfo, error) {
	return nil, nil
}

func (r *RehearsalQueryClient) RoundDetail(_ context.Context, _ string) (*agentsdk.RoundInfo, error) {
	return &agentsdk.RoundInfo{ID: "rehearsal-round", Phase: "complete"}, nil
}

func (r *RehearsalQueryClient) FactsByDomain(_ context.Context, _ string) ([]agentsdk.FactInfo, error) {
	return nil, nil
}

func (r *RehearsalQueryClient) ClaimsBySubmitter(_ context.Context, _ string) ([]agentsdk.ClaimInfo, error) {
	return nil, nil
}

func (r *RehearsalQueryClient) GetClaim(_ context.Context, claimID string) (*agentsdk.ClaimInfo, error) {
	return &agentsdk.ClaimInfo{
		ID:       claimID,
		Text:     "Rehearsal claim: " + claimID,
		Domain:   "general",
		Category: "empirical",
		Status:   "pending",
		Stake:    "1000000",
	}, nil
}

func (r *RehearsalQueryClient) FactReferences(_ context.Context, _ string) ([]agentsdk.FactReference, error) {
	return nil, nil
}

func (r *RehearsalQueryClient) CounterFacts(_ context.Context, _ string) ([]agentsdk.CounterFactInfo, error) {
	return nil, nil
}

// SetBalance sets a specific agent's balance.
func (r *RehearsalQueryClient) SetBalance(address string, balance uint64) {
	r.balancesMu.Lock()
	defer r.balancesMu.Unlock()
	r.balances[address] = balance
}

// AdvanceBlocks increments the chain height.
func (r *RehearsalQueryClient) AdvanceBlocks(n int64) {
	r.height.Add(n)
}

// Height returns the current simulated chain height.
func (r *RehearsalQueryClient) Height() int64 {
	return r.height.Load()
}

// RehearsalTxClient is a mock TxClient for dry-run execution.
// All transactions succeed immediately with synthetic hashes.
type RehearsalTxClient struct {
	txCount atomic.Int64
}

// NewRehearsalTxClient creates a tx client for rehearsal mode.
func NewRehearsalTxClient() *RehearsalTxClient {
	return &RehearsalTxClient{}
}

func (r *RehearsalTxClient) nextHash() string {
	n := r.txCount.Add(1)
	return fmt.Sprintf("REHEARSAL_TX_%04d", n)
}

func (r *RehearsalTxClient) RegisterAccount(_ context.Context, _ string) (string, error) {
	return r.nextHash(), nil
}

func (r *RehearsalTxClient) RegisterAgent(_ context.Context, _ agentsdk.RegisterAgentRequest) (string, error) {
	return r.nextHash(), nil
}

func (r *RehearsalTxClient) Heartbeat(_ context.Context) (string, error) {
	return r.nextHash(), nil
}

func (r *RehearsalTxClient) SubmitClaim(_ context.Context, _ agentsdk.SubmitClaimRequest) (string, error) {
	return r.nextHash(), nil
}

func (r *RehearsalTxClient) SubmitCommitment(_ context.Context, _, _ string) (string, error) {
	return r.nextHash(), nil
}

func (r *RehearsalTxClient) SubmitReveal(_ context.Context, _ string, _ agentsdk.Verdict, _ uint32, _ string) (string, error) {
	return r.nextHash(), nil
}

func (r *RehearsalTxClient) ChallengeFact(_ context.Context, _, _ string, _ uint64) (string, error) {
	return r.nextHash(), nil
}

func (r *RehearsalTxClient) CreateHome(_ context.Context, _ string) (string, error) {
	return r.nextHash(), nil
}

func (r *RehearsalTxClient) UpdateMemoryCID(_ context.Context, _, _ string) (string, error) {
	return r.nextHash(), nil
}

func (r *RehearsalTxClient) Stake(_ context.Context, _ string, _ uint64) (string, error) {
	return r.nextHash(), nil
}

func (r *RehearsalTxClient) DeployContract(_ context.Context, _ string) (string, error) {
	return r.nextHash(), nil
}

func (r *RehearsalTxClient) CallContract(_ context.Context, _, _ string, _ uint64) (string, error) {
	return r.nextHash(), nil
}

func (r *RehearsalTxClient) ScheduleExecution(_ context.Context, _, _, _ string, _, _ uint64) (string, error) {
	return r.nextHash(), nil
}

func (r *RehearsalTxClient) ConfigureGuardian(_ context.Context, _, _ string, _ bool, _ uint64) (string, error) {
	return r.nextHash(), nil
}

func (r *RehearsalTxClient) RegisterHomeKey(_ context.Context, _, _, _ string, _ []string, _ uint64) (string, error) {
	return r.nextHash(), nil
}

func (r *RehearsalTxClient) StartSession(_ context.Context, _, _ string, _ []string) (string, error) {
	return r.nextHash(), nil
}

func (r *RehearsalTxClient) EndSession(_ context.Context, _, _ string) (string, error) {
	return r.nextHash(), nil
}

func (r *RehearsalTxClient) SetSpendingLimit(_ context.Context, _, _ string, _, _ uint64) (string, error) {
	return r.nextHash(), nil
}

// RehearsalExecutor always succeeds for dry-run mode.
type RehearsalExecutor struct{}

func (r *RehearsalExecutor) Execute(_ context.Context, _ *agentsdk.Agent, action ActionType, params map[string]string) (*PartResult, error) {
	return &PartResult{
		Success: true,
		TxHash:  fmt.Sprintf("REHEARSAL_%s", action),
		Data:    params,
	}, nil
}

// BuildRehearsalSwarm creates a fully-initialized swarm with mock clients,
// suitable for dry-run testing. Uses a temp directory for key storage.
// The caller should defer os.RemoveAll on the returned dir.
func BuildRehearsalSwarm(score *Score, qc agentsdk.QueryClient, tc agentsdk.TxClient) (*swarm.Swarm, string, error) {
	tmpDir, err := os.MkdirTemp("", "orchestrator-rehearsal-*")
	if err != nil {
		return nil, "", fmt.Errorf("create temp dir: %w", err)
	}

	// Build agent specs from score ensemble.
	var agentSpecs []swarm.AgentSpec
	for _, sec := range score.Ensemble.Sections {
		role := SectionRoleMap[sec.Name]
		for i := 0; i < sec.MinCount; i++ {
			agentSpecs = append(agentSpecs, swarm.AgentSpec{
				Name:         fmt.Sprintf("%s-%d", sec.Name, i+1),
				KeyName:      fmt.Sprintf("%s-%d", sec.Name, i+1),
				Role:         role,
				Domains:      sec.Domains,
				InitialStake: score.Economics.StakePerAgent,
				Enabled:      true,
			})
		}
	}

	cfg := swarm.SwarmConfig{
		Name:    "rehearsal-" + score.Name,
		BaseDir: tmpDir,
		NodeRPC: agentsdk.DefaultNode,
		ChainID: "legible-rehearsal-1",
		Agents:  agentSpecs,
	}

	sw := swarm.NewSwarm(cfg)
	if err := sw.Init(); err != nil {
		os.RemoveAll(tmpDir)
		return nil, "", fmt.Errorf("init swarm: %w", err)
	}

	// Replace CLI clients with mock clients and set status to active
	// so AgentsByRole() returns them.
	for _, spec := range agentSpecs {
		ma, ok := sw.GetAgent(spec.Name)
		if !ok {
			continue
		}
		ma.Agent.SetQueryClient(qc)
		ma.Agent.SetTxClient(tc)
		ma.Status = swarm.StatusActive
	}

	return sw, tmpDir, nil
}

// Rehearse runs the score in dry-run mode with mock clients.
// Returns the sequence of events that would occur.
func Rehearse(score *Score) ([]Event, error) {
	if err := score.Validate(); err != nil {
		return nil, fmt.Errorf("invalid score: %w", err)
	}

	qc := NewRehearsalQueryClient()
	tc := NewRehearsalTxClient()

	sw, tmpDir, err := BuildRehearsalSwarm(score, qc, tc)
	if err != nil {
		return nil, fmt.Errorf("build rehearsal swarm: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	var events []Event
	var mu sync.Mutex
	handler := func(e Event) {
		mu.Lock()
		events = append(events, e)
		mu.Unlock()
	}

	poller := NewChainPoller(time.Second, func(ctx context.Context) (int64, error) {
		return qc.Height(), nil
	})
	poller.SetHeight(100)

	conductor, err := NewConductor(score, sw,
		WithEventHandler(handler),
		WithExecutor(&RehearsalExecutor{}),
	)
	if err != nil {
		return nil, err
	}
	conductor.SetPoller(poller)

	ctx := context.Background()
	if err := conductor.Tune(ctx); err != nil {
		return events, fmt.Errorf("tune: %w", err)
	}

	// Auto-advance blocks for block height gates.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			time.Sleep(10 * time.Millisecond)
			qc.AdvanceBlocks(10)
			poller.SetHeight(qc.Height())
			s := conductor.State()
			if s == StateComplete || s == StateAborted || s == StateFailed {
				return
			}
		}
	}()

	err = conductor.Perform(ctx)
	<-done
	if err != nil {
		return events, fmt.Errorf("perform: %w", err)
	}

	return events, nil
}
