package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestPlanningPublicPaths200(t *testing.T) {
	repoRoot := findTestModuleRoot(t)

	t.Run("direct commands and managed wrappers share Go authority", func(t *testing.T) {
		spec := newSpecCommand()
		if spec.Use != "spec" || spec.Flags().Lookup("inspect") == nil || spec.Flags().Lookup("approve") == nil || spec.Flags().Lookup("repair-projection") == nil {
			t.Fatalf("direct aether spec constructor is incomplete: use=%q", spec.Use)
		}
		for _, flag := range []string{"preset", "candidate", "accept-candidate", "spec-revision", "timeline-digest", "acceptance-token"} {
			if planCmd.Flags().Lookup(flag) == nil {
				t.Errorf("direct aether plan command is missing --%s", flag)
			}
		}

		contracts := []struct {
			name      string
			canonical string
			claude    string
			opencode  string
			runtime   string
			public    string
		}{
			{name: "spec", canonical: ".aether/commands/spec.yaml", claude: ".claude/commands/ant/spec.md", opencode: ".opencode/commands/ant/spec.md", runtime: "aether spec", public: "/ant-spec"},
			{name: "plan", canonical: ".aether/commands/plan.yaml", claude: ".claude/commands/ant/plan.md", opencode: ".opencode/commands/ant/plan.md", runtime: "aether plan", public: "/ant-plan"},
		}
		for _, contract := range contracts {
			t.Run(contract.name, func(t *testing.T) {
				canonical := planningPublicPaths200Read(t, repoRoot, contract.canonical)
				if !strings.Contains(canonical, contract.runtime) || !strings.Contains(canonical, "Go") {
					t.Fatalf("canonical %s contract omits runtime spelling or Go authority", contract.name)
				}
				for _, projectionPath := range []string{contract.claude, contract.opencode} {
					projection := planningPublicPaths200Read(t, repoRoot, projectionPath)
					managed := "Aether-managed: runtime spec at " + contract.canonical
					if !strings.Contains(projection, managed) || !strings.Contains(projection, contract.public) || !strings.Contains(projection, contract.runtime) {
						t.Fatalf("%s is not an honest managed projection of %s", projectionPath, contract.canonical)
					}
					for _, forbidden := range []string{"write COLONY_STATE.json", "mint approval token", "calculate confidence"} {
						if strings.Contains(strings.ToLower(projection), strings.ToLower(forbidden)) {
							t.Fatalf("%s claims forbidden wrapper authority %q", projectionPath, forbidden)
						}
					}
				}
			})
		}

		command := exec.Command("go", "run", "./cmd/aether", "source-check")
		command.Dir = repoRoot
		command.Env = append(os.Environ(), "NO_COLOR=1", "AETHER_OUTPUT_MODE=json")
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("go run ./cmd/aether source-check: %v\n%s", err, output)
		}
	})

	t.Run("responsive visual and JSON projections retain one semantic result", func(t *testing.T) {
		t.Setenv("NO_COLOR", "1")
		review := planningVisualCandidateFixture()
		projection := projectPlanningCandidate(review)
		encoded, err := json.Marshal(projection)
		if err != nil {
			t.Fatal(err)
		}
		machine := string(encoded)
		for reason, label := range map[colony.PlanningStopReason]string{
			colony.PlanningStopTargetMet: "target sufficiency", colony.PlanningStopDiminishingReturns: "diminishing returns",
			colony.PlanningStopStalledGap: "stall detected", colony.PlanningStopPassCap: "iteration cap",
		} {
			candidate := review
			candidate.StopDecision.Reason = reason
			candidate.Candidate.StopDecision.Reason = reason
			projected := projectPlanningCandidate(candidate)
			if projected.StopReason != string(reason) || projected.StopReasonPublicLabel != label {
				t.Errorf("%s projection = %q/%q", reason, projected.StopReason, projected.StopReasonPublicLabel)
			}
		}
		if projection.EvidenceThatWouldChange != review.EvidenceThatWouldChange || projection.RecommendationRationale != review.Recommendation.Rationale || projection.RecommendationProducer != string(review.Recommendation.Producer) {
			t.Fatalf("JSON projection invented candidate evidence or recommendation: %+v", projection)
		}
		for _, token := range []string{`"stop_reason":"target_met"`, `"stop_reason_public_label":"target sufficiency"`, `"recommendation_producer":"queen"`} {
			if !strings.Contains(machine, token) {
				t.Errorf("machine projection missing %s: %s", token, machine)
			}
		}

		for _, width := range []int{47, 48, 63, 64, 95, 96} {
			t.Run(fmt.Sprintf("width_%d", width), func(t *testing.T) {
				options := planningVisualOptions{Width: width}
				views := map[string]string{
					"specification": renderPlanningSpecificationVisual(planningVisualSpecificationFixture(), options),
					"iteration":     renderPlanningIterationVisual(planningVisualIterationFixture(colony.PlanningStopPassCap), options),
					"candidate":     renderPlanningCandidateVisual(review, options),
				}
				for name, output := range views {
					assertPlanningVisualWidth(t, name, output, width)
					if strings.Contains(output, "\x1b[") || strings.Contains(output, "\r") {
						t.Fatalf("%s width %d violates redirected no-color output", name, width)
					}
				}
				if !strings.Contains(views["specification"], "Specification") || strings.Contains(views["specification"], "│ Spec ") {
					t.Fatalf("width %d abbreviates Specification:\n%s", width, views["specification"])
				}
				candidateWords := strings.Join(strings.Fields(views["candidate"]), " ")
				if !strings.Contains(views["iteration"], "iteration cap") || !strings.Contains(candidateWords, review.EvidenceThatWouldChange) || !strings.Contains(candidateWords, review.Recommendation.Rationale) {
					t.Fatalf("width %d lost persisted planning semantics", width)
				}
			})
		}
	})
}

func planningPublicPaths200Read(t *testing.T, root, relative string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
	if err != nil {
		t.Fatalf("read %s: %v", relative, err)
	}
	return string(data)
}
