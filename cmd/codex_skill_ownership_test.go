package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

type antLegacyFixture struct {
	SourceRevision string `json:"source_revision"`
	Files          []struct {
		RelativePath string `json:"relative_path"`
		OriginalPath string `json:"original_path"`
		SHA256       string `json:"sha256"`
		Mode         uint32 `json:"mode"`
		Body         string `json:"body"`
	} `json:"files"`
}

func antLegacy(t *testing.T) antLegacyFixture {
	t.Helper()
	var fixture antLegacyFixture
	if err := json.Unmarshal(mustReadLifecycleFixtureFile(t, "testdata/codex-skills/legacy-v1.0.79.json"), &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.SourceRevision != "404731ccffda7bbca64ce801b74b0752a7161315" {
		t.Fatal("wrong legacy provenance")
	}
	want := []string{"aether-build", "aether-colonize", "aether-colony-build-cycle", "aether-colony-creation", "aether-colony-research", "aether-command-guide", "aether-continue", "aether-discuss", "aether-init", "aether-oracle", "aether-plan", "aether-seal", "aether-skill-loader", "aether-swarm"}
	var got []string
	for _, f := range fixture.Files {
		got = append(got, strings.TrimSuffix(f.RelativePath, "/SKILL.md"))
		if f.SHA256 != lifecycleDigest([]byte(f.Body)) || f.Mode != 0644 || f.OriginalPath != "~/.codex/skills/aether/"+f.RelativePath {
			t.Fatalf("invalid frozen file: %s", f.RelativePath)
		}
	}
	sort.Strings(got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("legacy inventory: %v", got)
	}
	return fixture
}

func antSeedLegacy(t *testing.T, root string) {
	t.Helper()
	for _, f := range antLegacy(t).Files {
		writeMaintenanceMutation199File(t, filepath.Join(root, f.RelativePath), []byte(f.Body))
	}
}
func antPayload(t *testing.T) codexSkillPayload {
	t.Helper()
	p, err := buildCodexSkillPayload(antSkillSourceRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	return p
}
func antPlan(t *testing.T, f maintenanceMutation199Fixture, id string, p codexSkillPayload) maintenanceMutationPlan {
	t.Helper()
	plan := f.plan(id)
	if err := planCodexSkillTargets(&plan, p); err != nil {
		t.Fatal(err)
	}
	return plan
}
func antCommit(t *testing.T, plan maintenanceMutationPlan) maintenanceMutationResult {
	t.Helper()
	r, err := commitMaintenanceMutation(plan)
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func antSnapshot(t *testing.T, root string) map[string]lifecycleFileState {
	t.Helper()
	out := map[string]lifecycleFileState{}
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		if d.Type()&os.ModeSymlink != 0 {
			link, e := os.Readlink(p)
			if e != nil {
				return e
			}
			out[rel] = lifecycleFileState{Exists: true, Bytes: []byte(link), Mode: os.ModeSymlink}
			return nil
		}
		state, e := readLifecycleFileState(p)
		if e != nil {
			return e
		}
		out[rel] = state
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}
func antAssertSnapshot(t *testing.T, root string, before map[string]lifecycleFileState) {
	t.Helper()
	if got := antSnapshot(t, root); !reflect.DeepEqual(got, before) {
		t.Fatalf("bytes, modes or path inventory changed under %s", root)
	}
}

func TestCodexAntSkillLegacyMigration(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, ".codex", "skills", "aether")
	antSeedLegacy(t, root)
	for _, rel := range []string{"aether-plan/notes.md", "custom/SKILL.md", "domain/typescript/SKILL.md", "aether-unknown/SKILL.md"} {
		writeMaintenanceMutation199File(t, filepath.Join(root, rel), []byte("owner: "+rel))
	}
	if ok, output := runAntSkillInstall(t, home); !ok || !strings.Contains(output, "aether-unknown/SKILL.md") {
		t.Fatalf("registered install missing success/preservation report: %s", output)
	}
	for _, f := range antLegacy(t).Files {
		if _, err := os.Lstat(filepath.Join(root, f.RelativePath)); !os.IsNotExist(err) {
			t.Errorf("legacy entry not retired: %s (%v)", f.RelativePath, err)
		}
	}
	for _, name := range expectedAntSkills {
		if _, err := os.Stat(filepath.Join(root, name, "SKILL.md")); err != nil {
			t.Error(err)
		}
	}
	for _, rel := range []string{"aether-plan/notes.md", "custom/SKILL.md", "domain/typescript/SKILL.md", "aether-unknown/SKILL.md"} {
		if got := string(mustReadLifecycleFixtureFile(t, filepath.Join(root, rel))); got != "owner: "+rel {
			t.Fatalf("custom bytes lost: %s", rel)
		}
	}
}

func TestCodexAntSkillOwnership(t *testing.T) {
	for _, kind := range []string{"edited-legacy", "custom-frontmatter", "custom-plain", "source-shipped", "canonical-symlink", "legacy-symlink", "malformed-manifest", "modified-support", "deleted-ownership", "extra-sibling"} {
		t.Run(kind, func(t *testing.T) {
			f := newMaintenanceMutation199Fixture(t)
			root := filepath.Join(f.codex, "skills", "aether")
			p := antPayload(t)
			switch kind {
			case "edited-legacy":
				antSeedLegacy(t, root)
				writeMaintenanceMutation199File(t, filepath.Join(root, "aether-plan/SKILL.md"), []byte("edited legacy"))
			case "custom-frontmatter":
				writeMaintenanceMutation199File(t, filepath.Join(root, "ant-plan/SKILL.md"), []byte("---\nname: ant-plan\nsource: custom\n---\nowner"))
			case "custom-plain":
				writeMaintenanceMutation199File(t, filepath.Join(root, "ant-plan/SKILL.md"), []byte("owner"))
			case "source-shipped":
				writeMaintenanceMutation199File(t, filepath.Join(root, "ant-plan/SKILL.md"), []byte("---\nsource: shipped\n---\nowner"))
			case "canonical-symlink", "legacy-symlink":
				name := "ant-plan"
				if kind == "legacy-symlink" {
					name = "aether-plan"
				}
				writeMaintenanceMutation199File(t, filepath.Join(f.repository, "owner"), []byte("owner"))
				if err := os.MkdirAll(filepath.Join(root, name), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(f.repository, "owner"), filepath.Join(root, name, "SKILL.md")); err != nil {
					t.Fatal(err)
				}
			case "malformed-manifest":
				writeMaintenanceMutation199File(t, filepath.Join(root, ".aether-owned.json"), []byte("{}"))
			case "modified-support", "deleted-ownership":
				antCommit(t, antPlan(t, f, "seed", p))
				if kind == "modified-support" {
					writeMaintenanceMutation199File(t, filepath.Join(root, "support/aether-colony-creation.md"), []byte("owner"))
				} else if err := os.Remove(filepath.Join(root, ".aether-owned.json")); err != nil {
					t.Fatal(err)
				}
			case "extra-sibling":
				antSeedLegacy(t, root)
				writeMaintenanceMutation199File(t, filepath.Join(root, "aether-plan/notes.md"), []byte("owner"))
			}
			before := antSnapshot(t, root)
			plan := f.plan("ownership")
			err := planCodexSkillTargets(&plan, p)
			preserve := kind == "edited-legacy" || kind == "legacy-symlink" || kind == "extra-sibling"
			if !preserve {
				if err == nil {
					t.Fatal("unproven collision authorized")
				}
				if len(plan.Targets) != 0 {
					t.Fatal("partial targets on refusal")
				}
				antAssertSnapshot(t, root, before)
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			preview, err := prepareMaintenanceMutation(plan)
			if err != nil {
				t.Fatal(err)
			}
			if kind != "extra-sibling" {
				raw, _ := json.Marshal(preview)
				if !strings.Contains(string(raw), "aether-plan/SKILL.md") || !strings.Contains(string(raw), "preserv") {
					t.Fatal("unproven legacy preservation not reported")
				}
			}
			antCommit(t, plan)
			retained := "aether-plan/SKILL.md"
			if kind == "extra-sibling" {
				retained = "aether-plan/notes.md"
			}
			if got := antSnapshot(t, root)[retained]; !reflect.DeepEqual(got, before[retained]) {
				t.Fatal("preserved file changed")
			}
		})
	}
}

func TestCodexAntSkillNewerInventory(t *testing.T) {
	f := newMaintenanceMutation199Fixture(t)
	p := antPayload(t)
	root := filepath.Join(f.codex, "skills", "aether")
	antCommit(t, antPlan(t, f, "seed", p))
	ownerPath := filepath.Join(root, ".aether-owned.json")
	var o codexSkillOwnership
	if err := json.Unmarshal(mustReadLifecycleFixtureFile(t, ownerPath), &o); err != nil {
		t.Fatal(err)
	}
	// An unfamiliar owned entry is not an implicit retirement request.
	rel := "ant-future/SKILL.md"
	body := []byte("future content")
	writeMaintenanceMutation199File(t, filepath.Join(root, rel), body)
	o.Files = append(o.Files, codexSkillPayloadFile{RelativePath: rel, SHA256: lifecycleDigest(body), Mode: 0644})
	raw, _ := json.Marshal(o)
	writeMaintenanceMutation199File(t, ownerPath, raw)
	antCommit(t, antPlan(t, f, "retain", p))
	if got := string(mustReadLifecycleFixtureFile(t, filepath.Join(root, rel))); got != string(body) {
		t.Fatal("newer entry lost")
	}
	if !strings.Contains(string(mustReadLifecycleFixtureFile(t, ownerPath)), rel) {
		t.Fatal("newer ownership forgotten")
	}
	p.SourceVersion = "0.9.0"
	before := antSnapshot(t, root)
	plan := f.plan("downgrade")
	if err := planCodexSkillTargets(&plan, p); err == nil {
		t.Fatal("downgrade allowed")
	}
	antAssertSnapshot(t, root, before)
}

func TestCodexAntSkillBaselineCapture(t *testing.T) {
	f := newMaintenanceMutation199Fixture(t)
	p := antPayload(t)
	root := filepath.Join(f.codex, "skills", "aether")
	antSeedLegacy(t, root)
	for _, id := range []string{"migration", "unchanged"} {
		plan := antPlan(t, f, id, p)
		want := 27
		if id == "unchanged" {
			want = 27
		} // 12 desired, 14 exact retirement baselines, 1 ownership.
		if len(plan.Targets) != want {
			t.Errorf("%s: got %d targets want %d", id, len(plan.Targets), want)
		}
		for _, target := range plan.Targets {
			state, err := readLifecycleFileState(filepath.Join(f.codex, target.RelativeTarget))
			if err != nil {
				t.Fatal(err)
			}
			if target.ExpectedDigest == "" || target.ExpectedDigest != state.Digest {
				t.Errorf("%s: missing/or wrong digest expectation", target.RelativeTarget)
			}
			field := reflect.ValueOf(target).FieldByName("ExpectedMode")
			if !field.IsValid() || field.IsNil() {
				t.Errorf("%s: missing mode expectation", target.RelativeTarget)
				continue
			}
			if os.FileMode(field.Elem().Uint()) != state.Mode.Perm() {
				t.Errorf("%s: wrong captured mode", target.RelativeTarget)
			}
		}
		antCommit(t, plan)
	}
}
