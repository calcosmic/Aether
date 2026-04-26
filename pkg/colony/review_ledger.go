package colony

import (
	"fmt"
	"strconv"
	"strings"
)

// ReviewSeverity represents the severity level of a review finding.
type ReviewSeverity string

const (
	ReviewSeverityHigh   ReviewSeverity = "HIGH"
	ReviewSeverityMedium ReviewSeverity = "MEDIUM"
	ReviewSeverityLow    ReviewSeverity = "LOW"
	ReviewSeverityInfo   ReviewSeverity = "INFO"
)

// ValidReviewDomains maps domain names to their ID prefixes.
var ValidReviewDomains = map[string]string{
	"security":    "sec",
	"quality":     "qlt",
	"performance": "prf",
	"resilience":  "res",
	"testing":     "tst",
	"history":     "hst",
	"bugs":        "bug",
}

// DomainOrder defines the deterministic iteration order for review domains.
// Used by both CLI commands and colony-prime section assembly.
var DomainOrder = []string{"security", "quality", "performance", "resilience", "testing", "history", "bugs"}

// ReviewLedgerEntry represents a single finding in a domain review ledger.
type ReviewLedgerEntry struct {
	ID          string         `json:"id"`
	Phase       int            `json:"phase"`
	PhaseName   string         `json:"phase_name,omitempty"`
	Agent       string         `json:"agent"`
	AgentName   string         `json:"agent_name,omitempty"`
	GeneratedAt string         `json:"generated_at"`
	Status      string         `json:"status"`
	Severity    ReviewSeverity `json:"severity"`
	File        string         `json:"file,omitempty"`
	Line        int            `json:"line,omitempty"`
	Category    string         `json:"category,omitempty"`
	Description string         `json:"description"`
	Suggestion  string         `json:"suggestion,omitempty"`
	ResolvedAt  *string        `json:"resolved_at,omitempty"`
}

// ReviewLedgerSeverityCounts tracks the number of findings per severity level.
type ReviewLedgerSeverityCounts struct {
	High   int `json:"high"`
	Medium int `json:"medium"`
	Low    int `json:"low"`
	Info   int `json:"info"`
}

// ReviewLedgerSummary provides a computed summary of ledger entries.
type ReviewLedgerSummary struct {
	Total      int                      `json:"total"`
	Open       int                      `json:"open"`
	Resolved   int                      `json:"resolved"`
	BySeverity ReviewLedgerSeverityCounts `json:"by_severity"`
}

// ReviewLedgerFile represents a domain review ledger file on disk.
type ReviewLedgerFile struct {
	Entries []ReviewLedgerEntry `json:"entries"`
	Summary ReviewLedgerSummary `json:"summary"`
}

// ComputeSummary tallies open/resolved counts and per-severity breakdowns
// from the given entries.
func ComputeSummary(entries []ReviewLedgerEntry) ReviewLedgerSummary {
	var s ReviewLedgerSummary
	s.Total = len(entries)
	for _, e := range entries {
		switch e.Status {
		case "open":
			s.Open++
		case "resolved":
			s.Resolved++
		}
		switch e.Severity {
		case ReviewSeverityHigh:
			s.BySeverity.High++
		case ReviewSeverityMedium:
			s.BySeverity.Medium++
		case ReviewSeverityLow:
			s.BySeverity.Low++
		case ReviewSeverityInfo:
			s.BySeverity.Info++
		}
	}
	return s
}

// FormatEntryID produces a deterministic entry ID in the form
// {prefix}-{phase}-{index}, where index is zero-padded to 3 digits.
func FormatEntryID(domainPrefix string, phase int, index int) string {
	return fmt.Sprintf("%s-%d-%03d", domainPrefix, phase, index)
}

// NextEntryIndex scans entries for IDs matching the given domain prefix
// and phase, returning the next available index (1 if no matches).
func NextEntryIndex(entries []ReviewLedgerEntry, domainPrefix string, phase int) int {
	maxIdx := 0
	prefix := fmt.Sprintf("%s-%d-", domainPrefix, phase)
	for _, e := range entries {
		if strings.HasPrefix(e.ID, prefix) {
			// Extract the numeric suffix after the prefix.
			suffix := e.ID[len(prefix):]
			if idx, err := strconv.Atoi(suffix); err == nil && idx > maxIdx {
				maxIdx = idx
			}
		}
	}
	return maxIdx + 1
}
