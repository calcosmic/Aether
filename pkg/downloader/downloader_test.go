package downloader

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// --- Platform Detection ---

func TestGetPlatformArch_Supported(t *testing.T) {
	platform := getPlatformArch()
	if platform == nil {
		t.Fatalf("getPlatformArch returned nil on %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if platform.OS == "" || platform.Arch == "" {
		t.Errorf("got empty OS or Arch: %+v", platform)
	}
}

func TestGetPlatformArch_OS(t *testing.T) {
	platform := getPlatformArch()
	expected := runtime.GOOS
	if platform.OS != expected {
		t.Errorf("expected OS=%q, got %q", expected, platform.OS)
	}
}

func TestGetPlatformArch_ArchMapping(t *testing.T) {
	tests := []struct {
		goos   string
		goarch string
		os     string
		arch   string
		ok     bool
	}{
		{"darwin", "amd64", "darwin", "amd64", true},
		{"darwin", "arm64", "darwin", "arm64", true},
		{"linux", "amd64", "linux", "amd64", true},
		{"linux", "arm64", "linux", "arm64", true},
		{"windows", "amd64", "windows", "amd64", true},
		{"windows", "arm64", "windows", "arm64", true},
		{"freebsd", "amd64", "", "", false},
		{"linux", "386", "", "", false},
		{"darwin", "arm", "", "", false},
	}
	for _, tc := range tests {
		t.Run(fmt.Sprintf("%s_%s", tc.goos, tc.goarch), func(t *testing.T) {
			p := detectPlatform(tc.goos, tc.goarch)
			if !tc.ok {
				if p != nil {
					t.Errorf("expected nil for unsupported %s/%s, got %+v", tc.goos, tc.goarch, p)
				}
				return
			}
			if p == nil {
				t.Fatalf("expected non-nil for %s/%s", tc.goos, tc.goarch)
			}
			if p.OS != tc.os {
				t.Errorf("expected OS=%q, got %q", tc.os, p.OS)
			}
			if p.Arch != tc.arch {
				t.Errorf("expected Arch=%q, got %q", tc.arch, p.Arch)
			}
		})
	}
}

// --- URL Construction ---

func TestBuildArchiveURL(t *testing.T) {
	tests := []struct {
		version string
		os      string
		arch    string
		want    string
	}{
		{
			"1.0.0", "darwin", "amd64",
			"https://github.com/calcosmic/Aether/releases/download/v1.0.0/Aether_1.0.0_darwin_amd64.tar.gz",
		},
		{
			"2.3.0", "linux", "arm64",
			"https://github.com/calcosmic/Aether/releases/download/v2.3.0/Aether_2.3.0_linux_arm64.tar.gz",
		},
		{
			"0.1.0", "windows", "amd64",
			"https://github.com/calcosmic/Aether/releases/download/v0.1.0/Aether_0.1.0_windows_amd64.tar.gz",
		},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			got := buildArchiveURL(tc.version, tc.os, tc.arch)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBuildChecksumsURL(t *testing.T) {
	got := buildChecksumsURL("2.5.0")
	want := "https://github.com/calcosmic/Aether/releases/download/v2.5.0/checksums.txt"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestArchiveFilename(t *testing.T) {
	tests := []struct {
		version string
		os      string
		arch    string
		want    string
	}{
		{"1.0.0", "darwin", "amd64", "Aether_1.0.0_darwin_amd64.tar.gz"},
		{"1.0.0", "windows", "amd64", "Aether_1.0.0_windows_amd64.tar.gz"},
		{"2.0.0", "linux", "arm64", "Aether_2.0.0_linux_arm64.tar.gz"},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			got := archiveFilename(tc.version, tc.os, tc.arch)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestArchiveFilenameCandidatesIncludesLegacyFallback(t *testing.T) {
	got := archiveFilenameCandidates("1.0.0", "windows", "amd64")
	want := []string{
		"Aether_1.0.0_windows_amd64.tar.gz",
		"aether_v1.0.0_windows_amd64.zip",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d candidates, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("candidate %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestBuildChecksumsURLCandidatesIncludesLegacyFallback(t *testing.T) {
	got := buildChecksumsURLCandidates("2.5.0")
	want := []string{
		"https://github.com/calcosmic/Aether/releases/download/v2.5.0/checksums.txt",
		"https://github.com/calcosmic/Aether/releases/download/v2.5.0/aether_v2.5.0_checksums.txt",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d candidates, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("candidate %d = %q, want %q", i, got[i], want[i])
		}
	}
}

// --- Checksum Parsing ---

func TestParseChecksums_Found(t *testing.T) {
	content := "abc123def  Aether_1.0.0_darwin_amd64.tar.gz\n" +
		"deadbeef  Aether_1.0.0_linux_amd64.tar.gz\n"
	hash, err := parseChecksum(content, "Aether_1.0.0_darwin_amd64.tar.gz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hash != "abc123def" {
		t.Errorf("got %q, want %q", hash, "abc123def")
	}
}

func TestParseChecksum_NotFound(t *testing.T) {
	_, err := parseChecksum("abc123  some_other_file.tar.gz\n", "aether_v1.0.0_darwin_amd64.tar.gz")
	if err == nil {
		t.Fatal("expected error for missing filename")
	}
}

func TestParseChecksum_EmptyContent(t *testing.T) {
	_, err := parseChecksum("", "aether_v1.0.0_darwin_amd64.tar.gz")
	if err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestParseChecksum_MultipleSpaces(t *testing.T) {
	// goreleaser uses two-space separator
	content := "abcdef123456  aether_v1.0.0_darwin_amd64.tar.gz\n"
	hash, err := parseChecksum(content, "aether_v1.0.0_darwin_amd64.tar.gz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hash != "abcdef123456" {
		t.Errorf("got %q, want %q", hash, "abcdef123456")
	}
}

// --- Binary Name ---

func TestBinaryName(t *testing.T) {
	tests := []struct {
		os   string
		want string
	}{
		{"darwin", "aether"},
		{"linux", "aether"},
		{"windows", "aether.exe"},
	}
	for _, tc := range tests {
		t.Run(tc.os, func(t *testing.T) {
			got := binaryName(tc.os)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// --- Download with Redirects (mock server) ---

func TestDownloadWithRedirects_FollowsRedirects(t *testing.T) {
	var finalRequest bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !finalRequest {
			finalRequest = true
			http.Redirect(w, r, "/final", http.StatusFound)
			return
		}
		if r.URL.Path != "/final" {
			t.Errorf("expected /final, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "downloaded content")
	}))
	defer srv.Close()

	resp, err := downloadWithRedirects(srv.URL + "/start")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestDownloadWithRedirects_TooManyRedirects(t *testing.T) {
	redirectCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectCount++
		http.Redirect(w, r, "/next", http.StatusFound)
	}))
	defer srv.Close()

	_, err := downloadWithRedirects(srv.URL + "/start")
	if err == nil {
		t.Fatal("expected error for too many redirects")
	}
	if redirectCount <= 1 {
		t.Errorf("expected multiple redirects, got %d", redirectCount)
	}
}

func TestDownloadWithRedirects_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := downloadWithRedirects(srv.URL + "/missing")
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

// --- Download Text ---

func TestDownloadText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hello world")
	}))
	defer srv.Close()

	body, err := downloadText(srv.URL + "/file.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body != "hello world" {
		t.Errorf("got %q, want %q", body, "hello world")
	}
}

// --- DownloadAndHash ---

func TestDownloadAndHash_ComputesSHA256(t *testing.T) {
	content := "test binary content"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, content)
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "download.tmp")

	result, err := downloadAndHash(srv.URL+"/file.bin", tmpPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify hash matches expected SHA-256 of "test binary content"
	expectedHash := "56681959d2de970a2dbee51710bb02862bec0a603b725443b92063c02b5f0a0c"
	if result.Hash != expectedHash {
		t.Errorf("hash mismatch: got %q, want %q", result.Hash, expectedHash)
	}

	// Verify file was written
	data, err := os.ReadFile(tmpPath)
	if err != nil {
		t.Fatalf("failed to read temp file: %v", err)
	}
	if string(data) != content {
		t.Errorf("file content mismatch: got %q", string(data))
	}
}

// --- Supported Platforms ---

func TestIsSupportedPlatform(t *testing.T) {
	tests := []struct {
		goos   string
		goarch string
		want   bool
	}{
		{"darwin", "amd64", true},
		{"darwin", "arm64", true},
		{"linux", "amd64", true},
		{"linux", "arm64", true},
		{"windows", "amd64", true},
		{"windows", "arm64", true},
		{"freebsd", "amd64", false},
		{"linux", "386", false},
		{"linux", "riscv64", false},
	}
	for _, tc := range tests {
		t.Run(fmt.Sprintf("%s_%s", tc.goos, tc.goarch), func(t *testing.T) {
			got := isSupportedPlatform(tc.goos, tc.goarch)
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// --- ParseChecksumsStream (bufio scanner version for large files) ---

func TestParseChecksum_Stream(t *testing.T) {
	content := "abc123  aether_v1.0.0_darwin_amd64.tar.gz\n" +
		"deadbeef  aether_v1.0.0_linux_amd64.tar.gz\n" +
		"cafebabe  aether_v1.0.0_windows_amd64.zip\n"

	scanner := bufio.NewScanner(strings.NewReader(content))
	hash, err := parseChecksumFromScanner(scanner, "aether_v1.0.0_linux_amd64.tar.gz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hash != "deadbeef" {
		t.Errorf("got %q, want %q", hash, "deadbeef")
	}
}

// --- ZIP Extraction ---

// createTestZip creates a ZIP archive containing a single binary file
// nested inside a top-level directory (mimicking goreleaser output).
func createTestZip(t *testing.T, zipPath, topDir, binName, content string) {
	t.Helper()
	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	defer zipFile.Close()

	w := zip.NewWriter(zipFile)
	// Add directory entry for the top-level dir
	if _, err := w.Create(topDir + "/"); err != nil {
		t.Fatalf("create dir entry: %v", err)
	}
	// Add the binary file inside the top-level dir
	f, err := w.Create(topDir + "/" + binName)
	if err != nil {
		t.Fatalf("create file entry: %v", err)
	}
	if _, err := f.Write([]byte(content)); err != nil {
		t.Fatalf("write file entry: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
}

func TestExtractZipImpl_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "test.zip")
	stageDir := filepath.Join(tmpDir, "stage")
	destDir := filepath.Join(tmpDir, "dest")
	binName := "aether"
	binContent := "fake binary content"

	if err := os.MkdirAll(stageDir, 0755); err != nil {
		t.Fatalf("mkdir stage: %v", err)
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("mkdir dest: %v", err)
	}

	createTestZip(t, zipPath, "aether_v1.0.0_darwin_amd64", binName, binContent)

	err := extractZipImpl(zipPath, stageDir, destDir, binName)
	if err != nil {
		t.Fatalf("extractZipImpl returned error: %v", err)
	}

	// Verify the binary was moved to destDir
	destBin := filepath.Join(destDir, binName)
	data, err := os.ReadFile(destBin)
	if err != nil {
		t.Fatalf("read dest binary: %v", err)
	}
	if string(data) != binContent {
		t.Errorf("content mismatch: got %q, want %q", string(data), binContent)
	}
}

func TestExtractZipImpl_BinaryNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "test.zip")
	stageDir := filepath.Join(tmpDir, "stage")
	destDir := filepath.Join(tmpDir, "dest")

	if err := os.MkdirAll(stageDir, 0755); err != nil {
		t.Fatalf("mkdir stage: %v", err)
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("mkdir dest: %v", err)
	}

	// Create a zip with a different binary name
	createTestZip(t, zipPath, "aether_v1.0.0_darwin_amd64", "other.exe", "content")

	err := extractZipImpl(zipPath, stageDir, destDir, "aether")
	if err == nil {
		t.Fatal("expected error when binary not found")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
}

func TestExtractZipImpl_BadArchive(t *testing.T) {
	tmpDir := t.TempDir()
	badZip := filepath.Join(tmpDir, "bad.zip")
	stageDir := filepath.Join(tmpDir, "stage")
	destDir := filepath.Join(tmpDir, "dest")

	if err := os.WriteFile(badZip, []byte("not a zip file"), 0644); err != nil {
		t.Fatalf("write bad zip: %v", err)
	}
	if err := os.MkdirAll(stageDir, 0755); err != nil {
		t.Fatalf("mkdir stage: %v", err)
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("mkdir dest: %v", err)
	}

	err := extractZipImpl(badZip, stageDir, destDir, "aether")
	if err == nil {
		t.Fatal("expected error for invalid zip archive")
	}
}

func TestExtractZipImpl_WithSubdirs(t *testing.T) {
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "test.zip")
	stageDir := filepath.Join(tmpDir, "stage")
	destDir := filepath.Join(tmpDir, "dest")
	binName := "aether"
	binContent := "nested binary"

	if err := os.MkdirAll(stageDir, 0755); err != nil {
		t.Fatalf("mkdir stage: %v", err)
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("mkdir dest: %v", err)
	}

	// Create a zip with the binary nested in a subdirectory
	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	defer zipFile.Close()

	w := zip.NewWriter(zipFile)
	// top-level dir -> subdir -> binary
	f, err := w.Create("aether_v1.0.0_linux_amd64/subdir/" + binName)
	if err != nil {
		t.Fatalf("create nested file: %v", err)
	}
	if _, err := f.Write([]byte(binContent)); err != nil {
		t.Fatalf("write nested file: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}

	err = extractZipImpl(zipPath, stageDir, destDir, binName)
	if err != nil {
		t.Fatalf("extractZipImpl nested: %v", err)
	}

	destBin := filepath.Join(destDir, binName)
	data, err := os.ReadFile(destBin)
	if err != nil {
		t.Fatalf("read dest binary: %v", err)
	}
	if string(data) != binContent {
		t.Errorf("content mismatch: got %q, want %q", string(data), binContent)
	}
}

func createTestTarGz(t *testing.T, archivePath, topDir, binName, content string, typeflag byte) {
	t.Helper()
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create tar.gz: %v", err)
	}
	defer file.Close()

	gz := gzip.NewWriter(file)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()

	dirHeader := &tar.Header{Name: topDir + "/", Typeflag: tar.TypeDir, Mode: 0755}
	if err := tw.WriteHeader(dirHeader); err != nil {
		t.Fatalf("write dir header: %v", err)
	}
	header := &tar.Header{
		Name:     topDir + "/" + binName,
		Typeflag: typeflag,
		Mode:     0755,
		Size:     int64(len(content)),
	}
	if err := tw.WriteHeader(header); err != nil {
		t.Fatalf("write file header: %v", err)
	}
	if _, err := tw.Write([]byte(content)); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func TestExtractTarGzImpl_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test.tar.gz")
	stageDir := filepath.Join(tmpDir, "stage")
	destDir := filepath.Join(tmpDir, "dest")
	binContent := "fake binary content"

	if err := os.MkdirAll(stageDir, 0755); err != nil {
		t.Fatalf("mkdir stage: %v", err)
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("mkdir dest: %v", err)
	}

	createTestTarGz(t, archivePath, "aether_v1.0.0_darwin_arm64", "aether", binContent, tar.TypeReg)

	if err := extractTarGzImpl(archivePath, stageDir, destDir, "aether"); err != nil {
		t.Fatalf("extractTarGzImpl returned error: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(destDir, "aether"))
	if err != nil {
		t.Fatalf("read dest binary: %v", err)
	}
	if string(data) != binContent {
		t.Errorf("content mismatch: got %q, want %q", string(data), binContent)
	}
}

func TestExtractTarGzImpl_AcceptsAlternateRegularFileType(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test.tar.gz")
	stageDir := filepath.Join(tmpDir, "stage")
	destDir := filepath.Join(tmpDir, "dest")
	binContent := "fake binary content"

	if err := os.MkdirAll(stageDir, 0755); err != nil {
		t.Fatalf("mkdir stage: %v", err)
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("mkdir dest: %v", err)
	}

	createTestTarGz(t, archivePath, "aether_v1.0.0_darwin_arm64", "aether", binContent, tar.TypeRegA)

	if err := extractTarGzImpl(archivePath, stageDir, destDir, "aether"); err != nil {
		t.Fatalf("extractTarGzImpl TypeRegA returned error: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(destDir, "aether"))
	if err != nil {
		t.Fatalf("read dest binary: %v", err)
	}
	if string(data) != binContent {
		t.Errorf("content mismatch: got %q, want %q", string(data), binContent)
	}
}
