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

// stateChangingCommands lists commands that should reference runtime
// ceremony commands in their wrapper markdown.
var stateChangingCommands = []string{"build", "continue", "init", "seal", "plan"}

// ceremonyRefPattern matches runtime ceremony command references in wrappers.
// Wrappers delegate ceremony rendering to the Go runtime via subcommands
// like "aether ceremony spawn-plan", "aether ceremony wave-start", etc.
var ceremonyRefPattern = regexp.MustCompile(`aether ceremony`)

// claudeWrapperLocation says where the Claude wrapper for one command really
// is. The repo's own copy (.claude/commands/ant/<name>.md) exists only in the
// Aether source checkout, or in an older project not yet updated: in an
// ordinary project `aether update` deliberately removes it and installs the
// wrapper into the home Claude folder as ant-<name>.md instead. Looking only
// inside the project made `medic --deep` call every working project's
// wrappers "not found" (Phase 210 blocker 17).
type claudeWrapperLocation struct {
	path  string // the wrapper found, or where it should be when missing
	found bool
	// authored is true for the repo's own copy. Only that copy's content is
	// checked here: an installed copy is the shipped wrapper itself, whose
	// content is checked where it is written, in the Aether repo.
	authored bool
}

// installedClaudeWrapperPath is where install and update put the Claude
// wrapper for a command (claudeCommandDestRelPath under the home
// .claude/commands folder), or "" when the home folder is unknown.
func installedClaudeWrapperPath(command string) string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return ""
	}
	return filepath.Join(home, ".claude", "commands", claudeCommandDestRelPath(command+".md"))
}

func locateClaudeWrapper(fc *fileChecker, command string) claudeWrapperLocation {
	repoCopy := filepath.Join(fc.repoRoot, ".claude", "commands", "ant", command+".md")
	if fileExists(repoCopy) {
		return claudeWrapperLocation{path: repoCopy, found: true, authored: true}
	}
	installed := installedClaudeWrapperPath(command)
	if installed != "" && fileExists(installed) {
		return claudeWrapperLocation{path: installed, found: true}
	}
	if installed == "" || isAetherSourceCheckout(fc.repoRoot) {
		return claudeWrapperLocation{path: repoCopy, authored: true}
	}
	return claudeWrapperLocation{path: installed}
}

// missingWrapperRemedy is the plain next step for a wrapper that is missing
// from where it should be.
func missingWrapperRemedy(loc claudeWrapperLocation) string {
	if loc.authored {
		return "Its source file is missing from this checkout."
	}
	return "Run `aether update --force` to reinstall it."
}

// checkStageMarkers verifies that state-changing command wrappers reference
// the runtime ceremony commands (not literal stage markers), and that YAML
// source files exist for each state-changing command.
func checkStageMarkers(fc *fileChecker) []HealthIssue {
	var issues []HealthIssue

	for _, cmd := range stateChangingCommands {
		loc := locateClaudeWrapper(fc, cmd)
		if !loc.found {
			issues = append(issues, issueWarning("ceremony", loc.path,
				fmt.Sprintf("Wrapper for '%s' not found: the /ant-%s menu command is not where Claude Code looks for it. %s", cmd, cmd, missingWrapperRemedy(loc))))
			continue
		}
		if !loc.authored {
			continue
		}
		content, err := os.ReadFile(loc.path)
		if err != nil {
			continue
		}

		if !ceremonyRefPattern.Match(content) {
			issues = append(issues, issueWarning("ceremony", cmd,
				fmt.Sprintf("Wrapper for '%s' has no runtime ceremony references (state-changing command should reference 'aether ceremony')", cmd)))
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

	loc := locateClaudeWrapper(fc, "continue")
	if !loc.found {
		issues = append(issues, issueWarning("ceremony", loc.path,
			"continue.md not found (context-clear guidance missing). "+missingWrapperRemedy(loc)))
		return issues
	}
	if !loc.authored {
		return issues
	}
	content, err := os.ReadFile(loc.path)
	if err != nil {
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
// commandEmojiMap and casteEmojiMap, including the variation selector U+FE0F.
// Ranges: U+2600-U+27BF (misc symbols, dingbats), U+2B00-U+2BFF (arrows),
// U+1F300-U+1FAFF (full emoji range).
var emojiPattern = regexp.MustCompile(`[\x{2600}-\x{27BF}\x{2B00}-\x{2BFF}\x{1F300}-\x{1FAFF}]\x{FE0F}?`)

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
