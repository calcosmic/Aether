package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// CatalogEntry describes a single Cobra command in the audit catalog.
type CatalogEntry struct {
	Name             string   `json:"name"`
	ShortDescription string   `json:"short_description"`
	Flags            []string `json:"flags"`
	ParentCommand    string   `json:"parent_command"`
	HasSubcommands   bool     `json:"has_subcommands"`
	OutputMode       string   `json:"output_mode"`
}

// buildAuditCatalog walks the full Cobra command tree starting from root
// and produces a flat, sorted slice of CatalogEntry values. It skips
// commands that are not available (hidden, help, completion).
func buildAuditCatalog(root *cobra.Command) []CatalogEntry {
	var entries []CatalogEntry
	walkCommands(root, "", &entries)
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
func walkCommands(cmd *cobra.Command, parent string, entries *[]CatalogEntry) {
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
		*entries = append(*entries, entry)
		walkCommands(child, child.Name(), entries)
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
		catalog := buildAuditCatalog(rootCmd)
		if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
			data, err := json.MarshalIndent(catalog, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal audit catalog: %w", err)
			}
			fmt.Fprintln(stdout, string(data))
		} else {
			outputWorkflow(catalog, renderAuditCatalogVisual(catalog))
		}
		return nil
	},
}

func init() {
	auditCatalogCmd.Flags().Bool("json", false, "Output JSON instead of visual report")
	rootCmd.AddCommand(auditCatalogCmd)
}

func renderAuditCatalogVisual(catalog []CatalogEntry) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("audit-catalog"), "Command Catalog"))
	b.WriteString(visualDivider)

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
