package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

type reconcileReport struct {
	Status      string                    `json:"status"`
	Root        string                    `json:"root"`
	GeneratedAt string                    `json:"generated_at"`
	Git         reconcileGitSummary       `json:"git"`
	Versions    reconcileVersionSummary   `json:"versions"`
	State       reconcileStateSummary     `json:"state"`
	Session     reconcileSessionSummary   `json:"session"`
	Flags       reconcileFlagSummary      `json:"flags"`
	Pheromones  reconcilePheromoneSummary `json:"pheromones"`
	Inventory   []reconcileInventoryEntry `json:"inventory"`
	Findings    []reconcileFinding        `json:"findings"`
	Recovery    []reconcileRecoveryAction `json:"recovery"`
}

type reconcileGitSummary struct {
	Branch     string `json:"branch,omitempty"`
	Head       string `json:"head,omitempty"`
	Upstream   string `json:"upstream,omitempty"`
	Divergence string `json:"divergence,omitempty"`
	Describe   string `json:"describe,omitempty"`
	Dirty      bool   `json:"dirty"`
}

type reconcileVersionSummary struct {
	Source   string `json:"source,omitempty"`
	NPM      string `json:"npm,omitempty"`
	Resolved string `json:"resolved,omitempty"`
	Docs     string `json:"docs,omitempty"`
	Tag      string `json:"tag,omitempty"`
	Aligned  bool   `json:"aligned"`
}

type reconcileStateSummary struct {
	Present          bool   `json:"present"`
	Valid            bool   `json:"valid"`
	Goal             string `json:"goal,omitempty"`
	State            string `json:"state,omitempty"`
	CurrentPhase     int    `json:"current_phase,omitempty"`
	PlanPhases       int    `json:"plan_phases,omitempty"`
	InitializedAt    string `json:"initialized_at,omitempty"`
	PlanGeneratedAt  string `json:"plan_generated_at,omitempty"`
	CachePresent     bool   `json:"cache_present"`
	BackupCandidates int    `json:"backup_candidates"`
}

type reconcileSessionSummary struct {
	Present        bool   `json:"present"`
	Valid          bool   `json:"valid"`
	SessionID      string `json:"session_id,omitempty"`
	Goal           string `json:"goal,omitempty"`
	LastCommand    string `json:"last_command,omitempty"`
	LastCommandAt  string `json:"last_command_at,omitempty"`
	CurrentPhase   int    `json:"current_phase,omitempty"`
	SuggestedNext  string `json:"suggested_next,omitempty"`
	BaselineCommit string `json:"baseline_commit,omitempty"`
	AgeDays        int    `json:"age_days,omitempty"`
}

type reconcileFlagSummary struct {
	Present    bool `json:"present"`
	Valid      bool `json:"valid"`
	Total      int  `json:"total"`
	Unresolved int  `json:"unresolved"`
	Blockers   int  `json:"blockers"`
	Issues     int  `json:"issues"`
	Notes      int  `json:"notes"`
}

type reconcilePheromoneSummary struct {
	Present       bool `json:"present"`
	Valid         bool `json:"valid"`
	Total         int  `json:"total"`
	Active        int  `json:"active"`
	ExpiredActive int  `json:"expired_active"`
}

type reconcileInventoryEntry struct {
	Path          string `json:"path"`
	Category      string `json:"category"`
	Authority     string `json:"authority"`
	RecoveryClass string `json:"recovery_class"`
	Present       bool   `json:"present"`
}

type reconcileFinding struct {
	Severity       string `json:"severity"`
	Category       string `json:"category"`
	File           string `json:"file,omitempty"`
	Message        string `json:"message"`
	Recommendation string `json:"recommendation,omitempty"`
}

type reconcileRecoveryAction struct {
	Step    int    `json:"step"`
	Safety  string `json:"safety"`
	Command string `json:"command,omitempty"`
	Reason  string `json:"reason"`
}

var reconcileCmd = &cobra.Command{
	Use:   "reconcile",
	Short: "Read-only repository and colony state reconciliation report",
	Long: `Inspect repository, version, generated state, flags and signals without
mutating .aether/data. This command is safe to run when COLONY_STATE.json is
missing or stale because it does not require an active colony.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOut, _ := cmd.Flags().GetBool("json")
		report := buildReconcileReport(resolveAetherRoot(), time.Now().UTC())
		if jsonOut || !shouldRenderVisualOutput(stdout) {
			outputOK(report)
			return nil
		}
		outputWorkflow(report, renderReconcileVisual(report))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(reconcileCmd)
	reconcileCmd.Flags().Bool("json", false, "output structured JSON")
}

func buildReconcileReport(root string, now time.Time) reconcileReport {
	root = strings.TrimSpace(root)
	if root == "" {
		root = "."
	}
	dataDir := filepath.Join(root, ".aether", "data")
	report := reconcileReport{
		Status:      "healthy",
		Root:        root,
		GeneratedAt: now.Format(time.RFC3339),
		Git:         inspectReconcileGit(root),
		Versions:    inspectReconcileVersions(root),
		Inventory:   buildReconcileInventory(dataDir),
	}

	inspectReconcileVersionFindings(&report)
	report.State = inspectReconcileState(dataDir, now, &report)
	report.Session = inspectReconcileSession(dataDir, now, report.Git.Head, report.State, &report)
	report.Flags = inspectReconcileFlags(dataDir, &report)
	report.Pheromones = inspectReconcilePheromones(dataDir, now, &report)
	inspectReconcilePlanningDocs(root, report.Versions.Source, &report)
	inspectReconcileStaleRuntimeFiles(dataDir, now, &report)
	report.Recovery = buildReconcileRecovery(report)
	report.Status = reconcileStatus(report.Findings)
	return report
}

func inspectReconcileGit(root string) reconcileGitSummary {
	summary := reconcileGitSummary{
		Branch:     gitOutput(root, "branch", "--show-current"),
		Head:       gitOutput(root, "rev-parse", "HEAD"),
		Describe:   gitOutput(root, "describe", "--tags", "--always", "--dirty"),
		Divergence: gitOutput(root, "status", "-sb"),
	}
	if summary.Divergence != "" {
		lines := strings.Split(summary.Divergence, "\n")
		if len(lines) > 0 {
			summary.Upstream = strings.TrimSpace(strings.TrimPrefix(lines[0], "## "))
		}
	}
	summary.Dirty = strings.Contains(summary.Describe, "-dirty") ||
		strings.Contains(summary.Divergence, "\n M ") ||
		strings.Contains(summary.Divergence, "\nM ") ||
		strings.Contains(summary.Divergence, "\n?? ")
	return summary
}

func inspectReconcileVersions(root string) reconcileVersionSummary {
	source := readRepoVersion(root)
	npm := readNpmPackageVersion(root)
	resolved := resolveVersion(root)
	tag := strings.TrimPrefix(gitOutput(root, "describe", "--tags", "--abbrev=0"), "v")
	docs := firstDocVersion(root)
	aligned := source != "" && npm != "" && resolved != "" && source == npm && source == resolved
	if docs != "" && source != "" && docs != source {
		aligned = false
	}
	return reconcileVersionSummary{
		Source:   source,
		NPM:      npm,
		Resolved: resolved,
		Docs:     docs,
		Tag:      tag,
		Aligned:  aligned,
	}
}

func inspectReconcileVersionFindings(report *reconcileReport) {
	v := report.Versions
	if v.Source == "" {
		addReconcileFinding(report, "warning", "version", ".aether/version.json", "source version is missing", "Restore .aether/version.json before publish or release checks.")
	}
	if v.NPM == "" {
		addReconcileFinding(report, "info", "version", "npm/package.json", "npm package version is missing", "Only release packaging needs npm version agreement.")
	}
	if v.Source != "" && v.NPM != "" && v.Source != v.NPM {
		addReconcileFinding(report, "warning", "version", "npm/package.json", fmt.Sprintf("npm package version %s does not match source version %s", v.NPM, v.Source), "Align npm/package.json with .aether/version.json before release.")
	}
	if v.Source != "" && v.Resolved != "" && v.Source != v.Resolved {
		addReconcileFinding(report, "warning", "version", "aether version", fmt.Sprintf("resolved CLI version %s does not match source version %s", v.Resolved, v.Source), "Rebuild or reinstall the binary from the source checkout before release.")
	}
}

func inspectReconcileState(dataDir string, now time.Time, report *reconcileReport) reconcileStateSummary {
	path := filepath.Join(dataDir, "COLONY_STATE.json")
	summary := reconcileStateSummary{
		CachePresent:     reconcileFileExists(filepath.Join(dataDir, ".cache_COLONY_STATE.json")),
		BackupCandidates: countStateBackups(dataDir),
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			addReconcileFinding(report, "critical", "state", "COLONY_STATE.json", "COLONY_STATE.json is missing", "Inspect available cache/backups before running lifecycle commands.")
			return summary
		}
		addReconcileFinding(report, "critical", "state", "COLONY_STATE.json", fmt.Sprintf("COLONY_STATE.json could not be read: %v", err), "Fix file permissions or recover from a known-good backup.")
		return summary
	}
	summary.Present = true
	if !json.Valid(data) {
		addReconcileFinding(report, "critical", "state", "COLONY_STATE.json", "COLONY_STATE.json contains invalid JSON", "Recover from cache/backup; do not edit live state by hand unless no finalizer can recover it.")
		return summary
	}
	var state colony.ColonyState
	if err := json.Unmarshal(data, &state); err != nil {
		addReconcileFinding(report, "critical", "state", "COLONY_STATE.json", fmt.Sprintf("COLONY_STATE.json schema could not be decoded: %v", err), "Run a schema-aware migration before resuming.")
		return summary
	}
	summary.Valid = true
	summary.State = string(state.State)
	summary.CurrentPhase = state.CurrentPhase
	summary.PlanPhases = len(state.Plan.Phases)
	if state.Goal != nil {
		summary.Goal = *state.Goal
	}
	if state.InitializedAt != nil {
		summary.InitializedAt = state.InitializedAt.Format(time.RFC3339)
		if now.Sub(*state.InitializedAt) > 30*24*time.Hour {
			addReconcileFinding(report, "warning", "state", "COLONY_STATE.json", "COLONY_STATE.json is older than 30 days", "Treat lifecycle state as stale until reconciled with Git and planning docs.")
		}
	}
	if state.Plan.GeneratedAt != nil {
		summary.PlanGeneratedAt = state.Plan.GeneratedAt.Format(time.RFC3339)
	}
	if summary.Goal == "" {
		addReconcileFinding(report, "critical", "state", "COLONY_STATE.json", "COLONY_STATE.json has no goal", "Recover from backup or initialize a new colony after archiving generated state.")
	}
	if summary.PlanPhases == 0 && state.State != colony.StateIDLE {
		addReconcileFinding(report, "warning", "state", "COLONY_STATE.json", "COLONY_STATE.json has no plan phases", "Run planning only after state is reconciled and provider-backed planning is available.")
	}
	return summary
}

func inspectReconcileSession(dataDir string, now time.Time, head string, state reconcileStateSummary, report *reconcileReport) reconcileSessionSummary {
	path := filepath.Join(dataDir, "session.json")
	var summary reconcileSessionSummary
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			addReconcileFinding(report, "warning", "session", "session.json", fmt.Sprintf("session.json could not be read: %v", err), "Do not resume until session state can be inspected.")
		}
		return summary
	}
	summary.Present = true
	if !json.Valid(data) {
		addReconcileFinding(report, "critical", "session", "session.json", "session.json contains invalid JSON", "Archive the broken session and recover from handoff or colony state.")
		return summary
	}
	var session colony.SessionFile
	if err := json.Unmarshal(data, &session); err != nil {
		addReconcileFinding(report, "critical", "session", "session.json", fmt.Sprintf("session.json schema could not be decoded: %v", err), "Archive or migrate the session before resume.")
		return summary
	}
	summary.Valid = true
	summary.SessionID = session.SessionID
	summary.Goal = session.ColonyGoal
	summary.LastCommand = session.LastCommand
	summary.LastCommandAt = session.LastCommandAt
	summary.CurrentPhase = session.CurrentPhase
	summary.SuggestedNext = session.SuggestedNext
	summary.BaselineCommit = session.BaselineCommit
	if state.Valid {
		if summary.Goal != "" && state.Goal != "" && summary.Goal != state.Goal {
			addReconcileFinding(report, "warning", "session", "session.json", "session goal does not match COLONY_STATE.json goal", "Prefer generated-state reconciliation over direct resume.")
		}
		if summary.CurrentPhase != 0 && state.CurrentPhase != 0 && summary.CurrentPhase != state.CurrentPhase {
			addReconcileFinding(report, "warning", "session", "session.json", "session current_phase does not match COLONY_STATE.json", "Resume should choose the latest validated lifecycle transition, not the stale session value.")
		}
	} else if summary.Goal != "" {
		addReconcileFinding(report, "warning", "session", "session.json", "session.json exists but COLONY_STATE.json is unavailable", "Use session only as recovery evidence; it is not authoritative by itself.")
	}
	if head != "" && summary.BaselineCommit != "" && summary.BaselineCommit != head {
		addReconcileFinding(report, "warning", "session", "session.json", "session baseline_commit differs from current HEAD", "Revalidate evidence before resume or continue.")
	}
	if last := parseTimestamp(summary.LastCommandAt); !last.IsZero() {
		summary.AgeDays = int(now.Sub(last).Hours() / 24)
		if summary.AgeDays > 30 {
			addReconcileFinding(report, "warning", "session", "session.json", fmt.Sprintf("session is stale by %d days", summary.AgeDays), "Run reconciliation before resume; do not trust suggested_next blindly.")
		}
	}
	return summary
}

func inspectReconcileFlags(dataDir string, report *reconcileReport) reconcileFlagSummary {
	path := filepath.Join(dataDir, "pending-decisions.json")
	var summary reconcileFlagSummary
	data, err := os.ReadFile(path)
	if err != nil {
		return summary
	}
	summary.Present = true
	if !json.Valid(data) {
		addReconcileFinding(report, "critical", "flags", "pending-decisions.json", "pending-decisions.json contains invalid JSON", "Repair or archive flags before relying on blocker checks.")
		return summary
	}
	var flags colony.FlagsFile
	if err := json.Unmarshal(data, &flags); err != nil {
		addReconcileFinding(report, "warning", "flags", "pending-decisions.json", fmt.Sprintf("pending-decisions.json schema could not be decoded: %v", err), "Migrate flags before using them for lifecycle blocking.")
		return summary
	}
	summary.Valid = true
	summary.Total = len(flags.Decisions)
	for _, flag := range flags.Decisions {
		if flag.Resolved {
			continue
		}
		summary.Unresolved++
		switch strings.ToLower(flag.Type) {
		case "blocker":
			summary.Blockers++
		case "issue":
			summary.Issues++
		default:
			summary.Notes++
		}
	}
	if summary.Unresolved > 0 {
		addReconcileFinding(report, "warning", "flags", "pending-decisions.json", fmt.Sprintf("%d unresolved flag(s) found (%d blocker(s), %d issue(s))", summary.Unresolved, summary.Blockers, summary.Issues), "Review flags before dogfood; resolve placeholders through normal flag commands.")
	}
	return summary
}

func inspectReconcilePheromones(dataDir string, now time.Time, report *reconcileReport) reconcilePheromoneSummary {
	path := filepath.Join(dataDir, "pheromones.json")
	var summary reconcilePheromoneSummary
	data, err := os.ReadFile(path)
	if err != nil {
		return summary
	}
	summary.Present = true
	if !json.Valid(data) {
		addReconcileFinding(report, "critical", "pheromones", "pheromones.json", "pheromones.json contains invalid JSON", "Recover signal state before injecting worker context.")
		return summary
	}
	var pf colony.PheromoneFile
	if err := json.Unmarshal(data, &pf); err != nil {
		addReconcileFinding(report, "warning", "pheromones", "pheromones.json", fmt.Sprintf("pheromones.json schema could not be decoded: %v", err), "Migrate pheromone state before worker prompt injection.")
		return summary
	}
	summary.Valid = true
	summary.Total = len(pf.Signals)
	for _, sig := range pf.Signals {
		if sig.Active {
			summary.Active++
		}
		if sig.Active && sig.ExpiresAt != nil {
			if expiry := parseTimestamp(*sig.ExpiresAt); !expiry.IsZero() && now.After(expiry) {
				summary.ExpiredActive++
			}
		}
	}
	if summary.ExpiredActive > 0 {
		addReconcileFinding(report, "warning", "pheromones", "pheromones.json", fmt.Sprintf("%d active pheromone signal(s) are expired", summary.ExpiredActive), "Run a Go-owned housekeeping command before injecting signals into workers.")
	}
	return summary
}

func inspectReconcilePlanningDocs(root, sourceVersion string, report *reconcileReport) {
	if !reconcileFileExists(filepath.Join(root, ".planning", "REQUIREMENTS.md")) {
		addReconcileFinding(report, "info", "planning_docs", ".planning/REQUIREMENTS.md", ".planning/REQUIREMENTS.md is missing", "Create the next milestone requirements doc before starting v1.25 implementation tracking.")
	}
	if reconcileFileExists(filepath.Join(root, ".planning", "v1.24-MILESTONE-AUDIT.md")) && !reconcileFileExists(filepath.Join(root, ".planning", "milestones", "v1.24-MILESTONE-AUDIT.md")) {
		addReconcileFinding(report, "info", "planning_docs", ".planning/MILESTONES.md", "v1.24 audit exists outside the linked milestones path", "Fix the documentation link or move the archived audit in a documentation-only change.")
	}
	docVersion := firstDocVersion(root)
	if sourceVersion != "" && docVersion != "" && docVersion != sourceVersion {
		addReconcileFinding(report, "info", "version", ".planning", fmt.Sprintf("planning docs mention product version %s while source version is %s", docVersion, sourceVersion), "Align planning docs during release preparation, not during state recovery.")
	}
}

func inspectReconcileStaleRuntimeFiles(dataDir string, now time.Time, report *reconcileReport) {
	for _, rel := range []string{"worker-processes.json", "spawn-runs.json", "last-build-claims.json"} {
		path := filepath.Join(dataDir, rel)
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if now.Sub(info.ModTime()) > 24*time.Hour {
			addReconcileFinding(report, "warning", "runtime_state", rel, fmt.Sprintf("%s is older than 24 hours", rel), "Treat as stale evidence; do not let it satisfy a new run.")
		}
	}
}

func buildReconcileInventory(dataDir string) []reconcileInventoryEntry {
	specs := []reconcileInventoryEntry{
		{Path: "COLONY_STATE.json", Category: "protected_state", Authority: "go_finalizer", RecoveryClass: "recover_or_migrate"},
		{Path: ".cache_COLONY_STATE.json", Category: "state_cache", Authority: "generated", RecoveryClass: "candidate"},
		{Path: "COLONY_STATE.json.bak-plan", Category: "state_backup", Authority: "generated", RecoveryClass: "candidate"},
		{Path: "session.json", Category: "session_state", Authority: "go_runtime", RecoveryClass: "evidence"},
		{Path: "pending-decisions.json", Category: "flags", Authority: "go_runtime", RecoveryClass: "preserve"},
		{Path: "pheromones.json", Category: "signals", Authority: "go_runtime", RecoveryClass: "housekeep"},
		{Path: "spawn-runs.json", Category: "runtime_evidence", Authority: "go_runtime", RecoveryClass: "stale_if_old"},
		{Path: "worker-processes.json", Category: "runtime_evidence", Authority: "go_runtime", RecoveryClass: "stale_if_old"},
		{Path: "last-build-claims.json", Category: "runtime_evidence", Authority: "go_runtime", RecoveryClass: "stale_if_old"},
		{Path: "event-bus.jsonl", Category: "events", Authority: "go_runtime", RecoveryClass: "preserve"},
	}
	for i := range specs {
		specs[i].Present = reconcileFileExists(filepath.Join(dataDir, specs[i].Path))
	}
	return specs
}

func buildReconcileRecovery(report reconcileReport) []reconcileRecoveryAction {
	var actions []reconcileRecoveryAction
	addAction := func(safety, command, reason string) {
		actions = append(actions, reconcileRecoveryAction{
			Step:    len(actions) + 1,
			Safety:  safety,
			Command: command,
			Reason:  reason,
		})
	}
	addAction("read_only", "aether reconcile --json", "Re-run the reconciliation report after any manual decision or approved repair.")
	if !report.State.Present {
		addAction("read_only", "", "Inspect .aether/data/.cache_COLONY_STATE.json, COLONY_STATE.json.bak-plan, and .aether/data/backups before restoring state.")
	}
	if report.Session.Present && (!report.State.Present || report.Session.BaselineCommit != "" && report.Git.Head != "" && report.Session.BaselineCommit != report.Git.Head) {
		addAction("read_only", "", "Treat session.json as recovery evidence only until it matches current state and Git HEAD.")
	}
	if report.Pheromones.ExpiredActive > 0 {
		addAction("go_owned", "aether pheromones", "Review active but expired signals before worker prompt injection.")
	}
	if report.Flags.Unresolved > 0 {
		addAction("go_owned", "aether flags --status active", "Review unresolved flags before dogfood or release work.")
	}
	if !report.Versions.Aligned {
		addAction("release_only", "aether version --check", "Verify version alignment before publish or stable promotion.")
	}
	return actions
}

func addReconcileFinding(report *reconcileReport, severity, category, file, message, recommendation string) {
	report.Findings = append(report.Findings, reconcileFinding{
		Severity:       severity,
		Category:       category,
		File:           file,
		Message:        message,
		Recommendation: recommendation,
	})
}

func reconcileStatus(findings []reconcileFinding) string {
	status := "healthy"
	for _, finding := range findings {
		switch finding.Severity {
		case "critical":
			return "action_required"
		case "warning":
			status = "warning"
		}
	}
	return status
}

func gitOutput(root string, args ...string) string {
	allArgs := append([]string{"-C", root}, args...)
	out, err := exec.Command("git", allArgs...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func firstDocVersion(root string) string {
	re := regexp.MustCompile(`v?([0-9]+\.[0-9]+\.[0-9]+)`)
	for _, rel := range []string{
		filepath.Join(".planning", "STATE.md"),
		filepath.Join(".planning", "PROJECT.md"),
		filepath.Join(".planning", "ROADMAP.md"),
	} {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			continue
		}
		if match := re.FindStringSubmatch(string(data)); len(match) == 2 {
			return normalizeVersion(match[1])
		}
	}
	return ""
}

func reconcileFileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func countStateBackups(dataDir string) int {
	patterns := []string{
		filepath.Join(dataDir, "COLONY_STATE.json.bak-*"),
		filepath.Join(dataDir, "backups", "COLONY_STATE*.bak"),
	}
	count := 0
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		count += len(matches)
	}
	return count
}

func renderReconcileVisual(report reconcileReport) string {
	var b strings.Builder
	b.WriteString(renderBanner("reconcile", "Reconciliation Report"))
	b.WriteString(visualDividerStr())
	b.WriteString(fmt.Sprintf("Status: %s\n", report.Status))
	if report.Git.Describe != "" {
		b.WriteString(fmt.Sprintf("Git:    %s\n", report.Git.Describe))
	}
	if report.Versions.Source != "" {
		b.WriteString(fmt.Sprintf("Version: source=%s npm=%s resolved=%s\n", report.Versions.Source, report.Versions.NPM, report.Versions.Resolved))
	}
	b.WriteString(fmt.Sprintf("State:  present=%t valid=%t goal=%s phase=%d\n", report.State.Present, report.State.Valid, report.State.Goal, report.State.CurrentPhase))
	if len(report.Findings) > 0 {
		b.WriteString("\nFindings:\n")
		for _, finding := range report.Findings {
			b.WriteString(fmt.Sprintf("- [%s] %s: %s\n", strings.ToUpper(finding.Severity), finding.Category, finding.Message))
		}
	}
	if len(report.Recovery) > 0 {
		b.WriteString("\nNext safe actions:\n")
		sort.SliceStable(report.Recovery, func(i, j int) bool { return report.Recovery[i].Step < report.Recovery[j].Step })
		for _, action := range report.Recovery {
			if action.Command != "" {
				b.WriteString(fmt.Sprintf("%d. %s - %s\n", action.Step, action.Command, action.Reason))
			} else {
				b.WriteString(fmt.Sprintf("%d. %s\n", action.Step, action.Reason))
			}
		}
	}
	return b.String()
}
