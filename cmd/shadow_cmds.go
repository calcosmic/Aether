package cmd

// LEARN-06 (204-08-PLAN.md): the command surface over pkg/shadow. Two
// subcommands -- shadow-declare and shadow-compare -- declare a candidate
// (a proposed setting or rule, never a piece of code, Planner Assumption P)
// and run it beside the frozen baseline over the fixture bank's visible
// task set and the hidden holdout set, recording every comparison's result
// durably through the episode ledger (cmd/episode_ledger.go).
//
// The hidden-holdout resolver (declared in cmd/eval_gates.go) is called
// exactly once in this file, HERE and only here -- pkg/shadow is never
// given the holdout file's path, which is what keeps a candidate
// declaration structurally unable to reach it. See pkg/shadow/comparison.go's
// own doc comment on Compare for the other half of that division of
// responsibility.
import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/shadow"
	"github.com/spf13/cobra"
)

// shadowCandidateStorePath is the store-relative path declareShadowCandidate
// persists to -- one store, one directory, following credit/records.json
// and episodes/ledger.json's own convention.
const shadowCandidateStorePath = "shadow/candidates.json"

// shadowCandidateRecord is the durable, on-disk form of a declared
// candidate -- the same six declarations shadow.Candidate requires, plus a
// content digest (so a second declaration under the same id can be told
// apart as a replay or a genuine edit attempt) and when it was declared.
type shadowCandidateRecord struct {
	ID              string `json:"id"`
	Scope           string `json:"scope"`
	ExpectedBenefit string `json:"expected_benefit"`
	Harms           string `json:"harms"`
	Expiry          string `json:"expiry"`
	RollbackPlan    string `json:"rollback_plan"`
	ContentDigest   string `json:"content_digest"`
	DeclaredAt      string `json:"declared_at"`
}

// shadowCandidateFile is the on-disk container at shadowCandidateStorePath.
type shadowCandidateFile struct {
	Entries []shadowCandidateRecord `json:"entries"`
}

// shadowCandidateContentDigest is the deterministic content digest a
// candidate's declared fields hash to -- used to tell a replay (identical
// content under the same id) apart from a genuine edit attempt (different
// content under the same id), which is refused rather than applied.
func shadowCandidateContentDigest(scope, expectedBenefit, harms, expiry, rollbackPlan string) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{scope, expectedBenefit, harms, expiry, rollbackPlan}, "|")))
	return hex.EncodeToString(sum[:])
}

// errShadowCandidateNoChange is the internal replay sentinel
// declareShadowCandidate returns from its own UpdateJSONAtomically mutate
// closure to abort the write on a replay -- mirroring
// errRecruitmentCreditNoChange's and errEpisodeLedgerNoChange's role in
// this project's other append-only, replay-safe writers.
var errShadowCandidateNoChange = errors.New("shadow candidate already declared with identical content")

// declareShadowCandidate is the ONE function in cmd/ that writes
// shadow/candidates.json (TestCandidateStoreIsAppendOnlyAndRefusesEdits
// enforces this by name). It constructs the candidate through
// shadow.NewCandidate first, so every refusal -- a missing declaration, an
// expiry already past, a declaration naming its own grader -- comes from
// the package that owns the rules, never re-implemented here. A second
// declaration under the same identifier with different content is refused
// by name, never applied as a silent edit; an identical declaration is a
// replay and writes nothing. A stored candidate is never edited -- a
// changed proposal is a new candidate under a new identifier.
func declareShadowCandidate(id, scope, expectedBenefit, harms, expires, rollbackPlan string) (shadowCandidateRecord, bool, error) {
	if store == nil {
		return shadowCandidateRecord{}, false, fmt.Errorf("no store initialized")
	}
	expiry, err := time.Parse(time.RFC3339, strings.TrimSpace(expires))
	if err != nil {
		return shadowCandidateRecord{}, false, fmt.Errorf("candidate expiry %q is not a valid RFC3339 timestamp: %w", expires, err)
	}
	if _, err := shadow.NewCandidate(id, scope, expectedBenefit, harms, expiry, rollbackPlan); err != nil {
		return shadowCandidateRecord{}, false, err
	}

	normalizedExpiry := expiry.UTC().Format(time.RFC3339)
	digest := shadowCandidateContentDigest(scope, expectedBenefit, harms, normalizedExpiry, rollbackPlan)
	proposed := shadowCandidateRecord{
		ID:              id,
		Scope:           scope,
		ExpectedBenefit: expectedBenefit,
		Harms:           harms,
		Expiry:          normalizedExpiry,
		RollbackPlan:    rollbackPlan,
		ContentDigest:   digest,
		DeclaredAt:      time.Now().UTC().Format(time.RFC3339),
	}

	var result shadowCandidateRecord
	var file shadowCandidateFile
	updateErr := store.UpdateJSONAtomically(shadowCandidateStorePath, &file, func() error {
		for _, existing := range file.Entries {
			if existing.ID != id {
				continue
			}
			if existing.ContentDigest == digest {
				result = existing
				return errShadowCandidateNoChange
			}
			return fmt.Errorf(
				"candidate %q is already declared with different content -- a stored candidate is never edited; declare a new candidate under a different id instead",
				id,
			)
		}
		file.Entries = append(file.Entries, proposed)
		result = proposed
		return nil
	})
	if updateErr != nil {
		if errors.Is(updateErr, errShadowCandidateNoChange) {
			return result, false, nil
		}
		return shadowCandidateRecord{}, false, updateErr
	}
	return result, true, nil
}

// loadShadowCandidate returns the stored candidate record named id, or
// ok=false if none has been declared.
func loadShadowCandidate(id string) (shadowCandidateRecord, bool, error) {
	if store == nil {
		return shadowCandidateRecord{}, false, fmt.Errorf("no store initialized")
	}
	var file shadowCandidateFile
	if err := store.LoadJSON(shadowCandidateStorePath, &file); err != nil {
		return shadowCandidateRecord{}, false, nil
	}
	for _, r := range file.Entries {
		if r.ID == id {
			return r, true, nil
		}
	}
	return shadowCandidateRecord{}, false, nil
}

// shadowCandidateToDomain converts a stored candidate record back into a
// shadow.Candidate, re-running it through shadow.NewCandidate so the same
// rules apply on every read as on the original declaration.
func shadowCandidateToDomain(record shadowCandidateRecord) (shadow.Candidate, error) {
	expiry, err := time.Parse(time.RFC3339, record.Expiry)
	if err != nil {
		return shadow.Candidate{}, fmt.Errorf("stored candidate %q has an unparseable expiry %q: %w", record.ID, record.Expiry, err)
	}
	return shadow.NewCandidate(record.ID, record.Scope, record.ExpectedBenefit, record.Harms, expiry, record.RollbackPlan)
}

// shadowAcceptanceCriteriaDefinition and shadowEvaluatorDefinitionPrefix are
// the fixed definition bytes the grader and its acceptance criteria are
// built from. Folding the criteria's own digest into the evaluator's
// definition (shadowEvaluator, below) is what makes the grader's identity
// cover the acceptance criteria's identity -- altering the criteria alters
// the grader's own digest, without pkg/shadow's own constructor needing a
// criteria parameter (see pkg/shadow/evaluator.go's doc comment on
// NewFrozenEvaluator).
const shadowAcceptanceCriteriaDefinition = "shadow-acceptance-v1"
const shadowEvaluatorDefinitionPrefix = "shadow-evaluator-v1"

// shadowBaseline builds the frozen representation of the current policy
// value. It reuses codex.PermissionProfileSchemaVersion -- the one declared
// schema-version constant this codebase already carries for "the
// permission and admission policy currently in force" (the same source
// cmd/episode_ledger.go's own PolicyVersion field is populated from).
func shadowBaseline() shadow.Baseline {
	return shadow.NewBaseline([]byte(fmt.Sprintf("permission-profile-schema-v%d", codex.PermissionProfileSchemaVersion)))
}

// shadowEvaluator builds the grader every comparison runs through. The run
// function is a deterministic, always-pass placeholder: the concrete
// per-fixture grading mechanism (re-running a fixture's own guard test
// live, or a real policy predicate) is out of scope for this plan -- LEARN-06
// wires the structural isolation and comparison mechanism a candidate
// cannot reach or alter, not a production classifier for what "passing"
// concretely means for every fixture kind. Every comparison therefore
// reports a tied verdict against an unchanged baseline until a real run
// function is wired in by the plan that owns that classifier. Documented
// as a deviation in this plan's SUMMARY.
func shadowEvaluator() shadow.FrozenEvaluator {
	criteria := shadow.NewAcceptanceCriteria([]byte(shadowAcceptanceCriteriaDefinition))
	criteriaDigest := criteria.Digest()
	def := append([]byte(shadowEvaluatorDefinitionPrefix), criteriaDigest[:]...)
	run := func(shadow.Candidate, shadow.Task) shadow.Result {
		return shadow.NewResult(true)
	}
	return shadow.NewFrozenEvaluator(def, run)
}

// shadowVisibleAndHoldoutTasks resolves the visible task set from the
// fixture bank, excluding every fixture the hidden holdout resolver below
// (Task 2's own resolver, cmd/eval_gates.go) reports as held back. The call
// below is the ONE call to that resolver in this file
// (TestHoldoutResolutionHappensOnlyInTheCommandLayer enforces this by
// name) -- pkg/shadow itself never resolves the holdout file.
func shadowVisibleAndHoldoutTasks() (visible []shadow.Task, holdout []shadow.Task, err error) {
	bank, err := loadFixtureBank()
	if err != nil {
		return nil, nil, err
	}
	holdoutFixtures := resolveEvalGateHoldouts(bank)
	holdoutIDs := make(map[string]bool, len(holdoutFixtures))
	for _, f := range holdoutFixtures {
		holdoutIDs[f.ID] = true
		holdout = append(holdout, shadow.NewTask(f.ID))
	}
	for _, f := range bank.Fixtures {
		if holdoutIDs[f.ID] {
			continue
		}
		visible = append(visible, shadow.NewTask(f.ID))
	}
	return visible, holdout, nil
}

// shadowComparisonEpisodeID derives the durable episode identifier a
// candidate's own comparison is recorded under. One candidate, one
// comparison episode -- a second shadow-compare run for the same candidate
// re-derives the identical episode id, which is what makes the durable
// write replay-safe (recordEpisodeOutcome's own digest-based idempotency
// then does the rest: identical scores and digests collapse to the same
// record id and write nothing).
func shadowComparisonEpisodeID(candidateID string) string {
	return "shadow-compare:" + candidateID
}

// recordShadowComparisonOutcome writes comparison's result durably through
// recordEpisodeOutcome, cmd/episode_ledger.go's own one writer -- an open
// record and a terminal record, following the exact discipline every other
// lifecycle lane in this project already uses. A replay (an identical
// comparison run a second time) writes nothing: both calls are digest-based
// and idempotent, and every value that feeds a comparison's outcome
// (verdict, both digests, all four scores) is deterministic given the same
// candidate, baseline, evaluator and task sets, so the second run's payload
// digest is byte-identical to the first and recordEpisodeOutcome's own
// replay branch takes over.
func recordShadowComparisonOutcome(comparison shadow.Comparison, record shadowCandidateRecord) (bool, error) {
	episodeID := shadowComparisonEpisodeID(comparison.CandidateID)

	if _, _, err := recordEpisodeOutcome(episodeLedgerRecord{
		RecordKind:  episodeLedgerRecordKindOpened,
		EpisodeID:   episodeID,
		EpisodeKind: "shadow_comparison",
		StartedAt:   record.DeclaredAt,
	}); err != nil {
		return false, fmt.Errorf("record shadow comparison open: %w", err)
	}

	evaluatorDigest := comparison.EvaluatorDigest
	baselineDigest := comparison.BaselineDigest
	_, credited, err := recordEpisodeOutcome(episodeLedgerRecord{
		RecordKind:      episodeLedgerRecordKindClosed,
		EpisodeID:       episodeID,
		EpisodeKind:     "shadow_comparison",
		PolicyVersion:   hex.EncodeToString(baselineDigest[:]),
		EvaluatorDigest: hex.EncodeToString(evaluatorDigest[:]),
		EvidenceIDs: []string{
			fmt.Sprintf("visible_baseline:%d/%d", comparison.VisibleBaseline.Numerator, comparison.VisibleBaseline.Denominator),
			fmt.Sprintf("visible_candidate:%d/%d", comparison.VisibleCandidate.Numerator, comparison.VisibleCandidate.Denominator),
			fmt.Sprintf("holdout_baseline:%d/%d", comparison.HoldoutBaseline.Numerator, comparison.HoldoutBaseline.Denominator),
			fmt.Sprintf("holdout_candidate:%d/%d", comparison.HoldoutCandidate.Numerator, comparison.HoldoutCandidate.Denominator),
		},
		TerminalResult: string(comparison.Verdict),
		EndedAt:        time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return false, fmt.Errorf("record shadow comparison outcome: %w", err)
	}
	return credited, nil
}

// shadowVerdictWording translates a shadow.Verdict into an ordinary
// sentence -- no rendered line ever prints the raw internal token.
func shadowVerdictWording(v shadow.Verdict) string {
	switch v {
	case shadow.VerdictBeneficial:
		return "this looks worth adopting -- it did better than the current behaviour on the work it could see, and did not do worse on the checks it never saw"
	case shadow.VerdictNotBeneficial:
		return "this did not do better than the current behaviour on the work it could see -- not recommended"
	case shadow.VerdictOverfit:
		return "this looked better only on the work it could see, and did worse on the checks it never saw -- it fitted itself to the visible work rather than genuinely improving things, and is not recommended"
	case shadow.VerdictTied:
		return "this performed exactly the same as the current behaviour -- nothing to recommend"
	case shadow.VerdictInconclusive:
		return "there was no work to compare it against yet -- no recommendation can be made"
	default:
		return "the verdict could not be translated"
	}
}

// shadowScoreLine renders one score line as a count over a total for the
// proposal and for the current behaviour, with the percentage shown beside
// the count, never instead of it.
func shadowScoreLine(label string, candidateScore, baselineScore shadow.Score) string {
	return voiceLine("status", fmt.Sprintf(
		"%s: the proposal scored %d out of %d (%.0f%%), the current behaviour scored %d out of %d (%.0f%%)",
		label,
		candidateScore.Numerator, candidateScore.Denominator, candidateScore.Percent(),
		baselineScore.Numerator, baselineScore.Denominator, baselineScore.Percent(),
	))
}

// renderShadowComparison is the owner-facing result of a shadow-compare
// run: every line opens with a glyph from the one shared table, the
// verdict is stated in ordinary words, each score is shown as a count over
// a total alongside its percentage, and the candidate's five declarations
// are restated in plain English so the owner can see what was proposed
// without opening anything.
func renderShadowComparison(comparison shadow.Comparison, record shadowCandidateRecord) string {
	var sb strings.Builder
	sb.WriteString(voiceLine("decision", fmt.Sprintf("Comparison for proposal %q", comparison.CandidateID)) + "\n")
	sb.WriteString(voiceLine("status", "Verdict: "+shadowVerdictWording(comparison.Verdict)) + "\n")
	sb.WriteString(shadowScoreLine("On the work it could see", comparison.VisibleCandidate, comparison.VisibleBaseline) + "\n")
	sb.WriteString(shadowScoreLine("On the checks it never saw", comparison.HoldoutCandidate, comparison.HoldoutBaseline) + "\n")
	sb.WriteString(voiceLine("files", "What would change: "+record.Scope) + "\n")
	sb.WriteString(voiceLine("files", "Why it might help: "+record.ExpectedBenefit) + "\n")
	sb.WriteString(voiceLine("warning", "What could go wrong: "+record.Harms) + "\n")
	sb.WriteString(voiceLine("checkpoint", "Expires: "+record.Expiry) + "\n")
	sb.WriteString(voiceLine("alternative", "How to undo it: "+record.RollbackPlan) + "\n")
	return sb.String()
}

// shadowComparisonOutcome carries the result of runShadowCompare: the full
// comparison, the rendered plain-English result, and whether this run
// actually wrote a new durable record (false on a replay).
type shadowComparisonOutcome struct {
	Comparison shadow.Comparison
	Rendered   string
	Credited   bool
}

// runShadowCompare loads the stored candidate, builds the baseline and the
// grader, resolves the visible and holdout task sets, runs the comparison
// and records its result durably. A comparison run without a stored
// candidate is refused by name.
func runShadowCompare(candidateID string) (shadowComparisonOutcome, error) {
	candidateID = strings.TrimSpace(candidateID)
	if candidateID == "" {
		return shadowComparisonOutcome{}, fmt.Errorf("shadow-compare requires --candidate-id")
	}
	record, found, err := loadShadowCandidate(candidateID)
	if err != nil {
		return shadowComparisonOutcome{}, err
	}
	if !found {
		return shadowComparisonOutcome{}, fmt.Errorf(
			"no candidate declared with id %q -- declare it first with shadow-declare before comparing it", candidateID,
		)
	}
	candidate, err := shadowCandidateToDomain(record)
	if err != nil {
		return shadowComparisonOutcome{}, err
	}

	visible, holdout, err := shadowVisibleAndHoldoutTasks()
	if err != nil {
		return shadowComparisonOutcome{}, err
	}

	baseline := shadowBaseline()
	evaluator := shadowEvaluator()

	comparison, err := shadow.Compare(baseline, candidate, evaluator, visible, holdout)
	if err != nil {
		return shadowComparisonOutcome{}, err
	}

	credited, err := recordShadowComparisonOutcome(comparison, record)
	if err != nil {
		return shadowComparisonOutcome{}, err
	}

	return shadowComparisonOutcome{
		Comparison: comparison,
		Rendered:   renderShadowComparison(comparison, record),
		Credited:   credited,
	}, nil
}

// shadowDeclareCmd declares a candidate: a proposed setting or rule change,
// never a piece of code (Planner Assumption P), carrying the five
// declarations LEARN-06 requires beyond its identifier.
var shadowDeclareCmd = &cobra.Command{
	Use:   "shadow-declare",
	Short: "Declare a candidate to try beside the current behaviour",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		scope, _ := cmd.Flags().GetString("scope")
		expectedBenefit, _ := cmd.Flags().GetString("expected-benefit")
		harms, _ := cmd.Flags().GetString("harms")
		expires, _ := cmd.Flags().GetString("expires")
		rollbackPlan, _ := cmd.Flags().GetString("rollback-plan")

		record, isNew, err := declareShadowCandidate(id, scope, expectedBenefit, harms, expires, rollbackPlan)
		if err != nil {
			outputErrorMessage(err.Error())
			return nil
		}
		outputOK(map[string]interface{}{
			"declared":       true,
			"is_new":         isNew,
			"id":             record.ID,
			"content_digest": record.ContentDigest,
			"message":        voiceLine("done", fmt.Sprintf("Proposal %q declared -- expires %s", record.ID, record.Expiry)),
		})
		return nil
	},
}

// shadowCompareCmd runs a declared candidate beside the frozen baseline and
// records the verdict durably.
var shadowCompareCmd = &cobra.Command{
	Use:   "shadow-compare",
	Short: "Run a declared candidate beside the current behaviour and record the verdict",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		candidateID, _ := cmd.Flags().GetString("candidate-id")
		outcome, err := runShadowCompare(candidateID)
		if err != nil {
			outputErrorMessage(err.Error())
			return nil
		}
		outputOK(map[string]interface{}{
			"compared":    true,
			"candidate":   outcome.Comparison.CandidateID,
			"verdict":     string(outcome.Comparison.Verdict),
			"recommended": outcome.Comparison.Recommended(),
			"credited":    outcome.Credited,
			"rendered":    outcome.Rendered,
		})
		return nil
	},
}

func init() {
	shadowDeclareCmd.Flags().String("id", "", "Candidate identifier")
	shadowDeclareCmd.Flags().String("scope", "", "What the candidate would change")
	shadowDeclareCmd.Flags().String("expected-benefit", "", "Why the candidate might help")
	shadowDeclareCmd.Flags().String("harms", "", "What could go wrong if the candidate is adopted")
	shadowDeclareCmd.Flags().String("expires", "", "RFC3339 timestamp the candidate's declaration expires at")
	shadowDeclareCmd.Flags().String("rollback-plan", "", "How to undo the candidate if it is adopted and later needs reverting")
	rootCmd.AddCommand(shadowDeclareCmd)

	shadowCompareCmd.Flags().String("candidate-id", "", "Identifier of the declared candidate to compare")
	rootCmd.AddCommand(shadowCompareCmd)
}
