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
	"path"
	"path/filepath"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

const (
	planningStageReceiptSchemaVersion      = "planning-stage-receipt/v1"
	planningStageReceiptIndexSchemaVersion = "planning-stage-receipt-index/v1"
)

// StageReceipt is the immutable boundary between two visible planning
// stages. It binds one manifest and one worker output to the state reached by
// consuming that output. It grants no authority to dispatch the next stage.
type StageReceipt struct {
	SchemaVersion         string                   `json:"schema_version"`
	ID                    string                   `json:"id"`
	ContentHash           string                   `json:"content_hash"`
	RequestDigest         string                   `json:"request_digest"`
	RunID                 string                   `json:"run_id"`
	Pass                  int                      `json:"pass"`
	ManifestID            string                   `json:"manifest_id"`
	ManifestHash          string                   `json:"manifest_hash"`
	InputFrontierHash     string                   `json:"input_frontier_hash"`
	OutputPath            string                   `json:"output_path"`
	OutputHash            string                   `json:"output_hash"`
	Caste                 planningStageWorkerCaste `json:"caste"`
	PriorReceiptID        string                   `json:"prior_receipt_id,omitempty"`
	PriorReceiptHash      string                   `json:"prior_receipt_hash,omitempty"`
	ResultingState        planningStage            `json:"resulting_state"`
	CandidateSnapshotHash string                   `json:"candidate_snapshot_hash,omitempty"`
	DecisionResumeStage   planningStage            `json:"decision_resume_stage,omitempty"`
	FailureReason         string                   `json:"failure_reason,omitempty"`
	RouteCardRequestHash  string                   `json:"route_card_request_hash,omitempty"`
}

func (receipt StageReceipt) reference() planningStageReceiptRef {
	return planningStageReceiptRef{
		ID:           receipt.ID,
		ContentHash:  receipt.ContentHash,
		RunID:        receipt.RunID,
		Pass:         receipt.Pass,
		Caste:        receipt.Caste,
		ManifestHash: receipt.ManifestHash,
	}
}

type planningStageReceiptIndex struct {
	SchemaVersion string                           `json:"schema_version"`
	ID            string                           `json:"id"`
	ContentHash   string                           `json:"content_hash"`
	RunID         string                           `json:"run_id"`
	Entries       []planningStageReceiptIndexEntry `json:"entries"`
}

type planningStageReceiptIndexEntry struct {
	Pass             int                      `json:"pass"`
	Caste            planningStageWorkerCaste `json:"caste"`
	ManifestID       string                   `json:"manifest_id"`
	ManifestHash     string                   `json:"manifest_hash"`
	ReceiptID        string                   `json:"receipt_id"`
	ReceiptHash      string                   `json:"receipt_hash"`
	ReceiptPath      string                   `json:"receipt_path"`
	PriorReceiptID   string                   `json:"prior_receipt_id,omitempty"`
	PriorReceiptHash string                   `json:"prior_receipt_hash,omitempty"`
	OutputPath       string                   `json:"output_path"`
	OutputHash       string                   `json:"output_hash"`
	ResultingState   planningStage            `json:"resulting_state"`
	CardID           string                   `json:"card_id,omitempty"`
	CardHash         string                   `json:"card_hash,omitempty"`
}

type planningStageOutputReference struct {
	Path        string `json:"path"`
	ContentHash string `json:"content_hash"`
}

type planningStageWriteOptions struct {
	Fault  lifecycleTransactionFaultHook
	Rename func(oldPath, newPath string) error
}

type planningStageFinalizeRequest struct {
	To                    planningStage
	CandidateSnapshotHash string
	DecisionResumeStage   planningStage
	FailureReason         string
	RouteCard             *colony.PlanningIterationCard
	Fault                 lifecycleTransactionFaultHook
	Rename                func(oldPath, newPath string) error
}

type planningStageResumeAction string

const (
	planningStageResumeRetry           planningStageResumeAction = "retry"
	planningStageResumeFinalize        planningStageResumeAction = "finalize"
	planningStageResumeNextStage       planningStageResumeAction = "next_stage"
	planningStageResumeOwnerDecision   planningStageResumeAction = "owner_decision"
	planningStageResumeCandidateReview planningStageResumeAction = "candidate_review"
)

type planningStageResumeResult struct {
	Action      planningStageResumeAction      `json:"action"`
	State       planningStageState             `json:"state"`
	Manifest    *planningStageManifest         `json:"manifest,omitempty"`
	Output      *planningStageOutputReference  `json:"output,omitempty"`
	LastReceipt *StageReceipt                  `json:"last_receipt,omitempty"`
	Cards       []colony.PlanningIterationCard `json:"cards,omitempty"`
}

type planningStageReceiptConflictError struct {
	ManifestID string
	Detail     string
}

func (err *planningStageReceiptConflictError) Error() string {
	if strings.TrimSpace(err.ManifestID) == "" {
		return "planning stage receipt conflict: " + err.Detail
	}
	return fmt.Sprintf("planning stage manifest %q conflicts: %s", err.ManifestID, err.Detail)
}

type planningStageReceiptChain struct {
	Index    planningStageReceiptIndex
	Receipts []StageReceipt
	Cards    []colony.PlanningIterationCard
	Exists   bool
}

// recordPlanningStageDispatch freezes the one-stage manifest together with
// the corresponding running state. Dispatch persistence is intentionally
// separate from output and finalization so resume can distinguish retry from
// finalize without guessing from worker liveness.
func recordPlanningStageDispatch(root string, state planningStageState, manifest planningStageManifest, opts planningStageWriteOptions) error {
	repositoryRoot, dataRoot, err := planningStageRoots(root)
	if err != nil {
		return err
	}
	if err := validatePlanningStageManifest(manifest); err != nil {
		return err
	}
	if err := validatePlanningStageRunningState(state, manifest); err != nil {
		return err
	}
	manifestBytes, err := marshalPlanningStageJSON(manifest)
	if err != nil {
		return fmt.Errorf("marshal planning stage manifest: %w", err)
	}
	stateBytes, err := marshalPlanningStageJSON(state)
	if err != nil {
		return fmt.Errorf("marshal planning stage state: %w", err)
	}
	manifestPath := planningStageManifestRepositoryPath(manifest.RunID, manifest.ID)
	statePath := planningStageStateRepositoryPath(manifest.RunID)

	if existing, ok, err := readOptionalPlanningStageFile(repositoryRoot, manifestPath); err != nil {
		return err
	} else if ok {
		if !bytes.Equal(existing, manifestBytes) {
			return &planningStageReceiptConflictError{ManifestID: manifest.ID, Detail: "persisted manifest bytes differ from the authorized manifest"}
		}
		current, exists, err := readOptionalPlanningStageFile(repositoryRoot, statePath)
		if err != nil {
			return err
		}
		if exists && bytes.Equal(current, stateBytes) {
			return nil
		}
		return &planningStageReceiptConflictError{ManifestID: manifest.ID, Detail: "dispatch replay does not match the current planning state"}
	}

	config := planningStageWriteConfig(repositoryRoot, dataRoot, "planning-stage-dispatch-"+manifest.ContentHash[:24], "planning-stage-dispatch", opts)
	expected := map[string][]byte{
		planningStageDataRelativePath(manifestPath): manifestBytes,
		planningStageDataRelativePath(statePath):    stateBytes,
	}
	if resumed, err := resumePlanningStageWrite(config, expected, manifest.ID); err != nil {
		return err
	} else if resumed {
		return nil
	}
	tx, err := beginLifecycleTransaction(config)
	if err != nil {
		return err
	}
	if err := tx.DeclareWrite(lifecycleTransactionRootData, planningStageDataRelativePath(manifestPath), manifestBytes); err != nil {
		return err
	}
	if err := tx.DeclareWrite(lifecycleTransactionRootData, planningStageDataRelativePath(statePath), stateBytes); err != nil {
		return err
	}
	if err := tx.Validate(); err != nil {
		return err
	}
	_, err = tx.Commit()
	return err
}

// writePlanningStageOutput assigns one deterministic artifact path to a
// manifest. The path is write-once: an exact retry is a no-op and different
// bytes are a typed conflict.
func writePlanningStageOutput(root string, manifest planningStageManifest, output []byte, opts planningStageWriteOptions) (planningStageOutputReference, error) {
	empty := planningStageOutputReference{}
	repositoryRoot, dataRoot, err := planningStageRoots(root)
	if err != nil {
		return empty, err
	}
	if err := validatePlanningStageManifest(manifest); err != nil {
		return empty, err
	}
	if len(output) == 0 {
		return empty, fmt.Errorf("planning stage output is empty")
	}
	if err := validatePersistedPlanningStageManifest(repositoryRoot, manifest); err != nil {
		return empty, err
	}
	reference := planningStageOutputReference{
		Path:        planningStageOutputRepositoryPath(manifest),
		ContentHash: planningStageBytesHash(output),
	}
	if existing, ok, err := readOptionalPlanningStageFile(repositoryRoot, reference.Path); err != nil {
		return empty, err
	} else if ok {
		if planningStageBytesHash(existing) != reference.ContentHash || !bytes.Equal(existing, output) {
			return empty, &planningStageReceiptConflictError{ManifestID: manifest.ID, Detail: "output bytes differ from the staged artifact"}
		}
		return reference, nil
	}
	state, err := loadPlanningStageState(repositoryRoot, manifest.RunID)
	if err != nil {
		return empty, err
	}
	if err := validatePlanningStageRunningState(state, manifest); err != nil {
		return empty, err
	}

	config := planningStageWriteConfig(repositoryRoot, dataRoot, "planning-stage-output-"+manifest.ContentHash[:24], "planning-stage-output", opts)
	expected := map[string][]byte{planningStageDataRelativePath(reference.Path): bytes.Clone(output)}
	if resumed, err := resumePlanningStageWrite(config, expected, manifest.ID); err != nil {
		return empty, err
	} else if resumed {
		return reference, nil
	}
	tx, err := beginLifecycleTransaction(config)
	if err != nil {
		return empty, err
	}
	if err := tx.DeclareWrite(lifecycleTransactionRootData, planningStageDataRelativePath(reference.Path), output); err != nil {
		return empty, err
	}
	if err := tx.Validate(); err != nil {
		return empty, err
	}
	if _, err := tx.Commit(); err != nil {
		return empty, err
	}
	return reference, nil
}

// finalizePlanningStage consumes the persisted output, constructs the exact
// receipt, runs the legal transition reducer, and commits the receipt plus
// resulting state in one recoverable lifecycle transaction. A completed Route
// stage adds its iteration card and timeline index to that same transaction.
func finalizePlanningStage(root string, manifest planningStageManifest, request planningStageFinalizeRequest) (StageReceipt, error) {
	empty := StageReceipt{}
	repositoryRoot, dataRoot, err := planningStageRoots(root)
	if err != nil {
		return empty, err
	}
	if err := validatePlanningStageManifest(manifest); err != nil {
		return empty, err
	}
	if err := validatePersistedPlanningStageManifest(repositoryRoot, manifest); err != nil {
		return empty, err
	}
	chain, err := readPlanningStageReceiptChain(repositoryRoot, manifest.RunID)
	if err != nil {
		return empty, err
	}
	for index, receipt := range chain.Receipts {
		if receipt.ManifestID != manifest.ID {
			continue
		}
		if err := planningStageReceiptMatchesFinalizeRequest(repositoryRoot, receipt, request, chain.Index.Entries[index]); err != nil {
			return empty, &planningStageReceiptConflictError{ManifestID: manifest.ID, Detail: err.Error()}
		}
		config, err := planningStageFinalizationConfig(repositoryRoot, dataRoot, manifest, planningStageWriteOptions{})
		if err != nil {
			return empty, err
		}
		// A crash may leave all bounded stage files durable just before the
		// generic lifecycle receipt. Finish only the latest transaction; old
		// transaction targets are legitimately superseded by later stages.
		if index == len(chain.Receipts)-1 && !planningStageLifecycleReceiptExists(config) {
			if _, err := resumeLifecycleTransaction(config); err != nil {
				return empty, err
			}
		}
		return receipt, nil
	}

	state, err := loadPlanningStageState(repositoryRoot, manifest.RunID)
	if err != nil {
		return empty, err
	}
	if err := validatePlanningStageRunningState(state, manifest); err != nil {
		return empty, err
	}
	output, _, err := loadPlanningStageOutput(repositoryRoot, manifest)
	if err != nil {
		return empty, err
	}
	var prior *StageReceipt
	if len(chain.Receipts) > 0 {
		copy := chain.Receipts[len(chain.Receipts)-1]
		prior = &copy
	}
	if err := validatePlanningStageReceiptSequence(manifest, prior); err != nil {
		return empty, err
	}
	routeCardRequestHash, err := planningStageRouteCardRequestHash(manifest, request.RouteCard)
	if err != nil {
		return empty, err
	}
	receipt := StageReceipt{
		SchemaVersion:         planningStageReceiptSchemaVersion,
		RunID:                 manifest.RunID,
		Pass:                  manifest.Pass,
		ManifestID:            manifest.ID,
		ManifestHash:          manifest.ContentHash,
		InputFrontierHash:     manifest.InputFrontierHash,
		OutputPath:            output.Path,
		OutputHash:            output.ContentHash,
		Caste:                 manifest.ExpectedCaste,
		ResultingState:        request.To,
		CandidateSnapshotHash: strings.TrimSpace(request.CandidateSnapshotHash),
		DecisionResumeStage:   request.DecisionResumeStage,
		FailureReason:         strings.TrimSpace(request.FailureReason),
		RouteCardRequestHash:  routeCardRequestHash,
	}
	if prior != nil {
		receipt.PriorReceiptID = prior.ID
		receipt.PriorReceiptHash = prior.ContentHash
	}
	if err := addressStageReceipt(&receipt); err != nil {
		return empty, err
	}

	var card colony.PlanningIterationCard
	var cardBytes []byte
	var timelineIndexBytes []byte
	var timelineCards []colony.PlanningIterationCard
	if request.RouteCard != nil {
		card, cardBytes, _, timelineIndexBytes, timelineCards, err = preparePlanningStageRouteCard(repositoryRoot, manifest, receipt, *request.RouteCard)
		if err != nil {
			return empty, err
		}
	}
	next, err := reducePlanningStageReceiptTransition(state, receipt, request, card)
	if err != nil {
		return empty, err
	}
	stateBytes, err := marshalPlanningStageJSON(next)
	if err != nil {
		return empty, fmt.Errorf("marshal resulting planning stage state: %w", err)
	}
	receiptBytes, err := marshalPlanningStageJSON(receipt)
	if err != nil {
		return empty, fmt.Errorf("marshal planning stage receipt: %w", err)
	}
	entry := planningStageReceiptIndexEntry{
		Pass:             receipt.Pass,
		Caste:            receipt.Caste,
		ManifestID:       receipt.ManifestID,
		ManifestHash:     receipt.ManifestHash,
		ReceiptID:        receipt.ID,
		ReceiptHash:      receipt.ContentHash,
		ReceiptPath:      planningStageReceiptRepositoryPath(receipt.RunID, receipt.ID),
		PriorReceiptID:   receipt.PriorReceiptID,
		PriorReceiptHash: receipt.PriorReceiptHash,
		OutputPath:       receipt.OutputPath,
		OutputHash:       receipt.OutputHash,
		ResultingState:   receipt.ResultingState,
	}
	if card.ID != "" {
		entry.CardID = card.ID
		entry.CardHash = card.ContentHash
	}
	index := chain.Index
	if !chain.Exists {
		index = planningStageReceiptIndex{SchemaVersion: planningStageReceiptIndexSchemaVersion, RunID: manifest.RunID}
	}
	index.Entries = append(index.Entries, entry)
	if err := addressPlanningStageReceiptIndex(&index); err != nil {
		return empty, err
	}
	indexBytes, err := marshalPlanningStageJSON(index)
	if err != nil {
		return empty, fmt.Errorf("marshal planning stage receipt index: %w", err)
	}

	config, err := planningStageFinalizationConfig(repositoryRoot, dataRoot, manifest, planningStageWriteOptions{Fault: request.Fault, Rename: request.Rename})
	if err != nil {
		return empty, err
	}
	expected := map[string][]byte{
		planningStageDataRelativePath(entry.ReceiptPath):                                       receiptBytes,
		planningStageDataRelativePath(planningStageStateRepositoryPath(manifest.RunID)):        stateBytes,
		planningStageDataRelativePath(planningStageReceiptIndexRepositoryPath(manifest.RunID)): indexBytes,
	}
	if card.ID != "" {
		expected[planningStageDataRelativePath(planningTimelineCardRepositoryPath(manifest.RunID, card.Iteration, card.ID))] = cardBytes
		expected[planningStageDataRelativePath(planningTimelineIndexRepositoryPath(manifest.RunID))] = timelineIndexBytes
	}
	if resumed, err := resumePlanningStageWrite(config, expected, manifest.ID); err != nil {
		return empty, err
	} else if resumed {
		completed, err := readPlanningStageReceiptChain(repositoryRoot, manifest.RunID)
		if err != nil {
			return empty, err
		}
		return planningStageReceiptByManifest(completed, manifest.ID)
	}

	tx, err := beginLifecycleTransaction(config)
	if err != nil {
		return empty, err
	}
	if err := tx.DeclareWrite(lifecycleTransactionRootData, planningStageDataRelativePath(entry.ReceiptPath), receiptBytes); err != nil {
		return empty, err
	}
	if card.ID != "" {
		cardPath := planningTimelineCardRepositoryPath(manifest.RunID, card.Iteration, card.ID)
		if err := tx.DeclareWrite(lifecycleTransactionRootData, planningStageDataRelativePath(cardPath), cardBytes); err != nil {
			return empty, err
		}
		if err := tx.DeclareWrite(lifecycleTransactionRootData, planningStageDataRelativePath(planningTimelineIndexRepositoryPath(manifest.RunID)), timelineIndexBytes); err != nil {
			return empty, err
		}
	}
	if err := tx.DeclareWrite(lifecycleTransactionRootData, planningStageDataRelativePath(planningStageStateRepositoryPath(manifest.RunID)), stateBytes); err != nil {
		return empty, err
	}
	if err := tx.DeclareWrite(lifecycleTransactionRootData, planningStageDataRelativePath(planningStageReceiptIndexRepositoryPath(manifest.RunID)), indexBytes); err != nil {
		return empty, err
	}
	if err := tx.Validate(); err != nil {
		return empty, err
	}
	if _, err := tx.Commit(); err != nil {
		return empty, err
	}
	completed, err := readPlanningStageReceiptChain(repositoryRoot, manifest.RunID)
	if err != nil {
		return empty, err
	}
	if card.ID != "" && len(timelineCards) == 0 {
		return empty, fmt.Errorf("planning stage Route finalization lost its timeline card")
	}
	return planningStageReceiptByManifest(completed, manifest.ID)
}

// resumePlanningStage reconstructs one action from durable stage state,
// receipt/output hashes, manifests, and iteration cards. It performs no write;
// callers execute the returned retry/finalize/next/human action explicitly.
func resumePlanningStage(root, runID string) (planningStageResumeResult, error) {
	empty := planningStageResumeResult{}
	repositoryRoot, err := canonicalPlanningTimelineRoot(root)
	if err != nil {
		return empty, err
	}
	if err := validatePlanningTimelineSegment("run_id", runID); err != nil {
		return empty, err
	}
	chain, err := readPlanningStageReceiptChain(repositoryRoot, runID)
	if err != nil {
		return empty, err
	}
	state, err := loadPlanningStageState(repositoryRoot, runID)
	if err != nil {
		return empty, err
	}
	result := planningStageResumeResult{State: state, Cards: append([]colony.PlanningIterationCard(nil), chain.Cards...)}
	if len(chain.Receipts) > 0 {
		last := chain.Receipts[len(chain.Receipts)-1]
		result.LastReceipt = &last
	}
	if state.ActiveManifestID != "" || state.ActiveManifestHash != "" {
		if state.ActiveManifestID == "" || state.ActiveManifestHash == "" {
			return empty, fmt.Errorf("planning stage active manifest identity is incomplete")
		}
		manifest, err := loadPlanningStageManifest(repositoryRoot, runID, state.ActiveManifestID)
		if err != nil {
			return empty, err
		}
		if manifest.ContentHash != state.ActiveManifestHash {
			return empty, fmt.Errorf("planning stage active manifest hash conflicts with persisted state")
		}
		if err := validatePlanningStageRunningState(state, manifest); err != nil {
			return empty, err
		}
		result.Manifest = &manifest
		output, _, outputErr := loadPlanningStageOutput(repositoryRoot, manifest)
		if outputErr == nil {
			result.Output = &output
			result.Action = planningStageResumeFinalize
			return result, nil
		}
		if !errors.Is(outputErr, os.ErrNotExist) {
			return empty, outputErr
		}
		result.Action = planningStageResumeRetry
		return result, nil
	}

	if len(chain.Receipts) > 0 {
		last := chain.Receipts[len(chain.Receipts)-1]
		if state.Stage != last.ResultingState {
			return empty, fmt.Errorf("planning stage state %q does not match last receipt state %q", state.Stage, last.ResultingState)
		}
	}
	switch state.Stage {
	case planningStageOwnerDecision, planningStageSpecApprovalRequired, planningStageReconciliationRequired:
		result.Action = planningStageResumeOwnerDecision
	case planningStageCandidateReady:
		result.Action = planningStageResumeCandidateReview
	default:
		result.Action = planningStageResumeNextStage
	}
	return result, nil
}

func addressStageReceipt(receipt *StageReceipt) error {
	if receipt == nil {
		return fmt.Errorf("planning stage receipt is required")
	}
	receipt.SchemaVersion = planningStageReceiptSchemaVersion
	receipt.RunID = strings.TrimSpace(receipt.RunID)
	receipt.ManifestID = strings.TrimSpace(receipt.ManifestID)
	receipt.ManifestHash = strings.TrimSpace(receipt.ManifestHash)
	receipt.InputFrontierHash = strings.TrimSpace(receipt.InputFrontierHash)
	receipt.OutputPath = strings.TrimSpace(receipt.OutputPath)
	receipt.OutputHash = strings.TrimSpace(receipt.OutputHash)
	receipt.PriorReceiptID = strings.TrimSpace(receipt.PriorReceiptID)
	receipt.PriorReceiptHash = strings.TrimSpace(receipt.PriorReceiptHash)
	receipt.CandidateSnapshotHash = strings.TrimSpace(receipt.CandidateSnapshotHash)
	receipt.FailureReason = strings.TrimSpace(receipt.FailureReason)
	receipt.RouteCardRequestHash = strings.TrimSpace(receipt.RouteCardRequestHash)
	requestDigest, err := planningStageReceiptRequestDigest(*receipt)
	if err != nil {
		return err
	}
	receipt.RequestDigest = requestDigest
	if err := validateStageReceiptContract(*receipt); err != nil {
		return err
	}
	payload := *receipt
	payload.ID = ""
	payload.ContentHash = ""
	contentHash, err := jsonSHA256(payload)
	if err != nil {
		return fmt.Errorf("hash planning stage receipt: %w", err)
	}
	receipt.ContentHash = contentHash
	receipt.ID = "planning-stage-receipt-" + contentHash[:16]
	return nil
}

func validateStageReceipt(receipt StageReceipt) error {
	if receipt.SchemaVersion != planningStageReceiptSchemaVersion {
		return fmt.Errorf("planning stage receipt schema_version must be %q", planningStageReceiptSchemaVersion)
	}
	if err := validateStageReceiptContract(receipt); err != nil {
		return err
	}
	wantRequest, err := planningStageReceiptRequestDigest(receipt)
	if err != nil {
		return err
	}
	if receipt.RequestDigest != wantRequest {
		return fmt.Errorf("planning stage receipt request digest does not match its immutable boundary")
	}
	payload := receipt
	payload.ID = ""
	payload.ContentHash = ""
	wantHash, err := jsonSHA256(payload)
	if err != nil {
		return err
	}
	if receipt.ContentHash != wantHash || receipt.ID != "planning-stage-receipt-"+wantHash[:16] {
		return fmt.Errorf("planning stage receipt content address does not match its canonical payload")
	}
	return nil
}

func validateStageReceiptContract(receipt StageReceipt) error {
	if err := validatePlanningTimelineSegment("receipt run_id", receipt.RunID); err != nil {
		return err
	}
	if receipt.Pass <= 0 {
		return fmt.Errorf("planning stage receipt pass must be positive")
	}
	if err := validatePlanningTimelineSegment("manifest_id", receipt.ManifestID); err != nil {
		return err
	}
	for _, digest := range []struct {
		name  string
		value string
	}{
		{"manifest_hash", receipt.ManifestHash},
		{"input_frontier_hash", receipt.InputFrontierHash},
		{"output_hash", receipt.OutputHash},
		{"request_digest", receipt.RequestDigest},
	} {
		if !planningSHA256Pattern.MatchString(digest.value) {
			return fmt.Errorf("planning stage receipt %s must be a lowercase SHA-256 digest", digest.name)
		}
	}
	if !receipt.Caste.valid() {
		return fmt.Errorf("planning stage receipt caste %q is invalid", receipt.Caste)
	}
	if !receipt.ResultingState.valid() {
		return fmt.Errorf("planning stage receipt resulting state %q is invalid", receipt.ResultingState)
	}
	if err := validatePlanningArtifactPath(receipt.OutputPath); err != nil {
		return fmt.Errorf("planning stage receipt output_path: %w", err)
	}
	if (receipt.PriorReceiptID == "") != (receipt.PriorReceiptHash == "") {
		return fmt.Errorf("planning stage receipt predecessor ID and hash must be set together")
	}
	if receipt.PriorReceiptHash != "" && !planningSHA256Pattern.MatchString(receipt.PriorReceiptHash) {
		return fmt.Errorf("planning stage receipt prior_receipt_hash must be a lowercase SHA-256 digest")
	}
	if receipt.CandidateSnapshotHash != "" && !planningSHA256Pattern.MatchString(receipt.CandidateSnapshotHash) {
		return fmt.Errorf("planning stage receipt candidate snapshot hash is invalid")
	}
	if receipt.RouteCardRequestHash != "" && !planningSHA256Pattern.MatchString(receipt.RouteCardRequestHash) {
		return fmt.Errorf("planning stage receipt Route card request hash is invalid")
	}
	return nil
}

func planningStageReceiptRequestDigest(receipt StageReceipt) (string, error) {
	return jsonSHA256(struct {
		RunID                 string                   `json:"run_id"`
		Pass                  int                      `json:"pass"`
		ManifestID            string                   `json:"manifest_id"`
		ManifestHash          string                   `json:"manifest_hash"`
		InputFrontierHash     string                   `json:"input_frontier_hash"`
		OutputPath            string                   `json:"output_path"`
		OutputHash            string                   `json:"output_hash"`
		Caste                 planningStageWorkerCaste `json:"caste"`
		PriorReceiptID        string                   `json:"prior_receipt_id,omitempty"`
		PriorReceiptHash      string                   `json:"prior_receipt_hash,omitempty"`
		ResultingState        planningStage            `json:"resulting_state"`
		CandidateSnapshotHash string                   `json:"candidate_snapshot_hash,omitempty"`
		DecisionResumeStage   planningStage            `json:"decision_resume_stage,omitempty"`
		FailureReason         string                   `json:"failure_reason,omitempty"`
		RouteCardRequestHash  string                   `json:"route_card_request_hash,omitempty"`
	}{
		RunID: receipt.RunID, Pass: receipt.Pass, ManifestID: receipt.ManifestID,
		ManifestHash: receipt.ManifestHash, InputFrontierHash: receipt.InputFrontierHash,
		OutputPath: receipt.OutputPath, OutputHash: receipt.OutputHash, Caste: receipt.Caste,
		PriorReceiptID: receipt.PriorReceiptID, PriorReceiptHash: receipt.PriorReceiptHash,
		ResultingState: receipt.ResultingState, CandidateSnapshotHash: receipt.CandidateSnapshotHash,
		DecisionResumeStage: receipt.DecisionResumeStage, FailureReason: receipt.FailureReason,
		RouteCardRequestHash: receipt.RouteCardRequestHash,
	})
}

func reducePlanningStageReceiptTransition(state planningStageState, receipt StageReceipt, request planningStageFinalizeRequest, card colony.PlanningIterationCard) (planningStageState, error) {
	transition := planningStageTransition{
		To:                    receipt.ResultingState,
		CandidateSnapshotHash: receipt.CandidateSnapshotHash,
		DecisionResumeStage:   receipt.DecisionResumeStage,
		FailureReason:         receipt.FailureReason,
	}
	switch receipt.Caste {
	case planningStageCasteScout:
		ref := receipt.reference()
		transition.ScoutReceipt = &ref
	case planningStageCasteRouteSetter:
		if card.ID != "" {
			transition.ResultingCardHash = card.ContentHash
		} else {
			transition.ResultingCardHash = receipt.OutputHash
		}
	}
	next, manifest, err := reducePlanningStage(state, transition)
	if err != nil {
		return planningStageState{}, err
	}
	if manifest != nil || next.Stage != receipt.ResultingState {
		return planningStageState{}, fmt.Errorf("planning stage receipt transition emitted unexpected worker authority")
	}
	if next.ActiveManifestID != "" || next.ActiveManifestHash != "" {
		return planningStageState{}, fmt.Errorf("completed planning stage retained active manifest authority")
	}
	return next, nil
}

func validatePlanningStageReceiptSequence(manifest planningStageManifest, prior *StageReceipt) error {
	if prior == nil {
		if manifest.ExpectedCaste != planningStageCasteScout || manifest.Pass != 1 {
			return fmt.Errorf("first planning stage receipt must be Scout pass 1")
		}
		return nil
	}
	if manifest.Pass < prior.Pass || manifest.Pass > prior.Pass+1 {
		return fmt.Errorf("planning stage receipt pass %d does not follow predecessor pass %d", manifest.Pass, prior.Pass)
	}
	if manifest.Pass == prior.Pass+1 && manifest.ExpectedCaste != planningStageCasteScout {
		return fmt.Errorf("a new planning pass must begin with Scout")
	}
	if manifest.ExpectedCaste == planningStageCasteRouteSetter {
		if prior.Caste != planningStageCasteScout || prior.Pass != manifest.Pass || manifest.ScoutReceipt == nil ||
			manifest.ScoutReceipt.ID != prior.ID || manifest.ScoutReceipt.ContentHash != prior.ContentHash {
			return fmt.Errorf("Route receipt must directly follow the exact current Scout receipt")
		}
	}
	return nil
}

func preparePlanningStageRouteCard(root string, manifest planningStageManifest, receipt StageReceipt, requested colony.PlanningIterationCard) (colony.PlanningIterationCard, []byte, planningTimelineIndex, []byte, []colony.PlanningIterationCard, error) {
	emptyCard := colony.PlanningIterationCard{}
	if manifest.ExpectedCaste != planningStageCasteRouteSetter || manifest.ScoutReceipt == nil {
		return emptyCard, nil, planningTimelineIndex{}, nil, nil, fmt.Errorf("only Route-Setter with an exact Scout receipt can append an iteration card")
	}
	if requested.Iteration != manifest.Pass {
		return emptyCard, nil, planningTimelineIndex{}, nil, nil, fmt.Errorf("Route iteration card ordinal %d does not match manifest pass %d", requested.Iteration, manifest.Pass)
	}
	requested.SchemaVersion = colony.PlanningIterationSchemaVersion
	requested.ID = ""
	requested.ContentHash = ""
	requested.RunID = manifest.RunID
	requested.ScoutReceiptID = manifest.ScoutReceipt.ID
	requested.ScoutReceiptHash = manifest.ScoutReceipt.ContentHash
	requested.RouteSetterReceiptID = receipt.ID
	requested.RouteSetterReceiptHash = receipt.ContentHash
	card, cardBytes, err := canonicalPlanningTimelineCard(requested)
	if err != nil {
		return emptyCard, nil, planningTimelineIndex{}, nil, nil, err
	}
	index, cards, exists, err := readPlanningTimelineChain(root, manifest.RunID)
	if err != nil {
		return emptyCard, nil, planningTimelineIndex{}, nil, nil, err
	}
	for _, entry := range index.Entries {
		if entry.AppendReceiptID != manifest.ID {
			continue
		}
		if entry.CardID != card.ID || entry.CardHash != card.ContentHash || entry.Iteration != card.Iteration {
			return emptyCard, nil, planningTimelineIndex{}, nil, nil, &planningStageReceiptConflictError{ManifestID: manifest.ID, Detail: "Route card differs from the durable in-flight append"}
		}
		indexBytes, err := marshalPlanningTimelineJSON(index)
		if err != nil {
			return emptyCard, nil, planningTimelineIndex{}, nil, nil, err
		}
		return card, cardBytes, index, indexBytes, cards, nil
	}
	if card.Iteration != len(cards)+1 {
		return emptyCard, nil, planningTimelineIndex{}, nil, nil, fmt.Errorf("Route iteration card %d must be next timeline ordinal %d", card.Iteration, len(cards)+1)
	}
	previousHash := ""
	if len(cards) > 0 {
		previousHash = cards[len(cards)-1].ContentHash
		if manifest.PriorCardHash != previousHash {
			return emptyCard, nil, planningTimelineIndex{}, nil, nil, fmt.Errorf("Route manifest prior card hash does not match the current timeline")
		}
	}
	if !exists {
		legacy, err := planningTimelineLegacyArtifacts(root, manifest.RunID)
		if err != nil {
			return emptyCard, nil, planningTimelineIndex{}, nil, nil, err
		}
		if len(legacy) > 0 {
			return emptyCard, nil, planningTimelineIndex{}, nil, nil, fmt.Errorf("planning timeline has legacy_unbound iteration evidence")
		}
		index = planningTimelineIndex{SchemaVersion: planningTimelineIndexSchemaVersion, RunID: manifest.RunID}
	}
	transactionID, err := planningTimelineTransactionID(manifest.RunID, manifest.ID)
	if err != nil {
		return emptyCard, nil, planningTimelineIndex{}, nil, nil, err
	}
	requestDigest, err := planningTimelineAppendRequestDigest(manifest.ID, card, previousHash)
	if err != nil {
		return emptyCard, nil, planningTimelineIndex{}, nil, nil, err
	}
	cardPath := planningTimelineCardRepositoryPath(manifest.RunID, card.Iteration, card.ID)
	if _, statErr := os.Lstat(filepath.Join(root, filepath.FromSlash(cardPath))); statErr == nil {
		return emptyCard, nil, planningTimelineIndex{}, nil, nil, &planningStageReceiptConflictError{ManifestID: manifest.ID, Detail: "Route card path exists outside its timeline index"}
	} else if !os.IsNotExist(statErr) {
		return emptyCard, nil, planningTimelineIndex{}, nil, nil, statErr
	}
	index.Entries = append(index.Entries, planningTimelineIndexEntry{
		Iteration: card.Iteration, CardID: card.ID, CardHash: card.ContentHash,
		PreviousCardHash: previousHash, CardPath: cardPath, AppendReceiptID: manifest.ID,
		AppendRequestDigest: requestDigest, TransactionID: transactionID,
	})
	cards = append(cards, card)
	if err := addressPlanningTimelineIndex(&index, cards); err != nil {
		return emptyCard, nil, planningTimelineIndex{}, nil, nil, err
	}
	indexBytes, err := marshalPlanningTimelineJSON(index)
	if err != nil {
		return emptyCard, nil, planningTimelineIndex{}, nil, nil, err
	}
	return card, cardBytes, index, indexBytes, cards, nil
}

func planningStageRouteCardRequestHash(manifest planningStageManifest, card *colony.PlanningIterationCard) (string, error) {
	if card == nil {
		return "", nil
	}
	clone := *card
	clone.SchemaVersion = colony.PlanningIterationSchemaVersion
	clone.ID = ""
	clone.ContentHash = ""
	clone.RunID = manifest.RunID
	clone.Iteration = manifest.Pass
	clone.ScoutReceiptID = ""
	clone.ScoutReceiptHash = ""
	clone.RouteSetterReceiptID = ""
	clone.RouteSetterReceiptHash = ""
	return jsonSHA256(clone)
}

func readPlanningStageReceiptChain(root, runID string) (planningStageReceiptChain, error) {
	empty := planningStageReceiptChain{}
	indexPath := planningStageReceiptIndexRepositoryPath(runID)
	content, exists, err := readOptionalPlanningStageFile(root, indexPath)
	if err != nil {
		return empty, err
	}
	if !exists {
		return planningStageReceiptChain{Index: planningStageReceiptIndex{SchemaVersion: planningStageReceiptIndexSchemaVersion, RunID: runID}}, nil
	}
	var index planningStageReceiptIndex
	if err := decodePlanningStageJSON(content, &index); err != nil {
		return empty, fmt.Errorf("decode planning stage receipt index: %w", err)
	}
	if err := validatePlanningStageReceiptIndex(index); err != nil {
		return empty, err
	}
	if index.RunID != runID {
		return empty, fmt.Errorf("planning stage receipt index belongs to run %q", index.RunID)
	}
	chain := planningStageReceiptChain{Index: index, Exists: true}
	receiptByID := make(map[string]StageReceipt, len(index.Entries))
	needsTimeline := false
	for position, entry := range index.Entries {
		wantPath := planningStageReceiptRepositoryPath(runID, entry.ReceiptID)
		if entry.ReceiptPath != wantPath {
			return empty, fmt.Errorf("planning stage receipt index entry %d has a non-canonical receipt path", position)
		}
		receiptBytes, ok, err := readOptionalPlanningStageFile(root, entry.ReceiptPath)
		if err != nil {
			return empty, err
		}
		if !ok {
			return empty, fmt.Errorf("planning stage receipt %q is missing: %w", entry.ReceiptID, os.ErrNotExist)
		}
		var receipt StageReceipt
		if err := decodePlanningStageJSON(receiptBytes, &receipt); err != nil {
			return empty, fmt.Errorf("decode planning stage receipt %d: %w", position, err)
		}
		if err := validateStageReceipt(receipt); err != nil {
			return empty, fmt.Errorf("planning stage receipt %d: %w", position, err)
		}
		if err := validatePlanningStageReceiptIndexEntry(entry, receipt); err != nil {
			return empty, fmt.Errorf("planning stage receipt index entry %d: %w", position, err)
		}
		manifest, err := loadPlanningStageManifest(root, runID, receipt.ManifestID)
		if err != nil {
			return empty, err
		}
		if manifest.ContentHash != receipt.ManifestHash || manifest.ExpectedCaste != receipt.Caste || manifest.Pass != receipt.Pass || manifest.InputFrontierHash != receipt.InputFrontierHash {
			return empty, fmt.Errorf("planning stage receipt %q conflicts with its manifest", receipt.ID)
		}
		output, _, err := loadPlanningStageOutput(root, manifest)
		if err != nil {
			return empty, err
		}
		if output.Path != receipt.OutputPath || output.ContentHash != receipt.OutputHash {
			return empty, fmt.Errorf("planning stage receipt %q output hash does not match staged artifact", receipt.ID)
		}
		if position == 0 {
			if receipt.PriorReceiptID != "" || receipt.PriorReceiptHash != "" {
				return empty, fmt.Errorf("first planning stage receipt has a predecessor")
			}
		} else {
			prior := chain.Receipts[position-1]
			if receipt.PriorReceiptID != prior.ID || receipt.PriorReceiptHash != prior.ContentHash {
				return empty, fmt.Errorf("planning stage receipt %q has a broken prior receipt chain", receipt.ID)
			}
		}
		if receipt.Caste == planningStageCasteRouteSetter {
			if manifest.ScoutReceipt == nil {
				return empty, fmt.Errorf("Route receipt %q has no Scout manifest binding", receipt.ID)
			}
			scout, ok := receiptByID[manifest.ScoutReceipt.ID]
			if !ok || scout.ContentHash != manifest.ScoutReceipt.ContentHash || scout.Caste != planningStageCasteScout || scout.Pass != receipt.Pass {
				return empty, fmt.Errorf("Route receipt %q does not follow its exact Scout receipt", receipt.ID)
			}
		}
		if entry.CardID != "" || entry.CardHash != "" {
			if receipt.Caste != planningStageCasteRouteSetter || entry.CardID == "" || !planningSHA256Pattern.MatchString(entry.CardHash) {
				return empty, fmt.Errorf("planning stage receipt %q has an invalid iteration card binding", receipt.ID)
			}
			needsTimeline = true
		}
		receiptByID[receipt.ID] = receipt
		chain.Receipts = append(chain.Receipts, receipt)
	}
	if needsTimeline {
		_, cards, exists, err := readPlanningTimelineChain(root, runID)
		if err != nil {
			return empty, err
		}
		if !exists {
			return empty, fmt.Errorf("planning stage Route receipts require a current iteration-card timeline")
		}
		chain.Cards = cards
		cardsByID := make(map[string]colony.PlanningIterationCard, len(cards))
		for _, card := range cards {
			cardsByID[card.ID] = card
		}
		for _, entry := range index.Entries {
			if entry.CardID == "" {
				continue
			}
			card, ok := cardsByID[entry.CardID]
			if !ok || card.ContentHash != entry.CardHash {
				return empty, fmt.Errorf("planning stage receipt %q iteration card binding is missing or changed", entry.ReceiptID)
			}
			receipt := receiptByID[entry.ReceiptID]
			if card.RouteSetterReceiptID != receipt.ID || card.RouteSetterReceiptHash != receipt.ContentHash {
				return empty, fmt.Errorf("iteration card %q does not bind its exact Route receipt", card.ID)
			}
			manifest, err := loadPlanningStageManifest(root, runID, receipt.ManifestID)
			if err != nil {
				return empty, err
			}
			if manifest.ScoutReceipt == nil || card.ScoutReceiptID != manifest.ScoutReceipt.ID || card.ScoutReceiptHash != manifest.ScoutReceipt.ContentHash {
				return empty, fmt.Errorf("iteration card %q does not bind its exact Scout receipt", card.ID)
			}
		}
	}
	return chain, nil
}

func addressPlanningStageReceiptIndex(index *planningStageReceiptIndex) error {
	if index == nil || len(index.Entries) == 0 {
		return fmt.Errorf("planning stage receipt index requires at least one entry")
	}
	index.SchemaVersion = planningStageReceiptIndexSchemaVersion
	index.ID = ""
	index.ContentHash = ""
	hash, err := jsonSHA256(*index)
	if err != nil {
		return err
	}
	index.ContentHash = hash
	index.ID = "planning-stage-receipt-index-" + hash[:16]
	return validatePlanningStageReceiptIndex(*index)
}

func validatePlanningStageReceiptIndex(index planningStageReceiptIndex) error {
	if index.SchemaVersion != planningStageReceiptIndexSchemaVersion {
		return fmt.Errorf("planning stage receipt index schema_version must be %q", planningStageReceiptIndexSchemaVersion)
	}
	if err := validatePlanningTimelineSegment("receipt index run_id", index.RunID); err != nil {
		return err
	}
	if len(index.Entries) == 0 {
		return fmt.Errorf("planning stage receipt index is empty")
	}
	payload := index
	payload.ID = ""
	payload.ContentHash = ""
	wantHash, err := jsonSHA256(payload)
	if err != nil {
		return err
	}
	if index.ContentHash != wantHash || index.ID != "planning-stage-receipt-index-"+wantHash[:16] {
		return fmt.Errorf("planning stage receipt index content address does not match its canonical payload")
	}
	seenManifest := make(map[string]struct{}, len(index.Entries))
	seenReceipt := make(map[string]struct{}, len(index.Entries))
	for position, entry := range index.Entries {
		if entry.Pass <= 0 || !entry.Caste.valid() || !entry.ResultingState.valid() {
			return fmt.Errorf("planning stage receipt index entry %d has invalid stage vocabulary", position)
		}
		if _, duplicate := seenManifest[entry.ManifestID]; duplicate {
			return fmt.Errorf("planning stage receipt index repeats manifest %q", entry.ManifestID)
		}
		if _, duplicate := seenReceipt[entry.ReceiptID]; duplicate {
			return fmt.Errorf("planning stage receipt index repeats receipt %q", entry.ReceiptID)
		}
		seenManifest[entry.ManifestID] = struct{}{}
		seenReceipt[entry.ReceiptID] = struct{}{}
	}
	return nil
}

func validatePlanningStageReceiptIndexEntry(entry planningStageReceiptIndexEntry, receipt StageReceipt) error {
	if entry.Pass != receipt.Pass || entry.Caste != receipt.Caste || entry.ManifestID != receipt.ManifestID || entry.ManifestHash != receipt.ManifestHash ||
		entry.ReceiptID != receipt.ID || entry.ReceiptHash != receipt.ContentHash || entry.PriorReceiptID != receipt.PriorReceiptID ||
		entry.PriorReceiptHash != receipt.PriorReceiptHash || entry.OutputPath != receipt.OutputPath || entry.OutputHash != receipt.OutputHash ||
		entry.ResultingState != receipt.ResultingState {
		return fmt.Errorf("entry claims do not match receipt %q", receipt.ID)
	}
	return nil
}

func planningStageReceiptMatchesFinalizeRequest(root string, receipt StageReceipt, request planningStageFinalizeRequest, entry planningStageReceiptIndexEntry) error {
	if receipt.ResultingState != request.To || receipt.CandidateSnapshotHash != strings.TrimSpace(request.CandidateSnapshotHash) ||
		receipt.DecisionResumeStage != request.DecisionResumeStage || receipt.FailureReason != strings.TrimSpace(request.FailureReason) {
		return fmt.Errorf("finalizer transition differs from completed receipt")
	}
	manifest, err := loadPlanningStageManifest(root, receipt.RunID, receipt.ManifestID)
	if err != nil {
		return err
	}
	wantCardHash, err := planningStageRouteCardRequestHash(manifest, request.RouteCard)
	if err != nil {
		return err
	}
	if receipt.RouteCardRequestHash != wantCardHash {
		return fmt.Errorf("Route iteration card differs from completed receipt")
	}
	if (entry.CardID == "") != (request.RouteCard == nil) {
		return fmt.Errorf("Route iteration card presence differs from completed receipt")
	}
	return nil
}

func planningStageReceiptByManifest(chain planningStageReceiptChain, manifestID string) (StageReceipt, error) {
	for _, receipt := range chain.Receipts {
		if receipt.ManifestID == manifestID {
			return receipt, nil
		}
	}
	return StageReceipt{}, fmt.Errorf("planning stage manifest %q has no committed receipt", manifestID)
}

func validatePlanningStageRunningState(state planningStageState, manifest planningStageManifest) error {
	wantStage := planningStageScoutRunning
	if manifest.ExpectedCaste == planningStageCasteRouteSetter {
		wantStage = planningStageRouteRunning
	}
	if state.Stage != wantStage {
		return fmt.Errorf("planning stage state %q is not the running state %q for manifest caste %q", state.Stage, wantStage, manifest.ExpectedCaste)
	}
	if state.ActiveManifestID != manifest.ID || state.ActiveManifestHash != manifest.ContentHash {
		return fmt.Errorf("planning stage running state does not bind the exact active manifest")
	}
	if state.RunID != manifest.RunID || state.Pass != manifest.Pass || state.Preset != manifest.Preset || state.InputFrontierHash != manifest.InputFrontierHash ||
		state.Specification.RevisionID != manifest.Specification.RevisionID || state.Specification.ContentHash != manifest.Specification.ContentHash ||
		state.BasePlanRevisionID != manifest.BasePlanRevisionID || state.BasePlanRevisionHash != manifest.BasePlanRevisionHash || state.PriorCardHash != manifest.PriorCardHash {
		return fmt.Errorf("planning stage running state does not match manifest authority")
	}
	return nil
}

func planningStageRoots(root string) (string, string, error) {
	repositoryRoot, err := canonicalPlanningTimelineRoot(root)
	if err != nil {
		return "", "", err
	}
	dataRoot := filepath.Join(repositoryRoot, ".aether", "data")
	if err := os.MkdirAll(dataRoot, 0o755); err != nil {
		return "", "", fmt.Errorf("create planning lifecycle data root: %w", err)
	}
	return repositoryRoot, dataRoot, nil
}

func planningStageWriteConfig(root, dataRoot, transactionID, command string, opts planningStageWriteOptions) lifecycleTransactionConfig {
	return lifecycleTransactionConfig{
		TransactionID: transactionID,
		Command:       command,
		Allowlist: lifecycleTransactionAllowlist{
			RepositoryRoot: root, LifecycleDataRoot: dataRoot,
		},
		Fault: opts.Fault, Rename: opts.Rename,
	}
}

func planningStageFinalizationConfig(root, dataRoot string, manifest planningStageManifest, opts planningStageWriteOptions) (lifecycleTransactionConfig, error) {
	if manifest.ExpectedCaste == planningStageCasteRouteSetter {
		transactionID, err := planningTimelineTransactionID(manifest.RunID, manifest.ID)
		if err != nil {
			return lifecycleTransactionConfig{}, err
		}
		return planningStageWriteConfig(root, dataRoot, transactionID, "planning-timeline-append", opts), nil
	}
	digest, err := jsonSHA256(struct {
		RunID      string `json:"run_id"`
		ManifestID string `json:"manifest_id"`
	}{manifest.RunID, manifest.ID})
	if err != nil {
		return lifecycleTransactionConfig{}, err
	}
	return planningStageWriteConfig(root, dataRoot, "planning-stage-finalize-"+digest[:24], "planning-stage-finalize", opts), nil
}

func resumePlanningStageWrite(config lifecycleTransactionConfig, expected map[string][]byte, manifestID string) (bool, error) {
	pending, err := planningTimelineTransactionHasIntent(config)
	if err != nil {
		return false, err
	}
	if !pending {
		return false, nil
	}
	matches, err := planningTimelinePendingIntentMatches(config, expected)
	if err != nil {
		return false, err
	}
	if !matches {
		return false, &planningStageReceiptConflictError{ManifestID: manifestID, Detail: "durable transaction intent contains different stage bytes"}
	}
	if _, err := resumeLifecycleTransaction(config); err != nil {
		return false, err
	}
	return true, nil
}

func planningStageLifecycleReceiptExists(config lifecycleTransactionConfig) bool {
	tx, err := beginLifecycleTransaction(config)
	if err != nil {
		return false
	}
	info, err := os.Lstat(filepath.Join(tx.journalPath(), "receipt.json"))
	return err == nil && info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0
}

func validatePersistedPlanningStageManifest(root string, manifest planningStageManifest) error {
	stored, err := loadPlanningStageManifest(root, manifest.RunID, manifest.ID)
	if err != nil {
		return err
	}
	if stored.ContentHash != manifest.ContentHash {
		return &planningStageReceiptConflictError{ManifestID: manifest.ID, Detail: "persisted manifest content hash differs"}
	}
	storedBytes, _ := marshalPlanningStageJSON(stored)
	wantBytes, _ := marshalPlanningStageJSON(manifest)
	if !bytes.Equal(storedBytes, wantBytes) {
		return &planningStageReceiptConflictError{ManifestID: manifest.ID, Detail: "persisted manifest bytes differ"}
	}
	return nil
}

func loadPlanningStageManifest(root, runID, manifestID string) (planningStageManifest, error) {
	empty := planningStageManifest{}
	content, ok, err := readOptionalPlanningStageFile(root, planningStageManifestRepositoryPath(runID, manifestID))
	if err != nil {
		return empty, err
	}
	if !ok {
		return empty, fmt.Errorf("planning stage manifest %q is missing: %w", manifestID, os.ErrNotExist)
	}
	manifest, err := decodePlanningStageManifest(content)
	if err != nil {
		return empty, err
	}
	if manifest.RunID != runID || manifest.ID != manifestID {
		return empty, fmt.Errorf("planning stage manifest path identity conflicts with its payload")
	}
	return manifest, nil
}

func loadPlanningStageState(root, runID string) (planningStageState, error) {
	empty := planningStageState{}
	content, ok, err := readOptionalPlanningStageFile(root, planningStageStateRepositoryPath(runID))
	if err != nil {
		return empty, err
	}
	if !ok {
		return empty, fmt.Errorf("planning stage state for %q is missing: %w", runID, os.ErrNotExist)
	}
	var state planningStageState
	if err := decodePlanningStageJSON(content, &state); err != nil {
		return empty, fmt.Errorf("decode planning stage state: %w", err)
	}
	if state.RunID != runID || !state.Stage.valid() {
		return empty, fmt.Errorf("planning stage state has invalid run or stage identity")
	}
	if err := validatePlanningStageAuthority(state); err != nil {
		return empty, err
	}
	return state, nil
}

func loadPlanningStageOutput(root string, manifest planningStageManifest) (planningStageOutputReference, []byte, error) {
	empty := planningStageOutputReference{}
	outputPath := planningStageOutputRepositoryPath(manifest)
	content, ok, err := readOptionalPlanningStageFile(root, outputPath)
	if err != nil {
		return empty, nil, err
	}
	if !ok {
		return empty, nil, fmt.Errorf("planning stage output for manifest %q is missing: %w", manifest.ID, os.ErrNotExist)
	}
	return planningStageOutputReference{Path: outputPath, ContentHash: planningStageBytesHash(content)}, content, nil
}

func readOptionalPlanningStageFile(root, repositoryPath string) ([]byte, bool, error) {
	if err := validatePlanningArtifactPath(repositoryPath); err != nil {
		return nil, false, err
	}
	fullPath := filepath.Join(root, filepath.FromSlash(repositoryPath))
	info, err := os.Lstat(fullPath)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, false, fmt.Errorf("planning stage artifact %q must be a regular non-symlink file", repositoryPath)
	}
	content, err := readStableLifecycleRegularFile(fullPath, info)
	if err != nil {
		return nil, false, err
	}
	return content, true, nil
}

func planningStageManifestRepositoryPath(runID, manifestID string) string {
	return path.Join(".aether", "data", "planning", runID, "manifests", manifestID+".json")
}

func planningStageOutputRepositoryPath(manifest planningStageManifest) string {
	return path.Join(".aether", "data", "planning", manifest.RunID, "outputs",
		fmt.Sprintf("pass-%04d-%s-%s.json", manifest.Pass, manifest.ExpectedCaste, manifest.ContentHash[:16]))
}

func planningStageReceiptRepositoryPath(runID, receiptID string) string {
	return path.Join(".aether", "data", "planning", runID, "receipts", receiptID+".json")
}

func planningStageReceiptIndexRepositoryPath(runID string) string {
	return path.Join(".aether", "data", "planning", runID, "stage-receipts.json")
}

func planningStageStateRepositoryPath(runID string) string {
	return path.Join(".aether", "data", "planning", runID, "stage-state.json")
}

func planningStageDataRelativePath(repositoryPath string) string {
	return strings.TrimPrefix(repositoryPath, ".aether/data/")
}

func planningStageBytesHash(content []byte) string {
	digest := sha256.Sum256(content)
	return hex.EncodeToString(digest[:])
}

func marshalPlanningStageJSON(value any) ([]byte, error) {
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(content, '\n'), nil
}

func decodePlanningStageJSON(content []byte, destination any) error {
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
