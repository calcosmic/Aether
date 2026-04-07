package cmd

import (
	"fmt"
)

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
