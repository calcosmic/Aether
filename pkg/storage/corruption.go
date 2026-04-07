package storage

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

// DetectCorruption inspects a ColonyState for signs of corruption.
// It checks the Events field for strings that appear to be jq expressions
// rather than human-readable event descriptions.
// Returns an error describing the first corrupted entry found, or nil if clean.
func DetectCorruption(state *colony.ColonyState) error {
	if state == nil || len(state.Events) == 0 {
		return nil
	}

	for i, event := range state.Events {
		if looksLikeJQExpression(event) {
			return fmt.Errorf("corruption detected in events[%d]: value appears to be a jq expression: %q", i, event)
		}
	}

	return nil
}

// looksLikeJQExpression checks whether a string resembles a jq expression
// rather than a natural language event description.
// It detects patterns like:
//   - ".field = value" (assignment)
//   - ".field |= expr" (update)
//   - "| select(" or "| map(" (jq pipe operators)
func looksLikeJQExpression(s string) bool {
	// Pattern 1: Assignment: .field = value or .field.sub = value
	assignmentRe := regexp.MustCompile(`^\.([\w.\[\]]+)\s*[|=]`)
	if assignmentRe.MatchString(s) {
		return true
	}

	// Pattern 2: Suspicious jq operators that shouldn't appear in event descriptions
	jqOperators := []string{
		"|=",
		"| select(",
		"| map(",
	}
	for _, op := range jqOperators {
		if strings.Contains(s, op) {
			return true
		}
	}

	return false
}
