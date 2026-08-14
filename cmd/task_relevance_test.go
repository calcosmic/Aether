package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Phase 181. What a worker is given to read must be decided by the task, not by
// its job title.
//
// Measured before this phase: a builder asked to copy markdown files into an
// Obsidian vault received five reference documents, every one scored on job
// title and workflow with zero task relevance, and a skill section containing
// an AI design contract covering model selection, RAG chunking and HIPAA. The
// scoring weighted expected-output-type at 4 and job title at 3 while task
// relevance was worth 2 -- the lowest signal in the system. Task matching was
// also close to random: it stripped every space from both sides and did a
// substring compare, so "folder" contained "old".

func writeRefDoc(t *testing.T, root, filename, body string) {
	t.Helper()
	dir := filepath.Join(root, "contracts")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(body), 0644); err != nil {
		t.Fatalf("write %s: %v", filename, err)
	}
}

func refDoc(id, title string, roles, outputs, keywords []string) string {
	list := func(v []string) string { return "[" + strings.Join(v, ", ") + "]" }
	return "---\n" +
		"schema_version: \"1.0\"\n" +
		"id: " + id + "\n" +
		"kind: contract\n" +
		"category: contracts\n" +
		"title: " + title + "\n" +
		"description: \"Fixture.\"\n" +
		"output_types: " + list(outputs) + "\n" +
		"agent_roles: " + list(roles) + "\n" +
		"task_types: []\n" +
		"task_keywords: " + list(keywords) + "\n" +
		"workflow_triggers: [build]\n" +
		"priority: normal\n" +
		"version: \"1.0\"\n" +
		"render:\n  mode: full\n  max_chars: 1000\n" +
		"---\n# " + title + "\n\nBody.\n"
}

func inTempProject(t *testing.T) string {
	t.Helper()
	hub := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", hub)
	workDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldDir) })
	return filepath.Join(hub, "references")
}

func TestReferenceSelectionRequiresTaskRelevance(t *testing.T) {
	saveGlobals(t)
	root := inTempProject(t)

	// Matches the worker's job title and the workflow. Says nothing about the task.
	writeRefDoc(t, root, "ai-design-contract.md",
		refDoc("ai-design-contract", "AI Design Contract",
			[]string{"builder", "architect"}, []string{"architecture-review"},
			[]string{"llm", "rag", "embedding", "prompt"}))

	refs, _ := matchReferences(referenceMatchRequest{
		Role: "builder", Workflow: "build", Task: calVaultTask, Limit: 5,
	})

	if len(refs) != 0 {
		ids := []string{}
		for _, r := range refs {
			ids = append(ids, r.Meta.ID+" ("+strings.Join(r.Reasons, ",")+")")
		}
		t.Fatalf("a document matching only job title and workflow was still selected for a file-copying task: %v", ids)
	}
}

func TestTaskRelevanceOutranksRoleAndOutputType(t *testing.T) {
	saveGlobals(t)
	root := inTempProject(t)

	// Every non-task signal.
	writeRefDoc(t, root, "role-and-output.md",
		refDoc("role-and-output", "Role And Output",
			[]string{"builder"}, []string{"code-review"}, []string{"unrelated"}))
	// Task relevance only -- wrong job title, wrong output type.
	writeRefDoc(t, root, "task-only.md",
		refDoc("task-only", "Task Only",
			[]string{"oracle"}, []string{"research-brief"}, []string{"markdown"}))

	refs, _ := matchReferences(referenceMatchRequest{
		Role: "builder", Workflow: "build", OutputType: "code-review",
		Task: calVaultTask, Limit: 5,
	})

	if len(refs) == 0 {
		t.Fatal("expected the task-relevant document to be selected")
	}
	if refs[0].Meta.ID != "task-only" {
		ids := []string{}
		for _, r := range refs {
			ids = append(ids, r.Meta.ID)
		}
		t.Fatalf("job title and output type outranked task relevance; order was %v, want task-only first", ids)
	}
}

func TestNoTaskMatchYieldsNoReferences(t *testing.T) {
	saveGlobals(t)
	root := inTempProject(t)

	writeRefDoc(t, root, "a.md", refDoc("a", "A", []string{"builder"}, []string{"code-review"}, []string{"database"}))
	writeRefDoc(t, root, "b.md", refDoc("b", "B", []string{"builder"}, []string{"code-review"}, []string{"kubernetes"}))

	refs, _ := matchReferences(referenceMatchRequest{
		Role: "builder", Workflow: "build", OutputType: "code-review",
		Task: "rename a column in a spreadsheet", Limit: 5,
	})

	if len(refs) != 0 {
		t.Fatalf("a task matching nothing should produce an empty reading section, got %d documents", len(refs))
	}
}

// A skill or reference must not be able to win a slot by being renamed. This
// was found during the skill-authoring review: alphabetical position was a
// selection input, so `aaa-my-notes` could displace a shipped document.
func TestRenamingAReferenceDoesNotChangeSelection(t *testing.T) {
	saveGlobals(t)

	order := func(firstID, secondID string) []string {
		root := inTempProject(t)
		writeRefDoc(t, root, firstID+".md",
			refDoc(firstID, "First", []string{"builder"}, []string{"code-review"}, []string{"markdown"}))
		writeRefDoc(t, root, secondID+".md",
			refDoc(secondID, "Second", []string{"builder"}, []string{"code-review"}, []string{"markdown"}))
		refs, _ := matchReferences(referenceMatchRequest{
			Role: "builder", Workflow: "build", OutputType: "code-review",
			Task: calVaultTask, Limit: 5,
		})
		titles := []string{}
		for _, r := range refs {
			titles = append(titles, r.Meta.Title)
		}
		return titles
	}

	// Identical content and scores; only the identifiers differ in sort order.
	before := order("mmm-first", "zzz-second")
	after := order("aaa-first", "bbb-second")

	if len(before) != 2 || len(after) != 2 {
		t.Fatalf("expected both documents selected in both runs, got %v and %v", before, after)
	}
	if before[0] != after[0] || before[1] != after[1] {
		t.Fatalf("renaming changed selection order: %v then %v -- filename must not be a selection input", before, after)
	}
}

func TestSkillNeedsMoreThanRoleAndWorkflow(t *testing.T) {
	roleAndWorkflowOnly := []skillMatchReason{
		{Code: "role_match", Score: 3},
		{Code: "workflow_trigger", Score: 4},
	}
	if skillHasTaskOrWorkspaceEvidence(roleAndWorkflowOnly) {
		t.Fatal("a skill matching only the worker's job title and the workflow counts as evidence; that is how an AI design contract reached a job that creates folders")
	}

	withTask := append(append([]skillMatchReason{}, roleAndWorkflowOnly...),
		skillMatchReason{Code: "task_keyword", Score: 2, Evidence: []string{"markdown"}})
	if !skillHasTaskOrWorkspaceEvidence(withTask) {
		t.Fatal("task keyword evidence should qualify a skill")
	}

	withWorkspace := append(append([]skillMatchReason{}, roleAndWorkflowOnly...),
		skillMatchReason{Code: "workspace_file", Score: 2, Evidence: []string{"go.mod"}})
	if !skillHasTaskOrWorkspaceEvidence(withWorkspace) {
		t.Fatal("workspace evidence should qualify a skill")
	}
}

// A skill cut in half mid-sentence is worse than no skill: the worker pays for
// the tokens and receives something incoherent. Observed in a live dispatch,
// where a composed brief ended "If the prompt is large or [truncated]".
func TestOversizedSkillIsDroppedNotTruncated(t *testing.T) {
	tmpDir := t.TempDir()

	bigPath := filepath.Join(tmpDir, "BIG.md")
	if err := os.WriteFile(bigPath, []byte("---\nname: oversized\n---\n"+strings.Repeat("skill guidance ", 900)), 0644); err != nil {
		t.Fatalf("write big skill: %v", err)
	}
	smallPath := filepath.Join(tmpDir, "SMALL.md")
	if err := os.WriteFile(smallPath, []byte("---\nname: compact\n---\nShort and relevant."), 0644); err != nil {
		t.Fatalf("write small skill: %v", err)
	}

	result := renderSkillInjectResultWithBudget(skillMatchResult{
		Role: "builder",
		ColonySkills: []skillResolvedEntry{
			{skillIndexEntry: skillIndexEntry{Name: "oversized", Path: bigPath, Type: "colony"}, Score: 100},
			{skillIndexEntry: skillIndexEntry{Name: "compact", Path: smallPath, Type: "colony"}, Score: 50},
		},
	}, 2000)

	if strings.Contains(result.Section, "[truncated]") {
		t.Fatalf("skill section was cut mid-sentence rather than dropping the oversized skill:\n%s", result.Section)
	}
	if len(result.Section) > 2000 {
		t.Fatalf("section length %d exceeds the budget", len(result.Section))
	}
	// The oversized skill is skipped, but a later one that fits must still land --
	// otherwise one big skill silently starves every skill behind it.
	if !strings.Contains(result.Section, "Short and relevant.") {
		t.Fatalf("a skill that fits was dropped because an earlier one did not:\n%s", result.Section)
	}
}
