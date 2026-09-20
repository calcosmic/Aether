package cmd

import (
	"errors"
	"fmt"
	"strings"
	"time"

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

// recruitmentManifestPath is the store-relative path amendRecruitmentManifest
// persists to. New data file, per this plan's frontmatter.
const recruitmentManifestPath = "recruitment/manifest.json"

// recruitmentManifestSchemaVersion is the wire-shape version every
// amendment record carries.
const recruitmentManifestSchemaVersion = "recruitment-manifest/v1"

// The two manifest lifecycle states. recruitmentManifestStateAdmitted is
// the recoverable signature of a crash between admission and dispatch;
// recruitmentManifestStateDispatched marks that the runtime has committed
// to starting the child -- see amendRecruitmentManifest's own doc comment
// for the exact boundary this transition is written at.
const (
	recruitmentManifestStateAdmitted   = "admitted"
	recruitmentManifestStateDispatched = "dispatched"
)

// recruitmentManifestRecord is the amendment record: an admitted child,
// recorded atomically before that child ever runs.
type recruitmentManifestRecord struct {
	SchemaVersion string `json:"schema_version"`
	IntentID      string `json:"intent_id"`
	ChildName     string `json:"child_name,omitempty"`
	ParentName    string `json:"parent_name,omitempty"`
	Depth         int    `json:"depth,omitempty"`
	AdapterKind   string `json:"adapter_kind,omitempty"`
	Workspace     string `json:"workspace,omitempty"`
	AdmittedAt    string `json:"admitted_at,omitempty"`
	State         string `json:"state"`
}

// recruitmentManifestFile is the on-disk container at recruitmentManifestPath.
type recruitmentManifestFile struct {
	Entries []recruitmentManifestRecord `json:"entries"`
}

// errRecruitmentManifestUnknownSchema is returned by
// recruitmentManifestEntryByIntentID when a stored entry's SchemaVersion
// does not match recruitmentManifestSchemaVersion -- reported as unknown
// rather than the reader fabricating a field that version may not share.
var errRecruitmentManifestUnknownSchema = errors.New("recruitment manifest entry carries an unrecognised schema version")

// amendRecruitmentManifest writes or updates the manifest entry named by
// record.IntentID through store.UpdateJSONAtomically. BIO-02's rule that a
// failed manifest write must deny launch rather than proceed unrecorded is
// enforced at recruitCmd's own call site (cmd/recruitment.go): this
// function's only job is to return the write error honestly, never to
// swallow it.
//
// Timing note on the "admitted"->"dispatched" transition: BIO-04's
// behaviour spec asks for this write "after the child process actually
// starts". cmd/recruitment_dispatch.go's dispatchRecruitment is a
// synchronous call this plan does not own and cannot instrument
// mid-execution, so recruitCmd writes the "dispatched" transition
// immediately before invoking dispatchRecruitment -- the closest
// observable point to "the runtime has committed to starting this child"
// available without editing a file outside this plan's declared ownership.
// This is a considered, documented boundary, not a silent reinterpretation
// of the requirement.
func amendRecruitmentManifest(record recruitmentManifestRecord) (recruitmentManifestRecord, error) {
	if store == nil {
		return recruitmentManifestRecord{}, fmt.Errorf("no store initialized")
	}
	if strings.TrimSpace(record.IntentID) == "" {
		return recruitmentManifestRecord{}, fmt.Errorf("recruitment manifest amendment requires a non-empty IntentID")
	}
	if strings.TrimSpace(record.SchemaVersion) == "" {
		record.SchemaVersion = recruitmentManifestSchemaVersion
	}

	var bound recruitmentManifestRecord
	var file recruitmentManifestFile
	err := store.UpdateJSONAtomically(recruitmentManifestPath, &file, func() error {
		for i := range file.Entries {
			if file.Entries[i].IntentID != record.IntentID {
				continue
			}
			if strings.TrimSpace(record.State) != "" {
				file.Entries[i].State = record.State
			}
			if strings.TrimSpace(record.AdapterKind) != "" {
				file.Entries[i].AdapterKind = record.AdapterKind
			}
			bound = file.Entries[i]
			return nil
		}
		file.Entries = append(file.Entries, record)
		bound = record
		return nil
	})
	if err != nil {
		return recruitmentManifestRecord{}, err
	}
	return bound, nil
}

// recruitmentManifestEntryByIntentID is the read-side counterpart:
// TestRecruitmentManifestAmendmentUnknownSchemaVersion
// (cmd/recruitment_admission_test.go) proves an entry written under a
// schema version this runtime does not recognise is reported as unknown
// rather than a fabricated zero-value read.
func recruitmentManifestEntryByIntentID(intentID string) (recruitmentManifestRecord, error) {
	if store == nil {
		return recruitmentManifestRecord{}, fmt.Errorf("no store initialized")
	}
	var file recruitmentManifestFile
	if err := store.LoadJSON(recruitmentManifestPath, &file); err != nil {
		return recruitmentManifestRecord{}, err
	}
	for _, entry := range file.Entries {
		if entry.IntentID != intentID {
			continue
		}
		if entry.SchemaVersion != recruitmentManifestSchemaVersion {
			return recruitmentManifestRecord{}, fmt.Errorf("%w: %q", errRecruitmentManifestUnknownSchema, entry.SchemaVersion)
		}
		return entry, nil
	}
	return recruitmentManifestRecord{}, fmt.Errorf("no recruitment manifest entry for IntentID %q", intentID)
}

// recruitmentClaimAdmission is the host/autopilot lane's own entry into the
// exact sequence dispatchOneInRepoRecruitment (cmd/recruitment_lane.go)
// already uses to carry a worker's spawn claim through a real recruitment:
// build a full recruitmentIntent, validate its shape
// (validateRecruitmentIntent), durably record it BEFORE any decision is
// taken, decide through spawnCanSpawnDecision under spawnOriginRecruit --
// the SAME origin, and the SAME five extra admission dimensions (parent
// authority, permission, path, cost, duplicate) the in-repo lane and
// `aether recruit` already apply -- and durably record the decision.
//
// This is CR-01's fix (203-REVIEW.md). spawn-orchestrator.ts's own header
// comment and 203-CLASSIC-SYNTHESIS.md's SYN-203-02 both asserted that a
// host-lane recruitment and a native-lane recruitment against the same
// ledger state produce the same allow/deny answer with the same reason
// vocabulary. Before this function existed, the TypeScript host's
// spawn-orchestrator bridge asked ONLY under spawnOriginSpawnCanSpawn --
// whose declared check table (recruitmentAdmissionChecks) is EMPTY -- so
// permission/path/cost/duplicate were never evaluated for a host-lane
// recruitment even though the native lane enforced all five. A read-only
// caste requesting a write workspace was refused via `aether recruit` and
// silently admitted via the host/autopilot lane.
//
// spawnCanSpawnCmd's --recruitment flag (cmd/spawn.go) calls this function
// instead of the bare spawnCanSpawnDecision(in) call it otherwise makes --
// the one caller today, reached from the TypeScript host's
// spawn-orchestrator bridge. spawnOriginSpawnCanSpawn's OWN declared check
// table stays empty either way: an ordinary advisory spawn-can-spawn call
// (no --recruitment flag) is completely unaffected by this function's
// existence, exactly as recruitmentAdmissionChecks' own comment requires --
// spawn-log and an ordinary spawn-can-spawn call never populate
// Permission/Workspace/CostSlots/IntentID, and applying these checks to
// them would deny every ordinary spawn on missing data.
//
// parentName/parentDepth/depthIsAuthoritative are taken as already resolved
// by the caller: spawnCanSpawnCmd already derives them from --name against
// the spawn tree (a coordinator sentinel, or a recorded spawn-tree entry's
// own depth + 1 -- D-05, a caller's claimed depth is never trusted on its
// own), exactly the way recruitCmd and dispatchOneInRepoRecruitment resolve
// the same fact for their own callers. This function does not re-derive
// them, so there is exactly one place per caller that resolves a claimed
// parent's authority, never two that could quietly disagree.
//
// costSlots mirrors `aether recruit`'s own --cost-slots flag (default 1,
// one helper slot per recruitment -- the same default
// dispatchOneInRepoRecruitment hardcodes for the in-repo lane) rather than
// being fixed in this function: a whole-run budget already exhausted denies
// via the SAME generic spawnTreeBudgetReason check every origin shares
// (reason "budget") before this dimension is ever reached, so exercising
// recruitmentCostReason itself (reason "cost") -- this request's OWN
// declared slot count exceeding what remains, while slots still remain --
// requires a caller-suppliable value here, not a constant.
//
// It deliberately stops at the decision: it never writes a spawn-tree entry
// and never amends the recruitment manifest. Both the in-repo lane and the
// interactive `aether recruit` command record the spawn and dispatch the
// child themselves immediately after an identical admission call -- the
// host/autopilot lane already has its OWN mechanism for that
// (worker-dispatch.ts's own spawn-log call, immediately before the worker
// process actually starts), which performs the equivalent spawn-tree
// registration for EVERY worker it dispatches, recruited or not. Recording
// a second spawn-tree entry here would double-count this same child against
// the whole-run budget the moment worker-dispatch.ts's own spawn-log call
// runs -- so this function's contract is admission only, the one piece of
// the sequence the two lanes cannot already share.
func recruitmentClaimAdmission(parentName string, parentDepth int, depthIsAuthoritative bool, caste, task, workspace string, costSlots int) (intentID string, decision recruitmentDecisionResult) {
	ws := strings.TrimSpace(workspace)
	if ws == "" && store != nil {
		ws = repoRootFromStore(store)
	}
	if costSlots <= 0 {
		costSlots = 1
	}

	attemptID := fmt.Sprintf("hostrecruit_%d", time.Now().UTC().UnixNano())
	intent := recruitmentIntent{
		SchemaVersion:        recruitmentSchemaVersion,
		ParentName:           parentName,
		ParentDepth:          parentDepth,
		DepthIsAuthoritative: depthIsAuthoritative,
		AttemptID:            attemptID,
		Caste:                caste,
		Objective:            task,
		Reason:               "worker-requested backup on the host/autopilot build lane",
		Workspace:            ws,
		IntentID:             attemptID,
		Permission:           codex.PermissionProfileForCaste(caste),
		Urgency:              recruitmentUrgencyRoutine,
		CostSlots:            costSlots,
		CostSeconds:          int(resolvedRecruitmentTimeout().Seconds()),
	}

	validation := validateRecruitmentIntent(intent)
	storedIntent := intent
	if validation.Allowed {
		storedIntent = sanitizedRecruitmentIntentCopy(intent)
	}

	createdAt := time.Now().UTC().Format(time.RFC3339)
	if _, err := recordRecruitmentIntent(recruitmentIntentRecord{Intent: storedIntent, CreatedAt: createdAt}); err != nil {
		return intent.IntentID, recruitmentDecisionResult{
			Allowed: false,
			Reason:  recruitmentReasonUnresolved,
			Detail:  fmt.Sprintf("could not durably record this recruitment intent (%v)", err),
		}
	}

	result := validation
	if result.Allowed {
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
		result = recruitmentDecisionResult{Allowed: admission.Allowed, Reason: admission.Reason, Detail: admission.Detail}
	}

	decidedAt := time.Now().UTC().Format(time.RFC3339)
	_, _ = recordRecruitmentDecision(intent.IntentID, result, decidedAt)

	return intent.IntentID, result
}
