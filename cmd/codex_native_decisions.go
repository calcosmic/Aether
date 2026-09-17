package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

const codexNativeDecisionSource = "codex-native-question"

// A native question is a material clarification, never a checkpoint or waiver.
// QuestionID is the stable host question/event key; the durable row ID also
// commits to its exact assignment. Reusing the key with different text refuses.
type codexNativeQuestion struct {
	QuestionID string `json:"question_id"`
	Question   string `json:"question"`
}
type codexNativeDecisionBinding struct {
	SchemaVersion    int                    `json:"schema_version"`
	GoalHash         string                 `json:"goal_hash"`
	SessionID        string                 `json:"session_id"`
	Phase            int                    `json:"phase"`
	ExecutionBinding codex.ExecutionBinding `json:"execution_binding"`
	AttemptID        string                 `json:"attempt_id"`
	LaunchID         string                 `json:"launch_id"`
	WorkerName       string                 `json:"worker_name"`
	TaskID           string                 `json:"task_id"`
	ChildID          string                 `json:"child_id"`
	HostSessionID    string                 `json:"host_session_id"`
	Workspace        string                 `json:"workspace"`
	DispatchSHA256   string                 `json:"dispatch_sha256"`
	PromptSHA256     string                 `json:"prompt_sha256"`
	QuestionID       string                 `json:"question_id"`
	QuestionSHA256   string                 `json:"question_sha256"`
}
type codexNativeDecisionAnswerRequest struct {
	SchemaVersion int                        `json:"schema_version"`
	Binding       codexNativeDecisionBinding `json:"native_binding"`
	Question      string                     `json:"question"`
	Answer        string                     `json:"answer"`
}
type codexNativeDecisionHooks struct{ BeforeWrite func() }

var errCodexNativeDecisionReplay = errors.New("native decision replay: no write")

func isCodexNativeDecision(d PendingDecision) bool {
	return d.NativeBinding != nil || d.Source == codexNativeDecisionSource
}
func (b codexNativeDecisionBinding) workerRequest() codexNativeWorkerRequest {
	return codexNativeWorkerRequest{SchemaVersion: 1, Phase: b.Phase, ExecutionBinding: b.ExecutionBinding, WorkerName: b.WorkerName, TaskID: b.TaskID, LaunchID: b.LaunchID, HostSessionID: b.HostSessionID, ChildID: b.ChildID, DispatchSHA256: b.DispatchSHA256, PromptSHA256: b.PromptSHA256, Workspace: b.Workspace}
}
func codexNativeDecisionRowID(b codexNativeDecisionBinding) string {
	b.QuestionSHA256 = ""
	digest, _ := jsonSHA256(b)
	return "native-question-" + digest
}

// Caller holds the existing repository mutation session, then worker mutex.
// Reuse the bridge's exact saved-manifest validator and the shared native state
// validator rather than creating a second assignment or current-state authority.
func currentCodexNativeDecisionTarget(request codexNativeWorkerRequest) (buildAttemptRecord, *buildAttemptWorkerRun, error) {
	if request.SchemaVersion != 1 || request.Phase <= 0 {
		return buildAttemptRecord{}, nil, fmt.Errorf("native decision requires schema_version 1 and positive phase")
	}
	if _, err := executeCodexNativeWorkerRequest("inspect", request, codexNativeWorkerHooks{}); err != nil {
		return buildAttemptRecord{}, nil, err
	}
	_, record, ok := loadLatestBuildAttempt(request.Phase)
	if !ok {
		return record, nil, fmt.Errorf("native decision attempt is missing")
	}
	if record.PlanManifest.ContextScope == nil || record.PlanManifest.ContextScope.GoalHash == "" || record.PlanManifest.ContextScope.SessionID == "" {
		return record, nil, fmt.Errorf("native questions require the exact nonempty goal and session identity")
	}
	_, worker, err := validateCodexNativeContextTarget(record, request)
	if err != nil {
		return record, nil, err
	}
	var state colony.ColonyState
	if err = store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		return record, nil, err
	}
	if err = validateBuildFinalizeStateStillCurrent(state, request.Phase); err != nil {
		return record, nil, err
	}
	if err = validateCodexNativeAttemptState(record, state); err != nil {
		return record, nil, err
	}
	if worker.Native.ChildID == "" || worker.Native.LaunchState == "reserved" || worker.Native.LaunchState == "no_launch" {
		return record, nil, fmt.Errorf("native question requires a bound child")
	}
	if !codexNativeWorkerIsTerminal(*worker) {
		if err = validateCodexNativeContextLive(record, *worker); err != nil {
			return record, nil, err
		}
	}
	return record, worker, nil
}
func nativeDecisionBinding(record buildAttemptRecord, worker buildAttemptWorkerRun, q codexNativeQuestion) codexNativeDecisionBinding {
	n := worker.Native
	return codexNativeDecisionBinding{SchemaVersion: 1, GoalHash: record.PlanManifest.ContextScope.GoalHash, SessionID: record.PlanManifest.ContextScope.SessionID, Phase: record.Phase, ExecutionBinding: *record.PlanManifest.ExecutionBinding, AttemptID: record.ID, LaunchID: worker.ProviderRunID, WorkerName: worker.WorkerName, TaskID: worker.TaskID, ChildID: n.ChildID, HostSessionID: n.HostSessionID, Workspace: n.Workspace, DispatchSHA256: n.DispatchSHA256, PromptSHA256: n.PromptSHA256, QuestionID: q.QuestionID, QuestionSHA256: lifecycleDigest([]byte(q.Question))}
}
func validateCodexNativeQuestion(q codexNativeQuestion, file PendingDecisionFile) error {
	if strings.TrimSpace(q.QuestionID) == "" || len(q.QuestionID) > 256 || strings.TrimSpace(q.Question) == "" || len(q.Question) > 4096 || !utf8.ValidString(q.QuestionID) || !utf8.ValidString(q.Question) {
		return fmt.Errorf("native question requires a stable question_id and nonempty UTF-8 question within bounds")
	}
	if _, _, waiver := forcedReviewerWaiverSignalForQuestion(q.Question); waiver {
		return fmt.Errorf("native material questions cannot authorize reviewer waivers")
	}
	scope := loadCurrentPendingDecisionScope()
	for _, d := range file.Decisions {
		if isAutopilotCheckpointType(d.Type) && pendingDecisionMatchesScope(d, scope) && normalizeDecisionText(checkpointDecisionQuestion(d)) == normalizeDecisionText(q.Question) {
			return fmt.Errorf("native material questions cannot authorize owner checkpoints")
		}
	}
	return nil
}
func validateCodexNativeDecisionRow(d PendingDecision, b codexNativeDecisionBinding, question string) error {
	if d.NativeBinding == nil || *d.NativeBinding != b || d.ID != codexNativeDecisionRowID(b) || d.Source != codexNativeDecisionSource || d.Type != clarificationDecisionType || d.Description != question || b.QuestionSHA256 != lifecycleDigest([]byte(question)) || d.GoalHash != b.GoalHash || d.SessionID != b.SessionID || d.Phase == nil || *d.Phase != b.Phase || d.AttemptID != "" || d.CheckpointKey != "" || d.WorkGeneration != "" || d.HardConstraint || d.WaiverCapabilitySHA256 != "" || len(d.WaiverCapabilitySHA256s) > 0 || d.CheckpointCapabilitySHA256 != "" || len(d.CheckpointCapabilitySHA256s) > 0 {
		return fmt.Errorf("native decision question/binding or protected authorization metadata changed")
	}
	return nil
}
func admitCodexNativeDecision(request codexNativeWorkerRequest, q codexNativeQuestion, hooks codexNativeDecisionHooks) (PendingDecision, error) {
	if store == nil {
		return PendingDecision{}, fmt.Errorf("no store initialized")
	}
	if hooks.BeforeWrite != nil {
		hooks.BeforeWrite()
	}
	var decision PendingDecision
	err := withPlanningMutationSession(buildAttemptWorkspaceRoot(), "codex-native-question", func(_ *planningMutationSession) error {
		buildWorkerRunMutationMu.Lock()
		defer buildWorkerRunMutationMu.Unlock()
		var file PendingDecisionFile
		return store.UpdateJSONAtomically(pendingDecisionsFile, &file, func() error {
			record, worker, err := currentCodexNativeDecisionTarget(request)
			if err != nil {
				return err
			}
			if err = validateCodexNativeQuestion(q, file); err != nil {
				return err
			}
			b := nativeDecisionBinding(record, *worker, q)
			id := codexNativeDecisionRowID(b)
			for _, saved := range file.Decisions {
				if saved.ID != id {
					continue
				}
				if err = validateCodexNativeDecisionRow(saved, b, q.Question); err != nil {
					return err
				}
				decision = saved
				return errCodexNativeDecisionReplay
			}
			if codexNativeWorkerIsTerminal(*worker) && !codexNativeTerminalHasQuestion(*worker, q.Question) {
				return fmt.Errorf("terminal native question must be preserved in its saved handoff")
			}
			phase := record.Phase
			decision = PendingDecision{ID: id, Type: clarificationDecisionType, Description: q.Question, Phase: &phase, Source: codexNativeDecisionSource, GoalHash: b.GoalHash, SessionID: b.SessionID, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano), NativeBinding: &b}
			file.Decisions = append(file.Decisions, decision)
			return nil
		})
	})
	if errors.Is(err, errCodexNativeDecisionReplay) {
		err = nil
	}
	return decision, err
}
func codexNativeTerminalHasQuestion(worker buildAttemptWorkerRun, question string) bool {
	if worker.Result == nil {
		return false
	}
	for _, q := range worker.Result.Handoff.OpenDecisions {
		if q == question {
			return true
		}
	}
	for _, receipt := range worker.Result.TaskReceipts {
		for _, q := range receipt.Handoff.OpenDecisions {
			if q == question {
				return true
			}
		}
	}
	return false
}
func answerCodexNativeDecision(request codexNativeDecisionAnswerRequest, hooks codexNativeDecisionHooks) (PendingDecision, bool, error) {
	if store == nil {
		return PendingDecision{}, false, fmt.Errorf("no store initialized")
	}
	if request.SchemaVersion != 1 || request.Binding.SchemaVersion != 1 || strings.TrimSpace(request.Answer) == "" || len(request.Answer) > 8192 || !utf8.ValidString(request.Answer) {
		return PendingDecision{}, false, fmt.Errorf("native answer requires schema_version 1 and a nonempty bounded UTF-8 owner answer")
	}
	if hooks.BeforeWrite != nil {
		hooks.BeforeWrite()
	}
	var decision PendingDecision
	err := withPlanningMutationSession(buildAttemptWorkspaceRoot(), "codex-native-answer", func(_ *planningMutationSession) error {
		buildWorkerRunMutationMu.Lock()
		defer buildWorkerRunMutationMu.Unlock()
		var file PendingDecisionFile
		err := store.UpdateJSONAtomically(pendingDecisionsFile, &file, func() error {
			record, worker, err := currentCodexNativeDecisionTarget(request.Binding.workerRequest())
			if err != nil {
				return err
			}
			q := codexNativeQuestion{QuestionID: request.Binding.QuestionID, Question: request.Question}
			if err = validateCodexNativeQuestion(q, file); err != nil {
				return err
			}
			expected := nativeDecisionBinding(record, *worker, q)
			if request.Binding != expected {
				return fmt.Errorf("native answer binding is not current")
			}
			id := codexNativeDecisionRowID(expected)
			for i := range file.Decisions {
				d := &file.Decisions[i]
				if d.ID != id {
					continue
				}
				if err = validateCodexNativeDecisionRow(*d, expected, request.Question); err != nil {
					return err
				}
				if d.Resolved {
					if d.Resolution != request.Answer {
						return fmt.Errorf("native question already has a different answer")
					}
					decision = *d
					return errCodexNativeDecisionReplay
				}
				if d.Resolution != "" || d.ResolvedAt != "" {
					return fmt.Errorf("native pending question has inconsistent resolution")
				}
				d.Resolved = true
				d.Resolution = request.Answer
				d.ResolvedAt = time.Now().UTC().Format(time.RFC3339Nano)
				decision = *d
				return nil
			}
			return fmt.Errorf("native question does not exist; a stale answer cannot create a replacement")
		})
		if err != nil {
			return err
		}
		// Content-free first-commit facts only. The actual answer is private to this
		// bound row and its child renderer, never globally injected FEEDBACK.
		emitDecisionFeedback("Native question receipt "+decision.ID, "An owner answer was recorded for its exact native assignment; read it only through the bound context route.", request.Binding.Phase)
		episodeID, kind := currentLiveRecoveryEpisode(request.Binding.Phase)
		emitColonyLiveInterventionRecorded(episodeID, kind, episodeInterventionKindAnsweredWorkerQuestion)
		return nil
	})
	if errors.Is(err, errCodexNativeDecisionReplay) {
		return decision, true, nil
	}
	return decision, false, err
}

// Same bounded, regular-file and temporary-directory rules as the native bridge.
func loadCodexNativeDecisionAnswerRequest(path string) (codexNativeDecisionAnswerRequest, error) {
	var request codexNativeDecisionAnswerRequest
	if !filepath.IsAbs(path) {
		return request, fmt.Errorf("native answer requires an absolute --native-request file")
	}
	if err := validateFinalizerCompletionFilePath(path); err != nil {
		return request, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return request, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return request, fmt.Errorf("native answer request must be a regular non-symlink file")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return request, err
	}
	temp, err := filepath.EvalSymlinks(os.TempDir())
	if err != nil {
		return request, err
	}
	rel, err := filepath.Rel(temp, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || !strings.HasPrefix(filepath.Base(filepath.Dir(resolved)), "aether-worker-request-") {
		return request, fmt.Errorf("native answer request must use an aether-worker-request-* system temporary directory")
	}
	file, err := os.Open(path)
	if err != nil {
		return request, err
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, internalWorkerRequestMaxBytes+1))
	if err != nil {
		return request, err
	}
	if len(raw) > internalWorkerRequestMaxBytes {
		return request, fmt.Errorf("native answer request exceeds %d bytes", internalWorkerRequestMaxBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&request); err != nil {
		return request, err
	}
	if err = decoder.Decode(new(any)); err != io.EOF {
		return request, fmt.Errorf("native answer request must contain exactly one JSON object")
	}
	if request.SchemaVersion != 1 {
		return request, fmt.Errorf("native answer requires schema_version 1")
	}
	return request, nil
}

func hasCodexNativeDecisionQuestion(file PendingDecisionFile, question string) bool {
	for _, d := range file.Decisions {
		if isCodexNativeDecision(d) && normalizeDecisionText(d.Description) == normalizeDecisionText(question) {
			return true
		}
	}
	return false
}
