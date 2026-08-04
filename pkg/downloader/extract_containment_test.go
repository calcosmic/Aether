package downloader

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTarGz builds a .tar.gz whose entries are exactly the given names.
// Entry names are deliberately written raw so traversal sequences survive.
func writeTarGz(t *testing.T, path string, entries map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()

	for name, body := range entries {
		if err := tw.WriteHeader(&tar.Header{
			Name:     name,
			Mode:     0644,
			Size:     int64(len(body)),
			Typeflag: tar.TypeReg,
		}); err != nil {
			t.Fatalf("write header %q: %v", name, err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatalf("write body %q: %v", name, err)
		}
	}
}

func writeZip(t *testing.T, path string, entries map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	defer zw.Close()
	for name, body := range entries {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate})
		if err != nil {
			t.Fatalf("create zip entry %q: %v", name, err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatalf("write zip body %q: %v", name, err)
		}
	}
}

// TestExtractRejectsPathTraversalEntries is the WP-10a gate. Extraction runs
// during `aether update` — a command users are told to run — so an archive
// entry must never be able to write outside the staging directory.
func TestExtractRejectsPathTraversalEntries(t *testing.T) {
	// The leading component is stripped by the extractor (GoReleaser layout),
	// so each malicious name carries a prefix that survives stripping.
	malicious := []struct {
		name  string
		entry string
	}{
		{"parent traversal", "prefix/../../escaped.txt"},
		{"deep traversal", "prefix/../../../../../../tmp/escaped.txt"},
		{"absolute path", "prefix//etc/escaped.txt"},
	}

	for _, tc := range malicious {
		t.Run(tc.name, func(t *testing.T) {
			for _, kind := range []string{"tar", "zip"} {
				t.Run(kind, func(t *testing.T) {
					root := t.TempDir()
					stageDir := filepath.Join(root, "stage")
					destDir := filepath.Join(root, "dest")
					for _, d := range []string{stageDir, destDir} {
						if err := os.MkdirAll(d, 0755); err != nil {
							t.Fatalf("mkdir %s: %v", d, err)
						}
					}
					archive := filepath.Join(root, "archive."+kind)
					entries := map[string]string{tc.entry: "pwned"}
					if kind == "tar" {
						writeTarGz(t, archive, entries)
					} else {
						writeZip(t, archive, entries)
					}

					var err error
					if kind == "tar" {
						err = extractTarGzImpl(archive, stageDir, destDir, "aether")
					} else {
						err = extractZipImpl(archive, stageDir, destDir, "aether")
					}
					if err == nil {
						t.Fatalf("%s entry %q was accepted; extraction must reject it", kind, tc.entry)
					}
					if !strings.Contains(err.Error(), "rejected") {
						t.Fatalf("%s entry %q failed for the wrong reason: %v", kind, tc.entry, err)
					}

					// Nothing may exist outside the staging directory.
					assertNothingEscaped(t, root, stageDir)
				})
			}
		})
	}
}

// assertNothingEscaped walks the sandbox root and fails if any file landed
// outside the staging directory (dest is allowed but must stay empty here).
func assertNothingEscaped(t *testing.T, root, stageDir string) {
	t.Helper()
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasPrefix(path, stageDir+string(os.PathSeparator)) {
			return nil
		}
		if strings.HasSuffix(path, ".tar") || strings.HasSuffix(path, ".zip") {
			return nil // the archive fixture itself
		}
		t.Errorf("file escaped the staging directory: %s", path)
		return nil
	})
	if err != nil {
		t.Fatalf("walk sandbox: %v", err)
	}
}

// A well-formed archive must still extract — containment cannot break the
// normal release path.
func TestExtractAcceptsWellFormedArchive(t *testing.T) {
	for _, kind := range []string{"tar", "zip"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			stageDir := filepath.Join(root, "stage")
			destDir := filepath.Join(root, "dest")
			for _, d := range []string{stageDir, destDir} {
				if err := os.MkdirAll(d, 0755); err != nil {
					t.Fatalf("mkdir %s: %v", d, err)
				}
			}
			archive := filepath.Join(root, "archive."+kind)
			entries := map[string]string{
				"aether_1.0.0_darwin_arm64/aether":  "#!/bin/sh\necho aether\n",
				"aether_1.0.0_darwin_arm64/LICENSE": "MIT",
			}
			if kind == "tar" {
				writeTarGz(t, archive, entries)
			} else {
				writeZip(t, archive, entries)
			}

			var err error
			if kind == "tar" {
				err = extractTarGzImpl(archive, stageDir, destDir, "aether")
			} else {
				err = extractZipImpl(archive, stageDir, destDir, "aether")
			}
			if err != nil {
				t.Fatalf("well-formed %s archive rejected: %v", kind, err)
			}
			if _, statErr := os.Stat(filepath.Join(destDir, "aether")); statErr != nil {
				t.Fatalf("binary not delivered to destDir: %v", statErr)
			}
		})
	}
}

func TestContainedExtractPathRejectsEscapes(t *testing.T) {
	stage := t.TempDir()
	for _, name := range []string{"../evil", "../../evil", "/etc/passwd", "a/../../evil", "", `C:\evil`} {
		if _, err := containedExtractPath(stage, name); err == nil {
			t.Errorf("containedExtractPath accepted escaping entry %q", name)
		}
	}
	for _, name := range []string{"aether", "dir/aether", "a/b/../c"} {
		if _, err := containedExtractPath(stage, name); err != nil {
			t.Errorf("containedExtractPath rejected legitimate entry %q: %v", name, err)
		}
	}
}
