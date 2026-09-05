package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var retiredCurrentLifecycleCommand199 = regexp.MustCompile(`(?:/ant-(?:pause-colony|resume-colony|recover|abandon)\b|aether\s+resume-colony\b|\$ant-(?:pause|resume)\b)`)

func TestCurrentVocabularyDocs199(t *testing.T) {
	root, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}

	activeDocs := []struct {
		path string
		host string
	}{
		{"README.md", "public guide"},
		{"AGENTS.md", "Codex guide"},
		{"cmd/.opencode/OPENCODE.md", "generated OpenCode guide"},
		{"docs/phase3-section-commands.md", "command reference"},
	}

	t.Run("active-document-vocabulary", func(t *testing.T) {
		for _, doc := range activeDocs {
			content, readErr := os.ReadFile(filepath.Join(root, doc.path))
			if readErr != nil {
				t.Fatalf("read %s: %v", doc.path, readErr)
			}
			if retired := retiredCurrentLifecycleCommand199.FindString(string(content)); retired != "" {
				t.Errorf("%s (%s) teaches retired current lifecycle command %q", doc.path, doc.host, retired)
			}
		}

		for _, doc := range []string{"README.md", "docs/phase3-section-commands.md"} {
			content, readErr := os.ReadFile(filepath.Join(root, doc))
			if readErr != nil {
				t.Fatalf("read %s: %v", doc, readErr)
			}
			for _, command := range []string{"/ant-pause", "/ant-resume"} {
				if !strings.Contains(string(content), command) {
					t.Errorf("%s does not teach %s", doc, command)
				}
			}
		}

		agents, readErr := os.ReadFile(filepath.Join(root, "AGENTS.md"))
		if readErr != nil {
			t.Fatalf("read AGENTS.md: %v", readErr)
		}
		for _, command := range []string{"aether pause", "aether resume"} {
			if !strings.Contains(string(agents), command) {
				t.Errorf("AGENTS.md does not teach raw Codex command %q", command)
			}
		}
	})

	t.Run("opencode-is-generated-from-canonical-template", func(t *testing.T) {
		hubSystem := t.TempDir()
		canonicalTemplate, readErr := os.ReadFile(filepath.Join(root, ".aether", "templates", "opencode-md-template.md"))
		if readErr != nil {
			t.Fatalf("read canonical OpenCode template: %v", readErr)
		}
		if err := os.MkdirAll(filepath.Join(hubSystem, "templates"), 0755); err != nil {
			t.Fatalf("create project-document template directory: %v", err)
		}
		if err := os.WriteFile(filepath.Join(hubSystem, "templates", "opencode-md-template.md"), canonicalTemplate, 0644); err != nil {
			t.Fatalf("stage canonical OpenCode template for production generation: %v", err)
		}
		temporaryRepo := t.TempDir()
		if _, copied, _, errs := syncProjectDocs(hubSystem, temporaryRepo); len(errs) != 0 || copied == 0 {
			t.Fatalf("production project-document generation failed: copied=%d errors=%v", copied, errs)
		}
		generated, readErr := os.ReadFile(filepath.Join(temporaryRepo, ".opencode", "OPENCODE.md"))
		if readErr != nil {
			t.Fatalf("read generated OpenCode guide: %v", readErr)
		}
		current, readErr := os.ReadFile(filepath.Join(root, "cmd", ".opencode", "OPENCODE.md"))
		if readErr != nil {
			t.Fatalf("read checked-in OpenCode guide: %v", readErr)
		}
		if string(current) != string(generated) {
			t.Error("cmd/.opencode/OPENCODE.md drifted from production generation of .aether/templates/opencode-md-template.md")
		}
		for _, command := range []string{"/ant-pause", "/ant-resume"} {
			if !strings.Contains(string(generated), command) {
				t.Errorf("generated OpenCode guide does not teach %s", command)
			}
		}
	})

	t.Run("interrupted-execution-benchmark-uses-one-resume-door", func(t *testing.T) {
		benchmarkSources := []string{
			"bench/RUNBOOK.md",
			"bench/harness/permitted-inputs.md",
			"bench/tasks/03-interrupted-execution.md",
		}
		for _, source := range benchmarkSources {
			content, readErr := os.ReadFile(filepath.Join(root, source))
			if readErr != nil {
				t.Fatalf("read %s: %v", source, readErr)
			}
			if retired := retiredCurrentLifecycleCommand199.FindString(string(content)); retired != "" {
				t.Errorf("%s scripts retired interruption command %q", source, retired)
			}
			for _, lane := range []string{"aether-interactive", "aether-autopilot"} {
				if got := scriptedAetherResumeCommand199(t, source, string(content), lane); got != "/ant-resume" {
					t.Errorf("%s %s scripted resume = %q, want exactly /ant-resume", source, lane, got)
				}
			}
		}
	})
}

func scriptedAetherResumeCommand199(t *testing.T, source, content, lane string) string {
	t.Helper()
	if source == "bench/harness/permitted-inputs.md" {
		section := regexp.MustCompile(`(?ms)^## ` + regexp.QuoteMeta(lane) + ` 03-interrupted-execution\n(.*?)(?:^## |\z)`).FindStringSubmatch(content)
		if len(section) != 2 {
			t.Errorf("%s does not contain the %s interrupted-execution rules", source, lane)
			return ""
		}
		command := regexp.MustCompile("(?s)Permitted response:\\*\\*\\s*run exactly `([^`]+)`").FindStringSubmatch(section[1])
		if len(command) != 2 {
			t.Errorf("%s does not give one exact scripted resume command for %s", source, lane)
			return ""
		}
		return strings.TrimSpace(command[1])
	}

	displayLane := strings.ReplaceAll(lane, "-", " ")
	command := regexp.MustCompile(`(?mi)^\|\s*(?:` + regexp.QuoteMeta(lane) + `|` + regexp.QuoteMeta(displayLane) + `)\s*\|\s*([^|]+?)\s*\|\s*$`).FindStringSubmatch(content)
	if len(command) != 2 {
		t.Errorf("%s does not contain the %s resume-table row", source, lane)
		return ""
	}
	return strings.TrimSpace(command[1])
}
