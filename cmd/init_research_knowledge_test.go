package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Field report .planning/field-reports/2026-08-16-init-obsidian-vault.md:
// init-research on a 538-file Obsidian vault returned ok:true with every
// discriminating field empty, then generated a charter reading "A unknown
// project" that warned a personal diary about "regression risk", plus five
// CI/LICENSE/README/formatter housekeeping pheromones with a 100% discard
// rate. The assistant judged the output unusable and abandoned the command —
// nothing errored, so the failure was invisible.

// buildVaultFixture creates a mini Obsidian vault: markdown notes, a
// .obsidian/ config dir, zero source files.
func buildVaultFixture(t *testing.T, root string, notes int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, ".obsidian"), 0755); err != nil {
		t.Fatalf("mkdir .obsidian: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".obsidian", "app.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("write app.json: %v", err)
	}
	notesDir := filepath.Join(root, "Daily Notes")
	if err := os.MkdirAll(notesDir, 0755); err != nil {
		t.Fatalf("mkdir notes: %v", err)
	}
	for i := 0; i < notes; i++ {
		path := filepath.Join(notesDir, fmt.Sprintf("2026-08-%02d.md", i+1))
		if err := os.WriteFile(path, []byte("# Diary entry\n\nSome notes.\n"), 0644); err != nil {
			t.Fatalf("write note: %v", err)
		}
	}
}

func runInitResearchOn(t *testing.T, root, goal string) map[string]interface{} {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s

	rootCmd.SetArgs([]string{"init-research", "--goal", goal, "--target", root})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init-research: %v", err)
	}
	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}
	return env["result"].(map[string]interface{})
}

func TestObsidianVaultClassifiesAsKnowledgeBase(t *testing.T) {
	root := t.TempDir()
	buildVaultFixture(t, root, 15)

	result := runInitResearchOn(t, root, "reorganise my diary vault")

	if result["detected_type"] != "knowledge_base" {
		t.Fatalf("detected_type = %v, want knowledge_base — the scan is still blind to notes vaults", result["detected_type"])
	}
	dirClass := result["dir_classification"].(map[string]interface{})
	if dirClass["type"] != "knowledge_base" {
		t.Errorf("dir_classification.type = %v, want knowledge_base", dirClass["type"])
	}
	signals, _ := json.Marshal(dirClass["signals"])
	if !strings.Contains(string(signals), "markdown") || !strings.Contains(string(signals), ".obsidian") {
		t.Errorf("signals should describe the vault by its markdown files and .obsidian config, got: %s", signals)
	}
}

// A markdown-heavy directory with no vault config still classifies, so plain
// docs archives and diaries get the same treatment as Obsidian vaults.
func TestMarkdownOnlyRepoClassifiesAsKnowledgeBase(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 12; i++ {
		if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("note-%d.md", i)), []byte("# Note\n"), 0644); err != nil {
			t.Fatalf("write note: %v", err)
		}
	}
	result := runInitResearchOn(t, root, "organise notes")
	if result["detected_type"] != "knowledge_base" {
		t.Errorf("detected_type = %v, want knowledge_base for an all-markdown directory", result["detected_type"])
	}
}

// TestCodeRepoNeverClassifiesAsKnowledgeBase is the safety fence: the class
// requires zero detected languages AND zero source files, so a documented code
// repo — even one that is mostly markdown by file count — keeps its real type.
func TestCodeRepoNeverClassifiesAsKnowledgeBase(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module test\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	docs := filepath.Join(root, "docs")
	if err := os.MkdirAll(docs, 0755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	for i := 0; i < 40; i++ {
		if err := os.WriteFile(filepath.Join(docs, fmt.Sprintf("doc-%d.md", i)), []byte("# Doc\n"), 0644); err != nil {
			t.Fatalf("write doc: %v", err)
		}
	}

	result := runInitResearchOn(t, root, "build the CLI")
	if result["detected_type"] == "knowledge_base" {
		t.Fatal("a Go repo with a large docs/ directory was classified as a knowledge base")
	}

	// Belt and braces at the unit level: even a vault marker cannot override
	// the presence of source code.
	if _, ok := classifyKnowledgeRepo([]string{"go"}, 100, 0, 100, []string{".obsidian"}); ok {
		t.Error("classifyKnowledgeRepo claimed a repo with a detected language")
	}
	if _, ok := classifyKnowledgeRepo(nil, 100, 1, 101, []string{".obsidian"}); ok {
		t.Error("classifyKnowledgeRepo claimed a repo containing a source file")
	}
}

func TestKnowledgeRepoGetsNoCodeHousekeepingPheromones(t *testing.T) {
	root := t.TempDir()
	buildVaultFixture(t, root, 15)
	// No README, no LICENSE, no CI — the exact conditions that produced five
	// irrelevant suggestions on the field-report vault.

	result := runInitResearchOn(t, root, "reorganise my diary vault")

	raw, _ := json.Marshal(result["pheromone_suggestions"])
	text := string(raw)
	for _, wrong := range []string{"CI/CD", "LICENSE", "README", "formatting", "no documentation"} {
		if strings.Contains(text, wrong) {
			t.Errorf("knowledge repo still receives code-housekeeping suggestion mentioning %q:\n%s", wrong, text)
		}
	}
	if !strings.Contains(text, "preserve") {
		t.Errorf("knowledge repo should get a content-preservation FOCUS, got:\n%s", text)
	}

	// The secrets guards must survive the suppression — a vault leaks
	// credentials as readily as a codebase.
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("TOKEN=x\n"), 0644); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	result = runInitResearchOn(t, root, "reorganise my diary vault")
	raw, _ = json.Marshal(result["pheromone_suggestions"])
	if !strings.Contains(string(raw), "secrets") {
		t.Errorf("secrets REDIRECT was suppressed along with the housekeeping:\n%s", raw)
	}
}

func TestKnowledgeRepoRisksAreAboutContent(t *testing.T) {
	root := t.TempDir()
	buildVaultFixture(t, root, 15)

	result := runInitResearchOn(t, root, "reorganise my diary vault")
	charter := result["charter"].(map[string]interface{})
	risks, _ := charter["key_risks"].(string)

	for _, wrong := range []string{"regression risk", "deployment risk", "linter"} {
		if strings.Contains(risks, wrong) {
			t.Errorf("a notes vault is warned about %q:\n%s", wrong, risks)
		}
	}
	for _, want := range []string{"Content loss", "Broken links"} {
		if !strings.Contains(risks, want) {
			t.Errorf("knowledge-repo risks missing %q:\n%s", want, risks)
		}
	}
	if vision, _ := charter["vision"].(string); !strings.Contains(vision, "knowledge base") || !strings.Contains(vision, "15") {
		t.Errorf("vision should describe a knowledge base by its note count, got: %q", vision)
	}
}

// TestUnknownProjectCharterNeverPrintsBrokenFiller pins the string defect and
// the filler pile-up: an all-empty scan produced vision "A unknown project"
// and three speculative tooling risks.
func TestUnknownProjectCharterNeverPrintsBrokenFiller(t *testing.T) {
	root := t.TempDir()
	// A handful of unrecognisable files: not enough markdown to be a
	// knowledge repo, nothing any detector matches.
	for i := 0; i < 3; i++ {
		if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("data-%d.bin", i)), []byte("x"), 0644); err != nil {
			t.Fatalf("write file: %v", err)
		}
	}

	result := runInitResearchOn(t, root, "do something with these files")
	if result["detected_type"] != "unknown" {
		t.Fatalf("fixture should scan as unknown, got %v", result["detected_type"])
	}
	charter := result["charter"].(map[string]interface{})

	vision, _ := charter["vision"].(string)
	if strings.Contains(vision, "A unknown") {
		t.Errorf("the ungrammatical filler is back: %q", vision)
	}
	if !strings.Contains(vision, "not determined") {
		t.Errorf("an unknown project's vision should say the scan could not determine it, got: %q", vision)
	}

	risks, _ := charter["key_risks"].(string)
	for _, filler := range []string{"regression risk", "deployment risk", "quality may drift"} {
		if strings.Contains(risks, filler) {
			t.Errorf("all-empty scan still emits speculative tooling risk %q:\n%s", filler, risks)
		}
	}
	if !strings.Contains(risks, "could not assess") {
		t.Errorf("all-empty scan should say risks could not be assessed, got:\n%s", risks)
	}
}

func TestAetherScaffoldWritesDurableStateMarker(t *testing.T) {
	localAether := t.TempDir()

	result := ensureRepoLocalScaffold(localAether)
	if len(result.errors) > 0 {
		t.Fatalf("scaffold errors: %v", result.errors)
	}

	data, err := os.ReadFile(filepath.Join(localAether, "WHAT-IS-THIS.md"))
	if err != nil {
		t.Fatalf(".aether/ has no durable-state marker; a disk cleanup cannot tell it from a build cache: %v", err)
	}
	text := string(data)
	for _, want := range []string{"not a build cache", "colony state", "ts-host/node_modules"} {
		if !strings.Contains(text, want) {
			t.Errorf("marker missing %q — it must say what the directory is and what alone is safe to delete:\n%s", want, text)
		}
	}

	// Idempotent: a second scaffold run never rewrites a user-touched marker.
	custom := []byte("user edited\n")
	if err := os.WriteFile(filepath.Join(localAether, "WHAT-IS-THIS.md"), custom, 0644); err != nil {
		t.Fatalf("edit marker: %v", err)
	}
	ensureRepoLocalScaffold(localAether)
	after, _ := os.ReadFile(filepath.Join(localAether, "WHAT-IS-THIS.md"))
	if string(after) != string(custom) {
		t.Error("scaffold overwrote an existing marker; it must write once and never again")
	}
}

// TestInitWrappersCarryLowSignalBranch: the wrapper spec must tell the
// assistant what to do when the scan finds nothing — inventing that branch is
// what abandoning the command looked like on the field-report vault.
func TestInitWrappersCarryLowSignalBranch(t *testing.T) {
	files := []string{
		filepath.Join("..", ".aether", "commands", "init.yaml"),
		filepath.Join("..", ".claude", "commands", "ant", "init.md"),
		filepath.Join("..", ".claude", "commands", "ant-init.md"),
		filepath.Join("..", ".opencode", "commands", "ant", "init.md"),
	}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		text := string(data)
		if !strings.Contains(text, "knowledge_base") {
			t.Errorf("%s does not tell the wrapper how to treat a notes vault", file)
		}
		if !strings.Contains(text, "nothing to go on") {
			t.Errorf("%s has no low-signal branch for a scan that found nothing", file)
		}
	}
}
