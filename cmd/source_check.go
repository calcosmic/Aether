package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

type sourceCheckIssue struct {
	Area     string `json:"area"`
	Path     string `json:"path"`
	Message  string `json:"message"`
	Expected string `json:"expected,omitempty"`
	Actual   string `json:"actual,omitempty"`
}

type sourceCheckComponent struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Checked int    `json:"checked"`
	Message string `json:"message"`
}

type sourceCheckResult struct {
	OK         bool                   `json:"ok"`
	Root       string                 `json:"root"`
	Components []sourceCheckComponent `json:"components"`
	Issues     []sourceCheckIssue     `json:"issues,omitempty"`
	Next       string                 `json:"next"`
}

var sourceCheckGeneratedHeader = regexp.MustCompile(`^<!-- Generated from (\.aether/commands/[^ ]+\.yaml) - DO NOT EDIT DIRECTLY -->$`)

var sourceCheckRequiredExchangeXMLAssets = []string{
	"colony-archive.xml",
	"colony-registry.xml",
	"pheromones.xml",
	"queen-wisdom.xml",
}

var sourceCheckCmd = &cobra.Command{
	Use:   "source-check",
	Short: "Verify source-of-truth and generated wrapper parity",
	Long: "Checks Aether's source-of-truth layout without modifying files. " +
		"It verifies generated command wrappers before publish.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		rootFlag, _ := cmd.Flags().GetString("root")
		root, err := resolveSourceCheckRoot(rootFlag)
		if err != nil {
			return err
		}
		result := runSourceCheck(root)
		if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
			data, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal source-check result: %w", err)
			}
			fmt.Fprintln(stdout, string(data))
		} else {
			outputWorkflow(result, renderSourceCheckVisual(result))
		}
		if !result.OK {
			return fmt.Errorf("source check failed")
		}
		return nil
	},
}

func init() {
	sourceCheckCmd.Flags().String("root", "", "Aether source checkout root (default: auto-detect from current directory)")
	sourceCheckCmd.Flags().Bool("json", false, "Output JSON instead of visual report")
	rootCmd.AddCommand(sourceCheckCmd)
}

func resolveSourceCheckRoot(explicit string) (string, error) {
	if explicit != "" {
		abs, err := filepath.Abs(explicit)
		if err != nil {
			return "", fmt.Errorf("resolve source root %q: %w", explicit, err)
		}
		if !looksLikeAetherSourceRoot(abs) {
			return "", fmt.Errorf("%s does not look like an Aether source checkout", abs)
		}
		return abs, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("determine current directory: %w", err)
	}
	root := findAetherModuleRoot(cwd)
	if root == "" {
		root = cwd
	}
	if !looksLikeAetherSourceRoot(root) {
		return "", fmt.Errorf("%s does not look like an Aether source checkout", root)
	}
	return root, nil
}

func looksLikeAetherSourceRoot(root string) bool {
	required := []string{
		filepath.Join(root, ".aether", "commands"),
		filepath.Join(root, ".aether", "skills"),
		filepath.Join(root, ".claude", "agents", "ant"),
		filepath.Join(root, ".codex", "agents"),
		filepath.Join(root, ".opencode", "agents"),
	}
	for _, path := range required {
		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			return false
		}
	}
	return true
}

func runSourceCheck(root string) sourceCheckResult {
	result := sourceCheckResult{
		Root: root,
		Next: "Fix reported source surface drift, then rerun `aether source-check` before publishing.",
	}

	sourceChecked, sourceIssues := checkCanonicalSourceSurfaces(root)
	result.Components = append(result.Components, sourceCheckComponent{
		Name:    "canonical source surfaces",
		Status:  sourceCheckStatus(sourceIssues),
		Checked: sourceChecked,
		Message: sourceCheckMessage(sourceIssues, "canonical source files are present in the Aether repo"),
	})
	result.Issues = append(result.Issues, sourceIssues...)

	retiredChecked, retiredIssues := checkRetiredSourceMirrors(root)
	result.Components = append(result.Components, sourceCheckComponent{
		Name:    "retired source mirrors",
		Status:  sourceCheckStatus(retiredIssues),
		Checked: retiredChecked,
		Message: sourceCheckMessage(retiredIssues, "deleted packaging mirrors are absent"),
	})
	result.Issues = append(result.Issues, retiredIssues...)

	commandChecked, commandIssues := checkGeneratedCommandSurfaces(root)
	result.Components = append(result.Components, sourceCheckComponent{
		Name:    "generated command wrappers",
		Status:  sourceCheckStatus(commandIssues),
		Checked: commandChecked,
		Message: sourceCheckMessage(commandIssues, "all generated command wrappers match their YAML-owned surfaces"),
	})
	result.Issues = append(result.Issues, commandIssues...)

	sortSourceCheckIssues(result.Issues)
	result.OK = len(result.Issues) == 0
	if result.OK {
		result.Next = "Source surfaces are aligned. Publish only after active work is intentionally committed."
	}
	return result
}

func checkCanonicalSourceSurfaces(root string) (int, []sourceCheckIssue) {
	type sourceSurface struct {
		rel  string
		kind string
	}
	required := []sourceSurface{
		{".aether/commands", "dir"},
		{".aether/skills", "dir"},
		{".aether/templates", "dir"},
		{".aether/docs", "dir"},
		{".aether/utils", "dir"},
		{".aether/exchange", "dir"},
		{".aether/workers.md", "file"},
		{".claude/agents/ant", "dir"},
		{".claude/commands/ant", "dir"},
		{".opencode/agents", "dir"},
		{".opencode/commands/ant", "dir"},
		{".codex/agents", "dir"},
	}

	var issues []sourceCheckIssue
	for _, surface := range required {
		path := filepath.Join(root, filepath.FromSlash(surface.rel))
		info, err := os.Stat(path)
		if err != nil {
			issues = append(issues, sourceCheckIssue{
				Area:     "sources",
				Path:     surface.rel,
				Message:  "canonical source surface is missing",
				Expected: surface.kind,
				Actual:   "missing",
			})
			continue
		}
		if surface.kind == "dir" && !info.IsDir() {
			issues = append(issues, sourceCheckIssue{
				Area:     "sources",
				Path:     surface.rel,
				Message:  "canonical source surface should be a directory",
				Expected: "dir",
				Actual:   "file",
			})
		}
		if surface.kind == "file" && info.IsDir() {
			issues = append(issues, sourceCheckIssue{
				Area:     "sources",
				Path:     surface.rel,
				Message:  "canonical source surface should be a file",
				Expected: "file",
				Actual:   "dir",
			})
		}
	}

	checked := len(required)
	for _, name := range sourceCheckRequiredExchangeXMLAssets {
		checked++
		rel := filepath.ToSlash(filepath.Join(".aether", "exchange", name))
		path := filepath.Join(root, filepath.FromSlash(rel))
		info, err := os.Stat(path)
		if err != nil {
			issues = append(issues, sourceCheckIssue{
				Area:     "sources",
				Path:     rel,
				Message:  "required exchange XML asset is missing",
				Expected: "file",
				Actual:   "missing",
			})
			continue
		}
		if info.IsDir() {
			issues = append(issues, sourceCheckIssue{
				Area:     "sources",
				Path:     rel,
				Message:  "required exchange XML asset should be a file",
				Expected: "file",
				Actual:   "dir",
			})
		}
	}

	return checked, issues
}

func checkRetiredSourceMirrors(root string) (int, []sourceCheckIssue) {
	retired := []string{
		".aether/agents-claude",
		".aether/agents-codex",
		".aether/commands/claude",
		".aether/commands/opencode",
		".aether/skills-codex",
	}

	var issues []sourceCheckIssue
	for _, rel := range retired {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if _, err := os.Stat(path); err == nil {
			issues = append(issues, sourceCheckIssue{
				Area:     "sources",
				Path:     rel,
				Message:  "retired packaging mirror exists; edit canonical platform sources instead",
				Expected: "absent",
				Actual:   "present",
			})
		}
	}
	return len(retired), issues
}

func checkGeneratedCommandSurfaces(root string) (int, []sourceCheckIssue) {
	yamlDir := filepath.Join(root, ".aether", "commands")
	yamlNames := map[string]string{}
	for _, rel := range sourceCheckFiles(root, ".aether/commands", func(rel string) bool {
		return !strings.Contains(filepath.ToSlash(rel), "/") && filepath.Ext(rel) == ".yaml"
	}) {
		name := strings.TrimSuffix(filepath.Base(rel), ".yaml")
		yamlNames[name] = filepath.ToSlash(filepath.Join(".aether", "commands", rel))
	}

	var issues []sourceCheckIssue
	wrapperDirs := []string{
		".claude/commands/ant",
		".opencode/commands/ant",
	}
	checked := 0
	for _, wrapperDir := range wrapperDirs {
		for name, yamlRel := range yamlNames {
			wrapperRel := filepath.ToSlash(filepath.Join(wrapperDir, name+".md"))
			wrapperPath := filepath.Join(root, filepath.FromSlash(wrapperRel))
			data, err := os.ReadFile(wrapperPath)
			if err != nil {
				issues = append(issues, sourceCheckIssue{
					Area:     "commands",
					Path:     wrapperRel,
					Message:  "generated wrapper missing for YAML source",
					Expected: yamlRel,
					Actual:   "missing",
				})
				continue
			}
			checked++
			firstLine := strings.SplitN(string(data), "\n", 2)[0]
			matches := sourceCheckGeneratedHeader.FindStringSubmatch(firstLine)
			if matches == nil {
				issues = append(issues, sourceCheckIssue{
					Area:     "commands",
					Path:     wrapperRel,
					Message:  "generated wrapper is missing the generated-from header",
					Expected: "<!-- Generated from .aether/commands/<name>.yaml - DO NOT EDIT DIRECTLY -->",
					Actual:   firstLine,
				})
				continue
			}
			if matches[1] != yamlRel {
				issues = append(issues, sourceCheckIssue{
					Area:     "commands",
					Path:     wrapperRel,
					Message:  "generated wrapper header points at the wrong YAML source",
					Expected: yamlRel,
					Actual:   matches[1],
				})
			}
		}

		for _, rel := range sourceCheckFiles(root, wrapperDir, func(rel string) bool {
			return !strings.Contains(filepath.ToSlash(rel), "/") && filepath.Ext(rel) == ".md"
		}) {
			name := strings.TrimSuffix(filepath.Base(rel), ".md")
			if _, ok := yamlNames[name]; !ok {
				issues = append(issues, sourceCheckIssue{
					Area:    "commands",
					Path:    filepath.ToSlash(filepath.Join(wrapperDir, rel)),
					Message: "generated wrapper has no matching YAML source",
					Actual:  filepath.Join(yamlDir, name+".yaml"),
				})
			}
		}
	}

	return checked, issues
}

func sourceCheckFiles(root, relDir string, include func(string) bool) []string {
	base := filepath.Join(root, filepath.FromSlash(relDir))
	var files []string
	_ = filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(base, path)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if include == nil || include(rel) {
			files = append(files, rel)
		}
		return nil
	})
	sort.Strings(files)
	return files
}

func sourceCheckStatus(issues []sourceCheckIssue) string {
	if len(issues) == 0 {
		return "pass"
	}
	return "fail"
}

func sourceCheckMessage(issues []sourceCheckIssue, ok string) string {
	if len(issues) == 0 {
		return ok
	}
	return fmt.Sprintf("%d issue(s) found", len(issues))
}

func sortSourceCheckIssues(issues []sourceCheckIssue) {
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Area != issues[j].Area {
			return issues[i].Area < issues[j].Area
		}
		return issues[i].Path < issues[j].Path
	})
}

func renderSourceCheckVisual(result sourceCheckResult) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("source-check"), "Source Check"))
	b.WriteString(visualDivider)
	b.WriteString("Root: ")
	b.WriteString(result.Root)
	b.WriteString("\n\n")

	for _, component := range result.Components {
		marker := "✓"
		if component.Status != "pass" {
			marker = "✗"
		}
		b.WriteString(fmt.Sprintf("%s %s: %s (%d checked)\n", marker, component.Name, component.Status, component.Checked))
		if strings.TrimSpace(component.Message) != "" {
			b.WriteString("  ")
			b.WriteString(component.Message)
			b.WriteString("\n")
		}
	}

	if len(result.Issues) > 0 {
		b.WriteString("\n")
		b.WriteString(renderStageMarker("Issues"))
		limit := len(result.Issues)
		if limit > 20 {
			limit = 20
		}
		for _, issue := range result.Issues[:limit] {
			b.WriteString(fmt.Sprintf("- %s: %s\n", issue.Path, issue.Message))
		}
		if len(result.Issues) > limit {
			b.WriteString(fmt.Sprintf("... %d more issue(s); rerun with `AETHER_OUTPUT_MODE=json` for full details.\n", len(result.Issues)-limit))
		}
	}

	b.WriteString("\n")
	if result.OK {
		b.WriteString(renderNextUp(result.Next))
	} else {
		b.WriteString(renderNextUp(result.Next, "Use Aether repo source files as the authority: YAML for wrapper specs, platform source dirs for agents, and .aether/skills for shipped skills. Publish/install populates the global hub and platform homes; target repos keep only local state."))
	}
	return b.String()
}
