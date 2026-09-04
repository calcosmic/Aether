package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/BurntSushi/toml"
	"github.com/calcosmic/Aether/pkg/colony"
	"gopkg.in/yaml.v3"
)

const maintenanceMutationSchemaVersion = "maintenance-mutation/v1"

type maintenanceMutationChange string

const (
	maintenanceMutationChangeWrite     maintenanceMutationChange = "write"
	maintenanceMutationChangeRemove    maintenanceMutationChange = "remove"
	maintenanceMutationChangeUnchanged maintenanceMutationChange = "unchanged"
)

// maintenanceMutationTarget is the exact, ownership-proven input to the
// shared lifecycle transaction. A target is never discovered again during
// commit: preview and commit operate on this same closed set.
type maintenanceMutationTarget struct {
	Root           lifecycleTransactionRootKind
	RelativeTarget string
	Label          string
	Source         string
	Action         lifecycleTransactionAction
	Content        []byte
	Mode           os.FileMode
	ExpectedDigest string
	Managed        bool
}

type maintenanceMutationPlan struct {
	SchemaVersion   string
	Operation       string
	TransactionID   string
	SourceRoot      string
	DestinationRoot string
	Channel         runtimeChannel
	CurrentVersion  string
	DesiredVersion  string
	Checkpoint      string
	Recovery        string
	Allowlist       lifecycleTransactionAllowlist
	Targets         []maintenanceMutationTarget
	Rename          func(oldPath, newPath string) error
	Fault           lifecycleTransactionFaultHook
}

type maintenanceMutationTargetPreview struct {
	Root           lifecycleTransactionRootKind `json:"root"`
	RelativeTarget string                       `json:"target"`
	Label          string                       `json:"label,omitempty"`
	Source         string                       `json:"source"`
	Change         maintenanceMutationChange    `json:"change"`
	CurrentDigest  string                       `json:"current_digest"`
	DesiredDigest  string                       `json:"desired_digest"`
	CurrentMode    uint32                       `json:"current_mode,omitempty"`
	DesiredMode    uint32                       `json:"desired_mode,omitempty"`
	CommitOrder    int                          `json:"commit_order"`
}

type maintenanceMutationPreview struct {
	SchemaVersion   string                             `json:"schema_version"`
	Operation       string                             `json:"operation"`
	TransactionID   string                             `json:"transaction_id"`
	SourceRoot      string                             `json:"source_root"`
	DestinationRoot string                             `json:"destination_root"`
	Channel         runtimeChannel                     `json:"channel,omitempty"`
	CurrentVersion  string                             `json:"current_version,omitempty"`
	DesiredVersion  string                             `json:"desired_version,omitempty"`
	Checkpoint      string                             `json:"checkpoint"`
	CommitOrder     []string                           `json:"commit_order"`
	Targets         []maintenanceMutationTargetPreview `json:"targets"`
	Recovery        string                             `json:"recovery"`
}

type maintenanceMutationResult struct {
	SchemaVersion string                             `json:"schema_version"`
	Operation     string                             `json:"operation"`
	Preview       maintenanceMutationPreview         `json:"preview"`
	Targets       []maintenanceMutationTargetPreview `json:"targets"`
	StateEffect   colony.LifecycleStateEffect        `json:"state_effect"`
	Receipt       *colony.LifecycleReceipt           `json:"receipt,omitempty"`
	Verification  []colony.LifecycleVerification     `json:"verification,omitempty"`
	Recovery      string                             `json:"recovery"`
}

// prepareMaintenanceMutation is read-only. It resolves and validates each
// exact target with the lifecycle coordinator, records before/after digests,
// and assigns the order that commit will use without creating staging or a
// journal.
func prepareMaintenanceMutation(plan maintenanceMutationPlan) (maintenanceMutationPreview, error) {
	preview := maintenanceMutationPreview{
		SchemaVersion:   plan.SchemaVersion,
		Operation:       strings.TrimSpace(plan.Operation),
		TransactionID:   strings.TrimSpace(plan.TransactionID),
		SourceRoot:      filepath.Clean(plan.SourceRoot),
		DestinationRoot: filepath.Clean(plan.DestinationRoot),
		Channel:         plan.Channel,
		CurrentVersion:  normalizeVersion(plan.CurrentVersion),
		DesiredVersion:  normalizeVersion(plan.DesiredVersion),
		Checkpoint:      strings.TrimSpace(plan.Checkpoint),
		Recovery:        strings.TrimSpace(plan.Recovery),
	}
	if preview.SchemaVersion != maintenanceMutationSchemaVersion {
		return preview, fmt.Errorf("maintenance mutation: schema_version must be %s", maintenanceMutationSchemaVersion)
	}
	if preview.Operation == "" || preview.TransactionID == "" {
		return preview, fmt.Errorf("maintenance mutation: operation and transaction id are required")
	}
	if preview.Checkpoint == "" || preview.Recovery == "" {
		return preview, fmt.Errorf("maintenance mutation: checkpoint and recovery action are required")
	}
	for label, root := range map[string]string{"source": plan.SourceRoot, "destination": plan.DestinationRoot} {
		if strings.TrimSpace(root) == "" || !filepath.IsAbs(root) || filepath.Clean(root) != root {
			return preview, fmt.Errorf("maintenance mutation: %s root must be a canonical absolute path", label)
		}
		info, err := os.Lstat(root)
		if err != nil {
			return preview, fmt.Errorf("maintenance mutation: inspect %s root: %w", label, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return preview, fmt.Errorf("maintenance mutation: %s root must be a real directory", label)
		}
	}
	if plan.Channel != "" && plan.Channel != channelStable && plan.Channel != channelDev {
		return preview, fmt.Errorf("maintenance mutation: unsupported channel %q", plan.Channel)
	}
	if plan.Allowlist.Hub.Path != "" {
		want := lifecycleTransactionHubStable
		if plan.Channel == channelDev {
			want = lifecycleTransactionHubDev
		}
		if plan.Allowlist.Hub.Channel != want {
			return preview, fmt.Errorf("maintenance mutation: %s operation cannot use %s hub", plan.Channel, plan.Allowlist.Hub.Channel)
		}
	}
	if plan.Channel == channelDev {
		for _, target := range plan.Targets {
			switch target.Root {
			case lifecycleTransactionRootClaudeHome, lifecycleTransactionRootOpenCodeHome, lifecycleTransactionRootCodexHome:
				return preview, fmt.Errorf("maintenance mutation: dev channel cannot write stable platform homes")
			}
		}
	}

	tx, err := beginLifecycleTransaction(lifecycleTransactionConfig{
		TransactionID: plan.TransactionID,
		Command:       plan.Operation,
		Allowlist:     plan.Allowlist,
		Rename:        plan.Rename,
		Fault:         plan.Fault,
	})
	if err != nil {
		return preview, err
	}
	seen := make(map[string]bool, len(plan.Targets))
	for index, target := range plan.Targets {
		if !target.Managed {
			return preview, fmt.Errorf("maintenance mutation: target %q has no managed ownership proof", target.RelativeTarget)
		}
		action := target.Action
		if action == "" {
			action = lifecycleTransactionWrite
		}
		if action != lifecycleTransactionWrite && action != lifecycleTransactionRemove {
			return preview, fmt.Errorf("maintenance mutation: target %q has invalid action %q", target.RelativeTarget, action)
		}
		root, targetPath, clean, err := tx.resolveTarget(target.Root, target.RelativeTarget)
		if err != nil {
			return preview, err
		}
		if seen[targetPath] {
			return preview, fmt.Errorf("maintenance mutation: duplicate target %q", targetPath)
		}
		seen[targetPath] = true
		current, err := readLifecycleFileState(targetPath)
		if err != nil {
			return preview, fmt.Errorf("maintenance mutation: read target baseline: %w", err)
		}
		if target.ExpectedDigest != "" && current.Digest != target.ExpectedDigest {
			return preview, fmt.Errorf("maintenance mutation: baseline changed for %q", targetPath)
		}
		desiredDigest := lifecycleTransactionMissingDigest
		desiredMode := os.FileMode(0)
		change := maintenanceMutationChangeRemove
		if action == lifecycleTransactionWrite {
			desiredDigest = lifecycleDigest(target.Content)
			desiredMode = target.Mode.Perm()
			if desiredMode == 0 {
				desiredMode = current.Mode.Perm()
			}
			if desiredMode == 0 {
				desiredMode = 0o644
			}
			change = maintenanceMutationChangeWrite
		}
		if current.Digest == desiredDigest && (action == lifecycleTransactionRemove || current.Mode.Perm() == desiredMode) {
			change = maintenanceMutationChangeUnchanged
		}
		entry := maintenanceMutationTargetPreview{
			Root: root.Kind, RelativeTarget: clean, Label: strings.TrimSpace(target.Label), Source: strings.TrimSpace(target.Source),
			Change: change, CurrentDigest: current.Digest, DesiredDigest: desiredDigest,
			CurrentMode: uint32(current.Mode.Perm()), DesiredMode: uint32(desiredMode.Perm()), CommitOrder: index + 1,
		}
		preview.Targets = append(preview.Targets, entry)
		preview.CommitOrder = append(preview.CommitOrder, fmt.Sprintf("%d:%s:%s", index+1, root.Kind, filepath.ToSlash(clean)))
	}
	return preview, nil
}

func commitMaintenanceMutation(plan maintenanceMutationPlan) (maintenanceMutationResult, error) {
	preview, err := prepareMaintenanceMutation(plan)
	result := maintenanceMutationResult{
		SchemaVersion: maintenanceMutationSchemaVersion,
		Operation:     strings.TrimSpace(plan.Operation),
		Preview:       preview,
		Targets:       append([]maintenanceMutationTargetPreview(nil), preview.Targets...),
		StateEffect:   colony.LifecycleStateEffectNone,
		Recovery:      strings.TrimSpace(plan.Recovery),
	}
	if err != nil {
		return result, err
	}
	tx, err := beginLifecycleTransaction(lifecycleTransactionConfig{
		TransactionID: plan.TransactionID,
		Command:       plan.Operation,
		Allowlist:     plan.Allowlist,
		Rename:        plan.Rename,
		Fault:         plan.Fault,
	})
	if err != nil {
		return result, err
	}
	declared := 0
	for index, target := range plan.Targets {
		if preview.Targets[index].Change == maintenanceMutationChangeUnchanged {
			continue
		}
		action := target.Action
		if action == "" {
			action = lifecycleTransactionWrite
		}
		if action == lifecycleTransactionRemove {
			err = tx.DeclareRemoval(target.Root, target.RelativeTarget)
		} else {
			err = tx.DeclareWriteWithMode(target.Root, target.RelativeTarget, target.Content, target.Mode)
		}
		if err != nil {
			return result, err
		}
		declaration := tx.declarations[len(tx.declarations)-1]
		if declaration.BeforeDigest != preview.Targets[index].CurrentDigest {
			return result, fmt.Errorf("maintenance mutation: baseline changed for %q", declaration.TargetPath)
		}
		if uint32(declaration.BeforeMode.Perm()) != preview.Targets[index].CurrentMode {
			return result, fmt.Errorf("maintenance mutation: baseline mode changed for %q", declaration.TargetPath)
		}
		declared++
	}
	if declared == 0 {
		if receipt, ok, loadErr := tx.loadCommittedReceipt(); ok || loadErr != nil {
			if loadErr != nil {
				return result, loadErr
			}
			result.StateEffect = receipt.StateEffect
			result.Receipt = &receipt
			result.Verification = append([]colony.LifecycleVerification(nil), receipt.Verification...)
			return result, nil
		}
		return result, nil
	}
	receipt, commitErr := tx.Commit()
	if commitErr == nil {
		result.StateEffect = receipt.StateEffect
		result.Receipt = &receipt
		result.Verification = append([]colony.LifecycleVerification(nil), receipt.Verification...)
		return result, nil
	}
	if tx.progress != nil && tx.intent != nil {
		switch tx.progress.StateEffect {
		case colony.LifecycleStateEffectRolledBack:
			rollbackReceipt := tx.rolledBackResult()
			result.StateEffect = rollbackReceipt.StateEffect
			result.Receipt = &rollbackReceipt
			result.Verification = append([]colony.LifecycleVerification(nil), rollbackReceipt.Verification...)
		case colony.LifecycleStateEffectRecoveryRequired:
			recoveryReceipt, _ := tx.recoveryResult(commitErr, lifecycleRecoveryProvenance(commitErr))
			result.StateEffect = colony.LifecycleStateEffectRecoveryRequired
			result.Receipt = &recoveryReceipt
			result.Recovery = recoveryReceipt.Recovery.SafeNextStep
		default:
			// A fault can arrive after replacement but before the coordinator
			// records its final effect. In a live process we can still restore
			// every staged pre-image; never report "none" or "committed"
			// without a durable receipt while bytes may have changed.
			if rollbackErr := tx.rollbackPreparedTargets(); rollbackErr == nil {
				rollbackReceipt := tx.rolledBackResult()
				result.StateEffect = rollbackReceipt.StateEffect
				result.Receipt = &rollbackReceipt
				result.Verification = append([]colony.LifecycleVerification(nil), rollbackReceipt.Verification...)
				result.Recovery = rollbackReceipt.Recovery.SafeNextStep
			} else {
				cause := fmt.Errorf("maintenance commit failed (%v) and rollback failed: %w", commitErr, rollbackErr)
				recoveryReceipt, recoveryErr := tx.recoveryResult(cause, lifecycleRecoveryProvenance(rollbackErr))
				result.StateEffect = colony.LifecycleStateEffectRecoveryRequired
				result.Receipt = &recoveryReceipt
				result.Recovery = recoveryReceipt.Recovery.SafeNextStep
				commitErr = recoveryErr
			}
		}
	}
	return result, commitErr
}

type maintenanceSyncSpec struct {
	Root                lifecycleTransactionRootKind
	Label               string
	SourceDir           string
	DestinationBase     string
	Options             syncOptions
	PruneRetiredAliases bool
	CleanupOwned        func(relativePath string, content []byte) bool
}

// appendMaintenanceSyncTargets converts an existing sync contract into a
// read-only exact manifest. Cleanup is intentionally conservative: only a
// generated Aether ownership header can authorize removal of a stale command.
func appendMaintenanceSyncTargets(plan *maintenanceMutationPlan, spec maintenanceSyncSpec) error {
	if plan == nil {
		return fmt.Errorf("maintenance sync: plan is required")
	}
	info, err := os.Lstat(spec.SourceDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("maintenance sync: inspect source: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("maintenance sync: source must be a real directory")
	}
	roots, err := resolveLifecycleTransactionRoots(plan.Allowlist)
	if err != nil {
		return err
	}
	root, ok := roots[spec.Root]
	if !ok {
		return fmt.Errorf("maintenance sync: destination root %s is not configured", spec.Root)
	}
	base := filepath.Clean(spec.DestinationBase)
	if base == "" {
		base = "."
	}
	if filepath.IsAbs(base) || base == ".." || strings.HasPrefix(base, ".."+string(filepath.Separator)) {
		return fmt.Errorf("maintenance sync: destination base escapes root")
	}
	sourceFiles, err := listMaintenanceRegularFiles(spec.SourceDir)
	if err != nil {
		return err
	}
	if spec.Options.include != nil {
		sourceFiles = filterSyncFiles(sourceFiles, spec.Options.include)
	}
	sourceFiles, _ = filterIgnoredSyncFiles(sourceFiles)
	destinationSet := make(map[string]bool, len(sourceFiles))
	added := make(map[string]bool)
	for _, sourceRel := range sourceFiles {
		destRel := mapSyncDestRelPath(sourceRel, spec.Options.mapRelPath)
		if destRel == "" || syncPathProtected(destRel, spec.Options.protectedDirs, spec.Options.protectedFiles) {
			continue
		}
		targetRel := filepath.Clean(filepath.Join(base, destRel))
		destinationSet[filepath.ToSlash(destRel)] = true
		sourcePath := filepath.Join(spec.SourceDir, sourceRel)
		content, err := os.ReadFile(sourcePath)
		if err != nil {
			return fmt.Errorf("maintenance sync: read %s: %w", sourcePath, err)
		}
		if spec.Options.validate != nil {
			if err := spec.Options.validate(sourcePath, sourceRel, content); err != nil {
				return err
			}
		}
		destinationPath := filepath.Join(root.Path, targetRel)
		if existing, readErr := os.ReadFile(destinationPath); readErr == nil {
			if spec.Options.merge != nil {
				content, err = spec.Options.merge(content, existing)
				if err != nil {
					return fmt.Errorf("maintenance sync: merge %s: %w", destinationPath, err)
				}
			} else if spec.Options.preserveLocalChanges && !bytes.Equal(content, existing) {
				content = existing
			}
		} else if !os.IsNotExist(readErr) {
			return fmt.Errorf("maintenance sync: read destination %s: %w", destinationPath, readErr)
		}
		plan.Targets = append(plan.Targets, maintenanceMutationTarget{
			Root: spec.Root, RelativeTarget: targetRel, Label: spec.Label, Source: sourcePath,
			Action: lifecycleTransactionWrite, Content: content, Managed: true,
		})
		added[filepath.ToSlash(targetRel)] = true
	}

	if !spec.Options.cleanup && !spec.PruneRetiredAliases {
		return nil
	}
	destinationDir := filepath.Join(root.Path, base)
	destFiles, err := listMaintenanceRegularFilesIfPresent(destinationDir)
	if err != nil {
		return err
	}
	cleanupFilter := spec.Options.cleanupInclude
	if cleanupFilter == nil {
		cleanupFilter = spec.Options.include
	}
	for _, destRel := range destFiles {
		if destinationSet[filepath.ToSlash(destRel)] || syncPathProtected(destRel, spec.Options.protectedDirs, spec.Options.protectedFiles) {
			continue
		}
		retired := spec.PruneRetiredAliases && isRetiredLifecycleWrapperPath(destRel)
		if !retired && (!spec.Options.cleanup || (cleanupFilter != nil && !cleanupFilter(destRel))) {
			continue
		}
		targetPath := filepath.Join(destinationDir, destRel)
		content, err := os.ReadFile(targetPath)
		if err != nil {
			return fmt.Errorf("maintenance sync: read cleanup target %s: %w", targetPath, err)
		}
		owned := isGeneratedAetherCommandWrapper(content)
		if !owned && spec.CleanupOwned != nil {
			owned = spec.CleanupOwned(destRel, content)
		}
		if !owned {
			continue
		}
		targetRel := filepath.Clean(filepath.Join(base, destRel))
		if added[filepath.ToSlash(targetRel)] {
			continue
		}
		plan.Targets = append(plan.Targets, maintenanceMutationTarget{
			Root: spec.Root, RelativeTarget: targetRel, Label: spec.Label, Source: "managed generated wrapper ownership header",
			Action: lifecycleTransactionRemove, Managed: true,
		})
		added[filepath.ToSlash(targetRel)] = true
	}
	return nil
}

func listMaintenanceRegularFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("maintenance sync: symbolic link is not an owned file: %s", path)
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("maintenance sync: non-regular source file: %s", path)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func listMaintenanceRegularFilesIfPresent(root string) ([]string, error) {
	if _, err := os.Lstat(root); os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return listMaintenanceRegularFiles(root)
}

type installSyncPair struct {
	srcRel               string
	destRel              string
	label                string
	cleanup              bool
	preserveLocalChanges bool
	validate             syncValidator
	include              syncFilter
	mapRelPath           syncRelPathMapper
	cleanupInclude       syncFilter
	cleanupLegacyClaude  bool
}

type repoSyncPair struct {
	hubRel               string
	destRel              string
	label                string
	cleanup              bool
	preserveLocalChanges bool
	validate             syncValidator
	include              syncFilter
	mapRelPath           syncRelPathMapper
	cleanupInclude       syncFilter
	cleanupLegacyClaude  bool
	consumerOnly         bool
	merge                syncMerger
}

type syncValidator func(srcPath, relPath string, data []byte) error
type syncFilter func(relPath string) bool
type syncRelPathMapper func(relPath string) string

// syncMerger combines shipped template bytes with existing destination bytes.
// Returning the destination bytes unchanged marks the file as up to date.
type syncMerger func(templateData, existingData []byte) ([]byte, error)

type codexAgentDefinition struct {
	Name                  string   `toml:"name"`
	Description           string   `toml:"description"`
	NicknameCandidates    []string `toml:"nickname_candidates"`
	DeveloperInstructions string   `toml:"developer_instructions"`
}

func installSyncPairs() []installSyncPair {
	return []installSyncPair{
		{srcRel: ".claude/commands/ant", destRel: ".claude/commands", label: "Commands (claude)", cleanup: true, mapRelPath: claudeCommandDestRelPath, cleanupInclude: isManagedNonRetiredFlatClaudeCommandPath, cleanupLegacyClaude: true},
		{srcRel: ".claude/agents/ant", destRel: ".claude/agents/ant", label: "Agents (claude)", cleanup: true},
		{srcRel: ".opencode/commands/ant", destRel: ".opencode/command", label: "Commands (opencode home)", cleanup: true, cleanupInclude: neverSyncPath, cleanupLegacyClaude: true},
		{srcRel: ".opencode/agents", destRel: ".opencode/agent", label: "Agents (opencode home)", cleanup: false, validate: validateOpenCodeAgentFile},
		{srcRel: ".opencode/commands/ant", destRel: ".config/opencode/commands/ant", label: "Commands (opencode)", cleanup: true, cleanupInclude: isNonRetiredCommandPath, cleanupLegacyClaude: true},
		{srcRel: ".opencode/agents", destRel: ".config/opencode/agents", label: "Agents (opencode)", cleanup: false, validate: validateOpenCodeAgentFile},
		{srcRel: ".codex/agents", destRel: ".codex/agents", label: "Agents (codex)", cleanup: false, preserveLocalChanges: true, validate: validateCodexAgentFile, include: isShippedAetherCodexAgent},
	}
}

func platformHomeHubSyncPairs() []installSyncPair {
	return []installSyncPair{
		{srcRel: "commands/claude", destRel: ".claude/commands", label: "Commands (claude)", cleanup: true, mapRelPath: claudeCommandDestRelPath, cleanupInclude: isManagedNonRetiredFlatClaudeCommandPath, cleanupLegacyClaude: true},
		{srcRel: "agents-claude", destRel: ".claude/agents/ant", label: "Agents (claude)", cleanup: true},
		{srcRel: "commands/opencode", destRel: ".opencode/command", label: "Commands (opencode home)", cleanup: true, cleanupInclude: neverSyncPath, cleanupLegacyClaude: true},
		{srcRel: "agents", destRel: ".opencode/agent", label: "Agents (opencode home)", cleanup: false, validate: validateOpenCodeAgentFile},
		{srcRel: "commands/opencode", destRel: ".config/opencode/commands/ant", label: "Commands (opencode)", cleanup: true, cleanupInclude: isNonRetiredCommandPath, cleanupLegacyClaude: true},
		{srcRel: "agents", destRel: ".config/opencode/agents", label: "Agents (opencode)", cleanup: false, validate: validateOpenCodeAgentFile},
		{srcRel: "codex", destRel: ".codex/agents", label: "Agents (codex)", cleanup: false, preserveLocalChanges: true, validate: validateCodexAgentFile, include: isShippedAetherCodexAgent},
	}
}

func repoSyncPairs() []repoSyncPair {
	return []repoSyncPair{
		{
			hubRel:         ".",
			destRel:        ".",
			label:          "Repo .aether cleanup",
			cleanup:        true,
			include:        neverSyncPath,
			cleanupInclude: isManagedAetherSystemPath,
			consumerOnly:   true,
		},
		{hubRel: "settings/claude", destRel: "../.claude", label: "Settings (claude)", preserveLocalChanges: true, include: isClaudeSettingsFile, merge: mergeClaudeSettings},
		{hubRel: "rules", destRel: "../.claude/rules", label: "Rules (claude)"},
	}
}

type codexSkillShim struct {
	Dir              string
	Name             string
	Description      string
	Body             string
	WorkflowTriggers []string
	TaskKeywords     []string
}

func codexSkillShims() []codexSkillShim {
	shims := []codexSkillShim{
		{
			Dir:         "aether-command-guide",
			Name:        "aether-command-guide",
			Description: "Use for Aether lifecycle commands; ask the runtime for current orchestration guidance before acting.",
			Body:        "Run `aether command-guide <command> --platform codex` before intelligent Aether flows. Follow the guide over stale local notes. For raw user commands, run the literal command.",
		},
		{
			Dir:         "aether-skill-loader",
			Name:        "aether-skill-loader",
			Description: "Explains where Aether worker skill content comes from -- no on-demand loader command exists.",
			Body:        "Skill content is already included automatically in the worker brief text returned by `aether build`, `aether colonize`, `aether plan`, and `aether continue` -- it is assembled in-process from the matched shipped and custom Aether skills. There is no separate command to fetch it on demand (skill-inject, the CLI command this shim used to call, was deleted in Phase 191 as dead CLI surface -- its underlying matching logic is what dispatches use automatically). Do not preload full skill mirrors.",
		},
		{
			Dir:         "aether-colony-creation",
			Name:        "aether-colony-creation",
			Description: "Use when initializing an Aether colony in Codex; refine intent before calling the runtime.",
			Body:        "For `aether init` or setup requests, use `aether command-guide init --platform codex`, ask compact clarifying questions when needed, ask the user to choose Colony Mode or Orchestrator Mode, synthesize a precise charter, then run the runtime with `--colony-mode <selected>` so it creates state.",
		},
		{
			Dir:         "aether-colony-research",
			Name:        "aether-colony-research",
			Description: "Use when running Oracle or discuss flows in Codex; scope research before persistence begins.",
			Body:        "For `aether oracle` or `aether discuss`, use `aether command-guide <oracle|discuss> --platform codex`, clarify output shape, scope, depth, and confidence, then run the runtime flow.",
		},
		{
			Dir:              "aether-colony-build-cycle",
			Name:             "aether-colony-build-cycle",
			Description:      "Use when Codex is asked to colonize, plan, build, continue, swarm, or seal an Aether colony and must mirror wrapper orchestration safely.",
			Body:             "For `aether colonize`, `aether plan`, `aether build`, `aether continue`, `aether swarm`, or `aether seal`, run `aether command-guide <command> --platform codex`, use runtime JSON manifests and finalizers, pass worker briefs verbatim, honor loop guards, and never hand-edit `.aether/data`.",
			WorkflowTriggers: []string{"colonize", "plan", "build", "continue", "swarm", "seal"},
			TaskKeywords:     []string{"aether colonize", "aether plan", "aether build", "aether continue", "aether swarm", "aether seal", "dispatch manifest", "plan-only", "finalize"},
		},
	}
	return append(shims, codexCommandSkillShims()...)
}

func codexCommandSkillShims() []codexSkillShim {
	commands := []string{"init", "discuss", "oracle", "colonize", "plan", "build", "continue", "swarm", "seal"}
	catalog := commandGuideCatalog()
	shims := make([]codexSkillShim, 0, len(commands))
	for _, command := range commands {
		def, ok := catalog[command]
		if !ok || def.Literal {
			continue
		}
		keywords := []string{
			"aether " + command,
			"/ant-" + command,
			"ant-" + command,
			"command-guide " + command,
			"aether command-guide " + command,
		}
		if def.SkillReference != "" {
			keywords = append(keywords, def.SkillReference)
		}
		shims = append(shims, codexSkillShim{
			Dir:              "aether-" + command,
			Name:             "aether-" + command,
			Description:      fmt.Sprintf("Use when Codex is asked to run `aether %s` or the equivalent Aether lifecycle action.", command),
			Body:             renderCodexCommandSkillShimBody(command, def),
			WorkflowTriggers: []string{command},
			TaskKeywords:     keywords,
		})
	}
	return shims
}

func renderCodexCommandSkillShimBody(command string, def commandGuideDefinition) string {
	var b strings.Builder
	fmt.Fprintf(&b, "This is the Codex command-shaped skill for `aether %s`. Use it instead of relying on free-form natural language for this lifecycle action.\n\n", command)
	fmt.Fprintf(&b, "1. Run `aether command-guide %s --platform codex` first and treat that runtime guide as authoritative.\n", command)
	if def.SkillReference != "" {
		fmt.Fprintf(&b, "2. Load or follow `%s`; this command-specific skill is the entrypoint, not a replacement for the shared lifecycle skill.\n", def.SkillReference)
	} else {
		b.WriteString("2. Follow the runtime guide directly.\n")
	}
	b.WriteString("3. Preserve runtime ownership of state: wrappers and skills may interview, synthesize, spawn workers, and summarize, but must not hand-edit `.aether/data`.\n")
	b.WriteString("4. Honor raw/exact/no-orchestration requests by using the raw bypass below.\n\n")

	if def.Intent != "" {
		fmt.Fprintf(&b, "## Intent\n%s\n\n", def.Intent)
	}
	writeCodexCommandSkillList(&b, "Pre-steps", def.PreSteps)
	if def.RunCommand != "" {
		fmt.Fprintf(&b, "## Runtime Command\n`%s`\n\n", def.RunCommand)
	}
	writeCodexCommandSkillList(&b, "Post-steps", def.PostSteps)
	writeCodexCommandSkillList(&b, "Drift Guards", def.DriftGuards)
	if def.RawBypass != "" {
		fmt.Fprintf(&b, "## Raw Bypass\n%s\n", def.RawBypass)
	}
	return strings.TrimSpace(b.String())
}

func writeCodexCommandSkillList(b *strings.Builder, heading string, values []string) {
	if len(values) == 0 {
		return
	}
	fmt.Fprintf(b, "## %s\n", heading)
	for _, value := range values {
		fmt.Fprintf(b, "- %s\n", value)
	}
	b.WriteString("\n")
}

func syncCodexSkillShims(destDir string) syncResult {
	result := syncResult{}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		result.errors = append(result.errors, fmt.Sprintf("mkdir %s: %v", destDir, err))
		return result
	}

	allowed := map[string]bool{}
	for _, shim := range codexSkillShims() {
		allowed[filepath.ToSlash(shim.Dir)] = true
	}

	for _, dir := range findSkillDirs(destDir) {
		rel, err := filepath.Rel(destDir, dir)
		if err != nil {
			result.errors = append(result.errors, fmt.Sprintf("rel %s: %v", dir, err))
			continue
		}
		rel = filepath.ToSlash(rel)
		if allowed[rel] {
			continue
		}
		if skillDirDeclaresSource(dir, "custom") {
			result.skipped++
			continue
		}
		if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
			result.errors = append(result.errors, fmt.Sprintf("remove %s: %v", dir, err))
			continue
		}
		result.removed = append(result.removed, rel)
	}

	for _, shim := range codexSkillShims() {
		skillPath := filepath.Join(destDir, filepath.FromSlash(shim.Dir), "SKILL.md")
		content := renderCodexSkillShim(shim)
		if current, err := os.ReadFile(skillPath); err == nil && string(current) == content {
			result.skipped++
			continue
		}
		if err := os.MkdirAll(filepath.Dir(skillPath), 0755); err != nil {
			result.errors = append(result.errors, fmt.Sprintf("mkdir %s: %v", filepath.Dir(skillPath), err))
			continue
		}
		if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
			result.errors = append(result.errors, fmt.Sprintf("write %s: %v", skillPath, err))
			continue
		}
		result.copied++
	}

	cleanEmptyDirs(destDir)
	return result
}

func renderCodexSkillShim(shim codexSkillShim) string {
	extraFrontmatter := ""
	if len(shim.WorkflowTriggers) > 0 {
		extraFrontmatter += fmt.Sprintf("workflow_triggers: [%s]\n", strings.Join(shim.WorkflowTriggers, ", "))
	}
	if len(shim.TaskKeywords) > 0 {
		extraFrontmatter += fmt.Sprintf("task_keywords: [%s]\n", strings.Join(shim.TaskKeywords, ", "))
	}
	return fmt.Sprintf(`---
name: %s
description: %s
source: shipped
type: codex-shim
domains: [aether, codex, orchestration]
%spriority: high
version: "1.0"
---

# %s

%s
`, shim.Name, shim.Description, extraFrontmatter, shim.Name, shim.Body)
}

func skillDirDeclaresSource(dir, expected string) bool {
	raw, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		return false
	}
	fm := parseSkillFrontmatter(string(raw))
	return fm != nil && strings.TrimSpace(fm.Source) == expected
}

func neverSyncPath(string) bool {
	return false
}

var managedAetherSystemDirs = map[string]bool{
	"agents":        true,
	"agents-claude": true,
	"agents-codex":  true,
	"codex":         true,
	"commands":      true,
	"docs":          true,
	"exchange":      true,
	"references":    true,
	"rules":         true,
	"schemas":       true,
	"settings":      true,
	"skills-codex":  true,
	"templates":     true,
	"ts":            true,
	"utils":         true,
}

var managedAetherSystemFiles = map[string]bool{
	".npmignore":          true,
	"aether-utils.sh":     true,
	"ledger.jsonl":        true,
	"manifest.json":       true,
	"model-profiles.yaml": true,
	"registry.json":       true,
	"version.json":        true,
	"workers.md":          true,
}

func isManagedAetherSystemPath(relPath string) bool {
	clean := filepath.ToSlash(filepath.Clean(relPath))
	if clean == "." || clean == "" {
		return false
	}
	first := clean
	if idx := strings.Index(clean, "/"); idx >= 0 {
		first = clean[:idx]
	}
	if managedAetherSystemDirs[first] {
		return true
	}
	if strings.Contains(clean, "/") {
		return false
	}
	return managedAetherSystemFiles[clean]
}

func isShippedAetherCodexAgent(relPath string) bool {
	base := filepath.Base(relPath)
	return filepath.Ext(base) == ".toml" && strings.HasPrefix(base, "aether-")
}

func isClaudeSettingsFile(relPath string) bool {
	return filepath.Base(relPath) == "settings.json"
}

func isOraclePhaseDirectivesFile(relPath string) bool {
	return filepath.Base(relPath) == "oracle-phase-directives.yaml"
}

func claudeCommandDestRelPath(relPath string) string {
	base := filepath.Base(filepath.Clean(relPath))
	if filepath.Ext(base) != ".md" {
		return relPath
	}
	if strings.HasPrefix(base, "ant-") {
		return base
	}
	return "ant-" + base
}

func isManagedFlatClaudeCommandPath(relPath string) bool {
	clean := filepath.ToSlash(filepath.Clean(relPath))
	if strings.Contains(clean, "/") {
		return false
	}
	base := filepath.Base(clean)
	return strings.HasPrefix(base, "ant-") && filepath.Ext(base) == ".md"
}

func isManagedNonRetiredFlatClaudeCommandPath(relPath string) bool {
	return isManagedFlatClaudeCommandPath(relPath) && isNonRetiredCommandPath(relPath)
}

func isNonRetiredCommandPath(relPath string) bool {
	return !isRetiredLifecycleWrapperPath(relPath)
}

func isRetiredLifecycleWrapperPath(path string) bool {
	base := filepath.Base(filepath.Clean(path))
	if filepath.Ext(base) != ".md" {
		return false
	}
	name := strings.TrimSuffix(base, ".md")
	name = strings.TrimPrefix(name, "ant-")
	return name == "pause-colony" || name == "resume-colony"
}

// isGeneratedAetherCommandWrapper marks a file as Aether-managed for
// update/prune. It accepts both the current header and the legacy
// "Generated from" form so downstream repos installed before the header
// reform still get their stale wrappers pruned.
func isGeneratedAetherCommandWrapper(data []byte) bool {
	firstLine := strings.SplitN(string(data), "\n", 2)[0]
	if strings.HasPrefix(firstLine, "<!-- Aether-managed: runtime spec at .aether/commands/") &&
		strings.HasSuffix(firstLine, ". Synced by aether update. -->") {
		return true
	}
	return strings.HasPrefix(firstLine, "<!-- Generated from .aether/commands/") &&
		strings.HasSuffix(firstLine, ".yaml - DO NOT EDIT DIRECTLY -->")
}

func removeLegacyClaudeCommandNamespace(commandsDir string) ([]string, []string) {
	var removed []string
	var errs []string

	// Platform command homes can contain user-authored files beside Aether's
	// generated wrappers. Generic filename cleanup cannot distinguish those
	// owners, so only the managed header may authorize a stale-wrapper removal.
	if homeDir, ok := platformHomeFromCommandDir(commandsDir); ok {
		for _, commandDir := range platformCommandHomeDirs(homeDir) {
			pruned := pruneRetiredGeneratedCommandFiles(commandDir)
			for _, rel := range pruned.removed {
				removed = append(removed, filepath.ToSlash(filepath.Join(commandDir, rel)))
			}
			errs = append(errs, pruned.errors...)
		}
	}

	// Only Claude's old nested namespace is retired wholesale. The same
	// function is also invoked after OpenCode sync so parser-only files copied
	// later in the pair order are pruned before the operation completes.
	if !strings.HasSuffix(filepath.ToSlash(filepath.Clean(commandsDir)), "/.claude/commands") {
		return removed, errs
	}
	legacyDir := filepath.Join(commandsDir, "ant")
	entries, err := os.ReadDir(legacyDir)
	if err != nil {
		if os.IsNotExist(err) {
			return removed, errs
		}
		errs = append(errs, fmt.Sprintf("read legacy Claude commands %s: %v", legacyDir, err))
		return removed, errs
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		path := filepath.Join(legacyDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("read legacy Claude command %s: %v", path, err))
			continue
		}
		if !isGeneratedAetherCommandWrapper(data) {
			continue
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			errs = append(errs, fmt.Sprintf("remove legacy Claude command %s: %v", path, err))
			continue
		}
		removed = append(removed, filepath.Join("ant", entry.Name()))
	}

	if len(removed) > 0 {
		if err := os.Remove(legacyDir); err != nil && !os.IsNotExist(err) {
			if entries, readErr := os.ReadDir(legacyDir); readErr == nil && len(entries) > 0 {
				return removed, errs
			}
			errs = append(errs, fmt.Sprintf("remove legacy Claude command namespace %s: %v", legacyDir, err))
		}
	}

	return removed, errs
}

func platformHomeFromCommandDir(commandsDir string) (string, bool) {
	clean := filepath.ToSlash(filepath.Clean(commandsDir))
	for _, suffix := range []string{
		".claude/commands",
		".opencode/command",
		".config/opencode/commands/ant",
	} {
		needle := "/" + suffix
		if strings.HasSuffix(clean, needle) {
			home := strings.TrimSuffix(clean, needle)
			if home != "" {
				return filepath.FromSlash(home), true
			}
		}
	}
	return "", false
}

func platformCommandHomeDirs(homeDir string) []string {
	return []string{
		filepath.Join(homeDir, ".claude", "commands"),
		filepath.Join(homeDir, ".opencode", "command"),
		filepath.Join(homeDir, ".config", "opencode", "commands", "ant"),
	}
}

func appendSyncResult(details *[]map[string]interface{}, totals *updateSyncResult, label string, result syncResult) {
	entry := map[string]interface{}{
		"label":   label,
		"copied":  result.copied,
		"skipped": result.skipped,
		"removed": len(result.removed),
	}
	if len(result.errors) > 0 {
		entry["errors"] = result.errors
		totals.errors = append(totals.errors, result.errors...)
	}
	*details = append(*details, entry)
	totals.copied += result.copied
	totals.skipped += result.skipped
}

// declaredAliasSurfaceStatus names one platform-home destination where a
// declared alias command's wrapper must exist for that alias to actually
// work from that surface.
type declaredAliasSurfaceStatus struct {
	Alias string
	Label string
	Path  string
}

// aliasWrapperHomeSurfaces returns each platform-home destination path a
// command's wrapper is synced to, paired with a plain-English label for the
// surface. These mirror platformHomeHubSyncPairs's actual destinations.
func aliasWrapperHomeSurfaces(homeDir, name string) []declaredAliasSurfaceStatus {
	return []declaredAliasSurfaceStatus{
		{Alias: name, Label: "Claude", Path: filepath.Join(homeDir, ".claude", "commands", "ant-"+name+".md")},
		{Alias: name, Label: "OpenCode", Path: filepath.Join(homeDir, ".opencode", "command", name+".md")},
		{Alias: name, Label: "OpenCode (project config)", Path: filepath.Join(homeDir, ".config", "opencode", "commands", "ant", name+".md")},
	}
}

// declaredAliasSurfaces reads the alias declarations directly off the live
// Cobra command tree -- the same field cobra.Command.Find uses to route
// `aether pause-colony` to the `pause` handler -- rather than maintaining a
// second, feature-specific list of alias names in Go. A future command that
// declares an alias is picked up here automatically.
func declaredAliasSurfaces(homeDir string) []declaredAliasSurfaceStatus {
	var out []declaredAliasSurfaceStatus
	seen := map[string]bool{}
	for _, sub := range rootCmd.Commands() {
		for _, alias := range sub.Aliases {
			alias = strings.TrimSpace(alias)
			if alias == "" || seen[alias] || !wrapperCommandNames[alias] {
				continue
			}
			seen[alias] = true
			out = append(out, aliasWrapperHomeSurfaces(homeDir, alias)...)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Alias != out[j].Alias {
			return out[i].Alias < out[j].Alias
		}
		return out[i].Label < out[j].Label
	})
	return out
}

// missingDeclaredAliasSurfaces reports which declared-alias platform
// surfaces are absent from homeDir right now.
func missingDeclaredAliasSurfaces(homeDir string) []declaredAliasSurfaceStatus {
	var missing []declaredAliasSurfaceStatus
	for _, surface := range declaredAliasSurfaces(homeDir) {
		if _, err := os.Stat(surface.Path); err != nil {
			missing = append(missing, surface)
		}
	}
	return missing
}

// aliasWrapperRepair names one alias command and every platform surface it
// was missing from before an update repaired it.
type aliasWrapperRepair struct {
	Alias  string   `json:"alias"`
	Labels []string `json:"surfaces"`
}

// aliasWrapperRepairReport is the plain-words account of which declared
// alias wrappers an update run restored -- distinct from the ordinary
// copied/unchanged file counts, and distinct from the stale-publish signal:
// this is "a command came back", not "republish the hub".
type aliasWrapperRepairReport struct {
	Repairs []aliasWrapperRepair
}

func (r aliasWrapperRepairReport) Empty() bool {
	return len(r.Repairs) == 0
}

// Message renders the repair report in plain English, naming each restored
// command and the surface(s) it was missing from. Empty when nothing was
// repaired -- an update that reports a repair on every run is noise, and
// noise is how a real repair gets ignored.
func (r aliasWrapperRepairReport) Message() string {
	if r.Empty() {
		return ""
	}
	parts := make([]string, 0, len(r.Repairs))
	for _, repair := range r.Repairs {
		parts = append(parts, fmt.Sprintf("`%s` (missing from %s)", repair.Alias, strings.Join(repair.Labels, ", ")))
	}
	plural := ""
	if len(r.Repairs) != 1 {
		plural = "s"
	}
	return fmt.Sprintf("Restored missing command%s: %s.", plural, strings.Join(parts, "; "))
}

// diffAliasRepairs compares the alias surfaces that were missing before a
// sync ran against what exists now, and reports only the ones the sync
// actually restored. A surface that was missing before and is still missing
// after (e.g. because the platform-home sync was skipped, or the hub itself
// never shipped that wrapper) is not a repair -- silence, not a false claim.
func diffAliasRepairs(missingBefore []declaredAliasSurfaceStatus) aliasWrapperRepairReport {
	var report aliasWrapperRepairReport
	byAlias := map[string][]string{}
	var order []string
	for _, surface := range missingBefore {
		if _, err := os.Stat(surface.Path); err != nil {
			continue // still missing -- not a repair
		}
		if _, seen := byAlias[surface.Alias]; !seen {
			order = append(order, surface.Alias)
		}
		byAlias[surface.Alias] = append(byAlias[surface.Alias], surface.Label)
	}
	for _, alias := range order {
		report.Repairs = append(report.Repairs, aliasWrapperRepair{Alias: alias, Labels: byAlias[alias]})
	}
	return report
}

func pruneLegacyRepoPlatformAssets(repoDir string) syncResult {
	result := syncResult{}
	if isAetherSourceCheckout(repoDir) {
		return result
	}

	pruners := []struct {
		label string
		fn    func() syncResult
	}{
		{
			label: "claude commands",
			fn: func() syncResult {
				return pruneGeneratedCommandFiles(filepath.Join(repoDir, ".claude", "commands"))
			},
		},
		{
			label: "opencode commands",
			fn: func() syncResult {
				return pruneGeneratedCommandFiles(filepath.Join(repoDir, ".opencode", "commands", "ant"))
			},
		},
		{
			label: "claude agents",
			fn: func() syncResult {
				return pruneAetherNamedFiles(filepath.Join(repoDir, ".claude", "agents", "ant"), ".md")
			},
		},
		{
			label: "opencode agents",
			fn: func() syncResult {
				return pruneAetherNamedFiles(filepath.Join(repoDir, ".opencode", "agents"), ".md")
			},
		},
		{
			label: "codex agents",
			fn: func() syncResult {
				return pruneAetherNamedFiles(filepath.Join(repoDir, ".codex", "agents"), ".toml")
			},
		},
		{
			label: "codex skills",
			fn: func() syncResult {
				return pruneDirectoryTree(filepath.Join(repoDir, ".codex", "skills", "aether"))
			},
		},
	}

	for _, pruner := range pruners {
		pruned := pruner.fn()
		for _, removed := range pruned.removed {
			result.removed = append(result.removed, filepath.ToSlash(filepath.Join(pruner.label, removed)))
		}
		result.errors = append(result.errors, pruned.errors...)
	}
	return result
}

func pruneGeneratedCommandFiles(dir string) syncResult {
	return pruneGeneratedCommandFilesMatching(dir, func(string) bool { return true })
}

// pruneRetiredGeneratedCommandFiles removes only generated wrappers for the
// two bounded parser-only lifecycle tokens. A freshly supplied command may not
// exist in this binary's wrapper registry yet, so registry absence alone can
// never authorize deletion.
func pruneRetiredGeneratedCommandFiles(dir string) syncResult {
	return pruneGeneratedCommandFilesMatching(dir, isRetiredLifecycleWrapperPath)
}

func pruneGeneratedCommandFilesMatching(dir string, shouldRemove func(path string) bool) syncResult {
	result := syncResult{}
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return result
		}
		result.errors = append(result.errors, fmt.Sprintf("stat %s: %v", dir, err))
		return result
	}
	if !info.IsDir() {
		return result
	}

	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		if shouldRemove != nil && !shouldRemove(path) {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			result.errors = append(result.errors, fmt.Sprintf("read %s: %v", path, readErr))
			return nil
		}
		if !isGeneratedAetherCommandWrapper(data) {
			return nil
		}
		if removeErr := os.Remove(path); removeErr != nil && !os.IsNotExist(removeErr) {
			result.errors = append(result.errors, fmt.Sprintf("remove %s: %v", path, removeErr))
			return nil
		}
		if rel, relErr := filepath.Rel(dir, path); relErr == nil {
			result.removed = append(result.removed, filepath.ToSlash(rel))
		}
		return nil
	})
	if len(result.removed) > 0 {
		cleanEmptyDirs(dir)
	}
	return result
}

func pruneAetherNamedFiles(dir string, extensions ...string) syncResult {
	result := syncResult{}
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return result
		}
		result.errors = append(result.errors, fmt.Sprintf("stat %s: %v", dir, err))
		return result
	}
	if !info.IsDir() {
		return result
	}

	allowed := map[string]bool{}
	for _, ext := range extensions {
		allowed[ext] = true
	}

	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		base := filepath.Base(path)
		if !strings.HasPrefix(base, "aether-") {
			return nil
		}
		if len(allowed) > 0 && !allowed[filepath.Ext(base)] {
			return nil
		}
		if removeErr := os.Remove(path); removeErr != nil && !os.IsNotExist(removeErr) {
			result.errors = append(result.errors, fmt.Sprintf("remove %s: %v", path, removeErr))
			return nil
		}
		if rel, relErr := filepath.Rel(dir, path); relErr == nil {
			result.removed = append(result.removed, filepath.ToSlash(rel))
		}
		return nil
	})
	if len(result.removed) > 0 {
		cleanEmptyDirs(dir)
	}
	return result
}

func pruneDirectoryTree(dir string) syncResult {
	result := syncResult{}
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return result
		}
		result.errors = append(result.errors, fmt.Sprintf("stat %s: %v", dir, err))
		return result
	}
	if !info.IsDir() {
		return result
	}
	if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
		result.errors = append(result.errors, fmt.Sprintf("remove %s: %v", dir, err))
		return result
	}
	result.removed = append(result.removed, filepath.Base(dir))
	return result
}

func pruneShippedRepoSkills(hubSystem, localAether string, force bool) syncResult {
	result := syncResult{}
	if isAetherSourceCheckout(filepath.Dir(localAether)) {
		return result
	}

	hubSkills := filepath.Join(hubSystem, "skills")
	localSkills := filepath.Join(localAether, "skills")
	info, err := os.Stat(localSkills)
	if err != nil {
		if os.IsNotExist(err) {
			return result
		}
		result.errors = append(result.errors, fmt.Sprintf("stat %s: %v", localSkills, err))
		return result
	}
	if !info.IsDir() {
		return result
	}

	for _, rel := range listFilesRecursive(localSkills) {
		if syncPathIgnored(rel) {
			continue
		}
		localPath := filepath.Join(localSkills, rel)
		hubPath := filepath.Join(hubSkills, rel)
		if _, err := os.Stat(hubPath); err != nil {
			if !os.IsNotExist(err) {
				result.errors = append(result.errors, fmt.Sprintf("stat %s: %v", hubPath, err))
			}
			continue
		}
		remove := force
		if !remove {
			localHash, localErr := fileSHA256(localPath)
			hubHash, hubErr := fileSHA256(hubPath)
			remove = localErr == nil && hubErr == nil && localHash == hubHash
		}
		if !remove {
			result.skipped++
			continue
		}
		if err := os.Remove(localPath); err != nil && !os.IsNotExist(err) {
			result.errors = append(result.errors, fmt.Sprintf("remove %s: %v", localPath, err))
			continue
		}
		result.removed = append(result.removed, filepath.ToSlash(rel))
	}
	if len(result.removed) > 0 {
		cleanEmptyDirs(localSkills)
	}
	return result
}

func pruneShippedFromUserSkillsDir(hubSystem, hubDir string) syncResult {
	result := syncResult{}
	shippedSkills := filepath.Join(hubSystem, "skills")
	userSkills := filepath.Join(hubDir, "skills")
	info, err := os.Stat(userSkills)
	if err != nil {
		if os.IsNotExist(err) {
			return result
		}
		result.errors = append(result.errors, fmt.Sprintf("stat %s: %v", userSkills, err))
		return result
	}
	if !info.IsDir() {
		return result
	}

	for _, rel := range listFilesRecursive(userSkills) {
		if syncPathIgnored(rel) {
			continue
		}
		userPath := filepath.Join(userSkills, rel)
		shippedPath := filepath.Join(shippedSkills, rel)
		if _, err := os.Stat(shippedPath); err != nil {
			if !os.IsNotExist(err) {
				result.errors = append(result.errors, fmt.Sprintf("stat %s: %v", shippedPath, err))
			}
			continue
		}
		userHash, userErr := fileSHA256(userPath)
		shippedHash, shippedErr := fileSHA256(shippedPath)
		if userErr != nil || shippedErr != nil || userHash != shippedHash {
			result.skipped++
			continue
		}
		if err := os.Remove(userPath); err != nil && !os.IsNotExist(err) {
			result.errors = append(result.errors, fmt.Sprintf("remove %s: %v", userPath, err))
			continue
		}
		result.removed = append(result.removed, filepath.ToSlash(rel))
	}
	if len(result.removed) > 0 {
		cleanEmptyDirs(userSkills)
	}
	return result
}

func pruneRepoCodexSkillMirror(repoDir string, force bool) syncResult {
	result := syncResult{}
	if !force || isAetherSourceCheckout(repoDir) {
		return result
	}
	root := filepath.Join(repoDir, ".codex", "skills", "aether")
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return result
		}
		result.errors = append(result.errors, fmt.Sprintf("stat %s: %v", root, err))
		return result
	}
	if !info.IsDir() {
		return result
	}

	for _, dir := range findSkillDirs(root) {
		if skillDirDeclaresSource(dir, "custom") {
			result.skipped++
			continue
		}
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			result.errors = append(result.errors, fmt.Sprintf("rel %s: %v", dir, err))
			continue
		}
		if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
			result.errors = append(result.errors, fmt.Sprintf("remove %s: %v", dir, err))
			continue
		}
		result.removed = append(result.removed, filepath.ToSlash(rel))
	}
	if len(result.removed) > 0 {
		cleanEmptyDirs(root)
	}
	return result
}

func ensureRepoLocalScaffold(localAether string) syncResult {
	result := syncResult{}
	for _, dir := range []string{"data", "dreams", "oracle", "checkpoints", "locks"} {
		path := filepath.Join(localAether, dir)
		if _, err := os.Stat(path); err == nil {
			result.skipped++
			continue
		}
		if err := os.MkdirAll(path, 0755); err != nil {
			result.errors = append(result.errors, fmt.Sprintf("mkdir %s: %v", path, err))
			continue
		}
		result.copied++
	}

	// A durable-state marker, because .aether/ is indistinguishable from a
	// build cache to an outside observer: repo root, untracked, dominated by
	// ts-host/node_modules. A routine disk cleanup deleted one on 2026-08-16
	// for exactly that reason (.planning/field-reports/
	// 2026-08-16-init-obsidian-vault.md §4) — no colony existed there, but a
	// running colony would have been destroyed.
	markerPath := filepath.Join(localAether, "WHAT-IS-THIS.md")
	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		marker := `# What is this directory?

This is Aether's colony state for this repository — durable working memory,
not a build cache. Deleting it destroys any colony running here: its goal,
phase plan, learned lessons, and steering signals.

Safe to delete: ts-host/node_modules/ only (npm packages, ~60 MB — Aether
reinstalls them on demand).

Everything else here should be treated like your project's own files.
Managed by the aether CLI (https://github.com/calcosmic/Aether).
`
		if writeErr := os.WriteFile(markerPath, []byte(marker), 0644); writeErr != nil {
			result.errors = append(result.errors, fmt.Sprintf("write %s: %v", markerPath, writeErr))
		} else {
			result.copied++
		}
	} else if err == nil {
		result.skipped++
	}

	gitignorePath := filepath.Join(localAether, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		// ts-host/node_modules is ~60 MB of npm packages installed by
		// `aether update`; without this line a `git add .aether` (which the
		// versioned QUEEN.md invites) commits all of it.
		content := "# Aether local state - not versioned\ndata/\ncheckpoints/\nlocks/\ndreams/\noracle/\nts-host/node_modules/\n"
		if writeErr := os.WriteFile(gitignorePath, []byte(content), 0644); writeErr != nil {
			result.errors = append(result.errors, fmt.Sprintf("write %s: %v", gitignorePath, writeErr))
		} else {
			result.copied++
		}
	} else if err == nil {
		// Idempotent upgrade for repos scaffolded before the ts-host ignore
		// line existed: append it once, never rewrite user content.
		if data, readErr := os.ReadFile(gitignorePath); readErr == nil && !strings.Contains(string(data), "ts-host/node_modules") {
			appended := strings.TrimRight(string(data), "\n") + "\nts-host/node_modules/\n"
			if writeErr := os.WriteFile(gitignorePath, []byte(appended), 0644); writeErr != nil {
				result.errors = append(result.errors, fmt.Sprintf("append %s: %v", gitignorePath, writeErr))
			} else {
				result.copied++
			}
		} else {
			result.skipped++
		}
	} else {
		result.errors = append(result.errors, fmt.Sprintf("stat %s: %v", gitignorePath, err))
	}

	queenPath := filepath.Join(localAether, "QUEEN.md")
	if _, err := os.Stat(queenPath); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(queenPath), 0755); err != nil {
			result.errors = append(result.errors, fmt.Sprintf("mkdir %s: %v", filepath.Dir(queenPath), err))
		} else if err := os.WriteFile(queenPath, []byte(queenDefaultContent), 0644); err != nil {
			result.errors = append(result.errors, fmt.Sprintf("write %s: %v", queenPath, err))
		} else {
			result.copied++
		}
	} else if err == nil {
		result.skipped++
	} else {
		result.errors = append(result.errors, fmt.Sprintf("stat %s: %v", queenPath, err))
	}

	return result
}

func validateCodexAgentFile(srcPath, relPath string, data []byte) error {
	if filepath.Ext(relPath) != ".toml" {
		return fmt.Errorf("%s must use the .toml extension", relPath)
	}
	if !utf8.Valid(data) {
		return fmt.Errorf("%s is not valid UTF-8 text", relPath)
	}

	var agent codexAgentDefinition
	if err := toml.Unmarshal(data, &agent); err != nil {
		return fmt.Errorf("%s is not valid TOML: %w", relPath, err)
	}

	baseName := strings.TrimSuffix(filepath.Base(relPath), filepath.Ext(relPath))
	switch {
	case strings.TrimSpace(agent.Name) == "":
		return fmt.Errorf("%s is missing name", relPath)
	case agent.Name != baseName:
		return fmt.Errorf("%s name %q does not match filename %q", relPath, agent.Name, baseName)
	case strings.TrimSpace(agent.Description) == "":
		return fmt.Errorf("%s is missing description", relPath)
	case len(agent.NicknameCandidates) < 2:
		return fmt.Errorf("%s must define at least 2 nickname_candidates", relPath)
	case strings.TrimSpace(agent.DeveloperInstructions) == "":
		return fmt.Errorf("%s is missing developer_instructions", relPath)
	}

	// Reject binary-like content masquerading as text by ensuring the source can
	// be read back as a regular file. This keeps the validator conservative while
	// still allowing normal multiline TOML strings.
	if info, err := os.Stat(srcPath); err == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", relPath)
	}

	return nil
}

// openCodeAgentFrontmatter defines the expected YAML fields for an OpenCode
// agent file. The `name` field is required — it identifies the agent to the
// OpenCode runtime.
type openCodeAgentFrontmatter struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Mode        string                 `yaml:"mode"`
	Tools       map[string]interface{} `yaml:"tools"`
	Color       string                 `yaml:"color"`
	Model       string                 `yaml:"model"`
}

var openCodeThemeColors = map[string]bool{
	"primary": true, "secondary": true, "accent": true,
	"success": true, "warning": true, "error": true, "info": true,
}

var openCodeHexColorRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// validateOpenCodeAgentFile validates an OpenCode agent markdown file.
// It checks that the YAML frontmatter conforms to the OpenCode agent schema:
// name (required), description (20+ chars), tools (object/map), color (hex or theme),
// and model (provider/model-id format).
func validateOpenCodeAgentFile(srcPath, relPath string, data []byte) error {
	// Rule 1: must have .md extension
	if filepath.Ext(relPath) != ".md" {
		return fmt.Errorf("%s must use the .md extension", relPath)
	}

	// Rule 2: must be valid UTF-8
	if !utf8.Valid(data) {
		return fmt.Errorf("%s is not valid UTF-8 text", relPath)
	}

	// Rule 3: must have YAML frontmatter between --- delimiters
	content := string(data)
	start := strings.Index(content, "---")
	if start == -1 {
		return fmt.Errorf("%s is missing YAML frontmatter (no opening ---)", relPath)
	}
	end := strings.Index(content[start+3:], "---")
	if end == -1 {
		return fmt.Errorf("%s is missing YAML frontmatter (no closing ---)", relPath)
	}
	yamlContent := content[start+3 : start+3+end]

	var fm openCodeAgentFrontmatter
	if err := yaml.Unmarshal([]byte(yamlContent), &fm); err != nil {
		return fmt.Errorf("%s has invalid YAML frontmatter: %w", relPath, err)
	}

	// Rule 4: description must be present and at least 20 characters
	desc := strings.TrimSpace(fm.Description)
	if desc == "" {
		return fmt.Errorf("%s is missing description in frontmatter", relPath)
	}
	if len(desc) < 20 {
		return fmt.Errorf("%s description too short (%d chars, need at least 20): %q", relPath, len(desc), desc)
	}

	// Rule 5: mode must be a valid value
	mode := strings.TrimSpace(fm.Mode)
	if mode == "" {
		return fmt.Errorf("%s is missing mode in frontmatter", relPath)
	}
	if mode != "primary" && mode != "subagent" && mode != "all" {
		return fmt.Errorf("%s mode %q must be primary, subagent, or all", relPath, mode)
	}

	// Rule 6: tools must be a map/object (not a string, not nil)
	if fm.Tools == nil {
		return fmt.Errorf("%s is missing tools field in frontmatter", relPath)
	}
	// Also check the raw YAML to detect tools as a string (yaml.Unmarshal
	// would not error on that but would produce nil map). Re-parse the raw
	// frontmatter to check the actual type of tools.
	var rawFM map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlContent), &rawFM); err != nil {
		return fmt.Errorf("%s has invalid YAML: %w", relPath, err)
	}
	rawTools := rawFM["tools"]
	if rawTools == nil {
		return fmt.Errorf("%s is missing tools field in frontmatter", relPath)
	}
	if _, ok := rawTools.(map[string]interface{}); !ok {
		if _, isStr := rawTools.(string); isStr {
			return fmt.Errorf("%s tools must be a map/object with true/false values, not a string", relPath)
		}
		return fmt.Errorf("%s tools has unexpected type %T (must be a map/object)", relPath, rawTools)
	}

	// Rule 7: color must be a hex color or a theme color name
	color := strings.TrimSpace(fm.Color)
	if color == "" {
		return fmt.Errorf("%s is missing color in frontmatter", relPath)
	}
	if !openCodeHexColorRe.MatchString(color) && !openCodeThemeColors[color] {
		return fmt.Errorf("%s color %q must be a hex color (#rrggbb) or a theme color (primary, secondary, accent, success, warning, error, info)", relPath, color)
	}

	// Rule 8: name field is required
	if strings.TrimSpace(fm.Name) == "" {
		return fmt.Errorf("%s is missing name in frontmatter", relPath)
	}

	// Rule 9: model is optional — when absent, OpenCode uses its global default

	// Reject binary-like content masquerading as text
	if info, err := os.Stat(srcPath); err == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", relPath)
	}

	return nil
}
