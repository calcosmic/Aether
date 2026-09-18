package aetherassets

import (
	"embed"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
)

// installAssets contains the shipped companion files needed by `aether install`.
// Directory walks exclude Finder metadata and other dot/underscore files.
// Required hidden skill metadata is named explicitly below.
//
//go:embed .claude/commands/ant .claude/agents/ant .opencode/commands/ant .opencode/agents .opencode/opencode.json .codex .aether/commands .aether/docs .aether/exchange .aether/references .aether/rules .aether/schemas .aether/skills .aether/templates .aether/ts/dist/narrator.js .aether/ts/narrator.ts .aether/ts/package-lock.json .aether/ts/package.json .aether/ts/tsconfig.build.json .aether/ts/tsconfig.json .aether/utils .aether/workers.md
//go:embed .aether/ts-host/dist .aether/ts-host/package-lock.json .aether/ts-host/package.json
//go:embed .aether/version.json .aether/skills/.index.json .aether/skills/colony/.manifest.json .aether/skills/domain/.manifest.json .aether/commands/.gitkeep
var installAssets embed.FS

// SourceReleaseVersion identifies the source compiled into this executable,
// including builds made by go install without release linker flags. The same
// immutable metadata is materialized with the bundled installation package.
func SourceReleaseVersion() (string, error) {
	data, err := installAssets.ReadFile(".aether/version.json")
	if err != nil {
		return "", err
	}
	var metadata struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &metadata); err != nil {
		return "", err
	}
	return metadata.Version, nil
}

// MaterializeInstallPackage writes the embedded install assets into dest.
func MaterializeInstallPackage(dest string) error {
	return fs.WalkDir(installAssets, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "." {
			return nil
		}

		targetPath := filepath.Join(dest, filepath.FromSlash(path))
		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		data, readErr := installAssets.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}
		return os.WriteFile(targetPath, data, 0644)
	})
}
