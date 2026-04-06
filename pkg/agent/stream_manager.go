package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/llm"
)

// StreamState tracks the current state of a single agent's stream.
type StreamState struct {
	AgentName    string
	Caste        Caste
	Status       StreamStatus
	Tokens       []string
	TokenCount   int
	StartedAt    time.Time
	CompletedAt  *time.Time
	Error        error
	Result       *llm.StreamResult
	mu           sync.RWMutex
}

// StreamStatus represents the lifecycle state of a stream.
type StreamStatus string

const (
	StreamStatusPending    StreamStatus = "pending"
	StreamStatusActive     StreamStatus = "active"
	StreamStatusCompleted  StreamStatus = "completed"
	StreamStatusFailed     StreamStatus = "failed"
	StreamStatusCancelled  StreamStatus = "cancelled"
)

// IsActive returns true if the stream is currently active.
func (s *StreamState) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Status == StreamStatusActive
}

// IsComplete returns true if the stream has finished (completed, failed, or cancelled).
func (s *StreamState) IsComplete() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Status == StreamStatusCompleted || s.Status == StreamStatusFailed || s.Status == StreamStatusCancelled
}

// AddToken appends a token to the stream's token history.
func (s *StreamState) AddToken(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Tokens = append(s.Tokens, token)
	s.TokenCount++
}

// GetTokens returns a copy of all tokens received so far.
func (s *StreamState) GetTokens() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]string, len(s.Tokens))
	copy(result, s.Tokens)
	return result
}

// GetTokenCount returns the total number of tokens received.
func (s *StreamState) GetTokenCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.TokenCount
}

// SetStatus updates the stream status thread-safely.
func (s *StreamState) SetStatus(status StreamStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = status
	if status == StreamStatusCompleted || status == StreamStatusFailed || status == StreamStatusCancelled {
		now := time.Now()
		s.CompletedAt = &now
	}
}

// GetStatus returns the current stream status.
func (s *StreamState) GetStatus() StreamStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Status
}

// SetError records an error for the stream.
func (s *StreamState) SetError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Error = err
	s.Status = StreamStatusFailed
	now := time.Now()
	s.CompletedAt = &now
}

// SetResult records the final result for the stream.
func (s *StreamState) SetResult(result *llm.StreamResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Result = result
}

// GetResult returns the stream result (may be nil if not complete).
func (s *StreamState) GetResult() *llm.StreamResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Result
}

// GetError returns any error that occurred during streaming.
func (s *StreamState) GetError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Error
}

// Duration returns how long the stream has been running (or ran).
func (s *StreamState) Duration() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.CompletedAt != nil {
		return s.CompletedAt.Sub(s.StartedAt)
	}
	return time.Since(s.StartedAt)
}

// StreamManager coordinates multiple concurrent agent streams.
// It uses the spawn tree to track active agents and maps agent names to their stream states.
type StreamManager struct {
	spawnTree   *SpawnTree
	streams     map[string]*StreamState
	mu          sync.RWMutex
	maxStreams  int
}

// NewStreamManager creates a new StreamManager with the given spawn tree.
// maxStreams limits the number of concurrent streams (0 = unlimited).
func NewStreamManager(spawnTree *SpawnTree, maxStreams int) *StreamManager {
	if maxStreams < 0 {
		maxStreams = 0
	}
	return &StreamManager{
		spawnTree:  spawnTree,
		streams:    make(map[string]*StreamState),
		maxStreams: maxStreams,
	}
}

// RegisterAgent creates a new stream state for an agent.
// Returns an error if the agent is already registered.
func (sm *StreamManager) RegisterAgent(agent Agent) (*StreamState, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	name := agent.Name()
	if _, exists := sm.streams[name]; exists {
		return nil, fmt.Errorf("stream already exists for agent: %s", name)
	}

	// Check max streams limit (total registered, not just active)
	if sm.maxStreams > 0 && len(sm.streams) >= sm.maxStreams {
		return nil, fmt.Errorf("max concurrent streams reached: %d", sm.maxStreams)
	}

	state := &StreamState{
		AgentName:  name,
		Caste:      agent.Caste(),
		Status:     StreamStatusPending,
		Tokens:     make([]string, 0),
		StartedAt:  time.Now(),
	}

	sm.streams[name] = state
	return state, nil
}

// GetStream retrieves the stream state for a given agent name.
func (sm *StreamManager) GetStream(agentName string) (*StreamState, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	state, ok := sm.streams[agentName]
	return state, ok
}

// UnregisterAgent removes an agent's stream state from the manager.
func (sm *StreamManager) UnregisterAgent(agentName string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.streams, agentName)
}

// GetActiveStreams returns all currently active streams.
func (sm *StreamManager) GetActiveStreams() []*StreamState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var active []*StreamState
	for _, state := range sm.streams {
		if state.IsActive() {
			active = append(active, state)
		}
	}
	return active
}

// GetAllStreams returns all stream states (active and completed).
func (sm *StreamManager) GetAllStreams() []*StreamState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make([]*StreamState, 0, len(sm.streams))
	for _, state := range sm.streams {
		result = append(result, state)
	}
	return result
}

// GetCompletedStreams returns all completed streams.
func (sm *StreamManager) GetCompletedStreams() []*StreamState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var completed []*StreamState
	for _, state := range sm.streams {
		if state.IsComplete() {
			completed = append(completed, state)
		}
	}
	return completed
}

// ActiveCount returns the number of currently active streams.
func (sm *StreamManager) ActiveCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.activeCountLocked()
}

func (sm *StreamManager) activeCountLocked() int {
	count := 0
	for _, state := range sm.streams {
		if state.IsActive() {
			count++
		}
	}
	return count
}

// TotalCount returns the total number of registered streams.
func (sm *StreamManager) TotalCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.streams)
}

// WaitForAll blocks until all registered streams are complete.
// Returns a map of agent names to their final states.
func (sm *StreamManager) WaitForAll() map[string]*StreamState {
	for {
		active := sm.ActiveCount()
		if active == 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make(map[string]*StreamState, len(sm.streams))
	for name, state := range sm.streams {
		result[name] = state
	}
	return result
}

// WaitForAgent blocks until a specific agent's stream is complete.
func (sm *StreamManager) WaitForAgent(agentName string) (*StreamState, bool) {
	for {
		state, ok := sm.GetStream(agentName)
		if !ok {
			return nil, false
		}
		if state.IsComplete() {
			return state, true
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// agentStreamHandler wraps a StreamState to implement llm.StreamHandler.
type agentStreamHandler struct {
	state *StreamState
}

func (h *agentStreamHandler) OnToken(token string) {
	h.state.AddToken(token)
}

func (h *agentStreamHandler) OnToolStart(toolName, toolID string) {
	// Track tool usage if needed
}

func (h *agentStreamHandler) OnToolEnd(toolName, toolID, result string) {
	// Track tool completion if needed
}

func (h *agentStreamHandler) OnComplete(result *llm.StreamResult) {
	h.state.SetResult(result)
	h.state.SetStatus(StreamStatusCompleted)
}

func (h *agentStreamHandler) OnError(err error) {
	h.state.SetError(err)
}

// ExecuteStreaming runs an agent with streaming and tracks its state.
// The agent must be registered first with RegisterAgent.
// Note: This method requires agent registry integration. Use ExecuteStreamingWithAgent instead.
func (sm *StreamManager) ExecuteStreaming(ctx context.Context, agentName string, event events.Event) error {
	_, ok := sm.GetStream(agentName)
	if !ok {
		return fmt.Errorf("agent not registered: %s", agentName)
	}

	// Get the agent from registry (requires external registry)
	return fmt.Errorf("ExecuteStreaming requires agent registry integration - use ExecuteStreamingWithAgent instead")
}

// ExecuteStreamingWithAgent runs a specific agent with streaming and tracks its state.
func (sm *StreamManager) ExecuteStreamingWithAgent(ctx context.Context, agent StreamingAgent, event events.Event) (*StreamState, error) {
	// Register or get existing state
	state, err := sm.RegisterAgent(agent)
	if err != nil {
		// If already registered, get the existing state
		existing, ok := sm.GetStream(agent.Name())
		if !ok {
			return nil, err
		}
		state = existing
	}

	// Mark as active
	state.SetStatus(StreamStatusActive)

	// Create handler that updates state
	handler := &agentStreamHandler{state: state}

	// Execute with streaming
	go func() {
		err := agent.ExecuteStreaming(ctx, event, handler)
		if err != nil && state.GetError() == nil {
			state.SetError(err)
		}
		// Ensure status is set if not already completed
		if !state.IsComplete() {
			if state.GetError() != nil {
				state.SetStatus(StreamStatusFailed)
			} else {
				state.SetStatus(StreamStatusCompleted)
			}
		}
	}()

	return state, nil
}

// SyncWithSpawnTree updates stream states based on the spawn tree's active agents.
// This ensures the StreamManager stays in sync with the actual colony state.
func (sm *SpawnTree) SyncWithSpawnTree(sm2 *StreamManager) error {
	// This method is on SpawnTree to access its internals
	entries, err := sm.Parse()
	if err != nil {
		return err
	}

	// Build set of active agent names from spawn tree
	activeAgents := make(map[string]bool)
	for _, entry := range entries {
		if entry.Status == "spawned" || entry.Status == "active" {
			activeAgents[entry.AgentName] = true
		}
	}

	// Clean up streams for agents no longer in spawn tree
	sm2.mu.Lock()
	defer sm2.mu.Unlock()

	for name, state := range sm2.streams {
		if !activeAgents[name] && !state.IsComplete() {
			// Agent no longer in spawn tree but stream not marked complete
			state.SetStatus(StreamStatusCancelled)
		}
	}

	return nil
}

// GetStreamSummary returns a summary of all streams for display.
func (sm *StreamManager) GetStreamSummary() StreamSummary {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	summary := StreamSummary{
		Total:     len(sm.streams),
		ByStatus:  make(map[StreamStatus]int),
		ByCaste:   make(map[Caste]int),
	}

	for _, state := range sm.streams {
		status := state.GetStatus()
		summary.ByStatus[status]++
		summary.ByCaste[state.Caste]++

		if status == StreamStatusActive {
			summary.Active++
		} else if state.IsComplete() {
			summary.Completed++
		}

		summary.TotalTokens += state.GetTokenCount()
	}

	return summary
}

// StreamSummary provides a high-level overview of stream states.
type StreamSummary struct {
	Total       int
	Active      int
	Completed   int
	TotalTokens int
	ByStatus    map[StreamStatus]int
	ByCaste     map[Caste]int
}
