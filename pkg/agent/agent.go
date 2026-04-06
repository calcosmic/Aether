// Package agent implements the worker pool for managing colony agents,
// including spawn, lifecycle, and task distribution.
package agent

import (
	"context"
	"sort"
	"sync"

	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/llm"
)

// Caste represents the role category of an agent, matching the shell caste names.
type Caste string

const (
	CasteBuilder       Caste = "builder"
	CasteWatcher       Caste = "watcher"
	CasteScout         Caste = "scout"
	CasteOracle        Caste = "oracle"
	CasteCurator       Caste = "curator"
	CasteArchitect     Caste = "architect"
	CasteRouteSetter   Caste = "route_setter"
	CasteColonizer     Caste = "colonizer"
	CasteArchaeologist Caste = "archaeologist"
)

// Trigger defines when an agent should be activated.
type Trigger struct {
	// Topic is an event bus topic pattern, e.g. "learning.*".
	Topic string
	// Filter is an optional payload filter (may be nil).
	Filter map[string]any
}

// Agent is the interface every colony agent must implement.
type Agent interface {
	// Name returns the unique identifier for this agent.
	Name() string
	// Caste returns the role category of this agent.
	Caste() Caste
	// Triggers returns the event patterns that activate this agent.
	Triggers() []Trigger
	// Execute runs the agent's logic for the given event.
	Execute(ctx context.Context, event events.Event) error
}

// ExecuteOption configures agent execution behavior.
// Use WithStreaming to enable streaming mode.
type ExecuteOption func(*executeConfig)

type executeConfig struct {
	streamHandler llm.StreamHandler
}

// WithStreaming enables streaming mode for agent execution.
// The provided handler receives real-time token callbacks during execution.
// When streaming is enabled, the agent can stream its output progressively
// instead of buffering everything until completion.
func WithStreaming(handler llm.StreamHandler) ExecuteOption {
	return func(cfg *executeConfig) {
		cfg.streamHandler = handler
	}
}

// StreamingAgent extends Agent with streaming execution capabilities.
// Agents that support real-time output should implement this interface.
type StreamingAgent interface {
	Agent
	// ExecuteStreaming runs the agent's logic with streaming support.
	// The handler receives callbacks for each token/tool event.
	// Context cancellation is propagated to the stream.
	ExecuteStreaming(ctx context.Context, event events.Event, handler llm.StreamHandler) error
}

// Registry manages registered agents with thread-safe access.
type Registry struct {
	agents map[string]Agent
	mu     sync.RWMutex
}

// NewRegistry creates an empty agent registry.
func NewRegistry() *Registry {
	return &Registry{
		agents: make(map[string]Agent),
	}
}

// Register adds an agent to the registry.
// Returns an error if an agent with the same name is already registered.
func (r *Registry) Register(a Agent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := a.Name()
	if _, exists := r.agents[name]; exists {
		return &DuplicateAgentError{Name: name}
	}
	r.agents[name] = a
	return nil
}

// Get retrieves an agent by name.
// Returns an error if no agent with the given name exists.
func (r *Registry) Get(name string) (Agent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	a, ok := r.agents[name]
	if !ok {
		return nil, &AgentNotFoundError{Name: name}
	}
	return a, nil
}

// List returns all registered agents sorted by name.
func (r *Registry) List() []Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Agent, 0, len(r.agents))
	for _, a := range r.agents {
		result = append(result, a)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name() < result[j].Name()
	})
	return result
}

// Match returns agents whose Triggers match the given topic using events.TopicMatch.
func (r *Registry) Match(topic string) []Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matched []Agent
	for _, a := range r.agents {
		for _, trigger := range a.Triggers() {
			if events.TopicMatch(trigger.Topic, topic) {
				matched = append(matched, a)
				break
			}
		}
	}
	sort.Slice(matched, func(i, j int) bool {
		return matched[i].Name() < matched[j].Name()
	})
	return matched
}

// DuplicateAgentError is returned when registering an agent with a name
// that is already present in the registry.
type DuplicateAgentError struct {
	Name string
}

func (e *DuplicateAgentError) Error() string {
	return "agent already registered: " + e.Name
}

// AgentNotFoundError is returned when looking up an agent that does not exist.
type AgentNotFoundError struct {
	Name string
}

func (e *AgentNotFoundError) Error() string {
	return "agent not found: " + e.Name
}

// IsStreamingAgent checks if the given agent supports streaming execution.
func IsStreamingAgent(a Agent) (StreamingAgent, bool) {
	sa, ok := a.(StreamingAgent)
	return sa, ok
}

// ExecuteWithOptions runs the agent with the provided execution options.
// If the agent supports streaming and WithStreaming is provided, it uses
// ExecuteStreaming; otherwise it falls back to regular Execute.
// Context cancellation is always propagated to the underlying execution.
func ExecuteWithOptions(ctx context.Context, a Agent, event events.Event, opts ...ExecuteOption) error {
	cfg := &executeConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	// If streaming is requested and agent supports it, use streaming path
	if cfg.streamHandler != nil {
		if sa, ok := a.(StreamingAgent); ok {
			return sa.ExecuteStreaming(ctx, event, cfg.streamHandler)
		}
	}

	// Fall back to regular execute (backward compatible)
	return a.Execute(ctx, event)
}
