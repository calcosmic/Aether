package cmd

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// These tests prove CONTEXT-06: an approved charter's governance rules reach
// every worker through colony-prime, survive budget trimming, and pass
// through the same prompt-integrity pipeline as every other section (D-09's
// security guardrail). Before this test file, cmd/colony_prime_context.go
// contained zero references to state.Charter -- a user could approve
// "Linting: golangci-lint. Testing: pytest" and no worker would ever learn
// of it.

// TestColonyPrimeIncludesCharterGovernance is Test 1: a colony whose
// state.Charter.Governance is populated produces colony-prime output
// containing that governance text.
func TestColonyPrimeIncludesCharterGovernance(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "charter governance test"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase One", Status: "in_progress"},
			},
		},
		Charter: &colony.Charter{
			Governance: "Linting: golangci-lint. Testing: pytest",
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"colony-prime"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("colony-prime returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	result := envelope["result"].(map[string]interface{})
	promptSection, ok := result["prompt_section"].(string)
	if !ok {
		t.Fatalf("result.prompt_section not a string: %T", result["prompt_section"])
	}

	if !strings.Contains(promptSection, "Linting: golangci-lint") {
		t.Fatalf("prompt_section missing charter governance text:\n%s", promptSection)
	}
	if !strings.Contains(promptSection, "Testing: pytest") {
		t.Fatalf("prompt_section missing charter testing governance text:\n%s", promptSection)
	}

	ledger := result["ledger"].(map[string]interface{})
	included, ok := ledger["included"].([]interface{})
	if !ok {
		t.Fatalf("ledger.included not an array: %T", ledger["included"])
	}
	found := false
	for _, raw := range included {
		entry := raw.(map[string]interface{})
		if entry["name"] == "charter" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("charter section not found in ledger included items: %v", included)
	}
}

// TestColonyPrimeCharterSurvivesTrimming is Test 2: charter content is
// present in the output ledger as an included (not trimmed) section even
// when other sections are trimmed for budget (D-01).
func TestColonyPrimeCharterSurvivesTrimming(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Build state with lots of low-priority content to force trimming under
	// the 4000-char compact budget -- the same technique
	// TestColonyPrimeCompactTrimsLowPriorityFirst uses.
	learnings := make([]colony.Learning, 0, 30)
	for i := 0; i < 30; i++ {
		learnings = append(learnings, colony.Learning{
			Claim:  fmt.Sprintf("Learning %d: %s", i, strings.Repeat("text to fill space. ", 20)),
			Status: "confirmed",
		})
	}
	decisions := make([]colony.Decision, 0, 20)
	for i := 0; i < 20; i++ {
		decisions = append(decisions, colony.Decision{
			ID:        fmt.Sprintf("d%d", i),
			Phase:     1,
			Claim:     fmt.Sprintf("Decision %d: %s", i, strings.Repeat("long text to fill budget. ", 15)),
			Rationale: "rationale",
			Timestamp: "2026-01-01T00:00:00Z",
		})
	}

	goal := "charter trim survival test"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase One", Status: "in_progress"},
			},
		},
		Memory: colony.Memory{
			PhaseLearnings: []colony.PhaseLearning{
				{Phase: 1, PhaseName: "Phase One", Learnings: learnings},
			},
			Decisions: decisions,
		},
		Charter: &colony.Charter{
			Governance: "Linting: ESLint. Testing: Jest",
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"colony-prime", "--compact"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("colony-prime --compact returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	result := envelope["result"].(map[string]interface{})
	contextStr := result["context"].(string)
	trimmed := result["trimmed"].([]interface{})

	for _, name := range trimmed {
		if name.(string) == "charter" {
			t.Fatal("charter should NEVER be trimmed -- approved governance is grounding, not filler (D-01)")
		}
	}
	if !strings.Contains(contextStr, "Linting: ESLint") {
		t.Errorf("compact context should still contain charter governance text after trimming, got:\n%s", contextStr)
	}

	// Sanity: prove trimming actually happened in this fixture, otherwise
	// the assertion above is vacuous.
	if len(trimmed) == 0 {
		t.Fatal("test fixture did not force any trimming -- assertion that charter survives it is meaningless")
	}
}

// TestColonyPrimeCharterEmptyProducesNoSection is Test 3: a colony with an
// empty Charter produces no charter section and no empty heading.
func TestColonyPrimeCharterEmptyProducesNoSection(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "empty charter test"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase One", Status: "in_progress"},
			},
		},
		// Charter left nil -- matches a colony where init never populated one.
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"colony-prime"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("colony-prime returned error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	result := envelope["result"].(map[string]interface{})
	promptSection := result["prompt_section"].(string)

	if strings.Contains(promptSection, "Charter") {
		t.Fatalf("prompt_section should not mention Charter when charter is empty:\n%s", promptSection)
	}

	ledger := result["ledger"].(map[string]interface{})
	for _, bucket := range []string{"included", "trimmed", "blocked"} {
		items, ok := ledger[bucket].([]interface{})
		if !ok {
			continue
		}
		for _, raw := range items {
			entry := raw.(map[string]interface{})
			if entry["name"] == "charter" {
				t.Fatalf("charter section should not appear in ledger.%s when charter is empty", bucket)
			}
		}
	}

	// Also cover the fallback-string case: generateCharter's "no formal
	// governance" placeholder must not surface as a hard rule either.
	s2, tmpDir2 := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir2)
	store = s2
	goal2 := "fallback charter test"
	state2 := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal2,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase One", Status: "in_progress"},
			},
		},
		Charter: &colony.Charter{
			Governance: "No formal governance detected -- colony should establish conventions",
		},
	}
	if err := s2.SaveJSON("COLONY_STATE.json", state2); err != nil {
		t.Fatal(err)
	}

	output := buildColonyPrimeOutput(false)
	for _, item := range output.Ledger.Included {
		if item.Name == "charter" {
			t.Fatalf("charter section should not be produced for the 'no formal governance' fallback string")
		}
	}
}

// TestColonyPrimeCharterPassesThroughPromptIntegrity is Test 4: charter
// content passes through colony.AssessPromptSource -- the section appears
// in the same assessed set as `state` (asserted via the ledger/section list,
// not by re-implementing the assessment).
func TestColonyPrimeCharterPassesThroughPromptIntegrity(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "charter integrity test"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase One", Status: "in_progress"},
			},
		},
		Charter: &colony.Charter{
			Governance: "Linting: Prettier",
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	output := buildColonyPrimeOutput(false)

	var charterItem, stateItem *colonyPrimeLedgerItem
	for i := range output.Ledger.Included {
		switch output.Ledger.Included[i].Name {
		case "charter":
			charterItem = &output.Ledger.Included[i]
		case "state":
			stateItem = &output.Ledger.Included[i]
		}
	}
	if charterItem == nil {
		t.Fatalf("charter section missing from ledger.included: %+v", output.Ledger.Included)
	}
	if stateItem == nil {
		t.Fatalf("state section missing from ledger.included: %+v", output.Ledger.Included)
	}

	// A section that never went through AssessPromptSource would have a
	// zero-value TrustClass; a section that did pass through it gets a
	// trust class assigned by the assessment (same mechanism `state` uses).
	if charterItem.TrustClass == "" {
		t.Fatalf("charter section has no trust class -- it did not pass through colony.AssessPromptSource")
	}
	if charterItem.TrustClass != stateItem.TrustClass {
		t.Errorf("charter trust class %q differs from state trust class %q -- both sourced from COLONY_STATE.json should be assessed identically", charterItem.TrustClass, stateItem.TrustClass)
	}

	// Charter must never be delivered by string-concatenating into a brief
	// renderer -- it must arrive only through the colonyPrimeSection route
	// (T-163-03). The absence of any direct Charter reference in the build
	// brief renderer is the acceptance-criteria grep; here we additionally
	// confirm the assembled prompt_section carries the governance text,
	// proving the section route is what actually delivers it.
	if !strings.Contains(output.PromptSection, "Prettier") {
		t.Fatalf("assembled prompt section missing charter governance content:\n%s", output.PromptSection)
	}
}
