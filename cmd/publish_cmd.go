package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish Aether from source to the shared hub",
	Long: "Build the aether binary and sync companion files to the hub, " +
		"ensuring binary and hub versions agree atomically.\n\n" +
		"This replaces the ad-hoc `aether install --package-dir \"$PWD\"` pattern " +
		"with a dedicated, discoverable command that verifies version agreement " +
		"after publish completes.",
	Args: cobra.NoArgs,
	RunE: runPublish,
}

func init() {
	publishCmd.Flags().String("package-dir", "", "Source directory (default: current directory)")
	publishCmd.Flags().String("home-dir", "", "User home directory (default: $HOME)")
	publishCmd.Flags().String("channel", "", "Runtime channel (stable or dev; default: infer from binary/env)")
	publishCmd.Flags().String("binary-dest", "", "Destination directory for the built binary")
	publishCmd.Flags().Bool("skip-build-binary", false, "Skip go build and use existing binary")

	rootCmd.AddCommand(publishCmd)
}

func runPublish(cmd *cobra.Command, args []string) error {
	channel := runtimeChannelFromFlag(cmd.Flags())

	packageDir, err := cmd.Flags().GetString("package-dir")
	if err != nil {
		return fmt.Errorf("failed to read --package-dir: %w", err)
	}
	if packageDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("cannot determine current directory: %w", err)
		}
		packageDir = cwd
	}

	homeDir, err := cmd.Flags().GetString("home-dir")
	if err != nil {
		return fmt.Errorf("failed to read --home-dir: %w", err)
	}
	if homeDir == "" {
		homeDir = os.Getenv("HOME")
		if homeDir == "" {
			homeDir = os.Getenv("USERPROFILE")
		}
		if homeDir == "" {
			return fmt.Errorf("cannot determine home directory: set HOME or use --home-dir")
		}
	}

	if !isAetherSourceCheckout(packageDir) {
		return fmt.Errorf("%s does not appear to be an Aether source checkout (missing go.mod or cmd/aether/main.go)", packageDir)
	}

	sourceRoot := findAetherModuleRoot(packageDir)
	// During publish, the source version.json is authoritative — not the running
	// binary's ldflags version. If we used resolveVersion here, an old binary with
	// baked-in ldflags would override a freshly bumped version.json.
	version := readRepoVersion(sourceRoot)
	if version == "" {
		version = resolveVersion(sourceRoot)
	}

	skipBuildBinary, _ := cmd.Flags().GetBool("skip-build-binary")
	if !skipBuildBinary {
		destDir, _ := cmd.Flags().GetString("binary-dest")
		if destDir == "" {
			destDir = defaultLocalBinaryDest(homeDir, channel)
		}

		outputWorkflow(map[string]interface{}{
			"message": fmt.Sprintf("Building %s binary...", defaultBinaryName(channel)),
			"version": version,
			"dest":    destDir,
		}, renderBinaryActionVisual("Binary Build", fmt.Sprintf("Building %s binary...", defaultBinaryName(channel)), version, destDir))

		if _, err := buildLocalBinary(sourceRoot, destDir, version, channel); err != nil {
			return fmt.Errorf("binary build failed: %w", err)
		}
	}

	// Build TS host assets (best-effort — warn but don't fail publish)
	if tsHostErr := buildTsHostAssets(sourceRoot); tsHostErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: TS host build skipped: %v\n", tsHostErr)
	}

	hubDir := resolveHubPathForHome(homeDir, channel)

	if err := validateChannelIsolation(channel, hubDir); err != nil {
		return err
	}

	// Read old hub version before sync for the warning message
	oldHubVersion := readHubVersionAtPath(hubDir)

	hubResult := setupInstallHub(hubDir, packageDir)
	if errVal, ok := hubResult["error"].(string); ok && errVal != "" {
		return fmt.Errorf("hub sync failed: %v", errVal)
	}

	// Sync TS host assets to hub (best-effort)
	if tsHostErr := syncTsHostToHub(hubDir, sourceRoot); tsHostErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: TS host hub sync skipped: %v\n", tsHostErr)
	}

	if shouldSyncPlatformHomes(channel) {
		_, platformErrors := syncPlatformHomeAssets(packageDir, homeDir, channel)
		if len(platformErrors) > 0 {
			return fmt.Errorf("platform home sync failed: %s", strings.Join(platformErrors, "; "))
		}
	}

	// Verification: ensure binary version and hub version agree
	hubVersion := readHubVersionAtPath(hubDir)
	if hubVersion == "" {
		return fmt.Errorf("publish verification failed: no hub version found after sync")
	}
	if version != hubVersion {
		return fmt.Errorf("publish verification failed: binary version %s does not match hub version %s", version, hubVersion)
	}

	if oldHubVersion != "" && oldHubVersion != version {
		fmt.Fprintf(os.Stderr, "Warning: hub version updated from %s to %s\n", oldHubVersion, version)
	}

	outputWorkflow(map[string]interface{}{
		"ok":      true,
		"message": fmt.Sprintf("Publish complete: Aether v%s published to %s", version, hubDir),
		"version": version,
		"hub":     hubDir,
	}, renderBinaryActionVisual("Publish Complete", fmt.Sprintf("Aether v%s published", version), version, hubDir))

	warnBinaryCoLocation(channel, homeDir)

	return nil
}

// validateChannelIsolation rejects cross-channel publish operations to prevent
// dev publishes from contaminating stable hubs and vice versa.
func validateChannelIsolation(channel runtimeChannel, hubDir string) error {
	absHub, err := filepath.Abs(hubDir)
	if err != nil {
		absHub = hubDir
	}
	switch channel {
	case channelDev:
		if strings.Contains(absHub, "/.aether/") || strings.HasSuffix(absHub, "/.aether") {
			return fmt.Errorf("dev publish cannot target stable hub at %s: use --channel stable or unset AETHER_HUB_DIR", hubDir)
		}
	case channelStable:
		if strings.Contains(absHub, "/.aether-dev/") || strings.HasSuffix(absHub, "/.aether-dev") {
			return fmt.Errorf("stable publish cannot target dev hub at %s: use --channel dev or unset AETHER_HUB_DIR", hubDir)
		}
	}
	return nil
}

// warnBinaryCoLocation prints a note when both stable and dev binaries exist
// in the same destination directory. Purely informational — does not block publish.
func warnBinaryCoLocation(channel runtimeChannel, homeDir string) {
	other := defaultBinaryName(channelStable)
	if channel == channelStable {
		other = defaultBinaryName(channelDev)
	}
	destDir := defaultLocalBinaryDest(homeDir, channel)
	otherPath := filepath.Join(destDir, other)
	if _, err := os.Stat(otherPath); err == nil {
		fmt.Fprintf(os.Stderr, "Note: %s binary also present in %s\n", other, destDir)
	}
}

// readHubVersionAtPath reads the version from a hub directory's version.json.
func readHubVersionAtPath(hubDir string) string {
	for _, rel := range []string{"version.json", filepath.Join("system", "version.json")} {
		if version := readVersionJSONFile(filepath.Join(hubDir, rel)); version != "" {
			return version
		}
	}
	return ""
}

// buildTsHostAssets runs npm ci and npm run build in .aether/ts-host/.
// Returns an error if npm is missing or the build fails.
func buildTsHostAssets(sourceRoot string) error {
	tsHostDir := filepath.Join(sourceRoot, ".aether", "ts-host")
	if _, err := os.Stat(tsHostDir); os.IsNotExist(err) {
		return fmt.Errorf("ts-host directory not found: %s", tsHostDir)
	}

	// Check npm availability
	if _, err := exec.LookPath("npm"); err != nil {
		return fmt.Errorf("npm not found in PATH: %w", err)
	}

	// npm ci
	cmd := exec.Command("npm", "ci", "--prefix", tsHostDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("npm ci failed: %w", err)
	}

	// npm run build
	cmd = exec.Command("npm", "run", "build", "--prefix", tsHostDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("npm run build failed: %w", err)
	}

	return nil
}

// syncTsHostToHub copies TS host dist/ and package.json to the hub system/ts-host/.
func syncTsHostToHub(hubDir, sourceRoot string) error {
	srcDir := filepath.Join(sourceRoot, ".aether", "ts-host")
	dstDir := filepath.Join(hubDir, "system", "ts-host")

	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dstDir, err)
	}

	// Copy dist/
	srcDist := filepath.Join(srcDir, "dist")
	dstDist := filepath.Join(dstDir, "dist")
	if _, err := os.Stat(srcDist); err == nil {
		res := syncDir(srcDist, dstDist, syncOptions{cleanup: true})
		if len(res.errors) > 0 {
			return fmt.Errorf("sync dist/: %s", strings.Join(res.errors, "; "))
		}
	}

	// Copy package.json
	srcPkg := filepath.Join(srcDir, "package.json")
	dstPkg := filepath.Join(dstDir, "package.json")
	if _, err := os.Stat(srcPkg); err == nil {
		if err := copyFile(srcPkg, dstPkg); err != nil {
			return fmt.Errorf("copy package.json: %w", err)
		}
	}

	// Copy package-lock.json if present
	srcLock := filepath.Join(srcDir, "package-lock.json")
	dstLock := filepath.Join(dstDir, "package-lock.json")
	if _, err := os.Stat(srcLock); err == nil {
		if err := copyFile(srcLock, dstLock); err != nil {
			return fmt.Errorf("copy package-lock.json: %w", err)
		}
	}

	return nil
}
