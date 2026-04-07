package colony

// GranularityRange returns the (min, max) phase count for the given granularity.
// The default for unknown values is sprint (1-3).
func GranularityRange(g PlanGranularity) (min int, max int) {
	switch g {
	case GranularitySprint:
		return 1, 3
	case GranularityMilestone:
		return 4, 7
	case GranularityQuarter:
		return 8, 12
	case GranularityMajor:
		return 13, 20
	default:
		return 1, 3
	}
}
