package colony

import "encoding/json"

// MiddenEntry represents a single failure record in midden.json.
type MiddenEntry struct {
	ID                string   `json:"id"`
	Timestamp         string   `json:"timestamp"`
	Category          string   `json:"category"`
	Source            string   `json:"source"`
	Message           string   `json:"message"`
	Reviewed          bool     `json:"reviewed"`
	Acknowledged      *bool    `json:"acknowledged,omitempty"`
	AcknowledgedAt    *string  `json:"acknowledged_at,omitempty"`
	AcknowledgeReason *string  `json:"acknowledge_reason,omitempty"`
	Tags              []string `json:"tags,omitempty"`
}

// MiddenArchivedSignal represents an archived pheromone signal in midden.json.
// Signals and entries are different types with different fields.
type MiddenArchivedSignal struct {
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

// MiddenSpawnMetrics tracks spawn efficiency metrics in midden.json.
type MiddenSpawnMetrics struct {
	TotalSpawned  int `json:"total_spawned"`
	Completed     int `json:"completed"`
	Failed        int `json:"failed"`
	EfficiencyPct int `json:"efficiency_pct"`
}

// MiddenFile represents the top-level midden.json file.
type MiddenFile struct {
	Version         string               `json:"version"`
	ArchivedAtCount *int                 `json:"archived_at_count,omitempty"`
	Signals         []MiddenArchivedSignal `json:"signals"`
	Entries         []MiddenEntry        `json:"entries"`
	EntryCount      *int                 `json:"entry_count,omitempty"`
	SpawnMetrics    *MiddenSpawnMetrics  `json:"spawn_metrics,omitempty"`
}
