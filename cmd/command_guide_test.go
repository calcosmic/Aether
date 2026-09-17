package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
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
			"aether host plan --preset <fast|balanced|deep|exhaustive>",
			"visible live Task/subagent panels",
			"aether spawn-log",
			"aether spawn-complete",
			"ceremony worker-complete --workflow plan",
			"AETHER_OUTPUT_MODE=json aether plan-finalize",
		},
		"build": {
			"aether build <phase> --plan-only",
			"Parse `result.dispatch_manifest`",
			"spawn_agent",
			"codex-native-worker observe",
			"records actual running, unavailable or launch_unresolved evidence",
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
	for _, surface := range []struct{ command, platform string }{
		{"colonize", "codex"}, {"plan", "codex"}, {"continue", "codex"}, {"seal", "codex"},
		{"build", "claude"}, {"build", "opencode"},
	} {
		guide, err := buildCommandGuide(surface.command, surface.platform)
		if err != nil {
			t.Fatalf("buildCommandGuide(%q, %q): %v", surface.command, surface.platform, err)
		}
		text := strings.Join(append(append([]string{}, guide.PreSteps...), append([]string{guide.RunCommand}, guide.PostSteps...)...), "\n")
		for _, want := range []string{
			"${TMPDIR:-/tmp}/aether-<workflow>-<run>/<workflow>-completion.json",
			"never write wrapper result artifacts under `.aether/data`",
			"<approved temp completion JSON>",
		} {
			if !strings.Contains(text, want) {
				t.Errorf("%s/%s command-guide missing approved temp completion contract %q", surface.platform, surface.command, want)
			}
		}
	}
	guide, err := buildCommandGuide("build", "codex")
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Join(append(append([]string{}, guide.PreSteps...), append([]string{guide.RunCommand}, guide.PostSteps...)...), "\n")
	for _, want := range []string{
		"strict JSON requests", "absolute regular file", "aether-worker-request-*", "system temporary directory",
		"aether codex-native-worker record --request", "aether codex-native-worker stage --request",
		"result.completion_path", "--completion-file <Go-owned completion_path returned by codex-native-worker stage>",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("codex/build guide missing native request/journal completion contract %q", want)
		}
	}
	for _, forbidden := range []string{"<approved temp completion JSON>", "<workflow>-completion.json", "aether build-completion-stage"} {
		if strings.Contains(text, forbidden) {
			t.Errorf("codex/build guide still teaches parent-authored aggregate completion through %q", forbidden)
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
				"aether host plan --preset <fast|balanced|deep|exhaustive>",
				"result.plan_manifest.stage_manifest",
				"AETHER_OUTPUT_MODE=json aether plan-finalize",
			},
			retired: []string{
				"AETHER_OUTPUT_MODE=json aether plan --plan-only --depth <choice>",
				"result.planning_manifest",
				"result.depth_proposal_card",
				"result.research_proposal_card",
				"result.requires_next_iteration",
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
				"aether build <phase> --plan-only",
				"Parse `result.dispatch_manifest`",
				"AETHER_OUTPUT_MODE=json aether build-finalize",
			},
			retired: []string{
				"aether host build --dry-run <phase>",
				"Parse `result.manifest.dispatch_manifest`",
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
			"aether build $ARGUMENTS --plan-only",
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
			guideAnchors := []string{anchor}
			if command == "build" && anchor == "visible live Task/subagent" {
				// Native Codex activity uses observed host children; the YAML and
				// primary platform guides retain their Task/subagent contract.
				guideAnchors = []string{"spawn_agent", "codex-native-worker observe", "records actual running, unavailable or launch_unresolved evidence"}
				for _, platform := range []string{"claude", "opencode"} {
					primary, err := buildCommandGuide(command, platform)
					if err != nil {
						t.Fatal(err)
					}
					if !strings.Contains(strings.Join(primary.PreSteps, "\n"), anchor) {
						t.Errorf("%s build guide lost worker activity anchor %q", platform, anchor)
					}
				}
			}
			for _, want := range guideAnchors {
				if !strings.Contains(guideText, want) {
					t.Errorf("%s command-guide missing worker activity anchor %q", command, want)
				}
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
		"aether host plan --preset <fast|balanced|deep|exhaustive>",
		"aether build <phase> --plan-only",
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
		"result.planning_manifest",
		"result.depth_proposal_card",
		"result.research_proposal_card",
		"result.requires_next_iteration",
		"plan-research-approve",
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

func TestCommandGuideInit199(t *testing.T) {
	guide, err := buildCommandGuide("init", "codex")
	if err != nil {
		t.Fatalf("buildCommandGuide(init, codex): %v", err)
	}
	if guide.Intent != frontDoorInitDescription199 {
		t.Errorf("init guide intent = %q, want %q", guide.Intent, frontDoorInitDescription199)
	}
	if guide.Category != commandGuideCategoryFullOrchestration || guide.SkillReference != commandGuideSkillCreation || guide.Literal {
		t.Errorf("init guide lost Codex orchestration identity: %#v", guide)
	}
	wantRun := "AETHER_OUTPUT_MODE=visual aether init --colony-mode <selected colony|orchestrator> --charter-json '<synthesized charter JSON>' \"<refined goal>\""
	if guide.RunCommand != wantRun {
		t.Errorf("init guide runtime command = %q, want %q", guide.RunCommand, wantRun)
	}

	parts := append([]string{guide.Intent}, guide.PreSteps...)
	parts = append(parts, guide.RunCommand)
	parts = append(parts, guide.PostSteps...)
	parts = append(parts, guide.DriftGuards...)
	parts = append(parts, guide.RawBypass)
	text := strings.Join(parts, "\n")
	assertFrontDoorInitContract199(t, text)
	if !strings.Contains(text, "Next Up: $ant-plan") {
		t.Errorf("Codex guide does not render exact native closeout:\n%s", text)
	}
	for _, forbidden := range []string{"/ant-plan", "$ant-spec", "$ant-status", "aether lay-eggs"} {
		if strings.Contains(text, forbidden) {
			t.Errorf("Codex guide contains deferred or host-inappropriate guidance %q", forbidden)
		}
	}
	assertNoDirectInitStateWrites199(t, text)
	if !strings.Contains(guide.RawBypass, "raw") || !strings.Contains(guide.RawBypass, "aether init") {
		t.Errorf("init guide lost raw aether init bypass: %q", guide.RawBypass)
	}

	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("find repository root: %v", err)
	}
	skillPath := filepath.Join(repoRoot, ".aether", "skills", "colony", commandGuideSkillCreation, "SKILL.md")
	rawSkill, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read Codex creation skill: %v", err)
	}
	skillText := strings.ReplaceAll(string(rawSkill), "\r\n", "\n")
	assertFrontDoorInitContract199(t, skillText)
	if !strings.Contains(skillText, "aether command-guide init --platform codex") || !strings.Contains(skillText, "aether init") {
		t.Error("Codex creation skill no longer orchestrates raw aether init through command-guide")
	}
	if !strings.HasSuffix(strings.TrimSpace(skillText), "Next Up: $ant-plan") {
		t.Errorf("Codex creation skill does not close with exact $ant-plan:\n%s", skillText)
	}
	for _, forbidden := range []string{"/ant-plan", "$ant-spec", "$ant-status", "aether lay-eggs"} {
		if strings.Contains(skillText, forbidden) {
			t.Errorf("Codex creation skill contains deferred native lifecycle vocabulary %q", forbidden)
		}
	}
	assertNoDirectInitStateWrites199(t, skillText)
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

func TestCommandGuideSpecRuntimeNative(t *testing.T) {
	guide, err := buildCommandGuide("spec", "codex")
	if err != nil {
		t.Fatalf("buildCommandGuide(spec, codex): %v", err)
	}
	if guide.Category != commandGuideCategoryRuntimeNative || !guide.Literal || guide.SkillReference != "" {
		t.Fatalf("spec guide lost runtime-native identity: %#v", guide)
	}
	if guide.RunCommand != "AETHER_OUTPUT_MODE=visual aether spec $ARGUMENTS" {
		t.Errorf("spec guide runtime command = %q", guide.RunCommand)
	}
	text := strings.Join(append(append([]string{guide.Intent, guide.RunCommand, guide.RawBypass}, guide.DriftGuards...), guide.PreSteps...), "\n")
	for _, want := range []string{"nine typed body categories", "revision/hash", "approval token", "projection repair", "planning stop", "candidate acceptance", "aether spec", "/ant-spec"} {
		if !strings.Contains(text, want) {
			t.Errorf("spec command-guide missing %q", want)
		}
	}
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	meta := readCommandGuideYAMLMetadata(t, filepath.Join(repoRoot, ".aether", "commands", "spec.yaml"))
	if meta.CodexOrchestration.Category != guide.Category || meta.CodexOrchestration.Skill != "" {
		t.Errorf("spec YAML/runtime-native guide identity drifted: YAML=%#v guide=%#v", meta.CodexOrchestration, guide)
	}
}

func TestCommandGuideLifecycle199(t *testing.T) {
	const statusBeforeOptionalEntomb = "After sealing, run `AETHER_OUTPUT_MODE=visual aether status` first to review the retained sealed state; `aether entomb` is a separate optional owner-confirmed archive-and-clear action."

	guides := make(map[string]commandGuideResult)
	for _, command := range []string{"pause", "resume", "seal", "entomb"} {
		guide, err := buildCommandGuide(command, "codex")
		if err != nil {
			t.Fatalf("buildCommandGuide(%q, codex): %v", command, err)
		}
		guides[command] = guide
	}
	if guides["pause"].RunCommand != "AETHER_OUTPUT_MODE=visual aether pause $ARGUMENTS" {
		t.Errorf("pause guide does not delegate to the canonical runtime: %q", guides["pause"].RunCommand)
	}
	if guides["resume"].RunCommand != "AETHER_OUTPUT_MODE=visual aether resume $ARGUMENTS" {
		t.Errorf("resume guide does not delegate to the canonical runtime: %q", guides["resume"].RunCommand)
	}

	var guideParts []string
	for _, command := range []string{"pause", "resume", "seal", "entomb"} {
		guide := guides[command]
		guideParts = append(guideParts, guide.Intent)
		guideParts = append(guideParts, guide.PreSteps...)
		guideParts = append(guideParts, guide.RunCommand)
		guideParts = append(guideParts, guide.PostSteps...)
		guideParts = append(guideParts, guide.DriftGuards...)
	}
	guideText := strings.Join(guideParts, "\n")
	for _, required := range []string{
		"`aether pause` and `aether resume`",
		"Confirmed", "Reconstructed", "Conflicting", "Unknown",
		"state effect none", "must not inspect, select, or edit",
		statusBeforeOptionalEntomb,
		"Never invoke entomb automatically",
	} {
		if !strings.Contains(guideText, required) {
			t.Errorf("Codex lifecycle command-guide lacks %q", required)
		}
	}

	for _, retired := range []string{"recover", "resume-colony", "pause-colony"} {
		if _, err := buildCommandGuide(retired, "codex"); err == nil {
			t.Errorf("retired public lifecycle route %q is still documented by command-guide", retired)
		}
	}

	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("find repository root: %v", err)
	}
	skillPath := filepath.Join(repoRoot, ".aether", "skills", "colony", "colony-lifecycle", "SKILL.md")
	rawSkill, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read lifecycle skill: %v", err)
	}
	skillText := strings.ReplaceAll(string(rawSkill), "\r\n", "\n")
	for _, required := range []string{
		"`aether pause` and `aether resume`",
		"Confirmed", "Reconstructed", "Conflicting", "Unknown",
		"state effect none", "must not inspect, select, or edit",
		statusBeforeOptionalEntomb,
		"Never invoke entomb automatically",
	} {
		if !strings.Contains(skillText, required) {
			t.Errorf("existing Codex lifecycle skill lacks %q", required)
		}
	}
	for _, forbidden := range []string{
		"aether recover", "/ant-recover", "resume-colony", "pause-colony", "Create Codex-native `$ant-*` skills",
		"After seal, suggest entomb", "after seal automatically", "automatically invoke entomb",
		"write COLONY_STATE.json", "edit session.json", "remove HANDOFF.md",
	} {
		if strings.Contains(skillText, forbidden) {
			t.Errorf("existing Codex lifecycle skill contains retired/host-owned lifecycle guidance %q", forbidden)
		}
	}
}

func TestCommandGuideBuildCycle199(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("find repository root: %v", err)
	}

	var guideParts []string
	for _, command := range []string{"plan", "build", "seal"} {
		guide, err := buildCommandGuide(command, "codex")
		if err != nil {
			t.Fatalf("build %s command guide: %v", command, err)
		}
		guideParts = append(guideParts, guide.Intent)
		guideParts = append(guideParts, guide.PreSteps...)
		guideParts = append(guideParts, guide.PostSteps...)
	}
	guideText := strings.Join(guideParts, "\n")
	skillPath := filepath.Join(repoRoot, ".aether", "skills", "colony", commandGuideSkillBuildCycle, "SKILL.md")
	skill, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read %s: %v", skillPath, err)
	}

	for label, text := range map[string]string{"command guide": guideText, "build-cycle skill": string(skill)} {
		for _, required := range []string{
			"automatic typed territory freshness",
			"equal guided-build/Autopilot choice",
			"displayed Autopilot bounds",
			"concrete repair/debt receipts",
			"independent safe-path continuation",
			"explicit seal",
			"Full workflow coverage and native-worker parity remain pending.",
			"Force flags pass only when directly supplied by the owner.",
			"A forced-incomplete closure is not verified success.",
		} {
			if !strings.Contains(text, required) {
				t.Errorf("%s missing %q", label, required)
			}
		}
		for _, forbidden := range []string{"auto-seal", "auto-entomb", "Codex-native `$ant-*` skill"} {
			if strings.Contains(text, forbidden) {
				t.Errorf("%s must not introduce %q", label, forbidden)
			}
		}
	}
}

func TestInsertPhaseCommandGuideDocumentsGuidedAndExplicitForms(t *testing.T) {
	guide, err := buildCommandGuide("insert-phase", "codex")
	if err != nil {
		t.Fatalf("buildCommandGuide(insert-phase): %v", err)
	}
	if !guide.Literal || guide.Category != commandGuideCategoryLiteral {
		t.Fatalf("insert-phase guide must remain literal passthrough: %#v", guide)
	}
	if guide.RunCommand != "AETHER_OUTPUT_MODE=visual aether insert-phase $ARGUMENTS" {
		t.Fatalf("insert-phase run command = %q, want direct visual runtime delegation", guide.RunCommand)
	}

	text := strings.Join(append(append([]string{guide.Intent, guide.RunCommand}, guide.PreSteps...), guide.DriftGuards...), "\n")
	for _, want := range []string{
		"one issue sentence",
		`aether insert-phase "problem to stabilise"`,
		"non-interactive automation",
		`aether insert-phase --after 2 --name "Stabilize login retries" --description "login retries lose state" --constraints "do not change the provider"`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("insert-phase command-guide missing %q", want)
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

func TestCodexAntSkillGuidesPreserveOtherPlatforms(t *testing.T) {
	for _, platform := range []string{"claude", "opencode"} {
		for _, expected := range codexAntGuideExpectations {
			t.Run(platform+"/"+expected.command, func(t *testing.T) {
				guide, err := buildCommandGuide(expected.command, platform)
				if err != nil {
					t.Fatal(err)
				}
				if guide.Command != expected.command || guide.RunCommand != expected.runtime || guide.SkillReference != "" {
					t.Errorf("platform route changed: %+v", guide)
				}
				text := strings.Join(guide.PreSteps, "\n") + strings.Join(guide.PostSteps, "\n")
				if strings.Contains(text, "$ant-") || strings.Contains(text, "../support/") {
					t.Error("Codex-only public/support spelling leaked to another platform")
				}
				if !strings.Contains(guide.PreSteps[0], "slash-command wrapper") {
					t.Error("other platform lost its wrapper route")
				}
			})
		}
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

// Independently specified public names, internal identities, and executable routes.
var codexAntGuideExpectations = []struct{ command, support, runtime string }{
	{"init", "aether-colony-creation", `AETHER_OUTPUT_MODE=visual aether init --colony-mode <selected colony|orchestrator> --charter-json '<synthesized charter JSON>' "<refined goal>"`},
	{"discuss", "aether-colony-research", `AETHER_OUTPUT_MODE=visual aether discuss $ARGUMENTS`},
	{"oracle", "aether-colony-research", `AETHER_OUTPUT_MODE=visual aether oracle --depth <depth> --confidence-target <percent> --template <template> --background "<synthesized prompt>"`},
	{"colonize", "aether-colony-build-cycle", `AETHER_OUTPUT_MODE=json aether colonize-finalize --completion-file <approved temp completion JSON>`},
	{"plan", "aether-colony-build-cycle", `AETHER_OUTPUT_MODE=json aether plan-finalize --completion-file <approved temp completion JSON>`},
	{"build", "aether-colony-build-cycle", `AETHER_OUTPUT_MODE=json aether build-finalize <phase> --completion-file <Go-owned completion_path returned by build-completion-stage>`},
	{"continue", "aether-colony-build-cycle", `AETHER_OUTPUT_MODE=visual aether continue --verification-depth standard $ARGUMENTS`},
	{"swarm", "aether-colony-build-cycle", `AETHER_OUTPUT_MODE=json aether swarm-finalize --completion-file <worker completion JSON>`},
	{"seal", "aether-colony-build-cycle", `AETHER_OUTPUT_MODE=json aether seal-finalize --completion-file <approved temp completion JSON>`},
}

func TestCodexGeneratedShimsIncludeCommandGuideSkills(t *testing.T) {
	payload, err := buildCodexSkillPayload(antSkillSourceRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	supportFiles := map[string][]byte{}
	for _, file := range payload.Files {
		if strings.HasPrefix(file.RelativePath, "support/") {
			supportFiles[file.RelativePath] = file.Content
		}
	}
	if len(supportFiles) != 3 {
		t.Fatalf("private supports = %d, want 3", len(supportFiles))
	}
	for _, skill := range []string{"aether-colony-creation", "aether-colony-research", "aether-colony-build-cycle"} {
		source, err := os.ReadFile(filepath.Join(antSkillSourceRoot(t), ".aether", "skills", "colony", skill, "SKILL.md"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(supportFiles["support/"+skill+".md"], source) {
			t.Errorf("private support %s is not the full source", skill)
		}
	}
	for _, shim := range codexSkillShims() {
		if !strings.HasPrefix(shim.Name, "ant-") {
			t.Errorf("extra public helper %q", shim.Name)
		}
	}
	for _, want := range []string{"Colony Mode", "Orchestrator Mode", "--colony-mode"} {
		if !bytes.Contains(supportFiles["support/aether-colony-creation.md"], []byte(want)) {
			t.Errorf("creation support missing %q", want)
		}
	}
	buildCycle := string(supportFiles["support/aether-colony-build-cycle.md"])
	fm := parseSkillFrontmatter(buildCycle)
	if fm == nil {
		t.Fatal("private source lost frontmatter")
	}
	wantTriggers := []string{"colonize", "plan", "build", "continue", "swarm", "seal"}
	if !reflect.DeepEqual(fm.WorkflowTriggers, wantTriggers) {
		t.Errorf("build-cycle triggers = %v", fm.WorkflowTriggers)
	}
	for _, want := range []string{"aether colonize", "aether plan", "aether build", "aether continue", "aether swarm", "aether seal", "dispatch manifest", "plan-only", "finalize"} {
		if !stringSliceContains(fm.TaskKeywords, want) {
			t.Errorf("private build-cycle keywords missing %q", want)
		}
	}
}

func TestCodexGeneratedCommandShimsCoverIntelligentCommands(t *testing.T) {
	shims := map[string]codexSkillShim{}
	for _, shim := range codexSkillShims() {
		if _, exists := shims[shim.Name]; exists {
			t.Fatalf("duplicate shim %s", shim.Name)
		}
		shims[shim.Name] = shim
	}
	if len(shims) != 9 {
		t.Fatalf("public shim count = %d, want exactly nine", len(shims))
	}
	for _, expected := range codexAntGuideExpectations {
		if expected.command == "build" {
			expected.runtime = "AETHER_OUTPUT_MODE=json aether build-finalize <phase> --completion-file <Go-owned completion_path returned by codex-native-worker stage>"
		}
		t.Run(expected.command, func(t *testing.T) {
			name := "ant-" + expected.command
			shim, ok := shims[name]
			if !ok || shim.Dir != name {
				t.Fatalf("missing canonical public skill %s", name)
			}
			if !reflect.DeepEqual(shim.WorkflowTriggers, []string{name, expected.command}) {
				t.Errorf("%s triggers = %v", name, shim.WorkflowTriggers)
			}
			text := shim.Description + "\n" + shim.Body + "\n" + strings.Join(shim.TaskKeywords, "\n")
			for _, want := range []string{"$" + name, "aether command-guide " + expected.command + " --platform codex", expected.runtime, "../support/" + expected.support + ".md", "Raw Bypass", "raw/exact/no-orchestration", "must not hand-edit `.aether/data`"} {
				if !strings.Contains(text, want) {
					t.Errorf("%s missing %q", name, want)
				}
			}
			fm := parseSkillFrontmatter(renderCodexSkillShim(shim))
			if fm == nil || fm.Name != name || !stringSliceContains(fm.TaskKeywords, "aether "+expected.command) {
				t.Fatalf("bad frontmatter: %+v", fm)
			}
			guide, err := buildCommandGuide(expected.command, "codex")
			if err != nil {
				t.Fatal(err)
			}
			if guide.Command != expected.command || guide.RunCommand != expected.runtime || guide.SkillReference != expected.support || guide.Literal {
				t.Errorf("guide/runtime identity = %+v", guide)
			}
		})
	}
}

var codexGuideSupportReference = regexp.MustCompile("`(\\.\\./support/[a-z-]+\\.md)`")

func resolveInstalledCodexGuideSupport(skillPath, text string) error {
	refs := codexGuideSupportReference.FindAllStringSubmatch(text, -1)
	if len(refs) == 0 {
		return fmt.Errorf("no private support references in %s", skillPath)
	}
	for _, ref := range refs {
		path := filepath.Join(filepath.Dir(skillPath), filepath.FromSlash(ref[1]))
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if len(bytes.TrimSpace(content)) == 0 {
			return fmt.Errorf("empty private support: %s", path)
		}
	}
	return nil
}

func TestCodexAntSkillGuideSupport(t *testing.T) {
	home := t.TempDir()
	if ok, output := runAntSkillInstall(t, home); !ok {
		t.Fatalf("install failed: %s", output)
	}
	root := filepath.Join(home, ".codex", "skills", "aether")
	var names []string
	for _, dir := range findSkillDirs(root) {
		names = append(names, filepath.Base(dir))
	}
	sort.Strings(names)
	if !reflect.DeepEqual(names, expectedAntSkills) {
		t.Fatalf("installed public menu = %v", names)
	}
	for _, expected := range codexAntGuideExpectations {
		if expected.command == "build" {
			expected.runtime = "AETHER_OUTPUT_MODE=json aether build-finalize <phase> --completion-file <Go-owned completion_path returned by codex-native-worker stage>"
		}
		t.Run(expected.command, func(t *testing.T) {
			var output bytes.Buffer
			stdout, stderr = &output, &output
			rootCmd.SetArgs([]string{"command-guide", expected.command, "--platform", "codex"})
			if err := rootCmd.Execute(); err != nil {
				t.Fatal(err)
			}
			var envelope struct {
				OK     bool               `json:"ok"`
				Result commandGuideResult `json:"result"`
			}
			if err := json.Unmarshal(output.Bytes(), &envelope); err != nil || !envelope.OK {
				t.Fatalf("guide output: %v: %s", err, output.String())
			}
			guide := envelope.Result
			if guide.Command != expected.command || guide.RunCommand != expected.runtime || guide.SkillReference != expected.support {
				t.Fatalf("runtime route changed: %+v", guide)
			}
			guideText := strings.Join(guide.PreSteps, "\n") + "\n" + strings.Join(guide.PostSteps, "\n") + "\n" + strings.Join(guide.DriftGuards, "\n")
			if !strings.Contains(guideText, "$ant-"+expected.command) {
				t.Errorf("guide does not teach public invocation")
			}
			skillPath := filepath.Join(root, "ant-"+expected.command, "SKILL.md")
			skill, err := os.ReadFile(skillPath)
			if err != nil {
				t.Fatal(err)
			}
			for label, text := range map[string]string{"guide": guideText, "installed skill": string(skill)} {
				if err := resolveInstalledCodexGuideSupport(skillPath, text); err != nil {
					t.Errorf("%s support: %v", label, err)
				}
				for _, forbidden := range []string{"Load the aether-", "`aether-colony-creation` Codex skill", "`aether-colony-research` Codex skill", "`aether-colony-build-cycle` Codex skill", "$ant-status", "$ant-help", "$ant-spec", "$ant-maintenance", "deferred Codex-native lifecycle surface", "scope fence remains intact"} {
					if strings.Contains(text, forbidden) {
						t.Errorf("%s advertises obsolete guidance %q", label, forbidden)
					}
				}
			}
			t.Logf("%s: route=%s support=../support/%s.md", expected.command, guide.RunCommand, expected.support)
		})
	}
	t.Run("missing private support refuses resolution", func(t *testing.T) {
		path := filepath.Join(root, "support", "aether-colony-build-cycle.md")
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		guide, err := buildCommandGuide("plan", "codex")
		if err != nil {
			t.Fatal(err)
		}
		err = resolveInstalledCodexGuideSupport(filepath.Join(root, "ant-plan", "SKILL.md"), strings.Join(guide.PreSteps, "\n"))
		if !os.IsNotExist(err) {
			t.Fatalf("removed support must fail resolution: %v", err)
		}
	})
}

func stringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// yamlCommandNamesForGuideTest returns every command name a YAML source
// legitimises: each file's own name, plus every alias its `aliases:` field
// declares. A declared alias (e.g. pause-colony) has no YAML file of its
// own -- it is declared once, in its canonical command's YAML -- but it is
// still a legitimate wrapper/guide name, exactly like a name that does have
// its own file. Every consumer of this list (guide parity, wrapper parity,
// platform parity) needs that expansion, so it lives here once rather than
// being reimplemented per caller.
func yamlCommandNamesForGuideTest(t *testing.T) []string {
	t.Helper()
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}
	commandsDir := filepath.Join(repoRoot, ".aether", "commands")
	entries, err := os.ReadDir(commandsDir)
	if err != nil {
		t.Fatalf("read .aether/commands: %v", err)
	}
	names := map[string]bool{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".yaml")
		names[name] = true

		data, err := os.ReadFile(filepath.Join(commandsDir, entry.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		var spec sourceCheckCommandSpec
		if err := yaml.Unmarshal(data, &spec); err != nil {
			t.Fatalf("parse %s: %v", entry.Name(), err)
		}
		for _, alias := range spec.Aliases {
			alias = strings.TrimSpace(alias)
			if alias != "" {
				names[alias] = true
			}
		}
	}
	result := make([]string, 0, len(names))
	for name := range names {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
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

var phase200PlanGuidanceAnchors = []string{
	"AETHER_OUTPUT_MODE=json aether spec --inspect",
	"Fast 80/up to 4",
	"Balanced 90/up to 6",
	"Deep 95/up to 8",
	"Exhaustive 99/up to 12",
	"aether host plan --preset <fast|balanced|deep|exhaustive>",
	"result.plan_manifest.stage_manifest",
	"planning-scout-result/v1",
	"planning-route-setter-result/v1",
	"route_stage_manifest",
	"scout_stage_manifest",
	"decision_cards",
	"direct_resume",
	"successor_spec_required",
	"iteration_card",
	"evidence_that_would_change",
	"NOT ACTIVE",
	"AETHER_OUTPUT_MODE=json aether plan --candidate",
	"acceptance_command",
	"acceptance_receipt",
	"exact replay",
	"divergent replay",
	"Go validates",
	"never synthesize",
	"$ant-build 1",
	"aether run",
}

var phase200RetiredPlanGuidance = []string{
	"result.planning_manifest",
	"result.depth_proposal_card",
	"result.research_proposal_card",
	"result.requires_next_iteration",
	"plan-research-approve",
	"aether host plan --depth <choice> --planning-depth <choice>",
	"Scout `scout_report`",
	"Route-Setter `phase_plan`",
	"explicit `--accept`",
}

func TestCommandGuidePlan200StagedAuthority(t *testing.T) {
	guide, err := buildCommandGuide("plan", "codex")
	if err != nil {
		t.Fatalf("buildCommandGuide(plan, codex): %v", err)
	}
	parts := append([]string{guide.Intent}, guide.PreSteps...)
	parts = append(parts, guide.RunCommand)
	parts = append(parts, guide.PostSteps...)
	parts = append(parts, guide.DriftGuards...)
	text := strings.Join(parts, "\n")
	for _, want := range phase200PlanGuidanceAnchors {
		if !strings.Contains(strings.ToLower(text), strings.ToLower(want)) {
			t.Errorf("Codex plan guide missing Phase 200 authority marker %q", want)
		}
	}
	for _, retired := range phase200RetiredPlanGuidance {
		if strings.Contains(text, retired) {
			t.Errorf("Codex plan guide still teaches retired planning contract %q", retired)
		}
	}
	if strings.Contains(text, "/ant-") || strings.Contains(text, "$ant-spec") || strings.Contains(text, "$ant-maintenance") {
		t.Errorf("Codex plan guide advertises unsupported spelling:\n%s", text)
	}
	if !strings.Contains(text, "$ant-plan") || !strings.Contains(text, "$ant-build 1") {
		t.Error("plan guide missing supported ant invocations")
	}
	if guide.RunCommand != "AETHER_OUTPUT_MODE=json aether plan-finalize --completion-file <approved temp completion JSON>" {
		t.Errorf("plan finalizer command = %q", guide.RunCommand)
	}
}

func TestPhase200BuildCycleSkill(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	path := filepath.Join(repoRoot, ".aether", "skills", "colony", commandGuideSkillBuildCycle, "SKILL.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	text := string(raw)
	for _, want := range phase200PlanGuidanceAnchors {
		if !strings.Contains(strings.ToLower(text), strings.ToLower(want)) {
			t.Errorf("build-cycle skill missing Phase 200 authority marker %q", want)
		}
	}
	for _, retired := range phase200RetiredPlanGuidance {
		if strings.Contains(text, retired) {
			t.Errorf("build-cycle skill still teaches retired planning contract %q", retired)
		}
	}
	if strings.Contains(text, "/ant-") {
		t.Errorf("Codex build-cycle skill must use direct aether command spelling")
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
			wantHostCmd: "aether build $ARGUMENTS --plan-only",
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

func TestBuildCommandGuideIncludesPlatformContract(t *testing.T) {
	guide, err := buildCommandGuide("build", "opencode")
	if err != nil {
		t.Fatalf("buildCommandGuide: %v", err)
	}
	if guide.PlatformContract.Platform != "opencode" {
		t.Fatalf("platform contract = %#v", guide.PlatformContract)
	}
	if guide.PlatformContract.WorkerDispatch.Level != "limited" {
		t.Fatalf("worker dispatch support = %#v", guide.PlatformContract.WorkerDispatch)
	}
	if guide.PlatformContract.NamedCasteRouting.Level != "limited" {
		t.Fatalf("named caste routing support = %#v", guide.PlatformContract.NamedCasteRouting)
	}
}

// --- Phase 195 coherent-job parity contract (shared by three guards) ---
//
// The build lifecycle now has ONE contract spread across five shipped
// surfaces: .aether/commands/build.yaml (canonical YAML), cmd/command_guide.go
// (Codex guide), the aether-colony-build-cycle Codex skill, and the three
// byte-identical build wrappers. AGENTS.md requires them to change together.
// These three vars are the single definition of "what the contract says", so
// TestCommandGuideBuildCoherentJobsContract,
// TestBuildCommandYAMLCoherentJobsParity, and
// TestLifecycleWrappersCarryCoherentJobContract cannot drift apart by each
// re-typing their own list.

// buildCoherentJobFieldAnchors are the runtime flag and field names every
// parity-critical build surface must name. They are the real CLI flags and
// real JSON keys the Go runtime emits (cmd/codex_workflow_cmds.go,
// cmd/coherent_jobs.go, cmd/codex_build.go, cmd/coherent_job_retry.go), not
// invented vocabulary — a surface missing one is a surface that cannot
// describe what the runtime actually does.
var buildCoherentJobFieldAnchors = []string{
	"--job-proposal",
	"job_decisions",
	"job_name",
	"job_reason",
	"job_source",
	"covered_task_ids",
	"task_receipts",
	"completed_task_ids",
	"recovery_job",
	"unfinished_task_ids",
	"recovery_command",
	"--checkin",
	"checkin_requested",
	"checkin_reason",
	"checkin_summary",
}

// buildCoherentJobAuthorityAnchors are the three verbatim sentences that fix
// the Go-authority boundary identically on every surface. They are compared
// byte-for-byte deliberately: a paraphrase on one platform is exactly how
// this repo has previously shipped four surfaces that each described a
// slightly different contract.
var buildCoherentJobAuthorityAnchors = []string{
	"Go owns accepted groups, completion credit, retry, worktree reconciliation, and check-in policy; the wrapper proposes, renders, spawns, and submits.",
	"An accepted task receipt is admission, not completion credit: only the runtime's root-backed finalization can grant `completed_task_ids`.",
	"Never author `covered_task_ids` or `completed_task_ids` by hand in a manifest or in colony state; the runtime owns both.",
}

// buildCoherentJobForbiddenAnchors are instructions no build surface may
// carry. The first is the pre-195 claim that a false `checkin_requested`
// means `--no-checkin` was passed — false since D-11, because the automatic
// one-worker fast path also produces false and requires the compact summary
// to be rendered instead of the stage being skipped silently. The rest forbid
// teaching any platform to author task-credit fields itself.
var buildCoherentJobForbiddenAnchors = []string{
	"(`--no-checkin` was passed)",
	"write `completed_task_ids`",
	"write `covered_task_ids`",
	"set `completed_task_ids`",
	"set `covered_task_ids`",
}

// assertBuildCoherentJobContract runs the full field/authority/forbidden
// contract against one surface's text.
func assertBuildCoherentJobContract(t *testing.T, surface, text string) {
	t.Helper()

	if strings.TrimSpace(text) == "" {
		t.Fatalf("%s: empty surface text — this guard would pass vacuously", surface)
	}
	for _, anchor := range buildCoherentJobFieldAnchors {
		if !strings.Contains(text, anchor) {
			t.Errorf("%s: missing coherent-job contract item %q", surface, anchor)
		}
	}
	for _, anchor := range buildCoherentJobAuthorityAnchors {
		if !strings.Contains(text, anchor) {
			t.Errorf("%s: missing verbatim Go-authority statement:\n  %s", surface, anchor)
		}
	}
	for _, forbidden := range buildCoherentJobForbiddenAnchors {
		if strings.Contains(text, forbidden) {
			t.Errorf("%s: still carries a forbidden instruction %q", surface, forbidden)
		}
	}
}

// TestCommandGuideBuildCoherentJobsContract asserts the Codex build guide
// describes the same coherent-job, receipt, recovery and check-in contract
// the runtime implements, on every platform the guide serves.
func TestCommandGuideBuildCoherentJobsContract(t *testing.T) {
	for _, platform := range []string{"codex", "claude", "opencode"} {
		platform := platform
		t.Run(platform, func(t *testing.T) {
			guide, err := buildCommandGuide("build", platform)
			if err != nil {
				t.Fatalf("buildCommandGuide(build, %s): %v", platform, err)
			}
			parts := append([]string{}, guide.PreSteps...)
			parts = append(parts, guide.Intent, guide.RunCommand)
			parts = append(parts, guide.PostSteps...)
			assertBuildCoherentJobContract(t, "command-guide build --platform "+platform, strings.Join(parts, "\n"))
		})
	}
}
