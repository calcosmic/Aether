package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/storage"
)

type legacyRecovery199Fixture struct {
	root    string
	dataDir string
	hub     string
}

type legacyRecovery199Fingerprint struct {
	trees     map[string][]string
	gitHead   string
	gitStatus string
}

func newLegacyRecovery199Fixture(t *testing.T) legacyRecovery199Fixture {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)

	root := t.TempDir()
	home := t.TempDir()
	// Keep lifecycle data outside the repository so the contract proves every
	// reader follows the configured Store root instead of assuming the default
	// .aether/data location.
	dataDir := filepath.Join(home, "colony-data")
	hub := filepath.Join(home, ".aether-hub")
	for _, dir := range []string{dataDir, hub} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	t.Setenv("HOME", home)
	t.Setenv("AETHER_ROOT", root)
	t.Setenv("COLONY_DATA_DIR", dataDir)
	t.Setenv("AETHER_HUB_DIR", hub)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	t.Setenv("AETHER_PLATFORM", "codex")
	t.Setenv("NO_COLOR", "1")

	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	store = s

	// A legacy numeric current_phase is intentional: a read-only migration
	// route may normalize it in memory, but must not repair the source bytes.
	state := `{
  "version": "3.0",
  "goal": "Resume is the only recovery door",
  "state": "EXECUTING",
  "current_phase": "1",
  "plan": {
    "phases": [
      {
        "id": 1,
        "name": "Recovery boundary",
        "status": "in_progress",
        "tasks": [{"id": "1.1", "goal": "retain evidence", "status": "in_progress"}]
      }
    ]
  }
}
`
	write199File(t, filepath.Join(dataDir, "COLONY_STATE.json"), state)
	stale := time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)
	write199File(t, filepath.Join(dataDir, "spawn-runs.json"), fmt.Sprintf(`{
  "current_run_id": "run-stale",
  "runs": [{"id": "run-stale", "started_at": %q, "status": "running"}]
}
`, stale))
	write199File(t, filepath.Join(hub, "registry.json"), `{"colonies":[{"path":"fixture"}]}`+"\n")
	write199File(t, filepath.Join(root, "README.md"), "legacy recovery fixture\n")

	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "fixture")

	var output bytes.Buffer
	stdout = &output
	stderr = &output
	rootCmd.SetOut(&output)
	rootCmd.SetErr(&output)
	t.Cleanup(func() { rootCmd.SetErr(os.Stderr) })

	return legacyRecovery199Fixture{root: root, dataDir: dataDir, hub: hub}
}

func (fixture legacyRecovery199Fixture) fingerprint(t *testing.T) legacyRecovery199Fingerprint {
	t.Helper()
	head, err := gitOutputAt(fixture.root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("read git HEAD: %v", err)
	}
	status, err := gitOutputAt(fixture.root, "status", "--porcelain=v1", "--untracked-files=all")
	if err != nil {
		t.Fatalf("read git status: %v", err)
	}
	return legacyRecovery199Fingerprint{
		trees:     fingerprintLegacyRecovery199Trees(t, fixture.root, fixture.dataDir, fixture.hub),
		gitHead:   head,
		gitStatus: status,
	}
}

func fingerprintLegacyRecovery199Trees(t *testing.T, roots ...string) map[string][]string {
	t.Helper()
	result := make(map[string][]string, len(roots))
	for _, root := range roots {
		var entries []string
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			if rel == ".git" && entry.IsDir() {
				return filepath.SkipDir
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			line := fmt.Sprintf("%s|%s|%d|%d", filepath.ToSlash(rel), info.Mode(), info.Size(), info.ModTime().UnixNano())
			if info.Mode().IsRegular() {
				content, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				digest := sha256.Sum256(content)
				line += "|" + hex.EncodeToString(digest[:])
			}
			entries = append(entries, line)
			return nil
		})
		if err != nil {
			t.Fatalf("fingerprint %s: %v", root, err)
		}
		sort.Strings(entries)
		result[root] = entries
	}
	return result
}

func runLegacyRecovery199Command(t *testing.T, args ...string) (string, error) {
	t.Helper()
	resetFlags(rootCmd)
	var output bytes.Buffer
	stdout = &output
	stderr = &output
	rootCmd.SetOut(&output)
	rootCmd.SetErr(&output)
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	return output.String(), err
}

func assertLegacyRecovery199Unchanged(t *testing.T, before, after legacyRecovery199Fingerprint) {
	t.Helper()
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("legacy command mutated durable recovery evidence\nbefore: %#v\nafter:  %#v", before, after)
	}
}

func TestLegacyRecoveryCommands199Hidden(t *testing.T) {
	for name, command := range map[string]bool{
		"recover": recoverCmd.Hidden,
		"abandon": abandonCmd.Hidden,
	} {
		if !command {
			t.Errorf("%s command is public; want hidden parser compatibility", name)
		}
	}
	for _, input := range []string{"recove", "abando"} {
		for _, suggestion := range rootCmd.SuggestionsFor(input) {
			if suggestion == "recover" || suggestion == "abandon" {
				t.Errorf("%q suggested hidden lifecycle command %q", input, suggestion)
			}
		}
	}
}

func TestLegacyRecoveryCommands199RecoverRoutesToResume(t *testing.T) {
	for _, args := range [][]string{
		{"recover"},
		{"recover", "--apply", "--force"},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			fixture := newLegacyRecovery199Fixture(t)
			before := fixture.fingerprint(t)
			output, err := runLegacyRecovery199Command(t, args...)
			if err != nil {
				t.Fatalf("%v returned error: %v\n%s", args, err, output)
			}
			after := fixture.fingerprint(t)
			assertLegacyRecovery199Unchanged(t, before, after)
			if !strings.Contains(output, "aether resume") {
				t.Fatalf("legacy recover did not route to canonical resume:\n%s", output)
			}
			if strings.Contains(output, "recover --apply") {
				t.Fatalf("legacy recover still advertised an independent mutator:\n%s", output)
			}
		})
	}
}

func TestLegacyRecoveryCommands199AbandonZeroWrite(t *testing.T) {
	fixture := newLegacyRecovery199Fixture(t)
	before := fixture.fingerprint(t)
	output, err := runLegacyRecovery199Command(t, "abandon", "--confirm")
	if err != nil {
		t.Fatalf("legacy abandon returned error: %v\n%s", err, output)
	}
	after := fixture.fingerprint(t)
	assertLegacyRecovery199Unchanged(t, before, after)
	for _, want := range []string{"aether seal --force --reason", "direct owner", "did not invoke"} {
		if !strings.Contains(strings.ToLower(output), strings.ToLower(want)) {
			t.Errorf("legacy abandon guidance missing %q:\n%s", want, output)
		}
	}
}

func TestLegacyRecoveryCommands199MaintenanceDiagnosis(t *testing.T) {
	fixture := newLegacyRecovery199Fixture(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	before := fixture.fingerprint(t)
	output, err := runLegacyRecovery199Command(t, "maintenance", "recovery-inspect")
	if err != nil {
		t.Fatalf("maintenance recovery inspection returned error: %v\n%s", err, output)
	}
	after := fixture.fingerprint(t)
	assertLegacyRecovery199Unchanged(t, before, after)

	var envelope struct {
		OK     bool           `json:"ok"`
		Result map[string]any `json:"result"`
	}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("decode maintenance recovery output %q: %v", output, err)
	}
	if !envelope.OK {
		t.Fatalf("maintenance recovery result was not successful: %s", output)
	}
	if got := stringField(t, envelope.Result, "operation_id"); got != "recovery.inspect" {
		t.Fatalf("operation_id = %q, want recovery.inspect", got)
	}
	if got := stringField(t, envelope.Result, "state_effect"); got != "none" {
		t.Fatalf("state_effect = %q, want none", got)
	}
	if got := stringField(t, envelope.Result, "next_action"); got != "aether resume" {
		t.Fatalf("next_action = %q, want aether resume", got)
	}
	provenance, ok := envelope.Result["provenance"].([]any)
	if !ok || len(provenance) == 0 {
		t.Fatalf("provenance = %#v, want named lifecycle sources", envelope.Result["provenance"])
	}
	evidence, ok := envelope.Result["evidence"].([]any)
	if !ok || len(evidence) == 0 {
		t.Fatalf("evidence = %#v, want inspected paths", envelope.Result["evidence"])
	}
	issues, ok := envelope.Result["issues"].([]any)
	if !ok {
		t.Fatalf("issues = %T, want stable array", envelope.Result["issues"])
	}
	foundStale := false
	for _, raw := range issues {
		issue, _ := raw.(map[string]any)
		if issue["category"] == "stale_spawned" {
			foundStale = true
		}
	}
	if !foundStale {
		t.Fatalf("maintenance diagnosis lost the existing stale-worker scanner: %#v", issues)
	}
}
