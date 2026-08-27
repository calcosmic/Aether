package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

// renderCeremonyTeamCheckinFromFile renders the pre-spawn team check-in card
// from a lifecycle manifest JSON file. The wrapper pauses on this card and
// asks the owner to proceed, trim optional workers, or redirect — the
// runtime plan itself (COLONY_STATE.json, the lifecycle manifest) is never
// touched by this render, and TestTeamCheckinDoesNotMutate pins that. It is
// no longer a pure inspection, though: CR-01 (194-REVIEW.md) added exactly
// one side effect -- for every LIVE forced reviewer this render shows, it
// writes a pending, unresolved decision-answer row
// (ensureForcedReviewerWaiverPendingDecision) recording that the runtime
// itself displayed this exact question to whoever is looking at the card.
// That row is what lets `aether decision-answer` refuse to waive a signal
// no card ever actually rendered. The write is idempotent (a repeat render
// of the same live hit is a no-op) and best-effort/non-blocking.
func renderCeremonyTeamCheckinFromFile(workflow, path string) (map[string]interface{}, string, error) {
	raw, err := readCeremonyJSONFile(path)
	if err != nil {
		return nil, "", err
	}
	manifest, _ := extractCeremonyManifest(raw)
	if len(manifest) == 0 {
		return nil, "", fmt.Errorf("manifest file %s does not contain a lifecycle manifest", path)
	}
	dispatches := ceremonyDispatchesFromManifest(manifest)
	result, visual := renderCeremonyTeamCheckin(normalizedCeremonyWorkflow(workflow), manifest, dispatches)
	result["manifest_file"] = path
	return result, visual, nil
}

func renderCeremonyTeamCheckin(workflow string, manifest map[string]interface{}, dispatches []ceremonyDispatch) (map[string]interface{}, string) {
	policy := mapValue(manifest["queen_execution_policy"])
	budget := mapValue(policy["spawn_budget"])
	requiredSet := stringSet(stringSliceValue(budget["required_castes"]))
	selectedReasons := ceremonyStringMapValue(budget["selected_reasons"])
	prunedReasons := ceremonyStringMapValue(budget["pruned_reasons"])

	// Unique castes in dispatch order; each shows once whatever its worker count.
	seen := map[string]int{}
	orderedCastes := []string{}
	for _, dispatch := range dispatches {
		caste := strings.TrimSpace(dispatch.Caste)
		if caste == "" {
			continue
		}
		if _, ok := seen[caste]; !ok {
			orderedCastes = append(orderedCastes, caste)
		}
		seen[caste]++
	}

	required := []string{}
	optional := []string{}
	reasons := map[string]string{}
	whatItDoes := map[string]string{}
	for _, caste := range orderedCastes {
		reason := strings.TrimSpace(selectedReasons[caste])
		if reason == "" {
			// D-09: the roster's generic "what this caste does" description is
			// NOT a reason a worker was sent for THIS phase — showing it here
			// would read as a justification and is not one. By the time plan
			// 194-03 has landed, every dispatched caste should already carry a
			// per-phase reason (its refusal loop and its runtime-written
			// reasons both guarantee this) — a caste reaching the card with
			// none is a bug upstream, so say so plainly rather than papering
			// over it.
			reason = "no reason was recorded for sending this worker"
		}
		reasons[caste] = reason
		if produces := casteRosterProduces(manifest, caste); produces != "" {
			whatItDoes[caste] = produces
		}
		if requiredSet[caste] {
			required = append(required, caste)
		} else {
			optional = append(optional, caste)
		}
	}

	var b strings.Builder
	b.WriteString(renderOldStyleCeremonyHeader(commandEmoji(emptyFallback(workflow, "team-checkin")), "Team Check-In"))
	b.WriteString("\n")
	// Layout separation (2026-08-23 owner feedback on plan 194-07): the card
	// was a wall of text with no visual break between workers or sections.
	// `renderStageMarker` is the same `── Title ──` rule already used for
	// build/continue stage markers (cmd/codex_visuals.go); reused here rather
	// than inventing a second separator style. No sentence below changed --
	// this is spacing only.
	b.WriteString(renderStageMarker("Team"))
	for i, caste := range orderedCastes {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("  ")
		b.WriteString(casteIdentityWithModel(caste))
		if requiredSet[caste] {
			b.WriteString("  REQUIRED")
		} else {
			b.WriteString("  OPTIONAL")
		}
		if count := seen[caste]; count > 1 {
			b.WriteString(fmt.Sprintf("  ×%d", count))
		}
		if reason := reasons[caste]; reason != "" {
			b.WriteString("  — ")
			b.WriteString(reason)
		}
		if produces := whatItDoes[caste]; produces != "" {
			b.WriteString("  (what it does: ")
			b.WriteString(produces)
			b.WriteString(")")
		}
		b.WriteString("\n")
	}
	phaseID := intValue(manifest["phase"])
	allForcedHits := riskSignalHitsFromRecords(forcedReviewerRecordsFromManifest(manifest))
	liveForcedHits, waivedForcedHits := applyForcedReviewerWaivers(phaseID, allForcedHits)

	if announcement := composeForcedReviewerAnnouncement(forcedReviewerRecords(collapseToForcedReviewers(liveForcedHits))); announcement != "" {
		b.WriteString("\n")
		b.WriteString(renderStageMarker("Required After The Work Is Done"))
		b.WriteString("Not sent with this team, but required at the check after the work is done:\n")
		for _, line := range strings.Split(announcement, "\n") {
			b.WriteString("  ")
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	// D-03: a live forced reviewer is shown with the exact command that
	// would decline it -- only the owner, pasting this command through
	// `aether decision-answer`, can waive it (T-194-11). A waived reviewer is
	// shown as declined, with the owner's own recorded reason, never
	// silently dropped from the card.
	waiveCommands := map[string]string{}
	attemptID := strings.TrimSpace(stringValue(manifest["attempt_id"]))
	for _, hit := range liveForcedHits {
		// CR-01 (194-REVIEW.md): record the runtime's OWN pending row for
		// this exact question the moment it renders a live forced reviewer
		// -- this is what lets decisionAnswerCmd (cmd/handoff_decisions_cmd.go)
		// refuse to waive a signal from a forged --question with no
		// matching row. Best-effort/non-blocking: a write failure here must
		// never break the (otherwise read-only) card render; the owner
		// simply cannot waive until a later render succeeds.
		capability, err := ensureForcedReviewerWaiverPendingDecision(phaseID, hit.Signal.Name, hit.Signal.PlainEnglish, attemptID)
		if err == nil && capability != "" {
			waiveCommands[hit.Signal.Name] = forcedReviewerWaiverCommand(phaseID, hit.Signal.Name, hit.Signal.PlainEnglish, capability)
		}
	}
	waived := map[string]interface{}{}
	for _, hit := range waivedForcedHits {
		waived[hit.Signal.Name] = map[string]interface{}{
			"plain_english": hit.Signal.PlainEnglish,
			"reason":        hit.WaiverReason,
		}
	}
	if len(waiveCommands) > 0 || len(waived) > 0 {
		b.WriteString("\n")
		b.WriteString(renderStageMarker("Decline A Required Reviewer"))
		b.WriteString("Only you can decline a required reviewer, with a reason on the record:\n")
		liveNames := make([]string, 0, len(waiveCommands))
		for name := range waiveCommands {
			liveNames = append(liveNames, name)
		}
		sort.Strings(liveNames)
		for _, name := range liveNames {
			b.WriteString("  To decline, run: ")
			b.WriteString(waiveCommands[name])
			b.WriteString("\n")
		}
		waivedNames := make([]string, 0, len(waived))
		for name := range waived {
			waivedNames = append(waivedNames, name)
		}
		sort.Strings(waivedNames)
		for _, name := range waivedNames {
			entry := mapValue(waived[name])
			b.WriteString("  Declined by owner: ")
			b.WriteString(stringValue(entry["reason"]))
			b.WriteString("\n")
		}
	}
	if len(prunedReasons) > 0 {
		prunedCastes := make([]string, 0, len(prunedReasons))
		for caste := range prunedReasons {
			prunedCastes = append(prunedCastes, caste)
		}
		sort.Strings(prunedCastes)
		b.WriteString("\n")
		b.WriteString(renderStageMarker("Not Sent"))
		b.WriteString("Not sent:\n")
		for _, caste := range prunedCastes {
			b.WriteString("  ")
			b.WriteString(casteLabel(caste))
			if reason := strings.TrimSpace(prunedReasons[caste]); reason != "" {
				b.WriteString(" — ")
				b.WriteString(reason)
			}
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(renderStageMarker("Summary"))
	b.WriteString("Required workers stay — they are the safety floor. Optional workers can be trimmed.\n")

	forcedRecords := forcedReviewerRecordsFromManifest(manifest)
	forced := map[string]interface{}{}
	for _, record := range forcedRecords {
		forced[record.Caste] = map[string]interface{}{
			"signals": record.Signals,
			"reason":  record.Reason,
		}
	}

	result := map[string]interface{}{
		"workflow":       workflow,
		"required":       required,
		"optional":       optional,
		"reasons":        reasons,
		"what_it_does":   whatItDoes,
		"pruned":         prunedReasons,
		"forced":         forced,
		"waived":         waived,
		"waive_commands": waiveCommands,
	}
	return result, b.String()
}

// forcedReviewerRecordsFromManifest reads the "forced_reviewers" field back
// out of a lifecycle manifest map (the JSON form of []codexForcedReviewerRecord,
// cmd/codex_build.go) so the card can render the exact set the build recorded
// (D-05) without re-deriving anything.
func forcedReviewerRecordsFromManifest(manifest map[string]interface{}) []codexForcedReviewerRecord {
	raw, ok := manifest["forced_reviewers"].([]interface{})
	if !ok {
		return nil
	}
	records := make([]codexForcedReviewerRecord, 0, len(raw))
	for _, entry := range raw {
		row := mapValue(entry)
		caste := strings.TrimSpace(stringValue(row["caste"]))
		if caste == "" {
			continue
		}
		records = append(records, codexForcedReviewerRecord{
			Caste:   caste,
			Signals: stringSliceValue(row["signals"]),
			Matches: stringSliceValue(row["matches"]),
			Sources: stringSliceValue(row["sources"]),
			Reason:  strings.TrimSpace(stringValue(row["reason"])),
		})
	}
	return records
}

// casteRosterProduces returns the roster's "produces" prose for a caste, the
// fallback reason when the spawn budget carried no per-caste rationale.
func casteRosterProduces(manifest map[string]interface{}, caste string) string {
	roster, ok := manifest["caste_roster"].([]interface{})
	if !ok {
		return ""
	}
	for _, entry := range roster {
		row := mapValue(entry)
		if strings.TrimSpace(stringValue(row["caste"])) == caste {
			return strings.TrimSpace(stringValue(row["produces"]))
		}
	}
	return ""
}

// ---------------------------------------------------------------------------
// One-worker fast path (Phase 195, D-11..D-14)
//
// The pre-Phase-195 rule was unconditional: every build paused for the
// owner's team check-in, whatever the team size, and the only opt-out was
// the explicit --no-checkin flag (TestOneWorkerTeamStillPauses). The owner's
// own 2026-08-23 feedback reversed that for the specific, narrow case where
// there is nothing left to decide: one implementation worker, no pending
// owner decision. decideBuildCheckin is the pure policy that replaces the
// old shortcut with a decision-aware one; buildHasPendingOwnerDecision is
// the single predicate it consults for "is there anything left to ask" so
// nothing downstream can reintroduce len(dispatches)==1 as its own decision
// point.
// ---------------------------------------------------------------------------

// buildCheckinReasonCode is the machine-readable reason decideBuildCheckin
// returns alongside its Requested verdict. Wrappers and tests key off this
// value rather than parsing rendered prose.
type buildCheckinReasonCode string

const (
	// buildCheckinReasonNonInteractive covers autopilot and explicit
	// --no-checkin: the pause never renders, whatever else is true.
	buildCheckinReasonNonInteractive buildCheckinReasonCode = "non_interactive"
	// buildCheckinReasonExplicitCheckin is D-14's owner override: --checkin
	// always forces the pause on an otherwise-eligible one-worker build.
	buildCheckinReasonExplicitCheckin buildCheckinReasonCode = "explicit_checkin"
	// buildCheckinReasonPendingOwnerDecision is D-13: a live forced-reviewer
	// waiver, an unanswered orchestrator boundary question, or another
	// persisted unanswered owner decision keeps the pause even for one
	// worker.
	buildCheckinReasonPendingOwnerDecision buildCheckinReasonCode = "pending_owner_decision"
	// buildCheckinReasonOneWorkerFastPath is D-11: exactly one implementation
	// dispatch and nothing else pending takes the automatic fast path.
	buildCheckinReasonOneWorkerFastPath buildCheckinReasonCode = "one_worker_fast_path"
	// buildCheckinReasonDefaultPause is the unchanged default for every team
	// size and situation the rows above do not cover.
	buildCheckinReasonDefaultPause buildCheckinReasonCode = "default_pause"
)

// buildCheckinDecisionInput is every fact decideBuildCheckin needs, already
// normalized by the caller. It never reads the store or a manifest itself --
// that keeps the policy pure and trivially table-testable.
type buildCheckinDecisionInput struct {
	// Autopilot marks a non-interactive caller (e.g. an autopilot lane) that
	// never renders a check-in card, independent of any flag. No current
	// caller sets this true (autopilot does not call this policy at all),
	// but the precedence table names it explicitly (Pattern 7,
	// 195-RESEARCH.md) so a future non-interactive caller has a place to
	// plug in rather than inventing a second bypass.
	Autopilot bool
	// NoCheckin is the existing explicit --no-checkin flag.
	NoCheckin bool
	// Checkin is the new explicit --checkin flag (D-14). Rejecting
	// Checkin && NoCheckin together is the caller's job, before this
	// function is ever called (see codex_workflow_cmds.go) -- a conflict
	// must never reach a pure policy silently resolved one way.
	Checkin bool
	// PendingOwnerDecision is buildHasPendingOwnerDecision's verdict for this
	// exact phase, already computed against live runtime records.
	PendingOwnerDecision bool
	// PendingOwnerDecisionWhy is the plain-English reason to surface when
	// PendingOwnerDecision is true.
	PendingOwnerDecisionWhy string
	// ImplementationDispatches is the number of build workers this phase's
	// final coherent-job plan actually dispatches -- len(dispatches), not a
	// task count. A grouped job covering many tasks still counts as one.
	ImplementationDispatches int
}

// buildCheckinDecision is decideBuildCheckin's total, structured verdict.
type buildCheckinDecision struct {
	// Requested mirrors the existing result["checkin_requested"] contract:
	// true means the wrapper still renders the full pre-spawn check-in card
	// and pauses; false means it does not.
	Requested bool
	// Reason is the machine-readable code a wrapper or test can branch on.
	Reason buildCheckinReasonCode
	// Why is the plain-English sentence explaining Reason -- never repo
	// jargon, safe to show the owner as-is.
	Why string
}

// decideBuildCheckin is the pure D-11..D-14 policy. Evaluated in one fixed
// order, matching 195-RESEARCH.md's Pattern 7 precedence exactly:
//
//  1. Autopilot or explicit --no-checkin: stay non-interactive.
//  2. Explicit --checkin: force the pause.
//  3. Any live pending owner decision: force the pause, even for one worker.
//  4. Exactly one implementation dispatch and nothing else pending: fast path.
//  5. Everything else: pause, unchanged from pre-Phase-195 behavior.
//
// Total and side-effect free -- never touches the store, never infers
// pending state from rendered text. The caller (codex_workflow_cmds.go)
// supplies every input already resolved from live records.
func decideBuildCheckin(input buildCheckinDecisionInput) buildCheckinDecision {
	if input.Autopilot || input.NoCheckin {
		return buildCheckinDecision{
			Requested: false,
			Reason:    buildCheckinReasonNonInteractive,
			Why:       "autopilot and --no-checkin never pause for a check-in",
		}
	}
	if input.Checkin {
		return buildCheckinDecision{
			Requested: true,
			Reason:    buildCheckinReasonExplicitCheckin,
			Why:       "the owner asked to see the check-in with --checkin",
		}
	}
	if input.PendingOwnerDecision {
		why := strings.TrimSpace(input.PendingOwnerDecisionWhy)
		if why == "" {
			why = "a pending owner decision keeps the check-in"
		}
		return buildCheckinDecision{
			Requested: true,
			Reason:    buildCheckinReasonPendingOwnerDecision,
			Why:       why,
		}
	}
	if input.ImplementationDispatches == 1 {
		return buildCheckinDecision{
			Requested: false,
			Reason:    buildCheckinReasonOneWorkerFastPath,
			Why:       "no owner decision is pending, so dispatch continues",
		}
	}
	return buildCheckinDecision{
		Requested: true,
		Reason:    buildCheckinReasonDefaultPause,
		Why:       "more than one worker is dispatched, so the owner reviews the team before it spawns",
	}
}

// buildHasPendingOwnerDecision is the single predicate decideBuildCheckin
// consults for "is there anything left for the owner to decide" (D-13). It
// reads three live runtime records, never rendered prose:
//
//  1. An unwaived forced-reviewer signal (the same derivation the full
//     check-in card already renders and the same waiver Phase 194 built --
//     D-13 explicitly requires this stay live even when build itself has
//     only one Builder, because the owner's ONE opportunity to see this
//     card is right here).
//  2. An unanswered orchestrator boundary question for this phase's build
//     (materializeOrchestratorBoundaryQuestions; empty for every colony not
//     in orchestrator mode, so this is a true no-op there).
//  3. Any other persisted, unanswered owner decision already consumed at
//     the build boundary -- a worker's open_decisions handoff question the
//     owner has not yet answered (pendingHandoffDecisions), the same
//     surface `aether handoff-decisions` exposes.
//
// Returns the first true predicate's plain-English reason; callers needing
// only the boolean can ignore the second value.
func buildHasPendingOwnerDecision(manifest codexBuildManifest) (bool, string) {
	allForcedHits := riskSignalHitsFromRecords(manifest.ForcedReviewers)
	liveForcedHits, _ := applyForcedReviewerWaivers(manifest.Phase, allForcedHits)
	if len(liveForcedHits) > 0 {
		return true, "a forced reviewer is still waiting for the owner's check-in decision"
	}
	if manifest.BoundaryQuestionCount > 0 {
		return true, "an unanswered planning question is still waiting for the owner"
	}
	if len(pendingHandoffDecisions(manifest.Phase)) > 0 {
		return true, "a worker left a question only the owner can answer"
	}
	return false, ""
}

// splitCoherentJobReason recovers the relationship and benefit halves of a
// job_reason string built by structuredCoherentJobReason
// (cmd/coherent_jobs.go), which always joins them as "<relationship>, so
// <benefit>". That function is the ONLY writer of this shape anywhere in the
// codebase, so splitting on its exact separator is safe; a reason with no
// separator (or empty) returns it whole as the relationship with an empty
// benefit rather than guessing.
func splitCoherentJobReason(reason string) (relationship, benefit string) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "", ""
	}
	const sep = ", so "
	if idx := strings.Index(reason, sep); idx >= 0 {
		return strings.TrimSpace(reason[:idx]), strings.TrimSpace(reason[idx+len(sep):])
	}
	return reason, ""
}

// renderBuildFastPathSummary composes D-12's compact, non-blocking summary
// for a one-worker build that takes decideBuildCheckin's automatic fast
// path. It is a distinct renderer from renderCeremonyTeamCheckin -- the
// fast path never calls the full card and discards its question; it never
// asks anything at all. The result carries the same facts the visual text
// states: the worker, every covered task, the accepted relationship and
// benefit (or the honest single-task explanation), and why no approval is
// required.
func renderBuildFastPathSummary(phase colony.Phase, dispatch codexBuildDispatch, decision buildCheckinDecision) (map[string]interface{}, string) {
	coveredIDs := dispatchCoveredTaskIDs(dispatch)
	coveredTasks := findDispatchTasks(phase, dispatch)

	taskLines := make([]string, 0, len(coveredTasks))
	for _, covered := range coveredTasks {
		label := covered.ID
		if covered.Task != nil {
			if goal := strings.TrimSpace(covered.Task.Goal); goal != "" {
				label = fmt.Sprintf("%s (%s)", covered.ID, goal)
			}
		}
		taskLines = append(taskLines, label)
	}
	if len(taskLines) == 0 {
		taskLines = append(taskLines, coveredIDs...)
	}

	// D-12: "a single-task job uses an honest single-task reason rather than
	// an empty grouping slot." A dispatch this policy ever calls with is
	// either a genuine multi-task coherent job (JobSource "queen" or
	// "automatic", CoveredTaskIDs has >1 entry) or a single task (JobSource
	// "single", or no job metadata at all for a pre-wave dispatch) -- never
	// an in-between case, so len(coveredIDs) <= 1 is the exact and only
	// single-task signal.
	singleTask := len(coveredIDs) <= 1
	var relationship, benefit string
	if singleTask {
		relationship = "one worker owns this one task"
		benefit = "no grouping was needed"
	} else {
		relationship, benefit = splitCoherentJobReason(dispatch.JobReason)
		if relationship == "" {
			relationship = "these tasks were grouped into one job"
		}
		if benefit == "" {
			benefit = "one worker avoids repeated setup and write conflicts"
		}
	}

	worker := fmt.Sprintf("%s %s", casteIdentity(dispatch.Caste), emptyFallback(dispatch.Name, "worker"))
	whyNoApproval := strings.TrimSpace(decision.Why)
	if whyNoApproval == "" {
		whyNoApproval = "no owner decision is pending, so dispatch continues"
	}

	result := map[string]interface{}{
		"worker":           worker,
		"caste":            dispatch.Caste,
		"covered_task_ids": append([]string{}, coveredIDs...),
		"covered_tasks":    taskLines,
		"relationship":     relationship,
		"benefit":          benefit,
		"single_task":      singleTask,
		"checkin_reason":   string(decision.Reason),
		"why_no_approval":  whyNoApproval,
	}

	var b strings.Builder
	b.WriteString(renderStageMarker("One Worker, No Approval Needed"))
	b.WriteString("  ")
	b.WriteString(worker)
	b.WriteString("\n")
	b.WriteString("  Covers: ")
	b.WriteString(strings.Join(taskLines, "; "))
	b.WriteString("\n")
	if singleTask {
		b.WriteString("  One worker owns this one task -- no grouping was needed.\n")
	} else {
		b.WriteString("  Why one job: ")
		b.WriteString(relationship)
		b.WriteString(", so ")
		b.WriteString(benefit)
		b.WriteString(".\n")
	}
	b.WriteString("  ")
	b.WriteString(sentenceCase(whyNoApproval))
	b.WriteString(" -- no approval is needed.\n")

	return result, b.String()
}

// sentenceCase upper-cases the first character of an owner-facing sentence.
func sentenceCase(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

var ceremonyTeamCheckinCmd = &cobra.Command{
	Use:   "team-checkin",
	Short: "Render the pre-spawn team check-in card from a lifecycle manifest JSON file",
	RunE: func(cmd *cobra.Command, args []string) error {
		result, visual, err := renderCeremonyTeamCheckinFromFile(ceremonyFlags.Workflow, ceremonyFlags.ManifestFile)
		if err != nil {
			return err
		}
		outputWorkflow(result, visual)
		return nil
	},
}
