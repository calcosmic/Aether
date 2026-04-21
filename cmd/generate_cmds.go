package cmd

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

// castePrefixes maps each caste name to a list of name prefixes.
// This matches the behavior of the aether generate-ant-name command.
var castePrefixes = map[string][]string{
	"builder":       {"Chip", "Hammer", "Forge", "Mason", "Brick", "Anvil", "Weld", "Bolt"},
	"watcher":       {"Vigil", "Sentinel", "Guard", "Keen", "Sharp", "Hawk", "Watch", "Alert"},
	"scout":         {"Swift", "Dash", "Ranger", "Track", "Seek", "Path", "Roam", "Quest"},
	"surveyor":      {"Survey", "Map", "Chart", "Atlas", "Grid", "Plot", "Trace", "Scope"},
	"colonizer":     {"Pioneer", "Map", "Chart", "Venture", "Explore", "Compass", "Atlas", "Trek"},
	"architect":     {"Blueprint", "Draft", "Design", "Plan", "Schema", "Frame", "Sketch", "Model"},
	"prime":         {"Prime", "Alpha", "Lead", "Chief", "First", "Core", "Apex", "Crown"},
	"chaos":         {"Probe", "Stress", "Shake", "Twist", "Snap", "Breach", "Surge", "Jolt"},
	"archaeologist": {"Relic", "Fossil", "Dig", "Shard", "Epoch", "Strata", "Lore", "Glyph"},
	"oracle":        {"Sage", "Seer", "Vision", "Augur", "Mystic", "Sibyl", "Delph", "Pythia"},
	"ambassador":    {"Bridge", "Connect", "Link", "Diplomat", "Protocol", "Network", "Port", "Socket"},
	"auditor":       {"Review", "Inspect", "Exam", "Scrutin", "Verify", "Check", "Audit", "Assess"},
	"chronicler":    {"Record", "Write", "Document", "Chronicle", "Scribe", "Archive", "Script", "Ledger"},
	"gatekeeper":    {"Guard", "Protect", "Secure", "Shield", "Defend", "Bar", "Gate", "Checkpoint"},
	"guardian":      {"Defend", "Patrol", "Watch", "Vigil", "Shield", "Guard", "Armor", "Fort"},
	"includer":      {"Access", "Include", "Open", "Welcome", "Reach", "Universal", "Equal", "A11y"},
	"keeper":        {"Archive", "Store", "Curate", "Preserve", "Guard", "Keep", "Hold", "Save"},
	"measurer":      {"Metric", "Gauge", "Scale", "Measure", "Benchmark", "Track", "Count", "Meter"},
	"probe":         {"Test", "Probe", "Excavat", "Uncover", "Edge", "Mutant", "Trial", "Check"},
	"tracker":       {"Track", "Trace", "Debug", "Hunt", "Follow", "Trail", "Find", "Seek"},
	"weaver":        {"Weave", "Knit", "Spin", "Twine", "Transform", "Mend", "Thread", "Stitch"},
	"medic":         {"Heal", "Mend", "Cure", "Suture", "Salve", "Remedy", "Treat", "Patch"},
}

// defaultPrefixes is used when no caste matches.
var defaultPrefixes = []string{"Ant", "Worker", "Drone", "Toiler", "Marcher", "Runner", "Carrier", "Helper"}

// validCommitTypes are the allowed values for generate-commit-message --type.
var validCommitTypes = []string{"feat", "fix", "docs", "refactor", "test", "chore", "seal", "milestone", "pause", "contextual"}

var generateAntNameCmd = &cobra.Command{
	Use:   "generate-ant-name [caste]",
	Short: "Generate a random ant name for the given caste",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		caste := "builder"
		if len(args) > 0 && args[0] != "" {
			caste = args[0]
		}

		// Use a local random source so --seed produces deterministic output
		var rng *rand.Rand
		seedFlag := cmd.Flags().Lookup("seed")
		if seedFlag != nil && seedFlag.Changed {
			seedVal, _ := cmd.Flags().GetInt64("seed")
			rng = rand.New(rand.NewSource(seedVal))
		} else {
			rng = rand.New(rand.NewSource(rand.Int63()))
		}

		prefixes, ok := castePrefixes[caste]
		if !ok {
			prefixes = defaultPrefixes
		}

		prefix := prefixes[rng.Intn(len(prefixes))]
		number := rng.Intn(99) + 1
		name := fmt.Sprintf("%s-%d", prefix, number)

		outputOK(name)
		return nil
	},
}

var generateCommitMessageCmd = &cobra.Command{
	Use:   "generate-commit-message",
	Short: "Generate a template-based commit message",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		commitType := mustGetString(cmd, "type")
		if commitType == "" {
			return nil
		}
		subject := mustGetString(cmd, "subject")
		if subject == "" {
			return nil
		}
		scope, _ := cmd.Flags().GetString("scope")
		body, _ := cmd.Flags().GetString("body")

		// Validate type
		valid := false
		for _, t := range validCommitTypes {
			if t == commitType {
				valid = true
				break
			}
		}
		if !valid {
			outputError(1, fmt.Sprintf("invalid type %q, must be one of: %s", commitType, strings.Join(validCommitTypes, ", ")), nil)
			return nil
		}

		var message string
		if scope != "" {
			message = fmt.Sprintf("%s(%s): %s", commitType, scope, subject)
		} else {
			message = fmt.Sprintf("%s: %s", commitType, subject)
		}

		if body != "" {
			message += "\n\n" + body
		}

		outputOK(map[string]interface{}{
			"message": message,
		})
		return nil
	},
}

var generateProgressBarCmd = &cobra.Command{
	Use:   "generate-progress-bar",
	Short: "Render a terminal progress bar",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		current := mustGetInt(cmd, "current")
		total := mustGetInt(cmd, "total")
		if total == 0 {
			outputError(1, "total must be greater than 0", nil)
			return nil
		}

		width, _ := cmd.Flags().GetInt("width")
		if width <= 0 {
			width = 30
		}

		// Clamp current to [0, total]
		if current < 0 {
			current = 0
		}
		if current > total {
			current = total
		}

		percentage := current * 100 / total
		filled := width * current / total
		if filled > width {
			filled = width
		}

		bar := "[" + strings.Repeat("#", filled) + strings.Repeat("-", width-filled) + "]"

		outputOK(map[string]interface{}{
			"bar":        bar,
			"percentage": percentage,
			"current":    current,
			"total":      total,
		})
		return nil
	},
}

// generateThresholdBarCmd renders a threshold indicator bar.
var generateThresholdBarCmd = &cobra.Command{
	Use:   "generate-threshold-bar",
	Short: "Render a threshold indicator bar",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		value := mustGetFloat64(cmd, "value")
		maxVal := mustGetFloat64(cmd, "max")
		threshold := mustGetFloat64(cmd, "threshold")
		if maxVal == 0 {
			outputError(1, "max must be greater than 0", nil)
			return nil
		}

		width, _ := cmd.Flags().GetInt("width")
		if width <= 0 {
			width = 30
		}

		// Clamp value to [0, max]
		if value < 0 {
			value = 0
		}
		if value > maxVal {
			value = maxVal
		}

		filled := int(float64(width) * value / maxVal)
		if filled > width {
			filled = width
		}

		thresholdPos := int(float64(width) * threshold / maxVal)
		if thresholdPos > width {
			thresholdPos = width
		}

		bar := ""
		for i := 0; i < width; i++ {
			if i < filled {
				if i == thresholdPos {
					bar += "|"
				} else {
					bar += "#"
				}
			} else if i == thresholdPos {
				bar += "|"
			} else {
				bar += "-"
			}
		}

		percentage := int(value * 100 / maxVal)

		outputOK(map[string]interface{}{
			"bar":        "[" + bar + "]",
			"percentage": percentage,
			"value":      value,
			"threshold":  threshold,
			"max":        maxVal,
			"exceeds":    value > threshold,
		})
		return nil
	},
}

// antNamePattern validates the format of generated ant names.
var antNamePattern = regexp.MustCompile(`^[A-Z][A-Za-z0-9]+-\d{1,2}$`)

func init() {
	generateAntNameCmd.Flags().Int64("seed", 0, "Random seed for deterministic output")

	generateCommitMessageCmd.Flags().String("type", "", "Commit type (required: feat, fix, docs, refactor, test, chore, seal, milestone, pause, contextual)")
	generateCommitMessageCmd.Flags().String("scope", "", "Commit scope (optional)")
	generateCommitMessageCmd.Flags().String("subject", "", "Commit subject (required)")
	generateCommitMessageCmd.Flags().String("body", "", "Commit body (optional)")

	generateProgressBarCmd.Flags().Int("current", 0, "Current progress value (required)")
	generateProgressBarCmd.Flags().Int("total", 0, "Total value (required)")
	generateProgressBarCmd.Flags().Int("width", 30, "Bar width in characters")

	generateThresholdBarCmd.Flags().Float64("value", 0, "Current value")
	generateThresholdBarCmd.Flags().Float64("max", 0, "Maximum value")
	generateThresholdBarCmd.Flags().Float64("threshold", 0, "Threshold value")
	generateThresholdBarCmd.Flags().Int("width", 30, "Bar width in characters")

	rootCmd.AddCommand(generateAntNameCmd)
	rootCmd.AddCommand(generateCommitMessageCmd)
	rootCmd.AddCommand(generateProgressBarCmd)
	rootCmd.AddCommand(generateThresholdBarCmd)
}
