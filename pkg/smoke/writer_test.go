package smoke

import (
	"os"
	"testing"

	"github.com/calcosmic/Aether/pkg/storage"
)

func TestWritePlatformHealthCreatesFile(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	failed := []string{"cmd-a", "cmd-b"}
	mismatches := []FlagMismatch{
		{Command: "x", Flag: "--y", Expected: "registered", Actual: "missing", Severity: "warning"},
	}
	alignment := &DocCLIAlignment{
		CommandsChecked:      5,
		HostCriticalChecked:  2,
		HostCriticalFailures: []FlagMismatch{},
		Warnings:             []FlagMismatch{},
	}

	if err := WritePlatformHealth(s, failed, mismatches, alignment); err != nil {
		t.Fatalf("WritePlatformHealth failed: %v", err)
	}

	var loaded map[string]interface{}
	if err := s.LoadJSON("platform-health.json", &loaded); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if _, ok := loaded["failed_commands"]; !ok {
		t.Error("missing failed_commands")
	}
	if _, ok := loaded["flag_mismatches"]; !ok {
		t.Error("missing flag_mismatches")
	}
	if _, ok := loaded["doc_cli_alignment"]; !ok {
		t.Error("missing doc_cli_alignment")
	}
}

func TestWritePlatformHealthPreservesExisting(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	// Seed an existing platform-health.json
	old := map[string]interface{}{
		"failed_commands": []string{"old-cmd"},
		"flag_mismatches": []FlagMismatch{{Command: "old", Flag: "--old", Severity: "warning"}},
		"extra_key":       "should remain",
	}
	if err := s.SaveJSON("platform-health.json", old); err != nil {
		t.Fatal(err)
	}

	newFailed := []string{"new-cmd"}
	newMismatches := []FlagMismatch{{Command: "new", Flag: "--new", Severity: "blocking"}}
	alignment := &DocCLIAlignment{CommandsChecked: 3, HostCriticalChecked: 1}

	if err := WritePlatformHealth(s, newFailed, newMismatches, alignment); err != nil {
		t.Fatalf("WritePlatformHealth failed: %v", err)
	}

	var loaded map[string]interface{}
	if err := s.LoadJSON("platform-health.json", &loaded); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	// extra_key should survive because it's not one of the known keys
	if _, ok := loaded["extra_key"]; !ok {
		t.Error("extra_key was dropped")
	}
	if _, ok := loaded["doc_cli_alignment"]; !ok {
		t.Error("missing doc_cli_alignment")
	}
}

func TestWritePlatformHealthNilAlignment(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	if err := WritePlatformHealth(s, []string{}, []FlagMismatch{}, nil); err != nil {
		t.Fatalf("WritePlatformHealth failed: %v", err)
	}

	var loaded map[string]interface{}
	if err := s.LoadJSON("platform-health.json", &loaded); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if _, ok := loaded["doc_cli_alignment"]; ok {
		t.Error("doc_cli_alignment should not be present when alignment is nil")
	}
}
