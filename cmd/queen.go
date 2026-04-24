package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/storage"
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
		s := hubStore()
		if s == nil {
			return nil
		}
		queenPath := filepath.Join(resolveHubPath(), "QUEEN.md")

		if _, err := os.Stat(queenPath); err == nil {
			outputOK(map[string]interface{}{"created": false, "reason": "already exists", "path": queenPath})
			return nil
		}

		if err := s.AtomicWrite("QUEEN.md", []byte(queenDefaultContent)); err != nil {
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
		s := hubStore()
		if s == nil {
			return nil
		}
		queenPath := filepath.Join(resolveHubPath(), "QUEEN.md")

		data, err := s.ReadFile("QUEEN.md")
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
	Use:   "queen-promote [section|type] [content] [colony-name]",
	Short: "Write content to a QUEEN.md section",
	Args:  cobra.MaximumNArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		section := mapLegacyQueenSection(mustGetStringCompat(cmd, args, "section", 0))
		if section == "" {
			return nil
		}
		content := mustGetStringCompat(cmd, args, "content", 1)
		if content == "" {
			return nil
		}

		s := hubStore()
		if s == nil {
			return nil
		}
		text, _, err := loadQueenText(s)
		if err != nil {
			outputError(1, fmt.Sprintf("failed to load QUEEN.md: %v", err), nil)
			return nil
		}

		entry := fmt.Sprintf("- %s (promoted %s)", sanitizeQueenInline(content), time.Now().UTC().Format("2006-01-02"))
		text = appendEntryToQueenSection(text, section, entry)

		if err := writeQueenText(s, text); err != nil {
			outputError(2, fmt.Sprintf("failed to write QUEEN.md: %v", err), nil)
			return nil
		}

		emitLifecycleCeremony(events.CeremonyTopicQueenPromote, events.CeremonyPayload{
			Task:    section,
			Status:  "promoted",
			Message: sanitizeQueenInline(content),
		}, "aether-queen")

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
	Use:   "queen-write-learnings [learnings-json]",
	Short: "Write learning entries to QUEEN.md",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		learningsJSON := mustGetStringCompat(cmd, args, "learnings", 0)
		if learningsJSON == "" {
			return nil
		}

		var learnings []map[string]string
		if err := json.Unmarshal([]byte(learningsJSON), &learnings); err != nil {
			outputError(1, fmt.Sprintf("invalid learnings JSON: %v", err), nil)
			return nil
		}

		s := hubStore()
		if s == nil {
			return nil
		}
		text, _, err := loadQueenText(s)
		if err != nil {
			outputError(1, fmt.Sprintf("failed to load QUEEN.md: %v", err), nil)
			return nil
		}

		var entries []string
		for _, l := range learnings {
			claim := l["claim"]
			if claim != "" {
				entries = append(entries, fmt.Sprintf("- %s (phase learning, %s)", sanitizeQueenInline(claim), time.Now().UTC().Format("2006-01-02")))
			}
		}

		if len(entries) == 0 {
			outputOK(map[string]interface{}{"written": 0})
			return nil
		}

		text = appendEntriesToQueenSection(text, "Wisdom", entries)

		if err := writeQueenText(s, text); err != nil {
			outputError(2, fmt.Sprintf("failed to write QUEEN.md: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{"written": len(entries)})
		return nil
	},
}

// --- queen-promote-instinct ---

var queenPromoteInstinctCmd = &cobra.Command{
	Use:   "queen-promote-instinct [id]",
	Short: "Promote an instinct to QUEEN.md",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		instinctID := mustGetStringCompat(cmd, args, "id", 0)
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
		s := hubStore()
		if s == nil {
			return nil
		}
		text, _, err := loadQueenText(s)
		if err != nil {
			outputError(1, fmt.Sprintf("failed to load QUEEN.md: %v", err), nil)
			return nil
		}

		entry := fmt.Sprintf("- %s (instinct %s, %s)", sanitizeQueenInline(action), instinctID, time.Now().UTC().Format("2006-01-02"))
		text = appendEntryToQueenSection(text, "Wisdom", entry)

		if err := writeQueenText(s, text); err != nil {
			outputError(2, fmt.Sprintf("failed to write QUEEN.md: %v", err), nil)
			return nil
		}

		emitLifecycleCeremony(events.CeremonyTopicQueenPromote, events.CeremonyPayload{
			TaskID:  instinctID,
			Task:    "Wisdom",
			Status:  "promoted",
			Message: sanitizeQueenInline(action),
		}, "aether-queen")

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
		s := hubStore()
		if s == nil {
			return nil
		}

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

		text, _, err := loadQueenText(s)
		if err != nil {
			outputError(1, fmt.Sprintf("failed to load QUEEN.md: %v", err), nil)
			return nil
		}

		var entries []string
		for _, e := range wisdom.Entries {
			text, _ := e["text"].(string)
			if text != "" {
				entries = append(entries, fmt.Sprintf("- %s (hive wisdom)", sanitizeQueenInline(text)))
			}
		}

		text = appendEntriesToQueenSection(text, "Wisdom", entries)

		if err := writeQueenText(s, text); err != nil {
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
		s := hubStore()
		if s == nil {
			return nil
		}
		text, _, err := loadQueenText(s)
		if err != nil {
			outputError(1, fmt.Sprintf("failed to load QUEEN.md: %v", err), nil)
			return nil
		}

		// Check if already v2 (has Colony Charter section)
		if strings.Contains(text, "## Colony Charter") {
			outputOK(map[string]interface{}{"migrated": false, "reason": "already v2"})
			return nil
		}

		// Append Colony Charter section
		text += "\n## Colony Charter\n> Colony name and goal.\n"

		if err := writeQueenText(s, text); err != nil {
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
		name, _ := cmd.Flags().GetString("name")
		goal, _ := cmd.Flags().GetString("goal")
		intent, _ := cmd.Flags().GetString("intent")
		vision, _ := cmd.Flags().GetString("vision")
		governance, _ := cmd.Flags().GetString("governance")
		goals, _ := cmd.Flags().GetString("goals")
		if name == "" && goal == "" && intent == "" && vision == "" && governance == "" && goals == "" {
			return nil
		}
		domains, _ := cmd.Flags().GetString("domains")

		s := hubStore()
		if s == nil {
			return nil
		}
		text, _, err := loadQueenText(s)
		if err != nil {
			outputError(1, fmt.Sprintf("failed to load QUEEN.md: %v", err), nil)
			return nil
		}

		charterLines := buildCharterLines(name, goal, domains, intent, vision, governance, goals)
		text = replaceQueenSection(text, "Colony Charter", strings.Join(charterLines, "\n"))

		if err := writeQueenText(s, text); err != nil {
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
	charterWriteCmd.Flags().String("intent", "", "Legacy charter intent text")
	charterWriteCmd.Flags().String("vision", "", "Legacy charter vision text")
	charterWriteCmd.Flags().String("governance", "", "Legacy charter governance text")
	charterWriteCmd.Flags().String("goals", "", "Legacy charter goals text")

	for _, c := range []*cobra.Command{
		queenInitCmd, queenReadCmd, queenPromoteCmd,
		queenThresholdsCmd, queenWriteLearningsCmd, queenPromoteInstinctCmd,
		queenSeedFromHiveCmd, queenMigrateCmd, charterWriteCmd,
	} {
		rootCmd.AddCommand(c)
	}
}

func loadQueenText(s *storage.Store) (string, string, error) {
	queenPath := filepath.Join(resolveHubPath(), "QUEEN.md")
	data, err := s.ReadFile("QUEEN.md")
	if err != nil {
		return queenDefaultContent, queenPath, nil
	}
	return string(data), queenPath, nil
}

func writeQueenText(s *storage.Store, text string) error {
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("refusing to overwrite QUEEN.md with empty content")
	}
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	return s.AtomicWrite("QUEEN.md", []byte(text))
}

func sanitizeQueenInline(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.TrimSpace(strings.Join(strings.Fields(value), " "))
}

func appendEntryToQueenSection(text, section, entry string) string {
	return appendEntriesToQueenSection(text, section, []string{entry})
}

func appendEntriesToQueenSection(text, section string, entries []string) string {
	if len(entries) == 0 {
		return text
	}
	sectionHeader := "## " + section
	block := strings.Join(entries, "\n")
	idx := strings.Index(text, sectionHeader)
	if idx == -1 {
		if !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		return text + fmt.Sprintf("\n## %s\n\n%s\n", section, block)
	}
	insertAt := idx + len(sectionHeader)
	nlIdx := strings.Index(text[insertAt:], "\n")
	if nlIdx != -1 {
		insertAt += nlIdx + 1
	}
	return text[:insertAt] + block + "\n" + text[insertAt:]
}

func replaceQueenSection(text, section, replacement string) string {
	sectionHeader := "## " + section
	if !strings.HasPrefix(replacement, "\n") {
		replacement = "\n\n" + replacement
	}
	if !strings.HasSuffix(replacement, "\n") {
		replacement += "\n"
	}
	idx := strings.Index(text, sectionHeader)
	if idx == -1 {
		if !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		return text + fmt.Sprintf("\n## %s%s", section, replacement)
	}
	insertAt := idx + len(sectionHeader)
	nextSection := strings.Index(text[insertAt:], "\n## ")
	if nextSection != -1 {
		return text[:insertAt] + replacement + text[insertAt+nextSection:]
	}
	return text[:insertAt] + replacement
}

func mapLegacyQueenSection(section string) string {
	switch strings.ToLower(strings.TrimSpace(section)) {
	case "pattern", "patterns":
		return "Patterns"
	case "philosophy", "philosophies":
		return "Philosophies"
	case "anti-pattern", "anti-patterns", "antipattern", "antipatterns":
		return "Anti-Patterns"
	case "preference", "preferences", "user-preference", "user-preferences":
		return "User Preferences"
	case "learning", "learnings", "wisdom":
		return "Wisdom"
	default:
		return section
	}
}

func buildCharterLines(name, goal, domains, intent, vision, governance, goals string) []string {
	var lines []string
	if name != "" {
		lines = append(lines, "- **Name:** "+sanitizeQueenInline(name))
	}
	if goal != "" {
		lines = append(lines, "- **Goal:** "+sanitizeQueenInline(goal))
	}
	if domains != "" {
		lines = append(lines, "- **Domains:** "+sanitizeQueenInline(domains))
	}
	if intent != "" {
		lines = append(lines, "- **Intent:** "+sanitizeQueenInline(intent))
	}
	if vision != "" {
		lines = append(lines, "- **Vision:** "+sanitizeQueenInline(vision))
	}
	if governance != "" {
		lines = append(lines, "- **Governance:** "+sanitizeQueenInline(governance))
	}
	if goals != "" {
		lines = append(lines, "- **Goals:** "+sanitizeQueenInline(goals))
	}
	return lines
}
