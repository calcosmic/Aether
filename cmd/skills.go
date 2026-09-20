package cmd

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/calcosmic/Aether/pkg/codegraph"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type skillFrontmatter struct {
	Name             string   `json:"name" yaml:"name"`
	Description      string   `json:"description" yaml:"description"`
	Source           string   `json:"source,omitempty" yaml:"source"`
	Type             string   `json:"type,omitempty" yaml:"type"`
	Category         string   `json:"category,omitempty" yaml:"category"`
	Domains          []string `json:"domains,omitempty" yaml:"domains"`
	AgentRoles       []string `json:"agent_roles,omitempty" yaml:"agent_roles"`
	Roles            []string `json:"roles,omitempty" yaml:"roles"`
	WorkflowTriggers []string `json:"workflow_triggers,omitempty" yaml:"workflow_triggers"`
	TaskKeywords     []string `json:"task_keywords,omitempty" yaml:"task_keywords"`
	DetectFiles      []string `json:"detect_files,omitempty" yaml:"detect_files"`
	DetectPackages   []string `json:"detect_packages,omitempty" yaml:"detect_packages"`
	Detect           []string `json:"detect,omitempty" yaml:"detect"`
	Priority         string   `json:"priority,omitempty" yaml:"priority"`
	Version          string   `json:"version,omitempty" yaml:"version"`
}

type skillIndexEntry struct {
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	Type             string   `json:"type"`
	Category         string   `json:"category"`
	Domains          []string `json:"domains,omitempty"`
	AgentRoles       []string `json:"agent_roles,omitempty"`
	Roles            []string `json:"roles,omitempty"`
	WorkflowTriggers []string `json:"workflow_triggers,omitempty"`
	TaskKeywords     []string `json:"task_keywords,omitempty"`
	DetectFiles      []string `json:"detect_files,omitempty"`
	DetectPackages   []string `json:"detect_packages,omitempty"`
	Detect           []string `json:"detect,omitempty"`
	Priority         string   `json:"priority,omitempty"`
	Version          string   `json:"version,omitempty"`
	Path             string   `json:"path"`
	IsUserCreated    bool     `json:"is_user_created"`
	Source           string   `json:"source,omitempty"`
	DeclaredSource   string   `json:"declared_source,omitempty"`
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

const (
	maintenanceSkillsSchemaVersion     = "maintenance-skills/v1"
	maintenanceSkillsDiffSchemaVersion = "maintenance-skills-diff/v1"
)

type maintenanceSkillInventoryEntry struct {
	Identity       string           `json:"identity"`
	SourceIdentity string           `json:"source_identity"`
	Source         string           `json:"source"`
	Path           string           `json:"path"`
	RelativePath   string           `json:"relative_path"`
	PathClass      string           `json:"path_class"`
	Digest         string           `json:"digest"`
	ParseStatus    string           `json:"parse_status"`
	Metadata       skillFrontmatter `json:"metadata"`
	Errors         []string         `json:"errors"`
}

type maintenanceSkillInvalidEntry struct {
	Identity string   `json:"identity"`
	Path     string   `json:"path"`
	Digest   string   `json:"digest"`
	Errors   []string `json:"errors"`
}

type maintenanceSkillInventoryReceipt struct {
	SchemaVersion string                           `json:"schema_version"`
	OperationID   string                           `json:"operation_id"`
	Root          string                           `json:"root"`
	Hub           string                           `json:"hub"`
	Entries       []maintenanceSkillInventoryEntry `json:"entries"`
	Invalid       []maintenanceSkillInvalidEntry   `json:"invalid"`
	Verification  maintenanceSkillVerification     `json:"verification"`
	StateEffect   colony.LifecycleStateEffect      `json:"state_effect"`
	NextAction    string                           `json:"next_action"`
}

type maintenanceSkillVerification struct {
	Status       string `json:"status"`
	EntryCount   int    `json:"entry_count"`
	InvalidCount int    `json:"invalid_count"`
}

type maintenanceSkillDelta struct {
	Identity     string   `json:"identity"`
	Name         string   `json:"name,omitempty"`
	BeforePath   string   `json:"before_path,omitempty"`
	AfterPath    string   `json:"after_path,omitempty"`
	BeforeDigest string   `json:"before_digest,omitempty"`
	AfterDigest  string   `json:"after_digest,omitempty"`
	Errors       []string `json:"errors"`
}

type maintenanceSkillDiffResult struct {
	SchemaVersion string                           `json:"schema_version"`
	OperationID   string                           `json:"operation_id"`
	Before        string                           `json:"before"`
	After         string                           `json:"after"`
	Added         []maintenanceSkillDelta          `json:"added"`
	Removed       []maintenanceSkillDelta          `json:"removed"`
	Changed       []maintenanceSkillDelta          `json:"changed"`
	Invalid       []maintenanceSkillDelta          `json:"invalid"`
	Verification  maintenanceSkillDiffVerification `json:"verification"`
	StateEffect   colony.LifecycleStateEffect      `json:"state_effect"`
	NextAction    string                           `json:"next_action"`
}

type maintenanceSkillDiffVerification struct {
	Status       string `json:"status"`
	AddedCount   int    `json:"added_count"`
	RemovedCount int    `json:"removed_count"`
	ChangedCount int    `json:"changed_count"`
	InvalidCount int    `json:"invalid_count"`
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
	Workflow     string               `json:"workflow,omitempty"`
	Task         string               `json:"task,omitempty"`
	Root         string               `json:"root,omitempty"`
	ColonySkills []skillResolvedEntry `json:"colony_skills"`
	DomainSkills []skillResolvedEntry `json:"domain_skills"`
	Matched      []string             `json:"matched"`
	Count        int                  `json:"count"`
}

type skillInjectResult struct {
	Role         string               `json:"role"`
	Workflow     string               `json:"workflow,omitempty"`
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
	BudgetChars  int                  `json:"budget_chars"`
}

const skillInjectNormalBudgetChars = 8000
const skillInjectCompactBudgetChars = 4000
const skillInjectBudgetChars = skillInjectNormalBudgetChars
const skillInjectBudgetEnvVar = "AETHER_SKILL_BUDGET"

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

var skillIndexRuntimeCache = struct {
	mu      sync.Mutex
	entries map[string][]skillIndexEntry
}{
	entries: map[string][]skillIndexEntry{},
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

		isUserCreated := false
		if userExists == nil {
			isUserCreated = skillFileDeclaresSource(userPath, "custom") || (shippedExists != nil && !skillFileDeclaresSource(userPath, "shipped"))
		}
		outputOK(map[string]interface{}{
			"skill":           name,
			"is_user_created": isUserCreated,
			"in_hub":          userExists == nil,
			"in_shipped":      shippedExists == nil,
		})
		return nil
	},
}

func skillFileDeclaresSource(path, expected string) bool {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	fm := parseSkillFrontmatter(string(raw))
	return fm != nil && strings.TrimSpace(fm.Source) == expected
}

func buildFullIndex(hub string) []skillIndexEntry {
	selected := map[string]skillIndexEntry{}
	selectedRank := map[string]int{}
	sourceCheckout := isAetherSourceCheckout(skillWorkspaceRoot())

	for _, root := range skillScanRoots(hub) {
		for _, d := range findSkillDirs(root.Path) {
			entry := indexSkillDir(d, root.IsUserCreated, root.Source)
			if entry == nil {
				continue
			}
			key := entry.Type + ":" + entry.Name
			rank := skillIndexEntryPriority(*entry, sourceCheckout)
			if existingRank, ok := selectedRank[key]; ok && existingRank <= rank {
				continue
			}
			selected[key] = *entry
			selectedRank[key] = rank
		}
	}

	entries := make([]skillIndexEntry, 0, len(selected))
	for _, entry := range selected {
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Type != entries[j].Type {
			return entries[i].Type < entries[j].Type
		}
		return entries[i].Name < entries[j].Name
	})

	return entries
}

func skillIndexEntryPriority(entry skillIndexEntry, sourceCheckout bool) int {
	switch strings.TrimSpace(entry.DeclaredSource) {
	case "custom":
		return 10
	case "learned":
		return 20
	case "shipped":
		if entry.Source == "repo-aether" && sourceCheckout {
			return 25
		}
		if entry.Source == "hub-aether-shipped" {
			return 30
		}
		return 60
	}

	switch entry.Source {
	case "repo-learned":
		return 20
	case "hub-aether-shipped":
		return 30
	case "repo-aether":
		if sourceCheckout {
			return 35
		}
		return 55
	case "hub-aether-domain":
		return 50
	default:
		return 70
	}
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
			case strings.HasPrefix(line, "source:"):
				fm.Source = strings.TrimSpace(strings.TrimPrefix(line, "source:"))
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
	fm.Source = strings.TrimSpace(fm.Source)
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
	return skillScanRootsForWorkspace(hub, ".")
}

func skillScanRootsForWorkspace(hub, workspace string) []skillScanRoot {
	roots := []skillScanRoot{
		{Path: filepath.Join(workspace, ".aether", "skills"), Source: "repo-aether", IsUserCreated: false},
		{Path: filepath.Join(workspace, ".aether", "hive", "skills"), Source: "repo-learned", IsUserCreated: false},
		{Path: filepath.Join(hub, "system", "skills"), Source: "hub-aether-shipped", IsUserCreated: false},
		{Path: filepath.Join(hub, "skills", "domain"), Source: "hub-aether-domain", IsUserCreated: true},
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
		Name:             fm.Name,
		Description:      fm.Description,
		Type:             fm.Type,
		Category:         fm.Category,
		Domains:          fm.Domains,
		AgentRoles:       fm.AgentRoles,
		Roles:            fm.Roles,
		WorkflowTriggers: fm.WorkflowTriggers,
		TaskKeywords:     fm.TaskKeywords,
		DetectFiles:      fm.DetectFiles,
		DetectPackages:   fm.DetectPackages,
		Detect:           fm.Detect,
		Priority:         fm.Priority,
		Version:          fm.Version,
		Path:             skillPath,
		IsUserCreated:    skillIsUserCreatedFromSource(isUserCreated, sourceName, fm.Source),
		Source:           sourceName,
		DeclaredSource:   fm.Source,
	}
}

func skillIsUserCreatedFromSource(rootUserCreated bool, rootSource, declaredSource string) bool {
	switch strings.TrimSpace(declaredSource) {
	case "custom":
		return true
	case "shipped":
		return false
	}
	switch rootSource {
	case "repo-learned", "hub-aether-domain":
		return true
	default:
		return rootUserCreated
	}
}

func loadSkillIndexOrBuild(hub string) []skillIndexEntry {
	cacheKey := skillIndexRuntimeCacheKey(hub)
	skillIndexRuntimeCache.mu.Lock()
	if cached, ok := skillIndexRuntimeCache.entries[cacheKey]; ok {
		result := append([]skillIndexEntry(nil), cached...)
		skillIndexRuntimeCache.mu.Unlock()
		return result
	}
	skillIndexRuntimeCache.mu.Unlock()

	entries := buildFullIndex(hub)
	if len(entries) > 0 {
		skillIndexRuntimeCache.mu.Lock()
		skillIndexRuntimeCache.entries[cacheKey] = append([]skillIndexEntry(nil), entries...)
		skillIndexRuntimeCache.mu.Unlock()
		return entries
	}

	indexPath := filepath.Join(hub, "skills", "index.json")
	raw, err := os.ReadFile(indexPath)
	if err != nil {
		return entries
	}
	var data skillIndexData
	if err := json.Unmarshal(raw, &data); err != nil {
		return entries
	}
	if len(data.Entries) == 0 {
		return entries
	}
	skillIndexRuntimeCache.mu.Lock()
	skillIndexRuntimeCache.entries[cacheKey] = append([]skillIndexEntry(nil), data.Entries...)
	skillIndexRuntimeCache.mu.Unlock()
	return data.Entries
}

func skillIndexRuntimeCacheKey(hub string) string {
	parts := []string{hub}
	if hubAbs, err := filepath.Abs(hub); err == nil {
		parts[0] = hubAbs
	}
	if wd, err := os.Getwd(); err == nil {
		if wdAbs, absErr := filepath.Abs(wd); absErr == nil {
			wd = wdAbs
		}
		parts = append(parts, wd)
	}
	return strings.Join(parts, "|")
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
	return matchSkillsForWorkflow(hub, "", role, task)
}

func matchSkillsForWorkflow(hub, workflow, role, task string) skillMatchResult {
	return resolveSkillMatchesForRootWithWorkflow(hub, skillWorkspaceRoot(), workflow, role, task)
}

func resolveSkillMatchesForRoot(hub, root, role, task string) skillMatchResult {
	return resolveSkillMatchesForRootWithWorkflow(hub, root, "", role, task)
}

func resolveSkillMatchesForRootWithWorkflow(hub, root, workflow, role, task string) skillMatchResult {
	entries := loadSkillIndexOrBuild(hub)
	var colonyMatches []scoredSkill
	var domainMatches []scoredSkill
	workflow = strings.ToLower(strings.TrimSpace(workflow))
	taskLower := strings.ToLower(strings.TrimSpace(task))

	for _, e := range entries {
		reasons := resolveSkillMatchReasons(root, e, workflow, role, taskLower)
		if e.Type == "colony" && skillRequiresWorkflowOrTaskEvidence(e) {
			if skillHasRoleGate(e) && !skillRoleMatches(e, role) {
				continue
			}
			if !skillHasRequiredCuratedSkillEvidence(e, reasons) {
				continue
			}
		}
		if e.Type == "domain" && !skillHasNonRoleReason(reasons) {
			continue
		}
		// A skill that declares what it is for must actually be for this work.
		// Job title and workflow together used to be enough (3 + 4), so
		// ai-design-contract -- whose own description reads "use when a phase
		// involves LLMs, AI agents, RAG..." and which declares the keywords to
		// prove it -- reached a worker whose job was to create twelve folders in
		// a notes vault.
		//
		// Two deliberate exemptions. A skill declaring no task vocabulary is
		// genuinely role-general (commit style, TDD discipline) and still matches
		// on job title. And when no task is supplied there is nothing to gate on,
		// so the gate cannot apply.
		if taskLower != "" && skillDeclaresTaskVocabulary(e) && !skillHasTaskOrWorkspaceEvidence(reasons) {
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
		Workflow:     workflow,
		Task:         task,
		Root:         root,
		ColonySkills: colonySkills,
		DomainSkills: domainSkills,
		Matched:      matched,
		Count:        len(matched),
	}
}

// skillHasTaskOrWorkspaceEvidence reports whether anything about this skill
// connects it to the actual work -- the task's words, or something detected in
// the repository it is running in. Job title and workflow are not evidence: every
// build has a workflow and every worker has a job title.
//
// task_name_overlap is deliberately excluded. Matching on a skill's own name
// makes the filename a selection input, so a skill could win a slot by being
// renamed.
// skillDeclaresTaskVocabulary reports whether a skill states what work it is
// for. A skill that does is held to it; a skill that does not is treated as
// role-general.
func skillDeclaresTaskVocabulary(entry skillIndexEntry) bool {
	return len(entry.TaskKeywords) > 0 || len(entry.Domains) > 0
}

func skillHasTaskOrWorkspaceEvidence(reasons []skillMatchReason) bool {
	for _, reason := range reasons {
		switch reason.Code {
		case "task_keyword", "task_domain_overlap", "workspace_file", "workspace_package":
			if len(reason.Evidence) > 0 || reason.Score > 0 {
				return true
			}
		}
	}
	return false
}

func resolveSkillMatchReasons(root string, entry skillIndexEntry, workflow, role, taskLower string) []skillMatchReason {
	reasons := []skillMatchReason{}
	if role != "" && skillRoleMatches(entry, role) {
		reasons = append(reasons, skillMatchReason{
			Code:     "role_match",
			Score:    3,
			Evidence: []string{strings.ToLower(strings.TrimSpace(role))},
		})
	}
	if workflow != "" && containsString(entry.WorkflowTriggers, workflow) {
		reasons = append(reasons, skillMatchReason{
			Code:     "workflow_trigger",
			Score:    4,
			Evidence: []string{workflow},
		})
	}

	reasons = append(reasons, skillWorkspaceMatchReasons(root, entry)...)

	if taskLower != "" {
		if keywords := skillTaskKeywordEvidence(entry.TaskKeywords, taskLower); len(keywords) > 0 {
			reasons = append(reasons, skillMatchReason{
				Code:     "task_keyword",
				Score:    2,
				Evidence: keywords,
			})
		}
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

func skillTaskKeywordEvidence(keywords []string, taskLower string) []string {
	matched := []string{}
	for _, keyword := range keywords {
		keyword = strings.ToLower(strings.TrimSpace(keyword))
		if keyword != "" && strings.Contains(taskLower, keyword) {
			matched = append(matched, keyword)
		}
	}
	return uniqueSortedSkillStrings(matched)
}

func skillRequiresWorkflowOrTaskEvidence(entry skillIndexEntry) bool {
	return len(entry.WorkflowTriggers) > 0 || len(entry.TaskKeywords) > 0
}

func skillHasRoleGate(entry skillIndexEntry) bool {
	return len(entry.AgentRoles) > 0 || len(entry.Roles) > 0
}

func skillRoleMatches(entry skillIndexEntry, role string) bool {
	return containsString(entry.AgentRoles, role) || containsString(entry.Roles, role)
}

func reasonScoreTotal(reasons []skillMatchReason) int {
	total := 0
	for _, reason := range reasons {
		total += reason.Score
	}
	return total
}

func skillHasAnyReason(reasons []skillMatchReason, codes ...string) bool {
	for _, reason := range reasons {
		for _, code := range codes {
			if reason.Code == code {
				return true
			}
		}
	}
	return false
}

func skillHasRequiredCuratedSkillEvidence(entry skillIndexEntry, reasons []skillMatchReason) bool {
	hasWorkflowRule := len(entry.WorkflowTriggers) > 0
	hasKeywordRule := len(entry.TaskKeywords) > 0
	workflowMatched := skillHasAnyReason(reasons, "workflow_trigger")
	keywordMatched := skillHasAnyReason(reasons, "task_keyword")
	switch {
	case hasWorkflowRule && hasKeywordRule:
		return workflowMatched && keywordMatched
	case hasWorkflowRule:
		return workflowMatched
	case hasKeywordRule:
		return keywordMatched
	default:
		return true
	}
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
	return renderSkillInjectResultWithBudget(match, resolveSkillInjectBudget(false, 0))
}

func renderSkillInjectResultWithBudget(match skillMatchResult, budget int) skillInjectResult {
	budget = normalizeSkillInjectBudget(budget)
	sections := []string{}
	for _, entry := range append(match.ColonySkills, match.DomainSkills...) {
		content, err := os.ReadFile(entry.Path)
		if err != nil {
			continue
		}
		nextSection := fmt.Sprintf("### Skill: %s\n\n%s", entry.Name, string(content))
		candidate := strings.Join(append(append([]string{}, sections...), nextSection), "\n\n---\n\n")
		if len(candidate) > budget {
			// A skill that does not fit is skipped whole, not sliced. Cutting at
			// a character count ended a live worker's brief mid-sentence -- "If
			// the prompt is large or [truncated]" -- so the worker paid for the
			// tokens and received something incoherent. Skipping rather than
			// stopping means one oversized skill does not starve the smaller
			// ones behind it.
			continue
		}
		sections = append(sections, nextSection)
	}

	section := strings.Join(sections, "\n\n---\n\n")
	return skillInjectResult{
		Role:         match.Role,
		Workflow:     match.Workflow,
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
		BudgetChars:  budget,
	}
}

func resolveSkillInjectBudget(compact bool, explicit int) int {
	if explicit > 0 {
		return normalizeSkillInjectBudget(explicit)
	}
	if raw := strings.TrimSpace(os.Getenv(skillInjectBudgetEnvVar)); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			return normalizeSkillInjectBudget(value)
		}
	}
	if compact {
		return skillInjectCompactBudgetChars
	}
	return skillInjectNormalBudgetChars
}

func normalizeSkillInjectBudget(budget int) int {
	if budget <= 0 {
		return skillInjectNormalBudgetChars
	}
	if budget < 512 {
		return 512
	}
	return budget
}

// truncateSkillInjectSection was removed in Phase 181. It sliced a composed
// skill section at a character count and appended a "[truncated]" marker, which
// ended a live worker's brief on "If the prompt is large or". Over-budget skills
// are now skipped whole in renderSkillInjectResultWithBudget. Deleted rather than
// left in place: a function with no caller is the defect this milestone exists
// to stop shipping.

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
			if codegraph.ShouldSkipDir(d.Name()) {
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
		if codegraph.ShouldSkipDir(part) {
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

var maintenanceSkillsCmd = &cobra.Command{
	Use:         "skills",
	Short:       "Inspect live skill sources without rebuilding a cache",
	Args:        cobra.NoArgs,
	Annotations: map[string]string{"aether.io/read-only": "true", "aether.io/store-free": "true"},
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Help()
	},
}

var maintenanceSkillsInspectCmd = &cobra.Command{
	Use:         "inspect",
	Short:       "Create a read-only receipt from live skill sources",
	Args:        cobra.NoArgs,
	Annotations: map[string]string{"aether.io/read-only": "true", "aether.io/store-free": "true"},
	RunE: func(cmd *cobra.Command, _ []string) error {
		root, _ := cmd.Flags().GetString("root")
		hub, _ := cmd.Flags().GetString("hub")
		result := scanMaintenanceSkills(root, hub)
		outputWorkflow(result, renderMaintenanceSkillInventory(result))
		return nil
	},
}

var maintenanceSkillsDiffCmd = &cobra.Command{
	Use:         "diff",
	Short:       "Compare live skill inventory receipts or a supplied manifest",
	Args:        cobra.NoArgs,
	Annotations: map[string]string{"aether.io/read-only": "true", "aether.io/store-free": "true"},
	RunE:        runMaintenanceSkillsDiff,
}

func scanMaintenanceSkills(root, hub string) maintenanceSkillInventoryReceipt {
	root = resolveMaintenanceSkillRoot(root)
	hub = resolveMaintenanceSkillHub(hub)
	receipt := maintenanceSkillInventoryReceipt{
		SchemaVersion: maintenanceSkillsSchemaVersion,
		OperationID:   "skills.inspect",
		Root:          root,
		Hub:           hub,
		Entries:       []maintenanceSkillInventoryEntry{},
		Invalid:       []maintenanceSkillInvalidEntry{},
		StateEffect:   colony.LifecycleStateEffectNone,
		NextAction:    "aether maintenance skills diff --before <receipt> --after <receipt>",
	}

	for _, scanRoot := range skillScanRootsForWorkspace(hub, root) {
		for _, dir := range findSkillDirs(scanRoot.Path) {
			receipt.Entries = append(receipt.Entries, inspectMaintenanceSkill(scanRoot, dir))
		}
	}
	sortMaintenanceSkillInventoryEntries(receipt.Entries)
	for _, entry := range receipt.Entries {
		if entry.ParseStatus != "invalid" {
			continue
		}
		receipt.Invalid = append(receipt.Invalid, maintenanceSkillInvalidEntry{
			Identity: entry.Identity,
			Path:     entry.Path,
			Digest:   entry.Digest,
			Errors:   append([]string{}, entry.Errors...),
		})
	}
	status := "pass"
	if len(receipt.Invalid) > 0 {
		status = "warning"
	}
	receipt.Verification = maintenanceSkillVerification{
		Status:       status,
		EntryCount:   len(receipt.Entries),
		InvalidCount: len(receipt.Invalid),
	}
	return receipt
}

func resolveMaintenanceSkillRoot(root string) string {
	if strings.TrimSpace(root) == "" {
		root = skillWorkspaceRoot()
	}
	if absolute, err := filepath.Abs(root); err == nil {
		return filepath.Clean(absolute)
	}
	return filepath.Clean(root)
}

func resolveMaintenanceSkillHub(hub string) string {
	if strings.TrimSpace(hub) == "" {
		hub = resolveHubPath()
	}
	if absolute, err := filepath.Abs(hub); err == nil {
		return filepath.Clean(absolute)
	}
	return filepath.Clean(hub)
}

func inspectMaintenanceSkill(root skillScanRoot, dir string) maintenanceSkillInventoryEntry {
	path := filepath.Join(dir, "SKILL.md")
	relativePath, err := filepath.Rel(root.Path, path)
	if err != nil {
		relativePath = filepath.Base(dir) + "/SKILL.md"
	}
	relativePath = filepath.ToSlash(relativePath)
	sourceIdentity := root.Source + ":" + relativePath
	entry := maintenanceSkillInventoryEntry{
		Identity:       sourceIdentity,
		SourceIdentity: sourceIdentity,
		Source:         root.Source,
		Path:           filepath.Clean(path),
		RelativePath:   relativePath,
		PathClass:      maintenanceSkillPathClass(root.Source),
		ParseStatus:    "invalid",
		Errors:         []string{},
	}
	raw, readErr := os.ReadFile(path)
	if readErr != nil {
		entry.Errors = append(entry.Errors, fmt.Sprintf("read skill: %v", readErr))
		return entry
	}
	digest := sha256.Sum256(raw)
	entry.Digest = fmt.Sprintf("%x", digest[:])
	entry.Metadata, entry.Errors = parseMaintenanceSkillMetadata(string(raw))
	if len(entry.Errors) == 0 {
		entry.ParseStatus = "valid"
	}
	return entry
}

func maintenanceSkillPathClass(source string) string {
	switch source {
	case "repo-aether":
		return "repository"
	case "repo-learned":
		return "repository_learned"
	case "hub-aether-shipped":
		return "hub_shipped"
	case "hub-aether-domain":
		return "hub_custom"
	default:
		return "unknown"
	}
}

func parseMaintenanceSkillMetadata(content string) (skillFrontmatter, []string) {
	var metadata skillFrontmatter
	var problems []string
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		problems = append(problems, "missing opening YAML frontmatter delimiter")
	} else {
		closing := -1
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				closing = i
				break
			}
		}
		if closing < 0 {
			problems = append(problems, "missing closing YAML frontmatter delimiter")
		} else if err := yaml.Unmarshal([]byte(strings.Join(lines[1:closing], "\n")), &metadata); err != nil {
			problems = append(problems, "invalid YAML frontmatter: "+err.Error())
		}
	}
	if fallback := parseSkillFrontmatter(content); fallback != nil {
		if strings.TrimSpace(metadata.Name) == "" || len(problems) > 0 {
			metadata = *fallback
		}
	}
	metadata.normalize()
	if strings.TrimSpace(metadata.Name) == "" {
		problems = append(problems, "frontmatter field name is required")
	}
	return metadata, uniqueSortedSkillStrings(problems)
}

func runMaintenanceSkillsDiff(cmd *cobra.Command, _ []string) error {
	beforePath, _ := cmd.Flags().GetString("before")
	afterPath, _ := cmd.Flags().GetString("after")
	if strings.TrimSpace(beforePath) == "" {
		return fmt.Errorf("--before must name a live-scan receipt or skill manifest")
	}
	before, beforeManifest, err := readMaintenanceSkillInventory(beforePath)
	if err != nil {
		return fmt.Errorf("read --before %s: %w", beforePath, err)
	}

	var after maintenanceSkillInventoryReceipt
	afterManifest := false
	afterLabel := afterPath
	if strings.TrimSpace(afterPath) == "" {
		root, _ := cmd.Flags().GetString("root")
		hub, _ := cmd.Flags().GetString("hub")
		after = scanMaintenanceSkills(root, hub)
		afterLabel = "live"
	} else {
		after, afterManifest, err = readMaintenanceSkillInventory(afterPath)
		if err != nil {
			return fmt.Errorf("read --after %s: %w", afterPath, err)
		}
	}
	if beforeManifest && !afterManifest {
		after = alignMaintenanceSkillsToManifest(after)
	}
	if afterManifest && !beforeManifest {
		before = alignMaintenanceSkillsToManifest(before)
	}

	result := diffMaintenanceSkillInventories(before, after, beforePath, afterLabel)
	outputWorkflow(result, renderMaintenanceSkillDiff(result))
	return nil
}

func readMaintenanceSkillInventory(path string) (maintenanceSkillInventoryReceipt, bool, error) {
	var receipt maintenanceSkillInventoryReceipt
	raw, err := os.ReadFile(path)
	if err != nil {
		return receipt, false, err
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(raw, &document); err != nil {
		return receipt, false, err
	}
	if wrapped, ok := document["result"]; ok {
		raw = wrapped
		if err := json.Unmarshal(raw, &document); err != nil {
			return receipt, false, err
		}
	}
	if _, ok := document["entries"]; ok {
		if err := json.Unmarshal(raw, &receipt); err != nil {
			return receipt, false, err
		}
		if receipt.Entries == nil {
			receipt.Entries = []maintenanceSkillInventoryEntry{}
		}
		if receipt.Invalid == nil {
			receipt.Invalid = []maintenanceSkillInvalidEntry{}
		}
		return receipt, false, nil
	}
	if _, ok := document["skills"]; ok {
		var manifest skillManifestData
		if err := json.Unmarshal(raw, &manifest); err != nil {
			return receipt, true, err
		}
		receipt = maintenanceSkillInventoryReceipt{
			SchemaVersion: maintenanceSkillsSchemaVersion,
			OperationID:   "skills.inspect",
			Entries:       []maintenanceSkillInventoryEntry{},
			Invalid:       []maintenanceSkillInvalidEntry{},
			StateEffect:   colony.LifecycleStateEffectNone,
		}
		for _, skill := range manifest.Skills {
			name := strings.TrimSpace(skill.Name)
			identity := "manifest:" + strings.ToLower(name)
			entry := maintenanceSkillInventoryEntry{
				Identity:       identity,
				SourceIdentity: identity,
				Source:         "manifest",
				Path:           path,
				RelativePath:   name,
				PathClass:      "manifest",
				Digest:         skill.Checksum,
				ParseStatus:    "valid",
				Metadata:       skillFrontmatter{Name: name, Version: skill.Version},
				Errors:         []string{},
			}
			if name == "" || strings.TrimSpace(skill.Checksum) == "" {
				entry.ParseStatus = "invalid"
				entry.Errors = append(entry.Errors, "manifest skill requires name and checksum")
			}
			receipt.Entries = append(receipt.Entries, entry)
		}
		sortMaintenanceSkillInventoryEntries(receipt.Entries)
		return receipt, true, nil
	}
	return receipt, false, fmt.Errorf("unsupported document: expected entries or skills")
}

func alignMaintenanceSkillsToManifest(receipt maintenanceSkillInventoryReceipt) maintenanceSkillInventoryReceipt {
	for i := range receipt.Entries {
		name := strings.ToLower(strings.TrimSpace(receipt.Entries[i].Metadata.Name))
		if name != "" {
			receipt.Entries[i].Identity = "manifest:" + name
		}
	}
	sortMaintenanceSkillInventoryEntries(receipt.Entries)
	return receipt
}

func sortMaintenanceSkillInventoryEntries(entries []maintenanceSkillInventoryEntry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Identity != entries[j].Identity {
			return entries[i].Identity < entries[j].Identity
		}
		if entries[i].Path != entries[j].Path {
			return entries[i].Path < entries[j].Path
		}
		return entries[i].Digest < entries[j].Digest
	})
}

func diffMaintenanceSkillInventories(before, after maintenanceSkillInventoryReceipt, beforeLabel, afterLabel string) maintenanceSkillDiffResult {
	result := maintenanceSkillDiffResult{
		SchemaVersion: maintenanceSkillsDiffSchemaVersion,
		OperationID:   "skills.diff",
		Before:        beforeLabel,
		After:         afterLabel,
		Added:         []maintenanceSkillDelta{},
		Removed:       []maintenanceSkillDelta{},
		Changed:       []maintenanceSkillDelta{},
		Invalid:       []maintenanceSkillDelta{},
		StateEffect:   colony.LifecycleStateEffectNone,
		NextAction:    "aether maintenance skills inspect",
	}
	beforeEntries := maintenanceSkillEntriesByIdentity(before.Entries)
	afterEntries := maintenanceSkillEntriesByIdentity(after.Entries)
	for identity, entry := range afterEntries {
		previous, exists := beforeEntries[identity]
		if !exists {
			result.Added = append(result.Added, maintenanceSkillDeltaFromEntries(identity, nil, &entry))
			continue
		}
		if previous.Digest != entry.Digest {
			result.Changed = append(result.Changed, maintenanceSkillDeltaFromEntries(identity, &previous, &entry))
		}
	}
	for identity, entry := range beforeEntries {
		if _, exists := afterEntries[identity]; !exists {
			result.Removed = append(result.Removed, maintenanceSkillDeltaFromEntries(identity, &entry, nil))
		}
	}
	for _, side := range []struct {
		name    string
		entries []maintenanceSkillInventoryEntry
	}{
		{name: "before", entries: before.Entries},
		{name: "after", entries: after.Entries},
	} {
		for _, entry := range side.entries {
			if entry.ParseStatus != "invalid" {
				continue
			}
			delta := maintenanceSkillDeltaFromEntries(entry.Identity, nil, &entry)
			if side.name == "before" {
				delta = maintenanceSkillDeltaFromEntries(entry.Identity, &entry, nil)
			}
			delta.Identity = side.name + ":" + entry.Identity
			result.Invalid = append(result.Invalid, delta)
		}
	}
	for _, deltas := range [][]maintenanceSkillDelta{result.Added, result.Removed, result.Changed, result.Invalid} {
		sort.Slice(deltas, func(i, j int) bool { return deltas[i].Identity < deltas[j].Identity })
	}
	status := "clean"
	if len(result.Added)+len(result.Removed)+len(result.Changed)+len(result.Invalid) > 0 {
		status = "drift"
		result.NextAction = "aether maintenance"
	}
	result.Verification = maintenanceSkillDiffVerification{
		Status:       status,
		AddedCount:   len(result.Added),
		RemovedCount: len(result.Removed),
		ChangedCount: len(result.Changed),
		InvalidCount: len(result.Invalid),
	}
	return result
}

func maintenanceSkillEntriesByIdentity(entries []maintenanceSkillInventoryEntry) map[string]maintenanceSkillInventoryEntry {
	result := make(map[string]maintenanceSkillInventoryEntry, len(entries))
	for _, entry := range entries {
		if _, exists := result[entry.Identity]; !exists {
			result[entry.Identity] = entry
		}
	}
	return result
}

func maintenanceSkillDeltaFromEntries(identity string, before, after *maintenanceSkillInventoryEntry) maintenanceSkillDelta {
	delta := maintenanceSkillDelta{Identity: identity, Errors: []string{}}
	if before != nil {
		delta.Name = before.Metadata.Name
		delta.BeforePath = before.Path
		delta.BeforeDigest = before.Digest
		delta.Errors = append(delta.Errors, before.Errors...)
	}
	if after != nil {
		if delta.Name == "" {
			delta.Name = after.Metadata.Name
		}
		delta.AfterPath = after.Path
		delta.AfterDigest = after.Digest
		delta.Errors = append(delta.Errors, after.Errors...)
	}
	delta.Errors = uniqueSortedSkillStrings(delta.Errors)
	return delta
}

func renderMaintenanceSkillInventory(result maintenanceSkillInventoryReceipt) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("maintenance"), "Live Skill Inventory"))
	b.WriteString(visualDividerStr())
	fmt.Fprintf(&b, "Repository: %s\nHub: %s\n", result.Root, result.Hub)
	b.WriteString(renderStageMarker("Sources"))
	for _, entry := range result.Entries {
		name := entry.Metadata.Name
		if strings.TrimSpace(name) == "" {
			name = "invalid skill"
		}
		fmt.Fprintf(&b, "%s — %s [%s]\n", entry.Identity, name, entry.ParseStatus)
		fmt.Fprintf(&b, "  %s  %s\n", entry.Digest, entry.Path)
		for _, problem := range entry.Errors {
			fmt.Fprintf(&b, "  Error: %s\n", problem)
		}
	}
	fmt.Fprintf(&b, "\nState effect: none\nScanned: %d; invalid: %d\n", result.Verification.EntryCount, result.Verification.InvalidCount)
	b.WriteString(renderNextUp(result.NextAction))
	return b.String()
}

func renderMaintenanceSkillDiff(result maintenanceSkillDiffResult) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("maintenance"), "Live Skill Diff"))
	b.WriteString(visualDividerStr())
	fmt.Fprintf(&b, "Before: %s\nAfter: %s\n", result.Before, result.After)
	for _, group := range []struct {
		name   string
		values []maintenanceSkillDelta
	}{
		{name: "Added", values: result.Added},
		{name: "Removed", values: result.Removed},
		{name: "Changed", values: result.Changed},
		{name: "Invalid", values: result.Invalid},
	} {
		b.WriteString(renderStageMarker(group.name))
		if len(group.values) == 0 {
			b.WriteString("none\n")
			continue
		}
		for _, delta := range group.values {
			fmt.Fprintf(&b, "%s  %s -> %s\n", delta.Identity, emptyFallback(delta.BeforeDigest, "absent"), emptyFallback(delta.AfterDigest, "absent"))
		}
	}
	b.WriteString("\nState effect: none\n")
	b.WriteString(renderNextUp(result.NextAction))
	return b.String()
}

func init() {
	skillIsUserCreatedCmd.Flags().String("skill", "", "Skill name (required)")
	maintenanceSkillsInspectCmd.Flags().String("root", "", "Repository root to scan (default: current workspace)")
	maintenanceSkillsInspectCmd.Flags().String("hub", "", "Aether hub root to scan (default: active channel hub)")
	maintenanceSkillsDiffCmd.Flags().String("before", "", "Earlier live-scan receipt or skill manifest")
	maintenanceSkillsDiffCmd.Flags().String("after", "", "Later live-scan receipt or skill manifest (default: scan live)")
	maintenanceSkillsDiffCmd.Flags().String("root", "", "Repository root for a live comparison")
	maintenanceSkillsDiffCmd.Flags().String("hub", "", "Aether hub root for a live comparison")
	maintenanceSkillsCmd.AddCommand(maintenanceSkillsInspectCmd, maintenanceSkillsDiffCmd)
	maintenanceCmd.AddCommand(maintenanceSkillsCmd)

	rootCmd.AddCommand(skillIndexReadCmd)
	rootCmd.AddCommand(skillManifestReadCmd)
	rootCmd.AddCommand(skillIsUserCreatedCmd)
}
