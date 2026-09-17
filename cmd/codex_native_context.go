package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

type codexNativePrompt struct {
	Prompt      string
	SHA256      string
	DecisionIDs []string
}

// renderCodexNativeContextAnswers is the scope hook for Plan 04's protected
// native-question renderer. Ordinary answers still use the one current-scope
// resolver and integrity/budget renderer. Filter delivered IDs BEFORE packing
// so older answers cannot crowd out newly applicable ones.
func renderCodexNativeContextAnswers(manifest codexBuildManifest, dispatch codexBuildDispatch, launch, child string, excluded, onlyIDs []string) clarifiedIntentRenderResult {
	file, _ := loadScopedPendingDecisionFile(loadCurrentPendingDecisionScope())
	entries := resolvedClarifiedIntentEntries(file)
	seen := make(map[string]bool, len(excluded))
	for _, id := range excluded {
		seen[id] = true
	}
	included := make(map[string]bool, len(onlyIDs))
	for _, id := range onlyIDs {
		included[id] = true
	}
	pending := entries[:0]
	for _, entry := range entries {
		if !seen[entry.ID] && (onlyIDs == nil || included[entry.ID]) {
			pending = append(pending, entry)
		}
	}
	source := pendingDecisionsFile
	if store != nil {
		source = filepath.Join(store.BasePath(), pendingDecisionsFile)
	}
	return renderClarifiedIntentPromptEntriesWithIntegrity(pending, source)
}

func codexNativeAnswerSection(result clarifiedIntentRenderResult) string {
	if len(result.Lines) == 0 {
		return ""
	}
	return "## CLARIFIED INTENT\n\n" + strings.Join(result.Lines, "\n")
}

func composeCodexNativePrompt(manifest codexBuildManifest, dispatch codexBuildDispatch, launch string) (codexNativePrompt, error) {
	brief := dispatch.Brief
	if dispatch.BriefPath != "" {
		path := filepath.Join(manifest.Root, filepath.FromSlash(dispatch.BriefPath))
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			return codexNativePrompt{}, fmt.Errorf("read native brief: %w", err)
		}
		root, err := filepath.EvalSymlinks(manifest.Root)
		if err != nil {
			return codexNativePrompt{}, err
		}
		rel, err := filepath.Rel(root, resolved)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return codexNativePrompt{}, fmt.Errorf("native brief is outside accepted workspace")
		}
		info, err := os.Stat(resolved)
		if err != nil || !info.Mode().IsRegular() {
			return codexNativePrompt{}, fmt.Errorf("native brief must be a readable regular file")
		}
		raw, err := os.ReadFile(resolved)
		if err != nil {
			return codexNativePrompt{}, err
		}
		brief = string(raw)
		if dispatch.BriefSHA256 == "" || lifecycleDigest(raw) != dispatch.BriefSHA256 || (dispatch.Brief != "" && dispatch.Brief != brief) {
			return codexNativePrompt{}, fmt.Errorf("native brief bytes do not match the accepted manifest identity")
		}
	}
	if strings.TrimSpace(brief) == "" {
		return codexNativePrompt{}, fmt.Errorf("native assignment has no required brief")
	}
	if dispatch.BriefSHA256 != "" && dispatch.BriefSHA256 != lifecycleDigest([]byte(brief)) {
		return codexNativePrompt{}, fmt.Errorf("native brief digest does not match accepted bytes")
	}
	prompt := fmt.Sprintf("You are %s (%s), assigned workspace %s. You are not alone; preserve others' edits.\nWAIT: do not read files, run checks or edit until the parent sends AETHER_NATIVE_RELEASE %s with your actual child ID and the runtime release token after binding. If no release arrives, remain waiting; do not do the job.\n", dispatch.Name, dispatch.Caste, manifest.Root, launch)
	prompt += manifest.ContextCapsule + "\n\n" + brief + "\n\n" + dispatch.SkillSection
	answers := renderCodexNativeContextAnswers(manifest, dispatch, launch, "", manifest.ContextDecisionIDs, nil)
	if len(answers.Lines) > 0 {
		prompt += "\n\n" + codexNativeAnswerSection(answers)
	}
	prompt += fmt.Sprintf("\nReturn one JSON terminal result: name or ant_name=%q (if both are supplied they must agree), caste=%q, task_id=%q, status (code_written/completed/failed/blocked/timeout), summary, files_created, files_modified, tests_written, task_receipts, blockers, spawns, handoff; optional installed Builder tdd fields cycles_completed/tests_added/coverage_percent/all_passing are preserved without becoming provider telemetry. Each task receipt has task_id, status, summary, files_created, files_modified, tests_written, handoff. Every handoff uses %s. verification_status MUST be one enum value: pass, fail, partial, not_run, or unknown; put explanations in summary/known_failures, never in verification_status. Report only actual checks; omit usage. Do not stage, finalize, commit, recruit or launch helpers. Parent saves your terminal response.\n", dispatch.Name, dispatch.Caste, normalizedDispatchTaskID(dispatch), codex.HandoffFieldsSummary)
	if !utf8.ValidString(prompt) {
		return codexNativePrompt{}, fmt.Errorf("native prompt must contain valid UTF-8 bytes")
	}
	ids := append([]string(nil), manifest.ContextDecisionIDs...)
	ids = append(ids, answers.DecisionIDs...)
	return codexNativePrompt{Prompt: prompt, SHA256: lifecycleDigest([]byte(prompt)), DecisionIDs: ids}, nil
}

// codexNativeContextScope pins both colony identities. The ordinary decision
// resolver retains its legacy compatibility rules; native delivery may not.
type codexNativeContextScope struct {
	GoalHash  string `json:"goal_hash"`
	SessionID string `json:"session_id"`
}

type codexNativeContextDelivery struct {
	SchemaVersion    int                     `json:"schema_version"`
	DeliveryID       string                  `json:"delivery_id"`
	ExecutionBinding codex.ExecutionBinding  `json:"execution_binding"`
	Scope            codexNativeContextScope `json:"scope"`
	WorkerName       string                  `json:"worker_name"`
	TaskID           string                  `json:"task_id"`
	LaunchID         string                  `json:"launch_id"`
	HostSessionID    string                  `json:"host_session_id"`
	ChildID          string                  `json:"child_id"`
	Workspace        string                  `json:"workspace"`
	DispatchSHA256   string                  `json:"dispatch_sha256"`
	PromptSHA256     string                  `json:"prompt_sha256"`
	Payload          string                  `json:"payload"`
	PayloadSHA256    string                  `json:"payload_sha256"`
	DecisionIDs      []string                `json:"decision_ids"`
}

// This is an observed successful host send, not a render/queue receipt.
// Source event identity/hash and time accompany it in the observe request.
// As with other host observations, hashes retain provenance, not authentication.
type codexNativeContextSend struct {
	Status        string `json:"status"`
	ChildID       string `json:"child_id"`
	MessageSHA256 string `json:"message_sha256"`
}

type codexNativeContextReceipt struct {
	Delivery    codexNativeContextDelivery `json:"delivery"`
	Observation codexNativeHostObservation `json:"observation"`
}

func codexNativeContextScopeFromState(state colony.ColonyState) *codexNativeContextScope {
	scope := pendingDecisionScopeFromState(state)
	return &codexNativeContextScope{GoalHash: scope.GoalHash, SessionID: scope.SessionID}
}

func validateCodexNativeContextScope(manifest codexBuildManifest) error {
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		return err
	}
	current := codexNativeContextScopeFromState(state)
	if manifest.ContextScope == nil || *manifest.ContextScope != *current || current.GoalHash != pendingDecisionGoalHash(manifest.Goal) {
		return fmt.Errorf("native context goal/session no longer matches accepted manifest")
	}
	return nil
}

func validateCodexNativeContextTarget(record buildAttemptRecord, request codexNativeWorkerRequest) (*codexBuildDispatch, *buildAttemptWorkerRun, error) {
	if err := validateCodexNativeContextScope(*record.PlanManifest); err != nil {
		return nil, nil, err
	}
	for i := range record.PlanManifest.Dispatches {
		dispatch := &record.PlanManifest.Dispatches[i]
		if dispatch.Name != request.WorkerName || normalizedDispatchTaskID(*dispatch) != request.TaskID {
			continue
		}
		for j := range record.WorkerRuns {
			worker := &record.WorkerRuns[j]
			if worker.WorkerName != request.WorkerName || worker.TaskID != request.TaskID {
				continue
			}
			if err := validateCodexNativeSavedWorker(record, *dispatch, *worker); err != nil {
				return nil, nil, err
			}
			native := worker.Native
			if request.ChildID == "" || request.ChildID != native.ChildID || request.LaunchID != worker.ProviderRunID || request.HostSessionID != native.HostSessionID || request.PromptSHA256 != native.PromptSHA256 || request.DispatchSHA256 != native.DispatchSHA256 || request.Workspace != native.Workspace || (request.HostPermission != "" && request.HostPermission != string(native.PermissionProfile.Name)) {
				return nil, nil, fmt.Errorf("native context requires the exact saved child/launch/session/workspace/prompt binding")
			}
			return dispatch, worker, nil
		}
	}
	return nil, nil, fmt.Errorf("native context requires a saved bound assignment")
}

func validateCodexNativeContextLive(record buildAttemptRecord, worker buildAttemptWorkerRun) error {
	if worker.Native.ChildID == "" || worker.Native.LaunchState != "bound" || codexNativeWorkerIsTerminal(worker) {
		return fmt.Errorf("native context requires a live bound child")
	}
	for _, event := range worker.Native.Observations {
		if event.Status == "cancel_requested" {
			return fmt.Errorf("native context cannot message a child pending cancellation")
		}
	}
	return validateCodexNativeLaunchCurrency(record)
}

func composeCodexNativeContextDelivery(record buildAttemptRecord, dispatch codexBuildDispatch, worker buildAttemptWorkerRun, onlyIDs []string) (*codexNativeContextDelivery, error) {
	if err := validateCodexNativeContextLive(record, worker); err != nil {
		return nil, err
	}
	excluded := append([]string(nil), worker.Native.ContextDecisionIDs...)
	for _, saved := range worker.Native.ContextDeliveries {
		excluded = append(excluded, saved.Delivery.DecisionIDs...)
	}
	answers := renderCodexNativeContextAnswers(*record.PlanManifest, dispatch, worker.ProviderRunID, worker.Native.ChildID, excluded, onlyIDs)
	if len(answers.DecisionIDs) == 0 {
		return nil, nil
	}
	delivery := &codexNativeContextDelivery{
		SchemaVersion: 1, ExecutionBinding: *record.PlanManifest.ExecutionBinding, Scope: *record.PlanManifest.ContextScope,
		WorkerName: worker.WorkerName, TaskID: worker.TaskID, LaunchID: worker.ProviderRunID,
		HostSessionID: worker.Native.HostSessionID, ChildID: worker.Native.ChildID, Workspace: worker.Native.Workspace,
		DispatchSHA256: worker.Native.DispatchSHA256, PromptSHA256: worker.Native.PromptSHA256,
		Payload: codexNativeAnswerSection(answers), DecisionIDs: append([]string(nil), answers.DecisionIDs...),
	}
	delivery.PayloadSHA256 = lifecycleDigest([]byte(delivery.Payload))
	digest, err := jsonSHA256(delivery)
	if err != nil {
		return nil, err
	}
	delivery.DeliveryID = digest
	return delivery, nil
}

func readCodexNativeContext(record buildAttemptRecord, request codexNativeWorkerRequest) (codexNativeWorkerResponse, error) {
	response := codexNativeWorkerResponse{SchemaVersion: 1, ExecutionBinding: request.ExecutionBinding}
	if request.Result != nil || request.SourceEventID != "" || request.SourceEventSHA256 != "" || request.ContextDelivery != nil || request.ContextSend != nil {
		return response, fmt.Errorf("read-only native context cannot submit delivery evidence")
	}
	dispatch, worker, err := validateCodexNativeContextTarget(record, request)
	if err != nil {
		return response, err
	}
	if request.ContextDeliveryID != "" {
		for _, saved := range worker.Native.ContextDeliveries {
			if saved.Delivery.DeliveryID == request.ContextDeliveryID {
				if err := validateCodexNativeContextReceipt(saved, record, *worker); err != nil {
					return response, err
				}
				delivery := saved.Delivery
				response.ContextDelivery, response.ContextStatus = &delivery, "delivered"
				return response, nil
			}
		}
	}
	delivery, err := composeCodexNativeContextDelivery(record, *dispatch, *worker, nil)
	if err != nil {
		return response, err
	}
	if request.ContextDeliveryID != "" && (delivery == nil || delivery.DeliveryID != request.ContextDeliveryID) {
		return response, fmt.Errorf("native context delivery is not current or acknowledged")
	}
	response.ContextStatus = "no_updates"
	if delivery != nil {
		response.ContextDelivery, response.ContextStatus = delivery, "awaiting_delivery"
	}
	return response, nil
}

func observeCodexNativeContext(record *buildAttemptRecord, worker *buildAttemptWorkerRun, dispatch codexBuildDispatch, request codexNativeWorkerRequest, now string) (*codexNativeWorkerReceipt, error) {
	if _, _, err := validateCodexNativeContextTarget(*record, request); err != nil {
		return nil, err
	}
	delivery, send := request.ContextDelivery, request.ContextSend
	if delivery == nil || send == nil || request.Result != nil || request.ContextDeliveryID != "" || send.Status != "completed" || send.ChildID != request.ChildID || send.MessageSHA256 != delivery.PayloadSHA256 {
		return nil, fmt.Errorf("context delivery requires an observed completed send of the exact payload to its bound child")
	}
	digest, err := canonicalCodexNativeEventHash(request.SourceEventID, request.SourceEventSHA256)
	if err != nil {
		return nil, err
	}
	at, err := time.Parse(time.RFC3339Nano, request.ObservedAt)
	if err != nil {
		return nil, fmt.Errorf("context send requires an actual observed_at timestamp")
	}
	observation := codexNativeHostObservation{SchemaVersion: 1, Status: "context_delivered", ChildID: request.ChildID, ObservedAt: at.UTC().Format(time.RFC3339Nano), Detail: request.ObservationDetail, SourceEventID: request.SourceEventID, SourceEventSHA256: digest, ContextDeliveryID: delivery.DeliveryID}
	receipt := codexNativeTransitionReceipt("observe", *worker)
	receipt.At, receipt.SourceEventID, receipt.SourceEventSHA256 = observation.ObservedAt, observation.SourceEventID, digest
	receipt.ContextDeliveryID, receipt.ContextPayloadSHA256 = delivery.DeliveryID, delivery.PayloadSHA256
	if err := validateCodexNativeContextEnvelope(*delivery, *record, *worker); err != nil {
		return nil, err
	}
	native := worker.Native
	for _, saved := range native.ContextDeliveries {
		if saved.Delivery.DeliveryID != delivery.DeliveryID {
			continue
		}
		if err := validateCodexNativeContextReceipt(saved, *record, *worker); err != nil {
			return nil, err
		}
		savedDigest, _ := jsonSHA256(saved.Delivery)
		receivedDigest, _ := jsonSHA256(delivery)
		if savedDigest != receivedDigest || saved.Observation != observation {
			return nil, fmt.Errorf("native context acknowledgement conflicts with saved delivery")
		}
		return &receipt, errCodexNativeReplay
	}
	for _, saved := range native.Observations {
		if saved.SourceEventID == observation.SourceEventID {
			return nil, fmt.Errorf("native context source event already belongs to another observation")
		}
	}
	if len(native.Observations) > 0 {
		previous, _ := time.Parse(time.RFC3339Nano, native.Observations[len(native.Observations)-1].ObservedAt)
		if !at.After(previous) {
			return nil, fmt.Errorf("native context send observation arrived out of order")
		}
	}
	seen := make(map[string]bool)
	for _, id := range delivery.DecisionIDs {
		if id == "" || seen[id] {
			return nil, fmt.Errorf("native context answer IDs must be nonempty and unique")
		}
		seen[id] = true
	}
	if len(seen) == 0 {
		return nil, fmt.Errorf("native context delivery contains no answers")
	}
	// Revalidate only the sent answer set: a newer answer arriving during send
	// must not invalidate a successful delivery of still-current instructions.
	expected, err := composeCodexNativeContextDelivery(*record, dispatch, *worker, delivery.DecisionIDs)
	if err != nil {
		return nil, err
	}
	if expected == nil || expected.DeliveryID != delivery.DeliveryID {
		return nil, fmt.Errorf("native context delivery is stale or differs from runtime-rendered answers")
	}
	native.ContextDeliveryIDs = append(native.ContextDeliveryIDs, delivery.DeliveryID)
	native.ContextDeliveries = append(native.ContextDeliveries, codexNativeContextReceipt{Delivery: *expected, Observation: observation})
	native.Observations = append(native.Observations, observation)
	worker.UpdatedAt, record.UpdatedAt = now, now
	return &receipt, nil
}

func validateCodexNativeContextEnvelope(delivery codexNativeContextDelivery, record buildAttemptRecord, worker buildAttemptWorkerRun) error {
	manifest, native := record.PlanManifest, worker.Native
	if manifest == nil || manifest.ExecutionBinding == nil || manifest.ContextScope == nil || native == nil || delivery.SchemaVersion != 1 ||
		delivery.ExecutionBinding != *manifest.ExecutionBinding || delivery.Scope != *manifest.ContextScope ||
		delivery.WorkerName != worker.WorkerName || delivery.TaskID != worker.TaskID || delivery.LaunchID != worker.ProviderRunID ||
		delivery.HostSessionID != native.HostSessionID || delivery.ChildID == "" || delivery.ChildID != native.ChildID ||
		delivery.Workspace != native.Workspace || delivery.DispatchSHA256 != native.DispatchSHA256 || delivery.PromptSHA256 != native.PromptSHA256 {
		return fmt.Errorf("native context envelope target does not match saved assignment")
	}
	id := delivery.DeliveryID
	delivery.DeliveryID = ""
	digest, err := jsonSHA256(delivery)
	if err != nil || id == "" || id != digest || delivery.PayloadSHA256 != lifecycleDigest([]byte(delivery.Payload)) {
		return fmt.Errorf("native context payload or delivery identity changed")
	}
	return nil
}

func validateCodexNativeContextReceipt(saved codexNativeContextReceipt, record buildAttemptRecord, worker buildAttemptWorkerRun) error {
	if err := validateCodexNativeContextEnvelope(saved.Delivery, record, worker); err != nil {
		return err
	}
	observation := saved.Observation
	digest, err := canonicalCodexNativeEventHash(observation.SourceEventID, observation.SourceEventSHA256)
	if err != nil {
		return err
	}
	if _, err := time.Parse(time.RFC3339Nano, observation.ObservedAt); err != nil {
		return fmt.Errorf("saved native context observation has invalid time")
	}
	if observation.SchemaVersion != 1 || observation.Status != "context_delivered" || observation.ContextDeliveryID != saved.Delivery.DeliveryID || observation.ChildID != saved.Delivery.ChildID || digest != observation.SourceEventSHA256 {
		return fmt.Errorf("saved native context observation does not match delivered envelope")
	}
	acknowledged := 0
	for _, id := range worker.Native.ContextDeliveryIDs {
		if id == saved.Delivery.DeliveryID {
			acknowledged++
		}
	}
	observed := false
	for _, event := range worker.Native.Observations {
		if event == observation {
			observed = true
		}
	}
	if acknowledged != 1 || !observed {
		return fmt.Errorf("saved native context delivery lacks its exact acknowledgement")
	}
	return nil
}
