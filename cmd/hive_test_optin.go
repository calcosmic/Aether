package cmd

import (
	"testing"
)

// enableHiveForTest opts a test colony into cross-colony wisdom retrieval.
//
// Hive retrieval is off by default behind two independent gates: a machine-wide
// policy (AETHER_HIVE_POLICY, see hive_policy.go) and a per-colony consent file
// (hiveRetrievalOptedIn, see hive.go). Both must be satisfied, by design — the
// policy decides whether the feature exists at all, consent decides whether this
// particular repository receives other repositories' wisdom.
//
// Tests that assert on hive content must opt in explicitly rather than rely on a
// default. That is the point: a test which only passes because retrieval happens
// implicitly would not have caught the state this codebase was actually in,
// where the hub held 67 malformed entries that no repository had consented to.
//
// t.Setenv restores the previous value automatically at test end.
func enableHiveForTest(t *testing.T) {
	t.Helper()
	t.Setenv(hivePolicyEnv, "read")
	if err := writeHiveRetrievalConsent(true); err != nil {
		t.Fatalf("opt colony into hive retrieval: %v", err)
	}
}
