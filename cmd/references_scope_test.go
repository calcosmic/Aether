package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Phase 180. On 2026-08-14 a worker running in CalVault -- an Obsidian notes
// vault containing no program code -- was handed the Runtime Boundary Contract,
// a document describing which parts of Aether are written in Go, which in
// TypeScript, and which in shell. Its whole job was to create twelve folders.
// That job cost 51,914 tokens.
//
// References are never copied into downstream repos; they are read from the
// shared hub, so every project on the machine reads the same library. A
// document about Aether's own construction is meaningless there and is charged
// for on every dispatch.
//
// These tests assert the scoping in both directions. A blanket removal would
// pass the first test and break the Aether repo's own workers, so the second
// test exists to make that failure loud.

const scopeTestInternalDoc = `---
schema_version: "1.0"
id: runtime-boundary-contract
kind: contract
category: contracts
scope: aether-internal
title: Runtime Boundary Contract
description: "Ownership boundaries between the Go runtime, the TypeScript host, and editable assets."
output_types: [boundary-review, architecture-review]
agent_roles: [architect, builder, watcher, queen, chronicler]
task_types: [boundary, contract, architecture]
task_keywords: [boundary, contract, go, typescript, runtime]
workflow_triggers: [plan, build, continue, seal]
priority: critical
version: "1.0"
render:
  mode: full
  max_chars: 4200
---
# Runtime Boundary Contract

Go owns execution. TypeScript owns orchestration.
`

const scopeTestUniversalDoc = `---
schema_version: "1.0"
id: worker-handoff-contract
kind: contract
category: contracts
title: Worker Handoff Contract
description: "The result shape every worker must return."
output_types: [boundary-review, architecture-review]
agent_roles: [architect, builder, watcher, queen, chronicler]
task_types: [boundary, contract, architecture]
task_keywords: [boundary, contract, handoff, worker]
workflow_triggers: [plan, build, continue, seal]
priority: critical
version: "1.0"
render:
  mode: full
  max_chars: 4200
---
# Worker Handoff Contract

Return changed files, commands run, and what not to repeat.
`

func writeScopeFixture(t *testing.T, root string) {
	t.Helper()
	dir := filepath.Join(root, "contracts")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir references: %v", err)
	}
	for name, body := range map[string]string{
		"runtime-boundary-contract.md": scopeTestInternalDoc,
		"worker-handoff-contract.md":   scopeTestUniversalDoc,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
}

func matchedReferenceIDs(t *testing.T, task string) []string {
	t.Helper()
	refs, _ := matchReferences(referenceMatchRequest{
		Role:     "builder",
		Workflow: "build",
		Task:     task,
		Limit:    5,
	})
	ids := make([]string, 0, len(refs))
	for _, ref := range refs {
		ids = append(ids, ref.Meta.ID)
	}
	return ids
}

// The CalVault task text, verbatim. A paraphrase would let the fix pass while
// the real case still failed.
const calVaultTask = "copy 110 markdown files into a new folder tree in an Obsidian vault"

func TestAetherInternalReferencesStayOutOfOtherProjects(t *testing.T) {
	saveGlobals(t)

	hub := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", hub)
	writeScopeFixture(t, filepath.Join(hub, "references"))

	// A project that is not Aether: no authored reference library of its own,
	// so the shared hub copy is what gets read.
	workDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldDir) })

	ids := matchedReferenceIDs(t, calVaultTask)

	for _, id := range ids {
		if id == "runtime-boundary-contract" {
			t.Fatalf("a worker in a non-Aether project was handed %q -- a document about Aether's own Go/TypeScript boundaries. Matched set: %v", id, ids)
		}
	}
}

func TestUniversalReferencesStillReachOtherProjects(t *testing.T) {
	saveGlobals(t)

	hub := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", hub)
	writeScopeFixture(t, filepath.Join(hub, "references"))

	workDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldDir) })

	ids := matchedReferenceIDs(t, "return a handoff after changing files")

	found := false
	for _, id := range ids {
		if id == "worker-handoff-contract" {
			found = true
		}
	}
	if !found {
		t.Fatalf("scoping removed a universal reference from a non-Aether project; this is a scoping fix, not a deletion. Matched set: %v", ids)
	}
}

func TestAetherInternalReferencesStillLoadInsideAether(t *testing.T) {
	saveGlobals(t)

	hub := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", hub)

	// The Aether repo authors its own reference library at .aether/references.
	// No other repo has one -- references are never copied downstream.
	workDir := t.TempDir()
	writeScopeFixture(t, filepath.Join(workDir, ".aether", "references"))

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldDir) })

	ids := matchedReferenceIDs(t, "change the boundary between the Go runtime and the TypeScript host")

	found := false
	for _, id := range ids {
		if id == "runtime-boundary-contract" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Aether's own workers lost an internal reference they need. Matched set: %v", ids)
	}
}

// The shipped library must actually carry the markers, or the scoping above is
// a mechanism with nothing wired to it -- the failure mode this repo documents
// more than any other.
func TestShippedInternalReferencesAreMarked(t *testing.T) {
	mustBeInternal := []string{
		"runtime-boundary-contract",
		"command-wrapper-contract",
		"platform-parity-contract",
		"agent-definition-parity-contract",
		"source-of-truth-contract",
		"update-safety-contract",
		"reference-library-contract",
		"prompt-context-contract",
		"worker-handoff-injection-contract",
		"queen-execution-policy-contract",
		"hub-channel-isolation-field-guide",
		"platform-surface-map",
		"skills-references-boundary-field-guide",
		"prompt-budget-trim-field-guide",
		// Found by running the real command from a non-Aether directory rather
		// than by reading the library: a builder asked to copy markdown files
		// into an Obsidian vault was handed all five of its documents from this
		// group, including the playbook for publishing Aether releases.
		"distribution-safety-rubric",
		"platform-parity-rubric",
		"codex-direct-cli-ux-playbook",
		"command-wrapper-generation-playbook",
		"publish-update-playbook",
		"reference-distribution-playbook",
		"source-check-mirror-drift-playbook",
		"stale-publish-diagnosis-playbook",
	}

	root := filepath.Join("..", ".aether", "references")
	refs := readReferencesFromRoot(root)
	if len(refs) == 0 {
		t.Fatalf("no references found at %s", root)
	}

	byID := map[string]referenceDocument{}
	for _, ref := range refs {
		byID[ref.Meta.ID] = ref
	}

	for _, id := range mustBeInternal {
		ref, ok := byID[id]
		if !ok {
			t.Errorf("%s: not found in the shipped library", id)
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(ref.Meta.Scope), referenceScopeAetherInternal) {
			t.Errorf("%s: describes Aether's own construction but is not marked `scope: aether-internal`, so it ships to every project on the machine", id)
		}
	}
}
