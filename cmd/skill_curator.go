package cmd

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/calcosmic/Aether/pkg/learn"
	"github.com/calcosmic/Aether/pkg/storage"
	"github.com/spf13/cobra"
)

// resolveColonyDBPath returns the path to colony.db within the Aether data directory.
func resolveColonyDBPath() string {
	return filepath.Join(store.BasePath(), "colony.db")
}

// resolveSkillBaseDir returns the project root directory (containing .aether/).
func resolveSkillBaseDir() string {
	return storage.ResolveAetherRoot(context.Background())
}

var skillCuratorRunCmd = &cobra.Command{
	Use:   "skill-curator-run",
	Short: "Run the Keeper Curator to transition unused skills through lifecycle stages",
	Long:  "Transitions active skills to stale after 14 days unused, stale to archived after 28 days. Pinned skills are immune.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		dbPath := resolveColonyDBPath()
		sqliteStore, err := learn.NewSQLiteColonyStore(dbPath)
		if err != nil {
			outputError(2, fmt.Sprintf("failed to open database: %v", err), nil)
			return nil
		}
		defer sqliteStore.Close()

		curator := learn.NewCurator(sqliteStore.DB(), resolveSkillBaseDir())
		count, err := curator.RunTransitions()
		if err != nil {
			outputError(2, fmt.Sprintf("curator run failed: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"transitions": count,
			"message":     fmt.Sprintf("Transitioned %d skill(s) through lifecycle stages", count),
		})
		return nil
	},
}

var skillRecoverCmd = &cobra.Command{
	Use:   "skill-recover [name]",
	Short: "Recover an archived skill back to active",
	Long:  "Archived skills are always recoverable. This moves the skill back to active stage and resets its usage timestamps.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		dbPath := resolveColonyDBPath()
		sqliteStore, err := learn.NewSQLiteColonyStore(dbPath)
		if err != nil {
			outputError(2, fmt.Sprintf("failed to open database: %v", err), nil)
			return nil
		}
		defer sqliteStore.Close()

		curator := learn.NewCurator(sqliteStore.DB(), resolveSkillBaseDir())
		if err := curator.RecoverSkill(name); err != nil {
			outputError(2, fmt.Sprintf("failed to recover skill: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"recovered": true,
			"name":      name,
			"stage":     "active",
		})
		return nil
	},
}

func init() {
	rootCmd.AddCommand(skillCuratorRunCmd)
	rootCmd.AddCommand(skillRecoverCmd)
}
