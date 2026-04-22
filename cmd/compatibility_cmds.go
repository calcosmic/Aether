package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
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

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Compatibility alias for live worker activity",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		once, _ := cmd.Flags().GetBool("once")
		interval, _ := cmd.Flags().GetDuration("interval")
		if shouldUseLiveWatchRefresh(stdout, once) {
			return runLiveWatch(interval)
		}

		result := buildSwarmWatchResult("", true, false)
		visual := renderSwarmCompatibilityVisual(result)
		_ = writeWatchArtifacts(result, visual)
		outputWorkflow(result, visual)
		return nil
	},
}

var oracleCmd = &cobra.Command{
	Use:   "oracle [topic|status|stop]",
	Short: "Run the autonomous Oracle RALF research loop",
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
	watchCmd.Flags().Bool("once", false, "Render a single watch snapshot even in visual TTY mode")
	watchCmd.Flags().Duration("interval", 2*time.Second, "Refresh interval for live watch output")

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
	statusText := fmt.Sprintf("state=%s scope=%s active_workers=%d completed_workers=%d blocked_workers=%d failed_workers=%d live_refresh=%t next=%s\n",
		stringValue(result["state"]),
		stringValue(result["scope"]),
		intValue(result["active_count"]),
		intValue(result["completed_count"]),
		intValue(result["blocked_count"]),
		intValue(result["failed_count"]),
		boolValue(result["live_refresh"]),
		stringValue(result["next"]),
	)
	if err := store.AtomicWrite("watch-status.txt", []byte(statusText)); err != nil {
		return err
	}
	return store.AtomicWrite("watch-progress.txt", []byte(visual))
}

func runLiveWatch(interval time.Duration) error {
	if interval <= 0 {
		interval = 2 * time.Second
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		result := buildSwarmWatchResult("", true, true)
		visual := renderSwarmCompatibilityVisual(result)
		_ = writeWatchArtifacts(result, visual)

		frame := "\033[H\033[2J" + strings.TrimRight(visual, "\n") + "\n"
		fmt.Fprint(stdout, frame)

		stateName := strings.TrimSpace(stringValue(result["state"]))
		if intValue(result["active_count"]) == 0 && stateName != string(colony.StateEXECUTING) && stateName != string(colony.StateBUILT) {
			return nil
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
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

			buildResult, err := runCodexBuild(root, phase.ID, nil, false)
			if err != nil {
				fmt.Fprintf(os.Stderr, "⚠ Build failed for phase %d, attempting single retry...\n", phase.ID)
				time.Sleep(2 * time.Second)
				buildResult, err = runCodexBuild(root, phase.ID, nil, false)
				if err != nil {
					_ = syncRunAutopilotState(state, opts, "paused")
					return nil, err
				}
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
			continueResult, updatedState, phase, _, _, final, err := runCodexContinue(root, codexContinueOptions{})
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
				next := strings.TrimSpace(stringValue(continueResult["next"]))
				if next == "" {
					next = "aether continue"
				}
				return buildRunExecutionResult(state, opts, steps, phasesCompleted, "blocked", next), nil
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
	b.WriteString(renderBanner(commandEmoji("run"), "Run"))
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
	b.WriteString(renderBanner(commandEmoji("oracle"), "Oracle"))
	b.WriteString(visualDivider)
	b.WriteString("Mode: ")
	b.WriteString(emptyFallback(stringValue(result["mode"]), "status"))
	b.WriteString("\n")
	if topic := strings.TrimSpace(stringValue(result["topic"])); topic != "" {
		b.WriteString("Topic: ")
		b.WriteString(oracleTopicHeadline(topic))
		b.WriteString("\n")
	}
	b.WriteString("Status: ")
	b.WriteString(emptyFallback(stringValue(result["status"]), "idle"))
	b.WriteString("\n")
	if phase := strings.TrimSpace(stringValue(result["phase"])); phase != "" {
		b.WriteString("Phase: ")
		b.WriteString(phase)
		b.WriteString("\n")
	}
	if iteration := intValue(result["iteration"]); iteration > 0 {
		b.WriteString(fmt.Sprintf("Iteration: %d", iteration))
		if maxIterations := intValue(result["max_iterations"]); maxIterations > 0 {
			b.WriteString(fmt.Sprintf(" of %d", maxIterations))
		}
		b.WriteString("\n")
	}
	if attempt := intValue(result["active_attempt"]); attempt > 0 {
		b.WriteString(fmt.Sprintf("Attempt: %d\n", attempt))
	}
	if reasoning := strings.TrimSpace(stringValue(result["active_reasoning"])); reasoning != "" {
		b.WriteString("Reasoning: ")
		b.WriteString(reasoning)
		b.WriteString("\n")
	}
	if timeoutSec := intValue(result["active_timeout_sec"]); timeoutSec > 0 {
		b.WriteString(fmt.Sprintf("Watchdog: %ds\n", timeoutSec))
		if elapsedSec := intValue(result["active_elapsed_sec"]); elapsedSec > 0 {
			b.WriteString(fmt.Sprintf("Elapsed: %ds\n", elapsedSec))
		}
	}
	if confidence := intValue(result["overall_confidence"]); confidence > 0 {
		b.WriteString(fmt.Sprintf("Confidence: %d%%", confidence))
		if target := intValue(result["target_confidence"]); target > 0 {
			b.WriteString(fmt.Sprintf(" / %d%%", target))
		}
		b.WriteString("\n")
	}
	if questionCount := intValue(result["question_count"]); questionCount > 0 {
		b.WriteString(fmt.Sprintf("Questions: %d", questionCount))
		if answered := intValue(result["answered_count"]); answered >= 0 {
			b.WriteString(fmt.Sprintf(" (%d answered)", answered))
		}
		b.WriteString("\n")
	}
	if activeQuestion := strings.TrimSpace(stringValue(result["active_question"])); activeQuestion != "" {
		b.WriteString("Target: ")
		b.WriteString(activeQuestion)
		b.WriteString("\n")
	}
	if stopReason := strings.TrimSpace(stringValue(result["stop_reason"])); stopReason != "" {
		b.WriteString("Stop Reason: ")
		b.WriteString(stopReason)
		b.WriteString("\n")
	}
	if summary := strings.TrimSpace(stringValue(result["summary"])); summary != "" {
		b.WriteString("Summary: ")
		b.WriteString(summary)
		b.WriteString("\n")
	}
	if artifact := strings.TrimSpace(stringValue(result["last_artifact_path"])); artifact != "" {
		b.WriteString("Last Artifact: ")
		b.WriteString(artifact)
		b.WriteString("\n")
	}
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
