package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

func sealScreenOutcome(result map[string]interface{}, state colony.ColonyState) *colony.SealOutcome {
	if outcome, ok := result["seal_outcome"].(colony.SealOutcome); ok {
		return &outcome
	}
	if raw, ok := result["seal_outcome"].(map[string]interface{}); ok {
		data, err := json.Marshal(raw)
		var outcome colony.SealOutcome
		if err == nil && json.Unmarshal(data, &outcome) == nil {
			return &outcome
		}
		return nil
	}
	return state.SealOutcome
}

func sealDeliveryLines(readiness string) []string {
	var lines, versions []string
	distinct := map[string]bool{}
	for _, line := range screenTextLines(planningStripANSI(readiness)) {
		label, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		if strings.Contains(strings.ToLower(label), "version") {
			versions = append(versions, voiceLine("warning", line))
			if fields := strings.Fields(value); len(fields) > 0 {
				distinct[fields[0]] = true
			}
		} else if label == "Git status" && value != "clean" {
			value = strings.ReplaceAll(value, "uncommitted changes", "changes not saved to version history")
			lines = append(lines, voiceLine("warning", value))
		}
	}
	if len(distinct) > 1 || distinct["unknown"] {
		lines = append(versions, lines...)
	}
	return lines
}

func sealScreenVerdict(outcome *colony.SealOutcome, goal string) string {
	if outcome == nil {
		return renderVerdictBox("warning", "PROJECT CLOSURE NOT VERIFIED", "No verified finish was recorded. Review the project status.")
	}
	if outcome.Disposition == colony.SealDispositionForcedIncomplete || outcome.OutcomeKind == colony.OutcomeKindForcedIncompleteClosure {
		verdict := fmt.Sprintf("PROJECT CLOSED EARLY — %s unfinished", screenCount(len(outcome.IncompletePhases), "phase"))
		if len(outcome.IncompletePhases) == 0 {
			verdict = "PROJECT CLOSED EARLY — COMPLETION NOT VERIFIED"
		}
		return renderVerdictBox("warning", verdict, "Completion was not verified. "+strings.TrimSpace(outcome.OwnerReason))
	}
	if outcome.Disposition != colony.SealDispositionVerified || len(outcome.IncompletePhases) > 0 {
		return renderVerdictBox("warning", "PROJECT CLOSURE NOT VERIFIED", "The saved records do not confirm a completed project.")
	}
	sentence := "The project is finished and checked."
	if len(outcome.CompletedPhases) > 0 {
		sentence = fmt.Sprintf("All %s are finished and checked.", screenCount(len(outcome.CompletedPhases), "phase"))
	}
	if words := strings.Fields(goal); len(words) > 0 {
		if len(words) > 12 {
			sentence = strings.Join(words[:12], " ") + "…"
		} else {
			sentence = strings.Join(words, " ")
		}
	}
	return renderVerdictBox("finished", "PROJECT SEALED — FINISHED", sentence)
}

func sealScreenReviewLines(workers []map[string]interface{}) []string {
	var lines []string
	type note struct {
		text string
		rank int
	}
	var notes []note
	counts := map[string]int{}
	blocking, finished := 0, 0
	for _, worker := range workers {
		kind, status, failed := screenWorkerStatus(emptyFallback(stringValue(worker["status"]), stringValue(worker["result_status"])))
		if kind == "done" {
			finished++
		}
		if failed || kind != "done" {
			lines = append(lines, voiceLine("warning", emptyFallback(stringValue(worker["name"]), "Reviewer")+" — "+status+". "+stringValue(worker["summary"])))
		}
		for _, f := range mapSliceValue(worker["findings"]) {
			severity := strings.ToLower(stringValue(f["severity"]))
			rank := 3
			switch severity {
			case "critical", "high":
				severity, rank = "serious", 1
			case "medium":
				rank = 2
			default:
				severity = "small"
			}
			counts[severity]++
			if boolValue(f["blocking"]) {
				blocking++
				rank = 0
			}
			notes = append(notes, note{emptyFallback(stringValue(f["description"]), stringValue(f["summary"])), rank})
		}
	}
	if len(workers) == 0 {
		return []string{voiceLine("evidence", "No final reviewer results were recorded.")}
	}
	if finished == len(workers) {
		lines = append(lines, voiceLine("done", screenCount(finished, "reviewer")+" finished the final review."))
	}
	var countWords []string
	for _, severity := range []string{"serious", "medium", "small"} {
		if counts[severity] > 0 {
			countWords = append(countWords, fmt.Sprintf("%d %s", counts[severity], severity))
		}
	}
	if len(notes) == 0 {
		lines = append(lines, voiceLine("evidence", "No findings were reported."))
	} else {
		ending, kind := ", none blocking", "evidence"
		if blocking > 0 {
			ending, kind = fmt.Sprintf(", %d blocking", blocking), "blocked"
		}
		lines = append(lines, voiceLine(kind, strings.Join(countWords, " and ")+" notes"+ending))
	}
	sort.SliceStable(notes, func(i, j int) bool { return notes[i].rank < notes[j].rank })
	var findings []string
	for _, note := range notes {
		findings = append(findings, voiceLine("flag", note.text))
	}
	return append(lines, capScreenLines(findings, 3)...)
}

func renderSealCeremonyScreen(result map[string]interface{}) string {
	var b strings.Builder
	state, err := loadActiveColonyStateReadOnly()
	var outcome *colony.SealOutcome
	if err == nil {
		outcome = sealScreenOutcome(mapValue(result["completion_raw"]), state)
	}
	if boolValue(result["completion_finalizer_failed"]) || boolValue(result["completion_path_blocked"]) {
		b.WriteString(renderVerdictBox("blocked", "PROJECT NOT SEALED", emptyFallback(stringValue(result["completion_error_message"]), "The final step stopped before the project was sealed.")))
	} else {
		b.WriteString(sealScreenVerdict(outcome, stringValue(result["goal"])))
	}
	writeScreenSection(&b, checkScreenSectionEmoji("safety"), "Final Review", sealScreenReviewLines(mapSliceValue(result["completion_workers"])))
	writeScreenSection(&b, commandEmoji("porter"), "Delivery", sealDeliveryLines(stringValue(result["porter_readiness"])))
	b.WriteString("\n  Full detail: `aether status --detail`\n")
	b.WriteString(renderLifecycleClosing(result, "seal"))
	return b.String()
}
