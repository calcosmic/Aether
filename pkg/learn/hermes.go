package learn

// Hermes Concept Mapping (HIVE-01)
//
// Aether's learning system concepts are native Go implementations.
// The conceptual design draws from Hermes (MIT License) patterns for
// message-based learning and knowledge propagation.
//
// Concept mapping:
//
//   Hermes Concept          | Aether Implementation
//   ------------------------|-----------------------------------------------
//   MessageBus              | pkg/events/bus.go - Event bus with JSONL persistence
//   Agent Mailbox           | .aether/data/learn/entries.json - per-repo learning
//   Knowledge Fragment      | Entry struct with Evidence and Classification
//   Confidence Score        | pkg/memory/trust.go - 40/35/25 weighted scoring
//   Propagation             | HiveStore.Add - cross-colony wisdom promotion
//   Consolidation           | ColonyStore.Compact - budget-aware trimming
//   Privacy Filter          | ClassifyEntry - 4-way automatic classification
//
// No Hermes source code is included. This mapping documents the conceptual
// relationship for attribution per the MIT license notice requirement.
//
// MIT License Notice:
// Portions of the conceptual design were inspired by Hermes
// (https://github.com/hashicorp/hermes), Copyright (c) HashiCorp, Inc.,
// released under the MIT License. No source code from Hermes is used.

// HermesConceptMap documents the concept mapping for HIVE-01.
var HermesConceptMap = map[string]string{
	"MessageBus":        "pkg/events/bus.go - Event bus with JSONL persistence",
	"AgentMailbox":      "entries.json - per-repo learning store",
	"KnowledgeFragment": "Entry struct with Evidence and Classification",
	"ConfidenceScore":   "pkg/memory/trust.go - 40/35/25 weighted scoring",
	"Propagation":       "HiveStore.Add - cross-colony wisdom promotion",
	"Consolidation":     "ColonyStore.Compact - budget-aware trimming",
	"PrivacyFilter":     "ClassifyEntry - 4-way automatic classification",
}
