package cmd

// Phase 197 plan 04 -- the four commands that begin a project.
//
// Starting a project, talking the goal through, scanning existing code and
// drawing up the plan each used to end with their own hand-written block of
// advice. The starting one was the clearest case of the problem: three fixed
// alternatives, identical for every repository on earth, written once and never
// revisited.
//
// These tests drive the REAL commands over a saved project state and require
// two things of each:
//
//  1. the closing block on screen is the card the one resolver produced, byte
//     for byte -- not a block that merely looks similar; and
//  2. the machine-readable answer a wrapper reads names the same command and
//     the same alternatives as that card.
//
// Both are checked over the same prepared state, because a card that is right
// on screen and an envelope that disagrees with it is exactly the split
// criterion 2 exists to close, and neither half alone can catch it.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// lifecycleSurfaceCase is one command run over one prepared project.
type lifecycleSurfaceCase struct {
	name string
	// args is the command as the owner types it.
	args []string
	// prepare writes the saved state this run starts from. A nil prepare means
	// an empty project directory -- the fresh-checkout case.
	prepare func(t *testing.T, dataDir string)
	// keep are lines that report what THIS run did and must survive above the
	// card. Replacing the what-next block is the job; losing the report is a bug.
	keep []string
}

// lifecycleRun is one command's two outputs over one prepared state.
type lifecycleRun struct {
	visual   string
	envelope map[string]interface{}
	// answer is the one the project itself resolves after the run, and card is
	// that answer rendered. The command's screen output must contain the card.
	answer nextAction
	card   string
}

// runLifecycleSurface runs one command in one output mode over a freshly
// prepared project, and returns what it printed plus the answer the project
// resolves once it has finished.
//
// The workspace is built by the runtime's own store and the command is driven
// through the real cobra tree, so nothing here is a hand-typed fixture of a
// shape the runtime could not produce.
func runLifecycleSurface(t *testing.T, c lifecycleSurfaceCase, jsonMode bool) lifecycleRun {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	if jsonMode {
		t.Setenv("AETHER_OUTPUT_MODE", "json")
	} else {
		t.Setenv("AETHER_OUTPUT_MODE", "visual")
	}
	// Pin the platform so the card the command renders and the card this test
	// renders for comparison are spelled the same way. The runtime form is
	// pinned here; the slash spellings get their own test.
	t.Setenv("AETHER_PLATFORM", "codex")

	if c.prepare != nil {
		c.prepare(t, dataDir)
	}

	buf := &bytes.Buffer{}
	stdout = buf
	rootCmd.SetArgs(c.args)
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("%s returned an error: %v\noutput:\n%s", strings.Join(c.args, " "), err, buf.String())
	}

	run := lifecycleRun{}
	if jsonMode {
		run.envelope = parseClosingEnvelope(t, buf.String())
	} else {
		run.visual = stripANSI(buf.String())
	}
	run.answer = resolveNextAction(loadNextActionInput())
	run.card = stripANSI(renderNextActionCardForPlatform(run.answer, "codex"))
	return run
}

// parseClosingEnvelope pulls the result object out of a command's JSON output.
func parseClosingEnvelope(t *testing.T, output string) map[string]interface{} {
	t.Helper()
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &envelope); err != nil {
		t.Fatalf("command output is not JSON: %v\n%s", err, output)
	}
	if data, ok := envelope["data"].(map[string]interface{}); ok {
		return data
	}
	if result, ok := envelope["result"].(map[string]interface{}); ok {
		return result
	}
	return envelope
}

// startupStateWithGoal writes a saved project that has a goal and nothing else,
// through the runtime's own writer.
func writeLifecycleState(t *testing.T, dataDir string, state colony.ColonyState) {
	t.Helper()
	createTestColonyState(t, dataDir, normalizedFixtureState(t, state))
}

func startupLifecycleSurfaces() []lifecycleSurfaceCase {
	goal := "Ship the billing rewrite"

	readyNoPlan := func(t *testing.T, dataDir string) {
		writeLifecycleState(t, dataDir, colony.ColonyState{
			Version:   "3.0",
			Goal:      fixtureGoal(goal),
			State:     colony.StateREADY,
			Milestone: "First Mound",
		})
	}

	return []lifecycleSurfaceCase{
		{
			name: "starting a project",
			args: []string{"init", goal},
			keep: []string{"Queen has set the colony's intention"},
		},
		{
			name:    "talking the goal through",
			args:    []string{"discuss"},
			prepare: readyNoPlan,
			keep:    []string{"Goal: " + goal},
		},
		{
			name:    "scanning the existing code",
			args:    []string{"colonize"},
			prepare: readyNoPlan,
			keep:    []string{"Root: "},
		},
		{
			name:    "drawing up the plan",
			args:    []string{"plan"},
			prepare: readyNoPlan,
			keep:    []string{"Plan size: "},
		},
	}
}

// TestStartupLifecycleCardsComeFromTheResolver is the screen half. Each of the
// four commands must end with the card the one resolver produced, and must not
// have lost the part of its output that reports what this run actually did.
func TestStartupLifecycleCardsComeFromTheResolver(t *testing.T) {
	for _, surface := range startupLifecycleSurfaces() {
		t.Run(surface.name, func(t *testing.T) {
			run := runLifecycleSurface(t, surface, false)

			if !strings.Contains(run.visual, run.card) {
				t.Errorf("%s does not end with the shared card.\n--- the card the resolver produced ---\n%s\n--- what the command printed ---\n%s",
					surface.name, run.card, run.visual)
			}
			for _, keep := range surface.keep {
				if !strings.Contains(run.visual, keep) {
					t.Errorf("%s lost %q from above the closing block -- only the what-next block was supposed to change",
						surface.name, keep)
				}
			}
		})
	}
}

// TestStartupLifecycleEnvelopesMatchTheirCards is the machine-readable half.
// A wrapper reading the envelope and an owner reading the screen must be told
// the same thing.
func TestStartupLifecycleEnvelopesMatchTheirCards(t *testing.T) {
	for _, surface := range startupLifecycleSurfaces() {
		t.Run(surface.name, func(t *testing.T) {
			run := runLifecycleSurface(t, surface, true)
			assertEnvelopeMatchesCard(t, surface.name, run)
		})
	}
}

// assertEnvelopeMatchesCard compares the command and the alternatives carried in
// a command's envelope against the ones its card shows.
func assertEnvelopeMatchesCard(t *testing.T, label string, run lifecycleRun) {
	t.Helper()
	command, ok := run.envelope[nextActionCommandKey].(string)
	if !ok {
		t.Fatalf("%s emits no %q in its machine-readable answer; a wrapper cannot read the next step out of it",
			label, nextActionCommandKey)
	}
	if command != run.answer.Command {
		t.Errorf("%s: the envelope says %q, the card says %q -- the screen and the wrapper disagree",
			label, command, run.answer.Command)
	}
	if !strings.Contains(run.card, command) {
		t.Errorf("%s: the envelope names %q, which does not appear on the card at all", label, command)
	}
	for _, alternative := range run.answer.Alternatives {
		if !envelopeCarriesAlternative(run.envelope, alternative.Command) {
			t.Errorf("%s: the card offers %q as another way forward and the envelope does not carry it",
				label, alternative.Command)
		}
	}
	if _, ok := run.envelope[nextActionRecommendationKey]; !ok {
		t.Errorf("%s carries no plain-English reason (%q) beside its command", label, nextActionRecommendationKey)
	}
}

func envelopeCarriesAlternative(envelope map[string]interface{}, command string) bool {
	raw, ok := envelope[nextActionAlternativesKey].([]interface{})
	if !ok {
		return false
	}
	for _, entry := range raw {
		alternative, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		if strings.TrimSpace(stringValue(alternative["command"])) == command {
			return true
		}
	}
	return false
}

// TestMigratedStartupClosingsSayItOnce -- the card already says whether it is
// safe to close the chat. A renderer that also keeps its old sentence says it
// twice, and the two can then disagree.
func TestMigratedStartupClosingsSayItOnce(t *testing.T) {
	for _, surface := range startupLifecycleSurfaces() {
		t.Run(surface.name, func(t *testing.T) {
			run := runLifecycleSurface(t, surface, false)
			if count := strings.Count(run.visual, "clear your context"); count > 0 {
				t.Errorf("%s still prints the old context sentence %d time(s); the card carries that verdict now",
					surface.name, count)
			}
			if count := countCloseThisChatSentences(run.visual); count > 1 {
				t.Errorf("%s tells the owner whether to close the chat %d times", surface.name, count)
			}
		})
	}
}

func countCloseThisChatSentences(text string) int {
	return strings.Count(text, "close this chat")
}

// TestTheStartingCardNoLongerRecitesAFixedTrio is the structural half of the
// first task: the three alternatives the starting command used to recite for
// every project on earth must be gone from the source, not merely unreachable.
func TestTheStartingCardNoLongerRecitesAFixedTrio(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(repoRoot, "cmd", "codex_visuals.go"))
	if err != nil {
		t.Fatalf("read codex_visuals.go: %v", err)
	}
	source := string(data)
	for _, gone := range []string{
		"to lock down key clarifications before planning",
		"if you already know the tradeoffs and want the first phase map now",
		"first if you want a quick codebase scan before planning",
	} {
		if strings.Contains(source, gone) {
			t.Errorf("the starting command still recites its fixed alternative %q; the resolver produces the alternatives now", gone)
		}
	}
}
