package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

// changeThreshold is the minimum number of changed files (per git diff --stat)
// required to trigger a full re-analysis. Changes below this threshold are
// considered insignificant and skip analysis.
const changeThreshold = 5

// todoFixmeThreshold is the TODO/FIXME count above which a FEEDBACK suggestion
// is generated about tech debt density.
const todoFixmeThreshold = 10

// largeFileLineThreshold is the line count above which a Go file is flagged
// as needing to be split.
const largeFileLineThreshold = 500

// highDependencyThreshold is the number of direct dependencies above which a
// FEEDBACK suggestion is generated about dependency count.
const highDependencyThreshold = 20

// suggestAnalyzeInvocationCount tracks how many times runSuggestAnalyze has
// executed in the current process. It exists purely as a test seam: tests
// assert non-blocking callers (like build finalize) invoke it exactly once
// per operation, not once per worker dispatch inside a loop -- the exact
// regression class this project has shipped before. Production code never
// reads this value.
var suggestAnalyzeInvocationCount int

var suggestAnalyzeCmd = &cobra.Command{
	Use:   "suggest-analyze",
	Short: "Analyze codebase for patterns worth capturing as pheromone suggestions",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		target, _ := cmd.Flags().GetString("target")

		result, err := runSuggestAnalyze(target, dryRun)
		if err != nil {
			outputErrorMessage(err.Error())
			return nil
		}
		outputOK(result)
		return nil
	},
}

// runSuggestAnalyze is the entire suggest-analyze pattern-detection pipeline,
// factored out of the cobra RunE so it can be called from any live caller
// (the CLI command itself, and cmd/codex_build_finalize.go's end-of-build
// hook). Content sanitization (prompt-injection rejection) always runs
// inside this function -- callers must not re-implement it on top of this.
//
// Returns a non-nil error only for a hard precondition failure (no store
// initialized). A colony-state load failure is treated as non-blocking per
// RESEARCH Pitfall 3: it returns an empty, successful result rather than an
// error, so a suggestion-engine problem never fails a build or the CLI.
func runSuggestAnalyze(target string, dryRun bool) (map[string]interface{}, error) {
	suggestAnalyzeInvocationCount++
	if store == nil {
		return nil, fmt.Errorf("no store initialized")
	}

	// Load active colony state. If this fails, return ok:true with empty
	// suggestions (non-blocking per RESEARCH Pitfall 3).
	cs, err := loadActiveColonyState()
	if err != nil {
		return map[string]interface{}{
			"suggestions":   []interface{}{},
			"total":         0,
			"new_count":     0,
			"skipped_dedup": 0,
			"dry_run":       dryRun,
		}, nil
	}

	// --- Change Detection (D-01) ---
	currentHead, err := execGitHead(target)
	if err != nil {
		// If we can't get HEAD, proceed with analysis anyway.
		currentHead = ""
	}

	if cs.LastAnalyzeCommit != nil && currentHead != "" && *cs.LastAnalyzeCommit != "" {
		changedCount, err := countChangedFiles(target, *cs.LastAnalyzeCommit, currentHead)
		if err == nil && changedCount < changeThreshold {
			// Below threshold: skip analysis, return existing pending suggestions.
			existing := pendingSuggestionsToMap(cs.PendingSuggestions)
			return map[string]interface{}{
				"suggestions":   existing,
				"total":         len(existing),
				"new_count":     0,
				"skipped_dedup": 0,
				"dry_run":       dryRun,
			}, nil
		}
	}

	// --- Pattern Detection ---
	governance := detectGovernance(target)
	dirClass := classifyDirectory(target)
	techStack := parseDependencyFiles(target)

	// Get the 25 base pheromone suggestions.
	baseSuggestions := generatePheromoneSuggestions(target, governance, dirClass, techStack)

	// Add build-specific extra patterns.
	extraSuggestions := buildSpecificPatterns(target, techStack)
	allSuggestions := append(baseSuggestions, extraSuggestions...)

	// --- Deduplication (D-07, D-08) ---
	activeHashSet, err := loadActivePheromoneHashes()
	if err != nil {
		activeHashSet = make(map[string]struct{})
	}

	var filtered []pheromoneSuggestion
	skippedCount := 0
	for _, sug := range allSuggestions {
		contentHash := "sha256:" + sha256Sum(sug.Content)
		key := sug.Type + ":" + contentHash
		if _, exists := activeHashSet[key]; exists {
			skippedCount++
			continue
		}
		filtered = append(filtered, sug)
	}

	// --- Sanitize (T-74-01) ---
	var sanitized []pheromoneSuggestion
	for _, sug := range filtered {
		_, err := colony.SanitizeSignalContent(sug.Content)
		if err != nil {
			continue // skip unsanitizable content
		}
		sanitized = append(sanitized, sug)
	}

	// Build output suggestions as maps.
	newCount := len(sanitized)
	var resultSuggestions []map[string]interface{}
	for _, sug := range sanitized {
		contentHash := "sha256:" + sha256Sum(sug.Content)
		resultSuggestions = append(resultSuggestions, map[string]interface{}{
			"type":         sug.Type,
			"content":      sug.Content,
			"reason":       sug.Reason,
			"content_hash": contentHash,
		})
	}

	// --- Persist (unless dry-run) ---
	if !dryRun {
		now := time.Now().UTC().Format(time.RFC3339)
		var pending []colony.PendingSuggestion
		for _, sug := range sanitized {
			contentHash := "sha256:" + sha256Sum(sug.Content)
			pending = append(pending, colony.PendingSuggestion{
				ID:          generateSignalID(),
				Type:        sug.Type,
				Content:     sug.Content,
				Reason:      sug.Reason,
				ContentHash: contentHash,
				CreatedAt:   now,
				Dismissed:   false,
			})
		}

		// Read-modify-write COLONY_STATE.json as a single guarded operation
		// (WR-01). The analysis above shells out to git and walks the whole
		// repo tree -- a window of seconds during which another writer could
		// change state. Merging against the `cs` snapshot loaded at function
		// entry and overwriting the file wholesale would silently clobber
		// that change; UpdateJSONAtomically re-reads the file inside the
		// mutation callback instead, so the merge always starts from the
		// latest committed state.
		var updatedState colony.ColonyState
		_ = store.UpdateJSONAtomically("COLONY_STATE.json", &updatedState, func() error {
			// Merge with existing pending suggestions: keep any that aren't in
			// the new set (by content hash comparison).
			merged := append([]colony.PendingSuggestion{}, pending...)
			if updatedState.PendingSuggestions != nil {
				existingHashes := make(map[string]struct{}, len(pending))
				for _, p := range pending {
					existingHashes[p.ContentHash] = struct{}{}
				}
				for _, old := range *updatedState.PendingSuggestions {
					if _, exists := existingHashes[old.ContentHash]; !exists {
						merged = append(merged, old)
					}
				}
			}
			updatedState.PendingSuggestions = &merged
			updatedState.LastAnalyzeCommit = &currentHead
			return nil
		})
	}

	return map[string]interface{}{
		"suggestions":   resultSuggestions,
		"total":         len(resultSuggestions),
		"new_count":     newCount,
		"skipped_dedup": skippedCount,
		"dry_run":       dryRun,
	}, nil
}

func init() {
	suggestAnalyzeCmd.Flags().Bool("dry-run", false, "Preview suggestions without persisting")
	suggestAnalyzeCmd.Flags().String("target", ".", "Target directory to analyze")
	rootCmd.AddCommand(suggestAnalyzeCmd)
}

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

// execGitHead returns the current HEAD commit hash for the given target directory.
func execGitHead(target string) (string, error) {
	cmd := exec.Command("git", "-C", target, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// countChangedFiles counts the number of files changed between two commits
// using git diff --stat.
func countChangedFiles(target, oldCommit, newCommit string) (int, error) {
	cmd := exec.Command("git", "-C", target, "diff", "--stat", oldCommit, newCommit)
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	// Each line in --stat output represents one changed file.
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	count := 0
	for _, line := range lines {
		// Skip summary line like "3 files changed, ..."
		if strings.Contains(line, "files changed") || line == "" {
			continue
		}
		count++
	}
	return count, nil
}

// loadActivePheromoneHashes builds a set of "type:content_hash" keys from all
// ACTIVE signals in pheromones.json. Used for deduplication.
func loadActivePheromoneHashes() (map[string]struct{}, error) {
	var pf colony.PheromoneFile
	if err := store.LoadJSON("pheromones.json", &pf); err != nil {
		return nil, err
	}

	set := make(map[string]struct{})
	for _, sig := range pf.Signals {
		if !sig.Active {
			continue
		}
		if sig.ContentHash == nil {
			continue
		}
		key := sig.Type + ":" + *sig.ContentHash
		set[key] = struct{}{}
	}
	return set, nil
}

// buildSpecificPatterns runs build-time specific analysis patterns:
// TODO/FIXME density, large files, test gaps, and dependency count.
func buildSpecificPatterns(target string, techStack []techStackDetail) []pheromoneSuggestion {
	var suggestions []pheromoneSuggestion

	// TODO/FIXME density check
	todoCount := countPatternInDir(target, "TODO")
	fixmeCount := countPatternInDir(target, "FIXME")
	if todoCount+fixmeCount > todoFixmeThreshold {
		suggestions = append(suggestions, pheromoneSuggestion{
			Type:    "FEEDBACK",
			Content: fmt.Sprintf("high TODO/FIXME density (%d found) -- consider addressing tech debt", todoCount+fixmeCount),
			Reason:  "build-specific: TODO/FIXME density check",
		})
	}

	// Large files check (Go files over 500 lines)
	largeFiles := findLargeFiles(target)
	for _, lf := range largeFiles {
		suggestions = append(suggestions, pheromoneSuggestion{
			Type:    "FEEDBACK",
			Content: fmt.Sprintf("large file detected (%s) -- consider splitting", lf),
			Reason:  "build-specific: large file detection",
		})
	}

	// Test gaps check
	testGaps := findTestGaps(target)
	for _, dir := range testGaps {
		suggestions = append(suggestions, pheromoneSuggestion{
			Type:    "FEEDBACK",
			Content: fmt.Sprintf("no tests found in %s", dir),
			Reason:  "build-specific: test gap detection",
		})
	}

	// High dependency count
	totalDeps := countDirectDependencies(techStack)
	if totalDeps > highDependencyThreshold {
		suggestions = append(suggestions, pheromoneSuggestion{
			Type:    "FEEDBACK",
			Content: fmt.Sprintf("high dependency count (%d) -- review for necessity", totalDeps),
			Reason:  "build-specific: dependency count check",
		})
	}

	return suggestions
}

// countPatternInDir recursively counts occurrences of a pattern string in all
// files under target. Only checks known source file extensions to avoid
// scanning binary files or node_modules.
func countPatternInDir(target, pattern string) int {
	count := 0
	sourceExts := map[string]bool{
		".go": true, ".js": true, ".ts": true, ".py": true, ".rb": true,
		".java": true, ".rs": true, ".md": true, ".yaml": true, ".yml": true,
		".toml": true, ".json": true, ".sh": true, ".bash": true,
	}

	_ = filepath.WalkDir(target, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			if d != nil && d.IsDir() && shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(d.Name())
		if !sourceExts[ext] {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		count += strings.Count(string(data), pattern)
		return nil
	})
	return count
}

// shouldSkipDir returns true for directories that should not be scanned.
func shouldSkipDir(name string) bool {
	skipDirs := map[string]bool{
		".git": true, "node_modules": true, "vendor": true, ".aether": true,
		"dist": true, "build": true, "__pycache__": true, ".venv": true,
		"target": true, "bin": true, ".claude": true, ".opencode": true,
		".codex": true, ".planning": true,
	}
	return skipDirs[name]
}

// findLargeFiles returns paths of Go files exceeding largeFileLineThreshold lines.
func findLargeFiles(target string) []string {
	var large []string
	_ = filepath.WalkDir(target, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			if d != nil && d.IsDir() && shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(d.Name()) != ".go" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		lineCount := strings.Count(string(data), "\n") + 1
		if lineCount > largeFileLineThreshold {
			rel, _ := filepath.Rel(target, path)
			large = append(large, rel)
		}
		return nil
	})
	return large
}

// findTestGaps returns directories containing > 3 Go source files but no _test.go files.
func findTestGaps(target string) []string {
	type dirInfo struct {
		goCount  int
		testFile bool
	}
	dirs := make(map[string]*dirInfo)

	_ = filepath.WalkDir(target, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			if d != nil && d.IsDir() && shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		dir := filepath.Dir(path)
		if dirs[dir] == nil {
			dirs[dir] = &dirInfo{}
		}
		if filepath.Ext(d.Name()) == ".go" {
			dirs[dir].goCount++
			if strings.HasSuffix(d.Name(), "_test.go") {
				dirs[dir].testFile = true
			}
		}
		return nil
	})

	var gaps []string
	for dir, info := range dirs {
		if info.goCount > 3 && !info.testFile {
			rel, _ := filepath.Rel(target, dir)
			gaps = append(gaps, rel)
		}
	}
	return gaps
}

// countDirectDependencies counts total direct dependencies across all tech stacks.
func countDirectDependencies(techStack []techStackDetail) int {
	count := 0
	for _, ts := range techStack {
		count += len(ts.Deps)
	}
	return count
}

// pendingSuggestionsToMaps converts PendingSuggestions to a slice of maps for JSON output.
func pendingSuggestionsToMap(suggestions *[]colony.PendingSuggestion) []map[string]interface{} {
	if suggestions == nil {
		return []map[string]interface{}{}
	}
	result := make([]map[string]interface{}, 0, len(*suggestions))
	for _, s := range *suggestions {
		result = append(result, map[string]interface{}{
			"id":           s.ID,
			"type":         s.Type,
			"content":      s.Content,
			"reason":       s.Reason,
			"content_hash": s.ContentHash,
			"created_at":   s.CreatedAt,
			"dismissed":    s.Dismissed,
		})
	}
	return result
}
