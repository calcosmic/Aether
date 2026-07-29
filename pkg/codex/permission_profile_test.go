package codex

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPermissionProfileForCasteUsesOnlyEnforceableReleaseProfiles(t *testing.T) {
	tests := []struct {
		caste string
		want  PermissionProfileName
	}{
		{"scout", PermissionWorkspaceWrite},
		{"aether-includer", PermissionRepositoryReadOnly},
		{"builder", PermissionWorkspaceWrite},
		{"watcher", PermissionWorkspaceWrite},
		{"probe", PermissionWorkspaceWrite},
		{"surveyor-nest", PermissionWorkspaceWrite},
		{"custom-caste", PermissionWorkspaceWrite},
	}

	for _, tt := range tests {
		t.Run(tt.caste, func(t *testing.T) {
			profile := PermissionProfileForCaste(tt.caste)
			if profile.SchemaVersion != PermissionProfileSchemaVersion || profile.Name != tt.want {
				t.Fatalf("profile = %+v, want %q", profile, tt.want)
			}
			if profile.Approval != "never" || profile.Network != "provider_default" {
				t.Fatalf("profile omitted execution capabilities: %+v", profile)
			}
		})
	}
}

// TestScoutPermissionProfileAllowsPhaseResearchWrite pins the D-05 fix: scout
// converges onto the same default-workspace-write-plus-behavioral-restriction
// pattern the surveyor castes already use, so the permission profile no longer
// contradicts the write renderPhaseResearchBrief orders. includer stays the
// control case proving the carve-out is scout-specific, not a general
// loosening of repositoryReadOnlyCastes.
func TestScoutPermissionProfileAllowsPhaseResearchWrite(t *testing.T) {
	scout := PermissionProfileForCaste("scout")
	if scout.Name != PermissionWorkspaceWrite {
		t.Fatalf("scout profile = %q, want %q", scout.Name, PermissionWorkspaceWrite)
	}
	found := false
	for _, restriction := range scout.BehavioralRestrictions {
		if strings.Contains(restriction, ".aether/data/phase-research") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("scout behavioral restrictions do not name .aether/data/phase-research: %v", scout.BehavioralRestrictions)
	}

	includer := PermissionProfileForCaste("includer")
	if includer.Name != PermissionRepositoryReadOnly {
		t.Fatalf("includer profile = %q, want %q (carve-out must stay scout-specific)", includer.Name, PermissionRepositoryReadOnly)
	}
}

func TestCodexReadOnlyProfileSelectsReadOnlySandbox(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture uses POSIX sh")
	}
	dir := t.TempDir()
	agentPath := filepath.Join(dir, "aether-includer.toml")
	if err := os.WriteFile(agentPath, []byte(`name = "aether-includer"
description = "Includer"
nickname_candidates = ["includer"]
developer_instructions = '''Inspect only.'''
`), 0644); err != nil {
		t.Fatal(err)
	}
	argsPath := filepath.Join(dir, "args.txt")
	binary := filepath.Join(dir, "codex")
	script := `#!/bin/sh
if [ "$1" = "login" ] && [ "$2" = "status" ]; then
  echo "Logged in"
  exit 0
fi
printf '%s\n' "$@" > "$ARGS_PATH"
out=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then out="$2"; shift 2; else shift; fi
done
cat >/dev/null
printf '{"ant_name":"Includer-1","caste":"includer","task_id":"s.1","status":"completed","summary":"inspected","files_created":[],"files_modified":[],"tests_written":[],"tool_count":0,"blockers":[],"spawns":[]}' > "$out"
`
	if err := os.WriteFile(binary, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ARGS_PATH", argsPath)
	invoker := &RealInvoker{binaryName: binary}
	_, err := invoker.Invoke(context.Background(), WorkerConfig{
		AgentName:         "aether-includer",
		AgentTOMLPath:     agentPath,
		Caste:             "includer",
		WorkerName:        "Includer-1",
		TaskID:            "s.1",
		TaskBrief:         "Inspect",
		Root:              dir,
		PermissionProfile: PermissionProfileForCaste("includer"),
	})
	if err != nil {
		t.Fatal(err)
	}
	args, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(args)
	if !strings.Contains(text, "--sandbox\nread-only") || strings.Contains(text, "workspace-write") || strings.Contains(text, "--add-dir") {
		t.Fatalf("Codex read-only boundary was not applied:\n%s", text)
	}
}

func TestClaudeWorkspaceSettingsFailClosed(t *testing.T) {
	path, err := writeClaudeWorkspaceSettings()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(payload)
	for _, want := range []string{`"enabled":true`, `"failIfUnavailable":true`, `"allowUnsandboxedCommands":false`, `"disableBypassPermissionsMode":"disable"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("Claude sandbox settings missing %s: %s", want, text)
		}
	}
}

func TestShippedOpenCodeAgentsDeclarePermissionBoundary(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	matches, err := filepath.Glob(filepath.Join(root, ".opencode", "agents", "aether-*.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) < 28 {
		t.Fatalf("OpenCode agent definitions = %d, want 27 castes plus router", len(matches))
	}
	for _, path := range matches {
		base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		if base == defaultOpenCodePrimaryAgent {
			continue
		}
		caste := strings.TrimPrefix(base, "aether-")
		if err := validateOpenCodePermissionBoundary(path, PermissionProfileForCaste(caste)); err != nil {
			t.Fatalf("%s: %v", base, err)
		}
	}
	if err := validateOpenCodeRouter(root); err != nil {
		t.Fatal(err)
	}
}

func TestResolvePermissionProfileRejectsBroadeningAndStaleContracts(t *testing.T) {
	readOnly := PermissionProfileForCaste("includer")
	broad := PermissionProfileForCaste("builder")
	if _, err := ResolvePermissionProfile("includer", broad); err == nil || !strings.Contains(err.Error(), "mismatch") {
		t.Fatalf("read-only caste accepted broader profile: %v", err)
	}

	stale := readOnly
	stale.SchemaVersion = 99
	if _, err := ResolvePermissionProfile("includer", stale); err == nil || !strings.Contains(err.Error(), "mismatch") {
		t.Fatalf("stale profile was accepted: %v", err)
	}

	// Behavioral metadata is descriptive and may be absent from dynamic child
	// requests; Go restores the canonical copy after validating enforcement fields.
	watcher := PermissionProfileForCaste("watcher")
	request := watcher
	request.BehavioralRestrictions = nil
	resolved, err := ResolvePermissionProfile("watcher", request)
	if err != nil {
		t.Fatalf("canonical enforcement fields were rejected: %v", err)
	}
	if len(resolved.BehavioralRestrictions) == 0 {
		t.Fatal("canonical behavioral limitations were not restored")
	}
}

func TestPermissionDecisionMatrixFailsClosedForUnsupportedNarrowWrites(t *testing.T) {
	for _, platform := range []Platform{PlatformClaude, PlatformOpenCode, PlatformCodex, PlatformFake} {
		for _, caste := range []string{"scout", "builder"} {
			decision, err := ResolvePermissionDecision(platform, caste, PermissionProfileForCaste(caste))
			if err != nil || !decision.Allowed {
				t.Fatalf("%s/%s rejected canonical profile: decision=%+v err=%v", platform, caste, decision, err)
			}
		}
	}

	for _, name := range []PermissionProfileName{PermissionScopedWrite, PermissionTestWrite} {
		profile := PermissionProfile{
			SchemaVersion: PermissionProfileSchemaVersion,
			Name:          name,
			Filesystem:    FilesystemPermission(name),
			Shell:         "within_filesystem_boundary",
			Network:       "provider_default",
			Approval:      "never",
		}
		decision := PermissionDecisionFor(PlatformCodex, profile)
		if decision.Allowed || decision.Enforcement != CapabilityUnavailable {
			t.Fatalf("unsupported profile %q did not fail closed: %+v", name, decision)
		}
	}
}

func TestRenderPermissionProfileSectionDistinguishesEnforcementFromBehavior(t *testing.T) {
	decision, err := ResolvePermissionDecision(PlatformCodex, "probe", PermissionProfileForCaste("probe"))
	if err != nil {
		t.Fatal(err)
	}
	section := RenderPermissionProfileSection(decision)
	for _, want := range []string{"workspace_write", "not a narrower host sandbox", "write test files only"} {
		if !strings.Contains(section, want) {
			t.Fatalf("permission section missing %q:\n%s", want, section)
		}
	}
}
