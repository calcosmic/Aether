package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/calcosmic/Aether/pkg/storage"
)

// swarmResultRecord is the durable source of truth for retry history. Strike
// state is derived from these terminal records instead of a second mutable
// counter that can drift from what the swarm actually did.
type swarmResultRecord struct {
	SwarmID           string                 `json:"swarm_id"`
	Target            string                 `json:"target"`
	TargetFingerprint string                 `json:"target_fingerprint"`
	Status            string                 `json:"status"`
	RootCause         string                 `json:"root_cause,omitempty"`
	Solution          string                 `json:"solution,omitempty"`
	Recommendation    string                 `json:"recommendation,omitempty"`
	Workers           []swarmWorkerExecution `json:"workers,omitempty"`
	Files             []string               `json:"files,omitempty"`
	Tests             []string               `json:"tests,omitempty"`
	Blockers          []string               `json:"blockers,omitempty"`
	CompletedAt       string                 `json:"completed_at"`
	DispatchMode      string                 `json:"dispatch_mode,omitempty"`
}

type swarmStrikeEvidence struct {
	SwarmID     string `json:"swarm_id"`
	Status      string `json:"status"`
	CompletedAt string `json:"completed_at"`
}

type swarmStrikeHistory struct {
	TargetFingerprint string                `json:"target_fingerprint"`
	StrikeCount       int                   `json:"strike_count"`
	NextAttempt       int                   `json:"next_attempt"`
	Evidence          []swarmStrikeEvidence `json:"evidence,omitempty"`
}

type datedSwarmResult struct {
	record      swarmResultRecord
	completedAt time.Time
}

func swarmTargetFingerprint(target string) string {
	normalized := normalizeSwarmTarget(target)
	if normalized == "" {
		return ""
	}
	digest := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(digest[:])
}

func normalizeSwarmTarget(target string) string {
	runes := []rune(strings.TrimSpace(target))
	for len(runes) > 0 && isSwarmTargetBoundary(runes[0]) {
		runes = runes[1:]
	}
	for len(runes) > 0 && isSwarmTargetBoundary(runes[len(runes)-1]) {
		runes = runes[:len(runes)-1]
	}
	if len(runes) == 0 {
		return ""
	}
	return strings.ToLower(strings.Join(strings.Fields(string(runes)), " "))
}

func isSwarmTargetBoundary(r rune) bool {
	return unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r)
}

func saveSwarmResultRecord(s *storage.Store, record swarmResultRecord) error {
	if s == nil {
		return fmt.Errorf("save swarm result: no store initialized")
	}
	record.SwarmID = strings.TrimSpace(record.SwarmID)
	record.Target = strings.TrimSpace(record.Target)
	record.Status = strings.ToLower(strings.TrimSpace(record.Status))
	record.CompletedAt = strings.TrimSpace(record.CompletedAt)
	if _, err := safeIdentifierSegment(filepath.Join(s.BasePath(), "swarms"), "swarm id", record.SwarmID); err != nil {
		return fmt.Errorf("save swarm result: %w", err)
	}
	record.TargetFingerprint = swarmTargetFingerprint(record.Target)
	if record.TargetFingerprint == "" {
		return fmt.Errorf("save swarm result: target is required")
	}
	if record.Status == "" {
		return fmt.Errorf("save swarm result: status is required")
	}
	if _, err := time.Parse(time.RFC3339Nano, record.CompletedAt); err != nil {
		return fmt.Errorf("save swarm result: invalid completed_at: %w", err)
	}
	path := filepath.ToSlash(filepath.Join("swarms", record.SwarmID, "result.json"))
	if err := s.SaveJSON(path, record); err != nil {
		return fmt.Errorf("save swarm result: %w", err)
	}
	return nil
}

func evaluateSwarmStrikeHistory(s *storage.Store, target string) (swarmStrikeHistory, error) {
	fingerprint := swarmTargetFingerprint(target)
	history := swarmStrikeHistory{
		TargetFingerprint: fingerprint,
		NextAttempt:       1,
	}
	if s == nil {
		return history, fmt.Errorf("evaluate swarm strike history: no store initialized")
	}
	if fingerprint == "" {
		return history, fmt.Errorf("evaluate swarm strike history: target is required")
	}

	swarmsDir := filepath.Join(s.BasePath(), "swarms")
	entries, err := os.ReadDir(swarmsDir)
	if os.IsNotExist(err) {
		return history, nil
	}
	if err != nil {
		return history, fmt.Errorf("evaluate swarm strike history: read swarms: %w", err)
	}

	results := make([]datedSwarmResult, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		resultPath := filepath.Join(swarmsDir, entry.Name(), "result.json")
		info, statErr := os.Lstat(resultPath)
		if statErr != nil || !info.Mode().IsRegular() {
			continue
		}
		data, readErr := os.ReadFile(resultPath)
		if readErr != nil {
			continue
		}
		var record swarmResultRecord
		if json.Unmarshal(data, &record) != nil {
			continue
		}
		record.SwarmID = strings.TrimSpace(record.SwarmID)
		record.Target = strings.TrimSpace(record.Target)
		record.Status = strings.ToLower(strings.TrimSpace(record.Status))
		record.CompletedAt = strings.TrimSpace(record.CompletedAt)
		if record.SwarmID == "" || record.SwarmID != entry.Name() {
			continue
		}
		recordFingerprint := swarmTargetFingerprint(record.Target)
		if recordFingerprint == "" {
			continue
		}
		storedFingerprint := strings.TrimSpace(record.TargetFingerprint)
		if storedFingerprint != "" && storedFingerprint != recordFingerprint {
			continue
		}
		if recordFingerprint != fingerprint {
			continue
		}
		if record.Status != "completed" && record.Status != "failed" && record.Status != "blocked" {
			continue
		}
		completedAt, parseErr := time.Parse(time.RFC3339Nano, record.CompletedAt)
		if parseErr != nil {
			continue
		}
		results = append(results, datedSwarmResult{record: record, completedAt: completedAt})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].completedAt.Equal(results[j].completedAt) {
			return results[i].record.SwarmID < results[j].record.SwarmID
		}
		return results[i].completedAt.Before(results[j].completedAt)
	})
	for _, result := range results {
		if result.record.Status == "completed" {
			history.Evidence = nil
			continue
		}
		history.Evidence = append(history.Evidence, swarmStrikeEvidence{
			SwarmID:     result.record.SwarmID,
			Status:      result.record.Status,
			CompletedAt: result.record.CompletedAt,
		})
	}
	history.StrikeCount = len(history.Evidence)
	history.NextAttempt = history.StrikeCount + 1
	return history, nil
}
