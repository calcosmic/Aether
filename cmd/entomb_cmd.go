package cmd

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/exchange"
	"github.com/calcosmic/Aether/pkg/storage"
	"github.com/spf13/cobra"
)

var entombCmd = &cobra.Command{
	Use:   "entomb",
	Short: "Archive and clear the sealed colony.",
	Args:  cobra.NoArgs,
	RunE:  runEntomb,
}

const entombConfirmationQuestion = "Archive and clear this sealed colony after archive verification? [y/N]"

var entombOrderedStages = []string{
	"Stage archive",
	"Write digest manifest",
	"Verify bytes and cross-references",
	"Publish chamber and tombstone",
	"Clear active state",
}

type entombTransactionFaultHook func(point string) error

type entombTransactionInput struct {
	Root      string
	DataRoot  string
	Confirmed bool
	Now       time.Time
	Fault     entombTransactionFaultHook
	Rename    func(oldPath, newPath string) error
}

type entombTransactionResult struct {
	AwaitingConfirmation bool
	Confirmation         string
	ArchiveID            string
	Goal                 string
	Scope                colony.ColonyScope
	ChamberName          string
	ChamberPath          string
	ManifestDigest       string
	Receipt              colony.LifecycleReceipt
	Disposition          colony.SealDisposition
	OwnerReason          string
	Retained             []string
	Stages               []string
	Next                 string
	Replay               bool
}

type entombFailureError struct {
	Stage         string
	TransactionID string
	StateEffect   colony.LifecycleStateEffect
	cause         error
}

func (err *entombFailureError) Error() string {
	next := "retry `/ant-entomb --confirm` after inspecting the named source"
	if err.TransactionID != "" {
		next = fmt.Sprintf("inspect `.aether/data/transactions/%s/` and retry `/ant-entomb --confirm`", err.TransactionID)
	}
	return fmt.Sprintf("entomb %s failed: %v. Active colony state was retained; %s", err.Stage, err.cause, next)
}

func (err *entombFailureError) Unwrap() error { return err.cause }

type entombPreparedSource struct {
	Manifest entombArchiveSource
	Actual   string
	Content  []byte
	Clear    bool
}

type entombPreflight struct {
	Root        string
	DataRoot    string
	State       colony.ColonyState
	Outcome     colony.SealOutcome
	ArchiveID   string
	Transaction string
	ChamberName string
	ChamberPath string
	Goal        string
	Scope       colony.ColonyScope
	Sources     []entombPreparedSource
}

func entombConfirmationCopy() string {
	return entombConfirmationQuestion
}

func runEntomb(cmd *cobra.Command, args []string) error {
	if store == nil {
		outputErrorMessage("no store initialized")
		return nil
	}
	root := resolveAetherRootPath()
	confirmed, _ := cmd.Flags().GetBool("confirm")
	result, err := runEntombTransaction(entombTransactionInput{
		Root: root, DataRoot: store.BasePath(), Confirmed: confirmed, Now: time.Now().UTC(),
	})
	if err != nil {
		state := colony.ColonyState{}
		if _, _, loaded, loadErr := loadEntombTransactionState(root, store.BasePath()); loadErr == nil {
			state = loaded
		}
		failureResult, closeoutErr := entombFailureResultForState(state, err)
		if closeoutErr != nil {
			outputError(2, err.Error(), map[string]interface{}{"active_state_retained": true, "retry": `/ant-entomb --confirm`, "closeout_error": closeoutErr.Error()})
			return nil
		}
		if shouldRenderVisualOutput(stderr) {
			markRenderedCommandError(2)
			writeVisualOutput(stderr, renderEntombFailureVisual(err, failureResult))
			return nil
		}
		outputError(2, err.Error(), failureResult)
		return nil
	}
	if result.AwaitingConfirmation {
		payload := entombResultMap(result)
		outputWorkflow(payload, renderEntombConfirmationVisual(payload))
		return nil
	}
	emitLifecycleCeremony(events.CeremonyTopicChamberEntomb, events.CeremonyPayload{
		TaskID: result.ArchiveID, Task: result.ChamberName, Status: "entombed", Message: result.Goal,
	}, "aether-entomb")
	payload := entombResultMap(result)
	outputWorkflow(payload, renderEntombVisual(payload))
	return nil
}

func runEntombTransaction(input entombTransactionInput) (entombTransactionResult, error) {
	root, dataRoot, state, err := loadEntombTransactionState(input.Root, input.DataRoot)
	if err != nil {
		return entombTransactionResult{}, entombRetainedError("preflight", "", err)
	}
	input.Root, input.DataRoot = root, dataRoot
	if input.Now.IsZero() {
		input.Now = time.Now().UTC()
	}
	if replay, ok, replayErr := replayEntombTransaction(input, state); ok {
		return replay, replayErr
	}

	preflight, err := prepareEntombPreflight(input, state)
	if err != nil {
		return entombTransactionResult{}, entombRetainedError("preflight", "", err)
	}
	preview := entombTransactionResult{
		AwaitingConfirmation: !input.Confirmed,
		Confirmation:         entombConfirmationCopy(),
		ArchiveID:            preflight.ArchiveID,
		Goal:                 preflight.Goal,
		Scope:                preflight.Scope,
		ChamberName:          preflight.ChamberName,
		ChamberPath:          preflight.ChamberPath,
		Disposition:          preflight.Outcome.Disposition,
		OwnerReason:          preflight.Outcome.OwnerReason,
		Retained:             []string{"colony memory", "seal evidence", "tombstone input"},
		Next:                 `/ant-entomb --confirm`,
	}
	if !input.Confirmed {
		return preview, nil
	}

	if _, err := os.Lstat(preflight.ChamberPath); err == nil {
		return entombTransactionResult{}, entombRetainedError("publish", preflight.Transaction, fmt.Errorf("chamber destination %q already exists without a matching verified receipt", preflight.ChamberPath))
	} else if !os.IsNotExist(err) {
		return entombTransactionResult{}, entombRetainedError("publish", preflight.Transaction, fmt.Errorf("inspect chamber destination: %w", err))
	}

	stagingRoot, err := os.MkdirTemp("", "aether-entomb-stage-")
	if err != nil {
		return entombTransactionResult{}, entombRetainedError("stage", preflight.Transaction, err)
	}
	defer os.RemoveAll(stagingRoot)
	sourceStage := filepath.Join(stagingRoot, "source")
	archiveStage := filepath.Join(stagingRoot, "archive")
	if err := os.MkdirAll(sourceStage, 0o700); err != nil {
		return entombTransactionResult{}, entombRetainedError("stage", preflight.Transaction, err)
	}
	if err := os.MkdirAll(archiveStage, 0o700); err != nil {
		return entombTransactionResult{}, entombRetainedError("stage", preflight.Transaction, err)
	}

	stages := make([]string, 0, len(entombOrderedStages))
	stages = append(stages, entombOrderedStages[0])
	for _, source := range preflight.Sources {
		if err := writeEntombStagedFile(sourceStage, source.Manifest.SourcePath, source.Content); err != nil {
			return entombTransactionResult{}, entombRetainedError("stage", preflight.Transaction, err)
		}
		if err := writeEntombStagedFile(archiveStage, source.Manifest.ArchivePath, source.Content); err != nil {
			return entombTransactionResult{}, entombRetainedError("stage", preflight.Transaction, err)
		}
	}
	if err := callEntombFault(input.Fault, entombOrderedStages[0]); err != nil {
		return entombTransactionResult{}, entombRetainedError("stage", preflight.Transaction, err)
	}

	stages = append(stages, entombOrderedStages[1])
	manifestSources := make([]entombArchiveSource, 0, len(preflight.Sources))
	for _, source := range preflight.Sources {
		manifestSources = append(manifestSources, source.Manifest)
	}
	manifestInput := entombArchiveManifestInput{
		SourceRoot: sourceStage, ArchiveRoot: archiveStage,
		ArchiveID: preflight.ArchiveID, TransactionID: preflight.Transaction,
		ProjectionRevision: LifecycleProjectionRevision, SealOutcome: preflight.Outcome,
		Sources: manifestSources,
	}
	manifest, manifestBytes, err := buildEntombArchiveManifest(manifestInput)
	if err != nil {
		return entombTransactionResult{}, entombRetainedError("manifest", preflight.Transaction, err)
	}
	if err := writeEntombStagedFile(archiveStage, "manifest.json", manifestBytes); err != nil {
		return entombTransactionResult{}, entombRetainedError("manifest", preflight.Transaction, err)
	}
	if err := callEntombFault(input.Fault, entombOrderedStages[1]); err != nil {
		return entombTransactionResult{}, entombRetainedError("manifest", preflight.Transaction, err)
	}

	stages = append(stages, entombOrderedStages[2])
	if err := verifyEntombArchiveManifest(manifestInput, manifest); err != nil {
		return entombTransactionResult{}, entombRetainedError("verification", preflight.Transaction, err)
	}
	if err := verifyEntombLiveSources(preflight); err != nil {
		return entombTransactionResult{}, entombRetainedError("verification", preflight.Transaction, err)
	}
	if err := callEntombFault(input.Fault, entombOrderedStages[2]); err != nil {
		return entombTransactionResult{}, entombRetainedError("verification", preflight.Transaction, err)
	}

	stages = append(stages, entombOrderedStages[3])
	if err := callEntombFault(input.Fault, entombOrderedStages[3]); err != nil {
		return entombTransactionResult{}, entombRetainedError("publish", preflight.Transaction, err)
	}
	stages = append(stages, entombOrderedStages[4])

	tx, publishTargetCount, err := buildEntombLifecycleTransaction(input, preflight, archiveStage, manifest)
	if err != nil {
		return entombTransactionResult{}, entombRetainedError("transaction", preflight.Transaction, err)
	}
	clearBoundary := fmt.Sprintf("after_target_commit:target-%04d", publishTargetCount)
	tx.config.Fault = func(point string) error {
		if point == clearBoundary {
			if err := callEntombFault(input.Fault, entombOrderedStages[4]); err != nil {
				return err
			}
		}
		return nil
	}
	receipt, err := tx.Commit()
	if err != nil {
		effect := colony.LifecycleStateEffectRetained
		if tx.progress != nil && tx.progress.StateEffect.Valid() && tx.progress.StateEffect != colony.LifecycleStateEffectNone {
			effect = tx.progress.StateEffect
		}
		return entombTransactionResult{}, entombFailureWithEffect("transaction", preflight.Transaction, effect, err)
	}
	result, err := loadPublishedEntombResult(preflight.ChamberPath, receipt, stages, false)
	if err != nil {
		return entombTransactionResult{}, entombRetainedError("published verification", preflight.Transaction, err)
	}
	result.Goal = preflight.Goal
	result.Scope = preflight.Scope
	return result, nil
}

func loadEntombTransactionState(root, dataRoot string) (string, string, colony.ColonyState, error) {
	resolvedRoot, err := entombManifestRoot(root, "repository")
	if err != nil {
		return "", "", colony.ColonyState{}, err
	}
	resolvedData, err := entombManifestRoot(dataRoot, "lifecycle data")
	if err != nil {
		return "", "", colony.ColonyState{}, err
	}
	statePath := filepath.Join(resolvedData, "COLONY_STATE.json")
	content, err := readLifecycleEvidenceFile(statePath)
	if err != nil {
		return "", "", colony.ColonyState{}, fmt.Errorf("read active state %q: %w", statePath, err)
	}
	var state colony.ColonyState
	if err := json.Unmarshal(content, &state); err != nil {
		return "", "", colony.ColonyState{}, fmt.Errorf("decode active state %q: %w", statePath, err)
	}
	return resolvedRoot, resolvedData, state, nil
}

func prepareEntombPreflight(input entombTransactionInput, state colony.ColonyState) (entombPreflight, error) {
	if state.State != colony.StateCOMPLETED || strings.TrimSpace(state.Milestone) != "Crowned Anthill" {
		return entombPreflight{}, fmt.Errorf("colony is not sealed Crowned Anthill state; run `/ant-seal` first")
	}
	if state.Goal == nil || strings.TrimSpace(*state.Goal) == "" {
		return entombPreflight{}, fmt.Errorf("sealed colony goal is unavailable")
	}
	if state.SealOutcome == nil {
		return entombPreflight{}, fmt.Errorf("COLONY_STATE.json has no verifiable seal outcome — this colony was sealed by an older runtime that never wrote one. Re-seal it under this runtime to write the verifiable record, then retry: aether seal --force --reason \"re-seal legacy colony for archive\"")
	}
	outcome := *state.SealOutcome
	if err := outcome.Validate(); err != nil {
		return entombPreflight{}, fmt.Errorf("COLONY_STATE.json seal outcome: %w", err)
	}
	if outcome.Transaction.Stage != colony.TransactionStageVerified {
		return entombPreflight{}, fmt.Errorf("seal transaction %q is not verified", outcome.Transaction.ID)
	}
	if outcome.ProjectionRevision != "" && outcome.ProjectionRevision != LifecycleProjectionRevision {
		return entombPreflight{}, fmt.Errorf("seal projection revision %q does not match %q", outcome.ProjectionRevision, LifecycleProjectionRevision)
	}

	archiveID, transactionID := entombTransactionIdentity(outcome)
	goal := strings.TrimSpace(*state.Goal)
	scope := state.EffectiveScope()
	chamberName := "chamber-" + string(scope) + "-" + sanitizeChamberGoal(goal) + "-" + strings.TrimPrefix(archiveID, "archive-")
	preflight := entombPreflight{
		Root: input.Root, DataRoot: input.DataRoot, State: state, Outcome: outcome,
		ArchiveID: archiveID, Transaction: transactionID, ChamberName: chamberName,
		ChamberPath: filepath.Join(input.Root, ".aether", "chambers", chamberName),
		Goal:        goal, Scope: scope,
	}

	seenActual := make(map[string]bool)
	seenArchive := make(map[string]bool)
	addFile := func(logical, archived, kind, actual string, clear bool) error {
		if seenActual[actual] {
			return fmt.Errorf("duplicate entomb source %q", actual)
		}
		if seenArchive[archived] {
			return fmt.Errorf("duplicate entomb archive path %q", archived)
		}
		content, err := readEntombLiveFile(actual, input.Root, input.DataRoot)
		if err != nil {
			return fmt.Errorf("required source %q: %w", logical, err)
		}
		if err := verifyEntombClosureArtifact(logical, kind, content, outcome); err != nil {
			return err
		}
		seenActual[actual], seenArchive[archived] = true, true
		preflight.Sources = append(preflight.Sources, entombPreparedSource{
			Manifest: entombArchiveSource{SourcePath: logical, ArchivePath: archived, Kind: kind, Required: true},
			Actual:   actual, Content: content, Clear: clear,
		})
		return nil
	}

	// addRequiredOrSynthesizedTombstoneInput keeps every other aspect of
	// addFile's behaviour (the same archived name, the same tombstone_input
	// kind, the same closure-artifact verification, the same appending to
	// the preflight's source list) and differs only in its failure branch:
	// when the file genuinely cannot be found, it builds a minimal stand-in
	// document from the colony state and seal outcome already in scope
	// instead of failing the archive. A non-not-exist error (permission,
	// symlink, corrupt content) still fails hard exactly like addFile.
	addRequiredOrSynthesizedTombstoneInput := func(logical, archived, actual string) error {
		if seenActual[actual] {
			return fmt.Errorf("duplicate entomb source %q", actual)
		}
		if seenArchive[archived] {
			return fmt.Errorf("duplicate entomb archive path %q", archived)
		}
		content, err := readEntombLiveFile(actual, input.Root, input.DataRoot)
		synthesized := false
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("required source %q: %w", logical, err)
			}
			content = buildSynthesizedEntombTombstoneInput(state, outcome)
			synthesized = true
		}
		if err := verifyEntombClosureArtifact(logical, "tombstone_input", content, outcome); err != nil {
			return err
		}
		seenActual[actual], seenArchive[archived] = true, true
		// Actual is left empty for a synthesised stand-in: there is no live
		// file backing it, so verifyEntombLiveSources (which skips entries
		// with an empty Actual) never tries to re-read a file that was never
		// there, and entombSourceDigest reports the pre-declare baseline as
		// "missing" rather than the digest of invented content.
		liveActual := actual
		if synthesized {
			liveActual = ""
		}
		preflight.Sources = append(preflight.Sources, entombPreparedSource{
			Manifest: entombArchiveSource{SourcePath: logical, ArchivePath: archived, Kind: "tombstone_input", Required: true, Synthesized: synthesized},
			Actual:   liveActual, Content: content, Clear: false,
		})
		return nil
	}

	required := []struct {
		logical  string
		archived string
		kind     string
		actual   string
		clear    bool
	}{
		{".aether/data/COLONY_STATE.json", "COLONY_STATE.json", "state", filepath.Join(input.DataRoot, "COLONY_STATE.json"), false},
		{".aether/CROWNED-ANTHILL.md", "CROWNED-ANTHILL.md", "crowned_report", filepath.Join(input.Root, ".aether", "CROWNED-ANTHILL.md"), true},
		{".aether/data/seal/outcome.json", "seal/outcome.json", "seal_outcome", filepath.Join(input.DataRoot, "seal", "outcome.json"), true},
		{".aether/data/seal/findings.json", "seal/findings.json", "findings", filepath.Join(input.DataRoot, "seal", "findings.json"), true},
		{".aether/data/seal/learnings.json", "seal/learnings.json", "learnings", filepath.Join(input.DataRoot, "seal", "learnings.json"), true},
		{".aether/data/pheromones.json", "pheromones.json", "signals", filepath.Join(input.DataRoot, "pheromones.json"), true},
		{".aether/data/seal/checkpoints.json", "seal/checkpoints.json", "owner_checkpoints", filepath.Join(input.DataRoot, "seal", "checkpoints.json"), true},
		{".aether/data/seal/rollback.json", "seal/rollback.json", "rollback", filepath.Join(input.DataRoot, "seal", "rollback.json"), true},
		{".aether/QUEEN.md", "QUEEN.md", "retained_memory", filepath.Join(input.Root, ".aether", "QUEEN.md"), false},
	}
	for _, item := range required {
		if err := addFile(item.logical, item.archived, item.kind, item.actual, item.clear); err != nil {
			return entombPreflight{}, err
		}
	}
	if err := addRequiredOrSynthesizedTombstoneInput(".aether/HANDOFF.md", "HANDOFF.md", filepath.Join(input.Root, ".aether", "HANDOFF.md")); err != nil {
		return entombPreflight{}, err
	}

	xmlBytes, err := buildEntombArchiveXMLBytes(input.DataRoot, state, outcome, input.Now)
	if err != nil {
		return entombPreflight{}, fmt.Errorf("build colony archive XML: %w", err)
	}
	xmlLogical := ".aether/data/entomb/colony-archive.xml"
	if err := verifyEntombClosureArtifact(xmlLogical, "archive_xml", xmlBytes, outcome); err != nil {
		return entombPreflight{}, err
	}
	preflight.Sources = append(preflight.Sources, entombPreparedSource{
		Manifest: entombArchiveSource{SourcePath: xmlLogical, ArchivePath: "colony-archive.xml", Kind: "archive_xml", Required: true},
		Content:  xmlBytes,
	})
	seenArchive["colony-archive.xml"] = true

	if err := appendEntombDataSources(&preflight, seenActual, seenArchive); err != nil {
		return entombPreflight{}, err
	}
	if err := appendEntombRepositorySources(&preflight, seenActual, seenArchive); err != nil {
		return entombPreflight{}, err
	}
	sort.Slice(preflight.Sources, func(i, j int) bool {
		return preflight.Sources[i].Manifest.ArchivePath < preflight.Sources[j].Manifest.ArchivePath
	})
	return preflight, nil
}

func entombTransactionIdentity(outcome colony.SealOutcome) (string, string) {
	digest := strings.TrimPrefix(lifecycleDigest([]byte(outcome.OutcomeID+"\x00"+outcome.Transaction.ID)), "sha256:")
	token := digest[:20]
	return "archive-" + token, "entomb-" + token
}

func appendEntombDataSources(preflight *entombPreflight, seenActual, seenArchive map[string]bool) error {
	return filepath.WalkDir(preflight.DataRoot, func(actual string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(preflight.DataRoot, actual)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		first := strings.Split(relative, string(filepath.Separator))[0]
		if entry.IsDir() && (first == "transactions" || first == lifecycleTransactionDirectory) {
			return fs.SkipDir
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("entomb source %q is a symlink", actual)
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() || seenActual[actual] {
			return nil
		}
		archivePath := filepath.ToSlash(relative)
		if seenArchive[archivePath] {
			return fmt.Errorf("archive path %q collides while enumerating active data", archivePath)
		}
		content, err := readEntombLiveFile(actual, preflight.Root, preflight.DataRoot)
		if err != nil {
			return err
		}
		kind := "runtime_data"
		switch archivePath {
		case "session.json":
			kind = "session"
		case "seal/receipt.json":
			kind = "seal_receipt"
		case "seal/closure-evidence.json":
			kind = "closure_evidence"
		}
		seenActual[actual], seenArchive[archivePath] = true, true
		preflight.Sources = append(preflight.Sources, entombPreparedSource{
			Manifest: entombArchiveSource{
				SourcePath: ".aether/data/" + archivePath, ArchivePath: archivePath,
				Kind: kind, Required: true,
			},
			Actual: actual, Content: content, Clear: true,
		})
		return nil
	})
}

func appendEntombRepositorySources(preflight *entombPreflight, seenActual, seenArchive map[string]bool) error {
	optionalFiles := []struct {
		logical  string
		archived string
		actual   string
		kind     string
	}{
		// Archived as "repository-context.md", not "CONTEXT.md": the data-root
		// walk in appendEntombDataSources already claims the archive name
		// "CONTEXT.md" for .aether/data/CONTEXT.md (a distinct runtime data
		// file with the same basename but different content/purpose). Both
		// files exist on a normal active colony, so giving this one the same
		// archive name collided in seenArchive and failed every entomb.
		{".aether/CONTEXT.md", "repository-context.md", filepath.Join(preflight.Root, ".aether", "CONTEXT.md"), "tombstone_context"},
	}
	for _, item := range optionalFiles {
		if seenActual[item.actual] {
			continue
		}
		content, err := readEntombLiveFile(item.actual, preflight.Root, preflight.DataRoot)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if seenArchive[item.archived] {
			return fmt.Errorf("duplicate optional archive path %q", item.archived)
		}
		seenActual[item.actual], seenArchive[item.archived] = true, true
		preflight.Sources = append(preflight.Sources, entombPreparedSource{
			Manifest: entombArchiveSource{SourcePath: item.logical, ArchivePath: item.archived, Kind: item.kind, Required: true},
			Actual:   item.actual, Content: content,
		})
	}
	for _, directory := range []string{"dreams", "exchange"} {
		base := filepath.Join(preflight.Root, ".aether", directory)
		if _, err := os.Lstat(base); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return err
		}
		if err := filepath.WalkDir(base, func(actual string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("entomb source %q is a symlink", actual)
			}
			if entry.IsDir() {
				return nil
			}
			if !entry.Type().IsRegular() || seenActual[actual] {
				return nil
			}
			relative, err := filepath.Rel(filepath.Join(preflight.Root, ".aether"), actual)
			if err != nil {
				return err
			}
			archivePath := filepath.ToSlash(relative)
			if seenArchive[archivePath] {
				return fmt.Errorf("duplicate repository archive path %q", archivePath)
			}
			content, err := readEntombLiveFile(actual, preflight.Root, preflight.DataRoot)
			if err != nil {
				return err
			}
			seenActual[actual], seenArchive[archivePath] = true, true
			preflight.Sources = append(preflight.Sources, entombPreparedSource{
				Manifest: entombArchiveSource{SourcePath: ".aether/" + archivePath, ArchivePath: archivePath, Kind: "retained_" + directory, Required: true},
				Actual:   actual, Content: content,
			})
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

func readEntombLiveFile(actual, root, dataRoot string) ([]byte, error) {
	contained := pathIsWithin(root, actual) || pathIsWithin(dataRoot, actual)
	if !contained {
		return nil, fmt.Errorf("entomb source %q is outside the repository and lifecycle-data roots", actual)
	}
	containingRoot := root
	if pathIsWithin(dataRoot, actual) {
		containingRoot = dataRoot
	}
	if err := rejectLifecycleSymlinkTarget(containingRoot, actual); err != nil {
		return nil, err
	}
	return readLifecycleEvidenceFile(actual)
}

type entombClosureArchiveXML struct {
	XMLName         xml.Name               `xml:"colony-archive"`
	ColonyID        string                 `xml:"colony_id,attr"`
	SealOutcomeID   string                 `xml:"seal_outcome_id,attr"`
	SealTransaction string                 `xml:"seal_transaction_id,attr"`
	Disposition     colony.SealDisposition `xml:"disposition,attr"`
	OwnerReason     string                 `xml:"owner_reason,attr,omitempty"`
	SealedAt        string                 `xml:"sealed_at,attr"`
	Version         string                 `xml:"version,attr"`
	Pheromones      *exchange.PheromoneXML `xml:"pheromones"`
	Wisdom          *exchange.WisdomXML    `xml:"queen-wisdom"`
	Registry        *exchange.RegistryXML  `xml:"colony-registry"`
}

func buildEntombArchiveXMLBytes(dataRoot string, state colony.ColonyState, outcome colony.SealOutcome, now time.Time) ([]byte, error) {
	var pheromones colony.PheromoneFile
	pheromoneBytes, err := readLifecycleEvidenceFile(filepath.Join(dataRoot, "pheromones.json"))
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(pheromoneBytes, &pheromones); err != nil {
		return nil, fmt.Errorf("decode pheromones: %w", err)
	}
	pheromoneXMLBytes, err := exchange.ExportPheromones(pheromones.Signals)
	if err != nil {
		return nil, err
	}
	var pheromoneXML exchange.PheromoneXML
	if err := xml.Unmarshal(pheromoneXMLBytes, &pheromoneXML); err != nil {
		return nil, err
	}

	wisdomEntries := make([]exchange.WisdomEntry, 0, len(state.Memory.Instincts))
	for _, instinct := range state.Memory.Instincts {
		wisdomEntries = append(wisdomEntries, exchange.WisdomEntry{
			ID: instinct.ID, Category: "pattern", Confidence: instinct.Confidence,
			Domain: instinct.Domain, Source: instinct.Source,
			CreatedAt: instinct.CreatedAt, Content: instinct.Trigger,
		})
	}
	wisdomXMLBytes, err := exchange.ExportWisdom(wisdomEntries, 0, "")
	if err != nil {
		return nil, err
	}
	var wisdomXML exchange.WisdomXML
	if err := xml.Unmarshal(wisdomXMLBytes, &wisdomXML); err != nil {
		return nil, err
	}
	sealedAt := now.UTC().Format(time.RFC3339)
	if state.MilestoneUpdatedAt != nil && strings.TrimSpace(*state.MilestoneUpdatedAt) != "" {
		sealedAt = strings.TrimSpace(*state.MilestoneUpdatedAt)
	}
	archive := entombClosureArchiveXML{
		ColonyID: outcome.OutcomeID, SealOutcomeID: outcome.OutcomeID,
		SealTransaction: outcome.Transaction.ID, Disposition: outcome.Disposition,
		OwnerReason: outcome.OwnerReason, SealedAt: sealedAt, Version: "1.0",
		Pheromones: &pheromoneXML, Wisdom: &wisdomXML,
		Registry: &exchange.RegistryXML{Version: "1.0"},
	}
	content, err := xml.MarshalIndent(archive, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), content...), nil
}

func writeEntombStagedFile(root, relative string, content []byte) error {
	target, err := entombContainedManifestPath(root, relative, "staged")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return err
	}
	return os.WriteFile(target, content, 0o600)
}

func verifyEntombLiveSources(preflight entombPreflight) error {
	for _, source := range preflight.Sources {
		if source.Actual == "" {
			continue
		}
		content, err := readEntombLiveFile(source.Actual, preflight.Root, preflight.DataRoot)
		if err != nil {
			return fmt.Errorf("source %q changed after preflight: %w", source.Manifest.SourcePath, err)
		}
		if lifecycleDigest(content) != lifecycleDigest(source.Content) || !bytes.Equal(content, source.Content) {
			return fmt.Errorf("source %q digest changed after preflight", source.Manifest.SourcePath)
		}
	}
	return nil
}

func buildEntombLifecycleTransaction(input entombTransactionInput, preflight entombPreflight, archiveStage string, manifest colony.ArchiveManifest) (*lifecycleTransaction, int, error) {
	tx, err := beginLifecycleTransaction(lifecycleTransactionConfig{
		TransactionID: preflight.Transaction,
		Command:       "entomb",
		Allowlist: lifecycleTransactionAllowlist{
			RepositoryRoot: input.Root, LifecycleDataRoot: input.DataRoot,
		},
		Rename: input.Rename,
	})
	if err != nil {
		return nil, 0, err
	}

	var archiveFiles []string
	if err := filepath.WalkDir(archiveStage, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(archiveStage, path)
		if err != nil {
			return err
		}
		archiveFiles = append(archiveFiles, relative)
		return nil
	}); err != nil {
		return nil, 0, err
	}
	sort.Strings(archiveFiles)
	chamberRelative := filepath.Join(".aether", "chambers", preflight.ChamberName)
	for _, relative := range archiveFiles {
		content, err := readLifecycleEvidenceFile(filepath.Join(archiveStage, relative))
		if err != nil {
			return nil, 0, err
		}
		if err := tx.DeclareWrite(lifecycleTransactionRootRepository, filepath.Join(chamberRelative, relative), content); err != nil {
			return nil, 0, err
		}
	}

	tombstone := buildEntombTombstone(preflight, manifest)
	context := buildEntombContext(preflight, manifest)
	if err := declareEntombWrite(tx, lifecycleTransactionRootRepository, filepath.Join(".aether", "HANDOFF.md"), []byte(tombstone), entombSourceDigest(preflight.Sources, "tombstone_input")); err != nil {
		return nil, 0, err
	}
	contextDigest := entombSourceDigest(preflight.Sources, "tombstone_context")
	if err := declareEntombWrite(tx, lifecycleTransactionRootRepository, filepath.Join(".aether", "CONTEXT.md"), []byte(context), contextDigest); err != nil {
		return nil, 0, err
	}
	publishTargetCount := len(tx.declarations)

	if err := declareEntombRemoval(tx, lifecycleTransactionRootRepository, filepath.Join(".aether", "CROWNED-ANTHILL.md"), entombSourceDigest(preflight.Sources, "crowned_report")); err != nil {
		return nil, 0, err
	}
	for _, source := range preflight.Sources {
		if !source.Clear || source.Actual == filepath.Join(preflight.DataRoot, "COLONY_STATE.json") || !pathIsWithin(preflight.DataRoot, source.Actual) {
			continue
		}
		relative, err := filepath.Rel(preflight.DataRoot, source.Actual)
		if err != nil {
			return nil, 0, err
		}
		if err := declareEntombRemoval(tx, lifecycleTransactionRootData, relative, lifecycleDigest(source.Content)); err != nil {
			return nil, 0, err
		}
	}
	reset := resetColonyStateForEntomb(preflight.State)
	reset.ArchiveReference = &colony.ArchiveReference{
		ID: preflight.ArchiveID, TransactionID: preflight.Transaction,
		ManifestDigest: manifest.ManifestDigest,
		Path:           filepath.ToSlash(filepath.Join(".aether", "chambers", preflight.ChamberName, "manifest.json")),
	}
	resetBytes, err := json.MarshalIndent(reset, "", "  ")
	if err != nil {
		return nil, 0, err
	}
	resetBytes = append(resetBytes, '\n')
	if err := declareEntombWrite(tx, lifecycleTransactionRootData, "COLONY_STATE.json", resetBytes, entombSourceDigest(preflight.Sources, "state")); err != nil {
		return nil, 0, err
	}
	return tx, publishTargetCount, nil
}

func declareEntombWrite(tx *lifecycleTransaction, root lifecycleTransactionRootKind, relative string, content []byte, expectedBefore string) error {
	if err := tx.DeclareWrite(root, relative, content); err != nil {
		return err
	}
	return verifyEntombDeclaredBaseline(tx, expectedBefore)
}

func declareEntombRemoval(tx *lifecycleTransaction, root lifecycleTransactionRootKind, relative, expectedBefore string) error {
	if err := tx.DeclareRemoval(root, relative); err != nil {
		return err
	}
	return verifyEntombDeclaredBaseline(tx, expectedBefore)
}

func verifyEntombDeclaredBaseline(tx *lifecycleTransaction, expected string) error {
	if expected == "" {
		return nil
	}
	declaration := tx.declarations[len(tx.declarations)-1]
	if declaration.BeforeDigest != expected {
		return fmt.Errorf("source %q digest changed before transaction intent", declaration.TargetPath)
	}
	return nil
}

func entombSourceDigest(sources []entombPreparedSource, kind string) string {
	for _, source := range sources {
		if source.Manifest.Kind == kind {
			if source.Actual == "" {
				// A synthesised stand-in never had a live file backing it, so
				// the correct pre-declare baseline is "missing" (the same
				// sentinel readLifecycleFileState reports for a genuinely
				// absent file) rather than the digest of invented content —
				// otherwise verifyEntombDeclaredBaseline would reject every
				// synthesised entry as "changed" before the write.
				return lifecycleTransactionMissingDigest
			}
			return lifecycleDigest(source.Content)
		}
	}
	return ""
}

func buildEntombTombstone(preflight entombPreflight, manifest colony.ArchiveManifest) string {
	lines := []string{
		"# Entombed colony tombstone",
		"",
		"The sealed colony was archived and its active state was cleared only after archive verification.",
		"",
		"- Archive: " + manifest.ArchiveID,
		"- Chamber: " + filepath.ToSlash(preflight.ChamberPath),
		"- Manifest digest: " + manifest.ManifestDigest,
		"- Transaction receipt: " + preflight.Transaction + "-receipt",
		"- Seal disposition: " + string(preflight.Outcome.Disposition),
		"- Retained: colony memory, seal evidence, and tombstone input",
	}
	if preflight.Outcome.Disposition == colony.SealDispositionForcedIncomplete {
		lines = append(lines, "- Owner reason: "+preflight.Outcome.OwnerReason)
	}
	lines = append(lines, "", `Next: /ant-init "next goal"`, "")
	return strings.Join(lines, "\n")
}

// buildSynthesizedEntombTombstoneInput builds the minimal stand-in used when
// .aether/HANDOFF.md is missing at archive time (e.g. a resume that removed
// it before this runtime paired that removal with a fresh write — see
// session_flow_cmds.go). It carries only facts already loaded in the same
// preflight scope: the colony's goal, its phase count and completion state,
// and the seal outcome's verdict. It never invents content the archive
// cannot otherwise attest to.
func buildSynthesizedEntombTombstoneInput(state colony.ColonyState, outcome colony.SealOutcome) []byte {
	goal := "No goal set"
	if state.Goal != nil && strings.TrimSpace(*state.Goal) != "" {
		goal = strings.TrimSpace(*state.Goal)
	}
	totalPhases := len(state.Plan.Phases)
	var b strings.Builder
	b.WriteString("# Colony Handoff (synthesised stand-in)\n\n")
	b.WriteString("The hand-off note was missing when this colony was archived. The archive\n")
	b.WriteString("synthesised this minimal stand-in from the colony's own records rather than\n")
	b.WriteString("failing the archive.\n\n")
	b.WriteString("Goal: ")
	b.WriteString(goal)
	b.WriteString("\n")
	b.WriteString("State: ")
	b.WriteString(string(state.State))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Phase: %d/%d\n", state.CurrentPhase, totalPhases))
	b.WriteString("Seal verdict: ")
	b.WriteString(string(outcome.Disposition))
	b.WriteString("\n")
	return []byte(b.String())
}

func buildEntombContext(preflight entombPreflight, manifest colony.ArchiveManifest) string {
	lines := []string{
		"# Colony Context",
		"",
		"State: IDLE",
		"Latest chamber: " + filepath.ToSlash(preflight.ChamberPath),
		"Archive manifest: " + manifest.ManifestDigest,
		"Seal disposition: " + string(preflight.Outcome.Disposition),
	}
	if preflight.Outcome.Disposition == colony.SealDispositionForcedIncomplete {
		lines = append(lines, "Owner reason: "+preflight.Outcome.OwnerReason)
	}
	lines = append(lines, `Recommended next action: /ant-init "next goal"`, "")
	return strings.Join(lines, "\n")
}

func replayEntombTransaction(input entombTransactionInput, state colony.ColonyState) (entombTransactionResult, bool, error) {
	transactionID := ""
	chamberPath := ""
	if state.ArchiveReference != nil && strings.TrimSpace(state.ArchiveReference.TransactionID) != "" {
		transactionID = state.ArchiveReference.TransactionID
		manifestPath := filepath.Join(input.Root, filepath.FromSlash(state.ArchiveReference.Path))
		if !pathIsWithin(filepath.Join(input.Root, ".aether", "chambers"), manifestPath) {
			return entombTransactionResult{}, true, entombRetainedError("replay", transactionID, fmt.Errorf("archive reference path %q escapes chambers", state.ArchiveReference.Path))
		}
		chamberPath = filepath.Dir(manifestPath)
	} else if state.SealOutcome != nil && state.SealOutcome.Validate() == nil {
		archiveID, candidate := entombTransactionIdentity(*state.SealOutcome)
		transactionID = candidate
		goal := "colony"
		if state.Goal != nil {
			goal = strings.TrimSpace(*state.Goal)
		}
		chamberName := "chamber-" + string(state.EffectiveScope()) + "-" + sanitizeChamberGoal(goal) + "-" + strings.TrimPrefix(archiveID, "archive-")
		chamberPath = filepath.Join(input.Root, ".aether", "chambers", chamberName)
	}
	if transactionID == "" {
		return entombTransactionResult{}, false, nil
	}
	journalIntent := filepath.Join(input.DataRoot, "transactions", transactionID, "intent.json")
	journalReceipt := filepath.Join(input.DataRoot, "transactions", transactionID, "receipt.json")
	if _, err := os.Lstat(journalIntent); os.IsNotExist(err) {
		return entombTransactionResult{}, false, nil
	} else if err != nil {
		return entombTransactionResult{}, true, entombRetainedError("replay", transactionID, err)
	}
	receipt, err := resumeLifecycleTransaction(lifecycleTransactionConfig{
		TransactionID: transactionID, Command: "entomb",
		Allowlist: lifecycleTransactionAllowlist{RepositoryRoot: input.Root, LifecycleDataRoot: input.DataRoot},
		Rename:    input.Rename,
	})
	if err != nil {
		return entombTransactionResult{}, true, entombRetainedError("replay", transactionID, fmt.Errorf("inspect %s and retry `/ant-entomb --confirm`: %w", filepath.ToSlash(journalIntent), err))
	}
	if _, err := os.Lstat(journalReceipt); err != nil {
		return entombTransactionResult{}, true, entombRetainedError("replay", transactionID, fmt.Errorf("verified receipt %q is unavailable: %w", journalReceipt, err))
	}
	result, err := loadPublishedEntombResult(chamberPath, receipt, append([]string(nil), entombOrderedStages...), true)
	if err != nil {
		return entombTransactionResult{}, true, entombRetainedError("replay", transactionID, err)
	}
	return result, true, nil
}

func loadPublishedEntombResult(chamberPath string, receipt colony.LifecycleReceipt, stages []string, replay bool) (entombTransactionResult, error) {
	manifestPath := filepath.Join(chamberPath, "manifest.json")
	manifestBytes, err := readLifecycleEvidenceFile(manifestPath)
	if err != nil {
		return entombTransactionResult{}, fmt.Errorf("verified chamber manifest %q is unavailable: %w", manifestPath, err)
	}
	var manifest colony.ArchiveManifest
	if err := decodeLifecycleJSON(manifestBytes, &manifest); err != nil {
		return entombTransactionResult{}, fmt.Errorf("decode verified chamber manifest: %w", err)
	}
	if err := verifyPublishedEntombManifest(chamberPath, manifest, receipt); err != nil {
		return entombTransactionResult{}, err
	}
	result := entombTransactionResult{
		ArchiveID: manifest.ArchiveID, ChamberName: filepath.Base(chamberPath), ChamberPath: chamberPath,
		ManifestDigest: manifest.ManifestDigest, Receipt: receipt,
		Disposition: manifest.Seal.Disposition, OwnerReason: manifest.Seal.OwnerReason,
		Retained: []string{"colony memory", "seal evidence", "tombstone input"},
		Stages:   stages, Next: `/ant-init "next goal"`, Replay: replay,
	}
	for _, entry := range manifest.Entries {
		if entry.Kind != "state" {
			continue
		}
		content, err := readLifecycleEvidenceFile(filepath.Join(chamberPath, filepath.FromSlash(entry.Path)))
		if err != nil {
			return entombTransactionResult{}, err
		}
		var archivedState colony.ColonyState
		if err := json.Unmarshal(content, &archivedState); err != nil {
			return entombTransactionResult{}, err
		}
		if archivedState.Goal != nil {
			result.Goal = strings.TrimSpace(*archivedState.Goal)
		}
		result.Scope = archivedState.EffectiveScope()
		break
	}
	return result, nil
}

func verifyPublishedEntombManifest(chamberPath string, manifest colony.ArchiveManifest, receipt colony.LifecycleReceipt) error {
	if err := manifest.Validate(); err != nil {
		return err
	}
	if lifecycleDigest(entombManifestDigestBytes(manifest)) != manifest.ManifestDigest {
		return fmt.Errorf("manifest digest does not match published bytes")
	}
	if manifest.Transaction.ID != receipt.Transaction.ID || manifest.Receipt == nil || manifest.Receipt.ID != receipt.ReceiptID {
		return fmt.Errorf("manifest transaction/receipt does not match the durable transaction receipt")
	}
	var outcome colony.SealOutcome
	for _, entry := range manifest.Entries {
		path, err := entombContainedManifestPath(chamberPath, entry.Path, "published archive")
		if err != nil {
			return err
		}
		content, err := readEntombManifestFile(chamberPath, entry.Path, path)
		if err != nil {
			return entombManifestReadError("published archive", entry.Path, err)
		}
		if int64(len(content)) != entry.Size || lifecycleDigest(content) != entry.ArchiveDigest {
			return fmt.Errorf("published archive %q size/digest does not match manifest", entry.Path)
		}
		if entry.Kind == "seal_outcome" {
			if err := decodeLifecycleJSON(content, &outcome); err != nil {
				return err
			}
		}
	}
	if outcome.OutcomeID == "" {
		return fmt.Errorf("published archive is missing its seal outcome")
	}
	if outcome.OutcomeID != manifest.Seal.OutcomeID || outcome.Transaction.ID != manifest.Seal.TransactionID || outcome.Disposition != manifest.Seal.Disposition || outcome.OwnerReason != manifest.Seal.OwnerReason {
		return fmt.Errorf("published seal outcome conflicts with manifest cross-reference")
	}
	for _, entry := range manifest.Entries {
		if !entombManifestReferenceExists(manifest, "source:"+entry.Kind, entry.Path, entry.ArchiveDigest, false) {
			return fmt.Errorf("published archive %q is missing its source cross-reference", entry.Path)
		}
		if entombClosureArchiveKinds[entry.Kind] && !entombManifestReferenceExists(manifest, "closure:"+entry.Kind, entry.Path, entry.ArchiveDigest, true) {
			return fmt.Errorf("published archive %q is missing its closure cross-reference", entry.Path)
		}
		if entombClosureArchiveKinds[entry.Kind] {
			content, err := os.ReadFile(filepath.Join(chamberPath, filepath.FromSlash(entry.Path)))
			if err != nil {
				return err
			}
			if err := verifyEntombClosureArtifact(entry.Path, entry.Kind, content, outcome); err != nil {
				return err
			}
		}
	}
	return nil
}

func entombManifestReferenceExists(manifest colony.ArchiveManifest, kind, target, digest string, closure bool) bool {
	for _, reference := range manifest.CrossReferences {
		if reference.Kind != kind || reference.TargetID != target || reference.Digest != digest {
			continue
		}
		if closure && reference.SourceID != manifest.Seal.OutcomeID {
			continue
		}
		return true
	}
	return false
}

func callEntombFault(fault entombTransactionFaultHook, point string) error {
	if fault == nil {
		return nil
	}
	return fault(point)
}

func entombRetainedError(stage, transactionID string, cause error) error {
	return entombFailureWithEffect(stage, transactionID, colony.LifecycleStateEffectRetained, cause)
}

func entombFailureWithEffect(stage, transactionID string, effect colony.LifecycleStateEffect, cause error) error {
	if !effect.Valid() || effect == colony.LifecycleStateEffectNone || effect == colony.LifecycleStateEffectCommitted {
		effect = colony.LifecycleStateEffectRetained
	}
	return &entombFailureError{Stage: strings.TrimSpace(stage), TransactionID: strings.TrimSpace(transactionID), StateEffect: effect, cause: cause}
}

func entombResultMap(result entombTransactionResult) map[string]interface{} {
	payload := map[string]interface{}{
		"entombed":              !result.AwaitingConfirmation,
		"awaiting_confirmation": result.AwaitingConfirmation,
		"confirmation":          result.Confirmation,
		"archive_id":            result.ArchiveID,
		"goal":                  result.Goal,
		"scope":                 string(result.Scope),
		"chamber":               result.ChamberName,
		"chamber_path":          result.ChamberPath,
		"manifest_digest":       result.ManifestDigest,
		"transaction_receipt":   result.Receipt.ReceiptID,
		"seal_disposition":      string(result.Disposition),
		"owner_reason":          result.OwnerReason,
		"retained":              result.Retained,
		"stages":                result.Stages,
		"replay":                result.Replay,
		"next":                  result.Next,
	}
	projection := entombLifecycleProjection(result)
	outcome := colony.OutcomeKindArchived
	effect := result.Receipt.StateEffect
	summary := "The verified archive was published before active colony state was cleared."
	if result.AwaitingConfirmation {
		outcome = colony.OutcomeKindNoChange
		effect = colony.LifecycleStateEffectNone
		summary = "The archive preview is ready; active sealed state remains unchanged pending confirmation."
	}
	if !effect.Valid() {
		effect = colony.LifecycleStateEffectCommitted
	}
	payload["outcome_kind"] = outcome
	payload["state_effect"] = effect
	payload["projection_revision"] = projection.ProjectionRevision
	applyNextActionToResult(payload, nextAction{
		Command:        projection.NextAction.RuntimeCommand,
		Recommendation: projection.NextAction.Reason,
		Projection:     &projection,
	})
	details := LifecycleCloseoutDetails{Summary: summary}
	if !result.AwaitingConfirmation {
		details.Evidence = []colony.LifecycleEvidence{
			{ID: "entomb-archive-manifest", Kind: "manifest", Source: filepath.Join(result.ChamberPath, "manifest.json"), Digest: result.ManifestDigest, Summary: "Verified archive bytes and cross-references"},
			{ID: "entomb-transaction-receipt", Kind: "receipt", Source: result.Receipt.Transaction.JournalPath, Summary: "Committed archive and clear transaction receipt"},
		}
		details.Verification = []colony.LifecycleVerification{{Name: "archive publication and clear", Passed: true, EvidenceIDs: []string{"entomb-archive-manifest", "entomb-transaction-receipt"}}}
	}
	if err := applyLifecycleCloseout(payload, "entomb", details); err != nil {
		payload["lifecycle_closeout_error"] = err.Error()
	}
	return payload
}

func entombLifecycleProjection(result entombTransactionResult) LifecycleProjection {
	revision := strings.TrimSpace(result.Receipt.ProjectionRevision)
	if revision == "" {
		revision = LifecycleProjectionRevision
	}
	closure := LifecycleClosureProjection{Status: "archived", Inspectable: true}
	next := entombRuntimeCommand(result.Next)
	reason := "The verified chamber exists and active colony state is clear; a new colony may now be founded."
	standing := "Archived"
	if result.AwaitingConfirmation {
		closure = LifecycleClosureProjection{Status: "verified", Inspectable: true, ArchiveReady: true}
		if result.Disposition == colony.SealDispositionForcedIncomplete {
			closure.Status = "forced_incomplete"
			closure.Forced = true
			closure.OwnerReason = strings.TrimSpace(result.OwnerReason)
		}
		reason = "Confirm the separately authorized archive operation before any active state is cleared."
		standing = "Sealed and retained pending archive confirmation"
	}
	effect := result.Receipt.StateEffect
	if result.AwaitingConfirmation {
		effect = colony.LifecycleStateEffectNone
	} else if !effect.Valid() {
		effect = colony.LifecycleStateEffectCommitted
	}
	return LifecycleProjection{
		SchemaVersion:      LifecycleResultSchemaVersion,
		Command:            "entomb",
		OutcomeKind:        colony.OutcomeKindArchived,
		ProjectionRevision: revision,
		View:               LifecycleViewFocused,
		Platform:           "codex",
		Identity: LifecycleFact[LifecycleIdentityFacts]{Value: LifecycleIdentityFacts{
			Name: result.ChamberName, Goal: result.Goal, Standing: standing,
		}},
		Goal:         LifecycleFact[string]{Value: result.Goal},
		Standing:     LifecycleFact[string]{Value: standing},
		Changes:      append([]colony.LifecycleChange(nil), result.Receipt.Changes...),
		Evidence:     append([]colony.LifecycleEvidence(nil), result.Receipt.Evidence...),
		Verification: append([]colony.LifecycleVerification(nil), result.Receipt.Verification...),
		Warnings:     append([]colony.LifecycleIssue(nil), result.Receipt.Warnings...),
		Debt:         append([]colony.LifecycleIssue(nil), result.Receipt.Debt...),
		Blockers:     append([]colony.LifecycleIssue(nil), result.Receipt.Blockers...),
		NextAction: LifecycleProjectedAction{
			ID: "initialize", RuntimeCommand: next, DisplayCommand: next, Reason: reason,
		},
		Alternatives: []LifecycleActionChoice{{ID: "status", RuntimeCommand: "aether status", DisplayCommand: "aether status", Reason: "Inspect the cleared lifecycle state and archive reference."}},
		StateEffect:  effect,
		Provenance:   colony.RecoveryProvenanceConfirmed,
		Closure:      closure,
	}
}

func entombRuntimeCommand(command string) string {
	command = strings.TrimSpace(command)
	if strings.HasPrefix(command, "/ant-") {
		return "aether " + strings.TrimPrefix(command, "/ant-")
	}
	return command
}

func entombFailureResultForState(state colony.ColonyState, cause error) (map[string]interface{}, error) {
	failure := &entombFailureError{Stage: "archive", StateEffect: colony.LifecycleStateEffectRetained, cause: cause}
	var typed *entombFailureError
	if errors.As(cause, &typed) {
		failure = typed
	}
	result, err := lifecycleCloseoutRefusalForState(
		state,
		"entomb",
		cause.Error(),
		"aether status",
		"Inspect the retained sealed state and transaction journal before retrying the archive.",
	)
	if err != nil {
		return nil, err
	}
	outcome := colony.OutcomeKindFailed
	if failure.StateEffect == colony.LifecycleStateEffectRecoveryRequired {
		outcome = colony.OutcomeKindRecoveryRequired
	}
	journal := ".aether/data/COLONY_STATE.json"
	if failure.TransactionID != "" {
		journal = filepath.ToSlash(filepath.Join(".aether", "data", "transactions", failure.TransactionID)) + "/"
	}
	evidence := colony.LifecycleEvidence{
		ID:      emptyFallback(failure.TransactionID, "entomb-failure") + "-journal",
		Kind:    "coordinator_journal",
		Source:  journal,
		Summary: fmt.Sprintf("Entomb %s failure retained its transaction evidence", emptyFallback(failure.Stage, "archive")),
	}
	result["outcome_kind"] = outcome
	result["state_effect"] = failure.StateEffect
	result["failure_stage"] = failure.Stage
	result["transaction_id"] = failure.TransactionID
	result["journal"] = journal
	result["active_state_retained"] = failure.StateEffect == colony.LifecycleStateEffectRetained || failure.StateEffect == colony.LifecycleStateEffectRolledBack
	result["retry"] = `/ant-entomb --confirm`
	if err := applyLifecycleCloseout(result, "entomb", LifecycleCloseoutDetails{
		Summary:      cause.Error(),
		Evidence:     []colony.LifecycleEvidence{evidence},
		Verification: []colony.LifecycleVerification{{Name: "archive publication and clear", Passed: false, EvidenceIDs: []string{evidence.ID}, Detail: cause.Error()}},
		Blockers:     []colony.LifecycleIssue{{ID: "entomb-" + emptyFallback(failure.Stage, "archive") + "-failure", Summary: cause.Error(), EvidenceIDs: []string{evidence.ID}}},
	}); err != nil {
		return nil, err
	}
	return result, nil
}

func renderEntombFailureVisual(cause error, result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner("❌", "Entomb Not Completed"))
	b.WriteString(visualDividerStr())
	b.WriteString(strings.TrimSpace(cause.Error()))
	b.WriteString("\nThe sealed colony remains available for inspection; no successful archive is claimed.\n\n")
	b.WriteString(renderLifecycleCloseoutFromResult(result, "codex"))
	return b.String()
}

func renderEntombConfirmationVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner("⚰️", "Entomb Preview"))
	b.WriteString(visualDividerStr())
	b.WriteString("Chamber (the verified colony archive): ")
	b.WriteString(stringValue(result["chamber_path"]))
	b.WriteString("\nSeal disposition: ")
	b.WriteString(stringValue(result["seal_disposition"]))
	b.WriteString("\nRetained: colony memory, seal evidence, and tombstone input\n")
	b.WriteString("Active sealed state will be cleared only after verification.\n\n")
	b.WriteString(entombConfirmationCopy())
	b.WriteString("\n")
	return appendLifecycleCloseoutVisual(b.String(), result, "codex")
}

var tunnelsCmd = &cobra.Command{
	Use:   "tunnels [chamber] [other_chamber]",
	Short: "Browse archived chambers",
	Args:  cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		aetherRoot := resolveAetherRootPath()
		chambersDir := filepath.Join(aetherRoot, ".aether", "chambers")
		importSignals, _ := cmd.Flags().GetBool("import-signals")

		if len(args) == 0 {
			chambers, byScope, err := loadChamberManifests(chambersDir)
			if err != nil {
				outputError(1, fmt.Sprintf("failed to read chambers directory: %v", err), nil)
				return nil
			}
			result := map[string]interface{}{
				"mode":     "list",
				"chambers": chambers,
				"by_scope": byScope,
				"total":    len(chambers),
			}
			outputWorkflow(result, renderTunnelsVisual(result))
			return nil
		}

		chamberName := strings.TrimSpace(args[0])
		chamberDir := filepath.Join(chambersDir, chamberName)
		if importSignals {
			result, err := importSignalsFromChamber(chamberName, chamberDir)
			if err != nil {
				outputError(1, err.Error(), nil)
				return nil
			}
			outputWorkflow(result, renderTunnelsVisual(result))
			return nil
		}
		if len(args) == 2 {
			result, err := compareChambers(chambersDir, chamberName, strings.TrimSpace(args[1]))
			if err != nil {
				outputError(1, err.Error(), nil)
				return nil
			}
			outputWorkflow(result, renderTunnelsVisual(result))
			return nil
		}

		manifestPath := filepath.Join(chamberDir, "manifest.json")
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			outputError(1, fmt.Sprintf("chamber %q not found", chamberName), nil)
			return nil
		}

		var manifest map[string]interface{}
		if err := json.Unmarshal(data, &manifest); err != nil {
			outputError(1, fmt.Sprintf("chamber %q has invalid manifest", chamberName), nil)
			return nil
		}
		manifest = manifestWithEffectiveScope(manifest)

		entries, _ := os.ReadDir(chamberDir)
		files := make([]string, 0, len(entries))
		for _, entry := range entries {
			files = append(files, entry.Name())
		}
		sort.Strings(files)

		sealSummary := ""
		if raw, err := os.ReadFile(filepath.Join(chamberDir, "CROWNED-ANTHILL.md")); err == nil {
			sealSummary = string(raw)
		}

		result := map[string]interface{}{
			"mode":         "detail",
			"chamber":      chamberName,
			"manifest":     manifest,
			"files":        files,
			"seal_summary": sealSummary,
		}
		if _, err := os.Stat(filepath.Join(chamberDir, "colony-archive.xml")); err == nil {
			result["archive_xml"] = filepath.ToSlash(filepath.Join(".aether", "chambers", chamberName, "colony-archive.xml"))
			result["import_command"] = fmt.Sprintf("aether tunnels %s --import-signals", chamberName)
		}
		outputWorkflow(result, renderTunnelsVisual(result))
		return nil
	},
}

func loadChamberManifests(chambersDir string) ([]map[string]interface{}, map[string][]interface{}, error) {
	entries, err := os.ReadDir(chambersDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []map[string]interface{}{}, emptyChamberScopeGroups(), nil
		}
		return nil, nil, err
	}
	chambers := []map[string]interface{}{}
	byScope := emptyChamberScopeGroups()
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifestPath := filepath.Join(chambersDir, entry.Name(), "manifest.json")
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}
		var manifest map[string]interface{}
		if err := json.Unmarshal(data, &manifest); err != nil {
			continue
		}
		manifest = manifestWithEffectiveScope(manifest)
		chambers = append(chambers, manifest)
		scope := string(chamberManifestScope(manifest))
		byScope[scope] = append(byScope[scope], manifest)
	}
	sort.SliceStable(chambers, func(i, j int) bool {
		return stringValue(chambers[i]["entombed_at"]) > stringValue(chambers[j]["entombed_at"])
	})
	return chambers, byScope, nil
}

func readChamberManifest(chambersDir, name string) (map[string]interface{}, string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, "", fmt.Errorf("chamber name is required")
	}
	chamberDir := filepath.Join(chambersDir, name)
	manifestPath := filepath.Join(chamberDir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, "", fmt.Errorf("chamber %q not found", name)
	}
	var manifest map[string]interface{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, "", fmt.Errorf("chamber %q has invalid manifest", name)
	}
	return manifestWithEffectiveScope(manifest), chamberDir, nil
}

func compareChambers(chambersDir, leftName, rightName string) (map[string]interface{}, error) {
	left, leftDir, err := readChamberManifest(chambersDir, leftName)
	if err != nil {
		return nil, err
	}
	right, rightDir, err := readChamberManifest(chambersDir, rightName)
	if err != nil {
		return nil, err
	}
	leftSummary := chamberStateMemorySummary(leftDir)
	rightSummary := chamberStateMemorySummary(rightDir)

	leftPhases := intValue(left["phases_completed"])
	rightPhases := intValue(right["phases_completed"])
	leftDecisions := intValue(leftSummary["decisions"])
	rightDecisions := intValue(rightSummary["decisions"])
	leftLearnings := intValue(leftSummary["learnings"])
	rightLearnings := intValue(rightSummary["learnings"])

	growth := map[string]interface{}{
		"phases_diff":    rightPhases - leftPhases,
		"decisions_diff": rightDecisions - leftDecisions,
		"learnings_diff": rightLearnings - leftLearnings,
		"milestone_from": stringValue(left["milestone"]),
		"milestone_to":   stringValue(right["milestone"]),
	}
	if days := daysBetweenChambers(stringValue(left["entombed_at"]), stringValue(right["entombed_at"])); days != 0 {
		growth["days_between"] = days
	}

	return map[string]interface{}{
		"mode":          "compare",
		"chamber_a":     leftName,
		"chamber_b":     rightName,
		"manifest_a":    left,
		"manifest_b":    right,
		"summary_a":     leftSummary,
		"summary_b":     rightSummary,
		"growth":        growth,
		"compare_count": 4,
	}, nil
}

func chamberStateMemorySummary(chamberDir string) map[string]interface{} {
	summary := map[string]interface{}{
		"decisions": 0,
		"learnings": 0,
	}
	data, err := os.ReadFile(filepath.Join(chamberDir, "COLONY_STATE.json"))
	if err != nil {
		return summary
	}
	var state colony.ColonyState
	if err := json.Unmarshal(data, &state); err != nil {
		return summary
	}
	learnings := 0
	for _, phase := range state.Memory.PhaseLearnings {
		learnings += len(phase.Learnings)
	}
	summary["decisions"] = len(state.Memory.Decisions)
	summary["learnings"] = learnings
	return summary
}

func daysBetweenChambers(left, right string) int {
	leftTime, leftErr := time.Parse(time.RFC3339, strings.TrimSpace(left))
	rightTime, rightErr := time.Parse(time.RFC3339, strings.TrimSpace(right))
	if leftErr != nil || rightErr != nil {
		return 0
	}
	diff := rightTime.Sub(leftTime)
	if diff < 0 {
		diff = -diff
	}
	return int(diff.Hours() / 24)
}

func importSignalsFromChamber(chamberName, chamberDir string) (map[string]interface{}, error) {
	if store == nil {
		return nil, fmt.Errorf("no colony initialized")
	}
	archivePath := filepath.Join(chamberDir, "colony-archive.xml")
	data, err := os.ReadFile(archivePath)
	if err != nil {
		return nil, fmt.Errorf("chamber %q has no colony-archive.xml", chamberName)
	}
	pheromoneXML, err := extractPheromonesFromArchiveXML(data)
	if err != nil {
		return nil, err
	}
	result, err := importPheromonesData(filepath.ToSlash(archivePath), pheromoneXML, chamberName)
	if err != nil {
		return nil, err
	}
	result["mode"] = "import"
	result["chamber"] = chamberName
	result["archive"] = filepath.ToSlash(archivePath)
	return result, nil
}

func extractPheromonesFromArchiveXML(data []byte) ([]byte, error) {
	var archive exchange.ColonyArchiveXML
	if err := xml.Unmarshal(data, &archive); err != nil {
		return nil, fmt.Errorf("parse colony archive XML: %w", err)
	}
	if archive.Pheromones == nil {
		return nil, fmt.Errorf("archive has no pheromones section")
	}
	pheromoneData, err := xml.MarshalIndent(archive.Pheromones, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal pheromones XML: %w", err)
	}
	return append([]byte(xml.Header), pheromoneData...), nil
}

func uniqueChamberName(chambersRoot string, scope colony.ColonyScope, goal string, now time.Time) string {
	prefix := now.Format("2006-01-02")
	scopeSlug := string(scope.Effective())
	slug := sanitizeChamberGoal(goal)
	name := prefix + "-" + scopeSlug + "-" + slug
	candidate := name
	counter := 1
	for {
		if _, err := os.Stat(filepath.Join(chambersRoot, candidate)); os.IsNotExist(err) {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d", name, counter)
		counter++
	}
}

func sanitizeChamberGoal(goal string) string {
	goal = strings.ToLower(strings.TrimSpace(goal))
	if goal == "" {
		return "colony"
	}
	var b strings.Builder
	lastDash := false
	for _, r := range goal {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		slug = "colony"
	}
	if len(slug) > 40 {
		slug = strings.Trim(slug[:40], "-")
	}
	if slug == "" {
		return "colony"
	}
	return slug
}

func writeEntombManifest(chamberDir, chamberName string, state colony.ColonyState, now time.Time) error {
	goal := ""
	if state.Goal != nil {
		goal = strings.TrimSpace(*state.Goal)
	}
	manifest := map[string]interface{}{
		"name":             chamberName,
		"goal":             goal,
		"scope":            string(state.EffectiveScope()),
		"milestone":        state.Milestone,
		"phases_completed": completedPhaseCount(state),
		"total_phases":     len(state.Plan.Phases),
		"entombed_at":      now.Format(time.RFC3339),
		"colony_version":   state.ColonyVersion,
		"state":            state.State,
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(chamberDir, "manifest.json"), append(data, '\n'), 0644)
}

func copyEntombArtifacts(aetherRoot, dataDir, chamberDir string) error {
	dataFiles := []string{
		"COLONY_STATE.json",
		"pheromones.json",
		"session.json",
		"activity.log",
		"flags.json",
		"constraints.json",
		"spawn-tree.txt",
		"spawn-runs.json",
		"timing.log",
		"view-state.json",
		"midden.json",
	}
	for _, name := range dataFiles {
		if err := copyIfExists(filepath.Join(dataDir, name), filepath.Join(chamberDir, name)); err != nil {
			return err
		}
	}

	rootFiles := []string{
		filepath.Join(aetherRoot, ".aether", "CROWNED-ANTHILL.md"),
		filepath.Join(aetherRoot, ".aether", "HANDOFF.md"),
		filepath.Join(aetherRoot, ".aether", "CONTEXT.md"),
	}
	for _, src := range rootFiles {
		if err := copyIfExists(src, filepath.Join(chamberDir, filepath.Base(src))); err != nil {
			return err
		}
	}

	if err := copyDirIfExists(filepath.Join(aetherRoot, ".aether", "dreams"), filepath.Join(chamberDir, "dreams")); err != nil {
		return err
	}
	if err := copyDirIfExists(filepath.Join(dataDir, "colonies"), filepath.Join(chamberDir, "colonies")); err != nil {
		return err
	}
	if err := copyDirIfExists(filepath.Join(dataDir, "reviews"), filepath.Join(chamberDir, "reviews")); err != nil {
		return err
	}

	xmlMatches, _ := filepath.Glob(filepath.Join(aetherRoot, ".aether", "exchange", "*.xml"))
	for _, src := range xmlMatches {
		if err := copyIfExists(src, filepath.Join(chamberDir, filepath.Base(src))); err != nil {
			return err
		}
	}

	return nil
}

func copyIfExists(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.IsDir() {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func copyDirIfExists(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return nil
	}
	return filepath.Walk(src, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		return copyIfExists(path, target)
	})
}

func exportArchiveXMLToFile(outputPath string) error {
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	oldStdout := stdout
	oldStderr := stderr
	stdout = &stdoutBuf
	stderr = &stderrBuf
	defer func() {
		stdout = oldStdout
		stderr = oldStderr
	}()

	tmpCmd := &cobra.Command{}
	tmpCmd.Flags().String("output", outputPath, "")
	if err := runExportArchive(tmpCmd, nil); err != nil {
		return err
	}
	if _, err := os.Stat(outputPath); err != nil {
		msg := strings.TrimSpace(stderrBuf.String())
		if msg == "" {
			msg = strings.TrimSpace(stdoutBuf.String())
		}
		if msg == "" {
			msg = errString(err)
		}
		return fmt.Errorf("%s", strings.TrimSpace(msg))
	}
	return nil
}

func verifyEntombedChamber(chamberDir string) error {
	required := []string{
		filepath.Join(chamberDir, "manifest.json"),
		filepath.Join(chamberDir, "COLONY_STATE.json"),
		filepath.Join(chamberDir, "CROWNED-ANTHILL.md"),
		filepath.Join(chamberDir, "colony-archive.xml"),
	}
	for _, requiredPath := range required {
		if _, err := os.Stat(requiredPath); err != nil {
			return fmt.Errorf("missing %s", filepath.Base(requiredPath))
		}
	}
	return nil
}

func resetColonyStateForEntomb(state colony.ColonyState) colony.ColonyState {
	state.Goal = nil
	state.Charter = nil
	state.AcceptedCharter = nil
	state.Scope = ""
	state.ColonyMode = ""
	state.ColonyName = nil
	state.ColonyVersion = 0
	state.State = colony.StateIDLE
	state.CurrentPhase = 0
	state.SessionID = nil
	state.InitializedAt = nil
	state.BuildStartedAt = nil
	// Whole-struct replacement (rather than clearing named fields one at a
	// time) is deliberate: it is what makes a future field added to
	// colony.Plan and never wired into this reset fail
	// TestEntombResetLeavesNoPlanAuthority by name instead of silently
	// surviving into an archived shell.
	state.Plan = colony.Plan{Phases: []colony.Phase{}}
	state.Specification = nil
	state.Memory.PhaseLearnings = []colony.PhaseLearning{}
	state.Memory.Decisions = []colony.Decision{}
	state.Memory.Instincts = []colony.Instinct{}
	state.Errors.Records = []colony.ErrorRecord{}
	state.Errors.FlaggedPatterns = []colony.FlaggedPattern{}
	state.Signals = []colony.Signal{}
	state.Graveyards = []colony.Graveyard{}
	state.Events = []string{}
	state.Milestone = ""
	state.MilestoneUpdatedAt = nil
	state.Paused = false
	state.PausedAt = nil
	state.Worktrees = nil
	state.RunID = nil
	state.GateResults = nil
	state.TerritorySurveyed = nil
	state.PendingSuggestions = nil
	state.LastAnalyzeCommit = nil
	state.LifecycleReceipt = nil
	state.PauseHandoff = nil
	state.RecoveryProvenance = nil
	state.SealOutcome = nil
	state.ArchiveReference = nil
	state.ResearchDocs = nil
	return state
}

func clearActiveColonyRuntimeFiles(aetherRoot, dataDir string) error {
	toRemove := []string{
		filepath.Join(dataDir, "session.json"),
		filepath.Join(dataDir, "pheromones.json"),
		filepath.Join(dataDir, "activity.log"),
		filepath.Join(dataDir, "flags.json"),
		filepath.Join(dataDir, "constraints.json"),
		filepath.Join(dataDir, "spawn-tree.txt"),
		filepath.Join(dataDir, "spawn-runs.json"),
		filepath.Join(dataDir, "timing.log"),
		filepath.Join(dataDir, "view-state.json"),
		filepath.Join(dataDir, ".version-check-cache"),
		filepath.Join(aetherRoot, ".aether", "CROWNED-ANTHILL.md"),
		filepath.Join(aetherRoot, ".aether", "CONTEXT.md"),
	}
	for _, path := range toRemove {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	if err := os.RemoveAll(filepath.Join(dataDir, "colonies")); err != nil {
		return err
	}
	if err := os.RemoveAll(filepath.Join(dataDir, "reviews")); err != nil {
		return err
	}

	// Clean up the worktrees directory and any tracked worktree entries --
	// but never at the cost of uncommitted or unmerged worker output
	// (187-VERIFICATION.md GAP-5). Both callers (entomb, abandon) have
	// already reset COLONY_STATE.json's Worktrees field to nil by the time
	// this runs, so a state-only check would see "nothing tracked" and wipe
	// unconditionally -- the same blind spot GAP-3 closed for `init`. Scan
	// the directory on disk directly instead, exactly like init's fix does,
	// and leave the directory in place if anything on it still holds work.
	worktreesDir := filepath.Join(aetherRoot, ".aether", "worktrees")
	unrecorded := scanUnrecordedWorktrees(aetherRoot, worktreesDir, map[string]bool{})
	var unsafe []worktreeSafety
	for _, safety := range unrecorded {
		if !safety.Safe {
			unsafe = append(unsafe, safety)
		}
	}
	if len(unsafe) > 0 {
		for _, safety := range unsafe {
			reportWorktreePreservation(safety, fmt.Sprintf("found in a worker workspace that was about to be cleared: %s", safety.Reason))
		}
		fmt.Fprintf(os.Stderr, "some worker workspaces still hold work, so they were left in place instead of being cleared -- inspect them with `aether maintenance recovery-inspect` (State effect: none); restore runnable lifecycle state with `aether resume`\n")
		return nil
	}
	if err := os.RemoveAll(worktreesDir); err != nil {
		return err
	}
	return nil
}

func writeEntombRecoveryDocs(chamberName, goal string, state colony.ColonyState, now time.Time) error {
	completed := completedPhaseCount(state)
	total := len(state.Plan.Phases)
	scope := string(state.EffectiveScope())

	handoff := strings.Join([]string{
		"# Colony Session — " + chamberName,
		"",
		"## A Colony's Rest",
		"",
		"This colony has been entombed. Its work is complete and archived.",
		"",
		"**Chamber:** .aether/chambers/" + chamberName + "/",
		"",
		"## Colony Summary",
		"",
		"- Goal: \"" + goal + "\"",
		"- Scope: " + scope,
		fmt.Sprintf("- Phases: %d completed of %d", completed, total),
		"- Milestone reached: Crowned Anthill",
		"- Entombed at: " + now.Format(time.RFC3339),
		"",
		"When you are ready to begin again:",
		"",
		"- Start fresh: `aether init \"new goal\"`",
		"- Browse archives: `aether tunnels`",
		"",
	}, "\n")
	if err := writeHandoffDocument(handoff); err != nil {
		return err
	}

	context := strings.Join([]string{
		"# Colony Context",
		"",
		"State: IDLE",
		"Latest chamber: .aether/chambers/" + chamberName + "/",
		"Previous goal: " + goal,
		"Previous scope: " + scope,
		"Recommended next action: `aether init \"new goal\"`",
		"",
	}, "\n")
	return writeContextDocument(context)
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// extractNearMissInstincts returns instincts with confidence in the near-miss
// range [0.5, 0.8). These are worth preserving in the chamber archive but not
// yet high-confidence enough for hive promotion.
func extractNearMissInstincts(state colony.ColonyState) []colony.Instinct {
	var nearMiss []colony.Instinct
	for _, inst := range state.Memory.Instincts {
		if inst.Confidence >= 0.5 && inst.Confidence < 0.8 {
			nearMiss = append(nearMiss, inst)
		}
	}
	return nearMiss
}

// entombTempSweep performs cleanup of stale data files that should not persist
// after entomb. It runs AFTER copyEntombArtifacts so the chamber copy is preserved.
// Returns the count of items cleaned.
func entombTempSweep(dataDir string, s *storage.Store) int {
	cleaned := 0

	// 1. Prune expired pheromones (Active=false and Strength=0 or nil)
	var pf colony.PheromoneFile
	if err := s.LoadJSON("pheromones.json", &pf); err == nil && len(pf.Signals) > 0 {
		var kept []colony.PheromoneSignal
		for _, sig := range pf.Signals {
			if !sig.Active && (sig.Strength == nil || *sig.Strength == 0) {
				cleaned++
				continue
			}
			kept = append(kept, sig)
		}
		if cleaned > 0 {
			pf.Signals = kept
			_ = s.SaveJSON("pheromones.json", pf)
		}
	}

	// 2. Prune midden entries older than 30 days
	var mf colony.MiddenFile
	if err := s.LoadJSON("midden.json", &mf); err == nil && len(mf.Entries) > 0 {
		cutoff := time.Now().Add(-30 * 24 * time.Hour)
		var kept []colony.MiddenEntry
		for _, entry := range mf.Entries {
			ts, err := time.Parse(time.RFC3339, entry.Timestamp)
			if err != nil || ts.After(cutoff) {
				kept = append(kept, entry) // keep if parse error or recent
				continue
			}
			cleaned++
		}
		if len(kept) < len(mf.Entries) {
			mf.Entries = kept
			_ = s.SaveJSON("midden.json", mf)
		}
	}

	// 3. Clean old session snapshots (backups directory)
	backupDir := filepath.Join(dataDir, "backups")
	if entries, err := os.ReadDir(backupDir); err == nil {
		cutoff := time.Now().Add(-7 * 24 * time.Hour)
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			if info.ModTime().Before(cutoff) {
				_ = os.Remove(filepath.Join(backupDir, entry.Name()))
				cleaned++
			}
		}
	}

	return cleaned
}

// countTotalPlans counts total plans across all phases in colony state.
// Since Phase struct doesn't have PlanCount, we count completed phases as proxy.
func countTotalPlans(state colony.ColonyState) int {
	return completedPhaseCount(state)
}

// computeColonyDuration returns a human-readable duration string since colony init.
func computeColonyDuration(state colony.ColonyState) string {
	if state.InitializedAt == nil {
		return ""
	}
	duration := time.Since(*state.InitializedAt)
	days := int(duration.Hours() / 24)
	if days > 0 {
		return fmt.Sprintf("%dd", days)
	}
	return fmt.Sprintf("%.0fh", duration.Hours())
}

func renderEntombVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner("⚰️", "Entomb"))
	b.WriteString(visualDividerStr())
	b.WriteString("Colony archived and active state cleared after verification.\n")
	b.WriteString("Goal: ")
	b.WriteString(emptyFallback(stringValue(result["goal"]), "No goal recorded"))
	b.WriteString("\n")
	b.WriteString("Scope: ")
	b.WriteString(emptyFallback(stringValue(result["scope"]), string(colony.ScopeProject)))
	b.WriteString("\n")
	b.WriteString("Chamber: ")
	b.WriteString(emptyFallback(stringValue(result["chamber_path"]), stringValue(result["chamber"])))
	b.WriteString("\n")
	b.WriteString("Manifest digest: ")
	b.WriteString(stringValue(result["manifest_digest"]))
	b.WriteString("\nTransaction receipt: ")
	b.WriteString(stringValue(result["transaction_receipt"]))
	b.WriteString("\nSeal disposition: ")
	b.WriteString(stringValue(result["seal_disposition"]))
	b.WriteString("\n")
	if reason := strings.TrimSpace(stringValue(result["owner_reason"])); reason != "" {
		b.WriteString("Owner reason: ")
		b.WriteString(reason)
		b.WriteString("\n")
	}
	b.WriteString("Retained: colony memory, seal evidence, and tombstone input\n\n")
	if stages, ok := result["stages"].([]string); ok {
		for _, stage := range stages {
			b.WriteString("── ")
			b.WriteString(stage)
			b.WriteString(" ──\n")
		}
	}
	return appendLifecycleCloseoutVisual(b.String(), result, "codex")
}

func init() {
	entombCmd.Flags().Bool("confirm", false, "Archive the verified chamber and clear active colony state")
	tunnelsCmd.Flags().Bool("import-signals", false, "Import pheromone signals from the chamber's colony-archive.xml")
	rootCmd.AddCommand(entombCmd)
	rootCmd.AddCommand(tunnelsCmd)
}
