package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var porterCmd = &cobra.Command{
	Use:   "porter",
	Short: "Deliver colony work to the outside world",
	Long:  "Porter handles delivery of colony work: validate pipeline readiness, publish to the hub, and push releases.",
}

var porterCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate pipeline readiness for delivery",
	Long: "Runs a full pipeline readiness check including version alignment, " +
		"hub companion files, git status, test status, and changelog completeness. " +
		"Reuses existing integrity check functions. Works with or without an active colony.",
	Args: cobra.NoArgs,
	RunE: runPorterCheck,
}

func init() {
	porterCheckCmd.Flags().Bool("json", false, "Output JSON instead of visual report")
	porterCheckCmd.Flags().String("channel", "", "Override channel (stable or dev)")
	porterCmd.AddCommand(porterCheckCmd)
	rootCmd.AddCommand(porterCmd)
}

func runPorterCheck(cmd *cobra.Command, args []string) error {
	channel := runtimeChannelFromFlag(cmd.Flags())

	// Resolve hub directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}
	hubDir := resolveHubPathForHome(homeDir, channel)

	// Check if hub exists (skip hub-dependent checks if not installed)
	hubVersionFile := hubDir + "/version.json"
	if _, statErr := os.Stat(hubVersionFile); os.IsNotExist(statErr) {
		// Still run non-hub checks
		checks := []integrityCheck{
			checkSourceVersion(),
			checkBinaryVersion(),
			checkGitStatus(),
			checkTestStatus(),
			checkChangelogCompleteness(),
		}
		overall := "critical"
		recoveryCommands := []string{"Run aether install to populate the hub"}
		for _, c := range checks {
			if c.Status == "fail" && c.RecoveryCommand != "" {
				recoveryCommands = append(recoveryCommands, c.RecoveryCommand)
			}
		}
		result := integrityResult{
			Context:          "porter",
			Channel:          string(channel),
			Checks:           checks,
			Overall:          overall,
			RecoveryCommands: recoveryCommands,
		}
		return renderPorterResult(cmd, result)
	}

	checks := buildPorterChecks(string(channel), false)

	// Aggregate results
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
		Context:          "porter",
		Channel:          string(channel),
		Checks:           checks,
		Overall:          overall,
		RecoveryCommands: recoveryCommands,
	}

	return renderPorterResult(cmd, result)
}

// buildPorterChecks constructs the full set of porter checks, reusing integrity
// functions and adding delivery-specific checks. When skipTests is true, the
// expensive go test subprocess is replaced with a skip-status placeholder.
func buildPorterChecks(channel string, skipTests bool) []integrityCheck {
	// Resolve hub directory
	homeDir, _ := os.UserHomeDir()
	rc := normalizeRuntimeChannel(channel)
	hubDir := resolveHubPathForHome(homeDir, rc)
	hubVersion := readHubVersionAtPath(hubDir)
	binaryVersion := resolveVersion()

	var testCheck integrityCheck
	if skipTests {
		testCheck = integrityCheck{
			Name:   "Test status",
			Status: "skip",
			Message: "Skipped (test mode)",
		}
	} else {
		testCheck = checkTestStatus()
	}

	return []integrityCheck{
		checkSourceVersion(),
		checkBinaryVersion(),
		checkHubVersion(hubDir),
		checkHubCompanionFiles(hubDir),
		checkDownstreamSimulation(hubDir, hubVersion, binaryVersion, rc),
		checkGitStatus(),
		testCheck,
		checkChangelogCompleteness(),
	}
}

func renderPorterResult(cmd *cobra.Command, result integrityResult) error {
	if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Fprintln(stdout, string(data))
	} else {
		visual := buildPorterVisual(result)
		fmt.Fprint(stdout, visual)
	}

	if result.Overall == "ok" {
		return nil
	}
	return fmt.Errorf("porter checks failed")
}

func buildPorterVisual(result integrityResult) string {
	var b strings.Builder
	b.WriteString(renderBanner("\U0001f4e6", "Porter Delivery Readiness"))
	b.WriteString(fmt.Sprintf("Channel: %s\n\n", result.Channel))

	passCount := 0
	for _, c := range result.Checks {
		if c.Status == "pass" {
			passCount++
			b.WriteString(fmt.Sprintf("  %s %s: %s\n", "\u2713", c.Name, c.Status))
			if msg := strings.TrimSpace(c.Message); msg != "" {
				b.WriteString(fmt.Sprintf("    %s\n", msg))
			}
		} else if c.Status == "skip" {
			b.WriteString(fmt.Sprintf("  %s %s: %s\n", "\u25cb", c.Name, c.Status))
			if msg := strings.TrimSpace(c.Message); msg != "" {
				b.WriteString(fmt.Sprintf("    %s\n", msg))
			}
		} else {
			b.WriteString(fmt.Sprintf("  %s %s: %s\n", "\u2717", c.Name, c.Status))
			if msg := strings.TrimSpace(c.Message); msg != "" {
				b.WriteString(fmt.Sprintf("    %s\n", msg))
			}
			if c.RecoveryCommand != "" {
				b.WriteString(fmt.Sprintf("    Recovery: %s\n", c.RecoveryCommand))
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

// checkGitStatus runs `git status --porcelain` and fails if uncommitted changes exist.
func checkGitStatus() integrityCheck {
	out, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil {
		return integrityCheck{
			Name:            "Git status",
			Status:          "skip",
			Message:         "Not in a git repository",
			RecoveryCommand: "",
		}
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	// Filter empty lines (from trimming)
	changed := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			changed++
		}
	}
	if changed == 0 {
		return integrityCheck{
			Name:    "Git status",
			Status:  "pass",
			Message: "Working tree clean",
		}
	}
	return integrityCheck{
		Name:            "Git status",
		Status:          "fail",
		Message:         fmt.Sprintf("%d uncommitted changes", changed),
		RecoveryCommand: "Commit or stash changes before delivery",
	}
}

// checkTestStatus runs `go test ./...` and fails if any test fails.
func checkTestStatus() integrityCheck {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "test", "./...", "-count=1")
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return integrityCheck{
			Name:            "Test status",
			Status:          "fail",
			Message:         "Tests timed out after 120s",
			RecoveryCommand: "Fix slow or hanging tests before delivery",
		}
	}
	if err != nil {
		// Extract last few lines of output for the message
		lines := strings.Split(string(output), "\n")
		lastLines := lines
		if len(lines) > 5 {
			lastLines = lines[len(lines)-5:]
		}
		return integrityCheck{
			Name:            "Test status",
			Status:          "fail",
			Message:         strings.Join(lastLines, "\n"),
			RecoveryCommand: "Fix failing tests before delivery",
		}
	}
	return integrityCheck{
		Name:    "Test status",
		Status:  "pass",
		Message: "All tests pass",
	}
}

// checkChangelogCompleteness reads CHANGELOG.md and checks for an entry matching
// the current source version.
func checkChangelogCompleteness() integrityCheck {
	version := resolveSourceVersion()
	if version == "unknown" {
		return integrityCheck{
			Name:   "Changelog completeness",
			Status: "skip",
			Message: "Source version unknown, cannot verify changelog",
		}
	}

	data, err := os.ReadFile("CHANGELOG.md")
	if err != nil {
		if os.IsNotExist(err) {
			return integrityCheck{
				Name:            "Changelog completeness",
				Status:          "fail",
				Message:         "CHANGELOG.md not found",
				RecoveryCommand: "Add changelog entry for current version",
			}
		}
		return integrityCheck{
			Name:            "Changelog completeness",
			Status:          "skip",
			Message:         fmt.Sprintf("Cannot read CHANGELOG.md: %v", err),
		}
	}

	content := string(data)
	// Strip "v" prefix from version for matching
	searchVersion := strings.TrimPrefix(version, "v")
	if strings.Contains(content, searchVersion) || strings.Contains(content, version) {
		return integrityCheck{
			Name:    "Changelog completeness",
			Status:  "pass",
			Message: fmt.Sprintf("Entry found for %s", version),
		}
	}
	return integrityCheck{
		Name:            "Changelog completeness",
		Status:          "fail",
		Message:         fmt.Sprintf("No changelog entry found for %s", version),
		RecoveryCommand: "Add changelog entry for current version",
	}
}

// buildPorterReadinessSummary creates a short post-seal readiness summary
// showing version alignment and git status for all platforms including Codex.
func buildPorterReadinessSummary() string {
	sourceVersion := resolveSourceVersion()
	binaryVersion := resolveVersion()
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	channel := resolveRuntimeChannel()
	hubDir := resolveHubPathForHome(homeDir, channel)
	hubVersion := readHubVersionAtPath(hubDir)

	// Return empty if not in a source repo context
	if sourceVersion == "unknown" && binaryVersion == "unknown" {
		return ""
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("  Source version: %s\n", formatVersionStatus(sourceVersion, binaryVersion)))
	b.WriteString(fmt.Sprintf("  Binary version: %s\n", formatVersionStatus(binaryVersion, sourceVersion)))
	b.WriteString(fmt.Sprintf("  Hub version:   %s\n", formatVersionStatus(hubVersion, sourceVersion)))

	gitCheck := checkGitStatus()
	if gitCheck.Status == "pass" {
		b.WriteString("  Git status:    clean\n")
	} else if gitCheck.Status == "fail" {
		b.WriteString(fmt.Sprintf("  Git status:    %s\n", gitCheck.Message))
	}

	return b.String()
}

// formatVersionStatus returns the version with a colored indicator:
// green checkmark if it matches the reference, red X if not.
func formatVersionStatus(version, reference string) string {
	if version == "" || version == "unknown" {
		return "unknown"
	}
	if reference != "" && reference != "unknown" && version == reference {
		if shouldUseANSIColors() {
			return "\x1b[32m" + version + " \u2713\x1b[0m"
		}
		return version + " \u2713"
	}
	return version
}
