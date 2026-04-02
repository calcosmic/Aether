package colony

import "encoding/json"

// PheromoneTag represents a categorization tag on a pheromone signal.
type PheromoneTag struct {
	Value    string  `json:"value"`
	Weight   float64 `json:"weight"`
	Category string  `json:"category"`
}

// PheromoneScope defines the scope of a pheromone signal.
type PheromoneScope struct {
	Global bool `json:"global"`
}

// PheromoneSignal represents a single pheromone signal in pheromones.json.
// Content uses json.RawMessage to preserve nested JSON objects like
// {"text": "..."} without double-escaping.
type PheromoneSignal struct {
	ID                 string          `json:"id"`
	Type               string          `json:"type"`
	Priority           string          `json:"priority"`
	Source             string          `json:"source"`
	CreatedAt          string          `json:"created_at"`
	ExpiresAt          *string         `json:"expires_at,omitempty"`
	Active             bool            `json:"active"`
	Strength           *float64        `json:"strength,omitempty"`
	Reason             *string         `json:"reason,omitempty"`
	Content            json.RawMessage `json:"content"`
	ContentHash        *string         `json:"content_hash,omitempty"`
	ReinforcementCount *int            `json:"reinforcement_count,omitempty"`
	ArchivedAt         *string         `json:"archived_at,omitempty"`
	Tags               []PheromoneTag  `json:"tags,omitempty"`
	Scope              *PheromoneScope `json:"scope,omitempty"`
}

// PheromoneFile represents the top-level pheromones.json file.
type PheromoneFile struct {
	Signals  []PheromoneSignal `json:"signals"`
	Version  *string           `json:"version,omitempty"`
	ColonyID *string           `json:"colony_id,omitempty"`
}
