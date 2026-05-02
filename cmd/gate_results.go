package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
)

// validGateStatuses lists the allowed gate status values.
var validGateStatuses = []string{"passed", "failed", "skipped", "not-reached"}

// ValidateGateResults validates gate-results JSON data, returning an
// actionable error message when the format is invalid. Error messages
// include: format name ("gate-results"), field name, expected value,
// and actual value.
func ValidateGateResults(data []byte) error {
	var fileData gateResultsFile
	if err := json.Unmarshal(data, &fileData); err != nil {
		return fmt.Errorf("gate-results: invalid JSON: %v", err)
	}
	if fileData.Results == nil {
		return fmt.Errorf("gate-results: missing required field 'results'")
	}
	validSet := make(map[string]bool, len(validGateStatuses))
	for _, s := range validGateStatuses {
		validSet[s] = true
	}
	for i, gate := range fileData.Results {
		if !validSet[gate.Status] {
			return fmt.Errorf(
				"gate-results: invalid status %q for gate %q (index %d), valid statuses: %s",
				gate.Status, gate.Name, i, strings.Join(validGateStatuses, ", "),
			)
		}
		if strings.TrimSpace(gate.Name) == "" {
			return fmt.Errorf(
				"gate-results: missing required field 'name' for gate at index %d",
				i,
			)
		}
	}
	return nil
}
