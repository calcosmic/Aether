package colony

// InstinctProvenance tracks the origin and application history of an instinct.
type InstinctProvenance struct {
	Source           string  `json:"source"`
	SourceType       string  `json:"source_type"`
	Evidence         string  `json:"evidence"`
	CreatedAt        string  `json:"created_at"`
	LastApplied      *string `json:"last_applied"`
	ApplicationCount int     `json:"application_count"`
	// OriginLabel is a human-readable origin ("from research: <topic>,
	// <date>") for an instinct auto-promoted from an Oracle run (D-06). Empty
	// for instincts promoted any other way.
	OriginLabel string `json:"origin_label,omitempty"`
}

// InstinctEntry represents a single instinct in the standalone instincts.json file.
// This is the richer schema managed by instinct-store.sh, distinct from the
// simpler Instinct type embedded in ColonyState.Memory.Instincts.
type InstinctEntry struct {
	ID                 string             `json:"id"`
	Trigger            string             `json:"trigger"`
	Action             string             `json:"action"`
	Domain             string             `json:"domain"`
	TrustScore         float64            `json:"trust_score"`
	TrustTier          string             `json:"trust_tier"`
	Confidence         float64            `json:"confidence"`
	Provenance         InstinctProvenance `json:"provenance"`
	ApplicationHistory []interface{}      `json:"application_history"`
	RelatedInstincts   []interface{}      `json:"related_instincts"`
	Archived           bool               `json:"archived"`
}

// InstinctsFile represents the standalone instincts.json file.
type InstinctsFile struct {
	Version   string          `json:"version"`
	Instincts []InstinctEntry `json:"instincts"`
}
