package cmd

// LEARN-08 (204-10-PLAN.md, Task 2/3): the propose-only boundary between
// this system suggesting a change to its own source and a human actually
// accepting one. proposeSourceImprovement opens one isolated branch,
// applies its change set, and records itself durably -- and does nothing
// else: no merge, no push, no publish, no deploy is reachable from here,
// proved structurally by cmd/source_proposal_test.go's
// TestSourceProposalCannotMergePublishOrDeploy rather than by a rule
// written down in a document. Marking a proposal verified is a wholly
// separate act (recordIndependentVerification), refused when the verifier
// is the same identity that created the proposal -- the same contribution
// cannot both propose and independently verify itself.
import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// sourceProposalPath is the store-relative path proposeSourceImprovement and
// recordIndependentVerification persist to.
const sourceProposalPath = "proposals/source.json"

// sourceProposalBranchPrefix names every branch a source proposal creates,
// so any branch carrying it is recognisable as this system's own proposal
// output rather than ordinary human work.
const sourceProposalBranchPrefix = "source-proposal/"

// sourceProposalRepeatedInterventionThreshold is the number of DISTINCT
// episodes a single declared episodeInterventionKind category must
// accumulate before the automatic trigger in cmd/improvement_pass.go
// proposes a source change for it -- "the same kind of thing keeps going
// wrong", the same reading cmd/rollback.go's canaryRegressionQuarantineThreshold
// already applies to a canary scope's regression count. Declared here,
// separately from that constant, because the two domains count a
// fundamentally different identity: a canary scope's regressions across
// candidate attempts, versus a preventable-intervention category's distinct
// episodes across the whole colony's history (204-16-PLAN.md, SC5d).
const sourceProposalRepeatedInterventionThreshold = 3

// sourceProposalVerificationState is the declared, closed vocabulary a
// proposal's verification state may hold.
type sourceProposalVerificationState string

const (
	sourceProposalStateProposed            sourceProposalVerificationState = "proposed"
	sourceProposalStateVerificationPending sourceProposalVerificationState = "verification_pending"
	sourceProposalStateVerified            sourceProposalVerificationState = "verified"
	sourceProposalStateRejected            sourceProposalVerificationState = "rejected"
)

// sourceProposalVerificationStateVocabulary is the declared, closed set of
// every verification state -- same completeness convention as
// episodeLedgerRecordKindVocabulary/recruitmentCreditOutcomeVocabulary: a
// state added to the const block above must also be added here.
var sourceProposalVerificationStateVocabulary = []sourceProposalVerificationState{
	sourceProposalStateProposed,
	sourceProposalStateVerificationPending,
	sourceProposalStateVerified,
	sourceProposalStateRejected,
}

func sourceProposalVerificationStateNames() []string {
	names := make([]string, 0, len(sourceProposalVerificationStateVocabulary))
	for _, s := range sourceProposalVerificationStateVocabulary {
		names = append(names, string(s))
	}
	return names
}

func sourceProposalVerificationStateDeclared(state sourceProposalVerificationState) bool {
	for _, s := range sourceProposalVerificationStateVocabulary {
		if s == state {
			return true
		}
	}
	return false
}

// sourceChangeSetFile is one file a proposed change writes, relative to the
// repository root.
type sourceChangeSetFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// sourceChangeSet is what a source proposal actually changes: a set of
// file writes and the commit message recorded on the proposal branch. A
// change set never carries a merge, push, publish or deploy instruction --
// there is no field here that could.
type sourceChangeSet struct {
	Files   []sourceChangeSetFile `json:"files"`
	Message string                `json:"message"`
}

// sourceProposal is the durable record of one proposed source improvement:
// the candidate it came from, the evidence for it, the branch it lives on,
// the commit it branched from, when it was created, its verification
// state, and -- only once verified -- the identifier of the independent
// verification that did it.
type sourceProposal struct {
	ProposalID                string                          `json:"proposal_id"`
	CandidateID               string                          `json:"candidate_id"`
	EvidenceIDs               []string                        `json:"evidence_ids,omitempty"`
	Branch                    string                          `json:"branch"`
	BaseCommit                string                          `json:"base_commit"`
	CreatedAt                 string                          `json:"created_at"`
	VerificationState         sourceProposalVerificationState `json:"verification_state"`
	IndependentVerificationID string                          `json:"independent_verification_id,omitempty"`
}

// sourceProposalFile is the on-disk container at sourceProposalPath.
type sourceProposalFile struct {
	Entries []sourceProposal `json:"entries"`
}

// sourceProposalID is the deterministic identity key a proposal is stored
// and replayed under: the same candidate, evidence and change set propose
// the same proposal, always -- proposing it twice returns the first
// proposal and creates no second branch.
func sourceProposalID(candidateID string, evidence []string, changes sourceChangeSet) string {
	sortedEvidence := append([]string{}, evidence...)
	sort.Strings(sortedEvidence)
	var sb strings.Builder
	sb.WriteString(candidateID)
	sb.WriteString("\x00")
	sb.WriteString(strings.Join(sortedEvidence, ","))
	sb.WriteString("\x00")
	sb.WriteString(changes.Message)
	for _, f := range changes.Files {
		sb.WriteString("\x00")
		sb.WriteString(f.Path)
		sb.WriteString("=")
		sb.WriteString(f.Content)
	}
	sum := sha256.Sum256([]byte(sb.String()))
	return fmt.Sprintf("proposal:%x", sum)
}

// sourceProposalBranchName derives the deterministic branch name a
// candidate's proposal lives on.
func sourceProposalBranchName(candidateID string) string {
	return sourceProposalBranchPrefix + sourceProposalSanitizeForBranch(candidateID)
}

// sourceProposalSanitizeForBranch replaces every character a git branch
// name cannot safely carry with a hyphen.
func sourceProposalSanitizeForBranch(raw string) string {
	var sb strings.Builder
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '/':
			sb.WriteRune(r)
		default:
			sb.WriteRune('-')
		}
	}
	sanitized := sb.String()
	if sanitized == "" {
		sanitized = "candidate"
	}
	return sanitized
}

// errSourceProposalNoChange is the internal replay sentinel
// writeSourceProposal returns from its own UpdateJSONAtomically mutate
// closure to abort the write on a replay.
var errSourceProposalNoChange = errors.New("source proposal already exists")

// writeSourceProposal is the writer proposeSourceImprovement uses to record
// a newly created proposal durably. A second call carrying the same
// ProposalID returns the first stored record and writes nothing.
func writeSourceProposal(proposal sourceProposal) (sourceProposal, bool, error) {
	if store == nil {
		return sourceProposal{}, false, fmt.Errorf("no store initialized")
	}
	var file sourceProposalFile
	var result sourceProposal
	updateErr := store.UpdateJSONAtomically(sourceProposalPath, &file, func() error {
		for _, existing := range file.Entries {
			if existing.ProposalID == proposal.ProposalID {
				result = existing
				return errSourceProposalNoChange
			}
		}
		file.Entries = append(file.Entries, proposal)
		result = proposal
		return nil
	})
	if updateErr != nil {
		if errors.Is(updateErr, errSourceProposalNoChange) {
			return result, false, nil
		}
		return sourceProposal{}, false, updateErr
	}
	return result, true, nil
}

// readSourceProposal returns the stored proposal named proposalID, if one
// exists.
func readSourceProposal(proposalID string) (sourceProposal, bool, error) {
	if store == nil {
		return sourceProposal{}, false, fmt.Errorf("no store initialized")
	}
	var file sourceProposalFile
	if err := store.LoadJSON(sourceProposalPath, &file); err != nil {
		return sourceProposal{}, false, nil
	}
	for _, p := range file.Entries {
		if p.ProposalID == proposalID {
			return p, true, nil
		}
	}
	return sourceProposal{}, false, nil
}

// sourceProposalWorkingTreeIsDirty runs `git status --porcelain` in root and
// reports whether the working tree carries any uncommitted change,
// following this repository's own dirty-tree safety convention
// (cmd/porter_cmd.go's checkGitStatus, cmd/session_flow_cmds.go's
// repositoryDirtyDigest).
func sourceProposalWorkingTreeIsDirty(root string) (bool, error) {
	out, err := gitOutputAt(root, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// sourceProposalBranchExists reports whether branch already exists in root,
// without treating "does not exist" as an error.
func sourceProposalBranchExists(root, branch string) bool {
	cmd := exec.Command("git", "-C", root, "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	return cmd.Run() == nil
}

// sourceProposalIsolatedWorktree creates a temporary git worktree for
// branch at baseCommit, in a directory OUTSIDE root entirely (os.MkdirTemp
// against the OS temp directory, never a path under root) -- so that
// creating and populating the proposal branch never requires switching
// root's own live checkout (CR-04, 204-REVIEW.md). Returns the worktree's
// absolute path and a cleanup function that removes the worktree checkout
// -- and, best-effort, prunes git's own stale worktree metadata -- but
// NEVER deletes branch itself: a source proposal's branch is the whole
// point of this function and must survive for a human to review, unlike a
// build worker's throwaway worktree branch (cmd/codex_build_worktree.go's
// removeGitWorktree, which deletes its branch once the worker's own
// changes have synced back -- a deliberately different lifecycle this
// function does not reuse).
func sourceProposalIsolatedWorktree(root, branch, baseCommit string) (string, func(), error) {
	parent, err := os.MkdirTemp("", "aether-source-proposal-")
	if err != nil {
		return "", nil, fmt.Errorf("create temporary worktree directory: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(parent) }

	// git worktree add refuses a target path that already exists (even
	// empty, on some git versions) -- join a not-yet-created subdirectory
	// so git creates the final directory itself, exactly as
	// allocateBuildWorktree (cmd/codex_build_worktree.go) already does for
	// the same reason.
	worktreePath := filepath.Join(parent, "wt")
	if _, err := gitOutputAt(root, "worktree", "add", "-b", branch, worktreePath, baseCommit); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("git worktree add: %w", err)
	}

	removeWorktree := func() {
		// Best-effort, in this order: `git worktree remove` first, so
		// git's own administrative metadata stays consistent; if it fails
		// for any reason (e.g. the directory was already partially
		// removed), os.RemoveAll below still reclaims the disk space, and
		// a later `git worktree prune` in root clears the stale entry.
		// The branch is never touched by either step.
		if _, err := gitOutputAt(root, "worktree", "remove", worktreePath, "--force"); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not cleanly remove source proposal worktree %q: %v\n", worktreePath, err)
		}
		cleanup()
	}
	return worktreePath, removeWorktree, nil
}

// sourceProposalApplyChangeSet writes every file in changes, stages them,
// and commits on whatever branch is currently checked out in root. It
// performs no other git operation -- no merge, no push, no fetch, no
// remote-touching command of any kind.
func sourceProposalApplyChangeSet(root string, changes sourceChangeSet) error {
	if len(changes.Files) == 0 {
		return fmt.Errorf("source proposal change set carries no files")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve proposal root: %w", err)
	}
	paths := make([]string, 0, len(changes.Files))
	for _, f := range changes.Files {
		relPath := strings.TrimSpace(f.Path)
		if relPath == "" {
			return fmt.Errorf("source proposal change set carries a file with an empty path")
		}
		if filepath.IsAbs(relPath) {
			return fmt.Errorf("source proposal change set carries an absolute path %q -- refused", relPath)
		}
		full := filepath.Join(absRoot, relPath)
		// filepath.Join already Cleans the result; it must still be inside
		// absRoot, otherwise a ".." segment escaped the proposal root -- the
		// same untrusted-input treatment CLAUDE.md requires for anything a
		// candidate or worker can supply (CR-01, 204-REVIEW.md).
		if full != absRoot && !strings.HasPrefix(full, absRoot+string(filepath.Separator)) {
			return fmt.Errorf("source proposal change set path %q escapes the proposal root -- refused", relPath)
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return fmt.Errorf("create directory for %s: %w", relPath, err)
		}
		if err := os.WriteFile(full, []byte(f.Content), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", relPath, err)
		}
		paths = append(paths, relPath)
	}
	addArgs := append([]string{"add"}, paths...)
	if _, err := gitOutputAt(root, addArgs...); err != nil {
		return fmt.Errorf("stage proposal changes: %w", err)
	}
	message := strings.TrimSpace(changes.Message)
	if message == "" {
		message = "source proposal"
	}
	if _, err := gitOutputAt(root, "commit", "-m", message); err != nil {
		return fmt.Errorf("commit proposal changes: %w", err)
	}
	return nil
}

// proposeSourceImprovement opens one isolated branch carrying changes,
// records a durable proposal, and does nothing else. It refuses a dirty
// working tree by name and creates nothing; refuses a branch-name collision
// by name rather than overwriting.
//
// CR-04 (204-REVIEW.md): the branch is created and the change set applied
// entirely inside a TEMPORARY, ISOLATED git worktree
// (sourceProposalIsolatedWorktree) -- this function never runs `git
// checkout` in root, the live working directory this process actually
// runs in, at all. Before this fix, this function checked the proposal
// branch out directly in root and checked back out to the original branch
// afterward; if that final checkout itself failed, the real repository was
// left on the new proposal branch with no retry and no recovery. This
// matters because triggerRepeatedInterventionProposal
// (cmd/improvement_pass.go) is this function's first real production
// caller, firing automatically and unattended at the end of every check --
// including inside unattended `aether run` autopilot loops -- with no
// human confirming the moment it fires. Working entirely in an isolated
// worktree means the live checkout's branch and working tree are never
// touched, so an automatic trigger can never leave a shared repository on
// the wrong branch or race a concurrent session's own checkout, whether
// this function succeeds or fails partway through.
func proposeSourceImprovement(candidateID string, evidence []string, changes sourceChangeSet) (sourceProposal, bool, error) {
	if store == nil {
		return sourceProposal{}, false, fmt.Errorf("no store initialized")
	}
	candidateID = strings.TrimSpace(candidateID)
	if candidateID == "" {
		return sourceProposal{}, false, fmt.Errorf("source proposal requires a non-empty candidate id")
	}
	if len(changes.Files) == 0 {
		return sourceProposal{}, false, fmt.Errorf("source proposal requires at least one file change")
	}

	proposalID := sourceProposalID(candidateID, evidence, changes)

	// Replay: an identical proposal already exists -- return it, create no
	// second branch.
	if existing, ok, err := readSourceProposal(proposalID); err != nil {
		return sourceProposal{}, false, err
	} else if ok {
		return existing, false, nil
	}

	root, err := os.Getwd()
	if err != nil {
		return sourceProposal{}, false, fmt.Errorf("resolve working directory: %w", err)
	}

	dirty, err := sourceProposalWorkingTreeIsDirty(root)
	if err != nil {
		return sourceProposal{}, false, fmt.Errorf("check working tree status: %w", err)
	}
	if dirty {
		return sourceProposal{}, false, fmt.Errorf(
			"source proposal refuses a dirty working tree -- commit or discard local changes before proposing a source improvement",
		)
	}

	baseCommit, err := gitOutputAt(root, "rev-parse", "HEAD")
	if err != nil {
		return sourceProposal{}, false, fmt.Errorf("resolve base commit: %w", err)
	}

	branch := sourceProposalBranchName(candidateID)
	if sourceProposalBranchExists(root, branch) {
		return sourceProposal{}, false, fmt.Errorf(
			"source proposal branch %q already exists -- refusing to overwrite it", branch,
		)
	}

	worktreePath, cleanupWorktree, err := sourceProposalIsolatedWorktree(root, branch, baseCommit)
	if err != nil {
		return sourceProposal{}, false, fmt.Errorf("create isolated proposal worktree for branch %q: %w", branch, err)
	}

	if err := sourceProposalApplyChangeSet(worktreePath, changes); err != nil {
		cleanupWorktree()
		_, _ = gitOutputAt(root, "branch", "-D", branch)
		return sourceProposal{}, false, fmt.Errorf("apply proposal change set: %w", err)
	}

	// The worktree checkout itself is disposable once the commit exists on
	// branch -- the branch is real, durable git history in root's own
	// repository regardless of whether this worktree directory survives.
	// cleanupWorktree removes ONLY the worktree checkout, never the
	// branch: unlike a build worker's throwaway worktree, a source
	// proposal's branch must survive for a human to review.
	cleanupWorktree()

	proposal := sourceProposal{
		ProposalID:        proposalID,
		CandidateID:       candidateID,
		EvidenceIDs:       append([]string{}, evidence...),
		Branch:            branch,
		BaseCommit:        baseCommit,
		CreatedAt:         time.Now().UTC().Format(time.RFC3339),
		VerificationState: sourceProposalStateProposed,
	}

	result, created, err := writeSourceProposal(proposal)
	if err != nil {
		return sourceProposal{}, false, err
	}
	if created {
		fmt.Println(renderSourceProposalAnnouncement(result))
	}
	return result, created, nil
}

// recordIndependentVerification marks proposalID verified, carrying resultID
// as the identifier of the independent verification that did it. It is the
// ONLY function in this file that ever sets a proposal's verification state
// to sourceProposalStateVerified (TestProposalCannotVerifyItself's own
// structural expectation), and it refuses a verifierID equal to the
// proposal's own creator (its CandidateID) by name -- the same contribution
// cannot both propose a change and independently verify it, mirroring the
// credit ledger's refusal of a contribution supplying its own outcome
// (cmd/recruitment_credit.go's recordRecruitmentCredit).
func recordIndependentVerification(proposalID, verifierID, resultID string) error {
	if store == nil {
		return fmt.Errorf("no store initialized")
	}
	proposalID = strings.TrimSpace(proposalID)
	verifierID = strings.TrimSpace(verifierID)
	resultID = strings.TrimSpace(resultID)
	if proposalID == "" {
		return fmt.Errorf("independent verification requires a non-empty proposal id")
	}
	if verifierID == "" {
		return fmt.Errorf("independent verification requires a non-empty verifier id")
	}
	if resultID == "" {
		return fmt.Errorf("independent verification requires a non-empty result id")
	}

	var file sourceProposalFile
	updateErr := store.UpdateJSONAtomically(sourceProposalPath, &file, func() error {
		for i := range file.Entries {
			if file.Entries[i].ProposalID != proposalID {
				continue
			}
			if verifierID == file.Entries[i].CandidateID {
				return fmt.Errorf(
					"independent verification refuses verifier %q: the same contribution that proposed %q cannot verify itself",
					verifierID, proposalID,
				)
			}
			file.Entries[i].VerificationState = sourceProposalStateVerified
			file.Entries[i].IndependentVerificationID = resultID
			return nil
		}
		return fmt.Errorf("independent verification found no proposal %q", proposalID)
	})
	return updateErr
}

// renderSourceProposalAnnouncement states, in plain English, what was
// proposed, which branch it is on, that nothing has been merged, and who
// has to act next.
func renderSourceProposalAnnouncement(proposal sourceProposal) string {
	return voiceLine("status", fmt.Sprintf(
		"Proposed a source change (%s) on branch %q, branched off %s. Nothing has been merged -- a person, not this system, must review it and merge it by hand before it takes effect.",
		proposal.ProposalID, proposal.Branch, proposal.BaseCommit,
	))
}

// renderSourceProposalReadiness states, in plain English, whether proposal
// is ready and, when it is not, names what verification is outstanding.
func renderSourceProposalReadiness(proposal sourceProposal) string {
	if proposal.VerificationState == sourceProposalStateVerified {
		return voiceLine("done", fmt.Sprintf(
			"Proposal %s is verified (independent check %s) and ready for a person to review and merge.",
			proposal.ProposalID, proposal.IndependentVerificationID,
		))
	}
	return voiceLine("blocked", fmt.Sprintf(
		"Proposal %s is not ready yet -- independent verification is still outstanding (current state: %s).",
		proposal.ProposalID, proposal.VerificationState,
	))
}

// sourceProposalForbiddenOperation names one family of operation a source
// proposal must never be able to perform.
type sourceProposalForbiddenOperation string

const (
	sourceProposalForbidMerge   sourceProposalForbiddenOperation = "merge"
	sourceProposalForbidPush    sourceProposalForbiddenOperation = "push"
	sourceProposalForbidPublish sourceProposalForbiddenOperation = "publish"
	sourceProposalForbidDeploy  sourceProposalForbiddenOperation = "deploy"
)

// sourceProposalForbiddenOperations is the named, authoritative list of
// operation families that must never be reachable from
// proposeSourceImprovement or anything it calls. This list plus the
// structural check that reads it (cmd/source_proposal_test.go's
// TestSourceProposalCannotMergePublishOrDeploy, which parses this
// declaration from source rather than retyping it --
// TestForbiddenOperationListIsReadFromTheSource asserts the two cannot
// drift) IS the guarantee that a proposal can only ever be proposed, never
// accepted, by this system's own code.
//
// The execution environment's own refusal to let a worker edit the
// project's automation settings is a second, independent layer that must
// never become the only one (Assumption V, 204-10-PLAN.md: "the sandbox is
// a second lock, never the only one" -- that guarantee is a property of one
// way of running this program and may not hold elsewhere). This repository's
// own defect register already records the general shape of this risk --
// a boundary trusted on a self-declared identity rather than proven
// independently -- at .planning/WINDOWS.md, entry 18 (the recruitment
// delegation-depth check trusting a short, fixed list of coordinator names
// on their own word alone, with nothing yet proving a caller claiming one of
// those names really is the coordinator). The structural check above is
// this file's own answer to that same class of gap: proof in the code, not
// trust in the environment running it.
var sourceProposalForbiddenOperations = []sourceProposalForbiddenOperation{
	sourceProposalForbidMerge,
	sourceProposalForbidPush,
	sourceProposalForbidPublish,
	sourceProposalForbidDeploy,
}
