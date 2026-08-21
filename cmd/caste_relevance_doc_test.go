package cmd

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestCasteRelevanceDoc_ReferencesAllAlwaysRequiredCastes verifies that the
// caste-relevance-reference.md playbook mentions all always-required castes
// for each flow type.
func TestCasteRelevanceDoc_ReferencesAllAlwaysRequiredCastes(t *testing.T) {
	docPath := "../.aether/docs/command-playbooks/caste-relevance-reference.md"
	contentBytes, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("failed to read %s: %v", docPath, err)
	}
	content := string(contentBytes)

	// Build a set of all castes that can ever be always-required across any flow.
	// We verify the doc mentions each caste in the always-required sections.
	// We use representative phase/state combos to exercise every branch of isAlwaysRequired.
	cases := []struct {
		flow  string
		phase colony.Phase
		state colony.ColonyState
		want  []string
	}{
		{
			flow: "build",
			phase: colony.Phase{
				Name: "Security hardening",
				Mode: colony.PhaseModeProduction,
				Tasks: []colony.Task{
					{Goal: "Implement hardened configuration checks"},
				},
			},
			state: colony.ColonyState{},
			want:  []string{"builder", "watcher", "probe", "auditor", "gatekeeper"},
		},
		{
			flow: "continue",
			phase: colony.Phase{
				Name: "Test phase",
				Mode: colony.PhaseModePrototype,
			},
			state: colony.ColonyState{VerificationDepth: string(colony.VerificationDepthLight)},
			want:  []string{"watcher"},
		},
		{
			flow: "continue",
			phase: colony.Phase{
				Name: "Test phase",
				Mode: colony.PhaseModePrototype,
			},
			state: colony.ColonyState{VerificationDepth: string(colony.VerificationDepthStandard)},
			want:  []string{"watcher", "probe"},
		},
		{
			flow: "continue",
			phase: colony.Phase{
				Name: "Test phase",
				Mode: colony.PhaseModePrototype,
			},
			state: colony.ColonyState{VerificationDepth: string(colony.VerificationDepthHeavy)},
			want:  []string{"watcher", "gatekeeper", "auditor", "probe"},
		},
		{
			flow: "plan",
			phase: colony.Phase{
				Name: "Plan phase",
				Mode: colony.PhaseModePrototype,
			},
			state: colony.ColonyState{},
			want:  []string{"scout", "route_setter"},
		},
		{
			flow: "colonize",
			phase: colony.Phase{
				Name: "Colonize repo",
				Mode: colony.PhaseModeDiscovery,
			},
			state: colony.ColonyState{},
			want:  []string{"surveyor-provisions", "surveyor-nest", "surveyor-disciplines", "surveyor-pathogens"},
		},
		{
			flow: "swarm",
			phase: colony.Phase{
				Name: "Swarm bug",
				Mode: colony.PhaseModeMaintenance,
				Tasks: []colony.Task{
					{Goal: "Fix the bug"},
				},
			},
			state: colony.ColonyState{},
			want:  []string{"tracker", "builder", "watcher"},
		},
		{
			flow: "seal",
			phase: colony.Phase{
				Name: "Production release",
				Mode: colony.PhaseModeProduction,
			},
			state: colony.ColonyState{VerificationDepth: string(colony.VerificationDepthLight)},
			want:  []string{},
		},
		{
			flow: "seal",
			phase: colony.Phase{
				Name: "Production release",
				Mode: colony.PhaseModeProduction,
			},
			state: colony.ColonyState{VerificationDepth: string(colony.VerificationDepthStandard)},
			want:  []string{"auditor", "probe"},
		},
		{
			flow: "seal",
			phase: colony.Phase{
				Name: "Production release",
				Mode: colony.PhaseModeProduction,
			},
			state: colony.ColonyState{VerificationDepth: string(colony.VerificationDepthHeavy)},
			want:  []string{"gatekeeper", "auditor", "probe"},
		},
	}

	for _, tc := range cases {
		required := queenRequiredCastesForBudget(tc.phase, tc.flow, tc.state)
		for _, caste := range tc.want {
			found := false
			for _, r := range required {
				if r == caste {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("fixture bug: %s flow expected %s always-required but Go code did not return it", tc.flow, caste)
				continue
			}
			if !strings.Contains(content, caste) {
				t.Errorf("doc missing mention of caste %q (always-required for %s flow)", caste, tc.flow)
			}
		}
	}
}

// TestCasteRelevanceDoc_SpawnBudgetNumbersMatch verifies that the documented
// spawn budget numbers match what queenMaxWorkersForBudget returns.
func TestCasteRelevanceDoc_SpawnBudgetNumbersMatch(t *testing.T) {
	docPath := "../.aether/docs/command-playbooks/caste-relevance-reference.md"
	contentBytes, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("failed to read %s: %v", docPath, err)
	}
	content := string(contentBytes)

	cases := []struct {
		flow      string
		phase     colony.Phase
		state     colony.ColonyState
		riskLevel string
		want      int
	}{
		// Build
		{flow: "build", phase: colony.Phase{Name: "Docs update", Mode: colony.PhaseModeMaintenance}, state: colony.ColonyState{}, riskLevel: "low", want: 4},
		{flow: "build", phase: colony.Phase{Name: "Security hardening", Mode: colony.PhaseModeProduction}, state: colony.ColonyState{}, riskLevel: "high", want: 8},
		{flow: "build", phase: colony.Phase{Name: "Core runtime changes", Mode: colony.PhaseModePrototype}, state: colony.ColonyState{}, riskLevel: "medium", want: 6},
		{flow: "build", phase: colony.Phase{Name: "Discovery spike", Mode: colony.PhaseModeDiscovery}, state: colony.ColonyState{}, riskLevel: "low", want: 5},
		{flow: "build", phase: colony.Phase{Name: "Feature work", Mode: colony.PhaseModePrototype}, state: colony.ColonyState{}, riskLevel: "low", want: 6},
		// Continue
		{flow: "continue", phase: colony.Phase{Name: "Test"}, state: colony.ColonyState{VerificationDepth: string(colony.VerificationDepthLight)}, riskLevel: "low", want: 3},
		{flow: "continue", phase: colony.Phase{Name: "Test"}, state: colony.ColonyState{VerificationDepth: string(colony.VerificationDepthHeavy)}, riskLevel: "low", want: 6},
		{flow: "continue", phase: colony.Phase{Name: "Test"}, state: colony.ColonyState{VerificationDepth: string(colony.VerificationDepthStandard)}, riskLevel: "low", want: 4},
		// Plan
		{flow: "plan", phase: colony.Phase{Name: "Security plan"}, state: colony.ColonyState{}, riskLevel: "high", want: 6},
		{flow: "plan", phase: colony.Phase{Name: "Feature plan"}, state: colony.ColonyState{}, riskLevel: "low", want: 4},
		// Colonize
		{flow: "colonize", phase: colony.Phase{Name: "Colonize"}, state: colony.ColonyState{}, riskLevel: "low", want: 4},
		// Swarm
		{flow: "swarm", phase: colony.Phase{Name: "Swarm"}, state: colony.ColonyState{}, riskLevel: "low", want: 5},
		// Seal
		{flow: "seal", phase: colony.Phase{Name: "Release"}, state: colony.ColonyState{VerificationDepth: string(colony.VerificationDepthHeavy)}, riskLevel: "high", want: 5},
		{flow: "seal", phase: colony.Phase{Name: "Release"}, state: colony.ColonyState{VerificationDepth: string(colony.VerificationDepthStandard)}, riskLevel: "low", want: 4},
	}

	for _, tc := range cases {
		got, reason := queenMaxWorkersForBudget(tc.phase, tc.flow, tc.state, tc.riskLevel)
		if got != tc.want {
			t.Fatalf("queenMaxWorkersForBudget(%q, %q, ...) = %d (%s), want %d", tc.flow, tc.phase.Name, got, reason, tc.want)
		}
		// Verify the doc contains the budget number for this flow
		if !strings.Contains(content, strconv.Itoa(got)) {
			t.Errorf("doc missing spawn budget number %d for %s flow (reason: %s)", got, tc.flow, reason)
		}
	}
}
