package codex

import (
	"regexp"
	"strings"
)

var workerDiagnosticSecretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`sk-[A-Za-z0-9_-]+`),
	regexp.MustCompile(`github_pat_[A-Za-z0-9_]+`),
	regexp.MustCompile(`gh[pousr]_[A-Za-z0-9_]+`),
	regexp.MustCompile(`npm_[A-Za-z0-9_]+`),
	regexp.MustCompile(`(?i)\b[A-Za-z0-9._-]*secret[A-Za-z0-9._-]*\b`),
}

var workerDiagnosticCredentialAssignmentPattern = regexp.MustCompile(`(?i)\b(token|api[_-]?key|secret|password|authorization)\b\s*[:=]\s*["']?[^"'\s;]+`)

// SanitizeWorkerDiagnosticOutput redacts provider/credential material before
// worker subprocess diagnostics are exposed through errors, reports, or debug
// artifacts. It must only be applied after worker output has been parsed.
func SanitizeWorkerDiagnosticOutput(value string) string {
	value = strings.TrimSpace(stripANSIEscapeCodes(value))
	if value == "" {
		return ""
	}
	value = workerDiagnosticCredentialAssignmentPattern.ReplaceAllStringFunc(value, func(match string) string {
		idx := strings.IndexAny(match, ":=")
		if idx < 0 {
			return "[redacted]"
		}
		return strings.TrimSpace(match[:idx]) + match[idx:idx+1] + "[redacted]"
	})
	for _, pattern := range workerDiagnosticSecretPatterns {
		value = pattern.ReplaceAllString(value, "[redacted]")
	}
	return value
}

func sanitizeWorkerDiagnosticOutput(value string) string {
	return SanitizeWorkerDiagnosticOutput(value)
}
