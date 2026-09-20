package codex

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestAuditorProducerSchemaAllowsReviewWithoutOpeningArtifacts(t *testing.T) {
	for _, caste := range []string{"auditor", "gatekeeper", "builder"} {
		t.Run(caste, func(t *testing.T) {
			raw, err := WorkerOutputSchema(WorkerConfig{Caste: caste})
			if err != nil {
				t.Fatal(err)
			}
			var doc map[string]any
			if err := json.Unmarshal(raw, &doc); err != nil {
				t.Fatal(err)
			}
			artifact := doc["properties"].(map[string]any)["artifacts"]
			compiler := jsonschema.NewCompiler()
			if err := compiler.AddResource("urn:test:artifacts", artifact); err != nil {
				t.Fatal(err)
			}
			schema, err := compiler.Compile("urn:test:artifacts")
			if err != nil {
				t.Fatal(err)
			}
			value := map[string]any{"research_file": nil, "survey_file": nil, "plan_file": nil}
			if caste == "auditor" {
				value["review"] = map[string]any{"overall_score": float64(93), "findings": []any{}}
			}
			if caste == "gatekeeper" {
				value["review"] = map[string]any{"findings": []any{}}
			}
			if err := schema.Validate(value); err != nil {
				t.Fatal(err)
			}
			value["unrecognized"] = "must reject"
			if schema.Validate(value) == nil {
				t.Fatal("arbitrary artifact accepted")
			}
			delete(value, "unrecognized")
			if caste == "builder" {
				value["review"] = nil
				if schema.Validate(value) == nil {
					t.Fatal("builder acquired review:null")
				}
				return
			}
			if caste == "gatekeeper" {
				value["review"] = map[string]any{"overall_score": float64(93), "findings": []any{}}
				if schema.Validate(value) == nil {
					t.Fatal("gatekeeper acquired Auditor score")
				}
				value["review"] = map[string]any{}
				if schema.Validate(value) == nil {
					t.Fatal("gatekeeper review without findings accepted")
				}
				value["review"] = nil
				if err := schema.Validate(value); err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(renderResponseContract(WorkerConfig{Caste: caste}), "Do not include overall_score") {
					t.Fatal("gatekeeper guidance requires score")
				}
				return
			}
			for _, bad := range []any{map[string]any{"overall_score": float64(101), "findings": []any{}}, map[string]any{"findings": []any{}}, map[string]any{"overall_score": float64(90), "findings": []any{map[string]any{"severity": "made-up", "title": "bad"}}}} {
				value["review"] = bad
				if schema.Validate(value) == nil {
					t.Fatalf("invalid review accepted: %v", bad)
				}
			}
			value["review"] = nil
			if err := schema.Validate(value); err != nil {
				t.Fatalf("blocked auditor cannot honestly omit score: %v", err)
			}
			if !strings.Contains(renderResponseContract(WorkerConfig{Caste: caste}), "never invent a score") {
				t.Fatal("missing truthful review guidance")
			}
		})
	}
}

func TestWorkerToolCountPresenceSurvivesParsingAndHostedResult(t *testing.T) {
	for _, tc := range []struct {
		field string
		known bool
		count int
	}{{"", false, 0}, {`,"tool_count":null`, false, 0}, {`,"tool_count":0`, true, 0}, {`,"tool_count":3`, true, 3}} {
		claims, err := ParseWorkerOutput(`{"ant_name":"Worker","caste":"builder","task_id":"1.1","status":"completed","summary":"done"` + tc.field + `}`)
		if err != nil {
			t.Fatal(err)
		}
		result := hostedWorkerResultFromClaims(WorkerConfig{}, claims, time.Second, "")
		if result.ToolCountReported != tc.known || result.ToolCount != tc.count {
			t.Fatalf("%s: %+v", tc.field, result)
		}
	}
	fake, err := (&FakeInvoker{}).Invoke(context.Background(), WorkerConfig{WorkerName: "Fake", Caste: "builder", TaskID: "1.1"})
	if err != nil || !fake.ToolCountReported || fake.ToolCount != 0 {
		t.Fatalf("fake zero provenance lost: %+v %v", fake, err)
	}
}

func TestWorkerHeartbeatDoesNotInventProviderActivity(t *testing.T) {
	var events []WorkerProgressEvent
	s := newWorkerRunningSignal(func(e WorkerProgressEvent) { events = append(events, e) })
	s.Pulse("worker heartbeat observed")
	if s.Observed() || len(events) != 1 || events[0].Source != "process_heartbeat" {
		t.Fatalf("parent heartbeat became provider activity: %+v", events)
	}
	s.Report("worker output observed")
	if !s.Observed() || len(events) != 2 || events[1].Source != "provider_output" {
		t.Fatalf("provider output lost after heartbeat: %+v", events)
	}
}

func TestCodexTimeoutPreservesObservedWorkWithoutCompletion(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fake process")
	}
	for _, noisy := range []bool{false, true} {
		name := "silent"
		if noisy {
			name = "observed"
		}
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			agent := filepath.Join(dir, "agent.toml")
			if err := os.WriteFile(agent, []byte("name = 'aether-builder'\ndescription = 'Builder'\ndeveloper_instructions = 'Do work'\n"), 0600); err != nil {
				t.Fatal(err)
			}
			script := "#!/bin/sh\nif [ \"$1 $2\" = \"login status\" ]; then echo 'Logged in'; exit 0; fi\ncat >/dev/null\n"
			if noisy {
				script += "printf draft > '" + filepath.Join(dir, "draft.txt") + "'\n" + `
printf '%s\n' '{"type":"item.started","item":{"id":"call-1","type":"command_execution"}}' '{"type":"item.completed","item":{"id":"call-1","type":"command_execution"}}' '{"type":"item.completed","item":{"id":"message-1","type":"agent_message"}}' '{"type":"item.started","item":{"id":"call-2","type":"file_change"}}'
printf '%s' '{"type":"item.started","item":'
printf '%s\n' 'ghp_fixture_token' >&2
`
			}
			script += "sleep 5\n"
			bin := filepath.Join(dir, "codex")
			if err := os.WriteFile(bin, []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			invoker := NewRealInvoker()
			invoker.binaryName = bin
			var events []DispatchLifecycleEvent
			dispatched := invokeDispatch(context.Background(), invoker, WorkerDispatch{Root: dir, AgentTOMLPath: agent, WorkerName: "Worker", Caste: "builder", TaskID: "1.1", TaskBrief: "Fixture", Timeout: 300 * time.Millisecond}, func(e DispatchLifecycleEvent) { events = append(events, e) })
			if dispatched.WorkerResult == nil {
				t.Fatalf("dispatch dropped timeout evidence: %+v", dispatched)
			}
			result := *dispatched.WorkerResult
			providerEvent := false
			for _, e := range events {
				if e.Source == "provider_output" {
					providerEvent = true
				}
			}
			if providerEvent != noisy {
				t.Fatalf("dispatch provider provenance = %v, noisy = %v", providerEvent, noisy)
			}
			if result.Status != "timeout" || result.ToolCountReported || len(result.TaskReceipts) != 0 || len(result.FilesCreated) != 0 {
				t.Fatalf("timeout inferred completion/count: %+v", result)
			}
			want := 0
			if noisy {
				want = 2
			}
			if result.ObservedToolCalls != want {
				t.Fatalf("observed calls %d want %d", result.ObservedToolCalls, want)
			}
			if result.DiagnosticPath == "" {
				t.Fatal("no retained diagnostics")
			}
			raw, err := os.ReadFile(filepath.Join(dir, result.DiagnosticPath))
			if err != nil {
				t.Fatal(err)
			}
			var diagnostic map[string]any
			if err := json.Unmarshal(raw, &diagnostic); err != nil {
				t.Fatal(err)
			}
			stdout, hasStdout := diagnostic["stdout"].(string)
			if diagnostic["failure_mode"] != "timeout" || !hasStdout || !strings.Contains(result.RawOutput, stdout) {
				t.Fatalf("diagnostic not retained: %s", raw)
			}
			if strings.Contains(string(raw), "ghp_fixture_token") || strings.Contains(result.RawOutput, "ghp_fixture_token") {
				t.Fatal("timeout retained unredacted credential-shaped diagnostic")
			}
			if noisy {
				if _, err := os.Stat(filepath.Join(dir, "draft.txt")); err != nil {
					t.Fatal("uncredited draft lost", err)
				}
			}
		})
	}
}
