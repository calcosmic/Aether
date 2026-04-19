package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestBehaviorObserveAppendsJSONL(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	goal := "Improve operator experience"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
	})

	rootCmd.SetArgs([]string{
		"behavior-observe",
		"--dimension", "communication_style",
		"--signal", "Prefer concise updates",
		"--strength", "0.8",
		"--evidence", "User cut off long explanations twice",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("behavior-observe returned error: %v", err)
	}

	entries, err := store.ReadJSONL(behaviorObservationsFile)
	if err != nil {
		t.Fatalf("read behavior observations: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 observation, got %d", len(entries))
	}

	var observation colony.BehaviorObservation
	if err := json.Unmarshal(entries[0], &observation); err != nil {
		t.Fatalf("unmarshal observation: %v", err)
	}
	if observation.Dimension != "communication_style" {
		t.Fatalf("dimension = %q", observation.Dimension)
	}
	if observation.ColonyGoal != goal {
		t.Fatalf("colony_goal = %q, want %q", observation.ColonyGoal, goal)
	}
	if observation.Signal != "Prefer concise updates" {
		t.Fatalf("signal = %q", observation.Signal)
	}
}

func TestProfileReadReturnsProfile(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	setupBuildFlowTest(t)
	hubDir := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", hubDir)

	hub := hubStore()
	if hub == nil {
		t.Fatal("expected hub store")
	}
	expected := colony.UserProfile{
		Version:     "1.0",
		GeneratedAt: "2026-04-19T10:00:00Z",
		ColonyCount: 2,
		Dimensions: []colony.BehavioralDimension{
			{Name: "communication_style", Score: 0.9, Evidence: []string{"status updates"}, UpdatedAt: "2026-04-19T10:00:00Z", SampleCount: 3},
		},
		Directives: []string{"[profiled] Prefer concise updates"},
	}
	if err := hub.SaveJSON(profileFileName, expected); err != nil {
		t.Fatalf("save profile: %v", err)
	}

	rootCmd.SetArgs([]string{"profile-read"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("profile-read returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if stringValue(result["path"]) != filepath.Join(hubDir, profileFileName) {
		t.Fatalf("unexpected profile path %q", result["path"])
	}
	profileMap := result["profile"].(map[string]interface{})
	if got := int(profileMap["colony_count"].(float64)); got != 2 {
		t.Fatalf("colony_count = %d, want 2", got)
	}
	directives := profileMap["directives"].([]interface{})
	if len(directives) != 1 || directives[0].(string) != "[profiled] Prefer concise updates" {
		t.Fatalf("unexpected directives: %#v", directives)
	}
}

func TestProfileUpdateWritesDirectivesToQueenMd(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	hubDir := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", hubDir)

	goal := "Improve operator experience"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
	})

	observations := []colony.BehaviorObservation{
		{
			Timestamp:  "2026-04-19T10:00:00Z",
			ColonyGoal: goal,
			Command:    "build",
			Dimension:  "communication_style",
			Signal:     "Prefer concise updates",
			Strength:   0.9,
			Evidence:   "User repeatedly prefers short summaries",
		},
		{
			Timestamp:  "2026-04-19T10:05:00Z",
			ColonyGoal: goal,
			Command:    "continue",
			Dimension:  "ux_philosophy",
			Signal:     "Bias early work toward function over polish",
			Strength:   0.8,
			Evidence:   "User deprioritized polish during iteration",
		},
		{
			Timestamp:  "2026-04-19T10:10:00Z",
			ColonyGoal: goal,
			Command:    "debug",
			Dimension:  "debugging_approach",
			Signal:     "Prefer fix-first debugging unless the failure pattern is unclear",
			Strength:   0.85,
			Evidence:   "User asked to fix the issue directly after a clear review finding",
		},
	}
	for _, observation := range observations {
		if err := store.AppendJSONL(behaviorObservationsFile, observation); err != nil {
			t.Fatalf("seed observation: %v", err)
		}
	}

	rootCmd.SetArgs([]string{"profile-update"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("profile-update returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if updated, _ := result["updated"].(bool); !updated {
		t.Fatalf("expected updated:true, got %#v", result)
	}
	if got := int(result["promoted_count"].(float64)); got != 3 {
		t.Fatalf("promoted_count = %d, want 3", got)
	}

	hub := hubStore()
	if hub == nil {
		t.Fatal("expected hub store")
	}
	var profile colony.UserProfile
	if err := hub.LoadJSON(profileFileName, &profile); err != nil {
		t.Fatalf("load profile.json: %v", err)
	}
	if len(profile.Directives) < 3 {
		t.Fatalf("expected at least 3 directives, got %d", len(profile.Directives))
	}
	for _, directive := range profile.Directives[:3] {
		if !strings.HasPrefix(strings.ToLower(directive), "[profiled]") {
			t.Fatalf("directive missing [profiled] prefix: %q", directive)
		}
	}

	queenData, err := os.ReadFile(filepath.Join(hubDir, "QUEEN.md"))
	if err != nil {
		t.Fatalf("read QUEEN.md: %v", err)
	}
	queenText := string(queenData)
	for _, snippet := range []string{
		"[profiled] Prefer concise updates",
		"[profiled] Bias early work toward function over polish",
		"[profiled] Prefer fix-first debugging unless the failure pattern is unclear",
	} {
		if !strings.Contains(queenText, snippet) {
			t.Fatalf("QUEEN.md missing %q:\n%s", snippet, queenText)
		}
	}
}
