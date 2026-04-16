package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

var layEggsCmd = &cobra.Command{
	Use:   "lay-eggs",
	Short: "Set up Aether in the current directory from the hub",
	Args:  cobra.NoArgs,
	RunE:  runSetup,
}

var colonizeCmd = &cobra.Command{
	Use:   "colonize",
	Short: "Survey the repository, write territory reports, and record surveyor dispatches",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		forceResurvey, _ := cmd.Flags().GetBool("force-resurvey")
		forceAlias, _ := cmd.Flags().GetBool("force")
		result, err := runCodexColonize(skillWorkspaceRoot(), forceResurvey || forceAlias)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		outputWorkflow(result, renderColonizeVisual(result))
		return nil
	},
}

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Generate or review a survey-aware phase plan for the current colony goal",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		refresh, _ := cmd.Flags().GetBool("refresh")
		forceAlias, _ := cmd.Flags().GetBool("force")
		result, err := runCodexPlan(skillWorkspaceRoot(), refresh || forceAlias)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		outputWorkflow(result, renderPlanVisual(result))
		return nil
	},
}

var buildCmd = &cobra.Command{
	Use:   "build <phase>",
	Short: "Dispatch a real Codex build packet with worker briefs, claims, and spawn tracking",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		phaseNum, err := strconv.Atoi(args[0])
		if err != nil || phaseNum < 1 {
			outputError(1, fmt.Sprintf("invalid phase %q", args[0]), nil)
			return nil
		}

		result, err := runCodexBuild(skillWorkspaceRoot(), phaseNum)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}

		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputError(2, fmt.Sprintf("failed to reload colony state: %v", err), nil)
			return nil
		}

		dispatches := plannedBuildDispatches(state.Plan.Phases[phaseNum-1], state.ColonyDepth)
		if manifestPath, ok := result["manifest"].(string); ok && strings.TrimSpace(manifestPath) != "" {
			rel := strings.TrimPrefix(manifestPath, ".aether/data/")
			var manifest codexBuildManifest
			if err := store.LoadJSON(rel, &manifest); err == nil && len(manifest.Dispatches) > 0 {
				dispatches = manifest.Dispatches
			}
		}
		outputWorkflow(result, renderBuildVisualWithDispatches(state, state.Plan.Phases[phaseNum-1], dispatches))
		return nil
	},
}

var continueCmd = &cobra.Command{
	Use:   "continue",
	Short: "Verify the active build packet, close dispatched workers, and advance honestly",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		result, state, phase, nextPhase, housekeeping, final, err := runCodexContinue(skillWorkspaceRoot())
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}

		if blocked, _ := result["blocked"].(bool); blocked {
			outputWorkflow(result, renderContinueBlockedVisual(state, phase, result))
			return nil
		}

		outputWorkflow(result, renderContinueVisual(state, phase, housekeeping, final, nextPhase, result))
		return nil
	},
}

var sealCmd = &cobra.Command{
	Use:   "seal",
	Short: "Seal a completed colony and write a summary artifact",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputError(1, "No colony initialized. Run `aether init \"goal\"` first.", nil)
			return nil
		}
		if len(state.Plan.Phases) == 0 {
			outputError(1, "No project plan. Run `aether plan` first.", nil)
			return nil
		}

		for _, phase := range state.Plan.Phases {
			if phase.Status != colony.PhaseCompleted {
				outputError(1, "all phases must be completed before sealing the colony", nil)
				return nil
			}
		}

		now := time.Now().UTC().Format(time.RFC3339)
		state.State = colony.StateCOMPLETED
		state.Milestone = "Crowned Anthill"
		state.MilestoneUpdatedAt = &now
		state.Events = append(trimmedEvents(state.Events), fmt.Sprintf("%s|sealed|seal|Colony sealed at Crowned Anthill", now))

		if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
			outputError(2, fmt.Sprintf("failed to save colony state: %v", err), nil)
			return nil
		}

		summaryPath := filepath.Join(filepath.Dir(store.BasePath()), "CROWNED-ANTHILL.md")
		summary := buildSealSummary(state, now)
		if err := os.WriteFile(summaryPath, []byte(summary), 0644); err != nil {
			outputError(2, fmt.Sprintf("failed to write %s: %v", summaryPath, err), nil)
			return nil
		}
		updateSessionSummary("seal", "aether init \"next goal\"", "Colony sealed")

		result := map[string]interface{}{
			"sealed":    true,
			"milestone": state.Milestone,
			"summary":   summaryPath,
		}
		outputWorkflow(result, renderSealVisual(state, summaryPath))
		return nil
	},
}

var focusCmd = newSignalShortcutCommand("focus", "FOCUS", "Guide colony attention")
var redirectCmd = newSignalShortcutCommand("redirect", "REDIRECT", "Add a hard constraint for the colony")
var feedbackCmd = newSignalShortcutCommand("feedback", "FEEDBACK", "Add gentle corrective feedback")

var preferencesCmd = &cobra.Command{
	Use:   "preferences [text]",
	Short: "Read or write user preferences stored in QUEEN.md",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		listOnly, _ := cmd.Flags().GetBool("list")
		hub := resolveHubPath()
		queenPath := filepath.Join(hub, "QUEEN.md")

		if listOnly {
			outputOK(map[string]interface{}{
				"preferences": readUserPreferences(queenPath),
				"path":        queenPath,
			})
			return nil
		}

		text := strings.TrimSpace(strings.Join(args, " "))
		if text == "" {
			outputError(1, "preference text is required unless --list is used", nil)
			return nil
		}
		if len(text) > 500 {
			outputError(1, "preference text exceeds 500 characters", nil)
			return nil
		}

		if _, err := os.Stat(queenPath); os.IsNotExist(err) {
			if err := os.MkdirAll(hub, 0755); err != nil {
				outputError(2, fmt.Sprintf("failed to create hub directory: %v", err), nil)
				return nil
			}
			if err := os.WriteFile(queenPath, []byte(queenDefaultContent), 0644); err != nil {
				outputError(2, fmt.Sprintf("failed to create QUEEN.md: %v", err), nil)
				return nil
			}
		}

		data, err := os.ReadFile(queenPath)
		if err != nil {
			outputError(2, fmt.Sprintf("failed to read QUEEN.md: %v", err), nil)
			return nil
		}

		entry := fmt.Sprintf("- %s", text)
		sectionHeader := "## User Preferences"
		body := string(data)
		idx := strings.Index(body, sectionHeader)
		if idx == -1 {
			body += "\n## User Preferences\n" + entry + "\n"
		} else {
			insertAt := idx + len(sectionHeader)
			if nlIdx := strings.Index(body[insertAt:], "\n"); nlIdx != -1 {
				insertAt += nlIdx + 1
			}
			body = body[:insertAt] + entry + "\n" + body[insertAt:]
		}

		if err := os.WriteFile(queenPath, []byte(body), 0644); err != nil {
			outputError(2, fmt.Sprintf("failed to write QUEEN.md: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"added":      true,
			"preference": text,
			"path":       queenPath,
		})
		return nil
	},
}

func newSignalShortcutCommand(use, signalType, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use + " <text>",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if store == nil {
				outputErrorMessage("no store initialized")
				return nil
			}
			result, err := createPheromoneSignal(signalType, args[0], "user", "", "", 1.0, "")
			if err != nil {
				outputError(1, err.Error(), nil)
				return nil
			}
			priorityValue := signalPriorityValue(signalType)
			if signal, ok := result["signal"].(map[string]interface{}); ok {
				if persisted, ok := signal["priority"].(string); ok && strings.TrimSpace(persisted) != "" {
					priorityValue = persisted
				}
			}
			replaced, _ := result["replaced"].(bool)
			outputWorkflow(result, renderSignalVisual(signalType, args[0], priorityValue, replaced))
			return nil
		},
	}
}

func createPheromoneSignal(sigType, content, sourceFlag, reasonFlag, ttlFlag string, strength float64, priority string) (map[string]interface{}, error) {
	if sigType == "" || strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("signal type and content are required")
	}

	sigType = strings.ToUpper(sigType)
	switch sigType {
	case "FOCUS", "REDIRECT", "FEEDBACK":
	default:
		return nil, fmt.Errorf("invalid signal type %q", sigType)
	}

	if priority == "" {
		switch sigType {
		case "FOCUS":
			priority = "normal"
		case "REDIRECT":
			priority = "high"
		case "FEEDBACK":
			priority = "low"
		}
	}

	if strength == 0 {
		strength = 1.0
	}

	tmpCmd := &cobra.Command{}
	tmpCmd.Flags().String("type", sigType, "")
	tmpCmd.Flags().String("content", content, "")
	tmpCmd.Flags().String("priority", priority, "")
	tmpCmd.Flags().Float64("strength", strength, "")
	tmpCmd.Flags().String("source", sourceFlag, "")
	tmpCmd.Flags().String("reason", reasonFlag, "")
	tmpCmd.Flags().String("ttl", ttlFlag, "")

	var buf strings.Builder
	oldStdout := stdout
	stdout = &buf
	defer func() { stdout = oldStdout }()

	if err := pheromoneWriteCmd.RunE(tmpCmd, nil); err != nil {
		return nil, err
	}

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(buf.String()), &envelope); err != nil {
		return nil, fmt.Errorf("failed to parse pheromone-write result: %w", err)
	}
	if ok, _ := envelope["ok"].(bool); !ok {
		return nil, fmt.Errorf("failed to create pheromone signal")
	}
	result, _ := envelope["result"].(map[string]interface{})
	return result, nil
}

func synthesizePlan(goal string, granularity colony.PlanGranularity, domains []string) []colony.Phase {
	count := 3
	switch granularity {
	case colony.GranularityMilestone:
		count = 5
	case colony.GranularityQuarter:
		count = 8
	case colony.GranularityMajor:
		count = 12
	}

	goalLower := strings.ToLower(goal)
	templates := []struct {
		name        string
		description string
		tasks       []string
	}{
		{
			name:        "Discovery and constraints",
			description: "Map the problem, existing code, and boundaries before implementation.",
			tasks: []string{
				"Review the existing code paths relevant to the goal",
				"Capture constraints, risks, and success criteria",
			},
		},
		{
			name:        "Implementation",
			description: "Make the main code changes required to achieve the goal.",
			tasks: []string{
				"Implement the primary changes for the goal",
				"Add or update automated coverage for the new behavior",
			},
		},
		{
			name:        "Verification and polish",
			description: "Verify the result, tighten loose ends, and prepare the colony for seal.",
			tasks: []string{
				"Run focused verification and address regressions",
				"Document key follow-ups, decisions, and user-visible changes",
			},
		},
	}

	if strings.Contains(goalLower, "fix") || strings.Contains(goalLower, "bug") || strings.Contains(goalLower, "broken") {
		templates = []struct {
			name        string
			description string
			tasks       []string
		}{
			{
				name:        "Reproduce and isolate",
				description: "Reproduce the failure and isolate the code path that causes it.",
				tasks:       []string{"Capture the failing behavior", "Identify the root cause and affected boundary"},
			},
			{
				name:        "Targeted fix",
				description: "Implement the smallest correct fix with focused coverage.",
				tasks:       []string{"Apply the fix", "Add regression coverage"},
			},
			{
				name:        "Regression verification",
				description: "Verify the fix and make sure adjacent behavior still holds.",
				tasks:       []string{"Run focused verification", "Document residual risk or follow-ups"},
			},
		}
	}

	if len(domains) > 0 {
		templates[0].tasks = append(templates[0].tasks, fmt.Sprintf("Validate domain-specific expectations for %s", strings.Join(domains, ", ")))
	}

	phases := make([]colony.Phase, 0, count)
	for i := 0; i < count; i++ {
		template := templates[i%len(templates)]
		if i >= len(templates) {
			template.name = fmt.Sprintf("Execution slice %d", i+1)
			template.description = fmt.Sprintf("Continue delivering the goal in bounded slice %d.", i+1)
		}
		phase := colony.Phase{
			ID:              i + 1,
			Name:            template.name,
			Description:     template.description,
			Status:          colony.PhasePending,
			Tasks:           []colony.Task{},
			SuccessCriteria: []string{"The phase outcome is testable", "The phase advances the colony goal without regressions"},
		}
		if i == 0 {
			phase.Status = colony.PhaseReady
		}
		for j, taskGoal := range template.tasks {
			taskID := fmt.Sprintf("%d.%d", i+1, j+1)
			phase.Tasks = append(phase.Tasks, colony.Task{
				ID:     &taskID,
				Goal:   taskGoal,
				Status: colony.TaskPending,
			})
		}
		phases = append(phases, phase)
	}
	return phases
}

func trimmedEvents(events []string) []string {
	if len(events) < 100 {
		return events
	}
	return append([]string{}, events[len(events)-99:]...)
}

func detectDomainsFromRoot(root string) []string {
	domains := []string{}
	checks := map[string][]string{
		"go":     {"go.mod", "go.sum"},
		"web":    {"package.json", "next.config.js", "vite.config.ts"},
		"ruby":   {"Gemfile", "Rakefile"},
		"python": {"requirements.txt", "setup.py", "pyproject.toml"},
		"rust":   {"Cargo.toml"},
	}

	for domain, files := range checks {
		for _, f := range files {
			if _, err := os.Stat(filepath.Join(root, f)); err == nil {
				domains = append(domains, domain)
				break
			}
		}
	}
	sort.Strings(domains)
	return domains
}

func buildSealSummary(state colony.ColonyState, sealedAt string) string {
	goal := ""
	if state.Goal != nil {
		goal = *state.Goal
	}
	var b strings.Builder
	b.WriteString("# CROWNED-ANTHILL\n\n")
	b.WriteString(fmt.Sprintf("- Goal: %s\n", goal))
	b.WriteString(fmt.Sprintf("- Sealed at: %s\n", sealedAt))
	b.WriteString(fmt.Sprintf("- Completed phases: %d\n", len(state.Plan.Phases)))
	if state.CurrentPhase > 0 {
		b.WriteString(fmt.Sprintf("- Final phase: %d\n", state.CurrentPhase))
	}
	b.WriteString("\n## Phase Summary\n")
	for _, phase := range state.Plan.Phases {
		b.WriteString(fmt.Sprintf("- Phase %d: %s [%s]\n", phase.ID, phase.Name, phase.Status))
	}
	return b.String()
}

func updateSessionSummary(commandName, suggestedNext, summary string) {
	if store == nil {
		return
	}
	var session colony.SessionFile
	if err := store.LoadJSON("session.json", &session); err != nil {
		return
	}
	session.LastCommand = commandName
	session.LastCommandAt = time.Now().UTC().Format(time.RFC3339)
	if suggestedNext != "" {
		session.SuggestedNext = suggestedNext
	}
	if summary != "" {
		session.Summary = summary
	}
	_ = store.SaveJSON("session.json", session)
}

func init() {
	layEggsCmd.Flags().String("repo-dir", "", "Path to the repository (default: $CWD)")
	layEggsCmd.Flags().String("home-dir", "", "User home directory (default: $HOME)")
	colonizeCmd.Flags().Bool("force-resurvey", false, "Refresh survey artifacts even when an existing survey is present")
	colonizeCmd.Flags().Bool("force", false, "Alias for --force-resurvey")
	planCmd.Flags().Bool("refresh", false, "Regenerate the plan even when an existing plan is already present")
	planCmd.Flags().Bool("force", false, "Alias for --refresh")
	preferencesCmd.Flags().Bool("list", false, "List stored preferences")

	rootCmd.AddCommand(layEggsCmd)
	rootCmd.AddCommand(colonizeCmd)
	rootCmd.AddCommand(planCmd)
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(continueCmd)
	rootCmd.AddCommand(sealCmd)
	rootCmd.AddCommand(focusCmd)
	rootCmd.AddCommand(redirectCmd)
	rootCmd.AddCommand(feedbackCmd)
	rootCmd.AddCommand(preferencesCmd)
}
