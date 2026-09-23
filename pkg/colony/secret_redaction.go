package colony

import "regexp"

// secretRedactionLabel replaces every matched secret-value span. It never
// varies by rule -- the report bundle already says, once, that redaction
// happened; naming which specific kind of secret matched inside the bundle
// itself would just be more sensitive detail sitting beside the thing being
// hidden.
const secretRedactionLabel = "[secret redacted]"

// secretValueRuleSpecs is CR-03's own list (208-REVIEW.md): common shapes a
// real secret VALUE takes when it rides along inside captured build/test
// output that a failing command genuinely printed to stdout/stderr -- an
// API key, a bearer token, a `KEY=value` env assignment, a credentialed
// URL. This is deliberately separate from promptInjectionRuleSpecs /
// shellInjectionRuleSpecs / secretsPathRuleSpecs above: those catch a PATH
// that names a secrets FILE, or an injection/shell-metacharacter SHAPE --
// none of them match a literal secret VALUE. `aether report`'s entire
// purpose is packaging local failure/refusal data to hand to a third
// party, so this is the one place that channel's own text is scanned for a
// value, not just a path or a phrase, before it ever reaches the bundle.
var secretValueRuleSpecs = []string{
	// AWS access key id.
	`\bAKIA[0-9A-Z]{16}\b`,
	// GitHub personal-access / app tokens (ghp_, gho_, ghu_, ghs_, ghr_).
	`\bgh[oprsu]_[A-Za-z0-9]{20,}\b`,
	// Common "sk-..." secret-key shape (OpenAI, Stripe, and lookalikes).
	`\bsk-[A-Za-z0-9_-]{16,}\b`,
	// A bearer token, wherever a header or log line names one.
	`(?i)\bBearer\s+[A-Za-z0-9\-_.=]{8,}`,
	// An Authorization or X-Api-Key header line, value included.
	`(?i)\b(?:Authorization|X-Api-Key)\s*:\s*\S+`,
	// A `SOMETHING_KEY=value` / `SOMETHING_SECRET=value` /
	// `SOMETHING_TOKEN=value` / `SOMETHING_PASSWORD=value` env assignment.
	`(?i)\b[A-Z][A-Z0-9_]*(?:KEY|SECRET|TOKEN|PASSWORD)\b\s*[:=]\s*\S+`,
	// user:password@ embedded in a URL.
	`\b[a-zA-Z][a-zA-Z0-9+.-]*://[^\s/:@]+:[^\s/@]+@\S+`,
	// A generic long base64-ish run -- the catch-all for a token shape
	// none of the named rules above matched. Deliberately broad: for a
	// channel whose entire purpose is leaving the project, over-redacting
	// an incidental long hash is a far safer failure than under-redacting
	// a real key.
	`\b[A-Za-z0-9+/]{32,}={0,2}\b`,
}

var secretValueRules = func() []*regexp.Regexp {
	rules := make([]*regexp.Regexp, 0, len(secretValueRuleSpecs))
	for _, spec := range secretValueRuleSpecs {
		rules = append(rules, regexp.MustCompile(spec))
	}
	return rules
}()

// RedactSecretValues replaces every span in content that matches one of
// secretValueRuleSpecs with secretRedactionLabel, returning the result.
// Unlike SanitizeSignalContent/NeutralizeForRecord, this never rejects or
// truncates content and is not itself a trust gate -- it exists
// specifically for `aether report`'s bundle (cmd/report_cmd.go), the one
// place local failure text is packaged for a third party, so a credential
// a failing build or test genuinely printed to stdout/stderr never rides
// along unredacted (CR-03, 208-REVIEW.md).
func RedactSecretValues(content string) string {
	if content == "" {
		return content
	}
	for _, rule := range secretValueRules {
		content = rule.ReplaceAllString(content, secretRedactionLabel)
	}
	return content
}
