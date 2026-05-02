package cmd

import (
	"fmt"
	"os"

	"github.com/calcosmic/Aether/pkg/learn"
	"github.com/spf13/cobra"
)

var skillCreateCmd = &cobra.Command{
	Use:   "skill-create [name]",
	Short: "Create a new repo-local pheromone skill",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		content, _ := cmd.Flags().GetString("content")
		pinned, _ := cmd.Flags().GetBool("pin")

		dbPath := resolveColonyDBPath()
		sqliteStore, err := learn.NewSQLiteColonyStore(dbPath)
		if err != nil {
			outputError(2, fmt.Sprintf("failed to open database: %v", err), nil)
			return nil
		}
		defer sqliteStore.Close()

		svc := learn.NewSkillService(sqliteStore.DB(), resolveSkillBaseDir())
		meta := learn.SkillMetadata{
			Name:        name,
			Stage:       learn.SkillStageActive,
			Pinned:      pinned,
			Confidence:  0.75,
			AutoCreated: false,
		}
		if err := svc.CreateSkill(meta, content); err != nil {
			outputError(2, fmt.Sprintf("failed to create skill: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"created": true,
			"name":    name,
			"pinned":  pinned,
		})
		return nil
	},
}

var skillPatchCmd = &cobra.Command{
	Use:   "skill-patch [name]",
	Short: "Update an existing skill's content",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		content, _ := cmd.Flags().GetString("content")

		dbPath := resolveColonyDBPath()
		sqliteStore, err := learn.NewSQLiteColonyStore(dbPath)
		if err != nil {
			outputError(2, fmt.Sprintf("database: %v", err), nil)
			return nil
		}
		defer sqliteStore.Close()

		svc := learn.NewSkillService(sqliteStore.DB(), resolveSkillBaseDir())
		if err := svc.PatchSkill(name, content); err != nil {
			outputError(2, fmt.Sprintf("failed to patch skill: %v", err), nil)
			return nil
		}
		outputOK(map[string]interface{}{"patched": true, "name": name})
		return nil
	},
}

var skillArchiveCmd = &cobra.Command{
	Use:   "skill-archive [name]",
	Short: "Archive a skill (recoverable, never deleted)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		dbPath := resolveColonyDBPath()
		sqliteStore, err := learn.NewSQLiteColonyStore(dbPath)
		if err != nil {
			outputError(2, fmt.Sprintf("database: %v", err), nil)
			return nil
		}
		defer sqliteStore.Close()

		svc := learn.NewSkillService(sqliteStore.DB(), resolveSkillBaseDir())
		if err := svc.ArchiveSkill(name); err != nil {
			outputError(2, fmt.Sprintf("failed to archive skill: %v", err), nil)
			return nil
		}
		outputOK(map[string]interface{}{"archived": true, "name": name})
		return nil
	},
}

var skillPinCmd = &cobra.Command{
	Use:   "skill-pin [name]",
	Short: "Pin a skill to prevent auto-transitions and agent writes",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		dbPath := resolveColonyDBPath()
		sqliteStore, err := learn.NewSQLiteColonyStore(dbPath)
		if err != nil {
			outputError(2, fmt.Sprintf("database: %v", err), nil)
			return nil
		}
		defer sqliteStore.Close()

		svc := learn.NewSkillService(sqliteStore.DB(), resolveSkillBaseDir())
		if err := svc.PinSkill(name); err != nil {
			outputError(2, fmt.Sprintf("failed to pin skill: %v", err), nil)
			return nil
		}
		outputOK(map[string]interface{}{"pinned": true, "name": name})
		return nil
	},
}

var skillListLifecycleCmd = &cobra.Command{
	Use:   "skill-list-lifecycle",
	Short: "List all skills with lifecycle metadata",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		stage, _ := cmd.Flags().GetString("stage")
		dbPath := resolveColonyDBPath()
		sqliteStore, err := learn.NewSQLiteColonyStore(dbPath)
		if err != nil {
			outputError(2, fmt.Sprintf("database: %v", err), nil)
			return nil
		}
		defer sqliteStore.Close()

		svc := learn.NewSkillService(sqliteStore.DB(), resolveSkillBaseDir())
		skills, err := svc.ListSkills(stage)
		if err != nil {
			outputError(2, fmt.Sprintf("failed to list skills: %v", err), nil)
			return nil
		}
		outputOK(map[string]interface{}{"skills": skills, "total": len(skills)})
		return nil
	},
}

var skillViewCmd = &cobra.Command{
	Use:   "skill-view [name]",
	Short: "View a skill's full content",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		dbPath := resolveColonyDBPath()
		sqliteStore, err := learn.NewSQLiteColonyStore(dbPath)
		if err != nil {
			outputError(2, fmt.Sprintf("database: %v", err), nil)
			return nil
		}
		defer sqliteStore.Close()

		svc := learn.NewSkillService(sqliteStore.DB(), resolveSkillBaseDir())
		meta, err := svc.GetSkill(name)
		if err != nil {
			outputError(2, fmt.Sprintf("failed to get skill: %v", err), nil)
			return nil
		}
		if meta == nil {
			outputError(2, fmt.Sprintf("skill %q not found", name), nil)
			return nil
		}

		content, err := os.ReadFile(meta.FilePath)
		if err != nil {
			outputError(2, fmt.Sprintf("failed to read skill file: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"metadata": meta,
			"content":  string(content),
		})
		return nil
	},
}

func init() {
	skillCreateCmd.Flags().String("content", "", "Skill content (markdown)")
	skillCreateCmd.Flags().Bool("pin", false, "Pin the skill immediately")
	skillPatchCmd.Flags().String("content", "", "New skill content")
	skillListLifecycleCmd.Flags().String("stage", "", "Filter by stage (active, stale, archived)")

	rootCmd.AddCommand(skillCreateCmd)
	rootCmd.AddCommand(skillPatchCmd)
	rootCmd.AddCommand(skillArchiveCmd)
	rootCmd.AddCommand(skillPinCmd)
	rootCmd.AddCommand(skillListLifecycleCmd)
	rootCmd.AddCommand(skillViewCmd)
}
