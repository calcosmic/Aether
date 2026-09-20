package cmd

// Phase 197 plan 07, task 1 -- criterion 2's own named test.
//
// Plans 197-01 through 197-06 built the one resolver, the one card, and
// migrated every lifecycle surface onto both. Each wave then wrote its OWN
// consistency test over the surfaces it migrated -- four files, four separate
// checks, none of which is the criterion the roadmap actually wrote. This file
// is that criterion, in one place:
//
//   "Every command's closing message (starting, discussing, planning,
//   building, continuing, pausing, resuming, sealing/finishing, updating,
//   recovering, checking status) is generated from that same shared logic,
//   and the machine-readable version carries the identical information
//   (TestEveryLifecycleCommandEndsWithNextAction)."
//
// -- ROADMAP.md, Phase 197, success criterion 2.
//
// The eleven names above are QUOTED, not derived: criterion 2 names them, and
// the set this test drives is checked against that exact list rather than
// against whatever the source happens to migrate next. The other direction --
// that no command OUTSIDE this list is quietly hand-writing its own advice --
// is the ratchet's job (cmd/next_action_hardcode_ratchet_test.go), not this
// test's.
//
// Every case below drives the REAL command (or, where a command's own
// override cannot be reproduced by a bare re-resolve -- update, recover,
// status -- the real command run twice, once per output mode, over the exact
// same saved fixture) through the exact same fixtures the four per-wave test
// files already built and already proved drive the real thing. Nothing here
// is a second, hand-typed copy of those fixtures that could quietly drift
// from what the command actually does.

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// criterionTwoLifecycleCommands is ROADMAP.md's own list for Phase 197
// success criterion 2, copied verbatim (see the file comment above). Order is
// not significant; membership is.
var criterionTwoLifecycleCommands = []string{
	"starting",
	"discussing",
	"planning",
	"building",
	"continuing",
	"pausing",
	"resuming",
	"sealing/finishing",
	"updating",
	"checking status",
}

// lifecycleCoverageCase drives one of criterion 2's eleven commands through
// the real cobra tree in both output modes.
type lifecycleCoverageCase struct {
	// name is one of criterionTwoLifecycleCommands; the anti-vacuity floor in
	// TestEveryLifecycleCommandEndsWithNextAction fails by name if it is not.
	name string
	// visual returns what the owner sees, pinned to the codex platform so the
	// command it recommends is spelled the same way the envelope carries it.
	visual func(t *testing.T) string
	// envelope returns the machine-readable result the SAME situation
	// produces -- not necessarily the same process run (update, recover and
	// status resolve an override from their own outcome that a bare re-resolve
	// cannot reproduce, so those three run the command twice, once per mode,
	// over the identical saved fixture), but always the identical decision.
	envelope func(t *testing.T) map[string]interface{}
}

// findLifecycleCase and findWorkLoopSurface look a case up by the exact name a
// per-wave test file already gave it, rather than by position -- so a
// reordered or renamed fixture fails loudly here by name instead of silently
// comparing the wrong situation.
func findLifecycleCase(t *testing.T, cases []lifecycleSurfaceCase, name string) lifecycleSurfaceCase {
	t.Helper()
	for _, c := range cases {
		if c.name == name {
			return c
		}
	}
	t.Fatalf("no lifecycle surface case named %q -- the per-wave fixture this coverage test depends on has moved or been renamed", name)
	return lifecycleSurfaceCase{}
}

func findWorkLoopSurface(t *testing.T, cases []workLoopSurface, name string) workLoopSurface {
	t.Helper()
	for _, c := range cases {
		if c.name == name {
			return c
		}
	}
	t.Fatalf("no work-loop surface named %q -- the per-wave fixture this coverage test depends on has moved or been renamed", name)
	return workLoopSurface{}
}

// lifecycleCoverageCases builds the eleven cases, one per name in
// criterionTwoLifecycleCommands, from the exact fixtures the four per-wave
// test files already drive through the real commands.
func lifecycleCoverageCases(t *testing.T) []lifecycleCoverageCase {
	t.Helper()

	startingCase := findLifecycleCase(t, startupLifecycleSurfaces(), "starting a project")
	discussingCase := findLifecycleCase(t, startupLifecycleSurfaces(), "talking the goal through")
	planningCase := findLifecycleCase(t, startupLifecycleSurfaces(), "drawing up the plan")
	pausingCase := findLifecycleCase(t, sessionLifecycleSurfaces(), "pausing")
	resumingCase := findLifecycleCase(t, sessionLifecycleSurfaces(), "resuming, canonical form")

	buildingSurface := findWorkLoopSurface(t, workLoopSurfaces(), "building a phase")
	continuingSurface := findWorkLoopSurface(t, workLoopSurfaces(), "checking the work")

	return []lifecycleCoverageCase{
		{
			name:     "starting",
			visual:   func(t *testing.T) string { return runLifecycleSurface(t, startingCase, false).visual },
			envelope: func(t *testing.T) map[string]interface{} { return runLifecycleSurface(t, startingCase, true).envelope },
		},
		{
			name:   "discussing",
			visual: func(t *testing.T) string { return runLifecycleSurface(t, discussingCase, false).visual },
			envelope: func(t *testing.T) map[string]interface{} {
				return runLifecycleSurface(t, discussingCase, true).envelope
			},
		},
		{
			name:     "planning",
			visual:   func(t *testing.T) string { return runLifecycleSurface(t, planningCase, false).visual },
			envelope: func(t *testing.T) map[string]interface{} { return runLifecycleSurface(t, planningCase, true).envelope },
		},
		{
			name:     "building",
			visual:   func(t *testing.T) string { return buildingSurface.visual(t).visual },
			envelope: func(t *testing.T) map[string]interface{} { return buildingSurface.envelope(t).envelope },
		},
		{
			name:     "continuing",
			visual:   func(t *testing.T) string { return continuingSurface.visual(t).visual },
			envelope: func(t *testing.T) map[string]interface{} { return continuingSurface.envelope(t).envelope },
		},
		{
			name:     "pausing",
			visual:   func(t *testing.T) string { return runLifecycleSurface(t, pausingCase, false).visual },
			envelope: func(t *testing.T) map[string]interface{} { return runLifecycleSurface(t, pausingCase, true).envelope },
		},
		{
			name:     "resuming",
			visual:   func(t *testing.T) string { return runLifecycleSurface(t, resumingCase, false).visual },
			envelope: func(t *testing.T) map[string]interface{} { return runLifecycleSurface(t, resumingCase, true).envelope },
		},
		{
			name:     "sealing/finishing",
			visual:   func(t *testing.T) string { return sealCardRun(t).visual },
			envelope: func(t *testing.T) map[string]interface{} { return sealCardRun(t).envelope },
		},
		{
			name:     "updating",
			visual:   lifecycleUpdateVisual,
			envelope: lifecycleUpdateEnvelope,
		},
		{
			name:     "checking status",
			visual:   lifecycleStatusVisual,
			envelope: lifecycleStatusEnvelope,
		},
	}
}

// ---------------------------------------------------------------------------
// The three commands whose own outcome is an override a bare re-resolve
// cannot reproduce: update's repair report, recover's own scan findings, and
// status's in-flight-workers / guided-action facts. Each is driven twice --
// once per output mode -- over the identical saved fixture, which is
// sufficient because none of the three mutates the fixture it reads (update
// and recover run read-only here; status never mutates).
// ---------------------------------------------------------------------------

func lifecycleUpdateVisual(t *testing.T) string {
	t.Helper()
	homeDir, repoDir := lifecycleUpdateProject(t)
	visual, _ := runSessionUpdate(t, homeDir, repoDir, false)
	return visual
}

func lifecycleUpdateEnvelope(t *testing.T) map[string]interface{} {
	t.Helper()
	homeDir, repoDir := lifecycleUpdateProject(t)
	_, envelope := runSessionUpdate(t, homeDir, repoDir, true)
	return envelope
}

// Give the isolated coverage child a contained repository and a complete
// current package. The older alias fixture relies on ambient store state and
// omits the private support files now required by install.
func lifecycleUpdateProject(t *testing.T) (homeDir, repoDir string) {
	t.Helper()
	saveGlobals(t)
	t.Setenv("AETHER_HUB_DIR", "")
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	packageDir := buildAliasReconcilePackageDir(t)
	seedCodexSkillSupportFixture(t, packageDir)
	homeDir, repoDir = t.TempDir(), t.TempDir()
	bindCommandTestRepositoryAt(t, repoDir)
	for _, args := range [][]string{
		{"install", "--package-dir", packageDir, "--home-dir", homeDir, "--skip-build-binary"},
		{"setup", "--repo-dir", repoDir, "--home-dir", homeDir},
	} {
		resetRootCmd(t)
		var buf bytes.Buffer
		stdout, stderr = &buf, &buf
		rootCmd.SetArgs(args)
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("%s fixture failed: %v\n%s", args[0], err, buf.String())
		}
		var result struct {
			OK bool `json:"ok"`
		}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil || !result.OK {
			t.Fatalf("%s fixture did not report success: %v\n%s", args[0], err, buf.String())
		}
	}
	assertCodexSkillFixtureInstalled(t, packageDir, homeDir)
	return homeDir, repoDir
}

func lifecycleRecoverVisual(t *testing.T) string {
	t.Helper()
	return recoverCardRun(t, recoverEndgameState(t)).visual
}

func lifecycleRecoverEnvelope(t *testing.T) map[string]interface{} {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStoreCmd(t)
	t.Cleanup(func() { _ = tmpDir })
	store = s
	t.Setenv("AETHER_PLATFORM", "codex")
	if err := store.SaveJSON("COLONY_STATE.json", recoverEndgameState(t)); err != nil {
		t.Fatalf("write the fixture project: %v", err)
	}

	var buf bytes.Buffer
	stdout = &buf
	rootCmd.SetArgs([]string{"recover", "--json"})
	_ = rootCmd.Execute()

	var parsed map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("recover --json output is not valid JSON: %v\n%s", err, buf.String())
	}
	return parsed
}

func lifecycleStatusVisual(t *testing.T) string {
	t.Helper()
	newNextActionFixtureStore(t)
	if err := store.SaveJSON("COLONY_STATE.json", statusReadyState(t)); err != nil {
		t.Fatalf("write the fixture project: %v", err)
	}
	visual, _ := runStatusCommand(t, false)
	return visual
}

func lifecycleStatusEnvelope(t *testing.T) map[string]interface{} {
	t.Helper()
	newNextActionFixtureStore(t)
	if err := store.SaveJSON("COLONY_STATE.json", statusReadyState(t)); err != nil {
		t.Fatalf("write the fixture project: %v", err)
	}
	_, envelope := runStatusCommand(t, true)
	return envelope
}

// ---------------------------------------------------------------------------
// The named test
// ---------------------------------------------------------------------------

// TestEveryLifecycleCommandEndsWithNextAction is criterion 2's own named
// test. For each of the eleven commands criterion 2 names, it requires: the
// screen ends with the shared card; the machine-readable answer names the
// SAME command the card showed (comparing the two is what "the
// machine-readable version carries the identical information" means --
// asserting each half separately would pass even if they disagreed); that
// command resolves against the live command tree; and, on every recognised
// platform, the screen shows that platform's own spelling while the
// machine-readable value stays in the runtime form (S-01).
//
// It replaces, as a full subset, four single-assertion per-wave tests that
// checked exactly this and nothing more over an identical fixture:
// TestSealEnvelopeMatchesCard, TestUpdateEnvelopeCarriesTheCardsFields,
// TestStatusEnvelopeCarriesTheCardsFields and TestStatusEndsWithTheCard --
// all four deleted (see the plan's summary for exactly which). Every other
// per-wave test in the four lifecycle_card_*_test.go files asserts something
// this one does not (a plan-only variant, a part-finished build, a blocked
// check, situation-specific "keep" content, plain-English wording, an
// override reaching both halves) and is kept unchanged.
func TestEveryLifecycleCommandEndsWithNextAction(t *testing.T) {
	runIsolatedProcessTest(t, "TestEveryLifecycleCommandEndsWithNextAction", testEveryLifecycleCommandEndsWithNextAction)
}

func testEveryLifecycleCommandEndsWithNextAction(t *testing.T) {
	cases := lifecycleCoverageCases(t)

	// Anti-vacuity floor: a coverage test driving fewer commands than
	// criterion 2 actually names would pass while covering less than it
	// claims to -- exactly the failure mode this test exists to prevent.
	if len(cases) < len(criterionTwoLifecycleCommands) {
		t.Fatalf("this coverage test drives %d lifecycle commands; criterion 2 names %d: %v",
			len(cases), len(criterionTwoLifecycleCommands), criterionTwoLifecycleCommands)
	}
	seen := make(map[string]bool, len(criterionTwoLifecycleCommands))
	for _, name := range criterionTwoLifecycleCommands {
		seen[name] = false
	}
	for _, c := range cases {
		if _, ok := seen[c.name]; !ok {
			t.Fatalf("lifecycleCoverageCases names %q, which criterion 2 does not list: %v", c.name, criterionTwoLifecycleCommands)
		}
		seen[c.name] = true
	}
	for _, name := range criterionTwoLifecycleCommands {
		if !seen[name] {
			t.Fatalf("criterion 2 names %q, and this coverage test drives no case for it", name)
		}
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			visual := c.visual(t)
			marker := spacedTitle("What Next")
			if !strings.Contains(visual, marker) {
				t.Fatalf("%s does not end with the shared card at all:\n%s", c.name, visual)
			}
			envelope := c.envelope(t)
			envelopeCommands := envelopeNextActionCommands(envelope)
			if len(envelopeCommands) == 0 {
				t.Fatalf("%s emits neither a %q nor exact %q in its machine-readable answer; a wrapper cannot read the next step out of it",
					c.name, nextActionCommandKey, nextActionChoicesKey)
			}
			for _, envelopeCommand := range envelopeCommands {
				if !strings.Contains(visual, "`"+expectedCodexDisplayCommand(envelopeCommand)+"`") {
					t.Errorf("%s: the machine-readable answer names %q, which does not appear on the screen card",
						c.name, envelopeCommand)
				}
				if !strings.HasPrefix(envelopeCommand, "aether ") {
					t.Errorf("%s: the machine-readable command is %q, which is not something a wrapper can execute",
						c.name, envelopeCommand)
				}
				if _, ok := availableCommand(envelopeCommand); !ok {
					t.Errorf("%s recommends %q, which this build of the program does not have", c.name, envelopeCommand)
				}
			}
			if _, ok := envelope[nextActionRecommendationKey]; !ok {
				t.Errorf("%s carries no plain-English reason (%q) beside its command", c.name, nextActionRecommendationKey)
			}
			for _, alt := range envelopeAlternativeCommands(envelope) {
				if _, ok := availableCommand(alt); !ok {
					t.Errorf("%s offers %q as an alternative, which this build of the program does not have", c.name, alt)
				}
			}

		})
	}
	assertLifecyclePlatformCorrectness(t, cases)
	assertResumingPlatformCorrectness(t)
}

// envelopeNextActionCommands keeps a deliberate coequal action set coequal in
// this end-to-end assertion. Most lifecycle states carry next_command; an
// accepted plan instead carries two exact, registered next_choices and no
// invented primary. Both shapes are executable wrapper contracts.
func envelopeNextActionCommands(envelope map[string]interface{}) []string {
	if command := strings.TrimSpace(stringValue(envelope[nextActionCommandKey])); command != "" {
		return []string{command}
	}
	var commands []string
	switch choices := envelope[nextActionChoicesKey].(type) {
	case []LifecycleActionChoice:
		for _, choice := range choices {
			if command := strings.TrimSpace(choice.RuntimeCommand); command != "" {
				commands = append(commands, command)
			}
		}
	case []interface{}:
		for _, entry := range choices {
			if choice, ok := entry.(map[string]interface{}); ok {
				if command := strings.TrimSpace(stringValue(choice["runtime_command"])); command != "" {
					commands = append(commands, command)
				}
			}
		}
	}
	return commands
}

// envelopeAlternativeCommands reads the alternatives list out of an envelope,
// whichever of the two shapes it arrives in: the in-process
// []nextActionAlternative a renderer is handed directly, or the []interface{}
// the same value becomes after a JSON round trip out to a wrapper.
func envelopeAlternativeCommands(envelope map[string]interface{}) []string {
	var out []string
	switch alternatives := envelope[nextActionAlternativesKey].(type) {
	case []nextActionAlternative:
		for _, alt := range alternatives {
			out = append(out, alt.Command)
		}
	case []interface{}:
		for _, entry := range alternatives {
			if m, ok := entry.(map[string]interface{}); ok {
				out = append(out, stringValue(m["command"]))
			}
		}
	}
	return out
}

// lifecycleCoverageLabelForRendering maps nine of criterion 2's eleven names
// onto the label migratedSurfaceRenderings (cmd/lifecycle_card_agreement_test.go)
// already uses for the identical command, so the platform check below drives
// the exact same production renderer calls that map already proves correct,
// rather than a second, parallel way of building the same card. Resuming is
// deliberately absent -- migratedSurfaceRenderings excludes it too, for the
// reason recorded on that map (its real content depends on files this shared
// fixture does not seed) -- and gets its own check below instead.
var lifecycleCoverageLabelForRendering = map[string]string{
	"starting":          "starting a project",
	"discussing":        "talking it through",
	"planning":          "drawing up the plan",
	"building":          "building a phase",
	"continuing":        "checking the work",
	"pausing":           "pausing the project",
	"sealing/finishing": "sealing the project",
	"updating":          "updating the project",
	"recovering":        "recovering a project",
	"checking status":   "checking the status",
}

// assertLifecyclePlatformCorrectness is S-01 stated as a check for every
// migrated command. Each platform gets one fixture and one real rendering map;
// that map contains every migrated surface, so rebuilding it per command only
// repeats the same production rendering work without adding coverage.
func assertLifecyclePlatformCorrectness(t *testing.T, cases []lifecycleCoverageCase) {
	t.Helper()
	platforms := []struct{ platform, prefix string }{
		{"claude", "/ant-"},
		{"opencode", "/ant-"},
		{"codex", "$ant-"},
	}
	for _, tc := range platforms {
		newNextActionFixtureStore(t)
		t.Setenv("AETHER_PLATFORM", tc.platform)
		state := oneAgreementState(t)
		if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
			t.Fatalf("write the fixture project: %v", err)
		}
		renderings := migratedSurfaceRenderings(t, state)
		for _, c := range cases {
			label, ok := lifecycleCoverageLabelForRendering[c.name]
			if !ok {
				continue
			}
			rendered, ok := renderings[label]
			if !ok {
				t.Fatalf("%s has no rendering labelled %q in migratedSurfaceRenderings", c.name, label)
			}
			got := commandInClosing(t, c.name+" ("+tc.platform+")", rendered)
			if !strings.HasPrefix(got, tc.prefix) {
				t.Errorf("%s on %s shows %q; this platform's owner types commands beginning %q",
					c.name, tc.platform, got, tc.prefix)
			}
		}
	}
}

// assertResumingPlatformCorrectness is the same check for resuming, which
// migratedSurfaceRenderings deliberately excludes. Since a resolved answer's
// Command never depends on platform (S-01), rendering the ONE answer a real
// resuming run resolves for every platform is a real check of the card
// renderer's platform behaviour, not a second, parallel resolve.
//
// renderNextActionCardForPlatform's Next Up section is funnelled through
// renderNextUp, which resolves the platform from AETHER_PLATFORM itself
// (detectPlatform()) rather than from the platform argument -- the same
// established pattern every other platform-correctness test in this package
// relies on -- so the environment variable is set before each render rather
// than trusted to the parameter alone.
func assertResumingPlatformCorrectness(t *testing.T) {
	t.Helper()
	resumingCase := findLifecycleCase(t, sessionLifecycleSurfaces(), "resuming, canonical form")
	answer := runLifecycleSurface(t, resumingCase, false).answer
	commands := nextActionRuntimeCommands(answer)
	if len(commands) == 0 {
		t.Fatal("resuming resolved no executable next action")
	}

	platforms := []struct {
		platform string
		displays map[string]string
	}{
		{"claude", map[string]string{"aether build 1": "/ant-build 1", "aether run": "/ant-run"}},
		{"opencode", map[string]string{"aether build 1": "/ant-build 1", "aether run": "/ant-run"}},
		{"codex", map[string]string{"aether build 1": "$ant-build 1", "aether run": "aether run"}},
	}
	for _, tc := range platforms {
		t.Setenv("AETHER_PLATFORM", tc.platform)
		rendered := stripANSI(renderNextActionCardForPlatform(answer, tc.platform))
		for _, command := range commands {
			display, ok := tc.displays[command]
			if !ok {
				t.Fatalf("resuming on %s issued unexpected runtime command %q", tc.platform, command)
			}
			if !strings.Contains(rendered, "`"+display+"`") {
				t.Errorf("resuming on %s omits %q from its card:\n%s", tc.platform, display, rendered)
			}
		}
	}
}

func nextActionRuntimeCommands(answer nextAction) []string {
	if command := strings.TrimSpace(answer.Command); command != "" {
		return []string{command}
	}
	if answer.Projection == nil {
		return nil
	}
	commands := make([]string, 0, len(answer.Projection.NextAction.Choices))
	for _, choice := range answer.Projection.NextAction.Choices {
		if command := strings.TrimSpace(choice.RuntimeCommand); command != "" {
			commands = append(commands, command)
		}
	}
	return commands
}
