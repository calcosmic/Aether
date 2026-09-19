package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

const codexNativeContextProtocolChildFetch = "child-fetch/v1"

type codexNativePrompt struct {
	Prompt      string
	SHA256      string
	DecisionIDs []string
}

// renderCodexNativeContextAnswers combines exactly scoped native answers with
// ordinary answers, which still use the one current-scope
// resolver and integrity/budget renderer. Filter delivered IDs BEFORE packing
// so older answers cannot crowd out newly applicable ones.
func renderCodexNativeContextAnswers(manifest codexBuildManifest, dispatch codexBuildDispatch, launch, child string, excluded, onlyIDs []string) clarifiedIntentRenderResult {
	file, _ := loadScopedPendingDecisionFile(loadCurrentPendingDecisionScope())
	entries := resolvedClarifiedIntentEntries(file)
	entries = append(entries, renderCodexNativeDecisionEntries(manifest, dispatch, launch, child)...)
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
	if manifest.ContextProtocol == codexNativeContextProtocolChildFetch {
		prompt += "While waiting, remain idle, wait passively with collaboration.wait_agent({\"timeout_ms\":3600000}), or send a brief commentary message; do not search ALL_TOOLS or any tool catalogue, run other tools, submit a final answer or ask a material question. The context and context-ack operations are aether codex-native-worker CLI subcommands, not tool-catalogue names. After the bound release, before any other tool work or questions, execute the exact parent-provided initial-context read command in the assigned workspace through tools.exec_command and read its entire returned payload. For code-mode execution use text(await tools.exec_command({cmd:COMMAND_JSON,workdir:WORKSPACE_JSON,max_output_tokens:20000})); substituting only the literal command and workspace JSON strings and printing the untouched complete result. After a successful complete read, your very next tool call must use the same literal wrapper to run: SAME_EXECUTABLE codex-native-worker context-ack --request SAME_READ_REQUEST --delivery-id FETCHED_DELIVERY_ID --payload-sha256 FETCHED_PAYLOAD_SHA256. Copy the absolute executable and exact --request path from that read command; replace only its context subcommand with context-ack, then append the values actually fetched. Append --decision-id FETCHED_ID once per returned decision ID, in order; null or empty decision_ids means no such flags. Never pass --release-token: it is not a context-ack option. Do not batch the read and ACK or send_message, wait_agent, request another parent handshake, ask questions, search tools, inspect files or make any other tool call between them. If the read fails or is incomplete, stop without claiming or sending an ACK and do not begin useful work. Fetch answers and separately ACK their exact bytes before answer-dependent work. A read alone is not an acknowledgement. Never fill ACK fields from parent-provided claims or write parent observations yourself.\n"
	}
	prompt += manifest.ContextCapsule + "\n\n" + brief + "\n\n" + dispatch.SkillSection
	answers := renderCodexNativeContextAnswers(manifest, dispatch, launch, "", manifest.ContextDecisionIDs, nil)
	if len(answers.Lines) > 0 {
		prompt += "\n\n" + codexNativeAnswerSection(answers)
	}
	prompt += fmt.Sprintf("\nReturn one JSON terminal result: name or ant_name=%q (if both are supplied they must agree), caste=%q, task_id=%q, status (code_written/completed/failed/blocked/timeout), summary, files_created, files_modified, tests_written, task_receipts, blockers, spawns, handoff; optional installed Builder tdd fields cycles_completed/tests_added/coverage_percent/all_passing are preserved without becoming provider telemetry. Each task receipt has task_id, status, summary, files_created, files_modified, tests_written, handoff. Every handoff uses %s. verification_status MUST be one enum value: pass, fail, partial, not_run, or unknown; put explanations in summary/known_failures, never in verification_status. Report only actual checks; omit usage. Do not stage, finalize, commit, recruit or launch helpers. Parent saves your terminal response.\n", dispatch.Name, dispatch.Caste, normalizedDispatchTaskID(dispatch), codex.HandoffFieldsSummary)
	if manifest.ContextProtocol == codexNativeContextProtocolChildFetch {
		prompt += "\nCHILD-FETCH BOOTSTRAP REMINDER: Before the bound release, wait passively. After release and a successful complete initial context read, use the release already received; the WAIT text repeated inside that fetched payload does not request a second release. Next tool call: SAME_EXECUTABLE codex-native-worker context-ack --request SAME_READ_REQUEST --delivery-id FETCHED_DELIVERY_ID --payload-sha256 FETCHED_PAYLOAD_SHA256. Reuse the read command's exact absolute executable and --request path; only the context subcommand changes to context-ack. Append --decision-id for each fetched ID in order, or none for null/empty decision_ids. Do not add --release-token or any guessed flags. Do not send_message, wait again, ask the parent to confirm, or make any other tool call before ACK. Apply the same immediate separate ACK rule to fetched answers. A failed or incomplete read is not ACK evidence: stop without inventing values or starting useful work.\n"
	}
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
	Protocol         string                  `json:"protocol,omitempty"`
	Purpose          string                  `json:"purpose,omitempty"`
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
	Fetch       *codexNativeContextFetch   `json:"fetch,omitempty"`
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
	if worker.Native.ContextProtocol == codexNativeContextProtocolChildFetch {
		delivery.SchemaVersion, delivery.Protocol, delivery.Purpose = 2, codexNativeContextProtocolChildFetch, "answers"
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
	if request.Result != nil || request.SourceEventID != "" || request.SourceEventSHA256 != "" || request.ContextDelivery != nil || request.ContextSend != nil || request.ContextFetch != nil || request.ContextPayloadSHA256 != "" || request.ContextDecisionIDs != nil {
		return response, fmt.Errorf("read-only native context cannot submit delivery evidence")
	}
	dispatch, worker, err := validateCodexNativeContextTarget(record, request)
	if err != nil {
		return response, err
	}
	if err := validateCodexNativeContextPurpose(worker.Native.ContextProtocol, request.ContextPurpose); err != nil {
		return response, err
	}
	if request.ContextDeliveryID != "" {
		for _, saved := range worker.Native.ContextDeliveries {
			if saved.Delivery.DeliveryID == request.ContextDeliveryID {
				if saved.Delivery.Purpose != request.ContextPurpose {
					return response, fmt.Errorf("native context purpose differs from saved receipt")
				}
				if err := validateCodexNativeContextReceipt(saved, record, *worker); err != nil {
					return response, err
				}
				delivery := saved.Delivery
				response.ContextDelivery, response.ContextStatus = &delivery, "delivered"
				return response, nil
			}
		}
	}
	delivery, err := composeCodexNativeContextForPurpose(record, *dispatch, *worker, request.ContextPurpose, nil)
	if err != nil {
		return response, err
	}
	if request.ContextDeliveryID != "" && (delivery == nil || delivery.DeliveryID != request.ContextDeliveryID) {
		return response, fmt.Errorf("native context delivery is not current or acknowledged")
	}
	response.ContextStatus = "no_updates"
	if delivery != nil {
		response.ContextDelivery, response.ContextStatus = delivery, "awaiting_delivery"
		if worker.Native.ContextProtocol == codexNativeContextProtocolChildFetch {
			response.ContextStatus = "awaiting_ack"
		}
	}
	return response, nil
}

func observeCodexNativeContext(record *buildAttemptRecord, worker *buildAttemptWorkerRun, dispatch codexBuildDispatch, request codexNativeWorkerRequest, now string) (*codexNativeWorkerReceipt, error) {
	if _, _, err := validateCodexNativeContextTarget(*record, request); err != nil {
		return nil, err
	}
	if record.PlanManifest.ContextProtocol != "" || request.ContextFetch != nil || request.ContextPurpose != "" || request.ContextPayloadSHA256 != "" || request.ContextDecisionIDs != nil {
		return nil, fmt.Errorf("host-send evidence cannot acknowledge a child-fetch protocol")
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
	if manifest == nil || manifest.ExecutionBinding == nil || manifest.ContextScope == nil || native == nil ||
		delivery.ExecutionBinding != *manifest.ExecutionBinding || delivery.Scope != *manifest.ContextScope ||
		delivery.WorkerName != worker.WorkerName || delivery.TaskID != worker.TaskID || delivery.LaunchID != worker.ProviderRunID ||
		delivery.HostSessionID != native.HostSessionID || delivery.ChildID == "" || delivery.ChildID != native.ChildID ||
		delivery.Workspace != native.Workspace || delivery.DispatchSHA256 != native.DispatchSHA256 || delivery.PromptSHA256 != native.PromptSHA256 {
		return fmt.Errorf("native context envelope target does not match saved assignment")
	}
	if err := validateCodexNativeContextPurpose(native.ContextProtocol, delivery.Purpose); err != nil {
		return err
	}
	if native.ContextProtocol != manifest.ContextProtocol || delivery.Protocol != native.ContextProtocol || (native.ContextProtocol == "" && delivery.SchemaVersion != 1) || (native.ContextProtocol != "" && delivery.SchemaVersion != 2) {
		return fmt.Errorf("native context envelope protocol does not match accepted manifest")
	}
	if !utf8.ValidString(delivery.Payload) || strings.TrimSpace(delivery.Payload) == "" {
		return fmt.Errorf("native context payload must contain complete UTF-8 text")
	}
	if err := validateCodexNativeContextDecisionIDs(delivery.DecisionIDs, delivery.Purpose == "answers" || delivery.Protocol == ""); err != nil {
		return err
	}
	if delivery.Purpose == "initial" && (delivery.Payload != native.Prompt || !equalCodexNativeContextIDs(delivery.DecisionIDs, native.ContextDecisionIDs)) {
		return fmt.Errorf("initial context must contain the exact saved prompt and decision IDs")
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
	expectedStatus := "context_delivered"
	if saved.Delivery.Protocol == codexNativeContextProtocolChildFetch {
		expectedStatus = "context_fetched"
		if saved.Fetch == nil {
			return fmt.Errorf("saved child-fetch receipt lacks read/ACK sources")
		}
		if err := validateCodexNativeContextFetch(*saved.Fetch, saved.Delivery); err != nil {
			return err
		}
		if observation.SourceEventID != saved.Fetch.Ack.Result.ID || observation.SourceEventSHA256 != saved.Fetch.Ack.Result.SHA256 || observation.ObservedAt != saved.Fetch.Ack.CompletedAt {
			return fmt.Errorf("saved fetch observation differs from ACK source")
		}
		bound, boundErr := time.Parse(time.RFC3339Nano, worker.Native.BoundAt)
		readAt, _ := time.Parse(time.RFC3339Nano, saved.Fetch.Read.StartedAt)
		if boundErr != nil || !readAt.After(bound) {
			return fmt.Errorf("saved context read precedes its accepted child binding")
		}
	} else if saved.Fetch != nil {
		return fmt.Errorf("legacy context receipt cannot contain child-fetch sources")
	}
	if observation.SchemaVersion != 1 || observation.Status != expectedStatus || observation.ContextDeliveryID != saved.Delivery.DeliveryID || observation.ChildID != saved.Delivery.ChildID || digest != observation.SourceEventSHA256 {
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

// These references identify original host records. They retain provenance;
// the qualification reader independently checks the original bytes and joins.
type codexNativeContextEventRef struct {
	ID     string `json:"id"`
	SHA256 string `json:"sha256"`
}

type codexNativeContextSource struct {
	ChildID     string                     `json:"child_id"`
	TurnID      string                     `json:"turn_id"`
	CallID      string                     `json:"call_id"`
	Call        codexNativeContextEventRef `json:"call"`
	Command     codexNativeContextEventRef `json:"command"`
	Result      codexNativeContextEventRef `json:"result"`
	StartedAt   string                     `json:"started_at"`
	CompletedAt string                     `json:"completed_at"`
}

type codexNativeContextFetch struct {
	SchemaVersion int                      `json:"schema_version"`
	Read          codexNativeContextSource `json:"read"`
	Ack           codexNativeContextSource `json:"ack"`
}

type codexNativeContextAck struct {
	SchemaVersion int      `json:"schema_version"`
	DeliveryID    string   `json:"delivery_id"`
	PayloadSHA256 string   `json:"payload_sha256"`
	ChildID       string   `json:"child_id"`
	DecisionIDs   []string `json:"decision_ids"`
}

func equalCodexNativeContextIDs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func validateCodexNativeContextDecisionIDs(ids []string, required bool) error {
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if strings.TrimSpace(id) == "" || seen[id] {
			return fmt.Errorf("native context decision IDs must be nonempty and unique")
		}
		seen[id] = true
	}
	if required && len(ids) == 0 {
		return fmt.Errorf("native answer context requires decision IDs")
	}
	return nil
}

func validateCodexNativeContextPurpose(protocol, purpose string) error {
	switch protocol {
	case "":
		if purpose == "" {
			return nil
		}
	case codexNativeContextProtocolChildFetch:
		if purpose == "initial" || purpose == "answers" {
			return nil
		}
	}
	return fmt.Errorf("native context protocol/purpose is unsupported or missing")
}

func composeCodexNativeContextForPurpose(record buildAttemptRecord, dispatch codexBuildDispatch, worker buildAttemptWorkerRun, purpose string, onlyIDs []string) (*codexNativeContextDelivery, error) {
	if err := validateCodexNativeContextPurpose(worker.Native.ContextProtocol, purpose); err != nil {
		return nil, err
	}
	if purpose != "initial" {
		return composeCodexNativeContextDelivery(record, dispatch, worker, onlyIDs)
	}
	if err := validateCodexNativeContextLive(record, worker); err != nil {
		return nil, err
	}
	for _, saved := range worker.Native.ContextDeliveries {
		if saved.Delivery.Purpose == "initial" {
			return nil, nil
		}
	}
	n := worker.Native
	delivery := &codexNativeContextDelivery{
		SchemaVersion: 2, Protocol: codexNativeContextProtocolChildFetch, Purpose: "initial",
		ExecutionBinding: *record.PlanManifest.ExecutionBinding, Scope: *record.PlanManifest.ContextScope,
		WorkerName: worker.WorkerName, TaskID: worker.TaskID, LaunchID: worker.ProviderRunID,
		HostSessionID: n.HostSessionID, ChildID: n.ChildID, Workspace: n.Workspace,
		DispatchSHA256: n.DispatchSHA256, PromptSHA256: n.PromptSHA256,
		Payload: n.Prompt, PayloadSHA256: n.PromptSHA256, DecisionIDs: append([]string(nil), n.ContextDecisionIDs...),
	}
	id, err := jsonSHA256(delivery)
	if err != nil {
		return nil, err
	}
	delivery.DeliveryID = id
	return delivery, nil
}

// ACK is a separate read-only command. It validates the exact envelope named
// by the child; it cannot create a delivery receipt or grant terminal credit.
func readCodexNativeContextAck(record buildAttemptRecord, request codexNativeWorkerRequest) (codexNativeWorkerResponse, error) {
	response := codexNativeWorkerResponse{SchemaVersion: 1, ExecutionBinding: request.ExecutionBinding}
	if request.Result != nil || request.SourceEventID != "" || request.SourceEventSHA256 != "" || request.ContextDelivery != nil || request.ContextSend != nil || request.ContextFetch != nil {
		return response, fmt.Errorf("native context ACK cannot submit parent observation evidence")
	}
	dispatch, worker, err := validateCodexNativeContextTarget(record, request)
	if err != nil {
		return response, err
	}
	if worker.Native.ContextProtocol != codexNativeContextProtocolChildFetch {
		return response, fmt.Errorf("native context ACK requires child-fetch/v1")
	}
	if err := validateCodexNativeContextPurpose(worker.Native.ContextProtocol, request.ContextPurpose); err != nil {
		return response, err
	}
	if err := validateCodexNativeContextDecisionIDs(request.ContextDecisionIDs, request.ContextPurpose == "answers"); err != nil {
		return response, err
	}
	if request.ContextDeliveryID == "" || request.ContextPayloadSHA256 == "" {
		return response, fmt.Errorf("native context ACK requires the fetched delivery ID and payload SHA-256")
	}
	delivery, err := composeCodexNativeContextForPurpose(record, *dispatch, *worker, request.ContextPurpose, request.ContextDecisionIDs)
	if err != nil {
		return response, err
	}
	if delivery == nil || delivery.DeliveryID != request.ContextDeliveryID || delivery.PayloadSHA256 != request.ContextPayloadSHA256 || !equalCodexNativeContextIDs(delivery.DecisionIDs, request.ContextDecisionIDs) {
		return response, fmt.Errorf("native context ACK differs from the current exact fetched envelope")
	}
	response.ContextStatus = "ack_validated"
	response.ContextAck = &codexNativeContextAck{SchemaVersion: 1, DeliveryID: delivery.DeliveryID, PayloadSHA256: delivery.PayloadSHA256, ChildID: delivery.ChildID, DecisionIDs: append([]string(nil), delivery.DecisionIDs...)}
	return response, nil
}

func validateCodexNativeContextFetch(fetch codexNativeContextFetch, delivery codexNativeContextDelivery) error {
	if fetch.SchemaVersion != 1 || delivery.Protocol != codexNativeContextProtocolChildFetch || delivery.SchemaVersion != 2 {
		return fmt.Errorf("native fetch source protocol is invalid")
	}
	seen := make(map[string]bool)
	var readCompleted time.Time
	for i, source := range []codexNativeContextSource{fetch.Read, fetch.Ack} {
		if source.ChildID == "" || source.ChildID != delivery.ChildID || strings.TrimSpace(source.TurnID) == "" || strings.TrimSpace(source.CallID) == "" || seen["call:"+source.CallID] {
			return fmt.Errorf("native fetch requires distinct calls from the exact child and attributed turns")
		}
		seen["call:"+source.CallID] = true
		for _, ref := range []codexNativeContextEventRef{source.Call, source.Command, source.Result} {
			digest, err := canonicalCodexNativeEventHash(ref.ID, ref.SHA256)
			if err != nil || digest != ref.SHA256 || seen["event:"+ref.ID] {
				return fmt.Errorf("native fetch requires unique original event identities and canonical raw hashes")
			}
			seen["event:"+ref.ID] = true
		}
		started, err := time.Parse(time.RFC3339Nano, source.StartedAt)
		if err != nil || started.UTC().Format(time.RFC3339Nano) != source.StartedAt {
			return fmt.Errorf("native fetch has an invalid call timestamp")
		}
		completed, err := time.Parse(time.RFC3339Nano, source.CompletedAt)
		if err != nil || completed.UTC().Format(time.RFC3339Nano) != source.CompletedAt || !completed.After(started) {
			return fmt.Errorf("native fetch requires a complete successful command interval")
		}
		if i == 0 {
			readCompleted = completed
		} else if !started.After(readCompleted) {
			return fmt.Errorf("native context ACK must start after the complete read")
		}
	}
	return nil
}

func observeCodexNativeContextFetch(record *buildAttemptRecord, worker *buildAttemptWorkerRun, dispatch codexBuildDispatch, request codexNativeWorkerRequest, now string) (*codexNativeWorkerReceipt, error) {
	if _, _, err := validateCodexNativeContextTarget(*record, request); err != nil {
		return nil, err
	}
	delivery, fetch := request.ContextDelivery, request.ContextFetch
	if worker.Native.ContextProtocol != codexNativeContextProtocolChildFetch || delivery == nil || fetch == nil || request.Result != nil || request.ContextSend != nil || request.ContextDeliveryID != "" || request.ContextPurpose != "" || request.ContextPayloadSHA256 != "" || request.ContextDecisionIDs != nil {
		return nil, fmt.Errorf("native fetched observation requires only the exact envelope and separate read/ACK sources")
	}
	if err := validateCodexNativeContextEnvelope(*delivery, *record, *worker); err != nil {
		return nil, err
	}
	if err := validateCodexNativeContextFetch(*fetch, *delivery); err != nil {
		return nil, err
	}
	if request.SourceEventID != fetch.Ack.Result.ID || request.SourceEventSHA256 != fetch.Ack.Result.SHA256 || request.ObservedAt != fetch.Ack.CompletedAt {
		return nil, fmt.Errorf("native fetched observation must identify the exact ACK result source")
	}
	observation := codexNativeHostObservation{SchemaVersion: 1, Status: "context_fetched", ChildID: request.ChildID, ObservedAt: request.ObservedAt, Detail: request.ObservationDetail, SourceEventID: request.SourceEventID, SourceEventSHA256: request.SourceEventSHA256, ContextDeliveryID: delivery.DeliveryID}
	receipt := codexNativeTransitionReceipt("observe", *worker)
	receipt.At, receipt.SourceEventID, receipt.SourceEventSHA256 = observation.ObservedAt, observation.SourceEventID, observation.SourceEventSHA256
	receipt.ContextDeliveryID, receipt.ContextPayloadSHA256 = delivery.DeliveryID, delivery.PayloadSHA256
	incoming := codexNativeContextReceipt{Delivery: *delivery, Observation: observation, Fetch: fetch}
	for _, saved := range worker.Native.ContextDeliveries {
		if saved.Delivery.DeliveryID == delivery.DeliveryID {
			if err := validateCodexNativeContextReceipt(saved, *record, *worker); err != nil {
				return nil, err
			}
			want, _ := jsonSHA256(saved)
			got, _ := jsonSHA256(incoming)
			if want != got {
				return nil, fmt.Errorf("native fetched acknowledgement conflicts with saved sources")
			}
			return &receipt, errCodexNativeReplay
		}
	}
	if err := validateCodexNativeContextLive(*record, *worker); err != nil {
		return nil, err
	}
	if delivery.Purpose == "answers" {
		initial := false
		readAt, _ := time.Parse(time.RFC3339Nano, fetch.Read.StartedAt)
		for _, saved := range worker.Native.ContextDeliveries {
			if saved.Delivery.Purpose == "initial" && saved.Fetch != nil {
				ackAt, _ := time.Parse(time.RFC3339Nano, saved.Fetch.Ack.CompletedAt)
				initial = readAt.After(ackAt)
			}
		}
		if !initial {
			return nil, fmt.Errorf("answer read must follow the initial context ACK")
		}
	}
	bound, err := time.Parse(time.RFC3339Nano, worker.Native.BoundAt)
	started, _ := time.Parse(time.RFC3339Nano, fetch.Read.StartedAt)
	if err != nil || !started.After(bound) {
		return nil, fmt.Errorf("native context read must follow the saved child binding")
	}
	// A delayed parent may save an ACK after a later running observation. Its
	// source time remains the actual ACK time; lifecycle status ignores it.
	for _, saved := range worker.Native.Observations {
		for _, source := range []codexNativeContextSource{fetch.Read, fetch.Ack} {
			for _, ref := range []codexNativeContextEventRef{source.Call, source.Command, source.Result} {
				if ref.ID == saved.SourceEventID {
					return nil, fmt.Errorf("native context source already belongs to another observation")
				}
			}
		}
	}
	for _, saved := range worker.Native.ContextDeliveries {
		if saved.Fetch == nil {
			return nil, fmt.Errorf("mixed legacy and child-fetch receipts")
		}
		priorACK, _ := time.Parse(time.RFC3339Nano, saved.Fetch.Ack.CompletedAt)
		if !started.After(priorACK) {
			return nil, fmt.Errorf("native context read must follow every previous context ACK")
		}
		for _, old := range []codexNativeContextSource{saved.Fetch.Read, saved.Fetch.Ack} {
			for _, fresh := range []codexNativeContextSource{fetch.Read, fetch.Ack} {
				if old.CallID == fresh.CallID {
					return nil, fmt.Errorf("native context source call was reused")
				}
				for _, a := range []codexNativeContextEventRef{old.Call, old.Command, old.Result} {
					for _, b := range []codexNativeContextEventRef{fresh.Call, fresh.Command, fresh.Result} {
						if a.ID == b.ID {
							return nil, fmt.Errorf("native context source event was reused")
						}
					}
				}
			}
		}
	}
	expected, err := composeCodexNativeContextForPurpose(*record, dispatch, *worker, delivery.Purpose, delivery.DecisionIDs)
	if err != nil {
		return nil, err
	}
	if expected == nil || expected.DeliveryID != delivery.DeliveryID {
		return nil, fmt.Errorf("native fetched context is stale or differs from the runtime payload")
	}
	worker.Native.ContextDeliveryIDs = append(worker.Native.ContextDeliveryIDs, delivery.DeliveryID)
	worker.Native.ContextDeliveries = append(worker.Native.ContextDeliveries, incoming)
	worker.Native.Observations = append(worker.Native.Observations, observation)
	worker.UpdatedAt, record.UpdatedAt = now, now
	return &receipt, nil
}

func codexNativeContextResultRequiresReceipts(result *internalWorkerResult) bool {
	if result == nil {
		return false
	}
	if result.Status == buildWorkerCompleted || result.Status == "code_written" {
		return true
	}
	for _, receipt := range result.TaskReceipts {
		if receipt.Status == buildWorkerCompleted || receipt.Status == "code_written" {
			return true
		}
	}
	return false
}

func validateCodexNativeContextClosure(record buildAttemptRecord, worker buildAttemptWorkerRun, currentQuestions bool) error {
	if worker.Native.ContextProtocol == "" {
		return nil
	}
	if worker.Native.ContextProtocol != codexNativeContextProtocolChildFetch {
		return fmt.Errorf("unsupported native context protocol")
	}
	initial := 0
	delivered := make(map[string]bool)
	for _, saved := range worker.Native.ContextDeliveries {
		if err := validateCodexNativeContextReceipt(saved, record, worker); err != nil {
			return err
		}
		if saved.Delivery.Purpose == "initial" {
			initial++
		}
		for _, id := range saved.Delivery.DecisionIDs {
			delivered[id] = true
		}
	}
	if initial != 1 {
		return fmt.Errorf("native completion requires one observed full initial context read and separate ACK")
	}
	if !currentQuestions {
		return nil
	}
	if err := validateCodexNativeContextLive(record, worker); err != nil {
		return err
	}
	var dispatch *codexBuildDispatch
	for i := range record.PlanManifest.Dispatches {
		d := &record.PlanManifest.Dispatches[i]
		if d.Name == worker.WorkerName && normalizedDispatchTaskID(*d) == worker.TaskID {
			dispatch = d
			break
		}
	}
	if dispatch == nil {
		return fmt.Errorf("native context closure has no accepted dispatch")
	}
	for _, saved := range worker.Native.ContextDeliveries {
		if saved.Delivery.Purpose != "answers" {
			continue
		}
		current := renderCodexNativeContextAnswers(*record.PlanManifest, *dispatch, worker.ProviderRunID, worker.Native.ChildID, worker.Native.ContextDecisionIDs, saved.Delivery.DecisionIDs)
		if !equalCodexNativeContextIDs(current.DecisionIDs, saved.Delivery.DecisionIDs) || codexNativeAnswerSection(current) != saved.Delivery.Payload {
			return fmt.Errorf("native fetched answer changed before terminal admission")
		}
	}
	var file PendingDecisionFile
	if err := store.LoadJSON(pendingDecisionsFile, &file); err != nil {
		// Storage preserves ErrNotExist through wrapped errors. A fresh job
		// can have no decisions file; corruption and other failures still refuse.
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, decision := range file.Decisions {
		b := decision.NativeBinding
		if b == nil || b.AttemptID != record.ID || b.LaunchID != worker.ProviderRunID || b.WorkerName != worker.WorkerName {
			continue
		}
		question := codexNativeQuestion{QuestionID: b.QuestionID, Question: decision.Description}
		if err := validateCodexNativeDecisionRow(decision, nativeDecisionBinding(record, worker, question), decision.Description); err != nil {
			return err
		}
		if !decision.Resolved || decision.Resolution == "" || !delivered[decision.ID] {
			return fmt.Errorf("native completion requires the exact resolved question answer's observed read and ACK")
		}
	}
	return nil
}

// The terminal seals this receipt set before becoming immutable. Replay and
// recovery check that set without consulting today's mutable question state.
func validateCodexNativeSavedContext(record buildAttemptRecord, worker buildAttemptWorkerRun) error {
	n := worker.Native
	if record.PlanManifest == nil || n.ContextProtocol != record.PlanManifest.ContextProtocol {
		return fmt.Errorf("saved native context protocol differs from accepted manifest")
	}
	if n.ContextProtocol == "" {
		if n.ContextReceiptSHA256 != "" {
			return fmt.Errorf("legacy native worker cannot carry a child-fetch closure")
		}
		for _, saved := range n.ContextDeliveries {
			if saved.Fetch != nil || saved.Delivery.Protocol != "" || saved.Delivery.Purpose != "" {
				return fmt.Errorf("legacy native receipt cannot carry a child-fetch protocol")
			}
		}
		return nil
	}
	if n.ContextProtocol != codexNativeContextProtocolChildFetch {
		return fmt.Errorf("unsupported saved native context protocol")
	}
	if len(n.ContextDeliveryIDs) != len(n.ContextDeliveries) {
		return fmt.Errorf("native context receipt index differs from saved deliveries")
	}
	seenSources, seenAnswers := make(map[string]bool), make(map[string]bool)
	var initialACK, previousACK time.Time
	for _, saved := range n.ContextDeliveries {
		if err := validateCodexNativeContextReceipt(saved, record, worker); err != nil {
			return err
		}
		readAt, _ := time.Parse(time.RFC3339Nano, saved.Fetch.Read.StartedAt)
		if !previousACK.IsZero() && !readAt.After(previousACK) {
			return fmt.Errorf("saved context delivery source chronology changed")
		}
		previousACK, _ = time.Parse(time.RFC3339Nano, saved.Fetch.Ack.CompletedAt)
		if saved.Delivery.Purpose == "initial" {
			if !initialACK.IsZero() {
				return fmt.Errorf("duplicate initial child-fetch receipt")
			}
			initialACK, _ = time.Parse(time.RFC3339Nano, saved.Fetch.Ack.CompletedAt)
		} else {
			readAt, _ := time.Parse(time.RFC3339Nano, saved.Fetch.Read.StartedAt)
			if initialACK.IsZero() || !readAt.After(initialACK) {
				return fmt.Errorf("saved answer read precedes initial context ACK")
			}
		}
		for _, id := range saved.Delivery.DecisionIDs {
			if seenAnswers[id] {
				return fmt.Errorf("native answer appears in multiple fetch receipts")
			}
			seenAnswers[id] = true
		}
		for _, source := range []codexNativeContextSource{saved.Fetch.Read, saved.Fetch.Ack} {
			if seenSources["call:"+source.CallID] {
				return fmt.Errorf("native fetch call appears in multiple receipts")
			}
			seenSources["call:"+source.CallID] = true
			for _, ref := range []codexNativeContextEventRef{source.Call, source.Command, source.Result} {
				if seenSources["event:"+ref.ID] || (n.SourceEventID != "" && ref.ID == n.SourceEventID) {
					return fmt.Errorf("native fetch source event was reused")
				}
				seenSources["event:"+ref.ID] = true
			}
		}
	}
	if codexNativeContextResultRequiresReceipts(worker.Result) {
		if err := validateCodexNativeContextClosure(record, worker, false); err != nil {
			return err
		}
		digest, err := jsonSHA256(n.ContextDeliveries)
		if err != nil || digest != n.ContextReceiptSHA256 {
			return fmt.Errorf("native context terminal receipt set no longer matches its saved digest")
		}
	} else if n.ContextReceiptSHA256 != "" {
		return fmt.Errorf("native context closure requires positive terminal evidence")
	}
	return nil
}
