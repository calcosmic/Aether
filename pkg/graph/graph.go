// Package graph provides a knowledge graph layer for instinct relationships,
// dependency tracking, and graph-based queries using in-memory structures.
// It replaces the shell's jq-based graph layer (graph.sh) with O(1) lookups
// via adjacency lists, supporting 5 node types and 16 edge types.
package graph

import (
	"crypto/rand"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// NodeType represents the kind of entity stored as a graph node.
type NodeType string

const (
	NodeLearning NodeType = "learning"
	NodeInstinct NodeType = "instinct"
	NodeQueen    NodeType = "queen"
	NodePhase    NodeType = "phase"
	NodeColony   NodeType = "colony"
)

// EdgeType represents the kind of relationship between two nodes.
type EdgeType string

const (
	// Learning-to-Instinct edges
	EdgePromotedFrom EdgeType = "promoted_from"
	EdgeDerivedFrom  EdgeType = "derived_from"
	// Instinct-to-Instinct edges (matching shell graph.sh)
	EdgeReinforces EdgeType = "reinforces"
	EdgeContradicts EdgeType = "contradicts"
	EdgeExtends    EdgeType = "extends"
	EdgeSupersedes EdgeType = "supersedes"
	EdgeRelated    EdgeType = "related"
	// Instinct-to-Queen edges
	EdgePromotedTo EdgeType = "promoted_to"
	// Phase-to-Learning/Instinct edges
	EdgeProduced   EdgeType = "produced"
	EdgeOriginated EdgeType = "originated"
	// Colony-to-Phase edges
	EdgeContains       EdgeType = "contains"
	EdgeColonyInstinct EdgeType = "colony_instinct"
	// Cross-type edges
	EdgeDependsOn      EdgeType = "depends_on"
	EdgeInfluenced     EdgeType = "influenced"
	EdgeSupersedesPhase EdgeType = "supersedes_phase"
)

// Node represents a typed entity in the knowledge graph.
type Node struct {
	ID   string   `json:"id"`
	Type NodeType `json:"type"`
}

// Edge represents a directed relationship between two nodes.
// JSON field names match the shell graph.sh format exactly.
type Edge struct {
	ID           string  `json:"edge_id"`
	Source       string  `json:"source"`
	Target       string  `json:"target"`
	Relationship EdgeType `json:"relationship"`
	Weight       float64 `json:"weight"`
	CreatedAt    string  `json:"created_at"`
}

// Neighbor represents a node connected to a query node.
type Neighbor struct {
	ID           string  `json:"id"`
	Relationship EdgeType `json:"relationship"`
	Weight       float64 `json:"weight"`
	Direction    string  `json:"direction"` // "out" or "in"
	Hop          int     `json:"hop,omitempty"`
}

// NeighborsResult holds the result of a neighbor query.
type NeighborsResult struct {
	Neighbors []Neighbor `json:"neighbors"`
	Count     int        `json:"count"`
}

// Graph is an in-memory directed graph with typed nodes and edges.
// All operations are goroutine-safe via sync.RWMutex.
type Graph struct {
	mu       sync.RWMutex
	nodes    map[string]*Node
	outEdges map[string][]*Edge // nodeID -> outbound edges from this node
	inEdges  map[string][]*Edge // nodeID -> inbound edges to this node
	edges    map[string]*Edge   // dedupKey -> edge (source\x00target\x00relationship)
}

// NewGraph creates an empty graph.
func NewGraph() *Graph {
	return &Graph{
		nodes:    make(map[string]*Node),
		outEdges: make(map[string][]*Edge),
		inEdges:  make(map[string][]*Edge),
		edges:    make(map[string]*Edge),
	}
}

// AddNode adds a node to the graph.
// Returns an error if a node with the same ID but different type exists.
// Adding a node with the same ID and same type is idempotent (no error).
func (g *Graph) AddNode(node Node) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if existing, ok := g.nodes[node.ID]; ok {
		if existing.Type != node.Type {
			return fmt.Errorf("graph: node %q already exists with type %q, cannot add as %q", node.ID, existing.Type, node.Type)
		}
		return nil
	}
	g.nodes[node.ID] = &node
	return nil
}

// GetNode returns a node by ID, or nil if not found.
func (g *Graph) GetNode(id string) *Node {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.nodes[id]
}

// AddEdge creates a directed edge between source and target nodes.
// If source or target nodes do not exist, they are auto-created with inferred types.
// If an edge with the same source+target+relationship already exists, the weight is updated.
// Returns the edge, a status string ("created" or "updated"), and any error.
func (g *Graph) AddEdge(source, target string, relType EdgeType, weight float64, createdAt string) (*Edge, string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	key := edgeKey(source, target, relType)
	if existing, ok := g.edges[key]; ok {
		existing.Weight = weight
		return existing, "updated", nil
	}

	// Generate edge ID matching the shell format: edge_{unix}_{4hex}
	now := time.Now()
	b := make([]byte, 2)
	rand.Read(b)
	hexStr := fmt.Sprintf("%x", b)
	for len(hexStr) < 4 {
		hexStr = "0" + hexStr
	}
	edgeID := fmt.Sprintf("edge_%d_%s", now.Unix(), hexStr)

	edge := &Edge{
		ID:           edgeID,
		Source:       source,
		Target:       target,
		Relationship: relType,
		Weight:       weight,
		CreatedAt:    createdAt,
	}

	// Auto-create nodes if missing
	if _, ok := g.nodes[source]; !ok {
		g.nodes[source] = &Node{ID: source, Type: inferType(source)}
	}
	if _, ok := g.nodes[target]; !ok {
		g.nodes[target] = &Node{ID: target, Type: inferType(target)}
	}

	g.edges[key] = edge
	g.outEdges[source] = append(g.outEdges[source], edge)
	g.inEdges[target] = append(g.inEdges[target], edge)

	return edge, "created", nil
}

// RemoveNode removes a node and all edges connected to it (both directions).
func (g *Graph) RemoveNode(id string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, ok := g.nodes[id]; !ok {
		return fmt.Errorf("graph: node %q not found", id)
	}

	// Remove all edges where this node is source or target
	// Collect edge keys to remove from the edges map
	var keysToRemove []string
	for _, e := range g.outEdges[id] {
		keysToRemove = append(keysToRemove, edgeKey(e.Source, e.Target, e.Relationship))
		// Remove from target's inEdges
		g.inEdges[e.Target] = removeEdgeFromList(g.inEdges[e.Target], e)
	}
	for _, e := range g.inEdges[id] {
		keysToRemove = append(keysToRemove, edgeKey(e.Source, e.Target, e.Relationship))
		// Remove from source's outEdges
		g.outEdges[e.Source] = removeEdgeFromList(g.outEdges[e.Source], e)
	}

	for _, k := range keysToRemove {
		delete(g.edges, k)
	}

	delete(g.nodes, id)
	delete(g.outEdges, id)
	delete(g.inEdges, id)

	return nil
}

// RemoveEdge removes a single edge by source, target, and relationship type.
func (g *Graph) RemoveEdge(source, target string, relType EdgeType) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	key := edgeKey(source, target, relType)
	edge, ok := g.edges[key]
	if !ok {
		return fmt.Errorf("graph: edge %q -> %q (%s) not found", source, target, relType)
	}

	delete(g.edges, key)
	g.outEdges[source] = removeEdgeFromList(g.outEdges[source], edge)
	g.inEdges[target] = removeEdgeFromList(g.inEdges[target], edge)

	return nil
}

// NodesByType returns all nodes of the given type.
func (g *Graph) NodesByType(nodeType NodeType) []*Node {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var result []*Node
	for _, n := range g.nodes {
		if n.Type == nodeType {
			result = append(result, n)
		}
	}
	return result
}

// NodeCount returns the number of nodes in the graph.
func (g *Graph) NodeCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.nodes)
}

// EdgeCount returns the number of edges in the graph.
func (g *Graph) EdgeCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.edges)
}

// Neighbors returns nodes connected to nodeID with the given direction filter.
// direction can be "out", "in", or "both".
// filterRel filters by relationship type (empty string means no filter).
// minWeight excludes edges with weight below the threshold (0 means no filter).
// Results are sorted by ID for deterministic output.
func (g *Graph) Neighbors(nodeID string, direction string, filterRel EdgeType, minWeight float64) (*NeighborsResult, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if _, ok := g.nodes[nodeID]; !ok {
		return nil, fmt.Errorf("graph: node %q not found", nodeID)
	}

	var neighbors []Neighbor

	if direction == "out" || direction == "both" {
		for _, e := range g.outEdges[nodeID] {
			if filterRel != "" && e.Relationship != filterRel {
				continue
			}
			if minWeight > 0 && e.Weight < minWeight {
				continue
			}
			neighbors = append(neighbors, Neighbor{
				ID:           e.Target,
				Relationship: e.Relationship,
				Weight:       e.Weight,
				Direction:    "out",
			})
		}
	}

	if direction == "in" || direction == "both" {
		for _, e := range g.inEdges[nodeID] {
			if filterRel != "" && e.Relationship != filterRel {
				continue
			}
			if minWeight > 0 && e.Weight < minWeight {
				continue
			}
			neighbors = append(neighbors, Neighbor{
				ID:           e.Source,
				Relationship: e.Relationship,
				Weight:       e.Weight,
				Direction:    "in",
			})
		}
	}

	// Sort by ID for deterministic output
	sort.Slice(neighbors, func(i, j int) bool {
		return neighbors[i].ID < neighbors[j].ID
	})

	return &NeighborsResult{Neighbors: neighbors, Count: len(neighbors)}, nil
}

// Neighbors2Hop returns unique nodes reachable within 2 hops from nodeID.
// Results are sorted by ID for deterministic output and tagged with hop count.
func (g *Graph) Neighbors2Hop(nodeID string, direction string, filterRel EdgeType, minWeight float64) ([]Neighbor, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if _, ok := g.nodes[nodeID]; !ok {
		return nil, fmt.Errorf("graph: node %q not found", nodeID)
	}

	visited := map[string]bool{nodeID: true}
	var results []Neighbor

	// 1-hop
	hop1 := g.neighborsInternal(nodeID, direction, filterRel, minWeight)
	hop1IDs := map[string]bool{}

	for _, n := range hop1 {
		if visited[n.ID] {
			continue
		}
		visited[n.ID] = true
		hop1IDs[n.ID] = true
		n.Hop = 1
		results = append(results, n)
	}

	// 2-hop: from each 1-hop neighbor
	for hop1ID := range hop1IDs {
		hop2 := g.neighborsInternal(hop1ID, direction, filterRel, minWeight)
		for _, n := range hop2 {
			if visited[n.ID] {
				continue
			}
			visited[n.ID] = true
			n.Hop = 2
			results = append(results, n)
		}
	}

	// Sort by ID for deterministic output
	sort.Slice(results, func(i, j int) bool {
		return results[i].ID < results[j].ID
	})

	return results, nil
}

// neighborsInternal returns neighbors without acquiring the lock (caller must hold read lock).
func (g *Graph) neighborsInternal(nodeID string, direction string, filterRel EdgeType, minWeight float64) []Neighbor {
	var neighbors []Neighbor

	if direction == "out" || direction == "both" {
		for _, e := range g.outEdges[nodeID] {
			if filterRel != "" && e.Relationship != filterRel {
				continue
			}
			if minWeight > 0 && e.Weight < minWeight {
				continue
			}
			neighbors = append(neighbors, Neighbor{
				ID:           e.Target,
				Relationship: e.Relationship,
				Weight:       e.Weight,
				Direction:    "out",
			})
		}
	}

	if direction == "in" || direction == "both" {
		for _, e := range g.inEdges[nodeID] {
			if filterRel != "" && e.Relationship != filterRel {
				continue
			}
			if minWeight > 0 && e.Weight < minWeight {
				continue
			}
			neighbors = append(neighbors, Neighbor{
				ID:           e.Source,
				Relationship: e.Relationship,
				Weight:       e.Weight,
				Direction:    "in",
			})
		}
	}

	return neighbors
}

// edgeKey constructs the dedup key for an edge: source + \x00 + target + \x00 + relationship.
func edgeKey(source, target string, relType EdgeType) string {
	return source + "\x00" + target + "\x00" + string(relType)
}

// inferType returns the NodeType for a node based on its ID prefix.
func inferType(id string) NodeType {
	if strings.HasPrefix(id, "obs_") {
		return NodeLearning
	}
	if strings.HasPrefix(id, "inst_") {
		return NodeInstinct
	}
	if strings.HasPrefix(id, "queen_") {
		return NodeQueen
	}
	if strings.HasPrefix(id, "phase_") {
		return NodePhase
	}
	if strings.HasPrefix(id, "colony_") {
		return NodeColony
	}
	// Default to instinct for unknown prefixes
	return NodeInstinct
}

// removeEdgeFromList removes an edge pointer from a slice of edge pointers.
func removeEdgeFromList(list []*Edge, target *Edge) []*Edge {
	for i, e := range list {
		if e == target {
			return append(list[:i], list[i+1:]...)
		}
	}
	return list
}
