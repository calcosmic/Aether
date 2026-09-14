package cmd

// LEARN-01 (204-02-PLAN.md Task 2, ruling (b)/SYN-204-01 in
// 204-CLASSIC-SYNTHESIS.md): one shared, status-aware vocabulary every
// render or admission path that claims learning content is "verified" or
// "confirmed" must call. The previous phase's synthesis found the real
// status vocabulary and the real filter machinery already existed and were
// already correct (pkg/learn/colony_store.go's Status field/EntryFilter) --
// the live defect was that neither of the two call sites this study named
// (cmd/colony_prime_context.go's render filter, and
// cmd/autopilot_lessons.go's admission predicate) ever applied it. This
// file is the one place that decision is made now, so a worker is never
// shown a guess under a heading that calls it proven.

import (
	"strings"

	"github.com/calcosmic/Aether/pkg/learn"
)

// learnedMemoryVerifiedHeading is the exact wording the worker capsule
// already used for its learned-memory section (cmd/colony_prime_context.go),
// moved here unchanged so its current text lives in exactly one place.
const learnedMemoryVerifiedHeading = "## LEARNED MEMORY (Verified Outcomes)\n\n"

// learnedMemoryUnverifiedHeading names, in the owner's and the worker's own
// language, that these are observations that have not been checked yet --
// never a claim of verification for content this project's own status
// vocabulary marks a hypothesis.
const learnedMemoryUnverifiedHeading = "## LEARNED MEMORY (Not Yet Verified -- Unchecked Observations)\n\n"

// learningEntryStatusEquals compares an entry's Status field against want,
// case-insensitively after trimming -- the same normalisation
// confirmedAutopilotLessonsSincePlan already applied inline before this file
// existed (cmd/autopilot_lessons.go).
func learningEntryStatusEquals(status, want string) bool {
	return strings.EqualFold(strings.TrimSpace(status), want)
}

// learningVerifiedEntries keeps only entries whose Status is genuinely
// validated. This is the ONE predicate every render or admission path that
// claims "verified" or "confirmed" learning content must call -- a second,
// independent predicate anywhere else in this package is refused by name by
// TestOneLearningStatusVocabulary (cmd/learning_status_vocabulary_test.go).
func learningVerifiedEntries(entries []learn.Entry) []learn.Entry {
	verified := make([]learn.Entry, 0, len(entries))
	for _, entry := range entries {
		if learningEntryStatusEquals(entry.Status, learn.StatusValidated) {
			verified = append(verified, entry)
		}
	}
	return verified
}

// learningUnverifiedEntries keeps entries whose Status is a hypothesis, or
// entirely empty -- a legacy record written before the status field existed
// is not evidence of validation, so it is treated as unverified rather than
// silently promoted to verified. A disproven entry belongs under NEITHER
// this function's result NOR learningVerifiedEntries's -- excluded from
// both, on every render path, rather than shown anywhere.
func learningUnverifiedEntries(entries []learn.Entry) []learn.Entry {
	unverified := make([]learn.Entry, 0, len(entries))
	for _, entry := range entries {
		status := strings.ToLower(strings.TrimSpace(entry.Status))
		if status == "" || learningEntryStatusEquals(entry.Status, learn.StatusHypothesis) {
			unverified = append(unverified, entry)
		}
	}
	return unverified
}
