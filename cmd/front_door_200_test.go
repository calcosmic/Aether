package cmd

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestFrontDoor200LifecycleJourney(t *testing.T) {
	const candidateID = "plan-candidate-front-door-200"

	baseFacts := func() LifecycleFacts {
		goal := "Ship a specification-first colony"
		state := colony.ColonyState{
			Goal:  &goal,
			State: colony.StateREADY,
			AcceptedCharter: &colony.AcceptedCharter{
				SchemaVersion: colony.AcceptedCharterSchemaVersion,
				EpisodeID:     "episode-front-door-200",
				Goal:          goal,
				Provenance:    "owner-provided",
				AcceptedAt:    time.Date(2026, time.September, 8, 0, 0, 0, 0, time.UTC),
			},
			Plan: colony.Plan{},
		}
		return lifecycleFactsFromStateSnapshot(state, false, time.Time{})
	}
	withApprovedSpec := func(facts LifecycleFacts) LifecycleFacts {
		facts.Specification.Value = LifecycleSpecificationFacts{
			Present:           true,
			SpecificationID:   "spec-front-door-200",
			CurrentRevisionID: "spec-revision-front-door-200",
			RevisionNumber:    1,
			ContentHash:       strings.Repeat("a", 64),
			Status:            colony.SpecStatusApproved,
			Approved:          true,
			ApprovalReceiptID: "spec-approval-front-door-200",
		}
		return facts
	}
	withAcceptedPlan := func(facts LifecycleFacts) LifecycleFacts {
		facts.Progress.Value = LifecycleProgressFacts{
			CurrentPhase: 1,
			Phases: []colony.Phase{{
				ID: 1, Name: "Build the accepted route", Status: colony.PhaseReady,
			}},
		}
		facts.Planning.Value = LifecyclePlanningFacts{
			ActivePlanRevisionID:    "plan-revision-front-door-200",
			ActivePlanRevisionHash:  strings.Repeat("b", 64),
			AcceptancePolicy:        colony.PlanAcceptanceExplicitOwner,
			AcceptanceBindingStatus: LifecyclePlanBindingAccepted,
			AcceptedPlan:            true,
		}
		return facts
	}

	tests := []struct {
		name             string
		facts            func() LifecycleFacts
		wantID           string
		wantCommand      string
		wantChoices      []string
		wantAlternatives []string
	}{
		{
			name:   "init to discuss",
			facts:  baseFacts,
			wantID: "discuss", wantCommand: "aether discuss",
		},
		{
			name: "discuss to draft specification review",
			facts: func() LifecycleFacts {
				facts := baseFacts()
				facts.Specification.Value = LifecycleSpecificationFacts{
					Present: true, SpecificationID: "spec-front-door-200", CurrentRevisionID: "spec-revision-front-door-200",
					RevisionNumber: 1, ContentHash: strings.Repeat("a", 64), Status: colony.SpecStatusDraft,
				}
				return facts
			},
			wantID: "specification", wantCommand: "aether spec",
		},
		{
			name:   "exact specification approval to planning",
			facts:  func() LifecycleFacts { return withApprovedSpec(baseFacts()) },
			wantID: "plan", wantCommand: "aether plan",
		},
		{
			name: "iterative stop to candidate review",
			facts: func() LifecycleFacts {
				facts := withApprovedSpec(baseFacts())
				facts.Planning.Value = LifecyclePlanningFacts{
					RunID: "planning-run-front-door-200", Stage: string(planningStageCandidateReady), Pass: 2,
					PendingCandidateID: candidateID, PendingCandidateHash: strings.Repeat("c", 64),
					PendingCandidateStatus:     colony.PlanCandidatePendingReview,
					PendingCandidateStopReason: colony.PlanningStopTargetMet,
					AcceptanceBindingStatus:    LifecyclePlanBindingAbsent,
				}
				return facts
			},
			wantID: "review_plan_candidate", wantCommand: "aether plan --candidate",
			wantAlternatives: []string{"aether plan --accept-candidate " + candidateID},
		},
		{
			name:   "exact candidate acceptance to build and run",
			facts:  func() LifecycleFacts { return withAcceptedPlan(withApprovedSpec(baseFacts())) },
			wantID: "choose_execution_mode", wantChoices: []string{"aether build 1", "aether run"},
		},
		{
			name: "scoped specification revision to reconciliation",
			facts: func() LifecycleFacts {
				facts := withAcceptedPlan(withApprovedSpec(baseFacts()))
				facts.Planning.Value.AcceptanceBindingStatus = LifecyclePlanBindingAffected
				facts.Planning.Value.AffectedUnresolvedSemanticIDs = []string{"requirement:report-export", "task:export-proof"}
				return facts
			},
			wantID: "reconcile_plan", wantCommand: "aether plan",
		},
	}

	var journey []string
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			answer := resolveNextAction(nextActionInput{Facts: tc.facts()})
			if answer.Projection == nil {
				t.Fatal("front-door answer omitted the shared lifecycle projection")
			}
			action := answer.Projection.NextAction
			if action.ID != tc.wantID || action.RuntimeCommand != tc.wantCommand {
				t.Fatalf("next action = %q %q, want %q %q", action.ID, action.RuntimeCommand, tc.wantID, tc.wantCommand)
			}
			var choices []string
			for _, choice := range action.Choices {
				choices = append(choices, choice.RuntimeCommand)
			}
			if strings.Join(choices, "|") != strings.Join(tc.wantChoices, "|") {
				t.Fatalf("execution choices = %v, want %v", choices, tc.wantChoices)
			}
			for _, want := range tc.wantAlternatives {
				found := false
				for _, alternative := range answer.Projection.Alternatives {
					found = found || alternative.RuntimeCommand == want
				}
				if !found {
					t.Errorf("alternatives %#v do not contain %q", answer.Projection.Alternatives, want)
				}
			}

			commands := []string{action.RuntimeCommand, action.DisplayCommand}
			for _, choice := range action.Choices {
				commands = append(commands, choice.RuntimeCommand, choice.DisplayCommand)
			}
			for _, command := range commands {
				if command == "" {
					continue
				}
				if !strings.HasPrefix(command, "aether ") || strings.Contains(command, "/ant-") || strings.Contains(command, "$ant-") {
					t.Errorf("direct Codex projection advertised unsupported command syntax %q", command)
				}
			}
		})
		journey = append(journey, tc.name)
	}

	wantJourney := []string{
		"init to discuss",
		"discuss to draft specification review",
		"exact specification approval to planning",
		"iterative stop to candidate review",
		"exact candidate acceptance to build and run",
		"scoped specification revision to reconciliation",
	}
	if strings.Join(journey, "|") != strings.Join(wantJourney, "|") {
		t.Fatalf("front-door journey order = %v, want %v", journey, wantJourney)
	}
}

func TestFrontDoor200InitCodexUsesNativeCommands(t *testing.T) {
	root, output := prepareFrontDoorInit199(t)
	t.Setenv("AETHER_PLATFORM", "codex")
	if err := runFrontDoorInit199(t, "Add CSV export to reports with tests"); err != nil {
		t.Fatalf("init: %v", err)
	}

	got := output.String()
	for _, want := range []string{
		"Discuss settles intent before specification review; it does not approve a specification or a plan.",
		"Next Up: $ant-discuss",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("direct Codex init is missing %q:\n%s", want, got)
		}
	}
	for _, forbidden := range []string{"/ant-", "$ant-spec", "$ant-status", "Next Up: aether plan", "Next Up: $ant-plan"} {
		if strings.Contains(got, forbidden) {
			t.Errorf("direct Codex init advertises unsupported or premature syntax %q:\n%s", forbidden, got)
		}
	}

	raw, err := os.ReadFile(root + "/.aether/data/COLONY_STATE.json")
	if err != nil {
		t.Fatalf("read initialized state: %v", err)
	}
	var state colony.ColonyState
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("decode initialized state: %v", err)
	}
	if state.Specification != nil || len(state.Plan.Phases) != 0 || len(state.Plan.Candidates) != 0 {
		t.Fatalf("init created specification or plan authority on the owner's behalf: specification=%#v plan=%#v", state.Specification, state.Plan)
	}
}
