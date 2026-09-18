package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// Synthetic fixture setup only: these protocol receipts never qualify actual
// delivery. Rehash before reservation; an active worker cannot change protocol.
func nativeManifestProtocolForTest(t *testing.T, manifest codexBuildManifest, protocol string) codexBuildManifest {
	t.Helper()
	path, record, ok := loadLatestBuildAttempt(manifest.Phase)
	if !ok || record.PlanManifest == nil || len(record.WorkerRuns) != 0 {
		t.Fatal("protocol fixture requires an unreserved saved attempt")
	}
	if manifest.ContextProtocol == protocol {
		return manifest
	}
	manifest.ContextProtocol = protocol
	digest, err := buildManifestSHA256(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifest.ExecutionBinding.ManifestSHA256 = digest
	record.PlanManifest, record.ManifestSHA256 = &manifest, digest
	if err := store.SaveJSON(path, record); err != nil {
		t.Fatal(err)
	}
	manifestPath, err := canonicalBuildAttemptDataPath(record.Manifest)
	if err != nil {
		t.Fatal(err)
	}
	if manifestPath != "" {
		if err := store.SaveJSON(manifestPath, manifest); err != nil {
			t.Fatal(err)
		}
	}
	return manifest
}

func nativeFetchBoundFixture(t *testing.T) (codexBuildManifest, codexNativeWorkerRequest) {
	t.Helper()
	manifest, request, _ := nativeContextFixture(t)
	manifest = nativeManifestProtocolForTest(t, manifest, codexNativeContextProtocolChildFetch)
	request.ExecutionBinding = *manifest.ExecutionBinding
	request = nativeReserveForTest(t, request)
	if _, err := runCodexNativeWorker("bind", nativeRequestPath(t, request)); err != nil {
		t.Fatal(err)
	}
	return manifest, request
}

func nativeFetchReadForTest(t *testing.T, request codexNativeWorkerRequest, purpose string) codexNativeContextDelivery {
	t.Helper()
	request.ContextPurpose = purpose
	response, err := runCodexNativeContextCommand("context", nativeRequestPath(t, request), "", "", nil)
	if err != nil || response.ContextDelivery == nil || response.ContextStatus != "awaiting_ack" || response.ContextAck != nil {
		t.Fatalf("fetch read: %+v %v", response, err)
	}
	return *response.ContextDelivery
}

func nativeFetchAckForTest(t *testing.T, request codexNativeWorkerRequest, delivery codexNativeContextDelivery) {
	t.Helper()
	request.ContextPurpose = delivery.Purpose
	response, err := runCodexNativeContextCommand("context-ack", nativeRequestPath(t, request), delivery.DeliveryID, delivery.PayloadSHA256, delivery.DecisionIDs)
	if err != nil || response.ContextStatus != "ack_validated" || response.ContextAck == nil || response.ContextDelivery != nil {
		t.Fatalf("separate fetch ACK: %+v %v", response, err)
	}
	want := codexNativeContextAck{SchemaVersion: 1, DeliveryID: delivery.DeliveryID, PayloadSHA256: delivery.PayloadSHA256, ChildID: delivery.ChildID, DecisionIDs: delivery.DecisionIDs}
	if !reflect.DeepEqual(*response.ContextAck, want) {
		t.Fatalf("ACK differs from fetched envelope: %+v", response.ContextAck)
	}
}

func nativeFetchObservationForTest(t *testing.T, request codexNativeWorkerRequest, delivery codexNativeContextDelivery, ordinal int) codexNativeWorkerRequest {
	t.Helper()
	_, record, ok := loadLatestBuildAttempt(request.Phase)
	if !ok {
		t.Fatal("missing fetch fixture")
	}
	bound, err := time.Parse(time.RFC3339Nano, record.WorkerRuns[0].Native.BoundAt)
	if err != nil {
		t.Fatal(err)
	}
	base := bound.Add(time.Duration(ordinal*10+1) * time.Second)
	source := func(kind string, offset time.Duration) codexNativeContextSource {
		prefix := fmt.Sprintf("synthetic-%d-%s-", ordinal, kind)
		ref := func(suffix string) codexNativeContextEventRef {
			return codexNativeContextEventRef{ID: prefix + suffix, SHA256: strings.TrimPrefix(lifecycleDigest([]byte(prefix+suffix)), "sha256:")}
		}
		return codexNativeContextSource{ChildID: request.ChildID, TurnID: "synthetic-turn", CallID: prefix + "call-id", Call: ref("call"), Command: ref("command"), Result: ref("result"), StartedAt: base.Add(offset).UTC().Format(time.RFC3339Nano), CompletedAt: base.Add(offset + time.Second).UTC().Format(time.RFC3339Nano)}
	}
	fetch := codexNativeContextFetch{SchemaVersion: 1, Read: source("read", 0), Ack: source("ack", 2*time.Second)}
	request.ObservationStatus, request.ObservedAt = "context_fetched", fetch.Ack.CompletedAt
	request.SourceEventID, request.SourceEventSHA256 = fetch.Ack.Result.ID, fetch.Ack.Result.SHA256
	request.ContextDelivery, request.ContextFetch = &delivery, &fetch
	return request
}

func nativeFetchObserveForTest(t *testing.T, request codexNativeWorkerRequest, delivery codexNativeContextDelivery, ordinal int) codexNativeWorkerRequest {
	t.Helper()
	observation := nativeFetchObservationForTest(t, request, delivery, ordinal)
	response, err := runCodexNativeWorker("observe", nativeRequestPath(t, observation))
	if err != nil || response.ContextStatus != "delivered" || response.Replay {
		t.Fatalf("observe fetch: %+v %v", response, err)
	}
	return observation
}

func TestCodexNativeChildFetchFactoryAndLegacy(t *testing.T) {
	t.Run("new-native-default", func(t *testing.T) {
		root := setupExternalBuildAttemptTest(t)
		t.Setenv("AETHER_ACTIVE_PLATFORM", "codex")
		result, _, _, _, err := runCodexBuildPlanOnly(root, 1, nil)
		if err != nil {
			t.Fatal(err)
		}
		manifest := result["dispatch_manifest"].(codexBuildManifest)
		if manifest.ContextProtocol != codexNativeContextProtocolChildFetch {
			t.Fatalf("new native manifest protocol: %q", manifest.ContextProtocol)
		}
	})
	t.Run("historical-protocol-stays-empty", func(t *testing.T) {
		manifest, request := nativeContextBoundFixture(t)
		if manifest.ContextProtocol != "" {
			t.Fatal("legacy fixture opted into child fetch")
		}
		if _, err := recordDecisionAnswer("Legacy delta?", "LEGACY_DELIVERY", 1, "test"); err != nil {
			t.Fatal(err)
		}
		response, err := runCodexNativeWorker("context", nativeRequestPath(t, request))
		if err != nil || response.ContextDelivery == nil || response.ContextDelivery.SchemaVersion != 1 || response.ContextStatus != "awaiting_delivery" {
			t.Fatalf("legacy read changed: %+v %v", response, err)
		}
		if _, err := runCodexNativeContextCommand("context-ack", nativeRequestPath(t, request), "id", strings.Repeat("a", 64), nil); err == nil {
			t.Fatal("legacy protocol accepted child ACK")
		}
	})
}

func TestCodexNativeChildFetchReadAckNoMutation(t *testing.T) {
	_, request := nativeFetchBoundFixture(t)
	before := nativeContextStateBytesForTest(t)
	delivery := nativeFetchReadForTest(t, request, "initial")
	_, record, _ := loadLatestBuildAttempt(request.Phase)
	if delivery.SchemaVersion != 2 || delivery.Protocol != codexNativeContextProtocolChildFetch || delivery.Purpose != "initial" || delivery.Payload != record.WorkerRuns[0].Native.Prompt || delivery.PayloadSHA256 != record.WorkerRuns[0].Native.PromptSHA256 || !strings.Contains(delivery.Payload, "café 日本語 é\r\n") {
		t.Fatal("fetch shortened or normalized full runtime-owned prompt")
	}
	nativeFetchAckForTest(t, request, delivery)
	nativeFetchAckForTest(t, request, delivery)
	if !reflect.DeepEqual(before, nativeContextStateBytesForTest(t)) {
		t.Fatal("read or repeated ACK created authority or journal writes")
	}
	observation := nativeFetchObserveForTest(t, request, delivery, 0)
	after := nativeContextStateBytesForTest(t)
	response, err := runCodexNativeWorker("observe", nativeRequestPath(t, observation))
	if err != nil || !response.Replay || !reflect.DeepEqual(after, nativeContextStateBytesForTest(t)) {
		t.Fatalf("exact observed replay rewrote state: %+v %v", response, err)
	}
	changed := observation
	changedFetch := *observation.ContextFetch
	changedFetch.Read.Command.SHA256 = strings.Repeat("e", 64)
	changed.ContextFetch = &changedFetch
	if _, err := runCodexNativeWorker("observe", nativeRequestPath(t, changed)); err == nil || !reflect.DeepEqual(after, nativeContextStateBytesForTest(t)) {
		t.Fatal("saved delivery accepted substituted source evidence")
	}
	for _, field := range []string{"child", "worker", "task", "host", "launch"} {
		other := request
		other.ContextPurpose = "initial"
		switch field {
		case "child":
			other.ChildID += "-other"
		case "worker":
			other.WorkerName += "-other"
		case "task":
			other.TaskID += "-other"
		case "host":
			other.HostSessionID += "-other"
		case "launch":
			other.LaunchID += "-other"
		}
		read, err := runCodexNativeWorker("context", nativeRequestPath(t, other))
		if err == nil || read.ContextDelivery != nil || !reflect.DeepEqual(after, nativeContextStateBytesForTest(t)) {
			t.Fatalf("wrong %s received context", field)
		}
	}
	request.ContextPurpose = "initial"
	if _, err := runCodexNativeContextCommand("context-ack", nativeRequestPath(t, request), delivery.DeliveryID, delivery.PayloadSHA256, delivery.DecisionIDs); err == nil {
		t.Fatal("durably acknowledged envelope accepted a fresh ACK")
	}
	_, record, _ = loadLatestBuildAttempt(request.Phase)
	if record.WorkerRuns[0].Result != nil || record.WorkerRuns[0].Native.ContextReceiptSHA256 != "" || projectCodexNativeWorkerState(record.WorkerRuns[0]).HostStatus != "unavailable" {
		t.Fatal("context observation became lifecycle/terminal credit")
	}
}

func TestCodexNativeChildFetchAckRefusals(t *testing.T) {
	for _, control := range []string{"child", "purpose", "missing-purpose", "delivery", "payload", "decision", "duplicate-decision", "missing-decision", "parent-prefill", "unknown-protocol", "protocol-downgrade", "cancel-pending"} {
		t.Run(control, func(t *testing.T) {
			_, request := nativeFetchBoundFixture(t)
			delivery := nativeFetchReadForTest(t, request, "initial")
			purpose, id, digest, ids := delivery.Purpose, delivery.DeliveryID, delivery.PayloadSHA256, append([]string(nil), delivery.DecisionIDs...)
			switch control {
			case "child":
				request.ChildID += "-other"
			case "purpose":
				purpose = "answers"
			case "missing-purpose":
				purpose = ""
			case "delivery":
				id = strings.Repeat("a", 64)
			case "payload":
				digest = strings.Repeat("a", 64)
			case "decision":
				ids = []string{"unread"}
			case "duplicate-decision":
				ids = append(ids, ids[0])
			case "missing-decision":
				ids = nil
			case "parent-prefill":
				request.ContextDeliveryID = delivery.DeliveryID
			case "unknown-protocol", "protocol-downgrade", "cancel-pending":
				path, record, _ := loadLatestBuildAttempt(request.Phase)
				switch control {
				case "unknown-protocol":
					record.WorkerRuns[0].Native.ContextProtocol = "child-fetch/unknown"
				case "protocol-downgrade":
					record.WorkerRuns[0].Native.ContextProtocol = ""
				case "cancel-pending":
					record.WorkerRuns[0].Native.Observations = append(record.WorkerRuns[0].Native.Observations, codexNativeHostObservation{Status: "cancel_requested"})
				}
				if err := store.SaveJSON(path, record); err != nil {
					t.Fatal(err)
				}
			}
			before := nativeContextStateBytesForTest(t)
			request.ContextPurpose = purpose
			response, err := runCodexNativeContextCommand("context-ack", nativeRequestPath(t, request), id, digest, ids)
			if err == nil || response.ContextAck != nil || !reflect.DeepEqual(before, nativeContextStateBytesForTest(t)) {
				t.Fatalf("%s ACK refusal failed: %+v %v", control, response, err)
			}
		})
	}
}

func TestCodexNativeChildFetchSourceRefusals(t *testing.T) {
	for _, control := range []string{"missing-fetch", "host-send", "read-child", "ack-child", "missing-turn", "reused-call", "reused-event", "bad-hash", "missing-event", "overlap", "pre-bind", "top-source", "payload-change", "wrong-purpose"} {
		t.Run(control, func(t *testing.T) {
			_, request := nativeFetchBoundFixture(t)
			delivery := nativeFetchReadForTest(t, request, "initial")
			r := nativeFetchObservationForTest(t, request, delivery, 0)
			switch control {
			case "missing-fetch":
				r.ContextFetch = nil
			case "host-send":
				r.ContextSend = &codexNativeContextSend{Status: "completed", ChildID: request.ChildID, MessageSHA256: delivery.PayloadSHA256}
			case "read-child":
				r.ContextFetch.Read.ChildID += "-other"
			case "ack-child":
				r.ContextFetch.Ack.ChildID += "-other"
			case "missing-turn":
				r.ContextFetch.Ack.TurnID = ""
			case "reused-call":
				r.ContextFetch.Ack.CallID = r.ContextFetch.Read.CallID
			case "reused-event":
				r.ContextFetch.Ack.Call = r.ContextFetch.Read.Call
			case "bad-hash":
				r.ContextFetch.Read.Command.SHA256 = "hash-only-claim"
			case "missing-event":
				r.ContextFetch.Read.Result.ID = ""
			case "overlap":
				r.ContextFetch.Ack.StartedAt = r.ContextFetch.Read.CompletedAt
			case "pre-bind":
				r.ContextFetch.Read.StartedAt = "2000-01-01T00:00:00Z"
			case "top-source":
				r.SourceEventID += "-substituted"
			case "payload-change":
				r.ContextDelivery.Payload += "normalized"
			case "wrong-purpose":
				r.ContextDelivery.Purpose = "answers"
			}
			before := nativeContextStateBytesForTest(t)
			if _, err := runCodexNativeWorker("observe", nativeRequestPath(t, r)); err == nil || !reflect.DeepEqual(before, nativeContextStateBytesForTest(t)) {
				t.Fatalf("%s source refusal failed: %v", control, err)
			}
		})
	}
}

func TestCodexNativeChildFetchQuestionClosure(t *testing.T) {
	_, manifest := nativeContextLaneManifestForTest(t)
	request := nativeFinalizeReserve(t, manifest)[0]
	terminal := nativeTerminalRequestForTest(t, request, manifest.Dispatches[0].Caste, "completed")
	refuseTerminal := func(stage string) {
		t.Helper()
		before := nativeJournalBytes(t)
		if _, err := runCodexNativeWorker("record", nativeRequestPath(t, terminal)); err == nil || !bytes.Equal(before, nativeJournalBytes(t)) {
			t.Fatalf("%s terminal bypassed missing context: %v", stage, err)
		}
	}
	refuseTerminal("no read")
	initial := nativeFetchReadForTest(t, request, "initial")
	nativeFetchAckForTest(t, request, initial)
	refuseTerminal("ACK without durable observation")
	nativeFetchObserveForTest(t, request, initial, 0)
	question, err := admitCodexNativeDecision(request, codexNativeQuestion{QuestionID: "actual-runtime-question-fixture", Question: "Which exact Unicode bytes?"}, codexNativeDecisionHooks{})
	if err != nil {
		t.Fatal(err)
	}
	refuseTerminal("unresolved scoped question")
	answer := nativeAnswerForTest(question)
	answer.Answer = "Exact café 日本語 é\nsecond line \"quote\""
	if _, _, err := answerCodexNativeDecision(answer, codexNativeDecisionHooks{}); err != nil {
		t.Fatal(err)
	}
	refuseTerminal("resolved answer without delivery")
	delivery := nativeFetchReadForTest(t, request, "answers")
	if !strings.Contains(delivery.Payload, answer.Answer) || !equalCodexNativeContextIDs(delivery.DecisionIDs, []string{question.ID}) {
		t.Fatal("full scoped answer was lost")
	}
	// A newer unrelated ordinary answer does not invalidate the exact fetched subset.
	if _, err := recordDecisionAnswer("Another later answer?", "LATER_UNRELATED", 1, "test"); err != nil {
		t.Fatal(err)
	}
	nativeFetchAckForTest(t, request, delivery)
	refuseTerminal("answer ACK without observation")
	nativeFetchObserveForTest(t, request, delivery, 1)
	var decisions PendingDecisionFile
	if err := store.LoadJSON(pendingDecisionsFile, &decisions); err != nil {
		t.Fatal(err)
	}
	for i := range decisions.Decisions {
		if decisions.Decisions[i].ID == question.ID {
			decisions.Decisions[i].Resolution += "CHANGED"
		}
	}
	if err := store.SaveJSON(pendingDecisionsFile, decisions); err != nil {
		t.Fatal(err)
	}
	refuseTerminal("answer changed after observed ACK")
	for i := range decisions.Decisions {
		if decisions.Decisions[i].ID == question.ID {
			decisions.Decisions[i].Resolution = answer.Answer
		}
	}
	if err := store.SaveJSON(pendingDecisionsFile, decisions); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodexNativeWorker("record", nativeRequestPath(t, terminal)); err != nil {
		t.Fatal(err)
	}
	_, record, _ := loadLatestBuildAttempt(request.Phase)
	if record.WorkerRuns[0].Native.ContextReceiptSHA256 == "" {
		t.Fatal("positive terminal did not pin its context receipt set")
	}
	// Terminal replay uses the frozen receipts, not today's mutable answer/file.
	for i := range decisions.Decisions {
		if decisions.Decisions[i].ID == question.ID {
			decisions.Decisions[i].Resolution += "AFTER_TERMINAL"
		}
	}
	if err := store.SaveJSON(pendingDecisionsFile, decisions); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(request.Workspace, "evidence.txt")); err != nil {
		t.Fatal(err)
	}
	before := nativeJournalBytes(t)
	if _, err := runCodexNativeWorker("record", nativeRequestPath(t, terminal)); err != nil || !bytes.Equal(before, nativeJournalBytes(t)) {
		t.Fatalf("immutable terminal replay changed: %v", err)
	}
	if err := validateCodexNativeSavedWorker(record, manifest.Dispatches[0], record.WorkerRuns[0]); err != nil {
		t.Fatal(err)
	}
	record.WorkerRuns[0].Native.ContextDeliveries[1].Fetch.Ack.Result.SHA256 = strings.Repeat("f", 64)
	if err := validateCodexNativeSavedWorker(record, manifest.Dispatches[0], record.WorkerRuns[0]); err == nil {
		t.Fatal("saved terminal accepted corrupted context sources")
	}
}

func TestCodexNativeChildFetchReadCurrencyAndBounds(t *testing.T) {
	for _, operation := range []string{"context", "context-ack"} {
		t.Run(operation, func(t *testing.T) {
			_, request := nativeFetchBoundFixture(t)
			delivery := nativeFetchReadForTest(t, request, "initial")
			request.ContextPurpose = "initial"
			if operation == "context-ack" {
				request.ContextDeliveryID, request.ContextPayloadSHA256, request.ContextDecisionIDs = delivery.DeliveryID, delivery.PayloadSHA256, delivery.DecisionIDs
			}
			path := nativeRequestPath(t, request)
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			exact := append(append([]byte(nil), raw...), bytes.Repeat([]byte(" "), internalWorkerRequestMaxBytes-len(raw))...)
			if err := os.WriteFile(path, exact, 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := runCodexNativeWorker(operation, path); err != nil {
				t.Fatalf("exact 2MiB valid request refused: %v", err)
			}
			if err := os.WriteFile(path, append(exact, ' '), 0600); err != nil {
				t.Fatal(err)
			}
			before := nativeContextStateBytesForTest(t)
			if _, err := runCodexNativeWorker(operation, path); err == nil || !reflect.DeepEqual(before, nativeContextStateBytesForTest(t)) {
				t.Fatal("oversized context request admitted or mutated")
			}
			path = nativeRequestPath(t, request)
			_, err = runCodexNativeWorkerWithHooks(operation, path, codexNativeWorkerHooks{AfterContextRender: func() {
				var state colony.ColonyState
				if e := store.LoadJSON("COLONY_STATE.json", &state); e != nil {
					t.Fatal(e)
				}
				state.Paused = true
				if e := store.SaveJSON("COLONY_STATE.json", state); e != nil {
					t.Fatal(e)
				}
			}})
			if err == nil {
				t.Fatal("context returned after currency changed during render")
			}
		})
	}
}

func TestCodexNativeChildFetchFailureAndDelayedObservation(t *testing.T) {
	t.Run("successive-answer-source-order", func(t *testing.T) {
		_, request := nativeFetchBoundFixture(t)
		initial := nativeFetchReadForTest(t, request, "initial")
		nativeFetchObserveForTest(t, request, initial, 0)
		if _, err := recordDecisionAnswer("First answer?", "FIRST_ORDERED_ANSWER", 1, "test"); err != nil {
			t.Fatal(err)
		}
		first := nativeFetchReadForTest(t, request, "answers")
		nativeFetchAckForTest(t, request, first)
		nativeFetchObserveForTest(t, request, first, 2)
		if _, err := recordDecisionAnswer("Next answer?", "SECOND_ORDERED_ANSWER", 1, "test"); err != nil {
			t.Fatal(err)
		}
		next := nativeFetchReadForTest(t, request, "answers")
		nativeFetchAckForTest(t, request, next)
		outOfOrder := nativeFetchObservationForTest(t, request, next, 1)
		before := nativeJournalBytes(t)
		if _, err := runCodexNativeWorker("observe", nativeRequestPath(t, outOfOrder)); err == nil || !bytes.Equal(before, nativeJournalBytes(t)) {
			t.Fatal("new answer read preceded the prior answer ACK")
		}
		nativeFetchObserveForTest(t, request, next, 3)
	})
	t.Run("failure-evidence-remains-recordable", func(t *testing.T) {
		manifest, request := nativeFetchBoundFixture(t)
		terminal := nativeTerminalRequestForTest(t, request, manifest.Dispatches[0].Caste, "failed")
		before := nativeJournalBytes(t)
		if _, err := runCodexNativeWorker("record", nativeRequestPath(t, terminal)); err == nil || !bytes.Equal(before, nativeJournalBytes(t)) {
			t.Fatal("failed envelope laundered positive task credit without context")
		}
		terminal.Result.TaskReceipts = nil
		if _, err := runCodexNativeWorker("record", nativeRequestPath(t, terminal)); err != nil {
			t.Fatalf("honest failure evidence refused: %v", err)
		}
	})
	t.Run("parent-observation-does-not-gate-child-work", func(t *testing.T) {
		_, request := nativeFetchBoundFixture(t)
		delivery := nativeFetchReadForTest(t, request, "initial")
		observation := nativeFetchObservationForTest(t, request, delivery, 0)
		at, _ := time.Parse(time.RFC3339Nano, observation.ObservedAt)
		running := request
		running.ObservationStatus = "running"
		running.ObservedAt = at.Add(time.Second).Format(time.RFC3339Nano)
		running.SourceEventID = "later-running"
		running.SourceEventSHA256 = strings.Repeat("b", 64)
		if _, err := runCodexNativeWorker("observe", nativeRequestPath(t, running)); err != nil {
			t.Fatal(err)
		}
		if _, err := runCodexNativeWorker("observe", nativeRequestPath(t, observation)); err != nil {
			t.Fatalf("delayed parent observation refused: %v", err)
		}
		_, record, _ := loadLatestBuildAttempt(request.Phase)
		if projectCodexNativeWorkerState(record.WorkerRuns[0]).HostStatus != "running" {
			t.Fatal("context observation replaced actual lifecycle status")
		}
		running.SourceEventID = "out-of-order-running"
		running.ObservedAt = at.Add(500 * time.Millisecond).Format(time.RFC3339Nano)
		before := nativeJournalBytes(t)
		if _, err := runCodexNativeWorker("observe", nativeRequestPath(t, running)); err == nil || !bytes.Equal(before, nativeJournalBytes(t)) {
			t.Fatal("delayed fetch receipt weakened lifecycle chronology")
		}
	})
}

func TestCodexNativeChildFetchFieldsCannotBypassQuestionAdmission(t *testing.T) {
	_, request := nativeFetchBoundFixture(t)
	request.Question = &codexNativeQuestion{QuestionID: "no-context-evidence", Question: "Which detail?"}
	request.ContextPurpose = "initial"
	before := nativeContextStateBytesForTest(t)
	if _, err := runCodexNativeWorker("question", nativeRequestPath(t, request)); err == nil || !reflect.DeepEqual(before, nativeContextStateBytesForTest(t)) {
		t.Fatal("question route accepted child-fetch-only fields")
	}
	// Unknown fields remain rejected by the shared strict request reader.
	raw, _ := json.Marshal(request)
	var wire map[string]any
	_ = json.Unmarshal(raw, &wire)
	delete(wire, "context_purpose")
	wire["context_acked"] = true
	if _, err := runCodexNativeWorker("question", writeCodexNativeRequestForTest(t, wire)); err == nil {
		t.Fatal("question accepted fabricated context acknowledgement field")
	}
}
