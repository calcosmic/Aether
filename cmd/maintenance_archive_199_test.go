package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
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

func TestMaintenanceArchive199RepairRollback(t *testing.T) {
	fixture, entombed, targetPath, original, tampered := newArchiveRepairFixture199(t)
	manifestBefore, err := os.ReadFile(filepath.Join(entombed.ChamberPath, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	chamberBefore := entombFixtureDigest199(t, entombed.ChamberPath)
	relativeTarget, err := filepath.Rel(fixture.root, targetPath)
	if err != nil {
		t.Fatal(err)
	}

	plan, err := prepareArchiveMaintenanceRepair(archiveMaintenanceRepairRequest{
		RepositoryRoot: fixture.root,
		DataRoot:       fixture.dataRoot,
		ChamberPath:    entombed.ChamberPath,
		TransactionID:  "maintenance-archive-rollback",
		Targets: []archiveMaintenanceRepairTarget{{
			RelativePath: filepath.ToSlash(relativeTarget),
			Content:      original,
			Source:       "operator-provided manifest-backed preimage",
		}},
		Fault: func(point string) error {
			if point == "after_target_commit:target-0001" {
				return errors.New("injected archive repair failure")
			}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("prepare repair: %v", err)
	}
	if plan.PreviewDigest == "" || plan.Checkpoint == "" || plan.Preview.StateEffect != colony.LifecycleStateEffectNone {
		t.Fatalf("repair preview omitted checkpoint/baseline proof: %+v", plan)
	}
	plan.ApprovedPreviewDigest = plan.PreviewDigest
	result, err := commitArchiveMaintenanceRepair(plan)
	if err == nil || !strings.Contains(err.Error(), "injected archive repair failure") {
		t.Fatalf("repair fault = %v, want injected failure", err)
	}
	if !result.RolledBack || result.StateEffect != colony.LifecycleStateEffectRolledBack || result.Receipt == nil || result.Receipt.Transaction.Stage != colony.TransactionStageRolledBack {
		t.Fatalf("repair rollback omitted auditable receipt/effect: %+v", result)
	}
	if got, readErr := os.ReadFile(targetPath); readErr != nil || !bytes.Equal(got, tampered) {
		t.Fatalf("repair fault did not restore exact historical preimage: %q err=%v", got, readErr)
	}
	if chamberAfter := entombFixtureDigest199(t, entombed.ChamberPath); chamberAfter != chamberBefore {
		t.Fatalf("rollback changed chamber bytes\nbefore=%s\nafter=%s", chamberBefore, chamberAfter)
	}
	if manifestAfter, readErr := os.ReadFile(filepath.Join(entombed.ChamberPath, "manifest.json")); readErr != nil || !bytes.Equal(manifestAfter, manifestBefore) {
		t.Fatal("rollback rewrote the historical manifest")
	}

	t.Run("forced marker refusal", func(t *testing.T) {
		forced := newEntombTransactionFixture199(t, colony.SealDispositionForcedIncomplete)
		forcedEntombed, entombErr := runEntombTransaction(entombTransactionInput{
			Root: forced.root, DataRoot: forced.dataRoot, Confirmed: true, Now: entombTransactionTime199(),
		})
		if entombErr != nil {
			t.Fatal(entombErr)
		}
		handoffPath := filepath.Join(forced.root, ".aether", "HANDOFF.md")
		handoff, readErr := os.ReadFile(handoffPath)
		if readErr != nil {
			t.Fatal(readErr)
		}
		writeEntombFile199(t, handoffPath, []byte(strings.ReplaceAll(string(handoff), forced.outcome.OwnerReason, "[removed]")))
		before := entombFixtureDigest199(t, forced.root)
		_, prepareErr := prepareArchiveMaintenanceRepair(archiveMaintenanceRepairRequest{
			RepositoryRoot: forced.root, DataRoot: forced.dataRoot, ChamberPath: forcedEntombed.ChamberPath,
			TransactionID: "maintenance-forced-marker-refusal",
			Targets: []archiveMaintenanceRepairTarget{{
				RelativePath: ".aether/HANDOFF.md", Content: handoff, Source: "verified tombstone copy",
			}},
		})
		if prepareErr == nil || !strings.Contains(strings.ToLower(prepareErr.Error()), "forced") {
			t.Fatalf("missing forced marker reached mutation preview: %v", prepareErr)
		}
		if after := entombFixtureDigest199(t, forced.root); after != before {
			t.Fatalf("forced-marker preflight wrote bytes\nbefore=%s\nafter=%s", before, after)
		}
	})
}

func TestMaintenanceArchive199Receipt(t *testing.T) {
	fixture, entombed, targetPath, original, _ := newArchiveRepairFixture199(t)
	manifestPath := filepath.Join(entombed.ChamberPath, "manifest.json")
	manifestBefore, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	relativeTarget, err := filepath.Rel(fixture.root, targetPath)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := prepareArchiveMaintenanceRepair(archiveMaintenanceRepairRequest{
		RepositoryRoot: fixture.root, DataRoot: fixture.dataRoot, ChamberPath: entombed.ChamberPath,
		TransactionID: "maintenance-archive-receipt",
		Targets: []archiveMaintenanceRepairTarget{{
			RelativePath: filepath.ToSlash(relativeTarget), Content: original, Source: "manifest-backed retained memory",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Preview.Targets) != 1 || plan.Preview.Targets[0].CurrentDigest == plan.Preview.Targets[0].DesiredDigest || plan.BaselineDigest == "" {
		t.Fatalf("repair preview lacks exact current/desired baseline: %+v", plan)
	}
	plan.ApprovedPreviewDigest = plan.PreviewDigest
	result, err := commitArchiveMaintenanceRepair(plan)
	if err != nil {
		t.Fatalf("commit repair: %v", err)
	}
	if result.StateEffect != colony.LifecycleStateEffectCommitted || result.Receipt == nil || result.Receipt.Transaction.Stage != colony.TransactionStageVerified || result.RolledBack {
		t.Fatalf("repair receipt is not a verified commit: %+v", result)
	}
	if len(result.ChangedFiles) != 1 || result.ChangedFiles[0] != filepath.ToSlash(relativeTarget) || len(result.Verification) == 0 || result.Rollback == "" || result.NextAction == "" {
		t.Fatalf("repair result omitted changed files/verification/rollback/next: %+v", result)
	}
	if got, readErr := os.ReadFile(targetPath); readErr != nil || !bytes.Equal(got, original) {
		t.Fatalf("committed repair bytes = %q err=%v", got, readErr)
	}
	if manifestAfter, readErr := os.ReadFile(manifestPath); readErr != nil || !bytes.Equal(manifestAfter, manifestBefore) {
		t.Fatal("repair rewrote the historical manifest")
	}
	if !result.Inspection.Valid || !result.Inspection.ContentVerified {
		t.Fatalf("committed repair did not re-establish archive truth: %+v", result.Inspection)
	}
}

func newArchiveRepairFixture199(t *testing.T) (entombTransactionFixture199, entombTransactionResult, string, []byte, []byte) {
	t.Helper()
	fixture := newEntombTransactionFixture199(t, colony.SealDispositionVerified)
	entombed, err := runEntombTransaction(entombTransactionInput{
		Root: fixture.root, DataRoot: fixture.dataRoot, Confirmed: true, Now: entombTransactionTime199(),
	})
	if err != nil {
		t.Fatal(err)
	}
	targetPath := filepath.Join(entombed.ChamberPath, "QUEEN.md")
	original, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	tampered := []byte("# Tampered retained colony memory\n")
	writeEntombFile199(t, targetPath, tampered)
	return fixture, entombed, targetPath, original, tampered
}

func archiveInspectionHasFinding199(inspection archiveMaintenanceInspectionResult, code string) bool {
	for _, finding := range inspection.Findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}
