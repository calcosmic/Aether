package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/trace"
)

// TraceDiagnostic holds the result of analyzing a trace export.
type TraceDiagnostic struct {
	RunID       string               `json:"run_id"`
	EntryCount  int                  `json:"entry_count"`
	Duration    string               `json:"duration"`
	Timeline    []StateTransition    `json:"timeline,omitempty"`
	ErrorGroups []ErrorCluster       `json:"error_groups,omitempty"`
	StateGaps   []StateGap           `json:"state_gaps,omitempty"`
	Stalled     []StalledPhase       `json:"stalled_phases,omitempty"`
	TokenTotals TokenTotals          `json:"token_totals"`
	Suggestions []string             `json:"suggestions,omitempty"`
	HealthScore int                  `json:"health_score"` // 0-100
}

// StateTransition represents a single state change in the timeline.
type StateTransition struct {
	Timestamp string `json:"timestamp"`
	From      string `json:"from"`
	To        string `json:"to"`
	Source    string `json:"source,omitempty"`
}

// ErrorCluster groups errors that occurred in the same phase.
type ErrorCluster struct {
	Phase      int      `json:"phase"`
	Count      int      `json:"count"`
	Severities []string `json:"severities"`
	ErrorIDs   []string `json:"error_ids,omitempty"`
}

// StateGap represents a missing expected state transition.
type StateGap struct {
	After    string `json:"after"`
	Missing  string `json:"missing"`
	Phase    int    `json:"phase,omitempty"`
	Message  string `json:"message"`
}

// StalledPhase represents a phase with no activity for an extended period.
type StalledPhase struct {
	Phase    int    `json:"phase"`
	LastSeen string `json:"last_seen"`
	StallAge string `json:"stall_age"`
}

// TokenTotals holds aggregated token usage.
type TokenTotals struct {
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	TotalCost    float64 `json:"total_usd_cost"`
	Entries      int     `json:"entries"`
}

// Expected state transition sequence for a healthy colony run.
var expectedTransitions = []struct{ from, to string }{
	{"initialized", "planning"},
	{"planning", "building"},
	{"building", "verifying"},
	{"verifying", "active"},
	{"active", "building"},
	{"active", "sealed"},
}

// analyzeTraceDiagnostics processes trace entries and produces a diagnostic report.
func analyzeTraceDiagnostics(entries []trace.TraceEntry) TraceDiagnostic {
	diag := TraceDiagnostic{
		EntryCount: len(entries),
	}

	if len(entries) == 0 {
		diag.HealthScore = 100
		return diag
	}

	// Extract run_id from first entry
	diag.RunID = entries[0].RunID

	// Calculate duration
	var firstTime, lastTime time.Time
	for _, e := range entries {
		t, err := time.Parse(time.RFC3339, e.Timestamp)
		if err != nil {
			continue
		}
		if firstTime.IsZero() || t.Before(firstTime) {
			firstTime = t
		}
		if lastTime.IsZero() || t.After(lastTime) {
			lastTime = t
		}
	}
	if !firstTime.IsZero() && !lastTime.IsZero() {
		diag.Duration = lastTime.Sub(firstTime).String()
	}

	// Reconstruct timeline
	diag.Timeline = extractTimeline(entries)

	// Error clusters
	diag.ErrorGroups = findErrorClusters(entries)

	// State gaps
	diag.StateGaps = findStateGaps(entries)

	// Stalled phases
	diag.Stalled = findStalledPhases(entries)

	// Token totals
	diag.TokenTotals = sumTokens(entries)

	// Generate suggestions
	diag.Suggestions = generateDiagnosticSuggestions(diag)

	// Health score
	diag.HealthScore = computeHealthScore(diag)

	return diag
}

// extractTimeline reconstructs state transitions from trace entries.
func extractTimeline(entries []trace.TraceEntry) []StateTransition {
	var timeline []StateTransition
	for _, e := range entries {
		if e.Level == trace.TraceLevelState && e.Topic == "state.transition" {
			from, _ := e.Payload["from"].(string)
			to, _ := e.Payload["to"].(string)
			if from != "" || to != "" {
				timeline = append(timeline, StateTransition{
					Timestamp: e.Timestamp,
					From:      from,
					To:        to,
					Source:    e.Source,
				})
			}
		}
	}
	return timeline
}

// findErrorClusters identifies phases with 3+ errors.
func findErrorClusters(entries []trace.TraceEntry) []ErrorCluster {
	phaseErrors := map[int]*ErrorCluster{}

	for _, e := range entries {
		if e.Level != trace.TraceLevelError {
			continue
		}
		phaseNum := 0
		if pn, ok := e.Payload["phase"].(float64); ok {
			phaseNum = int(pn)
		}
		sev, _ := e.Payload["severity"].(string)
		errID, _ := e.Payload["error_id"].(string)

		cluster, exists := phaseErrors[phaseNum]
		if !exists {
			cluster = &ErrorCluster{Phase: phaseNum}
			phaseErrors[phaseNum] = cluster
		}
		cluster.Count++
		if sev != "" {
			cluster.Severities = append(cluster.Severities, sev)
		}
		if errID != "" {
			cluster.ErrorIDs = append(cluster.ErrorIDs, errID)
		}
	}

	var clusters []ErrorCluster
	for _, c := range phaseErrors {
		if c.Count >= 3 {
			clusters = append(clusters, *c)
		}
	}
	if len(clusters) == 0 && len(phaseErrors) > 0 {
		// Include all error phases even if below threshold
		for _, c := range phaseErrors {
			clusters = append(clusters, *c)
		}
	}
	return clusters
}

// findStateGaps detects missing expected transitions.
func findStateGaps(entries []trace.TraceEntry) []StateGap {
	transitions := extractTimeline(entries)
	if len(transitions) == 0 {
		return nil
	}

	transitionSet := map[string]bool{}
	for _, t := range transitions {
		key := t.From + "->" + t.To
		transitionSet[key] = true
	}

	var gaps []StateGap
	for _, expected := range expectedTransitions {
		key := expected.from + "->" + expected.to
		if !transitionSet[key] {
			// Only flag if the "from" state was reached but "to" was not
			fromReached := false
			for _, t := range transitions {
				if t.To == expected.from || t.From == expected.from {
					fromReached = true
					break
				}
			}
			if fromReached {
				gaps = append(gaps, StateGap{
					After:   expected.from,
					Missing: expected.to,
					Message: fmt.Sprintf("State '%s' reached but no transition to '%s' recorded", expected.from, expected.to),
				})
			}
		}
	}
	return gaps
}

// findStalledPhases detects phases with no activity for >30 minutes.
func findStalledPhases(entries []trace.TraceEntry) []StalledPhase {
	// Find last activity per phase
	phaseLastActivity := map[int]time.Time{}
	var latestOverall time.Time

	for _, e := range entries {
		t, err := time.Parse(time.RFC3339, e.Timestamp)
		if err != nil {
			continue
		}
		if latestOverall.IsZero() || t.After(latestOverall) {
			latestOverall = t
		}
		if e.Level == trace.TraceLevelPhase {
			if pn, ok := e.Payload["phase"].(float64); ok {
				phase := int(pn)
				if lastTime, exists := phaseLastActivity[phase]; !exists || t.After(lastTime) {
					phaseLastActivity[phase] = t
				}
			}
		}
	}

	if latestOverall.IsZero() {
		return nil
	}

	var stalled []StalledPhase
	stallThreshold := 30 * time.Minute

	for phase, lastSeen := range phaseLastActivity {
		gap := latestOverall.Sub(lastSeen)
		if gap > stallThreshold {
			stalled = append(stalled, StalledPhase{
				Phase:    phase,
				LastSeen: lastSeen.Format(time.RFC3339),
				StallAge: gap.String(),
			})
		}
	}
	return stalled
}

// sumTokens aggregates token usage from all token entries.
func sumTokens(entries []trace.TraceEntry) TokenTotals {
	var totals TokenTotals
	for _, e := range entries {
		if e.Level != trace.TraceLevelToken {
			continue
		}
		totals.Entries++
		if it, ok := e.Payload["input_tokens"].(float64); ok {
			totals.InputTokens += int64(it)
		}
		if ot, ok := e.Payload["output_tokens"].(float64); ok {
			totals.OutputTokens += int64(ot)
		}
		if c, ok := e.Payload["usd_cost"].(float64); ok {
			totals.TotalCost += c
		}
	}
	return totals
}

// generateDiagnosticSuggestions produces actionable fix suggestions.
func generateDiagnosticSuggestions(diag TraceDiagnostic) []string {
	var suggestions []string

	for _, cluster := range diag.ErrorGroups {
		if cluster.Count >= 5 {
			suggestions = append(suggestions,
				fmt.Sprintf("Phase %d has %d errors — investigate root cause before retrying", cluster.Phase, cluster.Count))
		} else if cluster.Count >= 3 {
			suggestions = append(suggestions,
				fmt.Sprintf("Phase %d has %d errors — may indicate a systematic issue", cluster.Phase, cluster.Count))
		}
	}

	for _, gap := range diag.StateGaps {
		suggestions = append(suggestions,
			fmt.Sprintf("Missing transition to '%s' after '%s' — run may be incomplete", gap.Missing, gap.After))
	}

	for _, s := range diag.Stalled {
		suggestions = append(suggestions,
			fmt.Sprintf("Phase %d stalled for %s — check if worker is blocked", s.Phase, s.StallAge))
	}

	if diag.TokenTotals.TotalCost > 10.0 {
		suggestions = append(suggestions,
			fmt.Sprintf("High token cost ($%.2f) — consider using compact mode", diag.TokenTotals.TotalCost))
	}

	return suggestions
}

// computeHealthScore calculates a 0-100 score based on diagnostic findings.
func computeHealthScore(diag TraceDiagnostic) int {
	score := 100

	// Deduct for error clusters
	for _, cluster := range diag.ErrorGroups {
		if cluster.Count >= 5 {
			score -= 20
		} else if cluster.Count >= 3 {
			score -= 10
		} else {
			score -= 3
		}
	}

	// Deduct for state gaps
	score -= len(diag.StateGaps) * 10

	// Deduct for stalled phases
	score -= len(diag.Stalled) * 5

	if score < 0 {
		score = 0
	}
	return score
}

// renderTraceDiagnostic produces a human-readable report.
func renderTraceDiagnostic(diag TraceDiagnostic) string {
	var b strings.Builder

	b.WriteString(renderBanner(commandEmoji("medic"), "Trace Diagnostics"))
	b.WriteString(visualDividerStr())

	b.WriteString(renderStageMarker("Summary"))
	b.WriteString(fmt.Sprintf("Run ID:     %s\n", diag.RunID))
	b.WriteString(fmt.Sprintf("Entries:    %d\n", diag.EntryCount))
	b.WriteString(fmt.Sprintf("Duration:   %s\n", diag.Duration))
	b.WriteString(fmt.Sprintf("Health:     %d/100\n\n", diag.HealthScore))

	if len(diag.Timeline) > 0 {
		b.WriteString(renderStageMarker("State Timeline"))
		for _, t := range diag.Timeline {
			b.WriteString(fmt.Sprintf("  %s  %s → %s", t.Timestamp, t.From, t.To))
			if t.Source != "" {
				b.WriteString(fmt.Sprintf("  (%s)", t.Source))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	if len(diag.ErrorGroups) > 0 {
		b.WriteString(renderStageMarker("Error Clusters"))
		for _, cluster := range diag.ErrorGroups {
			b.WriteString(fmt.Sprintf("  Phase %d: %d errors", cluster.Phase, cluster.Count))
			if len(cluster.Severities) > 0 {
				b.WriteString(fmt.Sprintf(" (%s)", strings.Join(uniqueStrings(cluster.Severities), ", ")))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	if len(diag.StateGaps) > 0 {
		b.WriteString(renderStageMarker("State Gaps"))
		for _, gap := range diag.StateGaps {
			b.WriteString(fmt.Sprintf("  %s\n", gap.Message))
		}
		b.WriteString("\n")
	}

	if len(diag.Stalled) > 0 {
		b.WriteString(renderStageMarker("Stalled Phases"))
		for _, s := range diag.Stalled {
			b.WriteString(fmt.Sprintf("  Phase %d: last activity %s ago\n", s.Phase, s.StallAge))
		}
		b.WriteString("\n")
	}

	if diag.TokenTotals.Entries > 0 {
		b.WriteString(renderStageMarker("Token Usage"))
		b.WriteString(fmt.Sprintf("  Input:  %d tokens\n", diag.TokenTotals.InputTokens))
		b.WriteString(fmt.Sprintf("  Output: %d tokens\n", diag.TokenTotals.OutputTokens))
		b.WriteString(fmt.Sprintf("  Cost:   $%.4f\n\n", diag.TokenTotals.TotalCost))
	}

	if len(diag.Suggestions) > 0 {
		b.WriteString(renderStageMarker("Suggestions"))
		for _, s := range diag.Suggestions {
			b.WriteString(fmt.Sprintf("  - %s\n", s))
		}
		b.WriteString("\n")
	}

	if len(diag.Suggestions) == 0 {
		b.WriteString(renderStageMarker("Next Steps"))
		b.WriteString("Trace looks healthy. No issues detected.\n\n")
	}

	return b.String()
}

// renderTraceDiagnosticJSON produces structured JSON output.
func renderTraceDiagnosticJSON(diag TraceDiagnostic) string {
	data, err := json.MarshalIndent(diag, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": "failed to marshal diagnostic: %v"}`, err)
	}
	return string(data) + "\n"
}

// loadTraceExport reads and parses a trace export JSON file.
func loadTraceExport(path string) ([]trace.TraceEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read trace file: %w", err)
	}

	var entries []trace.TraceEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("failed to parse trace JSON: %w", err)
	}
	return entries, nil
}

// uniqueStrings deduplicates a string slice.
func uniqueStrings(s []string) []string {
	seen := map[string]bool{}
	var result []string
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}
