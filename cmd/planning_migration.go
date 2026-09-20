package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

// legacyPlanningArtifactMaterial records how a pre-Phase-200 planning file may
// be used. Both values are authority-neutral: neither can approve a
// specification, accept a candidate, or prove that evidence is fresh.
type legacyPlanningArtifactMaterial string

const (
	legacyPlanningEvidenceMaterial legacyPlanningArtifactMaterial = "legacy_evidence"
	legacyPlanningTimelineMaterial legacyPlanningArtifactMaterial = "legacy_timeline"
)

// legacyPlanningArtifactReference is a deterministic content index over one
// old planning artifact. It intentionally has no approval, freshness, semantic
// identity, or acceptance fields.
type legacyPlanningArtifactReference struct {
	Classification colony.PlanAcceptancePolicy    `json:"classification"`
	Material       legacyPlanningArtifactMaterial `json:"material"`
	RepositoryPath string                         `json:"repository_path"`
	ContentHash    string                         `json:"content_hash"`
}

type planningMigrationResult struct {
	State           colony.ColonyState
	LegacyArtifacts []legacyPlanningArtifactReference
	Changed         bool
}

// migratePlanningState classifies an executable pre-Phase-200 plan without
// changing its phases or historical revisions. Old planning files are indexed
// separately as non-authoritative material so their bytes remain inspectable
// without being mistaken for modern owner consent.
func migratePlanningState(root string, state colony.ColonyState) (planningMigrationResult, error) {
	result := planningMigrationResult{State: state}
	if result.State.Plan.AcceptancePolicy == "" && len(result.State.Plan.Phases) > 0 {
		if planHasCurrentAuthority(result.State.Plan) {
			return planningMigrationResult{}, fmt.Errorf("planning migration: current planning authority is partially populated without acceptance_policy")
		}
		result.State.Plan.AcceptancePolicy = colony.PlanAcceptanceLegacyUnbound
		result.Changed = true
	}

	normalized, err := normalizePlanningState(result.State)
	if err != nil {
		return planningMigrationResult{}, fmt.Errorf("planning migration: %w", err)
	}
	result.State = normalized

	if result.State.Plan.AcceptancePolicy != colony.PlanAcceptanceLegacyUnbound || len(result.State.Plan.Phases) == 0 {
		return result, nil
	}
	paths, err := discoverLegacyPlanningArtifactPaths(root)
	if err != nil {
		return planningMigrationResult{}, err
	}
	result.LegacyArtifacts, err = indexLegacyPlanningArtifacts(root, paths)
	if err != nil {
		return planningMigrationResult{}, err
	}
	return result, nil
}

func discoverLegacyPlanningArtifactPaths(root string) ([]string, error) {
	planningRoot := filepath.Join(root, ".aether", "data", "planning")
	info, err := os.Lstat(planningRoot)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("planning migration: inspect .aether/data/planning: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, fmt.Errorf("planning migration: .aether/data/planning must be a real directory")
	}

	paths := []string{}
	err = filepath.WalkDir(planningRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == planningRoot {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return fmt.Errorf("resolve legacy planning artifact path: %w", relErr)
		}
		rel = filepath.ToSlash(rel)
		if entry.IsDir() {
			return nil
		}
		if entry.Name() == ".fallback-marker" || strings.HasSuffix(entry.Name(), ".bak") {
			return nil
		}
		paths = append(paths, rel)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("planning migration: discover legacy artifacts: %w", err)
	}
	sort.Strings(paths)
	return paths, nil
}

// indexLegacyPlanningArtifacts validates every path before opening any file.
// Callers may therefore pass persisted legacy references without allowing an
// absolute path, ../ segment, or symlink to escape the planning data boundary.
func indexLegacyPlanningArtifacts(root string, paths []string) ([]legacyPlanningArtifactReference, error) {
	unique := make(map[string]struct{}, len(paths))
	cleaned := make([]string, 0, len(paths))
	for _, raw := range paths {
		rel, err := validateLegacyPlanningArtifactPath(root, raw)
		if err != nil {
			return nil, err
		}
		if _, exists := unique[rel]; exists {
			continue
		}
		unique[rel] = struct{}{}
		cleaned = append(cleaned, rel)
	}
	sort.Strings(cleaned)

	references := make([]legacyPlanningArtifactReference, 0, len(cleaned))
	for _, rel := range cleaned {
		path := filepath.Join(root, filepath.FromSlash(rel))
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("planning migration: read legacy artifact %s: %w", rel, err)
		}
		if strings.EqualFold(filepath.Ext(rel), ".json") && !json.Valid(data) {
			return nil, fmt.Errorf("planning migration: legacy artifact %s contains malformed JSON", rel)
		}
		sum := sha256.Sum256(data)
		references = append(references, legacyPlanningArtifactReference{
			Classification: colony.PlanAcceptanceLegacyUnbound,
			Material:       classifyLegacyPlanningArtifact(rel),
			RepositoryPath: rel,
			ContentHash:    hex.EncodeToString(sum[:]),
		})
	}
	return references, nil
}

func validateLegacyPlanningArtifactPath(root, raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("planning migration: legacy artifact path is required")
	}
	if filepath.IsAbs(raw) {
		return "", fmt.Errorf("planning migration: legacy artifact must be repository-relative: %s", raw)
	}
	clean := filepath.Clean(filepath.FromSlash(raw))
	cleanSlash := filepath.ToSlash(clean)
	const prefix = ".aether/data/planning/"
	if !strings.HasPrefix(cleanSlash, prefix) || cleanSlash == strings.TrimSuffix(prefix, "/") {
		return "", fmt.Errorf("planning migration: legacy artifact escapes .aether/data/planning: %s", raw)
	}

	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("planning migration: resolve repository root: %w", err)
	}
	path := filepath.Join(root, clean)
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("planning migration: resolve legacy artifact %s: %w", cleanSlash, err)
	}
	realPlanningRoot, err := filepath.EvalSymlinks(filepath.Join(root, ".aether", "data", "planning"))
	if err != nil {
		return "", fmt.Errorf("planning migration: resolve .aether/data/planning: %w", err)
	}
	if !pathContainedBy(realRoot, realPlanningRoot) {
		return "", fmt.Errorf("planning migration: .aether/data/planning resolves outside repository")
	}
	if !pathContainedBy(realPlanningRoot, realPath) || realPath == realPlanningRoot {
		return "", fmt.Errorf("planning migration: legacy artifact resolves outside .aether/data/planning: %s", cleanSlash)
	}
	if !pathContainedBy(realRoot, realPath) {
		return "", fmt.Errorf("planning migration: legacy artifact resolves outside repository: %s", cleanSlash)
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", fmt.Errorf("planning migration: inspect legacy artifact %s: %w", cleanSlash, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", fmt.Errorf("planning migration: legacy artifact must be a regular file: %s", cleanSlash)
	}
	return cleanSlash, nil
}

func pathContainedBy(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func classifyLegacyPlanningArtifact(rel string) legacyPlanningArtifactMaterial {
	rel = filepath.ToSlash(filepath.Clean(filepath.FromSlash(rel)))
	if rel == ".aether/data/planning/iteration-state.json" || strings.Contains(rel, "/iterations/") {
		return legacyPlanningTimelineMaterial
	}
	return legacyPlanningEvidenceMaterial
}
