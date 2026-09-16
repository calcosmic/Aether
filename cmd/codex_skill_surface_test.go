package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// This list deliberately does not derive from the production inventory.
var expectedAntSkills = []string{"ant-build", "ant-colonize", "ant-continue", "ant-discuss", "ant-init", "ant-oracle", "ant-plan", "ant-seal", "ant-swarm"}

func antSkillSourceRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func runAntSkillInstall(t *testing.T, home string) (bool, string) {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	t.Setenv("AETHER_HUB_DIR", "")
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	var output bytes.Buffer
	stdout = &output
	rootCmd.SetArgs([]string{"install", "--package-dir", antSkillSourceRoot(t), "--home-dir", home, "--channel", "stable", "--skip-build-binary"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(output.Bytes(), &envelope); err != nil {
		t.Fatalf("install output: %v: %s", err, output.String())
	}
	return envelope.OK, output.String()
}

func TestCodexAntSkillInstallTracer(t *testing.T) {
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
		t.Fatalf("installed public skills = %v; want %v", names, expectedAntSkills)
	}
	var ownership struct {
		SchemaVersion   string `json:"schema_version"`
		PayloadIdentity string `json:"payload_identity"`
		Files           []struct {
			RelativePath string `json:"relative_path"`
			SHA256       string `json:"sha256"`
			Mode         uint32 `json:"mode"`
		} `json:"files"`
	}
	raw, err := os.ReadFile(filepath.Join(root, ".aether-owned.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &ownership); err != nil {
		t.Fatal(err)
	}
	if ownership.SchemaVersion != "codex-skill-ownership/v1" || len(ownership.PayloadIdentity) != 64 || len(ownership.Files) != 12 {
		t.Fatalf("incomplete ownership: %+v", ownership)
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
			t.Fatalf("ownership mismatch: %s", path)
		}
	}
	for _, name := range expectedAntSkills {
		raw, err := os.ReadFile(filepath.Join(root, name, "SKILL.md"))
		if err != nil {
			t.Fatal(err)
		}
		fm := parseSkillFrontmatter(string(raw))
		if fm == nil || fm.Name != name {
			t.Fatalf("invalid YAML name in %s", name)
		}
		command := strings.TrimPrefix(name, "ant-")
		def := commandGuideCatalog()[command]
		for _, required := range []string{"aether command-guide " + command + " --platform codex", def.RunCommand, "../support/" + def.SkillReference + ".md", "automatically", "raw/exact"} {
			if required == "" || !strings.Contains(string(raw), required) {
				t.Errorf("%s missing %q", name, required)
			}
		}
		support := filepath.Join(root, name, "..", "support", def.SkillReference+".md")
		actual, err := os.ReadFile(support)
		if err != nil {
			t.Fatal(err)
		}
		source, err := os.ReadFile(filepath.Join(antSkillSourceRoot(t), ".aether", "skills", "colony", def.SkillReference, "SKILL.md"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(actual, source) {
			t.Errorf("private support changed: %s", support)
		}
	}
	journal, err := filepath.Glob(filepath.Join(home, ".aether", "data", "transactions", "*", "receipt.json"))
	if err != nil || len(journal) == 0 {
		t.Fatalf("no durable transaction receipt: %v %v", journal, err)
	}
}

func TestCodexAntSkillCleanInstallCollision(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, ".codex", "skills", "aether")
	path := filepath.Join(root, "ant-plan", "SKILL.md")
	custom := []byte("---\nname: ant-plan\nsource: custom\n---\nMy planning instructions\n")
	writeMaintenanceMutation199File(t, path, custom)
	if ok, output := runAntSkillInstall(t, home); ok || !strings.Contains(output, "collision") {
		t.Fatalf("want explicit non-success collision: %s", output)
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, custom) {
		t.Fatalf("custom bytes lost: %s (%v)", got, err)
	}
	if _, err := os.Stat(filepath.Join(root, ".aether-owned.json")); !os.IsNotExist(err) {
		t.Fatalf("fabricated ownership: %v", err)
	}
	if dirs := findSkillDirs(root); len(dirs) != 1 {
		t.Fatalf("partial skill install on collision: %v", dirs)
	}
}
