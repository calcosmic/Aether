package cmd

import (
	"crypto/sha256"
	"encoding/hex"
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
	Scope            string   `yaml:"scope" json:"scope,omitempty"`
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
		result := map[string]interface{}{
			"root":       root,
			"total":      len(refs),
			"categories": referenceCategoryCounts(refs),
			"references": referenceSummaries(refs),
		}
		outputWorkflow(result, renderReferenceIndexVisual(result))
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
		result := map[string]interface{}{
			"root":       root,
			"total":      len(refs),
			"references": referenceSummaries(refs),
		}
		outputWorkflow(result, renderReferenceListVisual(result))
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
		result := map[string]interface{}{
			"root":        root,
			"total":       len(matches),
			"role":        referenceMatchRole,
			"task":        task,
			"workflow":    referenceMatchWorkflow,
			"output_type": referenceMatchOutput,
			"references":  referenceSummaries(matches),
		}
		outputWorkflow(result, renderReferenceMatchVisual(result))
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
	return resolveReferenceSectionWithOutput(caste, task, workflow, "")
}

func resolveReferenceSectionWithOutput(caste, task, workflow, outputType string) string {
	matches, _ := matchReferences(referenceMatchRequest{
		Role:       caste,
		Task:       task,
		Workflow:   workflow,
		OutputType: outputType,
		Limit:      2,
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
	allowInternal := referenceRootIsAetherSource(root)
	for i := range refs {
		refs[i].Score, refs[i].Reasons = scoreReference(refs[i], req)
	}
	filtered := refs[:0]
	for _, ref := range refs {
		if ref.Score <= 0 {
			continue
		}
		if referenceIsAetherInternal(ref) && !allowInternal {
			continue
		}
		// Task relevance is required, not merely rewarded. A document that
		// matches only the worker's job title, the workflow, or the expected
		// output type says nothing about the work in hand, and a worker with no
		// task-relevant reading is better served by an empty section than by a
		// filler selection it still pays for.
		if !referenceHasReason(ref, "task") {
			continue
		}
		filtered = append(filtered, ref)
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].Score != filtered[j].Score {
			return filtered[i].Score > filtered[j].Score
		}
		if priorityWeight(filtered[i].Meta.Priority) != priorityWeight(filtered[j].Meta.Priority) {
			return priorityWeight(filtered[i].Meta.Priority) > priorityWeight(filtered[j].Meta.Priority)
		}
		// Ties break on content, never on identifier. Sorting by ID made
		// alphabetical position a selection input, so a document could win a
		// slot by being renamed.
		return referenceTieBreak(filtered[i]) < referenceTieBreak(filtered[j])
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

// referenceScopeAetherInternal marks a reference that describes how Aether
// itself is built -- its Go/TypeScript boundary, its wrapper chain, its hub
// layout, how it composes worker prompts. These are useful to a worker changing
// Aether and are noise everywhere else.
//
// The cost of getting this wrong is not theoretical. A worker in an Obsidian
// notes vault, asked to create twelve folders, was handed the Runtime Boundary
// Contract and spent 51,914 tokens on the job. References are read from the
// shared hub -- they are never copied into downstream repos -- so one
// mis-scoped document is charged to every project on the machine.
const referenceScopeAetherInternal = "aether-internal"

func referenceHasReason(ref referenceDocument, want string) bool {
	for _, reason := range ref.Reasons {
		if reason == want {
			return true
		}
	}
	return false
}

// referenceTieBreak orders equally-scored documents by their content rather than
// their name, so renaming a file cannot change what a worker receives.
func referenceTieBreak(ref referenceDocument) string {
	sum := sha256.Sum256([]byte(ref.Meta.Title + "\x00" + ref.Body))
	return hex.EncodeToString(sum[:8])
}

func referenceIsAetherInternal(ref referenceDocument) bool {
	return strings.EqualFold(strings.TrimSpace(ref.Meta.Scope), referenceScopeAetherInternal)
}

// referenceRootIsAetherSource reports whether the loaded library is Aether's own
// authored copy rather than the shared hub copy.
//
// The check is the directory the library came from, not the repository's name or
// module path: `.aether/references` is where the library is authored, and
// `aether update` deliberately leaves references global instead of copying them
// into each repo (see update_cmd.go). So a workspace holding that directory is
// the source repo, and any other workspace resolves to the hub.
func referenceRootIsAetherSource(root string) bool {
	root = strings.TrimSpace(root)
	if root == "" {
		return false
	}
	workspace := strings.TrimSpace(skillWorkspaceRoot())
	if workspace == "" {
		return false
	}
	source := filepath.Join(workspace, ".aether", "references")
	if root == source {
		return true
	}
	// Compare resolved paths so a symlinked or relative workspace still matches.
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return false
	}
	resolvedSource, err := filepath.EvalSymlinks(source)
	if err != nil {
		return false
	}
	return resolvedRoot == resolvedSource
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

// Reference scoring weights. Task relevance is deliberately worth more than
// every other signal combined.
//
// It used to be the lowest: output type 4, job title 3, workflow 1, task 2. A
// document therefore needed no connection to the work to be selected, and a
// builder copying markdown files in a notes vault was handed the playbook for
// publishing Aether releases. Task relevance is also now a requirement, not a
// bonus -- see matchReferences.
const (
	referenceScoreTask       = 6
	referenceScoreOutputType = 3
	referenceScoreRole       = 2
	referenceScoreWorkflow   = 1
)

func scoreReference(ref referenceDocument, req referenceMatchRequest) (int, []string) {
	score := 0
	var reasons []string

	if referenceTaskMatches(ref, req.Task) {
		score += referenceScoreTask
		reasons = append(reasons, "task")
	}
	if tokenListContains(ref.Meta.OutputTypes, req.OutputType) {
		score += referenceScoreOutputType
		reasons = append(reasons, "output_type")
	}
	if tokenListContains(ref.Meta.AgentRoles, req.Role) {
		score += referenceScoreRole
		reasons = append(reasons, "role")
	}
	if tokenListContains(ref.Meta.WorkflowTriggers, req.Workflow) {
		score += referenceScoreWorkflow
		reasons = append(reasons, "workflow")
	}
	return score, reasons
}

// referenceTaskMatches compares a document's declared task vocabulary against
// the words of the actual task.
//
// The previous implementation stripped every space from both the task and the
// candidate and then asked whether either contained the other. "copy 110
// markdown files into a new folder tree" became one 46-character word, so
// "folder" matched "old" and short keywords matched at random. Whole-word
// comparison replaces it, and only the declared task vocabulary is consulted --
// title, description and identifier are no longer matchable, because matching on
// the identifier makes the filename a selection input.
func referenceTaskMatches(ref referenceDocument, task string) bool {
	tokens := referenceTaskTokens(task)
	if len(tokens) == 0 {
		return false
	}
	candidates := append([]string{}, ref.Meta.TaskTypes...)
	candidates = append(candidates, ref.Meta.TaskKeywords...)
	for _, candidate := range candidates {
		candidate = strings.ToLower(strings.TrimSpace(candidate))
		if candidate == "" {
			continue
		}
		// A multi-word keyword must appear as a phrase.
		if strings.ContainsAny(candidate, " -_") {
			if referenceTaskPhraseMatches(tokens, candidate) {
				return true
			}
			continue
		}
		for token := range tokens {
			if referenceTokensAlike(token, candidate) {
				return true
			}
		}
	}
	return false
}

func referenceTaskTokens(task string) map[string]bool {
	tokens := map[string]bool{}
	for _, field := range strings.FieldsFunc(strings.ToLower(task), func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
	}) {
		if len(field) < 2 {
			continue
		}
		tokens[field] = true
	}
	return tokens
}

func referenceTaskPhraseMatches(tokens map[string]bool, candidate string) bool {
	parts := strings.FieldsFunc(candidate, func(r rune) bool {
		return r == ' ' || r == '-' || r == '_'
	})
	if len(parts) == 0 {
		return false
	}
	for _, part := range parts {
		if len(part) < 2 {
			continue
		}
		matched := false
		for token := range tokens {
			if referenceTokensAlike(token, part) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

// referenceTokensAlike treats a word and its simple plural as the same word, so
// a document declaring "file" matches a task that says "files". It deliberately
// does no other stemming: aggressive stemming is how the previous matcher
// became noise.
func referenceTokensAlike(token, candidate string) bool {
	if token == candidate {
		return true
	}
	for _, pair := range [][2]string{{token, candidate}, {candidate, token}} {
		long, short := pair[0], pair[1]
		if len(short) < 3 || len(long) <= len(short) {
			continue
		}
		switch long[len(short):] {
		case "s", "es":
			if long[:len(short)] == short {
				return true
			}
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
		// Surfaced so an operator can see why a document is or is not in a
		// worker's reading list without opening the file.
		if scope := strings.TrimSpace(ref.Meta.Scope); scope != "" {
			entry["scope"] = scope
		}
		if ref.Score > 0 {
			entry["score"] = ref.Score
			entry["reasons"] = ref.Reasons
		}
		summaries = append(summaries, entry)
	}
	return summaries
}
