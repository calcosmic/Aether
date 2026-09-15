package cmd

import (
	"fmt"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// sealStateOfPlay is the short state-of-play card shown before seal ever
// asks "finish this project?" (D-04). It computes nothing new — every
// field is read from what checkSealBlockers/state already computed, never
// re-derived.
type sealStateOfPlay struct {
	PhasesCompleted int
	PhasesTotal     int
	// FailingChecks and OpenWarnings are plain-English descriptions, drawn
	// from checkSealBlockers's blockers and issues respectively.
	FailingChecks []string
	OpenWarnings  []string
}

// buildSealStateOfPlay assembles the card from data the seal flow already
// computed (checkSealBlockers's blockers/issues, the state's own phase
// list) — no new scan, no new store read.
func buildSealStateOfPlay(state colony.ColonyState, blockers, issues []colony.FlagEntry) sealStateOfPlay {
	card := sealStateOfPlay{
		PhasesCompleted: completedPhaseCount(state),
		PhasesTotal:     len(state.Plan.Phases),
	}
	for _, b := range blockers {
		if desc := strings.TrimSpace(b.Description); desc != "" {
			card.FailingChecks = append(card.FailingChecks, desc)
		}
	}
	for _, i := range issues {
		if desc := strings.TrimSpace(i.Description); desc != "" {
			card.OpenWarnings = append(card.OpenWarnings, desc)
		}
	}
	return card
}

// namedProblems is everything the second, explicit question must name
// (D-06) — currently the same universe checkSealBlockers's blockers and
// issues cover: anything still failing or unresolved at seal time.
func (c sealStateOfPlay) namedProblems() []string {
	if len(c.FailingChecks) == 0 && len(c.OpenWarnings) == 0 {
		return nil
	}
	problems := make([]string, 0, len(c.FailingChecks)+len(c.OpenWarnings))
	problems = append(problems, c.FailingChecks...)
	problems = append(problems, c.OpenWarnings...)
	return problems
}

// sealStateOfPlayCap caps how many failing checks/open warnings the card
// lists by name before folding the rest into an honest "(+N more)" — SEE-12's
// compact nested-detail convention.
const sealStateOfPlayCap = 3

// renderSealStateOfPlayCard renders the card in house style (SEE-12): an
// emoji heading with a plain-English parenthetical, one line per item,
// nested `└──` detail capped per category with an honest "(+N more)".
// Never a bordered table.
func renderSealStateOfPlayCard(card sealStateOfPlay) string {
	var b strings.Builder
	b.WriteString(renderStageMarker("Before You Finish"))
	b.WriteString("🏺 Finishing this project (writes a summary document, files the project away, and pools its lessons into the shared store other projects read)\n")
	b.WriteString(fmt.Sprintf("  - Phases done: %d of %d\n", card.PhasesCompleted, card.PhasesTotal))
	writeSealStateOfPlayCategory(&b, "Checks still failing", card.FailingChecks)
	writeSealStateOfPlayCategory(&b, "Open warnings", card.OpenWarnings)
	return b.String()
}

func writeSealStateOfPlayCategory(b *strings.Builder, label string, items []string) {
	if len(items) == 0 {
		b.WriteString(fmt.Sprintf("  - %s: none\n", label))
		return
	}
	b.WriteString(fmt.Sprintf("  - %s: %d\n", label, len(items)))
	for i, item := range items {
		if i >= sealStateOfPlayCap {
			b.WriteString(fmt.Sprintf("      └── (+%d more)\n", len(items)-sealStateOfPlayCap))
			return
		}
		if item = strings.TrimSpace(item); item != "" {
			b.WriteString(fmt.Sprintf("      └── %s\n", item))
		}
	}
}

// sealConfirmationQuestionText is the single, deterministic question text
// for a given set of named problems — the FIRST plain question when nothing
// is named, the SECOND, more specific question when something is (D-06:
// "Finish anyway with 2 checks failing?"). This exact text is both what is
// shown to the owner and what a recorded answer's stored question is
// matched against (loadSealConfirmationRecordedAnswer), so a match here is
// itself proof the recorded answer covers exactly this set of problems —
// never an inference from card wording.
func sealConfirmationQuestionText(namedProblems []string) string {
	if len(namedProblems) == 0 {
		return "Seal this verified colony and write its Crowned Anthill record? [y/N]"
	}
	return fmt.Sprintf("Force-seal this incomplete colony with %d unresolved item(s)? This records an owner override; it does not verify completion. [y/N]", len(namedProblems))
}

// sealConfirmationAnswerSource is "seal-force-confirmation" when the
// question names outstanding problems, "seal-confirmation" otherwise — the
// PendingDecision Source tag D-06 requires so a forced "finish anyway" is
// always distinguishable from a plain confirmation.
func sealConfirmationAnswerSource(namedProblems []string) string {
	if len(namedProblems) > 0 {
		return "seal-force-confirmation"
	}
	return "seal-confirmation"
}

// sealRecordedAnswer is an already-resolved fact about what the owner
// previously typed via `aether decision-answer`, computed by the caller
// from a stored PendingDecision — never a live store handle or a rendered
// string, so decideSealConfirmation itself stays pure (D-06).
type sealRecordedAnswer struct {
	// Answer is the owner's normalized (lowercased, trimmed) typed answer,
	// e.g. "yes" or "no".
	Answer string
	// CoversNamedProblems is true only when this recorded answer's stored
	// question text matched TODAY's exact confirmation question, which
	// embeds every named problem verbatim — an exact match is itself proof
	// the answer covers exactly this set of problems, never fewer or more.
	CoversNamedProblems bool
}

// loadSealConfirmationRecordedAnswer looks up the most recently resolved
// owner answer whose Source is a seal-confirmation source and reports
// whether its stored question text matches question exactly (normalized).
// Read-only; never mutates pending-decisions.json.
func loadSealConfirmationRecordedAnswer(s *storage.Store, question string) *sealRecordedAnswer {
	if s == nil {
		return nil
	}
	var file PendingDecisionFile
	if err := s.LoadJSON(pendingDecisionsFile, &file); err != nil {
		return nil
	}
	normalizedQuestion := normalizeDecisionText(question)
	var latest *sealRecordedAnswer
	var latestAt string
	for _, d := range file.Decisions {
		if !d.Resolved {
			continue
		}
		if d.Source != "seal-confirmation" && d.Source != "seal-force-confirmation" {
			continue
		}
		if d.ResolvedAt < latestAt {
			continue
		}
		recordedQuestion, _ := parseClarificationDescription(d.Description)
		latestAt = d.ResolvedAt
		latest = &sealRecordedAnswer{
			Answer:              strings.ToLower(strings.TrimSpace(d.Resolution)),
			CoversNamedProblems: normalizeDecisionText(recordedQuestion) == normalizedQuestion,
		}
	}
	return latest
}

// recordSealConfirmationAnswer stores the owner's answer to a seal
// confirmation question as a PendingDecision, reusing the exact same
// question-normalized matching every other clarification answer uses
// (cmd/handoff_decisions_cmd.go's recordDecisionAnswer/decisionAnswerCmd) —
// there is no seal-only flag; the owner runs the same
// `aether decision-answer` command a worker's open decision would.
func recordSealConfirmationAnswer(question, answer, source string) (PendingDecision, error) {
	return recordDecisionAnswer(question, answer, 0, source)
}

// sealConfirmationInput carries every already-resolved fact
// decideSealConfirmation's fixed-order policy needs. No field is a live
// store handle or a rendered string — every value is computed by the
// caller (sealCmd) before decideSealConfirmation is ever called.
type sealConfirmationInput struct {
	// Autopilot marks a non-interactive caller: D-07 requires such a caller
	// never proceed, regardless of any recorded answer. Research confirms
	// autopilot never reaches sealCmd at all (Task 3's reachability test);
	// this field exists so the policy itself enforces D-07 even if that
	// ever changes, rather than relying solely on unreachability.
	Autopilot bool
	// NamedProblems is the same list the state-of-play card names (D-06).
	NamedProblems []string
	// RecordedAnswer is nil when no matching recorded answer exists yet.
	RecordedAnswer *sealRecordedAnswer
}

// sealConfirmationDecision is decideSealConfirmation's total, structured
// verdict — mirrors decideBuildCheckin's shape (cmd/ceremony_team_checkin.go).
type sealConfirmationDecision struct {
	Proceed bool
	// Why is the plain-English reason — never repo jargon, safe to log or
	// show the owner as-is.
	Why string
	// AskSecondQuestion is true when NamedProblems is non-empty and the
	// caller must ask the more specific "finish anyway" question rather
	// than the plain one.
	AskSecondQuestion bool
}

// decideSealConfirmation is the pure D-04..D-07 policy, evaluated in one
// fixed order:
//
//  1. Autopilot (or any other non-interactive caller): never proceed, and
//     never treat the absence of an answer as a yes (D-07).
//  2. A recorded owner answer covering the named problems: proceed.
//  3. A recorded plain "yes" with named problems still outstanding, or any
//     recorded answer that is not "yes": do not proceed — ask again (the
//     second, more specific question when problems are named).
//  4. No recorded answer at all: do not proceed, ask the first question
//     (the second, when problems are named).
//
// Total and side-effect free: never reads a rendered card, never touches
// the store. The caller supplies every fact already resolved.
func decideSealConfirmation(input sealConfirmationInput) sealConfirmationDecision {
	problemsNamed := len(input.NamedProblems) > 0

	if input.Autopilot {
		return sealConfirmationDecision{
			Proceed:           false,
			Why:               "autopilot never finishes a project — it hands the finish command to the owner instead",
			AskSecondQuestion: problemsNamed,
		}
	}

	if input.RecordedAnswer != nil {
		answer := strings.ToLower(strings.TrimSpace(input.RecordedAnswer.Answer))
		if answer == "yes" && input.RecordedAnswer.CoversNamedProblems {
			return sealConfirmationDecision{Proceed: true, Why: "the owner's recorded answer covers everything currently named"}
		}
		if answer == "yes" {
			// A "yes" was recorded, but not to today's exact question — either
			// it answered a plain question while problems are now named, or
			// it answered a stale/different named-problems question. Either
			// way it does not cover what is outstanding right now.
			return sealConfirmationDecision{
				Proceed:           false,
				Why:               "the recorded yes did not name what is currently outstanding",
				AskSecondQuestion: problemsNamed,
			}
		}
		return sealConfirmationDecision{
			Proceed:           false,
			Why:               "the owner's recorded answer was not yes",
			AskSecondQuestion: problemsNamed,
		}
	}

	return sealConfirmationDecision{
		Proceed:           false,
		Why:               "no recorded answer yet — the owner has not been asked",
		AskSecondQuestion: problemsNamed,
	}
}

// renderSealConfirmationQuestionVisual renders the stop-and-ask moment in
// house style, reusing the same decision-block frame other stop points in
// this codebase use (circuit breaker trips, autopilot pauses, wave
// failures) — never a bordered table.
func renderSealConfirmationQuestionVisual(question, nextCommand string) string {
	return renderDecisionBlock("❓", "Finish This Project?", question, "Answer with: "+nextCommand) + "\n"
}

// renderSealPreflightConfirmationQuestionVisual is the honest Phase 199
// confirmation surface. It consumes the typed preflight so the forced branch
// cannot accidentally inherit the verified branch's success-oriented title.
func renderSealPreflightConfirmationQuestionVisual(preflight SealPreflight, nextCommand string) string {
	question := SealConfirmationCopy(preflight)
	if preflight.Disposition == colony.SealDispositionForcedIncomplete {
		return renderDecisionBlock("⛔", "Forced Incomplete Closure", question, "Answer with: "+nextCommand) + "\n"
	}
	return renderDecisionBlock("❓", "Verified Colony Closure", question, "Answer with: "+nextCommand) + "\n"
}

// sealConfirmationAnswerCommand is the exact command the owner runs to
// record an answer — the single source both the printed prose and the
// JSON result's "next" field use, so two surfaces can never name different
// commands (Phase 197 S-01).
func sealConfirmationAnswerCommand(question, source string) string {
	return fmt.Sprintf("aether decision-answer --question %q --answer \"yes\" --source %s", question, source)
}

// runSealConfirmationGate is the one D-04..D-07 stop-and-ask point every
// path to completeSealRuntime goes through — the interactive `aether seal`
// command's own RunE and the host-mediated `aether seal-finalize` path
// (runSealFinalize) alike (198-RESEARCH.md Pitfall 5: seal's default flow,
// unlike build/continue, IS host-mediated, so both entry points must share
// this gate rather than only the direct one). It prints the state-of-play
// card, runs the wisdom review (D-05, exactly once, before the question),
// and evaluates decideSealConfirmation. extraFailingChecks lets a caller
// (runSealFinalize) fold in problems it discovered beyond
// checkSealBlockers's own blockers/issues (its review workers' own
// blocking findings) into the same card and the same question.
//
// Returns the already-run review so a proceeding caller never re-runs it,
// and — when not proceeding — the exact result map the caller should
// output instead of sealing.
func runSealConfirmationGate(state colony.ColonyState, blockers, issues []colony.FlagEntry, extraFailingChecks ...string) (proceed bool, review sealWisdomReview, notProceedingResult map[string]interface{}) {
	card := buildSealStateOfPlay(state, blockers, issues)
	card.FailingChecks = append(card.FailingChecks, extraFailingChecks...)
	namedProblems := card.namedProblems()
	visualFprint(stdout, renderSealStateOfPlayCard(card))

	review = runSealWisdomReview(state)

	question := sealConfirmationQuestionText(namedProblems)
	recordedAnswer := loadSealConfirmationRecordedAnswer(store, question)
	decision := decideSealConfirmation(sealConfirmationInput{
		NamedProblems:  namedProblems,
		RecordedAnswer: recordedAnswer,
	})

	if decision.Proceed {
		return true, review, nil
	}

	answerSource := sealConfirmationAnswerSource(namedProblems)
	nextCommand := sealConfirmationAnswerCommand(question, answerSource)
	visualFprint(stdout, renderSealConfirmationQuestionVisual(question, nextCommand))
	return false, review, map[string]interface{}{
		"sealed":                      false,
		"awaiting_owner_confirmation": true,
		"named_problems":              namedProblems,
		"question":                    question,
		"next":                        nextCommand,
	}
}

// runSealPreflightConfirmationGate is the Phase 199 owner boundary. Unlike the
// compatibility gate above it performs no review or persistence before the
// answer: the typed preflight is the only input, and every later effect belongs
// to CommitSealTransaction.
//
// This gate has two callers with two different contracts. The direct,
// interactive `aether seal` command (cmd/codex_workflow_cmds.go) deliberately
// mixes this prose with its final JSON envelope on stdout — a real owner
// benefits from seeing it, and a machine caller of THAT command already
// extracts only the last JSON line (see cmd/seal_confirmation_test.go's
// lastJSONLine and its executeSealForConfirmationTest doc comment: "prose and
// the final JSON envelope together... after any prose emitted earlier in the
// same run" — a documented, tested contract this fix must not disturb).
// `aether seal-finalize` (cmd/seal_final_review.go) is different: it is a
// machine-only finalizer a wrapper parses as pure JSON (the exact field
// report this fixes — .planning/field-reports/2026-09-14-cosmic-seal-entomb-lifecycle.md,
// finding 4), and never mixes prose with its envelope in any mode.
//
// Both human writes below are therefore gated on isExplicitlyQuietCommand,
// the SAME "-finalize commands are quiet" classification
// streamingAllowedForCurrentCommand already consults for this command family
// (cmd/codex_visuals.go) — reused here rather than a new flag or environment
// variable — combined with shouldRenderVisualOutput(stdout), the same
// detection outputError/outputWorkflow already consult before choosing
// between a rendered card and a JSON envelope. A quiet (-finalize) command in
// machine-readable mode suppresses both writes entirely, matching the
// established pattern this package already uses for other human-only inline
// announcements (e.g. cmd/rollback.go's emitCanaryStarted/Completed/RolledBack):
// return before writing, rather than redirecting to stderr. Any other caller
// (the direct `aether seal` command, or a quiet command asked for human
// output) keeps writing exactly as it does today — this is a routing fix
// only, scoped to the one contract the field report actually names.
func runSealPreflightConfirmationGate(preflight SealPreflight) (bool, map[string]interface{}) {
	if sealPreflightGateShouldWriteHumanOutput() {
		visualFprint(stdout, renderSealPreflightCard(preflight))
	}
	question := SealConfirmationCopy(preflight)
	recordedAnswer := loadSealConfirmationRecordedAnswer(store, question)
	named := make([]string, 0, len(preflight.UnresolvedItems))
	for _, item := range preflight.UnresolvedItems {
		named = append(named, item.Summary)
	}
	decision := decideSealConfirmation(sealConfirmationInput{NamedProblems: named, RecordedAnswer: recordedAnswer})
	if decision.Proceed {
		return true, nil
	}
	source := "seal-confirmation"
	if preflight.Disposition == colony.SealDispositionForcedIncomplete {
		source = "seal-force-confirmation"
	}
	next := sealConfirmationAnswerCommand(question, source)
	if sealPreflightGateShouldWriteHumanOutput() {
		visualFprint(stdout, renderSealPreflightConfirmationQuestionVisual(preflight, next))
	}
	return false, map[string]interface{}{
		"sealed": false, "awaiting_owner_confirmation": true,
		"outcome_kind": preflight.OutcomeKind, "disposition": preflight.Disposition,
		"owner_reason": preflight.OwnerReason, "unresolved_items": preflight.UnresolvedItems,
		"question": question, "next": next,
	}
}

// sealPreflightGateShouldWriteHumanOutput decides whether
// runSealPreflightConfirmationGate's two human writes go to stdout. It is
// false only for the narrow case the field report actually names: a "quiet"
// (-finalize) invocation whose caller asked for machine-readable output.
// Every other combination — a quiet command asked for human output, or any
// non-quiet command (the direct `aether seal` path) in any mode — keeps
// writing, preserving that command's own documented, tested contract.
func sealPreflightGateShouldWriteHumanOutput() bool {
	if !isExplicitlyQuietCommand(currentStreamingCommand) {
		return true
	}
	return shouldRenderVisualOutput(stdout)
}

func renderSealPreflightCard(preflight SealPreflight) string {
	var b strings.Builder
	b.WriteString(renderStageMarker("Seal Preflight"))
	b.WriteString(fmt.Sprintf("Outcome: %s\n", preflight.OutcomeKind))
	b.WriteString(fmt.Sprintf("Completed phases: %d; completed tasks: %d\n", len(preflight.CompletedPhaseIDs), len(preflight.CompletedTaskIDs)))
	b.WriteString(fmt.Sprintf("Passed gates: %d; evidence records: %d; owner checkpoints: %d\n", len(preflight.PassedGates), len(preflight.Evidence), len(preflight.OwnerCheckpoints)))
	if len(preflight.UnresolvedItems) > 0 {
		b.WriteString(fmt.Sprintf("Unresolved items: %d\n", len(preflight.UnresolvedItems)))
		for _, item := range preflight.UnresolvedItems {
			b.WriteString("  - " + formatSealUnresolvedItem(item) + "\n")
		}
	}
	if len(preflight.ResidualRisks) > 0 {
		b.WriteString(fmt.Sprintf("Residual risks retained: %d\n", len(preflight.ResidualRisks)))
		for _, risk := range preflight.ResidualRisks {
			b.WriteString(fmt.Sprintf("  - [%s] %s\n", strings.TrimSpace(risk.ID), strings.TrimSpace(risk.Summary)))
		}
	}
	b.WriteString("Retained after closure: active state, receipt, evidence, findings, learnings, signals, checkpoints, and rollback record.\n")
	return b.String()
}
