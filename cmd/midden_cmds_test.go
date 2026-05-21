package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestMiddenRecentFailuresReturnsNewestFirst(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Seed midden with multiple entries
	mf := colony.MiddenFile{
		Version: "1.0.0",
		Entries: []colony.MiddenEntry{
			{ID: "old", Timestamp: "2026-01-01T00:00:00Z", Category: "build", Message: "old failure", Source: "test"},
			{ID: "new", Timestamp: "2026-05-01T00:00:00Z", Category: "test", Message: "new failure", Source: "test"},
			{ID: "mid", Timestamp: "2026-03-01T00:00:00Z", Category: "build", Message: "mid failure", Source: "test"},
		},
	}
	if err := s.SaveJSON("midden.json", mf); err != nil {
		t.Fatalf("seed midden.json: %v", err)
	}

	rootCmd.SetArgs([]string{"midden-recent-failures", "--limit", "2"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env)
	}

	result := env["result"].(map[string]interface{})
	if result["total"] != float64(2) {
		t.Errorf("total = %v, want 2", result["total"])
	}

	entries := result["entries"].([]interface{})
	if len(entries) != 2 {
		t.Fatalf("entries length = %d, want 2", len(entries))
	}
	first := entries[0].(map[string]interface{})
	if first["id"] != "new" {
		t.Errorf("first entry id = %v, want new (newest first)", first["id"])
	}
}

func TestMiddenReviewReturnsUnacknowledged(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	ack := true
	mf := colony.MiddenFile{
		Version: "1.0.0",
		Entries: []colony.MiddenEntry{
			{ID: "unack1", Timestamp: "2026-01-01T00:00:00Z", Category: "build", Message: "unack", Source: "test"},
			{ID: "ack1", Timestamp: "2026-01-02T00:00:00Z", Category: "build", Message: "ack", Source: "test", Acknowledged: &ack},
			{ID: "unack2", Timestamp: "2026-01-03T00:00:00Z", Category: "test", Message: "unack2", Source: "test"},
		},
	}
	if err := s.SaveJSON("midden.json", mf); err != nil {
		t.Fatalf("seed midden.json: %v", err)
	}

	rootCmd.SetArgs([]string{"midden-review"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env)
	}

	result := env["result"].(map[string]interface{})
	if result["total"] != float64(2) {
		t.Errorf("total = %v, want 2", result["total"])
	}

	groups := result["groups"].(map[string]interface{})
	if len(groups) != 2 {
		t.Errorf("group count = %d, want 2", len(groups))
	}
}

func TestMiddenAcknowledgeByID(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	mf := colony.MiddenFile{
		Version: "1.0.0",
		Entries: []colony.MiddenEntry{
			{ID: "ack-target", Timestamp: "2026-01-01T00:00:00Z", Category: "build", Message: "target", Source: "test"},
			{ID: "other", Timestamp: "2026-01-02T00:00:00Z", Category: "build", Message: "other", Source: "test"},
		},
	}
	if err := s.SaveJSON("midden.json", mf); err != nil {
		t.Fatalf("seed midden.json: %v", err)
	}

	rootCmd.SetArgs([]string{
		"midden-acknowledge",
		"--id", "ack-target",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env)
	}

	result := env["result"].(map[string]interface{})
	if result["acknowledged"] != float64(1) {
		t.Errorf("acknowledged = %v, want 1", result["acknowledged"])
	}

	// Verify filesystem state
	var updated colony.MiddenFile
	if err := s.LoadJSON("midden.json", &updated); err != nil {
		t.Fatalf("load midden.json: %v", err)
	}
	if updated.Entries[0].Acknowledged == nil || !*updated.Entries[0].Acknowledged {
		t.Error("first entry should be acknowledged")
	}
	if updated.Entries[1].Acknowledged != nil && *updated.Entries[1].Acknowledged {
		t.Error("second entry should NOT be acknowledged")
	}
}

func TestMiddenAcknowledgeByCategory(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	mf := colony.MiddenFile{
		Version: "1.0.0",
		Entries: []colony.MiddenEntry{
			{ID: "c1", Timestamp: "2026-01-01T00:00:00Z", Category: "security", Message: "sec1", Source: "test"},
			{ID: "c2", Timestamp: "2026-01-02T00:00:00Z", Category: "security", Message: "sec2", Source: "test"},
			{ID: "c3", Timestamp: "2026-01-03T00:00:00Z", Category: "build", Message: "build1", Source: "test"},
		},
	}
	if err := s.SaveJSON("midden.json", mf); err != nil {
		t.Fatalf("seed midden.json: %v", err)
	}

	rootCmd.SetArgs([]string{
		"midden-acknowledge",
		"--category", "security",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env)
	}

	result := env["result"].(map[string]interface{})
	if result["acknowledged"] != float64(2) {
		t.Errorf("acknowledged = %v, want 2", result["acknowledged"])
	}

	// Verify filesystem state
	var updated colony.MiddenFile
	if err := s.LoadJSON("midden.json", &updated); err != nil {
		t.Fatalf("load midden.json: %v", err)
	}
	for _, e := range updated.Entries {
		if e.Category == "security" && (e.Acknowledged == nil || !*e.Acknowledged) {
			t.Errorf("security entry %s should be acknowledged", e.ID)
		}
		if e.Category == "build" && e.Acknowledged != nil && *e.Acknowledged {
			t.Errorf("build entry %s should NOT be acknowledged", e.ID)
		}
	}
}

func TestMiddenSearchFindsByText(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	mf := colony.MiddenFile{
		Version: "1.0.0",
		Entries: []colony.MiddenEntry{
			{ID: "s1", Timestamp: "2026-01-01T00:00:00Z", Category: "build", Message: "compiler error in main.go", Source: "test"},
			{ID: "s2", Timestamp: "2026-01-02T00:00:00Z", Category: "test", Message: "timeout in integration suite", Source: "test"},
			{ID: "s3", Timestamp: "2026-01-03T00:00:00Z", Category: "build", Message: "linker failure", Source: "test"},
		},
	}
	if err := s.SaveJSON("midden.json", mf); err != nil {
		t.Fatalf("seed midden.json: %v", err)
	}

	rootCmd.SetArgs([]string{
		"midden-search",
		"--query", "compiler",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env)
	}

	result := env["result"].(map[string]interface{})
	if result["total"] != float64(1) {
		t.Errorf("total = %v, want 1", result["total"])
	}

	entries := result["entries"].([]interface{})
	if len(entries) != 1 {
		t.Fatalf("entries length = %d, want 1", len(entries))
	}
	entry := entries[0].(map[string]interface{})
	if entry["id"] != "s1" {
		t.Errorf("id = %v, want s1", entry["id"])
	}
}

func TestMiddenTagAddsTagToEntry(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	mf := colony.MiddenFile{
		Version: "1.0.0",
		Entries: []colony.MiddenEntry{
			{ID: "tag-me", Timestamp: "2026-01-01T00:00:00Z", Category: "build", Message: "msg", Source: "test", Tags: []string{"existing"}},
		},
	}
	if err := s.SaveJSON("midden.json", mf); err != nil {
		t.Fatalf("seed midden.json: %v", err)
	}

	rootCmd.SetArgs([]string{
		"midden-tag",
		"--id", "tag-me",
		"--tag", "critical",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env)
	}

	result := env["result"].(map[string]interface{})
	if result["tagged"] != true {
		t.Errorf("tagged = %v, want true", result["tagged"])
	}

	// Verify filesystem state
	var updated colony.MiddenFile
	if err := s.LoadJSON("midden.json", &updated); err != nil {
		t.Fatalf("load midden.json: %v", err)
	}
	if len(updated.Entries[0].Tags) != 2 {
		t.Errorf("tag count = %d, want 2", len(updated.Entries[0].Tags))
	}
	found := false
	for _, tag := range updated.Entries[0].Tags {
		if tag == "critical" {
			found = true
			break
		}
	}
	if !found {
		t.Error("tags missing 'critical'")
	}
}
