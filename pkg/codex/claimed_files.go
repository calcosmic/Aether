package codex

import "strings"

// AllClaimedFiles returns every distinct path a worker result claims, from
// ALL THREE places a worker can name one: the top-level files_created /
// files_modified / tests_written lists, the same three lists inside each
// task receipt, and the worker's own handoff changed_files.
//
// It exists because those three are not redundant copies of one list, and
// treating any one of them as authoritative silently discards work.
//
// Downstream report, CosmicDashboard on v1.0.75: a phase commit saved 20
// files and left 7 verified ones uncommitted, with no warning. A builder had
// been asked to correct one task and resent its report; the resend's
// top-level lists AND its handoff changed_files both named only the
// follow-up fix, while its earlier, already-completed work survived only in
// task_receipts. The resend-after-correction path legitimately produces that
// narrowed shape, so this is not a malformed report to reject -- it is a
// subset to union.
//
// Order is deterministic (first appearance, top-level then receipts then
// handoff) so callers that render or commit the list get stable output;
// blanks are dropped, duplicates collapse.
func AllClaimedFiles(result WorkerResult) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(result.FilesCreated)+len(result.FilesModified)+len(result.TestsWritten))
	add := func(paths []string) {
		for _, p := range paths {
			p = strings.TrimSpace(p)
			if p == "" || seen[p] {
				continue
			}
			seen[p] = true
			out = append(out, p)
		}
	}
	add(result.FilesCreated)
	add(result.FilesModified)
	add(result.TestsWritten)
	for _, receipt := range result.TaskReceipts {
		add(receipt.FilesCreated)
		add(receipt.FilesModified)
		add(receipt.TestsWritten)
		add(receipt.Handoff.ChangedFiles)
	}
	add(result.Handoff.ChangedFiles)
	return out
}
