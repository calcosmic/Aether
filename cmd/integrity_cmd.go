package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

const integrityInspectionSchemaVersion = "integrity-result/v1"

type integrityCheck struct {
	Name            string                 `json:"name"`
	Status          string                 `json:"status"` // "pass", "fail", "skip"
	Message         string                 `json:"message"`
	RecoveryCommand string                 `json:"recovery_command,omitempty"`
	Details         map[string]interface{} `json:"details,omitempty"`
}

type integrityResult struct {
	Context          string           `json:"context"` // "source" or "consumer"
	Channel          string           `json:"channel"`
	Scope            string           `json:"scope,omitempty"`
	GeneratedAt      string           `json:"generated_at,omitempty"`
	Checks           []integrityCheck `json:"checks"`
	Overall          string           `json:"overall"` // "ok", "warning", "critical"
	Evidence         string           `json:"evidence,omitempty"`
	EvidenceError    string           `json:"evidence_error,omitempty"`
	SkippedChecks    []string         `json:"skipped_checks,omitempty"`
	RecoveryCommands []string         `json:"recovery_commands,omitempty"`
}

type integrityInspectionResult struct {
	SchemaVersion    string                            `json:"schema_version"`
	OperationID      string                            `json:"operation_id"`
	Context          string                            `json:"context"`
	Channel          string                            `json:"channel"`
	Checks           []integrityCheck                  `json:"checks"`
	Overall          string                            `json:"overall"`
	Findings         []maintenanceInspectionFinding    `json:"findings"`
	Evidence         []maintenanceInspectionEvidence   `json:"evidence"`
	Verification     maintenanceInspectionVerification `json:"verification"`
	StateEffect      colony.LifecycleStateEffect       `json:"state_effect"`
	Blockers         []maintenanceInspectionFinding    `json:"blockers"`
	NextAction       string                            `json:"next_action"`
	RecoveryCommands []string                          `json:"recovery_commands,omitempty"`
}

var integrityCmd = &cobra.Command{
	Use:         "integrity",
	Short:       "Validate the full release pipeline chain",
	Long:        "Checks source version, binary version, hub version, companion files, and downstream update result. Auto-detects source repo vs consumer repo context.",
	Annotations: map[string]string{"aether.io/read-only": "true", "aether.io/store-free": "true"},
	RunE:        runIntegrity,
}

func init() {
	integrityCmd.Flags().Bool("json", false, "Output JSON instead of visual report")
	integrityCmd.Flags().String("channel", "", "Override channel (stable or dev)")
	integrityCmd.Flags().Bool("source", false, "Force source-repo checks")

	rootCmd.AddCommand(integrityCmd)
}

func runIntegrity(cmd *cobra.Command, args []string) error {
	channel := runtimeChannelFromFlag(cmd.Flags())
	if explicitChannel, _ := cmd.Flags().GetString("channel"); explicitChannel != "" {
		if normalizeRuntimeChannel(explicitChannel) != channelDev && normalizeRuntimeChannel(explicitChannel) != channelStable {
			result := finalizeIntegrityInspection(
				detectIntegrityContext(),
				explicitChannel,
				"",
				[]integrityCheck{{
					Name:            "Channel",
					Status:          "fail",
					Message:         fmt.Sprintf("invalid channel %q: must be stable or dev", explicitChannel),
					RecoveryCommand: "aether integrity --channel stable",
				}},
			)
			if err := renderIntegrityResult(cmd, result); err != nil {
				return err
			}
			return fmt.Errorf("invalid channel %q: must be stable or dev", explicitChannel)
		}
	}

	ctx := detectIntegrityContext()
	if forceSource, _ := cmd.Flags().GetBool("source"); forceSource {
		ctx = "source"
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		result := finalizeIntegrityInspection(ctx, string(channel), "", []integrityCheck{{
			Name:            "Hub location",
			Status:          "fail",
			Message:         fmt.Sprintf("cannot determine home directory: %v", err),
			RecoveryCommand: "Set HOME or AETHER_HUB_DIR, then rerun aether integrity",
		}})
		if renderErr := renderIntegrityResult(cmd, result); renderErr != nil {
			return renderErr
		}
		return fmt.Errorf("cannot determine home directory: %w", err)
	}
	hubDir := resolveHubPathForHome(homeDir, channel)
	result := buildIntegrityInspection(ctx, channel, hubDir)
	if err := renderIntegrityResult(cmd, result); err != nil {
		return err
	}
	if result.Overall == "ok" {
		return nil
	}
	if readHubVersionAtPath(hubDir) == "" {
		return fmt.Errorf("hub not installed at %s", hubDir)
	}
	return fmt.Errorf("integrity checks failed")
}

func buildIntegrityInspection(ctx string, channel runtimeChannel, hubDir string) integrityInspectionResult {
	hubVersion := readHubVersionAtPath(hubDir)
	if hubVersion == "" {
		return finalizeIntegrityInspection(ctx, string(channel), hubDir, []integrityCheck{{
			Name:            "Hub installed",
			Status:          "fail",
			Message:         fmt.Sprintf("hub not installed at %s", hubDir),
			RecoveryCommand: "aether install",
		}})
	}

	binaryVersion := resolveVersion()
	checks := []integrityCheck{}
	if ctx == "source" {
		checks = append(checks, checkSourceVersion())
	}
	checks = append(checks,
		checkBinaryVersion(),
		checkHubVersion(hubDir),
		checkHubCompanionFiles(hubDir),
		checkDownstreamSimulation(hubDir, hubVersion, binaryVersion, channel),
	)
	return finalizeIntegrityInspection(ctx, string(channel), hubDir, checks)
}

func finalizeIntegrityInspection(ctx, channel, hubDir string, checks []integrityCheck) integrityInspectionResult {
	result := integrityInspectionResult{
		SchemaVersion: integrityInspectionSchemaVersion,
		OperationID:   "integrity.inspect",
		Context:       ctx,
		Channel:       channel,
		Checks:        append([]integrityCheck(nil), checks...),
		Overall:       "ok",
		Findings:      []maintenanceInspectionFinding{},
		Evidence:      []maintenanceInspectionEvidence{},
		StateEffect:   colony.LifecycleStateEffectNone,
		Blockers:      []maintenanceInspectionFinding{},
		NextAction:    "aether maintenance",
	}
	if result.Checks == nil {
		result.Checks = []integrityCheck{}
	}
	for _, check := range result.Checks {
		paths := integrityEvidencePaths(check.Name, ctx, hubDir)
		result.Evidence = append(result.Evidence, maintenanceInspectionEvidence{
			Scope:   check.Name,
			Paths:   paths,
			Checked: 1,
			Status:  check.Status,
		})
		if check.Status != "fail" {
			continue
		}
		result.Overall = "critical"
		finding := maintenanceInspectionFinding{
			Code:            "integrity." + strings.ReplaceAll(strings.ToLower(check.Name), " ", "_"),
			Summary:         check.Message,
			EvidencePaths:   append([]string(nil), paths...),
			RecoveryCommand: check.RecoveryCommand,
		}
		if len(paths) > 0 {
			finding.SourcePath = paths[0]
		}
		result.Findings = append(result.Findings, finding)
		result.Blockers = append(result.Blockers, finding)
		if check.RecoveryCommand != "" && !containsString(result.RecoveryCommands, check.RecoveryCommand) {
			result.RecoveryCommands = append(result.RecoveryCommands, check.RecoveryCommand)
		}
	}
	if len(result.RecoveryCommands) > 0 {
		result.NextAction = result.RecoveryCommands[0]
	}
	status := "pass"
	if result.Overall != "ok" {
		status = "fail"
	}
	result.Verification = maintenanceInspectionVerification{
		Status:        status,
		EvidenceCount: len(result.Evidence),
		FindingCount:  len(result.Findings),
	}
	return result
}

func integrityEvidencePaths(name, ctx, hubDir string) []string {
	var paths []string
	switch name {
	case "Hub installed", "Hub version", "Downstream simulation":
		if hubDir != "" {
			paths = append(paths, hubDir)
		}
	case "Hub companion files":
		if hubDir != "" {
			paths = append(paths, filepath.Join(hubDir, "system"))
		}
	case "Source version":
		if cwd, err := os.Getwd(); err == nil {
			if root := findAetherModuleRoot(cwd); root != "" {
				paths = append(paths, filepath.Join(root, ".aether", "version.json"))
			}
		}
	case "Binary version":
		if executable, err := os.Executable(); err == nil {
			paths = append(paths, executable)
		}
	}
	if ctx == "source" && name == "Hub installed" {
		if cwd, err := os.Getwd(); err == nil {
			if root := findAetherModuleRoot(cwd); root != "" {
				paths = append(paths, filepath.Join(root, ".aether", "version.json"))
			}
		}
	}
	sort.Strings(paths)
	return paths
}

func renderIntegrityResult(cmd *cobra.Command, result integrityInspectionResult) error {
	if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Fprintln(stdout, string(data))
		return nil
	}
	visualFprint(stdout, buildIntegrityVisual(result))
	return nil
}

func buildIntegrityVisual(result integrityInspectionResult) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("integrity"), "Release Integrity"))
	b.WriteString(fmt.Sprintf("Context: %s repo\n", result.Context))
	b.WriteString(fmt.Sprintf("Channel: %s\n\n", result.Channel))

	passCount := 0
	for _, c := range result.Checks {
		if c.Status == "pass" {
			passCount++
			b.WriteString(fmt.Sprintf("✓ %s: %s\n", c.Name, c.Status))
			if msg := strings.TrimSpace(c.Message); msg != "" {
				b.WriteString(fmt.Sprintf("  Version: %s\n", msg))
			}
		} else {
			b.WriteString(fmt.Sprintf("✗ %s: %s\n", c.Name, c.Status))
			if msg := strings.TrimSpace(c.Message); msg != "" {
				b.WriteString(fmt.Sprintf("  Message: %s\n", msg))
			}
			if c.RecoveryCommand != "" {
				b.WriteString(fmt.Sprintf("  Recovery: %s\n", c.RecoveryCommand))
			}
		}
	}

	b.WriteString("\n")
	b.WriteString(renderStageMarker("Summary"))
	b.WriteString(fmt.Sprintf("%d/%d checks passed\n", passCount, len(result.Checks)))
	b.WriteString("State effect: none\n")

	if len(result.RecoveryCommands) > 0 {
		b.WriteString("\nRecovery Commands\n")
		for _, rc := range result.RecoveryCommands {
			b.WriteString(fmt.Sprintf("  %s\n", rc))
		}
	}
	if strings.TrimSpace(result.NextAction) != "" {
		b.WriteString(fmt.Sprintf("\nNext command: %s\n", result.NextAction))
	}

	return b.String()
}

func detectIntegrityContext() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "consumer"
	}
	root := findAetherModuleRoot(cwd)
	if root == "" {
		return "consumer"
	}
	if _, err := os.Stat(filepath.Join(root, "cmd", "aether", "main.go")); err != nil {
		return "consumer"
	}
	if _, err := os.Stat(filepath.Join(root, ".aether", "version.json")); err != nil {
		return "consumer"
	}
	return "source"
}

func resolveSourceVersion() string {
	if v := readRepoVersion(""); v != "" {
		return v
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "unknown"
	}
	root := findAetherModuleRoot(cwd)
	if root == "" {
		return "unknown"
	}
	if v := readHubVersionAtPath(root); v != "" {
		return normalizeVersion(v)
	}
	return "unknown"
}

func checkBinaryVersion() integrityCheck {
	binaryVersion := resolveVersion()
	if binaryVersion != "unknown" && binaryVersion != "" {
		return integrityCheck{
			Name:    "Binary version",
			Status:  "pass",
			Message: binaryVersion,
			Details: map[string]interface{}{"version": binaryVersion},
		}
	}
	return integrityCheck{
		Name:            "Binary version",
		Status:          "fail",
		Message:         "Binary version could not be resolved",
		RecoveryCommand: "Rebuild the binary: go build ./cmd/aether",
	}
}

func checkHubVersion(hubDir string) integrityCheck {
	hubVersion := readHubVersionAtPath(hubDir)
	if hubVersion != "" {
		return integrityCheck{
			Name:    "Hub version",
			Status:  "pass",
			Message: hubVersion,
			Details: map[string]interface{}{"version": hubVersion},
		}
	}
	return integrityCheck{
		Name:            "Hub version",
		Status:          "fail",
		Message:         "Hub version could not be determined",
		RecoveryCommand: "Run aether install to populate the hub",
	}
}

func checkSourceVersion() integrityCheck {
	sourceVersion := resolveSourceVersion()
	if sourceVersion != "unknown" {
		return integrityCheck{
			Name:    "Source version",
			Status:  "pass",
			Message: sourceVersion,
			Details: map[string]interface{}{"version": sourceVersion},
		}
	}
	return integrityCheck{
		Name:            "Source version",
		Status:          "fail",
		Message:         "Source version could not be determined",
		RecoveryCommand: "Ensure .aether/version.json exists in the repo root",
	}
}

func checkHubCompanionFiles(hubDir string) integrityCheck {
	hubSystem := filepath.Join(hubDir, "system")
	checks := []struct {
		name      string
		path      string
		expected  int
		filter    func(string) bool
		recursive bool
	}{
		{"commands/claude/", filepath.Join(hubSystem, "commands", "claude"), expectedClaudeCommandCount, nil, false},
		{"commands/opencode/", filepath.Join(hubSystem, "commands", "opencode"), expectedOpenCodeCommandCount, nil, false},
		{"agents/opencode/", filepath.Join(hubSystem, "agents"), expectedOpenCodeAgentCount, nil, false},
		{"agents/codex/", filepath.Join(hubSystem, "codex"), expectedCodexAgentCount, func(name string) bool { return strings.HasSuffix(name, ".toml") }, false},
		{"skills/hub/", filepath.Join(hubSystem, "skills"), expectedCodexSkillCount, nil, true},
	}

	var discrepancies []string
	for _, c := range checks {
		var actual int
		if c.recursive {
			actual = countEntriesRecursive(c.path, c.filter)
		} else {
			actual = countEntriesInDir(c.path, c.filter)
		}
		if actual < c.expected {
			discrepancies = append(discrepancies, fmt.Sprintf("%s has %d files (expected %d)", c.name, actual, c.expected))
		}
	}

	if len(discrepancies) == 0 {
		return integrityCheck{
			Name:    "Hub companion files",
			Status:  "pass",
			Message: "All companion file directories match expected counts",
		}
	}
	return integrityCheck{
		Name:            "Hub companion files",
		Status:          "fail",
		Message:         strings.Join(discrepancies, "; "),
		RecoveryCommand: "Run aether install to refresh companion files",
	}
}

func checkDownstreamSimulation(hubDir, hubVersion, binaryVersion string, channel runtimeChannel) integrityCheck {
	result := checkStalePublish(hubDir, hubVersion, binaryVersion, channel, []map[string]interface{}{})
	switch result.Classification {
	case staleOK:
		return integrityCheck{
			Name:    "Downstream simulation",
			Status:  "pass",
			Message: result.Message,
		}
	case staleInfo:
		return integrityCheck{
			Name:            "Downstream simulation",
			Status:          "fail",
			Message:         result.Message,
			RecoveryCommand: result.RecoveryCommand,
		}
	case staleWarning:
		return integrityCheck{
			Name:            "Downstream simulation",
			Status:          "fail",
			Message:         result.Message,
			RecoveryCommand: result.RecoveryCommand,
		}
	case staleCritical:
		return integrityCheck{
			Name:            "Downstream simulation",
			Status:          "fail",
			Message:         result.Message,
			RecoveryCommand: result.RecoveryCommand,
		}
	default:
		return integrityCheck{
			Name:            "Downstream simulation",
			Status:          "fail",
			Message:         "Unknown stale publish classification",
			RecoveryCommand: result.RecoveryCommand,
		}
	}
}
