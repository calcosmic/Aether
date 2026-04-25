package cmd

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// recoverCmd is the cobra command for rescuing a stuck colony.
var recoverCmd = &cobra.Command{
	Use:   "recover",
	Short: "Rescue a stuck colony",
	Long: `Scan the colony for stuck-state conditions and diagnose why it cannot make progress.
Read-only by default; use --apply to attempt automatic fixes.`,
	Args: cobra.NoArgs,
	RunE: runRecover,
}

func init() {
	rootCmd.AddCommand(recoverCmd)
	recoverCmd.Flags().Bool("apply", false, "apply fixes for detected issues")
	recoverCmd.Flags().Bool("force", false, "allow destructive repairs")
	recoverCmd.Flags().Bool("json", false, "output structured JSON")
}

func runRecover(cmd *cobra.Command, args []string) error {
	state, err := loadActiveColonyState()
	if err != nil {
		if shouldRenderVisualOutput(stdout) && strings.Contains(colonyStateLoadMessage(err), "No colony initialized") {
			fmt.Fprint(stdout, renderNoColonyRecoverVisual())
			return nil
		}
		fmt.Fprintln(stdout, colonyStateLoadMessage(err))
		return nil
	}

	dataDir := filepath.Join(resolveAetherRoot(), ".aether", "data")

	apply, _ := cmd.Flags().GetBool("apply")
	force, _ := cmd.Flags().GetBool("force")
	jsonOut, _ := cmd.Flags().GetBool("json")

	scanStart := time.Now()
	issues, scanErr := performStuckStateScan(dataDir)
	scanDuration := time.Since(scanStart)
	if scanErr != nil {
		fmt.Fprintf(stdout, "Scan failed: %v\n", scanErr)
		return nil
	}

	var repairResult *RepairResult
	if apply && len(issues) > 0 {
		repairResult, err = performRecoverRepairs(issues, dataDir, force, jsonOut)
		if err != nil {
			fmt.Fprintf(stdout, "Repair failed: %v\n", err)
			if jsonOut {
				fmt.Fprint(stdout, renderRecoverJSON(issues, state, scanDuration, nil))
			} else {
				output := renderRecoverDiagnosis(issues, state, nil)
				fmt.Fprint(stdout, output)
			}
			if recoverExitCode(issues) != 0 {
				cmd.SilenceUsage = true
				return fmt.Errorf("issues detected")
			}
			return nil
		}

		// Re-scan to get post-repair state (matches medic pattern).
		postIssues, postErr := performStuckStateScan(dataDir)
		if postErr != nil {
			fmt.Fprintf(stdout, "Post-repair scan failed: %v\n", postErr)
		} else {
			issues = postIssues
		}
	}

	if jsonOut {
		fmt.Fprint(stdout, renderRecoverJSON(issues, state, scanDuration, repairResult))
	} else {
		output := renderRecoverDiagnosis(issues, state, repairResult)
		fmt.Fprint(stdout, output)
	}

	if recoverExitCode(issues) != 0 {
		cmd.SilenceUsage = true
		return fmt.Errorf("issues detected")
	}
	return nil
}

// renderNoColonyRecoverVisual renders the visual when no colony is initialized.
func renderNoColonyRecoverVisual() string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("recover"), "Colony Recovery"))
	b.WriteString(visualDivider)
	b.WriteString("No colony initialized in this repo.\n")
	b.WriteString(renderNextUp(
		`Run `+"`aether init \"goal\"`"+` to start a colony.`,
		`Run `+"`aether lay-eggs`"+` first if this repo has not been set up for Aether yet.`,
	))
	return b.String()
}

