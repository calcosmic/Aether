package cmd

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// colonyPrimeTemplates holds all loaded section templates from the prompts file.
type colonyPrimeTemplates struct {
	Version          string                     `yaml:"colony_prime_version"`
	SectionTemplates map[string]sectionTemplate `yaml:"section_templates"`
}

// sectionTemplate holds the format strings for a single prompt section.
type sectionTemplate struct {
	Header                string `yaml:"header,omitempty"`
	GoalFormat            string `yaml:"goal_format,omitempty"`
	StateFormat           string `yaml:"state_format,omitempty"`
	PhaseFormat           string `yaml:"phase_format,omitempty"`
	PhaseNameFormat       string `yaml:"phase_name_format,omitempty"`
	TaskFormat            string `yaml:"task_format,omitempty"`
	TasksHeader           string `yaml:"tasks_header,omitempty"`
	ParallelModeFormat    string `yaml:"parallel_mode_format,omitempty"`
	LightText             string `yaml:"light_text,omitempty"`
	StandardText          string `yaml:"standard_text,omitempty"`
	HeavyText             string `yaml:"heavy_text,omitempty"`
	DefaultText           string `yaml:"default_text,omitempty"`
	LifecycleContext      string `yaml:"lifecycle_context,omitempty"`
	SignalFormat          string `yaml:"signal_format,omitempty"`
	InstinctFormat        string `yaml:"instinct_format,omitempty"`
	DecisionFormat        string `yaml:"decision_format,omitempty"`
	PhaseHeaderFormat     string `yaml:"phase_header_format,omitempty"`
	LearningFormat        string `yaml:"learning_format,omitempty"`
	WorkerHeaderFormat    string `yaml:"worker_header_format,omitempty"`
	StatusFormat          string `yaml:"status_format,omitempty"`
	SummaryFormat         string `yaml:"summary_format,omitempty"`
	ListFormat            string `yaml:"list_format,omitempty"`
	EntryFormat           string `yaml:"entry_format,omitempty"`
	DomainFormat          string `yaml:"domain_format,omitempty"`
	DomainCountOnlyFormat string `yaml:"domain_count_only_format,omitempty"`
	BlockerFormat         string `yaml:"blocker_format,omitempty"`
	ScanTimestampFormat   string `yaml:"scan_timestamp_format,omitempty"`
	IssueFormat           string `yaml:"issue_format,omitempty"`
	IssueFileFormat       string `yaml:"issue_file_format,omitempty"`
}

// colonyPrimeTemplatesPathOverride allows tests to point at a temporary file.
// When empty, the default colony/prompts/colony-prime.md is used.
var colonyPrimeTemplatesPathOverride string

var (
	loadedColonyPrimeTemplates     *colonyPrimeTemplates
	loadedColonyPrimeTemplatesOnce sync.Once
)

// loadColonyPrimeTemplates loads the colony-prime prompt templates from YAML.
// It caches the result via sync.Once and falls back to nil on any error.
func loadColonyPrimeTemplates() *colonyPrimeTemplates {
	loadedColonyPrimeTemplatesOnce.Do(func() {
		path := colonyPrimeTemplatesPathOverride
		if path == "" {
			path = "colony/prompts/colony-prime.md"
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return
		}
		content := string(data)
		content = strings.TrimSpace(content)
		// Strip YAML frontmatter delimiters if present.
		if strings.HasPrefix(content, "---") {
			content = strings.TrimPrefix(content, "---")
			content = strings.TrimSpace(content)
			if idx := strings.Index(content, "---"); idx >= 0 {
				content = strings.TrimSpace(content[:idx])
			}
		}
		var templates colonyPrimeTemplates
		if err := yaml.Unmarshal([]byte(content), &templates); err != nil {
			return
		}
		loadedColonyPrimeTemplates = &templates
	})
	return loadedColonyPrimeTemplates
}

// getSectionTemplate returns the template for a named section, or nil if not loaded.
func getSectionTemplate(name string) *sectionTemplate {
	t := loadColonyPrimeTemplates()
	if t == nil || t.SectionTemplates == nil {
		return nil
	}
	if sec, ok := t.SectionTemplates[name]; ok {
		return &sec
	}
	return nil
}

// resetColonyPrimeTemplatesCache resets the sync.Once cache so tests can reload.
// This is test-only and must be called with care in production.
func resetColonyPrimeTemplatesCache() {
	loadedColonyPrimeTemplates = nil
	loadedColonyPrimeTemplatesOnce = sync.Once{}
}

// writeSection writes the header to b using the template if available, otherwise fallback.
func writeSectionHeader(b *strings.Builder, name string, fallback string) {
	if t := getSectionTemplate(name); t != nil && t.Header != "" {
		b.WriteString(t.Header)
	} else {
		b.WriteString(fallback)
	}
}

// charterFallbackHeading is the charter section heading used when no colony
// template overrides it. The single literal both writeSectionHeader (via
// buildColonyPrimeOutput) and charterSectionHeading below resolve against,
// so the producer and the checklist can never drift apart on the fallback
// text itself (WR-06).
const charterFallbackHeading = "## Charter -- Binding Rules"

// charterSectionHeading returns the exact heading line the charter section
// was written with -- a colony's SectionTemplates["charter"].Header override
// if one is configured, otherwise the fallback. Shared by the producer
// (buildColonyPrimeOutput, via writeSectionHeader) and the checklist
// (renderBriefChecklist) so charter-presence detection can't diverge from
// what was actually written: a hardcoded heading string breaks the moment a
// colony configures a custom charter header (WR-06).
func charterSectionHeading() string {
	if t := getSectionTemplate("charter"); t != nil && t.Header != "" {
		return strings.TrimRight(t.Header, "\n")
	}
	return charterFallbackHeading
}

// sectionString returns the template string for a named section field, or fallback.
func sectionString(name string, field func(*sectionTemplate) string, fallback string) string {
	if t := getSectionTemplate(name); t != nil {
		if v := field(t); v != "" {
			return v
		}
	}
	return fallback
}

// fmtOrFallback formats using the template format string if available, else fallback format.
func fmtOrFallback(name string, field func(*sectionTemplate) string, fallbackFormat string, args ...interface{}) string {
	format := sectionString(name, field, fallbackFormat)
	return fmt.Sprintf(format, args...)
}
