package colony

// SessionFile represents the top-level session.json file.
type SessionFile struct {
	SessionID        string   `json:"session_id"`
	StartedAt        string   `json:"started_at"`
	LastCommand      string   `json:"last_command"`
	LastCommandAt    string   `json:"last_command_at"`
	ColonyGoal       string   `json:"colony_goal"`
	CurrentPhase     int      `json:"current_phase"`
	CurrentMilestone string   `json:"current_milestone"`
	SuggestedNext    string   `json:"suggested_next"`
	ContextCleared   bool     `json:"context_cleared"`
	BaselineCommit   string   `json:"baseline_commit"`
	ResumedAt        *string  `json:"resumed_at"`
	ActiveTodos      []string `json:"active_todos"`
	Summary          string   `json:"summary"`
}
