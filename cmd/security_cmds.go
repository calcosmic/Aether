package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

// --- check-antipattern ---

// AntipatternFinding represents a single finding from the antipattern scanner.
type AntipatternFinding struct {
	Pattern string `json:"pattern"`
	File    string `json:"file"`
	Line    int    `json:"line,omitempty"`
	Count   int    `json:"count,omitempty"`
	Message string `json:"message"`
}

var checkAntipatternCmd = &cobra.Command{
	Use:   "check-antipattern",
	Short: "Scan a file for security antipatterns",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		filePath := mustGetString(cmd, "file")
		if filePath == "" {
			return nil
		}

		// If file doesn't exist, return clean
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			outputOK(map[string]interface{}{
				"critical": []interface{}{},
				"warnings": []interface{}{},
				"clean":    true,
			})
			return nil
		}

		var criticals []AntipatternFinding
		var warnings []AntipatternFinding

		ext := strings.TrimPrefix(filepath.Ext(filePath), ".")

		content, err := os.ReadFile(filePath)
		if err != nil {
			outputError(1, fmt.Sprintf("failed to read file: %v", err), nil)
			return nil
		}
		text := string(content)
		lines := strings.Split(text, "\n")

		// Language-specific checks
		switch ext {
		case "swift":
			// didSet infinite recursion
			didSetRe := regexp.MustCompile(`(?i)didSet`)
			selfDotRe := regexp.MustCompile(`self\.`)
			for i, line := range lines {
				if didSetRe.MatchString(line) && selfDotRe.MatchString(line) {
					criticals = append(criticals, AntipatternFinding{
						Pattern: "didSet-recursion",
						File:    filePath,
						Line:    i + 1,
						Message: "Potential didSet infinite recursion - self assignment in didSet",
					})
					break
				}
			}
		case "ts", "tsx", "js", "jsx":
			// TypeScript 'any' type check (only in non-comment lines)
			anyRe := regexp.MustCompile(`\bany\b`)
			anyCount := 0
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") {
					continue
				}
				if anyRe.MatchString(line) {
					anyCount++
				}
			}
			if anyCount > 0 {
				warnings = append(warnings, AntipatternFinding{
					Pattern: "typescript-any",
					File:    filePath,
					Count:   anyCount,
					Message: fmt.Sprintf("Found %d uses of 'any' type", anyCount),
				})
			}

			// console.log in non-test files
			if !strings.Contains(filePath, ".test.") && !strings.Contains(filePath, ".spec.") {
				consoleRe := regexp.MustCompile(`console\.log`)
				consoleCount := 0
				for _, line := range lines {
					if strings.Contains(line, "//") {
						// Check if console.log appears before the comment marker
						commentIdx := strings.Index(line, "//")
						if commentIdx >= 0 && strings.Contains(line[:commentIdx], "console.log") {
							consoleCount++
						} else if commentIdx < 0 {
							if consoleRe.MatchString(line) {
								consoleCount++
							}
						}
					} else if consoleRe.MatchString(line) {
						consoleCount++
					}
				}
				if consoleCount > 0 {
					warnings = append(warnings, AntipatternFinding{
						Pattern: "console-log",
						File:    filePath,
						Count:   consoleCount,
						Message: fmt.Sprintf("Found %d console.log statements", consoleCount),
					})
				}
			}
		case "py":
			// Bare except
			exceptRe := regexp.MustCompile(`^\s*except\s*:`)
			for i, line := range lines {
				if exceptRe.MatchString(line) && !strings.Contains(line, "#") {
					warnings = append(warnings, AntipatternFinding{
						Pattern: "bare-except",
						File:    filePath,
						Line:    i + 1,
						Message: "Bare except clause - specify exception type",
					})
					break
				}
			}
		}

		// Common patterns across all languages

		// Exposed secrets check (critical)
		secretRe := regexp.MustCompile(`(?i)(api_key|apikey|secret|password|token)\s*=\s*['"][^'"]+['"]`)
		for i, line := range lines {
			if secretRe.MatchString(line) {
				lowerLine := strings.ToLower(line)
				if !strings.Contains(lowerLine, "example") &&
					!strings.Contains(lowerLine, "test") &&
					!strings.Contains(lowerLine, "mock") &&
					!strings.Contains(lowerLine, "fake") {
					criticals = append(criticals, AntipatternFinding{
						Pattern: "exposed-secret",
						File:    filePath,
						Line:    i + 1,
						Message: "Potential hardcoded secret or credential",
					})
					break
				}
			}
		}

		// TODO/FIXME check (warning)
		todoRe := regexp.MustCompile(`(TODO|FIXME|XXX|HACK)`)
		todoCount := 0
		for _, line := range lines {
			if todoRe.MatchString(line) {
				todoCount++
			}
		}
		if todoCount > 0 {
			warnings = append(warnings, AntipatternFinding{
				Pattern: "todo-comment",
				File:    filePath,
				Count:   todoCount,
				Message: fmt.Sprintf("Found %d TODO/FIXME comments", todoCount),
			})
		}

		clean := len(criticals) == 0 && len(warnings) == 0
		if criticals == nil {
			criticals = []AntipatternFinding{}
		}
		if warnings == nil {
			warnings = []AntipatternFinding{}
		}

		outputOK(map[string]interface{}{
			"critical": criticals,
			"warnings": warnings,
			"clean":    clean,
		})
		return nil
	},
}

// --- signature-scan ---

var signatureScanCmd = &cobra.Command{
	Use:   "signature-scan",
	Short: "Scan a file for a named signature pattern",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		targetFile := mustGetString(cmd, "file")
		if targetFile == "" {
			return nil
		}
		signatureName := mustGetString(cmd, "name")
		if signatureName == "" {
			return nil
		}

		// If file doesn't exist, return not found
		if _, err := os.Stat(targetFile); os.IsNotExist(err) {
			outputOK(map[string]interface{}{
				"found":     false,
				"signature": nil,
			})
			return nil
		}

		// Load signatures file
		var sigFile struct {
			Signatures []struct {
				Name                string  `json:"name"`
				Description         string  `json:"description"`
				PatternString       string  `json:"pattern_string"`
				ConfidenceThreshold float64 `json:"confidence_threshold"`
			} `json:"signatures"`
		}
		if err := store.LoadJSON("signatures.json", &sigFile); err != nil {
			outputOK(map[string]interface{}{
				"found":     false,
				"signature": nil,
			})
			return nil
		}

		// Find the named signature
		var found *struct {
			Name                string  `json:"name"`
			Description         string  `json:"description"`
			PatternString       string  `json:"pattern_string"`
			ConfidenceThreshold float64 `json:"confidence_threshold"`
		}
		for i := range sigFile.Signatures {
			if sigFile.Signatures[i].Name == signatureName {
				found = &sigFile.Signatures[i]
				break
			}
		}
		if found == nil || found.PatternString == "" {
			outputOK(map[string]interface{}{
				"found":     false,
				"signature": nil,
			})
			return nil
		}

		// Search for the pattern in the target file
		file, err := os.Open(targetFile)
		if err != nil {
			outputOK(map[string]interface{}{
				"found":     false,
				"signature": nil,
			})
			return nil
		}
		defer file.Close()

		matchCount := 0
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), found.PatternString) {
				matchCount++
			}
		}

		if matchCount > 0 {
			outputOK(map[string]interface{}{
				"found":       true,
				"signature":   found,
				"match_count": matchCount,
			})
			return nil
		}

		outputOK(map[string]interface{}{
			"found":     false,
			"signature": nil,
		})
		return nil
	},
}

// --- signature-match ---

var signatureMatchCmd = &cobra.Command{
	Use:   "signature-match",
	Short: "Match regex pattern against file content",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		targetFile := mustGetString(cmd, "file")
		if targetFile == "" {
			return nil
		}
		patternStr := mustGetString(cmd, "pattern")
		if patternStr == "" {
			return nil
		}

		// If file doesn't exist, return no match
		if _, err := os.Stat(targetFile); os.IsNotExist(err) {
			outputOK(map[string]interface{}{
				"matched": false,
				"matches": []interface{}{},
			})
			return nil
		}

		re, err := regexp.Compile(patternStr)
		if err != nil {
			outputError(1, fmt.Sprintf("invalid regex pattern: %v", err), nil)
			return nil
		}

		file, err := os.Open(targetFile)
		if err != nil {
			outputError(1, fmt.Sprintf("failed to open file: %v", err), nil)
			return nil
		}
		defer file.Close()

		type matchEntry struct {
			Line int    `json:"line"`
			Text string `json:"text"`
		}
		var matches []matchEntry

		scanner := bufio.NewScanner(file)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			if re.MatchString(line) {
				matches = append(matches, matchEntry{Line: lineNum, Text: line})
			}
		}

		if matches == nil {
			matches = []matchEntry{}
		}

		outputOK(map[string]interface{}{
			"matched": len(matches) > 0,
			"matches": matches,
		})
		return nil
	},
}

// --- privacy-scan ---

// PrivacyScanResult reports what was found during a privacy scan.
type PrivacyScanResult struct {
	Blocked  bool     `json:"blocked"`
	Clean    string   `json:"clean,omitempty"`
	Findings []string `json:"findings,omitempty"`
}

// secretPatterns defines compiled regex patterns that trigger a block when matched.
// These are compiled once at package init for performance.
var secretPatterns = []struct {
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
	{"env_file_reference", regexp.MustCompile(`(?i)(?:^|\n|\s)(?:\.env(?:\.local|\.production|\.staging|\.development)?|credentials\.json)(?:\s|\n|$)`)},
}

// homePathPattern matches absolute paths starting with /Users/, /home/, or ~
// followed by a username and optional subpath.
var homePathPattern = regexp.MustCompile(`(?:/Users/[^/\s"']+|/home/[^/\s"']+|~[^/\s"']*)[/[^\s"']]*`)

// privacyScan scans content for secrets and home directory paths.
// Secrets trigger a block (write rejected, finding logged).
// Home paths are redacted (write proceeds with paths removed).
// Returns the scan result with blocked status, cleaned content, and findings.
func privacyScan(content string) PrivacyScanResult {
	// Check secrets first (block if found, per D-10)
	var findings []string
	for _, sp := range secretPatterns {
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

	// Redact home directory paths (proceed with clean content, per D-09)
	clean := homePathPattern.ReplaceAllString(content, "[REDACTED_PATH]")
	return PrivacyScanResult{
		Blocked: false,
		Clean:   clean,
	}
}

var privacyScanCmd = &cobra.Command{
	Use:   "privacy-scan",
	Short: "Scan content for secrets and private paths",
	Long:  `Scans input content for API keys, private keys, passwords, bearer tokens, and home directory paths. Secrets cause a block; paths are redacted.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		content := mustGetString(cmd, "content")
		result := privacyScan(content)
		outputOK(result)
		return nil
	},
}

func init() {
	checkAntipatternCmd.Flags().String("file", "", "File path to scan (required)")
	signatureScanCmd.Flags().String("file", "", "Target file to scan (required)")
	signatureScanCmd.Flags().String("name", "", "Signature name to search for (required)")
	signatureMatchCmd.Flags().String("file", "", "Target file to scan (required)")
	signatureMatchCmd.Flags().String("pattern", "", "Regex pattern to match (required)")
	privacyScanCmd.Flags().String("content", "", "Content to scan (required)")

	rootCmd.AddCommand(checkAntipatternCmd)
	rootCmd.AddCommand(signatureScanCmd)
	rootCmd.AddCommand(signatureMatchCmd)
	rootCmd.AddCommand(privacyScanCmd)
}
