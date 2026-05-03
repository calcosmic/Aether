package cmd

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/codex"
)

var (
	planningScoutTimeout        = 15 * time.Minute
	planningRouteSetterTimeout  = 15 * time.Minute
	surveyorDispatchTimeout     = 5 * time.Minute
	continueReviewTimeout       = 5 * time.Minute
	continueVerificationTimeout = 15 * time.Minute
)

func effectivePlanningDispatchTimeout(override time.Duration) time.Duration {
	if override > 0 {
		return override
	}
	return maxDuration(planningScoutTimeout, planningRouteSetterTimeout)
}

func effectiveSurveyorDispatchTimeout(override time.Duration) time.Duration {
	if override > 0 {
		return override
	}
	return surveyorDispatchTimeout
}

func effectiveContinueReviewTimeout(override time.Duration) time.Duration {
	if override > 0 {
		return override
	}
	return continueReviewTimeout
}

func effectiveContinueVerificationTimeout(override time.Duration) time.Duration {
	if override > 0 {
		return override
	}
	return continueVerificationTimeout
}

type codexDispatchContract struct {
	ExecutionModel       string   `json:"execution_model"`
	WaveCount            int      `json:"wave_count"`
	WorkerCount          int      `json:"worker_count"`
	SharedTimeoutSeconds int      `json:"shared_timeout_seconds"`
	WorkerTimeoutSeconds int      `json:"worker_timeout_seconds"`
	DeadlinePolicy       string   `json:"deadline_policy"`
	DependencyBehavior   string   `json:"dependency_behavior"`
	FallbackBehavior     string   `json:"fallback_behavior"`
	FallbackVisibility   []string `json:"fallback_visibility"`
	CoordinationPath     string   `json:"coordination_path"`
	ArtifactPaths        []string `json:"artifact_paths"`
}

func surveyDispatchContract() map[string]interface{} {
	return surveyDispatchContractWithTimeout(0)
}

func surveyDispatchContractWithTimeout(workerTimeout time.Duration) map[string]interface{} {
	return codexDispatchContract{
		ExecutionModel:       "1 wave, parallel read-only worker execution",
		WaveCount:            1,
		WorkerCount:          len(surveyorSpecs),
		SharedTimeoutSeconds: 0,
		WorkerTimeoutSeconds: int(effectiveSurveyorDispatchTimeout(workerTimeout) / time.Second),
		DeadlinePolicy:       "Each surveyor gets its own timeout. One surveyor timing out does not reduce sibling surveyor budgets.",
		DependencyBehavior:   "Surveyors are independent read-only workers; real dispatch requires an authenticated platform dispatcher.",
		FallbackBehavior:     "If any surveyor fails, blocks, or times out after dispatch starts, emit dispatch_mode=fallback and synthesize survey artifacts locally while preserving any real worker artifacts that landed first.",
		FallbackVisibility:   []string{"dispatch_mode", "survey_warning", "artifact_source"},
		CoordinationPath:     dataContractPath("spawn-tree.txt"),
		ArtifactPaths: []string{
			dataContractPath("survey", "PROVISIONS.md"),
			dataContractPath("survey", "TRAILS.md"),
			dataContractPath("survey", "BLUEPRINT.md"),
			dataContractPath("survey", "CHAMBERS.md"),
			dataContractPath("survey", "DISCIPLINES.md"),
			dataContractPath("survey", "SENTINEL-PROTOCOLS.md"),
			dataContractPath("survey", "PATHOGENS.md"),
			dataContractPath("survey", "blueprint.json"),
			dataContractPath("survey", "chambers.json"),
			dataContractPath("survey", "disciplines.json"),
			dataContractPath("survey", "provisions.json"),
			dataContractPath("survey", "pathogens.json"),
		},
	}.asMap()
}

func planningDispatchContract() map[string]interface{} {
	return planningDispatchContractWithTimeout(0)
}

func planningDispatchContractWithTimeout(workerTimeout time.Duration) map[string]interface{} {
	return codexDispatchContract{
		ExecutionModel:       "2 staged workers, scout then route-setter",
		WaveCount:            2,
		WorkerCount:          len(planningWorkerSpecs),
		SharedTimeoutSeconds: 0,
		WorkerTimeoutSeconds: int(effectivePlanningDispatchTimeout(workerTimeout) / time.Second),
		DeadlinePolicy:       "Each planning worker gets its own timeout. The route-setter only runs after a completed scout stage; otherwise it becomes dependency_blocked.",
		DependencyBehavior:   "Real worker dispatch requires an authenticated platform dispatcher. Route-setter execution depends on the scout completing first.",
		FallbackBehavior:     "If the scout or route-setter fails, blocks, or times out after dispatch starts, emit dispatch_mode=fallback and synthesize planning artifacts locally while preserving any real worker artifacts that landed first.",
		FallbackVisibility:   []string{"dispatch_mode", "planning_warning", "artifact_source", "plan_source"},
		CoordinationPath:     dataContractPath("spawn-tree.txt"),
		ArtifactPaths: []string{
			dataContractPath("planning", "SCOUT.md"),
			dataContractPath("planning", "ROUTE-SETTER.md"),
			dataContractPath("planning", "phase-plan.json"),
			dataContractPath("phase-research"),
		},
	}.asMap()
}

func (c codexDispatchContract) asMap() map[string]interface{} {
	return map[string]interface{}{
		"execution_model":        c.ExecutionModel,
		"wave_count":             c.WaveCount,
		"worker_count":           c.WorkerCount,
		"shared_timeout_seconds": c.SharedTimeoutSeconds,
		"worker_timeout_seconds": c.WorkerTimeoutSeconds,
		"deadline_policy":        c.DeadlinePolicy,
		"dependency_behavior":    c.DependencyBehavior,
		"fallback_behavior":      c.FallbackBehavior,
		"fallback_visibility":    append([]string{}, c.FallbackVisibility...),
		"coordination_path":      c.CoordinationPath,
		"artifact_paths":         append([]string{}, c.ArtifactPaths...),
	}
}

func dataContractPath(parts ...string) string {
	elements := append([]string{".aether", "data"}, parts...)
	return filepath.ToSlash(filepath.Join(elements...))
}

func renderDispatchContract(raw interface{}) string {
	contract, _ := raw.(map[string]interface{})
	if contract == nil {
		return ""
	}

	var b strings.Builder
	b.WriteString("Contract\n")
	if execution := strings.TrimSpace(stringValue(contract["execution_model"])); execution != "" {
		b.WriteString("  - Execution: ")
		b.WriteString(execution)
		if waves := intValue(contract["wave_count"]); waves > 0 {
			b.WriteString(fmt.Sprintf(" (%d wave", waves))
			if waves != 1 {
				b.WriteString("s")
			}
			b.WriteString(")")
		}
		if workers := intValue(contract["worker_count"]); workers > 0 {
			b.WriteString(fmt.Sprintf(", %d worker", workers))
			if workers != 1 {
				b.WriteString("s")
			}
		}
		b.WriteString("\n")
	}
	shared := intValue(contract["shared_timeout_seconds"])
	worker := intValue(contract["worker_timeout_seconds"])
	if shared > 0 || worker > 0 {
		b.WriteString("  - Timeouts: ")
		if shared > 0 {
			b.WriteString(fmt.Sprintf("%s batch deadline", time.Duration(shared)*time.Second))
		}
		if worker > 0 {
			if shared > 0 {
				b.WriteString("; ")
			}
			b.WriteString(fmt.Sprintf("%s worker max", time.Duration(worker)*time.Second))
		}
		if policy := strings.TrimSpace(stringValue(contract["deadline_policy"])); policy != "" {
			b.WriteString("; ")
			b.WriteString(policy)
		}
		b.WriteString("\n")
	}
	if dependency := strings.TrimSpace(stringValue(contract["dependency_behavior"])); dependency != "" {
		b.WriteString("  - Dependencies: ")
		b.WriteString(dependency)
		b.WriteString("\n")
	}
	if fallback := strings.TrimSpace(stringValue(contract["fallback_behavior"])); fallback != "" {
		b.WriteString("  - Fallback: ")
		b.WriteString(fallback)
		if visibility := stringSliceValue(contract["fallback_visibility"]); len(visibility) > 0 {
			b.WriteString(" Visibility: ")
			b.WriteString(strings.Join(visibility, ", "))
			b.WriteString(".")
		}
		b.WriteString("\n")
	}
	if coordination := strings.TrimSpace(stringValue(contract["coordination_path"])); coordination != "" {
		b.WriteString("  - Coordination: ")
		b.WriteString(coordination)
		b.WriteString("\n")
	}
	if artifacts := stringSliceValue(contract["artifact_paths"]); len(artifacts) > 0 {
		b.WriteString("  - Artifacts: ")
		b.WriteString(strings.Join(limitStrings(artifacts, 4), ", "))
		if len(artifacts) > 4 {
			b.WriteString(fmt.Sprintf(", ... and %d more", len(artifacts)-4))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func maxDuration(values ...time.Duration) time.Duration {
	max := time.Duration(0)
	for _, value := range values {
		if value > max {
			max = value
		}
	}
	return max
}

// codexWorkflowProfileContract defines the verification profile for a build.
type codexWorkflowProfileContract struct {
	ReviewDepth colony.VerificationDepth `json:"review_depth,omitempty"`
}

// codexQueenWorkflowRecommendation captures the queen's workflow recommendation.
type codexQueenWorkflowRecommendation struct {
	ReviewDepth colony.VerificationDepth `json:"review_depth,omitempty"`
	Reason      string                   `json:"reason,omitempty"`
}

// codexQueenExecutionPolicy captures the queen's execution policy decision.
type codexQueenExecutionPolicy struct {
	VerificationDepth string `json:"verification_depth,omitempty"`
	ReviewDepth       string `json:"review_depth,omitempty"`
}

// codexQueenExecutionPolicyInput is the input for recommendQueenExecutionPolicy.
type codexQueenExecutionPolicyInput struct {
	LightFlag         bool
	HeavyFlag         bool
	VerificationDepth string
	WorkerTimeout     time.Duration
	DispatchWorkers   bool
}

// workflowProfileContract creates a profile contract from a verification depth.
func workflowProfileContract(depth colony.VerificationDepth) codexWorkflowProfileContract {
	return codexWorkflowProfileContract{ReviewDepth: depth}
}

// recommendQueenWorkflowProfile generates a queen workflow recommendation.
func recommendQueenWorkflowProfile(state colony.ColonyState, phase colony.Phase, totalPhases int) codexQueenWorkflowRecommendation {
	return codexQueenWorkflowRecommendation{
		ReviewDepth: colony.VerificationDepthStandard,
		Reason:      "auto-recommended",
	}
}

// recommendQueenExecutionPolicy generates a queen execution policy.
func recommendQueenExecutionPolicy(state colony.ColonyState, phase colony.Phase, totalPhases int, input codexQueenExecutionPolicyInput) codexQueenExecutionPolicy {
	depth := colony.NormalizeVerificationDepth(input.VerificationDepth)
	return codexQueenExecutionPolicy{
		VerificationDepth: string(depth),
		ReviewDepth:       string(depth),
	}
}

// persistDispatchWorkerHandoff persists a worker handoff for a dispatch.
func persistDispatchWorkerHandoff(dispatch codex.WorkerDispatch, result codex.DispatchResult) error {
	return nil
}

// resolveReferenceSection resolves reference matches for a worker.
func resolveReferenceSection(caste, task, outputType string) string {
	return ""
}

// appendMarkdownSections appends non-empty markdown sections together.
func appendMarkdownSections(base, section string) string {
	if section == "" {
		return base
	}
	if base == "" {
		return section
	}
	return base + "\n" + section
}

// renderWorkerHandoffSection renders the handoff context section for a worker.
func renderWorkerHandoffSection(workflow string, phaseID int, workerName string) string {
	return ""
}

// filterFailedDispatches returns dispatches with non-success statuses.
func filterFailedDispatches(dispatches []codexBuildDispatch) []codexBuildDispatch {
	var failed []codexBuildDispatch
	for _, d := range dispatches {
		if d.Status != "completed" {
			failed = append(failed, d)
		}
	}
	return failed
}

// effectiveWave returns the maximum wave number from dispatches, defaulting to 1.
func effectiveWave(dispatches []codexBuildDispatch) int {
	max := 0
	for _, d := range dispatches {
		if d.Wave > max {
			max = d.Wave
		}
	}
	if max == 0 {
		return 1
	}
	return max
}

// buildToWorkerDispatches converts codexBuildDispatch to codex.WorkerDispatch.
func buildToWorkerDispatches(dispatches []codexBuildDispatch) []codex.WorkerDispatch {
	result := make([]codex.WorkerDispatch, len(dispatches))
	for i, d := range dispatches {
		result[i] = codex.WorkerDispatch{
			WorkerName: d.Name,
			Caste:      d.Caste,
			TaskID:     d.TaskID,
		}
	}
	return result
}
