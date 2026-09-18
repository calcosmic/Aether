package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// These are deterministic adversarial bytes, never actual host exports.
func TestCodexNativeGapHostCapture(t *testing.T) {
	for _, value := range []string{"", "a prompt", "arbitrary bytes", `{"tools":["exec_command"]}`, `{"available":false}`} {
		t.Run(value, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "substitution")
			raw := []byte(value)
			if err := os.WriteFile(path, raw, 0600); err != nil { t.Fatal(err) }
			ref := nativeEvidenceFile{path, lifecycleDigest(raw)}
			r := codexNativeLiveReceipt{ClientVersion:"measured", Model:"model", ClientPath:path, ClientSHA256:ref.SHA256, Args:[]string{"exec"}, Artifacts:map[string]string{path:ref.SHA256}}
			p := nativeGapHostProvenance{Version:r.ClientVersion, Model:r.Model, Args:r.Args, Executable:ref, Configuration:ref, ToolSchema:ref}
			if err := nativeGapHostIdentity(p,r); err == nil { t.Fatal("same-hash unrelated bytes accepted as configuration and full tool schema") }
		})
	}
}
