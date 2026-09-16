package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const codexSkillOwnershipSchema = "codex-skill-ownership/v1"

type codexSkillOwnership struct {
	SchemaVersion   string                  `json:"schema_version"`
	SourceVersion   string                  `json:"source_version"`
	PayloadIdentity string                  `json:"payload_identity"`
	Files           []codexSkillPayloadFile `json:"files"`
}

// Plan only exact destinations beneath the typed Codex root. No directory prune
// and no marker-based adoption: an existing file needs its recorded digest/mode.
func planCodexSkillTargets(plan *maintenanceMutationPlan, payload codexSkillPayload) error {
	if err := validateCodexSkillPayload(payload); err != nil {
		return err
	}
	tx, err := beginLifecycleTransaction(lifecycleTransactionConfig{TransactionID: plan.TransactionID, Command: plan.Operation, Allowlist: plan.Allowlist})
	if err != nil {
		return err
	}
	const prefix = "skills/aether"
	_, ownerPath, _, err := tx.resolveTarget(lifecycleTransactionRootCodexHome, filepath.FromSlash(prefix+"/.aether-owned.json"))
	if err != nil {
		return err
	}
	ownerState, err := readLifecycleFileState(ownerPath)
	if err != nil {
		return err
	}
	owned := map[string]codexSkillPayloadFile{}
	if ownerState.Exists {
		var previous codexSkillOwnership
		if err := decodeLifecycleJSON(ownerState.Bytes, &previous); err != nil {
			return fmt.Errorf("codex skills: ownership collision: %w", err)
		}
		if previous.SchemaVersion != codexSkillOwnershipSchema || len(previous.PayloadIdentity) != 71 || !codexSkillVersionPattern.MatchString(previous.SourceVersion) || len(previous.Files) == 0 {
			return fmt.Errorf("codex skills: invalid ownership collision at %s", ownerPath)
		}
		if compareVersions(payload.SourceVersion, previous.SourceVersion) < 0 {
			return fmt.Errorf("codex skills: refusing payload downgrade")
		}
		for _, file := range previous.Files {
			if _, duplicate := owned[file.RelativePath]; duplicate {
				return fmt.Errorf("codex skills: duplicate ownership path")
			}
			owned[file.RelativePath] = file
		}
	}
	var targets []maintenanceMutationTarget
	for _, file := range payload.Files {
		relative := filepath.Join(filepath.FromSlash(prefix), filepath.FromSlash(file.RelativePath))
		_, destination, _, err := tx.resolveTarget(lifecycleTransactionRootCodexHome, relative)
		if err != nil {
			return err
		}
		state, err := readLifecycleFileState(destination)
		if err != nil {
			return err
		}
		if state.Exists {
			baseline, proven := owned[file.RelativePath]
			if !proven || baseline.SHA256 != state.Digest || baseline.Mode != uint32(state.Mode.Perm()) {
				return fmt.Errorf("codex skills: collision at %s (unowned or modified)", destination)
			}
		}
		targets = append(targets, maintenanceMutationTarget{
			Root: lifecycleTransactionRootCodexHome, RelativeTarget: relative,
			Label: "Skills (codex shims)", Source: "payload " + codexSkillPayloadIdentity(payload),
			Content: append([]byte(nil), file.Content...), Mode: os.FileMode(file.Mode), ExpectedDigest: state.Digest, Managed: true,
		})
		delete(owned, file.RelativePath)
	}
	// Retirement/adoption belongs to the migration planner. Until then preserve
	// unknown ownership entries instead of silently forgetting or deleting them.
	if len(owned) != 0 {
		return fmt.Errorf("codex skills: ownership includes files outside this payload; migration required")
	}
	ownership := codexSkillOwnership{SchemaVersion: codexSkillOwnershipSchema, SourceVersion: payload.SourceVersion, PayloadIdentity: codexSkillPayloadIdentity(payload), Files: payload.Files}
	raw, err := json.MarshalIndent(ownership, "", "  ")
	if err != nil {
		return err
	}
	targets = append(targets, maintenanceMutationTarget{
		Root: lifecycleTransactionRootCodexHome, RelativeTarget: filepath.FromSlash(prefix + "/.aether-owned.json"),
		Label: "Skills (codex shims)", Source: "validated payload ownership",
		Content: append(raw, '\n'), Mode: 0o644, ExpectedDigest: ownerState.Digest, Managed: true,
	})
	plan.Targets = append(plan.Targets, targets...)
	return nil
}

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
	committed, err := commitMaintenanceMutation(plan)
	if err != nil {
		return fail(err)
	}
	for _, target := range committed.Targets {
		switch target.Change {
		case maintenanceMutationChangeWrite:
			result.copied++
		case maintenanceMutationChangeUnchanged:
			result.skipped++
		}
	}
	return result
}
