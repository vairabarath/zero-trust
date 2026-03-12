package state

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AgentState represents the agent's persistent state
type AgentState struct {
	AgentID        string          `json:"agent_id"`
	ProtectedPorts []ProtectedPort `json:"protected_ports"`
	LastSync       time.Time       `json:"last_sync"`
	mu             sync.RWMutex
	filePath       string
}

// ProtectedPort represents a port that is currently protected
type ProtectedPort struct {
	Port      int       `json:"port"`
	Protocol  string    `json:"protocol"`
	AppliedAt time.Time `json:"applied_at"`
	PolicyID  string    `json:"policy_id,omitempty"`
}

// NewAgentState creates a new agent state manager
func NewAgentState(filePath string) *AgentState {
	return &AgentState{
		ProtectedPorts: []ProtectedPort{},
		filePath:       filePath,
		LastSync:       time.Now(),
	}
}

// Load loads state from disk
func (s *AgentState) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet, that's OK
			log.Printf("No existing state file found at %s", s.filePath)
			return nil
		}
		return fmt.Errorf("failed to read state file: %v", err)
	}

	if err := json.Unmarshal(data, s); err != nil {
		return fmt.Errorf("failed to parse state file: %v", err)
	}

	log.Printf("✓ Loaded state from %s", s.filePath)
	log.Printf("  - Agent ID: %s", s.AgentID)
	log.Printf("  - Protected ports: %d", len(s.ProtectedPorts))

	return nil
}

// Save saves state to disk
func (s *AgentState) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	s.LastSync = time.Now()

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %v", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %v", err)
	}

	// Write to temporary file first, then rename (atomic)
	tmpFile := s.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write state file: %v", err)
	}

	if err := os.Rename(tmpFile, s.filePath); err != nil {
		return fmt.Errorf("failed to rename state file: %v", err)
	}

	return nil
}

// SetAgentID sets the agent ID
func (s *AgentState) SetAgentID(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.AgentID = id
}

// GetAgentID returns the agent ID
func (s *AgentState) GetAgentID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.AgentID
}

// AddProtectedPort adds a port to the protected list
func (s *AgentState) AddProtectedPort(port int, protocol string, policyID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if already exists
	for _, p := range s.ProtectedPorts {
		if p.Port == port {
			return // Already protected
		}
	}

	s.ProtectedPorts = append(s.ProtectedPorts, ProtectedPort{
		Port:      port,
		Protocol:  protocol,
		AppliedAt: time.Now(),
		PolicyID:  policyID,
	})

	// Auto-save after modification
	go func() {
		if err := s.Save(); err != nil {
			log.Printf("⚠️  Failed to save state: %v", err)
		}
	}()
}

// RemoveProtectedPort removes a port from the protected list
func (s *AgentState) RemoveProtectedPort(port int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	newPorts := []ProtectedPort{}
	for _, p := range s.ProtectedPorts {
		if p.Port != port {
			newPorts = append(newPorts, p)
		}
	}

	s.ProtectedPorts = newPorts

	// Auto-save after modification
	go func() {
		if err := s.Save(); err != nil {
			log.Printf("⚠️  Failed to save state: %v", err)
		}
	}()
}

// GetProtectedPorts returns all protected ports
func (s *AgentState) GetProtectedPorts() []ProtectedPort {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy
	ports := make([]ProtectedPort, len(s.ProtectedPorts))
	copy(ports, s.ProtectedPorts)
	return ports
}

// IsPortProtected checks if a port is protected
func (s *AgentState) IsPortProtected(port int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, p := range s.ProtectedPorts {
		if p.Port == port {
			return true
		}
	}
	return false
}

// Clear clears all state
func (s *AgentState) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ProtectedPorts = []ProtectedPort{}
	s.LastSync = time.Now()
}
