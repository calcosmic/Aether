package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// These controls exercise the actual production default, without changing it to
// the historical fixture protocol. Synthetic receipts here never qualify live delivery.
func nativeContextLaneManifestForTest(t *testing.T) (string, codexBuildManifest) {
	t.Helper()
	root := setupExternalBuildAttemptTest(t)
	// Native question identity must exist before the runtime binds its manifest.
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatal(err)
	}
	session := "native-context-lane-fixture-session"
	state.SessionID = &session
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AETHER_ACTIVE_PLATFORM", "codex")
	t.Setenv(codexNativeBuildOptInEnv, "1")
	result, _, _, _, err := runCodexBuildPlanOnly(root, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	manifest := result["dispatch_manifest"].(codexBuildManifest)
	if manifest.ContextProtocol != codexNativeContextProtocolChildFetch {
		t.Fatalf("production native protocol = %q", manifest.ContextProtocol)
	}
	return root, manifest
}

func TestCodexNativeChildFetchProviderLane(t *testing.T) {
	for _, protocol := range []string{codexNativeContextProtocolChildFetch, "unknown/v99"} {
		for _, entry := range []string{"adapter", "journal", "relabelled-workflow"} {
			t.Run(protocol+"/"+entry, func(t *testing.T) {
				root, manifest := nativeContextLaneManifestForTest(t)
				if protocol != manifest.ContextProtocol {
					manifest = nativeManifestProtocolForTest(t, manifest, protocol)
				}
				t.Chdir(root)
				dispatch := manifest.Dispatches[0]
				wire := boundBuildWorkerRequest(manifest, dispatch)
				if entry == "relabelled-workflow" {
					wire["workflow"], wire["phase"] = "oracle", 0
				}
				path := writeInternalWorkerRequestForTest(t, wire)
				before := nativeContextStateBytesForTest(t)
				var err error
				if entry != "journal" {
					var response internalWorkerAdapterResponse
					response, err = runInternalWorkerAdapter(context.Background(), path, false, true)
					if response.ProviderRunID != "" || response.Worker != nil {
						t.Fatal("refused native lane acquired a provider identity or result")
					}
				} else {
					var request internalWorkerDispatchRequest
					raw, marshalErr := json.Marshal(boundBuildWorkerRequest(manifest, dispatch))
					if marshalErr != nil {
						t.Fatal(marshalErr)
					}
					if err := json.Unmarshal(raw, &request); err != nil {
						t.Fatal(err)
					}
					_, err = beginBuildAttemptWorkerRun(manifest.Phase, *manifest.ExecutionBinding, request, "must-not-register", codex.PlatformFake)
				}
				if err == nil || !strings.Contains(err.Error(), "requires native workers") {
					t.Fatalf("generic %s admitted saved protocol %q: %v", entry, protocol, err)
				}
				if !reflect.DeepEqual(before, nativeContextStateBytesForTest(t)) {
					t.Fatal("refused generic entry changed durable authority")
				}
			})
		}
	}
	t.Run("historical-provider-remains-supported", func(t *testing.T) {
		root := setupExternalBuildAttemptTest(t)
		manifest := prepareBoundBuildManifestOnly(t, root)
		t.Chdir(root)
		response, err := runInternalWorkerAdapter(context.Background(), writeBoundBuildWorkerRequest(t, manifest, manifest.Dispatches[0]), false, true)
		if err != nil || response.ProviderRunID == "" || response.Worker == nil {
			t.Fatalf("historical generic provider changed: %+v %v", response, err)
		}
		if err := validateInternalWorkerExecutionBinding(root, internalWorkerDispatchRequest{Workflow: "oracle", ExecutionBinding: manifest.ExecutionBinding}); err != nil {
			t.Fatalf("historical non-build binding changed: %v", err)
		}
		if err := validateInternalWorkerExecutionBinding(root, internalWorkerDispatchRequest{Workflow: "oracle"}); err != nil {
			t.Fatalf("unrelated unbound non-build dispatch changed: %v", err)
		}
	})
}

func TestCodexNativeChildFetchCompletionLane(t *testing.T) {
	for _, journal := range []string{"no-workers", "generic-workers"} {
		for _, packetKind := range []string{"bound", "protocol-stripped", "all-binding-stripped"} {
			t.Run(journal+"/"+packetKind, func(t *testing.T) {
				root, manifest := nativeContextLaneManifestForTest(t)
				path, record, ok := loadLatestBuildAttempt(manifest.Phase)
				if !ok {
					t.Fatal("missing saved attempt")
				}
				packet := codexExternalBuildCompletion{DispatchManifest: &manifest}
				for _, dispatch := range manifest.Dispatches {
					packet.Dispatches = append(packet.Dispatches, codexExternalBuildWorkerResult{Stage: dispatch.Stage, Wave: dispatch.Wave, ExecutionWave: normalizedDispatchWave(dispatch), Caste: dispatch.Caste, Name: dispatch.Name, TaskID: normalizedDispatchTaskID(dispatch), Status: "completed", Summary: "packet-only claim"})
					if journal == "generic-workers" {
						// Deliberately constructed historical-shaped rows cannot select
						// a different lane from the accepted child-fetch manifest.
						record.WorkerRuns = append(record.WorkerRuns, buildAttemptWorkerRun{WorkerName: dispatch.Name, TaskID: normalizedDispatchTaskID(dispatch), Caste: dispatch.Caste, ProviderRunID: "generic-" + dispatch.Name, Platform: codex.PlatformFake, Status: buildWorkerCompleted, Result: &internalWorkerResult{Name: dispatch.Name, Caste: dispatch.Caste, TaskID: normalizedDispatchTaskID(dispatch), Status: buildWorkerCompleted, Summary: "generic claim"}})
					}
				}
				if journal == "generic-workers" {
					if err := store.SaveJSON(path, record); err != nil {
						t.Fatal(err)
					}
				}
				if packetKind != "bound" {
					manifest.ContextProtocol = ""
				}
				if packetKind == "all-binding-stripped" {
					manifest.ExecutionBinding, manifest.AttemptID = nil, ""
				}
				before := nativeContextStateBytesForTest(t)
				if err := validateCodexNativeCompletion(record, packet); err == nil || !strings.Contains(err.Error(), "requires the native worker journal") {
					t.Fatalf("saved protocol admitted %s packet through %s: %v", packetKind, journal, err)
				}
				if packetKind != "all-binding-stripped" {
					if _, _, err := stageBuildAttemptCompletion(path, packet); err == nil || !strings.Contains(err.Error(), "requires the native worker journal") {
						t.Fatalf("packet-only staging admitted: %v", err)
					}
				}
				if _, _, _, _, err := runCodexBuildFinalize(root, 1, packet, true); err == nil || !strings.Contains(err.Error(), "requires the native worker journal") {
					t.Fatalf("packet-only finalization admitted: %v", err)
				}
				if !reflect.DeepEqual(before, nativeContextStateBytesForTest(t)) {
					t.Fatal("refused packet staged data or granted task credit")
				}
			})
		}
	}
	t.Run("unknown-saved-protocol", func(t *testing.T) {
		_, manifest := nativeContextLaneManifestForTest(t)
		manifest = nativeManifestProtocolForTest(t, manifest, "unknown/v99")
		_, record, _ := loadLatestBuildAttempt(manifest.Phase)
		if err := validateCodexNativeCompletion(record, codexExternalBuildCompletion{DispatchManifest: &manifest}); err == nil || !strings.Contains(err.Error(), "unsupported") {
			t.Fatalf("unknown protocol used generic completion: %v", err)
		}
	})
}

func TestCodexNativeChildFetchNativeFinalization(t *testing.T) {
	root, manifest := nativeContextLaneManifestForTest(t)
	requests := nativeFinalizeReserve(t, manifest)
	for i, request := range requests {
		terminal := nativeTerminalRequestForTest(t, request, manifest.Dispatches[i].Caste, "completed")
		before := nativeContextStateBytesForTest(t)
		if _, err := runCodexNativeWorker("record", nativeRequestPath(t, terminal)); err == nil || !reflect.DeepEqual(before, nativeContextStateBytesForTest(t)) {
			t.Fatal("positive native completion bypassed initial delivery")
		}
		delivery := nativeFetchReadForTest(t, request, "initial")
		nativeFetchAckForTest(t, request, delivery)
		nativeFetchObserveForTest(t, request, delivery, 0)
		decisionPath := filepath.Join(store.BasePath(), pendingDecisionsFile)
		if _, err := os.Stat(decisionPath); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("fresh no-question fixture unexpectedly has a decisions store: %v", err)
		}
		if err := os.WriteFile(decisionPath, []byte("{corrupt-json"), 0600); err != nil {
			t.Fatal(err)
		}
		corrupt := nativeContextStateBytesForTest(t)
		if _, err := runCodexNativeWorker("record", nativeRequestPath(t, terminal)); err == nil || errors.Is(err, os.ErrNotExist) || !reflect.DeepEqual(corrupt, nativeContextStateBytesForTest(t)) {
			t.Fatalf("corrupt decisions store was treated as empty or mutated: %v", err)
		}
		if err := os.Remove(decisionPath); err != nil {
			t.Fatal(err)
		}
		if _, err := runCodexNativeWorker("record", nativeRequestPath(t, terminal)); err != nil {
			t.Fatal(err)
		}
	}
	path, packet := nativeFinalizeProjection(t)
	if _, _, err := stageBuildAttemptCompletion(path, packet); err != nil {
		t.Fatal(err)
	}
	_, state, _, dispatches, err := runCodexBuildFinalize(root, 1, packet, true)
	if err != nil || state.State != colony.StateBUILT || len(dispatches) != len(requests) {
		t.Fatalf("actual native admission path cannot finalize: state=%s workers=%d error=%v", state.State, len(dispatches), err)
	}
	for _, task := range state.Plan.Phases[0].Tasks {
		if task.Status != colony.TaskCompleted {
			t.Fatalf("native task credit missing: %+v", task)
		}
	}
}

func TestCodexNativeChildFetchAnswerGuidance(t *testing.T) {
	for _, protocol := range []string{codexNativeContextProtocolChildFetch, ""} {
		t.Run("protocol="+protocol, func(t *testing.T) {
			_, manifest := nativeContextLaneManifestForTest(t)
			if protocol == "" {
				manifest = nativeManifestProtocolForTest(t, manifest, "")
			}
			request := nativeFinalizeReserve(t, manifest)[0]
			d, err := admitCodexNativeDecision(request, codexNativeQuestion{QuestionID: "guidance-question", Question: "Which exact fixture bytes?"}, codexNativeDecisionHooks{})
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := answerCodexNativeDecision(nativeAnswerForTest(d), codexNativeDecisionHooks{}); err != nil {
				t.Fatal(err)
			}
			response, err := runCodexNativeWorker("question", nativeRequestPath(t, request))
			if err != nil || len(response.Decisions) != 1 || response.Decisions[0].Status != "answered" || response.Decisions[0].WorkerTerminal {
				t.Fatalf("answered question view: %+v %v", response, err)
			}
			next := response.Decisions[0].NextCommand
			if protocol == "" {
				if next != "aether codex-native-worker context --request <same bound worker request file>" {
					t.Fatalf("legacy answer guidance changed: %q", next)
				}
				return
			}
			for _, required := range []string{"bound child", "context_purpose=answers", "context-ack", "same answers request file", "fetched delivery_id", "fetched payload_sha256", "each fetched decision_id"} {
				if !strings.Contains(next, required) {
					t.Fatalf("new answer guidance lacks %q: %s", required, next)
				}
			}
			// Follow the advertised metadata-only request and separate ACK;
			// answering alone must not manufacture a delivery receipt.
			before := nativeContextStateBytesForTest(t)
			delivery := nativeFetchReadForTest(t, request, "answers")
			if !equalCodexNativeContextIDs(delivery.DecisionIDs, []string{d.ID}) {
				t.Fatal("guidance read did not fetch its scoped answer")
			}
			nativeFetchAckForTest(t, request, delivery)
			if !reflect.DeepEqual(before, nativeContextStateBytesForTest(t)) {
				t.Fatal("guidance read/ACK wrote an observation")
			}
		})
	}
}
