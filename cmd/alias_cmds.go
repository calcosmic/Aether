package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(pheromoneExportXMLCmd)
	rootCmd.AddCommand(pheromoneImportXMLCmd)
	rootCmd.AddCommand(wisdomExportXMLCmd)
	rootCmd.AddCommand(wisdomImportXMLCmd)
	rootCmd.AddCommand(registryExportXMLCmd)
	rootCmd.AddCommand(registryImportXMLCmd)
	rootCmd.AddCommand(colonyArchiveXMLCmd)
}

// --- Flat alias commands for XML exchange ---
//
// These commands provide flat names (e.g. "pheromone-export-xml") that map to
// the same logic as the nested exchange subcommands (e.g. "export pheromones").
// Slash commands reference the flat names directly.

var pheromoneExportXMLCmd = &cobra.Command{
	Use:          "pheromone-export-xml",
	Short:        "Export pheromone signals to XML (alias for export pheromones)",
	Aliases:      []string{"export-signals"},
	SilenceUsage: true,
	RunE:         runExportPheromones,
}

var pheromoneImportXMLCmd = &cobra.Command{
	Use:          "pheromone-import-xml <file>",
	Short:        "Import pheromone signals from XML (alias for import pheromones)",
	Aliases:      []string{"import-signals"},
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE:         runImportPheromones,
}

var wisdomExportXMLCmd = &cobra.Command{
	Use:          "wisdom-export-xml",
	Short:        "Export queen wisdom to XML (alias for export wisdom)",
	SilenceUsage: true,
	RunE:         runExportWisdom,
}

var wisdomImportXMLCmd = &cobra.Command{
	Use:          "wisdom-import-xml <file>",
	Short:        "Import queen wisdom from XML (alias for import wisdom)",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE:         runImportWisdom,
}

var registryExportXMLCmd = &cobra.Command{
	Use:          "registry-export-xml",
	Short:        "Export colony registry to XML (alias for export registry)",
	SilenceUsage: true,
	RunE:         runExportRegistry,
}

var registryImportXMLCmd = &cobra.Command{
	Use:          "registry-import-xml <file>",
	Short:        "Import colony registry from XML (alias for import registry)",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE:         runImportRegistry,
}

var colonyArchiveXMLCmd = &cobra.Command{
	Use:          "colony-archive-xml [output-path]",
	Short:        "Export complete colony archive to XML (alias for export archive)",
	SilenceUsage: true,
	Args:         cobra.MaximumNArgs(1),
	RunE:         runExportArchive,
}

func init() {
	pheromoneExportXMLCmd.Flags().String("output", "", "Write XML to file instead of stdout")

	pheromoneImportXMLCmd.Flags().String("input", "", "Input file path (positional arg preferred)")

	wisdomExportXMLCmd.Flags().String("output", "", "Write XML to file instead of stdout")
	wisdomExportXMLCmd.Flags().Float64("min-confidence", 0.5, "Minimum confidence threshold for wisdom export")
	wisdomExportXMLCmd.Flags().String("colony", "", "Colony ID for wisdom export")

	wisdomImportXMLCmd.Flags().String("input", "", "Input file path (positional arg preferred)")

	registryExportXMLCmd.Flags().String("output", "", "Write XML to file instead of stdout")

	registryImportXMLCmd.Flags().String("input", "", "Input file path (positional arg preferred)")

	colonyArchiveXMLCmd.Flags().String("output", "", "Output file path (--output or positional arg)")
}
