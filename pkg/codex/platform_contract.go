package codex

import "fmt"

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
	SchemaVersion        int                   `json:"schema_version"`
	Platform             Platform              `json:"platform"`
	SupportTier          string                `json:"support_tier"`
	StateLifecycle       CapabilitySupport     `json:"state_lifecycle"`
	WorkerDispatch       CapabilitySupport     `json:"worker_dispatch"`
	NamedCasteRouting    CapabilitySupport     `json:"named_caste_routing"`
	StructuredCompletion CapabilitySupport     `json:"structured_completion"`
	ProgressEvents       CapabilitySupport     `json:"progress_events"`
	PermissionIsolation  CapabilitySupport     `json:"permission_isolation"`
	NativeCommandSurface CapabilitySupport     `json:"native_command_surface"`
	NativeWorkers        *NativeWorkerContract `json:"native_workers,omitempty"`
}

// NativeWorkerContract describes the observed host-native route separately
// from the Go-owned subprocess adapter. Limited observations are not parity.
type NativeWorkerContract struct {
	Client             string            `json:"client"`
	Evidence           string            `json:"evidence"`
	Dispatch           CapabilitySupport `json:"dispatch"`
	PerChildIsolation  CapabilitySupport `json:"per_child_isolation"`
	WorktreeAllocation CapabilitySupport `json:"worktree_allocation"`
	GovernedNesting    CapabilitySupport `json:"governed_nesting"`
	Cancellation       CapabilitySupport `json:"cancellation"`
	Usage              CapabilitySupport `json:"usage"`
}

func codexNativeWorkerContract() *NativeWorkerContract {
	return &NativeWorkerContract{
		Client:             "codex-cli 0.154.0",
		Evidence:           ".planning/phases/204.2-codex-native-worker-lifecycle/evidence/native-capabilities.json",
		Dispatch:           CapabilitySupport{Level: CapabilityLimited, Mechanism: "actual host spawn_agent and child-attributed work observed in disposable accepted fixtures with inherited workspace-write; the Go bridge does not launch", Limitations: []string{"original eleven-scenario qualification is incomplete; no complete native lifecycle parity claim", "host-encrypted message exports establish call/child linkage, not exact plaintext delivery", "model and effort are inherited parent invocation preferences, not independently attested per-child controls"}},
		PerChildIsolation:  CapabilitySupport{Level: CapabilityUnavailable, Mechanism: "no qualified per-child permission selector; only the same inherited workspace-write envelope is admitted", Limitations: []string{"inherited parent sandbox observations do not prove per-child read-only, narrow write scope, shell or network restrictions"}},
		WorktreeAllocation: CapabilitySupport{Level: CapabilityUnavailable, Mechanism: "native admission requires the accepted shared workspace; separate native worktree allocation is unsupported"},
		GovernedNesting:    CapabilitySupport{Level: CapabilityUnavailable, Mechanism: "a real nested host helper was observed, but Aether-governed native recruitment is unqualified"},
		Cancellation:       CapabilitySupport{Level: CapabilityUnavailable, Mechanism: "no confirmed cancelled terminal observed", Limitations: []string{"interrupt, close, release, idle and process loss are not confirmed terminal cancellation"}},
		Usage:              CapabilitySupport{Level: CapabilityLimited, Mechanism: "raw child-attributed token_usage_record events expose thread, parent session, turn and response identities", Limitations: []string{"native worker usage is uncollected by Aether; empty saved Usage remains unreported", "no provider cost is collected; absent measurement is not zero cost", "worker prose, parent totals and submitted result usage are not trusted provider measurements"}},
	}
}

// NativeWorkerRequirements holds actual requested controls, not a caller's
// assertion that the host enforces them. Refusal never switches execution lanes.
type NativeWorkerRequirements struct {
	Profile                PermissionProfile
	HostPermission         string
	Workspace              string
	AcceptedWorkspace      string
	ParallelMode           string
	RequireGovernedNesting bool
}

func ValidateNativeWorkerRequirements(request NativeWorkerRequirements) error {
	contract := codexNativeWorkerContract()
	if request.RequireGovernedNesting && contract.GovernedNesting.Level == CapabilityUnavailable {
		return fmt.Errorf("native host cannot provide requested Aether-governed nesting; no launch authorized")
	}
	if request.Workspace == "" || request.Workspace != request.AcceptedWorkspace || request.HostPermission != string(PermissionWorkspaceWrite) || (request.ParallelMode != "" && request.ParallelMode != "in-repo") {
		return fmt.Errorf("native host must use the accepted shared workspace and inherited workspace_write permission; requested workspace or permission is unsupported; no automatic route change")
	}
	if !permissionEnforcementEqual(request.Profile, PermissionProfileForCaste("builder")) {
		return fmt.Errorf("native host cannot enforce this requested permission profile; no launch authorized")
	}
	return nil
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
			NativeWorkers:        codexNativeWorkerContract(),
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
			NativeCommandSurface: CapabilitySupport{
				Level:     CapabilityLimited,
				Mechanism: "Codex CLI 0.154.0 discovers nine installed dollar skills at ~/.codex/skills/aether: $ant-init, $ant-discuss, $ant-oracle, $ant-colonize, $ant-plan, $ant-build, $ant-continue, $ant-swarm, $ant-seal; direct aether CLI commands remain available",
				Limitations: []string{
					"proof covers fresh-session selection/loading, private support reads, and the first read-only command-guide step after clean installation and supported legacy update",
					"the remaining 55 actions, complete workflow parity, and native-helper behavior are unverified by this surface proof",
					"other client versions/discovery roots and preserved custom collisions are outside the qualification",
				},
			},
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
