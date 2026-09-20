package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestOneMiddenEntryReachesRuntimeConsumers proves, by direct execution (not
// by reading code), that a single appendMiddenEntry write is visible through
// the retained runtime consumers: colony-prime's context-capsule path,
// memory-health's failure count, and immune's auto-scar detector.
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
//
// 188-07 update: 188-VERIFICATION.md's Gap 1 went further than this test's
// own framing above -- it found that pr-context is not merely "not the
// function the plan's interface notes cited", it is also not on any live
// worker-dispatch path at all (grep across cmd/*.go found zero callers
// outside its own file, tests, and backups), while the function that IS on
// every live path (resolveCodexWorkerContext -> buildColonyPrimeOutput, the
// real "colony-prime") had zero midden-reading code, before or after this
// phase's original five plans. That is now fixed directly in
// buildColonyPrimeOutputOpts (cmd/colony_prime_context.go) -- see
// TestOneMiddenEntryReachesColonyPrimeCapsule below for the proof on the
// REAL path. This test is kept unchanged: pr-context's own midden section is
// still real, live-adjacent (used by CI's TestIntegrationPRContext), and
// worth continued regression coverage in its own right.
func TestOneMiddenEntryReachesRuntimeConsumers(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	setupHubDir(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Minimal colony state: good enough for pr-context to run without erroring,
	// not a full build/continue fixture.
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

	// 1. colony-prime's context-capsule path (the real `pr-context` command;
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

	// 2. memory-health's failure count.
	summary := loadMemoryHealthSummary(s)
	if summary.RecentFailures < 1 {
		t.Errorf("[memory-health regressed] loadMemoryHealthSummary(s).RecentFailures = %d, want >= 1 -- memory-health did not see the entry written through appendMiddenEntry",
			summary.RecentFailures)
	}

	// 3. immune's auto-scar detector -- also proves the entry.Message field
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

// TestOneMiddenEntryReachesColonyPrimeCapsule closes 188-VERIFICATION.md's
// Gap 1: resolveCodexWorkerContext() / buildColonyPrimeOutput() -- the
// function every live build, continue, colonize, plan, seal, and swarm
// worker dispatch actually calls for its context capsule -- previously had
// zero midden-reading code, before or after this phase's original five
// plans. The verifier proved this with a throwaway probe (written and
// deleted during verification): after appendMiddenEntry wrote a recognizable
// failure message, resolveCodexWorkerContext() returned a capsule with no
// trace of it. This test is that same probe shape, made permanent: write a
// failure record through the canonical shared writer, build the REAL
// capsule, and assert the record's message reaches the assembled prompt text
// -- exactly once (Phase 190's one-home law: cmd/build_pheromone_190_05_test.go
// -- no section may be delivered to a worker twice).
func TestOneMiddenEntryReachesColonyPrimeCapsule(t *testing.T) {
	saveGlobals(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "colony-prime midden capsule test"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Testing", Status: colony.PhaseInProgress},
			},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("seed COLONY_STATE.json: %v", err)
	}

	const wantMessage = "SENTINEL-188-07 adversarial probe found a real bug in colony-prime"
	if err := appendMiddenEntry(s, "chaos", "test", wantMessage, []string{"critical"}); err != nil {
		t.Fatalf("appendMiddenEntry: %v", err)
	}

	capsule := resolveCodexWorkerContext()
	if !strings.Contains(capsule, wantMessage) {
		t.Fatalf("[colony-prime regressed] resolveCodexWorkerContext() capsule does not carry the entry written through appendMiddenEntry -- Gap 1 reopened:\n%s", capsule)
	}
	if n := strings.Count(capsule, wantMessage); n != 1 {
		t.Fatalf("[one-home violated] colony-prime's capsule carries the failure record %d times, want exactly 1:\n%s", n, capsule)
	}
}

// TestMiddenCapsuleSectionOmittedWhenNoFailures proves the new section does
// not print an empty "Recent Failures" heading (or otherwise appear) when
// midden.json has no entries at all -- the same "don't show empty sections"
// discipline every other colony-prime section already follows.
func TestMiddenCapsuleSectionOmittedWhenNoFailures(t *testing.T) {
	saveGlobals(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "colony-prime midden capsule empty test"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Testing", Status: colony.PhaseInProgress},
			},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("seed COLONY_STATE.json: %v", err)
	}

	capsule := resolveCodexWorkerContext()
	if strings.Contains(capsule, "Recent Failures") {
		t.Errorf("capsule rendered a Recent Failures section with no midden entries at all:\n%s", capsule)
	}
}

// TestMiddenCapsuleSectionOmitsAcknowledgedEntries proves an acknowledged
// failure (one already handled, per midden-acknowledge) does not keep
// consuming capsule budget forever -- only unacknowledged entries are
// eligible for inclusion.
func TestMiddenCapsuleSectionOmitsAcknowledgedEntries(t *testing.T) {
	saveGlobals(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "colony-prime midden acknowledged test"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Testing", Status: colony.PhaseInProgress},
			},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("seed COLONY_STATE.json: %v", err)
	}

	const wantMessage = "SENTINEL-188-07-ACKNOWLEDGED already handled, should not resurface"
	if err := appendMiddenEntry(s, "chaos", "test", wantMessage, nil); err != nil {
		t.Fatalf("appendMiddenEntry: %v", err)
	}

	var mf colony.MiddenFile
	if err := s.LoadJSON("midden.json", &mf); err != nil {
		t.Fatalf("load midden.json: %v", err)
	}
	if len(mf.Entries) != 1 {
		t.Fatalf("expected exactly 1 seeded midden entry, got %d", len(mf.Entries))
	}
	acked := true
	mf.Entries[0].Acknowledged = &acked
	if err := s.SaveJSON("midden.json", mf); err != nil {
		t.Fatalf("save acknowledged midden.json: %v", err)
	}

	capsule := resolveCodexWorkerContext()
	if strings.Contains(capsule, wantMessage) {
		t.Errorf("capsule surfaced an already-acknowledged midden entry:\n%s", capsule)
	}
}
