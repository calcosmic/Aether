package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/learn"
)

func mustParseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

// TestV113Validation exercises all v1.13 file format validation functions.
// Each sub-test verifies that valid inputs pass and invalid inputs produce
// actionable error messages containing: format name, field name, expected
// value, and actual value.

func TestV113Validation(t *testing.T) {
	t.Run("ValidateHeartbeatFile accepts valid heartbeat JSON", func(t *testing.T) {
		data := []byte(`{
			"worker_id": "Hammer-23",
			"caste": "builder",
			"timestamp": "2026-05-02T14:30:00Z",
			"phase": 1
		}`)
		hf, err := ValidateHeartbeatFile(data)
		if err != nil {
			t.Fatalf("expected valid heartbeat to pass, got error: %v", err)
		}
		if hf.WorkerID != "Hammer-23" {
			t.Errorf("expected worker_id 'Hammer-23', got %q", hf.WorkerID)
		}
		if hf.Phase != 1 {
			t.Errorf("expected phase 1, got %d", hf.Phase)
		}
	})

	t.Run("ValidateHeartbeatFile rejects missing worker_id", func(t *testing.T) {
		data := []byte(`{
			"caste": "builder",
			"timestamp": "2026-05-02T14:30:00Z",
			"phase": 1
		}`)
		_, err := ValidateHeartbeatFile(data)
		if err == nil {
			t.Fatal("expected error for missing worker_id, got nil")
		}
		if !strings.Contains(err.Error(), "heartbeat file") {
			t.Errorf("error should reference 'heartbeat file', got: %v", err)
		}
		if !strings.Contains(err.Error(), "worker_id") {
			t.Errorf("error should reference field 'worker_id', got: %v", err)
		}
		if !strings.Contains(err.Error(), "missing") {
			t.Errorf("error should describe the problem (missing), got: %v", err)
		}
	})

	t.Run("ValidateHeartbeatFile rejects invalid timestamp format", func(t *testing.T) {
		data := []byte(`{
			"worker_id": "Hammer-23",
			"caste": "builder",
			"timestamp": "not-a-date",
			"phase": 1
		}`)
		_, err := ValidateHeartbeatFile(data)
		if err == nil {
			t.Fatal("expected error for invalid timestamp, got nil")
		}
		if !strings.Contains(err.Error(), "heartbeat file") {
			t.Errorf("error should reference 'heartbeat file', got: %v", err)
		}
		if !strings.Contains(err.Error(), "timestamp") {
			t.Errorf("error should reference field 'timestamp', got: %v", err)
		}
		if !strings.Contains(err.Error(), "RFC3339") {
			t.Errorf("error should mention expected format RFC3339, got: %v", err)
		}
		if !strings.Contains(err.Error(), "not-a-date") {
			t.Errorf("error should include the actual value 'not-a-date', got: %v", err)
		}
	})

	t.Run("ValidateHeartbeatFile rejects negative phase", func(t *testing.T) {
		data := []byte(`{
			"worker_id": "Hammer-23",
			"caste": "builder",
			"timestamp": "2026-05-02T14:30:00Z",
			"phase": -1
		}`)
		_, err := ValidateHeartbeatFile(data)
		if err == nil {
			t.Fatal("expected error for negative phase, got nil")
		}
		if !strings.Contains(err.Error(), "heartbeat file") {
			t.Errorf("error should reference 'heartbeat file', got: %v", err)
		}
		if !strings.Contains(err.Error(), "phase") {
			t.Errorf("error should reference field 'phase', got: %v", err)
		}
		if !strings.Contains(err.Error(), "0") {
			t.Errorf("error should mention expected range (>= 0), got: %v", err)
		}
	})

	t.Run("ValidateGateResults accepts valid gate-results JSON", func(t *testing.T) {
		data := []byte(`{
			"results": [
				{"name": "tests_gate", "status": "passed", "timestamp": "2026-05-02T14:30:00Z"},
				{"name": "flags_gate", "status": "failed", "timestamp": "2026-05-02T14:30:00Z", "detail": "1 critical flag"}
			]
		}`)
		err := ValidateGateResults(data)
		if err != nil {
			t.Fatalf("expected valid gate-results to pass, got error: %v", err)
		}
	})

	t.Run("ValidateGateResults rejects missing results field", func(t *testing.T) {
		data := []byte(`{"unblock_attempts": 0}`)
		err := ValidateGateResults(data)
		if err == nil {
			t.Fatal("expected error for missing results, got nil")
		}
		if !strings.Contains(err.Error(), "gate-results") {
			t.Errorf("error should reference 'gate-results', got: %v", err)
		}
		if !strings.Contains(err.Error(), "results") {
			t.Errorf("error should reference field 'results', got: %v", err)
		}
	})

	t.Run("ValidateGateResults rejects invalid gate status", func(t *testing.T) {
		data := []byte(`{
			"results": [
				{"name": "tests_gate", "status": "unknown", "timestamp": "2026-05-02T14:30:00Z"}
			]
		}`)
		err := ValidateGateResults(data)
		if err == nil {
			t.Fatal("expected error for invalid status, got nil")
		}
		if !strings.Contains(err.Error(), "gate-results") {
			t.Errorf("error should reference 'gate-results', got: %v", err)
		}
		if !strings.Contains(err.Error(), "unknown") {
			t.Errorf("error should include actual value 'unknown', got: %v", err)
		}
		// Error should list valid statuses
		for _, valid := range []string{"passed", "failed", "skipped", "not-reached"} {
			if !strings.Contains(err.Error(), valid) {
				t.Errorf("error should list valid status %q, got: %v", valid, err)
			}
		}
	})

	t.Run("ValidateLearningEntry accepts valid learning entry", func(t *testing.T) {
		entry := learn.Entry{
			ID:             "learn-001",
			Content:        "Tests should cover edge cases",
			Phase:          1,
			Classification: learn.ClassRepoLocal,
			Confidence:     0.85,
			Evidence: learn.Evidence{
				RunID:     "run-1",
				Phase:     1,
				Timestamp: "2026-05-02T14:30:00Z",
			},
		}
		err := ValidateLearningEntry(entry)
		if err != nil {
			t.Fatalf("expected valid entry to pass, got error: %v", err)
		}
	})

	t.Run("ValidateLearningEntry rejects confidence out of range", func(t *testing.T) {
		for _, conf := range []float64{-0.1, 1.1} {
			entry := learn.Entry{
				ID:             "learn-001",
				Content:        "Some content",
				Phase:          1,
				Classification: learn.ClassRepoLocal,
				Confidence:     conf,
				Evidence: learn.Evidence{
					Timestamp: "2026-05-02T14:30:00Z",
				},
			}
			err := ValidateLearningEntry(entry)
			if err == nil {
				t.Fatalf("expected error for confidence %v, got nil", conf)
			}
			if !strings.Contains(err.Error(), "learning entry") {
				t.Errorf("error should reference 'learning entry', got: %v", err)
			}
			if !strings.Contains(err.Error(), "confidence") {
				t.Errorf("error should reference field 'confidence', got: %v", err)
			}
			if !strings.Contains(err.Error(), "0.0") || !strings.Contains(err.Error(), "1.0") {
				t.Errorf("error should state range 0.0-1.0, got: %v", err)
			}
		}
	})

	t.Run("ValidateSkillFrontmatter accepts valid SKILL.md", func(t *testing.T) {
		content := `---
name: go-testing
category: domain
roles: [builder, watcher]
detect: ["*.go", "go.mod"]
---

Go testing best practices.
`
		err := ValidateSkillFrontmatter(content)
		if err != nil {
			t.Fatalf("expected valid skill frontmatter to pass, got error: %v", err)
		}
	})

	t.Run("ValidateSkillFrontmatter rejects missing name", func(t *testing.T) {
		content := `---
category: domain
roles: [builder]
---

Content.
`
		err := ValidateSkillFrontmatter(content)
		if err == nil {
			t.Fatal("expected error for missing name, got nil")
		}
		if !strings.Contains(err.Error(), "skill frontmatter") {
			t.Errorf("error should reference 'skill frontmatter', got: %v", err)
		}
		if !strings.Contains(err.Error(), "name") {
			t.Errorf("error should reference field 'name', got: %v", err)
		}
		if !strings.Contains(err.Error(), "missing") {
			t.Errorf("error should describe the problem (missing), got: %v", err)
		}
	})

	t.Run("ValidateTrackedProcessJSON accepts valid worker-processes.json", func(t *testing.T) {
		processes := []codex.TrackedProcess{
			{
				PID:        12345,
				WorkerName: "Builder-1",
				Caste:      "builder",
				SpawnedAt:  mustParseTime("2026-05-02T14:30:00Z"),
			},
		}
		err := ValidateTrackedProcessJSON(processes)
		if err != nil {
			t.Fatalf("expected valid process data to pass, got error: %v", err)
		}
	})

	t.Run("ValidateTrackedProcessJSON rejects PID <= 0", func(t *testing.T) {
		for _, pid := range []int{0, -1} {
			processes := []codex.TrackedProcess{
				{
					PID:        pid,
					WorkerName: "Builder-1",
					SpawnedAt:  mustParseTime("2026-05-02T14:30:00Z"),
				},
			}
			err := ValidateTrackedProcessJSON(processes)
			if err == nil {
				t.Fatalf("expected error for PID %d, got nil", pid)
			}
			if !strings.Contains(err.Error(), "worker-processes") {
				t.Errorf("error should reference 'worker-processes', got: %v", err)
			}
			if !strings.Contains(err.Error(), "PID") {
				t.Errorf("error should reference field 'PID', got: %v", err)
			}
			if !strings.Contains(err.Error(), "positive") {
				t.Errorf("error should say 'positive', got: %v", err)
			}
		}
	})

	t.Run("Error messages contain all required parts", func(t *testing.T) {
		// Test that error messages contain: format name, field name, expected, actual
		// We check a representative set of errors.

		// Heartbeat: missing worker_id
		_, err := ValidateHeartbeatFile([]byte(`{"timestamp":"2026-05-02T14:30:00Z","phase":1}`))
		if err == nil {
			t.Fatal("expected error")
		}
		msg := err.Error()
		if !strings.Contains(msg, "heartbeat file") {
			t.Errorf("missing format name in: %s", msg)
		}
		if !strings.Contains(msg, "worker_id") {
			t.Errorf("missing field name in: %s", msg)
		}

		// Gate results: invalid status
		err = ValidateGateResults([]byte(`{"results":[{"name":"test","status":"bogus","timestamp":"2026-05-02T14:30:00Z"}]}`))
		if err == nil {
			t.Fatal("expected error")
		}
		msg = err.Error()
		if !strings.Contains(msg, "gate-results") {
			t.Errorf("missing format name in: %s", msg)
		}
		if !strings.Contains(msg, "status") {
			t.Errorf("missing field name in: %s", msg)
		}
		if !strings.Contains(msg, "bogus") {
			t.Errorf("missing actual value in: %s", msg)
		}

		// Learning entry: empty ID
		entry := learn.Entry{
			ID:             "",
			Content:        "content",
			Phase:          1,
			Classification: learn.ClassRepoLocal,
			Confidence:     0.5,
			Evidence:       learn.Evidence{Timestamp: "2026-05-02T14:30:00Z"},
		}
		err = ValidateLearningEntry(entry)
		if err == nil {
			t.Fatal("expected error")
		}
		msg = err.Error()
		if !strings.Contains(msg, "learning entry") {
			t.Errorf("missing format name in: %s", msg)
		}
		if !strings.Contains(msg, "id") {
			t.Errorf("missing field name in: %s", msg)
		}
	})
}
