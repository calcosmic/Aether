package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func syncCodexSkillsFromPayload(payload codexSkillPayload, homeDir string) syncResult {
	result := syncResult{}
	fail := func(err error) syncResult { result.errors = append(result.errors, err.Error()); return result }
	if err := validateCodexSkillPayload(payload); err != nil {
		return fail(err)
	}
	homeDir, err := filepath.Abs(homeDir)
	if err != nil {
		return fail(err)
	}
	codexRoot := filepath.Join(homeDir, ".codex")
	// Installation has no colony. Its journal lives in the selected home's hub,
	// never in the source checkout or an owner's active project.
	hubRoot := filepath.Join(homeDir, ".aether")
	dataRoot := filepath.Join(hubRoot, "data")
	for _, dir := range []string{codexRoot, hubRoot, dataRoot} {
		if err := rejectLifecycleSymlinkTarget(homeDir, filepath.Join(dir, ".root-check")); err != nil {
			return fail(err)
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fail(err)
		}
	}
	plan := maintenanceMutationPlan{
		SchemaVersion: maintenanceMutationSchemaVersion, Operation: "install-codex-skills",
		TransactionID: fmt.Sprintf("codex-skills-%d", time.Now().UnixNano()),
		SourceRoot:    hubRoot, DestinationRoot: codexRoot, Channel: channelStable,
		DesiredVersion: payload.SourceVersion, Checkpoint: "codex-skills:validated", Recovery: "inspect retained install transaction before retrying",
		Allowlist: lifecycleTransactionAllowlist{RepositoryRoot: hubRoot, LifecycleDataRoot: dataRoot, CodexHome: codexRoot},
	}
	if err := planCodexSkillTargets(&plan, payload); err != nil {
		return fail(err)
	}
	result.preserved = append(result.preserved, plan.PreservedCodexSkills...)
	committed, err := commitMaintenanceMutation(plan)
	if err != nil {
		return fail(err)
	}
	for _, target := range committed.Targets {
		switch target.Change {
		case maintenanceMutationChangeWrite:
			result.copied++
		case maintenanceMutationChangeRemove:
			result.removed = append(result.removed, target.RelativeTarget)
		case maintenanceMutationChangeUnchanged:
			result.skipped++
		}
	}
	return result
}

// loadCodexSkillPayload reads one closed, versioned envelope. Never regenerate
// from this consumer's compiled command inventory or trust paths before checking
// containment. A partial concurrent publication fails validation and is retryable.
func loadCodexSkillPayload(hubDir string) (codexSkillPayload, error) {
	var payload codexSkillPayload
	root := filepath.Join(hubDir, "system", "codex-skills")
	read := func(relative string) (lifecycleFileState, error) {
		if filepath.IsAbs(relative) || filepath.ToSlash(filepath.Clean(relative)) != relative || strings.Contains(relative, "\\") || strings.HasPrefix(relative, "../") {
			return lifecycleFileState{}, fmt.Errorf("codex skills: invalid published path %q", relative)
		}
		filename := filepath.Join(root, filepath.FromSlash(relative))
		if err := rejectLifecycleSymlinkTarget(hubDir, filename); err != nil {
			return lifecycleFileState{}, err
		}
		state, err := readLifecycleFileState(filename)
		if err == nil && !state.Exists {
			err = fmt.Errorf("codex skills: missing published file %s", relative)
		}
		return state, err
	}
	manifest, err := read("manifest.json")
	if err != nil {
		return payload, err
	}
	if err := decodeLifecycleJSON(manifest.Bytes, &payload); err != nil {
		return payload, fmt.Errorf("codex skills: invalid published manifest: %w", err)
	}
	for i := range payload.Files {
		file := &payload.Files[i]
		state, err := read(file.RelativePath)
		if err != nil {
			return codexSkillPayload{}, err
		}
		if uint32(state.Mode.Perm()) != file.Mode {
			return codexSkillPayload{}, fmt.Errorf("codex skills: published mode mismatch for %s", file.RelativePath)
		}
		file.Content = state.Bytes
	}
	return payload, validateCodexSkillPayload(payload)
}

// Publish the manifest and all described bytes in the same existing transaction.
// Generic companion cleanup must never own this generated subtree.
func publishCodexSkillPayload(hubDir string, payload codexSkillPayload) (maintenanceMutationResult, error) {
	var result maintenanceMutationResult
	if err := validateCodexSkillPayload(payload); err != nil {
		return result, err
	}
	hubDir, err := filepath.Abs(hubDir)
	if err != nil {
		return result, err
	}
	dataRoot := filepath.Join(hubDir, "data")
	for _, dir := range []string{hubDir, dataRoot} {
		if err := rejectLifecycleSymlinkTarget(filepath.Dir(hubDir), filepath.Join(dir, ".root-check")); err != nil {
			return result, err
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			return result, err
		}
	}
	plan := maintenanceMutationPlan{SchemaVersion: maintenanceMutationSchemaVersion, Operation: "publish-codex-skills", TransactionID: fmt.Sprintf("publish-codex-skills-%d", time.Now().UnixNano()), SourceRoot: hubDir, DestinationRoot: hubDir, DesiredVersion: payload.SourceVersion, Checkpoint: "codex-skills:validated", Recovery: "inspect retained publish transaction before retrying", Allowlist: lifecycleTransactionAllowlist{RepositoryRoot: hubDir, LifecycleDataRoot: dataRoot}}
	files := append([]codexSkillPayloadFile(nil), payload.Files...)
	manifest, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return result, err
	}
	files = append(files, codexSkillPayloadFile{RelativePath: "manifest.json", Mode: 0644, Content: append(manifest, '\n')})
	for _, file := range files {
		relative := filepath.Join("system", "codex-skills", filepath.FromSlash(file.RelativePath))
		if err := rejectLifecycleSymlinkTarget(hubDir, filepath.Join(hubDir, relative)); err != nil {
			return result, err
		}
		state, err := readLifecycleFileState(filepath.Join(hubDir, relative))
		if err != nil {
			return result, err
		}
		mode := state.Mode.Perm()
		plan.Targets = append(plan.Targets, maintenanceMutationTarget{Root: lifecycleTransactionRootRepository, RelativeTarget: relative, Label: "Codex skill payload", Source: "payload " + codexSkillPayloadIdentity(payload), Content: file.Content, Mode: os.FileMode(file.Mode), ExpectedDigest: state.Digest, ExpectedMode: &mode, Managed: true})
	}
	return commitMaintenanceMutation(plan)
}

func installHubErrors(result map[string]interface{}) []string {
	var errors []string
	for _, key := range []string{"error", "version_error"} {
		if value, ok := result[key].(string); ok && value != "" {
			errors = append(errors, value)
		}
	}
	if values, ok := result["errors"].([]string); ok {
		errors = append(errors, values...)
	}
	return errors
}

// Join the registered update's existing transaction before preview/commit. The
// producer's complete manifest owns the inventory, including future commands.
func appendMaintenanceUpdateCodexSkillTargets(plan *maintenanceMutationPlan, hubRoot, homeDir string) (codexSkillPayload, error) {
	payload, err := loadCodexSkillPayload(hubRoot)
	if err != nil {
		return payload, err
	}
	if normalizeVersion(payload.SourceVersion) != normalizeVersion(plan.DesiredVersion) {
		return payload, fmt.Errorf("codex skills: payload version %s does not match hub version %s", payload.SourceVersion, plan.DesiredVersion)
	}
	codexRoot := filepath.Join(homeDir, ".codex")
	info, err := os.Lstat(codexRoot)
	if err != nil {
		return payload, fmt.Errorf("codex skills: home unavailable; run aether install first: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return payload, fmt.Errorf("codex skills: home is not a real directory")
	}
	plan.Allowlist.CodexHome = codexRoot
	return payload, planCodexSkillTargets(plan, payload)
}
