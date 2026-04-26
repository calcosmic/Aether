package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestColonyPrimeIncludesGlobalQueen(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	hubDir := filepath.Join(tmpDir, "hub")
	os.MkdirAll(hubDir, 0755)
	origHub := os.Getenv("AETHER_HUB_DIR")
	os.Setenv("AETHER_HUB_DIR", hubDir)
	t.Cleanup(func() { os.Setenv("AETHER_HUB_DIR", origHub) })

	// Write global QUEEN.md with wisdom
	globalQueenContent := `# QUEEN.md

## Wisdom

- Always prefer explicit error handling over panics
- Use table-driven tests for exhaustive coverage

## Patterns

- Factory pattern for agent creation
`
	if err := os.WriteFile(filepath.Join(hubDir, "QUEEN.md"), []byte(globalQueenContent), 0644); err != nil {
		t.Fatalf("write global QUEEN.md: %v", err)
	}

	goal := "global queen test"
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
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"colony-prime"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse the JSON envelope -- prompt_section is nested inside "result"
	envelope := parseEnvelopeCmd(t, buf.String())
	result := envelope["result"].(map[string]interface{})
	promptSection, ok := result["prompt_section"].(string)
	if !ok {
		t.Fatalf("result.prompt_section not a string: %T", result["prompt_section"])
	}

	if !strings.Contains(promptSection, "GLOBAL QUEEN WISDOM") {
		t.Fatalf("prompt_section missing GLOBAL QUEEN WISDOM section:\n%s", promptSection)
	}
	if !strings.Contains(promptSection, "Always prefer explicit error handling") {
		t.Fatalf("prompt_section missing global queen wisdom content:\n%s", promptSection)
	}

	// Verify the section appears in the ledger as preserved (protected)
	ledger := result["ledger"].(map[string]interface{})
	included, ok := ledger["included"].([]interface{})
	if !ok {
		t.Fatalf("ledger.included not an array: %T", ledger["included"])
	}

	found := false
	for _, raw := range included {
		entry := raw.(map[string]interface{})
		if entry["name"] == "global_queen_md" {
			found = true
			if preserved, _ := entry["preserved"].(bool); !preserved {
				t.Fatalf("global_queen_md should be preserved (protected), but was not")
			}
			break
		}
	}
	if !found {
		t.Fatalf("global_queen_md not found in ledger included items")
	}
}

func TestColonyPrimeGlobalQueenSurvivesWithoutFile(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s

	hubDir := filepath.Join(tmpDir, "hub")
	os.MkdirAll(hubDir, 0755)
	origHub := os.Getenv("AETHER_HUB_DIR")
	os.Setenv("AETHER_HUB_DIR", hubDir)
	t.Cleanup(func() { os.Setenv("AETHER_HUB_DIR", origHub) })

	// Intentionally do NOT write QUEEN.md to hub

	goal := "no global queen test"
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
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"colony-prime"})
	defer rootCmd.SetArgs([]string{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	envelope := parseEnvelopeCmd(t, buf.String())
	result := envelope["result"].(map[string]interface{})
	promptSection, ok := result["prompt_section"].(string)
	if !ok {
		t.Fatalf("result.prompt_section not a string: %T", result["prompt_section"])
	}

	// Should NOT contain global queen wisdom since file doesn't exist
	if strings.Contains(promptSection, "GLOBAL QUEEN WISDOM") {
		t.Fatalf("prompt_section should not contain GLOBAL QUEEN WISDOM when file is absent:\n%s", promptSection)
	}

	// Verify ledger doesn't list it
	ledger := result["ledger"].(map[string]interface{})
	included, ok := ledger["included"].([]interface{})
	if !ok {
		t.Fatalf("ledger.included not an array: %T", ledger["included"])
	}
	for _, raw := range included {
		entry := raw.(map[string]interface{})
		if entry["name"] == "global_queen_md" {
			t.Fatal("global_queen_md should not appear in ledger when file is absent")
		}
	}
}
