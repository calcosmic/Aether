package cmd

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// progressFixtureState wraps phase in a minimal, valid ColonyState fit for
// driving buildStatusResult/renderDashboard directly -- the real functions
// `aether status` calls (see WINDOWS.md rows 22/23: renderLifecycleStatus
// has no production caller; renderDashboard does).
func progressFixtureState(phase colony.Phase) colony.ColonyState {
	goal := "Progress fixture"
	return colony.ColonyState{
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			AcceptancePolicy: colony.PlanAcceptanceLegacyUnbound,
			EvidencePolicy:   colony.PlanEvidenceNotRequired,
			Phases:           []colony.Phase{phase},
		},
	}
}

// TestLessFinishedRecordIsBelieved table-drives both directions of
// disagreement between the stored phase and the durable check record
// through the real resolvePhaseProgressFromDisk, then renders the real
// dashboard (buildStatusResult + renderDashboard, the exact functions
// `aether status` calls) and asserts the rendered text carries the
// less-finished status and a disagreement line.
func TestLessFinishedRecordIsBelieved(t *testing.T) {
	cases := []struct {
		name           string
		phase          colony.Phase
		report         codexContinueReport
		wantStatus     string
		wantTasksTotal int
		wantTasksDone  int
	}{
		{
			// Stored phase status says complete, the check record shows
			// neither task verified: the owner is shown the check
			// record's less-finished answer.
			name: "stored says complete, check says nothing verified",
			phase: colony.Phase{
				ID: 1, Name: "Complete-but-behind", Status: colony.PhaseCompleted,
				Tasks: []colony.Task{
					{Goal: "A", Status: colony.TaskCompleted},
					{Goal: "B", Status: colony.TaskCompleted},
				},
			},
			report: codexContinueReport{
				Phase: 1,
				Tasks: []codexContinueTaskAssessment{
					{TaskID: "task-1", Verified: false},
					{TaskID: "task-2", Verified: false},
				},
			},
			wantStatus:     "not started",
			wantTasksTotal: 2,
			wantTasksDone:  0,
		},
		{
			// Stored phase status says pending, the check record says
			// complete: the owner is still shown the stored phase's own
			// less-finished answer.
			name: "stored says pending, check says complete",
			phase: colony.Phase{
				ID: 1, Name: "Pending-but-ahead", Status: colony.PhasePending,
				Tasks: []colony.Task{
					{Goal: "A", Status: colony.TaskPending},
					{Goal: "B", Status: colony.TaskPending},
				},
			},
			report: codexContinueReport{
				Phase:     1,
				Completed: true,
				Tasks: []codexContinueTaskAssessment{
					{TaskID: "task-1", Verified: true},
					{TaskID: "task-2", Verified: true},
				},
			},
			wantStatus:     "not started",
			wantTasksTotal: 2,
			wantTasksDone:  0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := newTestStoreCmd(t)
			store = s

			if err := store.SaveJSON(continuePlanArtifactsPath(tc.phase.ID, "continue.json"), tc.report); err != nil {
				t.Fatalf("seed continue.json: %v", err)
			}

			progress := resolvePhaseProgressFromDisk(tc.phase)
			if !progress.Disagreed {
				t.Fatalf("expected Disagreed=true, got false: %+v", progress)
			}
			if progress.Status != tc.wantStatus {
				t.Fatalf("Status = %q, want %q: %+v", progress.Status, tc.wantStatus, progress)
			}
			if progress.TasksTotal != tc.wantTasksTotal || progress.TasksDone != tc.wantTasksDone {
				t.Fatalf("TasksTotal/TasksDone = %d/%d, want %d/%d: %+v", progress.TasksTotal, progress.TasksDone, tc.wantTasksTotal, tc.wantTasksDone, progress)
			}

			state := progressFixtureState(tc.phase)
			result := buildStatusResult(state, store)
			rendered := renderDashboard(state, store, result)
			if !strings.Contains(rendered, progress.Status) {
				t.Fatalf("rendered dashboard does not carry the less-finished status %q:\n%s", progress.Status, rendered)
			}
			if !strings.Contains(strings.ToLower(rendered), "disagree") {
				t.Fatalf("rendered dashboard does not carry a disagreement line:\n%s", rendered)
			}
		})
	}
}

// TestProgressFromDiskAgreesWhenRecordsAgree asserts Disagreed is false and
// no extra disagreement line is rendered when the two records agree, and
// that an absent check record returns the stored status unchanged.
func TestProgressFromDiskAgreesWhenRecordsAgree(t *testing.T) {
	t.Run("records agree", func(t *testing.T) {
		s, _ := newTestStoreCmd(t)
		store = s

		phase := colony.Phase{
			ID: 1, Name: "Agreeing phase", Status: colony.PhaseInProgress,
			Tasks: []colony.Task{
				{Goal: "A", Status: colony.TaskCompleted},
				{Goal: "B", Status: colony.TaskPending},
			},
		}
		report := codexContinueReport{
			Phase: 1,
			Tasks: []codexContinueTaskAssessment{
				{TaskID: "task-1", Verified: true},
				{TaskID: "task-2", Verified: false},
			},
		}
		if err := store.SaveJSON(continuePlanArtifactsPath(phase.ID, "continue.json"), report); err != nil {
			t.Fatalf("seed continue.json: %v", err)
		}

		progress := resolvePhaseProgressFromDisk(phase)
		if progress.Disagreed {
			t.Fatalf("expected Disagreed=false when the two records rank the same, got true: %+v", progress)
		}
		if progress.Status != "in progress" {
			t.Fatalf("Status = %q, want %q: %+v", progress.Status, "in progress", progress)
		}

		state := progressFixtureState(phase)
		result := buildStatusResult(state, store)
		rendered := renderDashboard(state, store, result)
		if strings.Contains(strings.ToLower(rendered), "disagree") {
			t.Fatalf("rendered dashboard carries a disagreement line when the two records agree:\n%s", rendered)
		}
	})

	t.Run("absent check record returns the stored status unchanged", func(t *testing.T) {
		s, _ := newTestStoreCmd(t)
		store = s

		phase := colony.Phase{
			ID: 1, Name: "Never checked", Status: colony.PhaseInProgress,
			Tasks: []colony.Task{{Goal: "A", Status: colony.TaskCompleted}},
		}
		progress := resolvePhaseProgressFromDisk(phase)
		if progress.Disagreed {
			t.Fatalf("expected Disagreed=false with no check record on disk, got true: %+v", progress)
		}
		want := phaseProgressRankFromStoredPhase(phase)
		if progress.Status != want {
			t.Fatalf("Status = %q, want the stored phase's own rank %q: %+v", progress.Status, want, progress)
		}
		if progress.TasksTotal != 1 || progress.TasksDone != 1 {
			t.Fatalf("TasksTotal/TasksDone = %d/%d, want 1/1: %+v", progress.TasksTotal, progress.TasksDone, progress)
		}
	})
}

// TestResolvePhaseProgressFromDiskWritesNothing proves the resolution reads
// and never writes: snapshot the store directory before and after with the
// shared snapshotDirFiles helper and compare.
func TestResolvePhaseProgressFromDiskWritesNothing(t *testing.T) {
	s, root := newTestStoreCmd(t)
	store = s

	phase := colony.Phase{
		ID: 1, Name: "Snapshot phase", Status: colony.PhaseInProgress,
		Tasks: []colony.Task{{Goal: "A", Status: colony.TaskPending}},
	}
	report := codexContinueReport{Phase: 1, Advanced: true}
	if err := store.SaveJSON(continuePlanArtifactsPath(phase.ID, "continue.json"), report); err != nil {
		t.Fatalf("seed continue.json: %v", err)
	}

	before := snapshotDirFiles(t, root)
	_ = resolvePhaseProgressFromDisk(phase)
	after := snapshotDirFiles(t, root)

	if len(before) != len(after) {
		t.Fatalf("file count changed after a read-only resolution: before=%d after=%d", len(before), len(after))
	}
	for path, data := range before {
		afterData, ok := after[path]
		if !ok {
			t.Fatalf("file %s present before, missing after a read-only resolution", path)
		}
		if string(afterData) != string(data) {
			t.Fatalf("file %s changed content after a read-only resolution", path)
		}
	}
}

// --- Structural guard: one ordered progress-rank scale ---

// phaseProgressRankOrderAllowedNames is the closed set of variable names
// allowed to hold the ordered least-to-most-finished progress rank scale.
// Anything else matching the same shape is a second ranking rule.
var phaseProgressRankOrderAllowedNames = map[string]bool{
	"phaseProgressRankOrder": true,
}

// TestPhaseProgressRankOrderHasOneSource fails, naming the file, if a
// composite literal carrying at least three of phaseProgressRankOrder's own
// five plain-English phrases is ever assigned to any variable other than
// phaseProgressRankOrder itself, anywhere in cmd/*.go (production files
// only -- a test fixture reusing the same words to assert against the real
// scale is not a second scale).
func TestPhaseProgressRankOrderHasOneSource(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	fset := token.NewFileSet()
	filesScanned := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(".", name)
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		filesScanned++
		checkPhaseProgressRankOrderSite(t, path, file)
	}
	if filesScanned == 0 {
		t.Fatalf("scanned zero non-test .go files in .")
	}
}

func checkPhaseProgressRankOrderSite(t *testing.T, path string, file *ast.File) {
	t.Helper()
	report := func(varName string) {
		t.Errorf("%s: found a second ordered progress-rank scale assigned to %q -- only phaseProgressRankOrder may hold this vocabulary", path, varName)
	}
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.ValueSpec:
			for i, value := range node.Values {
				lit, ok := value.(*ast.CompositeLit)
				if !ok || !compositeLitCarriesRankPhrases(lit) {
					continue
				}
				varName := ""
				if i < len(node.Names) {
					varName = node.Names[i].Name
				}
				if !phaseProgressRankOrderAllowedNames[varName] {
					report(varName)
				}
			}
		case *ast.AssignStmt:
			for i, value := range node.Rhs {
				lit, ok := value.(*ast.CompositeLit)
				if !ok || !compositeLitCarriesRankPhrases(lit) {
					continue
				}
				varName := ""
				if i < len(node.Lhs) {
					if ident, ok := node.Lhs[i].(*ast.Ident); ok {
						varName = ident.Name
					}
				}
				if !phaseProgressRankOrderAllowedNames[varName] {
					report(varName)
				}
			}
		}
		return true
	})
}

func compositeLitCarriesRankPhrases(lit *ast.CompositeLit) bool {
	matches := 0
	for _, elt := range lit.Elts {
		basic, ok := elt.(*ast.BasicLit)
		if !ok || basic.Kind != token.STRING {
			continue
		}
		v, err := strconv.Unquote(basic.Value)
		if err != nil {
			continue
		}
		for _, phrase := range phaseProgressRankOrder {
			if v == phrase {
				matches++
				break
			}
		}
	}
	return matches >= 3
}
