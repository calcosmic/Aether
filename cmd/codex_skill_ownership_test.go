package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
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
	future := antPayload(t)
	future.SourceVersion = "1.0.80"
	future.Commands = append(future.Commands, "future")
	rel := "ant-future/SKILL.md"
	body := []byte("---\nname: ant-future\n---\nFuture content\n")
	future.Files = append(future.Files, codexSkillPayloadFile{RelativePath: rel, SHA256: lifecycleDigest(body), Mode: 0644, Content: body})
	root := filepath.Join(f.codex, "skills/aether")
	antCommit(t, antPlan(t, f, "newer-producer", future))
	// A compatible later payload that omits an unfamiliar entry cannot retire it.
	next := antPayload(t)
	next.SourceVersion = "1.0.81"
	antCommit(t, antPlan(t, f, "retain", next))
	if got := mustReadLifecycleFixtureFile(t, filepath.Join(root, rel)); !bytes.Equal(got, body) {
		t.Fatal("newer entry lost")
	}
	if !strings.Contains(string(mustReadLifecycleFixtureFile(t, filepath.Join(root, ".aether-owned.json"))), rel) {
		t.Fatal("newer ownership forgotten")
	}
	before := antSnapshot(t, root)
	plan := f.plan("downgrade")
	if err := planCodexSkillTargets(&plan, antPayload(t)); err == nil || !strings.Contains(err.Error(), "downgrade") {
		t.Fatalf("downgrade allowed: %v", err)
	}
	antAssertSnapshot(t, root, before)
}

func antLogExecutionIdentity(t *testing.T, payload codexSkillPayload) {
	t.Helper()
	exe, err := os.Open(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	defer exe.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, exe); err != nil {
		t.Fatal(err)
	}
	t.Logf("Executed test binary sha256:%x; payload %s; source version %s; generator %s", hash.Sum(nil), codexSkillPayloadIdentity(payload), payload.SourceVersion, payload.GeneratorIdentity)
}

func TestCodexAntSkillBaselineCapture(t *testing.T) {
	f := newMaintenanceMutation199Fixture(t)
	p := antPayload(t)
	antLogExecutionIdentity(t, p)
	root := filepath.Join(f.codex, "skills", "aether")
	antSeedLegacy(t, root)
	for _, id := range []string{"migration", "unchanged"} {
		if id == "unchanged" {
			// The manifest's desired mode is 0644, but classification must capture 0600.
			if err := os.Chmod(filepath.Join(root, ".aether-owned.json"), 0600); err != nil {
				t.Fatal(err)
			}
		}
		plan := antPlan(t, f, id, p)
		want := 27 // 12 desired, 14 exact retirement baselines, 1 ownership.
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

func antNoMutationReceipt(t *testing.T, f maintenanceMutation199Fixture, plan maintenanceMutationPlan, before map[string]lifecycleFileState) {
	t.Helper()
	r, err := commitMaintenanceMutation(plan)
	if err == nil || !strings.Contains(err.Error(), "baseline") {
		t.Fatalf("want baseline refusal, got %v", err)
	}
	if r.Receipt != nil || r.StateEffect != colony.LifecycleStateEffectNone {
		t.Fatalf("false mutation receipt: %+v", r)
	}
	antAssertSnapshot(t, filepath.Join(f.codex, "skills/aether"), before)
	for _, p := range []string{filepath.Join(f.data, "transactions", plan.TransactionID), filepath.Join(f.codex, lifecycleTransactionDirectory, plan.TransactionID)} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("refusal created transaction evidence: %s (%v)", p, err)
		}
	}
}
func TestCodexAntSkillPostPlanCreationRefused(t *testing.T) {
	f := newMaintenanceMutation199Fixture(t)
	plan := antPlan(t, f, "post-create", antPayload(t))
	if _, err := prepareMaintenanceMutation(plan); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(f.codex, "skills/aether")
	writeMaintenanceMutation199File(t, filepath.Join(root, "ant-plan/SKILL.md"), []byte("new custom planning instructions"))
	antNoMutationReceipt(t, f, plan, antSnapshot(t, root))
}
func TestCodexAntSkillPostPlanModeChangeRefused(t *testing.T) {
	for _, rel := range []string{"ant-plan/SKILL.md", "support/aether-colony-creation.md", "aether-plan/SKILL.md", ".aether-owned.json"} {
		t.Run(rel, func(t *testing.T) {
			f := newMaintenanceMutation199Fixture(t)
			p := antPayload(t)
			root := filepath.Join(f.codex, "skills/aether")
			antCommit(t, antPlan(t, f, "seed", p))
			antSeedLegacy(t, root)
			plan := antPlan(t, f, "post-mode", p)
			if _, err := prepareMaintenanceMutation(plan); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(filepath.Join(root, rel), 0600); err != nil {
				t.Fatal(err)
			}
			antNoMutationReceipt(t, f, plan, antSnapshot(t, root))
		})
	}
}
func TestCodexAntSkillBaselineCompatibility(t *testing.T) {
	for _, kind := range []string{"missing-digest", "missing-mode", "zero-mode", "other-caller", "lock-directory-symlink", "lock-file-symlink"} {
		t.Run(kind, func(t *testing.T) {
			f := newMaintenanceMutation199Fixture(t)
			plan := antPlan(t, f, "compat", antPayload(t))
			if strings.HasPrefix(kind, "lock-") {
				physical, err := filepath.EvalSymlinks(f.codex)
				if err != nil {
					t.Fatal(err)
				}
				lockDir := filepath.Join(physical, ".aether-skill-locks")
				outside := t.TempDir()
				marker := filepath.Join(outside, "owner")
				writeMaintenanceMutation199File(t, marker, []byte("untouched"))
				if kind == "lock-directory-symlink" {
					err = os.Symlink(outside, lockDir)
				} else {
					if err = os.MkdirAll(lockDir, 0755); err != nil {
						t.Fatal(err)
					}
					identity := filepath.Join(physical, "skills/aether/.aether-owned.json")
					filename := strings.TrimPrefix(lifecycleDigest([]byte(filepath.ToSlash(identity))), "sha256:") + ".lock"
					err = os.Symlink(marker, filepath.Join(lockDir, filename))
				}
				if err != nil {
					t.Fatal(err)
				}
				before := antSnapshot(t, outside)
				if result, err := commitMaintenanceMutation(plan); err == nil || result.Receipt != nil {
					t.Fatalf("unsafe lock accepted: %+v %v", result, err)
				}
				antAssertSnapshot(t, outside, before)
				if _, err := os.Stat(filepath.Join(f.data, "transactions")); !os.IsNotExist(err) {
					t.Fatal("unsafe lock created journal")
				}
				return
			}
			switch kind {
			case "missing-digest":
				plan.Targets[0].ExpectedDigest = ""
			case "missing-mode":
				plan.Targets[0].ExpectedMode = nil
			case "zero-mode": // zero is required for a missing path and is not a wildcard on an existing file.
				target := &plan.Targets[0]
				writeMaintenanceMutation199File(t, filepath.Join(f.codex, target.RelativeTarget), target.Content)
				target.ExpectedDigest = lifecycleDigest(target.Content)
			case "other-caller":
				plan.Targets = []maintenanceMutationTarget{{Root: lifecycleTransactionRootRepository, RelativeTarget: "unrelated.txt", Source: "fixture", Content: []byte("after"), Managed: true}}
				writeMaintenanceMutation199File(t, filepath.Join(f.repository, "unrelated.txt"), []byte("before"))
			}
			_, err := prepareMaintenanceMutation(plan)
			if kind == "other-caller" {
				if err != nil {
					t.Fatal(err)
				}
				antCommit(t, plan)
			} else if err == nil {
				t.Fatal("missing/mismatched skill baseline accepted")
			}
		})
	}
}
func TestCodexAntSkillReplay(t *testing.T) {
	f := newMaintenanceMutation199Fixture(t)
	p := antPayload(t)
	root := filepath.Join(f.codex, "skills/aether")
	antSeedLegacy(t, root)
	antCommit(t, antPlan(t, f, "first", p))
	before := antSnapshot(t, root)
	r := antCommit(t, antPlan(t, f, "second", p))
	antAssertSnapshot(t, root, before)
	if r.Receipt != nil || r.StateEffect != colony.LifecycleStateEffectNone {
		t.Fatal("unchanged sync reports a change")
	}
	for _, target := range r.Targets {
		if target.Change != maintenanceMutationChangeUnchanged {
			t.Errorf("false restart-worthy change: %s", target.RelativeTarget)
		}
	}
	tx, err := beginLifecycleTransaction(lifecycleTransactionConfig{TransactionID: "first", Command: "update", Allowlist: f.allowlist})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(); err == nil || !strings.Contains(err.Error(), "verified transaction") {
		t.Fatalf("verified rollback accepted: %v", err)
	}
	antAssertSnapshot(t, root, before)
}
func TestCodexAntSkillPreview(t *testing.T) {
	f := newMaintenanceMutation199Fixture(t)
	root := filepath.Join(f.codex, "skills/aether")
	antSeedLegacy(t, root)
	before := antSnapshot(t, f.codex)
	times := map[string]time.Time{}
	for rel := range before {
		info, err := os.Stat(filepath.Join(f.codex, rel))
		if err != nil {
			t.Fatal(err)
		}
		times[rel] = info.ModTime()
	}
	plan := antPlan(t, f, "preview", antPayload(t))
	if _, err := prepareMaintenanceMutation(plan); err != nil {
		t.Fatal(err)
	}
	antAssertSnapshot(t, f.codex, before)
	if _, err := os.Stat(filepath.Join(f.codex, ".aether-skill-locks")); !os.IsNotExist(err) {
		t.Fatalf("preview created lock directory: %v", err)
	}
	for rel, want := range times {
		info, err := os.Stat(filepath.Join(f.codex, rel))
		if err != nil || !info.ModTime().Equal(want) {
			t.Fatalf("preview mtime changed: %s (%v)", rel, err)
		}
	}
	if entries, err := os.ReadDir(f.data); err != nil || len(entries) != 0 {
		t.Fatalf("preview wrote journal: %v %v", entries, err)
	}
}
func antChangedPayload(t *testing.T) codexSkillPayload {
	t.Helper()
	p := antPayload(t)
	p.SourceVersion = "1.0.80"
	p.Files[0].Content = append(append([]byte(nil), p.Files[0].Content...), []byte("\nUpdated instructions.\n")...)
	p.Files[0].SHA256 = lifecycleDigest(p.Files[0].Content)
	return p
}
func TestCodexAntSkillRollback(t *testing.T) {
	for _, kind := range []string{"legacy-after-write", "legacy-after-removal", "after-write", "after-removal", "later-edit", "interrupted", "interrupted-edit"} {
		t.Run(kind, func(t *testing.T) {
			f := newMaintenanceMutation199Fixture(t)
			root := filepath.Join(f.codex, "skills/aether")
			if !strings.HasPrefix(kind, "legacy-") {
				antCommit(t, antPlan(t, f, "seed", antPayload(t)))
			}
			antSeedLegacy(t, root)
			if !strings.HasPrefix(kind, "legacy-") {
				if err := os.Chmod(filepath.Join(root, ".aether-owned.json"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			before := antSnapshot(t, root)
			plan := antPlan(t, f, "rollback", antChangedPayload(t))
			// Put a real removal immediately after the changed public file.
			for i, target := range plan.Targets {
				if target.Action == lifecycleTransactionRemove {
					plan.Targets[1], plan.Targets[i] = plan.Targets[i], plan.Targets[1]
					break
				}
			}
			editPath := filepath.Join(f.codex, plan.Targets[0].RelativeTarget)
			injected := errors.New("injected failure")
			point := "after_target_commit:target-0002"
			if strings.HasSuffix(kind, "after-write") {
				point = "after_target_commit:target-0001"
			}
			faultReached := false
			plan.Fault = func(at string) error {
				if at != point {
					return nil
				}
				faultReached = true
				if kind == "later-edit" || kind == "interrupted-edit" {
					writeMaintenanceMutation199File(t, editPath, []byte("later owner edit"))
					if err := os.Chmod(editPath, 0600); err != nil {
						t.Fatal(err)
					}
				}
				return injected
			}
			var result maintenanceMutationResult
			var err error
			if strings.HasPrefix(kind, "interrupted") {
				unlock, lockErr := lockCodexSkillTargets(plan)
				if lockErr != nil {
					t.Fatal(lockErr)
				}
				defer unlock()
				tx, e := beginLifecycleTransaction(lifecycleTransactionConfig{TransactionID: plan.TransactionID, Command: plan.Operation, Allowlist: plan.Allowlist, Fault: plan.Fault})
				if e != nil {
					t.Fatal(e)
				}
				preview, e := prepareMaintenanceMutation(plan)
				if e != nil {
					t.Fatal(e)
				}
				for i, target := range plan.Targets {
					if preview.Targets[i].Change == maintenanceMutationChangeUnchanged {
						continue
					}
					if target.Action == lifecycleTransactionRemove {
						e = tx.DeclareRemoval(target.Root, target.RelativeTarget)
					} else {
						e = tx.DeclareWriteWithMode(target.Root, target.RelativeTarget, target.Content, target.Mode)
					}
					if e != nil {
						t.Fatal(e)
					}
				}
				if _, e = tx.Commit(); !errors.Is(e, injected) {
					t.Fatalf("interruption not reached: %v", e)
				}
				// A newly opened coordinator reads the durable unverified journal.
				fresh, e := beginLifecycleTransaction(lifecycleTransactionConfig{TransactionID: plan.TransactionID, Command: plan.Operation, Allowlist: plan.Allowlist})
				if e != nil {
					t.Fatal(e)
				}
				err = fresh.Rollback()
				if err == nil {
					result.StateEffect = colony.LifecycleStateEffectRolledBack
				} else {
					result.StateEffect = fresh.progress.StateEffect
				}
			} else {
				result, err = commitMaintenanceMutation(plan)
				if !faultReached || err == nil {
					t.Fatalf("fault not observed: %v", err)
				}
			}
			if strings.Contains(kind, "edit") {
				if result.StateEffect != colony.LifecycleStateEffectRecoveryRequired || err == nil {
					t.Fatalf("later edit must require recovery: %+v %v", result, err)
				}
				state, e := readLifecycleFileState(editPath)
				if e != nil || string(state.Bytes) != "later owner edit" || state.Mode.Perm() != 0600 {
					t.Fatalf("lost later edit: %+v %v", state, e)
				}
				if got := mustReadLifecycleFixtureFile(t, filepath.Join(root, ".aether-owned.json")); string(got) != string(before[".aether-owned.json"].Bytes) {
					t.Fatal("ownership altered on recovery refusal")
				}
			} else {
				if result.StateEffect != colony.LifecycleStateEffectRolledBack {
					t.Fatalf("did not roll back: %+v %v", result, err)
				}
				antAssertSnapshot(t, root, before)
			}
		})
	}
}
func TestCodexAntSkillChannelIsolation(t *testing.T) {
	t.Setenv("AETHER_HUB_DIR", "")
	home := t.TempDir()
	root := filepath.Join(home, ".codex/skills/aether")
	antSeedLegacy(t, root)
	before := antSnapshot(t, home)
	results, errs := syncPlatformHomeAssets(antSkillSourceRoot(t), home, channelDev, false)
	if len(errs) != 0 || len(results) != 1 {
		t.Fatalf("dev default: %v %v", results, errs)
	}
	antAssertSnapshot(t, home, before)
	for _, optIn := range []bool{false, true} {
		collisionHome := t.TempDir()
		skillRoot := filepath.Join(collisionHome, ".codex/skills/aether")
		writeMaintenanceMutation199File(t, filepath.Join(skillRoot, "ant-plan/SKILL.md"), []byte("custom"))
		saved := antSnapshot(t, skillRoot)
		if _, err := publishCodexSkillPayload(filepath.Join(collisionHome, ".aether"), antPayload(t)); err != nil {
			t.Fatal(err)
		}
		_, errs := syncPlatformHomeAssets(antSkillSourceRoot(t), collisionHome, channelStable, optIn)
		if len(errs) == 0 || !strings.Contains(strings.Join(errs, " "), "collision") {
			t.Fatalf("force=%v authorized custom collision: %v", optIn, errs)
		}
		antAssertSnapshot(t, skillRoot, saved)
	}
	f := newMaintenanceMutation199Fixture(t)
	plan := antPlan(t, f, "dev", antPayload(t))
	plan.Channel = channelDev
	plan.Allowlist.Hub.Channel = lifecycleTransactionHubDev
	if _, err := prepareMaintenanceMutation(plan); err == nil {
		t.Fatal("dev transaction wrote stable skill targets")
	}
}

func TestCodexAntSkillConcurrentCommit(t *testing.T) {
	if config := os.Getenv("AETHER_TEST_SKILL_LOCK_CHILD"); config != "" {
		var allow lifecycleTransactionAllowlist
		if err := json.Unmarshal(mustReadLifecycleFixtureFile(t, config), &allow); err != nil {
			t.Fatal(err)
		}
		f := maintenanceMutation199Fixture{repository: allow.RepositoryRoot, data: allow.LifecycleDataRoot, hub: allow.Hub.Path, codex: allow.CodexHome, allowlist: allow}
		plan := antPlan(t, f, "child-new", antChangedPayload(t))
		if _, err := prepareMaintenanceMutation(plan); err != nil {
			t.Fatal(err)
		}
		original := maintenanceCodexSkillLocker
		maintenanceCodexSkillLocker = func(p maintenanceMutationPlan) (func() error, error) {
			if err := os.WriteFile(config+".planned", []byte("entered common commit lock boundary"), 0600); err != nil {
				return nil, err
			}
			return lockCodexSkillTargets(p)
		}
		t.Cleanup(func() { maintenanceCodexSkillLocker = original })
		antCommit(t, plan)
		return
	}
	f := newMaintenanceMutation199Fixture(t)
	antLogExecutionIdentity(t, antPayload(t))
	old := antPlan(t, f, "parent-old", antPayload(t))
	physical, err := filepath.EvalSymlinks(f.codex)
	if err != nil {
		t.Fatal(err)
	}
	locker, err := storage.NewFileLocker(filepath.Join(physical, ".aether-skill-locks"))
	if err != nil {
		t.Fatal(err)
	}
	identity := filepath.Join(physical, "skills/aether/.aether-owned.json")
	if err := locker.Lock(identity); err != nil {
		t.Fatal(err)
	}
	held := true
	t.Cleanup(func() {
		if held {
			_ = locker.Unlock(identity)
		}
	})
	// A different project's coordinator shares exactly the same selected home.
	other := newMaintenanceMutation199Fixture(t)
	allow := other.allowlist
	allow.CodexHome = f.codex
	config := filepath.Join(t.TempDir(), "child.json")
	raw, _ := json.Marshal(allow)
	writeMaintenanceMutation199File(t, config, raw)
	var output bytes.Buffer
	child := exec.Command(os.Args[0], "-test.run=^TestCodexAntSkillConcurrentCommit$", "-test.count=1", "-test.timeout=30s")
	child.Env = append(os.Environ(), "AETHER_TEST_SKILL_LOCK_CHILD="+config)
	child.Stdout = &output
	child.Stderr = &output
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = child.Process.Kill() })
	done := make(chan error, 1)
	go func() { done <- child.Wait() }()
	deadline := time.Now().Add(15 * time.Second)
	for {
		if _, err := os.Stat(config + ".planned"); err == nil {
			break
		}
		select {
		case err := <-done:
			t.Fatalf("child before planning: %v %s", err, output.String())
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("child planning timeout")
		}
		time.Sleep(10 * time.Millisecond)
	}
	select {
	case err := <-done:
		t.Fatalf("common commit ignored cross-process home lock: %v %s", err, output.String())
	case <-time.After(500 * time.Millisecond):
	}
	if _, err := os.Stat(filepath.Join(f.codex, "skills/aether/.aether-owned.json")); !os.IsNotExist(err) {
		t.Fatalf("child wrote under held lock: %v", err)
	}
	if err := locker.Unlock(identity); err != nil {
		t.Fatal(err)
	}
	held = false
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("child commit: %v %s", err, output.String())
		}
	case <-time.After(15 * time.Second):
		t.Fatal("lock was not released")
	}
	t.Logf("Separate process %d completed under the shared home lock: %s", child.Process.Pid, output.String())
	before := antSnapshot(t, filepath.Join(f.codex, "skills/aether"))
	antNoMutationReceipt(t, f, old, before)
	// New classification also refuses the older desired version, without mutation.
	downgrade := f.plan("older-inventory")
	if err := planCodexSkillTargets(&downgrade, antPayload(t)); err == nil || !strings.Contains(err.Error(), "downgrade") {
		t.Fatalf("older inventory accepted: %v", err)
	}
	antAssertSnapshot(t, filepath.Join(f.codex, "skills/aether"), before)
}
