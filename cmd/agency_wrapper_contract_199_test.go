package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

func TestAgencyReceipt199CommandWiring(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceJSONOutputModeForTest(t)
	root := t.TempDir()
	dataDir := filepath.Join(root, ".aether", "data")
	var err error
	store, err = storage.NewStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	jobID := "job-independent"
	state := colony.ColonyState{
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan:         colony.Plan{Phases: []colony.Phase{{ID: 1, Status: colony.PhaseInProgress, Tasks: []colony.Task{{ID: &jobID, Goal: "Independent work", Status: colony.TaskInProgress}}}}},
	}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}
	beforeState, err := os.ReadFile(filepath.Join(dataDir, "COLONY_STATE.json"))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("AETHER_ROOT", root)

	run := func(args ...string) map[string]interface{} {
		t.Helper()
		var output bytes.Buffer
		stdout = &output
		rootCmd.SetArgs(args)
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		var envelope map[string]interface{}
		if err := json.Unmarshal(output.Bytes(), &envelope); err != nil {
			t.Fatalf("%v returned invalid JSON %q: %v", args, output.String(), err)
		}
		result, ok := envelope["result"].(map[string]interface{})
		if !ok {
			t.Fatalf("%v result = %T in %s", args, envelope["result"], output.String())
		}
		return result
	}

	assertReceipt := func(result map[string]interface{}, wantType string) string {
		t.Helper()
		signal, ok := result["signal"].(map[string]interface{})
		if !ok || signal["type"] != wantType {
			t.Fatalf("signal = %#v, want type %s", result["signal"], wantType)
		}
		receipt, ok := result["receipt"].(map[string]interface{})
		if !ok {
			t.Fatalf("receipt = %T, want object: %#v", result["receipt"], result)
		}
		if receipt["signal_id"] != signal["id"] {
			t.Fatalf("receipt signal_id = %v, durable id = %v", receipt["signal_id"], signal["id"])
		}
		if got := result["delivery"]; got != "next safe boundary" {
			t.Fatalf("delivery = %v, want next safe boundary", got)
		}
		if got := result["acknowledgement"]; got != AgencyAcknowledgementPending {
			t.Fatalf("acknowledgement = %v", got)
		}
		if got := result["measured_effect"]; got != AgencyMeasuredEffectPending {
			t.Fatalf("measured effect = %v", got)
		}
		if got := result["work_effect"]; got != string(AgencyWorkContinuedIndependent) {
			t.Fatalf("work effect = %v, want continued independent work", got)
		}
		return signal["id"].(string)
	}

	first := run("focus", "prefer score < 10 changes")
	firstID := assertReceipt(first, "FOCUS")
	content := first["signal"].(map[string]interface{})["content"].(map[string]interface{})
	if content["text"] != "prefer score &lt; 10 changes" {
		t.Fatalf("signal did not retain sanitization: %#v", content)
	}
	second := run("focus", "prefer score < 10 changes")
	if secondID := assertReceipt(second, "FOCUS"); secondID != firstID {
		t.Fatalf("reinforcement changed durable identity: %s -> %s", firstID, secondID)
	}
	if reinforced, _ := second["reinforced"].(bool); !reinforced {
		t.Fatalf("replayed signal was not reported as reinforced: %#v", second)
	}
	assertReceipt(run("feedback", "keep the existing boundary"), "FEEDBACK")
	assertReceipt(run("redirect", "avoid changing unrelated work"), "REDIRECT")

	var pheromones colony.PheromoneFile
	if err := store.LoadJSON("pheromones.json", &pheromones); err != nil {
		t.Fatal(err)
	}
	if len(pheromones.Signals) != 3 {
		t.Fatalf("stored signal count = %d, want 3 after one reinforcement", len(pheromones.Signals))
	}
	afterState, err := os.ReadFile(filepath.Join(dataDir, "COLONY_STATE.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(beforeState, afterState) {
		t.Fatal("FOCUS/FEEDBACK/non-conflicting REDIRECT changed independent job state")
	}
}

func TestAgencyWrapperAuthority199(t *testing.T) {
	repoRoot := filepath.Clean("..")
	commands := []string{"focus", "feedback", "redirect"}
	for _, command := range commands {
		for _, path := range []string{
			filepath.Join(repoRoot, ".aether", "commands", command+".yaml"),
			filepath.Join(repoRoot, ".claude", "commands", "ant", command+".md"),
			filepath.Join(repoRoot, ".opencode", "commands", "ant", command+".md"),
		} {
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			text := strings.ToLower(string(body))
			if !strings.Contains(text, "aether "+command) {
				t.Errorf("%s does not invoke the Go runtime", path)
			}
			for _, forbidden := range []string{"delivery: live", "measured effect:", "acknowledged by"} {
				if strings.Contains(text, forbidden) {
					t.Errorf("%s makes host-owned agency claim %q", path, forbidden)
				}
			}
		}
	}

	for _, path := range []string{
		filepath.Join(repoRoot, ".aether", "commands", "swarm.yaml"),
		filepath.Join(repoRoot, ".claude", "commands", "ant", "swarm.md"),
		filepath.Join(repoRoot, ".opencode", "commands", "ant", "swarm.md"),
	} {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := strings.ToLower(string(body))
		if !strings.Contains(text, "aether swarm --plan-only") || !strings.Contains(text, "aether swarm-finalize") {
			t.Errorf("%s bypasses the runtime manifest/finalizer authority", path)
		}
		if strings.Contains(text, "delivery: live") || strings.Contains(text, "measured effect:") {
			t.Errorf("%s invents causal steering evidence", path)
		}
	}
}
