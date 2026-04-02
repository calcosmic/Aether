package colony

// FlagEntry represents a single pending decision in pending-decisions.json.
type FlagEntry struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Phase       *int   `json:"phase"`
	Source      string `json:"source"`
	CreatedAt   string `json:"created_at"`
	Resolved    bool   `json:"resolved"`
}

// FlagsFile represents the top-level pending-decisions.json file.
type FlagsFile struct {
	Version   string     `json:"version"`
	Decisions []FlagEntry `json:"decisions"`
}
