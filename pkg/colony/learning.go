package colony

// Observation represents a single learning observation in learning-observations.json.
type Observation struct {
	ContentHash      string   `json:"content_hash"`
	Content          string   `json:"content"`
	WisdomType       string   `json:"wisdom_type"`
	ObservationCount int      `json:"observation_count"`
	FirstSeen        string   `json:"first_seen"`
	LastSeen         string   `json:"last_seen"`
	Colonies         []string `json:"colonies"`
	TrustScore       *float64 `json:"trust_score,omitempty"`
	SourceType       string   `json:"source_type,omitempty"`
	EvidenceType     string   `json:"evidence_type,omitempty"`
	CompressionLevel int      `json:"compression_level,omitempty"`
}

// LearningFile represents the top-level learning-observations.json file.
type LearningFile struct {
	Observations []Observation `json:"observations"`
}
