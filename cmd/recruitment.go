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
// back. 203-02-PLAN.md proved the minimal end-to-end path; 203-03-PLAN.md
// grows the intent to BIO-01's full field set, adds a by-name validator, and
// records every intent durably before any decision is taken.
var recruitCmd = &cobra.Command{
	Use:   "recruit",
	Short: "Ask the program to admit a helper for the current task",
	Long: "Ask the program to admit a helper for the current task. " +
		"A refusal here is not an error -- it means the program's own limits " +
		"denied the request, and the caller should carry on and finish the work alone.",
	Args: cobra.NoArgs,
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

		capability, _ := cmd.Flags().GetString("capability")
		parentAttemptID, _ := cmd.Flags().GetString("attempt")
		evidence, _ := cmd.Flags().GetStringArray("evidence")
		urgency, _ := cmd.Flags().GetString("urgency")
		declaredPaths, _ := cmd.Flags().GetStringArray("declared-path")
		costSlots, _ := cmd.Flags().GetInt("cost-slots")
		costSeconds, _ := cmd.Flags().GetInt("cost-seconds")
		if costSeconds <= 0 {
			// BIO-01's own cost-seconds default: the same value
			// resolvedRecruitmentTimeout (cmd/recruitment_dispatch.go)
			// yields, so an unset --cost-seconds never disagrees with the
			// timeout the child actually runs under.
			costSeconds = int(resolvedRecruitmentTimeout().Seconds())
		}

		// Resolve the parent's authoritative depth exactly the way
		// spawnCanSpawnCmd already does (cmd/spawn.go): a name that resolves
		// to a recorded spawn-tree entry is authoritative; a coordinator
		// sentinel (spawnParentIsRoot) is authoritative at depth 0 with no
		// spawn-tree entry of its own; anything else is not authoritative,
		// and validateRecruitmentIntent refuses that case rather than
		// trusting a self-declared depth of 0.
		st := agent.NewSpawnTree(store, "spawn-tree.txt")
		depth := 0
		authoritative := false
		if entry := latestSpawnEntryByName(st, parent); entry != nil {
			depth = entry.Depth
			authoritative = true
		} else if spawnParentIsRoot(parent) {
			depth = 0
			authoritative = true
		}

		parentRunID := ""
		if run, ok, err := st.CurrentRun(); err == nil && ok {
			parentRunID = run.ID
		}

		attemptID := fmt.Sprintf("recruit_%d", time.Now().UTC().UnixNano())

		intent := recruitmentIntent{
			SchemaVersion:        recruitmentSchemaVersion,
			ParentName:           parent,
			ParentDepth:          depth,
			DepthIsAuthoritative: authoritative,
			AttemptID:            attemptID,
			Caste:                caste,
			Objective:            objective,
			Reason:               reason,
			Workspace:            workspace,

			IntentID:        attemptID,
			ParentAttemptID: parentAttemptID,
			ParentRunID:     parentRunID,
			Capability:      capability,
			Evidence:        evidence,
			Permission:      codex.PermissionProfileForCaste(caste),
			Urgency:         urgency,
			DeclaredPaths:   declaredPaths,
			CostSlots:       costSlots,
			CostSeconds:     costSeconds,
		}

		// must_haves: every field BIO-01 names is refused by name when
		// wrong, BEFORE the admission decision is even attempted.
		validation := validateRecruitmentIntent(intent)

		// Worker-authored objective/reason/capability text is sanitised
		// before it is stored, because it is replayed verbatim into later
		// worker briefs -- but only once validation has confirmed the
		// content is safe to store as clean text. A refused intent's raw
		// content is exactly the evidence an operator needs to see, so it
		// is stored as-is.
		storedIntent := intent
		if validation.Allowed {
			storedIntent = sanitizedRecruitmentIntentCopy(intent)
		}

		// must_haves: every emitted intent is durably recorded with its
		// identifier BEFORE any admission decision is taken, so a refusal
		// is as recoverable as an admission. An unrecordable ask must never
		// silently become a granted one -- refuse rather than proceed
		// unrecorded (BIO-02's rule that failed logging denies launch,
		// applied one layer earlier).
		createdAt := time.Now().UTC().Format(time.RFC3339)
		if _, err := recordRecruitmentIntent(recruitmentIntentRecord{Intent: storedIntent, CreatedAt: createdAt}); err != nil {
			detail := fmt.Sprintf("could not durably record this recruitment intent (%v)", err)
			outputOK(map[string]interface{}{
				"admitted": false,
				"parent":   intent.ParentName,
				"reason":   recruitmentReasonScope,
				"detail":   detail,
				"message":  fmt.Sprintf("%s -- carry on with the task alone", detail),
			})
			return nil
		}

		// "validate, record, then decide, then update the record with the
		// decision" (203-03-PLAN.md Task 2): validation's own refusal IS the
		// overall decision when it fails; only a validation pass reaches
		// the SAME spawnCanSpawnDecision chokepoint every ordinary spawn
		// already uses -- SYN-203-02, no second admission authority.
		maxDepthFlag, _ := cmd.Flags().GetInt("max-depth")
		depthRaised := false
		decision := validation
		if decision.Allowed {
			admission := spawnCanSpawnDecision(spawnDecisionInput{
				RequesterName:        intent.ParentName,
				RequesterDepth:       intent.ParentDepth,
				DepthIsAuthoritative: intent.DepthIsAuthoritative,
				Caste:                intent.Caste,
				Task:                 intent.Objective,
				Origin:               spawnOriginRecruit,
				Permission:           intent.Permission,
				Workspace:            intent.Workspace,
				CostSlots:            intent.CostSlots,
				IntentID:             intent.IntentID,
				AttemptID:            intent.AttemptID,
			})
			if !admission.Allowed && admission.Reason == "depth" {
				// D-11: the owner's per-run flag may raise the effective
				// depth cap for this one recruitment. Every other denial
				// reason is untouched by this flag -- it raises only the
				// depth ceiling, never budget, permission, path, cost, or
				// duplicate.
				if cap, raised := recruitmentDepthOverride(maxDepthFlag); raised && intent.ParentDepth+1 <= cap {
					admission = spawnDecisionResult{Allowed: true}
					depthRaised = true
				}
			}
			decision = recruitmentDecisionResult{Allowed: admission.Allowed, Reason: admission.Reason, Detail: admission.Detail}
		}

		decidedAt := time.Now().UTC().Format(time.RFC3339)
		// Best-effort: a failure to attach the decision to the already-durable
		// intent record must never turn a real decision into an error the
		// caller has to retry -- the intent itself is already safely
		// recorded either way.
		_, _ = recordRecruitmentDecision(intent.IntentID, decision, decidedAt)

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

		// From here on, storedIntent is guaranteed to equal
		// sanitizedRecruitmentIntentCopy(intent) -- decision.Allowed is only
		// ever true when validation.Allowed was also true.
		dispatchIntent := storedIntent

		childName := deterministicAntName(dispatchIntent.Caste, dispatchIntent.AttemptID)
		childDepth := dispatchIntent.ParentDepth + 1

		task := dispatchIntent.Objective
		if depthRaised {
			// D-11: the raise is written onto the spawn-tree entry it permits,
			// so the append-only ledger carries the fact even though this
			// ledger's fixed pipe format (pkg/agent/spawn_tree.go, outside
			// this plan's file ownership) has no dedicated field for it.
			task = fmt.Sprintf("%s [depth-raise:max-depth=%d]", task, maxDepthFlag)
		}
		if err := st.RecordSpawn(dispatchIntent.ParentName, dispatchIntent.Caste, childName, task, childDepth); err != nil {
			outputError(2, fmt.Sprintf("failed to record recruitment spawn: %v", err), nil)
			return nil
		}

		emitColonyLiveRecruitAdmitted(dispatchIntent, childName)

		dispatchResult, dispatchErr := dispatchRecruitment(dispatchIntent, childName)

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
			RecruitmentID:  dispatchIntent.AttemptID,
			IntentID:       dispatchIntent.IntentID,
			ChildName:      childName,
			ParentName:     dispatchIntent.ParentName,
			TerminalStatus: terminalStatus,
			Summary:        summary,
			Transaction: colony.LifecycleTransactionReference{
				ID:    dispatchIntent.AttemptID,
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
				Caste:          dispatchIntent.Caste,
				Root:           dispatchIntent.Workspace,
				ParentWorkerID: dispatchIntent.ParentName,
			},
			codex.DispatchResult{
				WorkerName: childName,
				Status:     terminalStatus,
			},
		)

		outputOK(map[string]interface{}{
			"admitted":        true,
			"parent":          dispatchIntent.ParentName,
			"child":           childName,
			"caste":           dispatchIntent.Caste,
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
	recruitCmd.Flags().String("capability", "", "The specific ability needed from the requested caste (optional)")
	recruitCmd.Flags().String("attempt", "", "The parent's own recorded attempt ID, if any (optional)")
	recruitCmd.Flags().StringArray("evidence", nil, "An identifier naming recorded lifecycle evidence supporting this request (repeatable)")
	recruitCmd.Flags().String("urgency", recruitmentUrgencyRoutine, "How urgent this request is: routine or blocking")
	recruitCmd.Flags().StringArray("declared-path", nil, "A path this recruitment declares it will touch (repeatable)")
	recruitCmd.Flags().Int("cost-slots", 1, "Helper slots this recruitment counts against the whole-run budget")
	recruitCmd.Flags().Int("cost-seconds", 0, "Wall-clock seconds this recruitment is allowed (default: the dispatch timeout)")
	recruitCmd.Flags().Int("max-depth", 0, "Raise the effective depth cap for this one recruitment above the default of 2 (D-11); 0 or the default value means no raise")

	rootCmd.AddCommand(recruitCmd)
}
