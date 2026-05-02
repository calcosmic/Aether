package learn

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSkillCreate verifies that creating a skill writes SKILL.md and inserts SQLite metadata.
func TestSkillCreate(t *testing.T) {
	store, dir := newTestSQLiteStore(t)
	defer store.Close()

	svc := NewSkillService(store.DB(), dir)
	meta := SkillMetadata{
		Name:       "test-skill",
		Confidence: 0.8,
	}

	err := svc.CreateSkill(meta, "This skill helps with testing patterns")
	if err != nil {
		t.Fatalf("CreateSkill: %v", err)
	}

	// Verify SKILL.md exists at .aether/hive/skills/active/test-skill/SKILL.md
	skillPath := filepath.Join(dir, ".aether", "hive", "skills", "active", "test-skill", "SKILL.md")
	if _, err := os.Stat(skillPath); err != nil {
		t.Fatalf("SKILL.md not found: %v", err)
	}

	// Verify SQLite row exists with stage="active"
	result, err := svc.GetSkill("test-skill")
	if err != nil {
		t.Fatalf("GetSkill: %v", err)
	}
	if result == nil {
		t.Fatal("GetSkill returned nil")
	}
	if result.Stage != SkillStageActive {
		t.Errorf("stage = %q, want %q", result.Stage, SkillStageActive)
	}
	if result.Name != "test-skill" {
		t.Errorf("name = %q, want %q", result.Name, "test-skill")
	}
}

// TestSkillCreate_EvidenceFrontmatter verifies YAML frontmatter fields in SKILL.md.
func TestSkillCreate_EvidenceFrontmatter(t *testing.T) {
	store, dir := newTestSQLiteStore(t)
	defer store.Close()

	svc := NewSkillService(store.DB(), dir)
	meta := SkillMetadata{
		Name:        "evidence-skill",
		SourceRunID: "run-123",
		Confidence:  0.85,
		AutoCreated: true,
	}

	err := svc.CreateSkill(meta, "A skill created by the colony learning system")
	if err != nil {
		t.Fatalf("CreateSkill: %v", err)
	}

	// Read SKILL.md and verify frontmatter
	skillPath := filepath.Join(dir, ".aether", "hive", "skills", "active", "evidence-skill", "SKILL.md")
	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "source_run_id: run-123") {
		t.Errorf("frontmatter missing source_run_id: run-123")
	}
	if !strings.Contains(content, "confidence: 0.85") {
		t.Errorf("frontmatter missing confidence: 0.85")
	}
	if !strings.Contains(content, "auto_created: true") {
		t.Errorf("frontmatter missing auto_created: true")
	}
	if !strings.Contains(content, "---\n") {
		t.Errorf("frontmatter missing YAML delimiters")
	}
}

// TestSkillGet verifies retrieving skill metadata by name.
func TestSkillGet(t *testing.T) {
	store, dir := newTestSQLiteStore(t)
	defer store.Close()

	svc := NewSkillService(store.DB(), dir)
	meta := SkillMetadata{
		Name:       "get-test",
		Confidence: 0.9,
	}

	if err := svc.CreateSkill(meta, "content for get test"); err != nil {
		t.Fatalf("CreateSkill: %v", err)
	}

	result, err := svc.GetSkill("get-test")
	if err != nil {
		t.Fatalf("GetSkill: %v", err)
	}
	if result == nil {
		t.Fatal("GetSkill returned nil")
	}
	if result.Name != "get-test" {
		t.Errorf("name = %q, want %q", result.Name, "get-test")
	}
	if result.Stage != SkillStageActive {
		t.Errorf("stage = %q, want %q", result.Stage, SkillStageActive)
	}
	if result.Confidence != 0.9 {
		t.Errorf("confidence = %f, want %f", result.Confidence, 0.9)
	}
}

// TestSkillPatch verifies patching a skill updates content and increments patch_count.
func TestSkillPatch(t *testing.T) {
	store, dir := newTestSQLiteStore(t)
	defer store.Close()

	svc := NewSkillService(store.DB(), dir)
	meta := SkillMetadata{Name: "patch-test", Confidence: 0.7}

	if err := svc.CreateSkill(meta, "original content"); err != nil {
		t.Fatalf("CreateSkill: %v", err)
	}

	newContent := "---\nname: patch-test\n---\n\nupdated content"
	if err := svc.PatchSkill("patch-test", newContent); err != nil {
		t.Fatalf("PatchSkill: %v", err)
	}

	// Verify SKILL.md content updated
	result, err := svc.GetSkill("patch-test")
	if err != nil {
		t.Fatalf("GetSkill: %v", err)
	}
	data, err := os.ReadFile(result.FilePath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != newContent {
		t.Errorf("file content = %q, want %q", string(data), newContent)
	}

	// Verify patch_count incremented
	if result.PatchCount != 1 {
		t.Errorf("patch_count = %d, want 1", result.PatchCount)
	}
}

// TestSkillArchive verifies archiving moves SKILL.md and updates stage.
func TestSkillArchive(t *testing.T) {
	store, dir := newTestSQLiteStore(t)
	defer store.Close()

	svc := NewSkillService(store.DB(), dir)
	meta := SkillMetadata{Name: "archive-test", Confidence: 0.6}

	if err := svc.CreateSkill(meta, "to be archived"); err != nil {
		t.Fatalf("CreateSkill: %v", err)
	}

	if err := svc.ArchiveSkill("archive-test"); err != nil {
		t.Fatalf("ArchiveSkill: %v", err)
	}

	// Verify SKILL.md moved to archived/
	archivedPath := filepath.Join(dir, ".aether", "hive", "skills", "archived", "archive-test", "SKILL.md")
	if _, err := os.Stat(archivedPath); err != nil {
		t.Fatalf("archived SKILL.md not found: %v", err)
	}

	// Verify old path no longer exists
	activePath := filepath.Join(dir, ".aether", "hive", "skills", "active", "archive-test", "SKILL.md")
	if _, err := os.Stat(activePath); err == nil {
		t.Fatal("active SKILL.md still exists after archive")
	}

	// Verify stage updated in SQLite
	result, err := svc.GetSkill("archive-test")
	if err != nil {
		t.Fatalf("GetSkill: %v", err)
	}
	if result.Stage != SkillStageArchived {
		t.Errorf("stage = %q, want %q", result.Stage, SkillStageArchived)
	}
}

// TestSkillPin verifies pinning prevents archiving.
func TestSkillPin(t *testing.T) {
	store, dir := newTestSQLiteStore(t)
	defer store.Close()

	svc := NewSkillService(store.DB(), dir)
	meta := SkillMetadata{Name: "pin-test", Confidence: 0.75}

	if err := svc.CreateSkill(meta, "important skill"); err != nil {
		t.Fatalf("CreateSkill: %v", err)
	}

	// Pin the skill
	if err := svc.PinSkill("pin-test"); err != nil {
		t.Fatalf("PinSkill: %v", err)
	}

	// Verify pinned in SQLite
	result, err := svc.GetSkill("pin-test")
	if err != nil {
		t.Fatalf("GetSkill: %v", err)
	}
	if !result.Pinned {
		t.Error("skill should be pinned")
	}

	// Attempt to archive pinned skill -- should fail
	err = svc.ArchiveSkill("pin-test")
	if err == nil {
		t.Fatal("ArchiveSkill should fail for pinned skill")
	}
	if !strings.Contains(err.Error(), "pinned") {
		t.Errorf("error = %q, should mention pinned", err.Error())
	}
}

// TestSkillList verifies listing skills with optional stage filter.
func TestSkillList(t *testing.T) {
	store, dir := newTestSQLiteStore(t)
	defer store.Close()

	svc := NewSkillService(store.DB(), dir)

	// Create 3 skills
	for _, name := range []string{"skill-a", "skill-b", "skill-c"} {
		if err := svc.CreateSkill(SkillMetadata{Name: name, Confidence: 0.5}, "content"); err != nil {
			t.Fatalf("CreateSkill(%s): %v", name, err)
		}
	}

	// Archive one
	if err := svc.ArchiveSkill("skill-c"); err != nil {
		t.Fatalf("ArchiveSkill: %v", err)
	}

	// List all
	all, err := svc.ListSkills("")
	if err != nil {
		t.Fatalf("ListSkills(''): %v", err)
	}
	if len(all) != 3 {
		t.Errorf("ListSkills('') = %d, want 3", len(all))
	}

	// List active only
	active, err := svc.ListSkills(SkillStageActive)
	if err != nil {
		t.Fatalf("ListSkills('active'): %v", err)
	}
	if len(active) != 2 {
		t.Errorf("ListSkills('active') = %d, want 2", len(active))
	}

	// List archived only
	archived, err := svc.ListSkills(SkillStageArchived)
	if err != nil {
		t.Fatalf("ListSkills('archived'): %v", err)
	}
	if len(archived) != 1 {
		t.Errorf("ListSkills('archived') = %d, want 1", len(archived))
	}
}

// TestSkillBuildIndex verifies progressive disclosure returns index-only entries.
func TestSkillBuildIndex(t *testing.T) {
	store, dir := newTestSQLiteStore(t)
	defer store.Close()

	svc := NewSkillService(store.DB(), dir)

	// Create 2 skills with content longer than 200 chars (progressive disclosure truncates description)
	longContentA := strings.Repeat("Detailed content about skill A. ", 15) // ~360 chars
	longContentB := strings.Repeat("Detailed content about skill B. ", 15)
	if err := svc.CreateSkill(SkillMetadata{Name: "idx-a", Confidence: 0.8}, longContentA); err != nil {
		t.Fatalf("CreateSkill: %v", err)
	}
	if err := svc.CreateSkill(SkillMetadata{Name: "idx-b", Confidence: 0.7}, longContentB); err != nil {
		t.Fatalf("CreateSkill: %v", err)
	}

	entries, err := svc.BuildSkillIndex()
	if err != nil {
		t.Fatalf("BuildSkillIndex: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("BuildSkillIndex returned %d entries, want 2", len(entries))
	}

	// Verify each entry has Name, Description, FilePath
	for _, entry := range entries {
		if entry.Name == "" {
			t.Error("SkillIndexEntry.Name is empty")
		}
		if entry.FilePath == "" {
			t.Error("SkillIndexEntry.FilePath is empty")
		}
	}

	// Verify full content is NOT in the index entries (description truncated to 200 chars)
	for _, entry := range entries {
		if len(entry.Description) > 200 {
			t.Errorf("Description should be <= 200 chars, got %d: %q", len(entry.Description), entry.Description)
		}
	}
}

// TestSkillNameValidation verifies path traversal names are rejected.
func TestSkillNameValidation(t *testing.T) {
	store, dir := newTestSQLiteStore(t)
	defer store.Close()

	svc := NewSkillService(store.DB(), dir)

	badNames := []string{"bad/name", "../evil", "ok\x00bad"}
	for _, name := range badNames {
		err := svc.CreateSkill(SkillMetadata{Name: name}, "content")
		if err == nil {
			t.Errorf("CreateSkill(%q) should fail but didn't", name)
		}
	}

	// Empty name should also fail
	err := svc.CreateSkill(SkillMetadata{Name: ""}, "content")
	if err == nil {
		t.Error("CreateSkill('') should fail but didn't")
	}
}

// TestSkillPinnedPatchBlocked verifies pinned skills cannot be patched.
func TestSkillPinnedPatchBlocked(t *testing.T) {
	store, dir := newTestSQLiteStore(t)
	defer store.Close()

	svc := NewSkillService(store.DB(), dir)
	meta := SkillMetadata{Name: "pinned-patch", Confidence: 0.8}

	if err := svc.CreateSkill(meta, "original"); err != nil {
		t.Fatalf("CreateSkill: %v", err)
	}

	if err := svc.PinSkill("pinned-patch"); err != nil {
		t.Fatalf("PinSkill: %v", err)
	}

	err := svc.PatchSkill("pinned-patch", "new content")
	if err == nil {
		t.Fatal("PatchSkill should fail for pinned skill")
	}
	if !strings.Contains(err.Error(), "pinned") {
		t.Errorf("error = %q, should mention pinned", err.Error())
	}
}

// TestSkillCreatePinned verifies creating a pre-pinned skill.
func TestSkillCreatePinned(t *testing.T) {
	store, dir := newTestSQLiteStore(t)
	defer store.Close()

	svc := NewSkillService(store.DB(), dir)
	meta := SkillMetadata{
		Name:       "pre-pinned",
		Pinned:     true,
		Confidence: 0.9,
	}

	if err := svc.CreateSkill(meta, "pinned from creation"); err != nil {
		t.Fatalf("CreateSkill: %v", err)
	}

	result, err := svc.GetSkill("pre-pinned")
	if err != nil {
		t.Fatalf("GetSkill: %v", err)
	}
	if !result.Pinned {
		t.Error("skill should be pinned after creation with Pinned=true")
	}
}

// TestSkillGetNotFound verifies GetSkill returns nil for missing skills.
func TestSkillGetNotFound(t *testing.T) {
	store, dir := newTestSQLiteStore(t)
	defer store.Close()

	svc := NewSkillService(store.DB(), dir)
	result, err := svc.GetSkill("nonexistent")
	if err != nil {
		t.Fatalf("GetSkill: %v", err)
	}
	if result != nil {
		t.Error("GetSkill should return nil for nonexistent skill")
	}
}
