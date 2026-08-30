package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
	"github.com/spf13/cobra"
)

// outputOK writes a JSON success envelope to stdout:
//
//	{"ok":true,"result":<result>}
//
// This matches the shell's json_ok() function format for playbook compatibility.
func outputOK(result interface{}) {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		outputError(2, fmt.Sprintf("failed to marshal command result: %v", err), nil)
		return
	}
	fmt.Fprintf(stdout, "{\"ok\":true,\"result\":%s}\n", string(resultJSON))
}

// outputError writes a JSON error envelope to stderr:
//
//	{"ok":false,"error":"<message>","code":<code>}
//
// This matches the shell's json_err() function format for playbook compatibility.
func outputError(code int, message string, details interface{}) {
	markRenderedCommandError(code)
	if shouldRenderVisualOutput(stderr) {
		// Through writeVisualOutput, not fmt.Fprint: that is where command
		// naming is translated for slash-command platforms, and an error is
		// the moment a user most needs a command they can actually type.
		writeVisualOutput(stderr, renderVisualError(message, details))
		return
	}
	envelope := struct {
		OK      bool        `json:"ok"`
		Error   string      `json:"error"`
		Code    int         `json:"code"`
		Details interface{} `json:"details,omitempty"`
	}{
		OK:    false,
		Error: message,
		Code:  code,
	}
	if details != nil {
		envelope.Details = details
	}
	payload, err := json.Marshal(envelope)
	if err != nil {
		msgJSON, _ := json.Marshal(message)
		fmt.Fprintf(stderr, "{\"ok\":false,\"error\":%s,\"code\":%d}\n", string(msgJSON), code)
		return
	}
	fmt.Fprintf(stderr, "%s\n", string(payload))
}

// outputErrorMessage is a convenience wrapper for outputError with code 1.
func outputErrorMessage(message string) {
	outputError(1, message, nil)
}

// mustGetString retrieves a required string flag, calling outputError and
// exiting if the flag is missing or empty.
func mustGetString(cmd *cobra.Command, flag string) string {
	val, err := cmd.Flags().GetString(flag)
	if err != nil {
		outputError(1, fmt.Sprintf("missing flag --%s", flag), nil)
		return ""
	}
	if val == "" {
		outputError(1, fmt.Sprintf("flag --%s is required", flag), nil)
		return ""
	}
	return val
}

func mustGetStringCompat(cmd *cobra.Command, args []string, flag string, positional int) string {
	val, _ := cmd.Flags().GetString(flag)
	if strings.TrimSpace(val) != "" {
		return val
	}
	if positional >= 0 && len(args) > positional {
		val = strings.TrimSpace(args[positional])
		if val != "" {
			return val
		}
	}
	outputError(1, fmt.Sprintf("flag --%s is required", flag), nil)
	return ""
}

func mustGetStringCompatOptional(cmd *cobra.Command, flag string) string {
	val, _ := cmd.Flags().GetString(flag)
	return strings.TrimSpace(val)
}

// mustGetInt retrieves a required int flag, calling outputError and
// exiting if the flag is missing.
func mustGetInt(cmd *cobra.Command, flag string) int {
	val, err := cmd.Flags().GetInt(flag)
	if err != nil {
		outputError(1, fmt.Sprintf("missing flag --%s", flag), nil)
		return 0
	}
	return val
}

func optionalArg(args []string, index int) string {
	if index >= 0 && len(args) > index {
		return strings.TrimSpace(args[index])
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// mustGetBool retrieves a required bool flag, calling outputError and
// returning false if the flag is missing.
func mustGetBool(cmd *cobra.Command, flag string) bool {
	val, err := cmd.Flags().GetBool(flag)
	if err != nil {
		outputError(1, fmt.Sprintf("missing flag --%s", flag), nil)
		return false
	}
	return val
}

// mustGetFloat64 retrieves a required float64 flag, calling outputError and
// returning 0 if the flag is missing.
func mustGetFloat64(cmd *cobra.Command, flag string) float64 {
	val, err := cmd.Flags().GetFloat64(flag)
	if err != nil {
		outputError(1, fmt.Sprintf("missing flag --%s", flag), nil)
		return 0
	}
	return val
}

// resolveHubPath returns the active hub directory path.
func resolveHubPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		outputError(1, fmt.Sprintf("cannot determine home directory: %v", err), nil)
		return ""
	}
	return resolveHubPathForHome(home, resolveRuntimeChannel())
}

// hubStore returns a new Store rooted at the hub directory.
// Returns nil on failure (error already reported via outputError).
func hubStore() *storage.Store {
	dir := resolveHubPath()
	s, err := storage.NewStore(dir)
	if err != nil {
		outputError(1, fmt.Sprintf("failed to initialize hub store: %v", err), nil)
		return nil
	}
	return s
}

func renderVisualError(message string, details interface{}) string {
	if entry, ok := friendlyErrorForPattern(message); ok {
		return renderFriendlyError(entry, message)
	}
	var b strings.Builder
	b.WriteString(renderBanner("\u274C", "Error"))
	b.WriteString(visualDividerStr())
	b.WriteString(strings.TrimSpace(message))
	b.WriteString("\n")
	if details != nil {
		detailText := strings.TrimSpace(fmt.Sprint(details))
		if detailText != "" && detailText != "<nil>" {
			b.WriteString(detailText)
			b.WriteString("\n")
		}
	}
	// Append generic hint for unmatched errors (per D-05)
	b.WriteString("\nRun `aether patrol` for diagnostics or `aether status` to check colony health.\n")
	return b.String()
}

// surveyDigestBudgetChars bounds the whole condensed codebase-map digest
// (resolveSurveyDigestSection). This is its OWN allowance, deliberately kept
// outside colonyPrimeBudgetChars / colonyPrimeCompactBudgetChars
// (cmd/colony_prime_context.go) so no lesson, rule or failure the memory pack
// carries is ever displaced to make room for the map (D-01). Same
// separate-budget precedent as the 8K skill-injection budget and
// colonyResearchBuildBudgetChars.
const surveyDigestBudgetChars = 2000

// surveyDigestPerReportChars caps how much of any single survey report's
// condensed body enters the digest, so one long report cannot consume the
// entire digest budget and crowd out the others.
const surveyDigestPerReportChars = 400

// resolveSurveySection reads available survey artifacts from .aether/data/survey/
// and returns a markdown section summarizing them. Returns empty string if no
// survey data exists or if the store is not initialized.
func resolveSurveySection() string {
	if store == nil {
		return ""
	}
	surveyDir := filepath.Join(store.BasePath(), "survey")
	entries, err := os.ReadDir(surveyDir)
	if err != nil || len(entries) == 0 {
		return ""
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".json") {
			files = append(files, name)
		}
	}
	if len(files) == 0 {
		return ""
	}
	sort.Strings(files)

	var b strings.Builder
	b.WriteString("### Territory Survey\n\n")
	// Say how old the map is before handing over the pointer list (D-10) — a
	// worker (and the brief inspector) should never ground on the survey
	// without knowing whether it is fresh or stale.
	if notice := surveyStalenessNotice(); notice != "" {
		b.WriteString(notice)
	}
	// Real repo-relative paths, not bare filenames. A worker handed "BLUEPRINT.md"
	// with no directory cannot resolve it; ".aether/data/survey/BLUEPRINT.md" it
	// can open directly. The colony data dir is always <root>/.aether/data by
	// scaffold convention (ensureRepoLocalScaffold), so the relative form holds
	// from the repo root every worker runs in.
	b.WriteString("Survey documents (read the ones relevant to your task):\n")
	for _, f := range files {
		b.WriteString(fmt.Sprintf("- .aether/data/survey/%s\n", f))
	}
	return b.String()
}

// resolveSurveyDigestSection condenses the survey reports under
// .aether/data/survey/ into a small, bounded, age-labelled digest a prompt
// can carry as actual content -- not just the filename list resolveSurveySection
// already renders (WIRE-03). A helper handed a filename has to go and read
// it, or guess; a helper handed the content does not.
//
// Condensation is a plain read-and-truncate over the files on disk -- never a
// model call -- following resolveColonyResearchSection's exact
// bounded-content-with-omission loop shape (cmd/colony_research.go): a
// running total against surveyDigestBudgetChars, a per-report cap
// (surveyDigestPerReportChars), a named "not included" list for reports that
// did not fit, and a truncation notice at a paragraph boundary rather than a
// silent cut. This makes the digest deterministic (same files, same bytes,
// every run) and makes deleting a report provably change the digest, because
// there is no rescan of the repository anywhere in this path -- only these
// named files.
//
// The digest is headed by surveyStalenessNotice() whenever the map is old
// enough for one (D-02), followed by a plain-English guidance line: file
// paths in the map can be trusted, claims about behaviour should be checked
// against the code. A map that is very old is still sent, headed by that
// notice -- never silently dropped past a cutoff.
//
// Every condensed report body is sanitized with colony.SanitizeSignalContent
// before it enters the digest -- survey reports are helper-authored and are
// being replayed verbatim into three later worker prompts (T-198.2-18,
// cmd/memory_feed.go's established rule for worker-authored text). A report
// whose condensed body the sanitizer rejects is named in the omission list
// rather than silently dropped, matching the omission handling for reports
// that did not fit the budget.
//
// Returns "" when no survey reports exist, so a colony that has never been
// surveyed gets a byte-identical brief to before this digest existed
// (TestNoSurveyMeansNoDigest, TestBriefsAreUnchangedWithoutASurvey).
func resolveSurveyDigestSection() string {
	if store == nil {
		return ""
	}
	surveyDir := filepath.Join(store.BasePath(), "survey")
	entries, err := os.ReadDir(surveyDir)
	if err != nil || len(entries) == 0 {
		return ""
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".md") {
			files = append(files, entry.Name())
		}
	}
	if len(files) == 0 {
		return ""
	}
	sort.Strings(files)

	var body strings.Builder
	used := 0
	omitted := make([]string, 0, len(files))

	for _, name := range files {
		data, readErr := os.ReadFile(filepath.Join(surveyDir, name))
		if readErr != nil {
			omitted = append(omitted, fmt.Sprintf("%s (unreadable)", name))
			continue
		}
		content := strings.TrimSpace(string(data))
		if content == "" {
			continue
		}
		remaining := surveyDigestBudgetChars - used
		if remaining <= 0 {
			omitted = append(omitted, name)
			continue
		}
		budget := surveyDigestPerReportChars
		if budget > remaining {
			budget = remaining
		}
		if len(content) > budget {
			cut := content[:budget]
			if idx := strings.LastIndex(cut, "\n\n"); idx > budget/2 {
				cut = cut[:idx]
			}
			content = cut + fmt.Sprintf("\n\n_(truncated — full report: .aether/data/survey/%s)_", name)
		}
		sanitized, sanErr := colony.SanitizeSignalContent(content)
		if sanErr != nil {
			omitted = append(omitted, fmt.Sprintf("%s (content rejected: %v)", name, sanErr))
			continue
		}
		fmt.Fprintf(&body, "### %s\n\n%s\n\n", name, sanitized)
		used += len(content)
	}

	if body.Len() == 0 && len(omitted) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("### Codebase Map Digest\n\n")
	if notice := surveyStalenessNotice(); notice != "" {
		b.WriteString(notice)
	}
	b.WriteString("File paths in this digest can be trusted; claims about how the code behaves should be checked against the code itself.\n\n")
	b.WriteString(body.String())
	if len(omitted) > 0 {
		fmt.Fprintf(&b, "_Not included here (over budget or rejected) — read the full report directly if relevant: %s_\n", strings.Join(omitted, ", "))
	}
	return strings.TrimSpace(b.String())
}
