package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

type pheromoneSyncOptions struct {
	ActiveOnly bool
}

type pheromoneSyncResult struct {
	SourceRoot        string `json:"source_root"`
	TargetRoot        string `json:"target_root"`
	SourceSignals     int    `json:"source_signals"`
	ProcessedSignals  int    `json:"processed_signals"`
	NewSignalsWritten int    `json:"new_signals_written"`
	UpdatedSignals    int    `json:"updated_signals"`
	DedupedExisting   int    `json:"deduped_existing"`
	SkippedCount      int    `json:"skipped_count"`
	TotalSignals      int    `json:"total_signals"`
}

func syncPheromoneStores(sourceRoot, targetRoot string, opts pheromoneSyncOptions) (pheromoneSyncResult, error) {
	sourceStore, resolvedSource, err := newPheromoneStoreForRoot(sourceRoot)
	if err != nil {
		return pheromoneSyncResult{}, err
	}
	targetStore, resolvedTarget, err := newPheromoneStoreForRoot(targetRoot)
	if err != nil {
		return pheromoneSyncResult{}, err
	}

	sourceFile, err := loadPheromoneFileWithFallback(sourceStore)
	if err != nil {
		return pheromoneSyncResult{}, fmt.Errorf("load source pheromones: %w", err)
	}
	targetFile, err := loadPheromoneFileWithFallback(targetStore)
	if err != nil {
		return pheromoneSyncResult{}, fmt.Errorf("load target pheromones: %w", err)
	}

	result := mergePheromoneFiles(&targetFile, sourceFile, opts)
	result.SourceRoot = resolvedSource
	result.TargetRoot = resolvedTarget

	if result.NewSignalsWritten == 0 && result.UpdatedSignals == 0 {
		return result, nil
	}
	if err := targetStore.SaveJSON("pheromones.json", targetFile); err != nil {
		return pheromoneSyncResult{}, fmt.Errorf("save target pheromones: %w", err)
	}
	return result, nil
}

func newPheromoneStoreForRoot(root string) (*storage.Store, string, error) {
	resolvedRoot, err := resolvePheromoneRoot(root)
	if err != nil {
		return nil, "", err
	}
	dataPath := filepath.Join(resolvedRoot, ".aether", "data")
	s, err := storage.NewStore(dataPath)
	if err != nil {
		return nil, "", fmt.Errorf("create pheromone store for %s: %w", resolvedRoot, err)
	}
	return s, resolvedRoot, nil
}

func resolvePheromoneRoot(root string) (string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		root = os.Getenv("AETHER_ROOT")
	}
	if strings.TrimSpace(root) == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		root = cwd
	}

	root = filepath.Clean(root)
	switch {
	case filepath.Base(root) == "data" && filepath.Base(filepath.Dir(root)) == ".aether":
		root = filepath.Dir(filepath.Dir(root))
	case filepath.Base(root) == ".aether":
		root = filepath.Dir(root)
	}

	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	return abs, nil
}

func loadPheromoneFileWithFallback(s *storage.Store) (colony.PheromoneFile, error) {
	var pf colony.PheromoneFile
	if err := s.LoadJSON("pheromones.json", &pf); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return colony.PheromoneFile{Signals: []colony.PheromoneSignal{}}, nil
		}
		return colony.PheromoneFile{}, err
	}
	normalizePheromoneFile(&pf)
	return pf, nil
}

func mergePheromoneFiles(target *colony.PheromoneFile, source colony.PheromoneFile, opts pheromoneSyncOptions) pheromoneSyncResult {
	normalizePheromoneFile(target)
	normalizePheromoneFile(&source)

	result := pheromoneSyncResult{
		SourceSignals: len(source.Signals),
		TotalSignals:  len(target.Signals),
	}

	if target.Version == nil && source.Version != nil {
		value := *source.Version
		target.Version = &value
	}
	if target.ColonyID == nil && source.ColonyID != nil {
		value := *source.ColonyID
		target.ColonyID = &value
	}

	idIndex := make(map[string]int, len(target.Signals))
	keyIndex := make(map[string]int, len(target.Signals))
	for i := range target.Signals {
		sig := target.Signals[i]
		if sig.ID != "" {
			idIndex[sig.ID] = i
		}
		key := pheromoneSignalKey(sig)
		if key != "" {
			keyIndex[key] = i
		}
	}

	for _, sig := range source.Signals {
		if opts.ActiveOnly && !sig.Active {
			result.SkippedCount++
			continue
		}

		normalizePheromoneSignal(&sig)
		result.ProcessedSignals++

		if idx, ok := idIndex[sig.ID]; ok {
			merged := mergePheromoneSignalsByID(target.Signals[idx], sig)
			if !reflect.DeepEqual(target.Signals[idx], merged) {
				target.Signals[idx] = merged
				result.UpdatedSignals++
			}
			key := pheromoneSignalKey(target.Signals[idx])
			if key != "" {
				keyIndex[key] = idx
			}
			continue
		}

		key := pheromoneSignalKey(sig)
		if idx, ok := keyIndex[key]; ok {
			merged := mergePheromoneSignalsByKey(target.Signals[idx], sig)
			if !reflect.DeepEqual(target.Signals[idx], merged) {
				target.Signals[idx] = merged
				result.UpdatedSignals++
			}
			result.DedupedExisting++
			continue
		}

		target.Signals = append(target.Signals, sig)
		index := len(target.Signals) - 1
		if sig.ID != "" {
			idIndex[sig.ID] = index
		}
		if key != "" {
			keyIndex[key] = index
		}
		result.NewSignalsWritten++
	}

	result.TotalSignals = len(target.Signals)
	return result
}

func normalizePheromoneFile(pf *colony.PheromoneFile) {
	if pf.Signals == nil {
		pf.Signals = []colony.PheromoneSignal{}
	}
	for i := range pf.Signals {
		normalizePheromoneSignal(&pf.Signals[i])
	}
}

func normalizePheromoneSignal(sig *colony.PheromoneSignal) {
	sig.Type = strings.ToUpper(strings.TrimSpace(sig.Type))
	if sig.Priority == "" {
		switch sig.Type {
		case "FOCUS":
			sig.Priority = "normal"
		case "REDIRECT":
			sig.Priority = "high"
		case "FEEDBACK":
			sig.Priority = "low"
		}
	}
	if sig.ContentHash == nil || strings.TrimSpace(*sig.ContentHash) == "" {
		hash := "sha256:" + sha256Sum(extractText(sig.Content))
		sig.ContentHash = &hash
	}
	if sig.Tags == nil {
		sig.Tags = []colony.PheromoneTag{}
	}
}

func pheromoneSignalKey(sig colony.PheromoneSignal) string {
	normalizePheromoneSignal(&sig)
	if sig.ContentHash == nil {
		return ""
	}
	return sig.Type + "|" + *sig.ContentHash
}

func mergePheromoneSignalsByID(dst, src colony.PheromoneSignal) colony.PheromoneSignal {
	normalizePheromoneSignal(&dst)
	normalizePheromoneSignal(&src)

	merged := dst
	if signalMutationAfter(src, dst) {
		merged = src
	}

	merged.ID = firstNonEmpty(merged.ID, src.ID, dst.ID)
	merged.Type = firstNonEmpty(merged.Type, src.Type, dst.Type)
	merged.Priority = firstNonEmpty(merged.Priority, src.Priority, dst.Priority)
	merged.Source = firstNonEmpty(merged.Source, src.Source, dst.Source)
	if len(merged.Content) == 0 {
		if len(src.Content) > 0 {
			merged.Content = src.Content
		} else {
			merged.Content = dst.Content
		}
	}
	merged.ContentHash = contentHashPtr(merged)
	merged.CreatedAt = latestTimestamp(dst.CreatedAt, src.CreatedAt)
	merged.ExpiresAt = latestTimestampPtr(dst.ExpiresAt, src.ExpiresAt)
	merged.ArchivedAt = latestTimestampPtr(dst.ArchivedAt, src.ArchivedAt)
	merged.Strength = maxFloat64Ptr(dst.Strength, src.Strength)
	setSignalObservationCount(&merged, maxInt(signalObservationCount(dst), signalObservationCount(src)))
	merged.Tags = unionPheromoneTags(dst.Tags, src.Tags)
	if merged.Reason == nil {
		merged.Reason = firstStringPtr(dst.Reason, src.Reason)
	}
	if merged.Scope == nil {
		merged.Scope = firstScopePtr(dst.Scope, src.Scope)
	}
	return merged
}

func mergePheromoneSignalsByKey(dst, src colony.PheromoneSignal) colony.PheromoneSignal {
	normalizePheromoneSignal(&dst)
	normalizePheromoneSignal(&src)

	merged := dst
	merged.Type = firstNonEmpty(dst.Type, src.Type)
	merged.Priority = firstNonEmpty(dst.Priority, src.Priority)
	merged.Source = firstNonEmpty(dst.Source, src.Source)
	if len(merged.Content) == 0 && len(src.Content) > 0 {
		merged.Content = src.Content
	}
	merged.ContentHash = contentHashPtr(merged)
	merged.CreatedAt = latestTimestamp(dst.CreatedAt, src.CreatedAt)
	merged.ExpiresAt = latestTimestampPtr(dst.ExpiresAt, src.ExpiresAt)
	merged.ArchivedAt = latestTimestampPtr(dst.ArchivedAt, src.ArchivedAt)
	merged.Active = dst.Active || src.Active
	merged.Strength = maxFloat64Ptr(dst.Strength, src.Strength)
	setSignalObservationCount(&merged, signalObservationCount(dst)+signalObservationCount(src))
	merged.Tags = unionPheromoneTags(dst.Tags, src.Tags)
	if merged.Reason == nil {
		merged.Reason = firstStringPtr(dst.Reason, src.Reason)
	}
	if merged.Scope == nil {
		merged.Scope = firstScopePtr(dst.Scope, src.Scope)
	}
	return merged
}

func signalObservationCount(sig colony.PheromoneSignal) int {
	if sig.ReinforcementCount == nil || *sig.ReinforcementCount < 0 {
		return 1
	}
	return 1 + *sig.ReinforcementCount
}

func setSignalObservationCount(sig *colony.PheromoneSignal, observations int) {
	if observations <= 1 {
		sig.ReinforcementCount = nil
		return
	}
	count := observations - 1
	sig.ReinforcementCount = &count
}

func unionPheromoneTags(left, right []colony.PheromoneTag) []colony.PheromoneTag {
	merged := make([]colony.PheromoneTag, 0, len(left)+len(right))
	seen := map[string]struct{}{}
	for _, tag := range append(append([]colony.PheromoneTag{}, left...), right...) {
		key := fmt.Sprintf("%s|%s|%.6f", tag.Value, tag.Category, tag.Weight)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		merged = append(merged, tag)
	}
	sort.SliceStable(merged, func(i, j int) bool {
		if merged[i].Category != merged[j].Category {
			return merged[i].Category < merged[j].Category
		}
		if merged[i].Value != merged[j].Value {
			return merged[i].Value < merged[j].Value
		}
		return merged[i].Weight < merged[j].Weight
	})
	return merged
}

func latestTimestamp(left, right string) string {
	switch {
	case left == "":
		return right
	case right == "":
		return left
	}
	leftTime, leftErr := time.Parse(time.RFC3339, left)
	rightTime, rightErr := time.Parse(time.RFC3339, right)
	if leftErr != nil || rightErr != nil {
		if right > left {
			return right
		}
		return left
	}
	if rightTime.After(leftTime) {
		return right
	}
	return left
}

func latestTimestampPtr(left, right *string) *string {
	switch {
	case left == nil && right == nil:
		return nil
	case left == nil:
		value := *right
		return &value
	case right == nil:
		value := *left
		return &value
	}
	value := latestTimestamp(*left, *right)
	return &value
}

func signalMutationAfter(left, right colony.PheromoneSignal) bool {
	leftTime := signalMutationTimestamp(left)
	rightTime := signalMutationTimestamp(right)
	return leftTime.After(rightTime)
}

func signalMutationTimestamp(sig colony.PheromoneSignal) time.Time {
	candidates := []string{sig.CreatedAt}
	if sig.ExpiresAt != nil {
		candidates = append(candidates, *sig.ExpiresAt)
	}
	if sig.ArchivedAt != nil {
		candidates = append(candidates, *sig.ArchivedAt)
	}
	best := time.Time{}
	for _, candidate := range candidates {
		ts, err := time.Parse(time.RFC3339, candidate)
		if err != nil {
			continue
		}
		if ts.After(best) {
			best = ts
		}
	}
	return best
}

func contentHashPtr(sig colony.PheromoneSignal) *string {
	if sig.ContentHash != nil && strings.TrimSpace(*sig.ContentHash) != "" {
		value := *sig.ContentHash
		return &value
	}
	hash := "sha256:" + sha256Sum(extractText(sig.Content))
	return &hash
}

func maxFloat64Ptr(left, right *float64) *float64 {
	switch {
	case left == nil && right == nil:
		return nil
	case left == nil:
		value := *right
		return &value
	case right == nil:
		value := *left
		return &value
	}
	if *right > *left {
		value := *right
		return &value
	}
	value := *left
	return &value
}

func firstStringPtr(values ...*string) *string {
	for _, value := range values {
		if value != nil && strings.TrimSpace(*value) != "" {
			copy := *value
			return &copy
		}
	}
	return nil
}

func firstScopePtr(values ...*colony.PheromoneScope) *colony.PheromoneScope {
	for _, value := range values {
		if value == nil {
			continue
		}
		copy := *value
		return &copy
	}
	return nil
}

func maxInt(left, right int) int {
	if right > left {
		return right
	}
	return left
}

func formatPheromoneSyncSummary(result pheromoneSyncResult) string {
	if result.NewSignalsWritten == 0 && result.UpdatedSignals == 0 {
		return ""
	}
	parts := []string{}
	if result.NewSignalsWritten > 0 {
		parts = append(parts, fmt.Sprintf("%d new", result.NewSignalsWritten))
	}
	if result.UpdatedSignals > 0 {
		parts = append(parts, fmt.Sprintf("%d updated", result.UpdatedSignals))
	}
	if result.DedupedExisting > 0 {
		parts = append(parts, fmt.Sprintf("%d deduped", result.DedupedExisting))
	}
	return "Pheromone sync: " + strings.Join(parts, ", ")
}
