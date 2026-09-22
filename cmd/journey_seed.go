package cmd

// 207-02-PLAN.md Task 1 (UED-07): two hidden, undocumented commands used only
// by scripts/build-messy-practice-project.sh to construct two traps whose
// on-disk shape is authenticity-checked by the runtime itself
// (validatePlanningStageAuthority, classifySurveyFreshness's digest checks) --
// a hand-typed JSON literal for either would either be rejected outright or
// would silently fail to "trip the real classifier" (CLAUDE.md's Definition
// of Done). Both commands call the exact same internal writer functions a
// real `aether spec`/`aether colonize` run would use, so the fixture is
// derived the way the runtime derives it, never guessed.
//
// Both are Hidden: true (mirrors cmd/abandon_cmd.go, cmd/hook_cmds.go, etc.)
// -- internal fixture-construction tooling, not a new product feature or a
// new hard runtime refusal (milestone rule, CLAUDE.md "Phase 207: context").

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

var journeySeedStaleSurveyCmd = &cobra.Command{
	Use:          "journey-seed-stale-survey <repo-root>",
	Short:        "Internal: seed a territory snapshot pinned to the current revision (journey trap: out-of-date-code-map)",
	Hidden:       true,
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return journeySeedStaleSurvey(args[0])
	},
}

var journeySeedSupersededPlanCmd = &cobra.Command{
	Use:          "journey-seed-superseded-plan <repo-root>",
	Short:        "Internal: park a planning run against a specification that is then corrected (journey trap: specification-corrected-mid-planning)",
	Hidden:       true,
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return journeySeedSupersededPlan(args[0])
	},
}

func init() {
	rootCmd.AddCommand(journeySeedStaleSurveyCmd)
	rootCmd.AddCommand(journeySeedSupersededPlanCmd)
}

// journeySeedStaleSurvey writes a genuinely valid, digest-correct territory
// snapshot (every required survey artifact plus a self-consistent
// territory-snapshot.json, using computeTerritorySnapshotID and
// stableRepoIdentity exactly as classifySurveyFreshness itself would produce
// them) pinned to the repository's CURRENT revision. It prints that revision
// to stdout so the caller can create surveyStaleCommitThreshold-or-more
// commits afterward -- staleness then comes from a genuine commit count, per
// 207-RESEARCH.md's own guidance, never from a fabricated digest mismatch.
func journeySeedStaleSurvey(root string) error {
	canonicalRoot, err := canonicalTerritoryRoot(root)
	if err != nil {
		return fmt.Errorf("resolve repository root: %w", err)
	}
	sourceRevision, err := currentTerritoryRevision(canonicalRoot)
	if err != nil {
		return fmt.Errorf("read current repository revision: %w", err)
	}

	digests := make(map[string]string)
	for _, rel := range requiredTerritoryArtifactPaths() {
		full := filepath.Join(canonicalRoot, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return fmt.Errorf("create directory for %s: %w", rel, err)
		}
		content := []byte("journey trap: out-of-date-code-map placeholder artifact -- content is irrelevant, only self-consistency with its own recorded digest matters\n")
		if err := os.WriteFile(full, content, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", rel, err)
		}
		sum := sha256.Sum256(content)
		digests[rel] = hex.EncodeToString(sum[:])
	}

	snapshot := territorySnapshotMetadata{
		SchemaVersion:      territorySnapshotSchemaVersion,
		RepositoryIdentity: stableRepoIdentity(canonicalRoot),
		RepositoryRoot:     canonicalRoot,
		SourceRevision:     sourceRevision,
		GeneratedAt:        time.Now().UTC(),
		ArtifactDigests:    digests,
	}
	snapshot.SnapshotID = computeTerritorySnapshotID(snapshot)

	payload, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal territory snapshot: %w", err)
	}
	snapshotPath := filepath.Join(canonicalRoot, filepath.FromSlash(territorySnapshotRelativePath))
	if err := os.MkdirAll(filepath.Dir(snapshotPath), 0o755); err != nil {
		return fmt.Errorf("create survey directory: %w", err)
	}
	if err := os.WriteFile(snapshotPath, payload, 0o644); err != nil {
		return fmt.Errorf("write territory snapshot: %w", err)
	}

	fmt.Fprintln(os.Stdout, sourceRevision)
	return nil
}

// journeyTrapSupersededPlanRunID is the fixed run ID the specification-
// corrected-mid-planning trap always uses -- a stable, well-known name the
// builder script's idempotency guard and the Go test's assertion both key
// off, rather than a randomly generated one.
const journeyTrapSupersededPlanRunID = "journey-trap-run"

// journeySeedSupersededPlan constructs, using the runtime's own specification
// writers (createSpecificationDraft / approveSpecification / reviseSpecification
// -- the exact functions `aether discuss`/`aether spec` call), a specification
// revision A that is approved, a planning run parked mid-flight
// (stage=route_running, which planningStageAwaitsAWorker treats as awaiting a
// worker with no further manifest file needed) bound to revision A, and then a
// corrected, approved successor revision B -- so the parked run's own frozen
// binding (A) no longer matches the specification's current approved revision
// (B). planningRunIsSuperseded(state, approved) is exactly the real runtime
// predicate that becomes true here, the same one discoverParkedPlanningRun
// uses to refuse resuming a run made for a specification the owner has since
// corrected (2026-09-21 blocker #4, commit f210d5d4).
//
// A hand-typed stage-state.json could not pass validatePlanningStageAuthority
// (it requires an approval receipt ID and a SHA-256-shaped approval receipt
// hash that must trace to a real approved revision) without either being
// rejected or silently not triggering the real refusal this trap exists to
// prove -- so this command calls the real writers instead of inventing the
// shape (CLAUDE.md's Definition of Done).
func journeySeedSupersededPlan(root string) error {
	repositoryRoot, err := canonicalSpecificationRoot(root)
	if err != nil {
		return fmt.Errorf("resolve specification repository root: %w", err)
	}
	now := time.Now().UTC()

	state, err := loadSpecificationColonyState(repositoryRoot)
	if err != nil {
		return fmt.Errorf("load colony state: %w", err)
	}
	sessionID := "journey-trap-session"
	if state.SessionID != nil && strings.TrimSpace(*state.SessionID) != "" {
		sessionID = strings.TrimSpace(*state.SessionID)
	}
	scope := colony.SpecScope{Kind: colony.SpecScopeWholeGoal, GoalID: "journey-trap-goal", SessionID: sessionID}

	item := func(lineage, description string) specificationItemInput {
		return specificationItemInput{Lineage: lineage, Description: description, EvidenceIDs: []string{"evidence:journey-trap"}}
	}

	draftReq := specificationDraftRequest{
		Scope:                scope,
		Outcomes:             []specificationItemInput{item("journey-trap-outcome", "Practice the daily lifecycle on a messy real project.")},
		IncludedBehaviors:    []specificationItemInput{item("journey-trap-behavior", "The practice project exercises Aether's lifecycle end to end.")},
		Exclusions:           []specificationItemInput{item("journey-trap-exclusion", "Nothing in this project is meant to ship.")},
		BindingDecisions:     []specificationItemInput{item("journey-trap-decision", "The owner accepted this scratch goal for journey testing.")},
		Requirements:         []specificationItemInput{item("journey-trap-requirement", "Exercise the daily lifecycle against a messy project.")},
		AcceptanceChecks:     []specificationItemInput{{Lineage: "journey-trap-acceptance", Description: "The owner can see the journey run.", Verification: "Run the journey and inspect its report.", EvidenceIDs: []string{"evidence:journey-trap"}}},
		NegativeExpectations: []specificationItemInput{item("journey-trap-negative", "The result must not touch anything outside this scratch project.")},
		RecoveryExpectations: []specificationItemInput{item("journey-trap-recovery", "If interrupted, resume the practice project safely.")},
		AffectedPublicPaths:  []specificationItemInput{{Lineage: "journey-trap-path", Description: "The practice project's own source tree.", Path: "src/main.go", EvidenceIDs: []string{"evidence:journey-trap"}}},
		CreatedAt:            now,
	}
	draft, err := createSpecificationDraft(repositoryRoot, draftReq, specificationMutationOptions{})
	if err != nil {
		return fmt.Errorf("create specification draft (revision A): %w", err)
	}
	revA := draft.Revision

	tokenA := specificationApprovalToken(draft.Specification.ID, revA.ID, revA.ContentHash)
	if _, err := approveSpecification(repositoryRoot, specificationApprovalRequest{
		RevisionID: revA.ID, RevisionContentHash: revA.ContentHash,
		ApprovalToken: tokenA, ApprovedBy: "journey-trap", ApprovedAt: now,
	}, specificationMutationOptions{}); err != nil {
		return fmt.Errorf("approve specification revision A: %w", err)
	}

	stateAfterA, err := loadSpecificationColonyState(repositoryRoot)
	if err != nil {
		return fmt.Errorf("reload colony state after approving revision A: %w", err)
	}
	approvedA, err := requireApprovedPlanningSpecification(repositoryRoot, stateAfterA)
	if err != nil {
		return fmt.Errorf("resolve approved binding for revision A: %w", err)
	}

	stageState := planningStageState{
		Stage:                planningStageRouteRunning,
		RunID:                journeyTrapSupersededPlanRunID,
		Pass:                 1,
		Preset:               planningStagePresetBalanced,
		Specification:        approvedA.Binding,
		BasePlanRevisionID:   "journey-trap-base-plan",
		BasePlanRevisionHash: strings.Repeat("a", 64),
		PriorCardHash:        strings.Repeat("b", 64),
		InputFrontierHash:    strings.Repeat("c", 64),
	}
	if err := validatePlanningStageAuthority(stageState); err != nil {
		return fmt.Errorf("constructed parked planning run state failed its own authority check: %w", err)
	}
	payload, err := marshalPlanningStageJSON(stageState)
	if err != nil {
		return fmt.Errorf("marshal parked planning run state: %w", err)
	}
	statePath := filepath.Join(repositoryRoot, filepath.FromSlash(planningStageStateRepositoryPath(journeyTrapSupersededPlanRunID)))
	if err := os.MkdirAll(filepath.Dir(statePath), 0o755); err != nil {
		return fmt.Errorf("create planning run directory: %w", err)
	}
	if err := os.WriteFile(statePath, payload, 0o644); err != nil {
		return fmt.Errorf("write parked planning run state: %w", err)
	}

	// The owner corrects the specification: a successor revision B is
	// proposed and approved, superseding A -- while the parked run's own
	// stage-state.json above still points at A.
	reviseReq := specificationRevisionRequest{
		PredecessorRevisionID:  revA.ID,
		PredecessorContentHash: revA.ContentHash,
		Scope:                  scope,
		Changes: []specificationRevisionChange{{
			Operation: specificationChangeAdd,
			Section:   specificationSectionBindingDecisions,
			Item:      item("journey-trap-owner-correction", "The owner corrected the specification while a planning run was parked mid-flight."),
		}},
		CreatedAt: now.Add(time.Minute),
	}
	revised, err := reviseSpecification(repositoryRoot, reviseReq, specificationMutationOptions{})
	if err != nil {
		return fmt.Errorf("revise specification to revision B: %w", err)
	}
	revB := revised.Revision
	tokenB := specificationApprovalToken(revised.Specification.ID, revB.ID, revB.ContentHash)
	if _, err := approveSpecification(repositoryRoot, specificationApprovalRequest{
		RevisionID: revB.ID, RevisionContentHash: revB.ContentHash,
		ApprovalToken: tokenB, ApprovedBy: "journey-trap", ApprovedAt: now.Add(2 * time.Minute),
	}, specificationMutationOptions{}); err != nil {
		return fmt.Errorf("approve specification revision B: %w", err)
	}

	fmt.Fprintln(os.Stdout, journeyTrapSupersededPlanRunID)
	return nil
}
