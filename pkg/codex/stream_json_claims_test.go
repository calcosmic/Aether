package codex

import (
	"encoding/json"
	"strings"
	"testing"
)

// streamJSONLine marshals one NDJSON event the way `claude -p --output-format
// stream-json --verbose` emits them.
func streamJSONLine(t *testing.T, event map[string]interface{}) string {
	t.Helper()
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal stream event: %v", err)
	}
	return string(data)
}

func assistantTextEvent(t *testing.T, text string) string {
	t.Helper()
	return streamJSONLine(t, map[string]interface{}{
		"type": "assistant",
		"message": map[string]interface{}{
			"role":    "assistant",
			"content": []interface{}{map[string]interface{}{"type": "text", "text": text}},
		},
	})
}

const observedProbeClaims = `{"ant_name":"Excavat-92","caste":"probe","task_id":"continue-review-probe","status":"completed","summary":"Independently verified: npm test passes 5/5, claims match evidence, no scope creep.","files_created":[],"files_modified":[],"tests_written":[],"tool_count":9,"blockers":[],"spawns":[],"artifacts":{"research_file":null,"survey_file":null,"plan_file":null},"handoff":{"changed_files":[],"commands_run":["npm test"],"verification_status":"pass","known_failures":[],"open_decisions":[],"assumptions":[],"next_worker_instructions":[],"do_not_repeat":[],"freshness":"2026-08-04T16:29:41Z"}}`

// The real tail the probe appended after its claims during the v1.0.47
// acceptance run. With --output-format json this prose was the ENTIRE
// envelope result, so the claims were unrecoverable and continue blocked.
const observedProseTail = `Advancing the lifecycle (` + "`aether continue`" + `) or pausing the colony is an orchestrator-level action, not something a Probe subagent invokes directly. My final JSON response above already contains the verdict for whichever agent is driving the workflow.

**Probe verdict: PASS — safe to advance.**

If you want the lifecycle actually advanced now, that requires running ` + "`aether continue`" + `. Let me know if you'd like anything further.`

// TestHostedParseRecoversClaimsFromEarlierAssistantTurn is the WP-1 gate. It
// reproduces the exact v1.0.47 acceptance failure: the worker emitted valid
// claims in an earlier assistant turn, kept talking, and the final result
// event carried only prose. Under stream-json the claims must still be found.
func TestHostedParseRecoversClaimsFromEarlierAssistantTurn(t *testing.T) {
	lines := []string{
		streamJSONLine(t, map[string]interface{}{"type": "system", "subtype": "init", "session_id": "34588cdc"}),
		assistantTextEvent(t, "Let me verify the builder's claims by running the test suite."),
		assistantTextEvent(t, observedProbeClaims),
		assistantTextEvent(t, observedProseTail),
		streamJSONLine(t, map[string]interface{}{
			"type":        "result",
			"subtype":     "success",
			"is_error":    false,
			"session_id":  "34588cdc",
			"stop_reason": "end_turn",
			"result":      observedProseTail,
		}),
	}
	raw := combinedWorkerOutput(strings.Join(lines, "\n"), "")

	claims, err := parseHostedWorkerOutput("claude", raw)
	if err != nil {
		t.Fatalf("claims emitted in an earlier assistant turn were not recovered: %v", err)
	}
	if claims.Status != "completed" {
		t.Fatalf("status = %q, want completed", claims.Status)
	}
	if claims.AntName != "Excavat-92" {
		t.Fatalf("ant_name = %q, want Excavat-92", claims.AntName)
	}
	if claims.Handoff.VerificationStatus != "pass" {
		t.Fatalf("handoff verification_status = %q, want pass", claims.Handoff.VerificationStatus)
	}
}

// A tool_use block echoing claim-shaped JSON must not beat the worker's real
// terminal claims: candidates are scanned newest-first and schema-validated.
func TestHostedParseIgnoresDecoyClaimsInToolUse(t *testing.T) {
	decoy := streamJSONLine(t, map[string]interface{}{
		"type": "assistant",
		"message": map[string]interface{}{
			"role": "assistant",
			"content": []interface{}{map[string]interface{}{
				"type":  "tool_use",
				"name":  "Write",
				"input": map[string]interface{}{"content": `{"ant_name":"DECOY","caste":"probe","status":"failed","summary":"decoy"}`},
			}},
		},
	})
	lines := []string{
		decoy,
		assistantTextEvent(t, observedProbeClaims),
		streamJSONLine(t, map[string]interface{}{"type": "result", "subtype": "success", "result": observedProseTail}),
	}

	claims, err := parseHostedWorkerOutput("claude", combinedWorkerOutput(strings.Join(lines, "\n"), ""))
	if err != nil {
		t.Fatalf("parseHostedWorkerOutput failed: %v", err)
	}
	if claims.AntName == "DECOY" || claims.Status == "failed" {
		t.Fatalf("decoy tool_use payload won over the real terminal claims: %+v", claims)
	}
}

// The plain-json envelope shape must keep working — downstream repos and the
// OpenCode path still produce it.
func TestHostedParseStillHandlesSingleEnvelope(t *testing.T) {
	raw, err := json.Marshal(map[string]interface{}{
		"type": "result", "subtype": "success", "is_error": false, "result": observedProbeClaims,
	})
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	claims, err := parseHostedWorkerOutput("claude", combinedWorkerOutput(string(raw), ""))
	if err != nil {
		t.Fatalf("single-envelope parsing regressed: %v", err)
	}
	if claims.Status != "completed" {
		t.Fatalf("status = %q, want completed", claims.Status)
	}
}

// The reason the transport had to change: under the old plain-json format the
// CLI returns ONE envelope whose result is the final assistant text only. When
// the worker put its claims in an earlier turn, they never reached stdout at
// all — no parser could recover them. This pins that premise, so nobody
// "simplifies" the transport back to json.
func TestSingleEnvelopeWithProseOnlyResultCannotBeRecovered(t *testing.T) {
	raw, err := json.Marshal(map[string]interface{}{
		"type": "result", "subtype": "success", "is_error": false, "result": observedProseTail,
	})
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	if _, err := parseHostedWorkerOutput("claude", combinedWorkerOutput(string(raw), "")); err == nil {
		t.Fatal("expected prose-only plain-json envelope to be unrecoverable; if this now parses, the premise for stream-json changed")
	}
}

// A provider error arriving as a stream-json event must still be classified as
// a provider error, not mislabelled as a claims parse failure.
func TestProviderErrorEnvelopeDetectedInStreamJSON(t *testing.T) {
	lines := []string{
		streamJSONLine(t, map[string]interface{}{"type": "system", "subtype": "init"}),
		streamJSONLine(t, map[string]interface{}{
			"type": "error",
			"error": map[string]interface{}{
				"name": "APIError",
				"data": map[string]interface{}{
					"message":    "Overloaded",
					"statusCode": 529,
				},
			},
		}),
		assistantTextEvent(t, "some earlier narration"),
	}
	env, ok := detectProviderErrorEnvelope(combinedWorkerOutput(strings.Join(lines, "\n"), ""))
	if !ok {
		t.Fatal("provider error event in stream-json was not detected")
	}
	if env.StatusCode != 529 {
		t.Fatalf("provider error status = %d, want 529", env.StatusCode)
	}
	if !strings.Contains(env.Message, "Overloaded") {
		t.Fatalf("provider error message = %q, want the provider's own text", env.Message)
	}
}

// TestClaudeDispatchUsesStreamJSON pins the transport: plain json loses claims
// emitted before the final turn.
func TestClaudeDispatchUsesStreamJSON(t *testing.T) {
	args := claudeBaseWorkerArgs("prompt", `{"type":"object"}`, "aether-probe")
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--output-format stream-json") {
		t.Fatalf("claude dispatch does not request stream-json: %v", args)
	}
	if !strings.Contains(joined, "--verbose") {
		t.Fatalf("stream-json requires --verbose: %v", args)
	}
	for _, required := range []string{"--json-schema", "--agent", "-p"} {
		if !strings.Contains(joined, required) {
			t.Fatalf("claude dispatch lost %s: %v", required, args)
		}
	}
}

// TestResponseContractForbidsTrailingProseAndNextStepAdvice pins the prompt
// side of the same defect: the worker was told to return JSON but never told
// to stop talking afterwards, and its trailing next-command advice is exactly
// what the observed failure appended.
func TestResponseContractForbidsTrailingProseAndNextStepAdvice(t *testing.T) {
	contract := renderResponseContract(WorkerConfig{Root: ".", Caste: "probe"})
	for _, want := range []string{
		"no prose before it, no prose after it",
		"Do not emit the JSON and then keep working",
		"Do not tell the user which command to run next",
	} {
		if !strings.Contains(contract, want) {
			t.Fatalf("response contract missing %q:\n%s", want, contract)
		}
	}
}

// TestBackgroundTaskNoticeIsNeverTheWorkersReport reproduces Phase 210
// blocker 5 (French Basics, 2026-09-26): builder Anvil-7 finished with valid
// claims (status "code_written"), but it had started a test run in the
// background, and when its session ended Claude Code emitted a
// task_notification for that run. The notice carries task_id, status and
// summary, so it passed isWorkerClaimsMap, and as the last candidate it won:
// the build saw status "stopped" with the shell command as its summary, and
// the wave failed. The notice shape is Claude Code 2.1.283's own
// (type "system", subtype "task_notification", task_id, tool_use_id, status,
// output_file, summary).
func TestBackgroundTaskNoticeIsNeverTheWorkersReport(t *testing.T) {
	realClaims := `{"ant_name":"Anvil-7","caste":"builder","task_id":"1.1,1.2,1.3","status":"code_written","summary":"Wrote docs/FRENCH_BASICS_RULES.md and docs/FRENCH_BASICS_BLOCKS.md.","files_created":["docs/FRENCH_BASICS_RULES.md","docs/FRENCH_BASICS_BLOCKS.md"],"files_modified":[],"tests_written":[],"blockers":[],"spawns":[]}`
	lines := []string{
		streamJSONLine(t, map[string]interface{}{"type": "system", "subtype": "init", "session_id": "9d0e8fbe"}),
		assistantTextEvent(t, realClaims),
		streamJSONLine(t, map[string]interface{}{
			"type": "result", "subtype": "success", "is_error": false, "session_id": "9d0e8fbe", "result": realClaims,
		}),
		streamJSONLine(t, map[string]interface{}{
			"type":        "system",
			"subtype":     "task_notification",
			"task_id":     "bmrqr58fm",
			"tool_use_id": "toolu_01B2Dmfbjsaca3uBTEzV4K5c",
			"status":      "stopped",
			"output_file": "/private/tmp/claude-501/tasks/bmrqr58fm.output",
			"summary":     `cd "/Users/callumcowie/WORKSPACE/French Fluency"; python3 -m pytest -q 2>&1 | tail -2`,
			"session_id":  "9d0e8fbe",
		}),
	}
	claims, err := parseHostedWorkerOutput("claude", combinedWorkerOutput(strings.Join(lines, "\n"), ""))
	if err != nil {
		t.Fatalf("the worker's real claims were not recovered: %v", err)
	}
	if claims.Status != "completed" || claims.AntName != "Anvil-7" {
		t.Fatalf("a background-task notice replaced the worker's report: status=%q ant=%q blockers=%v", claims.Status, claims.AntName, claims.Blockers)
	}
}
