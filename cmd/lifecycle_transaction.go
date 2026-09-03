package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

const (
	lifecycleTransactionMissingDigest = "missing"
	lifecycleTransactionDirectory     = ".aether-transactions"
)

var (
	lifecycleTransactionIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
	lifecycleTransactionProcessMu sync.Mutex
)

// lifecycleTransactionRootKind is deliberately closed over the only roots a
// lifecycle command may mutate. Callers cannot turn an arbitrary path into a
// transaction root by placing it in an untyped map.
type lifecycleTransactionRootKind string

const (
	lifecycleTransactionRootRepository        lifecycleTransactionRootKind = "repository"
	lifecycleTransactionRootData              lifecycleTransactionRootKind = "lifecycle_data"
	lifecycleTransactionRootHubStable         lifecycleTransactionRootKind = "hub_stable"
	lifecycleTransactionRootHubDev            lifecycleTransactionRootKind = "hub_dev"
	lifecycleTransactionRootClaudeHome        lifecycleTransactionRootKind = "claude_home"
	lifecycleTransactionRootOpenCodeHome      lifecycleTransactionRootKind = "opencode_home"
	lifecycleTransactionRootCodexHome         lifecycleTransactionRootKind = "codex_home"
	lifecycleTransactionRootBinaryDestination lifecycleTransactionRootKind = "binary_destination"
)

func (kind lifecycleTransactionRootKind) valid() bool {
	switch kind {
	case lifecycleTransactionRootRepository,
		lifecycleTransactionRootData,
		lifecycleTransactionRootHubStable,
		lifecycleTransactionRootHubDev,
		lifecycleTransactionRootClaudeHome,
		lifecycleTransactionRootOpenCodeHome,
		lifecycleTransactionRootCodexHome,
		lifecycleTransactionRootBinaryDestination:
		return true
	default:
		return false
	}
}

type lifecycleTransactionHubChannel string

const (
	lifecycleTransactionHubStable lifecycleTransactionHubChannel = "stable"
	lifecycleTransactionHubDev    lifecycleTransactionHubChannel = "dev"
)

type lifecycleTransactionHubRoot struct {
	Channel lifecycleTransactionHubChannel
	Path    string
}

// lifecycleTransactionAllowlist uses named fields instead of caller-provided
// path maps. The binary destination is an exact file; every other entry is a
// directory root whose descendants may be declared explicitly.
type lifecycleTransactionAllowlist struct {
	RepositoryRoot    string
	LifecycleDataRoot string
	Hub               lifecycleTransactionHubRoot
	ClaudeHome        string
	OpenCodeHome      string
	CodexHome         string
	BinaryDestination string
}

type lifecycleTransactionFaultHook func(point string) error

type lifecycleTransactionFaultError struct {
	point string
	cause error
}

func (err *lifecycleTransactionFaultError) Error() string {
	return fmt.Sprintf("lifecycle transaction interrupted at %s: %v", err.point, err.cause)
}

func (err *lifecycleTransactionFaultError) Unwrap() error {
	return err.cause
}

type lifecycleTransactionConfig struct {
	TransactionID string
	Command       string
	Allowlist     lifecycleTransactionAllowlist
	Rename        func(oldPath, newPath string) error
	Fault         lifecycleTransactionFaultHook
}

type lifecycleTransaction struct {
	config       lifecycleTransactionConfig
	roots        map[lifecycleTransactionRootKind]lifecycleResolvedRoot
	declarations []*lifecycleTransactionDeclaration
	targets      map[string]struct{}
	validated    bool
	intent       *lifecycleTransactionIntent
	progress     *lifecycleTransactionProgress
	journalStore *storage.Store
}

type lifecycleResolvedRoot struct {
	Kind        lifecycleTransactionRootKind
	Path        string
	ExactTarget string
}

type lifecycleTransactionAction string

const (
	lifecycleTransactionWrite  lifecycleTransactionAction = "write"
	lifecycleTransactionRemove lifecycleTransactionAction = "remove"
)

type lifecycleTransactionDeclaration struct {
	ID             string
	Root           lifecycleResolvedRoot
	RelativeTarget string
	TargetPath     string
	Action         lifecycleTransactionAction
	Content        []byte
	BeforeExists   bool
	BeforeDigest   string
	AfterDigest    string
	Mode           os.FileMode
}

type lifecycleTransactionRootManifest struct {
	SchemaVersion string                               `json:"schema_version"`
	TransactionID string                               `json:"transaction_id"`
	RootID        string                               `json:"root_id"`
	Kind          lifecycleTransactionRootKind         `json:"kind"`
	RootPath      string                               `json:"root_path"`
	Targets       []lifecycleTransactionTargetManifest `json:"targets"`
}

type lifecycleTransactionTargetManifest struct {
	ID             string                     `json:"id"`
	RelativeTarget string                     `json:"relative_target"`
	TargetPath     string                     `json:"target_path"`
	Action         lifecycleTransactionAction `json:"action"`
	BeforeExists   bool                       `json:"before_exists"`
	BeforeDigest   string                     `json:"before_digest"`
	AfterDigest    string                     `json:"after_digest"`
	Mode           uint32                     `json:"mode"`
	StagePath      string                     `json:"stage_path"`
	PreimagePath   string                     `json:"preimage_path,omitempty"`
}

type lifecycleTransactionIntentRoot struct {
	RootID         string                       `json:"root_id"`
	Kind           lifecycleTransactionRootKind `json:"kind"`
	RootPath       string                       `json:"root_path"`
	ManifestPath   string                       `json:"manifest_path"`
	ManifestDigest string                       `json:"manifest_digest"`
}

// lifecycleTransactionIntent is immutable after it is made durable. Progress
// lives in progress.json so a crash cannot rewrite the evidence that authorized
// the mutation.
type lifecycleTransactionIntent struct {
	SchemaVersion string                            `json:"schema_version"`
	TransactionID string                            `json:"transaction_id"`
	Record        colony.LifecycleTransactionRecord `json:"record"`
	Roots         []lifecycleTransactionIntentRoot  `json:"roots"`
	CommitOrder   []string                          `json:"commit_order"`
	Digest        string                            `json:"digest"`
}

type lifecycleTransactionProgress struct {
	SchemaVersion    string                            `json:"schema_version"`
	TransactionID    string                            `json:"transaction_id"`
	Stage            colony.LifecycleTransactionStage  `json:"stage"`
	CommittedTargets []string                          `json:"committed_targets,omitempty"`
	CommittedRoots   []string                          `json:"committed_roots,omitempty"`
	StateEffect      colony.LifecycleStateEffect       `json:"state_effect"`
	Receipt          *colony.LifecycleReceiptReference `json:"receipt,omitempty"`
	Digest           string                            `json:"digest"`
}

type lifecycleFileState struct {
	Exists bool
	Digest string
	Mode   os.FileMode
	Bytes  []byte
}

func beginLifecycleTransaction(config lifecycleTransactionConfig) (*lifecycleTransaction, error) {
	if !lifecycleTransactionIDPattern.MatchString(config.TransactionID) {
		return nil, fmt.Errorf("lifecycle transaction: invalid transaction id %q", config.TransactionID)
	}
	if strings.TrimSpace(config.Command) == "" {
		return nil, fmt.Errorf("lifecycle transaction: command is required")
	}
	roots, err := resolveLifecycleTransactionRoots(config.Allowlist)
	if err != nil {
		return nil, err
	}
	if config.Rename == nil {
		config.Rename = os.Rename
	}
	journalPath := filepath.Join(config.Allowlist.LifecycleDataRoot, "transactions", config.TransactionID)
	journalStore, err := storage.NewStore(journalPath)
	if err != nil {
		return nil, fmt.Errorf("lifecycle transaction: initialize coordinator journal: %w", err)
	}
	return &lifecycleTransaction{
		config:       config,
		roots:        roots,
		targets:      make(map[string]struct{}),
		journalStore: journalStore,
	}, nil
}

func resolveLifecycleTransactionRoots(allowlist lifecycleTransactionAllowlist) (map[lifecycleTransactionRootKind]lifecycleResolvedRoot, error) {
	roots := make(map[lifecycleTransactionRootKind]lifecycleResolvedRoot)
	addDirectory := func(kind lifecycleTransactionRootKind, path string, required bool) error {
		if strings.TrimSpace(path) == "" {
			if required {
				return fmt.Errorf("lifecycle transaction: %s root is required", kind)
			}
			return nil
		}
		canonical, err := validateLifecycleDirectoryRoot(kind, path)
		if err != nil {
			return err
		}
		roots[kind] = lifecycleResolvedRoot{Kind: kind, Path: canonical}
		return nil
	}
	if err := addDirectory(lifecycleTransactionRootRepository, allowlist.RepositoryRoot, true); err != nil {
		return nil, err
	}
	if err := addDirectory(lifecycleTransactionRootData, allowlist.LifecycleDataRoot, true); err != nil {
		return nil, err
	}

	if allowlist.Hub.Path != "" || allowlist.Hub.Channel != "" {
		var kind lifecycleTransactionRootKind
		switch allowlist.Hub.Channel {
		case lifecycleTransactionHubStable:
			kind = lifecycleTransactionRootHubStable
		case lifecycleTransactionHubDev:
			kind = lifecycleTransactionRootHubDev
		default:
			return nil, fmt.Errorf("lifecycle transaction: hub channel must be stable or dev")
		}
		if err := addDirectory(kind, allowlist.Hub.Path, true); err != nil {
			return nil, err
		}
	}
	for _, optional := range []struct {
		kind lifecycleTransactionRootKind
		path string
	}{
		{lifecycleTransactionRootClaudeHome, allowlist.ClaudeHome},
		{lifecycleTransactionRootOpenCodeHome, allowlist.OpenCodeHome},
		{lifecycleTransactionRootCodexHome, allowlist.CodexHome},
	} {
		if err := addDirectory(optional.kind, optional.path, false); err != nil {
			return nil, err
		}
	}

	if strings.TrimSpace(allowlist.BinaryDestination) != "" {
		if !filepath.IsAbs(allowlist.BinaryDestination) || filepath.Clean(allowlist.BinaryDestination) != allowlist.BinaryDestination {
			return nil, fmt.Errorf("lifecycle transaction: binary destination must be a canonical absolute path")
		}
		parent, err := validateLifecycleDirectoryRoot(lifecycleTransactionRootBinaryDestination, filepath.Dir(allowlist.BinaryDestination))
		if err != nil {
			return nil, err
		}
		if err := rejectLifecycleSymlinkTarget(parent, allowlist.BinaryDestination); err != nil {
			return nil, err
		}
		roots[lifecycleTransactionRootBinaryDestination] = lifecycleResolvedRoot{
			Kind:        lifecycleTransactionRootBinaryDestination,
			Path:        parent,
			ExactTarget: allowlist.BinaryDestination,
		}
	}

	seen := make(map[string]lifecycleTransactionRootKind)
	for kind, root := range roots {
		key := root.Path
		if previous, ok := seen[key]; ok {
			return nil, fmt.Errorf("lifecycle transaction: duplicate roots %s and %s resolve to %s", previous, kind, key)
		}
		seen[key] = kind
	}
	return roots, nil
}

func validateLifecycleDirectoryRoot(kind lifecycleTransactionRootKind, path string) (string, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return "", fmt.Errorf("lifecycle transaction: %s root must be a canonical absolute path", kind)
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", fmt.Errorf("lifecycle transaction: inspect %s root %q: %w", kind, path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("lifecycle transaction: %s root %q is a symlink", kind, path)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("lifecycle transaction: %s root %q is not a directory", kind, path)
	}
	// Resolve once to prove every existing ancestor is traversable. The final
	// component itself must not be a symlink (checked above), but platform temp
	// and home roots may legitimately sit beneath an OS-managed symlink such as
	// macOS /var -> /private/var.
	if _, err := filepath.EvalSymlinks(path); err != nil {
		return "", fmt.Errorf("lifecycle transaction: resolve %s root %q: %w", kind, path, err)
	}
	return path, nil
}

func (tx *lifecycleTransaction) DeclareWrite(kind lifecycleTransactionRootKind, relativeTarget string, content []byte) error {
	return tx.declare(kind, relativeTarget, lifecycleTransactionWrite, bytes.Clone(content))
}

func (tx *lifecycleTransaction) DeclareRemoval(kind lifecycleTransactionRootKind, relativeTarget string) error {
	return tx.declare(kind, relativeTarget, lifecycleTransactionRemove, nil)
}

func (tx *lifecycleTransaction) declare(kind lifecycleTransactionRootKind, relativeTarget string, action lifecycleTransactionAction, content []byte) error {
	if tx.intent != nil {
		return fmt.Errorf("lifecycle transaction: declarations are closed after intent persistence")
	}
	root, targetPath, cleanRelative, err := tx.resolveTarget(kind, relativeTarget)
	if err != nil {
		return err
	}
	if _, duplicate := tx.targets[targetPath]; duplicate {
		return fmt.Errorf("lifecycle transaction: duplicate target %q", targetPath)
	}
	state, err := readLifecycleFileState(targetPath)
	if err != nil {
		return fmt.Errorf("lifecycle transaction: read baseline for %q: %w", targetPath, err)
	}
	declaration := &lifecycleTransactionDeclaration{
		ID:             fmt.Sprintf("target-%04d", len(tx.declarations)+1),
		Root:           root,
		RelativeTarget: cleanRelative,
		TargetPath:     targetPath,
		Action:         action,
		Content:        content,
		BeforeExists:   state.Exists,
		BeforeDigest:   state.Digest,
		Mode:           state.Mode,
	}
	if declaration.Mode == 0 {
		declaration.Mode = 0o644
	}
	if action == lifecycleTransactionWrite {
		declaration.AfterDigest = lifecycleDigest(content)
	} else {
		declaration.AfterDigest = lifecycleTransactionMissingDigest
	}
	tx.declarations = append(tx.declarations, declaration)
	tx.targets[targetPath] = struct{}{}
	tx.validated = false
	return nil
}

func (tx *lifecycleTransaction) resolveTarget(kind lifecycleTransactionRootKind, relativeTarget string) (lifecycleResolvedRoot, string, string, error) {
	if !kind.valid() {
		return lifecycleResolvedRoot{}, "", "", fmt.Errorf("lifecycle transaction: root kind %q is not allowlisted", kind)
	}
	root, ok := tx.roots[kind]
	if !ok {
		return lifecycleResolvedRoot{}, "", "", fmt.Errorf("lifecycle transaction: root %s was not configured", kind)
	}
	clean := filepath.Clean(relativeTarget)
	if strings.TrimSpace(relativeTarget) == "" || filepath.IsAbs(relativeTarget) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean != relativeTarget {
		return lifecycleResolvedRoot{}, "", "", fmt.Errorf("lifecycle transaction: target %q must be a canonical contained relative path", relativeTarget)
	}
	targetPath := filepath.Join(root.Path, clean)
	if !pathIsWithin(root.Path, targetPath) {
		return lifecycleResolvedRoot{}, "", "", fmt.Errorf("lifecycle transaction: target %q escapes root %q", relativeTarget, root.Path)
	}
	if root.ExactTarget != "" && targetPath != root.ExactTarget {
		return lifecycleResolvedRoot{}, "", "", fmt.Errorf("lifecycle transaction: binary root permits only exact destination %q", root.ExactTarget)
	}
	if pathIsWithin(filepath.Join(root.Path, lifecycleTransactionDirectory), targetPath) {
		return lifecycleResolvedRoot{}, "", "", fmt.Errorf("lifecycle transaction: target %q overlaps transaction evidence", relativeTarget)
	}
	coordinatorRoot := filepath.Join(tx.config.Allowlist.LifecycleDataRoot, "transactions")
	if pathIsWithin(coordinatorRoot, targetPath) {
		return lifecycleResolvedRoot{}, "", "", fmt.Errorf("lifecycle transaction: target %q overlaps coordinator journal", relativeTarget)
	}
	if err := rejectLifecycleSymlinkTarget(root.Path, targetPath); err != nil {
		return lifecycleResolvedRoot{}, "", "", err
	}
	return root, targetPath, clean, nil
}

func rejectLifecycleSymlinkTarget(root, target string) error {
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return fmt.Errorf("lifecycle transaction: target %q escapes root %q", target, root)
	}
	current := root
	parts := strings.Split(relative, string(filepath.Separator))
	for index, part := range parts {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		info, statErr := os.Lstat(current)
		if os.IsNotExist(statErr) {
			return nil
		}
		if statErr != nil {
			return fmt.Errorf("lifecycle transaction: inspect target component %q: %w", current, statErr)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("lifecycle transaction: target %q contains symlink component %q", target, current)
		}
		if index < len(parts)-1 && !info.IsDir() {
			return fmt.Errorf("lifecycle transaction: target ancestor %q is not a directory", current)
		}
		if index == len(parts)-1 && info.IsDir() {
			return fmt.Errorf("lifecycle transaction: target %q is a directory", target)
		}
	}
	return nil
}

func (tx *lifecycleTransaction) Validate() error {
	if len(tx.declarations) == 0 {
		return fmt.Errorf("lifecycle transaction: at least one target is required")
	}
	for _, declaration := range tx.declarations {
		root, targetPath, clean, err := tx.resolveTarget(declaration.Root.Kind, declaration.RelativeTarget)
		if err != nil {
			return err
		}
		if root.Path != declaration.Root.Path || targetPath != declaration.TargetPath || clean != declaration.RelativeTarget {
			return fmt.Errorf("lifecycle transaction: resolved target changed for %q", declaration.RelativeTarget)
		}
		state, err := readLifecycleFileState(declaration.TargetPath)
		if err != nil {
			return fmt.Errorf("lifecycle transaction: verify baseline for %q: %w", declaration.TargetPath, err)
		}
		if state.Exists != declaration.BeforeExists || state.Digest != declaration.BeforeDigest {
			return fmt.Errorf("lifecycle transaction: baseline changed for %q", declaration.TargetPath)
		}
	}
	tx.validated = true
	return nil
}

func (tx *lifecycleTransaction) Commit() (colony.LifecycleReceipt, error) {
	lifecycleTransactionProcessMu.Lock()
	defer lifecycleTransactionProcessMu.Unlock()

	if receipt, ok, err := tx.loadCommittedReceipt(); ok || err != nil {
		return receipt, err
	}
	if err := tx.Validate(); err != nil {
		return colony.LifecycleReceipt{}, err
	}
	if err := tx.callFault("after_validation"); err != nil {
		return colony.LifecycleReceipt{}, err
	}
	if err := tx.stageAndPersistIntent(); err != nil {
		return colony.LifecycleReceipt{}, err
	}
	if err := tx.commitPreparedTargets(); err != nil {
		return colony.LifecycleReceipt{}, err
	}
	receipt, err := tx.verifyAndWriteReceipt()
	if err != nil {
		var faultErr *lifecycleTransactionFaultError
		if errors.As(err, &faultErr) {
			return colony.LifecycleReceipt{}, err
		}
		rollbackErr := tx.rollbackPreparedTargets()
		if rollbackErr != nil {
			return colony.LifecycleReceipt{}, fmt.Errorf("lifecycle transaction: verification failed: %w", errors.Join(err, rollbackErr))
		}
		return colony.LifecycleReceipt{}, fmt.Errorf("lifecycle transaction: verification failed and rolled back: %w", err)
	}
	return receipt, nil
}

func (tx *lifecycleTransaction) stageAndPersistIntent() error {
	rootOrder, grouped := tx.groupDeclarations()
	intent := &lifecycleTransactionIntent{
		SchemaVersion: colony.LifecycleSchemaVersion,
		TransactionID: tx.config.TransactionID,
		Record: colony.LifecycleTransactionRecord{
			SchemaVersion:  colony.LifecycleSchemaVersion,
			TransactionID:  tx.config.TransactionID,
			Command:        tx.config.Command,
			OutcomeKind:    colony.OutcomeKindInProgress,
			Stage:          colony.TransactionStageIntentRecorded,
			Changes:        tx.lifecycleChanges(),
			StateEffect:    colony.LifecycleStateEffectNone,
			Provenance:     colony.RecoveryProvenanceConfirmed,
			BaselineDigest: tx.baselineDigest(),
		},
	}
	for rootIndex, kind := range rootOrder {
		declarations := grouped[kind]
		root := declarations[0].Root
		rootID := fmt.Sprintf("root-%02d-%s", rootIndex+1, kind)
		localDirectory := filepath.Join(root.Path, lifecycleTransactionDirectory, tx.config.TransactionID, rootID)
		localStore, err := storage.NewStore(localDirectory)
		if err != nil {
			return fmt.Errorf("lifecycle transaction: create local staging for %s: %w", kind, err)
		}
		manifest := lifecycleTransactionRootManifest{
			SchemaVersion: colony.LifecycleSchemaVersion,
			TransactionID: tx.config.TransactionID,
			RootID:        rootID,
			Kind:          kind,
			RootPath:      root.Path,
		}
		for _, declaration := range declarations {
			stageRelative := filepath.Join("staged", declaration.ID+".bin")
			stagePath := filepath.Join(localDirectory, stageRelative)
			stageBytes := declaration.Content
			if declaration.Action == lifecycleTransactionRemove {
				stageBytes = []byte("remove\n")
			}
			if err := durableAtomicWrite(localStore, stageRelative, stageBytes); err != nil {
				return fmt.Errorf("lifecycle transaction: stage %q: %w", declaration.TargetPath, err)
			}
			if err := tx.callFault("after_stage:" + declaration.ID); err != nil {
				return err
			}
			preimagePath := ""
			if declaration.BeforeExists {
				state, err := readLifecycleFileState(declaration.TargetPath)
				if err != nil {
					return err
				}
				if state.Digest != declaration.BeforeDigest {
					return fmt.Errorf("lifecycle transaction: baseline changed for %q during staging", declaration.TargetPath)
				}
				preimageRelative := filepath.Join("preimages", declaration.ID+".bin")
				preimagePath = filepath.Join(localDirectory, preimageRelative)
				if err := durableAtomicWrite(localStore, preimageRelative, state.Bytes); err != nil {
					return fmt.Errorf("lifecycle transaction: stage preimage %q: %w", declaration.TargetPath, err)
				}
			}
			manifest.Targets = append(manifest.Targets, lifecycleTransactionTargetManifest{
				ID:             declaration.ID,
				RelativeTarget: declaration.RelativeTarget,
				TargetPath:     declaration.TargetPath,
				Action:         declaration.Action,
				BeforeExists:   declaration.BeforeExists,
				BeforeDigest:   declaration.BeforeDigest,
				AfterDigest:    declaration.AfterDigest,
				Mode:           uint32(declaration.Mode.Perm()),
				StagePath:      stagePath,
				PreimagePath:   preimagePath,
			})
			intent.CommitOrder = append(intent.CommitOrder, declaration.ID)
		}
		manifestPath := filepath.Join(localDirectory, "manifest.json")
		manifestBytes, err := durableSaveJSON(localStore, "manifest.json", manifest)
		if err != nil {
			return fmt.Errorf("lifecycle transaction: persist root manifest %s: %w", rootID, err)
		}
		intent.Roots = append(intent.Roots, lifecycleTransactionIntentRoot{
			RootID:         rootID,
			Kind:           kind,
			RootPath:       root.Path,
			ManifestPath:   manifestPath,
			ManifestDigest: lifecycleDigest(manifestBytes),
		})
	}

	// This second baseline pass closes the staging window. No live target has
	// changed yet, and intent.json does not exist if the check fails.
	if err := tx.Validate(); err != nil {
		return err
	}
	intent.Record.Evidence = lifecycleRootEvidence(intent.Roots)
	intent.Digest = lifecycleIntentDigest(*intent)
	if _, err := durableSaveJSON(tx.journalStore, "intent.json", intent); err != nil {
		return fmt.Errorf("lifecycle transaction: persist coordinator intent: %w", err)
	}
	progress := &lifecycleTransactionProgress{
		SchemaVersion: tx.intentSchemaVersion(intent),
		TransactionID: tx.config.TransactionID,
		Stage:         colony.TransactionStageIntentRecorded,
		StateEffect:   colony.LifecycleStateEffectNone,
	}
	progress.Digest = lifecycleProgressDigest(*progress)
	tx.intent = intent
	tx.progress = progress
	if err := tx.persistProgress(); err != nil {
		return err
	}
	if err := tx.callFault("after_intent"); err != nil {
		return err
	}
	return nil
}

func (tx *lifecycleTransaction) intentSchemaVersion(intent *lifecycleTransactionIntent) string {
	if intent == nil || intent.SchemaVersion == "" {
		return colony.LifecycleSchemaVersion
	}
	return intent.SchemaVersion
}

func (tx *lifecycleTransaction) groupDeclarations() ([]lifecycleTransactionRootKind, map[lifecycleTransactionRootKind][]*lifecycleTransactionDeclaration) {
	order := make([]lifecycleTransactionRootKind, 0)
	grouped := make(map[lifecycleTransactionRootKind][]*lifecycleTransactionDeclaration)
	for _, declaration := range tx.declarations {
		if _, exists := grouped[declaration.Root.Kind]; !exists {
			order = append(order, declaration.Root.Kind)
		}
		grouped[declaration.Root.Kind] = append(grouped[declaration.Root.Kind], declaration)
	}
	return order, grouped
}

func (tx *lifecycleTransaction) commitPreparedTargets() error {
	if tx.intent == nil || tx.progress == nil {
		return fmt.Errorf("lifecycle transaction: no durable intent")
	}
	tx.progress.Stage = colony.TransactionStageCommitting
	if err := tx.persistProgress(); err != nil {
		return err
	}
	manifests, err := loadLifecycleRootManifests(tx.intent)
	if err != nil {
		return tx.requireRecovery(err, colony.RecoveryProvenanceConflicting)
	}
	for _, root := range tx.intent.Roots {
		manifest := manifests[root.RootID]
		for _, target := range manifest.Targets {
			if containsLifecycleString(tx.progress.CommittedTargets, target.ID) {
				continue
			}
			if err := tx.applyTarget(target); err != nil {
				rollbackErr := tx.rollbackPreparedTargets()
				if rollbackErr != nil {
					return fmt.Errorf("lifecycle transaction: commit %s: %w", target.ID, errors.Join(err, rollbackErr))
				}
				return fmt.Errorf("lifecycle transaction: commit %s failed and rolled back: %w", target.ID, err)
			}
			tx.progress.CommittedTargets = append(tx.progress.CommittedTargets, target.ID)
			if err := tx.persistProgress(); err != nil {
				return err
			}
			if err := tx.callFault("after_target_commit:" + target.ID); err != nil {
				return err
			}
		}
		if !containsLifecycleString(tx.progress.CommittedRoots, root.RootID) {
			tx.progress.CommittedRoots = append(tx.progress.CommittedRoots, root.RootID)
			if err := tx.persistProgress(); err != nil {
				return err
			}
		}
		if err := tx.callFault("after_root_commit:" + root.RootID); err != nil {
			return err
		}
	}
	tx.progress.Stage = colony.TransactionStageCommitted
	tx.progress.StateEffect = colony.LifecycleStateEffectCommitted
	return tx.persistProgress()
}

func (tx *lifecycleTransaction) applyTarget(target lifecycleTransactionTargetManifest) error {
	current, err := readLifecycleFileState(target.TargetPath)
	if err != nil {
		return err
	}
	if current.Digest == target.AfterDigest {
		return nil
	}
	if current.Exists != target.BeforeExists || current.Digest != target.BeforeDigest {
		return fmt.Errorf("target %q baseline changed before commit", target.TargetPath)
	}
	switch target.Action {
	case lifecycleTransactionWrite:
		staged, err := readLifecycleEvidenceFile(target.StagePath)
		if err != nil {
			return fmt.Errorf("read staged bytes: %w", err)
		}
		if lifecycleDigest(staged) != target.AfterDigest {
			return fmt.Errorf("staged bytes for %q do not match manifest digest", target.TargetPath)
		}
		if err := atomicReplaceLifecycleTarget(target.TargetPath, staged, os.FileMode(target.Mode), tx.config.Rename); err != nil {
			return err
		}
	case lifecycleTransactionRemove:
		if current.Exists {
			if err := os.Remove(target.TargetPath); err != nil {
				return fmt.Errorf("remove target %q: %w", target.TargetPath, err)
			}
			if err := syncLifecycleDirectory(filepath.Dir(target.TargetPath)); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unknown action %q", target.Action)
	}
	after, err := readLifecycleFileState(target.TargetPath)
	if err != nil {
		return err
	}
	if after.Digest != target.AfterDigest {
		return fmt.Errorf("target %q failed post-commit digest verification", target.TargetPath)
	}
	return nil
}

func (tx *lifecycleTransaction) verifyAndWriteReceipt() (colony.LifecycleReceipt, error) {
	tx.progress.Stage = colony.TransactionStageVerifying
	if err := tx.persistProgress(); err != nil {
		return colony.LifecycleReceipt{}, err
	}
	manifests, err := loadLifecycleRootManifests(tx.intent)
	if err != nil {
		return colony.LifecycleReceipt{}, tx.requireRecovery(err, colony.RecoveryProvenanceConflicting)
	}
	receipt := colony.LifecycleReceipt{
		SchemaVersion: colony.LifecycleSchemaVersion,
		ReceiptID:     tx.config.TransactionID + "-receipt",
		Command:       tx.config.Command,
		OutcomeKind:   colony.OutcomeKindCompleted,
		Changes:       tx.intent.Record.Changes,
		Evidence:      append([]colony.LifecycleEvidence(nil), tx.intent.Record.Evidence...),
		StateEffect:   colony.LifecycleStateEffectCommitted,
		Transaction: colony.LifecycleTransactionReference{
			ID:          tx.config.TransactionID,
			Stage:       colony.TransactionStageVerified,
			JournalPath: tx.journalPath(),
		},
		Provenance: colony.RecoveryProvenanceConfirmed,
	}
	for _, root := range tx.intent.Roots {
		manifest := manifests[root.RootID]
		for _, target := range manifest.Targets {
			state, err := readLifecycleFileState(target.TargetPath)
			if err != nil {
				return colony.LifecycleReceipt{}, err
			}
			if state.Digest != target.AfterDigest {
				return colony.LifecycleReceipt{}, fmt.Errorf("target %q differs during global verification", target.TargetPath)
			}
			evidenceID := target.ID + "-result"
			receipt.Evidence = append(receipt.Evidence, colony.LifecycleEvidence{
				ID:      evidenceID,
				Kind:    "target_digest",
				Source:  target.TargetPath,
				Digest:  target.AfterDigest,
				Summary: fmt.Sprintf("verified %s target %s", root.Kind, target.RelativeTarget),
			})
			receipt.Verification = append(receipt.Verification, colony.LifecycleVerification{
				Name:        "target:" + target.ID,
				Passed:      true,
				EvidenceIDs: []string{evidenceID},
				Detail:      fmt.Sprintf("%s matches declared digest", target.TargetPath),
			})
		}
	}
	rootEvidence := receipt.Evidence[0]
	receipt.Recovery = &colony.LifecycleRecovery{
		Provenance: colony.RecoveryProvenanceConfirmed,
		Facts: []colony.LifecycleRecoveryFact{{
			Name:       "coordinator-journal",
			Summary:    "Coordinator intent and root manifests are retained for deterministic replay.",
			Provenance: colony.RecoveryProvenanceConfirmed,
			Evidence:   []colony.LifecycleEvidence{rootEvidence},
		}},
		Transaction:  &receipt.Transaction,
		SafeNextStep: "If recovery is requested, run `aether resume`; committed replay returns this receipt without another mutation.",
	}
	if err := receipt.Validate(); err != nil {
		return colony.LifecycleReceipt{}, fmt.Errorf("lifecycle transaction: validate receipt: %w", err)
	}
	tx.progress.Stage = colony.TransactionStageVerified
	tx.progress.StateEffect = colony.LifecycleStateEffectCommitted
	if err := tx.persistProgress(); err != nil {
		return colony.LifecycleReceipt{}, err
	}
	if err := tx.callFault("after_global_verification"); err != nil {
		return colony.LifecycleReceipt{}, err
	}
	receiptBytes, err := durableSaveJSON(tx.journalStore, "receipt.json", receipt)
	if err != nil {
		return colony.LifecycleReceipt{}, fmt.Errorf("lifecycle transaction: persist receipt: %w", err)
	}
	receiptDigest := lifecycleDigest(receiptBytes)
	if err := durableAtomicWrite(tx.journalStore, "receipt.sha256", []byte(receiptDigest+"\n")); err != nil {
		return colony.LifecycleReceipt{}, fmt.Errorf("lifecycle transaction: persist receipt digest: %w", err)
	}
	tx.progress.Receipt = &colony.LifecycleReceiptReference{ID: receipt.ReceiptID, Digest: receiptDigest}
	if err := tx.persistProgress(); err != nil {
		return colony.LifecycleReceipt{}, err
	}
	return receipt, nil
}

func (tx *lifecycleTransaction) rollbackPreparedTargets() error {
	if tx.intent == nil || tx.progress == nil {
		return nil
	}
	tx.progress.Stage = colony.TransactionStageRollingBack
	tx.progress.StateEffect = colony.LifecycleStateEffectRecoveryRequired
	if err := tx.persistProgress(); err != nil {
		return err
	}
	manifests, err := loadLifecycleRootManifests(tx.intent)
	if err != nil {
		return tx.requireRecovery(err, colony.RecoveryProvenanceConflicting)
	}
	targets := flattenLifecycleManifestTargets(tx.intent, manifests)
	for index := len(targets) - 1; index >= 0; index-- {
		if err := tx.restoreLifecycleTarget(targets[index]); err != nil {
			return tx.requireRecovery(err, lifecycleRecoveryProvenance(err))
		}
	}
	tx.progress.Stage = colony.TransactionStageRolledBack
	tx.progress.StateEffect = colony.LifecycleStateEffectRolledBack
	tx.progress.CommittedTargets = nil
	tx.progress.CommittedRoots = nil
	return tx.persistProgress()
}

func (tx *lifecycleTransaction) restoreLifecycleTarget(target lifecycleTransactionTargetManifest) error {
	current, err := readLifecycleFileState(target.TargetPath)
	if err != nil {
		return err
	}
	if current.Exists == target.BeforeExists && current.Digest == target.BeforeDigest {
		return nil
	}
	if current.Digest != target.AfterDigest {
		return fmt.Errorf("target %q has conflicting rollback bytes", target.TargetPath)
	}
	if target.BeforeExists {
		preimage, err := readLifecycleEvidenceFile(target.PreimagePath)
		if err != nil {
			return fmt.Errorf("read preimage for %q: %w", target.TargetPath, err)
		}
		if lifecycleDigest(preimage) != target.BeforeDigest {
			return fmt.Errorf("preimage for %q has conflicting digest", target.TargetPath)
		}
		if err := atomicReplaceLifecycleTarget(target.TargetPath, preimage, os.FileMode(target.Mode), tx.config.Rename); err != nil {
			return err
		}
	} else if current.Exists {
		if err := os.Remove(target.TargetPath); err != nil {
			return err
		}
		if err := syncLifecycleDirectory(filepath.Dir(target.TargetPath)); err != nil {
			return err
		}
	}
	restored, err := readLifecycleFileState(target.TargetPath)
	if err != nil {
		return fmt.Errorf("target %q failed rollback read: %w", target.TargetPath, err)
	}
	if restored.Exists != target.BeforeExists || restored.Digest != target.BeforeDigest {
		return fmt.Errorf("target %q failed rollback digest verification", target.TargetPath)
	}
	return nil
}

func (tx *lifecycleTransaction) Rollback() error {
	lifecycleTransactionProcessMu.Lock()
	defer lifecycleTransactionProcessMu.Unlock()
	if tx.intent == nil {
		intent, progress, err := tx.loadJournal()
		if err != nil {
			return err
		}
		tx.intent, tx.progress = intent, progress
	}
	return tx.rollbackPreparedTargets()
}

// resumeLifecycleTransaction reduces only durable evidence. It never guesses
// success from a file's existence: immutable intent and root manifests must
// validate first, and every live target must match either its declared baseline
// or staged result before the reducer commits or rolls back anything.
func resumeLifecycleTransaction(config lifecycleTransactionConfig) (colony.LifecycleReceipt, error) {
	tx, err := beginLifecycleTransaction(config)
	if err != nil {
		return colony.LifecycleReceipt{}, err
	}
	lifecycleTransactionProcessMu.Lock()
	defer lifecycleTransactionProcessMu.Unlock()

	if receipt, ok, receiptErr := tx.loadCommittedReceipt(); ok || receiptErr != nil {
		if receiptErr != nil {
			return tx.recoveryResult(receiptErr, lifecycleRecoveryProvenance(receiptErr))
		}
		return receipt, nil
	}
	intent, progress, err := tx.loadJournal()
	if err != nil {
		return tx.recoveryResult(err, lifecycleRecoveryProvenance(err))
	}
	tx.intent, tx.progress = intent, progress
	manifests, err := tx.validateRecoveryEvidence()
	if err != nil {
		return tx.recoveryResult(err, lifecycleRecoveryProvenance(err))
	}

	switch progress.Stage {
	case colony.TransactionStageIntentRecorded,
		colony.TransactionStageCommitting,
		colony.TransactionStageCommitted,
		colony.TransactionStageVerifying,
		colony.TransactionStageVerified:
		if err := tx.validateResumeTargetStates(manifests); err != nil {
			rollbackErr := tx.rollbackCommittedTargets(manifests)
			if rollbackErr != nil {
				err = errors.Join(err, rollbackErr)
			}
			return tx.recoveryResult(err, lifecycleRecoveryProvenance(err))
		}
		if !allLifecycleTargetsMatch(intent, manifests, true) {
			if err := tx.commitPreparedTargets(); err != nil {
				return tx.recoveryResult(err, lifecycleRecoveryProvenance(err))
			}
		}
		return tx.verifyAndWriteReceipt()
	case colony.TransactionStageRollingBack, colony.TransactionStageRecoveryRequired:
		if err := tx.rollbackPreparedTargets(); err != nil {
			return tx.recoveryResult(err, lifecycleRecoveryProvenance(err))
		}
		return tx.rolledBackResult(), nil
	case colony.TransactionStageRolledBack:
		if !allLifecycleTargetsMatch(intent, manifests, false) {
			err := fmt.Errorf("lifecycle transaction: rolled-back journal does not match baseline bytes")
			return tx.recoveryResult(err, colony.RecoveryProvenanceConflicting)
		}
		return tx.rolledBackResult(), nil
	default:
		err := fmt.Errorf("lifecycle transaction: stage %q cannot be resumed from durable intent", progress.Stage)
		return tx.recoveryResult(err, colony.RecoveryProvenanceConflicting)
	}
}

func (tx *lifecycleTransaction) validateRecoveryEvidence() (map[string]lifecycleTransactionRootManifest, error) {
	if tx.intent == nil || tx.progress == nil {
		return nil, fmt.Errorf("lifecycle transaction: coordinator evidence is incomplete: %w", os.ErrNotExist)
	}
	if err := tx.intent.Record.Validate(); err != nil {
		return nil, fmt.Errorf("lifecycle transaction: invalid intent record: %w", err)
	}
	if tx.intent.Record.TransactionID != tx.config.TransactionID || tx.intent.Record.Command != tx.config.Command || tx.intent.Record.Stage != colony.TransactionStageIntentRecorded {
		return nil, fmt.Errorf("lifecycle transaction: intent record identity or stage conflict")
	}
	if !tx.progress.Stage.Valid() || !tx.progress.StateEffect.Valid() {
		return nil, fmt.Errorf("lifecycle transaction: invalid progress vocabulary")
	}
	manifests, err := loadLifecycleRootManifests(tx.intent)
	if err != nil {
		return nil, err
	}
	knownTargets := make(map[string]struct{})
	knownRoots := make(map[string]struct{})
	absoluteTargets := make(map[string]struct{})
	for _, rootReference := range tx.intent.Roots {
		knownRoots[rootReference.RootID] = struct{}{}
		configuredRoot, ok := tx.roots[rootReference.Kind]
		if !ok || configuredRoot.Path != rootReference.RootPath {
			return nil, fmt.Errorf("lifecycle transaction: root %s conflicts with current allowlist", rootReference.Kind)
		}
		localDirectory := filepath.Join(configuredRoot.Path, lifecycleTransactionDirectory, tx.config.TransactionID, rootReference.RootID)
		if rootReference.ManifestPath != filepath.Join(localDirectory, "manifest.json") {
			return nil, fmt.Errorf("lifecycle transaction: root %s manifest path conflicts with owned staging directory", rootReference.Kind)
		}
		manifest := manifests[rootReference.RootID]
		for _, target := range manifest.Targets {
			if target.ID == "" || target.BeforeDigest == "" || target.AfterDigest == "" {
				return nil, fmt.Errorf("lifecycle transaction: target manifest has missing identity or digest")
			}
			if _, duplicate := knownTargets[target.ID]; duplicate {
				return nil, fmt.Errorf("lifecycle transaction: duplicate target id %s", target.ID)
			}
			knownTargets[target.ID] = struct{}{}
			resolvedRoot, targetPath, relativeTarget, err := tx.resolveTarget(rootReference.Kind, target.RelativeTarget)
			if err != nil {
				return nil, err
			}
			if resolvedRoot.Path != rootReference.RootPath || targetPath != target.TargetPath || relativeTarget != target.RelativeTarget {
				return nil, fmt.Errorf("lifecycle transaction: target %s resolution conflicts with manifest", target.ID)
			}
			if _, duplicate := absoluteTargets[target.TargetPath]; duplicate {
				return nil, fmt.Errorf("lifecycle transaction: duplicate absolute target %q", target.TargetPath)
			}
			absoluteTargets[target.TargetPath] = struct{}{}
			expectedStagePath := filepath.Join(localDirectory, "staged", target.ID+".bin")
			if target.StagePath != expectedStagePath {
				return nil, fmt.Errorf("lifecycle transaction: target %s stage path conflicts with manifest ownership", target.ID)
			}
			staged, err := readLifecycleEvidenceFile(target.StagePath)
			if err != nil {
				return nil, fmt.Errorf("lifecycle transaction: read staged evidence for %s: %w", target.ID, err)
			}
			switch target.Action {
			case lifecycleTransactionWrite:
				if lifecycleDigest(staged) != target.AfterDigest {
					return nil, fmt.Errorf("lifecycle transaction: staged evidence for %s has conflicting digest", target.ID)
				}
			case lifecycleTransactionRemove:
				if target.AfterDigest != lifecycleTransactionMissingDigest || !bytes.Equal(staged, []byte("remove\n")) {
					return nil, fmt.Errorf("lifecycle transaction: removal evidence for %s conflicts with manifest", target.ID)
				}
			default:
				return nil, fmt.Errorf("lifecycle transaction: target %s has unknown action %q", target.ID, target.Action)
			}
			if target.BeforeExists {
				expectedPreimagePath := filepath.Join(localDirectory, "preimages", target.ID+".bin")
				if target.PreimagePath != expectedPreimagePath {
					return nil, fmt.Errorf("lifecycle transaction: target %s preimage path conflicts with manifest ownership", target.ID)
				}
				preimage, err := readLifecycleEvidenceFile(target.PreimagePath)
				if err != nil {
					return nil, fmt.Errorf("lifecycle transaction: read preimage evidence for %s: %w", target.ID, err)
				}
				if lifecycleDigest(preimage) != target.BeforeDigest {
					return nil, fmt.Errorf("lifecycle transaction: preimage evidence for %s has conflicting digest", target.ID)
				}
			} else if target.PreimagePath != "" || target.BeforeDigest != lifecycleTransactionMissingDigest {
				return nil, fmt.Errorf("lifecycle transaction: absent baseline evidence for %s conflicts with manifest", target.ID)
			}
		}
	}
	if len(tx.intent.CommitOrder) != len(knownTargets) {
		return nil, fmt.Errorf("lifecycle transaction: commit order length conflicts with targets")
	}
	seenOrder := make(map[string]struct{})
	for _, targetID := range tx.intent.CommitOrder {
		if _, ok := knownTargets[targetID]; !ok {
			return nil, fmt.Errorf("lifecycle transaction: commit order names unknown target %s", targetID)
		}
		if _, duplicate := seenOrder[targetID]; duplicate {
			return nil, fmt.Errorf("lifecycle transaction: commit order repeats target %s", targetID)
		}
		seenOrder[targetID] = struct{}{}
	}
	if err := validateLifecycleProgressIDs(tx.progress.CommittedTargets, knownTargets, "target"); err != nil {
		return nil, err
	}
	for index, targetID := range tx.progress.CommittedTargets {
		if index >= len(tx.intent.CommitOrder) || tx.intent.CommitOrder[index] != targetID {
			return nil, fmt.Errorf("lifecycle transaction: committed targets are not a prefix of commit order")
		}
	}
	if err := validateLifecycleProgressIDs(tx.progress.CommittedRoots, knownRoots, "root"); err != nil {
		return nil, err
	}
	return manifests, nil
}

func validateLifecycleProgressIDs(values []string, known map[string]struct{}, kind string) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, ok := known[value]; !ok {
			return fmt.Errorf("lifecycle transaction: progress names unknown %s %s", kind, value)
		}
		if _, duplicate := seen[value]; duplicate {
			return fmt.Errorf("lifecycle transaction: progress repeats %s %s", kind, value)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func (tx *lifecycleTransaction) validateResumeTargetStates(manifests map[string]lifecycleTransactionRootManifest) error {
	committed := make(map[string]struct{}, len(tx.progress.CommittedTargets))
	for _, targetID := range tx.progress.CommittedTargets {
		committed[targetID] = struct{}{}
	}
	for _, target := range flattenLifecycleManifestTargets(tx.intent, manifests) {
		state, err := readLifecycleFileState(target.TargetPath)
		if err != nil {
			return fmt.Errorf("lifecycle transaction: inspect recovery target %s: %w", target.ID, err)
		}
		matchesBefore := state.Exists == target.BeforeExists && state.Digest == target.BeforeDigest
		matchesAfter := state.Digest == target.AfterDigest
		if !matchesBefore && !matchesAfter {
			return fmt.Errorf("lifecycle transaction: recovery target %s conflicts with baseline and staged result", target.ID)
		}
		if _, markedCommitted := committed[target.ID]; markedCommitted && !matchesAfter {
			return fmt.Errorf("lifecycle transaction: committed target %s no longer matches staged result", target.ID)
		}
	}
	return nil
}

// rollbackCommittedTargets is used when resume discovers that a remaining
// target is no longer safe to commit. It restores the proven committed prefix
// in reverse order while leaving the unrelated conflicting target untouched.
func (tx *lifecycleTransaction) rollbackCommittedTargets(manifests map[string]lifecycleTransactionRootManifest) error {
	tx.progress.Stage = colony.TransactionStageRollingBack
	tx.progress.StateEffect = colony.LifecycleStateEffectRecoveryRequired
	if err := tx.persistProgress(); err != nil {
		return err
	}
	targets := flattenLifecycleManifestTargets(tx.intent, manifests)
	committedPrefix := len(tx.progress.CommittedTargets)
	if committedPrefix < len(targets) {
		candidate := targets[committedPrefix]
		state, err := readLifecycleFileState(candidate.TargetPath)
		if err == nil && state.Digest == candidate.AfterDigest {
			// A crash can occur after replacement but before its progress write.
			// At most the next target in commit order can be in that state.
			committedPrefix++
		}
	}
	var rollbackErrors []error
	for index := committedPrefix - 1; index >= 0; index-- {
		if err := tx.restoreLifecycleTarget(targets[index]); err != nil {
			rollbackErrors = append(rollbackErrors, err)
		}
	}
	if len(rollbackErrors) > 0 {
		return tx.requireRecovery(errors.Join(rollbackErrors...), colony.RecoveryProvenanceConflicting)
	}
	tx.progress.CommittedTargets = nil
	tx.progress.CommittedRoots = nil
	tx.progress.Stage = colony.TransactionStageRecoveryRequired
	tx.progress.StateEffect = colony.LifecycleStateEffectRecoveryRequired
	return tx.persistProgress()
}

func allLifecycleTargetsMatch(intent *lifecycleTransactionIntent, manifests map[string]lifecycleTransactionRootManifest, after bool) bool {
	for _, target := range flattenLifecycleManifestTargets(intent, manifests) {
		state, err := readLifecycleFileState(target.TargetPath)
		if err != nil {
			return false
		}
		if after {
			if state.Digest != target.AfterDigest {
				return false
			}
		} else if state.Exists != target.BeforeExists || state.Digest != target.BeforeDigest {
			return false
		}
	}
	return true
}

func (tx *lifecycleTransaction) recoveryResult(cause error, provenance colony.RecoveryProvenance) (colony.LifecycleReceipt, error) {
	if provenance != colony.RecoveryProvenanceConflicting && provenance != colony.RecoveryProvenanceUnknown {
		provenance = colony.RecoveryProvenanceUnknown
	}
	evidence := colony.LifecycleEvidence{
		ID:      tx.config.TransactionID + "-recovery-evidence",
		Kind:    "coordinator_journal",
		Source:  tx.journalPath(),
		Summary: cause.Error(),
	}
	transaction := colony.LifecycleTransactionReference{
		ID:          tx.config.TransactionID,
		Stage:       colony.TransactionStageRecoveryRequired,
		JournalPath: tx.journalPath(),
	}
	receipt := colony.LifecycleReceipt{
		SchemaVersion: colony.LifecycleSchemaVersion,
		ReceiptID:     tx.config.TransactionID + "-recovery",
		Command:       tx.config.Command,
		OutcomeKind:   colony.OutcomeKindRecoveryRequired,
		Evidence:      []colony.LifecycleEvidence{evidence},
		StateEffect:   colony.LifecycleStateEffectRecoveryRequired,
		Transaction:   transaction,
		Recovery: &colony.LifecycleRecovery{
			Provenance: provenance,
			Facts: []colony.LifecycleRecoveryFact{{
				Name:       "recovery-evidence",
				Summary:    cause.Error(),
				Provenance: provenance,
				Evidence:   []colony.LifecycleEvidence{evidence},
			}},
			Transaction:  &transaction,
			SafeNextStep: "Preserve the journal and conflicting files, resolve the named evidence, then run `aether resume` again.",
		},
		Provenance: provenance,
	}
	if tx.intent != nil {
		receipt.Changes = append([]colony.LifecycleChange(nil), tx.intent.Record.Changes...)
	}
	if tx.progress != nil && tx.intent != nil {
		tx.progress.Stage = colony.TransactionStageRecoveryRequired
		tx.progress.StateEffect = colony.LifecycleStateEffectRecoveryRequired
		_ = tx.persistProgress()
	}
	if err := receipt.Validate(); err != nil {
		return colony.LifecycleReceipt{}, errors.Join(cause, fmt.Errorf("build recovery receipt: %w", err))
	}
	return receipt, fmt.Errorf("lifecycle transaction: recovery required (%s): %w", provenance, cause)
}

func (tx *lifecycleTransaction) rolledBackResult() colony.LifecycleReceipt {
	evidence := colony.LifecycleEvidence{
		ID:      tx.config.TransactionID + "-rollback-evidence",
		Kind:    "coordinator_journal",
		Source:  tx.journalPath(),
		Summary: "All declared targets match their pre-transaction baselines.",
	}
	transaction := colony.LifecycleTransactionReference{
		ID:          tx.config.TransactionID,
		Stage:       colony.TransactionStageRolledBack,
		JournalPath: tx.journalPath(),
	}
	return colony.LifecycleReceipt{
		SchemaVersion: colony.LifecycleSchemaVersion,
		ReceiptID:     tx.config.TransactionID + "-rolled-back",
		Command:       tx.config.Command,
		OutcomeKind:   colony.OutcomeKindNoChange,
		Changes:       append([]colony.LifecycleChange(nil), tx.intent.Record.Changes...),
		Evidence:      []colony.LifecycleEvidence{evidence},
		StateEffect:   colony.LifecycleStateEffectRolledBack,
		Transaction:   transaction,
		Recovery: &colony.LifecycleRecovery{
			Provenance: colony.RecoveryProvenanceConfirmed,
			Facts: []colony.LifecycleRecoveryFact{{
				Name:       "rollback-complete",
				Summary:    "Every declared target matches its recorded baseline.",
				Provenance: colony.RecoveryProvenanceConfirmed,
				Evidence:   []colony.LifecycleEvidence{evidence},
			}},
			Transaction:  &transaction,
			SafeNextStep: "The transaction is rolled back; rerun the original lifecycle command to start a new transaction.",
		},
		Provenance: colony.RecoveryProvenanceConfirmed,
	}
}

func lifecycleRecoveryProvenance(err error) colony.RecoveryProvenance {
	if errors.Is(err, os.ErrNotExist) {
		return colony.RecoveryProvenanceUnknown
	}
	return colony.RecoveryProvenanceConflicting
}

func (tx *lifecycleTransaction) requireRecovery(cause error, provenance colony.RecoveryProvenance) error {
	if tx.progress != nil {
		tx.progress.Stage = colony.TransactionStageRecoveryRequired
		tx.progress.StateEffect = colony.LifecycleStateEffectRecoveryRequired
		_ = tx.persistProgress()
	}
	return fmt.Errorf("lifecycle transaction: recovery required (%s): %w", provenance, cause)
}

func (tx *lifecycleTransaction) persistProgress() error {
	if tx.progress == nil {
		return fmt.Errorf("lifecycle transaction: progress is not initialized")
	}
	tx.progress.Digest = lifecycleProgressDigest(*tx.progress)
	if _, err := durableSaveJSON(tx.journalStore, "progress.json", tx.progress); err != nil {
		return fmt.Errorf("lifecycle transaction: persist progress: %w", err)
	}
	if tx.intent != nil {
		record := tx.intent.Record
		record.Stage = tx.progress.Stage
		record.StateEffect = tx.progress.StateEffect
		if tx.progress.Receipt != nil {
			reference := *tx.progress.Receipt
			record.Receipt = &reference
		}
		if _, err := durableSaveJSON(tx.journalStore, "record.json", record); err != nil {
			return fmt.Errorf("lifecycle transaction: persist lifecycle record: %w", err)
		}
	}
	return nil
}

func (tx *lifecycleTransaction) loadCommittedReceipt() (colony.LifecycleReceipt, bool, error) {
	receiptPath := filepath.Join(tx.journalPath(), "receipt.json")
	receiptBytes, err := readLifecycleEvidenceFile(receiptPath)
	if os.IsNotExist(err) {
		return colony.LifecycleReceipt{}, false, nil
	}
	if err != nil {
		return colony.LifecycleReceipt{}, false, fmt.Errorf("lifecycle transaction: read receipt: %w", err)
	}
	digestBytes, err := readLifecycleEvidenceFile(filepath.Join(tx.journalPath(), "receipt.sha256"))
	if err != nil {
		return colony.LifecycleReceipt{}, false, fmt.Errorf("lifecycle transaction: receipt evidence is incomplete: %w", err)
	}
	if strings.TrimSpace(string(digestBytes)) != lifecycleDigest(receiptBytes) {
		return colony.LifecycleReceipt{}, false, fmt.Errorf("lifecycle transaction: receipt digest conflicts with receipt bytes")
	}
	var receipt colony.LifecycleReceipt
	if err := decodeLifecycleJSON(receiptBytes, &receipt); err != nil {
		return colony.LifecycleReceipt{}, false, fmt.Errorf("lifecycle transaction: decode receipt: %w", err)
	}
	if err := receipt.Validate(); err != nil {
		return colony.LifecycleReceipt{}, false, fmt.Errorf("lifecycle transaction: invalid receipt: %w", err)
	}
	if receipt.Transaction.ID != tx.config.TransactionID {
		return colony.LifecycleReceipt{}, false, fmt.Errorf("lifecycle transaction: receipt belongs to %q", receipt.Transaction.ID)
	}
	return receipt, true, nil
}

func (tx *lifecycleTransaction) loadJournal() (*lifecycleTransactionIntent, *lifecycleTransactionProgress, error) {
	intentBytes, err := readLifecycleEvidenceFile(filepath.Join(tx.journalPath(), "intent.json"))
	if err != nil {
		return nil, nil, fmt.Errorf("lifecycle transaction: read intent: %w", err)
	}
	var intent lifecycleTransactionIntent
	if err := decodeLifecycleJSON(intentBytes, &intent); err != nil {
		return nil, nil, fmt.Errorf("lifecycle transaction: decode intent: %w", err)
	}
	if intent.Digest != lifecycleIntentDigest(intent) {
		return nil, nil, fmt.Errorf("lifecycle transaction: intent digest conflict")
	}
	if intent.TransactionID != tx.config.TransactionID || intent.Record.Command != tx.config.Command {
		return nil, nil, fmt.Errorf("lifecycle transaction: intent identity conflict")
	}
	progressBytes, err := readLifecycleEvidenceFile(filepath.Join(tx.journalPath(), "progress.json"))
	if err != nil {
		return nil, nil, fmt.Errorf("lifecycle transaction: read progress: %w", err)
	}
	var progress lifecycleTransactionProgress
	if err := decodeLifecycleJSON(progressBytes, &progress); err != nil {
		return nil, nil, fmt.Errorf("lifecycle transaction: decode progress: %w", err)
	}
	if progress.Digest != lifecycleProgressDigest(progress) {
		return nil, nil, fmt.Errorf("lifecycle transaction: progress digest conflict")
	}
	if progress.TransactionID != tx.config.TransactionID {
		return nil, nil, fmt.Errorf("lifecycle transaction: progress identity conflict")
	}
	if err := tx.validateIntentRoots(&intent); err != nil {
		return nil, nil, err
	}
	if _, err := loadLifecycleRootManifests(&intent); err != nil {
		return nil, nil, err
	}
	return &intent, &progress, nil
}

func (tx *lifecycleTransaction) validateIntentRoots(intent *lifecycleTransactionIntent) error {
	for _, journalRoot := range intent.Roots {
		configured, ok := tx.roots[journalRoot.Kind]
		if !ok || configured.Path != journalRoot.RootPath {
			return fmt.Errorf("lifecycle transaction: journal root %s does not match configured allowlist", journalRoot.Kind)
		}
		if !pathIsWithin(configured.Path, journalRoot.ManifestPath) {
			return fmt.Errorf("lifecycle transaction: root manifest for %s escaped its root", journalRoot.Kind)
		}
	}
	return nil
}

func loadLifecycleRootManifests(intent *lifecycleTransactionIntent) (map[string]lifecycleTransactionRootManifest, error) {
	manifests := make(map[string]lifecycleTransactionRootManifest, len(intent.Roots))
	seenTargets := make(map[string]struct{})
	for _, root := range intent.Roots {
		content, err := readLifecycleEvidenceFile(root.ManifestPath)
		if err != nil {
			return nil, fmt.Errorf("read root manifest %s: %w", root.RootID, err)
		}
		if lifecycleDigest(content) != root.ManifestDigest {
			return nil, fmt.Errorf("root manifest %s digest conflict", root.RootID)
		}
		var manifest lifecycleTransactionRootManifest
		if err := decodeLifecycleJSON(content, &manifest); err != nil {
			return nil, fmt.Errorf("decode root manifest %s: %w", root.RootID, err)
		}
		if manifest.SchemaVersion != colony.LifecycleSchemaVersion || manifest.TransactionID != intent.TransactionID || manifest.RootID != root.RootID || manifest.Kind != root.Kind || manifest.RootPath != root.RootPath {
			return nil, fmt.Errorf("root manifest %s identity conflict", root.RootID)
		}
		for _, target := range manifest.Targets {
			if _, exists := seenTargets[target.ID]; exists {
				return nil, fmt.Errorf("duplicate target id %s in root manifests", target.ID)
			}
			seenTargets[target.ID] = struct{}{}
			if !pathIsWithin(root.RootPath, target.TargetPath) || !pathIsWithin(root.RootPath, target.StagePath) || (target.PreimagePath != "" && !pathIsWithin(root.RootPath, target.PreimagePath)) {
				return nil, fmt.Errorf("target %s evidence escaped root %s", target.ID, root.Kind)
			}
		}
		manifests[root.RootID] = manifest
	}
	if len(seenTargets) != len(intent.CommitOrder) {
		return nil, fmt.Errorf("commit order does not cover root manifest targets")
	}
	for _, targetID := range intent.CommitOrder {
		if _, ok := seenTargets[targetID]; !ok {
			return nil, fmt.Errorf("commit order names unknown target %s", targetID)
		}
	}
	return manifests, nil
}

func flattenLifecycleManifestTargets(intent *lifecycleTransactionIntent, manifests map[string]lifecycleTransactionRootManifest) []lifecycleTransactionTargetManifest {
	byID := make(map[string]lifecycleTransactionTargetManifest)
	for _, root := range intent.Roots {
		for _, target := range manifests[root.RootID].Targets {
			byID[target.ID] = target
		}
	}
	result := make([]lifecycleTransactionTargetManifest, 0, len(intent.CommitOrder))
	for _, targetID := range intent.CommitOrder {
		result = append(result, byID[targetID])
	}
	return result
}

func (tx *lifecycleTransaction) lifecycleChanges() []colony.LifecycleChange {
	changes := make([]colony.LifecycleChange, 0, len(tx.declarations))
	for _, declaration := range tx.declarations {
		changes = append(changes, colony.LifecycleChange{
			Target:       string(declaration.Root.Kind) + ":" + filepath.ToSlash(declaration.RelativeTarget),
			Action:       string(declaration.Action),
			BeforeDigest: declaration.BeforeDigest,
			AfterDigest:  declaration.AfterDigest,
		})
	}
	return changes
}

func (tx *lifecycleTransaction) baselineDigest() string {
	parts := make([]string, 0, len(tx.declarations))
	for _, declaration := range tx.declarations {
		parts = append(parts, declaration.ID+":"+declaration.BeforeDigest)
	}
	return lifecycleDigest([]byte(strings.Join(parts, "\n")))
}

func lifecycleRootEvidence(roots []lifecycleTransactionIntentRoot) []colony.LifecycleEvidence {
	evidence := make([]colony.LifecycleEvidence, 0, len(roots))
	for _, root := range roots {
		evidence = append(evidence, colony.LifecycleEvidence{
			ID:      root.RootID + "-manifest",
			Kind:    "root_manifest",
			Source:  root.ManifestPath,
			Digest:  root.ManifestDigest,
			Summary: fmt.Sprintf("validated %s root manifest", root.Kind),
		})
	}
	return evidence
}

func lifecycleIntentDigest(intent lifecycleTransactionIntent) string {
	intent.Digest = ""
	content, _ := json.Marshal(intent)
	return lifecycleDigest(content)
}

func lifecycleProgressDigest(progress lifecycleTransactionProgress) string {
	progress.Digest = ""
	content, _ := json.Marshal(progress)
	return lifecycleDigest(content)
}

func lifecycleDigest(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func readLifecycleFileState(path string) (lifecycleFileState, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return lifecycleFileState{Digest: lifecycleTransactionMissingDigest}, nil
	}
	if err != nil {
		return lifecycleFileState{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return lifecycleFileState{}, fmt.Errorf("%q is a symlink", path)
	}
	if !info.Mode().IsRegular() {
		return lifecycleFileState{}, fmt.Errorf("%q is not a regular file", path)
	}
	content, err := readStableLifecycleRegularFile(path, info)
	if err != nil {
		return lifecycleFileState{}, err
	}
	return lifecycleFileState{Exists: true, Digest: lifecycleDigest(content), Mode: info.Mode().Perm(), Bytes: content}, nil
}

func readLifecycleEvidenceFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("lifecycle transaction: evidence %q is a symlink", path)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("lifecycle transaction: evidence %q is not a regular file", path)
	}
	return readStableLifecycleRegularFile(path, info)
}

func readStableLifecycleRegularFile(path string, before os.FileInfo) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(before, opened) {
		return nil, fmt.Errorf("lifecycle transaction: file %q changed while opening", path)
	}
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	after, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if after.Mode()&os.ModeSymlink != 0 || !os.SameFile(opened, after) {
		return nil, fmt.Errorf("lifecycle transaction: file %q changed while reading", path)
	}
	return content, nil
}

func atomicReplaceLifecycleTarget(target string, content []byte, mode os.FileMode, rename func(string, string) error) error {
	if mode == 0 {
		mode = 0o644
	}
	parent := filepath.Dir(target)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("create target parent %q: %w", parent, err)
	}
	if err := rejectLifecycleSymlinkTarget(parent, target); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(parent, ".lifecycle-transaction-*")
	if err != nil {
		return fmt.Errorf("create same-filesystem temporary file for %q: %w", target, err)
	}
	temporaryPath := temporary.Name()
	success := false
	defer func() {
		_ = temporary.Close()
		if !success {
			_ = os.Remove(temporaryPath)
		}
	}()
	if _, err := temporary.Write(content); err != nil {
		return fmt.Errorf("write temporary file for %q: %w", target, err)
	}
	if err := temporary.Chmod(mode.Perm()); err != nil {
		return fmt.Errorf("chmod temporary file for %q: %w", target, err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync temporary file for %q: %w", target, err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary file for %q: %w", target, err)
	}
	if filepath.Dir(temporaryPath) != parent {
		return fmt.Errorf("temporary file for %q is not on target filesystem", target)
	}
	if err := rename(temporaryPath, target); err != nil {
		return fmt.Errorf("replace target %q: %w", target, err)
	}
	success = true
	if err := syncLifecycleDirectory(parent); err != nil {
		return err
	}
	return nil
}

func durableAtomicWrite(store *storage.Store, path string, content []byte) error {
	if err := store.AtomicWrite(path, content); err != nil {
		return err
	}
	fullPath := path
	if !filepath.IsAbs(path) {
		fullPath = filepath.Join(store.BasePath(), path)
	}
	if err := syncLifecycleFile(fullPath); err != nil {
		return err
	}
	return syncLifecycleDirectory(filepath.Dir(fullPath))
}

func durableSaveJSON(store *storage.Store, path string, value any) ([]byte, error) {
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	content = append(content, '\n')
	if err := store.SaveJSON(path, value); err != nil {
		return nil, err
	}
	fullPath := path
	if !filepath.IsAbs(path) {
		fullPath = filepath.Join(store.BasePath(), path)
	}
	actual, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(actual, content) {
		return nil, fmt.Errorf("durable JSON bytes differ after write to %q", fullPath)
	}
	if err := syncLifecycleFile(fullPath); err != nil {
		return nil, err
	}
	if err := syncLifecycleDirectory(filepath.Dir(fullPath)); err != nil {
		return nil, err
	}
	return actual, nil
}

func syncLifecycleFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %q for sync: %w", path, err)
	}
	defer file.Close()
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync %q: %w", path, err)
	}
	return nil
}

func syncLifecycleDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open directory %q for sync: %w", path, err)
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return fmt.Errorf("sync directory %q: %w", path, err)
	}
	return nil
}

func decodeLifecycleJSON(content []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("multiple JSON values")
		}
		return err
	}
	return nil
}

func containsLifecycleString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func (tx *lifecycleTransaction) journalPath() string {
	return filepath.Join(tx.config.Allowlist.LifecycleDataRoot, "transactions", tx.config.TransactionID)
}

func (tx *lifecycleTransaction) callFault(point string) error {
	if tx.config.Fault == nil {
		return nil
	}
	if err := tx.config.Fault(point); err != nil {
		return &lifecycleTransactionFaultError{point: point, cause: err}
	}
	return nil
}

func pathIsWithin(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}
