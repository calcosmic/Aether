package cmd

import (
	"io/fs"
	"os"
	"path/filepath"
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
	derive := func(receipt codexNativeLiveReceipt) nativeGapCancellationEvidence {
		return nativeRetainedCancellationEvidence(receipt, root, filepath.Join(root, "home"))
	}
	got := derive(r)
	if !got.Qualified || got.Disposition != nativeGapCancellationRefused || got.RuntimeCancelled || got.NoPostAckWrites {
		t.Fatalf("inventoried synthetic replay control rejected: %+v", got)
	}
	for _, name := range []string{"uninventoried-capture", "uninventoried-request", "changed-capture", "legacy", "unknown-amendment"} {
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
		})
	}
	r.Cancellation = &nativeGapCancellationEvidence{Qualified: true, Disposition: nativeGapCancellationRefused, Refusal: &capture}
	resetCodexNativeDerivedEvidence(&r)
	if r.Cancellation != nil {
		t.Fatal("derived cancellation survived reset")
	}
}
