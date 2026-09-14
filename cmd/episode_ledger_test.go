package cmd

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEpisodeLedgerIsAppendOnly drives the real writer twice for the same
// episode -- an open record, then a close record -- and asserts both are
// present, distinct, and that a third open-record write attempt for a
// SECOND, genuinely different episode does not disturb the first episode's
// records: the ledger only ever grows, never rewrites.
func TestEpisodeLedgerIsAppendOnly(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	openRecord, credited, err := recordEpisodeOutcome(episodeLedgerRecord{
		RecordKind:  episodeLedgerRecordKindOpened,
		EpisodeID:   "ep-append-1",
		EpisodeKind: "build",
		StartedAt:   "2026-09-01T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("record open: %v", err)
	}
	if !credited {
		t.Fatalf("expected the first open write to be credited")
	}

	closeRecord, credited, err := recordEpisodeOutcome(episodeLedgerRecord{
		RecordKind:     episodeLedgerRecordKindClosed,
		EpisodeID:      "ep-append-1",
		EpisodeKind:    "build",
		EndedAt:        "2026-09-01T00:05:00Z",
		TerminalResult: "completed",
	})
	if err != nil {
		t.Fatalf("record close: %v", err)
	}
	if !credited {
		t.Fatalf("expected the close write to be credited")
	}
	if openRecord.RecordID == closeRecord.RecordID {
		t.Fatalf("open and close records collapsed to the same id %q", openRecord.RecordID)
	}

	if _, _, err := recordEpisodeOutcome(episodeLedgerRecord{
		RecordKind:  episodeLedgerRecordKindOpened,
		EpisodeID:   "ep-append-2",
		EpisodeKind: "continue",
		StartedAt:   "2026-09-02T00:00:00Z",
	}); err != nil {
		t.Fatalf("record second episode's open: %v", err)
	}

	all, err := readEpisodeLedger()
	if err != nil {
		t.Fatalf("read ledger: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 records total, got %d: %+v", len(all), all)
	}

	firstEpisode, err := episodeLedgerForEpisode("ep-append-1")
	if err != nil {
		t.Fatalf("read episode: %v", err)
	}
	if len(firstEpisode) != 2 {
		t.Fatalf("expected 2 records for ep-append-1, got %d", len(firstEpisode))
	}
}

// TestEpisodeCloseWithoutOpenIsRefusedByName drives a close for an episode
// that was never opened and asserts the refusal names the episode, and
// that nothing was written.
func TestEpisodeCloseWithoutOpenIsRefusedByName(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	_, credited, err := recordEpisodeOutcome(episodeLedgerRecord{
		RecordKind:     episodeLedgerRecordKindClosed,
		EpisodeID:      "ep-never-opened",
		TerminalResult: "completed",
	})
	if err == nil {
		t.Fatal("expected an error closing an episode that was never opened")
	}
	if !strings.Contains(err.Error(), "ep-never-opened") {
		t.Fatalf("refusal %q does not name the episode", err.Error())
	}
	if credited {
		t.Fatal("expected credited=false on refusal")
	}

	all, err := readEpisodeLedger()
	if err != nil {
		t.Fatalf("read ledger: %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("expected nothing written after a refused close, got %d records", len(all))
	}
}

// TestEpisodeLedgerReplayWritesNothing drives the same open record twice
// and asserts the file's own bytes on disk are byte-identical before and
// after the second call -- not merely that the returned record looks the
// same, but that the write genuinely did not happen.
func TestEpisodeLedgerReplayWritesNothing(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	store = s

	first, credited, err := recordEpisodeOutcome(episodeLedgerRecord{
		RecordKind:  episodeLedgerRecordKindOpened,
		EpisodeID:   "ep-replay",
		EpisodeKind: "build",
		StartedAt:   "2026-09-01T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("record open: %v", err)
	}
	if !credited {
		t.Fatalf("expected the first write to be credited")
	}

	path := filepath.Join(tmpDir, ".aether", "data", episodeLedgerPath)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read ledger file: %v", err)
	}

	second, credited, err := recordEpisodeOutcome(episodeLedgerRecord{
		RecordKind:  episodeLedgerRecordKindOpened,
		EpisodeID:   "ep-replay",
		EpisodeKind: "build",
		StartedAt:   "2026-09-01T09:99:99Z", // a different wall-clock stamp
	})
	if err != nil {
		t.Fatalf("record replay open: %v", err)
	}
	if credited {
		t.Fatalf("expected the replayed write to be uncredited")
	}
	if second.RecordID != first.RecordID {
		t.Fatalf("replay returned a different record id: %q vs %q", second.RecordID, first.RecordID)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read ledger file after replay: %v", err)
	}
	if string(before) != string(after) {
		t.Fatalf("ledger file changed on replay:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

// TestUnreportedUsageIsAbsentNotZero drives one closed episode with no
// Usage/ReportedCostUSD supplied and one with an explicit, genuinely-zero
// reported cost, and asserts the reader can tell the two apart -- an
// unreported figure round-trips as a nil pointer, never a zero value that
// looks like a real measurement.
func TestUnreportedUsageIsAbsentNotZero(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	if _, _, err := recordEpisodeOutcome(episodeLedgerRecord{
		RecordKind: episodeLedgerRecordKindOpened, EpisodeID: "ep-unreported", StartedAt: "2026-09-01T00:00:00Z",
	}); err != nil {
		t.Fatalf("open ep-unreported: %v", err)
	}
	if _, _, err := recordEpisodeOutcome(episodeLedgerRecord{
		RecordKind: episodeLedgerRecordKindClosed, EpisodeID: "ep-unreported", EndedAt: "2026-09-01T00:01:00Z", TerminalResult: "completed",
	}); err != nil {
		t.Fatalf("close ep-unreported: %v", err)
	}

	zeroCost := 0.0
	if _, _, err := recordEpisodeOutcome(episodeLedgerRecord{
		RecordKind: episodeLedgerRecordKindOpened, EpisodeID: "ep-zero-reported", StartedAt: "2026-09-01T00:00:00Z",
	}); err != nil {
		t.Fatalf("open ep-zero-reported: %v", err)
	}
	if _, _, err := recordEpisodeOutcome(episodeLedgerRecord{
		RecordKind: episodeLedgerRecordKindClosed, EpisodeID: "ep-zero-reported", EndedAt: "2026-09-01T00:01:00Z",
		TerminalResult: "completed", ReportedCostUSD: &zeroCost,
	}); err != nil {
		t.Fatalf("close ep-zero-reported: %v", err)
	}

	all, err := readEpisodeLedger()
	if err != nil {
		t.Fatalf("read ledger: %v", err)
	}
	unreportedTerminal, ok := episodeLedgerTerminalRecord(all, "ep-unreported")
	if !ok {
		t.Fatal("expected a terminal record for ep-unreported")
	}
	if unreportedTerminal.ReportedCostUSD != nil {
		t.Fatalf("expected nil ReportedCostUSD for an unreported run, got %v", *unreportedTerminal.ReportedCostUSD)
	}
	if unreportedTerminal.Usage != nil {
		t.Fatalf("expected nil Usage for an unreported run, got %+v", unreportedTerminal.Usage)
	}

	zeroReportedTerminal, ok := episodeLedgerTerminalRecord(all, "ep-zero-reported")
	if !ok {
		t.Fatal("expected a terminal record for ep-zero-reported")
	}
	if zeroReportedTerminal.ReportedCostUSD == nil {
		t.Fatal("expected a non-nil ReportedCostUSD for a genuinely-reported zero cost")
	}
	if *zeroReportedTerminal.ReportedCostUSD != 0 {
		t.Fatalf("ReportedCostUSD = %v, want 0", *zeroReportedTerminal.ReportedCostUSD)
	}
	// A summary over these records reporting one unaccounted run (rather
	// than a zero cost) is proven by Task 3's own derived view test,
	// TestSpendSummaryNamesUnaccountedRuns -- this task's own scope ends at
	// the record shape (a nil pointer round-trips as absence), which the
	// two assertions above already establish.
}

// TestEqualDigestsCollapseAndDifferentTimestampsDoNot proves the digest
// rule this file's own doc comment states: two episode_closed writes for
// the SAME episode differing only in EndedAt collapse to one record
// (open/close kinds exclude timestamps from identity), while two
// intervention_recorded writes for the same episode, carrying identical
// content but different RecordedAt-bearing lineage, DO NOT collapse
// (intervention kind includes timestamp in identity).
func TestEqualDigestsCollapseAndDifferentTimestampsDoNot(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	if _, _, err := recordEpisodeOutcome(episodeLedgerRecord{
		RecordKind: episodeLedgerRecordKindOpened, EpisodeID: "ep-digest", StartedAt: "2026-09-01T00:00:00Z",
	}); err != nil {
		t.Fatalf("open: %v", err)
	}

	first, credited, err := recordEpisodeOutcome(episodeLedgerRecord{
		RecordKind: episodeLedgerRecordKindClosed, EpisodeID: "ep-digest", EndedAt: "2026-09-01T00:05:00Z", TerminalResult: "completed",
	})
	if err != nil || !credited {
		t.Fatalf("first close: credited=%v err=%v", credited, err)
	}
	second, credited, err := recordEpisodeOutcome(episodeLedgerRecord{
		RecordKind: episodeLedgerRecordKindClosed, EpisodeID: "ep-digest", EndedAt: "2026-09-01T00:09:00Z", TerminalResult: "completed",
	})
	if err != nil {
		t.Fatalf("second close: %v", err)
	}
	if credited {
		t.Fatal("expected the second close (differing only by EndedAt) to be a replay, not credited")
	}
	if second.RecordID != first.RecordID {
		t.Fatalf("close records with different EndedAt should collapse to one id, got %q vs %q", first.RecordID, second.RecordID)
	}

	firstIntervention, credited, err := recordEpisodeOutcome(episodeLedgerRecord{
		RecordKind: episodeLedgerRecordKindIntervention, EpisodeID: "ep-digest",
		StartedAt: "2026-09-01T01:00:00Z", Interventions: []string{"owner declined a forced reviewer"},
	})
	if err != nil || !credited {
		t.Fatalf("first intervention: credited=%v err=%v", credited, err)
	}
	secondIntervention, credited, err := recordEpisodeOutcome(episodeLedgerRecord{
		RecordKind: episodeLedgerRecordKindIntervention, EpisodeID: "ep-digest",
		StartedAt: "2026-09-01T02:00:00Z", Interventions: []string{"owner declined a forced reviewer"},
	})
	if err != nil {
		t.Fatalf("second intervention: %v", err)
	}
	if !credited {
		t.Fatal("expected the second, genuinely-later intervention to be credited as a distinct record")
	}
	if secondIntervention.RecordID == firstIntervention.RecordID {
		t.Fatalf("two interventions at different timestamps collapsed to one id %q", firstIntervention.RecordID)
	}
}

// TestEpisodeLedgerOrderingIsTotalAndStable loads the ledger twice after
// writing records that share a timestamp, and asserts identical order both
// times, with the record identifier breaking the tie.
func TestEpisodeLedgerOrderingIsTotalAndStable(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	for _, id := range []string{"ep-order-c", "ep-order-a", "ep-order-b"} {
		if _, _, err := recordEpisodeOutcome(episodeLedgerRecord{
			RecordKind: episodeLedgerRecordKindOpened, EpisodeID: id, StartedAt: "2026-09-01T00:00:00Z",
		}); err != nil {
			t.Fatalf("open %s: %v", id, err)
		}
	}

	firstRead, err := readEpisodeLedger()
	if err != nil {
		t.Fatalf("first read: %v", err)
	}
	secondRead, err := readEpisodeLedger()
	if err != nil {
		t.Fatalf("second read: %v", err)
	}
	if len(firstRead) != 3 || len(secondRead) != 3 {
		t.Fatalf("expected 3 records on both reads, got %d and %d", len(firstRead), len(secondRead))
	}
	for i := range firstRead {
		if firstRead[i].RecordID != secondRead[i].RecordID {
			t.Fatalf("ordering differs between reads at index %d: %q vs %q", i, firstRead[i].RecordID, secondRead[i].RecordID)
		}
	}
	for i := 1; i < len(firstRead); i++ {
		if firstRead[i-1].RecordID > firstRead[i].RecordID {
			t.Fatalf("records sharing a timestamp are not ordered by record id ascending: %q before %q", firstRead[i-1].RecordID, firstRead[i].RecordID)
		}
	}
}

// TestEpisodeLedgerHasOneWriter is an AST-based scan of the cmd package
// mirroring cmd/recruitment_credit_test.go's TestCreditRequiresBothFacts:
// it derives every function whose body writes episodes/ledger.json via the
// store, and asserts recordEpisodeOutcome is the only one.
func TestEpisodeLedgerHasOneWriter(t *testing.T) {
	violations := scanForEpisodeLedgerWritesOutsideRecordEpisodeOutcome(t, ".")
	if len(violations) != 0 {
		t.Fatalf("found an episode-ledger write outside recordEpisodeOutcome:\n%s", strings.Join(violations, "\n"))
	}

	t.Run("a synthetic second writer is caught", func(t *testing.T) {
		fixtureSrc := `package cmd

func sneakilyWriteEpisodeLedger(episodeID string) error {
	var file episodeLedgerFile
	return store.UpdateJSONAtomically(episodeLedgerPath, &file, func() error {
		file.Entries = append(file.Entries, episodeLedgerRecord{EpisodeID: episodeID})
		return nil
	})
}
`
		violations := scanEpisodeLedgerSourceForViolations(t, "fixture_episode_ledger_second_writer.go", fixtureSrc)
		if len(violations) == 0 {
			t.Fatal("scanner failed to detect a synthetic second writer of episodes/ledger.json")
		}
		if !strings.Contains(violations[0], "sneakilyWriteEpisodeLedger") {
			t.Fatalf("violation %q does not name the offending function", violations[0])
		}
		if !strings.Contains(violations[0], "fixture_episode_ledger_second_writer.go") {
			t.Fatalf("violation %q does not name the offending file", violations[0])
		}
	})
}

// TestEpisodeLedgerRecordKindVocabularyIsComplete asserts the record-kind
// const block and its names slice have equal length -- a kind added to one
// without the other is caught here rather than silently drifting.
func TestEpisodeLedgerRecordKindVocabularyIsComplete(t *testing.T) {
	if len(episodeLedgerRecordKindVocabulary) != len(episodeLedgerRecordKindNames()) {
		t.Fatalf("episodeLedgerRecordKindVocabulary has %d members but episodeLedgerRecordKindNames() returned %d",
			len(episodeLedgerRecordKindVocabulary), len(episodeLedgerRecordKindNames()))
	}
	if len(episodeLedgerRecordKindVocabulary) != 3 {
		t.Fatalf("expected exactly 3 declared record kinds (opened, closed, intervention), got %d", len(episodeLedgerRecordKindVocabulary))
	}
}

func scanForEpisodeLedgerWritesOutsideRecordEpisodeOutcome(t *testing.T, dir string) []string {
	t.Helper()
	names, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatalf("glob cmd package files: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("fixture is broken: no .go files found in the cmd package directory")
	}
	fset := token.NewFileSet()
	var violations []string
	found := false
	for _, name := range names {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		fileFound, fileViolations := episodeLedgerWriteViolationsInFile(fset, file)
		found = found || fileFound
		violations = append(violations, fileViolations...)
	}
	if !found {
		t.Fatal("fixture is broken: no write call referencing episodeLedgerPath was found anywhere in the cmd package")
	}
	return violations
}

func scanEpisodeLedgerSourceForViolations(t *testing.T, filename, src string) []string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, 0)
	if err != nil {
		t.Fatalf("parse fixture source: %v", err)
	}
	_, violations := episodeLedgerWriteViolationsInFile(fset, file)
	return violations
}

// episodeLedgerWriteViolationsInFile walks file for every call writing
// episodeLedgerPath through the store, and requires the enclosing function
// to be literally named recordEpisodeOutcome.
func episodeLedgerWriteViolationsInFile(fset *token.FileSet, file *ast.File) (found bool, violations []string) {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "UpdateJSONAtomically" {
				return true
			}
			if len(call.Args) == 0 {
				return true
			}
			ident, ok := call.Args[0].(*ast.Ident)
			if !ok || ident.Name != "episodeLedgerPath" {
				return true
			}
			found = true
			fnName := "<unknown>"
			if fn.Name != nil {
				fnName = fn.Name.Name
			}
			if fnName != "recordEpisodeOutcome" {
				violations = append(violations, fmt.Sprintf(
					"%s: %s writes episodeLedgerPath via store.UpdateJSONAtomically -- only recordEpisodeOutcome may write it",
					fset.Position(call.Pos()).String(), fnName,
				))
			}
			return true
		})
	}
	return found, violations
}
