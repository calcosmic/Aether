package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

func reviewerArtifactRaw(t *testing.T, value string) map[string]json.RawMessage {
	t.Helper()
	return map[string]json.RawMessage{"review": json.RawMessage(value)}
}

func completedReviewerHandoff() codex.WorkerHandoff {
	return codex.WorkerHandoff{
		CommandsRun:            []string{"go test ./cmd/ -run TestReviewerArtifact"},
		VerificationStatus:     "pass",
		NextWorkerInstructions: []string{"Use the structured review artifact."},
		Freshness:              "not-run",
	}
}

func completedReviewerStep(caste, name string) codexContinueWorkerFlowStep {
	return codexContinueWorkerFlowStep{
		Stage: "review", Caste: caste, Name: name, Status: "completed",
	}
}

func validCompletedReviewerArtifacts(t *testing.T, caste string) map[string]json.RawMessage {
	t.Helper()
	if !strings.EqualFold(strings.TrimSpace(caste), "auditor") {
		return nil
	}
	return reviewerArtifactRaw(t, `{"overall_score":60,"findings":[]}`)
}

func reviewerEvaluation(t *testing.T, signals codexContinueAutopilotSignals, code autopilotTriggerCode) autopilotTriggerEvaluation {
	t.Helper()
	for _, evaluation := range signals.Evaluations {
		if evaluation.Spec.Code == code {
			return evaluation
		}
	}
	t.Fatalf("autopilot signal %q was not projected: %#v", code, signals)
	return autopilotTriggerEvaluation{}
}

func TestReviewerArtifactNormalizesDirectAndExternalLanes(t *testing.T) {
	raw := reviewerArtifactRaw(t, `{
		"overall_score": 59,
		"findings": [
			{"domain":"security", "severity":"critical", "title":"Token leak", "description":"token is logged", "suggestion":"redact it"},
			{"domain":"quality", "severity":"HIGH", "description":"large function", "blocking":true}
		]
	}`)

	direct := normalizeContinueReviewEvidence(codexContinueWorkerFlowStep{
		Stage: "review", Caste: "auditor", Name: "Audit-Direct", Status: "completed",
	}, raw)
	if direct.OverallScore == nil || *direct.OverallScore != 59 {
		t.Fatalf("direct lane lost the Auditor score: %#v", direct.OverallScore)
	}
	if len(direct.EvidenceErrors) != 0 {
		t.Fatalf("valid direct evidence produced errors: %v", direct.EvidenceErrors)
	}

	plan := codexContinuePlanManifest{Dispatches: []codexContinueExternalDispatch{{
		Stage: "review", Wave: 1, Caste: "auditor", Name: "Audit-External", Task: "audit", TaskID: "continue-review-auditor",
	}}}
	external, err := mergeExternalContinueResults(plan, []codexContinueExternalDispatch{{
		Stage: "review", Wave: 1, Caste: "auditor", Name: "Audit-External", Task: "audit", TaskID: "continue-review-auditor",
		Status: "completed", Artifacts: raw, Handoff: completedReviewerHandoff(),
	}})
	if err != nil {
		t.Fatalf("merge external reviewer artifact: %v", err)
	}
	if len(external) != 1 {
		t.Fatalf("external flow length = %d, want 1", len(external))
	}
	if !reflect.DeepEqual(direct.Findings, external[0].Findings) || !reflect.DeepEqual(direct.OverallScore, external[0].OverallScore) {
		t.Fatalf("review evidence drifted by lane:\ndirect: %#v\nexternal: %#v", direct, external[0])
	}

	signals := continueReviewAutopilotSignals(external)
	if !reviewerEvaluation(t, signals, autopilotTriggerAuditorScoreBelowFloor).Active {
		t.Fatal("a completed Auditor score of 59 did not activate the score-floor signal")
	}
	if !reviewerEvaluation(t, signals, autopilotTriggerCriticalReviewFinding).Active {
		t.Fatal("a Critical structured finding did not activate the critical-finding signal")
	}
	if got := len(signals.Findings); got != 2 {
		t.Fatalf("reportable findings = %d, want 2 (including High): %#v", got, signals.Findings)
	}
}

func TestAuditorScoreActivationRequiresValidCompletedAuditor(t *testing.T) {
	tests := []struct {
		name   string
		caste  string
		status string
		score  string
		active bool
	}{
		{name: "below floor", caste: "auditor", status: "completed", score: "59", active: true},
		{name: "at floor", caste: "auditor", status: "completed", score: "60", active: false},
		{name: "different caste", caste: "watcher", status: "completed", score: "12", active: false},
		{name: "failed auditor", caste: "auditor", status: "failed", score: "12", active: false},
		{name: "timed out auditor", caste: "auditor", status: "timeout", score: "12", active: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := normalizeContinueReviewEvidence(codexContinueWorkerFlowStep{
				Stage: "review", Caste: tt.caste, Name: "Reviewer", Status: tt.status,
			}, reviewerArtifactRaw(t, `{"overall_score":`+tt.score+`}`))
			evaluation := reviewerEvaluation(t, continueReviewAutopilotSignals([]codexContinueWorkerFlowStep{step}), autopilotTriggerAuditorScoreBelowFloor)
			if evaluation.Active != tt.active {
				t.Fatalf("score-floor active = %v, want %v; step=%#v evidence=%#v", evaluation.Active, tt.active, step, evaluation.Evidence)
			}
		})
	}

	noAuditor := continueReviewAutopilotSignals([]codexContinueWorkerFlowStep{{
		Stage: "review", Caste: "watcher", Name: "Watcher", Status: "completed",
	}})
	if reviewerEvaluation(t, noAuditor, autopilotTriggerAuditorScoreBelowFloor).Active {
		t.Fatal("an absent Auditor was silently treated as a zero score")
	}
}

func TestReviewerArtifactIsAuthoritativeAndMalformedEvidenceIsExplicit(t *testing.T) {
	plan := codexContinuePlanManifest{Dispatches: []codexContinueExternalDispatch{{
		Stage: "review", Wave: 1, Caste: "auditor", Name: "Audit", Task: "audit", TaskID: "audit",
	}}}
	flow, err := mergeExternalContinueResults(plan, []codexContinueExternalDispatch{{
		Stage: "review", Wave: 1, Caste: "auditor", Name: "Audit", Task: "audit", TaskID: "audit", Status: "completed",
		Summary: "CRITICAL score 0", Findings: []codexReviewFinding{{Severity: "CRITICAL", Description: "legacy output"}},
		Artifacts: reviewerArtifactRaw(t, `{"overall_score":60,"findings":[{"severity":"HIGH","description":"authoritative artifact","blocking":true}]}`),
		Handoff:   completedReviewerHandoff(),
	}})
	if err != nil {
		t.Fatalf("merge authoritative artifact: %v", err)
	}
	if got := flow[0].Findings; len(got) != 1 || got[0].Description != "authoritative artifact" {
		t.Fatalf("legacy prose/findings overrode artifacts.review: %#v", got)
	}
	signals := continueReviewAutopilotSignals(flow)
	if reviewerEvaluation(t, signals, autopilotTriggerAuditorScoreBelowFloor).Active || reviewerEvaluation(t, signals, autopilotTriggerCriticalReviewFinding).Active {
		t.Fatalf("summary or legacy output synthesized a stop: %#v", signals)
	}

	malformed := normalizeContinueReviewEvidence(codexContinueWorkerFlowStep{
		Stage: "review", Caste: "auditor", Name: "Bad-Audit", Status: "completed",
	}, reviewerArtifactRaw(t, `{"overall_score":-1,"findings":[{"severity":"catastrophic","description":"bad enum"}]}`))
	if malformed.OverallScore != nil {
		t.Fatalf("out-of-range score survived normalization: %v", *malformed.OverallScore)
	}
	if len(malformed.EvidenceErrors) == 0 {
		t.Fatal("malformed explicit artifact was silently converted into absent/zero evidence")
	}
	joined := strings.Join(malformed.EvidenceErrors, " ")
	if !strings.Contains(joined, "overall_score") || !strings.Contains(joined, "severity") {
		t.Fatalf("evidence errors do not name both malformed fields: %v", malformed.EvidenceErrors)
	}
	malformedSignals := continueReviewAutopilotSignals([]codexContinueWorkerFlowStep{malformed})
	if len(malformedSignals.EvidenceErrors) == 0 {
		t.Fatal("current-result autopilot signals omitted malformed evidence errors")
	}
}

func TestReviewerArtifactMissingAndEmptyCompletedAuditorEvidenceFailsClosed(t *testing.T) {
	tests := []struct {
		name      string
		artifacts map[string]json.RawMessage
		want      string
	}{
		{name: "missing artifact", artifacts: nil, want: "artifacts.review is required"},
		{name: "empty object", artifacts: reviewerArtifactRaw(t, `{}`), want: "overall_score is required"},
		{name: "null", artifacts: reviewerArtifactRaw(t, `null`), want: "must be a JSON object"},
		{name: "non object", artifacts: reviewerArtifactRaw(t, `[]`), want: "must be a JSON object"},
		{name: "malformed JSON", artifacts: reviewerArtifactRaw(t, `{"overall_score":`), want: "is malformed"},
		{name: "missing score", artifacts: reviewerArtifactRaw(t, `{"findings":[]}`), want: "overall_score is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := normalizeContinueReviewEvidence(completedReviewerStep("auditor", "Audit-Missing"), tt.artifacts)
			if len(step.EvidenceErrors) == 0 {
				t.Fatalf("invalid completed-Auditor evidence was accepted: %#v", step)
			}
			if got := strings.Join(step.EvidenceErrors, " "); !strings.Contains(got, tt.want) {
				t.Fatalf("evidence errors %q do not contain %q", got, tt.want)
			}
		})
	}
}

func TestReviewerArtifactScoreAndSeverityValidationMatrix(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "negative score", raw: `{"overall_score":-1}`, want: "between 0 and 100"},
		{name: "score above range", raw: `{"overall_score":101}`, want: "between 0 and 100"},
		{name: "fractional score", raw: `{"overall_score":59.5}`, want: "must be an integer"},
		{name: "string score", raw: `{"overall_score":"60"}`, want: "must be an integer"},
		{name: "invalid severity", raw: `{"overall_score":60,"findings":[{"severity":"catastrophic","description":"bad enum"}]}`, want: "severity must be"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := normalizeContinueReviewEvidence(
				completedReviewerStep("auditor", "Audit-Schema"),
				reviewerArtifactRaw(t, tt.raw),
			)
			if got := strings.Join(step.EvidenceErrors, " "); !strings.Contains(got, tt.want) {
				t.Fatalf("evidence errors %q do not contain %q; step=%#v", got, tt.want, step)
			}
		})
	}
}

func TestReviewerArtifactFindingBodyValidationMatrix(t *testing.T) {
	tests := []struct {
		name             string
		step             codexContinueWorkerFlowStep
		artifacts        map[string]json.RawMessage
		wantFinding      bool
		wantTitle        string
		wantDescription  string
		wantSuggestion   string
		wantErrorContext string
	}{
		{
			name:        "title only",
			step:        completedReviewerStep("watcher", "Review-Title"),
			artifacts:   reviewerArtifactRaw(t, `{"findings":[{"severity":"HIGH","title":"  concrete title  ","suggestion":"keep the advice"}]}`),
			wantFinding: true, wantTitle: "concrete title", wantDescription: "concrete title", wantSuggestion: "keep the advice",
		},
		{
			name:        "description only",
			step:        completedReviewerStep("watcher", "Review-Description"),
			artifacts:   reviewerArtifactRaw(t, `{"findings":[{"severity":"MEDIUM","description":"  concrete description  ","suggestion":"keep the advice"}]}`),
			wantFinding: true, wantDescription: "concrete description", wantSuggestion: "keep the advice",
		},
		{
			name:        "title and description",
			step:        completedReviewerStep("watcher", "Review-Both"),
			artifacts:   reviewerArtifactRaw(t, `{"findings":[{"severity":"LOW","title":"  concise title  ","description":"  detailed body  ","suggestion":"  recovery advice  "}]}`),
			wantFinding: true, wantTitle: "concise title", wantDescription: "detailed body", wantSuggestion: "recovery advice",
		},
		{
			name:             "whitespace only",
			step:             completedReviewerStep("watcher", "Review-Whitespace"),
			artifacts:        reviewerArtifactRaw(t, `{"findings":[{"severity":"INFO","title":"  ","description":"\n\t","suggestion":"recovery advice is not evidence"}]}`),
			wantErrorContext: "Review-Whitespace artifacts.review findings[0] must include a non-empty title or description",
		},
		{
			name:             "suggestion only critical",
			step:             completedReviewerStep("watcher", "Review-Suggestion"),
			artifacts:        reviewerArtifactRaw(t, `{"findings":[{"severity":"CRITICAL","suggestion":"rotate the credential"}]}`),
			wantErrorContext: "Review-Suggestion artifacts.review findings[0] must include a non-empty title or description",
		},
		{
			name: "legacy top-level suggestion only critical",
			step: codexContinueWorkerFlowStep{
				Stage: "review", Caste: "watcher", Name: "Review-Legacy", Status: "completed",
				Findings: []codexReviewFinding{{Severity: "CRITICAL", Suggestion: "rotate the credential"}},
			},
			wantErrorContext: "Review-Legacy legacy findings[0] must include a non-empty title or description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := normalizeContinueReviewEvidence(tt.step, tt.artifacts)
			if tt.wantFinding {
				if len(step.EvidenceErrors) != 0 {
					t.Fatalf("valid finding produced evidence errors: %v", step.EvidenceErrors)
				}
				if len(step.Findings) != 1 {
					t.Fatalf("valid findings = %#v, want exactly one", step.Findings)
				}
				finding := step.Findings[0]
				if finding.Title != tt.wantTitle || finding.Description != tt.wantDescription || finding.Suggestion != tt.wantSuggestion {
					t.Fatalf("normalized finding = %#v, want title=%q description=%q suggestion=%q", finding, tt.wantTitle, tt.wantDescription, tt.wantSuggestion)
				}
				return
			}

			if len(step.Findings) != 0 {
				t.Fatalf("blank-body finding survived normalization: %#v", step.Findings)
			}
			if got := strings.Join(step.EvidenceErrors, " "); !strings.Contains(got, tt.wantErrorContext) {
				t.Fatalf("evidence errors %q do not contain worker/index diagnostic %q", got, tt.wantErrorContext)
			}
		})
	}
}

func TestReviewerArtifactMalformedScorePreservesValidFindings(t *testing.T) {
	step := normalizeContinueReviewEvidence(
		completedReviewerStep("auditor", "Audit-Diagnostic"),
		reviewerArtifactRaw(t, `{
			"overall_score":"sixty",
			"findings":[{"severity":"HIGH","description":"retain this diagnostic"}]
		}`),
	)
	if len(step.EvidenceErrors) == 0 {
		t.Fatal("malformed score did not produce an evidence error")
	}
	if len(step.Findings) != 1 || step.Findings[0].Description != "retain this diagnostic" {
		t.Fatalf("valid findings were discarded with the malformed score: %#v", step.Findings)
	}
}

func TestReviewerArtifactScoreBoundaryAndAuditorAbsence(t *testing.T) {
	for _, score := range []int{59, 60} {
		t.Run(fmt.Sprintf("score %d", score), func(t *testing.T) {
			step := normalizeContinueReviewEvidence(
				completedReviewerStep("auditor", "Audit-Boundary"),
				reviewerArtifactRaw(t, fmt.Sprintf(`{"overall_score":%d}`, score)),
			)
			if len(step.EvidenceErrors) != 0 || step.OverallScore == nil || *step.OverallScore != score {
				t.Fatalf("valid boundary score %d was rejected: %#v", score, step)
			}
		})
	}

	for _, step := range []codexContinueWorkerFlowStep{
		completedReviewerStep("watcher", "No-Auditor"),
		{Stage: "review", Caste: "auditor", Name: "Failed-Auditor", Status: "failed"},
		{Stage: "review", Caste: "auditor", Name: "Timed-Out-Auditor", Status: "timeout"},
	} {
		normalized := normalizeContinueReviewEvidence(step, nil)
		if normalized.OverallScore != nil || len(normalized.EvidenceErrors) != 0 {
			t.Fatalf("non-completed or non-Auditor step gained score requirements: %#v", normalized)
		}
	}
}

type reviewerArtifactInvalidFixture struct {
	name        string
	artifacts   map[string]json.RawMessage
	wantError   string
	wantFinding bool
}

func reviewerArtifactInvalidFixtures(t *testing.T) []reviewerArtifactInvalidFixture {
	t.Helper()
	return []reviewerArtifactInvalidFixture{
		{name: "missing artifact", artifacts: nil, wantError: "artifacts.review is required"},
		{name: "empty object", artifacts: reviewerArtifactRaw(t, `{}`), wantError: "overall_score is required"},
		{name: "null", artifacts: reviewerArtifactRaw(t, `null`), wantError: "must be a JSON object"},
		{name: "non object", artifacts: reviewerArtifactRaw(t, `[]`), wantError: "must be a JSON object"},
		{name: "malformed JSON", artifacts: reviewerArtifactRaw(t, `{"overall_score":`), wantError: "is malformed"},
		{name: "missing score", artifacts: reviewerArtifactRaw(t, `{"findings":[{"severity":"HIGH","description":"retained high diagnostic"}]}`), wantError: "overall_score is required", wantFinding: true},
		{name: "negative score", artifacts: reviewerArtifactRaw(t, `{"overall_score":-1,"findings":[{"severity":"HIGH","description":"retained high diagnostic"}]}`), wantError: "between 0 and 100", wantFinding: true},
		{name: "score above range", artifacts: reviewerArtifactRaw(t, `{"overall_score":101,"findings":[{"severity":"HIGH","description":"retained high diagnostic"}]}`), wantError: "between 0 and 100", wantFinding: true},
		{name: "fractional score", artifacts: reviewerArtifactRaw(t, `{"overall_score":59.5,"findings":[{"severity":"HIGH","description":"retained high diagnostic"}]}`), wantError: "must be an integer", wantFinding: true},
		{name: "string score", artifacts: reviewerArtifactRaw(t, `{"overall_score":"60","findings":[{"severity":"HIGH","description":"retained high diagnostic"}]}`), wantError: "must be an integer", wantFinding: true},
		{name: "invalid severity", artifacts: reviewerArtifactRaw(t, `{"overall_score":60,"findings":[{"severity":"HIGH","description":"retained high diagnostic"},{"severity":"catastrophic","description":"invalid diagnostic"}]}`), wantError: "severity must be", wantFinding: true},
	}
}

type reviewerArtifactFlowInvoker struct {
	artifacts map[string]json.RawMessage
}

func (i *reviewerArtifactFlowInvoker) Invoke(_ context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	result := codex.WorkerResult{
		WorkerName: config.WorkerName,
		Caste:      config.Caste,
		TaskID:     config.TaskID,
		Status:     "completed",
		Summary:    config.Caste + " completed",
		RawOutput:  "direct reviewer diagnostic for " + config.WorkerName,
		Handoff:    completedReviewerHandoff(),
	}
	if strings.EqualFold(config.Caste, "auditor") {
		result.Artifacts = i.artifacts
	}
	return result, nil
}

func (*reviewerArtifactFlowInvoker) IsAvailable(context.Context) bool { return true }
func (*reviewerArtifactFlowInvoker) ValidateAgent(string) error       { return nil }

func reviewerArtifactAuditorStep(t *testing.T, report codexContinueReviewReport) codexContinueWorkerFlowStep {
	t.Helper()
	for _, step := range report.Workers {
		if strings.EqualFold(step.Caste, "auditor") {
			return step
		}
	}
	t.Fatalf("review report has no Auditor step: %#v", report.Workers)
	return codexContinueWorkerFlowStep{}
}

func assertReviewerArtifactBlockedEvidence(t *testing.T, result map[string]interface{}, state colony.ColonyState, diagnostic, wantError string, wantFinding bool) {
	t.Helper()
	if blocked, _ := result["blocked"].(bool); !blocked {
		t.Errorf("invalid completed-Auditor evidence did not block: %#v", result)
	}
	if advanced, _ := result["advanced"].(bool); advanced {
		t.Errorf("invalid completed-Auditor evidence advanced the phase: %#v", result)
	}
	if state.CurrentPhase != 1 || len(state.Plan.Phases) == 0 || state.Plan.Phases[0].Status == colony.PhaseCompleted {
		t.Errorf("phase state advanced despite invalid reviewer evidence: current=%d phases=%#v", state.CurrentPhase, state.Plan.Phases)
	}

	review, ok := result["review"].(codexContinueReviewReport)
	if !ok {
		t.Fatalf("blocked result omitted the typed review report: %#v", result["review"])
	}
	if review.Passed {
		t.Errorf("review.Passed = true, want false: %#v", review)
	}
	step := reviewerArtifactAuditorStep(t, review)
	if continueWorkerFlowStatus(step.Status) != buildWorkerCompleted {
		t.Errorf("Auditor status = %q, want completed", step.Status)
	}
	if !strings.Contains(step.Report, diagnostic) {
		t.Errorf("Auditor diagnostic %q was not retained in %q", diagnostic, step.Report)
	}
	evidenceText := strings.Join(step.EvidenceErrors, " ")
	if !strings.Contains(evidenceText, wantError) {
		t.Errorf("Auditor evidence errors %q do not contain %q", evidenceText, wantError)
	}
	if wantFinding && (len(step.Findings) != 1 || step.Findings[0].Description != "retained high diagnostic") {
		t.Errorf("valid High finding was not retained: %#v", step.Findings)
	}
	blockingText := strings.ToLower(strings.Join(review.BlockingIssues, " "))
	if !strings.Contains(blockingText, strings.ToLower(step.Name)) || !strings.Contains(blockingText, "auditor") || !strings.Contains(blockingText, strings.ToLower(wantError)) {
		t.Errorf("blocking issues must name the reviewer, caste, and validation reason: %v", review.BlockingIssues)
	}
}

func reviewerPhasePositionBytes(t *testing.T, state colony.ColonyState) []byte {
	t.Helper()
	position := struct {
		CurrentPhase int `json:"current_phase"`
		Phases       []struct {
			ID     int    `json:"id"`
			Status string `json:"status"`
		} `json:"phases"`
	}{
		CurrentPhase: state.CurrentPhase,
	}
	for _, phase := range state.Plan.Phases {
		position.Phases = append(position.Phases, struct {
			ID     int    `json:"id"`
			Status string `json:"status"`
		}{ID: phase.ID, Status: phase.Status})
	}
	raw, err := json.Marshal(position)
	if err != nil {
		t.Fatalf("marshal phase position: %v", err)
	}
	return raw
}

func loadReviewerArtifactState(t *testing.T) colony.ColonyState {
	t.Helper()
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load durable colony state: %v", err)
	}
	return state
}

func assertReviewerBoundaryResult(t *testing.T, result map[string]interface{}, state colony.ColonyState, before []byte, want string) {
	t.Helper()
	review, ok := result["review"].(codexContinueReviewReport)
	if !ok {
		t.Fatalf("continuation result omitted typed review diagnostics: %#v", result["review"])
	}
	blockingText := strings.ToLower(strings.Join(review.BlockingIssues, " "))

	switch want {
	case "invalid":
		if advanced, _ := result["advanced"].(bool); advanced {
			t.Fatalf("invalid review evidence advanced the phase: %#v", result)
		}
		if blocked, _ := result["blocked"].(bool); !blocked {
			t.Fatalf("invalid review evidence did not return blocked=true: %#v", result)
		}
		if review.Passed || !strings.Contains(blockingText, "review evidence is invalid") || !strings.Contains(blockingText, "findings[0] must include a non-empty title or description") {
			t.Fatalf("invalid review evidence diagnostic was not preserved: %#v", review)
		}
	case "critical":
		if advanced, _ := result["advanced"].(bool); advanced {
			t.Fatalf("valid Critical finding advanced the phase: %#v", result)
		}
		if blocked, _ := result["blocked"].(bool); !blocked {
			t.Fatalf("valid Critical finding did not block: %#v", result)
		}
		hasCritical := false
		for _, worker := range review.Workers {
			for _, finding := range worker.Findings {
				if strings.EqualFold(strings.TrimSpace(finding.Severity), "CRITICAL") {
					hasCritical = true
				}
			}
		}
		if review.Passed || !hasCritical || strings.Contains(blockingText, "review evidence is invalid") {
			t.Fatalf("valid Critical finding did not use the Critical-finding gate: %#v", review)
		}
	case "high":
		if advanced, _ := result["advanced"].(bool); !advanced {
			t.Fatalf("valid High finding did not advance: %#v", result)
		}
		if blocked, _ := result["blocked"].(bool); blocked {
			t.Fatalf("valid High finding returned blocked=true: %#v", result)
		}
		if !review.Passed || strings.Contains(blockingText, "critical finding") || strings.Contains(blockingText, "review evidence is invalid") {
			t.Fatalf("valid High finding changed D-05 severity policy: %#v", review)
		}
		return
	default:
		t.Fatalf("unknown reviewer boundary expectation %q", want)
	}

	if after := reviewerPhasePositionBytes(t, state); string(after) != string(before) {
		t.Fatalf("returned phase position changed before rejection:\nbefore: %s\nafter:  %s", before, after)
	}
	durable := loadReviewerArtifactState(t)
	if after := reviewerPhasePositionBytes(t, durable); string(after) != string(before) {
		t.Fatalf("durable phase position changed before rejection:\nbefore: %s\nafter:  %s", before, after)
	}
}

func TestSuggestionOnlyCriticalDirectFlowFailsBeforeAdvancement(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "suggestion-only Critical is invalid evidence",
			raw:  `{"overall_score":60,"findings":[{"domain":"quality","severity":"CRITICAL","suggestion":"rotate the credential"}]}`,
			want: "invalid",
		},
		{
			name: "valid Critical still blocks",
			raw:  `{"overall_score":60,"findings":[{"domain":"quality","severity":"CRITICAL","description":"credential is exposed","suggestion":"rotate the credential"}]}`,
			want: "critical",
		},
		{
			name: "valid High remains report-only",
			raw:  `{"overall_score":60,"findings":[{"domain":"quality","severity":"HIGH","description":"credential rotation should be scheduled","suggestion":"schedule the rotation","blocking":true}]}`,
			want: "high",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saveGlobals(t)
			resetRootCmd(t)
			root, _, _, _ := setupIntermediateContinueState(t, "Direct suggestion-only Critical boundary")
			before := reviewerPhasePositionBytes(t, loadReviewerArtifactState(t))

			newCodexWorkerInvoker = func() codex.WorkerInvoker {
				return &reviewerArtifactFlowInvoker{artifacts: reviewerArtifactRaw(t, tt.raw)}
			}
			result, state, _, _, _, _, err := runCodexContinue(root, codexContinueOptions{HeavyFlag: true})
			if err != nil {
				t.Fatalf("runCodexContinue: %v", err)
			}
			assertReviewerBoundaryResult(t, result, state, before, tt.want)
		})
	}
}

func TestSuggestionOnlyCriticalExternalFinalizeFailsBeforeAdvancement(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "suggestion-only Critical is invalid evidence",
			raw:  `{"overall_score":60,"findings":[{"domain":"quality","severity":"CRITICAL","suggestion":"rotate the credential"}]}`,
			want: "invalid",
		},
		{
			name: "valid Critical still blocks",
			raw:  `{"overall_score":60,"findings":[{"domain":"quality","severity":"CRITICAL","description":"credential is exposed","suggestion":"rotate the credential"}]}`,
			want: "critical",
		},
		{
			name: "valid High remains report-only",
			raw:  `{"overall_score":60,"findings":[{"domain":"quality","severity":"HIGH","description":"credential rotation should be scheduled","suggestion":"schedule the rotation","blocking":true}]}`,
			want: "high",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saveGlobals(t)
			resetRootCmd(t)
			root, _, _, _ := setupIntermediateContinueState(t, "External suggestion-only Critical boundary")

			planResult, _, _, _, err := runCodexContinuePlanOnly(root, codexContinueOptions{HeavyFlag: true})
			if err != nil {
				t.Fatalf("runCodexContinuePlanOnly: %v", err)
			}
			plan := planResult["continue_manifest"].(codexContinuePlanManifest)
			before := reviewerPhasePositionBytes(t, loadReviewerArtifactState(t))

			results := make([]codexContinueExternalDispatch, 0, len(plan.Dispatches))
			for _, dispatch := range plan.Dispatches {
				result := codexContinueExternalDispatch{
					Stage: dispatch.Stage, Wave: dispatch.Wave, Caste: dispatch.Caste, Name: dispatch.Name,
					Task: dispatch.Task, TaskID: dispatch.TaskID, Status: "completed",
					Summary: "external " + dispatch.Caste + " completed",
					Report:  "external reviewer boundary diagnostic for " + dispatch.Name,
					Handoff: completedReviewerHandoff(),
				}
				if strings.EqualFold(dispatch.Caste, "auditor") {
					result.Artifacts = reviewerArtifactRaw(t, tt.raw)
				}
				results = append(results, result)
			}

			result, state, _, _, _, _, err := runCodexContinueFinalize(root, codexExternalContinueCompletion{
				ContinueManifest: &plan,
				Dispatches:       results,
			}, false, 0, false)
			if err != nil {
				t.Fatalf("runCodexContinueFinalize: %v", err)
			}
			assertReviewerBoundaryResult(t, result, state, before, tt.want)
		})
	}
}

func TestReviewerArtifactDirectFlowFailsClosed(t *testing.T) {
	for _, fixture := range reviewerArtifactInvalidFixtures(t) {
		t.Run(fixture.name, func(t *testing.T) {
			saveGlobals(t)
			resetRootCmd(t)
			root, _, _, _ := setupIntermediateContinueState(t, "Direct reviewer evidence wall")

			invoker := &reviewerArtifactFlowInvoker{artifacts: fixture.artifacts}
			newCodexWorkerInvoker = func() codex.WorkerInvoker { return invoker }

			result, state, _, _, _, _, err := runCodexContinue(root, codexContinueOptions{HeavyFlag: true})
			if err != nil {
				t.Fatalf("runCodexContinue: %v", err)
			}
			assertReviewerArtifactBlockedEvidence(t, result, state, "direct reviewer diagnostic", fixture.wantError, fixture.wantFinding)
		})
	}
}

func TestReviewerArtifactExternalFinalizeFailsClosed(t *testing.T) {
	for _, fixture := range reviewerArtifactInvalidFixtures(t) {
		t.Run(fixture.name, func(t *testing.T) {
			saveGlobals(t)
			resetRootCmd(t)
			root, dataDir, _, _ := setupIntermediateContinueState(t, "External reviewer evidence wall")

			planResult, _, _, _, err := runCodexContinuePlanOnly(root, codexContinueOptions{HeavyFlag: true})
			if err != nil {
				t.Fatalf("runCodexContinuePlanOnly: %v", err)
			}
			plan := planResult["continue_manifest"].(codexContinuePlanManifest)
			results := make([]codexContinueExternalDispatch, 0, len(plan.Dispatches))
			for _, dispatch := range plan.Dispatches {
				result := codexContinueExternalDispatch{
					Stage: dispatch.Stage, Wave: dispatch.Wave, Caste: dispatch.Caste, Name: dispatch.Name,
					Task: dispatch.Task, TaskID: dispatch.TaskID, Status: "completed",
					Summary: "external " + dispatch.Caste + " completed",
					Report:  "external reviewer diagnostic for " + dispatch.Name,
					Handoff: completedReviewerHandoff(),
				}
				if strings.EqualFold(dispatch.Caste, "auditor") {
					result.Artifacts = fixture.artifacts
				}
				results = append(results, result)
			}

			result, state, _, _, _, _, err := runCodexContinueFinalize(root, codexExternalContinueCompletion{
				ContinueManifest: &plan,
				Dispatches:       results,
			}, false, 0, false)
			if err != nil {
				t.Fatalf("runCodexContinueFinalize: %v", err)
			}
			assertReviewerArtifactBlockedEvidence(t, result, state, "external reviewer diagnostic", fixture.wantError, fixture.wantFinding)

			var durable colony.ColonyState
			if err := store.LoadJSON(filepath.ToSlash("COLONY_STATE.json"), &durable); err != nil {
				t.Fatalf("load durable colony state from %s: %v", dataDir, err)
			}
			if durable.CurrentPhase != 1 || durable.Plan.Phases[0].Status == colony.PhaseCompleted {
				t.Fatalf("durable phase state advanced despite invalid reviewer evidence: %#v", durable)
			}
		})
	}
}

func TestHighReviewerFindingNeverBlocksSolelyOnLegacyBlockingBit(t *testing.T) {
	steps := []codexContinueWorkerFlowStep{{
		Stage: "review", Caste: "watcher", Name: "Sentinel", Status: "completed",
		Findings: []codexReviewFinding{{Severity: "HIGH", Description: "needs attention", Blocking: true}},
	}}
	planned := []codexContinueExternalDispatch{{Stage: "review", Caste: "watcher", Name: "Sentinel"}}
	report := externalContinueReviewReport(4, steps, time.Now().UTC(), false, colony.VerificationDepthStandard, planned)
	if !report.Passed {
		t.Fatalf("High finding blocked solely because blocking=true: %v", report.BlockingIssues)
	}
	if reviewerEvaluation(t, continueReviewAutopilotSignals(steps), autopilotTriggerCriticalReviewFinding).Active {
		t.Fatal("High finding activated the Critical-finding trigger")
	}
}
