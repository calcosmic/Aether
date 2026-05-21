package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestCLAUDEMDVerificationDepthClaims(t *testing.T) {
	data, err := os.ReadFile("../CLAUDE.md")
	if err != nil {
		t.Fatalf("read CLAUDE.md: %v", err)
	}
	content := string(data)

	// Check that CLAUDE.md documents the 5-level priority chain
	if !strings.Contains(content, "Explicit `--heavy` or `--light` flag") {
		t.Error("CLAUDE.md missing explicit flag priority")
	}
	if !strings.Contains(content, "Explicit `--verification-depth") {
		t.Error("CLAUDE.md missing explicit --verification-depth priority")
	}
	if !strings.Contains(content, "Keyword match in phase name") {
		t.Error("CLAUDE.md missing keyword match priority")
	}
	if !strings.Contains(content, "Smart default based on phase mode") {
		t.Error("CLAUDE.md missing smart default priority")
	}

	// Check that CLAUDE.md documents smart defaults correctly
	if !strings.Contains(content, "Discovery mode → light") {
		t.Error("CLAUDE.md missing discovery=light rule")
	}
	if !strings.Contains(content, "Production mode → at least standard") {
		t.Error("CLAUDE.md missing production=standard rule")
	}
	if !strings.Contains(content, "Final phase → heavy") {
		t.Error("CLAUDE.md missing final=heavy rule")
	}
	if !strings.Contains(content, "security") {
		t.Error("CLAUDE.md missing security keyword rule")
	}

	// Check that CLAUDE.md documents what each depth means
	if !strings.Contains(content, "Light") {
		t.Error("CLAUDE.md missing Light depth description")
	}
	if !strings.Contains(content, "Standard") {
		t.Error("CLAUDE.md missing Standard depth description")
	}
	if !strings.Contains(content, "Heavy") {
		t.Error("CLAUDE.md missing Heavy depth description")
	}

	// Verify claims against runtime behavior
	phase := colony.Phase{ID: 1, Name: "Security hardening", Mode: colony.PhaseModeProduction}
	depth := resolveVerificationDepth(phase, 5, false, false, "")
	if depth != colony.VerificationDepthHeavy {
		t.Errorf("security phase should get heavy, got %s", depth)
	}

	discoveryPhase := colony.Phase{ID: 2, Name: "Documentation cleanup", Mode: colony.PhaseModeDiscovery}
	discoveryDepth := resolveVerificationDepth(discoveryPhase, 5, false, false, "")
	if discoveryDepth != colony.VerificationDepthLight {
		t.Errorf("discovery phase should get light, got %s", discoveryDepth)
	}

	finalPhase := colony.Phase{ID: 5, Name: "Polish", Mode: colony.PhaseModePrototype}
	finalDepth := resolveVerificationDepth(finalPhase, 5, false, false, "")
	if finalDepth != colony.VerificationDepthHeavy {
		t.Errorf("final phase should get heavy, got %s", finalDepth)
	}
}

func TestCLAUDEMDNoOldModes(t *testing.T) {
	data, err := os.ReadFile("../CLAUDE.md")
	if err != nil {
		t.Fatalf("read CLAUDE.md: %v", err)
	}
	content := string(data)

	// The old 3-mode description should be gone
	if strings.Contains(content, "`fast`: low-risk work; light continue verification") {
		t.Error("CLAUDE.md still contains old 'fast' mode description")
	}
	if strings.Contains(content, "watcher subprocess skipped") {
		t.Error("CLAUDE.md still contains 'watcher subprocess skipped'")
	}
}
