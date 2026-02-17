package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
)

const defaultCacheTTL = 2 * time.Hour

// CachedVerdict stores a verdict for commit-reveal determinism.
// The same claimID must return the same verdict at both commit and reveal time.
type CachedVerdict struct {
	ClaimID    string          `json:"claim_id"`
	Verdict    agentsdk.Verdict `json:"verdict"`
	Confidence uint32          `json:"confidence"`
	CachedAt   time.Time       `json:"cached_at"`
	ExpiresAt  time.Time       `json:"expires_at"`
}

// VerdictCache provides deterministic verdict storage for the commit-reveal protocol.
// Verdicts are cached in memory and optionally persisted to disk.
type VerdictCache struct {
	mu      sync.RWMutex
	entries map[string]*CachedVerdict
	dir     string
	ttl     time.Duration

	// Clock for testability.
	now func() time.Time
}

// NewVerdictCache creates a cache. If dir is non-empty, verdicts are persisted to disk.
func NewVerdictCache(dir string) *VerdictCache {
	vc := &VerdictCache{
		entries: make(map[string]*CachedVerdict),
		dir:     dir,
		ttl:     defaultCacheTTL,
		now:     time.Now,
	}

	// Load persisted verdicts if directory exists.
	if dir != "" {
		vc.loadFromDisk()
	}

	return vc
}

// Get retrieves a cached verdict. Returns nil, false if not found or expired.
func (vc *VerdictCache) Get(claimID string) (*CachedVerdict, bool) {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	entry, ok := vc.entries[claimID]
	if !ok {
		return nil, false
	}

	if vc.now().After(entry.ExpiresAt) {
		return nil, false
	}

	return entry, true
}

// Set stores a verdict in the cache.
func (vc *VerdictCache) Set(claimID string, verdict agentsdk.Verdict, confidence uint32) {
	now := vc.now()
	entry := &CachedVerdict{
		ClaimID:    claimID,
		Verdict:    verdict,
		Confidence: confidence,
		CachedAt:   now,
		ExpiresAt:  now.Add(vc.ttl),
	}

	vc.mu.Lock()
	vc.entries[claimID] = entry
	vc.mu.Unlock()

	// Persist to disk (best-effort).
	if vc.dir != "" {
		vc.persistEntry(entry)
	}
}

// Cleanup removes expired entries from the cache and disk.
func (vc *VerdictCache) Cleanup() {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	now := vc.now()
	for id, entry := range vc.entries {
		if now.After(entry.ExpiresAt) {
			delete(vc.entries, id)
			if vc.dir != "" {
				_ = os.Remove(filepath.Join(vc.dir, id+".json"))
			}
		}
	}
}

// Size returns the number of cached entries (including expired ones not yet cleaned).
func (vc *VerdictCache) Size() int {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	return len(vc.entries)
}

func (vc *VerdictCache) persistEntry(entry *CachedVerdict) {
	if err := os.MkdirAll(vc.dir, 0700); err != nil {
		return
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	path := filepath.Join(vc.dir, entry.ClaimID+".json")
	_ = os.WriteFile(path, data, 0600)
}

func (vc *VerdictCache) loadFromDisk() {
	entries, err := os.ReadDir(vc.dir)
	if err != nil {
		return
	}

	now := vc.now()
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(vc.dir, e.Name()))
		if err != nil {
			continue
		}
		var cv CachedVerdict
		if err := json.Unmarshal(data, &cv); err != nil {
			continue
		}
		// Skip expired entries.
		if now.After(cv.ExpiresAt) {
			_ = os.Remove(filepath.Join(vc.dir, e.Name()))
			continue
		}
		vc.entries[cv.ClaimID] = &cv
	}
}

// SetTTL configures the cache time-to-live. Must be called before any Set operations.
func (vc *VerdictCache) SetTTL(ttl time.Duration) {
	vc.ttl = ttl
}

// Clear removes all entries from memory and disk.
func (vc *VerdictCache) Clear() {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	for id := range vc.entries {
		if vc.dir != "" {
			_ = os.Remove(filepath.Join(vc.dir, id+".json"))
		}
	}
	vc.entries = make(map[string]*CachedVerdict)
}

// Export returns all non-expired cached verdicts. Useful for debugging.
func (vc *VerdictCache) Export() []CachedVerdict {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	now := vc.now()
	var result []CachedVerdict
	for _, entry := range vc.entries {
		if now.Before(entry.ExpiresAt) {
			result = append(result, *entry)
		}
	}
	return result
}

// String returns a human-readable summary for logging.
func (vc *VerdictCache) String() string {
	return fmt.Sprintf("VerdictCache(entries=%d, dir=%q)", vc.Size(), vc.dir)
}
