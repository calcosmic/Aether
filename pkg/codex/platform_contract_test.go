package codex

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

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
		{PlatformCodex, "secondary", CapabilityProven, CapabilityLimited, CapabilityLimited, CapabilityLimited},
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

func TestCodexNativePlatformContract(t *testing.T) {
	contract, _ := PlatformContractFor(PlatformCodex)
	if contract.NativeWorkers == nil {
		t.Fatal("native support must be separately declared")
	}
	native := contract.NativeWorkers
	if native.Client != "codex-cli 0.154.0" || native.Evidence == "" || native.Dispatch.Level != CapabilityLimited || native.PerChildIsolation.Level != CapabilityUnavailable || native.WorktreeAllocation.Level != CapabilityUnavailable || native.GovernedNesting.Level != CapabilityUnavailable || native.Cancellation.Level != CapabilityUnavailable {
		t.Fatalf("native contract overstates observed support: %+v", native)
	}
	if !strings.Contains(native.Usage.Mechanism, "token_usage_record") || !strings.Contains(strings.Join(native.Usage.Limitations, " "), "uncollected") {
		t.Fatal("exposed provider usage must be distinct from uncollected measurements")
	}
	if !strings.Contains(contract.WorkerDispatch.Mechanism, "codex exec") || !strings.Contains(contract.PermissionIsolation.Mechanism, "sandbox mode") {
		t.Fatal("native claims replaced accurate subprocess claims")
	}
	for _, platform := range []Platform{PlatformClaude, PlatformOpenCode, PlatformFake} {
		other, _ := PlatformContractFor(platform)
		if other.NativeWorkers != nil {
			t.Fatalf("Codex native contract leaked to %s", platform)
		}
	}
	profile := PermissionProfileForCaste("builder")
	for _, tc := range []struct {
		name    string
		mutate  func(*NativeWorkerRequirements)
		allowed bool
	}{
		{"inherited-workspace", func(*NativeWorkerRequirements) {}, true},
		{"read-only-host", func(r *NativeWorkerRequirements) { r.HostPermission = "repository_read_only" }, false},
		{"other-workspace", func(r *NativeWorkerRequirements) { r.Workspace = "/other" }, false},
		{"worktree", func(r *NativeWorkerRequirements) { r.ParallelMode = "worktree" }, false},
		{"governed-nesting", func(r *NativeWorkerRequirements) { r.RequireGovernedNesting = true }, false},
		{"read-only-profile", func(r *NativeWorkerRequirements) { r.Profile = PermissionProfileForCaste("includer") }, false},
		{"narrow-write", func(r *NativeWorkerRequirements) { r.Profile.WriteScopes = []string{"tests"} }, false},
		{"shell-disabled", func(r *NativeWorkerRequirements) { r.Profile.Shell = "deny" }, false},
		{"network-disabled", func(r *NativeWorkerRequirements) { r.Profile.Network = "deny" }, false},
		{"approval-control", func(r *NativeWorkerRequirements) { r.Profile.Approval = "always" }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := NativeWorkerRequirements{Profile: profile, HostPermission: "workspace_write", Workspace: "/fixture", AcceptedWorkspace: "/fixture", ParallelMode: "in-repo"}
			tc.mutate(&r)
			if err := ValidateNativeWorkerRequirements(r); (err == nil) != tc.allowed {
				t.Fatalf("allowed=%v, err=%v", tc.allowed, err)
			}
		})
	}
}

func TestCodexAntSkillPlatformContract(t *testing.T) {
	contract, ok := PlatformContractFor(PlatformCodex)
	if !ok || contract.NativeCommandSurface.Level != CapabilityLimited {
		t.Fatal("nine bounded entry probes must declare a limited skill surface")
	}
	surface := contract.NativeCommandSurface
	for _, required := range []string{"dollar skills", "nine", "Codex CLI 0.154.0", "~/.codex/skills/aether", "direct aether CLI"} {
		if !strings.Contains(surface.Mechanism, required) {
			t.Errorf("missing surface qualification %q", required)
		}
	}
	for _, unsupported := range []string{"/ant-", "slash-command wrappers", "full parity", "all 64 commands"} {
		if strings.Contains(surface.Mechanism, unsupported) {
			t.Errorf("unsupported surface claim %q", unsupported)
		}
	}
	limits := strings.Join(surface.Limitations, "\n")
	for _, required := range []string{"first read-only command-guide step", "private support", "clean installation", "legacy update", "remaining 55", "native-helper", "unverified", "custom collisions"} {
		if !strings.Contains(limits, required) {
			t.Errorf("lost proof boundary %q", required)
		}
	}
	want := []string{"ant-init", "ant-discuss", "ant-oracle", "ant-colonize", "ant-plan", "ant-build", "ant-continue", "ant-swarm", "ant-seal"}
	names := regexp.MustCompile(`\$ant-[a-z-]+`).FindAllString(surface.Mechanism, -1)
	if len(names) != len(want) {
		t.Fatalf("declared skill count %d, want nine", len(names))
	}
	for i, name := range want {
		if names[i] != "$"+name {
			t.Errorf("surface name %d = %s, want %s", i, names[i], name)
		}
	}
	// Checked-in receipts keep this claim bound to the observed client/root and
	// complete clean/upgrade inventory. The cmd receipt validator independently
	// replays the durable raw tool events; this is not a substitute for that check.
	_, source, _, _ := runtime.Caller(0)
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(source), "..", "..", ".planning", "phases", "204.1-codex-ant-skill-surface", "evidence", "fresh-codex-skills.json"))
	if err != nil {
		t.Fatal(err)
	}
	var proof struct {
		Status string `json:"status"`
		Cases  []struct {
			Scenario  string `json:"scenario"`
			Name      string `json:"skill_name"`
			Outcome   string `json:"outcome"`
			Client    string `json:"client_version"`
			Root      string `json:"discovery_root"`
			Installed string `json:"installed_skill_sha256"`
			Loaded    string `json:"loaded_sha256"`
			Guide     string `json:"guide_event"`
			Support   string `json:"private_support_event"`
			Route     string `json:"route"`
			Exit      int    `json:"exit_status"`
			Missing   bool   `json:"missing_skill_control"`
			Wrong     bool   `json:"wrong_payload_control"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &proof); err != nil {
		t.Fatal(err)
	}
	if proof.Status != "qualified" {
		t.Fatal("surface lacks qualified live proof")
	}
	seen := map[string]bool{}
	for _, c := range proof.Cases {
		if c.Missing || c.Wrong {
			continue
		}
		key := c.Scenario + "/" + c.Name
		if seen[key] || c.Outcome != "qualified" || c.Exit != 0 || c.Client != "codex-cli 0.154.0" || !strings.HasSuffix(filepath.ToSlash(c.Root), "/.codex/skills/aether") || !regexp.MustCompile(`^sha256:[a-f0-9]{64}$`).MatchString(c.Loaded) || c.Installed != c.Loaded || c.Guide == "" || c.Support == "" || c.Route == "" {
			t.Fatalf("unqualified/duplicate proof for %s", key)
		}
		seen[key] = true
	}
	if len(seen) != 18 {
		t.Fatalf("positive proof cases = %d, want 18", len(seen))
	}
	for _, scenario := range []string{"clean", "upgrade"} {
		for _, name := range want {
			if !seen[scenario+"/"+name] {
				t.Errorf("missing %s/%s proof", scenario, name)
			}
		}
	}
	if contract.NamedCasteRouting.Level != CapabilityLimited || contract.PermissionIsolation.Level != CapabilityLimited || contract.ProgressEvents.Level != CapabilityLimited || contract.WorkerDispatch.Level != CapabilityProven || contract.SupportTier != "secondary" {
		t.Fatal("surface proof changed unrelated capability levels")
	}
}
