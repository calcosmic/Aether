package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// All answers in this file are harness/predeclared authorization, never owner testimony.
func nativeDecisionFixture(t *testing.T) (codexNativeWorkerRequest, PendingDecision) {
	t.Helper()
	_, requests := nativeDecisionWorkersFixture(t, 1)
	request := requests[0]
	d, err := admitCodexNativeDecision(request, codexNativeQuestion{QuestionID: "harness-event-question-1", Question: "Which fixture boundary should this worker keep?"}, codexNativeDecisionHooks{})
	if err != nil {
		t.Fatal(err)
	}
	return request, d
}
func nativeAnswerForTest(d PendingDecision) codexNativeDecisionAnswerRequest {
	return codexNativeDecisionAnswerRequest{SchemaVersion: 1, Binding: *d.NativeBinding, Question: d.Description, Answer: "HARNESS_ONLY_NATIVE_ANSWER café 日本語"}
}
func nativeAnswerPathForTest(t *testing.T, request codexNativeDecisionAnswerRequest) string {
	t.Helper()
	raw, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err = json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	return writeCodexNativeRequestForTest(t, wire)
}
func nativeDecisionStoreBytes(t *testing.T) map[string]string {
	t.Helper()
	result := map[string]string{}
	err := filepath.WalkDir(store.BasePath(), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		result[path] = string(raw)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}
func TestCodexNativeDecisionScope(t *testing.T) {
	for _, change := range []string{"goal", "session", "attempt", "launch", "worker", "task", "child", "question-id", "question-text", "workspace", "host", "prompt", "dispatch", "run", "plan"} {
		t.Run(change, func(t *testing.T) {
			_, d := nativeDecisionFixture(t)
			answer := nativeAnswerForTest(d)
			var state colony.ColonyState
			if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "goal":
				*state.Goal += " changed"
				_ = store.SaveJSON("COLONY_STATE.json", state)
			case "session":
				changed := "changed-session"
				state.SessionID = &changed
				_ = store.SaveJSON("COLONY_STATE.json", state)
			case "run":
				changed := "changed-run"
				state.RunID = &changed
				_ = store.SaveJSON("COLONY_STATE.json", state)
			case "plan":
				state.Plan.Phases[0].Description += " changed"
				_ = store.SaveJSON("COLONY_STATE.json", state)
			case "attempt":
				answer.Binding.AttemptID += "changed"
			case "launch":
				answer.Binding.LaunchID += "changed"
			case "worker":
				answer.Binding.WorkerName += "changed"
			case "task":
				answer.Binding.TaskID += "changed"
			case "child":
				answer.Binding.ChildID += "changed"
			case "question-id":
				answer.Binding.QuestionID += "changed"
			case "question-text":
				answer.Question += "changed"
			case "workspace":
				answer.Binding.Workspace += "changed"
			case "host":
				answer.Binding.HostSessionID += "changed"
			case "prompt":
				answer.Binding.PromptSHA256 = strings.Repeat("0", 64)
			case "dispatch":
				answer.Binding.DispatchSHA256 = strings.Repeat("0", 64)
			}
			before := nativeDecisionStoreBytes(t)
			if _, _, err := answerCodexNativeDecision(answer, codexNativeDecisionHooks{}); err == nil {
				t.Fatal("stale/mismatched answer admitted")
			}
			if !reflect.DeepEqual(before, nativeDecisionStoreBytes(t)) {
				t.Fatal("refused answer mutated decisions/signals/interventions")
			}
		})
	}
}
func TestCodexNativeDecisionReplay(t *testing.T) {
	request, d := nativeDecisionFixture(t)
	before := nativeDecisionStoreBytes(t)
	again, err := admitCodexNativeDecision(request, codexNativeQuestion{QuestionID: "harness-event-question-1", Question: d.Description}, codexNativeDecisionHooks{})
	if err != nil || !reflect.DeepEqual(d, again) || !reflect.DeepEqual(before, nativeDecisionStoreBytes(t)) {
		t.Fatalf("question replay changed row/store: %v", err)
	}
	if _, err := admitCodexNativeDecision(request, codexNativeQuestion{QuestionID: "harness-event-question-1", Question: "changed question"}, codexNativeDecisionHooks{}); err == nil {
		t.Fatal("question identity admitted changed content")
	}
	answer := nativeAnswerForTest(d)
	resolved, replay, err := answerCodexNativeDecision(answer, codexNativeDecisionHooks{})
	if err != nil || replay || !resolved.Resolved {
		t.Fatalf("first answer: %#v %v %v", resolved, replay, err)
	}
	after := nativeDecisionStoreBytes(t)
	again, replay, err = answerCodexNativeDecision(answer, codexNativeDecisionHooks{})
	if err != nil || !replay || !reflect.DeepEqual(resolved, again) || !reflect.DeepEqual(after, nativeDecisionStoreBytes(t)) {
		t.Fatalf("answer replay rewrote decision/facts: %v", err)
	}
	answer.Answer += " conflicting"
	if _, _, err := answerCodexNativeDecision(answer, codexNativeDecisionHooks{}); err == nil {
		t.Fatal("conflicting answer admitted")
	}
	if !reflect.DeepEqual(after, nativeDecisionStoreBytes(t)) {
		t.Fatal("conflicting replay changed store")
	}
}
func TestCodexNativeDecisionIsolation(t *testing.T) {
	manifest, requests := nativeDecisionWorkersFixture(t, 2)
	question := codexNativeQuestion{QuestionID: "same-event-id", Question: "Should this fixture retain the archive?"}
	first, err := admitCodexNativeDecision(requests[0], question, codexNativeDecisionHooks{})
	if err != nil {
		t.Fatal(err)
	}
	second, err := admitCodexNativeDecision(requests[1], question, codexNativeDecisionHooks{})
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == second.ID {
		t.Fatal("question text collapsed distinct workers")
	}
	answer := nativeAnswerForTest(first)
	if _, _, err := answerCodexNativeDecision(answer, codexNativeDecisionHooks{}); err != nil {
		t.Fatal(err)
	}
	if len(answeredDecisionTexts(loadCurrentPendingDecisionScope())) != 0 {
		t.Fatal("native row suppressed ordinary handoff question")
	}
	if text := renderClarifiedIntentSection(); strings.Contains(text, answer.Answer) || strings.Contains(text, question.Question) {
		t.Fatal("native answer leaked into broad clarification")
	}
	context, _ := resolveCodexWorkerContextWithTrim()
	if strings.Contains(context, answer.Answer) || strings.Contains(context, question.Question) {
		t.Fatal("native question/answer leaked into capsule via signals or decisions")
	}
	for name, data := range nativeDecisionStoreBytes(t) {
		if filepath.Base(name) != pendingDecisionsFile && (strings.Contains(data, answer.Answer) || strings.Contains(data, question.Question)) {
			t.Fatalf("native text escaped protected store: %s", name)
		}
	}
	_ = manifest
}
func TestCodexNativeDecisionProtectedPaths(t *testing.T) {
	request, d := nativeDecisionFixture(t)
	before := nativeDecisionStoreBytes(t)
	if _, err := recordDecisionAnswer(d.Description, "unbound bypass", 1, "worker-handoff"); err == nil {
		t.Fatal("generic append minted a shadow native answer")
	}
	if _, err := resolveDiscussQuestion(d.ID, "discuss bypass"); err == nil {
		t.Fatal("discuss resolved native row")
	}
	raw := nativeDecisionOutputForTest(t, func() {
		pendingDecisionResolveCmd.SetArgs(nil)
		_ = pendingDecisionResolveCmd.Flags().Set("id", d.ID)
		_ = pendingDecisionResolveCmd.Flags().Set("resolution", "generic bypass")
		_ = pendingDecisionResolveCmd.RunE(pendingDecisionResolveCmd, nil)
	})
	if !strings.Contains(raw, "native") {
		t.Fatalf("generic resolve did not explain bound route: %s", raw)
	}
	if !reflect.DeepEqual(before, nativeDecisionStoreBytes(t)) {
		t.Fatal("generic route changed protected native row")
	}
	waiver := forcedReviewerWaiverQuestionText(1, queenRiskSignalTable[0].Name, queenRiskSignalTable[0].PlainEnglish)
	if _, err := admitCodexNativeDecision(request, codexNativeQuestion{QuestionID: "protected", Question: waiver}, codexNativeDecisionHooks{}); err == nil {
		t.Fatal("native material question impersonated reviewer authorization")
	}
}
func TestCodexNativeDecisionConcurrentAnswer(t *testing.T) {
	_, d := nativeDecisionFixture(t)
	request := nativeAnswerForTest(d)
	var wg sync.WaitGroup
	results := make(chan bool, 2)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, replay, err := answerCodexNativeDecision(request, codexNativeDecisionHooks{})
			results <- replay
			errs <- err
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	replayed := 0
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	for replay := range results {
		if replay {
			replayed++
		}
	}
	if replayed != 1 {
		t.Fatal("concurrent answer did not resolve exactly once")
	}
	var file PendingDecisionFile
	_ = store.LoadJSON(pendingDecisionsFile, &file)
	if len(file.Decisions) != 1 || !file.Decisions[0].Resolved {
		t.Fatalf("duplicate answer rows: %#v", file)
	}
}
func TestCodexNativeDecisionAnswerRoute(t *testing.T) {
	_, d := nativeDecisionFixture(t)
	answer := nativeAnswerForTest(d)
	path := nativeAnswerPathForTest(t, answer)
	command := decisionAnswerCmd
	resetFlags(command)
	if err := command.ParseFlags([]string{"--native-request", path}); err != nil {
		t.Fatal(err)
	}
	old := stdout
	var out bytes.Buffer
	stdout = &out
	t.Cleanup(func() { stdout = old; resetFlags(command) })
	if err := command.RunE(command, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), d.ID) || !strings.Contains(out.String(), "\"recorded\":true") {
		t.Fatalf("missing native answer receipt: %s", out.String())
	}
	for _, bad := range []string{strings.Repeat(" ", internalWorkerRequestMaxBytes+1), "{", fmt.Sprintf("{\"schema_version\":1,\"unknown\":true}")} {
		if err := os.WriteFile(path, []byte(bad), 0600); err != nil {
			t.Fatal(err)
		}
		before := nativeDecisionStoreBytes(t)
		if _, err := loadCodexNativeDecisionAnswerRequest(path); err == nil {
			t.Fatal("invalid answer request accepted")
		}
		if !reflect.DeepEqual(before, nativeDecisionStoreBytes(t)) {
			t.Fatal("bad request changed state")
		}
	}
}

func nativeDecisionOutputForTest(t *testing.T, call func()) string {
	t.Helper()
	old, oldErr := stdout, stderr
	var out bytes.Buffer
	stdout, stderr = &out, &out
	defer func() { stdout, stderr = old, oldErr }()
	call()
	return out.String()
}

func nativeDecisionWorkersFixture(t *testing.T, count int) (codexBuildManifest, []codexNativeWorkerRequest) {
	t.Helper()
	root := setupExternalBuildAttemptTest(t)
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatal(err)
	}
	session := "native-decision-fixture-session"
	state.SessionID = &session
	if count == 2 {
		id := "1.2"
		state.Plan.Phases[0].Tasks = append(state.Plan.Phases[0].Tasks, colony.Task{ID: &id, Goal: "Write independent second evidence", Status: colony.TaskPending})
	}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}
	manifest := prepareBoundBuildManifestOnly(t, root)
	if len(manifest.Dispatches) != count {
		t.Fatalf("got %d dispatches", len(manifest.Dispatches))
	}
	return manifest, nativeFinalizeReserve(t, manifest)
}
func TestCodexNativeDecisionQuestionAdmission(t *testing.T) {
	for _, field := range []string{"worker", "child", "task", "launch", "phase", "paused"} {
		t.Run(field, func(t *testing.T) {
			_, requests := nativeDecisionWorkersFixture(t, 1)
			request := requests[0]
			switch field {
			case "worker":
				request.WorkerName += "wrong"
			case "child":
				request.ChildID += "wrong"
			case "task":
				request.TaskID += "wrong"
			case "launch":
				request.LaunchID += "wrong"
			case "phase":
				request.Phase++
			case "paused":
				var state colony.ColonyState
				_ = store.LoadJSON("COLONY_STATE.json", &state)
				state.Paused = true
				_ = store.SaveJSON("COLONY_STATE.json", state)
			}
			before := nativeDecisionStoreBytes(t)
			if _, err := admitCodexNativeDecision(request, codexNativeQuestion{QuestionID: "admission", Question: "Which path?"}, codexNativeDecisionHooks{}); err == nil {
				t.Fatal("wrong native question admitted")
			}
			if !reflect.DeepEqual(before, nativeDecisionStoreBytes(t)) {
				t.Fatal("refused question changed state")
			}
		})
	}
}
func TestCodexNativeDecisionFactsOnce(t *testing.T) {
	_, d := nativeDecisionFixture(t)
	request := nativeAnswerForTest(d)
	if _, _, err := answerCodexNativeDecision(request, codexNativeDecisionHooks{}); err != nil {
		t.Fatal(err)
	}
	var signals colony.PheromoneFile
	if err := store.LoadJSON("pheromones.json", &signals); err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, signal := range signals.Signals {
		if bytes.Contains(signal.Content, []byte(d.ID)) {
			found++
		}
	}
	if found != 1 {
		t.Fatalf("first answer emitted %d receipt facts", found)
	}
	var ledger episodeLedgerFile
	if err := store.LoadJSON(episodeLedgerPath, &ledger); err != nil {
		t.Fatal(err)
	}
	interventions := 0
	for _, entry := range ledger.Entries {
		if entry.RecordKind == episodeLedgerRecordKindIntervention {
			interventions++
		}
	}
	if interventions != 1 {
		t.Fatalf("first answer emitted %d interventions", interventions)
	}
	before := nativeDecisionStoreBytes(t)
	if _, _, err := answerCodexNativeDecision(request, codexNativeDecisionHooks{}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, nativeDecisionStoreBytes(t)) {
		t.Fatal("replay emitted another fact")
	}
}

func nativeDecisionCommandForTest(t *testing.T, args ...string) (map[string]any, error) {
	t.Helper()
	command, rest, err := rootCmd.Find(args)
	if err != nil {
		return nil, err
	}
	if command.Name() != args[0] && len(args) < 2 {
		return nil, fmt.Errorf("command missing")
	}
	resetFlags(command)
	defer resetFlags(command)
	if err = command.ParseFlags(rest); err != nil {
		return nil, err
	}
	oldOut, oldErr := stdout, stderr
	var out, errors bytes.Buffer
	stdout, stderr = &out, &errors
	defer func() { stdout, stderr = oldOut, oldErr }()
	if command.RunE == nil {
		return nil, fmt.Errorf("registered operation missing: %v", args)
	}
	if err = command.RunE(command, nil); err != nil {
		return nil, err
	}
	var envelope struct {
		OK     bool           `json:"ok"`
		Result map[string]any `json:"result"`
	}
	if err = json.Unmarshal(out.Bytes(), &envelope); err != nil {
		return nil, fmt.Errorf("command output %s / %s: %w", out.String(), errors.String(), err)
	}
	if !envelope.OK {
		return nil, fmt.Errorf("command refused: %s", out.String())
	}
	return envelope.Result, nil
}
func nativeQuestionCommandForTest(t *testing.T, request codexNativeWorkerRequest, key, question string) map[string]any {
	t.Helper()
	wire := nativeContextWireForTest(t, request)
	if key != "" {
		wire["question"] = map[string]any{"question_id": key, "question": question}
	}
	result, err := nativeDecisionCommandForTest(t, "codex-native-worker", "question", "--request", writeCodexNativeRequestForTest(t, wire))
	if err != nil {
		t.Fatal(err)
	}
	rows, ok := result["decisions"].([]any)
	if !ok || len(rows) != 1 {
		t.Fatalf("missing bound runtime question: %#v", result)
	}
	return rows[0].(map[string]any)
}
func answerNativeViewForTest(t *testing.T, view map[string]any, answer string) {
	t.Helper()
	path, ok := view["answer_request_path"].(string)
	if !ok {
		t.Fatalf("missing exact answer request path: %#v", view)
	}
	if view["answer_command"] != "aether decision-answer --native-request "+shellQuote(path) {
		t.Fatalf("answer command not bound to returned runtime file: %#v", view)
	}
	request, err := loadCodexNativeDecisionAnswerRequest(path)
	if err != nil {
		t.Fatal(err)
	}
	if request.Answer != "" {
		t.Fatal("runtime invented the owner answer")
	}
	request.Answer = answer
	raw, _ := json.Marshal(request)
	if err = os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = nativeDecisionCommandForTest(t, "decision-answer", "--native-request", path); err != nil {
		t.Fatal(err)
	}
}
func TestCodexNativeDecisionDelivered(t *testing.T) {
	_, requests := nativeDecisionWorkersFixture(t, 2)
	question := "Which archive boundary applies to this fixture?"
	view := nativeQuestionCommandForTest(t, requests[0], "harness-live-event", question)
	if view["status"] != "pending" {
		t.Fatalf("unanswered choice not pending: %#v", view)
	}
	answer := "HARNESS_DELIVERED_ONLY_TO_CHILD café 日本語"
	answerNativeViewForTest(t, view, answer)
	before := nativeDecisionStoreBytes(t)
	result, err := nativeDecisionCommandForTest(t, "codex-native-worker", "context", "--request", nativeRequestPath(t, requests[0]))
	if err != nil {
		t.Fatal(err)
	}
	delivery, ok := result["context_delivery"].(map[string]any)
	if !ok {
		t.Fatalf("no actual returned delivery: %#v", result)
	}
	payload := delivery["payload"].(string)
	if strings.Count(payload, answer) != 1 || !strings.Contains(payload, question) || delivery["child_id"] != requests[0].ChildID || delivery["payload_sha256"] != lifecycleDigest([]byte(payload)) {
		t.Fatalf("wrong downstream message bytes: %#v", delivery)
	}
	other, err := nativeDecisionCommandForTest(t, "codex-native-worker", "context", "--request", nativeRequestPath(t, requests[1]))
	if err != nil {
		t.Fatal(err)
	}
	if other["context_status"] != "no_updates" || other["context_delivery"] != nil {
		t.Fatalf("answer crossed child: %#v", other)
	}
	if !reflect.DeepEqual(before, nativeDecisionStoreBytes(t)) {
		t.Fatal("context read acknowledged or mutated answer")
	}
	ack := nativeContextAckForTest(t, requests[0], delivery)
	if _, err = nativeDecisionCommandForTest(t, "codex-native-worker", "observe", "--request", writeCodexNativeRequestForTest(t, ack)); err != nil {
		t.Fatal(err)
	}
	after := nativeDecisionStoreBytes(t)
	if _, err = nativeDecisionCommandForTest(t, "codex-native-worker", "observe", "--request", writeCodexNativeRequestForTest(t, ack)); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(after, nativeDecisionStoreBytes(t)) {
		t.Fatal("ACK replay changed durable facts")
	}
	result, err = nativeDecisionCommandForTest(t, "codex-native-worker", "context", "--request", nativeRequestPath(t, requests[0]))
	if err != nil {
		t.Fatal(err)
	}
	if result["context_status"] != "no_updates" {
		t.Fatal("same answer delivered again")
	}
	view = nativeQuestionCommandForTest(t, requests[0], "harness-live-event", question)
	if view["status"] != "answered" || view["answer_request_path"] != nil {
		t.Fatalf("resolved question reopened: %#v", view)
	}
}
func TestCodexNativeDecisionStaleInterleaving(t *testing.T) {
	for _, operation := range []string{"question", "answer", "context"} {
		t.Run(operation, func(t *testing.T) {
			request, d := nativeDecisionFixture(t)
			change := func() {
				var state colony.ColonyState
				_ = store.LoadJSON("COLONY_STATE.json", &state)
				*state.Goal += " changed after request"
				if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
					t.Fatal(err)
				}
			}
			var before map[string]string
			hook := func() { change(); before = nativeDecisionStoreBytes(t) }
			switch operation {
			case "question":
				_, err := admitCodexNativeDecision(request, codexNativeQuestion{QuestionID: "interleaved", Question: "Late question?"}, codexNativeDecisionHooks{BeforeWrite: hook})
				if err == nil {
					t.Fatal("stale question crossed lock")
				}
			case "answer":
				_, _, err := answerCodexNativeDecision(nativeAnswerForTest(d), codexNativeDecisionHooks{BeforeWrite: hook})
				if err == nil {
					t.Fatal("stale answer crossed lock")
				}
			case "context":
				if _, _, err := answerCodexNativeDecision(nativeAnswerForTest(d), codexNativeDecisionHooks{}); err != nil {
					t.Fatal(err)
				}
				_, err := runCodexNativeWorkerWithHooks("context", nativeRequestPath(t, request), codexNativeWorkerHooks{AfterContextRender: hook})
				if err == nil {
					t.Fatal("stale native answer returned after rendering")
				}
			}
			if !reflect.DeepEqual(before, nativeDecisionStoreBytes(t)) {
				t.Fatal("stale operation added a mutation")
			}
		})
	}
	t.Run("changed-answer-after-render", func(t *testing.T) {
		request, d := nativeDecisionFixture(t)
		if _, _, err := answerCodexNativeDecision(nativeAnswerForTest(d), codexNativeDecisionHooks{}); err != nil {
			t.Fatal(err)
		}
		var before map[string]string
		_, err := runCodexNativeWorkerWithHooks("context", nativeRequestPath(t, request), codexNativeWorkerHooks{AfterContextRender: func() {
			var file PendingDecisionFile
			_ = store.LoadJSON(pendingDecisionsFile, &file)
			file.Decisions[0].Resolution = "changed after rendering"
			_ = store.SaveJSON(pendingDecisionsFile, file)
			before = nativeDecisionStoreBytes(t)
		}})
		if err == nil {
			t.Fatal("changed answer returned under stale payload identity")
		}
		if !reflect.DeepEqual(before, nativeDecisionStoreBytes(t)) {
			t.Fatal("read changed state")
		}
	})
}
func TestCodexNativeDecisionTerminalHandoff(t *testing.T) {
	manifest, requests := nativeDecisionWorkersFixture(t, 1)
	request := requests[0]
	terminal := nativeTerminalRequestForTest(t, request, manifest.Dispatches[0].Caste, "blocked")
	question := "Which terminal handoff boundary needs owner input?"
	terminal.Result.Handoff.OpenDecisions = []string{question}
	if _, err := nativeDecisionCommandForTest(t, "codex-native-worker", "record", "--request", nativeRequestPath(t, terminal)); err != nil {
		t.Fatal(err)
	}
	journal := nativeJournalBytes(t)
	view := nativeQuestionCommandForTest(t, request, "", "")
	if view["status"] != "pending" || view["worker_terminal"] != true {
		t.Fatalf("terminal question not preserved: %#v", view)
	}
	answerNativeViewForTest(t, view, "HARNESS_TERMINAL_ANSWER")
	view = nativeQuestionCommandForTest(t, request, "", "")
	if view["status"] != "answered" || view["worker_terminal"] != true {
		t.Fatal("terminal decision reopened")
	}
	if _, err := nativeDecisionCommandForTest(t, "codex-native-worker", "context", "--request", nativeRequestPath(t, request)); err == nil {
		t.Fatal("terminal child implicitly reopened for answer")
	}
	if !bytes.Equal(journal, nativeJournalBytes(t)) {
		t.Fatal("terminal result changed while answering")
	}
}
func TestCodexNativeDecisionGuideContract(t *testing.T) {
	for _, platform := range []string{"codex", "claude", "opencode"} {
		t.Run(platform, func(t *testing.T) {
			guide, err := buildCommandGuide("build", platform)
			if err != nil {
				t.Fatal(err)
			}
			text := strings.Join(append(append(guide.PreSteps, guide.PostSteps...), guide.DriftGuards...), "\n")
			markers := []string{"codex-native-worker question", "codex-native-worker context", "decision-answer --native-request", "context_delivered"}
			for _, marker := range markers {
				if strings.Contains(text, marker) != (platform == "codex") {
					t.Errorf("%s native guide marker %q isolation failed", platform, marker)
				}
			}
			if platform != "codex" {
				if guide.RunCommand != "AETHER_OUTPUT_MODE=json aether build-finalize <phase> --completion-file <Go-owned completion_path returned by build-completion-stage>" {
					t.Fatal("primary platform finalizer changed")
				}
				wrapper := "generated " + platform + " slash-command wrapper"
				if !strings.Contains(text, wrapper) {
					t.Fatalf("primary wrapper guidance missing: %s", wrapper)
				}
			}
		})
	}
	source, err := os.ReadFile("../.aether/skills/colony/aether-colony-build-cycle/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{"codex-native-worker question", "codex-native-worker context", "decision-answer --native-request", "context_delivered"} {
		if !bytes.Contains(source, []byte(marker)) {
			t.Errorf("canonical support lacks %s", marker)
		}
	}
}

func TestCodexNativeDecisionOrdinaryIDCompatibility(t *testing.T) {
	_, native := nativeDecisionFixture(t)
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatal(err)
	}
	*state.Goal = "Later independent colony"
	session := "later-independent-session"
	state.SessionID = &session
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}
	file := loadPendingDecisionFile()
	ordinary := PendingDecision{ID: "ordinary-current", Type: clarificationDecisionType, Description: native.Description, Source: "discuss", CreatedAt: native.CreatedAt}
	stampPendingDecisionScope(&ordinary, loadCurrentPendingDecisionScope())
	file.Decisions = append(file.Decisions, ordinary)
	if err := store.SaveJSON(pendingDecisionsFile, file); err != nil {
		t.Fatal(err)
	}
	// Ambiguous text-only answers conservatively retain the native-route refusal.
	before := nativeDecisionStoreBytes(t)
	if _, err := recordDecisionAnswer(native.Description, "text-only answer", 1, "worker-handoff"); err == nil {
		t.Fatal("historical native text silently became an unbound answer")
	}
	if !reflect.DeepEqual(before, nativeDecisionStoreBytes(t)) {
		t.Fatal("text refusal mutated the store")
	}
	// A current explicit ordinary ID retains its original primary-platform route.
	if _, err := nativeDecisionCommandForTest(t, "discuss", "--resolve", ordinary.ID, "--answer", "ORDINARY_CURRENT_ANSWER"); err != nil {
		t.Fatal(err)
	}
	file = loadPendingDecisionFile()
	for _, d := range file.Decisions {
		if d.ID == native.ID && !reflect.DeepEqual(d, native) {
			t.Fatal("ordinary answer changed historical native question")
		}
		if d.ID == ordinary.ID && (!d.Resolved || d.Resolution != "ORDINARY_CURRENT_ANSWER") {
			t.Fatal("current ordinary ID not answerable")
		}
	}
}
func TestCodexNativeDecisionFlagCompatibility(t *testing.T) {
	for _, operation := range []string{"add", "resolve", "acknowledge", "age"} {
		t.Run(operation, func(t *testing.T) {
			_, d := nativeDecisionFixture(t)
			if operation == "age" {
				file := loadPendingDecisionFile()
				file.Decisions[0].CreatedAt = "2020-01-01T00:00:00Z"
				d = file.Decisions[0]
				if err := store.SaveJSON(pendingDecisionsFile, file); err != nil {
					t.Fatal(err)
				}
			}
			before := nativeDecisionStoreBytes(t)
			var err error
			switch operation {
			case "add":
				_, err = nativeDecisionCommandForTest(t, "flag-add", "--title", "Ordinary flag alongside native question")
			case "resolve":
				_, err = nativeDecisionCommandForTest(t, "flag-resolve", "--id", d.ID, "--message", "UNBOUND_FLAG_ANSWER")
			case "acknowledge":
				_, err = nativeDecisionCommandForTest(t, "flag-acknowledge", "--id", d.ID)
			case "age":
				_, err = nativeDecisionCommandForTest(t, "flag-auto-resolve", "--max-days", "1")
			}
			if operation == "resolve" || operation == "acknowledge" {
				if err == nil {
					t.Fatal("generic flag route admitted native decision mutation")
				}
				if !reflect.DeepEqual(before, nativeDecisionStoreBytes(t)) {
					t.Fatal("flag refusal mutated native decision")
				}
			} else if err != nil {
				t.Fatal(err)
			}
			var file PendingDecisionFile
			if err := store.LoadJSON(pendingDecisionsFile, &file); err != nil {
				t.Fatal(err)
			}
			found := false
			for _, row := range file.Decisions {
				if row.ID == d.ID {
					found = true
					if !reflect.DeepEqual(d, row) {
						t.Fatalf("ordinary flag operation corrupted native decision: before=%#v after=%#v", d, row)
					}
				}
			}
			if !found {
				t.Fatal("flag operation removed native decision")
			}
		})
	}
}
