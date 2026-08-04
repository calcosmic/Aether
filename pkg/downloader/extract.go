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

// containedExtractPath resolves an archive-supplied entry name against the
// staging directory and proves the result stays inside it.
//
// Archive entry names are attacker-controlled in the general case, and this
// code runs during `aether update` — a command users are told to run. Without
// this proof an entry named `../../.ssh/authorized_keys` writes wherever it
// likes, because filepath.Join happily resolves the traversal.
func containedExtractPath(stageDir, entryName string) (string, error) {
	name := strings.ReplaceAll(entryName, `\`, "/")
	if name == "" {
		return "", fmt.Errorf("archive entry has an empty name")
	}
	if filepath.IsAbs(name) || strings.HasPrefix(name, "/") {
		return "", fmt.Errorf("archive entry %q uses an absolute path", entryName)
	}
	// Windows-style drive prefixes (`C:\evil`) are absolute too, but
	// filepath.IsAbs only knows the host's rules.
	if len(name) >= 2 && name[1] == ':' {
		return "", fmt.Errorf("archive entry %q uses a drive-qualified path", entryName)
	}

	absStage, err := filepath.Abs(stageDir)
	if err != nil {
		return "", fmt.Errorf("resolve staging dir: %w", err)
	}
	target := filepath.Join(absStage, filepath.FromSlash(name))
	rel, err := filepath.Rel(absStage, target)
	if err != nil {
		return "", fmt.Errorf("archive entry %q is not resolvable inside the staging directory", entryName)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("archive entry %q escapes the staging directory", entryName)
	}
	return target, nil
}

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

		// Extract files to stage dir. GoReleaser may place files either at
		// archive root or under a top-level directory, so strip that directory
		// only when one exists.
		relPath := header.Name
		parts := strings.SplitN(relPath, "/", 2)
		if len(parts) == 2 {
			relPath = parts[1]
		}
		if relPath == "" {
			continue
		}

		targetPath, err := containedExtractPath(stageDir, relPath)
		if err != nil {
			return fmt.Errorf("tar entry rejected: %w", err)
		}

		switch header.Typeflag {
		case tar.TypeSymlink, tar.TypeLink:
			// Links are never materialised: a link planted outside the staging
			// directory would let a later regular-file entry be written
			// through it, defeating the containment check above.
			continue
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
		parts := strings.SplitN(relPath, "/", 2)
		if len(parts) == 2 {
			relPath = parts[1]
		}
		if relPath == "" {
			continue
		}

		targetPath, err := containedExtractPath(stageDir, relPath)
		if err != nil {
			return fmt.Errorf("zip entry rejected: %w", err)
		}

		if entry.Mode()&os.ModeSymlink != 0 {
			continue // never materialise links; see the tar path
		}

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
