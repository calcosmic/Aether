package codex

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

const (
	defaultPromptBudgetChars = 24000
	promptBudgetEnvVar       = "AETHER_CODEX_PROMPT_BUDGET"
	promptTrimMarker         = "\n\n[truncated]"
)

// TaskBriefData holds structured data for rendering a worker task brief.
type TaskBriefData struct {
	TaskID          string   // Task identifier (e.g., "2.1")
	Goal            string   // What the worker needs to accomplish
	Constraints     []string // Hard constraints the worker must follow
	Hints           []string // Helpful hints or pointers to relevant code
	SuccessCriteria []string // How success will be measured
}

// LoadAgentInstructions reads a Codex agent TOML file and returns the
// developer_instructions field. Returns an error if the file cannot be read,
// parsed, or if the developer_instructions field is missing or empty.
func LoadAgentInstructions(tomlPath string) (string, error) {
	data, err := os.ReadFile(tomlPath)
	if err != nil {
		return "", fmt.Errorf("load agent instructions: read %s: %w", tomlPath, err)
	}

	var agent agentTOML
	if _, err := toml.Decode(string(data), &agent); err != nil {
		return "", fmt.Errorf("load agent instructions: parse %s: %w", tomlPath, err)
	}

	if agent.DeveloperInstructions == "" {
		return "", fmt.Errorf("load agent instructions: %s: missing or empty developer_instructions field", tomlPath)
	}

	return agent.DeveloperInstructions, nil
}

// AssemblePrompt combines prompt sections in the correct order:
//  1. TOML developer_instructions (agent role definition)
//  2. Compact colony-prime context
//  3. Worker handoff section (prior relay data, if non-empty)
//  4. Skill section (skill guidance, if non-empty)
//  5. Pheromone section (pheromone signals, if non-empty)
//  6. Task brief (worker's specific assignment)
//
// Empty sections are omitted. The fully assembled prompt is then trimmed to the
// configured global prompt budget. Returns the concatenated prompt string.
func AssemblePrompt(agentTOMLPath, contextCapsule, handoffSection, skillSection, pheromoneSection, taskBrief string) (string, error) {
	instructions, err := LoadAgentInstructions(agentTOMLPath)
	if err != nil {
		return "", err
	}

	parts := []promptPart{
		{name: "instructions", content: strings.TrimSpace(instructions), required: true},
		{name: "context", content: strings.TrimSpace(contextCapsule), required: false},
		{name: "handoff", content: strings.TrimSpace(handoffSection), required: false},
		{name: "skill", content: strings.TrimSpace(skillSection), required: false},
		{name: "pheromone", content: strings.TrimSpace(pheromoneSection), required: false},
		{name: "brief", content: strings.TrimSpace(taskBrief), required: true},
	}

	return assemblePromptParts(parts, promptBudgetChars()), nil
}

// AssembleHostedPrompt builds the worker prompt for platforms that load their
// agent instructions from the host runtime instead of a Codex TOML file.
func AssembleHostedPrompt(contextCapsule, handoffSection, skillSection, pheromoneSection, taskBrief string) string {
	parts := []promptPart{
		{name: "context", content: strings.TrimSpace(contextCapsule), required: false},
		{name: "handoff", content: strings.TrimSpace(handoffSection), required: false},
		{name: "skill", content: strings.TrimSpace(skillSection), required: false},
		{name: "pheromone", content: strings.TrimSpace(pheromoneSection), required: false},
		{name: "brief", content: strings.TrimSpace(taskBrief), required: true},
	}
	return assemblePromptParts(parts, promptBudgetChars())
}

// RenderTaskBrief formats a TaskBriefData into a markdown task brief string.
// Sections with empty slices are omitted.
func RenderTaskBrief(task TaskBriefData) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# Task %s\n\n", task.TaskID)
	fmt.Fprintf(&b, "Goal: %s\n\n", task.Goal)

	if len(task.Constraints) > 0 {
		b.WriteString("## Constraints\n\n")
		for _, c := range task.Constraints {
			fmt.Fprintf(&b, "- %s\n", c)
		}
		b.WriteString("\n")
	}

	if len(task.Hints) > 0 {
		b.WriteString("## Hints\n\n")
		for _, h := range task.Hints {
			fmt.Fprintf(&b, "- %s\n", h)
		}
		b.WriteString("\n")
	}

	if len(task.SuccessCriteria) > 0 {
		b.WriteString("## Success Criteria\n\n")
		for _, sc := range task.SuccessCriteria {
			fmt.Fprintf(&b, "- %s\n", sc)
		}
	}

	return b.String()
}

type promptPart struct {
	name     string
	content  string
	required bool
}

func promptBudgetChars() int {
	raw := strings.TrimSpace(os.Getenv(promptBudgetEnvVar))
	if raw == "" {
		return defaultPromptBudgetChars
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return defaultPromptBudgetChars
	}
	return value
}

func assemblePromptParts(parts []promptPart, budget int) string {
	if budget <= 0 {
		budget = defaultPromptBudgetChars
	}

	joined := joinPromptParts(parts)
	if len(joined) <= budget {
		return joined
	}

	trimOrder := []struct {
		name     string
		minChars int
	}{
		{name: "skill", minChars: 0},
		{name: "pheromone", minChars: 0},
		{name: "handoff", minChars: 0},
		{name: "context", minChars: 32},
	}
	for _, item := range trimOrder {
		if len(joined) <= budget {
			break
		}
		idx := promptPartIndex(parts, item.name)
		if idx < 0 || strings.TrimSpace(parts[idx].content) == "" {
			continue
		}
		parts[idx].content = fitPartToBudget(parts, idx, budget, item.minChars)
		joined = joinPromptParts(parts)
	}

	requiredTrimOrder := []string{"brief", "instructions"}
	for _, name := range requiredTrimOrder {
		if len(joined) <= budget {
			break
		}
		idx := promptPartIndex(parts, name)
		if idx < 0 || strings.TrimSpace(parts[idx].content) == "" {
			continue
		}
		parts[idx].content = fitPartToBudget(parts, idx, budget, 32)
		joined = joinPromptParts(parts)
	}

	if len(joined) > budget {
		return truncatePromptContent(joined, budget)
	}
	return joined
}

func fitPartToBudget(parts []promptPart, idx, budget, minChars int) string {
	content := strings.TrimSpace(parts[idx].content)
	if content == "" {
		return ""
	}

	candidate := parts[idx]
	if !candidate.required && minChars == 0 {
		parts[idx].content = ""
		if len(joinPromptParts(parts)) <= budget {
			return ""
		}
	}
	parts[idx] = candidate

	runes := []rune(content)
	if minChars < 0 {
		minChars = 0
	}
	if minChars > len(runes) {
		minChars = len(runes)
	}

	best := ""
	low := minChars
	high := len(runes)
	for low <= high {
		mid := (low + high) / 2
		parts[idx].content = truncatePromptContent(content, mid)
		if len(joinPromptParts(parts)) <= budget {
			best = parts[idx].content
			low = mid + 1
			continue
		}
		high = mid - 1
	}

	if best != "" {
		return best
	}
	return truncatePromptContent(content, minChars)
}

func joinPromptParts(parts []promptPart) string {
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part.content) == "" {
			continue
		}
		segments = append(segments, strings.TrimSpace(part.content))
	}
	return strings.Join(segments, "\n\n")
}

func joinPromptPartsWithout(parts []promptPart, skip int) string {
	filtered := make([]promptPart, 0, len(parts)-1)
	for i, part := range parts {
		if i == skip {
			continue
		}
		filtered = append(filtered, part)
	}
	return joinPromptParts(filtered)
}

func promptPartIndex(parts []promptPart, name string) int {
	for i, part := range parts {
		if part.name == name {
			return i
		}
	}
	return -1
}

func truncatePromptContent(text string, maxChars int) string {
	text = strings.TrimSpace(text)
	if maxChars <= 0 || text == "" {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= maxChars {
		return text
	}

	marker := []rune(promptTrimMarker)
	if maxChars <= len(marker)+8 {
		return strings.TrimSpace(string(runes[:maxChars]))
	}

	keep := maxChars - len(marker)
	if keep < 8 {
		keep = 8
	}
	return strings.TrimSpace(string(runes[:keep])) + promptTrimMarker
}
