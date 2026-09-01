package cmd

import (
	"encoding/json"
	"fmt"
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
