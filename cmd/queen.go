package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// Queen commands operate on hub-level QUEEN.md and colony state.

const queenDefaultContent = `# QUEEN.md — Colony Wisdom Hub

## Wisdom
> Patterns and insights earned through colony work.

## Patterns
> Recurring solutions that worked.

## Philosophies
> Higher-level principles guiding decisions.

## Anti-Patterns
> Things to avoid.

## User Preferences
> Communication style and decision patterns.

## Colony Charter
> Colony name and goal.
`

// --- queen-init ---

var queenInitCmd = &cobra.Command{
	Use:   "queen-init",
	Short: "Create QUEEN.md with standard sections if not exists",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		hub := resolveHubPath()
		queenPath := filepath.Join(hub, "QUEEN.md")

		if _, err := os.Stat(queenPath); err == nil {
			outputOK(map[string]interface{}{"created": false, "reason": "already exists", "path": queenPath})
			return nil
		}

		if err := os.MkdirAll(hub, 0755); err != nil {
			outputError(2, fmt.Sprintf("failed to create hub dir: %v", err), nil)
			return nil
		}

		if err := os.WriteFile(queenPath, []byte(queenDefaultContent), 0644); err != nil {
			outputError(2, fmt.Sprintf("failed to write QUEEN.md: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{"created": true, "path": queenPath})
		return nil
	},
}

// --- queen-read ---

var queenReadCmd = &cobra.Command{
	Use:   "queen-read",
	Short: "Read and return QUEEN.md content",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		hub := resolveHubPath()
		queenPath := filepath.Join(hub, "QUEEN.md")

		data, err := os.ReadFile(queenPath)
		if err != nil {
			outputError(1, fmt.Sprintf("failed to read QUEEN.md: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"content": string(data),
			"path":    queenPath,
			"size":    len(data),
		})
		return nil
	},
}

// --- queen-promote ---

var queenPromoteCmd = &cobra.Command{
	Use:   "queen-promote",
	Short: "Write content to a QUEEN.md section",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		section := mustGetString(cmd, "section")
		if section == "" {
			return nil
		}
		content := mustGetString(cmd, "content")
		if content == "" {
			return nil
		}

		hub := resolveHubPath()
		queenPath := filepath.Join(hub, "QUEEN.md")

		data, err := os.ReadFile(queenPath)
		if err != nil {
			outputError(1, fmt.Sprintf("failed to read QUEEN.md: %v", err), nil)
			return nil
		}

		sectionHeader := "## " + section
		entry := fmt.Sprintf("- %s (promoted %s)", content, time.Now().UTC().Format("2006-01-02"))

		text := string(data)
		idx := strings.Index(text, sectionHeader)
		if idx == -1 {
			// Append new section
			text += fmt.Sprintf("\n## %s\n\n%s\n", section, entry)
		} else {
			// Find end of section header line, insert after it
			insertAt := idx + len(sectionHeader)
			// Skip to end of line
			nlIdx := strings.Index(text[insertAt:], "\n")
			if nlIdx != -1 {
				insertAt += nlIdx + 1
			}
			text = text[:insertAt] + entry + "\n" + text[insertAt:]
		}

		if err := os.WriteFile(queenPath, []byte(text), 0644); err != nil {
			outputError(2, fmt.Sprintf("failed to write QUEEN.md: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{"promoted": true, "section": section})
		return nil
	},
}

// --- queen-thresholds ---

var queenThresholdsCmd = &cobra.Command{
	Use:   "queen-thresholds",
	Short: "Return wisdom thresholds configuration",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		outputOK(map[string]interface{}{
			"trust_promote_threshold": 0.75,
			"trust_hive_threshold":    0.80,
			"trust_decay_half_life":   60,
			"trust_floor":             0.2,
			"max_instincts":           50,
			"max_wisdom_entries":      200,
		})
		return nil
	},
}

// --- queen-write-learnings ---

var queenWriteLearningsCmd = &cobra.Command{
	Use:   "queen-write-learnings",
	Short: "Write learning entries to QUEEN.md",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		learningsJSON := mustGetString(cmd, "learnings")
		if learningsJSON == "" {
			return nil
		}

		var learnings []map[string]string
		if err := json.Unmarshal([]byte(learningsJSON), &learnings); err != nil {
			outputError(1, fmt.Sprintf("invalid learnings JSON: %v", err), nil)
			return nil
		}

		hub := resolveHubPath()
		queenPath := filepath.Join(hub, "QUEEN.md")

		data, err := os.ReadFile(queenPath)
		if err != nil {
			outputError(1, fmt.Sprintf("failed to read QUEEN.md: %v", err), nil)
			return nil
		}

		var entries []string
		for _, l := range learnings {
			claim := l["claim"]
			if claim != "" {
				entries = append(entries, fmt.Sprintf("- %s (phase learning, %s)", claim, time.Now().UTC().Format("2006-01-02")))
			}
		}

		if len(entries) == 0 {
			outputOK(map[string]interface{}{"written": 0})
			return nil
		}

		text := string(data)
		sectionHeader := "## Wisdom"
		idx := strings.Index(text, sectionHeader)
		if idx == -1 {
			text += fmt.Sprintf("\n## Wisdom\n\n%s\n", strings.Join(entries, "\n"))
		} else {
			insertAt := idx + len(sectionHeader)
			nlIdx := strings.Index(text[insertAt:], "\n")
			if nlIdx != -1 {
				insertAt += nlIdx + 1
			}
			text = text[:insertAt] + strings.Join(entries, "\n") + "\n" + text[insertAt:]
		}

		if err := os.WriteFile(queenPath, []byte(text), 0644); err != nil {
			outputError(2, fmt.Sprintf("failed to write QUEEN.md: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{"written": len(entries)})
		return nil
	},
}

// --- queen-promote-instinct ---

var queenPromoteInstinctCmd = &cobra.Command{
	Use:   "queen-promote-instinct",
	Short: "Promote an instinct to QUEEN.md",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		instinctID := mustGetString(cmd, "id")
		if instinctID == "" {
			return nil
		}

		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		// Load instinct from instincts.json
		var instincts struct {
			Instincts []map[string]interface{} `json:"instincts"`
		}
		if err := store.LoadJSON("instincts.json", &instincts); err != nil {
			outputError(1, fmt.Sprintf("failed to load instincts: %v", err), nil)
			return nil
		}

		var found *map[string]interface{}
		for i := range instincts.Instincts {
			if id, ok := instincts.Instincts[i]["id"].(string); ok && id == instinctID {
				found = &instincts.Instincts[i]
				break
			}
		}
		if found == nil {
			outputError(1, fmt.Sprintf("instinct %s not found", instinctID), nil)
			return nil
		}

		action, _ := (*found)["action"].(string)
		if action == "" {
			outputError(1, "instinct has no action field", nil)
			return nil
		}

		// Write to QUEEN.md Wisdom section
		hub := resolveHubPath()
		queenPath := filepath.Join(hub, "QUEEN.md")

		data, err := os.ReadFile(queenPath)
		if err != nil {
			outputError(1, fmt.Sprintf("failed to read QUEEN.md: %v", err), nil)
			return nil
		}

		entry := fmt.Sprintf("- %s (instinct %s, %s)", action, instinctID, time.Now().UTC().Format("2006-01-02"))
		text := string(data)
		sectionHeader := "## Wisdom"
		idx := strings.Index(text, sectionHeader)
		if idx == -1 {
			text += fmt.Sprintf("\n## Wisdom\n\n%s\n", entry)
		} else {
			insertAt := idx + len(sectionHeader)
			nlIdx := strings.Index(text[insertAt:], "\n")
			if nlIdx != -1 {
				insertAt += nlIdx + 1
			}
			text = text[:insertAt] + entry + "\n" + text[insertAt:]
		}

		if err := os.WriteFile(queenPath, []byte(text), 0644); err != nil {
			outputError(2, fmt.Sprintf("failed to write QUEEN.md: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{"promoted": true, "instinct_id": instinctID})
		return nil
	},
}

// --- queen-seed-from-hive ---

var queenSeedFromHiveCmd = &cobra.Command{
	Use:   "queen-seed-from-hive",
	Short: "Seed QUEEN.md with relevant hive wisdom",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		hub := resolveHubPath()
		wisdomPath := filepath.Join(hub, "hive", "wisdom.json")
		queenPath := filepath.Join(hub, "QUEEN.md")

		var wisdom struct {
			Entries []map[string]interface{} `json:"entries"`
		}
		if raw, err := os.ReadFile(wisdomPath); err != nil {
			outputError(1, fmt.Sprintf("failed to read hive wisdom: %v", err), nil)
			return nil
		} else {
			if err := json.Unmarshal(raw, &wisdom); err != nil {
				log.Printf("queen-seed-from-hive: failed to unmarshal wisdom JSON: %v", err)
			}
		}

		if len(wisdom.Entries) == 0 {
			outputOK(map[string]interface{}{"seeded": 0, "reason": "no hive wisdom entries"})
			return nil
		}

		data, err := os.ReadFile(queenPath)
		if err != nil {
			outputError(1, fmt.Sprintf("failed to read QUEEN.md: %v", err), nil)
			return nil
		}

		var entries []string
		for _, e := range wisdom.Entries {
			text, _ := e["text"].(string)
			if text != "" {
				entries = append(entries, fmt.Sprintf("- %s (hive wisdom)", text))
			}
		}

		queenText := string(data)
		sectionHeader := "## Wisdom"
		idx := strings.Index(queenText, sectionHeader)
		if idx == -1 {
			queenText += fmt.Sprintf("\n## Wisdom\n\n%s\n", strings.Join(entries, "\n"))
		} else {
			insertAt := idx + len(sectionHeader)
			nlIdx := strings.Index(queenText[insertAt:], "\n")
			if nlIdx != -1 {
				insertAt += nlIdx + 1
			}
			queenText = queenText[:insertAt] + strings.Join(entries, "\n") + "\n" + queenText[insertAt:]
		}

		if err := os.WriteFile(queenPath, []byte(queenText), 0644); err != nil {
			outputError(2, fmt.Sprintf("failed to write QUEEN.md: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{"seeded": len(entries)})
		return nil
	},
}

// --- queen-migrate ---

var queenMigrateCmd = &cobra.Command{
	Use:   "queen-migrate",
	Short: "Migrate QUEEN.md from v1 to v2 format",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		hub := resolveHubPath()
		queenPath := filepath.Join(hub, "QUEEN.md")

		data, err := os.ReadFile(queenPath)
		if err != nil {
			outputError(1, fmt.Sprintf("failed to read QUEEN.md: %v", err), nil)
			return nil
		}

		text := string(data)

		// Check if already v2 (has Colony Charter section)
		if strings.Contains(text, "## Colony Charter") {
			outputOK(map[string]interface{}{"migrated": false, "reason": "already v2"})
			return nil
		}

		// Append Colony Charter section
		text += "\n## Colony Charter\n> Colony name and goal.\n"

		if err := os.WriteFile(queenPath, []byte(text), 0644); err != nil {
			outputError(2, fmt.Sprintf("failed to write QUEEN.md: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{"migrated": true})
		return nil
	},
}

// --- charter-write ---

var charterWriteCmd = &cobra.Command{
	Use:   "charter-write",
	Short: "Write colony charter to QUEEN.md",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := mustGetString(cmd, "name")
		if name == "" {
			return nil
		}
		goal := mustGetString(cmd, "goal")
		if goal == "" {
			return nil
		}
		domains, _ := cmd.Flags().GetString("domains")

		hub := resolveHubPath()
		queenPath := filepath.Join(hub, "QUEEN.md")

		data, err := os.ReadFile(queenPath)
		if err != nil {
			outputError(1, fmt.Sprintf("failed to read QUEEN.md: %v", err), nil)
			return nil
		}

		charter := fmt.Sprintf("- **Name:** %s\n- **Goal:** %s\n- **Domains:** %s", name, goal, domains)

		text := string(data)
		sectionHeader := "## Colony Charter"
		idx := strings.Index(text, sectionHeader)
		if idx == -1 {
			text += fmt.Sprintf("\n## Colony Charter\n\n%s\n", charter)
		} else {
			// Replace existing charter content
			insertAt := idx + len(sectionHeader)
			// Find next section or end of file
			nextSection := strings.Index(text[insertAt:], "\n## ")
			if nextSection != -1 {
				text = text[:insertAt] + "\n\n" + charter + "\n" + text[insertAt+nextSection:]
			} else {
				text = text[:insertAt] + "\n\n" + charter + "\n"
			}
		}

		if err := os.WriteFile(queenPath, []byte(text), 0644); err != nil {
			outputError(2, fmt.Sprintf("failed to write QUEEN.md: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{"written": true, "name": name})
		return nil
	},
}

func init() {
	queenPromoteCmd.Flags().String("content", "", "Content to promote (required)")
	queenPromoteCmd.Flags().String("section", "", "Target section name (required)")
	queenWriteLearningsCmd.Flags().String("learnings", "", "JSON array of learning objects (required)")
	queenPromoteInstinctCmd.Flags().String("id", "", "Instinct ID (required)")
	charterWriteCmd.Flags().String("name", "", "Colony name (required)")
	charterWriteCmd.Flags().String("goal", "", "Colony goal (required)")
	charterWriteCmd.Flags().String("domains", "", "Comma-separated domain tags")

	for _, c := range []*cobra.Command{
		queenInitCmd, queenReadCmd, queenPromoteCmd,
		queenThresholdsCmd, queenWriteLearningsCmd, queenPromoteInstinctCmd,
		queenSeedFromHiveCmd, queenMigrateCmd, charterWriteCmd,
	} {
		rootCmd.AddCommand(c)
	}
}
