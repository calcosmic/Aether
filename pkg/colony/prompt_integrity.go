package colony

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// RepoPlaceholder is the token that replaces a source repository's name when a
// colony-specific learning is generalised for the cross-colony hive.
//
// It lives here, beside the sanitizer, because it MUST survive
// SanitizeSignalContent. It previously did not: the placeholder was "<repo>",
// which xmlTagPattern classifies as an XML structural tag, so hive promotion
// rejected the very placeholder hive promotion had just inserted. Since a
// learning drawn from a repository almost always names that repository, nearly
// every promotion was silently discarded and the hive stayed empty while the
// read path injected an empty section into every worker dispatch.
//
// TestRepoPlaceholderSurvivesSanitizer locks the invariant. Any future change
// to this value or to the rule set must keep that test green.
const RepoPlaceholder = "[repo]"

type PromptTrustClass string

const (
	PromptTrustAuthorized PromptTrustClass = "authorized"
	PromptTrustTrusted    PromptTrustClass = "trusted"
	PromptTrustUnknown    PromptTrustClass = "unknown"
	PromptTrustSuspicious PromptTrustClass = "suspicious"
)

type PromptIntegrityAction string

const (
	PromptIntegrityActionAllow PromptIntegrityAction = "allow"
	PromptIntegrityActionWarn  PromptIntegrityAction = "warn"
	PromptIntegrityActionBlock PromptIntegrityAction = "block"
)

type PromptIntegrityFinding struct {
	Kind     string `json:"kind"`
	Message  string `json:"message"`
	Evidence string `json:"evidence,omitempty"`
}

type PromptIntegrityAssessment struct {
	BaseTrustClass PromptTrustClass         `json:"base_trust_class"`
	TrustClass     PromptTrustClass         `json:"trust_class"`
	Action         PromptIntegrityAction    `json:"action"`
	Findings       []PromptIntegrityFinding `json:"findings,omitempty"`
}

type PromptIntegrityRecord struct {
	Name           string                   `json:"name,omitempty"`
	Title          string                   `json:"title,omitempty"`
	Source         string                   `json:"source,omitempty"`
	BaseTrustClass PromptTrustClass         `json:"base_trust_class"`
	TrustClass     PromptTrustClass         `json:"trust_class"`
	Action         PromptIntegrityAction    `json:"action"`
	Blocked        bool                     `json:"blocked,omitempty"`
	Findings       []PromptIntegrityFinding `json:"findings,omitempty"`
}

var promptInjectionRuleSpecs = []struct {
	kind    string
	message string
	pattern string
}{
	{"prompt_injection", "content contains prompt injection patterns which are not allowed", `(?i)ignore\s+previous\s+instructions`},
	{"prompt_injection", "content contains prompt injection patterns which are not allowed", `(?i)ignore\s+all\s+previous`},
	{"prompt_injection", "content contains prompt injection patterns which are not allowed", `(?i)disregard\s+(all\s+)?(rules|prior|previous|instructions)`},
	{"prompt_injection", "content contains prompt injection patterns which are not allowed", `(?i)you\s+are\s+now`},
	{"prompt_injection", "content contains prompt injection patterns which are not allowed", `(?i)new\s+instructions\s*:`},
}

var shellInjectionRuleSpecs = []struct {
	kind    string
	message string
	name    string
	pattern string
}{
	{"shell_injection", "content contains shell injection patterns (command substitution) which are not allowed", "command substitution", `\$\([^)]*\)`},
	{"shell_injection", "content contains shell injection patterns (backticks) which are not allowed", "backticks", "`[^`]*`"},
	{"shell_injection", "content contains shell injection patterns (pipe rm) which are not allowed", "pipe rm", `\|\s*rm\b`},
	{"shell_injection", "content contains shell injection patterns (semicolon rm) which are not allowed", "semicolon rm", `;\s*rm\b`},
}

// secretsPathRuleSpecs catches content that names a secrets or credentials
// file path -- e.g. a reviewer-reported "lesson" that is really a shell
// command copying `.env.local` out of the project (the 2026-09-14 field
// report's sixth finding, verbatim:
// "cd .../dashboard && cp .../dashboard/.env.local . 2>/dev/null; npm install ...").
// That command has neither a pipe/semicolon `rm`, a backtick, nor a `$()`
// substitution, so none of shellInjectionRuleSpecs matched it -- it reached
// QUEEN.md's "Learned habit" surface unfiltered. Living here, beside the
// other rule lists DetectPromptIntegrityFindings already walks, means every
// caller of the shared detector (pheromone signals via SanitizeSignalContent,
// and seal-promoted lessons via sanitizeQueenPromotedLesson in cmd/queen.go)
// gains the protection at once, rather than each caller hand-rolling its own
// secrets-path list.
//
// The rule is deliberately PATH-shaped: the secrets file name must sit inside
// a path (`../dashboard/.env.local`, `~/.ssh/id_rsa`) or be the direct target
// of a file-handling shell verb (`cp .env.local .`). A bare mention of ".env
// files" in ordinary guidance is not a path, and refusing it broke
// suggest-analyze's own built-in "never commit secrets or .env files to
// version control" steering note (Phase 205 wave-1 post-merge gate; locked by
// TestPromptIntegritySecretsPathRuleIsPathShaped).
var secretsPathRuleSpecs = []struct {
	kind    string
	message string
	pattern string
}{
	{"secrets_path", "content references a secrets or credentials file path which is not allowed", `(?i)(?:/[^\s"']*?|\b(?:cp|cat|mv|scp|rsync|source|curl|wget|tee|less|more|head|tail|base64|xxd|type)\s+[^\s"']*?)(\.env(\.[a-zA-Z0-9_-]+)?|credentials\.json|secrets\.json|id_rsa|\.pem|\.netrc|\.npmrc)\b`},
}

type compiledPromptRule struct {
	kind    string
	message string
	pattern *regexp.Regexp
}

type compiledShellRule struct {
	kind    string
	message string
	name    string
	pattern *regexp.Regexp
}

var xmlTagPattern = regexp.MustCompile(`<[a-zA-Z/][a-zA-Z0-9_-]*\s*/?>`)

var promptInjectionRules = func() []compiledPromptRule {
	rules := make([]compiledPromptRule, 0, len(promptInjectionRuleSpecs))
	for _, spec := range promptInjectionRuleSpecs {
		rules = append(rules, compiledPromptRule{
			kind:    spec.kind,
			message: spec.message,
			pattern: regexp.MustCompile(spec.pattern),
		})
	}
	return rules
}()

var shellInjectionRules = func() []compiledShellRule {
	rules := make([]compiledShellRule, 0, len(shellInjectionRuleSpecs))
	for _, spec := range shellInjectionRuleSpecs {
		rules = append(rules, compiledShellRule{
			kind:    spec.kind,
			message: spec.message,
			name:    spec.name,
			pattern: regexp.MustCompile(spec.pattern),
		})
	}
	return rules
}()

var secretsPathRules = func() []compiledPromptRule {
	rules := make([]compiledPromptRule, 0, len(secretsPathRuleSpecs))
	for _, spec := range secretsPathRuleSpecs {
		rules = append(rules, compiledPromptRule{
			kind:    spec.kind,
			message: spec.message,
			pattern: regexp.MustCompile(spec.pattern),
		})
	}
	return rules
}()

func DetectPromptIntegrityFindings(content string) []PromptIntegrityFinding {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}

	findings := make([]PromptIntegrityFinding, 0, 4)
	if evidence := xmlTagPattern.FindString(content); evidence != "" {
		findings = append(findings, PromptIntegrityFinding{
			Kind:     "xml_tag",
			Message:  "content contains XML structural tags which are not allowed",
			Evidence: evidence,
		})
	}

	for _, rule := range promptInjectionRules {
		if evidence := rule.pattern.FindString(content); evidence != "" {
			findings = append(findings, PromptIntegrityFinding{
				Kind:     rule.kind,
				Message:  rule.message,
				Evidence: evidence,
			})
		}
	}

	for _, rule := range shellInjectionRules {
		if evidence := rule.pattern.FindString(content); evidence != "" {
			findings = append(findings, PromptIntegrityFinding{
				Kind:     rule.kind,
				Message:  rule.message,
				Evidence: evidence,
			})
		}
	}

	for _, rule := range secretsPathRules {
		if evidence := rule.pattern.FindString(content); evidence != "" {
			findings = append(findings, PromptIntegrityFinding{
				Kind:     rule.kind,
				Message:  rule.message,
				Evidence: evidence,
			})
		}
	}

	return findings
}

func ClassifyPromptSource(source string) PromptTrustClass {
	source = filepath.ToSlash(filepath.Clean(strings.TrimSpace(source)))
	switch {
	case source == "", source == ".", strings.HasPrefix(source, "inline:"):
		return PromptTrustUnknown
	case strings.HasSuffix(source, "COLONY_STATE.json"),
		strings.HasSuffix(source, "instincts.json"),
		strings.HasSuffix(source, "pheromones.json"),
		strings.HasSuffix(source, "pending-decisions.json"),
		strings.HasSuffix(source, "flags.json"),
		strings.HasSuffix(source, "rolling-summary.log"),
		strings.HasSuffix(source, filepath.ToSlash(filepath.Join("hive", "wisdom.json"))):
		return PromptTrustAuthorized
	case strings.HasSuffix(source, "QUEEN.md"):
		return PromptTrustTrusted
	default:
		return PromptTrustUnknown
	}
}

func AssessPromptSource(source, content string) PromptIntegrityAssessment {
	baseTrust := ClassifyPromptSource(source)
	findings := DetectPromptIntegrityFindings(content)
	if len(findings) == 0 {
		return PromptIntegrityAssessment{
			BaseTrustClass: baseTrust,
			TrustClass:     baseTrust,
			Action:         PromptIntegrityActionAllow,
		}
	}

	return PromptIntegrityAssessment{
		BaseTrustClass: baseTrust,
		TrustClass:     PromptTrustSuspicious,
		Action:         PromptIntegrityActionBlock,
		Findings:       findings,
	}
}

func (a PromptIntegrityAssessment) Record(name, title, source string) PromptIntegrityRecord {
	return PromptIntegrityRecord{
		Name:           name,
		Title:          title,
		Source:         filepath.ToSlash(strings.TrimSpace(source)),
		BaseTrustClass: a.BaseTrustClass,
		TrustClass:     a.TrustClass,
		Action:         a.Action,
		Blocked:        a.Action == PromptIntegrityActionBlock,
		Findings:       append([]PromptIntegrityFinding(nil), a.Findings...),
	}
}

func (a PromptIntegrityAssessment) Warning(name, source string) string {
	source = filepath.ToSlash(strings.TrimSpace(source))
	label := strings.TrimSpace(name)
	if label == "" {
		label = "prompt source"
	}
	if len(a.Findings) == 0 {
		return fmt.Sprintf("%s from %s requires review", label, emptyPromptSource(source))
	}
	return fmt.Sprintf("blocked suspicious %s from %s: %s", label, emptyPromptSource(source), a.Findings[0].Message)
}

func emptyPromptSource(source string) string {
	if strings.TrimSpace(source) == "" {
		return "unknown source"
	}
	return source
}
