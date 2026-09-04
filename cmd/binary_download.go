package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/downloader"
	"github.com/spf13/cobra"
)

type maintenanceBinaryFetcher func(version, destDir string) (*downloader.DownloadResult, error)

type maintenanceStagedBinary struct {
	Version string
	Name    string
	Source  string
	Digest  string
	Content []byte
}

// stageMaintenanceBinaryDownload downloads into an isolated temporary root,
// verifies the reported version and exact regular file, and returns immutable
// bytes for the shared lifecycle transaction. The live binary destination is
// not touched by this step.
func stageMaintenanceBinaryDownload(version string, channel runtimeChannel, fetch maintenanceBinaryFetcher) (maintenanceStagedBinary, error) {
	version = normalizeVersion(strings.TrimSpace(version))
	if version == "" || version == "latest" {
		return maintenanceStagedBinary{}, fmt.Errorf("binary maintenance: exact version is required")
	}
	if fetch == nil {
		return maintenanceStagedBinary{}, fmt.Errorf("binary maintenance: downloader is required")
	}
	tempDir, err := os.MkdirTemp("", "aether-binary-stage-*")
	if err != nil {
		return maintenanceStagedBinary{}, fmt.Errorf("binary maintenance: create staging root: %w", err)
	}
	defer os.RemoveAll(tempDir)

	result, err := fetch(version, tempDir)
	if err != nil {
		return maintenanceStagedBinary{}, fmt.Errorf("binary maintenance: download and checksum verification failed: %w", err)
	}
	if result == nil || !result.Success {
		return maintenanceStagedBinary{}, fmt.Errorf("binary maintenance: downloader did not report success")
	}
	result, err = alignDownloadedBinaryToChannel(result, tempDir, channel)
	if err != nil {
		return maintenanceStagedBinary{}, fmt.Errorf("binary maintenance: align channel binary: %w", err)
	}
	if normalizeVersion(result.Version) != version {
		return maintenanceStagedBinary{}, fmt.Errorf("binary maintenance: downloaded version %s does not match requested version %s", result.Version, version)
	}
	path := filepath.Clean(result.Path)
	if !filepath.IsAbs(path) || !pathIsWithin(tempDir, path) || path == tempDir {
		return maintenanceStagedBinary{}, fmt.Errorf("binary maintenance: downloader returned out-of-stage path %q", result.Path)
	}
	info, err := os.Lstat(path)
	if err != nil {
		return maintenanceStagedBinary{}, fmt.Errorf("binary maintenance: inspect staged binary: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return maintenanceStagedBinary{}, fmt.Errorf("binary maintenance: staged binary must be a regular file")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return maintenanceStagedBinary{}, fmt.Errorf("binary maintenance: read staged binary: %w", err)
	}
	if len(content) == 0 {
		return maintenanceStagedBinary{}, fmt.Errorf("binary maintenance: staged binary is empty")
	}
	return maintenanceStagedBinary{
		Version: version,
		Name:    filepath.Base(path),
		Source:  fmt.Sprintf("github-release:v%s:%s", version, lifecycleDigest(content)),
		Digest:  lifecycleDigest(content),
		Content: content,
	}, nil
}

// binaryDownloadCmd implements "aether binary-download" which downloads the
// Go binary from GitHub Releases for the current platform.
var binaryDownloadCmd = &cobra.Command{
	Use:   "binary-download",
	Short: "Download the aether Go binary from GitHub Releases",
	Long: `Download the aether Go binary from GitHub Releases for the current platform.

Detects your OS and architecture, downloads the correct archive from
https://github.com/calcosmic/Aether/releases, verifies the SHA-256
checksum, and installs the binary atomically into the selected channel bin.

Use this to update the binary without reinstalling everything.`,
	Args: cobra.NoArgs,
	RunE: runBinaryDownload,
}

func init() {
	binaryDownloadCmd.Flags().String("channel", "", "Runtime channel to download for (stable or dev; default: infer from binary/env)")
	binaryDownloadCmd.Flags().String("version", "", "Version to download (default: current aether version)")
	binaryDownloadCmd.Flags().String("dest", "", "Destination directory (default: channel-specific hub bin)")
	rootCmd.AddCommand(binaryDownloadCmd)
}

func runBinaryDownload(cmd *cobra.Command, args []string) error {
	channel := runtimeChannelFromFlag(cmd.Flags())

	versionFlag, _ := cmd.Flags().GetString("version")
	version, err := resolveReleaseVersion(versionFlag)
	if err != nil {
		return err
	}

	destDir, _ := cmd.Flags().GetString("dest")
	if destDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}
		destDir = filepath.Join(home, defaultBinaryDestSubdirForChannel(channel))
	}
	destDir, err = filepath.Abs(destDir)
	if err != nil {
		return fmt.Errorf("resolve destination directory: %w", err)
	}
	info, err := os.Lstat(destDir)
	if err != nil {
		return fmt.Errorf("binary destination directory must already exist: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("binary destination directory must be a real directory")
	}
	if store == nil {
		return fmt.Errorf("binary download requires an initialized lifecycle store")
	}
	repositoryRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolve repository root: %w", err)
	}
	repositoryRoot, err = filepath.Abs(repositoryRoot)
	if err != nil {
		return fmt.Errorf("resolve repository root: %w", err)
	}

	outputOK(map[string]interface{}{
		"message": fmt.Sprintf("Downloading %s v%s binary for %s/%s...", defaultBinaryName(channel), version, getGOOS(), getGOArch()),
		"version": version,
		"dest":    destDir,
	})

	staged, err := stageMaintenanceBinaryDownload(version, channel, downloader.DownloadBinary)
	if err != nil {
		if downloader.IsVersionNotFoundErr(err) {
			return fmt.Errorf("version v%s not found. Run 'aether version' to check the latest version: %w", version, err)
		}
		return fmt.Errorf("download failed: %w", err)
	}
	destination := filepath.Join(destDir, staged.Name)
	plan := maintenanceMutationPlan{
		SchemaVersion: maintenanceMutationSchemaVersion, Operation: "binary-download",
		TransactionID: "binary-download-" + time.Now().UTC().Format("20060102T150405.000000000Z"),
		SourceRoot:    repositoryRoot, DestinationRoot: destDir, Channel: channel,
		CurrentVersion: resolveVersion(), DesiredVersion: staged.Version,
		Checkpoint: "maintenance:binary-download:verified", Recovery: "aether resume",
		Allowlist: lifecycleTransactionAllowlist{RepositoryRoot: repositoryRoot, LifecycleDataRoot: filepath.Clean(store.BasePath()), BinaryDestination: destination},
		Targets:   []maintenanceMutationTarget{{Root: lifecycleTransactionRootBinaryDestination, RelativeTarget: filepath.Base(destination), Source: staged.Source, Action: lifecycleTransactionWrite, Content: staged.Content, Managed: true}},
	}
	mutation, err := commitMaintenanceMutation(plan)
	if err != nil {
		return fmt.Errorf("binary transaction failed (%s): %w", mutation.StateEffect, err)
	}

	outputOK(map[string]interface{}{
		"message":      fmt.Sprintf("Binary installed successfully to %s", destination),
		"path":         destination,
		"version":      staged.Version,
		"preview":      mutation.Preview,
		"receipt":      mutation.Receipt,
		"state_effect": mutation.StateEffect,
		"verification": mutation.Verification,
		"recovery":     mutation.Recovery,
	})

	return nil
}

// getGOOS returns runtime.GOOS for use in output messages.
func getGOOS() string { return runtime.GOOS }

// getGOArch returns runtime.GOARCH for use in output messages.
func getGOArch() string { return runtime.GOARCH }
