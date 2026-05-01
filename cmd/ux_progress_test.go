package cmd

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

func TestCeremonyProgressNonTTYAdvance(t *testing.T) {
	steps := []string{"Context", "Tasks", "Dispatch"}
	var buf bytes.Buffer
	p := newCeremonyProgress(steps, &buf)

	p.Advance("Context")
	p.Advance("Tasks")
	p.Advance("Dispatch")

	output := buf.String()
	if !strings.Contains(output, "Step 1/3: Context") {
		t.Errorf("expected 'Step 1/3: Context', got: %s", output)
	}
	if !strings.Contains(output, "Step 2/3: Tasks") {
		t.Errorf("expected 'Step 2/3: Tasks', got: %s", output)
	}
	if !strings.Contains(output, "Step 3/3: Dispatch") {
		t.Errorf("expected 'Step 3/3: Dispatch', got: %s", output)
	}
	// Each line should include elapsed time in parentheses
	if !strings.Contains(output, "(0s)") && !strings.Contains(output, "(1ms)") {
		t.Errorf("expected elapsed time in parentheses, got: %s", output)
	}
}

func TestCeremonyProgressNonTTYFinish(t *testing.T) {
	steps := []string{"Alpha", "Beta"}
	var buf bytes.Buffer
	p := newCeremonyProgress(steps, &buf)

	p.Advance("Alpha")
	p.Advance("Beta")
	p.Finish()

	output := buf.String()
	if !strings.Contains(output, "Ceremony complete in") {
		t.Errorf("expected 'Ceremony complete in', got: %s", output)
	}
	// Should contain a duration value
	durationPattern := regexp.MustCompile(`Ceremony complete in \d+(\.\d+)?(s|ms)`)
	if !durationPattern.MatchString(output) {
		t.Errorf("expected duration pattern in output, got: %s", output)
	}
}

func TestCeremonyProgressNonTTYIsPlainText(t *testing.T) {
	steps := []string{"Step1"}
	var buf bytes.Buffer
	p := newCeremonyProgress(steps, &buf)

	// With a bytes.Buffer (non-TTY), tty should be false
	if p.tty {
		t.Error("expected tty=false for bytes.Buffer writer")
	}
	if p.bar != nil {
		t.Error("expected nil progressbar for non-TTY writer")
	}
}

func TestCeremonyProgressSteps(t *testing.T) {
	steps := []string{"Alpha", "Beta", "Gamma"}
	var buf bytes.Buffer
	p := newCeremonyProgress(steps, &buf)

	result := p.Steps()
	if len(result) != 3 {
		t.Errorf("expected 3 steps, got %d", len(result))
	}
	if result[0] != "Alpha" || result[1] != "Beta" || result[2] != "Gamma" {
		t.Errorf("unexpected step names: %v", result)
	}
}

func TestCeremonyProgressElapsedTiming(t *testing.T) {
	steps := []string{"OnlyStep"}
	var buf bytes.Buffer
	p := newCeremonyProgress(steps, &buf)

	p.Advance("OnlyStep")
	p.Finish()

	output := buf.String()
	// Should contain a duration string matching patterns like "0s", "1ms", "0.5s"
	durationPattern := regexp.MustCompile(`(\d+(\.\d+)?(s|ms))`)
	if !durationPattern.MatchString(output) {
		t.Errorf("expected duration pattern in output, got: %s", output)
	}
	// The finish line specifically should have a duration
	if !strings.Contains(output, "Ceremony complete in") {
		t.Errorf("expected 'Ceremony complete in' in output, got: %s", output)
	}
}

func TestCeremonyProgressEmptySteps(t *testing.T) {
	steps := []string{}
	var buf bytes.Buffer
	p := newCeremonyProgress(steps, &buf)

	p.Finish()

	output := buf.String()
	if !strings.Contains(output, "Ceremony complete in") {
		t.Errorf("expected 'Ceremony complete in' even with empty steps, got: %s", output)
	}
}

func TestNewCeremonyProgressExported(t *testing.T) {
	steps := []string{"Test"}
	var buf bytes.Buffer
	p := NewCeremonyProgress(steps, &buf)
	if p == nil {
		t.Fatal("expected non-nil progress from NewCeremonyProgress")
	}
	if len(p.Steps()) != 1 {
		t.Errorf("expected 1 step, got %d", len(p.Steps()))
	}
}
