package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Display colony dashboard",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		state, err := loadActiveColonyState()
		if err != nil {
			if shouldRenderVisualOutput(stdout) && strings.Contains(colonyStateLoadMessage(err), "No colony initialized") {
				fmt.Fprint(stdout, renderNoColonyStatusVisual())
				return nil
			}
			fmt.Fprintln(stdout, colonyStateLoadMessage(err))
			return nil
		}

		output := renderDashboard(state, store)
		fmt.Fprint(stdout, output)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func renderNoColonyStatusVisual() string {
	var b strings.Builder
	b.WriteString(renderBanner("📊", "Colony Status"))
	b.WriteString(visualDivider)
	b.WriteString("No colony initialized in this repo.\n")
	b.WriteString(renderNextUp(
		`Run `+"`aether init \"goal\"`"+` to start a colony.`,
		`Run `+"`aether lay-eggs`"+` first if this repo has not been set up for Aether yet.`,
	))
	return b.String()
}

// renderDashboard produces the full colony status dashboard string.
func renderDashboard(state colony.ColonyState, s *storage.Store) string {
	var b strings.Builder

	// Banner
	b.WriteString(renderBanner("📊", "Colony Status"))
	b.WriteString(visualDivider)

	// Goal (truncated to 60 chars)
	goal := *state.Goal
	if len(goal) > 60 {
		goal = goal[:57] + "..."
	}
	fmt.Fprintf(&b, "Goal: %s\n\n", goal)

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
	fmt.Fprintf(&b, "Progress\n")
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
	fmt.Fprintf(&b, "   Phase: [Phase %d/%d] %s %d%%\n", phasePosition, totalPhases, phaseBar, phasePercent)

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
		fmt.Fprintf(&b, "   Tasks: [Tasks %d/%d] %s %d%% in Phase %d (%s)\n\n", tasksCompleted, tasksTotal, taskBar, taskPercent, displayPhaseNum, phaseName)
	} else {
		fmt.Fprintf(&b, "   Tasks: [Tasks %d/%d] %s %d%% in Phase %d\n\n", tasksCompleted, tasksTotal, taskBar, taskPercent, displayPhaseNum)
	}

	// Constraints
	focusCount, avoidCount := countConstraints(s)
	fmt.Fprintf(&b, "Focus: %d areas | Avoid: %d patterns\n", focusCount, avoidCount)

	// Instincts
	instincts := loadRuntimeInstincts(s, &state)
	totalInstincts := len(instincts)
	highConf := 0
	for _, inst := range instincts {
		if inst.Confidence >= 0.7 {
			highConf++
		}
	}
	fmt.Fprintf(&b, "Instincts: %d learned (%d strong)\n", totalInstincts, highConf)

	// Flags
	blockers, issues, notes := countFlags(s)
	fmt.Fprintf(&b, "Flags: %d blockers | %d issues | %d notes\n", blockers, issues, notes)

	// Scope
	fmt.Fprintf(&b, "Scope: %s\n", state.EffectiveScope())

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
	fmt.Fprintf(&b, "Depth: %s\n", depthLbl)

	// Granularity
	granularity := string(state.PlanGranularity)
	if granularity == "" {
		granularity = "not set"
	}
	granLbl := granularityLabel(granularity)
	fmt.Fprintf(&b, "Granularity: %s\n", granLbl)

	// Parallel mode
	parallelMode := string(state.ParallelMode)
	if parallelMode == "" {
		parallelMode = "in-repo"
	}
	fmt.Fprintf(&b, "Parallel: %s\n\n", parallelMode)

	proof := buildProofOutput(skillWorkspaceRoot(), state)
	b.WriteString("Proof\n")
	fmt.Fprintf(&b, "   Context: %s | %d included | %d preserved | %d trimmed | %d blocked\n",
		proof.Summary.ContextSurface,
		proof.Summary.ContextIncluded,
		proof.Summary.ContextPreserved,
		proof.Summary.ContextTrimmed,
		proof.Summary.ContextBlocked,
	)
	if proof.Summary.SkillDispatches > 0 {
		fmt.Fprintf(&b, "   Skills: %s | %d dispatches | %d matched skills\n",
			proof.Summary.SkillSource,
			proof.Summary.SkillDispatches,
			proof.Summary.SkillMatchedTotal,
		)
	} else {
		b.WriteString("   Skills: no phase-aware skill proof yet\n")
	}
	b.WriteString("   Inspect: aether proof\n")
	b.WriteString("\n")

	// Memory Health table
	b.WriteString("Memory Health\n")
	renderMemoryHealthTable(&b, s)

	// Pheromone Summary
	b.WriteString("\nActive Pheromones\n")
	renderPheromoneSummary(&b, s)

	spawnSummary := loadSpawnActivitySummary(s)
	liveSpawnView := state.State == colony.StateEXECUTING && !state.Paused && state.BuildStartedAt != nil
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

	if totalInstincts > 0 {
		recentInstincts := loadRecentRuntimeInstincts(s, &state, 3)
		if len(recentInstincts) > 0 {
			b.WriteString("\nRecent Instincts\n")
			renderRecentInstincts(&b, recentInstincts)
		}
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
		b.WriteString(renderNextUp(
			"Active workers are still running. Wait for the in-flight command to finish.",
			`Run `+"`aether proof`"+` to inspect the active context and skill proof.`,
			`Run `+"`aether status`"+` again to refresh the spawn view.`,
			`Run `+"`tail -f .aether/data/spawn-tree.txt`"+` in another terminal to watch status changes.`,
		))
		return b.String()
	}
	primary, alternatives := workflowSuggestionsForState(state)
	alternatives = append(alternatives, `Run `+"`aether proof`"+` to inspect the current context and skill proof.`)
	b.WriteString(renderNextUp(primary, alternatives...))

	return b.String()
}

func loadActiveSpawnEntries(s *storage.Store) []agent.SpawnEntry {
	return loadSpawnActivitySummary(s).ActiveEntries
}

type spawnActivitySummary struct {
	Entries        []agent.SpawnEntry
	ActiveEntries  []agent.SpawnEntry
	TotalCount     int
	ActiveCount    int
	CompletedCount int
	BlockedCount   int
	FailedCount    int
}

func loadSpawnActivitySummary(s *storage.Store) spawnActivitySummary {
	if s == nil {
		return spawnActivitySummary{}
	}

	entries, err := agent.NewSpawnTree(s, "spawn-tree.txt").Parse()
	if err != nil || len(entries) == 0 {
		return spawnActivitySummary{}
	}

	summary := spawnActivitySummary{
		Entries:    make([]agent.SpawnEntry, len(entries)),
		TotalCount: len(entries),
	}
	copy(summary.Entries, entries)
	sort.Slice(summary.Entries, func(i, j int) bool {
		return spawnEntryTimestamp(summary.Entries[i]).After(spawnEntryTimestamp(summary.Entries[j]))
	})

	for _, entry := range summary.Entries {
		switch entry.Status {
		case "spawned", "active":
			summary.ActiveCount++
			summary.ActiveEntries = append(summary.ActiveEntries, entry)
		case "completed":
			summary.CompletedCount++
		case "blocked":
			summary.BlockedCount++
		case "failed":
			summary.FailedCount++
		}
	}
	return summary
}

func spawnEntryTimestamp(entry agent.SpawnEntry) time.Time {
	ts, err := time.Parse(time.RFC3339, entry.Timestamp)
	if err != nil {
		return time.Time{}
	}
	return ts
}

func renderActiveWorkers(b *strings.Builder, entries []agent.SpawnEntry) {
	limit := len(entries)
	if limit > 6 {
		limit = 6
	}
	for i := 0; i < limit; i++ {
		entry := entries[i]
		fmt.Fprintf(b, "   %s %s (%s) — %s [%s]\n",
			casteEmoji(entry.Caste),
			entry.AgentName,
			entry.Caste,
			entry.Task,
			entry.Status,
		)
	}
	if len(entries) > limit {
		fmt.Fprintf(b, "   ... and %d more active workers\n", len(entries)-limit)
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

	limit := len(summary.Entries)
	if limit > 6 {
		limit = 6
	}
	for i := 0; i < limit; i++ {
		entry := summary.Entries[i]
		fmt.Fprintf(b, "   %s %s (%s) — %s [%s]\n",
			casteEmoji(entry.Caste),
			entry.AgentName,
			entry.Caste,
			entry.Task,
			entry.Status,
		)
	}
	if len(summary.Entries) > limit {
		fmt.Fprintf(b, "   ... and %d more recent workers\n", len(summary.Entries)-limit)
	}
}

func withoutLiveSpawnEntries(summary spawnActivitySummary) spawnActivitySummary {
	filtered := spawnActivitySummary{
		Entries: make([]agent.SpawnEntry, 0, len(summary.Entries)),
	}
	for _, entry := range summary.Entries {
		switch entry.Status {
		case "completed":
			filtered.CompletedCount++
		case "blocked":
			filtered.BlockedCount++
		case "failed":
			filtered.FailedCount++
		default:
			continue
		}
		filtered.Entries = append(filtered.Entries, entry)
	}
	filtered.TotalCount = len(filtered.Entries)
	filtered.ActiveEntries = []agent.SpawnEntry{}
	return filtered
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

// countFlags loads flags.json and counts by type.
func countFlags(s *storage.Store) (blockers, issues, notes int) {
	var flags colony.FlagsFile
	if err := s.LoadJSON("pending-decisions.json", &flags); err != nil {
		// Try alternate name
		if err2 := s.LoadJSON("flags.json", &flags); err2 != nil {
			return 0, 0, 0
		}
	}
	for _, f := range flags.Decisions {
		switch f.Type {
		case "blocker":
			blockers++
		case "issue":
			issues++
		default:
			notes++
		}
	}
	return
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

// renderMemoryHealthTable writes the memory health table to the builder.
func renderMemoryHealthTable(b *strings.Builder, s *storage.Store) {
	summary := loadMemoryHealthSummary(s)

	t := table.NewWriter()
	t.AppendHeader(table.Row{"Metric", "Count", "Last Updated"})
	t.AppendRow(table.Row{"Wisdom Entries", summary.WisdomTotal, formatTimestamp(summary.LastLearning)})
	t.AppendRow(table.Row{"Pending Promos", summary.PendingPromotions, formatTimestamp(summary.LastLearning)})
	t.AppendRow(table.Row{"Applied Instincts", summary.AppliedInstincts, formatTimestamp(summary.LastInstinctTouched)})
	t.AppendRow(table.Row{"Needs Review", summary.ReviewCandidates + summary.RereadCandidates, formatTimestamp(summary.LastInstinctTouched)})
	t.AppendRow(table.Row{"Recent Failures", summary.RecentFailures, formatTimestamp(summary.LastFailure)})
	t.SetStyle(table.StyleRounded)
	b.WriteString(t.Render() + "\n")
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

	t := table.NewWriter()
	t.AppendHeader(table.Row{"Type", "Strength", "Life", "Signal"})
	b.WriteString("   Strength reflects the active decay-adjusted signal weight.\n")
	b.WriteString("   Life shows remaining expiry or decay context for each signal.\n")

	for _, row := range rows {
		signal := row.Signal
		if signal == "" {
			signal = "none"
		}
		if len(signal) > 44 {
			signal = signal[:41] + "..."
		}
		life := row.Life
		if len(life) > 26 {
			life = life[:23] + "..."
		}
		t.AppendRow(table.Row{row.Type, fmt.Sprintf("%.2f", row.Strength), life, signal})
	}

	t.SetStyle(table.StyleRounded)
	b.WriteString(t.Render() + "\n")
}

func renderRecentInstincts(b *strings.Builder, instincts []colony.Instinct) {
	for _, inst := range instincts {
		domain := inst.Domain
		if domain == "" {
			domain = "general"
		}
		fmt.Fprintf(b, "   [%.2f] %s: %s\n", inst.Confidence, domain, inst.Action)
	}
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
