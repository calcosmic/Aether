package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/calcosmic/Aether/pkg/codegraph"
	"github.com/calcosmic/Aether/pkg/colony"
)

const codegraphWorkerContextBudgetChars = 2200

func renderCodegraphContextForText(root string, textParts []string, maxChars int) string {
	graph, err := loadCodegraphForContext(root)
	if err != nil || graph == nil {
		return ""
	}
	targets := inferCodegraphTargets(graph, textParts)
	if len(targets) == 0 {
		return ""
	}
	return renderCodegraphContext(graph, targets, maxChars)
}

func loadCodegraphForContext(root string) (*codegraph.CodeGraph, error) {
	var candidates []string
	if store != nil && strings.TrimSpace(store.BasePath()) != "" {
		candidates = append(candidates, filepath.Join(store.BasePath(), "codebase-graph.json"))
	}
	if strings.TrimSpace(root) != "" {
		candidates = append(candidates, filepath.Join(root, ".aether", "data", "codebase-graph.json"))
		candidates = append(candidates, filepath.Join(root, "codebase-graph.json"))
	}
	for _, candidate := range uniqueStringSlice(candidates) {
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		if _, err := os.Stat(candidate); err != nil {
			continue
		}
		return codegraph.Load(candidate)
	}
	return nil, os.ErrNotExist
}

func inferCodegraphTargets(graph *codegraph.CodeGraph, textParts []string) []string {
	haystack := strings.ToLower(filepath.ToSlash(strings.Join(textParts, "\n")))
	if strings.TrimSpace(haystack) == "" {
		return nil
	}
	var targets []string
	for _, file := range graph.Files {
		path := filepath.ToSlash(filepath.Clean(file.Path))
		lowerPath := strings.ToLower(path)
		base := strings.ToLower(filepath.Base(path))
		if lowerPath == "." || lowerPath == "" {
			continue
		}
		if strings.Contains(haystack, lowerPath) || codegraphBaseNameMentioned(haystack, base) {
			targets = append(targets, path)
		}
	}
	return uniqueSortedStrings(targets)
}

func codegraphBaseNameMentioned(haystack, base string) bool {
	if len(base) < 5 || !strings.Contains(base, ".") {
		return false
	}
	return strings.Contains(haystack, base)
}

func renderCodegraphContext(graph *codegraph.CodeGraph, targets []string, maxChars int) string {
	if maxChars <= 0 {
		maxChars = codegraphWorkerContextBudgetChars
	}
	related := graph.FilesForTask(targets)
	if len(related) > 12 {
		related = related[:12]
	}

	var b strings.Builder
	b.WriteString("## Codebase Graph Context\n\n")
	b.WriteString("- Target files inferred from this assignment: ")
	b.WriteString(strings.Join(targets, ", "))
	b.WriteString("\n")
	if len(related) > 0 {
		b.WriteString("- Related files within 2 hops: ")
		b.WriteString(strings.Join(related, ", "))
		b.WriteString("\n")
	}
	if deps := strings.TrimSpace(graph.FormatRelatedFiles(targets, maxChars/2)); deps != "" {
		b.WriteString("\n")
		b.WriteString(deps)
		b.WriteString("\n")
	}
	rendered := strings.TrimSpace(b.String())
	if len(rendered) <= maxChars {
		return rendered
	}
	return truncateTextWithMarker(rendered, maxChars, "\n\n[codegraph context truncated]")
}

func codegraphTextPartsForBuildBrief(phase colony.Phase, dispatch codexBuildDispatch) []string {
	parts := []string{dispatch.Task, dispatch.TaskID, dispatch.Stage, dispatch.Caste}
	if strings.TrimSpace(phase.Name) != "" {
		parts = append(parts, phase.Name)
	}
	if strings.TrimSpace(phase.Description) != "" {
		parts = append(parts, phase.Description)
	}
	if task := findDispatchTask(phase, dispatch); task != nil {
		parts = append(parts, task.Goal)
		parts = append(parts, task.Constraints...)
		parts = append(parts, task.Hints...)
		parts = append(parts, task.SuccessCriteria...)
	}
	return parts
}

func uniqueStringSlice(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func truncateTextWithMarker(text string, maxChars int, marker string) string {
	text = strings.TrimSpace(text)
	if maxChars <= 0 {
		return ""
	}
	if len(text) <= maxChars {
		return text
	}
	if maxChars <= len(marker) {
		return strings.TrimSpace(marker[:maxChars])
	}
	keep := maxChars - len(marker)
	return strings.TrimSpace(text[:keep]) + marker
}
