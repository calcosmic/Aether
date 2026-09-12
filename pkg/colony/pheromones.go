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
	SourcePhase        *int            `json:"source_phase,omitempty"` // Phase when signal was created

	// Provenance and Quarantined are pointer-backed and omitempty, per the
	// Phase 199 rule that new lifecycle evidence is pointer-backed and
	// omitted when absent (BIO-07): a legacy colony's signals, written
	// before these fields existed, stay readable and read as an explicit
	// "unknown" provenance / not quarantined (pheromoneSignalProvenance,
	// pheromoneSignalQuarantined in cmd/pheromone_resolver.go) rather than
	// a fabricated category.
	Provenance  *string `json:"provenance,omitempty"`
	Quarantined *bool   `json:"quarantined,omitempty"`
}

// Write-time provenance categories a stored pheromone signal may declare.
// PheromoneProvenanceUnknown is deliberately NOT one of these -- it is the
// read-time fallback for a legacy signal with no Provenance field, never a
// value writePheromoneSignal itself stamps.
const (
	PheromoneProvenanceOwner    = "owner"
	PheromoneProvenanceRuntime  = "runtime"
	PheromoneProvenanceLearning = "learning"
	PheromoneProvenanceImport   = "import"
	PheromoneProvenanceUnknown  = "unknown"
)

// PheromoneProvenances returns the four write-time provenance categories a
// stored pheromone signal may declare. cmd's
// TestPheromoneProvenanceRegistryIsComplete parses this file's declared
// PheromoneProvenance* constants and fails by name if a new one is added
// here without also being added to this accessor.
func PheromoneProvenances() []string {
	return []string{
		PheromoneProvenanceOwner,
		PheromoneProvenanceRuntime,
		PheromoneProvenanceLearning,
		PheromoneProvenanceImport,
	}
}

// PheromoneFile represents the top-level pheromones.json file.
type PheromoneFile struct {
	Signals  []PheromoneSignal `json:"signals"`
	Version  *string           `json:"version,omitempty"`
	ColonyID *string           `json:"colony_id,omitempty"`
}
