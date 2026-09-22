package cmd

// 207-05-PLAN.md Task 3 (UED-09, owner ruling D-01): the committed
// fix-revert table (cmd/journey_fix_reverts.go, cmd/testdata/journey/fix-reverts.json)
// is kept honest by fast, free tests -- it cannot rot against a changed
// source file, cannot lose an entry, cannot gain a sixth, and the harness
// that applies it (scripts/prove-journey-catches-the-2026-09-21-fixes.sh)
// cannot start operating on the owner's own checkout. No chat, no money.

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// commitExistsInRepo shells out to the repository's own git to check a
// commit is real -- never a hard-coded path, the same repoRoot every other
// check in this file resolves.
func commitExistsInRepo(t *testing.T, repoRoot, commit string) bool {
	t.Helper()
	cmd := exec.Command("git", "-C", repoRoot, "cat-file", "-e", commit+"^{commit}")
	return cmd.Run() == nil
}

// TestFixRevertTableNamesFiveLandedFixes proves every declared entry in the
// committed fix-revert table resolves against the real repository: the
// commit exists, the file exists, the symbol appears in that file, and the
// journey step is one the journey actually declares.
func TestFixRevertTableNamesFiveLandedFixes(t *testing.T) {
	file, err := loadJourneyFixReverts()
	if err != nil {
		t.Fatalf("load fix-revert table: %v", err)
	}
	if len(file.Reverts) != 5 {
		t.Fatalf("fix-revert table has %d entries, want exactly 5 -- five of the six 2026-09-21 "+
			"blockers have a landed fix (owner ruling D-01); the sixth stays the standing "+
			"expected-red case", len(file.Reverts))
	}

	repoRoot := findTestModuleRoot(t)
	for _, r := range file.Reverts {
		if !commitExistsInRepo(t, repoRoot, r.Commit) {
			t.Errorf("entry %q names commit %q, which does not exist in this repository", r.ID, r.Commit)
		}

		fullPath := filepath.Join(repoRoot, r.File)
		data, readErr := os.ReadFile(fullPath)
		if readErr != nil {
			t.Errorf("entry %q names file %q, which could not be read: %v", r.ID, r.File, readErr)
			continue
		}

		if !strings.Contains(string(data), r.Symbol) {
			t.Errorf("entry %q names symbol %q, which does not appear anywhere in %q", r.ID, r.Symbol, r.File)
		}

		if !journeyStepDeclared(journeyStep(r.JourneyStep)) {
			t.Errorf("entry %q names journey step %q, which is not one of the declared steps %v",
				r.ID, r.JourneyStep, journeyStepNames())
		}
	}
}

// TestFixRevertMutationsStillApply proves each entry's mutation.find is a
// real, currently-unique text swap in its own file at current HEAD, and
// that replace genuinely differs from find. A rotted find (edited away by a
// later commit, or no longer unique) fails here by name, naming the entry
// id, the file, and the count of occurrences actually found -- before this
// project's own executor ever spends money running the live proof against
// a table that can no longer apply.
func TestFixRevertMutationsStillApply(t *testing.T) {
	file, err := loadJourneyFixReverts()
	if err != nil {
		t.Fatalf("load fix-revert table: %v", err)
	}

	repoRoot := findTestModuleRoot(t)
	for _, r := range file.Reverts {
		fullPath := filepath.Join(repoRoot, r.File)
		data, readErr := os.ReadFile(fullPath)
		if readErr != nil {
			t.Errorf("entry %q: could not read %q: %v", r.ID, r.File, readErr)
			continue
		}
		content := string(data)
		count := strings.Count(content, r.Mutation.Find)
		if count != 1 {
			t.Errorf("entry %q: mutation.find occurs %d time(s) in %q, want exactly 1 -- "+
				"the fix-revert table has rotted against the current source and needs a fresh, "+
				"minimal swap derived from the file as it stands today", r.ID, count, r.File)
		}
		if r.Mutation.Find == r.Mutation.Replace {
			t.Errorf("entry %q: mutation.find and mutation.replace are identical -- this swap "+
				"would change nothing", r.ID)
		}
	}
}

// writeJourneyFixRevertsFixture writes file as an isolated fix-revert table
// under a fresh temp root (mirroring the committed layout exactly), sets
// journeyFixRevertsRepoRootOverride to point loadJourneyFixReverts at it for
// the duration of the test, and restores the override on cleanup. The
// committed table itself is never edited to prove a refusal.
func writeJourneyFixRevertsFixture(t *testing.T, file journeyFixRevertFile) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "cmd", "testdata", "journey")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir fixture dir: %v", err)
	}
	data, err := json.Marshal(file)
	if err != nil {
		t.Fatalf("marshal fixture table: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "fix-reverts.json"), data, 0o644); err != nil {
		t.Fatalf("write fixture table: %v", err)
	}

	prev := journeyFixRevertsRepoRootOverride
	journeyFixRevertsRepoRootOverride = root
	t.Cleanup(func() { journeyFixRevertsRepoRootOverride = prev })
}

// validateJourneyFixRevertFile enforces the two structural invariants
// TestFixRevertEntriesAreDistinct and TestFixRevertTableIsNeverEmpty prove:
// the table must carry at least one entry, and no two entries may name the
// same (file, symbol) pair -- two fixes touching one place stay separately
// proven, never silently merged.
func validateJourneyFixRevertFile(file journeyFixRevertFile) error {
	if len(file.Reverts) == 0 {
		return errEmptyFixRevertTable
	}
	seen := make(map[string]string, len(file.Reverts))
	for _, r := range file.Reverts {
		key := r.File + "::" + r.Symbol
		if otherID, dup := seen[key]; dup {
			return &duplicateFixRevertEntryError{firstID: otherID, secondID: r.ID, file: r.File, symbol: r.Symbol}
		}
		seen[key] = r.ID
	}
	return nil
}

type duplicateFixRevertEntryError struct {
	firstID, secondID, file, symbol string
}

func (e *duplicateFixRevertEntryError) Error() string {
	return "entries " + e.firstID + " and " + e.secondID + " both name file " + e.file +
		" and symbol " + e.symbol + " -- two fixes touching the same place must stay separate rows, not merged"
}

var errEmptyFixRevertTable = fixRevertTableEmptyError{}

type fixRevertTableEmptyError struct{}

func (fixRevertTableEmptyError) Error() string {
	return "the fix-revert table is empty -- a proof that proves nothing is not a pass"
}

// TestFixRevertEntriesAreDistinct proves two entries naming the same file
// and the same symbol are refused by name, while two entries that merely
// share a journey step are allowed and stay two separate rows.
func TestFixRevertEntriesAreDistinct(t *testing.T) {
	t.Run("same file and symbol is refused", func(t *testing.T) {
		writeJourneyFixRevertsFixture(t, journeyFixRevertFile{
			SchemaVersion: journeyFixRevertsSchemaVersion,
			Note:          "fixture",
			Reverts: []journeyFixRevert{
				{ID: "one", File: "cmd/a.go", Symbol: "sameSymbol", JourneyStep: "pause",
					Mutation: journeyFixRevertMutation{Find: "x", Replace: "y"}},
				{ID: "two", File: "cmd/a.go", Symbol: "sameSymbol", JourneyStep: "archive",
					Mutation: journeyFixRevertMutation{Find: "p", Replace: "q"}},
			},
		})
		file, err := loadJourneyFixReverts()
		if err != nil {
			t.Fatalf("load fixture table: %v", err)
		}
		err = validateJourneyFixRevertFile(file)
		if err == nil {
			t.Fatal("expected two entries naming the same file and symbol to be refused")
		}
		if !strings.Contains(err.Error(), "one") || !strings.Contains(err.Error(), "two") {
			t.Fatalf("refusal does not name both entry ids: %v", err)
		}
	})

	t.Run("shared journey step alone is allowed", func(t *testing.T) {
		writeJourneyFixRevertsFixture(t, journeyFixRevertFile{
			SchemaVersion: journeyFixRevertsSchemaVersion,
			Note:          "fixture",
			Reverts: []journeyFixRevert{
				{ID: "one", File: "cmd/a.go", Symbol: "symbolA", JourneyStep: "pause",
					Mutation: journeyFixRevertMutation{Find: "x", Replace: "y"}},
				{ID: "two", File: "cmd/b.go", Symbol: "symbolB", JourneyStep: "pause",
					Mutation: journeyFixRevertMutation{Find: "p", Replace: "q"}},
			},
		})
		file, err := loadJourneyFixReverts()
		if err != nil {
			t.Fatalf("load fixture table: %v", err)
		}
		if err := validateJourneyFixRevertFile(file); err != nil {
			t.Fatalf("two entries sharing only a journey step must be allowed: %v", err)
		}
		if len(file.Reverts) != 2 {
			t.Fatalf("expected both entries to remain as separate rows, got %d", len(file.Reverts))
		}
	})
}

// TestFixRevertTableIsNeverEmpty proves an empty table is refused with a
// message naming exactly why: a proof that proves nothing is not a pass.
func TestFixRevertTableIsNeverEmpty(t *testing.T) {
	writeJourneyFixRevertsFixture(t, journeyFixRevertFile{
		SchemaVersion: journeyFixRevertsSchemaVersion,
		Note:          "fixture",
		Reverts:       []journeyFixRevert{},
	})
	file, err := loadJourneyFixReverts()
	if err != nil {
		t.Fatalf("load fixture table: %v", err)
	}
	err = validateJourneyFixRevertFile(file)
	if err == nil {
		t.Fatal("expected an empty fix-revert table to be refused")
	}
	if !strings.Contains(err.Error(), "proves nothing") || !strings.Contains(err.Error(), "not a pass") {
		t.Fatalf("refusal does not name why an empty table is refused: %v", err)
	}
}

// journeyFixRevertResultsInDeclaredOrder returns the ids present in results,
// ordered by the table's own declared file order -- never map-iteration
// order, which Go deliberately randomizes.
func journeyFixRevertResultsInDeclaredOrder(file journeyFixRevertFile, results map[string]bool) []string {
	ordered := make([]string, 0, len(file.Reverts))
	for _, r := range file.Reverts {
		if _, ok := results[r.ID]; ok {
			ordered = append(ordered, r.ID)
		}
	}
	return ordered
}

// TestFixRevertResultsFollowTheDeclaredOrder proves a synthetic set of
// results is reported in the table's declared file order, not the
// randomized order Go's own map iteration would produce.
func TestFixRevertResultsFollowTheDeclaredOrder(t *testing.T) {
	file := journeyFixRevertFile{
		Reverts: []journeyFixRevert{
			{ID: "write-allowlist"},
			{ID: "archive-name-case"},
			{ID: "helper-fingerprint"},
			{ID: "superseded-specification"},
			{ID: "pause-follows-shortcuts"},
		},
	}
	// Deliberately built so map insertion order does not match declared
	// order, and run several times: Go randomizes map iteration order per
	// run, so a genuinely order-stable function must agree with itself
	// across repeated calls regardless.
	results := map[string]bool{
		"pause-follows-shortcuts":  true,
		"write-allowlist":          true,
		"superseded-specification": true,
	}
	want := []string{"write-allowlist", "superseded-specification", "pause-follows-shortcuts"}

	for i := 0; i < 5; i++ {
		got := journeyFixRevertResultsInDeclaredOrder(file, results)
		if len(got) != len(want) {
			t.Fatalf("run %d: got %v, want %v", i, got, want)
		}
		for j := range want {
			if got[j] != want[j] {
				t.Fatalf("run %d: got %v, want %v (declared file order)", i, got, want)
			}
		}
	}
}

// TestExpectedRedRegisterIsEmptyAndTheFiveRevertsStand asserts exactly five
// entries still stand in the fix-revert table and the expected-red
// register now holds zero cases, replacing
// TestPhaseProvesFiveFixesNotSix (cmd/journey_expected_red_test.go) and
// TestPhaseCountsFiveRevertsAndOneStandingRedCase. Phase 208 closed the
// sixth 2026-09-21 blocker with a real fix -- the menu wrapper for
// `aether midden-review` -- rather than by editing this register, so
// nothing in this phase can be read as proving six blockers by loosening
// the check; it is proven by the register being honestly empty.
func TestExpectedRedRegisterIsEmptyAndTheFiveRevertsStand(t *testing.T) {
	reverts, err := loadJourneyFixReverts()
	if err != nil {
		t.Fatalf("load fix-revert table: %v", err)
	}
	expectedRed, err := loadJourneyExpectedRed()
	if err != nil {
		t.Fatalf("load expected-red register: %v", err)
	}

	if len(reverts.Reverts) != 5 || len(expectedRed.Cases) != 0 {
		t.Fatalf("got %d fix-revert entries and %d expected-red case(s); want exactly 5 and 0 -- "+
			"per the owner's ruling (D-01, .planning/phases/207-messy-practice-project-gate/207-CONTEXT.md): "+
			"\"the journey proves it catches the five fixed blockers ... the check for the sixth is "+
			"built too and stays honestly red ... never a claim of six\" -- Phase 208 closed the "+
			"sixth with a real fix, so the register is now empty, not still standing",
			len(reverts.Reverts), len(expectedRed.Cases))
	}
}

// journeyFixRevertScriptNonCommentLines reads path and returns every line
// whose trimmed content does not start with "#" -- filtering out this
// script's own header comments (which describe, in prose, the very
// operations this test must prove are absent) before searching, so a
// sentence explaining what the script must never do cannot make this
// check fail on itself.
func journeyFixRevertScriptNonCommentLines(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var kept []string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

// TestFixRevertHarnessNeverTouchesTheOwnersCheckout proves the harness
// script uses git's own additional-working-copy mechanism (git worktree
// add) and never a working-copy-switching or content-destroying git
// operation (checkout, stash, reset, restore, clean) anywhere outside its
// own comments. WR-03 (207-REVIEW.md): "git clean" added -- `git clean -fd`
// would silently delete untracked files in the owner's own checkout and is
// at least as dangerous as the other four, so it belongs in the same
// forbidden list even though the script does not use it today.
func TestFixRevertHarnessNeverTouchesTheOwnersCheckout(t *testing.T) {
	repoRoot := findTestModuleRoot(t)
	scriptPath := filepath.Join(repoRoot, "scripts", "prove-journey-catches-the-2026-09-21-fixes.sh")
	code := journeyFixRevertScriptNonCommentLines(t, scriptPath)

	if !strings.Contains(code, "git worktree add") {
		t.Error("the harness script does not use git worktree add anywhere outside its own comments -- " +
			"the only permitted mechanism for applying a revert is an additional working copy")
	}
	for _, forbidden := range []string{"git checkout", "git stash", "git reset", "git restore", "git clean"} {
		if strings.Contains(code, forbidden) {
			t.Errorf("the harness script contains %q outside a comment -- this could switch, "+
				"shelve, or delete changes in the owner's own checkout, which is never permitted", forbidden)
		}
	}
}
