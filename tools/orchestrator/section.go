package orchestrator

import (
	"fmt"
	"sort"
	"sync"

	swarm "github.com/legible-money/legible/tools/agent-swarm"
)

// Section is a group of agents with a shared role, coordinated by a leader.
type Section struct {
	Name    SectionName
	Spec    SectionSpec
	Leader  *SectionMember
	Members []*SectionMember

	mu sync.RWMutex
}

// SectionMember wraps a ManagedAgent with orchestrator-specific tracking.
type SectionMember struct {
	Managed     *swarm.ManagedAgent
	ActiveParts int    // Number of currently executing parts
	TotalDone   int    // Total parts completed
	TotalFailed int    // Total parts failed
	Paused      bool   // Temporarily paused (e.g., low balance)
}

// PartAssignment tracks which agent is working on which part.
type PartAssignment struct {
	PartID    string
	Agent     string // Agent name
	Status    PartStatus
	Retries   int
	Result    *PartResult
}

// PartStatus tracks execution progress of a single part assignment.
type PartStatus string

const (
	PartPending   PartStatus = "pending"
	PartRunning   PartStatus = "running"
	PartCompleted PartStatus = "completed"
	PartFailed    PartStatus = "failed"
)

// PartResult captures the outcome of executing a part.
type PartResult struct {
	Success bool              `json:"success"`
	TxHash  string            `json:"tx_hash,omitempty"`
	Error   string            `json:"error,omitempty"`
	Data    map[string]string `json:"data,omitempty"`
}

// NewSection creates a section from a spec and a set of managed agents.
func NewSection(spec SectionSpec, agents []*swarm.ManagedAgent) (*Section, error) {
	if len(agents) < spec.MinCount {
		return nil, fmt.Errorf("section %s needs at least %d agents, got %d", spec.Name, spec.MinCount, len(agents))
	}

	members := make([]*SectionMember, len(agents))
	for i, ma := range agents {
		members[i] = &SectionMember{Managed: ma}
	}

	s := &Section{
		Name:    spec.Name,
		Spec:    spec,
		Members: members,
	}
	s.ElectLeader()
	return s, nil
}

// ElectLeader selects the section leader by highest reputation, then highest
// balance (from metrics), then deterministic name sort for tiebreak.
func (s *Section) ElectLeader() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.Members) == 0 {
		return
	}

	candidates := make([]*SectionMember, len(s.Members))
	copy(candidates, s.Members)

	sort.SliceStable(candidates, func(i, j int) bool {
		mi, mj := candidates[i], candidates[j]
		// Higher reputation first (from AgentProfileInfo cached in metrics).
		ri := mi.Managed.Metrics.RewardsEarned
		rj := mj.Managed.Metrics.RewardsEarned
		if ri != rj {
			return ri > rj
		}
		// Deterministic tiebreak: lexicographic name sort.
		return mi.Managed.Spec.Name < mj.Managed.Spec.Name
	})

	s.Leader = candidates[0]
}

// AssignPart picks the least-loaded active member for a part.
// Returns the selected member or error if no member is available.
func (s *Section) AssignPart(partID string) (*SectionMember, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var best *SectionMember
	for _, m := range s.Members {
		if m.Paused || m.Managed.Status == swarm.StatusFailed || m.Managed.Status == swarm.StatusPaused {
			continue
		}
		if best == nil || m.ActiveParts < best.ActiveParts {
			best = m
		}
	}

	if best == nil {
		return nil, fmt.Errorf("section %s: no available members for part %s", s.Name, partID)
	}
	return best, nil
}

// BroadcastPart returns all active (non-paused, non-failed) members for
// parts that should be executed by every member in the section.
func (s *Section) BroadcastPart() []*SectionMember {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var active []*SectionMember
	for _, m := range s.Members {
		if m.Paused || m.Managed.Status == swarm.StatusFailed || m.Managed.Status == swarm.StatusPaused {
			continue
		}
		active = append(active, m)
	}
	return active
}

// ActiveCount returns the number of non-paused, non-failed members.
func (s *Section) ActiveCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	n := 0
	for _, m := range s.Members {
		if !m.Paused && m.Managed.Status != swarm.StatusFailed {
			n++
		}
	}
	return n
}

// AddMember adds an agent to the section if max_count is not exceeded.
func (s *Section) AddMember(ma *swarm.ManagedAgent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.Members) >= s.Spec.MaxCount {
		return fmt.Errorf("section %s at max capacity (%d)", s.Name, s.Spec.MaxCount)
	}

	s.Members = append(s.Members, &SectionMember{Managed: ma})
	return nil
}

// RemoveMember removes a member by agent name, if above min_count.
// Cannot remove the leader.
func (s *Section) RemoveMember(agentName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.Members) <= s.Spec.MinCount {
		return fmt.Errorf("section %s at min capacity (%d)", s.Name, s.Spec.MinCount)
	}

	for i, m := range s.Members {
		if m.Managed.Spec.Name == agentName {
			if s.Leader != nil && s.Leader.Managed.Spec.Name == agentName {
				return fmt.Errorf("cannot remove leader %s; re-elect first", agentName)
			}
			s.Members = append(s.Members[:i], s.Members[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("agent %s not found in section %s", agentName, s.Name)
}

// MemberByName finds a member by agent name.
func (s *Section) MemberByName(name string) *SectionMember {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, m := range s.Members {
		if m.Managed.Spec.Name == name {
			return m
		}
	}
	return nil
}

// Health returns a SectionHealth snapshot.
func (s *Section) Health() SectionHealth {
	s.mu.RLock()
	defer s.mu.RUnlock()

	h := SectionHealth{
		Name:        s.Name,
		MemberCount: len(s.Members),
	}
	if s.Leader != nil {
		h.LeaderAgent = s.Leader.Managed.Spec.Name
	}
	for _, m := range s.Members {
		h.ActiveParts += m.ActiveParts
		h.CompletedParts += m.TotalDone
		h.FailedParts += m.TotalFailed
	}
	return h
}
