package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// loadDirectScreenFixture loads a real captured PostToolUse payload fixture
// from cmd/testdata/post-tool-use/ and returns its raw bytes, exactly as
// Claude Code delivered them to the hook's stdin (only absolute paths naming
// the capturing machine's home directory were redacted when the fixture was
// committed -- see the comment on ToolResponse in cmd/hook_cmds.go).
func loadDirectScreenFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "post-tool-use", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

// runDirectScreenFixture drives the real hook-post-tool-use cobra command
// against a given stdin payload, copying runStopHookFixture's shape
// (cmd/stop_hook_screen_test.go) for the sibling hook.
func runDirectScreenFixture(t *testing.T, stdin string) (stdoutOut string, stderrOut string) {
	t.Helper()
	var buf bytes.Buffer
	var errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf
	setHookStdin(t, stdin)
	rootCmd.SetArgs([]string{"hook-post-tool-use"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("hook-post-tool-use returned error: %v", err)
	}
	return buf.String(), errBuf.String()
}

// mutateDirectScreenPayload decodes a fixture's JSON, applies a mutation, and
// re-encodes it -- used by the "stays quiet" subtests to derive each no-op
// variant from the one real fixture rather than hand-typing five payloads.
func mutateDirectScreenPayload(t *testing.T, raw []byte, mutate func(map[string]interface{})) string {
	t.Helper()
	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal fixture payload: %v", err)
	}
	mutate(payload)
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal mutated payload: %v", err)
	}
	return string(encoded)
}

// TestDirectRouteHandsTheScreenToTheOwner is the tracer: a real captured
// payload (cmd/testdata/post-tool-use/status-screen-payload.json), fed to the
// real hook-post-tool-use command, produces exactly one line of stdout: JSON
// with a single key systemMessage whose value contains every banner line the
// screen drew, in order.
//
// Provenance: captured live, in this repository's own Claude Code session,
// via a temporary capture shim on the installed `aether` binary that
// intercepted only the `hook-post-tool-use` invocation (the real PostToolUse
// hook Claude Code fires after a genuine `AETHER_OUTPUT_MODE=visual aether
// status` Bash call). Every field's shape -- tool_response as {"stdout":
// ...}, tool_input.command, session_id, prompt_id -- is byte-real. Because
// the capturing session was itself a spawned gsd-executor subagent, the
// platform attached agent_id/agent_type identifying that subagent; those two
// fields were removed to represent a genuine top-level owner turn, which is
// not a guess -- this repo's own prior real top-level capture
// (cmd/testdata/stop-hook/menu-command-hide-screen-stop-payload.json) shows
// the identical key is simply ABSENT for a top-level session. Every other
// edit is the path redaction the plan specifies.
func TestDirectRouteHandsTheScreenToTheOwner(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)

	raw := loadDirectScreenFixture(t, "status-screen-payload.json")

	var fixture struct {
		ToolResponse struct {
			Stdout string `json:"stdout"`
		} `json:"tool_response"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	wantBanners := bannerLinesIn(fixture.ToolResponse.Stdout)
	if len(wantBanners) == 0 {
		t.Fatal("fixture's own screen carries no banner line -- this test would pass vacuously")
	}

	out, _ := runDirectScreenFixture(t, string(raw))
	trimmed := strings.TrimSpace(out)
	if trimmed == "" {
		t.Fatal("expected exactly one line of stdout carrying the screen, got none")
	}
	lines := strings.Split(trimmed, "\n")
	if len(lines) != 1 {
		t.Fatalf("expected exactly one line of stdout, got %d: %q", len(lines), out)
	}

	var result struct {
		SystemMessage string `json:"systemMessage"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &result); err != nil {
		t.Fatalf("stdout line is not valid JSON: %v (line=%q)", err, lines[0])
	}
	if result.SystemMessage == "" {
		t.Fatal("systemMessage was empty")
	}

	// directScreenDelivery hands back the screen exactly as it arrived
	// (unnormalized); bannerLinesIn returns whitespace-normalised banner
	// lines (the same normalisation the Stop-hook backstop compares against
	// -- cmd/hook_cmds.go's screenRelayBlockReason). Normalise the message
	// the same way before searching it, rather than the other way round,
	// since the plan requires the delivered message to be the screen
	// "exactly as it arrived".
	normalizedMessage := normalizeScreenWhitespace(result.SystemMessage)
	lastIdx := -1
	for _, banner := range wantBanners {
		idx := strings.Index(normalizedMessage, banner)
		if idx == -1 {
			t.Errorf("systemMessage is missing banner line %q", banner)
			continue
		}
		if idx < lastIdx {
			t.Errorf("banner line %q appeared out of order in systemMessage", banner)
		}
		lastIdx = idx
	}
}

// TestDirectRouteStaysQuietWhenThereIsNoScreen covers every "produces no
// output at all" case from the plan's behaviour block, each derived from the
// one real fixture by a single targeted mutation.
func TestDirectRouteStaysQuietWhenThereIsNoScreen(t *testing.T) {
	raw := loadDirectScreenFixture(t, "status-screen-payload.json")

	cases := []struct {
		name   string
		mutate func(map[string]interface{})
	}{
		{
			name: "tool name changed to Read",
			mutate: func(p map[string]interface{}) {
				p["tool_name"] = "Read"
			},
		},
		{
			name: "tool response stdout has no banner line",
			mutate: func(p map[string]interface{}) {
				tr, _ := p["tool_response"].(map[string]interface{})
				tr["stdout"] = "ordinary output with no drawn screen at all"
			},
		},
		{
			name: "shell command asks for machine output",
			mutate: func(p map[string]interface{}) {
				ti, _ := p["tool_input"].(map[string]interface{})
				ti["command"] = "AETHER_OUTPUT_MODE=json aether status"
			},
		},
		{
			name: "tool response is a bare JSON string",
			mutate: func(p map[string]interface{}) {
				p["tool_response"] = "just a plain string, not an object"
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			saveGlobalsCmd(t)
			resetRootCmd(t)

			stdin := mutateDirectScreenPayload(t, raw, tc.mutate)
			out, _ := runDirectScreenFixture(t, stdin)
			if strings.TrimSpace(out) != "" {
				t.Fatalf("expected no output, got %q", out)
			}
		})
	}

	t.Run("a payload with no owed screen at all produces no output", func(t *testing.T) {
		saveGlobalsCmd(t)
		resetRootCmd(t)

		stdin := `{"hook_event_name":"PostToolUse","tool_name":"Bash","tool_input":{"command":"echo hi"},"tool_response":{"stdout":"hi"}}`
		out, _ := runDirectScreenFixture(t, stdin)
		if strings.TrimSpace(out) != "" {
			t.Fatalf("expected no output for an ordinary non-Aether command, got %q", out)
		}
	})
}
