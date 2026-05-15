package smoke

import (
	"github.com/calcosmic/Aether/pkg/storage"
)

// DocCLIAlignment holds the results of the doc-CLI alignment smoke test.
type DocCLIAlignment struct {
	CommandsChecked      int            `json:"commands_checked"`
	HostCriticalChecked  int            `json:"host_critical_checked"`
	HostCriticalFailures []FlagMismatch `json:"host_critical_failures"`
	Warnings             []FlagMismatch `json:"warnings"`
}

// WritePlatformHealth writes a unified platform-health.json file that
// preserves existing failed_commands and flag_mismatches keys while
// adding (or updating) the doc_cli_alignment section.
func WritePlatformHealth(s *storage.Store, failedCommands []string, mismatches []FlagMismatch, alignment *DocCLIAlignment) error {
	// Try to load existing platform-health.json to preserve old keys
	existing := map[string]interface{}{
		"failed_commands": failedCommands,
		"flag_mismatches": mismatches,
	}
	if s != nil {
		var old map[string]interface{}
		if s.LoadJSON("platform-health.json", &old) == nil {
			// Preserve any extra keys from the old file
			for k, v := range old {
				if _, ok := existing[k]; !ok {
					existing[k] = v
				}
			}
		}
	}

	if alignment != nil {
		existing["doc_cli_alignment"] = alignment
	}

	return s.SaveJSON("platform-health.json", existing)
}
