package orchestrator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Checkpoint captures the conductor's state at a movement boundary for crash recovery.
type Checkpoint struct {
	ScoreName        string          `json:"score_name"`
	State            ConductorState  `json:"state"`
	CurrentMovement  int             `json:"current_movement"`   // Index into Score.Movements
	CompletedParts   map[string]bool `json:"completed_parts"`    // partID → completed
	SectionLeaders   map[string]string `json:"section_leaders"`  // section → leader agent name
	BlockHeight      int64           `json:"block_height"`
	Timestamp        time.Time       `json:"timestamp"`
	TotalGasSpent    uint64          `json:"total_gas_spent"`
	TotalRewards     uint64          `json:"total_rewards"`
}

// StateStore persists and retrieves checkpoints.
type StateStore interface {
	Save(cp *Checkpoint) error
	Load() (*Checkpoint, error)
	Exists() bool
	Clear() error
}

// FileStateStore implements StateStore using a JSON file on disk.
type FileStateStore struct {
	Dir string // Directory for checkpoint files
}

// NewFileStateStore creates a file-based state store.
func NewFileStateStore(dir string) *FileStateStore {
	return &FileStateStore{Dir: dir}
}

func (s *FileStateStore) path() string {
	return filepath.Join(s.Dir, "orchestrator-checkpoint.json")
}

// Save writes the checkpoint to disk atomically.
func (s *FileStateStore) Save(cp *Checkpoint) error {
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return fmt.Errorf("create checkpoint dir: %w", err)
	}

	data, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal checkpoint: %w", err)
	}

	// Atomic write via temp file + rename.
	tmp := s.path() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write checkpoint: %w", err)
	}
	if err := os.Rename(tmp, s.path()); err != nil {
		return fmt.Errorf("rename checkpoint: %w", err)
	}
	return nil
}

// Load reads a checkpoint from disk.
func (s *FileStateStore) Load() (*Checkpoint, error) {
	data, err := os.ReadFile(s.path())
	if err != nil {
		return nil, fmt.Errorf("read checkpoint: %w", err)
	}
	var cp Checkpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, fmt.Errorf("unmarshal checkpoint: %w", err)
	}
	return &cp, nil
}

// Exists returns true if a checkpoint file exists.
func (s *FileStateStore) Exists() bool {
	_, err := os.Stat(s.path())
	return err == nil
}

// Clear removes the checkpoint file.
func (s *FileStateStore) Clear() error {
	if !s.Exists() {
		return nil
	}
	return os.Remove(s.path())
}

// MemoryStateStore is an in-memory state store for testing.
type MemoryStateStore struct {
	cp *Checkpoint
}

func (m *MemoryStateStore) Save(cp *Checkpoint) error {
	m.cp = cp
	return nil
}

func (m *MemoryStateStore) Load() (*Checkpoint, error) {
	if m.cp == nil {
		return nil, fmt.Errorf("no checkpoint")
	}
	return m.cp, nil
}

func (m *MemoryStateStore) Exists() bool {
	return m.cp != nil
}

func (m *MemoryStateStore) Clear() error {
	m.cp = nil
	return nil
}
