package cmd

import (
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
	return fmt.Errorf("not yet implemented")
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
		name     string
		path     string
		expected int
		filter   func(string) bool
	}{
		{"commands/claude/", filepath.Join(hubSystem, "commands", "claude"), expectedClaudeCommandCount, nil},
		{"commands/opencode/", filepath.Join(hubSystem, "commands", "opencode"), expectedOpenCodeCommandCount, nil},
		{"agents/opencode/", filepath.Join(hubSystem, "agents"), expectedOpenCodeAgentCount, nil},
		{"agents/codex/", filepath.Join(hubSystem, "codex"), expectedCodexAgentCount, func(name string) bool { return strings.HasSuffix(name, ".toml") }},
		{"skills/codex/", filepath.Join(hubSystem, "skills-codex"), expectedCodexSkillCount, nil},
	}

	var discrepancies []string
	for _, c := range checks {
		actual := countEntriesInDir(c.path, c.filter)
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
