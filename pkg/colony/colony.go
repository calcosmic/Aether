// Package colony defines the core data types for the Aether colony state system.
// All types are designed for exact round-trip compatibility with the
// COLONY_STATE.json schema used by the shell-based colony implementation.
package colony

import (
	"fmt"
	"time"
)

// ---------------------------------------------------------------------------
// State constants
// ---------------------------------------------------------------------------

// State represents the lifecycle state of a colony.
type State string

const (
	StateREADY     State = "READY"
	StateEXECUTING State = "EXECUTING"
	StateBUILT     State = "BUILT"
	StateCOMPLETED State = "COMPLETED"
)

// ColonyDepth represents the build thoroughness level.
type ColonyDepth string

const (
	DepthLight    ColonyDepth = "light"
	DepthStandard ColonyDepth = "standard"
	DepthDeep     ColonyDepth = "deep"
	DepthFull     ColonyDepth = "full"
)

// Valid reports whether d is a recognized depth level.
func (d ColonyDepth) Valid() bool {
	switch d {
	case DepthLight, DepthStandard, DepthDeep, DepthFull:
		return true
	}
	return false
}

// ErrInvalidDepth is returned when a depth value is not recognized.
var ErrInvalidDepth = fmt.Errorf("invalid colony depth")

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

// ---------------------------------------------------------------------------
// Top-level state
// ---------------------------------------------------------------------------

// ColonyState is the top-level colony state matching COLONY_STATE.json.
type ColonyState struct {
	Version            string      `json:"version"`
	Goal               *string     `json:"goal"`
	ColonyName         *string     `json:"colony_name"`
	ColonyVersion      int         `json:"colony_version"`
	State              State       `json:"state"`
	CurrentPhase       int         `json:"current_phase"`
	SessionID          *string     `json:"session_id"`
	InitializedAt      *time.Time  `json:"initialized_at"`
	BuildStartedAt     *time.Time  `json:"build_started_at"`
	Plan               Plan        `json:"plan"`
	Memory             Memory      `json:"memory"`
	Errors             Errors      `json:"errors"`
	Signals            []Signal    `json:"signals"`
	Graveyards         []Graveyard `json:"graveyards"`
	Events             []string    `json:"events"`
	ColonyDepth        ColonyDepth     `json:"colony_depth,omitempty"`
	PlanGranularity    PlanGranularity `json:"plan_granularity,omitempty"`
	OrchestratorState  *OrchestratorState `json:"orchestrator_state,omitempty"`
	Milestone          string             `json:"milestone"`
	MilestoneUpdatedAt *string            `json:"milestone_updated_at,omitempty"`
}

// ---------------------------------------------------------------------------
// Orchestrator
// ---------------------------------------------------------------------------

// TaskAssignment tracks a single task's orchestration state.
type TaskAssignment struct {
	TaskID    string `json:"task_id"`
	Goal      string `json:"goal"`
	Caste     string `json:"caste"`
	AgentName string `json:"agent_name,omitempty"`
	Status    string `json:"status"` // pending, in_progress, completed, failed
	StartedAt string `json:"started_at,omitempty"`
	EndedAt   string `json:"ended_at,omitempty"`
	Error     string `json:"error,omitempty"`
}

// OrchestratorState holds the top-level orchestrator state.
type OrchestratorState struct {
	Phase          int              `json:"phase"`
	Status         string           `json:"status"` // idle, decomposing, dispatching, collecting, validating, completed, failed
	TaskCount      int              `json:"task_count"`
	Completed      int              `json:"completed"`
	Failed         int              `json:"failed"`
	Headless       bool             `json:"headless"`
	ReplanInterval int              `json:"replan_interval"`
	StartedAt      string           `json:"started_at,omitempty"`
	UpdatedAt      string           `json:"updated_at,omitempty"`
	Assignments    []TaskAssignment `json:"assignments"`
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
	StateREADY:     {StateEXECUTING, StateCOMPLETED},
	StateEXECUTING: {StateBUILT, StateCOMPLETED},
	StateBUILT:     {StateREADY},
}
