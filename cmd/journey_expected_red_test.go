package cmd

// 207-03-PLAN.md Task 2 (UED-09, owner ruling D-01): the sixth blocker's
// check runs for real against the runtime's own status guidance code, is
// red today, is recorded as red with what closes it, and cannot become a
// silent pass, a hidden new gap, or a claim of six.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// journeyExpectedRedCeiling is the recorded maximum set of standing
// expected-red case ids -- the mirror image of evalGateSentinelFloor (a
// minimum that may only grow). This ceiling may only shrink: Phase 208
// landing the sixth blocker's fix removes this file's one entry and
// promotes the case to an ordinary check, never adds a second one silently.
// Raising this ceiling (adding an id) requires the same written-reason
// discipline every other ratchet in this codebase requires; TestExpectedRedRegisterOnlyShrinks
// is what enforces it.
var journeyExpectedRedCeiling = []string{
	"status-card-advises-a-menu-command-that-does-not-exist",
}

// seedUnacknowledgedMiddenFailure writes a single unacknowledged midden
// entry to s, the same shape middenGuidedAction and the unacknowledged-
// midden status warning both require to fire for real.
func seedUnacknowledgedMiddenFailure(t *testing.T, s interface{ SaveJSON(string, interface{}) error }) {
	t.Helper()
	ack := false
	mf := colony.MiddenFile{
		Version: "1.0.0",
		Entries: []colony.MiddenEntry{
			{
				ID:           "unack-1",
				Timestamp:    "2026-09-21T00:00:00Z",
				Category:     "build",
				Message:      "a genuine unacknowledged failure",
				Source:       "test",
				Acknowledged: &ack,
			},
		},
	}
	if err := s.SaveJSON("midden.json", mf); err != nil {
		t.Fatalf("seed midden.json: %v", err)
	}
}

// TestSixthBlockerCheckIsStillRed proves the sixth 2026-09-21 blocker is
// genuinely red against the real repository today: the status card's own
// guidance code advises `aether midden-review`, and this checkout has no
// menu wrapper for it.
//
// If this test starts failing because the result comes back empty, that
// means Phase 208 landed its fix and the gap has closed. The correct
// response is to delete the single case from
// cmd/testdata/journey/expected-red.json and turn this into an ordinary
// green assertion -- never to loosen, skip, or delete this test to make it
// pass.
func TestSixthBlockerCheckIsStillRed(t *testing.T) {
	t.Setenv("AETHER_HUB_DIR", t.TempDir())
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)

	seedUnacknowledgedMiddenFailure(t, s)

	repoRoot := findTestModuleRoot(t)
	missing, err := statusGuidanceCommandsWithoutMenuWrapper(s, tmpDir, repoRoot)
	if err != nil {
		t.Fatalf("statusGuidanceCommandsWithoutMenuWrapper: %v", err)
	}
	if len(missing) == 0 {
		t.Fatal("the sixth blocker's gap has closed (statusGuidanceCommandsWithoutMenuWrapper reported no gap) -- " +
			"this means Phase 208 landed its fix. Remove the single case from " +
			"cmd/testdata/journey/expected-red.json and turn this test into an ordinary green " +
			"assertion; do not loosen or skip it.")
	}
	if len(missing) != 1 || missing[0] != "midden-review" {
		t.Fatalf("missing wrapper verbs = %v, want exactly [midden-review]", missing)
	}

	file, err := loadJourneyExpectedRed()
	if err != nil {
		t.Fatalf("load expected-red register: %v", err)
	}
	if len(file.Cases) != 1 {
		t.Fatalf("expected exactly one registered case, got %d", len(file.Cases))
	}
	if !strings.Contains(file.Cases[0].Detail, "midden-review") {
		t.Fatalf("registered case detail = %q, want it to name midden-review", file.Cases[0].Detail)
	}
}

// TestNoUnregisteredStatusGuidanceGap is the check that catches a NEW gap
// the day it appears: any command the status card advises that has no menu
// wrapper AND no case in the committed register fails, naming the verb.
func TestNoUnregisteredStatusGuidanceGap(t *testing.T) {
	t.Setenv("AETHER_HUB_DIR", t.TempDir())
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)

	seedUnacknowledgedMiddenFailure(t, s)

	repoRoot := findTestModuleRoot(t)
	missing, err := statusGuidanceCommandsWithoutMenuWrapper(s, tmpDir, repoRoot)
	if err != nil {
		t.Fatalf("statusGuidanceCommandsWithoutMenuWrapper: %v", err)
	}

	file, err := loadJourneyExpectedRed()
	if err != nil {
		t.Fatalf("load expected-red register: %v", err)
	}
	registered := journeyExpectedRedRegisteredVerbs(file)

	for _, verb := range missing {
		if !registered[verb] {
			t.Errorf("status guidance advises %q with no menu wrapper, and no case in "+
				"cmd/testdata/journey/expected-red.json names it -- either add the wrapper, "+
				"or record a case naming what closes it", verb)
		}
	}
}

// journeyExpectedRedRegisteredVerbs derives the set of runtime verbs the
// committed register's own case details name, using the same
// backtick-command extraction statusGuidanceAdvisedCommands uses for status
// warnings -- never a re-typed verb list.
func journeyExpectedRedRegisteredVerbs(file journeyExpectedRedFile) map[string]bool {
	registered := map[string]bool{}
	for _, c := range file.Cases {
		for _, match := range statusAdvisedCommandRe.FindAllStringSubmatch(c.Detail, -1) {
			if verb, ok := statusAdvisedCommandVerb(match[1]); ok {
				registered[verb] = true
			}
		}
	}
	return registered
}

// TestExpectedRedRegisterOnlyShrinks is the ratchet against the recorded
// ceiling: every case id in the committed register must already be present
// in journeyExpectedRedCeiling, naming any that is not -- the register may
// never gain a case. Mirrors TestSentinelListOnlyGrows's discipline, in the
// opposite direction.
func TestExpectedRedRegisterOnlyShrinks(t *testing.T) {
	file, err := loadJourneyExpectedRed()
	if err != nil {
		t.Fatalf("load expected-red register: %v", err)
	}
	allowed := make(map[string]bool, len(journeyExpectedRedCeiling))
	for _, id := range journeyExpectedRedCeiling {
		allowed[id] = true
	}
	for _, c := range file.Cases {
		if !allowed[c.ID] {
			t.Errorf("case %q is not in the recorded expected-red ceiling -- the register may "+
				"only shrink toward zero as Phase 208 lands fixes; adding a case requires the "+
				"same written-reason discipline as raising journeyExpectedRedCeiling", c.ID)
		}
	}
}

// TestPhaseProvesFiveFixesNotSix asserts the register holds exactly one
// case and that its closed_by names Phase 208, so nothing in this phase can
// be read as proving six blockers rather than five.
func TestPhaseProvesFiveFixesNotSix(t *testing.T) {
	file, err := loadJourneyExpectedRed()
	if err != nil {
		t.Fatalf("load expected-red register: %v", err)
	}
	if len(file.Cases) != 1 {
		t.Fatalf("expected exactly one standing expected-red case (five blockers proven, one "+
			"still open), got %d", len(file.Cases))
	}
	if !strings.Contains(file.Cases[0].ClosedBy, "Phase 208") {
		t.Fatalf("case %q closed_by = %q, want it to name Phase 208", file.Cases[0].ID, file.Cases[0].ClosedBy)
	}
}

// TestStatusGuidanceGapCheckHandlesAnEmptyStore: with a store carrying no
// failures, no flags and no research, the check returns no gaps and no
// error.
func TestStatusGuidanceGapCheckHandlesAnEmptyStore(t *testing.T) {
	t.Setenv("AETHER_HUB_DIR", t.TempDir())
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)

	repoRoot := findTestModuleRoot(t)
	missing, err := statusGuidanceCommandsWithoutMenuWrapper(s, tmpDir, repoRoot)
	if err != nil {
		t.Fatalf("statusGuidanceCommandsWithoutMenuWrapper: %v", err)
	}
	if len(missing) != 0 {
		t.Fatalf("missing = %v, want none for an empty store", missing)
	}
}

// TestStatusGuidanceGapsAreReportedSeparatelyAndInOrder proves two gaps are
// reported as two separate named rows, in the order the status card itself
// produces them -- without editing cmd/status.go. It seeds the store so a
// second guided action (an active flag) genuinely fires alongside the
// unacknowledged-failure guided action, then points the wrapper-existence
// lookup at an isolated temporary directory holding only a subset of the
// real wrapper files, using the repo-root parameter the checker exposes.
// The guidance code itself is never stubbed.
func TestStatusGuidanceGapsAreReportedSeparatelyAndInOrder(t *testing.T) {
	t.Setenv("AETHER_HUB_DIR", t.TempDir())
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)

	seedUnacknowledgedMiddenFailure(t, s)

	flags := colony.FlagsFile{
		Version: "1.0.0",
		Decisions: []colony.FlagEntry{
			{
				ID:          "flag-1",
				Type:        "issue",
				Description: "a genuine open issue",
				Source:      "test",
				CreatedAt:   "2026-09-21T00:00:00Z",
				Resolved:    false,
			},
		},
	}
	if err := s.SaveJSON("pending-decisions.json", flags); err != nil {
		t.Fatalf("seed pending-decisions.json: %v", err)
	}

	// An isolated fixture repo root carrying only a subset of the real
	// wrapper files: `swarm` has a wrapper, `flags` and `midden-review` do
	// not. This produces the second gap entirely through what wrapper files
	// exist on disk -- the guidance code in cmd/status.go is untouched.
	isolatedRoot := t.TempDir()
	wrapperDir := filepath.Join(isolatedRoot, ".claude", "commands", "ant")
	if err := os.MkdirAll(wrapperDir, 0755); err != nil {
		t.Fatalf("mkdir isolated wrapper dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wrapperDir, "swarm.md"), []byte("swarm wrapper\n"), 0644); err != nil {
		t.Fatalf("write isolated swarm wrapper: %v", err)
	}

	missing, err := statusGuidanceCommandsWithoutMenuWrapper(s, tmpDir, isolatedRoot)
	if err != nil {
		t.Fatalf("statusGuidanceCommandsWithoutMenuWrapper: %v", err)
	}
	want := []string{"flags", "midden-review"}
	if len(missing) != len(want) {
		t.Fatalf("missing = %v, want %v", missing, want)
	}
	for i, verb := range want {
		if missing[i] != verb {
			t.Fatalf("missing[%d] = %q, want %q (missing = %v)", i, missing[i], verb, missing)
		}
	}
}

// TestStatusGuidanceGapCheckDoesNotMutate proves running the check twice
// against the same store leaves every file under the store byte-identical,
// and returns the same result both times. Reuses the shared snapshotDirFiles
// helper (cmd/memory_details_render_test.go), which already excludes
// .aether/locks/ -- storage.FileLocker's own first-touch lock-file creation
// is concurrency-safety bookkeeping, not a mutation of colony content.
func TestStatusGuidanceGapCheckDoesNotMutate(t *testing.T) {
	t.Setenv("AETHER_HUB_DIR", t.TempDir())
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)

	seedUnacknowledgedMiddenFailure(t, s)

	before := snapshotDirFiles(t, tmpDir)

	repoRoot := findTestModuleRoot(t)
	first, err := statusGuidanceCommandsWithoutMenuWrapper(s, tmpDir, repoRoot)
	if err != nil {
		t.Fatalf("first run: %v", err)
	}
	second, err := statusGuidanceCommandsWithoutMenuWrapper(s, tmpDir, repoRoot)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}

	if len(first) != len(second) {
		t.Fatalf("first run = %v, second run = %v -- results differ across runs", first, second)
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("first run = %v, second run = %v -- results differ across runs", first, second)
		}
	}

	after := snapshotDirFiles(t, tmpDir)
	if len(before) != len(after) {
		t.Fatalf("store file count changed: before=%d after=%d", len(before), len(after))
	}
	for path, content := range before {
		afterContent, ok := after[path]
		if !ok || string(afterContent) != string(content) {
			t.Fatalf("store file %s changed across two runs of the check", path)
		}
	}
}
