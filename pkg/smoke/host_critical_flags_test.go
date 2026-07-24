package smoke

import (
	"testing"
)

func TestHostCriticalFlags(t *testing.T) {
	// Should contain exactly 6 host subcommand keys
	if len(HostCriticalFlags) != 6 {
		t.Errorf("expected exactly 6 host subcommand keys, got %d", len(HostCriticalFlags))
	}

	expectedKeys := []string{"ant-plan", "ant-build", "ant-continue", "ant-oracle", "ant-watch", "ant-swarm"}
	for _, key := range expectedKeys {
		if _, ok := HostCriticalFlags[key]; !ok {
			t.Errorf("expected host critical key %q to exist", key)
		}
	}
}

func TestHostCriticalCombinations(t *testing.T) {
	// Should contain at least the combinations from TCV-01 to TCV-05
	if len(HostCriticalCombinations) < 4 {
		t.Errorf("expected at least 4 host critical combinations, got %d", len(HostCriticalCombinations))
	}

	// TCV-01: plan --depth balanced --planning-depth standard
	planCombos := HostCriticalCombinations["ant-plan"]
	if len(planCombos) == 0 {
		t.Error("expected at least one plan combination")
	}

	// TCV-02: continue --verification-depth heavy
	continueCombos := HostCriticalCombinations["ant-continue"]
	if len(continueCombos) == 0 {
		t.Error("expected at least one continue combination")
	}

	// TCV-03: watch --no-dashboard
	watchCombos := HostCriticalCombinations["ant-watch"]
	if len(watchCombos) == 0 {
		t.Error("expected at least one watch combination")
	}

	// TCV-04: swarm --no-dashboard
	swarmCombos := HostCriticalCombinations["ant-swarm"]
	if len(swarmCombos) == 0 {
		t.Error("expected at least one swarm combination")
	}
}

func TestCrossReferenceWithTSHost(t *testing.T) {
	// Build a minimal yamlFlags map with flags that ARE in TS tests
	yamlFlags := map[string][]string{
		"ant-plan":     {"--depth", "--planning-depth"},
		"ant-continue": {"--verification-depth", "--light", "--heavy"},
		"ant-watch":    {"--no-dashboard"},
		"ant-swarm":    {"--no-dashboard"},
	}

	untested := CrossReferenceWithTSHost(yamlFlags)

	// --depth, --planning-depth, --verification-depth, --light, --heavy, --no-dashboard
	// are all covered by host-flags.test.ts and host-integration.test.ts
	// If any are missing, the test will surface it as a warning (not a failure here)
	for _, u := range untested {
		t.Logf("documented but untested flag: %s", u)
	}
}

func TestIsHostCritical(t *testing.T) {
	if !IsHostCritical("ant-plan") {
		t.Error("expected ant-plan to be host critical")
	}
	if !IsHostCritical("ant-build") {
		t.Error("expected ant-build to be host critical")
	}
	if IsHostCritical("ant-init") {
		t.Error("expected ant-init to NOT be host critical")
	}
}

func TestGetHostCommandGoName(t *testing.T) {
	if GetHostCommandGoName("ant-plan") != "plan" {
		t.Error("expected ant-plan to map to plan")
	}
	if GetHostCommandGoName("ant-build") != "build" {
		t.Error("expected ant-build to map to build")
	}
}
