package colony

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

const maxSignalContentLength = 500

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

	// Rules 2-4: Reject integrity violations using the shared detector.
	if findings := DetectPromptIntegrityFindings(content); len(findings) > 0 {
		first := findings[0]
		switch first.Kind {
		case "xml_tag":
			return "", fmt.Errorf("content contains XML structural tags which are not allowed")
		case "prompt_injection":
			return "", fmt.Errorf("content contains prompt injection patterns which are not allowed")
		case "shell_injection":
			return "", fmt.Errorf("%s", first.Message)
		default:
			return "", fmt.Errorf("%s", first.Message)
		}
	}

	// Rule 5: Escape remaining angle brackets
	content = strings.ReplaceAll(content, "<", "&lt;")
	content = strings.ReplaceAll(content, ">", "&gt;")

	return content, nil
}

// recordTruncationEllipsis marks text NeutralizeForRecord had to cut short
// to fit within maxSignalContentLength.
const recordTruncationEllipsis = "…"

// NeutralizeForRecord returns text SanitizeSignalContent always accepts,
// while keeping as much of the original wording as the rejection rules
// allow. Before this, a failure whose own words SanitizeSignalContent
// rejected -- the everyday shape of real build/test output, which is full
// of backticks -- was replaced with a canned "could not be safely
// recorded" sentence that told the owner nothing about what actually
// failed (208-03-PLAN.md Task 3).
//
// It defuses each rejected construct in turn -- backticks, command
// substitution, a pipe-rm or semicolon-rm chain, angle brackets, a matched
// prompt-injection span, a matched secrets-path span -- then truncates to
// SanitizeSignalContent's own limit at a rune boundary (never splitting a
// multi-byte character) and appends an ellipsis when it had to cut.
//
// The result is not guaranteed non-empty: if nothing readable survives (an
// empty or whitespace-only input), it returns "" and the caller supplies
// the structured facts it already has (check kind, phase number, question,
// worker name) instead of an empty message.
func NeutralizeForRecord(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}

	// Backticks: a backtick pair is what both the shell-injection
	// "backticks" rule and command substitution key off. An apostrophe
	// keeps the sentence readable without ever forming a backtick pair
	// again -- this is also what makes the command-substitution and
	// pipe/semicolon-rm defusing below safe to run afterward: they never
	// have to reason about a backtick sitting inside their own match.
	content = strings.ReplaceAll(content, "`", "'")

	// Command substitution ($(...)): break the literal "$(" adjacency the
	// rule matches on, without discarding the words inside.
	if pattern := shellRulePatternByName("command substitution"); pattern != nil {
		content = pattern.ReplaceAllStringFunc(content, func(match string) string {
			if len(match) < 2 {
				return match
			}
			return "$ " + match[1:]
		})
	}

	// Pipe-rm / semicolon-rm: replace the chaining separator with a comma
	// so the destructive-chain shape this rule exists to catch no longer
	// matches, while "rm ..." itself stays exactly as written -- deleting
	// the word "rm" would make the record lie about what the text said.
	for _, name := range []string{"pipe rm", "semicolon rm"} {
		pattern := shellRulePatternByName(name)
		if pattern == nil {
			continue
		}
		content = pattern.ReplaceAllStringFunc(content, func(match string) string {
			if len(match) < 1 {
				return match
			}
			return "," + match[1:]
		})
	}

	// Angle brackets: the same escape SanitizeSignalContent itself applies
	// to accepted content, run here so the XML structural-tag rule never
	// gets a chance to fire on what follows.
	content = strings.ReplaceAll(content, "<", "&lt;")
	content = strings.ReplaceAll(content, ">", "&gt;")

	// Prompt-injection and secrets-path spans: each matched span becomes
	// one short bracketed note saying what was removed, rather than
	// discarding the rest of the sentence around it.
	for _, rule := range promptInjectionRules {
		content = rule.pattern.ReplaceAllString(content, "[instruction-like phrase removed]")
	}
	for _, rule := range secretsPathRules {
		content = rule.pattern.ReplaceAllString(content, "[file path removed]")
	}

	return truncateForRecord(content)
}

// shellRulePatternByName looks up a compiled shell-injection rule's pattern
// by its name field (shellInjectionRuleSpecs), rather than by table
// position, so NeutralizeForRecord keeps working if that table is ever
// reordered.
func shellRulePatternByName(name string) *regexp.Regexp {
	for _, rule := range shellInjectionRules {
		if rule.name == name {
			return rule.pattern
		}
	}
	return nil
}

// truncateForRecord cuts content to fit within maxSignalContentLength --
// measured the same way SanitizeSignalContent measures its own limit, on
// len() (bytes), not rune count -- at a rune boundary, using
// utf8.DecodeLastRuneInString so a multi-byte character is never split, and
// appends recordTruncationEllipsis when truncation happened.
func truncateForRecord(content string) string {
	if len(content) <= maxSignalContentLength {
		return content
	}

	budget := maxSignalContentLength - len(recordTruncationEllipsis)
	if budget < 0 {
		budget = 0
	}
	cut := content
	if len(cut) > budget {
		cut = cut[:budget]
	}
	for len(cut) > 0 {
		r, size := utf8.DecodeLastRuneInString(cut)
		if r == utf8.RuneError && size <= 1 {
			cut = cut[:len(cut)-1]
			continue
		}
		break
	}
	return cut + recordTruncationEllipsis
}
