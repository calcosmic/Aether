package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/storage"
)

// installedVersionMarkerRel records which hub version this repo's companion
// files were last synced from.
//
// It lives under .aether/data/ deliberately: .aether/version.json is in
// managedAetherSystemFiles, so a repo-local copy there is pruned as a stale
// hub artifact on the next update — the marker would delete itself.
const installedVersionMarkerRel = "data/installed-version.json"

type installedVersionMarker struct {
	Version   string `json:"version"`
	UpdatedAt string `json:"updated_at"`
}

func installedVersionMarkerPath(repoDir string) string {
	return filepath.Join(repoDir, ".aether", filepath.FromSlash(installedVersionMarkerRel))
}

// repoRootFromStore derives the workspace root from the store's base path
// (<root>/.aether/data), mirroring buildAttemptWorkspaceRoot.
func repoRootFromStore(s *storage.Store) string {
	if s == nil {
		return ""
	}
	base := strings.TrimSpace(s.BasePath())
	if base == "" {
		return ""
	}
	return filepath.Dir(filepath.Dir(base))
}

// writeInstalledVersionMarker stamps the repo with the hub version it just
// synced from. Failure is never fatal: a missing stamp costs a notification,
// not correctness.
func writeInstalledVersionMarker(repoDir, version string) error {
	version = normalizeVersion(strings.TrimSpace(version))
	if version == "" {
		return nil
	}
	path := installedVersionMarkerPath(repoDir)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(installedVersionMarker{
		Version:   version,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}

func readInstalledVersionMarker(repoDir string) (installedVersionMarker, bool) {
	data, err := os.ReadFile(installedVersionMarkerPath(repoDir))
	if err != nil {
		return installedVersionMarker{}, false
	}
	var marker installedVersionMarker
	if err := json.Unmarshal(data, &marker); err != nil {
		return installedVersionMarker{}, false
	}
	if strings.TrimSpace(marker.Version) == "" {
		return installedVersionMarker{}, false
	}
	return marker, true
}

// updateAvailability describes whether this repo is behind the shared hub.
type updateAvailability struct {
	Available      bool
	RepoVersion    string // "" when the repo has never been stamped
	HubVersion     string
	NeverStamped   bool
	RecommendedCmd string
}

// checkUpdateAvailable compares the repo's synced version against the hub.
//
// Repos silently rot: the hub advances every time Aether is published, but a
// repo keeps whatever companion files it last synced, with nothing telling the
// user they are behind. One repo on this machine sat on v1.0.25 while the hub
// was 23 releases ahead.
func checkUpdateAvailable(repoDir string) updateAvailability {
	// The Aether source checkout publishes the hub; telling it to sync back
	// from its own output is circular and always wrong.
	if isAetherSourceCheckout(repoDir) {
		return updateAvailability{}
	}

	hubVersion := normalizeVersion(strings.TrimSpace(readHubVersionAtPath(resolveHubPath())))
	if hubVersion == "" {
		return updateAvailability{}
	}

	marker, ok := readInstalledVersionMarker(repoDir)
	if !ok {
		// Never stamped. Only worth reporting when this repo actually uses
		// Aether — an un-onboarded directory should stay quiet.
		if _, err := os.Stat(filepath.Join(repoDir, ".aether")); err != nil {
			return updateAvailability{}
		}
		return updateAvailability{
			Available:      true,
			HubVersion:     hubVersion,
			NeverStamped:   true,
			RecommendedCmd: "aether update --force",
		}
	}

	// A version we cannot parse is treated as current: nagging forever about
	// an unreadable marker is worse than staying quiet.
	cmp, err := compareSemver(marker.Version, hubVersion)
	if err != nil || cmp >= 0 {
		return updateAvailability{RepoVersion: marker.Version, HubVersion: hubVersion}
	}
	return updateAvailability{
		Available:      true,
		RepoVersion:    marker.Version,
		HubVersion:     hubVersion,
		RecommendedCmd: "aether update --force",
	}
}

// renderRepoVersionTransition states, in the update's own output, what this
// repo moved from and to.
//
// The update visual reported hub and binary versions but never the repo's own,
// so a user could not tell whether they had actually been behind, by how much,
// or whether the run changed anything that mattered. That is the one question
// `/ant-update` exists to answer.
func renderRepoVersionTransition(previous, hubVersion string, dryRun bool) string {
	hubVersion = normalizeVersion(strings.TrimSpace(hubVersion))
	if hubVersion == "" || hubVersion == "unknown" {
		return ""
	}
	previous = normalizeVersion(strings.TrimSpace(previous))

	verb := "This repo"
	if dryRun {
		verb = "This repo would move"
	}

	if previous == "" {
		if dryRun {
			return fmt.Sprintf("%s to %s (no previous sync on record).\n", verb, hubVersion)
		}
		return fmt.Sprintf("%s is now on %s (first sync on record).\n", verb, hubVersion)
	}
	if cmp, err := compareSemver(previous, hubVersion); err == nil && cmp >= 0 {
		if dryRun {
			return fmt.Sprintf("This repo is already up to date at %s — nothing to fetch.\n", previous)
		}
		return fmt.Sprintf("This repo was already up to date at %s.\n", previous)
	}
	if dryRun {
		return fmt.Sprintf("An update is available: %s -> %s.\n", previous, hubVersion)
	}
	return fmt.Sprintf("Updated this repo: %s -> %s.\n", previous, hubVersion)
}

// renderUpdateAvailableWarning phrases the availability as a status warning.
// Returns "" when the repo is current.
func renderUpdateAvailableWarning(availability updateAvailability) string {
	if !availability.Available {
		return ""
	}
	if availability.NeverStamped {
		return fmt.Sprintf("Aether update available: the hub is at %s and this repo has no record of its last sync. Run `aether update --force` to refresh it.", availability.HubVersion)
	}
	return fmt.Sprintf("Aether update available: this repo is on %s, the hub is at %s. Run `aether update --force` to refresh it.",
		availability.RepoVersion, availability.HubVersion)
}
