package cmd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type stagedReleaseCandidate struct {
	version     string
	archiveName string
	archiveData []byte
	archiveHash string
	npmPackage  string
	binary      string
}

type packedNPMConsumer struct {
	dir    string
	script string
}

func TestPackedNPMReleaseCandidateContract(t *testing.T) {
	t.Parallel()
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" && runtime.GOOS != "windows" {
		t.Skipf("npm bootstrap does not support %s", runtime.GOOS)
	}
	if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		t.Skipf("npm bootstrap does not support %s", runtime.GOARCH)
	}
	for _, tool := range []string{"go", "node", "npm"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s is required for packed release acceptance: %v", tool, err)
		}
	}

	sourceRoot := findTestModuleRoot(t)
	version := readRepoVersion(sourceRoot)
	if version == "" {
		t.Fatal("source release version is missing")
	}
	if npmVersion := readNpmPackageVersion(sourceRoot); npmVersion != version {
		t.Fatalf("npm version %q does not match source version %q", npmVersion, version)
	}
	candidate := stageReleaseCandidate(t, sourceRoot, version)
	oldBinary := buildReleaseCandidateBinary(t, sourceRoot, t.TempDir(), "0.9.0")
	consumer := installPackedNPMCandidate(t, candidate.npmPackage)

	t.Run("packed npm installs the matching staged archive", func(t *testing.T) {
		t.Parallel()
		server := serveStagedRelease(t, candidate, candidate.archiveHash)
		home := filepath.Join(t.TempDir(), "home")
		dest := filepath.Join(t.TempDir(), "bin")
		result := runPackedBootstrapScript(t, consumer, home, server.URL, "--dest", dest, "--", "version")
		if result.ExitCode != 0 {
			t.Fatalf("packed bootstrap failed: exit=%d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
		}
		installed := filepath.Join(dest, releaseBinaryName())
		if got := runBinaryVersion(t, installed, home); got != version {
			t.Fatalf("installed binary version = %q, want %q", got, version)
		}
		if got := readVersionJSONFile(filepath.Join(home, ".aether", "version.json")); got != version {
			t.Fatalf("installed hub version = %q, want %q", got, version)
		}
		if _, err := os.Stat(filepath.Join(home, ".codex", "agents", "aether-builder.toml")); err != nil {
			t.Fatalf("packed bootstrap did not install platform assets: %v", err)
		}
	})

	t.Run("checksum failure preserves the previous binary", func(t *testing.T) {
		t.Parallel()
		server := serveStagedRelease(t, candidate, strings.Repeat("0", 64))
		home := filepath.Join(t.TempDir(), "home")
		dest := filepath.Join(t.TempDir(), "bin")
		installed := filepath.Join(dest, releaseBinaryName())
		mustCopyExecutable(t, oldBinary, installed)
		before := testFileSHA256(t, installed)

		result := runPackedBootstrapScript(t, consumer, home, server.URL, "--dest", dest, "--", "version")
		if result.ExitCode == 0 || !strings.Contains(result.Stderr, "Checksum mismatch") {
			t.Fatalf("bad checksum was not rejected: exit=%d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
		}
		if after := testFileSHA256(t, installed); after != before {
			t.Fatalf("checksum failure changed the installed binary: before=%s after=%s", before, after)
		}
		if got := runBinaryVersion(t, installed, home); got != "0.9.0" {
			t.Fatalf("previous binary version = %q after checksum failure, want 0.9.0", got)
		}
	})

	t.Run("wrong version archive preserves the previous binary", func(t *testing.T) {
		t.Parallel()
		wrong := stageReleaseArchive(t, version, oldBinary)
		wrong.npmPackage = candidate.npmPackage
		server := serveStagedRelease(t, wrong, wrong.archiveHash)
		home := filepath.Join(t.TempDir(), "home")
		dest := filepath.Join(t.TempDir(), "bin")
		installed := filepath.Join(dest, releaseBinaryName())
		mustCopyExecutable(t, oldBinary, installed)
		before := testFileSHA256(t, installed)

		result := runPackedBootstrapScript(t, consumer, home, server.URL, "--dest", dest, "--", "version")
		if result.ExitCode == 0 || !strings.Contains(result.Stderr, "Downloaded binary version mismatch") {
			t.Fatalf("wrong-version archive was not rejected: exit=%d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
		}
		if after := testFileSHA256(t, installed); after != before {
			t.Fatalf("wrong-version archive changed the installed binary: before=%s after=%s", before, after)
		}
	})

	t.Run("retry recovers the state left by interrupted activation", func(t *testing.T) {
		t.Parallel()
		server := serveStagedRelease(t, candidate, candidate.archiveHash)
		home := filepath.Join(t.TempDir(), "home")
		dest := filepath.Join(t.TempDir(), "bin")
		installed := filepath.Join(dest, releaseBinaryName())
		rollback := installed + ".previous"
		mustCopyExecutable(t, oldBinary, rollback)

		result := runPackedBootstrapScript(t, consumer, home, server.URL, "--dest", dest, "--", "version")
		if result.ExitCode != 0 {
			t.Fatalf("retry after interrupted activation failed: exit=%d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
		}
		if got := runBinaryVersion(t, installed, home); got != version {
			t.Fatalf("recovered install version = %q, want %q", got, version)
		}
		if _, err := os.Stat(rollback); !os.IsNotExist(err) {
			t.Fatalf("successful retry left rollback file behind: %v", err)
		}
	})

	t.Run("packed candidate migrates and rolls back an n-1 colony without local data loss", func(t *testing.T) {
		t.Parallel()
		server := serveStagedRelease(t, candidate, candidate.archiveHash)
		home := filepath.Join(t.TempDir(), "home")
		dest := filepath.Join(t.TempDir(), "bin")
		bootstrap := runPackedBootstrapScript(t, consumer, home, server.URL, "--dest", dest, "--", "version")
		if bootstrap.ExitCode != 0 {
			t.Fatalf("prepare packed migration candidate: exit=%d\nstdout:\n%s\nstderr:\n%s", bootstrap.ExitCode, bootstrap.Stdout, bootstrap.Stderr)
		}
		binary := filepath.Join(dest, releaseBinaryName())
		repo := filepath.Join(t.TempDir(), "n-1-colony")
		if err := os.MkdirAll(repo, 0755); err != nil {
			t.Fatalf("create n-1 colony: %v", err)
		}
		setup := runCandidateAether(t, binary, repo, home, "lay-eggs", "--repo-dir", repo, "--home-dir", home)
		if setup.ExitCode != 0 {
			t.Fatalf("lay eggs with packed candidate: exit=%d\nstdout:\n%s\nstderr:\n%s", setup.ExitCode, setup.Stdout, setup.Stderr)
		}

		queenPath := filepath.Join(repo, ".aether", "QUEEN.md")
		queenContent := []byte("# Project Queen\n\nPreserve the N-1 project decision.\n")
		if err := os.WriteFile(queenPath, queenContent, 0644); err != nil {
			t.Fatalf("write N-1 Queen: %v", err)
		}
		skillPath := filepath.Join(repo, ".aether", "skills", "n-1-custom", "SKILL.md")
		if err := os.MkdirAll(filepath.Dir(skillPath), 0755); err != nil {
			t.Fatalf("create N-1 skill directory: %v", err)
		}
		skillContent := []byte("---\nname: n-1-custom\ndescription: Preserve this project skill.\n---\n")
		if err := os.WriteFile(skillPath, skillContent, 0644); err != nil {
			t.Fatalf("write N-1 skill: %v", err)
		}
		legacyState := []byte(`{
  "version": "2.0",
  "goal": "Upgrade this N-1 colony",
  "state": "READY",
  "current_phase": 0,
  "plan": {"phases": []}
}`)
		statePath := filepath.Join(repo, ".aether", "data", "COLONY_STATE.json")
		if err := os.WriteFile(statePath, legacyState, 0644); err != nil {
			t.Fatalf("write N-1 state: %v", err)
		}

		update := runCandidateAether(t, binary, repo, home, "update", "--force")
		if update.ExitCode != 0 {
			t.Fatalf("update N-1 colony: exit=%d\nstdout:\n%s\nstderr:\n%s", update.ExitCode, update.Stdout, update.Stderr)
		}
		assertFileEquals(t, queenPath, queenContent, "project Queen after update")
		assertFileEquals(t, skillPath, skillContent, "project skill after update")
		assertFileEquals(t, statePath, legacyState, "N-1 state before migration")

		migration := runCandidateAether(t, binary, repo, home, "migrate-state")
		if migration.ExitCode != 0 {
			t.Fatalf("migrate N-1 colony: exit=%d\nstdout:\n%s\nstderr:\n%s", migration.ExitCode, migration.Stdout, migration.Stderr)
		}
		var migrationEnvelope struct {
			Result struct {
				BackupPath      string `json:"backup_path"`
				RollbackCommand string `json:"rollback_command"`
			} `json:"result"`
		}
		if err := json.Unmarshal(bytes.TrimSpace([]byte(migration.Stdout)), &migrationEnvelope); err != nil {
			t.Fatalf("parse packed migration output: %v\n%s", err, migration.Stdout)
		}
		backupPath := migrationEnvelope.Result.BackupPath
		if backupPath == "" || migrationEnvelope.Result.RollbackCommand != "aether migrate-state --rollback "+filepath.ToSlash(backupPath) {
			t.Fatalf("packed migration lacks portable rollback: %+v", migrationEnvelope.Result)
		}
		rollback := runCandidateAether(t, binary, repo, home, "migrate-state", "--rollback", backupPath)
		if rollback.ExitCode != 0 {
			t.Fatalf("rollback N-1 migration: exit=%d\nstdout:\n%s\nstderr:\n%s", rollback.ExitCode, rollback.Stdout, rollback.Stderr)
		}
		assertFileEquals(t, statePath, legacyState, "N-1 state after rollback")
		assertFileEquals(t, queenPath, queenContent, "project Queen after rollback")
		assertFileEquals(t, skillPath, skillContent, "project skill after rollback")
	})
}

func TestStagedGoReleaserArtifactsInstallThroughPackedNPM(t *testing.T) {
	distDir := strings.TrimSpace(os.Getenv("AETHER_RELEASE_ACCEPTANCE_DIR"))
	if distDir == "" {
		t.Skip("set AETHER_RELEASE_ACCEPTANCE_DIR to a completed GoReleaser snapshot directory")
	}
	if !filepath.IsAbs(distDir) {
		sourceRoot := findTestModuleRoot(t)
		distDir = filepath.Join(sourceRoot, distDir)
	}
	version := readRepoVersion(findTestModuleRoot(t))
	archiveName := fmt.Sprintf("Aether_%s_%s_%s.tar.gz", version, runtime.GOOS, runtime.GOARCH)
	archiveData, err := os.ReadFile(filepath.Join(distDir, archiveName))
	if err != nil {
		t.Fatalf("read GoReleaser archive %s: %v", archiveName, err)
	}
	checksums, err := os.ReadFile(filepath.Join(distDir, "checksums.txt"))
	if err != nil {
		t.Fatalf("read GoReleaser checksums: %v", err)
	}
	expectedHash := checksumForReleaseArtifact(t, checksums, archiveName)
	actualDigest := sha256.Sum256(archiveData)
	actualHash := hex.EncodeToString(actualDigest[:])
	if actualHash != expectedHash {
		t.Fatalf("GoReleaser checksum mismatch for %s: expected=%s actual=%s", archiveName, expectedHash, actualHash)
	}

	var metadata struct {
		Version string `json:"version"`
	}
	metadataData, err := os.ReadFile(filepath.Join(distDir, "metadata.json"))
	if err != nil {
		t.Fatalf("read GoReleaser metadata: %v", err)
	}
	if err := json.Unmarshal(metadataData, &metadata); err != nil {
		t.Fatalf("parse GoReleaser metadata: %v", err)
	}
	if metadata.Version != version {
		t.Fatalf("GoReleaser metadata version = %q, want %q", metadata.Version, version)
	}

	sourceRoot := findTestModuleRoot(t)
	candidate := stagedReleaseCandidate{
		version:     version,
		archiveName: archiveName,
		archiveData: archiveData,
		archiveHash: expectedHash,
		npmPackage:  packNPMCandidate(t, sourceRoot, t.TempDir()),
	}
	server := serveStagedRelease(t, candidate, expectedHash)
	home := filepath.Join(t.TempDir(), "home")
	dest := filepath.Join(t.TempDir(), "bin")
	result := runPackedBootstrap(t, candidate.npmPackage, home, server.URL, "--dest", dest, "--", "version")
	if result.ExitCode != 0 {
		t.Fatalf("actual GoReleaser archive failed packed npm installation: exit=%d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}
	if got := runBinaryVersion(t, filepath.Join(dest, releaseBinaryName()), home); got != version {
		t.Fatalf("actual GoReleaser binary version = %q, want %q", got, version)
	}
	if got := readVersionJSONFile(filepath.Join(home, ".aether", "version.json")); got != version {
		var hubListing []string
		_ = filepath.Walk(home, func(p string, info os.FileInfo, err error) error {
			if err == nil {
				hubListing = append(hubListing, p)
			}
			return nil
		})
		t.Fatalf("actual GoReleaser hub version = %q, want %q\nbootstrap stdout:\n%s\nbootstrap stderr:\n%s\nhome tree (%d entries):\n%s",
			got, version, result.Stdout, result.Stderr, len(hubListing), strings.Join(hubListing[:min(len(hubListing), 40)], "\n"))
	}
}

func stageReleaseCandidate(t *testing.T, sourceRoot, version string) stagedReleaseCandidate {
	t.Helper()
	root := t.TempDir()
	binary := buildReleaseCandidateBinary(t, sourceRoot, root, version)
	candidate := stageReleaseArchive(t, version, binary)
	candidate.binary = binary
	candidate.npmPackage = packNPMCandidate(t, sourceRoot, root)
	return candidate
}

func stageReleaseArchive(t *testing.T, version, binary string) stagedReleaseCandidate {
	t.Helper()
	archiveName := fmt.Sprintf("Aether_%s_%s_%s.tar.gz", version, runtime.GOOS, runtime.GOARCH)
	binaryData, err := os.ReadFile(binary)
	if err != nil {
		t.Fatalf("read staged binary: %v", err)
	}
	var archive bytes.Buffer
	gzipWriter := gzip.NewWriter(&archive)
	tarWriter := tar.NewWriter(gzipWriter)
	if err := tarWriter.WriteHeader(&tar.Header{
		Name: releaseBinaryName(),
		Mode: 0755,
		Size: int64(len(binaryData)),
	}); err != nil {
		t.Fatalf("write staged archive header: %v", err)
	}
	if _, err := tarWriter.Write(binaryData); err != nil {
		t.Fatalf("write staged archive binary: %v", err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close staged tar archive: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("close staged gzip archive: %v", err)
	}
	digest := sha256.Sum256(archive.Bytes())
	return stagedReleaseCandidate{
		version:     version,
		archiveName: archiveName,
		archiveData: append([]byte(nil), archive.Bytes()...),
		archiveHash: hex.EncodeToString(digest[:]),
		binary:      binary,
	}
}

func buildReleaseCandidateBinary(t *testing.T, sourceRoot, outputRoot, version string) string {
	t.Helper()
	if err := os.MkdirAll(outputRoot, 0755); err != nil {
		t.Fatalf("create release binary directory: %v", err)
	}
	output := filepath.Join(outputRoot, releaseBinaryName())
	ldflag := "-X github.com/calcosmic/Aether/cmd.Version=" + version
	command := exec.Command("go", "build", "-ldflags", ldflag, "-o", output, "./cmd/aether")
	command.Dir = sourceRoot
	if combined, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build staged release binary %s: %v\n%s", version, err, combined)
	}
	return output
}

func packNPMCandidate(t *testing.T, sourceRoot, outputRoot string) string {
	t.Helper()
	command := exec.Command("npm", "pack", "--silent", "--pack-destination", outputRoot)
	command.Dir = filepath.Join(sourceRoot, "npm")
	if combined, err := command.CombinedOutput(); err != nil {
		t.Fatalf("pack npm release candidate: %v\n%s", err, combined)
	}
	matches, err := filepath.Glob(filepath.Join(outputRoot, "aether-colony-*.tgz"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("find packed npm candidate: matches=%v err=%v", matches, err)
	}
	return matches[0]
}

func serveStagedRelease(t *testing.T, candidate stagedReleaseCandidate, checksum string) *httptest.Server {
	t.Helper()
	base := "/v" + candidate.version + "/"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case base + "checksums.txt":
			_, _ = fmt.Fprintf(w, "%s  %s\n", checksum, candidate.archiveName)
		case base + candidate.archiveName:
			w.Header().Set("Content-Type", "application/gzip")
			_, _ = w.Write(candidate.archiveData)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

// dropEnvKeys removes entries for the named keys from an environment slice.
//
// The blackbox helpers below simulate a CLEAN USER MACHINE inside a sandboxed
// HOME. The parent `go test` process, however, runs under TestMain's suite-wide
// hub isolation (testing_main_test.go sets AETHER_HUB_DIR to a temp hub so
// tests can never pollute the developer's real ~/.aether — the v1.25
// hive-pollution fix). Passing that variable through to spawned release
// binaries makes `aether install` write its hub to the SUITE's isolation dir
// instead of the simulated user's $HOME/.aether, and the acceptance assertions
// then read an empty hub. No real user machine has AETHER_HUB_DIR set, so the
// faithful simulation is to strip it.
func dropEnvKeys(env []string, keys ...string) []string {
	dropped := make(map[string]bool, len(keys))
	for _, k := range keys {
		dropped[k] = true
	}
	result := make([]string, 0, len(env))
	for _, entry := range env {
		if key, _, ok := strings.Cut(entry, "="); ok && dropped[key] {
			continue
		}
		result = append(result, entry)
	}
	return result
}

func runPackedBootstrap(t *testing.T, packagePath, home, releaseURL string, args ...string) cliBlackBoxResult {
	t.Helper()
	return runPackedBootstrapScript(t, installPackedNPMCandidate(t, packagePath), home, releaseURL, args...)
}

// installPackedNPMCandidate installs the immutable packed candidate once for
// callers that exercise several independent bootstrap outcomes. The node
// wrapper only reads this consumer tree; each invocation below still receives
// its own HOME and destination, where all runtime state is written.
func installPackedNPMCandidate(t *testing.T, packagePath string) packedNPMConsumer {
	t.Helper()
	consumer := filepath.Join(t.TempDir(), "consumer")
	if err := os.MkdirAll(consumer, 0755); err != nil {
		t.Fatalf("create npm consumer: %v", err)
	}
	// The empty consumer has no package boundary yet. An explicit prefix keeps
	// npm from discovering and changing a project above the test's temporary root.
	install := exec.Command("npm", "install", "--prefix", consumer, "--ignore-scripts", "--no-audit", "--no-fund", packagePath)
	install.Dir = consumer
	if combined, err := install.CombinedOutput(); err != nil {
		t.Fatalf("install packed npm candidate: %v\n%s", err, combined)
	}
	return packedNPMConsumer{
		dir:    consumer,
		script: filepath.Join(consumer, "node_modules", "aether-colony", "bin", "aether.js"),
	}
}

func runPackedBootstrapScript(t *testing.T, consumer packedNPMConsumer, home, releaseURL string, args ...string) cliBlackBoxResult {
	t.Helper()
	command := exec.Command("node", append([]string{consumer.script}, args...)...)
	command.Dir = consumer.dir
	command.Env = replaceProcessEnv(dropEnvKeys(os.Environ(), "AETHER_HUB_DIR"), map[string]string{
		"AETHER_OUTPUT_MODE":      "json",
		"AETHER_RELEASE_BASE_URL": releaseURL,
		"CODEX_HOME":              filepath.Join(home, ".codex"),
		"HOME":                    home,
		"NO_COLOR":                "1",
		"USERPROFILE":             home,
	})
	var stdoutBuffer, stderrBuffer bytes.Buffer
	command.Stdout = &stdoutBuffer
	command.Stderr = &stderrBuffer
	err := command.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("run packed npm bootstrap: %v", err)
		}
	}
	return cliBlackBoxResult{Stdout: stdoutBuffer.String(), Stderr: stderrBuffer.String(), ExitCode: exitCode}
}

func runBinaryVersion(t *testing.T, binary, home string) string {
	t.Helper()
	command := exec.Command(binary, "version")
	command.Env = replaceProcessEnv(dropEnvKeys(os.Environ(), "AETHER_HUB_DIR"), map[string]string{
		"AETHER_OUTPUT_MODE": "json",
		"HOME":               home,
		"NO_COLOR":           "1",
		"USERPROFILE":        home,
	})
	output, err := command.Output()
	if err != nil {
		t.Fatalf("run staged binary version: %v", err)
	}
	var envelope struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(output), &envelope); err != nil {
		t.Fatalf("parse staged binary version: %v\n%s", err, output)
	}
	return envelope.Result
}

func runCandidateAether(t *testing.T, binary, repo, home string, args ...string) cliBlackBoxResult {
	t.Helper()
	command := exec.Command(binary, args...)
	command.Dir = repo
	command.Env = replaceProcessEnv(dropEnvKeys(os.Environ(), "AETHER_HUB_DIR"), map[string]string{
		"AETHER_HIVE_POLICY": "off",
		"AETHER_OUTPUT_MODE": "json",
		"AETHER_ROOT":        repo,
		"CODEX_HOME":         filepath.Join(home, ".codex"),
		"COLONY_DATA_DIR":    filepath.Join(repo, ".aether", "data"),
		"HOME":               home,
		"NO_COLOR":           "1",
		"USERPROFILE":        home,
	})
	var stdoutBuffer, stderrBuffer bytes.Buffer
	command.Stdout = &stdoutBuffer
	command.Stderr = &stderrBuffer
	err := command.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("run staged Aether %s: %v", strings.Join(args, " "), err)
		}
	}
	return cliBlackBoxResult{Stdout: stdoutBuffer.String(), Stderr: stderrBuffer.String(), ExitCode: exitCode}
}

func assertFileEquals(t *testing.T, path string, expected []byte, label string) {
	t.Helper()
	actual, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", label, err)
	}
	if !bytes.Equal(actual, expected) {
		t.Fatalf("%s changed:\n%s", label, actual)
	}
}

func mustCopyExecutable(t *testing.T, source, destination string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		t.Fatalf("create executable destination: %v", err)
	}
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read executable source: %v", err)
	}
	if err := os.WriteFile(destination, data, 0755); err != nil {
		t.Fatalf("copy executable: %v", err)
	}
}

func testFileSHA256(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s for checksum: %v", path, err)
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func releaseBinaryName() string {
	if runtime.GOOS == "windows" {
		return "aether.exe"
	}
	return "aether"
}

func checksumForReleaseArtifact(t *testing.T, checksums []byte, artifact string) string {
	t.Helper()
	for _, line := range strings.Split(string(checksums), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == artifact {
			return fields[0]
		}
	}
	t.Fatalf("checksum for %s not found", artifact)
	return ""
}
