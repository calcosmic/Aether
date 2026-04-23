package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

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
	Checks           []integrityCheck `json:"checks"`
	Overall          string           `json:"overall"` // "ok", "warning", "critical"
	RecoveryCommands []string         `json:"recovery_commands,omitempty"`
}

var integrityCmd = &cobra.Command{
	Use:   "integrity",
	Short: "Validate the full release pipeline chain",
	Long:  "Checks source version, binary version, hub version, companion files, and downstream update result. Auto-detects source repo vs consumer repo context.",
	RunE:  runIntegrity,
}

func init() {
	integrityCmd.Flags().Bool("json", false, "Output JSON instead of visual report")
	integrityCmd.Flags().String("channel", "", "Override channel (stable or dev)")
	integrityCmd.Flags().Bool("source", false, "Force source-repo checks")

	rootCmd.AddCommand(integrityCmd)
}

func runIntegrity(cmd *cobra.Command, args []string) error {
	// 1. Determine channel
	channel := runtimeChannelFromFlag(cmd.Flags())
	if explicitChannel, _ := cmd.Flags().GetString("channel"); explicitChannel != "" {
		if normalizeRuntimeChannel(explicitChannel) != channelDev && normalizeRuntimeChannel(explicitChannel) != channelStable {
			return fmt.Errorf("invalid channel %q: must be stable or dev", explicitChannel)
		}
	}

	// 2. Determine context
	ctx := detectIntegrityContext()
	if forceSource, _ := cmd.Flags().GetBool("source"); forceSource {
		ctx = "source"
	}

	// 3. Resolve hub directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}
	hubDir := resolveHubPathForHome(homeDir, channel)
	hubVersionFile := filepath.Join(hubDir, "version.json")
	if _, err := os.Stat(hubVersionFile); os.IsNotExist(err) {
		result := integrityResult{
			Context: ctx,
			Channel: string(channel),
			Checks: []integrityCheck{
				{Name: "Hub installed", Status: "fail", Message: fmt.Sprintf("hub not installed at %s", hubDir)},
			},
			Overall: "critical",
		}
		if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
			data, _ := json.MarshalIndent(result, "", "  ")
			fmt.Fprintln(stdout, string(data))
		} else {
			outputError(2, fmt.Sprintf("hub not installed at %s", hubDir), nil)
		}
		return fmt.Errorf("hub not installed at %s", hubDir)
	}

	// 4. Collect versions
	binaryVersion := resolveVersion()
	hubVersion := readHubVersionAtPath(hubDir)

	// 5. Run checks based on context
	var checks []integrityCheck
	if ctx == "source" {
		checks = []integrityCheck{
			checkSourceVersion(),
			checkBinaryVersion(),
			checkHubVersion(hubDir),
			checkHubCompanionFiles(hubDir),
			checkDownstreamSimulation(hubDir, hubVersion, binaryVersion, channel),
		}
	} else {
		checks = []integrityCheck{
			checkBinaryVersion(),
			checkHubVersion(hubDir),
			checkHubCompanionFiles(hubDir),
			checkDownstreamSimulation(hubDir, hubVersion, binaryVersion, channel),
		}
	}

	// 6. Aggregate results
	overall := "ok"
	var recoveryCommands []string
	for _, c := range checks {
		if c.Status == "fail" {
			overall = "critical"
			if c.RecoveryCommand != "" {
				recoveryCommands = append(recoveryCommands, c.RecoveryCommand)
			}
		}
	}

	result := integrityResult{
		Context:          ctx,
		Channel:          string(channel),
		Checks:           checks,
		Overall:          overall,
		RecoveryCommands: recoveryCommands,
	}

	// 7. Render output
	if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Fprintln(stdout, string(data))
	} else {
		visual := buildIntegrityVisual(result)
		fmt.Fprint(stdout, visual)
	}

	// 8. Return
	if overall == "ok" {
		return nil
	}
	return fmt.Errorf("integrity checks failed")
}

func buildIntegrityVisual(result integrityResult) string {
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

	if len(result.RecoveryCommands) > 0 {
		b.WriteString("\nRecovery Commands\n")
		for _, rc := range result.RecoveryCommands {
			b.WriteString(fmt.Sprintf("  %s\n", rc))
		}
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
		{"skills/codex/", filepath.Join(hubSystem, "skills-codex"), expectedCodexSkillCount, nil, true},
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
