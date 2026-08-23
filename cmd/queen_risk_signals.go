package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

// This file holds the entire named-risk vocabulary the program uses to force
// a reviewer onto a phase (D-01..D-05,
// .planning/phases/194-the-queen-decides-the-team/194-CONTEXT.md). Before
// this existed, "force a reviewer" was three independent implicit rules
// scattered across queen_spawn_budget.go and caste_relevance.go —
// "production mode", "high risk", "security wording" — each with its own
// threshold and none of them explaining itself to the owner. This table is
// now the ONLY place a reviewer can be forced from: every firing names its
// signal and quotes the phrase that matched, so a wrong entry is visible the
// first time it fires (D-01's reversibility argument).
//
// This plan is additive: it does not touch queenBuildSafetyRequiredCastes,
// queenBuildSafetyReviewRequired or queenPhaseHasSecuritySignal (the OLD
// build-side floor). Those are plan 194-02's job. Until that plan lands, a
// security-signal phase draws BOTH the old implicit build reviewer and this
// new forced reviewer at the checking step — a known, deliberate, temporary
// overlap (see 194-01-PLAN.md "Known interim state").

// riskSignal is one named high-risk vocabulary entry. Phrases are matched at
// a word boundary (matchesPhraseAtWordBoundary) against the phase's own
// wording (collectPhaseText); PathPatterns is declared here but left empty —
// plan 194-06 fills it with changed-file glob patterns that can only ADD a
// forced reviewer, never remove one.
type riskSignal struct {
	Name         string
	Caste        string
	Phrases      []string
	PathPatterns []string
	PlainEnglish string
}

// riskSignalHit is one matched phrase for one signal, carrying the literal
// text that matched so forcedReviewerReason can quote it back to the owner.
type riskSignalHit struct {
	Signal riskSignal
	Match  string
	Source string
}

// forcedReviewer is one caste forced onto the checking step by one or more
// named risk signals, already collapsed so a phase never pays for the same
// caste twice even when two signals of the same class both fire (D-04).
type forcedReviewer struct {
	Caste   string
	Signals []string
	Matches []string
	Sources []string
	Reason  string
}

// queenRiskSignalTable is the whole named-risk vocabulary (D-01). Exactly
// five entries, no more — TestReviewerForcedOnlyByNamedRisk fails if a row is
// added, removed or misrouted. Do not add a sixth entry without an owner
// ruling; see 194-CONTEXT.md D-01.
//
// The lone word "token" is deliberately excluded — "token bucket rate
// limiter" is the known false alarm named in D-02. Use "access token" and
// "api key" instead.
var queenRiskSignalTable = []riskSignal{
	{
		Name:  "credentials/auth",
		Caste: "gatekeeper",
		Phrases: []string{
			"password", "passwords", "password reset", "login", "log in",
			"sign in", "sign-in", "credential", "credentials",
			"authentication", "authenticate", "session handling",
			"session cookie", "access token", "api key", "api keys",
			"secret key", "secrets", "oauth", "sso", "2fa", "mfa",
		},
		PlainEnglish: "logins and passwords",
	},
	{
		Name:  "payments",
		Caste: "gatekeeper",
		Phrases: []string{
			"payment", "payments", "billing", "checkout", "invoice",
			"refund", "credit card", "subscription billing", "payout",
		},
		PlainEnglish: "money",
	},
	{
		Name:  "release sign-off",
		Caste: "gatekeeper",
		Phrases: []string{
			"release", "release candidate", "sign off", "sign-off",
			"signoff", "final review", "deploy to production",
			"ship the release",
		},
		PlainEnglish: "signing off a release",
	},
	{
		Name:  "data deletion",
		Caste: "auditor",
		Phrases: []string{
			"delete data", "delete account", "delete accounts",
			"delete stale", "data deletion", "hard delete", "purge",
			"drop table", "erase", "wipe",
		},
		PlainEnglish: "deleting data",
	},
	{
		Name:  "database migration",
		Caste: "auditor",
		Phrases: []string{
			"migration", "migrations", "schema change", "schema changes",
			"alter table", "add a column", "drop column",
			"database structure", "db migration",
		},
		PlainEnglish: "changing the database structure",
	},
}

// matchesPhraseAtWordBoundary reuses the byte test in isWordByte
// (cmd/caste_relevance.go) but, unlike containsKeyword, requires a boundary
// on BOTH ends: this signal table's phrases must never fire on a word that
// merely contains one ("tokenizer" must never match "token", and "migrations"
// must not spuriously match a plain substring inside an unrelated word). Case
// folding matches collectPhaseText, which already lowercases its output.
func matchesPhraseAtWordBoundary(text, phrase string) bool {
	phrase = strings.ToLower(strings.TrimSpace(phrase))
	if phrase == "" {
		return false
	}
	text = strings.ToLower(text)
	for offset := 0; offset < len(text); {
		idx := strings.Index(text[offset:], phrase)
		if idx < 0 {
			return false
		}
		start := offset + idx
		end := start + len(phrase)
		leftOK := start == 0 || !isWordByte(text[start-1])
		rightOK := end == len(text) || !isWordByte(text[end])
		if leftOK && rightOK {
			return true
		}
		offset = start + 1
	}
	return false
}

// queenRiskSignalHits returns one hit per matching signal — the LONGEST
// matching phrase wins, not the first, so a specific phrase like "password
// reset" is quoted back rather than the shorter "password" it contains
// (Go's map/slice iteration and the phrase list's authoring order must never
// change which phrase gets quoted). source records where the text came from
// ("plan wording" today; plan 194-06 adds "changed files").
func queenRiskSignalHits(text, source string) []riskSignalHit {
	var hits []riskSignalHit
	for _, signal := range queenRiskSignalTable {
		best := ""
		for _, phrase := range signal.Phrases {
			if matchesPhraseAtWordBoundary(text, phrase) && len(phrase) > len(best) {
				best = phrase
			}
		}
		if best != "" {
			hits = append(hits, riskSignalHit{Signal: signal, Match: best, Source: source})
		}
	}
	return hits
}

// collapseToForcedReviewers returns ONE forcedReviewer per caste, folding in
// every signal that named it. This is the rule that stops a phase mentioning
// both a refund and a login from paying for two security reviewers (D-04) —
// the must_haves collapse rule and TestTwoSignalsOneCasteCollapseToOneDispatch.
func collapseToForcedReviewers(hits []riskSignalHit) []forcedReviewer {
	byCaste := make(map[string][]riskSignalHit)
	for _, hit := range hits {
		byCaste[hit.Signal.Caste] = append(byCaste[hit.Signal.Caste], hit)
	}
	castes := make([]string, 0, len(byCaste))
	for caste := range byCaste {
		castes = append(castes, caste)
	}
	sort.Strings(castes)

	reviewers := make([]forcedReviewer, 0, len(castes))
	for _, caste := range castes {
		casteHits := append([]riskSignalHit(nil), byCaste[caste]...)
		sort.Slice(casteHits, func(i, j int) bool {
			if casteHits[i].Signal.Name != casteHits[j].Signal.Name {
				return casteHits[i].Signal.Name < casteHits[j].Signal.Name
			}
			return casteHits[i].Match < casteHits[j].Match
		})

		var signals, matches, sources []string
		seen := make(map[string]bool)
		for _, hit := range casteHits {
			if seen[hit.Signal.Name] {
				continue
			}
			seen[hit.Signal.Name] = true
			signals = append(signals, hit.Signal.Name)
			matches = append(matches, hit.Match)
			sources = append(sources, hit.Source)
		}

		reviewers = append(reviewers, forcedReviewer{
			Caste:   caste,
			Signals: signals,
			Matches: matches,
			Sources: sources,
			Reason:  forcedReviewerReason(caste, casteHits),
		})
	}
	return reviewers
}

// forcedReviewerReason produces the one plain-English sentence the owner
// reads: "this touches logins and passwords (the plan mentions "password
// reset")". No caste names, no signal identifiers, no scores (D-09) — every
// signal that fired is named, and one matched phrase per signal is quoted.
func forcedReviewerReason(caste string, hits []riskSignalHit) string {
	seen := make(map[string]bool)
	var plainEnglish, quoted []string
	for _, hit := range hits {
		if seen[hit.Signal.Name] {
			continue
		}
		seen[hit.Signal.Name] = true
		plainEnglish = append(plainEnglish, hit.Signal.PlainEnglish)
		quoted = append(quoted, fmt.Sprintf("%q", hit.Match))
	}
	if len(plainEnglish) == 0 {
		return ""
	}
	return fmt.Sprintf("this touches %s (the plan mentions %s)", joinWithAnd(plainEnglish), joinWithAnd(quoted))
}

// joinWithAnd (cmd/codex_project_docs.go) already renders a list the way a
// sentence needs it: "a", "a and b", "a, b, and c" — reused here rather than
// a second copy, so the forced-reviewer reason reads like a sentence instead
// of a bare comma-joined list (CLAUDE.md's plain-English mandate).

// queenForcedReviewersForPhase derives the forced-reviewer set from the
// phase's own wording (D-02's plan-wording detector: collectPhaseText — name,
// description, tasks). This is the single derivation the build manifest
// records (codexForcedReviewerRecord, cmd/codex_build.go) and the fallback
// queenForcedContinueReviewers uses when no build record exists.
func queenForcedReviewersForPhase(phase colony.Phase) []forcedReviewer {
	return collapseToForcedReviewers(queenRiskSignalHits(collectPhaseText(phase), "plan wording"))
}

// forcedReviewerRecords converts the in-memory forcedReviewer set into the
// durable codexForcedReviewerRecord shape the build manifest carries
// (cmd/codex_build.go). Kept next to the derivation it serializes rather than
// beside the manifest struct, so the two stay obviously in sync.
func forcedReviewerRecords(reviewers []forcedReviewer) []codexForcedReviewerRecord {
	if len(reviewers) == 0 {
		return nil
	}
	records := make([]codexForcedReviewerRecord, 0, len(reviewers))
	for _, reviewer := range reviewers {
		records = append(records, codexForcedReviewerRecord{
			Caste:   reviewer.Caste,
			Signals: reviewer.Signals,
			Matches: reviewer.Matches,
			Sources: reviewer.Sources,
			Reason:  reviewer.Reason,
		})
	}
	return records
}

// queenForcedContinueReviewers is the CONTINUE-side read of the forced set
// (D-05: one derivation, one boundary — this closes .planning/WINDOWS.md #1's
// continue half). It starts from the build's recorded set when one exists —
// so a build-time derivation and a continue-time derivation of the same phase
// can never disagree — and falls back to re-deriving from the phase's own
// wording when no manifest record is present, so a continue-only run (no
// build this session) is still protected.
//
// changedFiles is accepted now and ignored: plan 194-06 fills PathPatterns
// and wires this parameter to a second detector that can only ADD a forced
// reviewer, never remove one (D-02).
func queenForcedContinueReviewers(phase colony.Phase, recorded []codexForcedReviewerRecord, changedFiles []string) []forcedReviewer {
	if len(recorded) == 0 {
		return queenForcedReviewersForPhase(phase)
	}
	reviewers := make([]forcedReviewer, 0, len(recorded))
	for _, record := range recorded {
		reviewers = append(reviewers, forcedReviewer{
			Caste:   record.Caste,
			Signals: append([]string(nil), record.Signals...),
			Matches: append([]string(nil), record.Matches...),
			Sources: append([]string(nil), record.Sources...),
			Reason:  record.Reason,
		})
	}
	sort.Slice(reviewers, func(i, j int) bool { return reviewers[i].Caste < reviewers[j].Caste })
	return reviewers
}
