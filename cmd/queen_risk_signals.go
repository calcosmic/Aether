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
// Plan 194-02 shrank the old build-side floor (queenBuildSafetyRequiredCastes)
// to the builder alone and deleted queenBuildSafetyReviewRequired and
// queenPhaseHasSecuritySignal outright — this table is now the only place a
// build OR continue reviewer can be forced from. The temporary overlap
// 194-01-PLAN.md's "Known interim state" described (a security-signal phase
// drawing both the old implicit build reviewer and this table's forced
// reviewer) is closed as of that plan; see .planning/WINDOWS.md #1.

// riskSignal is one named high-risk vocabulary entry. Phrases are matched at
// a word boundary (matchesPhraseAtWordBoundary) against the phase's own
// wording (collectPhaseText); PathPatterns is matched as a lowercased,
// slash-normalised substring against the builder's own reported changed
// files (queenRiskSignalHitsFromPaths, D-02) — a second, independent
// detector that can only ADD a forced reviewer, never remove one. A signal
// with no PathPatterns never fires from the file detector.
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
	// WaiverReason is set only on a hit applyForcedReviewerWaivers
	// (cmd/forced_reviewer_waiver.go) has filtered out of the forced set --
	// the owner's own recorded reason for declining this signal's reviewer
	// (D-03), carried here so the check-in card can show it without a
	// second lookup.
	WaiverReason string
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
		// "auth/" (directory-anchored) replaced with the bare "auth"
		// (WR-03, 194-REVIEW.md): the old form only matched a file living
		// literally inside a directory named auth/, missing plausible
		// real files like auth.go, auth_service.go or auth-config.yaml at
		// any depth. Bare "auth" behaves the same as "session"/"credential"/
		// "secrets" below -- a boundary-matched substring
		// (matchesPathPatternAtBoundary) rather than a directory anchor --
		// so it is now consistent with its sibling patterns instead of the
		// odd one out. It still requires a non-letter/digit boundary on
		// both sides, so a name where "auth" is fused directly into a
		// longer identifier with no separator (authHandler.go,
		// authMiddleware.go, oauth.go) is not caught by this pattern alone
		// -- the same word/tokenizer trade-off this table already accepts
		// for "token" vs "tokenizer" in the phrase list above.
		PathPatterns: []string{"auth", "/login", "session", "credential", "secrets"},
		PlainEnglish: "logins and passwords",
	},
	{
		Name:  "payments",
		Caste: "gatekeeper",
		Phrases: []string{
			"payment", "payments", "billing", "checkout", "invoice",
			"refund", "credit card", "subscription billing", "payout",
		},
		PathPatterns: []string{"payment", "billing", "checkout", "stripe"},
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
		// No PathPatterns (D-01): no file path reliably means a release
		// gate, and a pattern that fires on the word "release" appearing in
		// a path would be a false alarm with no upside.
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
		PathPatterns: []string{"delete", "purge"},
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
		PathPatterns: []string{"migrations/", "migrate/", ".sql"},
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

// isPathWordByte draws PathPatterns' own boundary line (WR-02,
// 194-REVIEW.md), deliberately narrower than isWordByte's prose boundary: a
// path separator, underscore, hyphen, or dot must ALL count as a boundary
// here, so a bare pattern like "session" can never fire on a plain substring
// buried inside an unrelated word (the concrete false positive WR-02 named:
// "session" matching "repossession_handler.go" or "possession.go"). Only
// ASCII letters and digits count as "inside a word" for a path -- unlike
// isWordByte, which also treats "_" as a word character for prose
// identifiers.
func isPathWordByte(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

// matchesPathPatternAtBoundary requires a boundary on BOTH ends, the same
// discipline matchesPhraseAtWordBoundary already applies to the plan's own
// wording (WR-02) -- plain strings.Contains had no boundary check at all,
// so a bare word like "session" fired on any path containing that
// substring anywhere. path and pattern must already be lowercased/trimmed
// by the caller (queenRiskSignalHitsFromPaths already does both).
//
// A pattern that already ENDS or STARTS with its own separator character
// ("migrations/", "/login", ".sql") supplies its own boundary on that side:
// requiring ANOTHER non-word byte immediately past a literal "/" or "."
// that is already part of the pattern would reject a genuine match like
// "migrations/0007_add_column.sql" (the digit "0" right after the pattern's
// own trailing "/" is not a boundary violation -- the "/" already is one).
// Only a side of the pattern that ends in an alphanumeric character (a bare
// word like "session", or the "s" of ".sql") needs the adjacent path
// character checked.
func matchesPathPatternAtBoundary(path, pattern string) bool {
	if pattern == "" {
		return false
	}
	patternStartsWithSep := !isPathWordByte(pattern[0])
	patternEndsWithSep := !isPathWordByte(pattern[len(pattern)-1])
	for offset := 0; offset < len(path); {
		idx := strings.Index(path[offset:], pattern)
		if idx < 0 {
			return false
		}
		start := offset + idx
		end := start + len(pattern)
		leftOK := patternStartsWithSep || start == 0 || !isPathWordByte(path[start-1])
		rightOK := patternEndsWithSep || end == len(path) || !isPathWordByte(path[end])
		if leftOK && rightOK {
			return true
		}
		offset = start + 1
	}
	return false
}

// queenRiskSignalHitsFromPaths is D-02's second detector: it matches the
// builder's OWN reported changed files (phaseChangedFilesFromHandoffs)
// against the same five-signal table's PathPatterns, instead of the plan's
// wording. Same "longest match wins" discipline as queenRiskSignalHits, so
// the most specific pattern is the one quoted back -- compared and stored
// using the SAME trimmed/lowercased value (`p`) the match test itself runs
// against (IN-01, 194-REVIEW.md: comparing the untrimmed loop variable
// instead was a latent inconsistency with no live bug today only because
// every table entry already arrives pre-trimmed and lowercase). Source is
// always "changed files" so forcedReviewerReason can state where the hit
// came from rather than quoting wording the plan never contained. A signal
// with no PathPatterns (release sign-off) never fires here, by
// construction.
func queenRiskSignalHitsFromPaths(paths []string) []riskSignalHit {
	var hits []riskSignalHit
	for _, signal := range queenRiskSignalTable {
		if len(signal.PathPatterns) == 0 {
			continue
		}
		best := ""
		for _, rawPath := range paths {
			path := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(rawPath), "\\", "/"))
			if path == "" {
				continue
			}
			for _, pattern := range signal.PathPatterns {
				p := strings.ToLower(strings.TrimSpace(pattern))
				if p == "" {
					continue
				}
				if matchesPathPatternAtBoundary(path, p) && len(p) > len(best) {
					best = p
				}
			}
		}
		if best != "" {
			hits = append(hits, riskSignalHit{Signal: signal, Match: best, Source: "changed files"})
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
// signal that fired is named, and one clause per matching hit states where
// it came from (forcedReviewerReasonClause) — the plan's own wording, or a
// changed file the file detector matched (D-02, plan 194-06).
func forcedReviewerReason(caste string, hits []riskSignalHit) string {
	seenSignal := make(map[string]bool)
	seenClause := make(map[string]bool)
	var plainEnglish, clauses []string
	for _, hit := range hits {
		if !seenSignal[hit.Signal.Name] {
			seenSignal[hit.Signal.Name] = true
			plainEnglish = append(plainEnglish, hit.Signal.PlainEnglish)
		}
		if clause := forcedReviewerReasonClause(hit); clause != "" && !seenClause[clause] {
			seenClause[clause] = true
			clauses = append(clauses, clause)
		}
	}
	if len(plainEnglish) == 0 {
		return ""
	}
	return fmt.Sprintf("this touches %s (%s)", joinWithAnd(plainEnglish), joinWithAnd(clauses))
}

// forcedReviewerReasonClause names WHERE one hit came from — the plan's own
// wording quotes the matched phrase ("the plan mentions ..."); a changed
// file names the matched path pattern ("the files changed touched ..."),
// never a caste name or a score (D-09). This is the literal sentence-shape
// change D-02/plan 194-06 asked for: source reads "the files changed"
// rather than "the plan mentions" for a file-detected hit.
func forcedReviewerReasonClause(hit riskSignalHit) string {
	match := strings.TrimSpace(hit.Match)
	if match == "" {
		return ""
	}
	if hit.Source == "changed files" {
		return fmt.Sprintf("the files changed touched %q", match)
	}
	return fmt.Sprintf("the plan mentions %q", match)
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

// riskSignalHitsFromRecords reconstructs riskSignalHit values from a
// build-recorded forced-reviewer set (codexForcedReviewerRecord), by
// per-index name so queenForcedContinueReviewers can union the build's
// recorded hits with newly-detected changed-file hits through the SAME
// collapseToForcedReviewers merge the original build-time derivation used —
// one merge function, not two independent ones that could disagree.
// record.Signals[i]/Matches[i]/Sources[i] are aligned 1:1 by construction
// (forcedReviewerRecords copies them straight off collapseToForcedReviewers'
// own per-caste, per-signal slices).
func riskSignalHitsFromRecords(records []codexForcedReviewerRecord) []riskSignalHit {
	byName := make(map[string]riskSignal, len(queenRiskSignalTable))
	for _, signal := range queenRiskSignalTable {
		byName[signal.Name] = signal
	}
	var hits []riskSignalHit
	for _, record := range records {
		for i, name := range record.Signals {
			signal, ok := byName[name]
			if !ok {
				continue
			}
			match := ""
			if i < len(record.Matches) {
				match = record.Matches[i]
			}
			source := "plan wording"
			if i < len(record.Sources) {
				source = record.Sources[i]
			}
			hits = append(hits, riskSignalHit{Signal: signal, Match: match, Source: source})
		}
	}
	return hits
}

// queenForcedContinueReviewers is the CONTINUE-side read of the forced set
// (D-05: one derivation, one boundary — this closes .planning/WINDOWS.md #1's
// continue half). It starts from the build's recorded set when one exists —
// so a build-time derivation and a continue-time derivation of the same phase
// can never disagree — and falls back to re-deriving from the phase's own
// wording when no manifest record is present, so a continue-only run (no
// build this session) is still protected.
//
// changedFiles feeds queenRiskSignalHitsFromPaths (D-02, plan 194-06): its
// hits are UNIONED with the recorded/derived plan-wording hits through
// collapseToForcedReviewers, so the file detector can only ADD a caste (or
// add a signal to a caste's existing reason) — it can never remove a caste
// or a signal the build's record carried. See queenRiskSignalHitsFromPaths'
// own doc comment for why this direction is safe and the reverse is not.
func queenForcedContinueReviewers(phase colony.Phase, recorded []codexForcedReviewerRecord, changedFiles []string) []forcedReviewer {
	var hits []riskSignalHit
	if len(recorded) > 0 {
		hits = append(hits, riskSignalHitsFromRecords(recorded)...)
	} else {
		hits = append(hits, queenRiskSignalHits(collectPhaseText(phase), "plan wording")...)
	}
	hits = append(hits, queenRiskSignalHitsFromPaths(changedFiles)...)
	// D-03: the owner's waiver is applied here, at the ONE function both
	// continue lanes already go through, and BEFORE collapseToForcedReviewers
	// merges signals into castes -- filtering at the signal level is what
	// makes "one signal for one phase" true (a waived credentials signal
	// does not waive a live payments signal on the same caste). Because the
	// changed-file hits above are already unioned into hits by this point, a
	// signal waived from plan wording stays waived when the same signal is
	// re-detected from the files the builder changed.
	hits, _ = applyForcedReviewerWaivers(phase.ID, hits)
	if len(hits) == 0 {
		return nil
	}
	return collapseToForcedReviewers(hits)
}
