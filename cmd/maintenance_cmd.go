package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

const maintenanceCatalogSchemaVersion = "maintenance-catalog/v1"

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

var maintenanceCmd = &cobra.Command{
	Use:         "maintenance",
	Short:       "Inspect or repair Aether internals with preview and rollback.",
	Args:        cobra.NoArgs,
	Annotations: map[string]string{"aether.io/read-only": "true"},
	RunE:        runMaintenanceLanding,
}

func init() {
	rootCmd.AddCommand(maintenanceCmd)
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
