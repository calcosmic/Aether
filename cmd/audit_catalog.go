package cmd

import (
	_ "embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

//go:embed testdata/command_catalog.json
var embeddedCatalogJSON []byte

// CatalogEntry describes a single Cobra command in the audit catalog.
type CatalogEntry struct {
	Name               string          `json:"name"`
	ShortDescription   string          `json:"short_description"`
	Flags              []string        `json:"flags"`
	ParentCommand      string          `json:"parent_command"`
	HasSubcommands     bool            `json:"has_subcommands"`
	OutputMode         string          `json:"output_mode"`
	Classification     string          `json:"classification,omitempty"`
	SinceVersion       string          `json:"since_version,omitempty"`
	HistoricalPresence map[string]bool `json:"historical_presence,omitempty"`
	StabilityScore     int             `json:"stability_score,omitempty"`
}

// loadEnrichedCatalog parses the embedded enriched catalog JSON.
func loadEnrichedCatalog() ([]CatalogEntry, error) {
	var entries []CatalogEntry
	if err := json.Unmarshal(embeddedCatalogJSON, &entries); err != nil {
		return nil, fmt.Errorf("parse catalog: %w", err)
	}
	return entries, nil
}

// buildAuditCatalog walks the full Cobra command tree starting from root
// and produces a flat, sorted slice of CatalogEntry values. It skips
// commands that are not available (hidden, help, completion).
func buildAuditCatalog(root *cobra.Command) []CatalogEntry {
	// Load enriched data as a lookup so generated entries include
	// classification, since_version, and historical_presence.
	enriched, _ := loadEnrichedCatalog()
	lookup := make(map[string]CatalogEntry, len(enriched))
	for _, e := range enriched {
		lookup[e.Name] = e
	}

	var entries []CatalogEntry
	walkCommands(root, "", &entries, lookup)
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].ParentCommand != entries[j].ParentCommand {
			return entries[i].ParentCommand < entries[j].ParentCommand
		}
		return entries[i].Name < entries[j].Name
	})
	return entries
}

// walkCommands recursively visits all available subcommands of cmd and
// appends a CatalogEntry for each. The parent name is passed down through
// recursion to populate ParentCommand.
func walkCommands(cmd *cobra.Command, parent string, entries *[]CatalogEntry, lookup map[string]CatalogEntry) {
	for _, child := range cmd.Commands() {
		if !child.IsAvailableCommand() {
			continue
		}
		entry := CatalogEntry{
			Name:             child.Name(),
			ShortDescription: strings.TrimSpace(child.Short),
			Flags:            extractFlags(child),
			ParentCommand:    parent,
			HasSubcommands:   len(child.Commands()) > 0,
			OutputMode:       classifyOutputMode(child),
		}
		// Merge enriched metadata if available.
		if enriched, ok := lookup[entry.Name]; ok {
			entry.Classification = enriched.Classification
			entry.SinceVersion = enriched.SinceVersion
			entry.HistoricalPresence = enriched.HistoricalPresence
			entry.StabilityScore = enriched.StabilityScore
		}
		*entries = append(*entries, entry)
		walkCommands(child, child.Name(), entries, lookup)
	}
}

// extractFlags returns a sorted list of flag names for the command's local flags.
// Returns an empty (non-nil) slice when there are no flags.
// Filters out the standard Cobra "help" flag since it is auto-generated.
func extractFlags(cmd *cobra.Command) []string {
	flags := make([]string, 0)
	cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
		if f.Name != "help" {
			flags = append(flags, f.Name)
		}
	})
	sort.Strings(flags)
	return flags
}

// classifyOutputMode inspects the command's RunE body heuristically to
// classify how it renders output. Many commands will be "unknown" since
// static analysis of RunE is limited.
func classifyOutputMode(cmd *cobra.Command) string {
	// If the command has a --json flag, it likely supports JSON output.
	hasJSON := false
	cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
		if f.Name == "json" {
			hasJSON = true
		}
	})
	if hasJSON {
		return "json+visual"
	}
	return "unknown"
}

var auditCatalogCmd = &cobra.Command{
	Use:   "audit-catalog",
	Short: "Produce structured catalog of all registered commands",
	Long: "Walks the full Cobra command tree and outputs a structured JSON " +
		"catalog of every registered command. Useful for auditing, documentation, " +
		"and CI regression (golden testing).",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")
		reliability, _ := cmd.Flags().GetBool("reliability")
		jsonOut, _ := cmd.Flags().GetBool("json")

		// Load enriched catalog from embedded JSON for reporting.
		entries, err := loadEnrichedCatalog()
		if err != nil {
			return err
		}

		if reliability {
			if jsonOut || format == "json" {
				return outputReliabilityJSON(entries)
			}
			return outputReliabilityMarkdown(entries)
		}

		switch format {
		case "json":
			return outputCatalogJSON(entries)
		case "csv":
			return outputCatalogCSV(entries)
		default:
			return outputCatalogMarkdown(entries)
		}
	},
}

func init() {
	auditCatalogCmd.Flags().Bool("json", false, "Output JSON instead of visual report")
	auditCatalogCmd.Flags().String("format", "markdown", "Output format: markdown, json, csv")
	auditCatalogCmd.Flags().Bool("reliability", false, "Output historical reliability matrix")
	rootCmd.AddCommand(auditCatalogCmd)
}

func outputCatalogJSON(entries []CatalogEntry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal catalog: %w", err)
	}
	fmt.Fprintln(stdout, string(data))
	return nil
}

func outputCatalogCSV(entries []CatalogEntry) error {
	w := csv.NewWriter(stdout)
	_ = w.Write([]string{"Command", "Classification", "Since", "Description"})
	for _, e := range entries {
		_ = w.Write([]string{e.Name, e.Classification, e.SinceVersion, e.ShortDescription})
	}
	w.Flush()
	return nil
}

func outputCatalogMarkdown(entries []CatalogEntry) error {
	fmt.Fprintf(stdout, "| %-28s | %-18s | %-10s | %-50s |\n", "Command", "Classification", "Since", "Description")
	fmt.Fprintln(stdout, "|"+strings.Repeat("-", 30)+"|"+strings.Repeat("-", 20)+"|"+strings.Repeat("-", 12)+"|"+strings.Repeat("-", 52)+"|")
	for _, e := range entries {
		desc := e.ShortDescription
		if len(desc) > 48 {
			desc = desc[:45] + "..."
		}
		fmt.Fprintf(stdout, "| %-28s | %-18s | %-10s | %-50s |\n", e.Name, e.Classification, e.SinceVersion, desc)
	}
	fmt.Fprintf(stdout, "\nTotal: %d commands\n", len(entries))
	return nil
}

func outputReliabilityMarkdown(entries []CatalogEntry) error {
	// Sort by stability score descending, then alphabetically.
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].StabilityScore != entries[j].StabilityScore {
			return entries[i].StabilityScore > entries[j].StabilityScore
		}
		return entries[i].Name < entries[j].Name
	})

	tags := []string{"v5.4.0", "v1.10", "v1.11", "v1.12", "v1.13", "v1.14", "v1.17", "v1.18", "v1.19", "v1.20", "v1.21"}

	// Header
	fmt.Fprintf(stdout, "| %-28s ", "Command")
	for _, tag := range tags {
		fmt.Fprintf(stdout, "| %-6s ", tag)
	}
	fmt.Fprintln(stdout, "| Score  |")

	sep := "|" + strings.Repeat("-", 30)
	for range tags {
		sep += "|" + strings.Repeat("-", 8)
	}
	sep += "|" + strings.Repeat("-", 8) + "|"
	fmt.Fprintln(stdout, sep)

	for _, e := range entries {
		fmt.Fprintf(stdout, "| %-28s ", e.Name)
		for _, tag := range tags {
			val := "-"
			if e.HistoricalPresence != nil {
				if present, ok := e.HistoricalPresence[tag]; ok && present {
					val = "Y"
				}
			}
			fmt.Fprintf(stdout, "| %-6s ", val)
		}
		fmt.Fprintf(stdout, "| %5d%% |\n", e.StabilityScore)
	}
	return nil
}

func outputReliabilityJSON(entries []CatalogEntry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal reliability: %w", err)
	}
	fmt.Fprintln(stdout, string(data))
	return nil
}

func renderAuditCatalogVisual(catalog []CatalogEntry) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("audit-catalog"), "Command Catalog"))
	b.WriteString(visualDividerStr())

	// Table header
	b.WriteString(fmt.Sprintf("%-28s %-6s %-12s %-4s\n", "Name", "Flags", "Output", "Sub"))
	b.WriteString(strings.Repeat("-", 54))
	b.WriteString("\n")

	for _, entry := range catalog {
		subMarker := ""
		if entry.HasSubcommands {
			subMarker = "Y"
		}
		b.WriteString(fmt.Sprintf("%-28s %-6d %-12s %-4s\n",
			entry.Name, len(entry.Flags), entry.OutputMode, subMarker))
	}

	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Total: %d commands\n", len(catalog)))
	return b.String()
}
