package cmd

import "strings"

// isTestArtifact checks if a pheromone signal matches test artifact patterns.
// This is used by cmd/maintenance.go for data-clean.
func isTestArtifact(signal map[string]interface{}) bool {
	id, _ := signal["id"].(string)
	contentRaw := signal["content"]
	content := ""
	if contentMap, ok := contentRaw.(map[string]interface{}); ok {
		content, _ = contentMap["text"].(string)
	} else if contentStr, ok := contentRaw.(string); ok {
		content = contentStr
	}

	if strings.HasPrefix(id, "test_") || strings.HasPrefix(id, "demo_") {
		return true
	}

	lower := strings.ToLower(content)
	if strings.Contains(lower, "test signal") || strings.Contains(lower, "demo pattern") {
		return true
	}

	return false
}
