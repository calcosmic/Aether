package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

const workerHandoffsPath = "handoffs/worker-handoffs.json"

var (
	planningScoutTimeout        = 15 * time.Minute
	planningRouteSetterTimeout  = 15 * time.Minute
	surveyorDispatchTimeout     = 5 * time.Minute
	continueReviewTimeout       = 10 * time.Minute
	continueVerificationTimeout = 15 * time.Minute
)

// dispatchContractPathOverride allows tests to point at a temporary policy file.
// When empty, the default colony/policies/dispatch-contract.yaml is used.
var dispatchContractPathOverride string

// dispatchContractPolicy mirrors the YAML structure for the dispatch contract policy.
type dispatchContractPolicy struct {
	ExecutionModels          map[string]string   `yaml:"execution_models"`
	DeadlinePolicies         map[string]string   `yaml:"deadline_policies"`
	DependencyBehaviors      map[string]string   `yaml:"dependency_behaviors"`
	FallbackBehaviors        map[string]string   `yaml:"fallback_behaviors"`
	FallbackVisibility       map[string][]string `yaml:"fallback_visibility"`
	ResultCollectionPolicies map[string]string   `yaml:"result_collection_policies"`
}

// dispatchContractPolicyWrapper matches the top-level YAML key.
type dispatchContractPolicyWrapper struct {
	DispatchContract dispatchContractPolicy `yaml:"dispatch_contract"`
}

var (
	loadedDispatchPolicy     *dispatchContractPolicy
	loadedDispatchPolicyOnce sync.Once
)

// loadDispatchContractPolicy loads the dispatch contract policy from YAML.
// It caches the result and falls back to nil on any error so callers can use
// hardcoded defaults.
func loadDispatchContractPolicy() *dispatchContractPolicy {
	loadedDispatchPolicyOnce.Do(func() {
		path := dispatchContractPathOverride
		if path == "" {
			path = policyPath("dispatch-contract")
		}
		var wrapper dispatchContractPolicyWrapper
		if err := loadYAMLPolicy(path, &wrapper); err == nil {
			loadedDispatchPolicy = &wrapper.DispatchContract
		}
	})
	return loadedDispatchPolicy
}

// Hardcoded fallback strings for dispatch contract fields.
const (
	fallbackSurveyExecutionModel               = "1 wave, parallel read-only worker execution"
	fallbackPlanningExecutionModel             = "2 staged workers, scout then route-setter"
	fallbackSurveyDeadlinePolicy               = "Each surveyor gets its own timeout. One surveyor timing out does not reduce sibling surveyor budgets."
	fallbackPlanningDeadlinePolicy             = "Each planning worker gets its own timeout. The route-setter only runs after a completed scout stage; otherwise it becomes dependency_blocked."
	fallbackSurveyDependencyBehavior           = "Surveyors are independent read-only workers; real dispatch requires an authenticated platform dispatcher."
	fallbackPlanningDependencyBehavior         = "Real worker dispatch requires an authenticated platform dispatcher. Route-setter execution depends on the scout completing first."
	fallbackPlanningExtendedDependencyBehavior = "Real worker dispatch requires an authenticated platform dispatcher. Supporting planning castes may contribute evidence, and route-setter finalization is selected by caste identity rather than fixed array position."
	fallbackSurveyFallbackBehavior             = "If any surveyor fails, blocks, or times out after dispatch starts, emit dispatch_mode=fallback and synthesize survey artifacts locally while preserving any real worker artifacts that landed first."
	// fallbackPlanningFallbackBehavior deliberately does NOT match
	// colony/policies/dispatch-contract.yaml's (now-deleted) planning
	// fallback_behavior text. 191-02's field-by-field diff found this field
	// diverging, but folding the YAML's stale "synthesize locally" text into
	// this constant -- the plan's literal default fold direction -- would
	// silently regress a deliberate, already-shipped, already-tested
	// fail-closed design: TestPlanIncludesDispatchContract
	// (cmd/codex_plan_test.go), TestPlanVisualOutputShowsDispatchContractDetails
	// (cmd/codex_visuals_test.go), and the committed golden fixture
	// (cmd/testdata/golden_plan.txt) all pin THIS text -- not the YAML's --
	// as correct, and none of the three could ever have been exercising the
	// real colony/ file (Go's test runner sets CWD to the package directory,
	// so the bare "colony/policies/..." path never resolved in any of them).
	// Deleting the stale file fixes a real dev-checkout/every-other-install
	// inconsistency rather than preserving it. See 191-02-SUMMARY.md.
	fallbackPlanningFallbackBehavior       = "Normal planning does not fall back to local synthesis. If Scout or Route-Setter workers are unavailable, blocked, failed, or timed out, stop with recovery guidance; only explicit `aether plan --synthetic` may produce dispatch_mode=synthetic local preview artifacts."
	fallbackSurveyResultCollectionPolicy   = "Wrapper result artifacts must stay outside .aether/data; finalizers reject malformed completion JSON and .aether/data completion files."
	fallbackPlanningResultCollectionPolicy = "A structurally valid completed result wins over a timeout placeholder for the same worker; duplicate terminal results remain invalid."
)

var (
	fallbackSurveyFallbackVisibility = []string{"dispatch_mode", "survey_warning", "provider_diagnostics", "artifact_source"}
	// fallbackPlanningFallbackVisibility deliberately does NOT match
	// colony/policies/dispatch-contract.yaml's (now-deleted) planning
	// fallback_visibility list, for the same reason documented above
	// fallbackPlanningFallbackBehavior: "synthetic"/"synthetic_warning" are
	// the tested, golden-fixture-locked, fail-closed design this constant
	// must keep describing, not the stale YAML's "provider_diagnostics".
	fallbackPlanningFallbackVisibility = []string{"dispatch_mode", "planning_warning", "synthetic", "synthetic_warning", "artifact_source", "plan_source", "planning_loop"}
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
	ExecutionModel         string   `json:"execution_model"`
	WaveCount              int      `json:"wave_count"`
	WorkerCount            int      `json:"worker_count"`
	SharedTimeoutSeconds   int      `json:"shared_timeout_seconds"`
	WorkerTimeoutSeconds   int      `json:"worker_timeout_seconds"`
	DeadlinePolicy         string   `json:"deadline_policy"`
	DependencyBehavior     string   `json:"dependency_behavior"`
	FallbackBehavior       string   `json:"fallback_behavior"`
	FallbackVisibility     []string `json:"fallback_visibility"`
	CoordinationPath       string   `json:"coordination_path"`
	ArtifactPaths          []string `json:"artifact_paths"`
	ResultArtifactPaths    []string `json:"result_artifact_paths,omitempty"`
	ResultCollectionPolicy string   `json:"result_collection_policy,omitempty"`
}

func surveyDispatchContract() map[string]interface{} {
	return surveyDispatchContractWithTimeout(0)
}

func surveyDispatchContractWithTimeout(workerTimeout time.Duration) map[string]interface{} {
	p := loadDispatchContractPolicy()
	executionModel := fallbackSurveyExecutionModel
	deadlinePolicy := fallbackSurveyDeadlinePolicy
	dependencyBehavior := fallbackSurveyDependencyBehavior
	fallbackBehavior := fallbackSurveyFallbackBehavior
	fallbackVisibility := append([]string{}, fallbackSurveyFallbackVisibility...)
	resultCollectionPolicy := fallbackSurveyResultCollectionPolicy
	if p != nil {
		if v, ok := p.ExecutionModels["survey"]; ok && v != "" {
			executionModel = v
		}
		if v, ok := p.DeadlinePolicies["survey"]; ok && v != "" {
			deadlinePolicy = v
		}
		if v, ok := p.DependencyBehaviors["survey"]; ok && v != "" {
			dependencyBehavior = v
		}
		if v, ok := p.FallbackBehaviors["survey"]; ok && v != "" {
			fallbackBehavior = v
		}
		if v, ok := p.FallbackVisibility["survey"]; ok && len(v) > 0 {
			fallbackVisibility = append([]string{}, v...)
		}
		if v, ok := p.ResultCollectionPolicies["survey"]; ok && v != "" {
			resultCollectionPolicy = v
		}
	}
	return codexDispatchContract{
		ExecutionModel:         executionModel,
		WaveCount:              1,
		WorkerCount:            len(surveyorSpecs),
		SharedTimeoutSeconds:   0,
		WorkerTimeoutSeconds:   int(effectiveSurveyorDispatchTimeout(workerTimeout) / time.Second),
		DeadlinePolicy:         deadlinePolicy,
		DependencyBehavior:     dependencyBehavior,
		FallbackBehavior:       fallbackBehavior,
		FallbackVisibility:     fallbackVisibility,
		CoordinationPath:       dataContractPath("spawn-tree.txt"),
		ResultArtifactPaths:    []string{finalizerCompletionTempPattern},
		ResultCollectionPolicy: resultCollectionPolicy,
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
	p := loadDispatchContractPolicy()
	executionModel := fallbackPlanningExecutionModel
	deadlinePolicy := fallbackPlanningDeadlinePolicy
	dependencyBehavior := fallbackPlanningDependencyBehavior
	fallbackBehavior := fallbackPlanningFallbackBehavior
	fallbackVisibility := append([]string{}, fallbackPlanningFallbackVisibility...)
	resultCollectionPolicy := fallbackPlanningResultCollectionPolicy
	if p != nil {
		if v, ok := p.ExecutionModels["planning"]; ok && v != "" {
			executionModel = v
		}
		if v, ok := p.DeadlinePolicies["planning"]; ok && v != "" {
			deadlinePolicy = v
		}
		if v, ok := p.DependencyBehaviors["planning"]; ok && v != "" {
			dependencyBehavior = v
		}
		if v, ok := p.FallbackBehaviors["planning"]; ok && v != "" {
			fallbackBehavior = v
		}
		if v, ok := p.FallbackVisibility["planning"]; ok && len(v) > 0 {
			fallbackVisibility = append([]string{}, v...)
		}
		if v, ok := p.ResultCollectionPolicies["planning"]; ok && v != "" {
			resultCollectionPolicy = v
		}
	}
	return codexDispatchContract{
		ExecutionModel:         executionModel,
		WaveCount:              2,
		WorkerCount:            len(planningWorkerSpecs),
		SharedTimeoutSeconds:   0,
		WorkerTimeoutSeconds:   int(effectivePlanningDispatchTimeout(workerTimeout) / time.Second),
		DeadlinePolicy:         deadlinePolicy,
		DependencyBehavior:     dependencyBehavior,
		FallbackBehavior:       fallbackBehavior,
		FallbackVisibility:     fallbackVisibility,
		CoordinationPath:       dataContractPath("spawn-tree.txt"),
		ResultArtifactPaths:    []string{finalizerCompletionTempPattern},
		ResultCollectionPolicy: resultCollectionPolicy,
		ArtifactPaths: []string{
			dataContractPath("planning", "SCOUT.md"),
			dataContractPath("planning", "ROUTE-SETTER.md"),
			dataContractPath("planning", "phase-plan.json"),
			dataContractPath("phase-research"),
		},
	}.asMap()
}

func planningDispatchContractForDispatches(dispatches []codexPlanningDispatch, workerTimeout time.Duration) map[string]interface{} {
	contract := planningDispatchContractWithTimeout(workerTimeout)
	if len(dispatches) == 0 {
		return contract
	}

	maxWave := 0
	for _, dispatch := range dispatches {
		if dispatch.Wave > maxWave {
			maxWave = dispatch.Wave
		}
	}
	if maxWave == 0 {
		maxWave = len(dispatches)
	}
	contract["worker_count"] = len(dispatches)
	contract["wave_count"] = maxWave
	if len(dispatches) != len(planningWorkerSpecs) {
		p := loadDispatchContractPolicy()
		em := fmt.Sprintf("%d staged planning workers, scout plus route-setter with supporting castes", len(dispatches))
		if p != nil {
			if v, ok := p.ExecutionModels["planning_extended"]; ok && v != "" {
				em = fmt.Sprintf(v, len(dispatches))
			}
		}
		contract["execution_model"] = em
		dep := fallbackPlanningExtendedDependencyBehavior
		if p != nil {
			if v, ok := p.DependencyBehaviors["planning_extended"]; ok && v != "" {
				dep = v
			}
		}
		contract["dependency_behavior"] = dep
	}
	return contract
}

func (c codexDispatchContract) asMap() map[string]interface{} {
	return map[string]interface{}{
		"execution_model":          c.ExecutionModel,
		"wave_count":               c.WaveCount,
		"worker_count":             c.WorkerCount,
		"shared_timeout_seconds":   c.SharedTimeoutSeconds,
		"worker_timeout_seconds":   c.WorkerTimeoutSeconds,
		"deadline_policy":          c.DeadlinePolicy,
		"dependency_behavior":      c.DependencyBehavior,
		"fallback_behavior":        c.FallbackBehavior,
		"fallback_visibility":      append([]string{}, c.FallbackVisibility...),
		"coordination_path":        c.CoordinationPath,
		"artifact_paths":           append([]string{}, c.ArtifactPaths...),
		"result_artifact_paths":    append([]string{}, c.ResultArtifactPaths...),
		"result_collection_policy": c.ResultCollectionPolicy,
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
	if resultArtifacts := stringSliceValue(contract["result_artifact_paths"]); len(resultArtifacts) > 0 {
		b.WriteString("  - Result artifacts: ")
		b.WriteString(strings.Join(limitStrings(resultArtifacts, 4), ", "))
		if len(resultArtifacts) > 4 {
			b.WriteString(fmt.Sprintf(", ... and %d more", len(resultArtifacts)-4))
		}
		b.WriteString("\n")
	}
	if policy := strings.TrimSpace(stringValue(contract["result_collection_policy"])); policy != "" {
		b.WriteString("  - Result collection: ")
		b.WriteString(policy)
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
	VerificationDepth string                         `json:"verification_depth,omitempty"`
	ReviewDepth       string                         `json:"review_depth,omitempty"`
	SpawnBudget       *codexQueenSpawnBudgetContract `json:"spawn_budget,omitempty"`
}

// codexQueenSpawnBudgetContract captures allowlisted spawn selection metadata.
// Worker fields count concrete manifest dispatches; caste fields describe the
// Queen's relevance budget before build-specific worker expansion.
type codexQueenSpawnBudgetContract struct {
	MaxWorkers              int               `json:"max_workers,omitempty"`
	SelectedWorkers         int               `json:"selected_workers,omitempty"`
	WorkerCount             int               `json:"worker_count,omitempty"`
	MaxSelectedCastes       int               `json:"max_selected_castes,omitempty"`
	SelectedCastes          int               `json:"selected_castes,omitempty"`
	PrunedWorkers           *int              `json:"pruned_workers,omitempty"`
	PrunedCastes            *int              `json:"pruned_castes,omitempty"`
	PreservedCastes         []string          `json:"preserved_castes,omitempty"`
	RequiredCastes          []string          `json:"required_castes,omitempty"`
	PolicyAddedCastes       []string          `json:"policy_added_castes,omitempty"`
	SelectedReasons         map[string]string `json:"selected_reasons,omitempty"`
	PrunedReasons           map[string]string `json:"pruned_reasons,omitempty"`
	SkippedCastes           []string          `json:"skipped_castes,omitempty"`
	OverflowRequiredWorkers *int              `json:"overflow_required_workers,omitempty"`
	RelevanceThreshold      *int              `json:"relevance_threshold,omitempty"`
	BudgetUnit              string            `json:"budget_unit,omitempty"`
	Reason                  string            `json:"reason,omitempty"`
	FlowType                string            `json:"flow_type,omitempty"`
	RiskLevel               string            `json:"risk_level,omitempty"`
	Castes                  []string          `json:"castes,omitempty"`
	Counts                  map[string]int    `json:"counts,omitempty"`
}

// codexQueenExecutionPolicyInput is the input for recommendQueenExecutionPolicy.
type codexQueenExecutionPolicyInput struct {
	LightFlag         bool
	HeavyFlag         bool
	VerificationDepth string
	WorkerTimeout     time.Duration
	DispatchWorkers   bool
}

type workerHandoffFile struct {
	Entries []workerHandoffRecord `json:"entries"`
}

type workerHandoffRecord struct {
	ID                     string   `json:"id"`
	Workflow               string   `json:"workflow,omitempty"`
	Phase                  int      `json:"phase,omitempty"`
	Wave                   int      `json:"wave,omitempty"`
	WorkerName             string   `json:"worker_name"`
	Caste                  string   `json:"caste,omitempty"`
	TaskID                 string   `json:"task_id,omitempty"`
	Status                 string   `json:"status,omitempty"`
	Summary                string   `json:"summary,omitempty"`
	ChangedFiles           []string `json:"changed_files,omitempty"`
	CommandsRun            []string `json:"commands_run,omitempty"`
	VerificationStatus     string   `json:"verification_status,omitempty"`
	KnownFailures          []string `json:"known_failures,omitempty"`
	OpenDecisions          []string `json:"open_decisions,omitempty"`
	Assumptions            []string `json:"assumptions,omitempty"`
	NextWorkerInstructions []string `json:"next_worker_instructions,omitempty"`
	DoNotRepeat            []string `json:"do_not_repeat,omitempty"`
	Freshness              string   `json:"freshness,omitempty"`
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
// Uses resolveVerificationDepth to apply smart defaults (phase position, keyword
// detection) when no explicit depth is provided, matching the continue path's
// depth resolution in resolveEffectiveContinueDepth.
func recommendQueenExecutionPolicy(state colony.ColonyState, phase colony.Phase, totalPhases int, input codexQueenExecutionPolicyInput) codexQueenExecutionPolicy {
	effectiveDepth := resolveVerificationDepthFlag(input.LightFlag, input.HeavyFlag, input.VerificationDepth)
	if effectiveDepth == "" {
		effectiveDepth = strings.TrimSpace(state.VerificationDepth)
	}
	depth := resolveVerificationDepth(phase, totalPhases, input.LightFlag, input.HeavyFlag, effectiveDepth)
	return codexQueenExecutionPolicy{
		VerificationDepth: string(depth),
		ReviewDepth:       string(depth),
	}
}

func enrichQueenExecutionPolicyWithSpawnBudget(policy codexQueenExecutionPolicy, state colony.ColonyState, phase colony.Phase, flowType string, reviewDepth colony.VerificationDepth, dispatches []codexBuildDispatch) codexQueenExecutionPolicy {
	policy.SpawnBudget = buildQueenSpawnBudgetContract(state, phase, flowType, reviewDepth, dispatches)
	return policy
}

func buildQueenSpawnBudgetContract(state colony.ColonyState, phase colony.Phase, flowType string, reviewDepth colony.VerificationDepth, dispatches []codexBuildDispatch) *codexQueenSpawnBudgetContract {
	flowType = normalizeQueenFlowType(flowType)
	budgetState := state
	if reviewDepth != "" {
		budgetState.VerificationDepth = string(reviewDepth)
	}

	budget := queenSpawnBudgetForPhase(phase, flowType, budgetState)
	threshold := spawnThreshold(flowType, budgetState)
	contract := &codexQueenSpawnBudgetContract{
		MaxSelectedCastes:  budget.MaxWorkers,
		RequiredCastes:     append([]string{}, budget.RequiredCastes...),
		RelevanceThreshold: &threshold,
		BudgetUnit:         "caste",
		Reason:             budget.Reason,
		FlowType:           budget.FlowType,
		RiskLevel:          budget.RiskLevel,
	}
	candidateBudgetDispatches := queenCandidateDispatches(phase, flowType, budgetState)
	candidateBudgetCastes := casteDispatchSummary(candidateBudgetDispatches)
	selectedBudgetCastes := casteDispatchSummary(queenOrchestrate(phase, flowType, budgetState))
	prunedBudgetCastes := stringSliceDifference(candidateBudgetCastes, selectedBudgetCastes)
	contract.PrunedCastes = intRef(len(prunedBudgetCastes))
	// The Queen's pruning budget operates on castes before build policy expands
	// selected castes into concrete worker dispatches, so this count mirrors
	// pruned_castes while budget_unit remains "caste".
	contract.PrunedWorkers = intRef(len(prunedBudgetCastes))
	contract.OverflowRequiredWorkers = intRef(maxInt(0, len(contract.RequiredCastes)-budget.MaxWorkers))
	for _, decision := range queenSpawnBudgetDecisions(candidateBudgetDispatches, budget) {
		rationale := strings.TrimSpace(decision.Rationale)
		if rationale == "" {
			continue
		}
		if decision.Selected {
			if contract.SelectedReasons == nil {
				contract.SelectedReasons = make(map[string]string)
			}
			contract.SelectedReasons[decision.Caste] = rationale
			continue
		}
		if contract.PrunedReasons == nil {
			contract.PrunedReasons = make(map[string]string)
		}
		contract.PrunedReasons[decision.Caste] = rationale
		contract.SkippedCastes = append(contract.SkippedCastes, decision.Caste)
	}
	sort.Strings(contract.SkippedCastes)

	if len(dispatches) > 0 {
		castes, counts := concreteDispatchCasteSummary(dispatches)
		workerCount := len(dispatches)
		contract.MaxWorkers = workerCount
		contract.SelectedWorkers = workerCount
		contract.WorkerCount = workerCount
		contract.SelectedCastes = len(castes)
		contract.PolicyAddedCastes = stringSliceDifference(castes, selectedBudgetCastes)
		contract.PreservedCastes = stringSliceIntersection(castes, contract.RequiredCastes)
		contract.Castes = castes
		contract.Counts = counts
		return contract
	}

	contract.MaxWorkers = len(selectedBudgetCastes)
	contract.SelectedWorkers = len(selectedBudgetCastes)
	contract.WorkerCount = len(selectedBudgetCastes)
	contract.SelectedCastes = len(selectedBudgetCastes)
	contract.PreservedCastes = stringSliceIntersection(selectedBudgetCastes, contract.RequiredCastes)
	contract.Castes = selectedBudgetCastes
	return contract
}

func concreteDispatchCasteSummary(dispatches []codexBuildDispatch) ([]string, map[string]int) {
	counts := make(map[string]int)
	for _, dispatch := range dispatches {
		caste := strings.TrimSpace(dispatch.Caste)
		if caste == "" {
			continue
		}
		counts[caste]++
	}
	if len(counts) == 0 {
		return nil, nil
	}

	castes := make([]string, 0, len(counts))
	for caste := range counts {
		castes = append(castes, caste)
	}
	sort.Strings(castes)
	return castes, counts
}

func casteDispatchSummary(dispatches []CasteDispatch) []string {
	if len(dispatches) == 0 {
		return nil
	}
	castes := make([]string, 0, len(dispatches))
	for _, dispatch := range dispatches {
		caste := strings.TrimSpace(dispatch.Caste)
		if caste == "" {
			continue
		}
		castes = append(castes, caste)
	}
	sort.Strings(castes)
	return castes
}

func intRef(value int) *int {
	return &value
}

func stringSliceDifference(values []string, excluded []string) []string {
	if len(values) == 0 {
		return nil
	}
	excludedSet := stringSet(excluded)
	diff := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || excludedSet[value] {
			continue
		}
		diff = append(diff, value)
	}
	if len(diff) == 0 {
		return nil
	}
	sort.Strings(diff)
	return diff
}

func stringSliceIntersection(values []string, allowed []string) []string {
	if len(values) == 0 || len(allowed) == 0 {
		return nil
	}
	allowedSet := stringSet(allowed)
	intersection := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || !allowedSet[value] {
			continue
		}
		intersection = append(intersection, value)
	}
	if len(intersection) == 0 {
		return nil
	}
	sort.Strings(intersection)
	return intersection
}

// persistDispatchWorkerHandoff persists a worker handoff for a dispatch.
func persistDispatchWorkerHandoff(dispatch codex.WorkerDispatch, result codex.DispatchResult) error {
	if store == nil {
		return nil
	}
	record := buildWorkerHandoffRecord(dispatch, result)
	if strings.TrimSpace(record.WorkerName) == "" {
		return nil
	}
	return store.UpdateFile(workerHandoffsPath, func(existing []byte) ([]byte, error) {
		file := workerHandoffFile{}
		if len(existing) > 0 {
			if err := json.Unmarshal(existing, &file); err != nil {
				var legacy []workerHandoffRecord
				if legacyErr := json.Unmarshal(existing, &legacy); legacyErr != nil {
					return nil, fmt.Errorf("unmarshal worker handoffs: %w", err)
				}
				file.Entries = legacy
			}
		}
		file.Entries = append(file.Entries, record)
		file.Entries = pruneWorkerHandoffRecords(file.Entries, 100)
		data, err := json.MarshalIndent(file, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal worker handoffs: %w", err)
		}
		return append(data, '\n'), nil
	})
}

// renderWorkerHandoffSection renders the handoff context section for a
// worker, under the shared "## Previous Worker Handoffs" heading -- the SAME
// heading the colony-prime capsule (resolveCodexWorkerContext(), which
// unconditionally renders a "build"-workflow-scoped copy whenever a matching
// record exists, cmd/colony_prime_context.go:695) uses.
//
// A caller whose ContextCapsule is resolveCodexWorkerContext() and who ALSO
// needs a DIFFERENT workflow's handoffs delivered (e.g. continue's relay of
// sibling continue-review/watcher handoffs, distinct from the capsule's own
// build-carryover) must call renderRelatedWorkflowHandoffSection instead.
// Calling this function a second time for such a caller would deliver two
// "## Previous Worker Handoffs" sections into the same assembled prompt --
// the exact shape D-190-05-A found on continue's native review/watcher
// dispatch paths, closed by Phase 190 Plan 06.
func renderWorkerHandoffSection(workflow string, phaseID int, workerName string) string {
	return renderHandoffSectionNamed(workflow, phaseID, workerName, "worker_handoffs", "## Previous Worker Handoffs\n\n")
}

// renderRelatedWorkflowHandoffSection renders handoff records for a workflow
// OTHER than the "build" workflow the shared colony-prime capsule always
// carries (cmd/colony_prime_context.go:695) -- e.g. a "continue"-workflow
// handoff left by a sibling continue dispatch (a review caste, or the
// watcher) for the current or preceding phase.
//
// Use this instead of renderWorkerHandoffSection when the caller's
// ContextCapsule is resolveCodexWorkerContext(): that capsule already
// delivers "build"-workflow handoffs under "## Previous Worker Handoffs", so
// calling renderWorkerHandoffSection a second time for a DIFFERENT workflow
// would still collide on the same heading even though the underlying
// records differ (D-190-05-A) -- this repo's own convention treats a
// repeated HEADING as "the same section delivered twice" regardless of
// whether the body content is identical. This function renders under a
// distinct heading instead, so both the cross-phase build-carryover content
// (capsule) and the same-workflow sibling-relay content (this function)
// keep exactly one home each, and neither is silently dropped to zero.
func renderRelatedWorkflowHandoffSection(workflow string, phaseID int, workerName string) string {
	return renderHandoffSectionNamed(workflow, phaseID, workerName, "related_worker_handoffs", "## Related Worker Handoffs\n\nHandoffs from other workers on this same workflow -- distinct from any cross-phase build handoff above.\n\n")
}

// renderHandoffSectionNamed is the shared implementation behind
// renderWorkerHandoffSection and renderRelatedWorkflowHandoffSection. The
// filtering and per-record rendering are identical between the two; only the
// heading (and its section-template identity, for colonies that override
// section headers) differs.
func renderHandoffSectionNamed(workflow string, phaseID int, workerName string, sectionName string, fallbackHeading string) string {
	if store == nil {
		return ""
	}
	records, err := loadWorkerHandoffRecords()
	if err != nil || len(records) == 0 {
		return ""
	}
	workflow = strings.ToLower(strings.TrimSpace(workflow))
	workerName = strings.TrimSpace(workerName)
	filtered := make([]workerHandoffRecord, 0, len(records))
	for _, record := range records {
		if workflow != "" && strings.ToLower(strings.TrimSpace(record.Workflow)) != workflow {
			continue
		}
		if phaseID > 0 && record.Phase > 0 && (record.Phase < phaseID-1 || record.Phase > phaseID) {
			continue
		}
		if workerName != "" && strings.EqualFold(strings.TrimSpace(record.WorkerName), workerName) {
			continue
		}
		filtered = append(filtered, record)
	}
	if len(filtered) == 0 {
		return ""
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		return handoffFreshnessTime(filtered[i].Freshness).After(handoffFreshnessTime(filtered[j].Freshness))
	})
	if len(filtered) > 5 {
		filtered = filtered[:5]
	}

	var b strings.Builder
	writeSectionHeader(&b, sectionName, fallbackHeading)
	for _, record := range filtered {
		title := strings.TrimSpace(record.WorkerName)
		if title == "" {
			title = "worker"
		}
		if record.TaskID != "" {
			title += " (" + record.TaskID + ")"
		}
		b.WriteString(fmtOrFallback(sectionName, func(t *sectionTemplate) string { return t.WorkerHeaderFormat }, "### %s\n", title))
		// Caste, wave and age were stored on every record and rendered on none,
		// so a reader could not tell whether a handoff came from the worker
		// beside it or from a build three days ago — and weighted them equally.
		b.WriteString(fmt.Sprintf("- From: %s\n", handoffProvenance(record)))
		if record.Status != "" || record.VerificationStatus != "" {
			b.WriteString(fmtOrFallback(sectionName, func(t *sectionTemplate) string { return t.StatusFormat }, "- Status: %s; verification: %s\n", firstNonEmpty(record.Status, "unknown"), firstNonEmpty(record.VerificationStatus, "unknown")))
		}
		if record.Summary != "" {
			b.WriteString(fmtOrFallback(sectionName, func(t *sectionTemplate) string { return t.SummaryFormat }, "- Summary: %s\n", record.Summary))
		}
		appendHandoffList(&b, "Changed files", record.ChangedFiles)
		appendHandoffList(&b, "Commands run", record.CommandsRun)
		appendHandoffList(&b, "Known failures", record.KnownFailures)
		appendHandoffList(&b, "Open decisions", record.OpenDecisions)
		appendHandoffList(&b, "Assumptions", record.Assumptions)
		appendHandoffList(&b, "Next worker instructions", record.NextWorkerInstructions)
		appendHandoffList(&b, "Do not repeat", record.DoNotRepeat)
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func buildWorkerHandoffRecord(dispatch codex.WorkerDispatch, result codex.DispatchResult) workerHandoffRecord {
	root := strings.TrimSpace(dispatch.Root)
	if root == "" && store != nil {
		root = filepath.Dir(filepath.Dir(store.BasePath()))
	}
	workerName := strings.TrimSpace(dispatch.WorkerName)
	if workerName == "" {
		workerName = strings.TrimSpace(result.WorkerName)
	}
	status := strings.ToLower(strings.TrimSpace(result.Status))
	summary := ""
	handoff := codex.WorkerHandoff{}
	if result.WorkerResult != nil {
		if workerName == "" {
			workerName = strings.TrimSpace(result.WorkerResult.WorkerName)
		}
		if status == "" {
			status = strings.ToLower(strings.TrimSpace(result.WorkerResult.Status))
		}
		summary = strings.TrimSpace(result.WorkerResult.Summary)
		handoff = result.WorkerResult.Handoff
		// IN-01 (189-REVIEW.md): reuses the single canonical
		// freshness-inclusive emptiness check (pkg/codex) instead of a
		// third hand-copied definition.
		if codex.IsEmptyWorkerHandoffIncludingFreshness(handoff) {
			handoff = codex.WorkerHandoff{
				ChangedFiles:       append(append(append([]string{}, result.WorkerResult.FilesCreated...), result.WorkerResult.FilesModified...), result.WorkerResult.TestsWritten...),
				KnownFailures:      append([]string{}, result.WorkerResult.Blockers...),
				VerificationStatus: verificationStatusForWorkerStatus(status),
			}
		}
	}
	if result.Error != nil {
		handoff.KnownFailures = append(handoff.KnownFailures, result.Error.Error())
		if status == "" {
			status = "failed"
		}
	}
	if status == "" {
		status = "unknown"
	}
	if strings.TrimSpace(handoff.VerificationStatus) == "" {
		handoff.VerificationStatus = verificationStatusForWorkerStatus(status)
	}
	handoff = codex.NormalizeWorkerHandoff(root, handoff)
	if err := codex.ValidateWorkerHandoff(handoff); err != nil {
		handoff.VerificationStatus = "unknown"
		handoff.KnownFailures = append(handoff.KnownFailures, err.Error())
	}
	freshness := strings.TrimSpace(handoff.Freshness)
	if freshness == "" {
		freshness = time.Now().UTC().Format(time.RFC3339)
	}
	return workerHandoffRecord{
		ID:                     fmt.Sprintf("%s:%s:%s:%d", firstNonEmpty(dispatch.Workflow, "worker"), firstNonEmpty(dispatch.TaskID, workerName), workerName, time.Now().UTC().UnixNano()),
		Workflow:               strings.ToLower(strings.TrimSpace(dispatch.Workflow)),
		Phase:                  dispatch.Phase,
		Wave:                   dispatch.Wave,
		WorkerName:             workerName,
		Caste:                  strings.TrimSpace(dispatch.Caste),
		TaskID:                 strings.TrimSpace(dispatch.TaskID),
		Status:                 status,
		Summary:                summary,
		ChangedFiles:           handoff.ChangedFiles,
		CommandsRun:            handoff.CommandsRun,
		VerificationStatus:     handoff.VerificationStatus,
		KnownFailures:          handoff.KnownFailures,
		OpenDecisions:          handoff.OpenDecisions,
		Assumptions:            handoff.Assumptions,
		NextWorkerInstructions: handoff.NextWorkerInstructions,
		DoNotRepeat:            handoff.DoNotRepeat,
		Freshness:              freshness,
	}
}

func loadWorkerHandoffRecords() ([]workerHandoffRecord, error) {
	raw, err := store.ReadFile(workerHandoffsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var file workerHandoffFile
	if err := json.Unmarshal(raw, &file); err == nil {
		return file.Entries, nil
	}
	var legacy []workerHandoffRecord
	if err := json.Unmarshal(raw, &legacy); err != nil {
		return nil, err
	}
	return legacy, nil
}

func pruneWorkerHandoffRecords(records []workerHandoffRecord, limit int) []workerHandoffRecord {
	if limit <= 0 || len(records) <= limit {
		return records
	}
	sort.SliceStable(records, func(i, j int) bool {
		return handoffFreshnessTime(records[i].Freshness).After(handoffFreshnessTime(records[j].Freshness))
	})
	pruned := append([]workerHandoffRecord(nil), records[:limit]...)
	sort.SliceStable(pruned, func(i, j int) bool {
		return handoffFreshnessTime(pruned[i].Freshness).Before(handoffFreshnessTime(pruned[j].Freshness))
	})
	return pruned
}

func verificationStatusForWorkerStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed", "manually-reconciled":
		return "pass"
	case "failed", "blocked", "timeout":
		return "fail"
	case "":
		return "unknown"
	default:
		return "partial"
	}
}

func appendHandoffList(b *strings.Builder, label string, values []string) {
	values = uniqueSortedStrings(values)
	if len(values) == 0 {
		return
	}
	const maxItems = 5
	if len(values) > maxItems {
		values = append(values[:maxItems], fmt.Sprintf("... %d more", len(values)-maxItems))
	}
	fmt.Fprintf(b, "- %s: %s\n", label, strings.Join(values, "; "))
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
			WorkerName:        d.Name,
			Caste:             d.Caste,
			TaskID:            d.TaskID,
			PermissionProfile: d.PermissionProfile,
		}
	}
	return result
}

// handoffFreshnessTime turns a handoff's Freshness into something orderable.
//
// Freshness is normally RFC3339, but pkg/codex/handoff.go deliberately
// preserves the literal "not-run" for a worker whose verification never
// executed. Three sorts compared the field as a raw string, and "not-run"
// collates above every "2026-…" timestamp — so un-run handoffs took the top of
// the five-record window a worker actually reads, and survived pruning ahead of
// real ones. The relay built to stop workers repeating each other was
// preferentially handing them the entries with nothing in them.
//
// An unparsable value sorts as the zero time, i.e. last, which is what
// "we do not know when this happened, or it never ran" should mean.
func handoffFreshnessTime(freshness string) time.Time {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(freshness))
	if err != nil {
		return time.Time{}
	}
	return parsed
}

// handoffProvenance states who wrote a handoff, from where in the build, and
// how long ago — the three fields the record already carried and never showed.
func handoffProvenance(record workerHandoffRecord) string {
	parts := []string{}
	if caste := strings.TrimSpace(record.Caste); caste != "" {
		parts = append(parts, caste)
	}
	if record.Wave > 0 {
		parts = append(parts, fmt.Sprintf("wave %d", record.Wave))
	}
	parts = append(parts, handoffAgePhrase(record.Freshness))
	return strings.Join(parts, " · ")
}

// handoffAgePhrase renders age in the terms a reader judges relevance by.
// An un-run or undated handoff says so plainly rather than being presented
// with the same authority as a fresh one.
func handoffAgePhrase(freshness string) string {
	recorded := handoffFreshnessTime(freshness)
	if recorded.IsZero() {
		if strings.EqualFold(strings.TrimSpace(freshness), "not-run") {
			return "verification never ran"
		}
		return "undated"
	}
	age := time.Since(recorded)
	switch {
	case age < time.Minute:
		return "recorded just now"
	case age < time.Hour:
		return fmt.Sprintf("recorded %dm ago", int(age.Minutes()))
	case age < 24*time.Hour:
		return fmt.Sprintf("recorded %dh ago", int(age.Hours()))
	default:
		return fmt.Sprintf("recorded %dd ago", int(age.Hours()/24))
	}
}
