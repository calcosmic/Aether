package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

func TestTerritoryLifecycleFreshPassThrough(t *testing.T) {
	root, _ := setupTerritoryLifecycle199(t)
	now := time.Now().UTC().Add(-time.Minute)
	want := writeFreshTerritorySnapshot199(t, root, now)

	freshness, refresh, err := territoryPlanPreflight(root, codexPlanOptions{PlanOnly: true})
	if err != nil {
		t.Fatalf("territoryPlanPreflight: %v", err)
	}
	if refresh != nil {
		t.Fatalf("fresh territory produced %d survey dispatches; want zero", len(refresh.Dispatches))
	}
	if freshness.Freshness != colony.SurveyFreshnessFresh || freshness.SnapshotID != want.SnapshotID {
		t.Fatalf("freshness = %#v, want snapshot %s", freshness, want.SnapshotID)
	}

	manifest := codexPlanManifest{}
	attachTerritoryToPlanManifest(&manifest, freshness)
	if manifest.Territory.SnapshotID != want.SnapshotID || manifest.Territory.SourceRevision != want.SourceRevision {
		t.Fatalf("plan manifest did not receive exact snapshot evidence: %#v", manifest.Territory)
	}
	if !equalStringMaps199(manifest.Territory.ArtifactDigests, want.ArtifactDigests) {
		t.Fatalf("plan digest set changed during pass-through\ngot:  %v\nwant: %v", manifest.Territory.ArtifactDigests, want.ArtifactDigests)
	}
}

func TestTerritoryLifecycleAutomaticRefresh(t *testing.T) {
	root, _ := setupTerritoryLifecycle199(t)
	missing := ensureTerritoryFreshness(root)
	if missing.Freshness != colony.SurveyFreshnessMissing {
		t.Fatalf("initial freshness = %q, want missing", missing.Freshness)
	}

	freshness, refresh, err := territoryPlanPreflight(root, codexPlanOptions{PlanOnly: true})
	if err != nil {
		t.Fatalf("territoryPlanPreflight: %v", err)
	}
	if freshness.Freshness != colony.SurveyFreshnessMissing || refresh == nil {
		t.Fatalf("preflight = %#v, refresh=%#v; want missing plus automatic manifest", freshness, refresh)
	}
	if refresh.TransactionID == "" || refresh.BaselineDigest == "" || refresh.PublicationMode != territoryPublicationTransactional {
		t.Fatalf("refresh manifest lacks transaction binding: %#v", refresh)
	}
	if len(refresh.Dispatches) != len(surveyorSpecs) {
		t.Fatalf("automatic refresh manifest has %d surveyor dispatches, want complete roster %d", len(refresh.Dispatches), len(surveyorSpecs))
	}
	declaredOutputs := make(map[string]bool, len(requiredSurveyMarkdownFiles))
	for _, dispatch := range refresh.Dispatches {
		for _, output := range dispatch.Outputs {
			declaredOutputs[output] = true
		}
		for _, output := range dispatch.OutputPaths {
			if !strings.HasPrefix(output, refresh.CandidateSurveyDir+"/") {
				t.Fatalf("dispatch output %q is outside candidate survey dir %q", output, refresh.CandidateSurveyDir)
			}
		}
	}
	for _, output := range requiredSurveyMarkdownFiles {
		if !declaredOutputs[output] {
			t.Fatalf("automatic refresh omitted worker-authored survey output %s", output)
		}
	}
	if err := validateTerritoryPublicationMarkdown([]byte("# Survey\n\nplaceholder\n")); err == nil {
		t.Fatal("placeholder survey content was accepted for publication")
	}

	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	completed := make([]codexSurveyorDispatch, 0, len(refresh.Dispatches))
	for _, dispatch := range refresh.Dispatches {
		if err := spawnTree.RecordSpawn("Queen", dispatch.Caste, dispatch.Name, dispatch.Task, 1); err != nil {
			t.Fatalf("record survey spawn: %v", err)
		}
		dispatch.Status = "completed"
		dispatch.Summary = "candidate survey complete"
		for _, rel := range dispatch.OutputPaths {
			path := filepath.Join(root, filepath.FromSlash(rel))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatalf("mkdir candidate output: %v", err)
			}
			if err := os.WriteFile(path, []byte("# "+filepath.Base(path)+"\n\nautomatic refresh\n"), 0o644); err != nil {
				t.Fatalf("write candidate output: %v", err)
			}
			dispatch.FilesCreated = append(dispatch.FilesCreated, rel)
		}
		if err := spawnTree.UpdateStatus(dispatch.Name, "completed", dispatch.Summary); err != nil {
			t.Fatalf("complete survey spawn: %v", err)
		}
		completed = append(completed, dispatch)
	}

	finalized, err := runCodexColonizeFinalize(root, codexExternalColonizeCompletion{
		ColonizeManifest: refresh,
		Dispatches:       completed,
	})
	if err != nil {
		t.Fatalf("runCodexColonizeFinalize: %v", err)
	}
	publication, ok := finalized["territory_freshness"].(SurveyFreshnessResult)
	if !ok || publication.Freshness != colony.SurveyFreshnessFresh || !publication.Refreshed {
		t.Fatalf("publication freshness = %#v, want refreshed/fresh", finalized["territory_freshness"])
	}
	for _, rel := range requiredTerritoryArtifactPaths() {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("published artifact %s: %v", rel, err)
		}
	}
}

func TestTerritoryLifecycleUnavailableStops(t *testing.T) {
	root, dataDir := setupTerritoryLifecycle199(t)
	writeFreshTerritorySnapshot199(t, root, time.Now().UTC().Add(-time.Minute))
	if err := os.WriteFile(filepath.Join(root, territorySnapshotRelativePath), []byte("{tampered"), 0o644); err != nil {
		t.Fatalf("tamper snapshot: %v", err)
	}
	before := snapshotTerritoryData199(t, root)

	freshness, refresh, err := territoryPlanPreflight(root, codexPlanOptions{PlanOnly: true})
	if err == nil {
		t.Fatal("unavailable territory did not stop planning")
	}
	if freshness.Freshness != colony.SurveyFreshnessUnavailable || refresh != nil {
		t.Fatalf("freshness = %#v, refresh=%#v; want unavailable and no manifest", freshness, refresh)
	}
	if !strings.Contains(err.Error(), "aether resume") || freshness.SourceError == "" {
		t.Fatalf("unavailable error lacks source evidence and recovery command: %v / %#v", err, freshness)
	}
	after := snapshotTerritoryData199(t, root)
	if fmt.Sprint(after) != fmt.Sprint(before) {
		t.Fatalf("unavailable preflight mutated lifecycle data\nbefore: %#v\nafter:  %#v", before, after)
	}
	if _, statErr := os.Stat(filepath.Join(dataDir, "spawn-tree.txt")); !os.IsNotExist(statErr) {
		t.Fatalf("unavailable preflight fabricated survey activity: %v", statErr)
	}
}

func TestTerritoryLifecycleTransactionalPublish(t *testing.T) {
	targetCount := len(requiredTerritoryArtifactPaths()) + 2 // snapshot metadata + colony state
	for target := 1; target <= targetCount; target++ {
		t.Run(fmt.Sprintf("target_%02d", target), func(t *testing.T) {
			root, dataDir := setupTerritoryLifecycle199(t)
			writeFreshTerritorySnapshot199(t, root, time.Now().UTC().Add(-time.Hour))
			before := readTerritorySnapshotBytes199(t, root)
			transactionID := fmt.Sprintf("territory-fault-%02d", target)
			faultPoint := fmt.Sprintf("after_target_commit:target-%04d", target)

			_, err := publishTerritorySnapshot(territoryPublicationRequest{
				Root:           root,
				GeneratedAt:    time.Now().UTC(),
				TransactionID:  transactionID,
				BaselineDigest: territorySurveyBaselineDigest(root),
				Artifacts:      territoryArtifactFixture199("replacement"),
				Fault: func(point string) error {
					if point == faultPoint {
						return errors.New("injected territory publication fault")
					}
					return nil
				},
			})
			if err == nil {
				t.Fatalf("fault at %s did not interrupt publication", faultPoint)
			}

			after := readTerritorySnapshotBytes199(t, root)
			journal := filepath.Join(dataDir, "transactions", transactionID, "intent.json")
			_, journalErr := os.Stat(journal)
			if !equalByteMaps199(after, before) && journalErr != nil {
				t.Fatalf("partial publication changed the prior snapshot without recoverable journal %s: %v", journal, journalErr)
			}
		})
	}
}

func TestTerritoryLifecyclePlanRequiresSnapshot(t *testing.T) {
	root, _ := setupTerritoryLifecycle199(t)
	fresh := ensureTerritoryFreshness(root)
	if fresh.Freshness != colony.SurveyFreshnessMissing {
		t.Fatalf("freshness = %q, want missing", fresh.Freshness)
	}
	manifest := codexPlanManifest{}
	if err := validatePlanTerritorySnapshot(root, manifest); err == nil {
		t.Fatal("plan finalization accepted an absent territory snapshot")
	}

	writeFreshTerritorySnapshot199(t, root, time.Now().UTC().Add(-time.Minute))
	fresh = ensureTerritoryFreshness(root)
	attachTerritoryToPlanManifest(&manifest, fresh)
	if err := validatePlanTerritorySnapshot(root, manifest); err != nil {
		t.Fatalf("matching territory snapshot rejected: %v", err)
	}

	manifest.Territory.SnapshotID = strings.Repeat("0", 64)
	if err := validatePlanTerritorySnapshot(root, manifest); err == nil {
		t.Fatal("plan finalization accepted a mismatched territory snapshot")
	}
}

func setupTerritoryLifecycle199(t *testing.T) (string, string) {
	t.Helper()
	saveGlobals(t)
	root := initTerritoryFreshnessRepo199(t)
	dataDir := filepath.Join(root, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("mkdir lifecycle data: %v", err)
	}
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	store = s
	goal := "Plan from verified territory"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})
	return root, dataDir
}

func territoryArtifactFixture199(marker string) map[string][]byte {
	artifacts := make(map[string][]byte, len(requiredTerritoryArtifactPaths()))
	for _, rel := range requiredTerritoryArtifactPaths() {
		if strings.HasSuffix(rel, ".json") {
			artifacts[rel] = []byte(fmt.Sprintf("{\"marker\":%q}\n", marker+":"+filepath.Base(rel)))
			continue
		}
		artifacts[rel] = []byte("# " + filepath.Base(rel) + "\n\n" + marker + "\n")
	}
	return artifacts
}

func readTerritorySnapshotBytes199(t *testing.T, root string) map[string][]byte {
	t.Helper()
	paths := append([]string{}, requiredTerritoryArtifactPaths()...)
	paths = append(paths, territorySnapshotRelativePath, filepath.ToSlash(filepath.Join(".aether", "data", "COLONY_STATE.json")))
	result := make(map[string][]byte, len(paths))
	for _, rel := range paths {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		result[rel] = data
	}
	return result
}

func equalByteMaps199(a, b map[string][]byte) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		if !bytes.Equal(value, b[key]) {
			return false
		}
	}
	return true
}

func equalStringMaps199(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		if value != b[key] {
			return false
		}
	}
	return true
}
