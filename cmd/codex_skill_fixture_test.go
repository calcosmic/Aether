package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var codexFixtureSupportNames = []string{
	"aether-colony-creation", "aether-colony-research", "aether-colony-build-cycle",
}

// seedCodexSkillSupportFixture gives minimal install/publish packages the real
// private support sources required by the production payload validator. Resolve
// from this source file because some callers change cwd to a disposable repo.
func seedCodexSkillSupportFixture(t *testing.T, packageDir string) {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate Codex support fixture source")
	}
	root := filepath.Dir(filepath.Dir(source))
	for _, name := range codexFixtureSupportNames {
		rel := filepath.Join(".aether", "skills", "colony", name, "SKILL.md")
		content, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("read real support source %s: %v", name, err)
		}
		path := filepath.Join(packageDir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// assertCodexSkillFixtureInstalled checks the actual home payload, so fixture
// repairs cannot pass by suppressing skill synchronization or its errors.
func assertCodexSkillFixtureInstalled(t *testing.T, packageDir, homeDir string) {
	t.Helper()
	root := filepath.Join(homeDir, ".codex", "skills", "aether")
	for _, name := range codexFixtureSupportNames {
		want, err := os.ReadFile(filepath.Join(packageDir, ".aether", "skills", "colony", name, "SKILL.md"))
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join(root, "support", name+".md"))
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("installed private support %s differs from source: %v", name, err)
		}
	}
	for _, name := range expectedAntSkills {
		content, err := os.ReadFile(filepath.Join(root, name, "SKILL.md"))
		if err != nil {
			t.Fatal(err)
		}
		fm := parseSkillFrontmatter(string(content))
		command := strings.TrimPrefix(name, "ant-")
		if fm == nil || fm.Name != name || !bytes.Contains(content, []byte("aether command-guide "+command+" --platform codex")) {
			t.Errorf("installed %s lacks canonical name or command-guide route", name)
		}
	}
	var ownership codexSkillOwnership
	raw, err := os.ReadFile(filepath.Join(root, ".aether-owned.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &ownership); err != nil {
		t.Fatal(err)
	}
	if ownership.SchemaVersion != "codex-skill-ownership/v1" || len(ownership.Files) != 12 {
		t.Fatalf("incomplete installed ownership: %+v", ownership)
	}
	for _, file := range ownership.Files {
		path := filepath.Join(root, filepath.FromSlash(file.RelativePath))
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if lifecycleDigest(content) != file.SHA256 || uint32(info.Mode().Perm()) != file.Mode {
			t.Errorf("installed ownership does not match bytes/mode: %s", path)
		}
	}
}
