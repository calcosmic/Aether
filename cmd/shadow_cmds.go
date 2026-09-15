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
	"unicode"

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
const shadowEvaluatorDefinitionPrefix = "shadow-evaluator-v2"

// shadowBaselineCandidateID mirrors pkg/shadow's own baselineAsCandidate
// literal id ("baseline") -- the classifier below never receives a
// shadow.Candidate for the baseline directly (it is unexported inside
// pkg/shadow), so it recognises the baseline subject the same way every
// other reader of a shadow.Candidate would have to: by its declared ID().
const shadowBaselineCandidateID = "baseline"

// shadowClassifierMinWordLength and shadowClassifierStopWords bound which
// words extracted from a fixture's own Title/Invariant text count as a
// meaningful subject-matter term for the run function's word-overlap
// match below (204-12, D-08). Short connector words carry no subject-matter
// signal on their own; this list is deliberately small and generic (never a
// per-fixture table) so it works unchanged against whatever fixtures the
// bank happens to carry.
const shadowClassifierMinWordLength = 5

var shadowClassifierStopWords = map[string]bool{
	"which": true, "where": true, "while": true, "after": true, "before": true,
	"their": true, "there": true, "these": true, "those": true, "would": true,
	"could": true, "should": true, "never": true, "always": true, "every": true,
	"about": true, "against": true, "because": true, "without": true,
	"through": true, "another": true, "cannot": true, "still": true,
	"being": true, "other": true, "under": true, "between": true,
	"first": true, "second": true, "third": true,
}

// shadowFixtureSubjectWords extracts the significant (long enough, not a
// stop word) lowercase words from fixture's own Title and Invariant text --
// the subject matter it protects, resolved from the fixture itself at run
// time rather than a hand-typed per-fixture table.
//
// WR-03 (204-REVIEW.md): a fixture whose Title and Invariant happen to
// consist only of short (< shadowClassifierMinWordLength) or common
// (shadowClassifierStopWords) words used to produce zero subject words
// here -- silently making that fixture permanently un-addressable by ANY
// real candidate (shadowTextNamesFixtureSubject returns false for every
// possible text, forever, with no error or warning). When the
// length-filtered pass finds nothing, this falls back to every non-stop
// word regardless of length, so a terse fixture still has at least one
// real word to match against rather than none at all. The stricter,
// length-filtered set is still preferred whenever it is non-empty, so this
// change is a pure safety net for the degenerate case, never a general
// loosening of the match.
func shadowFixtureSubjectWords(fixture regressionFixture) []string {
	text := fixture.Title + " " + fixture.Invariant
	rawWords := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})

	var words []string
	for _, w := range rawWords {
		if len(w) < shadowClassifierMinWordLength || shadowClassifierStopWords[w] {
			continue
		}
		words = append(words, w)
	}
	if len(words) > 0 {
		return words
	}

	var fallback []string
	for _, w := range rawWords {
		if shadowClassifierStopWords[w] {
			continue
		}
		fallback = append(fallback, w)
	}
	return fallback
}

// shadowTextNamesFixtureSubject reports whether text shares at least one
// significant word with fixture's own Title/Invariant text -- the run-time
// stand-in for "the candidate claims to address (or, for harms, threatens)
// this fixture's own confirmed incident."
func shadowTextNamesFixtureSubject(text string, fixture regressionFixture) bool {
	lower := strings.ToLower(text)
	for _, w := range shadowFixtureSubjectWords(fixture) {
		if strings.Contains(lower, w) {
			return true
		}
	}
	return false
}

// shadowFixtureByID returns the fixture in bank carrying id, or ok=false --
// a task naming no fixture in the bank resolves to nothing, never a guess.
func shadowFixtureByID(bank regressionFixtureBank, id string) (regressionFixture, bool) {
	for _, f := range bank.Fixtures {
		if f.ID == id {
			return f, true
		}
	}
	return regressionFixture{}, false
}

// shadowClassifyAgainstBank is the real per-fixture classifier (204-12,
// D-08), constructed here in the command layer and handed into
// shadow.NewFrozenEvaluator through the existing seam -- pkg/shadow itself
// never sees bank, a regressionFixture, or anything else this file owns.
//
// Rules, in order (each a property this plan's <behavior> block names):
//  1. A task naming no fixture in bank grades as not passing for every
//     subject alike -- an unresolvable task can never advantage either
//     side.
//  2. The baseline subject (ID() == "baseline") passes a fixture exactly
//     when that fixture carries a non-nil Guard -- genuinely protected
//     where a guard exists today, genuinely unprotected where it does not.
//  3. A non-baseline subject FAILS a fixture -- even a guarded one -- when
//     its own declared Harms() text names the subject matter that
//     fixture's Invariant protects (it is declaring a risk to the very
//     thing the fixture protects). Otherwise it passes a guarded fixture
//     outright, and passes an unguarded fixture only when its own declared
//     Scope()+ExpectedBenefit() text names that fixture's subject matter
//     (it is claiming to address that exact incident).
func shadowClassifyAgainstBank(bank regressionFixtureBank, candidate shadow.Candidate, task shadow.Task) shadow.Result {
	fixture, ok := shadowFixtureByID(bank, task.ID())
	if !ok {
		return shadow.NewResult(false)
	}
	if candidate.ID() == shadowBaselineCandidateID {
		return shadow.NewResult(fixture.Guard != nil)
	}
	if shadowTextNamesFixtureSubject(candidate.Harms(), fixture) {
		return shadow.NewResult(false)
	}
	if fixture.Guard != nil {
		return shadow.NewResult(true)
	}
	subjectText := candidate.Scope() + " " + candidate.ExpectedBenefit()
	return shadow.NewResult(shadowTextNamesFixtureSubject(subjectText, fixture))
}

// shadowBaseline builds the frozen representation of the current policy
// value. It reuses codex.PermissionProfileSchemaVersion -- the one declared
// schema-version constant this codebase already carries for "the
// permission and admission policy currently in force" (the same source
// cmd/episode_ledger.go's own PolicyVersion field is populated from).
func shadowBaseline() shadow.Baseline {
	return shadow.NewBaseline([]byte(fmt.Sprintf("permission-profile-schema-v%d", codex.PermissionProfileSchemaVersion)))
}

// shadowEvaluator builds the grader every comparison runs through: a real
// per-fixture classifier (shadowClassifyAgainstBank) over the committed
// regression-fixture bank (204-12, D-08), replacing the always-pass
// placeholder LEARN-06 shipped structurally correct but ungraded. The
// grader's own definition -- and therefore its Digest() -- tracks only the
// grader's own code version (shadowEvaluatorDefinitionPrefix's "v2" bump)
// plus the acceptance-criteria digest; it deliberately never folds in the
// bank's own content digest, so the grader's identity does not drift every
// time a fixture is guarded. A bank read failure (e.g. no committed bank
// yet) degrades to an empty bank -- every task then resolves to nothing,
// which shadowClassifyAgainstBank's own first rule already grades as "not
// passing for every subject alike," never a crash and never an advantage
// to either side.
func shadowEvaluator() shadow.FrozenEvaluator {
	criteria := shadow.NewAcceptanceCriteria([]byte(shadowAcceptanceCriteriaDefinition))
	criteriaDigest := criteria.Digest()
	def := append([]byte(shadowEvaluatorDefinitionPrefix), criteriaDigest[:]...)
	bank, err := loadFixtureBank()
	if err != nil {
		bank = regressionFixtureBank{}
	}
	run := func(candidate shadow.Candidate, task shadow.Task) shadow.Result {
		return shadowClassifyAgainstBank(bank, candidate, task)
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

// The two commands below are built and tested but NOT registered on rootCmd
// yet. This repository refuses a registered command that nothing calls
// (TestNoRegisteredSubcommandIsUnreferenced, and its allowlist may only
// shrink), and no plan in Phase 204 gives the shadow surface a caller: the
// promotion gate (204-09) drives pkg/shadow through its Go functions, not
// through these commands. Registering them is one line here once a plan
// exposes them through a wrapper or a worker discipline file -- recorded as
// an open item in .planning/WINDOWS.md so it is decided, not forgotten.
func init() {
	shadowDeclareCmd.Flags().String("id", "", "Candidate identifier")
	shadowDeclareCmd.Flags().String("scope", "", "What the candidate would change")
	shadowDeclareCmd.Flags().String("expected-benefit", "", "Why the candidate might help")
	shadowDeclareCmd.Flags().String("harms", "", "What could go wrong if the candidate is adopted")
	shadowDeclareCmd.Flags().String("expires", "", "RFC3339 timestamp the candidate's declaration expires at")
	shadowDeclareCmd.Flags().String("rollback-plan", "", "How to undo the candidate if it is adopted and later needs reverting")

	shadowCompareCmd.Flags().String("candidate-id", "", "Identifier of the declared candidate to compare")
}
