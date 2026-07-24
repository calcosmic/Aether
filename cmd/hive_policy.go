package cmd

import (
	"os"
	"strings"
)

const hivePolicyEnv = "AETHER_HIVE_POLICY"

type hiveRuntimePolicy string

const (
	hivePolicyOff     hiveRuntimePolicy = "off"
	hivePolicyRead    hiveRuntimePolicy = "read"
	hivePolicyPromote hiveRuntimePolicy = "promote"
)

func currentHiveRuntimePolicy() hiveRuntimePolicy {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(hivePolicyEnv))) {
	case "read", "inject", "on":
		return hivePolicyRead
	case "promote", "full":
		return hivePolicyPromote
	default:
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
