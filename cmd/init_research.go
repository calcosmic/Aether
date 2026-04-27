package cmd

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// --- Struct types for deep scan data ---

type gitHistoryInfo struct {
	Commits      int    `json:"commits"`
	Contributors int    `json:"contributors"`
	Branch       string `json:"branch,omitempty"`
}

type governanceInfo struct {
	Linters        []string `json:"linters"`
	Formatters     []string `json:"formatters"`
	TestFrameworks []string `json:"test_frameworks"`
	CIConfigs      []string `json:"ci_configs"`
	BuildTools     []string `json:"build_tools"`
}

type fileInfo struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

type complexityMetrics struct {
	TotalFiles   int        `json:"total_files"`
	TotalDirs    int        `json:"total_dirs"`
	LargestFiles []fileInfo `json:"largest_files,omitempty"`
}

type pheromoneSuggestion struct {
	Type    string `json:"type"`
	Content string `json:"content"`
	Reason  string `json:"reason"`
}

type charterData struct {
	Intent     string `json:"intent"`
	Vision     string `json:"vision"`
	Governance string `json:"governance"`
	Goals      string `json:"goals"`
}

// projectDetectors maps a marker file to a project type description.
var projectDetectors = []struct {
	file       string
	typ        string
	frameworks []string
}{
	{"package.json", "node", []string{"node"}},
	{"go.mod", "go", []string{"go"}},
	{"Cargo.toml", "rust", []string{"rust"}},
	{"pyproject.toml", "python", []string{"python"}},
	{"requirements.txt", "python", []string{"python"}},
	{"Gemfile", "ruby", []string{"ruby"}},
	{"pom.xml", "java", []string{"java", "maven"}},
	{"build.gradle", "java", []string{"java", "gradle"}},
	{"build.sbt", "scala", []string{"scala"}},
	{"mix.exs", "elixir", []string{"elixir"}},
	{"composer.json", "php", []string{"php"}},
	{"Makefile", "make", []string{"make"}},
}

// governanceDetectors maps config files to governance categories.
var governanceDetectors = []struct {
	file     string
	category string
	label    string
}{
	{".eslintrc", "linter", "ESLint"},
	{".eslintrc.js", "linter", "ESLint"},
	{".eslintrc.json", "linter", "ESLint"},
	{".eslintrc.yml", "linter", "ESLint"},
	{".prettierrc", "formatter", "Prettier"},
	{".prettierrc.json", "formatter", "Prettier"},
	{"biome.json", "formatter", "Biome"},
	{"golangci.yml", "linter", "golangci-lint"},
	{".golangci.yml", "linter", "golangci-lint"},
	{".golangci.yaml", "linter", "golangci-lint"},
	{"pytest.ini", "test", "pytest"},
	{"jest.config.js", "test", "Jest"},
	{"jest.config.ts", "test", "Jest"},
	{"vitest.config.ts", "test", "Vitest"},
	{"vitest.config.js", "test", "Vitest"},
	{".github/workflows/ci.yml", "ci", "GitHub Actions"},
	{".github/workflows/test.yml", "ci", "GitHub Actions"},
	{".github/workflows/build.yml", "ci", "GitHub Actions"},
	{".gitlab-ci.yml", "ci", "GitLab CI"},
	{"Jenkinsfile", "ci", "Jenkins"},
	{"Makefile", "build", "Make"},
	{"Taskfile.yml", "build", "Task"},
	{"justfile", "build", "Just"},
}

// extendedSkipDirs lists directories to skip during recursive walk.
var extendedSkipDirs = map[string]bool{
	".git":        true,
	"node_modules": true,
	".next":       true,
	"dist":        true,
	"build":       true,
	"vendor":      true,
	".venv":       true,
	"venv":        true,
	"coverage":    true,
	".aether":     true,
	".claude":     true,
	".opencode":   true,
	".codex":      true,
	"__pycache__": true,
}

// detectGovernance scans the target directory for governance tool config files.
func detectGovernance(target string) governanceInfo {
	info := governanceInfo{}
	seen := make(map[string]string) // label -> category for dedup

	// Check specific detector files
	for _, det := range governanceDetectors {
		p := filepath.Join(target, det.file)
		if _, err := os.Stat(p); err == nil {
			key := det.label
			if _, exists := seen[key]; !exists {
				seen[key] = det.category
				switch det.category {
				case "linter":
					info.Linters = append(info.Linters, det.label)
				case "formatter":
					info.Formatters = append(info.Formatters, det.label)
				case "test":
					info.TestFrameworks = append(info.TestFrameworks, det.label)
				case "ci":
					info.CIConfigs = append(info.CIConfigs, det.label)
				case "build":
					info.BuildTools = append(info.BuildTools, det.label)
				}
			}
		}
	}

	// Also glob for any .github/workflows/*.yml files
	ghDir := filepath.Join(target, ".github", "workflows")
	if entries, err := os.ReadDir(ghDir); err == nil {
		hasGHActions := false
		for _, e := range entries {
			if !e.IsDir() && (strings.HasSuffix(e.Name(), ".yml") || strings.HasSuffix(e.Name(), ".yaml")) {
				hasGHActions = true
				break
			}
		}
		if hasGHActions {
			if _, exists := seen["GitHub Actions"]; !exists {
				seen["GitHub Actions"] = "ci"
				info.CIConfigs = append(info.CIConfigs, "GitHub Actions")
			}
		}
	}

	return info
}

// analyzeGitHistory runs git commands to extract commit count, contributor count, and branch name.
func analyzeGitHistory(target string) gitHistoryInfo {
	var info gitHistoryInfo

	// Commit count
	out, err := exec.Command("git", "-C", target, "rev-list", "--count", "HEAD").CombinedOutput()
	if err == nil {
		trimmed := strings.TrimSpace(string(out))
		if n, err := strconv.Atoi(trimmed); err == nil {
			info.Commits = n
		}
	}

	// Contributor count
	out, err = exec.Command("git", "-C", target, "shortlog", "-sn", "HEAD").CombinedOutput()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		count := 0
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				count++
			}
		}
		info.Contributors = count
	}

	// Branch name
	out, err = exec.Command("git", "-C", target, "rev-parse", "--abbrev-ref", "HEAD").CombinedOutput()
	if err == nil {
		info.Branch = strings.TrimSpace(string(out))
	}

	return info
}

// detectPriorColonies checks for archived colony directories in .aether/chambers/.
func detectPriorColonies(target string) map[string]interface{} {
	chambersDir := filepath.Join(target, ".aether", "chambers")
	entries, err := os.ReadDir(chambersDir)
	if err != nil {
		return map[string]interface{}{
			"count": 0,
			"names": []string{},
		}
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}

	return map[string]interface{}{
		"count": len(names),
		"names": names,
	}
}

// hasFile checks whether a file exists at target/name.
func hasFile(target, name string) bool {
	_, err := os.Stat(filepath.Join(target, name))
	return err == nil
}

// fileContains checks whether a file at target/name contains the given substring.
func fileContains(target, name, substr string) bool {
	data, err := os.ReadFile(filepath.Join(target, name))
	if err != nil {
		return false
	}
	return strings.Contains(string(data), substr)
}

// generatePheromoneSuggestions applies 10 deterministic patterns to produce pheromone suggestions.
func generatePheromoneSuggestions(target string, governance governanceInfo) []pheromoneSuggestion {
	var suggestions []pheromoneSuggestion

	// 1. .env or .env.local exists -> REDIRECT about secrets
	if hasFile(target, ".env") || hasFile(target, ".env.local") {
		suggestions = append(suggestions, pheromoneSuggestion{
			Type:    "REDIRECT",
			Content: "never commit secrets or .env files to version control",
			Reason:  ".env file detected in project root",
		})
	}

	// 2. .env exists but .gitignore doesn't mention .env -> REDIRECT about .gitignore
	if hasFile(target, ".env") && !fileContains(target, ".gitignore", ".env") {
		suggestions = append(suggestions, pheromoneSuggestion{
			Type:    "REDIRECT",
			Content: "add .env to .gitignore to prevent secret leaks",
			Reason:  ".env exists without .gitignore entry",
		})
	}

	// 3. No CI config -> FOCUS about CI/CD
	hasCI := len(governance.CIConfigs) > 0
	if !hasCI {
		suggestions = append(suggestions, pheromoneSuggestion{
			Type:    "FOCUS",
			Content: "consider adding CI/CD pipeline for automated testing",
			Reason:  "no CI configuration detected",
		})
	}

	// 4. No LICENSE file -> FEEDBACK
	if !hasFile(target, "LICENSE") {
		suggestions = append(suggestions, pheromoneSuggestion{
			Type:    "FEEDBACK",
			Content: "consider adding a LICENSE file",
			Reason:  "no LICENSE file detected",
		})
	}

	// 5. No README.md -> FEEDBACK
	if !hasFile(target, "README.md") {
		suggestions = append(suggestions, pheromoneSuggestion{
			Type:    "FEEDBACK",
			Content: "consider adding a README.md for project documentation",
			Reason:  "no README.md detected",
		})
	}

	// 6. package.json without lockfile -> FEEDBACK
	if hasFile(target, "package.json") && !hasFile(target, "package-lock.json") && !hasFile(target, "yarn.lock") && !hasFile(target, "pnpm-lock.yaml") {
		suggestions = append(suggestions, pheromoneSuggestion{
			Type:    "FEEDBACK",
			Content: "consider locking dependency versions with a lockfile",
			Reason:  "package.json exists without lockfile",
		})
	}

	// 7. go.mod without go.sum -> REDIRECT
	if hasFile(target, "go.mod") && !hasFile(target, "go.sum") {
		suggestions = append(suggestions, pheromoneSuggestion{
			Type:    "REDIRECT",
			Content: "go.sum is required for reproducible Go builds",
			Reason:  "go.mod exists without go.sum",
		})
	}

	// 8. Dockerfile without .dockerignore -> FOCUS
	if hasFile(target, "Dockerfile") && !hasFile(target, ".dockerignore") {
		suggestions = append(suggestions, pheromoneSuggestion{
			Type:    "FOCUS",
			Content: "add .dockerignore to reduce Docker build context size",
			Reason:  "Dockerfile exists without .dockerignore",
		})
	}

	// 9. Test files detected -> FOCUS about CI testing
	hasTests := len(governance.TestFrameworks) > 0 || hasFile(target, "test") || hasFile(target, "tests") || hasFile(target, "__tests__")
	if !hasTests {
		// Check for Go test files
		entries, err := os.ReadDir(target)
		if err == nil {
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), "_test.go") {
					hasTests = true
					break
				}
			}
		}
	}
	if hasTests {
		suggestions = append(suggestions, pheromoneSuggestion{
			Type:    "FOCUS",
			Content: "test directory detected -- ensure tests are part of CI pipeline",
			Reason:  "test infrastructure detected in project",
		})
	}

	// 10. No formatter config -> FEEDBACK
	hasFormatter := len(governance.Formatters) > 0 || hasFile(target, ".editorconfig")
	if !hasFormatter {
		suggestions = append(suggestions, pheromoneSuggestion{
			Type:    "FEEDBACK",
			Content: "consider adding code formatting configuration for consistency",
			Reason:  "no formatter or editorconfig detected",
		})
	}

	return suggestions
}

// generateCharter produces charter data from scan results.
func generateCharter(goal, detected string, governance governanceInfo, readmeSummary string, gitHistory gitHistoryInfo) charterData {
	ch := charterData{}

	// Intent: use the goal string directly
	ch.Intent = goal
	if ch.Intent == "" {
		ch.Intent = "Build and ship quality software"
	}

	// Vision: combine detected type with governance tools
	var govTools []string
	govTools = append(govTools, governance.Linters...)
	govTools = append(govTools, governance.Formatters...)
	govTools = append(govTools, governance.CIConfigs...)
	if len(govTools) > 0 {
		ch.Vision = "A " + detected + " project with " + joinWithCommaAnd(govTools)
	} else {
		ch.Vision = "A " + detected + " project"
	}

	// Governance: list all detected categories
	var govParts []string
	if len(governance.Linters) > 0 {
		govParts = append(govParts, "Linting: "+strings.Join(governance.Linters, ", "))
	}
	if len(governance.CIConfigs) > 0 {
		govParts = append(govParts, "CI: "+strings.Join(governance.CIConfigs, ", "))
	}
	if len(governance.TestFrameworks) > 0 {
		govParts = append(govParts, "Testing: "+strings.Join(governance.TestFrameworks, ", "))
	}
	if len(governance.Formatters) > 0 {
		govParts = append(govParts, "Formatting: "+strings.Join(governance.Formatters, ", "))
	}
	if len(governance.BuildTools) > 0 {
		govParts = append(govParts, "Build: "+strings.Join(governance.BuildTools, ", "))
	}
	if len(govParts) > 0 {
		ch.Governance = strings.Join(govParts, ". ")
	} else {
		ch.Governance = "No formal governance detected -- colony should establish conventions"
	}

	// Goals
	ch.Goals = "Goal: " + goal + ". Focus on quality, maintainability, and shipping working software."

	return ch
}

// joinWithCommaAnd joins items with ", " and " and " before the last.
func joinWithCommaAnd(items []string) string {
	if len(items) == 0 {
		return ""
	}
	if len(items) == 1 {
		return items[0]
	}
	if len(items) == 2 {
		return items[0] + " and " + items[1]
	}
	return strings.Join(items[:len(items)-1], ", ") + ", and " + items[len(items)-1]
}

var initResearchCmd = &cobra.Command{
	Use:   "init-research",
	Short: "Perform initial research for colony setup",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		goal := mustGetString(cmd, "goal")
		if goal == "" {
			return nil
		}

		target, _ := cmd.Flags().GetString("target")
		if target == "" {
			target = "."
		}

		languages := []string{}
		frameworks := []string{}
		detected := ""
		topLevelDirs := []string{}
		isGitRepo := false
		fileCount := 0
		totalDirs := 0
		var readmeSummary string
		var largestFiles []fileInfo

		entries, err := os.ReadDir(target)
		if err != nil {
			outputError(1, "failed to read directory", nil)
			return nil
		}

		entryNames := make(map[string]bool)
		for _, e := range entries {
			if e.IsDir() {
				if !strings.HasPrefix(e.Name(), ".") {
					topLevelDirs = append(topLevelDirs, e.Name())
				}
				if e.Name() == ".git" {
					isGitRepo = true
				}
			} else {
				entryNames[e.Name()] = true
			}
		}

		seenTypes := make(map[string]bool)
		seenFrameworks := make(map[string]bool)

		for _, det := range projectDetectors {
			if entryNames[det.file] {
				if !seenTypes[det.typ] {
					languages = append(languages, det.typ)
					seenTypes[det.typ] = true
				}
				if detected == "" {
					detected = det.typ
				}
				for _, fw := range det.frameworks {
					if !seenFrameworks[fw] {
						frameworks = append(frameworks, fw)
						seenFrameworks[fw] = true
					}
				}
			}
		}

		// Normalize detected type
		if detected == "" {
			detected = "unknown"
		}

		// Recursive walk with extended skip list
		filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if extendedSkipDirs[d.Name()] && path != target {
					return filepath.SkipDir
				}
				totalDirs++
				return nil
			}

			fileCount++

			// Read README.md summary (first 500 chars)
			if strings.EqualFold(d.Name(), "README.md") {
				data, err := os.ReadFile(path)
				if err == nil && len(data) > 0 {
					if len(data) > 500 {
						data = data[:500]
					}
					readmeSummary = string(data)
				}
			}

			// Track largest files (keep top 5)
			info, err := d.Info()
			if err == nil {
				largestFiles = append(largestFiles, fileInfo{
					Path: path,
					Size: info.Size(),
				})
			}

			return nil
		})

		// Sort largest files descending by size, keep top 5
		sort.Slice(largestFiles, func(i, j int) bool {
			return largestFiles[i].Size > largestFiles[j].Size
		})
		if len(largestFiles) > 5 {
			largestFiles = largestFiles[:5]
		}

		// Run deep scan functions
		governance := detectGovernance(target)
		gitHistory := analyzeGitHistory(target)
		priorColonies := detectPriorColonies(target)
		pheromoneSuggestions := generatePheromoneSuggestions(target, governance)
		charter := generateCharter(goal, detected, governance, readmeSummary, gitHistory)

		complexity := complexityMetrics{
			TotalFiles:   fileCount,
			TotalDirs:    totalDirs,
			LargestFiles: largestFiles,
		}

		outputOK(map[string]interface{}{
			"detected_type":         detected,
			"languages":             languages,
			"frameworks":            frameworks,
			"goal":                  goal,
			"top_level_dirs":        topLevelDirs,
			"file_count":            fileCount,
			"is_git_repo":           isGitRepo,
			"readme_summary":        readmeSummary,
			"git_history":           gitHistory,
			"governance":            governance,
			"complexity":            complexity,
			"prior_colonies":        priorColonies,
			"pheromone_suggestions": pheromoneSuggestions,
			"charter":               charter,
		})
		return nil
	},
}

func init() {
	initResearchCmd.Flags().String("goal", "", "Colony goal (required)")
	initResearchCmd.Flags().String("target", "", "Directory to scan (default: current directory)")

	rootCmd.AddCommand(initResearchCmd)
}

// hasSuffix checks if s has any of the given suffixes.
func hasSuffix(s string, suffixes []string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(s, suffix) {
			return true
		}
	}
	return false
}

// resolveFileList expands a glob pattern and returns matching file paths.
func resolveFileList(pattern string) ([]string, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	return matches, nil
}
