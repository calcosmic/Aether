package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// ---------------------------------------------------------------------------
// TestCountFilesInDir
// ---------------------------------------------------------------------------

func TestCountFilesInDir(t *testing.T) {
	dir := t.TempDir()

	// Create some files
	for i := 0; i < 5; i++ {
		f, err := os.Create(filepath.Join(dir, fmt.Sprintf("file%d.yaml", i)))
		if err != nil {
			t.Fatalf("create file: %v", err)
		}
		f.Close()
	}

	pattern := filepath.Join(dir, "*.yaml")
	count := countFilesInDir(pattern)
	if count != 5 {
		t.Errorf("expected 5 files, got %d", count)
	}

	// Non-matching pattern
	zeroCount := countFilesInDir(filepath.Join(dir, "*.md"))
	if zeroCount != 0 {
		t.Errorf("expected 0 for non-matching pattern, got %d", zeroCount)
	}
}

func markAetherSourceCheckout(t *testing.T, dir string) {
	t.Helper()
	writeFile(t, dir, "go.mod", []byte("module github.com/calcosmic/Aether\n"))
	writeFile(t, dir, filepath.Join("cmd", "aether", "main.go"), []byte("package main\n"))
}

// ---------------------------------------------------------------------------
// TestScanWrapperParityHealthy
// ---------------------------------------------------------------------------

func TestScanWrapperParityHealthy(t *testing.T) {
	dir := t.TempDir()
	markAetherSourceCheckout(t, dir)
	aetherDir := filepath.Join(dir, ".aether")
	claudeDir := filepath.Join(dir, ".claude")
	opencodeDir := filepath.Join(dir, ".opencode")
	codexDir := filepath.Join(dir, ".codex")

	// Create directories
	dirs := []string{
		filepath.Join(aetherDir, "commands"),
		filepath.Join(claudeDir, "commands", "ant"),
		filepath.Join(opencodeDir, "commands", "ant"),
		filepath.Join(codexDir, "agents"),
		filepath.Join(claudeDir, "agents", "ant"),
		filepath.Join(opencodeDir, "agents"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}

	// One YAML definition and one wrapper per platform for every menu command
	// this program ships, named the way the source tree names them.
	for name := range wrapperCommandNames {
		writeFile(t, aetherDir, "commands/"+name+".yaml", []byte("test"))
		writeFile(t, claudeDir, "commands/ant/"+name+".md", []byte("test"))
		writeFile(t, opencodeDir, "commands/ant/"+name+".md", []byte("test"))
	}
	// Create expected number of Codex agents (25)
	for i := 0; i < expectedCodexAgents; i++ {
		writeFile(t, codexDir, fmt.Sprintf("agents/agent%d.toml", i), []byte("test"))
	}
	// Create expected number of Claude agents (25)
	for i := 0; i < expectedClaudeAgents; i++ {
		writeFile(t, claudeDir, fmt.Sprintf("agents/ant/agent%d.md", i), []byte("test"))
	}
	// OpenCode ships the same castes plus one infrastructure-only worker router.
	for i := 0; i < expectedOpenCodeAgents-1; i++ {
		writeFile(t, opencodeDir, fmt.Sprintf("agents/agent%d.md", i), []byte("test"))
	}
	writeFile(t, opencodeDir, "agents/aether-worker-router.md", []byte("test"))
	// Create expected number of colony skills
	for i := 0; i < expectedColonySkills; i++ {
		name := fmt.Sprintf("colony-skill-%d", i)
		skillDir := filepath.Join(aetherDir, "skills", "colony", name)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		writeFile(t, aetherDir, fmt.Sprintf("skills/colony/%s/SKILL.md", name), []byte("test"))
	}

	// Create expected number of domain skills
	for i := 0; i < expectedDomainSkills; i++ {
		name := fmt.Sprintf("domain-skill-%d", i)
		skillDir := filepath.Join(aetherDir, "skills", "domain", name)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		writeFile(t, aetherDir, fmt.Sprintf("skills/domain/%s/SKILL.md", name), []byte("test"))
	}

	fc := newFileChecker(filepath.Join(dir, ".aether", "data"))
	issues := scanWrapperParity(fc)

	// Healthy setup should produce no warnings or criticals
	for _, issue := range issues {
		if issue.Severity == "warning" || issue.Severity == "critical" {
			t.Errorf("healthy setup produced issue: [%s] %s", issue.Severity, issue.Message)
		}
	}
}

// ---------------------------------------------------------------------------
// TestScanWrapperParityMismatch
// ---------------------------------------------------------------------------

func TestScanWrapperParityMismatch(t *testing.T) {
	dir := t.TempDir()
	markAetherSourceCheckout(t, dir)
	aetherDir := filepath.Join(dir, ".aether")

	// Create only 3 YAML commands instead of expected 50
	commandsDir := filepath.Join(aetherDir, "commands")
	if err := os.MkdirAll(commandsDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	for i := 0; i < 3; i++ {
		writeFile(t, aetherDir, fmt.Sprintf("commands/cmd%d.yaml", i), []byte("test"))
	}

	fc := newFileChecker(filepath.Join(dir, ".aether", "data"))
	issues := scanWrapperParity(fc)

	found := false
	for _, issue := range issues {
		if issue.Severity == "warning" && contains(issue.Message, "YAML commands") && contains(issue.Message, "3") && contains(issue.Message, fmt.Sprintf("%d", shippedWrapperCommandCount())) {
			found = true
		}
	}
	if !found {
		t.Errorf("expected mismatch warning for YAML commands; got issues: %+v", issues)
	}
}

// ---------------------------------------------------------------------------
// TestScanWrapperParityCrossSurfaceMismatch
// ---------------------------------------------------------------------------

func TestScanWrapperParityCrossSurfaceMismatch(t *testing.T) {
	dir := t.TempDir()
	markAetherSourceCheckout(t, dir)
	aetherDir := filepath.Join(dir, ".aether")
	claudeDir := filepath.Join(dir, ".claude")
	opencodeDir := filepath.Join(dir, ".opencode")

	// Create directories
	for _, d := range []string{
		filepath.Join(aetherDir, "commands"),
		filepath.Join(claudeDir, "commands", "ant"),
		filepath.Join(opencodeDir, "commands", "ant"),
	} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}

	// Every shipped YAML definition, but two Claude and three OpenCode
	// wrappers short.
	shipped := shippedWrapperCommandCount()
	for i := 0; i < shipped; i++ {
		writeFile(t, aetherDir, fmt.Sprintf("commands/cmd%d.yaml", i), []byte("test"))
	}
	for i := 0; i < shipped-2; i++ {
		writeFile(t, claudeDir, fmt.Sprintf("commands/ant/cmd%d.md", i), []byte("test"))
	}
	for i := 0; i < shipped-3; i++ {
		writeFile(t, opencodeDir, fmt.Sprintf("commands/ant/cmd%d.md", i), []byte("test"))
	}

	fc := newFileChecker(filepath.Join(dir, ".aether", "data"))
	issues := scanWrapperParity(fc)

	// Should have cross-surface command count mismatch warning
	foundCrossSurface := false
	for _, issue := range issues {
		if issue.Severity == "warning" && issue.Category == "wrapper" && contains(issue.Message, "Command count mismatch") {
			foundCrossSurface = true
		}
	}
	if !foundCrossSurface {
		t.Errorf("expected cross-surface command mismatch warning; got: %+v", issues)
	}
}

// ---------------------------------------------------------------------------
// TestScanHubPublishIntegrityHealthy
// ---------------------------------------------------------------------------

func TestScanHubPublishIntegrityHealthy(t *testing.T) {
	hubDir := t.TempDir()
	systemDir := filepath.Join(hubDir, "system")
	t.Setenv("AETHER_HUB_DIR", hubDir)

	for _, dir := range []string{
		filepath.Join(systemDir, "commands", "claude"),
		filepath.Join(systemDir, "commands", "opencode"),
		filepath.Join(systemDir, "agents-claude"),
		filepath.Join(systemDir, "agents"),
		filepath.Join(systemDir, "codex"),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	// The hub carries one wrapper per platform for every menu command this
	// program ships, under the command's own name.
	for name := range wrapperCommandNames {
		writeFile(t, systemDir, "commands/claude/"+name+".md", []byte("test"))
		writeFile(t, systemDir, "commands/opencode/"+name+".md", []byte("test"))
	}
	for i := 0; i < expectedClaudeAgents; i++ {
		writeFile(t, systemDir, fmt.Sprintf("agents-claude/agent%d.md", i), []byte("test"))
	}
	for i := 0; i < expectedOpenCodeAgents; i++ {
		writeFile(t, systemDir, fmt.Sprintf("agents/agent%d.md", i), []byte("test"))
	}
	for i := 0; i < expectedCodexAgents; i++ {
		writeFile(t, systemDir, fmt.Sprintf("codex/agent%d.toml", i), []byte("test"))
	}

	for i := 0; i < expectedColonySkills; i++ {
		name := fmt.Sprintf("colony-skill-%d", i)
		skillDir := filepath.Join(systemDir, "skills", "colony", name)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		writeFile(t, systemDir, fmt.Sprintf("skills/colony/%s/SKILL.md", name), []byte("test"))
	}
	for i := 0; i < expectedDomainSkills; i++ {
		name := fmt.Sprintf("domain-skill-%d", i)
		skillDir := filepath.Join(systemDir, "skills", "domain", name)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		writeFile(t, systemDir, fmt.Sprintf("skills/domain/%s/SKILL.md", name), []byte("test"))
	}

	issues := scanHubPublishIntegrity()
	if len(issues) != 0 {
		t.Errorf("healthy hub publish produced issues: %+v", issues)
	}
}

// ---------------------------------------------------------------------------
// TestScanHubPublishIntegrityMismatch
// ---------------------------------------------------------------------------

func TestScanHubPublishIntegrityMismatch(t *testing.T) {
	hubDir := t.TempDir()
	systemDir := filepath.Join(hubDir, "system")
	t.Setenv("AETHER_HUB_DIR", hubDir)

	if err := os.MkdirAll(filepath.Join(systemDir, "commands", "claude"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeFile(t, systemDir, "commands/claude/only-one.md", []byte("test"))

	issues := scanHubPublishIntegrity()

	found := false
	for _, issue := range issues {
		if issue.Category == "publish" && issue.Severity == "critical" && contains(issue.Message, "Hub Claude commands") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected hub publish integrity failure; got %+v", issues)
	}
}

// ---------------------------------------------------------------------------
// TestDeepScanIncludesWrapperParity
// ---------------------------------------------------------------------------

func TestDeepScanIncludesWrapperParity(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // installed wrappers live under HOME; never read the real one
	dir := t.TempDir()
	markAetherSourceCheckout(t, dir)
	dataDir := filepath.Join(dir, ".aether", "data")
	hubDir := t.TempDir()
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	t.Setenv("AETHER_ROOT", dir)
	t.Setenv("AETHER_HUB_DIR", hubDir)

	// Minimal healthy colony data
	goal := "Deep scan test"
	writeJSONFile(t, dataDir, "COLONY_STATE.json", map[string]string{
		"version": "3.0",
		"goal":    goal,
		"state":   "READY",
	})

	// Create wrapper directories with mismatched counts to trigger wrapper parity issues
	aetherDir := filepath.Join(dir, ".aether")
	commandsDir := filepath.Join(aetherDir, "commands")
	if err := os.MkdirAll(commandsDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Only 1 YAML command instead of expected 50
	writeFile(t, aetherDir, "commands/just-one.yaml", []byte("test"))

	// Run deep scan
	opts := MedicOptions{Deep: true}
	result, err := performHealthScan(opts)
	if err != nil {
		t.Fatalf("performHealthScan failed: %v", err)
	}

	// Should have wrapper parity warning about YAML commands
	foundWrapper := false
	for _, issue := range result.Issues {
		if issue.Category == "wrapper" {
			foundWrapper = true
		}
	}
	if !foundWrapper {
		t.Error("deep scan should include wrapper parity issues")
	}

	foundPublish := false
	for _, issue := range result.Issues {
		if issue.Category == "publish" {
			foundPublish = true
		}
	}
	if !foundPublish {
		t.Error("deep scan should include hub publish integrity issues")
	}

	// Run without deep -- should NOT have wrapper parity issues
	optsNoDeep := MedicOptions{Deep: false}
	resultNoDeep, err := performHealthScan(optsNoDeep)
	if err != nil {
		t.Fatalf("performHealthScan without deep failed: %v", err)
	}
	for _, issue := range resultNoDeep.Issues {
		if issue.Category == "wrapper" || issue.Category == "publish" {
			t.Error("non-deep scan should NOT include wrapper parity or publish integrity issues")
		}
	}
}

// ---------------------------------------------------------------------------
// TestMedicDeepTrustsACorrectInstall
// ---------------------------------------------------------------------------

// medicDeepInstallFaults returns the hub, wrapper and ceremony findings a deep
// scan raised at warning or critical level: the findings that tell the owner
// his install is broken.
func medicDeepInstallFaults(t *testing.T) []HealthIssue {
	t.Helper()
	result, err := performHealthScan(MedicOptions{Deep: true})
	if err != nil {
		t.Fatalf("performHealthScan: %v", err)
	}
	var faults []HealthIssue
	for _, issue := range result.Issues {
		switch issue.Category {
		case "publish", "wrapper", "ceremony":
			if issue.Severity == "critical" || issue.Severity == "warning" {
				faults = append(faults, issue)
			}
		}
	}
	return faults
}

// Phase 210 blocker 17: in an ordinary project with a working install,
// `aether medic --deep` said the shared install had "67 files, expected 60"
// (a typed count that went stale as commands were added) and that the
// build/continue/init/seal/plan wrappers were "not found" (it looked inside
// the project, where `aether update` deliberately removes them; they live in
// the home Claude folder). The install here is the real one: `aether install`
// run against this very source checkout into a throwaway home, so the hub and
// the installed wrappers have exactly the shape the runtime produces.
func TestMedicDeepTrustsACorrectInstall(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("AETHER_CHANNEL", "stable")
	if ok, output := runAntSkillInstall(t, home); !ok {
		t.Fatalf("install failed: %s", output)
	}
	hubDir := filepath.Join(home, ".aether")
	t.Setenv("AETHER_HUB_DIR", hubDir)

	project := t.TempDir()
	if isAetherSourceCheckout(project) {
		t.Fatalf("fixture project %s must be an ordinary project, not an Aether checkout", project)
	}
	goal := "An ordinary project with a working install"
	writeJSONFile(t, filepath.Join(project, ".aether", "data"), "COLONY_STATE.json", colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
	})
	t.Setenv("AETHER_ROOT", project)

	for _, issue := range medicDeepInstallFaults(t) {
		t.Errorf("a correct install was reported as faulty: [%s] %s (%s)", issue.Severity, issue.Message, issue.File)
	}

	// The same checks must still catch a genuinely incomplete install.
	// Remove one published wrapper and one installed wrapper, then rescan.
	if err := os.Remove(filepath.Join(hubDir, "system", "commands", "claude", "build.md")); err != nil {
		t.Fatalf("remove hub wrapper: %v", err)
	}
	installedBuild := filepath.Join(home, ".claude", "commands", claudeCommandDestRelPath("build.md"))
	if err := os.Remove(installedBuild); err != nil {
		t.Fatalf("remove installed wrapper: %v", err)
	}
	var hubCaught, wrapperCaught bool
	for _, issue := range medicDeepInstallFaults(t) {
		if issue.Category == "publish" && issue.Severity == "critical" && contains(issue.Message, "Hub Claude commands") {
			hubCaught = true
		}
		if issue.Category == "ceremony" && contains(issue.Message, "Wrapper for 'build' not found") {
			wrapperCaught = true
		}
	}
	if !hubCaught {
		t.Error("a hub missing a published Claude wrapper was not reported")
	}
	if !wrapperCaught {
		t.Error("a missing installed /ant-build wrapper was not reported")
	}
}
