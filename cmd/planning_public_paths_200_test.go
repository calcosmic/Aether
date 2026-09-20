package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

	t.Run("init discuss spec and plan retain four surface parity", func(t *testing.T) {
		contracts := []struct {
			name    string
			flags   []string
			anchors []string
		}{
			{name: "init", flags: []string{"scope", "colony-mode", "charter-json"}, anchors: []string{"goal", "colony", "charter"}},
			{name: "discuss", flags: []string{"max-questions", "dry-run", "resolve", "answer"}, anchors: []string{"question", "answer", "plan"}},
			{name: "spec", flags: []string{"inspect", "approve", "repair-projection", "revision-id", "revision-hash", "approval-token"}, anchors: []string{"specification", "approval", "plan"}},
			{name: "plan", flags: []string{"refresh", "preset", "candidate", "accept-candidate", "spec-revision", "spec-hash", "base-plan-revision", "timeline-digest", "proposal-hash", "acceptance-token"}, anchors: []string{"candidate", "acceptance", "specification"}},
		}
		for _, contract := range contracts {
			t.Run(contract.name, func(t *testing.T) {
				runtime := "aether " + contract.name
				public := "/ant-" + contract.name
				canonicalPath := ".aether/commands/" + contract.name + ".yaml"
				canonical := planningPublicPaths200Read(t, repoRoot, canonicalPath)
				guide, err := buildCommandGuide(contract.name, "codex")
				if err != nil {
					t.Fatalf("build %s command guide: %v", contract.name, err)
				}
				guideJSON, err := json.Marshal(guide)
				if err != nil {
					t.Fatalf("marshal %s command guide: %v", contract.name, err)
				}
				guideTerminal := strings.Join(append(append(append([]string{guide.Intent, guide.RunCommand}, guide.PreSteps...), guide.PostSteps...), guide.DriftGuards...), "\n")
				contractText := planningPublicPaths200Read(t, repoRoot, "cmd/contracts/"+contract.name+".md")
				surfaces := map[string]string{
					"canonical YAML":                   canonical,
					"Claude projection":                planningPublicPaths200Read(t, repoRoot, ".claude/commands/ant/"+contract.name+".md"),
					"OpenCode projection":              planningPublicPaths200Read(t, repoRoot, ".opencode/commands/ant/"+contract.name+".md"),
					"plan contract":                    contractText,
					"command-guide JSON":               string(guideJSON),
					"command-guide terminal semantics": guideTerminal,
				}
				for label, surface := range surfaces {
					normalized := strings.ToLower(surface)
					publicIdentity := runtime
					if label == "plan contract" {
						publicIdentity = "# " + contract.name
					}
					if !strings.Contains(normalized, publicIdentity) {
						t.Errorf("%s %s omits public identity %q", contract.name, label, publicIdentity)
					}
					for _, anchor := range contract.anchors {
						if !strings.Contains(normalized, anchor) {
							t.Errorf("%s %s omits shared semantic anchor %q", contract.name, label, anchor)
						}
					}
				}
				for _, projectionPath := range []string{".claude/commands/ant/" + contract.name + ".md", ".opencode/commands/ant/" + contract.name + ".md"} {
					projection := planningPublicPaths200Read(t, repoRoot, projectionPath)
					managed := "Aether-managed: runtime spec at " + canonicalPath
					if !strings.Contains(projection, managed) ||
						(!strings.Contains(projection, public) && !strings.Contains(projection, "name: ant-"+contract.name)) {
						t.Errorf("%s is not a managed %s projection", projectionPath, contract.name)
					}
					for _, forbidden := range []string{"wrapper owns state", "wrapper writes state", "wrapper mints", "wrapper computes planning"} {
						if strings.Contains(strings.ToLower(projection), forbidden) {
							t.Errorf("%s claims forbidden authority %q", projectionPath, forbidden)
						}
					}
				}
				for _, flag := range contract.flags {
					if !planningPublicPaths200CommandHasFlag(contract.name, flag) {
						t.Errorf("live aether %s command is missing --%s", contract.name, flag)
					}
				}
			})
		}

		wrapperHost := planningPublicPaths200Read(t, repoRoot, ".aether/docs/wrapper-host-contract.md")
		for _, required := range []string{"mutate colony state", "result.acceptance_command", "execute the full command verbatim"} {
			if !strings.Contains(wrapperHost, required) {
				t.Errorf("wrapper host contract omits thin-wrapper rule %q", required)
			}
		}
	})

	t.Run("candidate acceptance and refresh recovery stay exact and available", func(t *testing.T) {
		root, candidate := planCandidateTestPending(t)
		current, err := reviewPlanCandidateAt(root, candidate.CreatedAt.Add(time.Minute))
		if err != nil {
			t.Fatalf("review current candidate: %v", err)
		}
		exactAcceptance := planCandidateAcceptanceCommand(planCandidateTestAcceptanceRequest(candidate))
		currentProjection := projectPlanningCandidate(current)
		if !currentProjection.AcceptanceAvailable || currentProjection.AcceptanceCommand != exactAcceptance || currentProjection.Next != exactAcceptance {
			t.Fatalf("current candidate did not preserve exact acceptance: %+v", currentProjection)
		}
		if available, ok := availableCommand(exactAcceptance); !ok || available != exactAcceptance {
			t.Fatalf("exact acceptance is not available on the live Cobra tree: %q", exactAcceptance)
		}
		currentJSON, err := json.Marshal(currentProjection)
		if err != nil {
			t.Fatal(err)
		}
		currentTerminal := renderPlanningCandidateVisual(current, planningVisualOptions{})
		exactAcceptanceJSON, err := json.Marshal(exactAcceptance)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(currentJSON), `"acceptance_command":`+string(exactAcceptanceJSON)) || !strings.Contains(currentTerminal, "Acceptance command: "+exactAcceptance) {
			t.Fatalf("JSON and terminal candidate views diverged from exact acceptance\nJSON: %s\nterminal:\n%s", currentJSON, currentTerminal)
		}

		authority := planCandidateExpiry200Authority(t, root, candidate)
		staleAuthority := authority
		staleAuthority.SpecificationRevisionID += "-successor"
		cases := []struct {
			name       string
			assessment planCandidateStandingAssessment
		}{
			{name: "stale", assessment: assessPlanCandidateStanding(candidate, staleAuthority, candidate.CreatedAt.Add(time.Minute))},
			{name: "expired", assessment: assessPlanCandidateStanding(candidate, authority, candidate.ExpiresAt)},
		}
		for _, test := range cases {
			t.Run(test.name, func(t *testing.T) {
				review := planningExpiryReview200(current, candidate, test.assessment)
				projection := projectPlanningCandidate(review)
				if projection.AcceptanceAvailable || projection.AcceptanceCommand != "" || projection.Next != planCandidateRefreshCommand {
					t.Fatalf("%s candidate did not expose refresh-only recovery: %+v", test.name, projection)
				}
				if available, ok := availableCommand(projection.Next); !ok || available != planCandidateRefreshCommand {
					t.Fatalf("%s recovery is not exact and available: %q", test.name, projection.Next)
				}
				encoded, err := json.Marshal(projection)
				if err != nil {
					t.Fatal(err)
				}
				terminal := renderPlanningCandidateVisual(review, planningVisualOptions{})
				if !strings.Contains(string(encoded), `"next":"aether plan --refresh"`) ||
					!strings.Contains(terminal, "Next: aether plan --refresh") ||
					strings.Contains(string(encoded), "--acceptance-token") || strings.Contains(terminal, "Acceptance command:") {
					t.Fatalf("%s JSON and terminal recovery semantics diverged\nJSON: %s\nterminal:\n%s", test.name, encoded, terminal)
				}
			})
		}
	})

	t.Run("Phase 199 focused gates remain green", func(t *testing.T) {
		for _, proof := range []string{
			`^TestCurrentVocabulary199($|/)`,
			`^TestPhase199GateReceiptSchema$`,
			`^TestPhase199GateReceipt$`,
		} {
			planningPublicPaths200RunProof(t, repoRoot, proof)
		}
	})
}

func planningPublicPaths200CommandHasFlag(command, flag string) bool {
	switch command {
	case "init":
		return initCmd.Flags().Lookup(flag) != nil
	case "discuss":
		return discussCmd.Flags().Lookup(flag) != nil
	case "spec":
		return newSpecCommand().Flags().Lookup(flag) != nil
	case "plan":
		return planCmd.Flags().Lookup(flag) != nil
	default:
		return false
	}
}

func planningPublicPaths200RunProof(t *testing.T, repoRoot, pattern string) {
	t.Helper()
	command := exec.Command(os.Args[0], "-test.run="+pattern, "-test.count=1")
	command.Dir = repoRoot
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("focused proof %s failed: %v\n%s", pattern, err, output)
	}
}

func planningPublicPaths200Read(t *testing.T, root, relative string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
	if err != nil {
		t.Fatalf("read %s: %v", relative, err)
	}
	return string(data)
}
