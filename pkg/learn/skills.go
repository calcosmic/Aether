package learn

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Skill stage constants (D-08: 14 days per stage).
const (
	SkillStageActive   = "active"
	SkillStageStale    = "stale"
	SkillStageArchived = "archived"
)

// SkillMetadata tracks skill lifecycle state in SQLite skills table.
// Skills also exist as SKILL.md files on disk for progressive disclosure.
type SkillMetadata struct {
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	Stage              string  `json:"stage"`
	Pinned             bool    `json:"pinned"`
	ViewCount          int     `json:"view_count"`
	UseCount           int     `json:"use_count"`
	PatchCount         int     `json:"patch_count"`
	LastUsedAt         string  `json:"last_used_at,omitempty"`
	LastViewedAt       string  `json:"last_viewed_at,omitempty"`
	CreatedAt          string  `json:"created_at"`
	LastTransitionedAt string  `json:"last_transitioned_at,omitempty"`
	SourceRunID        string  `json:"source_run_id,omitempty"`
	Confidence         float64 `json:"confidence"`
	AutoCreated        bool    `json:"auto_created"`
	FilePath           string  `json:"file_path"`
}

// SkillIndexEntry is the lightweight representation for progressive disclosure (SKIL-02).
// Only includes index-level data -- full content loads on match.
type SkillIndexEntry struct {
	Name        string   `json:"name" yaml:"name"`
	Description string   `json:"description" yaml:"description"`
	Category    string   `json:"category" yaml:"category"`
	Roles       []string `json:"roles" yaml:"roles"`
	Type        string   `json:"type" yaml:"type"`
	FilePath    string   `json:"file_path"`
}

// SkillEvidenceFrontmatter extends skillFrontmatter with evidence fields for auto-created skills (AUTO-03).
type SkillEvidenceFrontmatter struct {
	Name              string   `yaml:"name"`
	Description       string   `yaml:"description"`
	Type              string   `yaml:"type,omitempty"`
	Category          string   `yaml:"category,omitempty"`
	Roles             []string `yaml:"roles,omitempty"`
	Detect            []string `yaml:"detect,omitempty"`
	SourceRunID       string   `yaml:"source_run_id,omitempty"`
	Confidence        float64  `yaml:"confidence"`
	DifficultyScore   float64  `yaml:"difficulty_score,omitempty"`
	DifficultyReasons []string `yaml:"difficulty_reasons,omitempty"`
	PrivacyScan       string   `yaml:"privacy_scan"`
	AutoCreated       bool     `yaml:"auto_created"`
	CreatedAt         string   `yaml:"created_at"`
}

// SkillDir returns the base directory for learned skills under the given base path.
func SkillDir(baseDir string) string {
	return filepath.Join(baseDir, ".aether", "hive", "skills")
}

// skillDirForStage returns the directory for skills in a given stage.
func skillDirForStage(baseDir, stage string) string {
	return filepath.Join(SkillDir(baseDir), stage)
}

// validateSkillName rejects names with path traversal or invalid characters.
func validateSkillName(name string) error {
	if name == "" {
		return fmt.Errorf("learn: skill name cannot be empty")
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("learn: skill name cannot contain path separators")
	}
	if strings.Contains(name, "..") {
		return fmt.Errorf("learn: skill name cannot contain '..'")
	}
	if strings.Contains(name, string(rune(0))) {
		return fmt.Errorf("learn: skill name cannot contain null bytes")
	}
	return nil
}

// SkillService manages skill lifecycle operations backed by SQLite metadata
// and SKILL.md files on disk.
type SkillService struct {
	db      *sql.DB
	baseDir string
}

// NewSkillService creates a skill service using the same SQLite database
// as the SQLiteColonyStore.
func NewSkillService(db *sql.DB, baseDir string) *SkillService {
	return &SkillService{db: db, baseDir: baseDir}
}

// CreateSkill creates a new skill with SKILL.md file and SQLite metadata (SKIL-01, SKIL-03).
func (svc *SkillService) CreateSkill(meta SkillMetadata, content string) error {
	if err := validateSkillName(meta.Name); err != nil {
		return err
	}
	if meta.ID == "" {
		meta.ID = fmt.Sprintf("skill_%s_%d", time.Now().Format("20060102"), time.Now().UnixNano())
	}
	if meta.CreatedAt == "" {
		meta.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if meta.Stage == "" {
		meta.Stage = SkillStageActive
	}

	// Write SKILL.md file
	skillDir := filepath.Join(skillDirForStage(svc.baseDir, meta.Stage), meta.Name)
	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("learn: create skill dir: %w", err)
	}

	// Build SKILL.md with YAML frontmatter
	fm := SkillEvidenceFrontmatter{
		Name:        meta.Name,
		Description: content[:minInt(200, len(content))],
		Type:        "learned",
		Category:    "colony",
		SourceRunID: meta.SourceRunID,
		Confidence:  meta.Confidence,
		PrivacyScan: "passed",
		AutoCreated: meta.AutoCreated,
		CreatedAt:   meta.CreatedAt,
	}
	fmBytes, err := yaml.Marshal(fm)
	if err != nil {
		return fmt.Errorf("learn: marshal skill frontmatter: %w", err)
	}

	fullContent := fmt.Sprintf("---\n%s---\n\n%s", string(fmBytes), content)
	if err := os.WriteFile(skillPath, []byte(fullContent), 0644); err != nil {
		return fmt.Errorf("learn: write skill file: %w", err)
	}
	meta.FilePath = skillPath

	// Insert into SQLite
	_, err = svc.db.Exec(`INSERT INTO skills (id, name, stage, pinned, view_count, use_count, patch_count,
		last_used_at, last_viewed_at, created_at, last_transitioned_at, source_run_id, confidence, auto_created, file_path)
		VALUES (?, ?, ?, ?, 0, 0, 0, NULL, NULL, ?, '', ?, ?, ?, ?)`,
		meta.ID, meta.Name, meta.Stage, boolToInt(meta.Pinned),
		meta.CreatedAt, meta.SourceRunID, meta.Confidence, boolToInt(meta.AutoCreated), meta.FilePath)
	if err != nil {
		// Clean up file if DB insert fails
		os.Remove(skillPath)
		os.Remove(skillDir)
		return fmt.Errorf("learn: insert skill metadata: %w", err)
	}
	return nil
}

// GetSkill retrieves skill metadata by name.
func (svc *SkillService) GetSkill(name string) (*SkillMetadata, error) {
	row := svc.db.QueryRow(`SELECT id, name, stage, pinned, view_count, use_count, patch_count,
		last_used_at, last_viewed_at, created_at, last_transitioned_at, source_run_id, confidence, auto_created, file_path
		FROM skills WHERE name = ?`, name)
	return scanSkillMetadata(row)
}

// PatchSkill updates an existing skill's SKILL.md content and increments patch_count (SKIL-03).
func (svc *SkillService) PatchSkill(name string, content string) error {
	meta, err := svc.GetSkill(name)
	if err != nil {
		return err
	}
	if meta == nil {
		return fmt.Errorf("learn: skill %q not found", name)
	}
	if meta.Pinned {
		return fmt.Errorf("learn: pinned skill %q cannot be patched", name)
	}

	// Rewrite SKILL.md
	if err := os.WriteFile(meta.FilePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("learn: patch skill file: %w", err)
	}

	_, err = svc.db.Exec(`UPDATE skills SET patch_count = patch_count + 1 WHERE name = ?`, name)
	if err != nil {
		return fmt.Errorf("learn: update patch count: %w", err)
	}
	return nil
}

// ArchiveSkill moves a skill from its current stage to archived (SKIL-03, SKIL-06).
func (svc *SkillService) ArchiveSkill(name string) error {
	meta, err := svc.GetSkill(name)
	if err != nil {
		return err
	}
	if meta == nil {
		return fmt.Errorf("learn: skill %q not found", name)
	}
	if meta.Pinned {
		return fmt.Errorf("learn: pinned skill %q cannot be archived", name)
	}

	// Move SKILL.md file
	newDir := filepath.Join(skillDirForStage(svc.baseDir, SkillStageArchived), meta.Name)
	newPath := filepath.Join(newDir, "SKILL.md")
	if err := os.MkdirAll(newDir, 0755); err != nil {
		return fmt.Errorf("learn: create archive dir: %w", err)
	}
	if err := os.Rename(meta.FilePath, newPath); err != nil {
		return fmt.Errorf("learn: move skill to archive: %w", err)
	}

	// Update SQLite metadata
	_, err = svc.db.Exec(`UPDATE skills SET stage = ?, file_path = ?, last_transitioned_at = ? WHERE name = ?`,
		SkillStageArchived, newPath, time.Now().UTC().Format(time.RFC3339), name)
	if err != nil {
		return fmt.Errorf("learn: update skill stage: %w", err)
	}
	return nil
}

// PinSkill sets pinned=true for a skill, making it immune to auto-transitions (SKIL-05).
func (svc *SkillService) PinSkill(name string) error {
	meta, err := svc.GetSkill(name)
	if err != nil {
		return err
	}
	if meta == nil {
		return fmt.Errorf("learn: skill %q not found", name)
	}
	_, err = svc.db.Exec(`UPDATE skills SET pinned = 1 WHERE name = ?`, name)
	if err != nil {
		return fmt.Errorf("learn: pin skill: %w", err)
	}
	return nil
}

// UnpinSkill sets pinned=false for a skill.
func (svc *SkillService) UnpinSkill(name string) error {
	_, err := svc.db.Exec(`UPDATE skills SET pinned = 0 WHERE name = ?`, name)
	if err != nil {
		return fmt.Errorf("learn: unpin skill: %w", err)
	}
	return nil
}

// ListSkills returns skills matching the optional stage filter.
func (svc *SkillService) ListSkills(stage string) ([]SkillMetadata, error) {
	query := `SELECT id, name, stage, pinned, view_count, use_count, patch_count,
		last_used_at, last_viewed_at, created_at, last_transitioned_at, source_run_id, confidence, auto_created, file_path
		FROM skills`
	var args []interface{}
	if stage != "" {
		query += ` WHERE stage = ?`
		args = append(args, stage)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := svc.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("learn: list skills: %w", err)
	}
	defer rows.Close()

	var result []SkillMetadata
	for rows.Next() {
		meta, err := scanSkillMetadataFromRows(rows)
		if err != nil {
			return nil, fmt.Errorf("learn: scan skill: %w", err)
		}
		result = append(result, *meta)
	}
	if result == nil {
		result = []SkillMetadata{}
	}
	return result, nil
}

// BuildSkillIndex creates lightweight index entries from active skill files (SKIL-02).
// Progressive disclosure: only includes name, description, roles, type, and file path.
// Full content is loaded later when skill-inject matches a skill.
func (svc *SkillService) BuildSkillIndex() ([]SkillIndexEntry, error) {
	skills, err := svc.ListSkills(SkillStageActive)
	if err != nil {
		return nil, err
	}

	var entries []SkillIndexEntry
	for _, meta := range skills {
		// Read SKILL.md and parse frontmatter for description and roles
		data, err := os.ReadFile(meta.FilePath)
		if err != nil {
			continue // Skip unreadable skills
		}
		content := string(data)

		// Extract YAML frontmatter
		desc := ""
		var roles []string
		if idx := strings.Index(content, "---\n"); idx == 0 {
			end := strings.Index(content[4:], "\n---\n")
			if end >= 0 {
				fmContent := content[4 : end+4]
				var fm map[string]interface{}
				if yaml.Unmarshal([]byte(fmContent), &fm) == nil {
					if d, ok := fm["description"].(string); ok {
						desc = d
					}
					if r, ok := fm["roles"].([]interface{}); ok {
						for _, v := range r {
							if s, ok := v.(string); ok {
								roles = append(roles, s)
							}
						}
					}
				}
			}
		}

		entries = append(entries, SkillIndexEntry{
			Name:        meta.Name,
			Description: desc,
			Category:    "colony",
			Roles:       roles,
			Type:        "learned",
			FilePath:    meta.FilePath,
		})
	}
	if entries == nil {
		entries = []SkillIndexEntry{}
	}
	return entries, nil
}

// scanSkillMetadata scans a single SkillMetadata from a sql.Row.
func scanSkillMetadata(row *sql.Row) (*SkillMetadata, error) {
	var m SkillMetadata
	var pinned, autoCreated int
	var lastUsed, lastViewed, lastTransitioned sql.NullString
	err := row.Scan(&m.ID, &m.Name, &m.Stage, &pinned, &m.ViewCount, &m.UseCount, &m.PatchCount,
		&lastUsed, &lastViewed, &m.CreatedAt, &lastTransitioned, &m.SourceRunID, &m.Confidence, &autoCreated, &m.FilePath)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	m.Pinned = pinned == 1
	m.AutoCreated = autoCreated == 1
	if lastUsed.Valid {
		m.LastUsedAt = lastUsed.String
	}
	if lastViewed.Valid {
		m.LastViewedAt = lastViewed.String
	}
	if lastTransitioned.Valid {
		m.LastTransitionedAt = lastTransitioned.String
	}
	return &m, nil
}

// scanSkillMetadataFromRows scans a single SkillMetadata from sql.Rows.
func scanSkillMetadataFromRows(rows *sql.Rows) (*SkillMetadata, error) {
	var m SkillMetadata
	var pinned, autoCreated int
	var lastUsed, lastViewed, lastTransitioned sql.NullString
	err := rows.Scan(&m.ID, &m.Name, &m.Stage, &pinned, &m.ViewCount, &m.UseCount, &m.PatchCount,
		&lastUsed, &lastViewed, &m.CreatedAt, &lastTransitioned, &m.SourceRunID, &m.Confidence, &autoCreated, &m.FilePath)
	if err != nil {
		return nil, err
	}
	m.Pinned = pinned == 1
	m.AutoCreated = autoCreated == 1
	if lastUsed.Valid {
		m.LastUsedAt = lastUsed.String
	}
	if lastViewed.Valid {
		m.LastViewedAt = lastViewed.String
	}
	if lastTransitioned.Valid {
		m.LastTransitionedAt = lastTransitioned.String
	}
	return &m, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
