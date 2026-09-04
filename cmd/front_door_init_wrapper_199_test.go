package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const (
	frontDoorInitDescription199 = "Start a guided colony for one goal."
	frontDoorInitAuthority199   = "Go owns setup, registry updates, accepted-charter persistence, colony state creation, territory evidence, and init result truth."
	frontDoorInitCharter199     = "Persist the owner-approved goal and material constraints as accepted-charter/v1 through aether init."
	frontDoorInitTerritory199   = "Territory result is exactly one of Fresh, Refreshed, Stale—refresh required, or Unavailable."
	frontDoorInitRefusal199     = "An existing active colony is refused before storage opens; the refusal changes no files."
)

var frontDoorInitStages199 = []string{
	"Stage 1 — Queen opening",
	"Stage 2 — Setup",
	"Stage 3 — Accepted intent",
	"Stage 4 — Territory",
	"Stage 5 — Closeout",
}

type frontDoorInitSpec199 struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Runtime     struct {
		Command string `yaml:"command"`
	} `yaml:"runtime"`
	CodexOrchestration struct {
		Category   string `yaml:"category"`
		Skill      string `yaml:"skill"`
		Guide      string `yaml:"guide"`
		DriftGuard string `yaml:"drift_guard"`
	} `yaml:"codex_orchestration"`
	GuidedContract struct {
		Stages                 []string `yaml:"stages"`
		RuntimeAuthority       string   `yaml:"runtime_authority"`
		AcceptedCharter        string   `yaml:"accepted_charter"`
		Territory              string   `yaml:"territory"`
		ActiveColonyRefusal    string   `yaml:"active_colony_refusal"`
		ClaudeOpenCodeCloseout string   `yaml:"claude_opencode_closeout"`
		CodexCloseout          string   `yaml:"codex_closeout"`
	} `yaml:"guided_contract"`
}

func TestFrontDoorInitWrapperParity(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("find repository root: %v", err)
	}

	canonicalPath := filepath.Join(repoRoot, ".aether", "commands", "init.yaml")
	rawCanonical, err := os.ReadFile(canonicalPath)
	if err != nil {
		t.Fatalf("read canonical init source: %v", err)
	}
	var spec frontDoorInitSpec199
	if err := yaml.Unmarshal(rawCanonical, &spec); err != nil {
		t.Fatalf("parse canonical init source: %v", err)
	}

	if spec.Name != "ant-init" {
		t.Errorf("canonical name = %q, want ant-init", spec.Name)
	}
	if spec.Description != frontDoorInitDescription199 {
		t.Errorf("canonical description = %q, want %q", spec.Description, frontDoorInitDescription199)
	}
	if spec.Runtime.Command != "AETHER_OUTPUT_MODE=visual aether init --colony-mode <colony|orchestrator> --charter-json '<synthesized charter JSON>' \"<refined goal>\"" {
		t.Errorf("canonical runtime command = %q", spec.Runtime.Command)
	}
	if spec.CodexOrchestration.Category != commandGuideCategoryFullOrchestration ||
		spec.CodexOrchestration.Skill != commandGuideSkillCreation ||
		spec.CodexOrchestration.Guide != "aether command-guide init --platform codex" {
		t.Errorf("canonical Codex orchestration is incomplete: %#v", spec.CodexOrchestration)
	}
	if !strings.Contains(spec.CodexOrchestration.DriftGuard, "cmd/command_guide.go") {
		t.Errorf("canonical Codex drift guard does not name command_guide.go: %q", spec.CodexOrchestration.DriftGuard)
	}

	if strings.Join(spec.GuidedContract.Stages, "\n") != strings.Join(frontDoorInitStages199, "\n") {
		t.Errorf("canonical stages = %#v, want %#v", spec.GuidedContract.Stages, frontDoorInitStages199)
	}
	if spec.GuidedContract.RuntimeAuthority != frontDoorInitAuthority199 {
		t.Errorf("canonical runtime authority = %q", spec.GuidedContract.RuntimeAuthority)
	}
	if spec.GuidedContract.AcceptedCharter != frontDoorInitCharter199 {
		t.Errorf("canonical accepted-charter contract = %q", spec.GuidedContract.AcceptedCharter)
	}
	if spec.GuidedContract.Territory != frontDoorInitTerritory199 {
		t.Errorf("canonical territory contract = %q", spec.GuidedContract.Territory)
	}
	if spec.GuidedContract.ActiveColonyRefusal != frontDoorInitRefusal199 {
		t.Errorf("canonical active-colony refusal = %q", spec.GuidedContract.ActiveColonyRefusal)
	}
	if spec.GuidedContract.ClaudeOpenCodeCloseout != "/ant-plan" || spec.GuidedContract.CodexCloseout != "aether plan" {
		t.Errorf("canonical host closeouts = Claude/OpenCode %q, Codex %q", spec.GuidedContract.ClaudeOpenCodeCloseout, spec.GuidedContract.CodexCloseout)
	}

	hostWrappers := []struct {
		host string
		path string
	}{
		{host: "claude-flat", path: filepath.Join(repoRoot, ".claude", "commands", "ant-init.md")},
		{host: "claude", path: filepath.Join(repoRoot, ".claude", "commands", "ant", "init.md")},
		{host: "opencode", path: filepath.Join(repoRoot, ".opencode", "commands", "ant", "init.md")},
	}
	managedHeader := "<!-- Aether-managed: runtime spec at .aether/commands/init.yaml. Synced by aether update. -->"
	for _, wrapper := range hostWrappers {
		wrapper := wrapper
		t.Run(wrapper.host, func(t *testing.T) {
			raw, err := os.ReadFile(wrapper.path)
			if err != nil {
				t.Fatalf("read %s: %v", wrapper.path, err)
			}
			text := strings.ReplaceAll(string(raw), "\r\n", "\n")
			if !strings.HasPrefix(text, managedHeader+"\n") {
				t.Errorf("wrapper does not identify its canonical source")
			}
			assertFrontDoorInitContract199(t, text)
			if !strings.Contains(text, `description: "`+frontDoorInitDescription199+`"`) {
				t.Errorf("wrapper does not use exact init description %q", frontDoorInitDescription199)
			}
			if !strings.HasSuffix(strings.TrimSpace(text), "Next Up: /ant-plan") {
				t.Errorf("wrapper does not close with exact /ant-plan:\n%s", text)
			}
			for _, forbidden := range []string{"Next Up: aether plan", "$ant-", "aether lay-eggs"} {
				if strings.Contains(text, forbidden) {
					t.Errorf("wrapper contains host-inappropriate init guidance %q", forbidden)
				}
			}
			assertNoDirectInitStateWrites199(t, text)
		})
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
	if !strings.HasSuffix(strings.TrimSpace(skillText), "Next Up: aether plan") {
		t.Errorf("Codex creation skill does not close with exact aether plan:\n%s", skillText)
	}
	for _, forbidden := range []string{"/ant-plan", "$ant-", "aether lay-eggs"} {
		if strings.Contains(skillText, forbidden) {
			t.Errorf("Codex creation skill contains deferred native lifecycle vocabulary %q", forbidden)
		}
	}
	assertNoDirectInitStateWrites199(t, skillText)
}

func assertFrontDoorInitContract199(t *testing.T, text string) {
	t.Helper()
	last := -1
	for _, stage := range frontDoorInitStages199 {
		index := strings.Index(text, stage)
		if index < 0 {
			t.Errorf("surface is missing %q", stage)
			continue
		}
		if index <= last {
			t.Errorf("surface has stage %q out of order", stage)
		}
		last = index
	}
	for _, anchor := range []string{
		frontDoorInitAuthority199,
		frontDoorInitCharter199,
		frontDoorInitTerritory199,
		frontDoorInitRefusal199,
	} {
		if !strings.Contains(text, anchor) {
			t.Errorf("surface is missing init contract %q", anchor)
		}
	}
}

func assertNoDirectInitStateWrites199(t *testing.T, text string) {
	t.Helper()
	lower := strings.ToLower(text)
	for _, forbidden := range []string{
		"> .aether/data/",
		">>.aether/data/",
		">> .aether/data/",
		"tee .aether/data/",
		"touch .aether/data/",
		"mkdir .aether/data/",
		"writefile(.aether/data/",
	} {
		if strings.Contains(lower, forbidden) {
			t.Errorf("surface contains a direct .aether/data write pattern %q", forbidden)
		}
	}
}
