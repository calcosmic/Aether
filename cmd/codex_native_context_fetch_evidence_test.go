package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/calcosmic/Aether/pkg/codex"
)

// Source shape: retained child-output-mapping-20260918T213544Z-rxnypc3i/result.json,
// sha256:8b503c727a24cbb6fa7bfd4392447fb60d31db19798a993e7a5746b9c288f009.
// That mapping established plaintext projection, not context delivery success.
// A fetch qualifies only when all six original child records and the complete
// candidate responses independently satisfy the checks below.
type nativeChildContextCommandExpectation struct {
	ParentID, ChildID, TaskPath, Workspace, Command string
}

type nativeChildContextCommandEvidence struct {
	Source                 codexNativeContextSource `json:"source"`
	Call                   json.RawMessage          `json:"call"`
	Command                json.RawMessage          `json:"command"`
	Result                 json.RawMessage          `json:"result"`
	Stdout                 string                   `json:"stdout"`
	callIndex, resultIndex int
}

type nativeChildContextFetchExpectation struct {
	nativeChildContextCommandExpectation
	Candidate, RequestPath, Purpose, NotBefore string
	Delivery                                   codexNativeContextDelivery
}

type nativeChildContextFetchEvidence struct {
	RequestPath string                            `json:"request_path"`
	Delivery    codexNativeContextDelivery        `json:"delivery"`
	Ack         codexNativeContextAck             `json:"ack"`
	Fetch       codexNativeContextFetch           `json:"fetch"`
	Read        nativeChildContextCommandEvidence `json:"read"`
	ACK         nativeChildContextCommandEvidence `json:"ack_source"`
}

type nativeContextSourceRecord struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Payload   struct {
		Type, ID, Name, Namespace, Arguments, Input, Status string
		CallID                                              string   `json:"call_id"`
		ParentThreadID                                      string   `json:"parent_thread_id"`
		ThreadID                                            string   `json:"thread_id"`
		TurnID                                              string   `json:"turn_id"`
		AgentPath                                           string   `json:"agent_path"`
		ThreadSource                                        string   `json:"thread_source"`
		Cwd                                                 string   `json:"cwd"`
		RuntimeWorkspaceRoots                               []string `json:"runtime_workspace_roots"`
		Source                                              struct {
			Subagent struct {
				ThreadSpawn struct {
					ParentThreadID string `json:"parent_thread_id"`
					AgentPath      string `json:"agent_path"`
				} `json:"thread_spawn"`
			} `json:"subagent"`
		} `json:"source"`
		Metadata struct {
			TurnID string `json:"turn_id"`
		} `json:"internal_chat_message_metadata_passthrough"`
		Output json.RawMessage `json:"output"`
		Item   struct {
			Type, ID, Status, Cwd, Stdout, Stderr, Phase string
			Command                                      []string
			ExitCode                                     *int   `json:"exit_code"`
			AggregatedOutput                             string `json:"aggregated_output"`
			FormattedOutput                              string `json:"formatted_output"`
		} `json:"item"`
	} `json:"payload"`
	Metadata struct {
		ClientAuthored *bool `json:"client_authored"`
	} `json:"metadata"`
}

// Reject ambiguous duplicate JSON members before decoding typed source or Go
// envelopes; ordinary json.Unmarshal would silently take the last value.
func nativeContextUnambiguousJSON(raw []byte) error {
	if !utf8.Valid(raw) {
		return fmt.Errorf("invalid UTF-8 source")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	var value func() error
	value = func() error {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		delim, composite := tok.(json.Delim)
		if !composite {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				key, err := d.Token()
				if err != nil {
					return err
				}
				name, ok := key.(string)
				if !ok || seen[strings.ToLower(name)] {
					return fmt.Errorf("duplicate or invalid JSON member")
				}
				seen[strings.ToLower(name)] = true
				if err := value(); err != nil {
					return err
				}
			}
		case '[':
			for d.More() {
				if err := value(); err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("unexpected JSON delimiter")
		}
		_, err = d.Token()
		return err
	}
	if err := value(); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return fmt.Errorf("trailing or incomplete JSON")
	}
	return nil
}

func nativeContextRequiredKeys(raw []byte, paths ...string) error {
	for _, path := range paths {
		part := json.RawMessage(raw)
		for _, key := range strings.Split(path, ".") {
			var object map[string]json.RawMessage
			if json.Unmarshal(part, &object) != nil {
				return fmt.Errorf("missing actual source field %s", path)
			}
			value, ok := object[key]
			if !ok || bytes.Equal(value, []byte("null")) {
				return fmt.Errorf("missing actual source field %s", path)
			}
			part = value
		}
	}
	return nil
}

func nativeContextExactCwd(got, want string) bool {
	if strings.HasPrefix(got, "file://") {
		u, err := url.Parse(got)
		if err != nil || u.Host != "" || u.RawQuery != "" || u.Fragment != "" {
			return false
		}
		got = u.Path
	}
	return got == want
}

func nativeContextCommand(input, cwd string) (nativeRecordedShellCommand, bool, bool) {
	if command, ok := nativeCodeModeOutputProjection(input, cwd); ok {
		// The shared operation recognizer admits several presentations. Context
		// evidence must distinguish exact stdout from the untouched full result
		// so its exit/session/output fields are still checked independently.
		pattern := regexp.MustCompile(`^\s*const\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*await\s+tools\.exec_command\((\{[\s\S]*\})\);\s*text\(([\s\S]*)\);?\s*$`)
		match := pattern.FindStringSubmatch(input)
		if len(match) == 4 {
			switch match[3] {
			case match[1] + ".output":
				return command, true, true
			case "JSON.stringify(" + match[1] + ")":
				return command, false, true
			}
		}
		// Extra EXIT_CODE annotations are outside the exact context-output
		// contract, even though ordinary operation attribution can classify them.
		return nativeRecordedShellCommand{}, false, false
	}
	// The same literal exec result may be printed untouched as an object. A
	// single command only: never accept batches, computed output or rewriting.
	if strings.Contains(input, "Promise.allSettled") {
		return nativeRecordedShellCommand{}, false, false
	}
	commands, ok := nativeCodeModeCommands(input, cwd)
	if !ok || len(commands) != 1 {
		return nativeRecordedShellCommand{}, false, false
	}
	return commands[0], false, true
}

func nativeContextSourceRecords(raw []byte) ([]nativeContextSourceRecord, []json.RawMessage, error) {
	var records []nativeContextSourceRecord
	var lines []json.RawMessage
	for _, line := range bytes.Split(raw, []byte{'\n'}) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var record nativeContextSourceRecord
		if err := nativeContextUnambiguousJSON(line); err != nil {
			return nil, nil, err
		}
		if err := json.Unmarshal(line, &record); err != nil {
			return nil, nil, err
		}
		records = append(records, record)
		lines = append(lines, append(json.RawMessage(nil), line...))
	}
	return records, lines, nil
}

func nativeDecodeChildContextCommand(raw []byte, want nativeChildContextCommandExpectation) (nativeChildContextCommandEvidence, error) {
	var evidence nativeChildContextCommandEvidence
	fail := func(reason string) (nativeChildContextCommandEvidence, error) {
		return evidence, fmt.Errorf("child context source: %s", reason)
	}
	if want.ParentID == "" || want.ChildID == "" || want.ParentID == want.ChildID || !strings.HasPrefix(want.TaskPath, "/root/") || !filepath.IsAbs(want.Workspace) {
		return fail("incomplete child identity")
	}
	wantWords, ok := nativeSimpleShellWords(want.Command)
	if !ok {
		return fail("nonliteral expected command")
	}
	records, lines, err := nativeContextSourceRecords(raw)
	if err != nil {
		return fail(err.Error())
	}
	metadata := 0
	callCounts, resultCounts, recordIDs := map[string]int{}, map[string]int{}, map[string]int{}
	for index, record := range records {
		p := record.Payload
		if record.Type == "session_meta" {
			metadata++
			if err := nativeContextRequiredKeys(lines[index], "type", "payload.id", "payload.parent_thread_id", "payload.thread_source", "payload.agent_path", "payload.cwd", "payload.runtime_workspace_roots", "payload.source.subagent.thread_spawn.parent_thread_id", "payload.source.subagent.thread_spawn.agent_path"); err != nil {
				return fail(err.Error())
			}
			spawn := p.Source.Subagent.ThreadSpawn
			if index != 0 || p.ID != want.ChildID || p.ParentThreadID != want.ParentID || p.ThreadSource != "subagent" || p.AgentPath != want.TaskPath || spawn.ParentThreadID != want.ParentID || spawn.AgentPath != want.TaskPath || !nativeContextExactCwd(p.Cwd, want.Workspace) || len(p.RuntimeWorkspaceRoots) != 1 || !nativeContextExactCwd(p.RuntimeWorkspaceRoots[0], want.Workspace) {
				return fail("session metadata is not the exact child workspace and parent")
			}
		}
		if record.Type == "response_item" {
			switch p.Type {
			case "custom_tool_call", "function_call":
				callCounts[p.CallID]++
				recordIDs[p.ID]++
			case "custom_tool_call_output", "function_call_output":
				resultCounts[p.CallID]++
				recordIDs[p.ID]++
			}
		}
		if record.Type == "event_msg" && p.Type == "item_completed" && p.Item.Type == "CommandExecution" {
			recordIDs[p.Item.ID]++
		}
	}
	if metadata != 1 {
		return fail("unique actual child metadata missing")
	}
	callIndex := -1
	var command nativeRecordedShellCommand
	projected := false
	for i, record := range records {
		p := record.Payload
		if record.Type != "response_item" || p.Type != "custom_tool_call" || (p.Name != "exec" && p.Name != "functions.exec") {
			continue
		}
		parsed, projection, valid := nativeContextCommand(p.Input, want.Workspace)
		words, literal := nativeSimpleShellWords(parsed.Command)
		if !valid || !literal || !reflect.DeepEqual(words, wantWords) {
			continue
		}
		if callIndex != -1 {
			return fail("duplicate candidate command")
		}
		callIndex, command, projected = i, parsed, projection
	}
	if callIndex < 0 {
		return fail("literal candidate output projection missing")
	}
	call := records[callIndex]
	cp := call.Payload
	if err := nativeContextRequiredKeys(lines[callIndex], "type", "timestamp", "payload.type", "payload.id", "payload.call_id", "payload.name", "payload.input", "payload.status", "payload.internal_chat_message_metadata_passthrough.turn_id"); err != nil {
		return fail(err.Error())
	}
	if cp.ID == "" || cp.CallID == "" || cp.Metadata.TurnID == "" || cp.Status != "completed" || callCounts[cp.CallID] != 1 || resultCounts[cp.CallID] != 1 || recordIDs[cp.ID] != 1 || !nativeContextExactCwd(command.Cwd, want.Workspace) {
		return fail("call identity, turn or cwd is missing or ambiguous")
	}
	resultIndex := -1
	for i, record := range records {
		if record.Type == "response_item" && record.Payload.Type == "custom_tool_call_output" && record.Payload.CallID == cp.CallID {
			resultIndex = i
		}
	}
	if resultIndex <= callIndex {
		return fail("result precedes its call")
	}
	commandIndex := -1
	for i := callIndex + 1; i < resultIndex; i++ {
		record := records[i]
		p := record.Payload
		if record.Type == "response_item" && (p.Type == "custom_tool_call" || p.Type == "function_call") {
			return fail("intervening tool call")
		}
		if record.Type != "event_msg" || p.Type != "item_completed" || p.Item.Type != "CommandExecution" {
			continue
		}
		if commandIndex != -1 {
			return fail("multiple executions for one projected result")
		}
		commandIndex = i
	}
	if commandIndex < 0 {
		return fail("actual completed CommandExecution missing")
	}
	execRecord, result := records[commandIndex], records[resultIndex]
	if err := nativeContextRequiredKeys(lines[commandIndex], "type", "timestamp", "payload.type", "payload.thread_id", "payload.turn_id", "payload.item.type", "payload.item.id", "payload.item.command", "payload.item.cwd", "payload.item.status", "payload.item.exit_code", "payload.item.stdout", "payload.item.stderr", "payload.item.aggregated_output", "payload.item.formatted_output"); err != nil {
		return fail(err.Error())
	}
	if err := nativeContextRequiredKeys(lines[resultIndex], "type", "timestamp", "payload.type", "payload.id", "payload.call_id", "payload.output", "payload.internal_chat_message_metadata_passthrough.turn_id", "metadata.client_authored"); err != nil {
		return fail(err.Error())
	}
	ep, rp := execRecord.Payload, result.Payload
	item := ep.Item
	if ep.ThreadID != want.ChildID || ep.TurnID != cp.Metadata.TurnID || rp.Metadata.TurnID != cp.Metadata.TurnID || item.ID == "" || rp.ID == "" || recordIDs[item.ID] != 1 || recordIDs[rp.ID] != 1 || cp.ID == item.ID || cp.ID == rp.ID || item.ID == rp.ID {
		return fail("call/execution/result child, turn or actual IDs differ")
	}
	if item.Status != "completed" || item.ExitCode == nil || *item.ExitCode != 0 || len(item.Command) != 3 || (item.Command[0] != "/bin/zsh" && item.Command[0] != "/bin/bash" && item.Command[0] != "/bin/sh") || (item.Command[1] != "-lc" && item.Command[1] != "-c") || item.Command[2] != command.Command || !nativeContextExactCwd(item.Cwd, want.Workspace) {
		return fail("execution did not run the exact successful literal command in the child workspace")
	}
	if result.Metadata.ClientAuthored == nil || *result.Metadata.ClientAuthored {
		return fail("result is not the retained host-authored output")
	}
	var output []struct{ Type, Text string }
	if json.Unmarshal(rp.Output, &output) != nil || len(output) != 2 || output[0].Type != "input_text" || output[1].Type != "input_text" || !regexp.MustCompile(`^Script completed\nWall time [0-9]+(?:\.[0-9]+)? seconds\nOutput:\n$`).MatchString(output[0].Text) || item.Stderr != "" || item.Stdout == "" || item.Stdout != item.AggregatedOutput || item.Stdout != item.FormattedOutput {
		return fail("full stdout does not match the completed projection exactly")
	}
	returned := output[1].Text
	if !projected {
		var result struct {
			Exit    *int   `json:"exit_code"`
			Session *int   `json:"session_id"`
			Output  string `json:"output"`
		}
		if nativeContextUnambiguousJSON([]byte(returned)) != nil || json.Unmarshal([]byte(returned), &result) != nil || result.Exit == nil || *result.Exit != 0 || result.Session != nil {
			return fail("untouched exec result is incomplete or unsuccessful")
		}
		returned = result.Output
	}
	if returned != item.Stdout {
		return fail("returned output differs from actual command stdout")
	}
	start, a := time.Parse(time.RFC3339Nano, call.Timestamp)
	middle, b := time.Parse(time.RFC3339Nano, execRecord.Timestamp)
	end, c := time.Parse(time.RFC3339Nano, result.Timestamp)
	if a != nil || b != nil || c != nil || middle.Before(start) || end.Before(middle) {
		return fail("source timestamps are missing or reversed")
	}
	evidence = nativeChildContextCommandEvidence{Source: codexNativeContextSource{ChildID: want.ChildID, TurnID: cp.Metadata.TurnID, CallID: cp.CallID, Call: codexNativeContextEventRef{ID: cp.ID, SHA256: strings.TrimPrefix(lifecycleDigest(lines[callIndex]), "sha256:")}, Command: codexNativeContextEventRef{ID: item.ID, SHA256: strings.TrimPrefix(lifecycleDigest(lines[commandIndex]), "sha256:")}, Result: codexNativeContextEventRef{ID: rp.ID, SHA256: strings.TrimPrefix(lifecycleDigest(lines[resultIndex]), "sha256:")}, StartedAt: start.UTC().Format(time.RFC3339Nano), CompletedAt: end.UTC().Format(time.RFC3339Nano)}, Call: lines[callIndex], Command: lines[commandIndex], Result: lines[resultIndex], Stdout: item.Stdout, callIndex: callIndex, resultIndex: resultIndex}
	return evidence, nil
}

func nativeContextLiteralCommand(words []string) string {
	quoted := make([]string, len(words))
	for i, word := range words {
		quoted[i] = "'" + strings.ReplaceAll(word, "'", "'\"'\"'") + "'"
	}
	return strings.Join(quoted, " ")
}

func nativeContextResponse(raw string) (codexNativeWorkerResponse, error) {
	var envelope struct {
		OK     bool                      `json:"ok"`
		Result codexNativeWorkerResponse `json:"result"`
	}
	if err := nativeContextUnambiguousJSON([]byte(raw)); err != nil {
		return envelope.Result, err
	}
	d := json.NewDecoder(strings.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&envelope); err != nil {
		return envelope.Result, err
	}
	if !envelope.OK || envelope.Result.SchemaVersion != 1 || envelope.Result.Disposition != "" || envelope.Result.LaunchAllowed || envelope.Result.Replay || envelope.Result.Complete {
		return envelope.Result, fmt.Errorf("unexpected context response envelope")
	}
	encoded, err := json.Marshal(envelope.Result)
	if err != nil || raw != "{\"ok\":true,\"result\":"+string(encoded)+"}\n" {
		return envelope.Result, fmt.Errorf("returned JSON is not the complete unmodified Go outputOK envelope")
	}
	return envelope.Result, nil
}

func nativeDecodeChildContextFetch(raw []byte, want nativeChildContextFetchExpectation) (nativeChildContextFetchEvidence, error) {
	var evidence nativeChildContextFetchEvidence
	if !filepath.IsAbs(want.Candidate) || !filepath.IsAbs(want.RequestPath) || (want.Purpose != "initial" && want.Purpose != "answers") || want.Delivery.SchemaVersion != 2 || want.Delivery.Protocol != codexNativeContextProtocolChildFetch || want.Delivery.Purpose != want.Purpose || want.Delivery.ChildID != want.ChildID || want.Delivery.HostSessionID != want.ParentID || want.Delivery.Workspace != want.Workspace || want.Delivery.Payload == "" || want.Delivery.PayloadSHA256 != lifecycleDigest([]byte(want.Delivery.Payload)) {
		return evidence, fmt.Errorf("incomplete or mismatched expected child-fetch envelope")
	}
	id := want.Delivery.DeliveryID
	identity := want.Delivery
	identity.DeliveryID = ""
	digest, err := jsonSHA256(identity)
	if err != nil || id == "" || id != digest {
		return evidence, fmt.Errorf("expected delivery identity changed")
	}
	base := []string{want.Candidate, "codex-native-worker", "context", "--request", want.RequestPath}
	commandWant := want.nativeChildContextCommandExpectation
	commandWant.Command = nativeContextLiteralCommand(base)
	read, err := nativeDecodeChildContextCommand(raw, commandWant)
	if err != nil {
		return evidence, err
	}
	response, err := nativeContextResponse(read.Stdout)
	if err != nil {
		return evidence, err
	}
	if response.ContextStatus != "awaiting_ack" || response.ContextDelivery == nil || response.ContextAck != nil || response.ExecutionBinding != want.Delivery.ExecutionBinding || !reflect.DeepEqual(*response.ContextDelivery, want.Delivery) {
		return evidence, fmt.Errorf("actual full context JSON does not match the Go-issued delivery")
	}
	delivery := *response.ContextDelivery
	base[2] = "context-ack"
	base = append(base, "--delivery-id", delivery.DeliveryID, "--payload-sha256", delivery.PayloadSHA256)
	for _, decision := range delivery.DecisionIDs {
		base = append(base, "--decision-id", decision)
	}
	commandWant.Command = nativeContextLiteralCommand(base)
	ack, err := nativeDecodeChildContextCommand(raw, commandWant)
	if err != nil {
		return evidence, err
	}
	ackResponse, err := nativeContextResponse(ack.Stdout)
	if err != nil {
		return evidence, err
	}
	expectedAck := codexNativeContextAck{SchemaVersion: 1, DeliveryID: delivery.DeliveryID, PayloadSHA256: delivery.PayloadSHA256, ChildID: delivery.ChildID, DecisionIDs: delivery.DecisionIDs}
	if ackResponse.ContextStatus != "ack_validated" || ackResponse.ContextDelivery != nil || ackResponse.ContextAck == nil || ackResponse.ExecutionBinding != delivery.ExecutionBinding || !reflect.DeepEqual(*ackResponse.ContextAck, expectedAck) {
		return evidence, fmt.Errorf("actual ACK response differs from the child's complete read")
	}
	readTime, _ := time.Parse(time.RFC3339Nano, read.Source.CompletedAt)
	ackTime, _ := time.Parse(time.RFC3339Nano, ack.Source.StartedAt)
	if read.resultIndex >= ack.callIndex || !ackTime.After(readTime) {
		return evidence, fmt.Errorf("ACK must be a separate command after the successful read")
	}
	if want.NotBefore != "" {
		bound, err := time.Parse(time.RFC3339Nano, want.NotBefore)
		start, _ := time.Parse(time.RFC3339Nano, read.Source.StartedAt)
		if err != nil || !start.After(bound) {
			return evidence, fmt.Errorf("context read predates its authorization")
		}
	}
	fetch := codexNativeContextFetch{SchemaVersion: 1, Read: read.Source, Ack: ack.Source}
	if err := validateCodexNativeContextFetch(fetch, delivery); err != nil {
		return evidence, err
	}
	return nativeChildContextFetchEvidence{RequestPath: want.RequestPath, Delivery: delivery, Ack: expectedAck, Fetch: fetch, Read: read, ACK: ack}, nil
}

func nativeChildContextBootstrapBoundary(parent []byte, r codexNativeLiveReceipt) (nativeGapBoundary, error) {
	var boundary nativeGapBoundary
	message, matches := "", 0
	if r.ChildSpawnCallID == "" || r.BoundHostSessionID == "" {
		return boundary, fmt.Errorf("actual bootstrap spawn identity missing")
	}
	for _, line := range bytes.Split(parent, []byte{'\n'}) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		if err := nativeContextUnambiguousJSON(line); err != nil {
			return boundary, err
		}
		var call struct {
			Type    string
			Payload struct {
				Type, Name, Arguments string
				CallID                string `json:"call_id"`
			}
		}
		if json.Unmarshal(line, &call) != nil || call.Type != "response_item" || call.Payload.Type != "function_call" || call.Payload.CallID != r.ChildSpawnCallID {
			continue
		}
		var args struct {
			Message   string
			ForkTurns string `json:"fork_turns"`
		}
		if nativeContextUnambiguousJSON([]byte(call.Payload.Arguments)) != nil || json.Unmarshal([]byte(call.Payload.Arguments), &args) != nil || args.ForkTurns != "none" || args.Message == "" {
			return boundary, fmt.Errorf("child bootstrap did not explicitly disable inherited context")
		}
		message = args.Message
		matches++
	}
	if matches != 1 {
		return boundary, fmt.Errorf("unique actual bootstrap call missing")
	}
	// The bootstrap may be encrypted; its payload is not claimed as prompt proof.
	// The independent child fetch proves the full prompt. Here only the actual
	// successful spawn's explicit fork mode and child linkage are established.
	return nativeGapHostBoundary(parent, r.BoundHostSessionID, r.ChildID, r.ChildTaskPath, "spawn_agent", message)
}

func nativeChildContextParent(r codexNativeLiveReceipt) ([]byte, string, error) {
	if r.BoundHostSessionID == "" {
		return nil, "", fmt.Errorf("actual child bound parent missing")
	}
	paths, err := filepath.Glob(filepath.Join(filepath.Dir(r.FixtureRoot), "home", ".codex", "sessions", "*", "*", "*", "*"+r.BoundHostSessionID+".jsonl"))
	if err != nil || len(paths) != 1 {
		return nil, "", fmt.Errorf("unique parent capture missing")
	}
	raw, err := r.readEvidence(paths[0])
	return raw, paths[0], err
}

// Every request is the retained binding plus its purpose, never a parent-authored
// delivery or acknowledgement. Both child commands use this exact immutable file.
func nativeChildContextRequest(r codexNativeLiveReceipt, raw []byte, saved codexNativeContextReceipt, attempt buildAttemptRecord) (string, error) {
	records, _, err := nativeContextSourceRecords(raw)
	if err != nil || saved.Fetch == nil {
		return "", fmt.Errorf("child-fetch request source missing")
	}
	var path string
	for _, record := range records {
		if record.Type != "response_item" || record.Payload.Type != "custom_tool_call" || record.Payload.ID != saved.Fetch.Read.Call.ID {
			continue
		}
		command, _, ok := nativeContextCommand(record.Payload.Input, r.FixtureRoot)
		words, literal := nativeSimpleShellWords(command.Command)
		if !ok || !literal || len(words) != 5 || words[0] != r.CandidatePath || words[1] != "codex-native-worker" || words[2] != "context" || words[3] != "--request" || path != "" {
			return "", fmt.Errorf("child-fetch request command differs from retained delivery")
		}
		path = words[4]
	}
	pattern := regexp.MustCompile(`^(w[0-9]+-)?context-(initial|answers)-request\.json$`)
	match := pattern.FindStringSubmatch(filepath.Base(path))
	if !filepath.IsAbs(path) || len(match) != 3 || match[2] != saved.Delivery.Purpose {
		return "", fmt.Errorf("child-fetch request path is not the purpose-bound pointer")
	}
	retained := filepath.Join(filepath.Dir(r.FixtureRoot), "coordination")
	requestRaw, err := r.readEvidence(filepath.Join(retained, filepath.Base(path)))
	if err != nil {
		return "", err
	}
	pointerRaw, err := r.readEvidence(filepath.Join(retained, match[1]+"context-"+match[2]+"-pointer.json"))
	if err != nil {
		return "", err
	}
	var pointer struct {
		SchemaVersion     int `json:"schema_version"`
		Protocol, Purpose string
		ChildID           string         `json:"child_id"`
		CandidatePath     string         `json:"candidate_path"`
		CandidateSHA256   string         `json:"candidate_sha256"`
		RequestPath       string         `json:"request_path"`
		RequestSHA256     string         `json:"request_sha256"`
		Binding           map[string]any `json:"binding"`
		ReadArgv          []string       `json:"read_argv"`
		ReadCommand       string         `json:"read_command"`
		AckArgvPrefix     []string       `json:"ack_argv_prefix"`
		AckCommandPrefix  string         `json:"ack_command_prefix"`
	}
	if nativeContextUnambiguousJSON(pointerRaw) != nil || json.Unmarshal(pointerRaw, &pointer) != nil || pointer.SchemaVersion != 1 || pointer.Protocol != codexNativeContextProtocolChildFetch || pointer.Purpose != saved.Delivery.Purpose || pointer.ChildID != r.ChildID || pointer.CandidatePath != r.CandidatePath || pointer.CandidateSHA256 != r.CandidateSHA256 || pointer.RequestPath != path || pointer.RequestSHA256 != lifecycleDigest(requestRaw) {
		return "", fmt.Errorf("immutable context pointer does not bind original command and retained request")
	}
	readArgv := []string{r.CandidatePath, "codex-native-worker", "context", "--request", path}
	ackArgv := []string{r.CandidatePath, "codex-native-worker", "context-ack", "--request", path}
	readWords, readOK := nativeSimpleShellWords(pointer.ReadCommand)
	ackWords, ackOK := nativeSimpleShellWords(pointer.AckCommandPrefix)
	if !readOK || !ackOK || !reflect.DeepEqual(pointer.ReadArgv, readArgv) || !reflect.DeepEqual(pointer.AckArgvPrefix, ackArgv) || !reflect.DeepEqual(readWords, readArgv) || !reflect.DeepEqual(ackWords, ackArgv) {
		return "", fmt.Errorf("context pointer literal commands differ from original bound argv")
	}
	bindRaw, err := r.readEvidence(filepath.Join(retained, match[1]+"bind-request.json"))
	if err != nil {
		return "", err
	}
	var request codexNativeWorkerRequest
	var requestMap, bindMap map[string]any
	if nativeContextUnambiguousJSON(requestRaw) != nil || nativeContextUnambiguousJSON(bindRaw) != nil || json.Unmarshal(requestRaw, &request) != nil || json.Unmarshal(requestRaw, &requestMap) != nil || json.Unmarshal(bindRaw, &bindMap) != nil {
		return "", fmt.Errorf("malformed retained child-fetch request")
	}
	bindMap["context_purpose"] = saved.Delivery.Purpose
	d := saved.Delivery
	if !reflect.DeepEqual(bindMap, requestMap) || !reflect.DeepEqual(pointer.Binding, requestMap) || request.SchemaVersion != 1 || request.Phase != attempt.Phase || request.ExecutionBinding != d.ExecutionBinding || request.WorkerName != d.WorkerName || request.TaskID != d.TaskID || request.LaunchID != d.LaunchID || request.HostSessionID != d.HostSessionID || request.ChildID != d.ChildID || request.DispatchSHA256 != d.DispatchSHA256 || request.PromptSHA256 != d.PromptSHA256 || request.Workspace != d.Workspace || request.ContextPurpose != d.Purpose {
		return "", fmt.Errorf("child-fetch request does not preserve the exact bound assignment")
	}
	return path, nil
}

// A source-bound collaboration.wait_agent is passive release coordination, not
// assignment work. Admit only the actual observed call/activity/result shape,
// completed before the initial context read; never a general tool exception.
func nativeChildContextPassiveWait(records []nativeContextSourceRecord, lines []json.RawMessage, at int, evidence nativeChildContextFetchEvidence) ([]int, error) {
	fail := func(reason string) ([]int, error) { return nil, fmt.Errorf("child passive wait: %s", reason) }
	call := records[at]
	cp := call.Payload
	if evidence.Delivery.Purpose != "initial" || at >= evidence.Read.callIndex || call.Type != "response_item" || cp.Type != "function_call" || cp.Namespace != "collaboration" || cp.Name != "wait_agent" || cp.ID == "" || cp.CallID == "" || cp.ID == cp.CallID || cp.Metadata.TurnID == "" || cp.Metadata.TurnID != evidence.Fetch.Read.TurnID {
		return fail("call is not the exact child's initial release wait")
	}
	decodeExact := func(raw []byte, value any) error {
		if err := nativeContextUnambiguousJSON(raw); err != nil {
			return err
		}
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.DisallowUnknownFields()
		return decoder.Decode(value)
	}
	var args struct {
		TimeoutMS *int `json:"timeout_ms"`
	}
	if nativeContextRequiredKeys([]byte(cp.Arguments), "timeout_ms") != nil || decodeExact([]byte(cp.Arguments), &args) != nil || args.TimeoutMS == nil || *args.TimeoutMS < 10000 || *args.TimeoutMS > 3600000 {
		return fail("wait arguments are not one literal supported timeout")
	}
	activityIndex, resultIndex, callCount, callIDCount := -1, -1, 0, 0
	for i, record := range records {
		p := record.Payload
		if p.ID == cp.ID {
			callIDCount++
		}
		if record.Type == "response_item" && p.CallID == cp.CallID {
			switch p.Type {
			case "function_call":
				callCount++
				if i != at {
					return fail("duplicate or substituted wait call")
				}
			case "function_call_output":
				if resultIndex != -1 {
					return fail("duplicate wait result")
				}
				resultIndex = i
			default:
				return fail("wait call ID reused by another operation")
			}
		}
		if record.Type == "event_msg" && p.Type == "item_completed" && p.Item.ID == cp.CallID {
			if activityIndex != -1 {
				return fail("duplicate wait activity")
			}
			activityIndex = i
		}
	}
	if callCount != 1 || callIDCount != 1 || activityIndex <= at || resultIndex <= activityIndex || resultIndex >= evidence.Read.callIndex {
		return fail("unique complete wait triplet must precede the initial read")
	}
	activity, result := records[activityIndex], records[resultIndex]
	rp := result.Payload
	if activity.Payload.ThreadID != evidence.Fetch.Read.ChildID || activity.Payload.TurnID != cp.Metadata.TurnID || rp.Metadata.TurnID != cp.Metadata.TurnID || rp.ID == "" || rp.ID == cp.ID || rp.ID == cp.CallID || result.Metadata.ClientAuthored == nil || *result.Metadata.ClientAuthored {
		return fail("wait activity/result child, turn or host authorship differs")
	}
	resultIDCount := 0
	for _, record := range records {
		if record.Payload.ID == rp.ID {
			resultIDCount++
		}
		if record.Payload.Item.ID == rp.ID || record.Payload.Item.ID == cp.ID {
			return fail("wait source ID reused by an activity")
		}
	}
	if resultIDCount != 1 {
		return fail("wait result ID is not unique")
	}
	var envelope struct {
		Payload struct {
			Item          json.RawMessage `json:"item"`
			StartedAtMS   *int64          `json:"started_at_ms"`
			CompletedAtMS *int64          `json:"completed_at_ms"`
		} `json:"payload"`
	}
	var item struct {
		Type, ID, Tool, Status string
		SenderThreadID         string                     `json:"sender_thread_id"`
		ReceiverThreadIDs      []string                   `json:"receiver_thread_ids"`
		ReceiverAgents         []json.RawMessage          `json:"receiver_agents"`
		AgentsStates           map[string]json.RawMessage `json:"agents_states"`
	}
	if json.Unmarshal(lines[activityIndex], &envelope) != nil || decodeExact(envelope.Payload.Item, &item) != nil || nativeContextRequiredKeys(envelope.Payload.Item, "type", "id", "tool", "status", "sender_thread_id", "receiver_thread_ids", "receiver_agents", "agents_states") != nil || item.Type != "CollabAgentToolCall" || item.ID != cp.CallID || item.Tool != "wait" || item.Status != "completed" || item.SenderThreadID != evidence.Fetch.Read.ChildID || len(item.ReceiverThreadIDs) != 0 || len(item.ReceiverAgents) != 0 || len(item.AgentsStates) != 0 || envelope.Payload.StartedAtMS == nil || envelope.Payload.CompletedAtMS == nil {
		return fail("activity is not a completed passive wait with no recipients")
	}
	var output string
	var completion struct {
		Message  string `json:"message"`
		TimedOut *bool  `json:"timed_out"`
	}
	if json.Unmarshal(rp.Output, &output) != nil || nativeContextRequiredKeys([]byte(output), "message", "timed_out") != nil || decodeExact([]byte(output), &completion) != nil || completion.Message != "Wait completed." || completion.TimedOut == nil || *completion.TimedOut {
		return fail("actual wait completion output is missing or substituted")
	}
	start, a := time.Parse(time.RFC3339Nano, call.Timestamp)
	activityTime, b := time.Parse(time.RFC3339Nano, activity.Timestamp)
	end, c := time.Parse(time.RFC3339Nano, result.Timestamp)
	readTime, d := time.Parse(time.RFC3339Nano, evidence.Fetch.Read.StartedAt)
	started := time.UnixMilli(*envelope.Payload.StartedAtMS)
	completed := time.UnixMilli(*envelope.Payload.CompletedAtMS)
	if a != nil || b != nil || c != nil || d != nil || started.Before(start) || completed.Before(started) || activityTime.Before(completed) || end.Before(activityTime) || readTime.Before(end) {
		return fail("wait timestamps do not precede the initial read in actual order")
	}
	return []int{at, activityIndex, resultIndex}, nil
}

// Useful work must follow the child's ACK. The initial protocol permits only
// source-bound passive release waits followed by its read and ACK. Later answer
// reads impose the same no-work rule without exempting further tool calls.
func nativeChildContextWorkOrder(raw []byte, evidence nativeChildContextFetchEvidence) error {
	records, lines, err := nativeContextSourceRecords(raw)
	if err != nil {
		return err
	}
	start := evidence.Read.callIndex
	if evidence.Delivery.Purpose == "initial" {
		start = 0
	}
	passive := map[int]bool{}
	for i := start; i < evidence.ACK.resultIndex; i++ {
		record := records[i]
		p := record.Payload
		if evidence.Delivery.Purpose == "initial" && record.Type == "response_item" && p.Type == "function_call" && p.Name == "wait_agent" {
			indices, err := nativeChildContextPassiveWait(records, lines, i, evidence)
			if err != nil {
				return err
			}
			for _, index := range indices {
				passive[index] = true
			}
		}
		if passive[i] {
			continue
		}
		if evidence.Delivery.Purpose == "initial" && record.Type == "event_msg" && p.Type == "item_completed" && p.ThreadID == evidence.Fetch.Read.ChildID && p.Item.Type == "AgentMessage" && p.Item.Phase == "final_answer" {
			return fmt.Errorf("child question or final answer preceded initial context ACK")
		}
		if record.Type == "response_item" && (p.Type == "custom_tool_call" || p.Type == "function_call") && i != evidence.Read.callIndex && i != evidence.ACK.callIndex {
			return fmt.Errorf("child used a tool before acknowledging its complete context")
		}
		if record.Type == "event_msg" && p.Type == "item_completed" && p.Item.Type != "AgentMessage" && p.Item.Type != "Reasoning" && p.Item.ID != evidence.Read.Source.Command.ID && p.Item.ID != evidence.ACK.Source.Command.ID {
			return fmt.Errorf("child work preceded its complete context ACK")
		}
	}
	return nil
}

func nativeValidateChildContextReceipts(r codexNativeLiveReceipt, worker buildAttemptWorkerRun) error {
	_, err := nativeChildContextReceiptEvidence(r, worker)
	return err
}

func nativeChildContextReceiptEvidence(r codexNativeLiveReceipt, worker buildAttemptWorkerRun) ([]nativeChildContextFetchEvidence, error) {
	if r.ContextProtocol != codexNativeContextProtocolChildFetch || !r.ChildIdentityCorroborated || r.BoundHostSessionID == "" || worker.Native == nil || worker.Native.ContextProtocol != r.ContextProtocol || len(worker.Native.ContextDeliveries) == 0 {
		return nil, fmt.Errorf("actual child-fetch protocol or durable delivery missing")
	}
	raw, err := r.readEvidence(r.ChildEvents)
	if err != nil {
		return nil, err
	}
	parent, _, err := nativeChildContextParent(r)
	if err != nil {
		return nil, err
	}
	if _, err := nativeChildContextBootstrapBoundary(parent, r); err != nil {
		return nil, err
	}
	attemptRaw, err := r.readEvidence(r.AttemptPath)
	if err != nil {
		return nil, err
	}
	var attempt buildAttemptRecord
	if json.Unmarshal(attemptRaw, &attempt) != nil || attempt.PlanManifest == nil || attempt.PlanManifest.ContextProtocol != r.ContextProtocol {
		return nil, fmt.Errorf("manifest and capture context protocols differ")
	}
	matched := 0
	for _, savedWorker := range attempt.WorkerRuns {
		if reflect.DeepEqual(savedWorker, worker) {
			matched++
		}
	}
	if matched != 1 || worker.WorkerName != r.WorkerName || worker.TaskID != r.TaskID || worker.ProviderRunID != r.LaunchID || worker.Native.ChildID != r.ChildID || worker.Native.HostSessionID != r.BoundHostSessionID || worker.Native.Workspace != r.FixtureRoot || lifecycleDigest([]byte(worker.Native.Prompt)) != r.PromptSHA256 {
		return nil, fmt.Errorf("context worker does not match the exact retained attempt and child")
	}
	if err := validateCodexNativeSavedContext(attempt, worker); err != nil {
		return nil, err
	}
	if worker.Native.BoundAt == "" {
		return nil, fmt.Errorf("child binding time absent")
	}
	var result []nativeChildContextFetchEvidence
	seen, initial := map[string]bool{}, 0
	previous := worker.Native.BoundAt
	for index, saved := range worker.Native.ContextDeliveries {
		if err := validateCodexNativeContextReceipt(saved, attempt, worker); err != nil {
			return nil, err
		}
		if saved.Fetch == nil || seen[saved.Delivery.DeliveryID] {
			return nil, fmt.Errorf("missing or repeated child-fetch source")
		}
		seen[saved.Delivery.DeliveryID] = true
		if saved.Delivery.Purpose == "initial" {
			initial++
			if index != 0 || saved.Delivery.Payload != worker.Native.Prompt || saved.Delivery.PayloadSHA256 != worker.Native.PromptSHA256 {
				return nil, fmt.Errorf("initial context is not the full immutable runtime prompt")
			}
		} else if index == 0 {
			return nil, fmt.Errorf("answer delivered before initial prompt")
		}
		requestPath, err := nativeChildContextRequest(r, raw, saved, attempt)
		if err != nil {
			return nil, err
		}
		want := nativeChildContextFetchExpectation{nativeChildContextCommandExpectation: nativeChildContextCommandExpectation{ParentID: r.BoundHostSessionID, ChildID: r.ChildID, TaskPath: r.ChildTaskPath, Workspace: r.FixtureRoot}, Candidate: r.CandidatePath, RequestPath: requestPath, Purpose: saved.Delivery.Purpose, Delivery: saved.Delivery, NotBefore: previous}
		evidence, err := nativeDecodeChildContextFetch(raw, want)
		if err != nil {
			return nil, err
		}
		if !reflect.DeepEqual(evidence.Fetch, *saved.Fetch) || saved.Observation.SourceEventID != evidence.Fetch.Ack.Result.ID || saved.Observation.SourceEventSHA256 != evidence.Fetch.Ack.Result.SHA256 || saved.Observation.ObservedAt != evidence.Fetch.Ack.CompletedAt {
			return nil, fmt.Errorf("durable context observation is not the actual child ACK source")
		}
		if err := nativeChildContextWorkOrder(raw, evidence); err != nil {
			return nil, err
		}
		previous = evidence.Fetch.Ack.CompletedAt
		result = append(result, evidence)
	}
	if initial != 1 {
		return nil, fmt.Errorf("exactly one full initial context read required")
	}
	return result, nil
}

func nativeContextFetchFixture(t *testing.T, purpose string) ([]map[string]any, nativeChildContextFetchExpectation) {
	t.Helper()
	workspace := "/fixture repo"
	delivery := codexNativeContextDelivery{SchemaVersion: 2, Protocol: codexNativeContextProtocolChildFetch, Purpose: purpose,
		ExecutionBinding: codex.ExecutionBinding{SchemaVersion: 1, RunID: "run-fetch", AttemptID: "attempt-fetch", ManifestSHA256: strings.Repeat("a", 64), WorkspaceFingerprint: strings.Repeat("b", 64), ExecutionOwner: "native"},
		Scope:            codexNativeContextScope{GoalHash: "goal-hash", SessionID: "colony-session"}, WorkerName: "builder-1", TaskID: "1.1", LaunchID: "launch-1", HostSessionID: "parent-1", ChildID: "child-1", Workspace: workspace, DispatchSHA256: "dispatch-hash", PromptSHA256: lifecycleDigest([]byte("full café 日本語\n  prompt\t\n")), Payload: "full café 日本語\n  prompt\t\n", DecisionIDs: []string{"decision-1", "decision-2"}}
	if purpose == "answers" {
		delivery.Payload = "## CLARIFIED INTENT\n\n- Keep café 日本語 and  two spaces\n"
	}
	delivery.PayloadSHA256 = lifecycleDigest([]byte(delivery.Payload))
	delivery.DeliveryID, _ = jsonSHA256(delivery)
	want := nativeChildContextFetchExpectation{nativeChildContextCommandExpectation: nativeChildContextCommandExpectation{ParentID: "parent-1", ChildID: "child-1", TaskPath: "/root/builder", Workspace: workspace}, Candidate: workspace + "/bin/aether", RequestPath: "/capture/aether-worker-request-1/context-" + purpose + "-request.json", Purpose: purpose, Delivery: delivery}
	meta := map[string]any{"timestamp": "2026-09-18T21:22:00.000Z", "type": "session_meta", "payload": map[string]any{"id": want.ChildID, "parent_thread_id": want.ParentID, "thread_source": "subagent", "agent_path": want.TaskPath, "cwd": workspace, "runtime_workspace_roots": []string{workspace}, "source": map[string]any{"subagent": map[string]any{"thread_spawn": map[string]any{"parent_thread_id": want.ParentID, "agent_path": want.TaskPath}}}}}
	response := codexNativeWorkerResponse{SchemaVersion: 1, ContextStatus: "awaiting_ack", ExecutionBinding: delivery.ExecutionBinding, ContextDelivery: &delivery}
	readJSON, _ := json.Marshal(map[string]any{"ok": true, "result": response})
	ack := codexNativeContextAck{SchemaVersion: 1, DeliveryID: delivery.DeliveryID, PayloadSHA256: delivery.PayloadSHA256, ChildID: delivery.ChildID, DecisionIDs: delivery.DecisionIDs}
	response.ContextStatus, response.ContextDelivery, response.ContextAck = "ack_validated", nil, &ack
	ackJSON, _ := json.Marshal(map[string]any{"ok": true, "result": response})
	words := []string{want.Candidate, "codex-native-worker", "context", "--request", want.RequestPath}
	makeTriplet := func(label string, second int, words []string, stdout string) []map[string]any {
		command := nativeContextLiteralCommand(words)
		literal, _ := json.Marshal(map[string]any{"cmd": command, "workdir": workspace, "max_output_tokens": 20000})
		call := map[string]any{"timestamp": fmt.Sprintf("2026-09-18T21:22:%02d.010Z", second), "type": "response_item", "payload": map[string]any{"type": "custom_tool_call", "id": "ctc-" + label, "call_id": "call-" + label, "status": "completed", "name": "exec", "input": "const r = await tools.exec_command(" + string(literal) + "); text(r.output);", "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": "turn-1"}}}
		exec := map[string]any{"timestamp": fmt.Sprintf("2026-09-18T21:22:%02d.020Z", second), "type": "event_msg", "payload": map[string]any{"type": "item_completed", "thread_id": want.ChildID, "turn_id": "turn-1", "item": map[string]any{"type": "CommandExecution", "id": "exec-" + label, "status": "completed", "command": []string{"/bin/zsh", "-lc", command}, "cwd": "file:///fixture%20repo", "exit_code": 0, "stdout": stdout, "stderr": "", "aggregated_output": stdout, "formatted_output": stdout}}}
		result := map[string]any{"timestamp": fmt.Sprintf("2026-09-18T21:22:%02d.030Z", second), "type": "response_item", "payload": map[string]any{"type": "custom_tool_call_output", "id": "ctco-" + label, "call_id": "call-" + label, "output": []map[string]any{{"type": "input_text", "text": "Script completed\nWall time 0.1 seconds\nOutput:\n"}, {"type": "input_text", "text": stdout}}, "internal_chat_message_metadata_passthrough": map[string]any{"turn_id": "turn-1"}}, "metadata": map[string]any{"client_authored": false}}
		return []map[string]any{call, exec, result}
	}
	records := append([]map[string]any{meta}, makeTriplet("read", 1, words, string(readJSON)+"\n")...)
	words[2] = "context-ack"
	words = append(words, "--delivery-id", delivery.DeliveryID, "--payload-sha256", delivery.PayloadSHA256)
	for _, id := range delivery.DecisionIDs {
		words = append(words, "--decision-id", id)
	}
	records = append(records, makeTriplet("ack", 2, words, string(ackJSON)+"\n")...)
	return records, want
}

func nativeContextFixtureBytes(t *testing.T, records []map[string]any) []byte {
	t.Helper()
	var raw []byte
	for _, record := range records {
		line, err := json.Marshal(record)
		if err != nil {
			t.Fatal(err)
		}
		raw = append(raw, line...)
		raw = append(raw, '\n')
	}
	return raw
}

func TestCodexNativeChildContextFetchEvidence(t *testing.T) {
	t.Run("coordinator-eof-source", nativeContextCoordinatorEOFSource)
	for _, purpose := range []string{"initial", "answers"} {
		t.Run("complete-"+purpose, func(t *testing.T) {
			records, want := nativeContextFetchFixture(t, purpose)
			raw := nativeContextFixtureBytes(t, records)
			got, err := nativeDecodeChildContextFetch(raw, want)
			if err != nil {
				t.Fatal(err)
			}
			if got.Delivery.Payload != want.Delivery.Payload || got.Fetch.Read.Call.ID != "ctc-read" || got.Fetch.Read.Result.ID != "ctco-read" || got.Fetch.Ack.Command.ID != "exec-ack" || got.Fetch.Read.Call.SHA256 != strings.TrimPrefix(lifecycleDigest(got.Read.Call), "sha256:") || got.Fetch.Read.StartedAt != "2026-09-18T21:22:01.01Z" {
				t.Fatal("full payload or actual source identity not retained")
			}
			if err := nativeChildContextWorkOrder(raw, got); err != nil {
				t.Fatal(err)
			}
		})
	}
	objectResultFixture := func(print string) ([]map[string]any, nativeChildContextFetchExpectation) {
		records, want := nativeContextFetchFixture(t, "initial")
		for _, index := range []int{1, 4} {
			p := records[index]["payload"].(map[string]any)
			p["input"] = strings.ReplaceAll(p["input"].(string), "text(r.output)", print)
			out := records[index+2]["payload"].(map[string]any)["output"].([]map[string]any)
			wrapped, _ := json.Marshal(map[string]any{"exit_code": 0, "output": out[1]["text"], "wall_time_seconds": 0.1})
			out[1]["text"] = string(wrapped)
		}
		return records, want
	}
	for _, presentation := range []struct{ name, print string }{
		{"untouched-exec-result", "text(r)"},
		{"json-stringified-exec-result", "text(JSON.stringify(r))"},
	} {
		t.Run(presentation.name, func(t *testing.T) {
			records, want := objectResultFixture(presentation.print)
			got, err := nativeDecodeChildContextFetch(nativeContextFixtureBytes(t, records), want)
			if err != nil {
				t.Fatal(err)
			}
			if got.Delivery.Payload != want.Delivery.Payload || got.Ack.PayloadSHA256 != want.Delivery.PayloadSHA256 || got.Fetch.Read.CallID != "call-read" || got.Fetch.Ack.CallID != "call-ack" {
				t.Fatal("full JSON presentation changed the exact read/ACK binding")
			}
		})
	}
	for _, operation := range []struct {
		name      string
		callIndex int
	}{{"read", 1}, {"ack", 4}} {
		for _, mode := range []string{"missing-output", "substituted-output", "trailing-output", "missing-exit", "active-session", "exit-annotation"} {
			t.Run("json-result-"+operation.name+"-"+mode, func(t *testing.T) {
				records, want := objectResultFixture("text(JSON.stringify(r))")
				out := records[operation.callIndex+2]["payload"].(map[string]any)["output"].([]map[string]any)
				var result map[string]any
				if err := json.Unmarshal([]byte(out[1]["text"].(string)), &result); err != nil {
					t.Fatal(err)
				}
				switch mode {
				case "missing-output":
					delete(result, "output")
				case "substituted-output":
					result["output"] = "substituted context or ACK"
				case "missing-exit":
					delete(result, "exit_code")
				case "active-session":
					result["session_id"] = 42
				case "exit-annotation":
					p := records[operation.callIndex]["payload"].(map[string]any)
					p["input"] = strings.ReplaceAll(p["input"].(string), "text(JSON.stringify(r));", "text(r.output); text(`\\nEXIT_CODE=${r.exit_code}`);")
					if _, ok := nativeCodeModeOutputProjection(p["input"].(string), want.Workspace); !ok {
						t.Fatal("annotation control no longer matches the shared operation recognizer")
					}
				}
				changed, _ := json.Marshal(result)
				out[1]["text"] = string(changed)
				if mode == "trailing-output" {
					out[1]["text"] = string(changed) + "\n{}"
				}
				if _, err := nativeDecodeChildContextFetch(nativeContextFixtureBytes(t, records), want); err == nil {
					t.Fatal("incomplete, substituted or annotated full result accepted")
				}
			})
		}
	}
	payload := func(record map[string]any) map[string]any { return record["payload"].(map[string]any) }
	item := func(record map[string]any) map[string]any { return payload(record)["item"].(map[string]any) }
	metadata := func(record map[string]any) map[string]any {
		return payload(record)["internal_chat_message_metadata_passthrough"].(map[string]any)
	}
	tests := []struct {
		name   string
		mutate func([]map[string]any, *nativeChildContextFetchExpectation) []map[string]any
	}{
		{"parent-laundering", func(r []map[string]any, w *nativeChildContextFetchExpectation) []map[string]any {
			payload(r[0])["id"] = w.ParentID
			return r
		}},
		{"wrong-parent", func(r []map[string]any, w *nativeChildContextFetchExpectation) []map[string]any {
			payload(r[0])["parent_thread_id"] = "other-parent"
			return r
		}},
		{"wrong-task", func(r []map[string]any, w *nativeChildContextFetchExpectation) []map[string]any {
			payload(r[0])["agent_path"] = "/root/copied"
			return r
		}},
		{"wrong-workspace-root", func(r []map[string]any, w *nativeChildContextFetchExpectation) []map[string]any {
			payload(r[0])["runtime_workspace_roots"] = []string{"/other"}
			return r
		}},
		{"duplicate-call", func(r []map[string]any, w *nativeChildContextFetchExpectation) []map[string]any {
			return append(r, r[1])
		}},
		{"duplicate-result", func(r []map[string]any, w *nativeChildContextFetchExpectation) []map[string]any {
			return append(r, r[3])
		}},
		{"duplicate-execution", func(r []map[string]any, w *nativeChildContextFetchExpectation) []map[string]any {
			return append(r, r[2])
		}},
		{"missing-real-call-id", func(r []map[string]any, w *nativeChildContextFetchExpectation) []map[string]any {
			delete(payload(r[1]), "id")
			return r
		}},
		{"missing-real-result-id", func(r []map[string]any, w *nativeChildContextFetchExpectation) []map[string]any {
			delete(payload(r[3]), "id")
			return r
		}},
		{"missing-real-command-id", func(r []map[string]any, w *nativeChildContextFetchExpectation) []map[string]any {
			delete(item(r[2]), "id")
			return r
		}},
		{"wrong-execution-child", func(r []map[string]any, w *nativeChildContextFetchExpectation) []map[string]any {
			payload(r[2])["thread_id"] = w.ParentID
			return r
		}},
		{"wrong-execution-turn", func(r []map[string]any, w *nativeChildContextFetchExpectation) []map[string]any {
			payload(r[2])["turn_id"] = "other-turn"
			return r
		}},
		{"wrong-result-turn", func(r []map[string]any, w *nativeChildContextFetchExpectation) []map[string]any {
			metadata(r[3])["turn_id"] = "other-turn"
			return r
		}},
		{"wrong-call-join", func(r []map[string]any, w *nativeChildContextFetchExpectation) []map[string]any {
			payload(r[3])["call_id"] = "call-other"
			return r
		}},
		{"wrong-call-cwd", func(r []map[string]any, w *nativeChildContextFetchExpectation) []map[string]any {
			payload(r[1])["input"] = strings.ReplaceAll(payload(r[1])["input"].(string), `"workdir":"/fixture repo"`, `"workdir":"/other"`)
			return r
		}},
		{"wrong-command-cwd", func(r []map[string]any, w *nativeChildContextFetchExpectation) []map[string]any {
			item(r[2])["cwd"] = "file:///other"
			return r
		}},
		{"copied-output-from-cat", func(r []map[string]any, w *nativeChildContextFetchExpectation) []map[string]any {
			item(r[2])["command"] = []string{"/bin/zsh", "-lc", "cat context.json"}
			return r
		}},
		{"nonzero-read", func(r []map[string]any, w *nativeChildContextFetchExpectation) []map[string]any {
			item(r[2])["exit_code"] = 1
			return r
		}},
		{"missing-exit-status", func(r []map[string]any, w *nativeChildContextFetchExpectation) []map[string]any {
			delete(item(r[2]), "exit_code")
			return r
		}},
		{"truncated-stdout", func(r []map[string]any, w *nativeChildContextFetchExpectation) []map[string]any {
			item(r[2])["stdout"] = "truncated"
			return r
		}},
		{"copied-result-only", func(r []map[string]any, w *nativeChildContextFetchExpectation) []map[string]any {
			payload(r[3])["output"].([]map[string]any)[1]["text"] = "copied JSON"
			return r
		}},
		{"truncation-header", func(r []map[string]any, w *nativeChildContextFetchExpectation) []map[string]any {
			payload(r[3])["output"].([]map[string]any)[0]["text"] = "Script completed\nOutput truncated\nOutput:\n"
			return r
		}},
		{"client-authored-result", func(r []map[string]any, w *nativeChildContextFetchExpectation) []map[string]any {
			r[3]["metadata"].(map[string]any)["client_authored"] = true
			return r
		}},
		{"reversed-ack", func(r []map[string]any, w *nativeChildContextFetchExpectation) []map[string]any {
			return []map[string]any{r[0], r[4], r[5], r[6], r[1], r[2], r[3]}
		}},
		{"ack-time-before-read", func(r []map[string]any, w *nativeChildContextFetchExpectation) []map[string]any {
			r[4]["timestamp"] = "2026-09-18T21:22:00Z"
			return r
		}},
		{"result-time-before-command", func(r []map[string]any, w *nativeChildContextFetchExpectation) []map[string]any {
			r[3]["timestamp"] = "2026-09-18T21:22:00Z"
			return r
		}},
		{"stale-ack-id", func(r []map[string]any, w *nativeChildContextFetchExpectation) []map[string]any {
			payload(r[4])["input"] = strings.ReplaceAll(payload(r[4])["input"].(string), w.Delivery.DeliveryID, "stale-id")
			return r
		}},
		{"stale-decision-id", func(r []map[string]any, w *nativeChildContextFetchExpectation) []map[string]any {
			payload(r[4])["input"] = strings.ReplaceAll(payload(r[4])["input"].(string), "decision-2", "stale-decision")
			return r
		}},
		{"wrong-candidate", func(r []map[string]any, w *nativeChildContextFetchExpectation) []map[string]any {
			w.Candidate = "/other/aether"
			return r
		}},
		{"wrong-request", func(r []map[string]any, w *nativeChildContextFetchExpectation) []map[string]any {
			w.RequestPath += ".old"
			return r
		}},
		{"wrong-purpose", func(r []map[string]any, w *nativeChildContextFetchExpectation) []map[string]any {
			w.Purpose = "answers"
			return r
		}},
		{"bad-payload-hash", func(r []map[string]any, w *nativeChildContextFetchExpectation) []map[string]any {
			w.Delivery.PayloadSHA256 = lifecycleDigest([]byte("abbreviated"))
			return r
		}},
		{"unknown-protocol", func(r []map[string]any, w *nativeChildContextFetchExpectation) []map[string]any {
			w.Delivery.Protocol = "host-send"
			return r
		}},
		{"read-before-answer-authorization", func(r []map[string]any, w *nativeChildContextFetchExpectation) []map[string]any {
			w.NotBefore = "2026-09-18T21:22:02Z"
			return r
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, want := nativeContextFetchFixture(t, "initial")
			r = tc.mutate(r, &want)
			if _, err := nativeDecodeChildContextFetch(nativeContextFixtureBytes(t, r), want); err == nil {
				t.Fatal("uncorroborated fetch accepted")
			}
		})
	}
	for _, field := range []string{"payload", "scope", "execution_binding", "worker_name", "task_id", "launch_id", "host_session_id", "child_id", "workspace", "dispatch_sha256", "prompt_sha256", "decision_ids"} {
		t.Run("actual-delivery-changed-"+field, func(t *testing.T) {
			r, want := nativeContextFetchFixture(t, "initial")
			var response map[string]any
			_ = json.Unmarshal([]byte(item(r[2])["stdout"].(string)), &response)
			d := response["result"].(map[string]any)["context_delivery"].(map[string]any)
			switch field {
			case "scope":
				d[field].(map[string]any)["session_id"] = "other"
			case "execution_binding":
				d[field].(map[string]any)["run_id"] = "run-other"
			case "decision_ids":
				d[field] = []string{"stale"}
			default:
				d[field] = "changed"
			}
			changed, _ := json.Marshal(response)
			for _, key := range []string{"stdout", "aggregated_output", "formatted_output"} {
				item(r[2])[key] = string(changed) + "\n"
			}
			payload(r[3])["output"].([]map[string]any)[1]["text"] = string(changed) + "\n"
			if _, err := nativeDecodeChildContextFetch(nativeContextFixtureBytes(t, r), want); err == nil {
				t.Fatal("changed full envelope accepted")
			}
		})
	}
	t.Run("work-before-ack", func(t *testing.T) {
		r, want := nativeContextFetchFixture(t, "initial")
		work := map[string]any{"timestamp": "2026-09-18T21:22:00Z", "type": "response_item", "payload": map[string]any{"type": "function_call", "name": "apply_patch", "id": "work", "arguments": "patch"}}
		r = append([]map[string]any{r[0], work}, r[1:]...)
		raw := nativeContextFixtureBytes(t, r)
		got, err := nativeDecodeChildContextFetch(raw, want)
		if err != nil {
			t.Fatal(err)
		}
		if err := nativeChildContextWorkOrder(raw, got); err == nil {
			t.Fatal("useful work before initial ACK accepted")
		}
	})
	for _, phase := range []string{"final_answer", "commentary"} {
		t.Run("message-before-ack-"+phase, func(t *testing.T) {
			records, want := nativeContextFetchFixture(t, "initial")
			message := map[string]any{"timestamp": "2026-09-18T21:22:00Z", "type": "event_msg", "payload": map[string]any{"type": "item_completed", "thread_id": want.ChildID, "turn_id": "turn-1", "item": map[string]any{"type": "AgentMessage", "id": "question", "phase": phase, "content": `{"question_id":"bounds","question":"What if bounds are reversed?"}`}}}
			records = append([]map[string]any{records[0], message}, records[1:]...)
			raw := nativeContextFixtureBytes(t, records)
			got, err := nativeDecodeChildContextFetch(raw, want)
			if err != nil {
				t.Fatal(err)
			}
			err = nativeChildContextWorkOrder(raw, got)
			if (err == nil) != (phase == "commentary") {
				t.Fatalf("phase=%s error=%v", phase, err)
			}
		})
	}
	t.Run("duplicate-json-member", func(t *testing.T) {
		r, want := nativeContextFetchFixture(t, "initial")
		raw := bytes.Replace(nativeContextFixtureBytes(t, r), []byte(`"id":"ctc-read"`), []byte(`"id":"ctc-read","id":"ctc-read"`), 1)
		if _, err := nativeDecodeChildContextFetch(raw, want); err == nil {
			t.Fatal("ambiguous JSON source accepted")
		}
	})
}

func nativeContextCoordinatorEOFSource(t *testing.T) {
	for _, presentation := range []string{"object", "output", "direct"} {
		for _, mode := range []string{"valid", "missing-separator", "extra-effect", "wrong-variable", "wrong-cwd", "wrong-turn", "missing-event", "duplicate-event", "wrong-stdout", "missing-exit", "substituted-result"} {
			t.Run(presentation+"/"+mode, func(t *testing.T) {
				records, want := nativeContextFetchFixture(t, "initial")
				for _, index := range []int{1, 4} {
					call := records[index]["payload"].(map[string]any)
					input := call["input"].(string)
					print := "text(r)"
					if presentation == "output" {
						print = "text(r.output)"
					}
					input = strings.Replace(input, "text(r.output);", print+"\n", 1)
					if presentation == "direct" {
						input = "text(" + strings.TrimSuffix(strings.TrimPrefix(input, "const r = "), "; text(r)\n") + ")\n"
					}
					if mode == "missing-separator" {
						if presentation == "direct" {
							input += input
						} else {
							input = strings.Replace(input, "; text(", "\ntext(", 1)
						}
					} else if mode == "extra-effect" {
						input += "mutate();"
					} else if mode == "wrong-variable" {
						if presentation == "direct" {
							input = strings.Replace(input, "text(", "other(", 1)
						} else {
							input = strings.ReplaceAll(input, "text(r", "text(other")
							input = strings.ReplaceAll(input, "stringify(r)", "stringify(other)")
						}
					}
					call["input"] = input
					item := records[index+1]["payload"].(map[string]any)["item"].(map[string]any)
					out := records[index+2]["payload"].(map[string]any)["output"].([]map[string]any)
					if presentation != "output" {
						result := map[string]any{"exit_code": 0, "output": out[1]["text"]}
						if mode == "missing-exit" {
							delete(result, "exit_code")
						}
						if mode == "substituted-result" {
							result["output"] = "substituted"
						}
						encoded, _ := json.Marshal(result)
						out[1]["text"] = string(encoded)
					} else if mode == "substituted-result" {
						out[1]["text"] = "substituted"
					}
					switch mode {
					case "wrong-cwd":
						item["cwd"] = "/other"
					case "wrong-turn":
						records[index+1]["payload"].(map[string]any)["turn_id"] = "foreign"
					case "missing-event":
						item["type"] = "Other"
					case "wrong-stdout":
						item["stdout"] = "conflicting"
					case "missing-exit":
						delete(item, "exit_code")
					}
				}
				if mode == "duplicate-event" {
					records = append(records, records[2])
				}
				raw := nativeContextFixtureBytes(t, records)
				got, err := nativeDecodeChildContextFetch(raw, want)
				if (err == nil) != (mode == "valid") {
					t.Fatalf("Go source acceptance=%v: %v", err == nil, err)
				}
				if err == nil && (got.Delivery.Payload != want.Delivery.Payload || got.Ack.PayloadSHA256 != want.Delivery.PayloadSHA256) {
					t.Fatal("EOF wrapper changed actual context bytes")
				}
				// Run the actual pure Python coordinator decoder on the same raw
				// triplets; a Go-only recognizer cannot guard this acquisition path.
				script := "import json,hashlib,sys,datetime\n" + nativeContextFetchPython + `
raw_lines = sys.stdin.buffer.read().splitlines()
events = [native_fetch_json(line) for line in raw_lines]
for ci in (1,4):
    parsed = native_fetch_exec(events[ci]["payload"]["input"], "/fixture repo")
    assert parsed is not None, "unclassified wrapper"
    native_fetch_source(raw_lines, events, ci, parsed, "/fixture repo", "child-1")
`
				cmd := exec.Command("python3", "-c", script)
				cmd.Stdin = bytes.NewReader(raw)
				output, pythonErr := cmd.CombinedOutput()
				if (pythonErr == nil) != (mode == "valid") {
					t.Fatalf("Python source acceptance=%v: %v\n%s", pythonErr == nil, pythonErr, output)
				}
			})
		}
	}
}

func TestCodexNativeChildContextBootstrap(t *testing.T) {
	for _, mode := range []string{"none", "all", "unknown", "wrong-child", "wrong-call", "duplicate", "opaque-bootstrap", "missing-bound-parent", "wrong-bound-parent"} {
		t.Run(mode, func(t *testing.T) {
			fork, child, message := "none", "child", "Read only your runtime metadata pointer after binding."
			if mode == "all" || mode == "unknown" {
				fork = mode
			}
			if mode == "wrong-child" {
				child = "other"
			}
			if mode == "opaque-bootstrap" {
				message = "gAAAAencrypted-bootstrap"
			}
			arguments, _ := json.Marshal(map[string]any{"message": message, "task_name": "worker", "fork_turns": fork})
			turn := map[string]any{"turn_id": "turn"}
			call := map[string]any{"type": "response_item", "payload": map[string]any{"type": "function_call", "name": "spawn_agent", "call_id": "spawn-call", "arguments": string(arguments), "internal_chat_message_metadata_passthrough": turn}}
			activity := map[string]any{"timestamp": "2026-09-18T21:22:00Z", "type": "event_msg", "payload": map[string]any{"type": "item_completed", "thread_id": "parent", "turn_id": "turn", "item": map[string]any{"type": "SubAgentActivity", "id": "spawn-call", "kind": "started", "agent_thread_id": child, "agent_path": "/root/worker"}}}
			result := map[string]any{"type": "response_item", "payload": map[string]any{"type": "function_call_output", "call_id": "spawn-call", "output": `{"task_name":"/root/worker"}`, "internal_chat_message_metadata_passthrough": turn}}
			records := []map[string]any{call, activity, result}
			if mode == "duplicate" {
				records = append(records, call)
			}
			r := codexNativeLiveReceipt{SessionID: "original-parent", BoundHostSessionID: "parent", ChildID: "child", ChildTaskPath: "/root/worker", ChildSpawnCallID: "spawn-call"}
			if mode == "wrong-call" {
				r.ChildSpawnCallID = "other-call"
			}
			if mode == "missing-bound-parent" {
				r.BoundHostSessionID = ""
			}
			if mode == "wrong-bound-parent" {
				r.BoundHostSessionID = "wrong-parent"
			}
			boundary, err := nativeChildContextBootstrapBoundary(nativeContextFixtureBytes(t, records), r)
			valid := mode == "none" || mode == "opaque-bootstrap"
			if (err == nil) != valid {
				t.Fatalf("valid=%v err=%v", valid, err)
			}
			if valid && (boundary.ContextMode != "none" || boundary.Message != message) {
				t.Fatal("actual bootstrap bytes or fork mode changed")
			}
		})
	}
}

// The passive-wait triplet mirrors actual Codex 0.155 child rollout lines
// 17/19/20 retained in child-fetch-passive-wait-20260918T234656Z-hcnvzd02/chronology.json.
func TestCodexNativeChildContextPassiveWait(t *testing.T) {
	fixture := func() ([]map[string]any, nativeChildContextFetchExpectation) {
		records, want := nativeContextFetchFixture(t, "initial")
		base, _ := time.Parse(time.RFC3339Nano, "2026-09-18T21:22:00Z")
		meta := func() map[string]any { return map[string]any{"turn_id": "turn-1"} }
		call := map[string]any{"timestamp": "2026-09-18T21:22:00.1Z", "type": "response_item", "payload": map[string]any{
			"type": "function_call", "id": "fc-wait", "namespace": "collaboration", "name": "wait_agent", "call_id": "call-wait", "arguments": `{"timeout_ms":3600000}`, "internal_chat_message_metadata_passthrough": meta()}}
		activity := map[string]any{"timestamp": "2026-09-18T21:22:00.2Z", "type": "event_msg", "payload": map[string]any{
			"type": "item_completed", "thread_id": want.ChildID, "turn_id": "turn-1", "started_at_ms": base.Add(110 * time.Millisecond).UnixMilli(), "completed_at_ms": base.Add(200 * time.Millisecond).UnixMilli(),
			"item": map[string]any{"type": "CollabAgentToolCall", "id": "call-wait", "tool": "wait", "status": "completed", "sender_thread_id": want.ChildID, "receiver_thread_ids": []string{}, "receiver_agents": []any{}, "agents_states": map[string]any{}}}}
		result := map[string]any{"timestamp": "2026-09-18T21:22:00.3Z", "type": "response_item", "payload": map[string]any{
			"type": "function_call_output", "id": "fco-wait", "call_id": "call-wait", "output": `{"message":"Wait completed.","timed_out":false}`, "internal_chat_message_metadata_passthrough": meta()}, "metadata": map[string]any{"client_authored": false}}
		return append([]map[string]any{records[0], call, activity, result}, records[1:]...), want
	}
	validate := func(records []map[string]any, want nativeChildContextFetchExpectation) error {
		raw := nativeContextFixtureBytes(t, records)
		evidence, err := nativeDecodeChildContextFetch(raw, want)
		if err != nil {
			return err
		}
		return nativeChildContextWorkOrder(raw, evidence)
	}
	payload := func(record map[string]any) map[string]any { return record["payload"].(map[string]any) }
	item := func(record map[string]any) map[string]any { return payload(record)["item"].(map[string]any) }
	work := func() map[string]any {
		return map[string]any{"timestamp": "2026-09-18T21:22:03Z", "type": "response_item", "payload": map[string]any{"type": "function_call", "name": "exec_command", "id": "work", "call_id": "work-call", "arguments": `{"cmd":"go test ./..."}`}}
	}
	t.Run("wait-read-separate-ack-then-work", func(t *testing.T) {
		r, want := fixture()
		r = append(r, work())
		if err := validate(r, want); err != nil {
			t.Fatal(err)
		}
	})
	tests := []struct {
		name   string
		mutate func([]map[string]any) []map[string]any
	}{
		{"mutating-tool-name", func(r []map[string]any) []map[string]any { payload(r[1])["name"] = "send_message"; return r }},
		{"wrong-namespace", func(r []map[string]any) []map[string]any { payload(r[1])["namespace"] = "other"; return r }},
		{"dynamic-arguments", func(r []map[string]any) []map[string]any {
			payload(r[1])["arguments"] = `{"timeout_ms":getTimeout()}`
			return r
		}},
		{"additional-recipient-argument", func(r []map[string]any) []map[string]any {
			payload(r[1])["arguments"] = `{"timeout_ms":3600000,"target":"another-worker"}`
			return r
		}},
		{"duplicate-argument", func(r []map[string]any) []map[string]any {
			payload(r[1])["arguments"] = `{"timeout_ms":10000,"timeout_ms":3600000}`
			return r
		}},
		{"null-timeout", func(r []map[string]any) []map[string]any {
			payload(r[1])["arguments"] = `{"timeout_ms":null}`
			return r
		}},
		{"out-of-range-timeout", func(r []map[string]any) []map[string]any {
			payload(r[1])["arguments"] = `{"timeout_ms":3600001}`
			return r
		}},
		{"substituted-result-call", func(r []map[string]any) []map[string]any { payload(r[3])["call_id"] = "other-call"; return r }},
		{"parent-activity", func(r []map[string]any) []map[string]any {
			payload(r[2])["thread_id"] = "parent"
			item(r[2])["sender_thread_id"] = "parent"
			return r
		}},
		{"wrong-turn", func(r []map[string]any) []map[string]any {
			payload(r[3])["internal_chat_message_metadata_passthrough"] = map[string]any{"turn_id": "other"}
			return r
		}},
		{"mutating-activity", func(r []map[string]any) []map[string]any { item(r[2])["tool"] = "spawn_agent"; return r }},
		{"nonempty-receiver", func(r []map[string]any) []map[string]any {
			item(r[2])["receiver_thread_ids"] = []string{"another-worker"}
			return r
		}},
		{"missing-receiver-field", func(r []map[string]any) []map[string]any { delete(item(r[2]), "receiver_thread_ids"); return r }},
		{"duplicate-activity", func(r []map[string]any) []map[string]any {
			return append(append([]map[string]any{}, r[:3]...), append([]map[string]any{r[2]}, r[3:]...)...)
		}},
		{"client-written-result", func(r []map[string]any) []map[string]any {
			r[3]["metadata"] = map[string]any{"client_authored": true}
			return r
		}},
		{"substituted-result", func(r []map[string]any) []map[string]any {
			payload(r[3])["output"] = `{"message":"Work completed.","timed_out":false}`
			return r
		}},
		{"reversed-result-time", func(r []map[string]any) []map[string]any { r[3]["timestamp"] = "2026-09-18T21:21:59Z"; return r }},
		{"work-interleaved-with-wait", func(r []map[string]any) []map[string]any {
			return append(append([]map[string]any{}, r[:2]...), append([]map[string]any{work()}, r[2:]...)...)
		}},
		{"wait-between-read-and-ack", func(r []map[string]any) []map[string]any {
			reordered := append([]map[string]any{r[0]}, r[4:7]...)
			reordered = append(reordered, r[1:4]...)
			return append(reordered, r[7:]...)
		}},
		{"question-before-ack", func(r []map[string]any) []map[string]any {
			question := map[string]any{"timestamp": "2026-09-18T21:22:00.15Z", "type": "event_msg", "payload": map[string]any{"type": "item_completed", "thread_id": "child-1", "turn_id": "turn-1", "item": map[string]any{"type": "AgentMessage", "id": "question", "phase": "final_answer", "content": `{"question":"Which behavior should I implement?"}`}}}
			return append(append([]map[string]any{}, r[:2]...), append([]map[string]any{question}, r[2:]...)...)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			records, want := fixture()
			if err := validate(test.mutate(records), want); err == nil {
				t.Fatal("unbound wait or pre-ACK work accepted")
			}
		})
	}
}
