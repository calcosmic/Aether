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
