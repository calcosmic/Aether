package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/calcosmic/Aether/pkg/learn"
	"github.com/spf13/cobra"
)

var learnExportCmd = &cobra.Command{
	Use:   "learn-export",
	Short: "Export learning entries as a portable pack",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		outputDir, _ := cmd.Flags().GetString("output")
		if outputDir == "" {
			outputDir = "."
		}
		outputPath := filepath.Join(outputDir, "learning-pack.json")

		learnStore := learn.NewColonyStore(store)
		path, report, err := learn.ExportPack(learnStore, outputPath)
		if err != nil {
			outputError(2, fmt.Sprintf("export failed: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"exported":   true,
			"path":       path,
			"redactions": len(report),
		})
		return nil
	},
}

var learnImportCmd = &cobra.Command{
	Use:   "learn-import [pack-path]",
	Short: "Import learning entries from a portable pack",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		packPath, _ := cmd.Flags().GetString("pack")
		if len(args) > 0 {
			packPath = args[0]
		}
		if packPath == "" {
			outputError(1, "pack path is required (use --pack flag or positional arg)", nil)
			return nil
		}

		preview, _ := cmd.Flags().GetBool("preview")
		learnStore := learn.NewColonyStore(store)

		if preview {
			entries, report, err := learn.ImportPreview(packPath)
			if err != nil {
				outputError(2, fmt.Sprintf("import preview failed: %v", err), nil)
				return nil
			}
			outputOK(map[string]interface{}{
				"preview":      true,
				"entry_count":  len(entries),
				"redactions":   len(report),
				"entries":      entries,
			})
			return nil
		}

		count, err := learn.ImportPack(learnStore, packPath)
		if err != nil {
			outputError(2, fmt.Sprintf("import failed: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"imported":    true,
			"entry_count": count,
			"pack":        packPath,
		})
		return nil
	},
}

func init() {
	learnExportCmd.Flags().String("output", ".", "Output directory for learning pack")
	learnImportCmd.Flags().String("pack", "", "Path to learning pack file")
	learnImportCmd.Flags().Bool("preview", false, "Preview entries without applying")

	rootCmd.AddCommand(learnExportCmd)
	rootCmd.AddCommand(learnImportCmd)
}
