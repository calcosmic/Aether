package cmd

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
	"github.com/tidwall/gjson"
	"gopkg.in/yaml.v3"
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

// depEntry represents a single parsed dependency.
type depEntry struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// techStackDetail holds parsed dependency data for a single language ecosystem.
type techStackDetail struct {
	Language   string     `json:"language"`
	SourceFile string     `json:"source_file"`
	Deps       []depEntry `json:"dependencies"`
	DevDeps    []depEntry `json:"dev_dependencies,omitempty"`
	Indirect   []depEntry `json:"indirect,omitempty"`
}

// dirClassification represents the detected directory structure type and the signals that led to it.
type dirClassification struct {
	Type    string   `json:"type"`
	Signals []string `json:"signals"`
}

// governanceDetail holds extracted rules/settings from a governance config file.
type governanceDetail struct {
	Tool     string                 `json:"tool"`
	File     string                 `json:"file"`
	Category string                 `json:"category"`
	Rules    map[string]interface{} `json:"rules,omitempty"`
	Extends  []string               `json:"extends,omitempty"`
	Config   map[string]interface{} `json:"config,omitempty"`
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

// hasDir checks whether a directory exists at target/name.
func hasDir(target, name string) bool {
	info, err := os.Stat(filepath.Join(target, name))
	return err == nil && info.IsDir()
}

// readFileContent reads a file and returns its content as a string.
// Returns empty string on error or if file exceeds 1MB (DoS mitigation).
func readFileContent(target, name string) string {
	data, err := os.ReadFile(filepath.Join(target, name))
	if err != nil || len(data) > maxDepFileSize {
		return ""
	}
	return string(data)
}

// fileContains checks whether a file at target/name contains the given substring.
func fileContains(target, name, substr string) bool {
	data, err := os.ReadFile(filepath.Join(target, name))
	if err != nil {
		return false
	}
	return strings.Contains(string(data), substr)
}

// classifyDirectory determines the directory structure type of a project.
// It checks monorepo, microservices, standard_app, library, and unknown patterns in order.
func classifyDirectory(target string) dirClassification {
	var signals []string

	// Monorepo signals (check first -- most specific)
	type monoSignal struct {
		path, label string
	}
	monoSignals := []monoSignal{
		{"packages", "packages/ directory found"},
		{"apps", "apps/ directory found"},
		{"pnpm-workspace.yaml", "pnpm-workspace.yaml detected"},
		{"lerna.json", "lerna.json detected"},
		{"nx.json", "nx.json detected"},
		{"turbo.json", "turbo.json detected"},
	}
	monoCount := 0
	for _, sig := range monoSignals {
		if hasDir(target, sig.path) || hasFile(target, sig.path) {
			signals = append(signals, sig.label)
			monoCount++
		}
	}
	if monoCount >= 1 {
		return dirClassification{Type: "monorepo", Signals: signals}
	}

	// Microservices signals: 2+ Dockerfiles in immediate subdirectories or root
	dockerfiles, _ := filepath.Glob(filepath.Join(target, "*", "Dockerfile*"))
	rootDockerfiles, _ := filepath.Glob(filepath.Join(target, "Dockerfile*"))
	dockerCount := len(dockerfiles) + len(rootDockerfiles)
	if dockerCount >= 2 {
		signals = append(signals, fmt.Sprintf("%d Dockerfiles detected", dockerCount))
		return dirClassification{Type: "microservices", Signals: signals}
	}

	// Standard app signals
	appDirs := []string{"src", "lib", "cmd", "app"}
	for _, d := range appDirs {
		if hasDir(target, d) {
			signals = append(signals, d+"/ directory found")
		}
	}
	if len(signals) >= 1 {
		return dirClassification{Type: "standard_app", Signals: signals}
	}

	// Library signals: no src/ or cmd/ dir, but entry point in root
	if !hasDir(target, "src") && !hasDir(target, "cmd") {
		libEntries := []string{"index.js", "index.ts", "main.go", "lib.rs"}
		for _, entry := range libEntries {
			if hasFile(target, entry) {
				signals = append(signals, "entry point in root, no src/ directory")
				return dirClassification{Type: "library", Signals: signals}
			}
		}
	}

	return dirClassification{Type: "unknown", Signals: []string{"no strong structural signals detected"}}
}

// --- Deep governance parsers ---

// parseEslintrcDeep parses ESLint config files for rules and extends.
func parseEslintrcDeep(target string) *governanceDetail {
	for _, name := range []string{".eslintrc.json", ".eslintrc.js", ".eslintrc.yml", ".eslintrc"} {
		path := filepath.Join(target, name)
		data, err := os.ReadFile(path)
		if err != nil || len(data) > maxDepFileSize {
			continue
		}

		detail := &governanceDetail{Tool: "ESLint", File: name, Category: "linter"}

		if strings.HasSuffix(name, ".json") || name == ".eslintrc" {
			parsed := gjson.ParseBytes(data)
			if rules := parsed.Get("rules"); rules.IsObject() {
				detail.Rules = make(map[string]interface{})
				rules.ForEach(func(k, v gjson.Result) bool {
					detail.Rules[k.String()] = v.Value()
					return true
				})
			}
			if extends := parsed.Get("extends"); extends.IsArray() {
				extends.ForEach(func(_, v gjson.Result) bool {
					detail.Extends = append(detail.Extends, v.String())
					return true
				})
			}
			if plugins := parsed.Get("plugins"); plugins.IsArray() {
				var pluginNames []string
				plugins.ForEach(func(_, v gjson.Result) bool {
					pluginNames = append(pluginNames, v.String())
					return true
				})
				if len(pluginNames) > 0 {
					if detail.Config == nil {
						detail.Config = make(map[string]interface{})
					}
					detail.Config["plugins"] = pluginNames
				}
			}
			return detail
		}

		if strings.HasSuffix(name, ".yml") {
			var raw map[string]interface{}
			if err := yaml.Unmarshal(data, &raw); err != nil {
				continue
			}
			if rules, ok := raw["rules"].(map[string]interface{}); ok {
				detail.Rules = rules
			}
			if extends, ok := raw["extends"].([]interface{}); ok {
				for _, e := range extends {
					if s, ok := e.(string); ok {
						detail.Extends = append(detail.Extends, s)
					}
				}
			}
			return detail
		}
	}
	return nil
}

// parseGolangciDeep parses golangci-lint config files for enabled linters and exclude rules.
func parseGolangciDeep(target string) *governanceDetail {
	for _, name := range []string{".golangci.yml", ".golangci.yaml", "golangci.yml"} {
		path := filepath.Join(target, name)
		data, err := os.ReadFile(path)
		if err != nil || len(data) > maxDepFileSize {
			continue
		}

		var raw map[string]interface{}
		if err := yaml.Unmarshal(data, &raw); err != nil {
			continue
		}

		detail := &governanceDetail{Tool: "golangci-lint", File: name, Category: "linter"}
		if detail.Config == nil {
			detail.Config = make(map[string]interface{})
		}

		// Extract enabled linters
		linters, _ := raw["linters"].(map[string]interface{})
		if enable, ok := linters["enable"].([]interface{}); ok {
			var names []string
			for _, e := range enable {
				if s, ok := e.(string); ok {
					names = append(names, s)
				}
			}
			detail.Config["enabled_linters"] = names
		}

		// Extract exclude rules
		issues, _ := raw["issues"].(map[string]interface{})
		if excludeRules, ok := issues["exclude-rules"].([]interface{}); ok {
			detail.Config["exclude_rules_count"] = len(excludeRules)
		}

		return detail
	}
	return nil
}

// parsePrettierDeep parses Prettier config files for formatting options.
func parsePrettierDeep(target string) *governanceDetail {
	for _, name := range []string{".prettierrc", ".prettierrc.json"} {
		path := filepath.Join(target, name)
		data, err := os.ReadFile(path)
		if err != nil || len(data) > maxDepFileSize {
			continue
		}

		parsed := gjson.ParseBytes(data)
		detail := &governanceDetail{Tool: "Prettier", File: name, Category: "formatter"}
		detail.Config = make(map[string]interface{})

		// Extract known Prettier options
		for _, key := range []string{"semi", "singleQuote", "tabWidth", "printWidth", "trailingComma", "useTabs", "bracketSpacing", "arrowParens", "endOfLine"} {
			if val := parsed.Get(key); val.Exists() {
				detail.Config[key] = val.Value()
			}
		}

		return detail
	}
	return nil
}

// parseBiomeDeep parses biome.json for formatter and linter configuration.
func parseBiomeDeep(target string) *governanceDetail {
	path := filepath.Join(target, "biome.json")
	data, err := os.ReadFile(path)
	if err != nil || len(data) > maxDepFileSize {
		return nil
	}

	parsed := gjson.ParseBytes(data)
	detail := &governanceDetail{Tool: "Biome", File: "biome.json", Category: "formatter"}
	detail.Config = make(map[string]interface{})

	if formatter := parsed.Get("formatter"); formatter.Exists() {
		var opts map[string]interface{}
		if err := json.Unmarshal([]byte(formatter.Raw), &opts); err == nil {
			detail.Config["formatter"] = opts
		}
	}

	if linter := parsed.Get("linter"); linter.Exists() {
		var opts map[string]interface{}
		if err := json.Unmarshal([]byte(linter.Raw), &opts); err == nil {
			detail.Config["linter"] = opts
		}
	}

	return detail
}

// parseJestDeep parses jest.config.js/ts for test configuration.
func parseJestDeep(target string) *governanceDetail {
	for _, name := range []string{"jest.config.js", "jest.config.ts", "jest.config.mjs"} {
		path := filepath.Join(target, name)
		data, err := os.ReadFile(path)
		if err != nil || len(data) > maxDepFileSize {
			continue
		}

		detail := &governanceDetail{Tool: "Jest", File: name, Category: "test"}
		detail.Config = make(map[string]interface{})

		// Use regex to extract config values from JS/TS files
		str := string(data)
		for _, key := range []string{"testMatch", "preset", "testEnvironment", "transform", "roots", "moduleNameMapper"} {
			re := regexp.MustCompile(key + `:\s*\[([^\]]+)\]`)
			if m := re.FindStringSubmatch(str); len(m) > 1 {
				values := strings.Split(strings.TrimSpace(m[1]), ",")
				var cleaned []string
				for _, v := range values {
					cleaned = append(cleaned, strings.TrimSpace(strings.Trim(v, "\"'`")))
				}
				detail.Config[key] = cleaned
			}
			// Also check string values
			reStr := regexp.MustCompile(key + `:\s*["']([^"']+)["']`)
			if m := reStr.FindStringSubmatch(str); len(m) > 1 {
				detail.Config[key] = m[1]
			}
		}

		return detail
	}
	return nil
}

// parseVitestDeep parses vitest.config.ts/js for test configuration.
func parseVitestDeep(target string) *governanceDetail {
	for _, name := range []string{"vitest.config.ts", "vitest.config.js"} {
		path := filepath.Join(target, name)
		data, err := os.ReadFile(path)
		if err != nil || len(data) > maxDepFileSize {
			continue
		}

		detail := &governanceDetail{Tool: "Vitest", File: name, Category: "test"}
		detail.Config = make(map[string]interface{})

		str := string(data)
		for _, key := range []string{"include", "exclude", "environment", "globals"} {
			re := regexp.MustCompile(key + `:\s*\[([^\]]+)\]`)
			if m := re.FindStringSubmatch(str); len(m) > 1 {
				values := strings.Split(strings.TrimSpace(m[1]), ",")
				var cleaned []string
				for _, v := range values {
					cleaned = append(cleaned, strings.TrimSpace(strings.Trim(v, "\"'`")))
				}
				detail.Config[key] = cleaned
			}
		}

		return detail
	}
	return nil
}

// parsePytestDeep parses pytest.ini or setup.cfg for pytest configuration.
func parsePytestDeep(target string) *governanceDetail {
	for _, name := range []string{"pytest.ini", "setup.cfg"} {
		path := filepath.Join(target, name)
		data, err := os.ReadFile(path)
		if err != nil || len(data) > maxDepFileSize {
			continue
		}

		detail := &governanceDetail{Tool: "pytest", File: name, Category: "test"}
		detail.Config = make(map[string]interface{})

		lines := strings.Split(string(data), "\n")
		inPytestSection := false
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "[tool:pytest]") || strings.HasPrefix(line, "[pytest]") {
				inPytestSection = true
				continue
			}
			if strings.HasPrefix(line, "[") && inPytestSection {
				break
			}
			if inPytestSection && strings.Contains(line, "=") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					val := strings.TrimSpace(parts[1])
					detail.Config[key] = val
				}
			}
			// pytest.ini uses simple key = value format without sections
			if name == "pytest.ini" && strings.Contains(line, "=") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					val := strings.TrimSpace(parts[1])
					detail.Config[key] = val
				}
			}
		}

		if len(detail.Config) > 0 {
			return detail
		}
	}
	return nil
}

// parseGHActionsDeep parses .github/workflows/*.yml files for CI configuration.
func parseGHActionsDeep(target string) []governanceDetail {
	workflowsDir := filepath.Join(target, ".github", "workflows")
	entries, err := os.ReadDir(workflowsDir)
	if err != nil {
		return nil
	}

	var details []governanceDetail
	count := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".yml") && !strings.HasSuffix(name, ".yaml") {
			continue
		}
		count++
		if count > 20 {
			break // cap at 20 workflows per T-73-05
		}

		path := filepath.Join(workflowsDir, name)
		data, err := os.ReadFile(path)
		if err != nil || len(data) > maxDepFileSize {
			continue
		}

		var raw map[string]interface{}
		if err := yaml.Unmarshal(data, &raw); err != nil {
			continue
		}

		detail := governanceDetail{
			Tool:     "GitHub Actions",
			File:     filepath.Join(".github/workflows", name),
			Category: "ci",
			Config:   make(map[string]interface{}),
		}

		if wfName, ok := raw["name"].(string); ok {
			detail.Config["name"] = wfName
		}
		if on, ok := raw["on"]; ok {
			detail.Config["triggers"] = on
		}
		if jobs, ok := raw["jobs"].(map[string]interface{}); ok {
			detail.Config["job_count"] = len(jobs)
			stepCount := 0
			for _, job := range jobs {
				if jobMap, ok := job.(map[string]interface{}); ok {
					if steps, ok := jobMap["steps"].([]interface{}); ok {
						stepCount += len(steps)
					}
				}
			}
			detail.Config["step_count"] = stepCount
		}

		details = append(details, detail)
	}
	return details
}

// parseGitlabCIDeep parses .gitlab-ci.yml for CI configuration.
func parseGitlabCIDeep(target string) *governanceDetail {
	path := filepath.Join(target, ".gitlab-ci.yml")
	data, err := os.ReadFile(path)
	if err != nil || len(data) > maxDepFileSize {
		return nil
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil
	}

	detail := &governanceDetail{
		Tool:     "GitLab CI",
		File:     ".gitlab-ci.yml",
		Category: "ci",
		Config:   make(map[string]interface{}),
	}

	if stages, ok := raw["stages"].([]interface{}); ok {
		detail.Config["stages"] = stages
	}

	// Count top-level job keys (exclude structural keys)
	skipKeys := map[string]bool{"stages": true, "variables": true, "default": true, "include": true, "image": true, "before_script": true, "after_script": true, "services": true, "cache": true}
	jobCount := 0
	for key := range raw {
		if !skipKeys[key] {
			jobCount++
		}
	}
	detail.Config["job_count"] = jobCount

	return detail
}

// parseJenkinsfileDeep parses Jenkinsfile for CI configuration.
func parseJenkinsfileDeep(target string) *governanceDetail {
	path := filepath.Join(target, "Jenkinsfile")
	data, err := os.ReadFile(path)
	if err != nil || len(data) > maxDepFileSize {
		return nil
	}

	detail := &governanceDetail{
		Tool:     "Jenkins",
		File:     "Jenkinsfile",
		Category: "ci",
		Config:   make(map[string]interface{}),
	}

	str := string(data)
	// Extract stage names
	stageRe := regexp.MustCompile(`stage\s*\(\s*['"]([^'"]+)['"]\s*\)`)
	matches := stageRe.FindAllStringSubmatch(str, -1)
	var stages []string
	for _, m := range matches {
		if len(m) > 1 {
			stages = append(stages, m[1])
		}
	}
	if len(stages) > 0 {
		detail.Config["stages"] = stages
	}

	// Check for agent directive
	agentRe := regexp.MustCompile(`agent\s+\{([^}]+)\}`)
	if m := agentRe.FindStringSubmatch(str); len(m) > 1 {
		detail.Config["agent"] = strings.TrimSpace(m[1])
	}

	return detail
}

// parseMakefileDeep parses Makefile for build targets.
func parseMakefileDeep(target string) *governanceDetail {
	path := filepath.Join(target, "Makefile")
	data, err := os.ReadFile(path)
	if err != nil || len(data) > maxDepFileSize {
		return nil
	}

	detail := &governanceDetail{
		Tool:     "Make",
		File:     "Makefile",
		Category: "build",
		Config:   make(map[string]interface{}),
	}

	re := regexp.MustCompile(`(?m)^([a-zA-Z0-9][a-zA-Z0-9_-]*):`)
	matches := re.FindAllStringSubmatch(string(data), -1)
	var targets []string
	for _, m := range matches {
		if len(m) > 1 {
			targets = append(targets, m[1])
		}
	}
	if len(targets) > 0 {
		detail.Config["targets"] = targets
	}

	return detail
}

// parseTaskfileDeep parses Taskfile.yml for task definitions.
func parseTaskfileDeep(target string) *governanceDetail {
	path := filepath.Join(target, "Taskfile.yml")
	data, err := os.ReadFile(path)
	if err != nil || len(data) > maxDepFileSize {
		return nil
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil
	}

	detail := &governanceDetail{
		Tool:     "Task",
		File:     "Taskfile.yml",
		Category: "build",
		Config:   make(map[string]interface{}),
	}

	if tasks, ok := raw["tasks"].(map[string]interface{}); ok {
		var names []string
		for name := range tasks {
			names = append(names, name)
		}
		sort.Strings(names)
		detail.Config["tasks"] = names
	}

	return detail
}

// parseJustfileDeep parses justfile for recipe definitions.
func parseJustfileDeep(target string) *governanceDetail {
	path := filepath.Join(target, "justfile")
	data, err := os.ReadFile(path)
	if err != nil || len(data) > maxDepFileSize {
		return nil
	}

	detail := &governanceDetail{
		Tool:     "Just",
		File:     "justfile",
		Category: "build",
		Config:   make(map[string]interface{}),
	}

	re := regexp.MustCompile(`(?m)^([a-zA-Z0-9][a-zA-Z0-9_-]*)\s*(?:\(|:)`)
	matches := re.FindAllStringSubmatch(string(data), -1)
	var recipes []string
	for _, m := range matches {
		if len(m) > 1 {
			recipes = append(recipes, m[1])
		}
	}
	if len(recipes) > 0 {
		detail.Config["recipes"] = recipes
	}

	return detail
}

// deepParseGovernance orchestrates all deep governance parsers across all 5 categories.
func deepParseGovernance(target string) []governanceDetail {
	var details []governanceDetail

	// Linter parsers
	if d := parseEslintrcDeep(target); d != nil {
		details = append(details, *d)
	}
	if d := parseGolangciDeep(target); d != nil {
		details = append(details, *d)
	}

	// Formatter parsers
	if d := parsePrettierDeep(target); d != nil {
		details = append(details, *d)
	}
	if d := parseBiomeDeep(target); d != nil {
		details = append(details, *d)
	}

	// Test framework parsers
	if d := parseJestDeep(target); d != nil {
		details = append(details, *d)
	}
	if d := parseVitestDeep(target); d != nil {
		details = append(details, *d)
	}
	if d := parsePytestDeep(target); d != nil {
		details = append(details, *d)
	}

	// CI parsers
	if ds := parseGHActionsDeep(target); len(ds) > 0 {
		details = append(details, ds...)
	}
	if d := parseGitlabCIDeep(target); d != nil {
		details = append(details, *d)
	}
	if d := parseJenkinsfileDeep(target); d != nil {
		details = append(details, *d)
	}

	// Build tool parsers
	if d := parseMakefileDeep(target); d != nil {
		details = append(details, *d)
	}
	if d := parseTaskfileDeep(target); d != nil {
		details = append(details, *d)
	}
	if d := parseJustfileDeep(target); d != nil {
		details = append(details, *d)
	}

	return details
}

// --- Dependency file parsers ---

// maxDepFileSize caps file reads for dependency files to prevent OOM on giant files.
const maxDepFileSize = 1 << 20 // 1 MB

// parsePackageJsonDeps parses package.json for production and dev dependencies.
func parsePackageJsonDeps(target string) ([]depEntry, []depEntry) {
	data, err := os.ReadFile(filepath.Join(target, "package.json"))
	if err != nil || len(data) > maxDepFileSize {
		return nil, nil
	}

	var prodDeps, devDeps []depEntry

	prodResult := gjson.GetBytes(data, "dependencies")
	if prodResult.IsObject() {
		prodResult.ForEach(func(key, val gjson.Result) bool {
			prodDeps = append(prodDeps, depEntry{Name: key.String(), Version: val.String()})
			return true
		})
	}

	devResult := gjson.GetBytes(data, "devDependencies")
	if devResult.IsObject() {
		devResult.ForEach(func(key, val gjson.Result) bool {
			devDeps = append(devDeps, depEntry{Name: key.String(), Version: val.String()})
			return true
		})
	}

	return prodDeps, devDeps
}

// parseGoModDeps parses go.mod for direct and indirect dependencies.
func parseGoModDeps(target string) ([]depEntry, []depEntry) {
	data, err := os.ReadFile(filepath.Join(target, "go.mod"))
	if err != nil || len(data) > maxDepFileSize {
		return nil, nil
	}

	var direct, indirect []depEntry
	lines := strings.Split(string(data), "\n")
	inRequireBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "require (" {
			inRequireBlock = true
			continue
		}
		if inRequireBlock && trimmed == ")" {
			inRequireBlock = false
			continue
		}
		if strings.HasPrefix(trimmed, "require ") || inRequireBlock {
			content := trimmed
			if strings.HasPrefix(content, "require ") {
				content = strings.TrimSpace(content[len("require "):])
			}
			parts := strings.Fields(content)
			if len(parts) >= 1 && !strings.HasPrefix(parts[0], "//") {
				name := parts[0]
				version := ""
				isIndirect := strings.Contains(content, "// indirect")
				if len(parts) >= 2 && !strings.HasPrefix(parts[1], "//") {
					version = parts[1]
				}
				entry := depEntry{Name: name, Version: version}
				if isIndirect {
					indirect = append(indirect, entry)
				} else {
					direct = append(direct, entry)
				}
			}
		}
	}
	return direct, indirect
}

// parseCargoTomlDeps parses Cargo.toml [dependencies] section.
func parseCargoTomlDeps(target string) []depEntry {
	data, err := os.ReadFile(filepath.Join(target, "Cargo.toml"))
	if err != nil || len(data) > maxDepFileSize {
		return nil
	}

	var raw map[string]interface{}
	if _, err := toml.Decode(string(data), &raw); err != nil {
		return nil
	}

	depsMap, ok := raw["dependencies"].(map[string]interface{})
	if !ok {
		return nil
	}

	var deps []depEntry
	for name, val := range depsMap {
		switch v := val.(type) {
		case string:
			deps = append(deps, depEntry{Name: name, Version: v})
		case map[string]interface{}:
			if ver, ok := v["version"].(string); ok {
				deps = append(deps, depEntry{Name: name, Version: ver})
			} else {
				deps = append(deps, depEntry{Name: name})
			}
		default:
			deps = append(deps, depEntry{Name: name})
		}
	}
	return deps
}

// parsePyprojectDeps parses pyproject.toml for project dependencies (PEP 621 and Poetry).
func parsePyprojectDeps(target string) []depEntry {
	data, err := os.ReadFile(filepath.Join(target, "pyproject.toml"))
	if err != nil || len(data) > maxDepFileSize {
		return nil
	}

	var raw map[string]interface{}
	if _, err := toml.Decode(string(data), &raw); err != nil {
		return nil
	}

	var deps []depEntry

	// PEP 621: [project.dependencies]
	if project, ok := raw["project"].(map[string]interface{}); ok {
		if depList, ok := project["dependencies"].([]interface{}); ok {
			for _, item := range depList {
				s, ok := item.(string)
				if !ok {
					continue
				}
				name, version := splitPepDep(s)
				deps = append(deps, depEntry{Name: name, Version: version})
			}
		}
	}

	// Poetry fallback: [tool.poetry.dependencies]
	if tool, ok := raw["tool"].(map[string]interface{}); ok {
		if poetry, ok := tool["poetry"].(map[string]interface{}); ok {
			if depMap, ok := poetry["dependencies"].(map[string]interface{}); ok {
				for name, val := range depMap {
					switch v := val.(type) {
					case string:
						deps = append(deps, depEntry{Name: name, Version: v})
					default:
						deps = append(deps, depEntry{Name: name})
					}
				}
			}
		}
	}

	return deps
}

// splitPepDep splits a PEP 508 dependency string like "requests>=2.0" into name and version.
func splitPepDep(s string) (string, string) {
	// Find the first version operator
	for i, r := range s {
		if r == '=' || r == '>' || r == '<' || r == '~' || r == '!' || r == ';' || r == ' ' {
			return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i:])
		}
	}
	return strings.TrimSpace(s), ""
}

// parseComposerJsonDeps parses composer.json for production and dev dependencies.
func parseComposerJsonDeps(target string) ([]depEntry, []depEntry) {
	data, err := os.ReadFile(filepath.Join(target, "composer.json"))
	if err != nil || len(data) > maxDepFileSize {
		return nil, nil
	}

	var prodDeps, devDeps []depEntry

	prodResult := gjson.GetBytes(data, "require")
	if prodResult.IsObject() {
		prodResult.ForEach(func(key, val gjson.Result) bool {
			prodDeps = append(prodDeps, depEntry{Name: key.String(), Version: val.String()})
			return true
		})
	}

	devResult := gjson.GetBytes(data, "require-dev")
	if devResult.IsObject() {
		devResult.ForEach(func(key, val gjson.Result) bool {
			devDeps = append(devDeps, depEntry{Name: key.String(), Version: val.String()})
			return true
		})
	}

	return prodDeps, devDeps
}

// parseRequirementsTxt parses requirements.txt for Python dependencies.
func parseRequirementsTxt(target string) []depEntry {
	data, err := os.ReadFile(filepath.Join(target, "requirements.txt"))
	if err != nil || len(data) > maxDepFileSize {
		return nil
	}

	var deps []depEntry
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-r") || strings.HasPrefix(line, "-e") || strings.HasPrefix(line, "--") {
			continue
		}
		name, version := splitPepDep(line)
		if name != "" {
			deps = append(deps, depEntry{Name: name, Version: version})
		}
	}
	return deps
}

// parseGemfileDeps parses a Ruby Gemfile using regex matching.
func parseGemfileDeps(target string) []depEntry {
	data, err := os.ReadFile(filepath.Join(target, "Gemfile"))
	if err != nil || len(data) > maxDepFileSize {
		return nil
	}

	var deps []depEntry
	re := regexp.MustCompile(`gem\s+["']([^"']+)["']\s*(?:,\s*["']([^"']+)["'])?`)
	matches := re.FindAllStringSubmatch(string(data), -1)
	for _, m := range matches {
		name := m[1]
		version := ""
		if len(m) > 2 {
			version = m[2]
		}
		deps = append(deps, depEntry{Name: name, Version: version})
	}
	return deps
}

// mavenProject is a minimal XML struct for parsing pom.xml.
type mavenProject struct {
	XMLName     xml.Name   `xml:"project"`
	Dependencies mavenDeps `xml:"dependencies"`
}

// mavenDeps holds Maven dependency entries.
type mavenDeps struct {
	Deps []mavenDep `xml:"dependency"`
}

// mavenDep represents a single Maven dependency.
type mavenDep struct {
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
	Version    string `xml:"version"`
}

// parsePomXmlDeps parses pom.xml for Java/Maven dependencies.
func parsePomXmlDeps(target string) []depEntry {
	data, err := os.ReadFile(filepath.Join(target, "pom.xml"))
	if err != nil || len(data) > maxDepFileSize {
		return nil
	}

	var project mavenProject
	if err := xml.Unmarshal(data, &project); err != nil {
		return nil
	}

	var deps []depEntry
	for _, d := range project.Dependencies.Deps {
		name := d.ArtifactID
		if d.GroupID != "" {
			name = d.GroupID + ":" + d.ArtifactID
		}
		deps = append(deps, depEntry{Name: name, Version: d.Version})
	}
	return deps
}

// parseMixExsDeps parses mix.exs for Elixir dependencies.
func parseMixExsDeps(target string) []depEntry {
	data, err := os.ReadFile(filepath.Join(target, "mix.exs"))
	if err != nil || len(data) > maxDepFileSize {
		return nil
	}

	content := string(data)

	// Find content between "defp deps do" and the matching "end"
	start := strings.Index(content, "defp deps do")
	if start == -1 {
		return nil
	}
	rest := content[start+len("defp deps do"):]
	end := strings.Index(rest, "end")
	if end == -1 {
		return nil
	}
	block := rest[:end]

	var deps []depEntry
	re := regexp.MustCompile(`\{:(\w+)`)
	matches := re.FindAllStringSubmatch(block, -1)
	for _, m := range matches {
		if len(m) > 1 {
			deps = append(deps, depEntry{Name: m[1]})
		}
	}
	return deps
}

// parseDependencyFiles orchestrates all dependency file parsers and returns structured results.
func parseDependencyFiles(target string) []techStackDetail {
	var details []techStackDetail

	if hasFile(target, "package.json") {
		prod, dev := parsePackageJsonDeps(target)
		details = append(details, techStackDetail{
			Language: "node", SourceFile: "package.json",
			Deps: prod, DevDeps: dev,
		})
	}
	if hasFile(target, "go.mod") {
		direct, indirect := parseGoModDeps(target)
		details = append(details, techStackDetail{
			Language: "go", SourceFile: "go.mod",
			Deps: direct, Indirect: indirect,
		})
	}
	if hasFile(target, "Cargo.toml") {
		deps := parseCargoTomlDeps(target)
		details = append(details, techStackDetail{
			Language: "rust", SourceFile: "Cargo.toml",
			Deps: deps,
		})
	}
	if hasFile(target, "pyproject.toml") {
		deps := parsePyprojectDeps(target)
		details = append(details, techStackDetail{
			Language: "python", SourceFile: "pyproject.toml",
			Deps: deps,
		})
	}
	if hasFile(target, "composer.json") {
		prod, dev := parseComposerJsonDeps(target)
		details = append(details, techStackDetail{
			Language: "php", SourceFile: "composer.json",
			Deps: prod, DevDeps: dev,
		})
	}
	if hasFile(target, "requirements.txt") {
		deps := parseRequirementsTxt(target)
		details = append(details, techStackDetail{
			Language: "python", SourceFile: "requirements.txt",
			Deps: deps,
		})
	}
	if hasFile(target, "Gemfile") {
		deps := parseGemfileDeps(target)
		details = append(details, techStackDetail{
			Language: "ruby", SourceFile: "Gemfile",
			Deps: deps,
		})
	}
	if hasFile(target, "pom.xml") {
		deps := parsePomXmlDeps(target)
		details = append(details, techStackDetail{
			Language: "java", SourceFile: "pom.xml",
			Deps: deps,
		})
	}
	if hasFile(target, "mix.exs") {
		deps := parseMixExsDeps(target)
		details = append(details, techStackDetail{
			Language: "elixir", SourceFile: "mix.exs",
			Deps: deps,
		})
	}

	return details
}

// generatePheromoneSuggestions applies deterministic patterns to produce pheromone suggestions.
// It checks ~25 patterns covering security, governance, documentation, containers, and more.
func generatePheromoneSuggestions(target string, governance governanceInfo, dirClass dirClassification, techStack []techStackDetail) []pheromoneSuggestion {
	var suggestions []pheromoneSuggestion

	// --- Original 10 patterns (preserved unchanged) ---

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

	// --- New patterns (D-07) ---

	// --- Monorepo workspace patterns (D-07) ---
	// 11. Monorepo without workspace tooling
	if dirClass.Type == "monorepo" && !hasFile(target, "pnpm-workspace.yaml") && !hasFile(target, "lerna.json") && !hasFile(target, "turbo.json") && !hasFile(target, "nx.json") {
		suggestions = append(suggestions, pheromoneSuggestion{
			Type:    "FOCUS",
			Content: "consider adopting a workspace management tool for monorepo consistency",
			Reason:  "monorepo detected without pnpm workspaces, lerna, turbo, or nx",
		})
	}

	// 12. Workspace config found
	if hasFile(target, "pnpm-workspace.yaml") || hasFile(target, "turbo.json") || hasFile(target, "nx.json") {
		suggestions = append(suggestions, pheromoneSuggestion{
			Type:    "FEEDBACK",
			Content: "workspace management detected -- maintain consistent dependency versions across packages",
			Reason:  "workspace config detected in monorepo",
		})
	}

	// --- API patterns (D-07) ---
	// 13. OpenAPI/Swagger spec detected
	if hasFile(target, "openapi.yaml") || hasFile(target, "openapi.yml") || hasFile(target, "swagger.yaml") || hasFile(target, "swagger.json") {
		suggestions = append(suggestions, pheromoneSuggestion{
			Type:    "FOCUS",
			Content: "API specification detected -- ensure implementations stay aligned with spec",
			Reason:  "OpenAPI or Swagger specification file found",
		})
	}

	// 14. API route patterns without spec
	if (hasDir(target, "routes") || hasDir(target, "api")) && !hasFile(target, "openapi.yaml") && !hasFile(target, "openapi.yml") && !hasFile(target, "swagger.yaml") && !hasFile(target, "swagger.json") {
		suggestions = append(suggestions, pheromoneSuggestion{
			Type:    "FEEDBACK",
			Content: "API routes detected without an API specification -- consider adding OpenAPI docs",
			Reason:  "routes/ or api/ directory found without API spec",
		})
	}

	// --- Database presence (D-07) ---
	// 15. Migration directory detected
	if hasDir(target, "migrations") || hasDir(target, "db/migrations") || hasDir(target, "prisma/migrations") || hasDir(target, "alembic") {
		suggestions = append(suggestions, pheromoneSuggestion{
			Type:    "FOCUS",
			Content: "database migrations detected -- ensure schema changes go through migration workflow",
			Reason:  "migration directory found in project",
		})
	}

	// 16. Schema file detected
	if hasFile(target, "schema.prisma") || hasFile(target, "schema.rb") {
		suggestions = append(suggestions, pheromoneSuggestion{
			Type:    "FOCUS",
			Content: "database schema definition detected -- treat schema as code, review changes carefully",
			Reason:  "schema definition file found",
		})
	}

	// --- Security patterns (D-07) ---
	// 17. CORS config detected
	if fileContains(target, ".env", "CORS") || fileContains(target, "next.config.js", "headers") || fileContains(target, "next.config.mjs", "headers") || hasFile(target, "cors.json") {
		suggestions = append(suggestions, pheromoneSuggestion{
			Type:    "FEEDBACK",
			Content: "CORS configuration detected -- review allowed origins to prevent over-permissive access",
			Reason:  "CORS-related configuration found",
		})
	}

	// 18. .env.example exists but .env doesn't (setup guidance)
	if hasFile(target, ".env.example") && !hasFile(target, ".env") && !hasFile(target, ".env.local") {
		suggestions = append(suggestions, pheromoneSuggestion{
			Type:    "FEEDBACK",
			Content: ".env.example found -- copy to .env and fill in values for local development",
			Reason:  ".env.example exists without corresponding .env",
		})
	}

	// --- Container patterns (D-07) ---
	// 19. docker-compose detected
	if hasFile(target, "docker-compose.yml") || hasFile(target, "docker-compose.yaml") || hasFile(target, "compose.yml") || hasFile(target, "compose.yaml") {
		suggestions = append(suggestions, pheromoneSuggestion{
			Type:    "FOCUS",
			Content: "docker compose detected -- ensure services are reproducible and health-checked",
			Reason:  "docker-compose file found",
		})
	}

	// 20. Multi-stage Dockerfile
	dockerfileContent := readFileContent(target, "Dockerfile")
	if dockerfileContent != "" && strings.Count(dockerfileContent, "FROM") > 1 {
		suggestions = append(suggestions, pheromoneSuggestion{
			Type:    "FEEDBACK",
			Content: "multi-stage Docker build detected -- optimize layer caching for faster builds",
			Reason:  "Dockerfile uses multiple FROM stages",
		})
	}

	// --- Documentation patterns (D-07) ---
	// 21. CHANGELOG.md detected
	if hasFile(target, "CHANGELOG.md") {
		suggestions = append(suggestions, pheromoneSuggestion{
			Type:    "FEEDBACK",
			Content: "CHANGELOG.md found -- keep it updated with each release for traceability",
			Reason:  "changelog file detected",
		})
	}

	// 22. No documentation at all
	if !hasFile(target, "README.md") && !hasDir(target, "docs") && !hasDir(target, "doc") {
		suggestions = append(suggestions, pheromoneSuggestion{
			Type:    "FEEDBACK",
			Content: "no documentation detected -- consider adding README.md or a docs/ directory",
			Reason:  "no README, docs, or documentation directory found",
		})
	}

	// --- Dependency health (D-07) ---
	// 23. Go project without linter
	hasGoLinter := false
	for _, l := range governance.Linters {
		if l == "golangci-lint" {
			hasGoLinter = true
		}
	}
	if hasFile(target, "go.mod") && !hasGoLinter {
		suggestions = append(suggestions, pheromoneSuggestion{
			Type:    "FEEDBACK",
			Content: "consider adding golangci-lint for Go code quality checks",
			Reason:  "Go project without configured linter",
		})
	}

	// 24. TypeScript detected without tsconfig.json
	hasTS := false
	for _, ts := range techStack {
		if ts.Language == "node" {
			for _, dep := range ts.DevDeps {
				if strings.HasPrefix(dep.Name, "typescript") {
					hasTS = true
					break
				}
			}
			if hasTS {
				break
			}
			for _, dep := range ts.Deps {
				if strings.HasPrefix(dep.Name, "typescript") {
					hasTS = true
					break
				}
			}
		}
	}
	if hasTS && hasFile(target, "tsconfig.json") {
		suggestions = append(suggestions, pheromoneSuggestion{
			Type:    "FEEDBACK",
			Content: "TypeScript detected -- ensure strict mode is enabled in tsconfig.json",
			Reason:  "tsconfig.json found in TypeScript project",
		})
	}

	// 25. Large number of dependencies
	totalDeps := 0
	for _, ts := range techStack {
		totalDeps += len(ts.Deps) + len(ts.DevDeps) + len(ts.Indirect)
	}
	if totalDeps > 100 {
		suggestions = append(suggestions, pheromoneSuggestion{
			Type:    "FOCUS",
			Content: "high dependency count detected -- regularly audit for unused or vulnerable packages",
			Reason:  fmt.Sprintf("%d total dependencies found", totalDeps),
		})
	}

	return suggestions
}

// generateCharter produces charter data from scan results.
func generateCharter(goal, detected string, governance governanceInfo, readmeSummary string, gitHistory gitHistoryInfo, languages []string, frameworks []string, isGitRepo bool, pheromoneSuggestions []pheromoneSuggestion) colony.Charter {
	ch := colony.Charter{}

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

	// TechStack
	ch.TechStack = generateTechStack(languages, frameworks)

	// KeyRisks
	ch.KeyRisks = generateKeyRisks(governance, isGitRepo, pheromoneSuggestions)

	// Constraints
	ch.Constraints = generateConstraints(governance)

	return ch
}

// generateTechStack builds a tech stack description from detected languages and frameworks.
func generateTechStack(languages []string, frameworks []string) string {
	// Deduplicate frameworks that overlap with languages
	langSet := make(map[string]bool)
	for _, l := range languages {
		langSet[l] = true
	}
	var uniqueFrameworks []string
	for _, fw := range frameworks {
		if !langSet[fw] {
			uniqueFrameworks = append(uniqueFrameworks, fw)
		}
	}

	var parts []string
	if len(languages) > 0 {
		parts = append(parts, "Languages: "+strings.Join(languages, ", "))
	}
	if len(uniqueFrameworks) > 0 {
		parts = append(parts, "Frameworks/Tools: "+strings.Join(uniqueFrameworks, ", "))
	}

	if len(parts) > 0 {
		return strings.Join(parts, ". ")
	}
	return "No specific tech stack detected"
}

// generateKeyRisks produces risk heuristics from governance data and pheromone suggestions.
func generateKeyRisks(governance governanceInfo, isGitRepo bool, pheromoneSuggestions []pheromoneSuggestion) string {
	var risks []string

	if len(governance.CIConfigs) == 0 {
		risks = append(risks, "No CI/CD pipeline detected -- manual deployment risk")
	}
	if len(governance.TestFrameworks) == 0 {
		risks = append(risks, "No test framework detected -- regression risk")
	}
	if len(governance.Linters) == 0 {
		risks = append(risks, "No linter configured -- code quality may drift")
	}

	// Check pheromone suggestions for secret-related REDIRECT
	for _, sug := range pheromoneSuggestions {
		if sug.Type == "REDIRECT" && strings.Contains(sug.Content, "secrets") {
			risks = append(risks, "Potential secret exposure -- .env files without .gitignore protection")
			break
		}
	}

	if !isGitRepo {
		risks = append(risks, "Not a git repository -- no version control")
	}

	if len(risks) > 0 {
		return strings.Join(risks, ". ")
	}
	return "No significant risks detected from initial scan"
}

// generateConstraints produces constraint descriptions from governance data.
func generateConstraints(governance governanceInfo) string {
	var parts []string

	if len(governance.Linters) > 0 {
		parts = append(parts, "Follow "+strings.Join(governance.Linters, "/")+" rules")
	}
	if len(governance.Formatters) > 0 {
		parts = append(parts, "Use "+strings.Join(governance.Formatters, "/")+" for code formatting")
	}
	if len(governance.TestFrameworks) > 0 {
		parts = append(parts, "Write tests using "+strings.Join(governance.TestFrameworks, "/"))
	}
	if len(governance.BuildTools) > 0 {
		parts = append(parts, "Build with "+strings.Join(governance.BuildTools, "/"))
	}

	if len(parts) > 0 {
		return strings.Join(parts, ". ")
	}
	return "No formal constraints detected -- colony should establish conventions"
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

// colonyContextSummary provides a formatted summary of all init-research sections.
// It is included in the outputOK envelope and automatically available to the
// init ceremony via the JSON envelope consumption pattern (PATTERNS.md).
type colonyContextSummary struct {
	DetectedType        string   `json:"detected_type"`
	Languages           []string `json:"languages"`
	DirType             string   `json:"dir_type"`
	DirSignals          []string `json:"dir_signals"`
	TechStackCount      int      `json:"tech_stack_count"`
	GovernanceToolCount int      `json:"governance_tool_count"`
	PheromoneCount      int      `json:"pheromone_count"`
	IsGitRepo           bool     `json:"is_git_repo"`
	FileCount           int      `json:"file_count"`
}

// generateColonyContextSummary builds a colony context summary from all scan results.
func generateColonyContextSummary(detected string, languages []string, dirClass dirClassification, techStack []techStackDetail, governance governanceInfo, pheromoneSuggestions []pheromoneSuggestion, isGitRepo bool, fileCount int) colonyContextSummary {
	govToolCount := len(governance.Linters) + len(governance.Formatters) + len(governance.TestFrameworks) + len(governance.CIConfigs) + len(governance.BuildTools)
	return colonyContextSummary{
		DetectedType:        detected,
		Languages:           languages,
		DirType:             dirClass.Type,
		DirSignals:          dirClass.Signals,
		TechStackCount:      len(techStack),
		GovernanceToolCount: govToolCount,
		PheromoneCount:      len(pheromoneSuggestions),
		IsGitRepo:           isGitRepo,
		FileCount:           fileCount,
	}
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

		complexity := complexityMetrics{
			TotalFiles:   fileCount,
			TotalDirs:    totalDirs,
			LargestFiles: largestFiles,
		}

		techStackDetail := parseDependencyFiles(target)
		dirClass := classifyDirectory(target)
		governanceDetails := deepParseGovernance(target)
		pheromoneSuggestions := generatePheromoneSuggestions(target, governance, dirClass, techStackDetail)
		charter := generateCharter(goal, detected, governance, readmeSummary, gitHistory, languages, frameworks, isGitRepo, pheromoneSuggestions)
		contextSummary := generateColonyContextSummary(detected, languages, dirClass, techStackDetail, governance, pheromoneSuggestions, isGitRepo, fileCount)

		outputOK(map[string]interface{}{
			"detected_type":          detected,
			"languages":              languages,
			"frameworks":             frameworks,
			"goal":                   goal,
			"top_level_dirs":         topLevelDirs,
			"file_count":             fileCount,
			"is_git_repo":            isGitRepo,
			"readme_summary":         readmeSummary,
			"git_history":            gitHistory,
			"governance":             governance,
			"complexity":             complexity,
			"prior_colonies":         priorColonies,
			"pheromone_suggestions":  pheromoneSuggestions,
			"charter":                charter,
			"tech_stack_detail":      techStackDetail,
			"dir_classification":     dirClass,
			"governance_details":     governanceDetails,
			"colony_context_summary": contextSummary,
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
