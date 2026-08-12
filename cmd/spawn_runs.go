package cmd

import (
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
)

type runtimeSpawnRun struct {
	Tree *agent.SpawnTree
	Run  agent.SpawnRun
}

func beginRuntimeSpawnRun(command string, startedAt time.Time) (*runtimeSpawnRun, error) {
	if store == nil {
		return nil, nil
	}
	tree := agent.NewSpawnTree(store, "spawn-tree.txt")
	run, err := tree.BeginRun(command, startedAt)
	if err != nil {
		return nil, err
	}
	// SPAWN-08/D-15: reap ghosts automatically at the start of every run,
	// before any spawn decision consults the whole-run budget (cmd/spawn.go's
	// spawnCanSpawnDecision), so a fresh run never inherits a budget already
	// full of entries that will never complete. A reaping failure must never
	// stop a run from beginning -- its error is deliberately ignored here;
	// the operator's `aether spawn-orphans` command surfaces any that
	// remain.
	_, _ = spawnReapStaleEntries(tree, time.Now().UTC())
	return &runtimeSpawnRun{Tree: tree, Run: run}, nil
}

func finishRuntimeSpawnRun(handle *runtimeSpawnRun, status string, endedAt time.Time) {
	if handle == nil || handle.Tree == nil || handle.Run.ID == "" {
		return
	}
	_ = handle.Tree.EndRun(handle.Run.ID, status, endedAt)
}

func summarizeRunStatus(statuses ...string) string {
	status := "completed"
	for _, raw := range statuses {
		switch strings.TrimSpace(raw) {
		case "", "spawned", "starting", "active", "running":
			status = "active"
		case "blocked":
			return "blocked"
		case "failed", "timeout":
			if status != "blocked" {
				status = "failed"
			}
		}
	}
	return status
}
