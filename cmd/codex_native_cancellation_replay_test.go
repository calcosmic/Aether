package cmd

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Synthetic records test replay rejection only. They are not live host proof.
func TestCodexNativeCancellationRefusalReplay(t *testing.T) {
	r, capture := nativeGapCancellationRefusalFixture(t)
	root := t.TempDir()
	capturePath := filepath.Join(root, "cancellation-refusal", "cancellation-refusal.json")
	nativeGapCancellationWrite(t, capturePath, mustNativeGapJSON(t, capture))
	inventory := map[string]string{}
	for _, dir := range []string{filepath.Dir(r.CandidatePath), root} {
		if err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return err
			}
			if !nativeRetainedArtifactPath(path) {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err == nil {
				inventory[path] = lifecycleDigest(raw)
			}
			return err
		}); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := os.ReadFile(r.AttemptPath)
	if err != nil {
		t.Fatal(err)
	}
	inventory[r.AttemptPath] = lifecycleDigest(raw)
	r.Artifacts = inventory
	for _, guard := range capture.Guards {
		for _, files := range []map[string]nativeEvidenceFile{guard.Before, guard.After} {
			seen := map[string]bool{}
			for _, name := range nativeGapCancellationSnapshotShapeNames() {
				original := filepath.Join(r.FixtureRoot, ".aether", "data", "snapshot-shapes", name)
				ref := files[original]
				if nativeRetainedArtifactPath(original) && filepath.Base(original) != "receipt.json" {
					t.Fatalf("control source unexpectedly passes outer policy: %s", original)
				}
				if inventory[ref.Path] != ref.SHA256 || ref.Path == "" || seen[ref.Path] || len(filepath.Base(ref.Path)) != 68 || !strings.HasSuffix(ref.Path, ".txt") {
					t.Fatalf("source shape missing, collided or unbounded in original inventory: %s", name)
				}
				seen[ref.Path] = true
				sourceRaw, sourceErr := os.ReadFile(original)
				copied, copyErr := nativeGapCancellationRead(r, ref)
				if sourceErr != nil || copyErr != nil || !bytes.Equal(sourceRaw, copied) {
					t.Fatalf("authority source bytes changed in snapshot: %s", name)
				}
			}
			for _, ref := range files {
				if inventory[ref.Path] != ref.SHA256 {
					t.Fatalf("original outer inventory omitted authority snapshot: %s", ref.Path)
				}
			}
		}
	}
	derive := func(receipt codexNativeLiveReceipt) nativeGapCancellationEvidence {
		return nativeRetainedCancellationEvidence(receipt, root, filepath.Join(root, "home"))
	}
	got := derive(r)
	if !got.Qualified || got.Disposition != nativeGapCancellationRefused || got.RuntimeCancelled || got.NoPostAckWrites {
		t.Fatalf("inventoried synthetic replay control rejected: %+v", got)
	}
	first, _ := json.Marshal(got)
	second, _ := json.Marshal(derive(r))
	if !bytes.Equal(first, second) {
		t.Fatal("same inventoried authority produced unstable successful replay")
	}
	for _, name := range []string{"uninventoried-capture", "uninventoried-request", "uninventoried-authority", "changed-capture", "legacy", "unknown-amendment"} {
		t.Run(name, func(t *testing.T) {
			copyReceipt := r
			copyReceipt.Artifacts = map[string]string{}
			for path, digest := range inventory {
				copyReceipt.Artifacts[path] = digest
			}
			copyReceipt.Cancellation = &nativeGapCancellationEvidence{Qualified: true, Disposition: nativeGapCancellationRefused, Refusal: &capture}
			switch name {
			case "uninventoried-capture":
				delete(copyReceipt.Artifacts, capturePath)
			case "uninventoried-request":
				delete(copyReceipt.Artifacts, capture.Guards[0].OriginalRequest.Path)
			case "uninventoried-authority":
				for _, name := range []string{"extensionless", "target-0001.bin"} {
					path := filepath.Join(r.FixtureRoot, ".aether", "data", "snapshot-shapes", name)
					delete(copyReceipt.Artifacts, capture.Guards[0].Before[path].Path)
				}
				want := "guard changed durable authority: " + filepath.Join(r.FixtureRoot, ".aether", "data", "snapshot-shapes", "extensionless")
				for repetition := 0; repetition < 16; repetition++ {
					err := nativeGapCancellationGuardCheck(copyReceipt, capture.Guards[0])
					if err == nil || err.Error() != want {
						t.Fatalf("missing authority rejection is absent or nondeterministic: %v", err)
					}
				}
			case "changed-capture":
				original, err := os.ReadFile(capturePath)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					if err := os.WriteFile(capturePath, original, 0600); err != nil {
						t.Error(err)
					}
				})
				if err := os.WriteFile(capturePath, []byte(`{"qualified":true,"disposition":"refused"}`), 0600); err != nil {
					t.Fatal(err)
				}
			case "legacy":
				copyReceipt.ProofContract, copyReceipt.ProofAmendmentSHA256 = "", ""
			case "unknown-amendment":
				copyReceipt.ProofAmendmentSHA256 = "sha256:unknown"
			}
			got := derive(copyReceipt)
			if got.Qualified || got.RuntimeCancelled || got.NoPostAckWrites {
				t.Fatalf("%s inherited forged cached acceptance: %+v", name, got)
			}
			if name == "uninventoried-authority" {
				first, _ := json.Marshal(got)
				second, _ := json.Marshal(derive(copyReceipt))
				if !bytes.Equal(first, second) {
					t.Fatal("same missing authority produced unstable incomplete replay")
				}
			}
		})
	}
	r.Cancellation = &nativeGapCancellationEvidence{Qualified: true, Disposition: nativeGapCancellationRefused, Refusal: &capture}
	resetCodexNativeDerivedEvidence(&r)
	if r.Cancellation != nil {
		t.Fatal("derived cancellation survived reset")
	}
}
