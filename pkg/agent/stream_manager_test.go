package agent

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/llm"
	"github.com/calcosmic/Aether/pkg/storage"
)

// TestStreamStateBasicOperations tests the StreamState type.
func TestStreamStateBasicOperations(t *testing.T) {
	state := &StreamState{
		AgentName: "test-agent",
		Caste:     CasteBuilder,
		Status:    StreamStatusPending,
		Tokens:    make([]string, 0),
		StartedAt: time.Now(),
	}

	// Test initial state
	if state.GetStatus() != StreamStatusPending {
		t.Errorf("Expected status %s, got %s", StreamStatusPending, state.GetStatus())
	}

	if state.IsActive() {
		t.Error("New state should not be active")
	}

	if state.IsComplete() {
		t.Error("New state should not be complete")
	}

	// Test adding tokens
	state.AddToken("Hello")
	state.AddToken(" ")
	state.AddToken("World")

	if state.GetTokenCount() != 3 {
		t.Errorf("Expected 3 tokens, got %d", state.GetTokenCount())
	}

	tokens := state.GetTokens()
	if len(tokens) != 3 {
		t.Errorf("Expected 3 tokens in slice, got %d", len(tokens))
	}

	// Test status transitions
	state.SetStatus(StreamStatusActive)
	if !state.IsActive() {
		t.Error("State should be active after setting to Active")
	}

	state.SetStatus(StreamStatusCompleted)
	if !state.IsComplete() {
		t.Error("State should be complete after setting to Completed")
	}

	if state.CompletedAt == nil {
		t.Error("CompletedAt should be set when status is Completed")
	}

	// Test result
	result := &llm.StreamResult{
		Text:       "Test result",
		Role:       "assistant",
		StopReason: "end_turn",
	}
	state.SetResult(result)

	if state.GetResult() == nil {
		t.Error("Result should be set")
	}

	if state.GetResult().Text != "Test result" {
		t.Errorf("Expected result text 'Test result', got %s", state.GetResult().Text)
	}
}

// TestStreamStateErrorHandling tests error recording.
func TestStreamStateErrorHandling(t *testing.T) {
	state := &StreamState{
		AgentName: "test-agent",
		Caste:     CasteBuilder,
		Status:    StreamStatusActive,
		StartedAt: time.Now(),
	}

	testErr := context.Canceled
	state.SetError(testErr)

	if state.GetStatus() != StreamStatusFailed {
		t.Errorf("Expected status %s after error, got %s", StreamStatusFailed, state.GetStatus())
	}

	if state.GetError() != testErr {
		t.Errorf("Expected error %v, got %v", testErr, state.GetError())
	}

	if !state.IsComplete() {
		t.Error("Failed state should be complete")
	}
}

// TestStreamStateDuration tests duration calculation.
func TestStreamStateDuration(t *testing.T) {
	start := time.Now()
	state := &StreamState{
		AgentName: "test-agent",
		Caste:     CasteBuilder,
		StartedAt: start,
	}

	// Active duration
	duration := state.Duration()
	if duration < 0 {
		t.Error("Duration should be non-negative")
	}

	// Completed duration
	time.Sleep(10 * time.Millisecond)
	state.SetStatus(StreamStatusCompleted)

	duration = state.Duration()
	if duration < 10*time.Millisecond {
		t.Error("Duration should be at least 10ms")
	}
}

// TestStreamManagerCreation tests creating a StreamManager.
func TestStreamManagerCreation(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	spawnTree := NewSpawnTree(store, "")

	// Test with unlimited streams
	sm := NewStreamManager(spawnTree, 0)
	if sm == nil {
		t.Fatal("StreamManager should not be nil")
	}

	if sm.TotalCount() != 0 {
		t.Errorf("Expected 0 streams, got %d", sm.TotalCount())
	}

	if sm.ActiveCount() != 0 {
		t.Errorf("Expected 0 active streams, got %d", sm.ActiveCount())
	}

	// Test with limited streams
	smLimited := NewStreamManager(spawnTree, 4)
	if smLimited == nil {
		t.Fatal("Limited StreamManager should not be nil")
	}
}

// TestStreamManagerRegisterAgent tests agent registration.
func TestStreamManagerRegisterAgent(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	spawnTree := NewSpawnTree(store, "")
	sm := NewStreamManager(spawnTree, 0)

	agent := NewBuilderAgent("builder-1")

	// Register agent
	state, err := sm.RegisterAgent(agent)
	if err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}

	if state == nil {
		t.Fatal("State should not be nil")
	}

	if state.AgentName != "builder-1" {
		t.Errorf("Expected agent name 'builder-1', got %s", state.AgentName)
	}

	if state.Caste != CasteBuilder {
		t.Errorf("Expected caste %s, got %s", CasteBuilder, state.Caste)
	}

	// Verify registration
	if sm.TotalCount() != 1 {
		t.Errorf("Expected 1 total stream, got %d", sm.TotalCount())
	}

	// Try to register duplicate
	_, err = sm.RegisterAgent(agent)
	if err == nil {
		t.Error("Should error when registering duplicate agent")
	}

	// Get existing stream
	existingState, ok := sm.GetStream("builder-1")
	if !ok {
		t.Error("Should find existing stream")
	}

	if existingState != state {
		t.Error("GetStream should return the same state object")
	}
}

// TestStreamManagerMaxStreams tests the max streams limit.
func TestStreamManagerMaxStreams(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	spawnTree := NewSpawnTree(store, "")
	sm := NewStreamManager(spawnTree, 2)

	// Register first agent
	agent1 := NewBuilderAgent("builder-1")
	_, err = sm.RegisterAgent(agent1)
	if err != nil {
		t.Fatalf("Failed to register first agent: %v", err)
	}

	// Register second agent
	agent2 := NewBuilderAgent("builder-2")
	_, err = sm.RegisterAgent(agent2)
	if err != nil {
		t.Fatalf("Failed to register second agent: %v", err)
	}

	// Try to register third agent (should fail)
	agent3 := NewBuilderAgent("builder-3")
	_, err = sm.RegisterAgent(agent3)
	if err == nil {
		t.Error("Should error when exceeding max streams")
	}
}

// TestStreamManagerMultipleConcurrentStreams tests tracking 4+ concurrent streams.
func TestStreamManagerMultipleConcurrentStreams(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	spawnTree := NewSpawnTree(store, "")
	sm := NewStreamManager(spawnTree, 10)

	// Register 4 agents
	agents := []Agent{
		NewBuilderAgent("builder-1"),
		NewBuilderAgent("builder-2"),
		NewBuilderAgent("builder-3"),
		NewBuilderAgent("builder-4"),
	}

	states := make([]*StreamState, 0, 4)
	for _, agent := range agents {
		state, err := sm.RegisterAgent(agent)
		if err != nil {
			t.Fatalf("Failed to register agent %s: %v", agent.Name(), err)
		}
		states = append(states, state)
	}

	if sm.TotalCount() != 4 {
		t.Errorf("Expected 4 total streams, got %d", sm.TotalCount())
	}

	// Mark some as active
	states[0].SetStatus(StreamStatusActive)
	states[1].SetStatus(StreamStatusActive)

	if sm.ActiveCount() != 2 {
		t.Errorf("Expected 2 active streams, got %d", sm.ActiveCount())
	}

	// Get active streams
	active := sm.GetActiveStreams()
	if len(active) != 2 {
		t.Errorf("Expected 2 active streams in list, got %d", len(active))
	}

	// Mark one complete
	states[0].SetStatus(StreamStatusCompleted)

	completed := sm.GetCompletedStreams()
	if len(completed) != 1 {
		t.Errorf("Expected 1 completed stream, got %d", len(completed))
	}

	// Get all streams
	all := sm.GetAllStreams()
	if len(all) != 4 {
		t.Errorf("Expected 4 total streams in list, got %d", len(all))
	}
}

// TestStreamManagerWaitForAll tests waiting for all streams to complete.
func TestStreamManagerWaitForAll(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	spawnTree := NewSpawnTree(store, "")
	sm := NewStreamManager(spawnTree, 0)

	// Register agents
	for i := 0; i < 3; i++ {
		agent := NewBuilderAgent("builder-" + string(rune('1'+i)))
		state, _ := sm.RegisterAgent(agent)
		state.SetStatus(StreamStatusActive)
	}

	// Complete streams asynchronously
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			time.Sleep(time.Duration(10+idx*5) * time.Millisecond)
			state, _ := sm.GetStream("builder-" + string(rune('1'+idx)))
			state.SetStatus(StreamStatusCompleted)
		}(i)
	}

	// Wait for all
	results := sm.WaitForAll()

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	for i := 0; i < 3; i++ {
		name := "builder-" + string(rune('1'+i))
		state, ok := results[name]
		if !ok {
			t.Errorf("Missing result for %s", name)
			continue
		}
		if state.GetStatus() != StreamStatusCompleted {
			t.Errorf("Expected %s to be completed, got %s", name, state.GetStatus())
		}
	}

	wg.Wait()
}

// TestStreamManagerWaitForAgent tests waiting for a specific agent.
func TestStreamManagerWaitForAgent(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	spawnTree := NewSpawnTree(store, "")
	sm := NewStreamManager(spawnTree, 0)

	agent := NewBuilderAgent("builder-1")
	state, _ := sm.RegisterAgent(agent)
	state.SetStatus(StreamStatusActive)

	// Complete asynchronously
	go func() {
		time.Sleep(20 * time.Millisecond)
		state.SetStatus(StreamStatusCompleted)
	}()

	// Wait for specific agent
	result, ok := sm.WaitForAgent("builder-1")
	if !ok {
		t.Error("Should find agent")
	}

	if result.GetStatus() != StreamStatusCompleted {
		t.Errorf("Expected completed status, got %s", result.GetStatus())
	}

	// Try non-existent agent
	_, ok = sm.WaitForAgent("non-existent")
	if ok {
		t.Error("Should not find non-existent agent")
	}
}

// TestStreamManagerExecuteStreamingWithAgent tests executing with streaming.
func TestStreamManagerExecuteStreamingWithAgent(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	spawnTree := NewSpawnTree(store, "")
	sm := NewStreamManager(spawnTree, 0)

	agent := NewBuilderAgent("streaming-builder")
	event := events.Event{Topic: "build.start"}
	ctx := context.Background()

	state, err := sm.ExecuteStreamingWithAgent(ctx, agent, event)
	if err != nil {
		t.Fatalf("ExecuteStreamingWithAgent failed: %v", err)
	}

	if state == nil {
		t.Fatal("State should not be nil")
	}

	// Wait for completion
	finalState, _ := sm.WaitForAgent("streaming-builder")

	if !finalState.IsComplete() {
		t.Error("Stream should be complete")
	}

	// Verify tokens were captured
	if finalState.GetTokenCount() == 0 {
		t.Error("Should have captured some tokens")
	}

	// Verify result
	if finalState.GetResult() == nil {
		t.Error("Should have a result")
	}
}

// TestStreamManagerExecuteStreamingMultipleAgents tests concurrent streaming execution.
func TestStreamManagerExecuteStreamingMultipleAgents(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	spawnTree := NewSpawnTree(store, "")
	sm := NewStreamManager(spawnTree, 10)

	ctx := context.Background()
	event := events.Event{Topic: "build.start"}

	// Start 4 agents concurrently
	agents := []*BuilderAgent{
		NewBuilderAgent("builder-1"),
		NewBuilderAgent("builder-2"),
		NewBuilderAgent("builder-3"),
		NewBuilderAgent("builder-4"),
	}

	for _, agent := range agents {
		_, err := sm.ExecuteStreamingWithAgent(ctx, agent, event)
		if err != nil {
			t.Fatalf("Failed to start agent %s: %v", agent.Name(), err)
		}
	}

	// Wait for all to complete
	results := sm.WaitForAll()

	if len(results) != 4 {
		t.Errorf("Expected 4 results, got %d", len(results))
	}

	// Verify all completed successfully
	for _, agent := range agents {
		state, ok := results[agent.Name()]
		if !ok {
			t.Errorf("Missing result for %s", agent.Name())
			continue
		}

		if state.GetStatus() != StreamStatusCompleted {
			t.Errorf("Expected %s to be completed, got %s", agent.Name(), state.GetStatus())
		}

		if state.GetError() != nil {
			t.Errorf("Expected no error for %s, got %v", agent.Name(), state.GetError())
		}

		if state.GetResult() == nil {
			t.Errorf("Expected result for %s", agent.Name())
		}

		if state.GetTokenCount() == 0 {
			t.Errorf("Expected tokens for %s", agent.Name())
		}
	}
}

// TestStreamManagerGetStreamSummary tests summary generation.
func TestStreamManagerGetStreamSummary(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	spawnTree := NewSpawnTree(store, "")
	sm := NewStreamManager(spawnTree, 0)

	// Create mix of builders and watchers with different statuses
	agents := []struct {
		name   string
		caste  Caste
		status StreamStatus
		tokens int
	}{
		{"builder-1", CasteBuilder, StreamStatusActive, 10},
		{"builder-2", CasteBuilder, StreamStatusCompleted, 25},
		{"watcher-1", CasteWatcher, StreamStatusActive, 5},
		{"watcher-2", CasteWatcher, StreamStatusFailed, 15},
	}

	for _, a := range agents {
		agent := NewBuilderAgent(a.name)
		state, _ := sm.RegisterAgent(agent)
		state.Caste = a.caste
		state.SetStatus(a.status)
		for i := 0; i < a.tokens; i++ {
			state.AddToken("token")
		}
	}

	summary := sm.GetStreamSummary()

	if summary.Total != 4 {
		t.Errorf("Expected total 4, got %d", summary.Total)
	}

	if summary.Active != 2 {
		t.Errorf("Expected active 2, got %d", summary.Active)
	}

	if summary.Completed != 2 {
		t.Errorf("Expected completed 2, got %d", summary.Completed)
	}

	if summary.TotalTokens != 55 {
		t.Errorf("Expected total tokens 55, got %d", summary.TotalTokens)
	}

	if summary.ByStatus[StreamStatusActive] != 2 {
		t.Errorf("Expected 2 active by status, got %d", summary.ByStatus[StreamStatusActive])
	}

	if summary.ByCaste[CasteBuilder] != 2 {
		t.Errorf("Expected 2 builders, got %d", summary.ByCaste[CasteBuilder])
	}
}

// TestStreamManagerUnregisterAgent tests agent unregistration.
func TestStreamManagerUnregisterAgent(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	spawnTree := NewSpawnTree(store, "")
	sm := NewStreamManager(spawnTree, 0)

	agent := NewBuilderAgent("builder-1")
	sm.RegisterAgent(agent)

	if sm.TotalCount() != 1 {
		t.Errorf("Expected 1 stream before unregister, got %d", sm.TotalCount())
	}

	sm.UnregisterAgent("builder-1")

	if sm.TotalCount() != 0 {
		t.Errorf("Expected 0 streams after unregister, got %d", sm.TotalCount())
	}

	_, ok := sm.GetStream("builder-1")
	if ok {
		t.Error("Should not find unregistered agent")
	}
}

// TestStreamManagerConcurrentAccess tests thread safety.
func TestStreamManagerConcurrentAccess(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	spawnTree := NewSpawnTree(store, "")
	sm := NewStreamManager(spawnTree, 0)

	var wg sync.WaitGroup
	numAgents := 10

	// Concurrent registrations
	for i := 0; i < numAgents; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			agent := NewBuilderAgent("builder-" + string(rune('0'+idx)))
			sm.RegisterAgent(agent)
		}(i)
	}

	wg.Wait()

	if sm.TotalCount() != numAgents {
		t.Errorf("Expected %d streams, got %d", numAgents, sm.TotalCount())
	}

	// Concurrent token additions
	for i := 0; i < numAgents; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			state, ok := sm.GetStream("builder-" + string(rune('0'+idx)))
			if ok {
				for j := 0; j < 10; j++ {
					state.AddToken("token")
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify all tokens were added
	for i := 0; i < numAgents; i++ {
		state, ok := sm.GetStream("builder-" + string(rune('0'+i)))
		if !ok {
			t.Errorf("Missing agent %d", i)
			continue
		}
		if state.GetTokenCount() != 10 {
			t.Errorf("Expected 10 tokens for agent %d, got %d", i, state.GetTokenCount())
		}
	}
}

// TestStreamManagerContextCancellation tests cancellation during streaming.
func TestStreamManagerContextCancellation(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	spawnTree := NewSpawnTree(store, "")
	sm := NewStreamManager(spawnTree, 0)

	agent := NewBuilderAgent("cancelled-builder")
	event := events.Event{Topic: "build.start"}

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = sm.ExecuteStreamingWithAgent(ctx, agent, event)
	if err != nil {
		t.Fatalf("ExecuteStreamingWithAgent failed: %v", err)
	}

	// Wait for completion
	finalState, _ := sm.WaitForAgent("cancelled-builder")

	// Should have failed due to cancellation
	if finalState.GetStatus() != StreamStatusFailed {
		t.Errorf("Expected failed status due to cancellation, got %s", finalState.GetStatus())
	}

	if finalState.GetError() == nil {
		t.Error("Expected error from cancelled context")
	}
}

// TestStreamStateTokenIsolation tests that tokens are isolated per stream.
func TestStreamStateTokenIsolation(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	spawnTree := NewSpawnTree(store, "")
	sm := NewStreamManager(spawnTree, 0)

	ctx := context.Background()
	event := events.Event{Topic: "build.start"}

	// Start two agents
	agent1 := NewBuilderAgent("builder-1")
	agent2 := NewBuilderAgent("builder-2")

	state1, _ := sm.ExecuteStreamingWithAgent(ctx, agent1, event)
	state2, _ := sm.ExecuteStreamingWithAgent(ctx, agent2, event)

	// Wait for completion
	sm.WaitForAll()

	// Verify tokens are different
	tokens1 := state1.GetTokens()
	tokens2 := state2.GetTokens()

	if len(tokens1) == 0 || len(tokens2) == 0 {
		t.Error("Both streams should have tokens")
	}

	// Each stream should contain its agent name
	var content1, content2 string
	for _, t := range tokens1 {
		content1 += t
	}
	for _, t := range tokens2 {
		content2 += t
	}

	if !strings.Contains(content1, "builder-1") {
		t.Error("Stream 1 should contain 'builder-1'")
	}

	if !strings.Contains(content2, "builder-2") {
		t.Error("Stream 2 should contain 'builder-2'")
	}
}

// TestStreamManagerWithDifferentCastes tests tracking agents of different castes.
func TestStreamManagerWithDifferentCastes(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	spawnTree := NewSpawnTree(store, "")
	sm := NewStreamManager(spawnTree, 0)

	// Create agents of different castes
	castes := []Caste{CasteBuilder, CasteWatcher, CasteScout, CasteOracle}

	for i, caste := range castes {
		agent := &mockStreamingAgent{
			name:     string(caste) + "-" + string(rune('1'+i)),
			caste:    caste,
			triggers: []Trigger{{Topic: "test.*"}},
		}
		state, err := sm.RegisterAgent(agent)
		if err != nil {
			t.Fatalf("Failed to register %s: %v", caste, err)
		}

		if state.Caste != caste {
			t.Errorf("Expected caste %s, got %s", caste, state.Caste)
		}
	}

	if sm.TotalCount() != 4 {
		t.Errorf("Expected 4 streams, got %d", sm.TotalCount())
	}

	summary := sm.GetStreamSummary()
	if len(summary.ByCaste) != 4 {
		t.Errorf("Expected 4 different castes, got %d", len(summary.ByCaste))
	}
}

// mockStreamingAgent is a test agent that implements StreamingAgent.
type mockStreamingAgent struct {
	name     string
	caste    Caste
	triggers []Trigger
}

func (m *mockStreamingAgent) Name() string {
	return m.name
}

func (m *mockStreamingAgent) Caste() Caste {
	return m.caste
}

func (m *mockStreamingAgent) Triggers() []Trigger {
	return m.triggers
}

func (m *mockStreamingAgent) Execute(ctx context.Context, event events.Event) error {
	return nil
}

func (m *mockStreamingAgent) ExecuteStreaming(ctx context.Context, event events.Event, handler llm.StreamHandler) error {
	if handler != nil {
		handler.OnToken("Token from " + m.name)
		handler.OnComplete(&llm.StreamResult{
			Text:       "Result from " + m.name,
			Role:       "assistant",
			StopReason: "end_turn",
		})
	}
	return nil
}

// TestStreamManagerIntegrationWithSpawnTree tests spawn tree integration.
func TestStreamManagerIntegrationWithSpawnTree(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	spawnTree := NewSpawnTree(store, "")

	// Record some spawns in the tree
	spawnTree.RecordSpawn("queen-1", "builder", "builder-1", "task1", 1)
	spawnTree.RecordSpawn("queen-1", "builder", "builder-2", "task2", 1)

	sm := NewStreamManager(spawnTree, 0)

	// Register matching agents
	agent1 := NewBuilderAgent("builder-1")
	agent2 := NewBuilderAgent("builder-2")

	sm.RegisterAgent(agent1)
	sm.RegisterAgent(agent2)

	// Verify we can track them
	if sm.TotalCount() != 2 {
		t.Errorf("Expected 2 streams, got %d", sm.TotalCount())
	}

	// Verify spawn tree integration
	active := spawnTree.Active()
	if len(active) != 2 {
		t.Errorf("Expected 2 active in spawn tree, got %d", len(active))
	}
}
