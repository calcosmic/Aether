package codex

import "testing"

func TestPlatformContractsStateHonestDifferences(t *testing.T) {
	tests := []struct {
		platform       Platform
		tier           string
		dispatch       CapabilityLevel
		routing        CapabilityLevel
		permissions    CapabilityLevel
		commandSurface CapabilityLevel
	}{
		{PlatformClaude, "primary", CapabilityProven, CapabilityProven, CapabilityLimited, CapabilityProven},
		{PlatformOpenCode, "primary", CapabilityLimited, CapabilityLimited, CapabilityLimited, CapabilityProven},
		{PlatformCodex, "secondary", CapabilityProven, CapabilityLimited, CapabilityLimited, CapabilityUnavailable},
	}

	for _, tt := range tests {
		t.Run(string(tt.platform), func(t *testing.T) {
			contract, ok := PlatformContractFor(tt.platform)
			if !ok {
				t.Fatalf("missing contract for %s", tt.platform)
			}
			if contract.SchemaVersion != 1 || contract.SupportTier != tt.tier {
				t.Fatalf("unexpected identity: %#v", contract)
			}
			if contract.WorkerDispatch.Level != tt.dispatch || contract.NamedCasteRouting.Level != tt.routing {
				t.Fatalf("unexpected dispatch contract: %#v", contract)
			}
			if contract.PermissionIsolation.Level != tt.permissions || contract.NativeCommandSurface.Level != tt.commandSurface {
				t.Fatalf("unexpected host contract: %#v", contract)
			}
			if contract.StateLifecycle.Level != CapabilityProven || contract.StructuredCompletion.Level != CapabilityProven {
				t.Fatalf("shared deterministic core is not marked proven: %#v", contract)
			}
		})
	}
}

func TestUnknownPlatformHasNoContract(t *testing.T) {
	if _, ok := PlatformContractFor(PlatformUnknown); ok {
		t.Fatal("unknown platform unexpectedly has a support contract")
	}
}
