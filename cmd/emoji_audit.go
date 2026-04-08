package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// commonEmojis is the set of emojis to search for in command files.
var commonEmojis = []string{
	"\U0001F528", // hammer
	"\U0001F41C", // ant
	"\U0001F3D7", // construction
	"\U0001F4CB", // clipboard
	"\u2705",     // check mark
	"\u274C",     // cross mark
	"\u26A0",     // warning
	"\U0001F504", // counterclockwise arrows
	"\U0001F4CD", // round pushpin
	"\U0001F4BE", // floppy disk
	"\U0001F3AF", // direct hit
	"\U0001F4CA", // bar chart
	"\U0001F50D", // magnifying glass
	"\U0001F6E1", // shield
	"\u2139",     // information
	"\U0001F4A1", // light bulb
	"\U0001F680", // rocket
	"\u2B50",     // star
	"\U0001F389", // party popper
	"\U0001F4DD", // memo
	"\U0001F4E6", // package
	"\u2728",     // sparkles
	"\U0001F525", // fire
	"\U0001F916", // robot
	"\U0001F9E0", // brain
	"\U0001F33F", // herb
	"\U0001F40D", // snake
	"\U0001F98B", // butterfly
	"\U0001FAB8", // nest
}

var emojiAuditCmd = &cobra.Command{
	Use:   "emoji-audit",
	Short: "Audit emoji usage in command files",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		patterns := []string{
			".claude/commands/ant/*.md",
			".aether/commands/*.yaml",
		}

		type fileResult struct {
			Path       string `json:"path"`
			EmojiCount int    `json:"emoji_count"`
		}

		var results []fileResult
		totalEmojis := 0
		filesScanned := 0

		for _, pattern := range patterns {
			matches, err := resolveFileList(pattern)
			if err != nil {
				continue
			}

			for _, filePath := range matches {
				data, err := os.ReadFile(filePath)
				if err != nil {
					continue
				}

				content := string(data)
				count := 0
				for _, emoji := range commonEmojis {
					count += strings.Count(content, emoji)
				}

				filesScanned++
				totalEmojis += count
				results = append(results, fileResult{
					Path:       filePath,
					EmojiCount: count,
				})
			}
		}

		if results == nil {
			results = []fileResult{}
		}

		outputOK(map[string]interface{}{
			"files_scanned": filesScanned,
			"total_emojis":  totalEmojis,
			"files":         results,
		})
		return nil
	},
}

func init() {
	rootCmd.AddCommand(emojiAuditCmd)
}

// countEmojisInString counts occurrences of common emojis in a string.
func countEmojisInString(s string) int {
	count := 0
	for _, emoji := range commonEmojis {
		count += strings.Count(s, emoji)
	}
	return count
}

// emojiAuditFile scans a single file and returns the emoji count.
func emojiAuditFile(filePath string) (int, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", filePath, err)
	}
	return countEmojisInString(string(data)), nil
}
