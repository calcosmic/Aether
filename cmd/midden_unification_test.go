package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestOneMiddenEntryReachesAllFourConsumers proves, by direct execution (not
// by reading code), that a single appendMiddenEntry write is visible through
// every ROADMAP-named consumer: autopilot's pause-condition check,
// colony-prime's context-capsule path, memory-health's failure count, and
// immune's auto-scar detector.
//
// A note on "colony-prime's context capsule": 188-01-PLAN.md's own interface
// notes cited cmd/context.go's buildContextCapsuleOutput() as this consumer.
// That attribution is wrong -- buildContextCapsuleOutput (cmd/context.go,
// the function spanning roughly lines 434-667) has no midden involvement at
// all: its ContextCapsuleOutput return type has no Midden field, and none of
// the five sections it assembles (state, signals, decisions, risks,
// recent_narrative) ever reads midden.json in any form. The actual code with
// the broken nested-path read -- the one this plan's Task 2 fixed -- lives in
// the sibling `pr-context` command (prContextCmd's RunE, same file, its own
// numbered "9. midden" section), which is live (see
// cmd/integration_test.go's TestIntegrationPRContext) and matches this
// repo's own pre-existing acknowledgment in
// cmd/colony_prime_audit_test.go's TestColonyPrimeAAC005Audit: midden data
// "goes through the context capsule path, not colony-prime". This test
// exercises the real path (`pr-context`) rather than the misattributed one,
// per this repo's Definition of Done ("prefer invariants over name checks").
func TestOneMiddenEntryReachesAllFourConsumers(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	setupHubDir(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Minimal colony state: good enough for checkAutopilotPauseConditions and
	// pr-context to run without erroring, not a full build/continue fixture.
	goal := "midden unification test"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Testing", Status: "in_progress"},
			},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("seed COLONY_STATE.json: %v", err)
	}

	const wantMessage = "adversarial probe found a real bug"
	if err := appendMiddenEntry(s, "chaos", "test", wantMessage, []string{"critical"}); err != nil {
		t.Fatalf("appendMiddenEntry: %v", err)
	}

	// 1. autopilot's own pause-condition check.
	if reason := checkAutopilotPauseConditions(); reason != "critical_chaos_findings" {
		t.Errorf("[autopilot regressed] checkAutopilotPauseConditions() = %q, want %q -- autopilot did not see the entry written through appendMiddenEntry",
			reason, "critical_chaos_findings")
	}

	// 2. colony-prime's context-capsule path (the real `pr-context` command;
	// see the function-level doc comment above for why this is the correct
	// target instead of buildContextCapsuleOutput).
	rootCmd.SetArgs([]string{"pr-context"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("[context-capsule regressed] pr-context returned error: %v", err)
	}
	capsuleEnvelope := parseEnvelope(t, buf.String())
	capsuleResult, ok := capsuleEnvelope["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("[context-capsule regressed] pr-context result is not a map: %v", capsuleEnvelope)
	}
	middenSection, ok := capsuleResult["midden"].(map[string]interface{})
	if !ok {
		t.Fatalf("[context-capsule regressed] pr-context result has no midden section: %v", capsuleResult)
	}
	if count, _ := middenSection["count"].(float64); count < 1 {
		t.Errorf("[context-capsule regressed] midden.count = %v, want >= 1 -- context capsule did not see the entry written through appendMiddenEntry",
			middenSection["count"])
	}
	items, _ := middenSection["items"].([]interface{})
	foundInCapsule := false
	for _, item := range items {
		if text, ok := item.(string); ok && strings.Contains(text, wantMessage) {
			foundInCapsule = true
			break
		}
	}
	if !foundInCapsule {
		t.Errorf("[context-capsule regressed] midden.items = %v, want an item containing %q", items, wantMessage)
	}

	// 3. memory-health's failure count.
	summary := loadMemoryHealthSummary(s)
	if summary.RecentFailures < 1 {
		t.Errorf("[memory-health regressed] loadMemoryHealthSummary(s).RecentFailures = %d, want >= 1 -- memory-health did not see the entry written through appendMiddenEntry",
			summary.RecentFailures)
	}

	// 4. immune's auto-scar detector -- also proves the entry.Message field
	// (not the nonexistent entry["description"]) is now readable.
	buf.Reset()
	rootCmd.SetArgs([]string{"immune-auto-scar"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("[immune regressed] immune-auto-scar returned error: %v", err)
	}
	scarEnvelope := parseEnvelope(t, buf.String())
	scarResult, ok := scarEnvelope["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("[immune regressed] immune-auto-scar result is not a map: %v", scarEnvelope)
	}
	detected, _ := scarResult["detected"].(float64)
	if detected < 1 {
		t.Errorf("[immune regressed] immune-auto-scar detected = %v, want >= 1 -- either the path is unreadable again or entry.Message did not reach the scar detector",
			scarResult["detected"])
	}
}
