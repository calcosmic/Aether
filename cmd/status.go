package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

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
		if store == nil {
			fmt.Fprintln(stdout, "No colony initialized. Run aether init first.")
			return nil
		}

		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			fmt.Fprintln(stdout, "No colony initialized. Run aether init first.")
			return nil
		}

		if state.Goal == nil {
			fmt.Fprintln(stdout, "No colony initialized. Run aether init first.")
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

// renderDashboard produces the full colony status dashboard string.
func renderDashboard(state colony.ColonyState, s *storage.Store) string {
	var b strings.Builder

	// Banner
	b.WriteString("       .-.\n")
	b.WriteString("      (o o)  AETHER COLONY\n")
	b.WriteString("      | O |  Status Report\n")
	b.WriteString("       `-'\n")
	b.WriteString("========================================\n\n")

	// Goal (truncated to 60 chars)
	goal := *state.Goal
	if len(goal) > 60 {
		goal = goal[:57] + "..."
	}
	fmt.Fprintf(&b, "Goal: %s\n\n", goal)

	// Progress
	totalPhases := len(state.Plan.Phases)
	phaseBar := generateProgressBar(state.CurrentPhase, totalPhases, 20)
	fmt.Fprintf(&b, "Progress\n")
	fmt.Fprintf(&b, "   Phase: %s %d/%d phases\n", phaseBar, state.CurrentPhase, totalPhases)

	// Task progress in current phase
	var tasksCompleted, tasksTotal int
	var phaseName string
	if state.CurrentPhase > 0 && state.CurrentPhase <= totalPhases {
		phase := state.Plan.Phases[state.CurrentPhase-1]
		phaseName = phase.Name
		tasksTotal = len(phase.Tasks)
		for _, task := range phase.Tasks {
			if task.Status == "completed" {
				tasksCompleted++
			}
		}
	}
	taskBar := generateProgressBar(tasksCompleted, tasksTotal, 20)
	if phaseName != "" {
		fmt.Fprintf(&b, "   Tasks: %s %d/%d tasks in Phase %d (%s)\n\n", taskBar, tasksCompleted, tasksTotal, state.CurrentPhase, phaseName)
	} else {
		fmt.Fprintf(&b, "   Tasks: %s %d/%d tasks in Phase %d\n\n", taskBar, tasksCompleted, tasksTotal, state.CurrentPhase)
	}

	// Constraints
	focusCount, avoidCount := countConstraints(s)
	fmt.Fprintf(&b, "Focus: %d areas | Avoid: %d patterns\n", focusCount, avoidCount)

	// Instincts
	totalInstincts := len(state.Memory.Instincts)
	highConf := 0
	for _, inst := range state.Memory.Instincts {
		if inst.Confidence >= 0.7 {
			highConf++
		}
	}
	fmt.Fprintf(&b, "Instincts: %d learned (%d strong)\n", totalInstincts, highConf)

	// Flags
	blockers, issues, notes := countFlags(s)
	fmt.Fprintf(&b, "Flags: %d blockers | %d issues | %d notes\n", blockers, issues, notes)

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
	fmt.Fprintf(&b, "Granularity: %s\n\n", granLbl)

	// Memory Health table
	b.WriteString("Memory Health\n")
	renderMemoryHealthTable(&b, s)

	// Pheromone Summary
	b.WriteString("\nActive Pheromones\n")
	renderPheromoneSummary(&b, s)

	// Top instincts
	if totalInstincts > 0 {
		b.WriteString("\nColony Instincts:\n")
		renderTopInstincts(&b, state.Memory.Instincts)
	}

	// State
	fmt.Fprintf(&b, "\nState: %s\n", state.State)

	return b.String()
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
	// Try loading learning observations
	var wisdomTotal, pendingTotal int
	var lastLearning string

	var learnings colony.LearningFile
	if err := s.LoadJSON("learning-observations.json", &learnings); err == nil {
		wisdomTotal = len(learnings.Observations)
		if wisdomTotal > 0 {
			lastLearning = learnings.Observations[wisdomTotal-1].LastSeen
		}
	}

	// Try loading midden for failure count
	var failureCount int
	var lastFailure string
	var midden colony.MiddenFile
	if err := s.LoadJSON("midden/midden.json", &midden); err == nil {
		failureCount = len(midden.Entries)
		if failureCount > 0 {
			lastFailure = midden.Entries[failureCount-1].Timestamp
		}
	}

	// Format timestamps
	lastLearningFormatted := formatTimestamp(lastLearning)
	lastFailureFormatted := formatTimestamp(lastFailure)

	t := table.NewWriter()
	t.AppendHeader(table.Row{"Metric", "Count", "Last Updated"})
	t.AppendRow(table.Row{"Wisdom Entries", wisdomTotal, lastLearningFormatted})
	t.AppendRow(table.Row{"Pending Promos", pendingTotal, lastLearningFormatted})
	t.AppendRow(table.Row{"Recent Failures", failureCount, lastFailureFormatted})
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

	// Group signals by type
	typeCounts := make(map[string]int)
	typeStrongest := make(map[string]string)

	for _, sig := range pf.Signals {
		if !sig.Active {
			continue
		}
		typeCounts[sig.Type]++
		// Track strongest signal
		content := extractContentText(sig.Content)
		if existing, ok := typeStrongest[sig.Type]; !ok || content != "" {
			if !ok || len(content) > len(existing) {
				typeStrongest[sig.Type] = content
			}
		}
	}

	if len(typeCounts) == 0 {
		b.WriteString("   No active signals\n")
		return
	}

	t := table.NewWriter()
	t.AppendHeader(table.Row{"Type", "Count", "Strongest Signal"})

	// Display in consistent order: FOCUS, REDIRECT, FEEDBACK
	for _, sigType := range []string{"FOCUS", "REDIRECT", "FEEDBACK"} {
		count := typeCounts[sigType]
		if count == 0 {
			continue
		}
		strongest := typeStrongest[sigType]
		if strongest == "" {
			strongest = "none"
		}
		if len(strongest) > 30 {
			strongest = strongest[:27] + "..."
		}
		t.AppendRow(table.Row{sigType, count, strongest})
	}

	t.SetStyle(table.StyleRounded)
	b.WriteString(t.Render() + "\n")
}

// renderTopInstincts shows the top 3 instincts sorted by confidence.
func renderTopInstincts(b *strings.Builder, instincts []colony.Instinct) {
	// Sort by confidence descending
	sorted := make([]colony.Instinct, len(instincts))
	copy(sorted, instincts)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Confidence > sorted[j].Confidence
	})

	limit := 3
	if len(sorted) < limit {
		limit = len(sorted)
	}
	for i := 0; i < limit; i++ {
		inst := sorted[i]
		domain := inst.Domain
		if domain == "" {
			domain = "general"
		}
		fmt.Fprintf(b, "   [%.1f] %s: %s\n", inst.Confidence, domain, inst.Action)
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
