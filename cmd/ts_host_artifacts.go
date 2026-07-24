package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	tsHostSourceRelDir = ".aether/ts-host"
	tsHostHubRelDir    = "system/ts-host"
	tsHostEntryRelPath = "dist/host.js"
)

func tsHostSourceDir(root string) string {
	return filepath.Join(root, filepath.FromSlash(tsHostSourceRelDir))
}

func tsHostHubDir(hubDir string) string {
	return filepath.Join(hubDir, filepath.FromSlash(tsHostHubRelDir))
}

func tsHostRepoDir(repoDir string) string {
	return tsHostSourceDir(repoDir)
}

func validateTsHostSourceArtifacts(sourceRoot string) error {
	return validateTsHostArtifactSet(tsHostSourceDir(sourceRoot))
}

func validateTsHostHubArtifacts(hubDir string) error {
	return validateTsHostArtifactSet(tsHostHubDir(hubDir))
}

func validateTsHostRepoArtifacts(repoDir string) error {
	return validateTsHostArtifactSet(tsHostRepoDir(repoDir))
}

func validateTsHostArtifactSet(tsHostDir string) error {
	for _, rel := range []string{"package.json", "package-lock.json", tsHostEntryRelPath} {
		if err := requireTsHostFile(tsHostDir, rel); err != nil {
			return err
		}
	}
	return nil
}

func requireTsHostFile(tsHostDir, rel string) error {
	path := filepath.Join(tsHostDir, filepath.FromSlash(rel))
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("required TS host artifact missing: %s", path)
		}
		return fmt.Errorf("stat TS host artifact %s: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("required TS host artifact is a directory: %s", path)
	}
	return nil
}

func tsHostHasRuntimeDependencies(tsHostDir string) (bool, error) {
	raw, err := os.ReadFile(filepath.Join(tsHostDir, "package.json"))
	if err != nil {
		return false, fmt.Errorf("read TS host package.json: %w", err)
	}
	var manifest struct {
		Dependencies         map[string]interface{} `json:"dependencies"`
		OptionalDependencies map[string]interface{} `json:"optionalDependencies"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return false, fmt.Errorf("parse TS host package.json: %w", err)
	}
	return len(manifest.Dependencies) > 0 || len(manifest.OptionalDependencies) > 0, nil
}
