package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// This file drives the real hook-stop cobra command against REAL Claude Code
// fixtures captured from a genuine session (not hand-typed transcript shapes).
// See cmd/testdata/stop-hook/ and the implementation report for exactly how
// they were produced.

// loadStopHookFixturePayload loads a captured Stop payload fixture and
// rewrites its transcript_path to the absolute path of the paired transcript
// fixture, exactly as the plan requires -- the payload's own recorded path
// pointed at the real machine that captured it, not this test run.
func loadStopHookFixturePayload(t *testing.T, payloadFile, transcriptFile string) map[string]interface{} {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "stop-hook", payloadFile))
	if err != nil {
		t.Fatalf("read fixture payload %s: %v", payloadFile, err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal fixture payload %s: %v", payloadFile, err)
	}
	abs, err := filepath.Abs(filepath.Join("testdata", "stop-hook", transcriptFile))
	if err != nil {
		t.Fatalf("resolve transcript path: %v", err)
	}
	payload["transcript_path"] = abs
	return payload
}

func encodeStopHookStdin(t *testing.T, payload map[string]interface{}) string {
	t.Helper()
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal stop hook payload: %v", err)
	}
	return string(encoded)
}

func runStopHookFixture(t *testing.T, stdin string) (stdoutOut string, stderrOut string) {
	t.Helper()
	var buf bytes.Buffer
	var errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf
	setHookStdin(t, stdin)
	rootCmd.SetArgs([]string{"hook-stop"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("hook-stop returned error: %v", err)
	}
	return buf.String(), errBuf.String()
}

func stopHookBlockReason(t *testing.T, out string) (blocked bool, reason string) {
	t.Helper()
	trimmed := strings.TrimSpace(out)
	if trimmed == "" {
		return false, ""
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &result); err != nil {
		t.Fatalf("hook-stop produced non-JSON output: %q (%v)", out, err)
	}
	if result["decision"] != "block" {
		t.Fatalf("hook-stop produced JSON without a block decision: %v", result)
	}
	reasonText, _ := result["reason"].(string)
	return true, reasonText
}

// TestStopHookSendsBackAReplyThatHidTheScreen is the real "hid the screen"
// case, and it needed no synthesis at all: a genuine Claude Code session ran
// the REAL `/ant-status` menu command, Aether drew the colony-status screen,
// and the chat's actual first reply paraphrased it ("No colony here yet --
// this repo hasn't been started with Aether...") rather than showing it.
// cmd/testdata/stop-hook/menu-command-* is that exact session and its exact
// first Stop payload, captured for real (see the implementation report).
func TestStopHookSendsBackAReplyThatHidTheScreen(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	store = nil
	tracer = nil

	payload := loadStopHookFixturePayload(t, "menu-command-hide-screen-stop-payload.json", "menu-command-transcript.jsonl")
	out, _ := runStopHookFixture(t, encodeStopHookStdin(t, payload))

	blocked, reason := stopHookBlockReason(t, out)
	if !blocked {
		t.Fatalf("expected a block for a reply that hid the screen, got no decision at all (stdout=%q)", out)
	}
	if !strings.Contains(reason, "did not show it") {
		t.Fatalf("reason = %q, want it to name the hidden screen", reason)
	}
	if !strings.Contains(reason, "━━") {
		t.Fatalf("reason = %q, want it to point at the banner marker", reason)
	}
}

// TestStopHookAllowsAReplyThatShowsTheScreen is the paired real session: the
// same real `/ant-status` transcript, but the reply relays the screen. The
// "good" reply text is DERIVED, not typed: it is the genuine screen text the
// same chat produced on its own retry (captured in the second, stop_hook_active
// Stop payload of the same real session, after this hook's own block sent it
// back once) -- see the implementation report for exactly which field was
// swapped and why.
func TestStopHookAllowsAReplyThatShowsTheScreen(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	store = nil
	tracer = nil

	payload := loadStopHookFixturePayload(t, "menu-command-show-screen-stop-payload.json", "menu-command-transcript.jsonl")

	out, _ := runStopHookFixture(t, encodeStopHookStdin(t, payload))
	if strings.TrimSpace(out) != "" {
		t.Fatalf("expected a reply that shows the screen to be allowed silently, got %q", out)
	}
}

// TestStopHookOnlyAppliesWhereMenuCommandsAreUsed is the scoping rule: a
// session that never invoked one of Aether's own `/ant-…` menu commands is
// out of scope, however visual the Bash output it ran looks -- a developer
// piping `AETHER_OUTPUT_MODE=visual aether status` through grep in an ad-hoc
// debug session has no reason to paste a screen and must never be blocked.
// cmd/testdata/stop-hook/hide-screen-* is the original plain-prompt fixture
// (no /ant- command anywhere in it) whose reply hides the same kind of
// screen -- proving the scope rule, not the containment check, is what
// allows it here.
func TestStopHookOnlyAppliesWhereMenuCommandsAreUsed(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	store = nil
	tracer = nil

	payload := loadStopHookFixturePayload(t, "hide-screen-stop-payload.json", "hide-screen-transcript.jsonl")
	out, _ := runStopHookFixture(t, encodeStopHookStdin(t, payload))
	if strings.TrimSpace(out) != "" {
		t.Fatalf("a session with no /ant- menu command must never be blocked, even with a hidden screen; got %q", out)
	}
}

// TestStopHookOffSwitchDisablesTheScreenCheck proves the owner's off switch:
// AETHER_SCREEN_RELAY=off must allow a reply that would otherwise be blocked
// (the real /ant-status "hid it" case), and the check is case-insensitive
// and trims whitespace around the value.
func TestStopHookOffSwitchDisablesTheScreenCheck(t *testing.T) {
	for _, value := range []string{"off", "OFF", "  Off  "} {
		t.Run(value, func(t *testing.T) {
			saveGlobalsCmd(t)
			resetRootCmd(t)
			store = nil
			tracer = nil
			t.Setenv("AETHER_SCREEN_RELAY", value)

			payload := loadStopHookFixturePayload(t, "menu-command-hide-screen-stop-payload.json", "menu-command-transcript.jsonl")
			out, _ := runStopHookFixture(t, encodeStopHookStdin(t, payload))
			if strings.TrimSpace(out) != "" {
				t.Fatalf("AETHER_SCREEN_RELAY=%q must disable the screen check entirely; got %q", value, out)
			}
		})
	}

	t.Run("still blocks when unset", func(t *testing.T) {
		saveGlobalsCmd(t)
		resetRootCmd(t)
		store = nil
		tracer = nil

		payload := loadStopHookFixturePayload(t, "menu-command-hide-screen-stop-payload.json", "menu-command-transcript.jsonl")
		out, _ := runStopHookFixture(t, encodeStopHookStdin(t, payload))
		blocked, _ := stopHookBlockReason(t, out)
		if !blocked {
			t.Fatalf("with AETHER_SCREEN_RELAY unset, the same hidden-screen reply must still block; got %q", out)
		}
	})

	t.Run("other values do not disable it", func(t *testing.T) {
		saveGlobalsCmd(t)
		resetRootCmd(t)
		store = nil
		tracer = nil
		t.Setenv("AETHER_SCREEN_RELAY", "on")

		payload := loadStopHookFixturePayload(t, "menu-command-hide-screen-stop-payload.json", "menu-command-transcript.jsonl")
		out, _ := runStopHookFixture(t, encodeStopHookStdin(t, payload))
		blocked, _ := stopHookBlockReason(t, out)
		if !blocked {
			t.Fatalf("AETHER_SCREEN_RELAY=on must not disable the screen check; got %q", out)
		}
	})
}

// TestStopHookNeverBlocksTwice pins the pre-existing loop guard: a Stop event
// carrying stop_hook_active must never be re-blocked, screen or lifecycle.
func TestStopHookNeverBlocksTwice(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	store = nil
	tracer = nil

	payload := loadStopHookFixturePayload(t, "menu-command-hide-screen-stop-payload.json", "menu-command-transcript.jsonl")
	payload["stop_hook_active"] = true

	out, _ := runStopHookFixture(t, encodeStopHookStdin(t, payload))
	if strings.TrimSpace(out) != "" {
		t.Fatalf("stop_hook_active must always allow, even with a hidden screen owed; got %q", out)
	}
}

// TestStopHookIgnoresHelpersAndWorkers proves the screen check inherits the
// same "not the owner's own chat" exemptions the lifecycle check already
// had: an Aether-spawned worker (AETHER_WORKER_NAME) and a platform-reported
// sub-agent (agent_id) both allow silently, even with a hidden screen owed.
func TestStopHookIgnoresHelpersAndWorkers(t *testing.T) {
	t.Run("aether spawned worker", func(t *testing.T) {
		saveGlobalsCmd(t)
		resetRootCmd(t)
		store = nil
		tracer = nil
		t.Setenv("AETHER_WORKER_NAME", "Weld-32")
		t.Setenv("AETHER_WORKER_CASTE", "builder")

		payload := loadStopHookFixturePayload(t, "menu-command-hide-screen-stop-payload.json", "menu-command-transcript.jsonl")
		out, _ := runStopHookFixture(t, encodeStopHookStdin(t, payload))
		if strings.TrimSpace(out) != "" {
			t.Fatalf("an Aether-spawned worker must never be blocked; got %q", out)
		}
	})

	t.Run("platform reported sub-agent", func(t *testing.T) {
		saveGlobalsCmd(t)
		resetRootCmd(t)
		store = nil
		tracer = nil

		payload := loadStopHookFixturePayload(t, "menu-command-hide-screen-stop-payload.json", "menu-command-transcript.jsonl")
		payload["agent_id"] = "agent-42"
		out, _ := runStopHookFixture(t, encodeStopHookStdin(t, payload))
		if strings.TrimSpace(out) != "" {
			t.Fatalf("a Stop event carrying a non-empty agent_id must never be blocked; got %q", out)
		}
	})
}

// TestStopHookFailsOpenOnAnUnknownTranscript proves the fail-open rule for
// every unreadable/unrecognised transcript shape: a missing file, an empty
// file, and a file of garbage lines all allow rather than guess.
func TestStopHookFailsOpenOnAnUnknownTranscript(t *testing.T) {
	cases := []struct {
		name    string
		prepare func(t *testing.T) string // returns transcript path
	}{
		{
			name: "missing file",
			prepare: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "does-not-exist.jsonl")
			},
		},
		{
			name: "empty file",
			prepare: func(t *testing.T) string {
				path := filepath.Join(t.TempDir(), "empty.jsonl")
				if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
					t.Fatalf("write empty transcript: %v", err)
				}
				return path
			},
		},
		{
			name: "garbage lines",
			prepare: func(t *testing.T) string {
				path := filepath.Join(t.TempDir(), "garbage.jsonl")
				content := "not json\n{\"type\":\"user\"\n{{{{\nnull\n42\n"
				if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
					t.Fatalf("write garbage transcript: %v", err)
				}
				return path
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			saveGlobalsCmd(t)
			resetRootCmd(t)
			store = nil
			tracer = nil

			payload := loadStopHookFixturePayload(t, "menu-command-hide-screen-stop-payload.json", "menu-command-transcript.jsonl")
			payload["transcript_path"] = tc.prepare(t)

			out, _ := runStopHookFixture(t, encodeStopHookStdin(t, payload))
			if strings.TrimSpace(out) != "" {
				t.Fatalf("an unrecognisable transcript must fail open (allow); got %q", out)
			}
		})
	}
}

// TestStopHookLifecycleBlockStillWins asserts the ordering rule from the
// doc comment on hookStopCmd: when BOTH the lifecycle check and the screen
// check would block, the lifecycle check runs first and its answer is the
// only one that reaches the owner. The existing five lifecycle tests in
// cmd/hook_cmds_test.go stay unchanged and green; this test additionally
// proves the new screen check never overrides that decision.
func TestStopHookLifecycleBlockStillWins(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)

	goal := "lifecycle must win over screen relay"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{{ID: 1, Name: "In Progress", Status: colony.PhaseInProgress}},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	// A hidden-screen payload that WOULD independently trigger the screen
	// block, layered on top of a colony state that independently triggers
	// the lifecycle block.
	payload := loadStopHookFixturePayload(t, "menu-command-hide-screen-stop-payload.json", "menu-command-transcript.jsonl")
	out, _ := runStopHookFixture(t, encodeStopHookStdin(t, payload))

	blocked, reason := stopHookBlockReason(t, out)
	if !blocked {
		t.Fatalf("expected a lifecycle block, got no decision (stdout=%q)", out)
	}
	if !strings.Contains(reason, "aether continue") {
		t.Fatalf("reason = %q, want the lifecycle's own continue guidance, not the screen-relay reason", reason)
	}
	if strings.Contains(reason, "did not show it") {
		t.Fatalf("reason = %q, the screen-relay reason must never appear when the lifecycle check already blocked", reason)
	}
}

// TestStopHookScreenCheckDoesNotMutate matches the shape of the existing
// TestSessionStartHookDoesNotMutate lock: the screen check reads a transcript
// and a colony's saved state, but it must never write to either.
func TestStopHookScreenCheckDoesNotMutate(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)

	goal := "screen check must stay read-only"
	state := colony.ColonyState{
		Version: "1.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "Foundations", Status: colony.PhaseReady}}},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	payload := loadStopHookFixturePayload(t, "menu-command-hide-screen-stop-payload.json", "menu-command-transcript.jsonl")
	stdin := encodeStopHookStdin(t, payload)

	before := snapshotProjectDataTree(t, s.BasePath())
	first, _ := runStopHookFixture(t, stdin)
	second, _ := runStopHookFixture(t, stdin)
	after := snapshotProjectDataTree(t, s.BasePath())

	if !reflect.DeepEqual(before, after) {
		t.Errorf("the screen check changed the project's saved data.\nbefore: %v\nafter:  %v", before, after)
	}
	if first != second {
		t.Errorf("two identical screen checks produced different output.\nfirst:\n%s\nsecond:\n%s", first, second)
	}
	if strings.TrimSpace(first) == "" {
		t.Fatal("the fixture produced no block at all, so this test would pass vacuously")
	}
}

// TestBannerPredicateMatchesTheRenderer is the structural lock named in the
// plan: every renderBanner output's first line must satisfy
// isAetherBannerLine, across a representative spread of emoji and titles
// (single word, multi word, digits, an em dash), so the renderer and the
// Stop-hook checkpoint cannot silently drift apart. The plain divider line
// that normally follows a banner must NOT satisfy the predicate.
func TestBannerPredicateMatchesTheRenderer(t *testing.T) {
	cases := []struct {
		emoji string
		title string
	}{
		{"📊", "Colony Status"},
		{"🐜", "Next Up"},
		{"🧠", "Assumptions"},
		{"⚠", "Phase 12 Cannot Be Started"},
		{"🔨", "Build Phase 3"},
		{"🔨", "Build Phase 7 — Partly Done"},
		{"📦", "Install"},
		{"🥚", "Lay Eggs"},
	}

	for _, tc := range cases {
		rendered := renderBanner(tc.emoji, tc.title)
		lines := strings.Split(rendered, "\n")
		if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
			t.Fatalf("renderBanner(%q, %q) produced no first line", tc.emoji, tc.title)
		}
		if !isAetherBannerLine(lines[0]) {
			t.Errorf("isAetherBannerLine did not recognise renderBanner(%q, %q)'s own first line: %q", tc.emoji, tc.title, lines[0])
		}
	}

	if isAetherBannerLine(visualDividerStr()) {
		t.Errorf("the plain divider line must not satisfy isAetherBannerLine: %q", visualDividerStr())
	}
	if isAetherBannerLine("No colony initialized in this repo.") {
		t.Error("an ordinary content line must not satisfy isAetherBannerLine")
	}
	if isAetherBannerLine("") {
		t.Error("an empty line must not satisfy isAetherBannerLine")
	}
}

// TestBannerPredicateMatchesTheCardHeaderToo covers the second banner shape
// the program draws. The cards shown while helpers are sent out (spawn plan,
// wave start, finished summary) open with a THREE-bar header from
// renderOldStyleCeremonyHeader, not renderBanner's two-bar one. When the
// predicate knew only the two-bar shape, the biggest screens an owner sees
// (the finished card of a build or a check) owed nothing, and an earlier
// two-bar screen from the same reply was demanded in their place.
func TestBannerPredicateMatchesTheCardHeaderToo(t *testing.T) {
	for _, title := range []string{"Spawn Plan", "Wave 1", "Build Complete"} {
		header := strings.TrimRight(renderOldStyleCeremonyHeader("🔨", title), "\n")
		if !isAetherBannerLine(header) {
			t.Errorf("isAetherBannerLine did not recognise the card header %q", header)
		}
		if got := bannerLinesIn("running commentary\n" + header + "\nbody line\n"); len(got) != 1 {
			t.Errorf("bannerLinesIn found %d banner(s) in a card headed %q, want 1", len(got), header)
		}
	}
	for _, notABanner := range []string{
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━",
		"━━━ ━━━",
		"── Stage ──",
		"━ one bar is not a banner ━",
	} {
		if isAetherBannerLine(notABanner) {
			t.Errorf("isAetherBannerLine wrongly accepted %q", notABanner)
		}
	}
}
