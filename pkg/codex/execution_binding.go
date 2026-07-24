package codex

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const ExecutionBindingSchemaVersion = 1

// ExecutionBinding binds one provider launch to the immutable build attempt
// that authorized it. ManifestSHA256 authenticates the work request while the
// workspace fingerprint prevents a result from another checkout or branch
// from being accepted as current work.
type ExecutionBinding struct {
	SchemaVersion        int    `json:"schema_version"`
	RunID                string `json:"run_id"`
	AttemptID            string `json:"attempt_id"`
	ManifestSHA256       string `json:"manifest_sha256"`
	WorkspaceFingerprint string `json:"workspace_fingerprint"`
	ExecutionOwner       string `json:"execution_owner"`
}

func (b ExecutionBinding) Validate() error {
	if b.SchemaVersion != ExecutionBindingSchemaVersion {
		return fmt.Errorf("execution binding schema_version must be %d", ExecutionBindingSchemaVersion)
	}
	for name, value := range map[string]string{
		"run_id":                b.RunID,
		"attempt_id":            b.AttemptID,
		"manifest_sha256":       b.ManifestSHA256,
		"workspace_fingerprint": b.WorkspaceFingerprint,
		"execution_owner":       b.ExecutionOwner,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("execution binding requires %s", name)
		}
	}
	if !strings.HasPrefix(strings.TrimSpace(b.RunID), "run-") {
		return fmt.Errorf("execution binding run_id is invalid")
	}
	if !strings.HasPrefix(strings.TrimSpace(b.AttemptID), "attempt-") {
		return fmt.Errorf("execution binding attempt_id is invalid")
	}
	for name, value := range map[string]string{
		"manifest_sha256":       b.ManifestSHA256,
		"workspace_fingerprint": b.WorkspaceFingerprint,
	} {
		value = strings.TrimSpace(value)
		if len(value) != sha256.Size*2 {
			return fmt.Errorf("execution binding %s must be a SHA-256 digest", name)
		}
		if _, err := hex.DecodeString(value); err != nil {
			return fmt.Errorf("execution binding %s must be a SHA-256 digest", name)
		}
	}
	return nil
}

func NewExecutionRunID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate execution run id: %w", err)
	}
	return "run-" + hex.EncodeToString(value), nil
}

// WorkspaceFingerprint identifies a checkout and branch without including
// mutable file contents or HEAD. Workers are expected to modify files, and may
// commit between sequential waves, so content hashes would reject valid work.
func WorkspaceFingerprint(root string) (string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return "", fmt.Errorf("workspace root is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve workspace root: %w", err)
	}
	canonical := filepath.Clean(abs)
	if resolved, resolveErr := filepath.EvalSymlinks(canonical); resolveErr == nil {
		canonical = filepath.Clean(resolved)
	}

	identity := map[string]string{
		"root": canonical,
		"mode": "directory",
	}
	if top, ok := gitIdentityValue(canonical, "rev-parse", "--show-toplevel"); ok {
		identity["mode"] = "git"
		identity["git_root"] = canonicalGitPath(canonical, top)
		if common, commonOK := gitIdentityValue(canonical, "rev-parse", "--git-common-dir"); commonOK {
			identity["git_common_dir"] = canonicalGitPath(canonical, common)
		}
		if branch, branchOK := gitIdentityValue(canonical, "symbolic-ref", "--quiet", "--short", "HEAD"); branchOK {
			identity["branch"] = branch
		} else {
			identity["branch"] = "detached"
		}
	}
	payload, err := json.Marshal(identity)
	if err != nil {
		return "", fmt.Errorf("encode workspace identity: %w", err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(payload)), nil
}

func ValidateExecutionWorkspace(root string, binding ExecutionBinding) error {
	if err := binding.Validate(); err != nil {
		return err
	}
	actual, err := WorkspaceFingerprint(root)
	if err != nil {
		return err
	}
	if actual != strings.TrimSpace(binding.WorkspaceFingerprint) {
		return fmt.Errorf("execution binding workspace does not match the active checkout or branch")
	}
	return nil
}

func gitIdentityValue(root string, args ...string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		return "", false
	}
	value := strings.TrimSpace(string(output))
	return value, value != ""
}

func canonicalGitPath(root, value string) string {
	value = strings.TrimSpace(value)
	if !filepath.IsAbs(value) {
		value = filepath.Join(root, value)
	}
	value = filepath.Clean(value)
	if resolved, err := filepath.EvalSymlinks(value); err == nil {
		return filepath.Clean(resolved)
	}
	return value
}
