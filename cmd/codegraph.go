package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/calcosmic/Aether/pkg/codegraph"
	"github.com/spf13/cobra"
)

var (
	codegraphScanLangs  []string
	codegraphQueryFile  string
	codegraphQueryDepth int
)

func init() {
	// codebase-scan
	scanCmd := &cobra.Command{
		Use:   "codebase-scan",
		Short: "Scan codebase file dependencies and build import graph",
		Long:  "Walks the repository source tree, parses import/require/include statements, and produces a file dependency graph stored in .aether/data/codebase-graph.json",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runCodebaseScan,
	}
	scanCmd.Flags().StringSliceVar(&codegraphScanLangs, "langs", nil, "languages to scan (comma-separated, e.g. go,typescript,python)")

	// codebase-query
	queryCmd := &cobra.Command{
		Use:   "codebase-query",
		Short: "Query the codebase dependency graph",
		Long:  "Loads the codebase-graph.json and returns files related to the given file within a specified depth",
		Args:  cobra.NoArgs,
		RunE:  runCodebaseQuery,
	}
	queryCmd.Flags().StringVar(&codegraphQueryFile, "file", "", "source file to query dependencies for")
	queryCmd.Flags().IntVar(&codegraphQueryDepth, "depth", 2, "max hops to traverse (default 2)")

	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(queryCmd)
}

func runCodebaseScan(cmd *cobra.Command, args []string) error {
	root := "."
	if len(args) > 0 {
		root = args[0]
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve root: %w", err)
	}

	graph, stats, err := codegraph.Scan(absRoot, codegraphScanLangs)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	// Save to store if available, otherwise to working directory
	if store != nil {
		outPath := filepath.Join(store.BasePath(), "codebase-graph.json")
		if err := graph.Save(outPath); err != nil {
			return fmt.Errorf("save graph: %w", err)
		}
		logActivity("codebase-scan", fmt.Sprintf("scanned %d files, %d edges, languages: %s", stats.FilesScanned, stats.EdgesFound, strings.Join(stats.Languages, ",")))
		outputOK(map[string]interface{}{
			"action":        "codebase-scan",
			"files_scanned": stats.FilesScanned,
			"edges_found":   stats.EdgesFound,
			"languages":     stats.Languages,
			"skipped_dirs":  stats.SkippedDirs,
			"output":        "codebase-graph.json",
		})
	} else {
		outPath := filepath.Join(absRoot, "codebase-graph.json")
		if err := graph.Save(outPath); err != nil {
			return fmt.Errorf("save graph: %w", err)
		}
		outputOK(map[string]interface{}{
			"action":        "codebase-scan",
			"files_scanned": stats.FilesScanned,
			"edges_found":   stats.EdgesFound,
			"languages":     stats.Languages,
			"skipped_dirs":  stats.SkippedDirs,
			"output":        outPath,
		})
	}

	return nil
}

func runCodebaseQuery(cmd *cobra.Command, args []string) error {
	if codegraphQueryFile == "" {
		return fmt.Errorf("--file is required")
	}

	// Load graph
	var graphPath string
	if store != nil {
		graphPath = filepath.Join(store.BasePath(), "codebase-graph.json")
	} else {
		graphPath = "codebase-graph.json"
	}

	graph, err := codegraph.Load(graphPath)
	if err != nil {
		if os.IsNotExist(err) {
			outputErrorMessage("no codebase graph found — run 'aether codebase-scan' first")
			return nil
		}
		return fmt.Errorf("load graph: %w", err)
	}

	// Normalize the query path
	queryFile := filepath.ToSlash(filepath.Clean(codegraphQueryFile))

	// Check if file exists in graph
	found := false
	for _, f := range graph.Files {
		if filepath.ToSlash(filepath.Clean(f.Path)) == queryFile {
			found = true
			break
		}
	}
	if !found {
		outputErrorMessage(fmt.Sprintf("file %s not found in graph", queryFile))
		return nil
	}

	related := graph.RelatedFiles(queryFile, codegraphQueryDepth)

	// Build direct imports info
	directDeps := make(map[string][]string)
	for _, e := range graph.Edges {
		src := filepath.ToSlash(filepath.Clean(e.Source))
		if src == queryFile {
			directDeps["imports"] = append(directDeps["imports"], filepath.ToSlash(filepath.Clean(e.Target)))
		}
	}

	// Also find reverse deps (who imports this file)
	for _, e := range graph.Edges {
		tgt := filepath.ToSlash(filepath.Clean(e.Target))
		if tgt == queryFile {
			directDeps["imported_by"] = append(directDeps["imported_by"], filepath.ToSlash(filepath.Clean(e.Source)))
		}
	}

	outputOK(map[string]interface{}{
		"action":       "codebase-query",
		"file":         queryFile,
		"depth":        codegraphQueryDepth,
		"imports":      directDeps["imports"],
		"imported_by":  directDeps["imported_by"],
		"related":      related,
		"related_count": len(related),
	})

	return nil
}

// runCodebaseScanFromColonize is the internal hook called during /ant-colonize.
// It runs the scan and saves to the store without CLI output.
func runCodebaseScanFromColonize(root string, langs []string) (*codegraph.Stats, error) {
	graph, stats, err := codegraph.Scan(root, langs)
	if err != nil {
		return nil, fmt.Errorf("codebase scan: %w", err)
	}

	if store != nil {
		outPath := filepath.Join(store.BasePath(), "codebase-graph.json")
		if err := graph.Save(outPath); err != nil {
			return stats, fmt.Errorf("save codebase graph: %w", err)
		}
	}

	return stats, nil
}
