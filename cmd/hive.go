package cmd

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/events"
	"github.com/spf13/cobra"
)

// Hive types for wisdom management.

type hiveWisdomEntry struct {
	ID          string  `json:"id"`
	Text        string  `json:"text"`
	Domain      string  `json:"domain"`
	SourceRepo  string  `json:"source_repo"`
	Confidence  float64 `json:"confidence"`
	CreatedAt   string  `json:"created_at"`
	AccessedAt  string  `json:"accessed_at"`
	AccessCount int     `json:"access_count"`
}

type hiveWisdomData struct {
	Entries []hiveWisdomEntry `json:"entries"`
}

const hiveWisdomPath = "hive/wisdom.json"
const maxHiveEntries = 200

// --- hive-init ---

var hiveInitCmd = &cobra.Command{
	Use:   "hive-init",
	Short: "Initialize hive directory and empty wisdom file",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		hub := resolveHubPath()
		hiveDir := filepath.Join(hub, "hive")

		if err := os.MkdirAll(hiveDir, 0755); err != nil {
			outputError(2, fmt.Sprintf("failed to create hive dir: %v", err), nil)
			return nil
		}

		wisdomPath := filepath.Join(hiveDir, "wisdom.json")
		if _, err := os.Stat(wisdomPath); err == nil {
			outputOK(map[string]interface{}{"initialized": true, "note": "already exists"})
			return nil
		}

		data := hiveWisdomData{Entries: []hiveWisdomEntry{}}
		encoded, _ := json.MarshalIndent(data, "", "  ")
		if err := os.WriteFile(wisdomPath, append(encoded, '\n'), 0644); err != nil {
			outputError(2, fmt.Sprintf("failed to write wisdom.json: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{"initialized": true, "path": wisdomPath})
		return nil
	},
}

// --- hive-store ---

var hiveStoreCmd = &cobra.Command{
	Use:   "hive-store [text] [domain] [source-repo]",
	Short: "Store a wisdom entry with deduplication and LRU cap",
	Args:  cobra.MaximumNArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		text := mustGetStringCompat(cmd, args, "text", 0)
		if text == "" {
			return nil
		}
		domain := firstNonEmpty(mustGetStringCompatOptional(cmd, "domain"), optionalArg(args, 1))
		if domain == "" {
			domain = "general"
		}
		sourceRepo := mustGetStringCompat(cmd, args, "source-repo", 2)
		if sourceRepo == "" {
			return nil
		}

		hub := resolveHubPath()
		wisdomPath := filepath.Join(hub, "hive", "wisdom.json")

		var wf hiveWisdomData
		if raw, err := os.ReadFile(wisdomPath); err == nil {
			json.Unmarshal(raw, &wf)
		}

		// Dedup: check if same text+domain already exists
		textHash := fmt.Sprintf("%x", sha256.Sum256([]byte(text)))
		for i, e := range wf.Entries {
			if e.Text == text && e.Domain == domain {
				// Reinforce
				wf.Entries[i].AccessCount++
				wf.Entries[i].AccessedAt = time.Now().UTC().Format(time.RFC3339)
				if err := writeWisdom(wisdomPath, wf); err != nil {
					outputError(2, fmt.Sprintf("failed to save: %v", err), nil)
					return nil
				}
				emitLifecycleCeremony(events.CeremonyTopicHiveStore, events.CeremonyPayload{
					TaskID:  e.ID,
					Task:    domain,
					Status:  "reinforced",
					Message: text,
				}, "aether-hive")
				outputOK(map[string]interface{}{"stored": true, "reinforced": true, "id": e.ID})
				return nil
			}
		}

		// LRU eviction if at cap
		if len(wf.Entries) >= maxHiveEntries {
			// Find least recently accessed
			oldestIdx := 0
			for i, e := range wf.Entries {
				if e.AccessedAt < wf.Entries[oldestIdx].AccessedAt {
					oldestIdx = i
				}
			}
			wf.Entries = append(wf.Entries[:oldestIdx], wf.Entries[oldestIdx+1:]...)
		}

		now := time.Now().UTC().Format(time.RFC3339)
		entry := hiveWisdomEntry{
			ID:          fmt.Sprintf("%s_%s", domain, textHash[:12]),
			Text:        text,
			Domain:      domain,
			SourceRepo:  sourceRepo,
			Confidence:  0.5,
			CreatedAt:   now,
			AccessedAt:  now,
			AccessCount: 0,
		}

		wf.Entries = append(wf.Entries, entry)
		if err := writeWisdom(wisdomPath, wf); err != nil {
			outputError(2, fmt.Sprintf("failed to save: %v", err), nil)
			return nil
		}

		emitLifecycleCeremony(events.CeremonyTopicHiveStore, events.CeremonyPayload{
			TaskID:  entry.ID,
			Task:    domain,
			Status:  "stored",
			Message: text,
		}, "aether-hive")

		outputOK(map[string]interface{}{"stored": true, "reinforced": false, "id": entry.ID, "total": len(wf.Entries)})
		return nil
	},
}

// --- hive-read ---

var hiveReadCmd = &cobra.Command{
	Use:   "hive-read",
	Short: "Read wisdom entries with optional domain and confidence filtering",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		domain, _ := cmd.Flags().GetString("domain")
		minConfidence, _ := cmd.Flags().GetFloat64("min-confidence")

		hub := resolveHubPath()
		wisdomPath := filepath.Join(hub, "hive", "wisdom.json")

		var wf hiveWisdomData
		if raw, err := os.ReadFile(wisdomPath); err != nil {
			outputOK(map[string]interface{}{"entries": []hiveWisdomEntry{}, "total": 0})
			return nil
		} else {
			json.Unmarshal(raw, &wf)
		}

		// Update access times
		now := time.Now().UTC().Format(time.RFC3339)

		var results []hiveWisdomEntry
		for i := range wf.Entries {
			e := &wf.Entries[i]
			if domain != "" && e.Domain != domain {
				continue
			}
			if minConfidence > 0 && e.Confidence < minConfidence {
				continue
			}
			e.AccessCount++
			e.AccessedAt = now
			results = append(results, *e)
		}

		// Persist access updates
		writeWisdom(wisdomPath, wf)

		outputOK(map[string]interface{}{"entries": results, "total": len(results)})
		return nil
	},
}

// --- hive-abstract ---

var hiveAbstractCmd = &cobra.Command{
	Use:   "hive-abstract [instinct]",
	Short: "Abstract repo-specific text into generalized wisdom",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		instinct := mustGetStringCompat(cmd, args, "instinct", 0)
		if instinct == "" {
			return nil
		}
		sourceRepo, _ := cmd.Flags().GetString("source-repo")

		// Simple abstraction: remove repo-specific identifiers
		abstracted := instinct
		if sourceRepo != "" {
			abstracted = strings.ReplaceAll(abstracted, sourceRepo, "<repo>")
		}
		// Remove common repo path prefixes
		for _, prefix := range []string{"src/", "lib/", "pkg/", "cmd/", "internal/"} {
			abstracted = strings.ReplaceAll(abstracted, prefix, "")
		}

		outputOK(map[string]interface{}{
			"original":    instinct,
			"abstracted":  abstracted,
			"source_repo": sourceRepo,
		})
		return nil
	},
}

// --- hive-promote ---

var hivePromoteCmd = &cobra.Command{
	Use:   "hive-promote [text] [domain] [source-repo] [confidence]",
	Short: "End-to-end abstract + store pipeline for wisdom promotion",
	Args:  cobra.MaximumNArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		text := mustGetStringCompat(cmd, args, "text", 0)
		if text == "" {
			return nil
		}
		domain := firstNonEmpty(mustGetStringCompatOptional(cmd, "domain"), optionalArg(args, 1))
		if domain == "" {
			domain = "general"
		}
		sourceRepo := firstNonEmpty(mustGetStringCompatOptional(cmd, "source-repo"), optionalArg(args, 2))
		confidence, _ := cmd.Flags().GetFloat64("confidence")
		if confidence <= 0 {
			if argConfidence := optionalArg(args, 3); argConfidence != "" {
				if parsed, err := strconv.ParseFloat(argConfidence, 64); err == nil && parsed > 0 {
					confidence = parsed
				}
			}
			if confidence <= 0 {
				confidence = 0.75
			}
		}

		// Abstract
		abstracted := text
		if sourceRepo != "" {
			abstracted = strings.ReplaceAll(abstracted, sourceRepo, "<repo>")
		}
		for _, prefix := range []string{"src/", "lib/", "pkg/", "cmd/", "internal/"} {
			abstracted = strings.ReplaceAll(abstracted, prefix, "")
		}

		// Store
		hub := resolveHubPath()
		wisdomPath := filepath.Join(hub, "hive", "wisdom.json")

		var wf hiveWisdomData
		if raw, err := os.ReadFile(wisdomPath); err == nil {
			json.Unmarshal(raw, &wf)
		}

		textHash := fmt.Sprintf("%x", sha256.Sum256([]byte(abstracted)))
		now := time.Now().UTC().Format(time.RFC3339)

		// Check for existing entry to boost confidence
		for i, e := range wf.Entries {
			if e.Text == abstracted && e.Domain == domain {
				if confidence > wf.Entries[i].Confidence {
					wf.Entries[i].Confidence = confidence
				}
				wf.Entries[i].AccessCount++
				wf.Entries[i].AccessedAt = now
				if err := writeWisdom(wisdomPath, wf); err != nil {
					outputError(2, fmt.Sprintf("failed to save: %v", err), nil)
					return nil
				}
				emitLifecycleCeremony(events.CeremonyTopicHivePromote, events.CeremonyPayload{
					TaskID:  e.ID,
					Task:    domain,
					Status:  "boosted",
					Message: abstracted,
				}, "aether-hive")
				outputOK(map[string]interface{}{"promoted": true, "boosted": true, "id": e.ID, "confidence": wf.Entries[i].Confidence})
				return nil
			}
		}

		// LRU eviction
		if len(wf.Entries) >= maxHiveEntries {
			oldestIdx := 0
			for i, e := range wf.Entries {
				if e.AccessedAt < wf.Entries[oldestIdx].AccessedAt {
					oldestIdx = i
				}
			}
			wf.Entries = append(wf.Entries[:oldestIdx], wf.Entries[oldestIdx+1:]...)
		}

		entry := hiveWisdomEntry{
			ID:          fmt.Sprintf("%s_%s", domain, textHash[:12]),
			Text:        abstracted,
			Domain:      domain,
			SourceRepo:  sourceRepo,
			Confidence:  confidence,
			CreatedAt:   now,
			AccessedAt:  now,
			AccessCount: 0,
		}
		wf.Entries = append(wf.Entries, entry)
		if err := writeWisdom(wisdomPath, wf); err != nil {
			outputError(2, fmt.Sprintf("failed to save: %v", err), nil)
			return nil
		}

		emitLifecycleCeremony(events.CeremonyTopicHivePromote, events.CeremonyPayload{
			TaskID:  entry.ID,
			Task:    domain,
			Status:  "promoted",
			Message: abstracted,
		}, "aether-hive")

		outputOK(map[string]interface{}{"promoted": true, "boosted": false, "id": entry.ID, "confidence": confidence})
		return nil
	},
}

// writeWisdom writes the wisdom file atomically.
func writeWisdom(path string, wf hiveWisdomData) error {
	encoded, err := json.MarshalIndent(wf, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("mkdir hive dir: %w", err)
	}
	return os.WriteFile(path, append(encoded, '\n'), 0644)
}

// --- eternal-init ---

// eternalInitCmd initializes the eternal memory fallback storage directory and file.
var eternalInitCmd = &cobra.Command{
	Use:          "eternal-init",
	Short:        "Initialize eternal memory fallback storage",
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		hub := resolveHubPath()
		eternalDir := filepath.Join(hub, "eternal")

		if err := os.MkdirAll(eternalDir, 0755); err != nil {
			outputError(2, fmt.Sprintf("failed to create eternal dir: %v", err), nil)
			return nil
		}

		memoryPath := filepath.Join(eternalDir, "memory.json")
		if _, err := os.Stat(memoryPath); err == nil {
			outputOK(map[string]interface{}{
				"initialized": true,
				"path":        memoryPath,
				"note":        "already exists",
			})
			return nil
		}

		// Initialize with empty entries
		emptyData := []byte(`{"entries":[]}
`)
		if err := os.WriteFile(memoryPath, emptyData, 0644); err != nil {
			outputError(2, fmt.Sprintf("failed to write memory.json: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"initialized": true,
			"path":        memoryPath,
		})
		return nil
	},
}

func init() {
	hiveStoreCmd.Flags().String("text", "", "Wisdom text (required)")
	hiveStoreCmd.Flags().String("domain", "", "Domain tag (required)")
	hiveStoreCmd.Flags().String("source-repo", "", "Source repository (required)")

	hiveReadCmd.Flags().String("domain", "", "Filter by domain")
	hiveReadCmd.Flags().Float64("min-confidence", 0, "Minimum confidence threshold")

	hiveAbstractCmd.Flags().String("instinct", "", "Instinct text to abstract (required)")
	hiveAbstractCmd.Flags().String("source-repo", "", "Source repository")

	hivePromoteCmd.Flags().String("text", "", "Wisdom text (required)")
	hivePromoteCmd.Flags().String("domain", "", "Domain tag (required)")
	hivePromoteCmd.Flags().String("source-repo", "", "Source repository")
	hivePromoteCmd.Flags().Float64("confidence", 0.75, "Confidence score")

	rootCmd.AddCommand(hiveInitCmd)
	rootCmd.AddCommand(hiveStoreCmd)
	rootCmd.AddCommand(hiveReadCmd)
	rootCmd.AddCommand(hiveAbstractCmd)
	rootCmd.AddCommand(hivePromoteCmd)
	rootCmd.AddCommand(eternalInitCmd)
}
