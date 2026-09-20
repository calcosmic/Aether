package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

const medicLastScanFile = "medic-last-scan.json"

// AutoSpawnCheck determines whether the Medic should auto-spawn.
type AutoSpawnCheck struct {
	ShouldSpawn bool   `json:"should_spawn"`
	Reason      string `json:"reason,omitempty"`
	Severity    string `json:"severity,omitempty"`
}

// shouldAutoSpawnMedic checks whether health conditions warrant auto-spawning
// the Medic agent during continue-gates.
func shouldAutoSpawnMedic(dataPath string) AutoSpawnCheck {
	// Check 1: Stale session (session.json older than 24h)
	if stale, age := checkStaleSession(dataPath); stale {
		return AutoSpawnCheck{
			ShouldSpawn: true,
			Reason:      fmt.Sprintf("Session is stale (last activity %s ago)", age),
			Severity:    "warning",
		}
	}

	// Check 2: Critical blocker flags
	if hasCriticalBlocker(dataPath) {
		return AutoSpawnCheck{
			ShouldSpawn: true,
			Reason:      "Critical blocker flag is active",
			Severity:    "critical",
		}
	}

	// Check 3: Corrupted state (run quick health scan)
	if hasCriticalHealthIssue(dataPath) {
		return AutoSpawnCheck{
			ShouldSpawn: true,
			Reason:      "Colony state has critical health issues",
			Severity:    "critical",
		}
	}

	return AutoSpawnCheck{ShouldSpawn: false}
}

// checkStaleSession returns true if session.json hasn't been modified in 24+ hours.
func checkStaleSession(dataPath string) (bool, string) {
	sessionPath := filepath.Join(dataPath, "session.json")
	info, err := os.Stat(sessionPath)
	if err != nil {
		return false, ""
	}
	age := time.Since(info.ModTime())
	if age > 24*time.Hour {
		hours := int(age.Hours())
		return true, fmt.Sprintf("%dh", hours)
	}
	return false, ""
}

// hasCriticalBlocker checks pending-decisions.json for unresolved critical blockers.
func hasCriticalBlocker(dataPath string) bool {
	// Try pending-decisions.json first, then flags.json
	for _, filename := range []string{"pending-decisions.json", "flags.json"} {
		data, err := os.ReadFile(filepath.Join(dataPath, filename))
		if err != nil {
			continue
		}
		var flagsFile colony.FlagsFile
		if err := json.Unmarshal(data, &flagsFile); err != nil {
			continue
		}
		for _, d := range flagsFile.Decisions {
			if !d.Resolved && d.Type == "blocker" {
				return true
			}
		}
	}
	return false
}

// hasCriticalHealthIssue runs a quick scan for critical health issues.
func hasCriticalHealthIssue(dataPath string) bool {
	lastScanPath := filepath.Join(dataPath, medicLastScanFile)
	data, err := os.ReadFile(lastScanPath)
	if err != nil {
		return false
	}
	var scan MedicLastScan
	if err := json.Unmarshal(data, &scan); err != nil {
		return false
	}
	for _, issue := range scan.Issues {
		if issue.Severity == "critical" {
			return true
		}
	}
	return false
}

// MedicLastScan holds the persisted result of the last medic scan.
type MedicLastScan struct {
	Timestamp string        `json:"timestamp"`
	Issues    []HealthIssue `json:"issues"`
	Goal      string        `json:"goal,omitempty"`
	Phase     int           `json:"phase"`
}

// saveMedicLastScan persists scan results for colony-prime to read.
func saveMedicLastScan(dataPath string, issues []HealthIssue, goal string, phase int) error {
	scan := MedicLastScan{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Issues:    issues,
		Goal:      goal,
		Phase:     phase,
	}
	data, err := json.MarshalIndent(scan, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dataPath, medicLastScanFile), append(data, '\n'), 0644)
}

// loadMedicLastScan reads the last medic scan results.
func loadMedicLastScan(dataPath string) (*MedicLastScan, error) {
	data, err := os.ReadFile(filepath.Join(dataPath, medicLastScanFile))
	if err != nil {
		return nil, err
	}
	var scan MedicLastScan
	if err := json.Unmarshal(data, &scan); err != nil {
		return nil, err
	}
	return &scan, nil
}
