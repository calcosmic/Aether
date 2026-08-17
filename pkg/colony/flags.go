package colony

// FlagEntry represents a single pending decision in pending-decisions.json.
type FlagEntry struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Description  string `json:"description"`
	Phase        *int   `json:"phase"`
	Source       string `json:"source"`
	CreatedAt    string `json:"created_at"`
	Resolved     bool   `json:"resolved"`
	ResolvedAt   string `json:"resolved_at,omitempty"`
	Resolution   string `json:"resolution,omitempty"`
	Acknowledged bool   `json:"acknowledged,omitempty"`
	// AcknowledgedAt records WHEN a flag was parked (classic flag lifecycle:
	// acknowledge = "noted but continuing", issues and notes only — blockers
	// cannot be acknowledged, they must be resolved).
	AcknowledgedAt string `json:"acknowledged_at,omitempty"`
}

// FlagsFile represents the top-level pending-decisions.json file.
type FlagsFile struct {
	Version   string      `json:"version"`
	Decisions []FlagEntry `json:"decisions"`
}
