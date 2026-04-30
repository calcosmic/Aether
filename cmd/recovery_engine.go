package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
)

// RecoveryOption represents a single recovery suggestion shown to the user
// when a lifecycle command fails.
type RecoveryOption struct {
	Label     string // e.g., "Check colony health"
	Command   string // e.g., "aether patrol"
	Rationale string // e.g., "Diagnostics may reveal the root cause"
}

// normalizeBaseCommand strips the "aether " prefix, flags, and normalizes
// aliases to a canonical base command name.
func normalizeBaseCommand(cmd string) string {
	c := strings.TrimSpace(cmd)

	// Strip "aether " prefix if present
	if strings.HasPrefix(strings.ToLower(c), "aether ") {
		c = strings.TrimSpace(c[len("aether "):])
	}

	// Strip everything from first space/flag onward
	if idx := strings.IndexFunc(c, func(r rune) bool {
		return r == ' ' || r == '-'
	}); idx >= 0 {
		c = c[:idx]
	}

	// Map known aliases
	switch c {
	case "resume-colony":
		return "resume"
	}

	return strings.TrimSpace(c)
}

// classifyError returns an error class string based on substring matching
// against the error message. Returns one of: "no_colony", "state_corruption",
// "missing_prerequisite", "permission_denied", "unknown".
func classifyError(errMsg string) string {
	lower := strings.ToLower(errMsg)

	switch {
	case strings.Contains(lower, "no colony initialized"):
		return "no_colony"
	case strings.Contains(lower, "failed to load colony state"),
		strings.Contains(lower, "json:"):
		return "state_corruption"
	case strings.Contains(lower, "no project plan"),
		strings.Contains(lower, "not been sealed"),
		strings.Contains(lower, "crowned-anthill.md not found"):
		return "missing_prerequisite"
	case strings.Contains(lower, "permission denied"):
		return "permission_denied"
	default:
		return "unknown"
	}
}

// recoveryCandidates returns recovery options for a given failed command and
// error class. Each lifecycle command has context-specific suggestions.
func recoveryCandidates(failedCmd string, errorClass string) []RecoveryOption {
	type classOptions map[string][]RecoveryOption

	candidates := map[string]classOptions{
		"seal": {
			"no_colony": {
				{Label: "Initialize a colony", Command: "aether init \"goal\"", Rationale: "A colony must exist before sealing"},
			},
			"missing_prerequisite": {
				{Label: "Generate a plan first", Command: "aether plan", Rationale: "Sealing requires a completed plan"},
				{Label: "Check current status", Command: "aether status", Rationale: "See what phase the colony is in"},
			},
			"state_corruption": {
				{Label: "Run diagnostics", Command: "aether patrol", Rationale: "Identify the root cause of state corruption"},
			},
		},
		"entomb": {
			"no_colony": {
				{Label: "Initialize a colony", Command: "aether init \"goal\"", Rationale: "A colony must exist before entombing"},
			},
			"missing_prerequisite": {
				{Label: "Seal the colony first", Command: "aether seal", Rationale: "Entombing requires a sealed colony"},
			},
			"state_corruption": {
				{Label: "Run diagnostics", Command: "aether patrol", Rationale: "Identify the root cause"},
			},
		},
		"status": {
			"no_colony": {
				{Label: "Initialize a colony", Command: "aether init \"goal\"", Rationale: "Status requires an active colony"},
				{Label: "Set up Aether first", Command: "aether lay-eggs", Rationale: "If this repo is brand new"},
			},
			"state_corruption": {
				{Label: "Run diagnostics", Command: "aether patrol", Rationale: "Identify the state issue"},
			},
		},
		"resume": {
			"no_colony": {
				{Label: "Initialize a colony", Command: "aether init \"goal\"", Rationale: "Resume requires an active colony"},
			},
			"state_corruption": {
				{Label: "Run diagnostics", Command: "aether patrol", Rationale: "Check state health"},
				{Label: "Check colony status", Command: "aether status", Rationale: "See current state"},
			},
		},
	}

	// Look up command-specific candidates
	if cmdClasses, ok := candidates[failedCmd]; ok {
		if opts, ok := cmdClasses[errorClass]; ok {
			return opts
		}
	}

	// Generic fallback for known lifecycle commands
	if _, ok := candidates[failedCmd]; ok {
		return []RecoveryOption{
			{Label: "Check colony status", Command: "aether status", Rationale: "See what needs attention"},
			{Label: "Run diagnostics", Command: "aether patrol", Rationale: "Check system health"},
		}
	}

	// Truly unknown command -- generic fallback
	return genericFallback(failedCmd)
}

// genericFallback returns recovery options that are guaranteed not to include
// the failed command.
func genericFallback(failedCmd string) []RecoveryOption {
	normalized := normalizeBaseCommand(failedCmd)
	options := []RecoveryOption{
		{Label: "Check colony status", Command: "aether status", Rationale: "See what needs attention"},
		{Label: "Run diagnostics", Command: "aether patrol", Rationale: "Check system health"},
		{Label: "Initialize a colony", Command: "aether init \"goal\"", Rationale: "Start fresh if needed"},
	}

	// Filter out any option matching the failed command
	filtered := make([]RecoveryOption, 0, len(options))
	for _, opt := range options {
		if normalizeBaseCommand(opt.Command) != normalized {
			filtered = append(filtered, opt)
		}
	}

	if len(filtered) < 2 {
		// As a last resort, add a lay-eggs option (never a lifecycle command)
		filtered = append(filtered, RecoveryOption{
			Label:     "Set up Aether first",
			Command:   "aether lay-eggs",
			Rationale: "If this repo is brand new",
		})
	}

	return filtered
}

// recoveryOptionsForCommand returns recovery options for a failed command,
// guaranteeing that the failed command itself is never suggested (LOOP-05)
// and at least 2 options are always returned.
func recoveryOptionsForCommand(failedCmd string, errMsg string) []RecoveryOption {
	errorClass := classifyError(errMsg)
	normalized := normalizeBaseCommand(failedCmd)
	candidates := recoveryCandidates(normalized, errorClass)

	// Filter out any option whose normalized command matches the failed command
	filtered := make([]RecoveryOption, 0, len(candidates))
	for _, opt := range candidates {
		if normalizeBaseCommand(opt.Command) != normalized {
			filtered = append(filtered, opt)
		}
	}

	// If filter removed all options, use generic fallback (which also filters)
	if len(filtered) == 0 {
		filtered = genericFallback(failedCmd)
	}

	// Guarantee minimum 2 options by supplementing from genericFallback
	if len(filtered) < 2 {
		seen := make(map[string]bool)
		for _, opt := range filtered {
			seen[normalizeBaseCommand(opt.Command)] = true
		}
		fallback := genericFallback(failedCmd)
		for _, opt := range fallback {
			if len(filtered) >= 2 {
				break
			}
			norm := normalizeBaseCommand(opt.Command)
			if !seen[norm] {
				filtered = append(filtered, opt)
				seen[norm] = true
			}
		}
	}

	return filtered
}

// renderRecoveryMenu builds a recovery menu string for display. In visual mode
// it renders a formatted banner with numbered options. In JSON mode it outputs
// a JSON error envelope with recovery_options in the details field.
func renderRecoveryMenu(failedCmd string, errMsg string, details interface{}) string {
	options := recoveryOptionsForCommand(failedCmd, errMsg)

	if shouldRenderVisualOutput(stderr) {
		return buildVisualRecoveryMenu(failedCmd, errMsg, options)
	}

	// JSON mode: output error envelope with recovery_options
	recoveryDetails := map[string]interface{}{
		"recovery_options": options,
	}
	if details != nil {
		recoveryDetails["original_details"] = details
	}
	detailBytes, _ := json.Marshal(recoveryDetails)
	fmt.Fprintf(stderr, "{\"ok\":false,\"error\":\"%s\",\"code\":1,\"details\":%s}\n",
		jsonEscape(errMsg), string(detailBytes))
	return ""
}

// buildVisualRecoveryMenu renders the visual (terminal) recovery menu.
func buildVisualRecoveryMenu(failedCmd string, errMsg string, options []RecoveryOption) string {
	var b strings.Builder

	b.WriteString(renderBanner("\U0001F504", "Recovery"))
	b.WriteString(visualDivider)
	b.WriteString(errMsg)
	b.WriteString("\n\n")
	b.WriteString("Suggested next steps:\n")

	for i, opt := range options {
		b.WriteString(fmt.Sprintf("  %d. %s\n", i+1, opt.Label))
		b.WriteString(fmt.Sprintf("     %s\n", opt.Command))
		if opt.Rationale != "" {
			b.WriteString(fmt.Sprintf("     %s\n", opt.Rationale))
		}
	}

	return b.String()
}

// jsonEscape escapes a string for safe inclusion in a JSON string literal.
func jsonEscape(s string) string {
	b, _ := json.Marshal(s)
	// json.Marshal adds surrounding quotes; strip them
	str := string(b)
	if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
		return str[1 : len(str)-1]
	}
	return str
}
