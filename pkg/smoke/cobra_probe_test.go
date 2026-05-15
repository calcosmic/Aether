package smoke

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestProbeCobraFlags(t *testing.T) {
	// Build a simple command tree for testing
	root := &cobra.Command{Use: "test"}
	root.AddCommand(&cobra.Command{
		Use:   "suggest-approve",
		Short: "Test command",
		RunE:  func(cmd *cobra.Command, args []string) error { return nil },
	})
	root.AddCommand(&cobra.Command{
		Use:                "plan",
		Short:              "Host command",
		DisableFlagParsing: true,
		RunE:               func(cmd *cobra.Command, args []string) error { return nil },
	})

	// Add flags to suggest-approve
	subCmd, _, _ := root.Find([]string{"suggest-approve"})
	if subCmd != nil {
		subCmd.Flags().Bool("dry-run", false, "Dry run")
		subCmd.Flags().String("output", "", "Output file")
	}

	result := ProbeCobraFlags(root)

	// Should contain suggest-approve
	if _, ok := result["suggest-approve"]; !ok {
		t.Error("expected suggest-approve in result")
	}

	// suggest-approve should have --dry-run and --output
	saFlags := result["suggest-approve"]
	if len(saFlags) != 2 {
		t.Errorf("expected 2 flags for suggest-approve, got %d: %v", len(saFlags), saFlags)
	}

	// plan should have empty flags (host command)
	planFlags := result["plan"]
	if len(planFlags) != 0 {
		t.Errorf("expected empty flags for host command plan, got %v", planFlags)
	}

	// help should not appear in any flag list
	for cmdName, flags := range result {
		for _, f := range flags {
			if f == "help" {
				t.Errorf("help flag should be excluded, found in %q", cmdName)
			}
		}
	}
}
