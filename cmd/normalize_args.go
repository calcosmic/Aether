package cmd

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

const legacySessionRedirectExpiry = "1.29.0"

type legacySessionRedirect struct {
	From      string
	To        string
	ExpiresAt string
}

// legacySessionProcessRedirect records only a pre-Cobra argv rewrite. The old
// tokens are deliberately absent from command metadata, help, completion, and
// generated wrappers; this pointer exists solely so the canonical command can
// print one replacement notice after parsing succeeds.
var legacySessionProcessRedirect *legacySessionRedirect

func normalizeLegacySessionInvocation(args []string, version string) ([]string, *legacySessionRedirect) {
	normalized := append([]string(nil), args...)
	if len(normalized) < 2 {
		return normalized, nil
	}
	to := normalizeLegacySessionCommand(normalized[1], version)
	if to == normalized[1] {
		return normalized, nil
	}
	redirect := &legacySessionRedirect{From: normalized[1], To: to, ExpiresAt: legacySessionRedirectExpiry}
	normalized[1] = to
	return normalized, redirect
}

// normalizeLegacySessionCommand is the one bounded compatibility boundary for
// persisted and CLI lifecycle command tokens. It accepts only exact historic
// values before the machine-checked 1.29 expiry; callers must persist and
// route with the canonical result, never with the input token.
func normalizeLegacySessionCommand(command, version string) string {
	if !versionBefore(version, legacySessionRedirectExpiry) {
		return command
	}
	switch command {
	case "pause-colony":
		return "pause"
	case "resume-colony":
		return "resume"
	default:
		return command
	}
}

func legacySessionRedirectNotice(redirect *legacySessionRedirect) string {
	if redirect == nil {
		return ""
	}
	expiry := strings.TrimSuffix(redirect.ExpiresAt, ".0")
	return fmt.Sprintf("notice: %s is deprecated and will be removed in %s; use aether %s.", redirect.From, expiry, redirect.To)
}

func versionBefore(left, right string) bool {
	leftParts, leftOK := parseMajorMinorPatch(left)
	rightParts, rightOK := parseMajorMinorPatch(right)
	if !leftOK || !rightOK {
		return false
	}
	for index := 0; index < len(leftParts); index++ {
		if leftParts[index] != rightParts[index] {
			return leftParts[index] < rightParts[index]
		}
	}
	return false
}

func parseMajorMinorPatch(version string) ([3]int, bool) {
	var parsed [3]int
	version = strings.TrimPrefix(strings.TrimSpace(version), "v")
	if cut, _, ok := strings.Cut(version, "-"); ok {
		version = cut
	}
	parts := strings.Split(version, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return parsed, false
	}
	for index, raw := range parts {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			return [3]int{}, false
		}
		parsed[index] = value
	}
	return parsed, true
}

var normalizeArgsCmd = &cobra.Command{
	Use:   "normalize-args [args...]",
	Short: "Normalize arguments from environment or positional params",
	Long:  "Outputs a single normalized argument string. Reads from ARGUMENTS env var first, then falls back to positional args. Collapses whitespace.",
	RunE: func(cmd *cobra.Command, args []string) error {
		normalized := ""

		// Try ARGUMENTS env var first (Claude Code style)
		if envArgs := os.Getenv("ARGUMENTS"); envArgs != "" {
			normalized = envArgs
		} else if len(args) > 0 {
			// Fall back to positional params (OpenCode style)
			normalized = strings.Join(args, " ")
		}

		// Collapse whitespace: replace runs of whitespace with single space, trim edges
		if normalized != "" {
			re := regexp.MustCompile(`\s+`)
			normalized = strings.TrimSpace(re.ReplaceAllString(normalized, " "))
		}

		outputOK(normalized)
		return nil
	},
}

func init() {
	os.Args, legacySessionProcessRedirect = normalizeLegacySessionInvocation(os.Args, resolveVersion())
	rootCmd.AddCommand(normalizeArgsCmd)
}
