package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

var newQuickWorkerInvoker = codex.NewWorkerInvoker

var maturityCmd = &cobra.Command{
	Use:         "maturity",
	Short:       "Show evidence-backed colony health and readiness",
	Args:        cobra.NoArgs,
	Annotations: map[string]string{"aether.io/read-only": "true"},
	RunE: func(cmd *cobra.Command, args []string) error {
		root := resolveAetherRoot()
		now := time.Now().UTC()
		facts, err := loadLifecycleFacts(root, store, now)
		if err != nil {
			facts = unavailableLifecycleFacts(root, now, err.Error())
		}
		projection := projectLifecycle(facts, LifecycleViewFocused, detectPlatform())
		projection.Command = "maturity"
		result := projectLifecycleHealth(facts, projection)
		outputWorkflow(result, renderLifecycleHealth(result, lifecycleStatusOutputWidth()))
		return nil
	},
}

var quickCmd = &cobra.Command{
	Use:   "quick [question]",
	Short: "Run a lightweight Scout query without build ceremony",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		question := strings.TrimSpace(strings.Join(args, " "))
		if question == "" {
			outputError(1, `usage: aether quick "question"`, nil)
			return nil
		}
		timeout, _ := cmd.Flags().GetDuration("timeout")
		if timeout <= 0 {
			timeout = codex.DefaultWorkerTimeout
		}
		result, err := runQuickScout(question, timeout)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		outputWorkflow(result, renderQuickVisual(result))
		return nil
	},
}

var bumpVersionCmd = &cobra.Command{
	Use:   "bump-version <semver>",
	Short: "Bump Aether source and npm package versions",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		root := resolveAetherRoot()
		result, err := runBumpVersion(root, args[0], dryRun)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		outputWorkflow(result, renderBumpVersionVisual(result))
		return nil
	},
}

var migrateStateCmd = &cobra.Command{
	Use:   "migrate-state",
	Short: "Migrate COLONY_STATE.json to the current runtime schema",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		rollbackPath, _ := cmd.Flags().GetString("rollback")
		var result map[string]interface{}
		var err error
		if strings.TrimSpace(rollbackPath) != "" {
			result, err = runMigrateStateRollback(rollbackPath, dryRun)
		} else {
			result, err = runMigrateState(dryRun)
		}
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		outputWorkflow(result, renderMigrateStateVisual(result))
		return nil
	},
}

var verifyCastesCmd = &cobra.Command{
	Use:   "verify-castes",
	Short: "Verify colony caste surfaces and counts",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		result := verifyCasteSurfaces(resolveAetherRoot())
		outputWorkflow(result, renderVerifyCastesVisual(result))
		return nil
	},
}

func init() {
	quickCmd.Flags().Duration("timeout", codex.DefaultWorkerTimeout, "Scout worker timeout")
	bumpVersionCmd.Flags().Bool("dry-run", false, "Preview version updates without writing files")
	migrateStateCmd.Flags().Bool("dry-run", false, "Preview migration without writing COLONY_STATE.json")
	migrateStateCmd.Flags().String("rollback", "", "Restore an Aether-created pre-migration backup")

	rootCmd.AddCommand(maturityCmd)
	rootCmd.AddCommand(quickCmd)
	rootCmd.AddCommand(bumpVersionCmd)
	rootCmd.AddCommand(migrateStateCmd)
	rootCmd.AddCommand(verifyCastesCmd)
}

type LifecycleHealthState string

const (
	LifecycleHealthVerified    LifecycleHealthState = "verified"
	LifecycleHealthBlocked     LifecycleHealthState = "blocked"
	LifecycleHealthDegraded    LifecycleHealthState = "degraded"
	LifecycleHealthUnavailable LifecycleHealthState = "unavailable"
	LifecycleHealthUnknown     LifecycleHealthState = "unknown"
)

type LifecycleHealthAssessment struct {
	Status      LifecycleHealthState `json:"status"`
	EvidenceIDs []string             `json:"evidence_ids,omitempty"`
	Timestamp   string               `json:"timestamp,omitempty"`
	Reasons     []string             `json:"reasons,omitempty"`
}

// LifecycleHealthProjection replaces the old percentage-derived maturity
// label with typed claims backed by the same immutable lifecycle snapshot as
// status. It intentionally reuses the projection's identity and Next Up.
type LifecycleHealthProjection struct {
	SchemaVersion      string                                  `json:"schema_version"`
	Command            string                                  `json:"command"`
	OutcomeKind        colony.OutcomeKind                      `json:"outcome_kind"`
	ProjectionRevision string                                  `json:"projection_revision"`
	Identity           LifecycleFact[LifecycleIdentityFacts]   `json:"identity"`
	Goal               LifecycleFact[string]                   `json:"goal"`
	Standing           LifecycleFact[string]                   `json:"standing"`
	Phase              LifecycleFact[LifecyclePhaseProjection] `json:"phase"`
	Tasks              LifecycleFact[[]colony.Task]            `json:"tasks"`
	Health             LifecycleHealthAssessment               `json:"health"`
	Readiness          LifecycleHealthAssessment               `json:"readiness"`
	NextAction         LifecycleProjectedAction                `json:"next_action"`
	Alternatives       []LifecycleActionChoice                 `json:"alternatives,omitempty"`
	StateEffect        colony.LifecycleStateEffect             `json:"state_effect"`
	Provenance         colony.RecoveryProvenance               `json:"provenance"`
}

func projectLifecycleHealth(facts LifecycleFacts, projection LifecycleProjection) LifecycleHealthProjection {
	assessment := classifyLifecycleHealth(facts, projection)
	readiness := assessment
	readiness.EvidenceIDs = append([]string(nil), assessment.EvidenceIDs...)
	readiness.Reasons = append([]string(nil), assessment.Reasons...)
	return LifecycleHealthProjection{
		SchemaVersion:      LifecycleResultSchemaVersion,
		Command:            "maturity",
		OutcomeKind:        projection.OutcomeKind,
		ProjectionRevision: LifecycleProjectionRevision,
		Identity:           projection.Identity,
		Goal:               projection.Goal,
		Standing:           projection.Standing,
		Phase:              projection.Phase,
		Tasks:              projection.Tasks,
		Health:             assessment,
		Readiness:          readiness,
		NextAction:         projection.NextAction,
		Alternatives:       append([]LifecycleActionChoice(nil), projection.Alternatives...),
		StateEffect:        colony.LifecycleStateEffectNone,
		Provenance:         projection.Provenance,
	}
}

func classifyLifecycleHealth(facts LifecycleFacts, projection LifecycleProjection) LifecycleHealthAssessment {
	for _, source := range []LifecycleFactSource{facts.State.Source, facts.Progress.Source, facts.Verification.Source} {
		if source.Provenance != LifecycleFactMalformed && source.Provenance != LifecycleFactUnavailable {
			continue
		}
		return LifecycleHealthAssessment{
			Status:      LifecycleHealthUnavailable,
			EvidenceIDs: lifecycleHealthEvidenceIDs(source.Domain, source.Path),
			Reasons:     []string{emptyFallback(strings.TrimSpace(source.Diagnostic), fmt.Sprintf("%s evidence is unavailable", source.Domain))},
		}
	}
	for _, source := range []LifecycleFactSource{facts.State.Source, facts.Progress.Source, facts.Verification.Source} {
		if source.Provenance != LifecycleFactMissing {
			continue
		}
		return LifecycleHealthAssessment{
			Status:      LifecycleHealthUnknown,
			EvidenceIDs: lifecycleHealthEvidenceIDs(source.Domain, source.Path),
			Reasons:     []string{emptyFallback(strings.TrimSpace(source.Diagnostic), fmt.Sprintf("%s evidence is not recorded", source.Domain))},
		}
	}

	if len(projection.Blockers) > 0 {
		assessment := LifecycleHealthAssessment{Status: LifecycleHealthBlocked}
		for _, blocker := range projection.Blockers {
			assessment.EvidenceIDs = appendUniqueString(assessment.EvidenceIDs, blocker.ID)
			assessment.Reasons = appendUniqueString(assessment.Reasons, blocker.Summary)
		}
		assessment.Timestamp = lifecycleHealthLatestFlagTime(facts.Blockers.Value)
		return assessment
	}
	if projection.Closure.Forced || projection.Closure.Status == "forced_incomplete" {
		return LifecycleHealthAssessment{
			Status:      LifecycleHealthBlocked,
			EvidenceIDs: []string{"forced-incomplete-closure"},
			Reasons:     []string{emptyFallback(strings.TrimSpace(projection.Closure.OwnerReason), "Closure was forced with incomplete work")},
		}
	}
	for _, gate := range facts.Verification.Value.Gates {
		if gate.Passed {
			continue
		}
		return LifecycleHealthAssessment{
			Status:      LifecycleHealthBlocked,
			EvidenceIDs: []string{emptyFallback(strings.TrimSpace(gate.Name), "unnamed-gate")},
			Timestamp:   strings.TrimSpace(gate.Timestamp),
			Reasons:     []string{emptyFallback(strings.TrimSpace(gate.Detail), "A recorded verification gate failed")},
		}
	}

	openFindings := make([]colony.ReviewLedgerEntry, 0)
	for _, finding := range facts.Memory.Value.Findings {
		if strings.EqualFold(strings.TrimSpace(finding.Status), "open") {
			openFindings = append(openFindings, finding)
		}
	}
	if len(openFindings) > 0 {
		assessment := LifecycleHealthAssessment{Status: LifecycleHealthDegraded}
		for _, finding := range openFindings {
			assessment.EvidenceIDs = appendUniqueString(assessment.EvidenceIDs, finding.ID)
			assessment.Reasons = appendUniqueString(assessment.Reasons, finding.Description)
			assessment.Timestamp = laterLifecycleTimestamp(assessment.Timestamp, finding.GeneratedAt)
		}
		return assessment
	}

	if len(facts.Verification.Value.Gates) == 0 {
		return LifecycleHealthAssessment{
			Status:      LifecycleHealthUnknown,
			EvidenceIDs: lifecycleHealthEvidenceIDs(facts.Verification.Source.Domain, facts.Verification.Source.Path),
			Reasons:     []string{emptyFallback(strings.TrimSpace(facts.Verification.Source.Diagnostic), "No verification evidence is recorded")},
		}
	}

	assessment := LifecycleHealthAssessment{Status: LifecycleHealthVerified}
	for _, gate := range facts.Verification.Value.Gates {
		assessment.EvidenceIDs = appendUniqueString(assessment.EvidenceIDs, gate.Name)
		assessment.Timestamp = laterLifecycleTimestamp(assessment.Timestamp, gate.Timestamp)
	}
	assessment.Reasons = []string{"All recorded verification gates passed"}
	return assessment
}

func lifecycleHealthEvidenceIDs(values ...string) []string {
	var result []string
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			result = appendUniqueString(result, strings.TrimSpace(value))
		}
	}
	return result
}

func appendUniqueString(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func lifecycleHealthLatestFlagTime(flags []colony.FlagEntry) string {
	latest := ""
	for _, flag := range flags {
		if !flag.Resolved {
			latest = laterLifecycleTimestamp(latest, flag.CreatedAt)
		}
	}
	return latest
}

func laterLifecycleTimestamp(current, candidate string) string {
	current = strings.TrimSpace(current)
	candidate = strings.TrimSpace(candidate)
	if candidate > current {
		return candidate
	}
	return current
}

func renderLifecycleHealth(result LifecycleHealthProjection, width int) string {
	phase := result.Phase.Value
	lines := []string{
		"━━ 🩺 H E A L T H   &   R E A D I N E S S ━━",
		"",
		"Colony",
		"Name: " + emptyFallback(strings.TrimSpace(result.Identity.Value.Name), "Unnamed colony"),
		"Goal: " + emptyFallback(strings.TrimSpace(result.Goal.Value), "Not recorded"),
		fmt.Sprintf("Phase %d/%d | Tasks %d/%d", phase.CurrentNumber, phase.TotalPhases, phase.CompletedTasks, phase.TotalTasks),
		"",
		"Health & Readiness",
		"Health: " + lifecycleHealthStateLabel(result.Health.Status),
		"Readiness: " + lifecycleHealthStateLabel(result.Readiness.Status),
	}
	if len(result.Health.EvidenceIDs) > 0 {
		lines = append(lines, "Evidence: "+strings.Join(result.Health.EvidenceIDs, ", "))
	}
	if result.Health.Timestamp != "" {
		lines = append(lines, "Observed: "+result.Health.Timestamp)
	}
	for _, reason := range result.Health.Reasons {
		lines = append(lines, "Reason: "+reason)
	}
	lines = append(lines,
		"",
		"Next Up",
		"Command: "+emptyFallback(lifecycleStatusActionCommand(result.NextAction), "Not available"),
		"Reason: "+emptyFallback(strings.TrimSpace(result.NextAction.Reason), "Not recorded"),
	)
	return colorLifecycleHealth(lifecycleStatusJoin(lines, width, false))
}

func lifecycleHealthStateLabel(state LifecycleHealthState) string {
	text := string(state)
	if text == "" {
		text = string(LifecycleHealthUnknown)
	}
	return strings.ToUpper(text[:1]) + text[1:]
}

func colorLifecycleHealth(output string) string {
	if !shouldUseANSIColors() {
		return output
	}
	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
	for index, line := range lines {
		if index == 0 || line == "Colony" || line == "Health & Readiness" || line == "Next Up" {
			lines[index] = "\x1b[96m" + line + "\x1b[0m"
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

// ---------------------------------------------------------------------------
// CAP-029: the quick one-question path on the same attempt, deterministic-
// check, and evidence model the rest of the work cycle uses.
// ---------------------------------------------------------------------------

// quickAttemptDispatch is the one dispatch record a quick request carries.
// Proportionality (CAP-029): a quick, one-question request draws exactly one
// worker unless one of the five named risk signals applies -- quick's own
// task brief already forbids file mutation of colony state and pheromones,
// so no named risk signal (credentials/auth, payments, release sign-off,
// data deletion, database migration) is ever detectable from a quick
// question, and this dispatch list stays at exactly one entry.
type quickAttemptDispatch struct {
	WorkerName string `json:"worker_name"`
	Caste      string `json:"caste"`
	Status     string `json:"status"`
}

// quickAttemptRecord is the one attempt a quick request opens. It carries
// its own dispatches and its own recorded verdict -- proportionate to a
// one-question request, on the same six-verdict work-outcome vocabulary
// (colony.WorkOutcome, D-05) the rest of the work cycle reports through.
type quickAttemptRecord struct {
	ID         string                 `json:"id"`
	Question   string                 `json:"question"`
	StartedAt  string                 `json:"started_at"`
	Dispatches []quickAttemptDispatch `json:"dispatches"`
	Verdict    colony.WorkOutcome     `json:"verdict,omitempty"`
	Summary    string                 `json:"summary,omitempty"`
	Evidence   []string               `json:"evidence,omitempty"`
}

// newQuickAttempt opens one attempt for a quick request. The ID is a plain,
// process-and-time-derived identifier -- unique enough to bind this
// request's dispatch and failure evidence to itself, without adopting the
// phase-keyed buildAttemptRecord model (cmd/build_attempt.go), which is
// keyed on a phase number a quick question never has.
func newQuickAttempt(question string, now time.Time) quickAttemptRecord {
	return quickAttemptRecord{
		ID:        fmt.Sprintf("quick-%s-%d", now.UTC().Format("20060102T150405.000000000Z"), os.Getpid()),
		Question:  strings.TrimSpace(question),
		StartedAt: now.UTC().Format(time.RFC3339Nano),
	}
}

// recordDispatch appends one dispatch record to the attempt. Quick's own
// production call site (runQuickScout) calls this exactly once per request,
// satisfying CAP-029's one-worker proportionality by construction.
func (a *quickAttemptRecord) recordDispatch(workerName, caste, status string) {
	a.Dispatches = append(a.Dispatches, quickAttemptDispatch{WorkerName: workerName, Caste: caste, Status: status})
}

// quickWorkVerdict derives CAP-029's verdict for a quick request: no_change
// when the request touched no files (the common, read-only case), success
// when it changed files and the deterministic checks over the changed area
// passed, and blocker when those checks failed. Never success or no_change
// on a failed check -- a changed-but-unverified file is never reported as
// though nothing needed changing.
func quickWorkVerdict(filesChanged []string, checksPassed bool) colony.WorkOutcome {
	if len(filesChanged) == 0 {
		return colony.WorkOutcomeNoChange
	}
	if checksPassed {
		return colony.WorkOutcomeSuccess
	}
	return colony.WorkOutcomeBlocker
}

// runQuickDeterministicChecks runs the deterministic check command set over
// the repository when a quick request changed files, mirroring the
// build/vet floor the rest of the work cycle already enforces
// (cmd/deterministic_floor.go). Zero files changed is the common,
// read-only case and is treated as passing without invoking a process. A
// package-level seam (mirroring newQuickWorkerInvoker) so tests can
// substitute a fake without shelling out.
var runQuickDeterministicChecks = func(root string, files []string) (bool, []string, error) {
	if len(files) == 0 {
		return true, nil, nil
	}
	steps := [][]string{{"go", "build", "./..."}, {"go", "vet", "./..."}}
	evidence := make([]string, 0, len(steps))
	for _, args := range steps {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		trimmed := strings.TrimSpace(string(out))
		if err != nil {
			if trimmed == "" {
				trimmed = err.Error()
			}
			evidence = append(evidence, fmt.Sprintf("%s: FAILED: %s", strings.Join(args, " "), trimmed))
			return false, evidence, nil
		}
		evidence = append(evidence, fmt.Sprintf("%s: passed", strings.Join(args, " ")))
	}
	return true, evidence, nil
}

func runQuickScout(question string, timeout time.Duration) (map[string]interface{}, error) {
	root := skillWorkspaceRoot()
	attempt := newQuickAttempt(question, time.Now().UTC())
	invoker := newQuickWorkerInvoker()
	if invoker == nil {
		invoker = &codex.FakeInvoker{}
	}
	ctx := context.Background()
	if _, ok := invoker.(*codex.FakeInvoker); !ok && !invoker.IsAvailable(ctx) {
		return nil, fmt.Errorf("quick scout cannot start because %s", dispatchAvailabilityMessage(invoker))
	}
	agentPath := dispatchAgentPath(root, invoker, "aether-scout")
	if err := invoker.ValidateAgent(agentPath); err != nil {
		return nil, fmt.Errorf("scout agent unavailable: %w", err)
	}
	workerName := deterministicAntName("scout", question)
	// CAP-029 proportionality: exactly one worker, recorded against this
	// request's own attempt, before dispatch -- no check-in pause is ever
	// added for a single worker with nothing pending.
	attempt.recordDispatch(workerName, "scout", "dispatched")
	taskBrief := codex.RenderTaskBrief(codex.TaskBriefData{
		TaskID: "quick.scout",
		Goal:   "Answer a lightweight user question about the current repository or Aether context.",
		Constraints: []string{
			"Read-only: do not modify source files, tests, colony state, session files, or pheromones.",
			"Keep the answer focused on the user's question.",
			"Use local codebase evidence first; use external sources only if the question requires it.",
		},
		Hints: []string{
			fmt.Sprintf("User question: %s", question),
		},
		SuccessCriteria: []string{
			"Return a concise answer with concrete file paths, commands, or sources where relevant.",
			"State remaining uncertainty instead of guessing.",
		},
	})
	workerResult, err := invoker.Invoke(ctx, codex.WorkerConfig{
		AgentName:        "aether-scout",
		AgentTOMLPath:    agentPath,
		Caste:            "scout",
		WorkerName:       workerName,
		TaskID:           "quick.scout",
		TaskBrief:        taskBrief,
		ContextCapsule:   renderQuickContextCapsule(question),
		HandoffSection:   renderWorkerHandoffSection("quick", 0, workerName),
		Root:             root,
		Timeout:          timeout,
		SkillSection:     resolveSkillSectionForWorkflow("quick", "scout", question),
		PheromoneSection: resolvePheromoneSection(),
	})
	if err != nil {
		attempt.Dispatches[0].Status = "failed"
		attempt.Verdict = colony.WorkOutcomeBlocker
		recordQuickFailureToMidden(question, attempt.ID, err)
		return nil, err
	}
	attempt.Dispatches[0].Status = emptyFallback(workerResult.Status, "completed")

	filesChanged := append(append([]string(nil), workerResult.FilesCreated...), workerResult.FilesModified...)
	checksPassed := true
	var checkEvidence []string
	if len(filesChanged) > 0 {
		var checkErr error
		checksPassed, checkEvidence, checkErr = runQuickDeterministicChecks(root, filesChanged)
		if checkErr != nil {
			checksPassed = false
			checkEvidence = append(checkEvidence, checkErr.Error())
		}
	}
	attempt.Verdict = quickWorkVerdict(filesChanged, checksPassed)
	attempt.Summary = strings.TrimSpace(workerResult.Summary)
	attempt.Evidence = checkEvidence
	if attempt.Verdict == colony.WorkOutcomeBlocker {
		recordQuickFailureToMidden(question, attempt.ID, fmt.Errorf("quick deterministic checks failed for %s: %s", question, strings.Join(checkEvidence, "; ")))
	}

	return map[string]interface{}{
		"mode":           "quick",
		"question":       question,
		"worker_name":    workerResult.WorkerName,
		"status":         emptyFallback(workerResult.Status, "completed"),
		"summary":        strings.TrimSpace(workerResult.Summary),
		"raw_output":     codex.SanitizeWorkerDiagnosticOutput(workerResult.RawOutput),
		"duration_ms":    workerResult.Duration.Milliseconds(),
		"files":          workerResult.FilesModified,
		"next":           "aether status",
		"attempt_id":     attempt.ID,
		"work_outcome":   attempt.Verdict,
		"check_evidence": checkEvidence,
	}, nil
}

func renderQuickContextCapsule(question string) string {
	var b strings.Builder
	b.WriteString("# Quick Scout Context\n\n")
	b.WriteString("This is a read-only, lightweight query. Avoid build, continue, seal, or state mutation ceremony.\n\n")
	fmt.Fprintf(&b, "- Question: %s\n", question)
	if root := strings.TrimSpace(skillWorkspaceRoot()); root != "" {
		fmt.Fprintf(&b, "- Repository root: %s\n", root)
	}
	return strings.TrimSpace(b.String())
}

func renderQuickVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner("⚡", "Quick Scout"))
	b.WriteString(visualDividerStr())
	b.WriteString("Question: ")
	b.WriteString(emptyFallback(stringValue(result["question"]), "(none)"))
	b.WriteString("\n")
	b.WriteString("Status: ")
	b.WriteString(emptyFallback(stringValue(result["status"]), "completed"))
	b.WriteString("\n")
	if worker := strings.TrimSpace(stringValue(result["worker_name"])); worker != "" {
		b.WriteString("Scout: ")
		b.WriteString(worker)
		b.WriteString("\n")
	}
	if verdict, ok := result["work_outcome"].(colony.WorkOutcome); ok && verdict.Valid() {
		if label := colony.WorkOutcomeLabels()[verdict]; label != "" {
			b.WriteString("Verdict: ")
			b.WriteString(label)
			b.WriteString("\n")
		}
	}
	if durationMS, ok := result["duration_ms"].(int64); ok && durationMS > 0 {
		b.WriteString(fmt.Sprintf("Time: %.1fs\n", float64(durationMS)/1000))
	}
	if summary := strings.TrimSpace(stringValue(result["summary"])); summary != "" {
		b.WriteString("\n")
		b.WriteString(summary)
		b.WriteString("\n")
	}
	return b.String()
}

var semverPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)

func runBumpVersion(root, target string, dryRun bool) (map[string]interface{}, error) {
	target = normalizeVersion(target)
	if !semverPattern.MatchString(target) {
		return nil, fmt.Errorf("invalid version %q: use X.Y.Z", target)
	}
	versionFiles := []string{
		filepath.Join(root, ".aether", "version.json"),
		filepath.Join(root, "npm", "package.json"),
	}
	updates := make([]map[string]interface{}, 0, len(versionFiles))
	for _, path := range versionFiles {
		oldVersion, err := readJSONVersion(path)
		if err != nil {
			return nil, err
		}
		cmp, err := compareSemver(target, oldVersion)
		if err != nil {
			return nil, err
		}
		if cmp < 0 {
			return nil, fmt.Errorf("target version %s is older than %s in %s", target, oldVersion, path)
		}
		changed := oldVersion != target
		if changed && !dryRun {
			if err := writeJSONVersion(path, target); err != nil {
				return nil, err
			}
		}
		rel, _ := filepath.Rel(root, path)
		updates = append(updates, map[string]interface{}{
			"path":    filepath.ToSlash(rel),
			"from":    oldVersion,
			"to":      target,
			"changed": changed,
			"dry_run": dryRun,
		})
	}
	return map[string]interface{}{
		"mode":    "bump-version",
		"version": target,
		"dry_run": dryRun,
		"updates": updates,
		"next":    "go test ./... && aether publish",
	}, nil
}

func readJSONVersion(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		return "", fmt.Errorf("parse %s: %w", path, err)
	}
	version, _ := doc["version"].(string)
	version = normalizeVersion(version)
	if !semverPattern.MatchString(version) {
		return "", fmt.Errorf("%s has invalid version %q", path, version)
	}
	return version, nil
}

func writeJSONVersion(path, version string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	doc["version"] = version
	if _, ok := doc["updated_at"]; ok {
		doc["updated_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	}
	encoded, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	return os.WriteFile(path, append(encoded, '\n'), 0644)
}

func compareSemver(a, b string) (int, error) {
	pa, err := parseSemver(a)
	if err != nil {
		return 0, err
	}
	pb, err := parseSemver(b)
	if err != nil {
		return 0, err
	}
	for i := 0; i < 3; i++ {
		switch {
		case pa[i] > pb[i]:
			return 1, nil
		case pa[i] < pb[i]:
			return -1, nil
		}
	}
	return 0, nil
}

func parseSemver(version string) ([3]int, error) {
	var out [3]int
	parts := strings.Split(normalizeVersion(version), ".")
	if len(parts) != 3 {
		return out, fmt.Errorf("invalid version %q", version)
	}
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			return out, fmt.Errorf("invalid version %q", version)
		}
		out[i] = n
	}
	return out, nil
}

func renderBumpVersionVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("bump-version"), "Bump Version"))
	b.WriteString(visualDividerStr())
	b.WriteString("Version: ")
	b.WriteString(emptyFallback(stringValue(result["version"]), "unknown"))
	b.WriteString("\n")
	if boolValue(result["dry_run"]) {
		b.WriteString("Mode: dry run\n")
	}
	if updates, ok := result["updates"].([]map[string]interface{}); ok {
		for _, update := range updates {
			b.WriteString(fmt.Sprintf("- %s: %s -> %s\n", stringValue(update["path"]), stringValue(update["from"]), stringValue(update["to"])))
		}
	}
	b.WriteString(renderNextUp(fmt.Sprintf("Run `%s` after reviewing the version bump.", emptyFallback(stringValue(result["next"]), "go test ./..."))))
	return b.String()
}

func prepareStateMigrationMutation(repositoryRoot, dataRoot string, originalData []byte, now time.Time) (maintenanceMutationPlan, map[string]interface{}, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(originalData, &raw); err != nil {
		return maintenanceMutationPlan{}, nil, fmt.Errorf("COLONY_STATE.json is not valid JSON: %w", err)
	}
	fromVersion := strings.TrimSpace(stringValue(raw["version"]))
	if fromVersion == "" {
		fromVersion = "legacy"
	}
	var state colony.ColonyState
	if err := json.Unmarshal(originalData, &state); err != nil {
		return maintenanceMutationPlan{}, nil, fmt.Errorf("legacy state is not compatible with automatic migration: %w", err)
	}
	originalEvidencePolicy := state.Plan.EvidencePolicy
	state.Plan.EvidencePolicy = inferredPlanEvidencePolicy(state.Plan)

	// Backfill typed phase modes. This is the one sanctioned use of keyword
	// inference: run once, at migration time, writing a durable, auditable
	// value to disk. Runtime keyword inference is gone (effectiveQueenPhaseMode
	// no longer falls back to it), so phases created before typed modes must
	// receive theirs here or dispatch decisions would fall to the neutral
	// default forever.
	modesBackfilled := 0
	for i := range state.Plan.Phases {
		if !state.Plan.Phases[i].Mode.Valid() {
			state.Plan.Phases[i].Mode = colony.InferPhaseMode(state.Plan.Phases[i].Name, state.Plan.Phases[i].Description)
			modesBackfilled++
		}
	}

	plan := maintenanceMutationPlan{
		SchemaVersion:   maintenanceMutationSchemaVersion,
		Operation:       "migrate-state",
		TransactionID:   "migrate-state-" + now.UTC().Format("20060102T150405.000000000Z"),
		SourceRoot:      dataRoot,
		DestinationRoot: dataRoot,
		CurrentVersion:  fromVersion,
		DesiredVersion:  "3.0",
		Checkpoint:      "maintenance:migrate-state:validated",
		Recovery:        "aether resume",
		Allowlist: lifecycleTransactionAllowlist{
			RepositoryRoot:    repositoryRoot,
			LifecycleDataRoot: dataRoot,
		},
	}
	if fromVersion == "3.0" && originalEvidencePolicy == state.Plan.EvidencePolicy && originalEvidencePolicy != "" && modesBackfilled == 0 {
		return plan, map[string]interface{}{
			"mode":            "migrate-state",
			"migrated":        false,
			"from":            fromVersion,
			"to":              "3.0",
			"reason":          "already current",
			"evidence_policy": string(state.Plan.EvidencePolicy),
			"next":            "aether medic --deep",
		}, nil
	}
	state.Version = "3.0"
	if strings.TrimSpace(string(state.State)) == "" {
		if state.Goal != nil && strings.TrimSpace(*state.Goal) != "" {
			state.State = colony.StateREADY
		} else {
			state.State = colony.StateIDLE
		}
	}
	state.Events = append(trimmedEvents(state.Events),
		fmt.Sprintf("%s|state_migrated|migrate-state|Migrated COLONY_STATE.json from %s to 3.0; plan evidence policy=%s", now.UTC().Format(time.RFC3339), fromVersion, state.Plan.EvidencePolicy),
	)
	migratedData, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return maintenanceMutationPlan{}, nil, fmt.Errorf("marshal migrated state: %w", err)
	}
	migratedData = append(migratedData, '\n')
	backupPath := filepath.Join("backups", fmt.Sprintf("COLONY_STATE.pre-migrate.%s.json", now.UTC().Format("20060102-150405.000000000")))
	plan.Targets = []maintenanceMutationTarget{
		{Root: lifecycleTransactionRootData, RelativeTarget: backupPath, Source: "COLONY_STATE.json byte-exact preimage", Action: lifecycleTransactionWrite, Content: originalData, Managed: true},
		{Root: lifecycleTransactionRootData, RelativeTarget: "COLONY_STATE.json", Source: "schema migration 3.0", Action: lifecycleTransactionWrite, Content: migratedData, Managed: true},
	}
	return plan, map[string]interface{}{
		"mode":             "migrate-state",
		"migrated":         false,
		"from":             fromVersion,
		"to":               "3.0",
		"evidence_policy":  string(state.Plan.EvidencePolicy),
		"modes_backfilled": modesBackfilled,
		"backup_path":      filepath.ToSlash(backupPath),
		"rollback_command": migrationRollbackCommand(backupPath),
		"next":             "aether medic --deep",
	}, nil
}

func runMigrateState(dryRun bool) (map[string]interface{}, error) {
	if store == nil {
		return nil, fmt.Errorf("no store initialized")
	}
	originalData, err := store.ReadFile("COLONY_STATE.json")
	if err != nil {
		return nil, fmt.Errorf("COLONY_STATE.json not found: %w", err)
	}
	dataRoot := filepath.Clean(store.BasePath())
	repositoryRoot := repoRootFromStore(store)
	if repositoryRoot == "" || repositoryRoot == dataRoot {
		repositoryRoot = filepath.Dir(dataRoot)
	}
	plan, result, err := prepareStateMigrationMutation(repositoryRoot, dataRoot, originalData, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	preview, err := prepareMaintenanceMutation(plan)
	if err != nil {
		return nil, err
	}
	result["dry_run"] = dryRun
	result["preview"] = preview
	result["state_effect"] = colony.LifecycleStateEffectNone
	result["recovery"] = plan.Recovery
	if dryRun || len(plan.Targets) == 0 {
		return result, nil
	}
	mutation, err := commitMaintenanceMutation(plan)
	result["transaction"] = mutation.Preview.TransactionID
	result["receipt"] = mutation.Receipt
	result["state_effect"] = mutation.StateEffect
	result["verification"] = mutation.Verification
	result["recovery"] = mutation.Recovery
	result["migrated"] = mutation.StateEffect == colony.LifecycleStateEffectCommitted
	if err != nil {
		return result, fmt.Errorf("migrate state transaction: %w", err)
	}
	return result, nil
}

func migrationRollbackCommand(backupPath string) string {
	backupPath = filepath.ToSlash(strings.TrimSpace(backupPath))
	if backupPath == "" {
		return ""
	}
	return fmt.Sprintf("aether migrate-state --rollback %s", backupPath)
}

func runMigrateStateRollback(backupPath string, dryRun bool) (map[string]interface{}, error) {
	if store == nil {
		return nil, fmt.Errorf("no store initialized")
	}
	cleanPath, err := validateMigrationBackupPath(backupPath)
	if err != nil {
		return nil, err
	}
	backupData, err := store.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("read migration rollback backup: %w", err)
	}
	var restored colony.ColonyState
	if err := json.Unmarshal(backupData, &restored); err != nil {
		return nil, fmt.Errorf("migration rollback backup is not valid colony state: %w", err)
	}
	current, err := store.ReadFile("COLONY_STATE.json")
	if err != nil {
		return nil, fmt.Errorf("COLONY_STATE.json not found: %w", err)
	}
	now := time.Now().UTC()
	safetyBackup := filepath.Join("backups", fmt.Sprintf("COLONY_STATE.pre-rollback.%s.json", now.Format("20060102-150405.000000000")))
	dataRoot := filepath.Clean(store.BasePath())
	repositoryRoot := repoRootFromStore(store)
	if repositoryRoot == "" || repositoryRoot == dataRoot {
		repositoryRoot = filepath.Dir(dataRoot)
	}
	plan := maintenanceMutationPlan{
		SchemaVersion: maintenanceMutationSchemaVersion, Operation: "migrate-state-rollback",
		TransactionID: "migrate-state-rollback-" + now.Format("20060102T150405.000000000Z"),
		SourceRoot:    dataRoot, DestinationRoot: dataRoot,
		CurrentVersion: "3.0", DesiredVersion: restored.Version,
		Checkpoint: "maintenance:migrate-state-rollback:validated", Recovery: "aether resume",
		Allowlist: lifecycleTransactionAllowlist{RepositoryRoot: repositoryRoot, LifecycleDataRoot: dataRoot},
		Targets: []maintenanceMutationTarget{
			{Root: lifecycleTransactionRootData, RelativeTarget: safetyBackup, Source: "COLONY_STATE.json byte-exact pre-rollback image", Action: lifecycleTransactionWrite, Content: current, Managed: true},
			{Root: lifecycleTransactionRootData, RelativeTarget: "COLONY_STATE.json", Source: filepath.ToSlash(cleanPath), Action: lifecycleTransactionWrite, Content: backupData, Managed: true},
		},
	}
	preview, err := prepareMaintenanceMutation(plan)
	if err != nil {
		return nil, err
	}
	result := map[string]interface{}{
		"mode":          "migrate-state-rollback",
		"rolled_back":   false,
		"dry_run":       dryRun,
		"restored_from": filepath.ToSlash(cleanPath),
		"safety_backup": filepath.ToSlash(safetyBackup),
		"state_version": restored.Version,
		"next":          "aether medic --deep",
		"preview":       preview,
		"state_effect":  colony.LifecycleStateEffectNone,
		"recovery":      plan.Recovery,
	}
	if dryRun {
		return result, nil
	}
	mutation, err := commitMaintenanceMutation(plan)
	result["rolled_back"] = mutation.StateEffect == colony.LifecycleStateEffectCommitted
	result["state_effect"] = mutation.StateEffect
	result["transaction"] = mutation.Preview.TransactionID
	result["receipt"] = mutation.Receipt
	result["verification"] = mutation.Verification
	result["recovery"] = mutation.Recovery
	if err != nil {
		return result, fmt.Errorf("restore migration backup transaction: %w", err)
	}
	return result, nil
}

func validateMigrationBackupPath(path string) (string, error) {
	path = filepath.Clean(filepath.FromSlash(strings.TrimSpace(path)))
	if path == "." || filepath.IsAbs(path) || path == ".." || strings.HasPrefix(path, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("rollback backup must be a relative path under backups/")
	}
	prefix := "backups" + string(filepath.Separator)
	if !strings.HasPrefix(path, prefix) {
		return "", fmt.Errorf("rollback backup must be under backups/")
	}
	name := filepath.Base(path)
	if !strings.HasPrefix(name, "COLONY_STATE.pre-migrate.") || filepath.Ext(name) != ".json" {
		return "", fmt.Errorf("rollback backup must be an Aether-created COLONY_STATE.pre-migrate backup")
	}
	fullPath := filepath.Join(store.BasePath(), path)
	info, err := os.Lstat(fullPath)
	if err != nil {
		return "", fmt.Errorf("rollback backup is unavailable: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("rollback backup must not be a symbolic link")
	}
	return path, nil
}

func renderMigrateStateVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("migrate-state"), "Migrate State"))
	b.WriteString(visualDividerStr())
	if boolValue(result["rolled_back"]) {
		b.WriteString("State rollback completed.\n")
	} else if stringValue(result["mode"]) == "migrate-state-rollback" && boolValue(result["dry_run"]) {
		b.WriteString("State rollback preview completed.\n")
	} else if boolValue(result["migrated"]) {
		b.WriteString("State migrated.\n")
	} else {
		b.WriteString("No migration applied.\n")
	}
	if stringValue(result["mode"]) == "migrate-state-rollback" {
		b.WriteString("Restored from: ")
		b.WriteString(emptyFallback(stringValue(result["restored_from"]), "unknown"))
		b.WriteString("\n")
		if safety := strings.TrimSpace(stringValue(result["safety_backup"])); safety != "" {
			b.WriteString("Safety backup: ")
			b.WriteString(safety)
			b.WriteString("\n")
		}
	} else {
		b.WriteString(fmt.Sprintf("Version: %s -> %s\n", emptyFallback(stringValue(result["from"]), "legacy"), emptyFallback(stringValue(result["to"]), "3.0")))
	}
	if backup := strings.TrimSpace(stringValue(result["backup_path"])); backup != "" {
		b.WriteString("Backup: ")
		b.WriteString(backup)
		b.WriteString("\n")
	}
	if rollback := strings.TrimSpace(stringValue(result["rollback_command"])); rollback != "" {
		b.WriteString("Rollback: ")
		b.WriteString(rollback)
		b.WriteString("\n")
	}
	b.WriteString(renderNextUp(fmt.Sprintf("Run `%s` to verify colony health.", emptyFallback(stringValue(result["next"]), "aether medic --deep"))))
	return b.String()
}

func verifyCasteSurfaces(root string) map[string]interface{} {
	surfaces := map[string]string{
		"claude_agents":   filepath.Join(root, ".claude", "agents", "ant", "*.md"),
		"opencode_agents": filepath.Join(root, ".opencode", "agents", "*.md"),
		"codex_agents":    filepath.Join(root, ".codex", "agents", "*.toml"),
	}
	counts := make(map[string]int, len(surfaces))
	ok := true
	for name, pattern := range surfaces {
		count := countFilesInDir(pattern)
		counts[name] = count
		if count != expectedClaudeAgents {
			ok = false
		}
	}
	castes := loadCasteAssignments(root)
	return map[string]interface{}{
		"mode":            "verify-castes",
		"ok":              ok,
		"expected_agents": expectedClaudeAgents,
		"counts":          counts,
		"castes":          castes,
		"total_castes":    len(castes),
		"next":            "aether medic --deep",
	}
}

func loadCasteAssignments(root string) []map[string]interface{} {
	matches, _ := filepath.Glob(filepath.Join(root, ".claude", "agents", "ant", "aether-*.md"))
	sort.Strings(matches)
	assignments := make([]map[string]interface{}, 0, len(matches))
	for _, path := range matches {
		name := strings.TrimSuffix(filepath.Base(path), ".md")
		name = strings.TrimPrefix(name, "aether-")
		model := "inherit"
		if data, err := os.ReadFile(path); err == nil {
			if found := regexp.MustCompile(`(?m)^model:\s*([A-Za-z0-9_.-]+)\s*$`).FindStringSubmatch(string(data)); len(found) == 2 {
				model = found[1]
			}
		}
		assignments = append(assignments, map[string]interface{}{
			"caste": name,
			"model": model,
		})
	}
	return assignments
}

func renderVerifyCastesVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("verify-castes"), "Verify Castes"))
	b.WriteString(visualDividerStr())
	b.WriteString(fmt.Sprintf("Expected agents per surface: %d\n", intValue(result["expected_agents"])))
	if counts, ok := result["counts"].(map[string]int); ok {
		keys := make([]string, 0, len(counts))
		for key := range counts {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			b.WriteString(fmt.Sprintf("- %s: %d\n", key, counts[key]))
		}
	}
	if !boolValue(result["ok"]) {
		b.WriteString("\nCaste surfaces are out of sync.\n")
	}
	b.WriteString(renderNextUp(fmt.Sprintf("Run `%s` for the full health scan.", emptyFallback(stringValue(result["next"]), "aether medic --deep"))))
	return b.String()
}
