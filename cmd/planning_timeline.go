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
	if !exists {
		index = planningTimelineIndex{SchemaVersion: planningTimelineIndexSchemaVersion, RunID: canonicalCard.RunID}
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
	for _, entry := range index.Entries {
		if entry.AppendReceiptID == opts.ReceiptID {
			return empty, fmt.Errorf("planning timeline append receipt %q already exists", opts.ReceiptID)
		}
		if entry.CardID == canonicalCard.ID || entry.Iteration == canonicalCard.Iteration {
			return empty, fmt.Errorf("planning timeline iteration %d or card %q already exists", canonicalCard.Iteration, canonicalCard.ID)
		}
	}
	if _, statErr := os.Lstat(filepath.Join(repositoryRoot, filepath.FromSlash(cardPath))); statErr == nil {
		return empty, fmt.Errorf("planning timeline card path %q already exists outside the index", cardPath)
	} else if !os.IsNotExist(statErr) {
		return empty, fmt.Errorf("inspect planning timeline card path %q: %w", cardPath, statErr)
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
	return planningTimelineAppendReceipt{
		SchemaVersion:  planningTimelineAppendSchemaVersion,
		ReceiptID:      opts.ReceiptID,
		RequestDigest:  requestDigest,
		RunID:          canonicalCard.RunID,
		Iteration:      canonicalCard.Iteration,
		CardID:         canonicalCard.ID,
		CardHash:       canonicalCard.ContentHash,
		CardPath:       cardPath,
		IndexPath:      indexPath,
		TimelineDigest: index.TimelineDigest,
		WriteReceipt:   writeReceipt,
	}, nil
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
	content, err := os.ReadFile(indexPath)
	if os.IsNotExist(err) {
		legacyMatches, globErr := filepath.Glob(filepath.Join(filepath.Dir(indexPath), "iterations", "*.json"))
		if globErr != nil {
			return planningTimelineIndex{}, nil, false, globErr
		}
		if len(legacyMatches) > 0 {
			return planningTimelineIndex{}, nil, false, fmt.Errorf("planning timeline has unindexed iteration artifacts")
		}
		return planningTimelineIndex{}, nil, false, nil
	}
	if err != nil {
		return planningTimelineIndex{}, nil, false, fmt.Errorf("read planning timeline index: %w", err)
	}
	var index planningTimelineIndex
	if err := decodePlanningTimelineJSON(content, &index); err != nil {
		return planningTimelineIndex{}, nil, false, fmt.Errorf("decode planning timeline index: %w", err)
	}
	cards := make([]colony.PlanningIterationCard, len(index.Entries))
	for i, entry := range index.Entries {
		cardPath := filepath.Join(root, filepath.FromSlash(entry.CardPath))
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
