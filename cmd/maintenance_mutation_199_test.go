package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/downloader"
)

type maintenanceMutation199Fixture struct {
	repository string
	data       string
	hub        string
	claude     string
	opencode   string
	codex      string
	binary     string
	allowlist  lifecycleTransactionAllowlist
}

func newMaintenanceMutation199Fixture(t *testing.T) maintenanceMutation199Fixture {
	t.Helper()
	root := t.TempDir()
	fixture := maintenanceMutation199Fixture{
		repository: filepath.Join(root, "repository"),
		data:       filepath.Join(root, "repository", ".aether", "data"),
		hub:        filepath.Join(root, "hub"),
		claude:     filepath.Join(root, "claude-home"),
		opencode:   filepath.Join(root, "opencode-home"),
		codex:      filepath.Join(root, "codex-home"),
		binary:     filepath.Join(root, "bin", "aether"),
	}
	for _, dir := range []string{fixture.repository, fixture.data, fixture.hub, fixture.claude, fixture.opencode, fixture.codex, filepath.Dir(fixture.binary)} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	fixture.allowlist = lifecycleTransactionAllowlist{
		RepositoryRoot:    fixture.repository,
		LifecycleDataRoot: fixture.data,
		Hub:               lifecycleTransactionHubRoot{Channel: lifecycleTransactionHubStable, Path: fixture.hub},
		ClaudeHome:        fixture.claude,
		OpenCodeHome:      fixture.opencode,
		CodexHome:         fixture.codex,
		BinaryDestination: fixture.binary,
	}
	return fixture
}

func (fixture maintenanceMutation199Fixture) plan(id string) maintenanceMutationPlan {
	return maintenanceMutationPlan{
		SchemaVersion:   maintenanceMutationSchemaVersion,
		Operation:       "update",
		TransactionID:   id,
		SourceRoot:      fixture.hub,
		DestinationRoot: fixture.repository,
		Channel:         channelStable,
		CurrentVersion:  "1.0.0",
		DesiredVersion:  "1.0.1",
		Checkpoint:      "maintenance:update:validated",
		Recovery:        "aether resume",
		Allowlist:       fixture.allowlist,
	}
}

func writeMaintenanceMutation199File(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestMaintenanceMutation199Update(t *testing.T) {
	fixture := newMaintenanceMutation199Fixture(t)
	target := filepath.Join(fixture.repository, ".aether", "managed.md")
	writeMaintenanceMutation199File(t, target, []byte("before\n"))
	beforeInfo, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}

	plan := fixture.plan("maintenance-update-preview")
	plan.Targets = []maintenanceMutationTarget{{
		Root: lifecycleTransactionRootRepository, RelativeTarget: ".aether/managed.md",
		Source: filepath.Join(fixture.hub, "system", "managed.md"), Content: []byte("after\n"), Managed: true,
	}}
	preview, err := prepareMaintenanceMutation(plan)
	if err != nil {
		t.Fatal(err)
	}
	if preview.SchemaVersion != maintenanceMutationSchemaVersion || preview.SourceRoot != fixture.hub || preview.DestinationRoot != fixture.repository || preview.CurrentVersion != "1.0.0" || preview.DesiredVersion != "1.0.1" {
		t.Fatalf("preview lost source/destination/version facts: %#v", preview)
	}
	if len(preview.Targets) != 1 || preview.Targets[0].Change != maintenanceMutationChangeWrite || preview.Targets[0].CurrentDigest == preview.Targets[0].DesiredDigest || preview.Targets[0].CommitOrder != 1 {
		t.Fatalf("preview target = %#v", preview.Targets)
	}
	if got := string(mustReadLifecycleFixtureFile(t, target)); got != "before\n" {
		t.Fatalf("preview mutated target: %q", got)
	}
	afterInfo, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if !afterInfo.ModTime().Equal(beforeInfo.ModTime()) {
		t.Fatal("preview changed target mtime")
	}
	if _, err := os.Stat(filepath.Join(fixture.data, "transactions", plan.TransactionID)); !os.IsNotExist(err) {
		t.Fatalf("preview created coordinator evidence: %v", err)
	}

	result, err := commitMaintenanceMutation(plan)
	if err != nil {
		t.Fatal(err)
	}
	if result.StateEffect != colony.LifecycleStateEffectCommitted || result.Receipt == nil || result.Receipt.Transaction.Stage != colony.TransactionStageVerified {
		t.Fatalf("commit result = %#v", result)
	}
	if got := string(mustReadLifecycleFixtureFile(t, target)); got != "after\n" {
		t.Fatalf("target = %q", got)
	}
	if len(result.Targets) != 1 || result.Targets[0].DesiredDigest != lifecycleDigest([]byte("after\n")) {
		t.Fatalf("result targets = %#v", result.Targets)
	}
}

func TestMaintenanceMutation199Migrate(t *testing.T) {
	fixture := newMaintenanceMutation199Fixture(t)
	original := []byte(`{"version":"2.0","state":"READY","plan":{"phases":[]}}` + "\n")
	statePath := filepath.Join(fixture.data, "COLONY_STATE.json")
	writeMaintenanceMutation199File(t, statePath, original)

	plan, facts, err := prepareStateMigrationMutation(fixture.repository, fixture.data, original, time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if facts["from"] != "2.0" || facts["to"] != "3.0" || plan.SchemaVersion != maintenanceMutationSchemaVersion || len(plan.Targets) != 2 {
		t.Fatalf("migration plan/facts = %#v %#v", plan, facts)
	}
	preview, err := prepareMaintenanceMutation(plan)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Targets) != 2 || preview.Targets[0].Change != maintenanceMutationChangeWrite || preview.Targets[1].Change != maintenanceMutationChangeWrite {
		t.Fatalf("migration preview = %#v", preview.Targets)
	}
	result, err := commitMaintenanceMutation(plan)
	if err != nil {
		t.Fatal(err)
	}
	if result.StateEffect != colony.LifecycleStateEffectCommitted {
		t.Fatalf("migration state effect = %q", result.StateEffect)
	}
	var migrated colony.ColonyState
	readJSON199(t, statePath, &migrated)
	if migrated.Version != "3.0" {
		t.Fatalf("migrated version = %q", migrated.Version)
	}
	backupRel, _ := facts["backup_path"].(string)
	if got := mustReadLifecycleFixtureFile(t, filepath.Join(fixture.data, filepath.FromSlash(backupRel))); !bytes.Equal(got, original) {
		t.Fatalf("migration backup differs: %q", got)
	}
}

func TestMaintenanceMutation199GeneratedSync(t *testing.T) {
	fixture := newMaintenanceMutation199Fixture(t)
	source := filepath.Join(fixture.hub, "system", "commands", "claude")
	writeMaintenanceMutation199File(t, filepath.Join(source, "build.md"), []byte("<!-- Aether-managed: runtime spec at .aether/commands/build.yaml. Synced by aether update. -->\n# Build\n"))
	writeMaintenanceMutation199File(t, filepath.Join(fixture.claude, "ant-pause-colony.md"), []byte("<!-- Generated from .aether/commands/pause-colony.yaml - DO NOT EDIT DIRECTLY -->\nold\n"))
	writeMaintenanceMutation199File(t, filepath.Join(fixture.claude, "custom.md"), []byte("custom\n"))

	plan := fixture.plan("maintenance-generated-sync")
	if err := appendMaintenanceSyncTargets(&plan, maintenanceSyncSpec{
		Root: lifecycleTransactionRootClaudeHome, SourceDir: source, DestinationBase: ".",
		Options:             syncOptions{cleanup: true, mapRelPath: claudeCommandDestRelPath, cleanupInclude: isManagedFlatClaudeCommandPath},
		PruneRetiredAliases: true,
	}); err != nil {
		t.Fatal(err)
	}
	preview, err := prepareMaintenanceMutation(plan)
	if err != nil {
		t.Fatal(err)
	}
	if !maintenancePreviewHas199(preview, "ant-build.md", maintenanceMutationChangeWrite) || !maintenancePreviewHas199(preview, "ant-pause-colony.md", maintenanceMutationChangeRemove) {
		t.Fatalf("generated preview = %#v", preview.Targets)
	}
	if maintenancePreviewHasTarget199(preview, "custom.md") {
		t.Fatalf("custom wrapper entered managed target set: %#v", preview.Targets)
	}
	if _, err := commitMaintenanceMutation(plan); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(fixture.claude, "ant-pause-colony.md")); !os.IsNotExist(err) {
		t.Fatalf("retired managed alias survived: %v", err)
	}
	if got := string(mustReadLifecycleFixtureFile(t, filepath.Join(fixture.claude, "custom.md"))); got != "custom\n" {
		t.Fatalf("custom wrapper changed: %q", got)
	}
}

func TestMaintenanceMutation199PlatformHomes(t *testing.T) {
	fixture := newMaintenanceMutation199Fixture(t)
	plan := fixture.plan("maintenance-platform-homes")
	for index, target := range []struct {
		root lifecycleTransactionRootKind
		rel  string
		body string
	}{
		{lifecycleTransactionRootClaudeHome, "commands/ant-build.md", "claude"},
		{lifecycleTransactionRootOpenCodeHome, "command/build.md", "opencode"},
		{lifecycleTransactionRootCodexHome, "agents/aether-builder.toml", "codex"},
	} {
		label := []string{"Commands (claude)", "Agents (opencode)", "Agents (codex)"}[index]
		plan.Targets = append(plan.Targets, maintenanceMutationTarget{Root: target.root, RelativeTarget: target.rel, Label: label, Source: "hub", Content: []byte(target.body), Managed: true})
		_ = index
	}
	preview, err := prepareMaintenanceMutation(plan)
	if err != nil {
		t.Fatal(err)
	}
	if got := []int{preview.Targets[0].CommitOrder, preview.Targets[1].CommitOrder, preview.Targets[2].CommitOrder}; !reflect.DeepEqual(got, []int{1, 2, 3}) {
		t.Fatalf("commit order = %v", got)
	}
	result, err := commitMaintenanceMutation(plan)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Receipt.Changes) != 3 || len(result.Receipt.Verification) != 3 {
		t.Fatalf("platform receipt = %#v", result.Receipt)
	}
	details, _, _ := maintenancePreviewSyncDetails(result.Preview)
	if got := platformRestartTargets(details); !reflect.DeepEqual(got, []string{"Codex agents", "OpenCode agents"}) {
		t.Fatalf("platform restart targets = %v; details = %#v", got, details)
	}
}

func TestMaintenanceMutation199DownloadBinary(t *testing.T) {
	fixture := newMaintenanceMutation199Fixture(t)
	fetch := func(version, destDir string) (*downloader.DownloadResult, error) {
		path := filepath.Join(destDir, "aether")
		writeMaintenanceMutation199File(t, path, []byte("#!/bin/sh\nprintf '{\"ok\":true,\"result\":\""+version+"\"}\\n'\n"))
		if err := os.Chmod(path, 0o755); err != nil {
			t.Fatal(err)
		}
		return &downloader.DownloadResult{Success: true, Path: path, Version: version}, nil
	}
	staged, err := stageMaintenanceBinaryDownload("1.2.3", channelStable, fetch)
	if err != nil {
		t.Fatal(err)
	}
	if staged.Version != "1.2.3" || staged.Name != "aether" || staged.Mode.Perm() != 0o755 {
		t.Fatalf("staged binary = %#v", staged)
	}
	plan := fixture.plan("maintenance-binary")
	plan.DesiredVersion = staged.Version
	plan.Targets = []maintenanceMutationTarget{{Root: lifecycleTransactionRootBinaryDestination, RelativeTarget: filepath.Base(fixture.binary), Source: staged.Source, Content: staged.Content, Mode: staged.Mode, Managed: true}}
	result, err := commitMaintenanceMutation(plan)
	if err != nil {
		t.Fatal(err)
	}
	if result.StateEffect != colony.LifecycleStateEffectCommitted {
		t.Fatalf("binary result = %#v", result)
	}
	info, err := os.Stat(fixture.binary)
	if err != nil || info.Mode().Perm() != 0o755 {
		t.Fatalf("installed binary mode = %v, err=%v", info.Mode(), err)
	}
	if err := verifyMaintenanceInstalledBinary(fixture.binary, staged.Version); err != nil {
		t.Fatalf("installed binary verification failed: %v", err)
	}
	if len(result.Preview.Targets) != 1 || result.Preview.Targets[0].DesiredMode != 0o755 {
		t.Fatalf("binary preview lost executable mode: %#v", result.Preview.Targets)
	}
	manifestPaths, err := filepath.Glob(filepath.Join(filepath.Dir(fixture.binary), lifecycleTransactionDirectory, plan.TransactionID, "*", "manifest.json"))
	if err != nil || len(manifestPaths) != 1 {
		t.Fatalf("binary root manifest paths = %v, err=%v", manifestPaths, err)
	}
	var manifest lifecycleTransactionRootManifest
	readJSON199(t, manifestPaths[0], &manifest)
	if len(manifest.Targets) != 1 || os.FileMode(manifest.Targets[0].Mode).Perm() != 0o755 {
		t.Fatalf("binary root manifest lost executable mode: %#v", manifest.Targets)
	}
	if err := os.Chmod(fixture.binary, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := commitMaintenanceMutation(plan); err == nil {
		t.Fatal("verified binary receipt accepted post-commit executable-mode drift")
	}

	if _, err := stageMaintenanceBinaryDownload("1.2.3", channelStable, func(string, string) (*downloader.DownloadResult, error) {
		return nil, errors.New("network unavailable")
	}); err == nil {
		t.Fatal("binary fetch failure was accepted")
	}
}

func TestMaintenanceMutation199ChannelIsolation(t *testing.T) {
	fixture := newMaintenanceMutation199Fixture(t)
	plan := fixture.plan("maintenance-channel-isolation")
	plan.Channel = channelDev
	plan.Targets = []maintenanceMutationTarget{{Root: lifecycleTransactionRootClaudeHome, RelativeTarget: "commands/ant-build.md", Source: "dev hub", Content: []byte("dev"), Managed: true}}
	if _, err := prepareMaintenanceMutation(plan); err == nil {
		t.Fatal("dev maintenance was allowed to write stable platform homes")
	}

	plan = fixture.plan("maintenance-hub-channel-mismatch")
	plan.Allowlist.Hub.Channel = lifecycleTransactionHubDev
	if _, err := prepareMaintenanceMutation(plan); err == nil {
		t.Fatal("stable operation accepted dev hub identity")
	}
}

func TestMaintenanceMutation199VersionAgreement(t *testing.T) {
	root := t.TempDir()
	hub := t.TempDir()
	writeMaintenanceMutation199File(t, filepath.Join(root, "go.mod"), []byte("module github.com/calcosmic/Aether\n"))
	writeMaintenanceMutation199File(t, filepath.Join(root, "cmd", "aether", "main.go"), []byte("package main\n"))
	writeMaintenanceMutation199File(t, filepath.Join(root, ".aether", "version.json"), []byte(`{"version":"1.2.3"}`))
	writeMaintenanceMutation199File(t, filepath.Join(root, "npm", "package.json"), []byte(`{"version":"1.2.3"}`))
	writeMaintenanceMutation199File(t, filepath.Join(hub, "system", "version.json"), []byte(`{"version":"1.2.3"}`))
	if err := validateMaintenanceVersionAgreement(root, hub, "1.2.3"); err != nil {
		t.Fatal(err)
	}
	if err := validateMaintenanceVersionAgreement(filepath.Join(root, "cmd"), hub, "1.2.3"); err != nil {
		t.Fatalf("source subdirectory was not normalized to module root: %v", err)
	}
	writeMaintenanceMutation199File(t, filepath.Join(root, "npm", "package.json"), []byte(`{"version":"1.2.4"}`))
	if err := validateMaintenanceVersionAgreement(root, hub, "1.2.3"); err == nil {
		t.Fatal("source/npm version mismatch was accepted")
	}
}

func TestMaintenanceMutation199UpdateFlagContract(t *testing.T) {
	usage := updateCmd.Flags().Lookup("sync-platform-homes").Usage
	if strings.Contains(strings.ToLower(usage), "for dev") || !strings.Contains(strings.ToLower(usage), "stable") {
		t.Fatalf("update --sync-platform-homes help contradicts dev isolation: %q", usage)
	}
}

func TestMaintenanceMutation199CrossRootRollback(t *testing.T) {
	fixture := newMaintenanceMutation199Fixture(t)
	repoTarget := filepath.Join(fixture.repository, "managed.txt")
	claudeTarget := filepath.Join(fixture.claude, "managed.txt")
	writeMaintenanceMutation199File(t, repoTarget, []byte("repo-before"))
	if err := os.Chmod(repoTarget, 0o600); err != nil {
		t.Fatal(err)
	}
	writeMaintenanceMutation199File(t, claudeTarget, []byte("claude-before"))
	plan := fixture.plan("maintenance-cross-root-rollback")
	plan.Targets = []maintenanceMutationTarget{
		{Root: lifecycleTransactionRootRepository, RelativeTarget: "managed.txt", Source: "hub/repo", Content: []byte("repo-after"), Mode: 0o755, Managed: true},
		{Root: lifecycleTransactionRootClaudeHome, RelativeTarget: "managed.txt", Source: "hub/claude", Content: []byte("claude-after"), Managed: true},
	}
	plan.Rename = func(oldPath, newPath string) error {
		if newPath == claudeTarget {
			return syscall.EXDEV
		}
		return os.Rename(oldPath, newPath)
	}
	result, err := commitMaintenanceMutation(plan)
	if !errors.Is(err, syscall.EXDEV) {
		t.Fatalf("commit error = %v, want EXDEV", err)
	}
	if result.StateEffect != colony.LifecycleStateEffectRolledBack || result.Receipt == nil || result.Receipt.Transaction.Stage != colony.TransactionStageRolledBack {
		t.Fatalf("rollback result = %#v", result)
	}
	if string(mustReadLifecycleFixtureFile(t, repoTarget)) != "repo-before" || string(mustReadLifecycleFixtureFile(t, claudeTarget)) != "claude-before" {
		t.Fatal("cross-root failure did not restore exact prior bytes")
	}
	if info, statErr := os.Stat(repoTarget); statErr != nil {
		t.Fatalf("cross-root rollback target unavailable: %v", statErr)
	} else if info.Mode().Perm() != 0o600 {
		t.Fatalf("cross-root failure did not restore prior mode: %04o", info.Mode().Perm())
	}
}

func TestMaintenanceMutation199Replay(t *testing.T) {
	fixture := newMaintenanceMutation199Fixture(t)
	plan := fixture.plan("maintenance-replay")
	plan.Targets = []maintenanceMutationTarget{{Root: lifecycleTransactionRootRepository, RelativeTarget: "once.txt", Source: "hub", Content: []byte("once"), Managed: true}}
	first, err := commitMaintenanceMutation(plan)
	if err != nil {
		t.Fatal(err)
	}
	second, err := commitMaintenanceMutation(plan)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first.Receipt, second.Receipt) || first.Receipt.ReceiptID == "" {
		t.Fatalf("replay receipt changed:\nfirst=%#v\nsecond=%#v", first.Receipt, second.Receipt)
	}
}

func TestMaintenanceMutation199CustomPreserved(t *testing.T) {
	fixture := newMaintenanceMutation199Fixture(t)
	source := filepath.Join(fixture.hub, "system", "commands", "claude")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	// A real hub always ships commands; an empty source is refused outright
	// (2026-09-12 incident), so seed one live wrapper alongside the stale one.
	writeMaintenanceMutation199File(t, filepath.Join(source, "ant-build.md"), []byte("<!-- Generated from .aether/commands/build.yaml - DO NOT EDIT DIRECTLY -->\nlive\n"))
	customPath := filepath.Join(fixture.claude, "ant-custom.md")
	writeMaintenanceMutation199File(t, customPath, []byte("# owner custom command\n"))
	managedPath := filepath.Join(fixture.claude, "ant-stale.md")
	writeMaintenanceMutation199File(t, managedPath, []byte("<!-- Generated from .aether/commands/stale.yaml - DO NOT EDIT DIRECTLY -->\nstale\n"))

	plan := fixture.plan("maintenance-custom-preserved")
	if err := appendMaintenanceSyncTargets(&plan, maintenanceSyncSpec{
		Root: lifecycleTransactionRootClaudeHome, SourceDir: source, DestinationBase: ".",
		Options: syncOptions{cleanup: true, cleanupInclude: isManagedFlatClaudeCommandPath},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := commitMaintenanceMutation(plan); err != nil {
		t.Fatal(err)
	}
	if got := string(mustReadLifecycleFixtureFile(t, customPath)); got != "# owner custom command\n" {
		t.Fatalf("custom path changed: %q", got)
	}
	if _, err := os.Stat(managedPath); !os.IsNotExist(err) {
		t.Fatalf("stale managed path survived: %v", err)
	}
}

func maintenancePreviewHas199(preview maintenanceMutationPreview, target string, change maintenanceMutationChange) bool {
	for _, candidate := range preview.Targets {
		if filepath.ToSlash(candidate.RelativeTarget) == filepath.ToSlash(target) && candidate.Change == change {
			return true
		}
	}
	return false
}

func maintenancePreviewHasTarget199(preview maintenanceMutationPreview, target string) bool {
	for _, candidate := range preview.Targets {
		if filepath.ToSlash(candidate.RelativeTarget) == filepath.ToSlash(target) {
			return true
		}
	}
	return false
}
