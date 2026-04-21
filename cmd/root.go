// Package cmd implements the Aether CLI commands using Cobra.
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/calcosmic/Aether/pkg/storage"
	"github.com/calcosmic/Aether/pkg/trace"
	"github.com/spf13/cobra"
)

// Version is set via -ldflags at build time.
var Version = "0.0.0-dev"

// resolveVersion returns the best available version in priority order:
// 1. ldflags Version (set by goreleaser for release builds)
// 2. Repo `.aether/version.json` when running from a source checkout
// 3. Nearest git tag from the given directory (for dev builds)
// 4. Installed hub version
// 5. Fallback "0.0.0-dev"
func resolveVersion(dir ...string) string {
	// If ldflags set a real version (not the dev default), use it.
	if Version != "0.0.0-dev" {
		return normalizeVersion(Version)
	}

	// Determine where to look for git tags.
	gitDir := ""
	if len(dir) > 0 && dir[0] != "" {
		gitDir = findAetherModuleRoot(dir[0])
	} else {
		// Walk up from the binary to find the Aether go.mod.
		exe, err := os.Executable()
		if err == nil {
			gitDir = findAetherModuleRoot(filepath.Dir(exe))
		}
		if gitDir == "" {
			if cwd, err := os.Getwd(); err == nil {
				gitDir = findAetherModuleRoot(cwd)
			}
		}
	}

	if gitDir != "" {
		if repoVersion := readRepoVersion(gitDir); repoVersion != "" {
			return repoVersion
		}

		args := []string{"-C", gitDir, "describe", "--tags", "--abbrev=0"}
		out, err := exec.Command("git", args...).Output()
		if err == nil {
			v := strings.TrimSpace(string(out))
			return normalizeVersion(v)
		}
	}

	if hubVersion := readInstalledHubVersion(); hubVersion != "" {
		return hubVersion
	}

	return Version
}

func normalizeVersion(version string) string {
	return strings.TrimPrefix(strings.TrimSpace(version), "v")
}

func findAetherModuleRoot(start string) string {
	if start == "" {
		return ""
	}
	d, err := filepath.Abs(start)
	if err != nil {
		d = start
	}
	for {
		goMod := filepath.Join(d, "go.mod")
		data, err := os.ReadFile(goMod)
		if err == nil && strings.Contains(string(data), "github.com/calcosmic/Aether") {
			return d
		}
		parent := filepath.Dir(d)
		if parent == d {
			return ""
		}
		d = parent
	}
}

func readInstalledHubVersion() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(home, ".aether", "version.json"))
	if err != nil {
		return ""
	}
	var v struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return ""
	}
	return normalizeVersion(v.Version)
}

func readRepoVersion(root string) string {
	data, err := os.ReadFile(filepath.Join(root, ".aether", "version.json"))
	if err != nil {
		return ""
	}
	var v struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return ""
	}
	return normalizeVersion(v.Version)
}

func resolveReleaseVersion(explicit string) (string, error) {
	version := normalizeVersion(explicit)
	if version == "" {
		version = normalizeVersion(resolveVersion())
	}
	if version == "" || version == "0.0.0-dev" {
		return "", fmt.Errorf("cannot infer a release version from this dev binary; pass --version/--binary-version explicitly or run from an installed Aether checkout")
	}
	return version, nil
}

func init() {
	// Override Cobra's default version template to print "aether v<version>"
	// instead of "aether version v<version>"
	rootCmd.SetVersionTemplate("aether {{ .Version }}\n")
	rootCmd.Version = "v" + resolveVersion()
}

// store is the shared storage instance initialized by PersistentPreRunE.
// Commands that need data access should check this variable.
var store *storage.Store

// tracer is the shared trace logger initialized alongside store.
var tracer *trace.Tracer

// stdout and stderr are package-level writers that tests can override.
var stdout io.Writer = os.Stdout
var stderr io.Writer = os.Stderr

// rootCmd is the root Cobra command for the aether CLI.
var rootCmd = &cobra.Command{
	Use:   "aether",
	Short: "Aether Colony Utility Layer",
	// SilenceUsage and SilenceErrors prevent Cobra from printing
	// usage/error text automatically -- we control output ourselves.
	SilenceUsage:  true,
	SilenceErrors: true,
	// Custom version printer to match expected format "aether v<version>"
	Version: "v" + Version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip store initialization for commands that don't need it.
		if skipStoreInit(cmd) {
			return nil
		}

		dataDir := storage.ResolveDataDir(context.Background())
		s, err := storage.NewStore(dataDir)
		if err != nil {
			return fmt.Errorf("failed to initialize store: %w", err)
		}
		store = s
		tracer = trace.NewTracer(s)
		return nil
	},
}

// skipStoreInit returns true for commands that don't require a store
// (completion, version, help).
func skipStoreInit(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		switch c.Name() {
		case "completion", "version", "help":
			return true
		}
	}
	return false
}

// Execute runs the root command and returns any error.
func Execute() error {
	return rootCmd.Execute()
}

// ExitWithError prints the error to stderr and exits with code 1.
func ExitWithError(err error) {
	if err != nil {
		fmt.Fprintln(stderr, "Error:", err.Error())
	}
	os.Exit(1)
}
