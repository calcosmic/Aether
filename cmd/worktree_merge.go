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

// worktreeMergeCmd is deprecated. Use git merge directly instead.
var worktreeMergeCmd = newDeprecatedCmd(
	"worktree-merge",
	"Merge a worktree branch back to target with safety checks [DEPRECATED]",
	0,
	[]flagDef{
		{name: "branch", boolType: false, default_: "", help: "Branch name (required)"},
		{name: "target", boolType: false, default_: "", help: "Target branch (default: current)"},
	},
)

func init() {
	rootCmd.AddCommand(worktreeMergeCmd)

	// Suppress usage output for the deprecated command.
	worktreeMergeCmd.SilenceUsage = true
	worktreeMergeCmd.SilenceErrors = true

	_ = fmt.Sprintf("registered deprecated worktree-merge command")
}
