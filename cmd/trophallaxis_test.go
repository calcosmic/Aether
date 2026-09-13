package cmd

// BIO-05 (203-10): the trophallaxis packet, its acknowledgement, and its
// decision join. Task 1 first proves the word "trophallaxis" no longer names
// two orphaned diagnostic/retry commands with no caller anywhere in the
// repository (cmd/testdata/orphan_allowlist.json's "unreviewed-pre-existing"
// entries for "aether trophallaxis-diagnose" and "aether trophallaxis-retry"
// are the retire-with-proof evidence). Tasks 2-3 add the real packet tests
// below this one.

import (
	"testing"
)

// TestTrophallaxisOrphanCommandsRetired proves the two retired command names
// no longer resolve through the real cobra command tree. This is the
// negative half of Task 1's behavior: "running either retired name reports
// an unknown command rather than a confusing partial success."
func TestTrophallaxisOrphanCommandsRetired(t *testing.T) {
	for _, leaf := range []string{"trophallaxis-diagnose", "trophallaxis-retry"} {
		target, _, err := rootCmd.Find([]string{leaf})
		if err == nil && target != nil && target != rootCmd {
			t.Errorf("%q still resolves via rootCmd.Find as %q -- the retired orphan was not removed from the registered command set", leaf, target.CommandPath())
		}
	}
}
