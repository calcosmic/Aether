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
// re-checking anything of its own. On the small route, once the attempt has
// run, the route authority is asked a second time with the attempt's own
// measured facts (D-03): a small job that proves bigger is moved up to the
// planning route by the program itself, never by asking. The big route
// decides and hands over -- it plans, builds and writes nothing of its own;
// it dispatches no helper and reports the route and reason for the owner
// and for the shared next-action authority to carry onward. Neither route
// ever refuses the owner's job.
func runGoJob(job string, timeout time.Duration) (map[string]interface{}, error) {
	root := skillWorkspaceRoot()
	facts := gatherJobSizeFacts(root, job)
	decision := resolveJobSizeRoute(facts)

	if decision.Route == jobSizeRouteSmall {
		result, err := runQuickJob(job, timeout)
		if err != nil {
			return nil, err
		}
		attempt := smallAttemptFactsFromQuickResult(result)
		facts.Attempt = &attempt
		finalDecision := resolveJobSizeRoute(facts)
		result["route"] = string(finalDecision.Route)
		result["route_reason"] = finalDecision.Reason
		result["job"] = job
		if finalDecision.Escalated {
			result["escalated"] = true
			result["escalation_reason"] = finalDecision.Reason
		}
		return result, nil
	}

	return map[string]interface{}{
		"mode":          "go-big",
		"job":           job,
		"goal":          job,
		"route":         string(decision.Route),
		"route_reason":  decision.Reason,
		"colony_active": facts.ColonyActive,
	}, nil
}

// renderGoVisual renders `/ant-go`'s screen: the banner, then the route
// sentence as the FIRST content line -- the owner reads which route was
// picked and why before anything else -- then, when the small route's own
// attempt was moved up to the planning route (D-03), exactly one escalation
// line naming the measured fact that made that clear, then, whenever an
// attempt actually ran, the existing quick-job lines exactly as
// renderQuickVisual renders them (an escalation never hides or undoes the
// attempt's own changes), then the shared closing card. Every content line
// goes through voiceLine; this file writes no symbol literal of its own,
// and it never asks a question or names a command of its own -- the
// closing card's command comes solely from the shared next-action
// authority.
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

	if escalated, _ := result["escalated"].(bool); escalated {
		escalationReason := strings.TrimSpace(stringValue(result["escalation_reason"]))
		b.WriteString(voiceLine("warning", "This turned out bigger than it looked -- "+escalationReason+
			" -- so it has been moved up to the planning route."))
		b.WriteString("\n\n")
	}

	if stringValue(result["mode"]) == "quick-job" {
		b.WriteString(renderQuickJobBody(result))
	}

	b.WriteString("\n")
	b.WriteString(renderLifecycleClosing(result, "go"))
	return b.String()
}
