package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// Skill types.

type skillFrontmatter struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Detect      []string `json:"detect,omitempty"`
	Roles       []string `json:"roles,omitempty"`
}

type skillIndexEntry struct {
	Name          string   `json:"name"`
	Category      string   `json:"category"`
	Path          string   `json:"path"`
	IsUserCreated bool     `json:"is_user_created"`
	Detect        []string `json:"detect,omitempty"`
	Roles         []string `json:"roles,omitempty"`
}

type skillIndexData struct {
	Entries   []skillIndexEntry `json:"entries"`
	UpdatedAt string            `json:"updated_at"`
}

type skillManifestEntry struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Checksum string `json:"checksum"`
}

type skillManifestData struct {
	Skills    []skillManifestEntry `json:"skills"`
	UpdatedAt string               `json:"updated_at"`
}

// --- skill-parse-frontmatter ---

var skillParseFrontmatterCmd = &cobra.Command{
	Use:   "skill-parse-frontmatter",
	Short: "Parse SKILL.md frontmatter and return as JSON",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		file := mustGetString(cmd, "file")
		if file == "" {
			return nil
		}

		raw, err := os.ReadFile(file)
		if err != nil {
			outputError(1, fmt.Sprintf("failed to read %s: %v", file, err), nil)
			return nil
		}

		fm := parseSkillFrontmatter(string(raw))
		if fm == nil {
			outputError(1, "no frontmatter found in file", nil)
			return nil
		}

		outputOK(fm)
		return nil
	},
}

// --- skill-index ---

var skillIndexCmd = &cobra.Command{
	Use:   "skill-index",
	Short: "Build skills index from installed skills",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		entries := []skillIndexEntry{}

		// Index shipped skills from local .aether/skills/
		if localSkills := findSkillDirs(".aether/skills"); len(localSkills) > 0 {
			for _, d := range localSkills {
				if e := indexSkillDir(d, false); e != nil {
					entries = append(entries, *e)
				}
			}
		}

		// Index user skills from hub ~/.aether/skills/
		hub := resolveHubPath()
		if hubSkills := findSkillDirs(filepath.Join(hub, "skills")); len(hubSkills) > 0 {
			for _, d := range hubSkills {
				if e := indexSkillDir(d, true); e != nil {
					entries = append(entries, *e)
				}
			}
		}

		data := skillIndexData{
			Entries:   entries,
			UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		}

		// Save to hub
		indexPath := filepath.Join(hub, "skills", "index.json")
		os.MkdirAll(filepath.Dir(indexPath), 0755)
		encoded, _ := json.MarshalIndent(data, "", "  ")
		os.WriteFile(indexPath, append(encoded, '\n'), 0644)

		outputOK(map[string]interface{}{"indexed": len(entries), "path": indexPath})
		return nil
	},
}

// --- skill-index-read ---

var skillIndexReadCmd = &cobra.Command{
	Use:   "skill-index-read",
	Short: "Read cached skills index",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		hub := resolveHubPath()
		indexPath := filepath.Join(hub, "skills", "index.json")

		raw, err := os.ReadFile(indexPath)
		if err != nil {
			outputOK(map[string]interface{}{"entries": []skillIndexEntry{}, "total": 0})
			return nil
		}

		var data skillIndexData
		if err := json.Unmarshal(raw, &data); err != nil {
			outputError(1, fmt.Sprintf("invalid index: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{"entries": data.Entries, "total": len(data.Entries), "updated_at": data.UpdatedAt})
		return nil
	},
}

// --- skill-detect ---

var skillDetectCmd = &cobra.Command{
	Use:   "skill-detect",
	Short: "Detect domain skills matching codebase file patterns",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		hub := resolveHubPath()
		indexPath := filepath.Join(hub, "skills", "index.json")

		var data skillIndexData
		if raw, err := os.ReadFile(indexPath); err == nil {
			json.Unmarshal(raw, &data)
		}

		var matched []skillIndexEntry
		for _, e := range data.Entries {
			for _, pattern := range e.Detect {
				if matches, _ := filepath.Glob(pattern); len(matches) > 0 {
					matched = append(matched, e)
					break
				}
			}
		}

		outputOK(map[string]interface{}{"matched": matched, "total": len(matched)})
		return nil
	},
}

// --- skill-match ---

var skillMatchCmd = &cobra.Command{
	Use:   "skill-match",
	Short: "Match skills to worker role and task",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		role := mustGetString(cmd, "role")
		if role == "" {
			return nil
		}
		task, _ := cmd.Flags().GetString("task")

		hub := resolveHubPath()
		indexPath := filepath.Join(hub, "skills", "index.json")

		var data skillIndexData
		if raw, err := os.ReadFile(indexPath); err == nil {
			json.Unmarshal(raw, &data)
		}

		// Score each skill
		type scored struct {
			entry skillIndexEntry
			score int
		}
		var results []scored
		for _, e := range data.Entries {
			score := 0
			for _, r := range e.Roles {
				if r == role {
					score += 2
				}
			}
			if task != "" && strings.Contains(strings.ToLower(e.Category), strings.ToLower(task)) {
				score += 1
			}
			if score > 0 {
				results = append(results, scored{entry: e, score: score})
			}
		}

		// Sort by score desc
		for i := 0; i < len(results)-1; i++ {
			for j := i + 1; j < len(results); j++ {
				if results[j].score > results[i].score {
					results[i], results[j] = results[j], results[i]
				}
			}
		}

		// Top 3
		top := results
		if len(top) > 3 {
			top = top[:3]
		}

		var names []string
		for _, s := range top {
			names = append(names, s.entry.Name)
		}

		outputOK(map[string]interface{}{"matched": names, "count": len(names), "role": role})
		return nil
	},
}

// --- skill-inject ---

var skillInjectCmd = &cobra.Command{
	Use:   "skill-inject",
	Short: "Load matched skills into prompt section text",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		role := mustGetString(cmd, "role")
		if role == "" {
			return nil
		}

		hub := resolveHubPath()
		indexPath := filepath.Join(hub, "skills", "index.json")

		var data skillIndexData
		if raw, err := os.ReadFile(indexPath); err == nil {
			json.Unmarshal(raw, &data)
		}

		var sections []string
		for _, e := range data.Entries {
			roleMatch := false
			for _, r := range e.Roles {
				if r == role {
					roleMatch = true
					break
				}
			}
			if !roleMatch {
				continue
			}

			// Read the skill file content
			skillPath := filepath.Join(hub, "skills", e.Category, "SKILL.md")
			if _, err := os.Stat(skillPath); err != nil {
				skillPath = e.Path
			}
			if content, err := os.ReadFile(skillPath); err == nil {
				sections = append(sections, fmt.Sprintf("### Skill: %s\n\n%s", e.Name, string(content)))
			}
		}

		if len(sections) == 0 {
			outputOK(map[string]interface{}{"section": "", "skill_count": 0})
			return nil
		}

		outputOK(map[string]interface{}{
			"section":     strings.Join(sections, "\n\n---\n\n"),
			"skill_count": len(sections),
		})
		return nil
	},
}

// --- skill-list ---

var skillListCmd = &cobra.Command{
	Use:   "skill-list",
	Short: "List all installed skills",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		entries := []skillIndexEntry{}

		// Local shipped skills
		if localSkills := findSkillDirs(".aether/skills"); len(localSkills) > 0 {
			for _, d := range localSkills {
				if e := indexSkillDir(d, false); e != nil {
					entries = append(entries, *e)
				}
			}
		}

		// Hub user skills
		hub := resolveHubPath()
		if hubSkills := findSkillDirs(filepath.Join(hub, "skills")); len(hubSkills) > 0 {
			for _, d := range hubSkills {
				if e := indexSkillDir(d, true); e != nil {
					entries = append(entries, *e)
				}
			}
		}

		outputOK(map[string]interface{}{"skills": entries, "total": len(entries)})
		return nil
	},
}

// --- skill-manifest-read ---

var skillManifestReadCmd = &cobra.Command{
	Use:   "skill-manifest-read",
	Short: "Read the skills manifest",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Try hub manifest first
		hub := resolveHubPath()
		manifestPath := filepath.Join(hub, "skills", "manifest.json")

		raw, err := os.ReadFile(manifestPath)
		if err != nil {
			// Try local
			manifestPath = ".aether/skills/manifest.json"
			raw, err = os.ReadFile(manifestPath)
			if err != nil {
				outputOK(map[string]interface{}{"skills": []skillManifestEntry{}, "total": 0})
				return nil
			}
		}

		var data skillManifestData
		if err := json.Unmarshal(raw, &data); err != nil {
			outputError(1, fmt.Sprintf("invalid manifest: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{"skills": data.Skills, "total": len(data.Skills), "updated_at": data.UpdatedAt})
		return nil
	},
}

// --- skill-cache-rebuild ---

var skillCacheRebuildCmd = &cobra.Command{
	Use:   "skill-cache-rebuild",
	Short: "Force rebuild of skills index cache",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		hub := resolveHubPath()
		indexPath := filepath.Join(hub, "skills", "index.json")

		entries := []skillIndexEntry{}

		if localSkills := findSkillDirs(".aether/skills"); len(localSkills) > 0 {
			for _, d := range localSkills {
				if e := indexSkillDir(d, false); e != nil {
					entries = append(entries, *e)
				}
			}
		}
		if hubSkills := findSkillDirs(filepath.Join(hub, "skills")); len(hubSkills) > 0 {
			for _, d := range hubSkills {
				if e := indexSkillDir(d, true); e != nil {
					entries = append(entries, *e)
				}
			}
		}

		data := skillIndexData{
			Entries:   entries,
			UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		}

		os.MkdirAll(filepath.Dir(indexPath), 0755)
		encoded, _ := json.MarshalIndent(data, "", "  ")
		if err := os.WriteFile(indexPath, append(encoded, '\n'), 0644); err != nil {
			outputError(2, fmt.Sprintf("failed to write: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{"rebuilt": true, "total": len(entries), "path": indexPath})
		return nil
	},
}

// --- skill-diff ---

var skillDiffCmd = &cobra.Command{
	Use:   "skill-diff",
	Short: "Compare user skill with shipped version",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := mustGetString(cmd, "skill")
		if name == "" {
			return nil
		}

		hub := resolveHubPath()
		userPath := filepath.Join(hub, "skills", "domain", name, "SKILL.md")
		shippedPath := filepath.Join(".aether", "skills", "domain", name, "SKILL.md")

		userContent, userErr := os.ReadFile(userPath)
		shippedContent, shippedErr := os.ReadFile(shippedPath)

		if userErr != nil && shippedErr != nil {
			outputError(1, fmt.Sprintf("skill %q not found in user or shipped locations", name), nil)
			return nil
		}

		result := map[string]interface{}{
			"skill":          name,
			"user_exists":    userErr == nil,
			"shipped_exists": shippedErr == nil,
			"identical":      false,
		}

		if userErr == nil && shippedErr == nil {
			result["identical"] = string(userContent) == string(shippedContent)
			result["user_size"] = len(userContent)
			result["shipped_size"] = len(shippedContent)
		}

		outputOK(result)
		return nil
	},
}

// --- skill-is-user-created ---

var skillIsUserCreatedCmd = &cobra.Command{
	Use:   "skill-is-user-created",
	Short: "Check if a skill was user-created or shipped",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := mustGetString(cmd, "skill")
		if name == "" {
			return nil
		}

		hub := resolveHubPath()
		userPath := filepath.Join(hub, "skills", "domain", name, "SKILL.md")
		shippedPath := filepath.Join(".aether", "skills", "domain", name, "SKILL.md")

		_, userExists := os.Stat(userPath)
		_, shippedExists := os.Stat(shippedPath)

		// User-created = exists in hub but not in shipped
		isUserCreated := userExists == nil && shippedExists != nil

		outputOK(map[string]interface{}{
			"skill":           name,
			"is_user_created": isUserCreated,
			"in_hub":          userExists == nil,
			"in_shipped":      shippedExists == nil,
		})
		return nil
	},
}

// Helper functions.

func parseSkillFrontmatter(content string) *skillFrontmatter {
	lines := strings.Split(content, "\n")
	inFrontmatter := false
	var fmLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if inFrontmatter {
				break
			}
			inFrontmatter = true
			continue
		}
		if inFrontmatter {
			fmLines = append(fmLines, line)
		}
	}

	if len(fmLines) == 0 {
		return nil
	}

	var fm skillFrontmatter
	for _, line := range fmLines {
		if strings.HasPrefix(line, "name:") {
			fm.Name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
		} else if strings.HasPrefix(line, "description:") {
			fm.Description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
		} else if strings.HasPrefix(line, "category:") {
			fm.Category = strings.TrimSpace(strings.TrimPrefix(line, "category:"))
		} else if strings.HasPrefix(line, "detect:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "detect:"))
			if val != "" {
				fm.Detect = strings.Split(val, ",")
				for i := range fm.Detect {
					fm.Detect[i] = strings.TrimSpace(fm.Detect[i])
				}
			}
		} else if strings.HasPrefix(line, "roles:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "roles:"))
			if val != "" {
				fm.Roles = strings.Split(val, ",")
				for i := range fm.Roles {
					fm.Roles[i] = strings.TrimSpace(fm.Roles[i])
				}
			}
		}
	}

	return &fm
}

func findSkillDirs(baseDir string) []string {
	var dirs []string
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return dirs
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dirPath := filepath.Join(baseDir, e.Name())
		// If this directory contains SKILL.md, it's a skill directory
		if _, err := os.Stat(filepath.Join(dirPath, "SKILL.md")); err == nil {
			dirs = append(dirs, dirPath)
			continue
		}
		// Otherwise recurse into it (e.g., colony/ and domain/ category folders)
		dirs = append(dirs, findSkillDirs(dirPath)...)
	}
	return dirs
}

func indexSkillDir(dir string, isUserCreated bool) *skillIndexEntry {
	skillPath := filepath.Join(dir, "SKILL.md")
	raw, err := os.ReadFile(skillPath)
	if err != nil {
		return nil
	}

	fm := parseSkillFrontmatter(string(raw))
	if fm == nil || fm.Name == "" {
		return nil
	}

	return &skillIndexEntry{
		Name:          fm.Name,
		Category:      fm.Category,
		Path:          skillPath,
		IsUserCreated: isUserCreated,
		Detect:        fm.Detect,
		Roles:         fm.Roles,
	}
}

func init() {
	skillParseFrontmatterCmd.Flags().String("file", "", "Path to SKILL.md (required)")
	skillMatchCmd.Flags().String("role", "", "Worker role (required)")
	skillMatchCmd.Flags().String("task", "", "Task description")
	skillInjectCmd.Flags().String("role", "", "Worker role (required)")
	skillDiffCmd.Flags().String("skill", "", "Skill name (required)")
	skillIsUserCreatedCmd.Flags().String("skill", "", "Skill name (required)")

	rootCmd.AddCommand(skillParseFrontmatterCmd)
	rootCmd.AddCommand(skillIndexCmd)
	rootCmd.AddCommand(skillIndexReadCmd)
	rootCmd.AddCommand(skillDetectCmd)
	rootCmd.AddCommand(skillMatchCmd)
	rootCmd.AddCommand(skillInjectCmd)
	rootCmd.AddCommand(skillListCmd)
	rootCmd.AddCommand(skillManifestReadCmd)
	rootCmd.AddCommand(skillCacheRebuildCmd)
	rootCmd.AddCommand(skillDiffCmd)
	rootCmd.AddCommand(skillIsUserCreatedCmd)
}
