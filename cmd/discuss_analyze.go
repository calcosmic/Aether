package cmd

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

const analyzeSourcePrefix = "analyze:"

type analyzeScanData struct {
	DetectedType string        `json:"detected_type"`
	Languages    []string      `json:"languages"`
	Frameworks   []string      `json:"frameworks"`
	Goal         string        `json:"goal,omitempty"`
	TopLevelDirs []string      `json:"top_level_dirs"`
	FileCount    int           `json:"file_count"`
	TotalDirs    int           `json:"total_dirs"`
	IsGitRepo    bool          `json:"is_git_repo"`
	Governance   governanceInfo `json:"governance"`
	HasDocker    bool          `json:"has_docker"`
	HasDockerCompose bool     `json:"has_docker_compose"`
	HasK8s       bool          `json:"has_k8s"`
	HasMakefile  bool          `json:"has_makefile"`
}

var discussAnalyzeCmd = &cobra.Command{
	Use:   "discuss-analyze",
	Short: "Analyze codebase and generate suggested discussion questions",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		target, _ := cmd.Flags().GetString("target")
		if target == "" {
			target = "."
		}
		goal, _ := cmd.Flags().GetString("goal")

		scan := runDiscussAnalyze(target, goal)
		questions := generateAnalyzeQuestions(scan)

		outputOK(map[string]interface{}{
			"scan":     scan,
			"questions": questions,
		})
		return nil
	},
}

func init() {
	discussAnalyzeCmd.Flags().String("target", "", "Directory to scan (default: current directory)")
	discussAnalyzeCmd.Flags().String("goal", "", "Colony goal for context")
	rootCmd.AddCommand(discussAnalyzeCmd)
}

// runDiscussAnalyze performs an inventory-only scan of the target directory.
// It reads directory entries, detects languages/frameworks, governance, and
// architecture patterns without reading source files (per D-01).
func runDiscussAnalyze(target, goal string) analyzeScanData {
	scan := analyzeScanData{
		Goal: goal,
	}

	entries, err := os.ReadDir(target)
	if err != nil {
		outputError(1, "failed to read directory", nil)
		return scan
	}

	entryNames := make(map[string]bool)
	for _, e := range entries {
		if e.IsDir() {
			if !strings.HasPrefix(e.Name(), ".") {
				scan.TopLevelDirs = append(scan.TopLevelDirs, e.Name())
			}
			if e.Name() == ".git" {
				scan.IsGitRepo = true
			}
		} else {
			entryNames[e.Name()] = true
		}
	}

	// Sort top-level dirs for deterministic output
	sort.Strings(scan.TopLevelDirs)

	// Detect languages and frameworks
	seenTypes := make(map[string]bool)
	seenFrameworks := make(map[string]bool)

	for _, det := range projectDetectors {
		if entryNames[det.file] {
			if !seenTypes[det.typ] {
				scan.Languages = append(scan.Languages, det.typ)
				seenTypes[det.typ] = true
			}
			if scan.DetectedType == "" {
				scan.DetectedType = det.typ
			}
			for _, fw := range det.frameworks {
				if !seenFrameworks[fw] {
					scan.Frameworks = append(scan.Frameworks, fw)
					seenFrameworks[fw] = true
				}
			}
		}
	}

	if scan.DetectedType == "" {
		scan.DetectedType = "unknown"
	}

	// Detect governance (test frameworks, linters, CI configs)
	scan.Governance = detectGovernance(target)

	// Detect architecture patterns (inventory-only: check for marker files)
	scan.HasDocker = hasFile(target, "Dockerfile")
	scan.HasDockerCompose = hasFile(target, "docker-compose.yml") || hasFile(target, "docker-compose.yaml")
	scan.HasK8s = hasFile(target, "k8s") || hasFile(target, "kubernetes") || hasFile(target, "deploy") || hasFile(target, "helm")
	scan.HasMakefile = hasFile(target, "Makefile")

	// Recursive walk for file count and total dirs (inventory only)
	filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if extendedSkipDirs[d.Name()] && path != target {
				return filepath.SkipDir
			}
			scan.TotalDirs++
			return nil
		}
		scan.FileCount++
		return nil
	})

	return scan
}

// generateAnalyzeQuestions produces suggested questions based on scan data.
// Categories are distinct from existing discuss categories (surface, integration,
// scope, verification) and cover architecture, dependencies, testing_infrastructure,
// deployment, and performance.
func generateAnalyzeQuestions(scan analyzeScanData) []discussQuestion {
	return []discussQuestion{
		buildAnalyzeArchitectureQuestion(scan),
		buildAnalyzeDependenciesQuestion(scan),
		buildAnalyzeTestingQuestion(scan),
		buildAnalyzeDeploymentQuestion(scan),
		buildAnalyzePerformanceQuestion(scan),
	}
}

func buildAnalyzeArchitectureQuestion(scan analyzeScanData) discussQuestion {
	options := []string{
		"monolithic single module",
		"modular with clear domain boundaries",
		"microservices with independent deployments",
	}

	reasoning := "The plan needs to know the target architecture to allocate work correctly."

	// Adjust options based on detected patterns
	if scan.HasDockerCompose || scan.HasK8s {
		options = []string{
			"keep current service architecture",
			"consolidate into fewer, larger services",
			"split into more focused services",
		}
		reasoning = "Docker Compose or Kubernetes manifests suggest a service-oriented architecture. The plan should confirm the target grain size."
	}

	// Check for multiple go.mod (monorepo indicator)
	if scan.HasMakefile && len(scan.TopLevelDirs) > 5 {
		options = []string{
			"single unified module",
			"monorepo with internal packages",
			"split into separate modules",
		}
		reasoning = "Multiple top-level directories with a Makefile suggest a monorepo layout. The plan should confirm whether to keep or restructure."
	}

	return discussQuestion{
		Category:  "architecture",
		Question:  "What architectural pattern should the plan assume?",
		Options:   options,
		Reasoning: reasoning,
		Source:    analyzeSource("architecture"),
	}
}

func buildAnalyzeDependenciesQuestion(scan analyzeScanData) discussQuestion {
	options := []string{
		"minimize new dependencies",
		"use existing project dependencies",
		"allow new dependencies if justified",
	}

	reasoning := "The plan needs a dependency policy to avoid scope creep from unnecessary libraries."

	if len(scan.Languages) > 1 {
		options = []string{
			"keep dependencies within each language ecosystem",
			"prefer cross-language compatibility",
			"minimize cross-language boundary dependencies",
		}
		reasoning = "Multiple languages detected (" + strings.Join(scan.Languages, ", ") + "). The plan should know how to handle cross-language dependency management."
	}

	if len(scan.Frameworks) > 0 {
		reasoning = "Detected frameworks (" + strings.Join(scan.Frameworks, ", ") + ") constrain the dependency space. The plan should know whether to stay within or extend the current framework set."
	}

	return discussQuestion{
		Category:  "dependencies",
		Question:  "How should new dependencies be managed?",
		Options:   options,
		Reasoning: reasoning,
		Source:    analyzeSource("dependencies"),
	}
}

func buildAnalyzeTestingQuestion(scan analyzeScanData) discussQuestion {
	options := []string{
		"follow existing test patterns",
		"add tests for new code only",
		"establish comprehensive test coverage",
	}

	reasoning := "The colony should know the testing expectation before writing code."

	if len(scan.Governance.TestFrameworks) > 0 {
		options = []string{
			"use existing " + strings.Join(scan.Governance.TestFrameworks, "/") + " conventions",
			"add integration tests alongside existing unit tests",
			"establish end-to-end test coverage",
		}
		reasoning = "Test frameworks detected (" + strings.Join(scan.Governance.TestFrameworks, ", ") + "). The plan should confirm whether to follow existing patterns or raise the bar."
	} else {
		options = []string{
			"add basic validation tests",
			"prototype first, add tests after",
			"establish a test framework before building",
		}
		reasoning = "No test frameworks detected. The plan should decide whether to front-load test infrastructure or defer it."
	}

	return discussQuestion{
		Category:  "testing_infrastructure",
		Question:  "What testing approach should the colony adopt?",
		Options:   options,
		Reasoning: reasoning,
		Source:    analyzeSource("testing_infrastructure"),
	}
}

func buildAnalyzeDeploymentQuestion(scan analyzeScanData) discussQuestion {
	options := []string{
		"local development only for now",
		"containerized with Docker",
		"cloud-native with CI/CD pipeline",
	}

	reasoning := "The plan needs to know the deployment target to structure build and release steps appropriately."

	if scan.HasDocker || scan.HasDockerCompose {
		options = []string{
			"build on existing Docker setup",
			"optimize container images for production",
			"add multi-stage build pipeline",
		}
		reasoning = "Docker detected in project. The plan should align with the existing containerization strategy."
	}

	if len(scan.Governance.CIConfigs) > 0 {
		options = []string{
			"integrate with existing " + strings.Join(scan.Governance.CIConfigs, "/") + " pipeline",
			"add new CI stage for the goal's deliverables",
			"keep deployment separate from existing CI",
		}
		reasoning = "CI configs detected (" + strings.Join(scan.Governance.CIConfigs, ", ") + "). The plan should confirm how new work integrates with the pipeline."
	}

	return discussQuestion{
		Category:  "deployment",
		Question:  "What deployment target should the plan optimize for?",
		Options:   options,
		Reasoning: reasoning,
		Source:    analyzeSource("deployment"),
	}
}

func buildAnalyzePerformanceQuestion(scan analyzeScanData) discussQuestion {
	options := []string{
		"optimize for correctness first",
		"balance correctness with performance",
		"optimize for performance from the start",
	}

	reasoning := "Performance priorities affect implementation choices (data structures, caching, async patterns)."

	if scan.FileCount > 500 {
		options = []string{
			"profile existing bottlenecks first",
			"focus on algorithmic efficiency for new code",
			"defer performance until after MVP",
		}
		reasoning = "Large codebase detected. Performance considerations should account for existing patterns and scale."
	}

	return discussQuestion{
		Category:  "performance",
		Question:  "What performance considerations should the plan account for?",
		Options:   options,
		Reasoning: reasoning,
		Source:    analyzeSource("performance"),
	}
}

// analyzeSource returns a source string for analyze-generated questions.
// Uses "analyze:" prefix to distinguish from "discuss:" prefix.
func analyzeSource(category string) string {
	return analyzeSourcePrefix + category
}
