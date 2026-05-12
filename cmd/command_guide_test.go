package cmd

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestCommandGuideRegistered(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"command-guide", "init"})
	if err != nil {
		t.Fatalf("command-guide is not registered: %v", err)
	}
	if cmd.Name() != "command-guide" {
		t.Fatalf("root lookup returned %q, want command-guide", cmd.Name())
	}
}

func TestCommandGuideCoversAllYamlCommands(t *testing.T) {
	yamlCommands := yamlCommandNamesForGuideTest(t)
	catalog := commandGuideCatalog()

	var missing []string
	for _, command := range yamlCommands {
		if _, ok := catalog[command]; !ok {
			missing = append(missing, command)
		}
	}

	var extra []string
	yamlSet := map[string]bool{}
	for _, command := range yamlCommands {
		yamlSet[command] = true
	}
	for command := range catalog {
		if !yamlSet[command] {
			extra = append(extra, command)
		}
	}

	if len(missing) > 0 || len(extra) > 0 {
		sort.Strings(missing)
		sort.Strings(extra)
		t.Fatalf("command-guide/YAML drift\nmissing from guide: %v\nextra in guide: %v", missing, extra)
	}
}

func TestCommandGuideIntelligentCommandsHaveOrchestration(t *testing.T) {
	tests := map[string]struct {
		category string
		skill    string
	}{
		"init":     {commandGuideCategoryFullOrchestration, commandGuideSkillCreation},
		"oracle":   {commandGuideCategoryFullOrchestration, commandGuideSkillResearch},
		"colonize": {commandGuideCategoryFullOrchestration, commandGuideSkillBuildCycle},
		"swarm":    {commandGuideCategoryFullOrchestration, commandGuideSkillBuildCycle},
		"plan":     {commandGuideCategoryFullOrchestration, commandGuideSkillBuildCycle},
		"build":    {commandGuideCategoryFullOrchestration, commandGuideSkillBuildCycle},
		"continue": {commandGuideCategorySemiIntelligent, commandGuideSkillBuildCycle},
		"seal":     {commandGuideCategorySemiIntelligent, commandGuideSkillBuildCycle},
		"discuss":  {commandGuideCategorySemiIntelligent, commandGuideSkillResearch},
	}

	for command, want := range tests {
		guide, err := buildCommandGuide(command, "codex")
		if err != nil {
			t.Fatalf("buildCommandGuide(%q): %v", command, err)
		}
		if guide.Category != want.category {
			t.Errorf("%s category = %q, want %q", command, guide.Category, want.category)
		}
		if guide.SkillReference != want.skill {
			t.Errorf("%s skill = %q, want %q", command, guide.SkillReference, want.skill)
		}
		if guide.Literal {
			t.Errorf("%s should not be literal passthrough", command)
		}
		if len(guide.PreSteps) == 0 {
			t.Errorf("%s should include orchestration pre_steps", command)
		}
		if len(guide.PostSteps) == 0 {
			t.Errorf("%s should include orchestration post_steps", command)
		}
		if !strings.Contains(guide.RawBypass, "raw") {
			t.Errorf("%s should document raw bypass, got %q", command, guide.RawBypass)
		}
	}
}

func TestOracleGuideCarriesBroadScopeTimeoutGuard(t *testing.T) {
	guide, err := buildCommandGuide("oracle", "codex")
	if err != nil {
		t.Fatalf("buildCommandGuide(oracle): %v", err)
	}
	text := strings.Join(append(append([]string{}, guide.PreSteps...), guide.PostSteps...), "\n")
	for _, want := range []string{
		"--depth quick",
		"--confidence-target",
		"95% recommended",
		"full-system audits",
		"large uncommitted diffs",
		"aether oracle status",
		"times out",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("oracle command-guide missing broad-scope timeout guard %q", want)
		}
	}
}

func TestLifecycleGuidesCarryOrchestratorBoundaryGuidance(t *testing.T) {
	for _, command := range []string{"plan", "build", "continue", "seal"} {
		guide, err := buildCommandGuide(command, "codex")
		if err != nil {
			t.Fatalf("buildCommandGuide(%q): %v", command, err)
		}
		text := strings.Join(append(append([]string{}, guide.PreSteps...), guide.PostSteps...), "\n")
		for _, want := range []string{
			"orchestrator_boundary_guidance",
			"after_discuss_next",
			"aether discuss",
			"fresh",
		} {
			if !strings.Contains(text, want) {
				t.Errorf("%s command-guide missing Orchestrator guidance anchor %q", command, want)
			}
		}
	}
}

func TestCodexLifecycleGuidesRequireVisibleWorkerActivity(t *testing.T) {
	tests := map[string][]string{
		"colonize": {
			"AETHER_OUTPUT_MODE=json aether colonize --plan-only",
			"visible live Task/subagent panels",
			"aether spawn-log",
			"aether spawn-complete",
			"ceremony worker-complete --workflow colonize",
			"AETHER_OUTPUT_MODE=json aether colonize-finalize",
		},
		"plan": {
			"AETHER_OUTPUT_MODE=json aether plan --plan-only",
			"visible live Task/subagent panels",
			"aether spawn-log",
			"aether spawn-complete",
			"ceremony worker-complete --workflow plan",
			"AETHER_OUTPUT_MODE=json aether plan-finalize",
		},
		"build": {
			"AETHER_OUTPUT_MODE=json aether build <phase> --plan-only",
			"visible live Task/subagent panels",
			"aether spawn-log",
			"aether spawn-complete",
			"ceremony worker-complete --workflow build",
			"AETHER_OUTPUT_MODE=json aether build-finalize",
		},
		"continue": {
			"AETHER_OUTPUT_MODE=visual aether continue --skip-watchers --verification-depth standard",
			"AETHER_OUTPUT_MODE=json aether continue --plan-only --verification-depth heavy",
			"visible live Task/subagent panels",
			"aether spawn-log",
			"aether spawn-complete",
			"ceremony worker-complete --workflow continue",
			"continue-finalize",
		},
	}

	for command, wants := range tests {
		guide, err := buildCommandGuide(command, "codex")
		if err != nil {
			t.Fatalf("buildCommandGuide(%q): %v", command, err)
		}
		text := strings.Join(append(append([]string{}, guide.PreSteps...), append([]string{guide.RunCommand}, guide.PostSteps...)...), "\n")
		for _, want := range wants {
			if !strings.Contains(text, want) {
				t.Errorf("%s command-guide missing visible worker activity anchor %q", command, want)
			}
		}
	}
}

func TestCodexLifecycleYamlAndGuidesAgreeOnWorkerActivity(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	tests := map[string][]string{
		"colonize": {"spawn-log", "spawn-complete", "colonize-finalize", "visible live Task/subagent"},
		"plan":     {"spawn-log", "spawn-complete", "plan-finalize", "visible live Task/subagent"},
		"build":    {"spawn-log", "spawn-complete", "build-finalize", "visible live Task/subagent"},
		"continue": {"spawn-log", "spawn-complete", "continue-finalize", "visible live Task/subagent"},
	}

	for command, anchors := range tests {
		guide, err := buildCommandGuide(command, "codex")
		if err != nil {
			t.Fatalf("buildCommandGuide(%q): %v", command, err)
		}
		guideText := strings.Join(append(append([]string{}, guide.PreSteps...), append([]string{guide.RunCommand}, guide.PostSteps...)...), "\n")
		yamlPath := filepath.Join(repoRoot, ".aether", "commands", command+".yaml")
		content, err := os.ReadFile(yamlPath)
		if err != nil {
			t.Fatalf("read %s: %v", yamlPath, err)
		}
		yamlText := string(content)
		for _, anchor := range anchors {
			if !strings.Contains(guideText, anchor) {
				t.Errorf("%s command-guide missing shared worker activity anchor %q", command, anchor)
			}
			if !strings.Contains(yamlText, anchor) {
				t.Errorf("%s YAML missing shared worker activity anchor %q", command, anchor)
			}
		}
	}
}

func TestCodexLifecycleSkillMirrorsWorkerActivityContract(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}
	path := filepath.Join(repoRoot, ".aether", "skills", "colony", commandGuideSkillBuildCycle, "SKILL.md")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	text := string(content)
	for _, want := range []string{
		"AETHER_OUTPUT_MODE=json aether colonize --plan-only",
		"AETHER_OUTPUT_MODE=json aether plan --plan-only",
		"AETHER_OUTPUT_MODE=json aether build <phase> --plan-only",
		"AETHER_OUTPUT_MODE=visual aether continue --skip-watchers --verification-depth standard",
		"AETHER_OUTPUT_MODE=json aether continue --plan-only --verification-depth heavy",
		"aether spawn-log",
		"aether spawn-complete",
		"aether ceremony worker-complete",
		"visible live Task/subagent panels",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("%s missing lifecycle worker activity anchor %q", path, want)
		}
	}
}

func TestInitGuideAndWrappersCarryColonyModeChoice(t *testing.T) {
	guide, err := buildCommandGuide("init", "codex")
	if err != nil {
		t.Fatalf("buildCommandGuide(init): %v", err)
	}
	guideText := strings.Join(append(append([]string{}, guide.PreSteps...), append([]string{guide.RunCommand}, guide.PostSteps...)...), "\n")
	for _, want := range []string{
		"Colony Mode",
		"Orchestrator Mode",
		"default to Colony Mode",
		"--colony-mode",
	} {
		if !strings.Contains(guideText, want) {
			t.Errorf("init command-guide missing colony mode choice anchor %q", want)
		}
	}

	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}
	files := []string{
		filepath.Join(repoRoot, ".aether", "commands", "init.yaml"),
		filepath.Join(repoRoot, ".claude", "commands", "ant", "init.md"),
		filepath.Join(repoRoot, ".opencode", "commands", "ant", "init.md"),
		filepath.Join(repoRoot, ".aether", "skills", "colony", "aether-colony-creation", "SKILL.md"),
	}
	for _, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(content)
		for _, want := range []string{
			"Colony Mode",
			"Orchestrator Mode",
			"--colony-mode",
		} {
			if !strings.Contains(text, want) {
				t.Errorf("%s missing colony mode choice anchor %q", path, want)
			}
		}
	}
}

func TestCommandGuideLiteralCommandsArePassthrough(t *testing.T) {
	for _, command := range []string{"status", "focus", "reference-list", "update"} {
		guide, err := buildCommandGuide(command, "codex")
		if err != nil {
			t.Fatalf("buildCommandGuide(%q): %v", command, err)
		}
		if guide.Category != commandGuideCategoryLiteral {
			t.Errorf("%s category = %q, want literal", command, guide.Category)
		}
		if !guide.Literal {
			t.Errorf("%s should be literal passthrough", command)
		}
		if guide.SkillReference != "" {
			t.Errorf("%s skill = %q, want empty", command, guide.SkillReference)
		}
		if len(guide.PreSteps) != 0 || len(guide.PostSteps) != 0 {
			t.Errorf("%s literal guide should not include pre/post orchestration", command)
		}
	}
}

func TestCommandGuideAdaptsNonCodexPlatform(t *testing.T) {
	guide, err := buildCommandGuide("init", "claude")
	if err != nil {
		t.Fatalf("buildCommandGuide(init, claude): %v", err)
	}
	if guide.Platform != "claude" {
		t.Fatalf("platform = %q, want claude", guide.Platform)
	}
	if guide.SkillReference != "" {
		t.Fatalf("Claude guide should not reference Codex skill, got %q", guide.SkillReference)
	}
	if len(guide.PreSteps) == 0 || !strings.Contains(guide.PreSteps[0], "slash-command wrapper") {
		t.Fatalf("Claude guide should point at wrapper orchestration, got %#v", guide.PreSteps)
	}
}

func TestCommandGuideYamlCodexMetadataMatches(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	for _, command := range []string{"init", "oracle", "colonize", "swarm", "plan", "build", "continue", "seal", "discuss"} {
		guide, err := buildCommandGuide(command, "codex")
		if err != nil {
			t.Fatalf("buildCommandGuide(%q): %v", command, err)
		}
		meta := readCommandGuideYAMLMetadata(t, filepath.Join(repoRoot, ".aether", "commands", command+".yaml"))
		if meta.CodexOrchestration.Category != guide.Category {
			t.Errorf("%s YAML category = %q, want %q", command, meta.CodexOrchestration.Category, guide.Category)
		}
		if meta.CodexOrchestration.Skill != guide.SkillReference {
			t.Errorf("%s YAML skill = %q, want %q", command, meta.CodexOrchestration.Skill, guide.SkillReference)
		}
		wantGuide := "aether command-guide " + command + " --platform codex"
		if meta.CodexOrchestration.Guide != wantGuide {
			t.Errorf("%s YAML guide = %q, want %q", command, meta.CodexOrchestration.Guide, wantGuide)
		}
		if !strings.Contains(meta.CodexOrchestration.DriftGuard, "cmd/command_guide.go") {
			t.Errorf("%s YAML drift guard should mention cmd/command_guide.go", command)
		}
	}
}

func TestIntelligentWrappersCarryCodexDriftGuard(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	commands := []string{"init", "oracle", "colonize", "swarm", "plan", "build", "continue", "seal", "discuss"}
	wrapperDirs := []string{
		filepath.Join(repoRoot, ".claude", "commands", "ant"),
		filepath.Join(repoRoot, ".opencode", "commands", "ant"),
	}

	for _, dir := range wrapperDirs {
		for _, command := range commands {
			guide, err := buildCommandGuide(command, "codex")
			if err != nil {
				t.Fatalf("buildCommandGuide(%q): %v", command, err)
			}
			path := filepath.Join(dir, command+".md")
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			text := string(content)
			for _, want := range []string{
				"Cross-Platform Drift Guard",
				"cmd/command_guide.go",
				guide.SkillReference,
				"aether command-guide " + command + " --platform codex",
			} {
				if !strings.Contains(text, want) {
					t.Errorf("%s missing drift guard anchor %q", path, want)
				}
			}
		}
	}
}

func TestLifecycleWrapperSourcesCarryOrchestratorBoundaryGuidance(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	var files []string
	for _, command := range []string{"plan", "build", "continue", "seal"} {
		files = append(files,
			filepath.Join(repoRoot, ".aether", "commands", command+".yaml"),
			filepath.Join(repoRoot, ".claude", "commands", "ant", command+".md"),
			filepath.Join(repoRoot, ".opencode", "commands", "ant", command+".md"),
		)
	}
	files = append(files,
		filepath.Join(repoRoot, ".aether", "skills", "colony", "aether-colony-build-cycle", "SKILL.md"),
		filepath.Join(repoRoot, ".aether", "docs", "wrapper-runtime-ux-contract.md"),
		filepath.Join(repoRoot, ".aether", "docs", "source-of-truth-map.md"),
	)

	for _, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(content)
		for _, want := range []string{
			"orchestrator_boundary_guidance",
			"after_discuss_next",
			"aether discuss",
			"fresh",
		} {
			if !strings.Contains(text, want) {
				t.Errorf("%s missing Orchestrator guidance anchor %q", path, want)
			}
		}
	}
}

func TestOracleWrappersAndSkillCarryTimeoutGuard(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	files := []string{
		filepath.Join(repoRoot, ".aether", "commands", "oracle.yaml"),
		filepath.Join(repoRoot, ".claude", "commands", "ant", "oracle.md"),
		filepath.Join(repoRoot, ".opencode", "commands", "ant", "oracle.md"),
		filepath.Join(repoRoot, ".aether", "skills", "colony", "aether-colony-research", "SKILL.md"),
	}
	for _, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(content)
		for _, want := range []string{
			"--depth quick",
			"--confidence-target",
			"95%",
			"aether oracle status",
			"full-system audit",
		} {
			if !strings.Contains(text, want) {
				t.Errorf("%s missing oracle timeout guard anchor %q", path, want)
			}
		}
	}
}

func TestCodexDocsReferenceCommandGuideAndSkills(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	files := []string{
		filepath.Join(repoRoot, "AGENTS.md"),
		filepath.Join(repoRoot, ".codex", "CODEX.md"),
		filepath.Join(repoRoot, ".aether", "docs", "source-of-truth-map.md"),
		filepath.Join(repoRoot, ".aether", "docs", "wrapper-runtime-ux-contract.md"),
		filepath.Join(repoRoot, ".aether", "skills", "colony", "colony-interaction", "SKILL.md"),
	}

	for _, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(content)
		for _, want := range []string{
			"aether command-guide",
			commandGuideSkillCreation,
			commandGuideSkillResearch,
			commandGuideSkillBuildCycle,
		} {
			if !strings.Contains(text, want) {
				t.Errorf("%s missing %q", path, want)
			}
		}
	}
}

func TestCodexLifecycleSkillsLiveOnlyInAetherSource(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	for _, skill := range []string{commandGuideSkillCreation, commandGuideSkillResearch, commandGuideSkillBuildCycle} {
		sourcePath := filepath.Join(repoRoot, ".aether", "skills", "colony", skill, "SKILL.md")
		source, err := os.ReadFile(sourcePath)
		if err != nil {
			t.Fatalf("read source skill %s: %v", skill, err)
		}
		if !strings.Contains(string(source), "source: shipped") {
			t.Fatalf("%s source skill missing source: shipped", skill)
		}
		codexPath := filepath.Join(repoRoot, ".codex", "skills", "aether", "colony", skill, "SKILL.md")
		if _, err := os.Stat(codexPath); !os.IsNotExist(err) {
			t.Fatalf("%s should not have a repo-local Codex full-skill mirror", skill)
		}
	}
}

func TestCodexGeneratedShimsIncludeCommandGuideSkills(t *testing.T) {
	shims := map[string]codexSkillShim{}
	for _, shim := range codexSkillShims() {
		shims[shim.Name] = shim
	}
	for _, skill := range []string{commandGuideSkillCreation, commandGuideSkillResearch, commandGuideSkillBuildCycle} {
		if _, ok := shims[skill]; !ok {
			t.Fatalf("codex generated shims missing command-guide skill %q", skill)
		}
	}
	creationShim := shims[commandGuideSkillCreation]
	for _, want := range []string{"Colony Mode", "Orchestrator Mode", "--colony-mode"} {
		if !strings.Contains(creationShim.Body, want) {
			t.Fatalf("codex creation shim missing %q", want)
		}
	}

	buildCycleShim := shims[commandGuideSkillBuildCycle]
	for _, want := range []string{"colonize", "aether colonize", "plan-only", "finalize"} {
		text := buildCycleShim.Description + "\n" + buildCycleShim.Body + "\n" + strings.Join(buildCycleShim.TaskKeywords, "\n")
		if !strings.Contains(text, want) {
			t.Fatalf("codex build-cycle shim missing %q", want)
		}
	}
	rendered := renderCodexSkillShim(buildCycleShim)
	fm := parseSkillFrontmatter(rendered)
	if fm == nil {
		t.Fatalf("generated build-cycle shim should have parseable frontmatter")
	}
	wantTriggers := []string{"colonize", "plan", "build", "continue", "swarm", "seal"}
	if strings.Join(fm.WorkflowTriggers, ",") != strings.Join(wantTriggers, ",") {
		t.Fatalf("build-cycle shim workflow triggers = %v, want %v", fm.WorkflowTriggers, wantTriggers)
	}
	for _, want := range []string{"aether colonize", "aether plan", "aether build", "aether continue", "aether swarm", "aether seal", "dispatch manifest", "plan-only", "finalize"} {
		if !stringSliceContains(fm.TaskKeywords, want) {
			t.Fatalf("build-cycle shim task keywords missing %q: %v", want, fm.TaskKeywords)
		}
	}
}

func stringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func yamlCommandNamesForGuideTest(t *testing.T) []string {
	t.Helper()
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}
	entries, err := os.ReadDir(filepath.Join(repoRoot, ".aether", "commands"))
	if err != nil {
		t.Fatalf("read .aether/commands: %v", err)
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		names = append(names, strings.TrimSuffix(entry.Name(), ".yaml"))
	}
	sort.Strings(names)
	return names
}

type commandGuideYAMLMetadata struct {
	CodexOrchestration struct {
		Category   string `yaml:"category"`
		Skill      string `yaml:"skill"`
		Guide      string `yaml:"guide"`
		DriftGuard string `yaml:"drift_guard"`
	} `yaml:"codex_orchestration"`
}

func readCommandGuideYAMLMetadata(t *testing.T, path string) commandGuideYAMLMetadata {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var meta commandGuideYAMLMetadata
	if err := yaml.Unmarshal(content, &meta); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	if meta.CodexOrchestration.Category == "" {
		t.Fatalf("%s missing codex_orchestration metadata", path)
	}
	return meta
}
