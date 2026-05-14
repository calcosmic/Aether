package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var hostCmd = &cobra.Command{
	Use:   "host",
	Short: "Run the TypeScript orchestration host",
	Long:  "Delegate to the TypeScript host for plan, build, continue, lifecycle, and oracle workflows.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("a subcommand is required. Run 'aether host --help' for usage")
	},
}

func init() {
	rootCmd.AddCommand(hostCmd)

	// lifecycle
	hostCmd.AddCommand(&cobra.Command{
		Use:   "lifecycle [phase]",
		Short: "Run full plan->build->continue lifecycle via TS host",
		RunE:  makeHostSubcommand("lifecycle", true),
	})

	// plan
	hostCmd.AddCommand(&cobra.Command{
		Use:   "plan",
		Short: "Run plan workflow via TS host",
		RunE:  makeHostSubcommand("plan", false),
	})

	// build
	hostCmd.AddCommand(&cobra.Command{
		Use:   "build <phase>",
		Short: "Run build workflow via TS host",
		Args:  cobra.ExactArgs(1),
		RunE:  makeHostSubcommand("build", true),
	})

	// continue
	hostCmd.AddCommand(&cobra.Command{
		Use:   "continue",
		Short: "Run continue workflow via TS host",
		RunE:  makeHostSubcommand("continue", false),
	})

	// oracle
	hostCmd.AddCommand(&cobra.Command{
		Use:   "oracle",
		Short: "Run oracle workflow via TS host",
		RunE:  makeHostSubcommand("oracle", false),
	})
}

// makeHostSubcommand returns a cobra RunE that delegates to the TS host.
// If acceptsPositional is true, any positional args are forwarded.
func makeHostSubcommand(subcommand string, acceptsPositional bool) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		nodePath, err := discoverNode()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return fmt.Errorf("node not available")
		}

		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("cannot determine working directory: %w", err)
		}

		tsHostPath, hint := resolveTsHostPath(cwd)
		if tsHostPath == "" {
			fmt.Fprintf(os.Stderr, "Error: %s\n", hint)
			return fmt.Errorf("ts host not found")
		}

		tsArgs := []string{tsHostPath, subcommand}
		if acceptsPositional && len(args) > 0 {
			tsArgs = append(tsArgs, args...)
		}

		c := exec.Command(nodePath, tsArgs...)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	}
}

// discoverNode checks PATH for `node` and validates version >= 20.
func discoverNode() (string, error) {
	nodePath, err := exec.LookPath("node")
	if err != nil {
		return "", fmt.Errorf("Node.js >= 20 is required for the TypeScript host. Install from https://nodejs.org/")
	}

	out, err := exec.Command(nodePath, "--version").Output()
	if err != nil {
		return "", fmt.Errorf("failed to check node version: %w", err)
	}

	version := strings.TrimPrefix(strings.TrimSpace(string(out)), "v")
	majorStr := strings.Split(version, ".")[0]
	major, err := strconv.Atoi(majorStr)
	if err != nil {
		return "", fmt.Errorf("Node.js %s found, but >= 20 is required", version)
	}
	if major < 20 {
		return "", fmt.Errorf("Node.js %s found, but >= 20 is required", version)
	}

	return nodePath, nil
}

// resolveTsHostPath returns the path to the built TS host entry point and a
// fallback hint message (empty when path is found).
func resolveTsHostPath(cwd string) (string, string) {
	candidates := []string{}
	if cwd != "" {
		candidates = append(candidates, filepath.Join(cwd, ".aether", "ts-host", "dist", "host.js"))
	}
	if hub := strings.TrimSpace(resolveHubPath()); hub != "" {
		candidates = append(candidates, filepath.Join(hub, "system", "ts-host", "dist", "host.js"))
	}

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, ""
		}
	}

	// Determine which fallback message to show
	srcDir := filepath.Join(cwd, ".aether", "ts-host", "src")
	if _, err := os.Stat(srcDir); err == nil {
		return "", "TS host needs to be built. Run `npm --prefix .aether/ts-host run build`"
	}

	nodeModulesDir := filepath.Join(cwd, ".aether", "ts-host", "node_modules")
	if _, err := os.Stat(nodeModulesDir); os.IsNotExist(err) {
		return "", "TS host dependencies missing. Run `npm --prefix .aether/ts-host ci`"
	}

	return "", "TS host assets not found. Run `aether update --force` to install them."
}

