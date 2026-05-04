package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type referenceMeta struct {
	SchemaVersion    string   `yaml:"schema_version" json:"schema_version,omitempty"`
	ID               string   `yaml:"id" json:"id"`
	Kind             string   `yaml:"kind" json:"kind"`
	Category         string   `yaml:"category" json:"category"`
	Title            string   `yaml:"title" json:"title"`
	Description      string   `yaml:"description" json:"description,omitempty"`
	OutputTypes      []string `yaml:"output_types" json:"output_types,omitempty"`
	AgentRoles       []string `yaml:"agent_roles" json:"agent_roles,omitempty"`
	TaskTypes        []string `yaml:"task_types" json:"task_types,omitempty"`
	TaskKeywords     []string `yaml:"task_keywords" json:"task_keywords,omitempty"`
	WorkflowTriggers []string `yaml:"workflow_triggers" json:"workflow_triggers,omitempty"`
	Priority         string   `yaml:"priority" json:"priority,omitempty"`
	Version          string   `yaml:"version" json:"version,omitempty"`
	Render           struct {
		Mode     string `yaml:"mode" json:"mode,omitempty"`
		MaxChars int    `yaml:"max_chars" json:"max_chars,omitempty"`
	} `yaml:"render" json:"render,omitempty"`
}

type referenceDocument struct {
	Meta    referenceMeta `json:"meta"`
	Path    string        `json:"path"`
	RelPath string        `json:"rel_path"`
	Body    string        `json:"-"`
	Score   int           `json:"score,omitempty"`
	Reasons []string      `json:"reasons,omitempty"`
}

type referenceMatchRequest struct {
	Role       string
	Task       string
	Workflow   string
	OutputType string
	Limit      int
}

var (
	referenceListCategory  string
	referenceListKind      string
	referenceMatchRole     string
	referenceMatchTask     string
	referenceMatchWorkflow string
	referenceMatchOutput   string
	referenceMatchLimit    int
)

var referenceIndexCmd = &cobra.Command{
	Use:   "reference-index",
	Short: "Build the global reference library index",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		refs, root := loadReferenceLibrary()
		outputOK(map[string]interface{}{
			"root":       root,
			"total":      len(refs),
			"categories": referenceCategoryCounts(refs),
			"references": referenceSummaries(refs),
		})
		return nil
	},
}

var referenceListCmd = &cobra.Command{
	Use:   "reference-list",
	Short: "List installed global references",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		refs, root := loadReferenceLibrary()
		refs = filterReferences(refs, referenceListCategory, referenceListKind)
		outputOK(map[string]interface{}{
			"root":       root,
			"total":      len(refs),
			"references": referenceSummaries(refs),
		})
		return nil
	},
}

var referenceMatchCmd = &cobra.Command{
	Use:   "reference-match [task]",
	Short: "Match global references to a worker role, task, and optional output type",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		task := referenceMatchTask
		if strings.TrimSpace(task) == "" && len(args) > 0 {
			task = strings.Join(args, " ")
		}
		matches, root := matchReferences(referenceMatchRequest{
			Role:       referenceMatchRole,
			Task:       task,
			Workflow:   referenceMatchWorkflow,
			OutputType: referenceMatchOutput,
			Limit:      referenceMatchLimit,
		})
		outputOK(map[string]interface{}{
			"root":        root,
			"total":       len(matches),
			"role":        referenceMatchRole,
			"task":        task,
			"workflow":    referenceMatchWorkflow,
			"output_type": referenceMatchOutput,
			"references":  referenceSummaries(matches),
		})
		return nil
	},
}

func init() {
	referenceListCmd.Flags().StringVar(&referenceListCategory, "category", "", "Filter by reference category")
	referenceListCmd.Flags().StringVar(&referenceListKind, "kind", "", "Filter by reference kind")
	referenceMatchCmd.Flags().StringVar(&referenceMatchRole, "role", "", "Worker role or caste")
	referenceMatchCmd.Flags().StringVar(&referenceMatchTask, "task", "", "Task text to match")
	referenceMatchCmd.Flags().StringVar(&referenceMatchWorkflow, "workflow", "", "Workflow trigger such as plan, build, continue, or oracle")
	referenceMatchCmd.Flags().StringVar(&referenceMatchOutput, "output-type", "", "Desired output type such as prd, code-review, or quality-gate")
	referenceMatchCmd.Flags().IntVar(&referenceMatchLimit, "limit", 5, "Maximum references to return")

	rootCmd.AddCommand(referenceIndexCmd)
	rootCmd.AddCommand(referenceListCmd)
	rootCmd.AddCommand(referenceMatchCmd)
}

// resolveReferenceSection returns a markdown section for relevant global
// references. It is intentionally read-only; target repos do not own references.
func resolveReferenceSection(caste, task, workflow string) string {
	matches, _ := matchReferences(referenceMatchRequest{
		Role:     caste,
		Task:     task,
		Workflow: workflow,
		Limit:    2,
	})
	if len(matches) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("## Reference Library\n\n")
	b.WriteString("Use these global Aether references for output shape and quality bars.\n")
	for _, ref := range matches {
		title := ref.Meta.Title
		if title == "" {
			title = ref.Meta.ID
		}
		fmt.Fprintf(&b, "\n### %s\n\n", title)
		body := truncateReferenceBody(ref.Body, referenceRenderLimit(ref))
		if body != "" {
			b.WriteString(body)
			if !strings.HasSuffix(body, "\n") {
				b.WriteString("\n")
			}
		}
	}
	return strings.TrimSpace(b.String())
}

// appendMarkdownSections appends additional markdown sections to the base section.
func appendMarkdownSections(base, additional string) string {
	if additional == "" {
		return base
	}
	if base == "" {
		return additional
	}
	return base + "\n" + additional
}

func matchReferences(req referenceMatchRequest) ([]referenceDocument, string) {
	refs, root := loadReferenceLibrary()
	for i := range refs {
		refs[i].Score, refs[i].Reasons = scoreReference(refs[i], req)
	}
	filtered := refs[:0]
	for _, ref := range refs {
		if ref.Score > 0 {
			filtered = append(filtered, ref)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].Score != filtered[j].Score {
			return filtered[i].Score > filtered[j].Score
		}
		if priorityWeight(filtered[i].Meta.Priority) != priorityWeight(filtered[j].Meta.Priority) {
			return priorityWeight(filtered[i].Meta.Priority) > priorityWeight(filtered[j].Meta.Priority)
		}
		return filtered[i].Meta.ID < filtered[j].Meta.ID
	})
	limit := req.Limit
	if limit <= 0 {
		limit = 5
	}
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered, root
}

func loadReferenceLibrary() ([]referenceDocument, string) {
	for _, root := range referenceLibraryRoots() {
		refs := readReferencesFromRoot(root)
		if len(refs) > 0 {
			return refs, root
		}
	}
	return nil, ""
}

func referenceLibraryRoots() []string {
	var roots []string
	workspace := skillWorkspaceRoot()
	if strings.TrimSpace(workspace) != "" {
		roots = append(roots, filepath.Join(workspace, ".aether", "references"))
	}
	hub := resolveHubPath()
	if hub != "" {
		roots = append(roots, filepath.Join(hub, "references"))
		roots = append(roots, filepath.Join(hub, "system", "references"))
	}

	seen := map[string]bool{}
	deduped := make([]string, 0, len(roots))
	for _, root := range roots {
		if root == "" || seen[root] {
			continue
		}
		seen[root] = true
		deduped = append(deduped, root)
	}
	return deduped
}

func readReferencesFromRoot(root string) []referenceDocument {
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil
	}

	var refs []referenceDocument
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		ref, err := parseReferenceFile(root, path)
		if err == nil && ref.Meta.ID != "" {
			refs = append(refs, ref)
		}
		return nil
	})
	sort.Slice(refs, func(i, j int) bool {
		return refs[i].RelPath < refs[j].RelPath
	})
	return refs
}

func parseReferenceFile(root, path string) (referenceDocument, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return referenceDocument{}, err
	}
	frontmatter, body, ok := splitReferenceFrontmatter(string(data))
	if !ok {
		return referenceDocument{}, fmt.Errorf("missing frontmatter")
	}
	var meta referenceMeta
	if err := yaml.Unmarshal([]byte(frontmatter), &meta); err != nil {
		return referenceDocument{}, err
	}
	if meta.ID == "" {
		meta.ID = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	if meta.Category == "" {
		meta.Category = filepath.Base(filepath.Dir(path))
	}
	rel, _ := filepath.Rel(root, path)
	return referenceDocument{
		Meta:    meta,
		Path:    path,
		RelPath: filepath.ToSlash(rel),
		Body:    strings.TrimSpace(body),
	}, nil
}

func splitReferenceFrontmatter(content string) (string, string, bool) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	if !strings.HasPrefix(content, "---\n") {
		return "", content, false
	}
	rest := strings.TrimPrefix(content, "---\n")
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return "", content, false
	}
	frontmatter := rest[:idx]
	body := rest[idx+len("\n---"):]
	body = strings.TrimPrefix(body, "\n")
	return frontmatter, body, true
}

func scoreReference(ref referenceDocument, req referenceMatchRequest) (int, []string) {
	score := 0
	var reasons []string

	if tokenListContains(ref.Meta.OutputTypes, req.OutputType) {
		score += 4
		reasons = append(reasons, "output_type")
	}
	if tokenListContains(ref.Meta.AgentRoles, req.Role) {
		score += 3
		reasons = append(reasons, "role")
	}
	if tokenListContains(ref.Meta.WorkflowTriggers, req.Workflow) {
		score++
		reasons = append(reasons, "workflow")
	}
	if referenceTaskMatches(ref, req.Task) {
		score += 2
		reasons = append(reasons, "task")
	}
	return score, reasons
}

func referenceTaskMatches(ref referenceDocument, task string) bool {
	taskNorm := normalizeReferenceToken(task)
	if taskNorm == "" {
		return false
	}
	candidates := append([]string{}, ref.Meta.TaskTypes...)
	candidates = append(candidates, ref.Meta.TaskKeywords...)
	candidates = append(candidates, ref.Meta.Title, ref.Meta.Description, ref.Meta.ID)
	for _, candidate := range candidates {
		candidateNorm := normalizeReferenceToken(candidate)
		if candidateNorm == "" {
			continue
		}
		if strings.Contains(taskNorm, candidateNorm) || strings.Contains(candidateNorm, taskNorm) {
			return true
		}
	}
	return false
}

func tokenListContains(values []string, want string) bool {
	want = normalizeReferenceToken(want)
	if want == "" {
		return false
	}
	for _, value := range values {
		if normalizeReferenceToken(value) == want {
			return true
		}
	}
	return false
}

func normalizeReferenceToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "")
	value = strings.ReplaceAll(value, "-", "")
	value = strings.ReplaceAll(value, " ", "")
	return value
}

func priorityWeight(priority string) int {
	switch strings.ToLower(strings.TrimSpace(priority)) {
	case "critical":
		return 4
	case "high":
		return 3
	case "normal":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

func referenceRenderLimit(ref referenceDocument) int {
	if ref.Meta.Render.MaxChars > 0 {
		return ref.Meta.Render.MaxChars
	}
	return 3200
}

func truncateReferenceBody(body string, limit int) string {
	body = strings.TrimSpace(body)
	if limit <= 0 || len(body) <= limit {
		return body
	}
	return strings.TrimSpace(body[:limit]) + "\n\n..."
}

func filterReferences(refs []referenceDocument, category, kind string) []referenceDocument {
	category = normalizeReferenceToken(category)
	kind = normalizeReferenceToken(kind)
	if category == "" && kind == "" {
		return refs
	}
	filtered := make([]referenceDocument, 0, len(refs))
	for _, ref := range refs {
		if category != "" && normalizeReferenceToken(ref.Meta.Category) != category {
			continue
		}
		if kind != "" && normalizeReferenceToken(ref.Meta.Kind) != kind {
			continue
		}
		filtered = append(filtered, ref)
	}
	return filtered
}

func referenceCategoryCounts(refs []referenceDocument) map[string]int {
	counts := map[string]int{}
	for _, ref := range refs {
		category := ref.Meta.Category
		if category == "" {
			category = "uncategorized"
		}
		counts[category]++
	}
	return counts
}

func referenceSummaries(refs []referenceDocument) []map[string]interface{} {
	summaries := make([]map[string]interface{}, 0, len(refs))
	for _, ref := range refs {
		entry := map[string]interface{}{
			"id":          ref.Meta.ID,
			"kind":        ref.Meta.Kind,
			"category":    ref.Meta.Category,
			"title":       ref.Meta.Title,
			"description": ref.Meta.Description,
			"path":        ref.RelPath,
			"priority":    ref.Meta.Priority,
		}
		if ref.Score > 0 {
			entry["score"] = ref.Score
			entry["reasons"] = ref.Reasons
		}
		summaries = append(summaries, entry)
	}
	return summaries
}
