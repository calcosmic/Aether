package colony

import (
	"fmt"
	"regexp"
	"strings"
)

const maxSignalContentLength = 500

// xmlTagPattern matches XML/HTML structural tags like <system>, </tag>, <br/>.
// It detects angle-bracket-enclosed tag names (letters, digits, hyphens, underscores).
var xmlTagPattern = regexp.MustCompile(`<[a-zA-Z/][a-zA-Z0-9_-]*\s*/?>`)

// promptInjectionPatterns matches common prompt injection phrases (case-insensitive).
var promptInjectionPatterns = []string{
	`(?i)ignore\s+previous\s+instructions`,
	`(?i)ignore\s+all\s+previous`,
	`(?i)disregard\s+(all\s+)?(rules|prior|previous|instructions)`,
	`(?i)you\s+are\s+now`,
	`(?i)new\s+instructions\s*:`,
}

// shellInjectionPatterns matches shell injection constructs.
var shellInjectionPatterns = []struct {
	name    string
	pattern *regexp.Regexp
}{
	{"command substitution", regexp.MustCompile(`\$\([^)]*\)`)},
	{"backticks", regexp.MustCompile("`[^`]*`")},
	{"pipe rm", regexp.MustCompile(`\|\s*rm\b`)},
	{"semicolon rm", regexp.MustCompile(`;\s*rm\b`)},
}

// SanitizeSignalContent validates and sanitizes pheromone signal content.
//
// Rules applied in order:
//  1. Trim whitespace
//  2. Check max length (500 characters)
//  3. Reject XML structural tags
//  4. Reject prompt injection patterns
//  5. Reject shell injection patterns
//  6. Escape remaining angle brackets
//
// Returns the sanitized content and an error if the content was rejected.
func SanitizeSignalContent(content string) (string, error) {
	content = strings.TrimSpace(content)

	// Rule 1: Max length check
	if len(content) > maxSignalContentLength {
		return "", fmt.Errorf("content exceeds maximum length of %d characters (%d)", maxSignalContentLength, len(content))
	}

	// Rule 2: Reject XML structural tags
	if xmlTagPattern.MatchString(content) {
		return "", fmt.Errorf("content contains XML structural tags which are not allowed")
	}

	// Rule 3: Reject prompt injection patterns
	for _, pattern := range promptInjectionPatterns {
		if matched, _ := regexp.MatchString(pattern, content); matched {
			return "", fmt.Errorf("content contains prompt injection patterns which are not allowed")
		}
	}

	// Rule 4: Reject shell injection patterns
	for _, shell := range shellInjectionPatterns {
		if shell.pattern.MatchString(content) {
			return "", fmt.Errorf("content contains shell injection patterns (%s) which are not allowed", shell.name)
		}
	}

	// Rule 5: Escape remaining angle brackets
	content = strings.ReplaceAll(content, "<", "&lt;")
	content = strings.ReplaceAll(content, ">", "&gt;")

	return content, nil
}
