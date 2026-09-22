package cmd

import (
	"strings"
)

// friendlyError is the rendering shape renderVisualError's fallback path
// still expects: a plain-language explanation plus an ordered list of next
// steps. It is kept as a thin, unexported view over a refusalRegistry row
// (cmd/refusal_register.go) -- the ONE refusal table in the program -- not a
// second, parallel table. errorPatternMap used to live here, hand-rolled;
// TestFriendlyErrorsReadTheOneRefusalTable fails naming this file if a
// second one is ever planted anywhere in cmd/*.go.
type friendlyError struct {
	Pattern     string
	Explanation string
	NextSteps   []string
}

// friendlyErrorForPattern looks up the refusalRegistry row whose Pattern is
// a case-insensitive substring of message -- ties broken by the longest
// (most specific) pattern, via refusalRowForPattern -- and adapts it into
// the friendlyError shape. Matching is case-insensitive.
func friendlyErrorForPattern(message string) (friendlyError, bool) {
	row, ok := refusalRowForPattern(message)
	if !ok {
		return friendlyError{}, false
	}
	return friendlyError{
		Pattern:     row.Pattern,
		Explanation: row.Why,
		NextSteps:   refusalRowNextSteps(row),
	}, true
}

// refusalRowNextSteps reconstructs the ordered next-steps list a row's
// NextCommand plus ExtraSteps represents, in the shape renderFriendlyError
// already renders: the command first, any extra steps after it.
func refusalRowNextSteps(row refusalRow) []string {
	steps := make([]string, 0, 1+len(row.ExtraSteps))
	if command := strings.TrimSpace(row.NextCommand); command != "" {
		steps = append(steps, "Run `"+command+"`.")
	}
	steps = append(steps, row.ExtraSteps...)
	return steps
}

// renderFriendlyError produces a visual error display with a plain-language
// explanation and actionable next steps.
func renderFriendlyError(entry friendlyError, rawMessage string) string {
	var b strings.Builder
	b.WriteString(renderBanner("❌", "Error"))
	b.WriteString(visualDividerStr())
	b.WriteString(entry.Explanation)
	b.WriteString("\n\n")
	b.WriteString("Next steps:\n")
	for _, step := range entry.NextSteps {
		b.WriteString("  - ")
		b.WriteString(step)
		b.WriteString("\n")
	}
	return b.String()
}
