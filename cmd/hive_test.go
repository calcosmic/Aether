package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/memory"
)

func TestHiveInitCreatesWisdomFile(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	// Redirect hub to temp dir
	hubDir := t.TempDir()
	origHub := os.Getenv("AETHER_HUB_DIR")
	os.Setenv("AETHER_HUB_DIR", hubDir)
	defer os.Setenv("AETHER_HUB_DIR", origHub)

	rootCmd.SetArgs([]string{"hive-init"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("hive-init returned error: %v", err)
	}

	out := parseEnvelope(t, buf.String())
	if out["ok"] != true {
		t.Fatalf("expected ok=true, got: %v", out["ok"])
	}
	result := out["result"].(map[string]interface{})
	if result["initialized"] != true {
		t.Fatalf("expected initialized=true, got: %v", result["initialized"])
	}

	wisdomPath := filepath.Join(hubDir, "hive", "wisdom.json")
	if _, err := os.Stat(wisdomPath); os.IsNotExist(err) {
		t.Fatalf("wisdom.json not created at %s", wisdomPath)
	}
}

func TestHiveStorePersistsEntry(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	hubDir := t.TempDir()
	origHub := os.Getenv("AETHER_HUB_DIR")
	os.Setenv("AETHER_HUB_DIR", hubDir)
	defer os.Setenv("AETHER_HUB_DIR", origHub)

	// Initialize first
	rootCmd.SetArgs([]string{"hive-init"})
	_ = rootCmd.Execute()
	resetRootCmd(t)
	stdout = &buf
	buf.Reset()

	rootCmd.SetArgs([]string{"hive-store", "Prefer table-driven cases in pkg/colony/state.go when states multiply", "testing", "aether"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("hive-store returned error: %v", err)
	}

	out := parseEnvelope(t, buf.String())
	if out["ok"] != true {
		t.Fatalf("expected ok=true, got: %v", out["ok"])
	}
	result := out["result"].(map[string]interface{})
	if result["stored"] != true {
		t.Fatalf("expected stored=true, got: %v", result["stored"])
	}

	// Verify filesystem
	wisdomPath := filepath.Join(hubDir, "hive", "wisdom.json")
	raw, err := os.ReadFile(wisdomPath)
	if err != nil {
		t.Fatalf("read wisdom.json: %v", err)
	}
	var wf hiveWisdomData
	if err := json.Unmarshal(raw, &wf); err != nil {
		t.Fatalf("parse wisdom.json: %v", err)
	}
	if len(wf.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(wf.Entries))
	}
	if wf.Entries[0].Text != "Prefer table-driven cases in pkg/colony/state.go when states multiply" {
		t.Fatalf("expected admissible fixture text, got %q", wf.Entries[0].Text)
	}
}

func TestHiveStoreReinforcesDuplicate(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	hubDir := t.TempDir()
	origHub := os.Getenv("AETHER_HUB_DIR")
	os.Setenv("AETHER_HUB_DIR", hubDir)
	defer os.Setenv("AETHER_HUB_DIR", origHub)

	rootCmd.SetArgs([]string{"hive-init"})
	_ = rootCmd.Execute()

	// Store first entry
	resetRootCmd(t)
	stdout = &buf
	buf.Reset()
	rootCmd.SetArgs([]string{"hive-store", "Run go vet ./... before commit; it catches shadowed err returns", "testing", "aether"})
	_ = rootCmd.Execute()

	// Store same text again
	resetRootCmd(t)
	stdout = &buf
	buf.Reset()
	rootCmd.SetArgs([]string{"hive-store", "Run go vet ./... before commit; it catches shadowed err returns", "testing", "aether"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("hive-store returned error: %v", err)
	}

	out := parseEnvelope(t, buf.String())
	result := out["result"].(map[string]interface{})
	if result["reinforced"] != true {
		t.Fatalf("expected reinforced=true for duplicate, got: %v", result["reinforced"])
	}

	// Verify only 1 entry with higher access count
	wisdomPath := filepath.Join(hubDir, "hive", "wisdom.json")
	raw, _ := os.ReadFile(wisdomPath)
	var wf hiveWisdomData
	json.Unmarshal(raw, &wf)
	if len(wf.Entries) != 1 {
		t.Fatalf("expected 1 entry after reinforce, got %d", len(wf.Entries))
	}
}

func TestHiveReadReturnsEntries(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	hubDir := t.TempDir()
	origHub := os.Getenv("AETHER_HUB_DIR")
	os.Setenv("AETHER_HUB_DIR", hubDir)
	defer os.Setenv("AETHER_HUB_DIR", origHub)

	rootCmd.SetArgs([]string{"hive-init"})
	_ = rootCmd.Execute()

	resetRootCmd(t)
	stdout = &buf
	buf.Reset()
	rootCmd.SetArgs([]string{"hive-store", "Guard nil store before use in cmd/helpers.go accessors", "testing", "aether"})
	_ = rootCmd.Execute()

	resetRootCmd(t)
	stdout = &buf
	buf.Reset()
	rootCmd.SetArgs([]string{"hive-read"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("hive-read returned error: %v", err)
	}

	out := parseEnvelope(t, buf.String())
	if out["ok"] != true {
		t.Fatalf("expected ok=true, got: %v", out["ok"])
	}
	result := out["result"].(map[string]interface{})
	entries, ok := result["entries"].([]interface{})
	if !ok {
		t.Fatalf("expected entries array, got: %T", result["entries"])
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}

func TestHiveAbstractRemovesRepoName(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"hive-abstract", "src/main.go in aether", "--source-repo", "aether"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("hive-abstract returned error: %v", err)
	}

	out := parseEnvelope(t, buf.String())
	if out["ok"] != true {
		t.Fatalf("expected ok=true, got: %v", out["ok"])
	}
	result := out["result"].(map[string]interface{})
	abstracted := result["abstracted"].(string)
	if strings.Contains(abstracted, "aether") {
		t.Fatalf("expected repo name removed from abstracted, got: %q", abstracted)
	}
}

// The v1.25 multi-agent review found "Always write tests first" (naming no
// file, command, or error) had entered the real hive via seal promotion,
// bypassing the admissibility gate. Every hive write now passes through the
// gate at the storeHiveWisdomEntry chokepoint — pin it.
func TestHiveStoreRejectsInadmissibleWisdom(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf
	hubDir := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", hubDir)
	s, _ := newTestStore(t)
	store = s

	rootCmd.SetArgs([]string{"hive-init"})
	_ = rootCmd.Execute()
	buf.Reset()

	rootCmd.SetArgs([]string{"hive-store", "Always write tests first because quality matters a lot", "testing", "aether"})
	_ = rootCmd.Execute()
	combined := buf.String() + errBuf.String()
	if !strings.Contains(combined, "not admissible") {
		t.Fatalf("anchor-free wisdom entered the hive without rejection: %s", combined)
	}
}

// TestSealPromotionSurvivesWithRepoNameInText is the end-to-end proof for the
// defect that kept the hive permanently empty.
//
// Seal promotes high-confidence instincts to the cross-colony hive. On the way
// it swaps the source repository's name for a placeholder, then sanitizes the
// result. The placeholder used to be "<repo>", which the sanitizer classifies
// as an XML structural tag and rejects — so any learning that named its own
// repository was silently discarded. Learnings drawn from a repository almost
// always name it, so in practice nothing was ever promoted, while the read path
// kept injecting an empty hive section into every worker dispatch.
//
// This asserts the whole path a real seal takes: repo-named text in, a stored
// entry out, with the name generalised away and no trace of the old tag shape.
func TestSealPromotionSurvivesWithRepoNameInText(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	hubDir := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", hubDir)
	s, _ := newTestStore(t)
	store = s

	const sourceRepo = "Aether"
	// Deliberately names its own repo, which is the case that used to be lost.
	// It also names a path and a command so it clears the admissibility gate —
	// abstraction must generalise the repo name without destroying the anchors
	// that make an entry falsifiable later.
	const learning = "Aether hub sync leaves cmd/hive.go stale unless aether publish runs first, so downstream repos keep the old binary"

	if err := promoteToHive(learning, "tooling", sourceRepo, 0.9); err != nil {
		t.Fatalf("promotion of a learning naming its own repo failed: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(hubDir, "hive", "wisdom.json"))
	if err != nil {
		t.Fatalf("wisdom.json unreadable after promotion: %v", err)
	}
	var wf hiveWisdomData
	if err := json.Unmarshal(raw, &wf); err != nil {
		t.Fatalf("wisdom.json malformed: %v", err)
	}
	if len(wf.Entries) != 1 {
		t.Fatalf("expected exactly 1 promoted entry, got %d — the hive is still swallowing promotions", len(wf.Entries))
	}

	got := wf.Entries[0].Text
	if strings.Contains(got, sourceRepo) {
		t.Errorf("source repo name was not generalised away: %q", got)
	}
	if !strings.Contains(got, colony.RepoPlaceholder) {
		t.Errorf("expected the placeholder %q in the stored entry, got %q", colony.RepoPlaceholder, got)
	}
	if strings.Contains(got, "<repo>") {
		t.Errorf("stored entry still carries the sanitizer-forbidden tag shape: %q", got)
	}

	// Abstraction must not strip the anchors that make an entry verifiable, or
	// promotion trades a sanitizer rejection for an admissibility rejection and
	// the hive stays empty for a different reason.
	if admissible, reason := memory.IsAdmissibleInstinctContent(got); !admissible {
		t.Errorf("abstraction destroyed the entry's falsifiability: %s (%q)", reason, got)
	}
}
