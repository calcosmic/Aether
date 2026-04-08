package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// ---------------------------------------------------------------------------
// Suggest commands (all deprecated)
// ---------------------------------------------------------------------------

var suggestAnalyzeCmd = newDeprecatedCmd(
	"suggest-analyze",
	"Analyze codebase for patterns worth capturing as pheromones [DEPRECATED]",
	0,
	[]flagDef{
		{name: "context", boolType: false, default_: "", help: "Optional context text to include as suggestion"},
		{name: "max", boolType: false, default_: "5", help: "Maximum suggestions to return"},
	},
)

var suggestRecordCmd = newDeprecatedCmd(
	"suggest-record",
	"Record a new suggestion to suggestions.json [DEPRECATED]",
	0,
	[]flagDef{
		{name: "content", boolType: false, default_: "", help: "Suggestion content (required)"},
		{name: "type", boolType: false, default_: "FOCUS", help: "Pheromone type (FOCUS, REDIRECT, FEEDBACK)"},
		{name: "reason", boolType: false, default_: "", help: "Reason for the suggestion"},
		{name: "priority", boolType: false, default_: "normal", help: "Priority (high, normal, low)"},
	},
)

var suggestCheckCmd = newDeprecatedCmd(
	"suggest-check",
	"Read pending suggestions with dedup against active signals [DEPRECATED]",
	0,
	[]flagDef{
		{name: "limit", boolType: false, default_: "20", help: "Maximum suggestions to return"},
	},
)

var suggestApproveCmd = newDeprecatedCmd(
	"suggest-approve",
	"Approve pending suggestions as pheromone signals [DEPRECATED]",
	0,
	[]flagDef{
		{name: "id", boolType: false, default_: "", help: "Suggestion ID to approve (omit to approve all)"},
		{name: "type", boolType: false, default_: "FOCUS", help: "Pheromone type (FOCUS, REDIRECT, FEEDBACK)"},
	},
)

var suggestQuickDismissCmd = newDeprecatedCmd(
	"suggest-quick-dismiss",
	"Dismiss all pending suggestions [DEPRECATED]",
	0,
	nil,
)

// isTestArtifact checks if a pheromone signal matches test artifact patterns.
// This is still used by cmd/maintenance.go for data-clean.
func isTestArtifact(signal map[string]interface{}) bool {
	id, _ := signal["id"].(string)
	contentRaw := signal["content"]
	content := ""
	if contentMap, ok := contentRaw.(map[string]interface{}); ok {
		content, _ = contentMap["text"].(string)
	} else if contentStr, ok := contentRaw.(string); ok {
		content = contentStr
	}

	if strings.HasPrefix(id, "test_") || strings.HasPrefix(id, "demo_") {
		return true
	}

	lower := strings.ToLower(content)
	if strings.Contains(lower, "test signal") || strings.Contains(lower, "demo pattern") {
		return true
	}

	return false
}

func init() {
	rootCmd.AddCommand(suggestAnalyzeCmd)
	rootCmd.AddCommand(suggestRecordCmd)
	rootCmd.AddCommand(suggestCheckCmd)
	rootCmd.AddCommand(suggestApproveCmd)
	rootCmd.AddCommand(suggestQuickDismissCmd)

	// Suppress usage output for all deprecated suggest commands.
	suggestCmds := []*cobra.Command{
		suggestAnalyzeCmd,
		suggestRecordCmd,
		suggestCheckCmd,
		suggestApproveCmd,
		suggestQuickDismissCmd,
	}
	for _, dc := range suggestCmds {
		dc.SilenceUsage = true
		dc.SilenceErrors = true
	}

	_ = fmt.Sprintf("registered %d deprecated suggest commands", len(suggestCmds))
}
