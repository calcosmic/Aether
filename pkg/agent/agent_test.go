package agent

import (
	"context"
	"testing"

	"github.com/aether-colony/aether/pkg/events"
)

// mockAgent is a test double implementing the Agent interface.
type mockAgent struct {
	name     string
	caste    Caste
	triggers []Trigger
	execErr  error
}

func (m *mockAgent) Name() string                { return m.name }
func (m *mockAgent) Caste() Caste                { return m.caste }
func (m *mockAgent) Triggers() []Trigger         { return m.triggers }
func (m *mockAgent) Execute(_ context.Context, _ events.Event) error {
	return m.execErr
}

func TestCasteConstants(t *testing.T) {
	cases := []struct {
		caste Caste
		want  string
	}{
		{CasteBuilder, "builder"},
		{CasteWatcher, "watcher"},
		{CasteScout, "scout"},
		{CasteOracle, "oracle"},
		{CasteCurator, "curator"},
		{CasteArchitect, "architect"},
		{CasteRouteSetter, "route_setter"},
		{CasteColonizer, "colonizer"},
		{CasteArchaeologist, "archaeologist"},
	}
	for _, tc := range cases {
		if string(tc.caste) != tc.want {
			t.Errorf("Caste constant %q = %q, want %q", tc.caste, string(tc.caste), tc.want)
		}
	}
}

func TestAgentInterface(t *testing.T) {
	a := &mockAgent{
		name:  "test-agent",
		caste: CasteBuilder,
		triggers: []Trigger{
			{Topic: "learning.*"},
		},
	}

	if a.Name() != "test-agent" {
		t.Errorf("Name() = %q, want %q", a.Name(), "test-agent")
	}
	if a.Caste() != CasteBuilder {
		t.Errorf("Caste() = %q, want %q", a.Caste(), CasteBuilder)
	}
	if len(a.Triggers()) != 1 {
		t.Fatalf("Triggers() returned %d triggers, want 1", len(a.Triggers()))
	}
	if a.Triggers()[0].Topic != "learning.*" {
		t.Errorf("Triggers()[0].Topic = %q, want %q", a.Triggers()[0].Topic, "learning.*")
	}
	if err := a.Execute(context.Background(), events.Event{}); err != nil {
		t.Errorf("Execute() returned unexpected error: %v", err)
	}
}

func TestRegistryRegister(t *testing.T) {
	r := NewRegistry()
	a := &mockAgent{name: "builder-1", caste: CasteBuilder}

	if err := r.Register(a); err != nil {
		t.Fatalf("Register() returned unexpected error: %v", err)
	}

	list := r.List()
	if len(list) != 1 {
		t.Fatalf("List() returned %d agents, want 1", len(list))
	}
	if list[0].Name() != "builder-1" {
		t.Errorf("List()[0].Name() = %q, want %q", list[0].Name(), "builder-1")
	}
}

func TestRegistryDuplicate(t *testing.T) {
	r := NewRegistry()
	a := &mockAgent{name: "builder-1", caste: CasteBuilder}

	if err := r.Register(a); err != nil {
		t.Fatalf("first Register() returned unexpected error: %v", err)
	}

	err := r.Register(a)
	if err == nil {
		t.Fatal("second Register() should return error for duplicate")
	}

	if _, ok := err.(*DuplicateAgentError); !ok {
		t.Errorf("error type = %T, want *DuplicateAgentError", err)
	}
}

func TestRegistryGet(t *testing.T) {
	r := NewRegistry()
	a := &mockAgent{name: "scout-1", caste: CasteScout}
	r.Register(a)

	got, err := r.Get("scout-1")
	if err != nil {
		t.Fatalf("Get(\"scout-1\") returned unexpected error: %v", err)
	}
	if got.Name() != "scout-1" {
		t.Errorf("Get(\"scout-1\").Name() = %q, want %q", got.Name(), "scout-1")
	}

	_, err = r.Get("unknown")
	if err == nil {
		t.Fatal("Get(\"unknown\") should return error for missing agent")
	}
	if _, ok := err.(*AgentNotFoundError); !ok {
		t.Errorf("error type = %T, want *AgentNotFoundError", err)
	}
}

func TestRegistryMatch(t *testing.T) {
	r := NewRegistry()
	a := &mockAgent{
		name:  "learning-agent",
		caste: CasteCurator,
		triggers: []Trigger{
			{Topic: "learning.*"},
		},
	}
	r.Register(a)

	matched := r.Match("learning.observe")
	if len(matched) != 1 {
		t.Fatalf("Match(\"learning.observe\") returned %d agents, want 1", len(matched))
	}
	if matched[0].Name() != "learning-agent" {
		t.Errorf("Match(\"learning.observe\")[0].Name() = %q, want %q", matched[0].Name(), "learning-agent")
	}

	nonMatch := r.Match("memory.consolidate")
	if len(nonMatch) != 0 {
		t.Fatalf("Match(\"memory.consolidate\") returned %d agents, want 0", len(nonMatch))
	}
}

func TestRegistryMatchMultiple(t *testing.T) {
	r := NewRegistry()

	learningAgent := &mockAgent{
		name:  "learning-handler",
		caste:  CasteCurator,
		triggers: []Trigger{
			{Topic: "learning.*"},
		},
	}
	memoryAgent := &mockAgent{
		name:  "memory-handler",
		caste:  CasteCurator,
		triggers: []Trigger{
			{Topic: "memory.*"},
		},
	}
	bothAgent := &mockAgent{
		name:  "multi-handler",
		caste:  CasteCurator,
		triggers: []Trigger{
			{Topic: "learning.*"},
			{Topic: "memory.*"},
		},
	}

	r.Register(learningAgent)
	r.Register(memoryAgent)
	r.Register(bothAgent)

	// learning.observe should match learning-handler and multi-handler
	learningMatches := r.Match("learning.observe")
	if len(learningMatches) != 2 {
		t.Fatalf("Match(\"learning.observe\") returned %d agents, want 2", len(learningMatches))
	}
	names := make(map[string]bool)
	for _, a := range learningMatches {
		names[a.Name()] = true
	}
	if !names["learning-handler"] || !names["multi-handler"] {
		t.Errorf("Match(\"learning.observe\") returned agents %v, want learning-handler and multi-handler", names)
	}

	// memory.consolidate should match memory-handler and multi-handler
	memoryMatches := r.Match("memory.consolidate")
	if len(memoryMatches) != 2 {
		t.Fatalf("Match(\"memory.consolidate\") returned %d agents, want 2", len(memoryMatches))
	}

	// unknown topic should match nothing
	unknownMatches := r.Match("unknown.event")
	if len(unknownMatches) != 0 {
		t.Fatalf("Match(\"unknown.event\") returned %d agents, want 0", len(unknownMatches))
	}
}
