package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
)

const (
	defaultOracleMaxIterations    = 8
	defaultOracleTargetConfidence = 85
	defaultOracleReasoningEffort  = "medium"
	defaultOracleScope            = "both"
	defaultOracleTemplate         = "custom"
	defaultOracleMaxAttempts      = 2
	defaultOracleStrategy         = "adaptive"
	defaultOracleTimeout          = 12 * time.Minute
	defaultOracleHeartbeat        = 15 * time.Second
)

type oracleDepthConfig struct {
	MaxIterations    int
	TargetConfidence int
	Label            string
	Description      string
}

var oracleDepthLevels = map[string]oracleDepthConfig{
	"quick":      {2, 60, "Quick", "Fast overview, 1-2 questions"},
	"balanced":   {4, 85, "Balanced", "Thorough investigation, 3-4 questions"},
	"deep":       {6, 95, "Deep", "Comprehensive analysis, 5-6 questions"},
	"exhaustive": {10, 99, "Exhaustive", "Full convergence, up to 10 iterations"},
}

func resolveOracleDepth(depth string) oracleDepthConfig {
	if cfg, ok := oracleDepthLevels[strings.ToLower(strings.TrimSpace(depth))]; ok {
		return cfg
	}
	return oracleDepthLevels["balanced"]
}

var newOracleWorkerInvoker = codex.NewWorkerInvoker
var oracleAttemptPolicyForPhase = defaultOracleAttemptPolicy

type oracleStateFile struct {
	Version            string   `json:"version,omitempty"`
	Topic              string   `json:"topic,omitempty"`
	Scope              string   `json:"scope,omitempty"`
	Template           string   `json:"template,omitempty"`
	Phase              string   `json:"phase,omitempty"`
	Iteration          int      `json:"iteration,omitempty"`
	MaxIterations      int      `json:"max_iterations,omitempty"`
	TargetConfidence   int      `json:"target_confidence,omitempty"`
	OverallConfidence  int      `json:"overall_confidence,omitempty"`
	StartedAt          string   `json:"started_at,omitempty"`
	LastUpdated        string   `json:"last_updated,omitempty"`
	Status             string   `json:"status,omitempty"`
	Strategy           string   `json:"strategy,omitempty"`
	FocusAreas         []string `json:"focus_areas,omitempty"`
	Platform           string   `json:"platform,omitempty"`
	StopReason         string   `json:"stop_reason,omitempty"`
	Summary            string   `json:"summary,omitempty"`
	ActiveQuestionID   string   `json:"active_question_id,omitempty"`
	ActiveQuestionText string   `json:"active_question_text,omitempty"`
	ActiveAttempt      int      `json:"active_attempt,omitempty"`
	ActiveReasoning    string   `json:"active_reasoning,omitempty"`
	ActiveTimeoutSec   int      `json:"active_timeout_sec,omitempty"`
	ActiveElapsedSec   int      `json:"active_elapsed_sec,omitempty"`
	ActiveStartedAt    string   `json:"active_started_at,omitempty"`
	ActiveDeadlineAt   string   `json:"active_deadline_at,omitempty"`
	LastArtifactPath   string   `json:"last_artifact_path,omitempty"`
	OpenGaps           []string `json:"open_gaps,omitempty"`
	Contradictions     []string `json:"contradictions,omitempty"`
	Recommendation     string   `json:"recommendation,omitempty"`
	ControllerPID      int      `json:"controller_pid,omitempty"`
	Depth              string   `json:"depth,omitempty"`
}

type oraclePlanFile struct {
	Version     string                  `json:"version,omitempty"`
	Sources     map[string]oracleSource `json:"sources"`
	Questions   []oracleQuestion        `json:"questions"`
	CreatedAt   string                  `json:"created_at,omitempty"`
	LastUpdated string                  `json:"last_updated,omitempty"`
}

type oracleSource struct {
	URL        string `json:"url"`
	Title      string `json:"title"`
	Type       string `json:"type"`
	AccessedAt string `json:"accessed_at"`
}

type oracleQuestion struct {
	ID                string          `json:"id"`
	Text              string          `json:"text"`
	Status            string          `json:"status"`
	Confidence        int             `json:"confidence"`
	KeyFindings       []oracleFinding `json:"key_findings"`
	IterationsTouched []int           `json:"iterations_touched"`
}

type oracleFinding struct {
	Text      string   `json:"text"`
	SourceIDs []string `json:"source_ids,omitempty"`
	Iteration int      `json:"iteration,omitempty"`
}

type oraclePaths struct {
	Root             string
	Dir              string
	ArchiveDir       string
	DiscoveriesDir   string
	ResponsesDir     string
	StatePath        string
	PlanPath         string
	GapsPath         string
	SynthesisPath    string
	ResearchPlanPath string
	StopPath         string
	LoopPath         string
	AgentName        string
}

type oracleProgressSnapshot struct {
	Answered   int
	Touched    int
	Findings   int
	Confidence int
}

type oracleAttemptPolicy struct {
	ReasoningEffort string
	Timeout         time.Duration
	Heartbeat       time.Duration
}

type oracleIterationArtifact struct {
	Iteration  int      `json:"iteration"`
	Attempt    int      `json:"attempt,omitempty"`
	Phase      string   `json:"phase"`
	WorkerName string   `json:"worker_name,omitempty"`
	Caste      string   `json:"caste,omitempty"`
	TaskID     string   `json:"task_id,omitempty"`
	Status     string   `json:"status,omitempty"`
	Summary    string   `json:"summary,omitempty"`
	Error      string   `json:"error,omitempty"`
	Blockers   []string `json:"blockers,omitempty"`
	Response   string   `json:"response_path,omitempty"`
	RawOutput  string   `json:"raw_output,omitempty"`
	DurationMS int64    `json:"duration_ms,omitempty"`
	Timestamp  string   `json:"timestamp"`
}

type oracleWorkerResponse struct {
	QuestionID     string                `json:"question_id"`
	Status         string                `json:"status"`
	Confidence     int                   `json:"confidence"`
	Summary        string                `json:"summary"`
	Findings       []oracleWorkerFinding `json:"findings,omitempty"`
	Gaps           []string              `json:"gaps,omitempty"`
	Contradictions []string              `json:"contradictions,omitempty"`
	Recommendation string                `json:"recommendation,omitempty"`
}

type oracleWorkerFinding struct {
	Text     string                 `json:"text"`
	Evidence []oracleWorkerEvidence `json:"evidence,omitempty"`
}

type oracleWorkerEvidence struct {
	Title    string `json:"title"`
	Location string `json:"location"`
	Type     string `json:"type"`
}

func runOracleCompatibility(root string, args []string, depth string) (map[string]interface{}, error) {
	mode := "status"
	if len(args) > 0 {
		mode = strings.ToLower(strings.TrimSpace(args[0]))
	}

	switch mode {
	case "", "status":
		return oracleStatusResult(root)
	case "stop":
		return stopOracleCompatibility(root)
	default:
		return startOracleCompatibility(root, strings.TrimSpace(strings.Join(args, " ")), depth)
	}
}

func oracleStatusResult(root string) (map[string]interface{}, error) {
	paths := oracleWorkspacePaths(root)
	state, _ := loadOracleStateFile(paths.StatePath)
	plan, _ := loadOraclePlanFile(paths.PlanPath)

	questionCount, answeredCount, touchedCount := oracleQuestionCounts(plan)
	active := strings.EqualFold(state.Status, "active") || strings.EqualFold(state.Status, "planned")
	next := "aether oracle \"research topic\""
	switch {
	case active:
		next = "aether oracle stop"
	case fileExists(paths.ResearchPlanPath):
		next = "aether oracle status"
	}

	return map[string]interface{}{
		"mode":               "status",
		"active":             active,
		"status":             emptyFallback(strings.TrimSpace(state.Status), "idle"),
		"topic":              strings.TrimSpace(state.Topic),
		"platform":           emptyFallback(strings.TrimSpace(state.Platform), "codex"),
		"phase":              emptyFallback(strings.TrimSpace(state.Phase), "idle"),
		"iteration":          state.Iteration,
		"max_iterations":     state.MaxIterations,
		"overall_confidence": state.OverallConfidence,
		"target_confidence":  state.TargetConfidence,
		"question_count":     questionCount,
		"answered_count":     answeredCount,
		"touched_count":      touchedCount,
		"focus_areas":        append([]string(nil), state.FocusAreas...),
		"active_question_id": strings.TrimSpace(state.ActiveQuestionID),
		"active_question":    strings.TrimSpace(state.ActiveQuestionText),
		"active_attempt":     state.ActiveAttempt,
		"active_reasoning":   strings.TrimSpace(state.ActiveReasoning),
		"active_timeout_sec": state.ActiveTimeoutSec,
		"active_elapsed_sec": state.ActiveElapsedSec,
		"active_started_at":  strings.TrimSpace(state.ActiveStartedAt),
		"active_deadline_at": strings.TrimSpace(state.ActiveDeadlineAt),
		"last_artifact_path": strings.TrimSpace(state.LastArtifactPath),
		"controller_pid":     state.ControllerPID,
		"stop_reason":        strings.TrimSpace(state.StopReason),
		"summary":            strings.TrimSpace(state.Summary),
		"state_path":         paths.StatePath,
		"plan_path":          paths.PlanPath,
		"synthesis_path":     paths.SynthesisPath,
		"research_plan":      paths.ResearchPlanPath,
		"has_state":          fileExists(paths.StatePath),
		"has_plan":           fileExists(paths.PlanPath),
		"has_synthesis":      fileExists(paths.SynthesisPath),
		"has_research_plan":  fileExists(paths.ResearchPlanPath),
		"next":               next,
	}, nil
}

func stopOracleCompatibility(root string) (map[string]interface{}, error) {
	paths := oracleWorkspacePaths(root)
	if err := os.MkdirAll(paths.Dir, 0755); err != nil {
		return nil, fmt.Errorf("create oracle dir: %w", err)
	}
	state, _ := loadOracleStateFile(paths.StatePath)
	if err := os.WriteFile(paths.StopPath, []byte(time.Now().UTC().Format(time.RFC3339)+"\n"), 0644); err != nil {
		return nil, fmt.Errorf("write stop marker: %w", err)
	}
	killedPIDs, killErr := terminateOracleProcessTree(state.ControllerPID)
	_ = os.Remove(paths.LoopPath)

	now := time.Now().UTC().Format(time.RFC3339)
	if strings.TrimSpace(state.StartedAt) == "" {
		state.StartedAt = now
	}
	state.Version = "1.1"
	state.Platform = "codex"
	state.Status = "stopped"
	state.StopReason = "manual_stop"
	state.LastUpdated = now
	if err := writeOracleStateFile(paths.StatePath, state); err != nil {
		return nil, err
	}

	if plan, err := loadOraclePlanFile(paths.PlanPath); err == nil {
		_ = writeOracleDerivedReports(paths, state, plan)
	}

	result, err := oracleStatusResult(root)
	if err != nil {
		return nil, err
	}
	result["mode"] = "stop"
	result["stopped"] = true
	result["stop_path"] = paths.StopPath
	result["killed_pids"] = killedPIDs
	result["next"] = "aether oracle status"
	if killErr != nil {
		result["kill_warning"] = killErr.Error()
	}
	return result, nil
}

func startOracleCompatibility(root, topic, depth string) (map[string]interface{}, error) {
	if strings.TrimSpace(topic) == "" {
		return oracleStatusResult(root)
	}

	paths := oracleWorkspacePaths(root)
	if fileExists(paths.LoopPath) {
		if state, err := loadOracleStateFile(paths.StatePath); err == nil && strings.EqualFold(strings.TrimSpace(state.Status), "active") {
			return nil, fmt.Errorf("Oracle loop already active. Run `aether oracle status` or `aether oracle stop` first.")
		}
	}
	if err := ensureOracleWorkspace(paths); err != nil {
		return nil, err
	}
	if err := archiveOracleWorkspace(paths); err != nil {
		return nil, err
	}
	if err := ensureOracleWorkspace(paths); err != nil {
		return nil, err
	}
	_ = os.Remove(paths.StopPath)
	_ = os.Remove(paths.LoopPath)

	detectedType, languages, frameworks := detectOracleProjectProfile(root)
	brief := formulateOracleBrief(root, topic, detectedType, languages, frameworks)
	depthCfg := resolveOracleDepth(depth)
	now := time.Now().UTC().Format(time.RFC3339)
	state := oracleStateFile{
		Version:           "1.1",
		Topic:             topic,
		Scope:             defaultOracleScope,
		Template:          defaultOracleTemplate,
		Phase:             "survey",
		Iteration:         0,
		MaxIterations:     depthCfg.MaxIterations,
		TargetConfidence:  depthCfg.TargetConfidence,
		OverallConfidence: 0,
		StartedAt:         now,
		LastUpdated:       now,
		Status:            "active",
		Strategy:          defaultOracleStrategy,
		FocusAreas:        currentOracleFocusAreas(),
		Platform:          "codex",
		ControllerPID:     os.Getpid(),
		Depth:             depthCfg.Label,
	}
	plan := oraclePlanFile{
		Version:     "1.1",
		Sources:     map[string]oracleSource{},
		Questions:   buildBriefInformedQuestions(topic, brief, detectedType),
		CreatedAt:   now,
		LastUpdated: now,
	}

	if err := writeOracleStateFile(paths.StatePath, state); err != nil {
		return nil, err
	}
	if err := writeOraclePlanFile(paths.PlanPath, plan); err != nil {
		return nil, err
	}
	if err := writeOracleDerivedReports(paths, state, plan); err != nil {
		return nil, err
	}
	if err := writeOracleLoopMarker(paths.LoopPath, state); err != nil {
		return nil, err
	}

	result, err := runOracleLoop(paths, detectedType, languages, frameworks)
	if err != nil {
		return nil, err
	}
	result["started"] = true
	return result, nil
}

func runOracleLoop(paths oraclePaths, detectedType string, languages, frameworks []string) (map[string]interface{}, error) {
	ctx := context.Background()
	invoker := newOracleWorkerInvoker()
	if invoker == nil {
		invoker = &codex.FakeInvoker{}
	}
	if _, ok := invoker.(*codex.FakeInvoker); !ok && !invoker.IsAvailable(ctx) {
		return nil, fmt.Errorf("oracle loop cannot start because %s", dispatchAvailabilityMessage(invoker))
	}
	agentPath := dispatchAgentPath(paths.Root, invoker, paths.AgentName)
	if err := invoker.ValidateAgent(agentPath); err != nil {
		return nil, fmt.Errorf("oracle agent unavailable: %w", err)
	}

	state, err := loadOracleStateFile(paths.StatePath)
	if err != nil {
		return nil, err
	}
	plan, err := loadOraclePlanFile(paths.PlanPath)
	if err != nil {
		return nil, err
	}

	iterationsRun := 0
	for state.Iteration < state.MaxIterations {
		if oracleStopRequested(paths.StopPath) {
			return finalizeOracleLoop(paths, state, plan, detectedType, languages, frameworks, iterationsRun, "stopped", "manual_stop", "aether oracle status")
		}

		state.Iteration++
		state.Phase = nextOraclePhase(plan, state)
		target := selectOracleQuestionSmart(plan, state)
		state.Status = "active"
		state.StopReason = ""
		state.ActiveQuestionID = strings.TrimSpace(target.ID)
		state.ActiveQuestionText = strings.TrimSpace(target.Text)
		state.ActiveAttempt = 0
		state.ActiveReasoning = ""
		state.ActiveTimeoutSec = 0
		state.ActiveElapsedSec = 0
		state.ActiveStartedAt = ""
		state.ActiveDeadlineAt = ""
		state.LastArtifactPath = ""
		state.Summary = fmt.Sprintf("Investigating %s during the %s phase.", oracleQuestionLabel(target), state.Phase)
		state.LastUpdated = time.Now().UTC().Format(time.RFC3339)
		if err := writeOracleStateFile(paths.StatePath, state); err != nil {
			return nil, err
		}
		if err := writeOracleLoopMarker(paths.LoopPath, state); err != nil {
			return nil, err
		}
		if err := writeOracleResearchPlan(paths.ResearchPlanPath, state, plan); err != nil {
			return nil, err
		}

		emitVisualProgress(renderOracleIterationPreview(state, plan))

		before := snapshotOracleProgress(plan, state)
		iterationsRun++
		var (
			result         codex.WorkerResult
			invokeErr      error
			artifactPath   string
			artifactErr    error
			responsePath   string
			workerReply    oracleWorkerResponse
			replyLoaded    bool
			loopStopReason string
		)
		for attempt := 1; attempt <= defaultOracleMaxAttempts; attempt++ {
			policy := oracleAttemptPolicyForPhase(state.Phase, attempt)
			startedAt := time.Now().UTC()
			deadlineAt := startedAt.Add(policy.Timeout)
			responsePath = oracleResponsePath(paths, state.Iteration, attempt)
			_ = os.Remove(responsePath)
			state.ActiveAttempt = attempt
			state.ActiveReasoning = policy.ReasoningEffort
			state.ActiveTimeoutSec = int(policy.Timeout.Seconds())
			state.ActiveElapsedSec = 0
			state.ActiveStartedAt = startedAt.Format(time.RFC3339)
			state.ActiveDeadlineAt = deadlineAt.Format(time.RFC3339)
			state.LastUpdated = startedAt.Format(time.RFC3339)
			state.Summary = fmt.Sprintf("Running attempt %d/%d for %s (%s reasoning, %s watchdog).", attempt, defaultOracleMaxAttempts, oracleQuestionLabel(target), policy.ReasoningEffort, oracleDurationLabel(policy.Timeout))
			if err := writeOracleStateFile(paths.StatePath, state); err != nil {
				return nil, err
			}
			if err := writeOracleResearchPlan(paths.ResearchPlanPath, state, plan); err != nil {
				return nil, err
			}

			result, invokeErr = runOracleIterationAttempt(ctx, invoker, paths, state, plan, detectedType, languages, frameworks, target, attempt, policy, responsePath)
			replyLoaded = false
			if invokeErr == nil && result.Error == nil && !strings.EqualFold(strings.TrimSpace(result.Status), "failed") {
				workerReply, invokeErr = loadOracleWorkerResponse(responsePath, target)
				if invokeErr == nil {
					replyLoaded = true
					if strings.TrimSpace(workerReply.Summary) != "" {
						result.Summary = workerReply.Summary
					}
					switch workerReply.Status {
					case "blocked":
						result.Status = "blocked"
						result.Blockers = append(result.Blockers, workerReply.Gaps...)
					case "answered", "partial":
						result.Status = "completed"
					}
					state, plan, invokeErr = applyOracleWorkerResponse(state, plan, workerReply)
					if invokeErr == nil {
						if err := writeOraclePlanFile(paths.PlanPath, plan); err != nil {
							return nil, err
						}
						if err := writeOracleStateFile(paths.StatePath, state); err != nil {
							return nil, err
						}
						if err := writeOracleDerivedReports(paths, state, plan); err != nil {
							return nil, err
						}
					}
				}
			}
			artifactPath, artifactErr = writeOracleIterationArtifact(paths, state, attempt, result, invokeErr, responsePath)
			state.LastArtifactPath = artifactPath
			if artifactErr != nil {
				return nil, artifactErr
			}
			if !oracleRetryableFailure(result, invokeErr) || attempt >= defaultOracleMaxAttempts {
				break
			}

			state.Summary = fmt.Sprintf("%s Retrying with a narrower recovery prompt.", oracleWorkerFailureSummary(result, invokeErr, artifactPath))
			state.LastUpdated = time.Now().UTC().Format(time.RFC3339)
			if err := writeOracleStateFile(paths.StatePath, state); err != nil {
				return nil, err
			}
			emitVisualProgress(renderOracleRetryPreview(state))
		}
		if invokeErr != nil {
			state.Status = "blocked"
			state.StopReason = "worker_error"
			state.Summary = oracleWorkerFailureSummary(result, invokeErr, artifactPath)
			state.LastUpdated = time.Now().UTC().Format(time.RFC3339)
			if err := writeOracleStateFile(paths.StatePath, state); err != nil {
				return nil, err
			}
			return finalizeOracleLoop(paths, state, plan, detectedType, languages, frameworks, iterationsRun, "blocked", "worker_error", "aether oracle status")
		}
		if result.Error != nil || strings.TrimSpace(result.Status) == "failed" || strings.TrimSpace(result.Status) == "blocked" {
			state.Status = "blocked"
			if result.Error != nil {
				state.StopReason = "worker_error"
			} else {
				state.StopReason = "worker_blocked"
			}
			state.Summary = oracleWorkerFailureSummary(result, nil, artifactPath)
			if replyLoaded && strings.TrimSpace(workerReply.Status) == "blocked" {
				loopStopReason = "worker_blocked"
			}
			state.LastUpdated = time.Now().UTC().Format(time.RFC3339)
			if err := writeOracleStateFile(paths.StatePath, state); err != nil {
				return nil, err
			}
			if loopStopReason == "" {
				loopStopReason = state.StopReason
			}
			return finalizeOracleLoop(paths, state, plan, detectedType, languages, frameworks, iterationsRun, "blocked", loopStopReason, "aether oracle status")
		}

		state.Iteration = iterationsRun
		state.OverallConfidence = oracleOverallConfidence(plan)
		state.Platform = "codex"
		state.ActiveAttempt = 0
		state.ActiveReasoning = ""
		state.ActiveTimeoutSec = 0
		state.ActiveElapsedSec = 0
		state.ActiveStartedAt = ""
		state.ActiveDeadlineAt = ""
		state.LastArtifactPath = artifactPath
		state.LastUpdated = time.Now().UTC().Format(time.RFC3339)
		if err := writeOracleStateFile(paths.StatePath, state); err != nil {
			return nil, err
		}
		if err := writeOracleDerivedReports(paths, state, plan); err != nil {
			return nil, err
		}

		if oracleStopRequested(paths.StopPath) {
			return finalizeOracleLoop(paths, state, plan, detectedType, languages, frameworks, iterationsRun, "stopped", "manual_stop", "aether oracle status")
		}
		if !oracleProgressedSince(before, plan, state) {
			state.Status = "blocked"
			state.StopReason = "no_progress"
			state.Summary = fmt.Sprintf("Oracle iteration finished without updating the tracked research state for %s.", oracleQuestionLabel(target))
			if err := writeOracleStateFile(paths.StatePath, state); err != nil {
				return nil, err
			}
			return finalizeOracleLoop(paths, state, plan, detectedType, languages, frameworks, iterationsRun, "blocked", "no_progress", "aether oracle status")
		}
		if oracleReadyForCompletion(plan, state) {
			state.Status = "complete"
			state.Phase = "verify"
			state.StopReason = ""
			state.Summary = fmt.Sprintf("Oracle completed at %d%% confidence after %d iterations.", state.OverallConfidence, iterationsRun)
			if err := writeOracleStateFile(paths.StatePath, state); err != nil {
				return nil, err
			}
			return finalizeOracleLoop(paths, state, plan, detectedType, languages, frameworks, iterationsRun, "complete", "", "aether oracle status")
		}
	}

	state.Status = "max_iterations_reached"
	state.Phase = "verify"
	state.StopReason = "max_iterations_reached"
	state.Summary = fmt.Sprintf("Oracle stopped at %d%% confidence after reaching the iteration cap.", state.OverallConfidence)
	if err := writeOracleStateFile(paths.StatePath, state); err != nil {
		return nil, err
	}
	return finalizeOracleLoop(paths, state, plan, detectedType, languages, frameworks, iterationsRun, "max_iterations_reached", "max_iterations_reached", "aether oracle status")
}

func finalizeOracleLoop(paths oraclePaths, state oracleStateFile, plan oraclePlanFile, detectedType string, languages, frameworks []string, iterationsRun int, status, stopReason, next string) (map[string]interface{}, error) {
	state.Status = status
	state.Platform = "codex"
	state.LastUpdated = time.Now().UTC().Format(time.RFC3339)
	if strings.TrimSpace(stopReason) != "" {
		state.StopReason = stopReason
	}
	if status != "active" {
		_ = os.Remove(paths.LoopPath)
	}
	if status == "complete" {
		_ = os.Remove(paths.StopPath)
	}
	if err := writeOracleStateFile(paths.StatePath, state); err != nil {
		return nil, err
	}
	if err := writeOracleDerivedReports(paths, state, plan); err != nil {
		return nil, err
	}

	questionCount, answeredCount, touchedCount := oracleQuestionCounts(plan)
	return map[string]interface{}{
		"mode":               "run",
		"autonomous":         true,
		"status":             status,
		"topic":              state.Topic,
		"phase":              state.Phase,
		"iteration":          state.Iteration,
		"iterations_run":     iterationsRun,
		"max_iterations":     state.MaxIterations,
		"overall_confidence": state.OverallConfidence,
		"target_confidence":  state.TargetConfidence,
		"question_count":     questionCount,
		"answered_count":     answeredCount,
		"touched_count":      touchedCount,
		"detected_type":      detectedType,
		"languages":          languages,
		"frameworks":         frameworks,
		"focus_areas":        append([]string(nil), state.FocusAreas...),
		"active_question_id": strings.TrimSpace(state.ActiveQuestionID),
		"active_question":    strings.TrimSpace(state.ActiveQuestionText),
		"active_attempt":     state.ActiveAttempt,
		"active_reasoning":   strings.TrimSpace(state.ActiveReasoning),
		"active_timeout_sec": state.ActiveTimeoutSec,
		"active_elapsed_sec": state.ActiveElapsedSec,
		"active_started_at":  strings.TrimSpace(state.ActiveStartedAt),
		"active_deadline_at": strings.TrimSpace(state.ActiveDeadlineAt),
		"last_artifact_path": strings.TrimSpace(state.LastArtifactPath),
		"state_path":         paths.StatePath,
		"plan_path":          paths.PlanPath,
		"gaps_path":          paths.GapsPath,
		"synthesis_path":     paths.SynthesisPath,
		"research_plan":      paths.ResearchPlanPath,
		"loop_path":          paths.LoopPath,
		"stop_reason":        stopReason,
		"summary":            strings.TrimSpace(state.Summary),
		"next":               next,
	}, nil
}

func runOracleIterationAttempt(ctx context.Context, invoker codex.WorkerInvoker, paths oraclePaths, state oracleStateFile, plan oraclePlanFile, detectedType string, languages, frameworks []string, target oracleQuestion, attempt int, policy oracleAttemptPolicy, responsePath string) (codex.WorkerResult, error) {
	attemptCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	type oracleAttemptResult struct {
		result codex.WorkerResult
		err    error
	}

	resultCh := make(chan oracleAttemptResult, 1)
	startedAt := time.Now()
	workerState := state
	go func() {
		result, err := invokeOracleIteration(attemptCtx, invoker, paths, workerState, plan, detectedType, languages, frameworks, target, attempt, policy, responsePath)
		resultCh <- oracleAttemptResult{result: result, err: err}
	}()

	heartbeat := policy.Heartbeat
	if heartbeat <= 0 {
		heartbeat = defaultOracleHeartbeat
	}
	ticker := time.NewTicker(heartbeat)
	defer ticker.Stop()

	for {
		select {
		case attemptResult := <-resultCh:
			return attemptResult.result, attemptResult.err
		case <-ticker.C:
			elapsed := time.Since(startedAt)
			if response, err := loadOracleWorkerResponse(responsePath, target); err == nil {
				cancel()
				return codex.WorkerResult{
					Caste:    "oracle",
					TaskID:   fmt.Sprintf("oracle.%d", state.Iteration),
					Status:   oracleResponseStatusToWorkerStatus(response.Status),
					Summary:  response.Summary,
					Blockers: append([]string(nil), response.Gaps...),
					Duration: elapsed,
				}, nil
			}
			state.ActiveElapsedSec = int(elapsed.Seconds())
			state.LastUpdated = time.Now().UTC().Format(time.RFC3339)
			state.Summary = fmt.Sprintf(
				"Running attempt %d/%d for %s (%s elapsed of %s watchdog, %s reasoning).",
				attempt,
				defaultOracleMaxAttempts,
				oracleQuestionLabel(target),
				oracleDurationLabel(elapsed),
				oracleDurationLabel(policy.Timeout),
				policy.ReasoningEffort,
			)
			if err := writeOracleStateFile(paths.StatePath, state); err != nil {
				cancel()
				return codex.WorkerResult{}, err
			}
		}
	}
}

func invokeOracleIteration(ctx context.Context, invoker codex.WorkerInvoker, paths oraclePaths, state oracleStateFile, plan oraclePlanFile, detectedType string, languages, frameworks []string, target oracleQuestion, attempt int, policy oracleAttemptPolicy, responsePath string) (codex.WorkerResult, error) {
	workerName := deterministicAntName("oracle", fmt.Sprintf("%s|%d", state.Topic, state.Iteration))
	topicHeadline := oracleTopicHeadline(state.Topic)
	if topicHeadline == "" {
		topicHeadline = "Oracle research topic"
	}
	targetLabel := oracleQuestionLabel(target)
	responseRelPath, _ := filepath.Rel(paths.Root, responsePath)
	responseRelPath = filepath.ToSlash(responseRelPath)
	brief := codex.RenderTaskBrief(codex.TaskBriefData{
		TaskID: fmt.Sprintf("oracle.%d", state.Iteration),
		Goal:   fmt.Sprintf("Advance the Oracle RALF loop for %s by investigating %s during the %s phase.", topicHeadline, targetLabel, state.Phase),
		Constraints: []string{
			fmt.Sprintf("Write exactly one Oracle response JSON file to %s.", emptyFallback(responseRelPath, responsePath)),
			"Do not read or rewrite .aether/oracle/state.json, plan.json, gaps.md, synthesis.md, or research-plan.md.",
			"Do not modify source code, tests, COLONY_STATE.json, session.json, or pheromones.json.",
			"Use integer confidence scores from 0-100.",
			fmt.Sprintf("Target only one question this iteration: %s", targetLabel),
		},
		Hints: []string{
			fmt.Sprintf("Current phase: %s", state.Phase),
			fmt.Sprintf("Iteration: %d of %d", state.Iteration, state.MaxIterations),
			fmt.Sprintf("Detected project type: %s", emptyFallback(detectedType, "unknown")),
			fmt.Sprintf("Languages: %s", renderCSV(languages, "unknown")),
			fmt.Sprintf("Frameworks: %s", renderCSV(frameworks, "none detected")),
			fmt.Sprintf("Target question text: %s", emptyFallback(strings.TrimSpace(target.Text), "select the lowest-confidence open question")),
			"Prefer direct codebase, generated-artifact, and runtime-command evidence first; use web sources only when local evidence cannot answer the target question.",
			oraclePhaseDirective(state, plan),
			"Return worker claims JSON normally, but put the actual research payload in the Oracle response file path provided above.",
		},
		SuccessCriteria: []string{
			"The response file contains one question-scoped payload with a truthful status, confidence score, findings, and evidence.",
			"Findings are source-backed with concrete file paths, commands, runtime outputs, or primary docs.",
			"Only new information is added; existing findings are not duplicated.",
		},
	})
	if attempt > 1 {
		brief = strings.TrimSpace(brief + "\n\n## Recovery Attempt\n\n- This is a retry after a failed or malformed worker pass.\n- Keep the pass narrow and question-scoped.\n- If you cannot make safe progress, write a blocked Oracle response with the exact concrete blocker instead of drifting.\n")
	}
	brief = strings.TrimSpace(brief + "\n\n## Oracle Response Contract\n\nResponse File: " + emptyFallback(responseRelPath, responsePath) + "\n\nWrite this JSON object to the response file:\n```json\n{\n  \"question_id\": \"" + target.ID + "\",\n  \"status\": \"answered | partial | blocked\",\n  \"confidence\": 0,\n  \"summary\": \"short concrete summary\",\n  \"findings\": [\n    {\n      \"text\": \"new finding\",\n      \"evidence\": [\n        {\"title\": \"what you inspected\", \"location\": \"file path, command, or URL\", \"type\": \"codebase | runtime | documentation | official | github | blog | forum | academic\"}\n      ]\n    }\n  ],\n  \"gaps\": [\"remaining unanswered point\"],\n  \"contradictions\": [\"conflicting evidence if any\"],\n  \"recommendation\": \"release recommendation or next concrete action\"\n}\n```\n- Do not write markdown to the response file.\n- `answered` and `partial` responses must contain at least one finding.\n- `blocked` responses must explain the blocker concretely in `summary` or `gaps`.\n")

	return invoker.Invoke(ctx, codex.WorkerConfig{
		AgentName:        "aether-oracle",
		AgentTOMLPath:    dispatchAgentPath(paths.Root, invoker, paths.AgentName),
		Caste:            "oracle",
		WorkerName:       workerName,
		TaskID:           fmt.Sprintf("oracle.%d", state.Iteration),
		TaskBrief:        brief,
		ContextCapsule:   renderOracleContextCapsule(state, plan, detectedType, languages, frameworks, target, attempt, responseRelPath),
		Root:             paths.Root,
		Timeout:          policy.Timeout,
		PheromoneSection: resolvePheromoneSection(),
		ConfigOverrides:  oracleWorkerConfigOverrides(policy),
		ResponsePath:     responsePath,
	})
}

func renderOracleContextCapsule(state oracleStateFile, plan oraclePlanFile, detectedType string, languages, frameworks []string, target oracleQuestion, attempt int, responseRelPath string) string {
	questionCount, answeredCount, touchedCount := oracleQuestionCounts(plan)
	var b strings.Builder
	b.WriteString("# Oracle Controller Packet\n\n")
	b.WriteString("The Go controller has already selected the target question and will merge your response into the oracle workspace.\n\n")
	fmt.Fprintf(&b, "- Topic: %s\n", oracleTopicHeadline(state.Topic))
	if topicSummary := oracleTopicSummary(state.Topic); topicSummary != "" {
		fmt.Fprintf(&b, "- Topic Summary: %s\n", topicSummary)
	}
	fmt.Fprintf(&b, "- Phase: %s\n", emptyFallback(state.Phase, "survey"))
	fmt.Fprintf(&b, "- Iteration: %d of %d\n", state.Iteration, state.MaxIterations)
	fmt.Fprintf(&b, "- Attempt: %d of %d\n", attempt, defaultOracleMaxAttempts)
	fmt.Fprintf(&b, "- Target Confidence: %d%%\n", state.TargetConfidence)
	fmt.Fprintf(&b, "- Current Confidence: %d%%\n", state.OverallConfidence)
	fmt.Fprintf(&b, "- Project Type: %s\n", emptyFallback(detectedType, "unknown"))
	fmt.Fprintf(&b, "- Languages: %s\n", renderCSV(languages, "unknown"))
	fmt.Fprintf(&b, "- Frameworks: %s\n", renderCSV(frameworks, "none detected"))
	fmt.Fprintf(&b, "- Questions: %d total, %d answered, %d touched\n", questionCount, answeredCount, touchedCount)
	fmt.Fprintf(&b, "- Response File: %s\n", emptyFallback(responseRelPath, "(controller-provided path missing)"))
	if strings.TrimSpace(target.ID) != "" || strings.TrimSpace(target.Text) != "" {
		fmt.Fprintf(&b, "- Active Question: %s\n", oracleQuestionLabel(target))
	}
	if len(state.FocusAreas) > 0 {
		fmt.Fprintf(&b, "- Focus Areas: %s\n", renderCSV(compactOracleFocusAreas(state.FocusAreas), "none"))
	}
	b.WriteString("\n## Prior Findings For This Question\n")
	if prior := renderOraclePriorFindings(plan, target); prior != "" {
		b.WriteString(prior)
	} else {
		b.WriteString("- None yet.\n")
	}
	b.WriteString("\n## Current Gaps / Contradictions\n")
	if gaps := renderOracleCurrentGaps(state); gaps != "" {
		b.WriteString(gaps)
	} else {
		b.WriteString("- None recorded yet.\n")
	}
	b.WriteString("\n## Working Rules\n")
	b.WriteString("- Use the packet above instead of rereading the entire oracle workspace.\n")
	b.WriteString("- Inspect only the specific code, docs, tests, commands, or runtime outputs needed to answer the target question.\n")
	b.WriteString("- Prefer a concrete blocker over a vague summary.\n")
	return strings.TrimSpace(b.String())
}

func oraclePhaseDirective(state oracleStateFile, plan oraclePlanFile) string {
	switch strings.ToLower(strings.TrimSpace(state.Phase)) {
	case "survey":
		return "Survey pass: prioritize untouched questions first and establish the evidence map."
	case "verify":
		return "Verify pass: resolve contradictions, tighten confidence scores, and sharpen the release recommendation."
	default:
		return "Investigate pass: deepen the lowest-confidence unresolved question with new source-backed findings."
	}
}

func oracleWorkspacePaths(root string) oraclePaths {
	dir := filepath.Join(root, ".aether", "oracle")
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
		StopPath:         filepath.Join(dir, ".stop"),
		LoopPath:         filepath.Join(dir, ".loop-active"),
		AgentName:        "aether-oracle",
	}
}

func ensureOracleWorkspace(paths oraclePaths) error {
	if err := os.MkdirAll(paths.ArchiveDir, 0755); err != nil {
		return fmt.Errorf("create oracle archive dir: %w", err)
	}
	if err := os.MkdirAll(paths.DiscoveriesDir, 0755); err != nil {
		return fmt.Errorf("create oracle discoveries dir: %w", err)
	}
	if err := os.MkdirAll(paths.ResponsesDir, 0755); err != nil {
		return fmt.Errorf("create oracle responses dir: %w", err)
	}
	return nil
}

func archiveOracleWorkspace(paths oraclePaths) error {
	entries, err := os.ReadDir(paths.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read oracle workspace: %w", err)
	}

	var archiveNames []string
	for _, entry := range entries {
		name := entry.Name()
		if name == "archive" {
			continue
		}
		archiveNames = append(archiveNames, name)
	}
	if len(archiveNames) == 0 {
		return nil
	}

	destDir := filepath.Join(paths.ArchiveDir, time.Now().UTC().Format("2006-01-02-150405"))
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("create oracle archive snapshot: %w", err)
	}
	for _, name := range archiveNames {
		src := filepath.Join(paths.Dir, name)
		dst := filepath.Join(destDir, name)
		if err := copyOracleWorkspaceEntry(src, dst); err != nil {
			return err
		}
		if err := os.RemoveAll(src); err != nil {
			return fmt.Errorf("clear oracle workspace entry %s: %w", src, err)
		}
	}
	return nil
}

func copyOracleWorkspaceEntry(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat oracle workspace entry %s: %w", src, err)
	}
	if info.IsDir() {
		if err := os.MkdirAll(dst, info.Mode().Perm()); err != nil {
			return fmt.Errorf("create oracle archive dir %s: %w", dst, err)
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return fmt.Errorf("read oracle dir %s: %w", src, err)
		}
		for _, entry := range entries {
			if err := copyOracleWorkspaceEntry(filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name())); err != nil {
				return err
			}
		}
		return nil
	}

	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read oracle file %s: %w", src, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("create oracle archive parent %s: %w", filepath.Dir(dst), err)
	}
	if err := os.WriteFile(dst, data, info.Mode().Perm()); err != nil {
		return fmt.Errorf("write oracle archive file %s: %w", dst, err)
	}
	return nil
}

func detectOracleProjectProfile(root string) (string, []string, []string) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return "unknown", nil, nil
	}

	entryNames := make(map[string]bool, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		entryNames[entry.Name()] = true
	}

	detected := "unknown"
	seenLanguages := map[string]bool{}
	seenFrameworks := map[string]bool{}
	languages := []string{}
	frameworks := []string{}

	for _, det := range projectDetectors {
		if !entryNames[det.file] {
			continue
		}
		if detected == "unknown" {
			detected = det.typ
		}
		if !seenLanguages[det.typ] {
			seenLanguages[det.typ] = true
			languages = append(languages, det.typ)
		}
		for _, framework := range det.frameworks {
			if seenFrameworks[framework] {
				continue
			}
			seenFrameworks[framework] = true
			frameworks = append(frameworks, framework)
		}
	}

	sort.Strings(languages)
	sort.Strings(frameworks)
	return detected, languages, frameworks
}

func buildOracleQuestions(topic, detectedType string) []oracleQuestion {
	lower := strings.ToLower(topic)
	var prompts []string
	switch {
	case containsAnyOracleKeyword(lower, "release", "parity", "audit", "readiness", "ready", "codex", "claude", "opencode", "pheromone", "lifecycle"):
		prompts = []string{
			"What concrete runtime behavior currently diverges from the expected cross-platform colony lifecycle?",
			"Which files, commands, agents, or packaged assets are responsible for the current parity gaps?",
			"How do pheromone signals, colony context, and recovery state behave in live command flows across platforms?",
			"What documentation or release-surface claims are contradicted by the current implementation or generated assets?",
			"What specific fixes and verification evidence are still required before this is honestly release-ready?",
		}
	case containsAnyOracleKeyword(lower, "bug", "failure", "regression", "issue", "error"):
		prompts = []string{
			"What is the exact failure behavior and where does it surface?",
			"What are the reproduction conditions and affected code paths?",
			"What is the most defensible root cause based on current evidence?",
			"What fixes are available and what tradeoffs do they carry?",
			"What regression risks or adjacent failures need verification?",
		}
	default:
		prompts = []string{
			fmt.Sprintf("What is the actual problem boundary for %s?", topic),
			"Which files, commands, or runtime surfaces matter most to this investigation?",
			"What evidence currently supports or contradicts the expected behavior?",
			"What changes would materially improve reliability or clarity?",
			"What release or operational recommendation follows from the evidence?",
		}
	}

	questions := make([]oracleQuestion, 0, len(prompts))
	for i, prompt := range prompts {
		questions = append(questions, oracleQuestion{
			ID:                fmt.Sprintf("q%d", i+1),
			Text:              prompt,
			Status:            "open",
			Confidence:        0,
			KeyFindings:       []oracleFinding{},
			IterationsTouched: []int{},
		})
	}
	return questions
}

func containsAnyOracleKeyword(text string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func currentOracleFocusAreas() []string {
	pf := loadPheromones()
	if pf == nil {
		return nil
	}
	now := time.Now()
	seen := map[string]bool{}
	var focus []string
	for _, sig := range pf.Signals {
		if sig.Type != "FOCUS" || !signalActiveForPrompt(sig, now) {
			continue
		}
		text := extractSignalText(sig.Content)
		if text == "" || seen[text] {
			continue
		}
		seen[text] = true
		focus = append(focus, text)
	}
	sort.Strings(focus)
	return focus
}

func currentOracleRedirectAreas() []string {
	pf := loadPheromones()
	if pf == nil {
		return nil
	}
	now := time.Now()
	seen := map[string]bool{}
	var redirects []string
	for _, sig := range pf.Signals {
		if sig.Type != "REDIRECT" || !signalActiveForPrompt(sig, now) {
			continue
		}
		text := extractSignalText(sig.Content)
		if text == "" || seen[text] {
			continue
		}
		seen[text] = true
		redirects = append(redirects, text)
	}
	sort.Strings(redirects)
	return redirects
}

func loadColonyLearnings(root string) []string {
	statePath := filepath.Join(root, ".aether", "data", "COLONY_STATE.json")
	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil
	}
	var state struct {
		Memory struct {
			Instincts []struct {
				Trigger string `json:"trigger"`
				Action  string `json:"action"`
			} `json:"instincts"`
		} `json:"memory"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return nil
	}
	instincts := state.Memory.Instincts
	if len(instincts) == 0 {
		return nil
	}
	// Take last 5 (most recent), reversed so most recent is first
	start := len(instincts) - 5
	if start < 0 {
		start = 0
	}
	recent := instincts[start:]
	learnings := make([]string, 0, len(recent))
	for i := len(recent) - 1; i >= 0; i-- {
		inst := recent[i]
		text := inst.Trigger
		if inst.Action != "" {
			text = inst.Trigger + " -> " + inst.Action
		}
		learnings = append(learnings, text)
	}
	return learnings
}

func formulateOracleBrief(root, topic, detectedType string, languages, frameworks []string) string {
	var b strings.Builder

	b.WriteString("# Oracle Research Brief\n\n")

	// Topic section
	fmt.Fprintf(&b, "## Topic\n%s\n\n", topic)

	// Project Profile section
	b.WriteString("## Project Profile\n")
	fmt.Fprintf(&b, "- Type: %s\n", detectedType)
	fmt.Fprintf(&b, "- Languages: %s\n", renderCSV(languages, "none detected"))
	fmt.Fprintf(&b, "- Frameworks: %s\n", renderCSV(frameworks, "none detected"))
	b.WriteString("\n")

	// Colony Goal section
	b.WriteString("## Colony Goal\n")
	colonyGoal := loadColonyGoal(root)
	if colonyGoal != "" {
		fmt.Fprintf(&b, "%s\n", colonyGoal)
	} else {
		b.WriteString("(no colony goal set)\n")
	}
	b.WriteString("\n")

	// Codebase Structure section
	b.WriteString("## Codebase Structure\n")
	structure := scanCodebaseStructure(root)
	if structure != "" {
		fmt.Fprintf(&b, "%s\n", structure)
	} else {
		b.WriteString("(unable to scan)\n")
	}
	b.WriteString("\n")

	// Active Signals section
	b.WriteString("## Active Signals\n")
	focusAreas := currentOracleFocusAreas()
	redirectAreas := currentOracleRedirectAreas()
	if len(focusAreas) == 0 && len(redirectAreas) == 0 {
		b.WriteString("(none)\n")
	} else {
		for _, f := range focusAreas {
			fmt.Fprintf(&b, "- FOCUS: %s\n", f)
		}
		for _, r := range redirectAreas {
			fmt.Fprintf(&b, "- REDIRECT: %s\n", r)
		}
	}
	b.WriteString("\n")

	// Recent Learnings section
	b.WriteString("## Recent Learnings\n")
	learnings := loadColonyLearnings(root)
	if len(learnings) == 0 {
		b.WriteString("(none)\n")
	} else {
		for _, l := range learnings {
			fmt.Fprintf(&b, "- %s\n", l)
		}
	}

	brief := strings.TrimSpace(b.String())

	// Write brief.md to oracle workspace
	paths := oracleWorkspacePaths(root)
	if err := os.MkdirAll(paths.Dir, 0755); err == nil {
		_ = os.WriteFile(filepath.Join(paths.Dir, "brief.md"), []byte(brief+"\n"), 0644)
	}

	return brief
}

func loadColonyGoal(root string) string {
	statePath := filepath.Join(root, ".aether", "data", "COLONY_STATE.json")
	data, err := os.ReadFile(statePath)
	if err != nil {
		return ""
	}
	var state struct {
		Goal *string `json:"goal"`
	}
	if err := json.Unmarshal(data, &state); err != nil || state.Goal == nil {
		return ""
	}
	return strings.TrimSpace(*state.Goal)
}

func scanCodebaseStructure(root string) string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return ""
	}
	var dirs []string
	var files []string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
			continue
		}
		if entry.IsDir() {
			dirs = append(dirs, name+"/")
		} else {
			files = append(files, name)
		}
	}
	sort.Strings(dirs)
	sort.Strings(files)

	var b strings.Builder
	// Show top-level dirs
	for _, d := range dirs {
		fmt.Fprintf(&b, "- %s\n", d)
		// One level deeper for key directories
		subEntries, err := os.ReadDir(filepath.Join(root, d))
		if err != nil {
			continue
		}
		subDirs := 0
		for _, sub := range subEntries {
			if sub.IsDir() && subDirs < 5 {
				fmt.Fprintf(&b, "  - %s/\n", sub.Name())
				subDirs++
			}
		}
	}
	// Show top-level files (limit to 5)
	count := 0
	for _, f := range files {
		if count >= 5 {
			break
		}
		fmt.Fprintf(&b, "- %s\n", f)
		count++
	}
	return strings.TrimSpace(b.String())
}

func buildBriefInformedQuestions(topic string, brief string, detectedType string) []oracleQuestion {
	var questions []oracleQuestion
	qID := 0

	nextQ := func(text string) {
		qID++
		questions = append(questions, oracleQuestion{
			ID:                fmt.Sprintf("q%d", qID),
			Text:              text,
			Status:            "open",
			Confidence:        0,
			KeyFindings:       []oracleFinding{},
			IterationsTouched: []int{},
		})
	}

	// Always include 2 context-establishing questions
	nextQ(fmt.Sprintf("What is the actual problem boundary for %s in the context of %s?", topic, detectedType))

	// Extract key directories from brief for file relevance question
	keyDirs := extractBriefSection(brief, "Codebase Structure")
	if keyDirs != "" {
		nextQ(fmt.Sprintf("Which specific files and modules in the following structure are most relevant to %s?\n%s", topic, keyDirs))
	} else {
		nextQ(fmt.Sprintf("Which files, commands, or runtime surfaces matter most to investigating %s?", topic))
	}

	// If signals exist, add a constraints question
	signals := extractBriefSection(brief, "Active Signals")
	if signals != "" && !strings.Contains(signals, "(none)") {
		nextQ(fmt.Sprintf("How do the following constraints affect the approach to %s?\n%s", topic, signals))
	}

	// If learnings exist, add a learning-relevance question
	learnings := extractBriefSection(brief, "Recent Learnings")
	if learnings != "" && !strings.Contains(learnings, "(none)") {
		// Take first learning for question text
		lines := strings.Split(learnings, "\n")
		learningSummary := ""
		if len(lines) > 0 {
			learningSummary = strings.TrimPrefix(lines[0], "- ")
		}
		if learningSummary != "" {
			nextQ(fmt.Sprintf("Given that %s was previously observed, how does %s relate?", learningSummary, topic))
		}
	}

	// If frameworks detected, add a framework-specific question
	frameworks := extractBriefField(brief, "Frameworks")
	if frameworks != "" && frameworks != "none detected" {
		nextQ(fmt.Sprintf("What %s-specific patterns or pitfalls apply to %s?", frameworks, topic))
	}

	// Colony goal relevance question
	goal := extractBriefSection(brief, "Colony Goal")
	if goal != "" && !strings.Contains(goal, "(no colony goal set)") {
		nextQ(fmt.Sprintf("How does investigating %s advance the colony goal of: %s?", topic, goal))
	}

  // Always include evidence and improvement questions
  nextQ(fmt.Sprintf("What evidence currently supports or contradicts the expected behavior of %s?", topic))
  nextQ(fmt.Sprintf("What changes would materially improve reliability or clarity for %s?", topic))

  // Ensure minimum of 5 questions by adding recommendation if needed
  if len(questions) < 5 {
    nextQ(fmt.Sprintf("What concrete, actionable recommendation follows from investigating %s?", topic))
  }

	return questions
}

// extractBriefSection returns the content after a "## Section Name" header until the next "## " or end of string.
func extractBriefSection(brief, sectionName string) string {
	header := "## " + sectionName
	idx := strings.Index(brief, header)
	if idx == -1 {
		return ""
	}
	content := brief[idx+len(header):]
	// Find next section header
	nextSection := strings.Index(content, "\n## ")
	if nextSection != -1 {
		content = content[:nextSection]
	}
	return strings.TrimSpace(content)
}

// extractBriefField returns the value after "- Field: " in a brief section.
func extractBriefField(brief, fieldName string) string {
	pattern := "- " + fieldName + ": "
	idx := strings.Index(brief, pattern)
	if idx == -1 {
		return ""
	}
	content := brief[idx+len(pattern):]
	// Take until end of line
	if nl := strings.Index(content, "\n"); nl != -1 {
		content = content[:nl]
	}
	return strings.TrimSpace(content)
}

func oracleResponsePath(paths oraclePaths, iteration, attempt int) string {
	return filepath.Join(paths.ResponsesDir, fmt.Sprintf("iteration-%02d-attempt-%d.json", iteration, attempt))
}

func loadOracleWorkerResponse(path string, target oracleQuestion) (oracleWorkerResponse, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return oracleWorkerResponse{}, fmt.Errorf("read oracle response: %w", err)
	}
	var response oracleWorkerResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return oracleWorkerResponse{}, fmt.Errorf("parse oracle response: %w", err)
	}
	return normalizeOracleWorkerResponse(response, target)
}

func normalizeOracleWorkerResponse(response oracleWorkerResponse, target oracleQuestion) (oracleWorkerResponse, error) {
	if strings.TrimSpace(response.QuestionID) == "" {
		response.QuestionID = target.ID
	}
	if target.ID != "" && response.QuestionID != target.ID {
		return oracleWorkerResponse{}, fmt.Errorf("oracle response targeted %q, expected %q", response.QuestionID, target.ID)
	}

	switch status := strings.ToLower(strings.TrimSpace(response.Status)); status {
	case "answered", "partial", "blocked":
		response.Status = status
	default:
		return oracleWorkerResponse{}, fmt.Errorf("oracle response has invalid status %q", response.Status)
	}

	response.Confidence = clampOracleConfidence(response.Confidence)
	response.Summary = strings.TrimSpace(response.Summary)
	response.Gaps = compactOracleStrings(response.Gaps)
	response.Contradictions = compactOracleStrings(response.Contradictions)
	response.Recommendation = strings.TrimSpace(response.Recommendation)

	findings := make([]oracleWorkerFinding, 0, len(response.Findings))
	for _, finding := range response.Findings {
		finding.Text = strings.TrimSpace(finding.Text)
		if finding.Text == "" {
			continue
		}
		evidence := make([]oracleWorkerEvidence, 0, len(finding.Evidence))
		for _, item := range finding.Evidence {
			item.Title = strings.TrimSpace(item.Title)
			item.Location = strings.TrimSpace(item.Location)
			item.Type = strings.TrimSpace(strings.ToLower(item.Type))
			if item.Type == "" {
				item.Type = inferOracleEvidenceType(item.Location)
			}
			if item.Title == "" && item.Location != "" {
				item.Title = item.Location
			}
			if item.Location == "" && item.Title == "" {
				continue
			}
			evidence = append(evidence, item)
		}
		finding.Evidence = evidence
		findings = append(findings, finding)
	}
	response.Findings = findings

	if response.Summary == "" {
		switch response.Status {
		case "blocked":
			response.Summary = fmt.Sprintf("Oracle worker blocked while investigating %s.", oracleQuestionLabel(target))
		case "partial":
			response.Summary = fmt.Sprintf("Oracle worker made partial progress on %s.", oracleQuestionLabel(target))
		default:
			response.Summary = fmt.Sprintf("Oracle worker answered %s.", oracleQuestionLabel(target))
		}
	}

	switch response.Status {
	case "answered", "partial":
		if len(response.Findings) == 0 {
			return oracleWorkerResponse{}, fmt.Errorf("oracle response for %s returned no findings", oracleQuestionLabel(target))
		}
	case "blocked":
		if len(response.Findings) == 0 && len(response.Gaps) == 0 && response.Summary == "" {
			return oracleWorkerResponse{}, fmt.Errorf("oracle blocked response for %s omitted blocker detail", oracleQuestionLabel(target))
		}
	}

	return response, nil
}

func applyOracleWorkerResponse(state oracleStateFile, plan oraclePlanFile, response oracleWorkerResponse) (oracleStateFile, oraclePlanFile, error) {
	idx := -1
	for i := range plan.Questions {
		if plan.Questions[i].ID == response.QuestionID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return state, plan, fmt.Errorf("oracle response references unknown question %q", response.QuestionID)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	question := plan.Questions[idx]
	if !containsOracleIteration(question.IterationsTouched, state.Iteration) {
		question.IterationsTouched = append(question.IterationsTouched, state.Iteration)
	}

	switch response.Status {
	case "answered", "partial":
		question.Status = response.Status
	case "blocked":
		if len(response.Findings) > 0 {
			question.Status = "partial"
		} else if strings.TrimSpace(question.Status) == "" {
			question.Status = "open"
		}
	}
	question.Confidence = clampOracleConfidence(response.Confidence)

	for _, finding := range response.Findings {
		sourceIDs := make([]string, 0, len(finding.Evidence))
		for _, evidence := range finding.Evidence {
			sourceIDs = append(sourceIDs, ensureOracleSource(&plan, evidence, now))
		}
		question.KeyFindings = appendOracleFinding(question.KeyFindings, oracleFinding{
			Text:      finding.Text,
			SourceIDs: compactOracleStrings(sourceIDs),
			Iteration: state.Iteration,
		})
	}

	plan.Questions[idx] = question
	plan.LastUpdated = now

	state.OpenGaps = mergeOracleNotes(state.OpenGaps, response.Gaps)
	state.Contradictions = mergeOracleNotes(state.Contradictions, response.Contradictions)
	if strings.TrimSpace(response.Recommendation) != "" {
		state.Recommendation = response.Recommendation
	}
	state.OverallConfidence = oracleOverallConfidence(plan)
	state.Summary = response.Summary
	state.LastUpdated = now
	return state, plan, nil
}

func ensureOracleSource(plan *oraclePlanFile, evidence oracleWorkerEvidence, accessedAt string) string {
	location := strings.TrimSpace(evidence.Location)
	title := strings.TrimSpace(evidence.Title)
	typ := strings.TrimSpace(evidence.Type)
	if typ == "" {
		typ = inferOracleEvidenceType(location)
	}
	for id, existing := range plan.Sources {
		if strings.TrimSpace(existing.URL) == location && location != "" {
			return id
		}
	}
	nextID := nextOracleSourceID(*plan)
	plan.Sources[nextID] = oracleSource{
		URL:        emptyFallback(location, title),
		Title:      emptyFallback(title, location),
		Type:       emptyFallback(typ, "codebase"),
		AccessedAt: accessedAt,
	}
	return nextID
}

func nextOracleSourceID(plan oraclePlanFile) string {
	maxID := 0
	for id := range plan.Sources {
		if strings.HasPrefix(id, "S") {
			var n int
			if _, err := fmt.Sscanf(id, "S%d", &n); err == nil && n > maxID {
				maxID = n
			}
		}
	}
	return fmt.Sprintf("S%d", maxID+1)
}

func appendOracleFinding(existing []oracleFinding, finding oracleFinding) []oracleFinding {
	text := strings.TrimSpace(finding.Text)
	if text == "" {
		return existing
	}
	for _, item := range existing {
		if strings.EqualFold(strings.TrimSpace(item.Text), text) {
			return existing
		}
	}
	return append(existing, finding)
}

func mergeOracleNotes(existing, incoming []string) []string {
	return compactOracleStrings(append(append([]string(nil), existing...), incoming...))
}

func inferOracleEvidenceType(location string) string {
	location = strings.TrimSpace(strings.ToLower(location))
	switch {
	case strings.HasPrefix(location, "http://"), strings.HasPrefix(location, "https://"):
		return "official"
	case strings.Contains(location, "go test"), strings.Contains(location, "aether "), strings.Contains(location, "codex "):
		return "runtime"
	case location == "":
		return "codebase"
	default:
		return "codebase"
	}
}

func clampOracleConfidence(confidence int) int {
	switch {
	case confidence < 0:
		return 0
	case confidence > 100:
		return 100
	default:
		return confidence
	}
}

func compactOracleStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func oracleResponseStatusToWorkerStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "blocked":
		return "blocked"
	default:
		return "completed"
	}
}

func writeOracleDerivedReports(paths oraclePaths, state oracleStateFile, plan oraclePlanFile) error {
	if err := writeOracleGapsReport(paths.GapsPath, state, plan); err != nil {
		return err
	}
	if err := writeOracleSynthesisReport(paths.SynthesisPath, state, plan); err != nil {
		return err
	}
	return writeOracleResearchPlan(paths.ResearchPlanPath, state, plan)
}

func writeOracleStateFile(path string, state oracleStateFile) error {
	if strings.TrimSpace(state.Version) == "" {
		state.Version = "1.1"
	}
	if strings.TrimSpace(state.Platform) == "" {
		state.Platform = "codex"
	}
	if strings.TrimSpace(state.LastUpdated) == "" {
		state.LastUpdated = time.Now().UTC().Format(time.RFC3339)
	}
	encoded, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal oracle state: %w", err)
	}
	return os.WriteFile(path, append(encoded, '\n'), 0644)
}

func loadOracleStateFile(path string) (oracleStateFile, error) {
	var state oracleStateFile
	data, err := os.ReadFile(path)
	if err != nil {
		return oracleStateFile{}, fmt.Errorf("read oracle state: %w", err)
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return oracleStateFile{}, fmt.Errorf("parse oracle state: %w", err)
	}
	if state.MaxIterations <= 0 {
		state.MaxIterations = defaultOracleMaxIterations
	}
	if state.TargetConfidence <= 0 {
		state.TargetConfidence = defaultOracleTargetConfidence
	}
	if strings.TrimSpace(state.Scope) == "" {
		state.Scope = defaultOracleScope
	}
	if strings.TrimSpace(state.Template) == "" {
		state.Template = defaultOracleTemplate
	}
	if strings.TrimSpace(state.Strategy) == "" {
		state.Strategy = defaultOracleStrategy
	}
	if strings.TrimSpace(state.Platform) == "" {
		state.Platform = "codex"
	}
	return state, nil
}

func writeOraclePlanFile(path string, plan oraclePlanFile) error {
	if plan.Sources == nil {
		plan.Sources = map[string]oracleSource{}
	}
	encoded, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal oracle plan: %w", err)
	}
	return os.WriteFile(path, append(encoded, '\n'), 0644)
}

func loadOraclePlanFile(path string) (oraclePlanFile, error) {
	var plan oraclePlanFile
	data, err := os.ReadFile(path)
	if err != nil {
		return oraclePlanFile{}, fmt.Errorf("read oracle plan: %w", err)
	}
	if err := json.Unmarshal(data, &plan); err != nil {
		return oraclePlanFile{}, fmt.Errorf("parse oracle plan: %w", err)
	}
	if plan.Sources == nil {
		plan.Sources = map[string]oracleSource{}
	}
	for i := range plan.Questions {
		if plan.Questions[i].KeyFindings == nil {
			plan.Questions[i].KeyFindings = []oracleFinding{}
		}
		if plan.Questions[i].IterationsTouched == nil {
			plan.Questions[i].IterationsTouched = []int{}
		}
	}
	return plan, nil
}

func writeOracleGapsReport(path string, state oracleStateFile, plan oraclePlanFile) error {
	var b strings.Builder
	b.WriteString("# Knowledge Gaps\n\n")
	b.WriteString("## Open Questions\n")
	openCount := 0
	for _, q := range plan.Questions {
		status := strings.ToLower(strings.TrimSpace(q.Status))
		if status == "answered" {
			continue
		}
		openCount++
		fmt.Fprintf(&b, "- %s (status: %s, confidence: %d%%)\n", oracleQuestionLabel(q), emptyFallback(q.Status, "open"), q.Confidence)
	}
	if openCount == 0 {
		b.WriteString("- None.\n")
	}

	b.WriteString("\n## Active Gaps\n")
	if len(state.OpenGaps) == 0 {
		b.WriteString("- None recorded.\n")
	} else {
		for _, gap := range state.OpenGaps {
			fmt.Fprintf(&b, "- %s\n", gap)
		}
	}

	b.WriteString("\n## Contradictions\n")
	if len(state.Contradictions) == 0 {
		b.WriteString("- None identified.\n")
	} else {
		for _, contradiction := range state.Contradictions {
			fmt.Fprintf(&b, "- %s\n", contradiction)
		}
	}

	fmt.Fprintf(&b, "\n## Last Updated\nIteration %d -- %s\n", state.Iteration, emptyFallback(state.LastUpdated, time.Now().UTC().Format(time.RFC3339)))
	return os.WriteFile(path, []byte(strings.TrimSpace(b.String())+"\n"), 0644)
}

func writeOracleSynthesisReport(path string, state oracleStateFile, plan oraclePlanFile) error {
	var b strings.Builder
	b.WriteString("# Research Synthesis\n\n")
	b.WriteString("## Topic\n")
	b.WriteString(emptyFallback(oracleTopicHeadline(state.Topic), "Oracle research topic"))
	if topicSummary := oracleTopicSummary(state.Topic); topicSummary != "" {
		b.WriteString("\n\n## Topic Summary\n")
		b.WriteString(topicSummary)
	}
	if strings.TrimSpace(state.Recommendation) != "" {
		b.WriteString("\n\n## Current Recommendation\n")
		b.WriteString(state.Recommendation)
	}

	b.WriteString("\n\n## Findings by Question\n")
	for _, q := range plan.Questions {
		fmt.Fprintf(&b, "\n### %s (status: %s, confidence: %d%%)\n", oracleQuestionLabel(q), emptyFallback(q.Status, "open"), q.Confidence)
		if len(q.KeyFindings) == 0 {
			b.WriteString("- No findings yet.\n")
			continue
		}
		for _, finding := range q.KeyFindings {
			b.WriteString("- ")
			b.WriteString(finding.Text)
			if len(finding.SourceIDs) > 0 {
				b.WriteString(" ")
				for i, sourceID := range finding.SourceIDs {
					if i > 0 {
						b.WriteString(" ")
					}
					fmt.Fprintf(&b, "[%s]", sourceID)
				}
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n## Sources\n")
	if len(plan.Sources) == 0 {
		b.WriteString("- None yet.\n")
	} else {
		sourceIDs := make([]string, 0, len(plan.Sources))
		for id := range plan.Sources {
			sourceIDs = append(sourceIDs, id)
		}
		sort.Strings(sourceIDs)
		for _, id := range sourceIDs {
			source := plan.Sources[id]
			fmt.Fprintf(&b, "- [%s] %s — %s (%s, accessed %s)\n", id, emptyFallback(source.Title, source.URL), source.URL, emptyFallback(source.Type, "codebase"), emptyFallback(source.AccessedAt, "unknown"))
		}
	}

	fmt.Fprintf(&b, "\n## Last Updated\nIteration %d -- %s\n", state.Iteration, emptyFallback(state.LastUpdated, time.Now().UTC().Format(time.RFC3339)))
	return os.WriteFile(path, []byte(strings.TrimSpace(b.String())+"\n"), 0644)
}

func writeOracleResearchPlan(path string, state oracleStateFile, plan oraclePlanFile) error {
	nextQuestion := "(all questions answered)"
	for _, q := range plan.Questions {
		status := strings.ToLower(strings.TrimSpace(q.Status))
		if status != "answered" {
			nextQuestion = fmt.Sprintf("%s — %s", q.ID, q.Text)
			break
		}
	}

	var b strings.Builder
	b.WriteString("# Research Plan\n\n")
	fmt.Fprintf(&b, "**Topic:** %s\n", emptyFallback(oracleTopicHeadline(state.Topic), "Oracle research topic"))
	if topicSummary := oracleTopicSummary(state.Topic); topicSummary != "" {
		fmt.Fprintf(&b, "**Topic Summary:** %s\n", topicSummary)
	}
	fmt.Fprintf(&b, "**Status:** %s | **Phase:** %s | **Iteration:** %d of %d\n", emptyFallback(state.Status, "active"), emptyFallback(state.Phase, "survey"), state.Iteration, state.MaxIterations)
	fmt.Fprintf(&b, "**Overall Confidence:** %d%% (target %d%%)\n", state.OverallConfidence, state.TargetConfidence)
	if strings.TrimSpace(state.ActiveQuestionID) != "" || strings.TrimSpace(state.ActiveQuestionText) != "" {
		fmt.Fprintf(&b, "**Active Question:** %s\n", oracleQuestionLabel(oracleQuestion{ID: state.ActiveQuestionID, Text: state.ActiveQuestionText}))
	}
	if len(state.FocusAreas) > 0 {
		fmt.Fprintf(&b, "**Focus Areas:** %s\n", strings.Join(compactOracleFocusAreas(state.FocusAreas), ", "))
	}
	b.WriteString("\n## Questions\n")
	b.WriteString("| # | Question | Status | Confidence |\n")
	b.WriteString("|---|----------|--------|------------|\n")
	for _, q := range plan.Questions {
		fmt.Fprintf(&b, "| %s | %s | %s | %d%% |\n", q.ID, escapeOracleTableCell(q.Text), emptyFallback(q.Status, "open"), q.Confidence)
	}
	b.WriteString("\n## Next Steps\n")
	b.WriteString("Next investigation: ")
	b.WriteString(nextQuestion)
	b.WriteString("\n\n---\n*Generated from plan.json -- do not edit directly*\n")
	return os.WriteFile(path, []byte(strings.TrimSpace(b.String())+"\n"), 0644)
}

func escapeOracleTableCell(text string) string {
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "\n", " ")
	return strings.ReplaceAll(text, "|", "\\|")
}

func writeOracleLoopMarker(path string, state oracleStateFile) error {
	marker := strings.TrimSpace(fmt.Sprintf(`---
iteration: %d
max_iterations: %d
phase: %s
target_confidence: %d
controller_pid: %d
oracle_md_path: .aether/utils/oracle/oracle.md
---
Oracle research loop active
`, state.Iteration, state.MaxIterations, emptyFallback(state.Phase, "survey"), state.TargetConfidence, state.ControllerPID)) + "\n"
	return os.WriteFile(path, []byte(marker), 0644)
}

func nextOraclePhase(plan oraclePlanFile, state oracleStateFile) string {
	for _, q := range plan.Questions {
		if len(q.IterationsTouched) == 0 {
			return "survey"
		}
	}
	if oracleReadyForCompletion(plan, state) || state.Iteration >= state.MaxIterations {
		return "verify"
	}
	return "investigate"
}

func snapshotOracleProgress(plan oraclePlanFile, state oracleStateFile) oracleProgressSnapshot {
	answered, touched := 0, 0
	findings := 0
	for _, q := range plan.Questions {
		if strings.EqualFold(strings.TrimSpace(q.Status), "answered") {
			answered++
		}
		touched += len(q.IterationsTouched)
		findings += len(q.KeyFindings)
	}
	return oracleProgressSnapshot{
		Answered:   answered,
		Touched:    touched,
		Findings:   findings,
		Confidence: state.OverallConfidence,
	}
}

func oracleProgressedSince(before oracleProgressSnapshot, plan oraclePlanFile, state oracleStateFile) bool {
	after := snapshotOracleProgress(plan, state)
	if after.Answered > before.Answered || after.Touched > before.Touched || after.Findings > before.Findings || after.Confidence > before.Confidence {
		return true
	}
	for _, q := range plan.Questions {
		if containsOracleIteration(q.IterationsTouched, state.Iteration) {
			return true
		}
	}
	return false
}

func containsOracleIteration(items []int, target int) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func oracleReadyForCompletion(plan oraclePlanFile, state oracleStateFile) bool {
	return state.OverallConfidence >= state.TargetConfidence || oracleAllQuestionsAnswered(plan)
}

func oracleAllQuestionsAnswered(plan oraclePlanFile) bool {
	if len(plan.Questions) == 0 {
		return false
	}
	for _, q := range plan.Questions {
		if !strings.EqualFold(strings.TrimSpace(q.Status), "answered") {
			return false
		}
	}
	return true
}

func oracleOverallConfidence(plan oraclePlanFile) int {
	if len(plan.Questions) == 0 {
		return 0
	}
	total := 0
	count := 0
	for _, q := range plan.Questions {
		total += q.Confidence
		count++
	}
	if count == 0 {
		return 0
	}
	return total / count
}

func oracleQuestionCounts(plan oraclePlanFile) (total, answered, touched int) {
	total = len(plan.Questions)
	for _, q := range plan.Questions {
		if strings.EqualFold(strings.TrimSpace(q.Status), "answered") {
			answered++
		}
		if len(q.IterationsTouched) > 0 {
			touched++
		}
	}
	return total, answered, touched
}

func oracleStopRequested(stopPath string) bool {
	return fileExists(stopPath)
}

func writeOracleIterationArtifact(paths oraclePaths, state oracleStateFile, attempt int, result codex.WorkerResult, invokeErr error, responsePath string) (string, error) {
	if err := os.MkdirAll(paths.DiscoveriesDir, 0755); err != nil {
		return "", fmt.Errorf("create oracle discoveries dir: %w", err)
	}
	artifact := oracleIterationArtifact{
		Iteration:  state.Iteration,
		Attempt:    attempt,
		Phase:      state.Phase,
		WorkerName: result.WorkerName,
		Caste:      result.Caste,
		TaskID:     result.TaskID,
		Status:     strings.TrimSpace(result.Status),
		Summary:    strings.TrimSpace(result.Summary),
		Blockers:   append([]string(nil), result.Blockers...),
		Response:   strings.TrimSpace(responsePath),
		RawOutput:  compactOracleText(result.RawOutput, 4000),
		DurationMS: result.Duration.Milliseconds(),
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}
	if invokeErr != nil {
		artifact.Error = invokeErr.Error()
	} else if result.Error != nil {
		artifact.Error = result.Error.Error()
	}
	if artifact.Status == "" && artifact.Error != "" {
		artifact.Status = "failed"
	}
	path := filepath.Join(paths.DiscoveriesDir, fmt.Sprintf("iteration-%02d.json", state.Iteration))
	encoded, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal oracle discovery artifact: %w", err)
	}
	if err := os.WriteFile(path, append(encoded, '\n'), 0644); err != nil {
		return "", fmt.Errorf("write oracle discovery artifact: %w", err)
	}
	return path, nil
}

func oracleWorkerFailureSummary(result codex.WorkerResult, invokeErr error, artifactPath string) string {
	reason := ""
	switch {
	case invokeErr != nil:
		reason = invokeErr.Error()
	case result.Error != nil:
		reason = result.Error.Error()
	case strings.TrimSpace(result.Summary) != "":
		reason = strings.TrimSpace(result.Summary)
	case len(result.Blockers) > 0:
		reason = strings.Join(result.Blockers, "; ")
	case strings.TrimSpace(result.RawOutput) != "":
		reason = compactOracleText(result.RawOutput, 240)
	default:
		reason = "Oracle worker blocked without a structured summary."
	}
	if strings.TrimSpace(artifactPath) != "" {
		return fmt.Sprintf("%s See %s.", reason, artifactPath)
	}
	return reason
}

func compactOracleText(text string, limit int) string {
	text = strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
	if limit <= 0 || len(text) <= limit {
		return text
	}
	return strings.TrimSpace(text[:limit]) + "..."
}

func renderOracleIterationPreview(state oracleStateFile, plan oraclePlanFile) string {
	nextTarget := emptyFallback(strings.TrimSpace(state.ActiveQuestionText), "all questions answered")
	if strings.TrimSpace(state.ActiveQuestionID) != "" {
		nextTarget = oracleQuestionLabel(oracleQuestion{ID: state.ActiveQuestionID, Text: state.ActiveQuestionText})
	}

	var b strings.Builder
	b.WriteString(renderBanner("🔮🐜", "Oracle Loop"))
	b.WriteString(visualDivider)
	fmt.Fprintf(&b, "Phase: %s\n", emptyFallback(state.Phase, "survey"))
	fmt.Fprintf(&b, "Iteration: %d of %d\n", state.Iteration, state.MaxIterations)
	if state.ActiveAttempt > 0 {
		fmt.Fprintf(&b, "Attempt: %d of %d\n", state.ActiveAttempt, defaultOracleMaxAttempts)
	}
	if strings.TrimSpace(state.ActiveReasoning) != "" {
		fmt.Fprintf(&b, "Reasoning: %s\n", state.ActiveReasoning)
	}
	if state.ActiveTimeoutSec > 0 {
		fmt.Fprintf(&b, "Watchdog: %s\n", oracleDurationLabel(time.Duration(state.ActiveTimeoutSec)*time.Second))
	}
	fmt.Fprintf(&b, "Confidence: %d%% / %d%%\n", state.OverallConfidence, state.TargetConfidence)
	fmt.Fprintf(&b, "Target: %s\n", nextTarget)
	return strings.TrimSpace(b.String())
}

func renderOracleRetryPreview(state oracleStateFile) string {
	var b strings.Builder
	b.WriteString(renderBanner("🔁", "Oracle Retry"))
	b.WriteString(visualDivider)
	fmt.Fprintf(&b, "Iteration: %d of %d\n", state.Iteration, state.MaxIterations)
	fmt.Fprintf(&b, "Attempt: %d of %d\n", state.ActiveAttempt+1, defaultOracleMaxAttempts)
	fmt.Fprintf(&b, "Target: %s\n", oracleQuestionLabel(oracleQuestion{ID: state.ActiveQuestionID, Text: state.ActiveQuestionText}))
	fmt.Fprintf(&b, "Reason: %s\n", emptyFallback(strings.TrimSpace(state.Summary), "Retrying after worker failure"))
	return strings.TrimSpace(b.String())
}

// extractKeywords splits text into unique lowercase words of 3+ characters.
// Used for keyword-overlap scoring in smart question selection.
func extractKeywords(text string) []string {
	words := strings.Fields(text)
	seen := make(map[string]bool, len(words))
	var result []string
	for _, w := range words {
		lower := strings.ToLower(w)
		// Strip common punctuation from edges
		lower = strings.Trim(lower, ".,;:!?\"'()[]{}")
		if len(lower) < 3 {
			continue
		}
		if seen[lower] {
			continue
		}
		seen[lower] = true
		result = append(result, lower)
	}
	return result
}

// countKeywordOverlap returns how many keywords in set B appear in set A.
func countKeywordOverlap(a, b []string) int {
	bSet := make(map[string]bool, len(b))
	for _, kw := range b {
		bSet[kw] = true
	}
	count := 0
	for _, kw := range a {
		if bSet[kw] {
			count++
		}
	}
	return count
}

// scoreQuestionImpact computes a weighted impact score for a question.
// Higher score = more impactful to research next.
// Weights: gap overlap (0.35), contradiction overlap (0.25), cross-question benefit (0.20), confidence deficit (0.20).
// Returns value in [0, 1].
func scoreQuestionImpact(q oracleQuestion, plan oraclePlanFile, state oracleStateFile) float64 {
	qKeywords := extractKeywords(q.Text)

	// Factor 1: Gap overlap (0.35) -- how many open gaps share keywords with this question
	gapScore := 0.0
	if len(state.OpenGaps) > 0 {
		gapCount := 0
		for _, gap := range state.OpenGaps {
			gapCount += countKeywordOverlap(extractKeywords(gap), qKeywords)
		}
		// Normalize by total possible overlap (all gap keywords)
		totalGapKeywords := 0
		for _, gap := range state.OpenGaps {
			totalGapKeywords += len(extractKeywords(gap))
		}
		if totalGapKeywords > 0 {
			gapScore = float64(gapCount) / float64(totalGapKeywords)
		}
	}

	// Factor 2: Contradiction overlap (0.25)
	contradictionScore := 0.0
	if len(state.Contradictions) > 0 {
		conCount := 0
		for _, con := range state.Contradictions {
			conCount += countKeywordOverlap(extractKeywords(con), qKeywords)
		}
		totalConKeywords := 0
		for _, con := range state.Contradictions {
			totalConKeywords += len(extractKeywords(con))
		}
		if totalConKeywords > 0 {
			contradictionScore = float64(conCount) / float64(totalConKeywords)
		}
	}

	// Factor 3: Cross-question benefit (0.20) -- how many other unanswered questions share keywords
	crossScore := 0.0
	otherCount := 0
	for _, other := range plan.Questions {
		if other.ID == q.ID {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(other.Status), "answered") {
			continue
		}
		otherCount++
	}
	if otherCount > 0 {
		overlapCount := 0
		for _, other := range plan.Questions {
			if other.ID == q.ID {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(other.Status), "answered") {
				continue
			}
			if countKeywordOverlap(extractKeywords(other.Text), qKeywords) > 0 {
				overlapCount++
			}
		}
		crossScore = float64(overlapCount) / float64(otherCount)
	}

	// Factor 4: Confidence deficit (0.20)
	deficit := float64(state.TargetConfidence - q.Confidence)
	if deficit < 0 {
		deficit = 0
	}
	confidenceScore := deficit / 100.0
	if confidenceScore > 1.0 {
		confidenceScore = 1.0
	}

	score := 0.35*gapScore + 0.25*contradictionScore + 0.20*crossScore + 0.20*confidenceScore
	if score > 1.0 {
		score = 1.0
	}
	return score
}

// selectOracleQuestionSmart picks the most impactful next question using multi-factor scoring.
// It replaces the naive lowest-confidence selection with gap/contradiction/benefit/confidence analysis.
func selectOracleQuestionSmart(plan oraclePlanFile, state oracleStateFile) oracleQuestion {
	// Edge case: no questions
	if len(plan.Questions) == 0 {
		return oracleQuestion{ID: fmt.Sprintf("iteration-%d", state.Iteration), Text: "No oracle questions are currently available."}
	}

	var best oracleQuestion
	bestScore := -1.0
	anyUnanswered := false
	for _, q := range plan.Questions {
		if strings.EqualFold(strings.TrimSpace(q.Status), "answered") {
			continue
		}
		anyUnanswered = true
		score := scoreQuestionImpact(q, plan, state)
		// Priority boost for untouched questions
		if len(q.IterationsTouched) == 0 {
			score += 0.1
		}
		if score > bestScore {
			bestScore = score
			best = q
		}
	}
	if anyUnanswered {
		return best
	}
	// All answered -- return indicator
	return oracleQuestion{
		ID:   fmt.Sprintf("iteration-%d", state.Iteration),
		Text: "All oracle questions have been answered.",
	}
}

func selectOracleQuestion(plan oraclePlanFile, state oracleStateFile) oracleQuestion {
	for _, q := range plan.Questions {
		if len(q.IterationsTouched) == 0 {
			return q
		}
	}
	best := oracleQuestion{}
	bestSet := false
	for _, q := range plan.Questions {
		if strings.EqualFold(strings.TrimSpace(q.Status), "answered") {
			continue
		}
		if !bestSet || q.Confidence < best.Confidence {
			best = q
			bestSet = true
		}
	}
	if bestSet {
		return best
	}
	if len(plan.Questions) > 0 {
		return plan.Questions[0]
	}
	return oracleQuestion{ID: fmt.Sprintf("iteration-%d", state.Iteration), Text: "No oracle questions are currently available."}
}

func oracleQuestionLabel(question oracleQuestion) string {
	id := strings.TrimSpace(question.ID)
	text := strings.TrimSpace(question.Text)
	switch {
	case id != "" && text != "":
		return fmt.Sprintf("%s — %s", id, text)
	case id != "":
		return id
	case text != "":
		return text
	default:
		return "unassigned oracle question"
	}
}

func oracleTopicHeadline(topic string) string {
	for _, line := range strings.Split(topic, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return truncateString(line, 140)
		}
	}
	return ""
}

func oracleTopicSummary(topic string) string {
	headline := oracleTopicHeadline(topic)
	if headline == "" {
		return ""
	}
	body := strings.TrimSpace(topic)
	body = strings.TrimPrefix(body, headline)
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}
	return compactOracleText(body, 240)
}

func compactOracleFocusAreas(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = truncateString(strings.TrimSpace(item), 100)
		if item == "" {
			continue
		}
		result = append(result, item)
		if len(result) >= 4 {
			break
		}
	}
	return result
}

func renderOraclePriorFindings(plan oraclePlanFile, target oracleQuestion) string {
	for _, question := range plan.Questions {
		if question.ID != target.ID {
			continue
		}
		if len(question.KeyFindings) == 0 {
			return ""
		}
		var b strings.Builder
		limit := len(question.KeyFindings)
		if limit > 4 {
			limit = 4
		}
		for i := 0; i < limit; i++ {
			finding := question.KeyFindings[i]
			b.WriteString("- ")
			b.WriteString(finding.Text)
			if len(finding.SourceIDs) > 0 {
				b.WriteString(" [")
				b.WriteString(strings.Join(finding.SourceIDs, ", "))
				b.WriteString("]")
			}
			b.WriteString("\n")
		}
		return b.String()
	}
	return ""
}

func renderOracleCurrentGaps(state oracleStateFile) string {
	lines := make([]string, 0, len(state.OpenGaps)+len(state.Contradictions))
	for _, gap := range state.OpenGaps {
		lines = append(lines, "- Gap: "+gap)
	}
	for _, contradiction := range state.Contradictions {
		lines = append(lines, "- Contradiction: "+contradiction)
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

func oracleRetryableFailure(result codex.WorkerResult, invokeErr error) bool {
	if invokeErr != nil {
		return true
	}
	if result.Error != nil {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(result.Status), "failed")
}

func oracleWorkerConfigOverrides(policy oracleAttemptPolicy) []string {
	effort := strings.TrimSpace(policy.ReasoningEffort)
	if effort == "" {
		effort = strings.TrimSpace(os.Getenv("AETHER_CODEX_ORACLE_REASONING_EFFORT"))
	}
	if effort == "" {
		effort = defaultOracleReasoningEffort
	}
	overrides := []string{
		fmt.Sprintf("model_reasoning_effort=%q", effort),
	}
	if model := strings.TrimSpace(os.Getenv("AETHER_CODEX_ORACLE_MODEL")); model != "" {
		overrides = append(overrides, fmt.Sprintf("model=%q", model))
	}
	return overrides
}

func defaultOracleAttemptPolicy(phase string, attempt int) oracleAttemptPolicy {
	policy := oracleAttemptPolicy{
		ReasoningEffort: defaultOracleReasoningEffort,
		Timeout:         defaultOracleTimeout,
		Heartbeat:       defaultOracleHeartbeat,
	}

	switch strings.ToLower(strings.TrimSpace(phase)) {
	case "survey":
		policy.ReasoningEffort = "low"
		policy.Timeout = 3 * time.Minute
	case "verify":
		policy.ReasoningEffort = "high"
		policy.Timeout = 6 * time.Minute
	default:
		policy.ReasoningEffort = "medium"
		policy.Timeout = 5 * time.Minute
	}

	if attempt > 1 {
		policy.Timeout += time.Minute
	}

	if override := strings.TrimSpace(os.Getenv("AETHER_CODEX_ORACLE_REASONING_EFFORT")); override != "" {
		policy.ReasoningEffort = override
	}
	return policy
}

func oracleDurationLabel(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}
	return d.Truncate(time.Second).String()
}
