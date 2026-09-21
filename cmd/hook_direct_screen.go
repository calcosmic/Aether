package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// hookToolResponseScreen reads the stdout text a Bash tool_response carries.
// A real captured PostToolUse payload (cmd/testdata/post-tool-use/status-screen-payload.json)
// shows tool_response as an object with a "stdout" string field; this mirrors
// toolResultText's two-shape handling for a different field on the Stop hook,
// solving the same "the platform's JSON shape for this field is not
// documented, only observed" problem. Any shape this cannot decode -- a bare
// string, or something else entirely -- returns "".
func hookToolResponseScreen(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var obj struct {
		Stdout string `json:"stdout"`
	}
	if json.Unmarshal(raw, &obj) == nil && obj.Stdout != "" {
		return obj.Stdout
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return text
	}
	return ""
}

// directScreenDelivery is the ONE decision both the direct route
// (hookPostToolUseCmd) and the Stop-hook finish-check backstop
// (screenRelayBlockReason) call. Task 1 (this commit) implements only the
// whole-screen path: `complete` is always true here, and the length rule
// (Task 2) extends this same function rather than adding a second one --
// TestDirectRouteAndTheBackstopUseOneDecision asserts there is exactly one.
func directScreenDelivery(command, screen string) (message string, complete bool, deliver bool) {
	if screenRelayDisabledByEnv() {
		return "", false, false
	}
	if !isAetherVisualCommand(command) {
		return "", false, false
	}
	if len(bannerLinesIn(screen)) == 0 {
		return "", false, false
	}
	return screen, true, true
}

// emitDirectScreen writes the one line of stdout Claude Code delivers to the
// owner directly, with no model turn spent: a JSON object carrying exactly
// one key, systemMessage, the same shape emitHookBlock already uses for its
// own single-key payload. hookSpecificOutput is deliberately never used here
// -- a message placed there reaches the model, not the owner, which is the
// opposite of what this route exists to do.
func emitDirectScreen(message string) error {
	payload := struct {
		SystemMessage string `json:"systemMessage"`
	}{SystemMessage: message}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, string(encoded))
	return nil
}

// hookPostToolUseCmd is Phase 206's (UED-05) direct route: when a Bash tool
// call drew one of Aether's screens, Claude Code's own PostToolUse hook
// fires this command, and it hands that screen to Claude Code as a message
// shown straight to the owner -- no model turn spent, nothing for the chat
// to paste. It loads no store, builds no tracer, and reads no file; every
// decision comes from directScreenDelivery, the one function this route
// shares with the Stop-hook finish-check backstop.
var hookPostToolUseCmd = &cobra.Command{
	Use:    "hook-post-tool-use",
	Short:  "Claude hook: hand a drawn screen straight to the owner",
	Hidden: true,
	Args:   cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		input, raw := readClaudeHookInput()
		captureRawHookPayload(raw)

		if isAetherSpawnedWorker() {
			return nil
		}
		if strings.TrimSpace(input.AgentID) != "" {
			return nil
		}
		if input.ToolName != "Bash" {
			return nil
		}

		command, _ := input.ToolInput["command"].(string)
		message, _, deliver := directScreenDelivery(command, hookToolResponseScreen(input.ToolResponse))
		if !deliver {
			return nil
		}
		return emitDirectScreen(message)
	},
}
