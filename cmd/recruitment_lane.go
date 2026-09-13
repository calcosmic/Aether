package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// This file is the Go in-repo build lane's own consumer of a worker's
// spawn claims (203-09 Task 2, SYN-203-02 ruling (b)): "this phase's
// governed recruitment... is delivered for BOTH lanes." Before this file
// existed, pkg/codex/worker.go parsed a worker's `spawns` field into
// codex.WorkerResult.Spawns and nothing ever read it -- a worker asking
// for backup on the native/interactive lane got silence, never a refusal
// and never a helper. This carries each claim to the SAME
// spawnCanSpawnDecision chokepoint (origin recruit) every other lane
// already uses, and dispatches an admitted claim through the SAME
// dispatchRecruitment path `aether recruit` already uses -- no second
// admission authority, no second dispatch mechanism.
//
// File-ownership note: this plan's declared files_modified do not name
// pkg/codex/worker.go, cmd/codex_build.go, or this new file. Task 2's own
// action text requires reading and wiring into the Go in-repo dispatch
// path, which those files own; this gap is documented as a deviation in
// 203-09-SUMMARY.md rather than silently worked around. This file is new
// and self-contained; the one unavoidable call site addition lives in
// cmd/codex_build.go's existing per-result loop, kept to the minimum
// needed to reach a worker's own WorkerResult.Spawns.

// recruitmentLaneUnsupportedDetail is the plain-language refusal an owner
// sees when a worker's spawn claim arrives on a lane this phase does not
// yet govern for real dispatch: worktree-isolated build workers. Recruiting
// a real child process from inside an isolated worktree checkout raises
// workspace-containment questions this phase's ruling (d) does not resolve
// (unlike the root/host-mediated subprocess fallback ruled on for native
// nesting); until a future plan decides that question, this lane fails
// closed and says so in one plain sentence, rather than silently
// dispatching an ungoverned child or quietly discarding the request.
const recruitmentLaneUnsupportedDetail = "recruitment requests from a worktree-isolated worker are not yet governed on this lane -- the request was refused before launch, no helper was started"

// recruitmentLaneClaim is a normalized in-repo lane spawn claim.
type recruitmentLaneClaim struct {
	Caste      string
	Task       string
	WellFormed bool
	Raw        string
}

// parseInRepoSpawnClaim decodes one entry of a worker's Spawns list. A
// worker that emits a structured spawn claim as a JSON object (e.g.
// `{"caste":"scout","task":"..."}`) survives pkg/codex/worker.go's
// stringList decoding as raw JSON text -- its own documented mixed-array
// fallback (pkg/codex/string_list.go, "a mixed array still carries usable
// content") -- rather than a parsed struct; this function re-parses that
// text. A bare string (a claim naming no caste or objective) carries too
// little information to build a valid recruitmentIntent and is reported
// not well-formed, never silently guessed at.
func parseInRepoSpawnClaim(raw string) recruitmentLaneClaim {
	trimmed := strings.TrimSpace(raw)
	if strings.HasPrefix(trimmed, "{") {
		var obj struct {
			Caste string `json:"caste"`
			Task  string `json:"task"`
		}
		if err := json.Unmarshal([]byte(trimmed), &obj); err == nil {
			caste := strings.TrimSpace(obj.Caste)
			task := strings.TrimSpace(obj.Task)
			if caste != "" && task != "" {
				return recruitmentLaneClaim{Caste: caste, Task: task, WellFormed: true, Raw: raw}
			}
		}
	}
	return recruitmentLaneClaim{Raw: raw}
}

// routeInRepoSpawnClaims is the single entry point cmd/codex_build.go calls
// once per dispatch result carrying spawn claims. A worker with no claims
// reaches no code in this file at all (the caller's own len(...)>0 guard),
// so an ordinary build that never asks for help pays nothing new.
func routeInRepoSpawnClaims(root string, parallelMode colony.ParallelMode, dispatch codex.WorkerDispatch, result codex.WorkerResult) {
	if len(result.Spawns) == 0 {
		return
	}

	if parallelMode == colony.ModeWorktree {
		for range result.Spawns {
			emitVisualProgress(fmt.Sprintf("%s asked for help, but %s", dispatch.WorkerName, recruitmentLaneUnsupportedDetail))
		}
		return
	}

	if store == nil {
		return
	}

	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	parentDepth := 0
	authoritative := false
	if entry := latestSpawnEntryByName(st, dispatch.WorkerName); entry != nil {
		parentDepth = entry.Depth
		authoritative = true
	} else if spawnParentIsRoot(dispatch.WorkerName) {
		authoritative = true
	}

	for _, raw := range result.Spawns {
		claim := parseInRepoSpawnClaim(raw)
		if !claim.WellFormed {
			emitVisualProgress(fmt.Sprintf(
				"%s's request for help could not be understood (missing a caste or an objective: %q) -- refused, no helper started",
				dispatch.WorkerName, claim.Raw,
			))
			continue
		}
		dispatchOneInRepoRecruitment(root, dispatch, parentDepth, authoritative, claim)
	}
}

// dispatchOneInRepoRecruitment carries one well-formed claim through the
// identical validate -> record -> admit -> record -> dispatch -> bind
// sequence `aether recruit` (cmd/recruitment.go) already uses, calling the
// SAME package-private helpers directly (this file is part of package cmd)
// rather than re-implementing any of them.
func dispatchOneInRepoRecruitment(root string, dispatch codex.WorkerDispatch, parentDepth int, authoritative bool, claim recruitmentLaneClaim) {
	workspace := strings.TrimSpace(dispatch.Root)
	if workspace == "" {
		workspace = root
	}

	attemptID := fmt.Sprintf("inrepo_%d", time.Now().UTC().UnixNano())
	intent := recruitmentIntent{
		SchemaVersion:        recruitmentSchemaVersion,
		ParentName:           dispatch.WorkerName,
		ParentDepth:          parentDepth,
		DepthIsAuthoritative: authoritative,
		AttemptID:            attemptID,
		Caste:                claim.Caste,
		Objective:            claim.Task,
		Reason:               "worker-requested backup on the in-repo build lane",
		Workspace:            workspace,
		IntentID:             attemptID,
		Permission:           codex.PermissionProfileForCaste(claim.Caste),
		Urgency:              recruitmentUrgencyRoutine,
		CostSlots:            1,
		CostSeconds:          int(resolvedRecruitmentTimeout().Seconds()),
	}

	validation := validateRecruitmentIntent(intent)
	storedIntent := intent
	if validation.Allowed {
		storedIntent = sanitizedRecruitmentIntentCopy(intent)
	}

	createdAt := time.Now().UTC().Format(time.RFC3339)
	if _, err := recordRecruitmentIntent(recruitmentIntentRecord{Intent: storedIntent, CreatedAt: createdAt}); err != nil {
		emitVisualProgress(fmt.Sprintf(
			"%s's request for help could not be durably recorded (%v) -- refused, no helper started",
			dispatch.WorkerName, err,
		))
		return
	}

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
		decision = recruitmentDecisionResult{Allowed: admission.Allowed, Reason: admission.Reason, Detail: admission.Detail}
	}

	decidedAt := time.Now().UTC().Format(time.RFC3339)
	_, _ = recordRecruitmentDecision(intent.IntentID, decision, decidedAt)

	if !decision.Allowed {
		emitColonyLiveRecruitRefused(intent, decision.Reason, decision.Detail)
		return
	}

	dispatchIntent := storedIntent
	childName := deterministicAntName(dispatchIntent.Caste, dispatchIntent.AttemptID)
	childDepth := dispatchIntent.ParentDepth + 1

	manifestRecord := recruitmentManifestRecord{
		SchemaVersion: recruitmentManifestSchemaVersion,
		IntentID:      dispatchIntent.IntentID,
		ChildName:     childName,
		ParentName:    dispatchIntent.ParentName,
		Depth:         childDepth,
		Workspace:     dispatchIntent.Workspace,
		AdmittedAt:    time.Now().UTC().Format(time.RFC3339),
		State:         recruitmentManifestStateAdmitted,
	}
	if _, err := amendRecruitmentManifest(manifestRecord); err != nil {
		emitVisualProgress(fmt.Sprintf(
			"%s's admitted request for help could not be durably recorded (%v) -- refused, no helper started",
			dispatch.WorkerName, err,
		))
		return
	}

	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	if err := st.RecordSpawn(dispatchIntent.ParentName, dispatchIntent.Caste, childName, dispatchIntent.Objective, childDepth); err != nil {
		return
	}

	emitColonyLiveRecruitAdmitted(dispatchIntent, childName)

	_, _ = amendRecruitmentManifest(recruitmentManifestRecord{
		SchemaVersion: recruitmentManifestSchemaVersion,
		IntentID:      dispatchIntent.IntentID,
		State:         recruitmentManifestStateDispatched,
	})

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
	_ = st.UpdateStatus(childName, terminalStatus, summary)

	res := recruitmentResult{
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
	_, _ = bindRecruitmentResult(res)

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
}
