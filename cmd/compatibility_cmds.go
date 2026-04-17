package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

type runCompatibilityOptions struct {
	MaxPhases             int
	ReplanInterval        int
	ContinueWithoutReplan bool
	DryRun                bool
	Headless              bool
	Verbose               bool
}

type oracleStateFile struct {
	Status    string `json:"status"`
	Topic     string `json:"topic,omitempty"`
	Platform  string `json:"platform,omitempty"`
	StartedAt string `json:"started_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Compatibility alias for live worker activity",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		result := buildSwarmCompatibilityResult("", true)
		visual := renderSwarmCompatibilityVisual(result)
		_ = writeWatchArtifacts(result, visual)
		outputWorkflow(result, visual)
		return nil
	},
}

var oracleCmd = &cobra.Command{
	Use:   "oracle [topic|status|stop]",
	Short: "Compatibility entrypoint for oracle research workspace management",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		result, err := runOracleCompatibility(skillWorkspaceRoot(), args)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		outputWorkflow(result, renderOracleCompatibilityVisual(result))
		return nil
	},
}

var runCompatibilityCmd = &cobra.Command{
	Use:   "run",
	Short: "Run remaining phases through the Codex build and continue loop",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		maxPhases, _ := cmd.Flags().GetInt("max-phases")
		replanInterval, _ := cmd.Flags().GetInt("replan-interval")
		continueWithoutReplan, _ := cmd.Flags().GetBool("continue")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		headless, _ := cmd.Flags().GetBool("headless")
		verbose, _ := cmd.Flags().GetBool("verbose")

		result, err := runCompatibilityAutopilot(skillWorkspaceRoot(), runCompatibilityOptions{
			MaxPhases:             maxPhases,
			ReplanInterval:        replanInterval,
			ContinueWithoutReplan: continueWithoutReplan,
			DryRun:                dryRun,
			Headless:              headless,
			Verbose:               verbose,
		})
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		outputWorkflow(result, renderRunCompatibilityVisual(result))
		return nil
	},
}

func init() {
	runCompatibilityCmd.Flags().Int("max-phases", 0, "Run at most N phases before pausing")
	runCompatibilityCmd.Flags().Int("replan-interval", 0, "Pause for replanning every N completed phases")
	runCompatibilityCmd.Flags().Bool("continue", false, "Ignore the next replan pause and keep running")
	runCompatibilityCmd.Flags().Bool("dry-run", false, "Preview the autopilot steps without mutating state")
	runCompatibilityCmd.Flags().Bool("headless", false, "Record headless mode in autopilot state")
	runCompatibilityCmd.Flags().BoolP("verbose", "v", false, "Include extra execution detail in the result")

	rootCmd.AddCommand(watchCmd)
	rootCmd.AddCommand(oracleCmd)
	rootCmd.AddCommand(runCompatibilityCmd)
}

func writeWatchArtifacts(result map[string]interface{}, visual string) error {
	if store == nil {
		return nil
	}
	statusText := fmt.Sprintf("state=%s active_workers=%d next=%s\n",
		stringValue(result["state"]),
		intValue(result["active_count"]),
		stringValue(result["next"]),
	)
	if err := store.AtomicWrite("watch-status.txt", []byte(statusText)); err != nil {
		return err
	}
	return store.AtomicWrite("watch-progress.txt", []byte(visual))
}

func runOracleCompatibility(root string, args []string) (map[string]interface{}, error) {
	mode := "status"
	if len(args) > 0 {
		mode = strings.ToLower(strings.TrimSpace(args[0]))
	}

	switch mode {
	case "", "status":
		return oracleStatusResult(root)
	case "stop":
		return stopOracleCompatibility(root)
	default:
		return startOracleCompatibility(root, strings.TrimSpace(strings.Join(args, " ")))
	}
}

func oracleStatusResult(root string) (map[string]interface{}, error) {
	oracleDir := filepath.Join(root, ".aether", "oracle")
	statePath := filepath.Join(oracleDir, "state.json")
	planPath := filepath.Join(oracleDir, "plan.json")
	synthesisPath := filepath.Join(oracleDir, "synthesis.md")
	researchPlanPath := filepath.Join(oracleDir, "research-plan.md")

	state := oracleStateFile{}
	if data, err := os.ReadFile(statePath); err == nil {
		_ = json.Unmarshal(data, &state)
	}

	active := strings.EqualFold(state.Status, "active") || strings.EqualFold(state.Status, "planned")
	next := "aether oracle \"research topic\""
	if active {
		next = "aether oracle stop"
	} else if fileExists(researchPlanPath) {
		next = "aether oracle status"
	}

	return map[string]interface{}{
		"mode":              "status",
		"active":            active,
		"status":            emptyFallback(strings.TrimSpace(state.Status), "idle"),
		"topic":             strings.TrimSpace(state.Topic),
		"platform":          emptyFallback(strings.TrimSpace(state.Platform), "codex"),
		"state_path":        statePath,
		"plan_path":         planPath,
		"synthesis_path":    synthesisPath,
		"research_plan":     researchPlanPath,
		"has_state":         fileExists(statePath),
		"has_plan":          fileExists(planPath),
		"has_synthesis":     fileExists(synthesisPath),
		"has_research_plan": fileExists(researchPlanPath),
		"next":              next,
	}, nil
}

func stopOracleCompatibility(root string) (map[string]interface{}, error) {
	oracleDir := filepath.Join(root, ".aether", "oracle")
	if err := os.MkdirAll(oracleDir, 0755); err != nil {
		return nil, fmt.Errorf("create oracle dir: %w", err)
	}
	stopPath := filepath.Join(oracleDir, ".stop")
	if err := os.WriteFile(stopPath, []byte(time.Now().UTC().Format(time.RFC3339)+"\n"), 0644); err != nil {
		return nil, fmt.Errorf("write stop marker: %w", err)
	}
	_ = os.Remove(filepath.Join(oracleDir, ".loop-active"))

	statePath := filepath.Join(oracleDir, "state.json")
	state := oracleStateFile{
		Status:    "stopped",
		Platform:  "codex",
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if data, err := os.ReadFile(statePath); err == nil {
		_ = json.Unmarshal(data, &state)
		state.Status = "stopped"
		state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if encoded, err := json.MarshalIndent(state, "", "  "); err == nil {
		_ = os.WriteFile(statePath, append(encoded, '\n'), 0644)
	}

	return map[string]interface{}{
		"mode":       "stop",
		"stopped":    true,
		"status":     "stopped",
		"state_path": statePath,
		"stop_path":  stopPath,
		"next":       "aether oracle status",
	}, nil
}

func startOracleCompatibility(root, topic string) (map[string]interface{}, error) {
	if topic == "" {
		return oracleStatusResult(root)
	}

	oracleDir := filepath.Join(root, ".aether", "oracle")
	if err := os.MkdirAll(filepath.Join(oracleDir, "archive"), 0755); err != nil {
		return nil, fmt.Errorf("create oracle archive dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(oracleDir, "discoveries"), 0755); err != nil {
		return nil, fmt.Errorf("create oracle discoveries dir: %w", err)
	}

	detectedType, languages, frameworks := detectOracleProjectProfile(root)
	now := time.Now().UTC().Format(time.RFC3339)
	state := oracleStateFile{
		Status:    "planned",
		Topic:     topic,
		Platform:  "codex",
		StartedAt: now,
		UpdatedAt: now,
	}
	statePath := filepath.Join(oracleDir, "state.json")
	if encoded, err := json.MarshalIndent(state, "", "  "); err != nil {
		return nil, fmt.Errorf("marshal oracle state: %w", err)
	} else if err := os.WriteFile(statePath, append(encoded, '\n'), 0644); err != nil {
		return nil, fmt.Errorf("write oracle state: %w", err)
	}

	plan := map[string]interface{}{
		"topic":      topic,
		"detected":   detectedType,
		"languages":  languages,
		"frameworks": frameworks,
		"created_at": now,
		"questions": []string{
			"What is the actual failure mode or risk behind this topic?",
			"Which files, systems, or commands matter most for this investigation?",
			"What evidence would be strong enough to turn findings into a release decision?",
		},
	}
	planPath := filepath.Join(oracleDir, "plan.json")
	if encoded, err := json.MarshalIndent(plan, "", "  "); err != nil {
		return nil, fmt.Errorf("marshal oracle plan: %w", err)
	} else if err := os.WriteFile(planPath, append(encoded, '\n'), 0644); err != nil {
		return nil, fmt.Errorf("write oracle plan: %w", err)
	}

	researchPlanPath := filepath.Join(oracleDir, "research-plan.md")
	researchPlan := strings.TrimSpace(fmt.Sprintf(`# Oracle Research Plan

Topic: %s

Detected Type: %s
Languages: %s
Frameworks: %s

Next moves:
- Gather primary evidence from the affected files and commands.
- Record gaps in gaps.md as unknowns are discovered.
- Write findings and release recommendations in synthesis.md.
`, topic, emptyFallback(detectedType, "unknown"), renderCSV(languages, "unknown"), renderCSV(frameworks, "none detected"))) + "\n"
	if err := os.WriteFile(researchPlanPath, []byte(researchPlan), 0644); err != nil {
		return nil, fmt.Errorf("write research plan: %w", err)
	}

	gapsPath := filepath.Join(oracleDir, "gaps.md")
	gaps := strings.TrimSpace(fmt.Sprintf(`# Oracle Gaps

- Unknown root cause for: %s
- Unknown strongest evidence path
- Unknown release impact boundary
`, topic)) + "\n"
	if err := os.WriteFile(gapsPath, []byte(gaps), 0644); err != nil {
		return nil, fmt.Errorf("write oracle gaps: %w", err)
	}

	synthesisPath := filepath.Join(oracleDir, "synthesis.md")
	synthesis := strings.TrimSpace(fmt.Sprintf(`# Oracle Synthesis

Status: planned
Topic: %s

Findings will accumulate here as the investigation progresses.
`, topic)) + "\n"
	if err := os.WriteFile(synthesisPath, []byte(synthesis), 0644); err != nil {
		return nil, fmt.Errorf("write oracle synthesis: %w", err)
	}

	return map[string]interface{}{
		"mode":           "start",
		"started":        true,
		"status":         "planned",
		"topic":          topic,
		"detected_type":  detectedType,
		"languages":      languages,
		"frameworks":     frameworks,
		"state_path":     statePath,
		"plan_path":      planPath,
		"gaps_path":      gapsPath,
		"synthesis_path": synthesisPath,
		"research_plan":  researchPlanPath,
		"next":           "aether oracle status",
	}, nil
}

func detectOracleProjectProfile(root string) (string, []string, []string) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return "unknown", nil, nil
	}

	entryNames := make(map[string]bool, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		entryNames[entry.Name()] = true
	}

	detected := "unknown"
	seenLanguages := map[string]bool{}
	seenFrameworks := map[string]bool{}
	languages := []string{}
	frameworks := []string{}

	for _, det := range projectDetectors {
		if !entryNames[det.file] {
			continue
		}
		if detected == "unknown" {
			detected = det.typ
		}
		if !seenLanguages[det.typ] {
			seenLanguages[det.typ] = true
			languages = append(languages, det.typ)
		}
		for _, framework := range det.frameworks {
			if seenFrameworks[framework] {
				continue
			}
			seenFrameworks[framework] = true
			frameworks = append(frameworks, framework)
		}
	}

	return detected, languages, frameworks
}

func runCompatibilityAutopilot(root string, opts runCompatibilityOptions) (map[string]interface{}, error) {
	state, err := loadCompatibilityColonyState()
	if err != nil {
		return nil, err
	}
	if len(state.Plan.Phases) == 0 {
		return nil, fmt.Errorf("No project plan. Run `aether plan` first.")
	}

	if opts.DryRun {
		return buildRunDryRunResult(state, opts), nil
	}

	steps := make([]map[string]interface{}, 0, len(state.Plan.Phases)*2)
	phasesCompleted := 0

	for {
		if err := syncRunAutopilotState(state, opts, "running"); err != nil {
			return nil, err
		}

		switch state.State {
		case colony.StateCOMPLETED:
			_ = syncRunAutopilotState(state, opts, "completed")
			return buildRunExecutionResult(state, opts, steps, phasesCompleted, "completed", "aether seal"), nil

		case colony.StateREADY:
			if opts.MaxPhases > 0 && phasesCompleted >= opts.MaxPhases {
				_ = syncRunAutopilotState(state, opts, "paused")
				return buildRunExecutionResult(state, opts, steps, phasesCompleted, "max_phases_reached", nextCommandFromState(state)), nil
			}

			phase := recoveryPhase(&state)
			if phase == nil {
				_ = syncRunAutopilotState(state, opts, "completed")
				return buildRunExecutionResult(state, opts, steps, phasesCompleted, "completed", "aether seal"), nil
			}

			buildResult, err := runCodexBuild(root, phase.ID)
			if err != nil {
				_ = syncRunAutopilotState(state, opts, "paused")
				return nil, err
			}
			steps = append(steps, map[string]interface{}{
				"command":       fmt.Sprintf("aether build %d", phase.ID),
				"phase":         phase.ID,
				"phase_name":    phase.Name,
				"dispatch_mode": buildResult["dispatch_mode"],
				"dispatches":    buildResult["dispatch_count"],
				"state":         buildResult["state"],
				"next":          buildResult["next"],
			})

			state, err = loadCompatibilityColonyState()
			if err != nil {
				return nil, err
			}

		case colony.StateEXECUTING, colony.StateBUILT:
			continueResult, updatedState, phase, _, _, final, err := runCodexContinue(root)
			if err != nil {
				_ = syncRunAutopilotState(state, opts, "paused")
				return nil, err
			}

			steps = append(steps, map[string]interface{}{
				"command":    "aether continue",
				"phase":      phase.ID,
				"phase_name": phase.Name,
				"advanced":   continueResult["advanced"],
				"blocked":    continueResult["blocked"],
				"state":      continueResult["state"],
				"next":       continueResult["next"],
			})
			state = updatedState

			if blocked, _ := continueResult["blocked"].(bool); blocked {
				_ = syncRunAutopilotState(state, opts, "paused")
				return buildRunExecutionResult(state, opts, steps, phasesCompleted, "blocked", "aether continue"), nil
			}

			phasesCompleted++
			if final {
				_ = syncRunAutopilotState(state, opts, "completed")
				return buildRunExecutionResult(state, opts, steps, phasesCompleted, "completed", "aether seal"), nil
			}
			if opts.ReplanInterval > 0 && phasesCompleted > 0 && phasesCompleted%opts.ReplanInterval == 0 && !opts.ContinueWithoutReplan {
				_ = syncRunAutopilotState(state, opts, "paused")
				return buildRunExecutionResult(state, opts, steps, phasesCompleted, "replan_due", "aether plan"), nil
			}
			if opts.MaxPhases > 0 && phasesCompleted >= opts.MaxPhases {
				_ = syncRunAutopilotState(state, opts, "paused")
				return buildRunExecutionResult(state, opts, steps, phasesCompleted, "max_phases_reached", nextCommandFromState(state)), nil
			}

		default:
			return nil, fmt.Errorf("Colony state %q is not runnable. Run `%s` first.", state.State, nextCommandFromState(state))
		}
	}
}

func loadCompatibilityColonyState() (colony.ColonyState, error) {
	state, err := loadActiveColonyState()
	if err != nil {
		return colony.ColonyState{}, fmt.Errorf("%s", colonyStateLoadMessage(err))
	}
	return state, nil
}

func buildRunDryRunResult(state colony.ColonyState, opts runCompatibilityOptions) map[string]interface{} {
	steps := []map[string]interface{}{}
	phasesPlanned := 0
	working := state

	for {
		switch working.State {
		case colony.StateCOMPLETED:
			return map[string]interface{}{
				"mode":            "dry-run",
				"dry_run":         true,
				"headless":        opts.Headless,
				"steps":           steps,
				"phases_planned":  phasesPlanned,
				"stopped_reason":  "completed",
				"next":            "aether seal",
				"current_state":   working.State,
				"continue_armed":  opts.ContinueWithoutReplan,
				"replan_interval": opts.ReplanInterval,
			}

		case colony.StateEXECUTING, colony.StateBUILT:
			steps = append(steps, map[string]interface{}{
				"command": "aether continue",
				"phase":   working.CurrentPhase,
				"state":   working.State,
			})
			phasesPlanned++
			return map[string]interface{}{
				"mode":            "dry-run",
				"dry_run":         true,
				"headless":        opts.Headless,
				"steps":           steps,
				"phases_planned":  phasesPlanned,
				"stopped_reason":  "continue_required",
				"next":            "aether continue",
				"current_state":   working.State,
				"continue_armed":  opts.ContinueWithoutReplan,
				"replan_interval": opts.ReplanInterval,
			}

		case colony.StateREADY:
			if opts.MaxPhases > 0 && phasesPlanned >= opts.MaxPhases {
				return map[string]interface{}{
					"mode":            "dry-run",
					"dry_run":         true,
					"headless":        opts.Headless,
					"steps":           steps,
					"phases_planned":  phasesPlanned,
					"stopped_reason":  "max_phases_reached",
					"next":            nextCommandFromState(working),
					"current_state":   working.State,
					"continue_armed":  opts.ContinueWithoutReplan,
					"replan_interval": opts.ReplanInterval,
				}
			}

			phase := recoveryPhase(&working)
			if phase == nil {
				return map[string]interface{}{
					"mode":            "dry-run",
					"dry_run":         true,
					"headless":        opts.Headless,
					"steps":           steps,
					"phases_planned":  phasesPlanned,
					"stopped_reason":  "completed",
					"next":            "aether seal",
					"current_state":   working.State,
					"continue_armed":  opts.ContinueWithoutReplan,
					"replan_interval": opts.ReplanInterval,
				}
			}

			steps = append(steps,
				map[string]interface{}{"command": fmt.Sprintf("aether build %d", phase.ID), "phase": phase.ID, "phase_name": phase.Name},
				map[string]interface{}{"command": "aether continue", "phase": phase.ID, "phase_name": phase.Name},
			)
			phasesPlanned++
			if opts.ReplanInterval > 0 && phasesPlanned > 0 && phasesPlanned%opts.ReplanInterval == 0 && !opts.ContinueWithoutReplan {
				return map[string]interface{}{
					"mode":            "dry-run",
					"dry_run":         true,
					"headless":        opts.Headless,
					"steps":           steps,
					"phases_planned":  phasesPlanned,
					"stopped_reason":  "replan_due",
					"next":            "aether plan",
					"current_state":   working.State,
					"continue_armed":  opts.ContinueWithoutReplan,
					"replan_interval": opts.ReplanInterval,
				}
			}

			if phase.ID >= len(working.Plan.Phases) {
				return map[string]interface{}{
					"mode":            "dry-run",
					"dry_run":         true,
					"headless":        opts.Headless,
					"steps":           steps,
					"phases_planned":  phasesPlanned,
					"stopped_reason":  "completed",
					"next":            "aether seal",
					"current_state":   working.State,
					"continue_armed":  opts.ContinueWithoutReplan,
					"replan_interval": opts.ReplanInterval,
				}
			}

			working.Plan.Phases[phase.ID-1].Status = colony.PhaseCompleted
			working.CurrentPhase = phase.ID + 1
			working.State = colony.StateREADY

		default:
			return map[string]interface{}{
				"mode":            "dry-run",
				"dry_run":         true,
				"headless":        opts.Headless,
				"steps":           steps,
				"phases_planned":  phasesPlanned,
				"stopped_reason":  "not_runnable",
				"next":            nextCommandFromState(working),
				"current_state":   working.State,
				"continue_armed":  opts.ContinueWithoutReplan,
				"replan_interval": opts.ReplanInterval,
			}
		}
	}
}

func buildRunExecutionResult(state colony.ColonyState, opts runCompatibilityOptions, steps []map[string]interface{}, phasesCompleted int, reason, next string) map[string]interface{} {
	return map[string]interface{}{
		"mode":             "run",
		"dry_run":          false,
		"headless":         opts.Headless,
		"verbose":          opts.Verbose,
		"steps":            steps,
		"phases_completed": phasesCompleted,
		"stopped_reason":   reason,
		"current_state":    state.State,
		"current_phase":    state.CurrentPhase,
		"next":             next,
		"completed":        state.State == colony.StateCOMPLETED,
	}
}

func syncRunAutopilotState(state colony.ColonyState, opts runCompatibilityOptions, status string) error {
	if store == nil {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	apState := autopilotState{
		InitializedAt:  now,
		TotalPhases:    len(state.Plan.Phases),
		CurrentPhase:   state.CurrentPhase,
		Status:         status,
		Headless:       opts.Headless,
		ReplanInterval: opts.ReplanInterval,
		Phases:         make([]autopilotPhaseStatus, 0, len(state.Plan.Phases)),
		LastUpdated:    now,
	}

	for _, phase := range state.Plan.Phases {
		phaseStatus := string(phase.Status)
		if strings.TrimSpace(phaseStatus) == "" {
			phaseStatus = string(colony.PhasePending)
		}
		apState.Phases = append(apState.Phases, autopilotPhaseStatus{
			Phase:  phase.ID,
			Status: phaseStatus,
			At:     now,
		})
	}

	if existingData, err := os.ReadFile(filepath.Join(store.BasePath(), autopilotStatePath)); err == nil {
		var existing autopilotState
		if json.Unmarshal(existingData, &existing) == nil && strings.TrimSpace(existing.InitializedAt) != "" {
			apState.InitializedAt = existing.InitializedAt
		}
	}

	return store.SaveJSON(autopilotStatePath, apState)
}

func renderRunCompatibilityVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner("🏃", "Run"))
	b.WriteString(visualDivider)

	if dryRun, _ := result["dry_run"].(bool); dryRun {
		b.WriteString("Autopilot preview only.\n")
	} else {
		b.WriteString("Autopilot loop executed.\n")
	}

	b.WriteString("State: ")
	b.WriteString(emptyFallback(stringValue(result["current_state"]), "unknown"))
	b.WriteString("\n")
	if phase := intValue(result["current_phase"]); phase > 0 {
		b.WriteString(fmt.Sprintf("Current Phase: %d\n", phase))
	}
	if phasesCompleted := intValue(result["phases_completed"]); phasesCompleted > 0 {
		b.WriteString(fmt.Sprintf("Phases Completed: %d\n", phasesCompleted))
	} else if phasesPlanned := intValue(result["phases_planned"]); phasesPlanned > 0 {
		b.WriteString(fmt.Sprintf("Phases Planned: %d\n", phasesPlanned))
	}

	if steps, ok := result["steps"].([]interface{}); ok && len(steps) > 0 {
		b.WriteString("\nSteps\n")
		for _, raw := range steps {
			step, _ := raw.(map[string]interface{})
			if step == nil {
				continue
			}
			b.WriteString("  - ")
			b.WriteString(stringValue(step["command"]))
			if phase := intValue(step["phase"]); phase > 0 {
				b.WriteString(fmt.Sprintf(" [phase %d]", phase))
			}
			if state := strings.TrimSpace(stringValue(step["state"])); state != "" {
				b.WriteString(" -> ")
				b.WriteString(state)
			}
			b.WriteString("\n")
		}
	}

	next := strings.TrimSpace(stringValue(result["next"]))
	if next == "" {
		next = "aether status"
	}
	b.WriteString(renderNextUp(
		fmt.Sprintf("Run `%s` for the next lifecycle step.", next),
		fmt.Sprintf("Stop reason: %s", emptyFallback(stringValue(result["stopped_reason"]), "none")),
	))
	return b.String()
}

func renderOracleCompatibilityVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner("🔮", "Oracle"))
	b.WriteString(visualDivider)
	b.WriteString("Mode: ")
	b.WriteString(emptyFallback(stringValue(result["mode"]), "status"))
	b.WriteString("\n")
	if topic := strings.TrimSpace(stringValue(result["topic"])); topic != "" {
		b.WriteString("Topic: ")
		b.WriteString(topic)
		b.WriteString("\n")
	}
	b.WriteString("Status: ")
	b.WriteString(emptyFallback(stringValue(result["status"]), "idle"))
	b.WriteString("\n")
	if path := strings.TrimSpace(stringValue(result["research_plan"])); path != "" {
		b.WriteString("Research Plan: ")
		b.WriteString(path)
		b.WriteString("\n")
	}
	next := strings.TrimSpace(stringValue(result["next"]))
	if next == "" {
		next = "aether oracle status"
	}
	b.WriteString(renderNextUp(fmt.Sprintf("Run `%s` for the next oracle step.", next)))
	return b.String()
}
