package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func seedOraclePlan(t *testing.T, root string) {
	t.Helper()
	oracleDir := filepath.Join(root, ".aether", "oracle")
	if err := os.MkdirAll(oracleDir, 0755); err != nil {
		t.Fatal(err)
	}
	plan := oraclePlanFile{
		Questions: []oracleQuestion{
			{
				ID: "q1", Text: "How should the exporter retry?", Status: "answered", Confidence: 90,
				KeyFindings: []oracleFinding{
					{Text: "The retry helper in cmd/retry.go already handles exponential backoff; reuse it instead of writing a new loop."},
					{Text: "Everything went well and the phase completed successfully overall."}, // inadmissible: narration, no anchor
				},
			},
			{
				ID: "q2", Text: "Low-confidence question", Status: "partial", Confidence: 40,
				KeyFindings: []oracleFinding{
					{Text: "cmd/never.go should never be promoted from a 40% question."},
				},
			},
		},
	}
	data, err := json.MarshalIndent(plan, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oracleDir, "plan.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
}

// `aether oracle promote` was removed in the Go migration; findings have been
// hand-copied ever since. This locks the restored path: high-confidence
// findings become learnings + instincts THROUGH the admissibility gate.
func TestOraclePromoteWritesAdmissibleFindings(t *testing.T) {
	saveGlobals(t)
	s, dataDir := newTestStore(t)
	_ = dataDir
	store = s
	root := filepath.Dir(filepath.Dir(s.BasePath()))
	dataDir = s.BasePath()
	seedOraclePlan(t, root)

	result, err := runOraclePromote(root, 80, false, "")
	if err != nil {
		t.Fatalf("promote: %v", err)
	}
	if got := result["learnings"].(int); got != 1 {
		t.Errorf("learnings = %d, want 1 (admissible finding only)", got)
	}
	if got := result["instincts"].(int); got != 1 {
		t.Errorf("instincts = %d, want 1", got)
	}
	if got := result["questions_considered"].(int); got != 1 {
		t.Errorf("questions_considered = %d, want 1 (40%% question excluded)", got)
	}

	var instFile colony.InstinctsFile
	if err := store.LoadJSON("instincts.json", &instFile); err != nil {
		t.Fatalf("load instincts: %v", err)
	}
	if len(instFile.Instincts) != 1 {
		t.Fatalf("instincts on disk = %d, want 1", len(instFile.Instincts))
	}

	entriesData, err := os.ReadFile(filepath.Join(dataDir, "entries.json"))
	if err != nil {
		t.Fatalf("read learn entries: %v", err)
	}
	if !json.Valid(entriesData) {
		t.Fatal("entries.json is not valid JSON")
	}

	outcomes := result["outcomes"].([]oraclePromoteOutcome)
	skipped := 0
	for _, o := range outcomes {
		if o.Action == "skipped" {
			skipped++
			if o.SkipReason == "" {
				t.Error("skipped outcome must carry the admissibility reason")
			}
		}
	}
	if skipped != 1 {
		t.Errorf("skipped = %d, want 1 (the narration finding)", skipped)
	}
}

// --dry-run reports without mutating — the Phase 2 purity contract applies to
// new commands too.
func TestOraclePromoteDryRunDoesNotWrite(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s
	root := filepath.Dir(filepath.Dir(s.BasePath()))
	dataDir := s.BasePath()
	seedOraclePlan(t, root)

	result, err := runOraclePromote(root, 80, true, "")
	if err != nil {
		t.Fatalf("promote dry-run: %v", err)
	}
	if result["dry_run"] != true {
		t.Fatal("dry_run flag not honored in result")
	}
	if _, err := os.Stat(filepath.Join(dataDir, "instincts.json")); !os.IsNotExist(err) {
		var instFile colony.InstinctsFile
		if loadErr := store.LoadJSON("instincts.json", &instFile); loadErr == nil && len(instFile.Instincts) > 0 {
			t.Fatalf("dry-run wrote %d instincts", len(instFile.Instincts))
		}
	}
	if _, err := os.Stat(filepath.Join(dataDir, "entries.json")); !os.IsNotExist(err) {
		t.Fatal("dry-run wrote learn entries")
	}
}

// Without an oracle workspace the command fails with guidance, not a panic.
func TestOraclePromoteWithoutResearchFails(t *testing.T) {
	if _, err := runOraclePromote(t.TempDir(), 80, false, ""); err == nil {
		t.Fatal("expected error when no oracle plan exists")
	}
}

// Downstream repos have no colony/policies directory; every Oracle phase used
// to collapse to one generic directive there. The four real phase directives
// must survive with no file present at all.
func TestOraclePhaseDirectivesSurviveWithoutPolicyFile(t *testing.T) {
	withWorkingDir(t, t.TempDir())
	t.Setenv("AETHER_HUB_DIR", t.TempDir())
	t.Setenv("AETHER_ROOT", t.TempDir())

	seen := map[string]bool{}
	for _, phase := range []string{"survey", "verify", "investigate", "synthesize"} {
		directive := buildOraclePhaseDirective(phase)
		if directive == "" {
			t.Fatalf("phase %s has no directive", phase)
		}
		if seen[directive] {
			t.Fatalf("phase %s shares a directive with another phase — phases collapsed to a generic fallback", phase)
		}
		seen[directive] = true
	}
}
