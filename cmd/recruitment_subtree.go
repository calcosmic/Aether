package cmd

// BIO-06 (203-14-PLAN.md): the whole-subtree projection. Status, the live
// view, recovery, cost and depth all cover every governed descendant and
// every follow-on edge by reading the ONE existing spawn ledger
// (pkg/agent.SpawnTree) rather than a second tree structure -- exactly the
// trap this repository keeps falling into (see 203-09-SUMMARY.md's retired
// TypeScript budget counter). This file is that one projection, plus the
// end-of-run family tree and the closing "notes that changed a decision"
// list (the second half of D-08).

import (
	"fmt"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/colony"
)

// governedSubtreeRow is one row in the whole-subtree projection: an entry's
// own identity (name, caste, depth, status, parent), the dispatch mechanism
// that carried it (adapter kind, workspace) once known, its measured cost
// (read from the spend ledger, never estimated), and -- when this
// descendant is also the named follow-on consumer of a trophallaxis packet
// -- which child's result it is a follow-on step for. FollowOnFor is an
// ATTRIBUTE on this same row, never a second row: a descendant that is both
// a governed recruit and a follow-on consumer appears exactly once.
type governedSubtreeRow struct {
	Name        string
	ParentName  string
	Caste       string
	Depth       int
	Status      string
	AdapterKind string
	Workspace   string
	Task        string
	StartedAt   string
	FollowOnFor string
	CostFigure  string
}

// projectGovernedSubtreeParseCalls counts calls into agent.SpawnTree.Parse
// made BY projectGovernedSubtree, so a test can assert the ledger is parsed
// exactly once per call rather than once per descendant.
var projectGovernedSubtreeParseCalls int

// projectGovernedSubtree returns every descendant of root at any depth,
// each as its own governedSubtreeRow, read-only. root may be either a real
// spawn-tree agent name (in which case that entry's own row is included
// first, "itself") or one of the coordinator sentinels (spawnParentIsRoot)
// -- the coordinator has no spawn-tree entry of its own, so no "itself" row
// exists for it, and its children ARE the top of the projected subtree. A
// root that resolves to neither is refused by name.
//
// The tree is built by parsing agent.SpawnTree exactly once and walking the
// parsed slice in memory, mirroring spawnAncestorChain's own upward-walk
// discipline (cmd/spawn_ancestor.go) for this downward walk. A visited-name
// set stops a corrupted or hand-edited ledger with a self-referential
// parent from looping. phase scopes the cost lookup to one phase's spend
// ledgers (cmd/spend_ledger.go); pass <= 0 when no phase applies (every row
// then renders the dash sentinel).
func projectGovernedSubtree(root string, phase int) ([]governedSubtreeRow, error) {
	if store == nil {
		return nil, fmt.Errorf("no store initialized")
	}
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("projectGovernedSubtree requires a non-empty root")
	}

	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	entries, err := st.Parse()
	projectGovernedSubtreeParseCalls++
	if err != nil {
		return nil, err
	}

	// Last-write-wins de-dup, in file order -- mirrors spawnAncestorChain's
	// own byName construction (cmd/spawn_ancestor.go).
	byName := make(map[string]agent.SpawnEntry, len(entries))
	order := make([]string, 0, len(entries))
	for _, e := range entries {
		if _, exists := byName[e.AgentName]; !exists {
			order = append(order, e.AgentName)
		}
		byName[e.AgentName] = e
	}

	childrenByParent := make(map[string][]string, len(order))
	for _, name := range order {
		e := byName[name]
		childrenByParent[e.ParentName] = append(childrenByParent[e.ParentName], name)
	}

	followOnByReceiver := loadTrophallaxisFollowOnByReceiver()

	visited := make(map[string]bool, len(order))
	var rows []governedSubtreeRow

	var walk func(name string)
	walk = func(name string) {
		if visited[name] {
			// A corrupted or hand-edited ledger with a self-referential
			// parent must never loop forever here (T-173-29's own downward
			// analog).
			return
		}
		visited[name] = true
		entry, ok := byName[name]
		if !ok {
			return
		}
		rows = append(rows, buildGovernedSubtreeRow(entry, followOnByReceiver[name], phase))
		for _, child := range childrenByParent[name] {
			walk(child)
		}
	}

	switch _, rootIsEntry := byName[root]; {
	case rootIsEntry:
		// A real spawn-tree entry: its own row is "itself", included even
		// when it has no recruits at all (no error, no placeholder child).
		walk(root)
	case spawnParentIsRoot(root):
		// The coordinator sentinel has no spawn-tree entry of its own; its
		// children are the top of the projected subtree. A coordinator with
		// no recruits returns an EMPTY projection, not an error.
		for _, name := range childrenByParent[root] {
			walk(name)
		}
	default:
		return nil, fmt.Errorf("no recorded spawn entry for %q and not a coordinator sentinel", root)
	}

	sortGovernedSubtreeRows(rows)
	return rows, nil
}

// buildGovernedSubtreeRow assembles one row from its spawn-tree entry, its
// follow-on attribute (if any), and its measured cost. Read-only: it looks
// up the recruitment manifest, recruitment results and spend ledgers, and
// writes nothing.
func buildGovernedSubtreeRow(entry agent.SpawnEntry, followOnFor string, phase int) governedSubtreeRow {
	adapterKind, workspace := recruitmentAdapterAndWorkspaceForChild(entry.AgentName)
	return governedSubtreeRow{
		Name:        entry.AgentName,
		ParentName:  entry.ParentName,
		Caste:       entry.Caste,
		Depth:       entry.Depth,
		Status:      entry.Status,
		AdapterKind: adapterKind,
		Workspace:   workspace,
		Task:        entry.Task,
		StartedAt:   entry.Timestamp,
		FollowOnFor: followOnFor,
		CostFigure:  governedSubtreeCostFigure(entry.AgentName, phase),
	}
}

// sortGovernedSubtreeRows orders rows by depth ascending (a parent's depth
// is always strictly less than its own children's, by this codebase's own
// depth-assignment rule -- deriveSpawnDepth, cmd/spawn.go -- so this also
// keeps every row's own recruiter earlier in the list than the row itself),
// then by start time, then by worker identifier -- so rows sharing a depth
// and a start time come back in a fixed, worker-identifier order,
// identically across repeated reads.
func sortGovernedSubtreeRows(rows []governedSubtreeRow) {
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a.Depth != b.Depth {
			return a.Depth < b.Depth
		}
		if a.StartedAt != b.StartedAt {
			return a.StartedAt < b.StartedAt
		}
		return a.Name < b.Name
	})
}

// loadTrophallaxisFollowOnByReceiver reads recruitment/packets.json
// (read-only) and returns, for every receiver named as a packet's follow-on
// consumer, the ChildName of the recruitment result that packet carries --
// the value governedSubtreeRow.FollowOnFor names. An unreadable or absent
// packets file returns an empty map (no follow-on edges known), never an
// error -- a projection must still succeed when nothing has ever packed a
// follow-on handback.
func loadTrophallaxisFollowOnByReceiver() map[string]string {
	result := map[string]string{}
	if store == nil {
		return result
	}
	var file trophallaxisPacketsFile
	if err := store.LoadJSON(trophallaxisPacketsPath, &file); err != nil {
		return result
	}
	for _, packet := range file.Entries {
		if packet.ReceiverKind != trophallaxisReceiverFollowOn {
			continue
		}
		receiver := strings.TrimSpace(packet.Receiver)
		if receiver == "" {
			continue
		}
		if _, exists := result[receiver]; !exists {
			result[receiver] = packet.ChildName
		}
	}
	return result
}

// recruitmentAdapterAndWorkspaceForChild resolves a descendant's dispatch
// mechanism and workspace lease, read-only, from the two durable
// recruitment records that carry them: recruitment/manifest.json (Workspace,
// and AdapterKind once the manifest's "dispatched" transition records it)
// and, as a fallback for AdapterKind, recruitment/results.json (the bound
// result's own AdapterKind). Either or both may be empty for a descendant
// this projection did not admit through `aether recruit` at all (an
// ordinary dispatched worker) -- that is not an error, just unknown.
func recruitmentAdapterAndWorkspaceForChild(childName string) (adapterKind, workspace string) {
	childName = strings.TrimSpace(childName)
	if store == nil || childName == "" {
		return "", ""
	}
	var manifest recruitmentManifestFile
	if err := store.LoadJSON(recruitmentManifestPath, &manifest); err == nil {
		for i := len(manifest.Entries) - 1; i >= 0; i-- {
			if manifest.Entries[i].ChildName == childName {
				workspace = manifest.Entries[i].Workspace
				adapterKind = manifest.Entries[i].AdapterKind
				break
			}
		}
	}
	if adapterKind == "" {
		var results recruitmentResultsFile
		if err := store.LoadJSON(recruitmentResultsPath, &results); err == nil {
			for i := len(results.Entries) - 1; i >= 0; i-- {
				if results.Entries[i].ChildName == childName {
					adapterKind = results.Entries[i].AdapterKind
					break
				}
			}
		}
	}
	return adapterKind, workspace
}

// governedSubtreeCostFigure reads agentName's cost for phase from the spend
// ledger authority alone (loadSpendLedgersForPhase, cmd/spend_ledger.go),
// rendered through the existing cost formatting (spendCostLineFigure,
// cmd/spend_cost_line.go) -- never estimated. A phase with no ledger row for
// this branch renders the existing dash sentinel, exactly as the closing
// cost block already does for an unreported worker.
func governedSubtreeCostFigure(agentName string, phase int) string {
	if phase <= 0 {
		return spendNotReportedFigure
	}
	ledgers, ok := loadSpendLedgersForPhase(phase)
	if !ok {
		return spendNotReportedFigure
	}
	for _, row := range spendRowsAcross(ledgers) {
		if row.AgentName == agentName {
			return spendCostLineFigure(row)
		}
	}
	return spendNotReportedFigure
}

// recruitmentInlineCostFigure is governedSubtreeCostFigure scoped to the
// colony's own current phase (read-only, via readColonyStateWithoutWriting)
// -- the figure the inline "a recruit joined" line (cmd/codex_visuals.go)
// prints as "cost so far". A colony with no readable state renders the dash
// sentinel, matching every other "cost not known" case in this file.
func recruitmentInlineCostFigure(agentName string) string {
	state, ok := readColonyStateWithoutWriting()
	if !ok {
		return spendNotReportedFigure
	}
	return governedSubtreeCostFigure(agentName, state.CurrentPhase)
}

// renderGovernedSubtree renders rows as the "Governed Subtree" section
// (used by `aether status`, BIO-06's must_have that status covers the whole
// governed subtree for the current phase). Returns "" for an empty
// projection -- an empty heading is filler this codebase's own greeting
// card rule already forbids.
func renderGovernedSubtree(rows []governedSubtreeRow) string {
	if len(rows) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(voiceLine("family", "Governed Subtree (recruited helpers and their branches)") + "\n")
	for _, row := range rows {
		b.WriteString(renderGovernedSubtreeRowLine(row))
	}
	return b.String()
}

// renderGovernedSubtreeRowLine renders one row: caste identity (the same
// emoji/colour/name rendering every other worker in the colony's voice
// already uses), name, depth, status, who recruited it, and its cost. A
// follow-on descendant's line also names which child's result it is a
// follow-on step for.
func renderGovernedSubtreeRowLine(row governedSubtreeRow) string {
	parent := strings.TrimSpace(row.ParentName)
	if parent == "" {
		parent = "no recorded parent"
	}
	text := fmt.Sprintf("%s %s (depth %d, %s) -- recruited by %s, cost %s",
		casteIdentity(row.Caste), row.Name, row.Depth, emptyFallback(row.Status, "active"), parent, row.CostFigure)
	if followOn := strings.TrimSpace(row.FollowOnFor); followOn != "" {
		text += fmt.Sprintf(" -- follow-on for %s's result", followOn)
	}
	return voiceLine("family", text) + "\n"
}

// renderGovernedSubtreeStatusSection renders `aether status`'s own BIO-06
// must_have: the governed subtree for the current phase, covering every
// coordinator sentinel's own recruits (spawnRootParentNames, cmd/spawn.go)
// rather than a single named root, since a status inspection has no one
// worker in mind. Returns "" (no section) for a colony with no recruits
// under any sentinel for the current phase.
func renderGovernedSubtreeStatusSection(state colony.ColonyState) string {
	if store == nil {
		return ""
	}
	// Scope to the CURRENT run. projectGovernedSubtree walks the whole
	// spawn tree across every run, which is right for callers that want the
	// full history — but the status screen must not present a worker from a
	// finished run as if it were live. TestStatusPrefersCurrentRunWorkersOverStaleHistory
	// caught exactly that: the family tree added here showed Ghost-41, a
	// worker from an older run, beside the current one.
	//
	// No current run means no filter rather than an empty screen: a colony
	// with history but nothing running should still show what it has.
	currentRunNames := map[string]bool{}
	tree := agent.NewSpawnTree(store, "spawn-tree.txt")
	if run, ok, runErr := tree.CurrentRun(); runErr == nil && ok {
		if entries, entErr := tree.EntriesForRun(run.ID); entErr == nil {
			for _, e := range entries {
				currentRunNames[e.AgentName] = true
			}
		}
	}

	seen := map[string]bool{}
	var rows []governedSubtreeRow
	for _, root := range spawnRootParentNames {
		subtree, err := projectGovernedSubtree(root, state.CurrentPhase)
		if err != nil {
			continue
		}
		for _, row := range subtree {
			if len(currentRunNames) > 0 && !currentRunNames[row.Name] {
				continue
			}
			if seen[row.Name] {
				continue
			}
			seen[row.Name] = true
			rows = append(rows, row)
		}
	}
	sortGovernedSubtreeRows(rows)
	return renderGovernedSubtree(rows)
}

// --- Task 3: the end-of-run family tree, and the closing decision-changed list ---

// recruitmentRefusalRow is one refused recruitment attempt, for the
// end-of-run family tree's refusal list.
type recruitmentRefusalRow struct {
	Caste       string
	ReasonClass string
	Detail      string
}

// recruitedChildNameSet reads recruitment/manifest.json (read-only) and
// returns the set of every child name `aether recruit` has ever admitted --
// the signal that distinguishes a governed recruit from an ordinary
// dispatched team member sharing the same spawn ledger.
func recruitedChildNameSet() map[string]bool {
	set := map[string]bool{}
	if store == nil {
		return set
	}
	var manifest recruitmentManifestFile
	if err := store.LoadJSON(recruitmentManifestPath, &manifest); err == nil {
		for _, e := range manifest.Entries {
			if name := strings.TrimSpace(e.ChildName); name != "" {
				set[name] = true
			}
		}
	}
	return set
}

// recruitedChildNamesForRun returns the names, among runID's own spawn-tree
// entries (agent.SpawnTree.EntriesForRun), that are governed recruits
// (recruitedChildNameSet). Read-only.
func recruitedChildNamesForRun(runID string) []string {
	runID = strings.TrimSpace(runID)
	if store == nil || runID == "" {
		return nil
	}
	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	entries, err := st.EntriesForRun(runID)
	if err != nil {
		return nil
	}
	recruits := recruitedChildNameSet()
	var names []string
	for _, e := range entries {
		if recruits[e.AgentName] {
			names = append(names, e.AgentName)
		}
	}
	return names
}

// recruitmentRefusalsForRun reads recruitment/intents.json (read-only) and
// returns every refused intent whose ParentRunID names runID, in the file's
// own recorded (append) order.
func recruitmentRefusalsForRun(runID string) []recruitmentRefusalRow {
	runID = strings.TrimSpace(runID)
	if store == nil || runID == "" {
		return nil
	}
	var file recruitmentIntentsFile
	if err := store.LoadJSON(recruitmentIntentsPath, &file); err != nil {
		return nil
	}
	var refusals []recruitmentRefusalRow
	for _, rec := range file.Entries {
		if rec.Intent.ParentRunID != runID {
			continue
		}
		if rec.Decision == nil || rec.Decision.Allowed {
			continue
		}
		refusals = append(refusals, recruitmentRefusalRow{
			Caste:       rec.Intent.Caste,
			ReasonClass: rec.Decision.Reason,
			Detail:      rec.Decision.Detail,
		})
	}
	return refusals
}

// renderRecruitmentFamilyTree renders the closing family tree for the
// current spawn run (agent.SpawnTree.CurrentRun): every recruited
// descendant, its recruiter, what it cost and its status, plus every
// refusal recorded during the same run. Returns "" (no section, not an
// empty heading) when the run recruited nothing and refused nothing.
//
// No subtree reader here rewrites anything: every read goes through
// projectGovernedSubtree, agent.SpawnTree.Parse/EntriesForRun/CurrentRun, or
// a plain store.LoadJSON -- none of which ever calls
// store.UpdateJSONAtomically, store.AtomicWrite, or SpawnTree.RecordSpawn.
func renderRecruitmentFamilyTree(phase int) string {
	if store == nil {
		return ""
	}
	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	run, ok, err := st.CurrentRun()
	if err != nil || !ok {
		return ""
	}

	recruitNames := recruitedChildNamesForRun(run.ID)
	refusals := recruitmentRefusalsForRun(run.ID)
	if len(recruitNames) == 0 && len(refusals) == 0 {
		return ""
	}

	entries, _ := st.Parse()
	byName := make(map[string]agent.SpawnEntry, len(entries))
	for _, e := range entries {
		byName[e.AgentName] = e
	}

	roots := map[string]bool{}
	for _, name := range recruitNames {
		if e, ok := byName[name]; ok {
			roots[e.ParentName] = true
		}
	}

	seen := map[string]bool{}
	var rows []governedSubtreeRow
	for root := range roots {
		subtree, err := projectGovernedSubtree(root, phase)
		if err != nil {
			continue
		}
		for _, row := range subtree {
			if seen[row.Name] {
				continue
			}
			seen[row.Name] = true
			rows = append(rows, row)
		}
	}
	sortGovernedSubtreeRows(rows)

	var b strings.Builder
	b.WriteString(voiceLine("family", "The Family Tree (who recruited whom)") + "\n")
	for _, row := range rows {
		b.WriteString(renderGovernedSubtreeRowLine(row))
	}
	for _, refusal := range refusals {
		b.WriteString(renderRecruitmentRefusalLine(refusal))
	}
	return b.String()
}

// renderRecruitmentRefusalLine renders one refused recruitment attempt,
// naming the caste, the reason class and a plain sentence that the worker
// carried on alone (D-06).
func renderRecruitmentRefusalLine(refusal recruitmentRefusalRow) string {
	text := fmt.Sprintf("%s recruitment refused (%s)", casteLabel(refusal.Caste), emptyFallback(refusal.ReasonClass, "refused"))
	if detail := strings.TrimSpace(refusal.Detail); detail != "" {
		text += ": " + detail
	}
	text += " -- the worker carried on alone"
	return voiceLine("refusal", text) + "\n"
}

// renderNotesThatChangedDecisions renders D-08's closing list: every
// recorded note contribution (recruitmentContributionNote) that changed a
// decision, sourced from the same credit records (cmd/recruitment_credit.go)
// the inline decision-changed line reads. Omitted entirely (returns "")
// when no note has changed a decision.
func renderNotesThatChangedDecisions() string {
	records, err := recruitmentCreditAll()
	if err != nil {
		return ""
	}
	var changed []recruitmentCreditRecord
	for _, r := range records {
		if r.ContributionKind == recruitmentContributionNote && strings.TrimSpace(r.ChangedDecisionID) != "" {
			changed = append(changed, r)
		}
	}
	if len(changed) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(voiceLine("decision", "Notes That Changed A Decision") + "\n")
	for i := range changed {
		r := changed[i]
		outcome := renderRecruitmentCreditOutcome(&r)
		b.WriteString(voiceLine("decision", fmt.Sprintf("Note %s changed decision %s -- %s", r.ContributionID, r.ChangedDecisionID, outcome)) + "\n")
	}
	return b.String()
}
