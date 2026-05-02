package learn

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// makeSkill is a test helper that creates a skill via SkillService.
func makeSkill(t *testing.T, db *SQLiteColonyStore, baseDir, name, content string) {
	t.Helper()
	svc := NewSkillService(db.DB(), baseDir)
	meta := SkillMetadata{
		Name:        name,
		Confidence:  0.75,
		AutoCreated: false,
	}
	if err := svc.CreateSkill(meta, content); err != nil {
		t.Fatalf("makeSkill(%q): %v", name, err)
	}
}

// ageSkill sets a skill's timestamps to the given time (for age simulation).
func ageSkill(t *testing.T, db *SQLiteColonyStore, name string, ts time.Time) {
	t.Helper()
	_, err := db.DB().Exec(`UPDATE skills SET last_used_at = ?, last_viewed_at = ?, created_at = ? WHERE name = ?`,
		ts.Format(time.RFC3339), ts.Format(time.RFC3339), ts.Format(time.RFC3339), name)
	if err != nil {
		t.Fatalf("ageSkill(%q): %v", name, err)
	}
}

// TestCuratorTransitionActiveToStale verifies that active skills older than 14 days
// transition to stale, and their SKILL.md files are moved.
func TestCuratorTransitionActiveToStale(t *testing.T) {
	store, dir := newTestSQLiteStore(t)
	defer store.Close()

	makeSkill(t, store, dir, "old-skill", "An old skill that should transition")

	fifteenDaysAgo := time.Now().UTC().AddDate(0, 0, -15)
	ageSkill(t, store, "old-skill", fifteenDaysAgo)

	curator := NewCurator(store.DB(), dir)
	count, err := curator.RunTransitions()
	if err != nil {
		t.Fatalf("RunTransitions: %v", err)
	}
	if count != 1 {
		t.Errorf("RunTransitions count = %d, want 1", count)
	}

	// Verify stage is now "stale" in SQLite
	svc := NewSkillService(store.DB(), dir)
	meta, err := svc.GetSkill("old-skill")
	if err != nil {
		t.Fatalf("GetSkill: %v", err)
	}
	if meta.Stage != SkillStageStale {
		t.Errorf("stage = %q, want %q", meta.Stage, SkillStageStale)
	}

	// Verify file moved to stale/ directory
	expectedFile := filepath.Join(skillDirForStage(dir, SkillStageStale), "old-skill", "SKILL.md")
	if _, err := os.Stat(expectedFile); err != nil {
		t.Errorf("SKILL.md not found in stale/ directory: %v", err)
	}

	// Verify file no longer in active/ directory
	oldFile := filepath.Join(skillDirForStage(dir, SkillStageActive), "old-skill", "SKILL.md")
	if _, err := os.Stat(oldFile); err == nil {
		t.Error("SKILL.md still exists in active/ directory after transition")
	}
}

// TestCuratorTransitionStaleToArchived verifies that stale skills older than 28 days
// transition to archived.
func TestCuratorTransitionStaleToArchived(t *testing.T) {
	store, dir := newTestSQLiteStore(t)
	defer store.Close()

	makeSkill(t, store, dir, "stale-skill", "A stale skill that should be archived")

	// Set to stale and age beyond 28 days
	twentyNineDaysAgo := time.Now().UTC().AddDate(0, 0, -29)
	store.DB().Exec(`UPDATE skills SET stage = ? WHERE name = ?`, SkillStageStale, "stale-skill")
	ageSkill(t, store, "stale-skill", twentyNineDaysAgo)

	// Move the file to stale/ directory first
	staleDir := filepath.Join(skillDirForStage(dir, SkillStageStale), "stale-skill")
	os.MkdirAll(staleDir, 0755)
	svc := NewSkillService(store.DB(), dir)
	meta, _ := svc.GetSkill("stale-skill")
	if meta != nil && meta.FilePath != "" {
		if _, err := os.Stat(meta.FilePath); err == nil {
			os.Rename(meta.FilePath, filepath.Join(staleDir, "SKILL.md"))
			store.DB().Exec(`UPDATE skills SET file_path = ? WHERE name = ?`,
				filepath.Join(staleDir, "SKILL.md"), "stale-skill")
		}
	}

	curator := NewCurator(store.DB(), dir)
	count, err := curator.RunTransitions()
	if err != nil {
		t.Fatalf("RunTransitions: %v", err)
	}
	if count != 1 {
		t.Errorf("RunTransitions count = %d, want 1", count)
	}

	meta, err = svc.GetSkill("stale-skill")
	if err != nil {
		t.Fatalf("GetSkill: %v", err)
	}
	if meta.Stage != SkillStageArchived {
		t.Errorf("stage = %q, want %q", meta.Stage, SkillStageArchived)
	}

	// Verify file moved to archived/ directory
	expectedFile := filepath.Join(skillDirForStage(dir, SkillStageArchived), "stale-skill", "SKILL.md")
	if _, err := os.Stat(expectedFile); err != nil {
		t.Errorf("SKILL.md not found in archived/ directory: %v", err)
	}
}

// TestCuratorSkipsRecentSkill verifies that recently used skills are NOT transitioned.
func TestCuratorSkipsRecentSkill(t *testing.T) {
	store, dir := newTestSQLiteStore(t)
	defer store.Close()

	makeSkill(t, store, dir, "recent-skill", "A recently used skill")

	// Create the skill, then set last_used_at to now (recently used)
	now := time.Now().UTC()
	store.DB().Exec(`UPDATE skills SET last_used_at = ? WHERE name = ?`, now.Format(time.RFC3339), "recent-skill")

	curator := NewCurator(store.DB(), dir)
	count, err := curator.RunTransitions()
	if err != nil {
		t.Fatalf("RunTransitions: %v", err)
	}
	if count != 0 {
		t.Errorf("RunTransitions count = %d, want 0 (recent skill should not be transitioned)", count)
	}

	svc := NewSkillService(store.DB(), dir)
	meta, _ := svc.GetSkill("recent-skill")
	if meta == nil {
		t.Fatal("skill not found")
	}
	if meta.Stage != SkillStageActive {
		t.Errorf("stage = %q, want %q", meta.Stage, SkillStageActive)
	}
}

// TestCuratorPinnedImmunity verifies that pinned skills are never transitioned
// regardless of age (D-09, SKIL-05).
func TestCuratorPinnedImmunity(t *testing.T) {
	store, dir := newTestSQLiteStore(t)
	defer store.Close()

	makeSkill(t, store, dir, "pinned-skill", "A pinned skill that should never transition")

	// Pin the skill
	svc := NewSkillService(store.DB(), dir)
	if err := svc.PinSkill("pinned-skill"); err != nil {
		t.Fatalf("PinSkill: %v", err)
	}

	// Age it far beyond 14 days
	thirtyDaysAgo := time.Now().UTC().AddDate(0, 0, -30)
	ageSkill(t, store, "pinned-skill", thirtyDaysAgo)

	curator := NewCurator(store.DB(), dir)
	count, err := curator.RunTransitions()
	if err != nil {
		t.Fatalf("RunTransitions: %v", err)
	}
	if count != 0 {
		t.Errorf("RunTransitions count = %d, want 0 (pinned skill should be immune)", count)
	}

	meta, _ := svc.GetSkill("pinned-skill")
	if meta == nil {
		t.Fatal("skill not found")
	}
	if meta.Stage != SkillStageActive {
		t.Errorf("stage = %q, want %q (pinned skill should stay active)", meta.Stage, SkillStageActive)
	}
}

// TestCuratorFileMove verifies that SKILL.md files are physically moved between
// stage directories on transition.
func TestCuratorFileMove(t *testing.T) {
	store, dir := newTestSQLiteStore(t)
	defer store.Close()

	makeSkill(t, store, dir, "file-move-skill", "Skill to verify file moves")

	// Verify file exists in active/
	activeFile := filepath.Join(skillDirForStage(dir, SkillStageActive), "file-move-skill", "SKILL.md")
	if _, err := os.Stat(activeFile); err != nil {
		t.Fatalf("SKILL.md not found in active/: %v", err)
	}

	// Age the skill
	fifteenDaysAgo := time.Now().UTC().AddDate(0, 0, -15)
	ageSkill(t, store, "file-move-skill", fifteenDaysAgo)

	curator := NewCurator(store.DB(), dir)
	_, err := curator.RunTransitions()
	if err != nil {
		t.Fatalf("RunTransitions: %v", err)
	}

	// Verify file moved to stale/
	staleFile := filepath.Join(skillDirForStage(dir, SkillStageStale), "file-move-skill", "SKILL.md")
	if _, err := os.Stat(staleFile); err != nil {
		t.Errorf("SKILL.md not found in stale/ after transition: %v", err)
	}

	// Verify file_path updated in SQLite
	svc := NewSkillService(store.DB(), dir)
	meta, _ := svc.GetSkill("file-move-skill")
	if meta == nil {
		t.Fatal("skill not found")
	}
	if meta.FilePath != staleFile {
		t.Errorf("file_path = %q, want %q", meta.FilePath, staleFile)
	}
}

// TestCuratorArchivedRecovery verifies that archived skills can be recovered
// back to active stage (SKIL-06).
func TestCuratorArchivedRecovery(t *testing.T) {
	store, dir := newTestSQLiteStore(t)
	defer store.Close()

	makeSkill(t, store, dir, "recover-skill", "A skill to archive and recover")

	// Archive the skill
	svc := NewSkillService(store.DB(), dir)
	if err := svc.ArchiveSkill("recover-skill"); err != nil {
		t.Fatalf("ArchiveSkill: %v", err)
	}

	// Verify it's archived
	meta, _ := svc.GetSkill("recover-skill")
	if meta.Stage != SkillStageArchived {
		t.Fatalf("stage = %q, want archived before recovery", meta.Stage)
	}

	// Recover the skill
	curator := NewCurator(store.DB(), dir)
	if err := curator.RecoverSkill("recover-skill"); err != nil {
		t.Fatalf("RecoverSkill: %v", err)
	}

	// Verify it's back to active
	meta, err := svc.GetSkill("recover-skill")
	if err != nil {
		t.Fatalf("GetSkill after recovery: %v", err)
	}
	if meta.Stage != SkillStageActive {
		t.Errorf("stage = %q, want %q after recovery", meta.Stage, SkillStageActive)
	}

	// Verify file is in active/ directory
	activeFile := filepath.Join(skillDirForStage(dir, SkillStageActive), "recover-skill", "SKILL.md")
	if _, err := os.Stat(activeFile); err != nil {
		t.Errorf("SKILL.md not found in active/ after recovery: %v", err)
	}

	// Verify last_used_at was reset
	if meta.LastUsedAt == "" {
		t.Error("last_used_at should be reset after recovery")
	}
}

// TestCuratorEmptyTable verifies that the curator handles an empty skills table
// gracefully with no errors and zero transitions.
func TestCuratorEmptyTable(t *testing.T) {
	store, dir := newTestSQLiteStore(t)
	defer store.Close()

	curator := NewCurator(store.DB(), dir)
	count, err := curator.RunTransitions()
	if err != nil {
		t.Fatalf("RunTransitions on empty table: %v", err)
	}
	if count != 0 {
		t.Errorf("RunTransitions count = %d, want 0", count)
	}
}

// TestCuratorUsageTracking_View verifies that RecordSkillView increments
// view_count and updates last_viewed_at.
func TestCuratorUsageTracking_View(t *testing.T) {
	store, dir := newTestSQLiteStore(t)
	defer store.Close()

	makeSkill(t, store, dir, "view-skill", "A skill to view")

	curator := NewCurator(store.DB(), dir)
	if err := curator.RecordSkillView("view-skill"); err != nil {
		t.Fatalf("RecordSkillView: %v", err)
	}

	svc := NewSkillService(store.DB(), dir)
	meta, _ := svc.GetSkill("view-skill")
	if meta == nil {
		t.Fatal("skill not found")
	}
	if meta.ViewCount != 1 {
		t.Errorf("view_count = %d, want 1", meta.ViewCount)
	}
	if meta.LastViewedAt == "" {
		t.Error("last_viewed_at should be set after RecordSkillView")
	}
}

// TestCuratorUsageTracking_Use verifies that RecordSkillUse increments
// use_count and updates last_used_at.
func TestCuratorUsageTracking_Use(t *testing.T) {
	store, dir := newTestSQLiteStore(t)
	defer store.Close()

	makeSkill(t, store, dir, "use-skill", "A skill to use")

	curator := NewCurator(store.DB(), dir)
	if err := curator.RecordSkillUse("use-skill"); err != nil {
		t.Fatalf("RecordSkillUse: %v", err)
	}

	svc := NewSkillService(store.DB(), dir)
	meta, _ := svc.GetSkill("use-skill")
	if meta == nil {
		t.Fatal("skill not found")
	}
	if meta.UseCount != 1 {
		t.Errorf("use_count = %d, want 1", meta.UseCount)
	}
	if meta.LastUsedAt == "" {
		t.Error("last_used_at should be set after RecordSkillUse")
	}
}

// TestCuratorMultipleUsageResetsTransition verifies that a recent use
// resets the transition clock, preventing a skill from being transitioned.
func TestCuratorMultipleUsageResetsTransition(t *testing.T) {
	store, dir := newTestSQLiteStore(t)
	defer store.Close()

	makeSkill(t, store, dir, "recently-used", "A skill used recently but created long ago")

	// Set created_at to 15 days ago
	fifteenDaysAgo := time.Now().UTC().AddDate(0, 0, -15)
	store.DB().Exec(`UPDATE skills SET created_at = ? WHERE name = ?`,
		fifteenDaysAgo.Format(time.RFC3339), "recently-used")

	// Record a use (sets last_used_at to now)
	curator := NewCurator(store.DB(), dir)
	if err := curator.RecordSkillUse("recently-used"); err != nil {
		t.Fatalf("RecordSkillUse: %v", err)
	}

	// Run transitions -- skill should NOT be transitioned because it was recently used
	count, err := curator.RunTransitions()
	if err != nil {
		t.Fatalf("RunTransitions: %v", err)
	}
	if count != 0 {
		t.Errorf("RunTransitions count = %d, want 0 (recently used skill should not transition)", count)
	}

	svc := NewSkillService(store.DB(), dir)
	meta, _ := svc.GetSkill("recently-used")
	if meta == nil {
		t.Fatal("skill not found")
	}
	if meta.Stage != SkillStageActive {
		t.Errorf("stage = %q, want %q", meta.Stage, SkillStageActive)
	}
}

// TestCuratorReturnsTransitionCount verifies that RunTransitions returns
// the correct count of skills transitioned.
func TestCuratorReturnsTransitionCount(t *testing.T) {
	store, dir := newTestSQLiteStore(t)
	defer store.Close()

	// Create 3 active skills with old timestamps
	for i := 0; i < 3; i++ {
		name := fmt.Sprintf("old-skill-%d", i)
		makeSkill(t, store, dir, name, "Old skill content")
		fifteenDaysAgo := time.Now().UTC().AddDate(0, 0, -15)
		ageSkill(t, store, name, fifteenDaysAgo)
	}

	curator := NewCurator(store.DB(), dir)
	count, err := curator.RunTransitions()
	if err != nil {
		t.Fatalf("RunTransitions: %v", err)
	}
	if count != 3 {
		t.Errorf("RunTransitions count = %d, want 3", count)
	}
}
