package learn

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// exportPatterns mirrors cmd/security_cmds.go secretPatterns for use in pkg/
// without importing cmd/. Used by privacyScanForExport to detect secrets in
// export content.
var exportSecretPatterns = []struct {
	name    string
	pattern *regexp.Regexp
}{
	{"api_key", regexp.MustCompile(`(?i)sk-[a-zA-Z0-9]{20,}`)},
	{"api_key_prefix", regexp.MustCompile(`(?i)key-[a-zA-Z0-9]{10,}`)},
	{"token_prefix", regexp.MustCompile(`(?i)token-[a-zA-Z0-9]{10,}`)},
	{"bearer_token", regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9\-._~+/]+=*`)},
	{"private_key_rsa", regexp.MustCompile(`-----BEGIN\s+RSA\s+PRIVATE\s+KEY-----`)},
	{"private_key_ec", regexp.MustCompile(`-----BEGIN\s+EC\s+PRIVATE\s+KEY-----`)},
	{"private_key_openssh", regexp.MustCompile(`-----BEGIN\s+OPENSSH\s+PRIVATE\s+KEY-----`)},
	{"private_key_generic", regexp.MustCompile(`-----BEGIN\s+PRIVATE\s+KEY-----`)},
	{"password_assignment", regexp.MustCompile(`(?i)(?:password|passwd)\s*=\s*['"][^'"]{8,}['"]`)},
}

// exportHomePathPattern matches absolute paths starting with /Users/, /home/, or ~.
var exportHomePathPattern = regexp.MustCompile(`(?:/Users/[^/\s"']+|/home/[^/\s"']+|~[^/\s"']*)[/[^\s"']]*`)

// privacyScanForExport scans content for secrets and home directory paths.
// Mirrors cmd/security_cmds.go privacyScan but lives in pkg/ to avoid cmd/ imports.
// Secrets trigger a block; home paths are redacted.
func privacyScanForExport(content string) PrivacyScanResult {
	var findings []string
	for _, sp := range exportSecretPatterns {
		if sp.pattern.MatchString(content) {
			findings = append(findings, fmt.Sprintf("secret pattern matched: %s", sp.name))
		}
	}
	if len(findings) > 0 {
		return PrivacyScanResult{
			Blocked:  true,
			Findings: findings,
		}
	}

	clean := exportHomePathPattern.ReplaceAllString(content, "[REDACTED_PATH]")
	return PrivacyScanResult{
		Blocked: false,
		Clean:   clean,
	}
}

// ExportManifest is the top-level structure of a learning pack file.
type ExportManifest struct {
	SourceRepo    string    `json:"source_repo"`
	ExportedAt    string    `json:"exported_at"`
	EntryCount    int       `json:"entry_count"`
	Entries       []Entry   `json:"entries"`
	RedactionReport []string `json:"redaction_report,omitempty"`
}

// ExportPack generates a portable learning pack from the colony store.
// Blocked entries (containing secrets) are skipped and reported.
// Home directory paths are redacted from entry content.
// Returns the output path, redaction report, and any error.
func ExportPack(colonyStore *ColonyStore, outputPath string) (string, []string, error) {
	entries, err := colonyStore.List(EntryFilter{})
	if err != nil {
		return "", nil, fmt.Errorf("learn: export: list entries: %w", err)
	}

	var exported []Entry
	var report []string

	for _, entry := range entries {
		scanResult := privacyScanForExport(entry.Content)
		if scanResult.Blocked {
			report = append(report, fmt.Sprintf("Entry %s: blocked (contains secrets)", entry.ID))
			continue
		}

		cleanEntry := entry
		if scanResult.Clean != entry.Content {
			cleanEntry.Content = scanResult.Clean
			cleanEntry.Redacted = true
			report = append(report, fmt.Sprintf("Entry %s: paths redacted", entry.ID))
		}
		exported = append(exported, cleanEntry)
	}

	manifest := ExportManifest{
		ExportedAt:       time.Now().UTC().Format(time.RFC3339),
		EntryCount:       len(exported),
		Entries:          exported,
		RedactionReport:  report,
	}

	if exported == nil {
		exported = []Entry{}
	}

	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", nil, fmt.Errorf("learn: export: marshal manifest: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return "", nil, fmt.Errorf("learn: export: create output dir: %w", err)
	}

	if err := os.WriteFile(outputPath, append(encoded, '\n'), 0644); err != nil {
		return "", nil, fmt.Errorf("learn: export: write manifest: %w", err)
	}

	return outputPath, report, nil
}

// ImportPreview reads a learning pack and returns the entries and redaction
// report without applying them to the store.
func ImportPreview(packPath string) ([]Entry, []string, error) {
	raw, err := os.ReadFile(packPath)
	if err != nil {
		return nil, nil, fmt.Errorf("learn: import preview: read pack: %w", err)
	}

	var manifest ExportManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, nil, fmt.Errorf("learn: import preview: parse manifest: %w", err)
	}

	if manifest.Entries == nil {
		manifest.Entries = []Entry{}
	}

	return manifest.Entries, manifest.RedactionReport, nil
}

// ImportPack applies entries from a learning pack to the colony store.
// Returns the number of entries imported.
func ImportPack(colonyStore *ColonyStore, packPath string) (int, error) {
	entries, _, err := ImportPreview(packPath)
	if err != nil {
		return 0, err
	}

	imported := 0
	for _, entry := range entries {
		// Clear the ID so ColonyStore assigns a new one (avoid collisions)
		entry.ID = ""
		if err := colonyStore.Add(entry); err != nil {
			return imported, fmt.Errorf("learn: import: add entry: %w", err)
		}
		imported++
	}

	return imported, nil
}
