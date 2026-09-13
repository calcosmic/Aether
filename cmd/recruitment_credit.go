package cmd

// BIO-08/CEC-07 (203-12-PLAN.md): the credit record -- the runtime's own
// provable answer to "did this help?" A recruitment result, a note, a
// memory item, or a specialist contribution earns credit only when the
// runtime has recorded BOTH the decision it changed and the later verified
// outcome that followed, including a neutral or harmful one; a contribution
// with neither fact recorded earns no credit and no record at all -- never a
// default or inferred credit.

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// recruitmentCreditPath is the store-relative path recordRecruitmentCredit
// persists to. New data file, per this plan's frontmatter.
const recruitmentCreditPath = "credit/records.json"

// recruitmentCreditOutcome is the declared, closed outcome vocabulary a
// credit record's Outcome field may hold. helpful/neutral/harmful are the
// three earnable outcomes CEC-07 requires be equally recordable -- a
// contribution that made things worse must be exactly as visible as one
// that helped. pending names the one legitimate partial state: a changed
// decision has been recorded with no verified outcome yet, distinct from
// both an earned outcome and no record at all.
type recruitmentCreditOutcome string

const (
	recruitmentCreditOutcomeHelpful recruitmentCreditOutcome = "helpful"
	recruitmentCreditOutcomeNeutral recruitmentCreditOutcome = "neutral"
	recruitmentCreditOutcomeHarmful recruitmentCreditOutcome = "harmful"
	recruitmentCreditOutcomePending recruitmentCreditOutcome = "pending"
)

// recruitmentCreditOutcomeVocabulary is the declared, closed set of every
// outcome a credit record may carry. Mirrors the
// recruitmentTerminalStatuses()/ColonyLiveTopics() completeness convention
// (cmd/recruitment_result.go, pkg/events/colony_live.go): an outcome added
// to the const block above must also be added here.
var recruitmentCreditOutcomeVocabulary = []recruitmentCreditOutcome{
	recruitmentCreditOutcomeHelpful,
	recruitmentCreditOutcomeNeutral,
	recruitmentCreditOutcomeHarmful,
	recruitmentCreditOutcomePending,
}

// recruitmentCreditOutcomeNames returns the string form of every declared
// outcome, for display and for %v-formatted refusal messages.
func recruitmentCreditOutcomeNames() []string {
	names := make([]string, 0, len(recruitmentCreditOutcomeVocabulary))
	for _, o := range recruitmentCreditOutcomeVocabulary {
		names = append(names, string(o))
	}
	return names
}

func recruitmentCreditOutcomeDeclared(outcome recruitmentCreditOutcome) bool {
	for _, o := range recruitmentCreditOutcomeVocabulary {
		if o == outcome {
			return true
		}
	}
	return false
}

// recruitmentContributionKind names the four kinds of contribution CEC-07
// names as creditable: a recruitment result, a note (a pheromone signal), a
// memory item, and a specialist contribution.
type recruitmentContributionKind string

const (
	recruitmentContributionRecruitmentResult recruitmentContributionKind = "recruitment_result"
	recruitmentContributionNote              recruitmentContributionKind = "note"
	recruitmentContributionMemoryItem        recruitmentContributionKind = "memory_item"
	recruitmentContributionSpecialist        recruitmentContributionKind = "specialist_contribution"
)

// recruitmentContributionKindVocabulary is the declared, closed set of
// every contribution kind a credit record may carry. Same completeness
// convention as recruitmentCreditOutcomeVocabulary above.
var recruitmentContributionKindVocabulary = []recruitmentContributionKind{
	recruitmentContributionRecruitmentResult,
	recruitmentContributionNote,
	recruitmentContributionMemoryItem,
	recruitmentContributionSpecialist,
}

// recruitmentContributionKinds returns the string form of every declared
// contribution kind, for display and for %v-formatted refusal messages.
func recruitmentContributionKinds() []string {
	names := make([]string, 0, len(recruitmentContributionKindVocabulary))
	for _, k := range recruitmentContributionKindVocabulary {
		names = append(names, string(k))
	}
	return names
}

func recruitmentContributionKindDeclared(kind recruitmentContributionKind) bool {
	for _, k := range recruitmentContributionKindVocabulary {
		if k == kind {
			return true
		}
	}
	return false
}

// recruitmentCreditPendingWording names a changed decision recorded with no
// verified outcome yet -- distinct from both a genuinely earned outcome and
// the existing "no record at all" wording (cmd/agency_contract.go's
// AgencyMeasuredEffectPending), which renderRecruitmentCreditOutcome uses
// for that latter case rather than a new phrase, so the vocabulary the
// owner reads does not fork.
const recruitmentCreditPendingWording = "Decision recorded — outcome not yet verified"

// recruitmentCreditRecord is the runtime's own provable record of what a
// contribution did: which contribution, of what kind, which recorded
// decision it changed, which effect evidence proved the later outcome, what
// that outcome was, and when it was recorded.
type recruitmentCreditRecord struct {
	RecordID          string                      `json:"record_id"`
	ContributionID    string                      `json:"contribution_id"`
	ContributionKind  recruitmentContributionKind `json:"contribution_kind"`
	ChangedDecisionID string                      `json:"changed_decision_id,omitempty"`
	EffectEvidenceID  string                      `json:"effect_evidence_id,omitempty"`
	Outcome           recruitmentCreditOutcome    `json:"outcome"`
	RecordedAt        string                      `json:"recorded_at"`
}

// recruitmentCreditFile is the on-disk container at recruitmentCreditPath.
type recruitmentCreditFile struct {
	Entries []recruitmentCreditRecord `json:"entries"`
}

// recruitmentCreditRecordID is the deterministic identity key a credit
// record is stored and replayed under. One contribution changing one
// decision is one record, always -- writing the same (contributionID,
// changedDecisionID) pair twice is a replay, never a second record. A
// different contribution changing the SAME decision hashes to a different
// record ID, so both contributions' records coexist (see
// recordRecruitmentCredit and recruitmentCreditForDecision).
func recruitmentCreditRecordID(contributionID, changedDecisionID string) string {
	return fmt.Sprintf("credit:%s:%s", contributionID, changedDecisionID)
}

// errRecruitmentCreditNoChange is the internal replay sentinel
// recordRecruitmentCredit returns from its own UpdateJSONAtomically mutate
// closure to abort the write on a replay -- UpdateJSONAtomically's own
// contract is "if mutate returns an error, no write occurs" (pkg/storage),
// mirroring errRecruitmentResultAlreadyBound's role in bindRecruitmentResult
// (cmd/recruitment_result.go): exactly the "mutates nothing" guarantee a
// replay requires.
var errRecruitmentCreditNoChange = errors.New("recruitment credit record already exists")

// recordRecruitmentCredit is the ONE function in cmd/ that writes
// credit/records.json (TestCreditRequiresBothFacts enforces this by name,
// checking every write call site against this function's own signature).
// Its parameters carry both facts CEC-07 requires -- a changed decision
// identifier and an effect evidence identifier -- so no write this function
// performs can ever lose sight of either one:
//
//   - Neither identifier present: nothing is recorded at all. The
//     contribution earns no credit; a caller renders the existing
//     no-measured-effect wording (cmd/agency_contract.go's
//     AgencyMeasuredEffectPending) rather than a default or inferred
//     credit.
//   - A changed decision with no effect evidence yet: recorded as pending,
//     never as helpful, neutral, or harmful -- a decision having changed is
//     not yet proof of what it did.
//   - Effect evidence with no changed decision to attach it to: refused; an
//     outcome cannot exist without something it is the outcome OF.
//   - Both present: the outcome must be one of the three earnable, declared
//     values (helpful/neutral/harmful) -- an undeclared value, or pending
//     itself, is refused by name.
//
// A second call naming the SAME contribution and the SAME changed decision
// returns the first stored record and writes nothing (replay-safe, matching
// bindRecruitmentResult's own discipline). A different contribution against
// the SAME decision is a distinct record; recruitmentCreditForDecision
// returns both.
func recordRecruitmentCredit(contributionID string, kind recruitmentContributionKind, changedDecisionID, effectEvidenceID string, outcome recruitmentCreditOutcome, recordedAt string) (recruitmentCreditRecord, bool, error) {
	if store == nil {
		return recruitmentCreditRecord{}, false, fmt.Errorf("no store initialized")
	}
	contributionID = strings.TrimSpace(contributionID)
	if contributionID == "" {
		return recruitmentCreditRecord{}, false, fmt.Errorf("recruitment credit requires a non-empty contribution id")
	}
	if !recruitmentContributionKindDeclared(kind) {
		return recruitmentCreditRecord{}, false, fmt.Errorf(
			"recruitment credit contribution kind %q is not in the declared vocabulary %v", kind, recruitmentContributionKinds(),
		)
	}
	changedDecisionID = strings.TrimSpace(changedDecisionID)
	effectEvidenceID = strings.TrimSpace(effectEvidenceID)

	if changedDecisionID == "" && effectEvidenceID == "" {
		// Neither the decision this contribution changed nor the outcome
		// that followed is recorded -- earns no credit and no record at
		// all (CEC-07's own "never a default or inferred credit").
		return recruitmentCreditRecord{}, false, nil
	}
	if changedDecisionID == "" {
		return recruitmentCreditRecord{}, false, fmt.Errorf(
			"recruitment credit effect evidence %q has no changed decision to attach to", effectEvidenceID,
		)
	}

	resolvedOutcome := outcome
	if effectEvidenceID == "" {
		if outcome != "" && outcome != recruitmentCreditOutcomePending {
			return recruitmentCreditRecord{}, false, fmt.Errorf(
				"recruitment credit outcome %q requires effect evidence -- a changed decision alone is pending, never a default or inferred credit",
				outcome,
			)
		}
		resolvedOutcome = recruitmentCreditOutcomePending
	} else {
		if resolvedOutcome == recruitmentCreditOutcomePending {
			return recruitmentCreditRecord{}, false, fmt.Errorf(
				"recruitment credit has effect evidence %q but outcome %q -- an earned outcome must be helpful, neutral, or harmful",
				effectEvidenceID, resolvedOutcome,
			)
		}
		if !recruitmentCreditOutcomeDeclared(resolvedOutcome) {
			return recruitmentCreditRecord{}, false, fmt.Errorf(
				"recruitment credit outcome %q is not in the declared vocabulary %v", resolvedOutcome, recruitmentCreditOutcomeNames(),
			)
		}
	}

	recordedAt = strings.TrimSpace(recordedAt)
	if recordedAt == "" {
		recordedAt = time.Now().UTC().Format(time.RFC3339)
	}

	recordID := recruitmentCreditRecordID(contributionID, changedDecisionID)
	proposed := recruitmentCreditRecord{
		RecordID:          recordID,
		ContributionID:    contributionID,
		ContributionKind:  kind,
		ChangedDecisionID: changedDecisionID,
		EffectEvidenceID:  effectEvidenceID,
		Outcome:           resolvedOutcome,
		RecordedAt:        recordedAt,
	}

	var result recruitmentCreditRecord
	var file recruitmentCreditFile
	updateErr := store.UpdateJSONAtomically(recruitmentCreditPath, &file, func() error {
		for _, existing := range file.Entries {
			if existing.RecordID == recordID {
				result = existing
				return errRecruitmentCreditNoChange
			}
		}
		file.Entries = append(file.Entries, proposed)
		result = proposed
		return nil
	})
	if updateErr != nil && !errors.Is(updateErr, errRecruitmentCreditNoChange) {
		return recruitmentCreditRecord{}, false, updateErr
	}
	return result, true, nil
}

// sortRecruitmentCreditRecords orders records by recorded time descending,
// then by record identifier ascending on a tie -- an explicit, deterministic
// tie-break so repeated reads of the same underlying data always produce
// byte-identical output.
func sortRecruitmentCreditRecords(records []recruitmentCreditRecord) {
	sort.SliceStable(records, func(i, j int) bool {
		if records[i].RecordedAt != records[j].RecordedAt {
			return records[i].RecordedAt > records[j].RecordedAt
		}
		return records[i].RecordID < records[j].RecordID
	})
}

// recruitmentCreditAll returns every stored credit record, sorted by
// sortRecruitmentCreditRecords. Returns an empty slice (never an error) when
// no credit has ever been recorded.
func recruitmentCreditAll() ([]recruitmentCreditRecord, error) {
	if store == nil {
		return nil, fmt.Errorf("no store initialized")
	}
	var file recruitmentCreditFile
	if err := store.LoadJSON(recruitmentCreditPath, &file); err != nil {
		return []recruitmentCreditRecord{}, nil
	}
	records := append([]recruitmentCreditRecord{}, file.Entries...)
	sortRecruitmentCreditRecords(records)
	return records, nil
}

// recruitmentCreditForDecision returns every credit record naming decisionID
// as its ChangedDecisionID, in the same deterministic order recruitmentCreditAll
// uses. Two different contributions that each changed the same decision each
// get their own record, and both appear here.
func recruitmentCreditForDecision(decisionID string) ([]recruitmentCreditRecord, error) {
	decisionID = strings.TrimSpace(decisionID)
	if decisionID == "" {
		return []recruitmentCreditRecord{}, nil
	}
	all, err := recruitmentCreditAll()
	if err != nil {
		return nil, err
	}
	var matches []recruitmentCreditRecord
	for _, record := range all {
		if record.ChangedDecisionID == decisionID {
			matches = append(matches, record)
		}
	}
	if matches == nil {
		matches = []recruitmentCreditRecord{}
	}
	return matches, nil
}

// recruitmentCreditForContribution returns the single credit record for
// (contributionID, changedDecisionID), or ok=false if none exists yet.
func recruitmentCreditForContribution(contributionID, changedDecisionID string) (recruitmentCreditRecord, bool, error) {
	recordID := recruitmentCreditRecordID(strings.TrimSpace(contributionID), strings.TrimSpace(changedDecisionID))
	all, err := recruitmentCreditAll()
	if err != nil {
		return recruitmentCreditRecord{}, false, err
	}
	for _, record := range all {
		if record.RecordID == recordID {
			return record, true, nil
		}
	}
	return recruitmentCreditRecord{}, false, nil
}

// renderRecruitmentCreditOutcome renders a credit record's outcome: the
// existing no-measured-effect wording when record is nil (no record at all
// -- the contribution is uncredited), a distinct pending wording when a
// decision has changed with no verified outcome recorded yet, and the
// outcome name itself once genuinely earned.
func renderRecruitmentCreditOutcome(record *recruitmentCreditRecord) string {
	if record == nil {
		return AgencyMeasuredEffectPending
	}
	if record.Outcome == recruitmentCreditOutcomePending {
		return recruitmentCreditPendingWording
	}
	return string(record.Outcome)
}
