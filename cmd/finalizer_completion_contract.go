package cmd

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var durableBuildCompletionPattern = regexp.MustCompile(`(?:^|/)\.aether/data/build/phase-[1-9][0-9]*/attempts/attempt-[A-Za-z0-9._-]+\.completion\.json$`)

const (
	finalizerCompletionTempPattern = "${TMPDIR:-/tmp}/aether-<workflow>-<run>/<workflow>-completion.json"
	finalizerManifestMaxAge        = 24 * time.Hour
	finalizerManifestFutureSkew    = 5 * time.Minute
)

func finalizerCompletionContractStep(workflow string) string {
	workflow = strings.TrimSpace(strings.ToLower(workflow))
	if workflow == "" {
		workflow = "<workflow>"
	}
	return fmt.Sprintf("Write per-worker JSON and the final completion JSON under `%s`, replacing `<workflow>` with `%s`; never write wrapper result artifacts under `.aether/data`.", finalizerCompletionTempPattern, workflow)
}

func validateFinalizerCompletionFilePath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	if path == "-" {
		return fmt.Errorf("completion file cannot be read from stdin; write an approved temp result artifact path such as %s", finalizerCompletionTempPattern)
	}
	normalized := filepath.ToSlash(filepath.Clean(strings.ReplaceAll(path, "\\", "/")))
	if normalized == ".aether/data" || strings.HasPrefix(normalized, ".aether/data/") || strings.Contains(normalized, "/.aether/data/") {
		if durableBuildCompletionPattern.MatchString(normalized) {
			return nil
		}
		return fmt.Errorf("completion file must be outside .aether/data; use an approved temp result artifact path such as %s", finalizerCompletionTempPattern)
	}
	return nil
}

func validateFinalizerManifestFreshness(label, generatedAt string, now time.Time) error {
	generatedAt = strings.TrimSpace(generatedAt)
	if generatedAt == "" {
		return fmt.Errorf("%s generated_at is required for freshness validation; rerun the plan-only command for a fresh manifest", label)
	}
	generated, err := time.Parse(time.RFC3339, generatedAt)
	if err != nil {
		return fmt.Errorf("%s generated_at %q is not RFC3339; rerun the plan-only command for a fresh manifest", label, generatedAt)
	}
	generated = generated.UTC()
	now = now.UTC()
	if generated.After(now.Add(finalizerManifestFutureSkew)) {
		return fmt.Errorf("%s generated_at %s is too far in the future; rerun the plan-only command", label, generated.Format(time.RFC3339))
	}
	if now.Sub(generated) > finalizerManifestMaxAge {
		return fmt.Errorf("stale %s generated at %s; rerun the plan-only command before finalizing", label, generated.Format(time.RFC3339))
	}
	return nil
}

func preferCompletedResultOverTimeout(existingStatus, incomingStatus string) (useIncoming bool, ok bool) {
	existing := normalizeExternalBuildStatus(existingStatus)
	incoming := normalizeExternalBuildStatus(incomingStatus)
	switch {
	case existing == "timeout" && (incoming == "completed" || incoming == "manually-reconciled"):
		return true, true
	case incoming == "timeout" && (existing == "completed" || existing == "manually-reconciled"):
		return false, true
	default:
		return false, false
	}
}
