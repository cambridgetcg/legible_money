package orchestrator

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

// OpenClawManager manages OpenClaw AI reasoning sessions.
// Sessions are spawned as subprocesses that accept commands via stdin/stdout.
type OpenClawManager struct {
	binary   string // Path to openclaw binary
	sessions map[string]*OpenClawSession
	mu       sync.RWMutex
}

// OpenClawSession represents a running OpenClaw reasoning session.
type OpenClawSession struct {
	ID      string
	AgentID string // Agent this session belongs to
	Model   string // AI model to use
	Status  string // "running", "stopped", "error"
	cmd     *exec.Cmd
	cancel  context.CancelFunc
}

// NewOpenClawManager creates a manager for OpenClaw sessions.
func NewOpenClawManager(binary string) *OpenClawManager {
	return &OpenClawManager{
		binary:   binary,
		sessions: make(map[string]*OpenClawSession),
	}
}

// SpawnSession starts a new OpenClaw reasoning session for an agent.
// The session runs as a subprocess with the given model and system prompt.
func (m *OpenClawManager) SpawnSession(ctx context.Context, sessionID, agentID, model, systemPrompt string) (*OpenClawSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.sessions[sessionID]; exists {
		return nil, fmt.Errorf("session %s already exists", sessionID)
	}

	sessionCtx, cancel := context.WithCancel(ctx)

	args := []string{
		"session", "spawn",
		"--id", sessionID,
		"--model", model,
		"--system-prompt", systemPrompt,
	}

	cmd := exec.CommandContext(sessionCtx, m.binary, args...)

	session := &OpenClawSession{
		ID:      sessionID,
		AgentID: agentID,
		Model:   model,
		Status:  "running",
		cmd:     cmd,
		cancel:  cancel,
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("spawn session %s: %w", sessionID, err)
	}

	// Monitor process in background.
	go func() {
		_ = cmd.Wait()
		m.mu.Lock()
		if s, ok := m.sessions[sessionID]; ok {
			s.Status = "stopped"
		}
		m.mu.Unlock()
	}()

	m.sessions[sessionID] = session
	return session, nil
}

// SendCommand sends a command to an existing session and returns the response.
func (m *OpenClawManager) SendCommand(ctx context.Context, sessionID, command string) (string, error) {
	m.mu.RLock()
	session, ok := m.sessions[sessionID]
	m.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("session %s not found", sessionID)
	}
	if session.Status != "running" {
		return "", fmt.Errorf("session %s is %s", sessionID, session.Status)
	}

	args := []string{
		"session", "send",
		"--id", sessionID,
		"--command", command,
	}

	cmd := exec.CommandContext(ctx, m.binary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("send to session %s: %w (stderr: %s)", sessionID, err, strings.TrimSpace(stderr.String()))
	}

	return strings.TrimSpace(stdout.String()), nil
}

// StopSession stops a running session.
func (m *OpenClawManager) StopSession(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session %s not found", sessionID)
	}

	session.cancel()
	session.Status = "stopped"
	return nil
}

// StopAll stops all running sessions.
func (m *OpenClawManager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, session := range m.sessions {
		if session.Status == "running" {
			session.cancel()
			session.Status = "stopped"
		}
	}
}

// GetSession returns a session by ID.
func (m *OpenClawManager) GetSession(sessionID string) (*OpenClawSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[sessionID]
	return s, ok
}

// ListSessions returns all sessions.
func (m *OpenClawManager) ListSessions() []*OpenClawSession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*OpenClawSession, 0, len(m.sessions))
	for _, s := range m.sessions {
		result = append(result, s)
	}
	return result
}

// ActiveCount returns the number of running sessions.
func (m *OpenClawManager) ActiveCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	n := 0
	for _, s := range m.sessions {
		if s.Status == "running" {
			n++
		}
	}
	return n
}
