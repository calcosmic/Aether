package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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
	Name            string   `json:"name"`
	Passed          bool     `json:"passed"`
	Detail          string   `json:"detail,omitempty"`
	FixHint         string   `json:"fix_hint,omitempty"`
	RecoveryOptions []string `json:"recovery_options,omitempty"`
}

type gateResult struct {
	Allowed bool        `json:"allowed"`
	Reason  string      `json:"reason,omitempty"`
	Checks  []gateCheck `json:"checks"`
}

// QueenAnnotation records a queen decision about a gate finding.
// This is appended to the gate result without modifying the original finding.
// Per D-07: original Detail, FixHint, and RecoveryOptions are never touched.
type QueenAnnotation struct {
	Decision     string `json:"decision"`      // "auto-resolved", "escalated", "skipped"
	Rationale    string `json:"rationale"`     // Why the queen made this decision
	Timestamp    string `json:"timestamp"`     // RFC3339
	QueenVersion string `json:"queen_version"` // Runtime version for traceability
}

// GateCheckResult records a gate result for per-phase persistence.
// This is the richer format stored in gate-results-{N}.json files.
type GateCheckResult struct {
	Name            string           `json:"name"`
	Status          string           `json:"status"` // "passed", "failed", "skipped", "not-reached"
	Detail          string           `json:"detail,omitempty"`
	FixHint         string           `json:"fix_hint,omitempty"`
	RecoveryOptions []string         `json:"recovery_options,omitempty"`
	Timestamp       string           `json:"timestamp"`
	RetryCount      int              `json:"retry_count"`
	QueenAnnotation *QueenAnnotation `json:"queen_annotation,omitempty"` // Optional queen decision trail
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

	ctx, cancel := context.WithTimeout(context.Background(), BuildTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Dir = storage.ResolveAetherRoot(context.Background())
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return gateCheck{
			Name:    "tests_pass",
			Passed:  false,
			Detail:  fmt.Sprintf("test command timed out after %s", BuildTimeout),
			FixHint: "rerun the test command with a longer targeted timeout or investigate the hanging test",
		}
	}

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
	// Resolve against the colony's own root, not the process cwd. The previous
	// ResolveAetherRoot call answered "where am I running" rather than "which
	// repository is this gate checking" — under `go test` that resolved to the
	// Aether repo itself, so the gate extracted `go test ./...` from CLAUDE.md
	// and recursively ran the entire suite inside a 2-minute timeout.
	repoRoot := ""
	if store != nil && strings.TrimSpace(store.BasePath()) != "" {
		// BasePath is <root>/.aether/data; walk up to the repo root.
		repoRoot = filepath.Dir(filepath.Dir(store.BasePath()))
	}
	if repoRoot == "" {
		repoRoot = storage.ResolveAetherRoot(context.Background())
	}
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

// checkNoCriticalFlags checks for CRITICAL severity error records in the
// colony state. Deliberately NOT the Iron Law check: this gate also runs
// pre-build, and the classic law scopes to ADVANCEMENT — you may build with
// an open blocker (autopilot pauses on it, continue refuses to advance past
// it), so the blocker-flag check lives in checkUnresolvedBlockerFlags and is
// wired into the continue gates only.
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

// checkUnresolvedBlockerFlags enforces the classic Iron Law: "No phase
// advancement with unresolved blockers." It reads blocker-type flags from
// pending-decisions.json (flags.json fallback). This regressed silently in
// the Go port — the continue gates claimed to run a flag check "every time
// for safety" while never opening the flags file, so a blocker raised with
// /ant-flag did not actually block /ant-continue. Blockers cannot be
// acknowledged away — only resolved. Locked by TestBlockerFlagBlocksContinue.
func checkUnresolvedBlockerFlags() gateCheck {
	ff, _, err := readCanonicalBlockerFlags(store)
	if err != nil {
		return gateCheck{
			Name:   "no_unresolved_blockers",
			Passed: false,
			Detail: "blocker truth unavailable: " + blockerSnapshotErrorDetail(store, err),
		}
	}
	blockerDescriptions := []string{}
	for _, flag := range ff.Decisions {
		if flag.Resolved || !strings.EqualFold(strings.TrimSpace(flag.Type), "blocker") {
			continue
		}
		desc := strings.TrimSpace(flag.Description)
		if desc == "" {
			desc = flag.ID
		}
		blockerDescriptions = append(blockerDescriptions, desc)
	}
	if len(blockerDescriptions) > 0 {
		shown := blockerDescriptions
		if len(shown) > 3 {
			shown = shown[:3]
		}
		return gateCheck{
			Name:   "no_unresolved_blockers",
			Passed: false,
			Detail: fmt.Sprintf("%d unresolved blocker flag(s): %s", len(blockerDescriptions), strings.Join(shown, "; ")),
		}
	}
	return gateCheck{Name: "no_unresolved_blockers", Passed: true, Detail: "no blocker flags"}
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

// checkAntiPatternGate scans the files a phase claims to have touched for
// security antipatterns (hardcoded secrets, etc.) using the shared
// scanFileForAntipatterns implementation, and returns two gate checks:
//
//   - "anti_pattern" (soft_block): fails when a critical finding is present.
//     Before this function existed, "anti_pattern" was a fully specified gate
//     name in gateClassifications/gateRecoveryTemplates/
//     gateAutoResolveThresholds that no code anywhere produced -- this is the
//     producer ROADMAP success criterion 2 requires.
//   - "anti_pattern_executed" (hard_block): fails when the scan could not run
//     at all (no store, unresolvable root, or a per-file scan error). Per
//     D-01, a safety gate that cannot execute must never be reported as
//     passing.
//
// Paths in files are resolved against the colony root using the same idiom
// as effectiveGateAutoResolveThresholds: filepath.Dir(store.BasePath())
// joined with "..". Do NOT use storage.ResolveAetherRoot -- the comment
// above resolveTestCommand records it resolving to the wrong repository
// under `go test`.
func checkAntiPatternGate(files []string) (gateCheck, gateCheck) {
	findingsCheck := gateCheck{Name: "anti_pattern"}
	executedCheck := gateCheck{Name: "anti_pattern_executed"}

	if store == nil {
		executedCheck.Passed = false
		executedCheck.Detail = "antipattern scan could not execute: no store initialized, colony root unresolvable"
		executedCheck.FixHint = gateRecoveryTemplate("anti_pattern")
		executedCheck.RecoveryOptions = []string{
			"Fix manually and run aether continue",
			"Run aether unblock --dispatch for guided recovery",
		}
		findingsCheck.Passed = true
		findingsCheck.Detail = "antipattern scan did not run: no store initialized"
		return findingsCheck, executedCheck
	}

	root := filepath.Join(filepath.Dir(store.BasePath()), "..")

	var allCriticals []AntipatternFinding
	scannedFiles := 0
	claimedFiles := 0
	var scanErrors []string
	var absentFiles []string

	for _, f := range files {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		claimedFiles++
		resolved := f
		if !filepath.IsAbs(resolved) {
			resolved = filepath.Join(root, f)
		}
		// scanFileForAntipatterns deliberately returns clean (nil,nil,nil) for
		// a file that does not exist — correct for the CLI, where a phase may
		// legitimately have deleted the file. But the gate must NOT count an
		// absent file as scanned: doing so reports "scan executed successfully
		// across 1 file(s)" when nothing was read, which is precisely the
		// looks-like-it-ran-but-didn't failure this gate exists to prevent.
		if _, statErr := os.Stat(resolved); statErr != nil {
			absentFiles = append(absentFiles, f)
			continue
		}
		criticals, _, err := scanFileForAntipatterns(resolved)
		if err != nil {
			scanErrors = append(scanErrors, fmt.Sprintf("%s: %v", f, err))
			continue
		}
		scannedFiles++
		allCriticals = append(allCriticals, criticals...)
	}

	if len(scanErrors) > 0 {
		executedCheck.Passed = false
		executedCheck.Detail = fmt.Sprintf("antipattern scan could not execute for %d file(s): %s", len(scanErrors), strings.Join(scanErrors, "; "))
		executedCheck.FixHint = gateRecoveryTemplate("anti_pattern")
		executedCheck.RecoveryOptions = []string{
			"Fix manually and run aether continue",
			"Run aether unblock --dispatch for guided recovery",
		}
	} else if claimedFiles == 0 {
		executedCheck.Passed = true
		executedCheck.Detail = "no changed files were claimed for this phase; antipattern scan had nothing to scan"
	} else if scannedFiles == 0 {
		// Files were claimed but not one could be read. Whatever the cause —
		// wrong paths, a root-resolution mismatch, or claims describing work
		// that never landed — the security scan did not happen, and a phase
		// must not pass with its safety gate unexecuted (D-01).
		executedCheck.Passed = false
		executedCheck.Detail = fmt.Sprintf("antipattern scan executed against 0 of %d claimed file(s); none could be read (absent: %s)",
			claimedFiles, strings.Join(absentFiles, ", "))
		executedCheck.FixHint = gateRecoveryTemplate("anti_pattern")
		executedCheck.RecoveryOptions = []string{
			"Fix manually and run aether continue",
			"Run aether unblock --dispatch for guided recovery",
		}
	} else {
		executedCheck.Passed = true
		executedCheck.Detail = fmt.Sprintf("antipattern scan executed across %d of %d claimed file(s)", scannedFiles, claimedFiles)
		if len(absentFiles) > 0 {
			executedCheck.Detail += fmt.Sprintf("; %d absent (deleted or wrong path): %s", len(absentFiles), strings.Join(absentFiles, ", "))
		}
	}

	if len(allCriticals) > 0 {
		distinctFiles := map[string]bool{}
		var locations []string
		for _, c := range allCriticals {
			distinctFiles[c.File] = true
			if len(locations) < 5 {
				locations = append(locations, fmt.Sprintf("%s:%d", c.File, c.Line))
			}
		}
		findingsCheck.Passed = false
		findingsCheck.Detail = fmt.Sprintf("%d critical antipattern finding(s) across %d file(s): %s", len(allCriticals), len(distinctFiles), strings.Join(locations, ", "))
		findingsCheck.FixHint = gateRecoveryTemplate("anti_pattern")
		findingsCheck.RecoveryOptions = []string{
			"Fix manually and run aether continue",
			"Run aether unblock --dispatch for guided recovery",
		}
	} else {
		// A pass detail that does not state the number scanned is
		// unacceptable: it makes "scanned nothing" indistinguishable from
		// "scanned everything and found nothing" (T-160-10).
		findingsCheck.Passed = true
		findingsCheck.Detail = fmt.Sprintf("scanned %d of %d claimed file(s), no critical patterns", scannedFiles, claimedFiles)
	}

	return findingsCheck, executedCheck
}

// governanceInvocationTokens maps a governanceDetectors label (cmd/init_research.go)
// to the substring its tool's CLI invocation is expected to contain in a
// verification step's Command. Only labels in the "linter", "formatter", and
// "test" categories are mechanically checkable per D-09's Open Question 2 --
// "CI:" and "Build:" categories are deliberately excluded (CI runs outside
// the colony's observation; build tools are not conduct rules).
var governanceInvocationTokens = map[string]string{
	"ESLint":        "eslint",
	"Prettier":      "prettier",
	"Biome":         "biome",
	"golangci-lint": "golangci-lint",
	"pytest":        "pytest",
	"Jest":          "jest",
	"Vitest":        "vitest",
}

// governanceConfigFilesForLabel returns every governanceDetectors config file
// associated with a label, restricted to the linter/formatter/test
// categories checkCharterComplianceGate is scoped to enforce.
func governanceConfigFilesForLabel(label string) []string {
	var files []string
	for _, det := range governanceDetectors {
		if det.label != label {
			continue
		}
		switch det.category {
		case "linter", "formatter", "test":
			files = append(files, det.file)
		}
	}
	return files
}

// parseCharterGovernanceLabels parses generateCharter's Governance string
// (cmd/init_research.go:1640-1662 -- "Linting: <labels>. CI: <labels>. ..."
// joined with ". ", labels comma-separated) into category -> declared labels.
func parseCharterGovernanceLabels(governance string) map[string][]string {
	result := map[string][]string{}
	governance = strings.TrimSpace(governance)
	if governance == "" {
		return result
	}
	for _, part := range strings.Split(governance, ".") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		idx := strings.Index(part, ":")
		if idx < 0 {
			continue
		}
		category := strings.TrimSpace(part[:idx])
		labelsStr := strings.TrimSpace(part[idx+1:])
		if labelsStr == "" {
			continue
		}
		var labels []string
		for _, l := range strings.Split(labelsStr, ",") {
			if l = strings.TrimSpace(l); l != "" {
				labels = append(labels, l)
			}
		}
		if len(labels) > 0 {
			result[category] = labels
		}
	}
	return result
}

// checkCharterComplianceGate enforces the mechanically-checkable subset of
// the colony charter's Governance string (Linting/Testing/Formatting
// categories only -- see governanceInvocationTokens) and returns two gate
// checks, copying checkAntiPatternGate's two-check shape:
//
//   - "charter_compliance" (soft_block): fails when a governance tool is
//     declared in the charter AND its config file still exists in the repo
//     AND no verification step's Command references the tool's invocation
//     token. That triple condition is deliberately narrow: it is exactly
//     "the colony declared this tool, the tool is still installed, and
//     nothing ran it" (the ignored-rule case D-09 names), and it cannot fire
//     on a repo that simply dropped a tool (charter drift is not a worker's
//     fault to be blocked for).
//   - "charter_compliance_executed" (hard_block): fails when the scan could
//     not run at all (no store, or COLONY_STATE.json could not be loaded).
//
// "CI:" and "Build:" governance categories are intentionally never checked
// here -- CI runs outside the colony's observation and build tools are not
// conduct rules.
//
// Intent, Vision, Goals, TechStack and KeyRisks reach workers through the
// colony-prime charter section as *orientation*, not as hard rules, and are
// not gated here. This comment previously said they arrived "as prose hard
// rules"; they arrived as nothing at all -- colony-prime emitted only
// Governance and Constraints -- which is why the claim went unnoticed for so
// long. TestColonyPrimeIncludesCharterIntentAndGoals now pins the delivery.
func checkCharterComplianceGate(steps []codexVerificationStep) (gateCheck, gateCheck) {
	findingsCheck := gateCheck{Name: "charter_compliance"}
	executedCheck := gateCheck{Name: "charter_compliance_executed"}

	if store == nil {
		executedCheck.Passed = false
		executedCheck.Detail = "charter compliance scan could not execute: no store initialized, colony root unresolvable"
		executedCheck.FixHint = gateRecoveryTemplate("charter_compliance")
		executedCheck.RecoveryOptions = []string{
			"Fix manually and run aether continue",
			"Run aether unblock --dispatch for guided recovery",
		}
		findingsCheck.Passed = true
		findingsCheck.Detail = "charter compliance scan did not run: no store initialized"
		return findingsCheck, executedCheck
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		executedCheck.Passed = false
		executedCheck.Detail = fmt.Sprintf("charter compliance scan could not execute: %v", err)
		executedCheck.FixHint = gateRecoveryTemplate("charter_compliance")
		executedCheck.RecoveryOptions = []string{
			"Fix manually and run aether continue",
			"Run aether unblock --dispatch for guided recovery",
		}
		findingsCheck.Passed = true
		findingsCheck.Detail = "charter compliance scan did not run: could not load colony state"
		return findingsCheck, executedCheck
	}

	root := filepath.Join(filepath.Dir(store.BasePath()), "..")

	var governance string
	if state.Charter != nil {
		governance = strings.TrimSpace(state.Charter.Governance)
	}
	if governance == "" || governance == charterNoGovernanceFallback {
		executedCheck.Passed = true
		executedCheck.Detail = "charter compliance scan executed: no charter governance declared"
		findingsCheck.Passed = true
		findingsCheck.Detail = "no mechanically-checkable governance rules declared; nothing to enforce"
		return findingsCheck, executedCheck
	}

	labelsByCategory := parseCharterGovernanceLabels(governance)
	var declaredLabels []string
	for _, category := range []string{"Linting", "Testing", "Formatting"} {
		declaredLabels = append(declaredLabels, labelsByCategory[category]...)
	}

	checkedCount := 0
	var violations []string
	for _, label := range declaredLabels {
		configFiles := governanceConfigFilesForLabel(label)
		if len(configFiles) == 0 {
			continue // unknown label -- nothing mechanical to check
		}
		configExists := false
		for _, f := range configFiles {
			if _, statErr := os.Stat(filepath.Join(root, f)); statErr == nil {
				configExists = true
				break
			}
		}
		if !configExists {
			// Charter drift: the colony declared a tool the repo no longer
			// has configured. Not a violation -- do not manufacture a false
			// block over something outside the worker's control.
			continue
		}
		token, ok := governanceInvocationTokens[label]
		if !ok {
			continue
		}
		checkedCount++
		exercised := false
		for _, step := range steps {
			if strings.Contains(strings.ToLower(step.Command), strings.ToLower(token)) {
				exercised = true
				break
			}
		}
		if !exercised {
			violations = append(violations, label)
		}
	}

	executedCheck.Passed = true
	executedCheck.Detail = fmt.Sprintf("charter compliance scan executed: checked %d of %d declared governance label(s)", checkedCount, len(declaredLabels))

	if len(violations) > 0 {
		sort.Strings(violations)
		findingsCheck.Passed = false
		findingsCheck.Detail = fmt.Sprintf("declared governance tool(s) not exercised by any verification step: %s", strings.Join(violations, ", "))
		findingsCheck.FixHint = gateRecoveryTemplate("charter_compliance")
		findingsCheck.RecoveryOptions = []string{
			"Fix manually and run aether continue",
			"Run aether unblock --dispatch for guided recovery",
		}
	} else {
		findingsCheck.Passed = true
		findingsCheck.Detail = fmt.Sprintf("checked %d declared governance label(s), no compliance violations", checkedCount)
	}

	return findingsCheck, executedCheck
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
		"3. Re-run `aether continue` to re-verify",
	"spawn_gate": "Spawn gate failed: Prime Worker completed without specialists.\n" +
		"1. Run `aether build {phase}` again\n" +
		"2. Prime Worker must spawn at least 1 specialist (Builder or Watcher)\n" +
		"3. Re-run `aether continue` after spawns complete",
	"anti_pattern": "Anti-pattern gate failed: Critical patterns detected.\n" +
		"1. Review the critical anti-patterns listed above\n" +
		"2. Fix each critical finding (exposed secrets, SQL injection, crash patterns)\n" +
		"3. Re-run `aether continue` to re-scan",
	"charter_compliance": "Charter compliance gate failed: a declared governance tool was not exercised.\n" +
		"1. Review the unexercised tool(s) named above\n" +
		"2. Run the tool, or add it to a verification step's command\n" +
		"3. Re-run `aether continue` to re-scan",
	"complexity": "Complexity gate failed: Code exceeds maintainability thresholds.\n" +
		"1. Review files exceeding 300 lines or 50-line functions\n" +
		"2. Refactor to reduce complexity\n" +
		"3. Re-run `aether continue` to re-check",
	"gatekeeper": "Gatekeeper gate failed: Critical CVEs detected.\n" +
		"1. Run `npm audit` (or equivalent) to see full details\n" +
		"2. Fix or update vulnerable dependencies\n" +
		"3. Re-run `aether continue` after resolving",
	"auditor": "Auditor gate failed: Critical quality issues or score below 60.\n" +
		"1. Review the critical findings listed above\n" +
		"2. Fix each critical finding first, then address high-severity items\n" +
		"3. Re-run `aether continue` to re-audit",
	"tdd_evidence": "TDD gate failed: Claimed tests not found in codebase.\n" +
		"1. Run `aether build {phase}` again\n" +
		"2. Actually write test files (not just claim them)\n" +
		"3. Tests must exist and be runnable",
	"runtime": "Runtime gate failed: User reported application issues.\n" +
		"1. Fix the reported runtime issues\n" +
		"2. Test the application manually\n" +
		"3. Re-run `aether continue` and confirm the app works",
	"flags": "Flags gate failed: Unresolved blocker flags.\n" +
		"1. Review each blocker flag listed above\n" +
		"2. Fix the issues and resolve flags: `aether flag-resolve --id {id} --message \"resolution\"`\n" +
		"3. Re-run `aether continue` after resolving all blockers",
	"watcher_veto": "Watcher VETO: Quality score below 7 or critical issues found.\n" +
		"1. Review the critical issues and quality score\n" +
		"2. Fix issues, then run `aether build {phase}` again\n" +
		"3. Watcher must re-verify with score >= 7 and no CRITICAL issues",
	"medic": "Medic gate failed: Critical colony health issues.\n" +
		"1. Review the critical health issues listed above\n" +
		"2. Run `aether medic --fix` to attempt repairs\n" +
		"3. Re-run `aether continue` after repairs",
	"tests_pass": "Tests failed.\n" +
		"1. Run `go test ./...` (or project test command) to see failures\n" +
		"2. Fix the failing tests\n" +
		"3. Re-run `aether continue` to re-verify",
}

// gateRecoveryTemplate returns the recovery instructions for a gate name.
// Returns a fallback message if the gate name is not found.
func gateRecoveryTemplate(name string) string {
	if tmpl, ok := gateRecoveryTemplates[name]; ok {
		return tmpl
	}
	return "No specific recovery instructions available for this gate."
}

// alwaysRunGates lists gates that always execute regardless of prior results.
var alwaysRunGates = map[string]bool{
	"tests_pass":                  true,
	"flags":                       true,
	"watcher_veto":                true,
	"no_critical_flags":           true,
	"anti_pattern_executed":       true,
	"charter_compliance_executed": true,
}

// GateClassificationTier represents the severity tier of a gate.
// Classification is deterministic and code-level -- never user-configurable.
type GateClassificationTier string

const (
	hardBlock GateClassificationTier = "hard_block"
	softBlock GateClassificationTier = "soft_block"
	advisory  GateClassificationTier = "advisory"
)

// gateClassificationEntry records a gate's tier and why it has that tier.
type gateClassificationEntry struct {
	Tier      GateClassificationTier
	Rationale string
}

// gateClassifications maps every named gate to its classification tier.
// This is a read-only constant -- no configuration can change these values.
// Gatekeeper and watcher_veto are compile-time hard_block per D-06.
var gateClassifications = map[string]gateClassificationEntry{
	// hard_block gates (6): security, quality veto, human escalation, and pre-checks
	"gatekeeper":                  {hardBlock, "Security CVE findings require human judgment"},
	"watcher_veto":                {hardBlock, "Watcher has final say by colony design"},
	"flags":                       {hardBlock, "Flags represent intentional human escalation"},
	"tests_pass":                  {hardBlock, "Broken build is always a hard block"},
	"no_critical_flags":           {hardBlock, "Critical errors existing is always a hard block"},
	"anti_pattern_executed":       {hardBlock, "A safety scan which could not run must never be reported as passing"},
	"charter_compliance_executed": {hardBlock, "A compliance scan which could not run must never be reported as passing"},
	// soft_block gates (7): quality findings that auto-resolve when non-critical
	"auditor":            {softBlock, "Quality findings auto-resolve when non-critical"},
	"complexity":         {softBlock, "Maintainability thresholds are advisory until verified"},
	"tdd_evidence":       {softBlock, "Missing test claims can be fulfilled by re-build"},
	"anti_pattern":       {softBlock, "Critical patterns are actionable but non-blocking when addressed"},
	"charter_compliance": {softBlock, "Declared governance tools going unexercised are actionable but non-blocking when addressed"},
	"verification_loop":  {softBlock, "Build failures are transient and retriable"},
	"spawn_gate":         {softBlock, "Missing spawns are recoverable by re-dispatch"},
	// advisory gates (2): diagnostic/logging only
	"medic":   {advisory, "Health diagnostics are informational only"},
	"runtime": {advisory, "User-reported issues are logged but never gate advancement"},
}

// gateClassify returns the classification tier and rationale for a gate name.
// Returns ("", "") for unknown gates -- caller decides how to handle unclassified gates.
// Unknown gates (like continue-flow structural gates) are intentionally unclassified.
func gateClassify(gateName string) (GateClassificationTier, string) {
	if entry, ok := gateClassifications[gateName]; ok {
		return entry.Tier, entry.Rationale
	}
	return "", ""
}

// isHardBlockGate returns true if the gate is classified as hard_block.
// Returns false for unknown gates (fail-open for unclassified structural gates).
func isHardBlockGate(gateName string) bool {
	tier, _ := gateClassify(gateName)
	return tier == hardBlock
}

// phaseModeAwareGateClassify returns the classification tier for a gate name,
// potentially relaxed based on the phase mode. Discovery phases get the lightest
// treatment; prototype and maintenance get relaxed treatment; production keeps
// the default.
func phaseModeAwareGateClassify(gateName string, mode colony.PhaseMode) (GateClassificationTier, string) {
	baseTier, rationale := gateClassify(gateName)

	// Production keeps base classification.
	if mode == colony.PhaseModeProduction {
		return baseTier, rationale
	}

	switch gateName {
	case "gatekeeper":
		if mode == colony.PhaseModeDiscovery {
			return advisory, "Security gating is advisory for discovery phases"
		}
		return softBlock, "Security gating is soft for prototype/maintenance phases"
	case "auditor":
		if mode == colony.PhaseModeDiscovery {
			return advisory, "Quality audit is advisory for discovery phases"
		}
		return softBlock, "Quality audit is soft for prototype/maintenance phases"
	case "probe":
		if mode == colony.PhaseModeDiscovery {
			return "", "Probe skipped for discovery phases"
		}
		return softBlock, "Coverage probe is soft for prototype/maintenance phases"
	case "complexity":
		if mode == colony.PhaseModeDiscovery {
			return "", "Complexity check skipped for discovery phases"
		}
		return softBlock, "Complexity is soft for prototype/maintenance phases"
	case "tdd_evidence":
		if mode == colony.PhaseModeDiscovery {
			return "", "TDD evidence skipped for discovery phases"
		}
		return softBlock, "TDD evidence is soft for prototype/maintenance phases"
	}

	return baseTier, rationale
}

// isGateSkippedForMode returns true if the gate should not run at all for the
// given phase mode (e.g. probe for discovery phases).
func isGateSkippedForMode(gateName string, mode colony.PhaseMode) bool {
	tier, _ := phaseModeAwareGateClassify(gateName, mode)
	return tier == ""
}

// --- Auto-Resolve Engine (Phase 95) ---

// gateAutoResolveThreshold defines the auto-resolve threshold for a soft_block gate.
// Per D-05: thresholds are hardcoded constants, not configurable per-colony.
type gateAutoResolveThreshold struct {
	Threshold float64 // Numeric threshold; gate findings at or below this value auto-resolve
	Rationale string  // Why this threshold was chosen
}

// gateAutoResolveThresholds maps each soft_block gate to its auto-resolve threshold.
// Only gates classified as soft_block have entries here.
// For binary gates (pass/fail with no numeric score), threshold 0.0 means
// "always auto-resolve on failure when depth multiplier > 0".
// The depth multiplier adjusts these values: light depth multiplies by 1.5,
// standard by 1.0, heavy by 0.0 (no auto-resolve).
var gateAutoResolveThresholds = map[string]gateAutoResolveThreshold{
	"auditor":            {0.0, "Any auditor finding in continue flow is auto-resolvable -- structural quality gates handled at review, not gate level"},
	"complexity":         {0.0, "Complexity findings are advisory -- auto-resolvable in continue flow"},
	"tdd_evidence":       {0.0, "Missing test claims can be fulfilled by re-build -- auto-resolvable"},
	"anti_pattern":       {0.0, "Anti-pattern findings are actionable but non-blocking when addressed"},
	"charter_compliance": {0.0, "Charter compliance findings are actionable but non-blocking when addressed"},
	"verification_loop":  {0.0, "Verification failures are transient and retriable"},
	"spawn_gate":         {0.0, "Missing spawns are recoverable by re-dispatch"},
}

// effectiveGateAutoResolveThresholds returns the threshold map with any
// per-colony overrides from .planning/config.json applied (GATE-04).
// Config key: workflow.gate_auto_resolve_thresholds -- a map of gate name to float64 threshold.
// Missing or invalid overrides are silently ignored (defaults win).
func effectiveGateAutoResolveThresholds() map[string]gateAutoResolveThreshold {
	result := make(map[string]gateAutoResolveThreshold, len(gateAutoResolveThresholds))
	for k, v := range gateAutoResolveThresholds {
		result[k] = v
	}

	if store == nil {
		return result
	}
	configPath := filepath.Join(filepath.Dir(store.BasePath()), "..", ".planning", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return result
	}
	var cfg map[string]interface{}
	if json.Unmarshal(data, &cfg) != nil {
		return result
	}
	workflow, ok := cfg["workflow"].(map[string]interface{})
	if !ok {
		return result
	}
	overrides, ok := workflow["gate_auto_resolve_thresholds"].(map[string]interface{})
	if !ok {
		return result
	}
	for gateName, val := range overrides {
		if threshold, ok := val.(float64); ok {
			if existing, ok := result[gateName]; ok {
				result[gateName] = gateAutoResolveThreshold{
					Threshold: threshold,
					Rationale: existing.Rationale,
				}
			}
		}
	}
	return result
}

// autoResolveDepthMultiplier returns a multiplier applied to auto-resolve thresholds
// based on the current verification depth. Per D-06: light depth = more aggressive,
// heavy depth = more conservative.
// Light: multiplier 1.5 (thresholds effectively increase by 50%, more auto-resolves)
// Standard: multiplier 1.0 (thresholds as-is)
// Heavy: multiplier 0.0 (no auto-resolve at heavy depth -- user asked for thorough checking)
func autoResolveDepthMultiplier(depth colony.VerificationDepth) float64 {
	switch depth {
	case colony.VerificationDepthLight:
		return 1.5
	case colony.VerificationDepthHeavy:
		return 0.0
	default:
		return 1.0
	}
}

// annotateGateResult adds a QueenAnnotation to a specific gate in the per-phase
// gate-results-{N}.json file. Per D-10: the original Detail, FixHint, and
// RecoveryOptions are never modified -- only the QueenAnnotation pointer field
// is added/updated.
func annotateGateResult(phaseNum int, gateName string, annotation QueenAnnotation) error {
	results, err := gateResultsReadPhase(phaseNum)
	if err != nil {
		return fmt.Errorf("failed to read gate results for annotation: %w", err)
	}
	for i := range results {
		if results[i].Name == gateName {
			results[i].QueenAnnotation = &annotation
			break
		}
	}
	return gateResultsWritePhase(phaseNum, results)
}

// autoResolveSoftBlockGates evaluates failed soft_block gates against their thresholds.
// Per D-01: threshold-based, no LLM judgment. Per D-04: only soft_block gates.
// Returns the updated gate report and a list of auto-resolved gate names.
// The caller is responsible for persisting the updated report and dispatching
// the Fixer for remaining failures.
func autoResolveSoftBlockGates(phaseNum int, gates codexContinueGateReport, reviewDepth string, mode colony.PhaseMode) (codexContinueGateReport, []string) {
	depth := colony.NormalizeVerificationDepth(reviewDepth)
	multiplier := autoResolveDepthMultiplier(depth)
	thresholds := effectiveGateAutoResolveThresholds()

	var autoResolved []string
	var remainingBlockers []string

	for i, check := range gates.Checks {
		if check.Passed {
			continue
		}

		tier, _ := phaseModeAwareGateClassify(check.Name, mode)

		// Per D-04: only auto-resolve soft_block gates
		if tier != softBlock {
			remainingBlockers = append(remainingBlockers, check.Detail)
			continue
		}

		threshold, ok := thresholds[check.Name]
		if !ok {
			// Unclassified soft_block gate (should not happen, but safe default)
			remainingBlockers = append(remainingBlockers, check.Detail)
			continue
		}

		// For discovery phases, all soft_block gates auto-resolve regardless of depth.
		if mode == colony.PhaseModeDiscovery {
			gates.Checks[i].Passed = true
			autoResolved = append(autoResolved, check.Name)
			continue
		}

		// For binary gates (threshold 0.0): auto-resolve when multiplier > 0
		// For numeric gates: auto-resolve when threshold * multiplier covers the finding
		if shouldAutoResolve(check, threshold.Threshold, multiplier) {
			gates.Checks[i].Passed = true
			autoResolved = append(autoResolved, check.Name)
		} else {
			remainingBlockers = append(remainingBlockers, check.Detail)
		}
	}

	gates.Passed = len(remainingBlockers) == 0
	gates.BlockingIssues = remainingBlockers
	return gates, autoResolved
}

// shouldAutoResolve determines whether a specific gate check should be auto-resolved
// based on the threshold and depth multiplier. For binary gates (no numeric score),
// auto-resolve when the multiplier is positive (i.e., depth is not heavy).
// For numeric gates (future), auto-resolve when threshold * multiplier > 0.
func shouldAutoResolve(check gateCheck, threshold float64, multiplier float64) bool {
	// Heavy depth (multiplier 0.0) means no auto-resolve at all
	if multiplier <= 0 {
		return false
	}
	// Binary gates with any positive multiplier: auto-resolve
	// Numeric gates: effective threshold = threshold * multiplier
	return true
}

// shouldSkipGate determines whether a gate should be skipped based on prior results.
// Gates in alwaysRunGates never skip. Other gates with Status "passed" or "skipped" are skipped.
func shouldSkipGate(priorResults []GateCheckResult, gateName string) bool {
	if alwaysRunGates[gateName] {
		return false
	}
	for _, r := range priorResults {
		if r.Name == gateName && (r.Status == "passed" || r.Status == "skipped") {
			return true
		}
	}
	return false
}

// gateResultsWrite persists gate results to COLONY_STATE.json using atomic write.
// Entries are merged by Name key: existing entries with the same name are updated
// (upserted), and new entries are appended. This ensures sequential calls accumulate
// results instead of replacing all previous entries.
func gateResultsWrite(entries []colony.GateResultEntry) error {
	var updated colony.ColonyState
	return store.UpdateJSONAtomically("COLONY_STATE.json", &updated, func() error {
		indexByName := make(map[string]int, len(updated.GateResults))
		result := append([]colony.GateResultEntry{}, updated.GateResults...)
		for idx, e := range result {
			indexByName[e.Name] = idx
		}
		for _, e := range entries {
			if idx, ok := indexByName[e.Name]; ok {
				result[idx] = e
				continue
			}
			indexByName[e.Name] = len(result)
			result = append(result, e)
		}
		updated.GateResults = result
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

// gateResultsWritePhase persists gate results to a per-phase file gate-results-{N}.json.
func gateResultsWritePhase(phaseNum int, entries []GateCheckResult) error {
	rel := fmt.Sprintf("gate-results-%d.json", phaseNum)
	return store.SaveJSON(rel, entries)
}

// gateResultsReadPhase reads gate results from a per-phase file gate-results-{N}.json.
// Supports both the legacy plain array format and the newer gateResultsFile wrapper format.
// Returns an error if the file does not exist or cannot be read.
func gateResultsReadPhase(phaseNum int) ([]GateCheckResult, error) {
	rel := fmt.Sprintf("gate-results-%d.json", phaseNum)

	// Read raw content to detect format
	raw, err := store.LoadRawJSON(rel)
	if err != nil {
		return nil, err
	}

	// Try wrapper format first (newer format with unblock_attempts)
	// The wrapper is a JSON object, while the legacy format is a JSON array.
	if len(raw) > 0 && raw[0] == '{' {
		var wrapped gateResultsFile
		if err := json.Unmarshal(raw, &wrapped); err != nil {
			return nil, err
		}
		return wrapped.Results, nil
	}

	// Fall back to plain array format (legacy)
	var results []GateCheckResult
	if err := json.Unmarshal(raw, &results); err != nil {
		return nil, err
	}
	return results, nil
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
		phaseNum, _ := cmd.Flags().GetInt("phase")
		var prior []GateCheckResult
		if phaseNum > 0 {
			prior, _ = gateResultsReadPhase(phaseNum)
		}
		if prior == nil {
			prior = []GateCheckResult{}
		}
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

var gateClassifyCmd = &cobra.Command{
	Use:          "gate-classify",
	Short:        "Show gate classification tiers and rationale",
	Long:         "Display all gate classifications (hard_block, soft_block, advisory) with rationale.\nUse --json for structured output.",
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOutput, _ := cmd.Flags().GetBool("json")
		if jsonOutput {
			outputOK(gateClassifications)
			return nil
		}
		renderGateClassifyTable()
		return nil
	},
}

func renderGateClassifyTable() {
	type entry struct {
		name string
		gateClassificationEntry
	}
	var entries []entry
	for name, e := range gateClassifications {
		entries = append(entries, entry{name, e})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Tier != entries[j].Tier {
			return entries[i].Tier < entries[j].Tier
		}
		return entries[i].name < entries[j].name
	})

	// Classic headed style: one line per gate, rationale nested beneath.
	var b strings.Builder
	for _, e := range entries {
		b.WriteString(fmt.Sprintf("🚧 %s (%s)\n", e.name, string(e.Tier)))
		if rationale := strings.TrimSpace(e.Rationale); rationale != "" {
			b.WriteString("   └── " + rationale + "\n")
		}
	}
	visualFprintln(stdout, strings.TrimRight(b.String(), "\n"))
}

var gateAutoResolveCmd = &cobra.Command{
	Use:          "gate-auto-resolve",
	Short:        "Show gate auto-resolve thresholds and rationale",
	Long:         "Display auto-resolve thresholds for soft_block gates.\nHard_block and advisory gates are never auto-resolved.\nUse --json for structured output.\nThresholds can be overridden via .planning/config.json workflow.gate_auto_resolve_thresholds.",
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOutput, _ := cmd.Flags().GetBool("json")
		thresholds := effectiveGateAutoResolveThresholds()
		if jsonOutput {
			outputOK(thresholds)
			return nil
		}
		renderGateAutoResolveTableWith(thresholds)
		return nil
	},
}

func renderGateAutoResolveTable() {
	renderGateAutoResolveTableWith(effectiveGateAutoResolveThresholds())
}

func renderGateAutoResolveTableWith(thresholds map[string]gateAutoResolveThreshold) {
	type entry struct {
		name string
		gateAutoResolveThreshold
	}
	var entries []entry
	for name, e := range thresholds {
		entries = append(entries, entry{name, e})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].name < entries[j].name
	})

	// Classic headed style: per-gate line with the depth-adjusted thresholds
	// in prose, rationale nested beneath.
	var b strings.Builder
	for _, e := range entries {
		light := e.Threshold * autoResolveDepthMultiplier(colony.VerificationDepthLight)
		standard := e.Threshold * autoResolveDepthMultiplier(colony.VerificationDepthStandard)
		heavy := e.Threshold * autoResolveDepthMultiplier(colony.VerificationDepthHeavy)
		b.WriteString(fmt.Sprintf("🚧 %s: base %.1f (light %.1f / standard %.1f / heavy %.1f)\n", e.name, e.Threshold, light, standard, heavy))
		if rationale := strings.TrimSpace(e.Rationale); rationale != "" {
			b.WriteString("   └── " + rationale + "\n")
		}
	}
	visualFprintln(stdout, strings.TrimRight(b.String(), "\n"))
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
	shouldSkipGateCmd.Flags().Int("phase", 0, "Phase number for per-phase gate results lookup")
	rootCmd.AddCommand(shouldSkipGateCmd)

	gateRecoveryTemplateCmd.Flags().String("name", "", "Gate name (required)")
	rootCmd.AddCommand(gateRecoveryTemplateCmd)

	gateClassifyCmd.Flags().Bool("json", false, "Output as JSON")
	rootCmd.AddCommand(gateClassifyCmd)

	gateAutoResolveCmd.Flags().Bool("json", false, "Output as JSON")
	rootCmd.AddCommand(gateAutoResolveCmd)
}
