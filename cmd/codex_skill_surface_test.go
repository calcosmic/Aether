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
	stdout, stderr = &output, &output
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
	if ownership.SchemaVersion != "codex-skill-ownership/v1" || !strings.HasPrefix(ownership.PayloadIdentity, "sha256:") || len(ownership.PayloadIdentity) != 71 || len(ownership.Files) != 12 {
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
		def := adaptCommandGuideDefinitionForPlatform(command, "codex", commandGuideCatalog()[command])
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

func TestCodexNativeInstalledBuildGuidance(t *testing.T) {
	t.Setenv(codexNativeBuildOptInEnv, "1")
	home := t.TempDir()
	if ok, output := runAntSkillInstall(t, home); !ok {
		t.Fatalf("install: %s", output)
	}
	raw, err := os.ReadFile(filepath.Join(home, ".codex", "skills", "aether", "ant-build", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"codex-native-worker reserve", "codex-native-worker bind", "worker.native.release", "codex-native-worker record", "codex-native-worker stage", "codex-native-worker observe", "codex-native-worker context", "codex-native-worker question", "aether resume", "inspect --phase", "normalizedDispatchTaskID"} {
		if !strings.Contains(string(raw), required) {
			t.Errorf("installed native skill missing %q", required)
		}
	}
	for _, forbidden := range []string{"build-wave playbook", "visible live Task/subagent", "Write per-worker JSON and the final completion JSON", "build-completion-stage", "In worktree mode one job"} {
		if strings.Contains(string(raw), forbidden) {
			t.Errorf("installed native skill retained %q", forbidden)
		}
	}
	for _, command := range codexPublicSkillCommands() {
		if command == "build" {
			continue
		}
		p := filepath.Join(home, ".codex", "skills", "aether", "ant-"+command, "SKILL.md")
		got, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		def := commandGuideCatalog()[command]
		if !strings.Contains(string(got), def.RunCommand) || strings.Contains(string(got), "codex-native-worker") {
			t.Errorf("unrelated %s installed skill changed route", command)
		}
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

func TestCodexAntSkillInventory(t *testing.T) {
	commands := []string{"init", "discuss", "oracle", "colonize", "plan", "build", "continue", "swarm", "seal"}
	for _, name := range []string{"nil", "empty", "missing", "duplicate", "case", "unicode", "missing-guide", "literal-guide", "empty-route", "invalid-support"} {
		t.Run(name, func(t *testing.T) {
			inventory := append([]string(nil), commands...)
			catalog := commandGuideCatalog()
			switch name {
			case "nil":
				inventory = nil
			case "empty":
				inventory = []string{}
			case "missing":
				inventory = inventory[:8]
			case "duplicate":
				inventory[8] = inventory[0]
			case "case":
				inventory[4] = "Plan"
			case "unicode":
				inventory[4] = "plаn" // Cyrillic a, not ASCII a.
			case "missing-guide":
				delete(catalog, "plan")
			default:
				def := catalog["plan"]
				switch name {
				case "literal-guide":
					def.Literal = true
				case "empty-route":
					def.RunCommand = ""
				case "invalid-support":
					def.SkillReference = "../custom"
				}
				catalog["plan"] = def
			}
			payload, err := buildCodexSkillPayloadFromInventory(antSkillSourceRoot(t), inventory, catalog)
			if err == nil {
				t.Fatalf("accepted invalid inventory %s", name)
			}
			fixture := newMaintenanceMutation199Fixture(t)
			plan := fixture.plan("invalid-inventory")
			if err := planCodexSkillTargets(&plan, payload); err == nil || len(plan.Targets) != 0 {
				t.Fatalf("invalid generator output planned targets: %v", err)
			}
			entries, err := os.ReadDir(fixture.codex)
			if err != nil || len(entries) != 0 {
				t.Fatalf("validation wrote targets: %v %v", entries, err)
			}
		})
	}
	for _, name := range []string{"null", "empty", "schema", "missing-command", "duplicate-command", "case", "unicode", "missing-file", "duplicate-file", "escape", "mode", "digest", "yaml-name", "future-runtime", "empty-version"} {
		t.Run("payload-"+name, func(t *testing.T) {
			payload, err := buildCodexSkillPayload(antSkillSourceRoot(t))
			if err != nil {
				t.Fatal(err)
			}
			switch name {
			case "null":
				payload.Commands = nil
			case "empty":
				payload.Commands = []string{}
			case "schema":
				payload.SchemaVersion = "unknown/v2"
			case "missing-command":
				payload.Commands = payload.Commands[:8]
			case "duplicate-command":
				payload.Commands[8] = payload.Commands[0]
			case "case":
				payload.Commands[4] = "PLAN"
			case "unicode":
				payload.Commands[4] = "plаn"
			case "missing-file":
				payload.Files = payload.Files[:11]
			case "duplicate-file":
				payload.Files[11] = payload.Files[0]
			case "escape":
				payload.Files[0].RelativePath = "../.agents/skills/ant-init/SKILL.md"
			case "mode":
				payload.Files[0].Mode = 0o777
			case "digest":
				payload.Files[0].Content = []byte("tampered")
			case "yaml-name":
				payload.Files[0].Content = []byte(strings.Replace(string(payload.Files[0].Content), "name: ant-init", "name: Ant-init", 1))
				payload.Files[0].SHA256 = lifecycleDigest(payload.Files[0].Content)
			case "future-runtime":
				payload.MinRuntimeVersion = "9999.0.0"
			case "empty-version":
				payload.SourceVersion = ""
			}
			fixture := newMaintenanceMutation199Fixture(t)
			plan := fixture.plan("invalid-payload")
			if err := planCodexSkillTargets(&plan, payload); err == nil || len(plan.Targets) != 0 {
				t.Fatalf("invalid payload planned targets: %v", err)
			}
			entries, err := os.ReadDir(fixture.codex)
			if err != nil || len(entries) != 0 {
				t.Fatalf("validation wrote targets: %v %v", entries, err)
			}
		})
	}
	t.Run("missing-support", func(t *testing.T) {
		if _, err := buildCodexSkillPayload(t.TempDir()); err == nil {
			t.Fatal("accepted missing support")
		}
	})
	t.Run("deterministic", func(t *testing.T) {
		first, err := buildCodexSkillPayload(antSkillSourceRoot(t))
		if err != nil {
			t.Fatal(err)
		}
		second, err := buildCodexSkillPayload(antSkillSourceRoot(t))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(first, second) || codexSkillPayloadIdentity(first) != codexSkillPayloadIdentity(second) {
			t.Fatal("nondeterministic payload")
		}
	})
}
