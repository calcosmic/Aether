package cmd

// BIO-05 (203-10-PLAN.md): the trophallaxis packet -- a scoped, acknowledged
// carrier for a child's useful result on its way home. The existing worker
// handoff relay (cmd/codex_dispatch_contract.go's workerHandoffRecord)
// already carries a summary from one worker to the next; what BIO-05 adds on
// top of that shape is the two things that make a handback COUNT for CEC-07
// purposes: an explicit acknowledgement from whoever receives it, and (Task
// 3) a recorded note of the decision the receiver made from it.
//
// A packet is packed from a bound recruitment result (cmd/recruitment_result.go),
// never from raw worker output -- see packTrophallaxisPacket's own guard.

import (
	"fmt"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/colony"
)

// trophallaxisPacketsPath is the store-relative path packed/acknowledged
// packets persist to. New data file, per this plan's frontmatter.
const trophallaxisPacketsPath = "recruitment/packets.json"

// TrophallaxisPacketSchemaVersion is the wire-shape version every packet
// carries, mirroring RecruitmentResultSchemaVersion's "one version, additive
// fields only" discipline (cmd/recruitment_result.go).
const TrophallaxisPacketSchemaVersion = "trophallaxis/v1"

// trophallaxisPacketRetention bounds recruitment/packets.json the same way
// immune.go's scarsData caps scars.json -- oldest entries are trimmed first,
// newest kept.
const trophallaxisPacketRetention = 200

// trophallaxisReceiverKind names the two BIO-05 receiver shapes. A packet has
// exactly one receiver, of exactly one kind.
type trophallaxisReceiverKind string

const (
	trophallaxisReceiverParent   trophallaxisReceiverKind = "parent"
	trophallaxisReceiverFollowOn trophallaxisReceiverKind = "follow_on"
)

// trophallaxisReceiverKinds returns every declared receiver-kind constant,
// mirroring the recruitmentTerminalStatuses()/ColonyLiveTopics() completeness
// convention: a kind added to the const block above must also be added here.
func trophallaxisReceiverKinds() []trophallaxisReceiverKind {
	return []trophallaxisReceiverKind{trophallaxisReceiverParent, trophallaxisReceiverFollowOn}
}

// Section names a packet's scope may name. These mirror workerHandoffRecord's
// own field set (cmd/codex_dispatch_contract.go) one-for-one, minus the
// identity fields (ID, WorkerName, Caste, TaskID, Status) that are not
// travelling content -- those are carried on the packet directly, never
// gated by scope.
const (
	trophallaxisSectionSummary                = "summary"
	trophallaxisSectionChangedFiles           = "changed_files"
	trophallaxisSectionCommandsRun            = "commands_run"
	trophallaxisSectionVerificationStatus     = "verification_status"
	trophallaxisSectionKnownFailures          = "known_failures"
	trophallaxisSectionOpenDecisions          = "open_decisions"
	trophallaxisSectionAssumptions            = "assumptions"
	trophallaxisSectionNextWorkerInstructions = "next_worker_instructions"
	trophallaxisSectionDoNotRepeat            = "do_not_repeat"
)

// trophallaxisSections returns every declared scope-section constant, same
// completeness convention as trophallaxisReceiverKinds above.
func trophallaxisSections() []string {
	return []string{
		trophallaxisSectionSummary,
		trophallaxisSectionChangedFiles,
		trophallaxisSectionCommandsRun,
		trophallaxisSectionVerificationStatus,
		trophallaxisSectionKnownFailures,
		trophallaxisSectionOpenDecisions,
		trophallaxisSectionAssumptions,
		trophallaxisSectionNextWorkerInstructions,
		trophallaxisSectionDoNotRepeat,
	}
}

func trophallaxisSectionValid(section string) bool {
	for _, known := range trophallaxisSections() {
		if known == section {
			return true
		}
	}
	return false
}

// trophallaxisPacket is the scoped child-result carrier. Its field shape
// copies workerHandoffRecord's content fields verbatim (summary, changed
// files, commands run, verification status, known failures, open decisions,
// assumptions, next-worker instructions, do-not-repeat, freshness) and adds
// what BIO-05 needs beyond it: the packet/result/intent identity, the
// receiver, the declared scope, the recorded omissions, and the
// acknowledgement.
type trophallaxisPacket struct {
	SchemaVersion string                   `json:"schema_version"`
	PacketID      string                   `json:"packet_id"`
	ResultID      string                   `json:"result_id"`
	IntentID      string                   `json:"intent_id,omitempty"`
	ChildName     string                   `json:"child_name,omitempty"`
	Receiver      string                   `json:"receiver"`
	ReceiverKind  trophallaxisReceiverKind `json:"receiver_kind"`
	Scope         []string                 `json:"scope"`
	Omissions     []string                 `json:"omissions,omitempty"`

	Summary                string   `json:"summary,omitempty"`
	ChangedFiles           []string `json:"changed_files,omitempty"`
	CommandsRun            []string `json:"commands_run,omitempty"`
	VerificationStatus     string   `json:"verification_status,omitempty"`
	KnownFailures          []string `json:"known_failures,omitempty"`
	OpenDecisions          []string `json:"open_decisions,omitempty"`
	Assumptions            []string `json:"assumptions,omitempty"`
	NextWorkerInstructions []string `json:"next_worker_instructions,omitempty"`
	DoNotRepeat            []string `json:"do_not_repeat,omitempty"`
	Freshness              string   `json:"freshness,omitempty"`

	Acknowledgement *colony.SignalAcknowledgement `json:"acknowledgement,omitempty"`
	AcknowledgedAt  string                        `json:"acknowledged_at,omitempty"`
}

// trophallaxisPacketsFile is the on-disk container at trophallaxisPacketsPath.
type trophallaxisPacketsFile struct {
	Entries []trophallaxisPacket `json:"entries"`
}

// trophallaxisPackInput is packTrophallaxisPacket's caller-supplied input. A
// packet is packed from Result (a value already returned by
// bindRecruitmentResult -- see the Receipt guard below) and Handoff (a
// workerHandoffRecord-shaped source, reusing that struct's own field shape
// rather than inventing a second one).
//
// Exactly one of ParentReceiver / FollowOnReceiver must be set: naming both,
// or neither, is refused by packTrophallaxisPacket.
type trophallaxisPackInput struct {
	Result           recruitmentResult
	Handoff          workerHandoffRecord
	ParentReceiver   string
	FollowOnReceiver string
	Scope            []string
}

// trophallaxisReceiverResolver reports whether receiver resolves to a real
// worker (kind parent) or a real named follow-on step (kind follow_on).
// Injected as a package-level var (mirroring cmd/spawn.go's
// spawnCanSpawnDecision var doc comment: "exists so a test can substitute a
// deny answer for a single test case") rather than hardcoded, so
// packTrophallaxisPacket does not couple to one registry implementation.
var trophallaxisResolveReceiver trophallaxisReceiverResolver = defaultTrophallaxisReceiverResolver

type trophallaxisReceiverResolver func(receiver string, kind trophallaxisReceiverKind) bool

// defaultTrophallaxisReceiverResolver resolves against the same whole-run
// spawn ledger BIO-06 reuses (pkg/agent.SpawnTree) rather than a second
// registry -- 203-14-PLAN.md treats a follow-on consumer as itself a
// spawn-tree descendant (its own FollowOnFor attribute on the same row), so
// both receiver kinds resolve against one ledger. D-19 fail-closed: an
// unreadable ledger denies resolution rather than allowing it.
func defaultTrophallaxisReceiverResolver(receiver string, _ trophallaxisReceiverKind) bool {
	receiver = strings.TrimSpace(receiver)
	if receiver == "" || store == nil {
		return false
	}
	entries, err := agent.NewSpawnTree(store, "spawn-tree.txt").Parse()
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if strings.TrimSpace(entry.AgentName) == receiver {
			return true
		}
	}
	return false
}

// packTrophallaxisPacket packs input into a stored trophallaxisPacket. It
// enforces every BIO-05 packing guarantee: exactly one receiver, a receiver
// that actually resolves, scope-gated content with recorded omissions, and
// sanitised free text -- before ever calling store.UpdateJSONAtomically.
func packTrophallaxisPacket(input trophallaxisPackInput) (trophallaxisPacket, error) {
	resultID := strings.TrimSpace(input.Result.RecruitmentID)
	if resultID == "" {
		return trophallaxisPacket{}, fmt.Errorf("trophallaxis packet requires a bound recruitment result (empty RecruitmentID)")
	}
	if input.Result.Receipt == nil {
		return trophallaxisPacket{}, fmt.Errorf(
			"trophallaxis packet requires a bound recruitment result (recruitment %s has no receipt -- pack only from bindRecruitmentResult's own return, never raw worker output)",
			resultID,
		)
	}

	parentReceiver := strings.TrimSpace(input.ParentReceiver)
	followOnReceiver := strings.TrimSpace(input.FollowOnReceiver)
	if parentReceiver != "" && followOnReceiver != "" {
		return trophallaxisPacket{}, fmt.Errorf(
			"trophallaxis packet names two receivers (parent %q and follow-on %q); a packet has exactly one receiver",
			parentReceiver, followOnReceiver,
		)
	}
	var receiver string
	var receiverKind trophallaxisReceiverKind
	switch {
	case parentReceiver != "":
		receiver, receiverKind = parentReceiver, trophallaxisReceiverParent
	case followOnReceiver != "":
		receiver, receiverKind = followOnReceiver, trophallaxisReceiverFollowOn
	default:
		return trophallaxisPacket{}, fmt.Errorf("trophallaxis packet names no receiver; a packet needs exactly one")
	}

	resolve := trophallaxisResolveReceiver
	if resolve == nil {
		resolve = defaultTrophallaxisReceiverResolver
	}
	if !resolve(receiver, receiverKind) {
		return trophallaxisPacket{}, fmt.Errorf(
			"trophallaxis packet receiver %q resolves to no worker and no named follow-on step", receiver,
		)
	}

	scopeSet := make(map[string]bool, len(input.Scope))
	var declaredScope []string
	for _, s := range input.Scope {
		s = strings.TrimSpace(s)
		if !trophallaxisSectionValid(s) || scopeSet[s] {
			continue
		}
		scopeSet[s] = true
		declaredScope = append(declaredScope, s)
	}

	packet := trophallaxisPacket{
		SchemaVersion: TrophallaxisPacketSchemaVersion,
		PacketID:      fmt.Sprintf("trophallaxis_%s_%d", resultID, time.Now().UnixNano()),
		ResultID:      resultID,
		IntentID:      input.Result.IntentID,
		ChildName:     input.Result.ChildName,
		Receiver:      receiver,
		ReceiverKind:  receiverKind,
		Scope:         declaredScope,
	}

	var omissions []string
	for _, section := range trophallaxisSections() {
		if scopeSet[section] {
			continue
		}
		omissions = append(omissions, section)
	}
	packet.Omissions = omissions

	if scopeSet[trophallaxisSectionSummary] {
		packet.Summary = input.Handoff.Summary
	}
	if scopeSet[trophallaxisSectionChangedFiles] {
		packet.ChangedFiles = append([]string{}, input.Handoff.ChangedFiles...)
	}
	if scopeSet[trophallaxisSectionCommandsRun] {
		packet.CommandsRun = append([]string{}, input.Handoff.CommandsRun...)
	}
	if scopeSet[trophallaxisSectionVerificationStatus] {
		packet.VerificationStatus = input.Handoff.VerificationStatus
	}
	if scopeSet[trophallaxisSectionKnownFailures] {
		packet.KnownFailures = append([]string{}, input.Handoff.KnownFailures...)
	}
	if scopeSet[trophallaxisSectionOpenDecisions] {
		packet.OpenDecisions = append([]string{}, input.Handoff.OpenDecisions...)
	}
	if scopeSet[trophallaxisSectionAssumptions] {
		packet.Assumptions = append([]string{}, input.Handoff.Assumptions...)
	}
	if scopeSet[trophallaxisSectionNextWorkerInstructions] {
		packet.NextWorkerInstructions = append([]string{}, input.Handoff.NextWorkerInstructions...)
	}
	if scopeSet[trophallaxisSectionDoNotRepeat] {
		packet.DoNotRepeat = append([]string{}, input.Handoff.DoNotRepeat...)
	}
	packet.Freshness = strings.TrimSpace(input.Handoff.Freshness)

	if err := trophallaxisSanitizePacketText(&packet); err != nil {
		return trophallaxisPacket{}, err
	}

	return storeTrophallaxisPacket(packet)
}

// trophallaxisSanitizePacketText runs colony.SanitizeSignalContent over every
// free-text field on packet, because these fields are worker-authored and
// are replayed verbatim into later worker briefs. A rejection on any field
// refuses the whole pack (fail-closed) -- nothing is ever written to
// recruitment/packets.json carrying rejected content.
func trophallaxisSanitizePacketText(packet *trophallaxisPacket) error {
	var err error
	if packet.Summary, err = trophallaxisSanitizeField("summary", packet.Summary); err != nil {
		return err
	}
	if packet.VerificationStatus, err = trophallaxisSanitizeField("verification_status", packet.VerificationStatus); err != nil {
		return err
	}
	if packet.KnownFailures, err = trophallaxisSanitizeFieldList("known_failures", packet.KnownFailures); err != nil {
		return err
	}
	if packet.OpenDecisions, err = trophallaxisSanitizeFieldList("open_decisions", packet.OpenDecisions); err != nil {
		return err
	}
	if packet.Assumptions, err = trophallaxisSanitizeFieldList("assumptions", packet.Assumptions); err != nil {
		return err
	}
	if packet.NextWorkerInstructions, err = trophallaxisSanitizeFieldList("next_worker_instructions", packet.NextWorkerInstructions); err != nil {
		return err
	}
	if packet.DoNotRepeat, err = trophallaxisSanitizeFieldList("do_not_repeat", packet.DoNotRepeat); err != nil {
		return err
	}
	return nil
}

func trophallaxisSanitizeField(label, value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return value, nil
	}
	sanitized, err := colony.SanitizeSignalContent(value)
	if err != nil {
		return "", fmt.Errorf("trophallaxis packet %s: %w", label, err)
	}
	return sanitized, nil
}

func trophallaxisSanitizeFieldList(label string, values []string) ([]string, error) {
	if len(values) == 0 {
		return values, nil
	}
	sanitized := make([]string, len(values))
	for i, v := range values {
		s, err := trophallaxisSanitizeField(fmt.Sprintf("%s[%d]", label, i), v)
		if err != nil {
			return nil, err
		}
		sanitized[i] = s
	}
	return sanitized, nil
}

func storeTrophallaxisPacket(packet trophallaxisPacket) (trophallaxisPacket, error) {
	if store == nil {
		return trophallaxisPacket{}, fmt.Errorf("no store initialized")
	}
	var file trophallaxisPacketsFile
	err := store.UpdateJSONAtomically(trophallaxisPacketsPath, &file, func() error {
		file.Entries = append(file.Entries, packet)
		file.Entries = pruneTrophallaxisPackets(file.Entries, trophallaxisPacketRetention)
		return nil
	})
	if err != nil {
		return trophallaxisPacket{}, err
	}
	return packet, nil
}

func pruneTrophallaxisPackets(entries []trophallaxisPacket, limit int) []trophallaxisPacket {
	if limit <= 0 || len(entries) <= limit {
		return entries
	}
	return entries[len(entries)-limit:]
}

// errTrophallaxisPacketNotFound and errTrophallaxisPacketNoChange are the
// internal sentinels acknowledgeTrophallaxisPacket returns from its own
// UpdateJSONAtomically mutate closure to abort a write that must not
// happen -- mirroring errRecruitmentResultAlreadyBound's role in
// bindRecruitmentResult (cmd/recruitment_result.go): "if mutate returns an
// error, no write occurs" is exactly the "mutates nothing" guarantee a
// not-found lookup or an already-acknowledged replay requires.
var (
	errTrophallaxisPacketNotFound = fmt.Errorf("trophallaxis packet not found")
	errTrophallaxisPacketNoChange = fmt.Errorf("trophallaxis packet already acknowledged")
)

// acknowledgeTrophallaxisPacket records who acknowledged packetID and when.
// A second acknowledgement of the same packet is read-only: the stored file
// is left byte-identical, and the FIRST acknowledgement is returned.
func acknowledgeTrophallaxisPacket(packetID string, ack colony.SignalAcknowledgement, acknowledgedAt string) (trophallaxisPacket, error) {
	if store == nil {
		return trophallaxisPacket{}, fmt.Errorf("no store initialized")
	}
	packetID = strings.TrimSpace(packetID)
	if packetID == "" {
		return trophallaxisPacket{}, fmt.Errorf("trophallaxis packet id is required")
	}
	if err := ack.Validate(); err != nil {
		return trophallaxisPacket{}, fmt.Errorf("trophallaxis packet acknowledgement: %w", err)
	}
	acknowledgedAt = strings.TrimSpace(acknowledgedAt)
	if acknowledgedAt == "" {
		acknowledgedAt = time.Now().UTC().Format(time.RFC3339)
	}

	var result trophallaxisPacket
	var file trophallaxisPacketsFile
	updateErr := store.UpdateJSONAtomically(trophallaxisPacketsPath, &file, func() error {
		for i := range file.Entries {
			if file.Entries[i].PacketID != packetID {
				continue
			}
			if file.Entries[i].Acknowledgement != nil {
				result = file.Entries[i]
				return errTrophallaxisPacketNoChange
			}
			ackCopy := ack
			file.Entries[i].Acknowledgement = &ackCopy
			file.Entries[i].AcknowledgedAt = acknowledgedAt
			result = file.Entries[i]
			return nil
		}
		return errTrophallaxisPacketNotFound
	})
	if updateErr != nil {
		if updateErr == errTrophallaxisPacketNotFound {
			return trophallaxisPacket{}, fmt.Errorf("trophallaxis packet %q does not exist", packetID)
		}
		if updateErr == errTrophallaxisPacketNoChange {
			return result, nil
		}
		return trophallaxisPacket{}, updateErr
	}
	return result, nil
}

// trophallaxisPacketAcknowledgementState reports the packet's acknowledgement
// state using the SAME wording the existing acknowledgement-pending
// vocabulary already uses (cmd/agency_contract.go's AgencyAcknowledgementPending)
// rather than a second phrase for the same state.
func trophallaxisPacketAcknowledgementState(packet trophallaxisPacket) string {
	if packet.Acknowledgement == nil {
		return AgencyAcknowledgementPending
	}
	return fmt.Sprintf("%s (evidence %s)", packet.Acknowledgement.ActorID, packet.Acknowledgement.EvidenceID)
}
