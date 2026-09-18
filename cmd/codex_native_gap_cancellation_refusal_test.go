package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

const nativeGapCancellationRefusalSchema = "aether-native-cancellation-refusal/v1"

type nativeGapCancellationDisposition string

const (
	nativeGapCancellationSupported  nativeGapCancellationDisposition = "supported"
	nativeGapCancellationRefused    nativeGapCancellationDisposition = "refused"
	nativeGapCancellationIncomplete nativeGapCancellationDisposition = "incomplete"
)

// These are harness-owned executions of the actual candidate, never purported
// host tool events. Their original files must be included in the outer capture
// inventory. No caller-supplied "refused" boolean is part of the contract.
type nativeGapCancellationProcess struct {
	Provenance string             `json:"provenance"`
	Candidate  nativeEvidenceFile `json:"candidate"`
	Argv       []string           `json:"argv"`
	Cwd        string             `json:"cwd"`
	StartedAt  string             `json:"started_at"`
	FinishedAt string             `json:"finished_at"`
	ExitCode   *int               `json:"exit_code"`
	Stdout     nativeEvidenceFile `json:"stdout"`
	Stderr     nativeEvidenceFile `json:"stderr"`
}

type nativeGapCancellationGuard struct {
	Kind            string                        `json:"kind"`
	Process         nativeEvidenceFile            `json:"process"`
	RequestPath     string                        `json:"request_path,omitempty"`
	Request         nativeEvidenceFile            `json:"request,omitempty"`
	OriginalRequest nativeEvidenceFile            `json:"original_request,omitempty"`
	Before          map[string]nativeEvidenceFile `json:"before"`
	After           map[string]nativeEvidenceFile `json:"after"`
	BeforeInventory nativeEvidenceFile            `json:"before_inventory"`
	AfterInventory  nativeEvidenceFile            `json:"after_inventory"`
}

type nativeGapCancellationRefusalCapture struct {
	SchemaVersion string                       `json:"schema_version"`
	StartedAt     string                       `json:"started_at"`
	FinishedAt    string                       `json:"finished_at"`
	ParentBefore  nativeEvidenceFile           `json:"parent_before"`
	ParentAfter   nativeEvidenceFile           `json:"parent_after"`
	ChildBefore   nativeEvidenceFile           `json:"child_before"`
	ChildAfter    nativeEvidenceFile           `json:"child_after"`
	Guards        []nativeGapCancellationGuard `json:"guards"`
}

func nativeGapCancellationRead(r codexNativeLiveReceipt, file nativeEvidenceFile) ([]byte, error) {
	if !filepath.IsAbs(file.Path) || file.SHA256 == "" {
		return nil, fmt.Errorf("cancellation evidence path/digest missing")
	}
	raw, err := r.readEvidence(file.Path)
	if err != nil || lifecycleDigest(raw) != file.SHA256 {
		return nil, fmt.Errorf("cancellation evidence absent/changed: %s", file.Path)
	}
	return raw, nil
}

func nativeGapCancellationGuardArgv(candidate, kind string, phase int, request string) []string {
	switch kind {
	case "require-cancellation", "reserve":
		return []string{candidate, "codex-native-worker", "reserve", "--request", request}
	case "stage":
		return []string{candidate, "codex-native-worker", "stage", "--phase", strconv.Itoa(phase)}
	case "pause", "resume":
		return []string{candidate, kind}
	case "redispatch":
		return []string{candidate, "build", strconv.Itoa(phase), "--plan-only", "--force"}
	case "completion":
		return []string{candidate, "build-finalize", strconv.Itoa(phase), "--completion-file", request}
	}
	return nil
}

// Refusal snapshots cover durable runtime authority. Existing child source
// writes are deliberately outside this inventory: refusal is not a no-write
// guarantee, and must not turn continuing child activity into a cancelled fact.
func nativeGapCancellationSnapshot(t *testing.T, root, directory string) map[string]nativeEvidenceFile {
	t.Helper()
	result := map[string]nativeEvidenceFile{}
	data := filepath.Join(root, ".aether", "data")
	err := filepath.WalkDir(data, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink in durable authority: %s", path)
		}
		if entry.IsDir() || strings.HasSuffix(path, ".lock") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(data, path)
		result[path] = nativeGapCancellationWrite(t, filepath.Join(directory, rel), raw)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"HANDOFF.md", "CONTEXT.md"} {
		path := filepath.Join(root, ".aether", name)
		if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
			t.Fatalf("nonregular recovery authority: %s", path)
		}
		raw, err := os.ReadFile(path)
		if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		result[path] = nativeGapCancellationWrite(t, filepath.Join(directory, name+"-presence.json"), mustNativeGapJSON(t, map[string]any{"present": err == nil, "content": string(raw)}))
	}
	return result
}

func nativeGapCancellationWrite(t *testing.T, path string, raw []byte) nativeEvidenceFile {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, writeErr := file.Write(raw)
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		t.Fatalf("preserve cancellation evidence: %v %v", writeErr, closeErr)
	}
	return nativeEvidenceFile{Path: path, SHA256: lifecycleDigest(raw)}
}

// Invoke only after the live harness has independently admitted the actual
// bound child and the resource/candidate freeze. This helper launches no host.
func nativeGapCaptureCancellationRefusalOperation(t *testing.T, r codexNativeLiveReceipt, directory, kind, requestPath string, env []string) nativeGapCancellationGuard {
	t.Helper()
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	guard := nativeGapCancellationGuard{Kind: kind, RequestPath: requestPath}
	guard.Before = nativeGapCancellationSnapshot(t, r.FixtureRoot, filepath.Join(directory, "before"))
	guard.BeforeInventory = nativeGapCancellationWrite(t, filepath.Join(directory, "before-inventory.json"), mustNativeGapJSON(t, guard.Before))
	attempt, err := nativeGapCancellationActiveAttempt(r, guard.Before)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := r.readEvidence(r.CandidatePath)
	if err != nil || lifecycleDigest(candidate) != r.CandidateSHA256 {
		t.Fatal("cancellation guard candidate is not the frozen executable")
	}
	argv := nativeGapCancellationGuardArgv(r.CandidatePath, kind, attempt.Phase, requestPath)
	if len(argv) == 0 {
		t.Fatal("unknown cancellation guard operation")
	}
	if requestPath != "" {
		raw, err := os.ReadFile(requestPath)
		if err != nil {
			t.Fatal(err)
		}
		guard.Request = nativeGapCancellationWrite(t, filepath.Join(directory, "request.json"), raw)
		guard.OriginalRequest = nativeEvidenceFile{Path: requestPath, SHA256: lifecycleDigest(raw)}
	}
	process := nativeGapCancellationProcess{Provenance: "harness-owned-candidate-execution", Candidate: nativeEvidenceFile{Path: r.CandidatePath, SHA256: r.CandidateSHA256}, Argv: argv, Cwd: r.FixtureRoot, StartedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, argv[0], argv[1:]...)
	command.Dir, command.Env = r.FixtureRoot, env
	var out, stderr bytes.Buffer
	command.Stdout, command.Stderr = &out, &stderr
	_ = command.Run()
	process.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if command.ProcessState != nil && ctx.Err() == nil {
		code := command.ProcessState.ExitCode()
		process.ExitCode = &code
	}
	process.Stdout = nativeGapCancellationWrite(t, filepath.Join(directory, "stdout.json"), out.Bytes())
	process.Stderr = nativeGapCancellationWrite(t, filepath.Join(directory, "stderr.json"), stderr.Bytes())
	guard.After = nativeGapCancellationSnapshot(t, r.FixtureRoot, filepath.Join(directory, "after"))
	guard.AfterInventory = nativeGapCancellationWrite(t, filepath.Join(directory, "after-inventory.json"), mustNativeGapJSON(t, guard.After))
	guard.Process = nativeGapCancellationWrite(t, filepath.Join(directory, "process.json"), mustNativeGapJSON(t, process))
	return guard
}

func nativeGapCancellationActiveAttempt(r codexNativeLiveReceipt, files map[string]nativeEvidenceFile) (buildAttemptRecord, error) {
	var attempt buildAttemptRecord
	raw, err := nativeGapCancellationRead(r, files[r.AttemptPath])
	if err != nil || json.Unmarshal(raw, &attempt) != nil || attempt.ID != r.AttemptID || attempt.RunID != r.RunID || attempt.PlanManifest == nil || attempt.PlanManifest.ExecutionBinding == nil || attempt.PlanManifest.ExecutionBinding.Validate() != nil || len(attempt.WorkerRuns) != 1 || len(attempt.PlanManifest.Dispatches) != 1 || attempt.CompletionPath != "" || attempt.CompletionSHA256 != "" || attempt.CompletedAt != "" || len(attempt.CreditedFiles) != 0 {
		return attempt, fmt.Errorf("bound nonterminal attempt or absence of credit unproved")
	}
	b := attempt.PlanManifest.ExecutionBinding
	digest, err := buildManifestSHA256(*attempt.PlanManifest)
	w := attempt.WorkerRuns[0]
	if err != nil || digest != attempt.ManifestSHA256 || b.AttemptID != attempt.ID || b.RunID != attempt.RunID || b.ManifestSHA256 != attempt.ManifestSHA256 || w.Native == nil || w.Native.ChildID != r.ChildID || w.Native.HostSessionID != r.SessionID || w.ProviderRunID != r.LaunchID || w.WorkerName != r.WorkerName || w.TaskID != r.TaskID || w.Result != nil || w.ResultSHA256 != "" || w.CompletedAt != "" || codexNativeWorkerIsTerminal(w) || w.Native.LaunchState != "bound" {
		return attempt, fmt.Errorf("worker/child/launch binding changed or terminal credit present")
	}
	if !nativeSameCwd(attempt.PlanManifest.Root, r.FixtureRoot) || attempt.PlanManifest.Phase != attempt.Phase || !attempt.PlanManifest.PlanOnly || attempt.ExecutionOwner != "host-queen" || b.ExecutionOwner != attempt.ExecutionOwner || b.WorkspaceFingerprint != attempt.WorkspaceSHA256 || (attempt.Status != buildAttemptAwaiting && attempt.Status != buildAttemptDispatching) || validateCodexNativeSavedWorker(attempt, attempt.PlanManifest.Dispatches[0], w) != nil {
		return attempt, fmt.Errorf("saved assignment/prompt/permission/workspace identity changed")
	}
	unknown := false
	for _, observation := range w.Native.Observations {
		if observation.ChildID == r.ChildID && (observation.Status == "cancel_requested" || observation.Status == "unavailable") {
			unknown = true
		}
	}
	var state colony.ColonyState
	raw, err = nativeGapCancellationRead(r, files[filepath.Join(r.FixtureRoot, ".aether", "data", "COLONY_STATE.json")])
	if !unknown || err != nil || json.Unmarshal(raw, &state) != nil || state.Paused || state.State == colony.StateBUILT {
		return attempt, fmt.Errorf("unknown cancellation or unchanged active colony unproved")
	}
	if validateBuildFinalizeStateStillCurrent(state, attempt.Phase) != nil || validateBuildManifestPlanRevision(*attempt.PlanManifest, state, false) != nil {
		return attempt, fmt.Errorf("active phase or accepted plan revision changed")
	}
	found := 0
	for _, phase := range state.Plan.Phases {
		if phase.ID == attempt.Phase {
			for _, task := range phase.Tasks {
				if task.ID != nil && *task.ID == w.TaskID {
					found++
					if task.Status == colony.TaskCompleted {
						return attempt, fmt.Errorf("active native task already received completion credit")
					}
				}
			}
		}
	}
	if found != 1 {
		return attempt, fmt.Errorf("exact active native task missing or duplicated")
	}
	return attempt, nil
}

func nativeGapCancellationGuardCheck(r codexNativeLiveReceipt, guard nativeGapCancellationGuard) error {
	attempt, err := nativeGapCancellationActiveAttempt(r, guard.Before)
	if err != nil {
		return err
	}
	if _, err := nativeGapCancellationActiveAttempt(r, guard.After); err != nil {
		return err
	}
	if len(guard.Before) < 4 || len(guard.Before) != len(guard.After) {
		return fmt.Errorf("durable authority inventory missing or changed")
	}
	for _, item := range []struct {
		ref   nativeEvidenceFile
		files map[string]nativeEvidenceFile
	}{{guard.BeforeInventory, guard.Before}, {guard.AfterInventory, guard.After}} {
		raw, err := nativeGapCancellationRead(r, item.ref)
		var inventory map[string]nativeEvidenceFile
		if err != nil || json.Unmarshal(raw, &inventory) != nil || !reflect.DeepEqual(inventory, item.files) {
			return fmt.Errorf("original durable inventory missing or substituted")
		}
	}
	handoff := filepath.Join(r.FixtureRoot, ".aether", "HANDOFF.md")
	contextPath := filepath.Join(r.FixtureRoot, ".aether", "CONTEXT.md")
	var presence struct {
		Present *bool  `json:"present"`
		Content string `json:"content"`
	}
	raw, err := nativeGapCancellationRead(r, guard.Before[handoff])
	if err != nil || json.Unmarshal(raw, &presence) != nil || presence.Present == nil || *presence.Present || presence.Content != "" {
		return fmt.Errorf("unsafe or unobserved handoff boundary")
	}
	raw, err = nativeGapCancellationRead(r, guard.Before[contextPath])
	presence.Present = nil
	if err != nil || json.Unmarshal(raw, &presence) != nil || presence.Present == nil || (!*presence.Present && presence.Content != "") {
		return fmt.Errorf("unobserved recovery context boundary")
	}
	for path, before := range guard.Before {
		rel, err := filepath.Rel(filepath.Join(r.FixtureRoot, ".aether", "data"), path)
		if path != handoff && path != contextPath && (err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || strings.HasSuffix(path, ".lock")) {
			return fmt.Errorf("inventory escaped durable authority")
		}
		a, ae := nativeGapCancellationRead(r, before)
		b, be := nativeGapCancellationRead(r, guard.After[path])
		if ae != nil || be != nil || !bytes.Equal(a, b) {
			return fmt.Errorf("guard changed durable authority: %s", path)
		}
	}
	var process nativeGapCancellationProcess
	raw, err = nativeGapCancellationRead(r, guard.Process)
	if err != nil || json.Unmarshal(raw, &process) != nil || process.Provenance != "harness-owned-candidate-execution" || process.ExitCode == nil || *process.ExitCode < 0 || process.Candidate.Path != r.CandidatePath || process.Candidate.SHA256 != r.CandidateSHA256 || !nativeSameCwd(process.Cwd, r.FixtureRoot) || !reflect.DeepEqual(process.Argv, nativeGapCancellationGuardArgv(r.CandidatePath, guard.Kind, attempt.Phase, guard.RequestPath)) {
		return fmt.Errorf("actual candidate runtime invocation missing or substituted")
	}
	if _, err := nativeGapCancellationRead(r, process.Candidate); err != nil {
		return err
	}
	start, se := time.Parse(time.RFC3339Nano, process.StartedAt)
	finish, fe := time.Parse(time.RFC3339Nano, process.FinishedAt)
	if se != nil || fe != nil || finish.Before(start) || finish.Sub(start) > 31*time.Second {
		return fmt.Errorf("runtime invocation timed out or has invalid ordering")
	}
	stdout, oe := nativeGapCancellationRead(r, process.Stdout)
	stderr, ee := nativeGapCancellationRead(r, process.Stderr)
	if oe != nil || ee != nil || (len(bytes.TrimSpace(stdout)) > 0 && len(bytes.TrimSpace(stderr)) > 0) {
		return fmt.Errorf("runtime response missing or ambiguous")
	}
	output := append(append([]byte(nil), stdout...), stderr...)
	var response struct {
		OK     *bool  `json:"ok"`
		Error  string `json:"error"`
		Code   *int   `json:"code"`
		Result struct {
			Paused        *bool                  `json:"paused"`
			Resumed       *bool                  `json:"resumed"`
			StateEffect   string                 `json:"state_effect"`
			LaunchAllowed *bool                  `json:"launch_allowed"`
			Replay        *bool                  `json:"replay"`
			Complete      bool                   `json:"complete"`
			Worker        *buildAttemptWorkerRun `json:"worker"`
		} `json:"result"`
	}
	if json.Unmarshal(output, &response) != nil || response.OK == nil || response.Result.Complete || (response.Result.LaunchAllowed != nil && *response.Result.LaunchAllowed) {
		return fmt.Errorf("runtime response is malformed or allows new work/credit")
	}
	denied := *process.ExitCode > 0 && !*response.OK && response.Code != nil && *response.Code > 0
	if guard.RequestPath != "" {
		original, originalErr := nativeGapCancellationRead(r, guard.OriginalRequest)
		copied, copyErr := nativeGapCancellationRead(r, guard.Request)
		if guard.OriginalRequest.Path != guard.RequestPath || originalErr != nil || copyErr != nil || !bytes.Equal(original, copied) {
			return fmt.Errorf("executed request path differs from original retained request bytes")
		}
	}
	w := attempt.WorkerRuns[0]
	switch guard.Kind {
	case "require-cancellation", "reserve":
		raw, err := nativeGapCancellationRead(r, guard.Request)
		var request codexNativeWorkerRequest
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.DisallowUnknownFields()
		if err != nil || !json.Valid(raw) || decoder.Decode(&request) != nil || !filepath.IsAbs(guard.RequestPath) {
			return fmt.Errorf("reserve request missing or targets another assignment")
		}
		expected := codexNativeWorkerRequest{SchemaVersion: 1, Phase: attempt.Phase, ExecutionBinding: *attempt.PlanManifest.ExecutionBinding, WorkerName: w.WorkerName, TaskID: w.TaskID, HostSessionID: request.HostSessionID, Workspace: w.Native.Workspace, HostPermission: "workspace_write", RequireCancellation: guard.Kind == "require-cancellation"}
		if guard.Kind == "require-cancellation" {
			expected.HostSessionID = r.SessionID
		}
		if expected.HostSessionID == "" || !reflect.DeepEqual(request, expected) {
			return fmt.Errorf("reserve request contains unrelated fields or targets another assignment")
		}
		if guard.Kind == "require-cancellation" {
			if !request.RequireCancellation || request.HostSessionID != r.SessionID || !denied || !strings.Contains(response.Error, "cannot provide requested cancellation guarantee") {
				return fmt.Errorf("unsupported cancellation guarantee was not explicitly denied")
			}
		} else if request.RequireCancellation {
			return fmt.Errorf("duplicate reserve substituted a guarantee refusal")
		} else if request.HostSessionID == r.SessionID {
			if *process.ExitCode != 0 || !*response.OK || response.Result.Replay == nil || !*response.Result.Replay || response.Result.LaunchAllowed == nil || *response.Result.LaunchAllowed || response.Result.Worker == nil || !reflect.DeepEqual(*response.Result.Worker, w) {
				return fmt.Errorf("same-host duplicate reserve did not retain the exact worker")
			}
		} else if request.HostSessionID == "" || !denied || !strings.Contains(response.Error, "already reserved by a different execution") {
			return fmt.Errorf("foreign host replacement was not refused")
		}
	case "pause", "resume":
		value := response.Result.Paused
		if guard.Kind == "resume" {
			value = response.Result.Resumed
		}
		if *process.ExitCode != 0 || !*response.OK || value == nil || *value || response.Result.StateEffect != "none" {
			return fmt.Errorf("public recovery did not explicitly retain a no-op boundary")
		}
	case "stage":
		if !denied || !strings.Contains(response.Error, "native stage requires bound terminal results") {
			return fmt.Errorf("incomplete native staging was not refused")
		}
	case "redispatch":
		if !denied || !strings.Contains(response.Error, fmt.Sprintf("phase %d already has workers in flight for build attempt %s", attempt.Phase, attempt.ID)) {
			return fmt.Errorf("public redispatch lacks a bound in-flight refusal")
		}
	case "completion":
		raw, err := nativeGapCancellationRead(r, guard.Request)
		var packet codexExternalBuildCompletion
		if err != nil || json.Unmarshal(raw, &packet) != nil || packet.DispatchManifest == nil || packet.DispatchManifest.ExecutionBinding == nil || *packet.DispatchManifest.ExecutionBinding != *attempt.PlanManifest.ExecutionBinding || !denied || !strings.Contains(response.Error, "native completion requires terminal evidence") {
			return fmt.Errorf("public completion lacks a bound native-terminal refusal")
		}
		manifestDigest, err := buildManifestSHA256(*packet.DispatchManifest)
		if err != nil || manifestDigest != attempt.ManifestSHA256 {
			return fmt.Errorf("completion probe changed the accepted manifest")
		}
		input, err := packet.structuralInput()
		if err != nil || len(validateCompletionPacketStructure(input)) != 0 || len(packet.Dispatches) != 1 || packet.Dispatches[0].Name != w.WorkerName || packet.Dispatches[0].TaskID != w.TaskID || packet.Dispatches[0].Caste != w.Caste || packet.Dispatches[0].Status != "completed" {
			return fmt.Errorf("completion probe is malformed or targets another worker")
		}
	default:
		return fmt.Errorf("unrecognized cancellation guard")
	}
	return nil
}

// Capture uses actual parent/child rollout paths supplied by the existing
// harness, not reconstructed messages or an alternate host launcher.
func nativeGapCaptureCancellationRefusal(t *testing.T, r codexNativeLiveReceipt, directory, parentPath, childPath string, env []string) nativeGapCancellationRefusalCapture {
	t.Helper()
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	capture := nativeGapCancellationRefusalCapture{SchemaVersion: nativeGapCancellationRefusalSchema}
	copyStream := func(path, name string) nativeEvidenceFile {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return nativeGapCancellationWrite(t, filepath.Join(directory, name), raw)
	}
	capture.ParentBefore = copyStream(parentPath, "parent-before.jsonl")
	capture.ChildBefore = copyStream(childPath, "child-before.jsonl")
	capture.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
	raw, err := r.readEvidence(r.AttemptPath)
	var attempt buildAttemptRecord
	if err != nil || json.Unmarshal(raw, &attempt) != nil || attempt.PlanManifest == nil || attempt.PlanManifest.ExecutionBinding == nil || len(attempt.WorkerRuns) != 1 || attempt.WorkerRuns[0].Native == nil {
		t.Fatal("saved bound cancellation assignment unavailable")
	}
	w := attempt.WorkerRuns[0]
	requestDirectory, err := os.MkdirTemp("", "aether-worker-request-cancellation-")
	if err != nil {
		t.Fatal(err)
	}
	// Retain these original argv-addressed files. The caller adds OriginalRequest
	// refs to the outer immutable inventory even when this temp directory lies
	// outside runRoot; replay must never depend on a deleted request file.
	reserve := codexNativeWorkerRequest{SchemaVersion: 1, Phase: attempt.Phase, ExecutionBinding: *attempt.PlanManifest.ExecutionBinding, WorkerName: w.WorkerName, TaskID: w.TaskID, HostSessionID: w.Native.HostSessionID, Workspace: w.Native.Workspace, HostPermission: "workspace_write"}
	paths := map[string]string{}
	for _, kind := range []string{"require-cancellation", "reserve"} {
		value := reserve
		value.RequireCancellation = kind == "require-cancellation"
		paths[kind] = nativeGapCancellationWrite(t, filepath.Join(requestDirectory, kind+".json"), mustNativeGapJSON(t, value)).Path
	}
	// This deliberately unauthorized completion claim is an adversarial input,
	// not purported worker evidence. The actual journal must reject it.
	packet := codexExternalBuildCompletion{DispatchManifest: attempt.PlanManifest, Dispatches: []codexExternalBuildWorkerResult{{Name: w.WorkerName, Caste: w.Caste, TaskID: w.TaskID, Status: "completed", Summary: "HARNESS unsafe completion probe; no child termination is established"}}}
	value, err := packet.structuralInput()
	if err != nil || len(validateCompletionPacketStructure(value)) != 0 {
		t.Fatal("unsafe-completion probe is not structurally valid")
	}
	paths["completion"] = nativeGapCancellationWrite(t, filepath.Join(requestDirectory, "completion.json"), mustNativeGapJSON(t, packet)).Path
	for _, kind := range []string{"require-cancellation", "stage", "pause", "resume", "reserve", "redispatch", "completion"} {
		guard := nativeGapCaptureCancellationRefusalOperation(t, r, filepath.Join(directory, kind), kind, paths[kind], env)
		capture.Guards = append(capture.Guards, guard)
		if err := nativeGapCancellationGuardCheck(r, guard); err != nil {
			// Preserve a partial, explicitly incomplete capture. Do not execute
			// further operations after an unexpected authority mutation.
			t.Logf("cancellation guard %s incomplete: %v", kind, err)
			break
		}
	}
	capture.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
	capture.ParentAfter = copyStream(parentPath, "parent-after.jsonl")
	capture.ChildAfter = copyStream(childPath, "child-after.jsonl")
	nativeGapCancellationWrite(t, filepath.Join(directory, "cancellation-refusal.json"), mustNativeGapJSON(t, capture))
	return capture
}

// These observational shapes are grounded in the retained Plan 18 rollout:
// rollout-2026-09-18T09-44-23-01a0b379-429d-7e43-9774-661e2eb21d74.jsonl
// lines 7/63 (world state), 15 (usage), and 74 (communication metadata).
// Recognizing them grants no cancellation, no-write or usage measurement credit.
func nativeGapCancellationKnownObservation(raw []byte, kind string) bool {
	var record struct {
		Ordinal   *int                       `json:"ordinal"`
		Timestamp string                     `json:"timestamp"`
		Type      string                     `json:"type"`
		Payload   map[string]json.RawMessage `json:"payload"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&record) != nil || record.Ordinal == nil || *record.Ordinal < 0 || record.Type != kind || record.Payload == nil {
		return false
	}
	if _, err := time.Parse(time.RFC3339Nano, record.Timestamp); err != nil {
		return false
	}
	boolean := func(name string) bool {
		var value *bool
		return json.Unmarshal(record.Payload[name], &value) == nil && value != nil
	}
	switch kind {
	case "world_state":
		var state map[string]json.RawMessage
		return len(record.Payload) == 2 && boolean("full") && json.Unmarshal(record.Payload["state"], &state) == nil && state != nil
	case "inter_agent_communication_metadata":
		return len(record.Payload) == 1 && boolean("trigger_turn")
	case "token_usage_record":
		if len(record.Payload) != 8 {
			return false
		}
		for _, name := range []string{"response_id", "root_turn_id", "session_id", "thread_id", "turn_id"} {
			var value *string
			if json.Unmarshal(record.Payload[name], &value) != nil || value == nil {
				return false
			}
		}
		for _, name := range []string{"usage", "thread_token_usage", "turn_token_usage"} {
			var usage map[string]*int64
			if json.Unmarshal(record.Payload[name], &usage) != nil || len(usage) != 6 {
				return false
			}
			for _, key := range []string{"cache_write_input_tokens", "cached_input_tokens", "input_tokens", "output_tokens", "reasoning_output_tokens", "total_tokens"} {
				if value, ok := usage[key]; !ok || value == nil || *value < 0 {
					return false
				}
			}
		}
		return true
	}
	return false
}

func nativeGapCancellationSpawnCheck(r codexNativeLiveReceipt, parent, child []byte) error {
	var events []json.RawMessage
	spawns, activities := 0, 0
	for stream, raw := range [][]byte{parent, child} {
		for _, line := range bytes.Split(bytes.TrimSpace(raw), []byte{'\n'}) {
			var e struct {
				Type    string
				Payload struct {
					Type, Name, Input string
					Item              struct{ Type, Kind string }
				}
			}
			if len(line) == 0 || json.Unmarshal(line, &e) != nil {
				return fmt.Errorf("malformed cancellation host record")
			}
			switch e.Type {
			case "session_meta", "event_msg", "response_item", "turn_context":
			case "world_state", "token_usage_record", "inter_agent_communication_metadata":
				if !nativeGapCancellationKnownObservation(line, e.Type) {
					return fmt.Errorf("malformed observational host record")
				}
			default:
				return fmt.Errorf("unknown cancellation host record type")
			}
			spawn := (e.Payload.Type == "function_call" || e.Payload.Type == "custom_tool_call") && strings.HasSuffix(e.Payload.Name, "spawn_agent")
			activity := e.Type == "event_msg" && e.Payload.Type == "item_completed" && e.Payload.Item.Type == "SubAgentActivity" && e.Payload.Item.Kind == "started"
			if stream == 1 && (spawn || activity || strings.Contains(e.Payload.Input, "spawn_agent")) {
				return fmt.Errorf("unaccounted child recruitment")
			}
			if stream == 0 {
				if spawn {
					spawns++
				}
				if activity {
					activities++
				}
				if e.Payload.Type == "custom_tool_call" && !spawn && strings.Contains(e.Payload.Input, "spawn_agent") {
					return fmt.Errorf("unclassified indirect launch")
				}
				events = append(events, append(json.RawMessage(nil), line...))
			}
		}
	}
	if spawns != 1 || activities != 1 {
		return fmt.Errorf("one original launch not independently established")
	}
	metadata := bytes.SplitN(child, []byte{'\n'}, 2)[0]
	input, err := json.Marshal(map[string]any{"events": events, "metadata": []json.RawMessage{metadata}, "target": r.ChildID, "parent": r.SessionID, "cwd": r.FixtureRoot})
	if err != nil {
		return err
	}
	// Reuse the existing source-grounded call/result/activity/metadata decoder.
	start, end := strings.Index(nativeFixtureCoordinator, "def resolve_native_child("), strings.Index(nativeFixtureCoordinator, "def bound_native_child(")
	if start < 0 || end <= start {
		return fmt.Errorf("existing child identity decoder absent")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "python3", "-c", "import json,sys\n"+nativeFixtureCoordinator[start:end]+"\nprint(json.dumps(resolve_native_child(**json.load(sys.stdin))))")
	command.Stdin = bytes.NewReader(input)
	output, err := command.Output()
	var identity struct {
		Child string `json:"child_id"`
		Call  string `json:"call_id"`
	}
	if err != nil || json.Unmarshal(output, &identity) != nil || identity.Child != r.ChildID || identity.Call != r.ChildSpawnCallID {
		return fmt.Errorf("actual child call/result/activity identity unproved")
	}
	return nil
}

func nativeGapCancellationRefusalFacts(r codexNativeLiveReceipt, capture nativeGapCancellationRefusalCapture) nativeGapCancellationEvidence {
	evidence := nativeGapCancellationEvidence{Parent: r.SessionID, Child: r.ChildID, Disposition: nativeGapCancellationIncomplete, Refusal: &capture, Gaps: []string{}}
	fail := func(err error) nativeGapCancellationEvidence {
		evidence.Gaps = append(evidence.Gaps, err.Error())
		return evidence
	}
	if r.Scenario != "cancellation" || r.ProofContract != nativeCapabilityProofContract || r.ProofAmendmentSHA256 != nativeCapabilityProofAmendmentSHA256 || capture.SchemaVersion != nativeGapCancellationRefusalSchema || r.SessionID == "" || r.ChildID == "" || r.ChildSpawnCallID == "" || r.AttemptID == "" || r.LaunchID == "" || len(capture.Guards) != 7 {
		return fail(fmt.Errorf("amended cancellation identity or exact guard set missing"))
	}
	beforeParent, e1 := nativeGapCancellationRead(r, capture.ParentBefore)
	afterParent, e2 := nativeGapCancellationRead(r, capture.ParentAfter)
	beforeChild, e3 := nativeGapCancellationRead(r, capture.ChildBefore)
	afterChild, e4 := nativeGapCancellationRead(r, capture.ChildAfter)
	if e1 != nil || e2 != nil || e3 != nil || e4 != nil || !bytes.HasPrefix(afterParent, beforeParent) || !bytes.HasPrefix(afterChild, beforeChild) {
		return fail(fmt.Errorf("original cancellation host snapshots missing or substituted"))
	}
	for _, pair := range [][2][]byte{{beforeParent, beforeChild}, {afterParent, afterChild}} {
		if err := nativeGapCancellationSpawnCheck(r, pair[0], pair[1]); err != nil {
			return fail(err)
		}
	}
	order := []string{"require-cancellation", "stage", "pause", "resume", "reserve", "redispatch", "completion"}
	start, startErr := time.Parse(time.RFC3339Nano, capture.StartedAt)
	finish, finishErr := time.Parse(time.RFC3339Nano, capture.FinishedAt)
	if startErr != nil || finishErr != nil || finish.Before(start) || finish.Sub(start) > 5*time.Minute {
		return fail(fmt.Errorf("guard capture interval missing or invalid"))
	}
	previousFinish := start
	var previous map[string]nativeEvidenceFile
	for index, guard := range capture.Guards {
		if guard.Kind != order[index] {
			return fail(fmt.Errorf("required cancellation guard absent, duplicated or reordered"))
		}
		if previous != nil {
			if len(previous) != len(guard.Before) {
				return fail(fmt.Errorf("authority inventory changed between guard calls"))
			}
			for path, file := range previous {
				if file.SHA256 != guard.Before[path].SHA256 {
					return fail(fmt.Errorf("authority changed between guard calls"))
				}
			}
		}
		if err := nativeGapCancellationGuardCheck(r, guard); err != nil {
			return fail(fmt.Errorf("%s: %w", guard.Kind, err))
		}
		var process nativeGapCancellationProcess
		raw, _ := nativeGapCancellationRead(r, guard.Process)
		if json.Unmarshal(raw, &process) != nil {
			return fail(fmt.Errorf("guard process is malformed"))
		}
		processStart, _ := time.Parse(time.RFC3339Nano, process.StartedAt)
		processFinish, _ := time.Parse(time.RFC3339Nano, process.FinishedAt)
		if processStart.Before(previousFinish) || processFinish.After(finish) {
			return fail(fmt.Errorf("guard process lies outside ordered capture interval"))
		}
		previousFinish = processFinish
		previous = guard.After
	}
	// Refused means only the unsupported operation was prevented. Never promote
	// this to successful cancellation, a terminal result or absence of writes.
	evidence.Disposition, evidence.Qualified = nativeGapCancellationRefused, true
	return evidence
}

// Entirely synthetic host/process records exercise the replay checker only.
// The runtime journal comes from normal fixture admission; no host is launched
// and a successful checker result is not a live qualification receipt.
func nativeGapCancellationRefusalFixture(t *testing.T) (codexNativeLiveReceipt, nativeGapCancellationRefusalCapture) {
	t.Helper()
	fixtureRoot := setupExternalBuildAttemptTest(t)
	// The general admission fixture deliberately starts with legacy state whose
	// compatibility fields are normalized only in memory by the runtime. This
	// immutable replay control needs an already-current raw state: persist its
	// normal loaded form before admission, never rewrite an accepted manifest.
	state, err := loadActiveColonyState()
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}
	manifest := prepareBoundBuildManifestOnly(t, fixtureRoot)
	fixtureRoot, err = filepath.EvalSymlinks(fixtureRoot)
	if err != nil {
		t.Fatal(err)
	}
	dispatch := manifest.Dispatches[0]
	request := codexNativeWorkerRequest{SchemaVersion: 1, Phase: 1, ExecutionBinding: *manifest.ExecutionBinding, WorkerName: dispatch.Name, TaskID: normalizedDispatchTaskID(dispatch), HostSessionID: "admission-host", Workspace: fixtureRoot, HostPermission: "workspace_write"}
	request = nativeReserveForTest(t, request)
	if _, err := runCodexNativeWorker("bind", nativeRequestPath(t, request)); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodexNativeWorker("observe", nativeObservationPath(t, request, "cancel_requested", time.Now().UTC())); err != nil {
		t.Fatal(err)
	}
	attemptPath, attempt, ok := loadLatestBuildAttempt(1)
	if !ok {
		t.Fatal("missing actual fixture attempt")
	}
	root := t.TempDir()
	candidate := nativeGapCancellationWrite(t, filepath.Join(root, "synthetic-candidate.txt"), []byte("synthetic validator input; never executed\n"))
	r := codexNativeLiveReceipt{Scenario: "cancellation", ProofContract: nativeCapabilityProofContract, ProofAmendmentSHA256: nativeCapabilityProofAmendmentSHA256, FixtureRoot: request.Workspace, CandidatePath: candidate.Path, CandidateSHA256: candidate.SHA256, SessionID: request.HostSessionID, ChildID: request.ChildID, ChildSpawnCallID: "spawn", AttemptID: attempt.ID, AttemptPath: filepath.Join(store.BasePath(), attemptPath), RunID: attempt.RunID, LaunchID: request.LaunchID, WorkerName: request.WorkerName, TaskID: request.TaskID}
	at := time.Now().UTC()
	event := func(typ string, payload any) []byte {
		return append(mustNativeGapJSONCompact(t, map[string]any{"timestamp": at.Format(time.RFC3339Nano), "type": typ, "payload": payload}), '\n')
	}
	turn := map[string]any{"turn_id": "turn"}
	parent := event("session_meta", map[string]any{"id": r.SessionID, "cwd": r.FixtureRoot})
	parent = append(parent, event("response_item", map[string]any{"type": "function_call", "name": "spawn_agent", "call_id": "spawn", "arguments": `{"task_name":"brick","agent_type":"aether-builder"}`, "internal_chat_message_metadata_passthrough": turn})...)
	parent = append(parent, event("event_msg", map[string]any{"type": "item_completed", "thread_id": r.SessionID, "turn_id": "turn", "item": map[string]any{"type": "SubAgentActivity", "kind": "started", "id": "spawn", "agent_thread_id": r.ChildID, "agent_path": "/root/brick"}})...)
	parent = append(parent, event("response_item", map[string]any{"type": "function_call_output", "call_id": "spawn", "output": `{"task_name":"/root/brick"}`, "internal_chat_message_metadata_passthrough": turn})...)
	child := event("session_meta", map[string]any{"id": r.ChildID, "parent_thread_id": r.SessionID, "cwd": r.FixtureRoot, "agent_path": "/root/brick", "agent_role": "aether-builder"})
	capture := nativeGapCancellationRefusalCapture{SchemaVersion: nativeGapCancellationRefusalSchema, StartedAt: at.Format(time.RFC3339Nano), FinishedAt: at.Add(7 * time.Second).Format(time.RFC3339Nano),
		ParentBefore: nativeGapCancellationWrite(t, filepath.Join(root, "parent-before.jsonl"), parent), ParentAfter: nativeGapCancellationWrite(t, filepath.Join(root, "parent-after.jsonl"), parent),
		ChildBefore: nativeGapCancellationWrite(t, filepath.Join(root, "child-before.jsonl"), child), ChildAfter: nativeGapCancellationWrite(t, filepath.Join(root, "child-after.jsonl"), child)}
	for index, kind := range []string{"require-cancellation", "stage", "pause", "resume", "reserve", "redispatch", "completion"} {
		directory := filepath.Join(root, kind)
		guard := nativeGapCancellationGuard{Kind: kind, Before: nativeGapCancellationSnapshot(t, r.FixtureRoot, filepath.Join(directory, "before")), After: nativeGapCancellationSnapshot(t, r.FixtureRoot, filepath.Join(directory, "after"))}
		guard.BeforeInventory = nativeGapCancellationWrite(t, filepath.Join(directory, "before-inventory.json"), mustNativeGapJSON(t, guard.Before))
		guard.AfterInventory = nativeGapCancellationWrite(t, filepath.Join(directory, "after-inventory.json"), mustNativeGapJSON(t, guard.After))
		code := 1
		message := "native stage requires bound terminal results for every worker"
		output := map[string]any{"ok": false, "code": 1}
		switch kind {
		case "require-cancellation", "reserve":
			value := codexNativeWorkerRequest{SchemaVersion: 1, Phase: attempt.Phase, ExecutionBinding: *attempt.PlanManifest.ExecutionBinding, WorkerName: request.WorkerName, TaskID: request.TaskID, HostSessionID: request.HostSessionID, Workspace: request.Workspace, HostPermission: "workspace_write", RequireCancellation: kind == "require-cancellation"}
			guard.OriginalRequest = nativeGapCancellationWrite(t, filepath.Join(directory, "original-request.json"), mustNativeGapJSON(t, value))
			guard.RequestPath = guard.OriginalRequest.Path
			guard.Request = nativeGapCancellationWrite(t, filepath.Join(directory, "request.json"), mustNativeGapJSON(t, value))
			message = "native host cannot provide requested cancellation guarantee; no launch or state transition authorized"
			if kind == "reserve" {
				code = 0
				output = map[string]any{"ok": true, "result": map[string]any{"launch_allowed": false, "replay": true, "worker": attempt.WorkerRuns[0]}}
			}
		case "pause", "resume":
			code = 0
			field := "paused"
			if kind == "resume" {
				field = "resumed"
			}
			output = map[string]any{"ok": true, "result": map[string]any{field: false, "state_effect": "none"}}
		case "redispatch":
			message = fmt.Sprintf("phase %d already has workers in flight for build attempt %s; finalize or fail that attempt before requesting a fresh plan", attempt.Phase, attempt.ID)
		case "completion":
			packet := codexExternalBuildCompletion{DispatchManifest: attempt.PlanManifest, Dispatches: []codexExternalBuildWorkerResult{{Name: request.WorkerName, Caste: attempt.WorkerRuns[0].Caste, TaskID: request.TaskID, Status: "completed", Summary: "synthetic unauthorized completion probe"}}}
			guard.OriginalRequest = nativeGapCancellationWrite(t, filepath.Join(directory, "original-request.json"), mustNativeGapJSON(t, packet))
			guard.RequestPath = guard.OriginalRequest.Path
			guard.Request = nativeGapCancellationWrite(t, filepath.Join(directory, "request.json"), mustNativeGapJSON(t, packet))
			message = "native completion requires terminal evidence for " + r.WorkerName
		}
		if code != 0 {
			output["error"] = message
		}
		process := nativeGapCancellationProcess{Provenance: "harness-owned-candidate-execution", Candidate: candidate, Cwd: r.FixtureRoot, Argv: nativeGapCancellationGuardArgv(r.CandidatePath, kind, attempt.Phase, guard.RequestPath), ExitCode: &code, StartedAt: at.Add(time.Duration(index) * time.Second).Format(time.RFC3339Nano), FinishedAt: at.Add(time.Duration(index)*time.Second + time.Millisecond).Format(time.RFC3339Nano), Stdout: nativeGapCancellationWrite(t, filepath.Join(directory, "stdout.json"), mustNativeGapJSON(t, output)), Stderr: nativeGapCancellationWrite(t, filepath.Join(directory, "stderr.json"), nil)}
		guard.Process = nativeGapCancellationWrite(t, filepath.Join(directory, "process.json"), mustNativeGapJSON(t, process))
		capture.Guards = append(capture.Guards, guard)
	}
	return r, capture
}

func TestCodexNativeCancellationRefusalEvidence(t *testing.T) {
	r, capture := nativeGapCancellationRefusalFixture(t)
	valid := nativeGapCancellationRefusalFacts(r, capture)
	if !valid.Qualified || valid.Disposition != nativeGapCancellationRefused || valid.RuntimeCancelled || valid.NoPostAckWrites || valid.IntervalStructurallyComplete {
		t.Fatalf("synthetic refusal checker control failed: %+v", valid)
	}
	t.Run("known-observational-records", func(t *testing.T) {
		copyCapture := capture
		raw, _ := nativeGapCancellationRead(r, capture.ChildAfter)
		usage := map[string]int{"cache_write_input_tokens": 0, "cached_input_tokens": 0, "input_tokens": 1, "output_tokens": 1, "reasoning_output_tokens": 0, "total_tokens": 2}
		for _, item := range []struct {
			kind    string
			payload any
		}{
			{"world_state", map[string]any{"full": false, "state": map[string]any{"environments": map[string]any{"subagents": "observed"}}}},
			{"inter_agent_communication_metadata", map[string]any{"trigger_turn": false}},
			{"token_usage_record", map[string]any{"response_id": "response", "root_turn_id": "turn", "session_id": r.ChildID, "thread_id": r.ChildID, "turn_id": "turn", "usage": usage, "thread_token_usage": usage, "turn_token_usage": usage}},
		} {
			record := mustNativeGapJSONCompact(t, map[string]any{"ordinal": 7, "timestamp": capture.StartedAt, "type": item.kind, "payload": item.payload})
			raw = append(append(raw, record...), '\n')
		}
		copyCapture.ChildAfter = nativeGapCancellationWrite(t, filepath.Join(t.TempDir(), "observed.jsonl"), raw)
		if got := nativeGapCancellationRefusalFacts(r, copyCapture); !got.Qualified || got.RuntimeCancelled || got.NoPostAckWrites {
			t.Fatalf("known observational records rejected or promoted: %+v", got)
		}
	})
	for _, name := range []string{"missing-candidate", "wrong-candidate", "wrong-child", "missing-runtime-call", "wrong-runtime-call", "wrong-request", "foreign-guarantee-host", "unrelated-request-field", "missing-inventory", "changed-authority", "changed-context", "missing-task", "changed-prompt", "hidden-launch", "late-credit", "forged-flag", "missing-no-op-field", "different-manifest", "unknown-record", "truncated-record", "process-before-capture", "process-reordered"} {
		t.Run(name, func(t *testing.T) {
			var copyCapture nativeGapCancellationRefusalCapture
			if err := json.Unmarshal(mustNativeGapJSON(t, capture), &copyCapture); err != nil {
				t.Fatal(err)
			}
			copyReceipt := r
			replace := func(file *nativeEvidenceFile, raw []byte) {
				*file = nativeGapCancellationWrite(t, filepath.Join(t.TempDir(), filepath.Base(file.Path)), raw)
			}
			changeProcess := func(index int, change func(*nativeGapCancellationProcess)) {
				var p nativeGapCancellationProcess
				raw, _ := nativeGapCancellationRead(r, copyCapture.Guards[index].Process)
				if json.Unmarshal(raw, &p) != nil {
					t.Fatal("invalid fixture process")
				}
				change(&p)
				replace(&copyCapture.Guards[index].Process, mustNativeGapJSON(t, p))
			}
			switch name {
			case "missing-candidate":
				copyReceipt.CandidateSHA256 = ""
			case "wrong-candidate":
				changeProcess(0, func(p *nativeGapCancellationProcess) { p.Candidate.Path += ".other" })
			case "wrong-child":
				copyReceipt.ChildID += "-other"
			case "missing-runtime-call":
				copyCapture.Guards[0].Process = nativeEvidenceFile{}
			case "wrong-runtime-call":
				changeProcess(0, func(p *nativeGapCancellationProcess) { p.Argv[1] = "status" })
			case "process-before-capture", "process-reordered":
				index, offset := 0, -time.Second
				if name == "process-reordered" {
					index, offset = 1, 0
				}
				at, _ := time.Parse(time.RFC3339Nano, capture.StartedAt)
				changeProcess(index, func(p *nativeGapCancellationProcess) {
					p.StartedAt = at.Add(offset).Format(time.RFC3339Nano)
					p.FinishedAt = at.Add(offset + time.Millisecond).Format(time.RFC3339Nano)
				})
			case "wrong-request":
				replace(&copyCapture.Guards[0].Request, []byte(`{"require_cancellation":true}`))
			case "foreign-guarantee-host", "unrelated-request-field":
				g := &copyCapture.Guards[0]
				raw, _ := nativeGapCancellationRead(r, g.Request)
				var request codexNativeWorkerRequest
				_ = json.Unmarshal(raw, &request)
				if name == "foreign-guarantee-host" {
					request.HostSessionID = "foreign-host"
				} else {
					request.ContextDeliveryID = "unrelated-context"
				}
				replace(&g.Request, mustNativeGapJSON(t, request))
				replace(&g.OriginalRequest, mustNativeGapJSON(t, request))
				g.RequestPath = g.OriginalRequest.Path
				changeProcess(0, func(p *nativeGapCancellationProcess) { p.Argv[len(p.Argv)-1] = g.RequestPath })
			case "missing-inventory":
				copyCapture.Guards[0].BeforeInventory = nativeEvidenceFile{}
			case "changed-authority", "late-credit":
				file := copyCapture.Guards[0].After[r.AttemptPath]
				raw, _ := nativeGapCancellationRead(r, file)
				var attempt buildAttemptRecord
				_ = json.Unmarshal(raw, &attempt)
				if name == "late-credit" {
					attempt.CompletionPath = "late-credit.json"
				} else {
					attempt.WorkerRuns[0].Native.ChildID = "replacement"
				}
				replace(&file, mustNativeGapJSON(t, attempt))
				copyCapture.Guards[0].After[r.AttemptPath] = file
				replace(&copyCapture.Guards[0].AfterInventory, mustNativeGapJSON(t, copyCapture.Guards[0].After))
			case "changed-context":
				g := &copyCapture.Guards[0]
				path := filepath.Join(r.FixtureRoot, ".aether", "CONTEXT.md")
				file := g.After[path]
				replace(&file, []byte(`{"present":true,"content":"unsafe recovery"}`))
				g.After[path] = file
				replace(&g.AfterInventory, mustNativeGapJSON(t, g.After))
			case "missing-task", "changed-prompt":
				g := &copyCapture.Guards[0]
				path := r.AttemptPath
				if name == "missing-task" {
					path = filepath.Join(r.FixtureRoot, ".aether", "data", "COLONY_STATE.json")
				}
				file := g.Before[path]
				raw, _ := nativeGapCancellationRead(r, file)
				if name == "missing-task" {
					var state colony.ColonyState
					_ = json.Unmarshal(raw, &state)
					state.Plan.Phases[0].Tasks = nil
					raw = mustNativeGapJSON(t, state)
				} else {
					var attempt buildAttemptRecord
					_ = json.Unmarshal(raw, &attempt)
					attempt.WorkerRuns[0].Native.Prompt += " substituted"
					raw = mustNativeGapJSON(t, attempt)
				}
				replace(&file, raw)
				g.Before[path], g.After[path] = file, file
				replace(&g.BeforeInventory, mustNativeGapJSON(t, g.Before))
				replace(&g.AfterInventory, mustNativeGapJSON(t, g.After))
			case "hidden-launch":
				raw, _ := nativeGapCancellationRead(r, copyCapture.ParentAfter)
				raw = append(raw, []byte("{\"type\":\"response_item\",\"payload\":{\"type\":\"function_call\",\"name\":\"spawn_agent\",\"call_id\":\"hidden\"}}\n")...)
				replace(&copyCapture.ParentAfter, raw)
			case "unknown-record", "truncated-record":
				raw, _ := nativeGapCancellationRead(r, copyCapture.ChildAfter)
				more := []byte("{\n")
				if name == "unknown-record" {
					more = []byte("{\"type\":\"unmapped_event\"}\n")
				}
				replace(&copyCapture.ChildAfter, append(raw, more...))
			case "forged-flag":
				changeProcess(0, func(p *nativeGapCancellationProcess) { replace(&p.Stdout, []byte(`{"refused":true}`)) })
			case "missing-no-op-field":
				changeProcess(2, func(p *nativeGapCancellationProcess) {
					replace(&p.Stdout, []byte(`{"ok":true,"result":{"state_effect":"none"}}`))
				})
			case "different-manifest":
				g := &copyCapture.Guards[6]
				raw, _ := nativeGapCancellationRead(r, g.Request)
				var packet codexExternalBuildCompletion
				_ = json.Unmarshal(raw, &packet)
				packet.DispatchManifest.Dispatches[0].Task += " replacement"
				replace(&g.Request, mustNativeGapJSON(t, packet))
				replace(&g.OriginalRequest, mustNativeGapJSON(t, packet))
				g.RequestPath = g.OriginalRequest.Path
				changeProcess(6, func(p *nativeGapCancellationProcess) { p.Argv[len(p.Argv)-1] = g.RequestPath })
			}
			got := nativeGapCancellationRefusalFacts(copyReceipt, copyCapture)
			if got.Qualified || got.Disposition != nativeGapCancellationIncomplete || got.RuntimeCancelled || got.NoPostAckWrites {
				t.Fatalf("invalid %s accepted: %+v", name, got)
			}
		})
	}
}
