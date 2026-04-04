package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print aether version",
	RunE: func(cmd *cobra.Command, args []string) error {
		outputOK(Version)
		return nil
	},
}
