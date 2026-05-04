package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

const (
	sealFinalReviewReportRel = "seal/final-review.json"
	sealFinalReviewFreshFor  = 24 * time.Hour
)

var sealFinalReviewRequiredCastes = []string{"gatekeeper", "auditor", "probe"}

type sealFinalReviewReport struct {
	Phase          int                           `json:"phase"`
	PhaseName      string                        `json:"phase_name,omitempty"`
	GeneratedAt    string                        `json:"generated_at"`
	ReviewDepth    string                        `json:"review_depth"`
	Source         string                        `json:"source"`
	Reused         bool                          `json:"reused,omitempty"`
	Workers        []codexContinueWorkerFlowStep `json:"workers"`
	Passed         bool                          `json:"passed"`
	BlockingIssues []string                      `json:"blocking_issues,omitempty"`
}

type sealFinalReviewGate struct {
	Report    sealFinalReviewReport
	ReportRel string
	Ran       bool
	Reused    bool
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

func loadFreshSealFinalReview(state colony.ColonyState, now time.Time) (sealFinalReviewReport, string, bool) {
	var own sealFinalReviewReport
	if err := store.LoadJSON(sealFinalReviewReportRel, &own); err == nil && sealFinalReviewIsFresh(own.GeneratedAt, now) && sealFinalReviewSatisfiesGate(own.Passed, own.Workers) {
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
	if review.Phase != finalPhase.ID || !sealFinalReviewIsFresh(review.GeneratedAt, now) || !sealFinalReviewSatisfiesGate(review.Passed, review.Workers) {
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

func sealFinalReviewSatisfiesGate(passed bool, workers []codexContinueWorkerFlowStep) bool {
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
	for _, caste := range sealFinalReviewRequiredCastes {
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

	dispatches := plannedSealFinalReviewDispatches(root, state, phase, invoker, workerTimeout)
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
	report.Passed = sealFinalReviewSatisfiesGate(len(report.BlockingIssues) == 0, report.Workers)
	if !report.Passed && len(report.BlockingIssues) == 0 {
		report.BlockingIssues = []string{"final seal review did not produce completed Gatekeeper, Auditor, and Probe evidence"}
	}
	return report
}

func plannedSealFinalReviewDispatches(root string, state colony.ColonyState, phase colony.Phase, invoker codex.WorkerInvoker, workerTimeout time.Duration) []codex.WorkerDispatch {
	capsule := resolveCodexWorkerContext()
	pheromoneSection := resolvePheromoneSection()
	timeout := effectiveContinueReviewTimeout(workerTimeout)
	dispatches := make([]codex.WorkerDispatch, 0, len(codexContinueReviewSpecs))
	for idx, spec := range codexContinueReviewSpecs {
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
	b.WriteString("\n\nEvidence to inspect:\n")
	b.WriteString("- Colony state: .aether/data/COLONY_STATE.json\n")
	b.WriteString(fmt.Sprintf("- Final phase build manifest, if present: .aether/data/build/phase-%d/manifest.json\n", phase.ID))
	b.WriteString(fmt.Sprintf("- Final phase verification, gates, continue, and review reports, if present: .aether/data/build/phase-%d/\n", phase.ID))
	b.WriteString("- Review ledgers: .aether/data/reviews/\n")
	b.WriteString("- Active constraints/signals from the injected pheromone section\n\n")
	if len(state.Plan.Phases) > 0 {
		b.WriteString("Completed phase summary:\n")
		for _, p := range state.Plan.Phases {
			b.WriteString(fmt.Sprintf("- Phase %d: %s [%s]\n", p.ID, p.Name, p.Status))
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
