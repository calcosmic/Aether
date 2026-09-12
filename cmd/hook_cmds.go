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
	// AgentID, AgentType, and SessionID are Phase 173 (SPAWN-04) additions.
	// These field names come from current official Claude Code hooks
	// documentation and are a hypothesis, not a confirmed contract — Task 3
	// of the 173-01 plan runs a real nested dispatch and records the actual
	// observed field names in 173-HOOK-FINDINGS.md. If the observed names
	// differ, plan 07 corrects them. All three are plain strings so an
	// absent field decodes to the empty string rather than failing decode.
	AgentID   string `json:"agent_id"`
	AgentType string `json:"agent_type"`
	SessionID string `json:"session_id"`
	// TranscriptPath is a Phase 174 (SPEND-02) addition. Unlike AgentID,
	// AgentType and SessionID above -- which were a hypothesis Phase 173
	// confirmed after the fact -- this field name was empirically confirmed
	// FIRST, by the real captured payloads in
	// .planning/phases/173-delegation-guard/173-HOOK-FINDINGS.md
	// (2026-08-13). It is the platform's own record of where this session's
	// transcript lives, and is the one thing an orchestrating LLM cannot
	// fabricate: the runtime reads it from the platform, not from anything
	// a wrapper typed.
	TranscriptPath string `json:"transcript_path"`
}

const postResumeStopGracePeriod = 15 * time.Minute

var hookPreToolUseCmd = &cobra.Command{
	Use:    "hook-pre-tool-use [tool_name] [target]",
	Short:  "Claude hook: validate edits before tool execution",
	Hidden: true,
	Args:   cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		input, raw := readClaudeHookInput()
		captureRawHookPayload(raw)
		recordSpendSessionFromHook(input)

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
				if tracer != nil {
					var state colony.ColonyState
					if loadErr := store.LoadJSON("COLONY_STATE.json", &state); loadErr == nil && state.RunID != nil {
						_ = tracer.LogIntervention(*state.RunID, "hook.pre-tool-use.block", "hook-cmd", map[string]interface{}{
							"hook":   "pre-tool-use",
							"reason": reason,
							"tool":   toolName,
							"target": target,
						})
					}
				}
				return emitHookBlock(reason)
			}
			if reason := redirectWriteReason(target, cwd); reason != "" {
				if tracer != nil {
					var state colony.ColonyState
					if loadErr := store.LoadJSON("COLONY_STATE.json", &state); loadErr == nil && state.RunID != nil {
						_ = tracer.LogIntervention(*state.RunID, "hook.pre-tool-use.redirect", "hook-cmd", map[string]interface{}{
							"hook":   "pre-tool-use",
							"reason": reason,
							"tool":   toolName,
							"target": target,
						})
					}
				}
				return emitHookBlock(reason)
			}
		}

		// SPAWN-04/D-20: 173-HOOK-FINDINGS.md (2026-08-13) observed the
		// dispatch tool's name as "Agent", not "Task". Accept "Task" too for
		// forward compatibility -- .claude/settings.json's matcher already
		// covers both, so a future runtime that reverts to "Task" must not
		// silently stop being covered here.
		if strings.EqualFold(toolName, "Agent") || strings.EqualFold(toolName, "Task") {
			if reason := hookSpawnDenyReason(input); reason != "" {
				if tracer != nil {
					var state colony.ColonyState
					if loadErr := store.LoadJSON("COLONY_STATE.json", &state); loadErr == nil && state.RunID != nil {
						_ = tracer.LogIntervention(*state.RunID, "hook.pre-tool-use.spawn-deny", "hook-cmd", map[string]interface{}{
							"hook":       "pre-tool-use",
							"reason":     reason,
							"tool":       toolName,
							"agent_id":   input.AgentID,
							"agent_type": input.AgentType,
						})
					}
				}
				return emitHookBlock(reason)
			}
		}

		return nil
	},
}

// hookAetherAgentTypePrefix is the naming convention every one of the repo's
// named worker castes follows (aether-builder, aether-watcher, ...,
// confirmed by `ls .claude/agents/ant/`). A requester whose agent_type
// carries this prefix was dispatched as one of Aether's own first-tier
// workers.
const hookAetherAgentTypePrefix = "aether-"

// hookSpawnDenyReason is Phase 173's SPAWN-04/D-20 guard: the PreToolUse
// hook's fail-closed answer for a delegation dispatch (an "Agent"/"Task" tool
// call) BEFORE the platform acts on it. It follows protectedHookWriteReason's
// shape -- a reason string on deny, the empty string on allow, one dispatch
// point -- but DELIBERATELY INVERTS that function's unresolvable-input
// branch (`if normalized == "" { return "" }`, which fails open). Here,
// failing to resolve who is asking must fail closed.
//
// Empirically observed coverage (173-HOOK-FINDINGS.md, captured 2026-08-13,
// a real two-level nested dispatch performed live inside this repo): the
// hook fired for BOTH the coordinator's own dispatch (tool_name "Agent", no
// agent_id -- a depth-0 requester) AND a subagent's dispatch of its own
// helper (tool_name "Agent", agent_id present, agent_type carrying the
// REQUESTER's own dispatched type, not the target's). No third-level
// (helper-of-helper) dispatch was captured in that run. This function's
// claims and comments are bounded to those two observed levels; nothing
// here asserts coverage of a dispatch depth the capture did not demonstrate.
// See 173-HOOK-FINDINGS.md for the raw payloads and the four answers derived
// from them.
//
// There is no mapping from Claude Code's own agent_id (e.g.
// "ae93ff782863d564f") to Aether's spawn-tree AgentName -- 173-HOOK-FINDINGS.md
// recorded which fields the platform supplies, not an identity bridge. Every
// rule below is therefore a heuristic over agent_id/agent_type's
// presence/absence and value, not a lookup into Aether's own spawn records.
// The AUTHORITATIVE depth enforcement remains spawn-log's deriveSpawnDepth,
// which derives depth from the parent's own recorded spawn-tree entry and
// cannot be fooled by a caller's claimed identity; this hook is a
// before-the-fact deterrent layered in front of it, not a replacement for it.
func hookSpawnDenyReason(in claudeHookInput) string {
	agentID := strings.TrimSpace(in.AgentID)
	agentType := strings.TrimSpace(in.AgentType)

	if agentID == "" {
		// D-20's deliberate exception to fail-closed: absence of a subagent
		// identifier is positive evidence this dispatch originates from the
		// main session (depth 0), not an unresolved lookup --
		// 173-HOOK-FINDINGS.md confirmed this by direct comparison of the
		// two captured payloads: the coordinator's own dispatch carried
		// neither agent_id nor agent_type at all. Depth 0 is always
		// permitted to dispatch its own first-tier workers.
		return ""
	}

	var requesterDepth int
	switch {
	case strings.HasPrefix(strings.ToLower(agentType), hookAetherAgentTypePrefix):
		// A named first-tier worker, dispatched by the coordinator with one
		// of the repo's own aether-* castes as its subagent_type.
		requesterDepth = 1
	default:
		// Covers the observed "general-purpose" value and every other
		// unclassified value. 173-HOOK-FINDINGS.md showed agent_type names
		// the REQUESTER's own dispatched type, not the target's -- and
		// .aether/workers.md's own spawn protocol instructs every worker to
		// dispatch its own helper with subagent_type="general-purpose". A
		// first-tier worker that followed that documented fallback verbatim
		// would ALSO carry agent_type "general-purpose" on its own dispatch,
		// so this single value cannot distinguish "first-tier worker using
		// the documented fallback" (should resolve depth 1) from "second-tier
		// helper spawning past the cap" (should resolve depth 2). Inventing
		// a resolution the capture never demonstrated would be exactly the
		// identity-lookup mistake this function's own comment warns against
		// -- so every unclassified value denies, not just this one.
		return fmt.Sprintf(
			"cannot resolve who is asking to delegate (agent_id=%q agent_type=%q); refusing the spawn",
			agentID, agentType,
		)
	}

	// This branch is reachable only if a future, better-resolved requester
	// type pushes requesterDepth to spawnMaxDelegationDepth or beyond; today
	// the one resolvable value (an aether-* type) always yields
	// requesterDepth 1, and 1+1 never exceeds the cap of 2. It is kept
	// because the resolution rule above is deliberately bounded to what
	// 173-HOOK-FINDINGS.md demonstrated and may widen later, and because this
	// is the one place the hook must use spawnMaxDelegationDepth directly --
	// the same cap constant the CLI guards use, not a second number.
	prospectiveDepth := requesterDepth + 1
	if prospectiveDepth > spawnMaxDelegationDepth {
		return fmt.Sprintf(
			"requester type %q resolved to depth %d; a helper spawned from here would be depth %d, past the cap of %d",
			agentType, requesterDepth, prospectiveDepth, spawnMaxDelegationDepth,
		)
	}

	return ""
}

var hookStopCmd = &cobra.Command{
	Use:    "hook-stop",
	Short:  "Claude hook: prevent accidental stop mid-phase",
	Hidden: true,
	Args:   cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		input, _ := readClaudeHookInput()
		if input.StopHookActive {
			return nil
		}
		// A worker Aether spawned is not a person ending a session early. This
		// hook exists to stop the OWNER walking away mid-phase; applied to
		// Aether's own build workers it did the opposite of its purpose --
		// it blocked a worker from finishing and advised `aether pause`, which
		// a worker must never run mid-build. One worker followed that advice
		// and paused a live Autopilot run.
		if isAetherSpawnedWorker() {
			return nil
		}
		if store == nil {
			return nil
		}

		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			return nil
		}
		if (state.State != colony.StateEXECUTING && state.State != colony.StateBUILT) || state.Paused {
			return nil
		}
		if allowStopAfterRecentResume() {
			return nil
		}

		phaseLabel := fmt.Sprintf("phase %d", state.CurrentPhase)
		if state.CurrentPhase > 0 && state.CurrentPhase <= len(state.Plan.Phases) {
			phaseLabel = fmt.Sprintf("phase %d (%s)", state.CurrentPhase, state.Plan.Phases[state.CurrentPhase-1].Name)
		}

		if tracer != nil && state.RunID != nil {
			_ = tracer.LogIntervention(*state.RunID, "hook.stop.block", "hook-cmd", map[string]interface{}{
				"hook":       "stop",
				"phase":      state.CurrentPhase,
				"phaseLabel": phaseLabel,
			})
		}

		return emitHookBlock(fmt.Sprintf(
			"Aether is still in %s. Finish the lifecycle with `aether continue`, or run `aether pause` before stopping.",
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
		input, _ := readClaudeHookInput()
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

// hookSessionStartCmd is Phase 197 plan 03: the greeting the program owns.
//
// Four shipped documents used to ASK the assistant to check the saved session
// file on the first message of a conversation and report what it found. That is
// a request, not a mechanism -- it ran only when it was noticed. This hook fires
// when a session opens, when one is resumed, and when a conversation is carried
// on after being cleared, and prints the one card every other surface renders.
//
// Two rules bind it, and both are asserted by tests rather than promised here:
//
//  1. It DECIDES nothing. It composes no advice of its own; the card and the
//     command inside it come from resolveNextAction. A branch here would be a
//     fifth rival decider, which is the drift this phase exists to end.
//     (TestSessionStartHookChoosesNoCommandOfItsOwn.)
//
//  2. It WRITES nothing. Unlike hookPreCompactCmd in this same file -- which
//     deliberately refreshes the session summary -- this hook is pure
//     inspection, and it fires before the owner has typed a word. This
//     repository has shipped two inspection surfaces that quietly wrote to
//     saved state while documenting that they did not; this is the third such
//     lock rather than the third such defect.
//     (TestSessionStartHookDoesNotMutate.)
//
// Silence is the correct output when there is no project. Aether is installed
// in repositories that are not running a colony, and greeting one of those is
// noise -- so the loader's own no-colony signal is honoured rather than an
// empty state being mistaken for a project that just started.
var hookSessionStartCmd = &cobra.Command{
	Use:    "hook-session-start",
	Short:  "Claude hook: greet a new session with where things stand",
	Hidden: true,
	Args:   cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Drain the payload the same way the sibling hooks do. Nothing in it is
		// needed: where things stand is read from the project, not from what
		// the platform says about the session.
		_, _ = readClaudeHookInput()

		// Failure tolerance, matching the sibling hooks: a greeting that errors
		// on a malformed project is worse than one that stays quiet, because it
		// fires before the owner has typed anything. loadNextActionInputForGreeting
		// reports NoColony for a missing store, an unreadable state file and a
		// state file carrying no goal alike, so every one of those paths is
		// silence rather than noise. This is the ONE caller of
		// loadNextActionInputForGreeting (198.2 plan 03) -- the memory block
		// it adds (preferences, learned habits, the last helper's note)
		// appears on this greeting only; every other closing card still
		// calls loadNextActionInput and is unchanged.
		in := loadNextActionInputForGreeting()
		if in.NoColony {
			return nil
		}

		writeVisualOutput(stdout, renderNextActionCard(resolveNextAction(in)))
		return nil
	},
}

func allowStopAfterRecentResume() bool {
	if store == nil {
		return false
	}

	var session colony.SessionFile
	if err := store.LoadJSON("session.json", &session); err != nil {
		return false
	}
	session.LastCommand = normalizeLegacySessionCommand(session.LastCommand, resolveVersion())
	if session.LastCommand != "resume" {
		return false
	}

	resumedAt, err := time.Parse(time.RFC3339, strings.TrimSpace(session.LastCommandAt))
	if err != nil {
		return false
	}
	age := time.Since(resumedAt)
	return age >= 0 && age <= postResumeStopGracePeriod
}

func readClaudeHookInput() (claudeHookInput, []byte) {
	var input claudeHookInput

	info, err := os.Stdin.Stat()
	if err != nil {
		return input, nil
	}
	if (info.Mode() & os.ModeCharDevice) != 0 {
		return input, nil
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil || len(strings.TrimSpace(string(data))) == 0 {
		return input, nil
	}
	_ = json.Unmarshal(data, &input)
	return input, data
}

// captureRawHookPayload appends the raw stdin bytes received by a hook to
// the file named by AETHER_HOOK_CAPTURE_FILE, when that environment variable
// is set. This is Phase 173 (SPAWN-04) Wave 0's opt-in evidence recorder: it
// exists so the field names a future deny path matches on are copied from an
// observed payload rather than assumed from documentation. It is off by
// default (empty env var short-circuits immediately) and every error path
// returns silently -- a capture failure must never affect the hook's
// allow/deny answer (T-173-02).
//
// The environment variable is the ONLY switch, deliberately. A briefly-lived
// sentinel-file fallback (~/.aether/hook-capture-path) existed on 2026-08-13
// to run the 173-HOOK-FINDINGS.md capture when the env var could not be
// delivered through session launch; the phase code review (173-REVIEW.md
// CR-01) showed a home-directory file is writable by any worker's ordinary
// Write tool, letting capture be aimed at protected state or the spawn
// ledger. The fallback was removed the same day, restoring T-173-03's
// boundary: only someone with shell access to the operator's environment
// can turn capture on. Do not re-add a file-based switch without a
// destination validation story.
func captureRawHookPayload(raw []byte) {
	path := strings.TrimSpace(os.Getenv("AETHER_HOOK_CAPTURE_FILE"))
	if path == "" {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(raw)
	_, _ = f.Write([]byte("\n"))
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

// sanctionedDataWritePrefixes lists the exact-subpath carve-outs under
// .aether/data/ that a worker is actually ordered to write by a real runtime
// instruction. Each entry is a full directory segment (leading and trailing
// slash) so a substring match can never widen past a directory boundary.
// Widening any entry to a bare "/.aether/data/" match is a regression —
// TestHookPreToolUseBlocksProtectedPath must fail if that ever happens.
//
//   - /.aether/data/planning/       — D-04: planning artifacts a worker is
//     told to persist during the plan workflow.
//   - /.aether/data/phase-research/ — cmd/phase_research.go:124 orders the
//     scout to write phase-N-research.md here.
//   - /.aether/data/survey/         — pkg/codex/permission_profile.go's
//     surveyor behavioral restriction names this directory.
//   - /.aether/data/worker-debug/   — D-04: worker debug artifacts a worker
//     is told to persist for diagnostics.
var sanctionedDataWritePrefixes = []string{
	"/.aether/data/planning/",
	"/.aether/data/phase-research/",
	"/.aether/data/survey/",
	"/.aether/data/worker-debug/",
}

func protectedHookWriteReason(target, cwd string) string {
	normalized := normalizeHookPath(target, cwd)
	if normalized == "" {
		return ""
	}

	slash := filepath.ToSlash(normalized)
	base := filepath.Base(slash)
	for _, prefix := range sanctionedDataWritePrefixes {
		if strings.Contains(slash, prefix) {
			return ""
		}
	}
	switch {
	case strings.Contains(slash, "/.aether/data/"):
		return "Protected colony state path. Update `.aether/data/*` through the `aether` CLI, not direct edits. Sanctioned scratch subpaths (planning/, phase-research/, survey/, worker-debug/) are writable."
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
		if computeEffectiveStrength(sig, now) < pheromoneEffectiveFloor {
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
		abs = filepath.Clean(target)
	} else {
		abs = filepath.Clean(abs)
	}
	return resolveHookPathSymlinks(abs)
}

// resolveHookPathSymlinks resolves symlinks in path before allowlist and
// blocklist matching (WR-05). normalizeHookPath was purely lexical
// (filepath.Clean only), so a symlink planted inside a sanctioned scratch
// subdir (e.g. .aether/data/planning/link -> ../COLONY_STATE.json) made a
// Write to the "allowed" path land on protected state instead.
//
// A Write's target frequently does not exist yet (Write creates new files),
// so filepath.EvalSymlinks on the full path fails outright when the leaf is
// new. Walk up to the deepest existing ancestor, resolve symlinks on that
// ancestor, then rejoin the not-yet-created remainder — this is the standard
// approach for symlink-safe path resolution of paths that may not exist.
func resolveHookPathSymlinks(path string) string {
	if path == "" {
		return path
	}
	cleaned := filepath.Clean(path)
	if resolved, err := filepath.EvalSymlinks(cleaned); err == nil {
		return resolved
	}
	parent := filepath.Dir(cleaned)
	if parent == cleaned {
		// Reached the filesystem root without finding an existing,
		// resolvable ancestor -- nothing left to resolve.
		return cleaned
	}
	return filepath.Join(resolveHookPathSymlinks(parent), filepath.Base(cleaned))
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

// nextCommandForHookState was the fifth rival decider. Phase 197 plan 02
// collapsed the other four onto resolveNextAction and deliberately left this
// one for plan 03, which owns this file. It is an adapter now: it gathers the
// facts for a state the caller already holds and returns the one answer.
//
// It kept its own branches longer than the others, and they disagreed with
// them -- a READY project whose phases were all finished was told to check
// status here while three other deciders said something else. Every command it
// can name now comes from the resolver's candidate set, so a rename fails the
// build instead of being quietly absorbed.
func nextCommandForHookState(state colony.ColonyState) string {
	return resolveNextAction(nextActionInputForState(state, "")).Command
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
	commandName = normalizeLegacySessionCommand(commandName, resolveVersion())

	contextCleared := true
	if _, err := syncColonyArtifacts(state, colonyArtifactOptions{
		CommandName:    commandName,
		SuggestedNext:  suggestedNext,
		Summary:        summary,
		HandoffTitle:   "Pre-Compact Snapshot",
		WriteHandoff:   true,
		ContextCleared: &contextCleared,
	}); err == nil {
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
			ColonyMode:       state.EffectiveColonyMode(),
			CurrentPhase:     state.CurrentPhase,
			CurrentMilestone: state.Milestone,
			SuggestedNext:    suggestedNext,
			ContextCleared:   true,
			BaselineCommit:   getGitHEAD(),
			ActiveTodos:      []string{},
			Summary:          summary,
		}
	}
	session.LastCommand = commandName
	session.LastCommandAt = time.Now().UTC().Format(time.RFC3339)
	session.ColonyMode = state.EffectiveColonyMode()
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
	rootCmd.AddCommand(hookSessionStartCmd)
}

// isAetherSpawnedWorker reports whether the current process is running inside a
// worker Aether spawned, rather than a session a person is driving. The spawn
// path sets AETHER_WORKER_NAME on the worker's own environment
// (pkg/codex.workerProcessEnv), and Claude Code passes its environment through
// to the hooks it runs, so the variable is visible here.
func isAetherSpawnedWorker() bool {
	return strings.TrimSpace(os.Getenv("AETHER_WORKER_NAME")) != ""
}
