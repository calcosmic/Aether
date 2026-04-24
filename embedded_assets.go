package aetherassets

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

// installAssets contains the shipped companion files needed by `aether install`.
//
//go:embed all:.claude/commands/ant all:.claude/agents/ant all:.opencode/commands/ant all:.opencode/agents .opencode/opencode.json all:.codex all:.aether/agents-claude all:.aether/agents-codex all:.aether/commands all:.aether/docs all:.aether/exchange all:.aether/rules all:.aether/schemas all:.aether/skills all:.aether/skills-codex all:.aether/templates .aether/ts/narrator.ts .aether/ts/package-lock.json .aether/ts/package.json .aether/ts/tsconfig.json all:.aether/utils .aether/workers.md
var installAssets embed.FS

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
