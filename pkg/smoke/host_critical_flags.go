package smoke

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// HostCriticalFlags maps host command names to the list of flags documented
// in their corresponding YAML source files.
var HostCriticalFlags = loadHostCriticalFlags()

// HostCriticalCombinations maps host command names to realistic flag
// combination slices sourced from TCV-01 through TCV-05 requirements.
var HostCriticalCombinations = map[string][][]string{
	"ant-plan": {
		{"--depth", "balanced", "--planning-depth", "standard"},
	},
	"ant-continue": {
		{"--verification-depth", "heavy"},
	},
	"ant-watch": {
		{"--no-dashboard"},
	},
	"ant-swarm": {
		{"--no-dashboard"},
	},
}

// loadHostCriticalFlags reads the 6 host command YAML files and extracts
// documented flags. Returns a map of command name -> sorted flag list.
func loadHostCriticalFlags() map[string][]string {
	result := make(map[string][]string)

	hostFiles := map[string]string{
		"ant-plan":     "plan.yaml",
		"ant-build":    "build.yaml",
		"ant-continue": "continue.yaml",
		"ant-oracle":   "oracle.yaml",
		"ant-watch":    "watch.yaml",
		"ant-swarm":    "swarm.yaml",
	}

	for cmdName, filename := range hostFiles {
		path := filepath.Join(".aether", "commands", filename)
		flags, err := extractFlagsFromYAML(path)
		if err != nil {
			// If file doesn't exist, record empty list
			result[cmdName] = []string{}
			continue
		}
		result[cmdName] = flags
	}

	return result
}

// extractFlagsFromYAML reads a single YAML file and extracts flag mentions.
func extractFlagsFromYAML(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var doc map[string]interface{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	var textFields []string

	if runtime, ok := doc["runtime"].(map[string]interface{}); ok {
		if cmd, ok := runtime["command"].(string); ok {
			textFields = append(textFields, cmd)
		}
	}

	if additions, ok := doc["wrapper_additions"].(map[string]interface{}); ok {
		for _, v := range additions {
			if s, ok := v.(string); ok {
				textFields = append(textFields, s)
			}
		}
	}

	if guardrails, ok := doc["guardrails"].([]interface{}); ok {
		for _, v := range guardrails {
			if s, ok := v.(string); ok {
				textFields = append(textFields, s)
			}
		}
	}

	if intent, ok := doc["intent_refinement"].([]interface{}); ok {
		for _, v := range intent {
			if s, ok := v.(string); ok {
				textFields = append(textFields, s)
			}
		}
	}

	if followUp, ok := doc["follow_up"].(map[string]interface{}); ok {
		for _, v := range followUp {
			if s, ok := v.(string); ok {
				textFields = append(textFields, s)
			}
		}
	}

	if codex, ok := doc["codex_orchestration"].(map[string]interface{}); ok {
		for _, v := range codex {
			if s, ok := v.(string); ok {
				textFields = append(textFields, s)
			}
		}
	}

	flags := extractFlags(textFields)
	return flags, nil
}

// tsHostFlagPattern matches --flag-name style flags in TypeScript source.
var tsHostFlagPattern = regexp.MustCompile(`--[a-zA-Z0-9-]+`)

// CrossReferenceWithTSHost reads the TS host test files and returns any
// YAML-documented host flags that do NOT appear in the TS host tests.
// These are "documented but untested" flags and produce warnings.
func CrossReferenceWithTSHost(yamlFlags map[string][]string) []string {
	var untested []string

	// Read TS host test files
	tsFlags := make(map[string]bool)

	tsTestFiles := []string{
		".aether/ts-host/test/host-flags.test.ts",
		".aether/ts-host/test/host-integration.test.ts",
	}

	for _, path := range tsTestFiles {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		matches := tsHostFlagPattern.FindAllString(string(data), -1)
		for _, m := range matches {
			tsFlags[m] = true
		}
	}

	// Also scan the TS host source file for flag references
	tsSourceFiles := []string{
		".aether/ts-host/src/host.ts",
	}

	for _, path := range tsSourceFiles {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		matches := tsHostFlagPattern.FindAllString(string(data), -1)
		for _, m := range matches {
			tsFlags[m] = true
		}
	}

	// Check each YAML-documented host flag against TS test coverage
	for cmdName, flags := range yamlFlags {
		for _, flag := range flags {
			if !tsFlags[flag] {
				untested = append(untested, fmt.Sprintf("%s:%s", cmdName, flag))
			}
		}
	}

	sort.Strings(untested)
	return untested
}

// IsHostCritical returns true if the given command name is a host-critical
// command that should be tested with blocking severity.
func IsHostCritical(name string) bool {
	_, ok := HostCriticalFlags[name]
	return ok
}

// GetHostCommandGoName maps ant-* names to their Go subcommand names.
func GetHostCommandGoName(antName string) string {
	// Strip ant- prefix
	return strings.TrimPrefix(antName, "ant-")
}
