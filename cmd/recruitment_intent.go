package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// recruitmentSchemaVersion is the wire-shape version for every recruitment
// artifact this plan and its successors write (intent, decision, result).
// Plan 203-03 grows recruitmentIntent's field set under this SAME version
// discipline -- see 203-CLASSIC-SYNTHESIS.md SYN-203-01.
const recruitmentSchemaVersion = "recruitment/v1"

// Bounds named per 203-03-PLAN.md Task 1. These are declared constants, not
// magic numbers, so a future reviewer can find and reason about them in one
// place.
const (
	recruitmentObjectiveMaxChars = 2000
	recruitmentReasonMaxChars    = 1000
	recruitmentMaxEvidence       = 20
	recruitmentMaxDeclaredPaths  = 50
)

// recruitmentIntentRetention bounds recruitment/intents.json exactly the way
// pkg/agent/spawn_tree.go's trimSpawnRuns bounds spawn-runs.json and
// cmd/codex_dispatch_contract.go's pruneWorkerHandoffRecords bounds
// worker-handoffs.json -- a fixed entry-count cap enforced on every write,
// oldest pruned first, so an append-only intent ledger never grows without
// bound.
const recruitmentIntentRetention = 200

// The two accepted Urgency values (BIO-01). recruitmentUrgencies is the
// completeness accessor, mirroring ColonyLiveTopics()'s own
// enumerate-everything convention, so a caller (or a test) never has to
// hardcode this pair a second time.
const (
	recruitmentUrgencyRoutine  = "routine"
	recruitmentUrgencyBlocking = "blocking"
)

func recruitmentUrgencies() []string {
	return []string{recruitmentUrgencyRoutine, recruitmentUrgencyBlocking}
}

// The fixed reason-class vocabulary validateRecruitmentIntent (and, from
// plan 203-06 onward, the full admission gate) returns -- mirroring
// spawnDecisionResult's own fixed-string convention (cmd/spawn.go) so every
// caller can switch on Reason reliably regardless of which layer refused.
// Never a bare boolean (RESEARCH.md Pattern 2, CONTEXT.md D-06).
const (
	recruitmentReasonSchema     = "schema"
	recruitmentReasonParent     = "parent"
	recruitmentReasonCaste      = "caste"
	recruitmentReasonCapability = "capability"
	recruitmentReasonObjective  = "objective"
	recruitmentReasonReason     = "reason"
	recruitmentReasonEvidence   = "evidence"
	recruitmentReasonPermission = "permission"
	recruitmentReasonUrgency    = "urgency"
	recruitmentReasonScope      = "scope"
	recruitmentReasonCost       = "cost"
)

// recruitmentIntent is the wire shape a worker's recruitment request carries
// through this runtime. The first nine fields are the tracer's own minimal
// shape (203-02-PLAN.md) and are UNCHANGED here, including their lack of
// JSON tags -- plan 203-03 only ADDS fields, never renames or re-tags an
// existing one. It deliberately mirrors spawnDecisionInput's field
// discipline (cmd/spawn.go) -- exported-style fields, no behavior, no
// money-shaped field anywhere on this type -- because BIO-02's admission
// gate consumes it through the SAME spawnCanSpawnDecision chokepoint an
// ordinary spawn already uses, never a second one (SYN-203-02).
//
// ParentName/ParentDepth/DepthIsAuthoritative mirror spawnDecisionInput's own
// RequesterName/RequesterDepth/DepthIsAuthoritative naming and meaning: the
// WOULD-BE PARENT, not the child being proposed. AttemptID is this
// recruitment's own attempt identity (also used, unchanged, as
// recruitmentResult's RecruitmentID).
//
// The new fields below complete BIO-01's full field set: ParentAttemptID and
// ParentRunID describe WHICH attempt and run of the parent is asking (parent
// identity, distinct from the parent's plain name); Capability narrows the
// requested Caste; Evidence names supporting lifecycle evidence by
// identifier; Permission is the resolved (never caller-trusted) permission
// promise; Urgency is one of recruitmentUrgencies(); DeclaredPaths is the
// requested scope; CostSlots/CostSeconds are cost limits expressed as helper
// slots against the existing whole-run tree budget and a wall-clock second
// count -- never a money amount (must_haves: no recruitment type carries a
// money field).
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

	IntentID        string
	ParentAttemptID string
	ParentRunID     string
	Capability      string
	Evidence        []string
	Permission      codex.PermissionProfile
	Urgency         string
	DeclaredPaths   []string
	CostSlots       int
	CostSeconds     int
}

// recruitmentDecisionResult mirrors spawnDecisionResult's exact shape
// (cmd/spawn.go) so a caller can switch on Reason using the same
// fixed-string convention whether the decision came from an ordinary spawn,
// intent validation, or a recruitment admission gate.
type recruitmentDecisionResult struct {
	Allowed bool
	Reason  string
	Detail  string
}

// validateRecruitmentIntent refuses by name (RESEARCH.md Pattern 2): every
// deny carries one of the eleven fixed reason classes above and a human
// Detail sentence naming the offending value, following
// spawnAncestorCycleReason's sentence style. Checks run in a fixed order and
// the first failing check wins -- callers needing every problem at once
// should not assume this reports more than one.
//
// Every unreadable-state branch here denies, never allows (D-19 fail-closed,
// cmd/spawn_ancestor.go:95-98's discipline, copied verbatim): a store error
// while resolving recorded evidence is a refusal, not a silent pass.
func validateRecruitmentIntent(in recruitmentIntent) recruitmentDecisionResult {
	if in.SchemaVersion != recruitmentSchemaVersion {
		return recruitmentDecisionResult{
			Allowed: false,
			Reason:  recruitmentReasonSchema,
			Detail: fmt.Sprintf(
				"unrecognised schema version %q; this runtime only accepts %q",
				in.SchemaVersion, recruitmentSchemaVersion,
			),
		}
	}

	// A parent name that resolves to no recorded spawn entry is refused
	// rather than trusted (must_haves) -- a coordinator sentinel
	// (spawnParentIsRoot, cmd/spawn.go) is the one legitimate case with no
	// spawn-tree entry of its own.
	if !spawnParentIsRoot(in.ParentName) && !in.DepthIsAuthoritative {
		return recruitmentDecisionResult{
			Allowed: false,
			Reason:  recruitmentReasonParent,
			Detail: fmt.Sprintf(
				"%q resolves to no recorded spawn entry and is not a coordinator sentinel; a parent's depth is never trusted from a self-declared claim",
				in.ParentName,
			),
		}
	}

	if strings.TrimSpace(in.Caste) == "" {
		return recruitmentDecisionResult{
			Allowed: false,
			Reason:  recruitmentReasonCaste,
			Detail:  "a recruitment intent must name the requested caste",
		}
	}

	if length := len(in.Objective); length > recruitmentObjectiveMaxChars {
		return recruitmentDecisionResult{
			Allowed: false,
			Reason:  recruitmentReasonObjective,
			Detail: fmt.Sprintf(
				"objective is %d characters, past the limit of %d",
				length, recruitmentObjectiveMaxChars,
			),
		}
	}
	// Worker-authored objective and reason text is sanitised before it is
	// stored, because it is replayed verbatim into later worker briefs
	// (must_haves). SanitizeSignalContent's own returned value is discarded
	// here -- validateRecruitmentIntent only judges whether the content
	// would be accepted; the sanitized text itself is applied to the stored
	// copy by sanitizedRecruitmentIntentCopy once validation has passed.
	if _, err := colony.SanitizeSignalContent(in.Objective); err != nil {
		return recruitmentDecisionResult{Allowed: false, Reason: recruitmentReasonObjective, Detail: err.Error()}
	}

	if length := len(in.Reason); length > recruitmentReasonMaxChars {
		return recruitmentDecisionResult{
			Allowed: false,
			Reason:  recruitmentReasonReason,
			Detail: fmt.Sprintf(
				"reason is %d characters, past the limit of %d",
				length, recruitmentReasonMaxChars,
			),
		}
	}
	if _, err := colony.SanitizeSignalContent(in.Reason); err != nil {
		return recruitmentDecisionResult{Allowed: false, Reason: recruitmentReasonReason, Detail: err.Error()}
	}

	// Capability is optional (BIO-01 pairs it with Caste, but an empty
	// Capability simply means the caste's own definition is specific
	// enough) -- when present, it is worker-authored free text exactly like
	// Objective and Reason, so it is sanitised the same way rather than
	// trusted unchecked.
	if _, err := colony.SanitizeSignalContent(in.Capability); err != nil {
		return recruitmentDecisionResult{Allowed: false, Reason: recruitmentReasonCapability, Detail: err.Error()}
	}

	if len(in.Evidence) > recruitmentMaxEvidence {
		return recruitmentDecisionResult{
			Allowed: false,
			Reason:  recruitmentReasonEvidence,
			Detail: fmt.Sprintf(
				"%d evidence identifiers were named, past the limit of %d",
				len(in.Evidence), recruitmentMaxEvidence,
			),
		}
	}
	if len(in.Evidence) > 0 {
		// loadWorkerHandoffRecords (cmd/codex_dispatch_contract.go) only
		// treats a missing file as "no handoffs recorded yet" when the
		// error it sees is exactly os.ErrNotExist -- but store.ReadFile
		// always wraps its own error via fmt.Errorf("...: %w", err), and
		// os.IsNotExist does not unwrap generic %w chains (it only peels
		// *PathError/*LinkError/*SyscallError). Calling it directly against
		// a genuinely absent file would therefore always take the fail-closed
		// branch below, refusing every evidence-carrying recruitment on a
		// fresh colony that has never recorded a single handoff. Checking
		// existence first distinguishes "nothing recorded yet" (an empty,
		// definite answer -- no evidence identifier can possibly resolve)
		// from "the file exists but cannot be read" (a genuinely ambiguous
		// state, which still fails closed).
		var records []workerHandoffRecord
		exists, existsErr := store.FileExists(workerHandoffsPath)
		if existsErr != nil {
			// D-19 fail-closed: an unreadable evidence store must deny,
			// never allow.
			return recruitmentDecisionResult{
				Allowed: false,
				Reason:  recruitmentReasonEvidence,
				Detail:  fmt.Sprintf("recorded evidence unreadable (%v): refusing to spawn", existsErr),
			}
		}
		if exists {
			var err error
			records, err = loadWorkerHandoffRecords()
			if err != nil {
				// D-19 fail-closed: an unreadable evidence store must deny,
				// never allow.
				return recruitmentDecisionResult{
					Allowed: false,
					Reason:  recruitmentReasonEvidence,
					Detail:  fmt.Sprintf("recorded evidence unreadable (%v): refusing to spawn", err),
				}
			}
		}
		known := make(map[string]bool, len(records))
		for _, record := range records {
			known[record.ID] = true
		}
		for _, id := range in.Evidence {
			if !known[id] {
				return recruitmentDecisionResult{
					Allowed: false,
					Reason:  recruitmentReasonEvidence,
					Detail:  fmt.Sprintf("evidence %q resolves to no recorded lifecycle evidence", id),
				}
			}
		}
	}

	// Permission is resolved from PermissionProfileForCaste for the
	// requested caste rather than accepting whatever the caller sent
	// (must_haves) -- an explicit caller-supplied profile Name that
	// disagrees with the resolved one is refused, naming both.
	canonicalPermission := codex.PermissionProfileForCaste(in.Caste)
	if in.Permission.Name != "" && in.Permission.Name != canonicalPermission.Name {
		return recruitmentDecisionResult{
			Allowed: false,
			Reason:  recruitmentReasonPermission,
			Detail: fmt.Sprintf(
				"requested permission profile %q disagrees with the resolved profile %q for caste %q",
				in.Permission.Name, canonicalPermission.Name, in.Caste,
			),
		}
	}

	validUrgency := false
	for _, u := range recruitmentUrgencies() {
		if in.Urgency == u {
			validUrgency = true
			break
		}
	}
	if !validUrgency {
		return recruitmentDecisionResult{
			Allowed: false,
			Reason:  recruitmentReasonUrgency,
			Detail: fmt.Sprintf(
				"urgency %q is not one of the accepted values (%s)",
				in.Urgency, strings.Join(recruitmentUrgencies(), ", "),
			),
		}
	}

	if len(in.DeclaredPaths) > recruitmentMaxDeclaredPaths {
		return recruitmentDecisionResult{
			Allowed: false,
			Reason:  recruitmentReasonScope,
			Detail: fmt.Sprintf(
				"%d declared paths were named, past the limit of %d",
				len(in.DeclaredPaths), recruitmentMaxDeclaredPaths,
			),
		}
	}

	// Cost is a slot count and a second count, never money (must_haves): a
	// CostSlots of zero, negative, or greater than the whole-run tree
	// budget ceiling (spawnTreeBudgetMax, cmd/spawn_budget.go) is refused --
	// D-12 requires recruits draw from that SAME ledger, so nothing here
	// consults a second dial.
	if in.CostSlots <= 0 || in.CostSlots > spawnTreeBudgetMax {
		return recruitmentDecisionResult{
			Allowed: false,
			Reason:  recruitmentReasonCost,
			Detail: fmt.Sprintf(
				"cost-slots %d is outside the allowed range of 1 to %d (the whole-run helper budget ceiling)",
				in.CostSlots, spawnTreeBudgetMax,
			),
		}
	}
	if in.CostSeconds <= 0 {
		return recruitmentDecisionResult{
			Allowed: false,
			Reason:  recruitmentReasonCost,
			Detail:  fmt.Sprintf("cost-seconds %d must be a positive number of seconds", in.CostSeconds),
		}
	}

	return recruitmentDecisionResult{Allowed: true}
}

// sanitizedRecruitmentIntentCopy returns a copy of in with its
// worker-authored free-text fields (Objective, Reason, Capability) replaced
// by their sanitized form, so the durable record and any downstream
// dispatch never carry raw, unsanitized worker text. Called only after
// validateRecruitmentIntent has already confirmed each field sanitizes
// cleanly -- a sanitize failure here (which should not occur post-validation)
// is treated as "leave the original value", never as a reason to panic or
// silently drop the field.
func sanitizedRecruitmentIntentCopy(in recruitmentIntent) recruitmentIntent {
	out := in
	if v, err := colony.SanitizeSignalContent(in.Objective); err == nil {
		out.Objective = v
	}
	if v, err := colony.SanitizeSignalContent(in.Reason); err == nil {
		out.Reason = v
	}
	if v, err := colony.SanitizeSignalContent(in.Capability); err == nil {
		out.Capability = v
	}
	return out
}

// recruitmentIntentsPath is the store-relative path recordRecruitmentIntent
// persists to. New data file, per this plan's frontmatter.
const recruitmentIntentsPath = "recruitment/intents.json"

// recruitmentIntentRecord is the durable envelope recordRecruitmentIntent
// writes: the validated (and, when allowed, sanitized) intent, when it was
// created, and -- once known -- the admission decision and when it was
// decided. must_haves: "Every emitted intent is durably recorded with its
// identifier before any admission decision is taken, so a refusal is as
// recoverable as an admission" -- Decision/DecidedAt therefore start empty
// and are attached afterward by recordRecruitmentDecision, never by
// re-calling recordRecruitmentIntent itself.
type recruitmentIntentRecord struct {
	Intent    recruitmentIntent          `json:"intent"`
	Decision  *recruitmentDecisionResult `json:"decision,omitempty"`
	CreatedAt string                     `json:"created_at"`
	DecidedAt string                     `json:"decided_at,omitempty"`
}

// recruitmentIntentsFile is the on-disk container at recruitmentIntentsPath.
type recruitmentIntentsFile struct {
	Entries []recruitmentIntentRecord `json:"entries"`
}

// errRecruitmentIntentAlreadyRecorded is the internal replay sentinel
// recordRecruitmentIntent returns from its own UpdateJSONAtomically mutate
// closure to abort the write on a replay -- UpdateJSONAtomically's contract
// is "if mutate returns an error, no write occurs" (pkg/storage), which is
// exactly the "leaves the file byte-identical" guarantee a replay requires.
// Mirrors cmd/recruitment_result.go's errRecruitmentResultAlreadyBound.
var errRecruitmentIntentAlreadyRecorded = errors.New("recruitment intent already recorded")

// recordRecruitmentIntent stores record under recruitment/intents.json,
// keyed on record.Intent.IntentID. A second call carrying an already-stored
// IntentID returns the STORED record and mutates nothing -- recording an
// intent is a pure, idempotent create; attaching a decision afterward is
// recordRecruitmentDecision's separate job. Entries beyond
// recruitmentIntentRetention are pruned, oldest first, on every write that
// actually appends.
func recordRecruitmentIntent(record recruitmentIntentRecord) (recruitmentIntentRecord, error) {
	if store == nil {
		return recruitmentIntentRecord{}, fmt.Errorf("no store initialized")
	}
	if strings.TrimSpace(record.Intent.IntentID) == "" {
		return recruitmentIntentRecord{}, fmt.Errorf("recruitment intent requires a non-empty IntentID")
	}

	var bound recruitmentIntentRecord
	var file recruitmentIntentsFile
	err := store.UpdateJSONAtomically(recruitmentIntentsPath, &file, func() error {
		for _, existing := range file.Entries {
			if existing.Intent.IntentID == record.Intent.IntentID {
				bound = existing
				return errRecruitmentIntentAlreadyRecorded
			}
		}
		file.Entries = append(file.Entries, record)
		file.Entries = trimRecruitmentIntents(file.Entries)
		bound = record
		return nil
	})
	if err != nil && !errors.Is(err, errRecruitmentIntentAlreadyRecorded) {
		return recruitmentIntentRecord{}, err
	}
	return bound, nil
}

// recordRecruitmentDecision attaches decision to the already-recorded intent
// named by intentID, completing task2's "validate, record, then decide,
// then update the record with the decision" sequence. The FIRST decision
// recorded for a given IntentID is final: a second call for the same
// IntentID leaves the stored decision untouched, matching
// recordRecruitmentIntent's own idempotent-create discipline one layer up.
// An IntentID with no recorded intent yet is an error, not a silent create
// -- recordRecruitmentIntent must always run first, on this exact IntentID,
// before this is ever reached.
func recordRecruitmentDecision(intentID string, decision recruitmentDecisionResult, decidedAt string) (recruitmentIntentRecord, error) {
	if store == nil {
		return recruitmentIntentRecord{}, fmt.Errorf("no store initialized")
	}
	if strings.TrimSpace(intentID) == "" {
		return recruitmentIntentRecord{}, fmt.Errorf("recruitment decision requires a non-empty IntentID")
	}

	var bound recruitmentIntentRecord
	var file recruitmentIntentsFile
	err := store.UpdateJSONAtomically(recruitmentIntentsPath, &file, func() error {
		for i := range file.Entries {
			if file.Entries[i].Intent.IntentID != intentID {
				continue
			}
			if file.Entries[i].Decision == nil {
				d := decision
				file.Entries[i].Decision = &d
				file.Entries[i].DecidedAt = decidedAt
			}
			bound = file.Entries[i]
			return nil
		}
		return fmt.Errorf("no recorded intent for IntentID %q", intentID)
	})
	if err != nil {
		return recruitmentIntentRecord{}, err
	}
	return bound, nil
}

// trimRecruitmentIntents keeps at most the most recent recruitmentIntentRetention
// entries, oldest first dropped -- mirrors pkg/agent/spawn_tree.go's
// trimSpawnRuns exactly (entries are appended in chronological order, so the
// tail slice is the most recent N).
func trimRecruitmentIntents(entries []recruitmentIntentRecord) []recruitmentIntentRecord {
	if len(entries) <= recruitmentIntentRetention {
		return entries
	}
	return append([]recruitmentIntentRecord{}, entries[len(entries)-recruitmentIntentRetention:]...)
}
