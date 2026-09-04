package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

const (
	maintenanceCatalogSchemaVersion            = "maintenance-catalog/v1"
	maintenanceRecoveryInspectionSchemaVersion = "maintenance-recovery-inspection/v1"
)

type maintenanceInspectionFinding struct {
	Code            string   `json:"code"`
	Summary         string   `json:"summary"`
	SourcePath      string   `json:"source_path,omitempty"`
	GeneratedPath   string   `json:"generated_path,omitempty"`
	EvidencePaths   []string `json:"evidence_paths"`
	RecoveryCommand string   `json:"recovery_command"`
}

type maintenanceInspectionEvidence struct {
	Scope   string   `json:"scope"`
	Paths   []string `json:"paths"`
	Checked int      `json:"checked"`
	Status  string   `json:"status"`
}

type maintenanceInspectionVerification struct {
	Status        string `json:"status"`
	EvidenceCount int    `json:"evidence_count"`
	FindingCount  int    `json:"finding_count"`
}

type maintenanceOperation struct {
	OperationID         string `json:"operation_id"`
	Label               string `json:"label"`
	RuntimeCommand      string `json:"runtime_command"`
	DisplayCommand      string `json:"display_command"`
	MutationClass       string `json:"mutation_class"`
	PreviewAvailable    bool   `json:"preview_available"`
	TransactionRequired bool   `json:"transaction_required"`
	ReceiptType         string `json:"receipt_type"`
	RecoveryAction      string `json:"recovery_action"`
}

type maintenanceCatalog struct {
	Inspection []maintenanceOperation `json:"inspection"`
	Mutation   []maintenanceOperation `json:"mutation"`
}

type maintenanceLandingResult struct {
	SchemaVersion string                      `json:"schema_version"`
	Command       string                      `json:"command"`
	OutcomeKind   colony.OutcomeKind          `json:"outcome_kind"`
	Explanation   string                      `json:"explanation"`
	Catalog       maintenanceCatalog          `json:"catalog"`
	Lifecycle     LifecycleProjection         `json:"lifecycle"`
	StateEffect   colony.LifecycleStateEffect `json:"state_effect"`
}

type maintenanceRecoveryInspectionResult struct {
	SchemaVersion  string                            `json:"schema_version"`
	OperationID    string                            `json:"operation_id"`
	Command        string                            `json:"command"`
	OutcomeKind    colony.OutcomeKind                `json:"outcome_kind"`
	Explanation    string                            `json:"explanation"`
	Issues         []HealthIssue                     `json:"issues"`
	Findings       []maintenanceInspectionFinding    `json:"findings"`
	Evidence       []maintenanceInspectionEvidence   `json:"evidence"`
	Verification   maintenanceInspectionVerification `json:"verification"`
	Provenance     []LifecycleFactSource             `json:"provenance"`
	Lifecycle      LifecycleProjection               `json:"lifecycle"`
	StateEffect    colony.LifecycleStateEffect       `json:"state_effect"`
	RecoveryAction string                            `json:"recovery_action"`
	NextAction     string                            `json:"next_action"`
}

var maintenanceCmd = &cobra.Command{
	Use:         "maintenance",
	Short:       "Inspect or repair Aether internals with preview and rollback.",
	Args:        cobra.NoArgs,
	Annotations: map[string]string{"aether.io/read-only": "true"},
	RunE:        runMaintenanceLanding,
}

var maintenanceRecoveryInspectCmd = &cobra.Command{
	Use:         "recovery-inspect",
	Short:       "Inspect recovery evidence without changing colony state",
	Args:        cobra.NoArgs,
	Annotations: map[string]string{"aether.io/read-only": "true"},
	RunE:        runMaintenanceRecoveryInspect,
}

func init() {
	rootCmd.AddCommand(maintenanceCmd)
	maintenanceCmd.AddCommand(maintenanceRecoveryInspectCmd)
}

func runMaintenanceLanding(cmd *cobra.Command, _ []string) error {
	platform := detectPlatform()
	result := buildMaintenanceLanding(platform, time.Now().UTC().Truncate(time.Second))
	outputWorkflow(result, renderMaintenanceLanding(result))
	return nil
}

func buildMaintenanceLanding(platform string, observedAt time.Time) maintenanceLandingResult {
	root := resolveAetherRootPath()
	facts, _ := loadLifecycleFacts(root, store, observedAt)
	return maintenanceLandingResult{
		SchemaVersion: maintenanceCatalogSchemaVersion,
		Command:       "maintenance",
		OutcomeKind:   colony.OutcomeKindNoChange,
		Explanation:   "The Queen keeps expert inspection and repair tools here so the ordinary colony journey stays clear. Opening this catalog never starts a repair.",
		Catalog:       buildMaintenanceCatalog(platform),
		Lifecycle:     projectLifecycle(facts, LifecycleViewFocused, platform),
		StateEffect:   colony.LifecycleStateEffectNone,
	}
}

func buildMaintenanceCatalog(platform string) maintenanceCatalog {
	readOnly := func(id, label, command, receipt string) maintenanceOperation {
		return newMaintenanceOperation(platform, id, label, command, "read_only", false, false, receipt, "No rollback is needed because this operation writes no state.")
	}
	mutating := func(id, label, command string, preview bool, recovery string) maintenanceOperation {
		return newMaintenanceOperation(platform, id, label, command, "mutating", preview, true, "maintenance-transaction/v1", recovery)
	}
	return maintenanceCatalog{
		Inspection: []maintenanceOperation{
			readOnly("recovery.inspect", "Recovery evidence and stuck-state diagnosis", "aether maintenance recovery-inspect", maintenanceRecoveryInspectionSchemaVersion),
			readOnly("integrity.inspect", "Release and hub integrity", "aether integrity", integrityInspectionSchemaVersion),
			readOnly("source.parity.inspect", "Generated and source parity", "aether source-check", sourceCheckResultSchemaVersion),
			readOnly("registry.inspect", "Registered colony inventory", "aether registry-list", "registry-inspection/v1"),
			readOnly("chamber.inspect", "Chamber inventory", "aether chamber-list", "chamber-inspection/v1"),
			readOnly("context.inspect", "Current lifecycle context", "aether status --compact", LifecycleResultSchemaVersion),
			readOnly("archive.inspect", "Archived chamber integrity", "aether chamber-verify --name <chamber>", "archive-inspection/v1"),
			readOnly("skills.inspect", "Live skill inventory", "aether maintenance skills inspect", maintenanceSkillsSchemaVersion),
			readOnly("skills.diff", "Live skill inventory drift", "aether maintenance skills diff --before <receipt> --after <receipt>", maintenanceSkillsDiffSchemaVersion),
		},
		Mutation: []maintenanceOperation{
			mutating("update.apply", "Update companion surfaces", "aether update", true, "Restore every path named by the transaction receipt, then rerun aether integrity."),
			mutating("state.migration.apply", "Migrate colony state", "aether migrate-state", true, "Run aether migrate-state --rollback <backup-path> from the receipt."),
			mutating("data.cleanup.apply", "Remove test data", "aether data-clean --confirm", true, "Restore the checkpoint named by the transaction receipt."),
			mutating("backup.prune.apply", "Prune old backups", "aether backup-prune-global --cap <count>", false, "Restore the checkpoint named by the transaction receipt."),
			mutating("temp.cleanup.apply", "Remove expired temporary files", "aether temp-clean", false, "Restore the checkpoint named by the transaction receipt."),
			mutating("registry.update", "Update a colony registry entry", "aether registry-add --path <repo>", false, "Restore the registry pre-image named by the transaction receipt."),
			mutating("chamber.create", "Create a chamber archive", "aether chamber-create --name <chamber>", false, "Remove only the staged chamber named by a failed transaction receipt."),
		},
	}
}

func runMaintenanceRecoveryInspect(_ *cobra.Command, _ []string) error {
	root := resolveAetherRootPath()
	facts, loadErr := loadLifecycleFacts(root, store, time.Now().UTC().Truncate(time.Second))
	result := buildMaintenanceRecoveryInspection(facts, detectPlatform(), loadErr)
	outputWorkflow(result, renderMaintenanceRecoveryInspection(result))
	return nil
}

func buildMaintenanceRecoveryInspection(facts LifecycleFacts, platform string, loadErr error) maintenanceRecoveryInspectionResult {
	issues := inspectRecoveryIssuesReadOnly(facts)
	if loadErr != nil {
		issues = append(issues, issueCritical("state", facts.State.Source.Path, fmt.Sprintf("Lifecycle evidence could not be loaded: %v", loadErr)))
	}
	if issues == nil {
		issues = []HealthIssue{}
	}

	provenance := facts.Sources()
	evidence := make([]maintenanceInspectionEvidence, 0, len(provenance)+1)
	for _, source := range provenance {
		paths := splitMaintenanceEvidencePaths(source.Path)
		evidence = append(evidence, maintenanceInspectionEvidence{
			Scope:   source.Domain,
			Paths:   paths,
			Checked: len(paths),
			Status:  string(source.Provenance),
		})
	}
	scanPaths := maintenanceRecoveryScanPaths(facts)
	evidence = append(evidence, maintenanceInspectionEvidence{
		Scope:   "stuck-state scanners",
		Paths:   scanPaths,
		Checked: len(scanPaths),
		Status:  maintenanceRecoveryEvidenceStatus(issues),
	})

	findings := make([]maintenanceInspectionFinding, 0, len(issues))
	for _, issue := range issues {
		paths := splitMaintenanceEvidencePaths(issue.File)
		finding := maintenanceInspectionFinding{
			Code:            "recovery." + issue.Category,
			Summary:         issue.Message,
			EvidencePaths:   paths,
			RecoveryCommand: "aether resume",
		}
		if len(paths) > 0 {
			finding.SourcePath = paths[0]
		}
		findings = append(findings, finding)
	}

	outcome := colony.OutcomeKindNoChange
	verificationStatus := "pass"
	if len(issues) > 0 {
		outcome = colony.OutcomeKindRecoveryRequired
		verificationStatus = "attention_required"
	}
	return maintenanceRecoveryInspectionResult{
		SchemaVersion: maintenanceRecoveryInspectionSchemaVersion,
		OperationID:   "recovery.inspect",
		Command:       "maintenance recovery-inspect",
		OutcomeKind:   outcome,
		Explanation:   "Recovery evidence was inspected without repairing, normalizing, or restoring any durable state. Resume remains the only recovery owner.",
		Issues:        issues,
		Findings:      findings,
		Evidence:      evidence,
		Verification: maintenanceInspectionVerification{
			Status:        verificationStatus,
			EvidenceCount: len(evidence),
			FindingCount:  len(findings),
		},
		Provenance:     provenance,
		Lifecycle:      projectLifecycle(facts, LifecycleViewFocused, platform),
		StateEffect:    colony.LifecycleStateEffectNone,
		RecoveryAction: "aether resume",
		NextAction:     "aether resume",
	}
}

// inspectRecoveryIssuesReadOnly retains the useful recover scanners without
// calling their repair-capable orchestration. The state and manifest readers
// below use os.ReadFile directly so an inspection cannot create lock files or
// normalize legacy bytes on disk.
func inspectRecoveryIssuesReadOnly(facts LifecycleFacts) []HealthIssue {
	dataDir := maintenanceRecoveryDataDir(facts)
	var issues []HealthIssue
	issues = append(issues, scanStaleSpawnedWorkers(dataDir)...)

	if facts.State.Source.Provenance != LifecycleFactConfirmed {
		message := facts.State.Source.Diagnostic
		if strings.TrimSpace(message) == "" {
			message = "Colony state is not confirmed by durable evidence"
		}
		issues = append(issues, issueCritical("state", facts.State.Source.Path, message))
		issues = append(issues, scanMissingAgentFiles()...)
		return issues
	}

	state := facts.State.Value
	manifest := loadRecoveryManifestReadOnly(dataDir, state.CurrentPhase)
	issues = append(issues, scanBadManifest(state, dataDir)...)
	issues = append(issues, scanMissingBuildPacketReadOnly(state, manifest)...)
	issues = append(issues, scanPartialPhaseReadOnly(state, dataDir, manifest)...)
	issues = append(issues, scanDirtyWorktrees(state)...)
	issues = append(issues, scanBrokenSurvey(state, dataDir)...)
	issues = append(issues, scanMissingAgentFiles()...)
	issues = append(issues, scanUnreconciledWorkerChangesReadOnly(facts.Root, dataDir, state, manifest)...)
	return issues
}

// maintenanceRecoveryDataDir binds the scanners to the same state source that
// LifecycleFacts inspected. COLONY_DATA_DIR may legitimately place lifecycle
// evidence outside the repository's default .aether/data directory.
func maintenanceRecoveryDataDir(facts LifecycleFacts) string {
	statePath := strings.TrimSpace(facts.State.Source.Path)
	if statePath != "" && statePath != "(unavailable)" && !strings.Contains(statePath, ",") {
		statePath = filepath.FromSlash(statePath)
		if !filepath.IsAbs(statePath) {
			statePath = filepath.Join(facts.Root, statePath)
		}
		if filepath.Base(statePath) == "COLONY_STATE.json" {
			return filepath.Dir(statePath)
		}
	}
	return filepath.Join(facts.Root, ".aether", "data")
}

func loadRecoveryManifestReadOnly(dataDir string, phaseID int) codexContinueManifest {
	if phaseID < 1 {
		return codexContinueManifest{}
	}
	rel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phaseID), "manifest.json"))
	raw, err := os.ReadFile(filepath.Join(dataDir, filepath.FromSlash(rel)))
	if err != nil {
		return codexContinueManifest{}
	}
	var manifest codexBuildManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return codexContinueManifest{}
	}
	return codexContinueManifest{Present: true, Path: rel, Data: manifest}
}

func scanMissingBuildPacketReadOnly(state colony.ColonyState, manifest codexContinueManifest) []HealthIssue {
	if state.State != colony.StateEXECUTING && state.State != colony.StateBUILT || state.CurrentPhase < 1 {
		return nil
	}
	if manifest.Present && !manifest.Data.PlanOnly && len(manifest.Data.Dispatches) > 0 {
		return nil
	}
	return []HealthIssue{fixableIssue(issueCritical(
		"missing_build_packet",
		fmt.Sprintf("build/phase-%d/manifest.json", state.CurrentPhase),
		fmt.Sprintf("No build packet for phase %d (state=%s)", state.CurrentPhase, state.State),
	))}
}

func scanPartialPhaseReadOnly(state colony.ColonyState, dataDir string, manifest codexContinueManifest) []HealthIssue {
	if state.State != colony.StateEXECUTING || state.CurrentPhase < 1 {
		return nil
	}
	var issues []HealthIssue
	if manifest.Present && len(manifest.Data.Dispatches) > 0 {
		allTerminal := true
		for _, dispatch := range manifest.Data.Dispatches {
			if dispatch.Status != "completed" && dispatch.Status != "failed" {
				allTerminal = false
				break
			}
		}
		if allTerminal {
			continueRel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", state.CurrentPhase), "continue.json"))
			if _, err := os.Stat(filepath.Join(dataDir, filepath.FromSlash(continueRel))); os.IsNotExist(err) {
				issues = append(issues, fixableIssue(issueWarning("partial_phase", continueRel, "Build completed but continue not run")))
			}
		}
	}
	if !manifest.Present {
		for _, phase := range state.Plan.Phases {
			if phase.ID == state.CurrentPhase && phase.Status == colony.PhaseInProgress {
				issues = append(issues, fixableIssue(issueWarning(
					"partial_phase",
					"COLONY_STATE.json",
					fmt.Sprintf("Phase %d marked in_progress but never built", state.CurrentPhase),
				)))
				break
			}
		}
	}
	return issues
}

func scanUnreconciledWorkerChangesReadOnly(root, dataDir string, state colony.ColonyState, manifest codexContinueManifest) []HealthIssue {
	status, err := gitOutputAt(root, "status", "--short")
	if err != nil || strings.TrimSpace(status) == "" {
		return nil
	}
	var changed []string
	for _, line := range strings.Split(status, "\n") {
		if len(line) < 3 {
			continue
		}
		path := strings.TrimSpace(line[2:])
		if path == "" {
			continue
		}
		if parts := strings.Split(path, " -> "); len(parts) == 2 {
			changed = append(changed, strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
			continue
		}
		changed = append(changed, path)
	}

	recorded := make(map[string]bool)
	var claims codexBuildClaims
	if raw, err := os.ReadFile(filepath.Join(dataDir, "last-build-claims.json")); err == nil && json.Unmarshal(raw, &claims) == nil {
		for _, path := range append(append(append([]string{}, claims.FilesCreated...), claims.FilesModified...), claims.TestsWritten...) {
			recorded[strings.TrimSpace(path)] = true
		}
		for _, claim := range claims.TaskClaims {
			for _, path := range append(append(append([]string{}, claim.FilesCreated...), claim.FilesModified...), claim.TestsWritten...) {
				recorded[strings.TrimSpace(path)] = true
			}
		}
	}
	if state.CurrentPhase > 0 && manifest.Present {
		for _, dispatch := range manifest.Data.Dispatches {
			for _, path := range dispatch.Outputs {
				recorded[strings.TrimSpace(path)] = true
			}
		}
	}

	var unreconciled []string
	for _, path := range changed {
		if !recorded[path] {
			unreconciled = append(unreconciled, path)
		}
	}
	sort.Strings(unreconciled)
	if len(unreconciled) == 0 {
		return nil
	}
	return []HealthIssue{fixableIssue(issueWarning(
		"unreconciled_worker_changes",
		"git working tree",
		fmt.Sprintf("%d unreconciled file change(s) not recorded by any worker", len(unreconciled)),
	))}
}

func splitMaintenanceEvidencePaths(raw string) []string {
	var paths []string
	for _, path := range strings.Split(raw, ",") {
		path = strings.TrimSpace(path)
		if path != "" && path != "(unavailable)" && !containsString(paths, path) {
			paths = append(paths, filepath.ToSlash(path))
		}
	}
	sort.Strings(paths)
	if paths == nil {
		return []string{}
	}
	return paths
}

func maintenanceRecoveryScanPaths(facts LifecycleFacts) []string {
	dataDir := maintenanceRecoveryDataDir(facts)
	paths := []string{
		filepath.Join(dataDir, "COLONY_STATE.json"),
		filepath.Join(dataDir, "spawn-runs.json"),
		filepath.Join(dataDir, "last-build-claims.json"),
	}
	if facts.State.Value.CurrentPhase > 0 {
		phaseDir := filepath.Join(dataDir, "build", fmt.Sprintf("phase-%d", facts.State.Value.CurrentPhase))
		paths = append(paths, filepath.Join(phaseDir, "manifest.json"), filepath.Join(phaseDir, "continue.json"))
	}
	for i := range paths {
		paths[i] = filepath.ToSlash(paths[i])
	}
	sort.Strings(paths)
	return paths
}

func maintenanceRecoveryEvidenceStatus(issues []HealthIssue) string {
	if len(issues) > 0 {
		return "attention_required"
	}
	return "pass"
}

func renderMaintenanceRecoveryInspection(result maintenanceRecoveryInspectionResult) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("maintenance"), "Recovery Inspection"))
	b.WriteString(visualDividerStr())
	b.WriteString(result.Explanation)
	b.WriteString("\n\n")
	b.WriteString(renderStageMarker("Evidence"))
	for _, source := range result.Provenance {
		fmt.Fprintf(&b, "%s: %s — %s\n", source.Domain, source.Provenance, source.Path)
	}
	b.WriteString("\n")
	b.WriteString(renderStageMarker("Diagnosis"))
	if len(result.Issues) == 0 {
		b.WriteString("No stuck-state condition was found in the available evidence.\n")
	} else {
		for _, issue := range result.Issues {
			fmt.Fprintf(&b, "%s [%s]: %s\n", strings.ToUpper(issue.Severity), issue.Category, issue.Message)
		}
	}
	b.WriteString("State effect: none\n")
	b.WriteString(renderNextUp("Run `aether resume`; it is the only command allowed to restore lifecycle progress."))
	return b.String()
}

func newMaintenanceOperation(platform, id, label, command, mutationClass string, preview, transaction bool, receipt, recovery string) maintenanceOperation {
	return maintenanceOperation{
		OperationID:         id,
		Label:               label,
		RuntimeCommand:      command,
		DisplayCommand:      lifecycleProjectionCommand(command, platform),
		MutationClass:       mutationClass,
		PreviewAvailable:    preview,
		TransactionRequired: transaction,
		ReceiptType:         receipt,
		RecoveryAction:      recovery,
	}
}

func renderMaintenanceLanding(result maintenanceLandingResult) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("maintenance"), "Expert Maintenance"))
	b.WriteString(visualDividerStr())
	b.WriteString(result.Explanation)
	b.WriteString("\n\n")
	renderMaintenanceOperationGroup(&b, "Inspection — no state changes", result.Catalog.Inspection)
	b.WriteString("\n")
	renderMaintenanceOperationGroup(&b, "Mutation — explicit invocation only", result.Catalog.Mutation)
	b.WriteString("\n")
	b.WriteString(renderStageMarker("Focused closeout"))
	if name := strings.TrimSpace(result.Lifecycle.Identity.Value.Name); name != "" {
		fmt.Fprintf(&b, "Colony: %s\n", name)
	}
	if goal := strings.TrimSpace(result.Lifecycle.Goal.Value); goal != "" {
		fmt.Fprintf(&b, "Goal: %s\n", goal)
	}
	fmt.Fprintf(&b, "Standing: %s\n", emptyFallback(strings.TrimSpace(result.Lifecycle.Standing.Value), "UNKNOWN"))
	if count := len(result.Lifecycle.Blockers); count > 0 {
		fmt.Fprintf(&b, "Unresolved blockers: %d\n", count)
	}
	b.WriteString("State effect: none\n")
	if next := strings.TrimSpace(result.Lifecycle.NextAction.DisplayCommand); next != "" {
		b.WriteString(renderNextUp(fmt.Sprintf("The colony's current next action remains `%s`. Select a maintenance operation explicitly only when you need it.", next)))
	} else {
		b.WriteString(renderNextUp("Select a maintenance operation explicitly; opening this catalog did not run one."))
	}
	return b.String()
}

func renderMaintenanceOperationGroup(b *strings.Builder, title string, operations []maintenanceOperation) {
	b.WriteString(renderStageMarker(title))
	for _, operation := range operations {
		fmt.Fprintf(b, "%s — %s\n", operation.OperationID, operation.Label)
		fmt.Fprintf(b, "  Command: %s\n", operation.DisplayCommand)
		fmt.Fprintf(b, "  Contract: %s; preview=%t; transaction=%t; receipt=%s\n", operation.MutationClass, operation.PreviewAvailable, operation.TransactionRequired, operation.ReceiptType)
		fmt.Fprintf(b, "  Recovery: %s\n", operation.RecoveryAction)
	}
}
