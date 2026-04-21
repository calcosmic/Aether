package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// scanCeremonyIntegrity validates emoji consistency, stage markers, and
// context-clear guidance across wrapper files and the Go runtime renderer.
func scanCeremonyIntegrity(fc *fileChecker) []HealthIssue {
	var issues []HealthIssue

	issues = append(issues, checkEmojiConsistency(fc)...)
	issues = append(issues, checkStageMarkers(fc)...)
	issues = append(issues, checkContextClearGuidance(fc)...)

	return issues
}

// checkStageMarkers is implemented in the stage marker check task.
func checkStageMarkers(fc *fileChecker) []HealthIssue {
	return nil
}

// checkContextClearGuidance is implemented in the context-clear check task.
func checkContextClearGuidance(fc *fileChecker) []HealthIssue {
	return nil
}

// emojiPattern matches Unicode emoji characters commonly used in command
// descriptions and wrapper markdown. Covers emoji in the ranges used by
// commandEmojiMap and casteEmojiMap.
var emojiPattern = regexp.MustCompile(`[\x{1F300}-\x{1FAFF}]`)

// extractEmojisFromMarkdown returns unique emoji characters found in the
// given markdown content.
func extractEmojisFromMarkdown(content string) []string {
	matches := emojiPattern.FindAllString(content, -1)
	seen := make(map[string]bool)
	var unique []string
	for _, m := range matches {
		if !seen[m] {
			seen[m] = true
			unique = append(unique, m)
		}
	}
	return unique
}

// getCommandEmoji returns the expected emoji for a command from commandEmojiMap.
func getCommandEmoji(command string) string {
	if emoji, ok := commandEmojiMap[command]; ok {
		return emoji
	}
	return ""
}

// checkEmojiConsistency validates that emojis used in wrapper markdown files
// match the ground truth in commandEmojiMap. Checks both Claude and OpenCode
// wrappers.
func checkEmojiConsistency(fc *fileChecker) []HealthIssue {
	var issues []HealthIssue

	wrapperDirs := []struct {
		label string
		dir   string
	}{
		{"Claude", filepath.Join(fc.repoRoot, ".claude", "commands", "ant")},
		{"OpenCode", filepath.Join(fc.repoRoot, ".opencode", "commands", "ant")},
	}

	for _, wd := range wrapperDirs {
		entries, err := os.ReadDir(wd.dir)
		if err != nil {
			// Directory missing — skip; wrapper parity already catches this.
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}

			command := strings.TrimSuffix(entry.Name(), ".md")
			expected := getCommandEmoji(command)

			filePath := filepath.Join(wd.dir, entry.Name())
			content, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}

			emojis := extractEmojisFromMarkdown(string(content))

			// Only check commands that are in commandEmojiMap
			if expected == "" {
				continue
			}

			if len(emojis) == 0 {
				issues = append(issues, issueInfo("ceremony", fmt.Sprintf("%s/%s", wd.label, entry.Name()),
					fmt.Sprintf("Wrapper for '%s' has no emoji (runtime uses '%s')", command, expected)))
				continue
			}

			// Check if the expected emoji is among the found emojis
			found := false
			var unexpected []string
			for _, e := range emojis {
				if e == expected {
					found = true
				} else {
					unexpected = append(unexpected, e)
				}
			}

			if !found && len(unexpected) > 0 {
				issues = append(issues, issueWarning("ceremony", fmt.Sprintf("%s/%s", wd.label, entry.Name()),
					fmt.Sprintf("Wrapper for '%s' uses emoji '%s' but runtime expects '%s'",
						command, strings.Join(unexpected, ""), expected)))
			}
		}
	}

	return issues
}
