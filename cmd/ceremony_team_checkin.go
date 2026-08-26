package cmd

import (
	"fmt"
	"sort"
	"strings"

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
