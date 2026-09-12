package cmd

// recruitmentSchemaVersion is the wire-shape version for every recruitment
// artifact this plan and its successors write (intent, decision, result).
// Plan 203-03 grows recruitmentIntent's field set under this SAME version
// discipline -- see 203-CLASSIC-SYNTHESIS.md SYN-203-01. Do not add BIO-01's
// remaining fields (evidence, permissions, urgency, scope, cost limit) here;
// this tracer intentionally carries only the minimal end-to-end shape.
const recruitmentSchemaVersion = "recruitment/v1"

// recruitmentIntent is the minimal wire shape a worker's recruitment request
// carries through this tracer. It deliberately mirrors spawnDecisionInput's
// field discipline (cmd/spawn.go) -- exported-style fields, no behavior, no
// currency-shaped field -- because BIO-02's admission gate (cmd/recruitment.go)
// consumes it through the SAME spawnCanSpawnDecision chokepoint an ordinary
// spawn already uses, never a second one (SYN-203-02).
//
// ParentName/ParentDepth/DepthIsAuthoritative mirror spawnDecisionInput's own
// RequesterName/RequesterDepth/DepthIsAuthoritative naming and meaning: the
// WOULD-BE PARENT, not the child being proposed.
type recruitmentIntent struct {
	SchemaVersion        string
	ParentName           string
	ParentDepth          int
	DepthIsAuthoritative bool
	AttemptID            string
	Caste                string
	Objective            string
	Reason               string
	Workspace            string
}

// recruitmentDecisionResult mirrors spawnDecisionResult's exact shape
// (cmd/spawn.go) so a caller can switch on Reason using the same
// fixed-string convention ("depth", "budget", "ancestor-cycle", ...)
// whether the decision came from an ordinary spawn or a recruitment. This
// tracer does not itself construct one -- recruitCmd reads the answer
// straight off spawnCanSpawnDecision's own spawnDecisionResult -- but the
// type is declared here as the recruitment-side name plan 203-06 (BIO-02's
// full admission gate) will return once it adds the permission/path/cost/
// duplicate checks.
type recruitmentDecisionResult struct {
	Allowed bool
	Reason  string
	Detail  string
}
