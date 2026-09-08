package cmd

import (
	"strconv"
	"strings"
)

// joinInts is shared by the automatic phase-research and visual reporting
// paths. The former public plan-research-approve command was retired in Phase
// 200: selecting a planning preset now authorizes routine read-only research,
// while material owner choices flow through receipt-bound decision batches.
func joinInts(ids []int) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.Itoa(id)
	}
	return strings.Join(parts, ",")
}
