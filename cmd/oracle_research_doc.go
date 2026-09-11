package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/spf13/cobra"
)

var researchCmd = &cobra.Command{
	Use:   "research [list]",
	Short: "Browse research documents saved from completed Oracle runs",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := runResearchList(skillWorkspaceRoot())
		if err != nil {
			outputError(1, err.Error(), nil)
			return renderedErrorExit(1)
		}
		outputWorkflow(result, renderResearchList(result))
		return nil
	},
}

// Durable research documents.
//
// Oracle's write-up lived only in .aether/oracle/synthesis.md, which nothing
// read and which the *next* run destroyed: archiveOracleWorkspace sweeps the
// whole workspace into archive/<timestamp>/ before a new topic starts. So the
// deliverable of a long research run survived only until the next question was
// asked.
//
// .aether/research/ is the durable home. It is not gitignored (so it is
// browsable and committable), it is not in managedAetherSystemDirs (so
// `aether update` never sweeps it), and it is outside every data-reset path.

const (
	oracleResearchDirName = "research"
	oracleResearchSlugMax = 60
)

func oracleResearchDir(root string) string {
	return filepath.Join(root, ".aether", oracleResearchDirName)
}

// oracleResearchSlug turns a topic into a filename component that still tells
// the operator what the document is about.
func oracleResearchSlug(text string) string {
	var b strings.Builder
	lastDash := true
	for _, r := range strings.ToLower(strings.TrimSpace(text)) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastDash = false
		case !lastDash:
			b.WriteByte('-')
			lastDash = true
		}
		if b.Len() >= oracleResearchSlugMax {
			break
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "research"
	}
	return slug
}

// renderOracleResearchDocument prefixes the synthesis with front matter. The
// front matter is what lets `aether research list` tell a 92%-confidence
// document from a 35% stub, and what records the question the run set out to
// answer -- the synthesis heading alone is a 140-character truncation of the
// raw topic.
func renderOracleResearchDocument(state oracleStateFile, plan oraclePlanFile, body, sourceWorkspace string) string {
	questionCount, answeredCount, _ := oracleQuestionCounts(plan)

	title := strings.TrimSpace(state.CoreQuestion)
	if title == "" {
		title = oracleTopicHeadline(state.Topic)
	}
	if title == "" {
		title = "Oracle research"
	}

	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "title: %s\n", oracleYAMLScalar(title))
	if core := strings.TrimSpace(state.CoreQuestion); core != "" {
		fmt.Fprintf(&b, "core_question: %s\n", oracleYAMLScalar(core))
	}
	fmt.Fprintf(&b, "topic: %s\n", oracleYAMLScalar(strings.Join(strings.Fields(state.Topic), " ")))
	if len(state.SuccessCriteria) > 0 {
		b.WriteString("success_criteria:\n")
		for _, item := range state.SuccessCriteria {
			fmt.Fprintf(&b, "  - %s\n", oracleYAMLScalar(item))
		}
	}
	fmt.Fprintf(&b, "generated: %s\n", time.Now().UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "template: %s\n", emptyFallback(strings.TrimSpace(state.Template), "custom"))
	fmt.Fprintf(&b, "scope: %s\n", emptyFallback(strings.TrimSpace(state.Scope), defaultOracleScope))
	fmt.Fprintf(&b, "iterations: %d\n", state.Iteration)
	fmt.Fprintf(&b, "confidence: %d\n", state.OverallConfidence)
	fmt.Fprintf(&b, "target_confidence: %d\n", state.TargetConfidence)
	fmt.Fprintf(&b, "status: %s\n", emptyFallback(strings.TrimSpace(state.Status), "unknown"))
	if reason := strings.TrimSpace(state.StopReason); reason != "" {
		fmt.Fprintf(&b, "stop_reason: %s\n", reason)
	}
	fmt.Fprintf(&b, "questions_answered: %d\n", answeredCount)
	fmt.Fprintf(&b, "questions_total: %d\n", questionCount)
	if sourceWorkspace != "" {
		fmt.Fprintf(&b, "source_workspace: %s\n", sourceWorkspace)
	}
	b.WriteString("---\n\n")
	// The partial label lives in the body, not only in the front matter's
	// status field, because the body is what a worker actually reads --
	// stripResearchFrontMatter removes the header before the document ever
	// reaches a prompt. It is the first line so a partial answer is never
	// mistaken for a finished one (D-07).
	if label := oracleResearchPartialLabel(state); label != "" {
		b.WriteString(label)
		b.WriteString("\n\n")
	}
	b.WriteString(strings.TrimSpace(body))
	b.WriteString("\n")
	return b.String()
}

// oracleResearchPartialLabel names a plain-English partial-run warning for
// anything short of a clean completion -- an owner stop, a worker time-out,
// or the iteration cap -- so the document itself says what a status field
// buried in the front matter would not. Empty for a clean completion.
func oracleResearchPartialLabel(state oracleStateFile) string {
	if strings.TrimSpace(state.Status) == "complete" {
		return ""
	}
	rounds := state.Iteration
	roundWord := "round"
	if rounds != 1 {
		roundWord = "rounds"
	}
	return fmt.Sprintf("partial — stopped after %d %s", rounds, roundWord)
}

// oracleYAMLScalar quotes a value so a topic containing a colon cannot produce
// unparseable front matter.
func oracleYAMLScalar(value string) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
}

// saveOracleResearchDocument writes the finished research somewhere it will
// survive the next run. It returns the repo-relative path.
func saveOracleResearchDocument(paths oraclePaths, state oracleStateFile, plan oraclePlanFile, name string) (string, error) {
	body, err := os.ReadFile(paths.SynthesisPath)
	if err != nil {
		return "", fmt.Errorf("no research to save: %s is unavailable (%v)", paths.SynthesisPath, err)
	}
	if strings.TrimSpace(string(body)) == "" {
		return "", fmt.Errorf("no research to save: %s is empty", paths.SynthesisPath)
	}

	// 202-12 (LIVE-07, D-10): the durable document's body is the
	// recommendation-first synthesis, not the raw per-template write-up --
	// the existing per-template report (tech-eval/generic/etc, still written
	// to paths.SynthesisPath by writeOracleSynthesisReport) becomes the
	// document's closing evidence-trail section rather than the whole body.
	// A conclusion citing a source not in plan.Sources refuses here, naming
	// the conclusion, before anything is written to disk.
	synthesized, synthErr := renderOracleFinalSynthesis(state, plan, string(body))
	if synthErr != nil {
		return "", synthErr
	}

	slug := strings.TrimSpace(name)
	if slug == "" {
		slug = strings.TrimSpace(state.CoreQuestion)
		if slug == "" {
			slug = state.Topic
		}
	}
	filename := fmt.Sprintf("%s-%s.md", time.Now().UTC().Format("2006-01-02"), oracleResearchSlug(slug))

	dir := oracleResearchDir(paths.Root)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create research dir: %w", err)
	}
	target := filepath.Join(dir, filename)
	// Two runs on the same topic on the same day must not overwrite each other.
	for i := 2; fileExists(target); i++ {
		target = filepath.Join(dir, fmt.Sprintf("%s-%s-%d.md", time.Now().UTC().Format("2006-01-02"), oracleResearchSlug(slug), i))
	}

	sourceWorkspace := ""
	if rel, relErr := filepath.Rel(paths.Root, paths.Dir); relErr == nil {
		sourceWorkspace = rel
	}
	document := renderOracleResearchDocument(state, plan, synthesized, sourceWorkspace)
	if err := os.WriteFile(target, []byte(document), 0644); err != nil {
		return "", fmt.Errorf("write research document: %w", err)
	}

	rel, relErr := filepath.Rel(paths.Root, target)
	if relErr != nil {
		return target, nil
	}
	return rel, nil
}

// runOracleSave backs `aether oracle save`, so research from a run the operator
// stopped early is still keepable.
func runOracleSave(root, name string, dryRun bool) (map[string]interface{}, error) {
	paths := oracleWorkspacePaths(root)
	state, err := loadOracleStateFile(paths.StatePath)
	if err != nil {
		return nil, fmt.Errorf("no Oracle research to save: %s is unavailable (%v); run `aether oracle --from-brief` first", paths.StatePath, err)
	}
	plan, err := loadOraclePlanFile(paths.PlanPath)
	if err != nil {
		return nil, fmt.Errorf("no Oracle research to save: %s is unavailable (%v)", paths.PlanPath, err)
	}

	if dryRun {
		body, readErr := os.ReadFile(paths.SynthesisPath)
		if readErr != nil || strings.TrimSpace(string(body)) == "" {
			return nil, fmt.Errorf("no research to save: %s is empty or unavailable", paths.SynthesisPath)
		}
		slug := strings.TrimSpace(name)
		if slug == "" {
			slug = emptyFallback(strings.TrimSpace(state.CoreQuestion), state.Topic)
		}
		synthesized, synthErr := renderOracleFinalSynthesis(state, plan, string(body))
		if synthErr != nil {
			return nil, synthErr
		}
		preview := renderOracleResearchDocument(state, plan, synthesized, "")
		return map[string]interface{}{
			"mode":         "save",
			"dry_run":      true,
			"destination":  filepath.Join(".aether", oracleResearchDirName, fmt.Sprintf("%s-%s.md", time.Now().UTC().Format("2006-01-02"), oracleResearchSlug(slug))),
			"front_matter": strings.SplitN(preview, "---\n\n", 2)[0] + "---",
		}, nil
	}

	saved, err := saveOracleResearchDocument(paths, state, plan, name)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"mode":              "save",
		"dry_run":           false,
		"research_document": saved,
		"confidence":        state.OverallConfidence,
		"next":              fmt.Sprintf("aether init --research %s \"<goal>\"", saved),
	}, nil
}

// --- listing -----------------------------------------------------------

type oracleResearchEntry struct {
	Path         string `json:"path"`
	Title        string `json:"title"`
	CoreQuestion string `json:"core_question,omitempty"`
	Generated    string `json:"generated,omitempty"`
	Template     string `json:"template,omitempty"`
	Confidence   int    `json:"confidence"`
	Status       string `json:"status,omitempty"`
	Iterations   int    `json:"iterations"`
}

// runResearchList backs `aether research list`. Without it the directory is
// only discoverable by already knowing it exists, and pointing at a document is
// the operator's half of the handoff.
func runResearchList(root string) (map[string]interface{}, error) {
	dir := oracleResearchDir(root)
	matches, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		return nil, fmt.Errorf("list research: %w", err)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(matches)))

	entries := make([]oracleResearchEntry, 0, len(matches))
	for _, match := range matches {
		data, readErr := os.ReadFile(match)
		if readErr != nil {
			continue
		}
		rel, relErr := filepath.Rel(root, match)
		if relErr != nil {
			rel = match
		}
		entry := parseOracleResearchFrontMatter(string(data))
		entry.Path = rel
		if entry.Title == "" {
			entry.Title = strings.TrimSuffix(filepath.Base(match), ".md")
		}
		entries = append(entries, entry)
	}

	return map[string]interface{}{
		"mode":      "research-list",
		"directory": filepath.Join(".aether", oracleResearchDirName),
		"count":     len(entries),
		"documents": entries,
		"next":      "aether init --research <path> \"<goal>\"",
	}, nil
}

func parseOracleResearchFrontMatter(text string) oracleResearchEntry {
	entry := oracleResearchEntry{}
	if !strings.HasPrefix(text, "---\n") {
		return entry
	}
	end := strings.Index(text[4:], "\n---")
	if end < 0 {
		return entry
	}
	for _, line := range strings.Split(text[4:4+end], "\n") {
		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"`)
		switch key {
		case "title":
			entry.Title = value
		case "core_question":
			entry.CoreQuestion = value
		case "generated":
			entry.Generated = value
		case "template":
			entry.Template = value
		case "status":
			entry.Status = value
		case "confidence":
			entry.Confidence = atoiOrZero(value)
		case "iterations":
			entry.Iterations = atoiOrZero(value)
		}
	}
	return entry
}

func renderResearchList(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner("📚🐜", "Saved Research"))
	b.WriteString(visualDividerStr())

	entries, _ := result["documents"].([]oracleResearchEntry)
	if len(entries) == 0 {
		fmt.Fprintf(&b, "No saved research yet in %v.\n\n", result["directory"])
		b.WriteString("Finish a research run, or save the current one with `aether oracle save`.")
		return strings.TrimSpace(b.String())
	}

	for _, entry := range entries {
		fmt.Fprintf(&b, "%s\n", entry.Path)
		fmt.Fprintf(&b, "   %s\n", truncateString(emptyFallback(entry.CoreQuestion, entry.Title), 88))
		fmt.Fprintf(&b, "   %d%% confidence after %d rounds — %s\n\n", entry.Confidence, entry.Iterations, emptyFallback(entry.Status, "unknown"))
	}
	b.WriteString("Point a new colony at one with:\n  aether init --research <path> \"<goal>\"")
	return strings.TrimSpace(b.String())
}
