package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// sourceProposalTestRepo builds a real, throwaway git repository with one
// commit on branch "main" -- never the working repository this test process
// itself lives in. Every test in this file operates inside one of these.
func sourceProposalTestRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test")
	runGit(t, root, "checkout", "-b", "main")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("initial\n"), 0644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "initial")
	return root
}

// chdirTemp changes the process working directory to dir for the duration
// of the test, restoring the original directory on cleanup.
func chdirTemp(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(orig)
	})
}

// sourceProposalCurrentBranch returns root's currently checked-out branch.
func sourceProposalCurrentBranch(t *testing.T, root string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", root, "rev-parse", "--abbrev-ref", "HEAD").CombinedOutput()
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v\n%s", err, out)
	}
	return strings.TrimSpace(string(out))
}

// sourceProposalBranchesWithPrefix lists every local branch in root carrying
// sourceProposalBranchPrefix.
func sourceProposalBranchesWithPrefix(t *testing.T, root string) []string {
	t.Helper()
	out, err := exec.Command("git", "-C", root, "branch", "--list", sourceProposalBranchPrefix+"*").CombinedOutput()
	if err != nil {
		t.Fatalf("list branches: %v\n%s", err, out)
	}
	var names []string
	for _, line := range strings.Split(string(out), "\n") {
		trimmed := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "* "))
		if trimmed != "" {
			names = append(names, trimmed)
		}
	}
	return names
}

// TestProposalCreatesABranchAndNothingElse asserts the branch exists, the
// original branch is checked out, and the working tree is as it was.
func TestProposalCreatesABranchAndNothingElse(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	root := sourceProposalTestRepo(t)
	chdirTemp(t, root)

	changes := sourceChangeSet{
		Files:   []sourceChangeSetFile{{Path: "improvement.txt", Content: "a proposed change\n"}},
		Message: "propose an improvement",
	}
	result, created, err := proposeSourceImprovement("candidate-alpha", []string{"evidence-1"}, changes)
	if err != nil {
		t.Fatalf("proposeSourceImprovement: %v", err)
	}
	if !created {
		t.Fatal("expected created=true on a fresh proposal")
	}
	if result.VerificationState != sourceProposalStateProposed {
		t.Fatalf("VerificationState = %q, want %q", result.VerificationState, sourceProposalStateProposed)
	}
	if !sourceProposalBranchExists(root, result.Branch) {
		t.Fatalf("branch %q was not created", result.Branch)
	}
	if got := sourceProposalCurrentBranch(t, root); got != "main" {
		t.Fatalf("current branch = %q, want main (original branch restored)", got)
	}
	dirty, err := sourceProposalWorkingTreeIsDirty(root)
	if err != nil {
		t.Fatalf("check dirty: %v", err)
	}
	if dirty {
		t.Fatal("working tree is dirty after a successful proposal, want clean")
	}
	if _, err := os.Stat(filepath.Join(root, "improvement.txt")); err == nil {
		t.Fatal("improvement.txt exists on the original branch's working tree -- the change leaked off its own branch")
	}
}

// TestDirtyTreeProposalIsRefusedByName asserts a proposal on a dirty
// working tree is refused by name and creates nothing.
func TestDirtyTreeProposalIsRefusedByName(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	root := sourceProposalTestRepo(t)
	chdirTemp(t, root)

	if err := os.WriteFile(filepath.Join(root, "uncommitted.txt"), []byte("wip\n"), 0644); err != nil {
		t.Fatalf("write uncommitted file: %v", err)
	}

	before := sourceProposalBranchesWithPrefix(t, root)

	changes := sourceChangeSet{
		Files:   []sourceChangeSetFile{{Path: "improvement.txt", Content: "x\n"}},
		Message: "propose an improvement",
	}
	_, created, err := proposeSourceImprovement("candidate-dirty", nil, changes)
	if err == nil {
		t.Fatal("expected an error on a dirty working tree, got nil")
	}
	if !strings.Contains(err.Error(), "dirty") {
		t.Fatalf("error %q does not name the dirty working tree", err.Error())
	}
	if created {
		t.Fatal("expected created=false on a refused dirty-tree proposal")
	}

	after := sourceProposalBranchesWithPrefix(t, root)
	if len(after) != len(before) {
		t.Fatalf("branch count changed on a refused proposal: before=%v after=%v", before, after)
	}
}

// TestBranchNameCollisionIsRefused asserts a proposal whose branch name
// collides with an existing branch is refused by name rather than
// overwriting it.
func TestBranchNameCollisionIsRefused(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	root := sourceProposalTestRepo(t)
	chdirTemp(t, root)

	collidingBranch := sourceProposalBranchName("candidate-collide")
	runGit(t, root, "branch", collidingBranch)

	changes := sourceChangeSet{
		Files:   []sourceChangeSetFile{{Path: "improvement.txt", Content: "x\n"}},
		Message: "propose an improvement",
	}
	_, created, err := proposeSourceImprovement("candidate-collide", nil, changes)
	if err == nil {
		t.Fatal("expected an error on a branch-name collision, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("error %q does not name the collision", err.Error())
	}
	if created {
		t.Fatal("expected created=false on a refused collision")
	}
	if got := sourceProposalCurrentBranch(t, root); got != "main" {
		t.Fatalf("current branch = %q, want main (untouched by the refused proposal)", got)
	}
}

// TestProposalReplayCreatesNoSecondBranch asserts proposing the same
// improvement twice returns the first proposal and creates no second
// branch.
func TestProposalReplayCreatesNoSecondBranch(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	root := sourceProposalTestRepo(t)
	chdirTemp(t, root)

	changes := sourceChangeSet{
		Files:   []sourceChangeSetFile{{Path: "improvement.txt", Content: "same content\n"}},
		Message: "propose an improvement",
	}
	first, created1, err := proposeSourceImprovement("candidate-replay", []string{"evidence-1"}, changes)
	if err != nil {
		t.Fatalf("first proposeSourceImprovement: %v", err)
	}
	if !created1 {
		t.Fatal("expected created=true on the first call")
	}

	second, created2, err := proposeSourceImprovement("candidate-replay", []string{"evidence-1"}, changes)
	if err != nil {
		t.Fatalf("second proposeSourceImprovement: %v", err)
	}
	if created2 {
		t.Fatal("expected created=false on a replayed proposal")
	}
	if second.ProposalID != first.ProposalID {
		t.Fatalf("replay returned a different proposal: %q vs %q", second.ProposalID, first.ProposalID)
	}

	branches := sourceProposalBranchesWithPrefix(t, root)
	if len(branches) != 1 {
		t.Fatalf("expected exactly 1 proposal branch after a replay, got %d: %v", len(branches), branches)
	}
}

// TestUnverifiedProposalIsReportedNotReady asserts the readiness report
// names the outstanding verification for a proposal that has not been
// independently verified.
func TestUnverifiedProposalIsReportedNotReady(t *testing.T) {
	if len(sourceProposalVerificationStateVocabulary) != len(sourceProposalVerificationStateNames()) {
		t.Fatalf("verification-state const block (%d) and names slice (%d) have different lengths",
			len(sourceProposalVerificationStateVocabulary), len(sourceProposalVerificationStateNames()))
	}

	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	root := sourceProposalTestRepo(t)
	chdirTemp(t, root)

	changes := sourceChangeSet{
		Files:   []sourceChangeSetFile{{Path: "improvement.txt", Content: "x\n"}},
		Message: "propose an improvement",
	}
	proposal, _, err := proposeSourceImprovement("candidate-not-ready", nil, changes)
	if err != nil {
		t.Fatalf("proposeSourceImprovement: %v", err)
	}

	report := renderSourceProposalReadiness(proposal)
	if !strings.Contains(report, "not ready") {
		t.Fatalf("report %q does not say the proposal is not ready", report)
	}
	if !strings.Contains(report, string(sourceProposalStateProposed)) {
		t.Fatalf("report %q does not name the outstanding verification state %q", report, sourceProposalStateProposed)
	}
}

// TestProposalCannotVerifyItself asserts a proposal is never marked
// verified by the same identity that created it, and that recordIndependent
// Verification is the only function ever setting the verified state.
func TestProposalCannotVerifyItself(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	root := sourceProposalTestRepo(t)
	chdirTemp(t, root)

	changes := sourceChangeSet{
		Files:   []sourceChangeSetFile{{Path: "improvement.txt", Content: "x\n"}},
		Message: "propose an improvement",
	}
	proposal, _, err := proposeSourceImprovement("candidate-self-verify", nil, changes)
	if err != nil {
		t.Fatalf("proposeSourceImprovement: %v", err)
	}

	if err := recordIndependentVerification(proposal.ProposalID, "candidate-self-verify", "result-1"); err == nil {
		t.Fatal("expected an error when the proposal's own creator verifies it")
	} else if !strings.Contains(err.Error(), "cannot verify itself") {
		t.Fatalf("error %q does not name the self-verification refusal", err.Error())
	}

	unchanged, ok, err := readSourceProposal(proposal.ProposalID)
	if err != nil || !ok {
		t.Fatalf("read back proposal: ok=%v err=%v", ok, err)
	}
	if unchanged.VerificationState != sourceProposalStateProposed {
		t.Fatalf("VerificationState = %q after a refused self-verification, want unchanged %q",
			unchanged.VerificationState, sourceProposalStateProposed)
	}

	if err := recordIndependentVerification(proposal.ProposalID, "an-independent-verifier", "result-1"); err != nil {
		t.Fatalf("recordIndependentVerification from a different verifier: %v", err)
	}
	verified, ok, err := readSourceProposal(proposal.ProposalID)
	if err != nil || !ok {
		t.Fatalf("read back verified proposal: ok=%v err=%v", ok, err)
	}
	if verified.VerificationState != sourceProposalStateVerified {
		t.Fatalf("VerificationState = %q, want %q", verified.VerificationState, sourceProposalStateVerified)
	}
	if verified.IndependentVerificationID != "result-1" {
		t.Fatalf("IndependentVerificationID = %q, want %q", verified.IndependentVerificationID, "result-1")
	}
}

// TestProposalRecordNamesItsCandidateAndEvidence asserts the proposal
// record carries the candidate it came from and the evidence for it.
func TestProposalRecordNamesItsCandidateAndEvidence(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	root := sourceProposalTestRepo(t)
	chdirTemp(t, root)

	changes := sourceChangeSet{
		Files:   []sourceChangeSetFile{{Path: "improvement.txt", Content: "x\n"}},
		Message: "propose an improvement",
	}
	proposal, _, err := proposeSourceImprovement("candidate-naming", []string{"evidence-a", "evidence-b"}, changes)
	if err != nil {
		t.Fatalf("proposeSourceImprovement: %v", err)
	}
	if proposal.CandidateID != "candidate-naming" {
		t.Fatalf("CandidateID = %q, want %q", proposal.CandidateID, "candidate-naming")
	}
	if len(proposal.EvidenceIDs) != 2 || proposal.EvidenceIDs[0] != "evidence-a" || proposal.EvidenceIDs[1] != "evidence-b" {
		t.Fatalf("EvidenceIDs = %v, want [evidence-a evidence-b]", proposal.EvidenceIDs)
	}
	if proposal.Branch == "" {
		t.Fatal("Branch is empty")
	}
	if proposal.BaseCommit == "" {
		t.Fatal("BaseCommit is empty")
	}
}
