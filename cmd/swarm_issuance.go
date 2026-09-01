package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/storage"
)

const (
	externalSwarmIssuanceSchemaVersion = 1
	externalSwarmIssuanceIssued        = "issued"
	externalSwarmIssuanceFinalizing    = "finalizing"
	externalSwarmIssuanceFinalized     = "finalized"
)

var errExternalSwarmReplayNoWrite = errors.New("external swarm exact replay")

// externalSwarmManifestIssuance is the runtime-owned trust anchor for one
// externally dispatched swarm. The completion packet may echo the manifest,
// but it cannot mint or alter this row. Once finalized, the exact completion
// digest and frozen outcome make replay read-only and make every different
// packet fail before it can rewrite result/strike history.
type externalSwarmManifestIssuance struct {
	SchemaVersion    int                    `json:"schema_version"`
	SwarmID          string                 `json:"swarm_id"`
	ManifestSHA256   string                 `json:"manifest_sha256"`
	IssuedAt         string                 `json:"issued_at"`
	Status           string                 `json:"status"`
	CompletionSHA256 string                 `json:"completion_sha256,omitempty"`
	FinalizedAt      string                 `json:"finalized_at,omitempty"`
	Outcome          *swarmResultRecord     `json:"outcome,omitempty"`
	Next             string                 `json:"next,omitempty"`
	WaveCount        int                    `json:"wave_count,omitempty"`
	DispatchContract map[string]interface{} `json:"dispatch_contract,omitempty"`
}

func validateDurableSwarmID(s *storage.Store, swarmID string) (string, error) {
	if s == nil {
		return "", fmt.Errorf("no store initialized")
	}
	id := strings.TrimSpace(swarmID)
	target, err := safeIdentifierSegment(filepath.Join(s.BasePath(), "swarms"), "swarm id", id)
	if err != nil {
		return "", err
	}
	if !isValidSwarmResultID(id) {
		return "", fmt.Errorf("swarm id %q does not match the durable result convention", swarmID)
	}
	return target, nil
}

func externalSwarmIssuancePath(swarmID string) string {
	return filepath.ToSlash(filepath.Join("swarms", strings.TrimSpace(swarmID), "issuance.json"))
}

func issueExternalSwarmManifest(manifest swarmManifest) error {
	swarmDir, err := validateDurableSwarmID(store, manifest.SwarmID)
	if err != nil {
		return fmt.Errorf("issue external swarm manifest: %w", err)
	}
	if _, err := os.Lstat(swarmDir); err == nil {
		return fmt.Errorf("issue external swarm manifest: swarm id %q already exists", manifest.SwarmID)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("issue external swarm manifest: inspect swarm id %q: %w", manifest.SwarmID, err)
	}
	digest, err := jsonSHA256(manifest)
	if err != nil {
		return fmt.Errorf("issue external swarm manifest: hash manifest: %w", err)
	}
	issuedAt := strings.TrimSpace(manifest.GeneratedAt)
	if _, err := time.Parse(time.RFC3339, issuedAt); err != nil {
		return fmt.Errorf("issue external swarm manifest: generated_at is invalid: %w", err)
	}
	record := externalSwarmManifestIssuance{
		SchemaVersion:  externalSwarmIssuanceSchemaVersion,
		SwarmID:        strings.TrimSpace(manifest.SwarmID),
		ManifestSHA256: digest,
		IssuedAt:       issuedAt,
		Status:         externalSwarmIssuanceIssued,
	}
	if err := store.SaveJSON(externalSwarmIssuancePath(manifest.SwarmID), record); err != nil {
		return fmt.Errorf("issue external swarm manifest: %w", err)
	}
	return nil
}

func loadExternalSwarmManifestIssuance(manifest swarmManifest, manifestDigest string) (externalSwarmManifestIssuance, error) {
	swarmDir, err := validateDurableSwarmID(store, manifest.SwarmID)
	if err != nil {
		return externalSwarmManifestIssuance{}, err
	}
	issuanceAbs := filepath.Join(swarmDir, "issuance.json")
	info, err := os.Lstat(issuanceAbs)
	if err != nil {
		if os.IsNotExist(err) {
			return externalSwarmManifestIssuance{}, fmt.Errorf("swarm_manifest %s was not issued by this runtime; rerun `aether swarm --plan-only`", manifest.SwarmID)
		}
		return externalSwarmManifestIssuance{}, fmt.Errorf("inspect external swarm issuance: %w", err)
	}
	if !info.Mode().IsRegular() {
		return externalSwarmManifestIssuance{}, fmt.Errorf("external swarm issuance for %s is not a regular file", manifest.SwarmID)
	}
	var record externalSwarmManifestIssuance
	if err := store.LoadJSON(externalSwarmIssuancePath(manifest.SwarmID), &record); err != nil {
		return externalSwarmManifestIssuance{}, fmt.Errorf("load external swarm issuance: %w", err)
	}
	if err := validateExternalSwarmIssuanceRecord(record, manifest, manifestDigest); err != nil {
		return externalSwarmManifestIssuance{}, err
	}
	return record, nil
}

func validateExternalSwarmIssuanceRecord(record externalSwarmManifestIssuance, manifest swarmManifest, manifestDigest string) error {
	if record.SchemaVersion != externalSwarmIssuanceSchemaVersion {
		return fmt.Errorf("external swarm issuance for %s has unsupported schema version %d", manifest.SwarmID, record.SchemaVersion)
	}
	if strings.TrimSpace(record.SwarmID) != strings.TrimSpace(manifest.SwarmID) {
		return fmt.Errorf("external swarm issuance identity does not match manifest swarm_id %s", manifest.SwarmID)
	}
	if strings.TrimSpace(record.ManifestSHA256) == "" || strings.TrimSpace(record.ManifestSHA256) != strings.TrimSpace(manifestDigest) {
		return fmt.Errorf("swarm_manifest content does not match runtime issuance for %s", manifest.SwarmID)
	}
	if _, err := time.Parse(time.RFC3339, strings.TrimSpace(record.IssuedAt)); err != nil {
		return fmt.Errorf("external swarm issuance for %s has invalid issued_at: %w", manifest.SwarmID, err)
	}
	switch strings.TrimSpace(record.Status) {
	case externalSwarmIssuanceIssued:
		if record.CompletionSHA256 != "" || record.Outcome != nil {
			return fmt.Errorf("issued external swarm %s already carries terminal evidence", manifest.SwarmID)
		}
	case externalSwarmIssuanceFinalizing:
		if strings.TrimSpace(record.CompletionSHA256) == "" {
			return fmt.Errorf("external swarm %s is finalizing without a completion digest", manifest.SwarmID)
		}
	case externalSwarmIssuanceFinalized:
		if strings.TrimSpace(record.CompletionSHA256) == "" || record.Outcome == nil {
			return fmt.Errorf("finalized external swarm %s is missing its frozen completion outcome", manifest.SwarmID)
		}
		if strings.TrimSpace(record.Outcome.SwarmID) != strings.TrimSpace(manifest.SwarmID) {
			return fmt.Errorf("finalized external swarm %s has mismatched outcome identity", manifest.SwarmID)
		}
	default:
		return fmt.Errorf("external swarm issuance for %s has invalid status %q", manifest.SwarmID, record.Status)
	}
	return nil
}

func replayExternalSwarmFinalization(record externalSwarmManifestIssuance, completionDigest string) (map[string]interface{}, bool, error) {
	switch strings.TrimSpace(record.Status) {
	case externalSwarmIssuanceFinalized:
		if strings.TrimSpace(record.CompletionSHA256) != strings.TrimSpace(completionDigest) {
			return nil, false, fmt.Errorf("swarm %s was already finalized with a different completion packet", record.SwarmID)
		}
		return externalSwarmFinalizationResult(record), true, nil
	case externalSwarmIssuanceFinalizing:
		if strings.TrimSpace(record.CompletionSHA256) != strings.TrimSpace(completionDigest) {
			return nil, false, fmt.Errorf("swarm %s is already finalizing a different completion packet", record.SwarmID)
		}
		return nil, false, fmt.Errorf("swarm %s finalization is already in progress", record.SwarmID)
	default:
		return nil, false, nil
	}
}

func reserveExternalSwarmFinalization(manifest swarmManifest, manifestDigest, completionDigest string) (externalSwarmManifestIssuance, bool, error) {
	path := externalSwarmIssuancePath(manifest.SwarmID)
	var record externalSwarmManifestIssuance
	replay := false
	err := store.UpdateJSONAtomically(path, &record, func() error {
		if err := validateExternalSwarmIssuanceRecord(record, manifest, manifestDigest); err != nil {
			return err
		}
		if record.Status == externalSwarmIssuanceFinalized {
			if strings.TrimSpace(record.CompletionSHA256) != strings.TrimSpace(completionDigest) {
				return fmt.Errorf("swarm %s was already finalized with a different completion packet", record.SwarmID)
			}
			replay = true
			return errExternalSwarmReplayNoWrite
		}
		if record.Status == externalSwarmIssuanceFinalizing {
			if strings.TrimSpace(record.CompletionSHA256) != strings.TrimSpace(completionDigest) {
				return fmt.Errorf("swarm %s is already finalizing a different completion packet", record.SwarmID)
			}
			return fmt.Errorf("swarm %s finalization is already in progress", record.SwarmID)
		}
		record.Status = externalSwarmIssuanceFinalizing
		record.CompletionSHA256 = strings.TrimSpace(completionDigest)
		return nil
	})
	if errors.Is(err, errExternalSwarmReplayNoWrite) {
		return record, replay, nil
	}
	if err != nil {
		return externalSwarmManifestIssuance{}, false, fmt.Errorf("reserve external swarm finalization: %w", err)
	}
	return record, false, nil
}

func releaseExternalSwarmFinalization(manifest swarmManifest, manifestDigest, completionDigest string) {
	path := externalSwarmIssuancePath(manifest.SwarmID)
	var record externalSwarmManifestIssuance
	_ = store.UpdateJSONAtomically(path, &record, func() error {
		if err := validateExternalSwarmIssuanceRecord(record, manifest, manifestDigest); err != nil {
			return err
		}
		if record.Status != externalSwarmIssuanceFinalizing || strings.TrimSpace(record.CompletionSHA256) != strings.TrimSpace(completionDigest) {
			return fmt.Errorf("external swarm finalization reservation changed")
		}
		record.Status = externalSwarmIssuanceIssued
		record.CompletionSHA256 = ""
		return nil
	})
}

func completeExternalSwarmFinalization(manifest swarmManifest, manifestDigest, completionDigest string, outcome swarmResultRecord, next string) (externalSwarmManifestIssuance, error) {
	path := externalSwarmIssuancePath(manifest.SwarmID)
	var record externalSwarmManifestIssuance
	err := store.UpdateJSONAtomically(path, &record, func() error {
		if err := validateExternalSwarmIssuanceRecord(record, manifest, manifestDigest); err != nil {
			return err
		}
		if record.Status != externalSwarmIssuanceFinalizing || strings.TrimSpace(record.CompletionSHA256) != strings.TrimSpace(completionDigest) {
			return fmt.Errorf("external swarm %s lost its finalization reservation", manifest.SwarmID)
		}
		outcomeCopy := outcome
		record.Status = externalSwarmIssuanceFinalized
		record.FinalizedAt = strings.TrimSpace(outcome.CompletedAt)
		record.Outcome = &outcomeCopy
		record.Next = strings.TrimSpace(next)
		record.WaveCount = manifest.WaveCount
		record.DispatchContract = manifest.DispatchContract
		return nil
	})
	if err != nil {
		return externalSwarmManifestIssuance{}, fmt.Errorf("commit external swarm finalization receipt: %w", err)
	}
	return record, nil
}

func externalSwarmFinalizationResult(record externalSwarmManifestIssuance) map[string]interface{} {
	if record.Outcome == nil {
		return nil
	}
	outcome := record.Outcome
	return map[string]interface{}{
		"mode":                "destroy",
		"autopilot_available": true,
		"swarm_id":            outcome.SwarmID,
		"target":              outcome.Target,
		"status":              outcome.Status,
		"root_cause":          outcome.RootCause,
		"solution":            outcome.Solution,
		"recommendation":      outcome.Recommendation,
		"workers":             swarmExecutionsForJSON(outcome.Workers),
		"dispatches":          swarmExecutionsForJSON(outcome.Workers),
		"worker_count":        len(outcome.Workers),
		"wave_count":          record.WaveCount,
		"files_touched":       append([]string{}, outcome.Files...),
		"tests_written":       append([]string{}, outcome.Tests...),
		"blockers":            append([]string{}, outcome.Blockers...),
		"dispatch_mode":       "external-task",
		"dispatch_contract":   record.DispatchContract,
		"next":                record.Next,
		"watch":               false,
	}
}
