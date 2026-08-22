package cmd

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// deterministicFloorFixture builds a temp repo (AGENTS.md ## Verification
// Commands section), a phase, and a manifest for one row of the lane-parity
// table. Modeled on cmd/red_phase_test.go's redPhaseTestRoot and
// cmd/criterion_evidence_test.go's setupCriterionEvidenceTest.
type deterministicFloorFixture struct {
	name  string
	build func(t *testing.T) (root string, phase colony.Phase, manifest codexContinueManifest)
}

func writeAgentsVerificationCommands(t *testing.T, root string, lines ...string) {
	t.Helper()
	content := "## Verification Commands\n\n" + strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte(content), 0644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
}

func deterministicFloorFixtures() []deterministicFloorFixture {
	return []deterministicFloorFixture{
		{
			name: "all checks green",
			build: func(t *testing.T) (string, colony.Phase, codexContinueManifest) {
				s, root := newTestStore(t)
				store = s
				writeAgentsVerificationCommands(t, root,
					"- build: true", "- types: true", "- lint: true", "- tests: true")
				phase := colony.Phase{ID: 1, Name: "All green"}
				return root, phase, codexContinueManifest{}
			},
		},
		{
			name: "one check failing",
			build: func(t *testing.T) (string, colony.Phase, codexContinueManifest) {
				s, root := newTestStore(t)
				store = s
				writeAgentsVerificationCommands(t, root,
					"- build: true", "- types: true", "- lint: true", "- tests: false")
				phase := colony.Phase{ID: 1, Name: "One check failing"}
				return root, phase, codexContinueManifest{}
			},
		},
		{
			name: "a check blocked because a required command could not run",
			build: func(t *testing.T) (string, colony.Phase, codexContinueManifest) {
				s, root := newTestStore(t)
				store = s
				// Only "tests" resolves to a real command; "types" is bound
				// as required by the criterion below but AGENTS.md never
				// names one, so it must Block rather than silently pass
				// (D-10).
				writeAgentsVerificationCommands(t, root, "- tests: true")
				phase := colony.Phase{
					ID:              1,
					Name:            "Blocked required check",
					SuccessCriteria: []string{"types check enforced"},
					EvidenceRequirements: []colony.CriterionEvidenceRequirement{
						{Criterion: "types check enforced", Checks: []string{"types"}},
					},
				}
				manifest := codexContinueManifest{
					Present: true,
					Data: codexBuildManifest{
						Phase:                   1,
						CriterionEvidencePolicy: criterionEvidencePolicyBoundV1,
						EvidenceRequirements:    flattenPhaseCriterionEvidenceRequirements(phase),
					},
				}
				return root, phase, manifest
			},
		},
		{
			name: "no command resolves at all",
			build: func(t *testing.T) (string, colony.Phase, codexContinueManifest) {
				s, root := newTestStore(t)
				store = s
				// Deliberately no AGENTS.md/CLAUDE.md and no go.mod/package.json
				// etc, so resolveCodexVerificationCommands resolves nothing.
				phase := colony.Phase{ID: 1, Name: "Nothing mechanical to check"}
				return root, phase, codexContinueManifest{}
			},
		},
		{
			name: "claims mismatched",
			build: func(t *testing.T) (string, colony.Phase, codexContinueManifest) {
				s, root := newTestStore(t)
				store = s
				writeAgentsVerificationCommands(t, root,
					"- build: true", "- types: true", "- lint: true", "- tests: true")
				phase := colony.Phase{
					ID:              1,
					Name:            "Claims mismatched",
					SuccessCriteria: []string{"claims check enforced"},
					EvidenceRequirements: []colony.CriterionEvidenceRequirement{
						{Criterion: "claims check enforced", Checks: []string{"claims"}},
					},
				}
				claims := codexBuildClaims{
					BuildPhase:    1,
					Timestamp:     time.Now().UTC().Format(time.RFC3339),
					FilesModified: []string{"missing.txt"},
				}
				if err := s.SaveJSON("last-build-claims.json", claims); err != nil {
					t.Fatalf("save mismatched claims: %v", err)
				}
				manifest := codexContinueManifest{
					Present: true,
					Data: codexBuildManifest{
						Phase:                   1,
						ClaimsPath:              ".aether/data/last-build-claims.json",
						DispatchMode:            "real",
						CriterionEvidencePolicy: criterionEvidencePolicyBoundV1,
						EvidenceRequirements:    flattenPhaseCriterionEvidenceRequirements(phase),
						Dispatches: []codexBuildDispatch{
							{Stage: "wave", Caste: "builder", Name: "Forge-1", Task: "Build it", Status: "completed"},
						},
					},
				}
				return root, phase, manifest
			},
		},
	}
}

// TestBothContinueLanesApplyTheSameFloor compares, field by field, the
// report the in-process lane (runCodexContinueVerification) and the
// wrapper/external lane (runCodexContinueVerificationSnapshot) produce for
// the same inputs. Both lanes now share runDeterministicFloor, so this test
// fails if a future change grows a rule in one lane the other lacks.
func TestBothContinueLanesApplyTheSameFloor(t *testing.T) {
	for _, fixture := range deterministicFloorFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			saveGlobals(t)
			root, phase, manifest := fixture.build(t)

			inProcess, _ := runCodexContinueVerification(context.Background(), root, colony.ColonyState{}, phase, manifest, time.Second, 5*time.Second, true)
			snapshot := runCodexContinueVerificationSnapshot(root, phase, manifest, time.Now().UTC(), 5*time.Second, true)

			if inProcess.ChecksPassed != snapshot.ChecksPassed {
				t.Fatalf("ChecksPassed disagree: in-process=%v snapshot=%v\nin-process=%+v\nsnapshot=%+v", inProcess.ChecksPassed, snapshot.ChecksPassed, inProcess, snapshot)
			}
			if inProcess.CriteriaEnforced != snapshot.CriteriaEnforced {
				t.Fatalf("CriteriaEnforced disagree: in-process=%v snapshot=%v", inProcess.CriteriaEnforced, snapshot.CriteriaEnforced)
			}
			if inProcess.CriteriaPassed != snapshot.CriteriaPassed {
				t.Fatalf("CriteriaPassed disagree: in-process=%v snapshot=%v", inProcess.CriteriaPassed, snapshot.CriteriaPassed)
			}

			inProcessSteps := stepNames(inProcess.Steps)
			snapshotSteps := stepNames(snapshot.Steps)
			if strings.Join(inProcessSteps, ",") != strings.Join(snapshotSteps, ",") {
				t.Fatalf("step name order disagrees: in-process=%v snapshot=%v", inProcessSteps, snapshotSteps)
			}
			if strings.Join(inProcessSteps, ",") != "build,types,lint,tests" {
				t.Fatalf("step order = %v, want build,types,lint,tests", inProcessSteps)
			}

			inProcessIssues := sortedCopy(inProcess.BlockingIssues)
			snapshotIssues := sortedCopy(snapshot.BlockingIssues)
			if strings.Join(inProcessIssues, "|") != strings.Join(snapshotIssues, "|") {
				t.Fatalf("blocking-issue set disagrees:\nin-process=%v\nsnapshot=%v", inProcessIssues, snapshotIssues)
			}
		})
	}
}

func stepNames(steps []codexVerificationStep) []string {
	names := make([]string, 0, len(steps))
	for _, step := range steps {
		names = append(names, step.Name)
	}
	return names
}

func sortedCopy(values []string) []string {
	out := append([]string{}, values...)
	sort.Strings(out)
	return out
}

// TestDeterministicFloorIsTheOnlySourceOfAPass locks the assumption_delta
// decision in 193-01-PLAN.md: the deterministic floor is promoted to the
// primary verification anchor, and a reviewer verdict is demoted to one
// optional additional evidence source that can only ever ADD a block, never
// supply the pass. For every fixture whose floor genuinely fails, no watcher
// value handed to runDeterministicFloor -- passing, skipped, or absent --
// flips ChecksPassed to true.
func TestDeterministicFloorIsTheOnlySourceOfAPass(t *testing.T) {
	failingFixtures := []string{
		"one check failing",
		"a check blocked because a required command could not run",
		"claims mismatched",
	}
	watcherInputs := []struct {
		name    string
		watcher codexWatcherVerification
	}{
		{"absent", codexWatcherVerification{}},
		{"skipped", codexWatcherVerification{Present: true, Passed: true, Status: "skipped", Worker: "skip-watchers"}},
		{"passing reviewer", codexWatcherVerification{Present: true, Passed: true, Status: "completed", Worker: "Keen-1"}},
	}

	for _, fixture := range deterministicFloorFixtures() {
		if !containsString(failingFixtures, fixture.name) {
			continue
		}
		for _, wi := range watcherInputs {
			t.Run(fixture.name+"/"+wi.name, func(t *testing.T) {
				saveGlobals(t)
				root, phase, manifest := fixture.build(t)
				floor := runDeterministicFloor(context.Background(), root, phase, manifest, wi.watcher, 5*time.Second)
				if floor.ChecksPassed {
					t.Fatalf("a %q watcher flipped a failing floor to ChecksPassed=true: %+v", wi.name, floor)
				}
			})
		}
	}
}

// TestDeterministicFloorStepOrderIsFixed proves the floor's step names come
// back in the order build, types, lint, tests on both lanes, and that two
// consecutive runs over identical unchanged state return the identical
// ordered list.
func TestDeterministicFloorStepOrderIsFixed(t *testing.T) {
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s
	writeAgentsVerificationCommands(t, root,
		"- build: true", "- types: true", "- lint: true", "- tests: true")
	phase := colony.Phase{ID: 1, Name: "Order fixture"}
	manifest := codexContinueManifest{}
	watcher := codexWatcherVerification{}

	first := runDeterministicFloor(context.Background(), root, phase, manifest, watcher, 5*time.Second)
	second := runDeterministicFloor(context.Background(), root, phase, manifest, watcher, 5*time.Second)

	want := []string{"build", "types", "lint", "tests"}
	firstNames := stepNames(first.Steps)
	secondNames := stepNames(second.Steps)
	if strings.Join(firstNames, ",") != strings.Join(want, ",") {
		t.Fatalf("first run step order = %v, want %v", firstNames, want)
	}
	if strings.Join(secondNames, ",") != strings.Join(want, ",") {
		t.Fatalf("second run step order = %v, want %v", secondNames, want)
	}
	if first.ChecksPassed != second.ChecksPassed {
		t.Fatalf("two runs over identical unchanged state disagreed on ChecksPassed: first=%v second=%v", first.ChecksPassed, second.ChecksPassed)
	}

	// Same proof, driven through both continue lanes.
	inProcess, _ := runCodexContinueVerification(context.Background(), root, colony.ColonyState{}, phase, manifest, time.Second, 5*time.Second, true)
	snapshot := runCodexContinueVerificationSnapshot(root, phase, manifest, time.Now().UTC(), 5*time.Second, true)
	if strings.Join(stepNames(inProcess.Steps), ",") != strings.Join(want, ",") {
		t.Fatalf("in-process lane step order = %v, want %v", stepNames(inProcess.Steps), want)
	}
	if strings.Join(stepNames(snapshot.Steps), ",") != strings.Join(want, ",") {
		t.Fatalf("snapshot lane step order = %v, want %v", stepNames(snapshot.Steps), want)
	}
}
