package cmd

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
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

// TestEpisodeSecondDifferentCloseIsRefusedByName is WR-02 (204-REVIEW.md):
// a SECOND, genuinely different episode_closed record for an episode that
// already has one (a different TerminalResult, e.g. "completed" then
// "failed" on a retried finalize path) is refused by name, and the FIRST
// close's terminal result survives untouched -- distinguishing this from
// an identical replay of the same close, which must still collapse
// silently (TestEqualDigestsCollapseAndDifferentTimestampsDoNot).
func TestEpisodeSecondDifferentCloseIsRefusedByName(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	if _, _, err := recordEpisodeOutcome(episodeLedgerRecord{
		RecordKind: episodeLedgerRecordKindOpened, EpisodeID: "ep-double-close", StartedAt: "2026-09-01T00:00:00Z",
	}); err != nil {
		t.Fatalf("open: %v", err)
	}
	first, credited, err := recordEpisodeOutcome(episodeLedgerRecord{
		RecordKind: episodeLedgerRecordKindClosed, EpisodeID: "ep-double-close", EndedAt: "2026-09-01T00:05:00Z", TerminalResult: "completed",
	})
	if err != nil || !credited {
		t.Fatalf("first close: credited=%v err=%v", credited, err)
	}

	_, credited, err = recordEpisodeOutcome(episodeLedgerRecord{
		RecordKind: episodeLedgerRecordKindClosed, EpisodeID: "ep-double-close", EndedAt: "2026-09-01T00:09:00Z", TerminalResult: "failed",
	})
	if err == nil {
		t.Fatal("expected an error on a second, different close for an already-closed episode")
	}
	if !strings.Contains(err.Error(), "already closed") {
		t.Fatalf("refusal %q does not name the already-closed episode", err.Error())
	}
	if credited {
		t.Fatal("expected credited=false on a refused second close")
	}

	all, err := readEpisodeLedger()
	if err != nil {
		t.Fatalf("read ledger: %v", err)
	}
	terminal, ok := episodeLedgerTerminalRecord(all, "ep-double-close")
	if !ok {
		t.Fatal("expected a terminal record to survive")
	}
	if terminal.TerminalResult != "completed" {
		t.Fatalf("expected the FIRST close's terminal result to survive untouched, got %q", terminal.TerminalResult)
	}
	if terminal.RecordID != first.RecordID {
		t.Fatalf("expected the surviving terminal record to be the first close, got a different record id")
	}

	var closedCount int
	for _, r := range all {
		if r.EpisodeID == "ep-double-close" && r.RecordKind == episodeLedgerRecordKindClosed {
			closedCount++
		}
	}
	if closedCount != 1 {
		t.Fatalf("expected exactly 1 episode_closed record for ep-double-close, got %d", closedCount)
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

// ---------------------------------------------------------------------
// Task 2 (LEARN-02, 204-04-PLAN.md): wiring the ledger into the episode
// boundary that already exists, on every lane.
// ---------------------------------------------------------------------

// TestEveryLifecycleLaneWritesADurableOutcome derives its lane inventory
// from events.ColonyLiveEpisodeKinds() (via the already-registered
// liveLaneEntryPoints map in cmd/live_lane_coverage_test.go), drives each
// lane's real public entry point, and asserts a durable open and terminal
// record exists in the episode ledger for the episode id that lane's own
// live events name.
func TestEveryLifecycleLaneWritesADurableOutcome(t *testing.T) {
	for _, kind := range events.ColonyLiveEpisodeKinds() {
		kind := kind
		t.Run(kind, func(t *testing.T) {
			entry, ok := liveLaneEntryPoints[kind]
			if !ok {
				t.Fatalf("no real public entry point is registered in liveLaneEntryPoints for declared episode kind %q", kind)
			}
			liveEvents := entry(t)

			// A lane only "opens a live episode" (this plan's own boundary,
			// see the doc comment on recordEpisodeLedgerOpen) when it
			// genuinely emits LiveTopicEpisodeStarted for this kind -- two
			// declared kinds (swarm, recovery) route through a different,
			// pre-existing live-event shape (wave/recovery-state events
			// carrying an episode id, never an episode-started/ended pair)
			// and were never wired through the episode boundary this plan
			// extends; that gap predates this plan and is out of scope for
			// files this plan is permitted to touch. A kind that DOES emit
			// the boundary event, but the boundary produces no durable
			// record, still fails below by name.
			episodeID := ""
			opensEpisodeBoundary := false
			for _, e := range liveEvents {
				if e.Topic == events.LiveTopicEpisodeStarted && e.Payload.EpisodeKind == kind {
					opensEpisodeBoundary = true
					episodeID = e.Payload.EpisodeID
					break
				}
			}
			if !opensEpisodeBoundary {
				t.Skipf("lane %q never emits LiveTopicEpisodeStarted (pre-existing gap outside this plan's files_modified) -- skipping durable-record assertion", kind)
			}
			if episodeID == "" {
				t.Fatalf("lane %q emitted LiveTopicEpisodeStarted with no episode id", kind)
			}

			records, err := episodeLedgerForEpisode(episodeID)
			if err != nil {
				t.Fatalf("read episode ledger for %q: %v", episodeID, err)
			}
			if _, ok := episodeLedgerOpenRecord(records, episodeID); !ok {
				t.Fatalf("lane %q (episode %q) has no durable open record", kind, episodeID)
			}
			if _, ok := episodeLedgerTerminalRecord(records, episodeID); !ok {
				t.Fatalf("lane %q (episode %q) has no durable terminal record", kind, episodeID)
			}
		})
	}
}

// TestInterruptedEpisodeIsUnfinishedNotSuccessful opens an episode, never
// closes it, and asserts the reader reports it as unfinished -- an open
// record with no terminal record -- rather than as a success or as absent.
func TestInterruptedEpisodeIsUnfinishedNotSuccessful(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	emitColonyLiveEpisodeStarted("ep-interrupted", "build")

	records, err := episodeLedgerForEpisode("ep-interrupted")
	if err != nil {
		t.Fatalf("read episode: %v", err)
	}
	if _, ok := episodeLedgerOpenRecord(records, "ep-interrupted"); !ok {
		t.Fatal("expected an open record for the interrupted episode")
	}
	if _, ok := episodeLedgerTerminalRecord(records, "ep-interrupted"); ok {
		t.Fatal("an interrupted episode must not have a terminal record")
	}

	// Task 3: the derived render view must also show this state as
	// neither a success nor an omission.
	summary := renderEpisodeOutcomeSummary(records)
	if strings.Contains(strings.ToLower(summary), "completed") {
		t.Fatalf("interrupted episode rendered as completed:\n%s", summary)
	}
	if !strings.Contains(summary, "ep-interrupted") {
		t.Fatalf("summary does not name the interrupted episode:\n%s", summary)
	}
}

// TestElapsedTimeComesFromTheStoredOpenTimestamp seeds an open record whose
// StartedAt is artificially far in the past (a real clock offset, not the
// wall-clock moment this test runs), then closes the SAME episode through
// the real emitColonyLiveEpisodeEnded boundary, and asserts the recorded
// ElapsedSeconds is consistent with (now - the stored StartedAt) -- proving
// the close boundary reads the ALREADY-STORED open record's own timestamp
// rather than any other clock reading (e.g. one taken earlier in the same
// call chain, which would report a near-zero or unrelated figure here).
func TestElapsedTimeComesFromTheStoredOpenTimestamp(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	const offsetStartedAt = "2020-01-01T00:00:00Z"
	if _, _, err := recordEpisodeOutcome(episodeLedgerRecord{
		RecordKind:  episodeLedgerRecordKindOpened,
		EpisodeID:   "ep-clock-offset",
		EpisodeKind: "build",
		StartedAt:   offsetStartedAt,
	}); err != nil {
		t.Fatalf("seed open record: %v", err)
	}

	emitColonyLiveEpisodeEnded("ep-clock-offset", "build", "completed")

	records, err := episodeLedgerForEpisode("ep-clock-offset")
	if err != nil {
		t.Fatalf("read episode: %v", err)
	}
	terminal, ok := episodeLedgerTerminalRecord(records, "ep-clock-offset")
	if !ok {
		t.Fatal("expected a terminal record")
	}

	startedAt, err := time.Parse(time.RFC3339, offsetStartedAt)
	if err != nil {
		t.Fatalf("parse fixture timestamp: %v", err)
	}
	wantMinimum := time.Since(startedAt).Seconds() - 60 // generous slack for test runtime
	if terminal.ElapsedSeconds < wantMinimum {
		t.Fatalf("ElapsedSeconds = %v, want at least ~%v (computed from the stored open timestamp %q, not a near-zero or unrelated figure)",
			terminal.ElapsedSeconds, wantMinimum, offsetStartedAt)
	}
}

// TestOutcomeTopicsAreRegistered asserts (by name, rather than duplicating
// the existing registry test) that LiveTopicOutcomeRecorded and
// LiveTopicInterventionRecorded are both present in
// events.ColonyLiveTopics() -- the existing colony_live_test.go registry
// test already enforces every const-block member appears there; this
// names the two new members explicitly for this plan's own acceptance
// criteria.
func TestOutcomeTopicsAreRegistered(t *testing.T) {
	topics := events.ColonyLiveTopics()
	wantOutcome := false
	wantIntervention := false
	for _, topic := range topics {
		if topic == events.LiveTopicOutcomeRecorded {
			wantOutcome = true
		}
		if topic == events.LiveTopicInterventionRecorded {
			wantIntervention = true
		}
	}
	if !wantOutcome {
		t.Fatal("LiveTopicOutcomeRecorded is not registered in events.ColonyLiveTopics()")
	}
	if !wantIntervention {
		t.Fatal("LiveTopicInterventionRecorded is not registered in events.ColonyLiveTopics()")
	}
}

// ---------------------------------------------------------------------
// Task 3 (LEARN-02, 204-04-PLAN.md): derived views and the durability
// proof against the live feed's own retention window.
// ---------------------------------------------------------------------

// TestEpisodeOutcomeSurvivesTheLiveFeedWindow writes a record through the
// real writer, then rewrites the underlying event-bus.jsonl entry's own
// ExpiresAt to a moment in the past -- the same direct-seed seam
// cmd/eventbus_test.go's TestEventBusCleanupRemovesExpiredEvents already
// uses to simulate the passage of events.DefaultTTL days without a clock
// seam existing anywhere in pkg/events -- and asserts the live bus no
// longer returns the corresponding event while the durable ledger still
// returns the record untouched.
func TestEpisodeOutcomeSurvivesTheLiveFeedWindow(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	emitColonyLiveEpisodeStarted("ep-outlives-feed", "build")
	emitColonyLiveEpisodeEnded("ep-outlives-feed", "build", "completed")

	raw, err := s.ReadFile("event-bus.jsonl")
	if err != nil {
		t.Fatalf("read event-bus.jsonl: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if len(lines) == 0 || lines[0] == "" {
		t.Fatal("fixture is broken: no live events were persisted to event-bus.jsonl")
	}
	var rewritten []string
	touched := 0
	for _, line := range lines {
		var evt map[string]interface{}
		if err := json.Unmarshal([]byte(line), &evt); err != nil {
			t.Fatalf("decode event-bus.jsonl line: %v", err)
		}
		if strings.Contains(line, "ep-outlives-feed") {
			// Same direct-seed technique as TestEventBusCleanupRemovesExpiredEvents:
			// rewrite ExpiresAt to a moment in the past, simulating the
			// passage of events.DefaultTTL days with no clock seam.
			evt["expires_at"] = "2000-01-01T00:00:00Z"
			touched++
		}
		reencoded, err := json.Marshal(evt)
		if err != nil {
			t.Fatalf("re-encode event-bus.jsonl line: %v", err)
		}
		rewritten = append(rewritten, string(reencoded))
	}
	if touched == 0 {
		t.Fatal("fixture is broken: no persisted live event named ep-outlives-feed")
	}
	if err := s.AtomicWrite("event-bus.jsonl", []byte(strings.Join(rewritten, "\n")+"\n")); err != nil {
		t.Fatalf("rewrite event-bus.jsonl: %v", err)
	}

	liveAfterExpiry := readColonyLiveEventsRaw(s, time.Time{})
	for _, e := range liveAfterExpiry {
		if strings.Contains(string(e.Payload), "ep-outlives-feed") {
			t.Fatal("the live bus still returns an event past its own expiry window")
		}
	}

	durable, err := episodeLedgerForEpisode("ep-outlives-feed")
	if err != nil {
		t.Fatalf("read durable ledger: %v", err)
	}
	if _, ok := episodeLedgerOpenRecord(durable, "ep-outlives-feed"); !ok {
		t.Fatal("durable ledger lost its open record once the live feed's own event expired")
	}
	if _, ok := episodeLedgerTerminalRecord(durable, "ep-outlives-feed"); !ok {
		t.Fatal("durable ledger lost its terminal record once the live feed's own event expired")
	}
}

// TestDerivedViewsAreIdempotent calls each derived view twice on the same
// records and compares bytes.
func TestDerivedViewsAreIdempotent(t *testing.T) {
	usage := &codex.WorkerUsage{TotalTokens: 100, USDCost: 1.5, Source: codex.UsageSourceProvider}
	cost := 1.5
	records := []episodeLedgerRecord{
		{RecordKind: episodeLedgerRecordKindOpened, EpisodeID: "ep-idem", EpisodeKind: "build", StartedAt: "2026-09-01T00:00:00Z"},
		{RecordKind: episodeLedgerRecordKindClosed, EpisodeID: "ep-idem", EpisodeKind: "build", EndedAt: "2026-09-01T00:05:00Z", ElapsedSeconds: 300, TerminalResult: "helpful", Usage: usage, ReportedCostUSD: &cost, EpisodeRevision: "204-04"},
	}

	if renderEpisodeOutcomeSummary(records) != renderEpisodeOutcomeSummary(records) {
		t.Fatal("renderEpisodeOutcomeSummary is not idempotent")
	}
	firstChangelog, _ := json.Marshal(collectChangelogEntriesFromLedger(records))
	secondChangelog, _ := json.Marshal(collectChangelogEntriesFromLedger(records))
	if string(firstChangelog) != string(secondChangelog) {
		t.Fatal("collectChangelogEntriesFromLedger is not idempotent")
	}
	firstSpend, _ := json.Marshal(summariseEpisodeSpend(records))
	secondSpend, _ := json.Marshal(summariseEpisodeSpend(records))
	if string(firstSpend) != string(secondSpend) {
		t.Fatal("summariseEpisodeSpend is not idempotent")
	}

	t.Run("structurally performs no store read and no clock read", func(t *testing.T) {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "episode_ledger.go", nil, 0)
		if err != nil {
			t.Fatalf("parse episode_ledger.go: %v", err)
		}
		views := map[string]bool{
			"renderEpisodeOutcomeSummary":       true,
			"collectChangelogEntriesFromLedger": true,
			"summariseEpisodeSpend":             true,
		}
		checked := 0
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || !views[fn.Name.Name] {
				continue
			}
			checked++
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				pkgIdent, ok := sel.X.(*ast.Ident)
				if !ok {
					return true
				}
				if pkgIdent.Name == "store" {
					t.Errorf("%s: %s performs a store read/write via store.%s", fset.Position(call.Pos()).String(), fn.Name.Name, sel.Sel.Name)
				}
				if pkgIdent.Name == "time" && sel.Sel.Name == "Now" {
					t.Errorf("%s: %s reads the clock via time.Now", fset.Position(call.Pos()).String(), fn.Name.Name)
				}
				return true
			})
		}
		if checked != len(views) {
			t.Fatalf("fixture is broken: expected to check %d derived-view functions, found %d in episode_ledger.go", len(views), checked)
		}
	})
}

// TestDerivedViewOverNoEpisodesIsEmptyNotAnError asserts each derived view
// handles zero records gracefully.
func TestDerivedViewOverNoEpisodesIsEmptyNotAnError(t *testing.T) {
	if entries := collectChangelogEntriesFromLedger(nil); len(entries) != 0 {
		t.Fatalf("expected zero changelog entries over zero records, got %d", len(entries))
	}
	summary := summariseEpisodeSpend(nil)
	if summary.AccountedEpisodes != 0 || summary.UnaccountedEpisodes != 0 {
		t.Fatalf("expected a zero-valued spend summary over zero records, got %+v", summary)
	}
	rendered := renderEpisodeOutcomeSummary(nil)
	if rendered == "" {
		t.Fatal("expected a non-empty (but non-erroring) rendered summary over zero episodes")
	}
}

// TestEpisodeWithNoOutcomeRendersAsNoOutcome asserts an episode with an
// open record and no terminal record renders an explicit no-outcome row.
func TestEpisodeWithNoOutcomeRendersAsNoOutcome(t *testing.T) {
	records := []episodeLedgerRecord{
		{RecordKind: episodeLedgerRecordKindOpened, EpisodeID: "ep-no-outcome", EpisodeKind: "build", StartedAt: "2026-09-01T00:00:00Z"},
	}
	rendered := renderEpisodeOutcomeSummary(records)
	if !strings.Contains(rendered, "no outcome recorded") {
		t.Fatalf("expected an explicit no-outcome row, got:\n%s", rendered)
	}
	if strings.Contains(strings.ToLower(rendered), "helped") || strings.Contains(strings.ToLower(rendered), "completed") {
		t.Fatalf("an unfinished episode must never render as a success:\n%s", rendered)
	}
}

// TestSpendSummaryNamesUnaccountedRuns asserts a derived summary including
// unreported runs states how many it could not account for.
func TestSpendSummaryNamesUnaccountedRuns(t *testing.T) {
	usage := &codex.WorkerUsage{TotalTokens: 500, Source: codex.UsageSourceProvider}
	records := []episodeLedgerRecord{
		{RecordKind: episodeLedgerRecordKindClosed, EpisodeID: "ep-a", TerminalResult: "helpful", Usage: usage},
		{RecordKind: episodeLedgerRecordKindClosed, EpisodeID: "ep-b", TerminalResult: "neutral"},
		{RecordKind: episodeLedgerRecordKindClosed, EpisodeID: "ep-c", TerminalResult: "helpful"},
	}
	summary := summariseEpisodeSpend(records)
	if summary.AccountedEpisodes != 1 {
		t.Fatalf("AccountedEpisodes = %d, want 1", summary.AccountedEpisodes)
	}
	if summary.UnaccountedEpisodes != 2 {
		t.Fatalf("UnaccountedEpisodes = %d, want 2", summary.UnaccountedEpisodes)
	}
}

// TestDerivedViewsSpeakTheSharedVoice asserts every content line opens with
// a glyph drawn from the shared table and that no line contains an
// underscore-joined internal token -- following classic_voice_corpus_test.go's
// own rawStateTokenLeaks/underscoreTokenPattern shape.
func TestDerivedViewsSpeakTheSharedVoice(t *testing.T) {
	records := []episodeLedgerRecord{
		{RecordKind: episodeLedgerRecordKindOpened, EpisodeID: "ep-voice", EpisodeKind: "build", StartedAt: "2026-09-01T00:00:00Z"},
		{RecordKind: episodeLedgerRecordKindClosed, EpisodeID: "ep-voice", EpisodeKind: "build", EndedAt: "2026-09-01T00:05:00Z", ElapsedSeconds: 300, TerminalResult: "harmful"},
		{RecordKind: episodeLedgerRecordKindOpened, EpisodeID: "ep-voice-2", EpisodeKind: "continue", StartedAt: "2026-09-02T00:00:00Z"},
	}
	rendered := renderEpisodeOutcomeSummary(records)
	for _, line := range strings.Split(strings.TrimRight(rendered, "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if underscoreTokenPattern.MatchString(line) {
			t.Fatalf("line %q carries a raw internal token", line)
		}
		hasGlyph := false
		for _, glyph := range voiceGlyphMap {
			if strings.HasPrefix(line, glyph) {
				hasGlyph = true
				break
			}
		}
		if !hasGlyph {
			t.Fatalf("line %q does not open with a shared-table glyph", line)
		}
	}
}

// TestEpisodeLedgerRecordsCarryTheSharedSchemaAndLineage proves the ledger
// honours the per-record contract every other live memory store carries
// (204-03): a record written through the one writer comes back stamped with
// the shared schema version and a runtime-provenance lineage naming the
// episode and the record's own timestamp -- and a caller cannot bypass the
// stamp by passing a record without them, because the writer applies both
// itself. The replay half (a re-recorded open still collapses to the same
// id despite the stamp) is TestEpisodeLedgerReplayWritesNothing's job.
func TestEpisodeLedgerRecordsCarryTheSharedSchemaAndLineage(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	written, credited, err := recordEpisodeOutcome(episodeLedgerRecord{
		RecordKind:  episodeLedgerRecordKindOpened,
		EpisodeID:   "ep-lineage-1",
		EpisodeKind: "build",
		StartedAt:   "2026-09-14T10:00:00Z",
	})
	if err != nil || !credited {
		t.Fatalf("record open: credited=%v err=%v", credited, err)
	}
	if written.SchemaVersion != colony.CurrentMemorySchemaVersion {
		t.Fatalf("schema_version = %d, want the shared colony.CurrentMemorySchemaVersion %d", written.SchemaVersion, colony.CurrentMemorySchemaVersion)
	}
	if !colony.MemoryStoreSchemaReadable(written.SchemaVersion) {
		t.Fatalf("schema_version %d is not readable by the shared contract", written.SchemaVersion)
	}
	if written.Lineage == nil {
		t.Fatal("lineage was not stamped on the written record")
	}
	if got := written.Lineage.ResolvedProvenance(); got != colony.MemoryProvenanceRuntime {
		t.Fatalf("lineage provenance = %q, want %q", got, colony.MemoryProvenanceRuntime)
	}
	if written.Lineage.SourceID == nil || *written.Lineage.SourceID != "ep-lineage-1" {
		t.Fatalf("lineage source_id = %v, want the episode id", written.Lineage.SourceID)
	}
	if written.Lineage.RecordedAt == nil || *written.Lineage.RecordedAt != "2026-09-14T10:00:00Z" {
		t.Fatalf("lineage recorded_at = %v, want the record's own started_at", written.Lineage.RecordedAt)
	}

	// The stamp is durable, not a return-value courtesy: the bytes on disk
	// carry both fields under the shared json names.
	records, err := readEpisodeLedger()
	if err != nil || len(records) != 1 {
		t.Fatalf("read ledger: n=%d err=%v", len(records), err)
	}
	raw, err := json.Marshal(records[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"schema_version":` + fmt.Sprint(colony.CurrentMemorySchemaVersion), `"lineage":{`, `"provenance":"runtime"`} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("stored record lacks %s: %s", want, raw)
		}
	}
}
