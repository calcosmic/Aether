package codex

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// The exact scout payload that failed three /ant-plan runs in
// M4L-AnalogWave-System on 27 July 2026, captured from the claude CLI session
// transcript. It must parse both on its own and inside the CLI's JSON
// envelope, which is how the runtime actually receives it.
func TestRealWorldScoutPayloadParses(t *testing.T) {
	inner, err := os.ReadFile("testdata/m4l_scout_payload_20260727.txt")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	t.Run("bare", func(t *testing.T) {
		claims, err := ParseWorkerOutput(string(inner))
		if err != nil {
			t.Fatalf("ParseWorkerOutput failed: %v", err)
		}
		assertScoutClaims(t, claims)
	})

	t.Run("inside_cli_envelope", func(t *testing.T) {
		raw, err := json.Marshal(map[string]interface{}{
			"is_error":    false,
			"subtype":     "success",
			"num_turns":   25,
			"session_id":  "5de89102-15e6-4bc3-bd20-a6dfd54e5efa",
			"stop_reason": "end_turn",
			"result":      string(inner),
		})
		if err != nil {
			t.Fatalf("marshal envelope: %v", err)
		}
		claims, err := parseHostedWorkerOutput("claude", combinedWorkerOutput(string(raw), ""))
		if err != nil {
			t.Fatalf("parseHostedWorkerOutput failed: %v", err)
		}
		assertScoutClaims(t, claims)
	})
}

// Parsing is not enough — the hosted result constructor must carry every
// content field through to WorkerResult. Artifacts and ScoutReport were
// silently dropped there, so a scout's research survived parsing and was then
// discarded before planning could read it.
func TestHostedWorkerResultCarriesAllClaimsContent(t *testing.T) {
	inner, err := os.ReadFile("testdata/m4l_scout_payload_20260727.txt")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	claims, err := ParseWorkerOutput(string(inner))
	if err != nil {
		t.Fatalf("ParseWorkerOutput failed: %v", err)
	}
	if len(claims.ScoutReport) == 0 {
		t.Fatal("fixture has no scout_report; test cannot prove the copy")
	}

	config := WorkerConfig{WorkerName: "seek-69", Caste: "scout", TaskID: "plan-0"}
	hosted := hostedWorkerResultFromClaims(config, claims, 0, "")
	if len(hosted.ScoutReport) == 0 {
		t.Fatal("ScoutReport dropped between claims and WorkerResult")
	}
	if len(claims.Artifacts) > 0 && len(hosted.Artifacts) == 0 {
		t.Fatal("Artifacts dropped between claims and WorkerResult")
	}
	if len(hosted.Blockers) != len(claims.Blockers) {
		t.Fatal("Blockers dropped between claims and WorkerResult")
	}
}

func assertScoutClaims(t *testing.T, claims workerClaims) {
	t.Helper()
	if claims.Status != "completed" {
		t.Fatalf("Status = %q, want completed", claims.Status)
	}
	if len(strings.TrimSpace(claims.Summary)) < 100 {
		t.Fatalf("Summary too short to be the real research result: %q", claims.Summary)
	}
	if len(claims.Blockers) == 0 {
		t.Fatal("Blockers empty — the planning risks the scout found were dropped")
	}
	if len(claims.Handoff.NextWorkerInstructions) == 0 {
		t.Fatal("NextWorkerInstructions empty — the scalar handoff field was dropped")
	}
	if len(claims.ScoutReport) == 0 {
		t.Fatal("ScoutReport empty — the research payload the planner consumes was lost")
	}
}
