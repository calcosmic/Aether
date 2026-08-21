package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Agent definitions describe runtime behaviour, and nothing was checking that
// the behaviour they described existed.
//
// The Oracle agent told its reader for months that research ran as an
// in-session loop driven by a Stop hook, that a `--legacy` tmux mode was
// available as a fallback, and that findings belonged in a directory no Go
// code has ever read. None of it was true, and no test in the repo could have
// noticed: the doc-hygiene suite covers `commands/**` only, and the agent
// mirror test compares filenames rather than content.

func oracleAgentDocPaths() []string {
	return []string{
		filepath.Join(repoRootForDocTests(), ".claude", "agents", "ant", "aether-oracle.md"),
		filepath.Join(repoRootForDocTests(), ".opencode", "agents", "aether-oracle.md"),
		filepath.Join(repoRootForDocTests(), ".codex", "agents", "aether-oracle.toml"),
	}
}

// repoRootForDocTests resolves the repository root from the cmd package dir.
func repoRootForDocTests() string {
	return ".."
}

// TestOracleAgentDocsDescribeControllerOwnedLoop is the fast diagnostic: it
// names the specific claims that were wrong.
func TestOracleAgentDocsDescribeControllerOwnedLoop(t *testing.T) {
	required := []string{
		".aether/oracle/",
	}
	// Claims that were documented and false. A doc that says any of these is
	// describing a mechanism the runtime does not have.
	forbidden := []string{
		"Stop hook",
		"--legacy",
		"In-Session Loop",
		"in-session loop",
		"full conversation context",
		".aether/data/research/",
	}

	for _, path := range oracleAgentDocPaths() {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(data)
		for _, needle := range required {
			if !strings.Contains(text, needle) {
				t.Errorf("%s does not mention %q; it must describe the real workspace", filepath.Base(path), needle)
			}
		}
		for _, needle := range forbidden {
			if strings.Contains(text, needle) {
				t.Errorf("%s still describes %q, which the runtime does not have", filepath.Base(path), needle)
			}
		}
	}

	// The markdown definitions must name the function that actually owns the
	// loop, so the next reader can check the claim against the code.
	for _, path := range oracleAgentDocPaths()[:2] {
		data, _ := os.ReadFile(path)
		if !strings.Contains(string(data), "runOracleLoop") {
			t.Errorf("%s does not name runOracleLoop, so its description of the loop cannot be checked against the code", filepath.Base(path))
		}
	}
}

var aetherPathLiteralRe = regexp.MustCompile("`(\\.aether/[A-Za-z0-9._/{}-]+)`")

// TestAgentDocsNameOnlyRealRuntimePaths is the fence rather than the
// diagnostic. It catches the *next* invented path, not just this one: every
// `.aether/...` path an agent definition names must either exist in the repo or
// appear as a string literal in the Go source that would create it.
func TestAgentDocsNameOnlyRealRuntimePaths(t *testing.T) {
	root := repoRootForDocTests()

	goSources := &strings.Builder{}
	for _, dir := range []string{"cmd", "pkg"} {
		_ = filepath.Walk(filepath.Join(root, dir), func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr == nil {
				goSources.Write(data)
			}
			return nil
		})
	}
	sources := goSources.String()
	if len(sources) == 0 {
		t.Fatal("read no Go source; the fence would pass vacuously")
	}

	agentDirs := []string{
		filepath.Join(root, ".claude", "agents"),
		filepath.Join(root, ".opencode", "agents"),
	}

	for _, dir := range agentDirs {
		_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			for _, match := range aetherPathLiteralRe.FindAllStringSubmatch(string(data), -1) {
				named := match[1]
				// Placeholders stand in for a real path shape; check the
				// directory they live in instead of the literal.
				probe := named
				if idx := strings.Index(probe, "{"); idx >= 0 {
					probe = probe[:idx]
				}
				probe = strings.TrimSuffix(probe, "/")
				if probe == "" || probe == ".aether" {
					continue
				}
				if _, statErr := os.Stat(filepath.Join(root, probe)); statErr == nil {
					continue
				}
				// Not on disk: it must at least be a path the runtime builds.
				segments := strings.Split(strings.TrimPrefix(probe, ".aether/"), "/")
				found := false
				for _, segment := range segments {
					if segment != "" && strings.Contains(sources, `"`+segment+`"`) {
						found = true
						break
					}
				}
				if !found {
					rel, _ := filepath.Rel(root, path)
					t.Errorf("%s names %s, which does not exist and which no Go source builds — an agent told to write there writes into a void", rel, named)
				}
			}
			return nil
		})
	}
}

// TestRetiredOracleUtilsPromptIsAbsent mirrors the existing retired-mirror
// fence. The file was a hand-maintained duplicate of a prompt the runtime
// composes itself, so it could only ever drift.
func TestRetiredOracleUtilsPromptIsAbsent(t *testing.T) {
	retired := filepath.Join(repoRootForDocTests(), ".aether", "utils", "oracle", "oracle.md")
	if _, err := os.Stat(retired); !os.IsNotExist(err) {
		t.Errorf("%s is back; the worker prompt is composed in Go (buildOracleWorkerConfig), and a second copy can only drift", retired)
	}

	// The loop marker must not advertise it either.
	data, err := os.ReadFile(filepath.Join(repoRootForDocTests(), "cmd", "oracle_loop.go"))
	if err != nil {
		t.Fatalf("read oracle_loop.go: %v", err)
	}
	if strings.Contains(string(data), "oracle_md_path") {
		t.Error("cmd/oracle_loop.go still writes oracle_md_path into the loop marker, pointing at a deleted file")
	}
}

// TestOracleDocsOnlyAdvertiseRealFlags is invariant-style: it re-reads the
// documented flags rather than checking for the two that happened to be wrong.
func TestOracleDocsOnlyAdvertiseRealFlags(t *testing.T) {
	resetRootCmd(t)
	var oracle *struct{}
	_ = oracle

	flagRe := regexp.MustCompile(`--[a-z][a-z0-9-]*`)
	docs := []string{
		filepath.Join(repoRootForDocTests(), "README.md"),
		filepath.Join(repoRootForDocTests(), "docs", "phase3-section-commands.md"),
	}

	for _, doc := range docs {
		data, err := os.ReadFile(doc)
		if err != nil {
			t.Fatalf("read %s: %v", doc, err)
		}
		for _, line := range strings.Split(string(data), "\n") {
			if !strings.HasPrefix(strings.TrimSpace(line), "| `/ant-oracle`") {
				continue
			}
			for _, flag := range flagRe.FindAllString(line, -1) {
				name := strings.TrimPrefix(flag, "--")
				if oracleCmd.Flags().Lookup(name) == nil {
					t.Errorf("%s advertises `%s` on /ant-oracle, which is not a registered flag", filepath.Base(doc), flag)
				}
			}
		}
	}
}

// TestOracleYamlWrapperRoleMatchesRuntimeOwnership: the YAML claimed the
// TypeScript host may conduct the outer loop, which host.ts itself refuses to
// do and which the orphan allowlist records as unreachable.
func TestOracleYamlWrapperRoleMatchesRuntimeOwnership(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRootForDocTests(), ".aether", "commands", "oracle.yaml"))
	if err != nil {
		t.Fatalf("read oracle.yaml: %v", err)
	}
	text := string(data)
	if strings.Contains(text, "conduct the outer lifecycle loop") {
		t.Error("oracle.yaml still says the TypeScript host may conduct the Oracle loop; Go owns it end to end")
	}
	if !strings.Contains(text, "Go owns the entire Oracle loop") {
		t.Error("oracle.yaml does not state who owns the Oracle loop")
	}
}
