package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

// Swarm types for findings and display state.

type swarmFinding struct {
	Agent   string `json:"agent"`
	Finding string `json:"finding"`
}

type swarmFindingsFile struct {
	SwarmID  string         `json:"swarm_id"`
	Findings []swarmFinding `json:"findings"`
	Solution string         `json:"solution,omitempty"`
}

type swarmAgentStatus struct {
	Agent  string `json:"agent"`
	Status string `json:"status"`
}

type swarmDisplayFile struct {
	SwarmID string             `json:"swarm_id"`
	Agents  []swarmAgentStatus `json:"agents"`
}

type swarmTimingFile struct {
	SwarmID string `json:"swarm_id"`
	StartAt string `json:"start_at"`
}

// --- swarm-findings-init ---

var swarmFindingsInitCmd = &cobra.Command{
	Use:   "swarm-findings-init",
	Short: "Create an empty findings file for a swarm",
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

		path := fmt.Sprintf("swarms/%s/findings.json", id)
		if err := store.SaveJSON(path, swarmFindingsFile{SwarmID: id, Findings: []swarmFinding{}}); err != nil {
			outputError(2, fmt.Sprintf("failed to create findings: %v", err), nil)
			return nil
		}
		outputOK(map[string]interface{}{"created": true, "swarm_id": id})
		return nil
	},
}

// --- swarm-findings-add ---

var swarmFindingsAddCmd = &cobra.Command{
	Use:   "swarm-findings-add",
	Short: "Append a finding to a swarm",
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
		agent := mustGetString(cmd, "agent")
		if agent == "" {
			return nil
		}
		finding := mustGetString(cmd, "finding")
		if finding == "" {
			return nil
		}

		path := fmt.Sprintf("swarms/%s/findings.json", id)
		var ff swarmFindingsFile
		if err := store.LoadJSON(path, &ff); err != nil {
			outputError(1, fmt.Sprintf("findings not found for swarm %s: %v", id, err), nil)
			return nil
		}
		ff.Findings = append(ff.Findings, swarmFinding{Agent: agent, Finding: finding})
		if err := store.SaveJSON(path, ff); err != nil {
			outputError(2, fmt.Sprintf("failed to save: %v", err), nil)
			return nil
		}
		outputOK(map[string]interface{}{"added": true, "swarm_id": id, "agent": agent, "total": len(ff.Findings)})
		return nil
	},
}

// --- swarm-findings-read ---

var swarmFindingsReadCmd = &cobra.Command{
	Use:   "swarm-findings-read",
	Short: "Return all findings for a swarm",
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

		path := fmt.Sprintf("swarms/%s/findings.json", id)
		var ff swarmFindingsFile
		if err := store.LoadJSON(path, &ff); err != nil {
			outputError(1, fmt.Sprintf("findings not found for swarm %s: %v", id, err), nil)
			return nil
		}
		outputOK(map[string]interface{}{
			"swarm_id": id,
			"findings": ff.Findings,
			"solution": ff.Solution,
			"total":    len(ff.Findings),
		})
		return nil
	},
}

// --- swarm-solution-set ---

var swarmSolutionSetCmd = &cobra.Command{
	Use:   "swarm-solution-set",
	Short: "Set the solution for a swarm",
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
		solution := mustGetString(cmd, "solution")
		if solution == "" {
			return nil
		}

		path := fmt.Sprintf("swarms/%s/findings.json", id)
		var ff swarmFindingsFile
		if err := store.LoadJSON(path, &ff); err != nil {
			outputError(1, fmt.Sprintf("findings not found for swarm %s: %v", id, err), nil)
			return nil
		}
		ff.Solution = solution
		if err := store.SaveJSON(path, ff); err != nil {
			outputError(2, fmt.Sprintf("failed to save: %v", err), nil)
			return nil
		}
		outputOK(map[string]interface{}{"set": true, "swarm_id": id})
		return nil
	},
}

// --- swarm-cleanup ---

var swarmCleanupCmd = &cobra.Command{
	Use:   "swarm-cleanup",
	Short: "Remove swarm data files",
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

		// Actually remove the swarm's data directory. The previous version
		// loaded a file "to check existence", admitted in a comment that the
		// store has no delete, then reported cleaned:true having removed
		// nothing. A cleanup report must describe the filesystem, not the
		// intention.
		swarmDirRel := fmt.Sprintf("swarms/%s", id)
		swarmDirAbs := filepath.Join(store.BasePath(), "swarms", id)

		existedBefore := false
		if _, err := os.Stat(swarmDirAbs); err == nil {
			existedBefore = true
		}

		if existedBefore {
			if err := os.RemoveAll(swarmDirAbs); err != nil {
				outputError(2, fmt.Sprintf("failed to remove swarm data at %s: %v", swarmDirRel, err), nil)
				return nil
			}
		}

		// Verify against the filesystem before claiming success.
		if _, err := os.Stat(swarmDirAbs); err == nil {
			outputError(2, fmt.Sprintf("swarm data at %s still present after removal attempt", swarmDirRel), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"cleaned":  existedBefore,
			"existed":  existedBefore,
			"swarm_id": id,
			"dir":      swarmDirRel,
		})
		return nil
	},
}

// --- swarm-display-init ---

var swarmDisplayInitCmd = &cobra.Command{
	Use:   "swarm-display-init",
	Short: "Initialize display state for a swarm",
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

		path := fmt.Sprintf("swarms/%s/display.json", id)
		if err := store.SaveJSON(path, swarmDisplayFile{SwarmID: id, Agents: []swarmAgentStatus{}}); err != nil {
			outputError(2, fmt.Sprintf("failed to init display: %v", err), nil)
			return nil
		}
		outputOK(map[string]interface{}{"initialized": true, "swarm_id": id})
		return nil
	},
}

// --- swarm-display-update ---

var swarmDisplayUpdateCmd = &cobra.Command{
	Use:   "swarm-display-update",
	Short: "Update agent status in swarm display",
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
		agent := mustGetString(cmd, "agent")
		if agent == "" {
			return nil
		}
		status := mustGetString(cmd, "status")
		if status == "" {
			return nil
		}

		path := fmt.Sprintf("swarms/%s/display.json", id)
		var df swarmDisplayFile
		if err := store.LoadJSON(path, &df); err != nil {
			outputError(1, fmt.Sprintf("display not found for swarm %s: %v", id, err), nil)
			return nil
		}

		found := false
		for i, a := range df.Agents {
			if a.Agent == agent {
				df.Agents[i].Status = status
				found = true
				break
			}
		}
		if !found {
			df.Agents = append(df.Agents, swarmAgentStatus{Agent: agent, Status: status})
		}

		if err := store.SaveJSON(path, df); err != nil {
			outputError(2, fmt.Sprintf("failed to save: %v", err), nil)
			return nil
		}
		outputOK(map[string]interface{}{"updated": true, "swarm_id": id, "agent": agent, "status": status})
		return nil
	},
}

// --- swarm-timing-start ---

var swarmTimingStartCmd = &cobra.Command{
	Use:   "swarm-timing-start",
	Short: "Record start time for a swarm",
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

		path := fmt.Sprintf("swarms/%s/timing.json", id)
		now := time.Now().UTC().Format(time.RFC3339)
		if err := store.SaveJSON(path, swarmTimingFile{SwarmID: id, StartAt: now}); err != nil {
			outputError(2, fmt.Sprintf("failed to save timing: %v", err), nil)
			return nil
		}
		outputOK(map[string]interface{}{"started": true, "swarm_id": id, "start_at": now})
		return nil
	},
}

// --- swarm-timing-get ---

var swarmTimingGetCmd = &cobra.Command{
	Use:   "swarm-timing-get",
	Short: "Return elapsed time for a swarm",
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

		path := fmt.Sprintf("swarms/%s/timing.json", id)
		var tf swarmTimingFile
		if err := store.LoadJSON(path, &tf); err != nil {
			outputError(1, fmt.Sprintf("timing not found for swarm %s: %v", id, err), nil)
			return nil
		}

		start, err := time.Parse(time.RFC3339, tf.StartAt)
		if err != nil {
			outputError(1, fmt.Sprintf("invalid start time: %v", err), nil)
			return nil
		}
		elapsed := time.Since(start).Seconds()

		outputOK(map[string]interface{}{
			"swarm_id":        id,
			"start_at":        tf.StartAt,
			"elapsed_seconds": elapsed,
		})
		return nil
	},
}

// --- swarm-timing-eta ---

var swarmTimingEtaCmd = &cobra.Command{
	Use:   "swarm-timing-eta",
	Short: "Compute estimated completion time for a swarm",
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
		progress := mustGetFloat64(cmd, "progress")

		path := fmt.Sprintf("swarms/%s/timing.json", id)
		var tf swarmTimingFile
		if err := store.LoadJSON(path, &tf); err != nil {
			outputError(1, fmt.Sprintf("timing not found for swarm %s: %v", id, err), nil)
			return nil
		}

		start, err := time.Parse(time.RFC3339, tf.StartAt)
		if err != nil {
			outputError(1, fmt.Sprintf("invalid start time: %v", err), nil)
			return nil
		}

		elapsed := time.Since(start).Seconds()
		var etaSeconds float64
		if progress > 0 {
			etaSeconds = elapsed / progress * (1 - progress)
		} else {
			etaSeconds = 0
		}
		etaTime := time.Now().Add(time.Duration(etaSeconds) * time.Second).UTC().Format(time.RFC3339)

		outputOK(map[string]interface{}{
			"swarm_id":        id,
			"elapsed_seconds": elapsed,
			"progress":        progress,
			"eta_seconds":     etaSeconds,
			"eta_at":          etaTime,
		})
		return nil
	},
}

func init() {
	for _, c := range []*cobra.Command{
		swarmFindingsInitCmd, swarmFindingsAddCmd, swarmFindingsReadCmd,
		swarmSolutionSetCmd, swarmCleanupCmd,
		swarmDisplayInitCmd, swarmDisplayUpdateCmd,
		swarmTimingStartCmd, swarmTimingGetCmd, swarmTimingEtaCmd,
	} {
		c.Flags().String("id", "", "Swarm ID (required)")
	}
	swarmFindingsAddCmd.Flags().String("agent", "", "Agent name (required)")
	swarmFindingsAddCmd.Flags().String("finding", "", "Finding text (required)")
	swarmSolutionSetCmd.Flags().String("solution", "", "Solution text (required)")
	swarmDisplayUpdateCmd.Flags().String("agent", "", "Agent name (required)")
	swarmDisplayUpdateCmd.Flags().String("status", "", "Status (required)")
	swarmTimingEtaCmd.Flags().Float64("progress", 0, "Progress fraction 0.0-1.0 (required)")

	for _, c := range []*cobra.Command{
		swarmFindingsInitCmd, swarmFindingsAddCmd, swarmFindingsReadCmd,
		swarmSolutionSetCmd, swarmCleanupCmd,
		swarmDisplayInitCmd, swarmDisplayUpdateCmd,
		swarmTimingStartCmd, swarmTimingGetCmd, swarmTimingEtaCmd,
	} {
		rootCmd.AddCommand(c)
	}
}
