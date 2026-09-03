package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// surveyStaleCommitThreshold is the number of commits after which a changed
// source revision also receives the explicit expired-revision reason.
const surveyStaleCommitThreshold = 25

const (
	territorySnapshotSchemaVersion = "territory-snapshot/v1"
	territorySnapshotRelativePath  = ".aether/data/survey/territory-snapshot.json"
)

// SurveyFreshnessReasonCode is a stable, machine-readable explanation for a
// territory classification. Presentation layers may translate these values,
// but lifecycle decisions must consume the codes rather than prose.
type SurveyFreshnessReasonCode string

const (
	surveyReasonSnapshotMissing          SurveyFreshnessReasonCode = "snapshot_metadata_missing"
	surveyReasonSnapshotMalformed        SurveyFreshnessReasonCode = "snapshot_metadata_malformed"
	surveyReasonSnapshotUnsafe           SurveyFreshnessReasonCode = "snapshot_metadata_unsafe"
	surveyReasonSnapshotDigestMismatch   SurveyFreshnessReasonCode = "snapshot_digest_mismatch"
	surveyReasonSchemaMismatch           SurveyFreshnessReasonCode = "schema_version_mismatch"
	surveyReasonRepositoryUnavailable    SurveyFreshnessReasonCode = "repository_unavailable"
	surveyReasonRepositoryRootChanged    SurveyFreshnessReasonCode = "repository_root_changed"
	surveyReasonRepositoryIDChanged      SurveyFreshnessReasonCode = "repository_identity_changed"
	surveyReasonSourceRevisionChanged    SurveyFreshnessReasonCode = "source_revision_changed"
	surveyReasonSourceRevisionExpired    SurveyFreshnessReasonCode = "source_revision_expired"
	surveyReasonGeneratedAtFuture        SurveyFreshnessReasonCode = "generated_at_in_future"
	surveyReasonArtifactMissing          SurveyFreshnessReasonCode = "artifact_missing"
	surveyReasonArtifactUnsafe           SurveyFreshnessReasonCode = "artifact_unsafe"
	surveyReasonArtifactReadUnavailable  SurveyFreshnessReasonCode = "artifact_read_unavailable"
	surveyReasonArtifactDigestMissing    SurveyFreshnessReasonCode = "artifact_digest_missing"
	surveyReasonArtifactDigestMismatch   SurveyFreshnessReasonCode = "artifact_digest_mismatch"
	surveyReasonArtifactDigestUnexpected SurveyFreshnessReasonCode = "artifact_digest_unexpected"
)

// SurveyFreshnessAge records both wall-clock and repository age. The commit
// count is meaningful only when CommitCountAvailable is true.
type SurveyFreshnessAge struct {
	Duration             time.Duration `json:"duration"`
	Commits              int           `json:"commits"`
	CommitCountAvailable bool          `json:"commit_count_available"`
}

// SurveyFreshnessResult is the typed territory fact consumed by lifecycle
// code. It deliberately carries the immutable snapshot evidence alongside the
// enum so callers never need to re-read or infer freshness from display text.
type SurveyFreshnessResult struct {
	Freshness          colony.SurveyFreshness      `json:"freshness"`
	SchemaVersion      string                      `json:"schema_version,omitempty"`
	SnapshotID         string                      `json:"snapshot_id,omitempty"`
	RepositoryIdentity string                      `json:"repository_identity,omitempty"`
	RepositoryRoot     string                      `json:"repository_root,omitempty"`
	SourceRevision     string                      `json:"source_revision,omitempty"`
	GeneratedAt        time.Time                   `json:"generated_at,omitempty"`
	ArtifactDigests    map[string]string           `json:"artifact_digests,omitempty"`
	Age                SurveyFreshnessAge          `json:"age"`
	ReasonCodes        []SurveyFreshnessReasonCode `json:"reason_codes,omitempty"`
	EvidencePaths      []string                    `json:"evidence_paths,omitempty"`
	SourceError        string                      `json:"source_error,omitempty"`
	Refreshed          bool                        `json:"refreshed,omitempty"`
}

// OutcomeLabel is the only UI label mapping for territory freshness.
func (r SurveyFreshnessResult) OutcomeLabel() string {
	if r.Refreshed && r.Freshness == colony.SurveyFreshnessFresh {
		return "Refreshed"
	}
	switch r.Freshness {
	case colony.SurveyFreshnessFresh:
		return "Fresh"
	case colony.SurveyFreshnessMissing, colony.SurveyFreshnessStale:
		return "Stale—refresh required"
	case colony.SurveyFreshnessUnavailable:
		return "Unavailable"
	default:
		return "Unavailable"
	}
}

// territorySnapshotMetadata is written only after a complete survey has been
// validated. SnapshotID authenticates every other field, including the exact
// digest set for all required survey artifacts.
type territorySnapshotMetadata struct {
	SchemaVersion      string            `json:"schema_version"`
	SnapshotID         string            `json:"snapshot_id"`
	RepositoryIdentity string            `json:"repository_identity"`
	RepositoryRoot     string            `json:"repository_root"`
	SourceRevision     string            `json:"source_revision"`
	GeneratedAt        time.Time         `json:"generated_at"`
	ArtifactDigests    map[string]string `json:"artifact_digests"`
}

func computeTerritorySnapshotID(snapshot territorySnapshotMetadata) string {
	snapshot.SnapshotID = ""
	payload, _ := json.Marshal(snapshot)
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

// classifySurveyFreshness is a read-only projection. It never repairs files,
// refreshes timestamps, or writes state: missing and stale results tell the
// lifecycle that work is needed, while unavailable is an explicit hard stop.
func classifySurveyFreshness(root string, now time.Time) SurveyFreshnessResult {
	result := SurveyFreshnessResult{Freshness: colony.SurveyFreshnessUnavailable}
	canonicalRoot, err := canonicalTerritoryRoot(root)
	if err != nil {
		return unavailableSurveyResult(result, surveyReasonRepositoryUnavailable, root, err)
	}

	metadataPath := filepath.Join(canonicalRoot, territorySnapshotRelativePath)
	missing := make([]string, 0)
	for _, rel := range append([]string{territorySnapshotRelativePath}, requiredTerritoryArtifactPaths()...) {
		path := filepath.Join(canonicalRoot, rel)
		info, statErr := os.Lstat(path)
		if os.IsNotExist(statErr) {
			missing = append(missing, filepath.ToSlash(rel))
			continue
		}
		if statErr != nil {
			reason := surveyReasonArtifactReadUnavailable
			if rel == territorySnapshotRelativePath {
				reason = surveyReasonSnapshotMalformed
			}
			return unavailableSurveyResult(result, reason, filepath.ToSlash(rel), statErr)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			reason := surveyReasonArtifactUnsafe
			if rel == territorySnapshotRelativePath {
				reason = surveyReasonSnapshotUnsafe
			}
			return unavailableSurveyResult(result, reason, filepath.ToSlash(rel), fmt.Errorf("evidence must be a regular file"))
		}
	}
	if len(missing) > 0 {
		result.Freshness = colony.SurveyFreshnessMissing
		result.EvidencePaths = missing
		result.ReasonCodes = []SurveyFreshnessReasonCode{surveyReasonArtifactMissing}
		for _, path := range missing {
			if path == filepath.ToSlash(territorySnapshotRelativePath) {
				result.ReasonCodes = append(result.ReasonCodes, surveyReasonSnapshotMissing)
				break
			}
		}
		normalizeSurveyResult(&result)
		return result
	}

	payload, err := os.ReadFile(metadataPath)
	if err != nil {
		return unavailableSurveyResult(result, surveyReasonSnapshotMalformed, filepath.ToSlash(territorySnapshotRelativePath), err)
	}
	var snapshot territorySnapshotMetadata
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&snapshot); err != nil {
		return unavailableSurveyResult(result, surveyReasonSnapshotMalformed, filepath.ToSlash(territorySnapshotRelativePath), err)
	}
	if snapshot.SnapshotID == "" || snapshot.RepositoryIdentity == "" || snapshot.RepositoryRoot == "" ||
		snapshot.SourceRevision == "" || snapshot.GeneratedAt.IsZero() || snapshot.ArtifactDigests == nil {
		return unavailableSurveyResult(result, surveyReasonSnapshotMalformed, filepath.ToSlash(territorySnapshotRelativePath), fmt.Errorf("snapshot metadata is incomplete"))
	}
	if !validGitRevision(snapshot.SourceRevision) {
		return unavailableSurveyResult(result, surveyReasonSnapshotMalformed, filepath.ToSlash(territorySnapshotRelativePath), fmt.Errorf("source revision is not a full git object ID"))
	}
	if !validSHA256(snapshot.SnapshotID) || snapshot.SnapshotID != computeTerritorySnapshotID(snapshot) {
		return unavailableSurveyResult(result, surveyReasonSnapshotDigestMismatch, filepath.ToSlash(territorySnapshotRelativePath), fmt.Errorf("snapshot digest does not match metadata"))
	}

	result.SchemaVersion = snapshot.SchemaVersion
	result.SnapshotID = snapshot.SnapshotID
	result.RepositoryIdentity = snapshot.RepositoryIdentity
	result.RepositoryRoot = snapshot.RepositoryRoot
	result.SourceRevision = snapshot.SourceRevision
	result.GeneratedAt = snapshot.GeneratedAt.UTC()
	result.ArtifactDigests = cloneStringMap(snapshot.ArtifactDigests)
	result.EvidencePaths = []string{filepath.ToSlash(territorySnapshotRelativePath)}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	if snapshot.GeneratedAt.After(now) {
		return unavailableSurveyResult(result, surveyReasonGeneratedAtFuture, filepath.ToSlash(territorySnapshotRelativePath), fmt.Errorf("snapshot was generated in the future"))
	}
	result.Age.Duration = now.Sub(snapshot.GeneratedAt.UTC())

	staleReasons := make([]SurveyFreshnessReasonCode, 0)
	if snapshot.SchemaVersion != territorySnapshotSchemaVersion {
		staleReasons = append(staleReasons, surveyReasonSchemaMismatch)
	}
	if filepath.Clean(snapshot.RepositoryRoot) != canonicalRoot {
		staleReasons = append(staleReasons, surveyReasonRepositoryRootChanged)
	}
	if snapshot.RepositoryIdentity != stableRepoIdentity(canonicalRoot) {
		staleReasons = append(staleReasons, surveyReasonRepositoryIDChanged)
	}

	currentRevision, err := currentTerritoryRevision(canonicalRoot)
	if err != nil {
		return unavailableSurveyResult(result, surveyReasonRepositoryUnavailable, canonicalRoot, err)
	}
	if currentRevision != snapshot.SourceRevision {
		staleReasons = append(staleReasons, surveyReasonSourceRevisionChanged)
		count, ok := commitsSinceRevision(canonicalRoot, snapshot.SourceRevision)
		if ok {
			result.Age.Commits = count
			result.Age.CommitCountAvailable = true
			if count >= surveyStaleCommitThreshold {
				staleReasons = append(staleReasons, surveyReasonSourceRevisionExpired)
			}
		} else {
			staleReasons = append(staleReasons, surveyReasonSourceRevisionExpired)
		}
	} else {
		result.Age.CommitCountAvailable = true
	}

	expected := requiredTerritoryArtifactPaths()
	expectedSet := make(map[string]struct{}, len(expected))
	for _, rel := range expected {
		rel = filepath.ToSlash(rel)
		expectedSet[rel] = struct{}{}
		result.EvidencePaths = append(result.EvidencePaths, rel)
		wantDigest, ok := snapshot.ArtifactDigests[rel]
		if !ok || !validSHA256(wantDigest) {
			staleReasons = append(staleReasons, surveyReasonArtifactDigestMissing)
			continue
		}
		body, readErr := os.ReadFile(filepath.Join(canonicalRoot, filepath.FromSlash(rel)))
		if readErr != nil {
			return unavailableSurveyResult(result, surveyReasonArtifactReadUnavailable, rel, readErr)
		}
		actual := sha256.Sum256(body)
		if !strings.EqualFold(wantDigest, hex.EncodeToString(actual[:])) {
			staleReasons = append(staleReasons, surveyReasonArtifactDigestMismatch)
		}
	}
	for rel := range snapshot.ArtifactDigests {
		if _, ok := expectedSet[filepath.ToSlash(rel)]; !ok {
			staleReasons = append(staleReasons, surveyReasonArtifactDigestUnexpected)
		}
	}

	if len(staleReasons) > 0 {
		result.Freshness = colony.SurveyFreshnessStale
		result.ReasonCodes = staleReasons
	} else {
		result.Freshness = colony.SurveyFreshnessFresh
	}
	normalizeSurveyResult(&result)
	return result
}

func canonicalTerritoryRoot(root string) (string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return "", fmt.Errorf("repository root is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	return filepath.Clean(resolved), nil
}

func requiredTerritoryArtifactPaths() []string {
	artifacts := requiredSurveyArtifacts()
	paths := make([]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		paths = append(paths, filepath.ToSlash(filepath.Join(".aether", "data", "survey", artifact.Name)))
	}
	return paths
}

func unavailableSurveyResult(result SurveyFreshnessResult, reason SurveyFreshnessReasonCode, evidence string, err error) SurveyFreshnessResult {
	result.Freshness = colony.SurveyFreshnessUnavailable
	result.ReasonCodes = append(result.ReasonCodes, reason)
	if strings.TrimSpace(evidence) != "" {
		result.EvidencePaths = append(result.EvidencePaths, filepath.ToSlash(evidence))
	}
	if err != nil {
		result.SourceError = err.Error()
	}
	normalizeSurveyResult(&result)
	return result
}

func normalizeSurveyResult(result *SurveyFreshnessResult) {
	result.ReasonCodes = uniqueSortedSurveyReasons(result.ReasonCodes)
	result.EvidencePaths = uniqueSortedTerritoryStrings(result.EvidencePaths)
}

func uniqueSortedSurveyReasons(values []SurveyFreshnessReasonCode) []SurveyFreshnessReasonCode {
	seen := make(map[SurveyFreshnessReasonCode]struct{}, len(values))
	out := make([]SurveyFreshnessReasonCode, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func uniqueSortedTerritoryStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func cloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	result := make(map[string]string, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func validSHA256(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validGitRevision(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func currentTerritoryRevision(root string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), GitTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "git", "-C", root, "rev-parse", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("read repository revision: %w", err)
	}
	revision := strings.TrimSpace(string(out))
	if !validGitRevision(revision) {
		return "", fmt.Errorf("repository returned an invalid revision")
	}
	return revision, nil
}

func commitsSinceRevision(root, revision string) (int, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), GitTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "git", "-C", root, "rev-list", "--count", revision+"..HEAD").Output()
	if err != nil {
		return 0, false
	}
	count, err := strconv.Atoi(strings.TrimSpace(string(out)))
	return count, err == nil
}

// surveyStalenessNotice is retained for compatibility with worker briefs. It
// renders the typed snapshot result when available and falls back to the old
// state timestamp only for colonies that predate territory-snapshot/v1.
func surveyStalenessNotice() string {
	if store == nil {
		return ""
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		return ""
	}
	root := filepath.Dir(filepath.Dir(store.BasePath()))
	result := classifySurveyFreshness(root, time.Now().UTC())

	var surveyedAt time.Time
	if !result.GeneratedAt.IsZero() {
		surveyedAt = result.GeneratedAt
	} else if state.TerritorySurveyed != nil && strings.TrimSpace(*state.TerritorySurveyed) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*state.TerritorySurveyed))
		if err != nil {
			return ""
		}
		surveyedAt = parsed
	}
	if surveyedAt.IsZero() {
		return "_Territory has never been surveyed. Run `/ant-colonize` to map the codebase before grounding on file locations or structure claims._\n\n"
	}

	count := result.Age.Commits
	ok := result.Age.CommitCountAvailable
	if result.SnapshotID == "" {
		count, ok = commitsSinceSurvey(root, surveyedAt)
	}
	if !ok {
		return fmt.Sprintf("_Territory surveyed %s. Commit count since survey unavailable._\n\n", surveyedAt.Format("2006-01-02"))
	}
	if count >= surveyStaleCommitThreshold {
		return fmt.Sprintf(
			"_STALE MAP WARNING: territory surveyed %d commits ago (%s). The codebase map may no longer match the tree — run `/ant-colonize` to refresh before trusting file locations or structure claims._\n\n",
			count, surveyedAt.Format("2006-01-02"),
		)
	}
	return fmt.Sprintf("_Territory mapped %d commit(s) ago (%s)._\n\n", count, surveyedAt.Format("2006-01-02"))
}

func commitsSinceSurvey(root string, since time.Time) (int, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), GitTimeout)
	defer cancel()

	sinceArg := "--since=" + since.UTC().Format(time.RFC3339)
	out, err := exec.CommandContext(ctx, "git", "-C", root, "rev-list", "--count", sinceArg, "HEAD").Output()
	if err != nil {
		return 0, false
	}

	count, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 0, false
	}
	return count, true
}
