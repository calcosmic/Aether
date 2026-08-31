package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
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
	WorkerTimeout         time.Duration
	RunTimeout            time.Duration
	Context               context.Context
}

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Compatibility alias for live worker activity",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return renderedErrorExit(1)
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
	Use:   "oracle [topic|propose|brief|status|stop|recover|promote|selftest]",
	Short: "Run the autonomous Oracle RALF research loop",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		if len(args) > 0 && strings.EqualFold(strings.TrimSpace(args[0]), "promote") {
			minConfidence, _ := cmd.Flags().GetInt("min-confidence")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			promoteRoot := skillWorkspaceRoot()
			promoteState, _ := loadOracleStateFile(oracleWorkspacePaths(promoteRoot).StatePath)
			result, err := runOraclePromote(promoteRoot, minConfidence, dryRun, oracleResearchProvenanceLabel(promoteState))
			if err != nil {
				outputError(1, err.Error(), nil)
				return renderedErrorExit(1)
			}
			outputOK(result)
			return nil
		}

		// The setup ritual: propose suggests how to scope the run and writes
		// nothing; brief records the approved scope that `--from-brief` then
		// requires.
		if len(args) > 0 && strings.EqualFold(strings.TrimSpace(args[0]), "propose") {
			topic, _ := cmd.Flags().GetString("topic")
			if strings.TrimSpace(topic) == "" {
				topic = strings.TrimSpace(strings.Join(args[1:], " "))
			}
			result, err := runOraclePropose(skillWorkspaceRoot(), topic)
			if err != nil {
				outputError(1, err.Error(), nil)
				return renderedErrorExit(1)
			}
			outputWorkflow(result, renderOraclePropose(result))
			return nil
		}

		if len(args) > 0 && strings.EqualFold(strings.TrimSpace(args[0]), "brief") {
			opts := oracleBriefOptions{}
			opts.Topic, _ = cmd.Flags().GetString("topic")
			opts.CoreQuestion, _ = cmd.Flags().GetString("core-question")
			opts.Context, _ = cmd.Flags().GetString("context")
			opts.SuccessCriteria, _ = cmd.Flags().GetStringArray("success-criteria")
			opts.Template, _ = cmd.Flags().GetString("template")
			opts.Depth, _ = cmd.Flags().GetString("depth")
			opts.Scope, _ = cmd.Flags().GetString("scope")
			opts.MaxIterations, _ = cmd.Flags().GetInt("max-iterations")
			if raw, _ := cmd.Flags().GetString("confidence-target"); strings.TrimSpace(raw) != "" {
				parsed, parseErr := strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(raw), "%"))
				if parseErr != nil {
					outputError(1, fmt.Sprintf("--confidence-target must be a number 1-100, got %q", raw), nil)
					return renderedErrorExit(1)
				}
				opts.TargetConfidence = parsed
			}
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			result, err := runOracleBriefApprove(skillWorkspaceRoot(), opts, dryRun)
			if err != nil {
				outputError(1, err.Error(), nil)
				return renderedErrorExit(1)
			}
			panel, _ := result["panel"].(string)
			outputWorkflow(result, panel)
			return nil
		}

		if len(args) > 0 && strings.EqualFold(strings.TrimSpace(args[0]), "save") {
			name, _ := cmd.Flags().GetString("name")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			result, err := runOracleSave(skillWorkspaceRoot(), name, dryRun)
			if err != nil {
				outputError(1, err.Error(), nil)
				return renderedErrorExit(1)
			}
			outputOK(result)
			return nil
		}

		if len(args) > 0 && strings.EqualFold(strings.TrimSpace(args[0]), "selftest") {
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			result, err := runOracleSelftest(skillWorkspaceRoot(), dryRun)
			if result != nil {
				outputWorkflow(result, renderOracleSelftest(result))
			}
			if err != nil {
				return renderedErrorExit(1)
			}
			return nil
		}

		if len(args) > 0 && strings.EqualFold(strings.TrimSpace(args[0]), "recover") {
			result, err := oracleRecoverStaleRun(skillWorkspaceRoot())
			if err != nil {
				outputError(1, err.Error(), nil)
				return renderedErrorExit(1)
			}
			outputOK(result)
			return nil
		}

		// `status --follow` streams the round log and never touches state.
		//
		// The --from-brief exclusion is load-bearing: `oracle --from-brief
		// --background --follow` — the exact command the wrapper instructs —
		// also has zero positional args, and without the exclusion it was
		// swallowed here as a bare status-follow. The research never started;
		// follow replayed the PREVIOUS run's log and exited as if work had
		// happened. Locked by TestOracleFromBriefBackgroundFollowStartsTheRun.
		follow, _ := cmd.Flags().GetBool("follow")
		followInterval, _ := cmd.Flags().GetDuration("follow-interval")
		fromBrief, _ := cmd.Flags().GetBool("from-brief")
		if follow && !fromBrief && (len(args) == 0 || strings.EqualFold(strings.TrimSpace(args[0]), "status")) {
			if err := followOracleProgress(skillWorkspaceRoot(), followInterval); err != nil {
				outputError(1, err.Error(), nil)
				return renderedErrorExit(1)
			}
			return nil
		}

		depth, _ := cmd.Flags().GetString("depth")
		confidenceTarget, _ := cmd.Flags().GetString("confidence-target")
		scope, _ := cmd.Flags().GetString("scope")
		template, _ := cmd.Flags().GetString("template")
		maxIterations, _ := cmd.Flags().GetInt("max-iterations")
		background, _ := cmd.Flags().GetBool("background")
		maxIterationArg := ""
		if maxIterations > 0 {
			maxIterationArg = fmt.Sprintf("%d", maxIterations)
		}

		// --from-brief is the gated path: it refuses to run unless the setup
		// ritual actually produced an approved brief. Without this the ritual
		// is only prose in a wrapper, which nothing can enforce.
		if fromBrief {
			brief, briefErr := resolveOracleBriefRun(skillWorkspaceRoot())
			if briefErr != nil {
				outputError(1, briefErr.Error(), nil)
				return renderedErrorExit(1)
			}
			args = []string{brief.Topic}
			depth = brief.Depth
			scope = brief.Scope
			template = brief.Template
			confidenceTarget = fmt.Sprintf("%d", brief.TargetConfidence)
			maxIterationArg = ""
			if brief.MaxIterations > 0 {
				maxIterationArg = fmt.Sprintf("%d", brief.MaxIterations)
			}
		}

		result, err := runOracleCompatibility(skillWorkspaceRoot(), args, depth, confidenceTarget, scope, template, maxIterationArg, fmt.Sprintf("%t", background))
		if err != nil {
			outputError(1, err.Error(), nil)
			return renderedErrorExit(1)
		}
		outputWorkflow(result, renderOracleCompatibilityVisual(result))

		// `--background --follow` is the wrapper's normal path: detach the
		// controller so a long run cannot time out the host's tool call, then
		// stream its rounds back so the operator can still watch it work.
		// Only when the run actually detached: a foreground run already
		// printed its rounds live, and replaying the log would print every
		// line twice.
		detached, _ := result["background"].(bool)
		if follow && detached {
			if err := followOracleProgress(skillWorkspaceRoot(), followInterval); err != nil {
				outputError(1, err.Error(), nil)
				return renderedErrorExit(1)
			}
		}
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
		runTimeout, _ := cmd.Flags().GetDuration("timeout")
		workerTimeout, err := resolveWorkerTimeoutFlag(cmd)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		parentCtx := cmd.Context()
		if parentCtx == nil {
			parentCtx = context.Background()
		}
		runCtx, stopSignals := signal.NotifyContext(parentCtx, os.Interrupt, syscall.SIGTERM)
		defer stopSignals()
		if runTimeout > 0 {
			var cancel context.CancelFunc
			runCtx, cancel = context.WithTimeout(runCtx, runTimeout)
			defer cancel()
		}

		result, err := runCompatibilityAutopilot(skillWorkspaceRoot(), runCompatibilityOptions{
			MaxPhases:             maxPhases,
			ReplanInterval:        replanInterval,
			ContinueWithoutReplan: continueWithoutReplan,
			DryRun:                dryRun,
			Headless:              headless,
			Verbose:               verbose,
			WorkerTimeout:         workerTimeout,
			RunTimeout:            runTimeout,
			Context:               runCtx,
		})
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		outputWorkflow(result, renderRunCompatibilityVisual(result))
		return nil
	},
}

var versionsCmd = &cobra.Command{
	Use:   "versions",
	Short: "Show version information for binary, hub, and repo",
	RunE: func(cmd *cobra.Command, args []string) error {
		outputOK(map[string]interface{}{
			"binary": resolveVersion(),
			"hub":    readInstalledHubVersion(),
		})
		return nil
	},
}

func init() {
	watchCmd.Flags().Bool("once", false, "Render a single watch snapshot even in visual TTY mode")
	watchCmd.Flags().Duration("interval", 2*time.Second, "Refresh interval for live watch output")

	runCompatibilityCmd.Flags().Int("max-phases", 0, "Run at most N phases before pausing")
	runCompatibilityCmd.Flags().Int("replan-interval", 2, "Pause for replanning every N completed phases (0 disables; classic default is 2)")
	runCompatibilityCmd.Flags().Bool("continue", false, "Ignore the next replan pause and keep running")
	runCompatibilityCmd.Flags().Bool("dry-run", false, "Preview the autopilot steps without mutating state")
	runCompatibilityCmd.Flags().Bool("headless", false, "Record headless mode in autopilot state")
	runCompatibilityCmd.Flags().BoolP("verbose", "v", false, "Include extra execution detail in the result")
	runCompatibilityCmd.Flags().Duration("worker-timeout", 0, "Override per-worker timeout for build and continue dispatches (e.g. 15m)")
	runCompatibilityCmd.Flags().Duration("timeout", 0, "Optional overall autopilot deadline; normal runs are bounded by phase and worker timeouts")

	oracleCmd.Flags().Int("min-confidence", 80, "For `oracle promote`: minimum question confidence to promote findings from")
	oracleCmd.Flags().Bool("dry-run", false, "For `oracle promote`: report what would be promoted without writing")
	oracleCmd.Flags().String("depth", "", "Research depth: quick, balanced, deep, exhaustive (default: balanced)")
	oracleCmd.Flags().String("confidence-target", "", "Target confidence percentage 1-100 (default: per depth level). Oracle will not finalize below this target unless a hard blocker is reported or max iterations are reached.")
	oracleCmd.Flags().String("scope", defaultOracleScope, "Research scope: auto, repo, web, or both")
	oracleCmd.Flags().String("template", defaultOracleTemplate, "Output template: auto, prd, tech-eval, architecture-review, bug-investigation, research-brief, or custom")
	oracleCmd.Flags().Int("max-iterations", 0, "Override depth iteration cap, 1-50")
	oracleCmd.Flags().Bool("background", false, "Start the Oracle loop in a detached background controller and return immediately")
	oracleCmd.Flags().String("topic", "", "For `oracle propose` and `oracle brief`: the research topic")
	oracleCmd.Flags().String("core-question", "", "For `oracle brief`: the single question this run must answer")
	oracleCmd.Flags().String("context", "", "For `oracle brief`: why this research is happening and what decision it feeds")
	oracleCmd.Flags().StringArray("success-criteria", nil, "For `oracle brief`: what a finished answer contains (repeatable)")
	oracleCmd.Flags().Bool("from-brief", false, "Start the Oracle loop from the approved research brief; fails when no brief has been approved")
	oracleCmd.Flags().String("name", "", "For `oracle save`: filename slug for the saved research document")
	oracleCmd.Flags().Bool("follow", false, "Stream one line per research round until the run ends")
	oracleCmd.Flags().Duration("follow-interval", defaultOracleFollowInterval, "How often `--follow` checks for new rounds")

	rootCmd.AddCommand(watchCmd)
	rootCmd.AddCommand(oracleCmd)
	rootCmd.AddCommand(researchCmd)
	rootCmd.AddCommand(runCompatibilityCmd)
	rootCmd.AddCommand(versionsCmd)
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
		writeVisualOutput(stdout, frame)

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
	ctx := opts.Context
	if ctx == nil {
		ctx = context.Background()
	}

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

	emitVisualProgress(renderRunEngageLine(state, opts))

	for {
		if err := ctx.Err(); err != nil {
			_ = syncRunAutopilotState(state, opts, "paused", "")
			reason := "cancelled"
			if err == context.DeadlineExceeded {
				reason = "timeout"
			}
			result := buildRunExecutionResult(state, opts, steps, phasesCompleted, reason, nextCommandFromState(state))
			result["error"] = err.Error()
			if opts.RunTimeout > 0 {
				result["run_timeout_sec"] = int(opts.RunTimeout / time.Second)
			}
			return result, nil
		}
		if err := syncRunAutopilotState(state, opts, "running", ""); err != nil {
			return nil, err
		}

		switch state.State {
		case colony.StateCOMPLETED:
			_ = syncRunAutopilotState(state, opts, "completed", "")
			return buildRunExecutionResult(state, opts, steps, phasesCompleted, "completed", "aether seal"), nil

		case colony.StateREADY:
			if opts.MaxPhases > 0 && phasesCompleted >= opts.MaxPhases {
				_ = syncRunAutopilotState(state, opts, "paused", "max_phases_reached")
				emitVisualLine(fmt.Sprintf("--- Autopilot: paused after %d phase(s) — max reached ---", phasesCompleted))
				return buildRunExecutionResult(state, opts, steps, phasesCompleted, "max_phases_reached", nextCommandFromState(state)), nil
			}

			phase := recoveryPhase(&state)
			if phase == nil {
				_ = syncRunAutopilotState(state, opts, "completed", "")
				return buildRunExecutionResult(state, opts, steps, phasesCompleted, "completed", "aether seal"), nil
			}

			emitVisualProgress(renderRunPhaseHeader(phase, len(state.Plan.Phases)))

			buildResult, err := runCodexBuildWithOptions(root, phase.ID, nil, false, codexBuildOptions{
				WorkerTimeout: opts.WorkerTimeout,
				ParentContext: ctx,
			})
			if err != nil {
				firstErr := err
				emitVisualLine(fmt.Sprintf("⚠ Build failed for phase %d, attempting single retry...", phase.ID))
				select {
				case <-ctx.Done():
					_ = syncRunAutopilotState(state, opts, "paused", "")
					result := buildRunExecutionResult(state, opts, steps, phasesCompleted, "cancelled", nextCommandFromState(state))
					result["error"] = ctx.Err().Error()
					return result, nil
				case <-time.After(2 * time.Second):
				}
				buildResult, err = runCodexBuildWithOptions(root, phase.ID, nil, false, codexBuildOptions{
					WorkerTimeout: opts.WorkerTimeout,
					ParentContext: ctx,
				})
				if err != nil {
					// Best-effort record, pause regardless (T-188-15): only
					// the write's own error is discarded here, never the
					// pause below. recordAutopilotRetryExhaustion still
					// returns its error to any caller that wants to check.
					_ = recordAutopilotRetryExhaustion(store, phase.ID, firstErr, err)
					_ = syncRunAutopilotState(state, opts, "paused", "")
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

			// Classic pause check between build and verification: the run
			// stops on purpose, with the reason on screen, not mid-flight.
			if reason := checkAutopilotPauseConditions(); reason != "" {
				return pauseAutopilotRun(state, opts, steps, phasesCompleted, reason), nil
			}

		case colony.StateEXECUTING, colony.StateBUILT:
			continueResult, updatedState, phase, _, _, final, err := runCodexContinue(root, codexContinueOptions{
				WorkerTimeout: opts.WorkerTimeout,
				ParentContext: ctx,
			})
			if err != nil {
				_ = syncRunAutopilotState(state, opts, "paused", "")
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
				_ = syncRunAutopilotState(state, opts, "paused", "blocked")
				next := strings.TrimSpace(stringValue(continueResult["next"]))
				if next == "" {
					next = "aether continue"
				}
				if opts.Headless {
					queueAutopilotPauseDecision("blocked", state.CurrentPhase)
				}
				emitVisualProgress(renderRunPauseBlock("blocked", next))
				return buildRunExecutionResult(state, opts, steps, phasesCompleted, "blocked", next), nil
			}

			phasesCompleted++
			if final {
				_ = syncRunAutopilotState(state, opts, "completed", "")
				emitVisualProgress(renderAutopilotComplete(phasesCompleted))
				emitVisualProgress(renderProjectComplete(state, phasesCompleted))
				return buildRunExecutionResult(state, opts, steps, phasesCompleted, "completed", "aether seal"), nil
			}
			emitVisualProgress(renderRunPhaseAdvancement(phase, continueResult, phasesCompleted, len(state.Plan.Phases)))
			if reason := checkAutopilotPauseConditions(); reason != "" {
				return pauseAutopilotRun(state, opts, steps, phasesCompleted, reason), nil
			}
			lessons, lessonErr := loadConfirmedAutopilotLessonsSincePlan(state.Plan)
			if lessonErr != nil {
				_ = syncRunAutopilotState(state, opts, "paused", string(autopilotTriggerColonyNotRunnable))
				return nil, lessonErr
			}
			evidenceReplanDue := lessonAwareReplanDue(phasesCompleted, opts.ReplanInterval, lessons, opts.ContinueWithoutReplan)
			legacyReplanDue := legacyInteractiveReplanDue(state.Plan, phasesCompleted, opts.ReplanInterval, opts.ContinueWithoutReplan, opts.Headless)
			if evidenceReplanDue || legacyReplanDue {
				if opts.Headless {
					decision, err := upsertAutopilotReplanDecision(state, phase.ID, lessons, time.Now().UTC())
					if err != nil {
						_ = syncRunAutopilotState(state, opts, "paused", string(autopilotTriggerColonyNotRunnable))
						return nil, err
					}
					steps = append(steps, map[string]interface{}{
						"event":            "decision_queued",
						"trigger_code":     autopilotTriggerReplanDue,
						"decision_id":      decision.ID,
						"phase":            phase.ID,
						"lesson_count":     decision.LessonCount,
						"plan_revision_id": decision.PlanRevisionID,
					})
					emitVisualProgress(renderRunReplanQueued(decision))
				} else {
					_ = syncRunAutopilotState(state, opts, "paused", string(autopilotTriggerReplanDue))
					emitVisualProgress(renderRunReplanBanner(phasesCompleted, opts.ReplanInterval, len(lessons)))
					result := buildRunExecutionResult(state, opts, steps, phasesCompleted, string(autopilotTriggerReplanDue), "aether plan")
					result["trigger_code"] = autopilotTriggerReplanDue
					result["confirmed_lessons"] = lessons
					result["lesson_count"] = len(lessons)
					return result, nil
				}
			}
			if opts.MaxPhases > 0 && phasesCompleted >= opts.MaxPhases {
				_ = syncRunAutopilotState(state, opts, "paused", "max_phases_reached")
				emitVisualLine(fmt.Sprintf("--- Autopilot: paused after %d phase(s) — max reached ---", phasesCompleted))
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
	finish := func(reason, next string) map[string]interface{} {
		return map[string]interface{}{
			"mode":              "dry-run",
			"dry_run":           true,
			"headless":          opts.Headless,
			"steps":             steps,
			"phases_planned":    phasesPlanned,
			"stopped_reason":    reason,
			"next":              next,
			"current_state":     working.State,
			"continue_armed":    opts.ContinueWithoutReplan,
			"replan_interval":   opts.ReplanInterval,
			"trigger_catalogue": autopilotTriggerSpecs(),
		}
	}

	for {
		switch working.State {
		case colony.StateCOMPLETED:
			return finish("completed", "aether seal")

		case colony.StateEXECUTING, colony.StateBUILT:
			steps = append(steps, map[string]interface{}{
				"command": "aether continue",
				"phase":   working.CurrentPhase,
				"state":   working.State,
			})
			phasesPlanned++
			return finish("continue_required", "aether continue")

		case colony.StateREADY:
			if opts.MaxPhases > 0 && phasesPlanned >= opts.MaxPhases {
				return finish("max_phases_reached", nextCommandFromState(working))
			}

			phase := recoveryPhase(&working)
			if phase == nil {
				return finish("completed", "aether seal")
			}

			steps = append(steps,
				map[string]interface{}{"command": fmt.Sprintf("aether build %d", phase.ID), "phase": phase.ID, "phase_name": phase.Name},
				map[string]interface{}{"command": "aether continue", "phase": phase.ID, "phase_name": phase.Name},
			)
			phasesPlanned++
			if opts.ReplanInterval > 0 && phasesPlanned > 0 && phasesPlanned%opts.ReplanInterval == 0 && !opts.ContinueWithoutReplan {
				return finish("replan_due", "aether plan")
			}

			if phase.ID >= len(working.Plan.Phases) {
				return finish("completed", "aether seal")
			}

			working.Plan.Phases[phase.ID-1].Status = colony.PhaseCompleted
			working.CurrentPhase = phase.ID + 1
			working.State = colony.StateREADY

		default:
			return finish("not_runnable", nextCommandFromState(working))
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

func syncRunAutopilotState(state colony.ColonyState, opts runCompatibilityOptions, status, reason string) error {
	if store == nil {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	apState := autopilotState{
		InitializedAt:  now,
		TotalPhases:    len(state.Plan.Phases),
		CurrentPhase:   state.CurrentPhase,
		Status:         status,
		Reason:         reason,
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
	b.WriteString(visualDividerStr())

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

	// The result reaches this renderer both in-process (steps is
	// []map[string]interface{}) and after a JSON round trip (steps is
	// []interface{}); handle both or the visual silently drops the section.
	var stepMaps []map[string]interface{}
	switch steps := result["steps"].(type) {
	case []map[string]interface{}:
		stepMaps = steps
	case []interface{}:
		for _, raw := range steps {
			if step, ok := raw.(map[string]interface{}); ok {
				stepMaps = append(stepMaps, step)
			}
		}
	}
	if len(stepMaps) > 0 {
		b.WriteString("\nSteps\n")
		for _, step := range stepMaps {
			if step == nil {
				continue
			}
			b.WriteString("  - ")
			b.WriteString(stringValue(step["command"]))
			if phase := intValue(step["phase"]); phase > 0 {
				if name := strings.TrimSpace(stringValue(step["phase_name"])); name != "" {
					b.WriteString(fmt.Sprintf(" [phase %d: %s]", phase, name))
				} else {
					b.WriteString(fmt.Sprintf(" [phase %d]", phase))
				}
			}
			if state := strings.TrimSpace(stringValue(step["state"])); state != "" {
				b.WriteString(" -> ")
				b.WriteString(state)
			}
			b.WriteString("\n")
		}
	}

	if dryRun, _ := result["dry_run"].(bool); dryRun {
		b.WriteString(renderRunDryRunTriggerCatalogue(result["trigger_catalogue"]))
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
	b.WriteString(visualDividerStr())
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
	if boolValue(result["background"]) {
		b.WriteString("Controller: background")
		if pid := intValue(result["controller_pid"]); pid > 0 {
			b.WriteString(fmt.Sprintf(" (PID %d)", pid))
		}
		b.WriteString("\n")
	}
	if boolValue(result["auto_background"]) {
		b.WriteString("Auto-Background: hosted agent session detected; Oracle detached the controller.\n")
	}
	if phase := strings.TrimSpace(stringValue(result["phase"])); phase != "" {
		b.WriteString("Phase: ")
		b.WriteString(phase)
		b.WriteString("\n")
	}
	if template := strings.TrimSpace(stringValue(result["template"])); template != "" {
		b.WriteString("Template: ")
		b.WriteString(template)
		b.WriteString("\n")
	}
	if iteration, maxIterations := intValue(result["iteration"]), intValue(result["max_iterations"]); iteration > 0 || maxIterations > 0 {
		b.WriteString(fmt.Sprintf("Iteration: %d", iteration))
		if maxIterations > 0 {
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
	if logPath := strings.TrimSpace(stringValue(result["log_path"])); logPath != "" {
		b.WriteString("Log: ")
		b.WriteString(logPath)
		b.WriteString("\n")
	}
	next := strings.TrimSpace(stringValue(result["next"]))
	if next == "" {
		next = "aether oracle status"
	}
	b.WriteString(renderNextUp(fmt.Sprintf("Run `%s` for the next oracle step.", next)))
	return b.String()
}
