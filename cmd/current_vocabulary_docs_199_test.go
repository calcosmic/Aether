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
}
