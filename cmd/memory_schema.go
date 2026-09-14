package cmd

// SYN-204-02 (204-03-PLAN.md Task 1, LEARN-01): cmd-package-local names for
// pkg/colony's shared memory schema and provenance contract. The contract
// itself lives in pkg/colony (pkg/colony/memory_schema.go), not here:
// pkg/memory's PromoteService.Promote and pkg/learn's ColonyStore.Add both
// already import pkg/colony for the record types (InstinctEntry,
// MiddenEntry) they stamp, but neither package may import package cmd --
// cmd imports both of them, and the reverse would cycle. Declaring these
// as cmd-local aliases/wrappers of the SAME colony.* symbols (never a
// second, competing declaration) is what lets cmd's own writer
// (appendMiddenEntry, cmd/midden_shared.go) and this file's Task 3
// field-writer census refer to the ONE declared contract under the names
// this phase's plan uses.

import "github.com/calcosmic/Aether/pkg/colony"

// memoryStoreSchemaVersion / memoryStoreLegacySchemaVersion are cmd-local
// aliases of colony.CurrentMemorySchemaVersion /
// colony.LegacyMemorySchemaVersion -- the SAME two integers every writer
// (cmd's appendMiddenEntry, pkg/memory's PromoteService.Promote,
// pkg/learn's ColonyStore.Add) stamps against.
const (
	memoryStoreSchemaVersion       = colony.CurrentMemorySchemaVersion
	memoryStoreLegacySchemaVersion = colony.LegacyMemorySchemaVersion
)

// memoryProvenanceKind / memoryRecordLineage are cmd-local type aliases
// (not copies -- `type X = Y` is a genuine alias, so a colony.MemoryRecordLineage
// value and a memoryRecordLineage value are the identical Go type) of
// colony's shared provenance vocabulary and lineage shape.
type (
	memoryProvenanceKind = colony.MemoryProvenanceKind
	memoryRecordLineage  = colony.MemoryRecordLineage
)

// memoryStoreSchemaReadable reports whether version is a version this
// runtime can read: the legacy version or the current one. Delegates to
// colony.MemoryStoreSchemaReadable -- the ONE declared readability rule,
// never reimplemented here.
func memoryStoreSchemaReadable(version int) bool {
	return colony.MemoryStoreSchemaReadable(version)
}
