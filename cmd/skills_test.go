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

func TestParseSkillFrontmatterSource(t *testing.T) {
	input := `---
name: Custom Skill
description: User-created behavior
source: custom
type: domain
---

# Body content
`
	fm := parseSkillFrontmatter(input)
	if fm == nil {
		t.Fatal("expected non-nil frontmatter")
	}
	if fm.Source != "custom" {
		t.Errorf("Source = %q, want custom", fm.Source)
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
// skill-index-read command (build side is now direct buildFullIndex calls --
// skill-index, the CLI command that used to populate index.json, was deleted
// in Phase 191 as dead CLI surface; skill-parse-frontmatter was deleted the
// same way, its parsing logic covered directly by the parseSkillFrontmatter
// tests above)
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
		filepath.Join(tmpHub, "system", "skills", "colony", "build-discipline", "SKILL.md"),
	)
	return tmpHub
}

func TestSkillIndexBuildAndRead(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpHub := setupSkillTestHub(t)

	// Build the index directly. skill-index (the CLI command whose RunE used
	// to do exactly this) was deleted in Phase 191 as dead CLI surface --
	// buildFullIndex is the surviving logic, still load-bearing today via
	// matchSkillsForWorkflow's own loadSkillIndexOrBuild fallback. Persist it
	// the same way the deleted command's RunE did, so skill-index-read (which
	// survives -- it is not one of the 8 SKILL-01 commands) has something to
	// read.
	entries := buildFullIndex(tmpHub)
	if len(entries) != 2 {
		t.Fatalf("indexed = %d entries, want 2", len(entries))
	}
	indexPath := filepath.Join(tmpHub, "skills", "index.json")
	if err := os.MkdirAll(filepath.Dir(indexPath), 0755); err != nil {
		t.Fatalf("mkdir index dir: %v", err)
	}
	encoded, err := json.MarshalIndent(skillIndexData{Entries: entries}, "", "  ")
	if err != nil {
		t.Fatalf("marshal index: %v", err)
	}
	if err := os.WriteFile(indexPath, append(encoded, '\n'), 0644); err != nil {
		t.Fatalf("write index.json: %v", err)
	}

	// Read the index via the surviving skill-index-read command.
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

	readEntries := readResult["entries"].([]interface{})
	found := map[string]bool{}
	for _, e := range readEntries {
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
	t.Setenv("AETHER_HUB_DIR", tmpHub)

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
// repoMatchesFilePattern (used by skillWorkspaceMatchReasons, the live
// workspace-detection path called from matchSkillsForWorkflow). skill-detect,
// the CLI command that used to expose domain-only detection standalone, was
// deleted in Phase 191 as dead CLI surface -- its filtering loop had no
// separate function of its own to preserve, and skillWorkspaceMatchReasons
// itself remains covered through the match-path tests below.
// ---------------------------------------------------------------------------

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
	tmpHub := setupSkillTestHub(t)

	// matchSkillsForWorkflow is the live, in-process skill-injection function
	// (called directly from cmd/codex_build.go:3268's composeBuildManifestBrief).
	// Exercised directly since its CLI wrapper (skill-match) was deleted in
	// Phase 191 as dead CLI surface.
	result := matchSkillsForWorkflow(tmpHub, "", "builder", "")
	if result.Count < 1 {
		t.Errorf("count = %d, want >= 1 for role builder", result.Count)
	}
	if len(result.Matched) < 1 {
		t.Error("expected at least 1 matched skill name")
	}

	if len(result.ColonySkills) == 0 {
		t.Fatal("expected proof-bearing colony skill entries")
	}
	first := result.ColonySkills[0]
	if first.Name != "TDD Discipline" {
		t.Fatalf("first colony skill = %v, want TDD Discipline", first.Name)
	}
	if first.Path == "" {
		t.Fatal("expected skill path in match result")
	}
	if first.Source == "" {
		t.Fatal("expected skill source in match result")
	}
	if first.Score < 3 {
		t.Fatalf("score = %d, want >= 3 for role-matched builder skill", first.Score)
	}
	if len(first.Reasons) == 0 {
		t.Fatal("expected reasons in match result")
	}
	if first.Reasons[0].Code != "role_match" {
		t.Fatalf("first reason code = %v, want role_match", first.Reasons[0].Code)
	}
}

func TestSkillMatchWithTask(t *testing.T) {
	tmpHub := setupSkillTestHub(t)

	result := matchSkillsForWorkflow(tmpHub, "", "builder", "custom")
	if result.Count < 1 {
		t.Errorf("count = %d, want >= 1 for role builder + task custom", result.Count)
	}

	if len(result.DomainSkills) == 0 {
		t.Fatal("expected proof-bearing domain skill entries")
	}
	first := result.DomainSkills[0]
	foundTaskOverlap := false
	for _, reason := range first.Reasons {
		if reason.Code == "task_name_overlap" || reason.Code == "task_domain_overlap" {
			foundTaskOverlap = true
			break
		}
	}
	if !foundTaskOverlap {
		t.Fatal("expected task overlap proof in skill-match result")
	}
}

func TestSkillMatchTop3(t *testing.T) {
	tmpHub := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", tmpHub)

	// Create 5 colony skills that all match role "builder"
	for i := 0; i < 5; i++ {
		skillDir := filepath.Join(tmpHub, "system", "skills", "colony", "skill-"+string(rune('A'+i)))
		os.MkdirAll(skillDir, 0755)
		os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(
			"---\nname: Skill "+string(rune('A'+i))+"\ncategory: colony\nroles: builder\n---\nContent\n",
		), 0644)
	}

	// Match should return at most 3
	result := matchSkillsForWorkflow(tmpHub, "", "builder", "custom")
	if result.Count > 3 {
		t.Errorf("count = %d, want <= 3 (top-3 cap)", result.Count)
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

	skillDir := filepath.Join(tmpHub, "system", "skills", "colony", "continue-review")
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

	skillDir := filepath.Join(tmpHub, "system", "skills", "colony", "brownfield-analysis")
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

	skillDir := filepath.Join(tmpHub, "system", "skills", "colony", "legacy-build")
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

	skillDir := filepath.Join(tmpHub, "system", "skills", "colony", "live-build")
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
	tmpHub := setupSkillTestHub(t)

	// renderSkillInjectResult composes the same worker-brief skill section
	// cmd/codex_build.go:3268 injects into every build. Exercised directly
	// since its CLI wrapper (skill-inject) was deleted in Phase 191 as dead
	// CLI surface.
	match := matchSkillsForWorkflow(tmpHub, "", "builder", "fallback")
	result := renderSkillInjectResult(match)
	if result.SkillCount < 1 {
		t.Errorf("skill_count = %d, want >= 1", result.SkillCount)
	}
	if !strings.Contains(result.Section, "TDD Discipline") {
		t.Error("section missing TDD Discipline skill content")
	}
	if len(result.ColonySkills) == 0 {
		t.Fatal("expected injected result to preserve proof-bearing colony skills")
	}
}

func TestRenderSkillInjectResultEnforcesStandaloneBudget(t *testing.T) {
	tmpDir := t.TempDir()
	skillPath := filepath.Join(tmpDir, "SKILL.md")
	content := "---\nname: oversized\n---\n" + strings.Repeat("skill guidance ", 900)
	if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
		t.Fatalf("write skill: %v", err)
	}

	result := renderSkillInjectResult(skillMatchResult{
		Role: "builder",
		ColonySkills: []skillResolvedEntry{{
			skillIndexEntry: skillIndexEntry{Name: "oversized", Path: skillPath, Type: "colony"},
			Score:           100,
		}},
	})

	if len(result.Section) > skillInjectBudgetChars {
		t.Fatalf("skill section length = %d, want <= %d", len(result.Section), skillInjectBudgetChars)
	}
	// Phase 181 changed what "enforce the budget" means. It used to mean slicing
	// the section at a character count and appending a marker, which ended a live
	// worker's brief mid-sentence. A skill that does not fit is now dropped whole.
	if strings.Contains(result.Section, "[truncated]") {
		t.Fatalf("oversized skill was sliced rather than dropped:\n%s", result.Section)
	}
	if strings.Contains(result.Section, "skill guidance") {
		t.Fatalf("oversized skill should have been dropped entirely, got:\n%s", result.Section)
	}
}

func TestRenderSkillInjectResultUsesCompactBudget(t *testing.T) {
	tmpDir := t.TempDir()
	skillPath := filepath.Join(tmpDir, "SKILL.md")
	content := "---\nname: compact-oversized\n---\n" + strings.Repeat("compact skill guidance ", 500)
	if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
		t.Fatalf("write skill: %v", err)
	}

	result := renderSkillInjectResultWithBudget(skillMatchResult{
		Role: "builder",
		ColonySkills: []skillResolvedEntry{{
			skillIndexEntry: skillIndexEntry{Name: "compact-oversized", Path: skillPath, Type: "colony"},
			Score:           100,
		}},
	}, skillInjectCompactBudgetChars)

	if result.BudgetChars != skillInjectCompactBudgetChars {
		t.Fatalf("budget = %d, want %d", result.BudgetChars, skillInjectCompactBudgetChars)
	}
	if len(result.Section) > skillInjectCompactBudgetChars {
		t.Fatalf("skill section length = %d, want <= %d", len(result.Section), skillInjectCompactBudgetChars)
	}
	// See the standalone-budget test above: over-budget skills are dropped whole,
	// not sliced mid-sentence.
	if strings.Contains(result.Section, "[truncated]") {
		t.Fatalf("oversized skill was sliced rather than dropped:\n%s", result.Section)
	}
}

func TestResolveSkillInjectBudgetExplicitBeatsCompact(t *testing.T) {
	if got := resolveSkillInjectBudget(true, 2048); got != 2048 {
		t.Fatalf("explicit budget = %d, want 2048", got)
	}
	if got := resolveSkillInjectBudget(true, 0); got != skillInjectCompactBudgetChars {
		t.Fatalf("compact budget = %d, want %d", got, skillInjectCompactBudgetChars)
	}
	if got := resolveSkillInjectBudget(false, 0); got != skillInjectNormalBudgetChars {
		t.Fatalf("normal budget = %d, want %d", got, skillInjectNormalBudgetChars)
	}
}

func TestSkillInjectStatThenFallback(t *testing.T) {
	tmpHub := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", tmpHub)

	// Create a skill whose category path does NOT contain SKILL.md at the
	// standard hub/<category>/SKILL.md location, so injection falls back
	// to the entry's recorded Path field.
	skillDir := filepath.Join(tmpHub, "skills", "domain", "fallback-skill")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(
		"---\nname: Fallback Skill\ncategory: domain\nroles: builder\n---\nFallback content\n",
	), 0644)

	// The standard stat path (hub/skills/domain/SKILL.md) does NOT exist,
	// so renderSkillInjectResult should fall back to the entry's Path.
	match := matchSkillsForWorkflow(tmpHub, "", "builder", "fallback")
	result := renderSkillInjectResult(match)
	if result.SkillCount != 1 {
		t.Errorf("skill_count = %d, want 1 (fallback path should work)", result.SkillCount)
	}
	if !strings.Contains(result.Section, "Fallback content") {
		t.Error("section missing fallback skill content")
	}
}

func TestSkillInjectNoMatchingRole(t *testing.T) {
	tmpHub := setupSkillTestHub(t)

	// Inject with a role that doesn't match any skill
	match := matchSkillsForWorkflow(tmpHub, "", "nonexistent-role", "")
	result := renderSkillInjectResult(match)
	if result.SkillCount != 0 {
		t.Errorf("skill_count = %d, want 0 for nonexistent role", result.SkillCount)
	}
	if result.Section != "" {
		t.Errorf("section = %q, want empty string", result.Section)
	}
}

// ---------------------------------------------------------------------------
// skill-list, skill-cache-rebuild, and skill-diff commands were deleted in
// Phase 191 as dead CLI surface. skill-list and skill-cache-rebuild's shared
// computation (buildFullIndex) remains covered by the TestBuildFullIndex*
// family below; their disk-persistence side effect had no separate caller
// once both commands were gone. skill-diff's compare-and-report logic lived
// entirely inside its own RunE closure with no shared helper to preserve.
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// skill-is-user-created command
// ---------------------------------------------------------------------------

func TestSkillIsUserCreated(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpHub := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", tmpHub)

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
	t.Setenv("AETHER_HUB_DIR", tmpHub)

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
	t.Setenv("AETHER_HUB_DIR", tmpHub)

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
	t.Setenv("AETHER_HUB_DIR", tmpHub)

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
// buildFullIndex shared helper
//
// Two documentation-only tests previously lived here (index.json read-count
// redundancy across skill-index-read/skill-detect/skill-match/skill-inject,
// and triple directory-scan duplication across skill-index/skill-list/
// skill-cache-rebuild). Both premises depended on CLI commands deleted in
// Phase 191 as dead CLI surface; buildFullIndex's own computation remains
// covered directly by the tests below.
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

func TestBuildFullIndexPrefersHubShippedOverLegacyRepoMirror(t *testing.T) {
	tmpHub := t.TempDir()
	workDir := filepath.Join(tmpHub, "work")
	localSkillDir := filepath.Join(workDir, ".aether", "skills", "domain", "typescript")
	hubSkillDir := filepath.Join(tmpHub, "system", "skills", "domain", "typescript")
	if err := os.MkdirAll(localSkillDir, 0755); err != nil {
		t.Fatalf("mkdir local skill: %v", err)
	}
	if err := os.MkdirAll(hubSkillDir, 0755); err != nil {
		t.Fatalf("mkdir hub skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(localSkillDir, "SKILL.md"), []byte(
		"---\nname: TypeScript\ntype: domain\n---\nLegacy repo mirror\n",
	), 0644); err != nil {
		t.Fatalf("write local skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubSkillDir, "SKILL.md"), []byte(
		"---\nname: TypeScript\nsource: shipped\ntype: domain\n---\nCurrent hub shipped skill\n",
	), 0644); err != nil {
		t.Fatalf("write hub skill: %v", err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	entries := buildFullIndex(tmpHub)
	if len(entries) != 1 {
		t.Fatalf("expected 1 deduped entry, got %d: %+v", len(entries), entries)
	}
	if entries[0].Source != "hub-aether-shipped" {
		t.Fatalf("expected hub-shipped skill to win, got source %q", entries[0].Source)
	}
	if entries[0].DeclaredSource != "shipped" {
		t.Fatalf("expected declared source shipped, got %q", entries[0].DeclaredSource)
	}
}

func TestBuildFullIndexRepoCustomOverridesHubShipped(t *testing.T) {
	tmpHub := t.TempDir()
	workDir := filepath.Join(tmpHub, "work")
	localSkillDir := filepath.Join(workDir, ".aether", "skills", "domain", "typescript")
	hubSkillDir := filepath.Join(tmpHub, "system", "skills", "domain", "typescript")
	if err := os.MkdirAll(localSkillDir, 0755); err != nil {
		t.Fatalf("mkdir local skill: %v", err)
	}
	if err := os.MkdirAll(hubSkillDir, 0755); err != nil {
		t.Fatalf("mkdir hub skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(localSkillDir, "SKILL.md"), []byte(
		"---\nname: TypeScript\nsource: custom\ntype: domain\n---\nRepo custom override\n",
	), 0644); err != nil {
		t.Fatalf("write local skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubSkillDir, "SKILL.md"), []byte(
		"---\nname: TypeScript\nsource: shipped\ntype: domain\n---\nCurrent hub shipped skill\n",
	), 0644); err != nil {
		t.Fatalf("write hub skill: %v", err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	entries := buildFullIndex(tmpHub)
	if len(entries) != 1 {
		t.Fatalf("expected 1 deduped entry, got %d: %+v", len(entries), entries)
	}
	if entries[0].Source != "repo-aether" {
		t.Fatalf("expected repo custom skill to win, got source %q", entries[0].Source)
	}
	if !entries[0].IsUserCreated {
		t.Fatal("expected repo custom skill to be user-created")
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

// ---------------------------------------------------------------------------
// resolveHubPath call frequency documentation test
// ---------------------------------------------------------------------------

// TestResolveHubPathFrequency documents that resolveHubPath() is called
// repeatedly across this file (once per command/test that needs the hub
// path). With AETHER_HUB_DIR set, each call is just an env lookup, so this is
// cheap. Without the env var, each call does os.UserHomeDir() + filepath.Join().
func TestResolveHubPathReturnsConsistentValue(t *testing.T) {
	tmpHub := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", tmpHub)

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
	tmpHub := setupSkillTestHub(t)

	result := matchSkillsForWorkflow(tmpHub, "", "builder", "custom")

	if len(result.DomainSkills) == 0 {
		t.Fatal("expected proof-bearing domain skill entries")
	}
	first := result.DomainSkills[0]
	if first.Path == "" {
		t.Fatal("expected skill path in domain skill result")
	}
	if first.Score <= 0 {
		t.Fatalf("expected positive score in domain skill result, got %d", first.Score)
	}
	if len(first.Reasons) == 0 {
		t.Fatal("expected reasons in proof-bearing domain skill result")
	}
}

func TestSkillMatchTailwindFixtureIncludesEvidence(t *testing.T) {
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

	result := matchSkillsForWorkflow(resolveHubPath(), "", "builder", stack.Task)

	var tailwind *skillResolvedEntry
	for i := range result.DomainSkills {
		if result.DomainSkills[i].Name == "tailwind" {
			tailwind = &result.DomainSkills[i]
			break
		}
	}
	if tailwind == nil {
		t.Fatalf("expected tailwind skill in matched domain skills: %#v", result.DomainSkills)
	}
	if tailwind.Score < 4 {
		t.Fatalf("tailwind score = %d, want >= 4", tailwind.Score)
	}

	foundEvidence := make(map[string]bool, len(stack.ExpectedEvidence))
	for _, reason := range tailwind.Reasons {
		joined := strings.Join(reason.Evidence, ",")
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

	result := matchSkillsForWorkflow(resolveHubPath(), "", "builder", stack.Task)

	foundGo := false
	for _, skill := range result.DomainSkills {
		if stack.ForbiddenSkill != "" && skill.Name == stack.ForbiddenSkill {
			t.Fatalf("did not expect tailwind skill for go fixture: %#v", result.DomainSkills)
		}
		if skill.Name == stack.ExpectedSkill {
			foundGo = true
		}
	}
	if !foundGo {
		t.Fatalf("expected golang skill for go fixture: %#v", result.DomainSkills)
	}
}
