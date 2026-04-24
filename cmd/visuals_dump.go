package cmd

import (
	"sort"

	"github.com/spf13/cobra"
)

type casteVisualContract struct {
	Emoji string `json:"emoji"`
	Color string `json:"color"`
	Label string `json:"label"`
}

var visualsDumpJSON bool

var visualsDumpCmd = &cobra.Command{
	Use:   "visuals-dump",
	Short: "Dump shared visual identity metadata",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = visualsDumpJSON
		outputOK(map[string]interface{}{
			"castes": casteVisualContracts(),
		})
		return nil
	},
}

func casteVisualContracts() map[string]casteVisualContract {
	keys := make([]string, 0, len(casteLabelMap))
	for caste := range casteLabelMap {
		keys = append(keys, caste)
	}
	sort.Strings(keys)

	contracts := make(map[string]casteVisualContract, len(keys))
	for _, caste := range keys {
		contracts[caste] = casteVisualContract{
			Emoji: casteEmojiMap[caste],
			Color: casteColorMap[caste],
			Label: casteLabelMap[caste],
		}
	}
	return contracts
}

func init() {
	rootCmd.AddCommand(visualsDumpCmd)
	visualsDumpCmd.Flags().BoolVar(&visualsDumpJSON, "json", false, "Emit JSON visual metadata")
}
