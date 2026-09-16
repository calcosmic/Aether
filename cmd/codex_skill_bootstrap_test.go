package cmd

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	aetherassets "github.com/calcosmic/Aether"
)

// Keep this separate from blackBoxBinaries: go install does not supply the
// release linker stamp. Running outside the checkout must exercise that path.
func TestCodexAntSkillUnstampedEmbeddedInstall(t *testing.T) {
	source := findTestModuleRoot(t)
	binary := filepath.Join(t.TempDir(), "aether")
	build := exec.Command("go", "build", "-o", binary, "./cmd/aether")
	build.Dir = source
	output, err := build.CombinedOutput()
	t.Logf("unstamped build args=%q error=%v output=%s", build.Args, err, output)
	if err != nil {
		t.Fatal(err)
	}
	h := codexBootstrapHost(t, binary)
	result := h.run(t, "install")
	t.Logf("embedded install exit=%d stdout=%s stderr=%s", result.ExitCode, result.Stdout, result.Stderr)
	assertBlackBoxSuccess(t, "unstamped embedded install", result)
	hub := filepath.Join(h.home, ".aether")
	payload := antAssertPublishedHome(t, hub, h.home)
	if payload.SourceVersion != readRepoVersion(source) || readHubVersionAtPath(hub) != readRepoVersion(source) {
		t.Fatalf("embedded source/hub version did not survive publication: %+v", payload)
	}
	if !strings.HasPrefix(payload.GeneratorIdentity, "aether/"+readRepoVersion(source)) {
		t.Fatalf("generator does not identify its executable: %s", payload.GeneratorIdentity)
	}
	assertCodexSkillFixtureInstalled(t, source, h.home)
	root := filepath.Join(h.home, ".codex", "skills", "aether")
	var names []string
	for _, dir := range findSkillDirs(root) {
		names = append(names, filepath.Base(dir))
	}
	sort.Strings(names)
	if !reflect.DeepEqual(names, expectedAntSkills) || len(antSnapshot(t, root)) != 13 {
		t.Fatalf("expected exactly nine public skills, three supports and ownership; names=%v", names)
	}
	if len(payload.Files) != 12 || len(antSnapshot(t, filepath.Join(hub, "system", "codex-skills"))) != 13 {
		t.Fatal("embedded hub payload is incomplete or has extra files")
	}
	for _, rel := range []string{"workers.md", "commands", "docs", "skills", "codex", "commands/claude", "commands/opencode"} {
		if _, err := os.Stat(filepath.Join(hub, "system", rel)); err != nil {
			t.Fatalf("normal embedded hub missing %s: %v", rel, err)
		}
	}
}

func TestCodexAntSkillOlderStampedInstallRefused(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "aether")
	build := exec.Command("go", "build", "-ldflags", "-X github.com/calcosmic/Aether/cmd.Version=1.0.78", "-o", binary, "./cmd/aether")
	build.Dir = findTestModuleRoot(t)
	output, err := build.CombinedOutput()
	t.Logf("older stamped build args=%q error=%v output=%s", build.Args, err, output)
	if err != nil {
		t.Fatal(err)
	}
	for _, external := range []bool{false, true} {
		name := "embedded-fresh-home"
		if external {
			name = "high-external-source-hub-and-cwd"
		}
		t.Run(name, func(t *testing.T) {
			h := codexBootstrapHost(t, binary)
			args := []string{"install"}
			if external {
				source := t.TempDir()
				if err := aetherassets.MaterializeInstallPackage(source); err != nil {
					t.Fatal(err)
				}
				for _, root := range []string{source, h.repo} {
					writeMaintenanceMutation199File(t, filepath.Join(root, ".aether", "version.json"), []byte(`{"version":"999.0.0"}`))
					writeMaintenanceMutation199File(t, filepath.Join(root, "go.mod"), []byte("module github.com/calcosmic/Aether\n"))
				}
				writeMaintenanceMutation199File(t, filepath.Join(h.home, ".aether", "version.json"), []byte(`{"version":"999.0.0"}`))
				args = append(args, "--package-dir", source)
			}
			before := antSnapshot(t, h.home)
			result := h.run(t, args...)
			t.Logf("refused install args=%q exit=%d stdout=%s stderr=%s", args, result.ExitCode, result.Stdout, result.Stderr)
			if result.ExitCode != 2 || !strings.Contains(result.Stdout+result.Stderr, "runtime 1.0.78 is older than required 1.0.79") {
				t.Fatalf("incompatible stamped executable not refused: %+v", result)
			}
			antAssertSnapshot(t, h.home, before)
			for _, rel := range []string{".aether/system", ".codex", ".claude", ".config"} {
				if _, err := os.Stat(filepath.Join(h.home, rel)); !os.IsNotExist(err) {
					t.Fatalf("refusal created destination %s: %v", rel, err)
				}
			}
		})
	}
}

func TestCodexAntSkillUnstampedMinimumUsesExecutable(t *testing.T) {
	// External desired payload, installed hub and cwd all advertise a future
	// version. None may authorize this unstamped executable to accept that minimum.
	payload, err := buildCodexSkillPayload(findTestModuleRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	prior := Version
	Version = "0.0.0-dev"
	t.Cleanup(func() { Version = prior })
	home, cwd := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	t.Setenv("AETHER_HUB_DIR", filepath.Join(home, ".aether"))
	writeMaintenanceMutation199File(t, filepath.Join(home, ".aether", "version.json"), []byte(`{"version":"999.0.0"}`))
	writeMaintenanceMutation199File(t, filepath.Join(cwd, "go.mod"), []byte("module github.com/calcosmic/Aether\n"))
	writeMaintenanceMutation199File(t, filepath.Join(cwd, ".aether", "version.json"), []byte(`{"version":"999.0.0"}`))
	t.Chdir(cwd)
	payload.SourceVersion, payload.MinRuntimeVersion = "999.0.0", "999.0.0"
	before := antSnapshot(t, home)
	if _, err := publishCodexSkillPayload(filepath.Join(home, ".aether"), payload); err == nil || !strings.Contains(err.Error(), "older than required 999.0.0") {
		t.Fatalf("external version authorized incompatible publication: %v", err)
	}
	if result := syncCodexSkillsFromPayload(payload, home); len(result.errors) != 1 || !strings.Contains(result.errors[0], "older than required 999.0.0") {
		t.Fatalf("external version authorized incompatible home sync: %+v", result)
	}
	antAssertSnapshot(t, home, before)
	// Exercise the published-envelope loader as well as the producer/home paths.
	hub := t.TempDir()
	manifest, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(hub, "system", "codex-skills")
	writeMaintenanceMutation199File(t, filepath.Join(root, "manifest.json"), manifest)
	for _, file := range payload.Files {
		writeMaintenanceMutation199File(t, filepath.Join(root, file.RelativePath), file.Content)
	}
	if _, err := loadCodexSkillPayload(hub); err == nil || !strings.Contains(err.Error(), "older than required 999.0.0") {
		t.Fatalf("external version authorized incompatible loaded payload: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".codex")); !os.IsNotExist(err) {
		t.Fatalf("refused home sync created Codex home: %v", err)
	}
}

func codexBootstrapHost(t *testing.T, binary string) *cliBlackBox {
	t.Helper()
	root := t.TempDir()
	home, repo, tmp := filepath.Join(root, "home"), filepath.Join(root, "consumer"), filepath.Join(root, "tmp")
	for _, dir := range []string{home, repo, tmp} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	return &cliBlackBox{binary: binary, home: home, repo: repo, env: replaceProcessEnv(os.Environ(), map[string]string{
		"HOME": home, "USERPROFILE": home, "CODEX_HOME": filepath.Join(home, ".codex"),
		"XDG_CONFIG_HOME": filepath.Join(home, ".config"), "XDG_DATA_HOME": filepath.Join(home, ".local", "share"),
		"XDG_CACHE_HOME": filepath.Join(home, ".cache"), "XDG_STATE_HOME": filepath.Join(home, ".local", "state"),
		"AETHER_HUB_DIR": filepath.Join(home, ".aether"), "AETHER_ROOT": repo,
		"COLONY_DATA_DIR": filepath.Join(repo, ".aether", "data"), "TMPDIR": tmp,
		"AETHER_OUTPUT_MODE": "json", "NO_COLOR": "1",
	})}
}
