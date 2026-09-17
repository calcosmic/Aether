package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// This consumer captures exactly the serialized payload the native host reads.
// It deliberately never calls the prompt composer or reconstructs its sections.
func nativeLaunchPayloadForTest(t *testing.T, request codexNativeWorkerRequest) (string, codexNativeWorkerResponse) {
	t.Helper()
	response, err := runCodexNativeWorker("reserve", nativeRequestPath(t, request))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	var downstream codexNativeWorkerResponse
	if err := json.Unmarshal(raw, &downstream); err != nil {
		t.Fatal(err)
	}
	if !downstream.LaunchAllowed || downstream.Worker == nil {
		t.Fatal("host received no launch payload")
	}
	return downstream.Worker.Native.Prompt, downstream
}

func nativeContextFixture(t *testing.T) (codexBuildManifest, codexNativeWorkerRequest, string) {
	t.Helper()
	root := setupExternalBuildAttemptTest(t)
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatal(err)
	}
	*state.Goal += " CAPSULE_NATIVE_MARKER"
	state.Plan.Phases[0].Description = "BRIEF_NATIVE_MARKER\n exact \"quote\" café 日本語 é\r\nEnd"
	state.Memory.Decisions = []colony.Decision{{ID: "context-marker", Phase: 1, Claim: "CAPSULE_NATIVE_MARKER", Rationale: "Carry precise worker context", Timestamp: time.Now().UTC().Format(time.RFC3339)}}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}
	now, old := time.Now().UTC().Format(time.RFC3339), time.Now().Add(-48*time.Hour).UTC().Format(time.RFC3339)
	strength, quarantined := 1.0, true
	signals := []colony.PheromoneSignal{}
	for _, name := range []string{"FOCUS_NATIVE_MARKER", "REDIRECT_NATIVE_MARKER", "EXPIRED_NATIVE_MARKER", "REVOKED_NATIVE_MARKER", "QUARANTINED_NATIVE_MARKER"} {
		raw, _ := json.Marshal(map[string]string{"text": name})
		signal := colony.PheromoneSignal{ID: name, Type: "FOCUS", Priority: "normal", Source: "user", CreatedAt: now, Active: true, Strength: &strength, Content: raw}
		switch name {
		case "REDIRECT_NATIVE_MARKER":
			signal.Type, signal.Priority = "REDIRECT", "high"
		case "EXPIRED_NATIVE_MARKER":
			signal.ExpiresAt = &old
		case "REVOKED_NATIVE_MARKER":
			signal.RevokedAt = &now
		case "QUARANTINED_NATIVE_MARKER":
			signal.Quarantined = &quarantined
		}
		signals = append(signals, signal)
	}
	if err := store.SaveJSON("pheromones.json", colony.PheromoneFile{Signals: signals}); err != nil {
		t.Fatal(err)
	}
	hub := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", hub)
	skill := filepath.Join(hub, "system", "skills", "colony", "native-context")
	if err := os.MkdirAll(skill, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte("---\nname: native-context\ntype: colony\nagent_roles:\n  - builder\n---\nSKILL_NATIVE_MARKER\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := persistDispatchWorkerHandoff(codex.WorkerDispatch{WorkerName: "Prior", Caste: "builder", TaskID: "0.1", Workflow: "build", Phase: 1, Root: root}, codex.DispatchResult{WorkerName: "Prior", Status: "completed", WorkerResult: &codex.WorkerResult{WorkerName: "Prior", Caste: "builder", TaskID: "0.1", Status: "completed", Summary: "Prior concrete work", Handoff: codex.WorkerHandoff{VerificationStatus: "pass", NextWorkerInstructions: []string{"HANDOFF_NATIVE_MARKER"}, Freshness: now}}}); err != nil {
		t.Fatal(err)
	}
	answer, err := recordDecisionAnswer("Which detail?", "ANSWER_NATIVE_MARKER", 1, "native-context-test")
	if err != nil {
		t.Fatal(err)
	}
	manifest := prepareBoundBuildManifestOnly(t, root)
	canonicalRoot, _ := filepath.EvalSymlinks(root)
	d := manifest.Dispatches[0]
	return manifest, codexNativeWorkerRequest{SchemaVersion: 1, Phase: 1, ExecutionBinding: *manifest.ExecutionBinding, WorkerName: d.Name, TaskID: normalizedDispatchTaskID(d), HostSessionID: "native-context-host", Workspace: canonicalRoot, HostPermission: "workspace_write"}, answer.ID
}

func TestCodexNativeContextDelivered(t *testing.T) {
	manifest, request, _ := nativeContextFixture(t)
	prompt, response := nativeLaunchPayloadForTest(t, request)
	for _, marker := range []string{"CAPSULE_NATIVE_MARKER", "BRIEF_NATIVE_MARKER", "SKILL_NATIVE_MARKER", "FOCUS_NATIVE_MARKER", "REDIRECT_NATIVE_MARKER", "HANDOFF_NATIVE_MARKER", "- Which detail? => ANSWER_NATIVE_MARKER"} {
		if got := strings.Count(prompt, marker); got != 1 {
			t.Errorf("downstream prompt has %s %d times; want exactly once", marker, got)
		}
	}
	for _, marker := range []string{"EXPIRED_NATIVE_MARKER", "REVOKED_NATIVE_MARKER", "QUARANTINED_NATIVE_MARKER"} {
		if strings.Contains(prompt, marker) {
			t.Errorf("ineffective steering delivered: %s", marker)
		}
	}
	brief, err := os.ReadFile(filepath.Join(manifest.Root, manifest.Dispatches[0].BriefPath))
	if err != nil {
		t.Fatal(err)
	}
	capsuleAt, briefAt, skillAt := strings.Index(prompt, manifest.ContextCapsule), strings.Index(prompt, string(brief)), strings.Index(prompt, manifest.Dispatches[0].SkillSection)
	if capsuleAt < 0 || briefAt <= capsuleAt || skillAt <= briefAt {
		t.Fatal("downstream context sections were dropped or reordered")
	}
	if response.Worker.Native.PromptSHA256 != lifecycleDigest([]byte(prompt)) {
		t.Fatal("launch digest does not hash delivered bytes")
	}
}

func TestCodexNativeContextBriefIdentity(t *testing.T) {
	for _, change := range []string{"changed", "missing", "directory"} {
		t.Run(change, func(t *testing.T) {
			manifest, request := nativeAdmissionFixture(t)
			path := filepath.Join(manifest.Root, manifest.Dispatches[0].BriefPath)
			before := nativeJournalBytes(t)
			switch change {
			case "changed":
				if err := os.WriteFile(path, []byte("substituted brief"), 0600); err != nil {
					t.Fatal(err)
				}
			case "missing":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := runCodexNativeWorker("reserve", nativeRequestPath(t, request)); err == nil {
				t.Fatal("required brief identity mismatch was admitted")
			}
			if !bytes.Equal(before, nativeJournalBytes(t)) {
				t.Fatal("brief refusal changed journal")
			}
		})
	}
}

func TestCodexNativeContextAnswerDedup(t *testing.T) {
	manifest, request, answerID := nativeContextFixture(t)
	newer, err := recordDecisionAnswer("Which next detail?", "NEWER_ANSWER_NATIVE_MARKER", 1, "native-context-test")
	if err != nil {
		t.Fatal(err)
	}
	prompt, response := nativeLaunchPayloadForTest(t, request)
	if strings.Count(prompt, "- Which detail? => ANSWER_NATIVE_MARKER") != 1 || strings.Count(prompt, "NEWER_ANSWER_NATIVE_MARKER") != 1 {
		t.Fatal("capsule answer duplicated or new answer dropped")
	}
	raw, _ := json.Marshal(manifest)
	var metadata map[string]any
	_ = json.Unmarshal(raw, &metadata)
	if !reflect.DeepEqual(metadata["context_decision_ids"], []any{answerID}) {
		t.Fatalf("manifest lacks exact included answer identity: %v", metadata["context_decision_ids"])
	}
	raw, _ = json.Marshal(response.Worker.Native)
	_ = json.Unmarshal(raw, &metadata)
	if !reflect.DeepEqual(metadata["context_decision_ids"], []any{answerID, newer.ID}) {
		t.Fatalf("launch lacks included answer identity: %v", metadata["context_decision_ids"])
	}
	if strings.Index(prompt, "NEWER_ANSWER_NATIVE_MARKER") < strings.Index(prompt, "SKILL_NATIVE_MARKER") {
		t.Fatal("new answer must follow skills")
	}
}

func TestCodexNativeContextBounds(t *testing.T) {
	_, request := nativeAdmissionFixture(t)
	path := nativeRequestPath(t, request)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	exact := append(append([]byte(nil), raw...), bytes.Repeat([]byte(" "), internalWorkerRequestMaxBytes-len(raw))...)
	if err := os.WriteFile(path, exact, 0600); err != nil {
		t.Fatal(err)
	}
	before := nativeJournalBytes(t)
	if _, err := loadCodexNativeWorkerRequest(path); err != nil {
		t.Fatalf("valid exact 2 MiB request refused: %v", err)
	}
	for _, bad := range [][]byte{append(append([]byte(nil), exact...), ' '), raw[:len(raw)-1], append(append([]byte(nil), raw...), []byte("{}")...)} {
		if err := os.WriteFile(path, bad, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := runCodexNativeWorker("reserve", path); err == nil {
			t.Fatal("oversized/partial/extra JSON request admitted")
		}
		if !bytes.Equal(before, nativeJournalBytes(t)) {
			t.Fatal("invalid request wrote state")
		}
	}
	if err := os.WriteFile(path, exact, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runCodexNativeWorker("reserve", path); err != nil {
		t.Fatalf("exact 2 MiB request not admitted: %v", err)
	}
}

func TestCodexNativeContextExactBytes(t *testing.T) {
	manifest, request, _ := nativeContextFixture(t)
	brief, err := os.ReadFile(filepath.Join(manifest.Root, manifest.Dispatches[0].BriefPath))
	if err != nil {
		t.Fatal(err)
	}
	prompt, response := nativeLaunchPayloadForTest(t, request)
	if !strings.Contains(prompt, string(brief)) || !strings.Contains(prompt, "café 日本語 é\r\n") {
		t.Fatal("launch normalized exact multiline/Unicode/quote brief bytes")
	}
	if lifecycleDigest([]byte(prompt)) != response.Worker.Native.PromptSHA256 {
		t.Fatal("wrong prompt digest")
	}
}

func TestCodexNativeContextOptionalAndFallback(t *testing.T) {
	root := t.TempDir()
	brief := "\n exact \"quoted\" café 日本語 é\r\ntrailing spaces  \n"
	d := codexBuildDispatch{Name: "Fixture", Caste: "builder", TaskID: "1.1", Brief: brief}
	manifest := codexBuildManifest{Root: root}
	// No colony store: optional context and decisions really are absent.
	saveGlobals(t)
	store = nil
	prompt, err := composeCodexNativePrompt(manifest, d, "fixture-launch")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(prompt.Prompt, brief) || prompt.SHA256 != lifecycleDigest([]byte(prompt.Prompt)) {
		t.Fatal("inline fallback bytes changed")
	}
	d.Brief = ""
	if _, err := composeCodexNativePrompt(manifest, d, "fixture-launch"); err == nil {
		t.Fatal("missing required brief admitted")
	}
	d.Brief = string([]byte{0xff})
	if _, err := composeCodexNativePrompt(manifest, d, "fixture-launch"); err == nil {
		t.Fatal("invalid UTF-8 would be normalized on the JSON boundary")
	}
}
