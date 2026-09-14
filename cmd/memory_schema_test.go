package cmd

// SYN-204-02 (204-03-PLAN.md, LEARN-01): proves the shared memory schema
// and provenance contract (pkg/colony/memory_schema.go, cmd/memory_schema.go)
// against real legacy bytes and real store writers -- never a fixture typed
// as a plausible-looking literal for anything the runtime itself can
// produce (CLAUDE.md's Definition of Done). The one deliberate exception is
// the legacy fixture: the runtime can no longer produce the pre-Phase-204
// shape (every writer now stamps schema_version/lineage), so that shape is
// written as raw bytes directly to the store's own file path -- see each
// legacy fixture's own comment.

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/learn"
)

// ---------------------------------------------------------------------------
// TestLegacyRecordsReadAsLegacy
// ---------------------------------------------------------------------------

// legacyInstinctsJSON is the pre-Phase-204 instincts.json shape: no
// "schema_version" or "lineage" key on the entry. Written as raw bytes
// because promoteRealInstinct (this package's own real-writer helper) can
// no longer produce this shape -- every real writer now stamps both fields.
const legacyInstinctsJSON = `{
  "version": "1.0",
  "instincts": [
    {
      "id": "inst_legacy_1",
      "trigger": "run gofmt -l cmd/ before committing",
      "action": "run gofmt -l cmd/ before committing",
      "domain": "pattern",
      "trust_score": 0.5,
      "trust_tier": "bronze",
      "confidence": 0.6,
      "provenance": {
        "source": "sha256:legacyabc",
        "source_type": "success_pattern",
        "evidence": "single_phase",
        "created_at": "2026-01-01T00:00:00Z",
        "last_applied": null,
        "application_count": 0
      },
      "application_history": [],
      "related_instincts": [],
      "archived": false
    }
  ]
}
`

// legacyMiddenJSON is the pre-Phase-204 midden.json shape: no
// "schema_version" or "lineage" key on the entry.
const legacyMiddenJSON = `{
  "version": "1.0.0",
  "entries": [
    {
      "id": "midden_legacy_1",
      "timestamp": "2026-01-01T00:00:00Z",
      "category": "test-category",
      "source": "test-source",
      "message": "legacy failure message",
      "reviewed": false
    }
  ]
}
`

// legacyLearnEntriesJSON is the pre-Phase-204 entries.json shape: a bare
// top-level array, no "schema_version" or "lineage" key on the entry.
const legacyLearnEntriesJSON = `[
  {
    "id": "lrn_legacy_1",
    "content": "legacy learning content",
    "evidence": {
      "run_id": "run-legacy-1",
      "phase": 1,
      "workers": [],
      "gates_passed": 1,
      "gates_total": 1,
      "confidence": 0.5,
      "timestamp": "2026-01-01T00:00:00Z",
      "scope": "phase"
    },
    "classification": "repo-local",
    "created_at": "2026-01-01T00:00:00Z",
    "phase": 1,
    "confidence": 0.5,
    "status": "hypothesis"
  }
]
`

func TestLegacyRecordsReadAsLegacy(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	t.Run("instincts", func(t *testing.T) {
		if err := s.SaveRawJSON("instincts.json", []byte(legacyInstinctsJSON)); err != nil {
			t.Fatalf("seed legacy instincts.json: %v", err)
		}
		file := loadInstinctFileOrEmpty(s)
		if len(file.Instincts) != 1 {
			t.Fatalf("loaded %d instincts, want 1", len(file.Instincts))
		}
		entry := file.Instincts[0]
		if entry.SchemaVersion != memoryStoreLegacySchemaVersion {
			t.Errorf("legacy instinct SchemaVersion = %d, want %d (legacy)", entry.SchemaVersion, memoryStoreLegacySchemaVersion)
		}
		if entry.Lineage != nil {
			t.Errorf("legacy instinct Lineage = %+v, want nil", entry.Lineage)
		}
		if entry.Lineage.ResolvedProvenance() != colony.MemoryProvenanceUnknown {
			t.Errorf("legacy instinct ResolvedProvenance() = %q, want %q", entry.Lineage.ResolvedProvenance(), colony.MemoryProvenanceUnknown)
		}
	})

	t.Run("midden", func(t *testing.T) {
		if err := s.SaveRawJSON(middenCanonicalPath, []byte(legacyMiddenJSON)); err != nil {
			t.Fatalf("seed legacy midden.json: %v", err)
		}
		mf, err := loadMiddenFile(s)
		if err != nil {
			t.Fatalf("loadMiddenFile: %v", err)
		}
		if len(mf.Entries) != 1 {
			t.Fatalf("loaded %d midden entries, want 1", len(mf.Entries))
		}
		entry := mf.Entries[0]
		if entry.SchemaVersion != memoryStoreLegacySchemaVersion {
			t.Errorf("legacy midden SchemaVersion = %d, want %d (legacy)", entry.SchemaVersion, memoryStoreLegacySchemaVersion)
		}
		if entry.Lineage != nil {
			t.Errorf("legacy midden Lineage = %+v, want nil", entry.Lineage)
		}
		if entry.Lineage.ResolvedProvenance() != colony.MemoryProvenanceUnknown {
			t.Errorf("legacy midden ResolvedProvenance() = %q, want %q", entry.Lineage.ResolvedProvenance(), colony.MemoryProvenanceUnknown)
		}
	})

	t.Run("learn_entries", func(t *testing.T) {
		if err := s.SaveRawJSON("entries.json", []byte(legacyLearnEntriesJSON)); err != nil {
			t.Fatalf("seed legacy entries.json: %v", err)
		}
		ls := learn.NewColonyStore(s)
		entry, err := ls.Get("lrn_legacy_1")
		if err != nil {
			t.Fatalf("Get legacy learn entry: %v", err)
		}
		if entry == nil {
			t.Fatal("legacy learn entry not found")
		}
		if entry.SchemaVersion != memoryStoreLegacySchemaVersion {
			t.Errorf("legacy learn entry SchemaVersion = %d, want %d (legacy)", entry.SchemaVersion, memoryStoreLegacySchemaVersion)
		}
		if entry.Lineage != nil {
			t.Errorf("legacy learn entry Lineage = %+v, want nil", entry.Lineage)
		}
		if entry.Lineage.ResolvedProvenance() != colony.MemoryProvenanceUnknown {
			t.Errorf("legacy learn entry ResolvedProvenance() = %q, want %q", entry.Lineage.ResolvedProvenance(), colony.MemoryProvenanceUnknown)
		}
	})
}

// ---------------------------------------------------------------------------
// TestNewRecordsCarryVersionAndLineage
// ---------------------------------------------------------------------------

func TestNewRecordsCarryVersionAndLineage(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	t.Run("instincts", func(t *testing.T) {
		inst := promoteRealInstinct(t, s, "run go build ./cmd/aether before go vet ./cmd/", "pattern")
		if inst.SchemaVersion != colony.CurrentMemorySchemaVersion {
			t.Errorf("new instinct SchemaVersion = %d, want %d (current)", inst.SchemaVersion, colony.CurrentMemorySchemaVersion)
		}
		if inst.Lineage == nil {
			t.Fatal("new instinct Lineage is nil, want stamped")
		}
		if got := inst.Lineage.ResolvedProvenance(); got != colony.MemoryProvenanceLearning {
			t.Errorf("new instinct provenance = %q, want %q", got, colony.MemoryProvenanceLearning)
		}
	})

	t.Run("midden", func(t *testing.T) {
		if err := appendMiddenEntry(s, "test-category", "test-source", "a genuinely new failure", nil); err != nil {
			t.Fatalf("appendMiddenEntry: %v", err)
		}
		mf, err := loadMiddenFile(s)
		if err != nil {
			t.Fatalf("loadMiddenFile: %v", err)
		}
		var entry *colony.MiddenEntry
		for i := range mf.Entries {
			if mf.Entries[i].Message == "a genuinely new failure" {
				entry = &mf.Entries[i]
			}
		}
		if entry == nil {
			t.Fatal("newly written midden entry not found")
		}
		if entry.SchemaVersion != colony.CurrentMemorySchemaVersion {
			t.Errorf("new midden SchemaVersion = %d, want %d (current)", entry.SchemaVersion, colony.CurrentMemorySchemaVersion)
		}
		if entry.Lineage == nil {
			t.Fatal("new midden Lineage is nil, want stamped")
		}
		if got := entry.Lineage.ResolvedProvenance(); got != colony.MemoryProvenanceRuntime {
			t.Errorf("new midden provenance = %q, want %q", got, colony.MemoryProvenanceRuntime)
		}
	})

	t.Run("learn_entries", func(t *testing.T) {
		ls := learn.NewColonyStore(s)
		if err := ls.Add(learn.Entry{
			Content:        "a genuinely new learning entry",
			Classification: learn.ClassRepoLocal,
			CreatedAt:      "2026-09-14T00:00:00Z",
			Phase:          1,
			Confidence:     0.5,
		}); err != nil {
			t.Fatalf("Add: %v", err)
		}
		entries, err := ls.List(learn.EntryFilter{})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		var entry *learn.Entry
		for i := range entries {
			if entries[i].Content == "a genuinely new learning entry" {
				entry = &entries[i]
			}
		}
		if entry == nil {
			t.Fatal("newly written learn entry not found")
		}
		if entry.SchemaVersion != colony.CurrentMemorySchemaVersion {
			t.Errorf("new learn entry SchemaVersion = %d, want %d (current)", entry.SchemaVersion, colony.CurrentMemorySchemaVersion)
		}
		if entry.Lineage == nil {
			t.Fatal("new learn entry Lineage is nil, want stamped")
		}
		if got := entry.Lineage.ResolvedProvenance(); got != colony.MemoryProvenanceRuntime {
			t.Errorf("new learn entry provenance = %q, want %q", got, colony.MemoryProvenanceRuntime)
		}
	})
}

// ---------------------------------------------------------------------------
// TestFutureSchemaVersionIsRefusedNotCoerced
// ---------------------------------------------------------------------------

func TestFutureSchemaVersionIsRefusedNotCoerced(t *testing.T) {
	if !memoryStoreSchemaReadable(memoryStoreLegacySchemaVersion) {
		t.Errorf("legacy version %d must be readable", memoryStoreLegacySchemaVersion)
	}
	if !memoryStoreSchemaReadable(memoryStoreSchemaVersion) {
		t.Errorf("current version %d must be readable", memoryStoreSchemaVersion)
	}
	future := memoryStoreSchemaVersion + 1
	if memoryStoreSchemaReadable(future) {
		t.Errorf("future version %d must NOT be readable -- refused by name, never coerced", future)
	}

	// Drive the real record type through a future-versioned raw byte
	// fixture (the runtime itself never writes a future version -- there
	// is no "current+1" writer to drive) and confirm no panic occurs and
	// the record is reported unreadable, not silently treated as current.
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)

	futureInstinctsJSON := `{"version":"1.0","instincts":[{"id":"inst_future_1","trigger":"t","action":"a","domain":"pattern","trust_score":0.5,"trust_tier":"bronze","confidence":0.5,"provenance":{"source":"s","source_type":"success_pattern","evidence":"single_phase","created_at":"2026-01-01T00:00:00Z","application_count":0},"application_history":[],"related_instincts":[],"archived":false,"schema_version":999}]}`
	if err := s.SaveRawJSON("instincts.json", []byte(futureInstinctsJSON)); err != nil {
		t.Fatalf("seed future-versioned instincts.json: %v", err)
	}

	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("loading a future-versioned record panicked: %v", r)
			}
		}()
		file := loadInstinctFileOrEmpty(s)
		if len(file.Instincts) != 1 {
			t.Fatalf("loaded %d instincts, want 1", len(file.Instincts))
		}
		entry := file.Instincts[0]
		if entry.SchemaVersion != 999 {
			t.Fatalf("SchemaVersion = %d, want 999 (read verbatim, not coerced)", entry.SchemaVersion)
		}
		if memoryStoreSchemaReadable(entry.SchemaVersion) {
			t.Errorf("a schema_version of 999 must be reported unreadable, not treated as current or legacy")
		}
	}()
}

// ---------------------------------------------------------------------------
// TestMemoryProvenanceVocabularyIsClosed
// ---------------------------------------------------------------------------

func TestMemoryProvenanceVocabularyIsClosed(t *testing.T) {
	names := colony.MemoryProvenanceNames()
	if len(names) != len(colony.MemoryProvenanceVocabulary) {
		t.Fatalf("MemoryProvenanceNames() has %d entries, MemoryProvenanceVocabulary has %d -- must be equal length", len(names), len(colony.MemoryProvenanceVocabulary))
	}

	const undeclared colony.MemoryProvenanceKind = "not-a-real-kind"
	if colony.MemoryProvenanceDeclared(undeclared) {
		t.Errorf("MemoryProvenanceDeclared(%q) = true, want false -- must be refused", undeclared)
	}

	err := colony.ValidateMemoryProvenanceKind(undeclared)
	if err == nil {
		t.Fatal("ValidateMemoryProvenanceKind(undeclared) = nil, want a refusal error")
	}
	if !strings.Contains(err.Error(), string(undeclared)) {
		t.Errorf("refusal error %q does not name the undeclared value %q", err.Error(), undeclared)
	}

	for _, want := range []colony.MemoryProvenanceKind{
		colony.MemoryProvenanceOwner,
		colony.MemoryProvenanceRuntime,
		colony.MemoryProvenanceLearning,
		colony.MemoryProvenanceImport,
		colony.MemoryProvenanceUnknown,
	} {
		if !colony.MemoryProvenanceDeclared(want) {
			t.Errorf("MemoryProvenanceDeclared(%q) = false, want true (declared member)", want)
		}
		if err := colony.ValidateMemoryProvenanceKind(want); err != nil {
			t.Errorf("ValidateMemoryProvenanceKind(%q) = %v, want nil (declared member)", want, err)
		}
	}
}

// ---------------------------------------------------------------------------
// TestMemoryStoreOrderingIsTotalAndStable
// ---------------------------------------------------------------------------

// TestMemoryStoreOrderingIsTotalAndStable seeds each store with two entries
// sharing the exact same timestamp, loads twice, and asserts the raw load
// is stable (JSON array order is preserved verbatim by encoding/json,
// proving loading twice never reorders anything) AND that the store's
// canonical recency sort (colony.SortInstinctEntriesByRecency /
// SortMiddenEntriesByRecency / learn.SortEntriesByRecency) breaks the tie
// deterministically by ID, producing the identical total order both times
// regardless of on-disk array order.
func TestMemoryStoreOrderingIsTotalAndStable(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	t.Run("instincts", func(t *testing.T) {
		tiedJSON := `{"version":"1.0","instincts":[
			{"id":"inst_b","trigger":"t","action":"a","domain":"pattern","trust_score":0.5,"trust_tier":"bronze","confidence":0.5,"provenance":{"source":"s","source_type":"success_pattern","evidence":"single_phase","created_at":"2026-01-01T00:00:00Z","application_count":0},"application_history":[],"related_instincts":[],"archived":false},
			{"id":"inst_a","trigger":"t2","action":"a2","domain":"pattern","trust_score":0.5,"trust_tier":"bronze","confidence":0.5,"provenance":{"source":"s2","source_type":"success_pattern","evidence":"single_phase","created_at":"2026-01-01T00:00:00Z","application_count":0},"application_history":[],"related_instincts":[],"archived":false}
		]}`
		if err := s.SaveRawJSON("instincts.json", []byte(tiedJSON)); err != nil {
			t.Fatalf("seed tied instincts.json: %v", err)
		}
		first := loadInstinctFileOrEmpty(s)
		second := loadInstinctFileOrEmpty(s)
		if !reflect.DeepEqual(first, second) {
			t.Fatalf("loading instincts.json twice produced different results:\nfirst:  %+v\nsecond: %+v", first, second)
		}

		sortedFirst := append([]colony.InstinctEntry{}, first.Instincts...)
		colony.SortInstinctEntriesByRecency(sortedFirst)
		sortedSecond := append([]colony.InstinctEntry{}, second.Instincts...)
		colony.SortInstinctEntriesByRecency(sortedSecond)
		if !reflect.DeepEqual(sortedFirst, sortedSecond) {
			t.Fatalf("canonical sort is not stable across two loads")
		}
		if len(sortedFirst) != 2 || sortedFirst[0].ID != "inst_a" || sortedFirst[1].ID != "inst_b" {
			t.Fatalf("canonical sort did not tie-break equal timestamps by ID ascending: %+v", sortedFirst)
		}
	})

	t.Run("midden", func(t *testing.T) {
		tiedJSON := `{"version":"1.0.0","entries":[
			{"id":"midden_b","timestamp":"2026-01-01T00:00:00Z","category":"c","source":"s","message":"m1","reviewed":false},
			{"id":"midden_a","timestamp":"2026-01-01T00:00:00Z","category":"c","source":"s","message":"m2","reviewed":false}
		]}`
		if err := s.SaveRawJSON(middenCanonicalPath, []byte(tiedJSON)); err != nil {
			t.Fatalf("seed tied midden.json: %v", err)
		}
		first, err := loadMiddenFile(s)
		if err != nil {
			t.Fatalf("loadMiddenFile: %v", err)
		}
		second, err := loadMiddenFile(s)
		if err != nil {
			t.Fatalf("loadMiddenFile: %v", err)
		}
		if !reflect.DeepEqual(first, second) {
			t.Fatalf("loading midden.json twice produced different results")
		}

		sortedFirst := append([]colony.MiddenEntry{}, first.Entries...)
		colony.SortMiddenEntriesByRecency(sortedFirst)
		sortedSecond := append([]colony.MiddenEntry{}, second.Entries...)
		colony.SortMiddenEntriesByRecency(sortedSecond)
		if !reflect.DeepEqual(sortedFirst, sortedSecond) {
			t.Fatalf("canonical sort is not stable across two loads")
		}
		if len(sortedFirst) != 2 || sortedFirst[0].ID != "midden_a" || sortedFirst[1].ID != "midden_b" {
			t.Fatalf("canonical sort did not tie-break equal timestamps by ID ascending: %+v", sortedFirst)
		}
	})

	t.Run("learn_entries", func(t *testing.T) {
		tiedJSON := `[
			{"id":"lrn_b","content":"c1","evidence":{"run_id":"r","phase":1,"workers":[],"gates_passed":1,"gates_total":1,"confidence":0.5,"timestamp":"2026-01-01T00:00:00Z","scope":"phase"},"classification":"repo-local","created_at":"2026-01-01T00:00:00Z","phase":1,"confidence":0.5,"status":"hypothesis"},
			{"id":"lrn_a","content":"c2","evidence":{"run_id":"r","phase":1,"workers":[],"gates_passed":1,"gates_total":1,"confidence":0.5,"timestamp":"2026-01-01T00:00:00Z","scope":"phase"},"classification":"repo-local","created_at":"2026-01-01T00:00:00Z","phase":1,"confidence":0.5,"status":"hypothesis"}
		]`
		if err := s.SaveRawJSON("entries.json", []byte(tiedJSON)); err != nil {
			t.Fatalf("seed tied entries.json: %v", err)
		}
		ls := learn.NewColonyStore(s)
		first, err := ls.List(learn.EntryFilter{})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		second, err := ls.List(learn.EntryFilter{})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if !reflect.DeepEqual(first, second) {
			t.Fatalf("loading entries.json twice produced different results")
		}

		sortedFirst := append([]learn.Entry{}, first...)
		learn.SortEntriesByRecency(sortedFirst)
		sortedSecond := append([]learn.Entry{}, second...)
		learn.SortEntriesByRecency(sortedSecond)
		if !reflect.DeepEqual(sortedFirst, sortedSecond) {
			t.Fatalf("canonical sort is not stable across two loads")
		}
		if len(sortedFirst) != 2 || sortedFirst[0].ID != "lrn_a" || sortedFirst[1].ID != "lrn_b" {
			t.Fatalf("canonical sort did not tie-break equal timestamps by ID ascending: %+v", sortedFirst)
		}
	})
}
