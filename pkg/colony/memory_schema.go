package colony

import (
	"fmt"
	"sort"
)

// SYN-204-02 (204-03-PLAN.md Task 1, LEARN-01): the one shared schema
// version and provenance contract every live memory store (instincts,
// the failure log, learning entries) agrees to. Declared in this package,
// not in cmd, because every layer that writes a memory-store record --
// cmd's appendMiddenEntry, pkg/memory's PromoteService.Promote, and
// pkg/learn's ColonyStore.Add -- already imports pkg/colony for the
// record types themselves (InstinctEntry, MiddenEntry). Neither
// pkg/memory nor pkg/learn may import package cmd (cmd imports both of
// them; the reverse would cycle), so the contract has to live at the
// layer every writer can already reach -- here. cmd/memory_schema.go
// declares cmd-package-local names for these same symbols so cmd's own
// tests and its own writer (appendMiddenEntry) can refer to them under
// the identifiers this phase's plan names, without inventing a second,
// competing contract.

// CurrentMemorySchemaVersion is the schema version every live memory
// store's write chokepoint stamps on a newly-written record. Follows
// pkg/codex/permission_profile.go's PermissionProfileSchemaVersion idiom
// -- this repository's existing precedent for a declared schema constant.
const CurrentMemorySchemaVersion = 1

// LegacyMemorySchemaVersion is the version a record with no stamp at all
// (written before this contract existed) is read as. It is deliberately
// the zero value of the SchemaVersion field's int type: a legacy record's
// absent "schema_version" key and an explicit legacy stamp both read
// identically, and json's own "omitempty" convention already omits a
// zero-valued int on write -- there is no third state to track.
const LegacyMemorySchemaVersion = 0

// MemoryStoreSchemaReadable reports whether version is a version this
// runtime knows how to read: the legacy version (no stamp) or the current
// one. A version above the current one is a record written by a newer
// runtime this build has never seen -- refused by name (the caller reports
// it as unreadable-by-this-runtime), never silently coerced into looking
// like a current or legacy record.
func MemoryStoreSchemaReadable(version int) bool {
	return version == LegacyMemorySchemaVersion || version == CurrentMemorySchemaVersion
}

// MemoryProvenanceKind is the closed, declared vocabulary a memory
// record's lineage may name as its origin -- the same five values
// pkg/colony/pheromones.go's PheromoneSignal.Provenance already proves
// workable (owner/runtime/learning/import as write-time values, unknown
// as the read-time fallback for an absent provenance).
type MemoryProvenanceKind string

const (
	MemoryProvenanceOwner    MemoryProvenanceKind = "owner"
	MemoryProvenanceRuntime  MemoryProvenanceKind = "runtime"
	MemoryProvenanceLearning MemoryProvenanceKind = "learning"
	MemoryProvenanceImport   MemoryProvenanceKind = "import"
	MemoryProvenanceUnknown  MemoryProvenanceKind = "unknown"
)

// MemoryProvenanceVocabulary is the declared, closed set of every value a
// memory record's lineage Provenance field may carry. Mirrors
// cmd/recruitment_credit.go's recruitmentCreditOutcomeVocabulary
// completeness convention exactly: a declared constant, a slice naming
// every member (this one), a names helper for refusal messages, and a
// declared-membership predicate below.
var MemoryProvenanceVocabulary = []MemoryProvenanceKind{
	MemoryProvenanceOwner,
	MemoryProvenanceRuntime,
	MemoryProvenanceLearning,
	MemoryProvenanceImport,
	MemoryProvenanceUnknown,
}

// MemoryProvenanceNames returns the string form of every declared
// provenance kind, for display and for refusal messages.
func MemoryProvenanceNames() []string {
	names := make([]string, 0, len(MemoryProvenanceVocabulary))
	for _, k := range MemoryProvenanceVocabulary {
		names = append(names, string(k))
	}
	return names
}

// MemoryProvenanceDeclared reports whether kind is one of the closed
// vocabulary's declared members.
func MemoryProvenanceDeclared(kind MemoryProvenanceKind) bool {
	for _, k := range MemoryProvenanceVocabulary {
		if k == kind {
			return true
		}
	}
	return false
}

// ValidateMemoryProvenanceKind refuses, by name, any value outside the
// closed vocabulary -- mirroring recordRecruitmentCredit's own refusal
// shape (cmd/recruitment_credit.go) for an undeclared contribution kind.
func ValidateMemoryProvenanceKind(kind MemoryProvenanceKind) error {
	if !MemoryProvenanceDeclared(kind) {
		return fmt.Errorf("memory provenance kind %q is not in the declared vocabulary %v", kind, MemoryProvenanceNames())
	}
	return nil
}

// MemoryRecordLineage is the common provenance shape every live memory
// store's per-record lineage field shares: the provenance kind, the
// identifier of the record or actor it came from, the identifier of the
// outcome record that justified it where one exists, and the timestamp it
// was recorded. Every field is pointer-backed and omitempty, following
// pheromones.go's PheromoneSignal.Provenance/Quarantined convention (the
// census's own best-shaped schema) -- a legacy record with no lineage at
// all reads every field as nil / its documented safe default, never a
// fabricated value. Structurally compatible with PheromoneSignal's own
// provenance string field: both share the SAME underlying string
// vocabulary (owner/runtime/learning/import/unknown), even though
// PheromoneSignal predates this shared struct and is not migrated to it.
type MemoryRecordLineage struct {
	Provenance  *MemoryProvenanceKind `json:"provenance,omitempty"`
	SourceID    *string               `json:"source_id,omitempty"`
	JustifiedBy *string               `json:"justified_by,omitempty"`
	RecordedAt  *string               `json:"recorded_at,omitempty"`
}

// ResolvedProvenance returns l's declared provenance kind, or
// MemoryProvenanceUnknown when l itself is nil or its Provenance field was
// never stamped -- the one place "absent reads as unknown, not as any of
// the four write-time values" is decided.
func (l *MemoryRecordLineage) ResolvedProvenance() MemoryProvenanceKind {
	if l == nil || l.Provenance == nil {
		return MemoryProvenanceUnknown
	}
	return *l.Provenance
}

// NewMemoryRecordLineage builds a lineage value for a freshly-written
// record: the given provenance kind (must be one of the four write-time
// values -- owner/runtime/learning/import; callers never stamp unknown,
// which is a read-time fallback only) and source identifier, recorded at
// recordedAt. sourceID is omitted (left nil) when empty, rather than
// stored as an empty string that looks deliberate.
func NewMemoryRecordLineage(kind MemoryProvenanceKind, sourceID, recordedAt string) MemoryRecordLineage {
	k := kind
	lineage := MemoryRecordLineage{Provenance: &k}
	if sourceID != "" {
		lineage.SourceID = &sourceID
	}
	if recordedAt != "" {
		lineage.RecordedAt = &recordedAt
	}
	return lineage
}

// SortInstinctEntriesByRecency orders entries by their most recent known
// timestamp (lineage.RecordedAt when stamped, else the existing
// Provenance.CreatedAt) descending, tie-broken by ID ascending -- so two
// entries recorded in the same instant sort identically every time,
// regardless of the order they happened to be read from disk. Mirrors
// cmd/recruitment_credit.go's sortRecruitmentCreditRecords tie-break shape.
func SortInstinctEntriesByRecency(entries []InstinctEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		ti, tj := instinctEntryRecordedAt(entries[i]), instinctEntryRecordedAt(entries[j])
		if ti != tj {
			return ti > tj
		}
		return entries[i].ID < entries[j].ID
	})
}

func instinctEntryRecordedAt(e InstinctEntry) string {
	if e.Lineage != nil && e.Lineage.RecordedAt != nil {
		return *e.Lineage.RecordedAt
	}
	return e.Provenance.CreatedAt
}

// SortMiddenEntriesByRecency orders entries by Timestamp descending,
// tie-broken by ID ascending -- the same total-order tie-break shape as
// SortInstinctEntriesByRecency, applied to the failure log.
func SortMiddenEntriesByRecency(entries []MiddenEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Timestamp != entries[j].Timestamp {
			return entries[i].Timestamp > entries[j].Timestamp
		}
		return entries[i].ID < entries[j].ID
	})
}
