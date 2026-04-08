package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// deprecatedMessage is the standard deprecation warning returned by all
// deprecated commands. Callers that parse the JSON envelope will still
// see ok:true so they do not break.
const deprecatedMessage = "This command is deprecated and will be removed in a future version"

// flagDef describes a flag to register on a deprecated command.
type flagDef struct {
	name     string
	boolType bool // if true, register as Bool; otherwise String
	default_ string
	help     string
}

// newDeprecatedCmd creates a cobra.Command that outputs a deprecation notice
// but returns ok:true so downstream callers do not break.
//
// Args validation is controlled by maxArgs (use -1 for any number of args).
func newDeprecatedCmd(use string, short string, maxArgs int, flags []flagDef) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			outputOK(map[string]interface{}{
				"deprecated": true,
				"command":    use,
				"message":    deprecatedMessage,
			})
			return nil
		},
	}

	switch maxArgs {
	case 0:
		cmd.Args = cobra.NoArgs
	case 1:
		cmd.Args = cobra.MaximumNArgs(1)
	case 2:
		cmd.Args = cobra.MaximumNArgs(2)
	case 3:
		cmd.Args = cobra.MaximumNArgs(3)
	case -1:
		cmd.Args = cobra.ArbitraryArgs
	}

	for _, f := range flags {
		if f.boolType {
			cmd.Flags().Bool(f.name, f.default_ == "true", f.help)
		} else {
			cmd.Flags().String(f.name, f.default_, f.help)
		}
	}

	return cmd
}

// ---------------------------------------------------------------------------
// Semantic commands (all deprecated)
// ---------------------------------------------------------------------------

var semanticInitCmd = newDeprecatedCmd(
	"semantic-init",
	"Initialize semantic store [DEPRECATED]",
	0,
	nil,
)

var semanticIndexCmd = newDeprecatedCmd(
	"semantic-index",
	"Index text for semantic search [DEPRECATED]",
	3, // text, source, optional entry_id
	nil,
)

var semanticSearchCmd = newDeprecatedCmd(
	"semantic-search",
	"Search semantic index [DEPRECATED]",
	-1, // query, optional top_k, threshold, source_filter
	nil,
)

var semanticRebuildCmd = newDeprecatedCmd(
	"semantic-rebuild",
	"Rebuild semantic index [DEPRECATED]",
	0,
	nil,
)

var semanticStatusCmd = newDeprecatedCmd(
	"semantic-status",
	"Show semantic store status [DEPRECATED]",
	0,
	nil,
)

var semanticContextCmd = newDeprecatedCmd(
	"semantic-context",
	"Semantic search context retrieval [DEPRECATED]",
	-1,
	nil,
)

// ---------------------------------------------------------------------------
// Survey commands (all deprecated)
// ---------------------------------------------------------------------------

var surveyClearCmd = newDeprecatedCmd(
	"survey-clear",
	"Clear survey state [DEPRECATED]",
	0,
	[]flagDef{
		{name: "dry-run", boolType: true, default_: "false", help: "Show what would be cleared without deleting"},
	},
)

var surveyVerifyFreshCmd = newDeprecatedCmd(
	"survey-verify-fresh",
	"Check survey freshness [DEPRECATED]",
	1, // timestamp arg
	[]flagDef{
		{name: "force", boolType: true, default_: "false", help: "Force all files to be considered fresh"},
	},
)

// ---------------------------------------------------------------------------
// init: register all deprecated commands
// ---------------------------------------------------------------------------

func init() {
	rootCmd.AddCommand(semanticInitCmd)
	rootCmd.AddCommand(semanticIndexCmd)
	rootCmd.AddCommand(semanticSearchCmd)
	rootCmd.AddCommand(semanticRebuildCmd)
	rootCmd.AddCommand(semanticStatusCmd)
	rootCmd.AddCommand(semanticContextCmd)
	rootCmd.AddCommand(surveyClearCmd)
	rootCmd.AddCommand(surveyVerifyFreshCmd)

	// Suppress usage output for all deprecated commands.
	deprecatedCmds := []*cobra.Command{
		semanticInitCmd,
		semanticIndexCmd,
		semanticSearchCmd,
		semanticRebuildCmd,
		semanticStatusCmd,
		semanticContextCmd,
		surveyClearCmd,
		surveyVerifyFreshCmd,
	}
	for _, dc := range deprecatedCmds {
		dc.SilenceUsage = true
		dc.SilenceErrors = true
	}

	// Log deprecation notice at registration time (non-breaking).
	// This is informational only; the deprecation is communicated via JSON output.
	_ = fmt.Sprintf("registered %d deprecated commands", len(deprecatedCmds))
}
