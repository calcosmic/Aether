package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type skillFrontmatter struct {
	Name           string   `json:"name" yaml:"name"`
	Description    string   `json:"description" yaml:"description"`
	Type           string   `json:"type,omitempty" yaml:"type"`
	Category       string   `json:"category,omitempty" yaml:"category"`
	Domains        []string `json:"domains,omitempty" yaml:"domains"`
	AgentRoles     []string `json:"agent_roles,omitempty" yaml:"agent_roles"`
	Roles          []string `json:"roles,omitempty" yaml:"roles"`
	DetectFiles    []string `json:"detect_files,omitempty" yaml:"detect_files"`
	DetectPackages []string `json:"detect_packages,omitempty" yaml:"detect_packages"`
	Detect         []string `json:"detect,omitempty" yaml:"detect"`
	Priority       string   `json:"priority,omitempty" yaml:"priority"`
	Version        string   `json:"version,omitempty" yaml:"version"`
}

type skillIndexEntry struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Type           string   `json:"type"`
	Category       string   `json:"category"`
	Domains        []string `json:"domains,omitempty"`
	AgentRoles     []string `json:"agent_roles,omitempty"`
	Roles          []string `json:"roles,omitempty"`
	DetectFiles    []string `json:"detect_files,omitempty"`
	DetectPackages []string `json:"detect_packages,omitempty"`
	Detect         []string `json:"detect,omitempty"`
	Priority       string   `json:"priority,omitempty"`
	Version        string   `json:"version,omitempty"`
	Path           string   `json:"path"`
	IsUserCreated  bool     `json:"is_user_created"`
	Source         string   `json:"source,omitempty"`
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

type skillScanRoot struct {
	Path          string
	Source        string
	IsUserCreated bool
}

type skillMatchReason struct {
	Code     string   `json:"code"`
	Score    int      `json:"score"`
	Evidence []string `json:"evidence,omitempty"`
}

type skillResolvedEntry struct {
	skillIndexEntry
	Score   int                `json:"score"`
	Reasons []skillMatchReason `json:"reasons,omitempty"`
}

type scoredSkill struct {
	entry skillResolvedEntry
	score int
}

type skillMatchResult struct {
	Role         string               `json:"role"`
	Task         string               `json:"task,omitempty"`
	Root         string               `json:"root,omitempty"`
	ColonySkills []skillResolvedEntry `json:"colony_skills"`
	DomainSkills []skillResolvedEntry `json:"domain_skills"`
	Matched      []string             `json:"matched"`
	Count        int                  `json:"count"`
}

type skillInjectResult struct {
	Role         string               `json:"role"`
	Task         string               `json:"task,omitempty"`
	Root         string               `json:"root,omitempty"`
	Section      string               `json:"section"`
	SkillSection string               `json:"skill_section"`
	SkillCount   int                  `json:"skill_count"`
	ColonyCount  int                  `json:"colony_count"`
	DomainCount  int                  `json:"domain_count"`
	ColonySkills []skillResolvedEntry `json:"colony_skills"`
	DomainSkills []skillResolvedEntry `json:"domain_skills"`
	Matched      []string             `json:"matched"`
	Count        int                  `json:"count"`
}

type workspaceFileSnapshot struct {
	RelPaths       []string
	BaseNames      []string
	PatternMatches map[string][]string
}

var workspaceFileSnapshotCache = struct {
	mu        sync.Mutex
	snapshots map[string]*workspaceFileSnapshot
}{
	snapshots: map[string]*workspaceFileSnapshot{},
}

var skillScanSkipDirs = map[string]struct{}{
	".git":          {},
	".aether":       {},
	".claude":       {},
	".codex":        {},
	".opencode":     {},
	".idea":         {},
	".vscode":       {},
	".cache":        {},
	".next":         {},
	".nuxt":         {},
	".svelte-kit":   {},
	".venv":         {},
	".tox":          {},
	".pytest_cache": {},
	".mypy_cache":   {},
	"node_modules":  {},
	"vendor":        {},
	"dist":          {},
	"build":         {},
	"coverage":      {},
	"tmp":           {},
	"temp":          {},
	"venv":          {},
}

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

var skillIndexCmd = &cobra.Command{
	Use:   "skill-index",
	Short: "Build skills index from installed skills",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		hub := resolveHubPath()
		entries := buildFullIndex(hub)

		data := skillIndexData{
			Entries:   entries,
			UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		}

		indexPath := filepath.Join(hub, "skills", "index.json")
		if err := os.MkdirAll(filepath.Dir(indexPath), 0755); err != nil {
			outputError(2, fmt.Sprintf("failed to create index directory: %v", err), nil)
			return nil
		}
		encoded, _ := json.MarshalIndent(data, "", "  ")
		if err := os.WriteFile(indexPath, append(encoded, '\n'), 0644); err != nil {
			outputError(2, fmt.Sprintf("failed to write index: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{"indexed": len(entries), "path": indexPath})
		return nil
	},
}

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

var skillDetectCmd = &cobra.Command{
	Use:   "skill-detect",
	Short: "Detect domain skills matching codebase file patterns",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		hub := resolveHubPath()
		entries := loadSkillIndexOrBuild(hub)
		root := skillWorkspaceRoot()

		var matched []skillResolvedEntry
		for _, e := range entries {
			if e.Type != "domain" {
				continue
			}
			reasons := skillWorkspaceMatchReasons(root, e)
			if len(reasons) == 0 {
				continue
			}
			matched = append(matched, skillResolvedEntry{
				skillIndexEntry: e,
				Score:           reasonScoreTotal(reasons),
				Reasons:         reasons,
			})
		}
		sortScoredResolvedEntries(matched)

		outputOK(map[string]interface{}{"matched": matched, "total": len(matched), "root": root})
		return nil
	},
}

var skillMatchCmd = &cobra.Command{
	Use:   "skill-match [role] [task]",
	Short: "Match skills to worker role and task",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		role, task := resolveSkillMatchInput(cmd, args)
		if role == "" {
			outputError(1, "worker role is required", nil)
			return nil
		}

		result := matchSkills(resolveHubPath(), role, task)
		outputOK(result)
		return nil
	},
}

var skillInjectCmd = &cobra.Command{
	Use:   "skill-inject [role] [task]",
	Short: "Load matched skills into prompt section text",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		role, task := resolveSkillMatchInput(cmd, args)
		if role == "" {
			outputError(1, "worker role is required", nil)
			return nil
		}

		match := matchSkills(resolveHubPath(), role, task)
		outputOK(renderSkillInjectResult(match))
		return nil
	},
}

var skillListCmd = &cobra.Command{
	Use:   "skill-list",
	Short: "List all installed skills",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		hub := resolveHubPath()
		entries := buildFullIndex(hub)
		outputOK(map[string]interface{}{"skills": entries, "total": len(entries)})
		return nil
	},
}

var skillManifestReadCmd = &cobra.Command{
	Use:   "skill-manifest-read",
	Short: "Read the skills manifest",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		hub := resolveHubPath()
		manifestPath := filepath.Join(hub, "skills", "manifest.json")

		raw, err := os.ReadFile(manifestPath)
		if err != nil {
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

var skillCacheRebuildCmd = &cobra.Command{
	Use:   "skill-cache-rebuild",
	Short: "Force rebuild of skills index cache",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		hub := resolveHubPath()
		indexPath := filepath.Join(hub, "skills", "index.json")
		entries := buildFullIndex(hub)

		data := skillIndexData{
			Entries:   entries,
			UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		}

		if err := os.MkdirAll(filepath.Dir(indexPath), 0755); err != nil {
			outputError(2, fmt.Sprintf("failed to create index directory: %v", err), nil)
			return nil
		}
		encoded, _ := json.MarshalIndent(data, "", "  ")
		if err := os.WriteFile(indexPath, append(encoded, '\n'), 0644); err != nil {
			outputError(2, fmt.Sprintf("failed to write: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{"rebuilt": true, "total": len(entries), "path": indexPath})
		return nil
	},
}

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

func buildFullIndex(hub string) []skillIndexEntry {
	entries := []skillIndexEntry{}
	seen := map[string]bool{}

	for _, root := range skillScanRoots(hub) {
		for _, d := range findSkillDirs(root.Path) {
			entry := indexSkillDir(d, root.IsUserCreated, root.Source)
			if entry == nil {
				continue
			}
			key := entry.Type + ":" + entry.Name
			if seen[key] {
				continue
			}
			seen[key] = true
			entries = append(entries, *entry)
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Type != entries[j].Type {
			return entries[i].Type < entries[j].Type
		}
		return entries[i].Name < entries[j].Name
	})

	return entries
}

func parseSkillFrontmatter(content string) *skillFrontmatter {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil
	}

	var fmLines []string
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			break
		}
		fmLines = append(fmLines, line)
	}
	if len(fmLines) == 0 {
		return nil
	}

	var fm skillFrontmatter
	raw := strings.Join(fmLines, "\n")
	if err := yaml.Unmarshal([]byte(raw), &fm); err != nil {
		fm = skillFrontmatter{}
		for _, line := range fmLines {
			line = strings.TrimSpace(line)
			switch {
			case strings.HasPrefix(line, "name:"):
				fm.Name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
			case strings.HasPrefix(line, "description:"):
				fm.Description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
			case strings.HasPrefix(line, "category:"):
				fm.Category = strings.TrimSpace(strings.TrimPrefix(line, "category:"))
			case strings.HasPrefix(line, "type:"):
				fm.Type = strings.TrimSpace(strings.TrimPrefix(line, "type:"))
			case strings.HasPrefix(line, "detect:"):
				fm.Detect = parseLegacyCSV(strings.TrimSpace(strings.TrimPrefix(line, "detect:")))
			case strings.HasPrefix(line, "roles:"):
				fm.Roles = parseLegacyCSV(strings.TrimSpace(strings.TrimPrefix(line, "roles:")))
			}
		}
	}
	fm.normalize()
	return &fm
}

func parseLegacyCSV(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func (fm *skillFrontmatter) normalize() {
	fm.Type = strings.TrimSpace(fm.Type)
	fm.Category = strings.TrimSpace(fm.Category)
	if fm.Type == "" {
		fm.Type = fm.Category
	}
	if fm.Category == "" {
		fm.Category = fm.Type
	}
	if len(fm.AgentRoles) == 0 {
		fm.AgentRoles = append([]string{}, fm.Roles...)
	}
	if len(fm.Roles) == 0 {
		fm.Roles = append([]string{}, fm.AgentRoles...)
	}
	if len(fm.DetectFiles) == 0 {
		fm.DetectFiles = append([]string{}, fm.Detect...)
	}
	if len(fm.Detect) == 0 {
		fm.Detect = append([]string{}, fm.DetectFiles...)
	}
	if fm.Priority == "" {
		fm.Priority = "normal"
	}
}

func skillScanRoots(hub string) []skillScanRoot {
	roots := []skillScanRoot{
		{Path: ".codex/skills/aether", Source: "repo-codex", IsUserCreated: false},
		{Path: ".agents/skills/aether", Source: "repo-agents", IsUserCreated: false},
		{Path: ".aether/skills-codex", Source: "repo-aether-codex", IsUserCreated: false},
		{Path: ".aether/skills", Source: "repo-aether", IsUserCreated: false},
		{Path: filepath.Join(hub, "skills"), Source: "hub-aether", IsUserCreated: true},
		{Path: filepath.Join(hub, "skills-codex"), Source: "hub-aether-codex", IsUserCreated: false},
	}

	includeUserRoots := true
	if envHub := strings.TrimSpace(os.Getenv("AETHER_HUB_DIR")); envHub != "" {
		includeUserRoots = false
	} else if defaultHub := resolveHubPath(); defaultHub != "" && !samePathOrAncestor(defaultHub, hub) && !samePathOrAncestor(hub, defaultHub) {
		includeUserRoots = false
	}

	if includeUserRoots {
		home, err := os.UserHomeDir()
		if err == nil {
			roots = append(roots,
				skillScanRoot{Path: filepath.Join(home, ".codex", "skills", "aether"), Source: "user-codex", IsUserCreated: true},
				skillScanRoot{Path: filepath.Join(home, ".agents", "skills", "aether"), Source: "user-agents", IsUserCreated: true},
			)
		}
	}

	deduped := make([]skillScanRoot, 0, len(roots))
	seen := map[string]bool{}
	for _, root := range roots {
		if root.Path == "" || seen[root.Path] {
			continue
		}
		seen[root.Path] = true
		deduped = append(deduped, root)
	}
	return deduped
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
		if _, err := os.Stat(filepath.Join(dirPath, "SKILL.md")); err == nil {
			dirs = append(dirs, dirPath)
			continue
		}
		dirs = append(dirs, findSkillDirs(dirPath)...)
	}
	return dirs
}

func indexSkillDir(dir string, isUserCreated bool, source ...string) *skillIndexEntry {
	skillPath := filepath.Join(dir, "SKILL.md")
	raw, err := os.ReadFile(skillPath)
	if err != nil {
		return nil
	}

	fm := parseSkillFrontmatter(string(raw))
	if fm == nil || fm.Name == "" {
		return nil
	}

	sourceName := ""
	if len(source) > 0 {
		sourceName = source[0]
	}

	return &skillIndexEntry{
		Name:           fm.Name,
		Description:    fm.Description,
		Type:           fm.Type,
		Category:       fm.Category,
		Domains:        fm.Domains,
		AgentRoles:     fm.AgentRoles,
		Roles:          fm.Roles,
		DetectFiles:    fm.DetectFiles,
		DetectPackages: fm.DetectPackages,
		Detect:         fm.Detect,
		Priority:       fm.Priority,
		Version:        fm.Version,
		Path:           skillPath,
		IsUserCreated:  isUserCreated,
		Source:         sourceName,
	}
}

func loadSkillIndexOrBuild(hub string) []skillIndexEntry {
	indexPath := filepath.Join(hub, "skills", "index.json")
	raw, err := os.ReadFile(indexPath)
	if err != nil {
		return buildFullIndex(hub)
	}
	var data skillIndexData
	if err := json.Unmarshal(raw, &data); err != nil {
		return buildFullIndex(hub)
	}
	if len(data.Entries) == 0 {
		return buildFullIndex(hub)
	}
	return data.Entries
}

func resolveSkillMatchInput(cmd *cobra.Command, args []string) (string, string) {
	role, _ := cmd.Flags().GetString("role")
	task, _ := cmd.Flags().GetString("task")
	if role == "" && len(args) > 0 {
		role = args[0]
	}
	if task == "" && len(args) > 1 {
		task = strings.Join(args[1:], " ")
	}
	return strings.TrimSpace(role), strings.TrimSpace(task)
}

func matchSkills(hub, role, task string) skillMatchResult {
	return resolveSkillMatchesForRoot(hub, skillWorkspaceRoot(), role, task)
}

func resolveSkillMatchesForRoot(hub, root, role, task string) skillMatchResult {
	entries := loadSkillIndexOrBuild(hub)
	var colonyMatches []scoredSkill
	var domainMatches []scoredSkill
	taskLower := strings.ToLower(strings.TrimSpace(task))

	for _, e := range entries {
		reasons := resolveSkillMatchReasons(root, e, role, taskLower)
		if e.Type == "domain" && !skillHasNonRoleReason(reasons) {
			continue
		}
		score := reasonScoreTotal(reasons)

		if score == 0 {
			continue
		}

		scored := scoredSkill{
			entry: skillResolvedEntry{
				skillIndexEntry: e,
				Score:           score,
				Reasons:         reasons,
			},
			score: score,
		}
		switch e.Type {
		case "colony":
			colonyMatches = append(colonyMatches, scored)
		case "domain":
			domainMatches = append(domainMatches, scored)
		default:
			domainMatches = append(domainMatches, scored)
		}
	}

	sortScoredSkills(colonyMatches)
	sortScoredSkills(domainMatches)

	colonySkills := topResolvedSkillEntries(colonyMatches, 3)
	domainSkills := topResolvedSkillEntries(domainMatches, 3)
	matched := append(extractResolvedSkillNames(colonySkills), extractResolvedSkillNames(domainSkills)...)

	return skillMatchResult{
		Role:         role,
		Task:         task,
		Root:         root,
		ColonySkills: colonySkills,
		DomainSkills: domainSkills,
		Matched:      matched,
		Count:        len(matched),
	}
}

func resolveSkillMatchReasons(root string, entry skillIndexEntry, role, taskLower string) []skillMatchReason {
	reasons := []skillMatchReason{}
	if role != "" && (containsString(entry.AgentRoles, role) || containsString(entry.Roles, role)) {
		reasons = append(reasons, skillMatchReason{
			Code:     "role_match",
			Score:    3,
			Evidence: []string{strings.ToLower(strings.TrimSpace(role))},
		})
	}

	reasons = append(reasons, skillWorkspaceMatchReasons(root, entry)...)

	if taskLower != "" {
		nameLower := strings.ToLower(strings.TrimSpace(entry.Name))
		if nameLower != "" && (strings.Contains(nameLower, taskLower) || strings.Contains(taskLower, nameLower)) {
			reasons = append(reasons, skillMatchReason{
				Code:     "task_name_overlap",
				Score:    1,
				Evidence: []string{nameLower},
			})
		}
		if domains := skillTaskDomainEvidence(entry.Domains, taskLower); len(domains) > 0 {
			reasons = append(reasons, skillMatchReason{
				Code:     "task_domain_overlap",
				Score:    1,
				Evidence: domains,
			})
		}
	}

	return reasons
}

func skillWorkspaceMatchReasons(root string, entry skillIndexEntry) []skillMatchReason {
	if len(entry.DetectFiles) == 0 && len(entry.DetectPackages) == 0 && len(entry.Detect) == 0 {
		return nil
	}

	fileMatches := []string{}
	for _, pattern := range append(entry.DetectFiles, entry.Detect...) {
		fileMatches = append(fileMatches, repoFilePatternMatches(root, pattern)...)
	}
	fileMatches = uniqueSortedSkillStrings(fileMatches)

	packageMatches := []string{}
	for _, pkg := range entry.DetectPackages {
		packageMatches = append(packageMatches, repoPackageMatches(root, pkg)...)
	}
	packageMatches = uniqueSortedSkillStrings(packageMatches)

	reasons := []skillMatchReason{}
	switch {
	case len(fileMatches) > 0 && len(packageMatches) > 0:
		reasons = append(reasons,
			skillMatchReason{Code: "workspace_file", Score: 1, Evidence: fileMatches},
			skillMatchReason{Code: "workspace_package", Score: 1, Evidence: packageMatches},
		)
	case len(fileMatches) > 0:
		reasons = append(reasons, skillMatchReason{Code: "workspace_file", Score: 2, Evidence: fileMatches})
	case len(packageMatches) > 0:
		reasons = append(reasons, skillMatchReason{Code: "workspace_package", Score: 2, Evidence: packageMatches})
	}

	return reasons
}

func skillTaskDomainEvidence(domains []string, taskLower string) []string {
	matched := []string{}
	for _, domain := range domains {
		domain = strings.ToLower(strings.TrimSpace(domain))
		if domain != "" && strings.Contains(taskLower, domain) {
			matched = append(matched, domain)
		}
	}
	return uniqueSortedSkillStrings(matched)
}

func reasonScoreTotal(reasons []skillMatchReason) int {
	total := 0
	for _, reason := range reasons {
		total += reason.Score
	}
	return total
}

func skillHasNonRoleReason(reasons []skillMatchReason) bool {
	for _, reason := range reasons {
		if reason.Code != "role_match" {
			return true
		}
	}
	return false
}

func renderSkillInjectResult(match skillMatchResult) skillInjectResult {
	sections := []string{}
	for _, entry := range append(match.ColonySkills, match.DomainSkills...) {
		content, err := os.ReadFile(entry.Path)
		if err != nil {
			continue
		}
		sections = append(sections, fmt.Sprintf("### Skill: %s\n\n%s", entry.Name, string(content)))
	}

	section := strings.Join(sections, "\n\n---\n\n")
	return skillInjectResult{
		Role:         match.Role,
		Task:         match.Task,
		Root:         match.Root,
		Section:      section,
		SkillSection: section,
		SkillCount:   len(sections),
		ColonyCount:  len(match.ColonySkills),
		DomainCount:  len(match.DomainSkills),
		ColonySkills: match.ColonySkills,
		DomainSkills: match.DomainSkills,
		Matched:      match.Matched,
		Count:        match.Count,
	}
}

func sortScoredSkills(skills []scoredSkill) {
	sort.Slice(skills, func(i, j int) bool {
		if skills[i].score != skills[j].score {
			return skills[i].score > skills[j].score
		}
		if skills[i].entry.Name != skills[j].entry.Name {
			return skills[i].entry.Name < skills[j].entry.Name
		}
		if skills[i].entry.Source != skills[j].entry.Source {
			return skills[i].entry.Source < skills[j].entry.Source
		}
		return skills[i].entry.Path < skills[j].entry.Path
	})
}

func sortScoredResolvedEntries(entries []skillResolvedEntry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Score != entries[j].Score {
			return entries[i].Score > entries[j].Score
		}
		if entries[i].Name != entries[j].Name {
			return entries[i].Name < entries[j].Name
		}
		if entries[i].Source != entries[j].Source {
			return entries[i].Source < entries[j].Source
		}
		return entries[i].Path < entries[j].Path
	})
}

func topResolvedSkillEntries(skills []scoredSkill, limit int) []skillResolvedEntry {
	if len(skills) > limit {
		skills = skills[:limit]
	}
	result := make([]skillResolvedEntry, 0, len(skills))
	for _, item := range skills {
		result = append(result, item.entry)
	}
	return result
}

func extractResolvedSkillNames(entries []skillResolvedEntry) []string {
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name)
	}
	return names
}

func uniqueSortedSkillStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(filepath.ToSlash(value))
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func skillWorkspaceRoot() string {
	if wd, err := os.Getwd(); err == nil {
		if store != nil {
			storeRoot := filepath.Dir(filepath.Dir(store.BasePath()))
			if samePathOrAncestor(storeRoot, wd) || samePathOrAncestor(wd, storeRoot) {
				return storeRoot
			}
		}
		return wd
	}
	if store != nil {
		return filepath.Dir(filepath.Dir(store.BasePath()))
	}
	return "."
}

func samePathOrAncestor(base, target string) bool {
	baseAbs, err := filepath.Abs(base)
	if err != nil {
		return false
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	if baseAbs == targetAbs {
		return true
	}
	return strings.HasPrefix(targetAbs, baseAbs+string(filepath.Separator))
}

func skillMatchesWorkspace(root string, entry skillIndexEntry) bool {
	return len(skillWorkspaceMatchReasons(root, entry)) > 0
}

func repoMatchesFilePattern(root, pattern string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return false
	}

	return len(repoFilePatternMatches(root, pattern)) > 0
}

func repoFilePatternMatches(root, pattern string) []string {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil
	}

	snapshot := getWorkspaceFileSnapshot(root)
	workspaceFileSnapshotCache.mu.Lock()
	if matches, ok := snapshot.PatternMatches[pattern]; ok {
		cached := append([]string(nil), matches...)
		workspaceFileSnapshotCache.mu.Unlock()
		return cached
	}
	workspaceFileSnapshotCache.mu.Unlock()

	matches := []string{}
	for i, rel := range snapshot.RelPaths {
		if ok, _ := filepath.Match(pattern, snapshot.BaseNames[i]); ok {
			matches = append(matches, rel)
			continue
		}
		if ok, _ := filepath.Match(pattern, rel); ok {
			matches = append(matches, rel)
		}
	}
	matches = uniqueSortedSkillStrings(matches)

	workspaceFileSnapshotCache.mu.Lock()
	snapshot.PatternMatches[pattern] = append([]string(nil), matches...)
	workspaceFileSnapshotCache.mu.Unlock()
	return matches
}

func getWorkspaceFileSnapshot(root string) *workspaceFileSnapshot {
	cacheKey := root
	if abs, err := filepath.Abs(root); err == nil {
		cacheKey = abs
	}

	workspaceFileSnapshotCache.mu.Lock()
	if snapshot, ok := workspaceFileSnapshotCache.snapshots[cacheKey]; ok {
		workspaceFileSnapshotCache.mu.Unlock()
		return snapshot
	}
	workspaceFileSnapshotCache.mu.Unlock()

	snapshot := &workspaceFileSnapshot{
		RelPaths:       []string{},
		BaseNames:      []string{},
		PatternMatches: map[string][]string{},
	}

	if populateWorkspaceSnapshotFromGit(cacheKey, snapshot) {
		workspaceFileSnapshotCache.mu.Lock()
		workspaceFileSnapshotCache.snapshots[cacheKey] = snapshot
		workspaceFileSnapshotCache.mu.Unlock()
		return snapshot
	}

	_ = filepath.WalkDir(cacheKey, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if _, skip := skillScanSkipDirs[d.Name()]; skip {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}

		rel, relErr := filepath.Rel(cacheKey, path)
		if relErr != nil {
			return nil
		}
		appendWorkspaceSnapshotPath(snapshot, rel)
		return nil
	})

	workspaceFileSnapshotCache.mu.Lock()
	workspaceFileSnapshotCache.snapshots[cacheKey] = snapshot
	workspaceFileSnapshotCache.mu.Unlock()
	return snapshot
}

func populateWorkspaceSnapshotFromGit(root string, snapshot *workspaceFileSnapshot) bool {
	cmd := exec.Command("git", "-C", root, "ls-files", "-co", "--exclude-standard")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	for _, rel := range strings.Split(string(output), "\n") {
		appendWorkspaceSnapshotPath(snapshot, rel)
	}
	return len(snapshot.RelPaths) > 0
}

func appendWorkspaceSnapshotPath(snapshot *workspaceFileSnapshot, rel string) {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if rel == "" || shouldSkipSkillScanPath(rel) {
		return
	}
	snapshot.RelPaths = append(snapshot.RelPaths, rel)
	snapshot.BaseNames = append(snapshot.BaseNames, filepath.Base(rel))
}

func shouldSkipSkillScanPath(rel string) bool {
	for _, part := range strings.Split(rel, "/") {
		if _, skip := skillScanSkipDirs[part]; skip {
			return true
		}
	}
	return false
}

func repoContainsPackage(root, pkg string) bool {
	return len(repoPackageMatches(root, pkg)) > 0
}

func repoPackageMatches(root, pkg string) []string {
	pkg = strings.TrimSpace(strings.ToLower(pkg))
	if pkg == "" {
		return nil
	}

	manifestPaths := []string{
		"package.json",
		"go.mod",
		"requirements.txt",
		"pyproject.toml",
		"Gemfile",
		"Cargo.toml",
	}

	matches := []string{}
	for _, rel := range manifestPaths {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			continue
		}
		if strings.Contains(strings.ToLower(string(data)), pkg) {
			matches = append(matches, fmt.Sprintf("%s:%s", filepath.ToSlash(rel), pkg))
		}
	}
	return uniqueSortedSkillStrings(matches)
}

func containsString(items []string, want string) bool {
	want = strings.TrimSpace(strings.ToLower(want))
	for _, item := range items {
		if strings.ToLower(strings.TrimSpace(item)) == want {
			return true
		}
	}
	return false
}

func init() {
	skillParseFrontmatterCmd.Flags().String("file", "", "Path to SKILL.md (required)")
	skillMatchCmd.Flags().String("role", "", "Worker role")
	skillMatchCmd.Flags().String("task", "", "Task description")
	skillInjectCmd.Flags().String("role", "", "Worker role")
	skillInjectCmd.Flags().String("task", "", "Task description")
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
