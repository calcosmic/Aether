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

	d17Guides := []d17Guide199{
		{path: "README.md", host: "public Claude/OpenCode guide", seal: "/ant-seal", status: "/ant-status", entomb: "/ant-entomb", init: "/ant-init"},
		{path: "AGENTS.md", host: "Codex guide", seal: "aether seal", status: "aether status", entomb: "aether entomb", init: "aether init"},
		{path: ".aether/rules/aether-colony.md", host: "canonical Claude rule", seal: "/ant-seal", status: "/ant-status", entomb: "/ant-entomb", init: "/ant-init"},
		{path: ".claude/rules/aether-colony.md", host: "generated Claude rule", seal: "/ant-seal", status: "/ant-status", entomb: "/ant-entomb", init: "/ant-init"},
		{path: ".aether/templates/opencode-md-template.md", host: "canonical OpenCode template", seal: "/ant-seal", status: "/ant-status", entomb: "/ant-entomb", init: "/ant-init"},
		{path: "cmd/.opencode/OPENCODE.md", host: "generated OpenCode guide", seal: "/ant-seal", status: "/ant-status", entomb: "/ant-entomb", init: "/ant-init"},
		{path: ".aether/templates/agents-md-template.md", host: "canonical Codex template", seal: "aether seal", status: "aether status", entomb: "aether entomb", init: "aether init"},
		{path: "cmd/AGENTS.md", host: "generated Codex guide", seal: "aether seal", status: "aether status", entomb: "aether entomb", init: "aether init"},
	}

	t.Run("StatusPrimaryAfterSeal", func(t *testing.T) {
		for _, guide := range d17Guides {
			section := readD17GuideSection199(t, root, guide)
			if !d17StatusPrimary199(section, guide) {
				t.Errorf("%s (%s) must order %q before %q before optional %q", guide.path, guide.host, guide.seal, guide.status, guide.entomb)
			}
		}
		unsafe := d17GuideMarker199 + "\n" + "/ant-seal\n/ant-entomb\n/ant-status\n/ant-init\n"
		if d17StatusPrimary199(unsafe, d17Guides[0]) {
			t.Error("D-17 status-order guard accepted a direct seal-to-entomb sequence")
		}
	})

	t.Run("EntombOptional", func(t *testing.T) {
		for _, guide := range d17Guides {
			section := readD17GuideSection199(t, root, guide)
			if !d17EntombOptional199(section) {
				t.Errorf("%s (%s) must describe entomb as an optional, explicit owner-invoked archive-and-clear alternative that is never automatic or required", guide.path, guide.host)
			}
		}
		unsafe := d17GuideMarker199 + "\n/ant-seal\n/ant-status\n/ant-entomb is required after sealing\n/ant-init\n"
		if d17EntombOptional199(unsafe) {
			t.Error("D-17 optionality guard accepted required entomb")
		}
	})

	t.Run("InitAfterArchiveClear", func(t *testing.T) {
		for _, guide := range d17Guides {
			section := readD17GuideSection199(t, root, guide)
			if !d17InitAfterArchiveClear199(section, guide) {
				t.Errorf("%s (%s) must allow %q only after a successful archive-and-clear receipt verifies the archive and clears active state", guide.path, guide.host, guide.init)
			}
		}
		unsafe := d17GuideMarker199 + "\n/ant-seal\n/ant-status\n/ant-entomb optional, explicit owner-invoked archive-and-clear alternative; never automatic or required\n/ant-init\n"
		if d17InitAfterArchiveClear199(unsafe, d17Guides[0]) {
			t.Error("D-17 init guard accepted direct seal-to-init without a successful archive-and-clear receipt")
		}
	})

	t.Run("ForcedSealStillReviewable", func(t *testing.T) {
		for _, guide := range d17Guides {
			section := readD17GuideSection199(t, root, guide)
			want := "A forced-incomplete marker remains visible in " + guide.status + " and optional " + guide.entomb + "."
			if !strings.Contains(section, want) {
				t.Errorf("%s (%s) loses the forced-incomplete marker through status or optional entomb", guide.path, guide.host)
			}
		}
	})

	t.Run("ClaudeRuleGenerationParity", func(t *testing.T) {
		canonical, readErr := os.ReadFile(filepath.Join(root, ".aether", "rules", "aether-colony.md"))
		if readErr != nil {
			t.Fatalf("read canonical Claude rule: %v", readErr)
		}
		hubDir := t.TempDir()
		hubSystem := filepath.Join(hubDir, "system")
		if err := os.MkdirAll(filepath.Join(hubSystem, "rules"), 0755); err != nil {
			t.Fatalf("create isolated production rules source: %v", err)
		}
		if err := os.WriteFile(filepath.Join(hubSystem, "rules", "aether-colony.md"), canonical, 0644); err != nil {
			t.Fatalf("stage canonical Claude rule for production generation: %v", err)
		}
		temporaryRepo := t.TempDir()
		if got := runUpdateSync(hubDir, temporaryRepo, true); len(got.errors) != 0 || got.copied == 0 {
			t.Fatalf("production Claude-rule generation failed: copied=%d errors=%v", got.copied, got.errors)
		}
		generated, readErr := os.ReadFile(filepath.Join(temporaryRepo, ".claude", "rules", "aether-colony.md"))
		if readErr != nil {
			t.Fatalf("read production-generated Claude rule: %v", readErr)
		}
		current, readErr := os.ReadFile(filepath.Join(root, ".claude", "rules", "aether-colony.md"))
		if readErr != nil {
			t.Fatalf("read checked-in Claude rule: %v", readErr)
		}
		if string(generated) != string(canonical) || string(current) != string(generated) {
			t.Error(".claude/rules/aether-colony.md drifted from production generation of .aether/rules/aether-colony.md")
		}
	})

	t.Run("opencode-is-generated-from-canonical-template", func(t *testing.T) {
		hubSystem := t.TempDir()
		if err := os.MkdirAll(filepath.Join(hubSystem, "templates"), 0755); err != nil {
			t.Fatalf("create project-document template directory: %v", err)
		}
		for _, template := range []string{"opencode-md-template.md", "agents-md-template.md"} {
			canonicalTemplate, readErr := os.ReadFile(filepath.Join(root, ".aether", "templates", template))
			if readErr != nil {
				t.Fatalf("read canonical %s: %v", template, readErr)
			}
			if err := os.WriteFile(filepath.Join(hubSystem, "templates", template), canonicalTemplate, 0644); err != nil {
				t.Fatalf("stage canonical %s for production generation: %v", template, err)
			}
		}
		temporaryRepo := t.TempDir()
		if _, copied, _, errs := syncProjectDocs(hubSystem, temporaryRepo); len(errs) != 0 || copied == 0 {
			t.Fatalf("production project-document generation failed: copied=%d errors=%v", copied, errs)
		}
		for _, generatedDoc := range []struct {
			generated string
			current   string
			label     string
		}{
			{filepath.Join(temporaryRepo, ".opencode", "OPENCODE.md"), filepath.Join(root, "cmd", ".opencode", "OPENCODE.md"), "OpenCode"},
			{filepath.Join(temporaryRepo, "AGENTS.md"), filepath.Join(root, "cmd", "AGENTS.md"), "Codex"},
		} {
			generated, readErr := os.ReadFile(generatedDoc.generated)
			if readErr != nil {
				t.Fatalf("read generated %s guide: %v", generatedDoc.label, readErr)
			}
			current, readErr := os.ReadFile(generatedDoc.current)
			if readErr != nil {
				t.Fatalf("read checked-in %s guide: %v", generatedDoc.label, readErr)
			}
			if string(current) != string(generated) {
				t.Errorf("%s drifted from production project-document generation", generatedDoc.current)
			}
			for _, command := range []string{"/ant-pause", "/ant-resume"} {
				if generatedDoc.label == "Codex" {
					continue
				}
				if !strings.Contains(string(generated), command) {
					t.Errorf("generated %s guide does not teach %s", generatedDoc.label, command)
				}
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

		taskSpec, readErr := os.ReadFile(filepath.Join(root, "bench/tasks/03-interrupted-execution.md"))
		if readErr != nil {
			t.Fatalf("read interruption task specification: %v", readErr)
		}
		for _, required := range []string{"log the conflict", "/ant-maintenance recovery-inspect", "fresh `/ant-resume`", "not part of the ordinary"} {
			if !strings.Contains(string(taskSpec), required) {
				t.Errorf("interruption diagnostic exception must make %q explicit", required)
			}
		}
	})
}

const d17GuideMarker199 = "### Sealed colony: review before archive"

type d17Guide199 struct {
	path, host, seal, status, entomb, init string
}

func readD17GuideSection199(t *testing.T, root string, guide d17Guide199) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, guide.path))
	if err != nil {
		t.Fatalf("read %s: %v", guide.path, err)
	}
	start := strings.Index(string(content), d17GuideMarker199)
	if start < 0 {
		t.Errorf("%s (%s) is missing %q", guide.path, guide.host, d17GuideMarker199)
		return ""
	}
	section := string(content)[start:]
	if end := strings.Index(section, "\n---"); end >= 0 {
		section = section[:end]
	}
	return section
}

func d17StatusPrimary199(section string, guide d17Guide199) bool {
	seal := strings.Index(section, guide.seal)
	status := strings.Index(section, guide.status)
	entomb := strings.Index(section, guide.entomb)
	return seal >= 0 && status > seal && entomb > status
}

func d17EntombOptional199(section string) bool {
	return strings.Contains(section, "optional, explicit owner-invoked archive-and-clear alternative") &&
		strings.Contains(section, "never automatic or required")
}

func d17InitAfterArchiveClear199(section string, guide d17Guide199) bool {
	precondition := "Only after a successful archive-and-clear receipt has verified the archive and cleared active state may you run " + guide.init + "."
	return strings.Contains(section, precondition) &&
		strings.Index(section, guide.init) > strings.Index(section, guide.entomb)
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
	return strings.Trim(strings.TrimSpace(command[1]), "`")
}
