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

// extractLastFixtureToolResultText parses a real captured transcript fixture
// and returns the text of the LAST tool_result block a user-role message
// carries -- the same text a Bash call's own output put on screen. Deriving
// the "good" reply this way (rather than typing the banner text as a Go
// string literal) ties the test to what the fixture actually recorded.
func extractLastFixtureToolResultText(t *testing.T, transcriptFile string) string {
	t.Helper()
	path := filepath.Join("testdata", "stop-hook", transcriptFile)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read transcript fixture %s: %v", transcriptFile, err)
	}

	type block struct {
		Type    string          `json:"type"`
		Content json.RawMessage `json:"content"`
	}
	type entry struct {
		Type    string `json:"type"`
		Message struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		} `json:"message"`
	}

	var lastText string
	for _, raw := range strings.Split(string(data), "\n") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		var e entry
		if err := json.Unmarshal([]byte(raw), &e); err != nil {
			continue
		}
		if e.Type != "user" || e.Message.Role != "user" {
			continue
		}
		var blocks []block
		if json.Unmarshal(e.Message.Content, &blocks) != nil {
			continue
		}
		for _, b := range blocks {
			if b.Type != "tool_result" {
				continue
			}
			var text string
			if json.Unmarshal(b.Content, &text) == nil && text != "" {
				lastText = text
			}
		}
	}
	if lastText == "" {
		t.Fatalf("fixture transcript %s carried no tool_result text to derive a reply from", transcriptFile)
	}
	return lastText
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
// case: a genuine Claude Code session ran `aether status` in visual mode,
// got a screen back, and replied with the single word "done" -- never
// showing the owner what Aether drew. cmd/testdata/stop-hook/hide-screen-*
// is that exact session, captured for real (see the implementation report).
func TestStopHookSendsBackAReplyThatHidTheScreen(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	store = nil
	tracer = nil

	payload := loadStopHookFixturePayload(t, "hide-screen-stop-payload.json", "hide-screen-transcript.jsonl")
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
// same command, but the reply relayed the screen inside a fenced block. The
// "good" reply text is derived from the fixture's OWN tool result
// (extractLastFixtureToolResultText), never typed as a Go literal.
func TestStopHookAllowsAReplyThatShowsTheScreen(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	store = nil
	tracer = nil

	screenText := extractLastFixtureToolResultText(t, "show-screen-transcript.jsonl")
	payload := loadStopHookFixturePayload(t, "show-screen-stop-payload.json", "show-screen-transcript.jsonl")
	payload["last_assistant_message"] = "```\n" + screenText + "\n```"

	out, _ := runStopHookFixture(t, encodeStopHookStdin(t, payload))
	if strings.TrimSpace(out) != "" {
		t.Fatalf("expected a reply that shows the screen to be allowed silently, got %q", out)
	}
}

// TestStopHookNeverBlocksTwice pins the pre-existing loop guard: a Stop event
// carrying stop_hook_active must never be re-blocked, screen or lifecycle.
func TestStopHookNeverBlocksTwice(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	store = nil
	tracer = nil

	payload := loadStopHookFixturePayload(t, "hide-screen-stop-payload.json", "hide-screen-transcript.jsonl")
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

		payload := loadStopHookFixturePayload(t, "hide-screen-stop-payload.json", "hide-screen-transcript.jsonl")
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

		payload := loadStopHookFixturePayload(t, "hide-screen-stop-payload.json", "hide-screen-transcript.jsonl")
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

			payload := loadStopHookFixturePayload(t, "hide-screen-stop-payload.json", "hide-screen-transcript.jsonl")
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
	payload := loadStopHookFixturePayload(t, "hide-screen-stop-payload.json", "hide-screen-transcript.jsonl")
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

	payload := loadStopHookFixturePayload(t, "hide-screen-stop-payload.json", "hide-screen-transcript.jsonl")
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
