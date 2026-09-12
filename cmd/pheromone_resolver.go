package cmd

import (
	"math"
	"sort"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// pheromoneEffectiveFloor is the single named effective-strength floor every
// pheromone reader in this runtime uses. It restores Classic's
// _pheromone_read floor (v5.4.0 pheromone.sh, 0.1) as one constant instead
// of three independently drifting inline literals -- see 203-CLASSIC-SYNTHESIS.md
// SYN-203-10.
const pheromoneEffectiveFloor = 0.1

// Exclusion reasons resolveEffectivePheromones records on a resolvedPheromone
// that is not currently in effect. These are the exact, named category
// strings every caller and test compares against -- never a bare boolean.
const (
	pheromoneExcludedInactive   = "inactive"
	pheromoneExcludedQuarantine = "quarantined"
	pheromoneExcludedExpired    = "expired"
	pheromoneExcludedBelowFloor = "below-floor"
	pheromoneExcludedMalformed  = "malformed"
)

// resolvedPheromone carries a pheromone signal plus the ONE runtime decision
// about whether it is currently in effect, how strong it is right now, its
// origin, and (if excluded) why. Every caller that needs "is this note live"
// reads this struct's InEffect field rather than recomputing its own
// active/expiry/strength predicate.
type resolvedPheromone struct {
	Signal            colony.PheromoneSignal
	EffectiveStrength float64
	InEffect          bool
	ExcludedReason    string
	Provenance        string
}

// resolveEffectivePheromones is the single decision for whether a pheromone
// signal is in effect right now, how strong it is, and (if not) why not.
// Every reader that assembles pheromone text for a worker brief, a REDIRECT
// constraint, or an approval/listing surface must call this function rather
// than recomputing its own active/expiry/strength predicate --
// TestOneEffectivePheromonePredicate fails any second predicate by name.
//
// Decision order per signal: active flag, then quarantine state, then
// expiry (signalExpiredByTime), then effective strength against
// pheromoneEffectiveFloor (computeEffectiveStrength), then a final
// malformed-field check that overrides an otherwise-passing result -- a
// note with an unreadable timestamp or a non-finite strength is excluded
// even if the earlier checks would have let it through, so a malformed
// field is never silently defaulted into effect.
//
// The returned slice is sorted by signalPriority ascending, then effective
// strength descending, then signal ID ascending -- an explicit, total tie
// break so two equal-priority equal-strength signals return in the same
// order on every call, rather than depending on sort.SliceStable's input
// order (the correctness gap NOW-09/NOW-10/NOW-11 shared before this file).
func resolveEffectivePheromones(pf *colony.PheromoneFile, now time.Time) []resolvedPheromone {
	if pf == nil || len(pf.Signals) == 0 {
		return nil
	}

	resolved := make([]resolvedPheromone, 0, len(pf.Signals))
	for _, sig := range pf.Signals {
		provenance := pheromoneSignalProvenance(sig)

		if !sig.Active {
			resolved = append(resolved, resolvedPheromone{
				Signal: sig, InEffect: false, ExcludedReason: pheromoneExcludedInactive, Provenance: provenance,
			})
			continue
		}

		if pheromoneSignalQuarantined(sig) {
			resolved = append(resolved, resolvedPheromone{
				Signal: sig, InEffect: false, ExcludedReason: pheromoneExcludedQuarantine, Provenance: provenance,
			})
			continue
		}

		if signalExpiredByTime(sig, now) {
			resolved = append(resolved, resolvedPheromone{
				Signal: sig, InEffect: false, ExcludedReason: pheromoneExcludedExpired, Provenance: provenance,
			})
			continue
		}

		eff := computeEffectiveStrength(sig, now)

		if eff < pheromoneEffectiveFloor {
			resolved = append(resolved, resolvedPheromone{
				Signal: sig, EffectiveStrength: eff, InEffect: false, ExcludedReason: pheromoneExcludedBelowFloor, Provenance: provenance,
			})
			continue
		}

		if malformed, _ := pheromoneSignalMalformed(sig); malformed {
			resolved = append(resolved, resolvedPheromone{
				Signal: sig, EffectiveStrength: eff, InEffect: false, ExcludedReason: pheromoneExcludedMalformed, Provenance: provenance,
			})
			continue
		}

		resolved = append(resolved, resolvedPheromone{
			Signal: sig, EffectiveStrength: eff, InEffect: true, Provenance: provenance,
		})
	}

	sort.Slice(resolved, func(i, j int) bool {
		pi := signalPriority(resolved[i].Signal.Type)
		pj := signalPriority(resolved[j].Signal.Type)
		if pi != pj {
			return pi < pj
		}
		if resolved[i].EffectiveStrength != resolved[j].EffectiveStrength {
			return resolved[i].EffectiveStrength > resolved[j].EffectiveStrength
		}
		return resolved[i].Signal.ID < resolved[j].Signal.ID
	})

	return resolved
}

// pheromoneSignalMalformed reports whether a signal's timestamp or strength
// fields cannot be trusted. An unreadable field must exclude the signal,
// never default it into effect -- the reason string identifies which field.
//
// An EMPTY CreatedAt is not itself malformed: computeEffectiveStrength
// already treats a blank/unparseable CreatedAt as "no decay information,"
// returning the raw strength unchanged, and a large share of this
// codebase's existing pheromone fixtures omit CreatedAt entirely because
// they are not exercising date math. Only a NON-EMPTY value that fails to
// parse counts as genuinely corrupted.
func pheromoneSignalMalformed(sig colony.PheromoneSignal) (bool, string) {
	if sig.CreatedAt != "" {
		if _, err := time.Parse(time.RFC3339, sig.CreatedAt); err != nil {
			return true, "created_at"
		}
	}
	if sig.ExpiresAt != nil && *sig.ExpiresAt != "" {
		if _, err := time.Parse(time.RFC3339, *sig.ExpiresAt); err != nil {
			return true, "expires_at"
		}
	}
	if sig.Strength != nil {
		s := *sig.Strength
		if math.IsNaN(s) || math.IsInf(s, 0) {
			return true, "strength"
		}
	}
	return false, ""
}

// pheromoneSignalQuarantined reports whether a signal is currently
// quarantined. colony.PheromoneSignal has no Quarantined field yet -- this
// stub always reports false until the field is added (BIO-07 provenance
// task), at which point this body is extended to read it. No signal can be
// quarantined before that field exists, so "never quarantined" is the
// correct, honest answer today rather than a placeholder guess.
func pheromoneSignalQuarantined(sig colony.PheromoneSignal) bool {
	return false
}

// pheromoneSignalProvenance reports the origin of a signal.
// colony.PheromoneSignal has no Provenance field yet -- this stub always
// reports empty until the field is added (BIO-07 provenance task), at which
// point this body is extended to read it and fall back to
// colony.PheromoneProvenanceUnknown for a legacy signal with no value.
func pheromoneSignalProvenance(sig colony.PheromoneSignal) string {
	return ""
}
