package cmd

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
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

// directScreenFixtureCore is the minimum shape this file's Task 2 tests read
// out of the committed fixture: the command that drew the screen, and the
// screen text itself.
type directScreenFixtureCore struct {
	ToolInput struct {
		Command string `json:"command"`
	} `json:"tool_input"`
	ToolResponse struct {
		Stdout string `json:"stdout"`
	} `json:"tool_response"`
}

func loadDirectScreenFixtureCore(t *testing.T) directScreenFixtureCore {
	t.Helper()
	raw := loadDirectScreenFixture(t, "status-screen-payload.json")
	var fixture directScreenFixtureCore
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	return fixture
}

// TestDirectRouteNeverCutsALineInHalf is the length rule: an over-cap screen
// is delivered as its last whole lines behind the truncation notice, never a
// fragment of a line, and it reports itself incomplete.
func TestDirectRouteNeverCutsALineInHalf(t *testing.T) {
	fixture := loadDirectScreenFixtureCore(t)

	var b strings.Builder
	for b.Len() <= directScreenMessageCapBytes*2 {
		b.WriteString(fixture.ToolResponse.Stdout)
		b.WriteString("\n")
	}
	overLength := strings.TrimSuffix(b.String(), "\n")
	originalLines := strings.Split(overLength, "\n")

	message, complete, deliver := directScreenDelivery(fixture.ToolInput.Command, overLength)
	if !deliver {
		t.Fatal("expected delivery to stay true for an over-length screen")
	}
	if complete {
		t.Fatal("expected an over-length screen to report itself incomplete")
	}
	if got := len([]byte(message)); got > directScreenMessageCapBytes {
		t.Fatalf("message is %d bytes, want at most %d", got, directScreenMessageCapBytes)
	}
	if !strings.HasPrefix(message, directScreenTruncationNotice) {
		end := len(message)
		if end > 200 {
			end = 200
		}
		t.Fatalf("message does not start with the truncation notice: %q", message[:end])
	}

	body := strings.TrimPrefix(message, directScreenTruncationNotice)
	body = strings.TrimPrefix(body, "\n\n")

	originalSet := make(map[string]bool, len(originalLines))
	for _, l := range originalLines {
		originalSet[l] = true
	}

	var deliveredLines []string
	if body != "" {
		deliveredLines = strings.Split(body, "\n")
	}
	if len(deliveredLines) == 0 {
		t.Fatal("expected at least one whole line to survive truncation")
	}
	for _, line := range deliveredLines {
		if !originalSet[line] {
			t.Errorf("delivered line %q is not verbatim from the original screen", line)
		}
	}
	if last, want := deliveredLines[len(deliveredLines)-1], originalLines[len(originalLines)-1]; last != want {
		t.Errorf("message does not end with the original's own last line: got %q, want %q", last, want)
	}

	t.Run("a single line longer than the cap on its own", func(t *testing.T) {
		hugeLine := strings.Repeat("x", directScreenMessageCapBytes+1)
		screen := fixture.ToolResponse.Stdout + "\n" + hugeLine

		message, complete, deliver := directScreenDelivery(fixture.ToolInput.Command, screen)
		if !deliver {
			t.Fatal("expected delivery to stay true")
		}
		if complete {
			t.Fatal("expected incomplete")
		}
		if message != directScreenTruncationNotice {
			t.Fatalf("expected the notice alone with no line fragment, got %q", message)
		}
	})
}

// TestDirectRouteMessageIsValidJSONForEveryScreen proves the JSON envelope
// round-trips for a real screen and for a screen carrying an invalid UTF-8
// byte sequence: exactly one line of valid JSON is emitted either way, and
// decoding it returns text whose banner lines equal the original's.
func TestDirectRouteMessageIsValidJSONForEveryScreen(t *testing.T) {
	fixture := loadDirectScreenFixtureCore(t)
	invalidUTF8Screen := fixture.ToolResponse.Stdout + "\nsome text with an invalid byte: \xff\xfe end"

	cases := map[string]string{
		"real captured screen":                       fixture.ToolResponse.Stdout,
		"screen with an invalid UTF-8 byte sequence": invalidUTF8Screen,
	}

	for name, screen := range cases {
		t.Run(name, func(t *testing.T) {
			saveGlobalsCmd(t)

			message, _, deliver := directScreenDelivery(fixture.ToolInput.Command, screen)
			if !deliver {
				t.Fatal("expected delivery")
			}

			var buf bytes.Buffer
			stdout = &buf
			if err := emitDirectScreen(message); err != nil {
				t.Fatalf("emitDirectScreen: %v", err)
			}

			out := strings.TrimSpace(buf.String())
			lines := strings.Split(out, "\n")
			if len(lines) != 1 {
				t.Fatalf("expected exactly one line of stdout, got %d", len(lines))
			}

			var result struct {
				SystemMessage string `json:"systemMessage"`
			}
			if err := json.Unmarshal([]byte(lines[0]), &result); err != nil {
				t.Fatalf("stdout line is not valid JSON: %v", err)
			}

			wantBanners := bannerLinesIn(screen)
			gotBanners := bannerLinesIn(result.SystemMessage)
			if !reflect.DeepEqual(wantBanners, gotBanners) {
				t.Fatalf("banner lines differ after JSON round-trip.\nwant: %v\ngot:  %v", wantBanners, gotBanners)
			}
		})
	}
}

// TestDirectRouteIsSkippedForHelpersAndWorkers mirrors
// TestStopHookIgnoresHelpersAndWorkers for the direct route: an Aether-spawned
// worker and a platform-reported sub-agent must never receive the route.
func TestDirectRouteIsSkippedForHelpersAndWorkers(t *testing.T) {
	raw := loadDirectScreenFixture(t, "status-screen-payload.json")

	t.Run("aether spawned worker", func(t *testing.T) {
		saveGlobalsCmd(t)
		resetRootCmd(t)
		t.Setenv("AETHER_WORKER_NAME", "Weld-32")
		t.Setenv("AETHER_WORKER_CASTE", "builder")

		out, _ := runDirectScreenFixture(t, string(raw))
		if strings.TrimSpace(out) != "" {
			t.Fatalf("an Aether-spawned worker must never receive the direct route; got %q", out)
		}
	})

	t.Run("platform reported sub-agent", func(t *testing.T) {
		saveGlobalsCmd(t)
		resetRootCmd(t)

		stdin := mutateDirectScreenPayload(t, raw, func(p map[string]interface{}) {
			p["agent_id"] = "agent-42"
		})
		out, _ := runDirectScreenFixture(t, stdin)
		if strings.TrimSpace(out) != "" {
			t.Fatalf("a payload carrying a non-empty agent_id must never receive the direct route; got %q", out)
		}
	})
}

// TestDirectRouteOffSwitchStopsIt proves the owner's existing
// AETHER_SCREEN_RELAY=off switch also disables the direct route, not only the
// Stop-hook backstop.
func TestDirectRouteOffSwitchStopsIt(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	t.Setenv("AETHER_SCREEN_RELAY", "off")

	raw := loadDirectScreenFixture(t, "status-screen-payload.json")
	out, _ := runDirectScreenFixture(t, string(raw))
	if strings.TrimSpace(out) != "" {
		t.Fatalf("AETHER_SCREEN_RELAY=off must disable the direct route too; got %q", out)
	}
}

// TestDirectRouteWritesNothing matches TestStopHookScreenCheckDoesNotMutate's
// shape: the direct route reads its stdin payload and writes nothing to disk,
// with AETHER_HOOK_CAPTURE_FILE unset (the only thing that could make it
// write anything).
func TestDirectRouteWritesNothing(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	os.Unsetenv("AETHER_HOOK_CAPTURE_FILE")

	raw := loadDirectScreenFixture(t, "status-screen-payload.json")

	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "placeholder.txt"), []byte("keep"), 0o644); err != nil {
		t.Fatalf("seed temp project root: %v", err)
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWd) })

	before := snapshotProjectDataTree(t, tmpDir)
	runDirectScreenFixture(t, string(raw))
	after := snapshotProjectDataTree(t, tmpDir)

	if !reflect.DeepEqual(before, after) {
		t.Errorf("the direct route changed files under the project root.\nbefore: %v\nafter:  %v", before, after)
	}
}

// TestDirectRouteAndTheBackstopUseOneDecision is a structural guard: both the
// direct route (hookPostToolUseCmd) and the Stop-hook finish-check backstop
// (screenRelayBlockReason) must call directScreenDelivery -- the ONE decision
// -- and directScreenMessageCapBytes (the length rule's cap) must be
// referenced by exactly one function in the package, so a second, silently
// diverging copy of the length rule can never grow.
func TestDirectRouteAndTheBackstopUseOneDecision(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi fs.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("parse cmd package: %v", err)
	}

	identifierUsedIn := func(node ast.Node, name string) bool {
		found := false
		ast.Inspect(node, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && id.Name == name {
				found = true
			}
			return true
		})
		return found
	}

	var hookPostToolUseCmdValue ast.Node
	var screenRelayBlockReasonDecl ast.Node
	capReferencingFuncs := 0

	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				switch d := decl.(type) {
				case *ast.FuncDecl:
					if d.Name != nil && d.Name.Name == "screenRelayBlockReason" {
						screenRelayBlockReasonDecl = d
					}
					if d.Body != nil && identifierUsedIn(d.Body, "directScreenMessageCapBytes") {
						capReferencingFuncs++
					}
				case *ast.GenDecl:
					if d.Tok != token.VAR {
						continue
					}
					for _, spec := range d.Specs {
						vs, ok := spec.(*ast.ValueSpec)
						if !ok {
							continue
						}
						for i, name := range vs.Names {
							if name.Name == "hookPostToolUseCmd" && i < len(vs.Values) {
								hookPostToolUseCmdValue = vs.Values[i]
							}
						}
					}
				}
			}
		}
	}

	if hookPostToolUseCmdValue == nil {
		t.Fatal("hookPostToolUseCmd was not found -- this guard cannot check what it cannot see")
	}
	if screenRelayBlockReasonDecl == nil {
		t.Fatal("screenRelayBlockReason was not found -- this guard cannot check what it cannot see")
	}
	if !identifierUsedIn(hookPostToolUseCmdValue, "directScreenDelivery") {
		t.Error("hookPostToolUseCmd does not reach directScreenDelivery")
	}
	if !identifierUsedIn(screenRelayBlockReasonDecl, "directScreenDelivery") {
		t.Error("screenRelayBlockReason does not reach directScreenDelivery -- the backstop would be making its own, second copy of the delivery decision")
	}
	if capReferencingFuncs != 1 {
		t.Errorf("directScreenMessageCapBytes is referenced by %d function(s) in the package, want exactly 1 -- a second copy of the length rule is how the direct route and the backstop drift apart", capReferencingFuncs)
	}
}
