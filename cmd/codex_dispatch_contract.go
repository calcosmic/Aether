package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

// renderWorkerHandoffSection renders the handoff context section for a worker.
func renderWorkerHandoffSection(workflow string, phaseID int, workerName string) string {
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
		return filtered[i].Freshness > filtered[j].Freshness
	})
	if len(filtered) > 5 {
		filtered = filtered[:5]
	}

	var b strings.Builder
	b.WriteString("## Previous Worker Handoffs\n\n")
	for _, record := range filtered {
		title := strings.TrimSpace(record.WorkerName)
		if title == "" {
			title = "worker"
		}
		if record.TaskID != "" {
			title += " (" + record.TaskID + ")"
		}
		fmt.Fprintf(&b, "### %s\n", title)
		if record.Status != "" || record.VerificationStatus != "" {
			fmt.Fprintf(&b, "- Status: %s; verification: %s\n", firstNonEmpty(record.Status, "unknown"), firstNonEmpty(record.VerificationStatus, "unknown"))
		}
		if record.Summary != "" {
			fmt.Fprintf(&b, "- Summary: %s\n", record.Summary)
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
		if workerHandoffEmpty(handoff) {
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
		return records[i].Freshness > records[j].Freshness
	})
	pruned := append([]workerHandoffRecord(nil), records[:limit]...)
	sort.SliceStable(pruned, func(i, j int) bool {
		return pruned[i].Freshness < pruned[j].Freshness
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

func workerHandoffEmpty(h codex.WorkerHandoff) bool {
	return len(h.ChangedFiles) == 0 &&
		len(h.CommandsRun) == 0 &&
		strings.TrimSpace(h.VerificationStatus) == "" &&
		len(h.KnownFailures) == 0 &&
		len(h.OpenDecisions) == 0 &&
		len(h.Assumptions) == 0 &&
		len(h.NextWorkerInstructions) == 0 &&
		len(h.DoNotRepeat) == 0 &&
		strings.TrimSpace(h.Freshness) == ""
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
			WorkerName: d.Name,
			Caste:      d.Caste,
			TaskID:     d.TaskID,
		}
	}
	return result
}
