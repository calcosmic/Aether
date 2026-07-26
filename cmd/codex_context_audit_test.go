package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestResolveSurveySection_WithSurveyData(t *testing.T) {
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	surveyDir := filepath.Join(s.BasePath(), "survey")
	if err := os.MkdirAll(surveyDir, 0755); err != nil {
		t.Fatalf("mkdir survey: %v", err)
	}
	if err := os.WriteFile(filepath.Join(surveyDir, "BLUEPRINT.md"), []byte("# Blueprint\n"), 0644); err != nil {
		t.Fatalf("write blueprint: %v", err)
	}
	if err := os.WriteFile(filepath.Join(surveyDir, "PROVISIONS.json"), []byte(`{"deps":[]}`), 0644); err != nil {
		t.Fatalf("write provisions: %v", err)
	}

	section := resolveSurveySection()
	if section == "" {
		t.Fatal("expected non-empty survey section")
	}
	if !strings.Contains(section, "Territory Survey") {
		t.Errorf("expected 'Territory Survey' header, got: %s", section)
	}
	if !strings.Contains(section, "BLUEPRINT.md") {
		t.Errorf("expected BLUEPRINT.md, got: %s", section)
	}
	if !strings.Contains(section, "PROVISIONS.json") {
		t.Errorf("expected PROVISIONS.json, got: %s", section)
	}
}

func TestResolveSurveySection_NoSurveyData(t *testing.T) {
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	section := resolveSurveySection()
	if section != "" {
		t.Errorf("expected empty section when no survey data, got: %s", section)
	}
}

func TestResolveSurveySection_NoStore(t *testing.T) {
	store = nil
	section := resolveSurveySection()
	if section != "" {
		t.Errorf("expected empty section when store is nil, got: %s", section)
	}
}

func TestBuildBriefIncludesSurvey(t *testing.T) {
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	surveyDir := filepath.Join(s.BasePath(), "survey")
	_ = os.MkdirAll(surveyDir, 0755)
	_ = os.WriteFile(filepath.Join(surveyDir, "BLUEPRINT.md"), []byte("# Blueprint\n"), 0644)

	phase := colony.Phase{ID: 1, Name: "Test"}
	dispatch := codexBuildDispatch{Name: "Builder-1", Caste: "builder", Task: "Build feature"}
	brief := renderCodexBuildWorkerBrief(tmpDir, phase, dispatch, time.Now())

	if !strings.Contains(brief, "Territory Survey") {
		t.Errorf("expected survey section in build brief, got:\n%s", brief)
	}
}

func TestContinueBriefIncludesSurvey(t *testing.T) {
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	surveyDir := filepath.Join(s.BasePath(), "survey")
	_ = os.MkdirAll(surveyDir, 0755)
	_ = os.WriteFile(filepath.Join(surveyDir, "PATHOGENS.md"), []byte("# Pathogens\n"), 0644)

	phase := colony.Phase{ID: 2, Name: "Test Continue"}
	spec := codexContinueReviewSpec{Caste: "auditor", Task: "Review work"}
	brief := renderCodexContinueReviewBrief(tmpDir, phase, codexContinueManifest{}, codexContinueVerificationReport{}, codexContinueAssessment{}, spec)

	if !strings.Contains(brief, "Territory Survey") {
		t.Errorf("expected survey section in continue brief, got:\n%s", brief)
	}
}
