package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	// TestColonyPrimeCompactTrimsLowPriorityFirst uses. This used to be
	// PhaseLearnings + Decisions filler; both were removed in 198.2-04
	// (dead capsule slots -- neither field has a writer). instincts.json and
	// hive wisdom are the replacement: two real, prunable, unprotected
	// sections with real writers -- the large instincts section consumes
	// the remaining budget under truncation, leaving hive_wisdom with too
	// little room to fit even a truncated form, so it lands wholly in
	// "trimmed" rather than silently rendering nothing.
	hubDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(hubDir, "hive"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AETHER_HUB_DIR", hubDir)
	// Confidence 0.6 at 120 days decays to ~0.38 under the 180-day
	// half-life -- low enough to be outscored by instincts (below) but
	// still above the 0.3 dormancy floor, so the entry actually reaches the
	// candidate list instead of being dropped before ranking even runs.
	stale := time.Now().Add(-120 * 24 * time.Hour).UTC().Format(time.RFC3339)
	var hiveEntries []string
	for i := 0; i < 5; i++ {
		hiveEntries = append(hiveEntries, fmt.Sprintf(
			`{"id":"w_%d","text":"Wisdom %d: %s","domain":"go","source_repo":"test","confidence":0.6,"created_at":"%s","accessed_at":"%s","access_count":1}`,
			i, i, strings.Repeat("text to fill budget ", 20), stale, stale))
	}
	wisdomData := `{"entries":[` + strings.Join(hiveEntries, ",") + `]}`
	if err := os.WriteFile(filepath.Join(hubDir, "hive", "wisdom.json"), []byte(wisdomData), 0644); err != nil {
		t.Fatal(err)
	}

	// Many SHORT instinct lines rather than few long ones: the ranker fills
	// a truncated section line-by-line and stops at the first line that
	// would overflow the remaining budget, discarding that line's leftover
	// space rather than splitting it further. Short lines keep that
	// discarded slack small, so instincts consumes remaining budget closely
	// enough that hive_wisdom (below) is left with too little room to keep
	// its required two non-empty lines and is dropped outright rather than
	// truncated-and-kept.
	now := time.Now().UTC().Format(time.RFC3339)
	instinctEntries := make([]colony.InstinctEntry, 0, 200)
	for i := 0; i < 200; i++ {
		instinctEntries = append(instinctEntries, colony.InstinctEntry{
			ID:         fmt.Sprintf("i%d", i),
			Trigger:    fmt.Sprintf("t%d", i),
			Action:     fmt.Sprintf("fill budget %d", i),
			Confidence: 0.9,
			TrustScore: 0.9,
			Provenance: colony.InstinctProvenance{CreatedAt: now},
		})
	}
	if err := s.SaveJSON("instincts.json", colony.InstinctsFile{Instincts: instinctEntries}); err != nil {
		t.Fatal(err)
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

// TestColonyPrimeIncludesCharterIntentAndGoals pins the delivery of the five
// charter fields that were captured and then withheld.
//
// /ant-init synthesizes the operator's own words into Intent, Vision, Goals,
// TechStack and KeyRisks, shows them for approval, and stores them in colony
// state. Nothing read them: colony-prime emitted only Governance and
// Constraints, so a colony could record exactly what the operator wanted and
// tell its workers none of it. On the Aether repo itself that meant five
// populated fields silently withheld while Governance — the one field left
// empty — was the only thing forwarded.
//
// The framing matters as much as the delivery, so it is asserted too: approved
// governance is a hard rule, intent and risks are orientation. Presenting them
// identically would make the "hard rules" sentence untrue.
func TestColonyPrimeIncludesCharterIntentAndGoals(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "charter context test"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{{ID: 1, Name: "Phase One", Status: "in_progress"}},
		},
		Charter: &colony.Charter{
			// Governance deliberately empty, mirroring the real colony that
			// exposed this: the only forwarded field was the blank one.
			Intent:      "Dogfood Aether on itself with real iterative planning",
			Vision:      "Restore the useful Classic planning feel",
			Goals:       "Identify and fix the highest-value defects first",
			TechStack:   "Go CLI runtime, TypeScript host bridge",
			KeyRisks:    "Local state is stale; wrapper docs drift from runtime",
			Constraints: "Go owns .aether/data state and finalizers",
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

	for _, want := range []string{
		"Dogfood Aether on itself",
		"Restore the useful Classic planning feel",
		"Identify and fix the highest-value defects first",
		"Go CLI runtime, TypeScript host bridge",
		"Local state is stale",
	} {
		if !strings.Contains(promptSection, want) {
			t.Errorf("charter field never reached the worker prompt: %q\n---\n%s", want, promptSection)
		}
	}

	// Constraints stay in the hard-rules block; orientation is separate.
	if !strings.Contains(promptSection, "hard rules every worker must follow") {
		t.Errorf("approved constraints lost their hard-rule framing:\n%s", promptSection)
	}
	if !strings.Contains(promptSection, "orientation for judgement calls, not as additional hard rules") {
		t.Errorf("context fields were not distinguished from approved rules:\n%s", promptSection)
	}
}
