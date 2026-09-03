package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestSurveyFreshness199(t *testing.T) {
	now := time.Date(2026, time.September, 3, 18, 0, 0, 0, time.UTC)

	t.Run("missing required evidence is missing", func(t *testing.T) {
		root := initTerritoryFreshnessRepo199(t)

		got := classifySurveyFreshness(root, now)
		if got.Freshness != colony.SurveyFreshnessMissing {
			t.Fatalf("freshness = %q, want %q: %#v", got.Freshness, colony.SurveyFreshnessMissing, got)
		}
		assertSurveyReason199(t, got, surveyReasonArtifactMissing)
		if len(got.EvidencePaths) == 0 {
			t.Fatal("missing result must identify the absent evidence paths")
		}
	})

	t.Run("matching immutable snapshot is fresh", func(t *testing.T) {
		root := initTerritoryFreshnessRepo199(t)
		want := writeFreshTerritorySnapshot199(t, root, now.Add(-time.Hour))

		got := classifySurveyFreshness(root, now)
		if got.Freshness != colony.SurveyFreshnessFresh {
			t.Fatalf("freshness = %q, want %q: %#v", got.Freshness, colony.SurveyFreshnessFresh, got)
		}
		if got.SchemaVersion != territorySnapshotSchemaVersion || got.SnapshotID != want.SnapshotID {
			t.Fatalf("fresh result lost snapshot identity: %#v", got)
		}
		if got.RepositoryIdentity != want.RepositoryIdentity || got.RepositoryRoot != want.RepositoryRoot {
			t.Fatalf("fresh result lost repository binding: %#v", got)
		}
		if got.SourceRevision != want.SourceRevision || len(got.ArtifactDigests) != len(requiredTerritoryArtifactPaths199()) {
			t.Fatalf("fresh result lost revision or artifact digests: %#v", got)
		}
		if got.Age.Duration != time.Hour || got.Age.Commits != 0 {
			t.Fatalf("age = %#v, want one hour and zero commits", got.Age)
		}
		if len(got.ReasonCodes) != 0 {
			t.Fatalf("fresh result has reasons: %v", got.ReasonCodes)
		}
	})

	t.Run("changed inputs and revision are stale with stable reasons", func(t *testing.T) {
		t.Run("artifact digest changed", func(t *testing.T) {
			root := initTerritoryFreshnessRepo199(t)
			writeFreshTerritorySnapshot199(t, root, now.Add(-time.Hour))
			changed := filepath.Join(root, requiredTerritoryArtifactPaths199()[0])
			if err := os.WriteFile(changed, []byte("changed after the survey\n"), 0o644); err != nil {
				t.Fatalf("change surveyed artifact: %v", err)
			}

			got := classifySurveyFreshness(root, now)
			if got.Freshness != colony.SurveyFreshnessStale {
				t.Fatalf("freshness = %q, want stale: %#v", got.Freshness, got)
			}
			assertSurveyReason199(t, got, surveyReasonArtifactDigestMismatch)
		})

		t.Run("source revision changed", func(t *testing.T) {
			root := initTerritoryFreshnessRepo199(t)
			writeFreshTerritorySnapshot199(t, root, now.Add(-time.Hour))
			gitSurveyFreshness199(t, root, "commit", "--allow-empty", "-m", "repository changed")

			got := classifySurveyFreshness(root, now)
			if got.Freshness != colony.SurveyFreshnessStale {
				t.Fatalf("freshness = %q, want stale: %#v", got.Freshness, got)
			}
			assertSurveyReason199(t, got, surveyReasonSourceRevisionChanged)
			if got.Age.Commits != 1 {
				t.Fatalf("commit age = %d, want 1", got.Age.Commits)
			}
		})
	})

	t.Run("malformed evidence is unavailable and never fresh", func(t *testing.T) {
		root := initTerritoryFreshnessRepo199(t)
		writeFreshTerritorySnapshot199(t, root, now.Add(-time.Hour))
		path := filepath.Join(root, territorySnapshotRelativePath)
		if err := os.WriteFile(path, []byte("{not-json"), 0o644); err != nil {
			t.Fatalf("write malformed snapshot: %v", err)
		}

		got := classifySurveyFreshness(root, now)
		if got.Freshness != colony.SurveyFreshnessUnavailable {
			t.Fatalf("freshness = %q, want unavailable: %#v", got.Freshness, got)
		}
		assertSurveyReason199(t, got, surveyReasonSnapshotMalformed)
	})

	t.Run("fresh requires a verified digest for every declared artifact", func(t *testing.T) {
		root := initTerritoryFreshnessRepo199(t)
		snapshot := writeFreshTerritorySnapshot199(t, root, now.Add(-time.Hour))
		delete(snapshot.ArtifactDigests, requiredTerritoryArtifactPaths199()[0])
		snapshot.SnapshotID = computeTerritorySnapshotID(snapshot)
		writeTerritorySnapshotFixture199(t, root, snapshot)

		got := classifySurveyFreshness(root, now)
		if got.Freshness == colony.SurveyFreshnessFresh {
			t.Fatalf("incomplete digest set classified fresh: %#v", got)
		}
		if got.Freshness != colony.SurveyFreshnessStale {
			t.Fatalf("freshness = %q, want stale for incomplete digest set: %#v", got.Freshness, got)
		}
		assertSurveyReason199(t, got, surveyReasonArtifactDigestMissing)
	})

	t.Run("classification is zero write", func(t *testing.T) {
		root := initTerritoryFreshnessRepo199(t)
		writeFreshTerritorySnapshot199(t, root, now.Add(-time.Hour))
		before := snapshotTerritoryData199(t, root)

		got := classifySurveyFreshness(root, now)
		if got.Freshness != colony.SurveyFreshnessFresh {
			t.Fatalf("freshness = %q, want fresh: %#v", got.Freshness, got)
		}
		after := snapshotTerritoryData199(t, root)
		if !reflect.DeepEqual(after, before) {
			t.Fatalf("freshness classification mutated survey evidence\nbefore: %#v\nafter:  %#v", before, after)
		}
	})
}

type territoryDataFile199 struct {
	Path    string
	Mode    os.FileMode
	ModTime time.Time
	Digest  string
}

func initTerritoryFreshnessRepo199(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	gitSurveyFreshness199(t, root, "init")
	gitSurveyFreshness199(t, root, "config", "user.email", "test@aether.invalid")
	gitSurveyFreshness199(t, root, "config", "user.name", "Aether Test")
	gitSurveyFreshness199(t, root, "commit", "--allow-empty", "-m", "initial")
	return root
}

func gitSurveyFreshness199(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func writeFreshTerritorySnapshot199(t *testing.T, root string, generatedAt time.Time) territorySnapshotMetadata {
	t.Helper()
	digests := make(map[string]string, len(requiredTerritoryArtifactPaths199()))
	for _, rel := range requiredTerritoryArtifactPaths199() {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
		body := []byte("survey evidence for " + rel + "\n")
		if strings.HasSuffix(rel, ".json") {
			body = []byte(`{"artifact":"` + filepath.Base(rel) + `"}`)
		}
		if err := os.WriteFile(path, body, 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
		sum := sha256.Sum256(body)
		digests[rel] = hex.EncodeToString(sum[:])
	}
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("canonical root: %v", err)
	}
	snapshot := territorySnapshotMetadata{
		SchemaVersion:      territorySnapshotSchemaVersion,
		RepositoryIdentity: stableRepoIdentity(canonicalRoot),
		RepositoryRoot:     canonicalRoot,
		SourceRevision:     gitSurveyFreshness199(t, root, "rev-parse", "HEAD"),
		GeneratedAt:        generatedAt.UTC(),
		ArtifactDigests:    digests,
	}
	snapshot.SnapshotID = computeTerritorySnapshotID(snapshot)
	writeTerritorySnapshotFixture199(t, root, snapshot)
	return snapshot
}

func requiredTerritoryArtifactPaths199() []string {
	artifacts := requiredSurveyArtifacts()
	paths := make([]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		paths = append(paths, filepath.Join(".aether", "data", "survey", artifact.Name))
	}
	return paths
}

func writeTerritorySnapshotFixture199(t *testing.T, root string, snapshot territorySnapshotMetadata) {
	t.Helper()
	path := filepath.Join(root, territorySnapshotRelativePath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir snapshot directory: %v", err)
	}
	payload, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}
	payload = append(payload, '\n')
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}
}

func snapshotTerritoryData199(t *testing.T, root string) []territoryDataFile199 {
	t.Helper()
	base := filepath.Join(root, ".aether", "data")
	var files []territoryDataFile199
	err := filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		body, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		sum := sha256.Sum256(body)
		rel, relErr := filepath.Rel(base, path)
		if relErr != nil {
			return relErr
		}
		files = append(files, territoryDataFile199{
			Path:    filepath.ToSlash(rel),
			Mode:    info.Mode(),
			ModTime: info.ModTime(),
			Digest:  hex.EncodeToString(sum[:]),
		})
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot territory data: %v", err)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files
}

func assertSurveyReason199(t *testing.T, got SurveyFreshnessResult, want SurveyFreshnessReasonCode) {
	t.Helper()
	for _, reason := range got.ReasonCodes {
		if reason == want {
			return
		}
	}
	t.Fatalf("reasons = %v, want %q", got.ReasonCodes, want)
}
