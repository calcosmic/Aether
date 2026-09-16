package cmd

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/storage"
)

const codexSkillOwnershipSchema = "codex-skill-ownership/v1"
const codexSkillTargetPrefix = "skills/aether"

// Preserve the Plan 01 fields as aliases while publishing the explicit payload
// version/digest contract. Readers reject disagreeing identities.
type codexSkillOwnedFile = codexSkillPayloadFile
type codexSkillOwnership struct {
	SchemaVersion   string                `json:"schema_version"`
	SourceVersion   string                `json:"source_version"`
	PayloadIdentity string                `json:"payload_identity"`
	PayloadVersion  string                `json:"payload_version,omitempty"`
	PayloadDigest   string                `json:"payload_digest,omitempty"`
	Files           []codexSkillOwnedFile `json:"files"`
}

// Frozen from the original generator AND its guide dependencies at the cited
// revision, not from today's renamed renderer. Bodies make the evidence auditable.
//
//go:embed testdata/codex-skills/legacy-v1.0.79.json
var legacyCodexSkillFixture []byte

func legacyCodexSkillDigests() (map[string]codexSkillOwnedFile, error) {
	var fixture struct {
		SourceRevision string `json:"source_revision"`
		Files          []struct {
			RelativePath string `json:"relative_path"`
			SHA256       string `json:"sha256"`
			Mode         uint32 `json:"mode"`
			Body         string `json:"body"`
		} `json:"files"`
	}
	if err := json.Unmarshal(legacyCodexSkillFixture, &fixture); err != nil {
		return nil, err
	}
	if fixture.SourceRevision != "404731ccffda7bbca64ce801b74b0752a7161315" || len(fixture.Files) != 14 {
		return nil, fmt.Errorf("codex skills: invalid legacy provenance")
	}
	files := map[string]codexSkillOwnedFile{}
	for _, f := range fixture.Files {
		if !validCodexOwnedPath(f.RelativePath) || f.Mode != 0644 || lifecycleDigest([]byte(f.Body)) != f.SHA256 {
			return nil, fmt.Errorf("codex skills: invalid legacy evidence for %s", f.RelativePath)
		}
		if _, duplicate := files[f.RelativePath]; duplicate {
			return nil, fmt.Errorf("codex skills: duplicate legacy evidence")
		}
		files[f.RelativePath] = codexSkillOwnedFile{RelativePath: f.RelativePath, SHA256: f.SHA256, Mode: f.Mode}
	}
	return files, nil
}

func validCodexOwnedPath(relative string) bool {
	if path.Clean(relative) != relative || strings.Contains(relative, "\\") {
		return false
	}
	parts := strings.Split(relative, "/")
	if len(parts) != 2 {
		return false
	}
	if parts[0] == "support" {
		return strings.HasSuffix(parts[1], ".md") && codexSkillIDPattern.MatchString(strings.TrimSuffix(parts[1], ".md"))
	}
	return codexSkillIDPattern.MatchString(parts[0]) && parts[1] == "SKILL.md"
}
func validCodexDigest(digest string) bool {
	if !strings.HasPrefix(digest, "sha256:") || len(digest) != 71 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(digest, "sha256:"))
	return err == nil
}
func readCodexOwnership(state lifecycleFileState) (codexSkillOwnership, error) {
	var owner codexSkillOwnership
	if !state.Exists {
		return owner, nil
	}
	if err := decodeLifecycleJSON(state.Bytes, &owner); err != nil {
		return owner, fmt.Errorf("codex skills: ownership collision: %w", err)
	}
	if owner.SchemaVersion != codexSkillOwnershipSchema || !codexSkillVersionPattern.MatchString(owner.SourceVersion) || !validCodexDigest(owner.PayloadIdentity) || len(owner.Files) == 0 {
		return owner, fmt.Errorf("codex skills: malformed ownership record")
	}
	if (owner.PayloadVersion != "" && owner.PayloadVersion != owner.SourceVersion) || (owner.PayloadDigest != "" && owner.PayloadDigest != owner.PayloadIdentity) {
		return owner, fmt.Errorf("codex skills: conflicting ownership payload identity")
	}
	seen := map[string]bool{}
	for _, f := range owner.Files {
		if !validCodexOwnedPath(f.RelativePath) || seen[f.RelativePath] || !validCodexDigest(f.SHA256) || f.Mode == 0 || f.Mode & ^uint32(0777) != 0 {
			return owner, fmt.Errorf("codex skills: invalid ownership file %q", f.RelativePath)
		}
		seen[f.RelativePath] = true
	}
	return owner, nil
}

// Classify from exactly one captured read. Neither a shipped marker nor a
// familiar name grants permission; absence is an explicit baseline too.
func classifyCodexSkillTarget(state lifecycleFileState, prior codexSkillOwnedFile, proven bool) error {
	if !state.Exists {
		return nil
	}
	if !proven || state.Digest != prior.SHA256 || uint32(state.Mode.Perm()) != prior.Mode {
		return fmt.Errorf("unowned or modified collision")
	}
	return nil
}
func codexSkillTarget(relative string, state lifecycleFileState) maintenanceMutationTarget {
	mode := state.Mode.Perm()
	return maintenanceMutationTarget{Root: lifecycleTransactionRootCodexHome, RelativeTarget: filepath.FromSlash(codexSkillTargetPrefix + "/" + relative), Label: "Skills (codex shims)", ExpectedDigest: state.Digest, ExpectedMode: &mode, Managed: true}
}

// Assemble locally and append only after every ownership decision succeeds.
// Every targeted path, including absent retirements and unchanged desired files,
// carries the original existence/digest/mode observation into common commit.
func planCodexSkillTargets(plan *maintenanceMutationPlan, payload codexSkillPayload) error {
	if err := validateCodexSkillPayload(payload); err != nil {
		return err
	}
	tx, err := beginLifecycleTransaction(lifecycleTransactionConfig{TransactionID: plan.TransactionID, Command: plan.Operation, Allowlist: plan.Allowlist})
	if err != nil {
		return err
	}
	capture := func(relative string) (lifecycleFileState, error) {
		_, dest, _, err := tx.resolveTarget(lifecycleTransactionRootCodexHome, filepath.FromSlash(codexSkillTargetPrefix+"/"+relative))
		if err != nil {
			return lifecycleFileState{}, err
		}
		return readLifecycleFileState(dest)
	}
	ownerState, err := capture(".aether-owned.json")
	if err != nil {
		return err
	}
	previous, err := readCodexOwnership(ownerState)
	if err != nil {
		return err
	}
	if previous.SourceVersion != "" && compareVersions(payload.SourceVersion, previous.SourceVersion) < 0 {
		return fmt.Errorf("codex skills: refusing payload downgrade from %s to %s", previous.SourceVersion, payload.SourceVersion)
	}
	owned := map[string]codexSkillOwnedFile{}
	for _, f := range previous.Files {
		owned[f.RelativePath] = f
	}
	legacy, err := legacyCodexSkillDigests()
	if err != nil {
		return err
	}
	var targets []maintenanceMutationTarget
	var preserved []string
	files := append([]codexSkillOwnedFile(nil), payload.Files...)
	for _, file := range payload.Files {
		state, err := capture(file.RelativePath)
		if err != nil {
			return err
		}
		prior, proven := owned[file.RelativePath]
		if err := classifyCodexSkillTarget(state, prior, proven); err != nil {
			return fmt.Errorf("codex skills: collision at %s: %w", file.RelativePath, err)
		}
		target := codexSkillTarget(file.RelativePath, state)
		target.Source = "payload " + codexSkillPayloadIdentity(payload)
		target.Content = append([]byte(nil), file.Content...)
		target.Mode = os.FileMode(file.Mode)
		targets = append(targets, target)
		delete(owned, file.RelativePath)
	}
	names := make([]string, 0, len(legacy))
	for name := range legacy {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, relative := range names {
		state, err := capture(relative)
		if err != nil {
			preserved = append(preserved, relative+" (unproven legacy: "+err.Error()+")")
			continue
		}
		proof, known := owned[relative]
		if !known {
			proof = legacy[relative]
		}
		if err := classifyCodexSkillTarget(state, proof, true); err != nil {
			preserved = append(preserved, relative+" (edited or unknown legacy)")
			continue
		}
		target := codexSkillTarget(relative, state)
		target.Source = "explicit legacy retirement at 404731ccffda7bbca64ce801b74b0752a7161315"
		target.Action = lifecycleTransactionRemove
		targets = append(targets, target)
		delete(owned, relative)
	}
	// Report unfamiliar discoverable files without treating them as owned. The
	// directory walk does not follow links and never schedules directory removal.
	desired := map[string]bool{}
	for _, file := range payload.Files {
		desired[file.RelativePath] = true
	}
	root := filepath.Join(plan.Allowlist.CodexHome, filepath.FromSlash(codexSkillTargetPrefix))
	if err := filepath.WalkDir(root, func(filename string, entry os.DirEntry, walkErr error) error {
		if os.IsNotExist(walkErr) {
			return nil
		}
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || entry.Name() != "SKILL.md" {
			return nil
		}
		relative, err := filepath.Rel(root, filename)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if _, known := legacy[relative]; !known && !desired[relative] {
			if _, recorded := owned[relative]; !recorded {
				preserved = append(preserved, relative+" (unowned entry)")
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("codex skills: inspect preserved entries: %w", err)
	}
	// Absence from today's desired inventory is never permission to delete a
	// future/custom entry or to silently drop its existing ownership evidence.
	for _, file := range owned {
		files = append(files, file)
		preserved = append(preserved, file.RelativePath+" (outside desired inventory)")
	}
	sort.Slice(files, func(i, j int) bool { return files[i].RelativePath < files[j].RelativePath })
	sort.Strings(preserved)
	identity := codexSkillPayloadIdentity(payload)
	owner := codexSkillOwnership{SchemaVersion: codexSkillOwnershipSchema, SourceVersion: payload.SourceVersion, PayloadIdentity: identity, PayloadVersion: payload.SourceVersion, PayloadDigest: identity, Files: files}
	raw, err := json.MarshalIndent(owner, "", "  ")
	if err != nil {
		return err
	}
	target := codexSkillTarget(".aether-owned.json", ownerState)
	target.Source = "validated payload ownership"
	target.Content = append(raw, '\n')
	target.Mode = 0644
	targets = append(targets, target)
	plan.Targets = append(plan.Targets, targets...)
	plan.PreservedCodexSkills = append(plan.PreservedCodexSkills, preserved...)
	return nil
}

func isCodexSkillTarget(target maintenanceMutationTarget) bool {
	relative := filepath.ToSlash(filepath.Clean(target.RelativeTarget))
	return target.Root == lifecycleTransactionRootCodexHome && (relative == codexSkillTargetPrefix || strings.HasPrefix(relative, codexSkillTargetPrefix+"/"))
}

// One identity per physical Codex home, independent of the project/coordinator.
// A new locker per commit also serializes independent goroutines in this process.
// Preview never calls this function and therefore creates no lock artifacts.
func lockCodexSkillTargets(plan maintenanceMutationPlan) (func() error, error) {
	needed := false
	for _, target := range plan.Targets {
		if isCodexSkillTarget(target) {
			needed = true
			break
		}
	}
	if !needed {
		return func() error { return nil }, nil
	}
	if _, err := validateLifecycleDirectoryRoot(lifecycleTransactionRootCodexHome, plan.Allowlist.CodexHome); err != nil {
		return nil, err
	}
	root, err := filepath.EvalSymlinks(plan.Allowlist.CodexHome)
	if err != nil {
		return nil, err
	}
	identity := filepath.Join(root, filepath.FromSlash(codexSkillTargetPrefix), ".aether-owned.json")
	locks := filepath.Join(root, ".aether-skill-locks")
	// FileLocker names are the digest of the normalized complete identity. Check
	// the exact file as well as its ancestors, never follow a pre-existing link.
	normalized := filepath.ToSlash(identity)
	if runtime.GOOS == "windows" {
		normalized = strings.ToLower(normalized)
	}
	sum := sha256.Sum256([]byte(normalized))
	lockPath := filepath.Join(locks, hex.EncodeToString(sum[:])+".lock")
	if err := rejectLifecycleSymlinkTarget(root, lockPath); err != nil {
		return nil, err
	}
	locker, err := storage.NewFileLocker(locks)
	if err != nil {
		return nil, err
	}
	if err := rejectLifecycleSymlinkTarget(root, lockPath); err != nil {
		return nil, err
	}
	if err := locker.Lock(identity); err != nil {
		return nil, err
	}
	return func() error { return locker.Unlock(identity) }, nil
}

var maintenanceCodexSkillLocker = lockCodexSkillTargets
