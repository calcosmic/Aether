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
	for _, command := range []string{"colonize", "plan", "build", "continue", "seal"} {
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

func TestLifecycleGuidesSurfaceSpawnBudgetReasons(t *testing.T) {
	for _, command := range []string{"plan", "build", "continue", "seal"} {
		guide, err := buildCommandGuide(command, "codex")
		if err != nil {
			t.Fatalf("buildCommandGuide(%q): %v", command, err)
		}
		text := strings.Join(append(append([]string{}, guide.PreSteps...), guide.PostSteps...), "\n")
		for _, want := range []string{
			"queen_execution_policy.spawn_budget",
			"selected/pruned caste reasons",
			"why workers were or were not spawned",
		} {
			if !strings.Contains(text, want) {
				t.Errorf("%s command-guide missing spawn budget reason anchor %q", command, want)
			}
		}
	}
}

func TestCodexLifecycleGuidesRequireVisibleWorkerActivity(t *testing.T) {
	tests := map[string][]string{
		"colonize": {
			"aether host colonize",
			"visible live Task/subagent panels",
			"aether spawn-log",
			"aether spawn-complete",
			"ceremony worker-complete --workflow colonize",
			"AETHER_OUTPUT_MODE=json aether colonize-finalize",
		},
		"plan": {
			"aether host plan --depth <choice> --planning-depth <choice>",
			"visible live Task/subagent panels",
			"aether spawn-log",
			"aether spawn-complete",
			"ceremony worker-complete --workflow plan",
			"AETHER_OUTPUT_MODE=json aether plan-finalize",
		},
		"build": {
			"aether host build --dry-run <phase>",
			"Parse `result.manifest.dispatch_manifest`",
			"visible live Task/subagent panels",
			"aether spawn-log",
			"aether spawn-complete",
			"ceremony worker-complete --workflow build",
			"AETHER_OUTPUT_MODE=json aether build-finalize",
		},
		"continue": {
			"AETHER_OUTPUT_MODE=visual aether continue --verification-depth standard",
			"aether host continue --dry-run --classic-ceremony",
			"Parse `result.manifest.continue_manifest`",
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

func TestLifecycleGuidesDocumentApprovedTempCompletionContract(t *testing.T) {
	for _, command := range []string{"colonize", "plan", "build", "continue", "seal"} {
		guide, err := buildCommandGuide(command, "codex")
		if err != nil {
			t.Fatalf("buildCommandGuide(%q): %v", command, err)
		}
		text := strings.Join(append(append([]string{}, guide.PreSteps...), append([]string{guide.RunCommand}, guide.PostSteps...)...), "\n")
		for _, want := range []string{
			"${TMPDIR:-/tmp}/aether-<workflow>-<run>/<workflow>-completion.json",
			"never write wrapper result artifacts under `.aether/data`",
			"<approved temp completion JSON>",
		} {
			if !strings.Contains(text, want) {
				t.Errorf("%s command-guide missing approved temp completion contract %q", command, want)
			}
		}
	}
}

func TestCodexHostBackedGuidesUseTypeScriptHostSpine(t *testing.T) {
	tests := map[string]struct {
		required []string
		retired  []string
	}{
		"plan": {
			required: []string{
				"aether host plan --depth <choice> --planning-depth <choice>",
				"Parse `result.plan_manifest` or `result.planning_manifest`",
				"AETHER_OUTPUT_MODE=json aether plan-finalize",
			},
			retired: []string{
				"AETHER_OUTPUT_MODE=json aether plan --plan-only --depth <choice>",
			},
		},
		"colonize": {
			required: []string{
				"aether host colonize",
				"Parse `result.colonize_manifest`",
				"AETHER_OUTPUT_MODE=json aether colonize-finalize",
			},
			retired: []string{
				"AETHER_OUTPUT_MODE=json aether colonize --plan-only",
			},
		},
		"build": {
			required: []string{
				"aether host build --dry-run <phase>",
				"Parse `result.manifest.dispatch_manifest`",
				"AETHER_OUTPUT_MODE=json aether build-finalize",
			},
			retired: []string{
				"AETHER_OUTPUT_MODE=json aether build <phase> --plan-only",
				"aether host build <phase>",
				"Parse `result.dispatch_manifest`",
			},
		},
		"continue": {
			required: []string{
				"AETHER_OUTPUT_MODE=visual aether continue --verification-depth standard",
				"aether host continue --dry-run --classic-ceremony",
				"Parse `result.manifest.continue_manifest`",
				"continue-finalize",
			},
			retired: []string{
				"AETHER_OUTPUT_MODE=json aether continue --plan-only --verification-depth heavy",
				"aether host continue --classic-ceremony",
				"Parse `result.continue_manifest`",
			},
		},
		"seal": {
			required: []string{
				"aether host seal",
				"Parse `result.seal_manifest`",
				"AETHER_OUTPUT_MODE=json aether seal-finalize",
			},
			retired: []string{
				"AETHER_OUTPUT_MODE=json aether seal --plan-only",
			},
		},
	}

	for command, test := range tests {
		guide, err := buildCommandGuide(command, "codex")
		if err != nil {
			t.Fatalf("buildCommandGuide(%q): %v", command, err)
		}
		text := strings.Join(append(append([]string{}, guide.PreSteps...), append([]string{guide.RunCommand}, guide.PostSteps...)...), "\n")
		for _, want := range test.required {
			if !strings.Contains(text, want) {
				t.Errorf("%s command-guide missing TS host spine anchor %q", command, want)
			}
		}
		for _, forbidden := range test.retired {
			if strings.Contains(text, forbidden) {
				t.Errorf("%s command-guide still documents retired direct manifest path %q", command, forbidden)
			}
		}
	}
}

func TestCodexLifecycleGuidesDoNotDocumentRetiredHostFallbacks(t *testing.T) {
	forbidden := map[string][]string{
		"colonize": {
			"AETHER_OUTPUT_MODE=json aether colonize --plan-only",
		},
		"plan": {
			"AETHER_OUTPUT_MODE=json aether plan --plan-only --depth <choice>",
		},
		"build": {
			"AETHER_OUTPUT_MODE=json aether build <phase> --plan-only",
		},
		"continue": {
			"AETHER_OUTPUT_MODE=json aether continue --plan-only --verification-depth heavy",
		},
		"seal": {
			"AETHER_OUTPUT_MODE=json aether seal --plan-only",
		},
	}

	for command, needles := range forbidden {
		guide, err := buildCommandGuide(command, "codex")
		if err != nil {
			t.Fatalf("buildCommandGuide(%q): %v", command, err)
		}
		text := strings.Join(append(append([]string{}, guide.PreSteps...), append([]string{guide.RunCommand}, guide.PostSteps...)...), "\n")
		for _, needle := range needles {
			if strings.Contains(text, needle) {
				t.Errorf("%s command-guide still documents retired host fallback %q", command, needle)
			}
		}
	}
}

func TestWrapperSourcesUseTypeScriptHostManifestSpine(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	tests := map[string][]string{
		"colonize": {
			"aether host colonize",
			"colonize-finalize",
		},
		"plan": {
			"aether host plan",
			"--planning-depth",
			"TS host is the sole entry point",
			"plan-finalize",
		},
		"build": {
			"aether host build --dry-run",
			"build-finalize",
		},
		"continue": {
			"AETHER_OUTPUT_MODE=visual aether continue --verification-depth standard",
			"aether host continue --dry-run --classic-ceremony",
			"continue-finalize",
		},
		"seal": {
			"aether host seal",
			"seal-finalize",
		},
	}

	for command, anchors := range tests {
		files := []string{
			filepath.Join(repoRoot, ".aether", "commands", command+".yaml"),
			filepath.Join(repoRoot, ".claude", "commands", "ant", command+".md"),
			filepath.Join(repoRoot, ".opencode", "commands", "ant", command+".md"),
		}
		for _, path := range files {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			text := string(content)
			for _, want := range anchors {
				if !strings.Contains(text, want) {
					t.Errorf("%s missing TS host spine anchor %q", path, want)
				}
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
		"aether host colonize",
		"aether host plan --depth <choice> --planning-depth <choice>",
		"aether host build --dry-run <phase>",
		"AETHER_OUTPUT_MODE=visual aether continue --verification-depth standard",
		"aether host continue --dry-run --classic-ceremony",
		"aether host seal",
		"aether spawn-log",
		"aether spawn-complete",
		"aether ceremony worker-complete",
		"visible live Task/subagent panels",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("%s missing lifecycle worker activity anchor %q", path, want)
		}
	}
	for _, forbidden := range []string{
		"AETHER_OUTPUT_MODE=json aether plan --plan-only --depth <choice>",
		"AETHER_OUTPUT_MODE=json aether build <phase> --plan-only",
		"AETHER_OUTPUT_MODE=json aether continue --plan-only --verification-depth heavy",
	} {
		if strings.Contains(text, forbidden) {
			t.Errorf("%s still documents retired host fallback %q", path, forbidden)
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

func TestCodexGeneratedCommandShimsCoverIntelligentCommands(t *testing.T) {
	shims := map[string]codexSkillShim{}
	for _, shim := range codexSkillShims() {
		shims[shim.Name] = shim
	}
	catalog := commandGuideCatalog()
	for _, command := range []string{"init", "discuss", "oracle", "colonize", "plan", "build", "continue", "swarm", "seal"} {
		def := catalog[command]
		shim, ok := shims["aether-"+command]
		if !ok {
			t.Fatalf("codex generated shims missing command-shaped skill for %q", command)
		}
		if len(shim.WorkflowTriggers) != 1 || shim.WorkflowTriggers[0] != command {
			t.Fatalf("%s workflow triggers = %v, want [%s]", shim.Name, shim.WorkflowTriggers, command)
		}
		for _, want := range []string{
			"aether command-guide " + command + " --platform codex",
			"aether " + command,
			"/ant-" + command,
			"Raw Bypass",
		} {
			text := shim.Description + "\n" + shim.Body + "\n" + strings.Join(shim.TaskKeywords, "\n")
			if !strings.Contains(text, want) {
				t.Fatalf("%s command shim missing %q", shim.Name, want)
			}
		}
		if def.SkillReference != "" && !strings.Contains(shim.Body, def.SkillReference) {
			t.Fatalf("%s command shim missing lifecycle skill %q", shim.Name, def.SkillReference)
		}
		if def.RunCommand != "" && !strings.Contains(shim.Body, def.RunCommand) {
			t.Fatalf("%s command shim missing runtime command %q", shim.Name, def.RunCommand)
		}
		rendered := renderCodexSkillShim(shim)
		fm := parseSkillFrontmatter(rendered)
		if fm == nil {
			t.Fatalf("%s generated shim should have parseable frontmatter", shim.Name)
		}
		if !stringSliceContains(fm.TaskKeywords, "aether "+command) {
			t.Fatalf("%s generated frontmatter missing command keyword: %v", shim.Name, fm.TaskKeywords)
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

// --- Phase 144 Codex smoke tests (CLEAN-05) ---

func TestCommandGuideBuildSmoke(t *testing.T) {
	guide, err := buildCommandGuide("build", "codex")
	if err != nil {
		t.Fatalf("buildCommandGuide(build, codex) error: %v", err)
	}
	if guide.RunCommand == "" {
		t.Error("RunCommand should be non-empty")
	}
	if len(guide.PreSteps) < 1 {
		t.Errorf("PreSteps should have at least 1 entry, got %d", len(guide.PreSteps))
	}
	if len(guide.PostSteps) < 1 {
		t.Errorf("PostSteps should have at least 1 entry, got %d", len(guide.PostSteps))
	}
}

func TestCommandGuidePlanSmoke(t *testing.T) {
	guide, err := buildCommandGuide("plan", "codex")
	if err != nil {
		t.Fatalf("buildCommandGuide(plan, codex) error: %v", err)
	}
	if guide.RunCommand == "" {
		t.Error("RunCommand should be non-empty")
	}
	if len(guide.PreSteps) < 1 {
		t.Errorf("PreSteps should have at least 1 entry, got %d", len(guide.PreSteps))
	}
	if len(guide.PostSteps) < 1 {
		t.Errorf("PostSteps should have at least 1 entry, got %d", len(guide.PostSteps))
	}
}

// --- Phase 144 execution path audit (CLEAN-02) ---
// CLEAN-06 coverage note: cross-platform build ceremony alignment is verified
// by .aether/ts-host/test/cross-platform-parity.test.ts which checks that
// Claude and OpenCode build wrappers have matching ceremony invocation patterns
// and that dispatched command wrappers contain all 4 ceremony invocations.

type executionPathTest struct {
	name          string
	yamlFile      string
	wantHostCmd   string // runtime.command (or manifest_command/default_command) should contain this
	orchestration string // orchestration block should contain this (empty = no orchestration block expected)
}

func TestExecutionPathAudit_OneConductorPerWorkflow(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	tests := []executionPathTest{
		{
			name:        "build",
			yamlFile:    "build.yaml",
			wantHostCmd: "aether host build",
		},
		{
			name:        "plan",
			yamlFile:    "plan.yaml",
			wantHostCmd: "aether host plan",
		},
		{
			name:          "colonize",
			yamlFile:      "colonize.yaml",
			wantHostCmd:   "aether host colonize",
			orchestration: "aether host colonize",
		},
		{
			name:        "continue",
			yamlFile:    "continue.yaml",
			wantHostCmd: "aether continue",
		},
		{
			name:        "seal",
			yamlFile:    "seal.yaml",
			wantHostCmd: "aether host seal",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			yamlPath := filepath.Join(repoRoot, ".aether", "commands", tc.yamlFile)
			content, err := os.ReadFile(yamlPath)
			if err != nil {
				t.Fatalf("read %s: %v", yamlPath, err)
			}
			yamlText := string(content)

			// Assert runtime.command (or manifest_command/default_command) field
			// contains the expected host command. The YAML schema evolved: some
			// commands use runtime.command, others use runtime.manifest_command +
			// runtime.default_command (the single-conductor pattern).
			var meta struct {
				Runtime struct {
					Command         string `yaml:"command"`
					ManifestCommand string `yaml:"manifest_command"`
					DefaultCommand  string `yaml:"default_command"`
				} `yaml:"runtime"`
			}
			if err := yaml.Unmarshal(content, &meta); err != nil {
				t.Fatalf("parse %s: %v", yamlPath, err)
			}

			// Collect all runtime command references
			runtimeCmds := []string{meta.Runtime.Command, meta.Runtime.ManifestCommand, meta.Runtime.DefaultCommand}
			found := false
			for _, cmd := range runtimeCmds {
				if cmd != "" && strings.Contains(cmd, tc.wantHostCmd) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("%s: no runtime command field contains %q (got command=%q manifest_command=%q default_command=%q)",
					tc.name, tc.wantHostCmd, meta.Runtime.Command, meta.Runtime.ManifestCommand, meta.Runtime.DefaultCommand)
			}

			// For colonize: verify orchestration block references the host command
			if tc.orchestration != "" {
				if !strings.Contains(yamlText, tc.orchestration) {
					t.Errorf("%s: YAML missing orchestration reference %q", tc.name, tc.orchestration)
				}
			}

			// For continue: verify both default and heavy-review paths
			if tc.name == "continue" {
				if !strings.Contains(yamlText, "aether host continue --dry-run --classic-ceremony") {
					t.Errorf("%s: YAML missing heavy-review path reference", tc.name)
				}
			}

			// Each workflow should produce non-empty Codex command-guide output
			guide, err := buildCommandGuide(tc.name, "codex")
			if err != nil {
				t.Fatalf("buildCommandGuide(%q, codex): %v", tc.name, err)
			}
			if guide.RunCommand == "" {
				t.Errorf("%s: Codex command-guide RunCommand should be non-empty", tc.name)
			}
		})
	}
}
