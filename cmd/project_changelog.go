package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// The project changelog: when a project is marked finished, the program adds
// one entry to CHANGELOG.md in the project folder -- the date, the goal, and a
// line per phase -- creating the file when there is none (owner's ruling,
// 2026-09-21). It is a courtesy written AFTER the finish has been committed:
// nothing here can fail, undo, or delay the finish itself.

const projectChangelogFileName = "CHANGELOG.md"

const projectChangelogNewFileHeader = "# Changelog\n\n" +
	"What was finished in this project, newest first. Dated entries are added\n" +
	"automatically by Aether each time a project here is marked finished.\n"

type projectChangelogResult struct {
	Path           string `json:"path,omitempty"`
	Written        bool   `json:"written"`
	Created        bool   `json:"created,omitempty"`
	AlreadyPresent bool   `json:"already_present,omitempty"`
	Problem        string `json:"problem,omitempty"`
}

// writeProjectChangelogEntry never returns an error: a caller must not be able
// to turn a changelog problem into a failed finish. Problem says what went
// wrong, in words, when nothing was written.
func writeProjectChangelogEntry(root string, state colony.ColonyState, outcome colony.SealOutcome, transactionID string, now time.Time) projectChangelogResult {
	goal := projectChangelogOneLine(derefGoal(state.Goal))
	if state.AcceptedCharter != nil && strings.TrimSpace(state.AcceptedCharter.Goal) != "" {
		goal = projectChangelogOneLine(state.AcceptedCharter.Goal)
	}
	root = strings.TrimSpace(root)
	if goal == "" || root == "" {
		return projectChangelogResult{Problem: "no project goal to record"}
	}
	path := filepath.Join(root, projectChangelogFileName)
	result := projectChangelogResult{Path: path}

	existing := ""
	info, err := os.Lstat(path)
	switch {
	case err == nil && !info.Mode().IsRegular():
		// A link could point outside the project; a folder is not ours to replace.
		result.Problem = projectChangelogFileName + " is not an ordinary file, so it was left alone"
		return result
	case err == nil:
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			result.Problem = "could not read " + projectChangelogFileName + ": " + readErr.Error()
			return result
		}
		existing = string(data)
	case os.IsNotExist(err):
		result.Created = true
		existing = projectChangelogNewFileHeader
	default:
		result.Problem = "could not inspect " + projectChangelogFileName + ": " + err.Error()
		return result
	}

	marker := projectChangelogMarker(transactionID)
	if marker != "" && strings.Contains(existing, marker) {
		result.Created = false
		result.AlreadyPresent = true
		return result
	}

	entry := renderProjectChangelogEntry(goal, state, outcome, marker, now)
	updated := insertProjectChangelogEntry(existing, entry)

	mode := os.FileMode(0o644)
	if info != nil {
		mode = info.Mode().Perm()
	}
	temp, err := os.CreateTemp(root, ".changelog-*.tmp")
	if err != nil {
		result.Created = false
		result.Problem = "could not write " + projectChangelogFileName + ": " + err.Error()
		return result
	}
	tempPath := temp.Name()
	_, writeErr := temp.WriteString(updated)
	closeErr := temp.Close()
	if writeErr == nil && closeErr == nil {
		writeErr = os.Chmod(tempPath, mode)
	}
	if writeErr == nil && closeErr == nil {
		writeErr = os.Rename(tempPath, path)
	}
	if writeErr != nil || closeErr != nil {
		_ = os.Remove(tempPath)
		result.Created = false
		if writeErr == nil {
			writeErr = closeErr
		}
		result.Problem = "could not write " + projectChangelogFileName + ": " + writeErr.Error()
		return result
	}
	result.Written = true
	return result
}

func projectChangelogMarker(transactionID string) string {
	transactionID = strings.TrimSpace(transactionID)
	if transactionID == "" || strings.ContainsAny(transactionID, "<>\n\r") || strings.Contains(transactionID, "--") {
		return ""
	}
	return "<!-- aether:project-finished " + transactionID + " -->"
}

func projectChangelogOneLine(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	const limit = 160
	if runes := []rune(text); len(runes) > limit {
		text = strings.TrimSpace(string(runes[:limit-1])) + "…"
	}
	return text
}

func renderProjectChangelogEntry(goal string, state colony.ColonyState, outcome colony.SealOutcome, marker string, now time.Time) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## %s — %s\n", now.UTC().Format("2006-01-02"), goal)
	if marker != "" {
		b.WriteString(marker + "\n")
	}
	b.WriteString("\n")
	forced := outcome.Disposition == colony.SealDispositionForcedIncomplete
	if forced {
		b.WriteString("_This project was closed before every phase was finished._\n\n")
	}
	unfinished := map[int]bool{}
	for _, id := range outcome.IncompletePhases {
		unfinished[id] = true
	}
	for index, phase := range state.Plan.Phases {
		name := projectChangelogOneLine(phase.Name)
		if name == "" {
			continue
		}
		number := phase.ID
		if number == 0 {
			number = index + 1
		}
		line := fmt.Sprintf("- Phase %d: %s", number, name)
		if unfinished[phase.ID] || (forced && phase.Status != colony.PhaseCompleted) {
			line += " (not finished)"
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

// insertProjectChangelogEntry puts the newest entry first, but never above a
// hand-kept "Unreleased" section or the file's own introduction.
func insertProjectChangelogEntry(existing, entry string) string {
	lines := strings.Split(existing, "\n")
	insertAt := -1
	for index, line := range lines {
		if !strings.HasPrefix(line, "## ") {
			continue
		}
		if strings.Contains(strings.ToLower(line), "unreleased") {
			continue
		}
		insertAt = index
		break
	}
	if insertAt < 0 {
		return strings.TrimRight(existing, "\n") + "\n\n" + entry
	}
	head := strings.TrimRight(strings.Join(lines[:insertAt], "\n"), "\n")
	tail := strings.Join(lines[insertAt:], "\n")
	return head + "\n\n" + entry + "\n" + tail
}
