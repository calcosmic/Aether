package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

type maintenanceState199Fixture struct {
	repository string
	data       string
	hub        string
}

func newMaintenanceState199Fixture(t *testing.T) maintenanceState199Fixture {
	t.Helper()
	root := t.TempDir()
	fixture := maintenanceState199Fixture{
		repository: filepath.Join(root, "repository"),
		data:       filepath.Join(root, "repository", ".aether", "data"),
		hub:        filepath.Join(root, "hub"),
	}
	for _, dir := range []string{fixture.repository, fixture.data, fixture.hub} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return fixture
}

func writeMaintenanceState199File(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
}

func (fixture maintenanceState199Fixture) cleanupRequest(id string, targets ...maintenanceCleanupTarget) maintenanceCleanupRequest {
	return maintenanceCleanupRequest{
		RepositoryRoot: fixture.repository,
		DataRoot:       fixture.data,
		TransactionID:  id,
		Manifest: maintenanceCleanupManifest{
			SchemaVersion: maintenanceCleanupSchemaVersion,
			Owner:         maintenanceCleanupOwner,
			Checkpoint:    maintenanceCleanupCheckpoint,
			Targets:       targets,
		},
	}
}

func TestMaintenanceState199CleanupOwnership(t *testing.T) {
	fixture := newMaintenanceState199Fixture(t)
	ownedRel := "worker-debug/owned.json"
	decoyRel := "worker-debug/owned-decoy.json"
	ownedPath := filepath.Join(fixture.data, filepath.FromSlash(ownedRel))
	decoyPath := filepath.Join(fixture.data, filepath.FromSlash(decoyRel))
	owned := []byte(`{"schema_version":"worker-debug/v1","owner":"aether-runtime"}` + "\n")
	writeMaintenanceState199File(t, ownedPath, owned)
	writeMaintenanceState199File(t, decoyPath, []byte("custom\n"))
	before, err := os.Stat(ownedPath)
	if err != nil {
		t.Fatal(err)
	}

	request := fixture.cleanupRequest("maintenance-cleanup-ownership", maintenanceCleanupTarget{
		RelativePath: ownedRel,
		Owner:        maintenanceCleanupOwner,
		Digest:       lifecycleDigest(owned),
	})
	plan, err := prepareMaintenanceCleanup(request)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Preview.Checkpoint != maintenanceCleanupCheckpoint || len(plan.Preview.Targets) != 1 || plan.Preview.Targets[0].RelativeTarget != filepath.FromSlash(ownedRel) || plan.Preview.Targets[0].Change != maintenanceMutationChangeRemove {
		t.Fatalf("cleanup preview = %#v", plan.Preview)
	}
	if got := mustReadLifecycleFixtureFile(t, ownedPath); !bytes.Equal(got, owned) {
		t.Fatalf("preview changed owned target: %q", got)
	}
	after, err := os.Stat(ownedPath)
	if err != nil {
		t.Fatal(err)
	}
	if !after.ModTime().Equal(before.ModTime()) {
		t.Fatal("cleanup preview changed target mtime")
	}
	if _, err := os.Stat(filepath.Join(fixture.data, "transactions", request.TransactionID)); !os.IsNotExist(err) {
		t.Fatalf("cleanup preview created transaction evidence: %v", err)
	}

	result, err := commitMaintenanceCleanup(plan)
	if err != nil {
		t.Fatal(err)
	}
	if result.StateEffect != colony.LifecycleStateEffectCommitted || result.Receipt == nil || len(result.Receipt.Changes) != 1 || result.Receipt.Changes[0].AfterDigest != lifecycleTransactionMissingDigest {
		t.Fatalf("cleanup receipt = %#v", result)
	}
	if _, err := os.Stat(ownedPath); !os.IsNotExist(err) {
		t.Fatalf("owned target survived cleanup: %v", err)
	}
	if got := string(mustReadLifecycleFixtureFile(t, decoyPath)); got != "custom\n" {
		t.Fatalf("prefix-sharing decoy changed: %q", got)
	}

	for name, mutate := range map[string]func(*maintenanceCleanupRequest){
		"unknown owner": func(input *maintenanceCleanupRequest) { input.Manifest.Owner = "caller" },
		"path escape":   func(input *maintenanceCleanupRequest) { input.Manifest.Targets[0].RelativePath = "../decoy" },
	} {
		t.Run(name, func(t *testing.T) {
			input := fixture.cleanupRequest("maintenance-cleanup-reject-"+strings.ReplaceAll(name, " ", "-"), maintenanceCleanupTarget{RelativePath: decoyRel, Owner: maintenanceCleanupOwner, Digest: lifecycleDigest([]byte("custom\n"))})
			mutate(&input)
			if _, err := prepareMaintenanceCleanup(input); err == nil {
				t.Fatalf("%s was accepted", name)
			}
			if got := string(mustReadLifecycleFixtureFile(t, decoyPath)); got != "custom\n" {
				t.Fatalf("rejected cleanup changed decoy: %q", got)
			}
		})
	}

	outside := filepath.Join(filepath.Dir(fixture.data), "outside")
	writeMaintenanceState199File(t, outside, []byte("outside"))
	symlinkRel := "worker-debug/link"
	if err := os.Symlink(outside, filepath.Join(fixture.data, filepath.FromSlash(symlinkRel))); err != nil {
		t.Fatal(err)
	}
	symlinkInput := fixture.cleanupRequest("maintenance-cleanup-reject-symlink", maintenanceCleanupTarget{RelativePath: symlinkRel, Owner: maintenanceCleanupOwner, Digest: lifecycleDigest([]byte("outside"))})
	if _, err := prepareMaintenanceCleanup(symlinkInput); err == nil {
		t.Fatal("symlink cleanup target was accepted")
	}
	if got := string(mustReadLifecycleFixtureFile(t, outside)); got != "outside" {
		t.Fatalf("symlink rejection changed outside target: %q", got)
	}
}

func TestMaintenanceState199CleanupRollback(t *testing.T) {
	fixture := newMaintenanceState199Fixture(t)
	firstRel := "worker-debug/first.json"
	secondRel := "worker-debug/second.json"
	firstPath := filepath.Join(fixture.data, filepath.FromSlash(firstRel))
	secondPath := filepath.Join(fixture.data, filepath.FromSlash(secondRel))
	first := []byte("first-before\n")
	second := []byte("second-before\n")
	writeMaintenanceState199File(t, firstPath, first)
	writeMaintenanceState199File(t, secondPath, second)

	request := fixture.cleanupRequest("maintenance-cleanup-rollback",
		maintenanceCleanupTarget{RelativePath: firstRel, Owner: maintenanceCleanupOwner, Digest: lifecycleDigest(first)},
		maintenanceCleanupTarget{RelativePath: secondRel, Owner: maintenanceCleanupOwner, Digest: lifecycleDigest(second)},
	)
	request.Fault = func(point string) error {
		if point == "after_target_commit:target-0001" {
			return errors.New("injected cleanup failure")
		}
		return nil
	}
	plan, err := prepareMaintenanceCleanup(request)
	if err != nil {
		t.Fatal(err)
	}
	result, err := commitMaintenanceCleanup(plan)
	if err == nil {
		t.Fatal("cleanup fault unexpectedly committed")
	}
	if result.StateEffect != colony.LifecycleStateEffectRolledBack || result.Receipt == nil || result.Receipt.Transaction.Stage != colony.TransactionStageRolledBack {
		t.Fatalf("cleanup rollback result = %#v (err=%v)", result, err)
	}
	if got := mustReadLifecycleFixtureFile(t, firstPath); !bytes.Equal(got, first) {
		t.Fatalf("first target not restored: %q", got)
	}
	if got := mustReadLifecycleFixtureFile(t, secondPath); !bytes.Equal(got, second) {
		t.Fatalf("second target not restored: %q", got)
	}
}

func (fixture maintenanceState199Fixture) registryRequest(id, repo string) registryMutationRequest {
	return registryMutationRequest{
		SchemaVersion:     registryMutationSchemaVersion,
		Operation:         registryMutationUpsert,
		TransactionID:     id,
		RepositoryRoot:    fixture.repository,
		LifecycleDataRoot: fixture.data,
		HubRoot:           fixture.hub,
		RepoPath:          repo,
		RepoIdentity:      stableRepoIdentity(repo),
		ExpectedBaseline:  lifecycleTransactionMissingDigest,
		Domains:           []string{"go"},
		Goal:              "transactional maintenance",
		Active:            true,
		RegisteredAt:      time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC),
	}
}

func TestMaintenanceState199RegistryBaseline(t *testing.T) {
	fixture := newMaintenanceState199Fixture(t)
	repo := filepath.Join(fixture.repository, "colony")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	registryPath := filepath.Join(fixture.hub, "registry", "registry.json")
	original := []byte("{\"colonies\":[]}\n")
	writeMaintenanceState199File(t, registryPath, original)
	request := fixture.registryRequest("maintenance-registry-baseline", repo)
	request.ExpectedBaseline = lifecycleDigest(original)
	plan, err := prepareRegistryMutation(request)
	if err != nil {
		t.Fatal(err)
	}

	concurrent := []byte("{\"colonies\":[],\"external\":true}\n")
	writeMaintenanceState199File(t, registryPath, concurrent)
	result, err := commitRegistryMutation(plan)
	if err == nil {
		t.Fatal("registry baseline change was accepted")
	}
	if result.StateEffect != colony.LifecycleStateEffectNone || result.Receipt != nil {
		t.Fatalf("baseline conflict reported mutation: %#v", result)
	}
	if got := mustReadLifecycleFixtureFile(t, registryPath); !bytes.Equal(got, concurrent) {
		t.Fatalf("baseline conflict changed registry: %q", got)
	}

	duplicate := registryData{Colonies: []registryEntry{
		{RepoID: stableRepoIdentity(repo), RepoPath: repo, Active: true},
		{RepoID: stableRepoIdentity(repo), RepoPath: repo + "-other", Active: true},
	}}
	if err := writeRegistry(registryPath, duplicate); err != nil {
		t.Fatal(err)
	}
	duplicateBytes := mustReadLifecycleFixtureFile(t, registryPath)
	request = fixture.registryRequest("maintenance-registry-duplicate", repo)
	request.ExpectedBaseline = lifecycleDigest(duplicateBytes)
	if _, err := prepareRegistryMutation(request); err == nil {
		t.Fatal("duplicate stable repository identity was accepted")
	}
	if got := mustReadLifecycleFixtureFile(t, registryPath); !bytes.Equal(got, duplicateBytes) {
		t.Fatalf("duplicate rejection changed registry: %q", got)
	}
}

func TestMaintenanceState199RegistryReceipt(t *testing.T) {
	fixture := newMaintenanceState199Fixture(t)
	repo := filepath.Join(fixture.repository, "colony")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	request := fixture.registryRequest("maintenance-registry-receipt", repo)
	plan, err := prepareRegistryMutation(request)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Preview.BaselineDigest != lifecycleTransactionMissingDigest || plan.Preview.RepositoryIdentity != stableRepoIdentity(repo) || plan.Preview.DesiredDigest == "" {
		t.Fatalf("registry preview = %#v", plan.Preview)
	}
	result, err := commitRegistryMutation(plan)
	if err != nil {
		t.Fatal(err)
	}
	if result.StateEffect != colony.LifecycleStateEffectCommitted || result.Receipt == nil || result.Receipt.Transaction.Stage != colony.TransactionStageVerified || result.Recovery == "" {
		t.Fatalf("registry result = %#v", result)
	}
	registryPath := filepath.Join(fixture.hub, "registry", "registry.json")
	registryBytes := mustReadLifecycleFixtureFile(t, registryPath)
	if len(result.Receipt.Changes) != 1 || result.Receipt.Changes[0].AfterDigest != lifecycleDigest(registryBytes) || result.Preview.DesiredDigest != lifecycleDigest(registryBytes) {
		t.Fatalf("registry receipt digest does not match bytes: %#v", result)
	}
	var registry registryData
	readJSON199(t, registryPath, &registry)
	if len(registry.Colonies) != 1 || registry.Colonies[0].RepoPath != repo || registry.Colonies[0].RepoID != stableRepoIdentity(repo) || !registry.Colonies[0].Active || registry.Colonies[0].FinalStats != nil {
		t.Fatalf("registry contents = %#v", registry)
	}
}
