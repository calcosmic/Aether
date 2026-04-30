package colony

import (
	"errors"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Cycle detection tests (LOOP-04)
// ---------------------------------------------------------------------------

// strPtr is declared in colony_test.go; this file uses it from there.

func TestDetectCycles(t *testing.T) {
	tests := []struct {
		name      string
		phases    []Phase
		wantErr   bool
		errType   error // nil, CycleError, or MissingDepError
		errDetail string // substring expected in error message
	}{
		{
			name: "no dependencies returns nil",
			phases: []Phase{
				{
					ID:   1,
					Tasks: []Task{{ID: strPtr("1.1"), Goal: "task 1.1"}},
				},
			},
			wantErr: false,
		},
		{
			name: "valid linear chain",
			phases: []Phase{
				{
					ID: 1,
					Tasks: []Task{
						{ID: strPtr("1.1"), Goal: "first", DependsOn: []string{}},
						{ID: strPtr("1.2"), Goal: "second", DependsOn: []string{"1.1"}},
						{ID: strPtr("1.3"), Goal: "third", DependsOn: []string{"1.2"}},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "simple two-node cycle",
			phases: []Phase{
				{
					ID: 1,
					Tasks: []Task{
						{ID: strPtr("1.1"), Goal: "A", DependsOn: []string{"1.2"}},
						{ID: strPtr("1.2"), Goal: "B", DependsOn: []string{"1.1"}},
					},
				},
			},
			wantErr: true,
			errType: &CycleError{},
		},
		{
			name: "three-node cycle",
			phases: []Phase{
				{
					ID: 1,
					Tasks: []Task{
						{ID: strPtr("1.1"), Goal: "A", DependsOn: []string{"1.3"}},
						{ID: strPtr("1.2"), Goal: "B", DependsOn: []string{"1.1"}},
						{ID: strPtr("1.3"), Goal: "C", DependsOn: []string{"1.2"}},
					},
				},
			},
			wantErr: true,
			errType: &CycleError{},
		},
		{
			name: "cross-phase cycle",
			phases: []Phase{
				{
					ID: 1,
					Tasks: []Task{
						{ID: strPtr("1.1"), Goal: "phase 1 task", DependsOn: []string{"2.1"}},
					},
				},
				{
					ID: 2,
					Tasks: []Task{
						{ID: strPtr("2.1"), Goal: "phase 2 task", DependsOn: []string{"1.1"}},
					},
				},
			},
			wantErr: true,
			errType: &CycleError{},
		},
		{
			name: "missing dependency reference",
			phases: []Phase{
				{
					ID: 1,
					Tasks: []Task{
						{ID: strPtr("1.1"), Goal: "task", DependsOn: []string{"9.9"}},
					},
				},
			},
			wantErr: true,
			errType: &MissingDepError{},
			errDetail: "9.9",
		},
		{
			name: "CycleError produces readable string",
			phases: []Phase{
				{
					ID: 1,
					Tasks: []Task{
						{ID: strPtr("1.1"), Goal: "A", DependsOn: []string{"1.2"}},
						{ID: strPtr("1.2"), Goal: "B", DependsOn: []string{"1.1"}},
					},
				},
			},
			wantErr: true,
			errType: &CycleError{},
			errDetail: "circular dependency",
		},
		{
			name: "nil task IDs are skipped gracefully",
			phases: []Phase{
				{
					ID: 1,
					Tasks: []Task{
						{ID: nil, Goal: "no ID task", DependsOn: []string{"1.2"}},
						{ID: strPtr("1.2"), Goal: "has ID", DependsOn: nil},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DetectCycles(tt.phases)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if tt.wantErr && tt.errType != nil {
				if !errors.As(err, tt.errType) {
					t.Fatalf("expected error type %T, got: %T (%v)", tt.errType, err, err)
				}
				if tt.errDetail != "" {
					if !strings.Contains(err.Error(), tt.errDetail) {
						t.Fatalf("expected error to contain %q, got: %s", tt.errDetail, err.Error())
					}
				}
			}
		})
	}
}

func TestCycleErrorFormat(t *testing.T) {
	err := &CycleError{Tasks: []string{"1.1", "1.2", "1.1"}}
	msg := err.Error()
	expected := "circular dependency: 1.1 -> 1.2 -> 1.1"
	if msg != expected {
		t.Fatalf("expected %q, got %q", expected, msg)
	}
}

func TestMissingDepErrorFormat(t *testing.T) {
	err := &MissingDepError{Task: "1.1", MissingDep: "9.9"}
	msg := err.Error()
	expected := "task 1.1 depends on unknown task 9.9"
	if msg != expected {
		t.Fatalf("expected %q, got %q", expected, msg)
	}
}
