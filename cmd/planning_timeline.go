package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

const (
	planningTimelineIndexSchemaVersion  = "planning-timeline-index/v1"
	planningTimelineAppendSchemaVersion = "planning-timeline-append/v1"
	planningTimelineIndexFileName       = "timeline.json"
)

// planningTimelineIndex is the small ordered manifest for immutable card
// artifacts. The cards retain the detail; this file retains only locators and
// the hashes needed to prove their complete predecessor chain.
type planningTimelineIndex struct {
	SchemaVersion  string                       `json:"schema_version"`
	ID             string                       `json:"id"`
	ContentHash    string                       `json:"content_hash"`
	RunID          string                       `json:"run_id"`
	Entries        []planningTimelineIndexEntry `json:"entries"`
	FirstCardHash  string                       `json:"first_card_hash"`
	LastCardHash   string                       `json:"last_card_hash"`
	TimelineDigest string                       `json:"timeline_digest"`
}

type planningTimelineIndexEntry struct {
	Iteration           int    `json:"iteration"`
	CardID              string `json:"card_id"`
	CardHash            string `json:"card_hash"`
	PreviousCardHash    string `json:"previous_card_hash,omitempty"`
	CardPath            string `json:"card_path"`
	AppendReceiptID     string `json:"append_receipt_id"`
	AppendRequestDigest string `json:"append_request_digest"`
	TransactionID       string `json:"transaction_id"`
}

// planningTimelineAppendOptions carries the exact replay identity and testable
// transaction seams. PreviousCardHash is empty only for the first card.
type planningTimelineAppendOptions struct {
	PreviousCardHash string
	ReceiptID        string
	Fault            lifecycleTransactionFaultHook
	Rename           func(oldPath, newPath string) error
}

type planningTimelineAppendReceipt struct {
	SchemaVersion  string                  `json:"schema_version"`
	ReceiptID      string                  `json:"receipt_id"`
	RequestDigest  string                  `json:"request_digest"`
	RunID          string                  `json:"run_id"`
	Iteration      int                     `json:"iteration"`
	CardID         string                  `json:"card_id"`
	CardHash       string                  `json:"card_hash"`
	CardPath       string                  `json:"card_path"`
	IndexPath      string                  `json:"index_path"`
	TimelineDigest string                  `json:"timeline_digest"`
	WriteReceipt   colony.LifecycleReceipt `json:"write_receipt"`
}

type planningTimelineClassification string

const (
	planningTimelineCandidateOnly planningTimelineClassification = "candidate_only"
	planningTimelineAccepted      planningTimelineClassification = "accepted"
	planningTimelineLegacyUnbound planningTimelineClassification = "legacy_unbound"
)

type planningTimeline struct {
	Classification  planningTimelineClassification  `json:"classification"`
	RunID           string                          `json:"run_id"`
	Index           *planningTimelineIndex          `json:"index,omitempty"`
	Cards           []colony.PlanningIterationCard  `json:"cards,omitempty"`
	Binding         *colony.PlanningTimelineBinding `json:"binding,omitempty"`
	LegacyArtifacts []string                        `json:"legacy_artifacts,omitempty"`
}

type planningTimelineProtection struct {
	Classification  planningTimelineClassification `json:"classification"`
	BoundRevisionID string                         `json:"bound_revision_id,omitempty"`
	Prunable        bool                           `json:"prunable"`
	Reorderable     bool                           `json:"reorderable"`
}

type planningTimelineMutation string

const (
	planningTimelineMutationPrune   planningTimelineMutation = "prune"
	planningTimelineMutationReorder planningTimelineMutation = "reorder"
)

// planningTimelineConflictError distinguishes replay/binding conflicts from
// validation errors so callers can fail closed without guessing from prose.
type planningTimelineConflictError struct {
	ReceiptID string
	Detail    string
}

func (err *planningTimelineConflictError) Error() string {
	if strings.TrimSpace(err.ReceiptID) == "" {
		return "planning timeline conflict: " + err.Detail
	}
	return fmt.Sprintf("planning timeline receipt %q conflicts: %s", err.ReceiptID, err.Detail)
}

type planningTimelineProtectedError struct {
	TimelineID string
	RevisionID string
	Mutation   planningTimelineMutation
}

func (err *planningTimelineProtectedError) Error() string {
	return fmt.Sprintf("planning timeline %q is bound to accepted plan revision %q and cannot apply %q", err.TimelineID, err.RevisionID, err.Mutation)
}

// appendPlanningIterationCard validates a complete Route-Setter pass, freezes
// its canonical bytes, and commits the card plus its ordered index through one
// lifecycle transaction. It never writes renderer text into planning history.
func appendPlanningIterationCard(root string, card colony.PlanningIterationCard, opts planningTimelineAppendOptions) (planningTimelineAppendReceipt, error) {
	empty := planningTimelineAppendReceipt{}
	repositoryRoot, err := canonicalPlanningTimelineRoot(root)
	if err != nil {
		return empty, err
	}
	if err := validatePlanningTimelineSegment("run_id", card.RunID); err != nil {
		return empty, err
	}
	if err := validatePlanningTimelineSegment("receipt_id", opts.ReceiptID); err != nil {
		return empty, err
	}
	canonicalCard, cardBytes, err := canonicalPlanningTimelineCard(card)
	if err != nil {
		return empty, fmt.Errorf("planning timeline card: %w", err)
	}

	index, cards, exists, err := readPlanningTimelineChain(repositoryRoot, canonicalCard.RunID)
	if err != nil {
		return empty, err
	}
	cardPath := planningTimelineCardRepositoryPath(canonicalCard.RunID, canonicalCard.Iteration, canonicalCard.ID)
	indexPath := planningTimelineIndexRepositoryPath(canonicalCard.RunID)
	requestDigest, err := planningTimelineAppendRequestDigest(opts.ReceiptID, canonicalCard, opts.PreviousCardHash)
	if err != nil {
		return empty, err
	}
	transactionID, err := planningTimelineTransactionID(canonicalCard.RunID, opts.ReceiptID)
	if err != nil {
		return empty, err
	}
	for entryIndex, entry := range index.Entries {
		if entry.AppendReceiptID == opts.ReceiptID {
			if entry.AppendRequestDigest != requestDigest || entry.CardID != canonicalCard.ID || entry.CardHash != canonicalCard.ContentHash || entry.PreviousCardHash != opts.PreviousCardHash {
				return empty, &planningTimelineConflictError{ReceiptID: opts.ReceiptID, Detail: "the replay payload differs from the completed append"}
			}
			replayTimelineDigest, err := planningTimelineDigest(cards[:entryIndex+1])
			if err != nil {
				return empty, fmt.Errorf("hash replayed planning timeline prefix: %w", err)
			}
			writeReceipt, err := loadPlanningTimelineWriteReceipt(repositoryRoot, entry.TransactionID)
			if err != nil {
				return empty, err
			}
			return planningTimelineAppendReceiptFor(index, index.Entries[entryIndex], replayTimelineDigest, writeReceipt), nil
		}
		if entry.CardID == canonicalCard.ID || entry.Iteration == canonicalCard.Iteration {
			return empty, &planningTimelineConflictError{ReceiptID: opts.ReceiptID, Detail: fmt.Sprintf("iteration %d or card %q is already bound to another receipt", canonicalCard.Iteration, canonicalCard.ID)}
		}
	}
	expectedIteration := len(cards) + 1
	if canonicalCard.Iteration != expectedIteration {
		return empty, fmt.Errorf("planning timeline iteration %d must be the next ordinal %d", canonicalCard.Iteration, expectedIteration)
	}
	expectedPreviousHash := ""
	if len(cards) > 0 {
		expectedPreviousHash = cards[len(cards)-1].ContentHash
	}
	if opts.PreviousCardHash != expectedPreviousHash {
		return empty, fmt.Errorf("planning timeline previous_card_hash %q must match %q", opts.PreviousCardHash, expectedPreviousHash)
	}
	var legacyArtifacts []string
	if !exists {
		legacyArtifacts, err = planningTimelineLegacyArtifacts(repositoryRoot, canonicalCard.RunID)
		if err != nil {
			return empty, err
		}
		index = planningTimelineIndex{SchemaVersion: planningTimelineIndexSchemaVersion, RunID: canonicalCard.RunID}
	}

	index.Entries = append(index.Entries, planningTimelineIndexEntry{
		Iteration:           canonicalCard.Iteration,
		CardID:              canonicalCard.ID,
		CardHash:            canonicalCard.ContentHash,
		PreviousCardHash:    opts.PreviousCardHash,
		CardPath:            cardPath,
		AppendReceiptID:     opts.ReceiptID,
		AppendRequestDigest: requestDigest,
		TransactionID:       transactionID,
	})
	cards = append(cards, canonicalCard)
	if err := addressPlanningTimelineIndex(&index, cards); err != nil {
		return empty, err
	}
	indexBytes, err := marshalPlanningTimelineJSON(index)
	if err != nil {
		return empty, fmt.Errorf("marshal planning timeline index: %w", err)
	}
	if err := validatePlanningTimelineCardBytes(cardBytes, canonicalCard); err != nil {
		return empty, err
	}
	if err := validatePlanningTimelineIndexBytes(indexBytes, index, cards); err != nil {
		return empty, err
	}

	dataRoot := filepath.Join(repositoryRoot, ".aether", "data")
	if err := os.MkdirAll(dataRoot, 0o755); err != nil {
		return empty, fmt.Errorf("create planning lifecycle data root: %w", err)
	}
	config := lifecycleTransactionConfig{
		TransactionID: transactionID,
		Command:       "planning-timeline-append",
		Allowlist: lifecycleTransactionAllowlist{
			RepositoryRoot:    repositoryRoot,
			LifecycleDataRoot: dataRoot,
		},
		Fault:  opts.Fault,
		Rename: opts.Rename,
	}
	if pending, err := planningTimelineTransactionHasIntent(config); err != nil {
		return empty, err
	} else if pending {
		matches, err := planningTimelinePendingIntentMatches(config, map[string][]byte{
			planningTimelineDataRelativePath(cardPath):  cardBytes,
			planningTimelineDataRelativePath(indexPath): indexBytes,
		})
		if err != nil {
			return empty, err
		}
		if !matches {
			return empty, &planningTimelineConflictError{ReceiptID: opts.ReceiptID, Detail: "the durable staged append has different card or index content"}
		}
		writeReceipt, err := resumeLifecycleTransaction(config)
		if err != nil {
			return empty, err
		}
		return planningTimelineAppendReceiptFor(index, index.Entries[len(index.Entries)-1], index.TimelineDigest, writeReceipt), nil
	}
	if len(legacyArtifacts) > 0 {
		return empty, fmt.Errorf("planning timeline has legacy_unbound iteration evidence and cannot append a current chain")
	}
	if _, statErr := os.Lstat(filepath.Join(repositoryRoot, filepath.FromSlash(cardPath))); statErr == nil {
		return empty, &planningTimelineConflictError{ReceiptID: opts.ReceiptID, Detail: fmt.Sprintf("card path %q exists outside the timeline index", cardPath)}
	} else if !os.IsNotExist(statErr) {
		return empty, fmt.Errorf("inspect planning timeline card path %q: %w", cardPath, statErr)
	}
	tx, err := beginLifecycleTransaction(config)
	if err != nil {
		return empty, err
	}
	if err := tx.DeclareWrite(lifecycleTransactionRootData, planningTimelineDataRelativePath(cardPath), cardBytes); err != nil {
		return empty, err
	}
	if err := tx.DeclareWrite(lifecycleTransactionRootData, planningTimelineDataRelativePath(indexPath), indexBytes); err != nil {
		return empty, err
	}
	if err := tx.Validate(); err != nil {
		return empty, err
	}
	writeReceipt, err := tx.Commit()
	if err != nil {
		return empty, err
	}
	return planningTimelineAppendReceiptFor(index, index.Entries[len(index.Entries)-1], index.TimelineDigest, writeReceipt), nil
}

func planningTimelineAppendReceiptFor(index planningTimelineIndex, entry planningTimelineIndexEntry, timelineDigest string, writeReceipt colony.LifecycleReceipt) planningTimelineAppendReceipt {
	return planningTimelineAppendReceipt{
		SchemaVersion:  planningTimelineAppendSchemaVersion,
		ReceiptID:      entry.AppendReceiptID,
		RequestDigest:  entry.AppendRequestDigest,
		RunID:          index.RunID,
		Iteration:      entry.Iteration,
		CardID:         entry.CardID,
		CardHash:       entry.CardHash,
		CardPath:       entry.CardPath,
		IndexPath:      planningTimelineIndexRepositoryPath(index.RunID),
		TimelineDigest: timelineDigest,
		WriteReceipt:   writeReceipt,
	}
}

func loadPlanningTimelineWriteReceipt(root, transactionID string) (colony.LifecycleReceipt, error) {
	config := planningTimelineLifecycleConfig(root, transactionID)
	tx, err := beginLifecycleTransaction(config)
	if err != nil {
		return colony.LifecycleReceipt{}, err
	}
	lifecycleTransactionProcessMu.Lock()
	receipt, ok, receiptErr := loadPlanningTimelineHistoricalReceipt(tx)
	lifecycleTransactionProcessMu.Unlock()
	if ok || receiptErr != nil {
		if receiptErr != nil {
			return colony.LifecycleReceipt{}, receiptErr
		}
		return receipt, nil
	}
	pending, err := planningTimelineTransactionHasIntent(config)
	if err != nil {
		return colony.LifecycleReceipt{}, err
	}
	if !pending {
		return colony.LifecycleReceipt{}, fmt.Errorf("planning timeline transaction %q has no durable receipt or intent", transactionID)
	}
	receipt, err = resumeLifecycleTransaction(config)
	if err != nil {
		return colony.LifecycleReceipt{}, err
	}
	if receipt.StateEffect != colony.LifecycleStateEffectCommitted {
		return colony.LifecycleReceipt{}, fmt.Errorf("planning timeline transaction %q did not commit", transactionID)
	}
	return receipt, nil
}

// loadPlanningTimelineHistoricalReceipt validates immutable journal evidence
// without requiring the live index to equal this append's former index bytes.
// Later appends legitimately advance that one shared target; load of the full
// current chain separately proves that the historical prefix still exists.
func loadPlanningTimelineHistoricalReceipt(tx *lifecycleTransaction) (colony.LifecycleReceipt, bool, error) {
	receiptPath := filepath.Join(tx.journalPath(), "receipt.json")
	receiptBytes, err := readLifecycleEvidenceFile(receiptPath)
	if os.IsNotExist(err) {
		return colony.LifecycleReceipt{}, false, nil
	}
	if err != nil {
		return colony.LifecycleReceipt{}, false, fmt.Errorf("planning timeline: read historical receipt: %w", err)
	}
	digestBytes, err := readLifecycleEvidenceFile(filepath.Join(tx.journalPath(), "receipt.sha256"))
	if err != nil {
		return colony.LifecycleReceipt{}, false, fmt.Errorf("planning timeline: historical receipt evidence is incomplete: %w", err)
	}
	if strings.TrimSpace(string(digestBytes)) != lifecycleDigest(receiptBytes) {
		return colony.LifecycleReceipt{}, false, fmt.Errorf("planning timeline: historical receipt digest conflicts with receipt bytes")
	}
	var receipt colony.LifecycleReceipt
	if err := decodeLifecycleJSON(receiptBytes, &receipt); err != nil {
		return colony.LifecycleReceipt{}, false, fmt.Errorf("planning timeline: decode historical receipt: %w", err)
	}
	if err := receipt.Validate(); err != nil {
		return colony.LifecycleReceipt{}, false, fmt.Errorf("planning timeline: invalid historical receipt: %w", err)
	}
	if receipt.Transaction.ID != tx.config.TransactionID || receipt.Command != tx.config.Command || receipt.StateEffect != colony.LifecycleStateEffectCommitted {
		return colony.LifecycleReceipt{}, false, fmt.Errorf("planning timeline: historical receipt identity or state conflicts with its transaction")
	}
	intent, progress, err := tx.loadJournal()
	if err != nil {
		return colony.LifecycleReceipt{}, false, fmt.Errorf("planning timeline: historical receipt journal evidence is invalid: %w", err)
	}
	tx.intent, tx.progress = intent, progress
	if _, err := tx.validateRecoveryEvidence(); err != nil {
		return colony.LifecycleReceipt{}, false, fmt.Errorf("planning timeline: historical receipt root evidence is invalid: %w", err)
	}
	if progress.Stage != colony.TransactionStageVerified || progress.StateEffect != colony.LifecycleStateEffectCommitted {
		return colony.LifecycleReceipt{}, false, fmt.Errorf("planning timeline: historical receipt conflicts with coordinator stage %q", progress.Stage)
	}
	if progress.Receipt != nil && (progress.Receipt.ID != receipt.ReceiptID || progress.Receipt.Digest != lifecycleDigest(receiptBytes)) {
		return colony.LifecycleReceipt{}, false, fmt.Errorf("planning timeline: historical receipt reference conflicts with receipt bytes")
	}
	if receipt.Command != intent.Record.Command || !equalLifecycleChanges(receipt.Changes, intent.Record.Changes) {
		return colony.LifecycleReceipt{}, false, fmt.Errorf("planning timeline: historical receipt claims conflict with coordinator intent")
	}
	return receipt, true, nil
}

func planningTimelineLifecycleConfig(root, transactionID string) lifecycleTransactionConfig {
	return lifecycleTransactionConfig{
		TransactionID: transactionID,
		Command:       "planning-timeline-append",
		Allowlist: lifecycleTransactionAllowlist{
			RepositoryRoot:    root,
			LifecycleDataRoot: filepath.Join(root, ".aether", "data"),
		},
	}
}

func planningTimelineTransactionHasIntent(config lifecycleTransactionConfig) (bool, error) {
	tx, err := beginLifecycleTransaction(config)
	if err != nil {
		return false, err
	}
	_, err = os.Lstat(filepath.Join(tx.journalPath(), "intent.json"))
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect planning timeline transaction intent: %w", err)
	}
	return true, nil
}

// planningTimelinePendingIntentMatches proves that an interrupted transaction
// was staged for these exact bytes before allowing the lifecycle reducer to
// resume it. A shared receipt ID can therefore never authorize new content.
func planningTimelinePendingIntentMatches(config lifecycleTransactionConfig, expected map[string][]byte) (bool, error) {
	tx, err := beginLifecycleTransaction(config)
	if err != nil {
		return false, err
	}
	intent, progress, err := tx.loadJournal()
	if err != nil {
		return false, err
	}
	tx.intent, tx.progress = intent, progress
	manifests, err := tx.validateRecoveryEvidence()
	if err != nil {
		return false, err
	}
	targets := flattenLifecycleManifestTargets(intent, manifests)
	if len(targets) != len(expected) {
		return false, nil
	}
	seen := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		content, ok := expected[filepath.ToSlash(target.RelativeTarget)]
		if !ok || target.Action != lifecycleTransactionWrite {
			return false, nil
		}
		wantTarget := filepath.Join(config.Allowlist.LifecycleDataRoot, filepath.FromSlash(target.RelativeTarget))
		if target.TargetPath != wantTarget || target.AfterDigest != lifecycleDigest(content) {
			return false, nil
		}
		if _, duplicate := seen[target.RelativeTarget]; duplicate {
			return false, nil
		}
		seen[target.RelativeTarget] = struct{}{}
	}
	return len(seen) == len(expected), nil
}

func canonicalPlanningTimelineRoot(root string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", fmt.Errorf("planning timeline repository root is required")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve planning timeline repository root: %w", err)
	}
	absolute = filepath.Clean(absolute)
	info, err := os.Lstat(absolute)
	if err != nil {
		return "", fmt.Errorf("inspect planning timeline repository root: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", fmt.Errorf("planning timeline repository root must be a real directory")
	}
	return absolute, nil
}

func validatePlanningTimelineSegment(field, value string) error {
	if value != strings.TrimSpace(value) || !lifecycleTransactionIDPattern.MatchString(value) {
		return fmt.Errorf("planning timeline %s %q must be a safe path segment", field, value)
	}
	return nil
}

func canonicalPlanningTimelineCard(card colony.PlanningIterationCard) (colony.PlanningIterationCard, []byte, error) {
	content, err := json.Marshal(card)
	if err != nil {
		return colony.PlanningIterationCard{}, nil, fmt.Errorf("clone card: %w", err)
	}
	var frozen colony.PlanningIterationCard
	if err := decodePlanningTimelineJSON(content, &frozen); err != nil {
		return colony.PlanningIterationCard{}, nil, fmt.Errorf("clone card: %w", err)
	}
	frozen.ID = ""
	frozen.ContentHash = ""
	digest, err := jsonSHA256(frozen)
	if err != nil {
		return colony.PlanningIterationCard{}, nil, fmt.Errorf("hash card: %w", err)
	}
	frozen.ContentHash = digest
	frozen.ID = "planning-iteration-" + digest[:12]
	if err := validatePlanningIterationCardShape(frozen); err != nil {
		return colony.PlanningIterationCard{}, nil, err
	}
	bytes, err := marshalPlanningTimelineJSON(frozen)
	if err != nil {
		return colony.PlanningIterationCard{}, nil, err
	}
	return frozen, bytes, nil
}

func planningTimelineAppendRequestDigest(receiptID string, card colony.PlanningIterationCard, previousCardHash string) (string, error) {
	return jsonSHA256(struct {
		ReceiptID        string `json:"receipt_id"`
		RunID            string `json:"run_id"`
		Iteration        int    `json:"iteration"`
		CardID           string `json:"card_id"`
		CardHash         string `json:"card_hash"`
		PreviousCardHash string `json:"previous_card_hash"`
	}{
		ReceiptID: receiptID, RunID: card.RunID, Iteration: card.Iteration,
		CardID: card.ID, CardHash: card.ContentHash, PreviousCardHash: previousCardHash,
	})
}

func planningTimelineTransactionID(runID, receiptID string) (string, error) {
	digest, err := jsonSHA256(struct {
		RunID     string `json:"run_id"`
		ReceiptID string `json:"receipt_id"`
	}{RunID: runID, ReceiptID: receiptID})
	if err != nil {
		return "", fmt.Errorf("hash planning timeline transaction identity: %w", err)
	}
	return "planning-timeline-" + digest[:24], nil
}

func planningTimelineIndexRepositoryPath(runID string) string {
	return path.Join(".aether", "data", "planning", runID, planningTimelineIndexFileName)
}

func planningTimelineCardRepositoryPath(runID string, iteration int, cardID string) string {
	return path.Join(".aether", "data", "planning", runID, "iterations", fmt.Sprintf("iteration-%04d-%s.json", iteration, cardID))
}

func planningTimelineDataRelativePath(repositoryPath string) string {
	return strings.TrimPrefix(repositoryPath, ".aether/data/")
}

func readPlanningTimelineChain(root, runID string) (planningTimelineIndex, []colony.PlanningIterationCard, bool, error) {
	indexPath := filepath.Join(root, filepath.FromSlash(planningTimelineIndexRepositoryPath(runID)))
	info, err := os.Lstat(indexPath)
	if os.IsNotExist(err) {
		return planningTimelineIndex{}, nil, false, nil
	}
	if err != nil {
		return planningTimelineIndex{}, nil, false, fmt.Errorf("inspect planning timeline index: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return planningTimelineIndex{}, nil, false, fmt.Errorf("planning timeline index must be a regular non-symlink file")
	}
	content, err := os.ReadFile(indexPath)
	if err != nil {
		return planningTimelineIndex{}, nil, false, fmt.Errorf("read planning timeline index: %w", err)
	}
	var index planningTimelineIndex
	if err := decodePlanningTimelineJSON(content, &index); err != nil {
		return planningTimelineIndex{}, nil, false, fmt.Errorf("decode planning timeline index: %w", err)
	}
	if index.RunID != runID {
		return planningTimelineIndex{}, nil, false, fmt.Errorf("planning timeline index run_id %q does not match requested run %q", index.RunID, runID)
	}
	cards := make([]colony.PlanningIterationCard, len(index.Entries))
	for i, entry := range index.Entries {
		if err := validatePlanningArtifactPath(entry.CardPath); err != nil {
			return planningTimelineIndex{}, nil, false, fmt.Errorf("planning timeline entries[%d].card_path: %w", i, err)
		}
		wantPath := planningTimelineCardRepositoryPath(runID, entry.Iteration, entry.CardID)
		if entry.CardPath != wantPath {
			return planningTimelineIndex{}, nil, false, fmt.Errorf("planning timeline entries[%d].card_path is not the canonical card locator", i)
		}
		cardPath := filepath.Join(root, filepath.FromSlash(entry.CardPath))
		cardInfo, statErr := os.Lstat(cardPath)
		if statErr != nil {
			return planningTimelineIndex{}, nil, false, fmt.Errorf("inspect planning timeline card %d: %w", i+1, statErr)
		}
		if cardInfo.Mode()&os.ModeSymlink != 0 || !cardInfo.Mode().IsRegular() {
			return planningTimelineIndex{}, nil, false, fmt.Errorf("planning timeline card %d must be a regular non-symlink file", i+1)
		}
		cardBytes, readErr := os.ReadFile(cardPath)
		if readErr != nil {
			return planningTimelineIndex{}, nil, false, fmt.Errorf("read planning timeline card %d: %w", i+1, readErr)
		}
		if err := decodePlanningTimelineJSON(cardBytes, &cards[i]); err != nil {
			return planningTimelineIndex{}, nil, false, fmt.Errorf("decode planning timeline card %d: %w", i+1, err)
		}
	}
	if err := validatePlanningTimelineIndex(index, cards); err != nil {
		return planningTimelineIndex{}, nil, false, err
	}
	return index, cards, true, nil
}

func planningTimelineLegacyArtifacts(root, runID string) ([]string, error) {
	directory := filepath.Join(root, ".aether", "data", "planning", runID, "iterations")
	info, err := os.Lstat(directory)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("inspect legacy planning iteration directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, fmt.Errorf("legacy planning iteration directory must be a real directory")
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("read legacy planning iteration directory: %w", err)
	}
	artifacts := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		entryInfo, err := os.Lstat(filepath.Join(directory, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("inspect legacy planning artifact %q: %w", entry.Name(), err)
		}
		if entryInfo.Mode()&os.ModeSymlink != 0 || !entryInfo.Mode().IsRegular() {
			return nil, fmt.Errorf("legacy planning artifact %q must be a regular non-symlink file", entry.Name())
		}
		artifacts = append(artifacts, path.Join(".aether", "data", "planning", runID, "iterations", entry.Name()))
	}
	sort.Strings(artifacts)
	return artifacts, nil
}

// loadPlanningTimeline returns a binding only after every indexed card, content
// address, predecessor, locator, and whole-chain digest has been revalidated.
// Older unindexed JSON remains inspectable but can never become accepted state.
func loadPlanningTimeline(root, runID string) (planningTimeline, error) {
	empty := planningTimeline{}
	repositoryRoot, err := canonicalPlanningTimelineRoot(root)
	if err != nil {
		return empty, err
	}
	if err := validatePlanningTimelineSegment("run_id", runID); err != nil {
		return empty, err
	}
	index, cards, exists, err := readPlanningTimelineChain(repositoryRoot, runID)
	if err != nil {
		return empty, err
	}
	if !exists {
		artifacts, err := planningTimelineLegacyArtifacts(repositoryRoot, runID)
		if err != nil {
			return empty, err
		}
		if len(artifacts) == 0 {
			return empty, fmt.Errorf("planning timeline %q: %w", runID, os.ErrNotExist)
		}
		return planningTimeline{
			Classification:  planningTimelineLegacyUnbound,
			RunID:           runID,
			LegacyArtifacts: artifacts,
		}, nil
	}
	binding, err := planningTimelineBindingFor(index, cards)
	if err != nil {
		return empty, err
	}
	indexCopy := index
	return planningTimeline{
		Classification: planningTimelineCandidateOnly,
		RunID:          runID,
		Index:          &indexCopy,
		Cards:          cards,
		Binding:        &binding,
	}, nil
}

func planningTimelineBindingFor(index planningTimelineIndex, cards []colony.PlanningIterationCard) (colony.PlanningTimelineBinding, error) {
	cardIDs := make([]string, len(cards))
	for i := range cards {
		cardIDs[i] = cards[i].ID
	}
	binding := colony.PlanningTimelineBinding{
		SchemaVersion:  colony.PlanningTimelineSchemaVersion,
		RunID:          index.RunID,
		CardIDs:        cardIDs,
		FirstCardHash:  index.FirstCardHash,
		LastCardHash:   index.LastCardHash,
		TimelineDigest: index.TimelineDigest,
		Path:           planningTimelineIndexRepositoryPath(index.RunID),
	}
	digest, err := planningTimelineBindingContentHash(binding)
	if err != nil {
		return colony.PlanningTimelineBinding{}, fmt.Errorf("hash planning timeline binding: %w", err)
	}
	binding.ContentHash = digest
	binding.ID = "planning-timeline-" + digest[:12]
	if err := validatePlanningTimelineBindingContent(binding, cards); err != nil {
		return colony.PlanningTimelineBinding{}, err
	}
	return binding, nil
}

func planningTimelineBindingContentHash(binding colony.PlanningTimelineBinding) (string, error) {
	binding.ID = ""
	binding.ContentHash = ""
	return jsonSHA256(binding)
}

func validatePlanningTimelineBindingContent(binding colony.PlanningTimelineBinding, cards []colony.PlanningIterationCard) error {
	if err := validatePlanningTimelineBinding(binding, cards); err != nil {
		return err
	}
	digest, err := planningTimelineBindingContentHash(binding)
	if err != nil {
		return err
	}
	if binding.ContentHash != digest || binding.ID != "planning-timeline-"+digest[:12] {
		return fmt.Errorf("planning timeline binding content address does not match its canonical payload")
	}
	return nil
}

func planningTimelineProtectionFor(timeline planningTimeline, revisions []colony.PlanRevision) (planningTimelineProtection, error) {
	if timeline.Classification == planningTimelineLegacyUnbound {
		if timeline.Binding != nil || timeline.Index != nil || len(timeline.Cards) != 0 || len(timeline.LegacyArtifacts) == 0 {
			return planningTimelineProtection{}, fmt.Errorf("legacy_unbound planning timeline has inconsistent evidence")
		}
		return planningTimelineProtection{
			Classification: planningTimelineLegacyUnbound,
			Prunable:       true,
			Reorderable:    false,
		}, nil
	}
	if timeline.Classification != planningTimelineCandidateOnly && timeline.Classification != planningTimelineAccepted {
		return planningTimelineProtection{}, fmt.Errorf("planning timeline classification %q is invalid", timeline.Classification)
	}
	if timeline.Index == nil || timeline.Binding == nil || len(timeline.Cards) == 0 {
		return planningTimelineProtection{}, fmt.Errorf("current planning timeline is incomplete")
	}
	if timeline.RunID != timeline.Binding.RunID || timeline.Index.RunID != timeline.RunID {
		return planningTimelineProtection{}, fmt.Errorf("planning timeline run binding is inconsistent")
	}
	if err := validatePlanningTimelineIndex(*timeline.Index, timeline.Cards); err != nil {
		return planningTimelineProtection{}, err
	}
	if err := validatePlanningTimelineBindingContent(*timeline.Binding, timeline.Cards); err != nil {
		return planningTimelineProtection{}, err
	}
	boundRevisionID := ""
	for _, revision := range revisions {
		if revision.PlanningTimelineID != timeline.Binding.ID {
			continue
		}
		if revision.PlanningTimelineDigest != timeline.Binding.TimelineDigest {
			return planningTimelineProtection{}, &planningTimelineConflictError{
				Detail: fmt.Sprintf("accepted plan revision %q binds timeline ID %q with a different digest", revision.ID, revision.PlanningTimelineID),
			}
		}
		if strings.TrimSpace(revision.ID) == "" {
			return planningTimelineProtection{}, &planningTimelineConflictError{Detail: "accepted timeline binding names a revision without an ID"}
		}
		if boundRevisionID == "" {
			boundRevisionID = revision.ID
		}
	}
	if boundRevisionID != "" {
		return planningTimelineProtection{
			Classification:  planningTimelineAccepted,
			BoundRevisionID: boundRevisionID,
			Prunable:        false,
			Reorderable:     false,
		}, nil
	}
	return planningTimelineProtection{
		Classification: planningTimelineCandidateOnly,
		Prunable:       true,
		Reorderable:    false,
	}, nil
}

func authorizePlanningTimelineMutation(timeline planningTimeline, revisions []colony.PlanRevision, mutation planningTimelineMutation) error {
	protection, err := planningTimelineProtectionFor(timeline, revisions)
	if err != nil {
		return err
	}
	if protection.Classification == planningTimelineAccepted {
		return &planningTimelineProtectedError{
			TimelineID: timeline.Binding.ID,
			RevisionID: protection.BoundRevisionID,
			Mutation:   mutation,
		}
	}
	switch mutation {
	case planningTimelineMutationPrune:
		if !protection.Prunable {
			return &planningTimelineProtectedError{Mutation: mutation}
		}
		return nil
	case planningTimelineMutationReorder:
		return &planningTimelineConflictError{Detail: "planning timelines are append-only and cannot be reordered"}
	default:
		return fmt.Errorf("planning timeline mutation %q is invalid", mutation)
	}
}

func addressPlanningTimelineIndex(index *planningTimelineIndex, cards []colony.PlanningIterationCard) error {
	if index == nil || len(cards) == 0 {
		return fmt.Errorf("planning timeline index requires at least one card")
	}
	digest, err := planningTimelineDigest(cards)
	if err != nil {
		return fmt.Errorf("hash planning timeline: %w", err)
	}
	index.SchemaVersion = planningTimelineIndexSchemaVersion
	index.FirstCardHash = cards[0].ContentHash
	index.LastCardHash = cards[len(cards)-1].ContentHash
	index.TimelineDigest = digest
	index.ID = ""
	index.ContentHash = ""
	contentHash, err := planningTimelineIndexContentHash(*index)
	if err != nil {
		return err
	}
	index.ContentHash = contentHash
	index.ID = "planning-timeline-index-" + contentHash[:12]
	return validatePlanningTimelineIndex(*index, cards)
}

func planningTimelineIndexContentHash(index planningTimelineIndex) (string, error) {
	index.ID = ""
	index.ContentHash = ""
	return jsonSHA256(index)
}

func validatePlanningTimelineIndex(index planningTimelineIndex, cards []colony.PlanningIterationCard) error {
	if index.SchemaVersion != planningTimelineIndexSchemaVersion {
		return fmt.Errorf("planning timeline index schema_version must be %q", planningTimelineIndexSchemaVersion)
	}
	if err := validatePlanningTimelineSegment("run_id", index.RunID); err != nil {
		return err
	}
	if len(index.Entries) == 0 || len(index.Entries) != len(cards) {
		return fmt.Errorf("planning timeline index entries must resolve to the complete card chain")
	}
	wantContentHash, err := planningTimelineIndexContentHash(index)
	if err != nil {
		return err
	}
	if index.ContentHash != wantContentHash || index.ID != "planning-timeline-index-"+wantContentHash[:12] {
		return fmt.Errorf("planning timeline index content address does not match its canonical payload")
	}
	seenReceipts := make(map[string]struct{}, len(index.Entries))
	previousHash := ""
	for i, entry := range index.Entries {
		if entry.Iteration != i+1 {
			return fmt.Errorf("planning timeline entries[%d].iteration must be contiguous from one", i)
		}
		if entry.PreviousCardHash != previousHash {
			return fmt.Errorf("planning timeline entries[%d].previous_card_hash does not match its predecessor", i)
		}
		if err := validateSHA256(fmt.Sprintf("entries[%d].card_hash", i), entry.CardHash); err != nil {
			return err
		}
		if err := validateSHA256(fmt.Sprintf("entries[%d].append_request_digest", i), entry.AppendRequestDigest); err != nil {
			return err
		}
		if err := validatePlanningTimelineSegment(fmt.Sprintf("entries[%d].append_receipt_id", i), entry.AppendReceiptID); err != nil {
			return err
		}
		if _, duplicate := seenReceipts[entry.AppendReceiptID]; duplicate {
			return fmt.Errorf("planning timeline append receipt %q is duplicated", entry.AppendReceiptID)
		}
		seenReceipts[entry.AppendReceiptID] = struct{}{}
		if !lifecycleTransactionIDPattern.MatchString(entry.TransactionID) {
			return fmt.Errorf("planning timeline entries[%d].transaction_id is invalid", i)
		}
		if err := validatePlanningArtifactPath(entry.CardPath); err != nil {
			return fmt.Errorf("planning timeline entries[%d].card_path: %w", i, err)
		}
		card := cards[i]
		if card.RunID != index.RunID || card.Iteration != entry.Iteration || card.ID != entry.CardID || card.ContentHash != entry.CardHash {
			return fmt.Errorf("planning timeline entries[%d] does not match its card", i)
		}
		wantRequestDigest, err := planningTimelineAppendRequestDigest(entry.AppendReceiptID, card, entry.PreviousCardHash)
		if err != nil {
			return fmt.Errorf("planning timeline entries[%d] request digest: %w", i, err)
		}
		if entry.AppendRequestDigest != wantRequestDigest {
			return fmt.Errorf("planning timeline entries[%d].append_request_digest does not match its card and receipt", i)
		}
		wantTransactionID, err := planningTimelineTransactionID(index.RunID, entry.AppendReceiptID)
		if err != nil {
			return fmt.Errorf("planning timeline entries[%d] transaction identity: %w", i, err)
		}
		if entry.TransactionID != wantTransactionID {
			return fmt.Errorf("planning timeline entries[%d].transaction_id does not match its run and receipt", i)
		}
		wantPath := planningTimelineCardRepositoryPath(index.RunID, entry.Iteration, entry.CardID)
		if entry.CardPath != wantPath {
			return fmt.Errorf("planning timeline entries[%d].card_path is not the canonical card locator", i)
		}
		canonical, _, err := canonicalPlanningTimelineCard(card)
		if err != nil {
			return fmt.Errorf("planning timeline cards[%d]: %w", i, err)
		}
		if canonical.ID != card.ID || canonical.ContentHash != card.ContentHash {
			return fmt.Errorf("planning timeline cards[%d] canonical payload was mutated", i)
		}
		previousHash = entry.CardHash
	}
	if index.FirstCardHash != cards[0].ContentHash || index.LastCardHash != cards[len(cards)-1].ContentHash {
		return fmt.Errorf("planning timeline first or last card hash does not match the chain")
	}
	digest, err := planningTimelineDigest(cards)
	if err != nil {
		return err
	}
	if index.TimelineDigest != digest {
		return fmt.Errorf("planning timeline digest does not match the ordered card chain")
	}
	return nil
}

func validatePlanningTimelineCardBytes(content []byte, want colony.PlanningIterationCard) error {
	var decoded colony.PlanningIterationCard
	if err := decodePlanningTimelineJSON(content, &decoded); err != nil {
		return fmt.Errorf("validate staged planning card: %w", err)
	}
	if !planningTimelineJSONEqual(decoded, want) {
		return fmt.Errorf("validate staged planning card: bytes do not preserve the canonical payload")
	}
	canonical, _, err := canonicalPlanningTimelineCard(decoded)
	if err != nil {
		return fmt.Errorf("validate staged planning card: %w", err)
	}
	if canonical.ID != decoded.ID || canonical.ContentHash != decoded.ContentHash {
		return fmt.Errorf("validate staged planning card: content address mismatch")
	}
	return nil
}

func validatePlanningTimelineIndexBytes(content []byte, want planningTimelineIndex, cards []colony.PlanningIterationCard) error {
	var decoded planningTimelineIndex
	if err := decodePlanningTimelineJSON(content, &decoded); err != nil {
		return fmt.Errorf("validate staged planning timeline index: %w", err)
	}
	if !planningTimelineJSONEqual(decoded, want) {
		return fmt.Errorf("validate staged planning timeline index: bytes do not preserve the canonical payload")
	}
	return validatePlanningTimelineIndex(decoded, cards)
}

func marshalPlanningTimelineJSON(value interface{}) ([]byte, error) {
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(content, '\n'), nil
}

func decodePlanningTimelineJSON(content []byte, destination interface{}) error {
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

func planningTimelineJSONEqual(left, right interface{}) bool {
	leftBytes, leftErr := json.Marshal(left)
	rightBytes, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftBytes, rightBytes)
}
