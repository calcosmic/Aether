package downloader

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// extractTarGzImpl extracts a tar.gz archive, finds the binary, and moves it to destDir.
func extractTarGzImpl(archivePath, stageDir, destDir, bin string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar read: %w", err)
		}

		// Extract files to stage dir, stripping the top-level directory
		relPath := header.Name
		// Strip leading "./" or top-level dir
		parts := strings.SplitN(relPath, "/", 2)
		if len(parts) < 2 {
			continue
		}
		relPath = parts[1]

		targetPath := filepath.Join(stageDir, relPath)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return fmt.Errorf("mkdir: %w", err)
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("mkdir parent: %w", err)
			}
			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("create file: %w", err)
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("write file: %w", err)
			}
			outFile.Close()
		}
	}

	// Find and move the binary
	foundPath := findBinaryInDir(stageDir, bin)
	if foundPath == "" {
		return fmt.Errorf("binary %q not found in archive", bin)
	}

	return os.Rename(foundPath, filepath.Join(destDir, bin))
}

// findBinaryInDir searches for a binary file named bin recursively in dir.
func findBinaryInDir(dir, bin string) string {
	var found string
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Base(path), bin) && found == "" {
			found = path
		}
		return nil
	})
	return found
}

// extractZipImpl extracts a zip archive, finds the binary, and moves it to destDir.
func extractZipImpl(archivePath, stageDir, destDir, bin string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer reader.Close()

	for _, entry := range reader.File {
		// Skip directories
		if entry.FileInfo().IsDir() {
			continue
		}

		relPath := entry.Name
		// Strip leading "./" or top-level dir
		parts := strings.SplitN(relPath, "/", 2)
		if len(parts) < 2 {
			continue
		}
		relPath = parts[1]

		targetPath := filepath.Join(stageDir, relPath)

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("mkdir parent: %w", err)
		}

		rc, err := entry.Open()
		if err != nil {
			return fmt.Errorf("open entry %s: %w", entry.Name, err)
		}

		outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, entry.Mode())
		if err != nil {
			rc.Close()
			return fmt.Errorf("create file: %w", err)
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return fmt.Errorf("write file: %w", err)
		}
	}

	// Find and move the binary
	foundPath := findBinaryInDir(stageDir, bin)
	if foundPath == "" {
		return fmt.Errorf("binary %q not found in archive", bin)
	}

	return os.Rename(foundPath, filepath.Join(destDir, bin))
}
