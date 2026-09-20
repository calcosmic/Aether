package cmd

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

type nativeCancellationCommandResult struct {
	OK     bool                      `json:"ok"`
	Error  string                    `json:"error"`
	Code   int                       `json:"code"`
	Result codexNativeWorkerResponse `json:"result"`
}

// Exercise registered command parsing and output, rather than a test-authored
// refusal receipt. Every request file is outside the durable fixture inventory.
func nativeCancellationCommandForTest(t *testing.T, operation string, request codexNativeWorkerRequest) (nativeCancellationCommandResult, error) {
	t.Helper()
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	args := []string{"codex-native-worker", operation, "--request", nativeRequestPath(t, request)}
	command, rest, err := rootCmd.Find(args)
	if err != nil || command.Name() != operation || command.RunE == nil {
		t.Fatalf("missing registered native %s operation: %v", operation, err)
	}
	resetFlags(command)
	defer resetFlags(command)
	if err := command.ParseFlags(rest); err != nil {
		t.Fatal(err)
	}
	oldOut, oldErr := stdout, stderr
	oldExit := renderedCommandExitCode.Load()
	var output bytes.Buffer
	stdout, stderr = &output, &output
	defer func() {
		stdout, stderr = oldOut, oldErr
		renderedCommandExitCode.Store(oldExit)
	}()
	err = command.RunE(command, nil)
	var response nativeCancellationCommandResult
	if decodeErr := json.Unmarshal(output.Bytes(), &response); decodeErr != nil {
		t.Fatalf("%s returned malformed command output: %v: %s", operation, decodeErr, output.String())
	}
	return response, err
}

func assertNativeCancellationRefusal(t *testing.T, operation string, request codexNativeWorkerRequest) {
	t.Helper()
	before := nativeDecisionStoreBytes(t)
	response, err := nativeCancellationCommandForTest(t, operation, request)
	if err == nil || response.OK || response.Code == 0 || response.Result.LaunchAllowed || response.Result.Complete || response.Result.Receipt != nil || !strings.Contains(response.Error, "cannot provide requested cancellation guarantee") {
		t.Fatalf("%s did not explicitly refuse cancellation-dependent work: %+v, %v", operation, response, err)
	}
	if !reflect.DeepEqual(before, nativeDecisionStoreBytes(t)) {
		t.Fatalf("%s refusal changed durable inventory, worker state or completion credit", operation)
	}
}

func TestCodexNativeCancellationRequirementReserveRefusal(t *testing.T) {
	_, request := nativeAdmissionFixture(t)
	request.RequireCancellation = true
	assertNativeCancellationRefusal(t, "reserve", request)
	_, saved, ok := loadLatestBuildAttempt(request.Phase)
	if !ok || len(saved.WorkerRuns) != 0 || saved.CompletionPath != "" {
		t.Fatal("cancellation refusal reserved work or granted completion")
	}
	request.RequireCancellation = false
	response, err := nativeCancellationCommandForTest(t, "reserve", request)
	if err != nil || !response.OK || !response.Result.LaunchAllowed || response.Result.Worker == nil {
		t.Fatalf("ordinary reservation became unavailable: %+v, %v", response, err)
	}
}

func TestCodexNativeCancellationRequirementEscalationRefusal(t *testing.T) {
	for _, operation := range []string{"bind", "observe", "record", "stage", "question"} {
		t.Run(operation, func(t *testing.T) {
			var request codexNativeWorkerRequest
			switch operation {
			case "bind":
				_, request = nativeAdmissionFixture(t)
				request = nativeReserveForTest(t, request)
			case "observe":
				_, request = nativeBoundForTest(t)
				request.ObservationStatus = "cancel_requested"
				request.ObservedAt = time.Now().UTC().Format(time.RFC3339Nano)
				request.SourceEventID = "cancellation-request-fixture"
				request.SourceEventSHA256 = strings.Repeat("b", 64)
			case "record", "stage":
				manifest, bound := nativeBoundForTest(t)
				request = nativeTerminalRequestForTest(t, bound, manifest.Dispatches[0].Caste, "completed")
				if operation == "stage" {
					if _, err := runCodexNativeWorker("record", nativeRequestPath(t, request)); err != nil {
						t.Fatal(err)
					}
					request = codexNativeWorkerRequest{SchemaVersion: 1, Phase: bound.Phase, ExecutionBinding: bound.ExecutionBinding}
				}
			case "question":
				_, requests := nativeDecisionWorkersFixture(t, 1)
				request = requests[0]
				request.Question = &codexNativeQuestion{QuestionID: "cancellation-admission-fixture", Question: "Which fixture boundary should this worker keep?"}
			}
			request.RequireCancellation = true
			assertNativeCancellationRefusal(t, operation, request)
			// The same request without the unsupported guarantee remains valid;
			// cancellation requests are observations, not terminal acknowledgement.
			request.RequireCancellation = false
			response, err := nativeCancellationCommandForTest(t, operation, request)
			if err != nil || !response.OK {
				t.Fatalf("ordinary %s request should remain admitted: %+v, %v", operation, response, err)
			}
			if operation == "observe" {
				_, saved, ok := loadLatestBuildAttempt(request.Phase)
				if !ok || len(saved.WorkerRuns) != 1 || codexNativeWorkerIsTerminal(saved.WorkerRuns[0]) || saved.CompletionPath != "" {
					t.Fatal("a cancellation request became terminal acknowledgement or completion credit")
				}
			}
		})
	}
}
