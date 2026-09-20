package aetherassets

import (
	"io/fs"
	"path"
	"testing"
)

func TestInstallAssetsExcludeFinderMetadata(t *testing.T) {
	if err := fs.WalkDir(installAssets, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path.Base(name) == ".DS_Store" {
			t.Errorf("Finder metadata shipped in install package: %s", name)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestInstallAssetsRetainRequiredHiddenMetadata(t *testing.T) {
	for _, name := range []string{
		".aether/skills/.index.json",
		".aether/skills/colony/.manifest.json",
		".aether/skills/domain/.manifest.json",
	} {
		data, err := installAssets.ReadFile(name)
		if err != nil || len(data) == 0 {
			t.Errorf("required asset %s missing or empty: %v", name, err)
		}
	}
}
