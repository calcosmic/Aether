package cmd

import (
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

const legacyRecoverMigrationSchemaVersion = "legacy-recover-migration/v1"

type legacyRecoverMigrationResult struct {
	SchemaVersion string                      `json:"schema_version"`
	Command       string                      `json:"command"`
	OutcomeKind   colony.OutcomeKind          `json:"outcome_kind"`
	Explanation   string                      `json:"explanation"`
	StateEffect   colony.LifecycleStateEffect `json:"state_effect"`
	NextAction    string                      `json:"next_action"`
}

// recoverCmd remains parseable only so old automation receives a safe,
// deterministic migration route. Recovery is now owned by resume; detailed
// evidence inspection lives under maintenance.
var recoverCmd = &cobra.Command{
	Use:          "recover",
	Short:        "Legacy recovery migration route",
	Hidden:       true,
	Args:         cobra.NoArgs,
	Annotations:  map[string]string{"aether.io/read-only": "true", "aether.io/store-free": "true"},
	SilenceUsage: true,
	RunE:         runRecover,
}

func init() {
	rootCmd.AddCommand(recoverCmd)
	recoverCmd.Flags().Bool("apply", false, "legacy compatibility flag; no repair is performed")
	recoverCmd.Flags().Bool("force", false, "legacy compatibility flag; no repair is performed")
	recoverCmd.Flags().Bool("json", false, "output structured JSON")
	_ = recoverCmd.Flags().MarkHidden("apply")
	_ = recoverCmd.Flags().MarkHidden("force")
	_ = recoverCmd.Flags().MarkHidden("json")
}

func runRecover(_ *cobra.Command, _ []string) error {
	result := legacyRecoverMigrationResult{
		SchemaVersion: legacyRecoverMigrationSchemaVersion,
		Command:       "recover",
		OutcomeKind:   colony.OutcomeKindNoChange,
		Explanation:   "The standalone recover command is retired. Recovery now resumes the recorded lifecycle without creating a second repair owner.",
		StateEffect:   colony.LifecycleStateEffectNone,
		NextAction:    "aether resume",
	}
	outputWorkflow(result, renderLegacyRecoverMigration(result))
	return nil
}

func renderLegacyRecoverMigration(result legacyRecoverMigrationResult) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("recover"), "Recovery Moved"))
	b.WriteString(visualDividerStr())
	b.WriteString(result.Explanation)
	b.WriteString("\nState effect: none; no repair was attempted.\n")
	b.WriteString(renderNextUp(
		"Run `aether resume` to continue or recover the recorded lifecycle.",
		"Run `aether maintenance recovery-inspect` when you need read-only diagnostic evidence.",
	))
	return b.String()
}
