package cmd

import (
	"bytes"
	"encoding/json"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCodexNativeBuildGuidePlatformIsolation(t *testing.T) {
	baseline := map[string]commandGuideResult{}
	for _, platform := range []string{"codex", "claude", "opencode", "opencode", "codex", "claude", "codex"} {
		guide, err := buildCommandGuide("build", platform)
		if err != nil {
			t.Fatal(err)
		}
		if before, ok := baseline[platform]; ok && !reflect.DeepEqual(before, guide) {
			t.Fatalf("%s guide changed after another platform read", platform)
		}
		baseline[platform] = guide
		// Include every instruction-bearing field, including raw bypass text.
		all := strings.Join(append(append(append([]string{guide.Intent, guide.RunCommand, guide.RawBypass}, guide.PreSteps...), guide.PostSteps...), guide.DriftGuards...), "\n")
		if platform == "codex" {
			for _, required := range []string{"codex-native-worker reserve", "codex-native-worker bind", "codex-native-worker record", "codex-native-worker stage", "spawn_agent", "sole launcher"} {
				if !strings.Contains(all, required) {
					t.Errorf("Codex guide missing %q", required)
				}
			}
			if strings.Contains(all, "build-wave playbook") {
				t.Error("Codex guide still delegates authority to retired playbook")
			}
		} else {
			for _, forbidden := range []string{"codex-native-worker", "spawn_agent"} {
				if strings.Contains(all, forbidden) {
					t.Errorf("%s includes %s", platform, forbidden)
				}
			}
			if guide.SkillReference != "" || !strings.Contains(guide.PreSteps[0], "generated "+platform+" slash-command wrapper") {
				t.Errorf("%s lost wrapper entrypoint", platform)
			}
			for _, required := range []string{"visible live Task/subagent", "spawn-log", "spawn-complete", "build-completion-stage"} {
				if !strings.Contains(all, required) {
					t.Errorf("%s lost %q", platform, required)
				}
			}
			if guide.RunCommand != "AETHER_OUTPUT_MODE=json aether build-finalize <phase> --completion-file <Go-owned completion_path returned by build-completion-stage>" {
				t.Errorf("%s finalizer changed: %s", platform, guide.RunCommand)
			}
		}
	}
	// The adapter must not mutate caller-owned instruction slices either.
	def := commandGuideCatalog()["build"]
	before, _ := json.Marshal(def)
	_ = adaptCommandGuideDefinitionForPlatform("build", "codex", def)
	after, _ := json.Marshal(def)
	if string(before) != string(after) {
		t.Fatal("Codex adapter mutated shared definition")
	}
}

// A deterministic host-boundary double proves journal/credit behavior only;
// the separately opted-in fresh-host test is the native execution proof.
func TestCodexNativeWorkerTracer(t *testing.T) {
	source := filepath.Join(antSkillSourceRoot(t), "cmd", "codex_native_worker.go")
	syntax, err := parser.ParseFile(token.NewFileSet(), source, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, imported := range syntax.Imports {
		if imported.Path.Value == "\"os/exec\"" {
			t.Fatal("native bridge imports a process launcher")
		}
	}
	forbidden := map[string]bool{"invokeInternalWorker": true, "runInternalWorkerAdapter": true, "runInternalWorkerAdapterForPhase": true, "executeCodexBuildDispatches": true}
	ast.Inspect(syntax, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok {
			if name, ok := call.Fun.(*ast.Ident); ok && forbidden[name.Name] {
				t.Errorf("native bridge invokes second launcher %s", name.Name)
			}
		}
		return true
	})
	root := setupExternalBuildAttemptTest(t)
	manifest := prepareBoundBuildManifestOnly(t, root)
	if len(manifest.Dispatches) != 1 {
		t.Fatalf("ordinary fixture selected %d workers", len(manifest.Dispatches))
	}
	dispatch := manifest.Dispatches[0]
	root, _ = filepath.EvalSymlinks(root)
	request := codexNativeWorkerRequest{SchemaVersion: 1, Phase: 1, ExecutionBinding: *manifest.ExecutionBinding, WorkerName: dispatch.Name, TaskID: normalizedDispatchTaskID(dispatch), HostSessionID: "unit-host", Workspace: root, HostPermission: "workspace_write"}
	call := func(operation string, request codexNativeWorkerRequest, wantOK bool) codexNativeWorkerResponse {
		t.Helper()
		raw, _ := json.Marshal(request)
		var value map[string]interface{}
		if err := json.Unmarshal(raw, &value); err != nil {
			t.Fatal(err)
		}
		path := writeCodexNativeRequestForTest(t, value)
		var output bytes.Buffer
		stdout, stderr = &output, &output
		resetFlags(rootCmd)
		rootCmd.SetArgs([]string{"codex-native-worker", operation, "--request", path})
		rootCmd.SetOut(&output)
		rootCmd.SetErr(&output)
		err := rootCmd.Execute()
		if wantOK && err != nil {
			t.Fatalf("%s: %v %s", operation, err, output.String())
		}
		var envelope struct {
			OK     bool                      `json:"ok"`
			Result codexNativeWorkerResponse `json:"result"`
		}
		if err := json.Unmarshal(output.Bytes(), &envelope); err != nil {
			t.Fatalf("%s decode: %v %s", operation, err, output.String())
		}
		if envelope.OK != wantOK {
			t.Fatalf("%s success=%v want=%v: %s", operation, envelope.OK, wantOK, output.String())
		}
		renderedCommandExitCode.Store(0)
		return envelope.Result
	}
	call("reserve", codexNativeWorkerRequest{SchemaVersion: 1, Phase: 1}, false)
	for _, field := range []string{"workspace", "permission", "membership"} {
		refused := request
		switch field {
		case "workspace":
			refused.Workspace = filepath.Dir(root)
		case "permission":
			refused.HostPermission = "repository_read_only"
		case "membership":
			refused.TaskID = "not-assigned"
		}
		call("reserve", refused, false)
	}
	first := call("reserve", request, true)
	if !first.LaunchAllowed || first.Worker.Native == nil || first.Worker.ProviderRunID == "" || first.Worker.ProcessID != 0 {
		t.Fatalf("bad reservation: %+v", first)
	}
	if lifecycleDigest([]byte(first.Worker.Native.Prompt)) != first.Worker.Native.PromptSHA256 {
		t.Fatal("prompt digest differs from actual launch bytes")
	}
	attemptPath, _, _ := loadLatestBuildAttempt(1)
	snapshot := func() []byte {
		raw, err := os.ReadFile(filepath.Join(store.BasePath(), attemptPath))
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	before := snapshot()
	replay := call("reserve", request, true)
	if replay.LaunchAllowed || !replay.Replay || replay.Worker.ProviderRunID != first.Worker.ProviderRunID || !bytes.Equal(before, snapshot()) {
		t.Fatal("reservation replay changed bytes or licensed a duplicate launch")
	}
	// A legacy subprocess may not replace this reserved native worker.
	if _, err := beginBuildAttemptWorkerRun(1, request.ExecutionBinding, internalWorkerDispatchRequest{WorkerName: dispatch.Name, TaskID: request.TaskID, Caste: dispatch.Caste}, "forbidden-provider", codex.PlatformCodex); err == nil {
		t.Fatal("provider route reclaimed a native reservation")
	}
	request.LaunchID, request.ChildID = first.Worker.ProviderRunID, "unit-child"
	request.DispatchSHA256, request.PromptSHA256 = first.Worker.Native.DispatchSHA256, first.Worker.Native.PromptSHA256
	request.Result = &internalWorkerResult{Name: dispatch.Name, Caste: dispatch.Caste, TaskID: request.TaskID, Status: "completed", Summary: "deterministic boundary result", FilesCreated: []string{"evidence.txt"}, Handoff: codex.WorkerHandoff{ChangedFiles: []string{"evidence.txt"}, CommandsRun: []string{"deterministic fixture boundary"}, VerificationStatus: "pass", NextWorkerInstructions: []string{"inspect saved deterministic receipt"}}}
	call("record", request, false) // no binding, no source event
	request.Result = nil
	bound := call("bind", request, true)
	if bound.Worker.Native.Release == "" {
		t.Fatal("bind returned no release")
	}
	before = snapshot()
	call("bind", request, true)
	if !bytes.Equal(before, snapshot()) {
		t.Fatal("bind replay wrote")
	}
	wrong := request
	wrong.ChildID = "other-child"
	call("bind", wrong, false)
	call("record", request, false) // deliberately empty
	if err := os.WriteFile(filepath.Join(root, "evidence.txt"), []byte("deterministic worker-boundary fixture\n"), 0600); err != nil {
		t.Fatal(err)
	}
	request.SourceEventID = "unit-event"
	request.SourceEventSHA256 = strings.Repeat("a", 64)
	// The installed Builder uses ant_name/tdd/code_written. Exercise its wire
	// shape through the registered request reader, not the internal Go struct.
	wire := map[string]any{"schema_version": 1, "phase": 1, "execution_binding": request.ExecutionBinding,
		"worker_name": request.WorkerName, "task_id": request.TaskID, "host_session_id": request.HostSessionID,
		"launch_id": request.LaunchID, "child_id": request.ChildID, "dispatch_sha256": request.DispatchSHA256,
		"prompt_sha256": request.PromptSHA256, "source_event_id": request.SourceEventID, "source_event_sha256": request.SourceEventSHA256,
		"result": map[string]any{"ant_name": dispatch.Name, "caste": dispatch.Caste, "task_id": request.TaskID,
			"status": "code_written", "summary": "Builder wire regression", "files_created": []string{"evidence.txt"},
			"tdd":     map[string]any{"cycles_completed": 1, "tests_added": 0, "coverage_percent": nil, "all_passing": true},
			"handoff": codex.WorkerHandoff{ChangedFiles: []string{"evidence.txt"}, VerificationStatus: "pass", CommandsRun: []string{"fixture check"}}}}
	loaded, err := loadCodexNativeWorkerRequest(writeCodexNativeRequestForTest(t, wire))
	if err != nil || loaded.Result == nil || loaded.Result.Name != dispatch.Name || loaded.Result.Status != "completed" {
		t.Fatalf("installed Builder wire refused: %+v %v", loaded.Result, err)
	}
	request.Result = &internalWorkerResult{Name: dispatch.Name, Caste: dispatch.Caste, TaskID: request.TaskID, Status: "completed", Summary: "deterministic boundary result", FilesCreated: []string{"evidence.txt"}, Handoff: codex.WorkerHandoff{ChangedFiles: []string{"evidence.txt"}, CommandsRun: []string{"deterministic fixture boundary"}, VerificationStatus: "pass", NextWorkerInstructions: []string{"inspect saved deterministic receipt"}}}
	before = snapshot()
	invalid := *request.Result
	invalid.Handoff.VerificationStatus = "Tests passed after the edit"
	wrong = request
	wrong.Result = &invalid
	call("record", wrong, false)
	invalid.Handoff = codex.WorkerHandoff{}
	call("record", wrong, false)
	if !bytes.Equal(before, snapshot()) {
		t.Fatal("invalid handoff poisoned the immutable terminal journal")
	}
	terminal := call("record", request, true)
	if terminal.Worker.ResultSHA256 == "" || terminal.Worker.Native.LaunchState != "terminal" {
		t.Fatal("terminal was not durable")
	}
	before = snapshot()
	call("record", request, true)
	if !bytes.Equal(before, snapshot()) {
		t.Fatal("terminal replay wrote")
	}
	changed := *request.Result
	changed.Summary = "conflicting result"
	wrong = request
	wrong.Result = &changed
	call("record", wrong, false)
	minimal := codexNativeWorkerRequest{SchemaVersion: 1, Phase: 1, ExecutionBinding: request.ExecutionBinding}
	inspected := call("inspect", minimal, true)
	if inspected.Complete || len(inspected.Workers) != 1 || inspected.Workers[0].ResultSHA256 != terminal.Worker.ResultSHA256 || !bytes.Equal(before, snapshot()) {
		t.Fatal("inspect changed journal or lost terminal-before-aggregate state")
	}
	staged := call("stage", minimal, true)
	completion, err := loadExternalBuildCompletion(filepath.Join(root, filepath.FromSlash(staged.CompletionPath)))
	if err != nil {
		t.Fatal(err)
	}
	firstFinal, state, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
	if err != nil {
		t.Fatal(err)
	}
	if firstFinal["idempotent"] != false || state.Plan.Phases[0].Tasks[0].Status != colony.TaskCompleted {
		t.Fatalf("no finalizer credit: %+v", firstFinal)
	}
	_, saved, _ := loadLatestBuildAttempt(1)
	eventCount, historyCount := len(state.Events), len(saved.History)
	again, state, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
	if err != nil {
		t.Fatal(err)
	}
	_, saved, _ = loadLatestBuildAttempt(1)
	if again["idempotent"] != true || len(saved.WorkerRuns) != 1 || len(saved.History) != historyCount || len(state.Events) != eventCount {
		t.Fatal("finalizer replay repeated worker/credit")
	}
}

func writeCodexNativeRequestForTest(t *testing.T, value any) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "aether-worker-request-native-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	path := filepath.Join(dir, "request.json")
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCodexNativeWorkerFixturePreparation(t *testing.T) {
	root := setupExternalBuildAttemptTest(t)
	nativePrepareLiveFixture(t, root, t.TempDir())
	manifest := prepareBoundBuildManifestOnly(t, root)
	if manifest.ExecutionBinding == nil || len(manifest.Dispatches) != 1 || manifest.PlanRevisionID == "" {
		t.Fatalf("fixture is not an accepted one-worker plan: %+v", manifest)
	}
}

func TestCodexNativeWorkerReceiptValidation(t *testing.T) {
	if err := validateCodexNativeLiveReceipt(codexNativeLiveReceipt{ExitStatus: 0}); err == nil {
		t.Fatal("empty live evidence passed")
	}
	path := os.Getenv("AETHER_CODEX_NATIVE_RECEIPT_PATH")
	if path == "" {
		return
	} // Only the negative assertion ran; this is no live proof.
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var receipt codexNativeLiveReceipt
	if err := json.Unmarshal(raw, &receipt); err != nil {
		t.Fatal(err)
	}
	for file, want := range receipt.Artifacts {
		bytes, err := os.ReadFile(file)
		if err != nil || lifecycleDigest(bytes) != want {
			t.Fatalf("retained artifact changed or missing: %s", file)
		}
	}
	receipt.SkillRead, receipt.SupportRead, receipt.GuideRead = false, false, false
	receipt.ChildEditObserved, receipt.ChecksPassed, receipt.CreditObserved = false, false, false
	receipt.TerminalCorroborated, receipt.SourceEventCorroborated, receipt.ParentSubstitution = false, false, false
	receipt.NativeSpawnCount = 0
	nativeCollectLiveEvidence(t, &receipt, t.TempDir(), filepath.Join(filepath.Dir(receipt.FixtureRoot), "home"))
	if err := validateCodexNativeLiveReceipt(receipt); err != nil {
		t.Fatalf("raw receipt replay: %v", err)
	}
	t.Logf("raw native receipt replay passed for attempt %s child %s", receipt.AttemptID, receipt.ChildID)
	// Evidence-derived facts are mandatory; a terminal claim alone cannot pass.
	for _, field := range []string{"child_edit", "child_check", "terminal", "source_event", "credit", "parent_substitution"} {
		bad := receipt
		switch field {
		case "child_edit":
			bad.ChildEditObserved = false
		case "child_check":
			bad.ChecksPassed = false
		case "terminal":
			bad.TerminalCorroborated = false
		case "source_event":
			bad.SourceEventCorroborated = false
		case "credit":
			bad.CreditObserved = false
		case "parent_substitution":
			bad.ParentSubstitution = true
		}
		if err := validateCodexNativeLiveReceipt(bad); err == nil {
			t.Fatalf("missing/invalid %s still passed live validation", field)
		}
	}
}
