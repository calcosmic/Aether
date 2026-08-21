package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// T-191-04-02 (191-04-PLAN.md). colony/policies/review-depth.yaml was
// deleted in this plan's Task 1: every field it defined (heavy_keywords,
// security_risk_keywords, blast_radius_keywords, smart_default_reasons) was
// confirmed, field-by-field, to already match cmd/review_depth.go's
// compiled fallback slices/consts, and TestReviewDepthPolicyFieldsSurviveDeletion
// (cmd/review_depth_test.go) proves this permanently. If the file reappears,
// it is either a stray revert or a real new divergence between a future
// edit to the fallbacks and what the file would say -- either way, this
// must fail loudly and by name, not be silently re-read by
// loadReviewDepthPolicy()'s still-present (D-04, kept as a permanent
// fallback path) CWD-relative loader.
//
// Tier 1 per 191-PATTERNS.md "The Ratchet House Style": a plain os.Stat
// existence check is correct and sufficient for a single file's presence --
// no AST scanner needed.

func TestReviewDepthPolicyDoesNotReappear(t *testing.T) {
	root, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	p := filepath.Join(root, "colony", "policies", "review-depth.yaml")
	if _, err := os.Stat(p); err == nil {
		t.Errorf("colony/policies/review-depth.yaml has reappeared at %s -- this file was ruled zero-added-value and deleted in Phase 191 Plan 04 (191-CONTEXT.md D-04): every field it defined already matched cmd/review_depth.go's compiled fallbacks, proven by TestReviewDepthPolicyFieldsSurviveDeletion. If it is back, either fold any new divergent value into review_depth.go's fallbacks and update that regression test with fresh evidence, or delete it again -- never leave it as an untested, silently-read CWD-relative file", p)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat colony/policies/review-depth.yaml: %v", err)
	}
}
