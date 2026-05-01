package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/calcosmic/Aether/pkg/agent/curation"
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/graph"
	"github.com/calcosmic/Aether/pkg/learn"
	"github.com/spf13/cobra"
)

func pipelineConfigForStore() learn.PipelineConfig {
	colonyName := "unknown"
	if store != nil {
		var state struct {
			ColonyName string `json:"colony_name"`
		}
		if err := store.LoadJSON("COLONY_STATE.json", &state); err == nil && state.ColonyName != "" {
			colonyName = state.ColonyName
		}
	}
	return learn.PipelineConfig{
		ColonyName: colonyName,
		QueenPath:  "QUEEN.md",
	}
}

var (
	graphSource         string
	graphTarget         string
	graphRelationship   string
	graphWeight         float64
	graphNode           string
	graphDirection      string
	graphFilterRel      string
	graphMinWeight      float64
	graphMaxHops        int
	consolidationDryRun bool
)

// ============================================================================
// Graph commands (4)
// ============================================================================

var graphLinkCmd = &cobra.Command{
	Use:   "graph-link",
	Short: "Create an edge between two nodes in the instinct graph",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		source := mustGetString(cmd, "source")
		target := mustGetString(cmd, "target")
		relationship := mustGetString(cmd, "relationship")
		if source == "" || target == "" || relationship == "" {
			return nil
		}
		weight, _ := cmd.Flags().GetFloat64("weight")
		now := time.Now().UTC().Format("2006-01-02T15:04:05Z")

		g := graph.NewGraph()
		if err := g.LoadFromStore(store, "instinct-graph.json"); err != nil {
			// File may not exist yet — that's OK, we'll create it the edge anyway
		}

		edge, status, err := g.AddEdge(source, target, graph.EdgeType(relationship), weight, now)
		if err != nil {
			outputError(1, fmt.Sprintf("failed to add edge: %v", err), nil)
			return nil
		}

		if err := g.SaveToStore(store, "instinct-graph.json"); err != nil {
			outputError(1, fmt.Sprintf("failed to save graph: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"edge_id":      edge.ID,
			"source":       edge.Source,
			"target":       edge.Target,
			"relationship": edge.Relationship,
			"weight":       edge.Weight,
			"status":       status,
		})
		return nil
	},
}

var graphNeighborsCmd = &cobra.Command{
	Use:   "graph-neighbors",
	Short: "Query neighbors of a node in the instinct graph",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		node := mustGetString(cmd, "node")
		if node == "" {
			return nil
		}
		direction, _ := cmd.Flags().GetString("direction")
		filterRel, _ := cmd.Flags().GetString("relationship")
		minWeight, _ := cmd.Flags().GetFloat64("min-weight")

		g := graph.NewGraph()
		if err := g.LoadFromStore(store, "instinct-graph.json"); err != nil {
			outputError(1, fmt.Sprintf("failed to load graph: %v", err), nil)
			return nil
		}

		result, err := g.Neighbors(node, direction, graph.EdgeType(filterRel), minWeight)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"neighbors": result.Neighbors,
			"count":     result.Count,
		})
		return nil
	},
}

var graphReachCmd = &cobra.Command{
	Use:   "graph-reach",
	Short: "Find all nodes reachable from a node within max hops",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		node := mustGetString(cmd, "node")
		if node == "" {
			return nil
		}
		maxHops, _ := cmd.Flags().GetInt("max-hops")
		minWeight, _ := cmd.Flags().GetFloat64("min-weight")

		g := graph.NewGraph()
		if err := g.LoadFromStore(store, "instinct-graph.json"); err != nil {
			outputError(1, fmt.Sprintf("failed to load graph: %v", err), nil)
			return nil
		}

		result, err := g.Reach(node, maxHops, minWeight)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"reachable":     result.Reachable,
			"count":         result.Count,
			"hops_searched": result.HopsSearched,
		})
		return nil
	},
}

var graphClusterCmd = &cobra.Command{
	Use:   "graph-cluster",
	Short: "Detect cycles in the instinct graph",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		g := graph.NewGraph()
		if err := g.LoadFromStore(store, "instinct-graph.json"); err != nil {
			// No graph file means no cycles
			outputOK(map[string]interface{}{
				"cycles": [][]string{},
				"count":  0,
			})
			return nil
		}

		cycles := g.DetectCycles()
		outputOK(map[string]interface{}{
			"cycles": cycles,
			"count":  len(cycles),
		})
		return nil
	},
}

// ============================================================================
// Consolidation commands (2)
// ============================================================================

var consolidationPhaseEndCmd = &cobra.Command{
	Use:   "consolidation-phase-end",
	Short: "Run phase-end consolidation: decay, archive, check promotions",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")

		bus := events.NewBus(store, events.DefaultConfig())
		pipeline := learn.NewPipeline(store, bus, pipelineConfigForStore())

		ctx, cancel := timeoutCtx(cmd)
		defer cancel()

		var (
			result *learn.ConsolidationResult
			err    error
		)
		if dryRun {
			service := learn.NewConsolidationService(store, bus, "QUEEN.md", pipelineConfigForStore().ColonyName)
			result, err = service.Run(ctx)
		} else {
			result, err = pipeline.RunConsolidation(ctx)
		}
		if err != nil {
			outputError(1, fmt.Sprintf("consolidation failed: %v", err), nil)
			return nil
		}

		if dryRun {
			outputOK(map[string]interface{}{
				"type":                 "phase_end",
				"instincts_decayed":    result.InstinctsDecayed,
				"instincts_archived":   result.InstinctsArchived,
				"observations_decayed": result.ObservationsDecayed,
				"promotion_candidates": len(result.PromotionCandidates),
				"queen_eligible":       len(result.QueenEligible),
				"review_candidates":    len(result.ReviewCandidates),
				"reread_candidates":    len(result.RereadCandidates),
				"dry_run":              true,
			})
			return nil
		}

		outputOK(map[string]interface{}{
			"type":                 "phase_end",
			"instincts_decayed":    result.InstinctsDecayed,
			"instincts_archived":   result.InstinctsArchived,
			"observations_decayed": result.ObservationsDecayed,
			"promotion_candidates": len(result.PromotionCandidates),
			"queen_eligible":       len(result.QueenEligible),
			"review_candidates":    len(result.ReviewCandidates),
			"reread_candidates":    len(result.RereadCandidates),
		})
		return nil
	},
}

var consolidationSealCmd = &cobra.Command{
	Use:   "consolidation-seal",
	Short: "Run full seal consolidation: curation + decay + archive + event",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		bus := events.NewBus(store, events.DefaultConfig())
		ctx, cancel := timeoutCtx(cmd)
		defer cancel()

		type stepInfo struct {
			Name    string `json:"name"`
			Success bool   `json:"success"`
			Summary string `json:"summary,omitempty"`
		}

		var steps []stepInfo

		// Step 1: Run curation orchestrator
		orch := curation.NewOrchestrator(store, bus)
		curResult, err := orch.Run(ctx, dryRun)
		if err != nil {
			steps = append(steps, stepInfo{Name: "curation", Success: false, Summary: err.Error()})
		} else {
			steps = append(steps, stepInfo{
				Name:    "curation",
				Success: true,
				Summary: fmt.Sprintf("succeeded=%d failed=%d", curResult.Succeeded, curResult.Failed),
			})
		}

		// Step 2: Run consolidation (decay + archive)
		pipeline := learn.NewPipeline(store, bus, pipelineConfigForStore())
		consResult, err := pipeline.RunConsolidation(ctx)
		if err != nil {
			steps = append(steps, stepInfo{Name: "consolidation", Success: false, Summary: err.Error()})
		} else {
			steps = append(steps, stepInfo{
				Name:    "consolidation",
				Success: true,
				Summary: fmt.Sprintf("decayed=%d archived=%d review=%d reread=%d", consResult.InstinctsDecayed, consResult.InstinctsArchived, len(consResult.ReviewCandidates), len(consResult.RereadCandidates)),
			})
		}

		// Step 3: Publish seal event
		eventPublished := false
		if !dryRun {
			payload, _ := json.Marshal(map[string]string{
				"type":      "consolidation.seal",
				"timestamp": time.Now().UTC().Format("2006-01-02T15:04:05Z"),
			})
			_, pubErr := bus.Publish(ctx, "consolidation.seal", payload, "seal")
			eventPublished = pubErr == nil
		}

		outputOK(map[string]interface{}{
			"type":            "seal",
			"steps":           steps,
			"event_published": eventPublished,
			"dry_run":         dryRun,
		})
		return nil
	},
}

func init() {
	// graph-link
	rootCmd.AddCommand(graphLinkCmd)
	graphLinkCmd.Flags().StringVar(&graphSource, "source", "", "Source node ID (required)")
	graphLinkCmd.Flags().StringVar(&graphTarget, "target", "", "Target node ID (required)")
	graphLinkCmd.Flags().StringVar(&graphRelationship, "relationship", "", "Edge relationship type (required)")
	graphLinkCmd.Flags().Float64Var(&graphWeight, "weight", 1.0, "Edge weight")

	// graph-neighbors
	rootCmd.AddCommand(graphNeighborsCmd)
	graphNeighborsCmd.Flags().StringVar(&graphNode, "node", "", "Node ID (required)")
	graphNeighborsCmd.Flags().StringVar(&graphDirection, "direction", "both", "Direction: out, in, or both")
	graphNeighborsCmd.Flags().StringVar(&graphFilterRel, "relationship", "", "Filter by relationship type")
	graphNeighborsCmd.Flags().Float64Var(&graphMinWeight, "min-weight", 0, "Minimum edge weight")

	// graph-reach
	rootCmd.AddCommand(graphReachCmd)
	graphReachCmd.Flags().StringVar(&graphNode, "node", "", "Start node ID (required)")
	graphReachCmd.Flags().IntVar(&graphMaxHops, "max-hops", 3, "Maximum hops to traverse")
	graphReachCmd.Flags().Float64Var(&graphMinWeight, "min-weight", 0, "Minimum edge weight")

	// graph-cluster
	rootCmd.AddCommand(graphClusterCmd)

	// consolidation-phase-end
	rootCmd.AddCommand(consolidationPhaseEndCmd)
	consolidationPhaseEndCmd.Flags().BoolVar(&consolidationDryRun, "dry-run", false, "Report without modifying")

	// consolidation-seal
	rootCmd.AddCommand(consolidationSealCmd)
	consolidationSealCmd.Flags().BoolVar(&consolidationDryRun, "dry-run", false, "Dry run mode")
}
