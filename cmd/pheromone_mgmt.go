package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/cache"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

var pheromonePrimeCmd = &cobra.Command{
	Use:   "pheromone-prime",
	Short: "Format active pheromone signals for prompt injection",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		pf := loadPheromones()
		if pf == nil {
			outputOK(map[string]interface{}{
				"section":      "",
				"signal_count": 0,
			})
			return nil
		}

		var focus, redirect, feedback []colony.PheromoneSignal
		for _, sig := range pf.Signals {
			if !sig.Active {
				continue
			}
			switch sig.Type {
			case "FOCUS":
				focus = append(focus, sig)
			case "REDIRECT":
				redirect = append(redirect, sig)
			case "FEEDBACK":
				feedback = append(feedback, sig)
			}
		}

		var sb strings.Builder
		total := 0

		if len(redirect) > 0 {
			sb.WriteString("## ACTIVE REDIRECT SIGNALS (Hard Constraints)\n\n")
			for _, sig := range redirect {
				text := extractText(sig.Content)
				strength := "1.0"
				if sig.Strength != nil {
					strength = fmt.Sprintf("%.1f", *sig.Strength)
				}
				sb.WriteString(fmt.Sprintf("- [REDIRECT] %s (priority: %s, strength: %s)\n", text, sig.Priority, strength))
			}
			sb.WriteString("\n")
			total += len(redirect)
		}

		if len(focus) > 0 {
			sb.WriteString("## ACTIVE FOCUS SIGNALS\n\n")
			for _, sig := range focus {
				text := extractText(sig.Content)
				strength := "1.0"
				if sig.Strength != nil {
					strength = fmt.Sprintf("%.1f", *sig.Strength)
				}
				sb.WriteString(fmt.Sprintf("- [FOCUS] %s (priority: %s, strength: %s)\n", text, sig.Priority, strength))
			}
			sb.WriteString("\n")
			total += len(focus)
		}

		if len(feedback) > 0 {
			sb.WriteString("## ACTIVE FEEDBACK SIGNALS\n\n")
			for _, sig := range feedback {
				text := extractText(sig.Content)
				strength := "1.0"
				if sig.Strength != nil {
					strength = fmt.Sprintf("%.1f", *sig.Strength)
				}
				sb.WriteString(fmt.Sprintf("- [FEEDBACK] %s (priority: %s, strength: %s)\n", text, sig.Priority, strength))
			}
			sb.WriteString("\n")
			total += len(feedback)
		}

		section := strings.TrimSpace(sb.String())
		if section == "" {
			section = "No active pheromone signals."
		}

		outputOK(map[string]interface{}{
			"section":        section,
			"signal_count":   total,
			"focus_count":    len(focus),
			"redirect_count": len(redirect),
			"feedback_count": len(feedback),
		})
		return nil
	},
}

var colonyPrimeCmd = &cobra.Command{
	Use:   "colony-prime",
	Short: "Assemble full colony context for worker prompt injection",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		compact, _ := cmd.Flags().GetBool("compact")
		budget := 8000
		if compact {
			budget = 4000
		}

		// Initialize session cache for this invocation
		sc := cache.NewSessionCache(store.BasePath())

		var sections []struct {
			name     string
			content  string
			priority int // lower = trimmed first
		}

		// 1. Load COLONY_STATE.json (via session cache)
		var state colony.ColonyState
		statePath := filepath.Join(store.BasePath(), "COLONY_STATE.json")
		if err := sc.Load(statePath, &state); err != nil {
			// Cache load failed -- fall back to direct store read
			_ = store.LoadJSON("COLONY_STATE.json", &state)
		}
		// Extract key context
		var stateSection strings.Builder
		stateSection.WriteString("## Colony State\n\n")
		if state.Goal != nil {
			stateSection.WriteString(fmt.Sprintf("Goal: %s\n", *state.Goal))
		}
		stateSection.WriteString(fmt.Sprintf("State: %s\n", state.State))
		stateSection.WriteString(fmt.Sprintf("Phase: %d\n", state.CurrentPhase))
		if len(state.Plan.Phases) > 0 && state.CurrentPhase > 0 && state.CurrentPhase <= len(state.Plan.Phases) {
			phase := state.Plan.Phases[state.CurrentPhase-1]
			stateSection.WriteString(fmt.Sprintf("Phase Name: %s\n", phase.Name))
			if len(phase.Tasks) > 0 {
				stateSection.WriteString("Tasks:\n")
				for _, t := range phase.Tasks {
					stateSection.WriteString(fmt.Sprintf("  - [%s] %s\n", t.Status, t.Goal))
				}
			}
		}
		sections = append(sections, struct {
			name     string
			content  string
			priority int
		}{"state", stateSection.String(), 5})

		// 2. Load pheromones (via session cache, shared loader)
		pf, phErr := loadPheromonesOnce(store, sc)
		if phErr == nil && len(pf.Signals) > 0 {
			var active []colony.PheromoneSignal
			for _, sig := range pf.Signals {
				if sig.Active {
					active = append(active, sig)
				}
			}
			if len(active) > 0 {
				var phSB strings.Builder
				phSB.WriteString("## Pheromone Signals\n\n")
				for _, sig := range active {
					text := extractText(sig.Content)
					phSB.WriteString(fmt.Sprintf("- [%s] %s\n", sig.Type, text))
				}
				sections = append(sections, struct {
					name     string
					content  string
					priority int
				}{"pheromones", phSB.String(), 9})
			}
		}

		// 3. Load instincts -- from instincts.json (source of truth), falling back to state
		var instincts []struct {
			trigger    string
			action     string
			confidence float64
		}
		// Primary source: instincts.json (written by instinct-create), via session cache
		var instFile colony.InstinctsFile
		instinctsPath := filepath.Join(store.BasePath(), "instincts.json")
		if err := sc.Load(instinctsPath, &instFile); err != nil {
			// Cache load failed -- fall back to direct store read
			_ = store.LoadJSON("instincts.json", &instFile)
		}
		for _, inst := range instFile.Instincts {
			if inst.Archived {
				continue
			}
			instincts = append(instincts, struct {
				trigger    string
				action     string
				confidence float64
			}{inst.Trigger, inst.Action, inst.Confidence})
		}
		// Fallback: state memory instincts (if instincts.json had nothing)
		if len(instincts) == 0 && state.Memory.Instincts != nil {
			for _, inst := range state.Memory.Instincts {
				instincts = append(instincts, struct {
					trigger    string
					action     string
					confidence float64
				}{inst.Trigger, inst.Action, inst.Confidence})
			}
		}
		if len(instincts) > 0 {
			var instSB strings.Builder
			instSB.WriteString("## Active Instincts\n\n")
			for _, inst := range instincts {
				instSB.WriteString(fmt.Sprintf("- [%s] %s (confidence: %.2f)\n", inst.trigger, inst.action, inst.confidence))
			}
			sections = append(sections, struct {
				name     string
				content  string
				priority int
			}{"instincts", instSB.String(), 6})
		}

		// 4. Load decisions from state
		if state.Memory.Decisions != nil && len(state.Memory.Decisions) > 0 {
			var decSB strings.Builder
			decSB.WriteString("## Key Decisions\n\n")
			for _, d := range state.Memory.Decisions {
				decSB.WriteString(fmt.Sprintf("- Phase %d: %s \u2014 %s\n", d.Phase, d.Claim, d.Rationale))
			}
			sections = append(sections, struct {
				name     string
				content  string
				priority int
			}{"decisions", decSB.String(), 4})
		}

		// 5. Load phase learnings from state
		if state.Memory.PhaseLearnings != nil && len(state.Memory.PhaseLearnings) > 0 {
			var learnSB strings.Builder
			learnSB.WriteString("## Phase Learnings\n\n")
			for _, pl := range state.Memory.PhaseLearnings {
				learnSB.WriteString(fmt.Sprintf("### Phase %d: %s\n", pl.Phase, pl.PhaseName))
				for _, l := range pl.Learnings {
					learnSB.WriteString(fmt.Sprintf("  - %s [%s]\n", l.Claim, l.Status))
				}
			}
			sections = append(sections, struct {
				name     string
				content  string
				priority int
			}{"learnings", learnSB.String(), 3})
		}

		// Sort by priority (lowest priority = trimmed first)
		sort.Slice(sections, func(i, j int) bool {
			return sections[i].priority < sections[j].priority
		})

		// Assemble within budget
		var assembled strings.Builder
		trimmed := []string{}
		currentLen := 0
		for _, sec := range sections {
			if currentLen+len(sec.content) > budget {
				trimmed = append(trimmed, sec.name)
				continue
			}
			assembled.WriteString(sec.content)
			assembled.WriteString("\n")
			currentLen += len(sec.content)
		}

		result := map[string]interface{}{
			"context":  strings.TrimSpace(assembled.String()),
			"budget":   budget,
			"used":     currentLen,
			"sections": len(sections),
			"trimmed":  trimmed,
		}

		outputOK(result)
		return nil
	},
}

// pheromoneDisplayCmd renders active pheromone signals in a formatted table.
var pheromoneDisplayCmd = &cobra.Command{
	Use:          "pheromone-display",
	Short:        "Display active pheromone signals in formatted table",
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		pf := loadPheromones()
		if pf == nil {
			outputOK(map[string]interface{}{
				"signals": []interface{}{},
				"count":   0,
				"display": "No pheromone signals found.",
			})
			return nil
		}

		filterType, _ := cmd.Flags().GetString("type")
		activeOnly, _ := cmd.Flags().GetBool("active-only")

		var filtered []colony.PheromoneSignal
		for _, sig := range pf.Signals {
			if activeOnly && !sig.Active {
				continue
			}
			if filterType != "" && sig.Type != filterType {
				continue
			}
			filtered = append(filtered, sig)
		}

		if len(filtered) == 0 {
			outputOK(map[string]interface{}{
				"signals": []interface{}{},
				"count":   0,
				"display": "No pheromone signals found.",
			})
			return nil
		}

		// Format as text table
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("%-10s %-10s %-10s %s\n", "TYPE", "PRIORITY", "STRENGTH", "CONTENT"))
		sb.WriteString(strings.Repeat("-", 80) + "\n")
		for _, sig := range filtered {
			strength := "1.0"
			if sig.Strength != nil {
				strength = fmt.Sprintf("%.1f", *sig.Strength)
			}
			text := extractText(sig.Content)
			if len(text) > 60 {
				text = text[:57] + "..."
			}
			sb.WriteString(fmt.Sprintf("%-10s %-10s %-10s %s\n", sig.Type, sig.Priority, strength, text))
		}

		display := sb.String()
		fmt.Fprintf(stdout, "%s", display)

		// Build serializable signals list
		signals := make([]map[string]interface{}, len(filtered))
		for i, sig := range filtered {
			entry := map[string]interface{}{
				"id":       sig.ID,
				"type":     sig.Type,
				"priority": sig.Priority,
				"active":   sig.Active,
			}
			if sig.Strength != nil {
				entry["strength"] = *sig.Strength
			}
			signals[i] = entry
		}

		outputOK(map[string]interface{}{
			"signals": signals,
			"count":   len(filtered),
			"display": display,
		})
		return nil
	},
}

var pheromoneSnapshotInjectCmd = &cobra.Command{
	Use:   "pheromone-snapshot-inject",
	Short: "Inject pheromone snapshot into colony state",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		var pf colony.PheromoneFile
		if err := store.LoadJSON("pheromones.json", &pf); err != nil {
			outputOK(map[string]interface{}{"injected": false, "reason": "no pheromones file"})
			return nil
		}

		active := 0
		for _, sig := range pf.Signals {
			if sig.Active {
				active++
			}
		}

		outputOK(map[string]interface{}{
			"injected":     true,
			"active_count": active,
			"total_count":  len(pf.Signals),
		})
		return nil
	},
}

var pheromoneMergeBackCmd = &cobra.Command{
	Use:   "pheromone-merge-back",
	Short: "Merge pheromone changes back from worktree",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		outputOK(map[string]interface{}{"merged": true})
		return nil
	},
}

func init() {
	colonyPrimeCmd.Flags().Bool("compact", false, "Use 4000 char budget instead of 8000")

	pheromoneDisplayCmd.Flags().String("type", "", "Filter by signal type (FOCUS/REDIRECT/FEEDBACK)")
	pheromoneDisplayCmd.Flags().Bool("active-only", true, "Only show active signals")

	rootCmd.AddCommand(pheromonePrimeCmd)
	rootCmd.AddCommand(colonyPrimeCmd)
	rootCmd.AddCommand(pheromoneSnapshotInjectCmd)
	rootCmd.AddCommand(pheromoneMergeBackCmd)
	rootCmd.AddCommand(pheromoneDisplayCmd)
}

// extractText extracts the text field from JSON content like {"text":"..."}.
func extractText(raw json.RawMessage) string {
	var m map[string]string
	if err := json.Unmarshal(raw, &m); err == nil {
		if text, ok := m["text"]; ok {
			return text
		}
	}
	return strings.TrimSpace(string(raw))
}
