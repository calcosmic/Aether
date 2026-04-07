package cmd

import "strings"

// User-created signals (source="user" or source="cli") are never flagged as test
// artifacts, regardless of content. This prevents false positives where a user's
// pheromone containing words like "test" or "demo" would be deleted by data-clean.
// Only system-generated sources ("auto", "promotion", etc.) are checked for
// test artifact patterns.
func isTestArtifact(signal map[string]interface{}) bool {
	source, _ := signal["source"].(string)
	if source == "user" || source == "cli" {
		return false
	}

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
