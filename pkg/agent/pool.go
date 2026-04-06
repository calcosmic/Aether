package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/llm"
	"golang.org/x/sync/errgroup"
)

// defaultMaxGoroutines is the default concurrency limit for the worker pool.
const defaultMaxGoroutines = 4

// PoolOption configures a Pool during construction.
type PoolOption func(*Pool)

// WithMaxGoroutines sets the maximum number of concurrent agent executions.
// Panics if n <= 0.
func WithMaxGoroutines(n int) PoolOption {
	if n <= 0 {
		panic("pool: WithMaxGoroutines requires n > 0")
	}
	return func(p *Pool) {
		p.maxG = n
	}
}

// WithPoolStreaming enables streaming support for the pool.
// When enabled, the pool creates a StreamManager and routes streaming events
// from agents without blocking pool goroutines.
func WithPoolStreaming(streamMgr *StreamManager) PoolOption {
	return func(p *Pool) {
		p.streamMgr = streamMgr
		p.enableStream = true
	}
}

// Pool dispatches events from the bus to matching agents with bounded concurrency.
// It subscribes to the event bus, matches incoming events against registered agents,
// and runs matching agents using an errgroup with a configurable goroutine limit.
//
// Streaming Support:
// The pool supports streaming execution for agents that implement StreamingAgent.
// When streaming is enabled, the pool creates a StreamHandler per agent and
// subscribes to agent-specific topics (agent.{name}.*). Events flow from agents
// to the StreamManager without blocking pool goroutines.
type Pool struct {
	registry     *Registry
	bus          *events.Bus
	maxG         int
	eventCh      <-chan events.Event
	cancel       context.CancelFunc
	streamMgr    *StreamManager
	enableStream bool
	mu           sync.Mutex
}

// NewPool creates a worker pool that dispatches events to agents in the registry.
// The pool subscribes to all events ("*") on the bus.
// Returns an error if registry or bus is nil.
func NewPool(registry *Registry, bus *events.Bus, opts ...PoolOption) (*Pool, error) {
	if registry == nil {
		return nil, fmt.Errorf("pool: registry must not be nil")
	}
	if bus == nil {
		return nil, fmt.Errorf("pool: bus must not be nil")
	}

	p := &Pool{
		registry:     registry,
		bus:          bus,
		maxG:         defaultMaxGoroutines,
		enableStream: false,
	}

	for _, opt := range opts {
		opt(p)
	}

	return p, nil
}

// createStreamHandler creates a StreamHandler for an agent that publishes
// events to the bus without blocking the pool goroutine.
func (p *Pool) createStreamHandler(agent Agent) llm.StreamHandler {
	return &poolStreamHandler{
		agentName: agent.Name(),
		caste:     agent.Caste(),
		bus:       p.bus,
	}
}

// poolStreamHandler implements llm.StreamHandler and publishes events to the bus.
// This decouples the agent execution from the consumer, preventing slow consumers
// from blocking pool goroutines.
type poolStreamHandler struct {
	agentName string
	caste     Caste
	bus       *events.Bus
}

func (h *poolStreamHandler) OnToken(token string) {
	// Publish token event asynchronously - don't block on slow consumers
	payload := map[string]interface{}{
		"agent":   h.agentName,
		"caste":   string(h.caste),
		"type":    "token",
		"content": token,
	}
	h.publishEvent("token", payload)
}

func (h *poolStreamHandler) OnToolStart(toolName, toolID string) {
	payload := map[string]interface{}{
		"agent":   h.agentName,
		"caste":   string(h.caste),
		"type":    "tool_start",
		"tool":    toolName,
		"tool_id": toolID,
	}
	h.publishEvent("tool.start", payload)
}

func (h *poolStreamHandler) OnToolEnd(toolName, toolID, result string) {
	payload := map[string]interface{}{
		"agent":   h.agentName,
		"caste":   string(h.caste),
		"type":    "tool_end",
		"tool":    toolName,
		"tool_id": toolID,
		"result":  result,
	}
	h.publishEvent("tool.end", payload)
}

func (h *poolStreamHandler) OnComplete(result *llm.StreamResult) {
	payload := map[string]interface{}{
		"agent":       h.agentName,
		"caste":       string(h.caste),
		"type":        "complete",
		"text":        result.Text,
		"role":        result.Role,
		"model":       result.Model,
		"stop_reason": result.StopReason,
		"usage": map[string]int64{
			"input_tokens":  result.Usage.InputTokens,
			"output_tokens": result.Usage.OutputTokens,
		},
	}
	h.publishEvent("complete", payload)
}

func (h *poolStreamHandler) OnError(err error) {
	payload := map[string]interface{}{
		"agent":   h.agentName,
		"caste":   string(h.caste),
		"type":    "error",
		"message": err.Error(),
	}
	h.publishEvent("error", payload)
}

// publishEvent publishes an event to agent.{name}.{eventType} topic.
// Uses background context since the pool's context may be cancelled.
// Events are published asynchronously and won't block on slow consumers.
func (h *poolStreamHandler) publishEvent(eventType string, payload map[string]interface{}) {
	if h.bus == nil {
		return
	}

	topic := fmt.Sprintf("agent.%s.%s", h.agentName, eventType)

	// Use a goroutine to avoid blocking the agent execution
	// This ensures slow consumers don't block pool goroutines
	go func() {
		// Serialize payload
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return
		}

		// Publish with background context - don't let pool context cancellation stop event publishing
		ctx := context.Background()
		_, _ = h.bus.Publish(ctx, topic, payloadBytes, h.agentName)
	}()
}

// Start begins consuming events from the bus and dispatching them to matching agents.
// It blocks until the provided context is cancelled or the event channel is closed.
// For each event, matching agents are found via registry.Match and run concurrently
// using errgroup with SetLimit to bound concurrency.
func (p *Pool) Start(ctx context.Context) error {
	// Subscribe to all events
	ch, err := p.bus.Subscribe("*")
	if err != nil {
		return fmt.Errorf("pool: subscribe: %w", err)
	}

	// Create derived context with cancel -- must be protected for concurrent Stop() calls
	ctx, cancel := context.WithCancel(ctx)
	p.mu.Lock()
	p.eventCh = ch
	p.cancel = cancel
	p.mu.Unlock()

	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-ch:
			if !ok {
				// Channel closed
				return nil
			}
			p.dispatch(ctx, event)
		}
	}
}

// dispatch runs all matching agents for a single event with bounded concurrency.
// It blocks until all matching agents have completed (or the context is cancelled).
//
// Streaming Support:
// When streaming is enabled and an agent implements StreamingAgent, the pool
// creates a StreamHandler that publishes events to agent.{name}.* topics.
// This allows real-time progress without blocking pool goroutines.
func (p *Pool) dispatch(ctx context.Context, event events.Event) {
	p.mu.Lock()
	maxG := p.maxG
	enableStream := p.enableStream
	streamMgr := p.streamMgr
	p.mu.Unlock()

	agents := p.registry.Match(event.Topic)
	if len(agents) == 0 {
		return
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(maxG)

	for _, a := range agents {
		agent := a // capture loop variable
		g.Go(func() error {
			// Check if streaming is enabled and agent supports it
			if enableStream {
				if sa, ok := IsStreamingAgent(agent); ok {
					// Create stream handler that publishes to bus
					handler := p.createStreamHandler(agent)

					// Register with stream manager if available
					if streamMgr != nil {
						_, _ = streamMgr.RegisterAgent(agent)
					}

					// Execute with streaming
					err := sa.ExecuteStreaming(ctx, event, handler)

					// Update stream manager state if available
					if streamMgr != nil {
						if state, ok := streamMgr.GetStream(agent.Name()); ok {
							if err != nil {
								state.SetError(err)
							} else if !state.IsComplete() {
								state.SetStatus(StreamStatusCompleted)
							}
						}
					}

					return err
				}
			}

			// Fall back to regular execution (backward compatible)
			return agent.Execute(ctx, event)
		})
	}

	// Wait for all agents to complete before processing next event.
	// Errors from individual agents are logged but do not stop the pool.
	_ = g.Wait()
}

// Stop cancels the pool's context and unsubscribes from the event bus.
func (p *Pool) Stop() {
	p.mu.Lock()
	cancel := p.cancel
	ch := p.eventCh
	p.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if ch != nil {
		p.bus.Unsubscribe("*", ch)
	}
}

// SetConcurrency updates the maximum goroutine count for subsequent event dispatches.
func (p *Pool) SetConcurrency(n int) {
	if n <= 0 {
		return
	}
	p.mu.Lock()
	p.maxG = n
	p.mu.Unlock()
}

// StreamManager returns the pool's stream manager (may be nil if streaming not enabled).
func (p *Pool) StreamManager() *StreamManager {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.streamMgr
}

// IsStreamingEnabled returns true if the pool has streaming support enabled.
func (p *Pool) IsStreamingEnabled() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.enableStream
}

// EnableStreaming enables streaming support with the given StreamManager.
// This can be called after pool creation to enable streaming dynamically.
func (p *Pool) EnableStreaming(streamMgr *StreamManager) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.streamMgr = streamMgr
	p.enableStream = true
}
