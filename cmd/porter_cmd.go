package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/spf13/cobra"
)

// porterContext distinguishes between source and consumer repo contexts.
type porterContext string

const (
	porterContextSource   porterContext = "source"
	porterContextConsumer porterContext = "consumer"
)

type porterReadinessScope string

const (
	porterReadinessScopeQuick       porterReadinessScope = "quick"
	porterReadinessScopeFullRelease porterReadinessScope = "full-release"
	porterReadinessReportRel                             = "porter/readiness.json"
)

// detectPorterContext determines whether the current working directory is inside
// the Aether source repo or a consumer repo.
func detectPorterContext() porterContext {
	ctx := detectIntegrityContext()
	if ctx == "source" {
		return porterContextSource
	}
	return porterContextConsumer
}

var porterCmd = &cobra.Command{
	Use:   "porter",
	Short: "Deliver colony work to the outside world",
	Long:  "Porter handles delivery of colony work: validate pipeline readiness, publish to the hub, and push releases.",
}

var porterCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate pipeline readiness for delivery",
	Long: "Runs a quick pipeline readiness check by default, including version alignment, " +
		"hub companion files, git status, test status, and changelog completeness. " +
		"Use --full-release for the expanded release gate, including source-surface, " +
		"version agreement, race, vet, build, GoReleaser, TypeScript host, npm, and smoke checks. " +
		"Works with or without an active colony.",
	Args: cobra.NoArgs,
	RunE: runPorterCheck,
}

func init() {
	porterCheckCmd.Flags().Bool("json", false, "Output JSON instead of visual report")
	porterCheckCmd.Flags().String("channel", "", "Override channel (stable or dev)")
	porterCheckCmd.Flags().Bool("full-release", false, "Run expanded release readiness checks, including slower release-tool commands")
	porterCmd.AddCommand(porterCheckCmd)
	rootCmd.AddCommand(porterCmd)
}

func runPorterCheck(cmd *cobra.Command, args []string) error {
	channel := runtimeChannelFromFlag(cmd.Flags())
	scope := porterReadinessScopeFromCommand(cmd)

	// Resolve hub directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}
	hubDir := resolveHubPathForHome(homeDir, channel)

	// Check if hub exists (skip hub-dependent checks if not installed)
	hubVersionFile := hubDir + "/version.json"
	ctx := detectPorterContext()
	if _, statErr := os.Stat(hubVersionFile); os.IsNotExist(statErr) {
		// Still run non-hub checks
		checks := []integrityCheck{
			checkSourceVersion(),
			checkBinaryVersion(),
			checkGitStatus(),
			checkGitStashes(),
			checkGitWorktrees(),
			checkTestStatus(),
			checkChangelogCompleteness(),
		}
		if scope == porterReadinessScopeFullRelease && ctx == porterContextSource {
			checks = append(checks,
				checkReleaseVersionAgreementAt(resolveAetherRootPath(), hubDir, resolveVersion()),
				checkSourceSurfaceAlignmentAt(resolveAetherRootPath()),
				checkTsHostReleaseArtifactsAt(resolveAetherRootPath()),
			)
			checks = append(checks, buildFullReleaseCommandChecks(resolveAetherRootPath(), false)...)
		}
		overall := "critical"
		recoveryCommands := []string{"Run aether install to populate the hub"}
		for _, c := range checks {
			if c.Status == "fail" && c.RecoveryCommand != "" {
				recoveryCommands = append(recoveryCommands, c.RecoveryCommand)
			}
		}
		result := buildPorterResult(ctx, string(channel), scope, checks)
		result.Overall = overall
		result.RecoveryCommands = uniqueSortedStrings(recoveryCommands)
		result = recordPorterReadinessEvidenceForOutput(result)
		return renderPorterResult(cmd, result)
	}

	checks := buildPorterChecksForContextWithScope(ctx, string(channel), false, scope)

	// Aggregate results
	result := buildPorterResult(ctx, string(channel), scope, checks)
	result = recordPorterReadinessEvidenceForOutput(result)

	return renderPorterResult(cmd, result)
}

// buildPorterChecks constructs the full set of porter checks based on repo context.
func buildPorterChecks(channel string, skipTests bool) []integrityCheck {
	ctx := detectPorterContext()
	return buildPorterChecksForContext(ctx, channel, skipTests)
}

// buildPorterChecksForContext is the testable core that accepts an explicit context.
// Source repos get version alignment, companion, test, changelog, and publish readiness.
// Consumer repos get hub sync and update readiness.
func buildPorterChecksForContext(ctx porterContext, channel string, skipTests bool) []integrityCheck {
	return buildPorterChecksForContextWithScope(ctx, channel, skipTests, porterReadinessScopeQuick)
}

func buildPorterChecksForContextWithScope(ctx porterContext, channel string, skipTests bool, scope porterReadinessScope) []integrityCheck {
	gitStatus := checkGitStatus()
	gitStashes := checkGitStashes()
	gitWorktrees := checkGitWorktrees()

	homeDir, _ := os.UserHomeDir()
	rc := normalizeRuntimeChannel(channel)
	hubDir := resolveHubPathForHome(homeDir, rc)
	hubVersion := readHubVersionAtPath(hubDir)
	binaryVersion := resolveVersion()

	if ctx == porterContextSource {
		var testCheck integrityCheck
		if skipTests {
			testCheck = integrityCheck{
				Name:    "Test status",
				Status:  "skip",
				Message: "Skipped (test mode)",
			}
		} else {
			testCheck = checkTestStatus()
		}

		checks := []integrityCheck{
			checkSourceVersion(),
			checkBinaryVersion(),
			checkHubVersion(hubDir),
			checkHubCompanionFiles(hubDir),
			checkDownstreamSimulation(hubDir, hubVersion, binaryVersion, rc),
			gitStatus,
			gitStashes,
			gitWorktrees,
			testCheck,
			checkChangelogCompleteness(),
		}
		if scope == porterReadinessScopeFullRelease {
			checks = append(checks,
				checkReleaseVersionAgreementAt(resolveAetherRootPath(), hubDir, binaryVersion),
				checkSourceSurfaceAlignmentAt(resolveAetherRootPath()),
				checkTsHostReleaseArtifactsAt(resolveAetherRootPath()),
			)
			checks = append(checks, buildFullReleaseCommandChecks(resolveAetherRootPath(), skipTests)...)
		}
		return checks
	}

	// Consumer repo checks
	return []integrityCheck{
		checkBinaryVersion(),
		checkHubVersion(hubDir),
		checkHubCompanionSync(hubDir),
		gitStatus,
		gitStashes,
		gitWorktrees,
	}
}

func porterReadinessScopeFromCommand(cmd *cobra.Command) porterReadinessScope {
	fullRelease, _ := cmd.Flags().GetBool("full-release")
	if fullRelease {
		return porterReadinessScopeFullRelease
	}
	return porterReadinessScopeQuick
}

func buildPorterResult(ctx porterContext, channel string, scope porterReadinessScope, checks []integrityCheck) integrityResult {
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

	return integrityResult{
		Context:          string(ctx),
		Channel:          string(normalizeRuntimeChannel(channel)),
		Scope:            string(scope),
		GeneratedAt:      time.Now().UTC().Format(time.RFC3339),
		Checks:           checks,
		Overall:          overall,
		SkippedChecks:    skippedPorterChecks(checks),
		RecoveryCommands: uniqueSortedStrings(recoveryCommands),
	}
}

func skippedPorterChecks(checks []integrityCheck) []string {
	skipped := []string{}
	for _, check := range checks {
		if check.Status == "skip" {
			skipped = append(skipped, check.Name)
		}
	}
	return uniqueSortedStrings(skipped)
}

func recordPorterReadinessEvidenceForOutput(result integrityResult) integrityResult {
	if store == nil {
		return result
	}
	result.Evidence = displayDataPath(porterReadinessReportRel)
	if _, err := recordPorterReadinessEvidence(result); err != nil {
		result.Evidence = ""
		result.EvidenceError = err.Error()
	}
	return result
}

func recordPorterReadinessEvidence(result integrityResult) (string, error) {
	if store == nil {
		return "", nil
	}
	if strings.TrimSpace(result.GeneratedAt) == "" {
		result.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if result.SkippedChecks == nil {
		result.SkippedChecks = skippedPorterChecks(result.Checks)
	}
	if err := store.SaveJSON(porterReadinessReportRel, result); err != nil {
		return "", err
	}
	return porterReadinessReportRel, nil
}

func checkReleaseVersionAgreementAt(root, hubDir, binaryVersion string) integrityCheck {
	sourceVersion := readRepoVersion(root)
	if sourceVersion == "" {
		sourceVersion = resolveSourceVersion()
	}
	npmVersion := readNpmPackageVersion(root)
	hubVersion := readHubVersionAtPath(hubDir)
	binaryVersion = normalizeVersion(binaryVersion)

	versions := []struct {
		label   string
		version string
	}{
		{"source version", sourceVersion},
		{"npm package version", npmVersion},
		{"binary version", binaryVersion},
		{"hub version", hubVersion},
	}

	details := map[string]interface{}{}
	var missing []string
	for _, v := range versions {
		details[strings.ReplaceAll(v.label, " ", "_")] = v.version
		if v.version == "" || v.version == "unknown" {
			missing = append(missing, v.label)
		}
	}
	if len(missing) > 0 {
		return integrityCheck{
			Name:            "Release version agreement",
			Status:          "fail",
			Message:         fmt.Sprintf("Missing release version evidence: %s", strings.Join(missing, ", ")),
			RecoveryCommand: "Align .aether/version.json, npm/package.json, binary, and hub versions before release",
			Details:         details,
		}
	}

	reference := sourceVersion
	var mismatches []string
	for _, v := range versions {
		if v.version != reference {
			mismatches = append(mismatches, fmt.Sprintf("%s %s", v.label, v.version))
		}
	}
	if len(mismatches) > 0 {
		return integrityCheck{
			Name:            "Release version agreement",
			Status:          "fail",
			Message:         fmt.Sprintf("Release versions disagree: source version %s; %s", sourceVersion, strings.Join(mismatches, "; ")),
			RecoveryCommand: "Align .aether/version.json, npm/package.json, binary, and hub versions before release",
			Details:         details,
		}
	}

	return integrityCheck{
		Name:    "Release version agreement",
		Status:  "pass",
		Message: fmt.Sprintf("Source, npm package, binary, and hub all report %s", sourceVersion),
		Details: details,
	}
}

func readNpmPackageVersion(root string) string {
	root = strings.TrimSpace(root)
	if root == "" {
		if cwd, err := os.Getwd(); err == nil {
			root = findAetherModuleRoot(cwd)
		}
	}
	if root == "" {
		return ""
	}
	return readVersionJSONFile(filepath.Join(root, "npm", "package.json"))
}

func checkSourceSurfaceAlignmentAt(root string) integrityCheck {
	sourceRoot, err := resolveSourceCheckRoot(root)
	if err != nil {
		return integrityCheck{
			Name:            "Source surface alignment",
			Status:          "fail",
			Message:         err.Error(),
			RecoveryCommand: "Run aether source-check --json from the Aether source checkout",
		}
	}
	result := runSourceCheck(sourceRoot)
	if result.OK {
		return integrityCheck{
			Name:    "Source surface alignment",
			Status:  "pass",
			Message: "source-check passed",
			Details: map[string]interface{}{"checked_components": len(result.Components)},
		}
	}
	return integrityCheck{
		Name:            "Source surface alignment",
		Status:          "fail",
		Message:         fmt.Sprintf("source-check found %d issue(s)", len(result.Issues)),
		RecoveryCommand: "Run aether source-check --json and fix reported source surface drift",
		Details:         map[string]interface{}{"issue_count": len(result.Issues)},
	}
}

func checkTsHostReleaseArtifactsAt(root string) integrityCheck {
	if err := validateTsHostSourceArtifacts(root); err != nil {
		return integrityCheck{
			Name:            "TS host release artifacts",
			Status:          "fail",
			Message:         err.Error(),
			RecoveryCommand: "Run npm --prefix .aether/ts-host ci && npm --prefix .aether/ts-host run build, then rerun aether publish",
		}
	}
	return integrityCheck{
		Name:    "TS host release artifacts",
		Status:  "pass",
		Message: "Built TS host entrypoint and package metadata are present",
		Details: map[string]interface{}{
			"entrypoint": filepath.Join(tsHostSourceRelDir, filepath.FromSlash(tsHostEntryRelPath)),
		},
	}
}

type porterCommandCheck struct {
	Name            string
	Dir             string
	Args            []string
	Env             map[string]string
	Timeout         time.Duration
	RecoveryCommand string
}

func buildFullReleaseCommandChecks(root string, skipCommands bool) []integrityCheck {
	root = strings.TrimSpace(root)
	if root == "" {
		root = resolveAetherRootPath()
	}
	specs := []porterCommandCheck{
		{
			Name:            "Go vet",
			Dir:             root,
			Args:            []string{"go", "vet", "./..."},
			Timeout:         2 * time.Minute,
			RecoveryCommand: "Fix go vet findings before release",
		},
		{
			Name:            "Go race tests",
			Dir:             root,
			Args:            []string{"go", "test", "./...", "-race", "-count=1"},
			Timeout:         10 * time.Minute,
			RecoveryCommand: "Fix race test failures before release",
		},
		{
			Name:            "Go binary build",
			Dir:             root,
			Args:            []string{"go", "build", "./cmd/aether"},
			Timeout:         2 * time.Minute,
			RecoveryCommand: "Fix binary build failures before release",
		},
		{
			Name:            "GoReleaser config",
			Dir:             root,
			Args:            []string{"goreleaser", "check"},
			Timeout:         2 * time.Minute,
			RecoveryCommand: "Fix GoReleaser config before release",
		},
		{
			Name:            "GoReleaser snapshot",
			Dir:             root,
			Args:            []string{"goreleaser", "release", "--snapshot", "--clean"},
			Env:             map[string]string{"AETHER_RELEASE_VERSION": readRepoVersion(root)},
			Timeout:         10 * time.Minute,
			RecoveryCommand: "Fix GoReleaser snapshot archives or checksums before release",
		},
		{
			Name:            "TS host typecheck",
			Dir:             filepath.Join(root, ".aether", "ts-host"),
			Args:            []string{"npm", "run", "typecheck"},
			Timeout:         2 * time.Minute,
			RecoveryCommand: "Fix TypeScript host typecheck failures before release",
		},
		{
			Name:            "TS host tests",
			Dir:             filepath.Join(root, ".aether", "ts-host"),
			Args:            []string{"npm", "test"},
			Timeout:         3 * time.Minute,
			RecoveryCommand: "Fix TypeScript host tests before release",
		},
		{
			Name:            "TS host build",
			Dir:             filepath.Join(root, ".aether", "ts-host"),
			Args:            []string{"npm", "run", "build"},
			Timeout:         2 * time.Minute,
			RecoveryCommand: "Fix TypeScript host build before release",
		},
		{
			Name:            "npm package tests",
			Dir:             filepath.Join(root, "npm"),
			Args:            []string{"npm", "test"},
			Timeout:         2 * time.Minute,
			RecoveryCommand: "Fix npm package tests before release",
		},
		{
			Name:            "Aether binary smoke",
			Dir:             root,
			Args:            []string{"go", "run", "./cmd/aether", "version"},
			Timeout:         1 * time.Minute,
			RecoveryCommand: "Fix Aether binary smoke failure before release",
		},
	}

	checks := make([]integrityCheck, 0, len(specs))
	for _, spec := range specs {
		if skipCommands {
			checks = append(checks, skippedPorterCommandCheck(spec))
			continue
		}
		checks = append(checks, runPorterCommandCheck(spec))
	}
	return checks
}

func skippedPorterCommandCheck(spec porterCommandCheck) integrityCheck {
	return integrityCheck{
		Name:    spec.Name,
		Status:  "skip",
		Message: "Skipped by test-mode porter check construction",
		Details: map[string]interface{}{
			"command": strings.Join(spec.Args, " "),
			"dir":     spec.Dir,
		},
	}
}

func runPorterCommandCheck(spec porterCommandCheck) integrityCheck {
	if len(spec.Args) == 0 {
		return integrityCheck{Name: spec.Name, Status: "skip", Message: "No command configured"}
	}
	if strings.TrimSpace(spec.Dir) != "" {
		if _, err := os.Stat(spec.Dir); err != nil {
			return integrityCheck{
				Name:    spec.Name,
				Status:  "skip",
				Message: fmt.Sprintf("Required directory missing: %s", spec.Dir),
				Details: map[string]interface{}{
					"command": strings.Join(spec.Args, " "),
					"dir":     spec.Dir,
				},
			}
		}
	}

	timeout := spec.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, spec.Args[0], spec.Args[1:]...)
	cmd.Dir = spec.Dir
	cmd.Env = mergePorterEnvironment(porterTestEnv(os.Environ()), spec.Env)
	output, err := cmd.CombinedOutput()
	details := map[string]interface{}{
		"command": strings.Join(spec.Args, " "),
		"dir":     spec.Dir,
	}
	if ctx.Err() == context.DeadlineExceeded {
		return integrityCheck{
			Name:            spec.Name,
			Status:          "fail",
			Message:         fmt.Sprintf("Timed out after %s", timeout),
			RecoveryCommand: spec.RecoveryCommand,
			Details:         details,
		}
	}
	if err != nil {
		return integrityCheck{
			Name:            spec.Name,
			Status:          "fail",
			Message:         lastPorterOutputLines(output, 5),
			RecoveryCommand: spec.RecoveryCommand,
			Details:         details,
		}
	}
	return integrityCheck{
		Name:    spec.Name,
		Status:  "pass",
		Message: "Command passed",
		Details: details,
	}
}

func mergePorterEnvironment(base []string, replacements map[string]string) []string {
	if len(replacements) == 0 {
		return append([]string(nil), base...)
	}
	result := make([]string, 0, len(base)+len(replacements))
	seen := make(map[string]bool, len(replacements))
	for _, entry := range base {
		key, _, found := strings.Cut(entry, "=")
		if !found {
			result = append(result, entry)
			continue
		}
		if replacement, ok := replacements[key]; ok {
			if !seen[key] {
				result = append(result, key+"="+replacement)
				seen[key] = true
			}
			continue
		}
		result = append(result, entry)
	}
	keys := make([]string, 0, len(replacements))
	for key := range replacements {
		if !seen[key] {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		result = append(result, key+"="+replacements[key])
	}
	return result
}

func lastPorterOutputLines(output []byte, limit int) string {
	text := strings.TrimSpace(string(output))
	if text == "" {
		return "Command failed without output"
	}
	lines := strings.Split(text, "\n")
	if limit > 0 && len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}
	return codex.SanitizeWorkerDiagnosticOutput(strings.Join(lines, "\n"))
}

// checkHubCompanionSync wraps checkHubCompanionFiles for consumer repo context.
func checkHubCompanionSync(hubDir string) integrityCheck {
	result := checkHubCompanionFiles(hubDir)
	result.Name = "Hub companion sync"
	if result.Status == "pass" {
		result.Message = "Companion files synchronized with hub"
	} else {
		result.RecoveryCommand = "Run aether update --force to sync companion files"
	}
	return result
}

func renderPorterResult(cmd *cobra.Command, result integrityResult) error {
	jsonOut, _ := cmd.Flags().GetBool("json")
	if jsonOut || !shouldRenderVisualOutput(stdout) {
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
	b.WriteString(fmt.Sprintf("Channel: %s\n", result.Channel))
	b.WriteString(fmt.Sprintf("Context: %s repo\n", result.Context))
	b.WriteString(fmt.Sprintf("Scope: %s\n", porterReadinessScopeVisualLabel(result.Scope)))
	if note := porterReadinessScopeVisualNote(result.Scope); note != "" {
		b.WriteString(note)
		b.WriteString("\n")
	}
	if evidence := strings.TrimSpace(result.Evidence); evidence != "" {
		b.WriteString(fmt.Sprintf("Evidence: %s\n", evidence))
	}
	if len(result.SkippedChecks) > 0 {
		b.WriteString(fmt.Sprintf("Skipped: %s\n", strings.Join(result.SkippedChecks, ", ")))
	}
	if errText := strings.TrimSpace(result.EvidenceError); errText != "" {
		b.WriteString(fmt.Sprintf("Evidence warning: %s\n", errText))
	}
	b.WriteString("\n")

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

func porterReadinessScopeVisualLabel(scope string) string {
	switch porterReadinessScope(strings.TrimSpace(scope)) {
	case porterReadinessScopeFullRelease:
		return "full-release"
	default:
		return "quick"
	}
}

func porterReadinessScopeVisualNote(scope string) string {
	switch porterReadinessScope(strings.TrimSpace(scope)) {
	case porterReadinessScopeFullRelease:
		return "Full-release scope runs slower release-tool checks and blocks on failures."
	default:
		return "Quick scope omits slower release-tool checks; use --full-release before publishing."
	}
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

// checkGitStashes runs `git stash list` and warns if unapplied stashes exist,
// since they may represent incomplete or at-risk work.
func checkGitStashes() integrityCheck {
	out, err := exec.Command("git", "stash", "list").Output()
	if err != nil {
		return integrityCheck{
			Name:    "Git stashes",
			Status:  "skip",
			Message: "Not in a git repository",
		}
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	if count == 0 {
		return integrityCheck{
			Name:    "Git stashes",
			Status:  "pass",
			Message: "No stashes found",
		}
	}
	return integrityCheck{
		Name:            "Git stashes",
		Status:          "fail",
		Message:         fmt.Sprintf("%d unapplied stash(es) detected", count),
		RecoveryCommand: "Apply or drop stashes with git stash pop / git stash drop",
	}
}

// checkGitWorktrees detects active git worktrees and fails if any exist beyond
// the main working tree, since they may contain unmerged work.
func checkGitWorktrees() integrityCheck {
	out, err := exec.Command("git", "worktree", "list", "--porcelain").Output()
	if err != nil {
		return integrityCheck{
			Name:    "Git worktrees",
			Status:  "skip",
			Message: "Not in a git repository or worktrees unsupported",
		}
	}
	paths := parseWorktreePaths(string(out))
	// The first entry is always the main working tree
	if len(paths) <= 1 {
		return integrityCheck{
			Name:    "Git worktrees",
			Status:  "pass",
			Message: "No additional worktrees active",
		}
	}
	extraCount := len(paths) - 1
	return integrityCheck{
		Name:            "Git worktrees",
		Status:          "fail",
		Message:         fmt.Sprintf("%d active worktree(s) detected", extraCount),
		RecoveryCommand: "Merge or remove worktrees before delivery",
		Details:         map[string]interface{}{"worktree_paths": paths[1:]},
	}
}

// checkTestStatus runs `go test ./...` and fails if any test fails.
func checkTestStatus() integrityCheck {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "test", "./...", "-count=1")
	cmd.Env = porterTestEnv(os.Environ())
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
		return integrityCheck{
			Name:            "Test status",
			Status:          "fail",
			Message:         lastPorterOutputLines(output, 5),
			RecoveryCommand: "Fix failing tests before delivery",
		}
	}
	return integrityCheck{
		Name:    "Test status",
		Status:  "pass",
		Message: "All tests pass",
	}
}

func porterTestEnv(environ []string) []string {
	out := make([]string, 0, len(environ)+1)
	outputModeSet := false
	for _, entry := range environ {
		key := entry
		if idx := strings.IndexByte(entry, '='); idx >= 0 {
			key = entry[:idx]
		}
		if key == "AETHER_OUTPUT_MODE" {
			if !outputModeSet {
				out = append(out, "AETHER_OUTPUT_MODE=")
				outputModeSet = true
			}
			continue
		}
		out = append(out, entry)
	}
	if !outputModeSet {
		out = append(out, "AETHER_OUTPUT_MODE=")
	}
	return out
}

// checkChangelogCompleteness reads CHANGELOG.md and checks for an entry matching
// the current source version.
func checkChangelogCompleteness() integrityCheck {
	version := resolveSourceVersion()
	if version == "unknown" {
		return integrityCheck{
			Name:    "Changelog completeness",
			Status:  "skip",
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
			Name:    "Changelog completeness",
			Status:  "skip",
			Message: fmt.Sprintf("Cannot read CHANGELOG.md: %v", err),
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
