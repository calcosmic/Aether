package codex

import "strings"

const (
	// TaskReceiptStatusCompleted reports task-specific implementation evidence.
	TaskReceiptStatusCompleted = "completed"
	// TaskReceiptStatusCompletedNoChange reports task-specific proof that the
	// required behavior already existed without fabricating an edit.
	TaskReceiptStatusCompletedNoChange = "completed_no_change"
)

// TaskReceipt is additive task-specific completion evidence for a worker job.
// Covered task IDs remain assignment scope; this transport does not decide
// whether a receipt earns completion credit or redefine manifest criteria.
type TaskReceipt struct {
	TaskID        string        `json:"task_id"`
	Status        string        `json:"status"`
	Summary       string        `json:"summary"`
	FilesCreated  []string      `json:"files_created"`
	FilesModified []string      `json:"files_modified"`
	TestsWritten  []string      `json:"tests_written"`
	Handoff       WorkerHandoff `json:"handoff"`
}

// NormalizeTaskReceipts is the single normalization every lane runs before a
// receipt is judged: it lowercases the status, makes claimed paths
// repository-relative, and runs the receipt's handoff through the same
// normalizer the worker path uses (which maps accepted aliases such as
// "passed" to "pass" and "not run" to "not_run").
//
// It is exported because the wrapper/external lane decodes receipts straight
// off a submitted completion packet and never went through the worker path, so
// an identical receipt was credited on one lane and refused on the other
// (WR-08, 195-REVIEW.md).
func NormalizeTaskReceipts(root string, receipts []TaskReceipt) []TaskReceipt {
	return normalizeTaskReceipts(root, receipts)
}

func normalizeTaskReceipts(root string, receipts []TaskReceipt) []TaskReceipt {
	if receipts == nil {
		return nil
	}
	normalized := make([]TaskReceipt, len(receipts))
	for i, receipt := range receipts {
		receipt.TaskID = strings.TrimSpace(receipt.TaskID)
		receipt.Status = strings.ToLower(strings.TrimSpace(receipt.Status))
		receipt.Summary = strings.TrimSpace(receipt.Summary)
		receipt.FilesCreated = normalizeClaimPaths(root, receipt.FilesCreated)
		receipt.FilesModified = normalizeClaimPaths(root, receipt.FilesModified)
		receipt.TestsWritten = normalizeClaimPaths(root, receipt.TestsWritten)
		receipt.Handoff = NormalizeWorkerHandoff(root, receipt.Handoff)
		normalized[i] = receipt
	}
	return normalized
}
