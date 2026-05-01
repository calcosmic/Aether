package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

func copyShelfToChamber(s *storage.Store, chamberDir string) error {
	var sf colony.ShelfFile
	if err := s.LoadJSON("shelf.json", &sf); err != nil {
		// No shelf file — nothing to copy, not an error
		return nil
	}
	data, err := json.MarshalIndent(sf, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal shelf: %w", err)
	}
	if err := os.WriteFile(filepath.Join(chamberDir, "shelf.json"), append(data, '\n'), 0644); err != nil {
		return fmt.Errorf("write shelf to chamber: %w", err)
	}
	return nil
}

func shelfChamberSummary(s *storage.Store) string {
	var sf colony.ShelfFile
	if err := s.LoadJSON("shelf.json", &sf); err != nil {
		return "Shelved ideas: 0"
	}
	total := len(sf.Entries)
	if total == 0 {
		return "Shelved ideas: 0"
	}
	var promoted, dismissed, shelved int
	for _, e := range sf.Entries {
		switch e.Status {
		case colony.ShelfPromoted:
			promoted++
		case colony.ShelfDismissed:
			dismissed++
		case colony.ShelfShelved:
			shelved++
		}
	}
	return fmt.Sprintf("Shelved ideas: %d (%d promoted, %d dismissed, %d active)", total, promoted, dismissed, shelved)
}
