package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestMedicCommandNoColony(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	setupBuildFlowTest(t)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	// No colony state file exists -- should handle gracefully
	rootCmd.SetArgs([]string{"medic"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("medic returned error on no-colony repo: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	if !strings.Contains(output, "No colony initialized") {
		t.Errorf("medic no-colony output missing 'No colony initialized', got:\n%s", output)
	}
}

func TestMedicCommandWithColony(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	goal := "Test colony health"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateEXECUTING,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: colony.PhaseInProgress},
			},
		},
	})

	rootCmd.SetArgs([]string{"medic"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("medic returned error on active colony: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	if !strings.Contains(output, "C O L O N Y   H E A L T H") {
		t.Errorf("medic output missing banner, got:\n%s", output)
	}
}

func TestMedicCommandJSONOutput(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")

	goal := "Test JSON output"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: colony.PhaseReady},
			},
		},
	})

	rootCmd.SetArgs([]string{"medic", "--json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("medic --json returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("medic --json output is not valid JSON: %v\n%s", err, output)
	}
	if _, ok := parsed["issues"]; !ok {
		t.Errorf("medic --json output missing 'issues' key: %s", output)
	}
}

func TestMedicCommandFixFlag(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	goal := "Test fix flag"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	rootCmd.SetArgs([]string{"medic", "--fix"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("medic --fix returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	// Fix mode should still show the banner
	if !strings.Contains(output, "C O L O N Y   H E A L T H") {
		t.Errorf("medic --fix output missing banner, got:\n%s", output)
	}
}

func TestMedicCommandHelp(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	// Verify the command flags are registered by inspecting the cobra command directly
	flags := medicCmd.Flags()
	for _, name := range []string{"fix", "force", "json", "deep"} {
		f := flags.Lookup(name)
		if f == nil {
			t.Errorf("medic command missing flag --%s", name)
		}
	}
}

func TestMedicReportRendering(t *testing.T) {
	issues := []HealthIssue{
		{Severity: "critical", Category: "state", Message: "Colony state corrupted", File: "COLONY_STATE.json", Fixable: true},
		{Severity: "warning", Category: "pheromone", Message: "Stale pheromone signal", File: "pheromones.json", Fixable: false},
		{Severity: "info", Category: "session", Message: "Session file present", File: "session.json", Fixable: false},
	}

	opts := MedicOptions{Fix: false, Force: false, JSON: false, Deep: false}
	output := renderMedicReport(issues, opts, &colony.ColonyState{})

	for _, want := range []string{"C O L O N Y   H E A L T H", "Summary", "Critical", "Warnings", "Info", "Colony state corrupted"} {
		if !strings.Contains(output, want) {
			t.Errorf("renderMedicReport missing %q\n%s", want, output)
		}
	}
}

func TestMedicReportJSONOutput(t *testing.T) {
	issues := []HealthIssue{
		{Severity: "critical", Category: "state", Message: "Test issue", File: "test.json", Fixable: true},
	}

	state := &colony.ColonyState{}
	output := renderMedicJSON(issues, state)

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("renderMedicJSON output is not valid JSON: %v\n%s", err, output)
	}
	if parsed["timestamp"] == nil {
		t.Error("renderMedicJSON missing timestamp")
	}
	issuesList, ok := parsed["issues"].([]interface{})
	if !ok {
		t.Fatal("renderMedicJSON missing issues array")
	}
	if len(issuesList) != 1 {
		t.Errorf("renderMedicJSON expected 1 issue, got %d", len(issuesList))
	}
}

func TestMedicSeverityColor(t *testing.T) {
	tests := []struct {
		severity string
		want     string
	}{
		{"critical", "\033[31m"},
		{"warning", "\033[33m"},
		{"info", "\033[34m"},
		{"unknown", ""},
	}
	for _, tt := range tests {
		got := severityColor(tt.severity)
		if got != tt.want {
			t.Errorf("severityColor(%q) = %q, want %q", tt.severity, got, tt.want)
		}
	}
}

func TestMedicExitCodes(t *testing.T) {
	tests := []struct {
		name    string
		issues  []HealthIssue
		want    int
	}{
		{"healthy", nil, 0},
		{"warnings", []HealthIssue{{Severity: "warning", Message: "test"}}, 1},
		{"critical", []HealthIssue{{Severity: "critical", Message: "test"}}, 2},
		{"mixed", []HealthIssue{{Severity: "info", Message: "ok"}, {Severity: "critical", Message: "bad"}}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := medicExitCode(tt.issues)
			if got != tt.want {
				t.Errorf("medicExitCode() = %d, want %d", got, tt.want)
			}
		})
	}
}
