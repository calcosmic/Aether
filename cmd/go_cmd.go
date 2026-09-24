package cmd

// Phase 209 plan 01, Task 1 -- `/ant-go`, the one entry command (D-01):
// "<what you want>" resolves to a real, computed route (cmd/go_route.go),
// says which route it picked and why in plain English, and -- on the small
// route -- does the job through the existing one-helper quick path
// (runQuickJob, cmd/command_truth.go) and the project's own checks. The big
// route is left to a later plan: this command names the route and stops.

import (
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/spf13/cobra"
)

var goCmd = &cobra.Command{
	Use:   "go [what you want done]",
	Short: "Do what you asked for — the program picks a quick job or a planned one and says which",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		job := strings.TrimSpace(strings.Join(args, " "))
		if job == "" {
			outputError(1, `usage: aether go "what you want done"`, nil)
			return nil
		}
		timeout, _ := cmd.Flags().GetDuration("timeout")
		if timeout <= 0 {
			timeout = codex.DefaultWorkerTimeout
		}

		result, err := runGoJob(job, timeout)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		closeLifecycleCommand(result, "go", "", "")
		outputWorkflow(result, renderGoVisual(result))
		return nil
	},
}

func init() {
	goCmd.Flags().Duration("timeout", codex.DefaultWorkerTimeout, "Worker timeout")
	rootCmd.AddCommand(goCmd)
}

// runGoJob is `/ant-go`'s command body: gather the read-only facts, ask the
// one route authority once, and on the small route reuse the existing
// single-helper quick-job implementation rather than re-dispatching or
// re-checking anything of its own. The big route is filled in by a later
// plan; for now it reports the route and reason and stops -- it never
// refuses the owner's job.
func runGoJob(job string, timeout time.Duration) (map[string]interface{}, error) {
	root := skillWorkspaceRoot()
	facts := gatherJobSizeFacts(root, job)
	decision := resolveJobSizeRoute(facts)

	if decision.Route == jobSizeRouteSmall {
		result, err := runQuickJob(job, timeout)
		if err != nil {
			return nil, err
		}
		result["route"] = string(decision.Route)
		result["route_reason"] = decision.Reason
		result["job"] = job
		return result, nil
	}

	return map[string]interface{}{
		"mode":         "go-big",
		"job":          job,
		"route":        string(decision.Route),
		"route_reason": decision.Reason,
	}, nil
}

// renderGoVisual renders `/ant-go`'s screen: the banner, then the route
// sentence as the FIRST content line -- the owner reads which route was
// picked and why before anything else -- then, on the small route, the
// existing quick-job lines exactly as renderQuickVisual renders them, then
// the shared closing card. Every content line goes through voiceLine; this
// file writes no symbol literal of its own.
func renderGoVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner("🧭", "Go"))
	b.WriteString(visualDividerStr())

	if reason := strings.TrimSpace(stringValue(result["route_reason"])); reason != "" {
		route := stringValue(result["route"])
		routeWord := "a quick job for one helper"
		if route == string(jobSizeRouteBig) {
			routeWord = "the planning route"
		}
		b.WriteString(voiceLine("decision", "This is "+routeWord+": "+reason))
		b.WriteString("\n\n")
	}

	if stringValue(result["route"]) == string(jobSizeRouteSmall) && stringValue(result["mode"]) == "quick-job" {
		b.WriteString(renderQuickJobBody(result))
	} else if stringValue(result["mode"]) == "go-big" {
		b.WriteString(voiceLine("warning", "The planning route is not wired up yet in this build -- nothing was done. Run `aether plan` directly for now."))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(renderLifecycleClosing(result, "go"))
	return b.String()
}
