package cmd

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

type codexContinueExternalDispatch struct {
	Stage         string   `json:"stage"`
	Wave          int      `json:"wave"`
	Caste         string   `json:"caste"`
	AgentName     string   `json:"agent_name,omitempty"`
	Name          string   `json:"name"`
	Task          string   `json:"task"`
	TaskID        string   `json:"task_id"`
	Timeout       int      `json:"timeout_seconds,omitempty"`
	Status        string   `json:"status"`
	Summary       string   `json:"summary,omitempty"`
	Blockers      []string `json:"blockers,omitempty"`
	Duration      float64  `json:"duration,omitempty"`
	Brief         string   `json:"brief,omitempty"`
	SkillSection  string   `json:"skill_section,omitempty"`
	SkillCount    int      `json:"skill_count,omitempty"`
	ColonySkills  int      `json:"colony_skill_count,omitempty"`
	DomainSkills  int      `json:"domain_skill_count,omitempty"`
	MatchedSkills []string `json:"matched_skills,omitempty"`
}

type codexContinuePlanManifest struct {
	Phase             int                             `json:"phase"`
	PhaseName         string                          `json:"phase_name"`
	Root              string                          `json:"root"`
	GeneratedAt       string                          `json:"generated_at"`
	BuildManifest     string                          `json:"build_manifest,omitempty"`
	Verification      codexContinueVerificationReport `json:"verification"`
	Assessment        codexContinueAssessment         `json:"assessment"`
	ReconcileTaskIDs  []string                        `json:"reconcile_task_ids,omitempty"`
	WorkerTimeout     int                             `json:"worker_timeout_seconds,omitempty"`
	Dispatches        []codexContinueExternalDispatch `json:"dispatches"`
	DispatchMode      string                          `json:"dispatch_mode"`
	FinalizeSurface   string                          `json:"finalize_surface"`
	RequiresFinalizer bool                            `json:"requires_finalizer"`
}

func runCodexContinuePlanOnly(root string, options codexContinueOptions) (map[string]interface{}, colony.ColonyState, colony.Phase, []codexContinueExternalDispatch, error) {
	if store == nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("no store initialized")
	}

	state, err := loadActiveColonyState()
	if err != nil {
		return nil, state, colony.Phase{}, nil, fmt.Errorf("%s", colonyStateLoadMessage(err))
	}
	if len(state.Plan.Phases) == 0 {
		return nil, state, colony.Phase{}, nil, fmt.Errorf("No project plan. Run `aether plan` first.")
	}
	if state.State != colony.StateEXECUTING && state.State != colony.StateBUILT {
		return nil, state, colony.Phase{}, nil, fmt.Errorf("No active phase to continue. Run `aether build <phase>` first.")
	}
	if state.CurrentPhase < 1 || state.CurrentPhase > len(state.Plan.Phases) {
		return nil, state, colony.Phase{}, nil, fmt.Errorf("No active phase to continue. Run `aether build <phase>` first.")
	}

	phase := state.Plan.Phases[state.CurrentPhase-1]
	if phase.Status != colony.PhaseInProgress {
		return nil, state, colony.Phase{}, nil, fmt.Errorf("phase %d is not in progress; run `aether build %d` first", phase.ID, phase.ID)
	}
	if err := validateContinueReconcileTasks(phase, options.ReconcileTaskIDs); err != nil {
		return nil, state, phase, nil, err
	}

	manifest := loadCodexContinueManifest(phase.ID)
	if state.BuildStartedAt == nil && !manifest.Present {
		return nil, state, phase, nil, fmt.Errorf("No active build packet found. Run `aether build <phase>` first.")
	}
	if abandoned, _, summary := detectAbandonedBuild(manifest, state); abandoned {
		return nil, state, phase, nil, fmt.Errorf("%s", summary)
	}

	now := time.Now().UTC()
	verification := runCodexContinueVerificationSnapshot(root, phase, manifest, now)
	assessment := assessCodexContinue(phase, manifest, verification, options, now)
	verification = attachContinueClaimVerification(verification, assessment)
	dispatches := plannedExternalContinueDispatches(root, phase, manifest, verification, assessment, options.WorkerTimeout)
	plan := codexContinuePlanManifest{
		Phase:             phase.ID,
		PhaseName:         phase.Name,
		Root:              root,
		GeneratedAt:       now.Format(time.RFC3339),
		BuildManifest:     displayOptionalDataPath(manifest.Path),
		Verification:      verification,
		Assessment:        assessment,
		ReconcileTaskIDs:  append([]string{}, options.ReconcileTaskIDs...),
		WorkerTimeout:     int(effectiveContinueReviewTimeout(options.WorkerTimeout) / time.Second),
		Dispatches:        dispatches,
		DispatchMode:      "plan-only",
		FinalizeSurface:   "pending",
		RequiresFinalizer: true,
	}

	result := map[string]interface{}{
		"plan_only":         true,
		"phase":             phase.ID,
		"phase_name":        phase.Name,
		"state":             state.State,
		"continue_manifest": plan,
		"verification":      verification,
		"assessment":        assessment,
		"dispatches":        dispatches,
		"dispatch_count":    len(dispatches),
		"wave_count":        countContinueExternalWaves(dispatches),
		"dispatch_mode":     "plan-only",
		"next":              "spawn wrapper continue agents, then record completion",
		"wrapper_contract": map[string]interface{}{
			"source_command":            "AETHER_OUTPUT_MODE=json aether continue --plan-only",
			"spawn_log_required":        true,
			"spawn_complete_required":   true,
			"worker_timeout_seconds":    int(effectiveContinueReviewTimeout(options.WorkerTimeout) / time.Second),
			"finalize_surface":          "pending",
			"runtime_verification_only": true,
		},
	}
	return result, state, phase, dispatches, nil
}

func runCodexContinueVerificationSnapshot(root string, phase colony.Phase, manifest codexContinueManifest, now time.Time) codexContinueVerificationReport {
	commands := resolveCodexVerificationCommands(root)
	steps := []codexVerificationStep{
		runVerificationStep(root, "build", commands.Build),
		runVerificationStep(root, "types", commands.Type),
		runVerificationStep(root, "lint", commands.Lint),
		runVerificationStep(root, "tests", commands.Test),
	}
	claims := verifyCodexBuildClaims(root, manifest)
	watcher := evaluateContinueWatcherVerification(manifest)

	checksPassed := true
	blockers := []string{}
	for _, step := range steps {
		if !step.Passed && !step.Skipped {
			checksPassed = false
			blockers = append(blockers, fmt.Sprintf("%s failed: %s", step.Name, step.Summary))
		}
	}
	if watcher.Present && !watcher.Passed {
		checksPassed = false
		summary := strings.TrimSpace(watcher.Summary)
		if summary == "" {
			summary = "build watcher verification did not complete cleanly"
		}
		blockers = append(blockers, summary)
	}

	return codexContinueVerificationReport{
		Phase:          phase.ID,
		GeneratedAt:    now.Format(time.RFC3339),
		Steps:          steps,
		Claims:         claims,
		Watcher:        watcher,
		ChecksPassed:   checksPassed,
		Passed:         checksPassed,
		BlockingIssues: blockers,
	}
}

func plannedExternalContinueDispatches(root string, phase colony.Phase, manifest codexContinueManifest, verification codexContinueVerificationReport, assessment codexContinueAssessment, workerTimeout time.Duration) []codexContinueExternalDispatch {
	timeoutSeconds := int(effectiveContinueReviewTimeout(workerTimeout) / time.Second)
	watcherSkillAssignment := resolveWorkerSkillAssignment("watcher", "Independent verification before advancement")
	dispatches := []codexContinueExternalDispatch{
		{
			Stage:         "verification",
			Wave:          1,
			Caste:         "watcher",
			AgentName:     codexAgentNameForCaste("watcher"),
			Name:          deterministicAntName("watcher", fmt.Sprintf("phase:%d:continue:watcher", phase.ID)),
			Task:          "Independent verification before advancement",
			TaskID:        fmt.Sprintf("continue-verification-%d", phase.ID),
			Timeout:       timeoutSeconds,
			Status:        "planned",
			Brief:         renderCodexContinueWatcherBrief(root, phase, manifest, verification.Steps, verification.Claims, verification.Watcher),
			SkillSection:  watcherSkillAssignment.Section,
			SkillCount:    watcherSkillAssignment.SkillCount,
			ColonySkills:  watcherSkillAssignment.ColonyCount,
			DomainSkills:  watcherSkillAssignment.DomainCount,
			MatchedSkills: append([]string{}, watcherSkillAssignment.MatchedNames...),
		},
	}
	for _, spec := range codexContinueReviewSpecs {
		assignment := resolveWorkerSkillAssignment(spec.Caste, spec.Task)
		dispatches = append(dispatches, codexContinueExternalDispatch{
			Stage:         "review",
			Wave:          2,
			Caste:         spec.Caste,
			AgentName:     codexAgentNameForCaste(spec.Caste),
			Name:          deterministicAntName(spec.Caste, fmt.Sprintf("phase:%d:continue:%s", phase.ID, spec.Caste)),
			Task:          continueReviewTaskForCaste(spec.Caste),
			TaskID:        fmt.Sprintf("continue-review-%s", spec.Caste),
			Timeout:       timeoutSeconds,
			Status:        "planned",
			Brief:         renderCodexContinueReviewBrief(root, phase, manifest, verification, assessment, spec),
			SkillSection:  assignment.Section,
			SkillCount:    assignment.SkillCount,
			ColonySkills:  assignment.ColonyCount,
			DomainSkills:  assignment.DomainCount,
			MatchedSkills: append([]string{}, assignment.MatchedNames...),
		})
	}
	return dispatches
}

func countContinueExternalWaves(dispatches []codexContinueExternalDispatch) int {
	seen := map[int]struct{}{}
	for _, dispatch := range dispatches {
		if dispatch.Wave > 0 {
			seen[dispatch.Wave] = struct{}{}
		}
	}
	return len(seen)
}

func continuePlanArtifactsPath(phaseID int, name string) string {
	return filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phaseID), name))
}
