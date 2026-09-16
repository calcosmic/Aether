package cmd

import (
	"fmt"
	"os"
	"path/filepath"
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
