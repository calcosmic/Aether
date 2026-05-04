package cmd

import (
	"strings"
)

// friendlyError maps an error pattern to a human-readable explanation and next steps.
type friendlyError struct {
	Pattern     string
	Explanation string
	NextSteps   []string
}

// errorPatternMap lists known error patterns ordered from most specific to least.
// Each entry maps a substring match to a plain-language explanation and actionable
// next steps for the user.
var errorPatternMap = []friendlyError{
	{
		Pattern:     "invalid charter JSON",
		Explanation: "The charter passed to Aether is not valid JSON. The colony state file was not changed.",
		NextSteps: []string{
			"Retry with valid JSON, or run `aether init \"your goal\"` without `--charter-json`.",
			"If an assistant generated the command, ask it to compact the charter or escape quotes/newlines correctly.",
		},
	},
	{
		Pattern:     "no colony initialized",
		Explanation: "Aether needs a colony to work with. A colony is a workspace for building toward a specific goal.",
		NextSteps: []string{
			"Run `aether init \"your goal\"` to start a colony.",
			"Run `aether lay-eggs` first if this repo is brand new.",
		},
	},
	{
		Pattern:     "failed to load colony state",
		Explanation: "Aether could not read the colony data file. This may be corrupted or was modified outside of Aether.",
		NextSteps: []string{
			"Run `aether patrol` for diagnostics.",
			"Check `.aether/data/COLONY_STATE.json` for syntax errors.",
		},
	},
	{
		Pattern:     "flag --",
		Explanation: "This command needs more information to run. Check the required flags and try again.",
		NextSteps: []string{
			"Run `aether <command> --help` to see available flags.",
		},
	},
	{
		Pattern:     "failed to initialize store",
		Explanation: "Aether could not set up its data storage. This usually means the data directory is inaccessible.",
		NextSteps: []string{
			"Run `aether patrol` for diagnostics.",
			"Check that `.aether/data/` exists and is writable.",
		},
	},
	{
		Pattern:     "permission denied",
		Explanation: "Aether does not have permission to access a file or directory.",
		NextSteps: []string{
			"Check file permissions. On macOS/Linux: `ls -la <path>` to inspect.",
		},
	},
	{
		Pattern:     "json:",
		Explanation: "Aether's data file is corrupted or was modified outside of Aether.",
		NextSteps: []string{
			"Run `aether patrol` for diagnostics.",
			"Check `.aether/data/COLONY_STATE.json` for syntax errors.",
		},
	},
}

// friendlyErrorForPattern looks up a friendly error entry matching the given
// error message. Matching is case-insensitive using substring containment.
func friendlyErrorForPattern(message string) (friendlyError, bool) {
	lowerMessage := strings.ToLower(message)
	for _, entry := range errorPatternMap {
		if strings.Contains(lowerMessage, strings.ToLower(entry.Pattern)) {
			return entry, true
		}
	}
	return friendlyError{}, false
}

// renderFriendlyError produces a visual error display with a plain-language
// explanation and actionable next steps.
func renderFriendlyError(entry friendlyError, rawMessage string) string {
	var b strings.Builder
	b.WriteString(renderBanner("\u274C", "Error"))
	b.WriteString(visualDivider)
	b.WriteString(entry.Explanation)
	b.WriteString("\n\n")
	b.WriteString("Next steps:\n")
	for _, step := range entry.NextSteps {
		b.WriteString("  - ")
		b.WriteString(step)
		b.WriteString("\n")
	}
	return b.String()
}
