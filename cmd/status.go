package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/storage"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:         "status",
	Short:       "Show the complete authoritative colony snapshot.",
	Args:        cobra.NoArgs,
	Annotations: map[string]string{"aether.io/read-only": "true"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			if shouldRenderVisualOutput(stdout) {
				writeVisualOutput(stdout, renderNoColonyStatusVisual())
				return nil
			}
			renderRecoveryMenu("status", colonyStateLoadMessage(errNoColonyInitialized), nil)
			return nil
		}
		state, err := loadActiveColonyStateReadOnly()
		if err != nil {
			if shouldRenderVisualOutput(stdout) && strings.Contains(colonyStateLoadMessage(err), "No colony initialized") {
				writeVisualOutput(stdout, renderNoColonyStatusVisual())
				return nil
			}
			renderRecoveryMenu("status", colonyStateLoadMessage(err), nil)
			return nil
		}
		root := resolveAetherRoot()
		facts, factsErr := loadLifecycleFacts(root, store, time.Now().UTC())
		if factsErr != nil {
			facts = unavailableLifecycleFacts(root, time.Now().UTC(), factsErr.Error())
		}

		// Keep the mature dashboard as the authoritative status payload and
		// visual, then add the shared resolver answer through buildStatusResult.
		// That helper folds one next-action card and its projection into the same
		// map, so neither JSON nor the terminal loses dashboard facts or decides
		// the next action twice.
		result := buildStatusResult(state, store)
		projection := projectLifecycle(facts, LifecycleViewFull, detectPlatform())
		projection.Command = "status"
		result["lifecycle"] = projection
		outputWorkflow(result, renderDashboard(state, store, result))
		return nil
	},
}

func init() {
	statusCmd.Flags().Bool("compact", false, "Show the strict compact subset of the colony snapshot")
	rootCmd.AddCommand(statusCmd)
}

// renderColonyHealthLine compresses the vital-signs snapshot to one status
// line: label, score, and the three counts that produced it.
func renderColonyHealthLine(vitals map[string]interface{}) string {
	label := strings.TrimSpace(stringValue(vitals["health_label"]))
	score := intValue(vitals["overall_health"])
	if label == "" || score == 0 {
		return ""
	}
	signals := 0
	if section, ok := vitals["signal_health"].(map[string]interface{}); ok {
		signals = intValue(section["active_count"])
	}
	instincts := 0
	if section, ok := vitals["memory_pressure"].(map[string]interface{}); ok {
		instincts = intValue(section["instinct_count"])
	}
	return fmt.Sprintf("\nColony health (this project's health): %s (%d/100) — %d signal(s) active, %d instinct(s) learned\n", label, score, signals, instincts)
}

// renderColonyHealthBreakdown renders the five component signals beneath the
// health line (SEE-01): the score is explainable on screen, not a bare number.
func renderColonyHealthBreakdown(vitals map[string]interface{}) string {
	var b strings.Builder
	if section, ok := vitals["build_velocity"].(map[string]interface{}); ok {
		b.WriteString(fmt.Sprintf("   Build velocity: %v phase(s)/day (%s)\n", section["phases_per_day"], stringValue(section["trend"])))
	}
	if section, ok := vitals["error_rate"].(map[string]interface{}); ok {
		b.WriteString(fmt.Sprintf("   Error rate:     %v error(s)/day (%s)\n", section["errors_per_day"], stringValue(section["status"])))
	}
	if section, ok := vitals["signal_health"].(map[string]interface{}); ok {
		b.WriteString(fmt.Sprintf("   Signal health:  %d active (%s)\n", intValue(section["active_count"]), stringValue(section["status"])))
	}
	if section, ok := vitals["memory_pressure"].(map[string]interface{}); ok {
		b.WriteString(fmt.Sprintf("   Memory:         %d instinct(s) — lessons learned (%s)\n", intValue(section["instinct_count"]), stringValue(section["status"])))
	}
	ageHours := 0.0
	switch v := vitals["colony_age_hours"].(type) {
	case float64:
		ageHours = v
	case int:
		ageHours = float64(v)
	}
	if ageHours >= 48 {
		b.WriteString(fmt.Sprintf("   Colony age (how long this project has run): %.0fd\n", ageHours/24))
	} else if ageHours > 0 {
		b.WriteString(fmt.Sprintf("   Colony age (how long this project has run): %.0fh\n", ageHours))
	}
	return b.String()
}

func renderNoColonyStatusVisual() string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("status"), "Colony Status"))
	b.WriteString(visualDividerStr())
	b.WriteString("No colony initialized in this repo.\n")
	b.WriteString(renderNextUp(
		`Run `+"`aether init \"goal\"`"+` to start a colony.`,
		`Run `+"`aether lay-eggs`"+` first if this repo has not been set up for Aether yet.`,
	))
	return b.String()
}

// computeWarnings inspects colony state and data files for actionable issues.
// Returns a list of warning strings. Empty list means no warnings.
func computeWarnings(state colony.ColonyState, s *storage.Store) []string {
	var warnings []string

	// 0. Aether itself is behind the hub. Repos rot silently — one on this
	// machine sat 23 releases behind — so this is surfaced wherever the user
	// already looks for the colony's health.
	if s != nil {
		if repoDir := repoRootFromStore(s); repoDir != "" {
			if note := renderUpdateAvailableWarning(checkUpdateAvailable(repoDir)); note != "" {
				warnings = append(warnings, note)
			}
		}
	}

	// 1. Stale state warning
	if state.InitializedAt != nil && time.Since(*state.InitializedAt) > 7*24*time.Hour {
		warnings = append(warnings, "Stale: this project was last active more than 7 days ago. Recent work may not be reflected.")
	}
	if state.Plan.GeneratedAt != nil && time.Since(*state.Plan.GeneratedAt) > 7*24*time.Hour {
		warnings = append(warnings, "Stale: this project's plan was generated more than 7 days ago. Recent work may not be reflected.")
	}

	// 2. Failed phases warning
	for _, phase := range state.Plan.Phases {
		if phase.Status == "failed" {
			warnings = append(warnings, fmt.Sprintf("Failed phase %d (%s). Run `aether build %d` to retry.", phase.ID, phase.Name, phase.ID))
		}
	}

	// 3. Unacknowledged midden warning
	if s != nil {
		var mf colony.MiddenFile
		if s.LoadJSON("midden.json", &mf) == nil {
			unackCount := 0
			for _, entry := range mf.Entries {
				if entry.Acknowledged == nil || *entry.Acknowledged == false {
					unackCount++
				}
			}
			if unackCount > 0 {
				warnings = append(warnings, fmt.Sprintf("%d unacknowledged failure(s). Run `aether midden-review` to inspect.", unackCount))
			}
		}

		// 4. Pheromone expiry warning
		var pf colony.PheromoneFile
		if s.LoadJSON("pheromones.json", &pf) == nil {
			for _, sig := range pf.Signals {
				if !sig.Active || sig.ExpiresAt == nil {
					continue
				}
				expiry, err := time.Parse(time.RFC3339, *sig.ExpiresAt)
				if err != nil {
					continue
				}
				if time.Until(expiry) < 3*24*time.Hour {
					content := extractContentText(sig.Content)
					if len(content) > 40 {
						content = content[:37] + "..."
					}
					warnings = append(warnings, fmt.Sprintf("Expiring signal: %s -- %s (expires in less than 3 days)", sig.Type, content))
				}
			}
		}
	}

	// 5. Platform health warnings
	if s != nil {
		var ph map[string]interface{}
		if s.LoadJSON("platform-health.json", &ph) == nil {
			if failed, ok := ph["failed_commands"].([]interface{}); ok && len(failed) > 0 {
				warnings = append(warnings, fmt.Sprintf("Platform health: %d command(s) failed smoke test. Run `aether medic` to diagnose.", len(failed)))
			}
			if mismatches, ok := ph["flag_mismatches"].([]interface{}); ok && len(mismatches) > 0 {
				warnings = append(warnings, fmt.Sprintf("Platform health: %d CLI flag mismatch(es) detected. Run `aether audit-catalog` to review.", len(mismatches)))
			}
			// 5a. Doc-CLI alignment warnings
			if dcaRaw, ok := ph["doc_cli_alignment"]; ok {
				var dca map[string]interface{}
				if dcaMap, ok := dcaRaw.(map[string]interface{}); ok {
					dca = dcaMap
				} else {
					// Try JSON round-trip for typed structs
					b, _ := json.Marshal(dcaRaw)
					json.Unmarshal(b, &dca)
				}
				if hcf, ok := dca["host_critical_failures"].([]interface{}); ok && len(hcf) > 0 {
					warnings = append(warnings, fmt.Sprintf("Doc-CLI alignment: %d host-critical flag mismatch(es) detected", len(hcf)))
				}
				if w, ok := dca["warnings"].([]interface{}); ok && len(w) > 0 {
					warnings = append(warnings, fmt.Sprintf("Doc-CLI alignment: %d non-critical flag mismatch(es) detected", len(w)))
				}
				if cc, ok := dca["commands_checked"].(float64); ok && cc == 0 {
					warnings = append(warnings, "Doc-CLI alignment smoke test did not run")
				}
			}
		}
	}

	return warnings
}

// ---------------------------------------------------------------------------
// Reconciliation: unreconciled worker changes
// ---------------------------------------------------------------------------

// unreconciledChangesResult holds the outcome of comparing git working tree
// changes against worker-reported files.
type unreconciledChangesResult struct {
	HasUnreconciledChanges bool     `json:"has_unreconciled_changes"`
	ChangedFiles           []string `json:"changed_files,omitempty"`
	Recommendation         string   `json:"recommendation,omitempty"`
}

// detectUnreconciledChanges compares the current git working tree against
// files recorded in the latest build claims and manifest dispatches.
// If changed files exist that are NOT in any worker result, they are flagged.
func detectUnreconciledChanges(s *storage.Store, state *colony.ColonyState) unreconciledChangesResult {
	result := unreconciledChangesResult{}
	if s == nil {
		return result
	}

	// a. Run git status --short in the repo root
	root := resolveAetherRoot()
	cmd := exec.Command("git", "-C", root, "status", "--short")
	out, err := cmd.Output()
	if err != nil {
		// Not a git repo or git not available -- nothing to reconcile
		return result
	}

	// b. Collect modified/added/deleted files
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var changedFiles []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) < 3 {
			continue
		}
		// git status --short format: XY <path> or XY <path> -> <origpath>
		// The path starts at index 3
		path := strings.TrimSpace(line[2:])
		if path == "" {
			continue
		}
		// Handle rename format: "R  old -> new"
		if strings.Contains(path, " -> ") {
			parts := strings.Split(path, " -> ")
			if len(parts) == 2 {
				changedFiles = append(changedFiles, strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
				continue
			}
		}
		changedFiles = append(changedFiles, path)
	}
	if len(changedFiles) == 0 {
		return result
	}

	// c. Gather all files reported by workers (build claims + manifest dispatches)
	recorded := make(map[string]bool)

	// From last-build-claims.json
	claims, claimsOk := loadCodexBuildClaims()
	if claimsOk {
		for _, f := range claims.FilesCreated {
			recorded[strings.TrimSpace(f)] = true
		}
		for _, f := range claims.FilesModified {
			recorded[strings.TrimSpace(f)] = true
		}
		for _, f := range claims.TestsWritten {
			recorded[strings.TrimSpace(f)] = true
		}
		for _, tc := range claims.TaskClaims {
			for _, f := range tc.FilesCreated {
				recorded[strings.TrimSpace(f)] = true
			}
			for _, f := range tc.FilesModified {
				recorded[strings.TrimSpace(f)] = true
			}
			for _, f := range tc.TestsWritten {
				recorded[strings.TrimSpace(f)] = true
			}
		}
	}

	// From current phase manifest dispatches (if state available)
	if state != nil && state.CurrentPhase > 0 {
		manifest := loadCodexContinueManifest(state.CurrentPhase)
		if manifest.Present {
			for _, d := range manifest.Data.Dispatches {
				for _, f := range d.Outputs {
					recorded[strings.TrimSpace(f)] = true
				}
			}
		}
	}

	// d. Compare: are changed files listed in any worker result?
	var unreconciled []string
	for _, f := range changedFiles {
		if !recorded[f] {
			unreconciled = append(unreconciled, f)
		}
	}

	if len(unreconciled) > 0 {
		result.HasUnreconciledChanges = true
		result.ChangedFiles = unreconciled
		result.Recommendation = "Run `aether build-reconcile` to record changes"
	}

	return result
}

// renderReconciliationSection renders the reconciliation block for visual mode.
// Returns empty string when there are no unreconciled changes.
func renderReconciliationSection(result unreconciledChangesResult) string {
	if !result.HasUnreconciledChanges {
		return ""
	}
	var b strings.Builder
	b.WriteString(renderBanner("⚠️", "Reconciliation"))
	b.WriteString(visualDividerStr())
	fmt.Fprintf(&b, "%d unreconciled file change(s) not recorded by any worker\n", len(result.ChangedFiles))
	for _, f := range result.ChangedFiles {
		fmt.Fprintf(&b, "  - %s\n", f)
	}
	if result.Recommendation != "" {
		fmt.Fprintf(&b, "  %s\n", result.Recommendation)
	}
	b.WriteString("\n")
	return b.String()
}

// Warnings are visual-mode only. JSON output uses structured colony state data.
func renderWarningsSection(warnings []string) string {
	if len(warnings) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(renderBanner("\u26A0\uFE0F", "Warnings"))
	b.WriteString(visualDividerStr())
	for _, w := range warnings {
		b.WriteString(voiceLine("warning", w))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	return b.String()
}

// loadRecentLoopBreakEvents queries the event bus for loop-break events from the past 7 days.
// Returns at most 5 events in newest-first order. Returns nil if store is nil or no events found.
func loadRecentLoopBreakEvents(s *storage.Store) []events.Event {
	if s == nil {
		return nil
	}
	bus := events.NewBus(s, events.DefaultConfig())
	since := time.Now().AddDate(0, 0, -7)
	evts, err := bus.Query(context.Background(), events.CeremonyTopicLoopBreak, since, 5)
	if err != nil || len(evts) == 0 {
		return nil
	}
	// Reverse for newest-first display (Query returns oldest first)
	for i, j := 0, len(evts)-1; i < j; i, j = i+1, j-1 {
		evts[i], evts[j] = evts[j], evts[i]
	}
	return evts
}

// renderLoopSafetySection renders the Loop Safety dashboard section.
// Returns empty string when no events are provided (section omitted per D-07).
func renderLoopSafetySection(loopEvents []events.Event) string {
	if len(loopEvents) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(renderBanner("\U0001F527", "Loop Safety"))
	b.WriteString(visualDividerStr())
	fmt.Fprintf(&b, "Loop Safety: %d events in last 7 days\n", len(loopEvents))

	// Classic activity-feed lines, not a bordered table.
	for _, evt := range loopEvents {
		var payload events.CeremonyPayload
		if err := json.Unmarshal(evt.Payload, &payload); err != nil {
			continue
		}
		fmt.Fprintf(&b, "   [%s] 🔧 %s: %s\n", formatTimestamp(evt.Timestamp), payload.LoopType, strings.TrimSpace(payload.DetectionSignal))
		if action := strings.TrimSpace(payload.ActionTaken); action != "" {
			fmt.Fprintf(&b, "      └── %s\n", action)
		}
	}
	return b.String()
}

// loadMostRecentColonyEpisode returns the shared lineage's own newest entry
// (loadColonyEpisodeIndex, cmd/episode_index.go, already ordered newest
// first) -- never a second selection rule. ok is false only when the index
// has no entries at all.
func loadMostRecentColonyEpisode(root string, s *storage.Store) (colonyEpisodeEntry, bool) {
	idx, err := loadColonyEpisodeIndex(root, s)
	if err != nil || len(idx.Entries) == 0 {
		return colonyEpisodeEntry{}, false
	}
	return idx.Entries[0], true
}

// renderMostRecentEpisodeSection renders status's "most recent episode"
// section: what it was, how it ended, what it cost, and where to read it in
// full (LIVE-05, LIVE-07, D-11). Returns empty string when there is nothing
// to show, so the section is OMITTED entirely rather than rendered empty --
// mirrors renderLoopSafetySection's identical loader+renderer shape.
// Resolving and rendering this section performs no write of any kind: every
// read behind it (loadColonyEpisodeIndex, loadSpendLedgersForPhase) is a
// plain, already-proven read-only path.
func renderMostRecentEpisodeSection(entry colonyEpisodeEntry, ok bool) string {
	if !ok {
		return ""
	}
	var b strings.Builder
	b.WriteString(renderBanner("\U0001F4D6", "Most Recent Episode"))
	b.WriteString(visualDividerStr())
	fmt.Fprintf(&b, "%s: %s\n", colonyEpisodeKindLabel(entry.Kind), entry.Subject)

	outcome := entry.Outcome
	if !entry.OutcomeKnown {
		outcome = "unknown"
	}
	fmt.Fprintf(&b, "Outcome: %s\n", outcome)
	fmt.Fprintf(&b, "Standing: %s\n", workStandingLabel(entry.Standing, entry.StandingReason))

	if !entry.StartedAt.IsZero() && !entry.EndedAt.IsZero() {
		fmt.Fprintf(&b, "Elapsed: %s\n", entry.EndedAt.Sub(entry.StartedAt).Round(time.Second))
	}
	fmt.Fprintf(&b, "Read the full write-up: %s\n\n", emptyFallback(entry.Path, "not available"))

	b.WriteString(renderSpendCostLineFromLedgers(colonyEpisodeLedgers(entry.Cost)))
	return b.String()
}

// renderMostRecentEpisodeStatusSection resolves root through the same
// resolveAetherRoot the watch and history commands already use (never
// skillWorkspaceRoot's cwd-dependent guess, which a subagent's working
// directory can silently defeat) so status, history and watch can never
// disagree about which colony they are describing.
func renderMostRecentEpisodeStatusSection(s *storage.Store) string {
	entry, ok := loadMostRecentColonyEpisode(resolveAetherRoot(), s)
	return renderMostRecentEpisodeSection(entry, ok)
}

// renderGateStatusSection renders the Gate Status dashboard section.
// Returns empty string when no gate-results file exists for the current phase,
// or when the phase is 0 (not started).
func renderGateStatusSection(state colony.ColonyState, s *storage.Store) string {
	if s == nil || state.CurrentPhase <= 0 {
		return ""
	}

	rel := fmt.Sprintf("gate-results-%d.json", state.CurrentPhase)
	var results []GateCheckResult
	if err := s.LoadJSON(rel, &results); err != nil || len(results) == 0 {
		return ""
	}

	passed := 0
	failed := 0
	skipped := 0
	var lastRun string
	for _, r := range results {
		switch r.Status {
		case "passed":
			passed++
		case "failed":
			failed++
		case "skipped", "not-reached":
			skipped++
		}
		if r.Timestamp > lastRun {
			lastRun = r.Timestamp
		}
	}

	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("status"), "Gate Status"))
	b.WriteString(visualDividerStr())
	fmt.Fprintf(&b, "Phase: %d\n", state.CurrentPhase)
	fmt.Fprintf(&b, "Gates: %d (%d passed, %d failed, %d skipped)\n", len(results), passed, failed, skipped)
	if lastRun != "" {
		fmt.Fprintf(&b, "Last Run: %s\n", formatTimestamp(lastRun))
	}

	if failed > 0 {
		b.WriteString("\nFailed Gates:\n")
		for _, r := range results {
			if r.Status == "failed" {
				fmt.Fprintf(&b, "  - %s: %s", r.Name, emptyFallback(r.Detail, "no detail"))
				if r.FixHint != "" {
					fmt.Fprintf(&b, " (fix: %s)", r.FixHint)
				}
				b.WriteString("\n")
			}
		}
	}

	b.WriteString("\n")
	return b.String()
}

func loadGuidedActions(s *storage.Store, root string) []guidedAction {
	if s == nil {
		return nil
	}
	actions := []guidedAction{}
	if action, ok := activeFlagGuidedAction(s); ok {
		actions = append(actions, action)
	}
	if action, ok := oracleGuidedAction(root); ok {
		actions = append(actions, action)
	}
	if action, ok := middenGuidedAction(s); ok {
		actions = append(actions, action)
	}
	return actions
}

func activeFlagGuidedAction(s *storage.Store) (guidedAction, bool) {
	flags, ok := loadFlagsFile(s)
	if !ok {
		return guidedAction{}, false
	}
	active := make([]colony.FlagEntry, 0, len(flags.Decisions))
	blockers := 0
	issues := 0
	for _, flag := range flags.Decisions {
		if flag.Resolved {
			continue
		}
		active = append(active, flag)
		switch flag.Type {
		case "blocker":
			blockers++
		case "issue":
			issues++
		}
	}
	if len(active) == 0 {
		return guidedAction{}, false
	}
	top := firstActionableFlag(active)
	summary := fmt.Sprintf("%d active flag(s)", len(active))
	if blockers > 0 || issues > 0 {
		summary = fmt.Sprintf("%d active flag(s): %d blocker(s), %d issue(s)", len(active), blockers, issues)
	}
	if strings.TrimSpace(top.Description) != "" {
		summary += ". Top: " + compactActionText(top.Description, 90)
	}
	action := guidedAction{
		Title:   "Flags",
		Summary: summary,
		Command: "aether flags --status active",
	}
	if strings.TrimSpace(top.Description) != "" {
		action.AlternativeCommand = fmt.Sprintf("aether swarm %s", quoteCommandArg(top.Description))
	}
	return action, true
}

func firstActionableFlag(flags []colony.FlagEntry) colony.FlagEntry {
	for _, flag := range flags {
		if flag.Type == "blocker" {
			return flag
		}
	}
	for _, flag := range flags {
		if flag.Type == "issue" {
			return flag
		}
	}
	return flags[0]
}

func oracleGuidedAction(root string) (guidedAction, bool) {
	root = strings.TrimSpace(root)
	if root == "" {
		return guidedAction{}, false
	}
	paths := oracleWorkspacePaths(root)
	if _, err := os.Stat(paths.StatePath); err != nil {
		return guidedAction{}, false
	}
	state, err := loadOracleStateFile(paths.StatePath)
	if err != nil {
		return guidedAction{}, false
	}
	status := strings.ToLower(strings.TrimSpace(state.Status))
	if status == "" || status == "complete" || status == "stopped" {
		return guidedAction{}, false
	}
	summary := "Oracle research is " + status
	if strings.TrimSpace(state.Summary) != "" {
		summary += ": " + compactActionText(state.Summary, 100)
	} else if strings.TrimSpace(state.ActiveQuestionText) != "" {
		summary += ": " + compactActionText(state.ActiveQuestionText, 100)
	}
	return guidedAction{
		Title:   "Oracle",
		Summary: summary,
		Command: "aether oracle status",
	}, true
}

func middenGuidedAction(s *storage.Store) (guidedAction, bool) {
	var mf colony.MiddenFile
	if err := s.LoadJSON("midden.json", &mf); err != nil {
		return guidedAction{}, false
	}
	count := 0
	var top colony.MiddenEntry
	for _, entry := range mf.Entries {
		if entry.Acknowledged != nil && *entry.Acknowledged {
			continue
		}
		count++
		if top.ID == "" {
			top = entry
		}
	}
	if count == 0 {
		return guidedAction{}, false
	}
	summary := fmt.Sprintf("%d unacknowledged failure(s)", count)
	if strings.TrimSpace(top.Message) != "" {
		summary += ". Top: " + compactActionText(top.Message, 90)
	}
	action := guidedAction{
		Title:   "Failures",
		Summary: summary,
		Command: "aether midden-review",
	}
	if strings.TrimSpace(top.Message) != "" {
		action.AlternativeCommand = fmt.Sprintf("aether swarm %s", quoteCommandArg(top.Message))
	}
	return action, true
}

func renderGuidedActions(actions []guidedAction) string {
	if len(actions) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\nGuided Actions\n")
	for _, action := range actions {
		b.WriteString("   ")
		b.WriteString(action.Title)
		b.WriteString(": ")
		b.WriteString(action.Summary)
		b.WriteString("\n")
		if strings.TrimSpace(action.Command) != "" {
			b.WriteString("      " + voiceGlyph("next") + " Next: `")
			b.WriteString(action.Command)
			b.WriteString("`\n")
		}
		if strings.TrimSpace(action.AlternativeCommand) != "" {
			b.WriteString("      " + voiceGlyph("colony") + " Swarm: `")
			b.WriteString(action.AlternativeCommand)
			b.WriteString("`\n")
		}
	}
	return b.String()
}

func guidedActionNextUpPrimary(action guidedAction) string {
	if strings.TrimSpace(action.Command) == "" {
		return action.Summary
	}
	return fmt.Sprintf("Run `%s` to handle: %s.", action.Command, action.Summary)
}

func guidedActionNextUpAlternatives(actions []guidedAction) []string {
	alternatives := []string{}
	for _, action := range actions {
		if strings.TrimSpace(action.AlternativeCommand) != "" {
			alternatives = append(alternatives, fmt.Sprintf("Run `%s` to investigate with the swarm.", action.AlternativeCommand))
		}
		if strings.TrimSpace(action.Command) != "" {
			alternatives = append(alternatives, fmt.Sprintf("Run `%s` to inspect %s.", action.Command, strings.ToLower(action.Title)))
		}
	}
	return alternatives
}

func guidedActionMaps(actions []guidedAction) []map[string]interface{} {
	maps := make([]map[string]interface{}, 0, len(actions))
	for _, action := range actions {
		maps = append(maps, map[string]interface{}{
			"title":               action.Title,
			"summary":             action.Summary,
			"command":             action.Command,
			"alternative_command": action.AlternativeCommand,
		})
	}
	return maps
}

// buildStatusResult constructs the JSON result for the status command.
func buildStatusResult(state colony.ColonyState, s *storage.Store) map[string]interface{} {
	totalPhases := len(state.Plan.Phases)
	completedPhases := 0
	for _, phase := range state.Plan.Phases {
		if phase.Status == colony.PhaseCompleted {
			completedPhases++
		}
	}
	phasePosition := completedPhases
	switch state.State {
	case colony.StateEXECUTING, colony.StateBUILT:
		if state.CurrentPhase > phasePosition {
			phasePosition = state.CurrentPhase
		}
	case colony.StateCOMPLETED:
		phasePosition = totalPhases
	}

	var tasksCompleted, tasksTotal int
	var phaseName string
	displayPhase := recoveryPhase(&state)
	displayPhaseNum := 0
	if displayPhase != nil {
		displayPhaseNum = displayPhase.ID
		phase := *displayPhase
		phaseName = phase.Name
		tasksTotal = len(phase.Tasks)
		for _, task := range phase.Tasks {
			if task.Status == colony.TaskCompleted {
				tasksCompleted++
			}
		}
		if state.State == colony.StateCOMPLETED && phase.Status == colony.PhaseCompleted && tasksCompleted < tasksTotal {
			tasksCompleted = tasksTotal
		}
	}

	result := map[string]interface{}{
		"goal":                   state.Goal,
		"state":                  string(state.State),
		"current_phase":          state.CurrentPhase,
		"total_phases":           totalPhases,
		"phases_completed":       phasePosition,
		"phase_name":             phaseName,
		"display_phase":          displayPhaseNum,
		"tasks_completed":        tasksCompleted,
		"tasks_total":            tasksTotal,
		"colony_mode":            string(state.EffectiveColonyMode()),
		"agent_delegate_session": codex.IsAgentDelegateSession(),
		"plan_revision":          planRevisionSummary(state.Plan),
	}
	blockers, blockerErr := readBlockerSnapshot(s)
	issues, notes := countStatusNonBlockerFlags(s)
	result["blocker_snapshot_available"] = blockerErr == nil
	if blockerErr != nil {
		result["blocker_snapshot_error"] = blockerSnapshotErrorDetail(s, blockerErr)
		result["blockers"] = nil
		result["blocker_ids"] = nil
		result["escalated_blockers"] = nil
		result["escalated_blocker_ids"] = nil
	} else {
		addBlockerSnapshotFields(result, blockers)
	}
	result["issues"] = issues
	result["notes"] = notes

	if s != nil {
		warnings := computeWarnings(state, s)
		if len(warnings) > 0 {
			result["warnings"] = warnings
		}
	}
	if _, attempt, ok := loadRelevantBuildAttempt(state); ok {
		result["build_attempt"] = buildAttemptSummary(attempt)
	}
	if report := loadAutopilotLastReport(s); report != nil {
		result["last_report"] = report
	}
	if runningTotal := computeColonyRunningSpendTotal(state); runningTotal.hasAnyFacts() {
		result["colony_running_total"] = runningTotal.jsonSummary()
	}

	// Reconciliation section (JSON mode)
	recon := detectUnreconciledChanges(s, &state)
	if recon.HasUnreconciledChanges {
		result["reconciliation"] = recon
	}

	// Status knows two things the saved project alone does not: workers are
	// still running, and a guided action (an open flag, active Oracle
	// research, an unacknowledged failure) needs attention before anything
	// else. Both enter the one decision as an override input here, so the
	// JSON envelope and the dashboard screen -- built from this same result
	// map -- cannot answer the question differently.
	activeWorkers := statusActiveWorkers(s, state)
	guidedActions := loadGuidedActions(s, skillWorkspaceRoot())
	overrideCommand, overrideWhy := statusOverrideFacts(activeWorkers, guidedActions)
	closeLifecycleCommand(result, "status", overrideCommand, overrideWhy)

	return result
}

// statusLiveSpawnView is the one rule for "may this screen present workers as
// live". A worker is live only while the colony is genuinely executing an
// unpaused build. Every section that shows workers -- the Active Workers list,
// the JSON envelope, and the BIO-06 governed family tree -- reads this single
// function, so two sections can never disagree about whether a paused or
// finished colony has anyone running. Phase 203 added the family tree without
// consulting it, which put a previous session's worker on a paused colony's
// status screen (TestStatusPausedColonyIgnoresStaleSpawnTreeWorkers).
func statusLiveSpawnView(state colony.ColonyState) bool {
	return state.State == colony.StateEXECUTING && !state.Paused && state.BuildStartedAt != nil
}

// statusActiveWorkers is the same "are workers genuinely still running"
// check renderDashboard makes, factored out so both the JSON envelope and
// the screen resolve their answer from the identical fact.
func statusActiveWorkers(s *storage.Store, state colony.ColonyState) []agent.SpawnEntry {
	spawnSummary := loadSpawnActivitySummaryForState(s, &state)
	liveSpawnView := statusLiveSpawnView(state)
	if !liveSpawnView {
		return nil
	}
	return spawnSummary.ActiveEntries
}

// statusOverrideFacts turns what status alone knows into an override for the
// one decision. In-flight workers outrank a guided action, matching the
// dashboard's own priority: there is no point steering the owner toward a
// flag or a failure while a command is still running.
func statusOverrideFacts(activeWorkers []agent.SpawnEntry, guidedActions []guidedAction) (string, string) {
	if len(activeWorkers) > 0 {
		return "aether status", fmt.Sprintf(
			"Active workers are still running (%d). Wait for the in-flight command to finish, then check again -- "+
				"this only refreshes the dashboard, it changes nothing.", len(activeWorkers))
	}
	if len(guidedActions) > 0 {
		top := guidedActions[0]
		if command := strings.TrimSpace(top.Command); command != "" {
			return command, fmt.Sprintf("%s needs attention: %s", top.Title, top.Summary)
		}
	}
	return "", ""
}

// renderDashboard produces the full colony status dashboard string. result is
// the same map buildStatusResult produced for the JSON envelope, carrying the
// one resolver's already-folded answer -- so the screen and the
// machine-readable result can never disagree about what to do next.
// skillSourceSentence turns the internal skill-source key into ordinary words
// at the moment it becomes text.
//
// The raw key (e.g. "phase_plan") used to be printed straight onto the status
// screen. TestVoicedScreensCarryNoRawStateToken forbids exactly that -- an
// internal state name or a raw key=value pair is never shown to the owner
// where a sentence belongs -- and it caught this the moment the real status
// screen joined the voice corpus. An unrecognised key degrades to its own
// underscores-to-spaces reading rather than being hidden, so a new source
// shows up readable instead of silently blank.
func skillSourceSentence(source string) string {
	switch strings.TrimSpace(source) {
	case "":
		return "none recorded"
	case "phase_plan":
		return "the phase plan"
	case "colony_prime":
		return "the shared context pack"
	case "worker_brief":
		return "the worker brief"
	default:
		return strings.ReplaceAll(strings.TrimSpace(source), "_", " ")
	}
}

func renderDashboard(state colony.ColonyState, s *storage.Store, result map[string]interface{}) string {
	var b strings.Builder

	// Banner
	b.WriteString(renderBanner(commandEmoji("status"), "Colony Status"))
	b.WriteString(visualDividerStr())

	// Goal (truncated to 60 chars)
	goal := *state.Goal
	if len(goal) > 60 {
		goal = goal[:57] + "..."
	}
	fmt.Fprintf(&b, "%s\n\n", voiceLine("goal", fmt.Sprintf("Goal: %s", goal)))

	// Version line
	renderVersionLine(&b)

	// Signal summary
	renderSignalSummaryLine(&b, s)

	// Warnings (visual-mode only)
	warnings := computeWarnings(state, s)
	b.WriteString(renderWarningsSection(warnings))

	// Loop Safety (per D-05: between Warnings and Progress)
	if s != nil {
		loopEvents := loadRecentLoopBreakEvents(s)
		if len(loopEvents) > 0 {
			b.WriteString(renderLoopSafetySection(loopEvents))
		}
	}

	// Gate Status (GATE-09)
	if gateSection := renderGateStatusSection(state, s); gateSection != "" {
		b.WriteString(gateSection)
	}

	// Progress
	totalPhases := len(state.Plan.Phases)
	completedPhases := 0
	for _, phase := range state.Plan.Phases {
		if phase.Status == colony.PhaseCompleted {
			completedPhases++
		}
	}
	phasePosition := completedPhases
	switch state.State {
	case colony.StateEXECUTING, colony.StateBUILT:
		if state.CurrentPhase > phasePosition {
			phasePosition = state.CurrentPhase
		}
	case colony.StateCOMPLETED:
		phasePosition = totalPhases
	}
	phaseBar := generateProgressBar(phasePosition, totalPhases, 20)
	fmt.Fprintf(&b, "%s\n", voiceLine("phase", "Progress"))
	phasePercent := 0
	if totalPhases > 0 {
		cappedPhase := phasePosition
		if cappedPhase < 0 {
			cappedPhase = 0
		}
		if cappedPhase > totalPhases {
			cappedPhase = totalPhases
		}
		phasePercent = cappedPhase * 100 / totalPhases
	}
	fmt.Fprintf(&b, "   %s\n", voiceLine("phase", fmt.Sprintf("Phase: [Phase %d/%d] %s %d%%", phasePosition, totalPhases, phaseBar, phasePercent)))

	// Task progress in current phase
	var tasksCompleted, tasksTotal int
	var phaseName string
	displayPhase := recoveryPhase(&state)
	displayPhaseNum := 0
	if displayPhase != nil {
		displayPhaseNum = displayPhase.ID
		phase := *displayPhase
		phaseName = phase.Name
		tasksTotal = len(phase.Tasks)
		for _, task := range phase.Tasks {
			if task.Status == colony.TaskCompleted {
				tasksCompleted++
			}
		}
		// Sealed/completed colonies should not show stale incomplete task counts
		// when the phase itself has already been marked completed.
		if state.State == colony.StateCOMPLETED && phase.Status == colony.PhaseCompleted && tasksCompleted < tasksTotal {
			tasksCompleted = tasksTotal
		}
	}
	taskBar := generateProgressBar(tasksCompleted, tasksTotal, 20)
	taskPercent := 0
	if tasksTotal > 0 {
		cappedTasks := tasksCompleted
		if cappedTasks < 0 {
			cappedTasks = 0
		}
		if cappedTasks > tasksTotal {
			cappedTasks = tasksTotal
		}
		taskPercent = cappedTasks * 100 / tasksTotal
	}
	if phaseName != "" {
		fmt.Fprintf(&b, "   %s\n\n", voiceLine("task", fmt.Sprintf("Tasks: [Tasks %d/%d] %s %d%% in Phase %d (%s)", tasksCompleted, tasksTotal, taskBar, taskPercent, displayPhaseNum, phaseName)))
	} else {
		fmt.Fprintf(&b, "   %s\n\n", voiceLine("task", fmt.Sprintf("Tasks: [Tasks %d/%d] %s %d%% in Phase %d", tasksCompleted, tasksTotal, taskBar, taskPercent, displayPhaseNum)))
	}

	// Constraints
	focusCount, avoidCount := countConstraints(s)
	fmt.Fprintf(&b, "%s\n", voiceLine("focus", fmt.Sprintf("Focus: %d areas | Avoid: %d patterns", focusCount, avoidCount)))

	// Instincts
	instincts := loadRuntimeInstincts(s, &state)
	totalInstincts := len(instincts)
	highConf := 0
	for _, inst := range instincts {
		if inst.Confidence >= 0.7 {
			highConf++
		}
	}
	fmt.Fprintf(&b, "%s\n", voiceLine("learning", fmt.Sprintf("Instincts: %d learned (%d strong)", totalInstincts, highConf)))

	// Flags. These values were captured once when buildStatusResult assembled
	// the JSON result, so visual and machine-readable status cannot disagree.
	issues := intValue(result["issues"])
	notes := intValue(result["notes"])
	if available, _ := result["blocker_snapshot_available"].(bool); !available {
		b.WriteString("Blocker truth: unavailable\n")
	} else {
		blockers := intValue(result["blockers"])
		escalatedBlockers := intValue(result["escalated_blockers"])
		fmt.Fprintf(&b, "%s\n", voiceLine("flag", fmt.Sprintf("Flags: %d blockers | %d issues | %d notes", blockers, issues, notes)))
		fmt.Fprintf(&b, "%s\n", voiceLine("blocked", fmt.Sprintf("Existing blocker work: %d active (%d escalated)", blockers, escalatedBlockers)))
	}
	if report := renderAutopilotReportFromResult(result); report != "" {
		b.WriteString("\n")
		b.WriteString(report)
		b.WriteString("\n")
	}

	// Scope
	fmt.Fprintf(&b, "%s\n", voiceLine("status", fmt.Sprintf("Scope: %s", state.EffectiveScope())))
	fmt.Fprintf(&b, "%s\n", voiceLine("colony", fmt.Sprintf("Colony Mode (how this project runs): %s", state.EffectiveColonyMode())))
	if revision, ok := activePlanRevision(state.Plan); ok {
		fmt.Fprintf(&b, "Plan Revision: r%d (%s) - %s\n", revision.Number, revision.ReasonType, revision.Reason)
	} else if len(state.Plan.Phases) > 0 {
		fmt.Fprintf(&b, "Plan Revision: legacy (%s)\n", activePlanRevisionID(state.Plan))
	}

	// Milestone
	if state.Milestone != "" {
		fmt.Fprintf(&b, "Milestone: %s\n", state.Milestone)
	}

	// Depth
	depth := state.ColonyDepth
	if depth == "" {
		depth = "standard"
	}
	depthLbl := depthLabel(depth)
	fmt.Fprintf(&b, "%s\n", voiceLine("decision", fmt.Sprintf("Depth: %s", depthLbl)))

	// Granularity
	granularity := string(state.PlanGranularity)
	if granularity == "" {
		granularity = "not set"
	}
	granLbl := granularityLabel(granularity)
	fmt.Fprintf(&b, "%s\n", voiceLine("decision", fmt.Sprintf("Granularity: %s", granLbl)))

	// Parallel mode
	parallelMode := string(state.ParallelMode)
	if parallelMode == "" {
		parallelMode = "in-repo"
	}
	fmt.Fprintf(&b, "%s\n\n", voiceLine("colony", fmt.Sprintf("Parallel: %s", parallelMode)))

	guidedActions := loadGuidedActions(s, skillWorkspaceRoot())
	b.WriteString(renderGuidedActions(guidedActions))
	if len(guidedActions) > 0 {
		b.WriteString("\n")
	}

	proof := buildProofOutput(skillWorkspaceRoot(), state)
	b.WriteString(voiceLine("evidence", "Proof") + "\n")
	fmt.Fprintf(&b, "   "+voiceGlyph("memory")+" Context assembled for this project: %s | %d included | %d preserved | %d trimmed | %d blocked\n",
		proof.Summary.ContextSurface,
		proof.Summary.ContextIncluded,
		proof.Summary.ContextPreserved,
		proof.Summary.ContextTrimmed,
		proof.Summary.ContextBlocked,
	)
	if proof.Summary.SkillDispatches > 0 {
		fmt.Fprintf(&b, "   "+voiceGlyph("artifact")+" Skills: %s | %d dispatches | %d matched skills\n",
			skillSourceSentence(proof.Summary.SkillSource),
			proof.Summary.SkillDispatches,
			proof.Summary.SkillMatchedTotal,
		)
	} else {
		b.WriteString("   Skills: no phase-aware skill proof yet\n")
	}
	b.WriteString("   " + voiceLine("evidence", "Inspect: aether proof") + "\n")
	b.WriteString("\n")

	// Memory Health table
	b.WriteString(voiceLine("memory", "Memory Health") + "\n")
	renderMemoryHealthTable(&b, s)

	// Review Findings (only if data exists)
	if hasReviewFindings(s) {
		b.WriteString("\nReview Findings\n")
		renderReviewFindingsTable(&b, s)
	}

	// Pheromone Summary
	b.WriteString("\n" + voiceLine("focus", "Active Pheromones (steering notes in effect)") + "\n")
	renderPheromoneSummary(&b, s)

	spawnSummary := loadSpawnActivitySummaryForState(s, &state)
	liveSpawnView := statusLiveSpawnView(state)
	if !liveSpawnView {
		spawnSummary = withoutLiveSpawnEntries(spawnSummary)
	}
	if spawnSummary.TotalCount > 0 {
		b.WriteString("\nSpawn Activity\n")
		renderSpawnActivity(&b, spawnSummary)
	}

	activeWorkers := []agent.SpawnEntry{}
	if liveSpawnView {
		activeWorkers = spawnSummary.ActiveEntries
	}
	if len(activeWorkers) > 0 {
		b.WriteString("\nActive Workers\n")
		renderActiveWorkers(&b, activeWorkers)
	}
	if len(spawnSummary.RecentOutcomeEntries) > 0 {
		b.WriteString("\nRecent Outcomes\n")
		renderRecentWorkerOutcomes(&b, spawnSummary.RecentOutcomeEntries)
	}
	// The governed family tree presents workers as live, so it is gated by the
	// same statusLiveSpawnView rule the Active Workers list above uses rather
	// than by a second rule of its own.
	if liveSpawnView {
		if subtreeSection := renderGovernedSubtreeStatusSection(state); subtreeSection != "" {
			b.WriteString("\n")
			b.WriteString(subtreeSection)
		}
	}
	if _, attempt, ok := loadRelevantBuildAttempt(state); ok {
		b.WriteString("\nBuild Attempt\n")
		b.WriteString(renderBuildAttemptStatus(attempt))
	}
	if episodeSection := renderMostRecentEpisodeStatusSection(s); episodeSection != "" {
		b.WriteString("\n")
		b.WriteString(episodeSection)
	}
	if guidance := loadActiveRecoveryGuidance(state); guidance != nil {
		b.WriteString("\nRecovery\n")
		if guidance.Summary != "" {
			b.WriteString("  ")
			b.WriteString(guidance.Summary)
			b.WriteString("\n")
		}
		if guidance.Next != "" {
			b.WriteString("  Next: ")
			b.WriteString(guidance.Next)
			b.WriteString("\n")
		}
		if guidance.ReportPath != "" {
			b.WriteString("  Report: ")
			b.WriteString(guidance.ReportPath)
			b.WriteString("\n")
		}
	}

	// Reconciliation section (visual mode)
	recon := detectUnreconciledChanges(s, &state)
	if reconSection := renderReconciliationSection(recon); reconSection != "" {
		b.WriteString(reconSection)
	}

	if totalInstincts > 0 {
		strongestInstincts := loadStrongestRuntimeInstincts(s, &state, 3)
		if len(strongestInstincts) > 0 {
			b.WriteString("\n" + voiceLine("learning", "Strongest Instincts (lessons learned)") + "\n")
			renderStrongestInstincts(&b, strongestInstincts)
		}
	}

	// Colony health, from the same computation colony-vital-signs runs.
	// v5.4.0's status surfaced this; the modern status computed it on request
	// only, behind a subcommand nobody was told about.
	vitals := computeColonyVitalSigns(s, state)
	b.WriteString(renderColonyHealthLine(vitals))
	b.WriteString(renderColonyHealthBreakdown(vitals))

	// Running colony total (Phase 201, D-06): elapsed time and reported
	// cost, summed across every planned phase's durable facts. Omitted
	// entirely on a colony that has recorded nothing at all, so a freshly
	// initialized colony is not shown a "not known" line about work it has
	// never attempted.
	runningTotal := computeColonyRunningSpendTotal(state)
	if runningTotal.hasAnyFacts() {
		b.WriteString("\n")
		b.WriteString(renderColonyRunningSpendTotal(runningTotal))
	}

	// State
	stateLabel := string(state.State)
	if state.Paused {
		stateLabel += " (paused)"
	}
	fmt.Fprintf(&b, "\nState: %s", stateLabel)
	if len(activeWorkers) > 0 {
		fmt.Fprintf(&b, " (%d active workers)", len(activeWorkers))
	}
	b.WriteString("\n")
	if len(activeWorkers) > 0 {
		// The "wait, and here is how to watch" tips are a report, not advice
		// the resolver can offer as an aether command -- tailing a log file
		// is not something the card can recommend. The card below still says
		// to wait: statusOverrideFacts fed that fact into the one decision
		// when this result was built.
		b.WriteString("\n")
		b.WriteString(renderStageMarker("Still running"))
		b.WriteString("Watch progress with `tail -f .aether/data/spawn-tree.txt`, or run `aether proof` to inspect the active context and skill proof.\n")
	}
	b.WriteString(renderLifecycleClosing(result, "status"))

	return b.String()
}

// colonyRunningSpendTotal is the colony-wide running total across every
// planned phase's durable facts (Phase 201, D-06). ElapsedMeasured and
// CostTokens are each the exact sum of the rows/attempts that answered --
// an unreported or unmeasured row is counted in its own field and never
// folded into either sum, matching the discipline cmd/spend_cost_line.go
// already holds for one phase's own block.
type colonyRunningSpendTotal struct {
	ElapsedMeasured    time.Duration
	ElapsedUnmeasured  int
	CostTokens         int64
	CostReportedRows   int
	CostUnreportedRows int
}

// hasAnyFacts reports whether this total carries anything at all to show --
// distinguishing "the colony has genuinely recorded nothing yet" (nothing
// rendered) from "the colony has recorded something, some of it unreported
// or unmeasured" (rendered, honestly).
func (t colonyRunningSpendTotal) hasAnyFacts() bool {
	return t.ElapsedMeasured > 0 || t.ElapsedUnmeasured > 0 ||
		t.CostReportedRows > 0 || t.CostUnreportedRows > 0
}

// jsonSummary is the machine-readable shape for buildStatusResult's JSON
// envelope, carrying the identical figures the visual dashboard renders so
// neither surface can disagree with the other.
func (t colonyRunningSpendTotal) jsonSummary() map[string]interface{} {
	return map[string]interface{}{
		"elapsed_seconds":      t.ElapsedMeasured.Seconds(),
		"elapsed_unmeasured":   t.ElapsedUnmeasured,
		"cost_tokens":          t.CostTokens,
		"cost_reported_rows":   t.CostReportedRows,
		"cost_unreported_rows": t.CostUnreportedRows,
	}
}

// computeColonyRunningSpendTotal sums the running colony total across every
// phase in state's plan. It is a pure read -- loadSpendLedgersForPhase and
// loadLatestBuildAttempt are both read-only, and nothing here saves
// anything, so status stays a reader (D-06): no ledger file, no lock file
// beyond the storage layer's own first-touch bookkeeping any read already
// carries, no state mutation.
//
// Cost sums through loadSpendLedgersForPhase, the SAME loader
// cmd/spend_cost_line.go's own per-phase block already reads -- one
// accounting path, never a second one that could disagree with it. A row
// with no reported usage (including a row carrying only a local estimate,
// D-01 as amended) counts toward CostUnreportedRows, never toward
// CostTokens.
//
// Elapsed sums each phase's own LATEST build attempt only, mirroring
// renderSpendCostLine's own attempt-bound scope (Task 1): a phase with no
// attempt recorded at all contributes nothing to either figure -- there is
// nothing to say about it -- while a phase whose attempt is missing a
// timestamp counts toward ElapsedUnmeasured and contributes no duration,
// via the identical spendElapsedFigure sentinel Task 1 established.
func computeColonyRunningSpendTotal(state colony.ColonyState) colonyRunningSpendTotal {
	var total colonyRunningSpendTotal
	for _, phase := range state.Plan.Phases {
		if phase.ID <= 0 {
			continue
		}
		if ledgers, ok := loadSpendLedgersForPhase(phase.ID); ok {
			totals := computeSpendTotals(ledgers)
			total.CostTokens += totals.MeasuredTokens
			total.CostReportedRows += totals.MeasuredRows
			rows := spendRowsAcross(ledgers)
			total.CostUnreportedRows += len(rows) - totals.MeasuredRows
		}
		if _, attempt, ok := loadLatestBuildAttempt(phase.ID); ok {
			figure := spendElapsedFigure(attempt.StartedAt, attempt.CompletedAt)
			if figure == spendNotReportedFigure {
				total.ElapsedUnmeasured++
				continue
			}
			duration, err := time.ParseDuration(figure)
			if err != nil {
				total.ElapsedUnmeasured++
				continue
			}
			total.ElapsedMeasured += duration
		}
	}
	return total
}

// renderColonyRunningSpendTotal renders the colony-wide running total as one
// small dashboard section: elapsed time and reported cost, each summed from
// the same durable per-attempt/per-row facts every other screen in this
// repository reads, plus how many attempts/rows were unreported or
// unmeasured, named as such so the total is never mistaken for a complete
// figure.
func renderColonyRunningSpendTotal(total colonyRunningSpendTotal) string {
	var b strings.Builder
	b.WriteString("Colony Total\n")

	switch {
	case total.ElapsedMeasured == 0 && total.ElapsedUnmeasured == 0:
		b.WriteString("   Elapsed: not known -- no attempt has recorded a start and end yet.\n")
	case total.ElapsedUnmeasured > 0:
		fmt.Fprintf(&b, "   Elapsed: %s (%d attempt(s) unmeasured -- missing a start or end timestamp)\n",
			total.ElapsedMeasured.Round(time.Second), total.ElapsedUnmeasured)
	default:
		fmt.Fprintf(&b, "   Elapsed: %s\n", total.ElapsedMeasured.Round(time.Second))
	}

	switch {
	case total.CostReportedRows == 0 && total.CostUnreportedRows == 0:
		b.WriteString("   Cost: not known -- no worker has reported a token figure yet.\n")
	case total.CostReportedRows == 0:
		fmt.Fprintf(&b, "   Cost: not known -- %d worker run(s) unreported.\n", total.CostUnreportedRows)
	case total.CostUnreportedRows > 0:
		fmt.Fprintf(&b, "   Cost: %s tokens (%d worker run(s) unreported)\n",
			spendCompactTokenFigure(total.CostTokens), total.CostUnreportedRows)
	default:
		fmt.Fprintf(&b, "   Cost: %s tokens\n", spendCompactTokenFigure(total.CostTokens))
	}

	return b.String()
}

func renderBuildAttemptStatus(attempt buildAttemptRecord) string {
	var b strings.Builder
	fmt.Fprintf(&b, "  %s | %s | %d workers\n", attempt.ID, attempt.Status, len(attempt.Dispatches))
	if attempt.RunID != "" {
		fmt.Fprintf(&b, "  Run: %s\n", attempt.RunID)
	}
	if attempt.ManifestSHA256 != "" {
		digest := attempt.ManifestSHA256
		if len(digest) > 12 {
			digest = digest[:12]
		}
		fmt.Fprintf(&b, "  Manifest: %s\n", digest)
	}
	workerCounts := map[string]int{}
	for _, workerRun := range attempt.WorkerRuns {
		workerCounts[workerRun.Status]++
	}
	if len(attempt.WorkerRuns) > 0 {
		fmt.Fprintf(
			&b,
			"  Worker runs: %d completed | %d active | %d failed | %d blocked | %d timed out | %d cancelled\n",
			workerCounts[buildWorkerCompleted],
			workerCounts[buildWorkerDispatching],
			workerCounts[buildWorkerFailed],
			workerCounts[buildWorkerBlocked],
			workerCounts[buildWorkerTimeout],
			workerCounts[buildWorkerCancelled],
		)
	}
	if attempt.RunID != "" {
		if processes, err := codex.GlobalProcessTracker().ProcessesForRun(buildAttemptWorkspaceRoot(), attempt.RunID); err == nil && len(processes) > 0 {
			fmt.Fprintf(&b, "  Provider processes: %d active\n", len(processes))
		}
	}
	if attempt.Error != "" {
		fmt.Fprintf(&b, "  Error: %s\n", compactActionText(attempt.Error, 160))
	}
	if attempt.RecoveryCommand != "" {
		fmt.Fprintf(&b, "  Next: %s\n", attempt.RecoveryCommand)
	}
	b.WriteString(renderJobTelemetryDrillDown(attempt.ID))
	return b.String()
}

// renderJobTelemetryDrillDown renders the full eight-segment timing
// breakdown for one build attempt (Phase 201 plan 12, WORK-08), naming
// unmeasured segments as such. It reads fresh from disk every call and
// writes nothing -- status.go is a reader only, never a store initializer,
// never a lock file, never a state mutation. Renders nothing at all when no
// telemetry record exists for this attempt: a run that measured no segment
// at all writes no record (writeJobTelemetryRecord), so "no record" and "a
// record with nothing measured" are the same case here too, matching
// renderJobTelemetryClosingLine's identical choice in
// cmd/spend_cost_line.go.
func renderJobTelemetryDrillDown(attemptID string) string {
	record, ok := readJobTelemetryRecord(attemptID)
	if !ok {
		return ""
	}
	var b strings.Builder
	b.WriteString("  Timing breakdown:\n")
	for _, named := range record.namedSegments() {
		fmt.Fprintf(&b, "    %s: %s\n", jobTelemetrySegmentLabel(named.Name), named.Segment.RenderedDuration())
	}
	return b.String()
}

func loadActiveSpawnEntries(s *storage.Store) []agent.SpawnEntry {
	return loadSpawnActivitySummary(s).ActiveEntries
}

type spawnActivitySummary struct {
	Entries              []agent.SpawnEntry
	ActiveEntries        []agent.SpawnEntry
	RecentOutcomeEntries []agent.SpawnEntry
	TotalCount           int
	ActiveCount          int
	CompletedCount       int
	BlockedCount         int
	FailedCount          int
	CurrentRunID         string
	CurrentCommand       string
}

type guidedAction struct {
	Title              string
	Summary            string
	Command            string
	AlternativeCommand string
}

func loadSpawnActivitySummary(s *storage.Store) spawnActivitySummary {
	return loadSpawnActivitySummaryForState(s, nil)
}

func loadSpawnActivitySummaryForState(s *storage.Store, state *colony.ColonyState) spawnActivitySummary {
	if s == nil {
		return spawnActivitySummary{}
	}

	tree := agent.NewSpawnTree(s, "spawn-tree.txt")
	entries, err := tree.Parse()
	if err != nil || len(entries) == 0 {
		return spawnActivitySummary{}
	}

	currentRunID := ""
	currentCommand := ""
	if run, ok, runErr := tree.CurrentRun(); runErr == nil && ok {
		if filtered, filterErr := tree.EntriesForRun(run.ID); filterErr == nil && len(filtered) > 0 {
			if strings.TrimSpace(run.Command) == "continue" {
				if continueFlow := filterSpawnEntriesByParent(filtered, "Continue"); len(continueFlow) > 0 {
					filtered = continueFlow
				}
			}
			entries = filtered
			currentRunID = run.ID
			currentCommand = run.Command
		}
	}
	if currentRunID == "" && state != nil && state.BuildStartedAt != nil {
		entries = filterSpawnEntriesSince(entries, *state.BuildStartedAt)
	}

	summary := spawnActivitySummary{
		Entries:        make([]agent.SpawnEntry, len(entries)),
		TotalCount:     len(entries),
		CurrentRunID:   currentRunID,
		CurrentCommand: currentCommand,
	}
	copy(summary.Entries, entries)
	sort.Slice(summary.Entries, func(i, j int) bool {
		return spawnEntryTimestamp(summary.Entries[i]).After(spawnEntryTimestamp(summary.Entries[j]))
	})

	for _, entry := range summary.Entries {
		switch {
		case agent.IsLiveSpawnStatus(entry.Status):
			summary.ActiveCount++
			summary.ActiveEntries = append(summary.ActiveEntries, entry)
		case agent.IsTerminalSpawnStatus(entry.Status):
			summary.RecentOutcomeEntries = append(summary.RecentOutcomeEntries, entry)
			switch entry.Status {
			case "completed", "completed_no_change", "manually-reconciled":
				summary.CompletedCount++
			case "blocked":
				summary.BlockedCount++
			case "failed", "timeout", "superseded":
				summary.FailedCount++
			}
		}
	}
	return summary
}

func filterSpawnEntriesSince(entries []agent.SpawnEntry, startedAt time.Time) []agent.SpawnEntry {
	if startedAt.IsZero() {
		return entries
	}
	filtered := make([]agent.SpawnEntry, 0, len(entries))
	for _, entry := range entries {
		ts := spawnEntryTimestamp(entry)
		if ts.IsZero() || ts.Before(startedAt) {
			continue
		}
		filtered = append(filtered, entry)
	}
	if len(filtered) == 0 {
		return entries
	}
	return filtered
}

func filterSpawnEntriesByParent(entries []agent.SpawnEntry, parent string) []agent.SpawnEntry {
	parent = strings.TrimSpace(parent)
	if parent == "" {
		return entries
	}
	filtered := make([]agent.SpawnEntry, 0, len(entries))
	for _, entry := range entries {
		if strings.TrimSpace(entry.ParentName) != parent {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}

func spawnEntryTimestamp(entry agent.SpawnEntry) time.Time {
	raw := entry.ActivityTimestamp
	if strings.TrimSpace(raw) == "" {
		raw = entry.Timestamp
	}
	ts, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}
	}
	return ts
}

func renderActiveWorkers(b *strings.Builder, entries []agent.SpawnEntry) {
	renderSpawnEntrySection(b, entries, 6, "active workers")
}

func renderRecentWorkerOutcomes(b *strings.Builder, entries []agent.SpawnEntry) {
	renderSpawnEntrySection(b, entries, 6, "recent outcomes")
}

func renderSpawnEntrySection(b *strings.Builder, entries []agent.SpawnEntry, maxEntries int, overflowLabel string) {
	limit := len(entries)
	if limit > maxEntries {
		limit = maxEntries
	}
	for i := 0; i < limit; i++ {
		renderSpawnEntry(b, entries[i])
	}
	if len(entries) > limit {
		fmt.Fprintf(b, "   ... and %d more %s\n", len(entries)-limit, overflowLabel)
	}
}

func renderSpawnActivity(b *strings.Builder, summary spawnActivitySummary) {
	if summary.TotalCount == 0 {
		b.WriteString("   No worker activity recorded\n")
		return
	}

	parts := []string{
		fmt.Sprintf("%d active", summary.ActiveCount),
		fmt.Sprintf("%d completed", summary.CompletedCount),
		fmt.Sprintf("%d blocked", summary.BlockedCount),
	}
	if summary.FailedCount > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", summary.FailedCount))
	}
	fmt.Fprintf(b, "   %s\n", strings.Join(parts, " | "))
	if command := strings.TrimSpace(summary.CurrentCommand); command != "" {
		fmt.Fprintf(b, "   Current run: %s", command)
		if runID := strings.TrimSpace(summary.CurrentRunID); runID != "" {
			fmt.Fprintf(b, " (%s)", runID)
		}
		b.WriteString("\n")
	}
}

func withoutLiveSpawnEntries(summary spawnActivitySummary) spawnActivitySummary {
	filtered := spawnActivitySummary{
		Entries:              make([]agent.SpawnEntry, 0, len(summary.Entries)),
		RecentOutcomeEntries: make([]agent.SpawnEntry, 0, len(summary.RecentOutcomeEntries)),
		CurrentRunID:         summary.CurrentRunID,
		CurrentCommand:       summary.CurrentCommand,
	}
	for _, entry := range summary.Entries {
		switch entry.Status {
		case "completed", "completed_no_change", "manually-reconciled":
			filtered.CompletedCount++
		case "blocked":
			filtered.BlockedCount++
		case "failed", "timeout", "superseded":
			filtered.FailedCount++
		case "skipped":
		default:
			continue
		}
		filtered.Entries = append(filtered.Entries, entry)
		filtered.RecentOutcomeEntries = append(filtered.RecentOutcomeEntries, entry)
	}
	filtered.TotalCount = len(filtered.Entries)
	filtered.ActiveEntries = []agent.SpawnEntry{}
	return filtered
}

func renderSpawnEntry(b *strings.Builder, entry agent.SpawnEntry) {
	fmt.Fprintf(b, "   %s %s %s — %s [%s]\n",
		dispatchStatusIcon(entry.Status),
		casteIdentity(entry.Caste),
		entry.AgentName,
		entry.Task,
		entry.Status,
	)
	if summary := strings.TrimSpace(entry.Summary); summary != "" && summary != strings.TrimSpace(entry.Task) {
		fmt.Fprintf(b, "      %s\n", summary)
	}
}

// generateProgressBar creates a Unicode progress bar string.
// Uses block characters: filled = \u2588, empty = \u2591.
func generateProgressBar(current, total, width int) string {
	if total == 0 {
		return "[" + strings.Repeat("\u2591", width) + "]"
	}
	if current > total {
		current = total
	}
	filled := width * current / total
	if filled > width {
		filled = width
	}
	return "[" + strings.Repeat("\u2588", filled) + strings.Repeat("\u2591", width-filled) + "]"
}

// countConstraints loads constraints.json and returns focus and avoid counts.
func countConstraints(s *storage.Store) (focus, avoid int) {
	// constraints.json is currently an empty object {}
	// Future: parse actual constraints when schema is defined
	return 0, 0
}

// countFlags preserves the existing status counting surface while delegating
// blocker truth to the shared snapshot reader.
func countFlags(s *storage.Store) (blockers, issues, notes int) {
	snapshot, err := readBlockerSnapshot(s)
	issues, notes = countStatusNonBlockerFlags(s)
	if err != nil {
		return 0, issues, notes
	}
	return snapshot.Count, issues, notes
}

func countStatusNonBlockerFlags(s *storage.Store) (issues, notes int) {
	flags, ok := loadFlagsFile(s)
	if !ok {
		return 0, 0
	}
	for _, f := range flags.Decisions {
		if f.Resolved {
			continue
		}
		switch f.Type {
		case "blocker":
			continue
		case "issue":
			issues++
		default:
			notes++
		}
	}
	return issues, notes
}

func loadFlagsFile(s *storage.Store) (colony.FlagsFile, bool) {
	if s == nil {
		return colony.FlagsFile{}, false
	}
	var flags colony.FlagsFile
	if err := s.LoadJSON("pending-decisions.json", &flags); err == nil {
		return flags, true
	}
	if err := s.LoadJSON("flags.json", &flags); err == nil {
		return flags, true
	}
	return colony.FlagsFile{}, false
}

func compactActionText(text string, limit int) string {
	text = strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
	if limit <= 0 || len(text) <= limit {
		return text
	}
	if limit <= 3 {
		return text[:limit]
	}
	return text[:limit-3] + "..."
}

func quoteCommandArg(text string) string {
	text = compactActionText(text, 140)
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "`", "'", "\n", " ", "\r", " ")
	return `"` + replacer.Replace(text) + `"`
}

// depthLabel maps colony depth to a human-readable description.
func depthLabel(depth string) string {
	switch depth {
	case "light":
		return "light (Builder only)"
	case "standard":
		return "standard (Builder + Scout)"
	case "deep":
		return "deep (Builder + Scout + Oracle)"
	case "full":
		return "full (All agents)"
	default:
		return depth
	}
}

// granularityLabel maps plan granularity to a human-readable description.
func granularityLabel(granularity string) string {
	switch granularity {
	case "sprint":
		return "sprint (1-3 phases)"
	case "milestone":
		return "milestone (4-7 phases)"
	case "quarter":
		return "quarter (8-12 phases)"
	case "major":
		return "major (13-20 phases)"
	default:
		return granularity
	}
}

// renderMemoryHealthTable writes memory health as classic labeled lines.
func renderMemoryHealthTable(b *strings.Builder, s *storage.Store) {
	summary := loadMemoryHealthSummary(s)

	writeLine := func(emoji, label string, count int, ts string) {
		fmt.Fprintf(b, "   %s %s: %d", emoji, label, count)
		if formatted := formatTimestamp(ts); formatted != "" {
			fmt.Fprintf(b, " (updated %s)", formatted)
		}
		b.WriteString("\n")
	}
	writeLine("🧠", "Wisdom entries", summary.WisdomTotal, summary.LastLearning)
	writeLine("📤", "Pending promotions", summary.PendingPromotions, summary.LastLearning)
	writeLine("🐜", "Applied instincts (lessons learned)", summary.AppliedInstincts, summary.LastInstinctTouched)
	writeLine("👀", "Needs review", summary.ReviewCandidates+summary.RereadCandidates, summary.LastInstinctTouched)
	writeLine("🗑", "Recent failures", summary.RecentFailures, summary.LastFailure)
}

// hasReviewFindings checks whether any review domain has non-zero findings.
func hasReviewFindings(s *storage.Store) bool {
	for _, d := range colony.DomainOrder {
		ledgerPath := fmt.Sprintf("reviews/%s/ledger.json", d)
		var lf colony.ReviewLedgerFile
		if err := s.LoadJSON(ledgerPath, &lf); err != nil {
			continue
		}
		if lf.Summary.Total > 0 {
			return true
		}
	}
	return false
}

// renderReviewFindingsTable writes a table of review findings per domain.
func renderReviewFindingsTable(b *strings.Builder, s *storage.Store) {
	for _, d := range colony.DomainOrder {
		ledgerPath := fmt.Sprintf("reviews/%s/ledger.json", d)
		var lf colony.ReviewLedgerFile
		if err := s.LoadJSON(ledgerPath, &lf); err != nil {
			continue
		}
		if lf.Summary.Total == 0 {
			continue
		}
		icon := "👀"
		if lf.Summary.Open == 0 {
			icon = "✅"
		}
		fmt.Fprintf(b, "   %s %s: %d open, %d resolved (%d total)\n", icon, d, lf.Summary.Open, lf.Summary.Resolved, lf.Summary.Total)
	}
}

// renderPheromoneSummary writes the pheromone summary table to the builder.
func renderPheromoneSummary(b *strings.Builder, s *storage.Store) {
	var pf colony.PheromoneFile
	if err := s.LoadJSON("pheromones.json", &pf); err != nil {
		b.WriteString("   No pheromone data available\n")
		return
	}

	now := time.Now().UTC()
	type pheromoneRow struct {
		Type     string
		Strength float64
		Life     string
		Signal   string
	}
	rows := []pheromoneRow{}
	for _, sig := range pf.Signals {
		if !sig.Active {
			continue
		}
		rows = append(rows, pheromoneRow{
			Type:     sig.Type,
			Strength: computeEffectiveStrength(sig, now),
			Life:     signalLifetimeSummary(sig, now),
			Signal:   extractContentText(sig.Content),
		})
	}

	if len(rows) == 0 {
		b.WriteString("   No active signals\n")
		return
	}

	sort.Slice(rows, func(i, j int) bool {
		if signalPriority(rows[i].Type) != signalPriority(rows[j].Type) {
			return signalPriority(rows[i].Type) < signalPriority(rows[j].Type)
		}
		if rows[i].Strength != rows[j].Strength {
			return rows[i].Strength > rows[j].Strength
		}
		return rows[i].Signal < rows[j].Signal
	})

	// Classic house style: one emoji-prefixed line per signal, grouped by
	// priority order — not a bordered machine table.
	for _, row := range rows {
		signal := row.Signal
		if signal == "" {
			signal = "(no content)"
		}
		if len(signal) > 60 {
			signal = signal[:57] + "..."
		}
		emoji := signalTypeGlyph(row.Type)
		fmt.Fprintf(b, "   %s [%d%%] %q — %s\n", emoji, int(math.Round(row.Strength*100)), signal, row.Life)
	}
	b.WriteString("   Strength fades over time; run `aether pheromone-display` for the full view.\n")
}

func renderStrongestInstincts(b *strings.Builder, instincts []colony.Instinct) {
	for _, inst := range instincts {
		domain := inst.Domain
		if domain == "" {
			domain = "general"
		}
		fmt.Fprintf(b, "   [%.2f] %s: %s\n", inst.Confidence, domain, inst.Action)
	}
}

// renderVersionLine writes the runtime version line to the dashboard.
func renderVersionLine(b *strings.Builder) {
	binaryVersion := resolveVersion()
	hubVersion := readInstalledHubVersion()
	if hubVersion != "" {
		if binaryVersion != hubVersion {
			fmt.Fprintf(b, "%s\n\n", voiceLine("warning", fmt.Sprintf("Runtime: %s | Hub (the installed copy on this machine): %s  MISMATCH", binaryVersion, hubVersion)))
		} else {
			fmt.Fprintf(b, "%s\n\n", voiceLine("status", fmt.Sprintf("Runtime: %s | Hub (the installed copy on this machine): %s", binaryVersion, hubVersion)))
		}
	} else {
		fmt.Fprintf(b, "Runtime: %s\n\n", binaryVersion)
	}
}

// renderSignalSummaryLine writes a one-line signal summary with expiry awareness.
func renderSignalSummaryLine(b *strings.Builder, s *storage.Store) {
	var pf colony.PheromoneFile
	if err := s.LoadJSON("pheromones.json", &pf); err != nil {
		return // no pheromones -- skip silently
	}
	focusCount, redirectCount, feedbackCount := 0, 0, 0
	for _, sig := range pf.Signals {
		if !sig.Active {
			continue
		}
		switch sig.Type {
		case "FOCUS":
			focusCount++
		case "REDIRECT":
			redirectCount++
		case "FEEDBACK":
			feedbackCount++
		}
	}
	total := focusCount + redirectCount + feedbackCount
	if total == 0 {
		return
	}
	var parts []string
	if focusCount > 0 {
		parts = append(parts, fmt.Sprintf("%d FOCUS expire at seal", focusCount))
	}
	if redirectCount > 0 {
		parts = append(parts, fmt.Sprintf("%d REDIRECT persists", redirectCount))
	}
	if feedbackCount > 0 {
		parts = append(parts, fmt.Sprintf("%d FEEDBACK", feedbackCount))
	}
	fmt.Fprintf(b, "%s\n", voiceLine("focus", fmt.Sprintf("Signals: %d active (%s)", total, strings.Join(parts, ", "))))
}

// extractContentText extracts the text field from a json.RawMessage content.
func extractContentText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return string(raw)
	}
	if text, ok := m["text"].(string); ok {
		return text
	}
	return ""
}

// formatTimestamp converts an RFC3339 timestamp to a shorter display format.
func formatTimestamp(ts string) string {
	if ts == "" {
		return "-"
	}
	// Try parsing RFC3339
	parsed := strings.ReplaceAll(ts, "T", " ")
	// Remove timezone info for display
	if idx := strings.Index(parsed, "+"); idx > 0 {
		parsed = parsed[:idx]
	}
	if idx := strings.Index(parsed, "Z"); idx > 0 {
		parsed = parsed[:idx]
	}
	// Trim seconds for cleaner display
	if len(parsed) > 16 {
		parsed = parsed[:16]
	}
	return parsed
}

// setupTestStore creates a temporary directory with .aether/data/ and copies
// test fixtures from cmd/testdata/. Returns the store and the temp dir path.
func setupTestStore(t interface{ Fatal(...interface{}) }) (*storage.Store, string) {
	return setupTestStoreWithName(t, "")
}

// setupTestStoreWithName creates a test store, optionally using a named subdirectory
// from cmd/testdata/.
func setupTestStoreWithName(t interface{ Fatal(...interface{}) }, name string) (*storage.Store, string) {
	tmpDir, err := os.MkdirTemp("", "aether-status-test-*")
	if err != nil {
		t.Fatal(err)
	}
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Copy test fixtures (Go tests run from the package directory, so testdata/ is relative to cmd/)
	testdataDir := "testdata"
	if name != "" {
		testdataDir = "testdata/" + name
	}

	entries, err := os.ReadDir(testdataDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		src := testdataDir + "/" + entry.Name()
		dst := dataDir + "/" + entry.Name()
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, data, 0644); err != nil {
			t.Fatal(err)
		}
	}

	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	return s, tmpDir
}
