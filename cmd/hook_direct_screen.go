package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// directScreenMessageCapBytes is the platform's own documented per-message
// limit for a hook's systemMessage field (research/2026-09-21-reliability-and-delivery.md
// records it as 10,000 characters; past that Claude Code replaces the field
// with a file path and a short preview). Aether measures UTF-8 BYTES rather
// than characters: for every character Aether draws, the byte count is never
// smaller than the platform's own character count (every multi-byte rune
// this codebase draws -- the bar character, helper emoji -- costs MORE than
// one byte per character, never fewer), so a screen kept at or under this
// byte cap can never overflow the platform's character cap either.
const directScreenMessageCapBytes = 10000

// directScreenTruncationNotice is the one plain sentence prepended when a
// screen did not fit the cap whole. Plain English, no word this repository
// invented.
const directScreenTruncationNotice = "This screen was too long to show in full here, so only its last part is below. The chat has been asked to show the whole thing before it finishes."

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
// (screenRelayBlockReason) call -- TestDirectRouteAndTheBackstopUseOneDecision
// asserts both reach it structurally. It never writes anything and never
// blocks; it purely reports what CAN be delivered.
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
	return directScreenMessageWithinCap(screen)
}

// directScreenMessageWithinCap is the length rule, and the ONLY function in
// this package that references directScreenMessageCapBytes --
// TestDirectRouteAndTheBackstopUseOneDecision asserts that structurally, so
// a second, silently-diverging copy of the cap can never grow.
//
// At or under the cap, the screen is delivered whole and reports itself
// complete. Over the cap, the message starts with the truncation notice plus
// a blank line, then walks the screen's lines from the LAST to the first,
// prepending each whole line while the running UTF-8 byte size stays at or
// under the cap, stopping at the first line that would not fit. It never
// slices inside a line and never indexes by rune -- `len(line)` is a byte
// count, not a rune count, and every unit added to the message is a whole
// line. If not one whole line fits, the notice is returned alone. Delivery
// stays true in every over-length case: a partial screen plus an honest
// sentence beats silence, because the finish-check backstop still asks for
// the rest when this reports itself incomplete.
func directScreenMessageWithinCap(screen string) (message string, complete bool, deliver bool) {
	if len(screen) <= directScreenMessageCapBytes {
		return screen, true, true
	}

	prefix := directScreenTruncationNotice + "\n\n"
	budget := directScreenMessageCapBytes - len(prefix)
	if budget <= 0 {
		return directScreenTruncationNotice, false, true
	}

	lines := strings.Split(screen, "\n")
	var kept []string
	used := 0
	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]
		sep := 0
		if len(kept) > 0 {
			sep = 1 // the joining "\n" the next prepend introduces
		}
		if used+len(line)+sep > budget {
			break
		}
		kept = append([]string{line}, kept...)
		used += len(line) + sep
	}

	if len(kept) == 0 {
		return directScreenTruncationNotice, false, true
	}
	return prefix + strings.Join(kept, "\n"), false, true
}

// directScreenHookSettingsFile is the minimum shape directScreenRouteRegistered
// needs out of a project's .claude/settings.json (or settings.local.json):
// which commands are registered under which hook events.
type directScreenHookSettingsFile struct {
	Hooks map[string][]struct {
		Matcher string `json:"matcher"`
		Hooks   []struct {
			Command string `json:"command"`
		} `json:"hooks"`
	} `json:"hooks"`
}

// directScreenMatcherCoversBash reports whether a PostToolUse entry's matcher
// would fire for the Bash tool -- the only tool that can draw an Aether
// screen. Claude Code treats an empty matcher and "*" as every tool, and a
// "|"-separated list as alternatives. An entry registered against some other
// tool names the right command but never runs for the call that drew the
// screen, so it must not count as the route being installed.
func directScreenMatcherCoversBash(matcher string) bool {
	matcher = strings.TrimSpace(matcher)
	if matcher == "" || matcher == "*" {
		return true
	}
	for _, alt := range strings.Split(matcher, "|") {
		if strings.TrimSpace(alt) == "Bash" {
			return true
		}
	}
	return false
}

// directScreenRouteRegistered reports whether a project's own shipped
// settings register the direct route (this file's hookPostToolUseCmd)
// against PostToolUse. It reads .claude/settings.json first, then
// .claude/settings.local.json only when the former is unreadable or carries
// no match. Every failure path -- an unreadable cwd, a missing or malformed
// settings file, no PostToolUse entry at all -- returns false. False is the
// conservative answer: it keeps the Stop-hook finish-check backstop
// behaving exactly as it did before this phase in a project whose settings
// have not been refreshed, which is what protects the behaviour already
// released in 1.0.88.
func directScreenRouteRegistered(cwd string) bool {
	if strings.TrimSpace(cwd) == "" {
		if wd, err := os.Getwd(); err == nil {
			cwd = wd
		}
	}
	if directScreenSettingsFileRegistersRoute(filepath.Join(cwd, ".claude", "settings.json")) {
		return true
	}
	return directScreenSettingsFileRegistersRoute(filepath.Join(cwd, ".claude", "settings.local.json"))
}

func directScreenSettingsFileRegistersRoute(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var parsed directScreenHookSettingsFile
	if err := json.Unmarshal(data, &parsed); err != nil {
		return false
	}
	for _, entry := range parsed.Hooks["PostToolUse"] {
		if !directScreenMatcherCoversBash(entry.Matcher) {
			continue
		}
		for _, h := range entry.Hooks {
			if strings.HasPrefix(strings.TrimSpace(h.Command), "aether hook-post-tool-use") {
				return true
			}
		}
	}
	return false
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
