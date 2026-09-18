package codex

import (
	"strings"
	"testing"
)

func TestNativeCancellationCapabilityRequiresProof(t *testing.T) {
	for _, tc := range []struct {
		name    string
		level   CapabilityLevel
		allowed bool
	}{
		{"empty", "", false},
		{"unknown", CapabilityLevel("unknown"), false},
		{"limited", CapabilityLimited, false},
		{"test-only", CapabilityTestOnly, false},
		{"unavailable", CapabilityUnavailable, false},
		{"proven", CapabilityProven, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// This exercises only the deterministic admission rule. It cannot
			// qualify the installed host or change its unavailable capability.
			err := ValidateNativeCancellationCapability(CapabilitySupport{Level: tc.level})
			if (err == nil) != tc.allowed {
				t.Fatalf("level %q: allowed=%v, error=%v", tc.level, tc.allowed, err)
			}
			if err != nil && !strings.Contains(err.Error(), "cannot provide requested cancellation guarantee") {
				t.Fatalf("missing explicit cancellation refusal: %v", err)
			}
		})
	}
}

func TestNativeWorkerCancellationRequirement(t *testing.T) {
	request := NativeWorkerRequirements{
		Profile:           PermissionProfileForCaste("builder"),
		HostPermission:    "workspace_write",
		Workspace:         "/fixture",
		AcceptedWorkspace: "/fixture",
		ParallelMode:      "in-repo",
	}
	if err := ValidateNativeWorkerRequirements(request); err != nil {
		t.Fatalf("ordinary inherited-workspace request refused: %v", err)
	}
	request.RequireCancellation = true
	if err := ValidateNativeWorkerRequirements(request); err == nil || !strings.Contains(err.Error(), "cannot provide requested cancellation guarantee") {
		t.Fatalf("unavailable cancellation guarantee was not explicitly refused: %v", err)
	}
	contract, ok := PlatformContractFor(PlatformCodex)
	if !ok || contract.NativeWorkers == nil || contract.NativeWorkers.Cancellation.Level != CapabilityUnavailable {
		t.Fatal("refusal must not promote cancellation to a supported capability")
	}
}
