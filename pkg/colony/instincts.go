package colony

import "encoding/json"

// InstinctApplicationEntry is one entry in an instinct's ApplicationHistory:
// what phase used this instinct, when, and what outcome resulted
// (SYN-204-05/06, 204-03-PLAN.md Task 2, LEARN-03). The new shape records
// Outcome from the credit ledger's own closed vocabulary
// (helpful/neutral/harmful/pending, cmd/recruitment_credit.go's
// recruitmentCreditOutcome) and, when earned, the CreditRecordID that
// justified it -- never a bare boolean self-report (SYN-204-06 directly
// repudiates Classic's unverified `instinct_outcomes` self-report,
// 204-CLASSIC-SYNTHESIS.md ruling OLD-02).
//
// UnmarshalJSON accepts two on-disk shapes so a single ApplicationHistory
// slice can hold both, side by side, for as long as a real colony's file
// does: the pre-Phase-204 shape (a bare "success" boolean, no
// outcome/credit_record_id key at all) is the READ compatibility path,
// captured into LegacySuccess; every writer after this change produces the
// new shape (see pkg/memory/instinct_stats.go's SummarizeInstinctApplications
// for how both shapes are read identically).
type InstinctApplicationEntry struct {
	Timestamp      string `json:"timestamp"`
	Phase          int    `json:"phase,omitempty"`
	Outcome        string `json:"outcome,omitempty"`
	CreditRecordID string `json:"credit_record_id,omitempty"`

	// LegacySuccess is set only when UnmarshalJSON reads the pre-Phase-204
	// shape's bare "success" boolean and no "outcome" key is present. It
	// is never written by a current writer (json:"-") -- it exists purely
	// so a reader can fold the compatibility path into the same counting
	// logic as a genuinely-typed entry.
	LegacySuccess *bool `json:"-"`
}

// UnmarshalJSON accepts both the old untyped shape
// ({"timestamp","success","phase"}) and the new typed shape
// ({"timestamp","phase","outcome","credit_record_id"}).
func (e *InstinctApplicationEntry) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if v, ok := raw["timestamp"]; ok {
		if err := json.Unmarshal(v, &e.Timestamp); err != nil {
			return err
		}
	}
	if v, ok := raw["phase"]; ok {
		if err := json.Unmarshal(v, &e.Phase); err != nil {
			return err
		}
	}
	if v, ok := raw["outcome"]; ok {
		if err := json.Unmarshal(v, &e.Outcome); err != nil {
			return err
		}
	}
	if v, ok := raw["credit_record_id"]; ok {
		if err := json.Unmarshal(v, &e.CreditRecordID); err != nil {
			return err
		}
	}
	if e.Outcome == "" {
		if v, ok := raw["success"]; ok {
			var success bool
			if err := json.Unmarshal(v, &success); err == nil {
				e.LegacySuccess = &success
			}
		}
	}
	return nil
}

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
	ID                 string                     `json:"id"`
	Trigger            string                     `json:"trigger"`
	Action             string                     `json:"action"`
	Domain             string                     `json:"domain"`
	TrustScore         float64                    `json:"trust_score"`
	TrustTier          string                     `json:"trust_tier"`
	Confidence         float64                    `json:"confidence"`
	Provenance         InstinctProvenance         `json:"provenance"`
	ApplicationHistory []InstinctApplicationEntry `json:"application_history"`
	RelatedInstincts   []interface{}              `json:"related_instincts"`
	Archived           bool                       `json:"archived"`

	// SchemaVersion and Lineage are the SYN-204-02 per-record schema
	// contract (204-03-PLAN.md Task 1, LEARN-01): SchemaVersion is
	// omitempty because its zero value IS LegacyMemorySchemaVersion --
	// an absent field and an explicit legacy stamp read identically, so
	// there is no third state to track. Lineage is nil for any record
	// written before this change (memory_schema.go's
	// MemoryRecordLineage.ResolvedProvenance reads a nil Lineage as
	// MemoryProvenanceUnknown, never a fabricated value).
	SchemaVersion int                  `json:"schema_version,omitempty"`
	Lineage       *MemoryRecordLineage `json:"lineage,omitempty"`
}

// InstinctsFile represents the standalone instincts.json file.
type InstinctsFile struct {
	Version   string          `json:"version"`
	Instincts []InstinctEntry `json:"instincts"`
}
