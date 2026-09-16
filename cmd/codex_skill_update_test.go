package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
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

// Every matrix row executes the registered controller. Expected outcomes are
// specified here, independently of the generator and update target inventory.
func TestCodexAntSkillUpdateMatrix(t *testing.T) {
	for _, kind := range []string{"clean", "legacy", "force", "repeat", "force-repeat", "collision", "force-collision", "malformed", "version-mismatch", "dev", "dev-forbidden", "removal-only", "preview"} {
		t.Run(kind, func(t *testing.T) {
			f := newAntUpdateFixture(t)
			args := []string{}
			refuse := false
			switch kind {
			case "clean":
				for _, old := range antLegacy(t).Files {
					if err := os.Remove(filepath.Join(f.skills, old.RelativePath)); err != nil {
						t.Fatal(err)
					}
				}
			case "force":
				args = append(args, "--force")
			case "repeat", "force-repeat", "removal-only":
				if _, _, err := antUpdateRun(t); err != nil {
					t.Fatal(err)
				}
				if kind == "force-repeat" {
					args = append(args, "--force")
				}
				if kind == "removal-only" {
					antSeedLegacy(t, f.skills)
				}
			case "collision", "force-collision":
				writeMaintenanceMutation199File(t, filepath.Join(f.skills, "ant-plan/SKILL.md"), []byte("custom canonical instructions"))
				if kind == "force-collision" {
					args = append(args, "--force")
				}
				refuse = true
			case "malformed":
				writeMaintenanceMutation199File(t, filepath.Join(f.hub, "system/codex-skills/manifest.json"), []byte("{invalid"))
				refuse = true
			case "version-mismatch":
				p := f.payload
				p.SourceVersion = "1.0.80"
				if _, err := publishCodexSkillPayload(f.hub, p); err != nil {
					t.Fatal(err)
				}
				refuse = true
			case "dev", "dev-forbidden":
				if errs := installHubErrors(setupInstallHub(filepath.Join(f.home, ".aether-dev"), createMockSourceCheckout(t, "1.0.79"), "1.0.79")); len(errs) != 0 {
					t.Fatal(errs)
				}
				args = append(args, "--channel", "dev")
				if kind == "dev-forbidden" {
					args = append(args, "--sync-platform-homes")
					refuse = true
				}
			case "preview":
				args = append(args, "--dry-run")
			}
			homeBefore, repoBefore := antSnapshot(t, f.home), antSnapshot(t, f.repo)
			skillsBefore := antSnapshot(t, f.skills)
			mtimes := antUpdateMtimes(t, f.home, f.repo)
			result, output, err := antUpdateRun(t, args...)
			if refuse {
				if err == nil || result["receipt"] != nil || result["state_effect"] != "none" {
					t.Fatalf("refusal missing: %v %s", err, output)
				}
				antAssertSnapshot(t, f.home, homeBefore)
				antAssertSnapshot(t, f.repo, repoBefore)
				return
			}
			if err != nil {
				t.Fatalf("registered %s: %v %s", kind, err, output)
			}
			if kind == "preview" {
				antAssertSnapshot(t, f.home, homeBefore)
				antAssertSnapshot(t, f.repo, repoBefore)
				if !reflect.DeepEqual(mtimes, antUpdateMtimes(t, f.home, f.repo)) {
					t.Fatal("preview changed file/directory mtimes")
				}
				for _, p := range []string{filepath.Join(f.home, ".codex/.aether-skill-locks"), filepath.Join(f.repo, ".aether/data/transactions"), filepath.Join(f.repo, lifecycleTransactionDirectory), filepath.Join(f.home, ".codex", lifecycleTransactionDirectory)} {
					if _, err := os.Stat(p); !os.IsNotExist(err) {
						t.Fatalf("preview created lock/journal %s: %v", p, err)
					}
				}
				if result["receipt"] != nil || result["state_effect"] != "none" || len(antSkillPreviewTargets(antUpdatePreview(t, result))) != 27 {
					t.Fatal("invalid preview")
				}
				return
			}
			if kind == "dev" {
				antAssertSnapshot(t, f.skills, skillsBefore)
				if len(antSkillPreviewTargets(antUpdatePreview(t, result))) != 0 {
					t.Fatal("dev entered stable skill targets")
				}
				return
			}
			antAssertPublishedHome(t, f.hub, f.home)
			var names []string
			for _, dir := range findSkillDirs(f.skills) {
				names = append(names, filepath.Base(dir))
			}
			sort.Strings(names)
			if !reflect.DeepEqual(names, expectedAntSkills) {
				t.Fatalf("public names = %v", names)
			}
			writes, removals := 0, 0
			for _, target := range antSkillPreviewTargets(antUpdatePreview(t, result)) {
				if target.Change == maintenanceMutationChangeWrite {
					writes++
				}
				if target.Change == maintenanceMutationChangeRemove {
					removals++
				}
			}
			if kind == "repeat" || kind == "force-repeat" {
				antAssertSnapshot(t, f.skills, skillsBefore)
				if writes != 0 || removals != 0 || strings.Contains(fmt.Sprint(result["restart_targets"]), "skill") {
					t.Fatalf("repeat changed skills/restart: %s", output)
				}
			} else {
				if result["receipt"] == nil || result["state_effect"] != "committed" {
					t.Fatalf("missing successful transaction: %s", output)
				}
				if kind == "removal-only" && (writes != 0 || removals != 14 || !strings.Contains(fmt.Sprint(result["restart_targets"]), "Codex skill shims")) {
					t.Fatalf("removal-only restart missing: %s", output)
				}
			}
		})
	}
}

func antUpdateMtimes(t *testing.T, roots ...string) map[string]int64 {
	t.Helper()
	out := map[string]int64{}
	for _, root := range roots {
		if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			info, err := entry.Info()
			if err == nil {
				out[path] = info.ModTime().UnixNano()
			}
			return err
		}); err != nil {
			t.Fatal(err)
		}
	}
	return out
}

func TestCodexAntSkillUpdatePostPlanCreationRefused(t *testing.T) {
	antUpdatePostPlanRefused(t, false)
}
func TestCodexAntSkillUpdatePostPlanModeChangeRefused(t *testing.T) {
	antUpdatePostPlanRefused(t, true)
}
func antUpdatePostPlanRefused(t *testing.T, modeOnly bool) {
	t.Helper()
	f := newAntUpdateFixture(t)
	if modeOnly {
		if _, _, err := antUpdateRun(t); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(f.skills, "ant-plan/SKILL.md")
	originalState, err := readLifecycleFileState(path)
	if err != nil {
		t.Fatal(err)
	}
	ownerPath := filepath.Join(f.skills, ".aether-owned.json")
	ownerBefore, err := readLifecycleFileState(ownerPath)
	if err != nil {
		t.Fatal(err)
	}
	originalLocker := maintenanceCodexSkillLocker
	t.Cleanup(func() { maintenanceCodexSkillLocker = originalLocker })
	reached := false
	var postEditHome, postEditRepo map[string]lifecycleFileState
	maintenanceCodexSkillLocker = func(plan maintenanceMutationPlan) (func() error, error) {
		if reached {
			t.Fatal("unexpected replan/reentry")
		}
		reached = true
		found := false
		for _, target := range plan.Targets {
			if !isCodexSkillTarget(target) {
				continue
			}
			if target.ExpectedDigest == "" || target.ExpectedMode == nil {
				t.Fatal("original skill baseline was dropped")
			}
			state, err := readLifecycleFileState(filepath.Join(plan.Allowlist.CodexHome, target.RelativeTarget))
			if err != nil {
				t.Fatal(err)
			}
			if target.ExpectedDigest != state.Digest || *target.ExpectedMode != state.Mode.Perm() {
				t.Fatal("original boundary expectations differ")
			}
			if filepath.ToSlash(target.RelativeTarget) == "skills/aether/ant-plan/SKILL.md" {
				found = true
				if target.ExpectedDigest != originalState.Digest || *target.ExpectedMode != originalState.Mode.Perm() {
					t.Fatal("wrong canonical baseline")
				}
				if modeOnly && *target.ExpectedMode != 0644 {
					t.Fatal("fixture is not owned 0644")
				}
				if !modeOnly && (target.ExpectedDigest != lifecycleTransactionMissingDigest || *target.ExpectedMode != 0) {
					t.Fatal("missing-state baseline lost")
				}
			}
		}
		if !found {
			t.Fatal("registered update did not assemble skill targets")
		}
		if modeOnly {
			if err := os.Chmod(path, 0600); err != nil {
				t.Fatal(err)
			}
		} else {
			writeMaintenanceMutation199File(t, path, []byte("owner created this after controller preview"))
			if err := os.Chmod(path, 0600); err != nil {
				t.Fatal(err)
			}
		}
		postEditHome, postEditRepo = antSnapshot(t, f.home), antSnapshot(t, f.repo)
		// Delegate unchanged, including all original expectations, to the real lock.
		return originalLocker(plan)
	}
	result, output, err := antUpdateRun(t) // normal registered update, never --force
	if !reached || err == nil || !strings.Contains(err.Error(), "baseline") || !strings.Contains(err.Error(), "ant-plan") {
		t.Fatalf("post-plan edit not refused at real commit: %v %s", err, output)
	}
	if result["receipt"] != nil || result["state_effect"] != "none" {
		t.Fatalf("refusal claimed success: %s", output)
	}
	// The real lock may leave its lock file. Nothing else may be created/changed,
	// including any previous journal, any new journal, or the ownership record.
	for key := range antSnapshot(t, f.home) {
		if strings.HasPrefix(filepath.ToSlash(key), ".codex/.aether-skill-locks/") {
			delete(postEditHome, key)
		}
	}
	got := antSnapshot(t, f.home)
	for key := range got {
		if strings.HasPrefix(filepath.ToSlash(key), ".codex/.aether-skill-locks/") {
			delete(got, key)
		}
	}
	if !reflect.DeepEqual(got, postEditHome) {
		t.Fatal("refusal mutated home targets or journals")
	}
	antAssertSnapshot(t, f.repo, postEditRepo)
	ownerAfter, err := readLifecycleFileState(ownerPath)
	if err != nil || !reflect.DeepEqual(ownerBefore, ownerAfter) {
		t.Fatalf("ownership changed: %v", err)
	}
	after, err := readLifecycleFileState(path)
	if err != nil || after.Mode.Perm() != 0600 || (modeOnly && after.Digest != originalState.Digest) {
		t.Fatalf("post-edit bytes/mode changed: %v", err)
	}
}

func TestCodexAntSkillUpdatePreservationReceipt(t *testing.T) {
	for _, force := range []bool{false, true} {
		t.Run(fmt.Sprintf("force-%t", force), func(t *testing.T) {
			f := newAntUpdateFixture(t)
			for _, name := range []string{"agents-md-template.md", "codex-md-template.md"} {
				content, err := os.ReadFile(filepath.Join(antSkillSourceRoot(t), ".aether/templates", name))
				if err != nil {
					t.Fatal(err)
				}
				writeMaintenanceMutation199File(t, filepath.Join(f.hub, "system/templates", name), content)
			}
			paths := []string{filepath.Join(f.skills, "aether-plan/SKILL.md"), filepath.Join(f.skills, "aether-colony-creation/SKILL.md"), filepath.Join(f.skills, "ant-future/SKILL.md"), filepath.Join(f.repo, "AGENTS.md"), filepath.Join(f.repo, ".codex/CODEX.md")}
			before := map[string]lifecycleFileState{}
			for _, path := range paths {
				writeMaintenanceMutation199File(t, path, []byte("owner instructions: "+filepath.Base(filepath.Dir(path))))
				if err := os.Chmod(path, 0600); err != nil {
					t.Fatal(err)
				}
				state, err := readLifecycleFileState(path)
				if err != nil {
					t.Fatal(err)
				}
				before[path] = state
			}
			args := []string{}
			if force {
				args = append(args, "--force")
			}
			result, output, err := antUpdateRun(t, args...)
			if err != nil || result["state_effect"] != "committed" {
				t.Fatalf("preservation update: %v %s", err, output)
			}
			for path, state := range before {
				after, err := readLifecycleFileState(path)
				if err != nil || !reflect.DeepEqual(state, after) {
					t.Fatalf("lost custom bytes/mode at %s: %v", path, err)
				}
			}
			for _, rel := range []string{"aether-plan/SKILL.md", "aether-colony-creation/SKILL.md", "ant-future/SKILL.md"} {
				if !strings.Contains(fmt.Sprint(result["preserved_codex_skills"]), rel) {
					t.Fatalf("not reported as preserved: %s", rel)
				}
				for _, target := range antSkillPreviewTargets(antUpdatePreview(t, result)) {
					if filepath.ToSlash(target.RelativeTarget) == "skills/aether/"+rel {
						t.Fatalf("custom path claimed managed: %s", rel)
					}
				}
			}
			for _, rel := range []string{"AGENTS.md", ".codex/CODEX.md"} {
				if !strings.Contains(fmt.Sprint(result["preserved_project_docs"]), rel) {
					t.Fatalf("unreported custom doc: %s", rel)
				}
			}
			if !strings.Contains(fmt.Sprint(result["message"]), "Preserved") {
				t.Fatal("missing preservation explanation")
			}
		})
	}
}
