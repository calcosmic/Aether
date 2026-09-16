package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func antPayloadCommand(t *testing.T, args ...string) (string, error) {
	t.Helper()
	resetRootCmd(t)
	var out bytes.Buffer
	stdout, stderr = &out, &out
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	t.Logf("registered %v: error=%v output=%s", args, err, out.String())
	return out.String(), err
}
func antPayloadEnvironment(t *testing.T) {
	t.Helper()
	saveGlobals(t)
	t.Setenv("AETHER_HUB_DIR", "")
	t.Setenv("AETHER_OUTPUT_MODE", "json")
}
func antReadPublished(t *testing.T, hub string) codexSkillPayload {
	t.Helper()
	root := filepath.Join(hub, "system", "codex-skills")
	raw, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var p codexSkillPayload
	if err = json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	for i := range p.Files {
		p.Files[i].Content = mustReadLifecycleFixtureFile(t, filepath.Join(root, p.Files[i].RelativePath))
	}
	if err = validateCodexSkillPayload(p); err != nil {
		t.Fatal(err)
	}
	return p
}
func antAssertPublishedHome(t *testing.T, hub, home string) codexSkillPayload {
	t.Helper()
	p := antReadPublished(t, hub)
	root := filepath.Join(home, ".codex", "skills", "aether")
	state, err := readLifecycleFileState(filepath.Join(root, ".aether-owned.json"))
	if err != nil {
		t.Fatal(err)
	}
	owner, err := readCodexOwnership(state)
	if err != nil {
		t.Fatal(err)
	}
	if owner.PayloadIdentity != codexSkillPayloadIdentity(p) {
		t.Fatalf("home identity %s != published %s", owner.PayloadIdentity, codexSkillPayloadIdentity(p))
	}
	for _, f := range p.Files {
		for _, base := range []string{root, filepath.Join(hub, "system", "codex-skills")} {
			s, e := readLifecycleFileState(filepath.Join(base, f.RelativePath))
			if e != nil || !bytes.Equal(s.Bytes, f.Content) || uint32(s.Mode.Perm()) != f.Mode {
				t.Fatalf("published bytes/mode mismatch %s: %v", f.RelativePath, e)
			}
		}
	}
	t.Logf("payload=%s source=%s generator=%s files=%d", codexSkillPayloadIdentity(p), p.SourceVersion, p.GeneratorIdentity, len(p.Files))
	return p
}
func TestCodexAntSkillPublishedPayload(t *testing.T) {
	antPayloadEnvironment(t)
	for _, command := range []string{"publish", "install"} {
		t.Run(command, func(t *testing.T) {
			home := t.TempDir()
			source := createMockSourceCheckout(t, "1.0.80")
			out, err := antPayloadCommand(t, command, "--package-dir", source, "--home-dir", home, "--channel", "stable", "--skip-build-binary")
			if err != nil || !strings.Contains(out, `"ok": true`) {
				t.Fatalf("%s failed: %v %s", command, err, out)
			}
			p := antAssertPublishedHome(t, filepath.Join(home, ".aether"), home)
			if p.SourceVersion != "1.0.80" || len(p.Commands) != 9 || len(p.Files) != 12 {
				t.Fatalf("wrong manifest %+v", p)
			}
		})
	}
}
func TestCodexAntSkillPayloadValidation(t *testing.T) {
	antPayloadEnvironment(t)
	for _, kind := range []string{"absent", "empty", "partial", "digest", "schema", "runtime", "mode", "traversal"} {
		t.Run(kind, func(t *testing.T) {
			home := t.TempDir()
			source := createMockSourceCheckout(t, "1.0.79")
			hub := filepath.Join(home, ".aether")
			p, err := buildCodexSkillPayload(source)
			if err != nil {
				t.Fatal(err)
			}
			root := filepath.Join(hub, "system", "codex-skills")
			for _, f := range p.Files {
				writeMaintenanceMutation199File(t, filepath.Join(root, f.RelativePath), f.Content)
			}
			switch kind {
			case "partial":
				if err := os.Remove(filepath.Join(root, p.Files[0].RelativePath)); err != nil {
					t.Fatal(err)
				}
			case "digest":
				writeMaintenanceMutation199File(t, filepath.Join(root, p.Files[0].RelativePath), []byte("changed"))
			case "schema":
				p.SchemaVersion = "future/v99"
			case "runtime":
				p.MinRuntimeVersion = "999.0.0"
			case "mode":
				if err := os.Chmod(filepath.Join(root, p.Files[0].RelativePath), 0600); err != nil {
					t.Fatal(err)
				}
			case "traversal":
				p.Files[0].RelativePath = "../outside"
			}
			raw, _ := json.Marshal(p)
			if kind == "empty" {
				raw = nil
			}
			if kind != "absent" {
				writeMaintenanceMutation199File(t, filepath.Join(root, "manifest.json"), raw)
			}
			before := antSnapshot(t, home)
			_, errs := syncPlatformHomeAssets(source, home, channelStable, false)
			if len(errs) == 0 {
				t.Fatal("invalid published payload accepted")
			}
			antAssertSnapshot(t, home, before)
		})
	}
	t.Run("aggregate-hub-errors", func(t *testing.T) {
		for _, command := range []string{"publish", "install"} {
			t.Run(command, func(t *testing.T) {
				source := createMockSourceCheckout(t, "1.0.79")
				home := t.TempDir()
				writeMaintenanceMutation199File(t, filepath.Join(source, ".codex", "agents", "aether-bad.toml"), []byte("malformed = ["))
				out, err := antPayloadCommand(t, command, "--package-dir", source, "--home-dir", home, "--skip-build-binary")
				if err == nil && strings.Contains(out, `"ok": true`) {
					t.Fatalf("hidden hub error: %s", out)
				}
				if _, e := os.Stat(filepath.Join(home, ".codex", "skills", "aether")); !os.IsNotExist(e) {
					t.Fatalf("home mutated on hub error: %v", e)
				}
			})
		}
	})
}
func TestCodexAntSkillPublishRepeat(t *testing.T) {
	antPayloadEnvironment(t)
	home := t.TempDir()
	source := createMockSourceCheckout(t, "1.0.79")
	args := []string{"publish", "--package-dir", source, "--home-dir", home, "--skip-build-binary"}
	if _, err := antPayloadCommand(t, args...); err != nil {
		t.Fatal(err)
	}
	antAssertPublishedHome(t, filepath.Join(home, ".aether"), home)
	payloadRoot := filepath.Join(home, ".aether", "system", "codex-skills")
	skillsRoot := filepath.Join(home, ".codex", "skills", "aether")
	before := antSnapshot(t, payloadRoot)
	installed := antSnapshot(t, skillsRoot)
	if _, err := antPayloadCommand(t, args...); err != nil {
		t.Fatal(err)
	}
	antAssertSnapshot(t, payloadRoot, before)
	antAssertSnapshot(t, skillsRoot, installed)
	antAssertPublishedHome(t, filepath.Join(home, ".aether"), home)
}
func TestCodexAntSkillPublishChannel(t *testing.T) {
	antPayloadEnvironment(t)
	for _, command := range []string{"publish", "install"} {
		t.Run(command, func(t *testing.T) {
			home := t.TempDir()
			stable := createMockSourceCheckout(t, "1.0.79")
			dev := createMockSourceCheckout(t, "1.0.80")
			if _, err := antPayloadCommand(t, command, "--package-dir", stable, "--home-dir", home, "--channel", "stable", "--skip-build-binary"); err != nil {
				t.Fatal(err)
			}
			skillRoot := filepath.Join(home, ".codex", "skills", "aether")
			before := antSnapshot(t, skillRoot)
			args := []string{command, "--package-dir", dev, "--home-dir", home, "--channel", "dev", "--skip-build-binary"}
			if _, err := antPayloadCommand(t, args...); err != nil {
				t.Fatal(err)
			}
			antAssertSnapshot(t, skillRoot, before)
			antReadPublished(t, filepath.Join(home, ".aether-dev"))
			out, err := antPayloadCommand(t, append(args, "--sync-platform-homes")...)
			if err != nil || !strings.Contains(out, "stable platform skills") {
				t.Fatalf("explicit scope missing: %v %s", err, out)
			}
			antAssertPublishedHome(t, filepath.Join(home, ".aether-dev"), home)
		})
	}
}
