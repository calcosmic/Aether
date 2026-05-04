package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const buildWorkerBriefPlaybookBudgetChars = 7000
const buildWorkerBriefPlaybookPerFileChars = 2800

func renderBuildPlaybookContext(root string, dispatch codexBuildDispatch, playbooks []string) string {
	selected := buildPlaybooksForDispatch(dispatch, playbooks)
	if len(selected) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("## Relevant Playbooks\n\n")
	remaining := buildWorkerBriefPlaybookBudgetChars
	for _, playbook := range selected {
		if remaining <= 0 {
			break
		}
		snippet := readBuildPlaybookSnippet(root, playbook, minInt(buildWorkerBriefPlaybookPerFileChars, remaining))
		if strings.TrimSpace(snippet) == "" {
			fmt.Fprintf(&b, "- `%s` (not found in repo or hub; use the manifest path as a reference)\n", playbook)
			continue
		}
		header := fmt.Sprintf("### %s\n\n", playbook)
		if len(header)+len(snippet)+2 > remaining {
			snippet = truncateTextWithMarker(snippet, remaining-len(header)-2, "\n\n[playbook truncated]")
		}
		b.WriteString(header)
		b.WriteString(snippet)
		b.WriteString("\n\n")
		remaining = buildWorkerBriefPlaybookBudgetChars - len(b.String())
	}
	return strings.TrimSpace(b.String())
}

func readBuildPlaybookSnippet(root, playbook string, maxChars int) string {
	if maxChars <= 0 {
		return ""
	}
	for _, candidate := range buildPlaybookCandidates(root, playbook) {
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		return truncateTextWithMarker(string(data), maxChars, "\n\n[playbook truncated]")
	}
	return ""
}

func buildPlaybookCandidates(root, playbook string) []string {
	playbook = filepath.ToSlash(strings.TrimSpace(playbook))
	if playbook == "" {
		return nil
	}
	var candidates []string
	if filepath.IsAbs(playbook) {
		candidates = append(candidates, filepath.FromSlash(playbook))
	}
	if strings.TrimSpace(root) != "" {
		candidates = append(candidates, filepath.Join(root, filepath.FromSlash(playbook)))
	}
	hubDir := resolveHubPath()
	if strings.TrimSpace(hubDir) != "" {
		trimmed := strings.TrimPrefix(playbook, ".aether/")
		candidates = append(candidates, filepath.Join(hubDir, "system", filepath.FromSlash(trimmed)))
		candidates = append(candidates, filepath.Join(hubDir, filepath.FromSlash(trimmed)))
	}
	candidates = append(candidates, filepath.FromSlash(playbook))
	return uniqueStringSlice(candidates)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
