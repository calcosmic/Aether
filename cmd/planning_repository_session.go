package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/calcosmic/Aether/pkg/storage"
)

const planningMutationSessionLockTarget = ".planning-repository-mutation-session"

var errPlanningMutationSessionRelease = errors.New("planning mutation session release")

// planningMutationSession owns one repository-root lock from the first
// authoritative read until its callback returns. Baselines are immutable
// within the session: transactions may consume them, but never recapture them.
type planningMutationSession struct {
	operation      string
	repositoryRoot string
	dataRoot       string
	authority      *storage.RepositoryRoot
	store          *storage.Store
	baselines      map[string]lifecycleFileState
	active         bool
}

// withPlanningMutationSession opens Plan 27's physical repository authority,
// acquires one cross-process root lock, and holds it for the callback's full
// read/derive/commit/rollback lifetime. Returning (or panicking) is the only
// way the callback can release the lock.
func withPlanningMutationSession(root, operation string, callback func(*planningMutationSession) error) error {
	if strings.TrimSpace(operation) == "" {
		return fmt.Errorf("planning mutation session: operation is required")
	}
	if callback == nil {
		return fmt.Errorf("planning mutation session: callback is required")
	}
	repositoryRoot, err := filepath.Abs(strings.TrimSpace(root))
	if err != nil {
		return fmt.Errorf("planning mutation session: resolve repository root: %w", err)
	}
	repositoryRoot = filepath.Clean(repositoryRoot)
	dataRoot := filepath.Join(repositoryRoot, ".aether", "data")
	authority, err := storage.OpenRepositoryRoot(repositoryRoot, dataRoot)
	if err != nil {
		return fmt.Errorf("planning mutation session: %w", err)
	}
	defer authority.Close()
	store, err := storage.NewRepositoryStore(authority)
	if err != nil {
		return fmt.Errorf("planning mutation session: open repository store: %w", err)
	}
	lockTargetExists, err := store.FileExists(planningMutationSessionLockTarget)
	if err != nil {
		return fmt.Errorf("planning mutation session: inspect root lock target: %w", err)
	}
	if !lockTargetExists {
		if err := store.AtomicWrite(planningMutationSessionLockTarget, []byte("planning-repository-mutation-session/v1\n")); errors.Is(err, os.ErrNotExist) {
			// Two first-use processes can both observe a missing locks directory.
			// Reopening through the same authority makes its idempotent creation
			// visible without introducing a path-based mkdir fallback.
			store, err = storage.NewRepositoryStore(authority)
			if err == nil {
				err = store.AtomicWrite(planningMutationSessionLockTarget, []byte("planning-repository-mutation-session/v1\n"))
			}
			if err != nil {
				return fmt.Errorf("planning mutation session: initialize root lock target: %w", err)
			}
		} else if err != nil {
			return fmt.Errorf("planning mutation session: initialize root lock target: %w", err)
		}
	}
	session := &planningMutationSession{
		operation:      strings.TrimSpace(operation),
		repositoryRoot: repositoryRoot,
		dataRoot:       store.BasePath(),
		authority:      authority,
		store:          store,
		baselines:      make(map[string]lifecycleFileState),
	}
	var callbackErr error
	err = store.UpdateFile(planningMutationSessionLockTarget, func([]byte) ([]byte, error) {
		session.active = true
		defer func() { session.active = false }()
		callbackErr = callback(session)
		// UpdateFile deliberately receives an error so the synthetic lock target
		// is never written. Only its repository-authorized lock file persists.
		return nil, errPlanningMutationSessionRelease
	})
	if !errors.Is(err, errPlanningMutationSessionRelease) {
		return fmt.Errorf("planning mutation session: hold root lock: %w", err)
	}
	return callbackErr
}

func (session *planningMutationSession) RepositoryRoot() string {
	if session == nil {
		return ""
	}
	return session.repositoryRoot
}

func (session *planningMutationSession) DataRoot() string {
	if session == nil {
		return ""
	}
	return session.dataRoot
}

// ReadFile captures one exact SHA-256-or-absent baseline beneath the held root
// lock. Repeated reads return the original bytes so one derivation cannot mix
// authority from two different repository moments.
func (session *planningMutationSession) ReadFile(kind lifecycleTransactionRootKind, relativeTarget string) ([]byte, bool, error) {
	if err := session.requireActive(); err != nil {
		return nil, false, err
	}
	key, clean, err := session.baselineKey(kind, relativeTarget)
	if err != nil {
		return nil, false, err
	}
	if baseline, ok := session.baselines[key]; ok {
		return bytes.Clone(baseline.Bytes), baseline.Exists, nil
	}
	state, err := session.readCurrentFileState(kind, clean)
	if err != nil {
		return nil, false, err
	}
	session.baselines[key] = cloneLifecycleFileState(state)
	return bytes.Clone(state.Bytes), state.Exists, nil
}

// LoadJSON captures the same immutable baseline as ReadFile and decodes it
// with the lifecycle transaction's strict single-value JSON rules.
func (session *planningMutationSession) LoadJSON(kind lifecycleTransactionRootKind, relativeTarget string, destination any) (bool, error) {
	content, exists, err := session.ReadFile(kind, relativeTarget)
	if err != nil || !exists {
		return exists, err
	}
	if err := decodeLifecycleJSON(content, destination); err != nil {
		return false, fmt.Errorf("planning mutation session: decode %s:%s: %w", kind, relativeTarget, err)
	}
	return true, nil
}

func (session *planningMutationSession) lifecycleRoots(allowlist lifecycleTransactionAllowlist) (map[lifecycleTransactionRootKind]lifecycleResolvedRoot, error) {
	if err := session.requireActive(); err != nil {
		return nil, err
	}
	if allowlist.Hub.Path != "" || allowlist.Hub.Channel != "" || allowlist.ClaudeHome != "" || allowlist.OpenCodeHome != "" || allowlist.CodexHome != "" || allowlist.BinaryDestination != "" {
		return nil, fmt.Errorf("planning mutation session: only repository and lifecycle data roots are supported")
	}
	if filepath.Clean(allowlist.RepositoryRoot) != session.repositoryRoot {
		return nil, fmt.Errorf("planning mutation session: repository root %q does not match held authority %q", allowlist.RepositoryRoot, session.repositoryRoot)
	}
	if filepath.Clean(allowlist.LifecycleDataRoot) != session.dataRoot {
		return nil, fmt.Errorf("planning mutation session: lifecycle data root %q does not match held authority %q", allowlist.LifecycleDataRoot, session.dataRoot)
	}
	return map[lifecycleTransactionRootKind]lifecycleResolvedRoot{
		lifecycleTransactionRootRepository: {Kind: lifecycleTransactionRootRepository, Path: session.repositoryRoot},
		lifecycleTransactionRootData:       {Kind: lifecycleTransactionRootData, Path: session.dataRoot},
	}, nil
}

func (session *planningMutationSession) capturedBaseline(kind lifecycleTransactionRootKind, relativeTarget string) (lifecycleFileState, error) {
	if err := session.requireActive(); err != nil {
		return lifecycleFileState{}, err
	}
	key, _, err := session.baselineKey(kind, relativeTarget)
	if err != nil {
		return lifecycleFileState{}, err
	}
	baseline, ok := session.baselines[key]
	if !ok {
		return lifecycleFileState{}, fmt.Errorf("planning mutation session: target %s:%s has no captured session baseline", kind, relativeTarget)
	}
	return cloneLifecycleFileState(baseline), nil
}

func (session *planningMutationSession) currentFileState(kind lifecycleTransactionRootKind, relativeTarget string) (lifecycleFileState, error) {
	if err := session.requireActive(); err != nil {
		return lifecycleFileState{}, err
	}
	_, clean, err := session.baselineKey(kind, relativeTarget)
	if err != nil {
		return lifecycleFileState{}, err
	}
	return session.readCurrentFileState(kind, clean)
}

func (session *planningMutationSession) updateBaseline(kind lifecycleTransactionRootKind, relativeTarget string, state lifecycleFileState) error {
	if err := session.requireActive(); err != nil {
		return err
	}
	key, _, err := session.baselineKey(kind, relativeTarget)
	if err != nil {
		return err
	}
	session.baselines[key] = cloneLifecycleFileState(state)
	return nil
}

func (session *planningMutationSession) readCurrentFileState(kind lifecycleTransactionRootKind, relativeTarget string) (lifecycleFileState, error) {
	switch kind {
	case lifecycleTransactionRootData:
		content, err := session.store.ReadFile(relativeTarget)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return lifecycleFileState{Digest: lifecycleTransactionMissingDigest}, nil
			}
			return lifecycleFileState{}, fmt.Errorf("planning mutation session: read lifecycle data target %q: %w", relativeTarget, err)
		}
		info, err := os.Lstat(filepath.Join(session.dataRoot, filepath.FromSlash(relativeTarget)))
		if err != nil {
			return lifecycleFileState{}, fmt.Errorf("planning mutation session: inspect lifecycle data target %q: %w", relativeTarget, err)
		}
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return lifecycleFileState{}, fmt.Errorf("planning mutation session: lifecycle data target %q is not a regular non-symlink file", relativeTarget)
		}
		return lifecycleFileState{Exists: true, Digest: lifecycleDigest(content), Mode: info.Mode().Perm(), Bytes: bytes.Clone(content)}, nil
	case lifecycleTransactionRootRepository:
		return readLifecycleFileState(filepath.Join(session.repositoryRoot, filepath.FromSlash(relativeTarget)))
	default:
		return lifecycleFileState{}, fmt.Errorf("planning mutation session: root %s is not held by this session", kind)
	}
}

func (session *planningMutationSession) baselineKey(kind lifecycleTransactionRootKind, relativeTarget string) (string, string, error) {
	if kind != lifecycleTransactionRootRepository && kind != lifecycleTransactionRootData {
		return "", "", fmt.Errorf("planning mutation session: root %s is not held by this session", kind)
	}
	clean := filepath.Clean(relativeTarget)
	if strings.TrimSpace(relativeTarget) == "" || filepath.IsAbs(relativeTarget) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean != relativeTarget {
		return "", "", fmt.Errorf("planning mutation session: target %q must be a canonical contained relative path", relativeTarget)
	}
	return string(kind) + ":" + filepath.ToSlash(clean), clean, nil
}

func (session *planningMutationSession) requireActive() error {
	if session == nil || !session.active || session.authority == nil || session.store == nil {
		return fmt.Errorf("planning mutation session: session is not active")
	}
	return nil
}

func cloneLifecycleFileState(state lifecycleFileState) lifecycleFileState {
	state.Bytes = bytes.Clone(state.Bytes)
	return state
}
