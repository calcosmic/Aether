package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestMaintenanceArchive199InspectReadOnly(t *testing.T) {
	fixture := newEntombTransactionFixture199(t, colony.SealDispositionVerified)
	entombed, err := runEntombTransaction(entombTransactionInput{
		Root: fixture.root, DataRoot: fixture.dataRoot, Confirmed: true, Now: entombTransactionTime199(),
	})
	if err != nil {
		t.Fatal(err)
	}

	before := entombFixtureDigest199(t, fixture.root)
	inspection, err := inspectArchiveMaintenance(archiveMaintenanceInspectRequest{
		RepositoryRoot: fixture.root,
		DataRoot:       fixture.dataRoot,
		ChamberPath:    entombed.ChamberPath,
	})
	if err != nil {
		t.Fatalf("inspect archive: %v", err)
	}
	if !inspection.Valid || !inspection.ManifestVerified || !inspection.ContentVerified || !inspection.CrossReferencesVerified || !inspection.ContextVerified {
		t.Fatalf("inspection did not prove manifest/content/cross-reference/context truth: %+v", inspection)
	}
	if inspection.ManifestDigest != entombed.ManifestDigest || inspection.SealDisposition != colony.SealDispositionVerified {
		t.Fatalf("inspection lost archive identity: %+v", inspection)
	}
	if inspection.EntriesChecked == 0 || inspection.ReferencesChecked == 0 || len(inspection.Evidence) == 0 {
		t.Fatalf("inspection cited no content evidence: %+v", inspection)
	}
	if inspection.StateEffect != colony.LifecycleStateEffectNone {
		t.Fatalf("read-only inspection state effect = %q", inspection.StateEffect)
	}

	check := integrityCheckForArchiveInspection(inspection)
	if check.Status != "pass" || !strings.Contains(check.Message, entombed.ManifestDigest) || !strings.Contains(check.Message, "content digests") || !strings.Contains(check.Message, "cross-references") {
		t.Fatalf("expert integrity line omitted typed archive evidence: %+v", check)
	}
	if after := entombFixtureDigest199(t, fixture.root); after != before {
		t.Fatalf("archive inspection mutated fixture bytes\nbefore=%s\nafter=%s", before, after)
	}
}

func TestMaintenanceArchive199ManifestRefusal(t *testing.T) {
	fixture := newEntombTransactionFixture199(t, colony.SealDispositionVerified)
	entombed, err := runEntombTransaction(entombTransactionInput{
		Root: fixture.root, DataRoot: fixture.dataRoot, Confirmed: true, Now: entombTransactionTime199(),
	})
	if err != nil {
		t.Fatal(err)
	}

	manifestPath := filepath.Join(entombed.ChamberPath, "manifest.json")
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest colony.ArchiveManifest
	if err := json.Unmarshal(content, &manifest); err != nil {
		t.Fatal(err)
	}
	manifest.CrossReferences[0].TargetID = "seal/tampered.json"
	writeEntombJSON199(t, manifestPath, manifest)

	before := entombFixtureDigest199(t, fixture.root)
	inspection, err := inspectArchiveMaintenance(archiveMaintenanceInspectRequest{
		RepositoryRoot: fixture.root,
		DataRoot:       fixture.dataRoot,
		ChamberPath:    entombed.ChamberPath,
	})
	if err != nil {
		t.Fatalf("tamper must return typed inspection evidence, not a hard failure: %v", err)
	}
	if inspection.Valid || inspection.ManifestVerified || len(inspection.Findings) == 0 {
		t.Fatalf("tampered manifest was accepted: %+v", inspection)
	}
	if !archiveInspectionHasFinding199(inspection, "manifest_digest") {
		t.Fatalf("manifest tamper was not named by digest evidence: %+v", inspection.Findings)
	}
	if check := integrityCheckForArchiveInspection(inspection); check.Status != "fail" || !strings.Contains(check.Message, "manifest") {
		t.Fatalf("integrity view hid manifest refusal: %+v", check)
	}
	if after := entombFixtureDigest199(t, fixture.root); after != before {
		t.Fatalf("manifest refusal mutated fixture bytes\nbefore=%s\nafter=%s", before, after)
	}
}

func TestMaintenanceArchive199ForcedMarker(t *testing.T) {
	t.Run("missing marker", func(t *testing.T) {
		fixture := newEntombTransactionFixture199(t, colony.SealDispositionForcedIncomplete)
		entombed, err := runEntombTransaction(entombTransactionInput{
			Root: fixture.root, DataRoot: fixture.dataRoot, Confirmed: true, Now: entombTransactionTime199(),
		})
		if err != nil {
			t.Fatal(err)
		}
		handoffPath := filepath.Join(fixture.root, ".aether", "HANDOFF.md")
		handoff, err := os.ReadFile(handoffPath)
		if err != nil {
			t.Fatal(err)
		}
		writeEntombFile199(t, handoffPath, []byte(strings.ReplaceAll(string(handoff), fixture.outcome.OwnerReason, "[removed]")))

		before := entombFixtureDigest199(t, fixture.root)
		inspection, err := inspectArchiveMaintenance(archiveMaintenanceInspectRequest{
			RepositoryRoot: fixture.root, DataRoot: fixture.dataRoot, ChamberPath: entombed.ChamberPath,
		})
		if err != nil {
			t.Fatal(err)
		}
		if inspection.Valid || !archiveInspectionHasFinding199(inspection, "forced_marker_missing") {
			t.Fatalf("missing forced marker was not reported as tamper evidence: %+v", inspection)
		}
		if after := entombFixtureDigest199(t, fixture.root); after != before {
			t.Fatalf("forced-marker refusal mutated fixture bytes\nbefore=%s\nafter=%s", before, after)
		}
	})

	t.Run("conflicting marker", func(t *testing.T) {
		fixture := newEntombTransactionFixture199(t, colony.SealDispositionVerified)
		entombed, err := runEntombTransaction(entombTransactionInput{
			Root: fixture.root, DataRoot: fixture.dataRoot, Confirmed: true, Now: entombTransactionTime199(),
		})
		if err != nil {
			t.Fatal(err)
		}
		contextPath := filepath.Join(fixture.root, ".aether", "CONTEXT.md")
		contextBytes, err := os.ReadFile(contextPath)
		if err != nil {
			t.Fatal(err)
		}
		writeEntombFile199(t, contextPath, append(contextBytes, []byte("Seal disposition: forced_incomplete\n")...))

		before := entombFixtureDigest199(t, fixture.root)
		inspection, err := inspectArchiveMaintenance(archiveMaintenanceInspectRequest{
			RepositoryRoot: fixture.root, DataRoot: fixture.dataRoot, ChamberPath: entombed.ChamberPath,
		})
		if err != nil {
			t.Fatal(err)
		}
		if inspection.Valid || !archiveInspectionHasFinding199(inspection, "forced_marker_conflict") {
			t.Fatalf("conflicting forced marker was not reported as tamper evidence: %+v", inspection)
		}
		if after := entombFixtureDigest199(t, fixture.root); after != before {
			t.Fatalf("forced-marker conflict mutated fixture bytes\nbefore=%s\nafter=%s", before, after)
		}
	})
}

func archiveInspectionHasFinding199(inspection archiveMaintenanceInspectionResult, code string) bool {
	for _, finding := range inspection.Findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}
