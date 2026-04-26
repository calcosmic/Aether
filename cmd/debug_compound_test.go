package cmd

import (
	"bytes"
	
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestDebugCompoundDestructive(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	_, dataDir := initRecoverTestStore(t)

	goal := "Destructive test colony"
	state := colony.ColonyState{
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Worktrees: []colony.WorktreeEntry{
			{
				ID:     "wt-1",
				Branch: "feature/test",
				Path:   "/tmp/nonexistent-wt-xyz",
				Status: colony.WorktreeAllocated,
			},
		},
		BuildStartedAt: recoverTimePtr(time.Now().Add(-2 * time.Hour)),
	}
	recoverWriteJSON(t, dataDir, "COLONY_STATE.json", state)
	recoverWriteFile(t, dataDir, "build/phase-1/manifest.json", "{broken")

	// Run repair and capture full output
	var buf bytes.Buffer
	stdout = &buf
	rootCmd.SetArgs([]string{"recover", "--apply", "--force", "--json"})
	err := rootCmd.Execute()
	
	fmt.Printf("Error: %v\n", err)
	fmt.Printf("Output:\n%s\n", buf.String())
	
	// Check the state of files after repair
	manifestPath := filepath.Join(dataDir, "build", "phase-1", "manifest.json")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		fmt.Println("Manifest was removed")
	} else {
		data, _ := os.ReadFile(manifestPath)
		fmt.Printf("Manifest still exists: %s\n", string(data))
	}
	
	stateData, _ := os.ReadFile(filepath.Join(dataDir, "COLONY_STATE.json"))
	fmt.Printf("State after repair:\n%s\n", string(stateData))
}
