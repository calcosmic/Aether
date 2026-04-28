package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitResearchGo(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	projectRoot := filepath.Dir(filepath.Dir(s.BasePath()))
	os.WriteFile(filepath.Join(projectRoot, "go.mod"), []byte("module test\n"), 0644)

	rootCmd.SetArgs([]string{"init-research", "--goal", "build CLI", "--target", projectRoot})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["detected_type"] != "go" {
		t.Errorf("detected_type = %v, want go", result["detected_type"])
	}
	if result["goal"] != "build CLI" {
		t.Errorf("goal = %v, want 'build CLI'", result["goal"])
	}
	if result["is_git_repo"] != false {
		t.Errorf("is_git_repo = %v, want false", result["is_git_repo"])
	}
	if result["file_count"].(float64) < 1 {
		t.Errorf("file_count = %v, want >= 1", result["file_count"])
	}
}

func TestInitResearchNode(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	projectRoot := filepath.Dir(filepath.Dir(s.BasePath()))
	os.WriteFile(filepath.Join(projectRoot, "package.json"), []byte(`{"name":"test"}`), 0644)

	rootCmd.SetArgs([]string{"init-research", "--goal", "web app", "--target", projectRoot})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["detected_type"] != "node" {
		t.Errorf("detected_type = %v, want node", result["detected_type"])
	}
}

func TestInitResearchUnknown(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Use a clean empty temp dir as target
	emptyDir := t.TempDir()

	rootCmd.SetArgs([]string{"init-research", "--goal", "new project", "--target", emptyDir})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["detected_type"] != "unknown" {
		t.Errorf("detected_type = %v, want unknown", result["detected_type"])
	}
	langs := result["languages"].([]interface{})
	if len(langs) != 0 {
		t.Errorf("languages = %v, want empty", langs)
	}
	dirs := result["top_level_dirs"].([]interface{})
	if len(dirs) != 0 {
		t.Errorf("top_level_dirs = %v, want empty", dirs)
	}
}

func TestInitResearchCapturesGitAndDirs(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	projectRoot := filepath.Dir(filepath.Dir(s.BasePath()))
	if err := os.Mkdir(filepath.Join(projectRoot, ".git"), 0755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	if err := os.Mkdir(filepath.Join(projectRoot, "src"), 0755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "src", "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	rootCmd.SetArgs([]string{"init-research", "--goal", "ship feature", "--target", projectRoot})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["is_git_repo"] != true {
		t.Errorf("is_git_repo = %v, want true", result["is_git_repo"])
	}
	dirs := result["top_level_dirs"].([]interface{})
	if len(dirs) != 1 || dirs[0] != "src" {
		t.Errorf("top_level_dirs = %v, want [src]", dirs)
	}
}

func TestInitResearchMissingGoal(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stderr = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"init-research", "--goal", ""})

	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	if env["ok"] != false {
		t.Errorf("expected ok:false for missing goal, got: %v", env["ok"])
	}
}

// --- Task 1 & 2: Deep scan, governance, pheromone, charter tests ---

func TestInitResearchDeepScan(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Create a directory structure with subdirs and files
	target := t.TempDir()
	os.MkdirAll(filepath.Join(target, "src", "pkg"), 0755)
	os.WriteFile(filepath.Join(target, "src", "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(target, "src", "util.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(target, "src", "pkg", "lib.go"), []byte("package pkg"), 0644)
	os.WriteFile(filepath.Join(target, "go.mod"), []byte("module test\n"), 0644)

	rootCmd.SetArgs([]string{"init-research", "--goal", "deep scan test", "--target", target})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})

	// Verify complexity metrics
	complexity := result["complexity"].(map[string]interface{})
	if complexity["total_files"].(float64) < 3 {
		t.Errorf("total_files = %v, want >= 3", complexity["total_files"])
	}
	if complexity["total_dirs"].(float64) < 2 {
		t.Errorf("total_dirs = %v, want >= 2", complexity["total_dirs"])
	}
	largest := complexity["largest_files"].([]interface{})
	if len(largest) == 0 {
		t.Errorf("largest_files is empty, want at least 1 entry")
	}
}

func TestInitResearchReadmeSummary(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	target := t.TempDir()
	readmeContent := "# My Project\n\nThis is a test project with some description."
	os.WriteFile(filepath.Join(target, "README.md"), []byte(readmeContent), 0644)

	rootCmd.SetArgs([]string{"init-research", "--goal", "readme test", "--target", target})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})

	readmeSummary := result["readme_summary"].(string)
	if !strings.Contains(readmeSummary, "My Project") {
		t.Errorf("readme_summary = %q, want to contain 'My Project'", readmeSummary)
	}
}

func TestInitResearchGitHistory(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Create a temp git repo with one commit
	target := t.TempDir()
	runGit(t, target, "init")
	runGit(t, target, "config", "user.email", "test@test.com")
	runGit(t, target, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(target, "go.mod"), []byte("module test\n"), 0644)
	runGit(t, target, "add", ".")
	runGit(t, target, "commit", "-m", "initial")

	rootCmd.SetArgs([]string{"init-research", "--goal", "git history test", "--target", target})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})

	gitHistory := result["git_history"].(map[string]interface{})
	if gitHistory["commits"].(float64) < 1 {
		t.Errorf("git_history.commits = %v, want >= 1", gitHistory["commits"])
	}
	if gitHistory["branch"].(string) == "" {
		t.Errorf("git_history.branch = %q, want non-empty", gitHistory["branch"])
	}
}

func TestInitResearchGovernance(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	target := t.TempDir()
	// Create governance config files
	os.WriteFile(filepath.Join(target, ".eslintrc"), []byte("{}"), 0644)
	os.MkdirAll(filepath.Join(target, ".github", "workflows"), 0755)
	os.WriteFile(filepath.Join(target, ".github", "workflows", "ci.yml"), []byte("name: CI"), 0644)
	os.WriteFile(filepath.Join(target, "jest.config.js"), []byte("module.exports = {}"), 0644)

	rootCmd.SetArgs([]string{"init-research", "--goal", "governance test", "--target", target})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})

	governance := result["governance"].(map[string]interface{})
	linters := governance["linters"].([]interface{})
	foundESLint := false
	for _, l := range linters {
		if l == "ESLint" {
			foundESLint = true
		}
	}
	if !foundESLint {
		t.Errorf("linters = %v, want to contain ESLint", linters)
	}

	testFrameworks := governance["test_frameworks"].([]interface{})
	foundJest := false
	for _, tf := range testFrameworks {
		if tf == "Jest" {
			foundJest = true
		}
	}
	if !foundJest {
		t.Errorf("test_frameworks = %v, want to contain Jest", testFrameworks)
	}

	ciConfigs := governance["ci_configs"].([]interface{})
	foundGH := false
	for _, ci := range ciConfigs {
		if ci == "GitHub Actions" {
			foundGH = true
		}
	}
	if !foundGH {
		t.Errorf("ci_configs = %v, want to contain GitHub Actions", ciConfigs)
	}
}

func TestInitResearchPheromoneSuggestions(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	target := t.TempDir()
	// Create .env without .gitignore -- should trigger REDIRECT about secrets
	os.WriteFile(filepath.Join(target, ".env"), []byte("KEY=value\n"), 0644)
	// No .gitignore, no LICENSE, no README -- should trigger multiple suggestions

	rootCmd.SetArgs([]string{"init-research", "--goal", "pheromone test", "--target", target})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})

	suggestions := result["pheromone_suggestions"].([]interface{})
	if len(suggestions) == 0 {
		t.Fatal("pheromone_suggestions is empty, want at least 1")
	}

	// Verify at least one REDIRECT about secrets
	foundSecretRedirect := false
	for _, s := range suggestions {
		sug := s.(map[string]interface{})
		if sug["type"] == "REDIRECT" && strings.Contains(sug["content"].(string), "secrets") {
			foundSecretRedirect = true
		}
	}
	if !foundSecretRedirect {
		t.Errorf("pheromone_suggestions = %v, want REDIRECT about secrets", suggestions)
	}
}

func TestInitResearchCharter(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	target := t.TempDir()
	os.WriteFile(filepath.Join(target, "go.mod"), []byte("module test\n"), 0644)

	rootCmd.SetArgs([]string{"init-research", "--goal", "Build X", "--target", target})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})

	charter := result["charter"].(map[string]interface{})
	if charter["intent"] != "Build X" {
		t.Errorf("charter.intent = %v, want 'Build X'", charter["intent"])
	}
	if charter["vision"].(string) == "" {
		t.Errorf("charter.vision = %q, want non-empty", charter["vision"])
	}
	if charter["governance"].(string) == "" {
		t.Errorf("charter.governance = %q, want non-empty", charter["governance"])
	}
	if !strings.Contains(charter["goals"].(string), "Build X") {
		t.Errorf("charter.goals = %v, want to contain 'Build X'", charter["goals"])
	}
}

func TestInitResearchPriorColonies(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	target := t.TempDir()
	// Create a prior colony archive
	os.MkdirAll(filepath.Join(target, ".aether", "chambers", "colony1"), 0755)
	os.WriteFile(filepath.Join(target, ".aether", "chambers", "colony1", "COLONY_STATE.json"), []byte(`{}`), 0644)

	rootCmd.SetArgs([]string{"init-research", "--goal", "prior colonies test", "--target", target})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})

	priorColonies := result["prior_colonies"].(map[string]interface{})
	if priorColonies["count"].(float64) < 1 {
		t.Errorf("prior_colonies.count = %v, want >= 1", priorColonies["count"])
	}
	names := priorColonies["names"].([]interface{})
	foundColony := false
	for _, n := range names {
		if n == "colony1" {
			foundColony = true
		}
	}
	if !foundColony {
		t.Errorf("prior_colonies.names = %v, want to contain 'colony1'", names)
	}
}


// --- Task 1: Dependency parser tests ---

func TestParsePackageJsonDeps(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	target := t.TempDir()
	os.WriteFile(filepath.Join(target, "package.json"), []byte(`{
		"dependencies": {
			"express": "^4.18.0",
			"lodash": "4.17.21"
		},
		"devDependencies": {
			"jest": "^29.0.0"
		}
	}`), 0644)

	rootCmd.SetArgs([]string{"init-research", "--goal", "pkg json test", "--target", target})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})

	tsd := result["tech_stack_detail"].([]interface{})
	if len(tsd) != 1 {
		t.Fatalf("tech_stack_detail length = %d, want 1", len(tsd))
	}
	entry := tsd[0].(map[string]interface{})
	if entry["language"] != "node" {
		t.Errorf("language = %v, want node", entry["language"])
	}
	if entry["source_file"] != "package.json" {
		t.Errorf("source_file = %v, want package.json", entry["source_file"])
	}
	deps := entry["dependencies"].([]interface{})
	if len(deps) < 1 {
		t.Fatalf("dependencies empty, want at least 1")
	}
	foundExpress := false
	for _, d := range deps {
		dep := d.(map[string]interface{})
		if dep["name"] == "express" {
			foundExpress = true
		}
	}
	if !foundExpress {
		t.Errorf("dependencies = %v, want to contain express", deps)
	}
	devDeps := entry["dev_dependencies"].([]interface{})
	if len(devDeps) < 1 {
		t.Fatalf("dev_dependencies empty, want at least 1")
	}
	foundJest := false
	for _, d := range devDeps {
		dep := d.(map[string]interface{})
		if dep["name"] == "jest" {
			foundJest = true
		}
	}
	if !foundJest {
		t.Errorf("dev_dependencies = %v, want to contain jest", devDeps)
	}
}

func TestParseGoModDeps(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	target := t.TempDir()
	os.WriteFile(filepath.Join(target, "go.mod"), []byte("module example.com/myapp\n\ngo 1.21\n\nrequire (\n\tgithub.com/spf13/cobra v1.8.0\n\tgithub.com/tidwall/gjson v1.17.0 // indirect\n)\n\nrequire github.com/BurntSushi/toml v1.3.2\n"), 0644)

	rootCmd.SetArgs([]string{"init-research", "--goal", "go mod test", "--target", target})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})

	tsd := result["tech_stack_detail"].([]interface{})
	if len(tsd) != 1 {
		t.Fatalf("tech_stack_detail length = %d, want 1", len(tsd))
	}
	entry := tsd[0].(map[string]interface{})
	if entry["language"] != "go" {
		t.Errorf("language = %v, want go", entry["language"])
	}
	direct := entry["dependencies"].([]interface{})
	if len(direct) < 1 {
		t.Fatalf("dependencies empty, want at least 1 direct dep")
	}
	indirect := entry["indirect"].([]interface{})
	if len(indirect) < 1 {
		t.Fatalf("indirect empty, want at least 1 indirect dep")
	}
}

func TestParseCargoTomlDeps(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	target := t.TempDir()
	os.WriteFile(filepath.Join(target, "Cargo.toml"), []byte("[package]\nname = \"myapp\"\nversion = \"0.1.0\"\n\n[dependencies]\nserde = { version = \"1.0\", features = [\"derive\"] }\ntokio = \"1.35\"\nclap = \"4.4\"\n"), 0644)

	rootCmd.SetArgs([]string{"init-research", "--goal", "cargo test", "--target", target})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})

	tsd := result["tech_stack_detail"].([]interface{})
	if len(tsd) != 1 {
		t.Fatalf("tech_stack_detail length = %d, want 1", len(tsd))
	}
	entry := tsd[0].(map[string]interface{})
	if entry["language"] != "rust" {
		t.Errorf("language = %v, want rust", entry["language"])
	}
	deps := entry["dependencies"].([]interface{})
	if len(deps) < 3 {
		t.Errorf("dependencies count = %d, want at least 3", len(deps))
	}
}

func TestParsePyprojectDeps(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	target := t.TempDir()
	os.WriteFile(filepath.Join(target, "pyproject.toml"), []byte("[project]\nname = \"myapp\"\nversion = \"0.1.0\"\ndependencies = [\"requests>=2.0\", \"flask==2.1.0\", \"click\"]\n"), 0644)

	rootCmd.SetArgs([]string{"init-research", "--goal", "pyproject test", "--target", target})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})

	tsd := result["tech_stack_detail"].([]interface{})
	if len(tsd) != 1 {
		t.Fatalf("tech_stack_detail length = %d, want 1", len(tsd))
	}
	entry := tsd[0].(map[string]interface{})
	if entry["language"] != "python" {
		t.Errorf("language = %v, want python", entry["language"])
	}
	if entry["source_file"] != "pyproject.toml" {
		t.Errorf("source_file = %v, want pyproject.toml", entry["source_file"])
	}
	deps := entry["dependencies"].([]interface{})
	if len(deps) < 2 {
		t.Errorf("dependencies count = %d, want at least 2", len(deps))
	}
	foundRequests := false
	foundFlask := false
	for _, d := range deps {
		dep := d.(map[string]interface{})
		if dep["name"] == "requests" {
			foundRequests = true
		}
		if dep["name"] == "flask" {
			foundFlask = true
		}
	}
	if !foundRequests {
		t.Errorf("dependencies = %v, want to contain requests", deps)
	}
	if !foundFlask {
		t.Errorf("dependencies = %v, want to contain flask", deps)
	}
}

func TestParseComposerJsonDeps(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	target := t.TempDir()
	os.WriteFile(filepath.Join(target, "composer.json"), []byte("{\"require\":{\"laravel/framework\":\"^10.0\",\"php\":\"^8.1\"},\"require-dev\":{\"phpunit/phpunit\":\"^10.0\"}}\n"), 0644)

	rootCmd.SetArgs([]string{"init-research", "--goal", "composer test", "--target", target})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})

	tsd := result["tech_stack_detail"].([]interface{})
	if len(tsd) != 1 {
		t.Fatalf("tech_stack_detail length = %d, want 1", len(tsd))
	}
	entry := tsd[0].(map[string]interface{})
	if entry["language"] != "php" {
		t.Errorf("language = %v, want php", entry["language"])
	}
	if entry["source_file"] != "composer.json" {
		t.Errorf("source_file = %v, want composer.json", entry["source_file"])
	}
	deps := entry["dependencies"].([]interface{})
	foundLaravel := false
	for _, d := range deps {
		dep := d.(map[string]interface{})
		if dep["name"] == "laravel/framework" {
			foundLaravel = true
		}
	}
	if !foundLaravel {
		t.Errorf("dependencies = %v, want to contain laravel/framework", deps)
	}
	devDeps := entry["dev_dependencies"].([]interface{})
	foundPhpunit := false
	for _, d := range devDeps {
		dep := d.(map[string]interface{})
		if dep["name"] == "phpunit/phpunit" {
			foundPhpunit = true
		}
	}
	if !foundPhpunit {
		t.Errorf("dev_dependencies = %v, want to contain phpunit/phpunit", devDeps)
	}
}
