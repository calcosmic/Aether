package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

// Package-level constants for review ledger domain management.

var validDomains = map[string]bool{
	"security": true, "quality": true, "performance": true,
	"resilience": true, "testing": true, "history": true, "bugs": true,
}

var domainPrefixes = map[string]string{
	"security": "sec", "quality": "qlt", "performance": "prf",
	"resilience": "res", "testing": "tst", "history": "hst", "bugs": "bug",
}

var agentAllowedDomains = map[string][]string{
	"gatekeeper":    {"security"},
	"auditor":       {"quality", "security", "performance"},
	"chaos":         {"resilience"},
	"watcher":       {"testing", "quality"},
	"archaeologist": {"history"},
	"measurer":      {"performance"},
	"tracker":       {"bugs"},
}

const maxFindingsPerWrite = 50

// --- review-ledger-write ---

var reviewLedgerWriteCmd = &cobra.Command{
	Use:   "review-ledger-write",
	Short: "Write findings to a domain review ledger",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		domain := mustGetString(cmd, "domain")
		if domain == "" {
			return nil
		}

		phase := mustGetInt(cmd, "phase")
		if phase <= 0 {
			outputError(1, "flag --phase is required and must be > 0", nil)
			return nil
		}

		findingsJSON := mustGetString(cmd, "findings")
		if findingsJSON == "" {
			return nil
		}

		agent := mustGetStringCompatOptional(cmd, "agent")
		agentName := mustGetStringCompatOptional(cmd, "agent-name")
		phaseName := mustGetStringCompatOptional(cmd, "phase-name")

		// Validate domain
		if !validDomains[domain] {
			outputError(1, fmt.Sprintf("invalid domain %q: must be one of %s", domain, strings.Join(colony.DomainOrder, ", ")), nil)
			return nil
		}

		// Validate agent-domain mapping (skip if agent is empty)
		if agent != "" {
			allowed, exists := agentAllowedDomains[agent]
			if !exists {
				outputError(1, fmt.Sprintf("unknown agent %q: allowed agents are %s", agent, agentList()), nil)
				return nil
			}
			allowedSet := make(map[string]bool, len(allowed))
			for _, d := range allowed {
				allowedSet[d] = true
			}
			if !allowedSet[domain] {
				outputError(1, fmt.Sprintf("agent %q is not allowed to write to domain %q (allowed: %s)", agent, domain, strings.Join(allowed, ", ")), nil)
				return nil
			}
		}

		// Parse findings JSON
		var findings []struct {
			Severity    string `json:"severity"`
			File        string `json:"file"`
			Line        int    `json:"line"`
			Category    string `json:"category"`
			Description string `json:"description"`
			Suggestion  string `json:"suggestion"`
		}
		if err := json.Unmarshal([]byte(findingsJSON), &findings); err != nil {
			outputError(1, "invalid --findings JSON", nil)
			return nil
		}

		if len(findings) > maxFindingsPerWrite {
			outputError(1, fmt.Sprintf("too many findings: %d exceeds maximum of %d per call", len(findings), maxFindingsPerWrite), nil)
			return nil
		}

		// Load existing ledger
		prefix := domainPrefixes[domain]
		ledgerPath := fmt.Sprintf("reviews/%s/ledger.json", domain)

		var lf colony.ReviewLedgerFile
		if err := store.LoadJSON(ledgerPath, &lf); err != nil {
			lf = colony.ReviewLedgerFile{Entries: []colony.ReviewLedgerEntry{}}
		}
		if lf.Entries == nil {
			lf.Entries = []colony.ReviewLedgerEntry{}
		}

		// Build and append entries
		now := time.Now().UTC().Format(time.RFC3339)
		for _, f := range findings {
			idx := colony.NextEntryIndex(lf.Entries, prefix, phase)
			id := colony.FormatEntryID(prefix, phase, idx)

			entry := colony.ReviewLedgerEntry{
				ID:          id,
				Phase:       phase,
				PhaseName:   phaseName,
				Agent:       agent,
				AgentName:   agentName,
				GeneratedAt: now,
				Status:      "open",
				Severity:    colony.ReviewSeverity(strings.ToUpper(f.Severity)),
				File:        f.File,
				Line:        f.Line,
				Category:    f.Category,
				Description: f.Description,
				Suggestion:  f.Suggestion,
			}
			lf.Entries = append(lf.Entries, entry)
		}

		// Recompute summary and save
		lf.Summary = colony.ComputeSummary(lf.Entries)
		if err := store.SaveJSON(ledgerPath, lf); err != nil {
			outputError(2, fmt.Sprintf("failed to save ledger: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"written": true,
			"domain":  domain,
			"total":   len(lf.Entries),
			"summary": lf.Summary,
		})
		return nil
	},
}

// --- review-ledger-read ---

var reviewLedgerReadCmd = &cobra.Command{
	Use:   "review-ledger-read",
	Short: "Read entries from a domain review ledger",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		domain := mustGetString(cmd, "domain")
		if domain == "" {
			return nil
		}

		if !validDomains[domain] {
			outputError(1, fmt.Sprintf("invalid domain %q: must be one of %s", domain, strings.Join(colony.DomainOrder, ", ")), nil)
			return nil
		}

		ledgerPath := fmt.Sprintf("reviews/%s/ledger.json", domain)
		var lf colony.ReviewLedgerFile
		if err := store.LoadJSON(ledgerPath, &lf); err != nil {
			// No ledger file yet -- return empty
			outputOK(map[string]interface{}{
				"entries": []colony.ReviewLedgerEntry{},
				"summary": colony.ReviewLedgerSummary{},
			})
			return nil
		}

		entries := lf.Entries

		// Filter by phase if explicitly set
		if cmd.Flags().Changed("phase") {
			phase, _ := cmd.Flags().GetInt("phase")
			var filtered []colony.ReviewLedgerEntry
			for _, e := range entries {
				if e.Phase == phase {
					filtered = append(filtered, e)
				}
			}
			entries = filtered
		}

		// Filter by status if provided
		status := mustGetStringCompatOptional(cmd, "status")
		if status != "" {
			var filtered []colony.ReviewLedgerEntry
			for _, e := range entries {
				if e.Status == status {
					filtered = append(filtered, e)
				}
			}
			entries = filtered
		}

		outputOK(map[string]interface{}{
			"entries": entries,
			"summary": lf.Summary,
		})
		return nil
	},
}

// --- review-ledger-summary ---

var reviewLedgerSummaryCmd = &cobra.Command{
	Use:   "review-ledger-summary",
	Short: "Summarize all domain review ledgers",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		var domains []map[string]interface{}
		for _, d := range colony.DomainOrder {
			ledgerPath := fmt.Sprintf("reviews/%s/ledger.json", d)
			var lf colony.ReviewLedgerFile
			if err := store.LoadJSON(ledgerPath, &lf); err != nil {
				continue // No ledger for this domain -- skip
			}

			domains = append(domains, map[string]interface{}{
				"domain":     d,
				"total":      lf.Summary.Total,
				"open":       lf.Summary.Open,
				"resolved":   lf.Summary.Resolved,
				"by_severity": lf.Summary.BySeverity,
			})
		}

		if domains == nil {
			domains = []map[string]interface{}{}
		}

		outputOK(map[string]interface{}{
			"domains": domains,
		})
		return nil
	},
}

// --- review-ledger-resolve ---

var reviewLedgerResolveCmd = &cobra.Command{
	Use:   "review-ledger-resolve",
	Short: "Mark a ledger entry as resolved",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		domain := mustGetString(cmd, "domain")
		if domain == "" {
			return nil
		}

		if !validDomains[domain] {
			outputError(1, fmt.Sprintf("invalid domain %q: must be one of %s", domain, strings.Join(colony.DomainOrder, ", ")), nil)
			return nil
		}

		id := mustGetString(cmd, "id")
		if id == "" {
			return nil
		}

		ledgerPath := fmt.Sprintf("reviews/%s/ledger.json", domain)
		var lf colony.ReviewLedgerFile
		if err := store.LoadJSON(ledgerPath, &lf); err != nil {
			outputError(1, "ledger not found", nil)
			return nil
		}

		found := false
		now := time.Now().UTC().Format(time.RFC3339)
		for i := range lf.Entries {
			if lf.Entries[i].ID == id {
				lf.Entries[i].Status = "resolved"
				lf.Entries[i].ResolvedAt = &now
				found = true
				break
			}
		}

		if !found {
			outputError(1, fmt.Sprintf("entry %q not found", id), nil)
			return nil
		}

		// Recompute summary and save
		lf.Summary = colony.ComputeSummary(lf.Entries)
		if err := store.SaveJSON(ledgerPath, lf); err != nil {
			outputError(2, fmt.Sprintf("failed to save ledger: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"resolved": true,
			"id":       id,
		})
		return nil
	},
}

// agentList returns a comma-separated list of known agent names for error messages.
func agentList() string {
	agents := make([]string, 0, len(agentAllowedDomains))
	for a := range agentAllowedDomains {
		agents = append(agents, a)
	}
	return strings.Join(agents, ", ")
}

func init() {
	reviewLedgerWriteCmd.Flags().String("domain", "", "Review domain (required): security, quality, performance, resilience, testing, history, bugs")
	reviewLedgerWriteCmd.Flags().Int("phase", 0, "Phase number (required, must be > 0)")
	reviewLedgerWriteCmd.Flags().String("findings", "", "JSON array of findings (required)")
	reviewLedgerWriteCmd.Flags().String("agent", "", "Agent writing the finding (optional)")
	reviewLedgerWriteCmd.Flags().String("agent-name", "", "Human-readable agent name (optional)")
	reviewLedgerWriteCmd.Flags().String("phase-name", "", "Human-readable phase name (optional)")

	reviewLedgerReadCmd.Flags().String("domain", "", "Review domain (required)")
	reviewLedgerReadCmd.Flags().Int("phase", 0, "Filter by phase number (optional)")
	reviewLedgerReadCmd.Flags().String("status", "", "Filter by status: open, resolved (optional)")

	reviewLedgerResolveCmd.Flags().String("domain", "", "Review domain (required)")
	reviewLedgerResolveCmd.Flags().String("id", "", "Entry ID to resolve (required)")

	rootCmd.AddCommand(reviewLedgerWriteCmd)
	rootCmd.AddCommand(reviewLedgerReadCmd)
	rootCmd.AddCommand(reviewLedgerSummaryCmd)
	rootCmd.AddCommand(reviewLedgerResolveCmd)
}
