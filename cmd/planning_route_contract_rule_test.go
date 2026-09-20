package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestStagedRouteBriefStatesTheProofLinkRule holds the Route-Setter's brief to
// the rule drafting and acceptance share. The expected wording is re-derived
// here from planningProofLinkRequired itself, so the brief cannot keep
// describing an old rule after the rule changes -- a Route-Setter that was
// never told the rule is how an unacceptable plan reached an owner.
func TestStagedRouteBriefStatesTheProofLinkRule(t *testing.T) {
	spec, ok := planningStageResumeWorkerSpec("route_setter")
	if !ok {
		t.Fatal("no Route-Setter worker spec")
	}
	stage := briefContractTestStageManifest(planningStageCasteRouteSetter)
	brief := renderPlanningWorkerBrief(t.TempDir(), codexSurveyContext{}, spec, &stage)

	everyNode := planningRuleTestLine(t, brief, "Every phase and every task must carry")
	userFacing := planningRuleTestLine(t, brief, "user_facing_semantic_ids")
	for _, kind := range planningProofLinkKinds() {
		field := planningProofField(kind)
		switch {
		case planningProofLinkRequired(kind, false):
			if !strings.Contains(everyNode, field) {
				t.Fatalf("brief does not tell the Route-Setter every node needs %s:\n%s", field, everyNode)
			}
		case planningProofLinkRequired(kind, true):
			if !strings.Contains(userFacing, field) || strings.Contains(everyNode, field) {
				t.Fatalf("brief must require %s only on user-facing work:\nevery node: %s\nuser-facing: %s", field, everyNode, userFacing)
			}
		default:
			t.Fatalf("proof kind %q is never required; the rule table has no case for it", kind)
		}
	}
	for _, want := range []string{"at least one phase or task must carry each kind", "exactly as the approved specification spells it"} {
		if !strings.Contains(brief, want) {
			t.Fatalf("Route-Setter brief omits %q", want)
		}
	}

	scoutSpec, _ := planningStageResumeWorkerSpec("scout")
	scoutStage := briefContractTestStageManifest(planningStageCasteScout)
	if scout := renderPlanningWorkerBrief(t.TempDir(), codexSurveyContext{}, scoutSpec, &scoutStage); strings.Contains(scout, "Every phase and every task must carry") {
		t.Fatal("Scout brief carries the Route-Setter's plan rule; Scouts do not return plans")
	}
}

// TestRouteSetterDefinitionsStateTheStagedContract checks the shipped
// Route-Setter definitions on every platform name the staged result schema and
// every proof-link field, so an agent file cannot keep describing only the
// legacy phase-plan.json lane.
func TestRouteSetterDefinitionsStateTheStagedContract(t *testing.T) {
	root := findTestModuleRoot(t)
	for _, relative := range []string{
		".claude/agents/ant/aether-route-setter.md",
		".opencode/agents/aether-route-setter.md",
		".codex/agents/aether-route-setter.toml",
	} {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		text := string(content)
		wants := []string{string(planningStageResultRouteSetter), "user_facing_semantic_ids"}
		for _, kind := range planningProofLinkKinds() {
			wants = append(wants, planningProofField(kind))
		}
		for _, want := range wants {
			if !strings.Contains(text, want) {
				t.Errorf("%s does not mention %q", relative, want)
			}
		}
	}
}

func planningRuleTestLine(t *testing.T, text, marker string) string {
	t.Helper()
	index := strings.Index(text, marker)
	if index < 0 {
		t.Fatalf("brief has no line containing %q:\n%s", marker, text)
	}
	start := strings.LastIndex(text[:index], "\n") + 1
	end := strings.Index(text[index:], "\n")
	if end < 0 {
		return text[start:]
	}
	return text[start : index+end]
}
