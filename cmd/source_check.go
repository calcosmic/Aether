package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const sourceCheckResultSchemaVersion = "source-check-result/v1"

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
	SchemaVersion string                            `json:"schema_version"`
	OperationID   string                            `json:"operation_id"`
	OK            bool                              `json:"ok"`
	Root          string                            `json:"root"`
	Components    []sourceCheckComponent            `json:"components"`
	Issues        []sourceCheckIssue                `json:"issues,omitempty"`
	Findings      []maintenanceInspectionFinding    `json:"findings"`
	Evidence      []maintenanceInspectionEvidence   `json:"evidence"`
	Verification  maintenanceInspectionVerification `json:"verification"`
	StateEffect   colony.LifecycleStateEffect       `json:"state_effect"`
	Blockers      []maintenanceInspectionFinding    `json:"blockers"`
	NextAction    string                            `json:"next_action"`
	Next          string                            `json:"next"`
}

type sourceCheckCommandSpec struct {
	Name          string   `yaml:"name"`
	Description   string   `yaml:"description"`
	SourceOfTruth string   `yaml:"source_of_truth"`
	Aliases       []string `yaml:"aliases"`
	Runtime       struct {
		Command          string `yaml:"command"`
		DefaultCommand   string `yaml:"default_command"`
		ManifestCommand  string `yaml:"manifest_command"`
		FinalizerCommand string `yaml:"finalizer_command"`
	} `yaml:"runtime"`
}

type sourceCheckWrapperFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// The header is honest now: no generator exists, and these wrappers ARE the
// edited files. The header's real job is pairing a wrapper with its YAML
// runtime spec and marking the file Aether-managed for update/prune.
var sourceCheckGeneratedHeader = regexp.MustCompile(`^<!-- Aether-managed: runtime spec at (\.aether/commands/[^ ]+\.yaml)\. Synced by aether update\. -->$`)

var sourceCheckRequiredExchangeXMLAssets = []string{
	"colony-archive.xml",
	"colony-registry.xml",
	"pheromones.xml",
	"queen-wisdom.xml",
}

var sourceCheckCmd = &cobra.Command{
	Use:         "source-check",
	Short:       "Verify source-of-truth and generated wrapper parity",
	Annotations: map[string]string{"aether.io/read-only": "true", "aether.io/store-free": "true"},
	Long: "Checks Aether's source-of-truth layout without modifying files. " +
		"It verifies generated command wrappers before publish.",
	Args: cobra.NoArgs,
	RunE: runSourceCheckCommand,
}

func init() {
	sourceCheckCmd.Flags().String("root", "", "Aether source checkout root (default: auto-detect from current directory)")
	sourceCheckCmd.Flags().Bool("json", false, "Output JSON instead of visual report")
	rootCmd.AddCommand(sourceCheckCmd)
}

func runSourceCheckCommand(cmd *cobra.Command, _ []string) error {
	rootFlag, _ := cmd.Flags().GetString("root")
	result, resolutionErr := buildSourceCheckResult(rootFlag)
	if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal source-check result: %w", err)
		}
		fmt.Fprintln(stdout, string(data))
	} else {
		outputWorkflow(result, renderSourceCheckVisual(result))
	}
	if resolutionErr != nil {
		return fmt.Errorf("source check failed: %w", resolutionErr)
	}
	if !result.OK {
		return fmt.Errorf("source check failed")
	}
	return nil
}

func buildSourceCheckResult(explicitRoot string) (sourceCheckResult, error) {
	root, err := resolveSourceCheckRoot(explicitRoot)
	if err == nil {
		return runSourceCheck(root), nil
	}

	root = sourceCheckCandidateRoot(explicitRoot)
	result := runSourceCheck(root)
	result.Issues = append(result.Issues, sourceCheckIssue{
		Area:    "input",
		Path:    root,
		Message: err.Error(),
		Actual:  "missing or invalid source checkout",
	})
	sortSourceCheckIssues(result.Issues)
	finalizeSourceCheckResult(&result)
	return result, err
}

func sourceCheckCandidateRoot(explicitRoot string) string {
	if strings.TrimSpace(explicitRoot) != "" {
		if abs, err := filepath.Abs(explicitRoot); err == nil {
			return abs
		}
		return explicitRoot
	}
	if cwd, err := os.Getwd(); err == nil {
		if root := findAetherModuleRoot(cwd); root != "" {
			return root
		}
		return cwd
	}
	return "."
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
		SchemaVersion: sourceCheckResultSchemaVersion,
		OperationID:   "source.parity.inspect",
		Root:          root,
		Components:    []sourceCheckComponent{},
		Issues:        []sourceCheckIssue{},
		Findings:      []maintenanceInspectionFinding{},
		Evidence:      []maintenanceInspectionEvidence{},
		StateEffect:   colony.LifecycleStateEffectNone,
		Blockers:      []maintenanceInspectionFinding{},
		NextAction:    "aether update --force",
		Next:          "Fix reported source surface drift, then rerun `aether source-check` before publishing.",
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
		result.NextAction = "aether integrity --source"
		result.Next = "Source surfaces are aligned. Publish only after active work is intentionally committed."
	}
	finalizeSourceCheckResult(&result)
	return result
}

func finalizeSourceCheckResult(result *sourceCheckResult) {
	if result == nil {
		return
	}
	result.SchemaVersion = sourceCheckResultSchemaVersion
	result.OperationID = "source.parity.inspect"
	result.StateEffect = colony.LifecycleStateEffectNone
	result.OK = len(result.Issues) == 0
	result.Findings = make([]maintenanceInspectionFinding, 0, len(result.Issues))
	for _, issue := range result.Issues {
		result.Findings = append(result.Findings, sourceCheckFinding(issue))
	}
	result.Blockers = append([]maintenanceInspectionFinding(nil), result.Findings...)
	if result.Blockers == nil {
		result.Blockers = []maintenanceInspectionFinding{}
	}
	result.Evidence = make([]maintenanceInspectionEvidence, 0, len(result.Components))
	for _, component := range result.Components {
		result.Evidence = append(result.Evidence, maintenanceInspectionEvidence{
			Scope:   component.Name,
			Paths:   sourceCheckComponentPaths(component.Name),
			Checked: component.Checked,
			Status:  component.Status,
		})
	}
	if result.Evidence == nil {
		result.Evidence = []maintenanceInspectionEvidence{}
	}
	status := "pass"
	if !result.OK {
		status = "fail"
		result.NextAction = "aether update --force"
	} else if strings.TrimSpace(result.NextAction) == "" {
		result.NextAction = "aether integrity --source"
	}
	result.Verification = maintenanceInspectionVerification{
		Status:        status,
		EvidenceCount: len(result.Evidence),
		FindingCount:  len(result.Findings),
	}
}

func sourceCheckFinding(issue sourceCheckIssue) maintenanceInspectionFinding {
	finding := maintenanceInspectionFinding{
		Code:            "source." + strings.ReplaceAll(strings.TrimSpace(issue.Area), " ", "_"),
		Summary:         issue.Message,
		EvidencePaths:   []string{},
		RecoveryCommand: "aether source-check",
	}
	path := filepath.ToSlash(issue.Path)
	expected := filepath.ToSlash(issue.Expected)
	if strings.HasPrefix(path, ".aether/") {
		finding.SourcePath = path
	}
	if strings.HasPrefix(path, ".claude/") || strings.HasPrefix(path, ".opencode/") || strings.HasPrefix(path, ".codex/") {
		finding.GeneratedPath = path
		finding.RecoveryCommand = "aether update --force"
		if strings.HasPrefix(expected, ".aether/") {
			finding.SourcePath = expected
		} else if strings.Contains(path, "/commands/") {
			name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
			finding.SourcePath = filepath.ToSlash(filepath.Join(".aether", "commands", name+".yaml"))
		}
	}
	for _, evidencePath := range []string{finding.SourcePath, finding.GeneratedPath, path} {
		if evidencePath == "" || containsString(finding.EvidencePaths, evidencePath) {
			continue
		}
		finding.EvidencePaths = append(finding.EvidencePaths, evidencePath)
	}
	sort.Strings(finding.EvidencePaths)
	return finding
}

func sourceCheckComponentPaths(name string) []string {
	switch name {
	case "canonical source surfaces":
		return []string{".aether/commands", ".aether/docs", ".aether/exchange", ".aether/skills", ".aether/templates", ".aether/utils", ".aether/workers.md", ".claude/agents/ant", ".claude/commands/ant", ".codex/agents", ".opencode/agents", ".opencode/commands/ant"}
	case "retired source mirrors":
		return []string{".aether/agents-claude", ".aether/agents-codex", ".aether/commands/claude", ".aether/commands/opencode", ".aether/skills-codex"}
	case "generated command wrappers":
		return []string{".aether/commands", ".claude/commands/ant", ".opencode/commands/ant"}
	default:
		return []string{}
	}
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
	yamlSpecs := map[string]sourceCheckCommandSpec{}
	// declaredAliases maps an alias command name (e.g. "pause-colony") to the
	// canonical command name that declares it (e.g. "pause"). The alias is
	// declared exactly once, in the canonical command's own YAML `aliases:`
	// field -- this is what legitimises the alias's wrapper files instead of
	// requiring a second YAML definition file for it.
	declaredAliases := map[string]string{}
	var issues []sourceCheckIssue
	for _, rel := range sourceCheckFiles(root, ".aether/commands", func(rel string) bool {
		return !strings.Contains(filepath.ToSlash(rel), "/") && filepath.Ext(rel) == ".yaml"
	}) {
		name := strings.TrimSuffix(filepath.Base(rel), ".yaml")
		yamlRel := filepath.ToSlash(filepath.Join(".aether", "commands", rel))
		yamlNames[name] = yamlRel
		spec, specIssues := readSourceCheckCommandSpec(root, yamlRel, name)
		yamlSpecs[name] = spec
		issues = append(issues, specIssues...)
		for _, alias := range spec.Aliases {
			alias = strings.TrimSpace(alias)
			if alias == "" {
				continue
			}
			if existing, ok := declaredAliases[alias]; ok && existing != name {
				issues = append(issues, sourceCheckIssue{
					Area:    "commands",
					Path:    yamlRel,
					Message: fmt.Sprintf("alias %q is declared by more than one command source (%s and %s)", alias, existing, name),
				})
				continue
			}
			declaredAliases[alias] = name
		}
	}

	// A declared alias must have a wrapper on all three hand-maintained
	// surfaces (S-04): the nested Claude copy, the flat Claude copy, and the
	// OpenCode copy. This is the other half of "declared once, enforced in
	// both directions" -- a missing wrapper for a declared alias is reported
	// here, even on the flat surface the loop below never scans.
	for alias, canonicalName := range declaredAliases {
		canonicalYAML := yamlNames[canonicalName]
		for _, aliasPath := range []string{
			filepath.ToSlash(filepath.Join(".claude", "commands", "ant", alias+".md")),
			filepath.ToSlash(filepath.Join(".claude", "commands", "ant-"+alias+".md")),
			filepath.ToSlash(filepath.Join(".opencode", "commands", "ant", alias+".md")),
		} {
			if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(aliasPath))); err != nil {
				issues = append(issues, sourceCheckIssue{
					Area:     "commands",
					Path:     aliasPath,
					Message:  "declared alias has no wrapper at this location",
					Expected: canonicalYAML,
					Actual:   "missing",
				})
			}
		}
	}

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
					Message:  "wrapper is missing the Aether-managed header",
					Expected: "<!-- Aether-managed: runtime spec at .aether/commands/<name>.yaml. Synced by aether update. -->",
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
			frontmatter, body, err := parseSourceCheckWrapper(data)
			if err != nil {
				issues = append(issues, sourceCheckIssue{
					Area:     "commands",
					Path:     wrapperRel,
					Message:  "generated wrapper frontmatter is invalid",
					Expected: "YAML frontmatter with name and description",
					Actual:   err.Error(),
				})
				continue
			}
			issues = append(issues, compareSourceCheckWrapperContract(wrapperRel, yamlSpecs[name], frontmatter, body)...)
		}

		for _, rel := range sourceCheckFiles(root, wrapperDir, func(rel string) bool {
			return !strings.Contains(filepath.ToSlash(rel), "/") && filepath.Ext(rel) == ".md"
		}) {
			name := strings.TrimSuffix(filepath.Base(rel), ".md")
			if _, ok := yamlNames[name]; ok {
				continue
			}
			if _, ok := declaredAliases[name]; ok {
				// Legitimised by its canonical command's alias declaration --
				// no second YAML definition file is required for it.
				continue
			}
			issues = append(issues, sourceCheckIssue{
				Area:    "commands",
				Path:    filepath.ToSlash(filepath.Join(wrapperDir, rel)),
				Message: "generated wrapper has no matching YAML source",
				Actual:  filepath.Join(yamlDir, name+".yaml"),
			})
		}
	}

	return checked, issues
}

func readSourceCheckCommandSpec(root, rel, commandName string) (sourceCheckCommandSpec, []sourceCheckIssue) {
	var spec sourceCheckCommandSpec
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return spec, []sourceCheckIssue{{
			Area:    "commands",
			Path:    rel,
			Message: "YAML command source is unreadable",
			Actual:  err.Error(),
		}}
	}
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return spec, []sourceCheckIssue{{
			Area:    "commands",
			Path:    rel,
			Message: "YAML command source is invalid",
			Actual:  err.Error(),
		}}
	}

	var issues []sourceCheckIssue
	expectedName := "ant-" + commandName
	if strings.TrimSpace(spec.Name) == "" {
		issues = append(issues, sourceCheckIssue{
			Area:     "commands",
			Path:     rel,
			Message:  "YAML command source is missing name",
			Expected: expectedName,
			Actual:   "",
		})
	} else if spec.Name != expectedName {
		issues = append(issues, sourceCheckIssue{
			Area:     "commands",
			Path:     rel,
			Message:  "YAML command name does not match command file",
			Expected: expectedName,
			Actual:   spec.Name,
		})
	}
	if strings.TrimSpace(spec.Description) == "" {
		issues = append(issues, sourceCheckIssue{
			Area:    "commands",
			Path:    rel,
			Message: "YAML command source is missing description",
		})
	}
	if strings.TrimSpace(spec.SourceOfTruth) == "" {
		issues = append(issues, sourceCheckIssue{
			Area:    "commands",
			Path:    rel,
			Message: "YAML command source is missing source_of_truth",
		})
	}
	return spec, issues
}

func parseSourceCheckWrapper(data []byte) (sourceCheckWrapperFrontmatter, string, error) {
	var frontmatter sourceCheckWrapperFrontmatter
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	_, bodyWithFrontmatter, ok := strings.Cut(text, "\n")
	if !ok {
		return frontmatter, "", fmt.Errorf("missing body after generated header")
	}
	if !strings.HasPrefix(bodyWithFrontmatter, "---\n") {
		return frontmatter, "", fmt.Errorf("missing opening frontmatter delimiter")
	}
	rest := strings.TrimPrefix(bodyWithFrontmatter, "---\n")
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		return frontmatter, "", fmt.Errorf("missing closing frontmatter delimiter")
	}
	rawFrontmatter := rest[:end]
	body := rest[end+len("\n---\n"):]
	if err := yaml.Unmarshal([]byte(rawFrontmatter), &frontmatter); err != nil {
		return frontmatter, "", err
	}
	return frontmatter, body, nil
}

func compareSourceCheckWrapperContract(wrapperRel string, spec sourceCheckCommandSpec, frontmatter sourceCheckWrapperFrontmatter, body string) []sourceCheckIssue {
	var issues []sourceCheckIssue
	if spec.Name != "" && frontmatter.Name != spec.Name {
		issues = append(issues, sourceCheckIssue{
			Area:     "commands",
			Path:     wrapperRel,
			Message:  "generated wrapper frontmatter name does not match YAML",
			Expected: spec.Name,
			Actual:   frontmatter.Name,
		})
	}
	if strings.TrimSpace(frontmatter.Description) == "" {
		issues = append(issues, sourceCheckIssue{
			Area:    "commands",
			Path:    wrapperRel,
			Message: "generated wrapper frontmatter is missing description",
		})
	}
	if spec.SourceOfTruth != "" && strings.Contains(body, spec.SourceOfTruth) {
		for _, command := range sourceCheckRuntimeCommands(spec) {
			if !sourceCheckWrapperContainsRuntimeCommand(body, command) {
				issues = append(issues, sourceCheckIssue{
					Area:     "commands",
					Path:     wrapperRel,
					Message:  "generated wrapper is missing YAML runtime command",
					Expected: command,
					Actual:   "missing",
				})
			}
		}
	}
	return issues
}

func sourceCheckRuntimeCommands(spec sourceCheckCommandSpec) []string {
	var commands []string
	for _, command := range []string{
		spec.Runtime.Command,
		spec.Runtime.DefaultCommand,
		spec.Runtime.ManifestCommand,
		spec.Runtime.FinalizerCommand,
	} {
		command = strings.TrimSpace(command)
		if command != "" {
			commands = append(commands, command)
		}
	}
	return commands
}

func sourceCheckWrapperContainsRuntimeCommand(body, command string) bool {
	for _, needle := range sourceCheckRuntimeCommandNeedles(command) {
		if strings.Contains(body, needle) {
			return true
		}
	}
	return false
}

func sourceCheckRuntimeCommandNeedles(command string) []string {
	var needles []string
	normalized := strings.Join(strings.Fields(strings.ReplaceAll(command, "$ARGUMENTS", "")), " ")
	if normalized != "" {
		needles = append(needles, normalized)
	}
	if anchor := sourceCheckRuntimeCommandAnchor(command); anchor != "" && anchor != normalized {
		needles = append(needles, anchor)
	}
	return needles
}

func sourceCheckRuntimeCommandAnchor(command string) string {
	fields := strings.Fields(command)
	for i, field := range fields {
		if field != "aether" {
			continue
		}
		if i+1 >= len(fields) {
			return "aether"
		}
		if fields[i+1] == "host" && i+2 < len(fields) {
			return "aether host " + fields[i+2]
		}
		return "aether " + fields[i+1]
	}
	return ""
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
	b.WriteString(visualDividerStr())
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
