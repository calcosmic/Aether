package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/storage"
)

const (
	repositoryBootstrapHelperEnv  = "AETHER_TEST_REPOSITORY_BOOTSTRAP_HELPER"
	repositoryBootstrapArgsEnv    = "AETHER_TEST_REPOSITORY_BOOTSTRAP_ARGS"
	repositoryBootstrapReadyEnv   = "AETHER_TEST_REPOSITORY_BOOTSTRAP_READY"
	repositoryBootstrapReleaseEnv = "AETHER_TEST_REPOSITORY_BOOTSTRAP_RELEASE"
	repositoryContainmentMessage  = "storage: repository containment refused"
)

type repositoryBootstrapResult200 struct {
	output   string
	exitCode int
}

// TestRepositoryBootstrapContainment200 exercises rootCmd in a fresh process.
// The child is the package test binary rather than a mocked command: it calls
// Execute, so Cobra's real PersistentPreRunE and the selected command both run.
func TestRepositoryBootstrapContainment200(t *testing.T) {
	if os.Getenv(repositoryBootstrapHelperEnv) == "1" {
		runRepositoryBootstrapHelper200()
		return
	}

	t.Run("outside custom data root is refused without mutation", func(t *testing.T) {
		fixture := newRepositoryBootstrapFixture200(t)
		outsideData := filepath.Join(fixture.outside, "missing", "parents", "data")
		assertRepositoryBootstrapRefusal200(t, fixture, fixture.repo, outsideData)
	})

	t.Run("adjacent prefix is not repository containment", func(t *testing.T) {
		fixture := newRepositoryBootstrapFixture200(t)
		adjacent := fixture.repo + "-adjacent"
		if err := os.MkdirAll(adjacent, 0755); err != nil {
			t.Fatalf("create adjacent directory: %v", err)
		}
		fixture.outside = adjacent
		assertRepositoryBootstrapRefusal200(t, fixture, fixture.repo, filepath.Join(adjacent, "data"))
	})

	t.Run("intermediate aether link is refused without mutation", func(t *testing.T) {
		fixture := newRepositoryBootstrapFixture200(t)
		target := filepath.Join(fixture.outside, "aether-target")
		if err := os.MkdirAll(target, 0755); err != nil {
			t.Fatalf("create outside aether target: %v", err)
		}
		mustRepositorySymlink200(t, target, filepath.Join(fixture.repo, ".aether"))
		assertRepositoryBootstrapRefusal200(t, fixture, fixture.repo, filepath.Join(fixture.repo, ".aether", "data"))
	})

	t.Run("data link is refused without mutation", func(t *testing.T) {
		fixture := newRepositoryBootstrapFixture200(t)
		if err := os.MkdirAll(filepath.Join(fixture.repo, ".aether"), 0755); err != nil {
			t.Fatalf("create real .aether directory: %v", err)
		}
		target := filepath.Join(fixture.outside, "data-target")
		if err := os.MkdirAll(target, 0755); err != nil {
			t.Fatalf("create outside data target: %v", err)
		}
		mustRepositorySymlink200(t, target, filepath.Join(fixture.repo, ".aether", "data"))
		assertRepositoryBootstrapRefusal200(t, fixture, fixture.repo, filepath.Join(fixture.repo, ".aether", "data"))
	})

	t.Run("hostile locks link is refused before data mutation", func(t *testing.T) {
		fixture := newRepositoryBootstrapFixture200(t)
		if err := os.MkdirAll(filepath.Join(fixture.repo, ".aether", "data"), 0755); err != nil {
			t.Fatalf("create real data directory: %v", err)
		}
		target := filepath.Join(fixture.outside, "locks-target")
		if err := os.MkdirAll(target, 0755); err != nil {
			t.Fatalf("create outside locks target: %v", err)
		}
		mustRepositorySymlink200(t, target, filepath.Join(fixture.repo, ".aether", "locks"))
		assertRepositoryBootstrapRefusal200(t, fixture, fixture.repo, filepath.Join(fixture.repo, ".aether", "data"))
	})

	for _, tc := range []struct {
		name       string
		dataRoot   string
		workingDir func(*repositoryBootstrapFixture200) string
	}{
		{name: "explicit empty data root", dataRoot: "", workingDir: func(f *repositoryBootstrapFixture200) string { return f.repo }},
		{name: "dot data root", dataRoot: ".", workingDir: func(f *repositoryBootstrapFixture200) string { return f.repo }},
		{name: "dot-dot data root", dataRoot: "..", workingDir: func(f *repositoryBootstrapFixture200) string {
			work := filepath.Join(f.repo, "work", "nested")
			if err := os.MkdirAll(work, 0755); err != nil {
				t.Fatalf("create nested working directory: %v", err)
			}
			return work
		}},
	} {
		t.Run(tc.name+" is refused", func(t *testing.T) {
			fixture := newRepositoryBootstrapFixture200(t)
			workingDir := tc.workingDir(fixture)
			before := snapshotRepositoryTree200(t, fixture.base)
			result := runRepositoryBootstrapProcess200(t, fixture, workingDir, tc.dataRoot, nil)
			assertRepositoryRefusalResult200(t, result)
			after := snapshotRepositoryTree200(t, fixture.base)
			if before != after {
				t.Fatalf("degenerate data root mutated the fixture\nbefore:\n%s\nafter:\n%s", before, after)
			}
		})
	}

	t.Run("component swap after validation fails closed", func(t *testing.T) {
		fixture := newRepositoryBootstrapFixture200(t)
		aetherDir := filepath.Join(fixture.repo, ".aether")
		if err := os.MkdirAll(filepath.Join(aetherDir, "data"), 0755); err != nil {
			t.Fatalf("create original data path: %v", err)
		}
		control := filepath.Join(fixture.base, "control")
		if err := os.MkdirAll(control, 0700); err != nil {
			t.Fatalf("create swap control directory: %v", err)
		}
		ready := filepath.Join(control, "ready")
		release := filepath.Join(control, "release")

		command, output := startRepositoryBootstrapProcess200(t, fixture, fixture.repo,
			filepath.Join(fixture.repo, ".aether", "data"), map[string]string{
				repositoryBootstrapReadyEnv:   ready,
				repositoryBootstrapReleaseEnv: release,
			})
		waitForRepositoryPath200(t, ready)

		captured := filepath.Join(fixture.outside, "captured-aether")
		redirect := filepath.Join(fixture.outside, "redirect-aether")
		if err := os.Rename(aetherDir, captured); err != nil {
			t.Fatalf("move validated .aether directory: %v", err)
		}
		if err := os.MkdirAll(redirect, 0755); err != nil {
			t.Fatalf("create redirect target: %v", err)
		}
		mustRepositorySymlink200(t, redirect, aetherDir)
		repoAfterAttacker := snapshotRepositoryTree200(t, fixture.repo)
		outsideAfterAttacker := snapshotRepositoryTree200(t, fixture.outside)

		if err := os.WriteFile(release, []byte("release\n"), 0600); err != nil {
			t.Fatalf("release bootstrap child: %v", err)
		}
		result := finishRepositoryBootstrapProcess200(t, command, output)
		assertRepositoryRefusalResult200(t, result)
		if got := snapshotRepositoryTree200(t, fixture.repo); got != repoAfterAttacker {
			t.Fatalf("refused swap mutated repository beyond the attacker change\nbefore:\n%s\nafter:\n%s", repoAfterAttacker, got)
		}
		if got := snapshotRepositoryTree200(t, fixture.outside); got != outsideAfterAttacker {
			t.Fatalf("refused swap mutated outside tree beyond the attacker change\nbefore:\n%s\nafter:\n%s", outsideAfterAttacker, got)
		}
	})

	t.Run("same basename paths use distinct lock identities", func(t *testing.T) {
		locksDir := filepath.Join(t.TempDir(), "locks")
		locker, err := storage.NewFileLocker(locksDir)
		if err != nil {
			t.Fatalf("create locker: %v", err)
		}
		for _, path := range []string{"one/shared.json", "two/shared.json"} {
			if err := locker.Lock(path); err != nil {
				t.Fatalf("lock %s: %v", path, err)
			}
			if err := locker.Unlock(path); err != nil {
				t.Fatalf("unlock %s: %v", path, err)
			}
		}
		entries, err := os.ReadDir(locksDir)
		if err != nil {
			t.Fatalf("read locks directory: %v", err)
		}
		var lockNames []string
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".lock") {
				lockNames = append(lockNames, entry.Name())
			}
		}
		sort.Strings(lockNames)
		if len(lockNames) != 2 || lockNames[0] == lockNames[1] {
			t.Fatalf("same-basename paths must create two distinct locks, got %v", lockNames)
		}
	})

	t.Run("first read-only status creates nothing", func(t *testing.T) {
		fixture := newRepositoryBootstrapFixture200(t)
		before := snapshotRepositoryTree200(t, fixture.base)
		result := runRepositoryBootstrapProcess200(t, fixture, fixture.repo,
			filepath.Join(fixture.repo, ".aether", "data"), nil, "status")
		if result.exitCode != 0 {
			t.Fatalf("read-only status failed with %d:\n%s", result.exitCode, result.output)
		}
		if after := snapshotRepositoryTree200(t, fixture.base); after != before {
			t.Fatalf("first read-only status created filesystem state\nbefore:\n%s\nafter:\n%s", before, after)
		}
	})

	t.Run("allowed mutation creates only verified repository data and locks", func(t *testing.T) {
		fixture := newRepositoryBootstrapFixture200(t)
		outsideBefore := snapshotRepositoryTree200(t, fixture.outside)
		result := runRepositoryBootstrapProcess200(t, fixture, fixture.repo,
			filepath.Join(fixture.repo, ".aether", "data"), nil)
		if result.exitCode != 0 {
			t.Fatalf("allowed mutation failed with %d:\n%s", result.exitCode, result.output)
		}
		if got := snapshotRepositoryTree200(t, fixture.outside); got != outsideBefore {
			t.Fatalf("allowed mutation touched outside tree\nbefore:\n%s\nafter:\n%s", outsideBefore, got)
		}
		statePath := filepath.Join(fixture.repo, ".aether", "data", "pending-decisions.json")
		if _, err := os.Stat(statePath); err != nil {
			t.Fatalf("allowed mutation did not create repository data: %v", err)
		}
		if _, err := os.Stat(filepath.Join(fixture.repo, ".aether", "locks")); err != nil {
			t.Fatalf("allowed mutation did not create repository locks: %v", err)
		}
		if _, err := os.Lstat(filepath.Join(fixture.repo, ".aether", "data", ".welcomed")); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("JSON-mode mutation unexpectedly created first-run marker: %v", err)
		}
		assertRepositoryMutationScope200(t, fixture.repo)
	})
}

type repositoryBootstrapFixture200 struct {
	base    string
	repo    string
	outside string
	home    string
}

func newRepositoryBootstrapFixture200(t *testing.T) *repositoryBootstrapFixture200 {
	t.Helper()
	base := t.TempDir()
	fixture := &repositoryBootstrapFixture200{
		base:    base,
		repo:    filepath.Join(base, "repo"),
		outside: filepath.Join(base, "outside"),
		home:    filepath.Join(base, "home"),
	}
	for _, dir := range []string{fixture.repo, fixture.outside, fixture.home} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("create fixture directory %s: %v", dir, err)
		}
	}
	if err := os.MkdirAll(filepath.Join(base, "tmp"), 0700); err != nil {
		t.Fatalf("create fixture temp directory: %v", err)
	}
	return fixture
}

func assertRepositoryBootstrapRefusal200(t *testing.T, fixture *repositoryBootstrapFixture200, workingDir, dataRoot string) {
	t.Helper()
	repoBefore := snapshotRepositoryTree200(t, fixture.repo)
	outsideBefore := snapshotRepositoryTree200(t, fixture.outside)
	result := runRepositoryBootstrapProcess200(t, fixture, workingDir, dataRoot, nil)
	assertRepositoryRefusalResult200(t, result)
	if got := snapshotRepositoryTree200(t, fixture.repo); got != repoBefore {
		t.Fatalf("refused bootstrap mutated repository\nbefore:\n%s\nafter:\n%s", repoBefore, got)
	}
	if got := snapshotRepositoryTree200(t, fixture.outside); got != outsideBefore {
		t.Fatalf("refused bootstrap mutated outside tree\nbefore:\n%s\nafter:\n%s", outsideBefore, got)
	}
}

func assertRepositoryRefusalResult200(t *testing.T, result repositoryBootstrapResult200) {
	t.Helper()
	if result.exitCode == 0 {
		t.Fatalf("hostile repository bootstrap succeeded:\n%s", result.output)
	}
	if !strings.Contains(result.output, repositoryContainmentMessage) {
		t.Fatalf("refusal diagnostic is not stable; want %q in:\n%s", repositoryContainmentMessage, result.output)
	}
}

func runRepositoryBootstrapProcess200(t *testing.T, fixture *repositoryBootstrapFixture200, workingDir, dataRoot string, extraEnv map[string]string, args ...string) repositoryBootstrapResult200 {
	t.Helper()
	command, output := startRepositoryBootstrapProcess200(t, fixture, workingDir, dataRoot, extraEnv, args...)
	return finishRepositoryBootstrapProcess200(t, command, output)
}

func startRepositoryBootstrapProcess200(t *testing.T, fixture *repositoryBootstrapFixture200, workingDir, dataRoot string, extraEnv map[string]string, args ...string) (*exec.Cmd, *bytes.Buffer) {
	t.Helper()
	if len(args) == 0 {
		args = []string{"flag-add", "--title", "containment-proof", "--type", "issue"}
	}
	encodedArgs, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("encode helper arguments: %v", err)
	}
	replacements := map[string]string{
		"AETHER_HUB_DIR":              "",
		"AETHER_OUTPUT_MODE":          "json",
		"AETHER_ROOT":                 fixture.repo,
		"CODEX_HOME":                  filepath.Join(fixture.home, ".codex"),
		"COLONY_DATA_DIR":             dataRoot,
		"HOME":                        fixture.home,
		"NO_COLOR":                    "1",
		"TMPDIR":                      filepath.Join(fixture.base, "tmp"),
		"USERPROFILE":                 fixture.home,
		repositoryBootstrapHelperEnv:  "1",
		repositoryBootstrapArgsEnv:    string(encodedArgs),
		repositoryBootstrapReadyEnv:   "",
		repositoryBootstrapReleaseEnv: "",
	}
	for key, value := range extraEnv {
		replacements[key] = value
	}
	command := exec.Command(os.Args[0], "-test.run=^TestRepositoryBootstrapContainment200$")
	command.Dir = workingDir
	command.Env = replaceProcessEnv(os.Environ(), replacements)
	output := &bytes.Buffer{}
	command.Stdout = output
	command.Stderr = output
	if err := command.Start(); err != nil {
		t.Fatalf("start Cobra helper process: %v", err)
	}
	return command, output
}

func finishRepositoryBootstrapProcess200(t *testing.T, command *exec.Cmd, output *bytes.Buffer) repositoryBootstrapResult200 {
	t.Helper()
	err := command.Wait()
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("wait for Cobra helper process: %v", err)
		}
		exitCode = exitErr.ExitCode()
	}
	return repositoryBootstrapResult200{output: output.String(), exitCode: exitCode}
}

func runRepositoryBootstrapHelper200() {
	var args []string
	if err := json.Unmarshal([]byte(os.Getenv(repositoryBootstrapArgsEnv)), &args); err != nil {
		fmt.Fprintf(os.Stderr, "decode Cobra helper arguments: %v\n", err)
		os.Exit(97)
	}
	if ready := os.Getenv(repositoryBootstrapReadyEnv); ready != "" {
		release := os.Getenv(repositoryBootstrapReleaseEnv)
		repositoryBootstrapValidatedHook = func() error {
			if err := os.WriteFile(ready, []byte("ready\n"), 0600); err != nil {
				return fmt.Errorf("signal validation barrier: %w", err)
			}
			deadline := time.Now().Add(10 * time.Second)
			for {
				if _, err := os.Stat(release); err == nil {
					return nil
				} else if !errors.Is(err, os.ErrNotExist) {
					return fmt.Errorf("inspect validation release: %w", err)
				}
				if time.Now().After(deadline) {
					return fmt.Errorf("timed out waiting for validation release")
				}
				time.Sleep(5 * time.Millisecond)
			}
		}
	}
	stdout = os.Stdout
	stderr = os.Stderr
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
	rootCmd.SetArgs(args)
	if err := Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

func waitForRepositoryPath200(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(path); err == nil {
			return
		} else if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("inspect barrier path %s: %v", path, err)
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for bootstrap validation barrier %s", path)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func mustRepositorySymlink200(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		if runtime.GOOS == "windows" {
			t.Fatalf("create Windows reparse-point symlink %s -> %s (developer mode or symlink privilege is required for this supported attack): %v", link, target, err)
		}
		t.Fatalf("create symlink %s -> %s: %v", link, target, err)
	}
}

func snapshotRepositoryTree200(t *testing.T, root string) string {
	t.Helper()
	if _, err := os.Lstat(root); errors.Is(err, os.ErrNotExist) {
		return "<missing>\n"
	} else if err != nil {
		t.Fatalf("inspect snapshot root %s: %v", root, err)
	}
	var rows []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		kind := info.Mode().Type().String()
		payload := ""
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			payload = "target=" + target
		case info.Mode().IsRegular():
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			sum := sha256.Sum256(data)
			payload = "sha256=" + hex.EncodeToString(sum[:])
		}
		rows = append(rows, fmt.Sprintf("%s|%s|%04o|%s", filepath.ToSlash(rel), kind, info.Mode().Perm(), payload))
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot %s: %v", root, err)
	}
	sort.Strings(rows)
	return strings.Join(rows, "\n") + "\n"
}

func assertRepositoryMutationScope200(t *testing.T, repo string) {
	t.Helper()
	allowedPrefixes := []string{".aether/data/", ".aether/locks/"}
	err := filepath.WalkDir(repo, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(repo, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "." || rel == ".aether" || rel == ".aether/data" || rel == ".aether/locks" {
			return nil
		}
		for _, prefix := range allowedPrefixes {
			if strings.HasPrefix(rel, prefix) {
				return nil
			}
		}
		return fmt.Errorf("unexpected repository mutation %s", rel)
	})
	if err != nil {
		t.Fatal(err)
	}
}
