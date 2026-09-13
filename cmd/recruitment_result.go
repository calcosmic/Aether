package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

// recruitmentResultsPath is the store-relative path bindRecruitmentResult
// persists to. New data file, per this plan's frontmatter.
const recruitmentResultsPath = "recruitment/results.json"

// RecruitmentResultSchemaVersion is the wire-shape version every bound
// recruitment result carries, mirroring colony.LifecycleSchemaVersion's own
// "one version, additive fields only" discipline. A future incompatible
// change bumps this constant rather than silently reinterpreting an old
// record.
const RecruitmentResultSchemaVersion = "recruitment-result/v1"

// Recruitment terminal-status vocabulary (BIO-04). These are the only three
// values dispatchRecruitment (cmd/recruitment_dispatch.go) ever produces --
// "completed" on a clean exit, "failed" on a non-timeout error, "timeout"
// when the bounded deadline elapsed. bindRecruitmentResult refuses any other
// value by name rather than folding it onto the nearest neighbour, matching
// the Phase 196 ruling that an empty/unknown ledger row status is refused
// rather than silently reinterpreted (cmd/spend_ledger.go
// normalizeSpendRowStatus's own doc comment) -- a durable record must never
// state a wrong outcome.
const (
	RecruitmentTerminalStatusCompleted = "completed"
	RecruitmentTerminalStatusFailed    = "failed"
	RecruitmentTerminalStatusTimeout   = "timeout"
)

// recruitmentTerminalStatuses returns every declared terminal-status
// constant, mirroring the ColonyLiveTopics()/ColonyLiveEpisodeKinds()
// completeness convention (pkg/events/colony_live.go): a status added to the
// const block above must also be added here.
func recruitmentTerminalStatuses() []string {
	return []string{
		RecruitmentTerminalStatusCompleted,
		RecruitmentTerminalStatusFailed,
		RecruitmentTerminalStatusTimeout,
	}
}

func recruitmentTerminalStatusValid(status string) bool {
	for _, known := range recruitmentTerminalStatuses() {
		if known == status {
			return true
		}
	}
	return false
}

// recruitmentResult is the exactly-once completion record for one admitted
// recruitment, shaped like colony.PauseHandoff's own ID + Transaction +
// optional Receipt discipline (pkg/colony/lifecycle.go) -- the same pattern
// this codebase already trusts for replay-safe completion binding, reused
// verbatim rather than inventing a second idempotency mechanism (SYN-203-06).
//
// This binds every BIO-04 element in one record: the child, the intent, the
// dispatch, the execution generation, the terminal status, the evidence, the
// artifacts, the handoff and the usage.
type recruitmentResult struct {
	SchemaVersion string `json:"schema_version"`
	RecruitmentID string `json:"recruitment_id"`
	IntentID      string `json:"intent_id"`
	// DispatchID names the specific dispatchRecruitment invocation that
	// produced this result, distinct from IntentID (the request) and
	// RecruitmentID (the idempotency key) -- a durable record should be able
	// to say WHICH dispatch attempt this is, even though today exactly one
	// dispatch attempt exists per RecruitmentID.
	DispatchID string `json:"dispatch_id,omitempty"`
	ChildName  string `json:"child_name"`
	ParentName string `json:"parent_name"`
	// ExecutionGeneration is the durable build-run identity this result was
	// produced under (flagged planner assumption, 203-07-PLAN.md objective:
	// BIO-04 requires binding "execution generation" without defining it --
	// this plan assumes it is the same durable run identity spendRow.RunID
	// already names, "the attempt at this phase that this row was filed
	// by" -- not a per-child retry counter). A result whose generation is
	// older than the one already recorded for this RecruitmentID is refused;
	// see recruitmentGenerationConflict.
	ExecutionGeneration string `json:"execution_generation,omitempty"`
	TerminalStatus      string `json:"terminal_status"`
	Summary             string `json:"summary,omitempty"`
	// Evidence is the durable proof backing this result -- e.g. a file the
	// child produced, with a digest bindRecruitmentResult's caller computed
	// at bind time. classifyRecruitmentRecovery (cmd/recruitment_recovery.go)
	// re-checks each entry's digest against the file on disk at classify
	// time to detect tampering (the "altered" recovery class).
	Evidence []colony.LifecycleEvidence `json:"evidence,omitempty"`
	// Artifacts names file paths the child produced, distinct from Evidence:
	// an artifact is an output; evidence is proof. A path may appear in
	// both.
	Artifacts []string `json:"artifacts,omitempty"`
	// HandoffID joins this result to the worker-handoff record
	// persistDispatchWorkerHandoff already writes for the same child
	// (cmd/codex_dispatch_contract.go, worker-handoffs.json) -- a reference,
	// not a duplicate copy of that record.
	HandoffID string `json:"handoff_id,omitempty"`
	// UsageRowID names the child's own deterministic worker name (e.g.
	// "Mason-67"), never a shared agent/caste name -- the Phase 196
	// accounting ruling that several workers in one build can share an
	// agent name, and accounting must key on the per-worker one
	// (cmd/wrapper_usage_resolve.go's AgentNameByWorker doc comment). This
	// is the identifier a usage reader would key a WorkerUsage row on for
	// this child; it is deliberately ChildName, not Caste.
	UsageRowID string `json:"usage_row_id,omitempty"`
	// AdapterKind names which dispatch surface produced this result (e.g.
	// "subprocess" for the default leased-workspace child dispatch), for a
	// future reader distinguishing dispatch mechanisms without inferring it
	// from other fields.
	AdapterKind string                               `json:"adapter_kind,omitempty"`
	StartedAt   string                               `json:"started_at,omitempty"`
	EndedAt     string                               `json:"ended_at,omitempty"`
	Provenance  colony.RecoveryProvenance            `json:"provenance,omitempty"`
	Transaction colony.LifecycleTransactionReference `json:"transaction"`
	Receipt     *colony.LifecycleReceiptReference    `json:"receipt,omitempty"`
}

// recruitmentResultsFile is the on-disk container at recruitmentResultsPath.
type recruitmentResultsFile struct {
	Entries []recruitmentResult `json:"entries"`
}

// errRecruitmentResultAlreadyBound is the internal replay sentinel
// bindRecruitmentResult returns from its own UpdateJSONAtomically mutate
// closure to abort the write on a replay -- UpdateJSONAtomically's own
// contract is "if mutate returns an error, no write occurs" (pkg/storage),
// which is exactly the "mutates nothing" guarantee a replay requires. It is
// also used to abort a refused conflicting/stale bind attempt, which must
// write nothing either.
var errRecruitmentResultAlreadyBound = errors.New("recruitment result already bound")

// recruitmentResultContentSnapshot is the caller-supplied CONTENT of a
// recruitment result -- everything bindRecruitmentResult's replay/conflict
// comparison cares about. SchemaVersion, RecruitmentID, Transaction and
// Receipt are deliberately excluded: SchemaVersion is a wire-shape
// identifier, RecruitmentID is the lookup key (not content), and
// Transaction/Receipt are what binding itself produces, never caller-
// supplied content to compare.
type recruitmentResultContentSnapshot struct {
	IntentID            string                     `json:"intent_id"`
	DispatchID          string                     `json:"dispatch_id,omitempty"`
	ChildName           string                     `json:"child_name"`
	ParentName          string                     `json:"parent_name"`
	ExecutionGeneration string                     `json:"execution_generation,omitempty"`
	TerminalStatus      string                     `json:"terminal_status"`
	Summary             string                     `json:"summary,omitempty"`
	Evidence            []colony.LifecycleEvidence `json:"evidence,omitempty"`
	Artifacts           []string                   `json:"artifacts,omitempty"`
	HandoffID           string                     `json:"handoff_id,omitempty"`
	UsageRowID          string                     `json:"usage_row_id,omitempty"`
	AdapterKind         string                     `json:"adapter_kind,omitempty"`
	StartedAt           string                     `json:"started_at,omitempty"`
	EndedAt             string                     `json:"ended_at,omitempty"`
	Provenance          colony.RecoveryProvenance  `json:"provenance,omitempty"`
}

func recruitmentResultContentSnapshotOf(result recruitmentResult) recruitmentResultContentSnapshot {
	return recruitmentResultContentSnapshot{
		IntentID:            result.IntentID,
		DispatchID:          result.DispatchID,
		ChildName:           result.ChildName,
		ParentName:          result.ParentName,
		ExecutionGeneration: result.ExecutionGeneration,
		TerminalStatus:      result.TerminalStatus,
		Summary:             result.Summary,
		Evidence:            result.Evidence,
		Artifacts:           result.Artifacts,
		HandoffID:           result.HandoffID,
		UsageRowID:          result.UsageRowID,
		AdapterKind:         result.AdapterKind,
		StartedAt:           result.StartedAt,
		EndedAt:             result.EndedAt,
		Provenance:          result.Provenance,
	}
}

// computeRecruitmentResultDigest hashes result's own content snapshot so a
// replay can be recognised by a single digest comparison rather than
// re-diffing every field on every call. recruitmentResultContentDiff (below)
// is what NAMES the first differing field once a digest mismatch is found.
func computeRecruitmentResultDigest(result recruitmentResult) (string, error) {
	data, err := json.Marshal(recruitmentResultContentSnapshotOf(result))
	if err != nil {
		return "", fmt.Errorf("marshal recruitment result content: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// recruitmentResultContentDiff returns the name of the first field on which
// existing and incoming disagree, or "" if their content is identical. The
// comparison order is fixed so "the first differing field" is deterministic
// across calls -- a replayed report that changes several fields at once
// always names the same one.
func recruitmentResultContentDiff(existing, incoming recruitmentResult) string {
	checks := []struct {
		name  string
		equal bool
	}{
		{"intent_id", existing.IntentID == incoming.IntentID},
		{"dispatch_id", existing.DispatchID == incoming.DispatchID},
		{"child_name", existing.ChildName == incoming.ChildName},
		{"parent_name", existing.ParentName == incoming.ParentName},
		{"execution_generation", existing.ExecutionGeneration == incoming.ExecutionGeneration},
		{"terminal_status", existing.TerminalStatus == incoming.TerminalStatus},
		{"summary", existing.Summary == incoming.Summary},
		{"handoff_id", existing.HandoffID == incoming.HandoffID},
		{"usage_row_id", existing.UsageRowID == incoming.UsageRowID},
		{"adapter_kind", existing.AdapterKind == incoming.AdapterKind},
		{"started_at", existing.StartedAt == incoming.StartedAt},
		{"ended_at", existing.EndedAt == incoming.EndedAt},
		{"provenance", existing.Provenance == incoming.Provenance},
		{"artifacts", reflect.DeepEqual(existing.Artifacts, incoming.Artifacts)},
		{"evidence", reflect.DeepEqual(existing.Evidence, incoming.Evidence)},
	}
	for _, c := range checks {
		if !c.equal {
			return c.name
		}
	}
	return ""
}

// recruitmentGenerationConflict reports a non-empty refusal message when
// incoming is an OLDER execution generation than existing -- a child that
// ran under a superseded run must never bind its result over a newer one.
// Generation identifiers are opaque strings; an order is established only
// when BOTH parse as integers (this codebase's own attempt-id convention --
// see recruitmentIntent.AttemptID's "recruit_<unix-nanoseconds>" shape,
// cmd/recruitment.go -- already produces monotonically increasing values).
// When either value does not parse, no order can safely be established, so
// this returns "" and lets the ordinary content comparison run instead of
// guessing which one is older.
func recruitmentGenerationConflict(existingGeneration, incomingGeneration string) string {
	existingGeneration = strings.TrimSpace(existingGeneration)
	incomingGeneration = strings.TrimSpace(incomingGeneration)
	if existingGeneration == "" || incomingGeneration == "" || existingGeneration == incomingGeneration {
		return ""
	}
	existingN, existingOK := parseRecruitmentGeneration(existingGeneration)
	incomingN, incomingOK := parseRecruitmentGeneration(incomingGeneration)
	if !existingOK || !incomingOK {
		return ""
	}
	if incomingN < existingN {
		return fmt.Sprintf("execution generation %q is older than the recorded generation %q", incomingGeneration, existingGeneration)
	}
	return ""
}

func parseRecruitmentGeneration(value string) (int64, bool) {
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

// bindRecruitmentResult stores result under recruitment/results.json, keyed
// on RecruitmentID -- the ONLY idempotency key; do not invent a second
// dedupe mechanism here (see cmd/recruitment_recovery.go's
// TestOneRecruitmentIdempotencyMechanism, which fails by name if one ever
// appears).
//
// A second call carrying an already-stored RecruitmentID with IDENTICAL
// content returns the STORED record and mutates nothing (a verified
// replay). A second call with DIFFERENT content is refused, naming the
// first differing field, and mutates nothing (a conflict). A second call
// whose ExecutionGeneration is older than the one already recorded is
// refused, naming both generations, before content is even compared.
func bindRecruitmentResult(result recruitmentResult) (recruitmentResult, error) {
	if store == nil {
		return recruitmentResult{}, fmt.Errorf("no store initialized")
	}
	if strings.TrimSpace(result.RecruitmentID) == "" {
		return recruitmentResult{}, fmt.Errorf("recruitment result requires a non-empty RecruitmentID")
	}
	if !recruitmentTerminalStatusValid(result.TerminalStatus) {
		return recruitmentResult{}, fmt.Errorf(
			"recruitment result terminal_status %q is not in the declared vocabulary %v",
			result.TerminalStatus, recruitmentTerminalStatuses(),
		)
	}
	if strings.TrimSpace(result.SchemaVersion) == "" {
		result.SchemaVersion = RecruitmentResultSchemaVersion
	}

	digest, err := computeRecruitmentResultDigest(result)
	if err != nil {
		return recruitmentResult{}, err
	}

	var bound recruitmentResult
	var bindErr error
	var file recruitmentResultsFile
	updateErr := store.UpdateJSONAtomically(recruitmentResultsPath, &file, func() error {
		for _, existing := range file.Entries {
			if existing.RecruitmentID != result.RecruitmentID {
				continue
			}
			if conflict := recruitmentGenerationConflict(existing.ExecutionGeneration, result.ExecutionGeneration); conflict != "" {
				bindErr = fmt.Errorf("recruitment %s: %s", result.RecruitmentID, conflict)
				return errRecruitmentResultAlreadyBound
			}
			if existing.Receipt != nil && existing.Receipt.Digest == digest {
				bound = existing
				return errRecruitmentResultAlreadyBound
			}
			if field := recruitmentResultContentDiff(existing, result); field != "" {
				bindErr = fmt.Errorf(
					"recruitment %s: a replayed completion report conflicts with the stored one on field %q",
					result.RecruitmentID, field,
				)
				return errRecruitmentResultAlreadyBound
			}
			// Digest differs but no field-level diff was found -- e.g. a
			// content-snapshot shape change across a schema version. Treat
			// conservatively as the stored record; never silently rewrite.
			bound = existing
			return errRecruitmentResultAlreadyBound
		}
		result.Receipt = &colony.LifecycleReceiptReference{
			ID:     result.RecruitmentID + "-receipt",
			Digest: digest,
		}
		file.Entries = append(file.Entries, result)
		bound = result
		return nil
	})
	if updateErr != nil && !errors.Is(updateErr, errRecruitmentResultAlreadyBound) {
		return recruitmentResult{}, updateErr
	}
	if bindErr != nil {
		return recruitmentResult{}, bindErr
	}
	return bound, nil
}
