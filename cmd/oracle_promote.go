package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/learn"
	"github.com/calcosmic/Aether/pkg/memory"
)

type oraclePromoteOutcome struct {
	Question   string `json:"question"`
	Finding    string `json:"finding"`
	Action     string `json:"action"` // "learning", "learning+instinct", "skipped"
	SkipReason string `json:"skip_reason,omitempty"`
}

// runOraclePromote promotes high-confidence Oracle findings into colony
// memory. Every finding passes the admissibility gate — content that cannot be
// checked against the repository later (no file, command, or error named) is
// rejected with its reason, exactly as hand-written observations are.
//
// v5 had this as a one-command capture; it was removed in the Go migration and
// findings have been hand-copied (or lost) ever since.
func runOraclePromote(root string, minConfidence int, dryRun bool) (map[string]interface{}, error) {
	if minConfidence <= 0 {
		minConfidence = 80
	}
	planPath := filepath.Join(root, ".aether", "oracle", "plan.json")
	data, err := os.ReadFile(planPath)
	if err != nil {
		return nil, fmt.Errorf("no Oracle research to promote: %s is unavailable (%v); run `aether oracle \"<topic>\"` first", planPath, err)
	}
	var plan oraclePlanFile
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("parse %s: %w", planPath, err)
	}

	outcomes := make([]oraclePromoteOutcome, 0, 8)
	questionsConsidered := 0
	learnings := 0
	instincts := 0

	for _, question := range plan.Questions {
		if question.Confidence < minConfidence {
			continue
		}
		questionsConsidered++
		questionLabel := strings.TrimSpace(question.Text)
		if len(questionLabel) > 120 {
			questionLabel = questionLabel[:120] + "…"
		}
		for _, finding := range question.KeyFindings {
			text := strings.TrimSpace(finding.Text)
			if text == "" {
				continue
			}
			admissible, reason := memory.IsAdmissibleInstinctContent(text)
			if !admissible {
				outcomes = append(outcomes, oraclePromoteOutcome{
					Question: questionLabel, Finding: truncateForReport(text), Action: "skipped", SkipReason: reason,
				})
				continue
			}
			if dryRun {
				outcomes = append(outcomes, oraclePromoteOutcome{
					Question: questionLabel, Finding: truncateForReport(text), Action: "learning+instinct",
				})
				learnings++
				instincts++
				continue
			}

			action := ""
			if promoteOracleFindingAsLearning(text, question.Confidence) {
				learnings++
				action = "learning"
			}
			if promoteOracleFindingAsInstinct(text, question.Confidence) {
				instincts++
				if action == "" {
					action = "instinct"
				} else {
					action += "+instinct"
				}
			}
			if action == "" {
				action = "skipped"
			}
			outcomes = append(outcomes, oraclePromoteOutcome{
				Question: questionLabel, Finding: truncateForReport(text), Action: action,
			})
		}
	}

	return map[string]interface{}{
		"promoted":             !dryRun,
		"dry_run":              dryRun,
		"min_confidence":       minConfidence,
		"questions_considered": questionsConsidered,
		"learnings":            learnings,
		"instincts":            instincts,
		"outcomes":             outcomes,
		"next":                 "aether pheromone-display",
	}, nil
}

func truncateForReport(text string) string {
	if len(text) > 160 {
		return text[:160] + "…"
	}
	return text
}

// promoteOracleFindingAsLearning stores a finding as a hypothesis learning
// entry through the same privacy scan and classification as continue-time
// capture. Returns false when the pipeline rejects it.
func promoteOracleFindingAsLearning(text string, confidence int) bool {
	if store == nil {
		return false
	}
	scanResult := privacyScan(text)
	classification := learn.ClassifyEntry(text, learn.PrivacyScanResult{
		Blocked:  scanResult.Blocked,
		Clean:    scanResult.Clean,
		Findings: scanResult.Findings,
	})
	if classification == learn.ClassBlocked {
		return false
	}
	entry := learn.Entry{
		Content:        scanResult.Clean,
		Classification: classification,
		Confidence:     float64(confidence) / 100.0,
		Status:         learn.StatusHypothesis,
		Caste:          "oracle",
	}
	if err := learn.NewColonyStore(store).Add(entry); err != nil {
		fmt.Fprintf(stderr, "warning: oracle promote could not store learning: %v\n", err)
		return false
	}
	return true
}

// promoteOracleFindingAsInstinct promotes a finding through the memory
// promotion service (which re-checks admissibility, dedupes, and records
// provenance). Returns false when rejected.
func promoteOracleFindingAsInstinct(text string, confidence int) bool {
	if store == nil {
		return false
	}
	now := time.Now().UTC().Format(time.RFC3339)
	obs := colony.Observation{
		ContentHash:      "sha256:" + sha256Sum(text),
		Content:          text,
		WisdomType:       "pattern",
		ObservationCount: 1,
		FirstSeen:        now,
		LastSeen:         now,
		SourceType:       "oracle",
		EvidenceType:     "research",
	}
	service := memory.NewPromoteService(store, events.NewBus(store, events.DefaultConfig()))
	if _, err := service.Promote(context.Background(), obs, "oracle-promote"); err != nil {
		fmt.Fprintf(stderr, "warning: oracle promote could not create instinct: %v\n", err)
		return false
	}
	return true
}

// Pheromones are deliberately NOT auto-written by promote: research findings
// become memory (learnings + instincts); colony steering stays a user
// decision. The oracle wizard offers a FOCUS write with explicit approval.
