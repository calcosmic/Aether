package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
	"gopkg.in/yaml.v3"
)

func TestResumeWrapperContract199(t *testing.T) {
	const (
		description = "Validate and restore the safest honest recovery point."
		runtimeCall = "AETHER_OUTPUT_MODE=visual aether resume $ARGUMENTS"
		source      = ".aether/commands/resume.yaml"
	)

	if resumeColonyCmd.Use != "resume" || resumeColonyCmd.Short != description {
		t.Fatalf("canonical resume command metadata drifted: use=%q short=%q", resumeColonyCmd.Use, resumeColonyCmd.Short)
	}
	if len(resumeColonyCmd.Aliases) != 0 {
		t.Fatalf("retired recovery names leaked into Cobra aliases: %v", resumeColonyCmd.Aliases)
	}

	provenance := []colony.RecoveryProvenance{
		colony.RecoveryProvenanceConfirmed,
		colony.RecoveryProvenanceReconstructed,
		colony.RecoveryProvenanceConflicting,
		colony.RecoveryProvenanceUnknown,
	}
	wantProvenance := []string{"confirmed", "reconstructed", "conflicting", "unknown"}
	for index, got := range provenance {
		if string(got) != wantProvenance[index] {
			t.Errorf("runtime provenance class %d = %q, want %q", index, got, wantProvenance[index])
		}
	}
	for name, outcome := range map[string]pauseResumeLifecycleOutcome{
		"conflicting": pauseResumeConflictOutcome("named evidence conflict"),
		"unknown":     pauseResumeUnknownOutcome("named evidence is unavailable"),
	} {
		if string(outcome.Provenance) != name || outcome.StateEffect != colony.LifecycleStateEffectNone {
			t.Errorf("%s runtime stop = %#v, want matching provenance and state effect none", name, outcome)
		}
	}

	repoRoot := filepath.Clean("..")
	yamlPath := filepath.Join(repoRoot, filepath.FromSlash(source))
	rawYAML := readResumeWrapperContract199File(t, yamlPath)
	var spec struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
		Runtime     struct {
			Command string `yaml:"command"`
		} `yaml:"runtime"`
		CeremonyContract struct {
			Authority        string   `yaml:"authority"`
			WrapperRole      string   `yaml:"wrapper_role"`
			ProvenanceGroups []string `yaml:"provenance_groups"`
			ConflictStop     string   `yaml:"conflict_stop"`
		} `yaml:"ceremony_contract"`
		Guardrails []string `yaml:"guardrails"`
	}
	if err := yaml.Unmarshal(rawYAML, &spec); err != nil {
		t.Fatalf("parse canonical resume YAML: %v", err)
	}
	if spec.Name != "ant-resume" || spec.Description != description || spec.Runtime.Command != runtimeCall {
		t.Fatalf("canonical resume wrapper contract drifted: %+v", spec)
	}
	canonicalContract := strings.Join(append([]string{
		spec.CeremonyContract.Authority,
		spec.CeremonyContract.WrapperRole,
		spec.CeremonyContract.ConflictStop,
	}, append(spec.CeremonyContract.ProvenanceGroups, spec.Guardrails...)...), "\n")
	assertResumeWrapperSemantics199(t, source, canonicalContract)

	wrapperPaths := []string{
		filepath.Join(repoRoot, ".claude", "commands", "ant-resume.md"),
		filepath.Join(repoRoot, ".claude", "commands", "ant", "resume.md"),
		filepath.Join(repoRoot, ".opencode", "commands", "ant", "resume.md"),
	}
	var firstWrapper string
	for _, path := range wrapperPaths {
		raw := readResumeWrapperContract199File(t, path)
		text := string(raw)
		wantHeader := "<!-- Aether-managed: runtime spec at " + source + ". Synced by aether update. -->"
		if !strings.HasPrefix(text, wantHeader+"\n") {
			t.Errorf("%s lacks canonical managed-source linkage", path)
		}
		if !strings.Contains(text, `description: "`+description+`"`) {
			t.Errorf("%s does not carry the exact canonical description", path)
		}
		if strings.Count(text, runtimeCall) != 1 {
			t.Errorf("%s has %d canonical runtime calls, want exactly one", path, strings.Count(text, runtimeCall))
		}
		assertResumeWrapperSemantics199(t, path, text)
		if firstWrapper == "" {
			firstWrapper = text
		} else if text != firstWrapper {
			t.Errorf("generated resume wrapper %s is not byte-identical to the other platform surfaces", path)
		}
	}
}

func assertResumeWrapperSemantics199(t *testing.T, surface, text string) {
	t.Helper()
	lower := strings.ToLower(text)
	for _, required := range []string{
		"confirmed", "reconstructed", "conflicting", "unknown",
		"state effect: none", "receipt", "exactly once",
		"do not inspect, select, or modify recovery evidence",
	} {
		if !strings.Contains(lower, required) {
			t.Errorf("%s lacks resume authority/provenance contract %q", surface, required)
		}
	}
	for _, forbidden := range []string{
		"resume-colony", "/ant-recover", "aether recover", "resume-dashboard",
		"colony_state.json", "session.json", "handoff.md", "savejson", "writefile(", "os.readfile", "jq ", "rm -",
	} {
		if strings.Contains(lower, forbidden) {
			t.Errorf("%s exposes retired routing or host-owned evidence/state behavior %q", surface, forbidden)
		}
	}
}

func readResumeWrapperContract199File(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return content
}
