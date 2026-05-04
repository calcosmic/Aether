package learn

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CuratorStageDays defines the number of days before a skill transitions
// to the next lifecycle stage (D-08: 14 days per stage).
const CuratorStageDays = 14

// Curator manages skill lifecycle transitions and usage tracking (SKIL-04/05/06).
// It runs periodically (on continue/seal) to transition unused skills through
// active -> stale -> archived stages, and tracks usage counts for lifecycle decisions.
type Curator struct {
	db      *sql.DB
	baseDir string
}

// NewCurator creates a Curator backed by the same SQLite database as SkillService.
func NewCurator(db *sql.DB, baseDir string) *Curator {
	return &Curator{db: db, baseDir: baseDir}
}

// RunTransitions performs lifecycle transitions for all eligible skills.
// Returns the total number of skills transitioned.
// Pinned skills are always skipped (D-09, SKIL-05).
// Transitions are non-blocking -- errors on individual skills are logged but do not stop the run.
func (c *Curator) RunTransitions() (int, error) {
	now := time.Now().UTC()
	staleThreshold := now.AddDate(0, 0, -CuratorStageDays)
	archivedThreshold := now.AddDate(0, 0, -CuratorStageDays*2)

	total := 0

	// Active -> Stale (14 days unused)
	activeCount, err := c.transitionStage(SkillStageActive, SkillStageStale, staleThreshold, now)
	if err != nil {
		return total, fmt.Errorf("learn: curator active->stale: %w", err)
	}
	total += activeCount

	// Stale -> Archived (another 14 days unused = 28 total)
	staleCount, err := c.transitionStage(SkillStageStale, SkillStageArchived, archivedThreshold, now)
	if err != nil {
		return total, fmt.Errorf("learn: curator stale->archived: %w", err)
	}
	total += staleCount

	return total, nil
}

// transitionSkill is a simple struct to hold skill data collected from a query.
type transitionSkill struct {
	id, name, filePath string
}

// transitionStage moves skills from sourceStage to targetStage if they have not been
// used or viewed since the threshold time. Pinned skills are excluded (D-09).
func (c *Curator) transitionStage(sourceStage, targetStage string, threshold, now time.Time) (int, error) {
	// Find eligible skills: not pinned, not recently used/viewed.
	// Must collect all rows before processing to avoid deadlock with MaxOpenConns(1).
	rows, err := c.db.Query(`
		SELECT id, name, file_path FROM skills
		WHERE stage = ? AND pinned = 0
		  AND (
		    (last_used_at IS NULL OR last_used_at = '' OR last_used_at < ?)
		    AND (last_viewed_at IS NULL OR last_viewed_at = '' OR last_viewed_at < ?)
		    AND created_at < ?
		  )`,
		sourceStage,
		threshold.Format(time.RFC3339),
		threshold.Format(time.RFC3339),
		threshold.Format(time.RFC3339),
	)
	if err != nil {
		return 0, fmt.Errorf("learn: query %s skills: %w", sourceStage, err)
	}

	// Collect all eligible skills, then close rows to release the connection
	var eligible []transitionSkill
	for rows.Next() {
		var s transitionSkill
		if err := rows.Scan(&s.id, &s.name, &s.filePath); err != nil {
			continue
		}
		eligible = append(eligible, s)
	}
	rows.Close()

	count := 0
	for _, s := range eligible {
		// Move SKILL.md file
		newDir := filepath.Join(skillDirForStage(c.baseDir, targetStage), s.name)
		newPath := filepath.Join(newDir, "SKILL.md")

		if s.filePath != "" {
			if _, err := os.Stat(s.filePath); err == nil {
				if err := os.MkdirAll(newDir, 0755); err != nil {
					continue
				}
				if err := os.Rename(s.filePath, newPath); err != nil {
					continue
				}
			}
		}

		// Update SQLite metadata
		_, err := c.db.Exec(`
			UPDATE skills SET stage = ?, file_path = ?, last_transitioned_at = ?
			WHERE id = ?`,
			targetStage, newPath, now.Format(time.RFC3339), s.id,
		)
		if err != nil {
			continue
		}
		count++
	}
	return count, nil
}

// RecoverSkill moves an archived skill back to active (SKIL-06).
// Archived skills are always recoverable and never auto-deleted.
func (c *Curator) RecoverSkill(name string) error {
	meta, err := c.getSkillByName(name)
	if err != nil {
		return fmt.Errorf("learn: recover skill: %w", err)
	}
	if meta == nil {
		return fmt.Errorf("learn: skill %q not found", name)
	}
	if meta.Stage != SkillStageArchived {
		return fmt.Errorf("learn: skill %q is not archived (stage=%s)", name, meta.Stage)
	}

	// Move file back to active/
	newDir := filepath.Join(skillDirForStage(c.baseDir, SkillStageActive), name)
	newPath := filepath.Join(newDir, "SKILL.md")
	if meta.FilePath != "" {
		if _, err := os.Stat(meta.FilePath); err == nil {
			if err := os.MkdirAll(newDir, 0755); err != nil {
				return fmt.Errorf("learn: create active dir: %w", err)
			}
			if err := os.Rename(meta.FilePath, newPath); err != nil {
				return fmt.Errorf("learn: recover skill file: %w", err)
			}
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = c.db.Exec(`
		UPDATE skills SET stage = ?, file_path = ?, last_transitioned_at = ?,
		    last_used_at = ?, last_viewed_at = ?
		WHERE name = ?`,
		SkillStageActive, newPath, now, now, now, name,
	)
	if err != nil {
		return fmt.Errorf("learn: update recovered skill: %w", err)
	}
	return nil
}

// RecordSkillView increments view_count and updates last_viewed_at (SKIL-04).
func (c *Curator) RecordSkillView(name string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := c.db.Exec(`
		UPDATE skills SET view_count = view_count + 1, last_viewed_at = ?
		WHERE name = ?`, now, name)
	if err != nil {
		return fmt.Errorf("learn: record skill view: %w", err)
	}
	return nil
}

// RecordSkillUse increments use_count and updates last_used_at (SKIL-04).
func (c *Curator) RecordSkillUse(name string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := c.db.Exec(`
		UPDATE skills SET use_count = use_count + 1, last_used_at = ?
		WHERE name = ?`, now, name)
	if err != nil {
		return fmt.Errorf("learn: record skill use: %w", err)
	}
	return nil
}

// getSkillByName retrieves skill metadata by name (internal helper).
func (c *Curator) getSkillByName(name string) (*SkillMetadata, error) {
	svc := NewSkillService(c.db, c.baseDir)
	return svc.GetSkill(name)
}
