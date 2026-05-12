package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

const (
	sealFinalReviewReportRel = "seal/final-review.json"
	sealFinalReviewFreshFor  = 24 * time.Hour
)

var sealFinalReviewRequiredCastes = []string{"gatekeeper", "auditor", "probe"}

// sealReviewDepthForColony selects the default review depth for seal based on
// the phase modes in the colony. If any phase was production, use heavy.
// If all phases were discovery, use light. Otherwise standard.
func sealReviewDepthForColony(state colony.ColonyState) colony.VerificationDepth {
	hasProduction := false
	allDiscovery := len(state.Plan.Phases) > 0
	for _, p := range state.Plan.Phases {
		if p.Mode == colony.PhaseModeProduction {
			hasProduction = true
		}
		if p.Mode != colony.PhaseModeDiscovery {
			allDiscovery = false
		}
	}
	if hasProduction {
		return colony.VerificationDepthHeavy
	}
	if allDiscovery {
		return colony.VerificationDepthLight
	}
	return colony.VerificationDepthStandard
}

// sealReviewRequiredCastes returns the castes required for seal final review
// based on the colony's phase modes.
func sealReviewRequiredCastes(state colony.ColonyState) []string {
	depth := sealReviewDepthForColony(state)
	phase, ok := finalCompletedPhase(state)
	if !ok && len(state.Plan.Phases) > 0 {
		phase = state.Plan.Phases[len(state.Plan.Phases)-1]
	}
	specs := queenSealReviewSpecs(state, phase, depth)
	castes := make([]string, 0, len(specs))
	for _, spec := range specs {
		castes = append(castes, spec.Caste)
	}
	return castes
}

type sealFinalReviewReport struct {
	Phase                 int                           `json:"phase"`
	PhaseName             string                        `json:"phase_name,omitempty"`
	GeneratedAt           string                        `json:"generated_at"`
	ReviewDepth           string                        `json:"review_depth"`
	Source                string                        `json:"source"`
	Reused                bool                          `json:"reused,omitempty"`
	Workers               []codexContinueWorkerFlowStep `json:"workers"`
	Findings              []sealFinalReviewFinding      `json:"findings,omitempty"`
	PostSealBacklog       []sealFinalReviewFinding      `json:"post_seal_backlog,omitempty"`
	ReusableLessons       []string                      `json:"reusable_lessons,omitempty"`
	LedgerWrites          map[string]int                `json:"ledger_writes,omitempty"`
	QueenLearningsWritten int                           `json:"queen_learnings_written,omitempty"`
	QueenLearningWarning  string                        `json:"queen_learning_warning,omitempty"`
	Passed                bool                          `json:"passed"`
	BlockingIssues        []string                      `json:"blocking_issues,omitempty"`
}

type sealFinalReviewFinding struct {
	Domain      string `json:"domain"`
	Severity    string `json:"severity"`
	Agent       string `json:"agent"`
	AgentName   string `json:"agent_name,omitempty"`
	File        string `json:"file,omitempty"`
	Line        int    `json:"line,omitempty"`
	Category    string `json:"category,omitempty"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion,omitempty"`
	Blocking    bool   `json:"blocking,omitempty"`
}

type sealFinalReviewGate struct {
	Report    sealFinalReviewReport
	ReportRel string
	Ran       bool
	Reused    bool
}

type sealPlanManifest struct {
	Workflow                  string                          `json:"workflow"`
	Phase                     int                             `json:"phase"`
	PhaseName                 string                          `json:"phase_name,omitempty"`
	Root                      string                          `json:"root"`
	GeneratedAt               string                          `json:"generated_at"`
	ColonyMode                string                          `json:"colony_mode,omitempty"`
	ReviewDepth               string                          `json:"review_depth"`
	DispatchMode              string                          `json:"dispatch_mode"`
	RequiresFinalizer         bool                            `json:"requires_finalizer"`
	FinalizeSurface           string                          `json:"finalize_surface"`
	FinalizerCommand          string                          `json:"finalizer_command"`
	Force                     bool                            `json:"force,omitempty"`
	WorkerTimeout             int                             `json:"worker_timeout_seconds,omitempty"`
	Dispatches                []codexContinueExternalDispatch `json:"dispatches"`
	DispatchContract          map[string]interface{}          `json:"dispatch_contract,omitempty"`
	PostSealDirectives        []string                        `json:"post_seal_directives,omitempty"`
	BoundaryQuestions         []discussQuestion               `json:"boundary_questions,omitempty"`
	BoundaryQuestionCount     int                             `json:"boundary_question_count,omitempty"`
	BoundaryQuestionsCreated  int                             `json:"boundary_questions_created,omitempty"`
	BoundaryQuestionsExisting int                             `json:"boundary_questions_existing,omitempty"`
	OrchestratorGuidance      *orchestratorBoundaryGuidance   `json:"orchestrator_boundary_guidance,omitempty"`
}

type externalSealCompletion struct {
	SealManifest *sealPlanManifest               `json:"seal_manifest,omitempty"`
	Manifest     *sealPlanManifest               `json:"manifest,omitempty"`
	Dispatches   []codexContinueExternalDispatch `json:"dispatches,omitempty"`
	Results      []codexContinueExternalDispatch `json:"results,omitempty"`
	Workers      []codexContinueExternalDispatch `json:"workers,omitempty"`
}

var sealFinalizeCmd = &cobra.Command{
	Use:   "seal-finalize",
	Short: "Record externally spawned seal review workers and seal the colony",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		completionPath, _ := cmd.Flags().GetString("completion-file")
		completion, err := loadExternalSealCompletion(completionPath)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		if err := runSealFinalize(resolveAetherRootPath(), completion); err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		return nil
	},
}

func loadExternalSealCompletion(path string) (externalSealCompletion, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return externalSealCompletion{}, fmt.Errorf("flag --completion-file is required")
	}
	var data []byte
	var err error
	if path == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return externalSealCompletion{}, fmt.Errorf("read completion file: %w", err)
	}

	var completion externalSealCompletion
	if err := json.Unmarshal(data, &completion); err != nil {
		return externalSealCompletion{}, fmt.Errorf("parse completion file: %w", err)
	}
	if completion.activeManifest() != nil {
		return completion, nil
	}

	var envelope struct {
		Result externalSealCompletion `json:"result"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return externalSealCompletion{}, fmt.Errorf("parse completion envelope: %w", err)
	}
	if envelope.Result.activeManifest() == nil {
		return externalSealCompletion{}, fmt.Errorf("completion file must include seal_manifest")
	}
	return envelope.Result, nil
}

func (c externalSealCompletion) activeManifest() *sealPlanManifest {
	if c.SealManifest != nil {
		return c.SealManifest
	}
	return c.Manifest
}

func (c externalSealCompletion) workerResults() []codexContinueExternalDispatch {
	results := make([]codexContinueExternalDispatch, 0, len(c.Dispatches)+len(c.Results)+len(c.Workers))
	results = append(results, c.Dispatches...)
	results = append(results, c.Results...)
	results = append(results, c.Workers...)
	return results
}

func ensureSealFinalReview(root string, state colony.ColonyState, workerTimeout time.Duration) (sealFinalReviewGate, error) {
	finalPhase, ok := finalCompletedPhase(state)
	if !ok {
		return sealFinalReviewGate{}, fmt.Errorf("no completed final phase found for seal review")
	}

	if report, rel, ok := loadFreshSealFinalReview(state, time.Now().UTC()); ok {
		return sealFinalReviewGate{Report: report, ReportRel: rel, Reused: true}, nil
	}

	report := runSealFinalReview(root, state, finalPhase, workerTimeout)
	if err := store.SaveJSON(sealFinalReviewReportRel, report); err != nil {
		return sealFinalReviewGate{}, fmt.Errorf("failed to write seal final review report: %w", err)
	}
	return sealFinalReviewGate{Report: report, ReportRel: sealFinalReviewReportRel, Ran: true}, nil
}

func finalCompletedPhase(state colony.ColonyState) (colony.Phase, bool) {
	if len(state.Plan.Phases) == 0 {
		return colony.Phase{}, false
	}
	phase := state.Plan.Phases[len(state.Plan.Phases)-1]
	return phase, phase.Status == colony.PhaseCompleted
}

func runSealPlanOnly(root string, force bool) (map[string]interface{}, error) {
	if store == nil {
		return nil, fmt.Errorf("no store initialized")
	}
	state, err := validateSealReady(force)
	if err != nil {
		return nil, err
	}
	phase, ok := finalCompletedPhase(state)
	if !ok {
		return nil, fmt.Errorf("no completed final phase found for seal review")
	}

	now := time.Now().UTC()
	invoker := newCodexWorkerInvoker()
	if invoker == nil {
		invoker = &codex.FakeInvoker{}
	}
	reviewDepth := sealReviewDepthForColony(state)
	requiredCastes := sealReviewRequiredCastes(state)
	dispatches := plannedExternalSealReviewDispatches(root, state, phase, invoker, 0, reviewDepth)
	dispatchMode := "plan-only"
	status := "plan-only"
	if codex.ShouldUseAgentDelegatePath() {
		dispatchMode = "agent-delegate"
		status = "agent-delegate"
	}
	manifest := sealPlanManifest{
		Workflow:          "seal",
		Phase:             phase.ID,
		PhaseName:         phase.Name,
		Root:              root,
		GeneratedAt:       now.Format(time.RFC3339),
		ColonyMode:        string(state.EffectiveColonyMode()),
		ReviewDepth:       string(reviewDepth),
		DispatchMode:      dispatchMode,
		RequiresFinalizer: true,
		FinalizeSurface:   "awaiting_wrapper_completion",
		FinalizerCommand:  "AETHER_OUTPUT_MODE=json aether seal-finalize --completion-file <file>",
		Force:             force,
		WorkerTimeout:     int(effectiveContinueReviewTimeout(0) / time.Second),
		Dispatches:        dispatches,
		DispatchContract: map[string]interface{}{
			"execution_model":        "single final review wave before runtime seal",
			"worker_count":           len(dispatches),
			"required_castes":        append([]string{}, requiredCastes...),
			"state_authority":        "runtime finalizer writes final review, sealed state, Crowned Anthill summary, and post-seal readiness output",
			"wrapper_write_policy":   "workers return structured terminal results to the wrapper; wrappers do not hand-edit .aether/data",
			"worker_status_values":   []string{"completed", "passed", "blocked", "failed", "timeout"},
			"required_result_fields": []string{"name", "caste", "task", "status", "summary"},
			"optional_result_fields": []string{"findings", "issues", "recommendations", "weak_spots", "edge_cases_discovered", "reusable_lessons"},
		},
		PostSealDirectives: []string{
			"After seal-finalize succeeds, follow the runtime's Porter readiness output.",
			"Do not run delivery commands unless the user chooses them.",
		},
	}
	boundary, err := materializeOrchestratorBoundaryQuestions("seal", state, phase, sealBoundaryQuestionCandidates(state, phase, reviewDepth))
	if err != nil {
		return nil, err
	}
	manifest.BoundaryQuestions = boundary.Questions
	manifest.BoundaryQuestionCount = len(boundary.Questions)
	manifest.BoundaryQuestionsCreated = boundary.Created
	manifest.BoundaryQuestionsExisting = boundary.Existing

	result := map[string]interface{}{
		"status":                status,
		"dispatch_mode":         dispatchMode,
		"colony_mode":           string(state.EffectiveColonyMode()),
		"requires_finalizer":    true,
		"execution_owner":       "host-platform",
		"agent_delegate":        dispatchMode == "agent-delegate",
		"agent_delegate_reason": strings.TrimSpace(codex.AgentDelegateFallbackReason()),
		"seal_manifest":         manifest,
		"dispatch_manifest":     manifest,
		"dispatches":            dispatches,
		"workers":               dispatches,
		"dispatch_count":        len(dispatches),
		"worker_count":          len(dispatches),
		"wave_count":            countContinueExternalWaves(dispatches),
		"phase":                 phase.ID,
		"phase_name":            phase.Name,
		"review_depth":          string(reviewDepth),
		"finalizer_command":     manifest.FinalizerCommand,
		"next":                  "spawn seal review workers, then run `aether seal-finalize --completion-file <file>`",
	}
	addBoundaryQuestionResultFields(result, boundary)
	if guidance, ok := addOrchestratorBoundaryGuidance(result, "seal", state, sealAfterDiscussNext(force), boundary.Questions); ok {
		manifest.OrchestratorGuidance = &guidance
		result["seal_manifest"] = manifest
		result["dispatch_manifest"] = manifest
	}
	return result, nil
}

func sealAfterDiscussNext(force bool) string {
	if force {
		return "aether seal --force"
	}
	return "aether seal"
}

func validateSealReady(force bool) (colony.ColonyState, error) {
	state, err := loadActiveColonyState()
	if err != nil {
		return state, fmt.Errorf("%s", colonyStateLoadMessage(err))
	}
	if len(state.Plan.Phases) == 0 {
		return state, fmt.Errorf("No project plan. Run `aether plan` first.")
	}
	for _, phase := range state.Plan.Phases {
		if phase.Status != colony.PhaseCompleted {
			return state, fmt.Errorf("all phases must be completed before sealing the colony")
		}
	}
	blockers, _ := checkSealBlockers(store)
	if len(blockers) > 0 && !force {
		return state, fmt.Errorf("%s", renderBlockerSummary(blockers, nil))
	}
	return state, nil
}

func plannedExternalSealReviewDispatches(root string, state colony.ColonyState, phase colony.Phase, invoker codex.WorkerInvoker, workerTimeout time.Duration, reviewDepth colony.VerificationDepth) []codexContinueExternalDispatch {
	dispatches := plannedSealFinalReviewDispatches(root, state, phase, invoker, workerTimeout, reviewDepth)
	external := make([]codexContinueExternalDispatch, 0, len(dispatches))
	for _, dispatch := range dispatches {
		external = append(external, codexContinueExternalDispatch{
			Stage:        "seal-review",
			Wave:         dispatch.Wave,
			Caste:        dispatch.Caste,
			AgentName:    dispatch.AgentName,
			Name:         dispatch.WorkerName,
			Task:         sealFinalReviewTaskForCaste(dispatch.Caste),
			TaskID:       dispatch.TaskID,
			Timeout:      int(dispatch.Timeout / time.Second),
			Status:       "planned",
			Brief:        dispatch.TaskBrief,
			SkillSection: dispatch.SkillSection,
		})
	}
	return external
}

func runSealFinalize(root string, completion externalSealCompletion) error {
	if store == nil {
		return fmt.Errorf("no store initialized")
	}
	manifest := completion.activeManifest()
	if manifest == nil {
		return fmt.Errorf("completion file must include seal_manifest")
	}
	if (manifest.DispatchMode != "plan-only" && manifest.DispatchMode != "agent-delegate") || !manifest.RequiresFinalizer {
		return fmt.Errorf("seal_manifest must come from `aether seal --plan-only` or an agent-delegate seal response")
	}
	if len(manifest.Dispatches) == 0 {
		return fmt.Errorf("seal_manifest contains no dispatches")
	}
	if err := validateFinalizerManifestRoot("seal_manifest", manifest.Root, root); err != nil {
		return err
	}

	state, err := validateSealReady(manifest.Force)
	if err != nil {
		return err
	}
	if err := validateFinalizerManifestColonyMode("seal_manifest", manifest.ColonyMode, state); err != nil {
		return err
	}
	phase, ok := finalCompletedPhase(state)
	if !ok {
		return fmt.Errorf("no completed final phase found for seal review")
	}
	if manifest.Phase != phase.ID {
		return fmt.Errorf("seal_manifest phase = %d, current final phase = %d", manifest.Phase, phase.ID)
	}
	if err := unresolvedOrchestratorBoundaryGuidanceError("seal", state, sealAfterDiscussNext(manifest.Force), manifest.BoundaryQuestions); err != nil {
		return err
	}

	flow, err := mergeExternalSealReviewResults(*manifest, completion.workerResults())
	if err != nil {
		return err
	}
	findings := sealFinalReviewFindings(flow)
	ledgerWrites, err := persistSealFinalReviewFindings(phase.ID, phase.Name, findings)
	if err != nil {
		return err
	}
	reusableLessons := sealFinalReviewReusableLessons(flow, findings)
	queenLearningsWritten, queenLearningWarning := writeSealReusableLessonsToQueen(phase.ID, reusableLessons)
	report := sealFinalReviewReport{
		Phase:                 phase.ID,
		PhaseName:             phase.Name,
		GeneratedAt:           time.Now().UTC().Format(time.RFC3339),
		ReviewDepth:           string(colony.VerificationDepthHeavy),
		Source:                "seal-finalize",
		Workers:               flow,
		Findings:              findings,
		PostSealBacklog:       sealFinalReviewBacklog(findings),
		ReusableLessons:       reusableLessons,
		LedgerWrites:          ledgerWrites,
		QueenLearningsWritten: queenLearningsWritten,
		QueenLearningWarning:  queenLearningWarning,
	}
	blockers := append(sealReviewBlockingIssues(flow), sealReviewFindingBlockingIssues(findings)...)
	report.BlockingIssues = uniqueSortedStrings(blockers)
	requiredCastes := sealReviewRequiredCastes(state)
	report.Passed = sealFinalReviewSatisfiesGate(len(report.BlockingIssues) == 0, report.Workers, requiredCastes)
	if !report.Passed && len(report.BlockingIssues) == 0 {
		report.BlockingIssues = []string{"final seal review did not produce completed required review evidence"}
	}
	if err := store.SaveJSON(sealFinalReviewReportRel, report); err != nil {
		return fmt.Errorf("failed to write seal final review report: %w", err)
	}
	if !report.Passed && !manifest.Force {
		return fmt.Errorf("%s", renderSealFinalReviewBlockers(sealFinalReviewGate{Report: report, ReportRel: sealFinalReviewReportRel, Ran: true}))
	}
	return completeSealRuntime(state)
}

func mergeExternalSealReviewResults(manifest sealPlanManifest, results []codexContinueExternalDispatch) ([]codexContinueWorkerFlowStep, error) {
	return mergeExternalContinueResults(codexContinuePlanManifest{
		Phase:             manifest.Phase,
		PhaseName:         manifest.PhaseName,
		Root:              manifest.Root,
		GeneratedAt:       manifest.GeneratedAt,
		ColonyMode:        manifest.ColonyMode,
		Dispatches:        manifest.Dispatches,
		DispatchMode:      manifest.DispatchMode,
		RequiresFinalizer: manifest.RequiresFinalizer,
		ReviewDepth:       manifest.ReviewDepth,
	}, results)
}

func sealReviewBlockingIssues(flow []codexContinueWorkerFlowStep) []string {
	blockers := []string{}
	for _, step := range flow {
		status := normalizeRuntimeDispatchStatus(step.Status)
		for _, blocker := range step.Blockers {
			if strings.TrimSpace(blocker) != "" {
				blockers = append(blockers, fmt.Sprintf("%s reported blocker: %s", step.Name, blocker))
			}
		}
		if status != "completed" {
			summary := strings.TrimSpace(step.Summary)
			if summary == "" {
				summary = status
			}
			blockers = append(blockers, fmt.Sprintf("%s final review did not complete cleanly: %s", step.Name, summary))
		}
	}
	return blockers
}

func sealFinalReviewFindings(flow []codexContinueWorkerFlowStep) []sealFinalReviewFinding {
	findings := []sealFinalReviewFinding{}
	seen := map[string]bool{}
	appendFinding := func(f sealFinalReviewFinding) {
		f.Domain = sealReviewDomainForCaste(f.Agent, f.Domain)
		f.Severity = normalizeSealReviewSeverity(f.Severity)
		f.Agent = strings.TrimSpace(strings.ToLower(f.Agent))
		f.AgentName = strings.TrimSpace(f.AgentName)
		f.File = strings.TrimSpace(f.File)
		f.Category = strings.TrimSpace(f.Category)
		f.Description = strings.TrimSpace(f.Description)
		f.Suggestion = strings.TrimSpace(f.Suggestion)
		if f.Description == "" {
			return
		}
		key := strings.Join([]string{
			f.Domain,
			f.Severity,
			f.Agent,
			f.AgentName,
			f.File,
			fmt.Sprintf("%d", f.Line),
			f.Category,
			f.Description,
			f.Suggestion,
			fmt.Sprintf("%t", f.Blocking),
		}, "\x00")
		if seen[key] {
			return
		}
		seen[key] = true
		findings = append(findings, f)
	}

	for _, step := range flow {
		agent := strings.TrimSpace(strings.ToLower(step.Caste))
		agentName := strings.TrimSpace(step.Name)
		for _, blocker := range step.Blockers {
			appendFinding(sealFinalReviewFinding{
				Domain:      sealReviewDomainForCaste(agent, ""),
				Severity:    "HIGH",
				Agent:       agent,
				AgentName:   agentName,
				Category:    "blocker",
				Description: blocker,
				Suggestion:  "Resolve before sealing the colony, or rerun seal with --force only if the risk is intentionally accepted.",
				Blocking:    true,
			})
		}
		for _, raw := range step.Findings {
			description := strings.TrimSpace(raw.Description)
			if description == "" {
				description = strings.TrimSpace(raw.Title)
			}
			appendFinding(sealFinalReviewFinding{
				Domain:      raw.Domain,
				Severity:    raw.Severity,
				Agent:       agent,
				AgentName:   agentName,
				File:        raw.File,
				Line:        raw.Line,
				Category:    raw.Category,
				Description: description,
				Suggestion:  raw.Suggestion,
				Blocking:    raw.Blocking,
			})
		}
		for _, recommendation := range step.Recommendations {
			appendFinding(sealFinalReviewFinding{
				Domain:      sealReviewDomainForCaste(agent, ""),
				Severity:    "INFO",
				Agent:       agent,
				AgentName:   agentName,
				Category:    "recommendation",
				Description: recommendation,
			})
		}
		for _, weakSpot := range step.WeakSpots {
			appendFinding(sealFinalReviewFinding{
				Domain:      sealReviewDomainForCaste(agent, "testing"),
				Severity:    "MEDIUM",
				Agent:       agent,
				AgentName:   agentName,
				Category:    "coverage-gap",
				Description: weakSpot,
				Suggestion:  "Add focused coverage before expanding the sealed surface.",
			})
		}
		for _, edgeCase := range step.EdgeCases {
			appendFinding(sealFinalReviewFinding{
				Domain:      sealReviewDomainForCaste(agent, "testing"),
				Severity:    "INFO",
				Agent:       agent,
				AgentName:   agentName,
				Category:    "edge-case",
				Description: edgeCase,
			})
		}
	}
	return findings
}

func sealReviewDomainForCaste(caste, requested string) string {
	caste = strings.TrimSpace(strings.ToLower(caste))
	requested = strings.TrimSpace(strings.ToLower(requested))
	if requested != "" && validDomains[requested] && sealReviewCasteAllowsDomain(caste, requested) {
		return requested
	}
	switch caste {
	case "gatekeeper":
		return "security"
	case "probe":
		return "testing"
	case "watcher":
		return "testing"
	case "auditor":
		return "quality"
	case "measurer":
		return "performance"
	case "chaos":
		return "resilience"
	case "tracker":
		return "bugs"
	case "archaeologist":
		return "history"
	default:
		return "quality"
	}
}

func sealReviewCasteAllowsDomain(caste, domain string) bool {
	if caste == "probe" {
		return domain == "testing"
	}
	if allowed, ok := agentAllowedDomains[caste]; ok {
		for _, candidate := range allowed {
			if candidate == domain {
				return true
			}
		}
		return false
	}
	return validDomains[domain]
}

func normalizeSealReviewSeverity(severity string) string {
	switch strings.ToUpper(strings.TrimSpace(severity)) {
	case "CRITICAL":
		return "CRITICAL"
	case "HIGH":
		return "HIGH"
	case "MEDIUM":
		return "MEDIUM"
	case "LOW":
		return "LOW"
	default:
		return "INFO"
	}
}

func sealReviewFindingBlockingIssues(findings []sealFinalReviewFinding) []string {
	var blockers []string
	for _, finding := range findings {
		if !finding.Blocking && finding.Severity != "CRITICAL" {
			continue
		}
		label := finding.AgentName
		if label == "" {
			label = finding.Agent
		}
		blockers = append(blockers, fmt.Sprintf("%s reported %s %s finding: %s", label, finding.Severity, finding.Domain, finding.Description))
	}
	return blockers
}

func sealFinalReviewBacklog(findings []sealFinalReviewFinding) []sealFinalReviewFinding {
	backlog := []sealFinalReviewFinding{}
	for _, finding := range findings {
		if finding.Blocking || finding.Severity == "CRITICAL" {
			continue
		}
		backlog = append(backlog, finding)
	}
	return backlog
}

func persistSealFinalReviewFindings(phase int, phaseName string, findings []sealFinalReviewFinding) (map[string]int, error) {
	writes := map[string]int{}
	for _, finding := range findings {
		if strings.TrimSpace(finding.Description) == "" {
			continue
		}
		domain := sealReviewDomainForCaste(finding.Agent, finding.Domain)
		prefix := domainPrefixes[domain]
		ledgerPath := fmt.Sprintf("reviews/%s/ledger.json", domain)
		var lf colony.ReviewLedgerFile
		if err := store.LoadJSON(ledgerPath, &lf); err != nil {
			lf = colony.ReviewLedgerFile{Entries: []colony.ReviewLedgerEntry{}}
		}
		if lf.Entries == nil {
			lf.Entries = []colony.ReviewLedgerEntry{}
		}
		if reviewLedgerHasEquivalentFinding(lf.Entries, phase, finding) {
			continue
		}
		idx := colony.NextEntryIndex(lf.Entries, prefix, phase)
		entry := colony.ReviewLedgerEntry{
			ID:          colony.FormatEntryID(prefix, phase, idx),
			Phase:       phase,
			PhaseName:   phaseName,
			Agent:       finding.Agent,
			AgentName:   finding.AgentName,
			GeneratedAt: time.Now().UTC().Format(time.RFC3339),
			Status:      "open",
			Severity:    sealReviewLedgerSeverity(finding.Severity),
			File:        finding.File,
			Line:        finding.Line,
			Category:    finding.Category,
			Description: finding.Description,
			Suggestion:  finding.Suggestion,
		}
		lf.Entries = append(lf.Entries, entry)
		lf.Summary = colony.ComputeSummary(lf.Entries)
		if err := store.SaveJSON(ledgerPath, lf); err != nil {
			return writes, fmt.Errorf("failed to save %s review ledger: %w", domain, err)
		}
		writes[domain]++
	}
	return writes, nil
}

func reviewLedgerHasEquivalentFinding(entries []colony.ReviewLedgerEntry, phase int, finding sealFinalReviewFinding) bool {
	for _, entry := range entries {
		if entry.Phase != phase {
			continue
		}
		if strings.EqualFold(entry.Agent, finding.Agent) &&
			strings.EqualFold(entry.AgentName, finding.AgentName) &&
			entry.File == finding.File &&
			entry.Line == finding.Line &&
			strings.EqualFold(entry.Category, finding.Category) &&
			strings.EqualFold(entry.Description, finding.Description) {
			return true
		}
	}
	return false
}

func sealReviewLedgerSeverity(severity string) colony.ReviewSeverity {
	switch normalizeSealReviewSeverity(severity) {
	case "CRITICAL", "HIGH":
		return colony.ReviewSeverityHigh
	case "MEDIUM":
		return colony.ReviewSeverityMedium
	case "LOW":
		return colony.ReviewSeverityLow
	default:
		return colony.ReviewSeverityInfo
	}
}

func sealFinalReviewReusableLessons(flow []codexContinueWorkerFlowStep, findings []sealFinalReviewFinding) []string {
	lessons := []string{}
	for _, step := range flow {
		lessons = append(lessons, step.ReusableLessons...)
	}
	for _, finding := range findings {
		category := strings.ToLower(strings.TrimSpace(finding.Category))
		if category == "lesson" || category == "learning" || category == "reusable-lesson" {
			lessons = append(lessons, finding.Description)
		}
	}
	return uniqueSortedStrings(lessons)
}

func writeSealReusableLessonsToQueen(phase int, lessons []string) (int, string) {
	if len(lessons) == 0 {
		return 0, ""
	}
	text, err := loadLocalQueenText()
	if err != nil {
		return 0, err.Error()
	}
	entries := []string{}
	for _, lesson := range lessons {
		lesson = sanitizeQueenInline(lesson)
		if lesson == "" {
			continue
		}
		entry := fmt.Sprintf("- %s (seal review phase %d, %s)", lesson, phase, time.Now().UTC().Format("2006-01-02"))
		if !isEntryInText(text, entry) {
			entries = append(entries, entry)
		}
	}
	if len(entries) == 0 {
		return 0, ""
	}
	text = appendEntriesToQueenSection(text, "Wisdom", entries)
	if err := writeLocalQueenText(text); err != nil {
		return 0, err.Error()
	}
	return len(entries), ""
}

func loadFreshSealFinalReview(state colony.ColonyState, now time.Time) (sealFinalReviewReport, string, bool) {
	requiredCastes := sealReviewRequiredCastes(state)
	var own sealFinalReviewReport
	if err := store.LoadJSON(sealFinalReviewReportRel, &own); err == nil && sealFinalReviewIsFresh(own.GeneratedAt, now) && sealFinalReviewSatisfiesGate(own.Passed, own.Workers, requiredCastes) {
		own.Reused = true
		return own, sealFinalReviewReportRel, true
	}

	finalPhase, ok := finalCompletedPhase(state)
	if !ok {
		return sealFinalReviewReport{}, "", false
	}
	continueReviewRel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", finalPhase.ID), "review.json"))
	var review codexContinueReviewReport
	if err := store.LoadJSON(continueReviewRel, &review); err != nil {
		return sealFinalReviewReport{}, "", false
	}
	if review.Phase != finalPhase.ID || !sealFinalReviewIsFresh(review.GeneratedAt, now) || !sealFinalReviewSatisfiesGate(review.Passed, review.Workers, requiredCastes) {
		return sealFinalReviewReport{}, "", false
	}
	return sealFinalReviewReport{
		Phase:          finalPhase.ID,
		PhaseName:      finalPhase.Name,
		GeneratedAt:    review.GeneratedAt,
		ReviewDepth:    string(colony.VerificationDepthHeavy),
		Source:         "continue-final-review",
		Reused:         true,
		Workers:        review.Workers,
		Passed:         review.Passed,
		BlockingIssues: review.BlockingIssues,
	}, continueReviewRel, true
}

func sealFinalReviewIsFresh(generatedAt string, now time.Time) bool {
	generatedAt = strings.TrimSpace(generatedAt)
	if generatedAt == "" {
		return false
	}
	parsed, err := time.Parse(time.RFC3339, generatedAt)
	if err != nil {
		return false
	}
	if parsed.After(now.Add(1 * time.Minute)) {
		return false
	}
	return now.Sub(parsed) <= sealFinalReviewFreshFor
}

func sealFinalReviewSatisfiesGate(passed bool, workers []codexContinueWorkerFlowStep, requiredCastes []string) bool {
	if !passed {
		return false
	}
	seen := map[string]bool{}
	for _, worker := range workers {
		caste := strings.ToLower(strings.TrimSpace(worker.Caste))
		if caste == "" {
			continue
		}
		if normalizeRuntimeDispatchStatus(worker.Status) == "completed" {
			seen[caste] = true
		}
	}
	for _, caste := range requiredCastes {
		if !seen[caste] {
			return false
		}
	}
	return true
}

func runSealFinalReview(root string, state colony.ColonyState, phase colony.Phase, workerTimeout time.Duration) sealFinalReviewReport {
	now := time.Now().UTC()
	report := sealFinalReviewReport{
		Phase:       phase.ID,
		PhaseName:   phase.Name,
		GeneratedAt: now.Format(time.RFC3339),
		ReviewDepth: string(colony.VerificationDepthHeavy),
		Source:      "seal-final-review",
		Workers:     []codexContinueWorkerFlowStep{},
		Passed:      false,
	}

	invoker := newCodexWorkerInvoker()
	if invoker == nil {
		report.BlockingIssues = []string{"final seal review cannot run because no worker invoker is configured"}
		return report
	}
	if _, ok := invoker.(*codex.FakeInvoker); !ok && !invoker.IsAvailable(context.Background()) {
		report.BlockingIssues = []string{fmt.Sprintf("final seal review cannot run because %s", dispatchAvailabilityMessage(invoker))}
		return report
	}

	reviewDepth := sealReviewDepthForColony(state)
	dispatches := plannedSealFinalReviewDispatches(root, state, phase, invoker, workerTimeout, reviewDepth)
	report.ReviewDepth = string(reviewDepth)
	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	reviewCtx, cancel := context.WithTimeout(context.Background(), effectiveContinueReviewTimeout(workerTimeout))
	defer cancel()

	results, err := dispatchBatchByWaveWithVisuals(
		reviewCtx,
		invoker,
		dispatches,
		colony.ModeInRepo,
		"Seal Final Review",
		true,
		func(wave int) codex.DispatchObserver {
			return runtimeVisualDispatchObserver(spawnTree, "Seal final review active", wave)
		},
	)

	blockers := []string{}
	if err != nil {
		blockers = append(blockers, err.Error())
	}

	flow := make([]codexContinueWorkerFlowStep, 0, len(dispatches))
	for i, dispatch := range dispatches {
		step := codexContinueWorkerFlowStep{
			Stage:  "seal-review",
			Caste:  dispatch.Caste,
			Name:   dispatch.WorkerName,
			Task:   sealFinalReviewTaskForCaste(dispatch.Caste),
			Status: "failed",
		}
		if i < len(results) {
			result := results[i]
			step.Name = result.WorkerName
			step.Status = normalizeRuntimeDispatchStatus(result.Status)
			if result.WorkerResult != nil {
				if len(result.WorkerResult.Blockers) > 0 {
					step.Summary = strings.Join(result.WorkerResult.Blockers, "; ")
				} else if summary := strings.TrimSpace(result.WorkerResult.Summary); summary != "" && !strings.HasPrefix(summary, "FakeInvoker completed task") {
					step.Summary = summary
				}
				step.Blockers = uniqueSortedStrings(result.WorkerResult.Blockers)
				step.Duration = result.WorkerResult.Duration.Seconds()
				step.Report = strings.TrimSpace(result.WorkerResult.RawOutput)
				for _, blocker := range result.WorkerResult.Blockers {
					if strings.TrimSpace(blocker) != "" {
						blockers = append(blockers, fmt.Sprintf("%s reported blocker: %s", result.WorkerName, blocker))
					}
				}
			}
			if step.Summary == "" && result.Error != nil {
				step.Summary = strings.TrimSpace(result.Error.Error())
			}
			if step.Summary == "" {
				step.Summary = sealFinalReviewFlowSummary(step)
			}
			if step.Status != "completed" {
				blockers = append(blockers, fmt.Sprintf("%s final review did not complete cleanly: %s", result.WorkerName, step.Status))
			}
		} else {
			blockers = append(blockers, fmt.Sprintf("%s final review did not return a result", dispatch.WorkerName))
		}
		flow = append(flow, step)
	}

	report.Workers = flow
	report.BlockingIssues = uniqueSortedStrings(blockers)
	requiredCastes := sealReviewRequiredCastes(state)
	report.Passed = sealFinalReviewSatisfiesGate(len(report.BlockingIssues) == 0, report.Workers, requiredCastes)
	if !report.Passed && len(report.BlockingIssues) == 0 {
		report.BlockingIssues = []string{"final seal review did not produce completed required review evidence"}
	}
	return report
}

func plannedSealFinalReviewDispatches(root string, state colony.ColonyState, phase colony.Phase, invoker codex.WorkerInvoker, workerTimeout time.Duration, reviewDepth colony.VerificationDepth) []codex.WorkerDispatch {
	capsule := resolveCodexWorkerContext()
	pheromoneSection := resolvePheromoneSection()
	timeout := effectiveContinueReviewTimeout(workerTimeout)
	specs := queenSealReviewSpecs(state, phase, reviewDepth)
	dispatches := make([]codex.WorkerDispatch, 0, len(specs))
	for idx, spec := range specs {
		agentName := codexAgentNameForCaste(spec.Caste)
		dispatches = append(dispatches, codex.WorkerDispatch{
			ID:               fmt.Sprintf("seal-review-%d", idx),
			WorkerName:       deterministicAntName(spec.Caste, fmt.Sprintf("seal:%d:%s", phase.ID, spec.Caste)),
			AgentName:        agentName,
			AgentTOMLPath:    dispatchAgentPath(root, invoker, agentName),
			Caste:            spec.Caste,
			TaskID:           fmt.Sprintf("seal-review-%s", spec.Caste),
			TaskBrief:        renderSealFinalReviewBrief(root, state, phase, spec),
			ContextCapsule:   capsule,
			HandoffSection:   renderWorkerHandoffSection("seal", phase.ID, deterministicAntName(spec.Caste, fmt.Sprintf("seal:%d:%s", phase.ID, spec.Caste))),
			Workflow:         "seal",
			Phase:            phase.ID,
			SkillSection:     resolveSkillSectionForWorkflow("seal", spec.Caste, spec.Task),
			PheromoneSection: pheromoneSection,
			Root:             root,
			Timeout:          timeout,
			Wave:             1,
		})
	}
	return dispatches
}

func queenSealReviewSpecs(state colony.ColonyState, phase colony.Phase, reviewDepth colony.VerificationDepth) []codexContinueReviewSpec {
	queenState := state
	queenState.VerificationDepth = string(reviewDepth)
	selected := queenOrchestrate(phase, "seal", queenState)
	specs := make([]codexContinueReviewSpec, 0, len(selected))
	for _, dispatch := range selected {
		spec, ok := sealFinalReviewSpecForCaste(dispatch.Caste)
		if !ok {
			continue
		}
		specs = append(specs, spec)
	}
	return specs
}

func sealFinalReviewSpecForCaste(caste string) (codexContinueReviewSpec, bool) {
	if spec, ok := continueReviewSpecForCaste(caste); ok {
		return spec, true
	}
	switch caste {
	case "porter":
		return codexContinueReviewSpec{Caste: "porter", Task: "Review packaging, delivery, and post-seal release readiness. Return blocked only for concrete delivery risks that make sealing unsafe."}, true
	case "chronicler":
		return codexContinueReviewSpec{Caste: "chronicler", Task: "Review final documentation, changelog, and Crowned Anthill evidence completeness before seal."}, true
	}
	return codexContinueReviewSpec{}, false
}

func renderSealFinalReviewBrief(root string, state colony.ColonyState, phase colony.Phase, spec codexContinueReviewSpec) string {
	var b strings.Builder
	b.WriteString("# Seal Final Review\n\n")
	if state.Goal != nil {
		b.WriteString("- Goal: ")
		b.WriteString(strings.TrimSpace(*state.Goal))
		b.WriteString("\n")
	}
	b.WriteString("- Repo: ")
	b.WriteString(root)
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("- Final phase: %d — %s\n", phase.ID, phase.Name))
	b.WriteString("- Role: ")
	b.WriteString(spec.Caste)
	b.WriteString("\n\n")
	b.WriteString("This is the final review before `aether seal` closes the colony. Do not modify repo source files. Return status `blocked` if the colony is not safe to seal.\n\n")
	b.WriteString(spec.Task)
	if strings.EqualFold(strings.TrimSpace(spec.Caste), "probe") {
		b.WriteString("\n\nCoverage guidance: if runtime verification checks passed, package-wide line coverage below an aspirational threshold is advisory by itself. Block only for red verification commands, missing focused regression coverage for changed behavior, or concrete unexercised edge cases that make sealing unsafe.")
	}
	b.WriteString("\n\nEvidence to inspect:\n")
	b.WriteString("- Colony state: .aether/data/COLONY_STATE.json\n")
	b.WriteString(fmt.Sprintf("- Final phase build manifest, if present: .aether/data/build/phase-%d/manifest.json\n", phase.ID))
	b.WriteString(fmt.Sprintf("- Final phase verification, gates, continue, and review reports, if present: .aether/data/build/phase-%d/\n", phase.ID))
	b.WriteString("- Review ledgers: .aether/data/reviews/\n")
	b.WriteString("- Active constraints/signals from the injected pheromone section\n\n")
	b.WriteString("Return structured findings when you discover useful release evidence. Use `findings` or `issues` with objects shaped as `{domain,severity,file,line,category,description,suggestion,blocking}`. Use `blocking:true` or a CRITICAL severity only for issues that must stop sealing. Put durable process lessons in `reusable_lessons` so the runtime can promote them into repo-local QUEEN.md.\n\n")
	if len(state.Plan.Phases) > 0 {
		b.WriteString("Completed phase summary:\n")
		if len(state.Plan.Phases) > 5 {
			b.WriteString(fmt.Sprintf("- %d phases total (see COLONY_STATE.json for full list)\n", len(state.Plan.Phases)))
		} else {
			for _, p := range state.Plan.Phases {
				b.WriteString(fmt.Sprintf("- Phase %d: %s [%s]\n", p.ID, p.Name, p.Status))
			}
		}
		b.WriteString("\n")
	}
	b.WriteString("Seal only clears if your review finds no unresolved blocker-level issue.\n")
	return b.String()
}

func sealFinalReviewTaskForCaste(caste string) string {
	switch strings.ToLower(strings.TrimSpace(caste)) {
	case "gatekeeper":
		return "Security and release integrity review before seal"
	case "auditor":
		return "Quality and completeness audit before seal"
	case "probe":
		return "Coverage and verification evidence review before seal"
	case "measurer":
		return "Performance and cost evidence review before seal"
	case "includer":
		return "Accessibility and inclusive-use review before seal"
	case "porter":
		return "Delivery readiness review before seal"
	case "chronicler":
		return "Documentation evidence review before seal"
	default:
		return "Final seal review"
	}
}

func sealFinalReviewFlowSummary(step codexContinueWorkerFlowStep) string {
	if strings.TrimSpace(step.Summary) != "" {
		return strings.TrimSpace(step.Summary)
	}
	status := strings.TrimSpace(step.Status)
	if status == "" {
		status = "unknown"
	}
	return fmt.Sprintf("%s %s completed seal final review with status %s", strings.Title(strings.TrimSpace(step.Caste)), strings.TrimSpace(step.Name), status)
}

func sealFinalReviewResultMap(gate sealFinalReviewGate) map[string]interface{} {
	return map[string]interface{}{
		"required":        true,
		"passed":          gate.Report.Passed,
		"ran":             gate.Ran,
		"reused":          gate.Reused || gate.Report.Reused,
		"source":          gate.Report.Source,
		"report":          displayDataPath(gate.ReportRel),
		"worker_count":    len(gate.Report.Workers),
		"blocking_issues": gate.Report.BlockingIssues,
	}
}

func renderSealFinalReviewBlockers(gate sealFinalReviewGate) string {
	issues := uniqueSortedStrings(gate.Report.BlockingIssues)
	if len(issues) == 0 {
		issues = []string{"final seal review did not pass"}
	}
	var b strings.Builder
	b.WriteString("BLOCKED: final seal review did not clear.\n")
	b.WriteString("Report: ")
	b.WriteString(displayDataPath(gate.ReportRel))
	b.WriteString("\n")
	for _, issue := range issues {
		b.WriteString("- ")
		b.WriteString(issue)
		b.WriteString("\n")
	}
	b.WriteString("Resolve the blockers and rerun `aether seal`, or rerun with `aether seal --force` only if you intentionally accept the risk.")
	return b.String()
}

func renderSealPlanOnlyVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("seal"), "Seal Review Dispatch"))
	b.WriteString("Seal final review manifest ready.\n")
	if phase := intValue(result["phase"]); phase > 0 {
		b.WriteString(fmt.Sprintf("Final phase: %d", phase))
		if name := strings.TrimSpace(stringValue(result["phase_name"])); name != "" {
			b.WriteString(" — " + name)
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	if dispatches, ok := result["dispatches"].([]codexContinueExternalDispatch); ok && len(dispatches) > 0 {
		b.WriteString("Planned Seal Review Workers\n")
		lastWave := 0
		for _, dispatch := range dispatches {
			if dispatch.Wave != lastWave {
				if lastWave > 0 {
					b.WriteString("\n")
				}
				b.WriteString(fmt.Sprintf("Wave %d\n", dispatch.Wave))
				lastWave = dispatch.Wave
			}
			b.WriteString("  ")
			b.WriteString(casteIdentity(dispatch.Caste))
			b.WriteString(" ")
			b.WriteString(dispatch.Name)
			b.WriteString("  ")
			b.WriteString(strings.TrimSpace(dispatch.Task))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	finalizer := strings.TrimSpace(stringValue(result["finalizer_command"]))
	if finalizer == "" {
		finalizer = "AETHER_OUTPUT_MODE=json aether seal-finalize --completion-file <file>"
	}
	b.WriteString(renderNextUp(
		"Dispatch the final review workers through the host platform.",
		"Then run `"+finalizer+"`.",
	))
	return b.String()
}
