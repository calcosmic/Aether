package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/calcosmic/Aether/pkg/storage"
	"github.com/spf13/cobra"
)

// outputOK writes a JSON success envelope to stdout:
//
//	{"ok":true,"result":<result>}
//
// This matches the shell's json_ok() function format for playbook compatibility.
func outputOK(result interface{}) {
	resultJSON, _ := json.Marshal(result)
	fmt.Fprintf(stdout, "{\"ok\":true,\"result\":%s}\n", string(resultJSON))
}

// outputError writes a JSON error envelope to stderr:
//
//	{"ok":false,"error":"<message>","code":<code>}
//
// This matches the shell's json_err() function format for playbook compatibility.
func outputError(code int, message string, details interface{}) {
	if shouldRenderVisualOutput(stderr) {
		fmt.Fprint(stderr, renderVisualError(message, details))
		return
	}
	msgJSON, _ := json.Marshal(message)
	fmt.Fprintf(stderr, "{\"ok\":false,\"error\":%s,\"code\":%d}\n", string(msgJSON), code)
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

// resolveHubPath returns the hub directory path (~/.aether/).
func resolveHubPath() string {
	if dir := os.Getenv("AETHER_HUB_DIR"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		outputError(1, fmt.Sprintf("cannot determine home directory: %v", err), nil)
		return ""
	}
	return filepath.Join(home, ".aether")
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
	var b strings.Builder
	b.WriteString(renderBanner("❌", "Error"))
	b.WriteString(visualDivider)
	b.WriteString(strings.TrimSpace(message))
	b.WriteString("\n")
	if details != nil {
		detailText := strings.TrimSpace(fmt.Sprint(details))
		if detailText != "" && detailText != "<nil>" {
			b.WriteString(detailText)
			b.WriteString("\n")
		}
	}
	return b.String()
}
