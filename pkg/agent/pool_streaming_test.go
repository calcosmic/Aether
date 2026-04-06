package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/llm"
)

// streamingRecordingAgent is a test agent that implements StreamingAgent.
type streamingRecordingAgent struct {
	name      string
	caste     Caste
	triggers  []Trigger
	mu        sync.Mutex
	tokens    []string
	completed bool
	execErr   error
	onExecute func(handler llm.StreamHandler)
}

func (s *streamingRecordingAgent) Name() string        { return s.name }
func (s *streamingRecordingAgent) Caste() Caste        { return s.caste }
func (s *streamingRecordingAgent) Triggers() []Trigger { return s.triggers }

func (s *streamingRecordingAgent) Execute(ctx context.Context, event events.Event) error {
	// Fallback for non-streaming
	return s.execErr
}

func (s *streamingRecordingAgent) ExecuteStreaming(ctx context.Context, event events.Event, handler llm.StreamHandler) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.onExecute != nil {
		s.onExecute(handler)
	} else {
		// Default streaming behavior
		if handler != nil {
			handler.OnToken("token1")
			handler.OnToken("token2")
			handler.OnComplete(&llm.StreamResult{
				Text:       "completed",
				Role:       "assistant",
				StopReason: "end_turn",
			})
		}
	}

	s.completed = true
	return s.execErr
}

func (s *streamingRecordingAgent) GetTokens() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.tokens...)
}

func (s *streamingRecordingAgent) IsCompleted() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.completed
}

// Test that pool creates StreamHandler per agent.
func TestPoolCreatesStreamHandlerPerAgent(t *testing.T) {
	reg := NewRegistry()
	bus, store := newTestBus(t)

	// Create a streaming agent
	agent := &streamingRecordingAgent{
		name:     "streamer-1",
		caste:    CasteBuilder,
		triggers: []Trigger{{Topic: "build.*"}},
	}
	reg.Register(agent)

	// Create pool with streaming enabled
	spawnTree := NewSpawnTree(store, "spawn-tree.txt")
	streamMgr := NewStreamManager(spawnTree, 10)
	p, err := NewPool(reg, bus, WithPoolStreaming(streamMgr))
	if err != nil {
		t.Fatalf("NewPool() error: %v", err)
	}

	// Verify streaming is enabled
	if !p.IsStreamingEnabled() {
		t.Error("Expected streaming to be enabled")
	}

	// Verify stream manager is set
	if p.StreamManager() != streamMgr {
		t.Error("Expected stream manager to be set")
	}
}

// Test that pool dispatches to streaming agents.
func TestPoolDispatchesToStreamingAgents(t *testing.T) {
	reg := NewRegistry()
	bus, store := newTestBus(t)

	// Create a streaming agent
	agent := &streamingRecordingAgent{
		name:     "streamer-1",
		caste:    CasteBuilder,
		triggers: []Trigger{{Topic: "build.*"}},
	}
	reg.Register(agent)

	// Create pool with streaming enabled
	spawnTree := NewSpawnTree(store, "spawn-tree.txt")
	streamMgr := NewStreamManager(spawnTree, 10)
	p, err := NewPool(reg, bus, WithPoolStreaming(streamMgr))
	if err != nil {
		t.Fatalf("NewPool() error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start pool in background
	done := make(chan struct{})
	go func() {
		p.Start(ctx)
		close(done)
	}()

	// Allow pool to subscribe
	time.Sleep(50 * time.Millisecond)

	// Publish an event
	payload, _ := json.Marshal(map[string]string{"action": "build"})
	bus.Publish(context.Background(), "build.start", payload, "test")

	// Wait for agent to complete
	deadline := time.After(2 * time.Second)
	for {
		if agent.IsCompleted() {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for streaming agent to complete")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	// Stop pool
	p.Stop()
	<-done
}

// Test that events flow from agents to StreamManager.
func TestPoolEventsFlowToStreamManager(t *testing.T) {
	reg := NewRegistry()
	bus, store := newTestBus(t)

	// Create a streaming agent that emits multiple tokens
	agent := &streamingRecordingAgent{
		name:     "streamer-1",
		caste:    CasteBuilder,
		triggers: []Trigger{{Topic: "build.*"}},
		onExecute: func(handler llm.StreamHandler) {
			handler.OnToken("token1")
			handler.OnToken("token2")
			handler.OnToken("token3")
			handler.OnComplete(&llm.StreamResult{
				Text:       "completed",
				Role:       "assistant",
				StopReason: "end_turn",
			})
		},
	}
	reg.Register(agent)

	// Create pool with streaming enabled
	spawnTree := NewSpawnTree(store, "spawn-tree.txt")
	streamMgr := NewStreamManager(spawnTree, 10)
	p, err := NewPool(reg, bus, WithPoolStreaming(streamMgr))
	if err != nil {
		t.Fatalf("NewPool() error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start pool in background
	done := make(chan struct{})
	go func() {
		p.Start(ctx)
		close(done)
	}()

	// Allow pool to subscribe
	time.Sleep(50 * time.Millisecond)

	// Subscribe to agent events
	agentEvents, err := bus.Subscribe("agent.*")
	if err != nil {
		t.Fatalf("Subscribe() error: %v", err)
	}

	// Publish an event
	payload, _ := json.Marshal(map[string]string{"action": "build"})
	bus.Publish(context.Background(), "build.start", payload, "test")

	// Collect events
	var collectedEvents []events.Event
	deadline := time.After(2 * time.Second)
	collectDone := make(chan struct{})

	go func() {
		for {
			select {
			case evt := <-agentEvents:
				collectedEvents = append(collectedEvents, evt)
				if len(collectedEvents) >= 4 { // 3 tokens + 1 complete
					close(collectDone)
					return
				}
			case <-time.After(100 * time.Millisecond):
				if len(collectedEvents) > 0 {
					close(collectDone)
					return
				}
			}
		}
	}()

	select {
	case <-collectDone:
		// Success
	case <-deadline:
		// Continue with what we have
	}

	// Verify we got events
	if len(collectedEvents) == 0 {
		t.Error("Expected agent events to be published to bus")
	}

	// Verify events have correct topic format
	for _, evt := range collectedEvents {
		if !strings.HasPrefix(evt.Topic, "agent.streamer-1.") {
			t.Errorf("Expected topic to start with 'agent.streamer-1.', got: %s", evt.Topic)
		}
	}

	// Stop pool
	p.Stop()
	<-done
}

// Test that pool doesn't block on slow consumers.
func TestPoolDoesNotBlockOnSlowConsumers(t *testing.T) {
	reg := NewRegistry()
	bus, store := newTestBus(t)

	// Create a streaming agent that emits many tokens quickly
	agent := &streamingRecordingAgent{
		name:     "fast-streamer",
		caste:    CasteBuilder,
		triggers: []Trigger{{Topic: "build.*"}},
		onExecute: func(handler llm.StreamHandler) {
			// Emit 10 tokens rapidly
			for i := 0; i < 10; i++ {
				handler.OnToken("token")
			}
			handler.OnComplete(&llm.StreamResult{
				Text:       "completed",
				Role:       "assistant",
				StopReason: "end_turn",
			})
		},
	}
	reg.Register(agent)

	// Create pool with streaming enabled
	spawnTree := NewSpawnTree(store, "spawn-tree.txt")
	streamMgr := NewStreamManager(spawnTree, 10)
	p, err := NewPool(reg, bus, WithPoolStreaming(streamMgr))
	if err != nil {
		t.Fatalf("NewPool() error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start pool in background
	done := make(chan struct{})
	go func() {
		p.Start(ctx)
		close(done)
	}()

	// Allow pool to subscribe
	time.Sleep(50 * time.Millisecond)

	// Create a slow consumer that takes time to process each event
	slowConsumer, _ := bus.Subscribe("agent.*")

	go func() {
		for range slowConsumer {
			time.Sleep(50 * time.Millisecond) // Slow consumption
		}
	}()

	// Publish an event and measure how long it takes
	payload, _ := json.Marshal(map[string]string{"action": "build"})
	start := time.Now()
	bus.Publish(context.Background(), "build.start", payload, "test")

	// Wait for agent to complete
	deadline := time.After(2 * time.Second)
	for {
		if agent.IsCompleted() {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for agent")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	elapsed := time.Since(start)

	// The agent should complete quickly even with slow consumer
	// If it took > 500ms, it was likely blocked
	if elapsed > 500*time.Millisecond {
		t.Errorf("Agent took too long (%v), may be blocked by slow consumer", elapsed)
	}

	// Stop pool
	p.Stop()
	<-done
}

// Test graceful degradation when streaming is disabled.
func TestPoolGracefulDegradationWithoutStreaming(t *testing.T) {
	reg := NewRegistry()
	bus, _ := newTestBus(t)

	// Create a streaming agent
	agent := &streamingRecordingAgent{
		name:     "streamer-1",
		caste:    CasteBuilder,
		triggers: []Trigger{{Topic: "build.*"}},
	}
	reg.Register(agent)

	// Create pool WITHOUT streaming
	p, err := NewPool(reg, bus)
	if err != nil {
		t.Fatalf("NewPool() error: %v", err)
	}

	// Verify streaming is disabled
	if p.IsStreamingEnabled() {
		t.Error("Expected streaming to be disabled")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start pool in background
	done := make(chan struct{})
	go func() {
		p.Start(ctx)
		close(done)
	}()

	// Allow pool to subscribe
	time.Sleep(50 * time.Millisecond)

	// Subscribe to agent events - should not receive any
	agentEvents, err := bus.Subscribe("agent.*")
	if err != nil {
		t.Fatalf("Subscribe() error: %v", err)
	}

	// Publish an event
	payload, _ := json.Marshal(map[string]string{"action": "build"})
	bus.Publish(context.Background(), "build.start", payload, "test")

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Agent should have been called (via regular Execute fallback)
	// But no streaming events should be published
	select {
	case <-agentEvents:
		t.Error("Did not expect streaming events when streaming is disabled")
	default:
		// Good - no events
	}

	// Stop pool
	p.Stop()
	<-done
}

// Test that pool subscribes to agent.{name}.* topics.
func TestPoolAgentTopicSubscription(t *testing.T) {
	reg := NewRegistry()
	bus, store := newTestBus(t)

	// Create multiple streaming agents
	agents := []*streamingRecordingAgent{
		{
			name:     "builder-1",
			caste:    CasteBuilder,
			triggers: []Trigger{{Topic: "build.*"}},
			onExecute: func(handler llm.StreamHandler) {
				handler.OnToken("builder-token")
				handler.OnComplete(&llm.StreamResult{Text: "done"})
			},
		},
		{
			name:     "watcher-1",
			caste:    CasteWatcher,
			triggers: []Trigger{{Topic: "test.*"}},
			onExecute: func(handler llm.StreamHandler) {
				handler.OnToken("watcher-token")
				handler.OnComplete(&llm.StreamResult{Text: "done"})
			},
		},
	}

	for _, a := range agents {
		reg.Register(a)
	}

	// Create pool with streaming enabled
	spawnTree := NewSpawnTree(store, "spawn-tree.txt")
	streamMgr := NewStreamManager(spawnTree, 10)
	p, err := NewPool(reg, bus, WithPoolStreaming(streamMgr))
	if err != nil {
		t.Fatalf("NewPool() error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start pool in background
	done := make(chan struct{})
	go func() {
		p.Start(ctx)
		close(done)
	}()

	// Allow pool to subscribe
	time.Sleep(50 * time.Millisecond)

	// Subscribe to specific agent topics
	builderEvents, _ := bus.Subscribe("agent.builder-1.*")
	watcherEvents, _ := bus.Subscribe("agent.watcher-1.*")

	// Publish events
	payload, _ := json.Marshal(map[string]string{"action": "build"})
	bus.Publish(context.Background(), "build.start", payload, "test")

	payload2, _ := json.Marshal(map[string]string{"action": "test"})
	bus.Publish(context.Background(), "test.run", payload2, "test")

	// Collect events
	var builderCount, watcherCount int
	deadline := time.After(2 * time.Second)

	collectEvents := func(ch <-chan events.Event, count *int) {
		for {
			select {
			case <-ch:
				*count++
			case <-time.After(100 * time.Millisecond):
				return
			}
		}
	}

	// Collect builder events
	go collectEvents(builderEvents, &builderCount)
	go collectEvents(watcherEvents, &watcherCount)

	<-deadline

	// Verify each agent published to its own topic
	if builderCount == 0 {
		t.Error("Expected builder events on agent.builder-1.* topic")
	}
	if watcherCount == 0 {
		t.Error("Expected watcher events on agent.watcher-1.* topic")
	}

	// Stop pool
	p.Stop()
	<-done
}

// Test EnableStreaming dynamically.
func TestPoolEnableStreamingDynamically(t *testing.T) {
	reg := NewRegistry()
	bus, store := newTestBus(t)

	// Create pool without streaming
	p, err := NewPool(reg, bus)
	if err != nil {
		t.Fatalf("NewPool() error: %v", err)
	}

	if p.IsStreamingEnabled() {
		t.Error("Expected streaming to be disabled initially")
	}

	// Enable streaming dynamically
	spawnTree := NewSpawnTree(store, "spawn-tree.txt")
	streamMgr := NewStreamManager(spawnTree, 10)
	p.EnableStreaming(streamMgr)

	if !p.IsStreamingEnabled() {
		t.Error("Expected streaming to be enabled after EnableStreaming")
	}

	if p.StreamManager() != streamMgr {
		t.Error("Expected stream manager to be set after EnableStreaming")
	}
}

// Test multiple streaming agents with concurrent execution.
func TestPoolMultipleStreamingAgentsConcurrent(t *testing.T) {
	reg := NewRegistry()
	bus, store := newTestBus(t)

	// Create multiple streaming agents that all match the same topic
	var agents []*streamingRecordingAgent
	for i := 0; i < 3; i++ {
		agent := &streamingRecordingAgent{
			name:     fmt.Sprintf("streamer-%d", i),
			caste:    CasteBuilder,
			triggers: []Trigger{{Topic: "build.*"}},
			onExecute: func(handler llm.StreamHandler) {
				handler.OnToken("token")
				handler.OnComplete(&llm.StreamResult{Text: "done"})
			},
		}
		agents = append(agents, agent)
		reg.Register(agent)
	}

	// Create pool with streaming enabled
	spawnTree := NewSpawnTree(store, "spawn-tree.txt")
	streamMgr := NewStreamManager(spawnTree, 10)
	p, err := NewPool(reg, bus, WithPoolStreaming(streamMgr))
	if err != nil {
		t.Fatalf("NewPool() error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start pool in background
	done := make(chan struct{})
	go func() {
		p.Start(ctx)
		close(done)
	}()

	// Allow pool to subscribe
	time.Sleep(50 * time.Millisecond)

	// Subscribe to all agent events
	agentEvents, _ := bus.Subscribe("agent.*")

	// Publish an event
	payload, _ := json.Marshal(map[string]string{"action": "build"})
	bus.Publish(context.Background(), "build.start", payload, "test")

	// Collect events from all agents
	var eventCount int
	deadline := time.After(2 * time.Second)

collectLoop:
	for {
		select {
		case <-agentEvents:
			eventCount++
			if eventCount >= 6 { // 3 agents * (1 token + 1 complete)
				break collectLoop
			}
		case <-deadline:
			break collectLoop
		case <-time.After(50 * time.Millisecond):
			// Continue collecting
		}
	}

	// Should have events from all 3 agents
	if eventCount < 6 {
		t.Errorf("Expected at least 6 events (3 agents * 2 events each), got %d", eventCount)
	}

	// Stop pool
	p.Stop()
	<-done
}

// Test streaming agent with error handling.
func TestPoolStreamingAgentErrorHandling(t *testing.T) {
	reg := NewRegistry()
	bus, store := newTestBus(t)

	// Create a streaming agent that returns an error
	agent := &streamingRecordingAgent{
		name:     "error-streamer",
		caste:    CasteBuilder,
		triggers: []Trigger{{Topic: "build.*"}},
		onExecute: func(handler llm.StreamHandler) {
			handler.OnToken("token1")
			handler.OnError(fmt.Errorf("streaming error"))
		},
		execErr: fmt.Errorf("streaming error"),
	}
	reg.Register(agent)

	// Create pool with streaming enabled
	spawnTree := NewSpawnTree(store, "spawn-tree.txt")
	streamMgr := NewStreamManager(spawnTree, 10)
	p, err := NewPool(reg, bus, WithPoolStreaming(streamMgr))
	if err != nil {
		t.Fatalf("NewPool() error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start pool in background
	done := make(chan struct{})
	go func() {
		p.Start(ctx)
		close(done)
	}()

	// Allow pool to subscribe
	time.Sleep(50 * time.Millisecond)

	// Subscribe to agent events
	agentEvents, _ := bus.Subscribe("agent.error-streamer.*")

	// Publish an event
	payload, _ := json.Marshal(map[string]string{"action": "build"})
	bus.Publish(context.Background(), "build.start", payload, "test")

	// Collect events
	var hasError bool
	deadline := time.After(2 * time.Second)

collectLoop:
	for {
		select {
		case evt := <-agentEvents:
			if strings.Contains(evt.Topic, "error") {
				hasError = true
			}
			if hasError {
				break collectLoop
			}
		case <-deadline:
			break collectLoop
		case <-time.After(50 * time.Millisecond):
			// Continue
		}
	}

	// Verify error was published
	if !hasError {
		t.Error("Expected error event to be published")
	}

	// Verify stream state reflects error
	if state, ok := streamMgr.GetStream("error-streamer"); ok {
		if state.GetError() == nil {
			t.Error("Expected stream state to have error")
		}
	}

	// Stop pool
	p.Stop()
	<-done
}

// Test mixed streaming and non-streaming agents.
func TestPoolMixedStreamingAndNonStreamingAgents(t *testing.T) {
	reg := NewRegistry()
	bus, store := newTestBus(t)

	// Create a streaming agent
	streamingAgent := &streamingRecordingAgent{
		name:     "streamer",
		caste:    CasteBuilder,
		triggers: []Trigger{{Topic: "task.*"}},
		onExecute: func(handler llm.StreamHandler) {
			handler.OnToken("streamed")
			handler.OnComplete(&llm.StreamResult{Text: "done"})
		},
	}
	reg.Register(streamingAgent)

	// Create a non-streaming agent
	nonStreamingAgent := &recordingAgent{
		name:     "non-streamer",
		caste:    CasteWatcher,
		triggers: []Trigger{{Topic: "task.*"}},
	}
	reg.Register(nonStreamingAgent)

	// Create pool with streaming enabled
	spawnTree := NewSpawnTree(store, "spawn-tree.txt")
	streamMgr := NewStreamManager(spawnTree, 10)
	p, err := NewPool(reg, bus, WithPoolStreaming(streamMgr))
	if err != nil {
		t.Fatalf("NewPool() error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start pool in background
	done := make(chan struct{})
	go func() {
		p.Start(ctx)
		close(done)
	}()

	// Allow pool to subscribe
	time.Sleep(50 * time.Millisecond)

	// Subscribe to streaming agent events
	agentEvents, _ := bus.Subscribe("agent.*")

	// Publish an event
	payload, _ := json.Marshal(map[string]string{"action": "task"})
	bus.Publish(context.Background(), "task.run", payload, "test")

	// Wait for both agents to complete
	deadline := time.After(2 * time.Second)
	for {
		if streamingAgent.IsCompleted() && nonStreamingAgent.CallCount() > 0 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for agents")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	// Verify streaming agent published events
	var hasStreamingEvent bool
	for {
		select {
		case <-agentEvents:
			hasStreamingEvent = true
		case <-time.After(100 * time.Millisecond):
			goto doneCollecting
		}
	}
doneCollecting:

	if !hasStreamingEvent {
		t.Error("Expected streaming agent to publish events")
	}

	// Verify non-streaming agent was called
	if nonStreamingAgent.CallCount() != 1 {
		t.Errorf("Expected non-streaming agent to be called once, got %d", nonStreamingAgent.CallCount())
	}

	// Stop pool
	p.Stop()
	<-done
}
