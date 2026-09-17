package colony

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestDependencyContractSemanticAndNumeric(t *testing.T) {
	for _, reference := range []string{"preserve-restoration-baseline", "1.1"} {
		t.Run(reference, func(t *testing.T) {
			phases := []Phase{
				{ID: 1, Tasks: []Task{{ID: strPtr("1.1"), SemanticID: "preserve-restoration-baseline"}}},
				{ID: 2, Tasks: []Task{{ID: strPtr("2.1"), SemanticID: "rehearse-daily", DependsOn: []string{reference}}}},
			}
			before, _ := json.Marshal(phases)
			if err := DetectCycles(phases); err != nil {
				t.Fatalf("existing cross-phase target was rejected: %v", err)
			}
			after, _ := json.Marshal(phases)
			if string(before) != string(after) {
				t.Fatal("dependency validation rewrote accepted plan content")
			}
		})
	}
}

func TestDependencyContractRejectsAmbiguityAndCycles(t *testing.T) {
	for _, kind := range []string{"missing", "cycle", "alias-runtime", "duplicate-semantic", "duplicate-runtime"} {
		t.Run(kind, func(t *testing.T) {
			phases := []Phase{{ID: 1, Tasks: []Task{
				{ID: strPtr("1.1"), SemanticID: "baseline"},
				{ID: strPtr("1.2"), SemanticID: "daily", DependsOn: []string{"baseline"}},
			}}}
			switch kind {
			case "missing":
				phases[0].Tasks[1].DependsOn = []string{"absent"}
			case "cycle":
				phases[0].Tasks[0].DependsOn = []string{"daily"}
			case "alias-runtime":
				phases[0].Tasks[0].SemanticID = "1.2"
				phases[0].Tasks[1].DependsOn = []string{"1.1"}
			case "duplicate-semantic":
				phases[0].Tasks[1].SemanticID = "baseline"
				phases[0].Tasks[1].DependsOn = []string{"1.1"}
			case "duplicate-runtime":
				phases[0].Tasks[1].ID = strPtr("1.1")
				phases[0].Tasks[1].DependsOn = nil
			}
			err := DetectCycles(phases)
			if err == nil {
				t.Fatal("invalid graph accepted")
			}
			if kind == "cycle" {
				var cycle *CycleError
				if !errors.As(err, &cycle) {
					t.Fatalf("semantic cycle misreported: %v", err)
				}
			}
			if kind == "missing" {
				var missing *MissingDepError
				if !errors.As(err, &missing) {
					t.Fatalf("missing target misreported: %v", err)
				}
			}
		})
	}
}
