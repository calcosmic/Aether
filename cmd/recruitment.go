package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

// recruitCmd is the public work path BIO-01 requires: a worker running with
// only Bash can carry a recruitment intent from a real public command
// through Go's existing admission chokepoint to a real dispatched child and
// back. This tracer proves the whole path on one caste, one child, one
// happy path plus its refusal (203-02-PLAN.md Task 1).
var recruitCmd = &cobra.Command{
	Use:   "recruit",
	Short: "Ask the program to admit a helper for the current task",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		parent := mustGetString(cmd, "parent")
		if parent == "" {
			return nil
		}
		caste := mustGetString(cmd, "caste")
		if caste == "" {
			return nil
		}
		objective := mustGetString(cmd, "objective")
		if objective == "" {
			return nil
		}
		reason := mustGetString(cmd, "reason")
		if reason == "" {
			return nil
		}
		workspace, _ := cmd.Flags().GetString("workspace")
		if strings.TrimSpace(workspace) == "" {
			// Default to the colony root -- D-19 requires an unreadable or
			// ambiguous workspace to deny, never guess; the colony root is
			// the one workspace this command always knows.
			workspace = repoRootFromStore(store)
		}

		// Resolve the parent's authoritative depth exactly the way
		// spawnCanSpawnCmd already does (cmd/spawn.go): a name that resolves
		// to a recorded spawn-tree entry is authoritative; an unresolved
		// name falls back to depth 0, not authoritative.
		st := agent.NewSpawnTree(store, "spawn-tree.txt")
		depth := 0
		authoritative := false
		if entry := latestSpawnEntryByName(st, parent); entry != nil {
			depth = entry.Depth
			authoritative = true
		}

		intent := recruitmentIntent{
			SchemaVersion:        recruitmentSchemaVersion,
			ParentName:           parent,
			ParentDepth:          depth,
			DepthIsAuthoritative: authoritative,
			AttemptID:            fmt.Sprintf("recruit_%d", time.Now().UTC().UnixNano()),
			Caste:                caste,
			Objective:            objective,
			Reason:               reason,
			Workspace:            workspace,
		}

		// SYN-203-02 / must_haves: spawnCanSpawnDecision is the ONLY
		// admission decision in this command -- this is the single call
		// site in this file. No recruitment-specific depth, budget or
		// cycle limiter is introduced beside it.
		decision := spawnCanSpawnDecision(spawnDecisionInput{
			RequesterName:        intent.ParentName,
			RequesterDepth:       intent.ParentDepth,
			DepthIsAuthoritative: intent.DepthIsAuthoritative,
			Caste:                intent.Caste,
			Task:                 intent.Objective,
		})

		if !decision.Allowed {
			emitColonyLiveRecruitRefused(intent, decision.Reason, decision.Detail)
			detail := strings.TrimSpace(decision.Detail)
			if detail == "" {
				detail = "recruitment refused"
			}
			// D-03/must_haves: a refused recruitment must exit 0 and never
			// stall or block the caller. outputError would mark a non-zero
			// process exit via markRenderedCommandError (cmd/root.go) --
			// that is correct for a command like spawn-log, whose deny truly
			// should propagate as a failure, but wrong here: a refusal is an
			// ordinary, expected outcome the caller reads and carries on
			// from, not a command failure. outputOK keeps the exit code 0
			// while still surfacing the reason class, detail, and the
			// explicit instruction to work alone.
			outputOK(map[string]interface{}{
				"admitted": false,
				"parent":   intent.ParentName,
				"reason":   decision.Reason,
				"detail":   detail,
				"message":  fmt.Sprintf("%s -- carry on with the task alone", detail),
			})
			return nil
		}

		childName := deterministicAntName(intent.Caste, intent.AttemptID)
		childDepth := intent.ParentDepth + 1
		if err := st.RecordSpawn(intent.ParentName, intent.Caste, childName, intent.Objective, childDepth); err != nil {
			outputError(2, fmt.Sprintf("failed to record recruitment spawn: %v", err), nil)
			return nil
		}

		emitColonyLiveRecruitAdmitted(intent, childName)

		dispatchResult, dispatchErr := dispatchRecruitment(intent, childName)

		terminalStatus := "completed"
		summary := ""
		if dispatchErr != nil {
			terminalStatus = "failed"
			summary = dispatchErr.Error()
		} else if dispatchResult != nil {
			summary = dispatchResult.Summary
			if strings.TrimSpace(dispatchResult.TerminalStatus) != "" {
				terminalStatus = dispatchResult.TerminalStatus
			}
		}
		// A completion-recording failure must never turn a real result into
		// an error the caller has to retry -- mirrors spawnCompleteTolerant's
		// own "the worker finished either way" discipline.
		_ = st.UpdateStatus(childName, terminalStatus, summary)

		result := recruitmentResult{
			SchemaVersion:  recruitmentSchemaVersion,
			RecruitmentID:  intent.AttemptID,
			IntentID:       intent.AttemptID,
			ChildName:      childName,
			ParentName:     intent.ParentName,
			TerminalStatus: terminalStatus,
			Summary:        summary,
			Transaction: colony.LifecycleTransactionReference{
				ID:    intent.AttemptID,
				Stage: colony.TransactionStageCommitted,
			},
		}
		bound, bindErr := bindRecruitmentResult(result)
		if bindErr != nil {
			outputError(2, fmt.Sprintf("failed to bind recruitment result: %v", bindErr), nil)
			return nil
		}

		// Record the child's terminal result into the existing worker
		// handoff store so the next worker brief reads the child's summary
		// through the path that already exists -- no second handback store.
		_ = persistDispatchWorkerHandoff(
			codex.WorkerDispatch{
				WorkerName:     childName,
				Caste:          intent.Caste,
				Root:           intent.Workspace,
				ParentWorkerID: intent.ParentName,
			},
			codex.DispatchResult{
				WorkerName: childName,
				Status:     terminalStatus,
			},
		)

		outputOK(map[string]interface{}{
			"admitted":        true,
			"parent":          intent.ParentName,
			"child":           childName,
			"caste":           intent.Caste,
			"recruitment_id":  bound.RecruitmentID,
			"terminal_status": bound.TerminalStatus,
		})
		return nil
	},
}

func init() {
	recruitCmd.Flags().String("parent", "", "Parent worker's recorded name (required)")
	recruitCmd.Flags().String("caste", "", "Requested helper caste (required)")
	recruitCmd.Flags().String("objective", "", "Bounded objective for the helper (required)")
	recruitCmd.Flags().String("reason", "", "Why help is needed (required)")
	recruitCmd.Flags().String("workspace", "", "Workspace lease for the child (default: colony root)")

	rootCmd.AddCommand(recruitCmd)
}
