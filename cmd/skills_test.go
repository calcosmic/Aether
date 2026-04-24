package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// parseSkillFrontmatter
// ---------------------------------------------------------------------------

func TestParseSkillFrontmatter(t *testing.T) {
	input := `---
name: TDD Discipline
description: Write tests first
category: colony
detect: *.go, *_test.go
roles: builder, watcher
---

# Body content
Some content here.
`
	fm := parseSkillFrontmatter(input)
	if fm == nil {
		t.Fatal("expected non-nil frontmatter")
	}
	if fm.Name != "TDD Discipline" {
		t.Errorf("Name = %q, want %q", fm.Name, "TDD Discipline")
	}
	if fm.Description != "Write tests first" {
		t.Errorf("Description = %q, want %q", fm.Description, "Write tests first")
	}
	if fm.Category != "colony" {
		t.Errorf("Category = %q, want %q", fm.Category, "colony")
	}
	if len(fm.Detect) != 2 || fm.Detect[0] != "*.go" || fm.Detect[1] != "*_test.go" {
		t.Errorf("Detect = %v, want [*.go *_test.go]", fm.Detect)
	}
	if len(fm.Roles) != 2 || fm.Roles[0] != "builder" || fm.Roles[1] != "watcher" {
		t.Errorf("Roles = %v, want [builder watcher]", fm.Roles)
	}
}

func TestParseSkillFrontmatterWorkflowMetadata(t *testing.T) {
	input := `---
name: Workflow Skill
description: Use during build workflows
type: colony
agent_roles: [builder, watcher]
workflow_triggers:
  - build
task_keywords:
  - test
  - coverage
---

# Body content
Some content here.
`
	fm := parseSkillFrontmatter(input)
	if fm == nil {
		t.Fatal("expected non-nil frontmatter")
	}
	if len(fm.WorkflowTriggers) != 1 || fm.WorkflowTriggers[0] != "build" {
		t.Errorf("WorkflowTriggers = %v, want [build]", fm.WorkflowTriggers)
	}
	if len(fm.TaskKeywords) != 2 || fm.TaskKeywords[0] != "test" || fm.TaskKeywords[1] != "coverage" {
		t.Errorf("TaskKeywords = %v, want [test coverage]", fm.TaskKeywords)
	}
	if len(fm.AgentRoles) != 2 || fm.AgentRoles[0] != "builder" || fm.AgentRoles[1] != "watcher" {
		t.Errorf("AgentRoles = %v, want [builder watcher]", fm.AgentRoles)
	}
	if len(fm.Roles) != 2 || fm.Roles[0] != "builder" || fm.Roles[1] != "watcher" {
		t.Errorf("Roles = %v, want [builder watcher]", fm.Roles)
	}
}

func TestParseSkillFrontmatterNilOnEmpty(t *testing.T) {
	if parseSkillFrontmatter("no frontmatter here") != nil {
		t.Error("expected nil for content without frontmatter")
	}
	if parseSkillFrontmatter("") != nil {
		t.Error("expected nil for empty content")
	}
}

func TestParseSkillFrontmatterPartialFields(t *testing.T) {
	input := `---
name: Minimal Skill
category: domain
---
`
	fm := parseSkillFrontmatter(input)
	if fm == nil {
		t.Fatal("expected non-nil frontmatter")
	}
	if fm.Name != "Minimal Skill" {
		t.Errorf("Name = %q, want %q", fm.Name, "Minimal Skill")
	}
	if fm.Category != "domain" {
		t.Errorf("Category = %q, want %q", fm.Category, "domain")
	}
	if fm.Description != "" {
		t.Errorf("Description = %q, want empty", fm.Description)
	}
	if len(fm.Detect) != 0 {
		t.Errorf("Detect = %v, want empty", fm.Detect)
	}
	if len(fm.Roles) != 0 {
		t.Errorf("Roles = %v, want empty", fm.Roles)
	}
}

// ---------------------------------------------------------------------------
// findSkillDirs
// ---------------------------------------------------------------------------

func TestFindSkillDirs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a skill directory with SKILL.md
	colonyDir := filepath.Join(tmpDir, "colony", "tdd")
	os.MkdirAll(colonyDir, 0755)
	os.WriteFile(filepath.Join(colonyDir, "SKILL.md"), []byte("---\nname: TDD\n---"), 0644)

	// Create a non-skill directory (no SKILL.md)
	nonSkillDir := filepath.Join(tmpDir, "colony", "empty")
	os.MkdirAll(nonSkillDir, 0755)

	// Create a domain skill
	domainDir := filepath.Join(tmpDir, "domain", "go")
	os.MkdirAll(domainDir, 0755)
	os.WriteFile(filepath.Join(domainDir, "SKILL.md"), []byte("---\nname: Go\n---"), 0644)

	dirs := findSkillDirs(tmpDir)
	if len(dirs) != 2 {
		t.Errorf("expected 2 skill dirs, got %d: %v", len(dirs), dirs)
	}

	found := map[string]bool{}
	for _, d := range dirs {
		found[filepath.Base(d)] = true
	}
	if !found["tdd"] {
		t.Error("missing tdd skill dir")
	}
	if !found["go"] {
		t.Error("missing go skill dir")
	}
}

func TestFindSkillDirsEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	dirs := findSkillDirs(tmpDir)
	if len(dirs) != 0 {
		t.Errorf("expected 0 dirs for empty directory, got %d", len(dirs))
	}
}

func TestFindSkillDirsNonexistentDir(t *testing.T) {
	dirs := findSkillDirs("/nonexistent/path/that/should/not/exist")
	if len(dirs) != 0 {
		t.Errorf("expected 0 dirs for nonexistent path, got %d", len(dirs))
	}
}

func TestFindSkillDirsTopLevelSkill(t *testing.T) {
	tmpDir := t.TempDir()

	// SKILL.md directly in a subdirectory (no category nesting)
	skillDir := filepath.Join(tmpDir, "standalone")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: Standalone\n---"), 0644)

	dirs := findSkillDirs(tmpDir)
	if len(dirs) != 1 {
		t.Errorf("expected 1 skill dir, got %d", len(dirs))
	}
}

// ---------------------------------------------------------------------------
// indexSkillDir
// ---------------------------------------------------------------------------

func TestIndexSkillDir(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(tmpDir, 0755)
	os.WriteFile(filepath.Join(tmpDir, "SKILL.md"), []byte(
		"---\nname: Test Skill\ncategory: colony\ndetect: [\"*.go\"]\nroles: [builder]\nworkflow_triggers: [build]\ntask_keywords: [implement]\n---\nContent\n",
	), 0644)

	entry := indexSkillDir(tmpDir, false)
	if entry == nil {
		t.Fatal("expected non-nil entry")
	}
	if entry.Name != "Test Skill" {
		t.Errorf("Name = %q, want %q", entry.Name, "Test Skill")
	}
	if entry.Category != "colony" {
		t.Errorf("Category = %q, want %q", entry.Category, "colony")
	}
	if entry.IsUserCreated != false {
		t.Errorf("IsUserCreated = %v, want false", entry.IsUserCreated)
	}
	if len(entry.Detect) != 1 || entry.Detect[0] != "*.go" {
		t.Errorf("Detect = %v, want [*.go]", entry.Detect)
	}
	if len(entry.Roles) != 1 || entry.Roles[0] != "builder" {
		t.Errorf("Roles = %v, want [builder]", entry.Roles)
	}
	if len(entry.WorkflowTriggers) != 1 || entry.WorkflowTriggers[0] != "build" {
		t.Errorf("WorkflowTriggers = %v, want [build]", entry.WorkflowTriggers)
	}
	if len(entry.TaskKeywords) != 1 || entry.TaskKeywords[0] != "implement" {
		t.Errorf("TaskKeywords = %v, want [implement]", entry.TaskKeywords)
	}
}

func TestIndexSkillDirUserCreated(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(tmpDir, 0755)
	os.WriteFile(filepath.Join(tmpDir, "SKILL.md"), []byte(
		"---\nname: Custom Skill\ncategory: domain\n---\nContent\n",
	), 0644)

	entry := indexSkillDir(tmpDir, true)
	if entry == nil {
		t.Fatal("expected non-nil entry")
	}
	if entry.IsUserCreated != true {
		t.Errorf("IsUserCreated = %v, want true", entry.IsUserCreated)
	}
}

func TestIndexSkillDirNoFile(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(tmpDir, 0755)

	entry := indexSkillDir(tmpDir, false)
	if entry != nil {
		t.Error("expected nil entry for missing SKILL.md")
	}
}

func TestIndexSkillDirEmptyName(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(tmpDir, 0755)
	os.WriteFile(filepath.Join(tmpDir, "SKILL.md"), []byte(
		"---\ncategory: colony\n---\nContent\n",
	), 0644)

	entry := indexSkillDir(tmpDir, false)
	if entry != nil {
		t.Error("expected nil entry for skill with empty name")
	}
}

// ---------------------------------------------------------------------------
// skill-parse-frontmatter command
// ---------------------------------------------------------------------------

func TestSkillParseFrontmatter(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	skillFile := filepath.Join(tmpDir, "SKILL.md")
	os.WriteFile(skillFile, []byte(
		"---\nname: Parse Test\ndescription: A test skill\ncategory: colony\nroles: builder, scout\n---\nBody\n",
	), 0644)

	rootCmd.SetArgs([]string{"skill-parse-frontmatter", "--file", skillFile})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-parse-frontmatter failed: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["name"] != "Parse Test" {
		t.Errorf("name = %v, want %q", result["name"], "Parse Test")
	}
	if result["category"] != "colony" {
		t.Errorf("category = %v, want %q", result["category"], "colony")
	}
}

func TestSkillParseFrontmatterNoFile(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stderr = &buf

	rootCmd.SetArgs([]string{"skill-parse-frontmatter", "--file", "/nonexistent/file.md"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-parse-frontmatter failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"ok":false`) {
		t.Errorf("expected error envelope, got: %s", output)
	}
}

func TestSkillParseFrontmatterEmptyFile(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	rootCmd.SetArgs([]string{"skill-parse-frontmatter", "--file", ""})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-parse-frontmatter with empty file failed: %v", err)
	}
}

// ---------------------------------------------------------------------------
// skill-index command (build + read)
// ---------------------------------------------------------------------------

func setupSkillTestHub(t *testing.T) string {
	t.Helper()
	tmpHub := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", tmpHub)
	testHome := filepath.Join(tmpHub, "home")
	if err := os.MkdirAll(testHome, 0755); err != nil {
		t.Fatalf("failed to create test home: %v", err)
	}
	t.Setenv("HOME", testHome)
	t.Setenv("USERPROFILE", testHome)

	// Create local shipped skills in .aether/skills/ relative to a temp working dir.
	// skill-index reads from ".aether/skills" (relative to CWD), so we create
	// a temp dir, populate it, and chdir there.
	localSkills := filepath.Join(tmpHub, "local", ".aether", "skills", "colony", "tdd")
	os.MkdirAll(localSkills, 0755)
	os.WriteFile(filepath.Join(localSkills, "SKILL.md"), []byte(
		"---\nname: TDD Discipline\ncategory: colony\nroles: builder, watcher\ndetect: *_test.go\n---\nTDD content\n",
	), 0644)

	// Create user skills in hub
	userSkill := filepath.Join(tmpHub, "skills", "domain", "custom")
	os.MkdirAll(userSkill, 0755)
	os.WriteFile(filepath.Join(userSkill, "SKILL.md"), []byte(
		"---\nname: Custom Domain\ncategory: domain\nroles: builder\ndetect: *.custom\n---\nCustom content\n",
	), 0644)

	// Chdir to the local dir so ".aether/skills" resolves correctly
	origDir, _ := os.Getwd()
	os.Chdir(filepath.Join(tmpHub, "local"))
	t.Cleanup(func() { os.Chdir(origDir) })

	return tmpHub
}

func skillsTestRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve skills_test.go path")
	}
	return filepath.Dir(filepath.Dir(file))
}

func skillsFixturePath(t *testing.T, parts ...string) string {
	t.Helper()
	items := append([]string{skillsTestRepoRoot(t), "cmd", "testdata", "skill-fixtures"}, parts...)
	return filepath.Join(items...)
}

func copyFileForTest(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read %s: %v", src, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(dst), err)
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		t.Fatalf("write %s: %v", dst, err)
	}
}

func copyDirForTest(t *testing.T, src, dst string) {
	t.Helper()
	if err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	}); err != nil {
		t.Fatalf("copy dir %s -> %s: %v", src, dst, err)
	}
}

func skillMatchContainsName(entries []skillResolvedEntry, name string) bool {
	for _, entry := range entries {
		if entry.Name == name {
			return true
		}
	}
	return false
}

func setupProofSkillHub(t *testing.T) string {
	t.Helper()
	tmpHub := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", tmpHub)

	repoRoot := skillsTestRepoRoot(t)
	copyFileForTest(t,
		filepath.Join(repoRoot, ".aether", "skills", "domain", "tailwind", "SKILL.md"),
		filepath.Join(tmpHub, "skills", "domain", "tailwind", "SKILL.md"),
	)
	copyFileForTest(t,
		filepath.Join(repoRoot, ".aether", "skills", "domain", "golang", "SKILL.md"),
		filepath.Join(tmpHub, "skills", "domain", "golang", "SKILL.md"),
	)
	copyFileForTest(t,
		filepath.Join(repoRoot, ".aether", "skills", "colony", "build-discipline", "SKILL.md"),
		filepath.Join(tmpHub, "skills", "colony", "build-discipline", "SKILL.md"),
	)
	return tmpHub
}

func TestSkillIndexBuildAndRead(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpHub := setupSkillTestHub(t)

	// Build the index
	var buildBuf bytes.Buffer
	stdout = &buildBuf
	rootCmd.SetArgs([]string{"skill-index"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-index failed: %v", err)
	}

	buildEnv := parseEnvelope(t, buildBuf.String())
	buildResult := buildEnv["result"].(map[string]interface{})
	if buildResult["indexed"] != float64(2) {
		t.Errorf("indexed = %v, want 2", buildResult["indexed"])
	}

	// Verify index.json was written
	indexPath := filepath.Join(tmpHub, "skills", "index.json")
	if _, err := os.Stat(indexPath); err != nil {
		t.Fatalf("index.json not created at %s: %v", indexPath, err)
	}

	// Read the index
	var readBuf bytes.Buffer
	stdout = &readBuf
	rootCmd.SetArgs([]string{"skill-index-read"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-index-read failed: %v", err)
	}

	readEnv := parseEnvelope(t, readBuf.String())
	readResult := readEnv["result"].(map[string]interface{})
	if readResult["total"] != float64(2) {
		t.Errorf("total = %v, want 2", readResult["total"])
	}

	entries := readResult["entries"].([]interface{})
	found := map[string]bool{}
	for _, e := range entries {
		entry := e.(map[string]interface{})
		found[entry["name"].(string)] = true
	}
	if !found["TDD Discipline"] {
		t.Error("missing TDD Discipline in index entries")
	}
	if !found["Custom Domain"] {
		t.Error("missing Custom Domain in index entries")
	}
}

func TestSkillIndexReadEmpty(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpHub := t.TempDir()
	os.Setenv("AETHER_HUB_DIR", tmpHub)
	t.Cleanup(func() { os.Unsetenv("AETHER_HUB_DIR") })

	rootCmd.SetArgs([]string{"skill-index-read"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-index-read failed: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["total"] != float64(0) {
		t.Errorf("total = %v, want 0 for empty index", result["total"])
	}
}

// ---------------------------------------------------------------------------
// skill-detect command
// ---------------------------------------------------------------------------

func TestSkillDetect(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	setupSkillTestHub(t)

	// Build index first
	rootCmd.SetArgs([]string{"skill-index"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-index failed: %v", err)
	}

	// Create a file that matches the domain detect pattern in CWD
	// (setupSkillTestHub already chdir'd to tmpHub/local)
	os.WriteFile("some.custom", []byte("custom"), 0644)

	// Read and detect
	var detectBuf bytes.Buffer
	stdout = &detectBuf
	rootCmd.SetArgs([]string{"skill-detect"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-detect failed: %v", err)
	}

	env := parseEnvelope(t, detectBuf.String())
	result := env["result"].(map[string]interface{})
	total := int(result["total"].(float64))
	if total < 1 {
		t.Errorf("total = %d, want >= 1 (should match *.custom domain skill)", total)
	}

	matched := result["matched"].([]interface{})
	first := matched[0].(map[string]interface{})
	if first["score"] == nil || first["score"].(float64) <= 0 {
		t.Fatalf("expected positive score in skill-detect result, got %#v", first["score"])
	}
	reasons := first["reasons"].([]interface{})
	if len(reasons) == 0 {
		t.Fatal("expected proof-bearing reasons in skill-detect result")
	}
}

func TestRepoMatchesFilePattern_UsesWorkspaceSnapshotAndSkipsIgnoredDirs(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "node_modules", "left-pad"), 0755); err != nil {
		t.Fatalf("failed to create node_modules dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "pipeline.py"), []byte("print('ok')\n"), 0644); err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "node_modules", "left-pad", "index.ts"), []byte("export {};\n"), 0644); err != nil {
		t.Fatalf("failed to write ignored ts file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "config"), []byte("[core]\n"), 0644); err != nil {
		t.Fatalf("failed to write ignored git file: %v", err)
	}

	if !repoMatchesFilePattern(root, "*.py") {
		t.Fatal("expected *.py to match source file in workspace snapshot")
	}
	if !repoMatchesFilePattern(root, "src/*.py") {
		t.Fatal("expected relative pattern src/*.py to match source file")
	}
	if repoMatchesFilePattern(root, "*.ts") {
		t.Fatal("expected *.ts to stay false when only node_modules contains matching files")
	}
	if repoMatchesFilePattern(root, "config") {
		t.Fatal("expected ignored .git/config not to influence workspace pattern matches")
	}
}

func TestRepoMatchesFilePattern_UsesGitIndexSnapshot(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "node_modules", "left-pad"), 0755); err != nil {
		t.Fatalf("failed to create node_modules dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "pipeline.py"), []byte("print('ok')\n"), 0644); err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "node_modules", "left-pad", "index.ts"), []byte("export {};\n"), 0644); err != nil {
		t.Fatalf("failed to write ignored ts file: %v", err)
	}

	if out, err := exec.Command("git", "-C", root, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, string(out))
	}
	if out, err := exec.Command("git", "-C", root, "add", "src/pipeline.py").CombinedOutput(); err != nil {
		t.Fatalf("git add failed: %v\n%s", err, string(out))
	}

	if !repoMatchesFilePattern(root, "*.py") {
		t.Fatal("expected *.py to match tracked source file through git snapshot")
	}
	if repoMatchesFilePattern(root, "*.ts") {
		t.Fatal("expected *.ts to stay false when only ignored node_modules contains matching files")
	}
}

// ---------------------------------------------------------------------------
// skill-match command
// ---------------------------------------------------------------------------

func TestSkillMatchByRole(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	setupSkillTestHub(t)

	// Build index first
	rootCmd.SetArgs([]string{"skill-index"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-index failed: %v", err)
	}

	// Match by role
	var matchBuf bytes.Buffer
	stdout = &matchBuf
	rootCmd.SetArgs([]string{"skill-match", "--role", "builder"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-match failed: %v", err)
	}

	env := parseEnvelope(t, matchBuf.String())
	result := env["result"].(map[string]interface{})
	count := int(result["count"].(float64))
	if count < 1 {
		t.Errorf("count = %d, want >= 1 for role builder", count)
	}

	matched := result["matched"].([]interface{})
	if len(matched) < 1 {
		t.Error("expected at least 1 matched skill name")
	}
	for _, m := range matched {
		if _, ok := m.(string); !ok {
			t.Errorf("matched entry %v is not a string (name), got %T", m, m)
		}
	}

	colonySkills := result["colony_skills"].([]interface{})
	if len(colonySkills) == 0 {
		t.Fatal("expected proof-bearing colony skill entries")
	}
	first := colonySkills[0].(map[string]interface{})
	if first["name"] != "TDD Discipline" {
		t.Fatalf("first colony skill = %v, want TDD Discipline", first["name"])
	}
	if first["path"] == "" {
		t.Fatal("expected skill path in match result")
	}
	if first["source"] == "" {
		t.Fatal("expected skill source in match result")
	}
	if score := int(first["score"].(float64)); score < 3 {
		t.Fatalf("score = %d, want >= 3 for role-matched builder skill", score)
	}
	reasons := first["reasons"].([]interface{})
	if len(reasons) == 0 {
		t.Fatal("expected reasons in match result")
	}
	reason := reasons[0].(map[string]interface{})
	if reason["code"] != "role_match" {
		t.Fatalf("first reason code = %v, want role_match", reason["code"])
	}
}

func TestSkillMatchWithTask(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	setupSkillTestHub(t)

	// Build index
	rootCmd.SetArgs([]string{"skill-index"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-index failed: %v", err)
	}

	// Match by role + task
	var matchBuf bytes.Buffer
	stdout = &matchBuf
	rootCmd.SetArgs([]string{"skill-match", "--role", "builder", "--task", "custom"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-match failed: %v", err)
	}

	env := parseEnvelope(t, matchBuf.String())
	result := env["result"].(map[string]interface{})
	count := int(result["count"].(float64))
	if count < 1 {
		t.Errorf("count = %d, want >= 1 for role builder + task custom", count)
	}

	domainSkills := result["domain_skills"].([]interface{})
	first := domainSkills[0].(map[string]interface{})
	reasons := first["reasons"].([]interface{})
	foundTaskOverlap := false
	for _, raw := range reasons {
		reason := raw.(map[string]interface{})
		if reason["code"] == "task_name_overlap" || reason["code"] == "task_domain_overlap" {
			foundTaskOverlap = true
			break
		}
	}
	if !foundTaskOverlap {
		t.Fatal("expected task overlap proof in skill-match result")
	}
}

func TestSkillMatchEmptyRole(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"skill-match", "--role", ""})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-match with empty role failed: %v", err)
	}

	// Empty role should produce no output (mustGetString returns early)
	if buf.Len() != 0 {
		t.Errorf("expected no output for empty role, got: %s", buf.String())
	}
}

func TestSkillMatchTop3(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpHub := t.TempDir()
	os.Setenv("AETHER_HUB_DIR", tmpHub)
	t.Cleanup(func() { os.Unsetenv("AETHER_HUB_DIR") })

	// Create 5 colony skills that all match role "builder"
	for i := 0; i < 5; i++ {
		skillDir := filepath.Join(tmpHub, "skills", "colony", "skill-"+string(rune('A'+i)))
		os.MkdirAll(skillDir, 0755)
		os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(
			"---\nname: Skill "+string(rune('A'+i))+"\ncategory: colony\nroles: builder\n---\nContent\n",
		), 0644)
	}

	// Build index
	rootCmd.SetArgs([]string{"skill-index"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-index failed: %v", err)
	}

	// Match should return at most 3
	var matchBuf bytes.Buffer
	stdout = &matchBuf
	rootCmd.SetArgs([]string{"skill-match", "--role", "builder", "--task", "custom"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-match failed: %v", err)
	}

	env := parseEnvelope(t, matchBuf.String())
	result := env["result"].(map[string]interface{})
	count := int(result["count"].(float64))
	if count > 3 {
		t.Errorf("count = %d, want <= 3 (top-3 cap)", count)
	}
}

func TestWorkflowGatedColonySkillRequiresWorkflowAndTaskEvidence(t *testing.T) {
	tmpHub := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", tmpHub)
	workDir := filepath.Join(tmpHub, "repo")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("mkdir work dir: %v", err)
	}
	origDir, _ := os.Getwd()
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("chdir work dir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	skillDir := filepath.Join(tmpHub, "skills", "colony", "continue-review")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
name: continue-review
description: Review completed phase work
type: colony
agent_roles: [watcher]
workflow_triggers: [continue]
task_keywords: [review]
---
Review content
`), 0644); err != nil {
		t.Fatalf("write skill: %v", err)
	}

	missingWorkflow := resolveSkillMatchesForRootWithWorkflow(tmpHub, workDir, "", "watcher", "review completed changes")
	if skillMatchContainsName(missingWorkflow.ColonySkills, "continue-review") {
		t.Fatal("workflow-gated skill matched without workflow evidence")
	}

	missingKeyword := resolveSkillMatchesForRootWithWorkflow(tmpHub, workDir, "continue", "watcher", "implement completed changes")
	if skillMatchContainsName(missingKeyword.ColonySkills, "continue-review") {
		t.Fatal("workflow-gated skill matched without task keyword evidence")
	}

	wrongRole := resolveSkillMatchesForRootWithWorkflow(tmpHub, workDir, "continue", "builder", "review completed changes")
	if skillMatchContainsName(wrongRole.ColonySkills, "continue-review") {
		t.Fatal("role-gated skill matched a different worker role")
	}

	matched := resolveSkillMatchesForRootWithWorkflow(tmpHub, workDir, "continue", "watcher", "review completed changes")
	if !skillMatchContainsName(matched.ColonySkills, "continue-review") {
		t.Fatalf("expected continue-review to match with workflow and task evidence: %#v", matched.ColonySkills)
	}
	if matched.Workflow != "continue" {
		t.Fatalf("workflow = %q, want continue", matched.Workflow)
	}
}

func TestWorkflowOnlyColonySkillMatchesWorkflowEvidence(t *testing.T) {
	tmpHub := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", tmpHub)
	workDir := filepath.Join(tmpHub, "repo")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("mkdir work dir: %v", err)
	}
	origDir, _ := os.Getwd()
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("chdir work dir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	skillDir := filepath.Join(tmpHub, "skills", "colony", "brownfield-analysis")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
name: brownfield-analysis
description: Analyze an existing codebase during colonize
type: colony
agent_roles: [surveyor-nest]
workflow_triggers: [colonize]
---
Analysis content
`), 0644); err != nil {
		t.Fatalf("write skill: %v", err)
	}

	matched := resolveSkillMatchesForRootWithWorkflow(tmpHub, workDir, "colonize", "surveyor-nest", "map project structure")
	if !skillMatchContainsName(matched.ColonySkills, "brownfield-analysis") {
		t.Fatalf("expected workflow-only skill to match colonize workflow: %#v", matched.ColonySkills)
	}
}

func TestLegacyRoleOnlyColonySkillStillMatchesByRole(t *testing.T) {
	tmpHub := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", tmpHub)
	workDir := filepath.Join(tmpHub, "repo")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("mkdir work dir: %v", err)
	}
	origDir, _ := os.Getwd()
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("chdir work dir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	skillDir := filepath.Join(tmpHub, "skills", "colony", "legacy-build")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
name: legacy-build
description: Legacy role-only build skill
type: colony
roles: [builder]
---
Build content
`), 0644); err != nil {
		t.Fatalf("write skill: %v", err)
	}

	matched := resolveSkillMatchesForRootWithWorkflow(tmpHub, workDir, "", "builder", "ordinary implementation")
	if !skillMatchContainsName(matched.ColonySkills, "legacy-build") {
		t.Fatalf("expected legacy role-only skill to keep matching by role: %#v", matched.ColonySkills)
	}
}

func TestSkillMatchBuildsLiveCatalogWhenDiskIndexIsStale(t *testing.T) {
	tmpHub := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", tmpHub)
	workDir := filepath.Join(tmpHub, "repo")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("mkdir work dir: %v", err)
	}
	origDir, _ := os.Getwd()
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("chdir work dir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	indexPath := filepath.Join(tmpHub, "skills", "index.json")
	if err := os.MkdirAll(filepath.Dir(indexPath), 0755); err != nil {
		t.Fatalf("mkdir index dir: %v", err)
	}
	staleIndex := `{"entries":[{"name":"stale-only","type":"colony","category":"colony","roles":["builder"],"path":"/missing/SKILL.md"}],"updated_at":"2026-01-01T00:00:00Z"}`
	if err := os.WriteFile(indexPath, []byte(staleIndex), 0644); err != nil {
		t.Fatalf("write stale index: %v", err)
	}

	skillDir := filepath.Join(tmpHub, "skills", "colony", "live-build")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
name: live-build
description: Live catalog skill
type: colony
roles: [builder]
---
Live content
`), 0644); err != nil {
		t.Fatalf("write skill: %v", err)
	}

	matched := resolveSkillMatchesForRootWithWorkflow(tmpHub, workDir, "build", "builder", "ordinary implementation")
	if !skillMatchContainsName(matched.ColonySkills, "live-build") {
		t.Fatalf("expected live skill catalog to override stale disk index: %#v", matched.ColonySkills)
	}
	if skillMatchContainsName(matched.ColonySkills, "stale-only") {
		t.Fatalf("stale disk index entry leaked into match result: %#v", matched.ColonySkills)
	}
}

// ---------------------------------------------------------------------------
// skill-inject command
// ---------------------------------------------------------------------------

func TestSkillInject(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	setupSkillTestHub(t)

	// Build index
	rootCmd.SetArgs([]string{"skill-index"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-index failed: %v", err)
	}

	// Inject skills for builder role
	var injectBuf bytes.Buffer
	stdout = &injectBuf
	rootCmd.SetArgs([]string{"skill-inject", "--role", "builder", "--task", "fallback"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-inject failed: %v", err)
	}

	env := parseEnvelope(t, injectBuf.String())
	result := env["result"].(map[string]interface{})
	skillCount := int(result["skill_count"].(float64))
	if skillCount < 1 {
		t.Errorf("skill_count = %d, want >= 1", skillCount)
	}

	section, ok := result["section"].(string)
	if !ok {
		t.Fatal("section is not a string")
	}
	if !strings.Contains(section, "TDD Discipline") {
		t.Error("section missing TDD Discipline skill content")
	}

	colonySkills := result["colony_skills"].([]interface{})
	if len(colonySkills) == 0 {
		t.Fatal("expected injected result to preserve proof-bearing colony skills")
	}
}

func TestSkillInjectStatThenFallback(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpHub := t.TempDir()
	os.Setenv("AETHER_HUB_DIR", tmpHub)
	t.Cleanup(func() { os.Unsetenv("AETHER_HUB_DIR") })

	// Create a skill whose category path does NOT contain SKILL.md at the
	// standard hub/<category>/SKILL.md location, so skill-inject falls back
	// to the entry's Path field (the stat-then-fallback path at lines 280-283).
	skillDir := filepath.Join(tmpHub, "skills", "domain", "fallback-skill")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(
		"---\nname: Fallback Skill\ncategory: domain\nroles: builder\n---\nFallback content\n",
	), 0644)

	// Build index (this records the full path)
	rootCmd.SetArgs([]string{"skill-index"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-index failed: %v", err)
	}

	// The standard stat path (hub/skills/domain/SKILL.md) does NOT exist,
	// so skill-inject should fall back to the entry's Path.
	// Verify that the fallback path still works.
	var injectBuf bytes.Buffer
	stdout = &injectBuf
	rootCmd.SetArgs([]string{"skill-inject", "--role", "builder", "--task", "fallback"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-inject failed: %v", err)
	}

	env := parseEnvelope(t, injectBuf.String())
	result := env["result"].(map[string]interface{})
	skillCount := int(result["skill_count"].(float64))
	if skillCount != 1 {
		t.Errorf("skill_count = %d, want 1 (fallback path should work)", skillCount)
	}

	section := result["section"].(string)
	if !strings.Contains(section, "Fallback content") {
		t.Error("section missing fallback skill content")
	}
}

func TestSkillInjectEmptyRole(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"skill-inject", "--role", ""})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-inject with empty role failed: %v", err)
	}

	if buf.Len() != 0 {
		t.Errorf("expected no output for empty role, got: %s", buf.String())
	}
}

func TestSkillInjectNoMatchingRole(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	setupSkillTestHub(t)

	// Build index
	rootCmd.SetArgs([]string{"skill-index"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-index failed: %v", err)
	}

	// Inject with a role that doesn't match any skill
	var injectBuf bytes.Buffer
	stdout = &injectBuf
	rootCmd.SetArgs([]string{"skill-inject", "--role", "nonexistent-role"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-inject failed: %v", err)
	}

	env := parseEnvelope(t, injectBuf.String())
	result := env["result"].(map[string]interface{})
	if result["skill_count"] != float64(0) {
		t.Errorf("skill_count = %v, want 0 for nonexistent role", result["skill_count"])
	}
	if result["section"] != "" {
		t.Errorf("section = %q, want empty string", result["section"])
	}
}

// ---------------------------------------------------------------------------
// skill-list command
// ---------------------------------------------------------------------------

func TestSkillList(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	setupSkillTestHub(t)

	rootCmd.SetArgs([]string{"skill-list"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-list failed: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	total := int(result["total"].(float64))
	if total < 1 {
		t.Errorf("total = %d, want >= 1", total)
	}

	skills := result["skills"].([]interface{})
	if len(skills) < 1 {
		t.Error("expected at least 1 skill in list")
	}
}

// ---------------------------------------------------------------------------
// skill-cache-rebuild command
// ---------------------------------------------------------------------------

func TestSkillCacheRebuild(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpHub := setupSkillTestHub(t)

	rootCmd.SetArgs([]string{"skill-cache-rebuild"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-cache-rebuild failed: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["rebuilt"] != true {
		t.Errorf("rebuilt = %v, want true", result["rebuilt"])
	}
	if result["total"] != float64(2) {
		t.Errorf("total = %v, want 2", result["total"])
	}

	// Verify the file was written
	indexPath := filepath.Join(tmpHub, "skills", "index.json")
	if _, err := os.Stat(indexPath); err != nil {
		t.Fatalf("index.json not created: %v", err)
	}
}

// ---------------------------------------------------------------------------
// skill-diff command
// ---------------------------------------------------------------------------

func TestSkillDiff(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpHub := t.TempDir()
	os.Setenv("AETHER_HUB_DIR", tmpHub)
	t.Cleanup(func() { os.Unsetenv("AETHER_HUB_DIR") })

	// Create shipped skill relative to CWD (.aether/skills/domain/<name>/SKILL.md)
	workDir := filepath.Join(tmpHub, "local")
	shippedDir := filepath.Join(workDir, ".aether", "skills", "domain", "test-skill")
	os.MkdirAll(shippedDir, 0755)
	os.WriteFile(filepath.Join(shippedDir, "SKILL.md"), []byte("---\nname: Test\n---\nOriginal\n"), 0644)

	// Create user skill in hub
	userDir := filepath.Join(tmpHub, "skills", "domain", "test-skill")
	os.MkdirAll(userDir, 0755)
	os.WriteFile(filepath.Join(userDir, "SKILL.md"), []byte("---\nname: Test\n---\nModified\n"), 0644)

	// skill-diff uses ".aether/skills/..." relative to CWD
	origDir, _ := os.Getwd()
	os.Chdir(workDir)
	t.Cleanup(func() { os.Chdir(origDir) })

	rootCmd.SetArgs([]string{"skill-diff", "--skill", "test-skill"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-diff failed: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["user_exists"] != true {
		t.Errorf("user_exists = %v, want true", result["user_exists"])
	}
	if result["shipped_exists"] != true {
		t.Errorf("shipped_exists = %v, want true", result["shipped_exists"])
	}
	if result["identical"] != false {
		t.Errorf("identical = %v, want false (content differs)", result["identical"])
	}
}

func TestSkillDiffIdentical(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpHub := t.TempDir()
	os.Setenv("AETHER_HUB_DIR", tmpHub)
	t.Cleanup(func() { os.Unsetenv("AETHER_HUB_DIR") })

	content := "---\nname: Same\n---\nSame content\n"

	workDir := filepath.Join(tmpHub, "local")
	shippedDir := filepath.Join(workDir, ".aether", "skills", "domain", "same-skill")
	os.MkdirAll(shippedDir, 0755)
	os.WriteFile(filepath.Join(shippedDir, "SKILL.md"), []byte(content), 0644)

	userDir := filepath.Join(tmpHub, "skills", "domain", "same-skill")
	os.MkdirAll(userDir, 0755)
	os.WriteFile(filepath.Join(userDir, "SKILL.md"), []byte(content), 0644)

	// skill-diff uses ".aether/skills/..." relative to CWD
	origDir, _ := os.Getwd()
	os.Chdir(workDir)
	t.Cleanup(func() { os.Chdir(origDir) })

	rootCmd.SetArgs([]string{"skill-diff", "--skill", "same-skill"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-diff failed: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["identical"] != true {
		t.Errorf("identical = %v, want true", result["identical"])
	}
}

func TestSkillDiffNotFound(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var errBuf bytes.Buffer
	stderr = &errBuf

	tmpHub := t.TempDir()
	os.Setenv("AETHER_HUB_DIR", tmpHub)
	t.Cleanup(func() { os.Unsetenv("AETHER_HUB_DIR") })

	rootCmd.SetArgs([]string{"skill-diff", "--skill", "nonexistent"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-diff failed: %v", err)
	}

	output := errBuf.String()
	if !strings.Contains(output, `"ok":false`) {
		t.Errorf("expected error envelope for nonexistent skill, got: %s", output)
	}
}

// ---------------------------------------------------------------------------
// skill-is-user-created command
// ---------------------------------------------------------------------------

func TestSkillIsUserCreated(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpHub := t.TempDir()
	os.Setenv("AETHER_HUB_DIR", tmpHub)
	t.Cleanup(func() { os.Unsetenv("AETHER_HUB_DIR") })

	// Only in hub (user-created)
	userDir := filepath.Join(tmpHub, "skills", "domain", "user-only")
	os.MkdirAll(userDir, 0755)
	os.WriteFile(filepath.Join(userDir, "SKILL.md"), []byte("---\nname: User Only\n---\n"), 0644)

	rootCmd.SetArgs([]string{"skill-is-user-created", "--skill", "user-only"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-is-user-created failed: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["is_user_created"] != true {
		t.Errorf("is_user_created = %v, want true for hub-only skill", result["is_user_created"])
	}
	if result["in_hub"] != true {
		t.Errorf("in_hub = %v, want true", result["in_hub"])
	}
	if result["in_shipped"] != false {
		t.Errorf("in_shipped = %v, want false", result["in_shipped"])
	}
}

func TestSkillIsUserCreatedShipped(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpHub := t.TempDir()
	os.Setenv("AETHER_HUB_DIR", tmpHub)
	t.Cleanup(func() { os.Unsetenv("AETHER_HUB_DIR") })

	// Only in shipped (relative to CWD)
	workDir := filepath.Join(tmpHub, "local")
	shippedDir := filepath.Join(workDir, ".aether", "skills", "domain", "shipped-only")
	os.MkdirAll(shippedDir, 0755)
	os.WriteFile(filepath.Join(shippedDir, "SKILL.md"), []byte("---\nname: Shipped Only\n---\n"), 0644)

	// skill-is-user-created uses ".aether/skills/..." relative to CWD
	origDir, _ := os.Getwd()
	os.Chdir(workDir)
	t.Cleanup(func() { os.Chdir(origDir) })

	rootCmd.SetArgs([]string{"skill-is-user-created", "--skill", "shipped-only"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-is-user-created failed: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["is_user_created"] != false {
		t.Errorf("is_user_created = %v, want false for shipped-only skill", result["is_user_created"])
	}
}

// ---------------------------------------------------------------------------
// skill-manifest-read command
// ---------------------------------------------------------------------------

func TestSkillManifestReadFromHub(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpHub := t.TempDir()
	os.Setenv("AETHER_HUB_DIR", tmpHub)
	t.Cleanup(func() { os.Unsetenv("AETHER_HUB_DIR") })

	os.MkdirAll(filepath.Join(tmpHub, "skills"), 0755)
	manifest := `{"skills":[{"name":"tdd","version":"1.0.0","checksum":"abc123"}],"updated_at":"2026-01-01T00:00:00Z"}`
	os.WriteFile(filepath.Join(tmpHub, "skills", "manifest.json"), []byte(manifest), 0644)

	rootCmd.SetArgs([]string{"skill-manifest-read"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-manifest-read failed: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["total"] != float64(1) {
		t.Errorf("total = %v, want 1", result["total"])
	}
}

func TestSkillManifestReadEmpty(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpHub := t.TempDir()
	os.Setenv("AETHER_HUB_DIR", tmpHub)
	t.Cleanup(func() { os.Unsetenv("AETHER_HUB_DIR") })

	rootCmd.SetArgs([]string{"skill-manifest-read"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-manifest-read failed: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["total"] != float64(0) {
		t.Errorf("total = %v, want 0 for no manifest", result["total"])
	}
}

// ---------------------------------------------------------------------------
// Redundancy documentation test: verify index.json read count
// ---------------------------------------------------------------------------

// TestSkillIndexReadPattern documents the current redundancy:
// skill-index-read, skill-detect, skill-match, and skill-inject each
// independently read and unmarshal index.json. This test verifies that
// the file exists after building, confirming the shared dependency.
func TestSkillIndexSharedDependency(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpHub := setupSkillTestHub(t)

	// Build the index
	rootCmd.SetArgs([]string{"skill-index"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-index failed: %v", err)
	}

	indexPath := filepath.Join(tmpHub, "skills", "index.json")
	raw, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("cannot read index.json: %v", err)
	}

	var data skillIndexData
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatalf("cannot unmarshal index.json: %v", err)
	}

	if len(data.Entries) != 2 {
		t.Errorf("expected 2 index entries, got %d", len(data.Entries))
	}

	// Document: these 4 commands all read index.json independently:
	//   skill-index-read (line 132)
	//   skill-detect       (line 160)
	//   skill-match        (line 196)
	//   skill-inject       (line 262)
	// Recommendation: share a single loadSkillIndex() helper.
	_ = data // used to confirm the data is valid for all 4 consumers
}

// ---------------------------------------------------------------------------
// Duplicated directory scanning documentation test
// ---------------------------------------------------------------------------

// TestDuplicatedScanningPattern documents the triple duplication:
// skill-index (lines 85-119), skill-list (lines 308-332), and
// skill-cache-rebuild (lines 374-409) all contain the same pattern of
// findSkillDirs(".aether/skills") + findSkillDirs(hub+"/skills").
func TestDuplicatedScanningPattern(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	setupSkillTestHub(t)

	// Each of these 3 commands scans the same directories independently.
	// skill-index writes the result; skill-list and skill-cache-rebuild
	// do the same scanning but don't cache it.

	// Run skill-index (builds and writes)
	var indexBuf bytes.Buffer
	stdout = &indexBuf
	rootCmd.SetArgs([]string{"skill-index"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-index failed: %v", err)
	}

	// Run skill-list (re-scans without using the cached index)
	var listBuf bytes.Buffer
	stdout = &listBuf
	rootCmd.SetArgs([]string{"skill-list"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-list failed: %v", err)
	}

	listEnv := parseEnvelope(t, listBuf.String())
	listResult := listEnv["result"].(map[string]interface{})
	listTotal := int(listResult["total"].(float64))

	// Run skill-cache-rebuild (re-scans again)
	var rebuildBuf bytes.Buffer
	stdout = &rebuildBuf
	rootCmd.SetArgs([]string{"skill-cache-rebuild"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-cache-rebuild failed: %v", err)
	}

	rebuildEnv := parseEnvelope(t, rebuildBuf.String())
	rebuildResult := rebuildEnv["result"].(map[string]interface{})
	rebuildTotal := int(rebuildResult["total"].(float64))

	// All three should produce the same count since they scan the same dirs
	if listTotal != rebuildTotal {
		t.Errorf("list total %d != rebuild total %d (should be identical)", listTotal, rebuildTotal)
	}

	// Recommendation: skill-list should read the cached index instead of
	// re-scanning. skill-cache-rebuild can keep scanning since its purpose
	// is to rebuild.
}

// ---------------------------------------------------------------------------
// buildFullIndex shared helper
// ---------------------------------------------------------------------------

func TestBuildFullIndex(t *testing.T) {
	tmpHub := t.TempDir()

	// Create local shipped skills relative to CWD
	localSkillsDir := filepath.Join(tmpHub, "local", ".aether", "skills", "colony", "tdd")
	os.MkdirAll(localSkillsDir, 0755)
	os.WriteFile(filepath.Join(localSkillsDir, "SKILL.md"), []byte(
		"---\nname: TDD Discipline\ncategory: colony\nroles: builder\ndetect: *_test.go\n---\nContent\n",
	), 0644)

	// Create hub user skills
	userSkillDir := filepath.Join(tmpHub, "skills", "domain", "custom")
	os.MkdirAll(userSkillDir, 0755)
	os.WriteFile(filepath.Join(userSkillDir, "SKILL.md"), []byte(
		"---\nname: Custom Skill\ncategory: domain\nroles: scout\n---\nContent\n",
	), 0644)

	// Chdir to local dir so ".aether/skills" resolves
	origDir, _ := os.Getwd()
	os.Chdir(filepath.Join(tmpHub, "local"))
	defer os.Chdir(origDir)

	entries := buildFullIndex(tmpHub)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d: %+v", len(entries), entries)
	}

	names := map[string]bool{}
	for _, e := range entries {
		names[e.Name] = true
	}
	if !names["TDD Discipline"] {
		t.Error("missing TDD Discipline entry")
	}
	if !names["Custom Skill"] {
		t.Error("missing Custom Skill entry")
	}

	// Verify IsUserCreated flag is set correctly
	for _, e := range entries {
		if e.Name == "TDD Discipline" && e.IsUserCreated {
			t.Error("TDD Discipline should not be user-created")
		}
		if e.Name == "Custom Skill" && !e.IsUserCreated {
			t.Error("Custom Skill should be user-created")
		}
	}
}

func TestBuildFullIndexEmpty(t *testing.T) {
	tmpHub := t.TempDir()

	// Create empty local dir with no skills
	localDir := filepath.Join(tmpHub, "local")
	os.MkdirAll(filepath.Join(localDir, ".aether", "skills"), 0755)

	origDir, _ := os.Getwd()
	os.Chdir(localDir)
	defer os.Chdir(origDir)

	entries := buildFullIndex(tmpHub)
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for empty dirs, got %d", len(entries))
	}
}

func TestBuildFullIndexMatchesCommandOutput(t *testing.T) {
	// Verify buildFullIndex produces the same results as the three commands
	// that previously had duplicated scanning logic.
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpHub := setupSkillTestHub(t)

	// Build index using command
	rootCmd.SetArgs([]string{"skill-index"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-index failed: %v", err)
	}

	// Read the index that skill-index wrote
	indexPath := filepath.Join(tmpHub, "skills", "index.json")
	raw, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("cannot read index: %v", err)
	}
	var cmdData skillIndexData
	json.Unmarshal(raw, &cmdData)

	// Build index using shared function
	entries := buildFullIndex(tmpHub)

	if len(entries) != len(cmdData.Entries) {
		t.Errorf("buildFullIndex returned %d entries, skill-index wrote %d",
			len(entries), len(cmdData.Entries))
	}

	cmdNames := map[string]bool{}
	for _, e := range cmdData.Entries {
		cmdNames[e.Name] = true
	}
	for _, e := range entries {
		if !cmdNames[e.Name] {
			t.Errorf("buildFullIndex entry %q not found in skill-index output", e.Name)
		}
	}
}

// ---------------------------------------------------------------------------
// resolveHubPath call frequency documentation test
// ---------------------------------------------------------------------------

// TestResolveHubPathFrequency documents that resolveHubPath() is called
// 10 times across the file (once per command that needs the hub path).
// With AETHER_HUB_DIR set, each call is just an env lookup, so this is
// cheap. Without the env var, each call does os.UserHomeDir() + filepath.Join().
func TestResolveHubPathReturnsConsistentValue(t *testing.T) {
	tmpHub := t.TempDir()
	os.Setenv("AETHER_HUB_DIR", tmpHub)
	defer os.Unsetenv("AETHER_HUB_DIR")

	// Call resolveHubPath multiple times and verify consistency
	results := make([]string, 10)
	for i := 0; i < 10; i++ {
		results[i] = resolveHubPath()
	}

	for i, r := range results {
		if r != tmpHub {
			t.Errorf("resolveHubPath() call %d = %q, want %q", i, r, tmpHub)
		}
	}

	// All calls should return the same value
	for i := 1; i < len(results); i++ {
		if results[i] != results[0] {
			t.Errorf("resolveHubPath() inconsistent: call 0 = %q, call %d = %q",
				results[0], i, results[i])
		}
	}
}

// ---------------------------------------------------------------------------
// proof-bearing skill resolution regression tests
// ---------------------------------------------------------------------------

func TestSkillMatchIncludesProofFields(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	setupSkillTestHub(t)

	// Build index (discard output)
	var indexBuf bytes.Buffer
	stdout = &indexBuf
	rootCmd.SetArgs([]string{"skill-index"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-index failed: %v", err)
	}

	// Match by role (capture this output)
	var matchBuf bytes.Buffer
	stdout = &matchBuf
	rootCmd.SetArgs([]string{"skill-match", "--role", "builder", "--task", "custom"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-match failed: %v", err)
	}

	env := parseEnvelope(t, matchBuf.String())
	result := env["result"].(map[string]interface{})
	matched := result["matched"].([]interface{})

	for i, m := range matched {
		if _, ok := m.(string); !ok {
			t.Errorf("matched[%d] = %v (%T), want string skill name", i, m, m)
		}
	}

	domainSkills := result["domain_skills"].([]interface{})
	if len(domainSkills) == 0 {
		t.Fatal("expected proof-bearing domain skill entries")
	}
	first := domainSkills[0].(map[string]interface{})
	if first["path"] == "" {
		t.Fatal("expected skill path in domain skill result")
	}
	if first["score"] == nil || first["score"].(float64) <= 0 {
		t.Fatalf("expected positive score in domain skill result, got %#v", first["score"])
	}
	reasons := first["reasons"].([]interface{})
	if len(reasons) == 0 {
		t.Fatal("expected reasons in proof-bearing domain skill result")
	}
}

func TestSkillMatchTailwindFixtureIncludesEvidence(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	setupProofSkillHub(t)
	store = nil

	fixtures := loadCompetitiveProofFixtures(t)
	var stack competitiveProofStackCase
	for _, candidate := range fixtures.StackCases {
		if candidate.ExpectedSkill == "tailwind" {
			stack = candidate
			break
		}
	}
	if stack.Name == "" {
		t.Fatal("missing tailwind competitive proof fixture")
	}

	fixture := filepath.Join(t.TempDir(), stack.Name)
	copyDirForTest(t, filepath.Join("testdata", stack.Workspace), fixture)
	origDir, _ := os.Getwd()
	if err := os.Chdir(fixture); err != nil {
		t.Fatalf("chdir fixture: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	var indexBuf bytes.Buffer
	stdout = &indexBuf
	rootCmd.SetArgs([]string{"skill-index"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-index failed: %v", err)
	}

	var matchBuf bytes.Buffer
	stdout = &matchBuf
	rootCmd.SetArgs([]string{"skill-match", "--role", "builder", "--task", stack.Task})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-match failed: %v", err)
	}

	env := parseEnvelope(t, matchBuf.String())
	result := env["result"].(map[string]interface{})
	domainSkills := result["domain_skills"].([]interface{})

	var tailwind map[string]interface{}
	for _, raw := range domainSkills {
		skill := raw.(map[string]interface{})
		if skill["name"] == "tailwind" {
			tailwind = skill
			break
		}
	}
	if tailwind == nil {
		t.Fatalf("expected tailwind skill in matched domain skills: %#v", domainSkills)
	}
	if int(tailwind["score"].(float64)) < 4 {
		t.Fatalf("tailwind score = %v, want >= 4", tailwind["score"])
	}

	reasons := tailwind["reasons"].([]interface{})
	foundEvidence := make(map[string]bool, len(stack.ExpectedEvidence))
	for _, raw := range reasons {
		reason := raw.(map[string]interface{})
		evidence := reason["evidence"].([]interface{})
		values := make([]string, 0, len(evidence))
		for _, item := range evidence {
			values = append(values, item.(string))
		}
		joined := strings.Join(values, ",")
		for _, want := range stack.ExpectedEvidence {
			if strings.Contains(joined, want) {
				foundEvidence[want] = true
			}
		}
	}
	for _, want := range stack.ExpectedEvidence {
		if !foundEvidence[want] {
			t.Fatalf("expected tailwind evidence %q in proof-bearing reasons", want)
		}
	}
}

func TestSkillMatchGoFixtureAvoidsTailwind(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	setupProofSkillHub(t)
	store = nil

	fixtures := loadCompetitiveProofFixtures(t)
	var stack competitiveProofStackCase
	for _, candidate := range fixtures.StackCases {
		if candidate.ExpectedSkill == "golang" {
			stack = candidate
			break
		}
	}
	if stack.Name == "" {
		t.Fatal("missing go competitive proof fixture")
	}

	fixture := filepath.Join(t.TempDir(), stack.Name)
	copyDirForTest(t, filepath.Join("testdata", stack.Workspace), fixture)
	origDir, _ := os.Getwd()
	if err := os.Chdir(fixture); err != nil {
		t.Fatalf("chdir fixture: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	var indexBuf bytes.Buffer
	stdout = &indexBuf
	rootCmd.SetArgs([]string{"skill-index"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-index failed: %v", err)
	}

	var matchBuf bytes.Buffer
	stdout = &matchBuf
	rootCmd.SetArgs([]string{"skill-match", "--role", "builder", "--task", stack.Task})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("skill-match failed: %v", err)
	}

	env := parseEnvelope(t, matchBuf.String())
	result := env["result"].(map[string]interface{})
	domainSkills := result["domain_skills"].([]interface{})

	foundGo := false
	for _, raw := range domainSkills {
		skill := raw.(map[string]interface{})
		if stack.ForbiddenSkill != "" && skill["name"] == stack.ForbiddenSkill {
			t.Fatalf("did not expect tailwind skill for go fixture: %#v", domainSkills)
		}
		if skill["name"] == stack.ExpectedSkill {
			foundGo = true
		}
	}
	if !foundGo {
		t.Fatalf("expected golang skill for go fixture: %#v", domainSkills)
	}
}
