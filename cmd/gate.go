package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
	"github.com/spf13/cobra"
)

// Gate checking prevents invalid state transitions.
// A gate-check verifies preconditions (tests passing, no critical flags, etc.)
// before allowing a task to be marked complete or a phase to advance.

type gateCheck struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail,omitempty"`
}

type gateResult struct {
	Allowed bool        `json:"allowed"`
	Reason  string      `json:"reason,omitempty"`
	Checks  []gateCheck `json:"checks"`
}

var gateCheckCmd = &cobra.Command{
	Use:   "gate-check",
	Short: "Validate whether a state transition is allowed",
	Long: `Check preconditions before allowing a task completion or phase advancement.
Runs verification checks (tests, flags, coverage) and returns a JSON result
indicating whether the transition is allowed and why.`,
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		action := mustGetString(cmd, "action")
		if action == "" {
			return nil
		}

		switch action {
		case "task-complete":
			return checkTaskComplete(cmd)
		case "phase-advance":
			return checkPhaseAdvance(cmd)
		default:
			outputError(1, fmt.Sprintf("unknown action %q: must be task-complete or phase-advance", action), nil)
			return nil
		}
	},
}

func checkTaskComplete(cmd *cobra.Command) error {
	taskID := mustGetString(cmd, "task")
	if taskID == "" {
		return nil
	}

	var checks []gateCheck

	// Check 1: Tests pass
	testCheck := checkTestsPass()
	checks = append(checks, testCheck)

	// Check 2: No critical flags
	flagCheck := checkNoCriticalFlags()
	checks = append(checks, flagCheck)

	// Determine overall result
	allPassed := true
	var reasons []string
	for _, c := range checks {
		if !c.Passed {
			allPassed = false
			reasons = append(reasons, c.Detail)
		}
	}

	result := gateResult{
		Allowed: allPassed,
		Checks:  checks,
	}
	if !allPassed {
		result.Reason = strings.Join(reasons, "; ")
	}

	outputOK(result)
	return nil
}

func checkPhaseAdvance(cmd *cobra.Command) error {
	phaseNum := mustGetInt(cmd, "phase")
	if phaseNum == 0 {
		return nil
	}

	var checks []gateCheck

	// Check 1: All tasks in the phase are completed
	taskCheck := checkAllTasksCompleted(phaseNum)
	checks = append(checks, taskCheck)

	// Check 2: Tests pass
	testCheck := checkTestsPass()
	checks = append(checks, testCheck)

	// Check 3: No critical flags
	flagCheck := checkNoCriticalFlags()
	checks = append(checks, flagCheck)

	// Determine overall result
	allPassed := true
	var reasons []string
	for _, c := range checks {
		if !c.Passed {
			allPassed = false
			reasons = append(reasons, c.Detail)
		}
	}

	result := gateResult{
		Allowed: allPassed,
		Checks:  checks,
	}
	if !allPassed {
		result.Reason = strings.Join(reasons, "; ")
	}

	outputOK(result)
	return nil
}

// checkTestsPass runs the project test command and checks if all tests pass.
// It looks for the test command from CLAUDE.md, CODEBASE.md, or language defaults.
func checkTestsPass() gateCheck {
	// Try to find a test command
	testCmd := resolveTestCommand()
	if testCmd == "" {
		// No test command found — pass by default (no tests to run)
		return gateCheck{
			Name:   "tests_pass",
			Passed: true,
			Detail: "no test command found, skipping",
		}
	}

	// Run the test command
	parts := strings.Fields(testCmd)
	if len(parts) == 0 {
		return gateCheck{
			Name:   "tests_pass",
			Passed: true,
			Detail: "empty test command, skipping",
		}
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir = storage.ResolveAetherRoot(context.Background())
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Test command failed — extract summary from output
		detail := "test command failed"
		outputStr := string(output)
		if outputStr != "" {
			// Try to extract a useful summary line
			lines := strings.Split(outputStr, "\n")
			for _, line := range lines {
				if strings.Contains(line, "FAIL") || strings.Contains(line, "failed") || strings.Contains(line, "error") {
					detail = strings.TrimSpace(line)
					if len(detail) > 200 {
						detail = detail[:200] + "..."
					}
					break
				}
			}
		}
		return gateCheck{
			Name:   "tests_pass",
			Passed: false,
			Detail: detail,
		}
	}

	return gateCheck{
		Name:   "tests_pass",
		Passed: true,
		Detail: "all tests passed",
	}
}

// resolveTestCommand determines the test command for the current project.
// Priority: CLAUDE.md → CODEBASE.md → language detection → empty (skip).
func resolveTestCommand() string {
	// Check CLAUDE.md for test command
	repoRoot := storage.ResolveAetherRoot(context.Background())
	claudeMD := repoRoot + "/CLAUDE.md"
	if data, err := os.ReadFile(claudeMD); err == nil {
		cmd := extractTestCommand(string(data))
		if cmd != "" {
			return cmd
		}
	}

	// Check CODEBASE.md
	codebaseMD := repoRoot + "/.aether/data/codebase.md"
	if data, err := os.ReadFile(codebaseMD); err == nil {
		cmd := extractTestCommand(string(data))
		if cmd != "" {
			return cmd
		}
	}

	// Language detection fallback
	if _, err := os.Stat(repoRoot + "/go.mod"); err == nil {
		return "go test ./..."
	}
	if _, err := os.Stat(repoRoot + "/package.json"); err == nil {
		return "npm test"
	}
	if _, err := os.Stat(repoRoot + "/Cargo.toml"); err == nil {
		return "cargo test"
	}
	if _, err := os.Stat(repoRoot + "/pom.xml"); err == nil {
		return "mvn test"
	}

	return ""
}

// extractTestCommand scans markdown content for a test command reference.
func extractTestCommand(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		// Look for common patterns
		if strings.Contains(line, "go test") && !strings.HasPrefix(strings.TrimSpace(line), "#") {
			// Extract just the command
			if idx := strings.Index(line, "go test"); idx >= 0 {
				cmd := line[idx:]
				// Trim at comment or end of useful content
				if ci := strings.Index(cmd, "#"); ci > 0 {
					cmd = cmd[:ci]
				}
				if ci := strings.Index(cmd, "//"); ci > 0 {
					cmd = cmd[:ci]
				}
				return strings.TrimSpace(cmd)
			}
		}
		if strings.Contains(line, "npm test") {
			return "npm test"
		}
		if strings.Contains(line, "cargo test") {
			return "cargo test"
		}
	}
	return ""
}

// checkNoCriticalFlags checks for CRITICAL severity error records in the colony state.
func checkNoCriticalFlags() gateCheck {
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		return gateCheck{
			Name:   "no_critical_flags",
			Passed: true,
			Detail: "no state file found, skipping flag check",
		}
	}

	// Check for critical severity error records
	criticalCount := 0
	for _, record := range state.Errors.Records {
		if strings.EqualFold(record.Severity, "CRITICAL") {
			criticalCount++
		}
	}

	if criticalCount > 0 {
		return gateCheck{
			Name:   "no_critical_flags",
			Passed: false,
			Detail: fmt.Sprintf("%d critical error record(s) found", criticalCount),
		}
	}

	return gateCheck{
		Name:   "no_critical_flags",
		Passed: true,
		Detail: "no critical flags",
	}
}

// checkAllTasksCompleted verifies that all tasks in a phase have completed status.
func checkAllTasksCompleted(phaseNum int) gateCheck {
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		return gateCheck{
			Name:   "all_tasks_completed",
			Passed: false,
			Detail: "COLONY_STATE.json not found",
		}
	}

	// Find the phase
	var phase *colony.Phase
	for i := range state.Plan.Phases {
		if state.Plan.Phases[i].ID == phaseNum {
			phase = &state.Plan.Phases[i]
			break
		}
	}
	if phase == nil {
		return gateCheck{
			Name:   "all_tasks_completed",
			Passed: false,
			Detail: fmt.Sprintf("phase %d not found", phaseNum),
		}
	}

	total := len(phase.Tasks)
	completed := 0
	pending := []string{}
	for _, t := range phase.Tasks {
		if t.Status == "completed" {
			completed++
		} else {
			taskID := "unknown"
			if t.ID != nil {
				taskID = *t.ID
			}
			pending = append(pending, taskID)
		}
	}

	if completed == total {
		return gateCheck{
			Name:   "all_tasks_completed",
			Passed: true,
			Detail: fmt.Sprintf("all %d tasks completed", total),
		}
	}

	return gateCheck{
		Name:   "all_tasks_completed",
		Passed: false,
		Detail: fmt.Sprintf("%d/%d tasks completed, pending: %s", completed, total, strings.Join(pending, ", ")),
	}
}

// runPreBuildGates checks preconditions before dispatching a build.
// Returns an error with the specific gate name if any check fails.
// Note: Phase state validation is handled by validateCodexBuildState;
// this gate focuses on critical flags/blockers.
func runPreBuildGates(dataDir string, phase int) error {
	flagCheck := checkNoCriticalFlags()
	if !flagCheck.Passed {
		return fmt.Errorf("pre-build gate %q failed: %s", flagCheck.Name, flagCheck.Detail)
	}
	return nil
}

// runPreContinueGates checks preconditions before continuing a phase.
// Returns an error with the specific gate name if any check fails.
// Note: Phase state validation is handled by runCodexContinue;
// this gate focuses on critical flags/blockers.
func runPreContinueGates(dataDir string, phase int) error {
	flagCheck := checkNoCriticalFlags()
	if !flagCheck.Passed {
		return fmt.Errorf("pre-continue gate %q failed: %s", flagCheck.Name, flagCheck.Detail)
	}
	return nil
}

// checkPhaseBuildable verifies a phase exists and is in a buildable state.
func checkPhaseBuildable(phaseNum int) gateCheck {
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		return gateCheck{
			Name:   "phase_buildable",
			Passed: false,
			Detail: "COLONY_STATE.json not found",
		}
	}

	var phase *colony.Phase
	for i := range state.Plan.Phases {
		if state.Plan.Phases[i].ID == phaseNum {
			phase = &state.Plan.Phases[i]
			break
		}
	}
	if phase == nil {
		return gateCheck{
			Name:   "phase_buildable",
			Passed: false,
			Detail: fmt.Sprintf("phase %d not found in plan", phaseNum),
		}
	}

	status := strings.ToLower(string(phase.Status))
	if status == "completed" || status == "in_progress" {
		return gateCheck{
			Name:   "phase_buildable",
			Passed: false,
			Detail: fmt.Sprintf("phase %d already %s", phaseNum, status),
		}
	}

	return gateCheck{
		Name:   "phase_buildable",
		Passed: true,
		Detail: fmt.Sprintf("phase %d ready to build", phaseNum),
	}
}

// checkPhaseBuilt verifies a phase has been built before continuing.
func checkPhaseBuilt(phaseNum int) gateCheck {
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		return gateCheck{
			Name:   "phase_built",
			Passed: false,
			Detail: "COLONY_STATE.json not found",
		}
	}

	var phase *colony.Phase
	for i := range state.Plan.Phases {
		if state.Plan.Phases[i].ID == phaseNum {
			phase = &state.Plan.Phases[i]
			break
		}
	}
	if phase == nil {
		return gateCheck{
			Name:   "phase_built",
			Passed: false,
			Detail: fmt.Sprintf("phase %d not found in plan", phaseNum),
		}
	}

	status := strings.ToLower(string(phase.Status))
	if status == "completed" || status == "in_progress" {
		return gateCheck{
			Name:   "phase_built",
			Passed: true,
			Detail: fmt.Sprintf("phase %d status: %s", phaseNum, status),
		}
	}

	return gateCheck{
		Name:   "phase_built",
		Passed: false,
		Detail: fmt.Sprintf("phase %d not yet built (status: %s)", phaseNum, status),
	}
}

// gateRecoveryTemplates maps gate names to recovery instructions.
// Each template has 3 numbered steps. Use {phase} as a placeholder for the current phase number.
var gateRecoveryTemplates = map[string]string{
	"verification_loop": "Verification commands failed.\n" +
		"1. Check the failed step output above for specific errors\n" +
		"2. Fix the build, type, lint, or test failures\n" +
		"3. Re-run `/ant-continue` to re-verify",
	"spawn_gate": "Spawn gate failed: Prime Worker completed without specialists.\n" +
		"1. Run `/ant-build {phase}` again\n" +
		"2. Prime Worker must spawn at least 1 specialist (Builder or Watcher)\n" +
		"3. Re-run `/ant-continue` after spawns complete",
	"anti_pattern": "Anti-pattern gate failed: Critical patterns detected.\n" +
		"1. Review the critical anti-patterns listed above\n" +
		"2. Fix each critical finding (exposed secrets, SQL injection, crash patterns)\n" +
		"3. Re-run `/ant-continue` to re-scan",
	"complexity": "Complexity gate failed: Code exceeds maintainability thresholds.\n" +
		"1. Review files exceeding 300 lines or 50-line functions\n" +
		"2. Refactor to reduce complexity\n" +
		"3. Re-run `/ant-continue` to re-check",
	"gatekeeper": "Gatekeeper gate failed: Critical CVEs detected.\n" +
		"1. Run `npm audit` (or equivalent) to see full details\n" +
		"2. Fix or update vulnerable dependencies\n" +
		"3. Re-run `/ant-continue` after resolving",
	"auditor": "Auditor gate failed: Critical quality issues or score below 60.\n" +
		"1. Review the critical findings listed above\n" +
		"2. Fix each critical finding first, then address high-severity items\n" +
		"3. Re-run `/ant-continue` to re-audit",
	"tdd_evidence": "TDD gate failed: Claimed tests not found in codebase.\n" +
		"1. Run `/ant-build {phase}` again\n" +
		"2. Actually write test files (not just claim them)\n" +
		"3. Tests must exist and be runnable",
	"runtime": "Runtime gate failed: User reported application issues.\n" +
		"1. Fix the reported runtime issues\n" +
		"2. Test the application manually\n" +
		"3. Re-run `/ant-continue` and confirm the app works",
	"flags": "Flags gate failed: Unresolved blocker flags.\n" +
		"1. Review each blocker flag listed above\n" +
		"2. Fix the issues and resolve flags: `/ant-flags --resolve {id} \"resolution\"`\n" +
		"3. Re-run `/ant-continue` after resolving all blockers",
	"watcher_veto": "Watcher VETO: Quality score below 7 or critical issues found.\n" +
		"1. Review the critical issues and quality score\n" +
		"2. Fix issues, then run `/ant-build {phase}` again\n" +
		"3. Watcher must re-verify with score >= 7 and no CRITICAL issues",
	"medic": "Medic gate failed: Critical colony health issues.\n" +
		"1. Review the critical health issues listed above\n" +
		"2. Run `aether medic --fix` to attempt repairs\n" +
		"3. Re-run `/ant-continue` after repairs",
	"tests_pass": "Tests failed.\n" +
		"1. Run `go test ./...` (or project test command) to see failures\n" +
		"2. Fix the failing tests\n" +
		"3. Re-run `/ant-continue` to re-verify",
}

// gateRecoveryTemplate returns the recovery instructions for a gate name.
// Returns a fallback message if the gate name is not found.
func gateRecoveryTemplate(name string) string {
	if tmpl, ok := gateRecoveryTemplates[name]; ok {
		return tmpl
	}
	return "No specific recovery instructions available for this gate."
}

// shouldSkipGate determines whether a gate should be skipped based on prior results.
// Per D-10: tests_pass is never skipped regardless of prior results.
// Per D-11: other passed gates are skipped on re-run.
func shouldSkipGate(priorResults []colony.GateResultEntry, gateName string) bool {
	if gateName == "tests_pass" {
		return false
	}
	for _, r := range priorResults {
		if r.Name == gateName && r.Passed {
			return true
		}
	}
	return false
}

// gateResultsWrite persists gate results to COLONY_STATE.json using atomic write.
func gateResultsWrite(entries []colony.GateResultEntry) error {
	var updated colony.ColonyState
	return store.UpdateJSONAtomically("COLONY_STATE.json", &updated, func() error {
		updated.GateResults = entries
		return nil
	})
}

// gateResultsRead returns gate results from COLONY_STATE.json.
// Returns nil if the file does not exist or cannot be read.
func gateResultsRead() []colony.GateResultEntry {
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		return nil
	}
	return state.GateResults
}

// formatSkipSummary produces a human-readable summary of prior gate results.
// Returns a string like "Skipping 8 passed gates -- re-checking 3 failures".
// Returns empty string if no prior results exist.
func formatSkipSummary(priorResults []colony.GateResultEntry) string {
	if len(priorResults) == 0 {
		return ""
	}
	passed := 0
	failed := 0
	for _, r := range priorResults {
		if r.Passed {
			passed++
		} else {
			failed++
		}
	}
	return fmt.Sprintf("Skipping %d passed gates -- re-checking %d failures", passed, failed)
}

// --- Cobra CLI subcommands for gate results ---

var gateResultsReadCmd = &cobra.Command{
	Use:          "gate-results-read",
	Short:        "Read gate results from COLONY_STATE.json",
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		results := gateResultsRead()
		if results == nil {
			results = []colony.GateResultEntry{}
		}
		data, _ := json.Marshal(results)
		fmt.Fprintln(stdout, string(data))
		return nil
	},
}

var gateResultsWriteCmd = &cobra.Command{
	Use:          "gate-results-write",
	Short:        "Write a gate result entry to COLONY_STATE.json",
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := mustGetString(cmd, "name")
		if name == "" {
			outputErrorMessage("--name is required")
			return nil
		}
		passed, _ := cmd.Flags().GetBool("passed")
		detail, _ := cmd.Flags().GetString("detail")

		entry := colony.GateResultEntry{
			Name:      name,
			Passed:    passed,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Detail:    detail,
		}
		if err := gateResultsWrite([]colony.GateResultEntry{entry}); err != nil {
			outputError(1, "failed to write gate result", err)
			return nil
		}
		data, _ := json.Marshal(map[string]interface{}{"ok": true, "entry": entry})
		fmt.Fprintln(stdout, string(data))
		return nil
	},
}

var shouldSkipGateCmd = &cobra.Command{
	Use:          "should-skip-gate",
	Short:        "Check whether a gate should be skipped based on prior results",
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := mustGetString(cmd, "name")
		if name == "" {
			outputErrorMessage("--name is required")
			return nil
		}
		prior := gateResultsRead()
		result := shouldSkipGate(prior, name)
		fmt.Fprintln(stdout, strconv.FormatBool(result))
		return nil
	},
}

var gateRecoveryTemplateCmd = &cobra.Command{
	Use:          "gate-recovery-template",
	Short:        "Get the recovery template for a gate type",
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := mustGetString(cmd, "name")
		if name == "" {
			outputErrorMessage("--name is required")
			return nil
		}
		template := gateRecoveryTemplate(name)
		fmt.Fprintln(stdout, template)
		return nil
	},
}

func init() {
	gateCheckCmd.Flags().String("action", "", "Action to check: task-complete or phase-advance (required)")
	gateCheckCmd.Flags().String("task", "", "Task ID for task-complete action (e.g., 1.1)")
	gateCheckCmd.Flags().Int("phase", 0, "Phase number for phase-advance action")
	rootCmd.AddCommand(gateCheckCmd)

	// Gate results CLI subcommands
	gateResultsWriteCmd.Flags().String("name", "", "Gate name (required)")
	gateResultsWriteCmd.Flags().Bool("passed", false, "Whether gate passed")
	gateResultsWriteCmd.Flags().String("detail", "", "Optional detail about the result")
	rootCmd.AddCommand(gateResultsReadCmd)
	rootCmd.AddCommand(gateResultsWriteCmd)

	shouldSkipGateCmd.Flags().String("name", "", "Gate name to check (required)")
	rootCmd.AddCommand(shouldSkipGateCmd)

	gateRecoveryTemplateCmd.Flags().String("name", "", "Gate name (required)")
	rootCmd.AddCommand(gateRecoveryTemplateCmd)
}
