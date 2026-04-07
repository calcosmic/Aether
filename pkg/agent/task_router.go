package agent

import (
	"regexp"
	"strings"
)

// typeHintRegex matches bracket notation like [implement], [test], [research].
var typeHintRegex = regexp.MustCompile(`\[([a-z]+)\]`)

// ParseTypeHint extracts the first bracket tag from a task goal string.
// Returns the lowercase tag content, or empty string if no bracket is found.
func ParseTypeHint(goal string) string {
	matches := typeHintRegex.FindStringSubmatch(goal)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// RouteTask determines the appropriate caste for a task using a two-pass approach:
// Pass 1 checks for explicit type hints in bracket notation; Pass 2 falls back
// to keyword matching against the task goal text.
func RouteTask(goal string) Caste {
	// Pass 1: type hint
	if hint := ParseTypeHint(goal); hint != "" {
		return hintToCaste(hint)
	}

	// Pass 2: keyword matching
	// Order matters: scout before watcher because words like "investigate"
	// contain "test" as a substring, which would misroute to watcher.
	lower := strings.ToLower(goal)

	if matchesKeyword(lower, "research", "investigate", "find", "discover", "explore", "analyze") {
		return CasteScout
	}
	if matchesKeyword(lower, "design", "architect", "structure") {
		return CasteArchitect
	}
	if matchesKeyword(lower, "test", "verify", "assert", "check", "validate") {
		return CasteWatcher
	}
	if matchesKeyword(lower, "implement", "create", "build", "add", "write", "fix", "code") {
		return CasteBuilder
	}

	// Default
	return CasteBuilder
}

// hintToCaste maps type hint strings to their corresponding caste.
func hintToCaste(hint string) Caste {
	switch hint {
	case "implement", "build", "code", "fix":
		return CasteBuilder
	case "test", "verify", "validate", "check":
		return CasteWatcher
	case "research", "investigate", "explore", "analyze":
		return CasteScout
	case "design", "architect", "plan":
		return CasteArchitect
	case "review":
		return CasteScout
	default:
		return CasteBuilder
	}
}

// matchesKeyword checks if the text contains any of the given keywords as whole words.
func matchesKeyword(text string, keywords ...string) bool {
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}
