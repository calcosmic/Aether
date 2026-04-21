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

// stateChangingCommands lists commands that should have stage marker
// ceremony in their wrapper markdown.
var stateChangingCommands = []string{"build", "continue", "init", "seal", "plan"}

// stageMarkerPattern matches stage markers in the form "── ... ──".
var stageMarkerPattern = regexp.MustCompile(`──.*──`)

// checkStageMarkers verifies that state-changing command wrappers contain
// stage marker references, and that YAML source files exist for each
// state-changing command.
func checkStageMarkers(fc *fileChecker) []HealthIssue {
	var issues []HealthIssue

	claudeCmdDir := filepath.Join(fc.repoRoot, ".claude", "commands", "ant")

	for _, cmd := range stateChangingCommands {
		wrapperPath := filepath.Join(claudeCmdDir, cmd+".md")
		content, err := os.ReadFile(wrapperPath)
		if err != nil {
			if os.IsNotExist(err) {
				issues = append(issues, issueWarning("ceremony", cmd,
					fmt.Sprintf("Wrapper for '%s' not found", cmd)))
			}
			continue
		}

		if !stageMarkerPattern.Match(content) {
			issues = append(issues, issueWarning("ceremony", cmd,
				fmt.Sprintf("Wrapper for '%s' has no stage markers (state-changing command should include ceremony)", cmd)))
		}

		// Verify YAML source exists
		yamlPath := filepath.Join(fc.repoRoot, ".aether", "commands", cmd+".yaml")
		if _, err := os.Stat(yamlPath); err != nil {
			issues = append(issues, issueWarning("ceremony", cmd,
				fmt.Sprintf("YAML source for '%s' not found at .aether/commands/%s.yaml", cmd, cmd)))
		}
	}

	return issues
}

// checkContextClearGuidance verifies that context-clear guidance in
// continue.md is runtime-owned (not hard-coded).
func checkContextClearGuidance(fc *fileChecker) []HealthIssue {
	var issues []HealthIssue

	continuePath := filepath.Join(fc.repoRoot, ".claude", "commands", "ant", "continue.md")
	content, err := os.ReadFile(continuePath)
	if err != nil {
		if os.IsNotExist(err) {
			issues = append(issues, issueWarning("ceremony", "continue.md",
				"continue.md not found (context-clear guidance missing)"))
		}
		return issues
	}

	text := string(content)

	// Verify context-clear guidance exists (references runtime emission)
	if !strings.Contains(text, "context-clear") && !strings.Contains(text, "context clear") {
		issues = append(issues, issueInfo("ceremony", "continue.md",
			"No context-clear guidance found in continue.md"))
	}

	// Check for hard-coded context-clear patterns that should be runtime-owned
	// The runtime owns context-clear via renderContextClearGuidance().
	// Wrappers should NOT contain their own context-clear instructions.
	hardcodedPatterns := []string{
		"It's safe to clear your context now",
		"You can safely clear",
		"safe to clear",
	}
	for _, pattern := range hardcodedPatterns {
		if strings.Contains(text, pattern) {
			issues = append(issues, issueWarning("ceremony", "continue.md",
				fmt.Sprintf("Context-clear guidance in continue.md contains hardcoded value '%s' (should be runtime-owned)", pattern)))
		}
	}

	return issues
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
