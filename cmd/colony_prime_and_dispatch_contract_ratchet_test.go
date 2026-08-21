package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// TestColonyPrimeAndDispatchContractDoNotReappear is 191-02's Tier-1
// reappearance ratchet (191-PATTERNS.md "The Ratchet House Style"): both
// loaders resolved by this plan read a single CWD-relative file each, so a
// plain os.Stat existence check is correct and sufficient -- no AST scanner
// is warranted for pure file-absence.
//
// Both files were confirmed zero-divergence-after-fold and deleted by
// 191-02-PLAN.md Task 2 (colony/prompts/colony-prime.md) and Task 3
// (colony/policies/dispatch-contract.yaml). Their Go-compiled fallback
// defaults in cmd/colony_prime_context.go and cmd/codex_dispatch_contract.go
// are now the sole, permanent source of this content -- loadColonyPrimeTemplates()
// and loadDispatchContractPolicy() remain in the codebase as permanently-
// fallback functions (191-CONTEXT.md D-04's explicit discretion), so they
// will silently pick the file back up if it reappears. This ratchet is what
// makes a silent reappearance loud instead.
//
// Each file gets its own t.Errorf (per-file, not aggregate) so a future
// failure names exactly which file came back, per 191-PATTERNS.md's Tier-1
// shape.
func TestColonyPrimeAndDispatchContractDoNotReappear(t *testing.T) {
	root := retiredPackagesRepoRoot(t)

	t.Run("ColonyPrimeMdStaysDeleted", func(t *testing.T) {
		p := filepath.Join(root, "colony", "prompts", "colony-prime.md")
		if _, err := os.Stat(p); err == nil {
			t.Errorf("colony/prompts/colony-prime.md has reappeared -- this file was ruled a CWD-relative silent-fallback loader target, folded into cmd/colony_prime_context.go's compiled defaults, and deleted in Phase 191 Plan 02 (ROADMAP criterion 2). If it is back, either it has a new reader whose divergence from the compiled default must be re-folded (update this ratchet and cmd/colony_prime_context.go with that evidence), or it should be deleted again.")
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat colony/prompts/colony-prime.md: %v", err)
		}
	})

	t.Run("DispatchContractYamlStaysDeleted", func(t *testing.T) {
		p := filepath.Join(root, "colony", "policies", "dispatch-contract.yaml")
		if _, err := os.Stat(p); err == nil {
			t.Errorf("colony/policies/dispatch-contract.yaml has reappeared -- this file was ruled a CWD-relative silent-fallback loader target, folded into cmd/codex_dispatch_contract.go's compiled fallback constants, and deleted in Phase 191 Plan 02 (ROADMAP criterion 2). If it is back, either it has a new reader whose divergence from the compiled default must be re-folded (update this ratchet and cmd/codex_dispatch_contract.go with that evidence), or it should be deleted again.")
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat colony/policies/dispatch-contract.yaml: %v", err)
		}
	})
}
