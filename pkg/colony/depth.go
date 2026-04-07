package colony

// DepthBudget returns (contextChars, skillsChars) for the given depth level.
// Per D-03: progressive (non-linear) scaling — deeper builds get disproportionately
// more context to support extra specialists.
func DepthBudget(d ColonyDepth) (context int, skills int) {
	switch d {
	case DepthLight:
		return 4000, 4000
	case DepthStandard:
		return 8000, 8000
	case DepthDeep:
		return 16000, 12000
	case DepthFull:
		return 24000, 16000
	default:
		return 8000, 8000
	}
}
