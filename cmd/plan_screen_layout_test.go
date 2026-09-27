package cmd

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// The plan result screen -- what `aether plan --candidate` shows when a plan
// is waiting for approval, what `aether plan --accept-candidate` shows once it
// is approved, and the whole-plan summary `aether plan` / `plan-finalize`
// print -- used to open with candidate IDs, specification and proposal
// hashes, per-dimension readiness scores and evidence-ID lists, with the
// phases themselves near the bottom. The owner chose the verdict-first
// layout for it on 2026-09-27: "PLAN READY — N PHASES" and what to do about
// it, then THE PHASES one line each, then WHAT IT RESTS ON only when there is
// something open, then the existing approval command / closing card. The
// codes stay in the JSON result.

func planScreenAcceptedResult() map[string]interface{} {
	_, revision, receipt := classicVoicePlanAcceptanceFixture()
	candidate := screenPlanCandidateFixture().Candidate
	candidate.Status = colony.PlanCandidateAccepted
	candidate.Acceptance = &receipt
	revision.Phases = []colony.Phase{
		{ID: 1, Name: "Import old cards", Status: colony.PhaseReady, Tasks: []colony.Task{{Goal: "Read the export"}, {Goal: "Map the fields"}}},
		{ID: 2, Name: "French Basics deck", Status: colony.PhasePending, Tasks: []colony.Task{{Goal: "Write the deck"}}},
		{ID: 3, Name: "Daily reminders", Status: colony.PhasePending},
	}
	return map[string]interface{}{
		"operation": planCandidateOperationAccept, "candidate": candidate,
		"revision": revision, "acceptance_receipt": receipt, "replayed": false,
	}
}

// planScreenLegacyResult is a whole-plan result shaped the way
// runCodexPlanFinalize returns an activated plan: ten phases with tasks,
// confidence, a plan revision, planning files, dispatches and open gaps.
func planScreenLegacyResult() map[string]interface{} {
	phases := make([]colony.Phase, 0, 10)
	for i := 1; i <= 10; i++ {
		id := fmt.Sprintf("%d.1", i)
		phase := colony.Phase{
			ID: i, Name: fmt.Sprintf("Step %d of the installer", i), Status: colony.PhasePending,
			Tasks: []colony.Task{{ID: &id, Goal: "Do the first part", Hints: []string{"COLONY_STATE.json"}}, {Goal: "Do the second part"}},
		}
		if i == 1 {
			phase.Status = colony.PhaseReady
		}
		phases = append(phases, phase)
	}
	return map[string]interface{}{
		"planned": true, "existing_plan": false,
		"goal":       "Safe Basics installer for the flashcard app",
		"phases":     phases,
		"count":      len(phases),
		"confidence": codexPlanConfidence{Knowledge: 90, Requirements: 88, Risks: 70, Dependencies: 80, Effort: 75, Overall: 84},
		"plan_revision": colony.PlanRevision{
			ID: "rev-3f9a2c1d", Number: 2, ReasonType: "initial", Reason: "first plan",
		},
		"evidence_hash":  strings.Repeat("a", 64),
		"planning_files": []string{"scout-3f9a.json", "route-setter-3f9a.json"},
		"gaps": []string{
			"Assumes the export file is always UTF-8",
			"Assumes a backup folder is writable",
			"Assumes the owner confirms before install",
			"Assumes the menu has room for one more deck",
		},
	}
}

var planScreenHexRun = regexp.MustCompile(`[0-9a-f]{12,}`)

// planScreenBody is the part of a plan screen above the approval command or
// the closing card -- the part this layout owns.
func planScreenBody(rendered string) string {
	cut := len(rendered)
	for _, marker := range []string{"Accept this candidate?", "Next Up: choose an operating mode", "━━ 📊 W H A T   N E X T ━━", "━━ 🐜 N E X T   U P ━━"} {
		if i := strings.Index(rendered, marker); i >= 0 && i < cut {
			cut = i
		}
	}
	return rendered[:cut]
}

func TestPlanScreenLeadsWithTheVerdict(t *testing.T) {
	cases := []struct {
		name     string
		render   func() string
		verdict  string
		sentence string
		phases   []string
	}{
		{
			name:    "waiting for approval",
			render:  func() string { return renderPlanVisual(planCandidateReviewResult(screenPlanCandidateFixture())) },
			verdict: "PLAN READY — 1 PHASE", sentence: "Review it, then approve it.",
			phases: []string{"Phase 2 — Render planning truth"},
		},
		{
			name:    "approved",
			render:  func() string { return renderPlanVisual(planScreenAcceptedResult()) },
			verdict: "PLAN READY — 3 PHASES", sentence: "Approved. Next: `aether build 1`.",
			phases: []string{"Phase 1 — Import old cards (2 tasks)", "Phase 2 — French Basics deck (1 task)", "Phase 3 — Daily reminders"},
		},
		{
			name:    "whole plan written",
			render:  func() string { return renderPlanVisual(planScreenLegacyResult()) },
			verdict: "PLAN READY — 10 PHASES", sentence: "Approved. Next: `aether build 1`.",
			phases: []string{"Phase 1 — Step 1 of the installer (2 tasks)", "Phase 8 — Step 8 of the installer", "… 2 more"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rendered := stripANSI(tc.render())
			box := firstNonBlankLines(rendered, 4)
			if len(box) != 4 {
				t.Fatalf("expected a four-line verdict box at the top, got %q\n%s", box, rendered)
			}
			if !isHeavyRuleLine(box[0]) || !isHeavyRuleLine(box[3]) {
				t.Errorf("verdict box is not framed by full heavy lines: %q\n%s", box, rendered)
			}
			if want := voiceLine("plan", tc.verdict); strings.TrimSpace(box[1]) != want {
				t.Errorf("verdict line = %q, want %q\n%s", strings.TrimSpace(box[1]), want, rendered)
			}
			if strings.TrimSpace(box[2]) != tc.sentence {
				t.Errorf("verdict sentence = %q, want %q\n%s", strings.TrimSpace(box[2]), tc.sentence, rendered)
			}
			top := strings.Index(rendered, box[0])
			if strings.TrimSpace(rendered[:top]) != "" {
				t.Errorf("something is drawn above the verdict box:\n%s", rendered[:top])
			}
			body := planScreenBody(rendered)
			assertSectionsSeparated(t, body, []string{"The Phases"})
			for _, want := range tc.phases {
				if !strings.Contains(body, want) {
					t.Errorf("THE PHASES is missing %q:\n%s", want, body)
				}
			}
		})
	}

	t.Run("open assumptions are listed, at most three", func(t *testing.T) {
		body := planScreenBody(stripANSI(renderPlanVisual(planScreenLegacyResult())))
		assertSectionsSeparated(t, body, []string{"The Phases", "What It Rests On"})
		for _, want := range []string{"Assumes the export file is always UTF-8", "… 1 more"} {
			if !strings.Contains(body, want) {
				t.Errorf("WHAT IT RESTS ON is missing %q:\n%s", want, body)
			}
		}
		if strings.Contains(body, "Assumes the menu has room") {
			t.Errorf("WHAT IT RESTS ON shows more than three items:\n%s", body)
		}
	})

	t.Run("a pending plan never offers building before approval", func(t *testing.T) {
		rendered := stripANSI(renderPlanVisual(planCandidateReviewResult(screenPlanCandidateFixture())))
		if strings.Contains(rendered, "aether build") || strings.Contains(rendered, "aether run") {
			t.Errorf("pending candidate exposed execution before owner acceptance:\n%s", rendered)
		}
		review := screenPlanCandidateFixture()
		if !strings.Contains(rendered, review.AcceptanceCommand) {
			t.Errorf("the approval command is gone from the pending plan screen:\n%s", rendered)
		}
	})
}

func TestPlanScreenHasNoCodes(t *testing.T) {
	for name, render := range map[string]func() string{
		"waiting for approval": func() string { return renderPlanVisual(planCandidateReviewResult(screenPlanCandidateFixture())) },
		"approved":             func() string { return renderPlanVisual(planScreenAcceptedResult()) },
		"whole plan written":   func() string { return renderPlanVisual(planScreenLegacyResult()) },
	} {
		t.Run(name, func(t *testing.T) {
			rendered := stripANSI(render())
			body := planScreenBody(rendered)
			if m := planScreenHexRun.FindString(body); m != "" {
				t.Errorf("a hash (%q) is shown above the closing card:\n%s", m, body)
			}
			for _, code := range []string{
				"CANDIDATE-0", "SPEC-REV-", "PLAN-REV-", "EVIDENCE-0", "GAP-0", "TIMELINE-", "rev-3f9a", "r2 (",
				"hash", "Hash", "Digest", "Timeline", "Producer", "%", "Planning readiness", "Knowledge", "Requirements:",
				"scout-3f9a.json", ".aether/data", "Coordination:", "Standing:", "Expires:",
			} {
				if strings.Contains(body, code) {
					t.Errorf("plan screen still shows %q above the closing card:\n%s", code, body)
				}
			}
		})
	}
}

// Populate nested enums omitted by the older visual-only fixture so this
// fixture can also exercise the runtime's JSON contract.
func screenPlanCandidateFixture() planCandidateReview {
	review := planningVisualCandidateFixture()
	for i := range review.Candidate.DimensionAssessments {
		a := &review.Candidate.DimensionAssessments[i]
		a.RemainingGap = colony.PlanningGap{Dimension: a.Dimension, Materiality: colony.PlanningGapNonMaterial}
	}
	for i := range review.Iterations {
		review.Iterations[i].DimensionAssessments = append([]colony.PlanningDimensionAssessment(nil), review.Candidate.DimensionAssessments...)
	}
	return review
}

func TestPlanScreenExplainsUnavailableApprovalWithoutCodes(t *testing.T) {
	for _, reason := range []string{"candidate_expired", "specification_changed", "base_plan_changed", "proposal_changed", "timeline_changed", "candidate_body_changed", "planning_stage_changed", "clock_before_candidate_creation", "acceptance_receipt_invalid"} {
		t.Run(reason, func(t *testing.T) {
			review := screenPlanCandidateFixture()
			review.Refusal = &planCandidateRefusalDetails{Standing: planCandidateStandingStale, WhyUnavailable: reason, RecoveryCommand: planCandidateRefreshCommand}
			body := renderPlanningCandidateVisual(review, planningVisualOptions{})
			if !strings.Contains(body, "PLAN NEEDS ATTENTION") || !strings.Contains(body, planCandidateRefreshCommand) {
				t.Fatalf("missing refusal or recovery:\n%s", body)
			}
			if strings.Contains(body, reason) || strings.Contains(body, "Accept this candidate?") || strings.Contains(body, "aether build") {
				t.Fatalf("unavailable approval leaked codes or execution authority:\n%s", body)
			}
		})
	}
}
