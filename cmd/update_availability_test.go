package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func withHubVersion(t *testing.T, version string) string {
	t.Helper()
	hub := t.TempDir()
	if err := os.MkdirAll(hub, 0755); err != nil {
		t.Fatalf("mkdir hub: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hub, "version.json"),
		[]byte(`{"version":"`+version+`"}`), 0644); err != nil {
		t.Fatalf("write hub version: %v", err)
	}
	t.Setenv("AETHER_HUB_DIR", hub)
	return hub
}

func aetherRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".aether", "data"), 0755); err != nil {
		t.Fatalf("mkdir repo .aether/data: %v", err)
	}
	return repo
}

// TestUpdateAvailableWhenRepoIsBehindHub is the gate for the update prompt: a
// repo that synced an older version must be told, because nothing else tells
// it. One repo on this machine sat on v1.0.25 while the hub was 23 releases
// ahead, with no signal anywhere.
func TestUpdateAvailableWhenRepoIsBehindHub(t *testing.T) {
	withHubVersion(t, "1.0.48")
	repo := aetherRepo(t)
	if err := writeInstalledVersionMarker(repo, "1.0.25"); err != nil {
		t.Fatalf("stamp repo: %v", err)
	}

	availability := checkUpdateAvailable(repo)
	if !availability.Available {
		t.Fatalf("repo on 1.0.25 vs hub 1.0.48 reported as current: %+v", availability)
	}
	warning := renderUpdateAvailableWarning(availability)
	for _, want := range []string{"1.0.25", "1.0.48", "aether update --force"} {
		if !strings.Contains(warning, want) {
			t.Fatalf("warning missing %q: %s", want, warning)
		}
	}
}

// A current repo must stay silent — a notification that always fires is noise
// the user learns to ignore.
func TestNoUpdatePromptWhenRepoIsCurrent(t *testing.T) {
	withHubVersion(t, "1.0.48")
	repo := aetherRepo(t)
	if err := writeInstalledVersionMarker(repo, "1.0.48"); err != nil {
		t.Fatalf("stamp repo: %v", err)
	}
	availability := checkUpdateAvailable(repo)
	if availability.Available {
		t.Fatalf("current repo prompted for an update: %+v", availability)
	}
	if warning := renderUpdateAvailableWarning(availability); warning != "" {
		t.Fatalf("current repo produced a warning: %s", warning)
	}
}

// A repo ahead of the hub (source checkout mid-development) must stay silent.
func TestNoUpdatePromptWhenRepoIsAheadOfHub(t *testing.T) {
	withHubVersion(t, "1.0.48")
	repo := aetherRepo(t)
	if err := writeInstalledVersionMarker(repo, "1.0.49"); err != nil {
		t.Fatalf("stamp repo: %v", err)
	}
	if checkUpdateAvailable(repo).Available {
		t.Fatal("repo ahead of the hub should not be prompted to update")
	}
}

// An un-stamped Aether repo (updated before stamping existed) is prompted, so
// existing repos get the notification on first status after upgrading.
func TestUnstampedAetherRepoIsPrompted(t *testing.T) {
	withHubVersion(t, "1.0.48")
	repo := aetherRepo(t)
	availability := checkUpdateAvailable(repo)
	if !availability.Available || !availability.NeverStamped {
		t.Fatalf("un-stamped Aether repo should be prompted: %+v", availability)
	}
	if !strings.Contains(renderUpdateAvailableWarning(availability), "no record of its last sync") {
		t.Fatalf("un-stamped warning should explain why: %s", renderUpdateAvailableWarning(availability))
	}
}

// A directory that does not use Aether must never be nagged.
func TestNonAetherDirectoryIsNeverPrompted(t *testing.T) {
	withHubVersion(t, "1.0.48")
	if checkUpdateAvailable(t.TempDir()).Available {
		t.Fatal("a directory without .aether was prompted to update Aether")
	}
}

// The marker must survive where update's own cleanup cannot prune it:
// .aether/version.json is a managed hub artifact and gets deleted.
func TestInstalledVersionMarkerIsNotAManagedHubPath(t *testing.T) {
	if isManagedAetherSystemPath(installedVersionMarkerRel) {
		t.Fatalf("marker path %q is treated as a managed hub artifact and would be pruned by update", installedVersionMarkerRel)
	}
	if !strings.HasPrefix(installedVersionMarkerRel, "data/") {
		t.Fatalf("marker must live under .aether/data/ to survive update cleanup, got %q", installedVersionMarkerRel)
	}
}

// The warning must reach `aether status`, which is where users look.
func TestStatusWarningsIncludeUpdateAvailable(t *testing.T) {
	saveGlobals(t)
	withHubVersion(t, "1.0.48")
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	warnGoal := "update prompt check"
	repo := repoRootFromStore(s)
	if repo == "" {
		t.Fatal("could not derive repo root from store")
	}
	if err := writeInstalledVersionMarker(repo, "1.0.25"); err != nil {
		t.Fatalf("stamp repo: %v", err)
	}

	warnings := computeWarnings(colony.ColonyState{Goal: &warnGoal}, s)
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "Aether update available") {
			found = true
		}
	}
	if !found {
		t.Fatalf("status warnings did not surface the available update: %v", warnings)
	}
}

// The Aether source checkout publishes the hub, so telling it to sync back
// from its own output would be circular noise.
func TestAetherSourceCheckoutIsNeverPrompted(t *testing.T) {
	withHubVersion(t, "1.0.48")
	repo := aetherRepo(t)
	if err := os.MkdirAll(filepath.Join(repo, "cmd", "aether"), 0755); err != nil {
		t.Fatalf("mkdir cmd/aether: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "cmd", "aether", "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "go.mod"), []byte("module github.com/calcosmic/Aether\n\ngo 1.26.5\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if checkUpdateAvailable(repo).Available {
		t.Fatal("the Aether source checkout was told to update from the hub it publishes")
	}
}
