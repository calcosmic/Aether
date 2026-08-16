package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// `aether oracle selftest` answers the question the operator actually has
// before committing an hour to a deep run: does this work at all?
//
// It runs one real iteration through the real dispatcher. A mocked selftest
// would pass in exactly the situation worth catching -- no worker dispatcher,
// a missing agent definition, a provider that cannot be reached -- which is why
// the repo's Definition of Done asks for a command that fails when the
// requirement is unmet rather than one that reports success by assertion.

const oracleSelftestTopic = "Which file in this repository documents how the project is built and tested?"

type oracleSelftestCheck struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail,omitempty"`
}

// oracleSelftestPaths keeps the probe run out of the real workspace so a
// selftest can never destroy research the operator still wants.
func oracleSelftestPaths(root string) oraclePaths {
	paths := oracleWorkspacePaths(root)
	dir := filepath.Join(paths.Dir, ".selftest")
	return oraclePaths{
		Root:             root,
		Dir:              dir,
		ArchiveDir:       filepath.Join(dir, "archive"),
		DiscoveriesDir:   filepath.Join(dir, "discoveries"),
		ResponsesDir:     filepath.Join(dir, "responses"),
		StatePath:        filepath.Join(dir, "state.json"),
		PlanPath:         filepath.Join(dir, "plan.json"),
		GapsPath:         filepath.Join(dir, "gaps.md"),
		SynthesisPath:    filepath.Join(dir, "synthesis.md"),
		ResearchPlanPath: filepath.Join(dir, "research-plan.md"),
		ProgressPath:     filepath.Join(dir, "progress.jsonl"),
		StopPath:         filepath.Join(dir, ".stop"),
		LoopPath:         filepath.Join(dir, ".loop-active"),
		AgentName:        paths.AgentName,
	}
}

func runOracleSelftest(root string, dryRun bool) (map[string]interface{}, error) {
	checks := make([]oracleSelftestCheck, 0, 8)
	record := func(name string, passed bool, format string, args ...interface{}) {
		checks = append(checks, oracleSelftestCheck{Name: name, Passed: passed, Detail: fmt.Sprintf(format, args...)})
	}

	// 1. Is there a worker dispatcher at all? This is the failure the real loop
	//    reports as "no worker dispatcher is available", and it is the most
	//    common reason a run goes nowhere.
	invoker := newOracleWorkerInvoker()
	if invoker == nil {
		record("dispatcher", false, "no worker dispatcher is available")
		return oracleSelftestResult(checks, dryRun, ""), fmt.Errorf("oracle selftest failed: no worker dispatcher is available")
	}
	platform := oracleInvokerPlatform(invoker)
	record("dispatcher", true, "using %s", platform)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if !invoker.IsAvailable(ctx) {
		record("dispatcher reachable", false, "%s", dispatchAvailabilityMessage(invoker))
		return oracleSelftestResult(checks, dryRun, platform), fmt.Errorf("oracle selftest failed: %s", dispatchAvailabilityMessage(invoker))
	}
	record("dispatcher reachable", true, "%s responded", platform)

	// 2. Does the Oracle agent definition this platform needs actually exist?
	agentPath := dispatchAgentPath(root, invoker, oracleWorkspacePaths(root).AgentName)
	if err := invoker.ValidateAgent(agentPath); err != nil {
		record("agent definition", false, "%v", err)
		return oracleSelftestResult(checks, dryRun, platform), fmt.Errorf("oracle selftest failed: oracle agent unavailable: %w", err)
	}
	record("agent definition", true, "%s", emptyFallback(agentPath, "resolved by the platform"))

	if dryRun {
		return oracleSelftestResult(checks, dryRun, platform), nil
	}

	// 3. One real round, in a throwaway workspace.
	paths := oracleSelftestPaths(root)
	_ = os.RemoveAll(paths.Dir)
	defer os.RemoveAll(paths.Dir)
	if err := ensureOracleWorkspace(paths); err != nil {
		return oracleSelftestResult(checks, dryRun, platform), err
	}

	detectedType, languages, frameworks := detectOracleProjectProfile(root)
	now := time.Now().UTC().Format(time.RFC3339)
	scopeProfile, err := resolveOracleScope(oracleSelftestTopic, "repo")
	if err != nil {
		return oracleSelftestResult(checks, dryRun, platform), err
	}
	state := oracleStateFile{
		Version:          "1.1",
		Topic:            oracleSelftestTopic,
		CoreQuestion:     oracleSelftestTopic,
		Scope:            scopeProfile.Scope,
		Template:         "custom",
		Phase:            "survey",
		Iteration:        0,
		MaxIterations:    1,
		TargetConfidence: 60,
		StartedAt:        now,
		LastUpdated:      now,
		Status:           "active",
		Strategy:         defaultOracleStrategy,
		Platform:         platform,
		ControllerPID:    os.Getpid(),
		Depth:            "Selftest (1 iteration)",
	}
	plan := oraclePlanFile{
		Version:     "1.1",
		Sources:     map[string]oracleSource{},
		Questions:   buildOracleQuestionPlan(oracleSelftestTopic, "", detectedType, oracleSelftestTopic, scopeProfile)[:1],
		CreatedAt:   now,
		LastUpdated: now,
	}
	if err := writeOracleStateFile(paths.StatePath, state); err != nil {
		return oracleSelftestResult(checks, dryRun, platform), err
	}
	if err := writeOraclePlanFile(paths.PlanPath, plan); err != nil {
		return oracleSelftestResult(checks, dryRun, platform), err
	}
	if err := writeOracleDerivedReports(paths, state, plan); err != nil {
		return oracleSelftestResult(checks, dryRun, platform), err
	}
	if err := writeOracleLoopMarker(paths.LoopPath, state); err != nil {
		return oracleSelftestResult(checks, dryRun, platform), err
	}

	emitVisualLine("🔮 running one real research round to prove the loop works...")
	if _, err := runOracleLoop(paths, detectedType, languages, frameworks); err != nil {
		record("research round", false, "%v", err)
		return oracleSelftestResult(checks, dryRun, platform), fmt.Errorf("oracle selftest failed: %w", err)
	}
	record("research round", true, "one iteration completed")

	// 4. Did the round leave behind everything a real run depends on?
	responses, _ := filepath.Glob(filepath.Join(paths.ResponsesDir, "*.json"))
	record("worker response written", len(responses) > 0, "%d file(s) in responses/", len(responses))

	discoveries, _ := filepath.Glob(filepath.Join(paths.DiscoveriesDir, "*.json"))
	record("iteration artifact written", len(discoveries) > 0, "%d file(s) in discoveries/", len(discoveries))

	finalPlan, planErr := loadOraclePlanFile(paths.PlanPath)
	findings := 0
	if planErr == nil {
		for _, question := range finalPlan.Questions {
			findings += len(question.KeyFindings)
		}
	}
	record("findings recorded", findings > 0, "%d finding(s) merged into plan.json", findings)

	synthesis, _ := os.ReadFile(paths.SynthesisPath)
	record("research written up", len(strings.TrimSpace(string(synthesis))) > 0, "%d bytes of synthesis", len(synthesis))

	// 5. The progress log is what `--follow` streams. If it is empty the
	//    operator is back to watching a silent run.
	progress, _ := os.ReadFile(paths.ProgressPath)
	progressText := string(progress)
	hasRounds := strings.Contains(progressText, oracleProgressEventRunStart) &&
		strings.Contains(progressText, oracleProgressEventIterationStart) &&
		strings.Contains(progressText, oracleProgressEventRunEnd)
	record("progress log usable", hasRounds, "%d bytes, start/round/end present: %t", len(progress), hasRounds)

	result := oracleSelftestResult(checks, dryRun, platform)
	if failed, _ := result["failed"].(int); failed > 0 {
		return result, fmt.Errorf("oracle selftest failed: %d of %d checks did not pass", failed, len(checks))
	}
	return result, nil
}

func oracleSelftestResult(checks []oracleSelftestCheck, dryRun bool, platform string) map[string]interface{} {
	failed := 0
	for _, check := range checks {
		if !check.Passed {
			failed++
		}
	}
	return map[string]interface{}{
		"mode":     "selftest",
		"dry_run":  dryRun,
		"platform": platform,
		"checks":   checks,
		"passed":   len(checks) - failed,
		"failed":   failed,
		"ok":       failed == 0,
	}
}

func renderOracleSelftest(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner("🔮🐜", "Oracle Selftest"))
	b.WriteString(visualDividerStr())
	if dryRun, _ := result["dry_run"].(bool); dryRun {
		b.WriteString("Setup checks only -- no research worker was run.\n\n")
	}
	if checks, ok := result["checks"].([]oracleSelftestCheck); ok {
		for _, check := range checks {
			mark := "✓"
			if !check.Passed {
				mark = "✗"
			}
			fmt.Fprintf(&b, "  %s %-26s %s\n", mark, check.Name, check.Detail)
		}
	}
	failed, _ := result["failed"].(int)
	passed, _ := result["passed"].(int)
	b.WriteString("\n")
	if failed == 0 {
		fmt.Fprintf(&b, "All %d checks passed. Deep research will run.\n", passed)
	} else {
		fmt.Fprintf(&b, "%d of %d checks failed. Fix these before starting a long run.\n", failed, passed+failed)
	}
	return strings.TrimSpace(b.String())
}

// oracleRecoverStaleRun is the mutating half that `aether oracle status` used
// to perform silently. Inspection reports; repair is asked for.
func oracleRecoverStaleRun(root string) (map[string]interface{}, error) {
	paths := oracleWorkspacePaths(root)
	state, err := loadOracleStateFile(paths.StatePath)
	if err != nil {
		return nil, fmt.Errorf("no Oracle run to recover: %s is unavailable", paths.StatePath)
	}
	if !oracleStateHasStaleController(state) {
		return map[string]interface{}{
			"mode":     "recover",
			"repaired": false,
			"status":   state.Status,
			"summary":  "Nothing to recover: no stale Oracle controller was found.",
		}, nil
	}

	pid := state.ControllerPID
	state.Status = "blocked"
	state.StopReason = "stale_controller"
	state.Summary = fmt.Sprintf("Oracle controller %d is no longer running; the run was marked blocked so a new one can start.", pid)
	state.LastUpdated = time.Now().UTC().Format(time.RFC3339)
	if err := writeOracleStateFile(paths.StatePath, state); err != nil {
		return nil, err
	}
	_ = os.Remove(paths.LoopPath)

	return map[string]interface{}{
		"mode":           "recover",
		"repaired":       true,
		"controller_pid": pid,
		"status":         state.Status,
		"stop_reason":    state.StopReason,
		"summary":        state.Summary,
		"next":           "aether oracle --from-brief",
	}, nil
}
