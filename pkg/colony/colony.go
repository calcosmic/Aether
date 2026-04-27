// Package colony defines the core data types for the Aether colony state system.
// All types are designed for exact round-trip compatibility with the
// COLONY_STATE.json schema used by the shell-based colony implementation.
package colony

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// State constants
// ---------------------------------------------------------------------------

// State represents the lifecycle state of a colony.
type State string

const (
	StateIDLE      State = "IDLE"
	StateREADY     State = "READY"
	StateEXECUTING State = "EXECUTING"
	StateBUILT     State = "BUILT"
	StateCOMPLETED State = "COMPLETED"
)

// Phase status constants.
const (
	PhasePending    = "pending"
	PhaseReady      = "ready"
	PhaseInProgress = "in_progress"
	PhaseCompleted  = "completed"
)

// Task status constants.
const (
	TaskPending    = "pending"
	TaskCompleted  = "completed"
	TaskInProgress = "in_progress"
)

// WorktreeStatus represents the lifecycle state of a worktree.
type WorktreeStatus string

const (
	WorktreeAllocated  WorktreeStatus = "allocated"
	WorktreeInProgress WorktreeStatus = "in-progress"
	WorktreeMerged     WorktreeStatus = "merged"
	WorktreeOrphaned   WorktreeStatus = "orphaned"
)

// PlanGranularity represents the planning scope level (how many phases).
type PlanGranularity string

const (
	GranularitySprint    PlanGranularity = "sprint"
	GranularityMilestone PlanGranularity = "milestone"
	GranularityQuarter   PlanGranularity = "quarter"
	GranularityMajor     PlanGranularity = "major"
)

// Valid reports whether g is a recognized granularity level.
func (g PlanGranularity) Valid() bool {
	switch g {
	case GranularitySprint, GranularityMilestone, GranularityQuarter, GranularityMajor:
		return true
	}
	return false
}

// ErrInvalidGranularity is returned when a granularity value is not recognized.
var ErrInvalidGranularity = fmt.Errorf("invalid plan granularity")

// ParallelMode represents the parallel execution strategy for colony work.
type ParallelMode string

const (
	ModeInRepo   ParallelMode = "in-repo"
	ModeWorktree ParallelMode = "worktree"
)

// Valid reports whether m is a recognized parallel mode.
func (m ParallelMode) Valid() bool {
	switch m {
	case ModeInRepo, ModeWorktree:
		return true
	}
	return false
}

// ErrInvalidParallelMode is returned when a parallel mode value is not recognized.
var ErrInvalidParallelMode = fmt.Errorf("invalid parallel mode")

// ColonyScope represents the identity scope of a colony.
type ColonyScope string

const (
	ScopeProject ColonyScope = "project"
	ScopeMeta    ColonyScope = "meta"
)

// Valid reports whether s is a recognized colony scope.
func (s ColonyScope) Valid() bool {
	switch s {
	case ScopeProject, ScopeMeta:
		return true
	}
	return false
}

// Effective returns a compatibility-safe scope value.
// Legacy no-scope colonies are treated as project-scoped.
func (s ColonyScope) Effective() ColonyScope {
	if s.Valid() {
		return s
	}
	return ScopeProject
}

// ErrInvalidColonyScope is returned when a scope value is not recognized.
var ErrInvalidColonyScope = fmt.Errorf("invalid colony scope")

// ParseColonyScope validates the user-facing raw scope string.
func ParseColonyScope(raw string) (ColonyScope, error) {
	scope := ColonyScope(strings.ToLower(strings.TrimSpace(raw)))
	if !scope.Valid() {
		return "", ErrInvalidColonyScope
	}
	return scope, nil
}

// WorktreeEntry tracks a single worktree's lifecycle in COLONY_STATE.json.
type WorktreeEntry struct {
	ID           string         `json:"id"`
	Branch       string         `json:"branch"`
	Path         string         `json:"path"`
	Status       WorktreeStatus `json:"status"`
	Phase        int            `json:"phase"`
	Agent        string         `json:"agent,omitempty"`
	CreatedAt    string         `json:"created_at"`
	UpdatedAt    string         `json:"updated_at,omitempty"`
	LastCommitAt string         `json:"last_commit_at,omitempty"`
}

// GateResultEntry records the outcome of a single gate check in a continue run.
type GateResultEntry struct {
	Name      string `json:"name"`
	Passed    bool   `json:"passed"`
	Timestamp string `json:"timestamp"`
	Detail    string `json:"detail,omitempty"`
}

// ---------------------------------------------------------------------------
// Top-level state
// ---------------------------------------------------------------------------

// ColonyState is the top-level colony state matching COLONY_STATE.json.
type ColonyState struct {
	Version            string          `json:"version"`
	Goal               *string         `json:"goal"`
	Scope              ColonyScope     `json:"scope,omitempty"`
	ColonyName         *string         `json:"colony_name"`
	ColonyVersion      int             `json:"colony_version"`
	State              State           `json:"state"`
	CurrentPhase       int             `json:"current_phase"`
	SessionID          *string         `json:"session_id"`
	InitializedAt      *time.Time      `json:"initialized_at"`
	BuildStartedAt     *time.Time      `json:"build_started_at"`
	Plan               Plan            `json:"plan"`
	Memory             Memory          `json:"memory"`
	Errors             Errors          `json:"errors"`
	Signals            []Signal        `json:"signals"`
	Graveyards         []Graveyard     `json:"graveyards"`
	Events             []string        `json:"events"`
	ColonyDepth        string          `json:"colony_depth,omitempty"`
	PlanGranularity    PlanGranularity `json:"plan_granularity,omitempty"`
	ParallelMode       ParallelMode    `json:"parallel_mode,omitempty"`
	TerritorySurveyed  *string         `json:"territory_surveyed,omitempty"`
	Milestone          string          `json:"milestone"`
	MilestoneUpdatedAt *string         `json:"milestone_updated_at,omitempty"`
	Paused             bool            `json:"paused,omitempty"`
	PausedAt           *string         `json:"paused_at,omitempty"`
	Worktrees          []WorktreeEntry `json:"worktrees,omitempty"`
	RunID              *string            `json:"run_id,omitempty"`
	GateResults        []GateResultEntry  `json:"gate_results,omitempty"`
}

// EffectiveScope returns the compatibility-safe colony scope.
func (s ColonyState) EffectiveScope() ColonyScope {
	return s.Scope.Effective()
}

// ---------------------------------------------------------------------------
// Plan
// ---------------------------------------------------------------------------

// Plan holds the generated phase plan.
type Plan struct {
	GeneratedAt *time.Time `json:"generated_at"`
	Confidence  *float64   `json:"confidence"`
	Phases      []Phase    `json:"phases"`
}

// UnmarshalJSON preserves compatibility with legacy plan confidence payloads.
// Older colonies may store confidence as an object with per-axis percentages and
// an "overall" field rather than the newer single numeric value.
func (p *Plan) UnmarshalJSON(data []byte) error {
	type rawPlan struct {
		GeneratedAt *time.Time      `json:"generated_at"`
		Confidence  json.RawMessage `json:"confidence"`
		Phases      []Phase         `json:"phases"`
	}

	var raw rawPlan
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	p.GeneratedAt = raw.GeneratedAt
	p.Phases = raw.Phases

	confidence, err := decodePlanConfidence(raw.Confidence)
	if err != nil {
		return err
	}
	p.Confidence = confidence
	return nil
}

func decodePlanConfidence(raw json.RawMessage) (*float64, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}

	var numeric float64
	if err := json.Unmarshal(raw, &numeric); err == nil {
		return &numeric, nil
	}

	var legacy struct {
		Overall      *float64 `json:"overall"`
		Knowledge    *float64 `json:"knowledge"`
		Requirements *float64 `json:"requirements"`
		Risks        *float64 `json:"risks"`
		Dependencies *float64 `json:"dependencies"`
		Effort       *float64 `json:"effort"`
	}
	if err := json.Unmarshal(raw, &legacy); err != nil {
		return nil, err
	}

	if legacy.Overall != nil {
		value := *legacy.Overall
		if value > 1 {
			value /= 100.0
		}
		return &value, nil
	}

	var sum float64
	var count float64
	for _, candidate := range []*float64{
		legacy.Knowledge,
		legacy.Requirements,
		legacy.Risks,
		legacy.Dependencies,
		legacy.Effort,
	} {
		if candidate == nil {
			continue
		}
		value := *candidate
		if value > 1 {
			value /= 100.0
		}
		sum += value
		count++
	}
	if count == 0 {
		return nil, nil
	}
	average := sum / count
	return &average, nil
}

// Phase represents a single phase in the colony plan.
type Phase struct {
	ID              int      `json:"id"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	Status          string   `json:"status"`
	Tasks           []Task   `json:"tasks"`
	SuccessCriteria []string `json:"success_criteria"`
}

// Task represents a single task within a phase.
type Task struct {
	ID              *string  `json:"id"`
	Goal            string   `json:"goal"`
	Status          string   `json:"status"`
	Constraints     []string `json:"constraints,omitempty"`
	Hints           []string `json:"hints,omitempty"`
	SuccessCriteria []string `json:"success_criteria,omitempty"`
	DependsOn       []string `json:"depends_on,omitempty"`
}

// ---------------------------------------------------------------------------
// Memory
// ---------------------------------------------------------------------------

// Memory holds colony learning, decisions, and instincts.
type Memory struct {
	PhaseLearnings []PhaseLearning `json:"phase_learnings"`
	Decisions      []Decision      `json:"decisions"`
	Instincts      []Instinct      `json:"instincts"`
}

// UnmarshalJSON preserves compatibility with legacy memory payloads.
// Older colonies may store memory arrays as JSON-encoded strings such as
// "[]" or "[{...}]" instead of as proper arrays.
func (m *Memory) UnmarshalJSON(data []byte) error {
	if strings.TrimSpace(string(data)) == "" || strings.TrimSpace(string(data)) == "null" {
		*m = Memory{}
		return nil
	}

	type rawMemory struct {
		PhaseLearnings json.RawMessage `json:"phase_learnings"`
		Decisions      json.RawMessage `json:"decisions"`
		Instincts      json.RawMessage `json:"instincts"`
	}

	var raw rawMemory
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	phaseLearnings, err := decodeLegacyJSONArray[PhaseLearning](raw.PhaseLearnings, "phase_learnings")
	if err != nil {
		return err
	}
	decisions, err := decodeLegacyJSONArray[Decision](raw.Decisions, "decisions")
	if err != nil {
		return err
	}
	instincts, err := decodeLegacyJSONArray[Instinct](raw.Instincts, "instincts")
	if err != nil {
		return err
	}

	m.PhaseLearnings = phaseLearnings
	m.Decisions = decisions
	m.Instincts = instincts
	return nil
}

func decodeLegacyJSONArray[T any](raw json.RawMessage, fieldName string) ([]T, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}

	var direct []T
	if err := json.Unmarshal(raw, &direct); err == nil {
		return direct, nil
	}

	var encoded string
	if err := json.Unmarshal(raw, &encoded); err == nil {
		encoded = strings.TrimSpace(encoded)
		if encoded == "" || encoded == "null" {
			return nil, nil
		}

		if err := json.Unmarshal([]byte(encoded), &direct); err != nil {
			return nil, fmt.Errorf("%s string payload is not a valid JSON array: %w", fieldName, err)
		}
		return direct, nil
	}

	return nil, fmt.Errorf("%s must be an array or a JSON-encoded array string", fieldName)
}

// PhaseLearning captures learnings from a specific phase.
type PhaseLearning struct {
	ID        string     `json:"id"`
	Phase     int        `json:"phase"`
	PhaseName string     `json:"phase_name"`
	Learnings []Learning `json:"learnings"`
	Timestamp string     `json:"timestamp"`
}

// Learning represents a single learned claim.
type Learning struct {
	Claim       string  `json:"claim"`
	Status      string  `json:"status"`
	Tested      bool    `json:"tested"`
	Evidence    string  `json:"evidence"`
	DisprovenBy *string `json:"disproven_by"`
}

// Decision represents a decision made during the colony lifecycle.
type Decision struct {
	ID        string `json:"id"`
	Phase     int    `json:"phase"`
	Claim     string `json:"claim"`
	Rationale string `json:"rationale"`
	Timestamp string `json:"timestamp"`
}

// Instinct represents a learned behavioral pattern.
type Instinct struct {
	ID           string   `json:"id"`
	Trigger      string   `json:"trigger"`
	Action       string   `json:"action"`
	Confidence   float64  `json:"confidence"`
	Status       string   `json:"status"`
	Domain       string   `json:"domain"`
	Source       string   `json:"source"`
	Evidence     []string `json:"evidence"`
	Tested       bool     `json:"tested"`
	CreatedAt    string   `json:"created_at"`
	LastApplied  *string  `json:"last_applied"`
	Applications int      `json:"applications"`
	Successes    int      `json:"successes"`
	Failures     int      `json:"failures"`
}

// ---------------------------------------------------------------------------
// Errors
// ---------------------------------------------------------------------------

// Errors holds error records and flagged patterns.
type Errors struct {
	Records         []ErrorRecord    `json:"records"`
	FlaggedPatterns []FlaggedPattern `json:"flagged_patterns"`
}

// ErrorRecord represents a single error event.
type ErrorRecord struct {
	ID          string  `json:"id"`
	Category    string  `json:"category"`
	Severity    string  `json:"severity"`
	Description string  `json:"description"`
	RootCause   *string `json:"root_cause"`
	Phase       *int    `json:"phase"`
	TaskID      *string `json:"task_id"`
	Timestamp   string  `json:"timestamp"`
}

// FlaggedPattern represents a recurring error pattern.
type FlaggedPattern struct {
	Pattern   string     `json:"pattern"`
	Count     int        `json:"count"`
	FirstSeen *time.Time `json:"first_seen"`
	LastSeen  *time.Time `json:"last_seen"`
}

// ---------------------------------------------------------------------------
// Signals (deprecated, kept for backward compatibility)
// ---------------------------------------------------------------------------

// Signal represents a colony signal (deprecated in favor of pheromones.json).
type Signal struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Priority  string `json:"priority"`
	Source    string `json:"source"`
	Content   string `json:"content"`
	Active    bool   `json:"active"`
	CreatedAt string `json:"created_at"`
}

// ---------------------------------------------------------------------------
// Graveyards
// ---------------------------------------------------------------------------

// Graveyard marks a file where a builder has failed.
type Graveyard struct {
	ID             string  `json:"id"`
	File           string  `json:"file"`
	AntName        string  `json:"ant_name"`
	TaskID         string  `json:"task_id"`
	Phase          *int    `json:"phase"`
	FailureSummary string  `json:"failure_summary"`
	Function       *string `json:"function"`
	Line           *int    `json:"line"`
	Timestamp      string  `json:"timestamp"`
}

// ---------------------------------------------------------------------------
// State machine errors
// ---------------------------------------------------------------------------

// ErrInvalidTransition is returned when a state transition is not allowed.
var ErrInvalidTransition = fmt.Errorf("invalid state transition")

// legalTransitions defines the allowed state transitions.
var legalTransitions = map[State][]State{
	StateIDLE:      {StateREADY},
	StateREADY:     {StateEXECUTING, StateCOMPLETED},
	StateEXECUTING: {StateBUILT, StateCOMPLETED},
	StateBUILT:     {StateREADY, StateCOMPLETED},
	StateCOMPLETED: {StateIDLE},
}
