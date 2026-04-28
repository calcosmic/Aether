package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

// Council system manages multi-perspective deliberation.

type councilPosition struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	At      string `json:"at"`
}

type councilDeliberation struct {
	Topic     string            `json:"topic"`
	CreatedAt string            `json:"created_at"`
	Positions []councilPosition `json:"positions"`
}

type councilHistoryData struct {
	Deliberations []councilDeliberation `json:"deliberations"`
}

const maxDeliberations = 50
const maxPositionsPerTopic = 20

// --- council-deliberate ---

var councilDeliberateCmd = &cobra.Command{
	Use:   "council-deliberate",
	Short: "Initiate council deliberation on a topic",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}
		topic := mustGetString(cmd, "topic")
		if topic == "" {
			return nil
		}

		var ch councilHistoryData
		if err := store.LoadJSON("council/history.json", &ch); err != nil {
			ch = councilHistoryData{}
		}

		// Check if topic already exists
		for _, d := range ch.Deliberations {
			if d.Topic == topic {
				outputOK(map[string]interface{}{
					"initiated": false,
					"reason":    "topic already exists",
					"topic":     topic,
					"positions": len(d.Positions),
				})
				return nil
			}
		}

		d := councilDeliberation{
			Topic:     topic,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
			Positions: []councilPosition{},
		}

		ch.Deliberations = append(ch.Deliberations, d)
		if len(ch.Deliberations) > maxDeliberations {
			ch.Deliberations = ch.Deliberations[len(ch.Deliberations)-maxDeliberations:]
		}

		if err := store.SaveJSON("council/history.json", ch); err != nil {
			outputError(2, fmt.Sprintf("failed to save: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{"initiated": true, "topic": topic})
		return nil
	},
}

// --- council-advocate ---

var councilAdvocateCmd = &cobra.Command{
	Use:   "council-advocate",
	Short: "Submit advocate position on a topic",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}
		topic := mustGetString(cmd, "topic")
		if topic == "" {
			return nil
		}
		position := mustGetString(cmd, "position")
		if position == "" {
			return nil
		}

		if err := addCouncilPosition(topic, "advocate", position); err != nil {
			outputError(2, fmt.Sprintf("%v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{"submitted": true, "role": "advocate", "topic": topic})
		return nil
	},
}

// --- council-challenger ---

var councilChallengerCmd = &cobra.Command{
	Use:   "council-challenger",
	Short: "Submit challenger position on a topic",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}
		topic := mustGetString(cmd, "topic")
		if topic == "" {
			return nil
		}
		challenge := mustGetString(cmd, "challenge")
		if challenge == "" {
			return nil
		}

		if err := addCouncilPosition(topic, "challenger", challenge); err != nil {
			outputError(2, fmt.Sprintf("%v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{"submitted": true, "role": "challenger", "topic": topic})
		return nil
	},
}

// --- council-sage ---

var councilSageCmd = &cobra.Command{
	Use:   "council-sage",
	Short: "Submit sage advice on a topic",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}
		topic := mustGetString(cmd, "topic")
		if topic == "" {
			return nil
		}
		wisdom := mustGetString(cmd, "wisdom")
		if wisdom == "" {
			return nil
		}

		if err := addCouncilPosition(topic, "sage", wisdom); err != nil {
			outputError(2, fmt.Sprintf("%v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{"submitted": true, "role": "sage", "topic": topic})
		return nil
	},
}

func addCouncilPosition(topic, role, content string) error {
	var ch councilHistoryData
	if err := store.LoadJSON("council/history.json", &ch); err != nil {
		ch = councilHistoryData{}
	}

	found := false
	for i, d := range ch.Deliberations {
		if d.Topic == topic {
			pos := councilPosition{
				Role:    role,
				Content: content,
				At:      time.Now().UTC().Format(time.RFC3339),
			}
			ch.Deliberations[i].Positions = append(ch.Deliberations[i].Positions, pos)
			if len(ch.Deliberations[i].Positions) > maxPositionsPerTopic {
				ch.Deliberations[i].Positions = ch.Deliberations[i].Positions[len(ch.Deliberations[i].Positions)-maxPositionsPerTopic:]
			}
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("topic %q not found", topic)
	}

	return store.SaveJSON("council/history.json", ch)
}

// --- council-history ---

var councilHistoryCmd = &cobra.Command{
	Use:   "council-history",
	Short: "Return council deliberation history",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}
		topicFilter, _ := cmd.Flags().GetString("topic")

		var ch councilHistoryData
		if err := store.LoadJSON("council/history.json", &ch); err != nil {
			outputOK(map[string]interface{}{"deliberations": []councilDeliberation{}, "total": 0})
			return nil
		}

		if topicFilter != "" {
			var filtered []councilDeliberation
			for _, d := range ch.Deliberations {
				if d.Topic == topicFilter {
					filtered = append(filtered, d)
				}
			}
			outputOK(map[string]interface{}{"deliberations": filtered, "total": len(filtered), "filtered_by": topicFilter})
			return nil
		}

		outputOK(map[string]interface{}{"deliberations": ch.Deliberations, "total": len(ch.Deliberations)})
		return nil
	},
}

// --- council-budget-check ---

var councilBudgetCheckCmd = &cobra.Command{
	Use:   "council-budget-check",
	Short: "Return remaining budget for council deliberations",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		var ch councilHistoryData
		if err := store.LoadJSON("council/history.json", &ch); err != nil {
			ch = councilHistoryData{}
		}

		totalPositions := 0
		for _, d := range ch.Deliberations {
			totalPositions += len(d.Positions)
		}

		maxBudget := maxDeliberations * maxPositionsPerTopic
		remaining := maxBudget - totalPositions
		if remaining < 0 {
			remaining = 0
		}

		outputOK(map[string]interface{}{
			"topics":     len(ch.Deliberations),
			"positions":  totalPositions,
			"max_budget": maxBudget,
			"remaining":  remaining,
		})
		return nil
	},
}

// councilCmd is the parent command for council subcommands.
// Subcommands are registered both under rootCmd (as council-deliberate etc.)
// and under councilCmd (as council deliberate etc.) for flexible invocation.
var councilCmd = &cobra.Command{
	Use:   "council",
	Short: "Multi-perspective deliberation system",
}

func init() {
	councilDeliberateCmd.Flags().String("topic", "", "Deliberation topic (required)")
	councilAdvocateCmd.Flags().String("topic", "", "Topic (required)")
	councilAdvocateCmd.Flags().String("position", "", "Advocate position (required)")
	councilChallengerCmd.Flags().String("topic", "", "Topic (required)")
	councilChallengerCmd.Flags().String("challenge", "", "Challenge text (required)")
	councilSageCmd.Flags().String("topic", "", "Topic (required)")
	councilSageCmd.Flags().String("wisdom", "", "Sage wisdom (required)")
	councilHistoryCmd.Flags().String("topic", "", "Filter by topic")

	councilSubcommands := []*cobra.Command{
		councilDeliberateCmd, councilAdvocateCmd, councilChallengerCmd,
		councilSageCmd, councilHistoryCmd, councilBudgetCheckCmd,
	}

	// Register as direct subcommands of rootCmd (e.g., aether council-deliberate)
	for _, c := range councilSubcommands {
		rootCmd.AddCommand(c)
	}

	// Also register under parent councilCmd (e.g., aether council deliberate)
	for _, c := range councilSubcommands {
		councilCmd.AddCommand(c)
	}
	rootCmd.AddCommand(councilCmd)
}
