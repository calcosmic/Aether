package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
	"github.com/spf13/cobra"
)

// --- force-unlock ---

var forceUnlockCmd = &cobra.Command{
	Use:   "force-unlock",
	Short: "Emergency release of a file lock",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		dataDir := resolveDataDir()
		locksDir := filepath.Join(filepath.Dir(dataDir), "locks")

		fileFilter, _ := cmd.Flags().GetString("file")

		count := 0
		entries, err := os.ReadDir(locksDir)
		if err != nil {
			// No locks directory = nothing to unlock
			outputOK(map[string]interface{}{"unlocked": true, "count": 0})
			return nil
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if filepath.Ext(name) != ".lock" {
				continue
			}
			if fileFilter != "" && name != fileFilter {
				continue
			}
			lockPath := filepath.Join(locksDir, name)
			if err := os.Remove(lockPath); err == nil {
				count++
			}
		}

		outputOK(map[string]interface{}{"unlocked": true, "count": count})
		return nil
	},
}

// --- entropy-score ---

var entropyScoreCmd = &cobra.Command{
	Use:   "entropy-score",
	Short: "Compute colony health/entropy score",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputError(1, "COLONY_STATE.json not found", nil)
			return nil
		}

		// Compute score (0-100) based on colony health factors
		score := 100.0
		factors := map[string]interface{}{}

		// Factor: error records (lower = better)
		errorCount := len(state.Errors.Records)
		errorPenalty := float64(errorCount) * 2.0
		if errorPenalty > 30 {
			errorPenalty = 30
		}
		score -= errorPenalty
		factors["error_count"] = errorCount

		// Factor: instincts (more = healthier, up to a point)
		instinctCount := len(state.Memory.Instincts)
		instinctBonus := float64(instinctCount) * 1.0
		if instinctBonus > 10 {
			instinctBonus = 10
		}
		score += instinctBonus
		factors["instinct_count"] = instinctCount

		// Factor: decisions (more = thoughtful)
		decisionCount := len(state.Memory.Decisions)
		decisionBonus := float64(decisionCount) * 0.5
		if decisionBonus > 10 {
			decisionBonus = 10
		}
		score += decisionBonus
		factors["decision_count"] = decisionCount

		// Factor: graveyards (failed builds, lower = better)
		graveyardCount := len(state.Graveyards)
		graveyardPenalty := float64(graveyardCount) * 5.0
		if graveyardPenalty > 20 {
			graveyardPenalty = 20
		}
		score -= graveyardPenalty
		factors["graveyard_count"] = graveyardCount

		// Clamp score
		if score < 0 {
			score = 0
		}
		if score > 100 {
			score = 100
		}
		factors["final_score"] = score

		outputOK(map[string]interface{}{
			"score":   score,
			"factors": factors,
		})
		return nil
	},
}

// --- eternal-store ---

var eternalStoreCmd = &cobra.Command{
	Use:   "eternal-store",
	Short: "Store a high-value signal in eternal memory",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		content := mustGetString(cmd, "content")
		if content == "" {
			return nil
		}
		category, _ := cmd.Flags().GetString("category")
		if category == "" {
			category = "general"
		}
		confidence, _ := cmd.Flags().GetFloat64("confidence")
		if confidence <= 0 {
			confidence = 0.9
		}

		hub := resolveHubPath()
		eternalDir := filepath.Join(hub, "eternal")

		if err := os.MkdirAll(eternalDir, 0755); err != nil {
			outputError(2, fmt.Sprintf("failed to create eternal dir: %v", err), nil)
			return nil
		}

		memoryPath := filepath.Join(eternalDir, "memory.json")

		type eternalEntry struct {
			ID         string  `json:"id"`
			Content    string  `json:"content"`
			Category   string  `json:"category"`
			Confidence float64 `json:"confidence"`
			CreatedAt  string  `json:"created_at"`
			AccessedAt string  `json:"accessed_at"`
		}

		type eternalData struct {
			Entries []eternalEntry `json:"entries"`
		}

		var ed eternalData
		if raw, err := os.ReadFile(memoryPath); err == nil {
			json.Unmarshal(raw, &ed)
		}

		now := time.Now().UTC().Format(time.RFC3339)
		entry := eternalEntry{
			ID:         fmt.Sprintf("eternal_%d", time.Now().Unix()),
			Content:    content,
			Category:   category,
			Confidence: confidence,
			CreatedAt:  now,
			AccessedAt: now,
		}

		ed.Entries = append(ed.Entries, entry)

		// Cap at 200 entries with LRU eviction
		if len(ed.Entries) > 200 {
			oldestIdx := 0
			for i, e := range ed.Entries {
				if e.AccessedAt < ed.Entries[oldestIdx].AccessedAt {
					oldestIdx = i
				}
			}
			ed.Entries = append(ed.Entries[:oldestIdx], ed.Entries[oldestIdx+1:]...)
		}

		encoded, _ := json.MarshalIndent(ed, "", "  ")
		if err := os.WriteFile(memoryPath, append(encoded, '\n'), 0644); err != nil {
			outputError(2, fmt.Sprintf("failed to write memory.json: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"stored": true,
			"id":     entry.ID,
			"total":  len(ed.Entries),
		})
		return nil
	},
}

// --- incident-rule-add ---

var incidentRuleAddCmd = &cobra.Command{
	Use:   "incident-rule-add",
	Short: "Add a rule to decree, constraint, or gate file",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		ruleType := mustGetString(cmd, "type")
		if ruleType == "" {
			return nil
		}
		rule := mustGetString(cmd, "rule")
		if rule == "" {
			return nil
		}
		priority, _ := cmd.Flags().GetString("priority")
		if priority == "" {
			priority = "normal"
		}

		// Map type to file
		var filename string
		switch ruleType {
		case "decree":
			filename = "decree.json"
		case "constraint":
			filename = "constraints.json"
		case "gate":
			filename = "gates.json"
		default:
			outputError(1, fmt.Sprintf("invalid type %q; must be decree, constraint, or gate", ruleType), nil)
			return nil
		}

		// Load existing file or create new
		var rules []interface{}
		if data, err := store.ReadFile(filename); err == nil {
			json.Unmarshal(data, &rules)
		}
		if rules == nil {
			rules = []interface{}{}
		}

		now := time.Now().UTC().Format(time.RFC3339)
		rules = append(rules, map[string]interface{}{
			"rule":       rule,
			"type":       ruleType,
			"priority":   priority,
			"created_at": now,
		})

		if err := store.SaveJSON(filename, rules); err != nil {
			outputError(2, fmt.Sprintf("failed to save %s: %v", filename, err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"added":    true,
			"type":     ruleType,
			"rule":     rule,
			"priority": priority,
		})
		return nil
	},
}

// --- bootstrap-system ---

var bootstrapSystemCmd = &cobra.Command{
	Use:   "bootstrap-system",
	Short: "Copy system files from hub to colony",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		hub := resolveHubPath()
		aetherRoot := storage.ResolveAetherRoot(context.Background())

		// Files to bootstrap from hub templates to colony
		templateFiles := []string{
			"templates/colony-state-template.json",
			"templates/pheromones-template.json",
		}

		var copied []string
		var skipped []string

		for _, tf := range templateFiles {
			src := filepath.Join(hub, tf)
			dst := filepath.Join(aetherRoot, ".aether", tf)

			if _, err := os.Stat(dst); err == nil {
				skipped = append(skipped, tf)
				continue
			}

			if _, err := os.Stat(src); err != nil {
				skipped = append(skipped, tf)
				continue
			}

			if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
				skipped = append(skipped, tf)
				continue
			}

			data, err := os.ReadFile(src)
			if err != nil {
				skipped = append(skipped, tf)
				continue
			}

			if err := os.WriteFile(dst, data, 0644); err != nil {
				skipped = append(skipped, tf)
				continue
			}

			copied = append(copied, tf)
		}

		if copied == nil {
			copied = []string{}
		}
		if skipped == nil {
			skipped = []string{}
		}

		outputOK(map[string]interface{}{
			"copied":  copied,
			"skipped": skipped,
		})
		return nil
	},
}

// --- error-patterns-check (deprecated alias for error-pattern-check) ---

var errorPatternsCheckCmd = &cobra.Command{
	Use:        "error-patterns-check",
	Short:      "[DEPRECATED] Check for known error patterns (use error-pattern-check)",
	Deprecated: "use error-pattern-check instead",
	Args:       cobra.NoArgs,
	RunE:       errorPatternCheckCmd.RunE,
}

// --- instinct-read ---

var instinctReadCmd = &cobra.Command{
	Use:   "instinct-read",
	Short: "Read an instinct by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		instinctID := args[0]

		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputError(1, "COLONY_STATE.json not found", nil)
			return nil
		}

		for _, inst := range state.Memory.Instincts {
			if inst.ID == instinctID {
				outputOK(map[string]interface{}{
					"instinct": inst,
					"found":    true,
				})
				return nil
			}
		}

		outputError(1, fmt.Sprintf("instinct %q not found", instinctID), nil)
		return nil
	},
}

// --- instinct-apply ---

var instinctApplyCmd = &cobra.Command{
	Use:   "instinct-apply",
	Short: "Mark an instinct as applied and update stats",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		instinctID := args[0]
		success, _ := cmd.Flags().GetBool("success")

		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputError(1, "COLONY_STATE.json not found", nil)
			return nil
		}

		found := false
		for i := range state.Memory.Instincts {
			if state.Memory.Instincts[i].ID == instinctID {
				found = true
				state.Memory.Instincts[i].Applications++
				if success {
					state.Memory.Instincts[i].Successes++
				} else {
					state.Memory.Instincts[i].Failures++
				}
				now := time.Now().UTC().Format("2006-01-02T15:04:05Z")
				state.Memory.Instincts[i].LastApplied = &now
				break
			}
		}

		if !found {
			outputError(1, fmt.Sprintf("instinct %q not found", instinctID), nil)
			return nil
		}

		if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
			outputError(2, fmt.Sprintf("failed to save: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"applied": true,
			"id":      instinctID,
			"success": success,
		})
		return nil
	},
}

// --- spawn-get-depth ---

var spawnGetDepthCmd = &cobra.Command{
	Use:   "spawn-get-depth",
	Short: "Get current spawn depth for a named ant",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		name := mustGetString(cmd, "name")
		if name == "" {
			return nil
		}

		// Load spawn tree and find the named ant
		data, err := store.ReadFile("spawn-tree.txt")
		if err != nil {
			outputOK(map[string]interface{}{"name": name, "depth": 0, "found": false})
			return nil
		}

		// Parse the spawn tree text format (pipe-delimited)
		depth := 0
		found := false
		lines := splitLines(string(data))
		for _, line := range lines {
			fields := splitPipe(line)
			// Format: timestamp|parent|caste|agentName|task|depth|status
			if len(fields) >= 7 && fields[3] == name {
				found = true
				depth = parseIntSafe(fields[5])
				break
			}
		}

		outputOK(map[string]interface{}{
			"name":  name,
			"depth": depth,
			"found": found,
		})
		return nil
	},
}

// --- spawn-can-spawn-swarm ---

var spawnCanSpawnSwarmCmd = &cobra.Command{
	Use:   "spawn-can-spawn-swarm",
	Short: "Check if budget allows swarm spawning",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			// No state = no spawns, budget available
			outputOK(map[string]interface{}{
				"can_spawn":        true,
				"remaining_budget": 5,
				"current_spawns":   0,
			})
			return nil
		}

		// Default budget is 5 unless colony_depth says otherwise
		maxBudget := 5
		if state.ColonyDepth != "" {
			if d := parseIntSafe(state.ColonyDepth); d > 0 {
				maxBudget = d
			}
		}

		currentSpawns := 0
		data, err := store.ReadFile("spawn-tree.txt")
		if err == nil {
			lines := splitLines(string(data))
			for _, line := range lines {
				fields := splitLineFields(line)
				for _, f := range fields {
					if f == "active" || f == "running" {
						currentSpawns++
						break
					}
				}
			}
		}

		canSpawn := currentSpawns < maxBudget
		remaining := maxBudget - currentSpawns
		if remaining < 0 {
			remaining = 0
		}

		outputOK(map[string]interface{}{
			"can_spawn":        canSpawn,
			"remaining_budget": remaining,
			"current_spawns":   currentSpawns,
			"max_budget":       maxBudget,
		})
		return nil
	},
}

// --- swarm-display-get ---

var swarmDisplayGetCmd = &cobra.Command{
	Use:   "swarm-display-get",
	Short: "Read current swarm display state",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		state, err := loadColonyState()
		if err != nil {
			outputErrorMessage(err.Error())
			return nil
		}
		if state == nil {
			outputOK(map[string]interface{}{
				"state":        nil,
				"colony_count": 0,
			})
			return nil
		}

		outputOK(map[string]interface{}{
			"goal":          goalStr(state.Goal),
			"milestone":     state.Milestone,
			"state":         string(state.State),
			"current_phase": state.CurrentPhase,
			"phases":        state.Plan.Phases,
			"colony_count":  len(state.Plan.Phases),
		})
		return nil
	},
}

// --- swarm-activity-log ---

var swarmActivityLogCmd = &cobra.Command{
	Use:   "swarm-activity-log",
	Short: "Log swarm activity",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		message := mustGetString(cmd, "message")
		if message == "" {
			return nil
		}
		severity, _ := cmd.Flags().GetString("severity")
		if severity == "" {
			severity = "info"
		}

		now := time.Now().UTC().Format(time.RFC3339)
		entry := map[string]interface{}{
			"message":   message,
			"severity":  severity,
			"timestamp": now,
		}

		// Append to swarm activity log as JSONL
		if err := store.AppendJSONL("swarm-activity.jsonl", entry); err != nil {
			outputError(2, fmt.Sprintf("failed to log activity: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"logged":    true,
			"severity":  severity,
			"timestamp": now,
		})
		return nil
	},
}

// Helper functions for text parsing

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			if line != "" {
				lines = append(lines, line)
			}
			start = i + 1
		}
	}
	if start < len(s) {
		remaining := s[start:]
		if remaining != "" {
			lines = append(lines, remaining)
		}
	}
	return lines
}

func splitLineFields(line string) []string {
	var fields []string
	inField := false
	start := 0
	for i := 0; i < len(line); i++ {
		if line[i] == ' ' || line[i] == '\t' {
			if inField {
				fields = append(fields, line[start:i])
				inField = false
			}
		} else {
			if !inField {
				start = i
				inField = true
			}
		}
	}
	if inField {
		fields = append(fields, line[start:])
	}
	return fields
}

// splitPipe splits a string by pipe character.
func splitPipe(s string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '|' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}

func parseDepthFromFields(fields []string) int {
	// Try to parse the last field or any numeric field as depth
	for i := len(fields) - 1; i >= 0; i-- {
		if d := parseIntSafe(fields[i]); d > 0 {
			return d
		}
	}
	return 0
}

func parseIntSafe(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			if n > 0 {
				return n
			}
		}
	}
	return n
}

func init() {
	forceUnlockCmd.Flags().String("file", "", "Specific lock file to remove")

	eternalStoreCmd.Flags().String("content", "", "Signal content (required)")
	eternalStoreCmd.Flags().String("category", "general", "Signal category")
	eternalStoreCmd.Flags().Float64("confidence", 0.9, "Confidence score")

	incidentRuleAddCmd.Flags().String("type", "", "Rule type: decree, constraint, or gate (required)")
	incidentRuleAddCmd.Flags().String("rule", "", "Rule content (required)")
	incidentRuleAddCmd.Flags().String("priority", "normal", "Rule priority")

	instinctApplyCmd.Flags().Bool("success", true, "Whether the application was successful")

	spawnGetDepthCmd.Flags().String("name", "", "Ant name to look up (required)")

	swarmActivityLogCmd.Flags().String("message", "", "Activity message (required)")
	swarmActivityLogCmd.Flags().String("severity", "info", "Severity level")

	rootCmd.AddCommand(forceUnlockCmd)
	rootCmd.AddCommand(entropyScoreCmd)
	rootCmd.AddCommand(eternalStoreCmd)
	rootCmd.AddCommand(incidentRuleAddCmd)
	rootCmd.AddCommand(bootstrapSystemCmd)
	rootCmd.AddCommand(errorPatternsCheckCmd)
	rootCmd.AddCommand(instinctReadCmd)
	rootCmd.AddCommand(instinctApplyCmd)
	rootCmd.AddCommand(spawnGetDepthCmd)
	rootCmd.AddCommand(spawnCanSpawnSwarmCmd)
	rootCmd.AddCommand(swarmDisplayGetCmd)
	rootCmd.AddCommand(swarmActivityLogCmd)
}
