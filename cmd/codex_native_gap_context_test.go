package cmd

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// Fixture authority lives in this harness process, not in the coordinator or
// child prompt. No selected answer exists before an attributed question.
type nativeGapChallenge struct {
	Authority      string `json:"authority"`
	SelectedAt     string `json:"selected_at"`
	QuestionSHA256 string `json:"question_sha256"`
	ChildID        string `json:"child_id"`
	PanicText      string `json:"panic_text"`
	Answer         string `json:"answer"`
	SnapshotSHA256 string `json:"snapshot_sha256"`
}

func nativeGapWrite(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	temporary := path + ".pending"
	if err = os.WriteFile(temporary, append(raw, '\n'), 0600); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

func nativeGapStartController(runRoot, repo, home, coord string) func() {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				_ = nativeGapWrite(filepath.Join(runRoot, "controller", "outcome.json"), map[string]any{"status": "incomplete", "reason": "no registered child question before host stopped"})
				return
			case <-ticker.C:
				raw, err := os.ReadFile(filepath.Join(coord, "w0-question.stdout.json"))
				var envelope struct {
					OK     bool
					Result struct {
						Decisions []struct {
							AnswerRequestPath string `json:"answer_request_path"`
						}
					}
				}
				if err != nil || json.Unmarshal(raw, &envelope) != nil || !envelope.OK || len(envelope.Result.Decisions) != 1 {
					continue
				}
				err = nativeGapAuthorize(runRoot, repo, home, coord, envelope.Result.Decisions[0].AnswerRequestPath)
				status := "authorized"
				if err != nil {
					status = err.Error()
				}
				_ = nativeGapWrite(filepath.Join(runRoot, "controller", "outcome.json"), map[string]any{"status": status})
				if err != nil {
					_ = nativeGapWrite(filepath.Join(coord, "controller-ready.json"), map[string]any{"status": status})
				}
				return
			}
		}
	}()
	return func() { cancel(); <-done }
}

func nativeGapAuthorize(runRoot, repo, home, coord, requestPath string) error {
	raw, err := os.ReadFile(requestPath)
	var answer codexNativeDecisionAnswerRequest
	if err != nil || json.Unmarshal(raw, &answer) != nil || answer.Answer != "" || answer.Binding.ChildID == "" {
		return fmt.Errorf("missing empty bound answer request")
	}
	source, err := os.ReadFile(filepath.Join(coord, "w0-question-source.jsonl"))
	var event nativeHostEvent
	if err != nil || json.Unmarshal(bytes.TrimSpace(source), &event) != nil || event.Type != "event_msg" || event.Payload.Type != "item_completed" || event.Payload.ThreadID != answer.Binding.ChildID || event.Payload.Item.Type != "AgentMessage" || event.Payload.Item.Phase != "final_answer" {
		return fmt.Errorf("no actual child question event")
	}
	paths, _ := filepath.Glob(filepath.Join(home, ".codex", "sessions", "*", "*", "*", "*"+answer.Binding.ChildID+".jsonl"))
	if len(paths) != 1 {
		return fmt.Errorf("missing unique child question capture")
	}
	child, err := os.ReadFile(paths[0])
	if err != nil || !bytes.Contains(child, bytes.TrimSpace(source)) {
		return fmt.Errorf("question source not in actual child capture")
	}
	// Decode actual child content, rather than trusting a coordinator description.
	var text string
	if json.Unmarshal(event.Payload.Item.Content, &text) != nil {
		var parts []struct {
			Text string `json:"text"`
		}
		if json.Unmarshal(event.Payload.Item.Content, &parts) != nil {
			return fmt.Errorf("unreadable child question")
		}
		for _, p := range parts {
			text += p.Text
		}
	}
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimSuffix(text, "```")
	var q codexNativeQuestion
	if json.Unmarshal([]byte(text), &q) != nil || q.Question != answer.Question || q.QuestionID != answer.Binding.QuestionID {
		return fmt.Errorf("registered question differs from actual child question")
	}
	// Snapshot all fixture files and public isolated host inputs/history BEFORE
	// entropy is drawn. Credentials are never copied into proof artifacts.
	snapshot := map[string][]byte{}
	for _, root := range []string{repo, coord, filepath.Join(home, ".codex", "sessions"), filepath.Join(home, ".codex", "skills"), filepath.Join(home, ".codex", "agents")} {
		err = filepath.WalkDir(root, func(path string, d fs.DirEntry, e error) error {
			if os.IsNotExist(e) {
				return nil
			}
			if e != nil {
				return e
			}
			if d.IsDir() {
				if d.Name() == ".git" {
					return filepath.SkipDir
				}
				return nil
			}
			if d.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("pre-question symlink: %s", path)
			}
			b, e := os.ReadFile(path)
			if e != nil {
				return e
			}
			snapshot[path] = b
			return nil
		})
		if err != nil {
			return err
		}
	}
	for _, path := range []string{filepath.Join(runRoot, "prompt.txt"), filepath.Join(home, ".codex", "config.toml")} {
		if b, e := os.ReadFile(path); e == nil {
			snapshot[path] = b
		}
	}
	if err = nativeGapWrite(filepath.Join(runRoot, "controller", "pre-question.json"), snapshot); err != nil {
		return err
	}
	snapshotRaw, err := os.ReadFile(filepath.Join(runRoot, "controller", "pre-question.json"))
	if err != nil {
		return err
	}
	nonce := make([]byte, 32)
	if _, err = rand.Read(nonce); err != nil {
		return err
	}
	panicText := "bounds-" + hex.EncodeToString(nonce)
	answer.Answer = fmt.Sprintf("Fixture-authorized delayed response: when low exceeds high, panic with exactly %q. Keep inclusive ordered bounds. Run your own check of the exact panic text after editing. This is fixture authority, not owner testimony.", panicText)
	if err = nativeGapNoLeak(snapshot, answer.Answer, panicText); err != nil {
		return err
	}
	challenge := nativeGapChallenge{Authority: "independent delayed fixture controller", SelectedAt: time.Now().UTC().Format(time.RFC3339Nano), QuestionSHA256: lifecycleDigest(bytes.TrimSpace(source)), ChildID: answer.Binding.ChildID, PanicText: panicText, Answer: answer.Answer, SnapshotSHA256: lifecycleDigest(snapshotRaw)}
	if err = nativeGapWrite(filepath.Join(runRoot, "controller", "challenge.json"), challenge); err != nil {
		return err
	}
	if err = nativeGapWrite(filepath.Join(coord, "fixture-answer-provenance.json"), challenge); err != nil {
		return err
	}
	// Only now fill the one Go-issued answer template. Never invent another store.
	if err = nativeGapWrite(requestPath, answer); err != nil {
		return err
	}
	raw, err = os.ReadFile(requestPath)
	if err != nil {
		return err
	}
	return nativeGapWrite(filepath.Join(coord, "controller-ready.json"), map[string]any{"status": "authorized", "request_sha256": strings.TrimPrefix(lifecycleDigest(raw), "sha256:")})
}

func nativeGapNoLeak(snapshot map[string][]byte, answer, token string) error {
	if len(snapshot) == 0 || answer == "" || token == "" {
		return fmt.Errorf("missing pre-question material or answer")
	}
	for path, raw := range snapshot {
		if bytes.Contains(raw, []byte(answer)) || bytes.Contains(raw, []byte(token)) {
			return fmt.Errorf("pre-question answer leakage in %s", path)
		}
		// Escaped JSON history is still readable context, not an absence proof.
		for _, line := range bytes.Split(raw, []byte{'\n'}) {
			var value any
			if json.Unmarshal(line, &value) == nil {
				var scan func(any) bool
				scan = func(v any) bool {
					switch x := v.(type) {
					case string:
						return strings.Contains(x, answer) || strings.Contains(x, token)
					case []any:
						for _, v := range x {
							if scan(v) {
								return true
							}
						}
					case map[string]any:
						for _, v := range x {
							if scan(v) {
								return true
							}
						}
					}
					return false
				}
				if scan(value) {
					return fmt.Errorf("escaped pre-question leakage in %s", path)
				}
			}
		}
	}
	return nil
}

var nativeGapMarkers = []string{"GAP_CAPSULE_café_日本語", "GAP_SKILL_exact_bytes", "GAP_STEERING_keep_inclusive", "GAP_HANDOFF_prior_ordered_bounds"}

func nativeGapPrepareContext(t *testing.T, root string) {
	for _, dir := range []string{filepath.Join(root, ".aether", "skills", "colony", "gap-context"), filepath.Join(root, ".aether", "data", "handoffs")} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(root, ".aether", "data", "COLONY_STATE.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var state colony.ColonyState
	if err = json.Unmarshal(raw, &state); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if state.Goal != nil && !strings.Contains(*state.Goal, nativeGapMarkers[0]) {
		*state.Goal += " " + nativeGapMarkers[0]
	}
	state.Memory.Decisions = append(state.Memory.Decisions, colony.Decision{ID: "gap-context", Phase: 1, Claim: nativeGapMarkers[0], Rationale: "Recognizable fixture context", Timestamp: now})
	liveSkillWriteJSON(t, path, state)
	strength := 1.0
	content, _ := json.Marshal(map[string]string{"text": nativeGapMarkers[2]})
	liveSkillWriteJSON(t, filepath.Join(root, ".aether", "data", "pheromones.json"), colony.PheromoneFile{Signals: []colony.PheromoneSignal{{ID: "gap-steering", Type: "FOCUS", Priority: "normal", Source: "user", CreatedAt: now, Active: true, Strength: &strength, Content: content}}})
	liveSkillWrite(t, filepath.Join(root, ".aether", "skills", "colony", "gap-context", "SKILL.md"), []byte("---\nname: gap-context\ntype: colony\nagent_roles:\n  - builder\nworkflow_triggers: [build]\ntask_keywords: [clamp, bounds, integer]\ndetect_files: [go.mod]\n---\n"+nativeGapMarkers[1]+"\n"))
	record := buildWorkerHandoffRecord(codex.WorkerDispatch{WorkerName: "FixturePrior", Caste: "builder", TaskID: "0.1", Workflow: "build", Phase: 1, Root: root}, codex.DispatchResult{WorkerName: "FixturePrior", Status: "completed", WorkerResult: &codex.WorkerResult{WorkerName: "FixturePrior", Caste: "builder", TaskID: "0.1", Status: "completed", Summary: "Fixture-prepared prior handoff", Handoff: codex.WorkerHandoff{VerificationStatus: "pass", NextWorkerInstructions: []string{nativeGapMarkers[3]}, Freshness: now}}})
	liveSkillWriteJSON(t, filepath.Join(root, ".aether", "data", workerHandoffsPath), workerHandoffFile{Entries: []workerHandoffRecord{record}})
}

// These are the actual host boundary records, retained verbatim for independent
// review. An encrypted message is unavailable evidence, not exact plaintext.
type nativeGapBoundary struct {
	Call        json.RawMessage `json:"call"`
	Result      json.RawMessage `json:"result"`
	Activity    json.RawMessage `json:"activity"`
	Message     string          `json:"message"`
	ContextMode string          `json:"context_mode"`
}

func nativeGapHostBoundary(raw []byte, parent, child, task, kind, expected string) (nativeGapBoundary, error) {
	var result nativeGapBoundary
	lines := bytes.Split(bytes.TrimSpace(raw), []byte{'\n'})
	matches := 0
	for ai, line := range lines {
		var e struct {
			Type    string
			Payload struct {
				Type     string
				ThreadID string `json:"thread_id"`
				TurnID   string `json:"turn_id"`
				Item     struct {
					ID, Type, Kind string
					Child          string `json:"agent_thread_id"`
					Path           string `json:"agent_path"`
				}
			}
		}
		if json.Unmarshal(line, &e) != nil || e.Type != "event_msg" || e.Payload.Type != "item_completed" || e.Payload.ThreadID != parent || e.Payload.Item.Type != "SubAgentActivity" || e.Payload.Item.Child != child || e.Payload.Item.Path != task {
			continue
		}
		want := "interacted"
		if kind == "spawn_agent" {
			want = "started"
		}
		if e.Payload.Item.Kind != want {
			continue
		}
		calls, outputs, ci, oi := 0, 0, -1, -1
		var call, output struct {
			Type    string
			Payload struct {
				Type, Name, Arguments, Output string
				CallID                        string `json:"call_id"`
				Metadata                      struct {
					TurnID string `json:"turn_id"`
				} `json:"internal_chat_message_metadata_passthrough"`
			}
		}
		var cb, ob []byte
		for i, l := range lines {
			var w = call
			if json.Unmarshal(l, &w) != nil || w.Type != "response_item" || w.Payload.CallID != e.Payload.Item.ID {
				continue
			}
			if w.Payload.Type == "function_call" {
				calls++
				call = w
				cb = l
				ci = i
			}
			if w.Payload.Type == "function_call_output" {
				outputs++
				output = w
				ob = l
				oi = i
			}
		}
		if calls != 1 || outputs != 1 || !(ci < ai && ai < oi) || e.Payload.TurnID == "" || call.Payload.Metadata.TurnID != e.Payload.TurnID || output.Payload.Metadata.TurnID != e.Payload.TurnID || call.Payload.Name[strings.LastIndex(call.Payload.Name, ".")+1:] != kind {
			continue
		}
		var args struct {
			Message, Target string
			ForkTurns       string `json:"fork_turns"`
			TaskName        string `json:"task_name"`
		}
		if json.Unmarshal([]byte(call.Payload.Arguments), &args) != nil {
			continue
		}
		var success map[string]any
		var resultFields struct{ Payload map[string]json.RawMessage }
		_ = json.Unmarshal(ob, &resultFields)
		if _, present := resultFields.Payload["output"]; !present {
			continue
		}
		if output.Payload.Output != "" && json.Unmarshal([]byte(output.Payload.Output), &success) != nil {
			continue
		}
		if kind == "spawn_agent" {
			if len(success) != 1 || success["task_name"] != task || task != "/root/"+args.TaskName {
				continue
			}
		} else if len(success) != 0 || (args.Target != child && args.Target != task) {
			continue
		}
		if args.Message != expected && !strings.HasPrefix(args.Message, "gAAAA") {
			continue
		}
		matches++
		result = nativeGapBoundary{Call: cb, Result: ob, Activity: line, Message: args.Message, ContextMode: args.ForkTurns}
	}
	if matches != 1 {
		return result, fmt.Errorf("missing or ambiguous successful %s call/result/child linkage", kind)
	}
	if result.Message != expected {
		return result, fmt.Errorf("%s plaintext unavailable in host export", kind)
	}
	if kind == "spawn_agent" && result.ContextMode != "none" {
		return result, fmt.Errorf("launch inherited or unknown context mode %q", result.ContextMode)
	}
	return result, nil
}

type nativeGapContextReport struct {
	Launch nativeGapBoundary `json:"launch"`
	Send   nativeGapBoundary `json:"send"`
	Gaps   []string          `json:"gaps"`
}

func nativeGapCaptureContext(r codexNativeLiveReceipt, runRoot string) nativeGapContextReport {
	report := nativeGapContextReport{}
	gap := func(err error) {
		if err != nil {
			report.Gaps = append(report.Gaps, err.Error())
		}
	}
	var challenge nativeGapChallenge
	raw, err := r.readEvidence(filepath.Join(runRoot, "controller", "challenge.json"))
	if err != nil || json.Unmarshal(raw, &challenge) != nil || challenge.Authority != "independent delayed fixture controller" {
		gap(fmt.Errorf("missing delayed fixture authorization (historical predeclared answer cannot qualify)"))
		return report
	}
	raw, err = r.readEvidence(filepath.Join(runRoot, "controller", "pre-question.json"))
	var snapshot map[string][]byte
	if err != nil || json.Unmarshal(raw, &snapshot) != nil || lifecycleDigest(raw) != challenge.SnapshotSHA256 {
		gap(fmt.Errorf("missing immutable pre-question snapshot"))
	} else {
		gap(nativeGapNoLeak(snapshot, challenge.Answer, challenge.PanicText))
	}
	question, err := r.readEvidence(filepath.Join(runRoot, "coordination", "w0-question-source.jsonl"))
	if err != nil || lifecycleDigest(bytes.TrimSpace(question)) != challenge.QuestionSHA256 {
		gap(fmt.Errorf("question provenance missing or changed"))
	}
	var qe struct{ Timestamp string }
	_ = json.Unmarshal(question, &qe)
	qt, e1 := time.Parse(time.RFC3339Nano, qe.Timestamp)
	at, e2 := time.Parse(time.RFC3339Nano, challenge.SelectedAt)
	if e1 != nil || e2 != nil || !at.After(qt) {
		gap(fmt.Errorf("answer not selected after genuine question"))
	}
	if challenge.ChildID != r.ChildID || !r.ChildIdentityCorroborated {
		gap(fmt.Errorf("answer child identity not corroborated"))
	}
	var reservation struct {
		Worker struct {
			Native struct {
				Prompt       string
				PromptSHA256 string `json:"prompt_sha256"`
			}
		}
	}
	raw, err = r.readEvidence(filepath.Join(runRoot, "coordination", "w0-reservation.json"))
	if err != nil || json.Unmarshal(raw, &reservation) != nil {
		gap(fmt.Errorf("runtime launch provenance missing"))
	}
	prompt := reservation.Worker.Native.Prompt
	if prompt == "" || lifecycleDigest([]byte(prompt)) != reservation.Worker.Native.PromptSHA256 || reservation.Worker.Native.PromptSHA256 != r.PromptSHA256 {
		gap(fmt.Errorf("runtime launch digest mismatch"))
	}
	for _, marker := range nativeGapMarkers {
		if !strings.Contains(prompt, marker) {
			gap(fmt.Errorf("required context missing: %s", marker))
		}
	}
	paths, _ := filepath.Glob(filepath.Join(runRoot, "home", ".codex", "sessions", "*", "*", "*", "*"+r.SessionID+".jsonl"))
	var parent []byte
	if len(paths) == 1 {
		parent, err = r.readEvidence(paths[0])
		gap(err)
	} else {
		gap(fmt.Errorf("unique parent capture missing"))
	}
	report.Launch, err = nativeGapHostBoundary(parent, r.SessionID, r.ChildID, r.ChildTaskPath, "spawn_agent", prompt)
	gap(err)
	var ctx struct {
		Result struct {
			Delivery *codexNativeContextDelivery `json:"context_delivery"`
		}
	}
	raw, err = r.readEvidence(filepath.Join(runRoot, "coordination", "w0-context.stdout.json"))
	if err != nil || json.Unmarshal(raw, &ctx) != nil || ctx.Result.Delivery == nil {
		gap(fmt.Errorf("Go-issued context envelope missing"))
		return report
	}
	delivery := ctx.Result.Delivery
	if delivery.ChildID != r.ChildID || !strings.Contains(delivery.Payload, challenge.Answer) || delivery.PayloadSHA256 != lifecycleDigest([]byte(delivery.Payload)) {
		gap(fmt.Errorf("stale/wrong-child/altered answer envelope"))
	}
	report.Send, err = nativeGapHostBoundary(parent, r.SessionID, r.ChildID, r.ChildTaskPath, "send_message", delivery.Payload)
	gap(err)
	// Another answer-bearing host message is an alternate causal path, even if
	// the intended exact send later succeeds. Never qualify that contamination.
	for _, line := range bytes.Split(parent, []byte{'\n'}) {
		var call struct {
			Type    string
			Payload struct{ Type, Name, Arguments string }
		}
		if json.Unmarshal(line, &call) != nil || call.Type != "response_item" || call.Payload.Type != "function_call" {
			continue
		}
		var args struct{ Message string }
		if json.Unmarshal([]byte(call.Payload.Arguments), &args) == nil && strings.Contains(args.Message, challenge.PanicText) && args.Message != delivery.Payload {
			gap(fmt.Errorf("answer disclosed through an alternate host message"))
		}
	}

	var se struct{ Timestamp string }
	_ = json.Unmarshal(report.Send.Activity, &se)
	st, e := time.Parse(time.RFC3339Nano, se.Timestamp)
	if e != nil || !st.After(at) {
		gap(fmt.Errorf("send did not follow answer selection"))
	}
	for _, name := range []string{"stale-answer", "wrong-child-answer"} {
		raw, err = r.readEvidence(filepath.Join(runRoot, "coordination", name+"-refusal.json"))
		var refusal struct {
			Exit          int `json:"exit_status"`
			Before, After map[string]string
		}
		if err != nil || json.Unmarshal(raw, &refusal) != nil || refusal.Exit == 0 || len(refusal.Before) == 0 || !reflect.DeepEqual(refusal.Before, refusal.After) {
			gap(fmt.Errorf("%s unchanged-journal refusal missing", name))
		}
	}
	raw, err = r.readEvidence(r.AttemptPath)
	var attempt buildAttemptRecord
	if err != nil || json.Unmarshal(raw, &attempt) != nil || len(attempt.WorkerRuns) != 1 || attempt.WorkerRuns[0].Native == nil || len(attempt.WorkerRuns[0].Native.ContextDeliveries) != 1 {
		gap(fmt.Errorf("completed-send ACK missing"))
	} else {
		saved := attempt.WorkerRuns[0].Native.ContextDeliveries[0]
		var activity struct {
			Timestamp string
			Payload   struct{ Item struct{ ID string } }
		}
		_ = json.Unmarshal(report.Send.Activity, &activity)
		if saved.Observation.SourceEventID != activity.Payload.Item.ID || strings.TrimPrefix(saved.Observation.SourceEventSHA256, "sha256:") != strings.TrimPrefix(lifecycleDigest(report.Send.Activity), "sha256:") || saved.Observation.ObservedAt != activity.Timestamp {
			gap(fmt.Errorf("ACK source does not match the actual completed host send"))
		}
		if !reflect.DeepEqual(saved.Delivery, *delivery) {
			gap(fmt.Errorf("ACK differs from sent immutable envelope"))
		}
		gap(validateCodexNativeContextReceipt(saved, attempt, attempt.WorkerRuns[0]))
	}
	// The harness-owned panic check cannot replace the child's required test.
	// Reuse the existing corroborated uncached Go-test predicate, rather than
	// calling a successful cat/echo of the challenge an answer-dependent check.
	if !r.ChecksPassed {
		gap(fmt.Errorf("child required uncached check not corroborated"))
	}
	return report
}

func TestCodexNativeGapContext(t *testing.T) {
	t.Run("delayed-controller", func(t *testing.T) {
		root := t.TempDir()
		repo := filepath.Join(root, "repository")
		home := filepath.Join(root, "home")
		coord := filepath.Join(root, "coord")
		for _, dir := range []string{repo, coord, filepath.Join(home, ".codex", "sessions", "2026", "09", "18")} {
			if err := os.MkdirAll(dir, 0700); err != nil {
				t.Fatal(err)
			}
		}
		answer := codexNativeDecisionAnswerRequest{SchemaVersion: 1, Binding: codexNativeDecisionBinding{ChildID: "child", QuestionID: "q"}, Question: "Which bound?"}
		path := filepath.Join(coord, "answer.json")
		if err := nativeGapWrite(path, answer); err != nil {
			t.Fatal(err)
		}
		if err := nativeGapAuthorize(root, repo, home, coord, path); err == nil {
			t.Fatal("controller authorized without question")
		}
		if _, err := os.Stat(filepath.Join(root, "controller", "challenge.json")); !os.IsNotExist(err) {
			t.Fatal("early answer material exists")
		}
		content, _ := json.Marshal(codexNativeQuestion{QuestionID: "q", Question: answer.Question})
		event := map[string]any{"type": "event_msg", "timestamp": time.Now().UTC().Format(time.RFC3339Nano), "payload": map[string]any{"type": "item_completed", "thread_id": "child", "item": map[string]any{"type": "AgentMessage", "phase": "final_answer", "content": string(content)}}}
		b, _ := json.Marshal(event)
		b = append(b, '\n')
		if err := os.WriteFile(filepath.Join(coord, "w0-question-source.jsonl"), b, 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(home, ".codex", "sessions", "2026", "09", "18", "child.jsonl"), b, 0600); err != nil {
			t.Fatal(err)
		}
		if err := nativeGapAuthorize(root, repo, home, coord, path); err != nil {
			t.Fatal(err)
		}
		raw, _ := os.ReadFile(filepath.Join(root, "controller", "challenge.json"))
		var challenge nativeGapChallenge
		if json.Unmarshal(raw, &challenge) != nil || len(challenge.PanicText) != 71 {
			t.Fatal("fresh challenge absent")
		}
		before, _ := os.ReadFile(filepath.Join(root, "controller", "pre-question.json"))
		var snapshot map[string][]byte
		_ = json.Unmarshal(before, &snapshot)
		if err := nativeGapNoLeak(snapshot, challenge.Answer, challenge.PanicText); err != nil {
			t.Fatal(err)
		}
		raw, _ = os.ReadFile(path)
		_ = json.Unmarshal(raw, &answer)
		if answer.Answer != challenge.Answer {
			t.Fatal("controller did not fill exact Go template")
		}
		if err := nativeGapAuthorize(root, repo, home, coord, path); err == nil {
			t.Fatal("controller replaced already authorized answer")
		}
	})
	t.Run("complete-runtime-context", func(t *testing.T) {
		root := setupExternalBuildAttemptTest(t)
		var fixture colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &fixture); err != nil {
			t.Fatal(err)
		}
		fixture.Plan.Phases[0].Tasks[0].Goal = "Fix Clamp integer bounds"
		if err := store.SaveJSON("COLONY_STATE.json", fixture); err != nil {
			t.Fatal(err)
		}
		liveSkillWrite(t, filepath.Join(root, "go.mod"), []byte("module example.invalid/contextfixture\n\ngo 1.23\n"))
		hub := t.TempDir()
		t.Setenv("AETHER_HUB_DIR", hub)
		// Installed skills compete for the same top-three budget in the live
		// fixture. Role-only fixture metadata must not silently drop its marker.
		for _, name := range []string{"a-competing", "b-competing", "c-competing"} {
			dir := filepath.Join(hub, "system", "skills", "colony", name)
			if err := os.MkdirAll(dir, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: "+name+"\ntype: colony\nagent_roles: [builder]\nworkflow_triggers: [build]\n---\nCompeting installed skill\n"), 0600); err != nil {
				t.Fatal(err)
			}
		}
		nativeGapPrepareContext(t, root)
		manifest := prepareBoundBuildManifestOnly(t, root)
		prompt, err := composeCodexNativePrompt(manifest, manifest.Dispatches[0], "fixture-launch")
		if err != nil {
			t.Fatal(err)
		}
		for _, marker := range nativeGapMarkers {
			if !strings.Contains(prompt.Prompt, marker) {
				t.Errorf("runtime dropped %s", marker)
			}
		}
	})

	t.Run("pre-question-leaks", func(t *testing.T) {
		answer := "delayed café 日本語 é\r\nexact  bytes"
		token := "unpredictable-unique-token"
		for _, where := range []string{"prompt", "history", "fixture.go", "coordinator.py", "tests.go"} {
			t.Run(where, func(t *testing.T) {
				if nativeGapNoLeak(map[string][]byte{where: []byte(answer)}, answer, token) == nil {
					t.Fatal("leak accepted")
				}
			})
		}
		if err := nativeGapNoLeak(map[string][]byte{"prompt": []byte("Ask first")}, answer, token); err != nil {
			t.Fatal(err)
		}
		if nativeGapNoLeak(map[string][]byte{"history": []byte(`{"text":"unpredictable-unique-\u0074oken"}`)}, answer, token) == nil {
			t.Fatal("escaped history leak accepted")
		}
	})
	t.Run("host-boundary", func(t *testing.T) {
		expected := "exact café 日本語 é\r\nbytes  "
		for _, kind := range []string{"spawn_agent", "send_message"} {
			for _, mode := range []string{"valid", "empty-result", "whitespace", "unicode", "wrong-child", "failed-send", "inspect-only", "missing-result", "duplicate-result", "encrypted", "inherited", "stale", "wrong-turn"} {
				t.Run(kind+"/"+mode, func(t *testing.T) {
					activityKind := "interacted"
					if kind == "spawn_agent" {
						activityKind = "started"
					}
					msg := expected
					if mode == "whitespace" {
						msg = strings.TrimSpace(msg)
					}
					if mode == "unicode" {
						msg = strings.ReplaceAll(msg, "é", "é")
					}
					if mode == "encrypted" {
						msg = "gAAAAopaque"
					}
					if mode == "stale" {
						msg = "old answer"
					}
					args := map[string]any{"message": msg, "task_name": "worker", "target": "/root/worker", "fork_turns": "none"}
					if mode == "inherited" {
						args["fork_turns"] = "all"
					}
					arguments, _ := json.Marshal(args)
					turn := map[string]string{"turn_id": "turn"}
					tool := kind
					if mode == "inspect-only" {
						tool = "inspect"
					}
					call := map[string]any{"type": "response_item", "payload": map[string]any{"type": "function_call", "name": tool, "call_id": "call", "arguments": string(arguments), "internal_chat_message_metadata_passthrough": turn}}
					child := "child"
					if mode == "wrong-child" {
						child = "other"
					}
					activity := map[string]any{"type": "event_msg", "payload": map[string]any{"type": "item_completed", "thread_id": "parent", "turn_id": "turn", "item": map[string]any{"type": "SubAgentActivity", "id": "call", "kind": activityKind, "agent_thread_id": child, "agent_path": "/root/worker"}}}
					output := "{}"
					if kind == "spawn_agent" {
						output = `{"task_name":"/root/worker"}`
					}
					if mode == "empty-result" {
						output = ""
					}
					if mode == "failed-send" {
						output = `{"error":"failed"}`
					}
					outTurn := turn
					if mode == "wrong-turn" {
						outTurn = map[string]string{"turn_id": "other"}
					}
					result := map[string]any{"type": "response_item", "payload": map[string]any{"type": "function_call_output", "call_id": "call", "output": output, "internal_chat_message_metadata_passthrough": outTurn}}
					events := []any{call, activity}
					if mode != "missing-result" {
						events = append(events, result)
					}
					if mode == "duplicate-result" {
						events = append(events, result)
					}
					var raw []byte
					for _, e := range events {
						b, _ := json.Marshal(e)
						raw = append(raw, append(b, '\n')...)
					}
					_, err := nativeGapHostBoundary(raw, "parent", "child", "/root/worker", kind, expected)
					valid := mode == "valid" || ((mode == "inherited" || mode == "empty-result") && kind == "send_message")
					if (err == nil) != valid {
						t.Fatalf("valid=%v err=%v", valid, err)
					}
				})
			}
		}
	})
	t.Run("historical-final165-contamination", func(t *testing.T) {
		r := codexNativeLiveReceipt{Scenario: "question"}
		report := nativeGapCaptureContext(r, t.TempDir())
		if len(report.Gaps) == 0 || !strings.Contains(report.Gaps[0], "historical predeclared") {
			t.Fatal("old answer fixture qualified")
		}
		s := nativeQualificationPrompt(r, "question")
		if strings.Contains(s, "predeclared harness response") || !strings.Contains(s, "fork_turns=none") {
			t.Fatal("contaminated/inherited launch instructions")
		}
	})
	t.Run("empty-required-brief", func(t *testing.T) {
		saveGlobals(t)
		store = nil
		for _, brief := range []string{"", " \n\t"} {
			if _, err := composeCodexNativePrompt(codexBuildManifest{Root: t.TempDir()}, codexBuildDispatch{Name: "worker", Brief: brief}, "launch"); err == nil {
				t.Fatal("empty required context accepted")
			}
		}
	})
	t.Run("scope-refusals-unchanged", func(t *testing.T) {
		for _, mode := range []string{"wrong-child", "stale-answer"} {
			t.Run(mode, func(t *testing.T) {
				_, decision := nativeDecisionFixture(t)
				answer := nativeAnswerForTest(decision)
				if mode == "wrong-child" {
					answer.Binding.ChildID += "-other"
				} else {
					answer.Binding.AttemptID += "-stale"
				}
				before := nativeDecisionStoreBytes(t)
				if _, _, err := answerCodexNativeDecision(answer, codexNativeDecisionHooks{}); err == nil {
					t.Fatal("invalid answer admitted")
				}
				if !reflect.DeepEqual(before, nativeDecisionStoreBytes(t)) {
					t.Fatal("refusal changed journals")
				}
			})
		}
	})
	t.Run("read-only-inspection-cannot-ack", func(t *testing.T) {
		_, reqs := nativeDecisionWorkersFixture(t, 1)
		view := nativeQuestionCommandForTest(t, reqs[0], "gap-question", "Which behavior?")
		answerNativeViewForTest(t, view, "fixture-only answer")
		before := nativeDecisionStoreBytes(t)
		result, err := nativeDecisionCommandForTest(t, "codex-native-worker", "context", "--request", nativeRequestPath(t, reqs[0]))
		if err != nil {
			t.Fatal(err)
		}
		if result["context_status"] != "awaiting_delivery" || !reflect.DeepEqual(before, nativeDecisionStoreBytes(t)) {
			t.Fatal("inspection acknowledged delivery")
		}
	})
	t.Run("coordinator-no-fallback", func(t *testing.T) {
		script := nativeQualificationCoordinator(nativeFixtureCoordinator, "question")
		for _, bad := range []string{"swap low and high first", "assert plaintext or", `value["answer"] = "Fixture-authorized`} {
			if strings.Contains(script, bad) {
				t.Fatalf("unsafe coordinator %q", bad)
			}
		}
		for _, required := range []string{"controller-ready.json", "assert plaintext,", "Missing successful host send result", "wrong-child-answer", "stale-answer"} {
			if !strings.Contains(script, required) {
				t.Fatalf("missing %q", required)
			}
		}
	})
}
