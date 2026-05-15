package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/spf13/cobra"
)

// oracleState is the persisted Oracle iteration state.
type oracleState struct {
	Topic             string              `json:"topic"`
	Depth             string              `json:"depth"`
	MaxIterations     int                 `json:"max_iterations"`
	ConfidenceTarget  int                 `json:"confidence_target"`
	CurrentIteration  int                 `json:"current_iteration"`
	CurrentConfidence int                 `json:"current_confidence"`
	History           []oracleHistoryEntry `json:"history"`
	ShouldContinue    bool                `json:"should_continue"`
	UpdatedAt         string              `json:"updated_at"`
	// Interrupt recovery fields
	PendingIteration  int    `json:"pending_iteration,omitempty"`
	PendingStartTime  string `json:"pending_start_time,omitempty"`
	LastWorkerStatus  string `json:"last_worker_status,omitempty"`
}

type oracleHistoryEntry struct {
	Iteration int    `json:"iteration"`
	Confidence int   `json:"confidence"`
	Summary   string `json:"summary"`
}

// iterationManifest is returned by oracle-iterate --plan-only.
type iterationManifest struct {
	Topic            string         `json:"topic"`
	Depth            string         `json:"depth"`
	MaxIterations    int            `json:"max_iterations"`
	ConfidenceTarget int            `json:"confidence_target"`
	CurrentIteration int            `json:"current_iteration"`
	Workers          []oracleWorker `json:"workers"`
	Resuming         bool           `json:"resuming,omitempty"`
}

type oracleWorker struct {
	Name  string `json:"name"`
	Caste string `json:"caste"`
	Task  string `json:"task"`
	Brief string `json:"brief"`
}

// oracleIterationResult is the top-level JSON envelope from oracle-iterate.
type oracleIterationResult struct {
	OK                bool              `json:"ok"`
	IterationManifest iterationManifest `json:"iteration_manifest"`
}

// oracleFinalizeResult is the top-level JSON envelope from oracle-iterate-finalize.
type oracleFinalizeResult struct {
	OK               bool   `json:"ok"`
	StatePath        string `json:"state_path"`
	CurrentConfidence int   `json:"current_confidence"`
	ConfidenceTarget  int   `json:"confidence_target"`
	ShouldContinue    bool   `json:"should_continue"`
	NextCommand       string `json:"next_command"`
}

var oracleIterateCmd = &cobra.Command{
	Use:   "oracle-iterate",
	Short: "Generate an Oracle iteration manifest",
	Long:  "Returns a manifest for the next Oracle research iteration. Go owns state and confidence math.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		planOnly, _ := cmd.Flags().GetBool("plan-only")
		topic, _ := cmd.Flags().GetString("topic")
		depth, _ := cmd.Flags().GetString("depth")
		confidenceTarget, _ := cmd.Flags().GetInt("confidence-target")
		maxIterations, _ := cmd.Flags().GetInt("max-iterations")

		if topic == "" {
			outputError(1, "flag --topic is required", nil)
			return nil
		}

		depthCfg := resolveOracleDepth(depth)
		if confidenceTarget <= 0 || confidenceTarget > 100 {
			confidenceTarget = depthCfg.TargetConfidence
		}
		if maxIterations <= 0 {
			maxIterations = depthCfg.MaxIterations
		}

		state, err := loadOracleState()
		if err != nil {
			state = &oracleState{
				Topic:            topic,
				Depth:            depthCfg.Label,
				MaxIterations:    maxIterations,
				ConfidenceTarget: confidenceTarget,
				CurrentIteration: 1,
				ShouldContinue:   true,
			}
		} else {
			// Resume existing session if topic matches
			if state.Topic != topic {
				// Start fresh for new topic
				state = &oracleState{
					Topic:            topic,
					Depth:            depthCfg.Label,
					MaxIterations:    maxIterations,
					ConfidenceTarget: confidenceTarget,
					CurrentIteration: 1,
					ShouldContinue:   true,
				}
			}
		}

		// Interrupt recovery: detect and clear stale pending iterations.
		resuming := false
		if state.PendingIteration > 0 {
			if isPendingStale(state.PendingStartTime) {
				state.PendingIteration = 0
				state.PendingStartTime = ""
				state.LastWorkerStatus = ""
			} else if state.PendingIteration == state.CurrentIteration {
				resuming = true
			}
		}

		if planOnly {
			// Write pending marker before returning manifest so interrupt
			// recovery can resume from this point.
			state.PendingIteration = state.CurrentIteration
			state.PendingStartTime = time.Now().UTC().Format(time.RFC3339)
			state.LastWorkerStatus = "dispatched"
			_ = saveOracleState(state)

			manifest := iterationManifest{
				Topic:            state.Topic,
				Depth:            state.Depth,
				MaxIterations:    state.MaxIterations,
				ConfidenceTarget: state.ConfidenceTarget,
				CurrentIteration: state.CurrentIteration,
				Workers: []oracleWorker{
					{
						Name:  fmt.Sprintf("Oracle-%02d", state.CurrentIteration),
						Caste: "oracle",
						Task:  fmt.Sprintf("Research iteration %d on topic: %s", state.CurrentIteration, state.Topic),
						Brief: fmt.Sprintf("Conduct research iteration %d/%d for '%s'. Target confidence: %d%%. Current confidence: %d%%. Return structured findings with confidence delta.", state.CurrentIteration, state.MaxIterations, state.Topic, state.ConfidenceTarget, state.CurrentConfidence),
					},
				},
				Resuming: resuming,
			}
			result := oracleIterationResult{
				OK:                true,
				IterationManifest: manifest,
			}
			outputOK(result)
			return nil
		}

		// Non-plan-only mode: just output the current state as JSON
		outputOK(state)
		return nil
	},
}

var oracleIterateFinalizeCmd = &cobra.Command{
	Use:   "oracle-iterate-finalize",
	Short: "Commit Oracle iteration results and update state",
	Long:  "Reads a completion JSON file, updates Oracle state, and determines whether to continue iterating.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		completionPath, _ := cmd.Flags().GetString("completion-file")
		if completionPath == "" {
			outputError(1, "flag --completion-file is required", nil)
			return nil
		}

		data, err := os.ReadFile(completionPath)
		if err != nil {
			outputError(1, fmt.Sprintf("read completion file: %v", err), nil)
			return nil
		}

		var completion struct {
			IterationManifest iterationManifest    `json:"iteration_manifest"`
			Dispatches        []struct {
				Worker          string `json:"worker"`
				Status          string `json:"status"`
				Summary         string `json:"summary"`
				ConfidenceDelta int    `json:"confidence_delta"`
			} `json:"dispatches"`
			CurrentConfidence int  `json:"current_confidence"`
			CurrentIteration  int  `json:"current_iteration"`
			ShouldContinue    bool `json:"should_continue"`
		}
		if err := json.Unmarshal(data, &completion); err != nil {
			outputError(1, fmt.Sprintf("parse completion file: %v", err), nil)
			return nil
		}

		state, err := loadOracleState()
		if err != nil {
			state = &oracleState{
				Topic:            completion.IterationManifest.Topic,
				Depth:            completion.IterationManifest.Depth,
				MaxIterations:    completion.IterationManifest.MaxIterations,
				ConfidenceTarget: completion.IterationManifest.ConfidenceTarget,
				CurrentIteration: completion.CurrentIteration,
				CurrentConfidence: completion.CurrentConfidence,
				ShouldContinue:   completion.ShouldContinue,
			}
		} else {
			state.CurrentIteration = completion.CurrentIteration
			state.CurrentConfidence = completion.CurrentConfidence
			if completion.IterationManifest.Topic != "" {
				state.Topic = completion.IterationManifest.Topic
			}
		}

		// Build history from dispatches
		for _, d := range completion.Dispatches {
			if d.Status == "completed" {
				state.History = append(state.History, oracleHistoryEntry{
					Iteration: state.CurrentIteration,
					Confidence: state.CurrentConfidence,
					Summary:   d.Summary,
				})
			}
		}

		// Auto-determine should_continue based on thresholds
		state.ShouldContinue = true
		if state.CurrentConfidence >= state.ConfidenceTarget {
			state.ShouldContinue = false
		}
		if state.CurrentIteration >= state.MaxIterations {
			state.ShouldContinue = false
		}

		// Clear pending iteration marker on successful finalize
		state.PendingIteration = 0
		state.PendingStartTime = ""
		state.LastWorkerStatus = ""

		state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

		if err := saveOracleState(state); err != nil {
			outputError(1, fmt.Sprintf("save oracle state: %v", err), nil)
			return nil
		}

		nextCmd := fmt.Sprintf("aether oracle-iterate --plan-only --topic %s", strconv.Quote(state.Topic))
		result := oracleFinalizeResult{
			OK:                true,
			StatePath:         oracleStatePath(),
			CurrentConfidence: state.CurrentConfidence,
			ConfidenceTarget:  state.ConfidenceTarget,
			ShouldContinue:    state.ShouldContinue,
			NextCommand:       nextCmd,
		}
		outputOK(result)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(oracleIterateCmd)
	rootCmd.AddCommand(oracleIterateFinalizeCmd)

	oracleIterateCmd.Flags().Bool("plan-only", false, "Output iteration manifest JSON only")
	oracleIterateCmd.Flags().String("topic", "", "Oracle research topic (required)")
	oracleIterateCmd.Flags().String("depth", "balanced", "Research depth: quick, balanced, deep, exhaustive")
	oracleIterateCmd.Flags().Int("confidence-target", 0, "Target confidence 1-100 (default from depth config)")
	oracleIterateCmd.Flags().Int("max-iterations", 0, "Max iterations 1-50 (default from depth config)")

	oracleIterateFinalizeCmd.Flags().String("completion-file", "", "Path to completion JSON file (required)")
}

func oracleStatePath() string {
	return filepath.Join(".aether", "data", "oracle", "state.json")
}

func loadOracleState() (*oracleState, error) {
	if store == nil {
		// Fallback to plain file I/O when store is not initialized
		// (e.g., in tests that bypass PersistentPreRunE)
		path := oracleStatePath()
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var state oracleState
		if err := json.Unmarshal(data, &state); err != nil {
			return nil, err
		}
		return &state, nil
	}

	// Store basePath is already .aether/data/, so use the relative path within it.
	data, err := store.ReadFile("oracle/state.json")
	if err != nil {
		return nil, err
	}
	var state oracleState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func saveOracleState(state *oracleState) error {
	if store == nil {
		// Fallback to plain file I/O when store is not initialized
		path := oracleStatePath()
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		data, err := json.MarshalIndent(state, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(path, data, 0644)
	}

	// Store basePath is already .aether/data/, so use the relative path within it.
	return store.SaveJSON("oracle/state.json", state)
}

// isPendingStale returns true if the pending start time is older than 1 hour.
func isPendingStale(startTime string) bool {
	if startTime == "" {
		return false
	}
	t, err := time.Parse(time.RFC3339, startTime)
	if err != nil {
		return false
	}
	return time.Since(t) > time.Hour
}

