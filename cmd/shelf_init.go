package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
	"github.com/spf13/cobra"
)

var shelfPromoteBatchCmd = &cobra.Command{
	Use:   "shelf-promote-batch",
	Short: "Promote multiple shelf entries by ID",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		idsRaw, _ := cmd.Flags().GetString("ids")
		colonyGoal, _ := cmd.Flags().GetString("colony")
		if idsRaw == "" {
			outputError(1, "flag --ids is required", nil)
			return nil
		}
		if colonyGoal == "" {
			outputError(1, "flag --colony is required", nil)
			return nil
		}

		ids := strings.Split(idsRaw, ",")
		var promoted []string
		var failed []string
		for _, id := range ids {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			if err := promoteShelfEntry(store, id, colonyGoal); err != nil {
				failed = append(failed, id)
			} else {
				promoted = append(promoted, id)
			}
		}

		outputOK(map[string]interface{}{
			"promoted": promoted,
			"failed":   failed,
			"count":    len(promoted),
		})
		return nil
	},
}

var shelfDismissBatchCmd = &cobra.Command{
	Use:   "shelf-dismiss-batch",
	Short: "Dismiss multiple shelf entries by ID",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		idsRaw, _ := cmd.Flags().GetString("ids")
		if idsRaw == "" {
			outputError(1, "flag --ids is required", nil)
			return nil
		}

		ids := strings.Split(idsRaw, ",")
		var dismissed []string
		var failed []string
		for _, id := range ids {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			if err := dismissShelfEntry(store, id); err != nil {
				failed = append(failed, id)
			} else {
				dismissed = append(dismissed, id)
			}
		}

		outputOK(map[string]interface{}{
			"dismissed": dismissed,
			"failed":    failed,
			"count":     len(dismissed),
		})
		return nil
	},
}

func loadActiveShelf(s *storage.Store) ([]colony.ShelfEntry, error) {
	sf, err := readShelfFile(s)
	if err != nil {
		return nil, err
	}
	var active []colony.ShelfEntry
	for _, e := range sf.Entries {
		if e.Status == colony.ShelfShelved {
			active = append(active, e)
		}
	}
	sort.Slice(active, func(i, j int) bool {
		return active[i].CreatedAt > active[j].CreatedAt
	})
	return active, nil
}

func promoteShelfEntry(s *storage.Store, id string, colonyGoal string) error {
	sf, err := readShelfFile(s)
	if err != nil {
		return err
	}
	found := false
	for i := range sf.Entries {
		if sf.Entries[i].ID == id {
			sf.Entries[i].Status = colony.ShelfPromoted
			sf.Entries[i].PromotedTo = colonyGoal
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("entry %q not found", id)
	}
	return writeShelfFile(s, sf)
}

func dismissShelfEntry(s *storage.Store, id string) error {
	sf, err := readShelfFile(s)
	if err != nil {
		return err
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
		return fmt.Errorf("entry %q not found", id)
	}
	return writeShelfFile(s, sf)
}

func shelfEntryToTodo(entry colony.ShelfEntry) string {
	return fmt.Sprintf("[shelf:%s] %s", entry.Category, entry.Text)
}

func formatShelfForInit(entries []colony.ShelfEntry) string {
	if len(entries) == 0 {
		return "No shelf entries."
	}

	// Group by category
	byCat := make(map[colony.ShelfCategory][]colony.ShelfEntry)
	for _, e := range entries {
		byCat[e.Category] = append(byCat[e.Category], e)
	}

	var b strings.Builder
	n := 1
	for _, cat := range []colony.ShelfCategory{
		colony.ShelfCategoryRedirect,
		colony.ShelfCategoryInstinct,
		colony.ShelfCategoryPheromone,
		colony.ShelfCategoryUserNote,
	} {
		group := byCat[cat]
		if len(group) == 0 {
			continue
		}
		for _, e := range group {
			phase := e.SourcePhase
			if phase == 0 {
				phaseStr := "unknown"
				_ = phaseStr
				b.WriteString(fmt.Sprintf("%d. [%s] %s\n", n, cat, e.Text))
			} else {
				b.WriteString(fmt.Sprintf("%d. [%s] %s (from phase %d)\n", n, cat, e.Text, phase))
			}
			n++
		}
	}
	return b.String()
}

func init() {
	shelfPromoteBatchCmd.Flags().String("ids", "", "Comma-separated entry IDs (required)")
	shelfPromoteBatchCmd.Flags().String("colony", "", "Colony goal for promotion (required)")
	shelfDismissBatchCmd.Flags().String("ids", "", "Comma-separated entry IDs (required)")

	rootCmd.AddCommand(shelfPromoteBatchCmd)
	rootCmd.AddCommand(shelfDismissBatchCmd)
}
