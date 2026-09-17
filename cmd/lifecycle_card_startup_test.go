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
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// lifecycleSurfaceCase is one command run over one prepared project.
type lifecycleSurfaceCase struct {
	name string
	// args is the command as the owner types it.
	args []string
	// command is what that command calls itself when it asks the one resolver
	// what happens next -- part of the answer, because "what changed" names it.
	command string
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
	// renders for comparison are spelled the same way. Codex skill display is
	// pinned here; executable envelope fields must still use aether.
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
		run.answer = issuedNextActionFromEnvelope(t, c.name, run.envelope)
	} else {
		run.visual = stripANSI(buf.String())
		run.answer = resolveNextAction(loadNextActionInputForCommand(c.command))
	}
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

// issuedNextActionFromEnvelope reads the resolver-issued object carried by the
// command result. JSON tests must render this exact object rather than resolve
// a second answer after the command has returned: two equivalent-looking
// decisions are still two authorities that can drift.
func issuedNextActionFromEnvelope(t *testing.T, label string, envelope map[string]interface{}) nextAction {
	t.Helper()
	raw, ok := envelope[nextActionResultKey]
	if !ok {
		t.Fatalf("%s emits no resolver-issued %q object", label, nextActionResultKey)
	}
	data, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("%s marshal resolver-issued action: %v", label, err)
	}
	var answer nextAction
	if err := json.Unmarshal(data, &answer); err != nil {
		t.Fatalf("%s decode resolver-issued action: %v", label, err)
	}
	if answer.Projection == nil {
		t.Fatalf("%s resolver-issued action carries no lifecycle projection", label)
	}
	return answer
}

func lifecycleProjectionFromEnvelope(t *testing.T, label string, envelope map[string]interface{}) LifecycleProjection {
	t.Helper()
	raw, ok := envelope[lifecycleProjectionKey]
	if !ok {
		t.Fatalf("%s emits no %q object", label, lifecycleProjectionKey)
	}
	data, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("%s marshal lifecycle projection: %v", label, err)
	}
	var projection LifecycleProjection
	if err := json.Unmarshal(data, &projection); err != nil {
		t.Fatalf("%s decode lifecycle projection: %v", label, err)
	}
	return projection
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
			name:    "starting a project",
			args:    []string{"init", goal},
			command: "init",
			keep:    []string{"Queen has set the colony's intention"},
		},
		{
			name:    "talking the goal through",
			args:    []string{"discuss"},
			command: "discuss",
			prepare: readyNoPlan,
			keep:    []string{"Goal: " + goal},
		},
		{
			name:    "scanning the existing code",
			args:    []string{"colonize"},
			command: "colonize",
			prepare: readyNoPlan,
			keep:    []string{"Root: "},
		},
		{
			name:    "drawing up the plan",
			args:    []string{"plan"},
			command: "plan",
			prepare: readyNoPlan,
			keep:    []string{"Choose Planning Preset", "Planning did not start. State: unchanged."},
		},
		{
			// The replan variant: a plan already exists and is reloaded.
			name:    "reloading a plan that already exists",
			args:    []string{"plan"},
			command: "plan",
			prepare: func(t *testing.T, dataDir string) {
				writeLifecycleState(t, dataDir, colony.ColonyState{
					Version:      "3.0",
					Goal:         fixtureGoal(goal),
					State:        colony.StateREADY,
					CurrentPhase: 1,
					Milestone:    "Open Chambers",
					Plan: colony.Plan{Phases: []colony.Phase{
						fixturePhase(1, "Foundations", colony.PhaseReady),
						fixturePhase(2, "Billing engine", colony.PhasePending),
					}},
				})
			},
			keep: []string{"Choose Planning Preset", "Planning did not start. State: unchanged."},
		},
		{
			// Artifact validation is an inspection/repair route, not a new planning run.
			name:    "validating an existing legacy planning artifact",
			args:    []string{"plan", "--repair-artifact"},
			command: "plan",
			prepare: func(t *testing.T, dataDir string) {
				writeLifecycleState(t, dataDir, colony.ColonyState{
					Version:      "3.0",
					Goal:         fixtureGoal(goal),
					State:        colony.StateREADY,
					CurrentPhase: 1,
					Milestone:    "Open Chambers",
					Plan: colony.Plan{Phases: []colony.Phase{
						fixturePhase(1, "Foundations", colony.PhaseReady),
					}},
				})
				// The repair path reads the planning helpers' own artifact, so
				// it is written here through the runtime's own writer rather
				// than typed as raw JSON.
				if err := store.SaveJSON("planning/phase-plan.json", codexWorkerPlanArtifact{
					Confidence: codexPlanConfidence{Overall: 82},
					Phases: []codexWorkerPlanPhase{
						{Name: "Foundations", Tasks: []codexWorkerPlanTask{{Goal: "Lay the foundations"}}},
					},
				}); err != nil {
					t.Fatalf("write the planning artifact the repair path reads: %v", err)
				}
			},
			keep: []string{"Phase-plan dependency references are already valid.", "Artifact: .aether/data/planning/phase-plan.json"},
		},
		{
			// The blocked variant: planning stopped on a problem that needs the
			// owner. The blocker itself must still be named above the card.
			name:    "planning while a planning problem is still open",
			args:    []string{"plan"},
			command: "plan",
			prepare: func(t *testing.T, dataDir string) {
				writeLifecycleState(t, dataDir, colony.ColonyState{
					Version:   "3.0",
					Goal:      fixtureGoal(goal),
					State:     colony.StateREADY,
					Milestone: "First Mound",
				})
				writeLifecyclePlanBlocker(t, dataDir, "the planning step could not write the phase list")
			},
			keep: []string{"Choose Planning Preset", "Planning did not start. State: unchanged."},
		},
	}
}

// writeLifecyclePlanBlocker records an unresolved planning failure the way the
// planning step itself records one, so the card is driven by a real blocker
// rather than by a flag shape the runtime never writes.
func writeLifecyclePlanBlocker(t *testing.T, dataDir, description string) {
	t.Helper()
	phase := 0
	flags := colony.FlagsFile{
		Version: "1.0",
		Decisions: []colony.FlagEntry{{
			ID:          "flag_plan_finalize_failure",
			Type:        "blocker",
			Description: description,
			Phase:       &phase,
			Source:      planFinalizeFailureSource,
			CreatedAt:   "2026-08-28T09:00:00Z",
		}},
	}
	data, err := json.MarshalIndent(flags, "", "  ")
	if err != nil {
		t.Fatalf("marshal blocker: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "pending-decisions.json"), data, 0644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
}

// TestStartupLifecycleCardsComeFromTheResolver is the screen half. Each of the
// four commands must end with the card the one resolver produced, and must not
// have lost the part of its output that reports what this run actually did.
func TestStartupLifecycleCardsComeFromTheResolver(t *testing.T) {
	for _, surface := range startupLifecycleSurfaces() {
		t.Run(surface.name, func(t *testing.T) {
			run := runLifecycleSurface(t, surface, false)

			artifactRepair := surface.command == "plan" && slices.Contains(surface.args, "--repair-artifact")
			if surface.command == "discuss" || (surface.command == "plan" && !artifactRepair) {
				assertPhase200StartupBoundary(t, surface, run.visual)
			} else if !strings.Contains(run.visual, run.card) {
				t.Errorf("%s does not end with the shared card.\n--- the card the resolver produced ---\n%s\n--- what the command printed ---\n%s",
					surface.name, run.card, run.visual)
			}
			if artifactRepair && strings.Contains(run.visual, "Choose Planning Preset") {
				t.Error("artifact repair was routed into a new planning preset decision")
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
			if surface.command == "discuss" {
				assertPhase200DiscussEnvelope(t, surface.name, run.envelope)
				return
			}
			assertEnvelopeMatchesCard(t, surface.name, run)
		})
	}
}

func assertPhase200StartupBoundary(t *testing.T, surface lifecycleSurfaceCase, visual string) {
	t.Helper()
	switch surface.command {
	case "discuss":
		for _, want := range []string{"Draft Specification", "[DRAFT]", "Run `aether spec`", "planning remains unauthorized until the exact contract is approved"} {
			if !strings.Contains(visual, want) {
				t.Errorf("%s lost Phase 200 discuss boundary %q\n%s", surface.name, want, visual)
			}
		}
		if strings.Contains(visual, "Run `aether plan`") || strings.Contains(visual, "Run `$ant-plan`") {
			t.Errorf("%s skipped exact Specification approval\n%s", surface.name, visual)
		}
	case "plan":
		for _, want := range []string{"Choose Planning Preset", "Fast", "Balanced", "Deep", "Exhaustive", "Planning did not start. State: unchanged."} {
			if !strings.Contains(visual, want) {
				t.Errorf("%s lost Phase 200 preset boundary %q\n%s", surface.name, want, visual)
			}
		}
		for _, forbidden := range []string{"Planning Wave", "P L A N   D I S P A T C H", ".aether/data/spawn-tree.txt"} {
			if strings.Contains(visual, forbidden) {
				t.Errorf("%s crossed the unselected preset boundary via %q\n%s", surface.name, forbidden, visual)
			}
		}
	}
}

func assertPhase200DiscussEnvelope(t *testing.T, label string, envelope map[string]interface{}) {
	t.Helper()
	if envelope[nextActionCommandKey] != "aether spec" {
		t.Fatalf("%s next action = %v, want exact Specification review", label, envelope[nextActionCommandKey])
	}
	if envelope["specification_status"] != string(colony.SpecStatusDraft) || envelope["draft_spec"] == nil {
		t.Fatalf("%s did not carry its persisted DRAFT Specification: %+v", label, envelope)
	}
	if next := strings.ToLower(stringValue(envelope["next"])); strings.Contains(next, "aether plan") {
		t.Fatalf("%s machine result skipped Specification approval: %+v", label, envelope)
	}
}

// assertEnvelopeMatchesCard compares the command and the alternatives carried in
// a command's envelope against the ones its card shows.
func assertEnvelopeMatchesCard(t *testing.T, label string, run lifecycleRun) {
	t.Helper()
	if run.answer.Projection == nil {
		t.Fatalf("%s card was not fed a resolver-issued lifecycle projection", label)
	}
	envelopeProjection := lifecycleProjectionFromEnvelope(t, label, run.envelope)
	if !reflect.DeepEqual(envelopeProjection.NextAction, run.answer.Projection.NextAction) {
		t.Errorf("%s: envelope and card do not carry the same typed next action\n envelope: %+v\n card: %+v",
			label, envelopeProjection.NextAction, run.answer.Projection.NextAction)
	}
	if len(envelopeProjection.Alternatives) != len(run.answer.Projection.Alternatives) ||
		!reflect.DeepEqual(append([]LifecycleActionChoice(nil), envelopeProjection.Alternatives...),
			append([]LifecycleActionChoice(nil), run.answer.Projection.Alternatives...)) {
		t.Errorf("%s: envelope and card do not carry the same typed alternatives\n envelope: %+v\n card: %+v",
			label, envelopeProjection.Alternatives, run.answer.Projection.Alternatives)
	}
	command, ok := run.envelope[nextActionCommandKey].(string)
	if !ok {
		t.Fatalf("%s emits no %q in its machine-readable answer; a wrapper cannot read the next step out of it",
			label, nextActionCommandKey)
	}
	if command != run.answer.Command {
		t.Errorf("%s: the envelope says %q, the card says %q -- the screen and the wrapper disagree",
			label, command, run.answer.Command)
	}
	for _, runtimeCommand := range envelopeNextActionCommands(run.envelope) {
		if !strings.HasPrefix(runtimeCommand, "aether ") {
			t.Errorf("%s: machine-readable command %q lost its executable route", label, runtimeCommand)
		}
		if display := expectedCodexDisplayCommand(runtimeCommand); !strings.Contains(run.card, "`"+display+"`") {
			t.Errorf("%s: the envelope names %q, whose display %q does not appear on the card", label, runtimeCommand, display)
		}
	}
	for _, alternative := range run.answer.Alternatives {
		if !strings.HasPrefix(alternative.Command, "aether ") {
			t.Errorf("%s: machine-readable alternative %q lost its executable route", label, alternative.Command)
		}
		if !envelopeCarriesAlternative(run.envelope, alternative.Command) {
			t.Errorf("%s: the card offers %q as another way forward and the envelope does not carry it",
				label, alternative.Command)
		}
	}
	if _, ok := run.envelope[nextActionRecommendationKey]; !ok {
		t.Errorf("%s carries no plain-English reason (%q) beside its command", label, nextActionRecommendationKey)
	}
}

// TestStartupLifecycleProjectionUsesResolverIssuedActions is the structural
// half of the shared-authority contract. lifecycleProjectionDecision chooses
// candidate identities; only next_action.go may spell commands. A literal here
// would create a second command catalogue behind every startup/work-loop card.
func TestStartupLifecycleProjectionUsesResolverIssuedActions(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(repoRoot, "cmd", "lifecycle_projection.go"))
	if err != nil {
		t.Fatalf("read lifecycle_projection.go: %v", err)
	}
	source := string(data)
	start := strings.Index(source, "func lifecycleProjectionDecision(")
	end := strings.Index(source, "func lifecycleApplyPlatform(")
	if start < 0 || end <= start {
		t.Fatal("could not isolate lifecycleProjectionDecision for the command-authority check")
	}
	decision := source[start:end]
	if strings.Contains(decision, `"aether `) || strings.Contains(decision, "`aether ") {
		t.Fatal("lifecycleProjectionDecision still hand-types command advice; select resolver candidates and carry their issued action objects instead")
	}
}

// envelopeCarriesAlternative reads both shapes the answer arrives in: the
// in-process map a renderer is handed, and the same map after a JSON round trip
// out to a wrapper.
func envelopeCarriesAlternative(envelope map[string]interface{}, command string) bool {
	switch alternatives := envelope[nextActionAlternativesKey].(type) {
	case []nextActionAlternative:
		for _, alternative := range alternatives {
			if strings.TrimSpace(alternative.Command) == command {
				return true
			}
		}
	case []interface{}:
		for _, entry := range alternatives {
			alternative, ok := entry.(map[string]interface{})
			if !ok {
				continue
			}
			if strings.TrimSpace(stringValue(alternative["command"])) == command {
				return true
			}
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
