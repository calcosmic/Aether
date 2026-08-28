package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Phase 196, COST-05 — the three preserved branches.
//
// Three branches were sitting in this repository holding half-finished work on
// measuring what a run costs. The requirement is not "explain what you would
// do with them"; it is that each one is reviewed, its reason written down, and
// the branch actually gone. CLAUDE.md's Definition of Done makes that the whole
// point: a requirement is satisfied only when a command exists that FAILS when
// it is unmet, and this is that command.
//
// It fails if the record loses a branch, loses a commit, or loses a reason —
// and it fails if any of the three branch names is still in the repository, so
// the record cannot drift away from the truth in either direction.

// disposedBranches is the three branches and the tip commit each one pointed at
// when it was deleted. The commits are written here as literals so that losing
// one from the record is a test failure rather than a silent gap: with the
// commit, the work is recoverable forever; without it, deleting the name really
// did throw something away.
var disposedBranches = map[string]string{
	"worktree-agent-a47f78913caf6fde0": "be1e160b09109c2af56f5df6c8a1adee5b002a5a",
	"worktree-agent-aa57076cd698da76f": "ecd98b5cc81d0c51b82c04139df14d90e38e4619",
	"worktree-agent-a59fd3ee68644ea21": "1cbf3615100203be7d59d49ffc2cf224367cedaa",
}

// branchDispositionRecordPath is the written record, relative to the repo root.
const branchDispositionRecordPath = ".planning/phases/196-see-what-it-cost/196-BRANCH-DISPOSITION.md"

// repoRootForDispositionTest walks up from this source file to the repository
// root, so the test does not depend on the working directory.
func repoRootForDispositionTest(t *testing.T) string {
	t.Helper()
	dir := filepath.Dir(goldenTestdataDir()) // .../cmd
	return filepath.Dir(dir)
}

func TestBranchDispositionRecordsAllThreeBranches(t *testing.T) {
	root := repoRootForDispositionTest(t)
	raw, err := os.ReadFile(filepath.Join(root, branchDispositionRecordPath))
	if err != nil {
		t.Fatalf("the branch disposition record is missing (%v); "+
			"three branches cannot be deleted with their reasons recorded if there is no record", err)
	}
	record := string(raw)

	// Sections are the per-branch write-ups. The record's summary table names
	// every branch too, so a check on the whole file would pass on a record
	// that had lost every explanation and kept only the table.
	sections := map[string]string{}
	for _, block := range strings.Split(record, "\n## ") {
		for branch := range disposedBranches {
			if strings.Contains(strings.SplitN(block, "\n", 2)[0], branch) {
				sections[branch] = block
			}
		}
	}

	for branch, commit := range disposedBranches {
		if !strings.Contains(record, branch) {
			t.Errorf("branch %s is not named in the disposition record at all", branch)
			continue
		}
		if !strings.Contains(record, commit) {
			t.Errorf("branch %s's tip commit %s is not recorded; without it the deleted work "+
				"is no longer reachable by any name", branch, commit)
		}

		section := sections[branch]
		if section == "" {
			t.Errorf("branch %s has no section of its own in the record — it is named in the "+
				"summary table and nowhere else, so no reason was written for it", branch)
			continue
		}
		if !strings.Contains(section, commit) {
			t.Errorf("branch %s's own section does not carry its tip commit %s", branch, commit)
		}

		reason := branchDispositionReason(section)
		if len(reason) < 300 {
			t.Errorf("branch %s's written reason is %d characters — that is not a reason, it is a label:\n%s",
				branch, len(reason), reason)
		}
		if !branchDispositionStatesAVerdict(section) {
			t.Errorf("branch %s's section never says whether its content was salvaged or discarded", branch)
		}
	}

	if len(sections) != len(disposedBranches) {
		t.Errorf("the record carries %d per-branch sections, want %d", len(sections), len(disposedBranches))
	}
}

// branchDispositionReason is the prose in a section: everything that is not a
// heading, a table row, or a metadata line. A section padded out with table
// rows has no reason in it.
func branchDispositionReason(section string) string {
	var prose strings.Builder
	for _, line := range strings.Split(section, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "":
		case strings.HasPrefix(trimmed, "#"):
		case strings.HasPrefix(trimmed, "|"):
		case strings.HasPrefix(trimmed, "**Tip commit:**"):
		case strings.HasPrefix(trimmed, "**Disposition:**"):
		default:
			prose.WriteString(trimmed)
			prose.WriteString(" ")
		}
	}
	return strings.TrimSpace(prose.String())
}

func branchDispositionStatesAVerdict(section string) bool {
	lowered := strings.ToLower(section)
	return strings.Contains(lowered, "salvaged") || strings.Contains(lowered, "discarded")
}

// TestDisposedBranchesAreGoneFromTheRepository is the half of COST-05 that a
// document cannot satisfy. The criterion is that the branches are GONE, not
// that the reasons for removing them are well written.
func TestDisposedBranchesAreGoneFromTheRepository(t *testing.T) {
	root := repoRootForDispositionTest(t)
	if err := exec.Command("git", "-C", root, "rev-parse", "--git-dir").Run(); err != nil {
		t.Skipf("%s is not a git checkout, so there are no branches to assert about", root)
	}

	args := []string{"-C", root, "branch", "--list"}
	for branch := range disposedBranches {
		args = append(args, branch)
	}
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		t.Fatalf("could not list branches: %v", err)
	}
	if listed := strings.TrimSpace(string(out)); listed != "" {
		t.Errorf("these branches were recorded as deleted but still exist:\n%s", listed)
	}

	// The record names the commits; those must still resolve, or the deletion
	// really did throw the work away rather than merely retiring its name.
	hexCommit := regexp.MustCompile(`^[0-9a-f]{40}$`)
	for branch, commit := range disposedBranches {
		if !hexCommit.MatchString(commit) {
			t.Errorf("branch %s's recorded commit %q is not a full commit id", branch, commit)
			continue
		}
		if err := exec.Command("git", "-C", root, "cat-file", "-e", commit+"^{commit}").Run(); err != nil {
			t.Errorf("branch %s's recorded commit %s no longer resolves, so the deleted work "+
				"is not recoverable after all: %v", branch, commit, err)
		}
	}
}
