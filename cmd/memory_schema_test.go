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
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/learn"
	"github.com/calcosmic/Aether/pkg/memory"
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

// ---------------------------------------------------------------------------
// Task 2 (204-03-PLAN.md, SYN-204-05/06, LEARN-03): the typed application
// history, its real (never inferred) outcome, and the two shapes reading
// identically.
// ---------------------------------------------------------------------------

// TestTypedAndUntypedApplicationHistoryAgree builds one history in the old
// shape (real legacy bytes, unmarshalled through
// colony.InstinctApplicationEntry's own compatibility path -- the runtime
// itself can no longer produce this shape) and one in the new shape (a
// direct Go struct literal, exactly what every current writer produces),
// carrying the same underlying fact (one successful/helpful application),
// and asserts SummarizeInstinctApplications returns identical values for
// both.
func TestTypedAndUntypedApplicationHistoryAgree(t *testing.T) {
	const ts = "2026-09-14T00:00:00Z"

	typedEntry := colony.InstinctEntry{
		Provenance: colony.InstinctProvenance{CreatedAt: ts},
		ApplicationHistory: []colony.InstinctApplicationEntry{
			{Timestamp: ts, Outcome: "helpful"},
		},
	}

	legacyJSON := `{"id":"i","trigger":"t","action":"a","domain":"d","provenance":{"created_at":"` + ts + `"},"application_history":[{"timestamp":"` + ts + `","success":true}],"related_instincts":[],"archived":false}`
	var legacyEntry colony.InstinctEntry
	if err := json.Unmarshal([]byte(legacyJSON), &legacyEntry); err != nil {
		t.Fatalf("unmarshal legacy entry: %v", err)
	}

	typedSummary := memory.SummarizeInstinctApplications(typedEntry)
	legacySummary := memory.SummarizeInstinctApplications(legacyEntry)

	// The two shapes must agree on everything a legacy record can express:
	// how often the instinct was applied, how many of those the old
	// success flag counted, and when. The outcome counters 204-06 added
	// (HelpfulApplications/HarmfulApplications/IgnoredApplications) are
	// typed-only by design and are asserted separately below -- a legacy
	// entry's success flag was the unconditional "the phase advanced"
	// success this phase removes, so it is never read as evidence that the
	// guidance helped.
	typedShared := typedSummary
	typedShared.HelpfulApplications, typedShared.HarmfulApplications, typedShared.IgnoredApplications = 0, 0, 0
	legacyShared := legacySummary
	legacyShared.HelpfulApplications, legacyShared.HarmfulApplications, legacyShared.IgnoredApplications = 0, 0, 0
	if !reflect.DeepEqual(typedShared, legacyShared) {
		t.Fatalf("typed vs legacy summaries differ on the legacy-expressible fields: typed=%+v legacy=%+v", typedSummary, legacySummary)
	}
	if typedSummary.Applications != 1 || typedSummary.Successes != 1 || typedSummary.Failures != 0 {
		t.Fatalf("summary = %+v, want Applications=1 Successes=1 Failures=0 in both shapes", typedSummary)
	}
	if typedSummary.HelpfulApplications != 1 || typedSummary.HarmfulApplications != 0 || typedSummary.IgnoredApplications != 0 {
		t.Fatalf("typed summary outcome counters = helpful %d harmful %d ignored %d, want 1/0/0 from the recorded helpful outcome", typedSummary.HelpfulApplications, typedSummary.HarmfulApplications, typedSummary.IgnoredApplications)
	}
	if legacySummary.HelpfulApplications != 0 || legacySummary.HarmfulApplications != 0 || legacySummary.IgnoredApplications != 0 {
		t.Fatalf("legacy summary outcome counters = helpful %d harmful %d ignored %d, want 0/0/0: a legacy success flag is not evidence the guidance helped", legacySummary.HelpfulApplications, legacySummary.HarmfulApplications, legacySummary.IgnoredApplications)
	}
}

// mustFindInstinctApplicationEntry returns the ApplicationHistory entry
// recorded for (instinctID, phaseID), failing the test if none exists.
func mustFindInstinctApplicationEntry(t *testing.T, file colony.InstinctsFile, instinctID string, phaseID int) colony.InstinctApplicationEntry {
	t.Helper()
	for _, inst := range file.Instincts {
		if inst.ID != instinctID {
			continue
		}
		for _, entry := range inst.ApplicationHistory {
			if entry.Phase == phaseID {
				return entry
			}
		}
	}
	t.Fatalf("no application history entry found for instinct %s phase %d", instinctID, phaseID)
	return colony.InstinctApplicationEntry{}
}

// TestApplicationOutcomeComesFromCreditNotFromAdvancement drives the real
// phase-end path (recordPhaseApplicationCredit then
// recordInstinctApplicationsForPhase, the order this plan's Task 2
// established in cmd/consolidation_lifecycle.go) twice -- once where a real
// credit record ends up recording harmful, once where no decision delta
// exists so no credit record is ever recorded -- and asserts the two runs'
// recorded outcomes differ: the first carries harmful, the second carries
// pending, never a default or inferred success either way (Pitfall 2: an
// assertion that a history entry merely EXISTS would already pass against
// the pre-204-03 code and proves nothing).
func TestApplicationOutcomeComesFromCreditNotFromAdvancement(t *testing.T) {
	t.Run("a real credit record exists: entry carries its real outcome", func(t *testing.T) {
		saveGlobals(t)
		phase, instinct := newApplicationCreditFixture(t, applicationCreditFixtureOptions{
			InstinctContent:   "check pkg/colony/context_ranking.go before assuming score-based trim order",
			HasEarlierAttempt: true, EarlierFailed: nil, LatestFailed: []string{"tests"},
			HasDecisionDelta: true,
		})
		creditSummary := recordPhaseApplicationCredit(phase.ID)
		if !creditSummary.Ran || creditSummary.Harmful != 1 {
			t.Fatalf("credit summary = %+v, want Ran=true Harmful=1", creditSummary)
		}

		if n := recordInstinctApplicationsForPhase(phase.ID); n != 1 {
			t.Fatalf("recordInstinctApplicationsForPhase = %d, want 1", n)
		}

		file := loadInstinctFileOrEmpty(store)
		entry := mustFindInstinctApplicationEntry(t, file, instinct.ID, phase.ID)
		if entry.Outcome != string(recruitmentCreditOutcomeHarmful) {
			t.Fatalf("entry outcome = %q, want harmful", entry.Outcome)
		}
		if entry.CreditRecordID == "" {
			t.Fatal("entry carries no credit record id, want the harmful record's id")
		}
	})

	t.Run("no credit record exists: entry carries pending, never a default success", func(t *testing.T) {
		saveGlobals(t)
		phase, instinct := newApplicationCreditFixture(t, applicationCreditFixtureOptions{
			InstinctContent:   "check .planning/WINDOWS.md before assuming a known gap is unrecorded",
			HasEarlierAttempt: true, EarlierFailed: []string{"tests"}, LatestFailed: nil,
			HasDecisionDelta: false,
		})
		creditSummary := recordPhaseApplicationCredit(phase.ID)
		if !creditSummary.Ran || creditSummary.Recorded != 0 {
			t.Fatalf("credit summary = %+v, want Ran=true Recorded=0 (no decision delta to earn credit against)", creditSummary)
		}

		if n := recordInstinctApplicationsForPhase(phase.ID); n != 1 {
			t.Fatalf("recordInstinctApplicationsForPhase = %d, want 1", n)
		}

		file := loadInstinctFileOrEmpty(store)
		entry := mustFindInstinctApplicationEntry(t, file, instinct.ID, phase.ID)
		if entry.Outcome != string(recruitmentCreditOutcomePending) {
			t.Fatalf("entry outcome = %q, want pending -- never a default or inferred success", entry.Outcome)
		}
		if entry.CreditRecordID != "" {
			t.Fatalf("entry carries credit record id %q, want none", entry.CreditRecordID)
		}
	})
}

// TestMixedShapeHistoryReadsCorrectly builds one history holding two
// legacy-shape entries (unmarshalled from real legacy bytes) and one
// typed-shape entry (a direct Go struct literal) side by side -- a real
// colony will hold both shapes for months -- and asserts
// SummarizeInstinctApplications reads every entry correctly regardless of
// which shape it was written in.
func TestMixedShapeHistoryReadsCorrectly(t *testing.T) {
	const ts1, ts2, ts3 = "2026-09-01T00:00:00Z", "2026-09-02T00:00:00Z", "2026-09-03T00:00:00Z"

	legacyJSON := `{"id":"i","trigger":"t","action":"a","domain":"d","provenance":{"created_at":"` + ts1 + `"},"application_history":[` +
		`{"timestamp":"` + ts1 + `","success":true},` +
		`{"timestamp":"` + ts2 + `","success":false}` +
		`],"related_instincts":[],"archived":false}`
	var entry colony.InstinctEntry
	if err := json.Unmarshal([]byte(legacyJSON), &entry); err != nil {
		t.Fatalf("unmarshal mixed-shape fixture: %v", err)
	}
	// A typed entry, appended directly as a Go value -- never through JSON
	// -- exactly what a real writer (recordInstinctApplicationsForPhase,
	// PromoteService.Promote's dedup path) produces.
	entry.ApplicationHistory = append(entry.ApplicationHistory, colony.InstinctApplicationEntry{Timestamp: ts3, Outcome: "harmful"})

	summary := memory.SummarizeInstinctApplications(entry)
	if summary.Applications != 3 {
		t.Fatalf("Applications = %d, want 3 (2 legacy + 1 typed)", summary.Applications)
	}
	if summary.Successes != 1 {
		t.Fatalf("Successes = %d, want 1 (from the legacy success:true entry)", summary.Successes)
	}
	if summary.Failures != 2 {
		t.Fatalf("Failures = %d, want 2 (1 legacy success:false + 1 typed harmful)", summary.Failures)
	}
	if summary.LastApplied != ts3 {
		t.Fatalf("LastApplied = %q, want %q (the most recent timestamp across both shapes)", summary.LastApplied, ts3)
	}
}

// ---------------------------------------------------------------------------
// Task 3 (204-03-PLAN.md, LEARN-01): the field-level writer census, in the
// same structure as TestEveryMemoryPackPartHasALiveWriter /
// TestMemoryPackWriterExceptionsOnlyShrink
// (cmd/capsule_writer_invariant_198_2_test.go).
// ---------------------------------------------------------------------------

// memoryStoreCensusType names one live memory store's record type plus the
// short store-name prefix the field-level census namespaces its field keys
// with (several of the six stores share field names like id/timestamp/
// created_at/schema_version/lineage; an unqualified flat map would silently
// conflate them).
type memoryStoreCensusType struct {
	store string
	typ   reflect.Type
}

// liveMemoryStoreCensusTypes names the seven live memory store record
// types this census covers: the six named in 204-03-PLAN.md Task 3's own
// list (the instinct entry, the failure-log entry, the learning entry, the
// pheromone signal, the credit record, and the worker-handoff record) plus
// the durable episode ledger (episodeLedgerRecord, cmd/episode_ledger.go),
// which joined as the seventh store in 204-15 (SC3a) once every one of its
// previously writerless fields (WINDOWS.md entry 38) gained a real
// production writer.
func liveMemoryStoreCensusTypes() []memoryStoreCensusType {
	return []memoryStoreCensusType{
		{"instinct", reflect.TypeOf(colony.InstinctEntry{})},
		{"midden", reflect.TypeOf(colony.MiddenEntry{})},
		{"learn", reflect.TypeOf(learn.Entry{})},
		{"pheromone", reflect.TypeOf(colony.PheromoneSignal{})},
		{"credit", reflect.TypeOf(recruitmentCreditRecord{})},
		{"handoff", reflect.TypeOf(workerHandoffRecord{})},
		{"episode", reflect.TypeOf(episodeLedgerRecord{})},
	}
}

// censusFieldsForType returns every JSON field name t's own struct tags
// declare, namespaced "<store>.<field>" -- derived by reflecting over t's
// live field list every time this runs, never from a list typed into the
// test (a field added to or removed from the real struct is discovered
// automatically on the next run; see
// TestMemoryStoreCensusDiscoversFieldsByReflectionNotByList for the
// non-vacuous proof). A field tagged "-" is skipped, matching
// encoding/json's own semantics; a field with no json tag falls back to
// its own Go field name (none of the six census types this census covers
// have one).
func censusFieldsForType(store string, t reflect.Type) []string {
	fields := make([]string, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("json")
		if tag == "-" {
			continue
		}
		name := strings.Split(tag, ",")[0]
		if name == "" {
			name = f.Name
		}
		fields = append(fields, store+"."+name)
	}
	return fields
}

// discoveredMemoryStoreFields returns every census field across every type
// in types, namespaced and sorted.
func discoveredMemoryStoreFields(types []memoryStoreCensusType) []string {
	var all []string
	for _, ct := range types {
		all = append(all, censusFieldsForType(ct.store, ct.typ)...)
	}
	sort.Strings(all)
	return all
}

// memoryStoreCensusMissing returns every field in discovered that is named
// in none of writers, exceptions or retired -- the shared check both
// TestEveryMemoryStoreFieldHasALiveWriter and the synthetic non-vacuousness
// proof below reuse, so the two can never silently diverge on what
// "missing" means. A retired field (204-03-PLAN.md Task 4's owner
// decision) counts as accounted-for here exactly like a writer or an
// exception does -- retirement is validated separately, by
// validateMemoryStoreFieldRetirements, for the reason/agreement-date
// requirement a bare "is it in the map" check cannot express.
func memoryStoreCensusMissing(discovered []string, writers map[string]memoryStoreFieldWriter, exceptions map[string]string, retired map[string]memoryStoreRetiredField) []string {
	var missing []string
	for _, f := range discovered {
		if _, ok := writers[f]; ok {
			continue
		}
		if _, ok := exceptions[f]; ok {
			continue
		}
		if _, ok := retired[f]; ok {
			continue
		}
		missing = append(missing, f)
	}
	sort.Strings(missing)
	return missing
}

// validateMemoryStoreFieldRetirements returns every field in retired whose
// entry is missing a reason or the owner's recorded agreement date --
// retiring a field is this phase's one irreversible action (204-03-PLAN.md
// Task 4) and must never happen by omission. Shared by
// TestEveryMemoryStoreFieldHasALiveWriter (run against the real
// memoryStoreFieldRetired map) and
// TestRetiredFieldWithoutOwnerAgreementIsRefused (run against a synthetic
// map proving the check can genuinely fail), so the two can never silently
// diverge on what "refused" means.
func validateMemoryStoreFieldRetirements(retired map[string]memoryStoreRetiredField) []string {
	var problems []string
	for f, rf := range retired {
		if strings.TrimSpace(rf.reason) == "" || strings.TrimSpace(rf.ownerAgreedOn) == "" {
			problems = append(problems, f)
		}
	}
	sort.Strings(problems)
	return problems
}

// TestEveryMemoryStoreFieldHasALiveWriter is the LEARN-01 field-level
// invariant: every field on every live memory store's record type must
// have either a writer in memoryStoreFieldWriters, or a justified entry in
// memoryStoreFieldExceptions. It fails in both directions -- an
// undiscovered field is named, and so is an orphaned writer-map entry for
// a field that no longer exists -- so the map can never quietly rot.
func TestEveryMemoryStoreFieldHasALiveWriter(t *testing.T) {
	discovered := discoveredMemoryStoreFields(liveMemoryStoreCensusTypes())

	if missing := memoryStoreCensusMissing(discovered, memoryStoreFieldWriters, memoryStoreFieldExceptions, memoryStoreFieldRetired); len(missing) > 0 {
		t.Errorf("memory-store field(s) with no writer, no exception-list entry, and no retirement: %s -- give the field a writer, add a justified exception, or retire it with the owner's recorded agreement", strings.Join(missing, ", "))
	}

	discoveredSet := map[string]bool{}
	for _, f := range discovered {
		discoveredSet[f] = true
	}
	var staleWriters []string
	for f := range memoryStoreFieldWriters {
		if !discoveredSet[f] {
			staleWriters = append(staleWriters, f)
		}
	}
	if len(staleWriters) > 0 {
		sort.Strings(staleWriters)
		t.Errorf("writer-map entries for field(s) that no longer exist on any census record type: %s -- the writer map has gone stale, delete these entries", strings.Join(staleWriters, ", "))
	}

	for f, reason := range memoryStoreFieldExceptions {
		if strings.TrimSpace(reason) == "" {
			t.Errorf("memoryStoreFieldExceptions[%q] has no written reason", f)
		}
	}

	if problems := validateMemoryStoreFieldRetirements(memoryStoreFieldRetired); len(problems) > 0 {
		t.Errorf("retired field(s) missing a written reason or the owner's recorded agreement date: %s -- a retirement requires both, recorded in the same reviewed change", strings.Join(problems, ", "))
	}
}

// TestRetiredFieldWithoutOwnerAgreementIsRefused is the non-vacuousness
// proof the plan's own Task 4 resume instructions require: a retirement
// recorded without the owner's agreement date, or without a reason, must be
// refused by name. Driven against a synthetic map -- never the real
// memoryStoreFieldRetired, which must stay valid -- so this test can fail
// independently of whatever the real map currently contains.
func TestRetiredFieldWithoutOwnerAgreementIsRefused(t *testing.T) {
	synthetic := map[string]memoryStoreRetiredField{
		"synthetic.no_agreement_date": {reason: "a plausible-sounding reason", ownerAgreedOn: ""},
		"synthetic.no_reason":         {reason: "", ownerAgreedOn: "2026-09-14"},
		"synthetic.properly_recorded": {reason: "a plausible-sounding reason", ownerAgreedOn: "2026-09-14"},
	}
	problems := validateMemoryStoreFieldRetirements(synthetic)
	problemSet := map[string]bool{}
	for _, f := range problems {
		problemSet[f] = true
	}
	if !problemSet["synthetic.no_agreement_date"] {
		t.Errorf("expected synthetic.no_agreement_date to be refused for missing the owner's agreement date, got problems=%v", problems)
	}
	if !problemSet["synthetic.no_reason"] {
		t.Errorf("expected synthetic.no_reason to be refused for missing a reason, got problems=%v", problems)
	}
	if problemSet["synthetic.properly_recorded"] {
		t.Errorf("synthetic.properly_recorded carries both a reason and an agreement date -- it must NOT be refused, got problems=%v", problems)
	}
}

// TestMemoryStoreFieldExceptionsOnlyShrink is the ratchet: the exception
// list may only shrink. Widening memoryStoreFieldExceptionFloor is a
// deliberate, reviewed act, not something that can happen silently.
func TestMemoryStoreFieldExceptionsOnlyShrink(t *testing.T) {
	if len(memoryStoreFieldExceptions) > memoryStoreFieldExceptionFloor {
		t.Errorf("memoryStoreFieldExceptions has grown to %d entries, exceeding the recorded floor of %d -- the exception list may only shrink; give the new field(s) a writer, or widen the floor in the SAME reviewed change with a written reason for each new entry", len(memoryStoreFieldExceptions), memoryStoreFieldExceptionFloor)
	}
}

// TestMemoryStoreCensusDiscoversFieldsByReflectionNotByList is the
// non-vacuousness proof the plan's own acceptance criteria requires: a
// synthetic record type, never seen by memoryStoreFieldWriters or
// memoryStoreFieldExceptions, has its fields discovered and reported
// missing purely by reflecting over its own struct tags. A hardcoded field
// list could never discover a brand-new type's fields at all -- this is
// what proves the census is live, not a fixed list dressed up as one.
func TestMemoryStoreCensusDiscoversFieldsByReflectionNotByList(t *testing.T) {
	type syntheticRecord struct {
		Known    string `json:"known"`
		Unmapped string `json:"unmapped_field_never_in_writer_map"`
	}
	synthetic := []memoryStoreCensusType{{"synthetic", reflect.TypeOf(syntheticRecord{})}}
	discovered := discoveredMemoryStoreFields(synthetic)

	missing := memoryStoreCensusMissing(discovered, memoryStoreFieldWriters, memoryStoreFieldExceptions, memoryStoreFieldRetired)
	missingSet := map[string]bool{}
	for _, f := range missing {
		missingSet[f] = true
	}
	if !missingSet["synthetic.known"] || !missingSet["synthetic.unmapped_field_never_in_writer_map"] {
		t.Fatalf("expected both synthetic fields to be reported missing (proving reflection-based discovery), got discovered=%v missing=%v", discovered, missing)
	}
}
