package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/spf13/cobra"
)

var spawnLogCmd = &cobra.Command{
	Use:   "spawn-log",
	Short: "Record a new agent spawn entry",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		parent, _ := cmd.Flags().GetString("parent")
		name, _ := cmd.Flags().GetString("name")
		legacyID, _ := cmd.Flags().GetString("id")
		legacyDescription, _ := cmd.Flags().GetString("description")
		if legacyID != "" {
			if parent == "" && name != "" {
				parent = name
			}
			name = legacyID
		}
		if parent == "" {
			outputError(1, "flag --parent is required", nil)
			return nil
		}
		caste := mustGetString(cmd, "caste")
		if caste == "" {
			return nil
		}
		if name == "" {
			outputError(1, "flag --name is required", nil)
			return nil
		}
		task := firstNonEmpty(mustGetStringCompatOptional(cmd, "task"), legacyDescription)
		if task == "" {
			outputError(1, "flag --task is required", nil)
			return nil
		}
		depth, _ := cmd.Flags().GetInt("depth")

		st := agent.NewSpawnTree(store, "spawn-tree.txt")
		if err := st.RecordSpawn(parent, caste, name, task, depth); err != nil {
			outputError(2, fmt.Sprintf("failed to record spawn: %v", err), nil)
			return nil
		}
		eventID := emitSpawnTreeCeremony(events.CeremonyPayload{
			SpawnID: name,
			Caste:   caste,
			Name:    name,
			Task:    task,
			Status:  "spawned",
		})

		result := map[string]interface{}{
			"recorded": true,
			"parent":   parent,
			"caste":    caste,
			"name":     name,
			"task":     task,
			"depth":    depth,
		}
		if eventID != "" {
			result["event_id"] = eventID
		}
		outputOK(result)
		return nil
	},
}

var spawnCompleteCmd = &cobra.Command{
	Use:   "spawn-complete",
	Short: "Mark a spawned agent as completed",
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
		status, _ := cmd.Flags().GetString("status")
		if status == "" {
			status = "completed"
		}
		summary, _ := cmd.Flags().GetString("summary")

		st := agent.NewSpawnTree(store, "spawn-tree.txt")
		if err := st.UpdateStatus(name, status, summary); err != nil {
			outputError(1, fmt.Sprintf("failed to update status: %v", err), nil)
			return nil
		}
		entry := latestSpawnEntryByName(st, name)
		payload := events.CeremonyPayload{
			SpawnID: name,
			Name:    name,
			Status:  status,
			Message: summary,
		}
		if entry != nil {
			payload.Caste = entry.Caste
			payload.Task = entry.Task
		}
		eventID := emitSpawnTreeCeremony(payload)

		result := map[string]interface{}{
			"completed": true,
			"name":      name,
			"status":    status,
		}
		if summary != "" {
			result["summary"] = summary
		}
		if eventID != "" {
			result["event_id"] = eventID
		}
		outputOK(result)
		return nil
	},
}

func emitSpawnTreeCeremony(payload events.CeremonyPayload) string {
	if store == nil {
		return ""
	}
	payload = trimCeremonyPayload(payload)
	raw, err := payload.RawMessage()
	if err != nil {
		return ""
	}
	bus := events.NewBus(store, events.DefaultConfig())
	evt, err := bus.Publish(context.Background(), events.CeremonyTopicBuildSpawn, raw, "aether-spawn")
	if err != nil || evt == nil {
		return ""
	}
	return evt.ID
}

func latestSpawnEntryByName(st *agent.SpawnTree, name string) *agent.SpawnEntry {
	if st == nil {
		return nil
	}
	entries, err := st.Parse()
	if err != nil {
		return nil
	}
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].AgentName == name {
			entry := entries[i]
			return &entry
		}
	}
	return nil
}

var spawnCanSpawnCmd = &cobra.Command{
	Use:   "spawn-can-spawn",
	Short: "Check if spawning is allowed at given depth",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		depth := mustGetInt(cmd, "depth")

		outputOK(map[string]interface{}{
			"can_spawn": true,
			"depth":     depth,
		})
		return nil
	},
}

var spawnTreeLoadCmd = &cobra.Command{
	Use:   "spawn-tree-load",
	Short: "Load and return the full spawn tree as JSON",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		st := agent.NewSpawnTree(store, "spawn-tree.txt")
		data, err := st.ToJSON()
		if err != nil {
			outputError(1, fmt.Sprintf("failed to load spawn tree: %v", err), nil)
			return nil
		}

		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			outputError(1, fmt.Sprintf("failed to parse spawn tree: %v", err), nil)
			return nil
		}

		outputOK(result)
		return nil
	},
}

var spawnTreeActiveCmd = &cobra.Command{
	Use:   "spawn-tree-active",
	Short: "List active (non-completed) spawn entries",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		st := agent.NewSpawnTree(store, "spawn-tree.txt")
		active := st.Active()

		if active == nil {
			active = []agent.SpawnEntry{}
		}

		type entryJSON struct {
			Name      string `json:"name"`
			Parent    string `json:"parent"`
			Caste     string `json:"caste"`
			Task      string `json:"task"`
			Depth     int    `json:"depth"`
			Status    string `json:"status"`
			SpawnedAt string `json:"spawned_at"`
		}

		entries := make([]entryJSON, len(active))
		for i, e := range active {
			entries[i] = entryJSON{
				Name:      e.AgentName,
				Parent:    e.ParentName,
				Caste:     e.Caste,
				Task:      e.Task,
				Depth:     e.Depth,
				Status:    e.Status,
				SpawnedAt: e.Timestamp,
			}
		}

		outputOK(map[string]interface{}{
			"active": entries,
			"count":  len(entries),
		})
		return nil
	},
}

var spawnTreeDepthCmd = &cobra.Command{
	Use:   "spawn-tree-depth",
	Short: "Get the maximum spawn depth",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		st := agent.NewSpawnTree(store, "spawn-tree.txt")
		entries, _ := st.Parse()

		maxDepth := 0
		for _, e := range entries {
			if e.Depth > maxDepth {
				maxDepth = e.Depth
			}
		}

		outputOK(map[string]interface{}{
			"max_depth": maxDepth,
			"total":     len(entries),
		})
		return nil
	},
}

var spawnEfficiencyCmd = &cobra.Command{
	Use:   "spawn-efficiency",
	Short: "Calculate spawn completion efficiency",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		st := agent.NewSpawnTree(store, "spawn-tree.txt")
		entries, _ := st.Parse()

		total := len(entries)
		completed := 0
		for _, e := range entries {
			if e.Status == "completed" || e.Status == "failed" || e.Status == "blocked" {
				completed++
			}
		}

		efficiency := float64(0)
		if total > 0 {
			efficiency = float64(completed) / float64(total) * 100
		}

		outputOK(map[string]interface{}{
			"total":      total,
			"completed":  completed,
			"active":     total - completed,
			"efficiency": fmt.Sprintf("%.1f%%", efficiency),
		})
		return nil
	},
}

var validateWorkerResponseCmd = &cobra.Command{
	Use:   "validate-worker-response",
	Short: "Validate a worker response for basic correctness",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		response := mustGetString(cmd, "response")
		if response == "" {
			return nil
		}

		valid := true
		reason := ""

		if len(response) < 10 {
			valid = false
			reason = "response too short"
		}

		// Check if it's expected to be JSON
		checkJSON, _ := cmd.Flags().GetBool("expect-json")
		if checkJSON && !json.Valid([]byte(response)) {
			valid = false
			reason = "expected valid JSON"
		}

		outputOK(map[string]interface{}{
			"valid":  valid,
			"reason": reason,
		})
		return nil
	},
}

func init() {
	spawnLogCmd.Flags().String("parent", "", "Parent agent name (required)")
	spawnLogCmd.Flags().String("caste", "", "Agent caste (required)")
	spawnLogCmd.Flags().String("name", "", "Agent name (required)")
	spawnLogCmd.Flags().String("id", "", "Legacy alias for child agent name")
	spawnLogCmd.Flags().String("task", "", "Task description (required)")
	spawnLogCmd.Flags().String("description", "", "Legacy alias for task description")
	spawnLogCmd.Flags().Int("depth", 0, "Spawn depth (required)")

	spawnCompleteCmd.Flags().String("name", "", "Agent name to complete (required)")
	spawnCompleteCmd.Flags().String("status", "", "Status: completed, failed, blocked (default: completed)")
	spawnCompleteCmd.Flags().String("summary", "", "Completion summary (optional)")

	spawnCanSpawnCmd.Flags().Int("depth", 0, "Spawn depth to check (required)")

	validateWorkerResponseCmd.Flags().String("response", "", "Response to validate (required)")
	validateWorkerResponseCmd.Flags().Bool("expect-json", false, "Check if response is valid JSON")

	rootCmd.AddCommand(spawnLogCmd)
	rootCmd.AddCommand(spawnCompleteCmd)
	rootCmd.AddCommand(spawnCanSpawnCmd)
	rootCmd.AddCommand(spawnTreeLoadCmd)
	rootCmd.AddCommand(spawnTreeActiveCmd)
	rootCmd.AddCommand(spawnTreeDepthCmd)
	rootCmd.AddCommand(spawnEfficiencyCmd)
	rootCmd.AddCommand(validateWorkerResponseCmd)
}
