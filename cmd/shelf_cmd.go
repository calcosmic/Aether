package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
	"github.com/spf13/cobra"
)

var (
	shelfStatusFilter string
	shelfListJSON     bool
)

var shelfListCmd = &cobra.Command{
	Use:     "shelf-list",
	Short:   "List all shelf entries",
	Args:    cobra.NoArgs,
	Aliases: []string{"shelf"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		sf, err := readShelfFile(store)
		if err != nil {
			outputError(1, fmt.Sprintf("failed to read shelf: %v", err), nil)
			return nil
		}

		filtered := filterShelfEntries(sf.Entries, shelfStatusFilter)
		result := map[string]interface{}{
			"entries": filtered,
			"total":   len(filtered),
			"status":  shelfStatusFilter,
		}

		if shelfListJSON {
			outputOK(result)
			return nil
		}

		outputWorkflow(result, renderShelfListVisual(result))
		return nil
	},
}

var shelfAddCmd = &cobra.Command{
	Use:   "shelf-add",
	Short: "Add a new shelf entry",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		text, _ := cmd.Flags().GetString("text")
		category, _ := cmd.Flags().GetString("category")
		source, _ := cmd.Flags().GetString("source")
		tags, _ := cmd.Flags().GetStringArray("tag")

		if text == "" {
			outputError(1, "flag --text is required", nil)
			return nil
		}

		cat := colony.ShelfCategory(category)
		switch cat {
		case colony.ShelfCategoryInstinct, colony.ShelfCategoryPheromone, colony.ShelfCategoryUserNote, colony.ShelfCategoryRedirect:
		default:
			outputError(1, fmt.Sprintf("invalid category %q: must be instinct, pheromone, user-note, or redirect", category), nil)
			return nil
		}

		entry := colony.ShelfEntry{
			ID:        generateShelfID(),
			Text:      text,
			Source:    firstNonEmpty(source, "cli"),
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
			Category:  cat,
			Status:    colony.ShelfShelved,
			Tags:      tags,
		}

		sf, err := readShelfFile(store)
		if err != nil {
			outputError(1, fmt.Sprintf("failed to read shelf: %v", err), nil)
			return nil
		}

		sf.Entries = append(sf.Entries, entry)
		sf.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

		if err := writeShelfFile(store, sf); err != nil {
			outputError(2, fmt.Sprintf("failed to save shelf: %v", err), nil)
			return nil
		}

		result := map[string]interface{}{
			"created": true,
			"entry":   entry,
			"total":   len(sf.Entries),
		}
		outputWorkflow(result, renderShelfActionVisual("shelf-add", "Shelf Entry Added", result))
		return nil
	},
}

var shelfPromoteCmd = &cobra.Command{
	Use:   "shelf-promote",
	Short: "Promote a shelf entry by ID",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		id := mustGetString(cmd, "id")
		if id == "" {
			return nil
		}
		to := mustGetString(cmd, "to")
		if to == "" {
			return nil
		}

		sf, err := readShelfFile(store)
		if err != nil {
			outputError(1, fmt.Sprintf("failed to read shelf: %v", err), nil)
			return nil
		}

		found := false
		for i := range sf.Entries {
			if sf.Entries[i].ID == id {
				sf.Entries[i].Status = colony.ShelfPromoted
				sf.Entries[i].PromotedTo = to
				found = true
				break
			}
		}

		if !found {
			outputError(1, fmt.Sprintf("entry %q not found", id), nil)
			return nil
		}

		sf.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		if err := writeShelfFile(store, sf); err != nil {
			outputError(2, fmt.Sprintf("failed to save shelf: %v", err), nil)
			return nil
		}

		result := map[string]interface{}{
			"promoted": true,
			"id":       id,
			"to":       to,
		}
		outputWorkflow(result, renderShelfActionVisual("shelf-promote", "Shelf Entry Promoted", result))
		return nil
	},
}

var shelfDismissCmd = &cobra.Command{
	Use:   "shelf-dismiss",
	Short: "Dismiss a shelf entry by ID",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		id := mustGetString(cmd, "id")
		if id == "" {
			return nil
		}

		sf, err := readShelfFile(store)
		if err != nil {
			outputError(1, fmt.Sprintf("failed to read shelf: %v", err), nil)
			return nil
		}

		found := false
		for i := range sf.Entries {
			if sf.Entries[i].ID == id {
				sf.Entries[i].Status = colony.ShelfDismissed
				found = true
				break
			}
		}

		if !found {
			outputError(1, fmt.Sprintf("entry %q not found", id), nil)
			return nil
		}

		sf.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		if err := writeShelfFile(store, sf); err != nil {
			outputError(2, fmt.Sprintf("failed to save shelf: %v", err), nil)
			return nil
		}

		result := map[string]interface{}{
			"dismissed": true,
			"id":        id,
		}
		outputWorkflow(result, renderShelfActionVisual("shelf-dismiss", "Shelf Entry Dismissed", result))
		return nil
	},
}

func readShelfFile(s *storage.Store) (colony.ShelfFile, error) {
	var sf colony.ShelfFile
	if err := s.LoadJSON("shelf.json", &sf); err != nil {
		if isNotExist(err) {
			return colony.NewShelfFile(), nil
		}
		return colony.ShelfFile{}, err
	}
	if sf.Entries == nil {
		sf.Entries = []colony.ShelfEntry{}
	}
	return sf, nil
}

func writeShelfFile(s *storage.Store, sf colony.ShelfFile) error {
	return s.SaveJSON("shelf.json", sf)
}

func generateShelfID() string {
	rnd := make([]byte, 4)
	rand.Read(rnd)
	return fmt.Sprintf("shelf_%d_%s", time.Now().Unix(), hex.EncodeToString(rnd))
}

func filterShelfEntries(entries []colony.ShelfEntry, status string) []colony.ShelfEntry {
	if status == "" {
		return entries
	}
	var result []colony.ShelfEntry
	for _, e := range entries {
		if string(e.Status) == status {
			result = append(result, e)
		}
	}
	return result
}

func isNotExist(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "no such file") || strings.Contains(err.Error(), "not found")
}

func init() {
	shelfListCmd.Flags().StringVar(&shelfStatusFilter, "status", "", "Filter by status (shelved, promoted, dismissed)")
	shelfListCmd.Flags().BoolVar(&shelfListJSON, "json", false, "Output as JSON")

	shelfAddCmd.Flags().String("text", "", "Entry text (required)")
	shelfAddCmd.Flags().String("category", "user-note", "Category: instinct, pheromone, user-note, redirect")
	shelfAddCmd.Flags().String("source", "cli", "Entry source")
	shelfAddCmd.Flags().StringArray("tag", nil, "Tags (repeatable)")

	shelfPromoteCmd.Flags().String("id", "", "Entry ID to promote (required)")
	shelfPromoteCmd.Flags().String("to", "", "Promotion target / colony goal (required)")

	shelfDismissCmd.Flags().String("id", "", "Entry ID to dismiss (required)")

	rootCmd.AddCommand(shelfListCmd)
	rootCmd.AddCommand(shelfAddCmd)
	rootCmd.AddCommand(shelfPromoteCmd)
	rootCmd.AddCommand(shelfDismissCmd)
}
