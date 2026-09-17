package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// This consumer captures exactly the serialized payload the native host reads.
// It deliberately never calls the prompt composer or reconstructs its sections.
func nativeLaunchPayloadForTest(t *testing.T, request codexNativeWorkerRequest) (string, codexNativeWorkerResponse) {
	t.Helper()
	response, err := runCodexNativeWorker("reserve", nativeRequestPath(t, request))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	var downstream codexNativeWorkerResponse
	if err := json.Unmarshal(raw, &downstream); err != nil {
		t.Fatal(err)
	}
	if !downstream.LaunchAllowed || downstream.Worker == nil {
		t.Fatal("host received no launch payload")
	}
	return downstream.Worker.Native.Prompt, downstream
}

func nativeContextFixture(t *testing.T) (codexBuildManifest, codexNativeWorkerRequest, string) {
	t.Helper()
	root := setupExternalBuildAttemptTest(t)
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatal(err)
	}
	*state.Goal += " CAPSULE_NATIVE_MARKER"
	state.Plan.Phases[0].Description = "BRIEF_NATIVE_MARKER\n exact \"quote\" café 日本語 é\r\nEnd"
	state.Memory.Decisions = []colony.Decision{{ID: "context-marker", Phase: 1, Claim: "CAPSULE_NATIVE_MARKER", Rationale: "Carry precise worker context", Timestamp: time.Now().UTC().Format(time.RFC3339)}}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}
	now, old := time.Now().UTC().Format(time.RFC3339), time.Now().Add(-48*time.Hour).UTC().Format(time.RFC3339)
	strength, quarantined := 1.0, true
	signals := []colony.PheromoneSignal{}
	for _, name := range []string{"FOCUS_NATIVE_MARKER", "REDIRECT_NATIVE_MARKER", "EXPIRED_NATIVE_MARKER", "REVOKED_NATIVE_MARKER", "QUARANTINED_NATIVE_MARKER"} {
		raw, _ := json.Marshal(map[string]string{"text": name})
		signal := colony.PheromoneSignal{ID: name, Type: "FOCUS", Priority: "normal", Source: "user", CreatedAt: now, Active: true, Strength: &strength, Content: raw}
		switch name {
		case "REDIRECT_NATIVE_MARKER":
			signal.Type, signal.Priority = "REDIRECT", "high"
		case "EXPIRED_NATIVE_MARKER":
			signal.ExpiresAt = &old
		case "REVOKED_NATIVE_MARKER":
			signal.RevokedAt = &now
		case "QUARANTINED_NATIVE_MARKER":
			signal.Quarantined = &quarantined
		}
		signals = append(signals, signal)
	}
	if err := store.SaveJSON("pheromones.json", colony.PheromoneFile{Signals: signals}); err != nil {
		t.Fatal(err)
	}
	hub := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", hub)
	skill := filepath.Join(hub, "system", "skills", "colony", "native-context")
	if err := os.MkdirAll(skill, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte("---\nname: native-context\ntype: colony\nagent_roles:\n  - builder\n---\nSKILL_NATIVE_MARKER\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := persistDispatchWorkerHandoff(codex.WorkerDispatch{WorkerName: "Prior", Caste: "builder", TaskID: "0.1", Workflow: "build", Phase: 1, Root: root}, codex.DispatchResult{WorkerName: "Prior", Status: "completed", WorkerResult: &codex.WorkerResult{WorkerName: "Prior", Caste: "builder", TaskID: "0.1", Status: "completed", Summary: "Prior concrete work", Handoff: codex.WorkerHandoff{VerificationStatus: "pass", NextWorkerInstructions: []string{"HANDOFF_NATIVE_MARKER"}, Freshness: now}}}); err != nil {
		t.Fatal(err)
	}
	answer, err := recordDecisionAnswer("Which detail?", "ANSWER_NATIVE_MARKER", 1, "native-context-test")
	if err != nil {
		t.Fatal(err)
	}
	manifest := prepareBoundBuildManifestOnly(t, root)
	canonicalRoot, _ := filepath.EvalSymlinks(root)
	d := manifest.Dispatches[0]
	return manifest, codexNativeWorkerRequest{SchemaVersion: 1, Phase: 1, ExecutionBinding: *manifest.ExecutionBinding, WorkerName: d.Name, TaskID: normalizedDispatchTaskID(d), HostSessionID: "native-context-host", Workspace: canonicalRoot, HostPermission: "workspace_write"}, answer.ID
}

func TestCodexNativeContextDelivered(t *testing.T) {
	manifest, request, _ := nativeContextFixture(t)
	prompt, response := nativeLaunchPayloadForTest(t, request)
	for _, marker := range []string{"CAPSULE_NATIVE_MARKER", "BRIEF_NATIVE_MARKER", "SKILL_NATIVE_MARKER", "FOCUS_NATIVE_MARKER", "REDIRECT_NATIVE_MARKER", "HANDOFF_NATIVE_MARKER", "- Which detail? => ANSWER_NATIVE_MARKER"} {
		if got := strings.Count(prompt, marker); got != 1 {
			t.Errorf("downstream prompt has %s %d times; want exactly once", marker, got)
		}
	}
	for _, marker := range []string{"EXPIRED_NATIVE_MARKER", "REVOKED_NATIVE_MARKER", "QUARANTINED_NATIVE_MARKER"} {
		if strings.Contains(prompt, marker) {
			t.Errorf("ineffective steering delivered: %s", marker)
		}
	}
	brief, err := os.ReadFile(filepath.Join(manifest.Root, manifest.Dispatches[0].BriefPath))
	if err != nil {
		t.Fatal(err)
	}
	capsuleAt, briefAt, skillAt := strings.Index(prompt, manifest.ContextCapsule), strings.Index(prompt, string(brief)), strings.Index(prompt, manifest.Dispatches[0].SkillSection)
	if capsuleAt < 0 || briefAt <= capsuleAt || skillAt <= briefAt {
		t.Fatal("downstream context sections were dropped or reordered")
	}
	if response.Worker.Native.PromptSHA256 != lifecycleDigest([]byte(prompt)) {
		t.Fatal("launch digest does not hash delivered bytes")
	}
}

func TestCodexNativeContextBriefIdentity(t *testing.T) {
	for _, change := range []string{"changed", "missing", "directory"} {
		t.Run(change, func(t *testing.T) {
			manifest, request := nativeAdmissionFixture(t)
			path := filepath.Join(manifest.Root, manifest.Dispatches[0].BriefPath)
			before := nativeJournalBytes(t)
			switch change {
			case "changed":
				if err := os.WriteFile(path, []byte("substituted brief"), 0600); err != nil {
					t.Fatal(err)
				}
			case "missing":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := runCodexNativeWorker("reserve", nativeRequestPath(t, request)); err == nil {
				t.Fatal("required brief identity mismatch was admitted")
			}
			if !bytes.Equal(before, nativeJournalBytes(t)) {
				t.Fatal("brief refusal changed journal")
			}
		})
	}
}

func TestCodexNativeContextAnswerDedup(t *testing.T) {
	manifest, request, answerID := nativeContextFixture(t)
	newer, err := recordDecisionAnswer("Which next detail?", "NEWER_ANSWER_NATIVE_MARKER", 1, "native-context-test")
	if err != nil {
		t.Fatal(err)
	}
	prompt, response := nativeLaunchPayloadForTest(t, request)
	if strings.Count(prompt, "- Which detail? => ANSWER_NATIVE_MARKER") != 1 || strings.Count(prompt, "NEWER_ANSWER_NATIVE_MARKER") != 1 {
		t.Fatal("capsule answer duplicated or new answer dropped")
	}
	raw, _ := json.Marshal(manifest)
	var metadata map[string]any
	_ = json.Unmarshal(raw, &metadata)
	if !reflect.DeepEqual(metadata["context_decision_ids"], []any{answerID}) {
		t.Fatalf("manifest lacks exact included answer identity: %v", metadata["context_decision_ids"])
	}
	raw, _ = json.Marshal(response.Worker.Native)
	_ = json.Unmarshal(raw, &metadata)
	if !reflect.DeepEqual(metadata["context_decision_ids"], []any{answerID, newer.ID}) {
		t.Fatalf("launch lacks included answer identity: %v", metadata["context_decision_ids"])
	}
	if strings.Index(prompt, "NEWER_ANSWER_NATIVE_MARKER") < strings.Index(prompt, "SKILL_NATIVE_MARKER") {
		t.Fatal("new answer must follow skills")
	}
}

func TestCodexNativeContextBounds(t *testing.T) {
	_, request := nativeAdmissionFixture(t)
	path := nativeRequestPath(t, request)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	exact := append(append([]byte(nil), raw...), bytes.Repeat([]byte(" "), internalWorkerRequestMaxBytes-len(raw))...)
	if err := os.WriteFile(path, exact, 0600); err != nil {
		t.Fatal(err)
	}
	before := nativeJournalBytes(t)
	if _, err := loadCodexNativeWorkerRequest(path); err != nil {
		t.Fatalf("valid exact 2 MiB request refused: %v", err)
	}
	for _, bad := range [][]byte{append(append([]byte(nil), exact...), ' '), raw[:len(raw)-1], append(append([]byte(nil), raw...), []byte("{}")...)} {
		if err := os.WriteFile(path, bad, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := runCodexNativeWorker("reserve", path); err == nil {
			t.Fatal("oversized/partial/extra JSON request admitted")
		}
		if !bytes.Equal(before, nativeJournalBytes(t)) {
			t.Fatal("invalid request wrote state")
		}
	}
	if err := os.WriteFile(path, exact, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodexNativeWorker("reserve", path); err != nil {
		t.Fatalf("exact 2 MiB request not admitted: %v", err)
	}
}

func TestCodexNativeContextExactBytes(t *testing.T) {
	manifest, request, _ := nativeContextFixture(t)
	brief, err := os.ReadFile(filepath.Join(manifest.Root, manifest.Dispatches[0].BriefPath))
	if err != nil {
		t.Fatal(err)
	}
	prompt, response := nativeLaunchPayloadForTest(t, request)
	if !strings.Contains(prompt, string(brief)) || !strings.Contains(prompt, "café 日本語 é\r\n") {
		t.Fatal("launch normalized exact multiline/Unicode/quote brief bytes")
	}
	if lifecycleDigest([]byte(prompt)) != response.Worker.Native.PromptSHA256 {
		t.Fatal("wrong prompt digest")
	}
}

func TestCodexNativeContextOptionalAndFallback(t *testing.T) {
	root := t.TempDir()
	brief := "\n exact \"quoted\" café 日本語 é\r\ntrailing spaces  \n"
	d := codexBuildDispatch{Name: "Fixture", Caste: "builder", TaskID: "1.1", Brief: brief}
	manifest := codexBuildManifest{Root: root}
	// No colony store: optional context and decisions really are absent.
	saveGlobals(t)
	store = nil
	prompt, err := composeCodexNativePrompt(manifest, d, "fixture-launch")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(prompt.Prompt, brief) || prompt.SHA256 != lifecycleDigest([]byte(prompt.Prompt)) {
		t.Fatal("inline fallback bytes changed")
	}
	d.Brief = ""
	if _, err := composeCodexNativePrompt(manifest, d, "fixture-launch"); err == nil {
		t.Fatal("missing required brief admitted")
	}
	d.Brief = string([]byte{0xff})
	if _, err := composeCodexNativePrompt(manifest, d, "fixture-launch"); err == nil {
		t.Fatal("invalid UTF-8 would be normalized on the JSON boundary")
	}
}

func nativeContextWireForTest(t *testing.T, request codexNativeWorkerRequest) map[string]any {
	t.Helper()
	raw, _ := json.Marshal(request)
	var wire map[string]any
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	return wire
}
func nativeContextResponseForTest(t *testing.T, request codexNativeWorkerRequest) map[string]any {
	t.Helper()
	response, err := runCodexNativeWorker("context", nativeRequestPath(t, request))
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(response)
	var wire map[string]any
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	return wire
}
func nativeContextAckForTest(t *testing.T, request codexNativeWorkerRequest, delivery map[string]any) map[string]any {
	t.Helper()
	wire := nativeContextWireForTest(t, request)
	wire["observation_status"] = "context_delivered"
	wire["observed_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	wire["source_event_id"] = "native-send-" + delivery["delivery_id"].(string)
	wire["source_event_sha256"] = lifecycleDigest([]byte(wire["source_event_id"].(string)))
	wire["context_delivery"] = delivery
	wire["context_send"] = map[string]any{"status": "completed", "child_id": request.ChildID, "message_sha256": delivery["payload_sha256"]}
	return wire
}
func nativeContextBoundFixture(t *testing.T) (codexBuildManifest, codexNativeWorkerRequest) {
	t.Helper()
	manifest, request, _ := nativeContextFixture(t)
	request = nativeReserveForTest(t, request)
	if _, err := runCodexNativeWorker("bind", nativeRequestPath(t, request)); err != nil {
		t.Fatal(err)
	}
	return manifest, request
}

func TestCodexNativeContextDelta(t *testing.T) {
	_, request := nativeContextBoundFixture(t)
	_, beforeWorker, _ := loadLatestBuildAttempt(1)
	originalPrompt := beforeWorker.WorkerRuns[0].Native.Prompt
	answer, err := recordDecisionAnswer("What changed after launch?", "DELTA_NATIVE_MARKER", 1, "native-context-test")
	if err != nil {
		t.Fatal(err)
	}
	before := nativeJournalBytes(t)
	first := nativeContextResponseForTest(t, request)
	delivery, ok := first["context_delivery"].(map[string]any)
	if !ok || first["context_status"] != "awaiting_delivery" {
		t.Fatalf("missing pending delta: %+v", first)
	}
	payload := delivery["payload"].(string)
	if strings.Count(payload, "DELTA_NATIVE_MARKER") != 1 || strings.Contains(payload, "ANSWER_NATIVE_MARKER") || !reflect.DeepEqual(delivery["decision_ids"], []any{answer.ID}) {
		t.Fatalf("wrong delta payload: %+v", delivery)
	}
	if delivery["payload_sha256"] != lifecycleDigest([]byte(payload)) || delivery["child_id"] != request.ChildID {
		t.Fatal("delta digest or target mismatch")
	}
	if !reflect.DeepEqual(first, nativeContextResponseForTest(t, request)) || !bytes.Equal(before, nativeJournalBytes(t)) {
		t.Fatal("render retry changed delivery/state")
	}
	// A newer answer may arrive while the host sends the first envelope.
	newer, err := recordDecisionAnswer("What changed during send?", "LATER_DELTA_MARKER", 1, "native-context-test")
	if err != nil {
		t.Fatal(err)
	}
	ack := nativeContextAckForTest(t, request, delivery)
	path := writeCodexNativeRequestForTest(t, ack)
	accepted, err := runCodexNativeWorker("observe", path)
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Disposition != codexNativeAccepted {
		t.Fatal("send was not acknowledged")
	}
	saved := nativeJournalBytes(t)
	replay, err := runCodexNativeWorker("observe", path)
	if err != nil || !replay.Replay || !reflect.DeepEqual(accepted.Receipt, replay.Receipt) || !bytes.Equal(saved, nativeJournalBytes(t)) {
		t.Fatalf("acknowledgement replay changed facts: %+v %v", replay, err)
	}
	second := nativeContextResponseForTest(t, request)
	next := second["context_delivery"].(map[string]any)
	if !reflect.DeepEqual(next["decision_ids"], []any{newer.ID}) || strings.Contains(next["payload"].(string), "DELTA_NATIVE_MARKER") {
		t.Fatal("acknowledged answer was delivered twice")
	}
	if _, err := runCodexNativeWorker("observe", writeCodexNativeRequestForTest(t, nativeContextAckForTest(t, request, next))); err != nil {
		t.Fatal(err)
	}
	done := nativeContextResponseForTest(t, request)
	if done["context_status"] != "no_updates" || done["context_delivery"] != nil {
		t.Fatalf("acknowledged context still pending: %v", done)
	}
	lookup := nativeContextWireForTest(t, request)
	lookup["context_delivery_id"] = delivery["delivery_id"]
	receipt, err := runCodexNativeWorker("context", writeCodexNativeRequestForTest(t, lookup))
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(receipt)
	var observed map[string]any
	_ = json.Unmarshal(raw, &observed)
	if observed["context_status"] != "delivered" || !reflect.DeepEqual(observed["context_delivery"], delivery) {
		t.Fatal("delivery receipt did not retain exact sent envelope")
	}
	_, record, _ := loadLatestBuildAttempt(1)
	if record.WorkerRuns[0].Native.Prompt != originalPrompt || record.WorkerRuns[0].Native.PromptSHA256 != request.PromptSHA256 || len(record.WorkerRuns[0].Native.ContextDeliveryIDs) != 2 {
		t.Fatal("context changed launch identity or omitted delivery facts")
	}
}

func TestCodexNativeContextDeliveryBinding(t *testing.T) {
	for _, change := range []string{"wrong-child", "wrong-host", "wrong-attempt", "wrong-workspace", "stale-goal", "stale-session", "superseded-attempt", "no-event", "send-pending", "send-failed", "send-wrong-child", "send-wrong-bytes", "payload-substitution", "same-delivery-new-event"} {
		t.Run(change, func(t *testing.T) {
			_, request := nativeContextBoundFixture(t)
			if _, err := recordDecisionAnswer("Which delta?", "BOUND_DELTA", 1, "native-context-test"); err != nil {
				t.Fatal(err)
			}
			delivery := nativeContextResponseForTest(t, request)["context_delivery"].(map[string]any)
			wire := nativeContextAckForTest(t, request, delivery)
			if change == "same-delivery-new-event" {
				if _, err := runCodexNativeWorker("observe", writeCodexNativeRequestForTest(t, wire)); err != nil {
					t.Fatal(err)
				}
			}
			switch change {
			case "wrong-child":
				request.ChildID = "other-child"
				wire["child_id"] = request.ChildID
			case "wrong-host":
				request.HostSessionID = "other-host"
				wire["host_session_id"] = request.HostSessionID
			case "wrong-attempt":
				request.ExecutionBinding.AttemptID = "other-attempt"
				wire["execution_binding"] = request.ExecutionBinding
			case "wrong-workspace":
				request.Workspace = t.TempDir()
				wire["workspace"] = request.Workspace
			case "stale-goal", "stale-session":
				var state colony.ColonyState
				if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
					t.Fatal(err)
				}
				changed := "replacement-context-scope"
				if change == "stale-goal" {
					state.Goal = &changed
				} else {
					state.SessionID = &changed
				}
				if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
					t.Fatal(err)
				}
			case "superseded-attempt":
				var pointer latestBuildAttemptPointer
				if err := store.LoadJSON(latestBuildAttemptPointerPath(1), &pointer); err != nil {
					t.Fatal(err)
				}
				pointer.AttemptID = "new-attempt"
				if err := store.SaveJSON(latestBuildAttemptPointerPath(1), pointer); err != nil {
					t.Fatal(err)
				}
			case "no-event":
				delete(wire, "source_event_id")
			case "send-pending":
				wire["context_send"].(map[string]any)["status"] = "running"
			case "send-failed":
				wire["context_send"].(map[string]any)["status"] = "failed"
			case "send-wrong-child":
				wire["context_send"].(map[string]any)["child_id"] = "other-child"
			case "send-wrong-bytes":
				wire["context_send"].(map[string]any)["message_sha256"] = lifecycleDigest([]byte("wrong"))
			case "payload-substitution":
				delivery["payload"] = "changed under same delivery ID"
			case "same-delivery-new-event":
				wire["source_event_id"] = "another-event"
			}
			var stateBefore map[string][]byte
			stateBefore = nativeContextStateBytesForTest(t)
			if _, err := runCodexNativeWorker("observe", writeCodexNativeRequestForTest(t, wire)); err == nil {
				t.Fatal("mismatched or unconfirmed send was acknowledged")
			}
			if !reflect.DeepEqual(stateBefore, nativeContextStateBytesForTest(t)) {
				t.Fatal("refused acknowledgement wrote state")
			}
			switch change {
			case "wrong-child", "wrong-host", "wrong-attempt", "wrong-workspace", "stale-goal", "stale-session", "superseded-attempt":
				if _, err := runCodexNativeWorker("context", nativeRequestPath(t, request)); err == nil {
					t.Fatal("invalid target received a context delta")
				}
			}
		})
	}
}

func nativeContextStateBytesForTest(t *testing.T) map[string][]byte {
	t.Helper()
	files := map[string][]byte{}
	if err := filepath.WalkDir(store.BasePath(), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[path] = raw
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return files
}
func TestCodexNativeContextReadOnly(t *testing.T) {
	_, request := nativeContextBoundFixture(t)
	if _, err := recordDecisionAnswer("Current instruction?", "READONLY_DELTA", 1, "native-context-test"); err != nil {
		t.Fatal(err)
	}
	before := nativeContextStateBytesForTest(t)
	for i := 0; i < 3; i++ {
		nativeContextResponseForTest(t, request)
	}
	if !reflect.DeepEqual(before, nativeContextStateBytesForTest(t)) {
		t.Fatal("read-only context generation mutated store")
	}
	_, record, _ := loadLatestBuildAttempt(1)
	if len(record.WorkerRuns[0].Native.ContextDeliveryIDs) != 0 {
		t.Fatal("rendering acknowledged an unsent message")
	}
}

func TestCodexNativeContextPacking(t *testing.T) {
	root := setupExternalBuildAttemptTest(t)
	ids := []string{}
	for i := 0; i < clarifiedIntentMaxEntries+3; i++ {
		answer, err := recordDecisionAnswer("Instruction "+strings.Repeat("q", i+1), "PACKED_"+strings.Repeat("a", clarifiedIntentMaxAnswerChars+40), 1, "native-context-test")
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, answer.ID)
	}
	manifest := prepareBoundBuildManifestOnly(t, root)
	if len(manifest.ContextDecisionIDs) != 3 || !reflect.DeepEqual(manifest.ContextDecisionIDs, ids[:3]) {
		t.Fatalf("capsule packing lost exact rendered IDs: %v", manifest.ContextDecisionIDs)
	}
	// The section itself survives the existing protected-intent policy; each
	// long answer was bounded by the existing renderer before its ID was captured.
	rendered := clarifiedIntentPromptRenderResult()
	if !reflect.DeepEqual(rendered.DecisionIDs, manifest.ContextDecisionIDs) {
		t.Fatal("identity came from a different render/budget")
	}
	for _, line := range rendered.Lines {
		if strings.Count(manifest.ContextCapsule, line) != 1 {
			t.Fatal("saved identity is not represented exactly once in capsule")
		}
	}
	dispatch := manifest.Dispatches[0]
	canonical, _ := filepath.EvalSymlinks(root)
	request := codexNativeWorkerRequest{SchemaVersion: 1, Phase: 1, ExecutionBinding: *manifest.ExecutionBinding, WorkerName: dispatch.Name, TaskID: dispatch.TaskID, HostSessionID: "packed-host", Workspace: canonical, HostPermission: "workspace_write"}
	_, response := nativeLaunchPayloadForTest(t, request)
	if !reflect.DeepEqual(response.Worker.Native.ContextDecisionIDs, ids[:6]) {
		t.Fatal("previously packed IDs starved remaining answers at launch")
	}
	request.LaunchID, request.ChildID = response.Worker.ProviderRunID, "packed-child"
	request.DispatchSHA256, request.PromptSHA256 = response.Worker.Native.DispatchSHA256, response.Worker.Native.PromptSHA256
	if _, err := runCodexNativeWorker("bind", nativeRequestPath(t, request)); err != nil {
		t.Fatal(err)
	}
	delivery := nativeContextResponseForTest(t, request)["context_delivery"].(map[string]any)
	expectedIDs := []any{ids[6], ids[7], ids[8]}
	if !reflect.DeepEqual(delivery["decision_ids"], expectedIDs) {
		t.Fatalf("packing did not leave exactly the still-undelivered answers: %v", delivery["decision_ids"])
	}
}

func TestCodexNativeContextConcurrentAck(t *testing.T) {
	_, request := nativeContextBoundFixture(t)
	if _, err := recordDecisionAnswer("Concurrent delivery?", "CONCURRENT_DELTA", 1, "native-context-test"); err != nil {
		t.Fatal(err)
	}
	delivery := nativeContextResponseForTest(t, request)["context_delivery"].(map[string]any)
	path := writeCodexNativeRequestForTest(t, nativeContextAckForTest(t, request, delivery))
	ready, release := make(chan struct{}, 2), make(chan struct{})
	type outcome struct {
		response codexNativeWorkerResponse
		err      error
	}
	results := make(chan outcome, 2)
	for i := 0; i < 2; i++ {
		go func() {
			response, err := runCodexNativeWorkerWithHooks("observe", path, codexNativeWorkerHooks{BeforeWrite: func() { ready <- struct{}{}; <-release }})
			results <- outcome{response, err}
		}()
	}
	<-ready
	<-ready
	close(release)
	accepted, replayed := 0, 0
	for i := 0; i < 2; i++ {
		result := <-results
		if result.err != nil {
			t.Fatal(result.err)
		}
		if result.response.Disposition == codexNativeAccepted {
			accepted++
		}
		if result.response.Disposition == codexNativeReplayed {
			replayed++
		}
	}
	_, record, _ := loadLatestBuildAttempt(1)
	if accepted != 1 || replayed != 1 || len(record.WorkerRuns[0].Native.ContextDeliveryIDs) != 1 {
		t.Fatal("concurrent send acknowledgement duplicated delivery")
	}
}

func TestCodexNativeContextStaleInterleaving(t *testing.T) {
	for _, operation := range []string{"context", "observe"} {
		t.Run(operation, func(t *testing.T) {
			_, request := nativeContextBoundFixture(t)
			if _, err := recordDecisionAnswer("Interleaved instruction?", "INTERLEAVED_DELTA", 1, "native-context-test"); err != nil {
				t.Fatal(err)
			}
			delivery := nativeContextResponseForTest(t, request)["context_delivery"].(map[string]any)
			path := nativeRequestPath(t, request)
			hooks := codexNativeWorkerHooks{}
			var after map[string][]byte
			change := func() {
				var state colony.ColonyState
				if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
					t.Fatal(err)
				}
				replacement := "changed-during-context"
				state.SessionID = &replacement
				if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
					t.Fatal(err)
				}
				after = nativeContextStateBytesForTest(t)
			}
			if operation == "context" {
				hooks.AfterContextRender = change
			} else {
				hooks.BeforeWrite = change
				path = writeCodexNativeRequestForTest(t, nativeContextAckForTest(t, request, delivery))
			}
			if _, err := runCodexNativeWorkerWithHooks(operation, path, hooks); err == nil {
				t.Fatal("stale interleaving was accepted")
			}
			if !reflect.DeepEqual(after, nativeContextStateBytesForTest(t)) {
				t.Fatal("refused interleaving wrote state")
			}
		})
	}
}

func TestCodexNativeRecoveryPhaseRoute(t *testing.T) {
	_, request := nativeContextBoundFixture(t)
	before := nativeContextStateBytesForTest(t)
	command, _, err := rootCmd.Find([]string{"codex-native-worker", "inspect"})
	if err != nil || command.Name() != "inspect" || command.Flags().Lookup("phase") == nil {
		t.Fatal("registered inspect --phase route missing")
	}
	if err := command.Flags().Set("phase", "1"); err != nil {
		t.Fatal(err)
	}
	var runErr error
	var buffer bytes.Buffer
	original := stdout
	stdout = &buffer
	t.Cleanup(func() { stdout = original })
	runErr = command.RunE(command, nil)
	raw := buffer.String()
	if runErr != nil || !strings.Contains(raw, request.ChildID) {
		t.Fatalf("phase inspection did not return bound child: %v %s", runErr, raw)
	}
	if !reflect.DeepEqual(before, nativeContextStateBytesForTest(t)) {
		t.Fatal("inspect --phase mutated state")
	}
	if err := command.Flags().Set("request", nativeRequestPath(t, request)); err != nil {
		t.Fatal(err)
	}
	runErr = command.RunE(command, nil)
	if runErr == nil {
		t.Fatal("ambiguous --phase and --request accepted")
	}
	for _, operation := range []string{"inspect", "stage"} {
		if _, err := runCodexNativeWorkerForPhase(operation, 2); err == nil {
			t.Fatal("missing phase accepted")
		}
	}
	if _, err := runCodexNativeWorkerForPhase("stage", 1); err == nil {
		t.Fatal("unfinished native work staged")
	}
	if _, err := runCodexNativeWorkerForPhase("reserve", 1); err == nil {
		t.Fatal("phase-only route allowed a mutation other than staging")
	}
	if !reflect.DeepEqual(before, nativeContextStateBytesForTest(t)) {
		t.Fatal("phase route refusal wrote state")
	}
}
