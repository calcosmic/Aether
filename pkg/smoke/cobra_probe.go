package smoke

import (
	"sort"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// ProbeCobraFlags walks all subcommands of root and collects each command's
// locally registered flag names (long form only). It returns a map of
// command name → sorted list of flag names.
//
// The "help" flag is excluded since Cobra auto-adds it.
// Host subcommands (those with DisableFlagParsing: true) are recorded with
// an empty flag list since no flags are registered in Go.
func ProbeCobraFlags(root *cobra.Command) map[string][]string {
	result := make(map[string][]string)
	walkAndProbe(root, result)
	return result
}

func walkAndProbe(cmd *cobra.Command, result map[string][]string) {
	for _, child := range cmd.Commands() {
		name := child.Name()

		// Host commands have DisableFlagParsing: true, meaning no flags
		// are registered in Go. Record empty list.
		if child.DisableFlagParsing {
			result[name] = []string{}
			// Still recurse into subcommands (e.g., host -> plan)
			walkAndProbe(child, result)
			continue
		}

		var flags []string
		child.LocalFlags().VisitAll(func(f *pflag.Flag) {
			if f.Name != "help" {
				flags = append(flags, f.Name)
			}
		})
		sort.Strings(flags)
		result[name] = flags

		// Recurse into subcommands
		walkAndProbe(child, result)
	}
}
