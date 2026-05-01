package learn

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/calcosmic/Aether/pkg/storage"
)

func TestExportPack(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(filepath.Join(dir, "data"))
	if err != nil {
		t.Fatal(err)
	}
	cs := NewColonyStore(store)

	// Add test entries
	if err := cs.Add(Entry{
		Content:        "Prefer early returns over deep nesting",
		Evidence:       Evidence{Confidence: 0.9},
		Classification: ClassHiveShareable,
		Phase:          1,
	}); err != nil {
		t.Fatal(err)
	}
	if err := cs.Add(Entry{
		Content:        "Phase 2 completed successfully",
		Evidence:       Evidence{Confidence: 0.85},
		Classification: ClassRepoLocal,
		Phase:          2,
	}); err != nil {
		t.Fatal(err)
	}

	outputPath := filepath.Join(dir, "learning-pack.json")
	path, report, err := ExportPack(cs, outputPath)
	if err != nil {
		t.Fatalf("ExportPack failed: %v", err)
	}
	if path != outputPath {
		t.Errorf("path = %q, want %q", path, outputPath)
	}

	// Verify the file exists
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read export file: %v", err)
	}
	if len(data) == 0 {
		t.Error("export file is empty")
	}

	// Verify manifest structure
	var manifest ExportManifest
	if err := unmarshalJSON(data, &manifest); err != nil {
		t.Fatalf("failed to parse manifest: %v", err)
	}
	if manifest.EntryCount != 2 {
		t.Errorf("EntryCount = %d, want 2", manifest.EntryCount)
	}
	if len(manifest.Entries) != 2 {
		t.Errorf("len(Entries) = %d, want 2", len(manifest.Entries))
	}
	if manifest.ExportedAt == "" {
		t.Error("ExportedAt is empty")
	}
	if len(report) != 0 {
		t.Errorf("report = %v, want empty (no redactions)", report)
	}
}

func TestExportPack_BlockedEntriesSkipped(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(filepath.Join(dir, "data"))
	if err != nil {
		t.Fatal(err)
	}
	cs := NewColonyStore(store)

	// Add a safe entry
	if err := cs.Add(Entry{
		Content:        "Use table-driven tests for multiple cases",
		Evidence:       Evidence{Confidence: 0.88},
		Classification: ClassHiveShareable,
		Phase:          1,
	}); err != nil {
		t.Fatal(err)
	}

	// Add an entry with an API key (should be blocked)
	if err := cs.Add(Entry{
		Content:        "Set API_KEY=sk-abc123456789012345678901234567890 for auth",
		Evidence:       Evidence{Confidence: 0.75},
		Classification: ClassRepoLocal,
		Phase:          2,
	}); err != nil {
		t.Fatal(err)
	}

	outputPath := filepath.Join(dir, "learning-pack.json")
	_, report, err := ExportPack(cs, outputPath)
	if err != nil {
		t.Fatalf("ExportPack failed: %v", err)
	}

	// Blocked entry should be reported
	foundBlocked := false
	for _, r := range report {
		if r != "" {
			foundBlocked = true
			break
		}
	}
	if !foundBlocked {
		t.Error("expected blocked entry in redaction report, got none")
	}

	// Manifest should only contain the safe entry
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read export file: %v", err)
	}
	var manifest ExportManifest
	if err := unmarshalJSON(data, &manifest); err != nil {
		t.Fatalf("failed to parse manifest: %v", err)
	}
	if manifest.EntryCount != 1 {
		t.Errorf("EntryCount = %d, want 1 (blocked entry excluded)", manifest.EntryCount)
	}
}

func TestExportPack_PathRedaction(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(filepath.Join(dir, "data"))
	if err != nil {
		t.Fatal(err)
	}
	cs := NewColonyStore(store)

	// Add entry with home directory path
	if err := cs.Add(Entry{
		Content:        "Fixed bug in /Users/testuser/projects/myapp/src/main.go",
		Evidence:       Evidence{Confidence: 0.82},
		Classification: ClassRepoLocal,
		Phase:          1,
	}); err != nil {
		t.Fatal(err)
	}

	outputPath := filepath.Join(dir, "learning-pack.json")
	_, report, err := ExportPack(cs, outputPath)
	if err != nil {
		t.Fatalf("ExportPack failed: %v", err)
	}

	// Path should be redacted
	foundRedaction := false
	for _, r := range report {
		if r != "" {
			foundRedaction = true
			break
		}
	}
	if !foundRedaction {
		t.Error("expected path redaction in report, got none")
	}

	// Verify content was actually redacted
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read export file: %v", err)
	}
	content := string(data)
	if containsString(content, "/Users/testuser") {
		t.Error("exported content contains unredacted home path")
	}
	if !containsString(content, "[REDACTED_PATH]") {
		t.Error("exported content does not contain [REDACTED_PATH]")
	}
}

func TestImportPreview(t *testing.T) {
	dir := t.TempDir()

	// Create a pack file
	manifest := ExportManifest{
		ExportedAt: "2026-05-01T00:00:00Z",
		EntryCount: 1,
		Entries: []Entry{
			{
				Content:        "Test-driven development catches bugs early",
				Evidence:       Evidence{Confidence: 0.91},
				Classification: ClassHiveShareable,
				Phase:          1,
			},
		},
		RedactionReport: []string{},
	}
	data := marshalJSONIndent(manifest)
	packPath := filepath.Join(dir, "pack.json")
	if err := os.WriteFile(packPath, append(data, '\n'), 0644); err != nil {
		t.Fatal(err)
	}

	entries, report, err := ImportPreview(packPath)
	if err != nil {
		t.Fatalf("ImportPreview failed: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("len(entries) = %d, want 1", len(entries))
	}
	if entries[0].Content != "Test-driven development catches bugs early" {
		t.Errorf("entry content = %q, want 'Test-driven development catches bugs early'", entries[0].Content)
	}
	if len(report) != 0 {
		t.Errorf("report = %v, want empty", report)
	}
}

func TestImportPack(t *testing.T) {
	dir := t.TempDir()

	// Create source pack
	manifest := ExportManifest{
		ExportedAt: "2026-05-01T00:00:00Z",
		EntryCount: 2,
		Entries: []Entry{
			{
				ID:             "lrn_old_1",
				Content:        "Keep functions short and focused",
				Evidence:       Evidence{Confidence: 0.87},
				Classification: ClassHiveShareable,
				Phase:          1,
			},
			{
				ID:             "lrn_old_2",
				Content:        "Name variables for clarity not brevity",
				Evidence:       Evidence{Confidence: 0.83},
				Classification: ClassRepoLocal,
				Phase:          2,
			},
		},
	}
	data := marshalJSONIndent(manifest)
	packPath := filepath.Join(dir, "pack.json")
	if err := os.WriteFile(packPath, append(data, '\n'), 0644); err != nil {
		t.Fatal(err)
	}

	// Import into a new store
	store, err := storage.NewStore(filepath.Join(dir, "data"))
	if err != nil {
		t.Fatal(err)
	}
	cs := NewColonyStore(store)

	count, err := ImportPack(cs, packPath)
	if err != nil {
		t.Fatalf("ImportPack failed: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}

	// Verify entries were imported with new IDs
	entries, err := cs.List(EntryFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Errorf("len(entries) = %d, want 2", len(entries))
	}

	// Verify old IDs were replaced (no collision)
	for _, e := range entries {
		if e.ID == "lrn_old_1" || e.ID == "lrn_old_2" {
			t.Errorf("imported entry retained old ID %q (should have new ID)", e.ID)
		}
	}
}

func TestHiveStore_AddOnlyHiveShareable(t *testing.T) {
	dir := t.TempDir()
	hs := NewHiveStore(dir, "testrepo")

	// Blocked entry should be rejected
	err := hs.Add(Entry{
		Content:        "some content",
		Classification: ClassBlocked,
	})
	if err == nil {
		t.Error("expected error for blocked entry, got nil")
	}

	// Repo-local entry should be rejected
	err = hs.Add(Entry{
		Content:        "some content",
		Classification: ClassRepoLocal,
	})
	if err == nil {
		t.Error("expected error for repo-local entry, got nil")
	}

	// Hive-shareable entry should succeed
	err = hs.Add(Entry{
		Content:        "Generic learning about testing patterns",
		Classification: ClassHiveShareable,
		Confidence:     0.85,
	})
	if err != nil {
		t.Errorf("expected no error for hive-shareable entry, got: %v", err)
	}
}

func TestHiveStore_AddAbstractsContent(t *testing.T) {
	dir := t.TempDir()
	hs := NewHiveStore(dir, "/Users/testuser/myrepo")

	err := hs.Add(Entry{
		Content:        "Fixed bug in src/main.go and pkg/utils/helper.go in /Users/testuser/myrepo",
		Classification: ClassHiveShareable,
		Confidence:     0.8,
	})
	if err != nil {
		t.Fatal(err)
	}

	entries, err := hs.List(EntryFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}

	// Verify content was abstracted
	content := entries[0].Content
	if containsString(content, "src/") {
		t.Error("content still contains src/ prefix (should be abstracted)")
	}
	if containsString(content, "pkg/") {
		t.Error("content still contains pkg/ prefix (should be abstracted)")
	}
	if containsString(content, "/Users/testuser/myrepo") {
		t.Error("content still contains repo path (should be abstracted)")
	}
	if !containsString(content, "<repo>") {
		t.Error("content does not contain <repo> placeholder")
	}
}

func TestHiveStore_Get(t *testing.T) {
	dir := t.TempDir()
	hs := NewHiveStore(dir, "testrepo")

	hs.Add(Entry{
		Content:        "Generic testing pattern",
		Classification: ClassHiveShareable,
		Confidence:     0.9,
	})

	entries, _ := hs.List(EntryFilter{Limit: 1})
	if len(entries) == 0 {
		t.Fatal("no entries after Add")
	}

	found, err := hs.Get(entries[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if found == nil {
		t.Fatal("Get returned nil for existing entry")
	}
	if found.Content != "Generic testing pattern" {
		t.Errorf("content = %q, want 'Generic testing pattern'", found.Content)
	}

	// Non-existent ID
	notFound, err := hs.Get("nonexistent")
	if err != nil {
		t.Errorf("expected no error for non-existent ID, got: %v", err)
	}
	if notFound != nil {
		t.Error("expected nil for non-existent entry")
	}
}

func TestHiveStore_LRUEviction(t *testing.T) {
	dir := t.TempDir()
	hs := NewHiveStore(dir, "testrepo")

	// Add entries up to max capacity
	for i := 0; i < maxHiveWisdomEntries+1; i++ {
		err := hs.Add(Entry{
			Content:        fmt.Sprintf("Learning entry number %d about generic patterns", i),
			Classification: ClassHiveShareable,
			Confidence:     0.8,
			Evidence:       Evidence{Scope: "general"},
		})
		if err != nil {
			t.Fatalf("Add(%d) failed: %v", i, err)
		}
	}

	entries, err := hs.List(EntryFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != maxHiveWisdomEntries {
		t.Errorf("len(entries) = %d, want %d (LRU eviction)", len(entries), maxHiveWisdomEntries)
	}
}

func TestHiveStore_Remove(t *testing.T) {
	dir := t.TempDir()
	hs := NewHiveStore(dir, "testrepo")

	hs.Add(Entry{
		Content:        "Entry to remove",
		Classification: ClassHiveShareable,
		Confidence:     0.8,
	})

	entries, _ := hs.List(EntryFilter{Limit: 1})
	if len(entries) == 0 {
		t.Fatal("no entries after Add")
	}

	err := hs.Remove(entries[0].ID)
	if err != nil {
		t.Errorf("Remove failed: %v", err)
	}

	entries, _ = hs.List(EntryFilter{})
	if len(entries) != 0 {
		t.Errorf("len(entries) = %d, want 0 after Remove", len(entries))
	}
}

func TestHiveStore_ConfidenceBoost(t *testing.T) {
	dir := t.TempDir()
	hs := NewHiveStore(dir, "testrepo")

	hs.Add(Entry{
		Content:        "Test coverage should be high",
		Classification: ClassHiveShareable,
		Confidence:     0.8,
		Evidence:       Evidence{Scope: "general"},
	})

	// Add same content with higher confidence
	hs.Add(Entry{
		Content:        "Test coverage should be high",
		Classification: ClassHiveShareable,
		Confidence:     0.95,
		Evidence:       Evidence{Scope: "general"},
	})

	entries, _ := hs.List(EntryFilter{})
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1 (dedup)", len(entries))
	}
	if entries[0].Confidence < 0.95 {
		t.Errorf("confidence = %.2f, want >= 0.95 (boosted)", entries[0].Confidence)
	}
}

func TestHermesConceptMap(t *testing.T) {
	if len(HermesConceptMap) == 0 {
		t.Error("HermesConceptMap is empty")
	}

	expectedKeys := []string{
		"MessageBus",
		"AgentMailbox",
		"KnowledgeFragment",
		"ConfidenceScore",
		"Propagation",
		"Consolidation",
		"PrivacyFilter",
	}
	for _, key := range expectedKeys {
		if _, ok := HermesConceptMap[key]; !ok {
			t.Errorf("HermesConceptMap missing key %q", key)
		}
	}
}

// Helper functions

func unmarshalJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func marshalJSONIndent(v interface{}) []byte {
	data, _ := json.MarshalIndent(v, "", "  ")
	return data
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
