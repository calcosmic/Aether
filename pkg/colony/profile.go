package colony

// BehaviorObservation records one behavioral signal observed during colony use.
type BehaviorObservation struct {
	Timestamp  string  `json:"timestamp"`
	ColonyGoal string  `json:"colony_goal,omitempty"`
	Command    string  `json:"command"`
	Dimension  string  `json:"dimension"`
	Signal     string  `json:"signal"`
	Strength   float64 `json:"strength"`
	Evidence   string  `json:"evidence"`
}

// BehavioralDimension stores a scored user preference dimension.
type BehavioralDimension struct {
	Name        string   `json:"name"`
	Score       float64  `json:"score"`
	Evidence    []string `json:"evidence,omitempty"`
	UpdatedAt   string   `json:"updated_at,omitempty"`
	SampleCount int      `json:"sample_count"`
}

// UserProfile is the hub-level behavioral profile shared across colonies.
type UserProfile struct {
	Version     string                `json:"version"`
	GeneratedAt string                `json:"generated_at"`
	ColonyCount int                   `json:"colony_count"`
	Dimensions  []BehavioralDimension `json:"dimensions"`
	Directives  []string              `json:"directives"`
}
