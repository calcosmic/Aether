package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Copy the completed coordinator observations before classification. The final
// outer capture still retains the same files on every exit, including failures.
// Replay never consults or recreates the removed temporary request directory.
func nativeRetainContextCoordination(t *testing.T, r codexNativeLiveReceipt) {
	t.Helper()
	if r.replaying || r.CoordinationPath == "" {
		return
	}
	if !filepath.IsAbs(r.CoordinationPath) || !strings.HasPrefix(filepath.Base(r.CoordinationPath), "aether-worker-request-") {
		t.Fatal("invalid owned context request directory")
	}
	info, err := os.Lstat(r.CoordinationPath)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("owned context request directory unavailable")
	}
	entries, err := os.ReadDir(r.CoordinationPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			t.Fatal("context coordination capture requires regular files")
		}
		raw, err := os.ReadFile(filepath.Join(r.CoordinationPath, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		liveSkillWrite(t, filepath.Join(filepath.Dir(r.FixtureRoot), "coordination", entry.Name()), raw)
	}
}

// Protocol identity also applies to interruption/refusal captures which do not
// assert that a child completed a context fetch. Legacy stays explicitly empty.
func nativeValidateLiveContextProtocol(r codexNativeLiveReceipt) (buildAttemptRecord, error) {
	var attempt buildAttemptRecord
	if r.ContextProtocol != "" && r.ContextProtocol != codexNativeContextProtocolChildFetch {
		return attempt, fmt.Errorf("unknown context delivery proof protocol")
	}
	raw, err := r.readEvidence(r.AttemptPath)
	if err != nil && r.ContextProtocol == "" {
		return attempt, nil
	}
	if err != nil || json.Unmarshal(raw, &attempt) != nil || attempt.PlanManifest == nil {
		return attempt, fmt.Errorf("context proof requires original accepted attempt")
	}
	if attempt.PlanManifest.ContextProtocol != r.ContextProtocol {
		return attempt, fmt.Errorf("receipt context protocol differs from accepted manifest")
	}
	for _, worker := range attempt.WorkerRuns {
		if worker.Native == nil {
			if r.ContextProtocol != "" {
				return attempt, fmt.Errorf("context proof worker lacks native binding")
			}
			continue
		}
		if worker.Native.ContextProtocol != r.ContextProtocol {
			return attempt, fmt.Errorf("worker context protocol differs from accepted manifest")
		}
	}
	return attempt, nil
}

func nativeValidateLiveContextReceipts(r codexNativeLiveReceipt) error {
	attempt, err := nativeValidateLiveContextProtocol(r)
	if err != nil {
		return err
	}
	if r.ContextProtocol == "" {
		return nil
	}
	if len(attempt.WorkerRuns) == 0 {
		return fmt.Errorf("context proof requires actual bound workers")
	}
	for _, worker := range attempt.WorkerRuns {
		child := r
		matched := worker.Native.ChildID == r.ChildID
		for _, actual := range r.Workers {
			if actual.ChildID == worker.Native.ChildID {
				child, matched = actual, true
				break
			}
		}
		if !matched {
			return fmt.Errorf("context proof has no matching actual child")
		}
		if child.ContextProtocol != r.ContextProtocol {
			return fmt.Errorf("child context protocol differs from parent receipt")
		}
		if err := nativeValidateChildContextReceipts(child, worker); err != nil {
			return err
		}
	}
	return nil
}

type nativeContextPointer struct {
	SchemaVersion   int                      `json:"schema_version"`
	Protocol        string                   `json:"protocol"`
	Purpose         string                   `json:"purpose"`
	ChildID         string                   `json:"child_id"`
	CandidatePath   string                   `json:"candidate_path"`
	CandidateSHA256 string                   `json:"candidate_sha256"`
	RequestPath     string                   `json:"request_path"`
	RequestSHA256   string                   `json:"request_sha256"`
	Binding         codexNativeWorkerRequest `json:"binding"`
}

// Classify only the two read-only operations against an inventoried bound
// pointer. Full successful output and ACK causality are checked independently.
func nativeChildFetchCommandAllowed(r codexNativeLiveReceipt, words []string) bool {
	if r.ContextProtocol != codexNativeContextProtocolChildFetch || len(words) < 5 || words[0] != r.CandidatePath || words[1] != "codex-native-worker" || words[3] != "--request" {
		return false
	}
	if words[2] != "context" && words[2] != "context-ack" {
		return false
	}
	requestPath := words[4]
	name := filepath.Base(requestPath)
	if !filepath.IsAbs(requestPath) || !strings.HasPrefix(filepath.Base(filepath.Dir(requestPath)), "aether-worker-request-") || !regexp.MustCompile(`^(?:w[0-9]+-)?context-(?:initial|answers)-request\.json$`).MatchString(name) {
		return false
	}
	coord := filepath.Join(filepath.Dir(r.FixtureRoot), "coordination")
	pointerRaw, err := r.readEvidence(filepath.Join(coord, strings.TrimSuffix(name, "-request.json")+"-pointer.json"))
	var pointer nativeContextPointer
	if err != nil || json.Unmarshal(pointerRaw, &pointer) != nil {
		return false
	}
	requestRaw, err := r.readEvidence(filepath.Join(coord, name))
	var request codexNativeWorkerRequest
	if err != nil || json.Unmarshal(requestRaw, &request) != nil {
		return false
	}
	a, _ := json.Marshal(pointer.Binding)
	b, _ := json.Marshal(request)
	if pointer.SchemaVersion != 1 || pointer.Protocol != r.ContextProtocol || pointer.RequestPath != requestPath || pointer.RequestSHA256 != lifecycleDigest(requestRaw) || pointer.CandidatePath != r.CandidatePath || pointer.CandidateSHA256 != r.CandidateSHA256 || pointer.ChildID != r.ChildID || !bytes.Equal(a, b) {
		return false
	}
	if request.SchemaVersion != codexNativeWorkerSchemaVersion || request.ExecutionBinding.Validate() != nil || request.ContextPurpose != pointer.Purpose || (pointer.Purpose != "initial" && pointer.Purpose != "answers") || request.ChildID != r.ChildID || request.HostSessionID != r.BoundHostSessionID || request.LaunchID != r.LaunchID || request.WorkerName != r.WorkerName || request.TaskID != r.TaskID || request.ExecutionBinding.AttemptID != r.AttemptID || request.ExecutionBinding.RunID != r.RunID || request.Workspace != r.FixtureRoot || request.PromptSHA256 != r.PromptSHA256 {
		return false
	}
	if request.Result != nil || request.SourceEventID != "" || request.SourceEventSHA256 != "" || request.ContextDeliveryID != "" || request.ContextPayloadSHA256 != "" || len(request.ContextDecisionIDs) != 0 || request.ContextDelivery != nil || request.ContextSend != nil || request.ContextFetch != nil || request.ObservationStatus != "" {
		return false
	}
	if words[2] == "context" {
		return len(words) == 5
	}
	if len(words) < 9 || (len(words)-9)%2 != 0 || words[5] != "--delivery-id" || words[7] != "--payload-sha256" || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(words[6]) || !regexp.MustCompile(`^sha256:[0-9a-f]{64}$`).MatchString(words[8]) {
		return false
	}
	ids := map[string]bool{}
	for i := 9; i < len(words); i += 2 {
		if words[i] != "--decision-id" || words[i+1] == "" || ids[words[i+1]] {
			return false
		}
		ids[words[i+1]] = true
	}
	return true
}

func TestCodexNativeChildFetchIntegration(t *testing.T) {
	for _, bad := range []string{"valid", "candidate", "child", "binding", "request-bytes", "prefilled-ack", "extra-command", "unindexed-pointer"} {
		t.Run(bad, func(t *testing.T) {
			root := t.TempDir()
			repo := filepath.Join(root, "repo")
			coord := filepath.Join(root, "coordination")
			if err := os.MkdirAll(coord, 0700); err != nil {
				t.Fatal(err)
			}
			requestPath := filepath.Join(root, "aether-worker-request-fixture", "context-initial-request.json")
			r := codexNativeLiveReceipt{ContextProtocol: codexNativeContextProtocolChildFetch, FixtureRoot: repo, CandidatePath: filepath.Join(root, "aether"), CandidateSHA256: lifecycleDigest([]byte("candidate")), ChildID: "actual-child", BoundHostSessionID: "actual-parent", LaunchID: "launch", WorkerName: "builder", TaskID: "task", AttemptID: "attempt-fixture", RunID: "run-fixture", PromptSHA256: lifecycleDigest([]byte("full prompt"))}
			request := codexNativeWorkerRequest{SchemaVersion: 1, Phase: 1, ContextPurpose: "initial", ChildID: r.ChildID, HostSessionID: r.BoundHostSessionID, LaunchID: r.LaunchID, WorkerName: r.WorkerName, TaskID: r.TaskID, Workspace: repo, PromptSHA256: r.PromptSHA256}
			request.ExecutionBinding.SchemaVersion = 1
			request.ExecutionBinding.ManifestSHA256 = strings.Repeat("a", 64)
			request.ExecutionBinding.WorkspaceFingerprint = strings.Repeat("b", 64)
			request.ExecutionBinding.ExecutionOwner = "host-queen"
			request.ExecutionBinding.AttemptID = r.AttemptID
			request.ExecutionBinding.RunID = r.RunID
			if bad == "child" {
				request.ChildID = "sibling"
			}
			if bad == "binding" {
				request.ExecutionBinding.AttemptID = "stale-attempt"
			}
			if bad == "prefilled-ack" {
				request.ContextPayloadSHA256 = lifecycleDigest([]byte("preselected"))
			}
			raw, _ := json.Marshal(request)
			pointer := nativeContextPointer{SchemaVersion: 1, Protocol: r.ContextProtocol, Purpose: "initial", ChildID: r.ChildID, CandidatePath: r.CandidatePath, CandidateSHA256: r.CandidateSHA256, RequestPath: requestPath, RequestSHA256: lifecycleDigest(raw), Binding: request}
			pointerRaw, _ := json.Marshal(pointer)
			retainedRequest := filepath.Join(coord, filepath.Base(requestPath))
			retainedPointer := filepath.Join(coord, "context-initial-pointer.json")
			if err := os.WriteFile(retainedRequest, raw, 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(retainedPointer, pointerRaw, 0600); err != nil {
				t.Fatal(err)
			}
			r.replaying = true
			r.replayArtifacts = map[string]string{retainedRequest: lifecycleDigest(raw), retainedPointer: lifecycleDigest(pointerRaw)}
			if bad == "request-bytes" {
				if err := os.WriteFile(retainedRequest, append(raw, ' '), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if bad == "unindexed-pointer" {
				delete(r.replayArtifacts, retainedPointer)
			}
			read := []string{r.CandidatePath, "codex-native-worker", "context", "--request", requestPath}
			if bad == "candidate" {
				read[0] = filepath.Join(root, "other-aether")
			}
			if bad == "extra-command" {
				read = append(read, ";", "echo", "forged")
			}
			if got := nativeChildFetchCommandAllowed(r, read); got != (bad == "valid") {
				t.Fatalf("read classification=%v for %s", got, bad)
			}
			if bad == "valid" {
				ack := append([]string{}, read...)
				ack[2] = "context-ack"
				ack = append(ack, "--delivery-id", strings.TrimPrefix(lifecycleDigest([]byte("delivery")), "sha256:"), "--payload-sha256", lifecycleDigest([]byte("payload")))
				if !nativeChildFetchCommandAllowed(r, ack) {
					t.Fatal("bound separate ACK was not classified")
				}
				if nativeChildFetchCommandAllowed(r, append(ack, "--delivery-id", lifecycleDigest([]byte("other")))) {
					t.Fatal("duplicate ACK identity flag admitted")
				}
				if nativeChildFetchCommandAllowed(r, append(ack, "--decision-id", "same", "--decision-id", "same")) {
					t.Fatal("duplicate decision IDs admitted")
				}
			}
		})
	}
}

func TestCodexNativeChildFetchProtocolIdentity(t *testing.T) {
	for _, name := range []string{"new-interrupted", "legacy", "empty-receipt", "unknown", "manifest-downgrade", "worker-downgrade", "legacy-worker-upgrade"} {
		t.Run(name, func(t *testing.T) {
			protocol := codexNativeContextProtocolChildFetch
			r := codexNativeLiveReceipt{ContextProtocol: protocol, AttemptPath: filepath.Join(t.TempDir(), "attempt.json")}
			attempt := buildAttemptRecord{SchemaVersion: 1, ID: "attempt", PlanManifest: &codexBuildManifest{ContextProtocol: protocol}, WorkerRuns: []buildAttemptWorkerRun{{Native: &codexNativeWorkerBinding{SchemaVersion: 1, ContextProtocol: protocol, LaunchState: "reserved"}}}}
			switch name {
			case "legacy", "legacy-worker-upgrade":
				r.ContextProtocol = ""
				attempt.PlanManifest.ContextProtocol = ""
				if name == "legacy" {
					attempt.WorkerRuns[0].Native.ContextProtocol = ""
				}
			case "empty-receipt":
				r.ContextProtocol = ""
			case "unknown":
				r.ContextProtocol = "unknown/v1"
			case "manifest-downgrade":
				attempt.PlanManifest.ContextProtocol = ""
			case "worker-downgrade":
				attempt.WorkerRuns[0].Native.ContextProtocol = ""
			}
			raw, err := json.Marshal(attempt)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(r.AttemptPath, raw, 0600); err != nil {
				t.Fatal(err)
			}
			r.replaying = true
			r.replayArtifacts = map[string]string{r.AttemptPath: lifecycleDigest(raw)}
			_, err = nativeValidateLiveContextProtocol(r)
			wantOK := name == "new-interrupted" || name == "legacy"
			if (err == nil) != wantOK {
				t.Fatalf("identity validation error=%v, wantOK=%v", err, wantOK)
			}
			if name == "new-interrupted" && nativeValidateLiveContextReceipts(r) == nil {
				t.Fatal("reserved worker incorrectly proved completed context")
			}
		})
	}
}
