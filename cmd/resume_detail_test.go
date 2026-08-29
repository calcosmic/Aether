package cmd

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// Phase 198 plan 08 (SHOW-02) -- buildResumeDashboardResult has always built
// result["recent"] (decisions and events) and result["plan_revision"], but
// renderResumeVisual has never read either. Per-phase progress is the one
// genuinely new computation: today only the overall fraction exists. This
// file proves all three now reach the screen, on both the in-process typed
// shape and the JSON-round-tripped completion-map shape
// (renderContinueWorkerFlowValue precedent, 198-PATTERNS.md), and that the
// drift note is derived from the plan's own recorded revision rather than
// any invented signal (Phase 196 D-01).

// ---- Task 1: progress, phase by phase ----

// TestResumeShowsProgressPhaseByPhase covers a mix of finished/in-progress/
// not-started phases, the display cap with an honest overflow count, a
// colony with no plan at all rendering nothing, and the pre-existing overall
// fraction line staying unchanged.
func TestResumeShowsProgressPhaseByPhase(t *testing.T) {
	t.Run("mixed statuses render in plain English", func(t *testing.T) {
		entries := []resumePhaseProgressEntry{
			{Phase: 1, Name: "Foundation", Status: colony.PhaseCompleted},
			{Phase: 2, Name: "Core Features", Status: colony.PhaseInProgress},
			{Phase: 3, Name: "Polish", Status: colony.PhasePending},
		}

		var typedBuilder strings.Builder
		renderResumePhaseProgress(&typedBuilder, entries)
		typedOutput := typedBuilder.String()

		for _, want := range []string{
			"Phase 1 — Foundation: finished",
			"Phase 2 — Core Features: in progress",
			"Phase 3 — Polish: not started",
		} {
			if !strings.Contains(typedOutput, want) {
				t.Errorf("missing %q, got:\n%s", want, typedOutput)
			}
		}
		// Never the internal snake_case status verbatim.
		for _, internal := range []string{colony.PhaseCompleted, colony.PhaseInProgress, colony.PhasePending} {
			if strings.Contains(typedOutput, ": "+internal) {
				t.Errorf("rendered output leaks internal status %q verbatim:\n%s", internal, typedOutput)
			}
		}

		asMap := roundTripToMap(t, map[string]interface{}{"phase_progress": entries})
		var mapBuilder strings.Builder
		renderResumePhaseProgress(&mapBuilder, asMap["phase_progress"])
		mapOutput := mapBuilder.String()

		if typedOutput != mapOutput {
			t.Fatalf("round-tripped render diverged from typed render:\n%s", firstDiffLine(typedOutput, mapOutput))
		}
	})

	t.Run("more phases than the cap render the cap plus an honest count", func(t *testing.T) {
		entries := make([]resumePhaseProgressEntry, 0, 10)
		for i := 1; i <= 10; i++ {
			entries = append(entries, resumePhaseProgressEntry{Phase: i, Name: "Phase", Status: colony.PhaseCompleted})
		}
		var b strings.Builder
		renderResumePhaseProgress(&b, entries)
		output := b.String()

		if got := strings.Count(output, "finished"); got != resumePhaseProgressCap {
			t.Errorf("expected exactly %d shown phase lines, got %d in:\n%s", resumePhaseProgressCap, got, output)
		}
		if !strings.Contains(output, "(+2 more)") {
			t.Errorf("expected an honest overflow count of the 2 remaining phases, got:\n%s", output)
		}
	})

	t.Run("no plan renders no per-phase block and does not error", func(t *testing.T) {
		var b strings.Builder
		renderResumePhaseProgress(&b, nil)
		if output := b.String(); output != "" {
			t.Errorf("expected no output for a colony with no plan, got:\n%s", output)
		}

		var emptyTyped strings.Builder
		renderResumePhaseProgress(&emptyTyped, []resumePhaseProgressEntry{})
		if output := emptyTyped.String(); output != "" {
			t.Errorf("expected no output for an empty phase list, got:\n%s", output)
		}
	})

	t.Run("the existing overall fraction line is unchanged and still present", func(t *testing.T) {
		saveGlobalsCmd(t)
		var buf strings.Builder
		state := colony.ColonyState{
			CurrentPhase: 2,
			Plan: colony.Plan{
				Phases: []colony.Phase{
					{ID: 1, Name: "Foundation", Status: "completed"},
					{ID: 2, Name: "Core Features", Status: "in_progress"},
					{ID: 3, Name: "Polish", Status: "pending"},
				},
			},
		}
		s, tmpDir := newTestStoreCmd(t)
		defer os.RemoveAll(tmpDir)
		store = s
		if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
			t.Fatal(err)
		}
		result := buildResumeDashboardResult()
		buf.WriteString(renderResumeVisual(result, "", false))
		output := buf.String()

		if !strings.Contains(output, "Phase: 2/3") {
			t.Errorf("expected the existing overall fraction 'Phase: 2/3' to still appear, got:\n%s", output)
		}
		if !strings.Contains(output, "Phase Progress") {
			t.Errorf("expected the new Phase Progress block to appear, got:\n%s", output)
		}
		for _, want := range []string{
			"Phase 1 — Foundation: finished",
			"Phase 2 — Core Features: in progress",
			"Phase 3 — Polish: not started",
		} {
			if !strings.Contains(output, want) {
				t.Errorf("missing %q in full resume output:\n%s", want, output)
			}
		}
	})
}

// ---- Task 2: recent decisions and the drift note ----

// TestResumeShowsRecentDecisions covers a colony with recorded decisions
// (up to five, most recent first, each with a reason) and a colony with none.
func TestResumeShowsRecentDecisions(t *testing.T) {
	t.Run("renders up to five decisions with reasons", func(t *testing.T) {
		decisions := []colony.Decision{
			{ID: "d1", Phase: 1, Claim: "Use cobra for CLI", Rationale: "Standard pattern", Timestamp: "2026-04-01T10:00:00Z"},
			{ID: "d2", Phase: 1, Claim: "Use outputOK for responses", Rationale: "Matches shell", Timestamp: "2026-04-01T11:00:00Z"},
			{ID: "d3", Phase: 2, Claim: "Use typed structs", Rationale: "Type safety", Timestamp: "2026-04-01T12:00:00Z"},
		}
		recent := extractRecentDecisions(decisions, 5)

		var b strings.Builder
		renderResumeRecentDecisions(&b, recent)
		output := b.String()

		if !strings.Contains(output, "Recent Decisions") {
			t.Errorf("expected a Recent Decisions heading, got:\n%s", output)
		}
		// Most recent first.
		d3Idx := strings.Index(output, "Use typed structs")
		d1Idx := strings.Index(output, "Use cobra for CLI")
		if d3Idx == -1 || d1Idx == -1 || d3Idx > d1Idx {
			t.Errorf("expected most-recent-first ordering, got:\n%s", output)
		}
		for _, want := range []string{
			"Use cobra for CLI",
			"└── Standard pattern",
			"Use typed structs",
			"└── Type safety",
		} {
			if !strings.Contains(output, want) {
				t.Errorf("missing %q, got:\n%s", want, output)
			}
		}
	})

	t.Run("no recorded decisions renders nothing and does not error", func(t *testing.T) {
		var b strings.Builder
		renderResumeRecentDecisions(&b, extractRecentDecisions(nil, 5))
		if output := b.String(); output != "" {
			t.Errorf("expected no output for a colony with no recorded decisions, got:\n%s", output)
		}
	})
}

// TestDriftNoteIsDerivedNotInvented covers a real recorded revision, the
// legacy-fallback revision, a colony with no plan at all, and a go/ast
// assertion that the renderer computes no time-since or count-of-changes
// figure of its own -- it only reads the plan-revision value it is given.
func TestDriftNoteIsDerivedNotInvented(t *testing.T) {
	t.Run("real revision names the reason and how many phases it replaced", func(t *testing.T) {
		plan := colony.Plan{
			ActiveRevisionID: "rev-2",
			Revisions: []colony.PlanRevision{
				{ID: "rev-2", Number: 2, ReasonType: colony.PlanRevisionUserFeedback, Reason: "owner asked for a different approach", SupersededPhaseIDs: []int{3, 4}},
			},
			Phases: []colony.Phase{{ID: 1, Name: "Foundation", Status: "completed"}},
		}
		summary := planRevisionSummary(plan)

		var typedBuilder strings.Builder
		renderResumeDriftNote(&typedBuilder, summary)
		typedOutput := typedBuilder.String()

		for _, want := range []string{
			"owner asked for a different approach",
			"replaced 2 phases",
		} {
			if !strings.Contains(typedOutput, want) {
				t.Errorf("missing %q, got:\n%s", want, typedOutput)
			}
		}

		asMap := roundTripToMap(t, map[string]interface{}{"plan_revision": summary})
		var mapBuilder strings.Builder
		renderResumeDriftNote(&mapBuilder, asMap["plan_revision"])
		mapOutput := mapBuilder.String()
		if typedOutput != mapOutput {
			t.Fatalf("round-tripped render diverged from typed render:\n%s", firstDiffLine(typedOutput, mapOutput))
		}
	})

	t.Run("legacy fallback says the plan has not been revised", func(t *testing.T) {
		plan := colony.Plan{
			Phases: []colony.Phase{{ID: 1, Name: "Foundation", Status: "completed"}},
		}
		summary := planRevisionSummary(plan)

		var b strings.Builder
		renderResumeDriftNote(&b, summary)
		output := b.String()

		if !strings.Contains(output, "has not been revised since it was written") {
			t.Errorf("expected the legacy-fallback wording, got:\n%s", output)
		}
		if strings.ContainsAny(output, "0123456789") {
			t.Errorf("legacy fallback must never carry a fabricated number, got:\n%s", output)
		}
	})

	t.Run("no plan says nothing has been recorded, never a fabricated one", func(t *testing.T) {
		plan := colony.Plan{}
		summary := planRevisionSummary(plan)

		var b strings.Builder
		renderResumeDriftNote(&b, summary)
		output := b.String()

		if !strings.Contains(output, "No plan revision has been recorded") {
			t.Errorf("expected the no-plan wording, got:\n%s", output)
		}
		if strings.ContainsAny(output, "0123456789") {
			t.Errorf("drift note for a colony with no plan must contain no number, got:\n%s", output)
		}

		// Also cover the raw-nil case (COLONY_STATE.json missing entirely --
		// result["plan_revision"] is absent, not even an empty map).
		var nilBuilder strings.Builder
		renderResumeDriftNote(&nilBuilder, nil)
		if output := nilBuilder.String(); !strings.Contains(output, "No plan revision has been recorded") {
			t.Errorf("expected the no-plan wording for a nil value, got:\n%s", output)
		}
	})

	t.Run("computes no time-since or count-of-changes figure of its own", func(t *testing.T) {
		repoRoot := resumeDetailRepoRoot(t)
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, filepath.Join(repoRoot, "cmd", "codex_visuals.go"), nil, 0)
		if err != nil {
			t.Fatalf("parse codex_visuals.go: %v", err)
		}

		var fn *ast.FuncDecl
		for _, decl := range file.Decls {
			if fd, ok := decl.(*ast.FuncDecl); ok && fd.Name.Name == "renderResumeDriftNote" {
				fn = fd
				break
			}
		}
		if fn == nil {
			t.Fatal("renderResumeDriftNote not found in cmd/codex_visuals.go")
		}

		ast.Inspect(fn.Body, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkgIdent, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}
			if pkgIdent.Name == "time" {
				t.Errorf("renderResumeDriftNote references time.%s -- the drift note must be derived only from "+
					"the plan-revision value it is given, never a time-since figure it computes itself", sel.Sel.Name)
			}
			return true
		})
	})
}

func resumeDetailRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find the repository root (no go.mod found walking up)")
		}
		dir = parent
	}
}
