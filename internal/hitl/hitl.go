package hitl

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

const (
	DefaultMaxPending = 100
	DefaultTTL        = 3600 * time.Second
)

// Action represents a staged mutating action awaiting human approval.
type Action struct {
	Type      string         `json:"type"`
	Details   map[string]any `json:"details"`
	CreatedAt time.Time      `json:"created_at"`
}

// Manager coordinates human-in-the-loop validation and token staging.
type Manager struct {
	mu         sync.Mutex
	pending    map[string]*Action
	maxPending int
	ttl        time.Duration
}

// NewManager creates a new HITL Manager instance.
func NewManager(maxPending int, ttl time.Duration) *Manager {
	if maxPending <= 0 {
		maxPending = DefaultMaxPending
	}
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	return &Manager{
		pending:    make(map[string]*Action),
		maxPending: maxPending,
		ttl:        ttl,
	}
}

func generateToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// CleanupExpired removes actions older than TTL. Must be called with lock held or via public method.
func (m *Manager) CleanupExpired() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cleanupExpiredLocked()
}

func (m *Manager) cleanupExpiredLocked() int {
	now := time.Now()
	cleaned := 0
	for tok, act := range m.pending {
		if now.Sub(act.CreatedAt) > m.ttl {
			delete(m.pending, tok)
			cleaned++
		}
	}
	return cleaned
}

// Request stages a mutating action, generates a token, and returns the blocking HITL notice.
func (m *Manager) Request(actionType string, details map[string]any) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cleanupExpiredLocked()

	// Evict oldest action if capacity reached
	if len(m.pending) >= m.maxPending {
		var oldestTok string
		var oldestTime time.Time
		first := true
		for tok, act := range m.pending {
			if first || act.CreatedAt.Before(oldestTime) {
				oldestTok = tok
				oldestTime = act.CreatedAt
				first = false
			}
		}
		if oldestTok != "" {
			delete(m.pending, oldestTok)
		}
	}

	token := generateToken()
	m.pending[token] = &Action{
		Type:      actionType,
		Details:   details,
		CreatedAt: time.Now(),
	}

	detailsJSON, _ := json.MarshalIndent(details, "", "  ")

	operator := os.Getenv("MAILRU_OPERATOR_NAME")
	if operator == "" {
		operator = "the operator (ZavLab)"
	}

	return fmt.Sprintf(
		"⚠️ ACTION BLOCKED BY HITL (Human-In-The-Loop) POLICY\n\n"+
			"Type: %s\n"+
			"Details: %s\n\n"+
			"To execute this action, obtain explicit confirmation from %s and run:\n"+
			"`execute_pending_action(token='%s')`\n"+
			"Note: This token is valid for 1 hour.",
		actionType, string(detailsJSON), operator, token,
	)
}

// Pop atomically retrieves and deletes a staged action for execution.
func (m *Manager) Pop(token string) (*Action, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cleanupExpiredLocked()

	act, ok := m.pending[token]
	if !ok {
		return nil, errors.New("Invalid, expired, or already executed HITL token.")
	}

	delete(m.pending, token)
	return act, nil
}

// Len returns the current count of pending actions.
func (m *Manager) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.pending)
}
