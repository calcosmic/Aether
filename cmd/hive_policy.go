package cmd

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

const hivePolicyEnv = "AETHER_HIVE_POLICY"

type hiveRuntimePolicy string

const (
	hivePolicyOff     hiveRuntimePolicy = "off"
	hivePolicyRead    hiveRuntimePolicy = "read"
	hivePolicyPromote hiveRuntimePolicy = "promote"
)

// hivePolicyWarnOnce guards the unrecognized-AETHER_HIVE_POLICY-value stderr
// warning so repeated policy lookups within one process only warn once.
var hivePolicyWarnOnce sync.Once

// currentHiveRuntimePolicy resolves AETHER_HIVE_POLICY to the runtime hive
// policy. D-02: there is exactly one control surface — no second consent
// mechanism may veto it. Unset (or explicitly empty) means promote: hive
// wisdom reaches worker context and high-confidence instincts promote to the
// hive at seal, by default (D-01). Explicit "off" disables both and must
// keep working even after the default flips — it gets its own case rather
// than falling through the default branch (Pitfall 4). Unrecognized values
// fail safe to "off" rather than silently widening cross-repo data flow, and
// warn once on stderr naming the offending value so the failure is
// discoverable (RESEARCH.md assumption A3, T-162-04).
func currentHiveRuntimePolicy() hiveRuntimePolicy {
	raw := os.Getenv(hivePolicyEnv)
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "":
		return hivePolicyPromote
	case "off":
		return hivePolicyOff
	case "read", "inject", "on":
		return hivePolicyRead
	case "promote", "full":
		return hivePolicyPromote
	default:
		hivePolicyWarnOnce.Do(func() {
			fmt.Fprintf(os.Stderr, "AETHER_HIVE_POLICY=%q is not recognized (expected off|read|promote); treating as off\n", raw)
		})
		return hivePolicyOff
	}
}

func automaticHiveReadEnabled() bool {
	policy := currentHiveRuntimePolicy()
	return policy == hivePolicyRead || policy == hivePolicyPromote
}

func automaticHivePromotionEnabled() bool {
	return currentHiveRuntimePolicy() == hivePolicyPromote
}
