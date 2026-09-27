package cmd

import (
	"fmt"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

func planScreenHeading(phases []colony.Phase, sentence string) string {
	return renderVerdictBox("plan", "PLAN READY — "+strings.ToUpper(screenCount(len(phases), "phase")), sentence)
}

func writePlanScreenPhases(b *strings.Builder, phases []colony.Phase) {
	var lines []string
	for _, phase := range phases {
		line := fmt.Sprintf("Phase %d — %s", phase.ID, phase.Name)
		if n := len(phase.Tasks); n > 0 {
			line += " (" + screenCount(n, "task") + ")"
		}
		lines = append(lines, voiceLine("phase", line))
	}
	writeScreenSection(b, commandEmoji("phase"), "The Phases", capScreenLines(lines, 8))
}

func approvedPlanSentence(phases []colony.Phase) string {
	for _, phase := range phases {
		if phase.Status != colony.PhaseCompleted {
			return fmt.Sprintf("Approved. Next: `aether build %d`.", phase.ID)
		}
	}
	return "Approved. All planned phases are complete."
}

func renderPlanVisual(result map[string]interface{}) string {
	if visual, ok := renderCanonicalPlanningResult(result, planningVisualOptions{Width: lifecycleStatusOutputWidth()}); ok {
		return visual
	}
	// Intermediate planning and repair screens carry dispatch instructions,
	// not a plan to approve. Keep those contracts until there is a final plan.
	if _, repair := result["repair_source"]; repair || boolValue(result["requires_next_iteration"]) || (boolValue(result["plan_only"]) && !boolValue(result["existing_plan"])) {
		return renderPlanDetail(result)
	}
	phases := phaseSliceValue(result["phases"])
	if len(phases) == 0 {
		return renderPlanDetail(result)
	}
	var b strings.Builder
	var concerns []string
	for _, worker := range mapSliceValue(result["dispatches"]) {
		_, status, failed := screenWorkerStatus(stringValue(worker["status"]))
		if failed {
			concerns = append(concerns, voiceLine("warning", stringValue(worker["name"])+" "+status+": "+stringValue(worker["summary"])))
		}
	}
	if len(concerns) > 0 {
		b.WriteString(renderVerdictBox("warning", "PLAN NEEDS ATTENTION", "Some planning helpers could not finish. Review the saved phases and concerns below."))
	} else {
		sentence := approvedPlanSentence(phases)
		if boolValue(result["existing_plan"]) {
			sentence = "Existing plan loaded. " + sentence
		}
		b.WriteString(planScreenHeading(phases, sentence))
	}
	writePlanScreenPhases(&b, phases)
	if count := intValue(result["unresolved_clarifications"]); count > 0 {
		concerns = append(concerns, voiceLine("question", fmt.Sprintf("%d unresolved questions; use `aether discuss` to answer them.", count)))
	}
	// Failed research and other warnings are shown before ordinary assumptions.
	for _, key := range []string{"planning_warning", "research_warning", "clarification_warning"} {
		if warning := strings.TrimSpace(stringValue(result[key])); warning != "" {
			concerns = append(concerns, voiceLine("warning", warning))
		}
	}
	for _, gap := range stringSliceValue(result["gaps"]) {
		concerns = append(concerns, voiceLine("question", gap))
	}
	writeScreenSection(&b, commandEmoji("oracle"), "What It Rests On", capScreenLines(concerns, 3))
	b.WriteString("\n  Full detail: `aether phase` or `AETHER_OUTPUT_MODE=json aether plan`.\n")
	b.WriteString(renderLifecycleClosing(result, "plan"))
	return b.String()
}

func renderPlanningCandidateVisual(review planCandidateReview, options planningVisualOptions) string {
	if options.Detail {
		return renderPlanningCandidateDetail(review, options)
	}
	projection := projectPlanningCandidate(review)
	phases := review.Candidate.Proposal.Phases
	var b strings.Builder
	switch {
	case projection.CandidateActive:
		b.WriteString(planScreenHeading(phases, approvedPlanSentence(phases)))
	case projection.AcceptanceAvailable:
		b.WriteString(planScreenHeading(phases, "Review it, then approve it."))
	default:
		b.WriteString(renderVerdictBox("warning", "PLAN NEEDS ATTENTION", planScreenRefusalReason(projection.WhyUnavailable)))
	}
	writePlanScreenPhases(&b, phases)
	var concerns []string
	for _, gap := range projection.ResidualGaps {
		concerns = append(concerns, voiceLine("question", gap.Description))
	}
	if projection.RecommendationRationale != "" && projection.RecommendationDisposition != "accept" {
		concerns = append(concerns, voiceLine("decision", projection.RecommendationRationale))
	}
	writeScreenSection(&b, commandEmoji("oracle"), "What It Rests On", capScreenLines(concerns, 3))
	b.WriteString("\n  Full detail: `AETHER_OUTPUT_MODE=json aether plan --candidate`.\n\n")
	if projection.CandidateActive {
		b.WriteString(voiceLine("done", "Owner acceptance is recorded; build and run are equal execution choices.") + "\n")
		b.WriteString(voiceLine("next", fmt.Sprintf("Next: %s", projection.Next)) + "\n")
	} else if projection.AcceptanceAvailable {
		b.WriteString(voiceLine("decision", "Accept this candidate?") + "\n")
		b.WriteString(voiceLine("decision", fmt.Sprintf("Acceptance command: %s", projection.AcceptanceCommand)) + "\n")
		b.WriteString(voiceLine("next", fmt.Sprintf("Next: %s", projection.Next)) + "\n")
	} else {
		b.WriteString(voiceLine("avoid", fmt.Sprintf("Why unavailable: %s", planScreenRefusalReason(projection.WhyUnavailable))) + "\n")
		b.WriteString(voiceLine("next", fmt.Sprintf("Next: %s", projection.Next)) + "\n")
	}
	return finalizePlanningVisual(b.String(), options)
}

func renderPlanningAcceptanceVisual(candidate colony.PlanCandidate, revision colony.PlanRevision, receipt colony.PlanAcceptanceReceipt, replayed bool, options planningVisualOptions) string {
	if options.Detail {
		return renderPlanningAcceptanceDetail(candidate, revision, receipt, replayed, options)
	}
	var b strings.Builder
	sentence := approvedPlanSentence(revision.Phases)
	if replayed {
		sentence = "Already approved. The saved plan is unchanged."
	}
	b.WriteString(planScreenHeading(revision.Phases, sentence))
	writePlanScreenPhases(&b, revision.Phases)
	b.WriteString("\n  Full detail: `aether phase` or `AETHER_OUTPUT_MODE=json aether plan --candidate`.\n\n")
	b.WriteString(voiceLine("next", "Next Up: choose an operating mode") + "\n")
	b.WriteString(voiceLine("alternative", "aether build") + "\n")
	b.WriteString(voiceLine("alternative", "aether run") + "\n")
	return finalizePlanningVisual(b.String(), options)
}

func planScreenRefusalReason(reason string) string {
	switch reason {
	case "candidate_expired":
		return "This proposed plan has expired. Refresh it before approving."
	case "specification_changed":
		return "The agreed requirements changed after this plan was prepared."
	case "base_plan_changed":
		return "The active plan changed after this proposal was prepared."
	case "proposal_changed", "candidate_body_changed":
		return "The saved proposal no longer matches the reviewed copy."
	case "timeline_changed", "planning_stage_changed":
		return "The planning record changed. Refresh the proposal before approving."
	case "candidate_lifetime_invalid", "clock_before_candidate_creation", "acceptance_time_invalid":
		return "The saved planning dates do not agree. Review the project details."
	case "acceptance_receipt_invalid":
		return "The saved approval could not be verified."
	case "candidate_status_changed":
		return "This proposal is no longer available for approval."
	default:
		return "This proposal cannot be approved yet. Review the project details."
	}
}
