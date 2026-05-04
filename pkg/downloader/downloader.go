// Package downloader provides platform-independent binary download from GitHub Releases.
// It detects the current platform, downloads the correct archive, verifies SHA-256
// checksums, and installs the binary atomically.
package downloader

import (
	"bufio"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	// maxRedirects is the maximum number of HTTP redirect hops to follow.
	maxRedirects = 5

	// maxRetries is the maximum number of download retry attempts.
	maxRetries = 3

	// defaultDownloadTimeout is the default HTTP client timeout for downloads.
	defaultDownloadTimeout = 60 * time.Second

	// baseReleaseURL is the GitHub Releases URL template for aether downloads.
	baseReleaseURL = "https://github.com/calcosmic/Aether/releases/download/v%s"

	// defaultDestSubdir is the default subdirectory under the home dir for the binary.
	defaultDestSubdir = ".aether/bin"
)

// Platform holds detected OS and architecture in goreleaser naming convention.
type Platform struct {
	OS   string
	Arch string
}

// DownloadResult holds the outcome of a binary download operation.
type DownloadResult struct {
	Success bool
	Path    string
	Version string
}

// supportedPlatforms maps GOOS values to goreleaser OS names.
var supportedOS = map[string]string{
	"darwin":  "darwin",
	"linux":   "linux",
	"windows": "windows",
}

// supportedArch maps GOARCH values to goreleaser arch names.
var supportedArch = map[string]string{
	"amd64": "amd64",
	"arm64": "arm64",
}

// getPlatformArch detects the current platform using runtime.GOOS and
// runtime.GOARCH, mapped to goreleaser naming convention.
// Returns nil if the current platform is not supported.
func getPlatformArch() *Platform {
	return detectPlatform(runtime.GOOS, runtime.GOARCH)
}

// detectPlatform maps a GOOS/GOARCH pair to goreleaser naming.
// Returns nil if the platform is not supported.
func detectPlatform(goos, goarch string) *Platform {
	osName, ok := supportedOS[goos]
	if !ok {
		return nil
	}
	archName, ok := supportedArch[goarch]
	if !ok {
		return nil
	}
	return &Platform{OS: osName, Arch: archName}
}

// isSupportedPlatform checks if a GOOS/GOARCH pair is in the supported set.
func isSupportedPlatform(goos, goarch string) bool {
	return detectPlatform(goos, goarch) != nil
}

// archiveFilename returns the primary GoReleaser archive filename for the given version and platform.
func archiveFilename(version, goos, goarch string) string {
	return fmt.Sprintf("Aether_%s_%s_%s.tar.gz", version, goos, goarch)
}

func legacyArchiveFilename(version, goos, goarch string) string {
	ext := ".tar.gz"
	if goos == "windows" {
		ext = ".zip"
	}
	return fmt.Sprintf("aether_v%s_%s_%s%s", version, goos, goarch, ext)
}

func archiveFilenameCandidates(version, goos, goarch string) []string {
	primary := archiveFilename(version, goos, goarch)
	legacy := legacyArchiveFilename(version, goos, goarch)
	if primary == legacy {
		return []string{primary}
	}
	return []string{primary, legacy}
}

// buildArchiveURL constructs the full download URL for the platform archive.
func buildArchiveURL(version, goos, goarch string) string {
	return fmt.Sprintf("%s/%s", fmt.Sprintf(baseReleaseURL, version), archiveFilename(version, goos, goarch))
}

func buildArchiveURLForFile(version, filename string) string {
	return fmt.Sprintf("%s/%s", fmt.Sprintf(baseReleaseURL, version), filename)
}

// buildChecksumsURL constructs the URL for the checksums.txt file.
func buildChecksumsURL(version string) string {
	return fmt.Sprintf("%s/checksums.txt", fmt.Sprintf(baseReleaseURL, version))
}

func buildLegacyChecksumsURL(version string) string {
	return fmt.Sprintf("%s/aether_v%s_checksums.txt", fmt.Sprintf(baseReleaseURL, version), version)
}

func buildChecksumsURLCandidates(version string) []string {
	return []string{buildChecksumsURL(version), buildLegacyChecksumsURL(version)}
}

// binaryName returns the binary name for the given OS.
func binaryName(goos string) string {
	if goos == "windows" {
		return "aether.exe"
	}
	return "aether"
}

// parseChecksum parses a checksums.txt content string to find the SHA-256 hash
// for a specific filename. Format: "<hash>  <filename>" (two-space separator).
func parseChecksum(content, filename string) (string, error) {
	return parseChecksumFromScanner(bufio.NewScanner(strings.NewReader(content)), filename)
}

// parseChecksumFromScanner scans through checksum lines to find the hash for a filename.
func parseChecksumFromScanner(scanner *bufio.Scanner, filename string) (string, error) {
	for scanner.Scan() {
		line := scanner.Text()
		// goreleaser format: "<sha256_hex>  <filename>"
		parts := strings.SplitN(line, "  ", 2)
		if len(parts) != 2 {
			continue
		}
		if parts[1] == filename {
			return strings.TrimSpace(parts[0]), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("reading checksums: %w", err)
	}
	return "", fmt.Errorf("checksum not found for %q", filename)
}

// downloadWithRedirects performs an HTTP GET following up to maxRedirects 302 redirects.
func downloadWithRedirects(url string) (*http.Response, error) {
	return downloadWithRedirectsAndClient(url, maxRedirects, &http.Client{
		Timeout: defaultDownloadTimeout,
		// Go's default http.Client follows redirects automatically,
		// but we limit the count explicitly.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("too many redirects (>%d)", maxRedirects)
			}
			return nil
		},
	})
}

// downloadWithRedirectsAndClient performs an HTTP GET with a custom client and redirect limit.
func downloadWithRedirectsAndClient(url string, _ int, client *http.Client) (*http.Response, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}
	return resp, nil
}

// downloadText downloads a URL and returns the full response body as a string.
func downloadText(url string) (string, error) {
	resp, err := downloadWithRedirects(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response body: %w", err)
	}
	return string(data), nil
}

func downloadFirstText(urls []string) (string, string, error) {
	var failures []string
	for _, url := range urls {
		content, err := downloadText(url)
		if err == nil {
			return content, url, nil
		}
		failures = append(failures, err.Error())
	}
	return "", "", errors.New(strings.Join(failures, "; "))
}

// HashResult holds the computed SHA-256 hash and path of a downloaded file.
type HashResult struct {
	Hash    string
	TmpPath string
}

// downloadAndHash downloads a URL to a temp file while computing SHA-256 hash.
func downloadAndHash(url, tmpPath string) (*HashResult, error) {
	resp, err := downloadWithRetries(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	outFile, err := os.Create(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	defer outFile.Close()

	hash := sha256.New()
	multiWriter := io.MultiWriter(outFile, hash)

	if _, err := io.Copy(multiWriter, resp.Body); err != nil {
		return nil, fmt.Errorf("writing download: %w", err)
	}

	if err := outFile.Sync(); err != nil {
		return nil, fmt.Errorf("sync temp file: %w", err)
	}

	return &HashResult{
		Hash:    fmt.Sprintf("%x", hash.Sum(nil)),
		TmpPath: tmpPath,
	}, nil
}

// downloadWithRetries downloads a URL with up to maxRetries attempts and exponential backoff.
func downloadWithRetries(url string) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		resp, err := downloadWithRedirects(url)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if attempt < maxRetries-1 {
			time.Sleep(time.Duration(1<<uint(attempt)) * time.Second)
		}
	}
	return nil, fmt.Errorf("after %d retries: %w", maxRetries, lastErr)
}

// DownloadBinary downloads and installs the aether binary for the current platform.
//
// Parameters:
//   - version: semver string (e.g. "1.0.0"), without "v" prefix
//   - destDir: directory to install the binary into (e.g. "~/.aether/bin")
//
// Returns a DownloadResult with success status and installed path.
func DownloadBinary(version, destDir string) (*DownloadResult, error) {
	// 1. Platform detection
	platform := getPlatformArch()
	if platform == nil {
		return nil, fmt.Errorf(
			"unsupported platform: %s/%s (supported: darwin/linux/windows + amd64/arm64)",
			runtime.GOOS, runtime.GOARCH,
		)
	}

	// 2. Download checksums. Current releases publish checksums.txt; older
	// releases used aether_v<version>_checksums.txt.
	checksumsContent, _, err := downloadFirstText(buildChecksumsURLCandidates(version))
	if err != nil {
		return nil, fmt.Errorf("failed to download checksums for v%s: %w", version, err)
	}

	// 3. Find the archive name used by this release. Current GoReleaser
	// archives are Aether_<version>_<os>_<arch>.tar.gz; legacy releases used
	// aether_v<version>_<os>_<arch> with zip archives on Windows.
	var archiveFile string
	var expectedHash string
	var checksumFailures []string
	for _, candidate := range archiveFilenameCandidates(version, platform.OS, platform.Arch) {
		expectedHash, err = parseChecksum(checksumsContent, candidate)
		if err == nil {
			archiveFile = candidate
			break
		}
		checksumFailures = append(checksumFailures, err.Error())
	}
	if archiveFile == "" {
		return nil, fmt.Errorf("checksum for supported archive names not found in checksums.txt: %s", strings.Join(checksumFailures, "; "))
	}
	archiveURL := buildArchiveURLForFile(version, archiveFile)

	// 4. Download archive to temp file, computing hash during stream
	tmpDir, err := os.MkdirTemp("", "aether-download-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpArchive := filepath.Join(tmpDir, archiveFile)
	hashResult, err := downloadAndHash(archiveURL, tmpArchive)
	if err != nil {
		return nil, fmt.Errorf("failed to download %s: %w", archiveFile, err)
	}

	// 5. Verify checksum
	if hashResult.Hash != expectedHash {
		return nil, fmt.Errorf(
			"checksum mismatch for %s: expected %s, got %s",
			archiveFile, expectedHash, hashResult.Hash,
		)
	}

	// 6. Extract binary from archive
	binName := binaryName(platform.OS)
	extractDir := filepath.Join(tmpDir, "extract")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return nil, fmt.Errorf("create extract dir: %w", err)
	}

	if err := extractArchive(tmpArchive, extractDir, platform.OS); err != nil {
		return nil, fmt.Errorf("failed to extract archive: %w", err)
	}

	// 7. Atomic install: ensure dest dir exists, rename binary into place
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("create dest dir %s: %w", destDir, err)
	}

	destPath := filepath.Join(destDir, binName)
	extractedBinary := filepath.Join(extractDir, binName)

	if err := atomicRename(extractedBinary, destPath); err != nil {
		return nil, fmt.Errorf("failed to install binary to %s: %w", destPath, err)
	}

	return &DownloadResult{
		Success: true,
		Path:    destPath,
		Version: version,
	}, nil
}

// extractArchive extracts a tar.gz or zip archive, stripping the top-level directory.
// It extracts only the aether binary file.
func extractArchive(archivePath, destDir, goos string) error {
	bin := binaryName(goos)

	// Use a simple extraction approach: extract all, then find the binary
	switch {
	case strings.HasSuffix(archivePath, ".tar.gz"), strings.HasSuffix(archivePath, ".tgz"):
		return extractTarGz(archivePath, destDir, bin)
	case strings.HasSuffix(archivePath, ".zip"):
		return extractZip(archivePath, destDir, bin)
	default:
		return fmt.Errorf("unsupported archive format: %s", filepath.Ext(archivePath))
	}
}

// extractTarGz extracts a tar.gz archive, finding and moving the binary to destDir.
func extractTarGz(archivePath, destDir, bin string) error {
	// We extract to a staging directory first, then move the binary
	stageDir := destDir + "_stage"
	if err := os.MkdirAll(stageDir, 0755); err != nil {
		return fmt.Errorf("create stage dir: %w", err)
	}
	defer os.RemoveAll(stageDir)

	// Use the archive/tar and compress/gzip packages for extraction
	return extractTarGzImpl(archivePath, stageDir, destDir, bin)
}

// extractZip extracts a zip archive, finding and moving the binary to destDir.
func extractZip(archivePath, destDir, bin string) error {
	stageDir := destDir + "_stage"
	if err := os.MkdirAll(stageDir, 0755); err != nil {
		return fmt.Errorf("create stage dir: %w", err)
	}
	defer os.RemoveAll(stageDir)

	return extractZipImpl(archivePath, stageDir, destDir, bin)
}

// atomicRename renames src to dest, making it executable on non-Windows platforms.
func atomicRename(src, dest string) error {
	if err := os.Rename(src, dest); err != nil {
		// If rename fails (cross-device), fall back to copy + remove
		if err := copyFileAtomic(src, dest); err != nil {
			return fmt.Errorf("rename and copy both failed: %w", err)
		}
	}

	// Set executable permission on Unix
	if runtime.GOOS != "windows" {
		if err := os.Chmod(dest, 0755); err != nil {
			return fmt.Errorf("chmod: %w", err)
		}
	}

	return nil
}

// copyFileAtomic copies a file from src to dest and removes src.
// This is a fallback when os.Rename fails (e.g., cross-device link).
func copyFileAtomic(src, dest string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	destFile, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, srcFile); err != nil {
		return err
	}

	if err := destFile.Sync(); err != nil {
		return err
	}

	return os.Remove(src)
}

// DefaultDestSubdir returns the default subdirectory under home for the binary.
func DefaultDestSubdir() string {
	return defaultDestSubdir
}

// SupportedPlatforms returns a list of supported platform strings for error messages.
func SupportedPlatforms() []string {
	osList := []string{}
	for goos := range supportedOS {
		for goarch := range supportedArch {
			osList = append(osList, fmt.Sprintf("%s/%s", goos, goarch))
		}
	}
	return osList
}

// IsVersionNotFoundErr checks if an error indicates the release version was not found (404).
func IsVersionNotFoundErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "HTTP 404")
}

// ErrUnsupportedPlatform is returned when the current platform is not supported.
var ErrUnsupportedPlatform = errors.New("unsupported platform")
