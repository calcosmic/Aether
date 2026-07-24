package cmd

import (
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestBuildLearningContent(t *testing.T) {
	phase := colony.Phase{ID: 42, Name: "Test Phase"}

	// Test with no workers
	content := buildLearningContent(phase, nil)
	if !strings.Contains(content, "Phase 42: Test Phase") {
		t.Errorf("expected phase header, got: %s", content)
	}
	if !strings.Contains(content, "No workers completed successfully") {
		t.Errorf("expected no-workers message, got: %s", content)
	}

	// Test with completed worker containing findings and lessons
	workerFlow := []codexContinueWorkerFlowStep{
		{
			Name:            "Builder-1",
			Caste:           "builder",
			Status:          "completed",
			Task:            "Fix auth bug",
			Findings:        []codexReviewFinding{{Description: "Worker A found issue X"}},
			ReusableLessons: []string{"Always validate tokens before use"},
			Recommendations: []string{"Add rate limiting"},
			WeakSpots:       []string{"Token refresh logic"},
			EdgeCases:       []string{"Empty token string"},
		},
		{
			Name:   "Watcher-1",
			Caste:  "watcher",
			Status: "failed",
			Task:   "Run tests",
		},
	}

	content = buildLearningContent(phase, workerFlow)

	if !strings.Contains(content, "Workers completed: 1") {
		t.Errorf("expected 1 completed worker, got: %s", content)
	}
	if !strings.Contains(content, "Builder-1 (builder)") {
		t.Errorf("expected worker name, got: %s", content)
	}
	if !strings.Contains(content, "Worker A found issue X") {
		t.Errorf("expected finding text, got: %s", content)
	}
	if !strings.Contains(content, "Always validate tokens before use") {
		t.Errorf("expected reusable lesson, got: %s", content)
	}
	if !strings.Contains(content, "Add rate limiting") {
		t.Errorf("expected recommendation, got: %s", content)
	}
	if !strings.Contains(content, "Token refresh logic") {
		t.Errorf("expected weak spot, got: %s", content)
	}
	if !strings.Contains(content, "Empty token string") {
		t.Errorf("expected edge case, got: %s", content)
	}

	// Failed worker should NOT appear
	if strings.Contains(content, "Watcher-1") {
		t.Errorf("failed worker should not appear, got: %s", content)
	}
}

func TestBuildLearningContent_MultipleWorkers(t *testing.T) {
	phase := colony.Phase{ID: 10, Name: "Multi Worker Phase"}
	workerFlow := []codexContinueWorkerFlowStep{
		{
			Name:     "Builder-1",
			Caste:    "builder",
			Status:   "completed",
			Findings: []codexReviewFinding{{Description: "Finding 1"}},
		},
		{
			Name:            "Builder-2",
			Caste:           "builder",
			Status:          "completed",
			ReusableLessons: []string{"Lesson 2"},
		},
	}

	content := buildLearningContent(phase, workerFlow)
	if !strings.Contains(content, "Finding 1") {
		t.Errorf("expected finding from worker 1, got: %s", content)
	}
	if !strings.Contains(content, "Lesson 2") {
		t.Errorf("expected lesson from worker 2, got: %s", content)
	}
}
