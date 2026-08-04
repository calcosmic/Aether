package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// currentVersionDeclRe matches the lines that declare the CURRENT version, as
// opposed to release-history entries (which must keep naming their own
// version). Each doc carries these in a fixed shape.
var currentVersionDecls = []*regexp.Regexp{
	regexp.MustCompile(`(?m)^> \*\*Current Version:\*\* v(\d+\.\d+\.\d+)`),
	regexp.MustCompile(`(?m)^\| Version \| v(\d+\.\d+\.\d+)`),
	regexp.MustCompile(`(?m)^\*Updated for Aether v(\d+\.\d+\.\d+)`),
	regexp.MustCompile(`badge/colony-v(\d+\.\d+\.\d+)`),
	regexp.MustCompile("`aether-colony@(\\d+\\.\\d+\\.\\d+)`"),
}

// TestDocumentedVersionsMatchVersionFile is the WP-5 gate. Six documents each
// hand-maintained a version string and drifted to three different values while
// the runtime shipped a fourth — the first thing a user reads was wrong.
// `make version-sync` rewrites them; this fails whenever they drift again.
func TestDocumentedVersionsMatchVersionFile(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(repoRoot, ".aether", "version.json"))
	if err != nil {
		t.Fatalf("read version.json: %v", err)
	}
	var versionFile struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(raw, &versionFile); err != nil {
		t.Fatalf("parse version.json: %v", err)
	}
	want := strings.TrimSpace(versionFile.Version)
	if want == "" {
		t.Fatal("version.json declares no version")
	}

	docs := []string{"README.md", "CLAUDE.md", "AGENTS.md",
		filepath.Join(".codex", "CODEX.md"), filepath.Join(".opencode", "OPENCODE.md")}

	checked := 0
	for _, doc := range docs {
		path := filepath.Join(repoRoot, doc)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("read %s: %v", doc, err)
			continue
		}
		text := string(data)
		for _, re := range currentVersionDecls {
			for _, m := range re.FindAllStringSubmatch(text, -1) {
				checked++
				if m[1] != want {
					t.Errorf("%s declares current version %s, but .aether/version.json says %s — run `make version-sync`", doc, m[1], want)
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("no current-version declarations found — this guard would pass vacuously")
	}
	t.Logf("verified %d current-version declarations against v%s", checked, want)
}

// TestReadmeInstallCommandsAreValid pins the install instructions to something
// that actually works: the repository root is package aetherassets, not main,
// so `go install github.com/calcosmic/Aether@latest` always failed.
func TestReadmeInstallCommandsAreValid(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(repoRoot, "README.md"))
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	text := string(data)

	// The module root has no main package, so this form can never install.
	if strings.Contains(text, "go install github.com/calcosmic/Aether@") {
		t.Error("README recommends `go install github.com/calcosmic/Aether@...`; the module root is not a main package. Use .../cmd/aether@latest")
	}
	if !strings.Contains(text, "go install github.com/calcosmic/Aether/cmd/aether@") {
		t.Error("README no longer documents a working `go install` path")
	}

	// The documented Go requirement must not undershoot go.mod.
	modData, err := os.ReadFile(filepath.Join(repoRoot, "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	modVersion := regexp.MustCompile(`(?m)^go (\d+)\.(\d+)`).FindStringSubmatch(string(modData))
	if modVersion == nil {
		t.Fatal("go.mod declares no go directive")
	}
	wantPrefix := "Go " + modVersion[1] + "." + modVersion[2]
	if !strings.Contains(text, wantPrefix) {
		t.Errorf("README does not state the real toolchain requirement %q (go.mod: go %s.%s)", wantPrefix, modVersion[1], modVersion[2])
	}
}
