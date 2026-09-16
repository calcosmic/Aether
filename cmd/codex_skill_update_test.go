package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type antUpdateFixture struct {
	home, hub, repo, skills string
	payload                 codexSkillPayload
}

func newAntUpdateFixture(t *testing.T) antUpdateFixture {
	t.Helper()
	antPayloadEnvironment(t)
	prior := Version
	Version = "1.0.79"
	t.Cleanup(func() { Version = prior })
	home := t.TempDir()
	hub := filepath.Join(home, ".aether")
	source := createMockSourceCheckout(t, "1.0.79")
	result := setupInstallHub(hub, source, "1.0.79")
	if errs := installHubErrors(result); len(errs) > 0 {
		t.Fatal(errs)
	}
	p := antReadPublished(t, hub)
	repo := bindCommandTestRepository(t).Root
	t.Setenv("HOME", home)
	skills := filepath.Join(home, ".codex", "skills", "aether")
	antSeedLegacy(t, skills)
	return antUpdateFixture{home: home, hub: hub, repo: repo, skills: skills, payload: p}
}
func antUpdateRun(t *testing.T, args ...string) (map[string]interface{}, string, error) {
	t.Helper()
	resetFlags(rootCmd)
	output, err := antPayloadCommand(t, append([]string{"update"}, args...)...)
	var envelope struct {
		Result map[string]interface{} `json:"result"`
	}
	if strings.HasPrefix(strings.TrimSpace(output), "{") {
		if e := json.Unmarshal([]byte(output), &envelope); e != nil {
			t.Fatalf("invalid update output: %v %s", e, output)
		}
	}
	return envelope.Result, output, err
}
func antUpdatePreview(t *testing.T, result map[string]interface{}) maintenanceMutationPreview {
	t.Helper()
	raw, err := json.Marshal(result["preview"])
	if err != nil {
		t.Fatal(err)
	}
	var preview maintenanceMutationPreview
	if err := json.Unmarshal(raw, &preview); err != nil {
		t.Fatal(err)
	}
	return preview
}
func antSkillPreviewTargets(preview maintenanceMutationPreview) []maintenanceMutationTargetPreview {
	var targets []maintenanceMutationTargetPreview
	for _, p := range preview.Targets {
		if p.Root == lifecycleTransactionRootCodexHome && strings.HasPrefix(filepath.ToSlash(p.RelativeTarget), "skills/aether/") {
			targets = append(targets, p)
		}
	}
	return targets
}
func TestCodexAntSkillRegisteredUpdate(t *testing.T) {
	for _, force := range []bool{false, true} {
		name := "normal"
		if force {
			name = "force"
		}
		t.Run(name, func(t *testing.T) {
			f := newAntUpdateFixture(t)
			var args []string
			if force {
				args = []string{"--force"}
			}
			result, output, err := antUpdateRun(t, args...)
			if err != nil {
				t.Fatal(err)
			}
			antAssertPublishedHome(t, f.hub, f.home)
			for _, old := range antLegacy(t).Files {
				if _, err := os.Stat(filepath.Join(f.skills, old.RelativePath)); !os.IsNotExist(err) {
					t.Fatalf("legacy not retired: %s %v", old.RelativePath, err)
				}
			}
			targets := antSkillPreviewTargets(antUpdatePreview(t, result))
			if len(targets) != 27 {
				t.Fatalf("skill targets not in main transaction: %d", len(targets))
			}
			if !strings.Contains(output, codexSkillPayloadIdentity(f.payload)) || !strings.Contains(output, `"details"`) || !strings.Contains(output, "Skills (codex shims)") {
				t.Fatal("missing structured skill result")
			}
			if result["state_effect"] != "committed" || result["receipt"] == nil {
				t.Fatalf("missing transaction receipt: %v", result)
			}
		})
	}
	t.Run("preserved", func(t *testing.T) {
		f := newAntUpdateFixture(t)
		for _, rel := range []string{"aether-plan/SKILL.md", "my-custom/SKILL.md"} {
			writeMaintenanceMutation199File(t, filepath.Join(f.skills, rel), []byte("owner's custom skill"))
		}
		for _, rel := range []string{"AGENTS.md", ".codex/CODEX.md"} {
			writeMaintenanceMutation199File(t, filepath.Join(f.repo, rel), []byte("owner's project instructions"))
		}
		writeMaintenanceMutation199File(t, filepath.Join(f.hub, "system", "templates", "agents-md-template.md"), []byte("Aether Colony -- Codex CLI Instructions\nnew guidance"))
		writeMaintenanceMutation199File(t, filepath.Join(f.hub, "system", "templates", "codex-md-template.md"), []byte("Aether Codex Workflow Guide\nnew guidance"))
		result, output, err := antUpdateRun(t, "--force")
		if err != nil {
			t.Fatal(err)
		}
		for _, rel := range []string{"aether-plan/SKILL.md", "my-custom/SKILL.md"} {
			if string(mustReadLifecycleFixtureFile(t, filepath.Join(f.skills, rel))) != "owner's custom skill" {
				t.Fatal("custom lost")
			}
		}
		for _, rel := range []string{"AGENTS.md", ".codex/CODEX.md"} {
			if string(mustReadLifecycleFixtureFile(t, filepath.Join(f.repo, rel))) != "owner's project instructions" {
				t.Fatal("custom doc lost")
			}
		}
		for _, want := range []string{"preserved_codex_skills", "preserved_project_docs", "aether-plan/SKILL.md", "my-custom/SKILL.md", "AGENTS.md", ".codex/CODEX.md"} {
			if !strings.Contains(output, want) {
				t.Fatalf("unreported preservation %s", want)
			}
		}
		if !strings.Contains(result["message"].(string), "Preserved") {
			t.Fatal("no plain-English preservation")
		}
		t.Setenv("AETHER_OUTPUT_MODE", "visual")
		_, visual, err := antUpdateRun(t)
		if err != nil || !strings.Contains(visual, "Preserved") || !strings.Contains(visual, "AGENTS.md") {
			t.Fatalf("missing visual preservation: %v %s", err, visual)
		}
	})
	for _, force := range []bool{false, true} {
		name := "collision"
		if force {
			name += "-force"
		}
		t.Run(name, func(t *testing.T) {
			f := newAntUpdateFixture(t)
			writeMaintenanceMutation199File(t, filepath.Join(f.skills, "ant-plan", "SKILL.md"), []byte("custom collision"))
			before := antSnapshot(t, f.home)
			repoBefore := antSnapshot(t, f.repo)
			args := []string{}
			if force {
				args = append(args, "--force")
			}
			_, output, err := antUpdateRun(t, args...)
			if err == nil || !strings.Contains(output, "collision") || !strings.Contains(output, "codex_skill_disposition") {
				t.Fatalf("collision accepted/unreported: %v %s", err, output)
			}
			antAssertSnapshot(t, f.home, before)
			antAssertSnapshot(t, f.repo, repoBefore)
		})
	}
}
func TestCodexAntSkillRegisteredUpdateRepeat(t *testing.T) {
	f := newAntUpdateFixture(t)
	if _, _, err := antUpdateRun(t); err != nil {
		t.Fatal(err)
	}
	before := antSnapshot(t, f.skills)
	result, _, err := antUpdateRun(t)
	if err != nil {
		t.Fatal(err)
	}
	antAssertSnapshot(t, f.skills, before)
	antAssertPublishedHome(t, f.hub, f.home)
	targets := antSkillPreviewTargets(antUpdatePreview(t, result))
	if len(targets) != 27 {
		t.Fatal("missing unchanged targets")
	}
	for _, target := range targets {
		if target.Change != maintenanceMutationChangeUnchanged {
			t.Fatalf("repeat changed %+v", target)
		}
	}
	details, _, _ := maintenancePreviewSyncDetails(antUpdatePreview(t, result))
	for _, restart := range platformRestartTargets(details) {
		if strings.Contains(restart, "skill") {
			t.Fatalf("unchanged skills require restart: %v", restart)
		}
	}
}
func TestCodexAntSkillUpdatePreview(t *testing.T) {
	f := newAntUpdateFixture(t)
	homeBefore := antSnapshot(t, f.home)
	repoBefore := antSnapshot(t, f.repo)
	result, _, err := antUpdateRun(t, "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	antAssertSnapshot(t, f.home, homeBefore)
	antAssertSnapshot(t, f.repo, repoBefore)
	for _, p := range []string{filepath.Join(f.home, ".codex", ".aether-skill-locks"), filepath.Join(f.repo, ".aether", "data", "transactions")} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("preview created evidence/lock: %s %v", p, err)
		}
	}
	preview := antSkillPreviewTargets(antUpdatePreview(t, result))
	if len(preview) != 27 {
		t.Fatalf("preview skill count=%d", len(preview))
	}
	applied, _, err := antUpdateRun(t)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(preview, antSkillPreviewTargets(antUpdatePreview(t, applied))) {
		t.Fatal("preview and commit differ")
	}
	antAssertPublishedHome(t, f.hub, f.home)
}
func antPublishUpdatePayload(t *testing.T, f antUpdateFixture, p codexSkillPayload) {
	t.Helper()
	if _, err := publishCodexSkillPayload(f.hub, p); err != nil {
		t.Fatal(err)
	}
	writeMaintenanceMutation199File(t, filepath.Join(f.hub, "version.json"), []byte(`{"version":"`+p.SourceVersion+`"}`))
}
func TestCodexAntSkillUpdateVersions(t *testing.T) {
	t.Run("prerelease-comparison", func(t *testing.T) {
		for _, tc := range []struct {
			a, b string
			want int
		}{
			{"1.0.99-test", "1.0.79", 1}, {"1.0.79-rc.1", "1.0.79", -1},
			{"1.0.79", "1.0.79-rc.1", 1}, {"1.0.80-rc.2", "1.0.80-rc.10", -1},
			{"1.0.80-2", "1.0.80-beta", -1}, {"1.0.80-beta", "1.0.80-2", 1},
		} {
			if got := compareVersions(tc.a, tc.b); got != tc.want {
				t.Errorf("compare %s %s = %d want %d", tc.a, tc.b, got, tc.want)
			}
		}
	})

	t.Run("compatible-newer-inventory", func(t *testing.T) {
		f := newAntUpdateFixture(t)
		p := f.payload
		p.SourceVersion = "1.0.80"
		p.Commands = append(p.Commands, "future")
		content := []byte("---\nname: ant-future\ndescription: A compatible future entry\n---\nfuture producer's bytes\n")
		p.Files = append(p.Files, codexSkillPayloadFile{RelativePath: "ant-future/SKILL.md", Mode: 0644, Content: content, SHA256: lifecycleDigest(content)})
		antPublishUpdatePayload(t, f, p)
		if _, _, err := antUpdateRun(t); err != nil {
			t.Fatal(err)
		}
		antAssertPublishedHome(t, f.hub, f.home)
		next := f.payload
		next.SourceVersion = "1.0.80"
		antPublishUpdatePayload(t, f, next)
		_, output, err := antUpdateRun(t)
		if err != nil {
			t.Fatal(err)
		}
		if string(mustReadLifecycleFixtureFile(t, filepath.Join(f.skills, "ant-future/SKILL.md"))) != string(content) || !strings.Contains(output, "outside desired inventory") {
			t.Fatal("future entry lost or silently forgotten")
		}
	})
	for _, kind := range []string{"missing", "partial", "digest", "schema", "runtime", "downgrade", "hub-version"} {
		t.Run(kind, func(t *testing.T) {
			f := newAntUpdateFixture(t)
			p := f.payload
			if kind == "downgrade" {
				newer := f.payload
				newer.SourceVersion = "1.0.80"
				if r := syncCodexSkillsFromPayload(newer, f.home); len(r.errors) > 0 {
					t.Fatal(r.errors)
				}
			}
			root := filepath.Join(f.hub, "system", "codex-skills")
			manifest := filepath.Join(root, "manifest.json")
			switch kind {
			case "missing":
				if err := os.Remove(manifest); err != nil {
					t.Fatal(err)
				}
			case "partial":
				if err := os.Remove(filepath.Join(root, p.Files[0].RelativePath)); err != nil {
					t.Fatal(err)
				}
			case "digest":
				writeMaintenanceMutation199File(t, filepath.Join(root, p.Files[0].RelativePath), []byte("tampered"))
			case "schema":
				p.SchemaVersion = "unsupported/v2"
			case "runtime":
				p.MinRuntimeVersion = "999.0.0"
			case "hub-version":
				p.SourceVersion = "1.0.81"
			}
			if kind == "schema" || kind == "runtime" || kind == "hub-version" {
				raw, _ := json.Marshal(p)
				writeMaintenanceMutation199File(t, manifest, raw)
			}
			before := antSnapshot(t, f.home)
			repoBefore := antSnapshot(t, f.repo)
			_, _, err := antUpdateRun(t)
			if err == nil {
				t.Fatal("invalid update accepted")
			}
			antAssertSnapshot(t, f.home, before)
			antAssertSnapshot(t, f.repo, repoBefore)
		})
	}
}
func TestCodexAntSkillUpdateChannel(t *testing.T) {
	f := newAntUpdateFixture(t)
	source := createMockSourceCheckout(t, "1.0.79")
	if errs := installHubErrors(setupInstallHub(filepath.Join(f.home, ".aether-dev"), source, "1.0.79")); len(errs) > 0 {
		t.Fatal(errs)
	}
	before := antSnapshot(t, f.skills)
	result, _, err := antUpdateRun(t, "--channel", "dev")
	if err != nil {
		t.Fatal(err)
	}
	antAssertSnapshot(t, f.skills, before)
	if len(antSkillPreviewTargets(antUpdatePreview(t, result))) != 0 {
		t.Fatal("dev planned stable homes")
	}
	all := antSnapshot(t, f.home)
	repo := antSnapshot(t, f.repo)
	if _, _, err := antUpdateRun(t, "--channel", "dev", "--sync-platform-homes"); err == nil {
		t.Fatal("dev opt-in accepted")
	}
	antAssertSnapshot(t, f.home, all)
	antAssertSnapshot(t, f.repo, repo)
}
