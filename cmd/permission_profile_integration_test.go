package cmd

import (
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

func TestBuildAndPlanningManifestsCarryCanonicalPermissionProfiles(t *testing.T) {
	build := []codexBuildDispatch{
		{Caste: "scout", Name: "Scout-1", Task: "inspect"},
		{Caste: "builder", Name: "Builder-1", Task: "implement"},
		{Caste: "probe", Name: "Probe-1", Task: "test"},
	}
	attachBuildDispatchContext(t.TempDir(), colony.Phase{ID: 1, Name: "Test"}, build, time.Now())
	for _, dispatch := range build {
		if dispatch.PermissionProfile.Name != codex.PermissionWorkspaceWrite {
			t.Fatalf("%s build profile = %+v", dispatch.Caste, dispatch.PermissionProfile)
		}
	}
	maps := codexBuildDispatchMaps(build)
	if _, ok := maps[0]["permission_profile"].(codex.PermissionProfile); !ok {
		t.Fatalf("build dispatch projection omitted typed permission profile: %#v", maps[0])
	}

	planning := plannedPlanningWorkersForGoal(t.TempDir(), "plan safely")
	if len(planning) < 2 {
		t.Fatalf("planning dispatches = %d, want at least 2", len(planning))
	}
	if planning[0].Caste != "scout" || planning[0].PermissionProfile.Name != codex.PermissionWorkspaceWrite {
		t.Fatalf("planning Scout profile = %+v", planning[0])
	}
	if planning[1].PermissionProfile.Name != codex.PermissionWorkspaceWrite {
		t.Fatalf("planning Route-Setter profile = %+v", planning[1].PermissionProfile)
	}
}
