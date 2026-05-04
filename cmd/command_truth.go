package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
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
	Use:   "maturity",
	Short: "View colony maturity journey",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}
		state, err := loadActiveColonyState()
		if err != nil {
			outputError(1, colonyStateLoadMessage(err), nil)
			return nil
		}
		result := buildMaturityResult(state)
		outputWorkflow(result, renderMaturityVisual(result))
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
		result, err := runMigrateState(dryRun)
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

	rootCmd.AddCommand(maturityCmd)
	rootCmd.AddCommand(quickCmd)
	rootCmd.AddCommand(bumpVersionCmd)
	rootCmd.AddCommand(migrateStateCmd)
	rootCmd.AddCommand(verifyCastesCmd)
}

func buildMaturityResult(state colony.ColonyState) map[string]interface{} {
	milestone, total, completed := deriveMilestoneProgress(state)
	progress := 0
	if total > 0 {
		progress = int(float64(completed) / float64(total) * 100)
	}
	ageDays := 0
	initializedAt := ""
	if state.InitializedAt != nil {
		initializedAt = state.InitializedAt.UTC().Format(time.RFC3339)
		ageDays = int(time.Since(*state.InitializedAt).Hours() / 24)
		if ageDays < 0 {
			ageDays = 0
		}
	}
	return map[string]interface{}{
		"mode":             "maturity",
		"goal":             goalText(state),
		"milestone":        milestone,
		"version":          resolveVersion(resolveAetherRoot()),
		"phases_completed": completed,
		"total_phases":     total,
		"progress_percent": progress,
		"colony_age_days":  ageDays,
		"initialized_at":   initializedAt,
		"state":            string(state.State),
		"next":             nextCommandFromState(state),
	}
}

func deriveMilestoneProgress(state colony.ColonyState) (string, int, int) {
	total := len(state.Plan.Phases)
	completed := 0
	for _, phase := range state.Plan.Phases {
		if phase.Status == colony.PhaseCompleted {
			completed++
		}
	}
	milestone := strings.TrimSpace(state.Milestone)
	if milestone == "" && total > 0 {
		ratio := float64(completed) / float64(total)
		switch {
		case ratio >= 1:
			milestone = "Sealed Chambers"
		case ratio >= 0.75:
			milestone = "Ventilated Nest"
		case ratio >= 0.5:
			milestone = "Brood Stable"
		case ratio >= 0.25:
			milestone = "Open Chambers"
		default:
			milestone = "First Mound"
		}
	}
	if milestone == "" {
		milestone = "First Mound"
	}
	return milestone, total, completed
}

func goalText(state colony.ColonyState) string {
	if state.Goal == nil {
		return ""
	}
	return strings.TrimSpace(*state.Goal)
}

func renderMaturityVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("maturity"), "Maturity"))
	b.WriteString(visualDivider)
	if goal := strings.TrimSpace(stringValue(result["goal"])); goal != "" {
		b.WriteString("Goal: ")
		b.WriteString(goal)
		b.WriteString("\n")
	}
	b.WriteString("Milestone: ")
	b.WriteString(emptyFallback(stringValue(result["milestone"]), "First Mound"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Progress: %d/%d phases (%d%%)\n", intValue(result["phases_completed"]), intValue(result["total_phases"]), intValue(result["progress_percent"])))
	if age := intValue(result["colony_age_days"]); age > 0 {
		b.WriteString(fmt.Sprintf("Age: %d days\n", age))
	}
	b.WriteString("Version: ")
	b.WriteString(emptyFallback(stringValue(result["version"]), "unknown"))
	b.WriteString("\n")
	b.WriteString(renderNextUp(fmt.Sprintf("Run `%s` for the next lifecycle step.", emptyFallback(stringValue(result["next"]), "aether status"))))
	return b.String()
}

func runQuickScout(question string, timeout time.Duration) (map[string]interface{}, error) {
	root := skillWorkspaceRoot()
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
		Root:             root,
		Timeout:          timeout,
		SkillSection:     resolveSkillSectionForWorkflow("quick", "scout", question),
		PheromoneSection: resolvePheromoneSection(),
	})
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"mode":        "quick",
		"question":    question,
		"worker_name": workerResult.WorkerName,
		"status":      emptyFallback(workerResult.Status, "completed"),
		"summary":     strings.TrimSpace(workerResult.Summary),
		"raw_output":  strings.TrimSpace(workerResult.RawOutput),
		"duration_ms": workerResult.Duration.Milliseconds(),
		"files":       workerResult.FilesModified,
		"next":        "aether status",
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
	b.WriteString(visualDivider)
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
	b.WriteString(visualDivider)
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

func runMigrateState(dryRun bool) (map[string]interface{}, error) {
	var raw map[string]interface{}
	if err := store.LoadJSON("COLONY_STATE.json", &raw); err != nil {
		return nil, fmt.Errorf("COLONY_STATE.json not found: %w", err)
	}
	fromVersion := strings.TrimSpace(stringValue(raw["version"]))
	if fromVersion == "" {
		fromVersion = "legacy"
	}
	if fromVersion == "3.0" {
		return map[string]interface{}{
			"mode":     "migrate-state",
			"migrated": false,
			"from":     fromVersion,
			"to":       "3.0",
			"reason":   "already current",
			"dry_run":  dryRun,
			"next":     "aether medic --deep",
		}, nil
	}
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		return nil, fmt.Errorf("legacy state is not compatible with automatic migration: %w", err)
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
		fmt.Sprintf("%s|state_migrated|migrate-state|Migrated COLONY_STATE.json from %s to 3.0", time.Now().UTC().Format(time.RFC3339), fromVersion),
	)
	backupPath := ""
	if !dryRun {
		backupPath = filepath.Join("backups", fmt.Sprintf("COLONY_STATE.pre-migrate.%s.json", time.Now().UTC().Format("20060102-150405")))
		data, _ := json.MarshalIndent(raw, "", "  ")
		if err := store.AtomicWrite(backupPath, append(data, '\n')); err != nil {
			return nil, fmt.Errorf("write migration backup: %w", err)
		}
		if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
			return nil, fmt.Errorf("write migrated state: %w", err)
		}
	}
	return map[string]interface{}{
		"mode":        "migrate-state",
		"migrated":    !dryRun,
		"from":        fromVersion,
		"to":          "3.0",
		"dry_run":     dryRun,
		"backup_path": backupPath,
		"next":        "aether medic --deep",
	}, nil
}

func renderMigrateStateVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("migrate-state"), "Migrate State"))
	b.WriteString(visualDivider)
	if boolValue(result["migrated"]) {
		b.WriteString("State migrated.\n")
	} else {
		b.WriteString("No migration applied.\n")
	}
	b.WriteString(fmt.Sprintf("Version: %s -> %s\n", emptyFallback(stringValue(result["from"]), "legacy"), emptyFallback(stringValue(result["to"]), "3.0")))
	if backup := strings.TrimSpace(stringValue(result["backup_path"])); backup != "" {
		b.WriteString("Backup: ")
		b.WriteString(backup)
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
	b.WriteString(visualDivider)
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
