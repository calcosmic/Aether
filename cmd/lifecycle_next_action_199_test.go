package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestLifecycleNextAction(t *testing.T) {
	tests := []struct {
		name            string
		facts           LifecycleFacts
		wantID          string
		wantRuntime     string
		wantChoiceIDs   []string
		wantAlternative string
	}{
		{name: "no goal", facts: projectionEmptyFacts(), wantID: "initialize", wantRuntime: `aether init "goal"`},
		{name: "goal without accepted plan", facts: projectionFacts(projectionState(colony.StateREADY, false, false)), wantID: "plan", wantRuntime: "aether plan"},
		{name: "accepted plan", facts: projectionFacts(projectionState(colony.StateREADY, true, false)), wantID: "choose_execution_mode", wantChoiceIDs: []string{"build", "run"}},
		{name: "paused", facts: projectionFacts(projectionState(colony.StateREADY, true, true)), wantID: "resume", wantRuntime: "aether resume"},
		{name: "uncertain recovery", facts: projectionMalformedEvidenceFacts(), wantID: "resume", wantRuntime: "aether resume"},
		{name: "sealed", facts: projectionSealedFacts(colony.SealDispositionVerified), wantID: "inspect_sealed", wantRuntime: "aether status", wantAlternative: "aether entomb"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveNextAction(nextActionInput{Facts: tc.facts, State: tc.facts.State.Value, NoColony: tc.facts.State.Source.Provenance == LifecycleFactMissing})
			projection := got.Projection
			if projection.NextAction.ID != tc.wantID || projection.NextAction.RuntimeCommand != tc.wantRuntime {
				t.Fatalf("projected next action = %#v, want id=%q runtime=%q", projection.NextAction, tc.wantID, tc.wantRuntime)
			}
			var choiceIDs []string
			for _, choice := range projection.NextAction.Choices {
				choiceIDs = append(choiceIDs, choice.ID)
				if choice.Recommended || choice.Rank != 0 || strings.EqualFold(choice.Label, "recommended") || strings.EqualFold(choice.Label, "preferred") {
					t.Fatalf("choice %q is ranked or preferred: %#v", choice.ID, choice)
				}
			}
			if strings.Join(choiceIDs, ",") != strings.Join(tc.wantChoiceIDs, ",") {
				t.Fatalf("choice ids = %#v, want %#v", choiceIDs, tc.wantChoiceIDs)
			}
			if tc.wantAlternative != "" {
				if len(projection.Alternatives) == 0 || projection.Alternatives[0].RuntimeCommand != tc.wantAlternative {
					t.Fatalf("alternatives = %#v, want first %q", projection.Alternatives, tc.wantAlternative)
				}
			}
		})
	}

	t.Run("runtime Claude and OpenCode spelling preserve one decision", func(t *testing.T) {
		facts := projectionFacts(projectionState(colony.StateREADY, true, false))
		answer := resolveNextAction(nextActionInput{Facts: facts, State: facts.State.Value})
		for _, platform := range []struct {
			name      string
			wantBuild string
			wantRun   string
		}{
			{name: "codex", wantBuild: "$ant-build 1", wantRun: "aether run"},
			{name: "claude", wantBuild: "/ant-build 1", wantRun: "/ant-run"},
			{name: "opencode", wantBuild: "/ant-build 1", wantRun: "/ant-run"},
		} {
			rendered := renderNextActionCardForPlatform(answer, platform.name)
			if !strings.Contains(rendered, platform.wantBuild) || !strings.Contains(rendered, platform.wantRun) {
				t.Fatalf("%s card lost coequal choices %q and %q:\n%s", platform.name, platform.wantBuild, platform.wantRun, rendered)
			}
			lower := strings.ToLower(rendered)
			if strings.Contains(lower, "recommended") || strings.Contains(lower, "preferred") {
				t.Fatalf("%s card ranks an accepted-plan choice:\n%s", platform.name, rendered)
			}
		}
	})
}

func TestNextActionCardUsesProjection(t *testing.T) {
	facts := projectionFacts(projectionState(colony.StateREADY, true, false))
	answer := resolveNextAction(nextActionInput{Facts: facts, State: facts.State.Value})
	card := renderNextActionCardForPlatform(answer, "claude")
	if !strings.Contains(card, answer.Projection.NextAction.Reason) {
		t.Fatalf("card changed or omitted projection reason %q:\n%s", answer.Projection.NextAction.Reason, card)
	}
	for _, choice := range answer.Projection.NextAction.Choices {
		want, ok := map[string]string{"aether build 1": "/ant-build 1", "aether run": "/ant-run"}[choice.RuntimeCommand]
		if !ok {
			t.Fatalf("unexpected runtime choice %q", choice.RuntimeCommand)
		}
		if !strings.Contains(card, want) {
			t.Fatalf("card omitted projected choice %q:\n%s", want, card)
		}
	}

	cardSource, err := os.ReadFile("next_action_card.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{
		"colony.State", "answer.State", "answer.Standing.State ==", "state.Paused",
		`"aether init`, `"aether plan`, `"aether build`, `"aether run`,
		`"aether resume`, `"aether seal`, `"aether entomb`,
	} {
		if strings.Contains(string(cardSource), forbidden) {
			t.Fatalf("next_action_card.go owns lifecycle policy %q instead of rendering the projection", forbidden)
		}
	}

	resolverSource, err := os.ReadFile("next_action.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(resolverSource), "func chooseNextAction(") {
		t.Fatal("next_action.go still contains an independent lifecycle state machine")
	}
	if !strings.Contains(string(resolverSource), "projectLifecycle(") {
		t.Fatal("next_action.go does not delegate lifecycle policy to projectLifecycle")
	}
}
