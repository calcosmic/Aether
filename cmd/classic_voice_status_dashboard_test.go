package cmd

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// classicVoiceStatusDashboardRender renders the status screen THE COMMAND
// ACTUALLY DRAWS.
//
// The corpus already registered "status-full" and "status-compact", but those
// measure renderLifecycleStatus (cmd/lifecycle_status_render.go) — which has
// ZERO non-test callers. `aether status` renders via renderDashboard
// (cmd/status.go), reached from statusCmd's RunE through outputWorkflow. So
// Phase 202.1's Classic-voice guarantee for "status" was proved against code
// no command runs, while the screen the owner opens every day sat at ~23% of
// content lines symbol-led against a 41.3% reference bar.
//
// The owner reported exactly that: the February Classic look never arrived on
// the screen they use. It had arrived everywhere the tests were pointed.
//
// This registration closes the gap the only way that stays closed: measure the
// real renderer, so the bar fails if this screen ever flattens again.
func classicVoiceStatusDashboardRender(t *testing.T) string {
	t.Helper()
	saveGlobals(t)
	s, root := setupTestStore(t)
	store = s
	t.Setenv("AETHER_ROOT", root)
	state := loadStatusTestState(t, s)
	// The fixture's stock goal text contains the word "colony". That is
	// OWNER-supplied prose, not the program's own words, and the plain-English
	// check is about what the program says. A neutral goal keeps the check
	// measuring the screen rather than the fixture.
	neutralGoal := "Ship the reporting feature end to end"
	state.Goal = &neutralGoal
	// The what-next card appended to this screen re-reads the goal and the
	// event log from the store, not from the state value above, so the
	// override has to be persisted or the two halves of one screen disagree.
	// The seeded events also carry wording an older version wrote; refresh
	// them to what init_cmd.go writes TODAY, because a fixture must be in a
	// shape the current runtime actually produces.
	for i := range state.Events {
		state.Events[i] = strings.ReplaceAll(state.Events[i], "Colony initialized", "Project initialized")
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("persist the neutral fixture: %v", err)
	}
	return stripANSI(renderDashboard(state, s, buildStatusResult(state, s)))
}

func init() {
	registerVoiceScreen("status-dashboard", classicVoiceStatusDashboardRender)
}

// TestStatusDashboardMeetsTheReferenceDensity holds the REAL status screen to
// the same bar every other voiced screen meets. It is deliberately separate
// from TestStatusScreenMeetsTheReferenceDensity, which guards the lifecycle
// renderer: both may exist, but only this one is about what the owner sees.
func TestStatusDashboardMeetsTheReferenceDensity(t *testing.T) {
	reference := classicReferenceDensity(t)
	rendered := classicVoiceStatusDashboardRender(t)
	led, total, ratio := voiceDensity(rendered)
	if total == 0 {
		t.Fatalf("the status dashboard produced no content lines to measure")
	}
	if ratio < reference {
		t.Errorf("the status screen `aether status` actually draws measures %v (led=%d total=%d), below the reference figure %v.\nThis is the screen the owner opens; a green TestStatusScreenMeetsTheReferenceDensity does NOT cover it.\n%s",
			ratio, led, total, reference, rendered)
	}
}

// TestTheStatusCommandRendersTheScreenTheCorpusMeasures guards the whole
// class of mistake this file exists for: a voiced renderer that drifts out of
// the command's path, leaving a green density test measuring code nothing
// runs. That is precisely what happened to renderLifecycleStatus, which the
// corpus measures and which has zero production callers.
//
// It walks the real AST of every non-test file under cmd/ rather than
// grepping: a grep ratchet is defeated silently by a rename or a reformat,
// which is the same failure mode in a different coat.
func TestTheStatusCommandRendersTheScreenTheCorpusMeasures(t *testing.T) {
	callers := productionCallersOf(t, "renderDashboard")
	if len(callers) == 0 {
		t.Fatalf("renderDashboard has no production caller, so the corpus entry \"status-dashboard\" is now measuring a renderer no command runs — exactly the defect this file was added to close (renderLifecycleStatus got there first)")
	}
}

// productionCallersOf returns every non-test file under cmd/ containing a call
// to name. Shared by the guard above and available to any later screen that
// needs the same proof.
func productionCallersOf(t *testing.T, name string) []string {
	t.Helper()
	entries, err := os.ReadDir("../cmd")
	if err != nil {
		if entries, err = os.ReadDir("."); err != nil {
			t.Fatalf("read cmd dir: %v", err)
		}
	}
	var callers []string
	fset := token.NewFileSet()
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") {
			continue
		}
		path := n
		if _, statErr := os.Stat(path); statErr != nil {
			path = filepath.Join("../cmd", n)
		}
		file, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			continue
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			if id, ok := call.Fun.(*ast.Ident); ok && id.Name == name {
				callers = append(callers, n)
			}
			return true
		})
	}
	return callers
}
