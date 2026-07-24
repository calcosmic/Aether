package codex

// CapabilityLevel states how strongly Aether can promise a platform capability.
type CapabilityLevel string

const (
	CapabilityProven      CapabilityLevel = "proven"
	CapabilityLimited     CapabilityLevel = "limited"
	CapabilityUnavailable CapabilityLevel = "unavailable"
	CapabilityTestOnly    CapabilityLevel = "test_only"
)

// CapabilitySupport records both the mechanism and the limitations behind a
// platform claim. Callers should not infer parity from a shared command name.
type CapabilitySupport struct {
	Level       CapabilityLevel `json:"level"`
	Mechanism   string          `json:"mechanism"`
	Limitations []string        `json:"limitations,omitempty"`
}

// PlatformContract is the versioned, machine-readable support contract for one
// worker host. State transitions remain Go-owned on every supported platform.
type PlatformContract struct {
	SchemaVersion        int               `json:"schema_version"`
	Platform             Platform          `json:"platform"`
	SupportTier          string            `json:"support_tier"`
	StateLifecycle       CapabilitySupport `json:"state_lifecycle"`
	WorkerDispatch       CapabilitySupport `json:"worker_dispatch"`
	NamedCasteRouting    CapabilitySupport `json:"named_caste_routing"`
	StructuredCompletion CapabilitySupport `json:"structured_completion"`
	ProgressEvents       CapabilitySupport `json:"progress_events"`
	PermissionIsolation  CapabilitySupport `json:"permission_isolation"`
	NativeCommandSurface CapabilitySupport `json:"native_command_surface"`
}

// PlatformContractFor returns the explicit support promise for a worker host.
func PlatformContractFor(platform Platform) (PlatformContract, bool) {
	commonLifecycle := CapabilitySupport{
		Level:     CapabilityProven,
		Mechanism: "the Go runtime owns locks, manifests, evidence validation, lifecycle transitions, and recovery state",
	}
	commonCompletion := CapabilitySupport{
		Level:     CapabilityProven,
		Mechanism: "worker claims are parsed into the shared schema and accepted only through Go finalizers",
	}
	commonProgress := CapabilitySupport{
		Level:     CapabilityLimited,
		Mechanism: "the Go dispatcher emits process start, output-observed, heartbeat, and terminal events",
		Limitations: []string{
			"provider-native tool steps are not normalized into the shared event stream",
		},
	}

	switch platform {
	case PlatformClaude:
		return PlatformContract{
			SchemaVersion:        1,
			Platform:             platform,
			SupportTier:          "primary",
			StateLifecycle:       commonLifecycle,
			WorkerDispatch:       CapabilitySupport{Level: CapabilityProven, Mechanism: "the Go dispatcher launches the Claude CLI for each worker"},
			NamedCasteRouting:    CapabilitySupport{Level: CapabilityProven, Mechanism: "the dispatcher passes the generated caste agent with --agent"},
			StructuredCompletion: commonCompletion,
			ProgressEvents:       commonProgress,
			PermissionIsolation: CapabilitySupport{
				Level:     CapabilityLimited,
				Mechanism: "typed repository-read-only workers use plan mode; workspace-write workers use acceptEdits with fail-closed native sandboxing",
				Limitations: []string{
					"test-only, ledger-only, documentation-only, and survey-only write scopes are behavioral restrictions inside the workspace boundary",
				},
			},
			NativeCommandSurface: CapabilitySupport{Level: CapabilityProven, Mechanism: "generated Claude slash-command wrappers drive the shared Go manifests and finalizers"},
		}, true
	case PlatformOpenCode:
		return PlatformContract{
			SchemaVersion:        1,
			Platform:             platform,
			SupportTier:          "primary",
			StateLifecycle:       commonLifecycle,
			WorkerDispatch:       CapabilitySupport{Level: CapabilityLimited, Mechanism: "the Go dispatcher launches the restricted Aether primary router and asks it to invoke Task exactly once", Limitations: []string{"dispatch depends on the host Task tool being available"}},
			NamedCasteRouting:    CapabilitySupport{Level: CapabilityLimited, Mechanism: "the restricted primary router receives the requested caste as Task subagent_type", Limitations: []string{"Aether validates worker identity but cannot independently attest OpenCode's internal subagent selection"}},
			StructuredCompletion: commonCompletion,
			ProgressEvents:       commonProgress,
			PermissionIsolation: CapabilitySupport{
				Level:     CapabilityLimited,
				Mechanism: "the Aether router denies edit and bash, read-only castes deny edit/bash, and all Aether agents deny external-directory access",
				Limitations: []string{
					"provider-owned temporary output directories remain available to OpenCode",
					"narrow write scopes inside the project workspace are behavioral rather than path-enforced",
				},
			},
			NativeCommandSurface: CapabilitySupport{Level: CapabilityProven, Mechanism: "generated OpenCode slash-command wrappers drive the shared Go manifests and finalizers"},
		}, true
	case PlatformCodex:
		return PlatformContract{
			SchemaVersion:        1,
			Platform:             platform,
			SupportTier:          "secondary",
			StateLifecycle:       commonLifecycle,
			WorkerDispatch:       CapabilitySupport{Level: CapabilityProven, Mechanism: "the Go dispatcher launches codex exec for each worker"},
			NamedCasteRouting:    CapabilitySupport{Level: CapabilityLimited, Mechanism: "Aether injects the generated TOML caste definition into the worker prompt", Limitations: []string{"Codex exec does not expose the same direct named-agent selector as Claude"}},
			StructuredCompletion: commonCompletion,
			ProgressEvents:       commonProgress,
			PermissionIsolation: CapabilitySupport{
				Level:     CapabilityLimited,
				Mechanism: "the Go adapter selects Codex read-only or workspace-write sandbox mode from the canonical caste profile",
				Limitations: []string{
					"Codex cannot disable shell access per caste",
					"narrow write scopes inside the project workspace are behavioral rather than path-enforced",
				},
			},
			NativeCommandSurface: CapabilitySupport{Level: CapabilityUnavailable, Mechanism: "Codex has no slash-command wrapper mechanism", Limitations: []string{"Codex uses direct CLI lifecycle commands, command-guide, and installed lifecycle skills"}},
		}, true
	case PlatformFake:
		return PlatformContract{
			SchemaVersion:        1,
			Platform:             platform,
			SupportTier:          "test_only",
			StateLifecycle:       CapabilitySupport{Level: CapabilityTestOnly, Mechanism: "deterministic in-process fixture"},
			WorkerDispatch:       CapabilitySupport{Level: CapabilityTestOnly, Mechanism: "deterministic in-process fixture"},
			NamedCasteRouting:    CapabilitySupport{Level: CapabilityTestOnly, Mechanism: "deterministic in-process fixture"},
			StructuredCompletion: CapabilitySupport{Level: CapabilityTestOnly, Mechanism: "deterministic in-process fixture"},
			ProgressEvents:       CapabilitySupport{Level: CapabilityTestOnly, Mechanism: "deterministic in-process fixture"},
			PermissionIsolation:  CapabilitySupport{Level: CapabilityTestOnly, Mechanism: "no external process is launched"},
			NativeCommandSurface: CapabilitySupport{Level: CapabilityTestOnly, Mechanism: "not a user platform"},
		}, true
	default:
		return PlatformContract{}, false
	}
}

func (contract PlatformContract) canDispatchWorkers() bool {
	return contract.WorkerDispatch.Level != CapabilityUnavailable
}
