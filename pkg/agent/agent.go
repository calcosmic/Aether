// Package agent implements the worker pool for managing colony agents,
// including spawn, lifecycle, and task distribution.
package agent

import (
	"context"
	"sort"
	"sync"

	"github.com/aether-colony/aether/pkg/events"
)

// Caste represents the role category of an agent, matching the shell caste names.
type Caste string

const (
	CasteBuilder      Caste = "builder"
	CasteWatcher      Caste = "watcher"
	CasteScout        Caste = "scout"
	CasteOracle       Caste = "oracle"
	CasteCurator      Caste = "curator"
	CasteArchitect    Caste = "architect"
	CasteRouteSetter  Caste = "route_setter"
	CasteColonizer    Caste = "colonizer"
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
