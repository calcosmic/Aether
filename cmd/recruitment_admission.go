package cmd

import (
	"fmt"
	"strings"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/codex"
)

// Reason classes new to this plan (BIO-02). recruitmentReasonParent,
// recruitmentReasonPermission, and recruitmentReasonCost already exist in
// cmd/recruitment_intent.go's eleven-reason validateRecruitmentIntent
// vocabulary and are reused here rather than redeclared -- they name the
// identical concept one layer later (admission, not intent-shape
// validation), and a single fixed string per concept keeps every caller
// switching on Reason able to rely on one vocabulary.
const (
	recruitmentReasonPath       = "path"
	recruitmentReasonDuplicate  = "duplicate"
	recruitmentReasonUnresolved = "unresolved"
)

// recruitmentAdmissionReasons is the complete declared reason vocabulary for
// the whole recruitment admission surface: spawnCanSpawnDecision's own three
// pre-existing reasons, this plan's five new ones, and "unresolved" (a
// failed manifest write, cmd/recruitment.go). TestEveryAdmissionReasonIsReachable
// (cmd/recruitment_admission_test.go) cross-checks this list against the
// checks that actually emit each string, in both directions.
func recruitmentAdmissionReasons() []string {
	return []string{
		"depth", "budget", "ancestor-cycle",
		recruitmentReasonParent, recruitmentReasonPermission, recruitmentReasonPath,
		recruitmentReasonCost, recruitmentReasonDuplicate, recruitmentReasonUnresolved,
	}
}

// recruitmentAdmissionChecks is the declared table BIO-02's must_haves
// require: which of the five new admission dimensions apply to which
// spawnDecisionOrigin. spawn-log and spawn-can-spawn are deliberately
// mapped to an EMPTY set -- they never populate Permission/Workspace/
// CostSlots/IntentID, and applying these checks to them would deny every
// ordinary spawn on missing data rather than skip a check that does not
// apply. Only a real recruitment (spawnOriginRecruit) carries the data
// these five checks need, so only it is subject to them.
var recruitmentAdmissionChecks = map[spawnDecisionOrigin][]string{
	spawnOriginSpawnLog:      {},
	spawnOriginSpawnCanSpawn: {},
	spawnOriginRecruit: {
		recruitmentReasonParent,
		recruitmentReasonPermission,
		recruitmentReasonPath,
		recruitmentReasonCost,
		recruitmentReasonDuplicate,
	},
}

// recruitmentCheckApplies reports whether check is declared applicable to
// origin. An origin with no row (including the empty/unset zero value) is
// never subject to any of these checks -- the correct behaviour for an
// undeclared caller is the one this codebase already had before this plan
// (no new checks), which recruitmentAdmissionChecks preserves EXPLICITLY
// (an empty slice, not an absent row for the three declared origins) rather
// than by silent omission.
func recruitmentCheckApplies(origin spawnDecisionOrigin, check string) bool {
	checks, ok := recruitmentAdmissionChecks[origin]
	if !ok {
		return false
	}
	for _, c := range checks {
		if c == check {
			return true
		}
	}
	return false
}

// recruitmentParentAuthorityReason is BIO-02's parent-authority dimension.
// This plan's own flagged, unresolved edge probe (203-06-PLAN.md's
// objective) assumes "parent authority" means the named parent resolves to
// a recorded, non-terminal spawn-tree entry -- surfaced, not silently
// decided elsewhere. Mirrors spawnAncestorCycleReason's D-19 fail-closed
// shape exactly: an unreadable ledger denies, never allows.
func recruitmentParentAuthorityReason(in spawnDecisionInput) string {
	if spawnParentIsRoot(in.RequesterName) {
		// The coordinator sentinel has no spawn-tree entry of its own and is
		// always authoritative -- the one legitimate no-entry case.
		return ""
	}
	if strings.TrimSpace(in.RequesterName) == "" {
		return "no parent name was named to verify authority against; refusing to spawn"
	}
	if store == nil {
		return "parent authority unreadable (no store initialized): refusing to spawn"
	}

	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	entries, err := st.Parse()
	if err != nil {
		// D-19 fail-closed: an unreadable ledger must deny, never allow.
		return fmt.Sprintf("parent authority unreadable (%v): refusing to spawn", err)
	}

	var entry *agent.SpawnEntry
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].AgentName == in.RequesterName {
			e := entries[i]
			entry = &e
			break
		}
	}
	if entry == nil {
		return fmt.Sprintf(
			"parent %q resolves to no recorded spawn entry and is not a coordinator sentinel; a recruitment's authority is never trusted from a self-declared parent",
			in.RequesterName,
		)
	}
	if agent.IsTerminalSpawnStatus(entry.Status) {
		return fmt.Sprintf(
			"parent %q is recorded with terminal status %q; a finished helper may not authorise a new recruitment",
			in.RequesterName, entry.Status,
		)
	}
	return ""
}

// recruitmentPermissionReason is BIO-02's permission dimension. It calls
// codex.PermissionProfileForCaste directly rather than reimplementing
// permission resolution (RESEARCH.md "Don't Hand-Roll").
//
// Unclassified planner assumption, recorded here rather than silently
// resolved: BIO-02's wording denies a read-only caste "asked to write."
// Every recruitment this runtime admits dispatches a real child process
// into a writable workspace (cmd/recruitment_dispatch.go's
// dispatchRecruitment has no separate read-only dispatch mode), so a caste
// whose resolved profile is repository-read-only is, in effect, ALWAYS
// "asked to write" the moment it is recruited this way -- there is
// currently no code path that recruits a caste without eventually running
// it in a write-capable child process. This collapses BIO-02's distinction
// to "every recruitment of a read-only caste is refused" for this codebase
// today; if a genuinely read-only dispatch mode is ever added, this check
// must be revisited.
func recruitmentPermissionReason(in spawnDecisionInput) string {
	if strings.TrimSpace(in.Caste) == "" {
		return "no caste was named to resolve a permission profile against; refusing to spawn"
	}
	profile := in.Permission
	if profile.Name == "" {
		// A caller that left Permission unresolved is not trusted to have
		// already checked it -- resolve it the same way
		// validateRecruitmentIntent already does, rather than silently
		// allowing on missing data.
		profile = codex.PermissionProfileForCaste(in.Caste)
	}
	if profile.Name == codex.PermissionRepositoryReadOnly {
		return fmt.Sprintf(
			"caste %q resolves to permission profile %q (repository-read-only); every recruitment dispatches a real, write-capable child process, so a read-only caste may never be recruited this way",
			in.Caste, profile.Name,
		)
	}
	return ""
}

// recruitmentPathReason is BIO-02's path-containment dimension. It calls
// validateSpendContainedPath directly (cmd/spend_session_capture.go) -- the
// same symlink-aware containment boundary the spend subsystem already uses
// -- rather than reimplementing path checking.
func recruitmentPathReason(in spawnDecisionInput) string {
	if strings.TrimSpace(in.Workspace) == "" {
		return "no workspace was declared for this recruitment; refusing to spawn"
	}
	if store == nil {
		return "colony root unresolvable (no store initialized): refusing to spawn"
	}
	root := repoRootFromStore(store)
	if strings.TrimSpace(root) == "" {
		return "colony root unresolvable: refusing to spawn"
	}
	if _, err := validateSpendContainedPath(root, in.Workspace, "recruitment workspace"); err != nil {
		return err.Error()
	}
	return ""
}

// recruitmentCostReason is BIO-02's cost dimension. D-12: it reads the SAME
// whole-run ledger every ordinary spawn already reads (spawnTreeBudgetState,
// cmd/spawn_budget.go) -- never a second counter, and no arithmetic on a
// budget total is performed anywhere in this file.
func recruitmentCostReason(in spawnDecisionInput) string {
	if in.CostSlots <= 0 {
		return "no cost-slots were declared for this recruitment; refusing to spawn"
	}
	state, err := spawnTreeBudgetState()
	if err != nil {
		// D-19 fail-closed: an unverifiable budget must deny, never allow.
		return fmt.Sprintf("whole-run helper budget unverifiable (%v): refusing to spawn", err)
	}
	if in.CostSlots > state.Remaining {
		return fmt.Sprintf(
			"this recruitment requests %d helper slot(s), but only %d of the whole-run budget of %d remain (%d already consumed)",
			in.CostSlots, state.Remaining, state.Max, state.Consumed,
		)
	}
	return ""
}

// recruitmentDuplicateReason is BIO-02's duplicate-intent dimension. "The
// same subtree" is read as the requester's own ancestor lineage
// (spawnAncestorChain, cmd/spawn_ancestor.go) plus the requester itself --
// reusing the exact chain-walking infrastructure spawnAncestorCycleReason
// already uses, so the duplicate rule and the cycle rule read the tree the
// same way and cannot silently drift apart, per this plan's own action
// text. "Pending" means a recorded recruitment/intents.json entry with no
// Decision attached yet -- a genuine concurrent request, not one already
// resolved either way.
func recruitmentDuplicateReason(in spawnDecisionInput) string {
	if store == nil {
		return "recorded recruitment intents unreadable (no store initialized): refusing to spawn"
	}
	exists, err := store.FileExists(recruitmentIntentsPath)
	if err != nil {
		return fmt.Sprintf("recorded recruitment intents unreadable (%v): refusing to spawn", err)
	}
	if !exists {
		return ""
	}
	var file recruitmentIntentsFile
	if err := store.LoadJSON(recruitmentIntentsPath, &file); err != nil {
		return fmt.Sprintf("recorded recruitment intents unreadable (%v): refusing to spawn", err)
	}

	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	chain, chainErr := spawnAncestorChain(st, in.RequesterName)
	if chainErr != nil {
		// D-19 fail-closed: an unreadable lineage must deny, never allow.
		return fmt.Sprintf("recruitment lineage unreadable (%v): refusing to spawn", chainErr)
	}
	subtree := map[string]bool{in.RequesterName: true}
	for _, ancestor := range chain {
		subtree[ancestor.AgentName] = true
	}

	normalizedTask := normalizeSpawnTask(in.Task)
	for _, entry := range file.Entries {
		if entry.Decision != nil {
			continue
		}
		if entry.Intent.IntentID == in.IntentID {
			continue
		}
		if !strings.EqualFold(entry.Intent.Caste, in.Caste) {
			continue
		}
		if normalizeSpawnTask(entry.Intent.Objective) != normalizedTask {
			continue
		}
		if !subtree[entry.Intent.ParentName] {
			continue
		}
		return fmt.Sprintf(
			"a recruitment for caste %q and this same objective is already pending in this subtree (intent %q)",
			in.Caste, entry.Intent.IntentID,
		)
	}
	return ""
}

// recruitmentDepthOverride resolves D-11's per-run raise. maxDepthFlag <= 0
// or <= spawnMaxDelegationDepth means no raise -- the default cap governs,
// unchanged, and raised is false (the default stays at two, as D-11
// requires). A positive value greater than the default cap raises the
// effective cap to exactly that value for this one recruitment.
// spawnTreeBudgetMax (the whole-run tree budget, cmd/spawn_budget.go) is a
// completely separate quantity from the depth cap and is never touched by
// this flag or this function.
func recruitmentDepthOverride(maxDepthFlag int) (cap int, raised bool) {
	if maxDepthFlag <= spawnMaxDelegationDepth {
		return spawnMaxDelegationDepth, false
	}
	return maxDepthFlag, true
}
