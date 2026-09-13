package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/spf13/cobra"
)

// spawnRootParentNames is the coordinator sentinel set under D-05: a spawn
// naming one of these as --parent is a child of the depth-0 coordinator (the
// coordinator itself is never recorded in the spawn tree, so its absence
// from spawn-tree.txt is expected and is the one legitimate not-found case)
// and is therefore recorded at depth 1.
var spawnRootParentNames = []string{"Queen", "Prime-1", "Swarm"}

// spawnParentIsRoot reports whether parent case-insensitively matches one of
// spawnRootParentNames.
func spawnParentIsRoot(parent string) bool {
	for _, root := range spawnRootParentNames {
		if strings.EqualFold(parent, root) {
			return true
		}
	}
	return false
}

// deriveSpawnDepth is the D-05/SPAWN-02 authority on recorded depth: the
// caller's claimed --depth is never trusted. It returns the depth to record
// and, on the D-19 fail-closed branch, a non-empty deny reason (in which
// case the returned depth is meaningless and must not be recorded).
//
// Rules, in order:
//  1. parent is a coordinator sentinel (spawnParentIsRoot) -> depth 1.
//  2. parent is a name already recorded in the spawn tree -> that entry's
//     own depth + 1.
//  3. otherwise -> deny. An unresolvable parent must never default to 0;
//     that would let a caller invent a parent to gain depth for free.
func deriveSpawnDepth(st *agent.SpawnTree, parent string) (int, string) {
	if spawnParentIsRoot(parent) {
		return 1, ""
	}
	if entry := latestSpawnEntryByName(st, parent); entry != nil {
		return entry.Depth + 1, ""
	}
	return 0, fmt.Sprintf("unknown parent %q: not a recorded spawn and not a coordinator sentinel (%s)", parent, strings.Join(spawnRootParentNames, ", "))
}

var spawnLogCmd = &cobra.Command{
	Use:   "spawn-log",
	Short: "Record a new agent spawn entry",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		parent, _ := cmd.Flags().GetString("parent")
		name, _ := cmd.Flags().GetString("name")
		legacyID, _ := cmd.Flags().GetString("id")
		legacyDescription, _ := cmd.Flags().GetString("description")
		if legacyID != "" {
			if parent == "" && name != "" {
				parent = name
			}
			name = legacyID
		}
		if parent == "" {
			outputError(1, "flag --parent is required", nil)
			return nil
		}
		caste := mustGetString(cmd, "caste")
		if caste == "" {
			return nil
		}
		if name == "" {
			outputError(1, "flag --name is required", nil)
			return nil
		}
		task := firstNonEmpty(mustGetStringCompatOptional(cmd, "task"), legacyDescription)
		if task == "" {
			outputError(1, "flag --task is required", nil)
			return nil
		}
		claimedDepth, _ := cmd.Flags().GetInt("depth")
		phase, _ := cmd.Flags().GetInt("phase")

		st := agent.NewSpawnTree(store, "spawn-tree.txt")
		depth, denyReason := deriveSpawnDepth(st, parent)
		if denyReason != "" {
			outputError(1, denyReason, nil)
			return nil
		}

		// SPAWN-01/D-09: the authoritative decision runs here, before
		// RecordSpawn, so a refusal leaves no spawn-tree entry and consumes
		// no budget. RequesterDepth is the PARENT's own depth (derivedDepth
		// - 1), not the prospective child's — the decision itself computes
		// the prospective child's depth as RequesterDepth + 1.
		decision := spawnCanSpawnDecision(spawnDecisionInput{
			RequesterName:        parent,
			RequesterDepth:       depth - 1,
			DepthIsAuthoritative: true,
			Caste:                caste,
			Task:                 task,
			Origin:               spawnOriginSpawnLog,
		})
		if !decision.Allowed {
			outputError(1, decision.Detail, nil)
			return nil
		}

		// CR-01 residual (194-REVIEW.md iteration 2): this is the earliest
		// point in the runtime that fires after the owner has answered the
		// check-in card (or chosen to proceed without answering) and before
		// any worker this spawn describes could possibly run. --phase remains
		// optional for callers outside the protected build ceremony. When it
		// is present, the durable marker is a prerequisite: failing open here
		// would leave the displayed owner capability usable during dispatch.
		if phase > 0 {
			if err := closeForcedReviewerWaiverWindowForPhase(phase, time.Now().UTC()); err != nil {
				outputError(2, fmt.Sprintf("cannot safely begin dispatch: %v", err), nil)
				return nil
			}
		}

		if err := st.RecordSpawn(parent, caste, name, task, depth); err != nil {
			outputError(2, fmt.Sprintf("failed to record spawn: %v", err), nil)
			return nil
		}

		eventID := emitSpawnTreeCeremony(events.CeremonyPayload{
			SpawnID: name,
			Caste:   caste,
			Name:    name,
			Task:    task,
			Status:  "spawned",
		})

		result := map[string]interface{}{
			"recorded":      true,
			"parent":        parent,
			"caste":         caste,
			"name":          name,
			"task":          task,
			"depth":         depth,
			"claimed_depth": claimedDepth,
			"depth_source":  "derived",
		}
		if eventID != "" {
			result["event_id"] = eventID
		}
		// D-13: a passive-awareness line in the run's own output, not a
		// second window. A reporting failure here must never turn a spawn
		// that already succeeded into an error, so the budget state error
		// (if any) is ignored.
		if state, err := spawnTreeBudgetState(); err == nil {
			result["budget_consumed"] = state.Consumed
			result["budget_max"] = state.Max
			if warning := spawnTreeBudgetWarning(state); warning != "" {
				result["budget_warning"] = warning
			}
		}
		outputOK(result)
		return nil
	},
}

var spawnCompleteCmd = &cobra.Command{
	Use:   "spawn-complete",
	Short: "Mark a spawned agent as completed",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		name := mustGetString(cmd, "name")
		if name == "" {
			return nil
		}
		status, _ := cmd.Flags().GetString("status")
		if status == "" {
			status = "completed"
		}
		summary, _ := cmd.Flags().GetString("summary")

		st := agent.NewSpawnTree(store, "spawn-tree.txt")
		// A completion for a worker the ledger has no record of used to be a
		// hard error. That is what leaked the budget on 2026-08-14: the entry
		// stayed live, live entries outside the current window raise the next
		// command's consumed count, and the operator's retry spent a second slot
		// on the same worker. The worker has finished either way; refusing to
		// record it only makes the colony keep paying for it.
		if err := spawnCompleteTolerant(st, name, status, summary, time.Now().UTC()); err != nil {
			outputError(1, fmt.Sprintf("failed to update status: %v", err), nil)
			return nil
		}
		entry := latestSpawnEntryByName(st, name)
		payload := events.CeremonyPayload{
			SpawnID: name,
			Name:    name,
			Status:  status,
			Message: summary,
		}
		if entry != nil {
			payload.Caste = entry.Caste
			payload.Task = entry.Task
		}
		eventID := emitSpawnTreeCeremony(payload)

		result := map[string]interface{}{
			"completed": true,
			"name":      name,
			"status":    status,
		}
		if summary != "" {
			result["summary"] = summary
		}
		if eventID != "" {
			result["event_id"] = eventID
		}
		outputOK(result)
		return nil
	},
}

func emitSpawnTreeCeremony(payload events.CeremonyPayload) string {
	if store == nil {
		return ""
	}
	payload = trimCeremonyPayload(payload)
	raw, err := payload.RawMessage()
	if err != nil {
		return ""
	}
	bus := events.NewBus(store, events.DefaultConfig())
	evt, err := bus.Publish(context.Background(), events.CeremonyTopicBuildSpawn, raw, "aether-spawn")
	if err != nil || evt == nil {
		return ""
	}
	return evt.ID
}

func latestSpawnEntryByName(st *agent.SpawnTree, name string) *agent.SpawnEntry {
	if st == nil {
		return nil
	}
	entries, err := st.Parse()
	if err != nil {
		return nil
	}
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].AgentName == name {
			entry := entries[i]
			return &entry
		}
	}
	return nil
}

// spawnMaxDelegationDepth is the deepest depth a spawn-tree entry may hold
// (D-01/D-05): the coordinator is depth 0, its own workers are depth 1, and
// their helpers are depth 2. A spawn that would be recorded at depth 3 is
// refused, so the refusal test is prospectiveDepth > spawnMaxDelegationDepth.
const spawnMaxDelegationDepth = 2

// spawnDecisionOrigin names which call site produced a spawnDecisionInput,
// so recruitmentAdmissionChecks (cmd/recruitment_admission.go) can declare
// exactly which of BIO-02's five additional dimensions apply to that
// caller, rather than a check silently skipping itself because a field
// happens to be unpopulated. spawn-log and spawn-can-spawn never populate
// the new fields below (Permission, Workspace, CostSlots, IntentID,
// AttemptID) and are deliberately mapped to an EMPTY check set in
// recruitmentAdmissionChecks -- only a real recruitment carries the data
// those five checks need.
type spawnDecisionOrigin string

const (
	spawnOriginSpawnLog      spawnDecisionOrigin = "spawn-log"
	spawnOriginSpawnCanSpawn spawnDecisionOrigin = "spawn-can-spawn"
	spawnOriginRecruit       spawnDecisionOrigin = "recruit"
)

// spawnDecisionOrigins returns every declared origin, mirroring
// recruitmentAdapterKinds()'s/ColonyLiveTopics()'s completeness convention:
// TestRecruitmentAdmissionChecksCoverEveryOrigin
// (cmd/recruitment_admission_test.go) derives its inventory from this
// function's RUNTIME output, so a new origin added here with no row in
// recruitmentAdmissionChecks fails that test by name.
func spawnDecisionOrigins() []spawnDecisionOrigin {
	return []spawnDecisionOrigin{spawnOriginSpawnLog, spawnOriginSpawnCanSpawn, spawnOriginRecruit}
}

// spawnDecisionInput is what a caller (or the recorder itself) knows about a
// prospective spawn at decision time. RequesterName/RequesterDepth describe
// the WOULD-BE PARENT, not the child being proposed — the decision computes
// the child's own prospective depth as RequesterDepth + 1.
//
// DepthIsAuthoritative distinguishes spawn-log's call (true — RequesterDepth
// came from the parent's own recorded spawn-tree entry, per deriveSpawnDepth)
// from spawn-can-spawn's advisory call without --name (false — RequesterDepth
// is whatever the caller claims about itself). This is the RESIDUE named in
// this plan's must_haves: spawn-can-spawn without --name reports on a depth
// the caller states about itself; only spawn-log is authoritative, because a
// caller can lie to the checker but cannot avoid the recorder.
type spawnDecisionInput struct {
	RequesterName        string
	RequesterDepth       int
	DepthIsAuthoritative bool
	Caste                string
	Task                 string

	// Origin declares which call site produced this input (spawnDecisionOrigin
	// above). It decides which of BIO-02's five additional checks apply, via
	// recruitmentAdmissionChecks. The zero value (empty string) matches no
	// declared origin and applies none of the five new checks -- exactly
	// what every pre-existing test and call site that never set Origin
	// continues to get, unchanged.
	Origin spawnDecisionOrigin
	// Permission is the resolved (never caller-trusted) permission profile
	// for the requested caste -- BIO-02's permission dimension.
	Permission codex.PermissionProfile
	// Workspace is the workspace lease this recruitment declares -- BIO-02's
	// path-containment dimension.
	Workspace string
	// CostSlots is the number of whole-run helper-budget slots this request
	// counts against -- BIO-02's cost dimension. D-12: read against the SAME
	// spawnTreeBudgetState() ledger, never a second counter.
	CostSlots int
	// IntentID/AttemptID identify this specific recruitment request, so the
	// duplicate-intent check can name the pending intent it matched and
	// exclude the request being decided from matching itself.
	IntentID  string
	AttemptID string
}

// spawnDecisionResult is the outcome of a spawnCanSpawnDecision call. Reason
// is one of the exact strings "depth", "budget", "ancestor-cycle", "parent",
// "permission", "path", "cost", "duplicate", "unresolved", or empty when
// Allowed is true. Detail is the human-readable sentence D-10 requires —
// naming which helper, whose child, and why — and is what reaches the
// operator through --enforce's error message.
type spawnDecisionResult struct {
	Allowed bool
	Reason  string
	Detail  string
}

// spawnCanSpawnDecision is the single chokepoint SPAWN-01/BIO-02 makes real:
// depth, then whole-run budget, then ancestor-cycle, then -- for a real
// recruitment only, per recruitmentAdmissionChecks' declared table -- parent
// authority, permission, path containment, cost, and duplicate intent, each
// named and each denying on the first hit. It is a package-level function
// variable (not a plain func) specifically so a test can substitute a deny
// answer for the duration of a single test case, driving --enforce's
// deny-to-non-zero-exit path.
var spawnCanSpawnDecision = func(in spawnDecisionInput) spawnDecisionResult {
	prospectiveDepth := in.RequesterDepth + 1
	if prospectiveDepth > spawnMaxDelegationDepth {
		requesterName := in.RequesterName
		if requesterName == "" {
			requesterName = "the requester"
		}
		return spawnDecisionResult{
			Allowed: false,
			Reason:  "depth",
			Detail: fmt.Sprintf(
				"%s is at depth %d; a helper spawned from here would be depth %d, past the cap of %d",
				requesterName, in.RequesterDepth, prospectiveDepth, spawnMaxDelegationDepth,
			),
		}
	}

	if reason := spawnTreeBudgetReason(in); reason != "" {
		return spawnDecisionResult{Allowed: false, Reason: "budget", Detail: reason}
	}

	if reason := spawnAncestorCycleReason(in); reason != "" {
		return spawnDecisionResult{Allowed: false, Reason: "ancestor-cycle", Detail: reason}
	}

	// BIO-02's four* new dimensions, layered onto the same chokepoint rather
	// than a parallel one (SYN-203-02). *Five: parent authority, permission,
	// path, cost, duplicate -- applying only to the origin(s) declared in
	// recruitmentAdmissionChecks (cmd/recruitment_admission.go), never
	// implicitly to a caller whose input lacks the data a check needs.
	if recruitmentCheckApplies(in.Origin, recruitmentReasonParent) {
		if reason := recruitmentParentAuthorityReason(in); reason != "" {
			return spawnDecisionResult{Allowed: false, Reason: recruitmentReasonParent, Detail: reason}
		}
	}
	if recruitmentCheckApplies(in.Origin, recruitmentReasonPermission) {
		if reason := recruitmentPermissionReason(in); reason != "" {
			return spawnDecisionResult{Allowed: false, Reason: recruitmentReasonPermission, Detail: reason}
		}
	}
	if recruitmentCheckApplies(in.Origin, recruitmentReasonPath) {
		if reason := recruitmentPathReason(in); reason != "" {
			return spawnDecisionResult{Allowed: false, Reason: recruitmentReasonPath, Detail: reason}
		}
	}
	if recruitmentCheckApplies(in.Origin, recruitmentReasonCost) {
		if reason := recruitmentCostReason(in); reason != "" {
			return spawnDecisionResult{Allowed: false, Reason: recruitmentReasonCost, Detail: reason}
		}
	}
	if recruitmentCheckApplies(in.Origin, recruitmentReasonDuplicate) {
		if reason := recruitmentDuplicateReason(in); reason != "" {
			return spawnDecisionResult{Allowed: false, Reason: recruitmentReasonDuplicate, Detail: reason}
		}
	}

	return spawnDecisionResult{Allowed: true}
}

var spawnCanSpawnCmd = &cobra.Command{
	Use:   "spawn-can-spawn",
	Short: "Check if spawning is allowed at given depth",
	// D-14: accept the positional depth exactly as .aether/workers.md:292
	// sends it (`aether spawn-can-spawn {your_depth} --enforce`), while still
	// accepting the flag-only form the build playbooks send
	// (`aether spawn-can-spawn --depth {depth}`).
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		depth := mustGetInt(cmd, "depth")
		if len(args) == 1 {
			parsed, err := strconv.Atoi(args[0])
			if err != nil {
				outputError(1, fmt.Sprintf("invalid depth %q: must be an integer", args[0]), nil)
				return nil
			}
			// Positional wins over --depth when both are present.
			depth = parsed
		}

		enforce, _ := cmd.Flags().GetBool("enforce")
		name, _ := cmd.Flags().GetString("name")

		in := spawnDecisionInput{RequesterDepth: depth, Origin: spawnOriginSpawnCanSpawn}
		// Set RequesterName from --name whenever --name is non-empty,
		// whether or not it resolves to a recorded entry: the ancestor
		// check keys off RequesterName, so dropping it on a failed lookup
		// would silently skip that check for exactly the caller whose
		// identity could not be confirmed.
		if name != "" {
			in.RequesterName = name
			if store != nil {
				st := agent.NewSpawnTree(store, "spawn-tree.txt")
				if entry := latestSpawnEntryByName(st, name); entry != nil {
					in.RequesterDepth = entry.Depth
					in.DepthIsAuthoritative = true
				}
			}
		}

		decision := spawnCanSpawnDecision(in)

		if enforce && !decision.Allowed {
			msg := fmt.Sprintf("spawn denied at depth %d", depth)
			if decision.Detail != "" {
				msg = fmt.Sprintf("%s: %s", msg, decision.Detail)
			}
			outputError(1, msg, nil)
			return nil
		}

		result := map[string]interface{}{
			"can_spawn":     decision.Allowed,
			"depth":         depth,
			"authoritative": in.DepthIsAuthoritative,
		}
		if !decision.Allowed {
			result["reason"] = decision.Reason
			result["detail"] = decision.Detail
		}
		outputOK(result)
		return nil
	},
}

var spawnTreeLoadCmd = &cobra.Command{
	Use:   "spawn-tree-load",
	Short: "Load and return the full spawn tree as JSON",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		st := agent.NewSpawnTree(store, "spawn-tree.txt")
		data, err := st.ToJSON()
		if err != nil {
			outputError(1, fmt.Sprintf("failed to load spawn tree: %v", err), nil)
			return nil
		}

		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			outputError(1, fmt.Sprintf("failed to parse spawn tree: %v", err), nil)
			return nil
		}

		outputOK(result)
		return nil
	},
}

var spawnTreeActiveCmd = &cobra.Command{
	Use:   "spawn-tree-active",
	Short: "List active (non-completed) spawn entries",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		st := agent.NewSpawnTree(store, "spawn-tree.txt")
		active := st.Active()

		if active == nil {
			active = []agent.SpawnEntry{}
		}

		// D-14: a non-technical operator reads this command's own terminal
		// output, live, while a build is still running -- not a JSON array.
		// The JSON payload below is untouched (same keys, same shape), so
		// this branch is purely additive and every existing reader keeps
		// working.
		if shouldRenderVisualOutput(stdout) {
			writeVisualOutput(stdout, renderSpawnTreeActiveVisual(active))
			return nil
		}

		type entryJSON struct {
			Name      string `json:"name"`
			Parent    string `json:"parent"`
			Caste     string `json:"caste"`
			Task      string `json:"task"`
			Depth     int    `json:"depth"`
			Status    string `json:"status"`
			SpawnedAt string `json:"spawned_at"`
		}

		entries := make([]entryJSON, len(active))
		for i, e := range active {
			entries[i] = entryJSON{
				Name:      e.AgentName,
				Parent:    e.ParentName,
				Caste:     e.Caste,
				Task:      e.Task,
				Depth:     e.Depth,
				Status:    e.Status,
				SpawnedAt: e.Timestamp,
			}
		}

		outputOK(map[string]interface{}{
			"active": entries,
			"count":  len(entries),
		})
		return nil
	},
}

// renderSpawnTreeActiveVisual is the D-14 English rendering of the active
// spawn tree: depth-indented, parent-attributed, and safe to call while a
// run is still writing spawn-tree.txt -- Active() already reads through the
// shared read-lock path, and this function performs no write, create,
// publish or update of its own (T-173-40/T-173-41).
//
// Indentation is driven directly by each entry's own recorded Depth (D-05's
// authority, not a tree walk this command would have to re-derive), two
// spaces per level. Ordering is depth-ascending, then spawn-time ascending,
// so two calls over an unchanged tree render byte-identical output --
// required by TestSpawnTreeActiveMutatesNothing and by an operator who runs
// the command twice in a row.
func renderSpawnTreeActiveVisual(active []agent.SpawnEntry) string {
	var b strings.Builder

	// D-13: reuse the same budget state spawn-log's own warning line already
	// computes, so this view and that warning never disagree. This is an
	// inspection command, so a budget-state read failure must not fail the
	// whole command -- the header is skipped, not the tree.
	state, budgetErr := spawnTreeBudgetState()

	if budgetErr == nil {
		fmt.Fprintf(&b, "This run has used %d of the %d helpers it is allowed to spawn.\n\n", state.Consumed, state.Max)
	}

	if len(active) == 0 {
		b.WriteString("No helpers are currently working.\n")
	} else {
		sorted := make([]agent.SpawnEntry, len(active))
		copy(sorted, active)
		sort.SliceStable(sorted, func(i, j int) bool {
			if sorted[i].Depth != sorted[j].Depth {
				return sorted[i].Depth < sorted[j].Depth
			}
			return sorted[i].Timestamp < sorted[j].Timestamp
		})

		byName := make(map[string]agent.SpawnEntry, len(sorted))
		for _, e := range sorted {
			byName[e.AgentName] = e
		}

		for _, e := range sorted {
			indent := strings.Repeat("  ", e.Depth)
			who := casteIdentity(e.Caste) + " " + e.AgentName
			task := strings.TrimSpace(e.Task)
			if task == "" {
				task = "a task with no description"
			}
			fmt.Fprintf(&b, "%s%s is still working -- %s, sent here by %s\n", indent, who, task, spawnTreeActiveParentPhrase(e.ParentName, byName))
		}
	}

	b.WriteString("\n")
	if budgetErr == nil {
		fmt.Fprintf(&b, "In one round, the Queen sends at most a handful of helpers; across the whole run, no more than %d helpers may ever spawn.\n", state.Max)
	} else {
		b.WriteString("In one round, the Queen sends at most a handful of helpers; the whole run has its own separate limit on how many it may ever spawn.\n")
	}

	return b.String()
}

// spawnTreeActiveParentPhrase names the caller in words. When the parent is
// itself still active it is shown with its own caste identity; when the
// parent is a coordinator sentinel or has already completed (and so dropped
// out of the active set), it is still named by the plain string it
// recorded -- losing that name would be exactly the runaway-subtree blind
// spot D-14/T-173-44 exists to close.
func spawnTreeActiveParentPhrase(parent string, byName map[string]agent.SpawnEntry) string {
	if p, ok := byName[parent]; ok {
		return casteIdentity(p.Caste) + " " + p.AgentName
	}
	return parent
}

var spawnTreeDepthCmd = &cobra.Command{
	Use:   "spawn-tree-depth",
	Short: "Get the maximum spawn depth",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		st := agent.NewSpawnTree(store, "spawn-tree.txt")
		entries, _ := st.Parse()

		maxDepth := 0
		for _, e := range entries {
			if e.Depth > maxDepth {
				maxDepth = e.Depth
			}
		}

		outputOK(map[string]interface{}{
			"max_depth": maxDepth,
			"total":     len(entries),
		})
		return nil
	},
}

var spawnEfficiencyCmd = &cobra.Command{
	Use:   "spawn-efficiency",
	Short: "Calculate spawn completion efficiency",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		st := agent.NewSpawnTree(store, "spawn-tree.txt")
		entries, _ := st.Parse()

		total := len(entries)
		completed := 0
		for _, e := range entries {
			if e.Status == "completed" || e.Status == "failed" || e.Status == "blocked" {
				completed++
			}
		}

		efficiency := float64(0)
		if total > 0 {
			efficiency = float64(completed) / float64(total) * 100
		}

		outputOK(map[string]interface{}{
			"total":      total,
			"completed":  completed,
			"active":     total - completed,
			"efficiency": fmt.Sprintf("%.1f%%", efficiency),
		})
		return nil
	},
}

var validateWorkerResponseCmd = &cobra.Command{
	Use:   "validate-worker-response",
	Short: "Validate a worker response for basic correctness",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		response := mustGetString(cmd, "response")
		if response == "" {
			return nil
		}

		valid := true
		reason := ""

		if len(response) < 10 {
			valid = false
			reason = "response too short"
		}

		// Check if it's expected to be JSON
		checkJSON, _ := cmd.Flags().GetBool("expect-json")
		if checkJSON && !json.Valid([]byte(response)) {
			valid = false
			reason = "expected valid JSON"
		}

		outputOK(map[string]interface{}{
			"valid":  valid,
			"reason": reason,
		})
		return nil
	},
}

func init() {
	spawnLogCmd.Flags().String("parent", "", "Parent agent name (required)")
	spawnLogCmd.Flags().String("caste", "", "Agent caste (required)")
	spawnLogCmd.Flags().String("name", "", "Agent name (required)")
	spawnLogCmd.Flags().String("id", "", "Legacy alias for child agent name")
	spawnLogCmd.Flags().String("task", "", "Task description (required)")
	spawnLogCmd.Flags().String("description", "", "Legacy alias for task description")
	spawnLogCmd.Flags().Int("depth", 0, "Advisory only; the recorded depth is derived from --parent")
	spawnLogCmd.Flags().Int("phase", 0, "Phase this worker is spawned for (optional); closes that phase's forced-reviewer decline window on first use")

	spawnCompleteCmd.Flags().String("name", "", "Agent name to complete (required)")
	spawnCompleteCmd.Flags().String("status", "", "Status: completed, failed, blocked (default: completed)")
	spawnCompleteCmd.Flags().String("summary", "", "Completion summary (optional)")

	spawnCanSpawnCmd.Flags().Int("depth", 0, "Spawn depth to check (required)")
	spawnCanSpawnCmd.Flags().Bool("enforce", false, "Exit non-zero when spawning is denied")
	spawnCanSpawnCmd.Flags().String("name", "", "Requester's recorded agent name; when it resolves, the recorded depth overrides --depth")

	validateWorkerResponseCmd.Flags().String("response", "", "Response to validate (required)")
	validateWorkerResponseCmd.Flags().Bool("expect-json", false, "Check if response is valid JSON")

	rootCmd.AddCommand(spawnLogCmd)
	rootCmd.AddCommand(spawnCompleteCmd)
	rootCmd.AddCommand(spawnCanSpawnCmd)
	rootCmd.AddCommand(spawnTreeLoadCmd)
	rootCmd.AddCommand(spawnTreeActiveCmd)
	rootCmd.AddCommand(spawnTreeDepthCmd)
	rootCmd.AddCommand(spawnEfficiencyCmd)
	rootCmd.AddCommand(validateWorkerResponseCmd)
}
