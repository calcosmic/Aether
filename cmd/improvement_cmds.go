package cmd

// LEARN-06/LEARN-07/LEARN-08 (204-12-PLAN.md, Task 3, D-02): the one
// hand-run improvement surface -- `aether improve` -- plus registration of
// the two commands WINDOWS.md entry 39 recorded as constructed but never
// reachable: `aether shadow-declare` and `aether shadow-compare`. Every
// mutation these three commands can ever perform already lives in
// cmd/shadow_cmds.go (declareShadowCandidate, runShadowCompare); this file
// adds no new writer.
//
// `aether improve` with no flags is an INSPECTION: it reads the durable
// episode ledger and the committed eval-gate sentinel list and renders the
// two-figure report (cmd/improvement_report.go) -- it mutates nothing on
// disk. `--declare` and `--compare` are the two mutating forms, and each
// routes to the exact same function the standalone shadow-declare/
// shadow-compare commands already call.
import (
	"fmt"

	"github.com/spf13/cobra"
)

// improveCmd is the hand-run improvement surface: inspect by default,
// declare or compare with a flag.
var improveCmd = &cobra.Command{
	Use:   "improve",
	Short: "Look at how the colony's own suggestions are doing, or try one by hand",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		declare, _ := cmd.Flags().GetBool("declare")
		compareID, _ := cmd.Flags().GetString("compare")

		switch {
		case declare && compareID != "":
			outputErrorMessage("--declare and --compare cannot both be used in the same call -- declare a candidate first, then compare it separately")
			return nil
		case declare:
			return runImproveDeclare(cmd)
		case compareID != "":
			return runImproveCompare(compareID)
		default:
			return runImproveReport()
		}
	},
}

// runImproveReport is the default, read-only form: builds and renders the
// two-figure improvement report (cmd/improvement_report.go) over the whole
// durable episode ledger. It writes nothing to the store, the working
// tree, or any colony file -- readEpisodeLedger and loadEvalGateSentinels
// are both pure reads, and buildImprovementReport/renderImprovementReport
// are both pure functions.
func runImproveReport() error {
	records, err := readEpisodeLedger()
	if err != nil {
		outputErrorMessage(err.Error())
		return nil
	}
	sentinelFile, err := loadEvalGateSentinels()
	if err != nil {
		outputErrorMessage(err.Error())
		return nil
	}
	report := buildImprovementReport(records, sentinelFile.Sentinels, reportWindow{})
	if shouldRenderVisualOutput(stdout) {
		writeVisualOutput(stdout, renderImprovementReport(report))
	} else {
		outputOK(report)
	}
	return nil
}

// runImproveDeclare routes to the exact same declareShadowCandidate
// shadowDeclareCmd already calls.
func runImproveDeclare(cmd *cobra.Command) error {
	id, _ := cmd.Flags().GetString("id")
	scope, _ := cmd.Flags().GetString("scope")
	expectedBenefit, _ := cmd.Flags().GetString("expected-benefit")
	harms, _ := cmd.Flags().GetString("harms")
	expires, _ := cmd.Flags().GetString("expires")
	rollbackPlan, _ := cmd.Flags().GetString("rollback-plan")

	record, isNew, err := declareShadowCandidate(id, scope, expectedBenefit, harms, expires, rollbackPlan)
	if err != nil {
		outputErrorMessage(err.Error())
		return nil
	}
	outputOK(map[string]interface{}{
		"declared":       true,
		"is_new":         isNew,
		"id":             record.ID,
		"content_digest": record.ContentDigest,
		"message":        voiceLine("done", fmt.Sprintf("Proposal %q declared -- expires %s", record.ID, record.Expiry)),
	})
	return nil
}

// runImproveCompare routes to the exact same runShadowCompare
// shadowCompareCmd already calls.
func runImproveCompare(candidateID string) error {
	outcome, err := runShadowCompare(candidateID)
	if err != nil {
		outputErrorMessage(err.Error())
		return nil
	}
	outputOK(map[string]interface{}{
		"compared":    true,
		"candidate":   outcome.Comparison.CandidateID,
		"verdict":     string(outcome.Comparison.Verdict),
		"recommended": outcome.Comparison.Recommended(),
		"credited":    outcome.Credited,
		"rendered":    outcome.Rendered,
	})
	return nil
}

// init registers improveCmd, and the two commands shadow_cmds.go's own
// init() built but deliberately left unregistered -- WINDOWS.md entry 39.
// shadowDeclareCmd and shadowCompareCmd's own flag declarations stay
// exactly where they are (cmd/shadow_cmds.go's init()); this is the one
// line that closes the reachability gap for each.
func init() {
	improveCmd.Flags().Bool("declare", false, "Declare a candidate to try beside the current behaviour")
	improveCmd.Flags().String("id", "", "Candidate identifier (with --declare)")
	improveCmd.Flags().String("scope", "", "What the candidate would change (with --declare)")
	improveCmd.Flags().String("expected-benefit", "", "Why the candidate might help (with --declare)")
	improveCmd.Flags().String("harms", "", "What could go wrong if the candidate is adopted (with --declare)")
	improveCmd.Flags().String("expires", "", "RFC3339 timestamp the candidate's declaration expires at (with --declare)")
	improveCmd.Flags().String("rollback-plan", "", "How to undo the candidate if it is adopted and later needs reverting (with --declare)")
	improveCmd.Flags().String("compare", "", "Identifier of a declared candidate to run beside the current behaviour")

	rootCmd.AddCommand(improveCmd)
	rootCmd.AddCommand(shadowDeclareCmd)
	rootCmd.AddCommand(shadowCompareCmd)
}
