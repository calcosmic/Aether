package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

type claudeHookInput struct {
	HookEventName        string                 `json:"hook_event_name"`
	ToolName             string                 `json:"tool_name"`
	ToolInput            map[string]interface{} `json:"tool_input"`
	Cwd                  string                 `json:"cwd"`
	Trigger              string                 `json:"trigger"`
	CustomInstructions   string                 `json:"custom_instructions"`
	StopHookActive       bool                   `json:"stop_hook_active"`
	LastAssistantMessage string                 `json:"last_assistant_message"`
}

var hookPreToolUseCmd = &cobra.Command{
	Use:    "hook-pre-tool-use [tool_name] [target]",
	Short:  "Claude hook: validate edits before tool execution",
	Hidden: true,
	Args:   cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		input := readClaudeHookInput()

		toolName := input.ToolName
		if toolName == "" && len(args) > 0 {
			toolName = args[0]
		}
		if toolName == "" {
			return nil
		}

		target := hookToolTargetPath(input.ToolInput)
		if target == "" && len(args) > 1 {
			target = args[1]
		}

		cwd := input.Cwd
		if cwd == "" {
			if wd, err := os.Getwd(); err == nil {
				cwd = wd
			}
		}

		if strings.EqualFold(toolName, "Write") || strings.EqualFold(toolName, "Edit") {
			if reason := protectedHookWriteReason(target, cwd); reason != "" {
				return emitHookBlock(reason)
			}
			if reason := redirectWriteReason(target, cwd); reason != "" {
				return emitHookBlock(reason)
			}
		}

		return nil
	},
}

var hookStopCmd = &cobra.Command{
	Use:    "hook-stop",
	Short:  "Claude hook: prevent accidental stop mid-phase",
	Hidden: true,
	Args:   cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		input := readClaudeHookInput()
		if input.StopHookActive {
			return nil
		}
		if store == nil {
			return nil
		}

		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			return nil
		}
		if state.State != colony.StateEXECUTING && state.State != colony.StateBUILT {
			return nil
		}

		phaseLabel := fmt.Sprintf("phase %d", state.CurrentPhase)
		if state.CurrentPhase > 0 && state.CurrentPhase <= len(state.Plan.Phases) {
			phaseLabel = fmt.Sprintf("phase %d (%s)", state.CurrentPhase, state.Plan.Phases[state.CurrentPhase-1].Name)
		}

		return emitHookBlock(fmt.Sprintf(
			"Aether is still in %s. Finish the lifecycle with `aether continue`, or run `/ant:pause-colony` before stopping.",
			phaseLabel,
		))
	},
}

var hookPreCompactCmd = &cobra.Command{
	Use:    "hook-pre-compact",
	Short:  "Claude hook: refresh Aether session summary before compaction",
	Hidden: true,
	Args:   cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		input := readClaudeHookInput()
		if store == nil {
			return nil
		}

		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			return nil
		}

		trigger := strings.TrimSpace(input.Trigger)
		if trigger == "" {
			trigger = "unknown"
		}

		next := nextCommandForHookState(state)
		summary := fmt.Sprintf("Pre-compact snapshot (%s): %s", trigger, summarizeHookState(state))
		ensureSessionSummary(state, "hook-pre-compact", next, summary)
		return nil
	},
}

func readClaudeHookInput() claudeHookInput {
	var input claudeHookInput

	info, err := os.Stdin.Stat()
	if err != nil {
		return input
	}
	if (info.Mode() & os.ModeCharDevice) != 0 {
		return input
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil || len(strings.TrimSpace(string(data))) == 0 {
		return input
	}
	_ = json.Unmarshal(data, &input)
	return input
}

func hookToolTargetPath(toolInput map[string]interface{}) string {
	if len(toolInput) == 0 {
		return ""
	}
	for _, key := range []string{"file_path", "path"} {
		if value, ok := toolInput[key].(string); ok {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func protectedHookWriteReason(target, cwd string) string {
	normalized := normalizeHookPath(target, cwd)
	if normalized == "" {
		return ""
	}

	slash := filepath.ToSlash(normalized)
	base := filepath.Base(slash)
	switch {
	case strings.Contains(slash, "/.aether/data/"):
		return "Protected colony state path. Update `.aether/data/*` through the `aether` CLI, not direct edits."
	case strings.Contains(slash, "/.aether/dreams/"):
		return "Protected dream journal path. Do not edit `.aether/dreams/` from a worker."
	case strings.HasPrefix(base, ".env"):
		return "Protected environment file. Do not edit `.env*` through a hook-triggered write."
	case strings.HasSuffix(slash, "/.codex/config.toml"):
		return "Protected Codex config path. Do not edit `.codex/config.toml` from a worker."
	case strings.Contains(slash, "/.github/workflows/"):
		return "Protected CI path. Workflow files require explicit user direction."
	default:
		return ""
	}
}

func redirectWriteReason(target, cwd string) string {
	if store == nil {
		return ""
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		return ""
	}
	if state.State != colony.StateEXECUTING && state.State != colony.StateBUILT {
		return ""
	}

	branch := detectGitBranch()
	if branch != "main" && branch != "master" {
		return ""
	}

	pf := loadPheromones()
	if pf == nil {
		return ""
	}

	now := time.Now()
	for _, sig := range pf.Signals {
		if !sig.Active || sig.Type != "REDIRECT" {
			continue
		}
		if computeEffectiveStrength(sig, now) < 0.1 {
			continue
		}
		text := strings.ToLower(extractSignalText(sig.Content))
		if strings.Contains(text, "main branch") {
			targetLabel := target
			if targetLabel == "" {
				targetLabel = "requested file"
			}
			return fmt.Sprintf(
				"Active REDIRECT forbids direct edits on branch %q during builds. Move %s onto a branch/worktree workflow before writing.",
				branch,
				targetLabel,
			)
		}
	}

	return ""
}

func normalizeHookPath(target, cwd string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return ""
	}
	if cwd == "" {
		if wd, err := os.Getwd(); err == nil {
			cwd = wd
		}
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(cwd, target)
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		return filepath.Clean(target)
	}
	return filepath.Clean(abs)
}

func emitHookBlock(reason string) error {
	payload := map[string]string{
		"decision": "block",
		"reason":   reason,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, string(encoded))
	return nil
}

func nextCommandForHookState(state colony.ColonyState) string {
	switch state.State {
	case colony.StateEXECUTING, colony.StateBUILT:
		return "aether continue"
	case colony.StateCOMPLETED:
		return "aether seal"
	case colony.StateREADY:
		if len(state.Plan.Phases) == 0 {
			return "aether plan"
		}
		for _, phase := range state.Plan.Phases {
			if phase.Status == colony.PhaseReady || phase.Status == colony.PhasePending || phase.Status == "" {
				return fmt.Sprintf("aether build %d", phase.ID)
			}
		}
		return "aether status"
	default:
		if state.Goal == nil || strings.TrimSpace(*state.Goal) == "" {
			return "aether init \"goal\""
		}
		return "aether status"
	}
}

func summarizeHookState(state colony.ColonyState) string {
	goal := "no goal"
	if state.Goal != nil && strings.TrimSpace(*state.Goal) != "" {
		goal = strings.TrimSpace(*state.Goal)
	}

	summary := fmt.Sprintf("state=%s phase=%d goal=%s", state.State, state.CurrentPhase, goal)
	if state.CurrentPhase > 0 && state.CurrentPhase <= len(state.Plan.Phases) {
		summary += fmt.Sprintf(" task=%s", state.Plan.Phases[state.CurrentPhase-1].Name)
	}
	return summary
}

func ensureSessionSummary(state colony.ColonyState, commandName, suggestedNext, summary string) {
	if store == nil {
		return
	}

	var session colony.SessionFile
	if err := store.LoadJSON("session.json", &session); err != nil {
		goal := ""
		if state.Goal != nil {
			goal = *state.Goal
		}
		session = colony.SessionFile{
			SessionID:        fmt.Sprintf("hook_%d", time.Now().Unix()),
			StartedAt:        time.Now().UTC().Format(time.RFC3339),
			ColonyGoal:       goal,
			CurrentPhase:     state.CurrentPhase,
			CurrentMilestone: state.Milestone,
			SuggestedNext:    suggestedNext,
			ContextCleared:   false,
			BaselineCommit:   getGitHEAD(),
			ActiveTodos:      []string{},
			Summary:          summary,
		}
	}

	session.LastCommand = commandName
	session.LastCommandAt = time.Now().UTC().Format(time.RFC3339)
	session.CurrentPhase = state.CurrentPhase
	if state.Milestone != "" {
		session.CurrentMilestone = state.Milestone
	}
	if suggestedNext != "" {
		session.SuggestedNext = suggestedNext
	}
	if summary != "" {
		session.Summary = summary
	}

	_ = store.SaveJSON("session.json", session)
}

func init() {
	rootCmd.AddCommand(hookPreToolUseCmd)
	rootCmd.AddCommand(hookStopCmd)
	rootCmd.AddCommand(hookPreCompactCmd)
}
