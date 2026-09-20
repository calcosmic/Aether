package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Exercise the very same pure resolver embedded in the disposable coordinator.
// These captures model the final165 spawn call/activity/result and metadata,
// not worker prose. They never launch a native worker.
func TestCodexNativeGapAttribution(t *testing.T) {
	t.Run("literal-command-attribution", TestCodexNativeGapLiteralCommandAttribution)
	parent := map[string]any{"type": "session_meta", "payload": map[string]any{"id": "parent", "cwd": "/fixture"}}
	child := map[string]any{"type": "session_meta", "payload": map[string]any{"id": "child-uuid", "parent_thread_id": "parent", "cwd": "/fixture", "agent_path": "/root/brick", "agent_role": "aether-builder"}}
	turn := map[string]any{"turn_id": "turn"}
	call := map[string]any{"type": "response_item", "payload": map[string]any{"type": "function_call", "name": "spawn_agent", "call_id": "spawn", "arguments": `{"task_name":"brick","agent_type":"aether-builder","message":"opaque"}`, "internal_chat_message_metadata_passthrough": turn}}
	activity := map[string]any{"type": "event_msg", "payload": map[string]any{"type": "item_completed", "thread_id": "parent", "turn_id": "turn", "item": map[string]any{"type": "SubAgentActivity", "id": "spawn", "kind": "started", "agent_thread_id": "child-uuid", "agent_path": "/root/brick"}}}
	output := map[string]any{"type": "response_item", "payload": map[string]any{"type": "function_call_output", "call_id": "spawn", "output": `{"task_name":"/root/brick"}`, "internal_chat_message_metadata_passthrough": turn}}
	encode := func(v any) []byte {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	for _, mode := range []string{"valid", "uuid_input", "missing", "ambiguous", "wrong_thread", "wrong_turn", "reused_activity", "duplicate_result", "wrong_metadata", "wrong_role", "result_before_call"} {
		t.Run(mode, func(t *testing.T) {
			events := []any{parent, call, activity, output}
			metas := []any{child}
			raw := encode(events)
			meta := encode(metas)
			target := "/root/brick"
			switch mode {
			case "uuid_input":
				target = "child-uuid"
			case "missing":
				raw = encode([]any{parent, call, output})
			case "ambiguous":
				meta = encode([]any{child, child})
			case "wrong_thread":
				raw = bytes.Replace(raw, []byte(`"thread_id":"parent"`), []byte(`"thread_id":"foreign"`), 1)
			case "wrong_turn":
				raw = bytes.Replace(raw, []byte(`"turn_id":"turn"`), []byte(`"turn_id":"foreign"`), 1)
			case "reused_activity":
				raw = encode([]any{parent, call, activity, activity, output})
			case "duplicate_result":
				raw = encode([]any{parent, call, activity, output, output})
			case "wrong_metadata":
				meta = bytes.Replace(meta, []byte(`"parent_thread_id":"parent"`), []byte(`"parent_thread_id":"foreign"`), 1)
			case "wrong_role":
				meta = bytes.Replace(meta, []byte(`aether-builder`), []byte(`aether-watcher`), 1)
			case "result_before_call":
				raw = encode([]any{parent, output, call, activity})
			}
			script := nativeGapResolverScript(t)
			input := []byte(`{"events":` + string(raw) + `,"metadata":` + string(meta) + `,"target":"` + target + `","parent":"parent","cwd":"/fixture"}`)
			cmd := exec.Command("python3", "-c", script+"\nimport sys,json\nx=json.load(sys.stdin)\nprint(json.dumps(resolve_native_child(**x)))\n")
			cmd.Stdin = bytes.NewReader(input)
			out, err := cmd.CombinedOutput()
			if mode == "valid" || mode == "uuid_input" {
				if err != nil || !bytes.Contains(out, []byte(`"child_id": "child-uuid"`)) || !bytes.Contains(out, []byte(`"task_path": "/root/brick"`)) {
					t.Fatalf("valid mapping: %v %s", err, out)
				}
			} else if err == nil {
				t.Fatalf("%s accepted: %s", mode, out)
			}
		})
	}
	t.Run("empty_result_refuses_without_mutation", func(t *testing.T) {
		_, request := nativeBoundForTest(t)
		before := nativeJournalBytes(t)
		for _, result := range []*internalWorkerResult{nil, {}} {
			request.Result = result
			if _, err := runCodexNativeWorker("record", nativeRequestPath(t, request)); err == nil {
				t.Fatal("empty result accepted")
			}
			if !bytes.Equal(before, nativeJournalBytes(t)) {
				t.Fatal("empty result mutated journal")
			}
		}
	})
	t.Run("equal_reservation_receipt_and_conflicting_key", func(t *testing.T) {
		_, request := nativeAdmissionFixture(t)
		first, err := runCodexNativeWorker("reserve", nativeRequestPath(t, request))
		if err != nil {
			t.Fatal(err)
		}
		before := nativeJournalBytes(t)
		second, err := runCodexNativeWorker("reserve", nativeRequestPath(t, request))
		if err != nil || !second.Replay {
			t.Fatalf("reservation replay: %v", err)
		}
		a, _ := json.Marshal(first.Receipt)
		b, _ := json.Marshal(second.Receipt)
		if !bytes.Equal(a, b) || !bytes.Equal(before, nativeJournalBytes(t)) {
			t.Fatal("reservation replay minted receipt or wrote journal")
		}
		request.HostSessionID = "different-host"
		if _, err := runCodexNativeWorker("reserve", nativeRequestPath(t, request)); err == nil {
			t.Fatal("conflicting equal key accepted")
		}
		if !bytes.Equal(before, nativeJournalBytes(t)) {
			t.Fatal("conflicting equal key wrote journal")
		}
	})
	t.Run("runtime_identity_and_empty_controls", func(t *testing.T) {
		TestCodexNativeWorkerTerminal(t)
	})
	t.Run("parent_only_edits_and_unclassified_writes", TestCodexNativeEvidenceDerivation)
	t.Run("unique_command_events", TestCodexNativeGapParentCoordination)
}

func TestCodexNativeGapLiteralCommandAttribution(t *testing.T) {
	t.Run("quoted-bytes-unchanged", func(t *testing.T) {
		for _, tc := range []struct{ input, want string }{
			{`text(await tools.exec_command({cmd:'printf "café 日本語 é"',workdir:'/fixture'}));`, `printf "café 日本語 é"`},
			{`text(await tools.exec_command({cmd:'printf \'quoted\' \\path\n',workdir:'/fixture'}));`, "printf 'quoted' \\path\n"},
			{`text(await tools.exec_command({cmd:'printf "{cmd: untouched}"',workdir:'/fixture'}));`, `printf "{cmd: untouched}"`},
			{`text(await tools.exec_command({cmd:'printf "\u65e5\u672c\u8a9e \ud83d\ude80"',workdir:'/fixture'}));`, `printf "日本語 🚀"`},
		} {
			got, ok := nativeCodeModeCommands(tc.input, "/fixture")
			if !ok || len(got) != 1 || got[0].Command != tc.want {
				t.Fatalf("literal bytes changed: got=%+v ok=%v want=%q", got, ok, tc.want)
			}
		}
	})
	for _, input := range []string{
		`text(await tools.exec_command({cmd:'aether spawn-log --help',max_output_tokens:2000}));`,
		`text(await tools.exec_command({'cmd':'aether spawn-log --name Brick-46 --caste builder --parent Queen --phase 1 --task "Fix Clamp"',workdir:'/fixture',max_output_tokens:1000}));`,
		`text(await tools.exec_command({cmd:"aether status"})); text(await tools.exec_command({cmd:'aether spawn-complete --name Brick-46 --status completed --summary "Fixed Clamp"'}));`,
		`const result = await tools.exec_command({cmd:'aether status',workdir:'/fixture'}); text(result);`,
	} {
		t.Run(input, func(t *testing.T) {
			if _, ok := nativeCodeModeCommands(input, "/fixture"); !ok {
				t.Fatal("observed literal grammar rejected")
			}
		})
	}
	for _, object := range []string{
		`{cmd:'aether status' + suffix}`, `{cmd:command}`, "{cmd:`aether status`}",
		`{...defaults,cmd:'aether status'}`, `{['cmd']:'aether status'}`, `{cmd:'aether status',cmd:'touch clamp.go'}`,
		`{"cmd":"aether status","cmd":"touch clamp.go"}`, `{cmd:'aether status',workdir:'/fixture',workdir:'/other'}`,
		`{cmd:'aether status',max_output_tokens:expression()}`, `{cmd:'aether status',max_output_tokens:{x:1}}`,
		`{cmd:'aether status',yield_time_ms:NaN}`, `{cmd:'aether status',unknown:1}`, `{cmd:'aether status',__proto__:{}}`,
		`{cmd:'aether status',"\u0063md":"touch clamp.go"}`, `{cmd:'printf "\ud800"'}`, `{cmd:'printf "\udc00"'}`,
		`{cmd:'printf "\ud800\u0041"'}`, `{cmd:'aether status',max_output_tokens:01}`, `{cmd:'aether status',max_output_tokens:1.0}`,
	} {
		t.Run("reject-"+object, func(t *testing.T) {
			if _, ok := nativeCodeModeCommands("text(await tools.exec_command("+object+"));", "/fixture"); ok {
				t.Fatal("nonliteral or ambiguous object accepted")
			}
		})
	}
	for _, input := range []string{
		`text(await tools.exec_command({cmd:'aether status'})); mutate();`,
		`text(await other.exec_command({cmd:'aether status'}));`,
		`const x=await tools.exec_command({cmd:'aether status'});text(y);`,
		`text(await tools.exec_command({cmd:'aether status'})); text(await tools.apply_patch('x'));`,
	} {
		t.Run("reject-extra-"+input, func(t *testing.T) {
			if _, ok := nativeCodeModeCommands(input, "/fixture"); ok {
				t.Fatal("unsupported execution accepted")
			}
		})
	}
	// Syntax decoding must not itself authorize a parent edit or required check.
	r := codexNativeLiveReceipt{SchemaVersion: "codex-native-tracer/v2", SessionID: "parent", FixtureRoot: "/fixture"}
	for _, cmd := range []string{"go test ./... -json -count=1", "touch unrelated", "python3 -c 'print(1)'", "echo wrong > clamp.go"} {
		if nativeParentCoordinationCommand(&r, []string{"/bin/sh", "-c", cmd}, "/fixture") {
			t.Fatalf("parent substitution accepted: %s", cmd)
		}
	}
}

func TestCodexNativeCodeModeOutputProjection(t *testing.T) {
	// Actual Plan 26 wrapper shape, with disposable paths shortened for this
	// deterministic decoder control. Printing output never supplies exit proof.
	input := `const r = await tools.exec_command({"cmd":"aether command-guide build --platform codex","workdir":"/fixture","yield_time_ms":30000,"max_output_tokens":30000}); text(r.output);`
	for _, tc := range []struct {
		name, input string
		want        bool
	}{
		{"observed", input, true},
		{"same-variable-renamed", strings.Replace(strings.Replace(input, "const r =", "const result =", 1), "text(r.output)", "text(result.output)", 1), true},
		{"single-quoted-command", `const result = await tools.exec_command({cmd:'aether status',workdir:'/fixture'}); text(result.output);`, true},
		{"different-variable", strings.Replace(input, "text(r.output)", "text(other.output)", 1), false},
		{"different-property", strings.Replace(input, ".output", ".exit_code", 1), false},
		{"computed-property", strings.Replace(input, ".output", `["output"]`, 1), false},
		{"rewritten-output", strings.Replace(input, "r.output", `r.output + "fabricated"`, 1), false},
		{"method-call", strings.Replace(input, "r.output", "r.output.trim()", 1), false},
		{"reassignment", strings.Replace(input, " text(", ` r.output = "forged"; text(`, 1), false},
		{"extra-command", input + `text(await tools.exec_command({cmd:'touch clamp.go',workdir:'/fixture'}));`, false},
		{"preceding-command", `text(await tools.exec_command({cmd:'aether status',workdir:'/fixture'}));` + input, false},
		{"extra-js", input + " mutate();", false},
		{"unawaited", strings.Replace(input, "await ", "", 1), false},
		{"other-tool", strings.Replace(input, "tools.exec_command", "other.exec_command", 1), false},
		{"computed-command", strings.Replace(input, `"aether command-guide build --platform codex"`, `command`, 1), false},
		{"duplicate-key", strings.Replace(input, `"cmd":`, `"cmd":"touch clamp.go","cmd":`, 1), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			commands, ok := nativeCodeModeCommands(tc.input, "/fixture")
			if ok != tc.want {
				t.Fatalf("projection accepted=%v want=%v: %+v", ok, tc.want, commands)
			}
			command, cwd, single := nativeCodeModeCommand(tc.input)
			if single != tc.want || (single && (len(commands) != 1 || command != commands[0].Command || cwd != commands[0].Cwd)) {
				t.Fatalf("single-command decoder diverged: command=%q cwd=%q accepted=%v", command, cwd, single)
			}
		})
	}
	if _, ok := nativeCodeModeCommands(strings.Replace(input, `,"workdir":"/fixture"`, "", 1), ""); ok {
		t.Fatal("output projection manufactured missing workspace attribution")
	}
}

func TestCodexNativeGapDocsSearch(t *testing.T) {
	observed := `rg "spawn-log" .aether -g '*.md' -g '*.json' -g '*.yaml'`
	for _, tc := range []struct {
		name, command, cwd string
		want               bool
	}{
		{"observed", observed, "/fixture", true},
		{"help-already-supported", "aether spawn-log --help", "/fixture", true},
		{"foreign-cwd", observed, "/other", false},
		{"missing-cwd", observed, "", false},
		{"foreign-root", strings.Replace(observed, ".aether", "/other/.aether", 1), "/fixture", false},
		{"escaping-root", strings.Replace(observed, ".aether", "../.aether", 1), "/fixture", false},
		{"preprocessor", observed + " --pre mutate", "/fixture", false},
		{"follow-symlinks", observed + " --follow", "/fixture", false},
		{"hidden-files", observed + " --hidden", "/fixture", false},
		{"extra-command", observed + "; touch clamp.go", "/fixture", false},
		{"redirect", observed + " > clamp.go", "/fixture", false},
		{"expansion", strings.Replace(observed, "spawn-log", "$(touch clamp.go)", 1), "/fixture", false},
		{"glob-expansion", strings.Replace(observed, "'*.md'", "*.md", 1), "/fixture", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := codexNativeLiveReceipt{FixtureRoot: "/fixture"}
			if got := nativeParentCoordinationCommand(&r, []string{"/bin/zsh", "-lc", tc.command}, tc.cwd); got != tc.want {
				t.Fatalf("docs query accepted=%v want=%v", got, tc.want)
			}
		})
	}
}

// Explicit immutable replay of the observed single-quoted call shapes. This
// diagnoses parser behavior only and never upgrades the original receipt.
func TestCodexNativeGapObservedSingleQuoteCapture(t *testing.T) {
	root := os.Getenv("AETHER_CODEX_GAP_CAPTURE")
	if root == "" {
		t.Skip("explicit immutable capture required")
	}
	rawReceipt, err := os.ReadFile(filepath.Join(root, "receipt.json"))
	if err != nil {
		t.Fatal(err)
	}
	var receipt codexNativeLiveReceipt
	if json.Unmarshal(rawReceipt, &receipt) != nil {
		t.Fatal("invalid original receipt")
	}
	paths, err := filepath.Glob(filepath.Join(root, "home/.codex/sessions/*/*/*/*.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	wanted := map[string]bool{"call_ymn3RdT67KSPzOXx22HtWAWe": false, "call_WkaJC0j7lB5KyMQ74kVWOoPO": false, "call_xTUsSlBu4mtuSnP3Q7Cf8FEf": false}
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var meta nativeHostEvent
		if json.Unmarshal(bytes.SplitN(raw, []byte{'\n'}, 2)[0], &meta) == nil && meta.Type == "session_meta" && meta.Payload.ID == receipt.SessionID {
			derived := receipt
			derived.ParentSubstitution = false
			derived.ParentUnclassified = nil
			nativeInspectParentEvents(&derived, raw)
			if derived.ParentSubstitution || len(derived.ParentUnclassified) != 0 {
				t.Errorf("parent remains unclassified: %v", derived.ParentUnclassified)
			}
		}
		for _, line := range bytes.Split(raw, []byte{'\n'}) {
			var w nativeCodeModeWire
			if json.Unmarshal(line, &w) != nil || w.Type != "response_item" || w.Payload.Type != "custom_tool_call" {
				continue
			}
			if _, ok := wanted[w.Payload.CallID]; !ok {
				continue
			}
			commands, ok := nativeCodeModeCommands(w.Payload.Input, receipt.FixtureRoot)
			if !ok || !nativeCorroboratedBatch(raw, receipt.SessionID, w.Payload.CallID, commands) {
				t.Errorf("actual call %s rejected", w.Payload.CallID)
				continue
			}
			wanted[w.Payload.CallID] = true
			t.Logf("actual raw=%s line=%s call=%s", lifecycleDigest(raw), lifecycleDigest(line), w.Payload.CallID)
		}
	}
	for call, seen := range wanted {
		if !seen {
			t.Errorf("missing accepted actual call %s", call)
		}
	}
}

func nativeGapResolverScript(t *testing.T) string {
	t.Helper()
	start := strings.Index(nativeFixtureCoordinator, "def resolve_native_child(")
	end := strings.Index(nativeFixtureCoordinator, "def latest_native_terminal(")
	if start < 0 || end <= start {
		t.Fatal("coordinator lacks causal task-path resolver")
	}
	return "import json\n" + nativeFixtureCoordinator[start:end]
}

// Optional read-only replay preserves original final165 files and identity.
func TestCodexNativeGapCapturedAttribution(t *testing.T) {
	root := os.Getenv("AETHER_CODEX_GAP_CAPTURE")
	if root == "" {
		t.Skip("explicit immutable capture path required")
	}
	paths, err := filepath.Glob(filepath.Join(root, "home/.codex/sessions/*/*/*/*.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var events []any
	var metas []any
	var parent string
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		lines := bytes.Split(bytes.TrimSpace(raw), []byte{'\n'})
		var first map[string]any
		if err := json.Unmarshal(lines[0], &first); err != nil {
			t.Fatal(err)
		}
		meta := first["payload"].(map[string]any)
		if meta["parent_thread_id"] == nil {
			parent = meta["id"].(string)
			for _, line := range lines {
				var e any
				if err := json.Unmarshal(line, &e); err != nil {
					t.Fatal(err)
				}
				events = append(events, e)
			}
		} else {
			metas = append(metas, first)
		}
	}
	input, _ := json.Marshal(map[string]any{"events": events, "metadata": metas, "target": "/root/brick_46", "parent": parent, "cwd": events[0].(map[string]any)["payload"].(map[string]any)["cwd"]})
	cmd := exec.Command("python3", "-c", nativeGapResolverScript(t)+"\nimport sys\nprint(json.dumps(resolve_native_child(**json.load(sys.stdin))))\n")
	cmd.Stdin = bytes.NewReader(input)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("captured final165: %v %s", err, out)
	}
	if !bytes.Contains(out, []byte("01a0afbd-0935-7512-ad12-825dd8c1c3ff")) {
		t.Fatalf("wrong captured child: %s", out)
	}
	t.Log(string(out))
	r := codexNativeLiveReceipt{BoundHostSessionID: parent, ChildID: "01a0afbd-0935-7512-ad12-825dd8c1c3ff", FixtureRoot: filepath.Join(root, "repository")}
	nativeCollectChildIdentity(&r, filepath.Join(root, "home"))
	if !r.ChildIdentityCorroborated || r.ChildTaskPath != "/root/brick_46" {
		t.Fatalf("observer did not retain both identities: %+v", r)
	}
}

func TestCodexNativeGapParentCoordination(t *testing.T) {
	t.Run("literal-display-env", nativeParentDisplayEnvControls)
	t.Run("owned-json-field-ranges", nativeParentJSONFieldControls)
	for _, wrapper := range []string{"direct", "output-projection", "json-result", "output-exit", "all-output", "all-json"} {
		modes := []string{"valid", "file_uri", "actual_command_failure", "missing", "wrong_thread", "wrong_turn", "wrong_cwd", "wrong_argv", "reused_event", "duplicate_call", "missing_output", "wrong_output_call", "wrong_output_turn", "failed_script"}
		if wrapper == "output-exit" {
			modes = append(modes, "legacy_single_block", "missing_exit_block", "reordered_blocks", "forged_exit", "forged_output", "extra_block", "wrong_event_output_field")
		}
		for _, mode := range modes {
			t.Run(wrapper+"/"+mode, func(t *testing.T) {
				var raw []byte
				add := func(v any) { b, _ := json.Marshal(v); raw = append(raw, append(b, '\n')...) }
				add(map[string]any{"type": "session_meta", "payload": map[string]any{"id": "parent", "cwd": "/fixture"}})
				input := `text(await tools.exec_command({cmd:"aether status",workdir:"/fixture"}));`
				if wrapper == "output-projection" {
					input = `const r = await tools.exec_command({cmd:"aether status",workdir:"/fixture"}); text(r.output);`
				}
				if wrapper == "json-result" {
					input = `const r = await tools.exec_command({cmd:"aether status",workdir:"/fixture"}); text(JSON.stringify(r));`
				}
				if wrapper == "output-exit" {
					input = "const r = await tools.exec_command({cmd:\"aether status\",workdir:\"/fixture\"}); text(r.output); text(`\\nEXIT_CODE=${r.exit_code}`);"
				}
				if wrapper == "all-output" || wrapper == "all-json" {
					input = `const results = await Promise.all([tools.exec_command({cmd:"aether status",workdir:"/fixture"})]); for (const r of results) text(r.output);`
					if wrapper == "all-json" {
						input = strings.Replace(input, "text(r.output)", "text(JSON.stringify(r))", 1)
					}
				}
				call := map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call", "name": "exec", "call_id": "call", "input": input, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": "turn"}}}
				add(call)
				if mode == "duplicate_call" {
					add(call)
				}
				thread, turn, cwd, command := "parent", "turn", "/fixture", "aether status"
				switch mode {
				case "file_uri":
					cwd = "file:///fixture"
				case "wrong_thread":
					thread = "sibling"
				case "wrong_turn":
					turn = "other"
				case "wrong_cwd":
					cwd = "/other"
				case "wrong_argv":
					command = "aether pause"
				}
				status, exit := "completed", 0
				if mode == "actual_command_failure" {
					status, exit = "failed", 1
				}
				event := map[string]any{"type": "event_msg", "payload": map[string]any{"type": "item_completed", "thread_id": thread, "turn_id": turn, "item": map[string]any{"type": "CommandExecution", "id": "shell", "status": status, "cwd": cwd, "command": []string{"/bin/sh", "-c", command}, "exit_code": exit, "aggregated_output": "actual output text"}}}
				if mode == "wrong_event_output_field" {
					item := event["payload"].(map[string]any)["item"].(map[string]any)
					delete(item, "aggregated_output")
					item["output"] = "actual output text"
				}
				if mode != "missing" {
					add(event)
				}
				outputCall, outputTurn, outputText := "call", "turn", "Script completed\nactual output text"
				switch mode {
				case "wrong_output_call":
					outputCall = "other-call"
				case "wrong_output_turn":
					outputTurn = "other-turn"
				case "failed_script":
					outputText = "Script failed\n"
				}
				if mode != "missing_output" {
					blocks := []any{map[string]any{"type": "input_text", "text": outputText}}
					if wrapper == "output-exit" && mode != "legacy_single_block" {
						// Two text calls emit two distinct blocks after the host header.
						// Keep this literal fixture independent of the presentation renderer.
						header := "Script completed\n"
						if mode == "failed_script" {
							header = "Script failed\n"
						}
						blocks = []any{map[string]any{"type": "input_text", "text": header}, map[string]any{"type": "input_text", "text": "actual output text"}, map[string]any{"type": "input_text", "text": "\nEXIT_CODE=" + strconv.Itoa(exit)}}
						switch mode {
						case "missing_exit_block":
							blocks = blocks[:2]
						case "reordered_blocks":
							blocks[1], blocks[2] = blocks[2], blocks[1]
						case "forged_exit":
							blocks[2] = map[string]any{"type": "input_text", "text": "\nEXIT_CODE=1"}
						case "forged_output":
							blocks[1] = map[string]any{"type": "input_text", "text": "different output"}
						case "extra_block":
							blocks = append(blocks, blocks[2])
						}
					}
					add(map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call_output", "call_id": outputCall, "output": blocks, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": outputTurn}}})
				}
				if mode == "reused_event" {
					add(event)
				}
				r := codexNativeLiveReceipt{SchemaVersion: "codex-native-tracer/v2", SessionID: "parent", FixtureRoot: "/fixture"}
				nativeInspectParentEvents(&r, raw)
				if r.ParentSubstitution != (mode != "valid" && mode != "file_uri" && mode != "actual_command_failure") || r.ChecksPassed {
					t.Fatalf("mode %s classification: %+v", mode, r.ParentUnclassified)
				}
			})
		}
	}
}

func nativeParentDisplayEnvControls(t *testing.T) {
	commands := []string{
		"env AETHER_OUTPUT_MODE=visual aether status",
		"env AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony spawn-plan --workflow build --manifest-file /coord/manifest-envelope.json",
		"env AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony wave-start --workflow build --manifest-file /coord/manifest-envelope.json --execution-wave 11",
		"env AETHER_OUTPUT_MODE=visual aether ceremony worker-complete --workflow build --worker-file /coord/child-terminal.jsonl",
		"env AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow build --completion-file .aether/data/build/phase-1/completion.json",
	}
	r := codexNativeLiveReceipt{FixtureRoot: "/fixture"}
	for _, command := range commands {
		if !nativeParentCoordinationCommand(&r, []string{"/bin/zsh", "-lc", command}, "/fixture") {
			t.Fatalf("literal renderer rejected: %s", command)
		}
		if nativeParentCoordinationCommand(&r, []string{"/bin/zsh", "-lc", command}, "/other") || nativeChildCommandAllowed(r, []string{"/bin/zsh", "-lc", command}) {
			t.Fatalf("renderer escaped parent fixture-only classification: %s", command)
		}
	}
	for _, prefix := range []string{"AETHER_OUTPUT_MODE=json", "AETHER_OUTPUT_MODE=visual", "AETHER_FORCE_COLOR=0", "AETHER_FORCE_COLOR=1", "AETHER_OUTPUT_MODE=json AETHER_FORCE_COLOR=0", "AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual"} {
		t.Run("value/"+prefix, func(t *testing.T) {
			if !nativeParentCoordinationCommand(&r, []string{"/bin/sh", "-c", "env " + prefix + " aether status"}, "/fixture") {
				t.Fatal("accepted display value rejected")
			}
		})
	}
	for _, command := range []string{
		"env aether status", "env -i AETHER_OUTPUT_MODE=visual aether status", "env -u PATH AETHER_OUTPUT_MODE=visual aether status", "env -S 'AETHER_OUTPUT_MODE=visual aether status'", "env -- AETHER_OUTPUT_MODE=visual aether status",
		"env PATH=/other AETHER_OUTPUT_MODE=visual aether status", "env LD_PRELOAD=/other AETHER_OUTPUT_MODE=visual aether status", "env DYLD_INSERT_LIBRARIES=/other AETHER_OUTPUT_MODE=visual aether status", "env HOME=/other AETHER_OUTPUT_MODE=visual aether status", "env GOCACHE=/other AETHER_OUTPUT_MODE=visual aether status",
		"env AETHER_OUTPUT_MODE=other aether status", "env AETHER_OUTPUT_MODE= aether status", "env AETHER_FORCE_COLOR=true aether status", "env AETHER_FORCE_COLOR=2 aether status", "env AETHER_OUTPUT_MODE=visual AETHER_OUTPUT_MODE=json aether status", "env AETHER_FORCE_COLOR=1 AETHER_FORCE_COLOR=1 aether status",
		"AETHER_OUTPUT_MODE=json env AETHER_OUTPUT_MODE=visual aether status", "env AETHER_OUTPUT_MODE=visual env AETHER_FORCE_COLOR=1 aether status", "env AETHER_OUTPUT_MODE=visual /other/aether status", "/usr/bin/env AETHER_OUTPUT_MODE=visual aether status", "env AETHER_OUTPUT_MODE=visual command aether status", "env AETHER_OUTPUT_MODE=visual 'aether status'",
		"env AETHER_OUTPUT_MODE=visual", "env AETHER_OUTPUT_MODE=visual aether", "env AETHER_OUTPUT_MODE=visual aether status AETHER_FORCE_COLOR=1", "env AETHER_OUTPUT_MODE=visual aether status --output /tmp/output", "env AETHER_OUTPUT_MODE=visual aether build 1", "env AETHER_OUTPUT_MODE=visual aether pause", "env AETHER_OUTPUT_MODE=visual aether resume", "env AETHER_OUTPUT_MODE=visual aether spawn-log", "env AETHER_OUTPUT_MODE=visual aether codex-native-worker context-ack --request /request", "env AETHER_OUTPUT_MODE=visual go test ./... -json -count=1", "env AETHER_OUTPUT_MODE=visual gofmt -w clamp.go", "env AETHER_OUTPUT_MODE=visual python3 /coord context-observe", "env AETHER_OUTPUT_MODE=visual cat clamp.go",
		"env AETHER_OUTPUT_MODE=visual aether status; touch clamp.go", "env AETHER_OUTPUT_MODE=visual aether status && go test ./...", "env AETHER_OUTPUT_MODE=visual aether status > clamp.go", "env AETHER_OUTPUT_MODE=$(touch clamp.go) aether status", "env AETHER_OUTPUT_MODE=visual aether $(echo status)",
		"env AETHER_OUTPUT_MODE=visual aether ceremony team-checkin --workflow build --manifest-file /coord/manifest-envelope.json", "env AETHER_OUTPUT_MODE=visual aether ceremony spawn-plan --workflow seal --manifest-file /coord/manifest-envelope.json", "env AETHER_OUTPUT_MODE=visual aether ceremony spawn-plan --workflow build --manifest-file -other", "env AETHER_OUTPUT_MODE=visual aether ceremony worker-complete --workflow build --manifest-file /coord/result.json", "env AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow build --completion-file /coord/result.json --write", "env AETHER_OUTPUT_MODE=visual aether ceremony wave-start --workflow build --manifest-file /coord/manifest-envelope.json --execution-wave $(date)", "env AETHER_OUTPUT_MODE=visual aether ceremony wave-start --workflow build --manifest-file /coord/manifest-envelope.json --execution-wave -1",
	} {
		t.Run("reject/"+command, func(t *testing.T) {
			if nativeParentCoordinationCommand(&r, []string{"/bin/zsh", "-lc", command}, "/fixture") {
				t.Fatalf("display prefix admitted unsafe/outside command: %s", command)
			}
		})
	}
	for index, command := range commands {
		for _, mode := range []string{"valid", "file-uri", "failed-command", "wrong-thread", "wrong-turn", "wrong-cwd", "wrong-argv", "missing-event", "duplicate-event", "missing-output", "wrong-output-call", "wrong-output-turn", "duplicate-call", "incomplete-script", "prefix-mutation"} {
			t.Run("raw/"+strconv.Itoa(index)+"/"+mode, func(t *testing.T) {
				var raw []byte
				add := func(value any) { b, _ := json.Marshal(value); raw = append(raw, append(b, '\n')...) }
				add(map[string]any{"type": "session_meta", "payload": map[string]any{"id": "parent", "cwd": "/fixture"}})
				selected := command
				if mode == "prefix-mutation" {
					selected = strings.Replace(command, "env ", "env PATH=/other ", 1)
				}
				input := "const r = await tools.exec_command({cmd:" + strconv.Quote(selected) + ",workdir:\"/fixture\"}); text(r.output);"
				call := map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call", "name": "exec", "call_id": "call", "input": input, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": "turn"}}}
				add(call)
				if mode == "duplicate-call" {
					add(call)
				}
				thread, turn, cwd := "parent", "turn", "/fixture"
				switch mode {
				case "file-uri":
					cwd = "file:///fixture"
				case "wrong-thread":
					thread = "other"
				case "wrong-turn":
					turn = "other"
				case "wrong-cwd":
					cwd = "/other"
				case "wrong-argv":
					selected = "aether status"
				}
				status, exit := "completed", 0
				if mode == "failed-command" {
					status, exit = "failed", 1
				}
				event := map[string]any{"type": "event_msg", "payload": map[string]any{"type": "item_completed", "thread_id": thread, "turn_id": turn, "item": map[string]any{"type": "CommandExecution", "id": "display", "status": status, "command": []string{"/bin/zsh", "-lc", selected}, "cwd": cwd, "exit_code": exit, "aggregated_output": "display only\n"}}}
				if mode != "missing-event" {
					add(event)
				}
				if mode == "duplicate-event" {
					add(event)
				}
				outputCall, outputTurn, header := "call", "turn", "Script completed\n"
				switch mode {
				case "wrong-output-call":
					outputCall = "other"
				case "wrong-output-turn":
					outputTurn = "other"
				case "incomplete-script":
					header = "Script running\n"
				}
				if mode != "missing-output" {
					add(map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call_output", "call_id": outputCall, "output": []any{map[string]any{"type": "input_text", "text": header}, map[string]any{"type": "input_text", "text": "display only\n"}}, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": outputTurn}}})
				}
				r := codexNativeLiveReceipt{SchemaVersion: "codex-native-tracer/v2", SessionID: "parent", FixtureRoot: "/fixture"}
				nativeInspectParentEvents(&r, raw)
				allowed := mode == "valid" || mode == "file-uri" || mode == "failed-command"
				if r.ParentSubstitution == allowed || r.ChecksPassed || r.ChildEditObserved || r.CreditObserved {
					t.Fatalf("mode=%s parent=%v checks=%v edit=%v credit=%v unclassified=%v", mode, r.ParentSubstitution, r.ChecksPassed, r.ChildEditObserved, r.CreditObserved, r.ParentUnclassified)
				}
			})
		}
	}
}

func nativeParentJSONFieldControls(t *testing.T) {
	root := t.TempDir()
	helper := filepath.Join(root, "coordinator.py")
	body := []byte("coord = pathlib.Path(" + strconv.Quote(root) + ")\n")
	if err := os.WriteFile(helper, body, 0600); err != nil {
		t.Fatal(err)
	}
	r := codexNativeLiveReceipt{FixtureRoot: "/fixture", CoordinationPath: root, CoordinatorPath: helper, CoordinatorSHA256: lifecycleDigest(body)}
	command := func(script, path string) string { return "sed -n '" + script + "' " + strconv.Quote(path) }
	commands := []string{
		command(`/"job_decisions"/,+24p`, filepath.Join(root, "manifest-envelope.json")),
		command(`/"orchestrator_boundary_guidance"/,+30p`, filepath.Join(root, "manifest.json")),
		command(`/"orchestrator_boundary_guidance"/,+30p`, filepath.Join(root, "manifest-envelope.json")),
		command(`/"permission_profile"/,+36p`, filepath.Join(root, "manifest.json")),
	}
	for _, script := range []string{`/"a"/,+1p`, `/"` + strings.Repeat("a", 64) + `"/,+256p`} {
		if !nativeParentCoordinationCommand(&r, []string{"/bin/sh", "-c", command(script, filepath.Join(root, "manifest.json"))}, "/fixture") {
			t.Fatal("valid field/count boundary rejected")
		}
	}
	for _, script := range []string{`/"a"/,+0p`, `/"a"/,+257p`, `/"a"/,+01p`, `/"a"/,+9999p`, `/""/,+1p`, `/"` + strings.Repeat("a", 65) + `"/,+1p`, `/"a.*"/,+1p`, `/"a|b"/,+1p`, `/"a/b"/,+1p`, `/"a"/,+1w`, `/"a"/,+1r`, `/"a"/,+1e`, `/"a"/,+1p;w out`, `/"a"/,+1p;d`, `/"a"/,$p`, `/"a"/,+-1p`} {
		t.Run("script/"+script, func(t *testing.T) {
			if nativeParentCoordinationCommand(&r, []string{"/bin/sh", "-c", command(script, filepath.Join(root, "manifest.json"))}, "/fixture") {
				t.Fatal("unsafe script admitted")
			}
		})
	}
	for _, bad := range []string{
		command(`/"a"/,+1p`, "/other/manifest.json"), command(`/"a"/,+1p`, filepath.Join(root, "other.json")), command(`/"a"/,+1p`, "manifest.json"), command(`/"a"/,+1p`, root+"/../"+filepath.Base(root)+"/manifest.json"),
		strings.Replace(commands[0], "sed -n", "sed -i", 1), strings.Replace(commands[0], "sed -n", "sed -n -e", 1), commands[0] + " > clamp.go", commands[0] + "; touch clamp.go", commands[0] + " " + strconv.Quote(filepath.Join(root, "manifest.json")),
	} {
		if nativeParentCoordinationCommand(&r, []string{"/bin/sh", "-c", bad}, "/fixture") {
			t.Fatalf("unsafe target/argv admitted: %s", bad)
		}
	}
	for _, mode := range []string{"digest", "root", "missing", "duplicate-binding"} {
		t.Run("source/"+mode, func(t *testing.T) {
			copy := r
			switch mode {
			case "digest":
				copy.CoordinatorSHA256 = "wrong"
			case "root":
				copy.CoordinationPath = "/other"
			case "missing":
				copy.CoordinatorPath = filepath.Join(root, "missing.py")
			case "duplicate-binding":
				file := filepath.Join(root, "duplicate.py")
				b := append(append([]byte(nil), body...), body...)
				if err := os.WriteFile(file, b, 0600); err != nil {
					t.Fatal(err)
				}
				copy.CoordinatorPath = file
				copy.CoordinatorSHA256 = lifecycleDigest(b)
			}
			if nativeParentCoordinationCommand(&copy, []string{"/bin/sh", "-c", commands[0]}, "/fixture") {
				t.Fatal("unbound source admitted")
			}
		})
	}
	for index, command := range commands {
		for _, mode := range []string{"valid", "file-uri", "failed-command", "wrong-thread", "wrong-turn", "wrong-cwd", "wrong-argv", "missing-event", "duplicate-event", "missing-output", "wrong-output-call", "wrong-output-turn", "duplicate-call", "incomplete-script", "prefix-mutation"} {
			t.Run("raw/"+strconv.Itoa(index)+"/"+mode, func(t *testing.T) {
				var raw []byte
				add := func(value any) { b, _ := json.Marshal(value); raw = append(raw, append(b, '\n')...) }
				add(map[string]any{"type": "session_meta", "payload": map[string]any{"id": "parent", "cwd": "/fixture"}})
				selected := command
				if mode == "prefix-mutation" {
					selected = strings.Replace(command, root, "/other", 1)
				}
				input := "const r = await tools.exec_command({cmd:" + strconv.Quote(selected) + ",workdir:\"/fixture\"}); text(r.output);"
				call := map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call", "name": "exec", "call_id": "call", "input": input, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": "turn"}}}
				add(call)
				if mode == "duplicate-call" {
					add(call)
				}
				thread, turn, cwd := "parent", "turn", "/fixture"
				switch mode {
				case "file-uri":
					cwd = "file:///fixture"
				case "wrong-thread":
					thread = "other"
				case "wrong-turn":
					turn = "other"
				case "wrong-cwd":
					cwd = "/other"
				case "wrong-argv":
					selected = "aether status"
				}
				status, exit := "completed", 0
				if mode == "failed-command" {
					status, exit = "failed", 1
				}
				event := map[string]any{"type": "event_msg", "payload": map[string]any{"type": "item_completed", "thread_id": thread, "turn_id": turn, "item": map[string]any{"type": "CommandExecution", "id": "display", "status": status, "command": []string{"/bin/zsh", "-lc", selected}, "cwd": cwd, "exit_code": exit, "aggregated_output": "display only\n"}}}
				if mode != "missing-event" {
					add(event)
				}
				if mode == "duplicate-event" {
					add(event)
				}
				outputCall, outputTurn, header := "call", "turn", "Script completed\n"
				switch mode {
				case "wrong-output-call":
					outputCall = "other"
				case "wrong-output-turn":
					outputTurn = "other"
				case "incomplete-script":
					header = "Script running\n"
				}
				if mode != "missing-output" {
					add(map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call_output", "call_id": outputCall, "output": []any{map[string]any{"type": "input_text", "text": header}, map[string]any{"type": "input_text", "text": "display only\n"}}, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": outputTurn}}})
				}
				r := codexNativeLiveReceipt{SchemaVersion: "codex-native-tracer/v2", SessionID: "parent", FixtureRoot: "/fixture", CoordinationPath: root, CoordinatorPath: helper, CoordinatorSHA256: lifecycleDigest(body)}
				nativeInspectParentEvents(&r, raw)
				allowed := mode == "valid" || mode == "file-uri" || mode == "failed-command"
				if r.ParentSubstitution == allowed || r.ChecksPassed || r.ChildEditObserved || r.CreditObserved {
					t.Fatalf("mode=%s parent=%v checks=%v edit=%v credit=%v unclassified=%v", mode, r.ParentSubstitution, r.ChecksPassed, r.ChildEditObserved, r.CreditObserved, r.ParentUnclassified)
				}
			})
		}
	}
}

func TestCodexNativeGapBoundTaskPathCommand(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "coordinator.py")
	raw := []byte(nativeFixtureCoordinator)
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"mapped_path", "mapped_uuid", "unmapped", "wrong_path", "wrong_parent", "review_watcher"} {
		t.Run(mode, func(t *testing.T) {
			r := codexNativeLiveReceipt{SchemaVersion: "codex-native-tracer/v2", FixtureRoot: root, CoordinatorPath: path, CoordinatorSHA256: lifecycleDigest(raw), SessionID: "parent", BoundHostSessionID: "parent", ChildID: "child-uuid", ChildTaskPath: "/root/brick", ChildIdentityCorroborated: true}
			target := "/root/brick"
			switch mode {
			case "mapped_uuid":
				target = "child-uuid"
			case "unmapped":
				r.ChildIdentityCorroborated = false
			case "wrong_path":
				target = "/root/other"
			case "wrong_parent":
				r.BoundHostSessionID = "other"
			case "review_watcher":
				r.Workers = []codexNativeLiveReceipt{{ChildID: "watcher-uuid", ChildTaskPath: "/root/keen", ChildIdentityCorroborated: true, BoundHostSessionID: "parent"}}
				target = "/root/keen"
			}
			got := nativeParentCoordinationCommand(&r, []string{"/bin/sh", "-c", "python3 " + path + " bind " + target}, root)
			want := mode == "mapped_path" || mode == "mapped_uuid" || mode == "review_watcher"
			if got != want {
				t.Fatalf("%s classified=%v want=%v", mode, got, want)
			}
		})
	}
}

func TestCodexNativeGapObservedRefusal(t *testing.T) {
	for _, mode := range []string{"actual_failure", "failed_zero", "missing_exit", "wrong_thread"} {
		t.Run(mode, func(t *testing.T) {
			var raw []byte
			add := func(v any) { b, _ := json.Marshal(v); raw = append(raw, append(b, '\n')...) }
			add(map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call", "call_id": "call", "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": "turn"}}})
			item := map[string]any{"type": "CommandExecution", "id": "event", "status": "failed", "command": []string{"/bin/sh", "-c", "python3 /fixture/coordinator.py empty-result"}, "cwd": "file:///fixture", "exit_code": 1}
			thread := "parent"
			switch mode {
			case "failed_zero":
				item["exit_code"] = 0
			case "missing_exit":
				delete(item, "exit_code")
			case "wrong_thread":
				thread = "foreign"
			}
			add(map[string]any{"type": "event_msg", "payload": map[string]any{"type": "item_completed", "thread_id": thread, "turn_id": "turn", "item": item}})
			add(map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call_output", "call_id": "call", "output": []any{map[string]any{"type": "input_text", "text": "Script completed\n"}}, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": "turn"}}})
			got := nativeCorroboratedBatch(raw, "parent", "call", []nativeRecordedShellCommand{{Command: "python3 /fixture/coordinator.py empty-result", Cwd: "/fixture"}})
			if got != (mode == "actual_failure") {
				t.Fatalf("%s observed=%v", mode, got)
			}
		})
	}
}

func TestCodexNativeGapUncachedCheckSurvivesPlainRecheck(t *testing.T) {
	t.Run("full-result-without-nested-event", nativeGapUncachedCodeModeResultWithoutNestedEvent)
	for _, mode := range []string{"valid", "no_prior", "wrong_thread", "wrong_turn", "duplicate_event", "failure", "missing_output", "later_edit"} {
		t.Run(mode, func(t *testing.T) {
			r := codexNativeLiveReceipt{ChildID: "child", BoundHostSessionID: "parent", FixtureRoot: "/fixture", Caste: "builder"}
			var raw []byte
			add := func(v any) { b, _ := json.Marshal(v); raw = append(raw, append(b, '\n')...) }
			add(map[string]any{"type": "session_meta", "payload": map[string]any{"id": "child", "parent_thread_id": "parent", "cwd": "/fixture", "agent_role": "aether-builder"}})
			if mode != "no_prior" {
				raw = append(raw, nativeEvidenceEvent(t, "item_completed", "child", map[string]any{"type": "CommandExecution", "status": "completed", "id": "uncached", "command": []string{"/bin/sh", "-c", "go test ./... -json -count=1"}, "cwd": "/fixture", "exit_code": 0, "aggregated_output": "{\"Action\":\"run\",\"Package\":\"example.invalid/nativefixture\",\"Test\":\"TestClamp\"}\n{\"Action\":\"pass\",\"Package\":\"example.invalid/nativefixture\",\"Test\":\"TestClamp\"}\n{\"Action\":\"pass\",\"Package\":\"example.invalid/nativefixture\"}\n"})...)
			}
			add(map[string]any{"type": "event_msg", "payload": map[string]any{"thread_id": "child", "turn_id": "turn"}})
			add(map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call", "name": "exec", "call_id": "plain", "input": `text(await tools.exec_command({cmd:"go test ./...",workdir:"/fixture"}));`, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": "turn"}}})
			thread, turn, exit, status := "child", "turn", 0, "completed"
			switch mode {
			case "wrong_thread":
				thread = "foreign"
			case "wrong_turn":
				turn = "foreign"
			case "failure":
				exit = 1
				status = "failed"
			}
			event := map[string]any{"type": "event_msg", "payload": map[string]any{"type": "item_completed", "thread_id": thread, "turn_id": turn, "item": map[string]any{"type": "CommandExecution", "id": "plain-event", "status": status, "command": []string{"/bin/sh", "-c", "go test ./..."}, "cwd": "/fixture", "exit_code": exit, "aggregated_output": "ok example.invalid/nativefixture (cached)"}}}
			add(event)
			if mode == "duplicate_event" {
				add(event)
			}
			if mode == "later_edit" {
				raw = append(raw, nativeEvidenceEvent(t, "item_completed", "child", map[string]any{"type": "FileChange", "status": "completed", "changes": map[string]any{}})...)
			}
			if mode != "missing_output" {
				result, _ := json.Marshal(map[string]any{"exit_code": exit, "output": "ok example.invalid/nativefixture (cached)"})
				add(map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call_output", "call_id": "plain", "output": []any{map[string]any{"type": "input_text", "text": "Script completed\n"}, map[string]any{"type": "input_text", "text": string(result)}}, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": "turn"}}})
			}
			nativeInspectChildEvents(&r, raw)
			if r.ChecksPassed != (mode == "valid") {
				t.Fatalf("%s checks=%v", mode, r.ChecksPassed)
			}
		})
	}
}

func TestCodexNativeGapSelectedSkillDelivery(t *testing.T) {
	for _, mode := range []string{"valid", "manual_user_text", "wrong_turn", "wrong_thread", "wrong_path", "changed_bytes", "duplicate"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "SKILL.md")
			skill := []byte("# ant-build\nExact installed bytes.\n")
			if err := os.WriteFile(path, skill, 0600); err != nil {
				t.Fatal(err)
			}
			text := "<skill>\n<name>ant-build</name>\n<path>" + path + "</path>\n" + string(skill) + "\n</skill>"
			kinds := []string{"skills.selected_skill_instructions"}
			turn, thread := "turn", "parent"
			switch mode {
			case "manual_user_text":
				kinds = nil
			case "wrong_turn":
				turn = "foreign"
			case "wrong_thread":
				thread = "foreign"
			case "wrong_path":
				text = strings.Replace(text, path, "/other/SKILL.md", 1)
			case "changed_bytes":
				text = strings.Replace(text, "Exact", "Changed", 1)
			}
			var raw []byte
			add := func(v any) { b, _ := json.Marshal(v); raw = append(raw, append(b, '\n')...) }
			add(map[string]any{"type": "session_meta", "payload": map[string]any{"id": "parent", "cwd": root}})
			add(map[string]any{"type": "event_msg", "payload": map[string]any{"thread_id": thread, "turn_id": "turn"}})
			message := map[string]any{"type": "response_item", "payload": map[string]any{"type": "message", "id": "skill-message", "role": "user", "content": []any{map[string]any{"type": "input_text", "text": text}}, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": turn, "content_item_kinds": kinds}}}
			add(message)
			if mode == "duplicate" {
				add(message)
			}
			r := codexNativeLiveReceipt{SchemaVersion: "codex-native-tracer/v2", SessionID: "parent", FixtureRoot: root, SkillPath: path}
			nativeInspectParentEvents(&r, raw)
			if r.SkillRead != (mode == "valid") {
				t.Fatalf("%s selected skill=%v", mode, r.SkillRead)
			}
		})
	}
}

func TestCodexNativeGapRecordedFailedRefusal(t *testing.T) {
	for _, mode := range []string{"failed", "wrong_thread", "wrong_cwd", "duplicate", "zero_exit", "wrong_operation"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			fixture := filepath.Join(root, "repository")
			log := filepath.Join(root, "home/.codex/sessions/2026/09/17/rollout-parent.jsonl")
			if err := os.MkdirAll(filepath.Dir(log), 0700); err != nil {
				t.Fatal(err)
			}
			coordinator := filepath.Join(fixture, "coordinator.py")
			thread, cwd, operation, exit := "parent", fixture, "empty-result", 1
			switch mode {
			case "wrong_thread":
				thread = "foreign"
			case "wrong_cwd":
				cwd = "/other"
			case "zero_exit":
				exit = 0
			case "wrong_operation":
				operation = "record"
			}
			event := map[string]any{"type": "event_msg", "payload": map[string]any{"type": "item_completed", "thread_id": thread, "turn_id": "turn", "item": map[string]any{"type": "CommandExecution", "id": "refusal", "status": "failed", "cwd": cwd, "command": []string{"/bin/sh", "-c", "python3 " + coordinator + " " + operation + " 0"}, "exit_code": exit, "aggregated_output": "nonempty terminal result"}}}
			raw, _ := json.Marshal(event)
			raw = append(raw, '\n')
			if mode == "duplicate" {
				raw = append(raw, raw...)
			}
			if err := os.WriteFile(log, raw, 0600); err != nil {
				t.Fatal(err)
			}
			r := codexNativeLiveReceipt{SchemaVersion: "codex-native-tracer/v2", FixtureRoot: fixture, CoordinatorPath: coordinator}
			got := nativeObservedFixtureOperation(r, codexNativeLiveReceipt{BoundHostSessionID: "parent"}, 0, "empty-result", true)
			if got != (mode == "failed") {
				t.Fatalf("%s observed=%v", mode, got)
			}
		})
	}
}

func nativeGapUncachedCodeModeResultWithoutNestedEvent(t *testing.T) {
	const testOutput = "{\"Action\":\"run\",\"Package\":\"example.invalid/nativefixture\",\"Test\":\"TestClamp\"}\n{\"Action\":\"pass\",\"Package\":\"example.invalid/nativefixture\",\"Test\":\"TestClamp\"}\n{\"Action\":\"pass\",\"Package\":\"example.invalid/nativefixture\"}\n"
	for _, mode := range []string{"valid", "projected", "projected_eof", "plain_check", "wrong_child", "wrong_turn", "duplicate_call", "duplicate_output", "missing_exit", "failure", "running", "interrupted", "not_test_json", "later_edit"} {
		t.Run(mode, func(t *testing.T) {
			r := codexNativeLiveReceipt{ChildID: "child", BoundHostSessionID: "parent", FixtureRoot: "/fixture", Caste: "builder"}
			var raw []byte
			add := func(v any) { b, _ := json.Marshal(v); raw = append(raw, append(b, '\n')...) }
			add(map[string]any{"type": "session_meta", "payload": map[string]any{"id": "child", "parent_thread_id": "parent", "cwd": "/fixture", "agent_role": "aether-builder"}})
			thread := "child"
			if mode == "wrong_child" {
				thread = "foreign"
			}
			add(map[string]any{"type": "event_msg", "payload": map[string]any{"thread_id": thread, "turn_id": "turn"}})
			command, print := "go test ./... -json -count=1", "text(JSON.stringify(r));"
			if mode == "plain_check" {
				command = "go test ./..."
			}
			if mode == "projected" || mode == "projected_eof" {
				print = "text(r.output);"
				if mode == "projected_eof" {
					print = "text(r.output)\n"
				}
			}
			call := map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call", "name": "exec", "call_id": "check", "input": "const r = await tools.exec_command({cmd:" + strconv.Quote(command) + ",workdir:\"/fixture\"}); " + print, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": "turn"}}}
			add(call)
			if mode == "duplicate_call" {
				add(call)
			}
			if mode == "later_edit" {
				raw = append(raw, nativeEvidenceEvent(t, "item_completed", "child", map[string]any{"type": "FileChange", "status": "completed", "changes": map[string]any{}})...)
			}
			result := map[string]any{"exit_code": 0, "output": testOutput}
			switch mode {
			case "missing_exit":
				delete(result, "exit_code")
			case "failure":
				result["exit_code"] = 1
			case "running":
				result["session_id"] = 123
			case "not_test_json":
				result["output"] = "ok example.invalid/nativefixture (cached)"
			}
			encoded, _ := json.Marshal(result)
			turn, header := "turn", "Script completed\n"
			if mode == "wrong_turn" {
				turn = "foreign"
			}
			if mode == "interrupted" {
				header = "Script running with cell ID unfinished\n"
			}
			// Deliberately supply a complete-looking result even for .output:
			// a projection cannot supply exit proof without its matched event.
			output := map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call_output", "call_id": "check", "output": []any{map[string]any{"type": "input_text", "text": header}, map[string]any{"type": "input_text", "text": string(encoded)}}, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": turn}}}
			add(output)
			if mode == "duplicate_output" {
				add(output)
			}
			nativeInspectChildEvents(&r, raw)
			if r.ChecksPassed != (mode == "valid") {
				t.Fatalf("%s checks=%v", mode, r.ChecksPassed)
			}
		})
	}
}

func TestCodexNativeGapUncachedCheckSurvivesCoverageRecheck(t *testing.T) {
	for _, check := range []string{"go test ./...", "go test ./... -cover", "go test ./... -cover -count=1", "go test -cover ./..."} {
		for _, projected := range []bool{false, true} {
			for _, mode := range []string{"valid", "no_prior", "prior_failure", "prior_edit", "failed_since_prior", "wrong_thread", "wrong_turn", "duplicate_event", "duplicate_call", "duplicate_output", "failure", "failed_zero", "missing_exit", "missing_event", "missing_output", "later_edit", "wrong_argv", "wrong_cwd", "copied_output", "interrupted_output", "interleaved_call", "forged_projection"} {
				t.Run(check+"/projected="+strconv.FormatBool(projected)+"/"+mode, func(t *testing.T) {
					r := codexNativeLiveReceipt{ChildID: "child", BoundHostSessionID: "parent", FixtureRoot: "/fixture", Caste: "builder"}
					var raw []byte
					add := func(v any) { b, _ := json.Marshal(v); raw = append(raw, append(b, '\n')...) }
					add(map[string]any{"type": "session_meta", "payload": map[string]any{"id": "child", "parent_thread_id": "parent", "cwd": "/fixture", "agent_role": "aether-builder"}})
					if mode != "no_prior" {
						exit, status, action := 0, "completed", "pass"
						if mode == "prior_failure" {
							exit, status, action = 1, "failed", "fail"
						}
						raw = append(raw, nativeEvidenceEvent(t, "item_completed", "child", map[string]any{"type": "CommandExecution", "status": status, "id": "uncached", "command": []string{"/bin/sh", "-c", "go test ./... -json -count=1"}, "cwd": "/fixture", "exit_code": exit, "aggregated_output": "{\"Action\":\"run\",\"Package\":\"example.invalid/nativefixture\",\"Test\":\"TestClamp\"}\n{\"Action\":\"" + action + "\",\"Package\":\"example.invalid/nativefixture\",\"Test\":\"TestClamp\"}\n{\"Action\":\"" + action + "\",\"Package\":\"example.invalid/nativefixture\"}\n"})...)
					}
					if mode == "prior_edit" {
						raw = append(raw, nativeEvidenceEvent(t, "item_completed", "child", map[string]any{"type": "FileChange", "status": "completed", "changes": map[string]any{}})...)
					}
					if mode == "failed_since_prior" {
						raw = append(raw, nativeEvidenceEvent(t, "item_completed", "child", map[string]any{"type": "CommandExecution", "status": "failed", "id": "later-failure", "command": []string{"/bin/sh", "-c", "go test ./..."}, "cwd": "/fixture", "exit_code": 1, "aggregated_output": "FAIL"})...)
					}
					add(map[string]any{"type": "event_msg", "payload": map[string]any{"thread_id": "child", "turn_id": "turn"}})
					print := "text(JSON.stringify(r));"
					if projected {
						print = "text(r.output);"
					}
					if mode == "forged_projection" {
						print = `text("ok example.invalid/nativefixture (cached)");`
					}
					input := "const r = await tools.exec_command({cmd:" + strconv.Quote(check) + ",workdir:\"/fixture\"}); " + print
					call := map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call", "name": "exec", "call_id": "plain", "input": input, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": "turn"}}}
					add(call)
					if mode == "duplicate_call" {
						add(call)
					}
					if mode == "interleaved_call" {
						add(map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call", "name": "exec", "call_id": "other", "input": `text(await tools.exec_command({cmd:"git status --short",workdir:"/fixture"}));`, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": "turn"}}})
					}
					thread, turn, exit, status := "child", "turn", 0, "completed"
					switch mode {
					case "wrong_thread":
						thread = "foreign"
					case "wrong_turn":
						turn = "foreign"
					case "failure":
						exit, status = 1, "failed"
					case "failed_zero":
						status = "failed"
					}
					command, cwd := check, "/fixture"
					if mode == "wrong_argv" {
						command = "go test -cover ./..."
						if command == check {
							command = "go test ./... -cover"
						}
					}
					if mode == "wrong_cwd" {
						cwd = "/other"
					}
					item := map[string]any{"type": "CommandExecution", "id": "plain-event", "status": status, "command": []string{"/bin/zsh", "-lc", command}, "cwd": cwd, "exit_code": exit, "aggregated_output": "ok example.invalid/nativefixture (cached)"}
					if mode == "missing_exit" {
						delete(item, "exit_code")
					}
					event := map[string]any{"type": "event_msg", "payload": map[string]any{"type": "item_completed", "thread_id": thread, "turn_id": turn, "item": item}}
					if mode != "missing_event" {
						add(event)
					}
					if mode == "duplicate_event" {
						add(event)
					}
					if mode == "later_edit" {
						raw = append(raw, nativeEvidenceEvent(t, "item_completed", "child", map[string]any{"type": "FileChange", "status": "completed", "changes": map[string]any{}})...)
					}
					if mode != "missing_output" {
						output := "ok example.invalid/nativefixture (cached)"
						if mode == "copied_output" {
							output = "copied success from another command"
						}
						if !projected {
							result, _ := json.Marshal(map[string]any{"exit_code": exit, "output": output})
							output = string(result)
						}
						header := "Script completed\n"
						if mode == "interrupted_output" {
							header = "Script running with cell ID unfinished\n"
						}
						result := map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call_output", "call_id": "plain", "output": []any{map[string]any{"type": "input_text", "text": header}, map[string]any{"type": "input_text", "text": output}}, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": "turn"}}}
						add(result)
						if mode == "duplicate_output" {
							add(result)
						}
					}
					nativeInspectChildEvents(&r, raw)
					if r.ChecksPassed != (mode == "valid") {
						t.Fatalf("%s checks=%v", mode, r.ChecksPassed)
					}
				})
			}
		}
	}
}

func nativeObservedExitOutputProjection(t *testing.T) {
	root := t.TempDir()
	coordinator := filepath.Join(root, "native-fixture-coordinate.py")
	body := []byte("# immutable synthetic coordinator identity\n")
	if err := os.WriteFile(coordinator, body, 0600); err != nil {
		t.Fatal(err)
	}
	command := "python3 " + coordinator + " empty-result"
	for _, presentation := range []struct {
		name, suffix string
		printed      func(int, string) []string
	}{
		{"json-exit-first", "text(JSON.stringify({exit_code:r.exit_code,output:r.output}));", func(exit int, output string) []string {
			encoded, _ := json.Marshal(output)
			return []string{`{"exit_code":` + strconv.Itoa(exit) + `,"output":` + string(encoded) + `}`}
		}},
		{"json-output-first-eof", "text(JSON.stringify({output:r.output,exit_code:r.exit_code}))\n", func(exit int, output string) []string {
			encoded, _ := json.Marshal(output)
			return []string{`{"output":` + string(encoded) + `,"exit_code":` + strconv.Itoa(exit) + `}`}
		}},
		{"exit-first-combined", "text(`exit_code=${r.exit_code}\\n${r.output}`);", func(exit int, output string) []string {
			return []string{"exit_code=" + strconv.Itoa(exit) + "\n" + output}
		}},
		{"output-first-combined", "text(`${r.output}\\nexit_code=${r.exit_code}`);", func(exit int, output string) []string { return []string{output + "\nexit_code=" + strconv.Itoa(exit)} }},
		{"actual-lowercase-sequence", "text(r.output); text(`\\nexit_code=${r.exit_code}`);", func(exit int, output string) []string { return []string{output, "\nexit_code=" + strconv.Itoa(exit)} }},
		{"uppercase-sequence-eof", "text(r.output); text(`\\nEXIT_CODE=${r.exit_code}`)\n", func(exit int, output string) []string { return []string{output, "\nEXIT_CODE=" + strconv.Itoa(exit)} }},
		{"exit-first-sequence", "text(`status=${r.exit_code}`); text(r.output);", func(exit int, output string) []string { return []string{"status=" + strconv.Itoa(exit), output} }},
		{"literal-annotation", "text(`\\nProcess status=${r.exit_code}`); text(r.output)\n", func(exit int, output string) []string {
			return []string{"\nProcess status=" + strconv.Itoa(exit), output}
		}},
		{"max-label", "text(`ABCDEFGHIJKLMNOPQRSTUVWXYZ_12345=${r.exit_code}\\n${r.output}`);", func(exit int, output string) []string {
			return []string{"ABCDEFGHIJKLMNOPQRSTUVWXYZ_12345=" + strconv.Itoa(exit) + "\n" + output}
		}},
	} {
		for _, mode := range []string{"success", "expected-refusal", "file-uri", "renamed-alias", "missing-event", "wrong-thread", "wrong-turn", "wrong-cwd", "changed-command", "duplicate-event", "duplicate-call", "interleaved-call", "missing-output", "duplicate-output", "wrong-output-call", "wrong-output-turn", "incomplete", "omitted-exit", "forged-exit", "forged-output", "reordered", "extra-output", "extra-field", "child-plain-no-credit", "json-reordered-keys", "json-unicode-whitespace", "json-duplicate-output", "json-number-output", "json-boolean-exit", "json-decimal-exit", "json-duplicate-key", "json-extra-key", "json-missing-output", "json-missing-exit", "json-string-exit", "json-null-exit", "json-null-output", "json-trailing-value"} {
			if strings.HasPrefix(mode, "json-") && !strings.HasPrefix(presentation.name, "json-") {
				continue
			}
			t.Run(presentation.name+"/"+mode, func(t *testing.T) {
				r := codexNativeLiveReceipt{SchemaVersion: "codex-native-tracer/v2", SessionID: "parent", FixtureRoot: root, CoordinatorPath: coordinator, CoordinatorSHA256: lifecycleDigest(body)}
				thread, cwd, selected := "parent", root, command
				if mode == "child-plain-no-credit" {
					thread, selected = "child", "go test ./..."
					r.ChildID, r.BoundHostSessionID, r.Caste = "child", "parent", "builder"
				}
				var raw []byte
				add := func(v any) { b, _ := json.Marshal(v); raw = append(raw, append(b, '\n')...) }
				parent := ""
				if thread == "child" {
					parent = "parent"
				}
				add(map[string]any{"type": "session_meta", "payload": map[string]any{"id": thread, "parent_thread_id": parent, "cwd": root, "agent_role": "aether-builder"}})
				add(map[string]any{"type": "event_msg", "payload": map[string]any{"thread_id": thread, "turn_id": "turn"}})
				input := "const r = await tools.exec_command({cmd:" + strconv.Quote(selected) + ",workdir:" + strconv.Quote(root) + "}); " + presentation.suffix
				if mode == "renamed-alias" {
					input = strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(input, "const r =", "const result ="), "${r.", "${result."), "text(r.", "text(result.")
					input = strings.ReplaceAll(input, ":r.", ":result.")
				}
				call := map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call", "name": "exec", "call_id": "call", "input": input, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": "turn"}}}
				add(call)
				if mode == "duplicate-call" {
					add(call)
				}
				if mode == "interleaved-call" {
					add(map[string]any{"type": "response_item", "payload": map[string]any{"type": "function_call", "name": "send_message", "call_id": "other"}})
				}
				turn, status, exit, output := "turn", "completed", 0, "actual output\n"
				if mode == "expected-refusal" {
					status, exit, output = "failed", 1, "{\"ok\":false,\"error\":\"nonempty terminal result and summary are required\",\"code\":1}\n\n\n"
				}
				if mode == "json-unicode-whitespace" {
					output = "  <tag>& café \u2028\u2029\t\n\n"
				}
				if mode == "child-plain-no-credit" {
					output = "ok  \texample.invalid/nativefixture\t0.2s\n"
				}
				switch mode {
				case "file-uri":
					cwd = "file://" + root
				case "wrong-thread":
					thread = "foreign"
				case "wrong-turn":
					turn = "foreign"
				case "wrong-cwd":
					cwd = "/other"
				case "changed-command":
					selected = "aether status"
				}
				event := map[string]any{"type": "event_msg", "payload": map[string]any{"type": "item_completed", "thread_id": thread, "turn_id": turn, "item": map[string]any{"type": "CommandExecution", "id": "event", "status": status, "command": []string{"/bin/zsh", "-lc", selected}, "cwd": cwd, "exit_code": exit, "aggregated_output": output}}}
				if mode != "missing-event" {
					add(event)
				}
				if mode == "duplicate-event" {
					add(event)
				}
				id, outputTurn, header := "call", "turn", "Script completed\n"
				printed := presentation.printed(exit, output)
				switch mode {
				case "wrong-output-call":
					id = "other"
				case "wrong-output-turn":
					outputTurn = "other"
				case "incomplete":
					header = "Script running with cell ID pending\n"
				case "omitted-exit":
					printed = []string{output}
				case "forged-exit":
					printed = presentation.printed(exit+1, output)
				case "forged-output":
					printed = presentation.printed(exit, "forged")
				case "reordered":
					if len(printed) == 2 {
						printed[0], printed[1] = printed[1], printed[0]
					} else {
						printed[0] = "reordered\n" + printed[0]
					}
				case "extra-field":
					printed[len(printed)-1] += "\nextra=field"
				case "json-reordered-keys":
					encoded, _ := json.Marshal(output)
					printed = []string{`{ "output": ` + string(encoded) + `, "exit_code": ` + strconv.Itoa(exit) + ` }`}
				case "json-unicode-whitespace":
					// JavaScript preserves these literals; Go's default JSON encoder
					// escapes them. Compare decoded output bytes without trimming.
					for _, pair := range [][2]string{{`\u003c`, "<"}, {`\u003e`, ">"}, {`\u0026`, "&"}, {`\u2028`, "\u2028"}, {`\u2029`, "\u2029"}} {
						printed[0] = strings.ReplaceAll(printed[0], pair[0], pair[1])
					}
				case "json-duplicate-output":
					printed[0] = strings.TrimSuffix(printed[0], "}") + `,"output":"actual output\n"}`
				case "json-number-output":
					printed = []string{`{"exit_code":0,"output":3}`}
				case "json-boolean-exit":
					printed = []string{`{"exit_code":false,"output":"actual output\n"}`}
				case "json-decimal-exit":
					printed = []string{`{"exit_code":0.0,"output":"actual output\n"}`}
				case "json-duplicate-key":
					printed[0] = strings.TrimSuffix(printed[0], "}") + `,"exit_code":0}`
				case "json-extra-key":
					printed[0] = strings.TrimSuffix(printed[0], "}") + `,"extra":0}`
				case "json-missing-output":
					printed = []string{`{"exit_code":0}`}
				case "json-missing-exit":
					printed = []string{`{"output":"actual output\n"}`}
				case "json-string-exit":
					printed = []string{`{"exit_code":"0","output":"actual output\n"}`}
				case "json-null-exit":
					printed = []string{`{"exit_code":null,"output":"actual output\n"}`}
				case "json-null-output":
					printed = []string{`{"exit_code":0,"output":null}`}
				case "json-trailing-value":
					printed[0] += `{}`
				}
				parts := []any{map[string]any{"type": "input_text", "text": header}}
				for _, text := range printed {
					parts = append(parts, map[string]any{"type": "input_text", "text": text})
				}
				if mode == "extra-output" {
					parts = append(parts, parts[1])
				}
				result := map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call_output", "call_id": id, "output": parts, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": outputTurn}}}
				if mode != "missing-output" {
					add(result)
				}
				if mode == "duplicate-output" {
					add(result)
				}
				if mode == "child-plain-no-credit" {
					nativeInspectChildEvents(&r, raw)
					if len(r.ChildUnclassified) != 0 || r.ChecksPassed || r.ChildEditObserved {
						t.Fatalf("presentation minted child proof: %+v", r)
					}
				} else {
					nativeInspectParentEvents(&r, raw)
					if mode == "expected-refusal" && !r.EmptyResultRefused {
						t.Fatal("actual failed empty-result refusal lost")
					}
					want := mode == "success" || mode == "expected-refusal" || mode == "file-uri" || mode == "renamed-alias" || mode == "json-reordered-keys" || mode == "json-unicode-whitespace"
					if r.ParentSubstitution == want || r.ChecksPassed || r.ChildEditObserved {
						t.Fatalf("parent classification: %+v", r.ParentUnclassified)
					}
				}
			})
		}
	}
}

func TestCodexNativeObservedOperationWrappers(t *testing.T) {
	t.Run("exit-output-event-projection", nativeObservedExitOutputProjection)
	t.Run("bounded-presentation-plan", nativePresentationPlanBoundaries)
	const prefix = `const r = await tools.exec_command({cmd:"go test ./... -json -count=1",workdir:"/fixture"}); `
	for _, tc := range []struct {
		name, suffix string
		want         bool
	}{
		{"full-json", `text(JSON.stringify(r));`, true},
		{"exit-output", "text(`exit_code=${r.exit_code}\\n${r.output}`);", true},
		{"exit-output-eof", "text(`exit_code=${r.exit_code}\\n${r.output}`)\n", true},
		{"exit-output-wrong-alias", "text(`exit_code=${other.exit_code}\\n${r.output}`);", false},
		{"exit-output-omitted", "text(`exit_code=${r.exit_code}`);", false},
		{"exit-output-source-order", "text(`${r.output}\\nexit_code=${r.exit_code}`);", true},
		{"exit-output-fake-exit", "text(`exit_code=0\\n${r.output}`);", false},
		{"exit-output-transformed", "text(`exit_code=${r.exit_code}\\n${r.output.trim()}`);", false},
		{"exit-output-extra", "text(`exit_code=${r.exit_code}\\n${r.output}extra`);", false},
		{"exit-output-side-effect", "text(`exit_code=${r.exit_code}\\n${r.output}`); mutate();", false},
		{"exit-output-extra-call", "text(`exit_code=${r.exit_code}\\n${r.output}`); text(r);", false},
		{"full-json-eof", "text(JSON.stringify(r))\n", true},
		{"full-result-eof", "text(r)\n", true},
		{"output-eof", "text(r.output)\n", true},
		{"output-exit", "text(r.output); text(`\\nEXIT_CODE=${r.exit_code}`);", true},
		{"output-exit-eof", "text(r.output); text(`\\nEXIT_CODE=${r.exit_code}`)\n", true},
		{"other-result", `text(JSON.stringify(other));`, false},
		{"json-replacer", `text(JSON.stringify(r, mutate));`, false},
		{"json-property", `text(JSON.stringify(r.output));`, false},
		{"rewritten-result", `r.exit_code=0; text(JSON.stringify(r));`, false},
		{"extra-side-effect", `text(JSON.stringify(r)); mutate();`, false},
		{"eof-side-effect", "text(JSON.stringify(r))\nmutate();", false},
		{"eof-output-side-effect", "text(r.output)\nmutate();", false},
		{"eof-full-result-side-effect", "text(r)\nmutate();", false},
		{"eof-output-transform", "text(r.output.trim())\n", false},
		{"eof-other-result", "text(JSON.stringify(other))\n", false},
		{"missing-output-exit-separator", "text(r.output)\ntext(`\\nEXIT_CODE=${r.exit_code}`)", false},
		{"wrong-exit-variable", "text(r.output); text(`\\nEXIT_CODE=${other.exit_code}`);", false},
		{"computed-exit", "text(r.output); text(`\\nEXIT_CODE=${r.exit_code || 0}`);", false},
		{"fake-exit", "text(r.output); text(`\\nEXIT_CODE=0`);", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := nativeCodeModeCommands(prefix+tc.suffix, "/fixture")
			if ok != tc.want || (ok && (len(got) != 1 || got[0].Command != "go test ./... -json -count=1" || got[0].Cwd != "/fixture")) {
				t.Fatalf("accepted=%v commands=%+v", ok, got)
			}
		})
	}
	for _, bad := range []string{
		strings.Replace(prefix, `"go test ./... -json -count=1"`, `command`, 1) + `text(JSON.stringify(r));`,
		strings.Replace(prefix, `cmd:`, `cmd:"touch clamp.go",cmd:`, 1) + `text(JSON.stringify(r));`,
		strings.Replace(prefix, `await `, ``, 1) + `text(JSON.stringify(r));`,
		`const JSON={stringify:mutate}; ` + prefix + `text(JSON.stringify(r));`,
		strings.TrimSuffix(strings.TrimSpace(prefix), ";") + "\ntext(r.output)",
		strings.TrimSuffix(strings.TrimSpace(prefix), ";") + "\ntext(JSON.stringify(r))",
		strings.TrimSuffix(strings.TrimSpace(prefix), ";") + "text(r)",
	} {
		if _, ok := nativeCodeModeCommands(bad, "/fixture"); ok {
			t.Fatalf("dynamic/side-effect wrapper accepted: %s", bad)
		}
	}
	t.Run("literal-eof-sequence-boundary", func(t *testing.T) {
		const direct = `text(await tools.exec_command({cmd:"git status --short",workdir:"/fixture"}))`
		for _, tc := range []struct {
			input string
			want  int
		}{
			{direct + "\n", 1},
			{direct + ";\n" + direct + "\n", 2},
			{direct + "\n" + direct, 0},
			{direct + direct, 0},
			{direct + "\nmutate();", 0},
		} {
			commands, ok := nativeCodeModeCommands(tc.input, "/fixture")
			if ok != (tc.want > 0) || (ok && len(commands) != tc.want) {
				t.Fatalf("EOF sequence accepted=%v commands=%v input=%s", ok, commands, tc.input)
			}
		}
		if !nativeCodeModePlainOutput(prefix+"text(r.output)\n", "/fixture") {
			t.Fatal("EOF projection lost its independent event requirement")
		}
	})
}

func TestCodexNativeObservedReadOnlyBatch(t *testing.T) {
	t.Run("named-results", nativeObservedNamedReadOnlyBatch)
	t.Run("untouched-for-of-results", nativeObservedFullResultReadOnlyBatch)
	batch := `const results = await Promise.all([tools.exec_command({cmd:"sed -n '1,200p' clamp_test.go",workdir:"/fixture"}),tools.exec_command({cmd:"git status --short",workdir:"/fixture"})]); for (const r of results) text(r.output);`
	for _, tc := range []struct {
		name, input string
		want        bool
	}{
		{"output", batch, true},
		{"full-json", strings.Replace(batch, "text(r.output)", "text(JSON.stringify(r))", 1), true},
		{"wrong-array", strings.Replace(batch, "of results", "of other", 1), false},
		{"wrong-result", strings.Replace(batch, "text(r.output)", "text(other.output)", 1), false},
		{"mutated-result", strings.Replace(batch, "text(r.output)", "text(r.output.trim())", 1), false},
		{"trailing", batch + `mutate();`, false},
		{"unawaited", strings.Replace(batch, "await Promise", "Promise", 1), false},
		{"write", strings.Replace(batch, "git status --short", "gofmt -w clamp.go", 1), false},
		{"compound", strings.Replace(batch, "git status --short", "git status --short; touch clamp.go", 1), false},
		{"computed-command", strings.Replace(batch, `"git status --short"`, `command`, 1), false},
		{"other-tool", strings.Replace(batch, "tools.exec_command", "tools.apply_patch", 1), false},
		{"spread-array", strings.Replace(batch, "[tools.exec_command", "[...tools.exec_command", 1), false},
		{"loop-body-side-effect", strings.Replace(batch, "text(r.output);", "{ mutate(); text(r.output); }", 1), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			commands, ok := nativeCodeModeCommands(tc.input, "/fixture")
			if ok != tc.want || (ok && len(commands) != 2) {
				t.Fatalf("accepted=%v commands=%+v", ok, commands)
			}
		})
	}
	calls := make([]string, 65)
	for i := range calls {
		calls[i] = `tools.exec_command({cmd:"git status --short",workdir:"/fixture"})`
	}
	if _, ok := nativeCodeModeCommands(`const results=await Promise.all([`+strings.Join(calls, ",")+`]); for (const r of results) text(r.output);`, "/fixture"); ok {
		t.Fatal("unbounded batch accepted")
	}
}

func nativeObservedFullResultReadOnlyBatch(t *testing.T) {
	const each = `const results = await Promise.all([tools.exec_command({cmd:"git status --short",workdir:"/fixture"}),tools.exec_command({cmd:"sed -n '1,200p' clamp.go && sed -n '1,200p' clamp_test.go && sed -n '1,120p' go.mod",workdir:"/fixture"})]); results.forEach(text);`
	const input = `const results = await Promise.all([tools.exec_command({cmd:"rg --files -g '!*vendor*'",workdir:"/fixture"}),tools.exec_command({cmd:"git status --short",workdir:"/fixture"})]); for (const r of results) text(r);`
	for _, tc := range []struct {
		name, input string
		want        bool
	}{
		{"untouched", input, true},
		{"foreach-composed-actual", each, true},
		{"foreach-renamed", strings.ReplaceAll(each, "results", "items"), true},
		{"forof-composed", strings.Replace(each, "results.forEach(text);", "for (const item of results) text(item);", 1), true},
		{"foreach-wrong-alias", strings.Replace(each, "results.forEach", "other.forEach", 1), false},
		{"foreach-write-member", strings.Replace(each, "git status --short", "gofmt -w clamp.go", 1), false},
		{"foreach-test-member", strings.Replace(each, "git status --short", "go test ./... -json -count=1", 1), false},
		{"foreach-mixed-chain-write", strings.Replace(each, "sed -n '1,120p' go.mod", "touch clamp.go", 1), false},
		{"foreach-mixed-chain-test", strings.Replace(each, "sed -n '1,120p' go.mod", "go test ./... -json -count=1", 1), false},
		{"foreach-chain-outside", strings.Replace(each, "clamp_test.go", "/other/clamp_test.go", 1), false},
		{"foreach-parent-discovery", strings.Replace(each, "git status --short", "rg --files /other", 1), false},
		{"foreach-callback-arrow", strings.Replace(each, "forEach(text)", "forEach(r => text(r))", 1), false},
		{"foreach-callback-other", strings.Replace(each, "forEach(text)", "forEach(mutate)", 1), false},
		{"foreach-callback-bind", strings.Replace(each, "forEach(text)", "forEach(text.bind(null))", 1), false},
		{"foreach-extra-this", strings.Replace(each, "forEach(text)", "forEach(text, null)", 1), false},
		{"foreach-computed-method", strings.Replace(each, ".forEach(text)", `["forEach"](text)`, 1), false},
		{"foreach-prototype", strings.Replace(each, ".forEach(text)", ".__proto__.forEach(text)", 1), false},
		{"foreach-mutation", strings.Replace(each, "results.forEach", "results.reverse(); results.forEach", 1), false},
		{"foreach-extra-result", each + "text(results);", false},
		{"foreach-extra-exec", each + `text(await tools.exec_command({cmd:"touch clamp.go",workdir:"/fixture"}));`, false},
		{"foreach-dynamic", strings.Replace(each, `"git status --short"`, "command", 1), false},
		{"foreach-reserved", strings.ReplaceAll(each, "results", "text"), false},

		{"foreach-chain-eight", strings.Replace(each, "sed -n '1,200p' clamp.go && sed -n '1,200p' clamp_test.go && sed -n '1,120p' go.mod", strings.TrimSuffix(strings.Repeat("pwd && ", 8), " && "), 1), true},
		{"foreach-chain-nine", strings.Replace(each, "sed -n '1,200p' clamp.go && sed -n '1,200p' clamp_test.go && sed -n '1,120p' go.mod", strings.TrimSuffix(strings.Repeat("pwd && ", 9), " && "), 1), false},
		{"renamed-identifiers", strings.Replace(strings.Replace(strings.ReplaceAll(input, "results", "items"), "const r of", "const item of", 1), "text(r)", "text(item)", 1), true},
		{"mismatched-alias-renaming", strings.ReplaceAll(strings.ReplaceAll(input, "results", "items"), " r", " item"), false},
		{"wrong-array", strings.Replace(input, "of results", "of other", 1), false},
		{"wrong-result", strings.Replace(input, "text(r)", "text(other)", 1), false},
		{"same-alias", strings.ReplaceAll(input, "results", "r"), false},
		{"prototype", strings.Replace(input, "text(r)", "text(r.__proto__)", 1), false},
		{"constructor", strings.Replace(input, "text(r)", "text(r.constructor())", 1), false},
		{"extra-output", input + "text(results);", false},
		{"effect", strings.Replace(input, "text(r);", "{ text(r); mutate(); }", 1), false},
		{"dynamic-binding", strings.Replace(input, "const results", "const [results]", 1), false},
		{"dynamic-call", strings.Replace(input, `"git status --short"`, `command`, 1), false},
		{"test", strings.Replace(input, "git status --short", "go test ./... -json -count=1", 1), false},
		{"write", strings.Replace(input, "git status --short", "gofmt -w clamp.go", 1), false},
		{"unawaited", strings.Replace(input, "await Promise", "Promise", 1), false},
		{"extra-statement", input + "mutate();", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, ok := nativeCodeModeCommands(tc.input, "/fixture")
			if ok != tc.want {
				t.Fatalf("accepted=%v input=%s", ok, tc.input)
			}
		})
	}
	for _, name := range []string{"text", "tools", "Promise", "JSON", "await", "const"} {
		for _, input := range []string{strings.ReplaceAll(input, "results", name), strings.Replace(strings.Replace(input, "const r of", "const "+name+" of", 1), "text(r)", "text("+name+")", 1)} {
			if _, ok := nativeCodeModeCommands(input, "/fixture"); ok {
				t.Fatalf("shadowed batch binding accepted: %s", input)
			}
		}
	}
	for _, size := range []int{64, 65} {
		calls := make([]string, size)
		for i := range calls {
			calls[i] = `tools.exec_command({cmd:"git status --short",workdir:"/fixture"})`
		}
		if _, ok := nativeCodeModeCommands(`const results=await Promise.all([`+strings.Join(calls, ",")+`]); for(const r of results) text(r);`, "/fixture"); ok != (size == 64) {
			t.Fatalf("batch bound %d accepted=%v", size, ok)
		}
		if _, ok := nativeCodeModeCommands(`const results=await Promise.all([`+strings.Join(calls, ",")+`]); results.forEach(text);`, "/fixture"); ok != (size == 64) {
			t.Fatalf("foreach batch bound %d accepted=%v", size, ok)
		}
	}
	for _, batch := range []string{input, each,
		`const results = await Promise.all([tools.exec_command({cmd:"sed -n '1,240p' clamp.go",workdir:"/fixture"}),tools.exec_command({cmd:"sed -n '1,260p' clamp_test.go",workdir:"/fixture"}),tools.exec_command({cmd:"sed -n '1,160p' go.mod",workdir:"/fixture"})]); for (const r of results) text(r);`,
		`const results = await Promise.all([tools.exec_command({cmd:"git diff --check",workdir:"/fixture"}),tools.exec_command({cmd:"git diff -- clamp.go",workdir:"/fixture"}),tools.exec_command({cmd:"git status --short",workdir:"/fixture"}),tools.exec_command({cmd:"date -u +%Y-%m-%dT%H:%M:%SZ",workdir:"/fixture"})]); for (const r of results) text(r);`,
	} {
		for _, mode := range []string{"valid", "no-prior", "wrong-source-parent", "wrong-thread", "wrong-turn", "wrong-cwd", "invocation-cwd", "wrong-command", "missing-event", "duplicate-event", "duplicate-call", "duplicate-output", "missing-output", "wrong-output-call", "wrong-output-turn", "incomplete-output", "raw-output-mismatch", "raw-exit-mismatch", "extra-output", "truncated-output", "duplicate-command", "reordered-output", "running-result", "interleaved-call", "later-failed-check", "later-edit", "failed-member", "completion-order-forward", "callback-index-output", "callback-array-output"} {
			if batch != input && batch != each && mode != "valid" {
				continue
			}
			if batch != each && (mode == "failed-member" || mode == "completion-order-forward" || strings.HasPrefix(mode, "callback-")) {
				continue
			}
			t.Run("raw/"+strconv.Itoa(len(batch))+"/"+mode, func(t *testing.T) {
				r := codexNativeLiveReceipt{ChildID: "child", BoundHostSessionID: "parent", FixtureRoot: "/fixture", Caste: "builder"}
				var raw []byte
				add := func(v any) { b, _ := json.Marshal(v); raw = append(raw, append(b, '\n')...) }
				parent := "parent"
				if mode == "wrong-source-parent" {
					parent = "foreign"
				}
				add(map[string]any{"type": "session_meta", "payload": map[string]any{"id": "child", "parent_thread_id": parent, "cwd": "/fixture", "agent_role": "aether-builder"}})
				if mode != "no-prior" {
					raw = append(raw, nativeEvidenceEvent(t, "item_completed", "child", map[string]any{"type": "CommandExecution", "status": "completed", "id": "uncached", "command": []string{"/bin/sh", "-c", "go test ./... -json -count=1"}, "cwd": "/fixture", "exit_code": 0, "aggregated_output": "{\"Action\":\"run\",\"Package\":\"example.invalid/nativefixture\",\"Test\":\"TestClamp\"}\n{\"Action\":\"pass\",\"Package\":\"example.invalid/nativefixture\",\"Test\":\"TestClamp\"}\n{\"Action\":\"pass\",\"Package\":\"example.invalid/nativefixture\"}\n"})...)
				}
				add(map[string]any{"type": "event_msg", "payload": map[string]any{"thread_id": "child", "turn_id": "turn"}})
				selected := batch
				if mode == "invocation-cwd" {
					selected = strings.Replace(batch, "/fixture", "/other", 1)
				}
				if mode == "duplicate-command" {
					original, _ := nativeCodeModeCommands(batch, "/fixture")
					selected = strings.Replace(batch, original[1].Command, original[0].Command, 1)
				}
				call := map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call", "name": "exec", "call_id": "batch", "input": selected, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": "turn"}}}
				add(call)
				if mode == "duplicate-call" {
					add(call)
				}
				if mode == "interleaved-call" {
					add(map[string]any{"type": "response_item", "payload": map[string]any{"type": "function_call", "name": "send_message", "call_id": "other"}})
				}
				commands, _ := nativeCodeModeCommands(batch, "/fixture")
				// Deliberately finish events out of array order, as in the real capture.
				for step := 0; step < len(commands); step++ {
					index := len(commands) - 1 - step
					if mode == "completion-order-forward" {
						index = step
					}
					if index == 0 && mode == "missing-event" {
						continue
					}
					thread, turn, cwd, command := "child", "turn", "/fixture", commands[index].Command
					if index == 0 {
						switch mode {
						case "wrong-thread":
							thread = "foreign"
						case "wrong-turn":
							turn = "foreign"
						case "wrong-cwd":
							cwd = "/other"
						case "wrong-command":
							command = "cat go.mod"
						}
					}
					status, exit := "completed", 0
					if mode == "failed-member" && index == 0 {
						status, exit = "failed", 1
					}
					event := map[string]any{"type": "event_msg", "payload": map[string]any{"type": "item_completed", "thread_id": thread, "turn_id": turn, "item": map[string]any{"type": "CommandExecution", "id": "event" + strconv.Itoa(index), "status": status, "command": []string{"/bin/zsh", "-lc", command}, "cwd": cwd, "exit_code": exit, "aggregated_output": "inspection" + strconv.Itoa(index)}}}
					add(event)
					if index == 0 && mode == "duplicate-event" {
						add(event)
					}
				}
				id, turn, header := "batch", "turn", "Script completed\n"
				if mode == "wrong-output-call" {
					id = "other"
				}
				if mode == "wrong-output-turn" {
					turn = "other"
				}
				if mode == "incomplete-output" {
					header = "Script running with cell ID pending\n"
				}
				parts := []any{map[string]any{"type": "input_text", "text": header}}
				for index := range commands {
					body := map[string]any{"chunk_id": "actual-shape", "wall_time_seconds": 0.01, "original_token_count": 1, "exit_code": 0, "output": "inspection" + strconv.Itoa(index)}
					if index == 0 {
						switch mode {
						case "raw-output-mismatch":
							body["output"] = "forged"
						case "raw-exit-mismatch":
							body["exit_code"] = 1
						case "failed-member":
							body["exit_code"] = 1
						case "running-result":
							body["session_id"] = 123
						}
					}
					encoded, _ := json.Marshal(body)
					parts = append(parts, map[string]any{"type": "input_text", "text": string(encoded)})
				}
				if mode == "truncated-output" {
					parts = parts[:len(parts)-1]
				}
				if mode == "extra-output" {
					parts = append(parts, parts[1])
				}
				if mode == "reordered-output" {
					parts[1], parts[2] = parts[2], parts[1]
				}
				if mode == "callback-index-output" {
					parts = append(parts, map[string]any{"type": "input_text", "text": "0"})
				}
				if mode == "callback-array-output" {
					parts = append(parts, map[string]any{"type": "input_text", "text": `[{"exit_code":0,"output":"inspection0"},{"exit_code":0,"output":"inspection1"}]`})
				}
				output := map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call_output", "call_id": id, "output": parts, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": turn}}}
				if mode != "missing-output" {
					add(output)
				}
				if mode == "duplicate-output" {
					add(output)
				}
				if mode == "later-failed-check" {
					raw = append(raw, nativeEvidenceEvent(t, "item_completed", "child", map[string]any{"type": "CommandExecution", "status": "completed", "id": "failed", "command": []string{"/bin/sh", "-c", "go test ./..."}, "cwd": "/fixture", "exit_code": 1, "aggregated_output": "FAIL"})...)
				}
				if mode == "later-edit" {
					raw = append(raw, nativeEvidenceEvent(t, "item_completed", "child", map[string]any{"type": "FileChange", "status": "completed", "changes": map[string]any{}})...)
				}
				nativeInspectChildEvents(&r, raw)
				if r.ChecksPassed != (mode == "valid" || mode == "completion-order-forward") || r.ChildEditObserved {
					t.Fatalf("mode=%s checks=%v edit=%v unclassified=%v", mode, r.ChecksPassed, r.ChildEditObserved, r.ChildUnclassified)
				}
				classified := mode == "valid" || mode == "completion-order-forward" || mode == "no-prior" || mode == "later-failed-check" || mode == "later-edit" || mode == "wrong-source-parent"
				if (len(r.ChildUnclassified) == 0) != classified {
					t.Fatalf("mode=%s classification=%v", mode, r.ChildUnclassified)
				}
			})
		}
	}
}

func nativeObservedNamedReadOnlyBatch(t *testing.T) {
	inspection := `const [status, source, tests] = await Promise.all([tools.exec_command({cmd:"git status --short",workdir:"/fixture"}),tools.exec_command({cmd:"sed -n '1,200p' clamp.go",workdir:"/fixture"}),tools.exec_command({cmd:"sed -n '1,200p' clamp_test.go",workdir:"/fixture"})]); text(JSON.stringify({status,source,tests}));`
	freshness := `const [diff, status, freshness] = await Promise.all([tools.exec_command({cmd:"git diff -- clamp.go",workdir:"/fixture"}),tools.exec_command({cmd:"git status --short",workdir:"/fixture"}),tools.exec_command({cmd:"date -u +%Y-%m-%dT%H:%M:%SZ",workdir:"/fixture"})]); text(JSON.stringify({diff,status,freshness}));`
	for _, tc := range []struct {
		name, input string
		want        bool
	}{
		{"inspection", inspection, true},
		{"freshness", freshness, true},
		{"single", `const [status] = await Promise.all([tools.exec_command({cmd:"git status --short",workdir:"/fixture"})]); text(JSON.stringify({status}));`, true},
		{"omitted-field", strings.Replace(inspection, "{status,source,tests}", "{status,source}", 1), false},
		{"duplicate-field", strings.Replace(inspection, "{status,source,tests}", "{status,source,source}", 1), false},
		{"reordered-fields", strings.Replace(inspection, "{status,source,tests}", "{source,status,tests}", 1), false},
		{"duplicate-binding", strings.ReplaceAll(inspection, "tests", "source"), false},
		{"too-few-bindings", strings.Replace(strings.Replace(inspection, "[status, source, tests]", "[status, source]", 1), "{status,source,tests}", "{status,source}", 1), false},
		{"too-many-bindings", strings.Replace(strings.Replace(inspection, "[status, source, tests]", "[status, source, tests, extra]", 1), "{status,source,tests}", "{status,source,tests,extra}", 1), false},
		{"default", strings.Replace(inspection, "[status,", "[status = mutate(),", 1), false},
		{"rest", strings.Replace(inspection, "source, tests]", "source, ...tests]", 1), false},
		{"hole", strings.Replace(inspection, "status, source, tests]", "status, , tests]", 1), false},
		{"alias", strings.Replace(inspection, "{status,source,tests}", "{status:source,source,tests}", 1), false},
		{"computed-key", strings.Replace(inspection, "{status,source,tests}", "{[status]:status,source,tests}", 1), false},
		{"spread-key", strings.Replace(inspection, "{status,source,tests}", "{...status,source,tests}", 1), false},
		{"transformed-value", strings.Replace(inspection, "{status,source,tests}", "{status:status.output,source,tests}", 1), false},
		{"json-replacer", strings.Replace(inspection, "{status,source,tests}", "{status,source,tests},mutate", 1), false},
		{"extra-statement", inspection + `mutate();`, false},
		{"mutated-binding", strings.Replace(inspection, "text(JSON", "status.exit_code=0; text(JSON", 1), false},
		{"unawaited", strings.Replace(inspection, "await Promise", "Promise", 1), false},
		{"settled", strings.Replace(inspection, "Promise.all(", "Promise.allSettled(", 1), false},
		{"spread-call", strings.Replace(inspection, "[tools.exec_command", "[...tools.exec_command", 1), false},
		{"dynamic-command", strings.Replace(inspection, `"git status --short"`, "command", 1), false},
		{"write", strings.Replace(inspection, "git status --short", "gofmt -w clamp.go", 1), false},
		{"test", strings.Replace(inspection, "git status --short", "go test ./... -json -count=1", 1), false},
		{"coverage", strings.Replace(inspection, "git status --short", "go test -cover ./...", 1), false},
		{"shell-effect", strings.Replace(inspection, "git status --short", "git status --short; touch clamp.go", 1), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, ok := nativeCodeModeCommands(tc.input, "/fixture")
			if ok != tc.want {
				t.Fatalf("named batch accepted=%v", ok)
			}
		})
	}
	for _, name := range []string{"text", "JSON", "tools", "Promise", "await", "null", "const"} {
		t.Run("shadow-"+name, func(t *testing.T) {
			input := strings.ReplaceAll(inspection, "source", name)
			if _, ok := nativeCodeModeCommands(input, "/fixture"); ok {
				t.Fatal("shadowed execution/printing name accepted")
			}
		})
	}
	for _, size := range []int{64, 65} {
		names, calls := make([]string, size), make([]string, size)
		for index := range names {
			names[index] = "result" + strconv.Itoa(index)
			calls[index] = `tools.exec_command({cmd:"git status --short",workdir:"/fixture"})`
		}
		input := "const [" + strings.Join(names, ",") + "] = await Promise.all([" + strings.Join(calls, ",") + "]); text(JSON.stringify({" + strings.Join(names, ",") + "}));"
		if commands, ok := nativeCodeModeCommands(input, "/fixture"); ok != (size == 64) || (ok && len(commands) != size) {
			t.Fatalf("named bound size=%d accepted=%v count=%d", size, ok, len(commands))
		}
	}
	for _, input := range []string{inspection, freshness} {
		for _, mode := range []string{"valid", "wrong_thread", "wrong_turn", "wrong_cwd", "wrong_command", "wrong_output", "wrong_output_turn", "duplicate_call", "duplicate_event", "duplicate_output", "missing_event", "missing_output", "interleaved_call", "incomplete_output", "invocation_cwd"} {
			t.Run(input+"/"+mode, func(t *testing.T) {
				commands, ok := nativeCodeModeCommands(input, "/fixture")
				if !ok || len(commands) != 3 {
					t.Fatal("observed named batch rejected")
				}
				r := codexNativeLiveReceipt{ChildID: "child", BoundHostSessionID: "parent", FixtureRoot: "/fixture", Caste: "builder"}
				var raw []byte
				add := func(v any) { b, _ := json.Marshal(v); raw = append(raw, append(b, '\n')...) }
				add(map[string]any{"type": "session_meta", "payload": map[string]any{"id": "child", "parent_thread_id": "parent", "cwd": "/fixture", "agent_role": "aether-builder"}})
				add(map[string]any{"type": "event_msg", "payload": map[string]any{"thread_id": "child", "turn_id": "turn"}})
				selected := input
				if mode == "invocation_cwd" {
					selected = strings.Replace(input, "/fixture", "/other", 1)
				}
				call := map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call", "name": "exec", "call_id": "batch", "input": selected, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": "turn"}}}
				add(call)
				if mode == "duplicate_call" {
					add(call)
				}
				if mode == "interleaved_call" {
					add(map[string]any{"type": "response_item", "payload": map[string]any{"type": "function_call", "name": "send_message", "call_id": "other"}})
				}
				for index, command := range commands {
					if mode == "missing_event" && index == 1 {
						continue
					}
					thread, turn, cwd, cmd := "child", "turn", command.Cwd, command.Command
					if index == 1 {
						switch mode {
						case "wrong_thread":
							thread = "foreign"
						case "wrong_turn":
							turn = "foreign"
						case "wrong_cwd":
							cwd = "/other"
						case "wrong_command":
							cmd = "cat go.mod"
						}
					}
					event := map[string]any{"type": "event_msg", "payload": map[string]any{"type": "item_completed", "thread_id": thread, "turn_id": turn, "item": map[string]any{"type": "CommandExecution", "id": "event" + strconv.Itoa(index), "status": "completed", "command": []string{"/bin/zsh", "-lc", cmd}, "cwd": cwd, "exit_code": 0, "aggregated_output": "actual inspection output"}}}
					add(event)
					if mode == "duplicate_event" && index == 1 {
						add(event)
					}
				}
				id, turn, header := "batch", "turn", "Script completed\n"
				switch mode {
				case "wrong_output":
					id = "other"
				case "wrong_output_turn":
					turn = "other"
				case "incomplete_output":
					header = "Script running with cell ID unfinished\n"
				}
				fields := []string{"status", "source", "tests"}
				if input == freshness {
					fields = []string{"diff", "status", "freshness"}
				}
				values := map[string]any{}
				for _, field := range fields {
					values[field] = map[string]any{"exit_code": 0, "output": "actual inspection output"}
				}
				printed, _ := json.Marshal(values)
				result := map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call_output", "call_id": id, "output": []any{map[string]any{"type": "input_text", "text": header}, map[string]any{"type": "input_text", "text": string(printed)}}, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": turn}}}
				if mode != "missing_output" {
					add(result)
				}
				if mode == "duplicate_output" {
					add(result)
				}
				nativeInspectChildEvents(&r, raw)
				if (len(r.ChildUnclassified) == 0) != (mode == "valid") || r.ChecksPassed || r.ChildEditObserved {
					t.Fatalf("mode=%s unclassified=%v checks=%v edits=%v", mode, r.ChildUnclassified, r.ChecksPassed, r.ChildEditObserved)
				}
			})
		}
	}
}

func TestCodexNativeObservedFixtureInspections(t *testing.T) {
	t.Run("single-invocation-semicolon", nativeObservedSemicolonInspection)
	t.Run("shared-inspection-atoms", nativeSharedInspectionAtoms)
	r := codexNativeLiveReceipt{FixtureRoot: "/fixture"}
	for _, tc := range []struct {
		command      string
		child, batch bool
	}{
		{`sed -n '1,200p' clamp.go`, true, true},
		{`git diff -- clamp.go && git status --short && date -u +%Y-%m-%dT%H:%M:%SZ`, true, false},
		{`git diff -- clamp.go clamp_test.go && date -u +%Y-%m-%dT%H:%M:%SZ`, true, false},
		{`git diff -- /fixture/clamp.go && date -u +%Y-%m-%dT%H:%M:%SZ`, true, false},
		{`git diff -- /other/clamp.go && date -u +%Y-%m-%dT%H:%M:%SZ`, false, false},
		{`git diff -- ../clamp.go && git status --short`, false, false},
		{`git diff --output=clamp.go -- clamp.go && git status --short`, false, false},
		{`git diff -- clamp.go --output=clamp.go && git status --short`, false, false},
		{`git diff -- clamp.go && date -u -s tomorrow`, false, false},
		{`git diff -- clamp.go && go test ./...`, false, false},
		{`git diff -- clamp.go && gofmt -w clamp.go`, false, false},
		{`git diff -- clamp.go && date -u +%Y-%m-%dT%H:%M:%SZ > clamp.go`, false, false},
		{`git diff -- clamp.go || git status --short`, false, false},
		{strings.Repeat(`git diff -- clamp.go && `, 7) + `date -u +%Y-%m-%dT%H:%M:%SZ`, true, false},
		{strings.Repeat(`git diff -- clamp.go && `, 8) + `date -u +%Y-%m-%dT%H:%M:%SZ`, false, false},
		{`sed -n '1,200p' clamp.go clamp_test.go`, true, true},
		{`sed -n '200p' clamp.go clamp_test.go`, true, true},
		{`sed -n '1,999999p' AGENTS.md clamp.go clamp_test.go double.go double_test.go go.mod`, true, true},
		{`sed -n '1,200p' /fixture/clamp.go clamp_test.go`, true, false},
		{`sed -n '1,200p' clamp.go clamp_test.go go.mod AGENTS.md double.go double_test.go clamp.go`, false, false},
		{`sed -n '1,200p' clamp.go clamp.go`, false, false},
		{`sed -n '1,200p' clamp.go /fixture/clamp.go`, false, false},
		{`sed -n '1,200p' clamp.go /other/clamp_test.go`, false, false},
		{`sed -n '1,200p' clamp.go ../clamp_test.go`, false, false},
		{`sed -n '1,200p' clamp.go other.go`, false, false},
		{`sed -n '1,200p' clamp.go -`, false, false},
		{`sed -n '1,200p' clamp.go --follow-symlinks`, false, false},
		{`sed -n '1,200p' clamp.go -i clamp_test.go`, false, false},
		{`sed -n -e '1,200p' clamp.go clamp_test.go`, false, false},
		{`sed -n -f clamp.go clamp_test.go`, false, false},
		{`sed -n '1,200p;w clamp.go' clamp.go clamp_test.go`, false, false},
		{`sed -n '1,200r' clamp.go clamp_test.go`, false, false},
		{`sed -n '1,200w' clamp.go clamp_test.go`, false, false},
		{`sed -n '1,1000000p' clamp.go clamp_test.go`, false, false},
		{`sed -n '1,200p' clamp.go clamp_test.go > clamp.go`, false, false},
		{`sed -n '1,200p' clamp.go clamp_test.go && go test ./...`, false, false},
		{`sed -n '1,200p' clamp.go clamp_test.go; gofmt -w clamp.go`, false, false},
		{`sed -n '1,200p' clamp.go $(touch clamp_test.go)`, false, false},
		{`sed -n '1,200p' /fixture/clamp_test.go`, true, true},
		{`gofmt -d clamp.go`, true, true},
		{`git diff --check -- clamp.go`, true, true},
		{`git diff --check`, true, true},
		{`rg --files`, true, true},
		{`rg --files -g '!*.sum'`, true, true},
		{`rg --files -g '!*vendor*'`, true, true},
		{`rg --files -g '!/.aether-transactions/**'`, true, true},
		{`rg --files -g '!nested/vendor/**'`, true, true},
		{`rg --files -g '!/../vendor/**'`, false, false},
		{`rg --files -g '!nested/./**'`, false, false},
		{`rg --files -g '!nested//**'`, false, false},
		{`rg --files -g '!/'`, false, false},
		{`rg --files -g '!nested/'`, false, false},
		{`rg --files -g '//vendor/**'`, false, false},
		{`rg --files -g '/vendor/**'`, false, false},
		{`rg --files -g '!/.aether-transactions/**' --hidden`, false, false},
		{`rg --files -g '!/.aether-transactions/**' --follow`, false, false},
		{`rg --files -g '!/.aether-transactions/**' /other`, false, false},
		{`rg --files -g '!/.aether-transactions/**' -g '!/.aether-transactions/**'`, false, false},
		{`rg --files -g !/.aether-transactions/**`, false, false},
		{`rg --files -g '!/.aether-transactions/'"**"`, false, false},
		{`rg --files -g '!/` + strings.Repeat("a", 125) + `/*'`, true, true},
		{`rg --files -g '!/` + strings.Repeat("a", 126) + `/*'`, false, false},
		{`date -Iseconds`, true, false},
		{`'date -Iseconds'`, false, false},
		{`rg --files -g 'AGENTS.md' -g 'CODEBASE.md' -g '*.go' -g 'go.mod'`, true, true},
		{`rg --files -g '*.go' -g '*.go'`, false, false},
		{`rg --files -g *.go`, false, false},
		{`rg --files -g '*'".go"`, false, false},
		{`rg --files -g '**.go'`, false, false},
		{`rg --files -g '../*.go'`, false, false},
		{`rg --files -g '/other/*.go'`, false, false},
		{`rg --files -g 'sub/*.go'`, false, false},
		{`rg --files -g '*.go' --hidden`, false, false},
		{`rg --files -g '*.go' --follow`, false, false},
		{`rg --files -g '*.go' /other`, false, false},
		{`rg --files -g '--pre'`, false, false},
		{`rg --files -g '{a,b}.go'`, false, false},
		{`rg --files -g '.'`, false, false},
		{`rg --files -g '..'`, false, false},
		{`rg --files -g '` + strings.Repeat("a", 128) + `'`, true, true},
		{`rg --files -g '` + strings.Repeat("a", 129) + `'`, false, false},
		{`rg --files -g 'a*' -g 'b*' -g 'c*' -g 'd*' -g 'e*' -g 'f*'`, true, true},
		{`rg --files -g 'a*' -g 'b*' -g 'c*' -g 'd*' -g 'e*' -g 'f*' -g 'g*'`, false, false},
		{`date --rfc-3339=seconds`, true, true},
		{`date --rfc-3339=date`, true, true},
		{`date --rfc-3339=ns`, true, true},
		{`date --rfc-3339=hours`, false, false},
		{`date --rfc-3339=seconds -s now`, false, false},
		{`date --rfc-3339=seconds 09201200`, false, false},
		{`date --rfc-3339=seconds > clamp.go`, false, false},
		{`date --rfc-3339=seconds; touch clamp.go`, false, false},
		{`'date --rfc-3339=seconds'`, false, false},
		{`date -Iseconds -s now`, false, false},
		{`date -Iseconds > clamp.go`, false, false},
		{`date -Iseconds /other`, false, false},

		{`rg --files -g "!temp-?.go"`, true, true},
		{`rg --files -g '!one' -g '!two' -g '!three' -g '!four' -g '!five' -g '!six'`, true, true},
		{`rg --files -g '!one' -g '!two' -g '!three' -g '!four' -g '!five' -g '!six' -g '!seven'`, false, false},
		{`rg --files -g '!*vendor*' -g '!*vendor*'`, false, false},
		{`rg --files -g '!*vendor*' --hidden`, false, false},
		{`rg --files -g '!*vendor*' --follow`, false, false},
		{`rg --files -g '!*vendor*' /other`, false, false},
		{`rg --files -g '!*vendor*' -g '--pre'`, false, false},
		{`rg --files -g !*vendor*`, false, false},
		{`rg --files -g '!vendor'"*"`, false, false},
		{`rg --files -g '!!vendor'`, false, false},
		{`rg --files -g '!'`, false, false},
		{`rg --files -g '!vendor/*'`, true, true},
		{`rg --files -g '!{one,two}'`, false, false},
		{`rg --files -g '![ab]'`, false, false},
		{`rg --files -g '!$(touch clamp.go)'`, false, false},
		{`rg --files -g '!*vendor*' > clamp.go`, false, false},
		{`rg --files -g '!*vendor*'; touch clamp.go`, false, false},
		{`rg --files -g '*vendor*'`, true, true},
		{"rg --files -g '!" + strings.Repeat("a", 128) + "'", true, true},
		{"rg --files -g '!" + strings.Repeat("a", 129) + "'", false, false},
		{`rg --files -g '!*.sum' -g 'clamp.go'`, true, true},
		{`rg --files -g '!*.sum' -g 'AGENTS.md' -g 'clamp.go' -g 'clamp_test.go' -g 'double.go' -g 'go.mod'`, true, true},
		{`rg --files -g '!*.sum' -g 'AGENTS.md' -g 'clamp.go' -g 'clamp_test.go' -g 'double.go' -g 'double_test.go' -g 'go.mod'`, false, false},
		{`rg --files -g '!*.sum' -g '!*.sum'`, false, false},
		{`rg --files -g '!*.sum' /other`, false, false},
		{`rg --files -g '!*.sum' --hidden`, false, false},
		{`rg --files -g '!*.sum' --follow`, false, false},
		{`rg --files -g '!*.sum' --pre mutate`, false, false},
		{`rg --files -g '!*.sum' --output clamp.go`, false, false},
		{`rg --files -g '!*.sum' -g '--follow'`, false, false},
		{`rg --files -g '!../*.sum'`, false, false},
		{`rg --files -g '!*.sum' > clamp.go`, false, false},
		{`rg --files -g '!*.sum' && go test ./...`, false, false},
		{`rg --files -g '!*.sum'; touch clamp.go`, false, false},
		{`rg --files -g "!$(touch clamp.go)"`, false, false},
		{`sed -n '1,240p' clamp.go; sed -n '1,280p' clamp_test.go; git status --short`, true, false},
		{`rg --files; git status --short`, true, false},
		// Generic parent-readonly syntax admits an absolute listing root;
		// the child classifier separately restricts operations to its fixture.
		{`rg --files /other`, false, true},
		{`rg --files --hidden`, false, false},
		{`rg --files --follow`, false, false},
		{`rg --files; go test ./...`, false, false},
		{`rg --files; gofmt -w clamp.go`, false, false},
		{`rg --files; /fixture/aether codex-native-worker context --request /fixture/request.json`, false, false},
		{`rg --files; cat ../clamp.go`, false, false},
		{`rg --files; sed -n '1,20p' /other/clamp.go`, false, false},
		{`rg --files; git status --short > clamp.go`, false, false},
		{`rg --files; $(touch clamp.go)`, false, false},
		{`rg --files; git status --short && git diff --check`, false, false},
		{`rg --files;; git status --short`, false, false},
		{`rg --files;`, false, false},
		{`;rg --files`, false, false},
		{strings.Repeat(`git status --short;`, 8), false, false},
		{strings.Repeat(`git status --short;`, 8) + `git status --short`, false, false},
		{`rg --files -g 'AGENTS.md' -g 'clamp.go' -g 'clamp_test.go' -g 'go.mod'`, true, true},
		{`rg --files -g 'AGENTS.md' -g 'clamp.go' -g 'clamp_test.go' -g 'go.mod' && git status --short`, true, false},
		{`sed -n '1,220p' clamp.go && sed -n '1,260p' clamp_test.go && sed -n '1,120p' go.mod`, true, false},
		{`test -r clamp.go`, true, true},
		{`date -u +%Y-%m-%dT%H:%M:%SZ`, true, true},
		{`go test ./... -cover -count=1`, true, false},
		{`go test ./... -cover`, true, false},
		{`go test -cover ./...`, true, false},
		{`rg --files -g '*.go'`, false, false},
		{`rg --files -g 'clamp.go' -g 'clamp.go'`, false, false},
		{`rg --files -g '../clamp.go'`, false, false},
		{`rg --files -g 'clamp.go' --pre mutate`, false, false},
		{`rg --files -g 'clamp.go' /other`, false, false},
		{`sed -n '1,200p' clamp.go && go test ./...`, false, false},
		{`sed -n '1,200p' clamp.go && gofmt -w clamp.go`, false, false},
		{`sed -n '1,200p' clamp.go && sed -n '1,200p' /other/clamp.go`, false, false},
		{`sed -n '1,200p' clamp.go && git status --short; touch clamp.go`, false, false},
		{`sed -n '1,200p' clamp.go || git status --short`, false, false},
		{`sed -n '1,200p' clamp.go && git status --short > clamp.go`, false, false},
		{`sed -n '1,200p' clamp.go && $(touch clamp.go)`, false, false},
		{`sed -n '1,200p' clamp.go && 'git status' --short`, false, false},
		{`sed -n '1,200p' clamp.go && 'git diff' --check`, false, false},
		{`sed -n '1,200p' clamp.go && `, false, false},
		{strings.Repeat(`git status --short && `, 8) + `git status --short`, false, false},
		{`sed -n '1,200p' /other/clamp.go`, false, true},
		{`sed -n '1,200e' clamp.go`, false, false},
		{`sed -i '' clamp.go`, false, false},
		{`gofmt -w clamp.go`, false, false},
		{`gofmt -d /other/clamp.go`, false, false},
		{`git diff --check -- /other/clamp.go`, false, false},
		{`git diff --output=/tmp/side-effect -- clamp.go`, false, false},
		{`test -r /other/clamp.go`, false, false},
		{`date -u -s tomorrow`, false, false},
		{`go test ./... -coverprofile=/tmp/profile`, false, false},
		{`go test ./... -cover -count=1; touch clamp.go`, false, false},
		{`go test -cover ./... -run TestClamp`, false, false},
		{`go test -cover ./... -coverprofile=/tmp/profile`, false, false},
		{`go test -cover ./...; touch clamp.go`, false, false},
		{`go test -cover ./... > /tmp/result`, false, false},
		{`go test -cover ./other`, false, false},
	} {
		t.Run(tc.command, func(t *testing.T) {
			if got := nativeChildCommandAllowed(r, []string{"/bin/sh", "-c", tc.command}); got != tc.child {
				t.Fatalf("child=%v want%v", got, tc.child)
			}
			if got := nativeReadOnlyBatchCommand(tc.command); got != tc.batch {
				t.Fatalf("batch=%v want%v", got, tc.batch)
			}
		})
	}
	for _, command := range []string{`gofmt -w clamp.go`, `go test ./... -cover -count=1`, `go test -cover ./...`} {
		if nativeParentCoordinationCommand(&r, []string{"/bin/sh", "-c", command}, "/fixture") {
			t.Fatalf("child operation admitted for parent: %s", command)
		}
	}
}

func nativeObservedSemicolonInspection(t *testing.T) {
	const chain = `sed -n '1,240p' clamp.go; sed -n '1,280p' clamp_test.go; git status --short`
	for _, command := range []string{`date --rfc-3339=seconds`, `pwd && rg --files -g 'AGENTS.md' -g 'CODEBASE.md' -g '*.go' -g 'go.mod' && sed -n '1,240p' clamp.go && sed -n '1,280p' clamp_test.go && sed -n '1,160p' go.mod && git status --short`, `rg --files -g '!/.aether-transactions/**' && date -Iseconds`, `rg --files -g '!/.aether-transactions/**'`, `pwd && rg --files -g '!*vendor*' && git status --short && sed -n '1,240p' clamp.go && sed -n '1,280p' clamp_test.go && sed -n '1,160p' go.mod`, `pwd; cat clamp.go; git diff -- clamp.go`, `git diff -- clamp.go && git status --short && date -u +%Y-%m-%dT%H:%M:%SZ`, chain, `sed -n '1,200p' clamp.go clamp_test.go`, `sed -n '1,200p' /fixture/clamp.go clamp_test.go`, `rg --files`, `rg --files -g '!*.sum'`, `rg --files -g '!*vendor*'`, `rg --files /other`, `rg --files -g '!*.sum' /other`} {
		for _, mode := range []string{"valid", "invocation-cwd", "wrong-cwd", "wrong-thread", "wrong-turn", "missing-event", "duplicate-event", "split-events", "changed-command", "missing-output", "duplicate-output", "interleaved-call", "prior-pass", "failed-inspection", "later-failed-check", "later-edit", "failed-no-prior", "result-exit-mismatch", "result-output-mismatch", "extra-result"} {
			rfc := command == "date --rfc-3339=seconds"
			if !rfc && (mode == "failed-no-prior" || strings.HasPrefix(mode, "result-") || mode == "extra-result") {
				continue
			}
			if rfc && mode == "split-events" {
				continue
			}
			if !rfc && !strings.Contains(command, " && ") && !strings.Contains(command, ";") && (mode == "prior-pass" || mode == "failed-inspection" || mode == "later-failed-check" || mode == "later-edit") {
				continue
			}
			// Bare listing uses the pre-existing direct-command classifier;
			// only its actual invocation/event workspace needs a new control.
			if strings.HasPrefix(command, "rg --files") && mode != "valid" && mode != "invocation-cwd" && mode != "wrong-cwd" {
				continue
			}
			if strings.HasSuffix(command, " /other") && mode != "valid" {
				continue
			}
			t.Run(command+"/"+mode, func(t *testing.T) {
				r := codexNativeLiveReceipt{ChildID: "child", BoundHostSessionID: "parent", FixtureRoot: "/fixture", Caste: "builder"}
				var raw []byte
				add := func(v any) { b, _ := json.Marshal(v); raw = append(raw, append(b, '\n')...) }
				add(map[string]any{"type": "session_meta", "payload": map[string]any{"id": "child", "parent_thread_id": "parent", "cwd": "/fixture", "agent_role": "aether-builder"}})
				add(map[string]any{"type": "event_msg", "payload": map[string]any{"thread_id": "child", "turn_id": "turn"}})
				cwd := "/fixture"
				if mode == "invocation-cwd" {
					cwd = "/other"
				}
				if mode == "prior-pass" || mode == "failed-inspection" || mode == "later-failed-check" || mode == "later-edit" {
					raw = append(raw, nativeEvidenceEvent(t, "item_completed", "child", map[string]any{"type": "CommandExecution", "status": "completed", "id": "uncached", "command": []string{"/bin/sh", "-c", "go test ./... -json -count=1"}, "cwd": "/fixture", "exit_code": 0, "aggregated_output": "{\"Action\":\"run\",\"Package\":\"example.invalid/nativefixture\",\"Test\":\"TestClamp\"}\n{\"Action\":\"pass\",\"Package\":\"example.invalid/nativefixture\",\"Test\":\"TestClamp\"}\n{\"Action\":\"pass\",\"Package\":\"example.invalid/nativefixture\"}\n"})...)
				}
				input := "const r = await tools.exec_command({cmd:" + strconv.Quote(command) + ",workdir:" + strconv.Quote(cwd) + "}); text(r)\n"
				commands, ok := nativeCodeModeCommands(input, "/fixture")
				if !ok || len(commands) != 1 || commands[0].Command != command {
					t.Fatal("one shell invocation was split or rewritten")
				}
				add(map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call", "name": "exec", "call_id": "inspection", "input": input, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": "turn"}}})
				if mode == "interleaved-call" {
					add(map[string]any{"type": "response_item", "payload": map[string]any{"type": "function_call", "name": "send_message", "call_id": "other"}})
				}
				thread, turn, observed := "child", "turn", command
				switch mode {
				case "wrong-cwd":
					cwd = "/other"
				case "wrong-thread":
					thread = "other"
				case "wrong-turn":
					turn = "other"
				case "changed-command":
					observed = "git status --short"
				}
				event := func(id, command string) map[string]any {
					exit, status := 0, "completed"
					if mode == "failed-inspection" || mode == "failed-no-prior" {
						exit, status = 1, "failed"
					}
					return map[string]any{"type": "event_msg", "payload": map[string]any{"type": "item_completed", "thread_id": thread, "turn_id": turn, "item": map[string]any{"type": "CommandExecution", "id": id, "status": status, "command": []string{"/bin/zsh", "-lc", command}, "cwd": cwd, "exit_code": exit, "aggregated_output": "actual inspection output"}}}
				}
				if mode == "split-events" {
					parts := strings.Split(command, ";")
					if strings.Contains(command, " && ") {
						parts = strings.Split(command, " && ")
					}
					if len(parts) == 1 && strings.HasPrefix(command, "sed ") {
						parts = []string{`sed -n '1,200p' clamp.go`, `sed -n '1,200p' clamp_test.go`}
					}
					for index, part := range parts {
						add(event("split"+strconv.Itoa(index), strings.TrimSpace(part)))
					}
				} else if mode != "missing-event" {
					add(event("event", observed))
					if mode == "duplicate-event" {
						add(event("event", observed))
					}
				}
				result := map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call_output", "call_id": "inspection", "output": []any{map[string]any{"type": "input_text", "text": "Script completed\n"}, map[string]any{"type": "input_text", "text": `{"exit_code":0,"output":"actual inspection output"}`}}, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": "turn"}}}
				if rfc {
					body := map[string]any{"exit_code": 0, "output": "actual inspection output"}
					if mode == "failed-inspection" || mode == "failed-no-prior" || mode == "result-exit-mismatch" {
						body["exit_code"] = 1
					}
					if mode == "result-output-mismatch" {
						body["output"] = "forged"
					}
					b, _ := json.Marshal(body)
					payload := result["payload"].(map[string]any)
					parts := payload["output"].([]any)
					parts[1] = map[string]any{"type": "input_text", "text": string(b)}
					if mode == "extra-result" {
						payload["output"] = append(parts, parts[1])
					}
				}
				if mode != "missing-output" {
					add(result)
				}
				if mode == "duplicate-output" {
					add(result)
				}
				if mode == "later-failed-check" {
					raw = append(raw, nativeEvidenceEvent(t, "item_completed", "child", map[string]any{"type": "CommandExecution", "status": "completed", "id": "failed", "command": []string{"/bin/sh", "-c", "go test ./..."}, "cwd": "/fixture", "exit_code": 1, "aggregated_output": "FAIL"})...)
				}
				if mode == "later-edit" {
					raw = append(raw, nativeEvidenceEvent(t, "item_completed", "child", map[string]any{"type": "FileChange", "status": "completed", "changes": map[string]any{"/fixture/clamp.go": map[string]any{"type": "update", "unified_diff": "@@ -1,1 +1,1 @@\n-package nativefixture\n+package nativefixture\n"}}})...)
				}
				nativeInspectChildEvents(&r, raw)
				wantClassified := (mode == "valid" || mode == "prior-pass" || mode == "later-failed-check" || mode == "later-edit" || (rfc && (mode == "failed-inspection" || mode == "failed-no-prior"))) && !strings.HasSuffix(command, " /other")
				if (len(r.ChildUnclassified) == 0) != wantClassified || r.ChecksPassed != (mode == "prior-pass" || (rfc && mode == "failed-inspection")) || (mode != "later-edit" && r.ChildEditObserved) {
					t.Fatalf("unclassified=%v checks=%v edits=%v", r.ChildUnclassified, r.ChecksPassed, r.ChildEditObserved)
				}
			})
		}
	}
}

func TestCodexNativeObservedLiteralPatch(t *testing.T) {
	patch := "*** Begin Patch\n*** Update File: /fixture/clamp.go\n@@\n-old\n+new\n*** End Patch"
	input := "const patch = " + strconv.Quote(patch) + "; text(await tools.apply_patch(patch));"
	for _, mode := range []string{"valid", "result-alias", "result-wrong-patch", "result-wrong-output", "result-reassigned", "result-unawaited", "result-shadowed", "result-trailing", "computed", "wrong-variable", "trailing", "missing-change", "wrong-child", "wrong-turn", "duplicate-change", "duplicate-call", "wrong-output", "protected-file"} {
		t.Run(mode, func(t *testing.T) {
			selected := input
			if strings.HasPrefix(mode, "result-") {
				selected = "const patch = " + strconv.Quote(patch) + "; const result = await tools.apply_patch(patch); text(result);"
			}
			switch mode {
			case "result-wrong-patch":
				selected = strings.Replace(selected, "apply_patch(patch)", "apply_patch(other)", 1)
			case "result-wrong-output":
				selected = strings.Replace(selected, "text(result)", "text(other)", 1)
			case "result-reassigned":
				selected = strings.Replace(selected, "text(result)", `result = {}; text(result)`, 1)
			case "result-unawaited":
				selected = strings.Replace(selected, "await ", "", 1)
			case "result-shadowed":
				selected = strings.ReplaceAll(selected, "result", "patch")
			case "result-trailing":
				selected += `mutate();`
			case "computed":
				selected = strings.Replace(selected, "; text", `+suffix; text`, 1)
			case "wrong-variable":
				selected = strings.Replace(selected, "apply_patch(patch)", "apply_patch(other)", 1)
			case "trailing":
				selected += `mutate();`
			}
			r := codexNativeLiveReceipt{ChildID: "child", BoundHostSessionID: "parent", FixtureRoot: "/fixture", BaselineSource: "old\n", FinalSource: "new\n"}
			var raw []byte
			add := func(v any) { b, _ := json.Marshal(v); raw = append(raw, append(b, '\n')...) }
			add(map[string]any{"type": "session_meta", "payload": map[string]any{"id": "child", "parent_thread_id": "parent", "agent_role": "aether-builder", "cwd": "/fixture"}})
			call := map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call", "name": "exec", "call_id": "call", "input": selected, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": "turn"}}}
			add(call)
			if mode == "duplicate-call" {
				add(call)
			}
			thread, turn, path := "child", "turn", "/fixture/clamp.go"
			switch mode {
			case "wrong-child":
				thread = "sibling"
			case "wrong-turn":
				turn = "other"
			case "protected-file":
				path = "/fixture/clamp_test.go"
			}
			change := map[string]any{"type": "event_msg", "payload": map[string]any{"type": "item_completed", "thread_id": thread, "turn_id": turn, "item": map[string]any{"type": "FileChange", "id": "edit", "status": "completed", "changes": map[string]any{path: map[string]any{"type": "update", "unified_diff": "@@ -1,1 +1,1 @@\n-old\n+new\n"}}}}}
			if mode != "missing-change" {
				add(change)
			}
			if mode == "duplicate-change" {
				add(change)
			}
			id := "call"
			if mode == "wrong-output" {
				id = "other"
			}
			add(map[string]any{"type": "response_item", "payload": map[string]any{"type": "custom_tool_call_output", "call_id": id, "output": []any{map[string]any{"type": "input_text", "text": "Script completed\n"}, map[string]any{"type": "input_text", "text": "{}"}}, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": "turn"}}})
			valid := mode == "valid" || mode == "result-alias"
			if got := nativeCorroboratedLiteralPatch(r, raw, "call"); got != valid {
				t.Fatalf("corroborated=%v", got)
			}
			if valid {
				nativeInspectChildEvents(&r, raw)
				if !r.ChildEditObserved || len(r.ChildUnclassified) != 0 {
					t.Fatalf("literal alias child edit unclassified: %v", r.ChildUnclassified)
				}
			}
			parent := codexNativeLiveReceipt{SchemaVersion: "codex-native-tracer/v2", SessionID: "child", FixtureRoot: "/fixture"}
			nativeInspectParentEvents(&parent, raw)
			if !parent.ParentSubstitution {
				t.Fatal("parent literal alias patch accepted")
			}
		})
	}
}

func nativePresentationPlanBoundaries(t *testing.T) {
	const prefix = `const r = await tools.exec_command({cmd:"git status --short",workdir:"/fixture"}); `
	const suffix = "text(r.output); text(`\\nexit_code=${r.exit_code}`);"
	for _, tc := range []struct{ name, input string }{
		{"json-duplicate-key", prefix + `text(JSON.stringify({exit_code:r.exit_code,output:r.output,exit_code:r.exit_code}));`},
		{"json-extra-key", prefix + `text(JSON.stringify({exit_code:r.exit_code,output:r.output,extra:0}));`},
		{"json-omitted-output", prefix + `text(JSON.stringify({exit_code:r.exit_code}));`},
		{"json-omitted-exit", prefix + `text(JSON.stringify({output:r.output}));`},
		{"json-forged-exit", prefix + `text(JSON.stringify({exit_code:0,output:r.output}));`},
		{"json-transformed-output", prefix + `text(JSON.stringify({exit_code:r.exit_code,output:r.output.trim()}));`},
		{"json-wrong-alias", prefix + `text(JSON.stringify({exit_code:q.exit_code,output:r.output}));`},
		{"json-computed", prefix + `text(JSON.stringify({exit_code:r["exit_code"],output:r.output}));`},
		{"json-spread", prefix + `text(JSON.stringify({...r}));`},
		{"json-prototype", prefix + `text(JSON.stringify({exit_code:r.exit_code,output:r.__proto__.output}));`},
		{"json-extra-print", prefix + `text(JSON.stringify({exit_code:r.exit_code,output:r.output})); text(r);`},
		{"json-reserved-alias", `const JSON = await tools.exec_command({cmd:"git status --short",workdir:"/fixture"}); text(JSON.stringify({exit_code:JSON.exit_code,output:JSON.output}));`},
		{"extra-exec", prefix + suffix + `text(await tools.exec_command({cmd:"touch clamp.go",workdir:"/fixture"}));`},
		{"extra-print", prefix + suffix + `text(r.output);`},
		{"omitted-output", prefix + "text(`exit_code=${r.exit_code}`);"},
		{"omitted-exit", prefix + `text(r.output);`},
		{"duplicate-output", prefix + "text(`${r.output}\\n${r.output}\\nexit_code=${r.exit_code}`);"},
		{"duplicate-exit", prefix + "text(`${r.output}\\nexit_code=${r.exit_code}${r.exit_code}`);"},
		{"other-property", prefix + strings.Replace(suffix, "r.exit_code", "r.stderr", 1)},
		{"computed-property", prefix + strings.Replace(suffix, "r.output", `r["output"]`, 1)},
		{"prototype", prefix + strings.Replace(suffix, "r.output", "r.__proto__.output", 1)},
		{"transformed", prefix + strings.Replace(suffix, "r.output", "r.output.trim()", 1)},
		{"fake-exit", prefix + strings.Replace(suffix, "${r.exit_code}", "0", 1)},
		{"wrong-alias", prefix + strings.Replace(suffix, "r.exit_code", "other.exit_code", 1)},
		{"long-label", prefix + strings.Replace(suffix, "exit_code", strings.Repeat("L", 33), 1)},
		{"label-expression", prefix + strings.Replace(suffix, "exit_code=", "${mutate()}=", 1)},
		{"label-escape", prefix + strings.Replace(suffix, "exit_code=", `exit\u005fcode=`, 1)},
		{"label-newline", prefix + strings.Replace(suffix, "exit_code=", "exit\ncode=", 1)},
		{"output-suffix", prefix + "text(`exit_code=${r.exit_code}\\n${r.output}forged`);"},
		{"missing-separator", prefix + strings.Replace(suffix, "); text", ")\ntext", 1)},
		{"middle-eof-effect", prefix + strings.TrimSuffix(suffix, ";") + "\nmutate();"},
		{"mutate-result", prefix + `r.exit_code=0; ` + suffix},
		{"dynamic-command", strings.Replace(prefix, `"git status --short"`, `command`, 1) + suffix},
		{"unawaited", strings.Replace(prefix, "await ", "", 1) + suffix},
		{"reserved-alias", strings.ReplaceAll(strings.ReplaceAll(prefix+suffix, "const r =", "const text ="), "r.", "text.")},
		{"reserved-uppercase-no-fallback", strings.ReplaceAll(strings.ReplaceAll(strings.Replace(prefix+suffix, "exit_code=", "EXIT_CODE=", 1), "const r =", "const text ="), "r.", "text.")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, ok := nativeCodeModePresentation(tc.input, "/fixture"); ok {
				t.Fatal("unsafe/incomplete presentation accepted")
			}
			// Existing single-output projection remains an explicitly separate
			// grammar; rejecting it as a complete presentation must not remove it.
			if tc.name != "omitted-exit" {
				if _, ok := nativeCodeModeCommands(tc.input, "/fixture"); ok {
					t.Fatal("unsafe presentation escaped via another decoder")
				}
			}
		})
	}
}

func nativeSharedInspectionAtoms(t *testing.T) {
	r := codexNativeLiveReceipt{FixtureRoot: "/fixture"}
	for _, tc := range []struct {
		command string
		allowed bool
	}{
		{`'pwd'`, true},
		{`'git' 'status' '--short'`, true},
		{`'date' '-u' '+%Y-%m-%dT%H:%M:%SZ'`, true},
		{`'date' '-Iseconds'`, true},
		{`'date -Iseconds'`, false},
		{`date -Iseconds -s now`, false},
		{`'git status' --short`, false},
		{`git 'status --short'`, false},
		{`'git status --short'`, false},
		{`'date -u' +%Y-%m-%dT%H:%M:%SZ`, false},
		{`date '-u +%Y-%m-%dT%H:%M:%SZ'`, false},
		{`'date -u +%Y-%m-%dT%H:%M:%SZ'`, false},
	} {
		t.Run("argv-boundaries/"+tc.command, func(t *testing.T) {
			if nativeChildInspectionAtom(r, tc.command) != tc.allowed {
				t.Fatal("inspection atom lost argv boundaries")
			}
			for _, command := range []string{tc.command, "pwd && " + tc.command, "pwd; " + tc.command} {
				if got := nativeChildCommandAllowed(r, []string{"/bin/sh", "-c", command}); got != tc.allowed {
					t.Fatalf("command %q classified=%v want=%v", command, got, tc.allowed)
				}
			}
		})
	}
	atoms := []string{"date -Iseconds", "rg --files -g '!/.aether-transactions/**'", "pwd", "git status --short", "date -u +%Y-%m-%dT%H:%M:%SZ", "cat clamp.go go.mod", "git diff -- /fixture/clamp.go", "git diff --check", "rg --files -g '!*vendor*'", "sed -n '1,240p' clamp.go clamp_test.go", "gofmt -d clamp.go", "test -r go.mod"}
	for _, atom := range atoms {
		t.Run(atom, func(t *testing.T) {
			if !nativeChildInspectionAtom(r, atom) || !nativeChildCommandAllowed(r, []string{"/bin/sh", "-c", atom}) {
				t.Fatal("standalone atom rejected")
			}
			for _, other := range atoms {
				for _, separator := range []string{" && ", "; "} {
					if !nativeChildCommandAllowed(r, []string{"/bin/sh", "-c", atom + separator + other}) {
						t.Fatalf("composition differs: %s%s%s", atom, separator, other)
					}
				}
			}
		})
	}
	for _, bad := range []string{"pwd -P", "pwd /other", "ls /other", "cat /other/clamp.go", "cat ../clamp.go", "git diff -- /other/clamp.go", "rg --files /other", "rg --files -g !*vendor*", "sed -n '1,3w' clamp.go", "sed -i '1,3p' clamp.go", "gofmt -w clamp.go", "GOCACHE=/fixture/cache cat clamp.go", "go test ./...", "go test ./... -json -count=1", "aether codex-native-worker context --request /fixture/request.json", "pwd > clamp.go", "pwd || cat clamp.go", "pwd && cat clamp.go; pwd", "pwd; cat clamp.go && pwd", "pwd $(touch clamp.go)", "pwd | cat clamp.go", "pwd & cat clamp.go", "pwd\ncat clamp.go", "", strings.Repeat("pwd && ", 8) + "pwd"} {
		t.Run("reject/"+bad, func(t *testing.T) {
			// Test and context commands may be individually permitted after ACK;
			// neither is an inspection atom and neither may enter a read chain.
			if nativeChildInspectionAtom(r, bad) {
				t.Fatal("noninspection atom accepted")
			}
			for _, separator := range []string{" && ", "; "} {
				if nativeChildCommandAllowed(r, []string{"/bin/sh", "-c", "pwd" + separator + bad}) {
					t.Fatal("noninspection composition accepted")
				}
			}
		})
	}
	if !nativeChildCommandAllowed(r, []string{"/bin/sh", "-c", "GOCACHE=/fixture/cache cat clamp.go"}) {
		t.Fatal("existing standalone cache-prefix read changed")
	}
	for _, separator := range []string{" && ", "; "} {
		if !nativeChildCommandAllowed(r, []string{"/bin/sh", "-c", strings.Repeat("pwd"+separator, 7) + "pwd"}) {
			t.Fatal("eight atom boundary rejected")
		}
	}
}
